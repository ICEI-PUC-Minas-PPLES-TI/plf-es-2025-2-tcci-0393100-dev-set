package estimator

import (
	"fmt"
	"math"
	"strings"
	"unicode"

	"set/internal/github"
	"set/internal/logger"
)

// CalculateSimilarity calculates similarity between a task and a historical issue
func CalculateSimilarity(task *Task, historical *HistoricalTask) float64 {
	if historical == nil || historical.Issue == nil {
		return 0.0
	}

	var totalScore float64
	var totalWeight float64

	// Title similarity (weight: 0.4)
	titleScore := calculateTextSimilarity(task.Title, historical.Issue.Title)
	totalScore += titleScore * 0.4
	totalWeight += 0.4

	// Description similarity (weight: 0.3)
	if task.Description != "" && historical.Issue.Body != "" {
		descScore := calculateTextSimilarity(task.Description, historical.Issue.Body)
		totalScore += descScore * 0.3
		totalWeight += 0.3
	}

	// Label similarity (weight: 0.2)
	labelScore := calculateLabelSimilarity(task.Labels, extractLabels(historical.Issue))
	totalScore += labelScore * 0.2
	totalWeight += 0.2

	// Custom fields similarity (weight: 0.1)
	if len(task.Context) > 0 && len(historical.Issue.CustomFields) > 0 {
		contextScore := calculateContextSimilarity(task.Context, historical.Issue.CustomFields)
		totalScore += contextScore * 0.1
		totalWeight += 0.1
	}

	// Normalize by total weight
	if totalWeight > 0 {
		return totalScore / totalWeight
	}

	return 0.0
}

// FindSimilarTasks finds similar historical tasks
func FindSimilarTasks(task *Task, historical []*HistoricalTask, config *EstimationConfig) []*SimilarityMatch {
	if config == nil {
		config = DefaultEstimationConfig()
	}

	var matches []*SimilarityMatch
	var maxSimilarity float64
	var maxSimilarityTask string

	for _, hist := range historical {
		similarity := CalculateSimilarity(task, hist)

		// Track highest similarity for debugging
		if similarity > maxSimilarity {
			maxSimilarity = similarity
			if hist.Issue != nil {
				maxSimilarityTask = hist.Issue.Title
			}
		}

		if similarity >= config.MinSimilarityThreshold {
			match := &SimilarityMatch{
				Task:       hist,
				Similarity: similarity,
				Matches:    identifyMatches(task, hist),
			}
			matches = append(matches, match)
		}
	}

	// Log debug info about similarity matching (only visible with --verbose)
	if len(matches) == 0 && len(historical) > 0 {
		logger.Debugf("No tasks above threshold (%.1f%%). Highest similarity: %.1f%% with '%s'",
			config.MinSimilarityThreshold*100, maxSimilarity*100, maxSimilarityTask)
	}

	// Sort by similarity (highest first)
	sortBySimilarity(matches)

	// Limit to max similar tasks
	if len(matches) > config.MaxSimilarTasks {
		matches = matches[:config.MaxSimilarTasks]
	}

	return matches
}

