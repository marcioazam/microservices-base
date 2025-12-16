defmodule MfaService.Repo.Migrations.CreatePasskeyCredentials do
  use Ecto.Migration

  def change do
    create table(:passkey_credentials, primary_key: false) do
      add :id, :binary_id, primary_key: true
      add :user_id, :binary_id, null: false
      add :credential_id, :binary, null: false
      add :public_key, :binary, null: false
      add :public_key_alg, :integer, null: false
      add :sign_count, :bigint, null: false, default: 0
      add :transports, {:array, :string}, default: []
      add :attestation_format, :string
      add :attestation_statement, :map
      add :aaguid, :binary_id
      add :device_name, :string
      add :is_discoverable, :boolean, null: false, default: true
      add :backed_up, :boolean, null: false, default: false
      add :last_used_at, :utc_datetime_usec

      timestamps(type: :utc_datetime_usec)
    end

    create unique_index(:passkey_credentials, [:credential_id])
    create index(:passkey_credentials, [:user_id])
    create index(:passkey_credentials, [:user_id, :is_discoverable])
  end
end
