defmodule SessionIdentityCore.Telemetry.Metrics do
  @moduledoc """
  Prometheus metrics via Telemetry for Session Identity Core.
  
  Exposes metrics for:
  - Session operations (create, get, delete, refresh)
  - OAuth operations (authorize, token, refresh)
  - PKCE verifications
  - Risk scoring
  - CAEP event emissions
  - Cache operations
  """

  import Telemetry.Metrics

  @doc """
  Returns the list of metrics to be exposed.
  """
  def metrics do
    [
      # Session metrics
      counter("session_identity.session.created.total",
        description: "Total sessions created",
        tags: [:status]
      ),
      counter("session_identity.session.deleted.total",
        description: "Total sessions deleted",
        tags: [:reason]
      ),
      counter("session_identity.session.refreshed.total",
        description: "Total sessions refreshed"
      ),
      distribution("session_identity.session.duration.seconds",
        description: "Session duration in seconds",
        unit: {:native, :second},
        reporter_options: [buckets: [60, 300, 900, 1800, 3600, 7200, 14400, 28800]]
      ),

      # OAuth metrics
      counter("session_identity.oauth.authorize.total",
        description: "Total OAuth authorization requests",
        tags: [:status]
      ),
      counter("session_identity.oauth.token.total",
        description: "Total OAuth token requests",
        tags: [:grant_type, :status]
      ),
      counter("session_identity.oauth.refresh.total",
        description: "Total OAuth refresh token requests",
        tags: [:status]
      ),
      distribution("session_identity.oauth.token.duration.milliseconds",
        description: "Token generation duration",
        unit: {:native, :millisecond},
        reporter_options: [buckets: [5, 10, 25, 50, 100, 250, 500, 1000]]
      ),

      # PKCE metrics
      counter("session_identity.pkce.verification.total",
        description: "Total PKCE verifications",
        tags: [:status]
      ),

      # Risk scoring metrics
      distribution("session_identity.risk.score",
        description: "Risk score distribution",
        reporter_options: [buckets: [0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0]]
      ),
      counter("session_identity.risk.step_up_required.total",
        description: "Total step-up authentications required"
      ),

      # CAEP metrics
      counter("session_identity.caep.event.emitted.total",
        description: "Total CAEP events emitted",
        tags: [:event_type, :status]
      ),

      # Cache metrics
      counter("session_identity.cache.hit.total",
        description: "Total cache hits"
      ),
      counter("session_identity.cache.miss.total",
        description: "Total cache misses"
      ),
      distribution("session_identity.cache.operation.duration.milliseconds",
        description: "Cache operation duration",
        unit: {:native, :millisecond},
        reporter_options: [buckets: [1, 2, 5, 10, 25, 50, 100]]
      ),

      # Event store metrics
      counter("session_identity.events.appended.total",
        description: "Total events appended",
        tags: [:event_type]
      ),
      last_value("session_identity.events.sequence_number",
        description: "Current event sequence number"
      ),

      # Health metrics
      last_value("session_identity.health.status",
        description: "Health check status (1=healthy, 0=unhealthy)"
      )
    ]
  end
end
