defmodule SessionIdentityCore.Sessions.SessionSerializer do
  @moduledoc """
  Session JSON serialization with round-trip guarantee.
  Ensures all properties are preserved through serialization/deserialization.
  """

  alias SessionIdentityCore.Sessions.Session

  @doc """
  Serializes a session to JSON format.
  """
  def serialize(%Session{} = session) do
    session
    |> to_serializable_map()
    |> Jason.encode!()
  end

  @doc """
  Deserializes JSON to a session struct.
  """
  def deserialize(json) when is_binary(json) do
    case Jason.decode(json) do
      {:ok, data} -> from_map(data)
      {:error, reason} -> {:error, {:json_decode_error, reason}}
    end
  end

  @doc """
  Converts session to a map suitable for JSON serialization.
  """
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
      "expires_at" => datetime_to_iso8601(session.expires_at),
      "last_activity" => datetime_to_iso8601(session.last_activity),
      "inserted_at" => datetime_to_iso8601(session.inserted_at),
      "updated_at" => datetime_to_iso8601(session.updated_at)
    }
  end

  @doc """
  Reconstructs a session from a map.
  """
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
       expires_at: parse_datetime(data["expires_at"]),
       last_activity: parse_datetime(data["last_activity"]),
       inserted_at: parse_datetime(data["inserted_at"]),
       updated_at: parse_datetime(data["updated_at"])
     }}
  end

  defp datetime_to_iso8601(nil), do: nil
  defp datetime_to_iso8601(%DateTime{} = dt), do: DateTime.to_iso8601(dt)
  defp datetime_to_iso8601(%NaiveDateTime{} = dt) do
    dt |> DateTime.from_naive!("Etc/UTC") |> DateTime.to_iso8601()
  end

  defp parse_datetime(nil), do: nil
  defp parse_datetime(str) when is_binary(str) do
    case DateTime.from_iso8601(str) do
      {:ok, dt, _} -> dt
      _ -> nil
    end
  end
  defp parse_datetime(%DateTime{} = dt), do: dt
end
