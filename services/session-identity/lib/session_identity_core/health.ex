defmodule SessionIdentityCore.Health do
  @moduledoc """
  Health check endpoints for Kubernetes probes.
  """

  alias AuthPlatform.Clients.Cache

  @doc """
  Liveness check - is the service running?
  """
  @spec liveness() :: {:ok, map()} | {:error, map()}
  def liveness do
    {:ok, %{status: "alive", timestamp: DateTime.utc_now()}}
  end

  @doc """
  Readiness check - is the service ready to accept traffic?
  """
  @spec readiness() :: {:ok, map()} | {:error, map()}
  def readiness do
    checks = %{
      cache: check_cache(),
      database: check_database()
    }

    all_healthy = Enum.all?(checks, fn {_, status} -> status == :ok end)

    if all_healthy do
      {:ok, %{status: "ready", checks: checks, timestamp: DateTime.utc_now()}}
    else
      {:error, %{status: "not_ready", checks: checks, timestamp: DateTime.utc_now()}}
    end
  end

  defp check_cache do
    case Cache.get("health_check") do
      {:ok, _} -> :ok
      {:error, _} -> :degraded
    end
  end

  defp check_database do
    # Placeholder - implement actual DB check
    :ok
  end
end
