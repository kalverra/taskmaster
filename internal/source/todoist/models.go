package todoist

// Task represents a Todoist task (active or completed).
type Task struct {
	ID          string   `json:"id"`
	Content     string   `json:"content"`
	Description string   `json:"description"`
	ProjectID   string   `json:"project_id"`
	SectionID   string   `json:"section_id"`
	Priority    int      `json:"priority"`
	Labels      []string `json:"labels"`
	AddedAt     string   `json:"added_at"`
	CompletedAt *string  `json:"completed_at"`
	UpdatedAt   string   `json:"updated_at"`
	Due         *Due     `json:"due"`
	Duration    *struct {
		Amount int    `json:"amount"`
		Unit   string `json:"unit"`
	} `json:"duration"`
}

// Due represents the due date information for a task.
type Due struct {
	Date      string `json:"date"`
	Datetime  string `json:"datetime"`
	String    string `json:"string"`
	Recurring bool   `json:"is_recurring"`
}

// TasksResponse is the response for GET /api/v1/tasks (cursor-paginated).
type TasksResponse struct {
	Results    []Task  `json:"results"`
	NextCursor *string `json:"next_cursor"`
}

// CompletedTasksResponse is the response for GET /api/v1/tasks/completed/by_completion_date.
type CompletedTasksResponse struct {
	Items      []Task  `json:"items"`
	NextCursor *string `json:"next_cursor"`
}

// Project represents a Todoist project.
type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ProjectsResponse is the response for GET /api/v1/projects (cursor-paginated).
type ProjectsResponse struct {
	Results    []Project `json:"results"`
	NextCursor *string   `json:"next_cursor"`
}

// Section represents a Todoist project section (e.g. "To Do", "In Progress").
type Section struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ProjectID string `json:"project_id"`
}

// SectionsResponse is the response for GET /api/v1/sections (cursor-paginated).
type SectionsResponse struct {
	Results    []Section `json:"results"`
	NextCursor *string   `json:"next_cursor"`
}

// Comment represents a Todoist task comment.
type Comment struct {
	ID       string `json:"id"`
	Content  string `json:"content"`
	PostedAt string `json:"posted_at"`
}

// CommentsResponse is the response for GET /api/v1/comments (cursor-paginated).
type CommentsResponse struct {
	Results    []Comment `json:"results"`
	NextCursor *string   `json:"next_cursor"`
}
