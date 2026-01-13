defmodule Crypto.V1.KeyId do
  @moduledoc false
  use Protobuf, protoc_gen_elixir_version: "0.12.0", syntax: :proto3

  field :namespace, 1, type: :string
  field :id, 2, type: :string
  field :version, 3, type: :uint32
end

defmodule Crypto.V1.KeyAlgorithm do
  @moduledoc false
  use Protobuf, enum: true, protoc_gen_elixir_version: "0.12.0", syntax: :proto3

  field :KEY_ALGORITHM_UNSPECIFIED, 0
  field :AES_128_GCM, 1
  field :AES_256_GCM, 2
  field :AES_128_CBC, 3
  field :AES_256_CBC, 4
  field :RSA_2048, 5
  field :RSA_3072, 6
  field :RSA_4096, 7
  field :ECDSA_P256, 8
  field :ECDSA_P384, 9
  field :ECDSA_P521, 10
end

defmodule Crypto.V1.KeyState do
  @moduledoc false
  use Protobuf, enum: true, protoc_gen_elixir_version: "0.12.0", syntax: :proto3

  field :KEY_STATE_UNSPECIFIED, 0
  field :PENDING_ACTIVATION, 1
  field :ACTIVE, 2
  field :DEPRECATED, 3
  field :PENDING_DESTRUCTION, 4
  field :DESTROYED, 5
end

defmodule Crypto.V1.HashAlgorithm do
  @moduledoc false
  use Protobuf, enum: true, protoc_gen_elixir_version: "0.12.0", syntax: :proto3

  field :HASH_ALGORITHM_UNSPECIFIED, 0
  field :SHA256, 1
  field :SHA384, 2
  field :SHA512, 3
end

defmodule Crypto.V1.EncryptRequest do
  @moduledoc false
  use Protobuf, protoc_gen_elixir_version: "0.12.0", syntax: :proto3

  field :plaintext, 1, type: :bytes
  field :key_id, 2, type: Crypto.V1.KeyId, json_name: "keyId"
  field :aad, 3, type: :bytes
  field :correlation_id, 4, type: :string, json_name: "correlationId"
end

defmodule Crypto.V1.EncryptResponse do
  @moduledoc false
  use Protobuf, protoc_gen_elixir_version: "0.12.0", syntax: :proto3

  field :ciphertext, 1, type: :bytes
  field :iv, 2, type: :bytes
  field :tag, 3, type: :bytes
  field :key_id, 4, type: Crypto.V1.KeyId, json_name: "keyId"
  field :algorithm, 5, type: :string
end


defmodule Crypto.V1.DecryptRequest do
  @moduledoc false
  use Protobuf, protoc_gen_elixir_version: "0.12.0", syntax: :proto3

  field :ciphertext, 1, type: :bytes
  field :iv, 2, type: :bytes
  field :tag, 3, type: :bytes
  field :key_id, 4, type: Crypto.V1.KeyId, json_name: "keyId"
  field :aad, 5, type: :bytes
  field :correlation_id, 6, type: :string, json_name: "correlationId"
end

defmodule Crypto.V1.DecryptResponse do
  @moduledoc false
  use Protobuf, protoc_gen_elixir_version: "0.12.0", syntax: :proto3

  field :plaintext, 1, type: :bytes
end

defmodule Crypto.V1.SignRequest do
  @moduledoc false
  use Protobuf, protoc_gen_elixir_version: "0.12.0", syntax: :proto3

  field :data, 1, type: :bytes
  field :key_id, 2, type: Crypto.V1.KeyId, json_name: "keyId"
  field :hash_algorithm, 3, type: Crypto.V1.HashAlgorithm, json_name: "hashAlgorithm", enum: true
  field :correlation_id, 4, type: :string, json_name: "correlationId"
end

defmodule Crypto.V1.SignResponse do
  @moduledoc false
  use Protobuf, protoc_gen_elixir_version: "0.12.0", syntax: :proto3

  field :signature, 1, type: :bytes
  field :key_id, 2, type: Crypto.V1.KeyId, json_name: "keyId"
  field :algorithm, 3, type: :string
end

