// Package main demonstrates client credentials flow usage.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	sdk "github.com/auth-platform/sdk-go/src"
)

func main() {
	// Create client from environment variables
	client, err := sdk.NewFromEnv(
		sdk.WithBaseURL(os.Getenv("AUTH_PLATFORM_BASE_URL")),
		sdk.WithClientID(os.Getenv("AUTH_PLATFORM_CLIENT_ID")),
		sdk.WithClientSecret(os.Getenv("AUTH_PLATFORM_CLIENT_SECRET")),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Example: Validate a token
	token := os.Getenv("TEST_TOKEN")
	if token != "" {
		claims, err := client.ValidateTokenCtx(context.Background(), token)
		if err != nil {
			if sdk.IsTokenExpired(err) {
				fmt.Println("Token has expired")
			} else if sdk.IsTokenInvalid(err) {
				fmt.Println("Token is invalid")
			} else {
				fmt.Printf("Validation error: %v\n", err)
			}
			return
		}

		fmt.Printf("Token validated successfully!\n")
		fmt.Printf("Subject: %s\n", claims.Subject)
		fmt.Printf("Issuer: %s\n", claims.Issuer)
		fmt.Printf("Scope: %s\n", claims.Scope)
	}

	fmt.Println("Client credentials example completed")
}
