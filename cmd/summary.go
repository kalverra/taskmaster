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

var summaryRange string

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Generate a high-level summary of your accomplishments",
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

		tr := source.NewTimeRange(summaryRange)
		if err := ensureFreshCache(ctx, cfg, c, tr); err != nil {
			return err
		}

		items, err := c.Query("", tr)
		if err != nil {
			return fmt.Errorf("querying cache: %w", err)
		}

		if len(items) == 0 {
			fmt.Printf("No activity data found for the past %s. Run 'taskmaster gather' first.\n", summaryRange)
			return nil
		}

		prompt := formatter.FormatSummary(items, summaryRange)

		ai, err := llm.New(ctx, cfg.Gemini.APIKey, cfg.Gemini.Model)
		if err != nil {
			return fmt.Errorf("initializing Gemini: %w", err)
		}

		result, err := ai.Generate(ctx, prompt)
		if err != nil {
			return fmt.Errorf("generating summary: %w", err)
		}

		fmt.Println(result)
		return nil
	},
}

func init() {
	summaryCmd.Flags().StringVarP(&summaryRange, "range", "r", "week", "Time range: week, month, quarter")
	rootCmd.AddCommand(summaryCmd)
}
