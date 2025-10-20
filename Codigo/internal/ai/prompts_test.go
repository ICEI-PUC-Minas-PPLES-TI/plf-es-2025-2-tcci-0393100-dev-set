package ai

import (
	"strings"
	"testing"
)

func TestSystemPrompt(t *testing.T) {
	if SystemPrompt == "" {
		t.Error("SystemPrompt should not be empty")
	}

	// Verify key components are present
	requiredPhrases := []string{
		"expert software estimation assistant",
		"software development tasks",
		"JSON object",
		"estimated_hours",
		"estimated_size",
		"story_points",
		"confidence_score",
	}

	for _, phrase := range requiredPhrases {
		if !strings.Contains(SystemPrompt, phrase) {
			t.Errorf("SystemPrompt should contain '%s'", phrase)
		}
	}
}

func TestBuildEstimationPrompt(t *testing.T) {
	tests := []struct {
		name    string
		request *EstimationRequest
		checks  []string
	}{
		{
			name: "basic task",
			request: &EstimationRequest{
				TaskTitle:       "Add user authentication",
				TaskDescription: "Implement OAuth 2.0",
			},
			checks: []string{
				"Add user authentication",
				"Implement OAuth 2.0",
			},
		},
		{
			name: "with similar tasks",
			request: &EstimationRequest{
				TaskTitle:       "Fix bug",
				TaskDescription: "Memory leak",
				SimilarTasks: []SimilarTask{
					{
						Title:         "Previous bug fix",
						ActualHours:   4.0,
						EstimatedSize: "S",
						Similarity:    0.85,
					},
				},
			},
			checks: []string{
				"Fix bug",
				"Memory leak",
				"Previous bug fix",
				"4.0",
			},
		},
		{
			name: "with dataset stats",
			request: &EstimationRequest{
				TaskTitle:       "Refactor code",
				TaskDescription: "Clean up",
				DatasetStats: &DatasetStats{
					TotalTasks:  100,
					ClosedTasks: 80,
					AvgHours:    8.5,
				},
			},
			checks: []string{
				"Refactor code",
				"100",
				"80",
				"8.5",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := BuildEstimationPrompt(tt.request)

			if prompt == "" {
				t.Error("Prompt should not be empty")
			}

			for _, check := range tt.checks {
				if !strings.Contains(prompt, check) {
					t.Errorf("Prompt should contain '%s'", check)
				}
			}
		})
	}
}

func TestBuildEstimationPrompt_EmptyRequest(t *testing.T) {
	request := &EstimationRequest{}
	prompt := BuildEstimationPrompt(request)

	// Should still generate a prompt even with empty request
	if prompt == "" {
		t.Error("Prompt should not be empty even with empty request")
	}
}

func TestBuildEstimationPrompt_WithContext(t *testing.T) {
	request := &EstimationRequest{
		TaskTitle:       "Test task",
		TaskDescription: "Description",
		Context: map[string]interface{}{
			"priority":    "high",
			"complexity":  "medium",
			"team_size":   5,
		},
	}

	prompt := BuildEstimationPrompt(request)

	// Context should be included in the prompt
	if !strings.Contains(prompt, "Test task") {
		t.Error("Prompt should contain task title")
	}
}

func TestBuildEstimationPrompt_WithSimilarityMeta(t *testing.T) {
	request := &EstimationRequest{
		TaskTitle:       "New feature",
		TaskDescription: "Add functionality",
		SimilarityContext: &SimilarityMeta{
			ThresholdUsed:     0.3,
			HighestSimilarity: 0.95,
			MatchesFound:      0, // No matches found
		},
	}

	prompt := BuildEstimationPrompt(request)

	if !strings.Contains(prompt, "New feature") {
		t.Error("Prompt should contain task title")
	}

	// Check if similarity context message is included when no matches are found
	if !strings.Contains(prompt, "No sufficiently similar tasks found") || !strings.Contains(prompt, "95%") {
		t.Error("Prompt should include similarity context message with highest similarity percentage")
	}
}

