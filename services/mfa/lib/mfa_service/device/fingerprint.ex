defmodule MfaService.Device.Fingerprint do
  @moduledoc """
  Device fingerprinting and change detection.
  Uses SHA-256 for deterministic fingerprint hashing.

  ## Features
  - Deterministic fingerprint computation
  - Similarity calculation based on matching attributes
  - Significant change detection (>30% difference)
  - Graceful handling of missing headers
  """

  @significant_change_threshold 0.3

  @fingerprint_attributes [
    :user_agent,
    :accept_language,
    :accept_encoding,
    :screen_resolution,
    :timezone,
    :platform,
    :plugins,
    :canvas_hash,
    :webgl_hash
  ]

  @type attributes :: map()
  @type fingerprint :: String.t()
  @type comparison_result :: %{
          match: boolean(),
          significant_change: boolean(),
          similarity: float()
        }

  @doc """
  Computes a device fingerprint hash from device attributes.
  Returns a SHA-256 hash as lowercase hex string.
  """
  @spec compute(attributes()) :: fingerprint()
  def compute(attributes) when is_map(attributes) do
    normalized = normalize_attributes(attributes)

    :crypto.hash(:sha256, Jason.encode!(normalized))
    |> Base.encode16(case: :lower)
  end

  @doc """
  Compares two fingerprints or attribute sets and determines similarity.
  Returns match status, significant_change flag, and similarity ratio.
  """
  @spec compare(fingerprint() | attributes(), fingerprint() | attributes()) :: comparison_result()
  def compare(old, new) when is_binary(old) and is_binary(new) and old == new do
    %{match: true, significant_change: false, similarity: 1.0}
  end

  def compare(old_attributes, new_attributes) when is_map(old_attributes) and is_map(new_attributes) do
    similarity = calculate_similarity(old_attributes, new_attributes)
    significant_change = 1.0 - similarity >= @significant_change_threshold

    %{
      match: false,
      significant_change: significant_change,
      similarity: similarity
    }
  end

  def compare(_, _) do
    %{match: false, significant_change: true, similarity: 0.0}
  end

  @doc """
  Determines if a fingerprint change requires re-authentication.
  Returns true if similarity < 70% (significant_change is true).
  """
  @spec requires_reauth?(comparison_result()) :: boolean()
  def requires_reauth?(comparison_result) do
    comparison_result.significant_change
  end

  @doc """
  Extracts device attributes from request headers and metadata.
  Handles missing headers gracefully by using empty strings.
  """
  @spec extract_attributes(map(), map()) :: attributes()
  def extract_attributes(headers, metadata \\ %{}) do
    %{
      user_agent: get_header(headers, "user-agent"),
      accept_language: get_header(headers, "accept-language"),
      accept_encoding: get_header(headers, "accept-encoding"),
      screen_resolution: Map.get(metadata, "screen_resolution", ""),
      timezone: Map.get(metadata, "timezone", ""),
      platform: Map.get(metadata, "platform", ""),
      plugins: Map.get(metadata, "plugins", []),
      canvas_hash: Map.get(metadata, "canvas_hash", ""),
      webgl_hash: Map.get(metadata, "webgl_hash", "")
    }
  end

  @doc """
  Returns the significant change threshold (30%).
  """
  @spec significant_change_threshold() :: float()
  def significant_change_threshold, do: @significant_change_threshold

  defp normalize_attributes(attributes) do
    attributes
    |> Map.take(@fingerprint_attributes)
    |> Enum.sort()
    |> Enum.into(%{})
  end

  defp calculate_similarity(old_attrs, new_attrs) do
    all_keys =
      MapSet.union(
        MapSet.new(Map.keys(old_attrs)),
        MapSet.new(Map.keys(new_attrs))
      )

    if MapSet.size(all_keys) == 0 do
      1.0
    else
      matching_count =
        Enum.count(all_keys, fn key ->
          Map.get(old_attrs, key) == Map.get(new_attrs, key)
        end)

      matching_count / MapSet.size(all_keys)
    end
  end

  defp get_header(headers, key) do
    Map.get(headers, key, Map.get(headers, String.downcase(key), ""))
  end
end
