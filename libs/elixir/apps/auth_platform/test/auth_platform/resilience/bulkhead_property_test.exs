defmodule AuthPlatform.Resilience.BulkheadPropertyTest do
  @moduledoc """
  Property-based tests for Bulkhead.

  Property 16: Bulkhead Isolation
  """
  use ExUnit.Case, async: false
  use ExUnitProperties

  alias AuthPlatform.Resilience.Bulkhead
  alias AuthPlatform.Resilience.Supervisor, as: ResilienceSupervisor

  @moduletag :property

  describe "Property 16: Bulkhead Isolation" do
    property "never exceeds max_concurrent active permits" do
      check all(max_concurrent <- integer(1..10)) do
        name = unique_name()

        {:ok, _} =
          ResilienceSupervisor.start_bulkhead(name, %{
            max_concurrent: max_concurrent,
            max_queue: 50,
            queue_timeout_ms: 100
          })

        max_observed = :counters.new(1, [:atomics])
        active_counter = :counters.new(1, [:atomics])

        # Fire many concurrent requests
        tasks =
          for _ <- 1..(max_concurrent * 3) do
            Task.async(fn ->
              Bulkhead.execute(
                name,
                fn ->
                  :counters.add(active_counter, 1, 1)
                  current = :counters.get(active_counter, 1)

                  # Update max observed
                  max_val = :counters.get(max_observed, 1)
                  if current > max_val, do: :counters.put(max_observed, 1, current)

                  Process.sleep(10)
                  :counters.sub(active_counter, 1, 1)
                  :ok
                end,
                timeout: 500
              )
            end)
          end

        Task.await_many(tasks, 5000)

        # Max observed should never exceed max_concurrent
        assert :counters.get(max_observed, 1) <= max_concurrent

        cleanup(name)
      end
    end

    property "available_permits + active = max_concurrent" do
      check all(max_concurrent <- integer(2..10)) do
        name = unique_name()

        {:ok, _} =
          ResilienceSupervisor.start_bulkhead(name, %{
            max_concurrent: max_concurrent,
            max_queue: 0,
            queue_timeout_ms: 100
          })

        # Acquire some permits
        acquired = :rand.uniform(max_concurrent)

        for _ <- 1..acquired do
          Bulkhead.acquire(name, 100)
        end

        status = Bulkhead.get_status(name)
        assert status.active + status.available == max_concurrent

        cleanup(name)
      end
    end

    property "rejects when queue is full" do
      check all(
              max_concurrent <- integer(1..5),
              max_queue <- integer(0..5)
            ) do
        name = unique_name()

        {:ok, _} =
          ResilienceSupervisor.start_bulkhead(name, %{
            max_concurrent: max_concurrent,
            max_queue: max_queue,
            queue_timeout_ms: 100
          })

        # Fill all permits
        for _ <- 1..max_concurrent, do: Bulkhead.acquire(name, 100)

        # Fill queue with async tasks
        queue_tasks =
          for _ <- 1..max_queue do
            Task.async(fn -> Bulkhead.acquire(name, 1000) end)
          end

        Process.sleep(20)

        # Next request should be rejected
        result = Bulkhead.acquire(name, 10)
        assert result == {:error, :bulkhead_full}

        # Cleanup - release permits so queued tasks can complete
        for _ <- 1..max_concurrent, do: Bulkhead.release(name)
        Task.await_many(queue_tasks, 2000)

        cleanup(name)
      end
    end

    property "permits are always released after execute" do
      check all(max_concurrent <- integer(1..5)) do
        name = unique_name()

        {:ok, _} =
          ResilienceSupervisor.start_bulkhead(name, %{
            max_concurrent: max_concurrent,
            max_queue: 0,
            queue_timeout_ms: 100
          })

        initial_available = Bulkhead.available_permits(name)

        # Execute some operations (some may fail)
        for _ <- 1..max_concurrent do
          Bulkhead.execute(name, fn ->
            if :rand.uniform(2) == 1, do: raise("random error"), else: :ok
          end)
        end

        Process.sleep(10)
        final_available = Bulkhead.available_permits(name)

        # All permits should be returned
        assert final_available == initial_available

        cleanup(name)
      end
    end
  end

  defp unique_name do
    :"bh_prop_#{:erlang.unique_integer([:positive])}"
  end

  defp cleanup(name) do
    ResilienceSupervisor.stop_child(name)
  end
end
