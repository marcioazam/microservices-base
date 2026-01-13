defmodule MfaService.Application do
  @moduledoc """
  MFA Service Application.
  Initializes platform service clients with circuit breakers.
  """

  use Application

  alias AuthPlatform.Clients.{Cache, Logging}

  @impl true
  def start(_type, _args) do
    children = [
      MfaService.Repo,
      {Task.Supervisor, name: MfaService.TaskSupervisor},
      {Task, fn -> initialize_platform_clients() end},
      {GRPC.Server.Supervisor, endpoint: MfaService.GRPC.Endpoint, port: grpc_port()}
    ]

    opts = [strategy: :one_for_one, name: MfaService.Supervisor]

    case Supervisor.start_link(children, opts) do
      {:ok, pid} ->
        setup_telemetry()
        {:ok, pid}

      error ->
        error
    end
  end

  defp initialize_platform_clients do
    circuit_breaker_config = %{
      failure_threshold: 5,
      timeout_ms: 30_000
    }

    Cache.start_circuit_breaker(circuit_breaker_config)
    Logging.start_circuit_breaker(circuit_breaker_config)

    Logging.info("MFA Service platform clients initialized",
      cache: :connected,
      logging: :connected
    )
  end

  defp setup_telemetry do
    :telemetry.attach_many(
      "mfa-service-telemetry",
      [
        [:mfa_service, :totp, :validate],
        [:mfa_service, :webauthn, :authenticate],
        [:mfa_service, :passkey, :register],
        [:mfa_service, :passkey, :authenticate],
        [:mfa_service, :challenge, :generate],
        [:mfa_service, :fingerprint, :compute]
      ],
      &handle_telemetry_event/4,
      nil
    )
  end

  defp handle_telemetry_event(event, measurements, metadata, _config) do
    Logging.debug("Telemetry event",
      event: Enum.join(event, "."),
      duration_ms: Map.get(measurements, :duration, 0) / 1_000_000,
      metadata: metadata
    )
  end

  defp grpc_port do
    System.get_env("GRPC_PORT", "50055") |> String.to_integer()
  end
end
