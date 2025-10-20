package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"set/internal/config"
	"set/internal/storage"

	"github.com/spf13/cobra"
)

var clearConfirm bool

// devClearCmd clears all data
var devClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all data from storage",
	Long: `Clear all data from the local storage database.

This command removes:
- All issues
- All pull requests
- Repository metadata
- Last sync timestamps

⚠️  This operation is irreversible!

Examples:
  # Clear with confirmation
  set dev clear --confirm

  # Clear without prompt (dangerous!)
  set dev clear --confirm --force`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if !cfg.Developer.Enabled {
			return fmt.Errorf("developer mode is not enabled. Run 'set dev enable' first")
		}

		return nil
	},
	RunE: runDevClear,
}

func init() {
	devClearCmd.Flags().BoolVar(&clearConfirm, "confirm", false, "Confirm data deletion")
}

func runDevClear(cmd *cobra.Command, args []string) error {
	if !clearConfirm {
		fmt.Println("⚠️  This will delete ALL data from the local storage!")
		fmt.Println()
		fmt.Println("To confirm, run:")
		fmt.Println("  set dev clear --confirm")
		return nil
	}

	// Open storage
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	storePath := filepath.Join(homeDir, ".set", "data.db")
	store, err := storage.NewStore(storePath)
	if err != nil {
		return fmt.Errorf("failed to open storage: %w", err)
	}
	defer store.Close()

	// Get counts before clearing
	issueCount, err := store.CountIssues()
	if err != nil {
		issueCount = 0
	}

	prCount, err := store.CountPullRequests()
	if err != nil {
		prCount = 0
	}

	fmt.Println("🗑️  Clearing storage...")

	// Clear all data
	if err := store.Clear(); err != nil {
		return fmt.Errorf("failed to clear storage: %w", err)
	}

	fmt.Println("✅ Storage cleared!")
	fmt.Println()
	fmt.Println("📊 Removed:")
	fmt.Printf("   Issues: %d\n", issueCount)
	fmt.Printf("   PRs:    %d\n", prCount)
	fmt.Println()
	fmt.Println("💡 To repopulate with fake data:")
	fmt.Println("   set dev seed --count 50")

	return nil
}
