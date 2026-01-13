defmodule AuthPlatform.Resilience.RegistryTest do
  use ExUnit.Case, async: true

  alias AuthPlatform.Resilience.Registry

  describe "via_tuple/1" do
    test "returns correct via tuple format" do
      result = Registry.via_tuple(:test_component)
      assert {:via, Registry, {AuthPlatform.Resilience.Registry, :test_component}} = result
    end
  end

  describe "lookup/1" do
    test "returns {:error, :not_found} for unregistered name" do
      assert {:error, :not_found} = Registry.lookup(:nonexistent_component)
    end

    test "returns {:ok, pid} for registered process" do
      # Register a test process
      name = :"test_process_#{:erlang.unique_integer([:positive])}"
      {:ok, pid} = Agent.start_link(fn -> :ok end, name: Registry.via_tuple(name))

      assert {:ok, ^pid} = Registry.lookup(name)

      # Cleanup
      Agent.stop(pid)
    end
  end

  describe "registered?/1" do
    test "returns false for unregistered name" do
      refute Registry.registered?(:unknown_component)
    end

    test "returns true for registered process" do
      name = :"registered_test_#{:erlang.unique_integer([:positive])}"
      {:ok, pid} = Agent.start_link(fn -> :ok end, name: Registry.via_tuple(name))

      assert Registry.registered?(name)

      Agent.stop(pid)
    end
  end

  describe "all_names/0" do
    test "returns list of registered names" do
      name1 = :"all_names_test_1_#{:erlang.unique_integer([:positive])}"
      name2 = :"all_names_test_2_#{:erlang.unique_integer([:positive])}"

      {:ok, pid1} = Agent.start_link(fn -> :ok end, name: Registry.via_tuple(name1))
      {:ok, pid2} = Agent.start_link(fn -> :ok end, name: Registry.via_tuple(name2))

      names = Registry.all_names()
      assert name1 in names
      assert name2 in names

      Agent.stop(pid1)
      Agent.stop(pid2)
    end
  end

  describe "count/0" do
    test "returns count of registered components" do
      initial_count = Registry.count()

      name = :"count_test_#{:erlang.unique_integer([:positive])}"
      {:ok, pid} = Agent.start_link(fn -> :ok end, name: Registry.via_tuple(name))

      assert Registry.count() == initial_count + 1

      Agent.stop(pid)
    end
  end

  describe "name/0" do
    test "returns the registry module name" do
      assert Registry.name() == AuthPlatform.Resilience.Registry
    end
  end
end
