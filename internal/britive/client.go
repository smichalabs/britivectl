package britive

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/smichalabs/britivectl/pkg/version"
)

// Client is the Britive HTTP API client.
type Client struct {
	tenant  string
	token   string
	baseURL string
	http    *http.Client
}

// NewClient creates a new Britive API client.
func NewClient(tenant, token string) *Client {
	return &Client{
		tenant:  tenant,
		token:   token,
		baseURL: fmt.Sprintf("https://%s.britive-app.com", tenant),
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// get performs a GET request and unmarshals the response into out.
func (c *Client) get(path string, out interface{}) error {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
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
func (c *Client) post(path string, body, out interface{}) error {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return fmt.Errorf("encoding body: %w", err)
		}
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, &buf)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	c.setHeaders(req)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	return c.parseResponse(resp, out)
}

// setHeaders applies standard headers to all requests.
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "TOKEN "+c.token)
	req.Header.Set("User-Agent", "bctl/"+version.Version)
	req.Header.Set("Accept", "application/json")
}

// parseResponse checks the response status and decodes JSON.
func (c *Client) parseResponse(resp *http.Response, out interface{}) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
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
func (c *Client) Ping() error {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/api/v1/users/whoami", nil)
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
		return fmt.Errorf("unauthorized: check your token")
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	return nil
}
