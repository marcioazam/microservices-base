defmodule MfaService.Passkeys.Credential do
  @moduledoc """
  Schema for WebAuthn passkey credentials with discoverable credential support.
  """
  use Ecto.Schema
  import Ecto.Changeset

  @primary_key {:id, :binary_id, autogenerate: true}
  @foreign_key_type :binary_id

  schema "passkey_credentials" do
    field :user_id, :binary_id
    field :credential_id, :binary
    field :public_key, :binary
    field :public_key_alg, :integer
    field :sign_count, :integer, default: 0
    field :transports, {:array, :string}, default: []
    field :attestation_format, :string
    field :attestation_statement, :map
    field :aaguid, :binary_id
    field :device_name, :string
    field :is_discoverable, :boolean, default: true
    field :backed_up, :boolean, default: false
    field :last_used_at, :utc_datetime_usec

    timestamps(type: :utc_datetime_usec)
  end

  @required_fields ~w(user_id credential_id public_key public_key_alg)a
  @optional_fields ~w(sign_count transports attestation_format attestation_statement aaguid device_name is_discoverable backed_up last_used_at)a

  @doc """
  Changeset for creating a new passkey credential.
  """
  def changeset(credential, attrs) do
    credential
    |> cast(attrs, @required_fields ++ @optional_fields)
    |> validate_required(@required_fields)
    |> validate_number(:public_key_alg, less_than_or_equal_to: 0)
    |> validate_number(:sign_count, greater_than_or_equal_to: 0)
    |> unique_constraint(:credential_id)
  end

  @doc """
  Changeset for updating sign count after authentication.
  """
  def update_sign_count_changeset(credential, new_count) do
    credential
    |> change(%{
      sign_count: new_count,
      last_used_at: DateTime.utc_now()
    })
    |> validate_number(:sign_count, greater_than: credential.sign_count)
  end

  @doc """
  Changeset for renaming a passkey.
  """
  def rename_changeset(credential, device_name) do
    credential
    |> change(%{device_name: device_name})
    |> validate_length(:device_name, max: 255)
  end
end
