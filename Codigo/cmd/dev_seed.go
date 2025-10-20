package cmd

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"set/internal/config"
	"set/internal/devdata"
	"set/internal/github"
	"set/internal/logger"
	"set/internal/storage"

	"github.com/spf13/cobra"
)

var (
	seedCount      int
	seedWithCF     bool
	seedClearFirst bool
	seedDataFile   string
)

// devSeedCmd seeds the database with fake estimation data
var devSeedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seed database with fake estimation data",
	Long: `Seed the local database with realistic fake estimation data.

This command generates realistic software development tasks with:
- Task titles and descriptions
- Estimated and actual hours
- Story points and sizes
- Custom fields (Worker Hours, Story Points, Size)
- Labels (feature, bug, enhancement, etc.)

Data is based on real software estimation datasets to provide
realistic estimation scenarios for testing.

Examples:
  # Seed with 50 tasks
  set dev seed --count 50

  # Seed with custom fields
  set dev seed --count 100 --with-custom-fields

  # Clear existing data first
  set dev seed --count 30 --clear`,
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
	RunE: runDevSeed,
}

func init() {
	devSeedCmd.Flags().IntVarP(&seedCount, "count", "n", 50, "Number of fake issues to create (max: available in JSON)")
	devSeedCmd.Flags().BoolVar(&seedWithCF, "with-custom-fields", true, "Include custom fields")
	devSeedCmd.Flags().BoolVar(&seedClearFirst, "clear", false, "Clear existing data first")
	devSeedCmd.Flags().StringVarP(&seedDataFile, "data-file", "f", "seed-data.json", "Path to seed data JSON file")
}

