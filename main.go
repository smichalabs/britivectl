package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/smichalabs/britivectl/cmd"
	"github.com/smichalabs/britivectl/internal/output"
	"github.com/smichalabs/britivectl/internal/update"
	"github.com/smichalabs/britivectl/pkg/version"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Kick the update check off as early as possible so the goroutine has
	// the maximum overlap with the user's actual command. The check is a
	// no-op when the cache is fresh (most of the time), and respects
	// BCTL_NO_UPDATE_CHECK / CI markers so it never runs in pipelines.
	notifier := update.DefaultNotifier(version.Version)
	refreshDone := notifier.RefreshIfStale(ctx)

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

	// Give the refresh goroutine a brief window to complete so its result
	// lands in the cache for *this* run's notice. Most bctl commands take
	// longer than 2s anyway (network calls), so the goroutine has usually
	// finished by the time we get here. The select cap exists for fast
	// commands like `bctl --help` or `bctl version`.
	select {
	case <-refreshDone:
	case <-time.After(2 * time.Second):
	}

	notifier.MaybePrintNotice(os.Stderr, term.IsTerminal(int(os.Stderr.Fd())))

	os.Exit(exitCode)
}
