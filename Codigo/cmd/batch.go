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
	batchInputFile    string
	batchOutputFile   string
	batchOutputFormat string
	batchMaxWorkers   int
	batchShowProgress bool
)

// batchCmd processes batch estimation requests
var batchCmd = &cobra.Command{
	Use:   "batch",
	Short: "Process batch estimation requests",
	Long: `Process multiple task estimations in batch mode.

The batch command reads a file containing multiple tasks and estimates them
in parallel, providing a consolidated report with aggregate statistics.

Input Formats:
  • JSON: Structured batch request with metadata and tasks
  • CSV: Simple CSV with columns: id, title, description, labels, priority, assignee

Output Formats:
  • text: Human-readable report (default)
  • json: JSON format for programmatic processing
  • csv: CSV format for spreadsheet analysis
  • markdown: Markdown report for documentation

Examples:
  # Process JSON batch file
  set batch --file tasks.json

  # Process CSV file with custom workers
  set batch --file tasks.csv --workers 10

  # Export results to JSON
  set batch --file tasks.json --output results.json --format json

  # Export markdown report
  set batch --file tasks.csv --output report.md --format markdown

  # With progress indicator
  set batch --file tasks.json --progress`,
	RunE: runBatch,
}

func init() {
	rootCmd.AddCommand(batchCmd)

	batchCmd.Flags().StringVarP(&batchInputFile, "file", "f", "", "Input file (JSON or CSV) - REQUIRED")
	batchCmd.Flags().StringVarP(&batchOutputFile, "output", "o", "", "Output file path (optional, prints to stdout if not specified)")
	batchCmd.Flags().StringVar(&batchOutputFormat, "format", "text", "Output format: text, json, csv, markdown")
	batchCmd.Flags().IntVarP(&batchMaxWorkers, "workers", "w", 5, "Maximum concurrent workers")
	batchCmd.Flags().BoolVarP(&batchShowProgress, "progress", "p", false, "Show progress indicator")

	batchCmd.MarkFlagRequired("file")
}

