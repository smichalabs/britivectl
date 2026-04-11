package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/smichalabs/britivectl/internal/config"
)

func TestCheckoutStatePath_Sanitizes(t *testing.T) {
	setupXDG(t)
	// Aliases that contain path-traversal characters should be sanitized.
	got := config.CheckoutStatePath("../../etc/passwd")
	if filepath.Base(got) == "passwd.json" {
		t.Errorf("CheckoutStatePath did not sanitize traversal: %q", got)
	}
}

func TestLoadCheckoutState_Missing(t *testing.T) {
	setupXDG(t)
	state, err := config.LoadCheckoutState("nonexistent")
	if !errors.Is(err, config.ErrCheckoutStateMiss) {
		t.Errorf("err = %v, want ErrCheckoutStateMiss", err)
	}
	if state != nil {
		t.Errorf("expected nil state on miss, got %+v", state)
	}
}

func TestSaveAndLoadCheckoutState(t *testing.T) {
	setupXDG(t)

	original := &config.CheckoutState{
		Alias:         "aws-admin-prod",
		TransactionID: "txn-123",
		CheckedOutAt:  time.Now().UTC().Truncate(time.Second),
		ExpiresAt:     time.Now().UTC().Truncate(time.Second).Add(4 * time.Hour),
	}
	if err := config.SaveCheckoutState(original); err != nil {
		t.Fatalf("SaveCheckoutState() error = %v", err)
	}

	loaded, err := config.LoadCheckoutState("aws-admin-prod")
	if err != nil {
		t.Fatalf("LoadCheckoutState() error = %v", err)
	}
	if loaded.Alias != original.Alias {
		t.Errorf("Alias = %q, want %q", loaded.Alias, original.Alias)
	}
	if loaded.TransactionID != original.TransactionID {
		t.Errorf("TransactionID = %q, want %q", loaded.TransactionID, original.TransactionID)
	}
	if !loaded.ExpiresAt.Equal(original.ExpiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", loaded.ExpiresAt, original.ExpiresAt)
	}
}

func TestSaveCheckoutState_Nil(t *testing.T) {
	setupXDG(t)
	if err := config.SaveCheckoutState(nil); err == nil {
		t.Error("expected error for nil state, got nil")
	}
}

func TestSaveCheckoutState_NoAlias(t *testing.T) {
	setupXDG(t)
	if err := config.SaveCheckoutState(&config.CheckoutState{}); err == nil {
		t.Error("expected error for empty alias, got nil")
	}
}

func TestLoadCheckoutState_Malformed(t *testing.T) {
	setupXDG(t)
	if err := os.MkdirAll(config.CheckoutsDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config.CheckoutStatePath("broken"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := config.LoadCheckoutState("broken"); err == nil {
		t.Fatal("expected error for malformed state file, got nil")
	}
}

func TestDeleteCheckoutState(t *testing.T) {
	setupXDG(t)

	// Save then delete.
	state := &config.CheckoutState{
		Alias:        "to-delete",
		CheckedOutAt: time.Now(),
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}
	if err := config.SaveCheckoutState(state); err != nil {
		t.Fatal(err)
	}
	if err := config.DeleteCheckoutState("to-delete"); err != nil {
		t.Fatalf("DeleteCheckoutState() error = %v", err)
	}
	if _, err := config.LoadCheckoutState("to-delete"); !errors.Is(err, config.ErrCheckoutStateMiss) {
		t.Errorf("after delete, LoadCheckoutState err = %v, want ErrCheckoutStateMiss", err)
	}
}

func TestDeleteCheckoutState_Missing(t *testing.T) {
	setupXDG(t)
	// Deleting a non-existent state should NOT be an error.
	if err := config.DeleteCheckoutState("never-existed"); err != nil {
		t.Errorf("DeleteCheckoutState on missing file returned error: %v", err)
	}
}

func TestCheckoutState_IsFresh(t *testing.T) {
	cases := []struct {
		name   string
		state  *config.CheckoutState
		buffer time.Duration
		want   bool
	}{
		{"nil", nil, 5 * time.Minute, false},
		{"zero expiry", &config.CheckoutState{}, 5 * time.Minute, false},
		{"expires in 4h", &config.CheckoutState{ExpiresAt: time.Now().Add(4 * time.Hour)}, 5 * time.Minute, true},
		{"expires in 1m", &config.CheckoutState{ExpiresAt: time.Now().Add(1 * time.Minute)}, 5 * time.Minute, false},
		{"already expired", &config.CheckoutState{ExpiresAt: time.Now().Add(-1 * time.Hour)}, 5 * time.Minute, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.state.IsFresh(tc.buffer); got != tc.want {
				t.Errorf("IsFresh() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCheckoutState_Remaining(t *testing.T) {
	cases := []struct {
		name        string
		state       *config.CheckoutState
		wantNonZero bool
	}{
		{"nil returns zero", nil, false},
		{"zero expiry returns zero", &config.CheckoutState{}, false},
		{"future expiry returns positive", &config.CheckoutState{ExpiresAt: time.Now().Add(1 * time.Hour)}, true},
		{"past expiry returns zero", &config.CheckoutState{ExpiresAt: time.Now().Add(-1 * time.Hour)}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.state.Remaining()
			if tc.wantNonZero && got <= 0 {
				t.Errorf("Remaining() = %v, want > 0", got)
			}
			if !tc.wantNonZero && got > 0 {
				t.Errorf("Remaining() = %v, want 0", got)
			}
		})
	}
}
