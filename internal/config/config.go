// Package config provides configuration management for taskmaster.
package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds all taskmaster configuration.
type Config struct {
	Todoist  TodoistConfig `mapstructure:"todoist"`
	Slack    SlackConfig   `mapstructure:"slack"`
	Jira     JiraConfig    `mapstructure:"jira"`
	Notion   NotionConfig  `mapstructure:"notion"`
	Gemini   GeminiConfig  `mapstructure:"gemini"`
	CacheDir string        `mapstructure:"cache_dir"`
}

// TodoistConfig holds Todoist API configuration.
type TodoistConfig struct {
	APIToken string `mapstructure:"api_token"` //nolint:gosec // not a hardcoded credential
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
	APIKey string `mapstructure:"api_key"` //nolint:gosec // not a hardcoded credential
	Model  string `mapstructure:"model"`
}

// Load reads configuration from viper and returns a populated Config.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	if cfg.CacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		cfg.CacheDir = filepath.Join(home, ".taskmaster")
	}

	if cfg.Gemini.Model == "" {
		cfg.Gemini.Model = "gemini-2.0-flash"
	}

	return cfg, nil
}
