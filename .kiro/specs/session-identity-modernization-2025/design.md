# Design Document

## Overview

This design document describes the modernization of the Session Identity Core service to December 2025 state-of-the-art standards. The architecture eliminates redundancy, centralizes logic, integrates with platform services (Cache Service, Logging Service), and ensures OAuth 2.1/RFC 9700 compliance with CAEP/SSF support.

The modernization follows these principles:
- **Zero Redundancy**: Every behavior exists in exactly one authoritative location
- **Extreme Centralization**: Business rules, validations, and transformations centralized
- **Platform Integration**: Use centralized Cache and Logging services
- **Security First**: OAuth 2.1, PKCE S256, 256-bit entropy tokens
- **Testability**: Property-based testing for all correctness properties

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        Session Identity Core (Elixir)                       │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                         API Layer (gRPC)                            │    │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────┐  │    │
│  │  │ Session.Server  │  │  OAuth.Server   │  │  Identity.Server    │  │    │
│  │  └────────┬────────┘  └────────┬────────┘  └──────────┬──────────┘  │    │
│  └───────────┼────────────────────┼──────────────────────┼─────────────┘    │
│              │                    │                      │                  │
│  ┌───────────┴────────────────────┴──────────────────────┴─────────────┐    │
│  │                        Core Domain Layer                            │    │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────┐  │    │
│  │  │ Sessions Module │  │  OAuth Module   │  │  Identity Module    │  │    │
│  │  │ - SessionManager│  │  - OAuth21      │  │  - RiskScorer       │  │    │
│  │  │ - Session       │  │  - PKCE         │  │                     │  │    │
│  │  │ - SessionStore  │  │  - Authorization│  │                     │  │    │
│  │  │ - Serializer    │  │  - IdToken      │  │                     │  │    │
│  │  └────────┬────────┘  └────────┬────────┘  └──────────┬──────────┘  │    │
│  └───────────┼────────────────────┼──────────────────────┼─────────────┘    │
│              │                    │                      │                  │
│  ┌───────────┴────────────────────┴──────────────────────┴─────────────┐    │
│  │                     Infrastructure Layer                            │    │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────┐  │    │
│  │  │  Event Store    │  │  CAEP Emitter   │  │  Shared Utilities   │  │    │
│  │  │  - Aggregate    │  │                 │  │  - Errors           │  │    │
│  │  │  - Event        │  │                 │  │  - Config           │  │    │
│  │  │  - Store        │  │                 │  │  - TTL              │  │    │
│  │  └────────┬────────┘  └────────┬────────┘  └──────────┬──────────┘  │    │
│  └───────────┼────────────────────┼──────────────────────┼─────────────┘    │
│              │                    │                      │                  │
├──────────────┴────────────────────┴──────────────────────┴──────────────────┤
│                        Platform Integration Layer                           │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │              AuthPlatform.Clients (libs/elixir)                     │    │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────┐  │    │
│  │  │  Cache Client   │  │ Logging Client  │  │  Resilience         │  │    │
│  │  │  (Cache Service)│  │(Logging Service)│  │  (Circuit Breaker)  │  │    │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────────┘  │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │              AuthPlatform (libs/elixir)                             │    │
│  │  ┌─────────────────┐  ┌─────────────────┐                           │    │
│  │  │    Security     │  │   Validation    │                           │    │
│  │  │ (Crypto Utils)  │  │ (Input Checks)  │                           │    │
│  │  └─────────────────┘  └─────────────────┘                           │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
            ┌───────────┐   ┌───────────┐   ┌───────────┐
            │  Cache    │   │  Logging  │   │ PostgreSQL│
            │  Service  │   │  Service  │   │           │
            │(platform/)│   │(platform/)│   │           │
            └───────────┘   └───────────┘   └───────────┘
