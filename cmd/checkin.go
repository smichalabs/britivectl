package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/smichalabs/britivectl/internal/britive"
	"github.com/smichalabs/britivectl/internal/config"
	"github.com/smichalabs/britivectl/internal/output"
	"github.com/smichalabs/britivectl/internal/resolver"
	"github.com/smichalabs/britivectl/internal/state"
	"github.com/spf13/cobra"
)

func newCheckinCmd() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "checkin [alias]",
		Short: "Return a checked-out profile early",
		Long: `Voluntarily return a Britive profile checkout before it expires.

Pass --all to check in every active session at once. This is handy at the
end of the day when multiple profiles are in flight and you want to
release them all without iterating one alias at a time.`,
		Example: `  bctl checkin aws-admin-prod
  bctl checkin --all`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if all {
				if len(args) > 0 {
					return fmt.Errorf("cannot combine --all with an alias argument")
				}
				return runCheckinAll(cmd.Context())
			}
			var query string
			if len(args) == 1 {
				query = args[0]
			}
			return runCheckin(cmd.Context(), query)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "check in every active session")
	return cmd
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

// runCheckinAll returns every active session at once. Unknown sessions (ones
// where we cannot derive a friendly alias from the profile cache) are still
// checked in -- we just identify them by their profile ID so the user sees
// what happened. A single failure does not abort the rest; we report per-row
// success/failure and return a combined error if any failed so the exit code
// reflects reality.
func runCheckinAll(ctx context.Context) error {
	ready, err := state.EnsureReady(ctx, stateCallbacks())
	if err != nil {
		return err
	}

	client := newAPIClient(ready.Tenant, ready.Token)

	sessions, err := client.MySessions(ctx)
	if err != nil {
		return fmt.Errorf("fetching active sessions: %w", err)
	}

	active := activeSessions(sessions)
	if len(active) == 0 {
		output.Info("No active checkouts.")
		return nil
	}

	aliasByProfileID := aliasByProfileIDFromMap(ready.Profiles)

	var failures []error
	for _, s := range active {
		label := aliasByProfileID[s.PapID]
		if label == "" {
			label = fmt.Sprintf("profile %s", s.PapID)
		}

		if err := client.Checkin(ctx, s.TransactionID); err != nil {
			output.Error("checkin failed for %s: %v", label, err)
			failures = append(failures, fmt.Errorf("%s: %w", label, err))
			continue
		}

		if alias := aliasByProfileID[s.PapID]; alias != "" {
			if err := config.DeleteCheckoutState(alias); err != nil {
				output.Warning("could not clear checkout cache for %s: %v", alias, err)
			}
		}
		output.Success("Checked in %s", label)
	}

	if len(failures) > 0 {
		return fmt.Errorf("%d of %d checkins failed", len(failures), len(active))
	}
	output.Success("Checked in all %d active sessions", len(active))
	return nil
}

// activeSessions filters the MySessions response down to rows that are still
// checked out (CheckedIn == nil).
func activeSessions(sessions []britive.CheckedOutProfile) []britive.CheckedOutProfile {
	out := make([]britive.CheckedOutProfile, 0, len(sessions))
	for _, s := range sessions {
		if s.CheckedIn == nil {
			out = append(out, s)
		}
	}
	return out
}

// aliasByProfileIDFromMap inverts the alias -> Profile map so we can look up
// a human-friendly label from a session's PapID.
func aliasByProfileIDFromMap(profiles map[string]config.Profile) map[string]string {
	out := make(map[string]string, len(profiles))
	for alias, p := range profiles {
		if p.ProfileID != "" {
			out[p.ProfileID] = alias
		}
	}
	return out
}
