defmodule AuthPlatform.Resilience.RateLimiter do
  @moduledoc """
  Token bucket rate limiter implementation using GenServer.

  Limits the rate of operations using a token bucket algorithm with
  configurable rate and burst size.

  ## Usage

      # Start via supervisor
      AuthPlatform.Resilience.Supervisor.start_rate_limiter(:api_limiter, %{
        rate: 100,        # tokens per second
        burst_size: 150   # max tokens
      })

      # Check if request is allowed (non-blocking)
      if RateLimiter.allow?(:api_limiter) do
        process_request()
      else
        {:error, :rate_limited}
      end

      # Acquire token with blocking wait
      case RateLimiter.acquire(:api_limiter, 5000) do
        :ok -> process_request()
        {:error, :timeout} -> {:error, :rate_limited}
      end

  ## Telemetry Events

  - `[:auth_platform, :rate_limiter, :allowed]` - Request allowed
  - `[:auth_platform, :rate_limiter, :rejected]` - Request rejected

  """
  use GenServer

  alias AuthPlatform.Resilience.Registry

  @type config :: %{
          rate: pos_integer(),
          burst_size: pos_integer()
        }

  @type t :: %__MODULE__{
          name: atom(),
          config: config(),
          tokens: float(),
          last_refill: integer()
        }

  defstruct [:name, :config, :tokens, :last_refill]

  @default_config %{
    rate: 100,
    burst_size: 100
  }

  # Client API

  @doc """
  Starts a rate limiter process.

  ## Options

    * `:name` - Required. Unique name for the rate limiter
    * `:config` - Optional. Configuration map with:
      * `:rate` - Tokens per second (default: 100)
      * `:burst_size` - Maximum tokens (default: 100)

  """
  @spec start_link(keyword()) :: GenServer.on_start()
  def start_link(opts) do
    name = Keyword.fetch!(opts, :name)
    config = Map.merge(@default_config, Keyword.get(opts, :config, %{}))
    GenServer.start_link(__MODULE__, {name, config}, name: Registry.via_tuple(name))
  end

  @doc """
  Checks if a request is allowed (non-blocking).

  Returns `true` if a token is available and consumed, `false` otherwise.
  """
  @spec allow?(atom()) :: boolean()
  def allow?(name), do: GenServer.call(Registry.via_tuple(name), :allow?)

  @doc """
  Acquires a token, blocking until available or timeout.

  Returns `:ok` if token acquired, `{:error, :timeout}` if timeout exceeded.
  """
  @spec acquire(atom(), timeout()) :: :ok | {:error, :timeout}
  def acquire(name, timeout \\ 5000) do
    GenServer.call(Registry.via_tuple(name), {:acquire, timeout}, timeout + 100)
  catch
    :exit, {:timeout, _} -> {:error, :timeout}
  end

  @doc """
  Gets the current number of available tokens.
  """
  @spec available_tokens(atom()) :: float()
  def available_tokens(name), do: GenServer.call(Registry.via_tuple(name), :available_tokens)

  @doc """
  Gets the current status of the rate limiter.
  """
  @spec get_status(atom()) :: map()
  def get_status(name), do: GenServer.call(Registry.via_tuple(name), :get_status)

  @doc """
  Resets the rate limiter to full capacity.
  """
  @spec reset(atom()) :: :ok
  def reset(name), do: GenServer.cast(Registry.via_tuple(name), :reset)

  # Server Callbacks

  @impl true
  def init({name, config}) do
    state = %__MODULE__{
      name: name,
      config: config,
      tokens: config.burst_size * 1.0,
      last_refill: System.monotonic_time(:millisecond)
    }

    {:ok, state}
  end

  @impl true
  def handle_call(:allow?, _from, state) do
    state = refill_tokens(state)

    if state.tokens >= 1.0 do
      emit_allowed_event(state.name)
      {:reply, true, %{state | tokens: state.tokens - 1.0}}
    else
      emit_rejected_event(state.name)
      {:reply, false, state}
    end
  end

  @impl true
  def handle_call({:acquire, timeout}, from, state) do
    state = refill_tokens(state)

    if state.tokens >= 1.0 do
      emit_allowed_event(state.name)
      {:reply, :ok, %{state | tokens: state.tokens - 1.0}}
    else
      # Calculate wait time for next token
      wait_ms = calculate_wait_time(state)

      if wait_ms <= timeout do
        # Schedule a delayed reply
        Process.send_after(self(), {:delayed_acquire, from, timeout - wait_ms}, wait_ms)
        {:noreply, state}
      else
        emit_rejected_event(state.name)
        {:reply, {:error, :timeout}, state}
      end
    end
  end

  @impl true
  def handle_call(:available_tokens, _from, state) do
    state = refill_tokens(state)
    {:reply, state.tokens, state}
  end

  @impl true
  def handle_call(:get_status, _from, state) do
    state = refill_tokens(state)

    status = %{
      name: state.name,
      tokens: state.tokens,
      config: state.config
    }

    {:reply, status, state}
  end

  @impl true
  def handle_cast(:reset, state) do
    new_state = %{
      state
      | tokens: state.config.burst_size * 1.0,
        last_refill: System.monotonic_time(:millisecond)
    }

    {:noreply, new_state}
  end

  @impl true
  def handle_info({:delayed_acquire, from, remaining_timeout}, state) do
    state = refill_tokens(state)

    if state.tokens >= 1.0 do
      emit_allowed_event(state.name)
      GenServer.reply(from, :ok)
      {:noreply, %{state | tokens: state.tokens - 1.0}}
    else
      wait_ms = calculate_wait_time(state)

      if wait_ms <= remaining_timeout do
        Process.send_after(self(), {:delayed_acquire, from, remaining_timeout - wait_ms}, wait_ms)
        {:noreply, state}
      else
        emit_rejected_event(state.name)
        GenServer.reply(from, {:error, :timeout})
        {:noreply, state}
      end
    end
  end

  # Private Functions

  defp refill_tokens(state) do
    now = System.monotonic_time(:millisecond)
    elapsed_ms = now - state.last_refill
    tokens_to_add = elapsed_ms * state.config.rate / 1000.0
    new_tokens = min(state.tokens + tokens_to_add, state.config.burst_size * 1.0)

    %{state | tokens: new_tokens, last_refill: now}
  end

  defp calculate_wait_time(state) do
    tokens_needed = 1.0 - state.tokens
    round(tokens_needed * 1000.0 / state.config.rate)
  end

  defp emit_allowed_event(name) do
    :telemetry.execute(
      [:auth_platform, :rate_limiter, :allowed],
      %{count: 1},
      %{name: name}
    )
  end

  defp emit_rejected_event(name) do
    :telemetry.execute(
      [:auth_platform, :rate_limiter, :rejected],
      %{count: 1},
      %{name: name}
    )
  end
end
