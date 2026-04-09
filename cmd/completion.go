package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	completionCmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for bctl.

To load completions:

  Bash:
    $ source <(bctl completion bash)
    # Permanently (Linux):
    $ bctl completion bash > /etc/bash_completion.d/bctl
    # Permanently (macOS with Homebrew):
    $ bctl completion bash > $(brew --prefix)/etc/bash_completion.d/bctl

  Zsh:
    $ echo "autoload -U compinit; compinit" >> ~/.zshrc
    $ bctl completion zsh > "${fpath[1]}/_bctl"

  Fish:
    $ bctl completion fish | source
    $ bctl completion fish > ~/.config/fish/completions/bctl.fish

  PowerShell:
    PS> bctl completion powershell | Out-String | Invoke-Expression
`,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return nil
		},
	}
	return completionCmd
}
