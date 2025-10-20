package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"set/internal/logger"
)

const (
	openAIAPIURL     = "https://api.openai.com/v1/chat/completions"
	defaultModel     = "gpt-4"
	defaultMaxTokens = 4000 // Increased from 2000 to handle longer responses
	defaultTimeout   = 30 * time.Second
)

// OpenAIClient implements AIProvider for OpenAI
type OpenAIClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
	metrics    *UsageMetrics
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(apiKey string) *OpenAIClient {
	return &OpenAIClient{
		apiKey: apiKey,
		model:  defaultModel,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		metrics: &UsageMetrics{
			Provider: "OpenAI",
		},
	}
}

// SetModel sets the OpenAI model to use
func (c *OpenAIClient) SetModel(model string) {
	c.model = model
}

// GetName returns the provider name
func (c *OpenAIClient) GetName() string {
	return "OpenAI"
}

// IsAvailable checks if OpenAI is configured
func (c *OpenAIClient) IsAvailable() bool {
	return c.apiKey != ""
}

// EstimateTask estimates a task using OpenAI
func (c *OpenAIClient) EstimateTask(request *EstimationRequest) (*EstimationResponse, error) {
	if !c.IsAvailable() {
		return nil, fmt.Errorf("OpenAI API key not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// Build the prompt
	userPrompt := BuildEstimationPrompt(request)

	// Create OpenAI request
	oaiReq := &OpenAIRequest{
		Model: c.model,
		Messages: []OpenAIMessage{
			{
				Role:    "system",
				Content: SystemPrompt,
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
	}

	// Determine model capabilities
	// Newer models (gpt-4o, gpt-4o-mini, o1-*, gpt-5-*) use max_completion_tokens
	// Older models (gpt-4, gpt-3.5-turbo) use max_tokens
	isNewerModel := c.model == "gpt-4o" || c.model == "gpt-4o-mini" ||
		c.model == "o1-preview" || c.model == "o1-mini" ||
		len(c.model) >= 5 && c.model[:5] == "gpt-5"

	// Some models (like gpt-5-nano) don't support custom temperature
	// o1-* models and some experimental models only support default temperature
	supportsTemperature := !(c.model == "o1-preview" || c.model == "o1-mini" ||
		(len(c.model) >= 9 && c.model[:9] == "gpt-5-nan"))

	if isNewerModel {
		oaiReq.MaxCompletionTokens = defaultMaxTokens
	} else {
		oaiReq.MaxTokens = defaultMaxTokens
	}

	// Only set temperature for models that support it
	if supportsTemperature {
		oaiReq.Temperature = 0.3 // Lower temperature for more consistent estimates
	}
	// Otherwise use default (1.0) by omitting the field

	// Only use JSON mode for models that support it (gpt-4o, gpt-4-turbo, gpt-3.5-turbo-1106+)
	// Standard gpt-4 doesn't support response_format
	// Note: o1-* and gpt-5-* models may have different JSON support
	supportsJSONMode := c.model == "gpt-4o" || c.model == "gpt-4-turbo" || c.model == "gpt-4o-mini" ||
		c.model == "gpt-3.5-turbo-1106" || c.model == "gpt-3.5-turbo"

	if supportsJSONMode {
		oaiReq.ResponseFormat = &ResponseFormat{
			Type: "json_object",
		}
	}

	logger.Infof("Requesting estimation from OpenAI (model: %s)", c.model)

	// Make the API call and track timing
	startTime := time.Now()
	oaiResp, err := c.makeRequest(ctx, oaiReq)
	apiDuration := time.Since(startTime)

	if err != nil {
		c.metrics.ErrorCount++
		logger.Warnf("OpenAI API call failed after %v: %v", apiDuration, err)
		return nil, fmt.Errorf("OpenAI API request failed: %w", err)
	}

	// Log API timing
	logger.Infof("OpenAI API call completed in %v", apiDuration)

	// Update metrics
	c.metrics.RequestCount++
	c.metrics.TotalTokens += oaiResp.Usage.TotalTokens
	c.metrics.LastRequestTime = time.Now()
	if c.metrics.RequestCount > 0 {
		c.metrics.AverageTokens = float64(c.metrics.TotalTokens) / float64(c.metrics.RequestCount)
	}

	// Log token usage
	logger.Infof("Token usage - Prompt: %d, Completion: %d, Total: %d",
		oaiResp.Usage.PromptTokens, oaiResp.Usage.CompletionTokens, oaiResp.Usage.TotalTokens)

	// Extract the response content
	if len(oaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	choice := oaiResp.Choices[0]
	responseContent := choice.Message.Content

	// Log finish reason for debugging
	logger.Infof("Response finish reason: %s, content length: %d chars", choice.FinishReason, len(responseContent))
	logger.Debugf("OpenAI response: %s", responseContent)

	// Parse the estimation response
	estimation, err := ParseEstimationJSON(responseContent)
	if err != nil {
		// Log the raw response for debugging
		logger.Warnf("Failed to parse response. Raw content (first 500 chars): %s",
			limitString(responseContent, 500))
		return nil, fmt.Errorf("failed to parse OpenAI response: %w", err)
	}

	logger.Infof("Estimation complete: %.1f hours, %s, %.0f story points, %.0f%% confidence",
		estimation.EstimatedHours,
		estimation.EstimatedSize,
		estimation.StoryPoints,
		estimation.ConfidenceScore*100)

	return estimation, nil
}

// makeRequest makes an HTTP request to OpenAI API
func (c *OpenAIClient) makeRequest(ctx context.Context, req *OpenAIRequest) (*OpenAIResponse, error) {
	// Serialize request
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", openAIAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Make the request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP errors
	if httpResp.StatusCode != http.StatusOK {
		var errorResp OpenAIResponse
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error != nil {
			return nil, fmt.Errorf("OpenAI API error: %s", errorResp.Error.Message)
		}
		return nil, fmt.Errorf("OpenAI API error: status %d, body: %s", httpResp.StatusCode, string(body))
	}

	// Parse response
	var oaiResp OpenAIResponse
	if err := json.Unmarshal(body, &oaiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &oaiResp, nil
}

// GetMetrics returns usage metrics
func (c *OpenAIClient) GetMetrics() *UsageMetrics {
	return c.metrics
}

// EstimateQuick provides a quick estimate with minimal context
func (c *OpenAIClient) EstimateQuick(title, description string) (*EstimationResponse, error) {
	req := &EstimationRequest{
		TaskTitle:       title,
		TaskDescription: description,
	}
	return c.EstimateTask(req)
}

// EstimateWithSimilar estimates with similar historical tasks
func (c *OpenAIClient) EstimateWithSimilar(title, description string, similar []SimilarTask) (*EstimationResponse, error) {
	req := &EstimationRequest{
		TaskTitle:       title,
		TaskDescription: description,
		SimilarTasks:    similar,
	}
	return c.EstimateTask(req)
}

// limitString limits a string to a maximum length
func limitString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ValidateAPIKey checks if the API key is valid by making a test request
func (c *OpenAIClient) ValidateAPIKey() error {
	if !c.IsAvailable() {
		return fmt.Errorf("API key not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Make a minimal request to test the API key
	req := &OpenAIRequest{
		Model: "gpt-3.5-turbo", // Use cheaper model for validation
		Messages: []OpenAIMessage{
			{
				Role:    "user",
				Content: "Say 'OK' if you can read this.",
			},
		},
		MaxTokens: 10,
	}

	_, err := c.makeRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("API key validation failed: %w", err)
	}

	return nil
}
