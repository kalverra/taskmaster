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

		c, err := cache.New(cfg.CacheDir)
		if err != nil {
			return fmt.Errorf("opening cache: %w", err)
		}
		defer c.Close() //nolint:errcheck // best-effort cleanup on exit

		tr := source.NewTimeRange("day")
		if err := ensureFreshCache(ctx, cfg, c, tr); err != nil {
			return err
		}

		items, err := c.Query("", tr)
		if err != nil {
			return fmt.Errorf("querying cache: %w", err)
		}

		if len(items) == 0 {
			fmt.Println("No activity data found for the past day. Run 'taskmaster gather' first.")
			return nil
		}

		prompt := formatter.FormatStandup(items)

		ai, err := llm.New(ctx, cfg.Gemini.APIKey, cfg.Gemini.Model)
		if err != nil {
			return fmt.Errorf("initializing Gemini: %w", err)
		}

		result, err := ai.Generate(ctx, prompt)
		if err != nil {
			return fmt.Errorf("generating standup: %w", err)
		}

		fmt.Println(result)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(standupCmd)
}
