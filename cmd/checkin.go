package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/smichalabs/britivectl/internal/config"
	"github.com/smichalabs/britivectl/internal/output"
	"github.com/smichalabs/britivectl/internal/resolver"
	"github.com/smichalabs/britivectl/internal/state"
	"github.com/spf13/cobra"
)

func newCheckinCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "checkin <alias>",
		Short: "Return a checked-out profile early",
		Long:  "Voluntarily return a Britive profile checkout before it expires.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var query string
			if len(args) == 1 {
				query = args[0]
			}
			return runCheckin(cmd.Context(), query)
		},
	}
}

func runCheckin(ctx context.Context, query string) error {
	// Reconcile state the same way checkout does so checkin can match any
	// profile the user can see -- not just ones explicitly written to the
	// config file. Earlier versions looked up cfg.Profiles[alias] directly,
	// which silently failed for any profile that came from the sync cache
	// rather than manual config editing.
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

	if match.Profile.ProfileID == "" {
		return fmt.Errorf("profile %q is missing API IDs -- run 'bctl profiles sync' to update", match.Alias)
	}

	client := newAPIClient(ready.Tenant, ready.Token)

	// Find the active transaction for this profile.
	sessions, err := client.MySessions(ctx)
	if err != nil {
		return fmt.Errorf("fetching active sessions: %w", err)
	}

	var transactionID string
	for _, s := range sessions {
		if s.CheckedIn == nil && s.PapID == match.Profile.ProfileID {
			transactionID = s.TransactionID
			break
		}
	}
	if transactionID == "" {
		return fmt.Errorf("no active checkout found for %q", match.Alias)
	}

	spin := output.NewSpinner(fmt.Sprintf("Checking in %s...", match.Alias))
	spin.Start()

	if err := client.Checkin(ctx, transactionID); err != nil {
		spin.Fail(fmt.Sprintf("Checkin failed: %v", err))
		return err
	}

	// Drop the local freshness cache so the next 'bctl checkout' actually
	// hits the Britive API instead of trusting stale state.
	if err := config.DeleteCheckoutState(match.Alias); err != nil {
		output.Warning("could not clear checkout cache: %v", err)
	}

	spin.Success(fmt.Sprintf("Checked in %s successfully", match.Alias))
	return nil
}
