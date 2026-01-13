defmodule AuthPlatform.Codec.CodecPropertyTest do
  @moduledoc """
  Property-based tests for Codecs.

  Property 17: JSON Codec Round-Trip
  Property 18: Base64 Codec Round-Trip
  """
  use ExUnit.Case, async: true
  use ExUnitProperties

  alias AuthPlatform.Codec.JSON
  alias AuthPlatform.Codec.Base64

  @moduletag :property

  describe "Property 17: JSON Codec Round-Trip" do
    property "encode then decode returns original value" do
      check all(value <- json_encodable_generator()) do
        {:ok, encoded} = JSON.encode(value)
        {:ok, decoded} = JSON.decode(encoded)

        assert decoded == normalize_keys(value)
      end
    end

    property "valid? returns true for all encoded values" do
      check all(value <- json_encodable_generator()) do
        {:ok, encoded} = JSON.encode(value)
        assert JSON.valid?(encoded)
      end
    end

    property "encode_pretty produces valid JSON" do
      check all(value <- json_encodable_generator()) do
        {:ok, pretty} = JSON.encode_pretty(value)
        assert JSON.valid?(pretty)
        {:ok, decoded} = JSON.decode(pretty)
        assert decoded == normalize_keys(value)
      end
    end
  end

  describe "Property 18: Base64 Codec Round-Trip" do
    property "standard Base64 encode/decode round-trip" do
      check all(data <- binary()) do
        encoded = Base64.encode(data)
        {:ok, decoded} = Base64.decode(encoded)
        assert decoded == data
      end
    end

    property "URL-safe Base64 encode/decode round-trip" do
      check all(data <- binary()) do
        encoded = Base64.encode_url_safe(data)
        {:ok, decoded} = Base64.decode_url_safe(encoded)
        assert decoded == data
      end
    end

    property "valid? returns true for all encoded values" do
      check all(data <- binary()) do
        encoded = Base64.encode(data)
        assert Base64.valid?(encoded)
      end
    end

    property "valid_url_safe? returns true for all URL-safe encoded values" do
      check all(data <- binary()) do
        encoded = Base64.encode_url_safe(data)
        assert Base64.valid_url_safe?(encoded)
      end
    end

    property "URL-safe encoding contains no + or / characters" do
      check all(data <- binary(min_length: 10)) do
        encoded = Base64.encode_url_safe(data)
        refute String.contains?(encoded, "+")
        refute String.contains?(encoded, "/")
      end
    end

    property "URL-safe encoding contains no padding" do
      check all(data <- binary()) do
        encoded = Base64.encode_url_safe(data)
        refute String.ends_with?(encoded, "=")
      end
    end
  end

  # Generators

  defp json_encodable_generator do
    one_of([
      # Primitives
      integer(),
      float(),
      boolean(),
      string(:printable),
      constant(nil),
      # Collections
      list_of(integer(), max_length: 5),
      list_of(string(:printable), max_length: 5),
      # Maps with string keys (JSON requirement)
      map_of(
        string(:alphanumeric, min_length: 1, max_length: 10),
        one_of([integer(), string(:printable), boolean()]),
        max_length: 5
      )
    ])
  end

  # JSON always decodes to string keys
  defp normalize_keys(value) when is_map(value) do
    value
    |> Enum.map(fn {k, v} -> {to_string(k), normalize_keys(v)} end)
    |> Map.new()
  end

  defp normalize_keys(value) when is_list(value) do
    Enum.map(value, &normalize_keys/1)
  end

  defp normalize_keys(value), do: value
end
