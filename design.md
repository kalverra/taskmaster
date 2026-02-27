# Taskmaster Design

Taskmaster is a CLI tool that aggregates personal activity data from multiple services, caches it locally, and uses an LLM to produce actionable outputs like standups, summaries, and prioritization suggestions.

## High-Level Architecture

```mermaid
flowchart TD
    User([User]) --> CLI

    subgraph CLI [CLI Layer - cmd/]
        gather
        standup
        summary
        suggest
    end

    subgraph Sources [Data Sources - internal/source/]
        Todoist
        Slack
        Jira
        Notion
    end

    subgraph Core [Core Services - internal/]
        Cache[(SQLite Cache)]
        Formatter[Prompt Formatter]
        LLM[Gemini Client]
    end

    gather -->|fetch| Sources
    Sources -->|DataItems| gather
    gather -->|store| Cache

    standup & summary & suggest -->|query| Cache
    Cache -->|DataItems| Formatter
    Formatter -->|prompt| LLM
    LLM -->|text| User
```

## Data Flow

```mermaid
sequenceDiagram
    participant U as User
    participant C as CLI Command
    participant S as Data Sources
    participant DB as SQLite Cache
    participant F as Formatter
    participant G as Gemini

    U->>C: taskmaster standup
    C->>DB: IsFresh(source, range)?
    alt Cache is stale
        C->>S: Fetch(timeRange)
        S-->>C: []DataItem
        C->>DB: Store(items)
    end
    C->>DB: Query(range)
    DB-->>C: []DataItem
    C->>F: FormatStandup(items)
    F-->>C: prompt string
    C->>G: Generate(prompt)
    G-->>C: response text
    C-->>U: formatted output
```

## Core Types

```mermaid
classDiagram
    class DataSource {
        <<interface>>
        +Name() string
        +Fetch(ctx, TimeRange) []DataItem
    }

    class TimeRange {
        +Start time.Time
        +End time.Time
        +Label string
    }

    class DataItem {
        +ID string
        +Source string
        +Type string
        +Title string
        +Content string
        +Metadata map
        +Timestamp time.Time
    }

    class Cache {
        +Store([]DataItem)
        +Query(source, TimeRange) []DataItem
        +IsFresh(source, TimeRange, maxAge) bool
        +Close()
    }

    DataSource ..> TimeRange : uses
    DataSource ..> DataItem : produces
    Cache ..> DataItem : stores/returns
```

## Configuration

Config is loaded via Viper from `~/.taskmaster.yaml` with `TASKMASTER_*` env var overrides.

```mermaid
flowchart LR
    File["~/.taskmaster.yaml"] --> Viper
    Env["TASKMASTER_* env vars"] --> Viper
    Viper --> Config

    Config --> TodoistConfig
    Config --> SlackConfig
    Config --> JiraConfig
    Config --> NotionConfig
    Config --> GeminiConfig
    Config --> CacheDir
```

Each source is only activated when its required config fields are non-empty. See `buildSources()` in `cmd/gather.go`.

## Cache Layer

SQLite database at `~/.taskmaster/cache.db` with a single `data_items` table:

| Column        | Type     | Notes                                 |
| ------------- | -------- | ------------------------------------- |
| id            | TEXT     | PK (with source)                      |
| source        | TEXT     | PK (with id)                          |
| type          | TEXT     | e.g. `task_completed`, `conversation` |
| title         | TEXT     |                                       |
| content       | TEXT     |                                       |
| metadata_json | TEXT     | JSON-encoded key-value pairs          |
| timestamp     | DATETIME | When the activity occurred            |
| fetched_at    | DATETIME | When this row was cached              |

Staleness is checked via `fetched_at` against a 1-hour window before re-fetching.

## Command Reference

| Command              | Default Range | Description                                    |
| -------------------- | ------------- | ---------------------------------------------- |
| `gather -r <range>`  | day           | Fetch from all configured sources and cache    |
| `standup`            | day           | Generate daily standup (auto-gathers if stale) |
| `summary -r <range>` | week          | Generate accomplishment summary                |
| `suggest`            | week          | Suggest priorities and time allocation         |

Ranges: `day`, `week`, `month`, `quarter`.
