package estimator

import (
	"context"
	"fmt"
	"time"

	"set/internal/ai"
	"set/internal/logger"
	"set/internal/storage"
)

// Estimator handles task estimation
type Estimator struct {
	aiProvider ai.AIProvider
	storage    *storage.Store
	config     *EstimationConfig
}

// NewEstimator creates a new estimator
func NewEstimator(aiProvider ai.AIProvider, store *storage.Store, config *EstimationConfig) *Estimator {
	if config == nil {
		config = DefaultEstimationConfig()
	}

	return &Estimator{
		aiProvider: aiProvider,
		storage:    store,
		config:     config,
	}
}

// Estimate estimates a task
func (e *Estimator) Estimate(ctx context.Context, task *Task) (*EstimationResult, error) {
	startTime := time.Now()
	logger.Infof("Estimating task: %s", task.Title)

	result := &EstimationResult{
		Task: task,
	}

	// Step 1: Find similar historical tasks
	loadStart := time.Now()
	historical, err := e.loadHistoricalTasks()
	if err != nil {
		logger.Warnf("Could not load historical tasks: %v", err)
		historical = []*HistoricalTask{}
	}
	loadDuration := time.Since(loadStart)

	logger.Infof("Loaded %d historical tasks from storage in %v", len(historical), loadDuration)

	// Calculate dataset statistics for AI context (includes percentiles and category breakdown)
	result.DatasetStats = calculateDatasetStatistics(historical)

	// Use adaptive similarity search to find most similar tasks
	similarStart := time.Now()
	similarTasks, simContext := FindSimilarTasksAdaptive(task, historical, e.config)
	result.SimilarityContext = simContext
	similarDuration := time.Since(similarStart)

	logger.Infof("Found %d similar tasks in %v", len(similarTasks), similarDuration)

	// Enhance dataset with stratified and percentile samples (minimum 10 tasks)
	enhancedStart := time.Now()
	enhancedDataset := selectEnhancedDataset(similarTasks, historical, e.config)
	result.SimilarTasks = enhancedDataset
	enhancedDuration := time.Since(enhancedStart)

	logger.Infof("Enhanced dataset with %d total tasks (stratified + percentile samples) in %v",
		len(enhancedDataset), enhancedDuration)

	// AI-only mode: Always use AI for estimation
	if e.aiProvider == nil || !e.aiProvider.IsAvailable() {
		return nil, fmt.Errorf("AI provider not available - this tool requires AI for estimation")
	}

	estimation, err := e.estimateWithAI(task, enhancedDataset, result.DatasetStats, result.SimilarityContext)
	if err != nil {
		return nil, fmt.Errorf("AI estimation failed: %w", err)
	}

	result.Estimation = estimation
	result.Method = "ai"
	result.DataSource = "openai"
	totalDuration := time.Since(startTime)

	logger.Infof("AI estimation: %.1f hours, %.0f%% confidence",
		estimation.EstimatedHours, estimation.ConfidenceScore*100)
	logger.Infof("Total estimation time: %v", totalDuration)

	return result, nil
}

