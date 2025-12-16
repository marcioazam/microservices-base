defmodule MfaService.Application do
  @moduledoc false

  use Application

  @impl true
  def start(_type, _args) do
    children = [
      MfaService.Repo,
      {Redix, name: :redix, host: redis_host(), port: redis_port()},
      {GRPC.Server.Supervisor, endpoint: MfaService.GRPC.Endpoint, port: grpc_port()}
    ]

    opts = [strategy: :one_for_one, name: MfaService.Supervisor]
    Supervisor.start_link(children, opts)
  end

  defp redis_host, do: System.get_env("REDIS_HOST", "localhost")
  defp redis_port, do: String.to_integer(System.get_env("REDIS_PORT", "6379"))
  defp grpc_port, do: String.to_integer(System.get_env("GRPC_PORT", "50055"))
end
