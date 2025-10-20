package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"set/internal/config"
	"set/internal/github"
	"set/internal/logger"
	"set/internal/storage"

	"github.com/spf13/cobra"
)

var (
	exportFormat        string
	exportOutput        string
	exportFilter        string
	exportDateFrom      string
	exportDateTo        string
	exportIncludeClosed bool
)

// exportCmd exports historical data to various formats
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export historical data to various formats",
	Long: `Export stored historical data and estimation results to various formats.

Supported formats:
  • csv         - Comma-separated values (spreadsheet compatible)
  • json        - JSON format (API/automation friendly)
  • jira        - Jira-compatible CSV import format
  • github      - GitHub Projects CSV import format
  • excel       - Excel-compatible CSV with rich data
  • markdown    - Markdown table format

Examples:
  # Export all historical issues to CSV
  set export --format csv --output issues.csv

  # Export to Jira import format
  set export --format jira --output jira-import.csv

  # Export to GitHub Projects format
  set export --format github --output github-import.csv

  # Export from specific date range
  set export --date-from 2025-09-19 --output recent.json

  # Filter by label
  set export --filter bug --format csv --output bugs.csv

  # Include closed issues
  set export --include-closed --output all-issues.csv`,
	RunE: runExport,
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "csv", "Export format: csv, json, jira, github, excel, markdown")
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "Output file path (prints to stdout if not specified)")
	exportCmd.Flags().StringVar(&exportFilter, "filter", "", "Filter by label")
	exportCmd.Flags().StringVar(&exportDateFrom, "date-from", "", "Start date (YYYY-MM-DD)")
	exportCmd.Flags().StringVar(&exportDateTo, "date-to", "", "End date (YYYY-MM-DD)")
	exportCmd.Flags().BoolVar(&exportIncludeClosed, "include-closed", true, "Include closed issues (default: true)")
}

