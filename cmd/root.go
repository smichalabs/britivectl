package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile  string
	outputFmt string
	noColor  bool
	tenant   string
)

// rootCmd is the base command for bctl.
var rootCmd = &cobra.Command{
	Use:   "bctl",
	Short: "A polished CLI for Britive JIT access management",
	Long: `bctl is a command-line tool for managing Just-In-Time (JIT) access
through the Britive platform. Check out profiles, manage AWS credentials,
update kubeconfig for EKS, and more.`,
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
		os.Setenv("BCTL_NO_COLOR", "1")
	}
}
