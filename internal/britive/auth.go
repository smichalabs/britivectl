package britive

import (
	"bytes"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"
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

// AuthWithToken validates an API token against the Britive API.
func AuthWithToken(tenant, token string) error {
	c := NewClient(tenant, token)
	if err := c.Ping(); err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}
	return nil
}

// AuthWithBrowser implements Britive's CLI polling auth flow:
//  1. Generate a random verifier and derive auth_token = base64url(sha512(verifier))
//  2. Open browser to /login?token=<auth_token>
//  3. Poll POST /api/auth/cli/retrieve-tokens with the verifier
//  4. Return the Bearer access token once the user completes login
func AuthWithBrowser(tenant string) (string, error) {
	verifier, authToken, err := generateVerifier()
	if err != nil {
		return "", fmt.Errorf("generating auth verifier: %w", err)
	}

	loginURL := fmt.Sprintf("https://%s.britive-app.com/login?token=%s", tenant, authToken)

	fmt.Printf("Opening browser for authentication...\n")
	fmt.Printf("If it does not open automatically, visit:\n  %s\n\n", loginURL)
	fmt.Printf("Waiting for authentication")

	if err := openBrowser(loginURL); err != nil {
		fmt.Printf("\nCould not open browser automatically: %v\n", err)
	}

	return pollForToken(tenant, verifier)
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
			continue // WAF filter — retry
		}

		hash := sha512.Sum512([]byte(verifier))
		authToken = base64.RawURLEncoding.EncodeToString(hash[:])
		return verifier, authToken, nil
	}
}

// pollForToken polls Britive's token retrieval endpoint until the user
// completes browser login or the timeout is reached.
func pollForToken(tenant, verifier string) (string, error) {
	url := fmt.Sprintf("https://%s.britive-app.com/api/auth/cli/retrieve-tokens", tenant)
	body := map[string]interface{}{
		"authParameters": map[string]string{
			"cliToken": verifier,
		},
	}

	client := &http.Client{Timeout: 15 * time.Second}
	deadline := time.Now().Add(5 * time.Minute)

	for time.Now().Before(deadline) {
		fmt.Print(".")

		data, _ := json.Marshal(body)
		resp, err := client.Post(url, "application/json", bytes.NewReader(data))
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

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

		// 4xx means user hasn't finished yet — keep polling
		time.Sleep(2 * time.Second)
	}

	return "", fmt.Errorf("authentication timed out after 5 minutes")
}

// openBrowser opens the specified URL in the default browser.
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return exec.Command(cmd, args...).Start() //nolint:gosec // cmd and args are hardcoded per OS, not user input
}
