defmodule AuthPlatformTesting do
  @moduledoc """
  Testing utilities for Auth Platform.

  This module provides StreamData generators and test helpers for property-based testing:

  ## Generators

  - `AuthPlatform.Testing.Generators.email_generator/0` - Valid RFC 5322 emails
  - `AuthPlatform.Testing.Generators.uuid_generator/0` - Valid UUID v4 strings
  - `AuthPlatform.Testing.Generators.ulid_generator/0` - Valid ULID strings
  - `AuthPlatform.Testing.Generators.money_generator/0` - Valid Money structs
  - `AuthPlatform.Testing.Generators.phone_number_generator/0` - E.164 format numbers
  - `AuthPlatform.Testing.Generators.url_generator/0` - Valid HTTP/HTTPS URLs
  - `AuthPlatform.Testing.Generators.correlation_id_generator/0` - Correlation IDs for tracing
  - `AuthPlatform.Testing.Generators.app_error_generator/0` - AppError structs

  ## Test Helpers

  - `AuthPlatformTesting.Helpers` - Circuit breaker, retry, and rate limiter test helpers

  ## Usage

      use ExUnit.Case
      use ExUnitProperties

      alias AuthPlatform.Testing.Generators

      property "emails are valid" do
        check all email <- Generators.email_generator() do
          assert {:ok, _} = AuthPlatform.Domain.Email.new(email)
        end
      end

  """

  @doc """
  Returns the library version.
  """
  @spec version() :: String.t()
  def version, do: "0.1.0"
end
