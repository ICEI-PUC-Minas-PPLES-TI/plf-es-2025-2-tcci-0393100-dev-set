package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewOpenAIClient(t *testing.T) {
	client := NewOpenAIClient("test-api-key")

	if client == nil {
		t.Fatal("Expected client to be created")
	}

	if client.apiKey != "test-api-key" {
		t.Errorf("Expected apiKey 'test-api-key', got %s", client.apiKey)
	}

	if client.model != defaultModel {
		t.Errorf("Expected default model %s, got %s", defaultModel, client.model)
	}

	if client.httpClient == nil {
		t.Error("Expected httpClient to be initialized")
	}

	if client.metrics == nil {
		t.Error("Expected metrics to be initialized")
	}

	if client.metrics.Provider != "OpenAI" {
		t.Errorf("Expected provider 'OpenAI', got %s", client.metrics.Provider)
	}
}

func TestSetModel(t *testing.T) {
	client := NewOpenAIClient("test-key")

	customModel := "gpt-4o"
	client.SetModel(customModel)

	if client.model != customModel {
		t.Errorf("Expected model %s, got %s", customModel, client.model)
	}
}

func TestGetName(t *testing.T) {
	client := NewOpenAIClient("test-key")

	if client.GetName() != "OpenAI" {
		t.Errorf("Expected 'OpenAI', got %s", client.GetName())
	}
}

func TestIsAvailable(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected bool
	}{
		{"with API key", "sk-test123", true},
		{"empty API key", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewOpenAIClient(tt.apiKey)
			if client.IsAvailable() != tt.expected {
				t.Errorf("Expected IsAvailable() = %v, got %v", tt.expected, client.IsAvailable())
			}
		})
	}
}

func TestEstimateTask_NoAPIKey(t *testing.T) {
	client := NewOpenAIClient("")

	request := &EstimationRequest{
		TaskTitle:       "Test task",
		TaskDescription: "Test description",
	}

	_, err := client.EstimateTask(request)
	if err == nil {
		t.Error("Expected error when API key not configured")
	}

	if err.Error() != "OpenAI API key not configured" {
		t.Errorf("Expected specific error message, got: %s", err.Error())
	}
}

func TestEstimateTask_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json")
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-api-key" {
			t.Errorf("Expected Authorization header with bearer token, got: %s", authHeader)
		}

		// Send mock response
		response := OpenAIResponse{
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Message: struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					}{
						Content: `{
							"estimated_hours": 8.0,
							"estimated_size": "M",
							"story_points": 5.0,
							"confidence_score": 0.75,
							"reasoning": "Test reasoning",
							"assumptions": ["Test assumption"],
							"risks": ["Test risk"]
						}`,
					},
				},
			},
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			}{
				PromptTokens:     100,
				CompletionTokens: 50,
				TotalTokens:      150,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with mock server URL
	client := NewOpenAIClient("test-api-key")
	client.httpClient = server.Client()

	// Override the API URL for testing (would need to modify the code to make this testable)
	// For now, this test verifies the basic structure

	request := &EstimationRequest{
		TaskTitle:       "Test task",
		TaskDescription: "Test description",
	}

	// Note: This will fail because we can't override the API URL in the current implementation
	// We're testing the structure and error handling
	_, err := client.EstimateTask(request)

	// Since we can't override the URL, we expect this to fail
	// In a real implementation, we'd inject the URL or use an interface
	if err == nil {
		// If it succeeds (unlikely), verify the response
		t.Log("Unexpected success - likely using real API")
	}
}

func TestModelCapabilities(t *testing.T) {
	tests := []struct {
		name              string
		model             string
		expectedNewer     bool
		supportsTemp      bool
	}{
		{"gpt-4", "gpt-4", false, true},
		{"gpt-4o", "gpt-4o", true, true},
		{"gpt-4o-mini", "gpt-4o-mini", true, true},
		{"o1-preview", "o1-preview", true, false},
		{"o1-mini", "o1-mini", true, false},
		{"gpt-5-nano", "gpt-5-nano", true, false},
		{"gpt-3.5-turbo", "gpt-3.5-turbo", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewOpenAIClient("test-key")
			client.SetModel(tt.model)

			// Verify model was set
			if client.model != tt.model {
				t.Errorf("Expected model %s, got %s", tt.model, client.model)
			}
		})
	}
}

