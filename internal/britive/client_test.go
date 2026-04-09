package britive

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient("mytenant", "mytoken")
	if !strings.Contains(c.baseURL, "britive-app.com") {
		t.Errorf("baseURL %q does not contain 'britive-app.com'", c.baseURL)
	}
	if c.tokenType != "TOKEN" {
		t.Errorf("tokenType = %q, want %q", c.tokenType, "TOKEN")
	}
}

func TestNewBearerClient(t *testing.T) {
	c := NewBearerClient("mytenant", "mytoken")
	if c.tokenType != "Bearer" {
		t.Errorf("tokenType = %q, want %q", c.tokenType, "Bearer")
	}
}

func TestGet_Success(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"key":"value"}`))
	}))

	var result map[string]string
	if err := c.get("/some/path", &result); err != nil {
		t.Fatalf("get() error: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("result[\"key\"] = %q, want %q", result["key"], "value")
	}
}

func TestGet_ServerError(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error"))
	}))

	var result map[string]string
	err := c.get("/some/path", &result)
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error %q does not contain '500'", err.Error())
	}
}

func TestGet_InvalidJSON(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json"))
	}))

	var result map[string]string
	if err := c.get("/some/path", &result); err == nil {
		t.Fatal("expected error for invalid JSON response, got nil")
	}
}

func TestPost_Success(t *testing.T) {
	var receivedMethod string
	var receivedBody []byte

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		receivedBody = buf
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))

	body := map[string]string{"hello": "world"}
	var result map[string]string
	if err := c.post("/some/path", body, &result); err != nil {
		t.Fatalf("post() error: %v", err)
	}
	if receivedMethod != http.MethodPost {
		t.Errorf("method = %q, want %q", receivedMethod, http.MethodPost)
	}
	if !strings.Contains(string(receivedBody), "world") {
		t.Errorf("request body %q does not contain 'world'", string(receivedBody))
	}
	if result["status"] != "ok" {
		t.Errorf("result[\"status\"] = %q, want %q", result["status"], "ok")
	}
}

func TestPost_NilBody(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))

	var result map[string]interface{}
	if err := c.post("/some/path", nil, &result); err != nil {
		t.Fatalf("post() with nil body error: %v", err)
	}
}

func TestPost_Error(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
	}))

	if err := c.post("/some/path", nil, nil); err == nil {
		t.Fatal("expected error for 400 response, got nil")
	}
}

func TestSetHeaders(t *testing.T) {
	var capturedReq *http.Request

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		w.WriteHeader(http.StatusOK)
	}))

	_ = c.get("/some/path", nil)

	if capturedReq == nil {
		t.Fatal("no request was received by the handler")
	}
	if auth := capturedReq.Header.Get("Authorization"); !strings.HasPrefix(auth, "TOKEN ") {
		t.Errorf("Authorization header = %q, want prefix 'TOKEN '", auth)
	}
	if ct := capturedReq.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
	if accept := capturedReq.Header.Get("Accept"); accept != "application/json" {
		t.Errorf("Accept = %q, want %q", accept, "application/json")
	}
	if ua := capturedReq.Header.Get("User-Agent"); !strings.HasPrefix(ua, "bctl/") {
		t.Errorf("User-Agent = %q, want prefix 'bctl/'", ua)
	}
}

func TestParseResponse_NoBody(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Write no body.
	}))

	if err := c.get("/some/path", nil); err != nil {
		t.Fatalf("get() with no body and nil out error: %v", err)
	}
}

func TestParseResponse_Error(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("forbidden"))
	}))

	if err := c.get("/some/path", nil); err == nil {
		t.Fatal("expected error for non-2xx response, got nil")
	}
}

func TestPing_Success(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/users/whoami" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))

	if err := c.Ping(); err != nil {
		t.Fatalf("Ping() error: %v", err)
	}
}

func TestPing_Unauthorized(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))

	err := c.Ping()
	if err == nil {
		t.Fatal("expected error for 401 response, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unauthorized") {
		t.Errorf("error %q does not contain 'unauthorized'", err.Error())
	}
}

func TestPing_ServerError(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	if err := c.Ping(); err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

func TestGet_InvalidURL(t *testing.T) {
	c := NewClient("test-tenant", "test-token")
	c.baseURL = "://invalid" // triggers NewRequestWithContext parse error
	if err := c.get("/path", nil); err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}

func TestGet_ClosedServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	c := NewClient("test-tenant", "test-token")
	c.baseURL = ts.URL
	ts.Close()
	if err := c.get("/path", nil); err == nil {
		t.Fatal("expected error for closed server, got nil")
	}
}

func TestPost_InvalidURL(t *testing.T) {
	c := NewClient("test-tenant", "test-token")
	c.baseURL = "://invalid"
	if err := c.post("/path", nil, nil); err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}

func TestPost_ClosedServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	c := NewClient("test-tenant", "test-token")
	c.baseURL = ts.URL
	ts.Close()
	if err := c.post("/path", nil, nil); err == nil {
		t.Fatal("expected error for closed server, got nil")
	}
}

func TestPost_BadMarshal(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	// channels cannot be JSON-encoded — triggers encode error path
	if err := c.post("/path", make(chan int), nil); err == nil {
		t.Fatal("expected error for unmarshalable body, got nil")
	}
}

func TestPing_InvalidURL(t *testing.T) {
	c := NewClient("test-tenant", "test-token")
	c.baseURL = "://invalid"
	if err := c.Ping(); err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}

func TestPing_ClosedServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	c := NewClient("test-tenant", "test-token")
	c.baseURL = ts.URL
	ts.Close()
	if err := c.Ping(); err == nil {
		t.Fatal("expected error for closed server, got nil")
	}
}