func TestFormatEstimationForDisplay(t *testing.T) {
	estimation := &EstimationResponse{
		EstimatedHours:  8.0,
		EstimatedSize:   "M",
		StoryPoints:     5.0,
		ConfidenceScore: 0.75,
		Reasoning:       "Based on historical data",
		Assumptions:     []string{"Team has experience", "No blockers"},
		Risks:           []string{"Dependencies", "Integration complexity"},
	}

	output := FormatEstimationForDisplay(estimation)

	if output == "" {
		t.Error("Output should not be empty")
	}

	// Check for key components
	requiredContent := []string{
		"8.0",
		"M",
		"5",
		"75%",
		"Based on historical data",
		"Team has experience",
		"Dependencies",
	}

	for _, content := range requiredContent {
		if !strings.Contains(output, content) {
			t.Errorf("Output should contain '%s'", content)
		}
	}
}

func TestFormatEstimationForDisplay_MinimalData(t *testing.T) {
	estimation := &EstimationResponse{
		EstimatedHours:  2.0,
		EstimatedSize:   "S",
		StoryPoints:     1.0,
		ConfidenceScore: 0.5,
		Reasoning:       "Minimal task",
	}

	output := FormatEstimationForDisplay(estimation)

	if output == "" {
		t.Error("Output should not be empty")
	}

	if !strings.Contains(output, "2.0") {
		t.Error("Output should contain estimated hours")
	}

	if !strings.Contains(output, "S") {
		t.Error("Output should contain estimated size")
	}
}

func TestFormatEstimationForDisplay_NoAssumptionsOrRisks(t *testing.T) {
	estimation := &EstimationResponse{
		EstimatedHours:  4.0,
		EstimatedSize:   "M",
		StoryPoints:     3.0,
		ConfidenceScore: 0.8,
		Reasoning:       "Test",
		Assumptions:     []string{},
		Risks:           []string{},
	}

	output := FormatEstimationForDisplay(estimation)

	if output == "" {
		t.Error("Output should not be empty")
	}

	// Should still display basic information
	if !strings.Contains(output, "4.0") {
		t.Error("Output should contain estimated hours")
	}
}

func TestFormatEstimationForDisplay_NilEstimation(t *testing.T) {
	output := FormatEstimationForDisplay(nil)

	// Should handle nil gracefully
	if output == "" {
		// This is acceptable - could return empty string or error message
		t.Log("Nil estimation returns empty output")
	}
}

func TestEstimationRequest(t *testing.T) {
	req := &EstimationRequest{
		TaskTitle:       "Test Task",
		TaskDescription: "Test Description",
		SimilarTasks: []SimilarTask{
			{
				Title:         "Similar 1",
				ActualHours:   5.0,
				EstimatedSize: "M",
				Similarity:    0.8,
			},
		},
		Context: map[string]interface{}{
			"key": "value",
		},
		DatasetStats: &DatasetStats{
			TotalTasks:  10,
			ClosedTasks: 8,
		},
		SimilarityContext: &SimilarityMeta{
			ThresholdUsed: 0.3,
			MatchesFound:  100,
		},
	}

	if req.TaskTitle != "Test Task" {
		t.Error("TaskTitle not set correctly")
	}

	if len(req.SimilarTasks) != 1 {
		t.Error("SimilarTasks not set correctly")
	}

	if req.Context["key"] != "value" {
		t.Error("Context not set correctly")
	}

	if req.DatasetStats.TotalTasks != 10 {
		t.Error("DatasetStats not set correctly")
	}
}

func TestSimilarTask(t *testing.T) {
	task := SimilarTask{
		Title:         "Test Similar Task",
		Description:   "Description",
		ActualHours:   8.0,
		EstimatedSize: "L",
		StoryPoints:   8.0,
		Labels:        []string{"bug", "priority"},
		Similarity:    0.92,
	}

	if task.Title != "Test Similar Task" {
		t.Error("Title not set correctly")
	}

	if task.ActualHours != 8.0 {
		t.Error("ActualHours not set correctly")
	}

	if task.Similarity != 0.92 {
		t.Error("Similarity not set correctly")
	}

	if len(task.Labels) != 2 {
		t.Error("Labels not set correctly")
	}
}

func TestDatasetStats(t *testing.T) {
	stats := &DatasetStats{
		TotalTasks:      100,
		ClosedTasks:     85,
		AvgHours:        8.5,
		MedianHours:     7.0,
		TasksBySize:     map[string]int{"S": 20, "M": 50, "L": 30},
		TasksByCategory: map[string]int{"bug": 40, "feature": 60},
	}

	if stats.TotalTasks != 100 {
		t.Error("TotalTasks not set correctly")
	}

	if stats.ClosedTasks != 85 {
		t.Error("ClosedTasks not set correctly")
	}

	if stats.AvgHours != 8.5 {
		t.Error("AvgHours not set correctly")
	}

	if len(stats.TasksBySize) != 3 {
		t.Error("TasksBySize not set correctly")
	}
}

