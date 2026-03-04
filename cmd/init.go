package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/secutec/testmo-cli/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Testmo CLI configuration",
	Long:  "Interactively configure the Testmo API endpoint and token, saving to .testmo.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Testmo instance URL (e.g., mycompany.testmo.net): ")
		url, _ := reader.ReadString('\n')
		url = strings.TrimSpace(url)
		if url == "" {
			return fmt.Errorf("URL is required")
		}
		if !strings.HasPrefix(url, "http") {
			url = "https://" + url
		}
		url = strings.TrimRight(url, "/")

		fmt.Print("API token: ")
		token, _ := reader.ReadString('\n')
		token = strings.TrimSpace(token)
		if token == "" {
			return fmt.Errorf("token is required")
		}

		cfg := &config.Config{
			URL:   url,
			Token: token,
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Println("Configuration saved to .testmo.yaml")
		return nil
	},
}