func runBatch(cmd *cobra.Command, args []string) error {
	// Validate input file
	if batchInputFile == "" {
		return fmt.Errorf("input file is required (--file)")
	}

	if _, err := os.Stat(batchInputFile); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", batchInputFile)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// AI is required for batch estimation
	if cfg.AI.Provider == "" || cfg.AI.APIKey == "" {
		return fmt.Errorf("AI provider is required for batch estimation. Run: set configure --ai-provider openai --ai-key \"sk-...\"")
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

	// Create AI provider
	client := ai.NewOpenAIClient(cfg.AI.APIKey)
	if cfg.AI.Model != "" {
		client.SetModel(cfg.AI.Model)
	}

	// Create estimator
	estimatorConfig := estimator.DefaultEstimationConfig()
	if cfg.Estimation.MaxSimilarTasks > 0 {
		estimatorConfig.MaxSimilarTasks = cfg.Estimation.MaxSimilarTasks
	}
	est := estimator.NewEstimator(client, store, estimatorConfig)

	// Create batch processor
	processor := estimator.NewBatchProcessor(est, batchMaxWorkers)

	// Load batch request based on file extension
	var request *estimator.BatchRequest
	ext := strings.ToLower(filepath.Ext(batchInputFile))

	logger.Infof("Loading batch request from: %s", batchInputFile)

	switch ext {
	case ".json":
		request, err = estimator.LoadBatchRequest(batchInputFile)
	case ".csv":
		request, err = estimator.LoadBatchRequestFromCSV(batchInputFile)
	default:
		return fmt.Errorf("unsupported file format: %s (use .json or .csv)", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to load batch request: %w", err)
	}

	logger.Infof("Loaded %d tasks for batch estimation", len(request.Tasks))

	// Print header
	fmt.Println("🚀 Batch Estimation")
	fmt.Println(strings.Repeat("═", 60))
	fmt.Printf("Input File: %s\n", batchInputFile)
	fmt.Printf("Tasks: %d\n", len(request.Tasks))
	fmt.Printf("Workers: %d\n", batchMaxWorkers)
	fmt.Printf("Model: %s\n", cfg.AI.Model)
	fmt.Println(strings.Repeat("═", 60))
	fmt.Println()

	// Process batch
	ctx := context.Background()
	startTime := time.Now()

	// Show progress if requested
	if batchShowProgress {
		fmt.Println("Processing tasks...")
		fmt.Println()
	}

	report, err := processor.ProcessBatch(ctx, request)
	if err != nil {
		return fmt.Errorf("batch processing failed: %w", err)
	}

	duration := time.Since(startTime)

	// Print completion summary
	fmt.Println()
	fmt.Println(strings.Repeat("═", 60))
	fmt.Printf("✅ Batch Processing Complete\n")
	fmt.Printf("Duration: %s\n", duration.Round(time.Second))
	fmt.Printf("Success: %d/%d tasks\n", report.SuccessfulTasks, report.TotalTasks)
	if report.FailedTasks > 0 {
		fmt.Printf("Failed: %d tasks\n", report.FailedTasks)
	}
	fmt.Println(strings.Repeat("═", 60))
	fmt.Println()

	// Generate output
	var outputData []byte

	switch batchOutputFormat {
	case "json":
		outputData, err = generateJSONReport(report)
	case "csv":
		outputData, err = generateCSVReport(report)
	case "markdown", "md":
		outputData, err = generateMarkdownReport(report)
	default:
		err = printTextReport(report)
		return err
	}

	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Write output
	if batchOutputFile != "" {
		if err := os.WriteFile(batchOutputFile, outputData, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		logger.Infof("Report saved to: %s", batchOutputFile)
	} else {
		fmt.Print(string(outputData))
	}

	return nil
}

func printTextReport(report *estimator.BatchReport) error {
	fmt.Println("╭─ Batch Estimation Report ─────────────────────────────────╮")
	fmt.Println("╰───────────────────────────────────────────────────────────╯")
	fmt.Println()

	// Overall statistics
	stats := report.Statistics
	if stats != nil {
		fmt.Println("╭─ Overall Statistics ──────────────────────────────────────╮")
		fmt.Println("│                                                           │")
		fmt.Printf("│  Total Hours      %-40.1f │\n", stats.TotalEstimatedHours)
		fmt.Printf("│  Average          %-40.1f │\n", stats.AverageHours)
		fmt.Printf("│  Median           %-40.1f │\n", stats.MedianHours)
		fmt.Printf("│  Range            %-20.1f - %-17.1f │\n", stats.MinHours, stats.MaxHours)
		fmt.Printf("│  Avg Confidence   %-39.0f%% │\n", stats.AverageConfidence*100)
		fmt.Println("│                                                           │")
		fmt.Println("╰───────────────────────────────────────────────────────────╯")
		fmt.Println()

		// Size distribution
		fmt.Println("┃ Size Distribution")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "  Size\tCount\tTotal Hours\tAvg Hours")
		fmt.Fprintln(w, "  "+strings.Repeat("─", 4)+"\t"+strings.Repeat("─", 5)+"\t"+strings.Repeat("─", 11)+"\t"+strings.Repeat("─", 9))

		for _, size := range []string{"XS", "S", "M", "L", "XL"} {
			count := stats.SizeDistribution[size]
			if count > 0 {
				totalHours := stats.SizeHours[size]
				avgHours := totalHours / float64(count)
				fmt.Fprintf(w, "  %s\t%d\t%.1f\t%.1f\n", size, count, totalHours, avgHours)
			}
		}
		w.Flush()
		fmt.Println()

		// Confidence distribution
		fmt.Println("┃ Confidence Distribution")
		fmt.Printf("  High (≥70%%)       %d tasks\n", stats.HighConfidenceTasks)
		fmt.Printf("  Medium (50-69%%)   %d tasks\n", stats.MediumConfidenceTasks)
		fmt.Printf("  Low (<50%%)        %d tasks\n", stats.LowConfidenceTasks)
		fmt.Println()

		// Category distribution
		if len(stats.CategoryDistribution) > 0 {
			fmt.Println("┃ Category Distribution")
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "  Category\tCount\tTotal Hours")
			fmt.Fprintln(w, "  "+strings.Repeat("─", 15)+"\t"+strings.Repeat("─", 5)+"\t"+strings.Repeat("─", 11))

			for category, count := range stats.CategoryDistribution {
				hours := stats.CategoryHours[category]
				fmt.Fprintf(w, "  %s\t%d\t%.1f\n", category, count, hours)
			}
			w.Flush()
			fmt.Println()
		}
	}

	// Individual results
	fmt.Println("┃ Task Results")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  ID\tTitle\tHours\tSize\tPoints\tConfidence")
	fmt.Fprintln(w, "  "+strings.Repeat("─", 10)+"\t"+strings.Repeat("─", 30)+"\t"+strings.Repeat("─", 5)+"\t"+strings.Repeat("─", 4)+"\t"+strings.Repeat("─", 6)+"\t"+strings.Repeat("─", 10))

	for _, result := range report.Results {
		title := result.Task.Title
		if len(title) > 30 {
			title = title[:27] + "..."
		}

		if result.Error != nil {
			fmt.Fprintf(w, "  %s\t%s\t✗ ERROR\t-\t-\t-\n",
				result.Task.ID,
				title)
		} else if result.Result != nil && result.Result.Estimation != nil {
			est := result.Result.Estimation
			fmt.Fprintf(w, "  %s\t%s\t%.1f\t%s\t%.0f\t%.0f%%\n",
				result.Task.ID,
				title,
				est.EstimatedHours,
				est.EstimatedSize,
				est.StoryPoints,
				est.ConfidenceScore*100)
		}
	}
	w.Flush()
	fmt.Println()

	// Errors summary
	if report.FailedTasks > 0 {
		fmt.Println("┃ Failed Tasks")
		for _, result := range report.Results {
			if result.Error != nil {
				fmt.Printf("  · %s: %v\n", result.Task.ID, result.Error)
			}
		}
		fmt.Println()
	}

	fmt.Println(strings.Repeat("─", 60))

	return nil
}

func generateJSONReport(report *estimator.BatchReport) ([]byte, error) {
	output := map[string]interface{}{
		"metadata": map[string]interface{}{
			"total_tasks":      report.TotalTasks,
			"successful_tasks": report.SuccessfulTasks,
			"failed_tasks":     report.FailedTasks,
			"start_time":       report.StartTime,
			"end_time":         report.EndTime,
			"duration_seconds": report.Duration.Seconds(),
		},
		"statistics": report.Statistics,
		"results":    make([]map[string]interface{}, 0),
	}

	// Add individual results
	results := make([]map[string]interface{}, 0, len(report.Results))
	for _, result := range report.Results {
		resultMap := map[string]interface{}{
			"task_id":      result.Task.ID,
			"task_title":   result.Task.Title,
			"processed_at": result.ProcessedAt,
		}

		if result.Error != nil {
			resultMap["error"] = result.Error.Error()
		} else if result.Result != nil && result.Result.Estimation != nil {
			resultMap["estimation"] = result.Result.Estimation
			resultMap["method"] = result.Result.Method
			resultMap["data_source"] = result.Result.DataSource
		}

		results = append(results, resultMap)
	}
	output["results"] = results

	return json.MarshalIndent(output, "", "  ")
}

func generateCSVReport(report *estimator.BatchReport) ([]byte, error) {
	var buf strings.Builder
	w := csv.NewWriter(&buf)

	// Header
	if err := w.Write([]string{
		"Task ID",
		"Title",
		"Estimated Hours",
		"Size",
		"Story Points",
		"Confidence %",
		"Method",
		"Status",
		"Error",
	}); err != nil {
		return nil, err
	}

	// Data rows
	for _, result := range report.Results {
		row := []string{
			result.Task.ID,
			result.Task.Title,
		}

		if result.Error != nil {
			row = append(row, "-", "-", "-", "-", "-", "failed", result.Error.Error())
		} else if result.Result != nil && result.Result.Estimation != nil {
			est := result.Result.Estimation
			row = append(row,
				fmt.Sprintf("%.1f", est.EstimatedHours),
				est.EstimatedSize,
				fmt.Sprintf("%.0f", est.StoryPoints),
				fmt.Sprintf("%.0f", est.ConfidenceScore*100),
				result.Result.Method,
				"success",
				"",
			)
		}

		if err := w.Write(row); err != nil {
			return nil, err
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}

	return []byte(buf.String()), nil
}

func generateMarkdownReport(report *estimator.BatchReport) ([]byte, error) {
	var md strings.Builder

	// Header
	md.WriteString("# Batch Estimation Report\n\n")
	md.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// Summary
	md.WriteString("## Summary\n\n")
	md.WriteString(fmt.Sprintf("- **Total Tasks:** %d\n", report.TotalTasks))
	md.WriteString(fmt.Sprintf("- **Successful:** %d\n", report.SuccessfulTasks))
	md.WriteString(fmt.Sprintf("- **Failed:** %d\n", report.FailedTasks))
	md.WriteString(fmt.Sprintf("- **Duration:** %s\n\n", report.Duration.Round(time.Second)))

	// Statistics
	if stats := report.Statistics; stats != nil {
		md.WriteString("## Statistics\n\n")
		md.WriteString("### Overall Metrics\n\n")
		md.WriteString(fmt.Sprintf("- **Total Estimated Hours:** %.1f\n", stats.TotalEstimatedHours))
		md.WriteString(fmt.Sprintf("- **Average Hours:** %.1f\n", stats.AverageHours))
		md.WriteString(fmt.Sprintf("- **Median Hours:** %.1f\n", stats.MedianHours))
		md.WriteString(fmt.Sprintf("- **Range:** %.1f - %.1f hours\n", stats.MinHours, stats.MaxHours))
		md.WriteString(fmt.Sprintf("- **Average Confidence:** %.0f%%\n\n", stats.AverageConfidence*100))

		// Size distribution
		md.WriteString("### Size Distribution\n\n")
		md.WriteString("| Size | Count | Total Hours | Avg Hours |\n")
		md.WriteString("|------|-------|-------------|-----------|\n")
		for _, size := range []string{"XS", "S", "M", "L", "XL"} {
			if count := stats.SizeDistribution[size]; count > 0 {
				totalHours := stats.SizeHours[size]
				avgHours := totalHours / float64(count)
				md.WriteString(fmt.Sprintf("| %s | %d | %.1f | %.1f |\n", size, count, totalHours, avgHours))
			}
		}
		md.WriteString("\n")

		// Confidence distribution
		md.WriteString("### Confidence Distribution\n\n")
		md.WriteString(fmt.Sprintf("- **High (≥70%%):** %d tasks\n", stats.HighConfidenceTasks))
		md.WriteString(fmt.Sprintf("- **Medium (50-69%%):** %d tasks\n", stats.MediumConfidenceTasks))
		md.WriteString(fmt.Sprintf("- **Low (<50%%):** %d tasks\n\n", stats.LowConfidenceTasks))
	}

	// Individual results
	md.WriteString("## Task Results\n\n")
	md.WriteString("| ID | Title | Hours | Size | Points | Confidence | Status |\n")
	md.WriteString("|----|-------|-------|------|--------|------------|--------|\n")

	for _, result := range report.Results {
		if result.Error != nil {
			md.WriteString(fmt.Sprintf("| %s | %s | - | - | - | - | ❌ Error |\n",
				result.Task.ID, result.Task.Title))
		} else if result.Result != nil && result.Result.Estimation != nil {
			est := result.Result.Estimation
			md.WriteString(fmt.Sprintf("| %s | %s | %.1f | %s | %.0f | %.0f%% | ✅ |\n",
				result.Task.ID,
				result.Task.Title,
				est.EstimatedHours,
				est.EstimatedSize,
				est.StoryPoints,
				est.ConfidenceScore*100))
		}
	}
	md.WriteString("\n")

	// Errors
	if report.FailedTasks > 0 {
		md.WriteString("## Errors\n\n")
		for _, result := range report.Results {
			if result.Error != nil {
				md.WriteString(fmt.Sprintf("- **%s**: %v\n", result.Task.ID, result.Error))
			}
		}
		md.WriteString("\n")
	}

	md.WriteString("---\n")
	md.WriteString("*Generated by SET CLI Estimation Engine v2.0*\n")

	return []byte(md.String()), nil
}
