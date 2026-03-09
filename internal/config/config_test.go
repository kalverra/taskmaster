package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad_Valid(t *testing.T) {
	t.Parallel()
	cfg, err := Load(WithConfigFile("testdata/taskmaster.valid.yaml"))
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Equal(t, "gemini-api-token", cfg.Gemini.APIKey)
	require.Equal(t, "gemini-3.1-flash-lite-preview", cfg.Gemini.Model)
	require.Equal(t, "todoist-api-token", cfg.Todoist.APIToken)
	require.Equal(t, "slack-user-token", cfg.Slack.UserToken)
	require.Equal(t, "slack-user-id", cfg.Slack.UserID)
	require.Equal(t, "https://jira-base-url", cfg.Jira.BaseURL)
	require.Equal(t, "you@example.com", cfg.Jira.Email)
	require.Equal(t, "jira-api-token", cfg.Jira.APIToken)
}
