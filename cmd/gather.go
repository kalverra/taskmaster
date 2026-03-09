package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/kalverra/taskmaster/internal/cache"
	"github.com/kalverra/taskmaster/internal/config"
	"github.com/kalverra/taskmaster/internal/source"
	"github.com/kalverra/taskmaster/internal/source/jira"
	"github.com/kalverra/taskmaster/internal/source/slack"
	"github.com/kalverra/taskmaster/internal/source/todoist"
)

var gatherRange string

var gatherCmd = &cobra.Command{
	Use:   "gather",
	Short: "Fetch activity data from all configured sources and cache it locally",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runGather(cmd.Context(), gatherRange)
	},
}

func init() {
	gatherCmd.Flags().StringVarP(&gatherRange, "range", "r", "day", "Time range to fetch: day, week, month, quarter")
	rootCmd.AddCommand(gatherCmd)
}

func runGather(ctx context.Context, rangeLabel string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	c, err := cache.New(cfg.CacheDir)
	if err != nil {
		return fmt.Errorf("opening cache: %w", err)
	}
	defer c.Close() //nolint:errcheck // best-effort cleanup on exit

	tr := source.NewTimeRange(rangeLabel)
	sources := buildSources(cfg)

	if len(sources) == 0 {
		return fmt.Errorf(
			"no data sources configured; set API tokens in ~/.config/taskmaster/taskmaster.yaml or environment variables",
		)
	}

	for _, src := range sources {
		sp := startSpinner(fmt.Sprintf("Gathering from %s (%s)...", src.Name(), rangeLabel))
		items, fetchErr := src.Fetch(ctx, tr)
		if fetchErr != nil {
			sp.Stop()
			fmt.Printf("  Warning: %s fetch failed: %v\n", src.Name(), fetchErr)
			continue
		}

		if storeErr := c.Store(items); storeErr != nil {
			sp.Stop()
			return fmt.Errorf("caching %s data: %w", src.Name(), storeErr)
		}
		sp.FinalMSG = fmt.Sprintf("  Cached %d items from %s\n", len(items), src.Name())
		sp.Stop()
	}

	return nil
}

func buildSources(cfg *config.Config) []source.DataSource {
	var sources []source.DataSource

	if cfg.Todoist.APIToken != "" {
		sources = append(sources, todoist.New(cfg.Todoist.APIToken))
	}
	if cfg.Slack.UserToken != "" && cfg.Slack.UserID != "" {
		sources = append(sources, slack.New(cfg.Slack.UserToken, cfg.Slack.UserID))
	}
	if cfg.Jira.BaseURL != "" && cfg.Jira.APIToken != "" {
		sources = append(sources, jira.New(cfg.Jira.BaseURL, cfg.Jira.Email, cfg.Jira.APIToken))
	}

	return sources
}

// ensureFreshCache gathers data if the cache is stale for the given time range.
func ensureFreshCache(ctx context.Context, cfg *config.Config, c *cache.Cache, tr source.TimeRange) error {
	staleness := 1 * time.Hour
	sources := buildSources(cfg)

	for _, src := range sources {
		fresh, err := c.IsFresh(src.Name(), tr, staleness)
		if err != nil {
			return fmt.Errorf("checking cache freshness for %s: %w", src.Name(), err)
		}
		if fresh {
			continue
		}

		sp := startSpinner(fmt.Sprintf("Gathering from %s...", src.Name()))
		items, fetchErr := src.Fetch(ctx, tr)
		if fetchErr != nil {
			sp.Stop()
			fmt.Printf("  Warning: %s fetch failed: %v\n", src.Name(), fetchErr)
			continue
		}
		if storeErr := c.Store(items); storeErr != nil {
			sp.Stop()
			return fmt.Errorf("caching %s data: %w", src.Name(), storeErr)
		}
		sp.FinalMSG = fmt.Sprintf("  Cached %d items from %s\n", len(items), src.Name())
		sp.Stop()
	}

	return nil
}
