package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/auth-platform/file-upload/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

// Config holds auth handler configuration
type Config struct {
	JWKSURL  string
	Issuer   string
	Audience string
	CacheTTL time.Duration
}

// UserContext contains authenticated user information
type UserContext struct {
	UserID    string
	TenantID  string
	Roles     []string
	ExpiresAt time.Time
}

// Handler handles JWT authentication and authorization
type Handler struct {
	jwksURL       string
	issuer        string
	audience      string
	cacheDuration time.Duration
	keys          map[string]*rsa.PublicKey
	keysMutex     sync.RWMutex
	lastFetch     time.Time
	httpClient    *http.Client
}

// NewHandler creates a new auth handler
func NewHandler(cfg Config) (*Handler, error) {
	return &Handler{
		jwksURL:       cfg.JWKSURL,
		issuer:        cfg.Issuer,
		audience:      cfg.Audience,
		cacheDuration: cfg.CacheTTL,
		keys:          make(map[string]*rsa.PublicKey),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// ValidateToken validates JWT and returns user context
func (h *Handler) ValidateToken(ctx context.Context, tokenString string) (*UserContext, error) {
	if tokenString == "" {
		return nil, domain.ErrMissingToken
	}

	// Remove "Bearer " prefix if present
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	tokenString = strings.TrimSpace(tokenString)

	if tokenString == "" {
		return nil, domain.ErrMissingToken
	}

	// Parse token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get key ID
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing key ID in token header")
		}

		// Get public key
		key, err := h.getPublicKey(ctx, kid)
		if err != nil {
			return nil, err
		}

		return key, nil
	})

	if err != nil {
		if strings.Contains(err.Error(), "token is expired") {
			return nil, domain.ErrTokenExpired
		}
		return nil, domain.NewDomainError(domain.ErrCodeInvalidToken, "invalid token", err)
	}

	if !token.Valid {
		return nil, domain.ErrInvalidToken
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, domain.ErrInvalidToken
	}

	// Validate issuer
	if h.issuer != "" {
		iss, _ := claims["iss"].(string)
		if iss != h.issuer {
			return nil, domain.NewDomainError(domain.ErrCodeInvalidToken, "invalid issuer", nil)
		}
	}

	// Validate audience
	if h.audience != "" {
		aud, _ := claims["aud"].(string)
		audList, _ := claims["aud"].([]interface{})
		
		validAud := aud == h.audience
		if !validAud && len(audList) > 0 {
			for _, a := range audList {
				if a.(string) == h.audience {
					validAud = true
					break
				}
			}
		}
		
		if !validAud {
			return nil, domain.NewDomainError(domain.ErrCodeInvalidToken, "invalid audience", nil)
		}
	}

	// Extract user context
	userCtx := &UserContext{}

	if sub, ok := claims["sub"].(string); ok {
		userCtx.UserID = sub
	}

	if tenantID, ok := claims["tenant_id"].(string); ok {
		userCtx.TenantID = tenantID
	} else if tenantID, ok := claims["tid"].(string); ok {
		userCtx.TenantID = tenantID
	}

	if roles, ok := claims["roles"].([]interface{}); ok {
		for _, r := range roles {
			if role, ok := r.(string); ok {
				userCtx.Roles = append(userCtx.Roles, role)
			}
		}
	}

	if exp, ok := claims["exp"].(float64); ok {
		userCtx.ExpiresAt = time.Unix(int64(exp), 0)
	}

	return userCtx, nil
}

// AuthorizeAccess checks if user can access resource
func (h *Handler) AuthorizeAccess(ctx context.Context, userCtx *UserContext, resourceTenantID string) error {
	if userCtx == nil {
		return domain.ErrAccessDenied
	}

	// Check tenant isolation
	if userCtx.TenantID != resourceTenantID {
		return domain.ErrAccessDenied
	}

	return nil
}

// getPublicKey retrieves the public key for the given key ID
func (h *Handler) getPublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	// Check cache
	h.keysMutex.RLock()
	key, exists := h.keys[kid]
	needsRefresh := time.Since(h.lastFetch) > h.cacheDuration
	h.keysMutex.RUnlock()

	if exists && !needsRefresh {
		return key, nil
	}

	// Fetch JWKS
	if err := h.fetchJWKS(ctx); err != nil {
		// If we have a cached key, use it even if refresh failed
		if exists {
			return key, nil
		}
		return nil, err
	}

	// Get key from refreshed cache
	h.keysMutex.RLock()
	key, exists = h.keys[kid]
	h.keysMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("key not found: %s", kid)
	}

	return key, nil
}

// fetchJWKS fetches the JWKS from the configured URL
func (h *Handler) fetchJWKS(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.jwksURL, nil)
	if err != nil {
		return err
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS fetch failed with status: %d", resp.StatusCode)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return err
	}

	// Parse keys
	h.keysMutex.Lock()
	defer h.keysMutex.Unlock()

	for _, key := range jwks.Keys {
		if key.Kty != "RSA" {
			continue
		}

		pubKey, err := parseRSAPublicKey(key)
		if err != nil {
			continue
		}

		h.keys[key.Kid] = pubKey
	}

	h.lastFetch = time.Now()
	return nil
}

// JWKS represents a JSON Web Key Set
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// parseRSAPublicKey parses a JWK into an RSA public key
func parseRSAPublicKey(jwk JWK) (*rsa.PublicKey, error) {
	// Decode N (modulus)
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, err
	}
	n := new(big.Int).SetBytes(nBytes)

	// Decode E (exponent)
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, err
	}
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}

// HasRole checks if user has a specific role
func (u *UserContext) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// IsExpired checks if the token has expired
func (u *UserContext) IsExpired() bool {
	return time.Now().After(u.ExpiresAt)
}
