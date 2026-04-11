package system

import (
	"runtime"
	"testing"
)

// TestBrowserCommand_KnownPlatforms verifies the helper returns the expected
// command for the current GOOS. Tests do not actually launch a browser --
// they only inspect the command name and args, which is why browserCommand
// was split out from OpenBrowser in the first place.
func TestBrowserCommand_KnownPlatforms(t *testing.T) {
	const url = "https://example.com/x?y=1"

	cmd, args, err := browserCommand(url)
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		// On any other GOOS we expect ErrUnsupportedPlatform.
		if err == nil {
			t.Fatalf("expected error on %s, got cmd=%q args=%v", runtime.GOOS, cmd, args)
		}
		return
	}

	if err != nil {
		t.Fatalf("unexpected error on %s: %v", runtime.GOOS, err)
	}

	switch runtime.GOOS {
	case "darwin":
		if cmd != "open" {
			t.Errorf("darwin cmd = %q, want open", cmd)
		}
		if len(args) != 1 || args[0] != url {
			t.Errorf("darwin args = %v, want [%q]", args, url)
		}
	case "linux":
		if cmd != "xdg-open" {
			t.Errorf("linux cmd = %q, want xdg-open", cmd)
		}
		if len(args) != 1 || args[0] != url {
			t.Errorf("linux args = %v, want [%q]", args, url)
		}
	case "windows":
		if cmd != "rundll32" {
			t.Errorf("windows cmd = %q, want rundll32", cmd)
		}
		if len(args) != 2 || args[0] != "url.dll,FileProtocolHandler" || args[1] != url {
			t.Errorf("windows args = %v, want [url.dll,FileProtocolHandler %q]", args, url)
		}
	}
}
