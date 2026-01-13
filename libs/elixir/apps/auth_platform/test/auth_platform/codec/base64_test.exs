defmodule AuthPlatform.Codec.Base64Test do
  use ExUnit.Case, async: true

  alias AuthPlatform.Codec.Base64

  describe "encode/1" do
    test "encodes string to Base64" do
      assert Base64.encode("hello") == "aGVsbG8="
    end

    test "encodes binary data" do
      assert Base64.encode(<<1, 2, 3>>) == "AQID"
    end

    test "encodes empty string" do
      assert Base64.encode("") == ""
    end

    test "encodes unicode" do
      encoded = Base64.encode("héllo")
      assert is_binary(encoded)
    end
  end

  describe "encode_url_safe/1" do
    test "encodes to URL-safe Base64" do
      # Standard Base64 would use + and /
      data = <<251, 255, 254>>
      encoded = Base64.encode_url_safe(data)
      refute encoded =~ "+"
      refute encoded =~ "/"
    end

    test "omits padding" do
      encoded = Base64.encode_url_safe("hello")
      refute encoded =~ "="
    end
  end

  describe "decode/1" do
    test "decodes valid Base64" do
      assert {:ok, "hello"} = Base64.decode("aGVsbG8=")
    end

    test "decodes binary data" do
      assert {:ok, <<1, 2, 3>>} = Base64.decode("AQID")
    end

    test "returns error for invalid Base64" do
      assert {:error, :invalid_base64} = Base64.decode("!!invalid!!")
    end

    test "decodes empty string" do
      assert {:ok, ""} = Base64.decode("")
    end
  end

  describe "decode!/1" do
    test "decodes and returns binary" do
      assert "hello" = Base64.decode!("aGVsbG8=")
    end

    test "raises on invalid input" do
      assert_raise ArgumentError, fn ->
        Base64.decode!("!!invalid!!")
      end
    end
  end

  describe "decode_url_safe/1" do
    test "decodes URL-safe Base64" do
      # First encode, then decode
      original = "hello?world"
      encoded = Base64.encode_url_safe(original)
      assert {:ok, ^original} = Base64.decode_url_safe(encoded)
    end

    test "returns error for invalid input" do
      assert {:error, :invalid_base64} = Base64.decode_url_safe("!!invalid!!")
    end
  end

  describe "decode_url_safe!/1" do
    test "decodes and returns binary" do
      original = "test data"
      encoded = Base64.encode_url_safe(original)
      assert ^original = Base64.decode_url_safe!(encoded)
    end

    test "raises on invalid input" do
      assert_raise ArgumentError, fn ->
        Base64.decode_url_safe!("!!invalid!!")
      end
    end
  end

  describe "valid?/1" do
    test "returns true for valid Base64" do
      assert Base64.valid?("aGVsbG8=")
      assert Base64.valid?("AQID")
      assert Base64.valid?("")
    end

    test "returns false for invalid Base64" do
      refute Base64.valid?("!!invalid!!")
      refute Base64.valid?("not base64!!!")
    end
  end

  describe "valid_url_safe?/1" do
    test "returns true for valid URL-safe Base64" do
      encoded = Base64.encode_url_safe("test")
      assert Base64.valid_url_safe?(encoded)
    end

    test "returns false for invalid input" do
      refute Base64.valid_url_safe?("!!invalid!!")
    end
  end

  describe "round-trip" do
    test "standard Base64 round-trip" do
      original = "Hello, World! 123"
      encoded = Base64.encode(original)
      assert {:ok, ^original} = Base64.decode(encoded)
    end

    test "URL-safe Base64 round-trip" do
      original = "data with special chars: ?&="
      encoded = Base64.encode_url_safe(original)
      assert {:ok, ^original} = Base64.decode_url_safe(encoded)
    end

    test "binary data round-trip" do
      original = :crypto.strong_rand_bytes(100)
      encoded = Base64.encode(original)
      assert {:ok, ^original} = Base64.decode(encoded)
    end
  end
end
