package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"set/internal/config"
	"set/internal/github"
	"set/internal/storage"

	"github.com/spf13/cobra"
)

var (
	inspectIssue int
	inspectPR    int
	listAll      bool
	outputJSON   bool
	showCustom   bool
	limitResults int
)

// inspectCmd allows viewing locally stored data
var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect locally stored GitHub data",
	Long: `Inspect and view locally stored GitHub issues and pull requests.

This command allows you to view data that has been synced from GitHub,
including custom fields from GitHub Projects V2.

Examples:
  # View a specific issue
  set inspect --issue 123

  # View a specific pull request
  set inspect --pr 45

  # List all stored issues (first 20)
  set inspect --list

  # List all issues with custom fields only
  set inspect --list --custom

  # Export issue to JSON
  set inspect --issue 123 --json

  # List up to 50 issues
  set inspect --list --limit 50`,
	RunE: runInspect,
}

func init() {
	rootCmd.AddCommand(inspectCmd)

	inspectCmd.Flags().IntVar(&inspectIssue, "issue", 0, "Issue number to inspect")
	inspectCmd.Flags().IntVar(&inspectPR, "pr", 0, "Pull request number to inspect")
	inspectCmd.Flags().BoolVar(&listAll, "list", false, "List all stored issues")
	inspectCmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")
	inspectCmd.Flags().BoolVar(&showCustom, "custom", false, "Show only items with custom fields")
	inspectCmd.Flags().IntVar(&limitResults, "limit", 20, "Maximum number of results to show when listing")
}

func runInspect(cmd *cobra.Command, args []string) error {
	// Determine storage path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	storePath := filepath.Join(homeDir, ".set", "data.db")

	// Check if database exists
	if _, err := os.Stat(storePath); os.IsNotExist(err) {
		return fmt.Errorf("no local data found. Run 'set sync' first to download data")
	}

	// Open storage
	store, err := storage.NewStore(storePath)
	if err != nil {
		return fmt.Errorf("failed to open storage: %w", err)
	}
	defer store.Close()

	// Handle different inspection modes
	if inspectIssue > 0 {
		return inspectSingleIssue(store, inspectIssue)
	}

	if inspectPR > 0 {
		return inspectSinglePR(store, inspectPR)
	}

	if listAll {
		return listAllIssues(store)
	}

	// Default: show statistics
	return showStatistics(store)
}

func inspectSingleIssue(store *storage.Store, number int) error {
	issue, err := store.GetIssue(number)
	if err != nil {
		return fmt.Errorf("issue #%d not found in local storage", number)
	}

	if outputJSON {
		return printJSON(issue)
	}

	return printIssueDetailed(issue)
}

func inspectSinglePR(store *storage.Store, number int) error {
	pr, err := store.GetPullRequest(number)
	if err != nil {
		return fmt.Errorf("pull request #%d not found in local storage", number)
	}

	if outputJSON {
		return printJSON(pr)
	}

	return printPRDetailed(pr)
}

func listAllIssues(store *storage.Store) error {
	issues, err := store.GetAllIssues()
	if err != nil {
		return fmt.Errorf("failed to retrieve issues: %w", err)
	}

	if len(issues) == 0 {
		fmt.Println("No issues found in local storage. Run 'set sync' to download data.")
		return nil
	}

	// Filter by custom fields if requested
	var filteredIssues []*github.Issue
	if showCustom {
		for _, issue := range issues {
			if len(issue.CustomFields) > 0 {
				filteredIssues = append(filteredIssues, issue)
			}
		}
		issues = filteredIssues
	}

	if outputJSON {
		// Limit results
		if limitResults > 0 && len(issues) > limitResults {
			issues = issues[:limitResults]
		}
		return printJSON(issues)
	}

	return printIssueTable(issues)
}

