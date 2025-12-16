defmodule SessionIdentityCore.Sessions.Session do
  @moduledoc """
  Session schema and functions for managing user sessions.
  """

  use Ecto.Schema
  import Ecto.Changeset

  @primary_key {:id, :binary_id, autogenerate: true}
  @foreign_key_type :binary_id

  schema "sessions" do
    field :user_id, :binary_id
    field :device_id, :binary_id
    field :ip_address, :string
    field :user_agent, :string
    field :device_fingerprint, :string
    field :risk_score, :float, default: 0.0
    field :mfa_verified, :boolean, default: false
    field :expires_at, :utc_datetime
    field :last_activity, :utc_datetime

    timestamps(type: :utc_datetime)
  end

  @required_fields [:user_id, :ip_address, :device_fingerprint]
  @optional_fields [:device_id, :user_agent, :risk_score, :mfa_verified, :expires_at, :last_activity]

  def changeset(session, attrs) do
    session
    |> cast(attrs, @required_fields ++ @optional_fields)
    |> validate_required(@required_fields)
    |> validate_number(:risk_score, greater_than_or_equal_to: 0.0, less_than_or_equal_to: 1.0)
    |> set_defaults()
  end

  defp set_defaults(changeset) do
    changeset
    |> put_change_if_nil(:expires_at, default_expiry())
    |> put_change_if_nil(:last_activity, DateTime.utc_now())
  end

  defp put_change_if_nil(changeset, field, value) do
    if get_field(changeset, field) do
      changeset
    else
      put_change(changeset, field, value)
    end
  end

  defp default_expiry do
    DateTime.utc_now() |> DateTime.add(24 * 60 * 60, :second)
  end

  def is_expired?(%__MODULE__{expires_at: expires_at}) do
    DateTime.compare(DateTime.utc_now(), expires_at) == :gt
  end

  def to_map(%__MODULE__{} = session) do
    %{
      session_id: session.id,
      user_id: session.user_id,
      device_id: session.device_id,
      ip_address: session.ip_address,
      user_agent: session.user_agent,
      device_fingerprint: session.device_fingerprint,
      risk_score: session.risk_score,
      mfa_verified: session.mfa_verified,
      created_at: DateTime.to_unix(session.inserted_at),
      last_activity: DateTime.to_unix(session.last_activity),
      expires_at: DateTime.to_unix(session.expires_at)
    }
  end
end
