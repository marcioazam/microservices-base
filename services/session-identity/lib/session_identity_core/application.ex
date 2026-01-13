defmodule SessionIdentityCore.Application do
  @moduledoc false

  use Application

  @impl true
  def start(_type, _args) do
    # Validate configuration at startup
    :ok = SessionIdentityCore.Config.validate!()

    # Setup OpenTelemetry
    OpenTelemetry.register_application_tracer(:session_identity_core)

    children = [
      # Telemetry supervisor
      {Telemetry.Metrics.ConsoleReporter, metrics: SessionIdentityCore.Telemetry.Metrics.metrics()},

      # Core infrastructure
      SessionIdentityCore.Repo,
      {Phoenix.PubSub, name: SessionIdentityCore.PubSub},
      {Redix, name: :redix, host: redis_host(), port: redis_port()},

      # Application services
      SessionIdentityCore.Sessions.SessionManager,

      # Web endpoints
      SessionIdentityCoreWeb.Endpoint,
      {GRPC.Server.Supervisor, endpoint: SessionIdentityCore.GRPC.Endpoint, port: grpc_port()}
    ]

    opts = [strategy: :one_for_one, name: SessionIdentityCore.Supervisor]
    Supervisor.start_link(children, opts)
  end

  defp redis_host, do: System.get_env("REDIS_HOST", "localhost")
  defp redis_port, do: String.to_integer(System.get_env("REDIS_PORT", "6379"))
  defp grpc_port, do: String.to_integer(System.get_env("GRPC_PORT", "50053"))
end
