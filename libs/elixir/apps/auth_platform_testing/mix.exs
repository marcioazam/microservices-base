defmodule AuthPlatformTesting.MixProject do
  use Mix.Project

  @version "0.1.0"

  def project do
    [
      app: :auth_platform_testing,
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
      # Docs
      name: "AuthPlatformTesting",
      docs: [
        main: "AuthPlatformTesting",
        extras: ["README.md"]
      ]
    ]
  end

  def application do
    [
      extra_applications: [:logger]
    ]
  end

  defp elixirc_paths(:test), do: ["lib", "test/support"]
  defp elixirc_paths(_), do: ["lib"]

  defp deps do
    [
      # Core library dependency
      {:auth_platform, in_umbrella: true},

      # Property-based testing
      {:stream_data, "~> 1.0"},

      # Development & Testing
      {:dialyxir, "~> 1.4", only: [:dev, :test], runtime: false},
      {:credo, "~> 1.7", only: [:dev, :test], runtime: false},
      {:ex_doc, "~> 0.31", only: :dev, runtime: false}
    ]
  end

  defp dialyzer do
    [
      plt_file: {:no_warn, "../../priv/plts/auth_platform_testing.plt"},
      plt_add_apps: [:mix, :ex_unit, :stream_data]
    ]
  end
end
