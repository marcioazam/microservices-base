defmodule MfaService.Passkeys.Provider do
  @moduledoc """
  Generic behaviour for passkey credential providers.
  Allows pluggable storage backends (PostgreSQL, Redis, etc.)
  """

  @type credential_id :: binary()
  @type user_id :: String.t()
  @type credential :: map()
  @type error :: {:error, atom() | String.t()}

  @callback store_credential(user_id(), credential()) :: {:ok, credential()} | error()
  @callback get_credential(credential_id()) :: {:ok, credential()} | {:error, :not_found}
  @callback get_credentials_for_user(user_id()) :: {:ok, [credential()]} | error()
  @callback update_credential(credential_id(), map()) :: {:ok, credential()} | error()
  @callback delete_credential(credential_id()) :: :ok | error()
  @callback increment_sign_count(credential_id(), non_neg_integer()) :: {:ok, credential()} | error()
end

defmodule MfaService.Passkeys.PostgresProvider do
  @moduledoc """
  PostgreSQL implementation of the Passkeys Provider behaviour.
  """
  @behaviour MfaService.Passkeys.Provider

  alias MfaService.Repo
  alias MfaService.Passkeys.Credential
  import Ecto.Query

  @impl true
  def store_credential(user_id, credential_attrs) do
    %Credential{}
    |> Credential.changeset(Map.put(credential_attrs, :user_id, user_id))
    |> Repo.insert()
  end

  @impl true
  def get_credential(credential_id) when is_binary(credential_id) do
    case Repo.get_by(Credential, credential_id: credential_id) do
      nil -> {:error, :not_found}
      cred -> {:ok, cred}
    end
  end

  @impl true
  def get_credentials_for_user(user_id) do
    credentials =
      Credential
      |> where([c], c.user_id == ^user_id)
      |> order_by([c], desc: c.inserted_at)
      |> Repo.all()

    {:ok, credentials}
  end

  @impl true
  def update_credential(credential_id, attrs) when is_binary(credential_id) do
    case get_credential(credential_id) do
      {:ok, cred} ->
        cred
        |> Credential.changeset(attrs)
        |> Repo.update()

      error ->
        error
    end
  end

  @impl true
  def delete_credential(credential_id) when is_binary(credential_id) do
    case get_credential(credential_id) do
      {:ok, cred} ->
        case Repo.delete(cred) do
          {:ok, _} -> :ok
          {:error, _} = error -> error
        end

      error ->
        error
    end
  end

  @impl true
  def increment_sign_count(credential_id, new_count) when is_binary(credential_id) do
    case get_credential(credential_id) do
      {:ok, cred} ->
        cred
        |> Credential.update_sign_count_changeset(new_count)
        |> Repo.update()

      error ->
        error
    end
  end
end
