// Package main demonstrates DPoP (Demonstrating Proof of Possession) usage.
package main

import (
	"context"
	"fmt"
	"log"

	sdk "github.com/auth-platform/sdk-go/src"
)

func main() {
	// Generate a key pair for DPoP
	keyPair, err := sdk.GenerateES256KeyPair()
	if err != nil {
		log.Fatalf("Failed to generate key pair: %v", err)
	}

	fmt.Println("DPoP Key Pair Generated:")
	fmt.Printf("  Algorithm: %s\n", keyPair.Algorithm)
	fmt.Printf("  Key ID: %s\n", keyPair.KeyID)

	// Create DPoP prover
	prover := sdk.NewDPoPProver(keyPair)

	// Generate a DPoP proof for a token request
	ctx := context.Background()
	method := "POST"
	uri := "https://auth.example.com/token"

	proof, err := prover.GenerateProof(ctx, method, uri, "")
	if err != nil {
		log.Fatalf("Failed to generate proof: %v", err)
	}

	fmt.Println("\nDPoP Proof Generated:")
	fmt.Printf("  Proof: %s...\n", proof[:50])

	// Validate the proof (simulating server-side validation)
	claims, err := prover.ValidateProof(ctx, proof, method, uri)
	if err != nil {
		log.Fatalf("Proof validation failed: %v", err)
	}

	fmt.Println("\nProof Validated:")
	fmt.Printf("  HTTP Method: %s\n", claims.HTTPMethod)
	fmt.Printf("  HTTP URI: %s\n", claims.HTTPUri)
	fmt.Printf("  JTI: %s\n", claims.ID)

	// Generate proof with access token hash (for resource requests)
	accessToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.example"
	proofWithATH, err := prover.GenerateProof(ctx, "GET", "https://api.example.com/resource", accessToken)
	if err != nil {
		log.Fatalf("Failed to generate proof with ATH: %v", err)
	}

	fmt.Println("\nDPoP Proof with ATH Generated:")
	fmt.Printf("  Proof: %s...\n", proofWithATH[:50])

	// Compute and verify ATH
	ath := sdk.ComputeATH(accessToken)
	fmt.Printf("  ATH: %s\n", ath)

	if sdk.VerifyATH(accessToken, ath) {
		fmt.Println("  ✓ ATH verification successful!")
	}

	// Compute JWK thumbprint
	thumbprint, err := sdk.ComputeJWKThumbprint(keyPair.PublicKey)
	if err != nil {
		log.Fatalf("Failed to compute thumbprint: %v", err)
	}
	fmt.Printf("\nJWK Thumbprint: %s\n", thumbprint)
}
