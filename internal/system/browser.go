// Package system holds small cross-platform helpers for shelling out to the
// host operating system. Anything in here must be platform-aware and degrade
// cleanly when the host does not support the requested operation.
package system

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
)

// ErrUnsupportedPlatform is returned by OpenBrowser when the current GOOS does
// not have a known way to launch a URL handler.
var ErrUnsupportedPlatform = errors.New("unsupported platform for browser launch")

// OpenBrowser opens the given URL in the user's default browser. The context
// controls cancellation of the launch process; note that once the browser
// process is started, cancelling the context does not close the browser.
//
// Returns ErrUnsupportedPlatform on a GOOS we do not know how to handle, or
// the error from exec.Start() on launch failure. Callers should treat any
// error as "browser did not open" and fall back to printing the URL so the
// user can open it manually.
func OpenBrowser(ctx context.Context, url string) error {
	cmd, args, err := browserCommand(url)
	if err != nil {
		return err
	}
	return exec.CommandContext(ctx, cmd, args...).Start() //nolint:gosec // cmd and args are hardcoded per OS, not user input
}

// browserCommand returns the OS-specific command and args to launch a URL.
// Split out from OpenBrowser for testability -- tests can assert which
// command would be invoked without actually launching a browser.
func browserCommand(url string) (string, []string, error) {
	switch runtime.GOOS {
	case "darwin":
		return "open", []string{url}, nil
	case "linux":
		// We do not special-case WSL here. The standard WSL setup either has
		// xdg-open backed by wslview, or has wslu installed which provides
		// wslview directly. Distros that ship neither are rare and the user
		// can copy the URL from the fallback message.
		return "xdg-open", []string{url}, nil
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}, nil
	default:
		return "", nil, fmt.Errorf("%w: %s", ErrUnsupportedPlatform, runtime.GOOS)
	}
}
