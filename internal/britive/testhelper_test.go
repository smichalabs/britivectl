package britive

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestClient creates an httptest server backed by handler and returns a Client
// whose baseURL points at that server. The server is closed automatically when
// the test finishes.
func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	c := NewClient("test-tenant", "test-token")
	c.baseURL = ts.URL
	return c
}
