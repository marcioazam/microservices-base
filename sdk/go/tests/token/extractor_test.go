// Package token provides unit tests for token extraction.
package token

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/auth-platform/sdk-go/src/errors"
	"github.com/auth-platform/sdk-go/src/token"
)

func TestHTTPExtractor_Extract(t *testing.T) {
	tests := []struct {
		name       string
		header     string
		wantToken  string
		wantScheme token.TokenScheme
		wantErr    bool
		errCode    errors.ErrorCode
	}{
		{
			name:       "valid bearer token",
			header:     "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			wantToken:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			wantScheme: token.SchemeBearer,
			wantErr:    false,
		},
		{
			name:       "valid dpop token",
			header:     "DPoP eyJhbGciOiJFUzI1NiIsInR5cCI6ImRwb3Arand0In0",
			wantToken:  "eyJhbGciOiJFUzI1NiIsInR5cCI6ImRwb3Arand0In0",
			wantScheme: token.SchemeDPoP,
			wantErr:    false,
		},
		{
			name:    "missing header",
			header:  "",
			wantErr: true,
			errCode: errors.ErrCodeTokenMissing,
		},
		{
			name:    "invalid format - no space",
			header:  "BearerToken",
			wantErr: true,
			errCode: errors.ErrCodeTokenInvalid,
		},
		{
			name:    "empty token",
			header:  "Bearer ",
			wantErr: true,
			errCode: errors.ErrCodeTokenMissing,
		},
		{
			name:    "unsupported scheme",
			header:  "Basic dXNlcjpwYXNz",
			wantErr: true,
			errCode: errors.ErrCodeTokenInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			extractor := token.NewHTTPExtractor(req)
			gotToken, gotScheme, err := extractor.Extract(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errCode != "" && errors.GetCode(err) != tt.errCode {
					t.Errorf("error code = %v, want %v", errors.GetCode(err), tt.errCode)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotToken != tt.wantToken {
				t.Errorf("token = %v, want %v", gotToken, tt.wantToken)
			}
			if gotScheme != tt.wantScheme {
				t.Errorf("scheme = %v, want %v", gotScheme, tt.wantScheme)
			}
		})
	}
}

func TestHTTPExtractor_NilRequest(t *testing.T) {
	extractor := token.NewHTTPExtractor(nil)
	_, _, err := extractor.Extract(context.Background())
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestCookieExtractor_Extract(t *testing.T) {
	tests := []struct {
		name       string
		cookieName string
		cookieVal  string
		scheme     token.TokenScheme
		wantToken  string
		wantScheme token.TokenScheme
		wantErr    bool
	}{
		{
			name:       "valid cookie with bearer scheme",
			cookieName: "access_token",
			cookieVal:  "mytoken123",
			scheme:     token.SchemeBearer,
			wantToken:  "mytoken123",
			wantScheme: token.SchemeBearer,
			wantErr:    false,
		},
		{
			name:       "valid cookie with unknown scheme defaults to bearer",
			cookieName: "access_token",
			cookieVal:  "mytoken123",
			scheme:     token.SchemeUnknown,
			wantToken:  "mytoken123",
			wantScheme: token.SchemeBearer,
			wantErr:    false,
		},
		{
			name:       "missing cookie",
			cookieName: "access_token",
			cookieVal:  "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.cookieVal != "" {
				req.AddCookie(&http.Cookie{Name: tt.cookieName, Value: tt.cookieVal})
			}

			extractor := token.NewCookieExtractor(req, tt.cookieName, tt.scheme)
			gotToken, gotScheme, err := extractor.Extract(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotToken != tt.wantToken {
				t.Errorf("token = %v, want %v", gotToken, tt.wantToken)
			}
			if gotScheme != tt.wantScheme {
				t.Errorf("scheme = %v, want %v", gotScheme, tt.wantScheme)
			}
		})
	}
}

func TestChainedExtractor_Extract(t *testing.T) {
	t.Run("first extractor succeeds", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer token1")

		chain := token.NewChainedExtractor(
			token.NewHTTPExtractor(req),
			token.NewCookieExtractor(req, "token", token.SchemeBearer),
		)

		gotToken, gotScheme, err := chain.Extract(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotToken != "token1" {
			t.Errorf("token = %v, want token1", gotToken)
		}
		if gotScheme != token.SchemeBearer {
			t.Errorf("scheme = %v, want Bearer", gotScheme)
		}
	})

	t.Run("fallback to second extractor", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{Name: "token", Value: "cookie_token"})

		chain := token.NewChainedExtractor(
			token.NewHTTPExtractor(req), // Will fail - no auth header
			token.NewCookieExtractor(req, "token", token.SchemeBearer),
		)

		gotToken, _, err := chain.Extract(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotToken != "cookie_token" {
			t.Errorf("token = %v, want cookie_token", gotToken)
		}
	})

	t.Run("all extractors fail", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)

		chain := token.NewChainedExtractor(
			token.NewHTTPExtractor(req),
			token.NewCookieExtractor(req, "token", token.SchemeBearer),
		)

		_, _, err := chain.Extract(context.Background())
		if err == nil {
			t.Fatal("expected error when all extractors fail")
		}
	})
}

func TestParseAuthorizationHeader(t *testing.T) {
	tests := []struct {
		name       string
		header     string
		wantToken  string
		wantScheme token.TokenScheme
		wantErr    bool
	}{
		{"bearer lowercase", "bearer token123", "token123", token.SchemeBearer, false},
		{"Bearer mixed case", "Bearer token123", "token123", token.SchemeBearer, false},
		{"dpop lowercase", "dpop proof123", "proof123", token.SchemeDPoP, false},
		{"DPoP mixed case", "DPoP proof123", "proof123", token.SchemeDPoP, false},
		{"no space", "Bearertoken", "", token.SchemeUnknown, true},
		{"empty", "", "", token.SchemeUnknown, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotToken, gotScheme, err := token.ParseAuthorizationHeader(tt.header)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotToken != tt.wantToken {
				t.Errorf("token = %v, want %v", gotToken, tt.wantToken)
			}
			if gotScheme != tt.wantScheme {
				t.Errorf("scheme = %v, want %v", gotScheme, tt.wantScheme)
			}
		})
	}
}

func TestFormatAuthorizationHeader(t *testing.T) {
	result := token.FormatAuthorizationHeader("mytoken", token.SchemeBearer)
	if result != "Bearer mytoken" {
		t.Errorf("got %v, want Bearer mytoken", result)
	}

	result = token.FormatAuthorizationHeader("proof", token.SchemeDPoP)
	if result != "DPoP proof" {
		t.Errorf("got %v, want DPoP proof", result)
	}
}

func TestExtractBearerToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer mytoken")

	tok, err := token.ExtractBearerToken(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "mytoken" {
		t.Errorf("token = %v, want mytoken", tok)
	}
}

func TestExtractDPoPToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "DPoP myproof")

	tok, err := token.ExtractDPoPToken(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "myproof" {
		t.Errorf("token = %v, want myproof", tok)
	}
}

func TestGetDPoPProof(t *testing.T) {
	t.Run("valid proof", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("DPoP", "proof123")

		proof, err := token.GetDPoPProof(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if proof != "proof123" {
			t.Errorf("proof = %v, want proof123", proof)
		}
	})

	t.Run("missing proof", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)

		_, err := token.GetDPoPProof(req)
		if err == nil {
			t.Fatal("expected error for missing DPoP header")
		}
	})
}
