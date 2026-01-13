defmodule AuthPlatform.Testing.Helpers do
  @moduledoc """
  Test helpers for Auth Platform resilience components.

  Provides utilities for testing circuit breakers, rate limiters, and bulkheads.

  ## Usage

      # Force circuit breaker to open state
      Helpers.force_circuit_breaker_open(:my_breaker)

      # Create a mock retryable operation
      {fun, counter} = Helpers.mock_retryable_operation(fail_times: 2)

  """

  alias AuthPlatform.Resilience.CircuitBreaker
  alias AuthPlatform.Resilience.RateLimiter
  alias AuthPlatform.Resilience.Bulkhead

  @doc """
  Forces a circuit breaker to the open state.
  """
  @spec force_circuit_breaker_open(atom()) :: :ok
  def force_circuit_breaker_open(name) do
    status = CircuitBreaker.get_status(name)
    threshold = status.config.failure_threshold

    for _ <- 1..threshold do
      CircuitBreaker.record_failure(name)
    end

    Process.sleep(10)
    :ok
  end

  @doc """
  Forces a circuit breaker to the closed state.
  """
  @spec force_circuit_breaker_closed(atom()) :: :ok
  def force_circuit_breaker_closed(name) do
    CircuitBreaker.reset(name)
    Process.sleep(10)
    :ok
  end

  @doc """
  Forces a circuit breaker to the half-open state.

  Note: Requires the circuit to be open first and timeout to have elapsed.
  """
  @spec force_circuit_breaker_half_open(atom()) :: :ok
  def force_circuit_breaker_half_open(name) do
    force_circuit_breaker_open(name)
    status = CircuitBreaker.get_status(name)
    Process.sleep(status.config.timeout_ms + 10)
    CircuitBreaker.allow_request?(name)
    :ok
  end

  @doc """
  Creates a mock operation that fails a specified number of times before succeeding.

  Returns a tuple of {function, counter} where counter can be used to verify call count.

  ## Options

    * `:fail_times` - Number of times to fail before succeeding (default: 2)
    * `:error` - Error to return on failure (default: `:timeout`)
    * `:success` - Value to return on success (default: `:success`)

  ## Examples

      {fun, counter} = Helpers.mock_retryable_operation(fail_times: 2)
      fun.() #=> {:error, :timeout}
      fun.() #=> {:error, :timeout}
      fun.() #=> {:ok, :success}

  """
  @spec mock_retryable_operation(keyword()) :: {(() -> {:ok, any()} | {:error, any()}), reference()}
  def mock_retryable_operation(opts \\ []) do
    fail_times = Keyword.get(opts, :fail_times, 2)
    error = Keyword.get(opts, :error, :timeout)
    success = Keyword.get(opts, :success, :success)

    counter = :counters.new(1, [:atomics])

    fun = fn ->
      count = :counters.get(counter, 1)
      :counters.add(counter, 1, 1)

      if count < fail_times do
        {:error, error}
      else
        {:ok, success}
      end
    end

    {fun, counter}
  end

  @doc """
  Gets the call count from a mock operation counter.
  """
  @spec get_call_count(reference()) :: non_neg_integer()
  def get_call_count(counter) do
    :counters.get(counter, 1)
  end

  @doc """
  Drains all tokens from a rate limiter.
  """
  @spec drain_rate_limiter(atom()) :: :ok
  def drain_rate_limiter(name) do
    status = RateLimiter.get_status(name)
    tokens = ceil(status.tokens)

    for _ <- 1..tokens do
      RateLimiter.allow?(name)
    end

    :ok
  end

  @doc """
  Fills a bulkhead to capacity.

  Returns a list of tasks that are holding the permits.
  """
  @spec fill_bulkhead(atom()) :: [Task.t()]
  def fill_bulkhead(name) do
    status = Bulkhead.get_status(name)
    available = status.available

    for _ <- 1..available do
      Task.async(fn ->
        Bulkhead.acquire(name, 60_000)
        receive do: (:release -> :ok)
      end)
    end
  end

  @doc """
  Releases bulkhead permits held by tasks.
  """
  @spec release_bulkhead_tasks([Task.t()]) :: :ok
  def release_bulkhead_tasks(tasks) do
    Enum.each(tasks, fn task ->
      send(task.pid, :release)
    end)

    Task.await_many(tasks, 5000)
    :ok
  end

  @doc """
  Waits for a circuit breaker to reach a specific state.
  """
  @spec wait_for_circuit_breaker_state(atom(), atom(), timeout()) :: :ok | :timeout
  def wait_for_circuit_breaker_state(name, expected_state, timeout \\ 5000) do
    deadline = System.monotonic_time(:millisecond) + timeout

    wait_loop(fn ->
      CircuitBreaker.get_state(name) == expected_state
    end, deadline)
  end

  @doc """
  Asserts that a function eventually returns true within a timeout.
  """
  @spec eventually((() -> boolean()), timeout()) :: :ok | :timeout
  def eventually(fun, timeout \\ 1000) do
    deadline = System.monotonic_time(:millisecond) + timeout
    wait_loop(fun, deadline)
  end

  defp wait_loop(fun, deadline) do
    if fun.() do
      :ok
    else
      now = System.monotonic_time(:millisecond)

      if now >= deadline do
        :timeout
      else
        Process.sleep(10)
        wait_loop(fun, deadline)
      end
    end
  end
end
