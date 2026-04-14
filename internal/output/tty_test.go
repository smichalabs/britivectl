package output

import (
	"bytes"
	"strings"
	"testing"
)

// TestResetTTYTo_WritesShowCursor pins the actual ANSI sequence we emit so
// future regressions in the byte stream are caught, not just whether
// "something" was written.
func TestResetTTYTo_WritesShowCursor(t *testing.T) {
	var buf bytes.Buffer
	resetTTYTo(&buf)

	out := buf.String()
	if !strings.Contains(out, "\x1b[?25h") {
		t.Errorf("expected show-cursor sequence \\x1b[?25h, got %q", out)
	}
	if !strings.Contains(out, "\x1b[0m") {
		t.Errorf("expected reset-attrs sequence \\x1b[0m, got %q", out)
	}
}

// TestResetTTYTo_Idempotent verifies calling reset multiple times produces
// the same bytes per call. The function must not mutate any package state.
func TestResetTTYTo_Idempotent(t *testing.T) {
	var first, second bytes.Buffer
	resetTTYTo(&first)
	resetTTYTo(&second)
	if first.String() != second.String() {
		t.Errorf("calls produced different output:\n first: %q\n second: %q", first.String(), second.String())
	}
}

// TestResetTTYTo_NoPanicOnRedirectedOutput verifies the function is safe to
// call when stdout is a non-terminal writer (file, pipe, byte buffer, etc.)
// -- the escape codes are just bytes and should be silently ignored
// downstream rather than crashing.
func TestResetTTYTo_NoPanicOnRedirectedOutput(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("resetTTYTo panicked on byte buffer: %v", r)
		}
	}()
	var buf bytes.Buffer
	resetTTYTo(&buf)
}
