// Package slack fetches conversation data from the Slack API.
package slack

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

const (
	apiBaseURL = "https://slack.com/api"
	sourceName = "slack"
)

// Source implements source.DataSource for Slack.
type Source struct {
	userToken  string
	userID     string
	httpClient *http.Client
}

// New creates a Slack data source. userToken must be a user-scoped OAuth token
// with scopes like search:read, channels:history, im:history.
func New(userToken, userID string) *Source {
	return &Source{
		userToken:  userToken,
		userID:     userID,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the source identifier.
func (s *Source) Name() string { return sourceName }

// Fetch retrieves messages from Slack where the user participated within the given time range.
func (s *Source) Fetch(ctx context.Context, tr source.TimeRange) ([]source.DataItem, error) {
	query := fmt.Sprintf("from:<@%s> after:%s before:%s",
		s.userID,
		tr.Start.Format("2006-01-02"),
		tr.End.Format("2006-01-02"),
	)

	var allItems []source.DataItem
	page := 1

	for {
		params := url.Values{}
		params.Set("query", query)
		params.Set("sort", "timestamp")
		params.Set("count", "100")
		params.Set("page", fmt.Sprintf("%d", page))

		reqURL := apiBaseURL + "/search.messages?" + params.Encode()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+s.userToken)

		resp, err := s.httpClient.Do(req) //nolint:gosec // URL built from known constants
		if err != nil {
			return nil, fmt.Errorf("slack API request: %w", err)
		}
		defer resp.Body.Close() //nolint:errcheck // best-effort cleanup

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("slack API returned %d: %s", resp.StatusCode, string(body))
		}

		var sr searchResponse
		if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
			return nil, fmt.Errorf("decoding slack response: %w", err)
		}

		if !sr.OK {
			return nil, fmt.Errorf("slack API error: %s", sr.Error)
		}

		for _, msg := range sr.Messages.Matches {
			ts, _ := parseSlackTimestamp(msg.Timestamp)
			allItems = append(allItems, source.DataItem{
				ID:      msg.Timestamp,
				Source:  sourceName,
				Type:    "conversation",
				Title:   truncate(msg.Text, 100),
				Content: msg.Text,
				Metadata: map[string]string{
					"channel":    msg.Channel.Name,
					"channel_id": msg.Channel.ID,
					"permalink":  msg.Permalink,
					"username":   msg.Username,
				},
				Timestamp: ts,
			})
		}

		if page >= sr.Messages.Paging.Pages || len(sr.Messages.Matches) == 0 {
			break
		}
		page++
	}

	return allItems, nil
}

type searchResponse struct {
	OK       bool   `json:"ok"`
	Error    string `json:"error,omitempty"`
	Messages struct {
		Matches []searchMatch `json:"matches"`
		Paging  struct {
			Pages int `json:"pages"`
			Total int `json:"total"`
		} `json:"paging"`
	} `json:"messages"`
}

type searchMatch struct {
	Text      string `json:"text"`
	Timestamp string `json:"ts"`
	Username  string `json:"username"`
	Permalink string `json:"permalink"`
	Channel   struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"channel"`
}

func parseSlackTimestamp(ts string) (time.Time, error) {
	var sec, usec int64
	_, err := fmt.Sscanf(ts, "%d.%d", &sec, &usec)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(sec, usec*1000), nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
