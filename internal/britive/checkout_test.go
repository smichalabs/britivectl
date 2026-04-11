package britive

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// ---------------------------------------------------------------------------
// GetCredentials
// ---------------------------------------------------------------------------

func TestGetCredentials_Success(t *testing.T) {
	want := Credentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		SessionToken:    "AQoDYXdzEJr...",
		Region:          "us-east-1",
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/access/txn123/tokens" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(want)
	})

	c := newTestClient(t, handler)
	got, err := c.GetCredentials(context.Background(), "txn123")
	if err != nil {
		t.Fatalf("GetCredentials() unexpected error: %v", err)
	}
	if got.AccessKeyID != want.AccessKeyID {
		t.Errorf("AccessKeyID = %q; want %q", got.AccessKeyID, want.AccessKeyID)
	}
	if got.SecretAccessKey != want.SecretAccessKey {
		t.Errorf("SecretAccessKey = %q; want %q", got.SecretAccessKey, want.SecretAccessKey)
	}
	if got.SessionToken != want.SessionToken {
		t.Errorf("SessionToken = %q; want %q", got.SessionToken, want.SessionToken)
	}
	if got.Region != want.Region {
		t.Errorf("Region = %q; want %q", got.Region, want.Region)
	}
}

func TestGetCredentials_Error(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	})

	c := newTestClient(t, handler)
	_, err := c.GetCredentials(context.Background(), "txn123")
	if err == nil {
		t.Fatal("expected error from GetCredentials() on 500 response, got nil")
	}
}

// ---------------------------------------------------------------------------
// MySessions
// ---------------------------------------------------------------------------

func TestMySessions_Success(t *testing.T) {
	sessions := []CheckedOutProfile{
		{
			TransactionID: "txn-A",
			PapID:         "prof-1",
			EnvironmentID: "env-1",
			Status:        "checkedOut",
			Expiration:    "2026-04-09T12:00:00Z",
		},
		{
			TransactionID: "txn-B",
			PapID:         "prof-2",
			EnvironmentID: "env-2",
			Status:        "checkedOut",
			Expiration:    "2026-04-09T13:00:00Z",
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/access/app-access-status" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sessions)
	})

	c := newTestClient(t, handler)
	got, err := c.MySessions(context.Background())
	if err != nil {
		t.Fatalf("MySessions() unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(got))
	}
	if got[0].TransactionID != "txn-A" {
		t.Errorf("got[0].TransactionID = %q; want %q", got[0].TransactionID, "txn-A")
	}
	if got[1].PapID != "prof-2" {
		t.Errorf("got[1].PapID = %q; want %q", got[1].PapID, "prof-2")
	}
}

func TestMySessions_Empty(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	})

	c := newTestClient(t, handler)
	got, err := c.MySessions(context.Background())
	if err != nil {
		t.Fatalf("MySessions() unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %d elements", len(got))
	}
}

// ---------------------------------------------------------------------------
// Checkin
// ---------------------------------------------------------------------------

func TestCheckin_EmptyID(t *testing.T) {
	// No HTTP call is expected; the function should short-circuit immediately.
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for empty transactionID")
	}))
	err := c.Checkin(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty transactionID, got nil")
	}
}

func TestCheckin_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/access/txn123" || r.Method != http.MethodPut {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	c := newTestClient(t, handler)
	if err := c.Checkin(context.Background(), "txn123"); err != nil {
		t.Fatalf("Checkin() unexpected error: %v", err)
	}
}

func TestCheckin_Error(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	})

	c := newTestClient(t, handler)
	err := c.Checkin(context.Background(), "txn123")
	if err == nil {
		t.Fatal("expected error from Checkin() on 400 response, got nil")
	}
}

// ---------------------------------------------------------------------------
// Checkout
// ---------------------------------------------------------------------------

func TestCheckout_EmptyProfileID(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for empty profileID")
	}))
	_, _, err := c.Checkout(context.Background(), "", "env1")
	if err == nil {
		t.Fatal("expected error for empty profileID, got nil")
	}
}

func TestCheckout_EmptyEnvironmentID(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for empty environmentID")
	}))
	_, _, err := c.Checkout(context.Background(), "prof1", "")
	if err == nil {
		t.Fatal("expected error for empty environmentID, got nil")
	}
}

