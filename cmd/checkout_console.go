package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/smichalabs/britivectl/internal/britive"
	"github.com/smichalabs/britivectl/internal/output"
	"github.com/smichalabs/britivectl/internal/resolver"
	"github.com/smichalabs/britivectl/internal/state"
	"github.com/smichalabs/britivectl/internal/system"
)

// runConsoleCheckout checks out a profile for web console access and opens
// the federated sign-in URL in the default browser. No local credentials are
// written and the freshness cache is not touched, since console sessions live
// entirely in the browser.
//
// The sign-in URL grants console access to anyone holding it, so it is only
// printed when the browser cannot be launched or when printURL is set.
func runConsoleCheckout(ctx context.Context, query string, printURL bool) error {
	ready, err := state.EnsureReady(ctx, stateCallbacks())
	if err != nil {
		return err
	}

	match, err := resolver.Resolve(ctx, ready.Profiles, query, os.Stdin, os.Stdout)
	if err != nil {
		if errors.Is(err, resolver.ErrCanceled) {
			output.Info("Canceled.")
			return nil
		}
		return err
	}

	if match.Profile.ProfileID == "" || match.Profile.EnvironmentID == "" {
		return fmt.Errorf("profile %q is missing API IDs -- run 'bctl profiles sync' to update", match.Alias)
	}

	client := newAPIClient(ready.Tenant, ready.Token)

	// Britive rejects a second checkout of the same profile and access type
	// while one is active, so reuse the live console session if there is one.
	consoleURL, err := existingConsoleURL(ctx, client, match)
	switch {
	case err == nil:
	case errors.Is(err, errNoActiveSession):
		spin := output.NewSpinner(fmt.Sprintf("Checking out %s for console access...", match.Alias))
		spin.Start()
		checkedOut, u, err := client.CheckoutConsole(ctx, match.Profile.ProfileID, match.Profile.EnvironmentID)
		if err != nil {
			spin.Fail(fmt.Sprintf("Console checkout failed: %v", err))
			return err
		}
		spin.Success(fmt.Sprintf("Checked out %s for console access (expires: %s)", match.Alias, checkedOut.Expiration))
		consoleURL = u
	default:
		return err
	}

	return openConsole(ctx, consoleURL, printURL)
}

// existingConsoleURL returns the sign-in URL for an already-active console
// checkout of the matched profile. Returns errNoActiveSession when there is
// no such checkout. A MySessions failure is also reported as
// errNoActiveSession so the fresh checkout can surface the real API error.
func existingConsoleURL(ctx context.Context, client *britive.Client, match resolver.Match) (string, error) {
	existing, err := findActiveSession(ctx, client, match.Profile.ProfileID, britive.AccessTypeConsole)
	if err != nil {
		return "", errNoActiveSession
	}
	consoleURL, err := client.GetConsoleURL(ctx, existing.TransactionID)
	if err != nil {
		return "", fmt.Errorf("fetching console URL for existing checkout: %w", err)
	}
	output.Success("Reusing existing console checkout for %s (expires: %s)", match.Alias, existing.Expiration)
	return consoleURL, nil
}

// openConsole launches the console sign-in URL in the default browser, or
// prints it when printURL is set or the browser cannot be launched.
func openConsole(ctx context.Context, consoleURL string, printURL bool) error {
	if printURL {
		fmt.Println(consoleURL)
		return nil
	}
	if err := system.OpenBrowser(ctx, consoleURL); err != nil {
		output.Warning("Could not open browser automatically: %v", err)
		fmt.Println()
		fmt.Println("Open this URL in your browser (it grants console access, do not share it):")
		fmt.Println("  " + consoleURL)
		return nil
	}
	output.Success("Opened the console in your browser")
	return nil
}
