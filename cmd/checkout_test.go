package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/smichalabs/britivectl/internal/britive"
	"github.com/smichalabs/britivectl/internal/config"
	"github.com/smichalabs/britivectl/internal/resolver"
)

// captureStdoutCmd temporarily redirects os.Stdout and returns what was written.
func captureStdoutCmd(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestNormalizeExpiration(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "already RFC3339", in: "2026-04-09T12:00:00Z", want: "2026-04-09T12:00:00Z"},
		{name: "bare Z layout", in: "2026-04-09T12:00:00Z", want: "2026-04-09T12:00:00Z"},
		{name: "millis Z layout", in: "2026-04-09T12:00:00.000Z", want: "2026-04-09T12:00:00Z"},
		{name: "RFC3339Nano with offset normalized to UTC", in: "2026-04-09T07:00:00.500-05:00", want: "2026-04-09T12:00:00Z"},
		{name: "unparseable is omitted", in: "not-a-time", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeExpiration(tt.in)
			if got != tt.want {
				t.Errorf("normalizeExpiration(%q) = %q, want %q", tt.in, got, tt.want)
			}
			if tt.want != "" {
				if _, err := time.Parse(time.RFC3339, got); err != nil {
					t.Errorf("normalizeExpiration(%q) = %q, not RFC3339: %v", tt.in, got, err)
				}
			}
		})
	}
}

const (
	testStatusGlyphs = "✓✗⚠⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
)

// TestResolverPicker_DoesNotWriteStdout proves the interactive picker menu is
// routed to stderr, not stdout, so an ambiguous alias on a machine output
// format cannot corrupt the credential payload. It drives Resolve with the same
// writer the checkout command passes (pickerOut) over a piped (non-TTY) stdin.
func TestResolverPicker_DoesNotWriteStdout(t *testing.T) {
	profiles := map[string]config.Profile{
		"gcp-prod": {BritivePath: "x", Cloud: "gcp"},
		"prod":     {BritivePath: "y", Cloud: "aws"},
	}

	var stderr bytes.Buffer
	stdout := captureStdoutCmd(t, func() {
		oldOut := os.Stdout
		// Resolve writes the menu to pickerOut; mirror the command's wiring but
		// capture it so the assertion can inspect the menu text.
		_, err := resolver.Resolve(t.Context(), profiles, "pro", strings.NewReader("1\n"), &stderr)
		os.Stdout = oldOut
		if err != nil {
			t.Fatalf("Resolve() error: %v", err)
		}
	})

	if stdout != "" {
		t.Errorf("picker leaked to stdout: %q", stdout)
	}
	if !strings.Contains(stderr.String(), "Pick one") {
		t.Errorf("picker menu did not reach the non-stdout writer: %q", stderr.String())
	}
	if pickerOut != os.Stderr {
		t.Errorf("checkout wires the picker to %v, want os.Stderr", pickerOut)
	}
}

// britiveTestServer stands up the four endpoints the checkout flow hits, with a
// non-RFC3339 expiration so the normalization path is exercised end to end.
// activeSession controls whether MySessions reports a live checkout for the
// profile, which drives the reuse path versus the fresh path.
func britiveTestServer(t *testing.T, profileID, envID, rawExpiration string, activeSession bool) *httptest.Server {
	t.Helper()
	const txnID = "txn-123"
	mux := http.NewServeMux()

	mux.HandleFunc("/api/access/app-access-status", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if activeSession {
			// Live, not-checked-in session for this profile: findActiveSession
			// matches it and the reuse path runs.
			_, _ = w.Write([]byte(`[{"transactionId":"` + txnID + `","papId":"` + profileID +
				`","status":"checkedOut","expiration":"` + rawExpiration + `","checkedIn":null}]`))
			return
		}
		// Fresh path: findActiveSession must skip this (checkedIn != nil), but
		// the Checkout poll still matches on transactionId+status.
		_, _ = w.Write([]byte(`[{"transactionId":"` + txnID + `","papId":"` + profileID +
			`","status":"checkedOut","expiration":"` + rawExpiration + `","checkedIn":"2026-01-01T00:00:00Z"}]`))
	})

	mux.HandleFunc("/api/access/"+profileID+"/environments/"+envID, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"transactionId":"` + txnID + `","profileId":"` + profileID +
			`","environmentId":"` + envID + `","status":"checkedOut"}`))
	})

	mux.HandleFunc("/api/access/"+txnID+"/tokens", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accessKeyId":"AKIATEST","secretAccessKey":"secret",` +
			`"sessionToken":"token","region":"us-east-1"}`))
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestCheckoutResolved_ProcessOutputIsCleanRFC3339(t *testing.T) {
	tests := []struct {
		name          string
		activeSession bool
		rawExp        string
		wantExp       string
	}{
		// Non-UTC offset: verbatim emission keeps the -05:00 form, so asserting
		// the exact UTC value bites if normalization is skipped.
		{name: "fresh path non-UTC offset", activeSession: false, rawExp: "2026-04-15T07:00:00-05:00", wantExp: "2026-04-15T12:00:00Z"},
		{name: "reuse path non-UTC offset", activeSession: true, rawExp: "2026-04-15T07:00:00-05:00", wantExp: "2026-04-15T12:00:00Z"},
		// Space-separated layout does not parse as raw RFC3339 at all, so
		// verbatim emission produces a non-RFC3339 string the assertion rejects.
		{name: "fresh path space-separated layout", activeSession: false, rawExp: "2026-04-15 12:00:00", wantExp: "2026-04-15T12:00:00Z"},
		{name: "reuse path space-separated layout", activeSession: true, rawExp: "2026-04-15 12:00:00", wantExp: "2026-04-15T12:00:00Z"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const (
				profileID = "pap-1"
				envID     = "env-1"
			)

			t.Setenv("HOME", t.TempDir())
			srv := britiveTestServer(t, profileID, envID, tt.rawExp, tt.activeSession)
			client := britive.NewClientWithBaseURL("test-tenant", "test-token", srv.URL)

			match := resolver.Match{
				Alias: "aws-test",
				Profile: config.Profile{
					Cloud:         "aws",
					ProfileID:     profileID,
					EnvironmentID: envID,
					AWSProfile:    "aws-test",
				},
			}

			out := captureStdoutCmd(t, func() {
				if err := checkoutResolved(t.Context(), client, "test-tenant", match, false, false, "process"); err != nil {
					t.Fatalf("checkoutResolved() error: %v", err)
				}
			})

			var v map[string]interface{}
			if err := json.Unmarshal([]byte(out), &v); err != nil {
				t.Fatalf("stdout is not pure JSON: %v\nstdout: %q", err, out)
			}

			exp, ok := v["Expiration"].(string)
			if !ok || exp == "" {
				t.Fatalf("Expiration missing or empty in %v", v)
			}
			if exp != tt.wantExp {
				t.Errorf("Expiration not normalized: got %q, want exact %q", exp, tt.wantExp)
			}
			if _, err := time.Parse(time.RFC3339, exp); err != nil {
				t.Errorf("Expiration %q is not RFC3339: %v", exp, err)
			}

			if strings.ContainsAny(out, testStatusGlyphs) {
				t.Errorf("stdout leaked a status glyph: %q", out)
			}
		})
	}
}
