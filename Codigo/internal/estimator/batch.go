package estimator

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"set/internal/logger"
)

// BatchTask represents a task in a batch estimation request
type BatchTask struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	Labels      []string          `json:"labels,omitempty"`
	Context     map[string]string `json:"context,omitempty"`
	Priority    string            `json:"priority,omitempty"`
	Assignee    string            `json:"assignee,omitempty"`
}

// BatchRequest represents a batch estimation request
type BatchRequest struct {
	Metadata struct {
		Project   string    `json:"project,omitempty"`
		Sprint    string    `json:"sprint,omitempty"`
		CreatedBy string    `json:"created_by,omitempty"`
		CreatedAt time.Time `json:"created_at,omitempty"`
	} `json:"metadata,omitempty"`
	Tasks []BatchTask `json:"tasks"`
}

// BatchResult represents the result of a single task estimation in a batch
type BatchResult struct {
	Task        *BatchTask
	Result      *EstimationResult
	Error       error
	ProcessedAt time.Time
}

// BatchReport represents the complete batch estimation report
type BatchReport struct {
	Request         *BatchRequest
	Results         []*BatchResult
	StartTime       time.Time
	EndTime         time.Time
	Duration        time.Duration
	TotalTasks      int
	SuccessfulTasks int
	FailedTasks     int
	Statistics      *BatchStatistics
}

// BatchStatistics contains aggregate statistics for a batch
type BatchStatistics struct {
	TotalEstimatedHours float64
	AverageHours        float64
	MedianHours         float64
	MinHours            float64
	MaxHours            float64
	AverageConfidence   float64

	// Size distribution
	SizeDistribution map[string]int     // XS, S, M, L, XL -> count
	SizeHours        map[string]float64 // XS, S, M, L, XL -> total hours

	// Category distribution
	CategoryDistribution map[string]int
	CategoryHours        map[string]float64

	// Confidence levels
	HighConfidenceTasks   int // >= 70%
	MediumConfidenceTasks int // 50-69%
	LowConfidenceTasks    int // < 50%
}

// BatchProcessor handles batch estimation operations
type BatchProcessor struct {
	estimator   *Estimator
	maxWorkers  int
	rateLimiter chan struct{} // Semaphore for rate limiting
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(estimator *Estimator, maxWorkers int) *BatchProcessor {
	if maxWorkers <= 0 {
		maxWorkers = 5 // Default to 5 concurrent workers
	}

	return &BatchProcessor{
		estimator:   estimator,
		maxWorkers:  maxWorkers,
		rateLimiter: make(chan struct{}, maxWorkers),
	}
}

// LoadBatchRequest loads a batch request from a JSON file
func LoadBatchRequest(filePath string) (*BatchRequest, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var request BatchRequest
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&request); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	if len(request.Tasks) == 0 {
		return nil, fmt.Errorf("no tasks found in batch request")
	}

	return &request, nil
}

// LoadBatchRequestFromCSV loads a batch request from a CSV file
func LoadBatchRequestFromCSV(filePath string) (*BatchRequest, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) < 2 { // Need at least header + 1 task
		return nil, fmt.Errorf("CSV file is empty or has no data rows")
	}

	// Parse header to find column indices
	header := records[0]
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[col] = i
	}

	// Required columns
	titleIdx, hasTitleIdx := colMap["title"]
	if !hasTitleIdx {
		return nil, fmt.Errorf("CSV must have 'title' column")
	}

	// Optional columns
	idIdx := colMap["id"]
	descIdx := colMap["description"]
	labelsIdx := colMap["labels"]
	priorityIdx := colMap["priority"]
	assigneeIdx := colMap["assignee"]

	request := &BatchRequest{}

	// Parse data rows
	for i, record := range records[1:] {
		if len(record) <= titleIdx {
			logger.Warnf("Skipping row %d: insufficient columns", i+2)
			continue
		}

		task := BatchTask{
			Title: record[titleIdx],
		}

		// ID (use row number if not provided)
		if idIdx < len(record) && record[idIdx] != "" {
			task.ID = record[idIdx]
		} else {
			task.ID = fmt.Sprintf("TASK-%03d", i+1)
		}

		// Description
		if descIdx < len(record) {
			task.Description = record[descIdx]
		}

		// Labels (pipe-separated)
		if labelsIdx < len(record) && record[labelsIdx] != "" {
			labels := splitAndTrim(record[labelsIdx], "|")
			task.Labels = labels
		}

		// Priority
		if priorityIdx < len(record) {
			task.Priority = record[priorityIdx]
		}

		// Assignee
		if assigneeIdx < len(record) {
			task.Assignee = record[assigneeIdx]
		}

		request.Tasks = append(request.Tasks, task)
	}

	if len(request.Tasks) == 0 {
		return nil, fmt.Errorf("no valid tasks found in CSV")
	}

	return request, nil
}

