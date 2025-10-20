package config

import (
	"fmt"
	"regexp"
	"strings"

	"set/internal/ai"
)

// ValidateGitHubToken validates a GitHub personal access token format
func ValidateGitHubToken(token string) error {
	if token == "" {
		return fmt.Errorf("github token cannot be empty")
	}

	// GitHub classic tokens start with "ghp_"
	// GitHub fine-grained tokens start with "github_pat_"
	// OAuth tokens start with "gho_"
	validPrefixes := []string{"ghp_", "github_pat_", "gho_"}
	hasValidPrefix := false

	for _, prefix := range validPrefixes {
		if strings.HasPrefix(token, prefix) {
			hasValidPrefix = true
			break
		}
	}

	if !hasValidPrefix {
		return fmt.Errorf("github token must start with ghp_, github_pat_, or gho_")
	}

	// Token should be at least 40 characters
	if len(token) < 40 {
		return fmt.Errorf("github token is too short (minimum 40 characters)")
	}

	return nil
}

// ValidateGitHubRepo validates a GitHub repository format (owner/repo)
func ValidateGitHubRepo(repo string) error {
	if repo == "" {
		return fmt.Errorf("repository cannot be empty")
	}

	// Match pattern: owner/repo
	repoPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+/[a-zA-Z0-9_.-]+$`)
	if !repoPattern.MatchString(repo) {
		return fmt.Errorf("repository must be in format 'owner/repo', got: %s", repo)
	}

	return nil
}

// ValidateAPIKey validates a generic API key (OpenAI, Claude, etc.)
func ValidateAPIKey(provider, key string) error {
	if key == "" {
		return fmt.Errorf("%s API key cannot be empty", provider)
	}

	switch provider {
	case "openai":
		if !strings.HasPrefix(key, "sk-") {
			return fmt.Errorf("openai API key must start with 'sk-'")
		}
		if len(key) < 20 {
			return fmt.Errorf("openai API key is too short")
		}
	case "claude":
		if !strings.HasPrefix(key, "sk-ant-") {
			return fmt.Errorf("claude API key must start with 'sk-ant-'")
		}
		if len(key) < 20 {
			return fmt.Errorf("claude API key is too short")
		}
	default:
		// Generic validation for unknown providers
		if len(key) < 10 {
			return fmt.Errorf("%s API key is too short", provider)
		}
	}

	return nil
}

// ValidateOpenAIKey validates an OpenAI API key by making a test request
func ValidateOpenAIKey(apiKey string) error {
	// First do format validation
	if err := ValidateAPIKey("openai", apiKey); err != nil {
		return err
	}

	// Then test with actual API call
	client := ai.NewOpenAIClient(apiKey)
	if err := client.ValidateAPIKey(); err != nil {
		return fmt.Errorf("OpenAI API key validation failed: %w", err)
	}

	return nil
}

// AIValidationResult holds the result of AI API validation
type AIValidationResult struct {
	Valid    bool
	Provider string
	Error    string
	Model    string
}

// ValidateAIProvider validates an AI provider's API key
func ValidateAIProvider(provider, apiKey string) (*AIValidationResult, error) {
	result := &AIValidationResult{
		Valid:    false,
		Provider: provider,
	}

	// Format validation first
	if err := ValidateAPIKey(provider, apiKey); err != nil {
		result.Error = err.Error()
		return result, err
	}

	// Live validation for supported providers
	switch provider {
	case "openai":
		client := ai.NewOpenAIClient(apiKey)
		if err := client.ValidateAPIKey(); err != nil {
			result.Error = err.Error()
			return result, fmt.Errorf("OpenAI API key is invalid: %w", err)
		}
		result.Valid = true
		result.Model = "gpt-4"

	case "claude":
		// Claude validation would go here when implemented
		result.Error = "Claude validation not yet implemented"
		return result, fmt.Errorf("claude provider validation not yet implemented")

	default:
		result.Error = fmt.Sprintf("unknown provider: %s", provider)
		return result, fmt.Errorf("unknown AI provider: %s", provider)
	}

	return result, nil
}
