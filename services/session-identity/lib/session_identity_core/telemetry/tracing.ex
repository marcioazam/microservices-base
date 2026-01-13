defmodule SessionIdentityCore.Telemetry.Tracing do
  @moduledoc """
  OpenTelemetry tracing with W3C Trace Context support.
  
  Provides distributed tracing for Session Identity Core operations
  with automatic context propagation.
  """

  require OpenTelemetry.Tracer, as: Tracer

  @doc """
  Starts a new span for the given operation.
  
  ## Options
  - `:attributes` - Map of span attributes
  - `:kind` - Span kind (:internal, :server, :client, :producer, :consumer)
  """
  def with_span(name, opts \\ [], fun) when is_function(fun, 0) do
    attributes = Keyword.get(opts, :attributes, %{})
    kind = Keyword.get(opts, :kind, :internal)

    Tracer.with_span name, %{kind: kind} do
      set_attributes(attributes)

      try do
        result = fun.()
        set_status(:ok)
        result
      rescue
        e ->
          set_status(:error, Exception.message(e))
          set_attributes(%{"exception.type" => inspect(e.__struct__)})
          reraise e, __STACKTRACE__
      end
    end
  end

  @doc """
  Sets attributes on the current span.
  """
  def set_attributes(attributes) when is_map(attributes) do
    Enum.each(attributes, fn {key, value} ->
      Tracer.set_attribute(to_string(key), value)
    end)
  end

  @doc """
  Sets the span status.
  """
  def set_status(:ok) do
    Tracer.set_status(:ok, "")
  end

  def set_status(:error, message \\ "Error") do
    Tracer.set_status(:error, message)
  end

  @doc """
  Adds an event to the current span.
  """
  def add_event(name, attributes \\ %{}) do
    Tracer.add_event(name, attributes)
  end

  @doc """
  Extracts trace context from headers (W3C Trace Context).
  """
  def extract_context(headers) when is_list(headers) do
    :otel_propagator_text_map.extract(headers)
  end

  def extract_context(headers) when is_map(headers) do
    headers
    |> Enum.map(fn {k, v} -> {to_string(k), v} end)
    |> extract_context()
  end

  @doc """
  Injects trace context into headers (W3C Trace Context).
  """
  def inject_context(headers \\ []) do
    :otel_propagator_text_map.inject(headers)
  end

  @doc """
  Gets the current trace ID as a hex string.
  """
  def current_trace_id do
    case Tracer.current_span_ctx() do
      :undefined -> nil
      ctx -> OpenTelemetry.Span.trace_id(ctx) |> format_id()
    end
  end

  @doc """
  Gets the current span ID as a hex string.
  """
  def current_span_id do
    case Tracer.current_span_ctx() do
      :undefined -> nil
      ctx -> OpenTelemetry.Span.span_id(ctx) |> format_id()
    end
  end

  # Session operation spans

  @doc "Trace session creation"
  def trace_session_create(session_id, user_id, fun) do
    with_span "session.create",
      kind: :internal,
      attributes: %{
        "session.id" => session_id,
        "user.id" => user_id,
        "operation" => "create"
      } do
      fun.()
    end
  end

  @doc "Trace session retrieval"
  def trace_session_get(session_id, fun) do
    with_span "session.get",
      kind: :internal,
      attributes: %{
        "session.id" => session_id,
        "operation" => "get"
      } do
      fun.()
    end
  end

  @doc "Trace session deletion"
  def trace_session_delete(session_id, reason, fun) do
    with_span "session.delete",
      kind: :internal,
      attributes: %{
        "session.id" => session_id,
        "deletion.reason" => to_string(reason),
        "operation" => "delete"
      } do
      fun.()
    end
  end

  # OAuth operation spans

  @doc "Trace OAuth authorization"
  def trace_oauth_authorize(client_id, fun) do
    with_span "oauth.authorize",
      kind: :server,
      attributes: %{
        "oauth.client_id" => client_id,
        "oauth.flow" => "authorization_code"
      } do
      fun.()
    end
  end

  @doc "Trace OAuth token exchange"
  def trace_oauth_token(grant_type, client_id, fun) do
    with_span "oauth.token",
      kind: :server,
      attributes: %{
        "oauth.grant_type" => grant_type,
        "oauth.client_id" => client_id
      } do
      fun.()
    end
  end

  # PKCE operation spans

  @doc "Trace PKCE verification"
  def trace_pkce_verify(fun) do
    with_span "pkce.verify", kind: :internal do
      fun.()
    end
  end

  # Risk scoring spans

  @doc "Trace risk score calculation"
  def trace_risk_score(user_id, fun) do
    with_span "risk.calculate",
      kind: :internal,
      attributes: %{"user.id" => user_id} do
      fun.()
    end
  end

  # Cache operation spans

  @doc "Trace cache operation"
  def trace_cache_operation(operation, key, fun) do
    with_span "cache.#{operation}",
      kind: :client,
      attributes: %{
        "cache.operation" => to_string(operation),
        "cache.key_prefix" => extract_key_prefix(key)
      } do
      fun.()
    end
  end

  # Event store spans

  @doc "Trace event append"
  def trace_event_append(event_type, aggregate_id, fun) do
    with_span "events.append",
      kind: :internal,
      attributes: %{
        "event.type" => event_type,
        "aggregate.id" => aggregate_id
      } do
      fun.()
    end
  end

  # Private helpers

  defp format_id(id) when is_integer(id) do
    Integer.to_string(id, 16) |> String.downcase()
  end

  defp format_id(_), do: nil

  defp extract_key_prefix(key) when is_binary(key) do
    case String.split(key, ":", parts: 2) do
      [prefix, _] -> prefix
      _ -> "unknown"
    end
  end

  defp extract_key_prefix(_), do: "unknown"
end
