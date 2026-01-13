defmodule MfaService.Device.FingerprintTest do
  @moduledoc """
  Unit tests for Device Fingerprint module.
  """

  use ExUnit.Case, async: true

  alias MfaService.Device.Fingerprint

  @moduletag :unit

  describe "compute/1" do
    test "returns consistent SHA-256 hash" do
      attrs = %{user_agent: "Test/1.0", timezone: "UTC"}

      hash1 = Fingerprint.compute(attrs)
      hash2 = Fingerprint.compute(attrs)

      assert hash1 == hash2
      assert String.length(hash1) == 64  # SHA256 hex
      assert String.match?(hash1, ~r/^[a-f0-9]{64}$/)
    end

    test "different attributes produce different hashes" do
      attrs1 = %{user_agent: "Browser/1.0", timezone: "UTC"}
      attrs2 = %{user_agent: "Browser/2.0", timezone: "UTC"}

      assert Fingerprint.compute(attrs1) != Fingerprint.compute(attrs2)
    end

    test "normalizes attributes before hashing" do
      # Same attributes in different order should produce same hash
      attrs1 = %{user_agent: "Test", timezone: "UTC", platform: "Win"}
      attrs2 = %{platform: "Win", user_agent: "Test", timezone: "UTC"}

      assert Fingerprint.compute(attrs1) == Fingerprint.compute(attrs2)
    end
  end

  describe "compare/2" do
    test "identical hash strings return match=true" do
      attrs = %{user_agent: "Test", timezone: "UTC"}
      hash = Fingerprint.compute(attrs)

      result = Fingerprint.compare(hash, hash)

      assert result.match == true
      assert result.similarity == 1.0
      assert result.significant_change == false
    end

    test "identical attributes return similarity=1.0" do
      attrs = %{user_agent: "Test", timezone: "UTC"}

      result = Fingerprint.compare(attrs, attrs)

      assert result.similarity == 1.0
      assert result.significant_change == false
    end

    test "completely different attributes return low similarity" do
      old = %{user_agent: "Old", timezone: "UTC", platform: "Win"}
      new = %{user_agent: "New", timezone: "PST", platform: "Mac"}

      result = Fingerprint.compare(old, new)

      assert result.similarity == 0.0
      assert result.significant_change == true
    end

    test "partial match calculates correct similarity" do
      old = %{user_agent: "Same", timezone: "UTC", platform: "Win", lang: "en"}
      new = %{user_agent: "Same", timezone: "UTC", platform: "Mac", lang: "fr"}

      result = Fingerprint.compare(old, new)

      # 2 out of 4 match = 50%
      assert result.similarity == 0.5
      assert result.significant_change == true
    end
  end

  describe "requires_reauth?/1" do
    test "returns true when significant_change is true" do
      result = %{match: false, significant_change: true, similarity: 0.5}
      assert Fingerprint.requires_reauth?(result) == true
    end

    test "returns false when significant_change is false" do
      result = %{match: false, significant_change: false, similarity: 0.8}
      assert Fingerprint.requires_reauth?(result) == false
    end
  end

  describe "extract_attributes/2" do
    test "extracts headers correctly" do
      headers = %{
        "user-agent" => "Mozilla/5.0",
        "accept-language" => "en-US"
      }

      attrs = Fingerprint.extract_attributes(headers, %{})

      assert attrs.user_agent == "Mozilla/5.0"
      assert attrs.accept_language == "en-US"
    end

    test "handles missing headers with empty defaults" do
      attrs = Fingerprint.extract_attributes(%{}, %{})

      assert attrs.user_agent == ""
      assert attrs.accept_language == ""
      assert attrs.timezone == ""
      assert attrs.platform == ""
    end

    test "extracts metadata correctly" do
      headers = %{}
      metadata = %{
        "screen_resolution" => "1920x1080",
        "timezone" => "America/New_York"
      }

      attrs = Fingerprint.extract_attributes(headers, metadata)

      assert attrs.screen_resolution == "1920x1080"
      assert attrs.timezone == "America/New_York"
    end
  end

  describe "significant_change_threshold/0" do
    test "returns 0.3 (30%)" do
      assert Fingerprint.significant_change_threshold() == 0.3
    end
  end
end
