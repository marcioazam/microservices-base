defmodule MfaService.TOTP.Validator do
  @moduledoc """
  TOTP validation with time window tolerance (±1 step).
  """

  @default_period 30
  @default_digits 6
  @time_window 1  # Allow ±1 time step for clock drift

  @doc """
  Validates a TOTP code against the secret.
  Accepts codes within the current and adjacent time windows (±1 step).
  """
  def validate(code, secret, opts \\ []) do
    period = Keyword.get(opts, :period, @default_period)
    digits = Keyword.get(opts, :digits, @default_digits)
    timestamp = Keyword.get(opts, :timestamp, System.system_time(:second))

    # Check current and adjacent time windows
    time_steps = for offset <- -@time_window..@time_window do
      timestamp + (offset * period)
    end

    valid? = Enum.any?(time_steps, fn ts ->
      expected_code = generate_code(secret, ts, period, digits)
      secure_compare(code, expected_code)
    end)

    if valid? do
      :ok
    else
      {:error, :invalid_code}
    end
  end

  @doc """
  Generates a TOTP code for the given timestamp.
  """
  def generate_code(secret, timestamp \\ nil, period \\ @default_period, digits \\ @default_digits) do
    timestamp = timestamp || System.system_time(:second)
    counter = div(timestamp, period)
    
    # HOTP algorithm
    counter_bytes = <<counter::unsigned-big-integer-size(64)>>
    
    hmac = :crypto.mac(:hmac, :sha, Base.decode32!(secret, padding: false), counter_bytes)
    
    # Dynamic truncation
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
  def is_within_window?(code, secret, opts \\ []) do
    case validate(code, secret, opts) do
      :ok -> true
      _ -> false
    end
  end

  @doc """
  Gets the current time step for debugging/testing.
  """
  def current_time_step(period \\ @default_period) do
    div(System.system_time(:second), period)
  end

  # Constant-time comparison to prevent timing attacks
  defp secure_compare(a, b) when byte_size(a) == byte_size(b) do
    :crypto.hash_equals(a, b)
  end

  defp secure_compare(_, _), do: false
end
