defmodule AuthPlatform.Clients.Cache do
  @moduledoc """
  Client for the platform Cache Service.

  Provides a resilient interface to the centralized cache service with
  circuit breaker protection and telemetry.

  ## Usage

      # Get a cached value
      {:ok, value} = Cache.get("user:123")

      # Set a value with TTL
      :ok = Cache.set("user:123", user_data, ttl: 3600)

      # Delete a cached value
      :ok = Cache.delete("user:123")

  ## Configuration

      config :auth_platform_clients, :cache,
        endpoint: "localhost:50052",
        circuit_breaker: :cache_breaker

  """

  alias AuthPlatform.Resilience.CircuitBreaker

  @circuit_breaker_name :cache_service_breaker

  @type key :: String.t()
  @type value :: any()
  @type ttl :: pos_integer()

  @doc """
  Gets a value from the cache.

  Returns `{:ok, value}` if found, `{:ok, nil}` if not found,
  or `{:error, reason}` on failure.

  ## Examples

      {:ok, user} = Cache.get("user:123")
      {:ok, nil} = Cache.get("nonexistent")

  """
  @spec get(key()) :: {:ok, value() | nil} | {:error, any()}
  def get(key) when is_binary(key) do
    emit_telemetry(:get, key, fn ->
      CircuitBreaker.execute(@circuit_breaker_name, fn ->
        do_get(key)
      end)
    end)
  end

  @doc """
  Sets a value in the cache with optional TTL.

  ## Options

    * `:ttl` - Time to live in seconds (default: 3600)

  ## Examples

      :ok = Cache.set("user:123", %{name: "John"})
      :ok = Cache.set("session:abc", session_data, ttl: 1800)

  """
  @spec set(key(), value(), keyword()) :: :ok | {:error, any()}
  def set(key, value, opts \\ []) when is_binary(key) do
    ttl = Keyword.get(opts, :ttl, 3600)

    emit_telemetry(:set, key, fn ->
      case CircuitBreaker.execute(@circuit_breaker_name, fn ->
             do_set(key, value, ttl)
           end) do
        {:ok, :ok} -> :ok
        {:error, reason} -> {:error, reason}
      end
    end)
  end

  @doc """
  Deletes a value from the cache.

  ## Examples

      :ok = Cache.delete("user:123")

  """
  @spec delete(key()) :: :ok | {:error, any()}
  def delete(key) when is_binary(key) do
    emit_telemetry(:delete, key, fn ->
      case CircuitBreaker.execute(@circuit_breaker_name, fn ->
             do_delete(key)
           end) do
        {:ok, :ok} -> :ok
        {:error, reason} -> {:error, reason}
      end
    end)
  end

  @doc """
  Checks if a key exists in the cache.

  ## Examples

      true = Cache.exists?("user:123")
      false = Cache.exists?("nonexistent")

  """
  @spec exists?(key()) :: boolean()
  def exists?(key) when is_binary(key) do
    case get(key) do
      {:ok, nil} -> false
      {:ok, _} -> true
      {:error, _} -> false
    end
  end

  @doc """
  Starts the cache client circuit breaker.

  Should be called during application startup.
  """
  @spec start_circuit_breaker(map()) :: {:ok, pid()} | {:error, any()}
  def start_circuit_breaker(config \\ %{}) do
    alias AuthPlatform.Resilience.Supervisor, as: ResilienceSupervisor
    ResilienceSupervisor.start_circuit_breaker(@circuit_breaker_name, config)
  end

  # Private functions - In real implementation, these would call gRPC

  defp do_get(_key) do
    # Simulated cache lookup
    {:ok, nil}
  end

  defp do_set(_key, _value, _ttl) do
    # Simulated cache set
    {:ok, :ok}
  end

  defp do_delete(_key) do
    # Simulated cache delete
    {:ok, :ok}
  end

  defp emit_telemetry(operation, key, fun) do
    start_time = System.monotonic_time()

    result = fun.()

    duration = System.monotonic_time() - start_time

    :telemetry.execute(
      [:auth_platform, :cache, operation],
      %{duration: duration},
      %{key: key, result: result_type(result)}
    )

    result
  end

  defp result_type({:ok, nil}), do: :miss
  defp result_type({:ok, _}), do: :hit
  defp result_type(:ok), do: :success
  defp result_type({:error, _}), do: :error
end