func TestSimilarityMeta(t *testing.T) {
	meta := &SimilarityMeta{
		ThresholdUsed:     0.3,
		HighestSimilarity: 0.95,
		MatchesFound:      15,
	}

	if meta.ThresholdUsed != 0.3 {
		t.Error("ThresholdUsed not set correctly")
	}

	if meta.HighestSimilarity != 0.95 {
		t.Error("HighestSimilarity not set correctly")
	}

	if meta.MatchesFound != 15 {
		t.Error("MatchesFound not set correctly")
	}
}

func TestBuildQuickEstimationPrompt(t *testing.T) {
	prompt := BuildQuickEstimationPrompt("Fix bug", "Memory leak in auth")

	if !strings.Contains(prompt, "Fix bug") {
		t.Error("Prompt should contain task title")
	}

	if !strings.Contains(prompt, "Memory leak in auth") {
		t.Error("Prompt should contain description")
	}

	if !strings.Contains(prompt, "estimated_hours") {
		t.Error("Prompt should mention required fields")
	}

	if !strings.Contains(prompt, "JSON object") {
		t.Error("Prompt should mention JSON format")
	}
}

func TestBuildQuickEstimationPrompt_NoDescription(t *testing.T) {
	prompt := BuildQuickEstimationPrompt("Simple task", "")

	if !strings.Contains(prompt, "Simple task") {
		t.Error("Prompt should contain task title")
	}
}

func TestParseEstimationJSON_Valid(t *testing.T) {
	jsonStr := `{
		"estimated_hours": 8.0,
		"estimated_size": "M",
		"story_points": 5,
		"confidence_score": 0.75,
		"reasoning": "Test reasoning",
		"assumptions": ["assumption1"],
		"risks": ["risk1"]
	}`

	resp, err := ParseEstimationJSON(jsonStr)

	if err != nil {
		t.Fatalf("ParseEstimationJSON failed: %v", err)
	}

	if resp.EstimatedHours != 8.0 {
		t.Errorf("Expected 8.0 hours, got %.1f", resp.EstimatedHours)
	}

	if resp.EstimatedSize != "M" {
		t.Errorf("Expected size M, got %s", resp.EstimatedSize)
	}

	if resp.ConfidenceScore != 0.75 {
		t.Errorf("Expected confidence 0.75, got %.2f", resp.ConfidenceScore)
	}
}

func TestParseEstimationJSON_WithMarkdown(t *testing.T) {
	jsonStr := "```json\n" + `{
		"estimated_hours": 4.0,
		"estimated_size": "S",
		"story_points": 3,
		"confidence_score": 0.8,
		"reasoning": "Quick task"
	}` + "\n```"

	resp, err := ParseEstimationJSON(jsonStr)

	if err != nil {
		t.Fatalf("ParseEstimationJSON with markdown failed: %v", err)
	}

	if resp.EstimatedHours != 4.0 {
		t.Errorf("Expected 4.0 hours, got %.1f", resp.EstimatedHours)
	}
}

func TestParseEstimationJSON_WithSurroundingText(t *testing.T) {
	jsonStr := `Here is the estimation:
	{"estimated_hours": 6.0, "estimated_size": "M", "story_points": 5, "confidence_score": 0.7, "reasoning": "Test"}
	That's my estimate.`

	resp, err := ParseEstimationJSON(jsonStr)

	if err != nil {
		t.Fatalf("ParseEstimationJSON with surrounding text failed: %v", err)
	}

	if resp.EstimatedHours != 6.0 {
		t.Errorf("Expected 6.0 hours, got %.1f", resp.EstimatedHours)
	}
}

