package crypto

import (
	"context"
	"time"

	"github.com/auth-platform/iam-policy-service/internal/logging"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Client provides cryptographic operations via the crypto-service.
type Client struct {
	conn            *grpc.ClientConn
	cryptoClient    CryptoServiceClient
	encryptionKeyID KeyID
	signingKeyID    KeyID
	keyCache        *KeyMetadataCache
	logger          *logging.Logger
	metrics         *Metrics
	config          ClientConfig
	propagator      propagation.TextMapPropagator
}

// ClientConfig holds configuration for the crypto client.
type ClientConfig struct {
	Address         string
	Timeout         time.Duration
	EncryptionKeyID KeyID
	SigningKeyID    KeyID
	KeyCacheTTL     time.Duration
	Enabled         bool
	CacheEncryption bool
	DecisionSigning bool
}

// NewClient creates a new crypto client.
func NewClient(cfg ClientConfig, logger *logging.Logger, metrics *Metrics) (*Client, error) {
	if !cfg.Enabled {
		return &Client{
			config:     cfg,
			logger:     logger,
			metrics:    metrics,
			propagator: otel.GetTextMapPropagator(),
		}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, cfg.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		if logger != nil {
			logger.Warn(context.Background(), "failed to connect to crypto-service, operating in degraded mode",
				logging.String("address", cfg.Address),
				logging.Error(err))
		}
		return &Client{
			config:     cfg,
			logger:     logger,
			metrics:    metrics,
			keyCache:   NewKeyMetadataCache(cfg.KeyCacheTTL),
			propagator: otel.GetTextMapPropagator(),
		}, nil
	}

	// TODO: Replace with properly generated protobuf client:
	// cryptoClient := pb.NewCryptoServiceClient(conn)
	cryptoClient := newCryptoServiceClient()

	return &Client{
		conn:            conn,
		cryptoClient:    cryptoClient,
		encryptionKeyID: cfg.EncryptionKeyID,
		signingKeyID:    cfg.SigningKeyID,
		keyCache:        NewKeyMetadataCache(cfg.KeyCacheTTL),
		logger:          logger,
		metrics:         metrics,
		config:          cfg,
		propagator:      otel.GetTextMapPropagator(),
	}, nil
}

// IsConnected returns true if the client is connected to the crypto-service.
func (c *Client) IsConnected() bool {
	return c.conn != nil
}

// IsEnabled returns true if crypto operations are enabled.
func (c *Client) IsEnabled() bool {
	return c.config.Enabled
}

// IsCacheEncryptionEnabled returns true if cache encryption is enabled.
func (c *Client) IsCacheEncryptionEnabled() bool {
	return c.config.CacheEncryption && c.IsConnected()
}

// IsDecisionSigningEnabled returns true if decision signing is enabled.
func (c *Client) IsDecisionSigningEnabled() bool {
	return c.config.DecisionSigning && c.IsConnected()
}


// Encrypt encrypts plaintext using AES-256-GCM via the crypto-service.
// CRITICAL SECURITY REQUIREMENT: This function REQUIRES crypto-service to be connected.
// If crypto-service is unavailable, this function returns an error to prevent using insecure fallback crypto.
func (c *Client) Encrypt(ctx context.Context, plaintext, aad []byte) (*EncryptResult, error) {
	correlationID := getCorrelationID(ctx)
	start := time.Now()

	if !c.IsConnected() {
		if c.metrics != nil {
			c.metrics.RecordFallback()
		}
		// CRITICAL: Fail-fast if crypto-service is unavailable
		// DO NOT use insecure fallback crypto in production
		if c.logger != nil {
			c.logger.Error(ctx, "CRITICAL: crypto-service not connected, cannot encrypt data",
				logging.String("correlation_id", correlationID))
		}
		return nil, NewCryptoError(ErrCodeServiceUnavailable, "crypto-service not connected - encryption unavailable", correlationID, nil)
	}

	// Propagate trace context
	ctx = c.injectTraceContext(ctx)

	// Call crypto-service via gRPC
	resp, err := c.cryptoClient.Encrypt(ctx, &EncryptRequestProto{
		Plaintext:     plaintext,
		KeyId:         c.encryptionKeyID.ToProto(),
		Aad:           aad,
		CorrelationId: correlationID,
	})
	if err != nil {
		if c.metrics != nil {
			c.metrics.RecordEncrypt("error", time.Since(start))
		}
		return nil, handleGRPCError(err, "encrypt", correlationID)
	}

	result := &EncryptResult{
		Ciphertext: resp.Ciphertext,
		IV:         resp.Iv,
		Tag:        resp.Tag,
		KeyID:      resp.KeyId.ToKeyID(),
		Algorithm:  resp.Algorithm,
	}

	duration := time.Since(start)
	if c.metrics != nil {
		c.metrics.RecordEncrypt("success", duration)
	}

	if c.logger != nil {
		c.logger.Debug(ctx, "encrypted data via crypto-service",
			logging.String("correlation_id", correlationID),
			logging.Int("plaintext_size", len(plaintext)),
			logging.Int("ciphertext_size", len(result.Ciphertext)),
			logging.String("key_id", result.KeyID.String()))
	}

	return result, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM via the crypto-service.
// CRITICAL SECURITY REQUIREMENT: This function REQUIRES crypto-service to be connected.
// If crypto-service is unavailable, this function returns an error to prevent using insecure fallback crypto.
func (c *Client) Decrypt(ctx context.Context, ciphertext, iv, tag, aad []byte) ([]byte, error) {
	correlationID := getCorrelationID(ctx)
	start := time.Now()

	if !c.IsConnected() {
		if c.metrics != nil {
			c.metrics.RecordFallback()
		}
		// CRITICAL: Fail-fast if crypto-service is unavailable
		if c.logger != nil {
			c.logger.Error(ctx, "CRITICAL: crypto-service not connected, cannot decrypt data",
				logging.String("correlation_id", correlationID))
		}
		return nil, NewCryptoError(ErrCodeServiceUnavailable, "crypto-service not connected - decryption unavailable", correlationID, nil)
	}

	// Propagate trace context
	ctx = c.injectTraceContext(ctx)

	// Call crypto-service via gRPC
	resp, err := c.cryptoClient.Decrypt(ctx, &DecryptRequestProto{
		Ciphertext:    ciphertext,
		Iv:            iv,
		Tag:           tag,
		KeyId:         c.encryptionKeyID.ToProto(),
		Aad:           aad,
		CorrelationId: correlationID,
	})
	if err != nil {
		if c.metrics != nil {
			c.metrics.RecordDecrypt("error", time.Since(start))
		}
		return nil, handleGRPCError(err, "decrypt", correlationID)
	}

	duration := time.Since(start)
	if c.metrics != nil {
		c.metrics.RecordDecrypt("success", duration)
	}

	if c.logger != nil {
		c.logger.Debug(ctx, "decrypted data via crypto-service",
			logging.String("correlation_id", correlationID),
			logging.Int("ciphertext_size", len(ciphertext)),
			logging.Int("plaintext_size", len(resp.Plaintext)))
	}

	return resp.Plaintext, nil
}

// Sign creates an ECDSA signature via the crypto-service.
// CRITICAL SECURITY REQUIREMENT: This function REQUIRES crypto-service to be connected.
// If crypto-service is unavailable, this function returns an error to prevent using insecure fallback signing.
func (c *Client) Sign(ctx context.Context, data []byte) (*SignResult, error) {
	correlationID := getCorrelationID(ctx)
	start := time.Now()

	if !c.IsConnected() {
		if c.metrics != nil {
			c.metrics.RecordFallback()
		}
		// CRITICAL: Fail-fast if crypto-service is unavailable
		if c.logger != nil {
			c.logger.Error(ctx, "CRITICAL: crypto-service not connected, cannot sign data",
				logging.String("correlation_id", correlationID))
		}
		return nil, NewCryptoError(ErrCodeServiceUnavailable, "crypto-service not connected - signing unavailable", correlationID, nil)
	}

	// Propagate trace context
	ctx = c.injectTraceContext(ctx)

	// Call crypto-service via gRPC
	resp, err := c.cryptoClient.Sign(ctx, &SignRequestProto{
		Data:          data,
		KeyId:         c.signingKeyID.ToProto(),
		HashAlgorithm: HashAlgorithmSHA256,
		CorrelationId: correlationID,
	})
	if err != nil {
		if c.metrics != nil {
			c.metrics.RecordSign("error", time.Since(start))
		}
		return nil, handleGRPCError(err, "sign", correlationID)
	}

	result := &SignResult{
		Signature: resp.Signature,
		KeyID:     resp.KeyId.ToKeyID(),
		Algorithm: resp.Algorithm,
	}

	duration := time.Since(start)
	if c.metrics != nil {
		c.metrics.RecordSign("success", duration)
	}

	if c.logger != nil {
		c.logger.Debug(ctx, "signed data via crypto-service",
			logging.String("correlation_id", correlationID),
			logging.Int("data_size", len(data)),
			logging.Int("signature_size", len(result.Signature)),
			logging.String("key_id", result.KeyID.String()))
	}

	return result, nil
}

// Verify verifies an ECDSA signature via the crypto-service.
// CRITICAL SECURITY REQUIREMENT: This function REQUIRES crypto-service to be connected.
// If crypto-service is unavailable, this function returns an error to prevent using insecure fallback verification.
func (c *Client) Verify(ctx context.Context, data, signature []byte, keyID KeyID) (bool, error) {
	correlationID := getCorrelationID(ctx)
	start := time.Now()

	if !c.IsConnected() {
		if c.metrics != nil {
			c.metrics.RecordFallback()
		}
		// CRITICAL: Fail-fast if crypto-service is unavailable
		if c.logger != nil {
			c.logger.Error(ctx, "CRITICAL: crypto-service not connected, cannot verify signature",
				logging.String("correlation_id", correlationID))
		}
		return false, NewCryptoError(ErrCodeServiceUnavailable, "crypto-service not connected - verification unavailable", correlationID, nil)
	}

	// Propagate trace context
	ctx = c.injectTraceContext(ctx)

	// Call crypto-service via gRPC
	resp, err := c.cryptoClient.Verify(ctx, &VerifyRequestProto{
		Data:          data,
		Signature:     signature,
		KeyId:         keyID.ToProto(),
		HashAlgorithm: HashAlgorithmSHA256,
		CorrelationId: correlationID,
	})
	if err != nil {
		if c.metrics != nil {
			c.metrics.RecordVerify("error", time.Since(start))
		}
		return false, handleGRPCError(err, "verify", correlationID)
	}

	duration := time.Since(start)
	if c.metrics != nil {
		c.metrics.RecordVerify("success", duration)
	}

	if c.logger != nil {
		c.logger.Debug(ctx, "verified signature via crypto-service",
			logging.String("correlation_id", correlationID),
			logging.Bool("valid", resp.Valid),
			logging.String("key_id", keyID.String()))
	}

	return resp.Valid, nil
}

// HealthCheck checks the health of the crypto-service.
func (c *Client) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	start := time.Now()

	if !c.IsConnected() {
		return &HealthStatus{
			Connected: false,
			LatencyMs: time.Since(start).Milliseconds(),
		}, nil
	}

	// Call crypto-service health check via gRPC
	resp, err := c.cryptoClient.HealthCheck(ctx, &HealthCheckRequestProto{})
	if err != nil {
		if c.logger != nil {
			c.logger.Warn(ctx, "crypto-service health check failed", logging.Error(err))
		}
		return &HealthStatus{
			Connected: false,
			LatencyMs: time.Since(start).Milliseconds(),
		}, err
	}

	return &HealthStatus{
		Connected:     resp.Status == ServingStatusServing,
		HSMConnected:  resp.HsmConnected,
		KMSConnected:  resp.KmsConnected,
		Version:       resp.Version,
		UptimeSeconds: resp.UptimeSeconds,
		LatencyMs:     time.Since(start).Milliseconds(),
	}, nil
}

// Close closes the gRPC connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// injectTraceContext injects W3C Trace Context into gRPC metadata.
func (c *Client) injectTraceContext(ctx context.Context) context.Context {
	carrier := make(propagation.MapCarrier)
	c.propagator.Inject(ctx, carrier)

	md := metadata.MD{}
	for k, v := range carrier {
		md.Set(k, v)
	}

	return metadata.NewOutgoingContext(ctx, md)
}

// getCorrelationID extracts correlation ID from context.
func getCorrelationID(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("x-correlation-id"); len(vals) > 0 {
			return vals[0]
		}
	}
	return ""
}

// isRetryableError returns true if the error is retryable.
func isRetryableError(err error) bool {
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted:
			return true
		}
	}
	return false
}

// ==============================================================================
// SECURITY NOTE: Insecure placeholder crypto functions removed (2026-01-09)
// ==============================================================================
//
// Previous versions of this file contained insecure placeholder crypto functions
// (encryptAESGCM, decryptAESGCM, signECDSA, verifyECDSA) that used simple XOR
// operations. These have been REMOVED to prevent accidental use in production.
//
// CRITICAL SECURITY REQUIREMENT:
// - All cryptographic operations MUST be performed via the crypto-service
// - The crypto-service MUST be available and connected before enabling
//   cache encryption or decision signing
// - No fallback crypto is provided - the system will fail-fast if crypto-service
//   is unavailable, which is the correct security behavior
//
// TODO: Implement gRPC integration with crypto-service
// - Add protobuf definitions for EncryptRequest, DecryptRequest, SignRequest, VerifyRequest
// - Implement client stubs for crypto-service API
// - Add retry logic with exponential backoff for transient failures
// - Add circuit breaker to prevent cascading failures
//
// ==============================================================================
