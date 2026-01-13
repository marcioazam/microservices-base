defmodule AuthPlatform.Domain.Money do
  @moduledoc """
  Monetary value with currency handling.

  Money values are stored as integers representing the smallest currency unit
  (e.g., cents for USD, pence for GBP).

  ## Usage

      alias AuthPlatform.Domain.Money

      # Creating money
      {:ok, price} = Money.new(1000, :USD)  # $10.00
      {:ok, tax} = Money.new(80, :USD)      # $0.80

      # Arithmetic
      {:ok, total} = Money.add(price, tax)  # $10.80

      # Formatting
      Money.format(total)  # "$10.80"

  ## Supported Currencies

  - USD (US Dollar)
  - EUR (Euro)
  - GBP (British Pound)
  - BRL (Brazilian Real)
  - JPY (Japanese Yen)

  """

  @type currency :: :USD | :EUR | :GBP | :BRL | :JPY

  @type t :: %__MODULE__{
          amount: integer(),
          currency: currency()
        }

  @enforce_keys [:amount, :currency]
  defstruct [:amount, :currency]

  @currencies [:USD, :EUR, :GBP, :BRL, :JPY]

  @currency_symbols %{
    USD: "$",
    EUR: "€",
    GBP: "£",
    BRL: "R$",
    JPY: "¥"
  }

  @currency_decimals %{
    USD: 2,
    EUR: 2,
    GBP: 2,
    BRL: 2,
    JPY: 0
  }

  @doc """
  Returns the list of supported currencies.
  """
  @spec supported_currencies() :: [currency()]
  def supported_currencies, do: @currencies

  @doc """
  Creates a new Money value.

  Amount should be in the smallest currency unit (cents, pence, etc.).

  ## Examples

      iex> AuthPlatform.Domain.Money.new(1000, :USD)
      {:ok, %AuthPlatform.Domain.Money{amount: 1000, currency: :USD}}

      iex> AuthPlatform.Domain.Money.new(1000, :INVALID)
      {:error, "unsupported currency: INVALID"}

      iex> AuthPlatform.Domain.Money.new("100", :USD)
      {:error, "amount must be an integer"}

  """
  @spec new(integer(), currency()) :: {:ok, t()} | {:error, String.t()}
  def new(amount, currency) when is_integer(amount) and currency in @currencies do
    {:ok, %__MODULE__{amount: amount, currency: currency}}
  end

  def new(_amount, currency) when currency not in @currencies do
    {:error, "unsupported currency: #{inspect(currency)}"}
  end

  def new(_amount, _currency) do
    {:error, "amount must be an integer"}
  end

  @doc """
  Creates a new Money value, raising on invalid input.
  """
  @spec new!(integer(), currency()) :: t()
  def new!(amount, currency) do
    case new(amount, currency) do
      {:ok, money} -> money
      {:error, reason} -> raise ArgumentError, reason
    end
  end

  @doc """
  Creates Money from a decimal amount (e.g., 10.50 for $10.50).

  ## Examples

      iex> AuthPlatform.Domain.Money.from_decimal(10.50, :USD)
      {:ok, %AuthPlatform.Domain.Money{amount: 1050, currency: :USD}}

  """
  @spec from_decimal(number(), currency()) :: {:ok, t()} | {:error, String.t()}
  def from_decimal(decimal, currency) when is_number(decimal) and currency in @currencies do
    decimals = Map.get(@currency_decimals, currency, 2)
    amount = round(decimal * :math.pow(10, decimals))
    new(amount, currency)
  end

  def from_decimal(_, currency) when currency not in @currencies do
    {:error, "unsupported currency: #{inspect(currency)}"}
  end

  @doc """
  Adds two Money values of the same currency.

  ## Examples

      iex> {:ok, a} = AuthPlatform.Domain.Money.new(100, :USD)
      iex> {:ok, b} = AuthPlatform.Domain.Money.new(50, :USD)
      iex> AuthPlatform.Domain.Money.add(a, b)
      {:ok, %AuthPlatform.Domain.Money{amount: 150, currency: :USD}}

      iex> {:ok, usd} = AuthPlatform.Domain.Money.new(100, :USD)
      iex> {:ok, eur} = AuthPlatform.Domain.Money.new(100, :EUR)
      iex> AuthPlatform.Domain.Money.add(usd, eur)
      {:error, "currency mismatch: USD vs EUR"}

  """
  @spec add(t(), t()) :: {:ok, t()} | {:error, String.t()}
  def add(%__MODULE__{currency: c, amount: a1}, %__MODULE__{currency: c, amount: a2}) do
    {:ok, %__MODULE__{amount: a1 + a2, currency: c}}
  end

  def add(%__MODULE__{currency: c1}, %__MODULE__{currency: c2}) do
    {:error, "currency mismatch: #{c1} vs #{c2}"}
  end

  @doc """
  Subtracts one Money value from another.

  ## Examples

      iex> {:ok, a} = AuthPlatform.Domain.Money.new(100, :USD)
      iex> {:ok, b} = AuthPlatform.Domain.Money.new(30, :USD)
      iex> AuthPlatform.Domain.Money.subtract(a, b)
      {:ok, %AuthPlatform.Domain.Money{amount: 70, currency: :USD}}

  """
  @spec subtract(t(), t()) :: {:ok, t()} | {:error, String.t()}
  def subtract(%__MODULE__{currency: c, amount: a1}, %__MODULE__{currency: c, amount: a2}) do
    {:ok, %__MODULE__{amount: a1 - a2, currency: c}}
  end

  def subtract(%__MODULE__{currency: c1}, %__MODULE__{currency: c2}) do
    {:error, "currency mismatch: #{c1} vs #{c2}"}
  end

  @doc """
  Multiplies a Money value by a factor.

  ## Examples

      iex> {:ok, price} = AuthPlatform.Domain.Money.new(100, :USD)
      iex> AuthPlatform.Domain.Money.multiply(price, 3)
      {:ok, %AuthPlatform.Domain.Money{amount: 300, currency: :USD}}

  """
  @spec multiply(t(), number()) :: {:ok, t()}
  def multiply(%__MODULE__{amount: amount, currency: currency}, factor) when is_number(factor) do
    {:ok, %__MODULE__{amount: round(amount * factor), currency: currency}}
  end

  @doc """
  Returns the decimal representation of the amount.

  ## Examples

      iex> {:ok, money} = AuthPlatform.Domain.Money.new(1050, :USD)
      iex> AuthPlatform.Domain.Money.to_decimal(money)
      10.5

  """
  @spec to_decimal(t()) :: float()
  def to_decimal(%__MODULE__{amount: amount, currency: currency}) do
    decimals = Map.get(@currency_decimals, currency, 2)
    amount / :math.pow(10, decimals)
  end

  @doc """
  Formats the Money value as a string with currency symbol.

  ## Examples

      iex> {:ok, money} = AuthPlatform.Domain.Money.new(1050, :USD)
      iex> AuthPlatform.Domain.Money.format(money)
      "$10.50"

      iex> {:ok, yen} = AuthPlatform.Domain.Money.new(1000, :JPY)
      iex> AuthPlatform.Domain.Money.format(yen)
      "¥1000"

  """
  @spec format(t()) :: String.t()
  def format(%__MODULE__{amount: amount, currency: currency}) do
    symbol = Map.get(@currency_symbols, currency, "")
    decimals = Map.get(@currency_decimals, currency, 2)

    if decimals == 0 do
      "#{symbol}#{amount}"
    else
      divisor = :math.pow(10, decimals)
      formatted = :erlang.float_to_binary(amount / divisor, decimals: decimals)
      "#{symbol}#{formatted}"
    end
  end

  @doc """
  Checks if the amount is zero.
  """
  @spec zero?(t()) :: boolean()
  def zero?(%__MODULE__{amount: 0}), do: true
  def zero?(%__MODULE__{}), do: false

  @doc """
  Checks if the amount is positive.
  """
  @spec positive?(t()) :: boolean()
  def positive?(%__MODULE__{amount: amount}), do: amount > 0

  @doc """
  Checks if the amount is negative.
  """
  @spec negative?(t()) :: boolean()
  def negative?(%__MODULE__{amount: amount}), do: amount < 0

  defimpl Jason.Encoder do
    def encode(%{amount: amount, currency: currency}, opts) do
      Jason.Encode.map(%{amount: amount, currency: currency}, opts)
    end
  end
end
