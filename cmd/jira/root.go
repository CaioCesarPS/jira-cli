package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/caiocesarps/jira-cli/internal/output"
)

var (
	profileFlag string
	jsonFlag    bool
)

var rootCmd = &cobra.Command{
	Use:   "jira",
	Short: "Jira CLI — manage issues from the terminal",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		output.JSONMode = jsonFlag
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&profileFlag, "profile", "", "Config profile to use (overrides JIRA_PROFILE and default_profile)")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
}

func main() {
	rootCmd.AddCommand(issueCmd)
	rootCmd.AddCommand(configCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
