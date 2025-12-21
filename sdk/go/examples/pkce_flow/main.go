// Example: PKCE authorization code flow with Auth Platform SDK
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	authplatform "github.com/auth-platform/sdk-go"
)

var (
	client    *authplatform.Client
	pkceStore = make(map[string]*authplatform.PKCEPair) // In production, use secure session storage
)

func main() {
	var err error
	client, err = authplatform.New(authplatform.Config{
		BaseURL:  "https://auth.example.com",
		ClientID: "your-client-id",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/callback", callbackHandler)

	log.Println("Server starting on :8080")
	log.Println("Visit http://localhost:8080/login to start the flow")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	// Generate PKCE pair
	pkce, err := authplatform.GeneratePKCE()
	if err != nil {
		http.Error(w, "Failed to generate PKCE", http.StatusInternalServerError)
		return
	}

	// Generate state
	state, err := authplatform.GenerateState()
	if err != nil {
		http.Error(w, "Failed to generate state", http.StatusInternalServerError)
		return
	}

	// Store PKCE for callback (use secure session in production)
	pkceStore[state] = pkce

	// Build authorization URL
	authURL, err := client.BuildAuthorizationURL(authplatform.AuthorizationRequest{
		RedirectURI:         "http://localhost:8080/callback",
		Scope:               "openid profile email",
		State:               state,
		CodeChallenge:       pkce.Challenge,
		CodeChallengeMethod: pkce.Method,
	})
	if err != nil {
		http.Error(w, "Failed to build auth URL", http.StatusInternalServerError)
		return
	}

	// Redirect to authorization server
	http.Redirect(w, r, authURL, http.StatusFound)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	// Get authorization code and state
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		errorMsg := r.URL.Query().Get("error")
		http.Error(w, fmt.Sprintf("Authorization failed: %s", errorMsg), http.StatusBadRequest)
		return
	}

	// Retrieve PKCE verifier
	pkce, ok := pkceStore[state]
	if !ok {
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}
	delete(pkceStore, state) // Clean up

	// Exchange code for tokens
	tokens, err := client.ExchangeCode(context.Background(), authplatform.TokenExchangeRequest{
		Code:         code,
		RedirectURI:  "http://localhost:8080/callback",
		CodeVerifier: pkce.Verifier,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Token exchange failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Return tokens (in production, store securely and redirect)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":      "Authentication successful!",
		"token_type":   tokens.TokenType,
		"expires_in":   tokens.ExpiresIn,
		"scope":        tokens.Scope,
		"access_token": tokens.AccessToken[:20] + "...", // Truncated for display
	})
}
