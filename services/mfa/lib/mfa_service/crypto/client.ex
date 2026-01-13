defmodule MfaService.Crypto.Client do
  @moduledoc """
  gRPC client for Crypto Service operations.
  Wraps all calls with circuit breaker and retry patterns.
  Propagates correlation_id for distributed tracing.
  """

  use GenServer
  require Logger

  alias MfaService.Crypto.{Config, Error, Telemetry}
  alias Crypto.V1.{CryptoService, EncryptRequest, DecryptRequest, HealthCheckRequest}
  alias Crypto.V1.{GenerateKeyRequest, RotateKeyRequest, GetKeyMetadataRequest}
  alias Crypto.V1.KeyId, as: ProtoKeyId
  alias Crypto.V1.KeyAlgorithm

  @type key_id :: %{namespace: String.t(), id: String.t(), version: non_neg_integer()}
  @type encrypt_result :: {:ok, %{
    ciphertext: binary(),
    iv: binary(),
    tag: binary(),
    key_id: key_id(),
    algorithm: String.t()
  }}
  @type decrypt_result :: {:ok, binary()}
  @type error_result :: {:error, Error.t()}

  # Client API

  @doc """
  Starts the CryptoClient GenServer.
  """
  def start_link(opts \\ []) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end

  @doc """
  Encrypts plaintext using AES-256-GCM via Crypto Service.
  
  ## Parameters
    - plaintext: The data to encrypt
    - key_id: The key identifier to use for encryption
    - aad: Additional Authenticated Data (typically user_id)
    - correlation_id: Request correlation ID for tracing
  """
  @spec encrypt(binary(), key_id(), binary(), String.t()) :: encrypt_result() | error_result()
  def encrypt(plaintext, key_id, aad, correlation_id) do
    start_time = System.monotonic_time(:millisecond)
    
    request = %EncryptRequest{
      plaintext: plaintext,
      key_id: to_proto_key_id(key_id),
      aad: aad,
      correlation_id: correlation_id
    }

    result = with_resilience(:encrypt, correlation_id, fn channel ->
      CryptoService.Stub.encrypt(channel, request, metadata: build_metadata(correlation_id))
    end)

    duration = System.monotonic_time(:millisecond) - start_time
    Telemetry.emit_rpc_call(:encrypt, result, duration, correlation_id)

    case result do
      {:ok, response} ->
        {:ok, %{
          ciphertext: response.ciphertext,
          iv: response.iv,
          tag: response.tag,
          key_id: from_proto_key_id(response.key_id),
          algorithm: response.algorithm
        }}

      {:error, reason} ->
        {:error, Error.new(:encryption_failed, reason, correlation_id)}
    end
  end

  @doc """
  Decrypts ciphertext using AES-256-GCM via Crypto Service.
  
  ## Parameters
    - ciphertext: The encrypted data
    - iv: Initialization vector
    - tag: Authentication tag
    - key_id: The key identifier used for encryption
    - aad: Additional Authenticated Data (must match encryption AAD)
    - correlation_id: Request correlation ID for tracing
  """
  @spec decrypt(binary(), binary(), binary(), key_id(), binary(), String.t()) :: 
    decrypt_result() | error_result()
  def decrypt(ciphertext, iv, tag, key_id, aad, correlation_id) do
    start_time = System.monotonic_time(:millisecond)
    
    request = %DecryptRequest{
      ciphertext: ciphertext,
      iv: iv,
      tag: tag,
      key_id: to_proto_key_id(key_id),
      aad: aad,
      correlation_id: correlation_id
    }

    result = with_resilience(:decrypt, correlation_id, fn channel ->
      CryptoService.Stub.decrypt(channel, request, metadata: build_metadata(correlation_id))
    end)

    duration = System.monotonic_time(:millisecond) - start_time
    Telemetry.emit_rpc_call(:decrypt, result, duration, correlation_id)

    case result do
      {:ok, response} ->
        {:ok, response.plaintext}

      {:error, reason} ->
        {:error, Error.new(:decryption_failed, reason, correlation_id)}
    end
  end

  @doc """
  Checks the health of the Crypto Service.
  """
  @spec health_check() :: {:ok, :serving} | {:ok, :not_serving} | error_result()
  def health_check do
    correlation_id = generate_correlation_id()
    
    result = with_resilience(:health_check, correlation_id, fn channel ->
      CryptoService.Stub.health_check(channel, %HealthCheckRequest{}, 
        metadata: build_metadata(correlation_id))
    end)

    case result do
      {:ok, response} ->
        case response.status do
          :SERVING -> {:ok, :serving}
          :NOT_SERVING -> {:ok, :not_serving}
          _ -> {:ok, :unknown}
        end

      {:error, reason} ->
        {:error, Error.new(:health_check_failed, reason, correlation_id)}
    end
  end

  @doc """
  Generates a new encryption key in Crypto Service.
  """
  @spec generate_key(String.t(), map(), String.t()) :: {:ok, key_id()} | error_result()
  def generate_key(namespace, metadata \\ %{}, correlation_id) do
    start_time = System.monotonic_time(:millisecond)
    
    request = %GenerateKeyRequest{
      algorithm: KeyAlgorithm.key(:AES_256_GCM),
      namespace: namespace,
      metadata: metadata,
      correlation_id: correlation_id
    }

    result = with_resilience(:generate_key, correlation_id, fn channel ->
      CryptoService.Stub.generate_key(channel, request, metadata: build_metadata(correlation_id))
    end)

    duration = System.monotonic_time(:millisecond) - start_time
    Telemetry.emit_rpc_call(:generate_key, result, duration, correlation_id)

    case result do
      {:ok, response} ->
        {:ok, from_proto_key_id(response.key_id)}

      {:error, reason} ->
        {:error, Error.new(:key_generation_failed, reason, correlation_id)}
    end
  end

  @doc """
  Rotates an existing encryption key.
  """
  @spec rotate_key(key_id(), String.t()) :: {:ok, key_id()} | error_result()
  def rotate_key(key_id, correlation_id) do
    start_time = System.monotonic_time(:millisecond)
    
    request = %RotateKeyRequest{
      key_id: to_proto_key_id(key_id),
      correlation_id: correlation_id
    }

    result = with_resilience(:rotate_key, correlation_id, fn channel ->
      CryptoService.Stub.rotate_key(channel, request, metadata: build_metadata(correlation_id))
    end)

    duration = System.monotonic_time(:millisecond) - start_time
    Telemetry.emit_rpc_call(:rotate_key, result, duration, correlation_id)

    case result do
      {:ok, response} ->
        {:ok, from_proto_key_id(response.new_key_id)}

      {:error, reason} ->
        {:error, Error.new(:key_rotation_failed, reason, correlation_id)}
    end
  end

  @doc """
  Gets metadata for a key.
  """
  @spec get_key_metadata(key_id(), String.t()) :: {:ok, map()} | error_result()
  def get_key_metadata(key_id, correlation_id) do
    start_time = System.monotonic_time(:millisecond)
    
    request = %GetKeyMetadataRequest{
      key_id: to_proto_key_id(key_id),
      correlation_id: correlation_id
    }

    result = with_resilience(:get_key_metadata, correlation_id, fn channel ->
      CryptoService.Stub.get_key_metadata(channel, request, metadata: build_metadata(correlation_id))
    end)

    duration = System.monotonic_time(:millisecond) - start_time
    Telemetry.emit_rpc_call(:get_key_metadata, result, duration, correlation_id)

    case result do
      {:ok, response} ->
        {:ok, from_proto_metadata(response.metadata)}

      {:error, reason} ->
        {:error, Error.new(:key_metadata_failed, reason, correlation_id)}
    end
  end

  # GenServer callbacks

  @impl true
  def init(_opts) do
    state = %{
      channel: nil,
      circuit_breaker: %{
        state: :closed,
        failure_count: 0,
        last_failure_time: nil,
        threshold: Config.circuit_breaker_threshold(),
        reset_timeout: Config.circuit_breaker_reset_timeout()
      }
    }

    {:ok, state, {:continue, :connect}}
  end

  @impl true
  def handle_continue(:connect, state) do
    case connect() do
      {:ok, channel} ->
        Logger.info("Connected to Crypto Service at #{Config.address()}")
        {:noreply, %{state | channel: channel}}

      {:error, reason} ->
        Logger.warning("Failed to connect to Crypto Service: #{inspect(reason)}")
        Process.send_after(self(), :reconnect, 5_000)
        {:noreply, state}
    end
  end

  @impl true
  def handle_info(:reconnect, state) do
    {:noreply, state, {:continue, :connect}}
  end

  @impl true
  def handle_call({:execute, operation, correlation_id, fun}, _from, state) do
    case check_circuit_breaker(state) do
      {:ok, state} ->
        case state.channel do
          nil ->
            {:reply, {:error, :not_connected}, state}

          channel ->
            case fun.(channel) do
              {:ok, _} = result ->
                new_state = record_success(state)
                {:reply, result, new_state}

              {:error, _} = error ->
                new_state = record_failure(state, operation, correlation_id)
                {:reply, error, new_state}
            end
        end

      {:error, :circuit_open} = error ->
        Telemetry.emit_circuit_breaker_rejection(operation, correlation_id)
        {:reply, error, state}
    end
  end

  # Private functions

  defp connect do
    opts = if Config.mtls_enabled?() do
      cred = GRPC.Credential.new(
        ssl: [
          certfile: Config.tls_cert_path(),
          keyfile: Config.tls_key_path(),
          cacertfile: Config.tls_ca_path()
        ]
      )
      [cred: cred]
    else
      []
    end

    GRPC.Stub.connect(Config.address(), opts)
  end

  defp with_resilience(operation, correlation_id, fun) do
    GenServer.call(__MODULE__, {:execute, operation, correlation_id, fun}, 
      Config.request_timeout())
  rescue
    e ->
      Logger.error("Crypto Service call failed: #{inspect(e)}", 
        correlation_id: correlation_id, operation: operation)
      {:error, e}
  end

  defp check_circuit_breaker(%{circuit_breaker: cb} = state) do
    case cb.state do
      :closed ->
        {:ok, state}

      :open ->
        if should_attempt_reset?(cb) do
          {:ok, put_in(state.circuit_breaker.state, :half_open)}
        else
          {:error, :circuit_open}
        end

      :half_open ->
        {:ok, state}
    end
  end

  defp should_attempt_reset?(cb) do
    case cb.last_failure_time do
      nil -> true
      time -> System.monotonic_time(:millisecond) - time > cb.reset_timeout
    end
  end

  defp record_success(state) do
    new_cb = %{state.circuit_breaker | 
      state: :closed, 
      failure_count: 0,
      last_failure_time: nil
    }
    
    if state.circuit_breaker.state != :closed do
      Telemetry.emit_circuit_breaker_state_change(:closed)
    end
    
    %{state | circuit_breaker: new_cb}
  end

  defp record_failure(state, operation, correlation_id) do
    cb = state.circuit_breaker
    new_count = cb.failure_count + 1
    
    new_state = if new_count >= cb.threshold do
      Telemetry.emit_circuit_breaker_state_change(:open)
      Logger.warning("Circuit breaker opened after #{new_count} failures",
        operation: operation, correlation_id: correlation_id)
      :open
    else
      cb.state
    end

    new_cb = %{cb | 
      state: new_state, 
      failure_count: new_count,
      last_failure_time: System.monotonic_time(:millisecond)
    }
    
    %{state | circuit_breaker: new_cb}
  end

  defp to_proto_key_id(%{namespace: ns, id: id, version: v}) do
    %ProtoKeyId{namespace: ns, id: id, version: v}
  end

  defp from_proto_key_id(%ProtoKeyId{namespace: ns, id: id, version: v}) do
    %{namespace: ns, id: id, version: v}
  end

  defp from_proto_key_id(nil), do: nil

  defp from_proto_metadata(nil), do: %{}
  defp from_proto_metadata(metadata) do
    %{
      id: from_proto_key_id(metadata.id),
      algorithm: metadata.algorithm,
      state: metadata.state,
      created_at: DateTime.from_unix!(metadata.created_at),
      expires_at: if(metadata.expires_at > 0, do: DateTime.from_unix!(metadata.expires_at)),
      rotated_at: if(metadata.rotated_at > 0, do: DateTime.from_unix!(metadata.rotated_at)),
      previous_version: from_proto_key_id(metadata.previous_version),
      owner_service: metadata.owner_service,
      allowed_operations: metadata.allowed_operations,
      usage_count: metadata.usage_count
    }
  end

  defp build_metadata(correlation_id) do
    %{"x-correlation-id" => correlation_id}
  end

  defp generate_correlation_id do
    UUID.uuid4()
  end
end
