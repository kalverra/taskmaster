// Package git fetches commit history from local git repositories using go-git.
package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/kalverra/taskmaster/internal/source"
)

const sourceName = "git"

// Client defines the interface for git log operations.
type Client interface {
	Log(repoPath string, since, until time.Time, author string) ([]Commit, error)
}

// goGitClient implements Client using the go-git library.
type goGitClient struct{}

// NewClient creates a git client backed by go-git.
func NewClient() Client {
	return &goGitClient{}
}

// Log opens the repo at repoPath and returns commits between since and until.
// Merge commits (>1 parent) are skipped. If author is non-empty, only commits
// whose author name or email contains the string (case-insensitive) are returned.
func (c *goGitClient) Log(repoPath string, since, until time.Time, author string) ([]Commit, error) {
	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("opening repo %s: %w", repoPath, err)
	}

	ref, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("getting HEAD for %s: %w", repoPath, err)
	}

	iter, err := repo.Log(&gogit.LogOptions{
		From:  ref.Hash(),
		Since: &since,
		Until: &until,
	})
	if err != nil {
		return nil, fmt.Errorf("git log for %s: %w", repoPath, err)
	}

	repoName := filepath.Base(repoPath)
	authorLower := strings.ToLower(author)

	var commits []Commit
	err = iter.ForEach(func(commit *object.Commit) error {
		if commit.NumParents() > 1 {
			return nil
		}

		if authorLower != "" {
			nameMatch := strings.Contains(strings.ToLower(commit.Author.Name), authorLower)
			emailMatch := strings.Contains(strings.ToLower(commit.Author.Email), authorLower)
			if !nameMatch && !emailMatch {
				return nil
			}
		}

		subject, body := splitMessage(commit.Message)

		commits = append(commits, Commit{
			Hash:    commit.Hash.String(),
			Subject: subject,
			Body:    body,
			Author:  commit.Author.Name,
			Date:    commit.Author.When,
			Repo:    repoName,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterating commits for %s: %w", repoPath, err)
	}

	return commits, nil
}

func splitMessage(msg string) (subject, body string) {
	msg = strings.TrimSpace(msg)
	if before, after, ok := strings.Cut(msg, "\n"); ok {
		return before, strings.TrimSpace(after)
	}
	return msg, ""
}

// Source implements source.DataSource for local git repositories.
type Source struct {
	client   Client
	scanDirs []string
	author   string
}

// New creates a git data source that scans the given directories for repos.
func New(scanDirs []string, author string) *Source {
	return &Source{
		client:   NewClient(),
		scanDirs: scanDirs,
		author:   author,
	}
}

// NewWithClient creates a git data source with a provided client (useful for testing).
func NewWithClient(client Client, scanDirs []string, author string) *Source {
	return &Source{
		client:   client,
		scanDirs: scanDirs,
		author:   author,
	}
}

// Name returns the name of this data source.
func (s *Source) Name() string { return sourceName }

// Fetch discovers repos under the configured scan directories and retrieves
// commits within the given time range.
func (s *Source) Fetch(_ context.Context, tr source.TimeRange) ([]source.DataItem, error) {
	repos, err := discoverRepos(s.scanDirs)
	if err != nil {
		return nil, fmt.Errorf("discovering repos: %w", err)
	}

	var allItems []source.DataItem
	for _, repoPath := range repos {
		commits, err := s.client.Log(repoPath, tr.Start, tr.End, s.author)
		if err != nil {
			return nil, fmt.Errorf("fetching commits from %s: %w", filepath.Base(repoPath), err)
		}

		for _, c := range commits {
			allItems = append(allItems, source.DataItem{
				ID:      c.Hash,
				Source:  sourceName,
				Type:    "commit",
				Title:   c.Subject,
				Content: c.Body,
				Metadata: map[string]string{
					"hash":   c.Hash[:12],
					"author": c.Author,
					"repo":   c.Repo,
				},
				Timestamp: c.Date,
			})
		}
	}

	return allItems, nil
}

// discoverRepos walks each scan directory's immediate children and returns
// paths to directories that contain a .git folder. Tilde (~) in paths is
// expanded to the user's home directory.
func discoverRepos(scanDirs []string) ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}

	var repos []string
	for _, dir := range scanDirs {
		if strings.HasPrefix(dir, "~/") {
			dir = filepath.Join(home, dir[2:])
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("reading directory %s: %w", dir, err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			candidate := filepath.Join(dir, entry.Name())
			gitDir := filepath.Join(candidate, ".git")
			if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
				repos = append(repos, candidate)
			}
		}
	}

	return repos, nil
}
