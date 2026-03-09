package jira

// SearchResponse represents the Jira JQL enhanced search response.
type SearchResponse struct {
	IsLast        bool    `json:"isLast"`
	NextPageToken string  `json:"nextPageToken,omitempty"`
	Issues        []Issue `json:"issues"`
}

// Issue represents a Jira issue returned from the search API.
type Issue struct {
	ID     string      `json:"id"`
	Key    string      `json:"key"`
	Self   string      `json:"self"`
	Fields IssueFields `json:"fields"`
}

// IssueFields contains the fields returned for a Jira issue.
type IssueFields struct {
	Summary   string    `json:"summary"`
	Updated   string    `json:"updated"`
	Status    Status    `json:"status"`
	Priority  Priority  `json:"priority"`
	IssueType IssueType `json:"issuetype"`
}

// Status represents a Jira issue status.
type Status struct {
	Name     string         `json:"name"`
	Category StatusCategory `json:"statusCategory"`
}

// StatusCategory groups statuses into high-level categories (e.g. "done").
type StatusCategory struct {
	Key string `json:"key"`
}

// Priority represents a Jira issue priority level.
type Priority struct {
	Name string `json:"name"`
}

// IssueType represents the type of a Jira issue (e.g. Bug, Story).
type IssueType struct {
	Name string `json:"name"`
}
