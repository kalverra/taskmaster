// Package source defines the common interface and types for activity data sources.
package source

import (
	"context"
	"time"
)

// DataSource represents a provider of activity data (e.g. Todoist, Slack).
type DataSource interface {
	Name() string
	Fetch(ctx context.Context, tr TimeRange) ([]DataItem, error)
}

// TimeRange represents a bounded window of time with a human-readable label.
type TimeRange struct {
	Start time.Time
	End   time.Time
	Label string // "day", "week", "month", "quarter"
}

// NewTimeRange creates a TimeRange ending now with the given label.
func NewTimeRange(label string) TimeRange {
	now := time.Now()
	var start time.Time

	switch label {
	case "day":
		start = now.AddDate(0, 0, -1)
	case "week":
		start = now.AddDate(0, 0, -7)
	case "month":
		start = now.AddDate(0, -1, 0)
	case "quarter":
		start = now.AddDate(0, -3, 0)
	default:
		start = now.AddDate(0, 0, -1)
		label = "day"
	}

	return TimeRange{Start: start, End: now, Label: label}
}

// DataItem represents a single piece of activity data from any source.
type DataItem struct {
	ID       string
	Source   string
	Type     string // "task_completed", "task_created", "task_updated", "conversation", "journal"
	Title    string
	Content  string
	Metadata map[string]string
	// Timestamp is when the activity occurred.
	Timestamp time.Time
}
