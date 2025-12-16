defmodule MfaService.Passkeys.Management do
  @moduledoc """
  Passkey credential management: list, rename, delete.
  Enforces re-authentication and prevents deletion of last passkey.
  """

  alias MfaService.Passkeys.{PostgresProvider, Credential}

  @reauth_window_seconds 300  # 5 minutes

  @type passkey_info :: %{
          id: String.t(),
          device_name: String.t(),
          created_at: DateTime.t(),
          last_used_at: DateTime.t() | nil,
          backed_up: boolean(),
          transports: [String.t()]
        }

  @doc """
  List all passkeys for a user.
  """
  @spec list_passkeys(String.t()) :: {:ok, [passkey_info()]}
  def list_passkeys(user_id) do
    case PostgresProvider.get_credentials_for_user(user_id) do
      {:ok, credentials} ->
        passkeys =
          Enum.map(credentials, fn cred ->
            %{
              id: cred.id,
              credential_id: Base.url_encode64(cred.credential_id, padding: false),
              device_name: cred.device_name || "Passkey",
              created_at: cred.inserted_at,
              last_used_at: cred.last_used_at,
              backed_up: cred.backed_up,
              transports: cred.transports,
              authenticator_type: determine_type(cred.transports)
            }
          end)

        {:ok, passkeys}

      error ->
        error
    end
  end

  @doc """
  Rename a passkey.
  """
  @spec rename_passkey(String.t(), String.t(), String.t()) ::
          {:ok, passkey_info()} | {:error, term()}
  def rename_passkey(user_id, passkey_id, new_name) do
    with {:ok, credential} <- get_user_credential(user_id, passkey_id),
         changeset <- Credential.rename_changeset(credential, new_name),
         {:ok, updated} <- MfaService.Repo.update(changeset) do
      {:ok,
       %{
         id: updated.id,
         device_name: updated.device_name,
         created_at: updated.inserted_at,
         last_used_at: updated.last_used_at
       }}
    end
  end

  @doc """
  Delete a passkey with re-authentication check.
  """
  @spec delete_passkey(String.t(), String.t(), DateTime.t()) ::
          :ok | {:error, term()}
  def delete_passkey(user_id, passkey_id, last_auth_at) do
    with :ok <- verify_recent_auth(last_auth_at),
         {:ok, _credential} <- get_user_credential(user_id, passkey_id),
         :ok <- verify_not_last_without_alternative(user_id, passkey_id),
         :ok <- PostgresProvider.delete_credential(passkey_id) do
      :ok
    end
  end

  @doc """
  Check if deletion is allowed (for UI feedback).
  """
  @spec can_delete?(String.t(), String.t()) :: {:ok, boolean(), String.t() | nil}
  def can_delete?(user_id, passkey_id) do
    case verify_not_last_without_alternative(user_id, passkey_id) do
      :ok ->
        {:ok, true, nil}

      {:error, :last_passkey_no_alternative} ->
        {:ok, false, "Cannot delete last passkey without alternative authentication method"}

      {:error, reason} ->
        {:ok, false, to_string(reason)}
    end
  end

  @doc """
  Get count of passkeys for a user.
  """
  @spec count_passkeys(String.t()) :: {:ok, non_neg_integer()}
  def count_passkeys(user_id) do
    case PostgresProvider.get_credentials_for_user(user_id) do
      {:ok, credentials} -> {:ok, length(credentials)}
      _ -> {:ok, 0}
    end
  end

  # Private functions

  defp get_user_credential(user_id, passkey_id) do
    case PostgresProvider.get_credential(passkey_id) do
      {:ok, credential} ->
        if credential.user_id == user_id do
          {:ok, credential}
        else
          {:error, :not_found}
        end

      error ->
        error
    end
  end

  defp verify_recent_auth(last_auth_at) do
    seconds_ago = DateTime.diff(DateTime.utc_now(), last_auth_at, :second)

    if seconds_ago <= @reauth_window_seconds do
      :ok
    else
      {:error, :reauth_required}
    end
  end

  defp verify_not_last_without_alternative(user_id, passkey_id) do
    with {:ok, credentials} <- PostgresProvider.get_credentials_for_user(user_id) do
      other_passkeys = Enum.reject(credentials, &(&1.id == passkey_id))

      cond do
        length(other_passkeys) > 0 ->
          # Has other passkeys
          :ok

        has_alternative_method?(user_id) ->
          # Has TOTP or other method
          :ok

        true ->
          {:error, :last_passkey_no_alternative}
      end
    end
  end

  defp has_alternative_method?(_user_id) do
    # Check for TOTP, backup codes, etc.
    # This would query the MFA methods table
    false
  end

  defp determine_type(transports) do
    cond do
      "internal" in transports -> "platform"
      "hybrid" in transports -> "hybrid"
      "usb" in transports or "nfc" in transports or "ble" in transports -> "security_key"
      true -> "unknown"
    end
  end
end
