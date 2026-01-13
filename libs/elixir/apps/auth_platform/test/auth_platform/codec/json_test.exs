defmodule AuthPlatform.Codec.JSONTest do
  use ExUnit.Case, async: true

  alias AuthPlatform.Codec.JSON

  describe "encode/1" do
    test "encodes map to JSON" do
      assert {:ok, json} = JSON.encode(%{name: "John", age: 30})
      assert json =~ "name"
      assert json =~ "John"
    end

    test "encodes list to JSON" do
      assert {:ok, "[1,2,3]"} = JSON.encode([1, 2, 3])
    end

    test "encodes nested structures" do
      data = %{user: %{name: "John", tags: ["admin", "user"]}}
      assert {:ok, json} = JSON.encode(data)
      assert json =~ "user"
      assert json =~ "tags"
    end

    test "returns error for non-encodable terms" do
      assert {:error, _} = JSON.encode({:tuple, :value})
    end
  end

  describe "encode!/1" do
    test "encodes and returns string" do
      json = JSON.encode!(%{key: "value"})
      assert is_binary(json)
      assert json =~ "key"
    end

    test "raises on error" do
      assert_raise Protocol.UndefinedError, fn ->
        JSON.encode!({:tuple, :value})
      end
    end
  end

  describe "encode_pretty/1" do
    test "encodes with formatting" do
      assert {:ok, json} = JSON.encode_pretty(%{name: "John"})
      assert json =~ "\n"
      assert json =~ "  "
    end
  end

  describe "decode/2" do
    test "decodes JSON to map with string keys" do
      assert {:ok, %{"name" => "John"}} = JSON.decode(~s({"name": "John"}))
    end

    test "decodes JSON to map with atom keys" do
      assert {:ok, %{name: "John"}} = JSON.decode(~s({"name": "John"}), keys: :atoms)
    end

    test "decodes arrays" do
      assert {:ok, [1, 2, 3]} = JSON.decode("[1, 2, 3]")
    end

    test "returns error for invalid JSON" do
      assert {:error, %Jason.DecodeError{}} = JSON.decode("not json")
    end

    test "returns error for empty string" do
      assert {:error, _} = JSON.decode("")
    end
  end

  describe "decode!/2" do
    test "decodes and returns term" do
      assert %{"key" => "value"} = JSON.decode!(~s({"key": "value"}))
    end

    test "raises on invalid JSON" do
      assert_raise Jason.DecodeError, fn ->
        JSON.decode!("invalid")
      end
    end
  end

  describe "valid?/1" do
    test "returns true for valid JSON" do
      assert JSON.valid?(~s({"name": "John"}))
      assert JSON.valid?("[1, 2, 3]")
      assert JSON.valid?("null")
      assert JSON.valid?("true")
      assert JSON.valid?("123")
    end

    test "returns false for invalid JSON" do
      refute JSON.valid?("not json")
      refute JSON.valid?("{invalid}")
      refute JSON.valid?("")
    end
  end
end
