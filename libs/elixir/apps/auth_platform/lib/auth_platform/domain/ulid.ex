defmodule AuthPlatform.Domain.ULID do
  @moduledoc """
  ULID (Universally Unique Lexicographically Sortable Identifier) value object.

  ULIDs are 128-bit identifiers that are:
  - Lexicographically sortable
  - Time-ordered (first 48 bits are timestamp)
  - URL-safe (Crockford's Base32)

  ## Usage

      alias AuthPlatform.Domain.ULID

      # Generate a new ULID
      ulid = ULID.generate()
      ulid.value  # "01ARZ3NDEKTSV4RRFFQ69G5FAV"

      # Parse an existing ULID
      {:ok, ulid} = ULID.new("01ARZ3NDEKTSV4RRFFQ69G5FAV")

      # Extract timestamp
      ULID.timestamp(ulid)  # ~U[2016-07-30 23:54:10.259Z]

  """

  @type t :: %__MODULE__{value: String.t()}

  @enforce_keys [:value]
  defstruct [:value]

  # Crockford's Base32 alphabet (excludes I, L, O, U to avoid confusion)
  @encoding "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
  @encoding_list String.graphemes(@encoding)

  # ULID regex: 26 characters from Crockford's Base32
  @ulid_regex ~r/^[0-9A-HJKMNP-TV-Z]{26}$/i

  @doc """
  Generates a new ULID with current timestamp.

  ## Examples

      iex> ulid = AuthPlatform.Domain.ULID.generate()
      iex> String.length(ulid.value)
      26

  """
  @spec generate() :: t()
  def generate do
    generate(System.system_time(:millisecond))
  end

  @doc """
  Generates a new ULID with a specific timestamp (milliseconds since Unix epoch).
  """
  @spec generate(non_neg_integer()) :: t()
  def generate(timestamp_ms) when is_integer(timestamp_ms) and timestamp_ms >= 0 do
    # 48 bits for timestamp, 80 bits for randomness
    random_bytes = :crypto.strong_rand_bytes(10)

    value = encode_ulid(timestamp_ms, random_bytes)
    %__MODULE__{value: value}
  end

  @doc """
  Creates a ULID from an existing string.

  ## Examples

      iex> AuthPlatform.Domain.ULID.new("01ARZ3NDEKTSV4RRFFQ69G5FAV")
      {:ok, %AuthPlatform.Domain.ULID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}}

      iex> AuthPlatform.Domain.ULID.new("invalid")
      {:error, "invalid ULID format"}

  """
  @spec new(String.t()) :: {:ok, t()} | {:error, String.t()}
  def new(value) when is_binary(value) do
    normalized = value |> String.trim() |> String.upcase()

    if Regex.match?(@ulid_regex, normalized) do
      {:ok, %__MODULE__{value: normalized}}
    else
      {:error, "invalid ULID format"}
    end
  end

  def new(_), do: {:error, "ULID must be a string"}

  @doc """
  Creates a ULID from a string, raising on invalid input.
  """
  @spec new!(String.t()) :: t()
  def new!(value) do
    case new(value) do
      {:ok, ulid} -> ulid
      {:error, reason} -> raise ArgumentError, reason
    end
  end

  @doc """
  Returns the ULID value as a string.
  """
  @spec to_string(t()) :: String.t()
  def to_string(%__MODULE__{value: value}), do: value

  @doc """
  Extracts the timestamp from a ULID as a DateTime.

  ## Examples

      iex> ulid = AuthPlatform.Domain.ULID.generate()
      iex> %DateTime{} = AuthPlatform.Domain.ULID.timestamp(ulid)

  """
  @spec timestamp(t()) :: DateTime.t()
  def timestamp(%__MODULE__{value: value}) do
    # First 10 characters encode the 48-bit timestamp
    timestamp_chars = String.slice(value, 0, 10)
    timestamp_ms = decode_timestamp(timestamp_chars)

    DateTime.from_unix!(timestamp_ms, :millisecond)
  end

  @doc """
  Extracts the timestamp as milliseconds since Unix epoch.
  """
  @spec timestamp_ms(t()) :: non_neg_integer()
  def timestamp_ms(%__MODULE__{value: value}) do
    timestamp_chars = String.slice(value, 0, 10)
    decode_timestamp(timestamp_chars)
  end

  # ============================================================================
  # Private Functions
  # ============================================================================

  defp encode_ulid(timestamp_ms, random_bytes) do
    timestamp_encoded = encode_timestamp(timestamp_ms)
    random_encoded = encode_random(random_bytes)

    timestamp_encoded <> random_encoded
  end

  defp encode_timestamp(timestamp_ms) do
    # Encode 48-bit timestamp into 10 Base32 characters
    for i <- 9..0//-1, into: "" do
      shift = i * 5
      index = timestamp_ms >>> shift &&& 0x1F
      Enum.at(@encoding_list, index)
    end
  end

  defp encode_random(<<bytes::binary-size(10)>>) do
    # Encode 80 bits (10 bytes) into 16 Base32 characters
    <<n::unsigned-big-integer-size(80)>> = bytes

    for i <- 15..0//-1, into: "" do
      shift = i * 5
      index = n >>> shift &&& 0x1F
      Enum.at(@encoding_list, index)
    end
  end

  defp decode_timestamp(chars) do
    chars
    |> String.graphemes()
    |> Enum.reduce(0, fn char, acc ->
      index = decode_char(char)
      acc * 32 + index
    end)
  end

  defp decode_char(char) do
    char = String.upcase(char)

    case :binary.match(@encoding, char) do
      {pos, _} -> pos
      :nomatch -> 0
    end
  end

  defimpl String.Chars do
    def to_string(%{value: value}), do: value
  end

  defimpl Jason.Encoder do
    def encode(%{value: value}, opts), do: Jason.Encode.string(value, opts)
  end
end
