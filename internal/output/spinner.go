package output

import (
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

// Spinner wraps briandowns/spinner for consistent CLI output.
type Spinner struct {
	sp      *spinner.Spinner
	message string
}

// NewSpinner creates a new spinner with the given message.
func NewSpinner(message string) *Spinner {
	s := spinner.New(spinner.CharSets[14], 80*time.Millisecond, spinner.WithWriter(os.Stderr))
	s.Suffix = " " + message
	return &Spinner{
		sp:      s,
		message: message,
	}
}

// Start begins animating the spinner.
func (s *Spinner) Start() {
	s.sp.Start()
}

// Stop halts the spinner without printing a final message. Always call
// ResetTTY after stopping so the cursor reappears even when the underlying
// spinner library misses the restore (observed on rare timing races).
func (s *Spinner) Stop() {
	s.sp.Stop()
	ResetTTY()
}

// Success stops the spinner and prints a success message to stderr.
// color.Green writes to color.Output (stdout) by default, which would corrupt
// machine-format payloads, so route the final line to stderr explicitly.
func (s *Spinner) Success(message string) {
	s.sp.Stop()
	ResetTTY()
	_, _ = color.New(color.FgGreen).Fprintf(os.Stderr, "✓ %s\n", message)
}

// Fail stops the spinner and prints a failure message to stderr (see Success).
func (s *Spinner) Fail(message string) {
	s.sp.Stop()
	ResetTTY()
	_, _ = color.New(color.FgRed).Fprintf(os.Stderr, "✗ %s\n", message)
}
