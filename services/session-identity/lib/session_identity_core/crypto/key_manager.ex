defmodule SessionIdentityCore.Crypto.KeyManager do
  @moduledoc """
  Key metadata caching and rotation management.
  
  Provides:
  - ETS-based key metadata caching with TTL
  - Active key resolution per namespace
  - Multi-version key support for rotation
  - Cache invalidation on rotation events
  """

  use GenServer
  require Logger

  alias SessionIdentityCore.Crypto.{Client, Config}

  @table_name :crypto_key_cache
  @cleanup_interval 60_000  # 1 minute

  @type key_id :: %{namespace: String.t(), id: String.t(), version: non_neg_integer()}

  # Client API

  def start_link(opts \\ []) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end

  @doc """
  Gets the active key for a namespace.
  Uses cache if available and not expired.
  """
  @spec get_active_key(String.t()) :: {:ok, key_id()} | {:error, term()}
  def get_active_key(namespace) do
    cache_key = {:active_key, namespace}
    
    case get_cached(cache_key) do
      {:ok, key_id} -> 
        {:ok, key_id}
      
      :miss ->
        GenServer.call(__MODULE__, {:fetch_active_key, namespace})
    end
  end

  @doc """
  Gets key metadata, using cache if available.
  """
  @spec get_key_metadata(key_id()) :: {:ok, map()} | {:error, term()}
  def get_key_metadata(key_id) do
    cache_key = {:metadata, key_id}
    
    case get_cached(cache_key) do
      {:ok, metadata} -> 
        {:ok, metadata}
      
      :miss ->
        GenServer.call(__MODULE__, {:fetch_metadata, key_id})
    end
  end

  @doc """
  Gets all available versions for a key namespace.
  """
  @spec get_key_versions(String.t(), String.t()) :: {:ok, [key_id()]} | {:error, term()}
  def get_key_versions(namespace, key_id) do
    cache_key = {:versions, namespace, key_id}
    
    case get_cached(cache_key) do
      {:ok, versions} -> 
        {:ok, versions}
      
      :miss ->
        GenServer.call(__MODULE__, {:fetch_versions, namespace, key_id})
    end
  end

  @doc """
  Invalidates cache for a namespace.
  Called when key rotation is detected.
  """
  @spec invalidate_cache(String.t()) :: :ok
  def invalidate_cache(namespace) do
    GenServer.cast(__MODULE__, {:invalidate, namespace})
  end

  @doc """
  Invalidates all cached data.
  """
  @spec invalidate_all() :: :ok
  def invalidate_all do
    GenServer.cast(__MODULE__, :invalidate_all)
  end

  @doc """
  Checks if a key version is deprecated.
  """
  @spec deprecated?(key_id()) :: boolean()
  def deprecated?(key_id) do
    case get_key_metadata(key_id) do
      {:ok, %{state: :DEPRECATED}} -> true
      {:ok, %{state: :PENDING_DESTRUCTION}} -> true
      _ -> false
    end
  end

  @doc """
  Gets the latest version for a key.
  """
  @spec get_latest_version(String.t(), String.t()) :: {:ok, key_id()} | {:error, term()}
  def get_latest_version(namespace, key_id) do
    case get_key_versions(namespace, key_id) do
      {:ok, versions} ->
        latest = Enum.max_by(versions, & &1.version, fn -> nil end)
        if latest, do: {:ok, latest}, else: {:error, :no_versions_found}
      
      error -> error
    end
  end

  # Server Callbacks

  @impl true
  def init(_opts) do
    # Create ETS table for caching
    :ets.new(@table_name, [:named_table, :set, :public, read_concurrency: true])
    
    # Schedule periodic cleanup
    Process.send_after(self(), :cleanup, @cleanup_interval)
    
    {:ok, %{}}
  end

  @impl true
  def handle_call({:fetch_active_key, namespace}, _from, state) do
    result = fetch_and_cache_active_key(namespace)
    {:reply, result, state}
  end

  @impl true
  def handle_call({:fetch_metadata, key_id}, _from, state) do
    result = fetch_and_cache_metadata(key_id)
    {:reply, result, state}
  end

  @impl true
  def handle_call({:fetch_versions, namespace, key_id}, _from, state) do
    result = fetch_and_cache_versions(namespace, key_id)
    {:reply, result, state}
  end

  @impl true
  def handle_cast({:invalidate, namespace}, state) do
    invalidate_namespace(namespace)
    {:noreply, state}
  end

  @impl true
  def handle_cast(:invalidate_all, state) do
    :ets.delete_all_objects(@table_name)
    Logger.info("Key cache invalidated")
    {:noreply, state}
  end

  @impl true
  def handle_info(:cleanup, state) do
    cleanup_expired()
    Process.send_after(self(), :cleanup, @cleanup_interval)
    {:noreply, state}
  end

  # Private Functions

  defp get_cached(key) do
    case :ets.lookup(@table_name, key) do
      [{^key, value, expires_at}] ->
        if System.monotonic_time(:second) < expires_at do
          {:ok, value}
        else
          :ets.delete(@table_name, key)
          :miss
        end
      
      [] ->
        :miss
    end
  end

  defp cache_value(key, value) do
    ttl = Config.get().cache_ttl
    expires_at = System.monotonic_time(:second) + ttl
    :ets.insert(@table_name, {key, value, expires_at})
    value
  end

  defp fetch_and_cache_active_key(namespace) do
    # For now, construct a default key_id
    # In production, this would query crypto-service for the active key
    key_id = %{
      namespace: namespace,
      id: default_key_id_for_namespace(namespace),
      version: 1
    }
    
    cache_key = {:active_key, namespace}
    cache_value(cache_key, key_id)
    {:ok, key_id}
  end

  defp fetch_and_cache_metadata(key_id) do
    case Client.get_key_metadata(key_id) do
      {:ok, metadata} ->
        cache_key = {:metadata, key_id}
        cache_value(cache_key, metadata)
        {:ok, metadata}
      
      error ->
        error
    end
  end

  defp fetch_and_cache_versions(namespace, key_id) do
    # For now, return single version
    # In production, this would query crypto-service for all versions
    versions = [
      %{namespace: namespace, id: key_id, version: 1}
    ]
    
    cache_key = {:versions, namespace, key_id}
    cache_value(cache_key, versions)
    {:ok, versions}
  end

  defp invalidate_namespace(namespace) do
    # Delete all entries for this namespace
    :ets.match_delete(@table_name, {{:active_key, namespace}, :_, :_})
    :ets.match_delete(@table_name, {{:metadata, %{namespace: namespace, id: :_, version: :_}}, :_, :_})
    :ets.match_delete(@table_name, {{:versions, namespace, :_}, :_, :_})
    Logger.info("Key cache invalidated for namespace: #{namespace}")
  end

  defp cleanup_expired do
    now = System.monotonic_time(:second)
    
    # Find and delete expired entries
    :ets.select_delete(@table_name, [
      {{:_, :_, :"$1"}, [{:<, :"$1", now}], [true]}
    ])
  end

  defp default_key_id_for_namespace(namespace) do
    case namespace do
      "session_identity:jwt" -> "jwt-signing-key"
      "session_identity:session" -> "session-dek"
      "session_identity:refresh_token" -> "refresh-token-dek"
      _ -> "default-key"
    end
  end
end
