// Package main demonstrates HTTP middleware usage.
package main

import (
	"fmt"
	"log"
	"net/http"

	sdk "github.com/auth-platform/sdk-go/src"
)

func main() {
	// Create client
	client, err := sdk.New(
		sdk.WithBaseURL("https://auth.example.com"),
		sdk.WithClientID("my-api"),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Create protected handler
	protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get claims from context
		claims, ok := sdk.GetClaimsFromContext(r.Context())
		if !ok {
			http.Error(w, "No claims in context", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Hello, %s!\n", claims.Subject)
		fmt.Fprintf(w, "Your scopes: %s\n", claims.Scope)
	})

	// Create admin handler with scope requirement
	adminHandler := sdk.RequireScope("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Welcome to admin area!\n")
	}))

	// Setup routes
	mux := http.NewServeMux()

	// Public endpoints (no auth)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Protected endpoints
	authMiddleware := client.HTTPMiddleware(
		// Skip authentication for health checks
		// middleware.WithSkipPatterns("^/health$"),
	)

	mux.Handle("/api/me", authMiddleware(protectedHandler))
	mux.Handle("/api/admin", authMiddleware(adminHandler))

	fmt.Println("Server starting on :8080")
	fmt.Println("Endpoints:")
	fmt.Println("  GET /health - Public health check")
	fmt.Println("  GET /api/me - Protected, returns user info")
	fmt.Println("  GET /api/admin - Protected, requires 'admin' scope")

	log.Fatal(http.ListenAndServe(":8080", mux))
}
