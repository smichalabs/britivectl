package britive

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"
)

const callbackPort = "18789"

// AuthWithToken validates the token against the Britive API.
func AuthWithToken(tenant, token string) error {
	c := NewClient(tenant, token)
	if err := c.Ping(); err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}
	return nil
}

// AuthWithBrowser starts a local HTTP server, opens the browser for SSO,
// waits for the redirect, and returns the token.
func AuthWithBrowser(tenant string) (string, error) {
	tokenCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:    ":" + callbackPort,
		Handler: mux,
	}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "missing token parameter", http.StatusBadRequest)
			errCh <- fmt.Errorf("callback received without token")
			return
		}
		fmt.Fprintf(w, "<html><body><h2>Authentication successful!</h2><p>You can close this window and return to bctl.</p></body></html>")
		tokenCh <- token
	})

	// Start listener
	ln, err := net.Listen("tcp", ":"+callbackPort)
	if err != nil {
		return "", fmt.Errorf("starting local server on :%s: %w", callbackPort, err)
	}

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	loginURL := fmt.Sprintf("https://%s.britive-app.com/login?redirect_uri=http://localhost:%s/callback",
		tenant, callbackPort)

	fmt.Printf("Opening browser for authentication...\nIf it does not open automatically, visit:\n  %s\n\n", loginURL)
	if err := openBrowser(loginURL); err != nil {
		fmt.Printf("Could not open browser automatically: %v\n", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var token string
	select {
	case token = <-tokenCh:
	case err = <-errCh:
		_ = srv.Shutdown(context.Background())
		return "", err
	case <-ctx.Done():
		_ = srv.Shutdown(context.Background())
		return "", fmt.Errorf("authentication timed out after 5 minutes")
	}

	_ = srv.Shutdown(context.Background())
	return token, nil
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

	return exec.Command(cmd, args...).Start()
}
