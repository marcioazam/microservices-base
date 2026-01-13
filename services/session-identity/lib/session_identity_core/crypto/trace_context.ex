defmodule SessionIdentityCore.Crypto.TraceContext do
  @moduledoc """
  W3C Trace Context propagation for crypto operations.
  
  Extracts trace context from OpenTelemetry and formats it for gRPC metadata.
  """

  @doc """
  Extracts current trace context from OpenTelemetry.
  Returns a keyword list with :traceparent and :tracestate if available.
  """
  @spec extract_current() :: keyword()
  def extract_current do
    opts = []

    opts = case get_traceparent() do
      nil -> opts
      traceparent -> Keyword.put(opts, :traceparent, traceparent)
    end

    opts = case get_tracestate() do
      nil -> opts
      tracestate -> Keyword.put(opts, :tracestate, tracestate)
    end

    opts
  end

  @doc """
  Builds gRPC metadata with trace context.
  """
  @spec build_metadata(keyword()) :: [{String.t(), String.t()}]
  def build_metadata(opts \\ []) do
    trace_opts = extract_current()
    merged_opts = Keyword.merge(trace_opts, opts)

    metadata = []

    metadata = case Keyword.get(merged_opts, :traceparent) do
      nil -> metadata
      traceparent -> [{"traceparent", traceparent} | metadata]
    end

    metadata = case Keyword.get(merged_opts, :tracestate) do
      nil -> metadata
      tracestate -> [{"tracestate", tracestate} | metadata]
    end

    metadata
  end

  @doc """
  Generates a new traceparent header value.
  Format: {version}-{trace-id}-{parent-id}-{trace-flags}
  """
  @spec generate_traceparent() :: String.t()
  def generate_traceparent do
    version = "00"
    trace_id = :crypto.strong_rand_bytes(16) |> Base.encode16(case: :lower)
    parent_id = :crypto.strong_rand_bytes(8) |> Base.encode16(case: :lower)
    trace_flags = "01"  # sampled

    "#{version}-#{trace_id}-#{parent_id}-#{trace_flags}"
  end

  @doc """
  Parses a traceparent header value.
  """
  @spec parse_traceparent(String.t()) :: {:ok, map()} | {:error, :invalid_format}
  def parse_traceparent(traceparent) when is_binary(traceparent) do
    case String.split(traceparent, "-") do
      [version, trace_id, parent_id, trace_flags] 
        when byte_size(version) == 2 
        and byte_size(trace_id) == 32 
        and byte_size(parent_id) == 16 
        and byte_size(trace_flags) == 2 ->
        {:ok, %{
          version: version,
          trace_id: trace_id,
          parent_id: parent_id,
          trace_flags: trace_flags
        }}

      _ ->
        {:error, :invalid_format}
    end
  end

  def parse_traceparent(_), do: {:error, :invalid_format}

  # Private functions

  defp get_traceparent do
    # Try to get from OpenTelemetry context
    try do
      case :otel_tracer.current_span_ctx() do
        :undefined -> nil
        span_ctx -> format_traceparent(span_ctx)
      end
    rescue
      _ -> nil
    end
  end

  defp get_tracestate do
    # Try to get from OpenTelemetry context
    try do
      case :otel_tracer.current_span_ctx() do
        :undefined -> nil
        span_ctx -> format_tracestate(span_ctx)
      end
    rescue
      _ -> nil
    end
  end

  defp format_traceparent(span_ctx) do
    try do
      trace_id = :otel_span.trace_id(span_ctx)
      span_id = :otel_span.span_id(span_ctx)
      trace_flags = :otel_span.trace_flags(span_ctx)

      trace_id_hex = Integer.to_string(trace_id, 16) |> String.pad_leading(32, "0") |> String.downcase()
      span_id_hex = Integer.to_string(span_id, 16) |> String.pad_leading(16, "0") |> String.downcase()
      flags_hex = Integer.to_string(trace_flags, 16) |> String.pad_leading(2, "0")

      "00-#{trace_id_hex}-#{span_id_hex}-#{flags_hex}"
    rescue
      _ -> nil
    end
  end

  defp format_tracestate(_span_ctx) do
    # Tracestate is optional, return nil for now
    nil
  end
end
