// Package todoist fetches task activity from the Todoist REST API v1.
// API docs: https://developer.todoist.com/api/v1/
package todoist

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"resty.dev/v3"

	"github.com/kalverra/taskmaster/internal/source"
)

const (
	apiBaseURL = "https://api.todoist.com/api/v1"
	sourceName = "todoist"

	defaultPageSize = 200
)

// Client defines the interface for Todoist API operations.
type Client interface {
	GetProjects(ctx context.Context, cursor string, limit int) (*ProjectsResponse, error)
	GetSections(ctx context.Context, cursor string, limit int) (*SectionsResponse, error)
	GetActiveTasks(ctx context.Context, cursor string, limit int) (*TasksResponse, error)
	GetCompletedTasks(ctx context.Context, since, until, cursor string, limit int) (*CompletedTasksResponse, error)
	GetComments(ctx context.Context, taskID, cursor string, limit int) (*CommentsResponse, error)
	Close()
}

// restyClient implements Client using resty and the Todoist REST API v1.
type restyClient struct {
	resty *resty.Client
	log   zerolog.Logger
}

// NewClient creates a Todoist API client.
func NewClient(apiToken string, log zerolog.Logger) Client {
	log = log.With().Str("source", sourceName).Logger()

	r := resty.New()
	r.SetBaseURL(apiBaseURL).
		SetAuthToken(apiToken).
		SetHeader("Accept", "application/json").
		SetTimeout(30 * time.Second).
		AddRequestMiddleware(source.RestyRequestLogger(log)).
		AddResponseMiddleware(source.RestyResponseLogger(log))
	return &restyClient{resty: r, log: log}
}

func (c *restyClient) Close() {
	_ = c.resty.Close()
}

// GetProjects calls GET /api/v1/projects with cursor-based pagination.
func (c *restyClient) GetProjects(ctx context.Context, cursor string, limit int) (*ProjectsResponse, error) {
	params := map[string]string{
		"limit": strconv.Itoa(limit),
	}
	if cursor != "" {
		params["cursor"] = cursor
	}

	var result ProjectsResponse
	resp, err := c.resty.R().
		SetContext(ctx).
		SetQueryParams(params).
		SetResult(&result).
		Get("/projects")
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("failed to get projects: %d: %s", resp.StatusCode(), resp.String())
	}

	return &result, nil
}

