package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kalverra/taskmaster/internal/cache"
	"github.com/kalverra/taskmaster/internal/config"
	"github.com/kalverra/taskmaster/internal/formatter"
	"github.com/kalverra/taskmaster/internal/llm"
	"github.com/kalverra/taskmaster/internal/source"
)

var standupProject string

var standupCmd = &cobra.Command{
	Use:   "standup",
	Short: "Generate a daily standup update using your recent activity",
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		if cfg.Gemini.APIKey == "" {
			return fmt.Errorf(
				"gemini API key not configured; set TASKMASTER_GEMINI_API_KEY or gemini.api_key in config",
			)
		}

		project := standupProject
		if !cmd.Flags().Changed("project") {
			project = cfg.Todoist.StandupProject
		}

		c, err := cache.New(cfg.CacheDir)
		if err != nil {
			return fmt.Errorf("opening cache: %w", err)
		}
		defer c.Close() //nolint:errcheck // best-effort cleanup on exit

		tr := source.NewTimeRange("week")
		if err := ensureFreshCache(ctx, cfg, c, tr); err != nil {
			return err
		}

		items, err := c.Query("", tr)
		if err != nil {
			return fmt.Errorf("querying cache: %w", err)
		}

		if project != "" {
			items = filterByProject(items, project)
		}

		if len(items) == 0 {
			fmt.Println("No activity data found for the past week. Run 'taskmaster gather' first.")
			return nil
		}

		prompt := formatter.FormatStandup(items)

		ai, err := llm.New(ctx, cfg.Gemini.APIKey, cfg.Gemini.Model, cfg.Gemini.ConversationLogDir, "standup")
		if err != nil {
			return fmt.Errorf("initializing Gemini: %w", err)
		}

		sp := startSpinner("Generating standup...")
		result, err := ai.Generate(ctx, prompt)
		sp.Stop()
		if err != nil {
			return fmt.Errorf("generating standup: %w", err)
		}

		fmt.Println(result)
		return nil
	},
}

func init() {
	standupCmd.Flags().StringVarP(&standupProject, "project", "p", "Work", "Todoist project name to filter tasks by")
	rootCmd.AddCommand(standupCmd)
}

// filterByProject keeps all non-todoist items and only todoist items matching the given project name.
func filterByProject(items []source.DataItem, project string) []source.DataItem {
	var filtered []source.DataItem
	for _, item := range items {
		if item.Source != "todoist" || item.Metadata["project_name"] == project {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
