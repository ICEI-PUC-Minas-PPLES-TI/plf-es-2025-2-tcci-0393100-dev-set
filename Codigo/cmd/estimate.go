package cmd

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"set/internal/ai"
	"set/internal/config"
	"set/internal/estimator"
	"set/internal/logger"
	"set/internal/storage"

	"github.com/spf13/cobra"
)

var (
	taskDescription     string
	taskLabels          []string
	taskContext         map[string]string
	useSimilar          bool
	noAI                bool
	outputFormat        string
	showSimilar         bool
	similarityThreshold float64
)

// estimateCmd estimates task effort
var estimateCmd = &cobra.Command{
	Use:   "estimate <task-title>",
	Short: "Estimate task effort using AI and historical data",
	Long: `Estimate software development task effort using AI and historical data.

The estimate command uses multiple strategies to provide accurate estimates:
1. AI-powered estimation using OpenAI GPT models
2. Similarity matching with historical tasks
3. Custom fields from GitHub Projects (size, story points, hours)
4. Statistical analysis of past performance

Examples:
  # Basic estimation
  set estimate "Add user authentication"

  # With description
  set estimate "Add authentication" --description "Implement OAuth 2.0 login"

  # With labels for better matching
  set estimate "Fix login bug" --labels bug,high-priority

  # Show similar historical tasks
  set estimate "Add feature" --similar

  # Without AI (use historical data only)
  set estimate "Refactor code" --no-ai

  # Export as JSON
  set estimate "Task name" --output json

  # Export as CSV
  set estimate "Task name" --output csv`,
	Args: cobra.MinimumNArgs(1),
	RunE: runEstimate,
}

func init() {
	rootCmd.AddCommand(estimateCmd)

	estimateCmd.Flags().StringVarP(&taskDescription, "description", "d", "", "Task description")
	estimateCmd.Flags().StringSliceVarP(&taskLabels, "labels", "l", []string{}, "Task labels (comma-separated)")
	estimateCmd.Flags().StringToStringVarP(&taskContext, "context", "c", map[string]string{}, "Additional context (key=value)")
	estimateCmd.Flags().BoolVar(&useSimilar, "similar", false, "Show similar historical tasks")
	estimateCmd.Flags().BoolVar(&noAI, "no-ai", false, "Don't use AI, only historical data")
	estimateCmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "Output format (text, json, csv)")
	estimateCmd.Flags().BoolVar(&showSimilar, "show-similar", false, "Show details of similar tasks found")
	estimateCmd.Flags().Float64Var(&similarityThreshold, "similarity-threshold", 0.0, "Minimum similarity threshold (0.0-1.0, default: 0.3)")
}

func runEstimate(cmd *cobra.Command, args []string) error {
	// Get task title
	taskTitle := strings.Join(args, " ")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Check if we need AI
	useAI := !noAI && cfg.AI.Provider != "" && cfg.AI.APIKey != ""

	if !useAI && noAI {
		logger.Info("AI disabled by flag, using historical data only")
	} else if !useAI {
		logger.Warn("AI not configured, using historical data only")
		logger.Info("To enable AI: set configure --ai-provider openai --ai-key \"sk-...\"")
	}

	// Open storage
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	storePath := filepath.Join(homeDir, ".set", "data.db")
	store, err := storage.NewStore(storePath)
	if err != nil {
		logger.Warnf("Could not open storage: %v", err)
		logger.Info("Run 'set sync' to download historical data for better estimates")
		store = nil
	}
	if store != nil {
		defer store.Close()
	}

	// Create AI provider if configured
	var aiProvider ai.AIProvider
	if useAI {
		client := ai.NewOpenAIClient(cfg.AI.APIKey)
		// Set model from configuration
		if cfg.AI.Model != "" {
			client.SetModel(cfg.AI.Model)
		}
		aiProvider = client
		logger.Infof("Using AI provider: %s (model: %s)", cfg.AI.Provider, cfg.AI.Model)
	}

	// Create estimator (AI is always required now)
	estimatorConfig := estimator.DefaultEstimationConfig()
	if cfg.Estimation.MaxSimilarTasks > 0 {
		estimatorConfig.MaxSimilarTasks = cfg.Estimation.MaxSimilarTasks
	}
	// Allow overriding similarity threshold via flag
	if similarityThreshold > 0 {
		estimatorConfig.MinSimilarityThreshold = similarityThreshold
		logger.Infof("Using custom similarity threshold: %.1f%%", similarityThreshold*100)
	}
	// Note: useAI flag is ignored - AI is always required in this version

	est := estimator.NewEstimator(aiProvider, store, estimatorConfig)

	// Build task
	task := &estimator.Task{
		Title:       taskTitle,
		Description: taskDescription,
		Labels:      taskLabels,
		Context:     make(map[string]interface{}),
	}

	// Add context
	for key, value := range taskContext {
		task.Context[key] = value
	}

	// Print estimation request
	if outputFormat == "text" {
		fmt.Println("╭─ Task ───────────────────────────────────────────────────╮")
		fmt.Printf("│  %s\n", taskTitle)
		if taskDescription != "" {
			fmt.Printf("│  %s\n", taskDescription)
		}
		if len(taskLabels) > 0 {
			fmt.Printf("│  Labels: %s\n", strings.Join(taskLabels, ", "))
		}
		fmt.Println("╰───────────────────────────────────────────────────────────╯")
		fmt.Println()
	}

	// Perform estimation
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := est.Estimate(ctx, task)
	if err != nil {
		return fmt.Errorf("estimation failed: %w", err)
	}

	// Output results
	switch outputFormat {
	case "json":
		return outputEstimationJSON(result)
	case "csv":
		return outputEstimationCSV(result)
	default:
		return outputEstimationText(result, showSimilar || useSimilar)
	}
}

