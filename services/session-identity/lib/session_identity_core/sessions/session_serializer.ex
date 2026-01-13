defmodule SessionIdentityCore.Sessions.SessionSerializer do
  @moduledoc """
  Session JSON serialization with round-trip guarantee.
  
  Ensures all properties are preserved through serialization/deserialization.
  Uses centralized Shared.DateTime for consistent datetime handling.
  
  ## Round-Trip Guarantee
  
  For any valid Session struct:
  `deserialize(serialize(session)) == {:ok, session}`
  """

  alias SessionIdentityCore.Sessions.Session
  alias SessionIdentityCore.Shared.DateTime, as: DT

  @doc """
  Serializes a session to JSON format.
  """
  @spec serialize(Session.t()) :: String.t()
  def serialize(%Session{} = session) do
    session
    |> to_serializable_map()
    |> Jason.encode!()
  end

  @doc """
  Deserializes JSON to a session struct.
  """
  @spec deserialize(String.t()) :: {:ok, Session.t()} | {:error, term()}
  def deserialize(json) when is_binary(json) do
    case Jason.decode(json) do
      {:ok, data} -> from_map(data)
      {:error, reason} -> {:error, {:json_decode_error, reason}}
    end
  end

  @doc """
  Converts session to a map suitable for JSON serialization.
  """
  @spec to_serializable_map(Session.t()) :: map()
  def to_serializable_map(%Session{} = session) do
    %{
      "id" => session.id,
      "user_id" => session.user_id,
      "device_id" => session.device_id,
      "ip_address" => session.ip_address,
      "user_agent" => session.user_agent,
      "device_fingerprint" => session.device_fingerprint,
      "risk_score" => session.risk_score,
      "mfa_verified" => session.mfa_verified,
      "expires_at" => DT.to_iso8601(session.expires_at),
      "last_activity" => DT.to_iso8601(session.last_activity),
      "inserted_at" => DT.to_iso8601(session.inserted_at),
      "updated_at" => DT.to_iso8601(session.updated_at)
    }
  end

  @doc """
  Reconstructs a session from a map.
  """
  @spec from_map(map()) :: {:ok, Session.t()}
  def from_map(data) when is_map(data) do
    {:ok,
     %Session{
       id: data["id"],
       user_id: data["user_id"],
       device_id: data["device_id"],
       ip_address: data["ip_address"],
       user_agent: data["user_agent"],
       device_fingerprint: data["device_fingerprint"],
       risk_score: data["risk_score"] || 0.0,
       mfa_verified: data["mfa_verified"] || false,
       expires_at: DT.from_iso8601(data["expires_at"]),
       last_activity: DT.from_iso8601(data["last_activity"]),
       inserted_at: DT.from_iso8601(data["inserted_at"]),
       updated_at: DT.from_iso8601(data["updated_at"])
     }}
  end
end
