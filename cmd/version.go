package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/smichalabs/britivectl/pkg/version"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the bctl version",
		Long:  "Print the current version, commit, and build date of bctl.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if jsonOutput {
				data := map[string]string{
					"version": version.Version,
					"commit":  version.Commit,
					"built":   version.BuildDate,
				}
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(data)
			}
			fmt.Println(version.String())
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output version info as JSON")
	return cmd
}
