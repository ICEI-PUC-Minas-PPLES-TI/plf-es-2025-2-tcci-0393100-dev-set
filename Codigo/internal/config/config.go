package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds all configuration for the SET CLI
type Config struct {
	GitHub     GitHubConfig     `mapstructure:"github"`
	AI         AIConfig         `mapstructure:"ai"`
	Estimation EstimationConfig `mapstructure:"estimation"`
	Output     OutputConfig     `mapstructure:"output"`
	Developer  DeveloperConfig  `mapstructure:"developer"`
}

// GitHubConfig holds GitHub-related configuration
type GitHubConfig struct {
	Token       string `mapstructure:"token"`
	DefaultRepo string `mapstructure:"default_repo"`
}

// AIConfig holds AI provider configuration
type AIConfig struct {
	Provider string `mapstructure:"provider"` // "openai" or "claude"
	APIKey   string `mapstructure:"api_key"`
	Model    string `mapstructure:"model"`
}

// EstimationConfig holds estimation-related settings
type EstimationConfig struct {
	ConfidenceThreshold int `mapstructure:"confidence_threshold"`
	MaxSimilarTasks     int `mapstructure:"max_similar_tasks"`
}

// OutputConfig holds output formatting preferences
type OutputConfig struct {
	Format string `mapstructure:"format"` // "table", "json", "csv"
	Colors bool   `mapstructure:"colors"`
}

// DeveloperConfig holds developer mode settings
type DeveloperConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	FakeDataDir string `mapstructure:"fake_data_dir"`
	DebugMode   bool   `mapstructure:"debug_mode"`
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		GitHub: GitHubConfig{
			Token:       "",
			DefaultRepo: "",
		},
		AI: AIConfig{
			Provider: "openai",
			APIKey:   "",
			Model:    "gpt-4",
		},
		Estimation: EstimationConfig{
			ConfidenceThreshold: 75,
			MaxSimilarTasks:     5,
		},
		Output: OutputConfig{
			Format: "table",
			Colors: true,
		},
		Developer: DeveloperConfig{
			Enabled:     false,
			FakeDataDir: "~/.set/dev-data",
			DebugMode:   false,
		},
	}
}

// Load reads configuration from the default config file or creates it if it doesn't exist
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get config path: %w", err)
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Config doesn't exist, create default
		cfg := DefaultConfig()
		if err := Save(cfg); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return cfg, nil
	}

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// Save writes the configuration to the config file
func Save(cfg *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Set values in viper
	viper.Set("github.token", cfg.GitHub.Token)
	viper.Set("github.default_repo", cfg.GitHub.DefaultRepo)
	viper.Set("ai.provider", cfg.AI.Provider)
	viper.Set("ai.api_key", cfg.AI.APIKey)
	viper.Set("ai.model", cfg.AI.Model)
	viper.Set("estimation.confidence_threshold", cfg.Estimation.ConfidenceThreshold)
	viper.Set("estimation.max_similar_tasks", cfg.Estimation.MaxSimilarTasks)
	viper.Set("output.format", cfg.Output.Format)
	viper.Set("output.colors", cfg.Output.Colors)
	viper.Set("developer.enabled", cfg.Developer.Enabled)
	viper.Set("developer.fake_data_dir", cfg.Developer.FakeDataDir)
	viper.Set("developer.debug_mode", cfg.Developer.DebugMode)

	if err := viper.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfigPath is a variable that returns the path to the configuration file
// It can be overridden for testing purposes
var GetConfigPath = func() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	return filepath.Join(home, ".set.yaml"), nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate GitHub token format (basic check)
	if c.GitHub.Token != "" && len(c.GitHub.Token) < 10 {
		return fmt.Errorf("github token appears to be invalid (too short)")
	}

	// Validate AI provider
	if c.AI.Provider != "" && c.AI.Provider != "openai" && c.AI.Provider != "claude" {
		return fmt.Errorf("ai provider must be 'openai' or 'claude', got: %s", c.AI.Provider)
	}

	// Validate output format
	if c.Output.Format != "table" && c.Output.Format != "json" && c.Output.Format != "csv" {
		return fmt.Errorf("output format must be 'table', 'json', or 'csv', got: %s", c.Output.Format)
	}

	// Validate confidence threshold
	if c.Estimation.ConfidenceThreshold < 0 || c.Estimation.ConfidenceThreshold > 100 {
		return fmt.Errorf("confidence threshold must be between 0 and 100, got: %d", c.Estimation.ConfidenceThreshold)
	}

	// Validate max similar tasks
	if c.Estimation.MaxSimilarTasks < 1 || c.Estimation.MaxSimilarTasks > 20 {
		return fmt.Errorf("max similar tasks must be between 1 and 20, got: %d", c.Estimation.MaxSimilarTasks)
	}

	return nil
}

// IsConfigured returns true if the essential configuration is set
func (c *Config) IsConfigured() bool {
	return c.GitHub.Token != "" && c.AI.APIKey != ""
}
