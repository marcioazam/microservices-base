defmodule AuthPlatform.Domain.PhoneNumber do
  @moduledoc """
  E.164 format phone number value object.

  E.164 is the international telephone numbering plan that ensures phone numbers
  are globally unique. Format: +[country code][subscriber number]

  ## Usage

      alias AuthPlatform.Domain.PhoneNumber

      # Creating phone numbers
      {:ok, phone} = PhoneNumber.new("+5511999999999")

      # String conversion
      to_string(phone)  # "+5511999999999"

  ## E.164 Format

  - Starts with + followed by country code
  - Maximum 15 digits (including country code)
  - No spaces, dashes, or other formatting

  """

  @type t :: %__MODULE__{value: String.t()}

  @enforce_keys [:value]
  defstruct [:value]

  # E.164 format: + followed by 1-15 digits
  @e164_regex ~r/^\+[1-9]\d{1,14}$/

  @doc """
  Creates a new PhoneNumber from a string.

  The phone number must be in E.164 format.

  ## Examples

      iex> AuthPlatform.Domain.PhoneNumber.new("+5511999999999")
      {:ok, %AuthPlatform.Domain.PhoneNumber{value: "+5511999999999"}}

      iex> AuthPlatform.Domain.PhoneNumber.new("5511999999999")
      {:error, "invalid E.164 phone number format"}

      iex> AuthPlatform.Domain.PhoneNumber.new("+1234")
      {:ok, %AuthPlatform.Domain.PhoneNumber{value: "+1234"}}

  """
  @spec new(String.t()) :: {:ok, t()} | {:error, String.t()}
  def new(value) when is_binary(value) do
    # Remove any whitespace
    normalized = String.replace(value, ~r/\s/, "")

    if Regex.match?(@e164_regex, normalized) do
      {:ok, %__MODULE__{value: normalized}}
    else
      {:error, "invalid E.164 phone number format"}
    end
  end

  def new(_), do: {:error, "phone number must be a string"}

  @doc """
  Creates a new PhoneNumber from a string, raising on invalid input.

  ## Examples

      iex> AuthPlatform.Domain.PhoneNumber.new!("+5511999999999")
      %AuthPlatform.Domain.PhoneNumber{value: "+5511999999999"}

  """
  @spec new!(String.t()) :: t()
  def new!(value) do
    case new(value) do
      {:ok, phone} -> phone
      {:error, reason} -> raise ArgumentError, reason
    end
  end

  @doc """
  Creates a PhoneNumber from country code and national number.

  ## Examples

      iex> AuthPlatform.Domain.PhoneNumber.from_parts("55", "11999999999")
      {:ok, %AuthPlatform.Domain.PhoneNumber{value: "+5511999999999"}}

  """
  @spec from_parts(String.t(), String.t()) :: {:ok, t()} | {:error, String.t()}
  def from_parts(country_code, national_number)
      when is_binary(country_code) and is_binary(national_number) do
    new("+#{country_code}#{national_number}")
  end

  @doc """
  Returns the phone number value as a string.
  """
  @spec to_string(t()) :: String.t()
  def to_string(%__MODULE__{value: value}), do: value

  @doc """
  Extracts the country code from the phone number.

  Note: This is a simplified extraction that assumes common country code lengths.
  For production use, consider using a proper phone number library.

  ## Examples

      iex> phone = AuthPlatform.Domain.PhoneNumber.new!("+5511999999999")
      iex> AuthPlatform.Domain.PhoneNumber.country_code(phone)
      "55"

      iex> phone = AuthPlatform.Domain.PhoneNumber.new!("+14155551234")
      iex> AuthPlatform.Domain.PhoneNumber.country_code(phone)
      "1"

  """
  @spec country_code(t()) :: String.t()
  def country_code(%__MODULE__{value: value}) do
    # Remove the leading +
    digits = String.slice(value, 1..-1//1)

    # Simple heuristic: 1 is always 1 digit, most others are 2-3
    cond do
      String.starts_with?(digits, "1") -> "1"
      String.starts_with?(digits, "7") -> "7"
      true -> String.slice(digits, 0, 2)
    end
  end

  @doc """
  Returns the national number (without country code).

  ## Examples

      iex> phone = AuthPlatform.Domain.PhoneNumber.new!("+5511999999999")
      iex> AuthPlatform.Domain.PhoneNumber.national_number(phone)
      "11999999999"

  """
  @spec national_number(t()) :: String.t()
  def national_number(%__MODULE__{value: value} = phone) do
    cc = country_code(phone)
    # Remove + and country code
    String.slice(value, (1 + String.length(cc))..-1//1)
  end

  @doc """
  Formats the phone number for display.

  ## Examples

      iex> phone = AuthPlatform.Domain.PhoneNumber.new!("+5511999999999")
      iex> AuthPlatform.Domain.PhoneNumber.format(phone)
      "+55 11 999999999"

  """
  @spec format(t()) :: String.t()
  def format(%__MODULE__{value: value} = phone) do
    cc = country_code(phone)
    national = national_number(phone)

    # Simple formatting: +CC XX XXXXXXXX
    area_code = String.slice(national, 0, 2)
    subscriber = String.slice(national, 2..-1//1)

    "+#{cc} #{area_code} #{subscriber}"
  end

  defimpl String.Chars do
    def to_string(%{value: value}), do: value
  end

  defimpl Jason.Encoder do
    def encode(%{value: value}, opts), do: Jason.Encode.string(value, opts)
  end
end
