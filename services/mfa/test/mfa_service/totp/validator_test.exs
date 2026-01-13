defmodule MfaService.TOTP.ValidatorTest do
  @moduledoc """
  Unit tests for TOTP Validator module.
  """

  use ExUnit.Case, async: true

  alias MfaService.TOTP.{Generator, Validator}

  @moduletag :unit

  describe "validate/3" do
    test "validates correct TOTP code" do
      secret = Generator.generate_secret()
      code = Validator.generate_code(secret)

      assert Validator.validate(code, secret) == :ok
    end

    test "rejects invalid TOTP code" do
      secret = Generator.generate_secret()

      assert Validator.validate("000000", secret) == {:error, :invalid_code}
    end

    test "accepts code from previous time window" do
      secret = Generator.generate_secret()
      timestamp = System.system_time(:second)

      # Generate code for previous window
      prev_code = Validator.generate_code(secret, timestamp - 30)

      # Should still be valid at current timestamp
      assert Validator.validate(prev_code, secret, timestamp: timestamp) == :ok
    end

    test "accepts code from next time window" do
      secret = Generator.generate_secret()
      timestamp = System.system_time(:second)

      # Generate code for next window
      next_code = Validator.generate_code(secret, timestamp + 30)

      # Should still be valid at current timestamp
      assert Validator.validate(next_code, secret, timestamp: timestamp) == :ok
    end

    test "rejects code from 2 windows ago" do
      secret = Generator.generate_secret()
      timestamp = System.system_time(:second)

      # Generate code for 2 windows ago
      old_code = Validator.generate_code(secret, timestamp - 90)

      # Should be rejected
      assert Validator.validate(old_code, secret, timestamp: timestamp) == {:error, :invalid_code}
    end

    test "respects custom period" do
      secret = Generator.generate_secret()
      timestamp = System.system_time(:second)

      code = Validator.generate_code(secret, timestamp, 60)

      assert Validator.validate(code, secret, period: 60, timestamp: timestamp) == :ok
    end
  end

  describe "generate_code/4" do
    test "generates 6-digit code by default" do
      secret = Generator.generate_secret()
      code = Validator.generate_code(secret)

      assert String.length(code) == 6
      assert String.match?(code, ~r/^\d{6}$/)
    end

    test "generates code with custom digits" do
      secret = Generator.generate_secret()
      code = Validator.generate_code(secret, nil, 30, 8)

      assert String.length(code) == 8
      assert String.match?(code, ~r/^\d{8}$/)
    end

    test "generates consistent code for same timestamp" do
      secret = Generator.generate_secret()
      timestamp = 1_700_000_000

      code1 = Validator.generate_code(secret, timestamp)
      code2 = Validator.generate_code(secret, timestamp)

      assert code1 == code2
    end

    test "generates different codes for different timestamps" do
      secret = Generator.generate_secret()

      code1 = Validator.generate_code(secret, 1_700_000_000)
      code2 = Validator.generate_code(secret, 1_700_000_030)

      assert code1 != code2
    end
  end

  describe "is_within_window?/3" do
    test "returns true for valid code" do
      secret = Generator.generate_secret()
      code = Validator.generate_code(secret)

      assert Validator.is_within_window?(code, secret) == true
    end

    test "returns false for invalid code" do
      secret = Generator.generate_secret()

      assert Validator.is_within_window?("000000", secret) == false
    end
  end

  describe "current_time_step/1" do
    test "returns positive integer" do
      step = Validator.current_time_step()

      assert is_integer(step)
      assert step > 0
    end

    test "respects custom period" do
      step_30 = Validator.current_time_step(30)
      step_60 = Validator.current_time_step(60)

      # 60-second period should have roughly half the steps
      assert step_30 >= step_60
    end
  end

  describe "seconds_remaining/1" do
    test "returns value between 1 and period" do
      remaining = Validator.seconds_remaining()

      assert remaining >= 1
      assert remaining <= 30
    end

    test "respects custom period" do
      remaining = Validator.seconds_remaining(60)

      assert remaining >= 1
      assert remaining <= 60
    end
  end
end