// estimateWithAI uses AI to estimate the task
func (e *Estimator) estimateWithAI(task *Task, similar []*SimilarityMatch, stats *DatasetStatistics, simContext *SimilarityContext) (*ai.EstimationResponse, error) {
	// Build AI request
	aiRequest := &ai.EstimationRequest{
		TaskTitle:       task.Title,
		TaskDescription: task.Description,
		Context:         task.Context,
	}

	// Add dataset statistics for general context
	if stats != nil {
		// Convert CategoryStats to AI type
		categoryBreakdown := make(map[string]*ai.CategoryStats)
		for k, v := range stats.CategoryBreakdown {
			categoryBreakdown[k] = &ai.CategoryStats{
				Count:      v.Count,
				AvgHours:   v.AvgHours,
				MinHours:   v.MinHours,
				MaxHours:   v.MaxHours,
				TaskTitles: v.TaskTitles,
			}
		}

		aiRequest.DatasetStats = &ai.DatasetStats{
			TotalTasks:        stats.TotalTasks,
			ClosedTasks:       stats.ClosedTasks,
			AvgHours:          stats.AvgHours,
			MedianHours:       stats.MedianHours,
			TasksBySize:       stats.TasksBySize,
			TasksByCategory:   stats.TasksByCategory,
			CategoryBreakdown: categoryBreakdown,
			PercentileHours:   stats.PercentileHours,
		}
	}

	// Add similarity context
	if simContext != nil {
		aiRequest.SimilarityContext = &ai.SimilarityMeta{
			ThresholdUsed:     simContext.ThresholdUsed,
			HighestSimilarity: simContext.HighestSimilarity,
			MatchesFound:      simContext.MatchesFound,
		}
	}

	// Add similar tasks for context
	for _, match := range similar {
		if match.Task.Issue == nil {
			continue
		}

		similarTask := ai.SimilarTask{
			Title:         match.Task.Issue.Title,
			Description:   match.Task.Issue.Body,
			ActualHours:   match.Task.ActualHours,
			EstimatedSize: match.Task.EstimatedSize,
			StoryPoints:   match.Task.StoryPoints,
			Labels:        extractLabels(match.Task.Issue),
			CustomFields:  match.Task.Issue.CustomFields,
			Similarity:    match.Similarity,
		}
		aiRequest.SimilarTasks = append(aiRequest.SimilarTasks, similarTask)
	}

	// Request estimation from AI
	return e.aiProvider.EstimateTask(aiRequest)
}

// calculateDatasetStatistics generates statistical summary of historical tasks
// Includes percentiles and category breakdown
func calculateDatasetStatistics(historical []*HistoricalTask) *DatasetStatistics {
	if len(historical) == 0 {
		return nil
	}

	stats := &DatasetStatistics{
		TasksBySize:       make(map[string]int),
		TasksByCategory:   make(map[string]int),
		CategoryBreakdown: make(map[string]*CategoryStats),
	}

	var totalHours float64
	var hoursCount int
	var hours []float64
	categoryHours := make(map[string][]float64)
	categoryTitles := make(map[string][]string)

	for _, hist := range historical {
		stats.TotalTasks++

		if hist.Issue != nil && hist.Issue.State == "closed" {
			stats.ClosedTasks++
		}

		// Collect hours data
		if hist.ActualHours > 0 {
			totalHours += hist.ActualHours
			hours = append(hours, hist.ActualHours)
			hoursCount++
		}

		// Count by size
		if hist.EstimatedSize != "" {
			stats.TasksBySize[hist.EstimatedSize]++
		}

		// Count by category (labels) and collect hours per category
		if hist.Issue != nil {
			for _, label := range hist.Issue.Labels {
				stats.TasksByCategory[label.Name]++

				if hist.ActualHours > 0 {
					categoryHours[label.Name] = append(categoryHours[label.Name], hist.ActualHours)
				}

				// Collect sample titles (limit to 3 per category)
				if len(categoryTitles[label.Name]) < 3 {
					categoryTitles[label.Name] = append(categoryTitles[label.Name], hist.Issue.Title)
				}
			}
		}
	}

	// Calculate averages
	if hoursCount > 0 {
		stats.AvgHours = totalHours / float64(hoursCount)

		// Sort hours for median and percentiles
		if len(hours) > 0 {
			sortedHours := make([]float64, len(hours))
			copy(sortedHours, hours)
			// Sort hours
			for i := 0; i < len(sortedHours); i++ {
				for j := i + 1; j < len(sortedHours); j++ {
					if sortedHours[i] > sortedHours[j] {
						sortedHours[i], sortedHours[j] = sortedHours[j], sortedHours[i]
					}
				}
			}

			// Calculate median
			mid := len(sortedHours) / 2
			if len(sortedHours)%2 == 0 {
				stats.MedianHours = (sortedHours[mid-1] + sortedHours[mid]) / 2
			} else {
				stats.MedianHours = sortedHours[mid]
			}

			// Calculate percentiles: 10th, 25th, 50th, 75th, 90th
			stats.PercentileHours = make([]float64, 5)
			percentiles := []float64{0.10, 0.25, 0.50, 0.75, 0.90}
			for i, p := range percentiles {
				idx := int(float64(len(sortedHours)-1) * p)
				stats.PercentileHours[i] = sortedHours[idx]
			}
		}
	}

	// Calculate category breakdown statistics
	for category, hoursSlice := range categoryHours {
		if len(hoursSlice) == 0 {
			continue
		}

		catStats := &CategoryStats{
			Count:      len(hoursSlice),
			TaskTitles: categoryTitles[category],
		}

		// Calculate min, max, avg
		var total float64
		catStats.MinHours = hoursSlice[0]
		catStats.MaxHours = hoursSlice[0]

		for _, h := range hoursSlice {
			total += h
			if h < catStats.MinHours {
				catStats.MinHours = h
			}
			if h > catStats.MaxHours {
				catStats.MaxHours = h
			}
		}
		catStats.AvgHours = total / float64(len(hoursSlice))

		stats.CategoryBreakdown[category] = catStats
	}

	return stats
}

