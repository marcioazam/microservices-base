defmodule SessionIdentityCore.Shared.Errors do
  @moduledoc """
  Centralized error definitions for session identity service.
  
  All errors MUST be defined and returned through this module to ensure:
  - Consistent error format across the service
  - RFC 6749 compliant OAuth error responses
  - Easy error handling and logging
  
  ## Error Categories
  
  - Session Errors: session_not_found, session_expired, session_invalid
  - OAuth Errors: RFC 6749 compliant (invalid_request, invalid_client, etc.)
  - PKCE Errors: pkce_required, pkce_plain_not_allowed, invalid_code_verifier
  - Event Store Errors: event_not_found, aggregate_not_found
  """

  # ===========================================================================
  # Session Errors
  # ===========================================================================

  @doc "Returns session not found error."
  @spec session_not_found() :: {:error, :session_not_found}
  def session_not_found, do: {:error, :session_not_found}

  @doc "Returns session expired error."
  @spec session_expired() :: {:error, :session_expired}
  def session_expired, do: {:error, :session_expired}

  @doc "Returns session invalid error."
  @spec session_invalid() :: {:error, :session_invalid}
  def session_invalid, do: {:error, :session_invalid}

  @doc "Returns session fixation error."
  @spec session_fixation() :: {:error, :session_fixation}
  def session_fixation, do: {:error, :session_fixation}

  @doc "Returns missing device binding error."
  @spec missing_device_binding() :: {:error, :missing_device_binding}
  def missing_device_binding, do: {:error, :missing_device_binding}

  # ===========================================================================
  # OAuth Errors (RFC 6749 Compliant)
  # ===========================================================================

  @doc """
  Creates an RFC 6749 compliant OAuth error response.
  
  ## Examples
  
      iex> Errors.oauth_error("invalid_request", "Missing parameter")
      {:error, %{error: "invalid_request", error_description: "Missing parameter"}}
  """
  @spec oauth_error(String.t(), String.t()) :: {:error, map()}
  def oauth_error(error, description) when is_binary(error) and is_binary(description) do
    {:error, %{error: error, error_description: description}}
  end

  @doc "Returns invalid_request OAuth error."
  @spec invalid_request(String.t()) :: {:error, map()}
  def invalid_request(description) do
    oauth_error("invalid_request", description)
  end

  @doc "Returns invalid_client OAuth error."
  @spec invalid_client() :: {:error, map()}
  def invalid_client do
    oauth_error("invalid_client", "Client authentication failed")
  end

  @doc "Returns invalid_grant OAuth error."
  @spec invalid_grant() :: {:error, map()}
  def invalid_grant do
    oauth_error("invalid_grant", "Invalid authorization grant")
  end

  @doc "Returns invalid_grant OAuth error with custom description."
  @spec invalid_grant(String.t()) :: {:error, map()}
  def invalid_grant(description) do
    oauth_error("invalid_grant", description)
  end

  @doc "Returns unsupported_grant_type OAuth error."
  @spec unsupported_grant_type(String.t()) :: {:error, map()}
  def unsupported_grant_type(grant_type) do
    oauth_error("unsupported_grant_type", "Grant type '#{grant_type}' is not supported")
  end

  @doc "Returns unsupported_response_type OAuth error."
  @spec unsupported_response_type(String.t()) :: {:error, map()}
  def unsupported_response_type(response_type) do
    oauth_error("unsupported_response_type", "Response type '#{response_type}' is not supported")
  end

  @doc "Returns unauthorized_client OAuth error."
  @spec unauthorized_client() :: {:error, map()}
  def unauthorized_client do
    oauth_error("unauthorized_client", "Client is not authorized for this grant type")
  end

  @doc "Returns access_denied OAuth error."
  @spec access_denied() :: {:error, map()}
  def access_denied do
    oauth_error("access_denied", "The resource owner denied the request")
  end

  @doc "Returns invalid_scope OAuth error."
  @spec invalid_scope() :: {:error, map()}
  def invalid_scope do
    oauth_error("invalid_scope", "The requested scope is invalid or unknown")
  end

  # ===========================================================================
  # PKCE Errors
  # ===========================================================================

  @doc "Returns PKCE required error."
  @spec pkce_required() :: {:error, map()}
  def pkce_required do
    invalid_request("PKCE is required. code_challenge parameter is missing")
  end

  @doc "Returns PKCE plain method not allowed error."
  @spec pkce_plain_not_allowed() :: {:error, map()}
  def pkce_plain_not_allowed do
    invalid_request("code_challenge_method 'plain' is not allowed. Use 'S256'")
  end

  @doc "Returns invalid code verifier error."
  @spec invalid_code_verifier() :: {:error, :invalid_code_verifier}
  def invalid_code_verifier, do: {:error, :invalid_code_verifier}

  @doc "Returns code verifier too short error."
  @spec code_verifier_too_short() :: {:error, :code_verifier_too_short}
  def code_verifier_too_short, do: {:error, :code_verifier_too_short}

  @doc "Returns code verifier too long error."
  @spec code_verifier_too_long() :: {:error, :code_verifier_too_long}
  def code_verifier_too_long, do: {:error, :code_verifier_too_long}

  @doc "Returns invalid code challenge error."
  @spec invalid_code_challenge() :: {:error, :invalid_code_challenge}
  def invalid_code_challenge, do: {:error, :invalid_code_challenge}

  @doc "Returns unsupported PKCE method error."
  @spec unsupported_pkce_method() :: {:error, :unsupported_method}
  def unsupported_pkce_method, do: {:error, :unsupported_method}

  # ===========================================================================
  # Event Store Errors
  # ===========================================================================

  @doc "Returns event not found error."
  @spec event_not_found() :: {:error, :event_not_found}
  def event_not_found, do: {:error, :event_not_found}

  @doc "Returns aggregate not found error."
  @spec aggregate_not_found() :: {:error, :aggregate_not_found}
  def aggregate_not_found, do: {:error, :aggregate_not_found}

  @doc "Returns concurrency conflict error."
  @spec concurrency_conflict() :: {:error, :concurrency_conflict}
  def concurrency_conflict, do: {:error, :concurrency_conflict}

  # ===========================================================================
  # ID Token Errors
  # ===========================================================================

  @doc "Returns missing required claims error."
  @spec missing_required_claims(list()) :: {:error, {:missing_claims, list()}}
  def missing_required_claims(claims) when is_list(claims) do
    {:error, {:missing_claims, claims}}
  end

  @doc "Returns invalid token error."
  @spec invalid_token() :: {:error, :invalid_token}
  def invalid_token, do: {:error, :invalid_token}

  # ===========================================================================
  # CAEP Errors
  # ===========================================================================

  @doc "Returns CAEP emission failed error."
  @spec caep_emission_failed(term()) :: {:error, {:caep_failed, term()}}
  def caep_emission_failed(reason) do
    {:error, {:caep_failed, reason}}
  end

  # ===========================================================================
  # Helper Functions
  # ===========================================================================

  @doc """
  Checks if an error is an OAuth error (has error and error_description).
  """
  @spec oauth_error?(term()) :: boolean()
  def oauth_error?({:error, %{error: _, error_description: _}}), do: true
  def oauth_error?(_), do: false

  @doc """
  Extracts error code from an error tuple.
  """
  @spec error_code(term()) :: atom() | String.t() | nil
  def error_code({:error, %{error: code}}), do: code
  def error_code({:error, code}) when is_atom(code), do: code
  def error_code({:error, {code, _}}) when is_atom(code), do: code
  def error_code(_), do: nil
end
