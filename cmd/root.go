package cmd

import (
	"fmt"
	"os"

	"github.com/secutec/testmo-cli/internal/api"
	"github.com/secutec/testmo-cli/internal/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "testmo",
	Short: "Testmo CLI - manage test cases and sync with Testmo",
}

// SetVersion sets the version string displayed by `testmo --version`.
func SetVersion(version, commit, date string) {
	rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// mustClient loads config and creates an API client, exiting on error.
func mustClient() *api.Client {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return api.NewClient(cfg)
}
