package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Profile struct {
	BaseURL           string `mapstructure:"baseurl"`
	Email             string `mapstructure:"email"`
	APIToken          string `mapstructure:"apitoken"`
	DefaultProjectKey string `mapstructure:"defaultprojectkey"`
	DefaultIssueType  string `mapstructure:"defaultissuetype"`
}

type Config struct {
	DefaultProfile string             `mapstructure:"default_profile"`
	Profiles       map[string]Profile `mapstructure:"profiles"`
}

func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".jira-cli")
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

func Load(profileOverride string) (*Profile, error) {
	v := viper.New()
	v.SetConfigFile(ConfigPath())
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("config not found — run 'jira config init' to set up: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config file: %w", err)
	}

	profileName := cfg.DefaultProfile
	if profileName == "" {
		profileName = "default"
	}

	// Priority: flag > JIRA_PROFILE env var > config default_profile
	if env := os.Getenv("JIRA_PROFILE"); env != "" {
		profileName = env
	}
	if profileOverride != "" {
		profileName = profileOverride
	}

	profile, ok := cfg.Profiles[profileName]
	if !ok {
		return nil, fmt.Errorf("profile %q not found in config — run 'jira config init --profile %s'", profileName, profileName)
	}

	// Env vars override profile fields
	if v := os.Getenv("JIRA_BASE_URL"); v != "" {
		profile.BaseURL = v
	}
	if v := os.Getenv("JIRA_EMAIL"); v != "" {
		profile.Email = v
	}
	if v := os.Getenv("JIRA_API_TOKEN"); v != "" {
		profile.APIToken = v
	}
	if v := os.Getenv("JIRA_PROJECT"); v != "" {
		profile.DefaultProjectKey = v
	}

	if err := profile.validate(); err != nil {
		return nil, err
	}

	return &profile, nil
}

func LoadAll() (*Config, error) {
	v := viper.New()
	v.SetConfigFile(ConfigPath())
	v.SetConfigType("yaml")

	_ = v.ReadInConfig()

	var cfg Config
	_ = v.Unmarshal(&cfg)

	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]Profile)
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	if err := os.MkdirAll(ConfigDir(), 0700); err != nil {
		return fmt.Errorf("cannot create config dir: %w", err)
	}

	v := viper.New()
	v.SetConfigFile(ConfigPath())
	v.SetConfigType("yaml")

	v.Set("default_profile", cfg.DefaultProfile)
	v.Set("profiles", cfg.Profiles)

	return v.WriteConfigAs(ConfigPath())
}

func (p *Profile) validate() error {
	if p.BaseURL == "" {
		return fmt.Errorf("base_url is required in the active profile")
	}
	if p.Email == "" {
		return fmt.Errorf("email is required in the active profile")
	}
	if p.APIToken == "" {
		return fmt.Errorf("api_token is required — set it in the config or via JIRA_API_TOKEN")
	}
	return nil
}
