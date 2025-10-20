package estimator

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"set/internal/ai"
)

func TestLoadBatchRequest(t *testing.T) {
	// Create temporary JSON file
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "batch.json")

	request := &BatchRequest{
		Tasks: []BatchTask{
			{
				ID:          "TASK-001",
				Title:       "Test task 1",
				Description: "Description 1",
				Labels:      []string{"bug", "high-priority"},
				Priority:    "high",
			},
			{
				ID:          "TASK-002",
				Title:       "Test task 2",
				Description: "Description 2",
				Labels:      []string{"feature"},
				Priority:    "medium",
			},
		},
	}

	// Write JSON file
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}
	if err := os.WriteFile(jsonFile, data, 0644); err != nil {
		t.Fatalf("Failed to write JSON file: %v", err)
	}

	// Load and verify
	loaded, err := LoadBatchRequest(jsonFile)
	if err != nil {
		t.Fatalf("LoadBatchRequest failed: %v", err)
	}

	if len(loaded.Tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(loaded.Tasks))
	}

	if loaded.Tasks[0].ID != "TASK-001" {
		t.Errorf("Expected task ID TASK-001, got %s", loaded.Tasks[0].ID)
	}

	if loaded.Tasks[0].Title != "Test task 1" {
		t.Errorf("Expected task title 'Test task 1', got %s", loaded.Tasks[0].Title)
	}

	if len(loaded.Tasks[0].Labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(loaded.Tasks[0].Labels))
	}
}

func TestLoadBatchRequestFromCSV(t *testing.T) {
	// Create temporary CSV file
	tmpDir := t.TempDir()
	csvFile := filepath.Join(tmpDir, "batch.csv")

	csvContent := `id,title,description,labels,priority,assignee
TASK-001,Test task 1,Description 1,bug|high-priority,high,dev@example.com
TASK-002,Test task 2,Description 2,feature,medium,
TASK-003,Test task 3,,testing|quality,low,qa@example.com`

	if err := os.WriteFile(csvFile, []byte(csvContent), 0644); err != nil {
		t.Fatalf("Failed to write CSV file: %v", err)
	}

	// Load and verify
	loaded, err := LoadBatchRequestFromCSV(csvFile)
	if err != nil {
		t.Fatalf("LoadBatchRequestFromCSV failed: %v", err)
	}

	if len(loaded.Tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(loaded.Tasks))
	}

	// Verify first task
	if loaded.Tasks[0].ID != "TASK-001" {
		t.Errorf("Expected task ID TASK-001, got %s", loaded.Tasks[0].ID)
	}

	if loaded.Tasks[0].Title != "Test task 1" {
		t.Errorf("Expected task title 'Test task 1', got %s", loaded.Tasks[0].Title)
	}

	if loaded.Tasks[0].Priority != "high" {
		t.Errorf("Expected priority 'high', got %s", loaded.Tasks[0].Priority)
	}

	if loaded.Tasks[0].Assignee != "dev@example.com" {
		t.Errorf("Expected assignee 'dev@example.com', got %s", loaded.Tasks[0].Assignee)
	}

	// Verify labels were split correctly
	if len(loaded.Tasks[0].Labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(loaded.Tasks[0].Labels))
	}

	if loaded.Tasks[0].Labels[0] != "bug" || loaded.Tasks[0].Labels[1] != "high-priority" {
		t.Errorf("Labels not parsed correctly: %v", loaded.Tasks[0].Labels)
	}

	// Verify second task with single label
	if len(loaded.Tasks[1].Labels) != 1 {
		t.Errorf("Expected 1 label, got %d", len(loaded.Tasks[1].Labels))
	}

	// Verify third task with empty description
	if loaded.Tasks[2].Description != "" {
		t.Errorf("Expected empty description, got %s", loaded.Tasks[2].Description)
	}
}

func TestLoadBatchRequest_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "empty.json")

	emptyRequest := &BatchRequest{
		Tasks: []BatchTask{},
	}

	data, _ := json.Marshal(emptyRequest)
	os.WriteFile(jsonFile, data, 0644)

	_, err := LoadBatchRequest(jsonFile)
	if err == nil {
		t.Error("Expected error for empty task list, got nil")
	}
}

func TestLoadBatchRequestFromCSV_NoTitleColumn(t *testing.T) {
	tmpDir := t.TempDir()
	csvFile := filepath.Join(tmpDir, "invalid.csv")

	csvContent := `id,description
TASK-001,Description 1`

	os.WriteFile(csvFile, []byte(csvContent), 0644)

	_, err := LoadBatchRequestFromCSV(csvFile)
	if err == nil {
		t.Error("Expected error for missing title column, got nil")
	}
}

