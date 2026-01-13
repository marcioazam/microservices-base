defmodule AuthPlatform.MixProject do
  use Mix.Project

  @version "0.1.0"

  def project do
    [
      app: :auth_platform,
      version: @version,
      build_path: "../../_build",
      config_path: "../../config/config.exs",
      deps_path: "../../deps",
      lockfile: "../../mix.lock",
      elixir: "~> 1.15",
      elixirc_paths: elixirc_paths(Mix.env()),
      start_permanent: Mix.env() == :prod,
      deps: deps(),
      dialyzer: dialyzer(),
      test_coverage: [tool: ExCoveralls],
      preferred_cli_env: [
        coveralls: :test,
        "coveralls.detail": :test,
        "coveralls.html": :test
      ],
      # Docs
      name: "AuthPlatform",
      docs: [
        main: "AuthPlatform",
        extras: ["README.md"]
      ]
    ]
  end

  def application do
    [
      extra_applications: [:logger, :crypto],
      mod: {AuthPlatform.Application, []}
    ]
  end

  defp elixirc_paths(:test), do: ["lib", "test/support"]
  defp elixirc_paths(_), do: ["lib"]

  defp deps do
    [
      # JSON encoding/decoding
      {:jason, "~> 1.4"},

      # Telemetry for observability
      {:telemetry, "~> 1.2"},

      # OpenTelemetry for distributed tracing
      {:opentelemetry_api, "~> 1.2"},

      # Property-based testing
      {:stream_data, "~> 1.0", only: [:dev, :test]},

      # Development & Testing
      {:dialyxir, "~> 1.4", only: [:dev, :test], runtime: false},
      {:credo, "~> 1.7", only: [:dev, :test], runtime: false},
      {:ex_doc, "~> 0.31", only: :dev, runtime: false},
      {:excoveralls, "~> 0.18", only: :test}
    ]
  end

  defp dialyzer do
    [
      plt_file: {:no_warn, "../../priv/plts/auth_platform.plt"},
      plt_add_apps: [:mix, :ex_unit],
      flags: [
        :error_handling,
        :underspecs,
        :unmatched_returns
      ]
    ]
  end
end
