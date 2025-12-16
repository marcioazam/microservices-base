defmodule MfaService.Device.Fingerprint do
  @moduledoc """
  Device fingerprinting and change detection.
  """

  @significant_change_threshold 0.3  # 30% difference is significant

  @doc """
  Computes a device fingerprint hash from device attributes.
  """
  def compute(attributes) when is_map(attributes) do
    normalized = normalize_attributes(attributes)
    
    :crypto.hash(:sha256, Jason.encode!(normalized))
    |> Base.encode16(case: :lower)
  end

  @doc """
  Compares two fingerprints and determines if the change is significant.
  """
  def compare(old_fingerprint, new_fingerprint) when old_fingerprint == new_fingerprint do
    %{
      match: true,
      significant_change: false,
      similarity: 1.0
    }
  end

  def compare(old_attributes, new_attributes) when is_map(old_attributes) and is_map(new_attributes) do
    similarity = calculate_similarity(old_attributes, new_attributes)
    significant_change = (1.0 - similarity) >= @significant_change_threshold

    %{
      match: false,
      significant_change: significant_change,
      similarity: similarity
    }
  end

  def compare(_, _) do
    %{
      match: false,
      significant_change: true,
      similarity: 0.0
    }
  end

  @doc """
  Determines if a fingerprint change requires re-authentication.
  """
  def requires_reauth?(comparison_result) do
    comparison_result.significant_change
  end

  @doc """
  Extracts device attributes from request headers and metadata.
  """
  def extract_attributes(headers, metadata \\ %{}) do
    %{
      user_agent: Map.get(headers, "user-agent", ""),
      accept_language: Map.get(headers, "accept-language", ""),
      accept_encoding: Map.get(headers, "accept-encoding", ""),
      screen_resolution: Map.get(metadata, "screen_resolution", ""),
      timezone: Map.get(metadata, "timezone", ""),
      platform: Map.get(metadata, "platform", ""),
      plugins: Map.get(metadata, "plugins", []),
      canvas_hash: Map.get(metadata, "canvas_hash", ""),
      webgl_hash: Map.get(metadata, "webgl_hash", "")
    }
  end

  # Private functions

  defp normalize_attributes(attributes) do
    attributes
    |> Map.take([
      :user_agent, :accept_language, :accept_encoding,
      :screen_resolution, :timezone, :platform,
      :plugins, :canvas_hash, :webgl_hash
    ])
    |> Enum.sort()
    |> Enum.into(%{})
  end

  defp calculate_similarity(old_attrs, new_attrs) do
    all_keys = MapSet.union(
      MapSet.new(Map.keys(old_attrs)),
      MapSet.new(Map.keys(new_attrs))
    )

    if MapSet.size(all_keys) == 0 do
      1.0
    else
      matching_count = Enum.count(all_keys, fn key ->
        Map.get(old_attrs, key) == Map.get(new_attrs, key)
      end)

      matching_count / MapSet.size(all_keys)
    end
  end
end
