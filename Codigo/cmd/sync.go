package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"set/internal/config"
	"set/internal/github"
	"set/internal/logger"
	"set/internal/storage"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var (
	fullSync            bool
	forceSync           bool
	issuesOnly          bool
	prsOnly             bool
	includeCustomFields bool
)

// syncCmd synchronizes GitHub repository data to local storage
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize GitHub repository data",
	Long: `Synchronize GitHub repository issues and pull requests to local storage.

The sync command fetches data from the configured GitHub repository and stores
it locally using BoltDB. This allows for offline analysis and faster queries.

By default, sync performs an incremental sync (only fetching updated items since
the last sync). Use --full to sync all data from scratch.

Examples:
  # Sync issues and pull requests (incremental)
  set sync

  # Full sync of all data
  set sync --full

  # Sync only issues
  set sync --issues-only

  # Sync only pull requests
  set sync --prs-only

  # Force sync even if recently synced
  set sync --force

  # Sync with GitHub Projects custom fields (size, priority, estimations, etc.)
  set sync --custom-fields`,
	RunE: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)

	syncCmd.Flags().BoolVar(&fullSync, "full", false, "Perform a full sync (ignore last sync time)")
	syncCmd.Flags().BoolVar(&forceSync, "force", false, "Force sync even if recently synced")
	syncCmd.Flags().BoolVar(&issuesOnly, "issues-only", false, "Sync only issues")
	syncCmd.Flags().BoolVar(&prsOnly, "prs-only", false, "Sync only pull requests")
	syncCmd.Flags().BoolVar(&includeCustomFields, "custom-fields", false, "Include GitHub Projects custom fields (requires Projects API access)")
}

func runSync(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate required configuration
	if cfg.GitHub.Token == "" {
		return fmt.Errorf("GitHub token not configured. Run 'set configure' first")
	}
	if cfg.GitHub.DefaultRepo == "" {
		return fmt.Errorf("repository not configured. Run 'set configure' first")
	}

	// Parse repository (owner/repo)
	parts := strings.Split(cfg.GitHub.DefaultRepo, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository format. Expected 'owner/repo', got '%s'", cfg.GitHub.DefaultRepo)
	}
	owner, repo := parts[0], parts[1]

	logger.Infof("Starting sync for %s/%s", owner, repo)

	// Initialize GitHub client
	ghClient := github.NewClient(cfg.GitHub.Token)

	// Initialize storage
	storePath := getStoragePath(cfg)
	store, err := storage.NewStore(storePath)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer store.Close()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Fetch repository information
	logger.Info("Fetching repository information...")
	repoInfo, err := ghClient.GetRepository(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to fetch repository: %w", err)
	}
	logger.Infof("Repository: %s", repoInfo.FullName)

	// Save repository info
	if err := store.SetRepository(repoInfo); err != nil {
		logger.Warnf("Failed to save repository info: %v", err)
	}

	// Check last sync time
	lastSync, err := store.GetLastSync()
	if err != nil {
		logger.Warnf("Failed to get last sync time: %v", err)
	}

	if lastSync != nil && !forceSync && !fullSync {
		timeSince := time.Since(*lastSync)
		if timeSince < 5*time.Minute {
			logger.Infof("Last sync was %v ago (less than 5 minutes). Use --force to sync anyway.", timeSince.Round(time.Second))
			return nil
		}
		logger.Infof("Last sync: %v ago", timeSince.Round(time.Second))
	}

	// Determine sync mode
	var since *time.Time
	if !fullSync && lastSync != nil {
		since = lastSync
		logger.Info("Performing incremental sync")
	} else {
		logger.Info("Performing full sync")
	}

	// Sync issues (unless prs-only)
	if !prsOnly {
		if err := syncIssues(ctx, ghClient, store, owner, repo, since); err != nil {
			return fmt.Errorf("failed to sync issues: %w", err)
		}
	}

	// Sync pull requests (unless issues-only)
	if !issuesOnly {
		if err := syncPullRequests(ctx, ghClient, store, owner, repo, since); err != nil {
			return fmt.Errorf("failed to sync pull requests: %w", err)
		}
	}

	// Update last sync time
	if err := store.SetLastSync(time.Now()); err != nil {
		logger.Warnf("Failed to update last sync time: %v", err)
	}

	// Print summary
	printSyncSummary(store)

	logger.Info("Sync completed successfully")
	return nil
}

func syncIssues(ctx context.Context, client *github.Client, store *storage.Store, owner, repo string, since *time.Time) error {
	logger.Info("Syncing issues...")

	opts := github.DefaultFetchOptions()
	opts.Since = since
	opts.IncludeCustomFields = includeCustomFields

	// Create progress bar
	bar := progressbar.NewOptions(-1,
		progressbar.OptionSetDescription("Fetching issues"),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionThrottle(100*time.Millisecond),
	)

	// Fetch issues with progress updates
	issues, err := client.FetchIssues(ctx, owner, repo, opts)
	bar.Finish()

	if err != nil {
		return err
	}

	if len(issues) == 0 {
		logger.Info("No new issues to sync")
		return nil
	}

	// Save issues in batches
	logger.Infof("Saving %d issues...", len(issues))
	batchSize := 100
	for i := 0; i < len(issues); i += batchSize {
		end := i + batchSize
		if end > len(issues) {
			end = len(issues)
		}

		if err := store.SaveIssues(issues[i:end]); err != nil {
			return fmt.Errorf("failed to save issues batch: %w", err)
		}
	}

	logger.Infof("✓ Synced %d issues", len(issues))
	return nil
}

func syncPullRequests(ctx context.Context, client *github.Client, store *storage.Store, owner, repo string, since *time.Time) error {
	logger.Info("Syncing pull requests...")

	opts := github.DefaultFetchOptions()
	opts.Since = since
	opts.IncludeCustomFields = includeCustomFields

	// Create progress bar
	bar := progressbar.NewOptions(-1,
		progressbar.OptionSetDescription("Fetching pull requests"),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionThrottle(100*time.Millisecond),
	)

	// Fetch pull requests with progress updates
	prs, err := client.FetchPullRequests(ctx, owner, repo, opts)
	bar.Finish()

	if err != nil {
		return err
	}

	if len(prs) == 0 {
		logger.Info("No new pull requests to sync")
		return nil
	}

	// Save pull requests in batches
	logger.Infof("Saving %d pull requests...", len(prs))
	batchSize := 100
	for i := 0; i < len(prs); i += batchSize {
		end := i + batchSize
		if end > len(prs) {
			end = len(prs)
		}

		if err := store.SavePullRequests(prs[i:end]); err != nil {
			return fmt.Errorf("failed to save pull requests batch: %w", err)
		}
	}

	logger.Infof("✓ Synced %d pull requests", len(prs))
	return nil
}

func printSyncSummary(store *storage.Store) {
	issueCount, _ := store.CountIssues()
	prCount, _ := store.CountPullRequests()

	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Println("Sync Complete ✓")
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Issues:        %d\n", issueCount)
	fmt.Printf("Pull Requests: %d\n", prCount)
	fmt.Println(strings.Repeat("─", 60))
}

func getStoragePath(cfg *config.Config) string {
	// Use default path in user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Warnf("Failed to get home directory: %v", err)
		return "set.db"
	}

	// Create .set directory if it doesn't exist
	setDir := filepath.Join(homeDir, ".set")
	if err := os.MkdirAll(setDir, 0755); err != nil {
		logger.Warnf("Failed to create .set directory: %v", err)
		return "set.db"
	}

	return filepath.Join(setDir, "data.db")
}
