package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SystemPrompt is the system message that defines the AI's role
const SystemPrompt = `You are an expert software estimation assistant. Your role is to provide accurate time and effort estimates for software development tasks.

You should:
1. Analyze the task description carefully
2. Consider similar historical tasks if provided
3. Factor in complexity, risks, and dependencies
4. Provide estimates in multiple formats (hours, story points, size)
5. Explain your reasoning clearly
6. Highlight key assumptions and potential risks
7. Be realistic and slightly conservative in estimates

Response Format:
Provide your response as a valid JSON object with this structure:
{
  "estimated_hours": <number>,
  "estimated_size": "<XS|S|M|L|XL>",
  "story_points": <number>,
  "confidence_score": <0.0 to 1.0>,
  "reasoning": "<explanation>",
  "assumptions": ["<assumption 1>", "<assumption 2>"],
  "risks": ["<risk 1>", "<risk 2>"],
  "recommended_action": "<optional recommendation>"
}

Size Guide:
- XS (< 2 hours): Trivial changes, small fixes
- S (2-4 hours): Simple features, minor refactoring
- M (4-8 hours): Standard features, moderate complexity
- L (8-16 hours): Complex features, significant refactoring
- XL (16+ hours): Very complex, should be broken down

Story Points Guide (Fibonacci scale):
- 1 point: XS tasks
- 2 points: S tasks
- 3 points: Small-Medium tasks
- 5 points: M tasks
- 8 points: Large M or small L tasks
- 13 points: L tasks
- 21 points: XL tasks (consider breaking down)`

// BuildEstimationPrompt creates a user prompt for task estimation
func BuildEstimationPrompt(req *EstimationRequest) string {
	var prompt strings.Builder

	// Task information
	prompt.WriteString("## Task to Estimate\n\n")
	prompt.WriteString(fmt.Sprintf("**Title:** %s\n\n", req.TaskTitle))

	if req.TaskDescription != "" {
		prompt.WriteString(fmt.Sprintf("**Description:**\n%s\n\n", req.TaskDescription))
	}

	// Context information
	if len(req.Context) > 0 {
		prompt.WriteString("## Additional Context\n\n")
		for key, value := range req.Context {
			prompt.WriteString(fmt.Sprintf("- **%s:** %v\n", key, value))
		}
		prompt.WriteString("\n")
	}

	// Dataset statistics (provides general context)
	if req.DatasetStats != nil {
		prompt.WriteString("## Historical Dataset Overview\n\n")
		prompt.WriteString(fmt.Sprintf("- **Total historical tasks:** %d (%d completed)\n", req.DatasetStats.TotalTasks, req.DatasetStats.ClosedTasks))

		if req.DatasetStats.AvgHours > 0 {
			prompt.WriteString(fmt.Sprintf("- **Average task duration:** %.1f hours (median: %.1f hours)\n", req.DatasetStats.AvgHours, req.DatasetStats.MedianHours))
		}

		// Percentile distribution
		if len(req.DatasetStats.PercentileHours) == 5 {
			prompt.WriteString(fmt.Sprintf("- **Effort percentiles:** 10th=%.1fh, 25th=%.1fh, 50th=%.1fh, 75th=%.1fh, 90th=%.1fh\n",
				req.DatasetStats.PercentileHours[0],
				req.DatasetStats.PercentileHours[1],
				req.DatasetStats.PercentileHours[2],
				req.DatasetStats.PercentileHours[3],
				req.DatasetStats.PercentileHours[4]))
		}

		if len(req.DatasetStats.TasksBySize) > 0 {
			prompt.WriteString(fmt.Sprintf("- **Size distribution:** "))
			sizeOrder := []string{"XS", "S", "M", "L", "XL"}
			sizeStrs := []string{}
			for _, size := range sizeOrder {
				if count, exists := req.DatasetStats.TasksBySize[size]; exists && count > 0 {
					sizeStrs = append(sizeStrs, fmt.Sprintf("%s=%d", size, count))
				}
			}
			prompt.WriteString(strings.Join(sizeStrs, ", ") + "\n")
		}

		// Category breakdown with effort ranges
		if len(req.DatasetStats.CategoryBreakdown) > 0 {
			prompt.WriteString("\n### Category Breakdown (Effort Patterns)\n\n")
			for category, stats := range req.DatasetStats.CategoryBreakdown {
				prompt.WriteString(fmt.Sprintf("**%s** (%d tasks):\n", category, stats.Count))
				prompt.WriteString(fmt.Sprintf("  - Average: %.1f hours\n", stats.AvgHours))
				prompt.WriteString(fmt.Sprintf("  - Range: %.1f - %.1f hours\n", stats.MinHours, stats.MaxHours))
				if len(stats.TaskTitles) > 0 {
					prompt.WriteString(fmt.Sprintf("  - Examples: %s\n", strings.Join(stats.TaskTitles, ", ")))
				}
				prompt.WriteString("\n")
			}
		}

		prompt.WriteString("\n")
	}

	// Similar tasks and diverse samples (enhanced dataset)
	if len(req.SimilarTasks) > 0 {
		prompt.WriteString("## Reference Tasks (Enhanced Dataset)\n\n")
		if req.SimilarityContext != nil {
			prompt.WriteString(fmt.Sprintf("*Dataset includes: %d similar matches (%.0f%% threshold) + stratified samples + percentile samples*\n",
				req.SimilarityContext.MatchesFound,
				req.SimilarityContext.ThresholdUsed*100))
			prompt.WriteString(fmt.Sprintf("*Highest similarity: %.0f%%*\n\n", req.SimilarityContext.HighestSimilarity*100))
		}
		prompt.WriteString("Use these tasks as reference for your estimate. Tasks are ordered by relevance:\n\n")

		for i, task := range req.SimilarTasks {
			// Determine task type
			taskType := "Similar"
			if task.Similarity == 0.0 {
				taskType = "Sample" // Stratified or percentile sample
			}

			prompt.WriteString(fmt.Sprintf("### Task %d [%s", i+1, taskType))
			if task.Similarity > 0 {
				prompt.WriteString(fmt.Sprintf(": %.0f%% match", task.Similarity*100))
			}
			prompt.WriteString("]\n")

			prompt.WriteString(fmt.Sprintf("**Title:** %s\n", task.Title))

			if task.Description != "" {
				prompt.WriteString(fmt.Sprintf("**Description:** %s\n", task.Description))
			}

			if task.ActualHours > 0 {
				prompt.WriteString(fmt.Sprintf("**Actual Time:** %.1f hours\n", task.ActualHours))
			}

			if task.EstimatedSize != "" {
				prompt.WriteString(fmt.Sprintf("**Size:** %s\n", task.EstimatedSize))
			}

			if task.StoryPoints > 0 {
				prompt.WriteString(fmt.Sprintf("**Story Points:** %.0f\n", task.StoryPoints))
			}

			if len(task.Labels) > 0 {
				prompt.WriteString(fmt.Sprintf("**Labels:** %s\n", strings.Join(task.Labels, ", ")))
			}

			if len(task.CustomFields) > 0 {
				prompt.WriteString("**Custom Fields:**\n")
				for k, v := range task.CustomFields {
					prompt.WriteString(fmt.Sprintf("  - %s: %v\n", k, v))
				}
			}

			prompt.WriteString("\n")
		}
	} else if req.SimilarityContext != nil && req.SimilarityContext.HighestSimilarity > 0 {
		// No matches above threshold, but provide context
		prompt.WriteString("## Similarity Search Results\n\n")
		prompt.WriteString(fmt.Sprintf("*No sufficiently similar tasks found. Highest similarity was %.0f%%.*\n", req.SimilarityContext.HighestSimilarity*100))
		prompt.WriteString("Consider the general dataset statistics and category breakdowns above for context.\n\n")
	}

	// Instructions
	prompt.WriteString("## Instructions\n\n")
	prompt.WriteString("Based on the task description and historical data above:\n")
	prompt.WriteString("1. Provide a realistic estimate in hours, size, and story points\n")
	prompt.WriteString("2. Explain your reasoning considering task complexity\n")
	prompt.WriteString("3. List key assumptions you're making\n")
	prompt.WriteString("4. Identify potential risks or unknowns\n")
	prompt.WriteString("5. Suggest any recommended actions\n\n")
	prompt.WriteString("Respond with a valid JSON object following the specified format.\n")

	return prompt.String()
}

