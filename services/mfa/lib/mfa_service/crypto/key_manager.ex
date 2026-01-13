defmodule MfaService.Crypto.KeyManager do
  @moduledoc """
  Manages MFA encryption keys in Crypto Service.
  Handles key creation, rotation, and metadata caching.
  """

  use GenServer
  require Logger

  alias MfaService.Crypto.{Client, Config, Error}

  @type key_id :: %{namespace: String.t(), id: String.t(), version: non_neg_integer()}
  @type key_metadata :: %{
    id: key_id(),
    algorithm: atom(),
    state: atom(),
    created_at: DateTime.t(),
    expires_at: DateTime.t() | nil
  }

  defstruct [
    :active_key_id,
    :cache,
    :cache_ttl
  ]

  # Client API

  @doc """
  Starts the KeyManager GenServer.
  """
  def start_link(opts \\ []) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end

  @doc """
  Ensures a TOTP encryption key exists in the crypto service.
  Creates one if it doesn't exist.
  """
  @spec ensure_key_exists() :: {:ok, key_id()} | {:error, Error.t()}
  def ensure_key_exists do
    GenServer.call(__MODULE__, :ensure_key_exists)
  end

  @doc """
  Returns the currently active key ID for TOTP encryption.
  """
  @spec get_active_key_id() :: {:ok, key_id()} | {:error, Error.t()}
  def get_active_key_id do
    GenServer.call(__MODULE__, :get_active_key_id)
  end

  @doc """
  Rotates the active encryption key.
  Returns the new key ID.
  """
  @spec rotate_key() :: {:ok, key_id()} | {:error, Error.t()}
  def rotate_key do
    GenServer.call(__MODULE__, :rotate_key)
  end

  @doc """
  Gets metadata for a specific key.
  Uses cache if available and not expired.
  """
  @spec get_key_metadata(key_id()) :: {:ok, key_metadata()} | {:error, Error.t()}
  def get_key_metadata(key_id) do
    GenServer.call(__MODULE__, {:get_key_metadata, key_id})
  end

  @doc """
  Invalidates the cache for a specific key.
  """
  @spec invalidate_cache(key_id()) :: :ok
  def invalidate_cache(key_id) do
    GenServer.cast(__MODULE__, {:invalidate_cache, key_id})
  end

  @doc """
  Clears all cached key metadata.
  """
  @spec clear_cache() :: :ok
  def clear_cache do
    GenServer.cast(__MODULE__, :clear_cache)
  end

  # GenServer callbacks

  @impl true
  def init(_opts) do
    state = %__MODULE__{
      active_key_id: nil,
      cache: %{},
      cache_ttl: Config.cache_ttl() * 1_000  # Convert to milliseconds
    }

    {:ok, state, {:continue, :initialize}}
  end

  @impl true
  def handle_continue(:initialize, state) do
    case do_ensure_key_exists(state) do
      {:ok, key_id, new_state} ->
        Logger.info("KeyManager initialized with active key",
          namespace: key_id.namespace, key_id: key_id.id, version: key_id.version)
        {:noreply, new_state}

      {:error, reason} ->
        Logger.warning("KeyManager initialization failed, will retry",
          reason: inspect(reason))
        Process.send_after(self(), :retry_initialize, 5_000)
        {:noreply, state}
    end
  end

  @impl true
  def handle_info(:retry_initialize, state) do
    {:noreply, state, {:continue, :initialize}}
  end

  @impl true
  def handle_call(:ensure_key_exists, _from, state) do
    case do_ensure_key_exists(state) do
      {:ok, key_id, new_state} ->
        {:reply, {:ok, key_id}, new_state}

      {:error, _} = error ->
        {:reply, error, state}
    end
  end

  @impl true
  def handle_call(:get_active_key_id, _from, %{active_key_id: nil} = state) do
    case do_ensure_key_exists(state) do
      {:ok, key_id, new_state} ->
        {:reply, {:ok, key_id}, new_state}

      {:error, _} = error ->
        {:reply, error, state}
    end
  end

  @impl true
  def handle_call(:get_active_key_id, _from, %{active_key_id: key_id} = state) do
    {:reply, {:ok, key_id}, state}
  end

  @impl true
  def handle_call(:rotate_key, _from, %{active_key_id: nil} = state) do
    {:reply, {:error, Error.new(:key_not_found, "No active key to rotate", nil)}, state}
  end

  @impl true
  def handle_call(:rotate_key, _from, %{active_key_id: current_key_id} = state) do
    correlation_id = generate_correlation_id()

    case Client.rotate_key(current_key_id, correlation_id) do
      {:ok, new_key_id} ->
        Logger.info("Key rotated successfully",
          old_key: current_key_id.id, new_key: new_key_id.id, correlation_id: correlation_id)
        
        # Invalidate old key cache
        new_cache = Map.delete(state.cache, cache_key(current_key_id))
        new_state = %{state | active_key_id: new_key_id, cache: new_cache}
        
        {:reply, {:ok, new_key_id}, new_state}

      {:error, _} = error ->
        Logger.error("Key rotation failed",
          key_id: current_key_id.id, correlation_id: correlation_id)
        {:reply, error, state}
    end
  end

  @impl true
  def handle_call({:get_key_metadata, key_id}, _from, state) do
    case get_cached_metadata(state, key_id) do
      {:ok, metadata} ->
        {:reply, {:ok, metadata}, state}

      :miss ->
        correlation_id = generate_correlation_id()
        
        case Client.get_key_metadata(key_id, correlation_id) do
          {:ok, metadata} ->
            new_state = cache_metadata(state, key_id, metadata)
            {:reply, {:ok, metadata}, new_state}

          {:error, _} = error ->
            {:reply, error, state}
        end
    end
  end

  @impl true
  def handle_cast({:invalidate_cache, key_id}, state) do
    new_cache = Map.delete(state.cache, cache_key(key_id))
    {:noreply, %{state | cache: new_cache}}
  end

  @impl true
  def handle_cast(:clear_cache, state) do
    {:noreply, %{state | cache: %{}}}
  end

  # Private functions

  defp do_ensure_key_exists(%{active_key_id: key_id} = state) when not is_nil(key_id) do
    {:ok, key_id, state}
  end

  defp do_ensure_key_exists(state) do
    namespace = Config.totp_key_namespace()
    correlation_id = generate_correlation_id()

    # Try to generate a new key
    # In a real implementation, we would first check if a key exists
    case Client.generate_key(namespace, %{"service" => "mfa"}, correlation_id) do
      {:ok, key_id} ->
        Logger.info("Generated new TOTP encryption key",
          namespace: namespace, key_id: key_id.id, correlation_id: correlation_id)
        {:ok, key_id, %{state | active_key_id: key_id}}

      {:error, _} = error ->
        error
    end
  end

  defp get_cached_metadata(state, key_id) do
    key = cache_key(key_id)
    
    case Map.get(state.cache, key) do
      nil ->
        :miss

      {metadata, cached_at} ->
        if cache_expired?(cached_at, state.cache_ttl) do
          :miss
        else
          {:ok, metadata}
        end
    end
  end

  defp cache_metadata(state, key_id, metadata) do
    key = cache_key(key_id)
    cached_at = System.monotonic_time(:millisecond)
    new_cache = Map.put(state.cache, key, {metadata, cached_at})
    %{state | cache: new_cache}
  end

  defp cache_expired?(cached_at, ttl) do
    System.monotonic_time(:millisecond) - cached_at > ttl
  end

  defp cache_key(%{namespace: ns, id: id, version: v}) do
    "#{ns}:#{id}:#{v}"
  end

  defp generate_correlation_id do
    UUID.uuid4()
  end
end
