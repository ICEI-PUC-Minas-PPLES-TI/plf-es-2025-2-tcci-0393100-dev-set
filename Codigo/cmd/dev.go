package cmd

import (
	"fmt"

	"set/internal/config"

	"github.com/spf13/cobra"
)

// devCmd provides developer utilities
var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Developer utilities and testing tools",
	Long: `Developer mode utilities for testing and development.

This command provides various tools for developers to test and debug
the SET CLI without affecting production data.

Subcommands:
  enable     Enable developer mode
  disable    Disable developer mode
  seed       Seed database with fake estimation data
  clear      Clear all data (storage + cache)
  status     Show developer mode status

Examples:
  # Enable developer mode
  set dev enable

  # Seed with fake data
  set dev seed --count 50

  # Check status
  set dev status`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var (
	enableDebugMode bool
)

// devEnableCmd enables developer mode
var devEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable developer mode",
	Long: `Enable developer mode for testing and development.

When enabled, developer mode provides:
- Additional debug logging
- Access to testing utilities
- Ability to seed fake data
- Safe data clearing

Flags:
  --debug    Enable debug mode logging`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		cfg.Developer.Enabled = true
		cfg.Developer.DebugMode = enableDebugMode

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Println("✅ Developer mode enabled")
		if enableDebugMode {
			fmt.Println("🐛 Debug mode enabled")
		}
		fmt.Println()
		fmt.Println("Available commands:")
		fmt.Println("  set dev seed    - Seed database with fake data")
		fmt.Println("  set dev clear   - Clear all data")
		fmt.Println("  set dev status  - Show status")
		fmt.Println()
		fmt.Println("⚠️  Developer mode should only be used for testing!")

		return nil
	},
}

// devDisableCmd disables developer mode
var devDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable developer mode",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		cfg.Developer.Enabled = false
		cfg.Developer.DebugMode = false

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Println("✅ Developer mode disabled")
		return nil
	},
}

// devStatusCmd shows developer mode status
var devStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show developer mode status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		fmt.Println("🔧 Developer Mode Status")
		fmt.Println("─────────────────────────")
		if cfg.Developer.Enabled {
			fmt.Println("Status:      ✅ ENABLED")
		} else {
			fmt.Println("Status:      ❌ DISABLED")
		}
		fmt.Printf("Debug Mode:  %v\n", cfg.Developer.DebugMode)
		fmt.Printf("Data Dir:    %s\n", cfg.Developer.FakeDataDir)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(devCmd)

	// Add subcommands
	devCmd.AddCommand(devEnableCmd)
	devCmd.AddCommand(devDisableCmd)
	devCmd.AddCommand(devStatusCmd)
	devCmd.AddCommand(devSeedCmd)
	devCmd.AddCommand(devClearCmd)

	// Flags for enable command
	devEnableCmd.Flags().BoolVar(&enableDebugMode, "debug", false, "Enable debug mode logging")
}
