package todoist

import "context"

// FakeClient implements Client for testing.
type FakeClient struct {
	GetProjectsFn       func(ctx context.Context, cursor string, limit int) (*ProjectsResponse, error)
	GetActiveTasksFn    func(ctx context.Context, cursor string, limit int) (*TasksResponse, error)
	GetCompletedTasksFn func(ctx context.Context, since, until, cursor string, limit int) (*CompletedTasksResponse, error)
	GetCommentsFn       func(ctx context.Context, taskID, cursor string, limit int) (*CommentsResponse, error)
}

// GetProjects delegates to GetProjectsFn if set, otherwise returns an empty response.
func (f *FakeClient) GetProjects(ctx context.Context, cursor string, limit int) (*ProjectsResponse, error) {
	if f.GetProjectsFn != nil {
		return f.GetProjectsFn(ctx, cursor, limit)
	}
	return &ProjectsResponse{}, nil
}

// GetActiveTasks delegates to GetActiveTasksFn if set, otherwise returns an empty response.
func (f *FakeClient) GetActiveTasks(ctx context.Context, cursor string, limit int) (*TasksResponse, error) {
	if f.GetActiveTasksFn != nil {
		return f.GetActiveTasksFn(ctx, cursor, limit)
	}
	return &TasksResponse{}, nil
}

// GetCompletedTasks delegates to GetCompletedTasksFn if set, otherwise returns an empty response.
func (f *FakeClient) GetCompletedTasks(
	ctx context.Context,
	since, until, cursor string,
	limit int,
) (*CompletedTasksResponse, error) {
	if f.GetCompletedTasksFn != nil {
		return f.GetCompletedTasksFn(ctx, since, until, cursor, limit)
	}
	return &CompletedTasksResponse{}, nil
}

// GetComments delegates to GetCommentsFn if set, otherwise returns an empty response.
func (f *FakeClient) GetComments(ctx context.Context, taskID, cursor string, limit int) (*CommentsResponse, error) {
	if f.GetCommentsFn != nil {
		return f.GetCommentsFn(ctx, taskID, cursor, limit)
	}
	return &CommentsResponse{}, nil
}

// Close is a no-op for the fake client.
func (f *FakeClient) Close() {}
