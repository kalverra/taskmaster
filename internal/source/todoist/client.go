// Package todoist fetches task activity from the Todoist API.
package todoist

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kalverra/taskmaster/internal/source"
)

const (
	restBaseURL = "https://api.todoist.com/rest/v2"
	syncBaseURL = "https://api.todoist.com/sync/v9"
	sourceName  = "todoist"

	completedPageSize = 200
)

// Source implements source.DataSource for Todoist.
type Source struct {
	apiToken   string
	httpClient *http.Client
}

// New creates a Todoist data source with the given API token.
func New(apiToken string) *Source {
	return &Source{
		apiToken:   apiToken,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the source identifier.
func (s *Source) Name() string { return sourceName }

// Fetch retrieves active and completed tasks from Todoist within the given time range.
func (s *Source) Fetch(ctx context.Context, tr source.TimeRange) ([]source.DataItem, error) {
	var allItems []source.DataItem

	active, err := s.fetchActiveTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching active tasks: %w", err)
	}
	allItems = append(allItems, active...)

	completed, err := s.fetchCompletedTasks(ctx, tr)
	if err != nil {
		return nil, fmt.Errorf("fetching completed tasks: %w", err)
	}
	allItems = append(allItems, completed...)

	return allItems, nil
}

type restTask struct {
	ID          string   `json:"id"`
	Content     string   `json:"content"`
	Description string   `json:"description"`
	ProjectID   string   `json:"project_id"`
	SectionID   string   `json:"section_id"`
	Priority    int      `json:"priority"`
	Labels      []string `json:"labels"`
	CreatedAt   string   `json:"created_at"`
	Due         *restDue `json:"due"`
	URL         string   `json:"url"`
}

type restDue struct {
	Date      string `json:"date"`
	Datetime  string `json:"datetime"`
	String    string `json:"string"`
	Recurring bool   `json:"is_recurring"`
}

func (s *Source) fetchActiveTasks(ctx context.Context) ([]source.DataItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, restBaseURL+"/tasks", nil)
	if err != nil {
		return nil, err
	}
	s.setAuth(req)

	resp, err := s.httpClient.Do(req) //nolint:gosec // URL is a known constant
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck // best-effort cleanup

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("todoist REST API returned %d: %s", resp.StatusCode, string(body))
	}

	var tasks []restTask
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return nil, fmt.Errorf("decoding active tasks: %w", err)
	}

	items := make([]source.DataItem, 0, len(tasks))
	for _, t := range tasks {
		created, _ := time.Parse(time.RFC3339, t.CreatedAt)
		meta := map[string]string{
			"project_id": t.ProjectID,
			"priority":   strconv.Itoa(t.Priority),
			"url":        t.URL,
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

		items = append(items, source.DataItem{
			ID:        t.ID,
			Source:    sourceName,
			Type:      "task_active",
			Title:     t.Content,
			Content:   t.Description,
			Metadata:  meta,
			Timestamp: created,
		})
	}

	return items, nil
}

type completedResponse struct {
	Items []completedItem `json:"items"`
}

type completedItem struct {
	ID          string `json:"id"`
	TaskID      string `json:"task_id"`
	Content     string `json:"content"`
	ProjectID   string `json:"project_id"`
	SectionID   string `json:"section_id"`
	CompletedAt string `json:"completed_at"`
}

func (s *Source) fetchCompletedTasks(ctx context.Context, tr source.TimeRange) ([]source.DataItem, error) {
	var allItems []source.DataItem
	offset := 0

	for {
		params := url.Values{}
		params.Set("since", tr.Start.Format("2006-1-2T15:04:05"))
		params.Set("until", tr.End.Format("2006-1-2T15:04:05"))
		params.Set("limit", strconv.Itoa(completedPageSize))
		params.Set("offset", strconv.Itoa(offset))

		reqURL := syncBaseURL + "/completed/get_all?" + params.Encode()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, err
		}
		s.setAuth(req)

		resp, err := s.httpClient.Do(req) //nolint:gosec // URL built from known constants
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close() //nolint:errcheck // best-effort cleanup

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("todoist Sync API returned %d: %s", resp.StatusCode, string(body))
		}

		var cr completedResponse
		if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
			return nil, fmt.Errorf("decoding completed tasks: %w", err)
		}

		for _, c := range cr.Items {
			completedAt, _ := time.Parse(time.RFC3339Nano, c.CompletedAt)
			allItems = append(allItems, source.DataItem{
				ID:     c.ID,
				Source: sourceName,
				Type:   "task_completed",
				Title:  c.Content,
				Metadata: map[string]string{
					"task_id":    c.TaskID,
					"project_id": c.ProjectID,
					"section_id": c.SectionID,
				},
				Timestamp: completedAt,
			})
		}

		if len(cr.Items) < completedPageSize {
			break
		}
		offset += completedPageSize
	}

	return allItems, nil
}

func (s *Source) setAuth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+s.apiToken)
}
