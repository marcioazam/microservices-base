defmodule AuthPlatform.Resilience.CircuitBreaker do
  @moduledoc """
  Circuit breaker pattern implementation using GenServer.

  The circuit breaker prevents cascading failures by monitoring operation
  success/failure rates and temporarily blocking requests when failures exceed
  a threshold.

  ## States

  - `:closed` - Normal operation, requests pass through
  - `:open` - Failures exceeded threshold, requests blocked
  - `:half_open` - Testing if service recovered, limited requests allowed

  ## Usage

      # Start via supervisor
      AuthPlatform.Resilience.Supervisor.start_circuit_breaker(:my_api, %{
        failure_threshold: 5,
        timeout_ms: 30_000
      })

      # Execute with circuit breaker protection
      CircuitBreaker.execute(:my_api, fn ->
        ExternalAPI.call()
      end)

  ## Telemetry Events

  - `[:auth_platform, :circuit_breaker, :state_change]` - State transitions
  - `[:auth_platform, :circuit_breaker, :request_blocked]` - Blocked requests

  """
  use GenServer

  alias AuthPlatform.Resilience.Registry

  @type state :: :closed | :open | :half_open

  @type config :: %{
          failure_threshold: pos_integer(),
          success_threshold: pos_integer(),
          timeout_ms: pos_integer(),
          half_open_max_requests: pos_integer()
        }

  @type t :: %__MODULE__{
          name: atom(),
          config: config(),
          state: state(),
          failures: non_neg_integer(),
          successes: non_neg_integer(),
          last_failure_at: integer() | nil,
          half_open_requests: non_neg_integer()
        }

  defstruct [
    :name,
    :config,
    state: :closed,
    failures: 0,
    successes: 0,
    last_failure_at: nil,
    half_open_requests: 0
  ]

  @default_config %{
    failure_threshold: 5,
    success_threshold: 2,
    timeout_ms: 30_000,
    half_open_max_requests: 3
  }

  # Client API

  @doc """
  Starts a circuit breaker process.

  ## Options

    * `:name` - Required. Unique name for the circuit breaker
    * `:config` - Optional. Configuration map with:
      * `:failure_threshold` - Failures before opening (default: 5)
      * `:success_threshold` - Successes to close from half-open (default: 2)
      * `:timeout_ms` - Time in open state before half-open (default: 30_000)
      * `:half_open_max_requests` - Max requests in half-open (default: 3)

  """
  @spec start_link(keyword()) :: GenServer.on_start()
  def start_link(opts) do
    name = Keyword.fetch!(opts, :name)
    config = Map.merge(@default_config, Keyword.get(opts, :config, %{}))
    GenServer.start_link(__MODULE__, {name, config}, name: Registry.via_tuple(name))
  end

  @doc """
  Checks if a request is allowed through the circuit breaker.
  """
  @spec allow_request?(atom()) :: boolean()
  def allow_request?(name), do: GenServer.call(Registry.via_tuple(name), :allow_request?)

  @doc """
  Records a successful operation.
  """
  @spec record_success(atom()) :: :ok
  def record_success(name), do: GenServer.cast(Registry.via_tuple(name), :record_success)

  @doc """
  Records a failed operation.
  """
  @spec record_failure(atom()) :: :ok
  def record_failure(name), do: GenServer.cast(Registry.via_tuple(name), :record_failure)

  @doc """
  Gets the current state of the circuit breaker.
  """
  @spec get_state(atom()) :: state()
  def get_state(name), do: GenServer.call(Registry.via_tuple(name), :get_state)

  @doc """
  Gets full status information about the circuit breaker.
  """
  @spec get_status(atom()) :: map()
  def get_status(name), do: GenServer.call(Registry.via_tuple(name), :get_status)

  @doc """
  Resets the circuit breaker to closed state.
  """
  @spec reset(atom()) :: :ok
  def reset(name), do: GenServer.cast(Registry.via_tuple(name), :reset)

  @doc """
  Executes a function with circuit breaker protection.

  Returns `{:ok, result}` on success, `{:error, reason}` on failure,
  or `{:error, :circuit_open}` if the circuit is open.
  """
  @spec execute(atom(), (() -> {:ok, any()} | {:error, any()})) ::
          {:ok, any()} | {:error, any()}
  def execute(name, fun) when is_function(fun, 0) do
    if allow_request?(name) do
      try do
        case fun.() do
          {:ok, result} ->
            record_success(name)
            {:ok, result}

          {:error, reason} ->
            record_failure(name)
            {:error, reason}
        end
      rescue
        e ->
          record_failure(name)
          {:error, e}
      end
    else
      emit_blocked_event(name)
      {:error, :circuit_open}
    end
  end

  # Server Callbacks

  @impl true
  def init({name, config}) do
    state = %__MODULE__{
      name: name,
      config: config,
      state: :closed,
      failures: 0,
      successes: 0
    }

    {:ok, state}
  end

  @impl true
  def handle_call(:allow_request?, _from, state) do
    {allowed, new_state} = check_and_update_state(state)
    {:reply, allowed, new_state}
  end

  @impl true
  def handle_call(:get_state, _from, state) do
    {:reply, state.state, state}
  end

  @impl true
  def handle_call(:get_status, _from, state) do
    status = %{
      name: state.name,
      state: state.state,
      failures: state.failures,
      successes: state.successes,
      config: state.config,
      last_failure_at: state.last_failure_at,
      half_open_requests: state.half_open_requests
    }

    {:reply, status, state}
  end

  @impl true
  def handle_cast(:record_success, state) do
    new_state = handle_success(state)
    {:noreply, new_state}
  end

  @impl true
  def handle_cast(:record_failure, state) do
    new_state = handle_failure(state)
    {:noreply, new_state}
  end

  @impl true
  def handle_cast(:reset, state) do
    new_state = %{state | state: :closed, failures: 0, successes: 0, half_open_requests: 0}
    emit_state_change(state.name, state.state, :closed)
    {:noreply, new_state}
  end

  # Private Functions

  defp check_and_update_state(%{state: :closed} = state), do: {true, state}

  defp check_and_update_state(%{state: :open} = state) do
    if timeout_elapsed?(state) do
      new_state = transition_to(:half_open, state)
      {true, %{new_state | half_open_requests: 1}}
    else
      {false, state}
    end
  end

  defp check_and_update_state(%{state: :half_open} = state) do
    if state.half_open_requests < state.config.half_open_max_requests do
      {true, %{state | half_open_requests: state.half_open_requests + 1}}
    else
      {false, state}
    end
  end

  defp handle_success(%{state: :closed} = state) do
    %{state | failures: 0}
  end

  defp handle_success(%{state: :half_open} = state) do
    new_successes = state.successes + 1

    if new_successes >= state.config.success_threshold do
      transition_to(:closed, %{state | successes: 0, failures: 0, half_open_requests: 0})
    else
      %{state | successes: new_successes}
    end
  end

  defp handle_success(state), do: state

  defp handle_failure(%{state: :closed} = state) do
    new_failures = state.failures + 1
    now = System.monotonic_time(:millisecond)

    if new_failures >= state.config.failure_threshold do
      transition_to(:open, %{state | failures: new_failures, last_failure_at: now})
    else
      %{state | failures: new_failures, last_failure_at: now}
    end
  end

  defp handle_failure(%{state: :half_open} = state) do
    now = System.monotonic_time(:millisecond)
    transition_to(:open, %{state | failures: state.failures + 1, last_failure_at: now})
  end

  defp handle_failure(%{state: :open} = state) do
    %{state | last_failure_at: System.monotonic_time(:millisecond)}
  end

  defp timeout_elapsed?(%{last_failure_at: nil}), do: true

  defp timeout_elapsed?(state) do
    now = System.monotonic_time(:millisecond)
    now - state.last_failure_at >= state.config.timeout_ms
  end

  defp transition_to(new_state, %{state: old_state, name: name} = state) do
    emit_state_change(name, old_state, new_state)
    %{state | state: new_state, successes: 0, half_open_requests: 0}
  end

  defp emit_state_change(name, from, to) do
    :telemetry.execute(
      [:auth_platform, :circuit_breaker, :state_change],
      %{system_time: System.system_time()},
      %{name: name, from: from, to: to}
    )
  end

  defp emit_blocked_event(name) do
    :telemetry.execute(
      [:auth_platform, :circuit_breaker, :request_blocked],
      %{count: 1},
      %{name: name}
    )
  end
end
