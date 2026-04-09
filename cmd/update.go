package cmd

import (
	"fmt"

	"github.com/smichalabs/britivectl/internal/output"
	"github.com/smichalabs/britivectl/internal/update"
	"github.com/smichalabs/britivectl/pkg/version"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	var checkOnly bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update bctl to the latest version",
		Long:  "Check for and download the latest bctl release from GitHub.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(checkOnly)
		},
	}

	cmd.Flags().BoolVar(&checkOnly, "check", false, "only check for updates, do not install")
	return cmd
}

func runUpdate(checkOnly bool) error {
	spin := output.NewSpinner("Checking for updates...")
	spin.Start()

	latest, isNewer, err := update.CheckLatest(version.Version)
	if err != nil {
		spin.Fail("Failed to check for updates")
		return fmt.Errorf("checking for updates: %w", err)
	}
	spin.Stop()

	if !isNewer {
		output.Success("bctl is up to date (version %s)", version.Version)
		return nil
	}

	output.Info("New version available: %s (current: %s)", latest, version.Version)

	if checkOnly {
		fmt.Printf("Run 'bctl update' to install v%s\n", latest)
		return nil
	}

	spin2 := output.NewSpinner(fmt.Sprintf("Downloading bctl v%s...", latest))
	spin2.Start()

	if err := update.DoUpdate(latest); err != nil {
		spin2.Fail("Update failed")
		return fmt.Errorf("updating: %w", err)
	}

	spin2.Success(fmt.Sprintf("Updated to bctl v%s", latest))
	return nil
}
