package devdata

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// LoadSeedData loads seed data from a JSON file
func LoadSeedData(filePath string) (*SeedData, error) {
	// If path is not absolute, try to find it relative to executable
	if !filepath.IsAbs(filePath) {
		// Try current directory first
		if _, err := os.Stat(filePath); err == nil {
			// File exists in current directory
		} else {
			// Try relative to executable
			execPath, err := os.Executable()
			if err == nil {
				execDir := filepath.Dir(execPath)
				testPath := filepath.Join(execDir, filePath)
				if _, err := os.Stat(testPath); err == nil {
					filePath = testPath
				}
			}
		}
	}

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read seed data file: %w", err)
	}

	// Parse JSON
	var seedData SeedData
	if err := json.Unmarshal(data, &seedData); err != nil {
		return nil, fmt.Errorf("failed to parse seed data JSON: %w", err)
	}

	// Validate
	if len(seedData.Tasks) == 0 {
		return nil, fmt.Errorf("seed data contains no tasks")
	}

	return &seedData, nil
}

// GetDefaultSeedDataPath returns the default path for seed data
func GetDefaultSeedDataPath() string {
	// Try to find seed-data.json in common locations
	locations := []string{
		"seed-data.json",
		"data/seed-data.json",
		"internal/devdata/seed-data.json",
		"../seed-data.json",
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}

	return "seed-data.json"
}
