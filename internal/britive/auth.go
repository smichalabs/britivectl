package britive

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/smichalabs/britivectl/internal/system"
)

// JWTExpiry decodes a JWT (without verification) and returns the exp claim, or 0 on failure.
func JWTExpiry(token string) int64 {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return 0
	}
	return claims.Exp
}

// testBaseURL is set by tests to redirect API calls to a local test server.
// It is always empty in production.
var testBaseURL string

// AuthWithToken validates an API token against the Britive API.
// The context controls cancellation and deadline for the validation request.
func AuthWithToken(ctx context.Context, tenant, token string) error {
	c := NewClient(tenant, token)
	if testBaseURL != "" {
		c.baseURL = testBaseURL
	}
	if err := c.Ping(ctx); err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}
	return nil
}

// AuthWithBrowser implements Britive's CLI polling auth flow:
//  1. Generate a random verifier and derive auth_token = base64url(sha512(verifier))
//  2. Open browser to /login?token=<auth_token>
//  3. Poll POST /api/auth/cli/retrieve-tokens with the verifier
//  4. Return the Bearer access token once the user completes login
//
// Returns ErrAuthTimeout (wrapped) if the context deadline is reached before
// the user completes browser login.
func AuthWithBrowser(ctx context.Context, tenant string) (string, error) {
	verifier, authToken, err := generateVerifier()
	if err != nil {
		return "", fmt.Errorf("generating auth verifier: %w", err)
	}

	loginURL := fmt.Sprintf("https://%s.britive-app.com/login?token=%s", tenant, authToken)

	fmt.Printf("Opening browser for authentication...\n")
	fmt.Printf("If it does not open automatically, visit:\n  %s\n\n", loginURL)
	fmt.Printf("Waiting for authentication")

	if err := openBrowser(ctx, loginURL); err != nil {
		fmt.Printf("\nCould not open browser automatically: %v\n", err)
	}

	pollURL := fmt.Sprintf("https://%s.britive-app.com/api/auth/cli/retrieve-tokens", tenant)
	if testBaseURL != "" {
		pollURL = testBaseURL + "/api/auth/cli/retrieve-tokens"
	}
	return pollForToken(ctx, pollURL, verifier)
}

// generateVerifier creates a random verifier and its SHA-512-based auth token.
// Retries if the verifier contains "--" which Britive's WAF blocks.
func generateVerifier() (verifier, authToken string, err error) {
	for {
		buf := make([]byte, 32)
		if _, err = rand.Read(buf); err != nil {
			return "", "", fmt.Errorf("generating random bytes: %w", err)
		}

		verifier = base64.RawURLEncoding.EncodeToString(buf)
		if strings.Contains(verifier, "--") {
			continue // WAF filter -- retry
		}

		hash := sha512.Sum512([]byte(verifier))
		authToken = base64.RawURLEncoding.EncodeToString(hash[:])
		return verifier, authToken, nil
	}
}

// pollForToken polls Britive's token retrieval endpoint until the user
// completes browser login or the context is canceled.
func pollForToken(ctx context.Context, url, verifier string) (string, error) {
	body := map[string]interface{}{
		"authParameters": map[string]string{
			"cliToken": verifier,
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("encoding auth request body: %w", err)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	pollCtx, cancel := contextWithDefaultDeadline(ctx, 5*time.Minute)
	defer cancel()

	for {
		fmt.Print(".")

		if err := pollCtx.Err(); err != nil {
			return "", fmt.Errorf("%w: %w", ErrAuthTimeout, err)
		}

		req, reqErr := http.NewRequestWithContext(pollCtx, http.MethodPost, url, bytes.NewReader(data))
		if reqErr != nil {
			if err := sleepCtx(pollCtx, 2*time.Second); err != nil {
				return "", fmt.Errorf("%w: %w", ErrAuthTimeout, err)
			}
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, doErr := client.Do(req)
		if doErr != nil {
			if err := sleepCtx(pollCtx, 2*time.Second); err != nil {
				return "", fmt.Errorf("%w: %w", ErrAuthTimeout, err)
			}
			continue
		}

		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return "", fmt.Errorf("reading auth response body: %w", readErr)
		}

		if resp.StatusCode == http.StatusOK {
			var result struct {
				AuthenticationResult struct {
					AccessToken string `json:"accessToken"`
				} `json:"authenticationResult"`
			}
			if err := json.Unmarshal(respBody, &result); err != nil {
				return "", fmt.Errorf("parsing token response: %w", err)
			}
			if result.AuthenticationResult.AccessToken != "" {
				fmt.Println(" authenticated!")
				return result.AuthenticationResult.AccessToken, nil
			}
		}

		// 4xx means user hasn't finished yet -- keep polling
		if err := sleepCtx(pollCtx, 2*time.Second); err != nil {
			return "", fmt.Errorf("%w: %w", ErrAuthTimeout, err)
		}
	}
}

// sleepCtx sleeps for d or returns early if the context is canceled.
func sleepCtx(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

// openBrowser opens the specified URL in the default browser. Wraps the
// shared system.OpenBrowser helper so the existing call sites in this
// package and tests do not change shape.
func openBrowser(ctx context.Context, url string) error {
	return system.OpenBrowser(ctx, url)
}
