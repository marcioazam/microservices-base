defmodule SessionIdentityCore.Crypto.HealthCheck do
  @moduledoc """
  Health check integration for crypto-service.
  
  Provides crypto-service status for readiness probes.
  """

  alias SessionIdentityCore.Crypto.{Client, Config, CircuitBreaker}

  @doc """
  Checks crypto-service health status.
  
  Returns :ok if healthy, :degraded if using fallback,
  or :unhealthy if completely unavailable.
  """
  @spec check() :: :ok | :degraded | :unhealthy
  def check do
    config = Config.get()
    
    cond do
      not config.enabled ->
        :ok  # Crypto disabled, not a health issue
      
      CircuitBreaker.open?() ->
        :degraded  # Using fallback
      
      true ->
        check_crypto_service()
    end
  end

  @doc """
  Returns health check result as map for health endpoint.
  """
  @spec status() :: map()
  def status do
    case check() do
      :ok -> %{crypto_service: :ok, using_fallback: false}
      :degraded -> %{crypto_service: :degraded, using_fallback: true}
      :unhealthy -> %{crypto_service: :unhealthy, using_fallback: true}
    end
  end

  @doc """
  Checks if crypto-service is required for readiness.
  """
  @spec required_for_readiness?() :: boolean()
  def required_for_readiness? do
    config = Config.get()
    config.enabled and config.required_for_readiness
  end

  defp check_crypto_service do
    case Client.health_check() do
      {:ok, _} -> :ok
      {:error, _} -> :unhealthy
    end
  end
end