func TestCheckout_Success(t *testing.T) {
	// atomic counter tracks how many times app-access-status has been polled.
	var pollCount int32

	creds := Credentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		SessionToken:    "FQoGZXIvYXdz...",
		Region:          "us-east-1",
	}

	mux := http.NewServeMux()

	// POST: initiate checkout
	mux.HandleFunc("/api/access/prof123/environments/env456", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		txn := Transaction{
			TransactionID: "txn999",
			ProfileID:     "prof123",
			EnvironmentID: "env456",
			Status:        "pending",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(txn)
	})

	// GET: poll active sessions
	// First call → empty (triggers 2-second sleep in Checkout)
	// Second call → checkedOut
	mux.HandleFunc("/api/access/app-access-status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		n := atomic.AddInt32(&pollCount, 1)
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			// First poll: not ready yet
			_, _ = w.Write([]byte("[]"))
			return
		}
		// Second poll: ready
		sessions := []CheckedOutProfile{
			{
				TransactionID: "txn999",
				PapID:         "prof123",
				Status:        "checkedOut",
				Expiration:    "2026-04-09T12:00:00Z",
			},
		}
		_ = json.NewEncoder(w).Encode(sessions)
	})

	// GET: fetch credentials
	mux.HandleFunc("/api/access/txn999/tokens", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(creds)
	})

	c := newTestClient(t, mux)

	// This call will sleep ~2 seconds because the first poll returns empty.
	session, gotCreds, err := c.Checkout(context.Background(), "prof123", "env456")
	if err != nil {
		t.Fatalf("Checkout() unexpected error: %v", err)
	}
	if session == nil {
		t.Fatal("expected non-nil CheckedOutProfile, got nil")
	}
	if gotCreds == nil {
		t.Fatal("expected non-nil Credentials, got nil")
	}
	if gotCreds.AccessKeyID != creds.AccessKeyID {
		t.Errorf("AccessKeyID = %q; want %q", gotCreds.AccessKeyID, creds.AccessKeyID)
	}
	if gotCreds.Region != creds.Region {
		t.Errorf("Region = %q; want %q", gotCreds.Region, creds.Region)
	}
	if session.TransactionID != "txn999" {
		t.Errorf("TransactionID = %q; want %q", session.TransactionID, "txn999")
	}
}

func TestCheckout_PostError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	})
	c := newTestClient(t, handler)
	_, _, err := c.Checkout(context.Background(), "prof123", "env456")
	if err == nil {
		t.Fatal("expected error when POST checkout returns 500, got nil")
	}
}

func TestCheckout_GetCredentialsError(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/access/prof123/environments/env456", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		txn := Transaction{
			TransactionID: "txn-cred-err",
			ProfileID:     "prof123",
			EnvironmentID: "env456",
			Status:        "pending",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(txn)
	})

	mux.HandleFunc("/api/access/app-access-status", func(w http.ResponseWriter, r *http.Request) {
		sessions := []CheckedOutProfile{
			{
				TransactionID: "txn-cred-err",
				Status:        "checkedOut",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sessions)
	})

	mux.HandleFunc("/api/access/txn-cred-err/tokens", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	})

	c := newTestClient(t, mux)
	_, _, err := c.Checkout(context.Background(), "prof123", "env456")
	if err == nil {
		t.Fatal("expected error when GetCredentials returns 500, got nil")
	}
}

func TestCheckin_NetworkError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	c := NewClient("test-tenant", "test-token")
	c.baseURL = ts.URL
	ts.Close()
	if err := c.Checkin(context.Background(), "txn123"); err == nil {
		t.Fatal("expected error for closed server, got nil")
	}
}

func TestCheckin_InvalidURL(t *testing.T) {
	c := NewClient("test-tenant", "test-token")
	c.baseURL = "://invalid"
	if err := c.Checkin(context.Background(), "txn123"); err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}

func TestCheckout_SessionError(t *testing.T) {
	mux := http.NewServeMux()

	// POST: initiate checkout — succeeds
	mux.HandleFunc("/api/access/prof123/environments/env456", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		txn := Transaction{
			TransactionID: "txn999",
			ProfileID:     "prof123",
			EnvironmentID: "env456",
			Status:        "pending",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(txn)
	})

	// GET: poll active sessions — always returns 500
	mux.HandleFunc("/api/access/app-access-status", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	})

	c := newTestClient(t, mux)
	_, _, err := c.Checkout(context.Background(), "prof123", "env456")
	if err == nil {
		t.Fatal("expected error when app-access-status returns 500, got nil")
	}
}
