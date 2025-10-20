package estimator

import (
	"set/internal/ai"
	"set/internal/github"
)

// Task represents a task to be estimated
type Task struct {
	Title       string
	Description string
	Labels      []string
	Context     map[string]interface{} // Custom fields, metadata
}

// HistoricalTask represents a completed task with actual data
type HistoricalTask struct {
	Issue         *github.Issue
	ActualHours   float64
	EstimatedSize string
	StoryPoints   float64
}

// SimilarityMatch represents a task match with similarity score
type SimilarityMatch struct {
	Task       *HistoricalTask
	Similarity float64  // 0.0 to 1.0
	Matches    []string // What matched (title, description, labels, etc.)
}

// EstimationConfig holds configuration for the estimator
type EstimationConfig struct {
	MaxSimilarTasks          int     // Maximum similar tasks to send (adaptive based on similarity)
	MinSimilarTasks          int     // Minimum tasks to send to AI
	MinSimilarityThreshold   float64 // Starting threshold for adaptive search
	StratifiedSamplesPerSize int     // Number of samples per size category
}

// DefaultEstimationConfig returns default configuration
func DefaultEstimationConfig() *EstimationConfig {
	return &EstimationConfig{
		MaxSimilarTasks:          15, // Can be increased based on similarity quality
		MinSimilarTasks:          10, // Always send at least 10 tasks
		MinSimilarityThreshold:   0.3,
		StratifiedSamplesPerSize: 2, // 2 samples per size (XS, S, M, L, XL = 10 total)
	}
}

// EstimationResult contains the complete estimation result
type EstimationResult struct {
	Task              *Task
	Estimation        *ai.EstimationResponse
	SimilarTasks      []*SimilarityMatch
	Method            string // "ai", "similarity", "custom_fields", "average"
	DataSource        string // Where the estimate came from
	DatasetStats      *DatasetStatistics
	SimilarityContext *SimilarityContext
}

// DatasetStatistics provides statistical context about the historical dataset
type DatasetStatistics struct {
	TotalTasks        int
	ClosedTasks       int
	AvgHours          float64
	MedianHours       float64
	TasksBySize       map[string]int
	TasksByCategory   map[string]int
	CategoryBreakdown map[string]*CategoryStats // Detailed breakdown by category
	PercentileHours   []float64                 // Hours at 10th, 25th, 50th, 75th, 90th percentiles
}

// CategoryStats provides detailed statistics for a category
type CategoryStats struct {
	Count      int
	AvgHours   float64
	MinHours   float64
	MaxHours   float64
	TaskTitles []string // Sample task titles from this category
}

// SimilarityContext provides information about the similarity search process
type SimilarityContext struct {
	ThresholdUsed        float64
	ThresholdsTriedCount int
	TotalTasksScanned    int
	MatchesFound         int
	HighestSimilarity    float64
}

// GetConfidenceLevel returns a text description of confidence
func (r *EstimationResult) GetConfidenceLevel() string {
	if r.Estimation == nil {
		return "unknown"
	}

	score := r.Estimation.ConfidenceScore
	switch {
	case score >= 0.8:
		return "high"
	case score >= 0.5:
		return "medium"
	default:
		return "low"
	}
}

// ConfidenceScore returns the numeric confidence score
func (r *EstimationResult) ConfidenceScore() float64 {
	if r.Estimation == nil {
		return 0.0
	}
	return r.Estimation.ConfidenceScore
}
