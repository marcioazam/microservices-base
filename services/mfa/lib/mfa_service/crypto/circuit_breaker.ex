defmodule MfaService.Crypto.CircuitBreaker do
  @moduledoc """
  Circuit breaker implementation for Crypto Service calls.
  Provides fail-fast behavior when the service is unhealthy.
  
  States:
  - :closed - Normal operation, requests pass through
  - :open - Service unhealthy, requests fail fast
  - :half_open - Testing if service recovered
  """

  use GenServer
  require Logger

  alias MfaService.Crypto.{Config, Telemetry}

  @type state :: :closed | :open | :half_open

  defstruct [
    :state,
    :failure_count,
    :success_count,
    :last_failure_time,
    :threshold,
    :reset_timeout,
    :half_open_max_calls
  ]

  # Client API

  @doc """
  Starts the circuit breaker GenServer.
  """
  def start_link(opts \\ []) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end

  @doc """
  Executes a function through the circuit breaker.
  Returns {:error, :circuit_open} if the circuit is open.
  """
  @spec call((() -> {:ok, term()} | {:error, term()})) :: {:ok, term()} | {:error, term()}
  def call(fun) when is_function(fun, 0) do
    GenServer.call(__MODULE__, {:call, fun})
  end

  @doc """
  Returns the current state of the circuit breaker.
  """
  @spec get_state() :: state()
  def get_state do
    GenServer.call(__MODULE__, :get_state)
  end

  @doc """
  Manually resets the circuit breaker to closed state.
  """
  @spec reset() :: :ok
  def reset do
    GenServer.call(__MODULE__, :reset)
  end

  @doc """
  Records a successful call (for external tracking).
  """
  @spec record_success() :: :ok
  def record_success do
    GenServer.cast(__MODULE__, :success)
  end

  @doc """
  Records a failed call (for external tracking).
  """
  @spec record_failure() :: :ok
  def record_failure do
    GenServer.cast(__MODULE__, :failure)
  end

  # GenServer callbacks

  @impl true
  def init(_opts) do
    state = %__MODULE__{
      state: :closed,
      failure_count: 0,
      success_count: 0,
      last_failure_time: nil,
      threshold: Config.circuit_breaker_threshold(),
      reset_timeout: Config.circuit_breaker_reset_timeout(),
      half_open_max_calls: 1
    }

    {:ok, state}
  end

  @impl true
  def handle_call({:call, fun}, _from, %{state: :open} = state) do
    if should_attempt_reset?(state) do
      # Transition to half-open and try the call
      new_state = %{state | state: :half_open, success_count: 0}
      Telemetry.emit_circuit_breaker_state_change(:half_open)
      
      execute_and_update(fun, new_state)
    else
      {:reply, {:error, :circuit_open}, state}
    end
  end

  @impl true
  def handle_call({:call, fun}, _from, state) do
    execute_and_update(fun, state)
  end

  @impl true
  def handle_call(:get_state, _from, state) do
    {:reply, state.state, state}
  end

  @impl true
  def handle_call(:reset, _from, state) do
    new_state = %{state | 
      state: :closed, 
      failure_count: 0, 
      success_count: 0,
      last_failure_time: nil
    }
    
    if state.state != :closed do
      Telemetry.emit_circuit_breaker_state_change(:closed)
    end
    
    {:reply, :ok, new_state}
  end

  @impl true
  def handle_cast(:success, state) do
    {:noreply, handle_success(state)}
  end

  @impl true
  def handle_cast(:failure, state) do
    {:noreply, handle_failure(state)}
  end

  # Private functions

  defp execute_and_update(fun, state) do
    try do
      case fun.() do
        {:ok, _} = result ->
          new_state = handle_success(state)
          {:reply, result, new_state}

        {:error, _} = error ->
          new_state = handle_failure(state)
          {:reply, error, new_state}
      end
    rescue
      e ->
        new_state = handle_failure(state)
        {:reply, {:error, e}, new_state}
    end
  end

  defp handle_success(%{state: :half_open} = state) do
    new_count = state.success_count + 1
    
    if new_count >= state.half_open_max_calls do
      Logger.info("Circuit breaker closed after successful probe")
      Telemetry.emit_circuit_breaker_state_change(:closed)
      
      %{state | 
        state: :closed, 
        failure_count: 0, 
        success_count: 0,
        last_failure_time: nil
      }
    else
      %{state | success_count: new_count}
    end
  end

  defp handle_success(state) do
    %{state | failure_count: 0}
  end

  defp handle_failure(%{state: :half_open} = state) do
    Logger.warning("Circuit breaker reopened after failed probe")
    Telemetry.emit_circuit_breaker_state_change(:open)
    
    %{state | 
      state: :open, 
      failure_count: state.threshold,
      last_failure_time: System.monotonic_time(:millisecond)
    }
  end

  defp handle_failure(%{state: :closed} = state) do
    new_count = state.failure_count + 1
    
    if new_count >= state.threshold do
      Logger.warning("Circuit breaker opened after #{new_count} consecutive failures")
      Telemetry.emit_circuit_breaker_state_change(:open)
      
      %{state | 
        state: :open, 
        failure_count: new_count,
        last_failure_time: System.monotonic_time(:millisecond)
      }
    else
      %{state | failure_count: new_count}
    end
  end

  defp handle_failure(state) do
    %{state | 
      failure_count: state.failure_count + 1,
      last_failure_time: System.monotonic_time(:millisecond)
    }
  end

  defp should_attempt_reset?(state) do
    case state.last_failure_time do
      nil -> 
        true
      time -> 
        elapsed = System.monotonic_time(:millisecond) - time
        elapsed >= state.reset_timeout
    end
  end
end
