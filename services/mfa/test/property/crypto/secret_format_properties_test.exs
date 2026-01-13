defmodule MfaService.Crypto.SecretFormatPropertiesTest do
  @moduledoc """
  Property-based tests for SecretFormat.
  Tests correctness properties defined in the design document.
  """
  use ExUnit.Case, async: true
  use ExUnitProperties

  alias MfaService.Crypto.SecretFormat

  # Generators

  defp iv_generator do
    binary(length: 12)
  end

  defp tag_generator do
    binary(length: 16)
  end

  defp ciphertext_generator do
    gen all size <- integer(1..1000),
            data <- binary(length: size) do
      data
    end
  end

  defp key_id_generator do
    gen all namespace <- string(:alphanumeric, min_length: 1, max_length: 50),
            id <- binary(length: 16),
            version <- integer(1..1000) do
      %{
        namespace: namespace,
        id: Base.encode16(id, case: :lower),
        version: version
      }
    end
  end

  describe "Property 7: Format Detection Selects Correct Decryption" do
    @tag :property
    @tag timeout: 120_000
    property "v1 format is correctly detected" do
      check all iv <- iv_generator(),
                tag <- tag_generator(),
                ciphertext <- ciphertext_generator(),
                max_runs: 100 do
        
        encoded = SecretFormat.encode_v1(iv, tag, ciphertext)
        
        assert SecretFormat.detect_version(encoded) == :local
      end
    end

    @tag :property
    property "v2 format is correctly detected" do
      check all key_id <- key_id_generator(),
                iv <- iv_generator(),
                tag <- tag_generator(),
                ciphertext <- ciphertext_generator(),
                max_runs: 100 do
        
        encoded = SecretFormat.encode_v2(key_id, iv, tag, ciphertext)
        
        assert SecretFormat.detect_version(encoded) == :crypto_service
      end
    end

    @tag :property
    property "decode returns correct version" do
      check all key_id <- key_id_generator(),
                iv <- iv_generator(),
                tag <- tag_generator(),
                ciphertext <- ciphertext_generator(),
                max_runs: 100 do
        
        # Test v1
        v1_encoded = SecretFormat.encode_v1(iv, tag, ciphertext)
        assert {:ok, :local, _} = SecretFormat.decode(v1_encoded)
        
        # Test v2
        v2_encoded = SecretFormat.encode_v2(key_id, iv, tag, ciphertext)
        assert {:ok, :crypto_service, _} = SecretFormat.decode(v2_encoded)
      end
    end
  end

  describe "Property 11: Version Byte Presence" do
    @tag :property
    property "all v1 encoded payloads have valid version byte" do
      check all iv <- iv_generator(),
                tag <- tag_generator(),
                ciphertext <- ciphertext_generator(),
                max_runs: 100 do
        
        encoded = SecretFormat.encode_v1(iv, tag, ciphertext)
        
        assert SecretFormat.valid_version?(encoded)
        assert <<0x01, _rest::binary>> = encoded
      end
    end

    @tag :property
    property "all v2 encoded payloads have valid version byte" do
      check all key_id <- key_id_generator(),
                iv <- iv_generator(),
                tag <- tag_generator(),
                ciphertext <- ciphertext_generator(),
                max_runs: 100 do
        
        encoded = SecretFormat.encode_v2(key_id, iv, tag, ciphertext)
        
        assert SecretFormat.valid_version?(encoded)
        assert <<0x02, _rest::binary>> = encoded
      end
    end

    @tag :property
    property "random data without valid version byte is rejected" do
      check all random_data <- binary(min_length: 1, max_length: 100),
                max_runs: 100 do
        
        # Skip if random data happens to start with valid version
        first_byte = :binary.first(random_data)
        
        if first_byte not in [0x01, 0x02] do
          refute SecretFormat.valid_version?(random_data)
          assert SecretFormat.detect_version(random_data) == :unknown
        end
      end
    end
  end

  describe "Property 10: Stored Format Completeness" do
    @tag :property
    property "v1 format contains all required fields" do
      check all iv <- iv_generator(),
                tag <- tag_generator(),
                ciphertext <- ciphertext_generator(),
                max_runs: 100 do
        
        encoded = SecretFormat.encode_v1(iv, tag, ciphertext)
        {:ok, {decoded_iv, decoded_tag, decoded_ciphertext}} = SecretFormat.decode_v1(encoded)
        
        # All fields present and correct
        assert decoded_iv == iv
        assert decoded_tag == tag
        assert decoded_ciphertext == ciphertext
        
        # Correct sizes
        assert byte_size(decoded_iv) == 12
        assert byte_size(decoded_tag) == 16
      end
    end

    @tag :property
    property "v2 format contains all required fields" do
      check all key_id <- key_id_generator(),
                iv <- iv_generator(),
                tag <- tag_generator(),
                ciphertext <- ciphertext_generator(),
                max_runs: 100 do
        
        encoded = SecretFormat.encode_v2(key_id, iv, tag, ciphertext)
        {:ok, payload} = SecretFormat.decode_v2(encoded)
        
        # All fields present
        assert Map.has_key?(payload, :key_id)
        assert Map.has_key?(payload, :iv)
        assert Map.has_key?(payload, :tag)
        assert Map.has_key?(payload, :ciphertext)
        
        # Key ID fields
        assert Map.has_key?(payload.key_id, :namespace)
        assert Map.has_key?(payload.key_id, :id)
        assert Map.has_key?(payload.key_id, :version)
        
        # Values match
        assert payload.key_id.namespace == key_id.namespace
        assert payload.key_id.id == key_id.id
        assert payload.key_id.version == key_id.version
        assert payload.iv == iv
        assert payload.tag == tag
        assert payload.ciphertext == ciphertext
        
        # Correct sizes
        assert byte_size(payload.iv) == 12
        assert byte_size(payload.tag) == 16
      end
    end
  end

  describe "Round-trip encoding/decoding" do
    @tag :property
    property "v1 encode then decode returns original data" do
      check all iv <- iv_generator(),
                tag <- tag_generator(),
                ciphertext <- ciphertext_generator(),
                max_runs: 100 do
        
        encoded = SecretFormat.encode_v1(iv, tag, ciphertext)
        {:ok, {decoded_iv, decoded_tag, decoded_ciphertext}} = SecretFormat.decode_v1(encoded)
        
        assert decoded_iv == iv
        assert decoded_tag == tag
        assert decoded_ciphertext == ciphertext
      end
    end

    @tag :property
    property "v2 encode then decode returns original data" do
      check all key_id <- key_id_generator(),
                iv <- iv_generator(),
                tag <- tag_generator(),
                ciphertext <- ciphertext_generator(),
                max_runs: 100 do
        
        encoded = SecretFormat.encode_v2(key_id, iv, tag, ciphertext)
        {:ok, payload} = SecretFormat.decode_v2(encoded)
        
        assert payload.key_id == key_id
        assert payload.iv == iv
        assert payload.tag == tag
        assert payload.ciphertext == ciphertext
      end
    end
  end

  describe "Minimum payload size" do
    @tag :property
    property "encoded v1 payloads meet minimum size" do
      check all iv <- iv_generator(),
                tag <- tag_generator(),
                ciphertext <- ciphertext_generator(),
                max_runs: 100 do
        
        encoded = SecretFormat.encode_v1(iv, tag, ciphertext)
        min_size = SecretFormat.min_payload_size(:local)
        
        assert byte_size(encoded) >= min_size
      end
    end

    @tag :property
    property "encoded v2 payloads meet minimum size" do
      check all key_id <- key_id_generator(),
                iv <- iv_generator(),
                tag <- tag_generator(),
                ciphertext <- ciphertext_generator(),
                max_runs: 100 do
        
        encoded = SecretFormat.encode_v2(key_id, iv, tag, ciphertext)
        min_size = SecretFormat.min_payload_size(:crypto_service)
        
        assert byte_size(encoded) >= min_size
      end
    end
  end
end
