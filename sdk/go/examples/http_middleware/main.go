// Example: HTTP middleware with Auth Platform SDK
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	authplatform "github.com/auth-platform/sdk-go"
)

func main() {
	// Create client
	client, err := authplatform.New(authplatform.Config{
		BaseURL:  "https://auth.example.com",
		ClientID: "your-client-id",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Create middleware with options
	authMiddleware := client.Middleware(
		authplatform.WithSkipPatterns("/health", "/public/.*"),
		authplatform.WithErrorHandler(jsonErrorHandler),
	)

	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/public/info", publicHandler)
	mux.Handle("/api/profile", authMiddleware(http.HandlerFunc(profileHandler)))
	mux.Handle("/api/data", authMiddleware(http.HandlerFunc(dataHandler)))

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func publicHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{
		"message": "This is public",
	})
}

func profileHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := authplatform.GetClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"subject": claims.Subject,
		"issuer":  claims.Issuer,
		"scope":   claims.Scope,
	})
}

func dataHandler(w http.ResponseWriter, r *http.Request) {
	claims, _ := authplatform.GetClaimsFromContext(r.Context())
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user": claims.Subject,
		"data": []string{"item1", "item2", "item3"},
	})
}

func jsonErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   "unauthorized",
		"message": fmt.Sprintf("Authentication failed: %v", err),
	})
}
