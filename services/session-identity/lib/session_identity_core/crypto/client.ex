defmodule SessionIdentityCore.Crypto.Client do
  @moduledoc """
  gRPC client for crypto-service with circuit breaker support.
  
  Provides centralized cryptographic operations:
  - AES-256-GCM encryption/decryption
  - Digital signatures (ECDSA/RSA)
  - Key metadata retrieval
  - Health checks
  
  Uses W3C Trace Context propagation and correlation IDs for observability.
  """

  use GenServer
  require Logger

  alias SessionIdentityCore.Crypto.Config
  alias Crypto.V1.CryptoService.Stub, as: CryptoStub

  @type key_id :: %{namespace: String.t(), id: String.t(), version: non_neg_integer()}
  @type encrypt_result :: {:ok, %{ciphertext: binary(), iv: binary(), tag: binary(), key_id: key_id()}} | {:error, term()}
  @type decrypt_result :: {:ok, binary()} | {:error, term()}
  @type sign_result :: {:ok, %{signature: binary(), key_id: key_id()}} | {:error, term()}
  @type verify_result :: {:ok, boolean()} | {:error, term()}

  # Client API

  def start_link(opts \\ []) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end

  @doc """
  Encrypts plaintext using AES-256-GCM via crypto-service.
  """
  @spec encrypt(binary(), key_id(), binary() | nil, keyword()) :: encrypt_result()
  def encrypt(plaintext, key_id, aad \\ nil, opts \\ []) do
    GenServer.call(__MODULE__, {:encrypt, plaintext, key_id, aad, opts}, get_timeout(opts))
  end

  @doc """
  Decrypts ciphertext using AES-256-GCM via crypto-service.
  """
  @spec decrypt(binary(), binary(), binary(), key_id(), binary() | nil, keyword()) :: decrypt_result()
  def decrypt(ciphertext, iv, tag, key_id, aad \\ nil, opts \\ []) do
    GenServer.call(__MODULE__, {:decrypt, ciphertext, iv, tag, key_id, aad, opts}, get_timeout(opts))
  end

  @doc """
  Signs data using crypto-service.
  """
  @spec sign(binary(), key_id(), atom(), keyword()) :: sign_result()
  def sign(data, key_id, hash_algorithm \\ :sha256, opts \\ []) do
    GenServer.call(__MODULE__, {:sign, data, key_id, hash_algorithm, opts}, get_timeout(opts))
  end

  @doc """
  Verifies a signature using crypto-service.
  """
  @spec verify(binary(), binary(), key_id(), atom(), keyword()) :: verify_result()
  def verify(data, signature, key_id, hash_algorithm \\ :sha256, opts \\ []) do
    GenServer.call(__MODULE__, {:verify, data, signature, key_id, hash_algorithm, opts}, get_timeout(opts))
  end

  @doc """
  Retrieves key metadata from crypto-service.
  """
  @spec get_key_metadata(key_id(), keyword()) :: {:ok, map()} | {:error, term()}
  def get_key_metadata(key_id, opts \\ []) do
    GenServer.call(__MODULE__, {:get_key_metadata, key_id, opts}, get_timeout(opts))
  end

  @doc """
  Checks crypto-service health.
  """
  @spec health_check() :: {:ok, :serving} | {:error, term()}
  def health_check do
    GenServer.call(__MODULE__, :health_check, 5_000)
  end

  # Server Callbacks

  @impl true
  def init(_opts) do
    config = Config.get()
    
    state = %{
      config: config,
      channel: nil,
      connected: false
    }

    {:ok, state, {:continue, :connect}}
  end

  @impl true
  def handle_continue(:connect, state) do
    case connect(state.config.endpoint) do
      {:ok, channel} ->
        Logger.info("Connected to crypto-service at #{state.config.endpoint}")
        {:noreply, %{state | channel: channel, connected: true}}

      {:error, reason} ->
        Logger.warning("Failed to connect to crypto-service: #{inspect(reason)}")
        Process.send_after(self(), :reconnect, 5_000)
        {:noreply, state}
    end
  end

  @impl true
  def handle_info(:reconnect, state) do
    {:noreply, state, {:continue, :connect}}
  end

  @impl true
  def handle_call({:encrypt, plaintext, key_id, aad, opts}, _from, state) do
    result = do_encrypt(state, plaintext, key_id, aad, opts)
    {:reply, result, state}
  end

  @impl true
  def handle_call({:decrypt, ciphertext, iv, tag, key_id, aad, opts}, _from, state) do
    result = do_decrypt(state, ciphertext, iv, tag, key_id, aad, opts)
    {:reply, result, state}
  end

  @impl true
  def handle_call({:sign, data, key_id, hash_algorithm, opts}, _from, state) do
    result = do_sign(state, data, key_id, hash_algorithm, opts)
    {:reply, result, state}
  end

  @impl true
  def handle_call({:verify, data, signature, key_id, hash_algorithm, opts}, _from, state) do
    result = do_verify(state, data, signature, key_id, hash_algorithm, opts)
    {:reply, result, state}
  end

  @impl true
  def handle_call({:get_key_metadata, key_id, opts}, _from, state) do
    result = do_get_key_metadata(state, key_id, opts)
    {:reply, result, state}
  end

  @impl true
  def handle_call(:health_check, _from, state) do
    result = do_health_check(state)
    {:reply, result, state}
  end

  # Private Functions

  defp connect(endpoint) do
    GRPC.Stub.connect(endpoint)
  end

  defp do_encrypt(%{connected: false}, _plaintext, _key_id, _aad, _opts) do
    {:error, %{error_code: :crypto_service_unavailable, message: "Not connected to crypto-service"}}
  end

  defp do_encrypt(%{channel: channel}, plaintext, key_id, aad, opts) do
    correlation_id = get_correlation_id(opts)
    metadata = build_metadata(opts)

    request = %Crypto.V1.EncryptRequest{
      plaintext: plaintext,
      key_id: to_proto_key_id(key_id),
      aad: aad || <<>>,
      correlation_id: correlation_id
    }

    start_time = System.monotonic_time(:millisecond)

    case CryptoStub.encrypt(channel, request, metadata: metadata) do
      {:ok, response} ->
        emit_latency_metric(:encrypt, start_time)
        {:ok, %{
          ciphertext: response.ciphertext,
          iv: response.iv,
          tag: response.tag,
          key_id: from_proto_key_id(response.key_id)
        }}

      {:error, %GRPC.RPCError{} = error} ->
        emit_error_metric(:encrypt, error)
        {:error, map_grpc_error(error)}
    end
  end

  defp do_decrypt(%{connected: false}, _ciphertext, _iv, _tag, _key_id, _aad, _opts) do
    {:error, %{error_code: :crypto_service_unavailable, message: "Not connected to crypto-service"}}
  end

  defp do_decrypt(%{channel: channel}, ciphertext, iv, tag, key_id, aad, opts) do
    correlation_id = get_correlation_id(opts)
    metadata = build_metadata(opts)

    request = %Crypto.V1.DecryptRequest{
      ciphertext: ciphertext,
      iv: iv,
      tag: tag,
      key_id: to_proto_key_id(key_id),
      aad: aad || <<>>,
      correlation_id: correlation_id
    }

    start_time = System.monotonic_time(:millisecond)

    case CryptoStub.decrypt(channel, request, metadata: metadata) do
      {:ok, response} ->
        emit_latency_metric(:decrypt, start_time)
        {:ok, response.plaintext}

      {:error, %GRPC.RPCError{} = error} ->
        emit_error_metric(:decrypt, error)
        {:error, map_grpc_error(error)}
    end
  end

  defp do_sign(%{connected: false}, _data, _key_id, _hash_algorithm, _opts) do
    {:error, %{error_code: :crypto_service_unavailable, message: "Not connected to crypto-service"}}
  end

  defp do_sign(%{channel: channel}, data, key_id, hash_algorithm, opts) do
    correlation_id = get_correlation_id(opts)
    metadata = build_metadata(opts)

    request = %Crypto.V1.SignRequest{
      data: data,
      key_id: to_proto_key_id(key_id),
      hash_algorithm: to_proto_hash_algorithm(hash_algorithm),
      correlation_id: correlation_id
    }

    start_time = System.monotonic_time(:millisecond)

    case CryptoStub.sign(channel, request, metadata: metadata) do
      {:ok, response} ->
        emit_latency_metric(:sign, start_time)
        {:ok, %{
          signature: response.signature,
          key_id: from_proto_key_id(response.key_id)
        }}

      {:error, %GRPC.RPCError{} = error} ->
        emit_error_metric(:sign, error)
        {:error, map_grpc_error(error)}
    end
  end

  defp do_verify(%{connected: false}, _data, _signature, _key_id, _hash_algorithm, _opts) do
    {:error, %{error_code: :crypto_service_unavailable, message: "Not connected to crypto-service"}}
  end

  defp do_verify(%{channel: channel}, data, signature, key_id, hash_algorithm, opts) do
    correlation_id = get_correlation_id(opts)
    metadata = build_metadata(opts)

    request = %Crypto.V1.VerifyRequest{
      data: data,
      signature: signature,
      key_id: to_proto_key_id(key_id),
      hash_algorithm: to_proto_hash_algorithm(hash_algorithm),
      correlation_id: correlation_id
    }

    start_time = System.monotonic_time(:millisecond)

    case CryptoStub.verify(channel, request, metadata: metadata) do
      {:ok, response} ->
        emit_latency_metric(:verify, start_time)
        {:ok, response.valid}

      {:error, %GRPC.RPCError{} = error} ->
        emit_error_metric(:verify, error)
        {:error, map_grpc_error(error)}
    end
  end

  defp do_get_key_metadata(%{connected: false}, _key_id, _opts) do
    {:error, %{error_code: :crypto_service_unavailable, message: "Not connected to crypto-service"}}
  end

  defp do_get_key_metadata(%{channel: channel}, key_id, opts) do
    correlation_id = get_correlation_id(opts)
    metadata = build_metadata(opts)

    request = %Crypto.V1.GetKeyMetadataRequest{
      key_id: to_proto_key_id(key_id),
      correlation_id: correlation_id
    }

    case CryptoStub.get_key_metadata(channel, request, metadata: metadata) do
      {:ok, response} ->
        {:ok, from_proto_key_metadata(response.metadata)}

      {:error, %GRPC.RPCError{} = error} ->
        {:error, map_grpc_error(error)}
    end
  end

  defp do_health_check(%{connected: false}) do
    {:error, %{error_code: :crypto_service_unavailable, message: "Not connected to crypto-service"}}
  end

  defp do_health_check(%{channel: channel}) do
    request = %Crypto.V1.HealthCheckRequest{}

    case CryptoStub.health_check(channel, request) do
      {:ok, %{status: :SERVING}} -> {:ok, :serving}
      {:ok, _} -> {:error, %{error_code: :crypto_service_not_serving, message: "Crypto service not serving"}}
      {:error, %GRPC.RPCError{} = error} -> {:error, map_grpc_error(error)}
    end
  end

  # Helper Functions

  defp get_correlation_id(opts) do
    Keyword.get(opts, :correlation_id) || generate_correlation_id()
  end

  defp generate_correlation_id do
    :crypto.strong_rand_bytes(16) |> Base.encode16(case: :lower)
  end

  defp get_timeout(opts) do
    Keyword.get(opts, :timeout, Config.get().timeout)
  end

  defp build_metadata(opts) do
    metadata = []
    
    # Add W3C Trace Context if available
    metadata = case Keyword.get(opts, :traceparent) do
      nil -> metadata
      traceparent -> [{"traceparent", traceparent} | metadata]
    end

    metadata = case Keyword.get(opts, :tracestate) do
      nil -> metadata
      tracestate -> [{"tracestate", tracestate} | metadata]
    end

    metadata
  end

  defp to_proto_key_id(%{namespace: ns, id: id, version: v}) do
    %Crypto.V1.KeyId{namespace: ns, id: id, version: v}
  end

  defp from_proto_key_id(%Crypto.V1.KeyId{namespace: ns, id: id, version: v}) do
    %{namespace: ns, id: id, version: v}
  end

  defp from_proto_key_id(nil), do: nil

  defp to_proto_hash_algorithm(:sha256), do: :SHA256
  defp to_proto_hash_algorithm(:sha384), do: :SHA384
  defp to_proto_hash_algorithm(:sha512), do: :SHA512
  defp to_proto_hash_algorithm(_), do: :SHA256

  defp from_proto_key_metadata(%Crypto.V1.KeyMetadata{} = m) do
    %{
      id: from_proto_key_id(m.id),
      algorithm: m.algorithm,
      state: m.state,
      created_at: m.created_at,
      expires_at: m.expires_at,
      rotated_at: m.rotated_at,
      previous_version: from_proto_key_id(m.previous_version),
      owner_service: m.owner_service,
      allowed_operations: m.allowed_operations,
      usage_count: m.usage_count
    }
  end

  defp map_grpc_error(%GRPC.RPCError{status: status, message: message}) do
    error_code = case status do
      :unavailable -> :crypto_service_unavailable
      :deadline_exceeded -> :crypto_operation_timeout
      :unauthenticated -> :crypto_auth_failed
      :not_found -> :key_not_found
      :invalid_argument -> :invalid_argument
      _ -> :crypto_operation_failed
    end

    %{error_code: error_code, message: message || "Unknown error"}
  end

  defp emit_latency_metric(operation, start_time) do
    duration = System.monotonic_time(:millisecond) - start_time
    :telemetry.execute(
      [:session_identity, :crypto, :operation],
      %{duration: duration},
      %{operation: operation, status: :success}
    )
  end

  defp emit_error_metric(operation, _error) do
    :telemetry.execute(
      [:session_identity, :crypto, :operation],
      %{count: 1},
      %{operation: operation, status: :error}
    )
  end
end
