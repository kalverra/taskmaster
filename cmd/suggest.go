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

var suggestCmd = &cobra.Command{
	Use:   "suggest",
	Short: "Get AI-powered suggestions on what to work on next and how to allocate your time",
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

		// Use a week of context for better suggestions
		tr := source.NewTimeRange("week")
		if err := ensureFreshCache(ctx, cfg, c, tr); err != nil {
			return err
		}

		items, err := c.Query("", tr)
		if err != nil {
			return fmt.Errorf("querying cache: %w", err)
		}

		if len(items) == 0 {
			fmt.Println("No activity data found. Run 'taskmaster gather' first.")
			return nil
		}

		prompt := formatter.FormatSuggestions(items)

		ai, err := llm.New(ctx, cfg.Gemini.APIKey, cfg.Gemini.Model, cfg.Gemini.ConversationLogDir, "suggest")
		if err != nil {
			return fmt.Errorf("initializing Gemini: %w", err)
		}

		sp := startSpinner("Generating suggestions...")
		result, err := ai.Generate(ctx, prompt)
		sp.Stop()
		if err != nil {
			return fmt.Errorf("generating suggestions: %w", err)
		}

		fmt.Println(result)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(suggestCmd)
}
