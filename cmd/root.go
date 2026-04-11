package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/smichalabs/britivectl/internal/britive"
	"github.com/smichalabs/britivectl/internal/config"
	"github.com/smichalabs/britivectl/internal/output"
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

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.bctl/config.yaml)")
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
	rootCmd.AddCommand(newUpdateCmd())
	rootCmd.AddCommand(newCompletionCmd())
}

// requireToken returns a valid token for the tenant, automatically re-triggering
// browser login if the stored token has expired. Mirrors PyBritive's behavior.
func requireToken(tenant string) (string, error) {
	token, err := config.GetToken(tenant)
	if err != nil {
		return "", fmt.Errorf("not logged in — run 'bctl login' first")
	}

	// Check expiry for Bearer (SSO) tokens
	if config.GetTokenType(tenant) == "Bearer" {
		exp := config.GetTokenExpiry(tenant)
		if exp > 0 && time.Now().Unix() >= exp {
			output.Info("Session expired — re-authenticating...")
			newToken, err := britive.AuthWithBrowser(tenant)
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
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		viper.AddConfigPath(home + "/.bctl")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.SetEnvPrefix("BCTL")
	viper.AutomaticEnv()

	_ = viper.ReadInConfig()

	if noColor {
		_ = os.Setenv("BCTL_NO_COLOR", "1")
	}
}
