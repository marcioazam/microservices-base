defmodule SessionIdentityCore.Crypto.HealthCheckPropertyTest do
  @moduledoc """
  Property tests for health check integration.
  
  Property 15: Health Check Integration
  Validates: Requirements 6.3
  """

  use ExUnit.Case, async: true
  use ExUnitProperties

  alias SessionIdentityCore.Crypto.HealthCheck

  @min_runs 100

  describe "Property 15: Health Check Integration" do
    property "check returns valid health states" do
      check all _ <- constant(nil),
                max_runs: @min_runs do
        result = HealthCheck.check()
        assert result in [:ok, :degraded, :unhealthy]
      end
    end

    property "status returns map with required keys" do
      check all _ <- constant(nil),
                max_runs: @min_runs do
        status = HealthCheck.status()
        
        assert is_map(status)
        assert Map.has_key?(status, :crypto_service)
        assert Map.has_key?(status, :using_fallback)
        assert status.crypto_service in [:ok, :degraded, :unhealthy]
        assert is_boolean(status.using_fallback)
      end
    end

    property "required_for_readiness returns boolean" do
      check all _ <- constant(nil),
                max_runs: @min_runs do
        result = HealthCheck.required_for_readiness?()
        assert is_boolean(result)
      end
    end

    property "status consistency with check result" do
      check all _ <- constant(nil),
                max_runs: @min_runs do
        check_result = HealthCheck.check()
        status = HealthCheck.status()
        
        assert status.crypto_service == check_result
        
        # Fallback should be true when degraded or unhealthy
        case check_result do
          :ok -> assert status.using_fallback == false
          :degraded -> assert status.using_fallback == true
          :unhealthy -> assert status.using_fallback == true
        end
      end
    end
  end
end
