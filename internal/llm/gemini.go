// Package llm provides LLM integration for generating standups, summaries, and suggestions.
package llm

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

// Client wraps the Gemini generative AI client.
type Client struct {
	client *genai.Client
	model  string
}

// New creates a Gemini client configured with the given API key and model.
func New(ctx context.Context, apiKey, model string) (*Client, error) {
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
	}, nil
}

// Generate sends a prompt to Gemini and returns the generated text.
func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	result, err := c.client.Models.GenerateContent(ctx, c.model, genai.Text(prompt), nil)
	if err != nil {
		return "", fmt.Errorf("generating content: %w", err)
	}

	return result.Text(), nil
}
