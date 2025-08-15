package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// LLMService provides LLM functionality for slug generation
type LLMService interface {
	GenerateSlug(title string) (string, error)
}

// anthropicService implements LLM service using Anthropic's SDK
type anthropicService struct {
	client *anthropic.Client
}

// NewLLMService creates a new LLM service instance
func NewLLMService(apiKey string) LLMService {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &anthropicService{
		client: &client,
	}
}

// GenerateSlug generates a short slug from the given title using Anthropic's API
func (s *anthropicService) GenerateSlug(title string) (string, error) {
	// Check if the title is longer than 3 words
	words := strings.Fields(title)
	if len(words) <= 3 {
		// If 3 words or less, use the existing sanitization logic
		return s.sanitizeSlug(title), nil
	}

	// Check if client is properly initialized
	if s.client == nil {
		return "", fmt.Errorf("anthropic client is nil - service not properly initialized")
	}

	prompt := fmt.Sprintf(`Given this title: "%s"

Generate a concise slug that is at most 3 words, uses only lowercase letters, numbers, and hyphens. The slug should capture the essence of the title while being brief and URL-friendly.

Return only the slug, nothing else.`, title)

	ctx := context.Background()
	message, err := s.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude_3_Haiku_20240307,
		MaxTokens: 50,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})

	if err != nil {
		return "", fmt.Errorf("failed to call Anthropic API: %w", err)
	}

	// Extract text from the response
	if len(message.Content) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	// Get the first content block and check if it's text
	contentBlock := message.Content[0]
	if contentBlock.Type != "text" {
		return "", fmt.Errorf("expected text content block, got: %s", contentBlock.Type)
	}

	textBlock := contentBlock.AsText()
	slug := strings.TrimSpace(textBlock.Text)
	slug = s.sanitizeSlug(slug)

	return slug, nil
}

// sanitizeSlug applies the same sanitization logic as the spec service
func (s *anthropicService) sanitizeSlug(title string) string {
	slug := strings.ToLower(title)
	// Replace non-alphanumeric characters with hyphens (same as spec service)
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
		} else {
			result.WriteRune('-')
		}
	}
	slug = result.String()

	// Remove consecutive hyphens and trim
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")

	if slug == "" {
		slug = "untitled"
	}
	return slug
}
