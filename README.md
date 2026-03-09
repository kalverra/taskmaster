# Taskmaster

Aggregate your activity from Todoist, Slack, Jira, and Notion, then use Gemini to generate daily standups, weekly summaries, and prioritization suggestions.

## Install

```sh
go install github.com/kalverra/taskmaster@latest
```

Or build from source:

```sh
git clone https://github.com/kalverra/taskmaster.git
cd taskmaster
go build -o taskmaster .
```

## Configuration

Create `~/.taskmaster.yaml` with the sources and API keys you want to use. Only configured sources are queried; all are optional except Gemini (required for generation commands).

```yaml
gemini:
  api_key: "your-gemini-api-key"
  model: "gemini-2.0-flash"  # optional, this is the default

todoist:
  api_token: "your-todoist-api-token"

slack:
  user_token: "xoxp-..."
  user_id: "U12345678"

jira:
  base_url: "https://yourorg.atlassian.net"
  email: "you@example.com"
  api_token: "your-jira-api-token"

notion:
  api_token: "secret_..."
  database_id: "your-journal-database-id"
```

Every config field can also be set via environment variables with the `TASKMASTER_` prefix and underscores for nesting:

```sh
export GEMINI_API_KEY="your-key"
export TODOIST_API_TOKEN="your-token"
```

## Usage

```sh
# Fetch and cache activity data (default: past day)
taskmaster gather
taskmaster gather --range week

# Generate a daily standup (auto-gathers if cache is stale)
taskmaster standup

# Generate a summary of accomplishments
taskmaster summary
taskmaster summary --range month

# Get suggestions on what to work on next
taskmaster suggest
```
