package types

// Claims represents JWT claims.
type Claims struct {
	Subject   string   `json:"sub"`
	Issuer    string   `json:"iss"`
	Audience  []string `json:"aud"`
	ExpiresAt int64    `json:"exp"`
	IssuedAt  int64    `json:"iat"`
	Scope     string   `json:"scope,omitempty"`
	ClientID  string   `json:"client_id,omitempty"`
}

// TokenResponse represents an OAuth token response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// ValidationOptions holds custom validation options.
type ValidationOptions struct {
	Audience       string
	Issuer         string
	RequiredClaims []string
	SkipExpiry     bool
}
