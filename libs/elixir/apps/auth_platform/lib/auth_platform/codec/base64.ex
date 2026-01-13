defmodule AuthPlatform.Codec.Base64 do
  @moduledoc """
  Base64 encoding and decoding utilities.

  Provides standard and URL-safe Base64 encoding/decoding with proper error handling.

  ## Usage

      # Standard Base64
      encoded = Base64.encode("hello world")
      {:ok, decoded} = Base64.decode(encoded)

      # URL-safe Base64
      encoded = Base64.encode_url_safe("hello world")
      {:ok, decoded} = Base64.decode_url_safe(encoded)

  """

  alias AuthPlatform.Functional.Result

  @doc """
  Encodes binary data to standard Base64 string.

  ## Examples

      iex> Base64.encode("hello")
      "aGVsbG8="

      iex> Base64.encode(<<1, 2, 3>>)
      "AQID"

  """
  @spec encode(binary()) :: String.t()
  def encode(data) when is_binary(data) do
    Base.encode64(data)
  end

  @doc """
  Encodes binary data to URL-safe Base64 string.

  Uses `-` and `_` instead of `+` and `/`, and omits padding.

  ## Examples

      iex> Base64.encode_url_safe("hello?world")
      "aGVsbG8_d29ybGQ"

  """
  @spec encode_url_safe(binary()) :: String.t()
  def encode_url_safe(data) when is_binary(data) do
    Base.url_encode64(data, padding: false)
  end

  @doc """
  Decodes a standard Base64 string to binary.

  Returns `{:ok, binary}` on success, `{:error, reason}` on failure.

  ## Examples

      iex> Base64.decode("aGVsbG8=")
      {:ok, "hello"}

      iex> Base64.decode("invalid!!!")
      {:error, :invalid_base64}

  """
  @spec decode(String.t()) :: Result.t(binary(), :invalid_base64)
  def decode(encoded) when is_binary(encoded) do
    case Base.decode64(encoded) do
      {:ok, decoded} -> {:ok, decoded}
      :error -> {:error, :invalid_base64}
    end
  end

  @doc """
  Decodes a standard Base64 string to binary, raising on error.

  ## Examples

      iex> Base64.decode!("aGVsbG8=")
      "hello"

  """
  @spec decode!(String.t()) :: binary()
  def decode!(encoded) when is_binary(encoded) do
    case decode(encoded) do
      {:ok, decoded} -> decoded
      {:error, reason} -> raise ArgumentError, "invalid Base64: #{inspect(reason)}"
    end
  end

  @doc """
  Decodes a URL-safe Base64 string to binary.

  Returns `{:ok, binary}` on success, `{:error, reason}` on failure.

  ## Examples

      iex> Base64.decode_url_safe("aGVsbG8_d29ybGQ")
      {:ok, "hello?world"}

  """
  @spec decode_url_safe(String.t()) :: Result.t(binary(), :invalid_base64)
  def decode_url_safe(encoded) when is_binary(encoded) do
    case Base.url_decode64(encoded, padding: false) do
      {:ok, decoded} -> {:ok, decoded}
      :error -> {:error, :invalid_base64}
    end
  end

  @doc """
  Decodes a URL-safe Base64 string to binary, raising on error.

  ## Examples

      iex> Base64.decode_url_safe!("aGVsbG8_d29ybGQ")
      "hello?world"

  """
  @spec decode_url_safe!(String.t()) :: binary()
  def decode_url_safe!(encoded) when is_binary(encoded) do
    case decode_url_safe(encoded) do
      {:ok, decoded} -> decoded
      {:error, reason} -> raise ArgumentError, "invalid URL-safe Base64: #{inspect(reason)}"
    end
  end

  @doc """
  Checks if a string is valid standard Base64.

  ## Examples

      iex> Base64.valid?("aGVsbG8=")
      true

      iex> Base64.valid?("invalid!!!")
      false

  """
  @spec valid?(String.t()) :: boolean()
  def valid?(encoded) when is_binary(encoded) do
    case decode(encoded) do
      {:ok, _} -> true
      {:error, _} -> false
    end
  end

  @doc """
  Checks if a string is valid URL-safe Base64.

  ## Examples

      iex> Base64.valid_url_safe?("aGVsbG8_d29ybGQ")
      true

  """
  @spec valid_url_safe?(String.t()) :: boolean()
  def valid_url_safe?(encoded) when is_binary(encoded) do
    case decode_url_safe(encoded) do
      {:ok, _} -> true
      {:error, _} -> false
    end
  end
end
