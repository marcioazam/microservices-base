defmodule SessionIdentityCore.Crypto.CircuitBreaker do
  @moduledoc """
  Circuit breaker wrapper for crypto-service operations.
  
  Implements the circuit breaker pattern to prevent cascading failures
  when crypto-service is unavailable.
  
  States:
  - :closed - Normal operation, requests pass through
  - :open - Circuit is open, requests fail fast with fallback
  - :half_open - Testing if service recovered
  
  Uses the :fuse library for circuit breaker implementation.
  """

  require Logger

  alias SessionIdentityCore.Crypto.{Config, Errors}

  @fuse_name :crypto_service_fuse

  @doc """
  Initializes the circuit breaker.
  Should be called at application startup.
  """
  @spec init() :: :ok
  def init do
    config = Config.get()
    
    opts = {
      {:standard, config.circuit_breaker_threshold, config.circuit_breaker_timeout},
      {:reset, config.circuit_breaker_timeout}
    }

    case :fuse.install(@fuse_name, opts) do
      :ok -> 
        Logger.info("Circuit breaker initialized for crypto-service")
        :ok
      {:error, :already_installed} -> 
        :ok
    end
  end

  @doc """
  Executes a function through the circuit breaker.
  
  If the circuit is open, returns {:error, :circuit_open}.
  If the function fails, records the failure.
  """
  @spec call((() -> {:ok, term()} | {:error, term()})) :: {:ok, term()} | {:error, term()}
  def call(fun) when is_function(fun, 0) do
    case :fuse.ask(@fuse_name, :sync) do
      :ok ->
        execute_with_tracking(fun)

      :blown ->
        Logger.warning("Circuit breaker is open for crypto-service")
        emit_circuit_open_metric()
        {:error, Errors.service_unavailable("Circuit breaker is open")}
    end
  end

  @doc """
  Executes a function with fallback when circuit is open.
  """
  @spec call_with_fallback((() -> {:ok, term()} | {:error, term()}), (() -> {:ok, term()} | {:error, term()})) :: {:ok, term()} | {:error, term()}
  def call_with_fallback(fun, fallback) when is_function(fun, 0) and is_function(fallback, 0) do
    case :fuse.ask(@fuse_name, :sync) do
      :ok ->
        execute_with_tracking(fun)

      :blown ->
        Logger.warning("Circuit breaker is open, using fallback")
        emit_fallback_metric()
        fallback.()
    end
  end

  @doc """
  Returns the current circuit breaker state.
  """
  @spec state() :: :ok | :blown
  def state do
    :fuse.ask(@fuse_name, :sync)
  end

  @doc """
  Manually resets the circuit breaker.
  """
  @spec reset() :: :ok
  def reset do
    :fuse.reset(@fuse_name)
  end

  @doc """
  Manually melts (opens) the circuit breaker.
  """
  @spec melt() :: :ok
  def melt do
    :fuse.melt(@fuse_name)
  end

  @doc """
  Records a successful operation.
  """
  @spec record_success() :: :ok
  def record_success do
    # Fuse automatically handles success tracking
    :ok
  end

  @doc """
  Records a failed operation.
  """
  @spec record_failure() :: :ok
  def record_failure do
    :fuse.melt(@fuse_name)
    :ok
  end

  # Private Functions

  defp execute_with_tracking(fun) do
    case fun.() do
      {:ok, result} ->
        {:ok, result}

      {:error, %{error_code: code} = error} when code in [:crypto_service_unavailable, :crypto_operation_timeout] ->
        record_failure()
        {:error, error}

      {:error, _} = error ->
        # Don't trip circuit for non-transient errors
        error
    end
  rescue
    e ->
      record_failure()
      Logger.error("Circuit breaker caught exception: #{inspect(e)}")
      {:error, Errors.operation_failed(Exception.message(e))}
  end

  defp emit_circuit_open_metric do
    :telemetry.execute(
      [:session_identity, :crypto, :circuit_breaker],
      %{count: 1},
      %{state: :open}
    )
  end

  defp emit_fallback_metric do
    :telemetry.execute(
      [:session_identity, :crypto, :fallback],
      %{count: 1},
      %{reason: :circuit_open}
    )
  end
end
