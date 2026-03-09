// Package config provides configuration management for taskmaster.
package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config holds all taskmaster configuration.
type Config struct {
	Todoist  TodoistConfig `mapstructure:"todoist"`
	Slack    SlackConfig   `mapstructure:"slack"`
	Jira     JiraConfig    `mapstructure:"jira"`
	Notion   NotionConfig  `mapstructure:"notion"`
	Gemini   GeminiConfig  `mapstructure:"gemini"`
	Git      GitConfig     `mapstructure:"git"`
	CacheDir string        `mapstructure:"cache_dir"`
}

// GitConfig holds git commit history configuration.
type GitConfig struct {
	ScanDirs []string `mapstructure:"scan_dirs"`
	Author   string   `mapstructure:"author"`
}

// TodoistConfig holds Todoist API configuration.
type TodoistConfig struct {
	APIToken       string `mapstructure:"api_token"` //nolint:gosec // not a hardcoded credential
	StandupProject string `mapstructure:"standup_project"`
}

// SlackConfig holds Slack API configuration.
type SlackConfig struct {
	UserToken string `mapstructure:"user_token"`
	UserID    string `mapstructure:"user_id"`
}

// JiraConfig holds Jira API configuration.
type JiraConfig struct {
	BaseURL  string `mapstructure:"base_url"`
	Email    string `mapstructure:"email"`
	APIToken string `mapstructure:"api_token"` //nolint:gosec // not a hardcoded credential
}

// NotionConfig holds Notion API configuration.
type NotionConfig struct {
	APIToken   string `mapstructure:"api_token"` //nolint:gosec // not a hardcoded credential
	DatabaseID string `mapstructure:"database_id"`
}

// GeminiConfig holds Gemini API configuration.
type GeminiConfig struct {
	APIKey             string `mapstructure:"api_token"` //nolint:gosec // not a hardcoded credential
	Model              string `mapstructure:"model"`
	ConversationLogDir string `mapstructure:"conversation_log_dir"`
}

// LoadOption is a function that can be used to configure the viper instance.
type LoadOption func(*viper.Viper) error

// WithFlags binds command-line flags to the viper instance.
func WithFlags(flags *pflag.FlagSet) LoadOption {
	return func(v *viper.Viper) error {
		return v.BindPFlags(flags)
	}
}

// WithConfigFile sets an explicit path to the config file to load.
func WithConfigFile(path string) LoadOption {
	return func(v *viper.Viper) error {
		v.SetConfigFile(path)
		return nil
	}
}

// Load reads configuration from viper and returns a populated Config.
func Load(opts ...LoadOption) (*Config, error) {
	v := viper.New()

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	configDir := filepath.Join(home, ".config", "taskmaster")
	v.SetConfigName("taskmaster")
	v.SetConfigType("yaml")
	v.AddConfigPath(configDir)
	v.AddConfigPath(".")

	v.SetDefault("todoist.standup_project", "Work")
	v.SetDefault("gemini.model", "gemini-3.1-flash-lite-preview")
	v.SetDefault("gemini.conversation_log_dir", filepath.Join(configDir, "conversations"))
	v.SetDefault("cache_dir", filepath.Join(configDir, "cache"))

	for _, opt := range opts {
		if err := opt(v); err != nil {
			return nil, err
		}
	}

	err = v.ReadInConfig()
	if err != nil {
		return nil, err
	}

	cfg := &Config{}

	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
