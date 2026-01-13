# Design: Components and Interfaces

## Error Handling

```go
package errors

type ErrorCode string

const (
    ErrCodeInvalidConfig ErrorCode = "INVALID_CONFIG"
    ErrCodeTokenExpired  ErrorCode = "TOKEN_EXPIRED"
    ErrCodeTokenInvalid  ErrorCode = "TOKEN_INVALID"
    ErrCodeTokenRefresh  ErrorCode = "TOKEN_REFRESH_FAILED"
    ErrCodeNetwork       ErrorCode = "NETWORK_ERROR"
    ErrCodeRateLimited   ErrorCode = "RATE_LIMITED"
    ErrCodeValidation    ErrorCode = "VALIDATION_FAILED"
    ErrCodeUnauthorized  ErrorCode = "UNAUTHORIZED"
    ErrCodeDPoPRequired  ErrorCode = "DPOP_REQUIRED"
    ErrCodeDPoPInvalid   ErrorCode = "DPOP_INVALID"
    ErrCodePKCEInvalid   ErrorCode = "PKCE_INVALID"
)

type SDKError struct {
    Code    ErrorCode
    Message string
    Cause   error
}

func (e *SDKError) Error() string
func (e *SDKError) Unwrap() error
func (e *SDKError) Is(target error) bool

func IsTokenExpired(err error) bool
func IsRateLimited(err error) bool
func IsNetwork(err error) bool
func IsValidation(err error) bool
func IsDPoPInvalid(err error) bool
func IsPKCEInvalid(err error) bool
func SanitizeError(err error) error
```

## Result and Option Types

```go
package types

type Result[T any] struct {
    value T
    err   error
    ok    bool
}

func Ok[T any](value T) Result[T]
func Err[T any](err error) Result[T]
func (r Result[T]) IsOk() bool
func (r Result[T]) IsErr() bool
func (r Result[T]) Unwrap() T
func (r Result[T]) UnwrapOr(defaultValue T) T
func (r Result[T]) Error() error
func Map[T, U any](r Result[T], fn func(T) U) Result[U]
func FlatMap[T, U any](r Result[T], fn func(T) Result[U]) Result[U]
func MapErr[T any](r Result[T], fn func(error) error) Result[T]

type Option[T any] struct {
    value T
    some  bool
}

func Some[T any](value T) Option[T]
func None[T any]() Option[T]
func (o Option[T]) IsSome() bool
func (o Option[T]) IsNone() bool
func (o Option[T]) Unwrap() T
func (o Option[T]) UnwrapOr(defaultValue T) T
func ToOption[T any](r Result[T]) Option[T]
func OkOr[T any](o Option[T], err error) Result[T]
```

## Token Extraction

```go
package token

type TokenScheme string

const (
    SchemeBearer  TokenScheme = "Bearer"
    SchemeDPoP    TokenScheme = "DPoP"
    SchemeUnknown TokenScheme = ""
)

type Extractor interface {
    Extract(ctx context.Context) (token string, scheme TokenScheme, err error)
}

type HTTPExtractor struct { request *http.Request }
type GRPCExtractor struct{}
type CookieExtractor struct { request *http.Request; cookieName string; scheme TokenScheme }
type ChainedExtractor struct { extractors []Extractor }

func NewChainedExtractor(extractors ...Extractor) *ChainedExtractor
```

## Retry Logic

```go
package retry

type Policy struct {
    MaxRetries           int
    BaseDelay            time.Duration
    MaxDelay             time.Duration
    Jitter               float64
    RetryableStatusCodes []int
}

func DefaultPolicy() *Policy
func (p *Policy) CalculateDelay(attempt int) time.Duration
func (p *Policy) IsRetryable(statusCode int) bool
func ParseRetryAfter(header string) (time.Duration, bool)

type Result[T any] struct { Value T; Err error; Attempts int }

func Retry[T any](ctx context.Context, p *Policy, op func(ctx context.Context) (T, error)) Result[T]
func RetryWithResponse(ctx context.Context, p *Policy, op func(ctx context.Context) (*http.Response, error)) (*http.Response, error)
```

## PKCE Implementation

