defmodule SessionIdentityCore.CAEP.Emitter do
  @moduledoc """
  CAEP (Continuous Access Evaluation Protocol) event emitter.
  
  Implements SSF (Shared Signals Framework) compliant event format.
  
  ## Event Types
  
  - session-revoked: Emitted on logout, admin termination, security violation
  
  ## Configuration
  
  Set CAEP_ENABLED=true to enable event emission.
  """

  alias AuthPlatform.Clients.Logging
  alias SessionIdentityCore.Shared.DateTime, as: DT

  @caep_enabled_env "CAEP_ENABLED"

  @doc """
  Emits a session-revoked event for user logout.
  """
  @spec emit_logout(map()) :: :ok | {:error, term()}
  def emit_logout(session) do
    emit_session_revoked(session, :user_logout, nil)
  end

  @doc """
  Emits a session-revoked event for admin termination.
  """
  @spec emit_admin_termination(map(), String.t()) :: :ok | {:error, term()}
  def emit_admin_termination(session, admin_id) do
    emit_session_revoked(session, :admin_termination, admin_id)
  end

  @doc """
  Emits a session-revoked event for security violation.
  """
  @spec emit_security_violation(map(), atom()) :: :ok | {:error, term()}
  def emit_security_violation(session, violation_type) do
    emit_session_revoked(session, violation_type, nil)
  end

  @doc """
  Checks if CAEP is enabled.
  """
  @spec enabled?() :: boolean()
  def enabled? do
    System.get_env(@caep_enabled_env, "false") == "true"
  end

  @doc """
  Builds an SSF-compliant event structure.
  """
  @spec build_event(String.t(), map(), atom(), String.t() | nil) :: map()
  def build_event(event_type, session, reason, admin_id) do
    %{
      "event_type" => event_type,
      "subject" => build_subject(session),
      "event_timestamp" => DT.to_iso8601(DateTime.utc_now()),
      "reason_admin" => build_reason_admin(reason, admin_id),
      "txn" => generate_transaction_id()
    }
  end

  # Private functions

  defp emit_session_revoked(session, reason, admin_id) do
    if enabled?() do
      event = build_event("session-revoked", session, reason, admin_id)
      do_emit(event)
    else
      :ok
    end
  end

  defp do_emit(event) do
    # In production, this would send to SSF receiver endpoint
    # For now, log and return success
    case send_to_receiver(event) do
      :ok ->
        Logging.info("CAEP event emitted",
          event_type: event["event_type"],
          subject: event["subject"]["sub"],
          reason: event["reason_admin"]["reason"]
        )
        :ok

      {:error, reason} = error ->
        # Log failure but don't fail the operation
        Logging.error("CAEP emission failed",
          event_type: event["event_type"],
          error: inspect(reason)
        )
        error
    end
  end

  defp send_to_receiver(_event) do
    # Placeholder - implement actual SSF receiver communication
    :ok
  end

  defp build_subject(session) do
    %{
      "format" => "opaque",
      "iss" => get_issuer(),
      "sub" => session[:user_id] || session["user_id"]
    }
  end

  defp build_reason_admin(reason, nil) do
    %{
      "reason" => Atom.to_string(reason)
    }
  end

  defp build_reason_admin(reason, admin_id) do
    %{
      "reason" => Atom.to_string(reason),
      "admin_id" => admin_id
    }
  end

  defp get_issuer do
    Application.get_env(:session_identity_core, :issuer, "https://auth.example.com")
  end

  defp generate_transaction_id do
    :crypto.strong_rand_bytes(16) |> Base.url_encode64(padding: false)
  end
end
