// Package crypto provides gRPC client implementation for crypto-service.
package crypto

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CryptoServiceClient is the interface for crypto-service operations.
// This is a minimal implementation matching the proto service definition.
// TODO: Replace with properly generated protobuf code using:
//   protoc --go_out=. --go-grpc_out=. services/crypto-service/proto/crypto_service.proto
type CryptoServiceClient interface {
	Encrypt(ctx context.Context, req *EncryptRequestProto) (*EncryptResponseProto, error)
	Decrypt(ctx context.Context, req *DecryptRequestProto) (*DecryptResponseProto, error)
	Sign(ctx context.Context, req *SignRequestProto) (*SignResponseProto, error)
	Verify(ctx context.Context, req *VerifyRequestProto) (*VerifyResponseProto, error)
	HealthCheck(ctx context.Context, req *HealthCheckRequestProto) (*HealthCheckResponseProto, error)
}

// Protobuf request/response types matching crypto_service.proto

type EncryptRequestProto struct {
	Plaintext     []byte
	KeyId         *KeyIdProto
	Aad           []byte
	CorrelationId string
}

type EncryptResponseProto struct {
	Ciphertext []byte
	Iv         []byte
	Tag        []byte
	KeyId      *KeyIdProto
	Algorithm  string
}

type DecryptRequestProto struct {
	Ciphertext    []byte
	Iv            []byte
	Tag           []byte
	KeyId         *KeyIdProto
	Aad           []byte
	CorrelationId string
}

type DecryptResponseProto struct {
	Plaintext []byte
}

type SignRequestProto struct {
	Data          []byte
	KeyId         *KeyIdProto
	HashAlgorithm HashAlgorithm
	CorrelationId string
}

type SignResponseProto struct {
	Signature []byte
	KeyId     *KeyIdProto
	Algorithm string
}

type VerifyRequestProto struct {
	Data          []byte
	Signature     []byte
	KeyId         *KeyIdProto
	HashAlgorithm HashAlgorithm
	CorrelationId string
}

type VerifyResponseProto struct {
	Valid bool
	KeyId *KeyIdProto
}

type HealthCheckRequestProto struct{}

type HealthCheckResponseProto struct {
	Status        ServingStatus
	HsmConnected  bool
	KmsConnected  bool
	Version       string
	UptimeSeconds int64
}

type ServingStatus int32

const (
	ServingStatusUnknown    ServingStatus = 0
	ServingStatusServing    ServingStatus = 1
	ServingStatusNotServing ServingStatus = 2
)

// cryptoServiceClientImpl implements CryptoServiceClient using gRPC.
type cryptoServiceClientImpl struct {
	// TODO: Add actual gRPC client connection field
	// conn *grpc.ClientConn
	// For now, we'll use JSON-RPC style calls over gRPC
}

// newCryptoServiceClient creates a new crypto service client.
// TODO: This should use the generated protobuf client:
//   pb.NewCryptoServiceClient(conn)
func newCryptoServiceClient() CryptoServiceClient {
	return &cryptoServiceClientImpl{}
}

// Encrypt calls the crypto-service Encrypt RPC.
func (c *cryptoServiceClientImpl) Encrypt(ctx context.Context, req *EncryptRequestProto) (*EncryptResponseProto, error) {
	// TODO: Replace with actual gRPC call:
	// resp, err := c.client.Encrypt(ctx, &pb.EncryptRequest{
	//     Plaintext: req.Plaintext,
	//     KeyId: &pb.KeyId{
	//         Namespace: req.KeyId.Namespace,
	//         Id: req.KeyId.Id,
	//         Version: req.KeyId.Version,
	//     },
	//     Aad: req.Aad,
	//     CorrelationId: req.CorrelationId,
	// })
	// if err != nil {
	//     return nil, err
	// }
	// return &EncryptResponseProto{
	//     Ciphertext: resp.Ciphertext,
	//     Iv: resp.Iv,
	//     Tag: resp.Tag,
	//     KeyId: &KeyIdProto{
	//         Namespace: resp.KeyId.Namespace,
	//         Id: resp.KeyId.Id,
	//         Version: resp.KeyId.Version,
	//     },
	//     Algorithm: resp.Algorithm,
	// }, nil

	// Temporary: Return unimplemented error
	return nil, status.Error(codes.Unimplemented, "Encrypt RPC not yet implemented - awaiting protobuf code generation")
}

// Decrypt calls the crypto-service Decrypt RPC.
func (c *cryptoServiceClientImpl) Decrypt(ctx context.Context, req *DecryptRequestProto) (*DecryptResponseProto, error) {
	// TODO: Replace with actual gRPC call (see Encrypt for example)
	return nil, status.Error(codes.Unimplemented, "Decrypt RPC not yet implemented - awaiting protobuf code generation")
}

// Sign calls the crypto-service Sign RPC.
func (c *cryptoServiceClientImpl) Sign(ctx context.Context, req *SignRequestProto) (*SignResponseProto, error) {
	// TODO: Replace with actual gRPC call (see Encrypt for example)
	return nil, status.Error(codes.Unimplemented, "Sign RPC not yet implemented - awaiting protobuf code generation")
}

// Verify calls the crypto-service Verify RPC.
func (c *cryptoServiceClientImpl) Verify(ctx context.Context, req *VerifyRequestProto) (*VerifyResponseProto, error) {
	// TODO: Replace with actual gRPC call (see Encrypt for example)
	return nil, status.Error(codes.Unimplemented, "Verify RPC not yet implemented - awaiting protobuf code generation")
}

// HealthCheck calls the crypto-service HealthCheck RPC.
func (c *cryptoServiceClientImpl) HealthCheck(ctx context.Context, req *HealthCheckRequestProto) (*HealthCheckResponseProto, error) {
	// TODO: Replace with actual gRPC call (see Encrypt for example)
	return nil, status.Error(codes.Unimplemented, "HealthCheck RPC not yet implemented - awaiting protobuf code generation")
}

// marshalRequest marshals a request to JSON for logging.
func marshalRequest(req interface{}) string {
	b, _ := json.Marshal(req)
	return string(b)
}

// handleGRPCError converts gRPC errors to CryptoError.
func handleGRPCError(err error, operation, correlationID string) error {
	st, ok := status.FromError(err)
	if !ok {
		return NewCryptoError("UNKNOWN_ERROR", fmt.Sprintf("%s failed: %v", operation, err), correlationID, err)
	}

	switch st.Code() {
	case codes.NotFound:
		return NewCryptoError(ErrCodeKeyNotFound, st.Message(), correlationID, err)
	case codes.InvalidArgument:
		return NewCryptoError(ErrCodeInvalidInput, st.Message(), correlationID, err)
	case codes.Unavailable:
		return NewCryptoError(ErrCodeServiceUnavailable, st.Message(), correlationID, err)
	case codes.PermissionDenied:
		return NewCryptoError("PERMISSION_DENIED", st.Message(), correlationID, err)
	case codes.Unauthenticated:
		return NewCryptoError("UNAUTHENTICATED", st.Message(), correlationID, err)
	default:
		return NewCryptoError("CRYPTO_ERROR", fmt.Sprintf("%s failed: %s", operation, st.Message()), correlationID, err)
	}
}
