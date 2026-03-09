// Package llm provides LLM integration for generating standups, summaries, and suggestions.
package llm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/genai"
)

// Client wraps the Gemini generative AI client.
type Client struct {
	client *genai.Client
	model  string
	logDir string
	label  string
}

// New creates a Gemini client configured with the given API key and model.
// logDir is the directory to write conversation logs to; if empty, logging is disabled.
// label identifies the command context (e.g. "standup") and is used in log filenames.
func New(ctx context.Context, apiKey, model, logDir, label string) (*Client, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("creating Gemini client: %w", err)
	}

	return &Client{
		client: client,
		model:  model,
		logDir: logDir,
		label:  label,
	}, nil
}

// Generate sends a prompt to Gemini and returns the generated text.
func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	result, err := c.client.Models.GenerateContent(ctx, c.model, genai.Text(prompt), nil)
	if err != nil {
		return "", fmt.Errorf("generating content: %w", err)
	}

	text := result.Text()

	if c.logDir != "" {
		if logErr := c.writeLog(prompt, text); logErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write conversation log: %v\n", logErr)
		}
	}

	return text, nil
}

func (c *Client) writeLog(prompt, response string) error {
	if err := os.MkdirAll(c.logDir, 0o750); err != nil {
		return fmt.Errorf("creating log directory: %w", err)
	}

	now := time.Now()
	filename := fmt.Sprintf("%s-%s.md", c.label, now.Format("2006-01-02T15-04-05"))
	path := filepath.Join(c.logDir, filename)

	var b strings.Builder
	fmt.Fprintf(&b, "# %s - %s\n\n", c.label, now.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "**Model:** %s\n\n", c.model)
	b.WriteString("## Prompt\n\n")
	b.WriteString(prompt)
	b.WriteString("\n\n## Response\n\n")
	b.WriteString(response)
	b.WriteString("\n")

	return os.WriteFile(path, []byte(b.String()), 0o600)
}
