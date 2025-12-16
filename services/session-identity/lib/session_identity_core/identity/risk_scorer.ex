defmodule SessionIdentityCore.Identity.RiskScorer do
  @moduledoc """
  Risk scoring for sessions based on device, behavior, and context.
  """

  @step_up_threshold 0.7

  defstruct [
    :ip_risk,
    :device_risk,
    :behavior_risk,
    :time_risk,
    :location_risk
  ]

  @doc """
  Calculate overall risk score for a session.
  Returns a score between 0.0 (low risk) and 1.0 (high risk).
  """
  def calculate_risk(session, context \\ %{}) do
    factors = %{
      ip_risk: calculate_ip_risk(session.ip_address, context),
      device_risk: calculate_device_risk(session.device_fingerprint, context),
      behavior_risk: calculate_behavior_risk(session.user_id, context),
      time_risk: calculate_time_risk(context),
      location_risk: calculate_location_risk(context)
    }

    # Weighted average of risk factors
    weights = %{
      ip_risk: 0.2,
      device_risk: 0.3,
      behavior_risk: 0.25,
      time_risk: 0.1,
      location_risk: 0.15
    }

    score =
      Enum.reduce(factors, 0.0, fn {key, value}, acc ->
        acc + value * Map.get(weights, key, 0.0)
      end)

    min(max(score, 0.0), 1.0)
  end

  @doc """
  Determines if step-up authentication is required based on risk score.
  """
  def requires_step_up?(risk_score) when risk_score >= @step_up_threshold, do: true
  def requires_step_up?(_), do: false

  @doc """
  Get required authentication factors based on risk level.
  """
  def get_required_factors(risk_score) do
    cond do
      risk_score >= 0.9 -> [:webauthn, :totp]
      risk_score >= 0.7 -> [:totp]
      risk_score >= 0.5 -> [:email_verification]
      true -> []
    end
  end

  # Private functions for calculating individual risk factors

  defp calculate_ip_risk(ip_address, context) do
    known_ips = Map.get(context, :known_ips, [])
    
    cond do
      ip_address in known_ips -> 0.0
      is_vpn_or_proxy?(ip_address) -> 0.6
      is_tor_exit_node?(ip_address) -> 0.9
      true -> 0.3
    end
  end

  defp calculate_device_risk(device_fingerprint, context) do
    known_devices = Map.get(context, :known_devices, [])
    
    if device_fingerprint in known_devices do
      0.0
    else
      0.5
    end
  end

  defp calculate_behavior_risk(user_id, context) do
    # In production, this would analyze login patterns, failed attempts, etc.
    failed_attempts = Map.get(context, :recent_failed_attempts, 0)
    
    cond do
      failed_attempts >= 5 -> 0.9
      failed_attempts >= 3 -> 0.6
      failed_attempts >= 1 -> 0.3
      true -> 0.0
    end
  end

  defp calculate_time_risk(context) do
    hour = Map.get(context, :hour, DateTime.utc_now().hour)
    
    # Higher risk for unusual hours (midnight to 5am)
    if hour >= 0 and hour < 5 do
      0.4
    else
      0.0
    end
  end

  defp calculate_location_risk(context) do
    # In production, this would check for impossible travel, etc.
    Map.get(context, :location_risk, 0.0)
  end

  # Placeholder functions - in production these would use external services
  defp is_vpn_or_proxy?(_ip), do: false
  defp is_tor_exit_node?(_ip), do: false
end
