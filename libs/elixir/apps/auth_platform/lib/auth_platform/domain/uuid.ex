defmodule AuthPlatform.Domain.UUID do
  @moduledoc """
  RFC 4122 UUID v4 identifier value object.

  Provides generation and validation of UUID v4 (random) identifiers.

  ## Usage

      alias AuthPlatform.Domain.UUID

      # Generate a new UUID
      uuid = UUID.generate()
      uuid.value  # "550e8400-e29b-41d4-a716-446655440000"

      # Parse an existing UUID
      {:ok, uuid} = UUID.new("550e8400-e29b-41d4-a716-446655440000")

      # String conversion
      to_string(uuid)  # "550e8400-e29b-41d4-a716-446655440000"

  """

  @type t :: %__MODULE__{value: String.t()}

  @enforce_keys [:value]
  defstruct [:value]

  # UUID v4 regex: 8-4-4-4-12 hex digits with version 4 and variant bits
  @uuid_regex ~r/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i

  @doc """
  Generates a new random UUID v4.

  ## Examples

      iex> uuid = AuthPlatform.Domain.UUID.generate()
      iex> String.length(uuid.value)
      36

  """
  @spec generate() :: t()
  def generate do
    # Generate 16 random bytes
    <<a::32, b::16, c::16, d::16, e::48>> = :crypto.strong_rand_bytes(16)

    # Set version (4) and variant (10xx) bits
    # Version 4: bits 12-15 of time_hi_and_version = 0100
    # Variant: bits 6-7 of clock_seq_hi_and_reserved = 10
    c_versioned = (c &&& 0x0FFF) ||| 0x4000
    d_varianted = (d &&& 0x3FFF) ||| 0x8000

    value =
      :io_lib.format(
        "~8.16.0b-~4.16.0b-~4.16.0b-~4.16.0b-~12.16.0b",
        [a, b, c_versioned, d_varianted, e]
      )
      |> IO.iodata_to_binary()
      |> String.downcase()

    %__MODULE__{value: value}
  end

  @doc """
  Creates a UUID from an existing string.

  Validates that the string is a valid UUID v4 format.

  ## Examples

      iex> AuthPlatform.Domain.UUID.new("550e8400-e29b-41d4-a716-446655440000")
      {:ok, %AuthPlatform.Domain.UUID{value: "550e8400-e29b-41d4-a716-446655440000"}}

      iex> AuthPlatform.Domain.UUID.new("invalid")
      {:error, "invalid UUID format"}

  """
  @spec new(String.t()) :: {:ok, t()} | {:error, String.t()}
  def new(value) when is_binary(value) do
    trimmed = String.trim(value)

    if Regex.match?(@uuid_regex, trimmed) do
      {:ok, %__MODULE__{value: String.downcase(trimmed)}}
    else
      {:error, "invalid UUID format"}
    end
  end

  def new(_), do: {:error, "UUID must be a string"}

  @doc """
  Creates a UUID from a string, raising on invalid input.

  ## Examples

      iex> AuthPlatform.Domain.UUID.new!("550e8400-e29b-41d4-a716-446655440000")
      %AuthPlatform.Domain.UUID{value: "550e8400-e29b-41d4-a716-446655440000"}

  """
  @spec new!(String.t()) :: t()
  def new!(value) do
    case new(value) do
      {:ok, uuid} -> uuid
      {:error, reason} -> raise ArgumentError, reason
    end
  end

  @doc """
  Returns the UUID value as a string.
  """
  @spec to_string(t()) :: String.t()
  def to_string(%__MODULE__{value: value}), do: value

  @doc """
  Returns the UUID as a binary (16 bytes).

  ## Examples

      iex> uuid = AuthPlatform.Domain.UUID.new!("550e8400-e29b-41d4-a716-446655440000")
      iex> byte_size(AuthPlatform.Domain.UUID.to_binary(uuid))
      16

  """
  @spec to_binary(t()) :: binary()
  def to_binary(%__MODULE__{value: value}) do
    value
    |> String.replace("-", "")
    |> Base.decode16!(case: :lower)
  end

  defimpl String.Chars do
    def to_string(%{value: value}), do: value
  end

  defimpl Jason.Encoder do
    def encode(%{value: value}, opts), do: Jason.Encode.string(value, opts)
  end
end