func showStatistics(store *storage.Store) error {
	// Load configuration to show repo info
	cfg, _ := config.Load()

	// Get counts
	issueCount, err := store.CountIssues()
	if err != nil {
		return fmt.Errorf("failed to count issues: %w", err)
	}

	prCount, err := store.CountPullRequests()
	if err != nil {
		return fmt.Errorf("failed to count pull requests: %w", err)
	}

	// Get last sync time
	lastSync, err := store.GetLastSync()
	if err != nil {
		lastSync = nil
	}

	// Get repository info
	repo, err := store.GetRepository()
	if err != nil {
		repo = nil
	}

	// Print statistics
	fmt.Println("╭─ Local Storage ───────────────────────────────────────────╮")
	fmt.Println("│                                                           │")

	if repo != nil {
		fmt.Printf("│  Repository   %-44s│\n", repo.FullName)
		if repo.Description != "" {
			// Truncate description if too long
			desc := repo.Description
			if len(desc) > 44 {
				desc = desc[:41] + "..."
			}
			fmt.Printf("│  Description  %-44s│\n", desc)
		}
	} else if cfg != nil && cfg.GitHub.DefaultRepo != "" {
		fmt.Printf("│  Repository   %-44s│\n", cfg.GitHub.DefaultRepo)
	}

	fmt.Println("│                                                           │")
	fmt.Printf("│  Issues           %-40d│\n", issueCount)
	fmt.Printf("│  Pull Requests    %-40d│\n", prCount)
	fmt.Printf("│  Total Items      %-40d│\n", issueCount+prCount)
	fmt.Println("│                                                           │")

	if lastSync != nil {
		fmt.Printf("│  Last Synced      %-40s│\n", lastSync.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("│  Last Synced      %-40s│\n", "Never")
	}

	// Check for custom fields
	issues, err := store.GetAllIssues()
	withCustomFields := 0
	if err == nil {
		for _, issue := range issues {
			if len(issue.CustomFields) > 0 {
				withCustomFields++
			}
		}
	}

	if withCustomFields > 0 {
		fmt.Println("│                                                           │")
		fmt.Printf("│  ✨ Custom Fields  %-40d│\n", withCustomFields)
	}

	fmt.Println("│                                                           │")
	fmt.Println("╰───────────────────────────────────────────────────────────╯")
	fmt.Println()

	fmt.Println("┃ Available Commands")
	fmt.Println("  · set inspect --list              List all issues")
	fmt.Println("  · set inspect --list --custom     Issues with custom fields")
	fmt.Println("  · set inspect --issue <number>    View specific issue")
	fmt.Println("  · set inspect --pr <number>       View specific PR")
	fmt.Println("  · set inspect --list --json       Export as JSON")
	fmt.Println()

	fmt.Println(strings.Repeat("─", 60))

	return nil
}

func printIssueDetailed(issue *github.Issue) error {
	fmt.Printf("🔍 Issue #%d\n", issue.Number)
	fmt.Println(strings.Repeat("─", 80))

	fmt.Printf("Title:       %s\n", issue.Title)
	fmt.Printf("State:       %s\n", issue.State)
	fmt.Printf("Author:      %s\n", issue.User.Login)
	fmt.Printf("Created:     %s\n", issue.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:     %s\n", issue.UpdatedAt.Format("2006-01-02 15:04:05"))

	if issue.ClosedAt != nil {
		fmt.Printf("Closed:      %s\n", issue.ClosedAt.Format("2006-01-02 15:04:05"))
	}

	fmt.Printf("Comments:    %d\n", issue.Comments)
	fmt.Printf("URL:         %s\n", issue.URL)

	// Labels
	if len(issue.Labels) > 0 {
		fmt.Printf("\nLabels:\n")
		for _, label := range issue.Labels {
			fmt.Printf("   - %s\n", label.Name)
		}
	}

	// Assignees
	if len(issue.Assignees) > 0 {
		fmt.Printf("\nAssignees:\n")
		for _, assignee := range issue.Assignees {
			fmt.Printf("   - %s\n", assignee.Login)
		}
	}

	// Milestone
	if issue.Milestone != nil {
		fmt.Printf("\nMilestone:   %s\n", issue.Milestone.Title)
	}

	// Custom Fields
	if len(issue.CustomFields) > 0 {
		fmt.Printf("\n✨ Custom Fields (GitHub Projects):\n")
		for key, value := range issue.CustomFields {
			fmt.Printf("   %s: %v\n", key, value)
		}
	}

	// Body
	if issue.Body != "" {
		fmt.Printf("\nDescription:\n")
		fmt.Println(strings.Repeat("─", 80))
		// Truncate long bodies
		body := issue.Body
		if len(body) > 500 {
			body = body[:500] + "\n... (truncated)"
		}
		fmt.Println(body)
	}

	fmt.Println(strings.Repeat("─", 80))

	return nil
}

func printPRDetailed(pr *github.PullRequest) error {
	fmt.Printf("🔀 Pull Request #%d\n", pr.Number)
	fmt.Println(strings.Repeat("─", 80))

	fmt.Printf("Title:       %s\n", pr.Title)
	fmt.Printf("State:       %s\n", pr.State)
	fmt.Printf("Author:      %s\n", pr.User.Login)
	fmt.Printf("Created:     %s\n", pr.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:     %s\n", pr.UpdatedAt.Format("2006-01-02 15:04:05"))

	if pr.MergedAt != nil {
		fmt.Printf("Merged:      %s\n", pr.MergedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Merged:      ✅ Yes\n")
	} else if pr.ClosedAt != nil {
		fmt.Printf("Closed:      %s\n", pr.ClosedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Merged:      ❌ No\n")
	} else {
		fmt.Printf("Merged:      ⏳ Open\n")
	}

	fmt.Printf("Comments:    %d\n", pr.Comments)
	fmt.Printf("Commits:     %d\n", pr.Commits)
	fmt.Printf("Changed:     +%d / -%d lines in %d files\n", pr.Additions, pr.Deletions, pr.ChangedFiles)
	fmt.Printf("URL:         %s\n", pr.URL)

	// Labels
	if len(pr.Labels) > 0 {
		fmt.Printf("\nLabels:\n")
		for _, label := range pr.Labels {
			fmt.Printf("   - %s\n", label.Name)
		}
	}

	// Assignees
	if len(pr.Assignees) > 0 {
		fmt.Printf("\nAssignees:\n")
		for _, assignee := range pr.Assignees {
			fmt.Printf("   - %s\n", assignee.Login)
		}
	}

	// Custom Fields
	if len(pr.CustomFields) > 0 {
		fmt.Printf("\n✨ Custom Fields (GitHub Projects):\n")
		for key, value := range pr.CustomFields {
			fmt.Printf("   %s: %v\n", key, value)
		}
	}

	// Body
	if pr.Body != "" {
		fmt.Printf("\nDescription:\n")
		fmt.Println(strings.Repeat("─", 80))
		// Truncate long bodies
		body := pr.Body
		if len(body) > 500 {
			body = body[:500] + "\n... (truncated)"
		}
		fmt.Println(body)
	}

	fmt.Println(strings.Repeat("─", 80))

	return nil
}

func printIssueTable(issues []*github.Issue) error {
	// Limit results
	if limitResults > 0 && len(issues) > limitResults {
		fmt.Printf("Showing first %d of %d issues\n\n", limitResults, len(issues))
		issues = issues[:limitResults]
	} else {
		fmt.Printf("Showing %d issues\n\n", len(issues))
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "#\tState\tTitle\tAuthor\tLabels\tCustom Fields")
	fmt.Fprintln(w, strings.Repeat("─", 10)+"\t"+strings.Repeat("─", 8)+"\t"+strings.Repeat("─", 40)+"\t"+strings.Repeat("─", 15)+"\t"+strings.Repeat("─", 20)+"\t"+strings.Repeat("─", 20))

	for _, issue := range issues {
		// Truncate title
		title := issue.Title
		if len(title) > 40 {
			title = title[:37] + "..."
		}

		// Format labels
		labels := ""
		if len(issue.Labels) > 0 {
			if len(issue.Labels) == 1 {
				labels = issue.Labels[0].Name
			} else {
				labels = fmt.Sprintf("%s +%d", issue.Labels[0].Name, len(issue.Labels)-1)
			}
		}

		// Format custom fields
		customFields := ""
		if len(issue.CustomFields) > 0 {
			customFields = fmt.Sprintf("✨ %d fields", len(issue.CustomFields))
		}

		fmt.Fprintf(w, "#%d\t%s\t%s\t%s\t%s\t%s\n",
			issue.Number,
			issue.State,
			title,
			issue.User.Login,
			labels,
			customFields,
		)
	}

	w.Flush()

	fmt.Println()
	fmt.Println("┃ Next Steps")
	fmt.Println("  · Use 'set inspect --issue <number>' to view details")
	if !showCustom {
		fmt.Println("  · Use 'set inspect --list --custom' for items with custom fields")
	}
	fmt.Println()

	return nil
}

func printJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}
