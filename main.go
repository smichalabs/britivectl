package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/smichalabs/britivectl/cmd"
	"github.com/smichalabs/britivectl/internal/output"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Always reset the terminal on exit. bctl uses a spinner that hides the
	// cursor and a bubbletea TUI for pickers; if either is interrupted mid-
	// frame the terminal can be left with the cursor hidden or color
	// attributes active, which then breaks `tmux attach` and similar
	// follow-on commands. ResetTTY is idempotent and safe even when stdout
	// is not a terminal.
	//
	// os.Exit skips deferred functions, so we run cleanup explicitly here
	// instead of using `defer output.ResetTTY()`.
	exitCode := cmd.Execute(ctx)
	output.ResetTTY()
	os.Exit(exitCode)
}
