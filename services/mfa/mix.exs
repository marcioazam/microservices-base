defmodule MfaService.MixProject do
  use Mix.Project

  def project do
    [
      app: :mfa_service,
      version: "0.2.0",
      elixir: "~> 1.17",
      start_permanent: Mix.env() == :prod,
      deps: deps(),
      elixirc_paths: elixirc_paths(Mix.env()),
      test_coverage: [tool: ExCoveralls],
      preferred_cli_env: [
        coveralls: :test,
        "coveralls.detail": :test,
        "coveralls.html": :test
      ]
    ]
  end

  def application do
    [
      extra_applications: [:logger, :crypto],
      mod: {MfaService.Application, []}
    ]
  end

  defp elixirc_paths(:test), do: ["lib", "test/support"]
  defp elixirc_paths(_), do: ["lib"]

  defp deps do
    [
      # Platform libs
      {:auth_platform, path: "../../libs/elixir/apps/auth_platform"},
      {:auth_platform_clients, path: "../../libs/elixir/apps/auth_platform_clients"},
      # gRPC
      {:grpc, "~> 0.9"},
      {:protobuf, "~> 0.13"},
      # JSON
      {:jason, "~> 1.4"},
      # Database
      {:ecto_sql, "~> 3.12"},
      {:postgrex, "~> 0.19"},
      # WebAuthn/Passkeys
      {:cbor, "~> 1.0"},
      # HTTP client for CAEP
      {:req, "~> 0.5"},
      # Testing
      {:stream_data, "~> 1.1", only: [:test, :dev]},
      {:excoveralls, "~> 0.18", only: :test},
      {:mox, "~> 1.2", only: :test},
      {:benchee, "~> 1.3", only: [:dev, :test]}
    ]
  end
end
