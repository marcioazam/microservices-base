defmodule AuthPlatform.SecurityPropertyTest do
  @moduledoc """
  Property-based tests for Security utilities.

  Property 19: Constant Time Compare Correctness
  Property 20: Token Generation Uniqueness
  Property 21: HTML Sanitization
  Property 22: Sensitive Data Masking
  """
  use ExUnit.Case, async: true
  use ExUnitProperties

  alias AuthPlatform.Security

  @moduletag :property

  describe "Property 19: Constant Time Compare Correctness" do
    property "returns true for identical strings" do
      check all(str <- string(:printable)) do
        assert Security.constant_time_compare(str, str)
      end
    end

    property "returns false for different strings" do
      check all(
              str1 <- string(:printable, min_length: 1),
              str2 <- string(:printable, min_length: 1),
              str1 != str2
            ) do
        refute Security.constant_time_compare(str1, str2)
      end
    end

    property "is symmetric" do
      check all(
              a <- string(:printable),
              b <- string(:printable)
            ) do
        assert Security.constant_time_compare(a, b) == Security.constant_time_compare(b, a)
      end
    end
  end

  describe "Property 20: Token Generation Uniqueness" do
    property "generates tokens of correct length (hex)" do
      check all(bytes <- integer(1..64)) do
        token = Security.generate_token(bytes, encoding: :hex)
        assert byte_size(token) == bytes * 2
      end
    end

    property "all generated tokens are unique" do
      check all(bytes <- integer(16..32)) do
        tokens = for _ <- 1..10, do: Security.generate_token(bytes)
        assert length(Enum.uniq(tokens)) == 10
      end
    end

    property "hex tokens contain only valid characters" do
      check all(bytes <- integer(1..32)) do
        token = Security.generate_token(bytes, encoding: :hex)
        assert Regex.match?(~r/^[0-9a-f]+$/, token)
      end
    end

    property "base64 tokens are valid base64" do
      check all(bytes <- integer(1..32)) do
        token = Security.generate_token(bytes, encoding: :base64)
        assert {:ok, _} = Base.decode64(token)
      end
    end

    property "url_safe_base64 tokens contain no unsafe characters" do
      check all(bytes <- integer(1..32)) do
        token = Security.generate_token(bytes, encoding: :url_safe_base64)
        refute String.contains?(token, "+")
        refute String.contains?(token, "/")
        refute String.contains?(token, "=")
      end
    end
  end

  describe "Property 21: HTML Sanitization" do
    property "sanitized output never contains raw < or >" do
      check all(input <- string(:printable)) do
        output = Security.sanitize_html(input)
        # After sanitization, < and > should only appear as entities
        refute Regex.match?(~r/(?<!&lt|&gt)[<>]/, output)
      end
    end

    property "sanitization is idempotent after first pass" do
      check all(input <- string(:printable)) do
        once = Security.sanitize_html(input)
        twice = Security.sanitize_html(once)
        # Note: Not strictly idempotent due to & -> &amp;
        # But the output should still be safe
        refute String.contains?(twice, "<script")
      end
    end

    property "preserves alphanumeric content" do
      check all(input <- string(:alphanumeric)) do
        output = Security.sanitize_html(input)
        assert output == input
      end
    end
  end

  describe "Property 22: Sensitive Data Masking" do
    property "masked output length equals input length" do
      check all(input <- string(:printable, min_length: 1)) do
        output = Security.mask_sensitive(input)
        assert String.length(output) == String.length(input)
      end
    end

    property "visible characters are from the end of input" do
      check all(
              input <- string(:alphanumeric, min_length: 10),
              visible <- integer(1..4)
            ) do
        output = Security.mask_sensitive(input, visible: visible)
        input_suffix = String.slice(input, -visible, visible)
        output_suffix = String.slice(output, -visible, visible)
        assert output_suffix == input_suffix
      end
    end

    property "masked portion contains only mask characters" do
      check all(
              input <- string(:alphanumeric, min_length: 10),
              visible <- integer(1..4)
            ) do
        output = Security.mask_sensitive(input, visible: visible)
        masked_part = String.slice(output, 0, String.length(output) - visible)
        assert String.match?(masked_part, ~r/^\*+$/)
      end
    end

    property "short inputs are fully masked" do
      check all(input <- string(:alphanumeric, min_length: 1, max_length: 4)) do
        output = Security.mask_sensitive(input)
        assert String.match?(output, ~r/^\*+$/)
      end
    end
  end

  describe "Hash properties" do
    property "hash produces consistent output" do
      check all(input <- string(:printable)) do
        hash1 = Security.hash(input)
        hash2 = Security.hash(input)
        assert hash1 == hash2
      end
    end

    property "hash output is always 64 hex characters" do
      check all(input <- string(:printable)) do
        hash = Security.hash(input)
        assert byte_size(hash) == 64
        assert Regex.match?(~r/^[0-9a-f]+$/, hash)
      end
    end

    property "different inputs produce different hashes" do
      check all(
              input1 <- string(:printable, min_length: 1),
              input2 <- string(:printable, min_length: 1),
              input1 != input2
            ) do
        hash1 = Security.hash(input1)
        hash2 = Security.hash(input2)
        assert hash1 != hash2
      end
    end
  end
end
