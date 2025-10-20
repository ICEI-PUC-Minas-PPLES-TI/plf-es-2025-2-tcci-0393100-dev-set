/*
Copyright © 2025 Inácio Moraes da Silva
*/
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"set/internal/config"
	"set/internal/github"

	"github.com/spf13/cobra"
)

var (
	initialSetup   bool
	githubToken    string
	defaultRepo    string
	aiProvider     string
	aiKey          string
	listConfig     bool
	validateConfig bool
)

// configureCmd represents the configure command
var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure SET CLI settings",
	Long: `Configure SET CLI settings including GitHub token, AI provider, and preferences.

Examples:
  # Interactive configuration wizard
  set configure --initial

  # Set GitHub token
  set configure --github-token "ghp_xxxxxxxxxxxx"

  # Set default repository
  set configure --default-repo "facebook/react"

  # Configure AI provider
  set configure --ai-provider openai --ai-key "sk-xxxxx"

  # List current configuration
  set configure --list

  # Validate GitHub token and repository access
  set configure --validate`,
	RunE: runConfigure,
}

func init() {
	rootCmd.AddCommand(configureCmd)

	configureCmd.Flags().BoolVar(&initialSetup, "initial", false, "Run interactive configuration wizard")
	configureCmd.Flags().StringVar(&githubToken, "github-token", "", "GitHub personal access token")
	configureCmd.Flags().StringVar(&defaultRepo, "default-repo", "", "Default GitHub repository (owner/repo)")
	configureCmd.Flags().StringVar(&aiProvider, "ai-provider", "", "AI provider (openai or claude)")
	configureCmd.Flags().StringVar(&aiKey, "ai-key", "", "AI provider API key")
	configureCmd.Flags().BoolVar(&listConfig, "list", false, "List current configuration")
	configureCmd.Flags().BoolVar(&validateConfig, "validate", false, "Validate GitHub token and repository access")
}

func runConfigure(cmd *cobra.Command, args []string) error {
	// Handle --validate flag
	if validateConfig {
		return validateConfiguration()
	}

	// Handle --list flag
	if listConfig {
		return listConfiguration()
	}

	// Handle --initial flag
	if initialSetup {
		return runInteractiveSetup()
	}

	// Handle manual configuration via flags
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	modified := false

	// Update GitHub token
	if githubToken != "" {
		if err := config.ValidateGitHubToken(githubToken); err != nil {
			return fmt.Errorf("invalid GitHub token: %w", err)
		}
		cfg.GitHub.Token = githubToken
		modified = true
		fmt.Println("✓ GitHub token updated")
	}

	// Update default repo
	if defaultRepo != "" {
		if err := config.ValidateGitHubRepo(defaultRepo); err != nil {
			return fmt.Errorf("invalid repository format: %w", err)
		}
		cfg.GitHub.DefaultRepo = defaultRepo
		modified = true
		fmt.Printf("✓ Default repository set to: %s\n", defaultRepo)
	}

	// Update AI provider and key
	if aiProvider != "" {
		if aiProvider != "openai" && aiProvider != "claude" {
			return fmt.Errorf("invalid AI provider: must be 'openai' or 'claude'")
		}
		cfg.AI.Provider = aiProvider
		modified = true
		fmt.Printf("✓ AI provider set to: %s\n", aiProvider)
	}

	if aiKey != "" {
		provider := cfg.AI.Provider
		if provider == "" {
			provider = "openai" // default
		}
		if err := config.ValidateAPIKey(provider, aiKey); err != nil {
			return fmt.Errorf("invalid API key: %w", err)
		}
		cfg.AI.APIKey = aiKey
		modified = true
		fmt.Println("✓ AI API key updated")
	}

	// Save if anything was modified
	if modified {
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		configPath, _ := config.GetConfigPath()
		fmt.Printf("\n✓ Configuration saved to: %s\n", configPath)
	} else {
		fmt.Println("No configuration changes specified.")
		fmt.Println("Use --help to see available options or --initial for interactive setup.")
	}

	return nil
}

