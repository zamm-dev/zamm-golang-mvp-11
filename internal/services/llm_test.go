package services

import (
	"strings"
	"testing"
)

func TestLLMService_GenerateSlug_ShortTitle(t *testing.T) {
	// Test with short titles (3 words or less) - should use sanitization only
	llmService := NewLLMService("fake-api-key")

	testCases := []struct {
		title    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"Test", "test"},
		{"One Two Three", "one-two-three"},
		{"Special@Characters#Here", "special-characters-here"},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			result, err := llmService.GenerateSlug(tc.title)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestLLMService_SanitizeSlug(t *testing.T) {
	llmService := &anthropicService{}

	testCases := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"test@example.com", "test-example-com"},
		{"Multiple   Spaces", "multiple-spaces"},
		{"--leading-and-trailing--", "leading-and-trailing"},
		{"", "untitled"},
		{"123Numbers", "123numbers"},
		{"mixedCASE", "mixedcase"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := llmService.sanitizeSlug(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestLLMService_GenerateSlug_LongTitle(t *testing.T) {
	// Test with long titles (more than 3 words) - should attempt LLM call
	// Since we don't have a real API key, this will fail, but we can test the logic
	llmService := NewLLMService("")

	longTitle := "This is a very long title with more than three words"

	_, err := llmService.GenerateSlug(longTitle)

	// We expect an error since we don't have a real API key
	if err == nil {
		t.Error("Expected error for long title without API key, got nil")
	}

	// The error should be related to the API call since we don't have a valid API key
	// This is expected behavior for the test
}

func TestLLMService_WordCount(t *testing.T) {
	// Test that word counting works correctly
	testCases := []struct {
		title     string
		wordCount int
	}{
		{"Hello", 1},
		{"Hello World", 2},
		{"One Two Three", 3},
		{"This is more than three words", 6},
		{"", 0},
		{"   Spaces   Around   ", 2},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			wordCount := len(strings.Fields(tc.title))
			if wordCount != tc.wordCount {
				t.Errorf("Expected %d words, got %d for title %q", tc.wordCount, wordCount, tc.title)
			}
		})
	}
}
