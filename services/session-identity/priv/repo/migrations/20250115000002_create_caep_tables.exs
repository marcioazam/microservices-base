defmodule SessionIdentityCore.Repo.Migrations.CreateCaepTables do
  use Ecto.Migration

  def change do
    # CAEP Streams table - stores stream configurations
    create table(:caep_streams, primary_key: false) do
      add :id, :binary_id, primary_key: true
      add :issuer, :string, null: false
      add :audience, :string, null: false
      add :delivery_method, :string, null: false  # "push" or "poll"
      add :endpoint_url, :string
      add :events_requested, {:array, :string}, null: false, default: []
      add :subject_format, :string, null: false, default: "iss_sub"
      add :status, :string, null: false, default: "active"

      # Health metrics
      add :events_delivered, :bigint, null: false, default: 0
      add :events_failed, :bigint, null: false, default: 0
      add :avg_latency_ms, :float, default: 0.0
      add :last_delivery_at, :utc_datetime_usec
      add :last_error, :text
      add :last_error_at, :utc_datetime_usec

      # Authentication
      add :auth_type, :string  # "bearer", "mtls", "none"
      add :auth_credentials_encrypted, :binary

      timestamps(type: :utc_datetime_usec)
    end

    create index(:caep_streams, [:issuer])
    create index(:caep_streams, [:audience])
    create index(:caep_streams, [:status])
    create unique_index(:caep_streams, [:issuer, :audience])

    # CAEP Events audit table - stores emitted events for audit/replay
    create table(:caep_events, primary_key: false) do
      add :id, :binary_id, primary_key: true
      add :stream_id, references(:caep_streams, type: :binary_id, on_delete: :nilify_all)
      add :event_type, :string, null: false
      add :subject_format, :string, null: false
      add :subject_identifier, :string, null: false
      add :jti, :string, null: false  # JWT ID for deduplication
      add :event_timestamp, :utc_datetime_usec, null: false
      add :reason_admin, :text
      add :extra_data, :map, default: %{}

      # Delivery tracking
      add :delivery_status, :string, null: false, default: "pending"
      add :delivery_attempts, :integer, null: false, default: 0
      add :delivered_at, :utc_datetime_usec
      add :last_attempt_at, :utc_datetime_usec
      add :last_error, :text

      timestamps(type: :utc_datetime_usec)
    end

    create index(:caep_events, [:stream_id])
    create index(:caep_events, [:event_type])
    create index(:caep_events, [:subject_identifier])
    create index(:caep_events, [:delivery_status])
    create index(:caep_events, [:event_timestamp])
    create unique_index(:caep_events, [:jti])

    # Partition by month for better performance (PostgreSQL 12+)
    execute """
    CREATE INDEX idx_caep_events_timestamp_brin ON caep_events
    USING BRIN (event_timestamp) WITH (pages_per_range = 128);
    """, """
    DROP INDEX IF EXISTS idx_caep_events_timestamp_brin;
    """
  end
end