func runInteractiveSetup() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║       Welcome to SET CLI - Initial Setup Wizard           ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	cfg := config.DefaultConfig()

	// GitHub Configuration
	fmt.Println("📦 GitHub Configuration")
	fmt.Println("────────────────────────────────────────────────────────────")

	// GitHub Token
	fmt.Print("Enter your GitHub personal access token: ")
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)

	if token != "" {
		if err := config.ValidateGitHubToken(token); err != nil {
			fmt.Printf("Warning: %v\n", err)
			fmt.Print("Continue anyway? (y/n): ")
			confirm, _ := reader.ReadString('\n')
			if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
				return fmt.Errorf("setup cancelled")
			}
		}
		cfg.GitHub.Token = token
	}

	// Default Repository
	fmt.Print("Enter default repository (e.g., facebook/react) [optional]: ")
	repo, _ := reader.ReadString('\n')
	repo = strings.TrimSpace(repo)

	if repo != "" {
		if err := config.ValidateGitHubRepo(repo); err != nil {
			fmt.Printf("Warning: %v\n", err)
		} else {
			cfg.GitHub.DefaultRepo = repo
		}
	}

	fmt.Println()

	// AI Configuration
	fmt.Println("🤖 AI Provider Configuration")
	fmt.Println("────────────────────────────────────────────────────────────")

	fmt.Print("Choose AI provider (openai/claude) [openai]: ")
	provider, _ := reader.ReadString('\n')
	provider = strings.TrimSpace(strings.ToLower(provider))

	if provider == "" {
		provider = "openai"
	}

	if provider != "openai" && provider != "claude" {
		fmt.Println("Invalid provider, defaulting to 'openai'")
		provider = "openai"
	}
	cfg.AI.Provider = provider

	fmt.Printf("Enter your %s API key: ", strings.ToUpper(provider))
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)

	if apiKey != "" {
		if err := config.ValidateAPIKey(provider, apiKey); err != nil {
			fmt.Printf("Warning: %v\n", err)
			fmt.Print("Continue anyway? (y/n): ")
			confirm, _ := reader.ReadString('\n')
			if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
				return fmt.Errorf("setup cancelled")
			}
		}
		cfg.AI.APIKey = apiKey
	}

	fmt.Println()

	// Preferences
	fmt.Println("⚙️  Preferences")
	fmt.Println("────────────────────────────────────────────────────────────")

	fmt.Print("Output format (table/json/csv) [table]: ")
	format, _ := reader.ReadString('\n')
	format = strings.TrimSpace(strings.ToLower(format))

	if format == "" {
		format = "table"
	}

	if format == "table" || format == "json" || format == "csv" {
		cfg.Output.Format = format
	}

	fmt.Print("Enable colored output? (y/n) [y]: ")
	colors, _ := reader.ReadString('\n')
	colors = strings.TrimSpace(strings.ToLower(colors))

	cfg.Output.Colors = colors != "n"

	fmt.Println()

	// Save configuration
	fmt.Println("💾 Saving configuration...")

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	configPath, _ := config.GetConfigPath()
	fmt.Printf("✓ Configuration saved to: %s\n", configPath)
	fmt.Println()
	fmt.Println("✨ Setup complete! You can now start using SET CLI.")
	fmt.Println("   Try: set estimate --task \"Your task description\"")

	return nil
}

