defmodule MfaService.Crypto.Retry do
  @moduledoc """
  Retry logic with exponential backoff for Crypto Service calls.
  Retries transient failures up to a configurable maximum.
  """

  require Logger

  alias MfaService.Crypto.Config

  @type retry_opts :: [
    max_attempts: pos_integer(),
    base_delay: pos_integer(),
    max_delay: pos_integer(),
    retryable_errors: [atom()]
  ]

  @default_retryable_errors [:timeout, :unavailable, :internal, :unknown]

  @doc """
  Executes a function with retry logic.
  
  ## Options
    - `:max_attempts` - Maximum number of attempts (default: from config)
    - `:base_delay` - Base delay in ms for exponential backoff (default: from config)
    - `:max_delay` - Maximum delay in ms (default: 10000)
    - `:retryable_errors` - List of error atoms to retry on
  """
  @spec with_retry((() -> {:ok, term()} | {:error, term()}), retry_opts()) :: 
    {:ok, term()} | {:error, term()}
  def with_retry(fun, opts \\ []) when is_function(fun, 0) do
    max_attempts = Keyword.get(opts, :max_attempts, Config.retry_max_attempts())
    base_delay = Keyword.get(opts, :base_delay, Config.retry_base_delay())
    max_delay = Keyword.get(opts, :max_delay, 10_000)
    retryable_errors = Keyword.get(opts, :retryable_errors, @default_retryable_errors)

    do_retry(fun, 1, max_attempts, base_delay, max_delay, retryable_errors)
  end

  @doc """
  Checks if an error is retryable.
  """
  @spec retryable?(term(), [atom()]) :: boolean()
  def retryable?(error, retryable_errors \\ @default_retryable_errors)

  def retryable?({:error, %{code: code}}, retryable_errors) do
    code in retryable_errors
  end

  def retryable?({:error, %GRPC.RPCError{status: status}}, _retryable_errors) do
    status in [:unavailable, :deadline_exceeded, :internal, :unknown]
  end

  def retryable?({:error, :timeout}, _retryable_errors), do: true
  def retryable?({:error, :unavailable}, _retryable_errors), do: true
  def retryable?({:error, :circuit_open}, _retryable_errors), do: false
  def retryable?(_, _), do: false

  @doc """
  Calculates the delay for a given attempt using exponential backoff with jitter.
  """
  @spec calculate_delay(pos_integer(), pos_integer(), pos_integer()) :: pos_integer()
  def calculate_delay(attempt, base_delay, max_delay) do
    # Exponential backoff: base_delay * 2^(attempt-1)
    exponential_delay = base_delay * :math.pow(2, attempt - 1) |> round()
    
    # Add jitter (±25%)
    jitter = :rand.uniform() * 0.5 - 0.25
    delay_with_jitter = exponential_delay * (1 + jitter) |> round()
    
    # Cap at max_delay
    min(delay_with_jitter, max_delay)
  end

  # Private functions

  defp do_retry(fun, attempt, max_attempts, base_delay, max_delay, retryable_errors) do
    case fun.() do
      {:ok, _} = result ->
        result

      {:error, _} = error when attempt >= max_attempts ->
        Logger.warning("Retry exhausted after #{attempt} attempts",
          attempt: attempt, max_attempts: max_attempts)
        error

      {:error, _} = error ->
        if retryable?(error, retryable_errors) do
          delay = calculate_delay(attempt, base_delay, max_delay)
          
          Logger.debug("Retrying after #{delay}ms (attempt #{attempt}/#{max_attempts})",
            attempt: attempt, delay: delay)
          
          Process.sleep(delay)
          do_retry(fun, attempt + 1, max_attempts, base_delay, max_delay, retryable_errors)
        else
          Logger.debug("Error not retryable, returning immediately",
            attempt: attempt, error: inspect(error))
          error
        end
    end
  rescue
    e ->
      if attempt >= max_attempts do
        Logger.warning("Retry exhausted after exception on attempt #{attempt}",
          attempt: attempt, exception: inspect(e))
        {:error, e}
      else
        delay = calculate_delay(attempt, base_delay, max_delay)
        
        Logger.debug("Retrying after exception (attempt #{attempt}/#{max_attempts})",
          attempt: attempt, delay: delay, exception: inspect(e))
        
        Process.sleep(delay)
        do_retry(fun, attempt + 1, max_attempts, base_delay, max_delay, retryable_errors)
      end
  end
end
