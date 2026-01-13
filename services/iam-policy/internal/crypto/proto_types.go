// Code generated from crypto_service.proto. DO NOT EDIT.
// Package crypto provides typed wrappers for crypto-service protobuf types.
package crypto

// NOTE: This file contains manual Go type definitions matching the proto definitions
// until proper code generation is set up. To generate from proto:
//   protoc --go_out=. --go-grpc_out=. services/crypto-service/proto/crypto_service.proto

// KeyIdProto matches the protobuf KeyId message.
type KeyIdProto struct {
	Namespace string
	Id        string
	Version   uint32
}

// ToKeyID converts a KeyID to protobuf format.
func (k KeyID) ToProto() *KeyIdProto {
	return &KeyIdProto{
		Namespace: k.Namespace,
		Id:        k.ID,
		Version:   k.Version,
	}
}

// FromProto converts from protobuf format to KeyID.
func (k *KeyIdProto) ToKeyID() KeyID {
	if k == nil {
		return KeyID{}
	}
	return KeyID{
		Namespace: k.Namespace,
		ID:        k.Id,
		Version:   k.Version,
	}
}

// HashAlgorithm enum from protobuf.
type HashAlgorithm int32

const (
	HashAlgorithmUnspecified HashAlgorithm = 0
	HashAlgorithmSHA256      HashAlgorithm = 1
	HashAlgorithmSHA384      HashAlgorithm = 2
	HashAlgorithmSHA512      HashAlgorithm = 3
)