func listConfiguration() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	configPath, _ := config.GetConfigPath()

	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║              Current SET CLI Configuration                 ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Configuration file: %s\n", configPath)
	fmt.Println()

	// GitHub
	fmt.Println("📦 GitHub:")
	if cfg.GitHub.Token != "" {
		maskedToken := maskToken(cfg.GitHub.Token)
		fmt.Printf("   Token:         %s\n", maskedToken)
	} else {
		fmt.Println("   Token:         <not set>")
	}

	if cfg.GitHub.DefaultRepo != "" {
		fmt.Printf("   Default Repo:  %s\n", cfg.GitHub.DefaultRepo)
	} else {
		fmt.Println("   Default Repo:  <not set>")
	}
	fmt.Println()

	// AI
	fmt.Println("🤖 AI Provider:")
	fmt.Printf("   Provider:      %s\n", cfg.AI.Provider)
	fmt.Printf("   Model:         %s\n", cfg.AI.Model)
	if cfg.AI.APIKey != "" {
		maskedKey := maskToken(cfg.AI.APIKey)
		fmt.Printf("   API Key:       %s\n", maskedKey)
	} else {
		fmt.Println("   API Key:       <not set>")
	}
	fmt.Println()

	// Estimation
	fmt.Println("📊 Estimation Settings:")
	fmt.Printf("   Confidence:    %d%%\n", cfg.Estimation.ConfidenceThreshold)
	fmt.Printf("   Similar Tasks: %d\n", cfg.Estimation.MaxSimilarTasks)
	fmt.Println()

	// Output
	fmt.Println("🎨 Output Settings:")
	fmt.Printf("   Format:        %s\n", cfg.Output.Format)
	fmt.Printf("   Colors:        %v\n", cfg.Output.Colors)
	fmt.Println()

	// Configuration status
	if cfg.IsConfigured() {
		fmt.Println("✓ Configuration is complete and ready to use")
	} else {
		fmt.Println("⚠  Configuration is incomplete")
		fmt.Println("   Run: set configure --initial")
	}

	return nil
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