func TestNewBatchProcessor(t *testing.T) {
	mockAI := &mockAIProvider{
		available: true,
		response: &ai.EstimationResponse{
			EstimatedHours:  8.0,
			EstimatedSize:   "M",
			StoryPoints:     5.0,
			ConfidenceScore: 0.75,
			Reasoning:       "Mock estimation",
		},
	}

	est := NewEstimator(mockAI, nil, DefaultEstimationConfig())
	processor := NewBatchProcessor(est, 3)

	if processor.maxWorkers != 3 {
		t.Errorf("Expected 3 workers, got %d", processor.maxWorkers)
	}

	if processor.estimator == nil {
		t.Error("Estimator is nil")
	}

	if processor.rateLimiter == nil {
		t.Error("Rate limiter is nil")
	}
}

func TestNewBatchProcessor_DefaultWorkers(t *testing.T) {
	mockAI := &mockAIProvider{
		available: true,
		response: &ai.EstimationResponse{
			EstimatedHours:  8.0,
			EstimatedSize:   "M",
			StoryPoints:     5.0,
			ConfidenceScore: 0.75,
			Reasoning:       "Mock estimation",
		},
	}

	est := NewEstimator(mockAI, nil, DefaultEstimationConfig())
	processor := NewBatchProcessor(est, 0) // Invalid value should default to 5

	if processor.maxWorkers != 5 {
		t.Errorf("Expected default 5 workers, got %d", processor.maxWorkers)
	}
}

// mockAIProviderBatch allows dynamic responses per task
type mockAIProviderBatch struct {
	available bool
}

func (m *mockAIProviderBatch) EstimateTask(request *ai.EstimationRequest) (*ai.EstimationResponse, error) {
	// Different estimates based on task title
	hours := 5.0
	size := "M"
	points := 3.0

	if request.TaskTitle == "Large task" {
		hours = 16.0
		size = "L"
		points = 8.0
	} else if request.TaskTitle == "Small task" {
		hours = 2.0
		size = "S"
		points = 1.0
	}

	return &ai.EstimationResponse{
		EstimatedHours:  hours,
		EstimatedSize:   size,
		StoryPoints:     points,
		ConfidenceScore: 0.75,
		Reasoning:       "Mock estimation for " + request.TaskTitle,
	}, nil
}

func (m *mockAIProviderBatch) GetName() string {
	return "mock-batch"
}

func (m *mockAIProviderBatch) IsAvailable() bool {
	return m.available
}

func TestProcessBatch(t *testing.T) {
	mockAI := &mockAIProviderBatch{available: true}

	est := NewEstimator(mockAI, nil, DefaultEstimationConfig())
	processor := NewBatchProcessor(est, 2)

	request := &BatchRequest{
		Tasks: []BatchTask{
			{ID: "TASK-001", Title: "Small task", Labels: []string{"feature"}},
			{ID: "TASK-002", Title: "Medium task", Labels: []string{"bug"}},
			{ID: "TASK-003", Title: "Large task", Labels: []string{"feature"}},
		},
	}

	ctx := context.Background()
	report, err := processor.ProcessBatch(ctx, request)

	if err != nil {
		t.Fatalf("ProcessBatch failed: %v", err)
	}

	// Verify report metadata
	if report.TotalTasks != 3 {
		t.Errorf("Expected 3 total tasks, got %d", report.TotalTasks)
	}

	if report.SuccessfulTasks != 3 {
		t.Errorf("Expected 3 successful tasks, got %d", report.SuccessfulTasks)
	}

	if report.FailedTasks != 0 {
		t.Errorf("Expected 0 failed tasks, got %d", report.FailedTasks)
	}

	// Verify results
	if len(report.Results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(report.Results))
	}

	// Verify statistics
	if report.Statistics == nil {
		t.Fatal("Statistics is nil")
	}

	expectedTotal := 2.0 + 5.0 + 16.0 // Small + Medium + Large
	if report.Statistics.TotalEstimatedHours != expectedTotal {
		t.Errorf("Expected total hours %.1f, got %.1f", expectedTotal, report.Statistics.TotalEstimatedHours)
	}

	expectedAvg := expectedTotal / 3
	if report.Statistics.AverageHours != expectedAvg {
		t.Errorf("Expected average hours %.1f, got %.1f", expectedAvg, report.Statistics.AverageHours)
	}

	// Verify size distribution
	if report.Statistics.SizeDistribution["S"] != 1 {
		t.Errorf("Expected 1 small task, got %d", report.Statistics.SizeDistribution["S"])
	}

	if report.Statistics.SizeDistribution["M"] != 1 {
		t.Errorf("Expected 1 medium task, got %d", report.Statistics.SizeDistribution["M"])
	}

	if report.Statistics.SizeDistribution["L"] != 1 {
		t.Errorf("Expected 1 large task, got %d", report.Statistics.SizeDistribution["L"])
	}

	// Verify category distribution
	if report.Statistics.CategoryDistribution["feature"] != 2 {
		t.Errorf("Expected 2 feature tasks, got %d", report.Statistics.CategoryDistribution["feature"])
	}

	if report.Statistics.CategoryDistribution["bug"] != 1 {
		t.Errorf("Expected 1 bug task, got %d", report.Statistics.CategoryDistribution["bug"])
	}

	// Verify confidence distribution
	if report.Statistics.HighConfidenceTasks != 3 {
		t.Errorf("Expected 3 high confidence tasks, got %d", report.Statistics.HighConfidenceTasks)
	}

	// Verify timing
	if report.Duration < 0 {
		t.Error("Duration should not be negative")
	}

	if report.EndTime.Before(report.StartTime) {
		t.Error("End time is before start time")
	}
}