func outputEstimationText(result *estimator.EstimationResult, showSimilarTasks bool) error {
	if result.Estimation == nil {
		fmt.Println("✗ Could not generate estimation")
		return nil
	}

	// Use the formatted output from prompts
	fmt.Print(ai.FormatEstimationForDisplay(result.Estimation))

	// Show estimation method
	fmt.Printf("%s · %s\n", result.Method, result.DataSource)
	fmt.Println()

	// Show similar tasks if requested
	if showSimilarTasks && len(result.SimilarTasks) > 0 {
		fmt.Println("┃ Similar Tasks")

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "  #\tTitle\tMatch\tHours\tSize")
		fmt.Fprintln(w, "  "+strings.Repeat("─", 4)+"\t"+strings.Repeat("─", 30)+"\t"+strings.Repeat("─", 5)+"\t"+strings.Repeat("─", 5)+"\t"+strings.Repeat("─", 4))

		for i, match := range result.SimilarTasks {
			if i >= 5 { // Show top 5
				break
			}

			title := match.Task.Issue.Title
			if len(title) > 30 {
				title = title[:27] + "..."
			}

			hours := "-"
			if match.Task.ActualHours > 0 {
				hours = fmt.Sprintf("%.1f", match.Task.ActualHours)
			}

			size := match.Task.EstimatedSize
			if size == "" {
				size = "-"
			}

			fmt.Fprintf(w, "  #%d\t%s\t%.0f%%\t%s\t%s\n",
				match.Task.Issue.Number,
				title,
				match.Similarity*100,
				hours,
				size,
			)
		}
		w.Flush()
		fmt.Println()
	}

	// Summary
	confLevel := result.GetConfidenceLevel()
	if confLevel == "low" {
		fmt.Println("┃ Next Steps")
		fmt.Println("  · Confidence is low - consider breaking down the task")
		fmt.Println("  · Sync more historical data: set sync --custom-fields")
		if result.Method != "ai" {
			fmt.Println("  · Enable AI for better estimates: set configure --ai-provider openai")
		}
	} else if confLevel == "medium" {
		fmt.Println("┃ Next Steps")
		fmt.Println("  · Review the assumptions and risks above")
		fmt.Println("  · Consider adding custom fields to GitHub Projects")
	}

	return nil
}

func outputEstimationJSON(result *estimator.EstimationResult) error {
	output := map[string]interface{}{
		"task": map[string]interface{}{
			"title":       result.Task.Title,
			"description": result.Task.Description,
			"labels":      result.Task.Labels,
			"context":     result.Task.Context,
		},
		"estimation":       result.Estimation,
		"method":           result.Method,
		"data_source":      result.DataSource,
		"confidence_level": result.GetConfidenceLevel(),
	}

	if len(result.SimilarTasks) > 0 {
		similarTasks := make([]map[string]interface{}, len(result.SimilarTasks))
		for i, match := range result.SimilarTasks {
			similarTasks[i] = map[string]interface{}{
				"issue_number": match.Task.Issue.Number,
				"title":        match.Task.Issue.Title,
				"similarity":   match.Similarity,
				"actual_hours": match.Task.ActualHours,
				"size":         match.Task.EstimatedSize,
				"story_points": match.Task.StoryPoints,
			}
		}
		output["similar_tasks"] = similarTasks
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func outputEstimationCSV(result *estimator.EstimationResult) error {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	// Header
	if err := w.Write([]string{
		"Task Title",
		"Estimated Hours",
		"Size",
		"Story Points",
		"Confidence %",
		"Method",
		"Reasoning",
	}); err != nil {
		return err
	}

	// Data
	if result.Estimation != nil {
		if err := w.Write([]string{
			result.Task.Title,
			fmt.Sprintf("%.1f", result.Estimation.EstimatedHours),
			result.Estimation.EstimatedSize,
			fmt.Sprintf("%.0f", result.Estimation.StoryPoints),
			fmt.Sprintf("%.0f", result.Estimation.ConfidenceScore*100),
			result.Method,
			result.Estimation.Reasoning,
		}); err != nil {
			return err
		}
	}

	return nil
}
