package jira

import "context"

// FakeClient implements Client for testing.
type FakeClient struct {
	SearchJQLFn func(ctx context.Context, jql string, fields []string, maxResults int, nextPageToken string) (*SearchResponse, error)
}

// SearchJQL delegates to SearchJQLFn if set, otherwise returns an empty response.
func (f *FakeClient) SearchJQL(
	ctx context.Context,
	jql string,
	fields []string,
	maxResults int,
	nextPageToken string,
) (*SearchResponse, error) {
	if f.SearchJQLFn != nil {
		return f.SearchJQLFn(ctx, jql, fields, maxResults, nextPageToken)
	}
	return &SearchResponse{IsLast: true}, nil
}

// Close is a no-op for the fake client.
func (f *FakeClient) Close() {}
