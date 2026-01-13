defmodule AuthPlatform.Resilience.SupervisorTest do
  use ExUnit.Case, async: false

  alias AuthPlatform.Resilience.Supervisor, as: ResilienceSupervisor
  alias AuthPlatform.Resilience.Registry

  describe "which_children/0" do
    test "returns list of children" do
      children = ResilienceSupervisor.which_children()
      assert is_list(children)
    end
  end

  describe "count_children/0" do
    test "returns map with child counts" do
      counts = ResilienceSupervisor.count_children()

      assert is_map(counts)
      assert Map.has_key?(counts, :specs)
      assert Map.has_key?(counts, :active)
      assert Map.has_key?(counts, :supervisors)
      assert Map.has_key?(counts, :workers)
    end
  end

  describe "stop_child/1" do
    test "returns {:error, :not_found} for unknown component" do
      assert {:error, :not_found} = ResilienceSupervisor.stop_child(:unknown_component)
    end
  end

  # Note: start_circuit_breaker, start_rate_limiter, start_bulkhead tests
  # will be added when those modules are implemented (Tasks 10, 12, 13)
end
