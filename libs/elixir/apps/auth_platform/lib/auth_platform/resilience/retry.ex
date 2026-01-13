defmodule AuthPlatform.Resilience.Retry do
  @moduledoc """
  Retry policy with exponential backoff and jitter.

  Provides automatic retry logic for transient failures with configurable
  backoff strategies.

  ## Usage

      # Execute with default retry policy
      Retry.execute(fn -> ExternalAPI.call() end)

      # Execute with custom config
      config = %{max_retries: 5, initial_delay_ms: 100}
      Retry.execute(fn -> ExternalAPI.call() end, config)

  ## Configuration

    * `:max_retries` - Maximum retry attempts (default: 3)
    * `:initial_delay_ms` - Initial delay in milliseconds (default: 100)
    * `:max_delay_ms` - Maximum delay cap (default: 10_000)
    * `:multiplier` - Backoff multiplier (default: 2.0)
    * `:jitter` - Jitter factor 0.0-1.0 (default: 0.1)

  ## Telemetry Events

  - `[:auth_platform, :retry, :attempt]` - Each retry attempt
  - `[:auth_platform, :retry, :exhausted]` - All retries exhausted

  """

  alias AuthPlatform.Errors.AppError

  @type config :: %{
          max_retries: non_neg_integer(),
          initial_delay_ms: pos_integer(),
          max_delay_ms: pos_integer(),
          multiplier: float(),
          jitter: float()
        }

  @default_config %{
    max_retries: 3,
    initial_delay_ms: 100,
    max_delay_ms: 10_000,
    multiplier: 2.0,
    jitter: 0.1
  }

  @doc """
  Returns the default retry configuration.
  """
  @spec default_config() :: config()
  def default_config, do: @default_config

  @doc """
  Calculates the delay for a given attempt number.

  Uses exponential backoff with optional jitter.

  ## Examples

      iex> Retry.delay_for_attempt(1, %{initial_delay_ms: 100, multiplier: 2.0, max_delay_ms: 10000, jitter: 0})
      100

      iex> Retry.delay_for_attempt(3, %{initial_delay_ms: 100, multiplier: 2.0, max_delay_ms: 10000, jitter: 0})
      400

  """
  @spec delay_for_attempt(pos_integer(), config()) :: non_neg_integer()
  def delay_for_attempt(attempt, config) when attempt >= 1 do
    base_delay = config.initial_delay_ms * :math.pow(config.multiplier, attempt - 1)
    capped_delay = min(round(base_delay), config.max_delay_ms)

    if config.jitter > 0 do
      jitter_range = round(capped_delay * config.jitter)
      jitter_value = :rand.uniform(jitter_range * 2 + 1) - jitter_range - 1
      max(0, capped_delay + jitter_value)
    else
      capped_delay
    end
  end

  @doc """
  Determines if an error should be retried.

  By default, retries on:
  - `AppError` with `retryable: true`
  - Timeout errors
  - Connection errors

  ## Examples

      iex> Retry.should_retry?({:error, AppError.timeout("op")}, 1, %{max_retries: 3})
      true

      iex> Retry.should_retry?({:error, AppError.validation("bad")}, 1, %{max_retries: 3})
      false

  """
  @spec should_retry?({:error, any()}, non_neg_integer(), config()) :: boolean()
  def should_retry?({:error, reason}, attempt, config) do
    attempt < config.max_retries and is_retryable_error?(reason)
  end

  def should_retry?(_, _, _), do: false

  @doc """
  Executes a function with automatic retry on failure.

  Returns `{:ok, result}` on success or `{:error, reason}` after all retries exhausted.

  ## Options

    * `:on_retry` - Callback function called before each retry `fn attempt, delay, error -> :ok end`

  ## Examples

      Retry.execute(fn -> {:ok, "success"} end)
      #=> {:ok, "success"}

      Retry.execute(fn -> {:error, :timeout} end, %{max_retries: 2})
      #=> {:error, :timeout}

  """
  @spec execute((() -> {:ok, any()} | {:error, any()}), config(), keyword()) ::
          {:ok, any()} | {:error, any()}
  def execute(fun, config \\ @default_config, opts \\ []) when is_function(fun, 0) do
    config = Map.merge(@default_config, config)
    on_retry = Keyword.get(opts, :on_retry, fn _, _, _ -> :ok end)
    do_execute(fun, config, 0, on_retry)
  end

  defp do_execute(fun, config, attempt, on_retry) do
    emit_attempt_event(attempt + 1, config.max_retries)

    case safe_execute(fun) do
      {:ok, result} ->
        {:ok, result}

      {:error, reason} = error ->
        if should_retry?(error, attempt, config) do
          delay = delay_for_attempt(attempt + 1, config)
          on_retry.(attempt + 1, delay, reason)
          Process.sleep(delay)
          do_execute(fun, config, attempt + 1, on_retry)
        else
          emit_exhausted_event(attempt + 1, reason)
          error
        end
    end
  end

  defp safe_execute(fun) do
    try do
      fun.()
    rescue
      e -> {:error, e}
    catch
      :exit, reason -> {:error, {:exit, reason}}
    end
  end

  defp is_retryable_error?(%AppError{retryable: true}), do: true
  defp is_retryable_error?(%AppError{}), do: false
  defp is_retryable_error?(:timeout), do: true
  defp is_retryable_error?(:econnrefused), do: true
  defp is_retryable_error?(:econnreset), do: true
  defp is_retryable_error?(:closed), do: true
  defp is_retryable_error?({:exit, _}), do: true
  defp is_retryable_error?(_), do: false

  defp emit_attempt_event(attempt, max_retries) do
    :telemetry.execute(
      [:auth_platform, :retry, :attempt],
      %{attempt: attempt},
      %{max_retries: max_retries}
    )
  end

  defp emit_exhausted_event(attempts, reason) do
    :telemetry.execute(
      [:auth_platform, :retry, :exhausted],
      %{attempts: attempts},
      %{reason: reason}
    )
  end
end