```go
package auth

type PKCEGenerator interface {
    GenerateVerifier() (string, error)
    ComputeChallenge(verifier string) string
}

type DefaultPKCEGenerator struct { VerifierLength int }
type PKCEPair struct { Verifier string; Challenge string; Method string }

func GeneratePKCE() (*PKCEPair, error)
func VerifyPKCE(verifier, challenge string) bool
func ValidateVerifier(verifier string) error
```

## DPoP Implementation

```go
type DPoPProver interface {
    GenerateProof(ctx context.Context, method, uri string, accessToken string) (string, error)
    ValidateProof(ctx context.Context, proof string, method, uri string) (*DPoPClaims, error)
}

type DPoPClaims struct {
    jwt.RegisteredClaims
    HTTPMethod      string `json:"htm"`
    HTTPUri         string `json:"htu"`
    AccessTokenHash string `json:"ath,omitempty"`
}

type DPoPKeyPair struct { PrivateKey crypto.Signer; PublicKey crypto.PublicKey; Algorithm string; KeyID string }

func GenerateES256KeyPair() (*DPoPKeyPair, error)
func GenerateRS256KeyPair() (*DPoPKeyPair, error)
func ComputeATH(accessToken string) string
func VerifyATH(accessToken, expectedATH string) bool
func ComputeJWKThumbprint(publicKey crypto.PublicKey) (string, error)
```

## JWKS Cache

```go
package token

type JWKSCache struct {
    uri          string
    ttl          time.Duration
    cache        *jwk.Cache
    fallbackKeys jwk.Set
    metrics      *JWKSMetrics
}

type JWKSMetrics struct { Hits int64; Misses int64; Refreshes int64; Errors int64 }

func NewJWKSCache(uri string, ttl time.Duration) *JWKSCache
func (c *JWKSCache) ValidateToken(ctx context.Context, token string, audience string) (*Claims, error)
func (c *JWKSCache) ValidateTokenWithOpts(ctx context.Context, token string, opts ValidationOptions) (*Claims, error)
func (c *JWKSCache) GetMetrics() JWKSMetrics
func (c *JWKSCache) Invalidate()
func (c *JWKSCache) AddJWKSEndpoint(uri string) error
```

## HTTP Middleware

```go
package middleware

type Config struct {
    SkipPatterns   []*regexp.Regexp
    ErrorHandler   ErrorHandler
    TokenExtractor token.Extractor
    Audience       string
    Issuer         string
    RequiredClaims []string
}

type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)
type Option func(*Config)

func WithSkipPatterns(patterns ...string) Option
func WithErrorHandler(handler ErrorHandler) Option
func WithAudience(audience string) Option
func WithIssuer(issuer string) Option
func (c *Client) HTTP(opts ...Option) func(http.Handler) http.Handler
func GetClaimsFromContext(ctx context.Context) (*Claims, bool)
```

## gRPC Interceptors

```go
type GRPCConfig struct {
    SkipMethods    []string
    TokenExtractor token.Extractor
    Audience       string
    Issuer         string
    RequiredClaims []string
}

type GRPCOption func(*GRPCConfig)

func (c *Client) UnaryServerInterceptor(opts ...GRPCOption) grpc.UnaryServerInterceptor
func (c *Client) StreamServerInterceptor(opts ...GRPCOption) grpc.StreamServerInterceptor
func MapToGRPCError(err error) error
```

## Configuration

```go
package client

type Config struct {
    BaseURL      string        `env:"AUTH_PLATFORM_BASE_URL"`
    ClientID     string        `env:"AUTH_PLATFORM_CLIENT_ID"`
    ClientSecret string        `env:"AUTH_PLATFORM_CLIENT_SECRET"`
    Timeout      time.Duration `env:"AUTH_PLATFORM_TIMEOUT" default:"30s"`
    JWKSCacheTTL time.Duration `env:"AUTH_PLATFORM_JWKS_CACHE_TTL" default:"1h"`
    MaxRetries   int           `env:"AUTH_PLATFORM_MAX_RETRIES" default:"3"`
    BaseDelay    time.Duration `env:"AUTH_PLATFORM_BASE_DELAY" default:"1s"`
    MaxDelay     time.Duration `env:"AUTH_PLATFORM_MAX_DELAY" default:"30s"`
    DPoPEnabled  bool          `env:"AUTH_PLATFORM_DPOP_ENABLED" default:"false"`
}

func (c *Config) Validate() error
func (c *Config) ApplyDefaults()
func LoadFromEnv() *Config
```