// selectEnhancedDataset selects a diverse set of tasks to send to AI
// Combines: similar tasks + stratified samples + percentile samples
func selectEnhancedDataset(similarTasks []*SimilarityMatch, historical []*HistoricalTask, config *EstimationConfig) []*SimilarityMatch {
	selectedMap := make(map[*HistoricalTask]bool)
	var enhanced []*SimilarityMatch

	// Step 1: Add top similar tasks (prioritized)
	for _, match := range similarTasks {
		if len(enhanced) >= config.MaxSimilarTasks {
			break
		}
		enhanced = append(enhanced, match)
		selectedMap[match.Task] = true
	}

	// Step 2: Add stratified samples by size (ensuring diversity)
	if len(enhanced) < config.MinSimilarTasks {
		sizeBuckets := make(map[string][]*HistoricalTask)
		for _, hist := range historical {
			if selectedMap[hist] {
				continue // Already selected
			}
			if hist.EstimatedSize != "" {
				sizeBuckets[hist.EstimatedSize] = append(sizeBuckets[hist.EstimatedSize], hist)
			}
		}

		// Sample from each size bucket
		sizes := []string{"XS", "S", "M", "L", "XL"}
		for _, size := range sizes {
			if len(enhanced) >= config.MinSimilarTasks {
				break
			}
			bucket := sizeBuckets[size]
			samplesNeeded := config.StratifiedSamplesPerSize
			if samplesNeeded > len(bucket) {
				samplesNeeded = len(bucket)
			}

			// Take random samples from bucket
			for i := 0; i < samplesNeeded && len(enhanced) < config.MinSimilarTasks; i++ {
				if i < len(bucket) {
					hist := bucket[i]
					enhanced = append(enhanced, &SimilarityMatch{
						Task:       hist,
						Similarity: 0.0, // Not from similarity search
						Matches:    []string{"stratified_sample"},
					})
					selectedMap[hist] = true
				}
			}
		}
	}

	// Step 3: Add percentile-based samples if still below minimum
	if len(enhanced) < config.MinSimilarTasks {
		// Sort historical by hours
		var tasksWithHours []*HistoricalTask
		for _, hist := range historical {
			if selectedMap[hist] {
				continue
			}
			if hist.ActualHours > 0 {
				tasksWithHours = append(tasksWithHours, hist)
			}
		}

		// Sort by hours
		for i := 0; i < len(tasksWithHours); i++ {
			for j := i + 1; j < len(tasksWithHours); j++ {
				if tasksWithHours[i].ActualHours > tasksWithHours[j].ActualHours {
					tasksWithHours[i], tasksWithHours[j] = tasksWithHours[j], tasksWithHours[i]
				}
			}
		}

		// Select tasks at percentiles: 10th, 25th, 50th, 75th, 90th
		if len(tasksWithHours) > 0 {
			percentiles := []float64{0.10, 0.25, 0.50, 0.75, 0.90}
			for _, p := range percentiles {
				if len(enhanced) >= config.MinSimilarTasks {
					break
				}
				idx := int(float64(len(tasksWithHours)-1) * p)
				hist := tasksWithHours[idx]
				if !selectedMap[hist] {
					enhanced = append(enhanced, &SimilarityMatch{
						Task:       hist,
						Similarity: 0.0,
						Matches:    []string{fmt.Sprintf("percentile_%.0f", p*100)},
					})
					selectedMap[hist] = true
				}
			}
		}
	}

	logger.Infof("Enhanced dataset: %d similar + %d stratified + %d percentile = %d total tasks",
		len(similarTasks),
		len(enhanced)-len(similarTasks)-(len(enhanced)-len(similarTasks))/2,
		(len(enhanced)-len(similarTasks))/2,
		len(enhanced))

	return enhanced
}