// FindSimilarTasksAdaptive performs adaptive similarity search with multiple thresholds
// It tries progressively lower thresholds until finding statistically relevant matches
func FindSimilarTasksAdaptive(task *Task, historical []*HistoricalTask, config *EstimationConfig) ([]*SimilarityMatch, *SimilarityContext) {
	if config == nil {
		config = DefaultEstimationConfig()
	}

	// Define threshold levels to try (from strict to permissive)
	thresholds := []float64{
		0.5,  // 50% - Very similar
		0.4,  // 40% - Quite similar
		0.3,  // 30% - Moderately similar (default)
		0.2,  // 20% - Somewhat similar
		0.15, // 15% - Loosely similar
	}

	// Minimum number of matches we want (statistically relevant)
	minDesiredMatches := 3
	maxMatches := config.MaxSimilarTasks

	var allSimilarities []float64
	var maxSimilarity float64
	var bestMatches []*SimilarityMatch
	var thresholdUsed float64
	var thresholdsTriedCount int

	// Calculate all similarities first
	type scoredTask struct {
		hist       *HistoricalTask
		similarity float64
		matches    []string
	}

	scoredTasks := make([]scoredTask, 0, len(historical))
	for _, hist := range historical {
		similarity := CalculateSimilarity(task, hist)
		allSimilarities = append(allSimilarities, similarity)

		if similarity > maxSimilarity {
			maxSimilarity = similarity
		}

		scoredTasks = append(scoredTasks, scoredTask{
			hist:       hist,
			similarity: similarity,
			matches:    identifyMatches(task, hist),
		})
	}

	// Try each threshold until we find enough matches
	for _, threshold := range thresholds {
		thresholdsTriedCount++
		var matches []*SimilarityMatch

		for _, scored := range scoredTasks {
			if scored.similarity >= threshold {
				matches = append(matches, &SimilarityMatch{
					Task:       scored.hist,
					Similarity: scored.similarity,
					Matches:    scored.matches,
				})
			}
		}

		// Sort by similarity (highest first)
		sortBySimilarity(matches)

		// Check if we have enough matches
		if len(matches) >= minDesiredMatches {
			thresholdUsed = threshold
			// Limit to max
			if len(matches) > maxMatches {
				bestMatches = matches[:maxMatches]
			} else {
				bestMatches = matches
			}
			logger.Infof("Adaptive similarity: Found %d matches at %.0f%% threshold (tried %d thresholds)",
				len(bestMatches), threshold*100, thresholdsTriedCount)
			break
		}

		// If this is the last threshold and we still don't have enough, use what we got
		if threshold == thresholds[len(thresholds)-1] && len(matches) > 0 {
			thresholdUsed = threshold
			if len(matches) > maxMatches {
				bestMatches = matches[:maxMatches]
			} else {
				bestMatches = matches
			}
			logger.Infof("Adaptive similarity: Found only %d matches at lowest threshold (%.0f%%)",
				len(bestMatches), threshold*100)
			break
		}
	}

	// If still no matches, use the configured threshold as fallback
	if len(bestMatches) == 0 {
		thresholdUsed = config.MinSimilarityThreshold
		logger.Debugf("No matches found even at 15%% threshold. Highest similarity was %.1f%%", maxSimilarity*100)
	}

	context := &SimilarityContext{
		ThresholdUsed:        thresholdUsed,
		ThresholdsTriedCount: thresholdsTriedCount,
		TotalTasksScanned:    len(historical),
		MatchesFound:         len(bestMatches),
		HighestSimilarity:    maxSimilarity,
	}

	return bestMatches, context
}

// calculateTextSimilarity calculates similarity between two text strings
func calculateTextSimilarity(text1, text2 string) float64 {
	// Normalize texts
	t1 := normalizeText(text1)
	t2 := normalizeText(text2)

	if t1 == "" || t2 == "" {
		return 0.0
	}

	// Use Jaccard similarity on word sets
	words1 := extractWords(t1)
	words2 := extractWords(t2)

	return jaccardSimilarity(words1, words2)
}

// calculateLabelSimilarity calculates similarity between label sets
func calculateLabelSimilarity(labels1, labels2 []string) float64 {
	if len(labels1) == 0 && len(labels2) == 0 {
		return 1.0 // Both empty = perfect match
	}
	if len(labels1) == 0 || len(labels2) == 0 {
		return 0.0
	}

	return jaccardSimilarity(labels1, labels2)
}

// calculateContextSimilarity calculates similarity between context maps
func calculateContextSimilarity(context1, context2 map[string]interface{}) float64 {
	if len(context1) == 0 && len(context2) == 0 {
		return 1.0
	}
	if len(context1) == 0 || len(context2) == 0 {
		return 0.0
	}

	matches := 0
	total := 0

	for key, val1 := range context1 {
		total++
		if val2, exists := context2[key]; exists {
			if compareValues(val1, val2) {
				matches++
			}
		}
	}

	// Also count keys in context2 that aren't in context1
	for key := range context2 {
		if _, exists := context1[key]; !exists {
			total++
		}
	}

	if total == 0 {
		return 0.0
	}

	return float64(matches) / float64(total)
}

