// Package main demonstrates PKCE usage for OAuth authorization code flow.
package main

import (
	"fmt"
	"log"
	"net/url"

	sdk "github.com/auth-platform/sdk-go/src"
)

func main() {
	// Generate PKCE pair
	pkce, err := sdk.GeneratePKCE()
	if err != nil {
		log.Fatalf("Failed to generate PKCE: %v", err)
	}

	fmt.Println("PKCE Generated:")
	fmt.Printf("  Verifier: %s\n", pkce.Verifier)
	fmt.Printf("  Challenge: %s\n", pkce.Challenge)
	fmt.Printf("  Method: %s\n", pkce.Method)

	// Build authorization URL with PKCE
	authURL := buildAuthURL(
		"https://auth.example.com/authorize",
		"my-client-id",
		"https://myapp.example.com/callback",
		"openid profile email",
		pkce.Challenge,
	)

	fmt.Println("\nAuthorization URL:")
	fmt.Println(authURL)

	// Verify PKCE (simulating server-side verification)
	fmt.Println("\nVerifying PKCE...")
	if sdk.VerifyPKCE(pkce.Verifier, pkce.Challenge) {
		fmt.Println("✓ PKCE verification successful!")
	} else {
		fmt.Println("✗ PKCE verification failed!")
	}

	// Validate verifier format
	if err := sdk.ValidateVerifier(pkce.Verifier); err != nil {
		fmt.Printf("Verifier validation error: %v\n", err)
	} else {
		fmt.Println("✓ Verifier format is valid")
	}
}

func buildAuthURL(baseURL, clientID, redirectURI, scope, challenge string) string {
	u, _ := url.Parse(baseURL)
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("scope", scope)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	u.RawQuery = q.Encode()
	return u.String()
}
