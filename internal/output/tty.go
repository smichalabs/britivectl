package output

import (
	"fmt"
	"io"
	"os"
)

// ANSI escape sequences used by ResetTTY. Defined as constants for clarity.
const (
	// showCursor restores cursor visibility. The spinner library hides the
	// cursor while running and is supposed to show it again on Stop, but
	// has been observed to leave it hidden when the process exits while
	// the spinner is mid-frame. This sequence is idempotent.
	showCursor = "\x1b[?25h"

	// resetAttrs clears any active foreground/background color, bold,
	// underline, etc. Belt and suspenders -- the color package usually
	// resets after each call, but a panic mid-write can strand attributes.
	resetAttrs = "\x1b[0m"
)

// ResetTTY writes ANSI escape sequences that put the terminal back in a
// known-good state: cursor visible, no leftover color or text attributes.
// Safe to call multiple times. Safe to call when stdout is redirected to a
// file or pipe -- the escape codes are just bytes and will be ignored by
// downstream consumers.
//
// Called by spinner Stop/Success/Fail and as a final defer in main, so a
// well-behaved bctl invocation always leaves the terminal in a clean state
// for the next command. This was a real bug: bctl was leaving the cursor
// hidden after some commands, which broke `tmux attach` in the same shell.
func ResetTTY() {
	resetTTYTo(os.Stdout)
}

// resetTTYTo is the testable inner. Writes the reset sequences to w.
func resetTTYTo(w io.Writer) {
	// Order matters: show cursor first so the user immediately sees it
	// even if the second write is interrupted.
	_, _ = fmt.Fprint(w, showCursor)
	_, _ = fmt.Fprint(w, resetAttrs)
}
