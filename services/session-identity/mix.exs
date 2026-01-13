defmodule SessionIdentityCore.MixProject do
  use Mix.Project

  def project do
    [
      app: :session_identity_core,
      version: "0.1.0",
      elixir: "~> 1.15",
      start_permanent: Mix.env() == :prod,
      deps: deps(),
      aliases: aliases()
    ]
  end

  def application do
    [
      extra_applications: [:logger, :runtime_tools],
      mod: {SessionIdentityCore.Application, []}
    ]
  end

  defp deps do
    [
      # Platform dependencies
      {:auth_platform, path: "../../libs/elixir/apps/auth_platform"},
      {:auth_platform_clients, path: "../../libs/elixir/apps/auth_platform_clients"},

      # Phoenix & Web
      {:phoenix, "~> 1.7"},
      {:phoenix_pubsub, "~> 2.1"},

      # Database
      {:ecto_sql, "~> 3.10"},
      {:postgrex, ">= 0.0.0"},

      # Serialization
      {:jason, "~> 1.4"},

      # gRPC
      {:grpc, "~> 0.7"},
      {:protobuf, "~> 0.12"},
      {:google_protos, "~> 0.3"},

      # Circuit Breaker
      {:fuse, "~> 2.5"},

      # Cache (fallback when platform service unavailable)
      {:redix, "~> 1.2"},

      # Security
      {:argon2_elixir, "~> 4.0"},
      {:joken, "~> 2.6"},

      # Observability
      {:telemetry, "~> 1.2"},
      {:telemetry_metrics, "~> 0.6"},
      {:telemetry_poller, "~> 1.0"},
      {:opentelemetry, "~> 1.3"},
      {:opentelemetry_api, "~> 1.2"},
      {:opentelemetry_exporter, "~> 1.6"},

      # Testing
      {:stream_data, "~> 0.6", only: [:test, :dev]},
      {:ex_machina, "~> 2.7", only: :test},
      {:mox, "~> 1.1", only: :test}
    ]
  end

  defp aliases do
    [
      setup: ["deps.get", "ecto.setup"],
      "ecto.setup": ["ecto.create", "ecto.migrate", "run priv/repo/seeds.exs"],
      "ecto.reset": ["ecto.drop", "ecto.setup"],
      test: ["ecto.create --quiet", "ecto.migrate --quiet", "test"]
    ]
  end
end
