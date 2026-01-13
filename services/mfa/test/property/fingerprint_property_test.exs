defmodule MfaService.Property.FingerprintPropertyTest do
  @moduledoc """
  Property-based tests for Device Fingerprint module.
  Validates universal correctness properties per spec.

  **Feature: mfa-service-modernization-2025**
  """

  use ExUnit.Case, async: true
  use ExUnitProperties

  alias MfaService.Device.Fingerprint
  alias MfaService.Test.Generators

  @moduletag :property

  describe "Property 9: Device Fingerprint Determinism" do
    @tag property: 9
    property "computing fingerprint multiple times produces identical SHA-256 hash" do
      check all attributes <- Generators.device_attributes(), max_runs: 100 do
        fingerprint1 = Fingerprint.compute(attributes)
        fingerprint2 = Fingerprint.compute(attributes)
        fingerprint3 = Fingerprint.compute(attributes)

        # All computations must produce identical results
        assert fingerprint1 == fingerprint2
        assert fingerprint2 == fingerprint3

        # Must be valid SHA-256 hex (64 lowercase hex chars)
        assert String.length(fingerprint1) == 64
        assert String.match?(fingerprint1, ~r/^[a-f0-9]{64}$/)
      end
    end

    property "different attributes produce different fingerprints" do
      check all attrs1 <- Generators.device_attributes(),
                attrs2 <- Generators.device_attributes(),
                attrs1 != attrs2,
                max_runs: 100 do
        fp1 = Fingerprint.compute(attrs1)
        fp2 = Fingerprint.compute(attrs2)

        # Different inputs should (almost always) produce different outputs
        # Note: SHA-256 collisions are astronomically unlikely
        assert fp1 != fp2
      end
    end
  end

  describe "Property 10: Device Fingerprint Reflexivity" do
    @tag property: 10
    property "comparing fingerprint with itself returns 100% similarity" do
      check all attributes <- Generators.device_attributes(), max_runs: 100 do
        result = Fingerprint.compare(attributes, attributes)

        assert result.match == false  # Same attributes, not same hash string
        assert result.similarity == 1.0
        assert result.significant_change == false
      end
    end

    property "comparing identical hash strings returns match=true" do
      check all attributes <- Generators.device_attributes(), max_runs: 100 do
        fingerprint = Fingerprint.compute(attributes)
        result = Fingerprint.compare(fingerprint, fingerprint)

        assert result.match == true
        assert result.similarity == 1.0
        assert result.significant_change == false
      end
    end
  end

  describe "Property 11: Device Fingerprint Similarity Calculation" do
    @tag property: 11
    property "similarity equals K/N where K is matching attributes out of N total" do
      check all base_attrs <- Generators.device_attributes(),
                max_runs: 100 do
        # Create a modified version with some attributes changed
        modified_attrs = modify_some_attributes(base_attrs)

        result = Fingerprint.compare(base_attrs, modified_attrs)

        # Calculate expected similarity
        all_keys = MapSet.union(
          MapSet.new(Map.keys(base_attrs)),
          MapSet.new(Map.keys(modified_attrs))
        )

        matching_count = Enum.count(all_keys, fn key ->
          Map.get(base_attrs, key) == Map.get(modified_attrs, key)
        end)

        expected_similarity = matching_count / MapSet.size(all_keys)

        # Allow small floating point tolerance
        assert_in_delta result.similarity, expected_similarity, 0.001
      end
    end
  end

  describe "Property 12: Device Fingerprint Significant Change Threshold" do
    @tag property: 12
    property "similarity < 70% triggers significant_change and requires_reauth" do
      check all base_attrs <- Generators.device_attributes(),
                max_runs: 100 do
        # Create attributes with >30% difference (significant change)
        significantly_different = create_significantly_different(base_attrs)

        result = Fingerprint.compare(base_attrs, significantly_different)

        if result.similarity < 0.7 do
          assert result.significant_change == true
          assert Fingerprint.requires_reauth?(result) == true
        end
      end
    end

    property "similarity >= 70% does not trigger significant_change" do
      check all base_attrs <- Generators.device_attributes(),
                max_runs: 100 do
        # Create attributes with <30% difference (minor change)
        minor_change = create_minor_change(base_attrs)

        result = Fingerprint.compare(base_attrs, minor_change)

        if result.similarity >= 0.7 do
          assert result.significant_change == false
          assert Fingerprint.requires_reauth?(result) == false
        end
      end
    end

    property "threshold is exactly 30% (0.3)" do
      # Verify the threshold constant
      assert Fingerprint.significant_change_threshold() == 0.3
    end
  end

  describe "Attribute Extraction" do
    property "missing headers default to empty strings" do
      check all _iteration <- StreamData.constant(:ok), max_runs: 100 do
        # Empty headers should produce attributes with empty defaults
        attrs = Fingerprint.extract_attributes(%{}, %{})

        assert attrs.user_agent == ""
        assert attrs.accept_language == ""
        assert attrs.accept_encoding == ""
        assert attrs.screen_resolution == ""
        assert attrs.timezone == ""
        assert attrs.platform == ""
        assert attrs.plugins == []
        assert attrs.canvas_hash == ""
        assert attrs.webgl_hash == ""
      end
    end

    property "headers are correctly extracted" do
      check all user_agent <- StreamData.string(:alphanumeric, min_length: 10, max_length: 100),
                accept_language <- StreamData.member_of(["en-US", "pt-BR", "es-ES"]),
                max_runs: 100 do
        headers = %{
          "user-agent" => user_agent,
          "accept-language" => accept_language
        }

        attrs = Fingerprint.extract_attributes(headers, %{})

        assert attrs.user_agent == user_agent
        assert attrs.accept_language == accept_language
      end
    end
  end

  # Helper functions

  defp modify_some_attributes(attrs) do
    # Randomly modify 2-3 attributes
    keys_to_modify = Enum.take_random(Map.keys(attrs), Enum.random(2..3))

    Enum.reduce(keys_to_modify, attrs, fn key, acc ->
      case key do
        :user_agent -> Map.put(acc, key, "Modified/1.0")
        :accept_language -> Map.put(acc, key, "modified-LANG")
        :timezone -> Map.put(acc, key, "Modified/Timezone")
        :screen_resolution -> Map.put(acc, key, "9999x9999")
        :platform -> Map.put(acc, key, "ModifiedPlatform")
        _ -> acc
      end
    end)
  end

  defp create_significantly_different(attrs) do
    # Change more than 30% of attributes (at least 4 out of 9)
    keys_to_modify = Enum.take_random(Map.keys(attrs), 5)

    Enum.reduce(keys_to_modify, attrs, fn key, acc ->
      Map.put(acc, key, "COMPLETELY_DIFFERENT_#{key}")
    end)
  end

  defp create_minor_change(attrs) do
    # Change less than 30% of attributes (at most 2 out of 9)
    keys_to_modify = Enum.take_random(Map.keys(attrs), 1)

    Enum.reduce(keys_to_modify, attrs, fn key, acc ->
      Map.put(acc, key, "minor_change_#{key}")
    end)
  end
end
