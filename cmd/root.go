package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/smichalabs/britivectl/internal/britive"
	"github.com/smichalabs/britivectl/internal/config"
	"github.com/smichalabs/britivectl/internal/output"
	"github.com/smichalabs/britivectl/internal/resolver"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile   string
	outputFmt string
	noColor   bool
	tenant    string
)

// rootCmd is the base command for bctl.
var rootCmd = &cobra.Command{
	Use:   "bctl",
	Short: "A polished CLI for Britive JIT access management",
	Long: `bctl is a command-line tool for managing Just-In-Time (JIT) access
through the Britive platform. Check out profiles, manage AWS credentials,
update kubeconfig for EKS, and more.

Documentation: https://smichalabs.dev/utils/bctl/`,
	SilenceUsage: true,
}

// Execute runs the root command with the given context and returns the
// process exit code. The context is propagated to all subcommand handlers
// via cmd.Context() and should be signal-aware so Ctrl-C cancels in-flight
// API calls.
//
// When invoked with no arguments, Execute opens an fzf-style command picker
// so the user can browse or fuzzy-search the available subcommands. The
// default selection is 'checkout' so hitting enter immediately runs the
// zero-touch flow.
//
// Returns an exit code instead of calling os.Exit so the caller (main) can
// run cleanup defers -- specifically output.ResetTTY() -- before the process
// terminates. os.Exit skips defers; returning lets them fire.
func Execute(ctx context.Context) int {
	if shouldShowCommandPicker() {
		chosen, err := resolver.PickCommand(ctx, commandChoices())
		if err != nil {
			if errors.Is(err, resolver.ErrCanceled) {
				return 0
			}
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		os.Args = append(os.Args, chosen)
	}
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		return 1
	}
	return 0
}

// shouldShowCommandPicker reports whether the user invoked 'bctl' with no
// arguments at all, in which case we launch the interactive command picker.
// Any flag or subcommand disables the picker and hands off to cobra.
func shouldShowCommandPicker() bool {
	return len(os.Args) == 1
}

// commandChoices builds the list of top-level subcommands shown in the
// interactive picker. 'checkout' is placed first so pressing enter without
// filtering runs the zero-touch flow.
func commandChoices() []resolver.CommandChoice {
	// checkout goes first -- it's the default action.
	ordered := []string{
		"checkout",
		"status",
		"checkin",
		"profiles",
		"eks",
		"login",
		"logout",
		"init",
		"doctor",
		"issue",
		"config",
		"update",
		"version",
	}

	shortByName := make(map[string]string, len(rootCmd.Commands()))
	for _, c := range rootCmd.Commands() {
		if c.Hidden || c.Name() == "help" || c.Name() == "completion" {
			continue
		}
		shortByName[c.Name()] = c.Short
	}

	choices := make([]resolver.CommandChoice, 0, len(shortByName))
	seen := make(map[string]bool, len(shortByName))
	for _, name := range ordered {
		if short, ok := shortByName[name]; ok {
			choices = append(choices, resolver.CommandChoice{Name: name, Short: short})
			seen[name] = true
		}
	}
	// Append any subcommands we did not explicitly order so new ones show
	// up automatically without anyone updating this list.
	for name, short := range shortByName {
		if !seen[name] {
			choices = append(choices, resolver.CommandChoice{Name: name, Short: short})
		}
	}
	return choices
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.config/bctl/config.yaml)")
	rootCmd.PersistentFlags().StringVarP(&outputFmt, "output", "o", "", "output format: table|json|env|awscreds|process")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable color output")
	rootCmd.PersistentFlags().StringVar(&tenant, "tenant", "", "Britive tenant name (overrides config)")

	_ = viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
	_ = viper.BindPFlag("no_color", rootCmd.PersistentFlags().Lookup("no-color"))
	_ = viper.BindPFlag("tenant", rootCmd.PersistentFlags().Lookup("tenant"))

	// Register all subcommands
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newLoginCmd())
	rootCmd.AddCommand(newLogoutCmd())
	rootCmd.AddCommand(newCheckoutCmd())
	rootCmd.AddCommand(newCheckinCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newProfilesCmd())
	rootCmd.AddCommand(newEKSCmd())
	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newDoctorCmd())
	rootCmd.AddCommand(newIssueCmd())
	rootCmd.AddCommand(newUpdateCmd())
	rootCmd.AddCommand(newCompletionCmd())
}

// requireToken returns a valid token for the tenant, automatically re-triggering
// browser login if the stored token has expired. Mirrors PyBritive's behavior.
// Returns a wrapped ErrNotLoggedIn if no token is found so callers can use
// errors.Is for targeted UX.
func requireToken(ctx context.Context, tenant string) (string, error) {
	token, err := config.GetToken(tenant)
	if err != nil {
		return "", fmt.Errorf("%w: run 'bctl login' first", britive.ErrNotLoggedIn)
	}

	// Check expiry for Bearer (SSO) tokens
	if config.GetTokenType(tenant) == "Bearer" {
		exp := config.GetTokenExpiry(tenant)
		if exp > 0 && time.Now().Unix() >= exp {
			output.Info("Session expired -- re-authenticating...")
			newToken, err := britive.AuthWithBrowser(ctx, tenant)
			if err != nil {
				return "", fmt.Errorf("re-authentication failed: %w", err)
			}
			if err := config.SetToken(tenant, newToken); err != nil {
				return "", fmt.Errorf("storing token: %w", err)
			}
			if newExp := britive.JWTExpiry(newToken); newExp > 0 {
				_ = config.SetTokenExpiry(tenant, newExp)
			}
			token = newToken
		}
	}

	return token, nil
}

// newAPIClient builds a Britive API client using the stored token,
// selecting the correct auth header type (TOKEN vs Bearer).
func newAPIClient(tenant, token string) *britive.Client {
	tokenType := config.GetTokenType(tenant)
	if tokenType == "Bearer" {
		return britive.NewBearerClient(tenant, token)
	}
	return britive.NewClient(tenant, token)
}

// initConfig reads in config file and ENV variables.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(config.ConfigDir())
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.SetEnvPrefix("BCTL")
	viper.AutomaticEnv()

	// Only swallow "file not found" errors -- first run is legitimate.
	// Anything else (malformed YAML, permissions, etc) should surface.
	if err := viper.ReadInConfig(); err != nil {
		var notFoundErr viper.ConfigFileNotFoundError
		if !errors.As(err, &notFoundErr) {
			fmt.Fprintf(os.Stderr, "warning: reading config: %v\n", err)
		}
	}

	if noColor {
		_ = os.Setenv("BCTL_NO_COLOR", "1")
	}
}
