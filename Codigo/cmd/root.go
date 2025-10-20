/*
Copyright © 2025 Inácio Moraes da Silva
*/
package cmd

import (
	"fmt"
	"os"

	"set/internal/logger"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	version = "0.1.0" // Version will be set during build
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "set",
	Short: "Software Estimation Tool - AI-powered effort estimation for development tasks",
	Long: `
╭─ SET CLI ─────────────────────────────────────────────────────╮
│  Software Estimation Tool                                     │
│  AI-Powered Effort Estimation for Agile Teams                 │
╰───────────────────────────────────────────────────────────────╯

A powerful CLI tool that combines AI intelligence with historical data
to provide accurate software development effort estimates.

┃ Core Features

  · AI-Powered Estimations
    Uses OpenAI GPT models to analyze task complexity, dependencies,
    and risks. Provides estimates in hours, story points, and t-shirt
    sizes (XS, S, M, L, XL).

  · Historical Data Analysis
    Learns from your GitHub repository history using advanced similarity
    algorithms (TF-IDF + cosine similarity). Matches new tasks with
    similar past work for data-driven estimates.

  · Batch Processing
    Estimate entire sprint backlogs in seconds with parallel processing.
    Supports JSON and CSV input formats. Perfect for sprint planning.

  · Multiple Export Formats
    Export results as JSON, CSV, Markdown, JIRA, or GitHub format for
    seamless integration with your existing tools.

  · Team Performance Analytics
    Track estimation accuracy, analyze trends by category, and optimize
    team velocity with detailed statistics.

  · GitHub Projects Integration
    Sync custom fields like story points, estimated hours, and priority
    from GitHub Projects V2 for enhanced accuracy.

┃ Perfect For

  · Developers          Accurate task estimates for better planning
  · Scrum Masters       Sprint capacity planning and backlog refinement
  · Product Owners      Roadmap planning and release forecasting
  · Engineering Leads   Resource allocation and timeline estimation

┃ Quick Start

  1. Initial Setup (interactive configuration)
     $ set configure --initial

  2. Load Historical Data (from GitHub or sample dataset)
     $ set sync --custom-fields
     $ set dev seed --count 100 --with-custom-fields

  3. Estimate a Single Task
     $ set estimate "Add user authentication" --labels backend,security

  4. Batch Process Sprint Backlog
     $ set batch --file sprint-backlog.json

  5. Export Results
     $ set export --format csv --output estimates.csv

┃ Common Workflows

  Sprint Planning:
    $ set sync --custom-fields
    $ set batch --file sprint-15.json --output sprint-15-report.md

  Single Task Estimation:
    $ set estimate "Fix memory leak" --description "Cache not releasing" --show-similar

  Data Analysis:
    $ set inspect
    $ set inspect --list --custom
    $ set export --format json --output data.json

┃ Configuration

  Config file: ~/.set.yaml (auto-created on first run)

  Required Settings:
    - AI Provider (OpenAI)
    - API Key (OpenAI API key)
    - Model (gpt-4, gpt-4-turbo, gpt-3.5-turbo)

  Optional Settings:
    - GitHub Token (for syncing repositories)
    - Default Repository (owner/repo)
    - Similarity Threshold (0.0-1.0, default: 0.3)
    - Max Similar Tasks (default: 10)

┃ Need Help?

  set [command] --help     Detailed help for any command
  set version              Show version information
  set configure list       View current configuration

For documentation, examples, and troubleshooting:
  https://github.com/ICEI-PUC-Minas-PPLES-TI/plf-es-2025-2-tcci-0393100-dev-inacio-moraes/tree/main/Codigo

────────────────────────────────────────────────────────────────`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.set.yaml)")
	rootCmd.PersistentFlags().String("log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().Bool("verbose", false, "enable verbose logging")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Initialize logger early
	initLogger()

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".set" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".set")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

// initLogger initializes the logger based on flags
func initLogger() {
	// Get log level from flags
	logLevel, _ := rootCmd.PersistentFlags().GetString("log-level")
	verbose, _ := rootCmd.PersistentFlags().GetBool("verbose")

	// If verbose is set, override log level to info (or debug if log-level was already set to debug)
	if verbose && logLevel == "info" {
		logLevel = "info" // Show INFO and above
	} else if verbose {
		logLevel = "debug" // Show everything if debug was explicitly requested
	} else if logLevel == "info" {
		// Default mode: only show warnings and errors
		logLevel = "warn"
	}

	// Initialize logger with console output enabled
	logger.Init(logLevel, true)
}