defmodule Crypto.V1.VerifyRequest do
  @moduledoc false
  use Protobuf, protoc_gen_elixir_version: "0.12.0", syntax: :proto3

  field :data, 1, type: :bytes
  field :signature, 2, type: :bytes
  field :key_id, 3, type: Crypto.V1.KeyId, json_name: "keyId"
  field :hash_algorithm, 4, type: Crypto.V1.HashAlgorithm, json_name: "hashAlgorithm", enum: true
  field :correlation_id, 5, type: :string, json_name: "correlationId"
end

defmodule Crypto.V1.VerifyResponse do
  @moduledoc false
  use Protobuf, protoc_gen_elixir_version: "0.12.0", syntax: :proto3

  field :valid, 1, type: :bool
  field :key_id, 2, type: Crypto.V1.KeyId, json_name: "keyId"
end

defmodule Crypto.V1.GetKeyMetadataRequest do
  @moduledoc false
  use Protobuf, protoc_gen_elixir_version: "0.12.0", syntax: :proto3

  field :key_id, 1, type: Crypto.V1.KeyId, json_name: "keyId"
  field :correlation_id, 2, type: :string, json_name: "correlationId"
end

defmodule Crypto.V1.KeyMetadata do
  @moduledoc false
  use Protobuf, protoc_gen_elixir_version: "0.12.0", syntax: :proto3

  field :id, 1, type: Crypto.V1.KeyId
  field :algorithm, 2, type: Crypto.V1.KeyAlgorithm, enum: true
  field :state, 3, type: Crypto.V1.KeyState, enum: true
  field :created_at, 4, type: :int64, json_name: "createdAt"
  field :expires_at, 5, type: :int64, json_name: "expiresAt"
  field :rotated_at, 6, type: :int64, json_name: "rotatedAt"
  field :previous_version, 7, type: Crypto.V1.KeyId, json_name: "previousVersion"
  field :owner_service, 8, type: :string, json_name: "ownerService"
  field :allowed_operations, 9, repeated: true, type: :string, json_name: "allowedOperations"
  field :usage_count, 10, type: :uint64, json_name: "usageCount"
end

defmodule Crypto.V1.GetKeyMetadataResponse do
  @moduledoc false
  use Protobuf, protoc_gen_elixir_version: "0.12.0", syntax: :proto3

  field :metadata, 1, type: Crypto.V1.KeyMetadata
end

defmodule Crypto.V1.HealthCheckRequest do
  @moduledoc false
  use Protobuf, protoc_gen_elixir_version: "0.12.0", syntax: :proto3
end

defmodule Crypto.V1.HealthCheckResponse.ServingStatus do
  @moduledoc false
  use Protobuf, enum: true, protoc_gen_elixir_version: "0.12.0", syntax: :proto3

  field :UNKNOWN, 0
  field :SERVING, 1
  field :NOT_SERVING, 2
end

defmodule Crypto.V1.HealthCheckResponse do
  @moduledoc false
  use Protobuf, protoc_gen_elixir_version: "0.12.0", syntax: :proto3

  field :status, 1, type: Crypto.V1.HealthCheckResponse.ServingStatus, enum: true
  field :hsm_connected, 2, type: :bool, json_name: "hsmConnected"
  field :kms_connected, 3, type: :bool, json_name: "kmsConnected"
  field :version, 4, type: :string
  field :uptime_seconds, 5, type: :int64, json_name: "uptimeSeconds"
end

defmodule Crypto.V1.CryptoService.Service do
  @moduledoc false
  use GRPC.Service, name: "crypto.v1.CryptoService", protoc_gen_elixir_version: "0.12.0"

  rpc :Encrypt, Crypto.V1.EncryptRequest, Crypto.V1.EncryptResponse
  rpc :Decrypt, Crypto.V1.DecryptRequest, Crypto.V1.DecryptResponse
  rpc :Sign, Crypto.V1.SignRequest, Crypto.V1.SignResponse
  rpc :Verify, Crypto.V1.VerifyRequest, Crypto.V1.VerifyResponse
  rpc :GetKeyMetadata, Crypto.V1.GetKeyMetadataRequest, Crypto.V1.GetKeyMetadataResponse
  rpc :HealthCheck, Crypto.V1.HealthCheckRequest, Crypto.V1.HealthCheckResponse
end

defmodule Crypto.V1.CryptoService.Stub do
  @moduledoc false
  use GRPC.Stub, service: Crypto.V1.CryptoService.Service
end
