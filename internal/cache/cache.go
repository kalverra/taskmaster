// Package cache provides a SQLite-backed local cache for activity data.
package cache

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kalverra/taskmaster/internal/source"

	_ "modernc.org/sqlite" // SQLite driver registration
)

// Cache wraps a SQLite database for storing and querying activity data.
type Cache struct {
	db *sql.DB
}

// New opens (or creates) a SQLite cache in the given directory.
func New(cacheDir string) (*Cache, error) {
	if err := os.MkdirAll(cacheDir, 0o750); err != nil {
		return nil, fmt.Errorf("creating cache directory: %w", err)
	}

	dbPath := filepath.Join(cacheDir, "cache.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening cache database: %w", err)
	}

	if err := migrate(db); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf("migrating cache database: %w (also failed to close: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("migrating cache database: %w", err)
	}

	return &Cache{db: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS data_items (
			id            TEXT NOT NULL,
			source        TEXT NOT NULL,
			type          TEXT NOT NULL,
			title         TEXT NOT NULL,
			content       TEXT NOT NULL,
			metadata_json TEXT NOT NULL DEFAULT '{}',
			timestamp     DATETIME NOT NULL,
			fetched_at    DATETIME NOT NULL DEFAULT (datetime('now')),
			PRIMARY KEY (id, source)
		);
		CREATE INDEX IF NOT EXISTS idx_data_items_source_ts ON data_items(source, timestamp);
	`)
	return err
}

// Store upserts a batch of data items into the cache.
func (c *Cache) Store(items []source.DataItem) error {
	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is a no-op

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO data_items (id, source, type, title, content, metadata_json, timestamp, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))
	`)
	if err != nil {
		return fmt.Errorf("preparing statement: %w", err)
	}
	defer stmt.Close() //nolint:errcheck // best-effort cleanup

	for _, item := range items {
		metaJSON, err := json.Marshal(item.Metadata)
		if err != nil {
			return fmt.Errorf("marshaling metadata for %s/%s: %w", item.Source, item.ID, err)
		}

		_, err = stmt.Exec(
			item.ID, item.Source, item.Type, item.Title,
			item.Content, string(metaJSON), item.Timestamp.UTC(),
		)
		if err != nil {
			return fmt.Errorf("inserting item %s/%s: %w", item.Source, item.ID, err)
		}
	}

	return tx.Commit()
}

// Query returns cached data items for a given source within a time range.
// If sourceName is empty, items from all sources are returned.
func (c *Cache) Query(sourceName string, tr source.TimeRange) ([]source.DataItem, error) {
	var (
		rows *sql.Rows
		err  error
	)

	if sourceName == "" {
		rows, err = c.db.Query(
			`SELECT id, source, type, title, content, metadata_json, timestamp
			 FROM data_items WHERE timestamp >= ? AND timestamp <= ? ORDER BY timestamp`,
			tr.Start.UTC(), tr.End.UTC(),
		)
	} else {
		rows, err = c.db.Query(
			`SELECT id, source, type, title, content, metadata_json, timestamp
			 FROM data_items WHERE source = ? AND timestamp >= ? AND timestamp <= ? ORDER BY timestamp`,
			sourceName, tr.Start.UTC(), tr.End.UTC(),
		)
	}
	if err != nil {
		return nil, fmt.Errorf("querying cache: %w", err)
	}
	defer rows.Close() //nolint:errcheck // best-effort cleanup

	var items []source.DataItem
	for rows.Next() {
		var item source.DataItem
		var metaJSON string
		if err := rows.Scan(
			&item.ID, &item.Source, &item.Type, &item.Title,
			&item.Content, &metaJSON, &item.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		if err := json.Unmarshal([]byte(metaJSON), &item.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshaling metadata: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

// IsFresh checks whether the cache has data fetched within the given staleness window for a source.
func (c *Cache) IsFresh(sourceName string, tr source.TimeRange, maxAge time.Duration) (bool, error) {
	var count int
	err := c.db.QueryRow(
		`SELECT COUNT(*) FROM data_items
		 WHERE source = ? AND timestamp >= ? AND timestamp <= ? AND fetched_at >= ?`,
		sourceName, tr.Start.UTC(), tr.End.UTC(), time.Now().Add(-maxAge).UTC(),
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking freshness: %w", err)
	}
	return count > 0, nil
}

// Close closes the underlying database connection.
func (c *Cache) Close() error {
	return c.db.Close()
}