```

## Components and Interfaces

### 1. Session Module (Centralized)

The Session module is the single source of truth for session data structures and operations.

```elixir
defmodule SessionIdentityCore.Sessions.Session do
  @moduledoc """
  Session schema - single source of truth for session structure.
  """
  
  use Ecto.Schema
  import Ecto.Changeset
  alias SessionIdentityCore.Shared.{TTL, Errors}
  alias AuthPlatform.Validation

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

  @type t :: %__MODULE__{}
  
  @spec changeset(t(), map()) :: Ecto.Changeset.t()
  def changeset(session, attrs) do
    session
    |> cast(attrs, [:user_id, :device_id, :ip_address, :user_agent, 
                    :device_fingerprint, :risk_score, :mfa_verified, 
                    :expires_at, :last_activity])
    |> validate_required([:user_id, :ip_address, :device_fingerprint])
    |> validate_number(:risk_score, greater_than_or_equal_to: 0.0, 
                                    less_than_or_equal_to: 1.0)
    |> set_defaults()
  end

  @spec is_expired?(t()) :: boolean()
  def is_expired?(%__MODULE__{expires_at: expires_at}) do
    DateTime.compare(DateTime.utc_now(), expires_at) == :gt
  end
end
```

### 2. Session Serializer (Single Implementation)

Centralized serialization with round-trip guarantee.

```elixir
defmodule SessionIdentityCore.Sessions.SessionSerializer do
  @moduledoc """
  Single source of truth for session serialization.
  Guarantees round-trip: deserialize(serialize(session)) == session
  """
  
  alias SessionIdentityCore.Sessions.Session
  alias SessionIdentityCore.Shared.DateTime, as: DT

  @spec serialize(Session.t()) :: String.t()
  def serialize(%Session{} = session) do
    session |> to_map() |> Jason.encode!()
  end

  @spec deserialize(String.t()) :: {:ok, Session.t()} | {:error, term()}
  def deserialize(json) when is_binary(json) do
    with {:ok, data} <- Jason.decode(json) do
      from_map(data)
    end
  end

  @spec to_map(Session.t()) :: map()
  def to_map(%Session{} = s) do
    %{
      "id" => s.id,
      "user_id" => s.user_id,
      "device_id" => s.device_id,
      "ip_address" => s.ip_address,
      "user_agent" => s.user_agent,
      "device_fingerprint" => s.device_fingerprint,
      "risk_score" => s.risk_score,
      "mfa_verified" => s.mfa_verified,
      "expires_at" => DT.to_iso8601(s.expires_at),
      "last_activity" => DT.to_iso8601(s.last_activity),
      "inserted_at" => DT.to_iso8601(s.inserted_at),
      "updated_at" => DT.to_iso8601(s.updated_at)
    }
  end

  @spec from_map(map()) :: {:ok, Session.t()}
  def from_map(data) when is_map(data) do
    {:ok, %Session{
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
```

### 3. Session Store (Platform Integration)

Uses Cache Service client instead of direct Redix.

```elixir
defmodule SessionIdentityCore.Sessions.SessionStore do
  @moduledoc """
  Session storage using centralized Cache Service.
  """
  
  alias AuthPlatform.Clients.Cache
  alias AuthPlatform.Clients.Logging
  alias SessionIdentityCore.Sessions.SessionSerializer
  alias SessionIdentityCore.Shared.{Keys, TTL}

  @spec store_session(Session.t()) :: {:ok, Session.t()} | {:error, term()}
  def store_session(session) do
    key = Keys.session_key(session.id)
    value = SessionSerializer.serialize(session)
    ttl = TTL.calculate(session.expires_at)

    with :ok <- Cache.set(key, value, ttl: ttl),
         :ok <- add_to_user_sessions(session.user_id, session.id) do
      Logging.info("Session stored", session_id: session.id, user_id: session.user_id)
      {:ok, session}
    else
      {:error, reason} = error ->
        Logging.error("Failed to store session", 
          session_id: session.id, error: inspect(reason))
        error
    end
  end

  @spec get_session(String.t()) :: {:ok, map()} | {:error, :not_found}
  def get_session(session_id) do
    key = Keys.session_key(session_id)
    
    case Cache.get(key) do
      {:ok, nil} -> {:error, :not_found}
      {:ok, value} -> SessionSerializer.deserialize(value)
      error -> error
    end
  end

  @spec delete_session(String.t(), String.t()) :: :ok | {:error, term()}
  def delete_session(session_id, user_id) do
    with :ok <- Cache.delete(Keys.session_key(session_id)),
         :ok <- remove_from_user_sessions(user_id, session_id) do
      Logging.info("Session deleted", session_id: session_id, user_id: user_id)
      :ok
    end
  end
end
```

### 4. Shared Utilities (Centralized)

Single implementations for common operations.

```elixir
defmodule SessionIdentityCore.Shared.Keys do
  @moduledoc "Centralized Redis key generation"
  
  @session_prefix "session:"
  @user_sessions_prefix "user_sessions:"
  @oauth_code_prefix "oauth_code:"
  @events_prefix "events:"
  
  def session_key(id), do: "#{@session_prefix}#{id}"
  def user_sessions_key(user_id), do: "#{@user_sessions_prefix}#{user_id}"
  def oauth_code_key(code), do: "#{@oauth_code_prefix}#{code}"
  def event_key(event_id), do: "#{@events_prefix}#{event_id}"
end

defmodule SessionIdentityCore.Shared.TTL do
  @moduledoc "Centralized TTL calculations"
  
  @default_session_ttl 86_400  # 24 hours
  @default_code_ttl 600        # 10 minutes
  
  def calculate(expires_at) when is_struct(expires_at, DateTime) do
    diff = DateTime.diff(expires_at, DateTime.utc_now())
    max(diff, 1)
  end
  
  def default_session_ttl, do: @default_session_ttl
  def default_code_ttl, do: @default_code_ttl
  
  def default_expiry do
    DateTime.utc_now() |> DateTime.add(@default_session_ttl, :second)
  end
end

defmodule SessionIdentityCore.Shared.DateTime do
  @moduledoc "Centralized datetime operations"
  
  def to_iso8601(nil), do: nil
  def to_iso8601(%DateTime{} = dt), do: DateTime.to_iso8601(dt)
  def to_iso8601(%NaiveDateTime{} = dt) do
    dt |> DateTime.from_naive!("Etc/UTC") |> DateTime.to_iso8601()
  end

  def from_iso8601(nil), do: nil
  def from_iso8601(str) when is_binary(str) do
    case DateTime.from_iso8601(str) do
      {:ok, dt, _} -> dt
      _ -> nil
    end
  end
  def from_iso8601(%DateTime{} = dt), do: dt
end
```

### 5. Centralized Errors Module

```elixir
defmodule SessionIdentityCore.Shared.Errors do
  @moduledoc "Centralized error definitions"
  
  # Session Errors
  def session_not_found, do: {:error, :session_not_found}
  def session_expired, do: {:error, :session_expired}
  def session_invalid, do: {:error, :session_invalid}
  
  # OAuth Errors (RFC 6749 compliant)
  def oauth_error(error, description) do
    {:error, %{error: error, error_description: description}}
  end
  
  def invalid_request(desc), do: oauth_error("invalid_request", desc)
  def invalid_client, do: oauth_error("invalid_client", "Client authentication failed")
  def invalid_grant, do: oauth_error("invalid_grant", "Invalid authorization grant")
  def unsupported_grant_type(type) do
    oauth_error("unsupported_grant_type", "Grant type '#{type}' is not supported")
  end
  def unsupported_response_type(type) do
    oauth_error("unsupported_response_type", "Response type '#{type}' is not supported")
  end
  
  # PKCE Errors
  def pkce_required do
    invalid_request("PKCE is required. code_challenge parameter is missing")
  end
  def pkce_plain_not_allowed do
    invalid_request("code_challenge_method 'plain' is not allowed. Use 'S256'")
  end
  def invalid_code_verifier, do: {:error, :invalid_code_verifier}
  def code_verifier_too_short, do: {:error, :code_verifier_too_short}
  def code_verifier_too_long, do: {:error, :code_verifier_too_long}
  
  # Event Store Errors
  def event_not_found, do: {:error, :event_not_found}
  def aggregate_not_found, do: {:error, :aggregate_not_found}
end
```

### 6. OAuth 2.1 Module (RFC 9700 Compliant)

```elixir
defmodule SessionIdentityCore.OAuth.OAuth21 do
  @moduledoc """
  OAuth 2.1 compliance per RFC 9700 (Security Best Current Practice).
  - Mandatory PKCE for ALL clients
  - S256 only (plain rejected)
  - No implicit grant
  - No ROPC grant
  - Exact redirect_uri matching
  """
  
  alias SessionIdentityCore.OAuth.PKCE
  alias SessionIdentityCore.Shared.Errors
  
  @spec validate_authorization_request(map()) :: {:ok, map()} | {:error, map()}
  def validate_authorization_request(params) do
    with :ok <- validate_response_type(params["response_type"]),
         :ok <- validate_pkce_required(params),
         :ok <- validate_redirect_uri(params["redirect_uri"], params["client_id"]) do
      {:ok, params}
    end
  end

  @spec validate_token_request(map()) :: {:ok, map()} | {:error, map()}
  def validate_token_request(params) do
    with :ok <- validate_grant_type(params["grant_type"]),
         :ok <- validate_pkce_verifier(params) do
      {:ok, params}
    end
  end

  # Reject implicit grant (OAuth 2.1 requirement)
  defp validate_response_type("token"), do: Errors.unsupported_response_type("token")
  defp validate_response_type("code"), do: :ok
  defp validate_response_type(nil), do: Errors.invalid_request("response_type is required")
  defp validate_response_type(type), do: Errors.unsupported_response_type(type)

  # Reject ROPC grant (OAuth 2.1 requirement)
  defp validate_grant_type("password"), do: Errors.unsupported_grant_type("password")
  defp validate_grant_type("authorization_code"), do: :ok
  defp validate_grant_type("refresh_token"), do: :ok
  defp validate_grant_type("client_credentials"), do: :ok
  defp validate_grant_type(nil), do: Errors.invalid_request("grant_type is required")
  defp validate_grant_type(type), do: Errors.unsupported_grant_type(type)

  # PKCE mandatory for ALL clients (OAuth 2.1 requirement)
  defp validate_pkce_required(params) do
    code_challenge = params["code_challenge"]
    method = params["code_challenge_method"]

    cond do
      is_nil(code_challenge) -> Errors.pkce_required()
      method == "plain" -> Errors.pkce_plain_not_allowed()
      method not in [nil, "S256"] -> Errors.invalid_request("Use 'S256' method")
      true -> PKCE.validate_code_challenge(code_challenge)
    end
  end

  defp validate_pkce_verifier(%{"grant_type" => "authorization_code"} = params) do
    case params["code_verifier"] do
      nil -> Errors.invalid_request("code_verifier is required")
      verifier -> PKCE.validate_code_verifier(verifier)
    end
  end
  defp validate_pkce_verifier(_), do: :ok

  # Exact redirect_uri matching (no patterns)
  defp validate_redirect_uri(nil, _), do: Errors.invalid_request("redirect_uri is required")
  defp validate_redirect_uri(uri, client_id) do
    registered = get_registered_uris(client_id)
    if uri in registered, do: :ok, else: Errors.invalid_request("redirect_uri mismatch")
  end

  defp get_registered_uris(_client_id), do: []
end
```

### 7. PKCE Module (S256 Only)

```elixir
defmodule SessionIdentityCore.OAuth.PKCE do
  @moduledoc """
  PKCE implementation - S256 method only per OAuth 2.1.
  Uses constant-time comparison from AuthPlatform.Security.
  """
  
  alias AuthPlatform.Security
  alias SessionIdentityCore.Shared.Errors

  @verifier_min_length 43
  @verifier_max_length 128
  @challenge_length 43
  @valid_chars ~r/^[A-Za-z0-9\-._~]+$/
  @valid_base64url ~r/^[A-Za-z0-9_-]+$/

  @spec verify(String.t(), String.t(), String.t()) :: :ok | {:error, atom()}
  def verify(code_verifier, code_challenge, "S256") do
    computed = compute_s256_challenge(code_verifier)
    if Security.constant_time_compare(computed, code_challenge) do
      :ok
    else
      Errors.invalid_code_verifier()
    end
  end
  def verify(_, _, "plain"), do: {:error, :plain_method_not_allowed}
  def verify(_, _, _), do: {:error, :unsupported_method}

  @spec compute_s256_challenge(String.t()) :: String.t()
  def compute_s256_challenge(code_verifier) do
    :crypto.hash(:sha256, code_verifier)
    |> Base.url_encode64(padding: false)
  end

  @spec validate_code_verifier(String.t()) :: :ok | {:error, atom()}
  def validate_code_verifier(v) when is_binary(v) do
    len = String.length(v)
    cond do
      len < @verifier_min_length -> Errors.code_verifier_too_short()
      len > @verifier_max_length -> Errors.code_verifier_too_long()
      not Regex.match?(@valid_chars, v) -> {:error, :invalid_characters}
      true -> :ok
    end
  end
  def validate_code_verifier(_), do: {:error, :invalid_code_verifier}

  @spec validate_code_challenge(String.t()) :: :ok | {:error, atom()}
  def validate_code_challenge(c) when is_binary(c) do
    if String.length(c) == @challenge_length and Regex.match?(@valid_base64url, c) do
      :ok
    else
      {:error, :invalid_code_challenge}
    end
  end
  def validate_code_challenge(_), do: {:error, :invalid_code_challenge}
end
```

### 8. Risk Scorer (Adaptive Authentication)

```elixir
defmodule SessionIdentityCore.Identity.RiskScorer do
  @moduledoc """
  Risk-based adaptive authentication.
  Score range: [0.0, 1.0]
  Threshold >= 0.7: step-up required
  Threshold >= 0.9: WebAuthn/TOTP required
  """
  
  @step_up_threshold 0.7
  @high_risk_threshold 0.9

  @weights %{
    ip_risk: 0.2,
    device_risk: 0.3,
    behavior_risk: 0.25,
    time_risk: 0.1,
    location_risk: 0.15
  }

  @spec calculate_risk(map(), map()) :: float()
  def calculate_risk(session, context \\ %{}) do
    factors = %{
      ip_risk: calculate_ip_risk(session.ip_address, context),
      device_risk: calculate_device_risk(session.device_fingerprint, context),
      behavior_risk: calculate_behavior_risk(session.user_id, context),
      time_risk: calculate_time_risk(context),
      location_risk: Map.get(context, :location_risk, 0.0)
    }

    score = Enum.reduce(factors, 0.0, fn {key, value}, acc ->
      acc + value * Map.get(@weights, key, 0.0)
    end)

    clamp(score, 0.0, 1.0)
  end

  @spec requires_step_up?(float()) :: boolean()
  def requires_step_up?(score) when score >= @step_up_threshold, do: true
  def requires_step_up?(_), do: false

  @spec get_required_factors(float()) :: [atom()]
  def get_required_factors(score) do
    cond do
      score >= @high_risk_threshold -> [:webauthn, :totp]
      score >= @step_up_threshold -> [:totp]
      score >= 0.5 -> [:email_verification]
      true -> []
    end
  end

  defp calculate_ip_risk(ip, context) do
    known_ips = Map.get(context, :known_ips, [])
    cond do
      ip in known_ips -> 0.0
      is_vpn_or_proxy?(ip) -> 0.6
      is_tor_exit_node?(ip) -> 0.9
      true -> 0.3
    end
  end

  defp calculate_device_risk(fingerprint, context) do
    known = Map.get(context, :known_devices, [])
    if fingerprint in known, do: 0.0, else: 0.5
  end

  defp calculate_behavior_risk(_user_id, context) do
    attempts = Map.get(context, :recent_failed_attempts, 0)
    cond do
      attempts >= 5 -> 0.9
      attempts >= 3 -> 0.6
      attempts >= 1 -> 0.3
      true -> 0.0
    end
  end

  defp calculate_time_risk(context) do
    hour = Map.get(context, :hour, DateTime.utc_now().hour)
    if hour >= 0 and hour < 5, do: 0.4, else: 0.0
  end

  defp clamp(value, min, max), do: min(max(value, min), max)
  defp is_vpn_or_proxy?(_ip), do: false
  defp is_tor_exit_node?(_ip), do: false
end
```

## Data Models

### Session Schema

```elixir
schema "sessions" do
  field :id, :binary_id, primary_key: true
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
```

### Event Schema

```elixir
defstruct [
  :event_id,           # UUID
  :event_type,         # String (SessionCreated, etc.)
  :aggregate_id,       # UUID (session_id)
  :aggregate_type,     # "Session"
  :sequence_number,    # Monotonically increasing integer
  :timestamp,          # DateTime (UTC)
  :schema_version,     # Integer for migrations
  :correlation_id,     # UUID for tracing
  :causation_id,       # UUID (optional)
  :payload,            # Map
  :metadata            # Map
]
```

### OAuth Authorization Code

```elixir
defstruct [
  :code,                    # 32-byte random, base64url encoded
  :client_id,               # String
  :redirect_uri,            # String (exact match)
  :user_id,                 # UUID
  :scopes,                  # List of strings
  :code_challenge,          # S256 challenge
  :code_challenge_method,   # "S256" only
  :nonce,                   # Optional OIDC nonce
  :created_at,              # DateTime
  :expires_at               # DateTime (10 min TTL)
]
```



## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

Based on the prework analysis, the following correctness properties have been identified and consolidated to eliminate redundancy:

### Property 1: Session Serialization Round-Trip

*For any* valid Session struct, serializing to JSON and then deserializing back SHALL produce an equivalent Session struct with all fields preserved.

**Validates: Requirements 2.5**

This is a round-trip property ensuring data integrity through serialization. The serializer is the single source of truth for session data conversion.

### Property 2: PKCE Verification Correctness

*For any* valid code_verifier (43-128 characters, valid charset), computing S256(code_verifier) and then verifying against the computed challenge SHALL succeed.

*For any* two different code_verifiers, verifying one against the challenge computed from the other SHALL fail.

*For any* string shorter than 43 characters or longer than 128 characters, validation SHALL reject it.

**Validates: Requirements 3.3, 3.7, 3.8**

This property ensures PKCE implementation correctness per OAuth 2.1/RFC 9700.

### Property 3: Redirect URI Exact Matching

*For any* redirect_uri and registered_uris list, the redirect_uri SHALL be accepted if and only if it exactly matches one of the registered URIs (no pattern matching, no substring matching).

**Validates: Requirements 3.5**

This property ensures OAuth 2.1 security requirement for exact redirect_uri matching.

### Property 4: Refresh Token Rotation

*For any* refresh token exchange, the returned new refresh token SHALL be different from the original, and the original token SHALL be invalidated.

**Validates: Requirements 3.6**

This property ensures refresh token rotation per OAuth 2.1 security best practices.

### Property 5: Session Token Entropy

*For any* generated session token, it SHALL have at least 256 bits (32 bytes) of entropy from crypto.strong_rand_bytes.

**Validates: Requirements 4.1**

This property ensures session tokens have sufficient entropy to prevent guessing attacks.

### Property 6: Session Device Binding

*For any* created session, it SHALL contain non-null device_fingerprint and ip_address fields that match the creation request.

**Validates: Requirements 4.3**

This property ensures sessions are bound to device context for security.

### Property 7: Session Events Correlation

*For any* session lifecycle operation (create, terminate), the emitted event SHALL contain a non-null correlation_id, and termination events SHALL contain a reason field.

**Validates: Requirements 4.4, 4.5**

This property ensures audit trail completeness for compliance.

### Property 8: Session Key Namespacing

*For any* session stored in cache, the key SHALL start with "session:" prefix.

**Validates: Requirements 4.7**

This property ensures key isolation in shared cache infrastructure.

### Property 9: Risk Score Bounds and Thresholds

*For any* session and context, the calculated risk score SHALL be in the range [0.0, 1.0].

*For any* risk score >= 0.7, requires_step_up? SHALL return true.

*For any* risk score >= 0.9, get_required_factors SHALL include :webauthn or :totp.

**Validates: Requirements 5.1, 5.2, 5.3**

This property ensures risk scoring produces valid scores and correct threshold actions.

### Property 10: Risk Factors Affect Score

*For any* session with a known device (device_fingerprint in known_devices), the risk score SHALL be lower than or equal to the score for the same session with an unknown device.

*For any* session with failed_attempts >= 5, the behavior_risk component SHALL be 0.9.

**Validates: Requirements 5.4, 5.5, 5.6**

This property ensures risk factors correctly influence the final score.

### Property 11: ID Token Claims Completeness

*For any* generated ID token, it SHALL contain all required claims: sub, iss, aud, exp, iat.

*For any* ID token where nonce was provided in the request, the token SHALL include the nonce claim with the same value.

*For any* ID token, exp SHALL equal iat + ttl.

**Validates: Requirements 6.1, 6.2, 6.3, 6.4, 6.5**

This property ensures OIDC compliance for ID tokens.

### Property 12: Event Structure Correctness

*For any* sequence of appended events, the sequence numbers SHALL be strictly monotonically increasing.

*For any* event, it SHALL contain a non-null correlation_id.

*For any* serialized event, the timestamp SHALL be in ISO 8601 UTC format.

**Validates: Requirements 7.1, 7.3, 7.5**

This property ensures event store integrity and traceability.

### Property 13: Event Replay Consistency

*For any* aggregate, loading by replaying all events SHALL produce the same state as loading from a snapshot plus subsequent events.

**Validates: Requirements 7.2, 7.6**

This is a round-trip property ensuring event sourcing correctness.

### Property 14: CAEP Event Format

*For any* emitted CAEP event, it SHALL contain: event_type, subject (with format, iss, sub), event_timestamp, and reason_admin fields per SSF specification.

**Validates: Requirements 8.4**

This property ensures CAEP/SSF compliance for continuous access evaluation.

## Error Handling

### Error Categories

1. **Session Errors**: session_not_found, session_expired, session_invalid
2. **OAuth Errors**: RFC 6749 compliant (invalid_request, invalid_client, invalid_grant, unsupported_grant_type, unsupported_response_type)
3. **PKCE Errors**: pkce_required, pkce_plain_not_allowed, invalid_code_verifier, code_verifier_too_short, code_verifier_too_long
4. **Event Store Errors**: event_not_found, aggregate_not_found

### Error Response Format

```elixir
# OAuth errors (RFC 6749)
{:error, %{
  error: "invalid_request",
  error_description: "Human-readable description"
}}

# Internal errors
{:error, :session_not_found}
{:error, :invalid_code_verifier}
```

### Error Handling Strategy

1. Use Result pattern consistently: `{:ok, value} | {:error, reason}`
2. Never expose internal error details to external clients
3. Log all errors with correlation_id for traceability
4. Use centralized Errors module for all error definitions

## Testing Strategy

### Dual Testing Approach

The testing strategy combines unit tests and property-based tests for comprehensive coverage:

1. **Unit Tests**: Verify specific examples, edge cases, and error conditions
2. **Property Tests**: Verify universal properties across all valid inputs using StreamData

### Property-Based Testing Configuration

- **Library**: StreamData (Elixir)
- **Minimum Iterations**: 100 per property test
- **Tag Format**: `**Feature: session-identity-modernization-2025, Property N: [property_text]**`

### Test Organization

```
test/
├── session_identity_core/
│   ├── sessions/
│   │   ├── session_test.exs           # Unit tests
│   │   ├── session_property_test.exs  # Property tests
│   │   ├── session_store_test.exs     # Integration tests
│   │   └── serializer_property_test.exs
│   ├── oauth/
│   │   ├── oauth21_test.exs
│   │   ├── pkce_test.exs
│   │   ├── pkce_property_test.exs
│   │   └── id_token_property_test.exs
│   ├── identity/
│   │   ├── risk_scorer_test.exs
│   │   └── risk_scorer_property_test.exs
│   ├── event_store/
│   │   ├── event_test.exs
│   │   ├── store_test.exs
│   │   └── aggregate_property_test.exs
│   └── caep/
│       └── emitter_test.exs
└── support/
    ├── generators.ex                   # Shared StreamData generators
    └── test_helpers.ex
```

### Coverage Requirements

- Minimum 80% code coverage on core modules
- 100% coverage on security-critical paths (PKCE, session management)
- All 14 correctness properties implemented as property tests
