defmodule MfaService.TOTP.ValidatorTest do
  use ExUnit.Case, async: true
  use ExUnitProperties

  alias MfaService.TOTP.{Generator, Validator}

  describe "TOTP Validation" do
    # **Feature: auth-microservices-platform, Property 18: TOTP Validation Window**
    # **Validates: Requirements 6.2**
    property "accepts codes within current and adjacent time windows (±1 step)" do
      check all secret <- StreamData.binary(length: 20),
                period <- StreamData.integer(30..60),
                max_runs: 100 do
        secret_b32 = Base.encode32(secret, padding: false)
        timestamp = System.system_time(:second)

        # Generate codes for current and adjacent windows
        current_code = Validator.generate_code(secret_b32, timestamp, period)
        prev_code = Validator.generate_code(secret_b32, timestamp - period, period)
        next_code = Validator.generate_code(secret_b32, timestamp + period, period)

        # All should be valid
        assert Validator.validate(current_code, secret_b32, period: period, timestamp: timestamp) == :ok
        assert Validator.validate(prev_code, secret_b32, period: period, timestamp: timestamp) == :ok
        assert Validator.validate(next_code, secret_b32, period: period, timestamp: timestamp) == :ok
      end
    end

    property "rejects codes outside the time window" do
      check all secret <- StreamData.binary(length: 20),
                period <- StreamData.integer(30..60),
                max_runs: 100 do
        secret_b32 = Base.encode32(secret, padding: false)
        timestamp = System.system_time(:second)

        # Generate code for 2 periods ago (outside window)
        old_code = Validator.generate_code(secret_b32, timestamp - (2 * period), period)

        # Should be rejected
        assert Validator.validate(old_code, secret_b32, period: period, timestamp: timestamp) == {:error, :invalid_code}
      end
    end

    property "generated codes have correct length" do
      check all secret <- StreamData.binary(length: 20),
                digits <- StreamData.integer(6..8),
                max_runs: 100 do
        secret_b32 = Base.encode32(secret, padding: false)
        code = Validator.generate_code(secret_b32, nil, 30, digits)

        assert String.length(code) == digits
        assert String.match?(code, ~r/^\d+$/)
      end
    end
  end

  describe "Unit tests" do
    test "validates correct TOTP code" do
      secret = Generator.generate_secret()
      code = Validator.generate_code(secret)

      assert Validator.validate(code, secret) == :ok
    end

    test "rejects invalid TOTP code" do
      secret = Generator.generate_secret()

      assert Validator.validate("000000", secret) == {:error, :invalid_code}
    end

    test "current_time_step returns integer" do
      step = Validator.current_time_step()
      assert is_integer(step)
      assert step > 0
    end
  end
end