// jaccardSimilarity calculates Jaccard similarity coefficient
func jaccardSimilarity(set1, set2 []string) float64 {
	if len(set1) == 0 && len(set2) == 0 {
		return 1.0
	}

	// Convert to maps for faster lookup
	map1 := make(map[string]bool)
	map2 := make(map[string]bool)

	for _, item := range set1 {
		map1[strings.ToLower(item)] = true
	}
	for _, item := range set2 {
		map2[strings.ToLower(item)] = true
	}

	// Calculate intersection
	intersection := 0
	for item := range map1 {
		if map2[item] {
			intersection++
		}
	}

	// Calculate union
	union := len(map1) + len(map2) - intersection

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// cosineSimilarity calculates cosine similarity between two word frequency maps
func cosineSimilarity(freq1, freq2 map[string]int) float64 {
	if len(freq1) == 0 || len(freq2) == 0 {
		return 0.0
	}

	// Calculate dot product
	var dotProduct float64
	for word, count1 := range freq1 {
		if count2, exists := freq2[word]; exists {
			dotProduct += float64(count1 * count2)
		}
	}

	// Calculate magnitudes
	var mag1, mag2 float64
	for _, count := range freq1 {
		mag1 += float64(count * count)
	}
	for _, count := range freq2 {
		mag2 += float64(count * count)
	}

	mag1 = math.Sqrt(mag1)
	mag2 = math.Sqrt(mag2)

	if mag1 == 0 || mag2 == 0 {
		return 0.0
	}

	return dotProduct / (mag1 * mag2)
}

// normalizeText normalizes text for comparison
func normalizeText(text string) string {
	// Convert to lowercase
	text = strings.ToLower(text)

	// Remove extra whitespace
	text = strings.TrimSpace(text)

	// Remove special characters except spaces
	var result strings.Builder
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// extractWords extracts unique words from text
func extractWords(text string) []string {
	words := strings.Fields(text)
	unique := make(map[string]bool)

	for _, word := range words {
		word = strings.ToLower(strings.TrimSpace(word))
		if len(word) > 2 { // Ignore very short words
			unique[word] = true
		}
	}

	result := make([]string, 0, len(unique))
	for word := range unique {
		result = append(result, word)
	}

	return result
}

// extractLabels extracts label names from GitHub issue
func extractLabels(issue *github.Issue) []string {
	labels := make([]string, len(issue.Labels))
	for i, label := range issue.Labels {
		labels[i] = label.Name
	}
	return labels
}

// identifyMatches identifies what matched between task and historical task
func identifyMatches(task *Task, historical *HistoricalTask) []string {
	var matches []string

	// Check title similarity
	titleSim := calculateTextSimilarity(task.Title, historical.Issue.Title)
	if titleSim > 0.5 {
		matches = append(matches, "title")
	}

	// Check description similarity
	if task.Description != "" && historical.Issue.Body != "" {
		descSim := calculateTextSimilarity(task.Description, historical.Issue.Body)
		if descSim > 0.3 {
			matches = append(matches, "description")
		}
	}

	// Check label matches
	labelSim := calculateLabelSimilarity(task.Labels, extractLabels(historical.Issue))
	if labelSim > 0.3 {
		matches = append(matches, "labels")
	}

	// Check custom field matches
	if len(task.Context) > 0 && len(historical.Issue.CustomFields) > 0 {
		contextSim := calculateContextSimilarity(task.Context, historical.Issue.CustomFields)
		if contextSim > 0.3 {
			matches = append(matches, "custom_fields")
		}
	}

	return matches
}

// compareValues compares two values for equality
func compareValues(v1, v2 interface{}) bool {
	// Simple string comparison
	s1 := strings.ToLower(strings.TrimSpace(toString(v1)))
	s2 := strings.ToLower(strings.TrimSpace(toString(v2)))

	return s1 == s2
}

// toString converts interface{} to string
func toString(v interface{}) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case int, int64, float64:
		return strings.TrimSpace(strings.ToLower(fmt.Sprintf("%v", val)))
	default:
		return strings.TrimSpace(strings.ToLower(fmt.Sprintf("%v", val)))
	}
}

// sortBySimilarity sorts matches by similarity score (descending)
func sortBySimilarity(matches []*SimilarityMatch) {
	// Simple bubble sort (fine for small lists)
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].Similarity > matches[i].Similarity {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}
}
