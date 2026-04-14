package output

import (
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
	s := spinner.New(spinner.CharSets[14], 80*time.Millisecond)
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

// Success stops the spinner and prints a success message.
func (s *Spinner) Success(message string) {
	s.sp.Stop()
	ResetTTY()
	color.Green("✓ %s", message)
}

// Fail stops the spinner and prints a failure message.
func (s *Spinner) Fail(message string) {
	s.sp.Stop()
	ResetTTY()
	color.Red("✗ %s", message)
}
