package britive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/smichalabs/britivectl/pkg/version"
)

// Client is the Britive HTTP API client.
type Client struct {
	tenant    string
	token     string
	tokenType string // "TOKEN" for API tokens, "Bearer" for browser SSO JWTs
	baseURL   string
	http      *http.Client
}

// NewClient creates a new Britive API client using an API token.
func NewClient(tenant, token string) *Client {
	return &Client{
		tenant:    tenant,
		token:     token,
		tokenType: "TOKEN",
		baseURL:   fmt.Sprintf("https://%s.britive-app.com", tenant),
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewBearerClient creates a Britive API client using a Bearer JWT (from browser SSO).
func NewBearerClient(tenant, token string) *Client {
	return &Client{
		tenant:    tenant,
		token:     token,
		tokenType: "Bearer",
		baseURL:   fmt.Sprintf("https://%s.britive-app.com", tenant),
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// get performs a GET request and unmarshals the response into out.
// The context controls cancellation and deadline for the request.
func (c *Client) get(ctx context.Context, path string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	return c.parseResponse(resp, out)
}

// post performs a POST request with the given body and unmarshals the response.
// The context controls cancellation and deadline for the request.
func (c *Client) post(ctx context.Context, path string, body, out interface{}) error {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return fmt.Errorf("encoding body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, &buf)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	return c.parseResponse(resp, out)
}

// setHeaders applies standard headers to all requests.
// Britive's API requires Content-Type: application/json on all requests (including GETs).
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", c.tokenType+" "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "bctl/"+version.Version)
}

// parseResponse checks the response status and decodes JSON.
// Returns ErrUnauthorized (wrapped) on HTTP 401 so callers can use errors.Is.
func (c *Client) parseResponse(resp *http.Response, out interface{}) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("%w: %s", ErrUnauthorized, string(body))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	if out != nil && len(body) > 0 {
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	return nil
}

// Ping checks connectivity to the Britive API.
// The context controls cancellation and deadline for the request.
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/users/whoami", nil)
	if err != nil {
		return fmt.Errorf("creating ping request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("%w: check your token", ErrUnauthorized)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	return nil
}