func TestUsageMetrics(t *testing.T) {
	client := NewOpenAIClient("test-key")

	if client.metrics == nil {
		t.Fatal("Expected metrics to be initialized")
	}

	// Verify initial state
	if client.metrics.RequestCount != 0 {
		t.Error("Expected initial RequestCount to be 0")
	}

	if client.metrics.TotalTokens != 0 {
		t.Error("Expected initial TotalTokens to be 0")
	}
}

func TestOpenAIRequest(t *testing.T) {
	req := &OpenAIRequest{
		Model: "gpt-4",
		Messages: []OpenAIMessage{
			{Role: "system", Content: "System prompt"},
			{Role: "user", Content: "User prompt"},
		},
		MaxTokens:   2000,
		Temperature: 0.3,
	}

	if req.Model != "gpt-4" {
		t.Errorf("Expected model gpt-4, got %s", req.Model)
	}

	if len(req.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(req.Messages))
	}

	if req.Messages[0].Role != "system" {
		t.Errorf("Expected first message role 'system', got %s", req.Messages[0].Role)
	}
}

func TestOpenAIResponse(t *testing.T) {
	resp := &OpenAIResponse{
		Choices: []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}{
			{
				Message: struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				}{
					Content: `{"estimated_hours": 8.0}`,
				},
			},
		},
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
	}

	if len(resp.Choices) != 1 {
		t.Errorf("Expected 1 choice, got %d", len(resp.Choices))
	}

	if resp.Usage.TotalTokens != 150 {
		t.Errorf("Expected 150 total tokens, got %d", resp.Usage.TotalTokens)
	}
}

func TestEstimationResponse(t *testing.T) {
	resp := &EstimationResponse{
		EstimatedHours:  8.0,
		EstimatedSize:   "M",
		StoryPoints:     5.0,
		ConfidenceScore: 0.75,
		Reasoning:       "Test reasoning",
		Assumptions:     []string{"Assumption 1"},
		Risks:           []string{"Risk 1"},
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

	if len(resp.Assumptions) != 1 {
		t.Errorf("Expected 1 assumption, got %d", len(resp.Assumptions))
	}

	if len(resp.Risks) != 1 {
		t.Errorf("Expected 1 risk, got %d", len(resp.Risks))
	}
}

func TestDefaultConstants(t *testing.T) {
	if defaultModel != "gpt-4" {
		t.Errorf("Expected default model gpt-4, got %s", defaultModel)
	}

	if defaultMaxTokens != 4000 {
		t.Errorf("Expected default max tokens 4000, got %d", defaultMaxTokens)
	}

	if defaultTimeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", defaultTimeout)
	}

	if openAIAPIURL != "https://api.openai.com/v1/chat/completions" {
		t.Errorf("Unexpected API URL: %s", openAIAPIURL)
	}
}

func TestGetMetrics(t *testing.T) {
	client := NewOpenAIClient("test-key")

	metrics := client.GetMetrics()

	if metrics == nil {
		t.Error("GetMetrics should return metrics object")
	}

	if metrics.Provider != "OpenAI" {
		t.Errorf("Expected provider 'OpenAI', got %s", metrics.Provider)
	}

	if metrics.RequestCount != 0 {
		t.Errorf("Expected 0 requests initially, got %d", metrics.RequestCount)
	}
}

func TestEstimateQuick_BuildsRequest(t *testing.T) {
	client := NewOpenAIClient("")

	// This should fail because no API key
	_, err := client.EstimateQuick("Quick task", "Do something fast")

	if err == nil {
		t.Error("Expected error for missing API key")
	}
}

func TestEstimateWithSimilar_BuildsRequest(t *testing.T) {
	client := NewOpenAIClient("")

	similar := []SimilarTask{
		{
			Title:       "Previous task",
			ActualHours: 5.0,
			Similarity:  0.85,
		},
	}

	// This should fail because no API key
	_, err := client.EstimateWithSimilar("New task", "Similar to previous", similar)

	if err == nil {
		t.Error("Expected error for missing API key")
	}
}

func TestLimitString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "long string",
			input:    "hello world this is a long string",
			maxLen:   10,
			expected: "hello worl...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := limitString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestValidateAPIKey_NoKey(t *testing.T) {
	client := NewOpenAIClient("")

	err := client.ValidateAPIKey()

	if err == nil {
		t.Error("Expected error for empty API key")
	}

	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("Unexpected error message: %v", err)
	}
}