func runDevSeed(cmd *cobra.Command, args []string) error {
	// Load seed data from JSON
	fmt.Printf("📂 Loading seed data from: %s\n", seedDataFile)
	seedData, err := devdata.LoadSeedData(seedDataFile)
	if err != nil {
		return fmt.Errorf("failed to load seed data: %w", err)
	}

	fmt.Printf("✅ Loaded %d tasks from %s\n", len(seedData.Tasks), seedData.Metadata.Source)
	fmt.Println()

	// Validate count
	maxCount := len(seedData.Tasks)
	if seedCount < 1 {
		return fmt.Errorf("count must be at least 1")
	}
	if seedCount > maxCount {
		fmt.Printf("⚠️  Requested %d tasks but only %d available. Using %d.\n", seedCount, maxCount, maxCount)
		seedCount = maxCount
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

	// Clear if requested
	if seedClearFirst {
		fmt.Println("🗑️  Clearing existing data...")
		if err := store.Clear(); err != nil {
			return fmt.Errorf("failed to clear storage: %w", err)
		}
		fmt.Println("✅ Data cleared")
		fmt.Println()
	}

	fmt.Printf("🌱 Seeding database with %d issues...\n", seedCount)
	fmt.Println()

	// Generate issues from seed data
	issues := generateIssuesFromSeedData(seedData.Tasks[:seedCount], seedWithCF)

	// Save to storage
	logger.Infof("Saving %d issues to storage", len(issues))

	if err := store.SaveIssues(issues); err != nil {
		return fmt.Errorf("failed to save issues: %w", err)
	}

	// Update repository metadata
	repo := &github.Repository{
		ID:          12345,
		Name:        "estimation-test-project",
		FullName:    "test-org/estimation-test-project",
		Description: fmt.Sprintf("Test project with data from %s", seedData.Metadata.Source),
		Private:     false,
		CreatedAt:   time.Now().Add(-365 * 24 * time.Hour),
		UpdatedAt:   time.Now(),
	}

	if err := store.SetRepository(repo); err != nil {
		return fmt.Errorf("failed to save repository metadata: %w", err)
	}

	// Update last sync time
	ctx := context.Background()
	_ = ctx // Suppress unused warning
	if err := store.SetLastSync(time.Now()); err != nil {
		logger.Warnf("Failed to update last sync time: %v", err)
	}

	fmt.Println("✅ Seeding complete!")
	fmt.Println()
	fmt.Println("📊 Summary:")
	fmt.Printf("   Data Source:      %s\n", seedData.Metadata.Source)
	fmt.Printf("   Total Issues:     %d\n", len(issues))

	// Count by category
	categoryCount := make(map[string]int)
	for _, issue := range issues {
		for _, label := range issue.Labels {
			categoryCount[label.Name]++
		}
	}

	if seedWithCF {
		withCF := 0
		for _, issue := range issues {
			if len(issue.CustomFields) > 0 {
				withCF++
			}
		}
		fmt.Printf("   With Custom Fields: %d\n", withCF)
	}

	// Count by state
	open := 0
	closed := 0
	for _, issue := range issues {
		if issue.State == "open" {
			open++
		} else {
			closed++
		}
	}
	fmt.Printf("   Open:             %d\n", open)
	fmt.Printf("   Closed:           %d\n", closed)

	// Show top categories
	fmt.Println()
	fmt.Println("📑 Top Categories:")
	count := 0
	for cat, num := range categoryCount {
		if count >= 5 {
			break
		}
		fmt.Printf("   %-20s %d\n", cat, num)
		count++
	}

	fmt.Println()
	fmt.Println("💡 Try:")
	fmt.Println("   set inspect")
	fmt.Println("   set estimate \"Add user authentication\"")

	return nil
}

// generateIssuesFromSeedData converts seed data tasks to GitHub issues
func generateIssuesFromSeedData(tasks []devdata.SeedTask, withCustomFields bool) []*github.Issue {
	rand.Seed(time.Now().UnixNano())

	issues := make([]*github.Issue, len(tasks))

	for i, task := range tasks {
		// Calculate timestamps
		// Spread issues over the last 180 days
		createdAt := time.Now().Add(-time.Duration(rand.Intn(180)) * 24 * time.Hour)

		var closedAt *time.Time
		if task.State == "closed" {
			// Use actual hours to calculate when it was closed
			closed := createdAt.Add(time.Duration(task.ActualHours*60) * time.Minute)
			closedAt = &closed
		}

		// Create the issue
		issue := &github.Issue{
			ID:     int64(1000 + i),
			Number: 1000 + i,
			Title:  task.Title,
			Body:   task.Description,
			State:  task.State,
			Labels: make([]github.Label, len(task.Labels)),
			User: github.User{
				Login: fmt.Sprintf("user%d", rand.Intn(10)),
			},
			CreatedAt: createdAt,
			UpdatedAt: createdAt.Add(time.Duration(rand.Intn(24)) * time.Hour),
			ClosedAt:  closedAt,
		}

		// Add labels
		for j, labelName := range task.Labels {
			issue.Labels[j] = github.Label{
				Name:  labelName,
				Color: getLabelColor(labelName),
			}
		}

		// Add custom fields if requested and task is closed
		if withCustomFields && task.State == "closed" {
			issue.CustomFields = map[string]interface{}{
				"Worker Hours": task.ActualHours,
				"Story Points": float64(task.StoryPoints),
				"Size":         task.Size,
			}

			// Add any additional custom fields from the seed data
			for key, value := range task.CustomFields {
				issue.CustomFields[key] = value
			}
		}

		issues[i] = issue
	}

	return issues
}

// getLabelColor returns a color for a label
func getLabelColor(label string) string {
	colors := map[string]string{
		"feature":        "0052CC",
		"bug":            "D73A4A",
		"enhancement":    "A2EEEF",
		"documentation":  "0075CA",
		"testing":        "FBCA04",
		"security":       "D93F0B",
		"performance":    "5319E7",
		"frontend":       "C2E0C6",
		"backend":        "BFD4F2",
		"infrastructure": "FEF2C0",
		"devops":         "FFA500",
		"critical":       "B60205",
		"refactor":       "EDEDED",
	}

	if color, ok := colors[label]; ok {
		return color
	}
	return "EDEDED"
}
