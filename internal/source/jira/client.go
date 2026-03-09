// Package jira fetches issue data from the Jira Cloud REST API v3.
// API docs: https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issue-search/
package jira

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"resty.dev/v3"

	"github.com/kalverra/taskmaster/internal/source"
)

const sourceName = "jira"

// Client defines the interface for Jira API operations.
type Client interface {
	SearchJQL(
		ctx context.Context,
		jql string,
		fields []string,
		maxResults int,
		nextPageToken string,
	) (*SearchResponse, error)
	Close()
}

// restyClient implements Client using resty and the Jira REST API v3.
type restyClient struct {
	resty   *resty.Client
	baseURL string
	log     zerolog.Logger
}

// NewClient creates a Jira API client. baseURL should be like "https://yourorg.atlassian.net".
func NewClient(baseURL, email, apiToken string, log zerolog.Logger) Client {
	log = log.With().Str("source", sourceName).Logger()

	r := resty.New()
	r.SetBaseURL(baseURL).
		SetBasicAuth(email, apiToken).
		SetHeader("Accept", "application/json").
		SetTimeout(30 * time.Second).
		AddRequestMiddleware(source.RestyRequestLogger(log)).
		AddResponseMiddleware(source.RestyResponseLogger(log))
	return &restyClient{resty: r, baseURL: baseURL, log: log}
}

func (c *restyClient) Close() {
	_ = c.resty.Close()
}

// SearchJQL uses the enhanced JQL search endpoint: GET /rest/api/3/search/jql
func (c *restyClient) SearchJQL(
	ctx context.Context,
	jql string,
	fields []string,
	maxResults int,
	nextPageToken string,
) (*SearchResponse, error) {
	params := map[string]string{
		"jql":        jql,
		"maxResults": fmt.Sprintf("%d", maxResults),
	}
	if len(fields) > 0 {
		params["fields"] = strings.Join(fields, ",")
	}
	if nextPageToken != "" {
		params["nextPageToken"] = nextPageToken
	}

	c.log.Trace().
		Str("method", "GET").
		Str("endpoint", "/rest/api/3/search/jql").
		Str("jql", jql).
		Int("maxResults", maxResults).
		Str("nextPageToken", nextPageToken).
		Msg("jira API request")

	var result SearchResponse
	resp, err := c.resty.R().
		SetContext(ctx).
		SetQueryParams(params).
		SetResult(&result).
		Get("/rest/api/3/search/jql")
	if err != nil {
		return nil, fmt.Errorf("jira API request: %w", err)
	}

	c.log.Trace().
		Int("status", resp.StatusCode()).
		Int("issueCount", len(result.Issues)).
		Bool("isLast", result.IsLast).
		Msg("jira API response")

	if resp.IsError() {
		return nil, fmt.Errorf("jira API returned %d: %s", resp.StatusCode(), resp.String())
	}

	return &result, nil
}

// Source implements source.DataSource for Jira Cloud.
type Source struct {
	client  Client
	baseURL string
}

// New creates a Jira data source with the default resty-based client.
func New(baseURL, email, apiToken string) *Source {
	log := zerolog.Nop()
	return &Source{
		client:  NewClient(baseURL, email, apiToken, log),
		baseURL: baseURL,
	}
}

// NewWithClient creates a Jira data source with a provided client (useful for testing).
func NewWithClient(client Client, baseURL string) *Source {
	return &Source{client: client, baseURL: baseURL}
}

// Name returns the name of this data source.
func (s *Source) Name() string { return sourceName }

// Fetch retrieves issues the user interacted with within the given time range.
func (s *Source) Fetch(ctx context.Context, tr source.TimeRange) ([]source.DataItem, error) {
	jql := fmt.Sprintf(
		"assignee = currentUser() AND updated >= \"%s\" ORDER BY updated DESC",
		tr.Start.Format("2006-01-02"),
	)
	fields := []string{"summary", "status", "priority", "updated", "issuetype"}

	var allItems []source.DataItem
	var nextPageToken string

	for {
		sr, err := s.client.SearchJQL(ctx, jql, fields, 50, nextPageToken)
		if err != nil {
			return nil, err
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

		if sr.IsLast || sr.NextPageToken == "" {
			break
		}
		nextPageToken = sr.NextPageToken
	}

	return allItems, nil
}
