defmodule MfaService.Passkeys.Config do
  @moduledoc """
  Configuration for WebAuthn/Passkeys.
  """

  @doc """
  Get the Relying Party ID (domain).
  """
  def rp_id do
    Application.get_env(:mfa_service, :passkeys)[:rp_id] || "localhost"
  end

  @doc """
  Get the Relying Party name.
  """
  def rp_name do
    Application.get_env(:mfa_service, :passkeys)[:rp_name] || "Auth Platform"
  end

  @doc """
  Get the allowed origins for WebAuthn.
  """
  def origins do
    Application.get_env(:mfa_service, :passkeys)[:origins] || ["https://localhost"]
  end

  @doc """
  Get the attestation preference.
  """
  def attestation do
    Application.get_env(:mfa_service, :passkeys)[:attestation] || "direct"
  end

  @doc """
  Get the timeout for WebAuthn ceremonies (in milliseconds).
  """
  def timeout do
    Application.get_env(:mfa_service, :passkeys)[:timeout] || 60_000
  end

  @doc """
  Get supported public key algorithms.
  Returns list of COSE algorithm identifiers.
  """
  def pub_key_cred_params do
    [
      %{type: "public-key", alg: -7},    # ES256 (ECDSA w/ SHA-256)
      %{type: "public-key", alg: -257},  # RS256 (RSASSA-PKCS1-v1_5 w/ SHA-256)
      %{type: "public-key", alg: -37},   # PS256 (RSASSA-PSS w/ SHA-256)
      %{type: "public-key", alg: -8}     # EdDSA
    ]
  end
end
