# EventStore Race Condition Fix (2026-01-09)

## Problem

The `get_next_sequence/2` function in `event_store.ex` had a critical race condition:

```elixir
# VULNERABLE CODE (BEFORE):
defp get_next_sequence(aggregate_type, aggregate_id) do
  key = sequence_key(aggregate_type, aggregate_id)
  current = current_sequence(aggregate_type, aggregate_id)  # READ
  next = current + 1                                         # INCREMENT
  Cache.set(key, Integer.to_string(next))                  # WRITE
  next
end
```

**Race Condition**: Between the READ and WRITE operations, another process could also read the same `current` value, leading to **duplicate sequence numbers**.

### Impact

- ✅ **Severity**: CRITICAL
- ✅ **Impact**: Violates event sourcing guarantees
- ✅ **Consequence**: Events with duplicate sequence numbers break event replay
- ✅ **Likelihood**: HIGH under concurrent load

## Solution

Use Redis's atomic `INCR` command which is guaranteed to be thread-safe:

```elixir
# SECURE CODE (AFTER):
defp get_next_sequence(aggregate_type, aggregate_id) do
  key = sequence_key(aggregate_type, aggregate_id)

  case Cache.increment(key) do
    {:ok, next} -> next
    {:error, _reason} ->
      # Fallback with warning (still vulnerable)
      # ...
  end
end
```

## Required Implementation

The `AuthPlatform.Clients.Cache` module must implement the `increment/1` function:

```elixir
defmodule AuthPlatform.Clients.Cache do
  @doc """
  Atomically increments the value at key and returns the new value.

  Uses Redis INCR command which is atomic.

  ## Examples

      iex> Cache.increment("counter:session")
      {:ok, 1}

      iex> Cache.increment("counter:session")
      {:ok, 2}
  """
  @spec increment(String.t()) :: {:ok, integer()} | {:error, term()}
  def increment(key) do
    case Redix.command(:redix, ["INCR", key]) do
      {:ok, value} when is_integer(value) -> {:ok, value}
      {:error, reason} -> {:error, reason}
    end
  end
end
```

## Testing

### Unit Test

```elixir
test "get_next_sequence returns monotonic sequence numbers" do
  aggregate_type = "Session"
  aggregate_id = "test-123"

  # Sequential calls should return increasing numbers
  seq1 = EventStore.get_next_sequence(aggregate_type, aggregate_id)
  seq2 = EventStore.get_next_sequence(aggregate_type, aggregate_id)
  seq3 = EventStore.get_next_sequence(aggregate_type, aggregate_id)

  assert seq2 == seq1 + 1
  assert seq3 == seq2 + 1
end
```

### Concurrency Test

```elixir
test "get_next_sequence is thread-safe under concurrent load" do
  aggregate_type = "Session"
  aggregate_id = "concurrent-test"

  # Spawn 100 concurrent processes
  tasks = for _ <- 1..100 do
    Task.async(fn ->
      EventStore.get_next_sequence(aggregate_type, aggregate_id)
    end)
  end

  # Collect all sequence numbers
  sequences = Task.await_many(tasks, 5000)

  # All sequence numbers should be unique
  assert length(Enum.uniq(sequences)) == 100

  # Sequence numbers should be contiguous (no gaps except first)
  sorted = Enum.sort(sequences)
  assert Enum.max(sorted) - Enum.min(sorted) == 99
end
```

## Verification

Before deploying to production:

1. ✅ Implement `Cache.increment/1` using Redis INCR
2. ✅ Run unit tests to verify monotonic behavior
3. ✅ Run concurrency tests to verify no duplicates under load
4. ✅ Monitor logs for "Failed to use atomic increment" warnings
5. ✅ If warnings appear, fix Cache client immediately

## Redis INCR Documentation

From Redis docs:

> **INCR** increments the number stored at key by one. If the key does not exist,
> it is set to 0 before performing the operation. An error is returned if the key
> contains a value of the wrong type or contains a string that can not be
> represented as integer.
>
> **This operation is atomic.** Even when multiple clients issue INCR commands
> simultaneously, there are no race conditions.

## References

- Redis INCR: https://redis.io/commands/incr/
- Event Sourcing Patterns: https://martinfowler.com/eaaDev/EventSourcing.html
- Atomic Operations: https://en.wikipedia.org/wiki/Linearizability