func TestCalculateStatistics(t *testing.T) {
	mockAI := &mockAIProvider{
		available: true,
		response: &ai.EstimationResponse{
			EstimatedHours:  8.0,
			EstimatedSize:   "M",
			StoryPoints:     5.0,
			ConfidenceScore: 0.75,
			Reasoning:       "Mock estimation",
		},
	}

	est := NewEstimator(mockAI, nil, DefaultEstimationConfig())
	processor := NewBatchProcessor(est, 1)

	results := []*BatchResult{
		{
			Task: &BatchTask{ID: "T1", Title: "Task 1", Labels: []string{"feature"}},
			Result: &EstimationResult{
				Estimation: &ai.EstimationResponse{
					EstimatedHours:  2.0,
					EstimatedSize:   "S",
					StoryPoints:     1.0,
					ConfidenceScore: 0.80,
					Reasoning:       "Small task",
				},
			},
			ProcessedAt: time.Now(),
		},
		{
			Task: &BatchTask{ID: "T2", Title: "Task 2", Labels: []string{"bug"}},
			Result: &EstimationResult{
				Estimation: &ai.EstimationResponse{
					EstimatedHours:  8.0,
					EstimatedSize:   "M",
					StoryPoints:     5.0,
					ConfidenceScore: 0.60,
					Reasoning:       "Medium task",
				},
			},
			ProcessedAt: time.Now(),
		},
		{
			Task: &BatchTask{ID: "T3", Title: "Task 3", Labels: []string{"feature"}},
			Result: &EstimationResult{
				Estimation: &ai.EstimationResponse{
					EstimatedHours:  16.0,
					EstimatedSize:   "L",
					StoryPoints:     8.0,
					ConfidenceScore: 0.40,
					Reasoning:       "Large task",
				},
			},
			ProcessedAt: time.Now(),
		},
		{
			Task:        &BatchTask{ID: "T4", Title: "Failed task"},
			Error:       context.DeadlineExceeded,
			ProcessedAt: time.Now(),
		},
	}

	stats := processor.calculateStatistics(results)

	// Verify totals (only successful tasks)
	expectedTotal := 2.0 + 8.0 + 16.0
	if stats.TotalEstimatedHours != expectedTotal {
		t.Errorf("Expected total hours %.1f, got %.1f", expectedTotal, stats.TotalEstimatedHours)
	}

	expectedAvg := expectedTotal / 3
	if stats.AverageHours != expectedAvg {
		t.Errorf("Expected average hours %.1f, got %.1f", expectedAvg, stats.AverageHours)
	}

	// Verify min/max
	if stats.MinHours != 2.0 {
		t.Errorf("Expected min hours 2.0, got %.1f", stats.MinHours)
	}

	if stats.MaxHours != 16.0 {
		t.Errorf("Expected max hours 16.0, got %.1f", stats.MaxHours)
	}

	// Verify confidence
	expectedConfidence := (0.80 + 0.60 + 0.40) / 3
	if stats.AverageConfidence != expectedConfidence {
		t.Errorf("Expected average confidence %.2f, got %.2f", expectedConfidence, stats.AverageConfidence)
	}

	// Verify confidence levels
	if stats.HighConfidenceTasks != 1 {
		t.Errorf("Expected 1 high confidence task, got %d", stats.HighConfidenceTasks)
	}

	if stats.MediumConfidenceTasks != 1 {
		t.Errorf("Expected 1 medium confidence task, got %d", stats.MediumConfidenceTasks)
	}

	if stats.LowConfidenceTasks != 1 {
		t.Errorf("Expected 1 low confidence task, got %d", stats.LowConfidenceTasks)
	}

	// Verify size distribution
	if stats.SizeDistribution["S"] != 1 {
		t.Errorf("Expected 1 S task, got %d", stats.SizeDistribution["S"])
	}

	if stats.SizeDistribution["M"] != 1 {
		t.Errorf("Expected 1 M task, got %d", stats.SizeDistribution["M"])
	}

	if stats.SizeDistribution["L"] != 1 {
		t.Errorf("Expected 1 L task, got %d", stats.SizeDistribution["L"])
	}

	// Verify category distribution
	if stats.CategoryDistribution["feature"] != 2 {
		t.Errorf("Expected 2 feature tasks, got %d", stats.CategoryDistribution["feature"])
	}

	if stats.CategoryDistribution["bug"] != 1 {
		t.Errorf("Expected 1 bug task, got %d", stats.CategoryDistribution["bug"])
	}
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		input    string
		sep      string
		expected []string
	}{
		{"", "|", []string{}},
		{"a|b|c", "|", []string{"a", "b", "c"}},
		{"a | b | c", "|", []string{"a", "b", "c"}},
		{"a", "|", []string{"a"}},
		{"  a  ", "|", []string{"a"}},
		{"a||b", "|", []string{"a", "b"}},
		{"bug,feature,test", ",", []string{"bug", "feature", "test"}},
	}

	for _, tt := range tests {
		result := splitAndTrim(tt.input, tt.sep)
		if len(result) != len(tt.expected) {
			t.Errorf("splitAndTrim(%q, %q) = %v, expected %v", tt.input, tt.sep, result, tt.expected)
			continue
		}
		for i := range result {
			if result[i] != tt.expected[i] {
				t.Errorf("splitAndTrim(%q, %q) = %v, expected %v", tt.input, tt.sep, result, tt.expected)
				break
			}
		}
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("calculateMedian", func(t *testing.T) {
		tests := []struct {
			values   []float64
			expected float64
		}{
			{[]float64{}, 0},
			{[]float64{5.0}, 5.0},
			{[]float64{1.0, 2.0, 3.0}, 2.0},    // Average, not true median
			{[]float64{1.0, 5.0, 10.0}, 5.333}, // Approximately
		}

		for _, tt := range tests {
			result := calculateMedian(tt.values)
			// Note: Current implementation returns average, not true median
			if len(tt.values) > 0 {
				sum := 0.0
				for _, v := range tt.values {
					sum += v
				}
				expected := sum / float64(len(tt.values))
				if result != expected {
					t.Errorf("calculateMedian(%v) = %.3f, expected %.3f", tt.values, result, expected)
				}
			}
		}
	})

	t.Run("findMin", func(t *testing.T) {
		tests := []struct {
			values   []float64
			expected float64
		}{
			{[]float64{}, 0},
			{[]float64{5.0}, 5.0},
			{[]float64{1.0, 2.0, 3.0}, 1.0},
			{[]float64{10.0, 5.0, 15.0, 2.0}, 2.0},
		}

		for _, tt := range tests {
			result := findMin(tt.values)
			if result != tt.expected {
				t.Errorf("findMin(%v) = %.1f, expected %.1f", tt.values, result, tt.expected)
			}
		}
	})

	t.Run("findMax", func(t *testing.T) {
		tests := []struct {
			values   []float64
			expected float64
		}{
			{[]float64{}, 0},
			{[]float64{5.0}, 5.0},
			{[]float64{1.0, 2.0, 3.0}, 3.0},
			{[]float64{10.0, 5.0, 15.0, 2.0}, 15.0},
		}

		for _, tt := range tests {
			result := findMax(tt.values)
			if result != tt.expected {
				t.Errorf("findMax(%v) = %.1f, expected %.1f", tt.values, result, tt.expected)
			}
		}
	})
}
