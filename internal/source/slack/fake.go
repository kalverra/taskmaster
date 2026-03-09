package slack

import (
	"context"

	slackapi "github.com/slack-go/slack"
)

// FakeClient implements Client for testing.
type FakeClient struct {
	SearchMessagesFn func(ctx context.Context, query string, params slackapi.SearchParameters) (*slackapi.SearchMessages, error)
}

// SearchMessages delegates to SearchMessagesFn if set, otherwise returns an empty response.
func (f *FakeClient) SearchMessages(
	ctx context.Context,
	query string,
	params slackapi.SearchParameters,
) (*slackapi.SearchMessages, error) {
	if f.SearchMessagesFn != nil {
		return f.SearchMessagesFn(ctx, query, params)
	}
	return &slackapi.SearchMessages{}, nil
}