// BuildQuickEstimationPrompt creates a simplified prompt for quick estimates
func BuildQuickEstimationPrompt(title, description string) string {
	var prompt strings.Builder

	prompt.WriteString("Estimate this software development task:\n\n")
	prompt.WriteString(fmt.Sprintf("**Title:** %s\n", title))

	if description != "" {
		prompt.WriteString(fmt.Sprintf("**Description:** %s\n", description))
	}

	prompt.WriteString("\nProvide your estimate as a JSON object with:\n")
	prompt.WriteString("- estimated_hours (number)\n")
	prompt.WriteString("- estimated_size (XS/S/M/L/XL)\n")
	prompt.WriteString("- story_points (number)\n")
	prompt.WriteString("- confidence_score (0.0-1.0)\n")
	prompt.WriteString("- reasoning (string)\n")

	return prompt.String()
}

// ParseEstimationJSON parses the AI response into EstimationResponse
func ParseEstimationJSON(jsonStr string) (*EstimationResponse, error) {
	// Try to extract JSON from markdown code blocks if present
	jsonStr = extractJSONFromMarkdown(jsonStr)

	var response EstimationResponse
	if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
		return nil, fmt.Errorf("failed to parse estimation response: %w", err)
	}

	// Validate the response
	if err := validateEstimationResponse(&response); err != nil {
		return nil, fmt.Errorf("invalid estimation response: %w", err)
	}

	return &response, nil
}

