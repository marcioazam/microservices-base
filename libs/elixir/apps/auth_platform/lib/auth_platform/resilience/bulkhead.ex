defmodule AuthPlatform.Resilience.Bulkhead do
  @moduledoc """
  Bulkhead pattern implementation for resource isolation.

  Limits concurrent executions to prevent resource exhaustion and
  isolate failures between different parts of the system.

  ## Usage

      # Start via supervisor
      AuthPlatform.Resilience.Supervisor.start_bulkhead(:db_pool, %{
        max_concurrent: 10,
        max_queue: 50,
        queue_timeout_ms: 5000
      })

      # Execute with bulkhead protection
      Bulkhead.execute(:db_pool, fn ->
        Database.query(sql)
      end)

  ## Telemetry Events

  - `[:auth_platform, :bulkhead, :acquired]` - Permit acquired
  - `[:auth_platform, :bulkhead, :released]` - Permit released
  - `[:auth_platform, :bulkhead, :rejected]` - Request rejected (full)
  - `[:auth_platform, :bulkhead, :queued]` - Request queued

  """
  use GenServer

  alias AuthPlatform.Resilience.Registry

  @type config :: %{
          max_concurrent: pos_integer(),
          max_queue: non_neg_integer(),
          queue_timeout_ms: pos_integer()
        }

  @type t :: %__MODULE__{
          name: atom(),
          config: config(),
          active: non_neg_integer(),
          queue: :queue.queue()
        }

  defstruct [:name, :config, active: 0, queue: :queue.new()]

  @default_config %{
    max_concurrent: 10,
    max_queue: 100,
    queue_timeout_ms: 5000
  }

  # Client API

  @doc """
  Starts a bulkhead process.

  ## Options

    * `:name` - Required. Unique name for the bulkhead
    * `:config` - Optional. Configuration map with:
      * `:max_concurrent` - Maximum concurrent executions (default: 10)
      * `:max_queue` - Maximum queued requests (default: 100)
      * `:queue_timeout_ms` - Queue timeout in milliseconds (default: 5000)

  """
  @spec start_link(keyword()) :: GenServer.on_start()
  def start_link(opts) do
    name = Keyword.fetch!(opts, :name)
    config = Map.merge(@default_config, Keyword.get(opts, :config, %{}))
    GenServer.start_link(__MODULE__, {name, config}, name: Registry.via_tuple(name))
  end

  @doc """
  Executes a function with bulkhead protection.

  Returns `{:ok, result}` on success, `{:error, reason}` on failure,
  or `{:error, :bulkhead_full}` if rejected.
  """
  @spec execute(atom(), (() -> any()), keyword()) :: {:ok, any()} | {:error, any()}
  def execute(name, fun, opts \\ []) when is_function(fun, 0) do
    timeout = Keyword.get(opts, :timeout, 5000)

    case acquire(name, timeout) do
      :ok ->
        try do
          result = fun.()
          {:ok, result}
        rescue
          e -> {:error, e}
        after
          release(name)
        end

      {:error, reason} ->
        {:error, reason}
    end
  end

  @doc """
  Acquires a permit from the bulkhead.

  Returns `:ok` if acquired, `{:error, :bulkhead_full}` if rejected,
  or `{:error, :timeout}` if queue timeout exceeded.
  """
  @spec acquire(atom(), timeout()) :: :ok | {:error, :bulkhead_full | :timeout}
  def acquire(name, timeout \\ 5000) do
    GenServer.call(Registry.via_tuple(name), {:acquire, timeout}, timeout + 100)
  catch
    :exit, {:timeout, _} -> {:error, :timeout}
  end

  @doc """
  Releases a permit back to the bulkhead.
  """
  @spec release(atom()) :: :ok
  def release(name), do: GenServer.cast(Registry.via_tuple(name), :release)

  @doc """
  Returns the number of available permits.
  """
  @spec available_permits(atom()) :: non_neg_integer()
  def available_permits(name), do: GenServer.call(Registry.via_tuple(name), :available_permits)

  @doc """
  Returns the current status of the bulkhead.
  """
  @spec get_status(atom()) :: map()
  def get_status(name), do: GenServer.call(Registry.via_tuple(name), :get_status)

  # Server Callbacks

  @impl true
  def init({name, config}) do
    state = %__MODULE__{
      name: name,
      config: config,
      active: 0,
      queue: :queue.new()
    }

    {:ok, state}
  end

  @impl true
  def handle_call({:acquire, timeout}, from, state) do
    cond do
      state.active < state.config.max_concurrent ->
        emit_acquired_event(state.name)
        {:reply, :ok, %{state | active: state.active + 1}}

      :queue.len(state.queue) < state.config.max_queue ->
        emit_queued_event(state.name)
        timer_ref = Process.send_after(self(), {:queue_timeout, from}, timeout)
        new_queue = :queue.in({from, timer_ref}, state.queue)
        {:noreply, %{state | queue: new_queue}}

      true ->
        emit_rejected_event(state.name)
        {:reply, {:error, :bulkhead_full}, state}
    end
  end

  @impl true
  def handle_call(:available_permits, _from, state) do
    available = max(0, state.config.max_concurrent - state.active)
    {:reply, available, state}
  end

  @impl true
  def handle_call(:get_status, _from, state) do
    status = %{
      name: state.name,
      active: state.active,
      queued: :queue.len(state.queue),
      available: max(0, state.config.max_concurrent - state.active),
      config: state.config
    }

    {:reply, status, state}
  end

  @impl true
  def handle_cast(:release, state) do
    emit_released_event(state.name)
    new_state = process_queue(%{state | active: state.active - 1})
    {:noreply, new_state}
  end

  @impl true
  def handle_info({:queue_timeout, from}, state) do
    # Remove from queue and reply with timeout
    new_queue = remove_from_queue(state.queue, from)

    if new_queue != state.queue do
      GenServer.reply(from, {:error, :timeout})
    end

    {:noreply, %{state | queue: new_queue}}
  end

  # Private Functions

  defp process_queue(state) do
    if state.active < state.config.max_concurrent and not :queue.is_empty(state.queue) do
      {{:value, {from, timer_ref}}, new_queue} = :queue.out(state.queue)
      Process.cancel_timer(timer_ref)
      GenServer.reply(from, :ok)
      emit_acquired_event(state.name)
      %{state | active: state.active + 1, queue: new_queue}
    else
      state
    end
  end

  defp remove_from_queue(queue, target_from) do
    :queue.filter(fn {from, _timer_ref} -> from != target_from end, queue)
  end

  defp emit_acquired_event(name) do
    :telemetry.execute([:auth_platform, :bulkhead, :acquired], %{count: 1}, %{name: name})
  end

  defp emit_released_event(name) do
    :telemetry.execute([:auth_platform, :bulkhead, :released], %{count: 1}, %{name: name})
  end

  defp emit_rejected_event(name) do
    :telemetry.execute([:auth_platform, :bulkhead, :rejected], %{count: 1}, %{name: name})
  end

  defp emit_queued_event(name) do
    :telemetry.execute([:auth_platform, :bulkhead, :queued], %{count: 1}, %{name: name})
  end
end
