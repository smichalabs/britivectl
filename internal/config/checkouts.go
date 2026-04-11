package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CheckoutsDir returns the directory where per-profile checkout state files
// live (one JSON file per alias).
func CheckoutsDir() string {
	return filepath.Join(xdgCacheDir(), "checkouts")
}

// CheckoutStatePath returns the absolute path to the state file for a given
// alias. Aliases are sanitized so they cannot escape the checkouts dir.
func CheckoutStatePath(alias string) string {
	return filepath.Join(CheckoutsDir(), sanitizeFilename(alias)+".json")
}

// CheckoutState records the result of the last successful checkout for one
// profile alias. It is written to ~/.cache/bctl/checkouts/<alias>.json so
// that subsequent invocations can decide whether the live credentials in
// ~/.aws/credentials are still valid -- avoiding a redundant Britive API
// call when the user runs the same checkout twice in a row.
type CheckoutState struct {
	Alias         string    `json:"alias"`
	TransactionID string    `json:"transactionId,omitempty"`
	CheckedOutAt  time.Time `json:"checkedOutAt"`
	ExpiresAt     time.Time `json:"expiresAt"`
}

// ErrCheckoutStateMiss is returned by LoadCheckoutState when no state file
// exists for the alias. Callers should treat it as "do a fresh checkout"
// rather than a hard error.
var ErrCheckoutStateMiss = errors.New("checkout state does not exist")

// LoadCheckoutState reads the on-disk state file for the given alias.
// Returns ErrCheckoutStateMiss if the file does not exist.
func LoadCheckoutState(alias string) (*CheckoutState, error) {
	path := CheckoutStatePath(alias)
	data, err := os.ReadFile(path) //nolint:gosec // path is under our controlled cache dir
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrCheckoutStateMiss
		}
		return nil, fmt.Errorf("reading checkout state: %w", err)
	}

	var state CheckoutState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parsing checkout state: %w", err)
	}
	return &state, nil
}

// SaveCheckoutState writes the given state to disk atomically.
func SaveCheckoutState(state *CheckoutState) error {
	if state == nil {
		return errors.New("checkout state is nil")
	}
	if state.Alias == "" {
		return errors.New("checkout state has no alias")
	}

	dir := CheckoutsDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating checkouts dir: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling checkout state: %w", err)
	}

	path := CheckoutStatePath(state.Alias)
	tmp, err := os.CreateTemp(dir, "checkout-*.json")
	if err != nil {
		return fmt.Errorf("creating temp checkout file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing temp checkout file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp checkout file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return fmt.Errorf("chmod temp checkout file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("renaming checkout file: %w", err)
	}
	return nil
}

// DeleteCheckoutState removes the state file for an alias. Used by checkin
// and logout flows. Missing files are not an error.
func DeleteCheckoutState(alias string) error {
	err := os.Remove(CheckoutStatePath(alias))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("removing checkout state: %w", err)
	}
	return nil
}

// IsFresh reports whether the recorded credentials still have at least
// `buffer` time remaining before they expire. A nil receiver is never fresh.
//
// Buffer is typically a few minutes so callers do not hand stale credentials
// to downstream tools that take a moment to actually use them.
func (s *CheckoutState) IsFresh(buffer time.Duration) bool {
	if s == nil || s.ExpiresAt.IsZero() {
		return false
	}
	return time.Until(s.ExpiresAt) > buffer
}

// Remaining returns the time until expiry, or 0 if already expired or unset.
func (s *CheckoutState) Remaining() time.Duration {
	if s == nil || s.ExpiresAt.IsZero() {
		return 0
	}
	d := time.Until(s.ExpiresAt)
	if d < 0 {
		return 0
	}
	return d
}

// sanitizeFilename strips characters that would let a malicious alias escape
// the checkouts directory. Aliases come from the Britive API so this is
// defense in depth, not the primary control.
func sanitizeFilename(name string) string {
	out := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z',
			c >= 'A' && c <= 'Z',
			c >= '0' && c <= '9',
			c == '-', c == '_', c == '.':
			out = append(out, c)
		default:
			out = append(out, '_')
		}
	}
	if len(out) == 0 {
		return "_"
	}
	return string(out)
}
