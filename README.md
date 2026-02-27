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
export TASKMASTER_GEMINI_API_KEY="your-key"
export TASKMASTER_TODOIST_API_TOKEN="your-token"
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

### Commands

| Command   | Description                                         | Default Range |
| --------- | --------------------------------------------------- | ------------- |
| `gather`  | Fetch from all configured sources and cache locally | `day`         |
| `standup` | Generate a daily standup update                     | `day`         |
| `summary` | Generate a high-level accomplishment summary        | `week`        |
| `suggest` | Suggest priorities and time allocation              | `week`        |

The `--range` / `-r` flag accepts `day`, `week`, `month`, or `quarter`.

## Data Storage

Cached data is stored in a SQLite database at `~/.taskmaster/cache.db`. The cache directory can be changed via the `cache_dir` config field or `TASKMASTER_CACHE_DIR` env var.
