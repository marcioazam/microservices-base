defmodule AuthPlatform.Domain.Email do
  @moduledoc """
  RFC 5322 compliant email address value object.

  Email addresses are normalized to lowercase and validated against RFC 5322 format.

  ## Usage

      alias AuthPlatform.Domain.Email

      # Creating emails
      {:ok, email} = Email.new("User@Example.com")
      email.value  # "user@example.com"

      # Bang version raises on invalid input
      email = Email.new!("user@example.com")

      # String conversion
      to_string(email)  # "user@example.com"

      # JSON encoding
      Jason.encode!(email)  # "\\"user@example.com\\""

  """

  @type t :: %__MODULE__{value: String.t()}

  @enforce_keys [:value]
  defstruct [:value]

  # RFC 5322 compliant email regex (simplified but practical)
  @email_regex ~r/^[a-zA-Z0-9.!#$%&'*+\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$/

  @doc """
  Creates a new Email from a string.

  The email is validated against RFC 5322 format and normalized to lowercase.

  ## Examples

      iex> AuthPlatform.Domain.Email.new("User@Example.com")
      {:ok, %AuthPlatform.Domain.Email{value: "user@example.com"}}

      iex> AuthPlatform.Domain.Email.new("invalid")
      {:error, "invalid email format"}

      iex> AuthPlatform.Domain.Email.new(123)
      {:error, "email must be a string"}

  """
  @spec new(String.t()) :: {:ok, t()} | {:error, String.t()}
  def new(value) when is_binary(value) do
    trimmed = String.trim(value)

    if Regex.match?(@email_regex, trimmed) do
      {:ok, %__MODULE__{value: String.downcase(trimmed)}}
    else
      {:error, "invalid email format"}
    end
  end

  def new(_), do: {:error, "email must be a string"}

  @doc """
  Creates a new Email from a string, raising on invalid input.

  ## Examples

      iex> AuthPlatform.Domain.Email.new!("user@example.com")
      %AuthPlatform.Domain.Email{value: "user@example.com"}

  ## Raises

      AuthPlatform.Domain.Email.new!("invalid")
      # ** (ArgumentError) invalid email format

  """
  @spec new!(String.t()) :: t()
  def new!(value) do
    case new(value) do
      {:ok, email} -> email
      {:error, reason} -> raise ArgumentError, reason
    end
  end

  @doc """
  Returns the email value as a string.

  ## Examples

      iex> email = AuthPlatform.Domain.Email.new!("user@example.com")
      iex> AuthPlatform.Domain.Email.to_string(email)
      "user@example.com"

  """
  @spec to_string(t()) :: String.t()
  def to_string(%__MODULE__{value: value}), do: value

  @doc """
  Returns the local part of the email (before @).

  ## Examples

      iex> email = AuthPlatform.Domain.Email.new!("user@example.com")
      iex> AuthPlatform.Domain.Email.local_part(email)
      "user"

  """
  @spec local_part(t()) :: String.t()
  def local_part(%__MODULE__{value: value}) do
    [local | _] = String.split(value, "@")
    local
  end

  @doc """
  Returns the domain part of the email (after @).

  ## Examples

      iex> email = AuthPlatform.Domain.Email.new!("user@example.com")
      iex> AuthPlatform.Domain.Email.domain(email)
      "example.com"

  """
  @spec domain(t()) :: String.t()
  def domain(%__MODULE__{value: value}) do
    [_, domain] = String.split(value, "@")
    domain
  end

  defimpl String.Chars do
    def to_string(%{value: value}), do: value
  end

  defimpl Jason.Encoder do
    def encode(%{value: value}, opts), do: Jason.Encode.string(value, opts)
  end
end
