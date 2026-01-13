defmodule AuthPlatform.Resilience.Supervisor do
  @moduledoc """
  Supervisor for resilience components.

  This supervisor manages the lifecycle of resilience components:
  - Circuit Breakers
  - Rate Limiters
  - Bulkheads

  It uses a DynamicSupervisor to allow starting and stopping components
  at runtime.

  ## Usage

      # Start a circuit breaker
      AuthPlatform.Resilience.Supervisor.start_circuit_breaker(:my_breaker, %{
        failure_threshold: 5,
        timeout_ms: 30_000
      })

      # Start a rate limiter
      AuthPlatform.Resilience.Supervisor.start_rate_limiter(:my_limiter, %{
        rate: 100,
        burst_size: 100
      })

      # Stop a component
      AuthPlatform.Resilience.Supervisor.stop_child(:my_breaker)

  """
  use DynamicSupervisor

  @supervisor_name __MODULE__

  @doc """
  Starts the resilience supervisor.
  """
  @spec start_link(keyword()) :: Supervisor.on_start()
  def start_link(opts \\ []) do
    DynamicSupervisor.start_link(__MODULE__, opts, name: @supervisor_name)
  end

  @impl true
  def init(_opts) do
    DynamicSupervisor.init(strategy: :one_for_one)
  end

  @doc """
  Starts a circuit breaker with the given name and configuration.

  ## Options

    * `:failure_threshold` - Number of failures before opening (default: 5)
    * `:success_threshold` - Successes needed to close from half-open (default: 2)
    * `:timeout_ms` - Time in open state before half-open (default: 30_000)
    * `:half_open_max_requests` - Max requests in half-open state (default: 3)

  ## Examples

      iex> AuthPlatform.Resilience.Supervisor.start_circuit_breaker(:api_breaker, %{
      ...>   failure_threshold: 5,
      ...>   timeout_ms: 30_000
      ...> })
      {:ok, #PID<0.123.0>}

  """
  @spec start_circuit_breaker(atom(), map()) :: DynamicSupervisor.on_start_child()
  def start_circuit_breaker(name, config \\ %{}) when is_atom(name) do
    child_spec = {AuthPlatform.Resilience.CircuitBreaker, [name: name, config: config]}
    DynamicSupervisor.start_child(@supervisor_name, child_spec)
  end

  @doc """
  Starts a rate limiter with the given name and configuration.

  ## Options

    * `:rate` - Tokens per second (default: 100)
    * `:burst_size` - Maximum tokens (default: 100)

  ## Examples

      iex> AuthPlatform.Resilience.Supervisor.start_rate_limiter(:api_limiter, %{
      ...>   rate: 100,
      ...>   burst_size: 150
      ...> })
      {:ok, #PID<0.124.0>}

  """
  @spec start_rate_limiter(atom(), map()) :: DynamicSupervisor.on_start_child()
  def start_rate_limiter(name, config \\ %{}) when is_atom(name) do
    child_spec = {AuthPlatform.Resilience.RateLimiter, [name: name, config: config]}
    DynamicSupervisor.start_child(@supervisor_name, child_spec)
  end

  @doc """
  Starts a bulkhead with the given name and configuration.

  ## Options

    * `:max_concurrent` - Maximum concurrent executions (default: 10)
    * `:max_queue` - Maximum queued requests (default: 100)
    * `:queue_timeout_ms` - Queue timeout in milliseconds (default: 5000)

  ## Examples

      iex> AuthPlatform.Resilience.Supervisor.start_bulkhead(:api_bulkhead, %{
      ...>   max_concurrent: 10,
      ...>   max_queue: 50
      ...> })
      {:ok, #PID<0.125.0>}

  """
  @spec start_bulkhead(atom(), map()) :: DynamicSupervisor.on_start_child()
  def start_bulkhead(name, config \\ %{}) when is_atom(name) do
    child_spec = {AuthPlatform.Resilience.Bulkhead, [name: name, config: config]}
    DynamicSupervisor.start_child(@supervisor_name, child_spec)
  end

  @doc """
  Stops a child process by name.

  ## Examples

      iex> AuthPlatform.Resilience.Supervisor.stop_child(:my_breaker)
      :ok

  """
  @spec stop_child(atom()) :: :ok | {:error, :not_found}
  def stop_child(name) when is_atom(name) do
    case AuthPlatform.Resilience.Registry.lookup(name) do
      {:ok, pid} ->
        DynamicSupervisor.terminate_child(@supervisor_name, pid)

      {:error, :not_found} ->
        {:error, :not_found}
    end
  end

  @doc """
  Returns all children managed by this supervisor.
  """
  @spec which_children() :: [{:undefined, pid() | :restarting, :worker | :supervisor, [module()]}]
  def which_children do
    DynamicSupervisor.which_children(@supervisor_name)
  end

  @doc """
  Returns the count of active children.
  """
  @spec count_children() :: %{
          specs: non_neg_integer(),
          active: non_neg_integer(),
          supervisors: non_neg_integer(),
          workers: non_neg_integer()
        }
  def count_children do
    DynamicSupervisor.count_children(@supervisor_name)
  end
end
