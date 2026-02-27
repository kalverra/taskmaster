// Package jira fetches issue data from the Jira REST API.
package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/kalverra/taskmaster/internal/source"
)

const sourceName = "jira"

// Source implements source.DataSource for Jira Cloud.
type Source struct {
	baseURL    string
	email      string
	apiToken   string
	httpClient *http.Client
}

// New creates a Jira data source. baseURL should be like "https://yourorg.atlassian.net".
func New(baseURL, email, apiToken string) *Source {
	return &Source{
		baseURL:    baseURL,
		email:      email,
		apiToken:   apiToken,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the source identifier.
func (s *Source) Name() string { return sourceName }

// Fetch retrieves issues the user interacted with within the given time range.
func (s *Source) Fetch(ctx context.Context, tr source.TimeRange) ([]source.DataItem, error) {
	jql := fmt.Sprintf(
		"assignee = currentUser() AND updated >= \"%s\" ORDER BY updated DESC",
		tr.Start.Format("2006-01-02"),
	)

	var allItems []source.DataItem
	startAt := 0

	for {
		params := url.Values{}
		params.Set("jql", jql)
		params.Set("maxResults", "50")
		params.Set("startAt", fmt.Sprintf("%d", startAt))
		params.Set("fields", "summary,status,priority,updated,created,assignee,labels,issuetype")

		reqURL := s.baseURL + "/rest/api/3/search?" + params.Encode()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, err
		}
		req.SetBasicAuth(s.email, s.apiToken)
		req.Header.Set("Accept", "application/json")

		resp, err := s.httpClient.Do(req) //nolint:gosec // URL from user config
		if err != nil {
			return nil, fmt.Errorf("jira API request: %w", err)
		}
		defer resp.Body.Close() //nolint:errcheck // best-effort cleanup

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("jira API returned %d: %s", resp.StatusCode, string(body))
		}

		var sr searchResponse
		if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
			return nil, fmt.Errorf("decoding jira response: %w", err)
		}

		for _, issue := range sr.Issues {
			updated, _ := time.Parse("2006-01-02T15:04:05.000-0700", issue.Fields.Updated)
			itemType := "task_active"
			if issue.Fields.Status.Category.Key == "done" {
				itemType = "task_completed"
			}

			meta := map[string]string{
				"key":        issue.Key,
				"status":     issue.Fields.Status.Name,
				"issue_type": issue.Fields.IssueType.Name,
				"url":        s.baseURL + "/browse/" + issue.Key,
			}
			if issue.Fields.Priority.Name != "" {
				meta["priority"] = issue.Fields.Priority.Name
			}

			allItems = append(allItems, source.DataItem{
				ID:        issue.ID,
				Source:    sourceName,
				Type:      itemType,
				Title:     fmt.Sprintf("[%s] %s", issue.Key, issue.Fields.Summary),
				Metadata:  meta,
				Timestamp: updated,
			})
		}

		if startAt+len(sr.Issues) >= sr.Total || len(sr.Issues) == 0 {
			break
		}
		startAt += len(sr.Issues)
	}

	return allItems, nil
}

type searchResponse struct {
	Total  int     `json:"total"`
	Issues []issue `json:"issues"`
}

type issue struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Fields struct {
		Summary  string `json:"summary"`
		Updated  string `json:"updated"`
		Status   status `json:"status"`
		Priority struct {
			Name string `json:"name"`
		} `json:"priority"`
		IssueType struct {
			Name string `json:"name"`
		} `json:"issuetype"`
	} `json:"fields"`
}

type status struct {
	Name     string `json:"name"`
	Category struct {
		Key string `json:"key"`
	} `json:"statusCategory"`
}
