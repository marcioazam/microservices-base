defmodule MfaService.MixProject do
  use Mix.Project

  def project do
    [
      app: :mfa_service,
      version: "0.1.0",
      elixir: "~> 1.15",
      start_permanent: Mix.env() == :prod,
      deps: deps()
    ]
  end

  def application do
    [
      extra_applications: [:logger, :crypto],
      mod: {MfaService.Application, []}
    ]
  end

  defp deps do
    [
      {:grpc, "~> 0.7"},
      {:protobuf, "~> 0.12"},
      {:jason, "~> 1.4"},
      {:redix, "~> 1.2"},
      {:ecto_sql, "~> 3.10"},
      {:postgrex, ">= 0.0.0"},
      # WebAuthn/Passkeys dependencies
      {:cbor, "~> 1.0"},
      # Property-based testing
      {:stream_data, "~> 0.6", only: [:test, :dev]}
    ]
  end
end
