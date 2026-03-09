// Package formatter transforms cached data items into structured prompts for LLM consumption.
package formatter

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kalverra/taskmaster/internal/source"
)

// FormatStandup builds a prompt for generating a daily standup update.
// It dynamically determines the last workday from completed tasks, so
// Monday or post-vacation standups look back to the most recent day
// with completed work instead of an empty yesterday.
func FormatStandup(items []source.DataItem) string {
	grouped := groupBySource(items)
	now := time.Now()

	lastWorkday := lastCompletedTaskDay(grouped["task_completed"], now)
	completedLabel, doneDescription := completedSectionMeta(lastWorkday, now)

	var b strings.Builder
	b.WriteString("You are an assistant that generates concise daily standup updates.\n\n")
	fmt.Fprintf(&b, "Today is %s.\n\n", now.Format("Monday, January 2, 2006"))
	b.WriteString("Based on the following activity data, generate a standup update with two sections:\n")
	fmt.Fprintf(&b, "1. **What I did %s** - summarize completed and progressed work\n", doneDescription)
	b.WriteString("2. **What I'm going to do today** - infer from active tasks, priorities, and due dates\n\n")
	b.WriteString(
		"3. **What's blocking me?** - list any blockers or dependencies, say just 'None' if there are none\n\n",
	)
	b.WriteString(
		"Be specific but concise. Use bullet points. For tasks that have Jira ticket links, reference them first like so: [TASK-123](https://your-jira-instance.com/browse/TASK-123): I accomplished x and y.\n\n",
	)
	b.WriteString("---\n\n")

	writeSection(&b, completedLabel, filterByTypeAndTime(grouped, "task_completed", lastWorkday, now))
	writeSection(&b, "Active Tasks", filterByType(grouped, "task_active"))
	writeSection(&b, "Conversations", filterByType(grouped, "conversation"))
	writeSection(&b, "Journal Entries", filterByType(grouped, "journal"))

	return b.String()
}

// lastCompletedTaskDay returns the start-of-day (local time) of the most
// recent completed task. If there are no completed tasks, it falls back
// to yesterday.
func lastCompletedTaskDay(completed []source.DataItem, now time.Time) time.Time {
	var latest time.Time
	for _, item := range completed {
		if item.Timestamp.After(latest) {
			latest = item.Timestamp
		}
	}

	if latest.IsZero() {
		return now.AddDate(0, 0, -1)
	}

	y, m, d := latest.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, latest.Location())
}

// completedSectionMeta returns a section header label and a prompt
// description fragment based on how far back the last workday was.
func completedSectionMeta(lastWorkday, now time.Time) (label, description string) {
	yesterdayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -1)

	if !lastWorkday.Before(yesterdayStart) {
		return "Completed Tasks (Yesterday)", "yesterday"
	}

	dayName := lastWorkday.Format("Monday")
	dateFmt := lastWorkday.Format("January 2")
	return fmt.Sprintf("Completed Tasks (since %s, %s)", dayName, dateFmt),
		fmt.Sprintf("since %s, %s", dayName, dateFmt)
}

// FormatSummary builds a prompt for generating a high-level summary of accomplishments.
func FormatSummary(items []source.DataItem, rangeLabel string) string {
	grouped := groupBySource(items)

	var b strings.Builder
	b.WriteString("You are an assistant that generates high-level work summaries.\n\n")
	fmt.Fprintf(&b, "Generate a summary of accomplishments for the past %s.\n", rangeLabel)
	b.WriteString("Organize by theme or project. Highlight key achievements and milestones.\n")
	b.WriteString(
		"Stay concise. Your audience is a direct or skip-level manager.\n\n",
	)
	b.WriteString("---\n\n")

	writeSection(&b, "Completed Tasks", filterByType(grouped, "task_completed"))
	writeSection(&b, "Active Tasks", filterByType(grouped, "task_active"))
	writeSection(&b, "Conversations", filterByType(grouped, "conversation"))
	writeSection(&b, "Journal Entries", filterByType(grouped, "journal"))

	return b.String()
}

// FormatSuggestions builds a prompt for generating prioritization and time allocation advice.
func FormatSuggestions(items []source.DataItem) string {
	grouped := groupBySource(items)
	now := time.Now()

	var b strings.Builder
	b.WriteString("You are a productivity coach that helps with task prioritization and time management.\n\n")
	fmt.Fprintf(&b, "Today is %s.\n\n", now.Format("Monday, January 2, 2006"))
	b.WriteString(
		"Based on the following data about my active tasks, recent completions, and conversations:\n",
	)
	b.WriteString("1. Suggest what I should work on next, ordered by priority\n")
	b.WriteString("2. Recommend how I should allocate my time today (in approximate hour blocks)\n")
	b.WriteString("3. Flag any tasks that seem overdue or at risk\n\n")
	b.WriteString("Consider due dates, task priorities, project groupings, and recent momentum.\n\n")
	b.WriteString("---\n\n")

	writeSection(&b, "Active Tasks", filterByType(grouped, "task_active"))
	writeSection(&b, "Recently Completed", filterByType(grouped, "task_completed"))
	writeSection(&b, "Conversations", filterByType(grouped, "conversation"))
	writeSection(&b, "Journal Entries", filterByType(grouped, "journal"))

	return b.String()
}

type groupedItems map[string][]source.DataItem

func groupBySource(items []source.DataItem) groupedItems {
	g := make(groupedItems)
	for _, item := range items {
		g[item.Type] = append(g[item.Type], item)
	}
	return g
}

func filterByType(g groupedItems, itemType string) []source.DataItem {
	items := g[itemType]
	sort.Slice(items, func(i, j int) bool {
		return items[i].Timestamp.Before(items[j].Timestamp)
	})
	return items
}

func filterByTypeAndTime(
	g groupedItems, itemType string, after, before time.Time,
) []source.DataItem {
	var filtered []source.DataItem
	for _, item := range g[itemType] {
		if (item.Timestamp.Equal(after) || item.Timestamp.After(after)) && item.Timestamp.Before(before) {
			filtered = append(filtered, item)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.Before(filtered[j].Timestamp)
	})
	return filtered
}

func writeSection(b *strings.Builder, title string, items []source.DataItem) {
	if len(items) == 0 {
		return
	}

	fmt.Fprintf(b, "## %s\n\n", title)
	for _, item := range items {
		fmt.Fprintf(b, "- **%s**", item.Title)
		if item.Content != "" {
			fmt.Fprintf(b, ": %s", item.Content)
		}

		var details []string
		if item.Source != "" {
			details = append(details, "source: "+item.Source)
		}
		if due, ok := item.Metadata["due"]; ok {
			details = append(details, "due: "+due)
		}
		if priority, ok := item.Metadata["priority"]; ok && priority != "1" {
			details = append(details, "priority: "+priority)
		}
		if labels, ok := item.Metadata["labels"]; ok {
			details = append(details, "labels: "+labels)
		}
		if !item.Timestamp.IsZero() {
			details = append(details, "at: "+item.Timestamp.Format(time.RFC822))
		}

		if len(details) > 0 {
			fmt.Fprintf(b, " (%s)", strings.Join(details, ", "))
		}
		b.WriteString("\n")

		if comments, ok := item.Metadata["comments"]; ok && comments != "" {
			for line := range strings.SplitSeq(comments, "\n") {
				fmt.Fprintf(b, "  - %s\n", line)
			}
		}
	}
	b.WriteString("\n")
}
