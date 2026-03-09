package git

import "time"

// Commit represents a single parsed git commit.
type Commit struct {
	Hash    string
	Subject string
	Body    string
	Author  string
	Date    time.Time
	Repo    string
}
