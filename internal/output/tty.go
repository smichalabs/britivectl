package output

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
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
// Safe to call multiple times.
//
// No-op when stdout is not a terminal. Writing escape codes into a file or
// pipe is not always harmless -- `bctl completion zsh > _bctl` left a
// trailing \x1b[?25h\x1b[0m in the completion script, which zsh tried to
// parse as a glob and failed with "bad pattern". Anything that consumes
// bctl output as data (completions, JSON output, scripts) needs the bytes
// clean.
//
// Called by spinner Stop/Success/Fail and as a final defer in main, so a
// well-behaved bctl invocation always leaves the terminal in a clean state
// for the next command. This was a real bug: bctl was leaving the cursor
// hidden after some commands, which broke `tmux attach` in the same shell.
func ResetTTY() {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return
	}
	resetTTYTo(os.Stdout)
}

// resetTTYTo is the testable inner. Writes the reset sequences to w.
func resetTTYTo(w io.Writer) {
	// Order matters: show cursor first so the user immediately sees it
	// even if the second write is interrupted.
	_, _ = fmt.Fprint(w, showCursor)
	_, _ = fmt.Fprint(w, resetAttrs)
}
