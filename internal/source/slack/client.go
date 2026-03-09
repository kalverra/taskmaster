// Package slack fetches conversation data from the Slack API using the slack-go package.
// API docs: https://docs.slack.dev/reference/methods/search.messages/
package slack

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	slackapi "github.com/slack-go/slack"

	"github.com/kalverra/taskmaster/internal/source"
)

const sourceName = "slack"

// Client defines the interface for Slack API operations.
type Client interface {
	SearchMessages(
		ctx context.Context,
		query string,
		params slackapi.SearchParameters,
	) (*slackapi.SearchMessages, error)
}

// slackGoClient implements Client using the slack-go/slack package.
type slackGoClient struct {
	api *slackapi.Client
	log zerolog.Logger
}

// NewClient creates a Slack API client wrapping slack-go.
func NewClient(userToken string, log zerolog.Logger) Client {
	httpClient := source.NewLoggingClient(log)
	return &slackGoClient{
		api: slackapi.New(userToken, slackapi.OptionHTTPClient(&httpClient)),
		log: log,
	}
}

func (c *slackGoClient) SearchMessages(
	ctx context.Context,
	query string,
	params slackapi.SearchParameters,
) (*slackapi.SearchMessages, error) {
	msgs, err := c.api.SearchMessagesContext(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("slack search.messages: %w", err)
	}

	return msgs, nil
}

// Source implements source.DataSource for Slack.
type Source struct {
	client Client
	userID string
}

// New creates a Slack data source with the default slack-go client.
func New(userToken, userID string) *Source {
	log := zerolog.Nop()
	return &Source{
		client: NewClient(userToken, log),
		userID: userID,
	}
}

// NewWithClient creates a Slack data source with a provided client (useful for testing).
func NewWithClient(client Client, userID string) *Source {
	return &Source{client: client, userID: userID}
}

// Name returns the name of this data source.
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
		params := slackapi.SearchParameters{
			Sort:          "timestamp",
			SortDirection: "desc",
			Count:         100,
			Page:          page,
		}

		msgs, err := s.client.SearchMessages(ctx, query, params)
		if err != nil {
			return nil, err
		}

		for _, msg := range msgs.Matches {
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

		if page >= msgs.Pages || len(msgs.Matches) == 0 {
			break
		}
		page++
	}

	return allItems, nil
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
