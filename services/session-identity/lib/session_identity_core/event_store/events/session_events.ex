defmodule SessionIdentityCore.EventStore.Events.SessionCreated do
  @moduledoc "Event emitted when a new session is created"
  @derive Jason.Encoder
  defstruct [
    :session_id,
    :user_id,
    :device_id,
    :device_fingerprint,
    :ip_address,
    :user_agent,
    :created_at,
    :expires_at,
    :auth_methods,
    :risk_score
  ]
end

defmodule SessionIdentityCore.EventStore.Events.SessionRefreshed do
  @moduledoc "Event emitted when a session is refreshed/renewed"
  @derive Jason.Encoder
  defstruct [
    :session_id,
    :old_token,
    :new_token,
    :refreshed_at,
    :new_expires_at,
    :reason
  ]
end

defmodule SessionIdentityCore.EventStore.Events.SessionInvalidated do
  @moduledoc "Event emitted when a session is invalidated/terminated"
  @derive Jason.Encoder
  defstruct [
    :session_id,
    :user_id,
    :invalidated_at,
    :reason,
    :initiated_by
  ]
end

defmodule SessionIdentityCore.EventStore.Events.DeviceBound do
  @moduledoc "Event emitted when a device is bound to a session"
  @derive Jason.Encoder
  defstruct [
    :session_id,
    :device_id,
    :device_fingerprint,
    :bound_at,
    :device_info
  ]
end

defmodule SessionIdentityCore.EventStore.Events.MfaVerified do
  @moduledoc "Event emitted when MFA verification occurs"
  @derive Jason.Encoder
  defstruct [
    :session_id,
    :user_id,
    :method,
    :result,
    :device_fingerprint,
    :verified_at
  ]
end

defmodule SessionIdentityCore.EventStore.Events.RiskScoreUpdated do
  @moduledoc "Event emitted when session risk score changes"
  @derive Jason.Encoder
  defstruct [
    :session_id,
    :old_score,
    :new_score,
    :factors,
    :updated_at
  ]
end