// ProcessBatch processes a batch of tasks and returns a report
func (bp *BatchProcessor) ProcessBatch(ctx context.Context, request *BatchRequest) (*BatchReport, error) {
	startTime := time.Now()

	logger.Infof("Starting batch processing: %d tasks", len(request.Tasks))

	// Initialize report
	report := &BatchReport{
		Request:    request,
		Results:    make([]*BatchResult, 0, len(request.Tasks)),
		StartTime:  startTime,
		TotalTasks: len(request.Tasks),
	}

	// Create channels for work distribution
	taskCh := make(chan *BatchTask, len(request.Tasks))
	resultCh := make(chan *BatchResult, len(request.Tasks))

	// Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < bp.maxWorkers; i++ {
		wg.Add(1)
		go bp.worker(ctx, &wg, taskCh, resultCh)
	}

	// Send tasks to workers
	for i := range request.Tasks {
		taskCh <- &request.Tasks[i]
	}
	close(taskCh)

	// Collect results
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Gather results
	for result := range resultCh {
		report.Results = append(report.Results, result)
		if result.Error != nil {
			report.FailedTasks++
			logger.Warnf("Task %s failed: %v", result.Task.ID, result.Error)
		} else {
			report.SuccessfulTasks++
		}
	}

	// Finalize report
	report.EndTime = time.Now()
	report.Duration = report.EndTime.Sub(report.StartTime)

	// Calculate statistics
	report.Statistics = bp.calculateStatistics(report.Results)

	logger.Infof("Batch processing completed: %d successful, %d failed, duration: %s",
		report.SuccessfulTasks, report.FailedTasks, report.Duration)

	return report, nil
}

// worker processes tasks from the task channel
func (bp *BatchProcessor) worker(ctx context.Context, wg *sync.WaitGroup, taskCh <-chan *BatchTask, resultCh chan<- *BatchResult) {
	defer wg.Done()

	for batchTask := range taskCh {
		// Rate limiting
		bp.rateLimiter <- struct{}{}

		result := &BatchResult{
			Task:        batchTask,
			ProcessedAt: time.Now(),
		}

		// Convert BatchTask to Task
		task := &Task{
			Title:       batchTask.Title,
			Description: batchTask.Description,
			Labels:      batchTask.Labels,
			Context:     make(map[string]interface{}),
		}

		// Convert context
		for k, v := range batchTask.Context {
			task.Context[k] = v
		}

		// Perform estimation
		estimation, err := bp.estimator.Estimate(ctx, task)
		if err != nil {
			result.Error = err
		} else {
			result.Result = estimation
		}

		resultCh <- result

		// Release rate limiter
		<-bp.rateLimiter
	}
}

// calculateStatistics computes aggregate statistics for the batch
func (bp *BatchProcessor) calculateStatistics(results []*BatchResult) *BatchStatistics {
	stats := &BatchStatistics{
		SizeDistribution:     make(map[string]int),
		SizeHours:            make(map[string]float64),
		CategoryDistribution: make(map[string]int),
		CategoryHours:        make(map[string]float64),
	}

	var hours []float64
	var confidenceSum float64
	var confidenceCount int

	for _, result := range results {
		if result.Error != nil || result.Result == nil || result.Result.Estimation == nil {
			continue
		}

		est := result.Result.Estimation

		// Hours
		hours = append(hours, est.EstimatedHours)
		stats.TotalEstimatedHours += est.EstimatedHours

		// Size
		stats.SizeDistribution[est.EstimatedSize]++
		stats.SizeHours[est.EstimatedSize] += est.EstimatedHours

		// Categories (from labels)
		for _, label := range result.Task.Labels {
			stats.CategoryDistribution[label]++
			stats.CategoryHours[label] += est.EstimatedHours
		}

		// Confidence
		confidenceSum += est.ConfidenceScore
		confidenceCount++

		// Confidence levels
		if est.ConfidenceScore >= 0.70 {
			stats.HighConfidenceTasks++
		} else if est.ConfidenceScore >= 0.50 {
			stats.MediumConfidenceTasks++
		} else {
			stats.LowConfidenceTasks++
		}
	}

	// Calculate averages
	if len(hours) > 0 {
		stats.AverageHours = stats.TotalEstimatedHours / float64(len(hours))
		stats.MedianHours = calculateMedian(hours)
		stats.MinHours = findMin(hours)
		stats.MaxHours = findMax(hours)
	}

	if confidenceCount > 0 {
		stats.AverageConfidence = confidenceSum / float64(confidenceCount)
	}

	return stats
}

// Helper functions

func splitAndTrim(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	parts := []string{}
	// Split by separator and trim whitespace
	for _, part := range strings.Split(s, sep) {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func calculateMedian(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Simple median calculation (not sorting for performance)
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func findMin(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func findMax(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}
