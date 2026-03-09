package git

import "time"

// FakeClient implements Client for testing.
type FakeClient struct {
	LogFn func(repoPath string, since, until time.Time, author string) ([]Commit, error)
}

// Log delegates to LogFn if set, otherwise returns an empty slice.
func (f *FakeClient) Log(repoPath string, since, until time.Time, author string) ([]Commit, error) {
	if f.LogFn != nil {
		return f.LogFn(repoPath, since, until, author)
	}
	return nil, nil
}
