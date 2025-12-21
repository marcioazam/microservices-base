// Example: DPoP sender-constrained tokens with Auth Platform SDK
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	authplatform "github.com/auth-platform/sdk-go"
)

func main() {
	// Create client with DPoP enabled
	client, err := authplatform.New(authplatform.Config{
		BaseURL:      "https://auth.example.com",
		ClientID:     "your-client-id",
		ClientSecret: "your-client-secret",
		DPoPEnabled:  true, // Automatically generates ES256 key pair
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Obtain DPoP-bound token
	tokens, err := client.ClientCredentials(ctx)
	if err != nil {
		log.Fatalf("Failed to get tokens: %v", err)
	}

	log.Printf("Token type: %s", tokens.TokenType) // Should be "DPoP"
	log.Printf("Access token obtained (DPoP-bound)")

	// Use the token with DPoP proof for API calls
	// The SDK automatically generates DPoP proofs for each request
	log.Println("DPoP tokens are bound to the client's key pair")
	log.Println("Each API request includes a fresh DPoP proof")
}

// Example of manual DPoP proof generation for custom use cases
func manualDPoPExample() {
	// Generate key pair
	keyPair, err := authplatform.GenerateES256KeyPair()
	if err != nil {
		log.Fatal(err)
	}

	// Create prover
	prover := authplatform.NewDPoPProver(keyPair)

	// Generate proof for a specific request
	proof, err := prover.GenerateProof(
		context.Background(),
		"POST",
		"https://api.example.com/resource",
		"existing-access-token", // For ath claim binding
	)
	if err != nil {
		log.Fatal(err)
	}

	// Use proof in request
	req, _ := http.NewRequest("POST", "https://api.example.com/resource", nil)
	req.Header.Set("Authorization", "DPoP existing-access-token")
	req.Header.Set("DPoP", proof)

	log.Printf("DPoP proof generated: %s...", proof[:50])
}

// Example of DPoP proof validation (server-side)
func validateDPoPExample() {
	keyPair, _ := authplatform.GenerateES256KeyPair()
	prover := authplatform.NewDPoPProver(keyPair)

	ctx := context.Background()

	// Generate a proof
	proof, _ := prover.GenerateProof(ctx, "GET", "https://api.example.com/data", "")

	// Validate the proof
	claims, err := prover.ValidateProof(ctx, proof, "GET", "https://api.example.com/data")
	if err != nil {
		log.Printf("Proof validation failed: %v", err)
		return
	}

	log.Printf("Proof valid - JTI: %s, Method: %s, URI: %s",
		claims.ID, claims.HTTPMethod, claims.HTTPUri)

	// Verify access token hash if present
	if claims.AccessTokenHash != "" {
		accessToken := "the-access-token"
		if authplatform.VerifyATH(accessToken, claims.AccessTokenHash) {
			log.Println("Access token hash verified")
		}
	}
}

// Example response structure
type TokenInfo struct {
	TokenType   string `json:"token_type"`
	BoundToKey  bool   `json:"bound_to_key"`
	KeyID       string `json:"key_id"`
	Algorithm   string `json:"algorithm"`
}

func displayTokenInfo(tokens *authplatform.TokenResponse) {
	info := TokenInfo{
		TokenType:  tokens.TokenType,
		BoundToKey: tokens.TokenType == "DPoP",
	}

	data, _ := json.MarshalIndent(info, "", "  ")
	log.Printf("Token info:\n%s", string(data))
}
