defmodule MfaService.TOTP.Validator do
  @moduledoc """
  TOTP validation with time window tolerance (±1 step).
  Uses AuthPlatform.Security for constant-time comparison.

  ## Features
  - RFC 6238 compliant validation
  - ±1 time step tolerance for clock drift
  - Constant-time comparison to prevent timing attacks
  """

  alias AuthPlatform.Security

  @default_period 30
  @default_digits 6
  @time_window 1

  @type code :: String.t()
  @type secret :: String.t()

  @doc """
  Validates a TOTP code against the secret.
  Accepts codes within the current and adjacent time windows (±1 step).

  ## Options
    * `:period` - Time step in seconds (default: 30)
    * `:digits` - Code length (default: 6)
    * `:timestamp` - Unix timestamp to validate against (default: current time)
  """
  @spec validate(code(), secret(), keyword()) :: :ok | {:error, :invalid_code}
  def validate(code, secret, opts \\ []) do
    period = Keyword.get(opts, :period, @default_period)
    digits = Keyword.get(opts, :digits, @default_digits)
    timestamp = Keyword.get(opts, :timestamp, System.system_time(:second))

    time_steps =
      for offset <- -@time_window..@time_window do
        timestamp + offset * period
      end

    valid? =
      Enum.any?(time_steps, fn ts ->
        expected_code = generate_code(secret, ts, period, digits)
        Security.constant_time_compare(code, expected_code)
      end)

    if valid?, do: :ok, else: {:error, :invalid_code}
  end

  @doc """
  Generates a TOTP code for the given timestamp.
  Implements HOTP algorithm per RFC 4226.
  """
  @spec generate_code(secret(), integer() | nil, pos_integer(), pos_integer()) :: code()
  def generate_code(secret, timestamp \\ nil, period \\ @default_period, digits \\ @default_digits) do
    timestamp = timestamp || System.system_time(:second)
    counter = div(timestamp, period)

    counter_bytes = <<counter::unsigned-big-integer-size(64)>>

    hmac = :crypto.mac(:hmac, :sha, Base.decode32!(secret, padding: false), counter_bytes)

    offset = :binary.at(hmac, byte_size(hmac) - 1) &&& 0x0F

    <<_::binary-size(offset), code::unsigned-big-integer-size(32), _::binary>> = hmac

    code = (code &&& 0x7FFFFFFF) |> rem(trunc(:math.pow(10, digits)))

    code
    |> Integer.to_string()
    |> String.pad_leading(digits, "0")
  end

  @doc """
  Checks if a code is within the valid time window.
  """
  @spec is_within_window?(code(), secret(), keyword()) :: boolean()
  def is_within_window?(code, secret, opts \\ []) do
    case validate(code, secret, opts) do
      :ok -> true
      _ -> false
    end
  end

  @doc """
  Gets the current time step for debugging/testing.
  """
  @spec current_time_step(pos_integer()) :: non_neg_integer()
  def current_time_step(period \\ @default_period) do
    div(System.system_time(:second), period)
  end

  @doc """
  Returns the remaining seconds until the next time step.
  """
  @spec seconds_remaining(pos_integer()) :: non_neg_integer()
  def seconds_remaining(period \\ @default_period) do
    period - rem(System.system_time(:second), period)
  end
end