func TestParseEstimationJSON_InvalidJSON(t *testing.T) {
	jsonStr := `{"invalid": "json", "missing": "fields"`

	_, err := ParseEstimationJSON(jsonStr)

	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestParseEstimationJSON_InvalidValues(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{
			name: "negative hours",
			json: `{"estimated_hours": -5.0, "estimated_size": "M", "story_points": 5, "confidence_score": 0.7, "reasoning": "Test"}`,
		},
		{
			name: "invalid confidence",
			json: `{"estimated_hours": 5.0, "estimated_size": "M", "story_points": 5, "confidence_score": 1.5, "reasoning": "Test"}`,
		},
		{
			name: "invalid size",
			json: `{"estimated_hours": 5.0, "estimated_size": "XXL", "story_points": 5, "confidence_score": 0.7, "reasoning": "Test"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseEstimationJSON(tt.json)
			if err == nil {
				t.Errorf("Expected error for %s", tt.name)
			}
		})
	}
}

func TestExtractJSONFromMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain JSON",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "markdown code block",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "markdown code block no language",
			input:    "```\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON with surrounding text",
			input:    "Here is data: {\"key\": \"value\"} end",
			expected: `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONFromMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestValidateEstimationResponse(t *testing.T) {
	tests := []struct {
		name    string
		resp    *EstimationResponse
		wantErr bool
	}{
		{
			name: "valid response",
			resp: &EstimationResponse{
				EstimatedHours:  8.0,
				EstimatedSize:   "M",
				StoryPoints:     5,
				ConfidenceScore: 0.75,
				Reasoning:       "This is a valid reasoning",
			},
			wantErr: false,
		},
		{
			name: "negative hours",
			resp: &EstimationResponse{
				EstimatedHours:  -1.0,
				EstimatedSize:   "M",
				StoryPoints:     5,
				ConfidenceScore: 0.75,
			},
			wantErr: true,
		},
		{
			name: "invalid confidence high",
			resp: &EstimationResponse{
				EstimatedHours:  8.0,
				EstimatedSize:   "M",
				StoryPoints:     5,
				ConfidenceScore: 1.5,
			},
			wantErr: true,
		},
		{
			name: "invalid confidence low",
			resp: &EstimationResponse{
				EstimatedHours:  8.0,
				EstimatedSize:   "M",
				StoryPoints:     5,
				ConfidenceScore: -0.1,
			},
			wantErr: true,
		},
		{
			name: "invalid size",
			resp: &EstimationResponse{
				EstimatedHours:  8.0,
				EstimatedSize:   "HUGE",
				StoryPoints:     5,
				ConfidenceScore: 0.75,
				Reasoning:       "Test",
			},
			wantErr: true,
		},
		{
			name: "missing reasoning",
			resp: &EstimationResponse{
				EstimatedHours:  8.0,
				EstimatedSize:   "M",
				StoryPoints:     5,
				ConfidenceScore: 0.75,
				Reasoning:       "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEstimationResponse(tt.resp)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEstimationResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWrapText(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		width  int
		minLen int // minimum expected length
	}{
		{
			name:   "short text",
			text:   "hello",
			width:  10,
			minLen: 5,
		},
		{
			name:   "long text",
			text:   "this is a very long sentence that should be wrapped",
			width:  20,
			minLen: 20,
		},
		{
			name:   "empty text",
			text:   "",
			width:  10,
			minLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wrapText(tt.text, tt.width)
			if len(result) < tt.minLen {
				t.Errorf("Result too short: expected at least %d, got %d", tt.minLen, len(result))
			}
		})
	}
}

func TestSizeToHours(t *testing.T) {
	tests := []struct {
		name    string
		size    Size
		wantMin float64
		wantMax float64
	}{
		{"XS", SizeXS, 0.5, 2.0},
		{"S", SizeS, 2.0, 4.0},
		{"M", SizeM, 4.0, 8.0},
		{"L", SizeL, 8.0, 16.0},
		{"XL", SizeXL, 16.0, 40.0},
		{"Invalid", Size("invalid"), 0.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			min, max := SizeToHours(tt.size)
			if min != tt.wantMin || max != tt.wantMax {
				t.Errorf("SizeToHours(%v) = (%v, %v), want (%v, %v)",
					tt.size, min, max, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestHoursToSize(t *testing.T) {
	tests := []struct {
		name  string
		hours float64
		want  Size
	}{
		{"0.5 hours", 0.5, SizeXS},
		{"1.9 hours", 1.9, SizeXS},
		{"2.5 hours", 2.5, SizeS},
		{"3.9 hours", 3.9, SizeS},
		{"5.0 hours", 5.0, SizeM},
		{"7.9 hours", 7.9, SizeM},
		{"10.0 hours", 10.0, SizeL},
		{"15.9 hours", 15.9, SizeL},
		{"20.0 hours", 20.0, SizeXL},
		{"100.0 hours", 100.0, SizeXL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HoursToSize(tt.hours)
			if got != tt.want {
				t.Errorf("HoursToSize(%v) = %v, want %v", tt.hours, got, tt.want)
			}
		})
	}
}
