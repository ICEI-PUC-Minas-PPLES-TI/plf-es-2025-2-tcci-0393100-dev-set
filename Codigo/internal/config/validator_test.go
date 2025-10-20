package config

import (
	"testing"
)

func TestValidateGitHubToken(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		wantError bool
	}{
		{
			name:      "Empty token",
			token:     "",
			wantError: true,
		},
		{
			name:      "Valid ghp_ token",
			token:     "ghp_1234567890123456789012345678901234567890",
			wantError: false,
		},
		{
			name:      "Valid github_pat_ token",
			token:     "github_pat_1234567890123456789012345678901234567890",
			wantError: false,
		},
		{
			name:      "Valid gho_ token",
			token:     "gho_1234567890123456789012345678901234567890",
			wantError: false,
		},
		{
			name:      "Invalid prefix",
			token:     "invalid_1234567890123456789012345678901234567890",
			wantError: true,
		},
		{
			name:      "Too short",
			token:     "ghp_short",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGitHubToken(tt.token)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateGitHubToken() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateGitHubRepo(t *testing.T) {
	tests := []struct {
		name      string
		repo      string
		wantError bool
	}{
		{
			name:      "Empty repo",
			repo:      "",
			wantError: true,
		},
		{
			name:      "Valid repo",
			repo:      "facebook/react",
			wantError: false,
		},
		{
			name:      "Valid repo with numbers",
			repo:      "user123/repo-456",
			wantError: false,
		},
		{
			name:      "Valid repo with underscore",
			repo:      "user_name/repo_name",
			wantError: false,
		},
		{
			name:      "Invalid - no slash",
			repo:      "facebookreact",
			wantError: true,
		},
		{
			name:      "Invalid - multiple slashes",
			repo:      "facebook/react/main",
			wantError: true,
		},
		{
			name:      "Invalid - starts with slash",
			repo:      "/facebook/react",
			wantError: true,
		},
		{
			name:      "Invalid - ends with slash",
			repo:      "facebook/react/",
			wantError: true,
		},
		{
			name:      "Invalid - special characters",
			repo:      "facebook@/react!",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGitHubRepo(tt.repo)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateGitHubRepo() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateAPIKey(t *testing.T) {
	tests := []struct {
		name      string
		provider  string
		key       string
		wantError bool
	}{
		{
			name:      "Empty OpenAI key",
			provider:  "openai",
			key:       "",
			wantError: true,
		},
		{
			name:      "Valid OpenAI key",
			provider:  "openai",
			key:       "sk-1234567890123456789012345678901234567890",
			wantError: false,
		},
		{
			name:      "Invalid OpenAI key prefix",
			provider:  "openai",
			key:       "invalid-1234567890123456789012345678901234567890",
			wantError: true,
		},
		{
			name:      "Too short OpenAI key",
			provider:  "openai",
			key:       "sk-short",
			wantError: true,
		},
		{
			name:      "Empty Claude key",
			provider:  "claude",
			key:       "",
			wantError: true,
		},
		{
			name:      "Valid Claude key",
			provider:  "claude",
			key:       "sk-ant-1234567890123456789012345678901234567890",
			wantError: false,
		},
		{
			name:      "Invalid Claude key prefix",
			provider:  "claude",
			key:       "sk-1234567890123456789012345678901234567890",
			wantError: true,
		},
		{
			name:      "Too short Claude key",
			provider:  "claude",
			key:       "sk-ant-short",
			wantError: true,
		},
		{
			name:      "Unknown provider - valid length",
			provider:  "unknown",
			key:       "12345678901234567890",
			wantError: false,
		},
		{
			name:      "Unknown provider - too short",
			provider:  "unknown",
			key:       "short",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAPIKey(tt.provider, tt.key)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateAPIKey() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateOpenAIKey(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		wantError bool
	}{
		{
			name:      "empty key",
			apiKey:    "",
			wantError: true,
		},
		{
			name:      "too short key",
			apiKey:    "short",
			wantError: true,
		},
		{
			name:      "invalid prefix",
			apiKey:    "invalid-key-1234567890123456789012345678901234567890",
			wantError: true,
		},
		{
			name:      "valid format but likely fake",
			apiKey:    "sk-1234567890123456789012345678901234567890123456789012",
			wantError: true, // Will fail on actual API validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOpenAIKey(tt.apiKey)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateOpenAIKey() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateAIProvider(t *testing.T) {
	tests := []struct {
		name      string
		provider  string
		apiKey    string
		wantError bool
		wantValid bool
	}{
		{
			name:      "empty provider",
			provider:  "",
			apiKey:    "test-key",
			wantError: true,
			wantValid: false,
		},
		{
			name:      "empty key",
			provider:  "openai",
			apiKey:    "",
			wantError: true,
			wantValid: false,
		},
		{
			name:      "invalid key format",
			provider:  "openai",
			apiKey:    "short",
			wantError: true,
			wantValid: false,
		},
		{
			name:      "valid format but fake key",
			provider:  "openai",
			apiKey:    "sk-1234567890123456789012345678901234567890123456789012",
			wantError: true, // Will fail on actual API validation
			wantValid: false,
		},
		{
			name:      "unknown provider",
			provider:  "unknown-provider",
			apiKey:    "test-key-1234567890",
			wantError: true,  // Unknown providers return error
			wantValid: false, // Not valid
		},
		{
			name:      "claude provider",
			provider:  "claude",
			apiKey:    "test-key-1234567890",
			wantError: true,  // Not yet implemented
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateAIProvider(tt.provider, tt.apiKey)

			if (err != nil) != tt.wantError {
				t.Errorf("ValidateAIProvider() error = %v, wantError %v", err, tt.wantError)
			}

			if result == nil {
				t.Fatal("ValidateAIProvider() returned nil result")
			}

			if result.Provider != tt.provider {
				t.Errorf("ValidateAIProvider() provider = %v, want %v", result.Provider, tt.provider)
			}

			if result.Valid != tt.wantValid {
				t.Errorf("ValidateAIProvider() valid = %v, want %v", result.Valid, tt.wantValid)
			}
		})
	}
}