func runExport(cmd *cobra.Command, args []string) error {
	// Load configuration
	_, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Open storage
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	storePath := filepath.Join(homeDir, ".set", "data.db")
	store, err := storage.NewStore(storePath)
	if err != nil {
		return fmt.Errorf("failed to open storage: %w (run 'set sync' first)", err)
	}
	defer store.Close()

	logger.Infof("Exporting data from storage")

	// Fetch issues from storage
	issues, err := store.GetAllIssues()
	if err != nil {
		return fmt.Errorf("failed to get issues: %w", err)
	}

	if len(issues) == 0 {
		logger.Warn("No issues found in storage. Run 'set sync' to download data.")
		return fmt.Errorf("no data to export")
	}

	// Apply filters
	filteredIssues := applyExportFilters(issues)

	logger.Infof("Found %d issues (filtered from %d total)", len(filteredIssues), len(issues))

	// Generate export
	var outputData []byte

	switch strings.ToLower(exportFormat) {
	case "csv":
		outputData, err = exportToCSV(filteredIssues)
	case "json":
		outputData, err = exportToJSON(filteredIssues)
	case "jira":
		outputData, err = exportToJira(filteredIssues)
	case "github":
		outputData, err = exportToGitHub(filteredIssues)
	case "excel":
		outputData, err = exportToExcel(filteredIssues)
	case "markdown", "md":
		outputData, err = exportToMarkdown(filteredIssues)
	default:
		return fmt.Errorf("unsupported format: %s", exportFormat)
	}

	if err != nil {
		return fmt.Errorf("failed to generate export: %w", err)
	}

	// Write output
	if exportOutput != "" {
		if err := os.WriteFile(exportOutput, outputData, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("✅ Exported %d issues to: %s\n", len(filteredIssues), exportOutput)
		fmt.Printf("📊 Format: %s\n", exportFormat)
		fmt.Printf("💾 File size: %d bytes\n", len(outputData))
	} else {
		fmt.Print(string(outputData))
	}

	return nil
}

func applyExportFilters(issues []*github.Issue) []*github.Issue {
	filtered := make([]*github.Issue, 0)

	for _, issue := range issues {
		// Skip pull requests
		if issue.PullRequest != nil {
			continue
		}

		// Filter by closed status
		if !exportIncludeClosed && issue.State == "closed" {
			continue
		}

		// Filter by date range
		if exportDateFrom != "" {
			fromDate, err := time.Parse("2006-01-02", exportDateFrom)
			if err == nil && issue.CreatedAt.Before(fromDate) {
				continue
			}
		}

		if exportDateTo != "" {
			toDate, err := time.Parse("2006-01-02", exportDateTo)
			if err == nil && issue.CreatedAt.After(toDate) {
				continue
			}
		}

		// Filter by labels
		if exportFilter != "" {
			hasLabel := false
			for _, label := range issue.Labels {
				if strings.Contains(strings.ToLower(label.Name), strings.ToLower(exportFilter)) {
					hasLabel = true
					break
				}
			}
			if !hasLabel {
				continue
			}
		}

		filtered = append(filtered, issue)
	}

	return filtered
}

func exportToCSV(issues []*github.Issue) ([]byte, error) {
	var buf strings.Builder
	w := csv.NewWriter(&buf)

	// Header
	if err := w.Write([]string{
		"Issue Number",
		"Title",
		"State",
		"Created At",
		"Closed At",
		"Labels",
		"Assignees",
		"Comments",
		"URL",
	}); err != nil {
		return nil, err
	}

	// Data rows
	for _, issue := range issues {
		closedAt := ""
		if issue.ClosedAt != nil {
			closedAt = issue.ClosedAt.Format("2006-01-02")
		}

		labels := make([]string, len(issue.Labels))
		for i, label := range issue.Labels {
			labels[i] = label.Name
		}

		assignees := make([]string, len(issue.Assignees))
		for i, assignee := range issue.Assignees {
			assignees[i] = assignee.Login
		}

		if err := w.Write([]string{
			fmt.Sprintf("%d", issue.Number),
			issue.Title,
			issue.State,
			issue.CreatedAt.Format("2006-01-02"),
			closedAt,
			strings.Join(labels, "; "),
			strings.Join(assignees, "; "),
			fmt.Sprintf("%d", issue.Comments),
			issue.URL,
		}); err != nil {
			return nil, err
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}

	return []byte(buf.String()), nil
}

func exportToJSON(issues []*github.Issue) ([]byte, error) {
	output := map[string]interface{}{
		"exported_at": time.Now().Format(time.RFC3339),
		"count":       len(issues),
		"issues":      issues,
	}

	return json.MarshalIndent(output, "", "  ")
}

func exportToJira(issues []*github.Issue) ([]byte, error) {
	var buf strings.Builder
	w := csv.NewWriter(&buf)

	// Jira import format
	if err := w.Write([]string{
		"Issue Type",
		"Summary",
		"Description",
		"Priority",
		"Labels",
		"Status",
		"Created",
	}); err != nil {
		return nil, err
	}

	for _, issue := range issues {
		issueType := "Story"
		if containsLabel(issue.Labels, "bug") {
			issueType = "Bug"
		} else if containsLabel(issue.Labels, "epic") {
			issueType = "Epic"
		}

		priority := "Medium"
		if containsLabel(issue.Labels, "high") || containsLabel(issue.Labels, "critical") {
			priority = "High"
		} else if containsLabel(issue.Labels, "low") {
			priority = "Low"
		}

		status := "To Do"
		if issue.State == "closed" {
			status = "Done"
		}

		labels := make([]string, len(issue.Labels))
		for i, label := range issue.Labels {
			labels[i] = label.Name
		}

		if err := w.Write([]string{
			issueType,
			issue.Title,
			issue.Body,
			priority,
			strings.Join(labels, " "),
			status,
			issue.CreatedAt.Format("2006-01-02"),
		}); err != nil {
			return nil, err
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}

	return []byte(buf.String()), nil
}

func exportToGitHub(issues []*github.Issue) ([]byte, error) {
	var buf strings.Builder
	w := csv.NewWriter(&buf)

	// GitHub Projects CSV import format
	if err := w.Write([]string{
		"Title",
		"Description",
		"Status",
		"Priority",
		"Labels",
		"Assignees",
		"Issue Number",
	}); err != nil {
		return nil, err
	}

	for _, issue := range issues {
		status := "Todo"
		if issue.State == "closed" {
			status = "Done"
		}

		priority := "Medium"
		if containsLabel(issue.Labels, "high") || containsLabel(issue.Labels, "critical") {
			priority = "High"
		} else if containsLabel(issue.Labels, "low") {
			priority = "Low"
		}

		labels := make([]string, len(issue.Labels))
		for i, label := range issue.Labels {
			labels[i] = label.Name
		}

		assignees := make([]string, len(issue.Assignees))
		for i, assignee := range issue.Assignees {
			assignees[i] = assignee.Login
		}

		if err := w.Write([]string{
			issue.Title,
			issue.Body,
			status,
			priority,
			strings.Join(labels, ", "),
			strings.Join(assignees, ", "),
			fmt.Sprintf("#%d", issue.Number),
		}); err != nil {
			return nil, err
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}

	return []byte(buf.String()), nil
}

func exportToExcel(issues []*github.Issue) ([]byte, error) {
	// Excel-compatible CSV with rich formatting hints
	var buf strings.Builder
	w := csv.NewWriter(&buf)

	// Header with Excel-friendly formatting
	if err := w.Write([]string{
		"ID",
		"Title",
		"Description",
		"Status",
		"Priority",
		"Created Date",
		"Closed Date",
		"Days Open",
		"Assignees",
		"Labels",
		"Comments",
		"URL",
	}); err != nil {
		return nil, err
	}

	for _, issue := range issues {
		closedAt := ""
		daysOpen := ""
		if issue.ClosedAt != nil {
			closedAt = issue.ClosedAt.Format("2006-01-02")
			days := int(issue.ClosedAt.Sub(issue.CreatedAt).Hours() / 24)
			daysOpen = fmt.Sprintf("%d", days)
		}

		labels := make([]string, len(issue.Labels))
		for i, label := range issue.Labels {
			labels[i] = label.Name
		}

		assignees := make([]string, len(issue.Assignees))
		for i, assignee := range issue.Assignees {
			assignees[i] = assignee.Login
		}

		priority := "Medium"
		if containsLabel(issue.Labels, "high") || containsLabel(issue.Labels, "critical") {
			priority = "High"
		} else if containsLabel(issue.Labels, "low") {
			priority = "Low"
		}

		if err := w.Write([]string{
			fmt.Sprintf("#%d", issue.Number),
			issue.Title,
			issue.Body,
			issue.State,
			priority,
			issue.CreatedAt.Format("2006-01-02"),
			closedAt,
			daysOpen,
			strings.Join(assignees, "; "),
			strings.Join(labels, "; "),
			fmt.Sprintf("%d", issue.Comments),
			issue.URL,
		}); err != nil {
			return nil, err
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}

	return []byte(buf.String()), nil
}

func exportToMarkdown(issues []*github.Issue) ([]byte, error) {
	var md strings.Builder

	md.WriteString("# Issue Export\n\n")
	md.WriteString(fmt.Sprintf("**Exported:** %s\n", time.Now().Format("2006-01-02 15:04:05")))
	md.WriteString(fmt.Sprintf("**Total Issues:** %d\n\n", len(issues)))

	// Summary statistics
	openCount := 0
	closedCount := 0

	for _, issue := range issues {
		if issue.State == "closed" {
			closedCount++
		} else {
			openCount++
		}
	}

	md.WriteString("## Summary\n\n")
	md.WriteString(fmt.Sprintf("- **Open Issues:** %d\n", openCount))
	md.WriteString(fmt.Sprintf("- **Closed Issues:** %d\n\n", closedCount))

	// Issue table
	md.WriteString("## Issues\n\n")
	md.WriteString("| ID | Title | Status | Labels | Comments |\n")
	md.WriteString("|----|-------|--------|--------|----------|\n")

	for _, issue := range issues {
		title := issue.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}

		labels := make([]string, len(issue.Labels))
		for i, label := range issue.Labels {
			labels[i] = label.Name
		}
		labelsStr := strings.Join(labels, ", ")
		if len(labelsStr) > 30 {
			labelsStr = labelsStr[:27] + "..."
		}

		md.WriteString(fmt.Sprintf("| #%d | %s | %s | %s | %d |\n",
			issue.Number,
			title,
			issue.State,
			labelsStr,
			issue.Comments,
		))
	}

	md.WriteString("\n---\n")
	md.WriteString("*Generated by SET CLI Export*\n")

	return []byte(md.String()), nil
}

// Helper functions

func containsLabel(labels []github.Label, search string) bool {
	searchLower := strings.ToLower(search)
	for _, label := range labels {
		if strings.Contains(strings.ToLower(label.Name), searchLower) {
			return true
		}
	}
	return false
}
