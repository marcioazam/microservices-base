defmodule SessionIdentityCore.Sessions.SessionManager do
  @moduledoc """
  Session lifecycle management with security features.
  
  Implements:
  - 256-bit entropy token generation
  - Device binding on session creation
  - Session fixation protection
  - Risk-based session handling
  """

  alias AuthPlatform.Security
  alias AuthPlatform.Clients.Logging
  alias SessionIdentityCore.Sessions.{Session, SessionStore}
  alias SessionIdentityCore.Identity.RiskScorer
  alias SessionIdentityCore.Shared.{TTL, Errors}

  @token_bytes 32  # 256 bits

  @doc """
  Creates a new session with device binding and risk assessment.
  
  Requires device_fingerprint and ip_address for security binding.
  """
  @spec create_session(map()) :: {:ok, Session.t()} | {:error, term()}
  def create_session(attrs) do
    with :ok <- validate_device_binding(attrs),
         session_id <- generate_session_token(),
         risk_score <- calculate_initial_risk(attrs),
         session <- build_session(session_id, attrs, risk_score),
         {:ok, stored} <- SessionStore.store_session(session) do
      Logging.info("Session created",
        session_id: session_id,
        user_id: attrs[:user_id],
        risk_score: risk_score
      )
      {:ok, stored}
    end
  end

  @doc """
  Regenerates session ID for fixation protection.
  
  Called on privilege escalation (e.g., MFA verification).
  """
  @spec regenerate_session(String.t(), String.t()) :: {:ok, Session.t()} | {:error, term()}
  def regenerate_session(old_session_id, user_id) do
    with {:ok, old_session} <- SessionStore.get_session(old_session_id),
         :ok <- SessionStore.delete_session(old_session_id, user_id),
         new_session_id <- generate_session_token(),
         new_session <- %{old_session | id: new_session_id},
         {:ok, stored} <- SessionStore.store_session(new_session) do
      Logging.info("Session regenerated for fixation protection",
        old_session_id: old_session_id,
        new_session_id: new_session_id,
        user_id: user_id
      )
      {:ok, stored}
    end
  end

  @doc """
  Marks a session as MFA verified and regenerates ID.
  """
  @spec verify_mfa(String.t(), String.t()) :: {:ok, Session.t()} | {:error, term()}
  def verify_mfa(session_id, user_id) do
    with {:ok, session} <- SessionStore.get_session(session_id),
         :ok <- SessionStore.delete_session(session_id, user_id),
         new_session_id <- generate_session_token(),
         updated <- %{session | id: new_session_id, mfa_verified: true},
         {:ok, stored} <- SessionStore.store_session(updated) do
      Logging.info("MFA verified, session regenerated",
        old_session_id: session_id,
        new_session_id: new_session_id,
        user_id: user_id
      )
      {:ok, stored}
    end
  end

  @doc """
  Terminates a session with reason tracking.
  """
  @spec terminate_session(String.t(), String.t(), atom()) :: :ok | {:error, term()}
  def terminate_session(session_id, user_id, reason \\ :user_logout) do
    with :ok <- SessionStore.delete_session(session_id, user_id) do
      Logging.info("Session terminated",
        session_id: session_id,
        user_id: user_id,
        reason: reason
      )
      :ok
    end
  end

  @doc """
  Terminates all sessions for a user.
  """
  @spec terminate_all_sessions(String.t(), atom()) :: :ok | {:error, term()}
  def terminate_all_sessions(user_id, reason \\ :user_logout_all) do
    with {:ok, sessions} <- SessionStore.get_user_sessions(user_id) do
      Enum.each(sessions, fn session ->
        SessionStore.delete_session(session.id, user_id)
      end)

      Logging.info("All sessions terminated",
        user_id: user_id,
        session_count: length(sessions),
        reason: reason
      )
      :ok
    end
  end

  @doc """
  Generates a cryptographically secure session token with 256-bit entropy.
  """
  @spec generate_session_token() :: String.t()
  def generate_session_token do
    Security.generate_token(@token_bytes, encoding: :url_safe_base64)
  end

  # Private functions

  defp validate_device_binding(attrs) do
    if is_binary(attrs[:device_fingerprint]) and is_binary(attrs[:ip_address]) do
      :ok
    else
      Errors.missing_device_binding()
    end
  end

  defp calculate_initial_risk(attrs) do
    context = %{
      known_devices: Map.get(attrs, :known_devices, []),
      known_ips: Map.get(attrs, :known_ips, []),
      recent_failed_attempts: Map.get(attrs, :recent_failed_attempts, 0)
    }

    session_data = %{
      ip_address: attrs[:ip_address],
      device_fingerprint: attrs[:device_fingerprint],
      user_id: attrs[:user_id]
    }

    RiskScorer.calculate_risk(session_data, context)
  end

  defp build_session(session_id, attrs, risk_score) do
    now = DateTime.utc_now()

    %Session{
      id: session_id,
      user_id: attrs[:user_id],
      device_id: attrs[:device_id],
      ip_address: attrs[:ip_address],
      user_agent: attrs[:user_agent],
      device_fingerprint: attrs[:device_fingerprint],
      risk_score: risk_score,
      mfa_verified: false,
      expires_at: TTL.default_expiry(),
      last_activity: now,
      inserted_at: now,
      updated_at: now
    }
  end
end
