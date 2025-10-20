package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	// Check defaults
	if cfg.AI.Provider != "openai" {
		t.Errorf("Expected AI provider 'openai', got '%s'", cfg.AI.Provider)
	}

	if cfg.AI.Model != "gpt-4" {
		t.Errorf("Expected AI model 'gpt-4', got '%s'", cfg.AI.Model)
	}

	if cfg.Estimation.ConfidenceThreshold != 75 {
		t.Errorf("Expected confidence threshold 75, got %d", cfg.Estimation.ConfidenceThreshold)
	}

	if cfg.Estimation.MaxSimilarTasks != 5 {
		t.Errorf("Expected max similar tasks 5, got %d", cfg.Estimation.MaxSimilarTasks)
	}

	if cfg.Output.Format != "table" {
		t.Errorf("Expected output format 'table', got '%s'", cfg.Output.Format)
	}

	if !cfg.Output.Colors {
		t.Error("Expected colors to be enabled by default")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError bool
	}{
		{
			name:      "Valid default config",
			config:    DefaultConfig(),
			wantError: false,
		},
		{
			name: "Invalid AI provider",
			config: &Config{
				GitHub:     GitHubConfig{},
				AI:         AIConfig{Provider: "invalid"},
				Estimation: EstimationConfig{ConfidenceThreshold: 75, MaxSimilarTasks: 5},
				Output:     OutputConfig{Format: "table", Colors: true},
			},
			wantError: true,
		},
		{
			name: "Invalid output format",
			config: &Config{
				GitHub:     GitHubConfig{},
				AI:         AIConfig{Provider: "openai"},
				Estimation: EstimationConfig{ConfidenceThreshold: 75, MaxSimilarTasks: 5},
				Output:     OutputConfig{Format: "invalid", Colors: true},
			},
			wantError: true,
		},
		{
			name: "Invalid confidence threshold (too low)",
			config: &Config{
				GitHub:     GitHubConfig{},
				AI:         AIConfig{Provider: "openai"},
				Estimation: EstimationConfig{ConfidenceThreshold: -1, MaxSimilarTasks: 5},
				Output:     OutputConfig{Format: "table", Colors: true},
			},
			wantError: true,
		},
		{
			name: "Invalid confidence threshold (too high)",
			config: &Config{
				GitHub:     GitHubConfig{},
				AI:         AIConfig{Provider: "openai"},
				Estimation: EstimationConfig{ConfidenceThreshold: 101, MaxSimilarTasks: 5},
				Output:     OutputConfig{Format: "table", Colors: true},
			},
			wantError: true,
		},
		{
			name: "Invalid max similar tasks (too low)",
			config: &Config{
				GitHub:     GitHubConfig{},
				AI:         AIConfig{Provider: "openai"},
				Estimation: EstimationConfig{ConfidenceThreshold: 75, MaxSimilarTasks: 0},
				Output:     OutputConfig{Format: "table", Colors: true},
			},
			wantError: true,
		},
		{
			name: "Invalid max similar tasks (too high)",
			config: &Config{
				GitHub:     GitHubConfig{},
				AI:         AIConfig{Provider: "openai"},
				Estimation: EstimationConfig{ConfidenceThreshold: 75, MaxSimilarTasks: 21},
				Output:     OutputConfig{Format: "table", Colors: true},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestIsConfigured(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   bool
	}{
		{
			name:   "Default config (not configured)",
			config: DefaultConfig(),
			want:   false,
		},
		{
			name: "Only GitHub token",
			config: &Config{
				GitHub: GitHubConfig{Token: "ghp_test"},
				AI:     AIConfig{},
			},
			want: false,
		},
		{
			name: "Only AI key",
			config: &Config{
				GitHub: GitHubConfig{},
				AI:     AIConfig{APIKey: "sk-test"},
			},
			want: false,
		},
		{
			name: "Fully configured",
			config: &Config{
				GitHub: GitHubConfig{Token: "ghp_test"},
				AI:     AIConfig{APIKey: "sk-test"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsConfigured(); got != tt.want {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "set-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test config
	testConfig := DefaultConfig()
	testConfig.GitHub.Token = "ghp_test123456789012345678901234567890"
	testConfig.GitHub.DefaultRepo = "facebook/react"
	testConfig.AI.Provider = "openai"
	testConfig.AI.APIKey = "sk-test123456789012345678901234567890"
	testConfig.AI.Model = "gpt-4"

	// Override GetConfigPath for testing
	originalGetConfigPath := GetConfigPath
	GetConfigPath = func() (string, error) {
		return filepath.Join(tempDir, ".set.yaml"), nil
	}
	defer func() { GetConfigPath = originalGetConfigPath }()

	// Save config
	if err := Save(testConfig); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load config
	loadedConfig, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify loaded config matches saved config
	if loadedConfig.GitHub.Token != testConfig.GitHub.Token {
		t.Errorf("GitHub token mismatch: got %s, want %s", loadedConfig.GitHub.Token, testConfig.GitHub.Token)
	}

	if loadedConfig.GitHub.DefaultRepo != testConfig.GitHub.DefaultRepo {
		t.Errorf("GitHub repo mismatch: got %s, want %s", loadedConfig.GitHub.DefaultRepo, testConfig.GitHub.DefaultRepo)
	}

	if loadedConfig.AI.Provider != testConfig.AI.Provider {
		t.Errorf("AI provider mismatch: got %s, want %s", loadedConfig.AI.Provider, testConfig.AI.Provider)
	}

	if loadedConfig.AI.APIKey != testConfig.AI.APIKey {
		t.Errorf("AI key mismatch: got %s, want %s", loadedConfig.AI.APIKey, testConfig.AI.APIKey)
	}
}

func TestLoadNonExistentConfig(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "set-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Override GetConfigPath for testing
	originalGetConfigPath := GetConfigPath
	GetConfigPath = func() (string, error) {
		return filepath.Join(tempDir, ".set.yaml"), nil
	}
	defer func() { GetConfigPath = originalGetConfigPath }()

	// Load should create default config when file doesn't exist
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load should not fail when config doesn't exist: %v", err)
	}

	// Should be default config
	if cfg.AI.Provider != "openai" {
		t.Errorf("Expected default provider 'openai', got '%s'", cfg.AI.Provider)
	}
}
