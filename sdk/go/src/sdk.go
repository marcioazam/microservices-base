// Package src provides the Auth Platform Go SDK.
//
// The SDK provides authentication and authorization utilities including:
//   - Token validation with JWKS caching
//   - HTTP middleware for authentication
//   - gRPC interceptors for authentication
//   - PKCE support for OAuth flows
//   - DPoP support for proof-of-possession
//   - Retry logic with exponential backoff
//
// Basic usage:
//
//	client, err := sdk.New(
//	    sdk.WithBaseURL("https://auth.example.com"),
//	    sdk.WithClientID("my-client"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	// Use HTTP middleware
//	http.Handle("/api/", client.HTTPMiddleware()(apiHandler))
package src

import (
	"github.com/auth-platform/sdk-go/src/auth"
	"github.com/auth-platform/sdk-go/src/client"
	"github.com/auth-platform/sdk-go/src/errors"
	"github.com/auth-platform/sdk-go/src/middleware"
	"github.com/auth-platform/sdk-go/src/retry"
	"github.com/auth-platform/sdk-go/src/token"
	"github.com/auth-platform/sdk-go/src/types"
)

// Client is the main SDK client.
type Client = client.Client

// Config holds SDK configuration.
type Config = client.Config

// New creates a new SDK client.
var New = client.New

// NewFromConfig creates a client from config.
var NewFromConfig = client.NewFromConfig

// NewFromEnv creates a client from environment.
var NewFromEnv = client.NewFromEnv

// Configuration options
var (
	WithBaseURL      = client.WithBaseURL
	WithClientID     = client.WithClientID
	WithClientSecret = client.WithClientSecret
	WithTimeout      = client.WithTimeout
	WithJWKSCacheTTL = client.WithJWKSCacheTTL
	WithMaxRetries   = client.WithMaxRetries
	WithBaseDelay    = client.WithBaseDelay
	WithMaxDelay     = client.WithMaxDelay
	WithDPoP         = client.WithDPoP
)

// Error types and helpers
type (
	SDKError  = errors.SDKError
	ErrorCode = errors.ErrorCode
)

var (
	NewError  = errors.NewError
	WrapError = errors.WrapError
	GetCode   = errors.GetCode

	IsTokenExpired  = errors.IsTokenExpired
	IsTokenInvalid  = errors.IsTokenInvalid
	IsTokenMissing  = errors.IsTokenMissing
	IsRateLimited   = errors.IsRateLimited
	IsNetwork       = errors.IsNetwork
	IsValidation    = errors.IsValidation
	IsInvalidConfig = errors.IsInvalidConfig
	IsUnauthorized  = errors.IsUnauthorized
	IsDPoPRequired  = errors.IsDPoPRequired
	IsDPoPInvalid   = errors.IsDPoPInvalid
	IsPKCEInvalid   = errors.IsPKCEInvalid
)

// Result and Option types
type (
	Result[T any] = types.Result[T]
	Option[T any] = types.Option[T]
	Claims        = types.Claims
	TokenResponse = types.TokenResponse
)

// Note: Generic functions (Ok, Err, Some, None, Map, FlatMap, etc.)
// must be called directly from the types package:
//   types.Ok[T](value)
//   types.Some[T](value)
//   types.Map[T, U](result, fn)

// Token extraction
type (
	TokenScheme      = token.TokenScheme
	Extractor        = token.Extractor
	ValidationResult = token.ValidationResult
)

var (
	SchemeBearer  = token.SchemeBearer
	SchemeDPoP    = token.SchemeDPoP
	SchemeUnknown = token.SchemeUnknown

	NewHTTPExtractor    = token.NewHTTPExtractor
	NewGRPCExtractor    = token.NewGRPCExtractor
	NewCookieExtractor  = token.NewCookieExtractor
	NewChainedExtractor = token.NewChainedExtractor
	ExtractBearerToken  = token.ExtractBearerToken
	ExtractDPoPToken    = token.ExtractDPoPToken
)

// PKCE
type PKCEPair = auth.PKCEPair

var (
	GeneratePKCE     = auth.GeneratePKCE
	GenerateVerifier = auth.GenerateVerifier
	ComputeChallenge = auth.ComputeChallenge
	VerifyPKCE       = auth.VerifyPKCE
	ValidateVerifier = auth.ValidateVerifier
)

// DPoP
type (
	DPoPProver  = auth.DPoPProver
	DPoPClaims  = auth.DPoPClaims
	DPoPKeyPair = auth.DPoPKeyPair
)

var (
	NewDPoPProver        = auth.NewDPoPProver
	GenerateES256KeyPair = auth.GenerateES256KeyPair
	GenerateRS256KeyPair = auth.GenerateRS256KeyPair
	ComputeATH           = auth.ComputeATH
	VerifyATH            = auth.VerifyATH
	ComputeJWKThumbprint = auth.ComputeJWKThumbprint
)

// Retry
type RetryPolicy = retry.Policy

var (
	DefaultRetryPolicy = retry.DefaultPolicy
	NewRetryPolicy     = retry.NewPolicy
	RetryWithResponse  = retry.RetryWithResponse
	ParseRetryAfter    = retry.ParseRetryAfter
)

// Note: Generic retry function must be called directly from the retry package:
//   retry.Retry[T](ctx, policy, fn)

// Middleware
var (
	GetClaimsFromContext = middleware.GetClaimsFromContext
	GetTokenFromContext  = middleware.GetTokenFromContext
	GetSubject           = middleware.GetSubject
	GetClientID          = middleware.GetClientID
	HasScope             = middleware.HasScope
	RequireScope         = middleware.RequireScope
	RequireAnyScope      = middleware.RequireAnyScope
	RequireAllScopes     = middleware.RequireAllScopes
	MapToGRPCError       = middleware.MapToGRPCError
)