// extractJSONFromMarkdown extracts JSON from markdown code blocks or finds JSON in text
func extractJSONFromMarkdown(text string) string {
	// Remove markdown code blocks if present
	text = strings.TrimSpace(text)

	// Check for ```json ... ``` or ``` ... ```
	if strings.HasPrefix(text, "```") {
		lines := strings.Split(text, "\n")
		if len(lines) > 2 {
			// Remove first line (```json or ```)
			lines = lines[1:]
			// Find the closing ```
			for i, line := range lines {
				if strings.TrimSpace(line) == "```" {
					lines = lines[:i]
					break
				}
			}
			text = strings.Join(lines, "\n")
		}
	}

	// Try to find JSON object if there's surrounding text
	startIdx := strings.Index(text, "{")
	if startIdx != -1 {
		// Find the matching closing brace
		braceCount := 0
		for i := startIdx; i < len(text); i++ {
			if text[i] == '{' {
				braceCount++
			} else if text[i] == '}' {
				braceCount--
				if braceCount == 0 {
					// Found complete JSON object
					text = text[startIdx : i+1]
					break
				}
			}
		}
	}

	return strings.TrimSpace(text)
}

// validateEstimationResponse validates the estimation response
func validateEstimationResponse(resp *EstimationResponse) error {
	if resp.EstimatedHours < 0 {
		return fmt.Errorf("estimated_hours must be non-negative")
	}

	if resp.StoryPoints < 0 {
		return fmt.Errorf("story_points must be non-negative")
	}

	if resp.ConfidenceScore < 0 || resp.ConfidenceScore > 1 {
		return fmt.Errorf("confidence_score must be between 0.0 and 1.0")
	}

	validSizes := map[string]bool{
		"XS": true, "S": true, "M": true, "L": true, "XL": true,
	}
	if resp.EstimatedSize != "" && !validSizes[resp.EstimatedSize] {
		return fmt.Errorf("estimated_size must be one of: XS, S, M, L, XL")
	}

	if resp.Reasoning == "" {
		return fmt.Errorf("reasoning is required")
	}

	return nil
}

// FormatEstimationForDisplay formats an estimation response for CLI display
func FormatEstimationForDisplay(resp *EstimationResponse) string {
	if resp == nil {
		return ""
	}

	var output strings.Builder

	// Header
	output.WriteString("╭─ Estimation ──────────────────────────────────────────────╮\n")
	output.WriteString("│                                                           │\n")

	// Metrics with progress bar for confidence
	confPercent := int(resp.ConfidenceScore * 100)
	confBar := generateProgressBar(confPercent, 10)

	// Box width: 60 chars total, │ on each side = 58 chars inside
	// Content area: 2 spaces at start + "Label" + spaces + value + padding = 57 chars (leaving 1 for │)
	// Format: │  Label       Value...padding...                          │
	// Total inside: 2 + 12 (label padded) + 45 (value+padding) = 59 - but we start with │ and end with │ = 61 total

	// Actually: │  Label       Value..............................        │
	// Start │ = 1, spaces = 2, label area = 12, value area = 45, end │ = 1 = 61 total

	timeStr := fmt.Sprintf("%.1fh", resp.EstimatedHours)
	confStr := fmt.Sprintf("%d%% %s", confPercent, confBar)

	output.WriteString(fmt.Sprintf("│  Time       %-45s  │\n", timeStr))
	output.WriteString(fmt.Sprintf("│  Size       %-45s  │\n", resp.EstimatedSize))
	output.WriteString(fmt.Sprintf("│  Points     %-45.0f  │\n", resp.StoryPoints))
	output.WriteString(fmt.Sprintf("│  Conf       %-45s  │\n", confStr))
	output.WriteString("│                                                           │\n")
	output.WriteString("╰───────────────────────────────────────────────────────────╯\n\n")

	// Analysis
	output.WriteString("┃ Analysis\n\n")
	output.WriteString("  " + wrapText(resp.Reasoning, 58) + "\n\n")

	// Assumptions
	if len(resp.Assumptions) > 0 {
		output.WriteString("┃ Assumptions\n\n")
		for _, assumption := range resp.Assumptions {
			output.WriteString(fmt.Sprintf("  · %s\n", assumption))
		}
		output.WriteString("\n")
	}

	// Risks
	if len(resp.Risks) > 0 {
		output.WriteString("┃ Risks\n\n")
		for _, risk := range resp.Risks {
			output.WriteString(fmt.Sprintf("  ⚡ %s\n", risk))
		}
		output.WriteString("\n")
	}

	// Recommendation
	if resp.RecommendedAction != "" {
		output.WriteString("┃ Recommendation\n\n")
		output.WriteString("  " + wrapText(resp.RecommendedAction, 58) + "\n\n")
	}

	output.WriteString(strings.Repeat("─", 60) + "\n")

	return output.String()
}

// generateProgressBar creates a visual progress bar
func generateProgressBar(percent int, width int) string {
	filled := (percent * width) / 100
	if filled > width {
		filled = width
	}
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return bar
}

// wrapText wraps text to a specified width
func wrapText(text string, width int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	var lines []string
	var currentLine string

	for _, word := range words {
		if len(currentLine)+len(word)+1 > width {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		} else {
			if currentLine == "" {
				currentLine = word
			} else {
				currentLine += " " + word
			}
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return strings.Join(lines, "\n  ")
}
