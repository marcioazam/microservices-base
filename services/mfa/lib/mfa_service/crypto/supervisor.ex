defmodule MfaService.Crypto.Supervisor do
  @moduledoc """
  Supervisor for Crypto Service client components.
  Manages the lifecycle of the crypto client, circuit breaker, and key manager.
  """

  use Supervisor

  require Logger

  alias MfaService.Crypto.{Config, Client, CircuitBreaker, KeyManager, Telemetry}

  @doc """
  Starts the Crypto Supervisor.
  """
  def start_link(opts \\ []) do
    Supervisor.start_link(__MODULE__, opts, name: __MODULE__)
  end

  @impl true
  def init(_opts) do
    config = Config.load()
    
    children = [
      # Circuit breaker state management
      {CircuitBreaker, config: config},
      
      # Key manager with caching
      {KeyManager, config: config},
      
      # Telemetry handlers
      {Task, fn -> Telemetry.attach_handlers() end}
    ]

    # Log startup
    Logger.info("Starting Crypto Service supervisor",
      host: config.host,
      port: config.port)

    Supervisor.init(children, strategy: :one_for_one)
  end

  @doc """
  Performs health check on crypto service.
  Returns :ok if healthy, {:error, reason} otherwise.
  """
  @spec health_check() :: :ok | {:error, term()}
  def health_check do
    correlation_id = UUID.uuid4()
    
    case Client.health_check(correlation_id) do
      {:ok, _} -> :ok
      {:error, reason} -> {:error, reason}
    end
  end

  @doc """
  Returns the current status of the crypto client.
  """
  @spec status() :: map()
  def status do
    %{
      supervisor: Process.alive?(Process.whereis(__MODULE__)),
      circuit_breaker: CircuitBreaker.state(),
      health: health_check_status(),
      config: Config.load() |> Map.take([:host, :port])
    }
  end

  defp health_check_status do
    case health_check() do
      :ok -> :healthy
      {:error, _} -> :unhealthy
    end
  end
end