// loadHistoricalTasks loads historical tasks from storage
func (e *Estimator) loadHistoricalTasks() ([]*HistoricalTask, error) {
	if e.storage == nil {
		return nil, fmt.Errorf("storage not configured")
	}

	// Load all closed issues
	issues, err := e.storage.GetAllIssues()
	if err != nil {
		return nil, err
	}

	historical := make([]*HistoricalTask, 0, len(issues))

	for _, issue := range issues {
		if issue.State != "closed" {
			continue // Only use completed tasks
		}

		hist := &HistoricalTask{
			Issue: issue,
		}

		// Extract actual hours from custom fields
		if len(issue.CustomFields) > 0 {
			hist.ActualHours = extractFloat(issue.CustomFields, "Worker Hours", "Hours", "Actual Hours")
			hist.StoryPoints = extractFloat(issue.CustomFields, "Story Points", "Points")
			hist.EstimatedSize = extractString(issue.CustomFields, "Size", "Complexity")
		}

		// Only add if we have some estimation data
		if hist.ActualHours > 0 || hist.StoryPoints > 0 || hist.EstimatedSize != "" {
			historical = append(historical, hist)
		}
	}

	logger.Infof("Loaded %d historical tasks with estimation data", len(historical))

	return historical, nil
}

// Helper functions

func extractFloat(fields map[string]interface{}, keys ...string) float64 {
	for _, key := range keys {
		if val, exists := fields[key]; exists {
			switch v := val.(type) {
			case float64:
				return v
			case int:
				return float64(v)
			case string:
				var f float64
				if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
					return f
				}
			}
		}
	}
	return 0
}

func extractString(fields map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, exists := fields[key]; exists {
			if s, ok := val.(string); ok {
				return s
			}
		}
	}
	return ""
}

func mostCommon(items []string) string {
	if len(items) == 0 {
		return ""
	}

	counts := make(map[string]int)
	for _, item := range items {
		counts[item]++
	}

	maxCount := 0
	most := items[0]

	for item, count := range counts {
		if count > maxCount {
			maxCount = count
			most = item
		}
	}

	return most
}

func calculateConfidence(similar []*SimilarityMatch, dataPoints int) float64 {
	if len(similar) == 0 || dataPoints == 0 {
		return 0.3
	}

	// Base confidence on number of similar tasks and their similarity scores
	avgSimilarity := 0.0
	for _, match := range similar {
		avgSimilarity += match.Similarity
	}
	avgSimilarity /= float64(len(similar))

	// More data points = higher confidence
	dataFactor := float64(dataPoints) / 5.0 // Normalize to max of 5 data points
	if dataFactor > 1.0 {
		dataFactor = 1.0
	}

	// Combine factors
	confidence := (avgSimilarity * 0.7) + (dataFactor * 0.3)

	// Cap between 0.3 and 0.9
	if confidence < 0.3 {
		confidence = 0.3
	}
	if confidence > 0.9 {
		confidence = 0.9
	}

	return confidence
}

func sizeToPoints(size ai.Size) float64 {
	switch size {
	case ai.SizeXS:
		return 1
	case ai.SizeS:
		return 2
	case ai.SizeM:
		return 5
	case ai.SizeL:
		return 8
	case ai.SizeXL:
		return 13
	default:
		return 3
	}
}
