defmodule AuthPlatformUmbrella.MixProject do
  use Mix.Project

  def project do
    [
      apps_path: "apps",
      version: "0.1.0",
      start_permanent: Mix.env() == :prod,
      deps: deps(),
      aliases: aliases(),
      dialyzer: dialyzer(),
      test_coverage: [tool: ExCoveralls],
      preferred_cli_env: [
        coveralls: :test,
        "coveralls.detail": :test,
        "coveralls.html": :test
      ],
      # Docs
      name: "Auth Platform Elixir",
      source_url: "https://github.com/auth-platform/libs-elixir",
      docs: docs()
    ]
  end

  defp deps do
    [
      # Development & Testing
      {:dialyxir, "~> 1.4", only: [:dev, :test], runtime: false},
      {:credo, "~> 1.7", only: [:dev, :test], runtime: false},
      {:ex_doc, "~> 0.31", only: :dev, runtime: false},
      {:excoveralls, "~> 0.18", only: :test}
    ]
  end

  defp aliases do
    [
      quality: ["format --check-formatted", "credo --strict", "dialyzer"],
      "test.all": ["test --cover"]
    ]
  end

  defp dialyzer do
    [
      plt_file: {:no_warn, "priv/plts/dialyzer.plt"},
      plt_add_apps: [:mix, :ex_unit],
      flags: [
        :error_handling,
        :underspecs,
        :unmatched_returns
      ]
    ]
  end

  defp docs do
    [
      main: "readme",
      extras: ["README.md", "CHANGELOG.md"],
      groups_for_modules: [
        "Functional Types": [
          AuthPlatform.Functional.Result,
          AuthPlatform.Functional.Option
        ],
        "Error Handling": [
          AuthPlatform.Errors.AppError
        ],
        Validation: [
          AuthPlatform.Validation
        ],
        "Domain Primitives": [
          AuthPlatform.Domain.Email,
          AuthPlatform.Domain.UUID,
          AuthPlatform.Domain.ULID,
          AuthPlatform.Domain.Money,
          AuthPlatform.Domain.PhoneNumber,
          AuthPlatform.Domain.URL
        ],
        Resilience: [
          AuthPlatform.Resilience.CircuitBreaker,
          AuthPlatform.Resilience.Retry,
          AuthPlatform.Resilience.RateLimiter,
          AuthPlatform.Resilience.Bulkhead
        ],
        Codecs: [
          AuthPlatform.Codec.JSON,
          AuthPlatform.Codec.Base64
        ],
        Security: [
          AuthPlatform.Security
        ],
        Observability: [
          AuthPlatform.Observability.Logger,
          AuthPlatform.Observability.Telemetry
        ]
      ]
    ]
  end
end