// GetSections calls GET /api/v1/sections with cursor-based pagination.
func (c *restyClient) GetSections(ctx context.Context, cursor string, limit int) (*SectionsResponse, error) {
	params := map[string]string{
		"limit": strconv.Itoa(limit),
	}
	if cursor != "" {
		params["cursor"] = cursor
	}

	var result SectionsResponse
	resp, err := c.resty.R().
		SetContext(ctx).
		SetQueryParams(params).
		SetResult(&result).
		Get("/sections")
	if err != nil {
		return nil, fmt.Errorf("failed to get sections: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("failed to get sections: %d: %s", resp.StatusCode(), resp.String())
	}

	return &result, nil
}

// GetActiveTasks calls GET /api/v1/tasks with cursor-based pagination.
func (c *restyClient) GetActiveTasks(ctx context.Context, cursor string, limit int) (*TasksResponse, error) {
	params := map[string]string{
		"limit": strconv.Itoa(limit),
	}
	if cursor != "" {
		params["cursor"] = cursor
	}

	var result TasksResponse
	resp, err := c.resty.R().
		SetContext(ctx).
		SetQueryParams(params).
		SetResult(&result).
		Get("/tasks")
	if err != nil {
		return nil, fmt.Errorf("failed to get active tasks: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("failed to get active tasks: %d: %s", resp.StatusCode(), resp.String())
	}

	return &result, nil
}

// GetCompletedTasks calls GET /api/v1/tasks/completed/by_completion_date.
func (c *restyClient) GetCompletedTasks(
	ctx context.Context,
	since, until, cursor string,
	limit int,
) (*CompletedTasksResponse, error) {
	params := map[string]string{
		"since": since,
		"until": until,
		"limit": strconv.Itoa(limit),
	}
	if cursor != "" {
		params["cursor"] = cursor
	}

	var result CompletedTasksResponse
	resp, err := c.resty.R().
		SetContext(ctx).
		SetQueryParams(params).
		SetResult(&result).
		Get("/tasks/completed/by_completion_date")
	if err != nil {
		return nil, fmt.Errorf("failed to get completed tasks: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("failed to get completed tasks: %d: %s", resp.StatusCode(), resp.String())
	}

	return &result, nil
}

// GetComments calls GET /api/v1/comments with cursor-based pagination for a specific task.
func (c *restyClient) GetComments(ctx context.Context, taskID, cursor string, limit int) (*CommentsResponse, error) {
	params := map[string]string{
		"task_id": taskID,
		"limit":   strconv.Itoa(limit),
	}
	if cursor != "" {
		params["cursor"] = cursor
	}

	var result CommentsResponse
	resp, err := c.resty.R().
		SetContext(ctx).
		SetQueryParams(params).
		SetResult(&result).
		Get("/comments")
	if err != nil {
		return nil, fmt.Errorf("failed to get comments for task %s: %w", taskID, err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("failed to get comments for task %s: %d: %s", taskID, resp.StatusCode(), resp.String())
	}

	return &result, nil
}

// Source implements source.DataSource for Todoist.
type Source struct {
	client Client
}

// New creates a Todoist data source with the default resty-based client.
func New(apiToken string) *Source {
	log := zerolog.Nop()
	return &Source{client: NewClient(apiToken, log)}
}

// NewWithClient creates a Todoist data source with a provided client (useful for testing).
func NewWithClient(client Client) *Source {
	return &Source{client: client}
}

// Name returns the name of this data source.
func (s *Source) Name() string { return sourceName }

// Fetch retrieves active and completed tasks from Todoist within the given time range.
func (s *Source) Fetch(ctx context.Context, tr source.TimeRange) ([]source.DataItem, error) {
	projectMap, err := s.fetchProjectMap(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching projects: %w", err)
	}

	sectionMap, err := s.fetchSectionMap(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching sections: %w", err)
	}

	var allItems []source.DataItem

	active, err := s.fetchActiveTasks(ctx, projectMap, sectionMap)
	if err != nil {
		return nil, fmt.Errorf("fetching active tasks: %w", err)
	}
	allItems = append(allItems, active...)

	completed, err := s.fetchCompletedTasks(ctx, tr, projectMap, sectionMap)
	if err != nil {
		return nil, fmt.Errorf("fetching completed tasks: %w", err)
	}
	allItems = append(allItems, completed...)

	return allItems, nil
}

func (s *Source) fetchProjectMap(ctx context.Context) (map[string]string, error) {
	projects := make(map[string]string)
	var cursor string

	for {
		resp, err := s.client.GetProjects(ctx, cursor, defaultPageSize)
		if err != nil {
			return nil, err
		}

		for _, p := range resp.Results {
			projects[p.ID] = p.Name
		}

		if resp.NextCursor == nil || *resp.NextCursor == "" {
			break
		}
		cursor = *resp.NextCursor
	}

	return projects, nil
}

func (s *Source) fetchSectionMap(ctx context.Context) (map[string]string, error) {
	sections := make(map[string]string)
	var cursor string

	for {
		resp, err := s.client.GetSections(ctx, cursor, defaultPageSize)
		if err != nil {
			return nil, err
		}

		for _, sec := range resp.Results {
			sections[sec.ID] = sec.Name
		}

		if resp.NextCursor == nil || *resp.NextCursor == "" {
			break
		}
		cursor = *resp.NextCursor
	}

	return sections, nil
}

// fetchComments retrieves all comments for a task and returns them as a
// newline-separated string with each line formatted as "YYYY-MM-DD: content".
// Returns an empty string if the task has no comments.
func (s *Source) fetchComments(ctx context.Context, taskID string) (string, error) {
	var lines []string
	var cursor string

	for {
		resp, err := s.client.GetComments(ctx, taskID, cursor, defaultPageSize)
		if err != nil {
			return "", err
		}

		for _, c := range resp.Results {
			datePart := c.PostedAt
			if t, err := time.Parse(time.RFC3339, c.PostedAt); err == nil {
				datePart = t.Local().Format("2006-01-02")
			}
			lines = append(lines, datePart+": "+c.Content)
		}

		if resp.NextCursor == nil || *resp.NextCursor == "" {
			break
		}
		cursor = *resp.NextCursor
	}

	return strings.Join(lines, "\n"), nil
}

func (s *Source) fetchActiveTasks(
	ctx context.Context,
	projectMap, sectionMap map[string]string,
) ([]source.DataItem, error) {
	var allItems []source.DataItem
	var cursor string

	for {
		resp, err := s.client.GetActiveTasks(ctx, cursor, defaultPageSize)
		if err != nil {
			return nil, err
		}

		for _, t := range resp.Results {
			addedAt, _ := time.Parse(time.RFC3339, t.AddedAt)
			meta := map[string]string{
				"project_id":   t.ProjectID,
				"project_name": projectMap[t.ProjectID],
				"priority":     strconv.Itoa(t.Priority),
			}
			if name, ok := sectionMap[t.SectionID]; ok && name != "" {
				meta["section_name"] = name
			}
			if len(t.Labels) > 0 {
				meta["labels"] = strings.Join(t.Labels, ",")
			}
			if t.Due != nil {
				meta["due"] = t.Due.String
				if t.Due.Recurring {
					meta["recurring"] = "true"
				}
			}

			if comments, err := s.fetchComments(ctx, t.ID); err == nil && comments != "" {
				meta["comments"] = comments
			}

			allItems = append(allItems, source.DataItem{
				ID:        t.ID,
				Source:    sourceName,
				Type:      "task_active",
				Title:     t.Content,
				Content:   t.Description,
				Metadata:  meta,
				Timestamp: addedAt,
			})
		}

		if resp.NextCursor == nil || *resp.NextCursor == "" {
			break
		}
		cursor = *resp.NextCursor
	}

	return allItems, nil
}

func (s *Source) fetchCompletedTasks(
	ctx context.Context,
	tr source.TimeRange,
	projectMap, sectionMap map[string]string,
) ([]source.DataItem, error) {
	since := tr.Start.Format(time.RFC3339)
	until := tr.End.Format(time.RFC3339)

	var allItems []source.DataItem
	var cursor string

	for {
		resp, err := s.client.GetCompletedTasks(ctx, since, until, cursor, defaultPageSize)
		if err != nil {
			return nil, err
		}

		for _, t := range resp.Items {
			var completedAt time.Time
			if t.CompletedAt != nil {
				completedAt, _ = time.Parse(time.RFC3339, *t.CompletedAt)
			}

			meta := map[string]string{
				"project_id":   t.ProjectID,
				"project_name": projectMap[t.ProjectID],
			}
			if name, ok := sectionMap[t.SectionID]; ok && name != "" {
				meta["section_name"] = name
			}

			if comments, err := s.fetchComments(ctx, t.ID); err == nil && comments != "" {
				meta["comments"] = comments
			}

			allItems = append(allItems, source.DataItem{
				ID:        t.ID,
				Source:    sourceName,
				Type:      "task_completed",
				Title:     t.Content,
				Metadata:  meta,
				Timestamp: completedAt,
			})
		}

		if resp.NextCursor == nil || *resp.NextCursor == "" {
			break
		}
		cursor = *resp.NextCursor
	}

	return allItems, nil
}
