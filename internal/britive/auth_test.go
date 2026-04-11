package britive

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// makeTestJWT builds a minimal JWT with the given payload map for testing.
func makeTestJWT(t *testing.T, payload map[string]interface{}) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("makeTestJWT: marshal payload: %v", err)
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	return header + "." + encodedPayload + ".fakesig"
}

func TestJWTExpiry(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  int64
	}{
		{
			name:  "valid token with exp claim",
			token: makeTestJWT(t, map[string]interface{}{"exp": 1234567890}),
			want:  1234567890,
		},
		{
			name:  "empty string",
			token: "",
			want:  0,
		},
		{
			name:  "invalid format not 3 parts",
			token: "only.two",
			want:  0,
		},
		{
			name:  "bad base64 in payload",
			token: "header.!!!invalid_base64!!!.sig",
			want:  0,
		},
		{
			name:  "valid format but no exp key",
			token: makeTestJWT(t, map[string]interface{}{"sub": "user123"}),
			want:  0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := JWTExpiry(tc.token)
			if got != tc.want {
				t.Errorf("JWTExpiry(%q) = %d, want %d", tc.token, got, tc.want)
			}
		})
	}
}

func TestGenerateVerifier(t *testing.T) {
	verifier, authToken, err := generateVerifier()
	if err != nil {
		t.Fatalf("generateVerifier() error: %v", err)
	}
	if verifier == "" {
		t.Error("verifier is empty")
	}
	if authToken == "" {
		t.Error("authToken is empty")
	}
	if strings.Contains(verifier, "--") {
		t.Errorf("verifier contains '--' which would be blocked by WAF: %q", verifier)
	}
	// Verify verifier is valid base64url by attempting to decode it.
	if _, err := base64.RawURLEncoding.DecodeString(verifier); err != nil {
		t.Errorf("verifier is not valid base64url: %v", err)
	}
}

func TestPollForToken_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"authenticationResult":{"accessToken":"test-token-xyz"}}`))
	}))
	defer ts.Close()

	token, err := pollForToken(context.Background(), ts.URL+"/whatever", "test-verifier")
	if err != nil {
		t.Fatalf("pollForToken() error: %v", err)
	}
	if token != "test-token-xyz" {
		t.Errorf("pollForToken() = %q, want %q", token, "test-token-xyz")
	}
}

func TestPollForToken_BadJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not valid json{{{`))
	}))
	defer ts.Close()

	_, err := pollForToken(context.Background(), ts.URL+"/whatever", "test-verifier")
	if err == nil {
		t.Error("expected error for bad JSON response, got nil")
	}
}

func TestPollForToken_KeepPolling(t *testing.T) {
	var callCount atomic.Int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		if n == 1 {
			// First call: return 401 to simulate user not yet authenticated.
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// Second call: return the token.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"authenticationResult":{"accessToken":"polled-token"}}`))
	}))
	defer ts.Close()

	// Note: pollForToken sleeps 2s between retries, so this test takes ~2s.
	token, err := pollForToken(context.Background(), ts.URL+"/whatever", "test-verifier")
	if err != nil {
		t.Fatalf("pollForToken() error: %v", err)
	}
	if token != "polled-token" {
		t.Errorf("pollForToken() = %q, want %q", token, "polled-token")
	}
	if callCount.Load() < 2 {
		t.Errorf("expected at least 2 calls, got %d", callCount.Load())
	}
}

func TestOpenBrowser(t *testing.T) {
	// Just verify no panic; error value varies by OS/environment.
	_ = openBrowser(context.Background(), "http://localhost:12345")
}

func TestJWTExpiry_InvalidJSONPayload(t *testing.T) {
	// Build a token whose payload is valid base64url but invalid JSON.
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{not valid json`))
	token := header + "." + payload + ".fakesig"
	if got := JWTExpiry(token); got != 0 {
		t.Errorf("JWTExpiry(%q) = %d, want 0 for invalid JSON payload", token, got)
	}
}

func TestAuthWithToken_Error(t *testing.T) {
	// Use a tenant that cannot resolve so Ping() returns a network error.
	err := AuthWithToken(context.Background(), "this-tenant-does-not-exist.invalid", "fake-token")
	if err == nil {
		t.Fatal("expected error from AuthWithToken with unreachable tenant, got nil")
	}
}

func TestAuthWithToken_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	testBaseURL = ts.URL
	defer func() { testBaseURL = "" }()

	if err := AuthWithToken(context.Background(), "test-tenant", "test-token"); err != nil {
		t.Fatalf("AuthWithToken() unexpected error: %v", err)
	}
}

func TestAuthWithBrowser_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"authenticationResult":{"accessToken":"browser-token"}}`))
	}))
	defer ts.Close()

	testBaseURL = ts.URL
	defer func() { testBaseURL = "" }()

	// openBrowser will fail in CI (no browser), but AuthWithBrowser continues anyway.
	token, err := AuthWithBrowser(context.Background(), "test-tenant")
	if err != nil {
		t.Fatalf("AuthWithBrowser() unexpected error: %v", err)
	}
	if token != "browser-token" {
		t.Errorf("AuthWithBrowser() = %q, want %q", token, "browser-token")
	}
}
