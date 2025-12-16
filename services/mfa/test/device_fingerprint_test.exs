defmodule MfaService.Device.FingerprintTest do
  use ExUnit.Case, async: true
  use ExUnitProperties

  alias MfaService.Device.Fingerprint

  describe "Device Fingerprint Change Detection" do
    # **Feature: auth-microservices-platform, Property 20: Device Fingerprint Change Detection**
    # **Validates: Requirements 6.6**
    property "detects significant changes in device fingerprint" do
      check all user_agent <- StreamData.string(:alphanumeric, min_length: 10, max_length: 100),
                timezone <- StreamData.member_of(["UTC", "America/New_York", "Europe/London", "Asia/Tokyo"]),
                platform <- StreamData.member_of(["Windows", "MacOS", "Linux", "iOS", "Android"]),
                max_runs: 100 do
        
        old_attrs = %{
          user_agent: user_agent,
          timezone: timezone,
          platform: platform,
          screen_resolution: "1920x1080",
          accept_language: "en-US"
        }

        # Completely different device
        new_attrs = %{
          user_agent: "Different/Browser",
          timezone: "Pacific/Auckland",
          platform: "Unknown",
          screen_resolution: "800x600",
          accept_language: "ja-JP"
        }

        result = Fingerprint.compare(old_attrs, new_attrs)

        # Should detect significant change
        assert result.significant_change == true
        assert result.similarity < 0.7
        assert Fingerprint.requires_reauth?(result) == true
      end
    end

    property "identical fingerprints match exactly" do
      check all user_agent <- StreamData.string(:alphanumeric, min_length: 10, max_length: 100),
                timezone <- StreamData.string(:alphanumeric, min_length: 3, max_length: 30),
                max_runs: 100 do
        
        attrs = %{
          user_agent: user_agent,
          timezone: timezone,
          platform: "Windows",
          screen_resolution: "1920x1080"
        }

        fingerprint1 = Fingerprint.compute(attrs)
        fingerprint2 = Fingerprint.compute(attrs)

        assert fingerprint1 == fingerprint2

        result = Fingerprint.compare(fingerprint1, fingerprint2)
        assert result.match == true
        assert result.similarity == 1.0
        assert result.significant_change == false
      end
    end

    property "minor changes do not trigger re-authentication" do
      check all user_agent <- StreamData.string(:alphanumeric, min_length: 10, max_length: 100),
                max_runs: 100 do
        
        old_attrs = %{
          user_agent: user_agent,
          timezone: "UTC",
          platform: "Windows",
          screen_resolution: "1920x1080",
          accept_language: "en-US"
        }

        # Only one attribute changed
        new_attrs = %{old_attrs | accept_language: "en-GB"}

        result = Fingerprint.compare(old_attrs, new_attrs)

        # Should not be significant (only 1 of 5 attributes changed = 80% similarity)
        assert result.similarity >= 0.7
        assert result.significant_change == false
        assert Fingerprint.requires_reauth?(result) == false
      end
    end
  end

  describe "Unit tests" do
    test "compute returns consistent hash" do
      attrs = %{user_agent: "Test/1.0", timezone: "UTC"}
      
      hash1 = Fingerprint.compute(attrs)
      hash2 = Fingerprint.compute(attrs)

      assert hash1 == hash2
      assert String.length(hash1) == 64  # SHA256 hex
    end

    test "extract_attributes handles missing headers" do
      attrs = Fingerprint.extract_attributes(%{})

      assert attrs.user_agent == ""
      assert attrs.timezone == ""
    end
  end
end
