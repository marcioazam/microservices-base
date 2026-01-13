defmodule AuthPlatform.Resilience.BulkheadTest do
  use ExUnit.Case, async: false

  alias AuthPlatform.Resilience.Bulkhead
  alias AuthPlatform.Resilience.Supervisor, as: ResilienceSupervisor

  @moduletag :bulkhead

  setup do
    name = :"bh_test_#{:erlang.unique_integer([:positive])}"
    {:ok, name: name}
  end

  describe "start_link/1" do
    test "starts with default config", %{name: name} do
      {:ok, pid} = start_bulkhead(name)
      assert Process.alive?(pid)

      status = Bulkhead.get_status(name)
      assert status.config.max_concurrent == 10
      assert status.config.max_queue == 100
    end

    test "starts with custom config", %{name: name} do
      config = %{max_concurrent: 5, max_queue: 20, queue_timeout_ms: 1000}
      {:ok, pid} = start_bulkhead(name, config)

      assert Process.alive?(pid)
      status = Bulkhead.get_status(name)
      assert status.config.max_concurrent == 5
      assert status.config.max_queue == 20
    end
  end

  describe "execute/3" do
    test "executes function when permits available", %{name: name} do
      {:ok, _} = start_bulkhead(name, %{max_concurrent: 5})

      result = Bulkhead.execute(name, fn -> :success end)
      assert result == {:ok, :success}
    end

    test "returns error when function raises", %{name: name} do
      {:ok, _} = start_bulkhead(name, %{max_concurrent: 5})

      result = Bulkhead.execute(name, fn -> raise "boom" end)
      assert {:error, %RuntimeError{message: "boom"}} = result
    end

    test "releases permit after execution", %{name: name} do
      {:ok, _} = start_bulkhead(name, %{max_concurrent: 1})

      assert Bulkhead.available_permits(name) == 1

      Bulkhead.execute(name, fn -> :ok end)

      assert Bulkhead.available_permits(name) == 1
    end

    test "releases permit even on error", %{name: name} do
      {:ok, _} = start_bulkhead(name, %{max_concurrent: 1})

      Bulkhead.execute(name, fn -> raise "error" end)

      assert Bulkhead.available_permits(name) == 1
    end
  end

  describe "acquire/2 and release/1" do
    test "acquires permit when available", %{name: name} do
      {:ok, _} = start_bulkhead(name, %{max_concurrent: 5})

      assert Bulkhead.acquire(name) == :ok
      assert Bulkhead.available_permits(name) == 4
    end

    test "releases permit", %{name: name} do
      {:ok, _} = start_bulkhead(name, %{max_concurrent: 5})

      Bulkhead.acquire(name)
      assert Bulkhead.available_permits(name) == 4

      Bulkhead.release(name)
      Process.sleep(5)
      assert Bulkhead.available_permits(name) == 5
    end

    test "queues request when no permits available", %{name: name} do
      {:ok, _} = start_bulkhead(name, %{max_concurrent: 1, max_queue: 10})

      # Acquire the only permit
      Bulkhead.acquire(name)

      # Start async acquire that should queue
      task =
        Task.async(fn ->
          Bulkhead.acquire(name, 1000)
        end)

      Process.sleep(10)
      status = Bulkhead.get_status(name)
      assert status.queued == 1

      # Release permit
      Bulkhead.release(name)

      # Queued request should succeed
      assert Task.await(task) == :ok
    end

    test "rejects when queue is full", %{name: name} do
      {:ok, _} = start_bulkhead(name, %{max_concurrent: 1, max_queue: 0})

      # Acquire the only permit
      Bulkhead.acquire(name)

      # Next request should be rejected
      result = Bulkhead.acquire(name, 100)
      assert result == {:error, :bulkhead_full}
    end

    test "returns timeout when queue wait exceeds timeout", %{name: name} do
      {:ok, _} = start_bulkhead(name, %{max_concurrent: 1, max_queue: 10})

      # Acquire the only permit
      Bulkhead.acquire(name)

      # Request with short timeout should timeout
      result = Bulkhead.acquire(name, 10)
      assert result == {:error, :timeout}
    end
  end

  describe "available_permits/1" do
    test "returns correct count", %{name: name} do
      {:ok, _} = start_bulkhead(name, %{max_concurrent: 5})

      assert Bulkhead.available_permits(name) == 5

      Bulkhead.acquire(name)
      Bulkhead.acquire(name)

      assert Bulkhead.available_permits(name) == 3
    end
  end

  describe "get_status/1" do
    test "returns status information", %{name: name} do
      config = %{max_concurrent: 5, max_queue: 20, queue_timeout_ms: 1000}
      {:ok, _} = start_bulkhead(name, config)

      Bulkhead.acquire(name)

      status = Bulkhead.get_status(name)

      assert status.name == name
      assert status.active == 1
      assert status.queued == 0
      assert status.available == 4
      assert status.config.max_concurrent == 5
    end
  end

  describe "concurrent execution" do
    test "limits concurrent executions", %{name: name} do
      {:ok, _} = start_bulkhead(name, %{max_concurrent: 3, max_queue: 10})

      counter = :counters.new(1, [:atomics])
      max_concurrent = :counters.new(1, [:atomics])

      tasks =
        for _ <- 1..10 do
          Task.async(fn ->
            Bulkhead.execute(name, fn ->
              current = :counters.add(counter, 1, 1)
              current_val = :counters.get(counter, 1)

              # Track max concurrent
              max_val = :counters.get(max_concurrent, 1)

              if current_val > max_val do
                :counters.put(max_concurrent, 1, current_val)
              end

              Process.sleep(50)
              :counters.sub(counter, 1, 1)
              :ok
            end)
          end)
        end

      Task.await_many(tasks, 5000)

      # Max concurrent should not exceed 3
      assert :counters.get(max_concurrent, 1) <= 3
    end
  end

  defp start_bulkhead(name, config \\ %{}) do
    ResilienceSupervisor.start_bulkhead(name, config)
  end
end