func validateConfiguration() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║           Validating SET CLI Configuration                ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Check if token is configured
	if cfg.GitHub.Token == "" {
		fmt.Println("❌ GitHub token not configured")
		fmt.Println("   Run: set configure --github-token \"your-token\"")
		return fmt.Errorf("github token not configured")
	}

	fmt.Println("🔍 Validating GitHub token...")

	// Validate token with repository if configured
	var result *github.ValidationResult
	if cfg.GitHub.DefaultRepo != "" {
		result, err = github.ValidateTokenAndRepo(cfg.GitHub.Token, cfg.GitHub.DefaultRepo)
	} else {
		result, err = github.ValidateToken(cfg.GitHub.Token)
	}

	if err != nil && !result.Valid {
		fmt.Printf("❌ Token validation failed: %s\n", result.Error)
		return fmt.Errorf("token validation failed")
	}

	// Display validation results
	fmt.Println()
	fmt.Println("📊 Validation Results:")
	fmt.Println("────────────────────────────────────────────────────────────")

	if result.Valid {
		fmt.Println("✅ Token is valid")
		fmt.Printf("   Authenticated as: %s\n", result.Username)
	} else {
		fmt.Println("❌ Token is invalid")
		return fmt.Errorf("invalid token")
	}

	// Display scopes
	if len(result.Scopes) > 0 {
		fmt.Println()
		fmt.Println("🔐 Token Scopes:")
		for _, scope := range result.Scopes {
			fmt.Printf("   • %s\n", scope)
		}
	}

	// Check required permissions
	fmt.Println()
	fmt.Println("📋 Required Permissions:")
	if result.HasRepoAccess {
		fmt.Println("   ✅ Repository access")
	} else {
		fmt.Println("   ❌ Repository access - Missing! (Required: 'repo' or 'public_repo' scope)")
	}

	if result.HasIssuesAccess {
		fmt.Println("   ✅ Issues access")
	} else {
		fmt.Println("   ⚠️  Issues access - Limited")
	}

	if result.HasPullsAccess {
		fmt.Println("   ✅ Pull requests access")
	} else {
		fmt.Println("   ⚠️  Pull requests access - Limited")
	}

	if result.HasProjectsAccess {
		fmt.Println("   ✅ GitHub Projects access (custom fields)")
	} else {
		fmt.Println("   ⚠️  GitHub Projects access - Not available (custom fields won't be fetched)")
	}

	// Check repository access
	if cfg.GitHub.DefaultRepo != "" {
		fmt.Println()
		fmt.Printf("📦 Repository: %s\n", cfg.GitHub.DefaultRepo)
		if result.RepoExists {
			fmt.Println("   ✅ Repository accessible")
			if result.RepoPrivate {
				fmt.Println("   🔒 Private repository")
			} else {
				fmt.Println("   🌐 Public repository")
			}

			// Check specific permissions
			fmt.Println()
			fmt.Println("   Testing specific permissions...")
			issues, pulls, err := github.CheckRepositoryPermissions(cfg.GitHub.Token, cfg.GitHub.DefaultRepo)
			if err != nil {
				fmt.Printf("   ⚠️  Could not test permissions: %v\n", err)
			} else {
				if issues {
					fmt.Println("   ✅ Can read issues")
				} else {
					fmt.Println("   ❌ Cannot read issues")
				}
				if pulls {
					fmt.Println("   ✅ Can read pull requests")
				} else {
					fmt.Println("   ❌ Cannot read pull requests")
				}
			}
		} else {
			fmt.Printf("   ❌ Cannot access repository\n")
			fmt.Printf("   Error: %s\n", result.Error)
			fmt.Println()
			fmt.Println("   Possible reasons:")
			fmt.Println("   • Repository doesn't exist")
			fmt.Println("   • Repository is private and token doesn't have access")
			fmt.Println("   • Token doesn't have 'repo' scope for private repos")
		}
	}

	// Display rate limit info
	fmt.Println()
	fmt.Println("📈 API Rate Limit:")
	fmt.Printf("   Limit:     %d requests/hour\n", result.RateLimit)
	fmt.Printf("   Remaining: %d requests\n", result.RateLimitRemaining)
	if result.RateLimitRemaining < 100 {
		fmt.Println("   ⚠️  Rate limit running low!")
	}

	// Validate AI configuration if present
	if cfg.AI.Provider != "" && cfg.AI.APIKey != "" {
		fmt.Println()
		fmt.Println("🤖 Validating AI Configuration...")
		fmt.Printf("   Provider: %s\n", cfg.AI.Provider)

		aiResult, aiErr := config.ValidateAIProvider(cfg.AI.Provider, cfg.AI.APIKey)
		if aiErr != nil {
			fmt.Printf("   ❌ AI API key validation failed: %s\n", aiResult.Error)
			fmt.Println()
			fmt.Println("   💡 Tip: Check your API key or set a valid one:")
			fmt.Printf("   set configure --ai-provider %s --ai-key \"your-key\"\n", cfg.AI.Provider)
		} else {
			fmt.Println("   ✅ AI API key is valid")
			fmt.Printf("   ✅ Model available: %s\n", aiResult.Model)
			fmt.Println()
			fmt.Println("   🚀 You can now use AI-powered estimation:")
			fmt.Println("   set estimate \"Add user authentication\"")
		}
	} else {
		fmt.Println()
		fmt.Println("🤖 AI Configuration:")
		fmt.Println("   ⚠️  AI provider not configured")
		fmt.Println()
		fmt.Println("   💡 To enable AI-powered estimation:")
		fmt.Println("   set configure --ai-provider openai --ai-key \"sk-...\"")
	}

	// Final assessment
	fmt.Println()
	fmt.Println("════════════════════════════════════════════════════════════")

	if result.Valid && result.HasRepoAccess {
		if cfg.GitHub.DefaultRepo == "" {
			fmt.Println("✅ Token is valid and has required permissions")
			fmt.Println()
			fmt.Println("💡 Tip: Set a default repository with:")
			fmt.Println("   set configure --default-repo \"owner/repo\"")
		} else if result.RepoExists {
			fmt.Println("✅ Configuration is valid and ready to use!")
			fmt.Println()
			fmt.Println("🚀 You can now run:")
			fmt.Println("   set sync       # Sync repository data (Sprint 2+)")
			fmt.Println("   set estimate   # Estimate tasks (Sprint 3+)")
		} else {
			fmt.Println("⚠️  Token is valid but cannot access the configured repository")
			fmt.Println()
			fmt.Println("   Please check:")
			fmt.Println("   • Repository name is correct")
			fmt.Println("   • Token has access to this repository")
			fmt.Println("   • Token has 'repo' scope for private repositories")
		}
	} else {
		fmt.Println("❌ Configuration validation failed")
		fmt.Println()
		fmt.Println("   Please fix the issues above and run:")
		fmt.Println("   set configure --validate")
	}

	fmt.Println()

	return nil
}
