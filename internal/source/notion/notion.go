// Package notion fetches journal entries from a Notion database.
package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kalverra/taskmaster/internal/source"
)

const (
	apiBaseURL = "https://api.notion.com/v1"
	sourceName = "notion"
	apiVersion = "2022-06-28"
)

// Source implements source.DataSource for Notion.
type Source struct {
	apiToken   string
	databaseID string
	httpClient *http.Client
}

// New creates a Notion data source. databaseID is the ID of the journal entries database.
func New(apiToken, databaseID string) *Source {
	return &Source{
		apiToken:   apiToken,
		databaseID: databaseID,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the source identifier.
func (s *Source) Name() string { return sourceName }

// Fetch retrieves journal entries from Notion within the given time range.
func (s *Source) Fetch(ctx context.Context, tr source.TimeRange) ([]source.DataItem, error) {
	filter := map[string]any{
		"filter": map[string]any{
			"and": []map[string]any{
				{
					"property": "Date",
					"date": map[string]string{
						"on_or_after": tr.Start.Format("2006-01-02"),
					},
				},
				{
					"property": "Date",
					"date": map[string]string{
						"on_or_before": tr.End.Format("2006-01-02"),
					},
				},
			},
		},
		"sorts": []map[string]string{
			{"property": "Date", "direction": "ascending"},
		},
	}

	var allItems []source.DataItem
	var startCursor string

	for {
		if startCursor != "" {
			filter["start_cursor"] = startCursor
		}

		body, err := json.Marshal(filter)
		if err != nil {
			return nil, fmt.Errorf("marshaling query: %w", err)
		}

		reqURL := fmt.Sprintf("%s/databases/%s/query", apiBaseURL, s.databaseID)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		s.setHeaders(req)

		resp, err := s.httpClient.Do(req) //nolint:gosec // URL from known constants + config
		if err != nil {
			return nil, fmt.Errorf("notion API request: %w", err)
		}
		defer resp.Body.Close() //nolint:errcheck // best-effort cleanup

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("notion API returned %d: %s", resp.StatusCode, string(respBody))
		}

		var qr queryResponse
		if err := json.NewDecoder(resp.Body).Decode(&qr); err != nil {
			return nil, fmt.Errorf("decoding notion response: %w", err)
		}

		for _, page := range qr.Results {
			item := pageToDataItem(page)
			allItems = append(allItems, item)
		}

		if !qr.HasMore || qr.NextCursor == "" {
			break
		}
		startCursor = qr.NextCursor
	}

	return allItems, nil
}

func (s *Source) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+s.apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", apiVersion)
}

func pageToDataItem(page page) source.DataItem {
	title := extractTitle(page.Properties)
	date := extractDate(page.Properties)
	content := extractRichText(page.Properties, "Content")

	ts := date
	if ts.IsZero() {
		ts, _ = time.Parse(time.RFC3339, page.CreatedTime)
	}

	return source.DataItem{
		ID:      page.ID,
		Source:  sourceName,
		Type:    "journal",
		Title:   title,
		Content: content,
		Metadata: map[string]string{
			"url": page.URL,
		},
		Timestamp: ts,
	}
}

func extractTitle(props map[string]property) string {
	for _, prop := range props {
		if prop.Type == "title" && len(prop.Title) > 0 {
			var parts []string
			for _, t := range prop.Title {
				parts = append(parts, t.PlainText)
			}
			return strings.Join(parts, "")
		}
	}
	return "(untitled)"
}

func extractDate(props map[string]property) time.Time {
	if dp, ok := props["Date"]; ok && dp.Date != nil {
		t, _ := time.Parse("2006-01-02", dp.Date.Start)
		return t
	}
	return time.Time{}
}

func extractRichText(props map[string]property, key string) string {
	prop, ok := props[key]
	if !ok || prop.Type != "rich_text" {
		return ""
	}
	var parts []string
	for _, rt := range prop.RichText {
		parts = append(parts, rt.PlainText)
	}
	return strings.Join(parts, "")
}

// API response types

type queryResponse struct {
	Results    []page `json:"results"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor"`
}

type page struct {
	ID          string              `json:"id"`
	CreatedTime string              `json:"created_time"`
	URL         string              `json:"url"`
	Properties  map[string]property `json:"properties"`
}

type property struct {
	Type     string     `json:"type"`
	Title    []richText `json:"title,omitempty"`
	RichText []richText `json:"rich_text,omitempty"`
	Date     *dateValue `json:"date,omitempty"`
}

type richText struct {
	PlainText string `json:"plain_text"`
}

type dateValue struct {
	Start string `json:"start"`
	End   string `json:"end,omitempty"`
}
