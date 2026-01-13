defmodule AuthPlatform.Application do
  @moduledoc """
  Application module for AuthPlatform.

  Starts the supervision tree with:
  - Resilience Registry for component lookup
  - Resilience Supervisor for managing resilience components
  """
  use Application

  @impl true
  def start(_type, _args) do
    children = [
      # Registry for resilience components (must start before supervisor)
      AuthPlatform.Resilience.Registry,
      # DynamicSupervisor for resilience components
      AuthPlatform.Resilience.Supervisor
    ]

    opts = [strategy: :one_for_one, name: AuthPlatform.Supervisor]
    Supervisor.start_link(children, opts)
  end
end
