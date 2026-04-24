package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/caiocesarps/jira-cli/internal/config"
	"github.com/caiocesarps/jira-cli/internal/output"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage jira-cli configuration",
}

// ── init ──────────────────────────────────────────────────────────────────────

var initProfileName string

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Add or update a profile in the config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _ := config.LoadAll()

		profileName := initProfileName
		if profileName == "" {
			profileName = "default"
		}

		existing := cfg.Profiles[profileName]

		fmt.Printf("Configuring profile: %s\n", profileName)
		fmt.Printf("Press Enter to keep the current value shown in [brackets].\n\n")

		baseURL := prompt("Jira base URL (e.g. https://your-domain.atlassian.net)", existing.BaseURL)
		email := prompt("Your Atlassian email", existing.Email)
		apiToken := prompt("API token (generate at id.atlassian.com/manage-profile/security/api-tokens)", existing.APIToken)
		projectKey := prompt("Default project key (e.g. PROJ)", existing.DefaultProjectKey)
		issueType := promptWithDefault("Default issue type", existing.DefaultIssueType, "Task")

		cfg.Profiles[profileName] = config.Profile{
			BaseURL:           baseURL,
			Email:             email,
			APIToken:          apiToken,
			DefaultProjectKey: projectKey,
			DefaultIssueType:  issueType,
		}

		if cfg.DefaultProfile == "" {
			cfg.DefaultProfile = profileName
		}

		if err := config.Save(cfg); err != nil {
			return exit(err, 1)
		}

		output.PrintResult(
			map[string]string{"profile": profileName, "config_path": config.ConfigPath()},
			fmt.Sprintf("Profile %q saved to %s", profileName, config.ConfigPath()),
		)
		return nil
	},
}

// ── list ──────────────────────────────────────────────────────────────────────

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadAll()
		if err != nil {
			return exit(err, 1)
		}

		if len(cfg.Profiles) == 0 {
			output.Print("No profiles configured — run 'jira config init' to add one.")
			return nil
		}

		activeProfile := cfg.DefaultProfile
		if env := os.Getenv("JIRA_PROFILE"); env != "" {
			activeProfile = env
		}
		if profileFlag != "" {
			activeProfile = profileFlag
		}

		if output.JSONMode {
			type profileEntry struct {
				Name    string `json:"name"`
				BaseURL string `json:"base_url"`
				Email   string `json:"email"`
				Project string `json:"default_project_key"`
				Active  bool   `json:"active"`
			}
			entries := make([]profileEntry, 0, len(cfg.Profiles))
			for name, p := range cfg.Profiles {
				entries = append(entries, profileEntry{
					Name:    name,
					BaseURL: p.BaseURL,
					Email:   p.Email,
					Project: p.DefaultProjectKey,
					Active:  name == activeProfile,
				})
			}
			output.PrintResult(entries, "")
			return nil
		}

		fmt.Printf("%-20s %-40s %-30s %s\n", "PROFILE", "BASE URL", "EMAIL", "PROJECT")
		fmt.Println(strings.Repeat("-", 100))
		for name, p := range cfg.Profiles {
			marker := "  "
			if name == activeProfile {
				marker = "* "
			}
			fmt.Printf("%s%-18s %-40s %-30s %s\n", marker, name, p.BaseURL, p.Email, p.DefaultProjectKey)
		}
		return nil
	},
}

// ── helpers ───────────────────────────────────────────────────────────────────

func init() {
	configInitCmd.Flags().StringVar(&initProfileName, "profile", "", "Profile name to create or update (default: \"default\")")
	configCmd.AddCommand(configInitCmd, configListCmd)
}

func prompt(label, current string) string {
	return promptWithDefault(label, current, "")
}

func promptWithDefault(label, current, fallback string) string {
	shown := current
	if shown == "" {
		shown = fallback
	}

	if shown != "" {
		fmt.Printf("%s [%s]: ", label, shown)
	} else {
		fmt.Printf("%s: ", label)
	}

	reader := bufio.NewReader(os.Stdin)
	val, _ := reader.ReadString('\n')
	val = strings.TrimSpace(val)

	if val == "" {
		return shown
	}
	return val
}
