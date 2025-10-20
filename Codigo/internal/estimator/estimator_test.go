package estimator

import (
	"context"
	"testing"
	"time"

	"set/internal/ai"
)

// Mock AI provider for testing
type mockAIProvider struct {
	available bool
	response  *ai.EstimationResponse
	err       error
}

func (m *mockAIProvider) EstimateTask(request *ai.EstimationRequest) (*ai.EstimationResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *mockAIProvider) GetName() string {
	return "mock"
}

func (m *mockAIProvider) IsAvailable() bool {
	return m.available
}

func TestNewEstimator(t *testing.T) {
	tests := []struct {
		name     string
		provider ai.AIProvider
		config   *EstimationConfig
		wantNil  bool
	}{
		{
			name:     "with config",
			provider: &mockAIProvider{available: true},
			config:   DefaultEstimationConfig(),
			wantNil:  false,
		},
		{
			name:     "nil config uses default",
			provider: &mockAIProvider{available: true},
			config:   nil,
			wantNil:  false,
		},
		{
			name:     "nil provider",
			provider: nil,
			config:   DefaultEstimationConfig(),
			wantNil:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			est := NewEstimator(tt.provider, nil, tt.config)
			if (est == nil) != tt.wantNil {
				t.Errorf("NewEstimator() = %v, wantNil %v", est, tt.wantNil)
			}
			if est != nil && est.config == nil {
				t.Error("NewEstimator() returned estimator with nil config")
			}
		})
	}
}

func TestEstimateWithAI(t *testing.T) {
	tests := []struct {
		name       string
		provider   *mockAIProvider
		task       *Task
		similar    []*SimilarityMatch
		wantErr    bool
		wantResult bool
	}{
		{
			name: "successful AI estimation",
			provider: &mockAIProvider{
				available: true,
				response: &ai.EstimationResponse{
					EstimatedHours:  10.0,
					EstimatedSize:   "L",
					StoryPoints:     8.0,
					ConfidenceScore: 0.85,
					Reasoning:       "AI-based estimate",
				},
			},
			task: &Task{
				Title:       "Add feature",
				Description: "New feature",
			},
			similar:    []*SimilarityMatch{},
			wantErr:    false,
			wantResult: true,
		},
		{
			name: "AI error",
			provider: &mockAIProvider{
				available: true,
				err:       context.DeadlineExceeded,
			},
			task: &Task{
				Title: "Add feature",
			},
			similar:    []*SimilarityMatch{},
			wantErr:    true,
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			est := NewEstimator(tt.provider, nil, DefaultEstimationConfig())

			// Create mock stats and context for testing
			mockStats := &DatasetStatistics{
				TotalTasks:  100,
				ClosedTasks: 80,
				AvgHours:    10.0,
				MedianHours: 8.0,
			}
			mockContext := &SimilarityContext{
				ThresholdUsed:     0.3,
				HighestSimilarity: 0.8,
				MatchesFound:      len(tt.similar),
			}

			result, err := est.estimateWithAI(tt.task, tt.similar, mockStats, mockContext)

			if (err != nil) != tt.wantErr {
				t.Errorf("estimateWithAI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if (result != nil) != tt.wantResult {
				t.Errorf("estimateWithAI() result = %v, wantResult %v", result, tt.wantResult)
			}

			if result != nil {
				if result.EstimatedHours <= 0 {
					t.Error("estimateWithAI() should return positive hours")
				}
				if result.EstimatedSize == "" {
					t.Error("estimateWithAI() should return size")
				}
			}
		})
	}
}

func TestEstimate(t *testing.T) {
	tests := []struct {
		name       string
		provider   ai.AIProvider
		config     *EstimationConfig
		task       *Task
		wantMethod string
		wantErr    bool
	}{
		{
			name: "AI estimation with historical data",
			provider: &mockAIProvider{
				available: true,
				response: &ai.EstimationResponse{
					EstimatedHours:  8.0,
					EstimatedSize:   "M",
					StoryPoints:     5.0,
					ConfidenceScore: 0.8,
					Reasoning:       "AI estimate based on enhanced dataset",
				},
			},
			config: &EstimationConfig{
				MaxSimilarTasks:          15,
				MinSimilarTasks:          10,
				MinSimilarityThreshold:   0.3,
				StratifiedSamplesPerSize: 2,
			},
			task: &Task{
				Title:       "Add user authentication",
				Description: "OAuth 2.0",
			},
			wantMethod: "ai",
			wantErr:    false,
		},
		{
			name:     "no AI provider (should error)",
			provider: nil,
			config:   DefaultEstimationConfig(),
			task: &Task{
				Title: "Unknown task",
			},
			wantMethod: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			est := NewEstimator(tt.provider, nil, tt.config)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, err := est.Estimate(ctx, tt.task)

			if (err != nil) != tt.wantErr {
				t.Errorf("Estimate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				// Expected error, test passed
				return
			}

			if result == nil {
				t.Fatal("Estimate() returned nil result")
			}

			if result.Method != tt.wantMethod {
				t.Errorf("Estimate() method = %v, want %v", result.Method, tt.wantMethod)
			}

			if result.Estimation == nil {
				t.Error("Estimate() returned nil estimation")
			}

			if result.Task != tt.task {
				t.Error("Estimate() result doesn't reference original task")
			}
		})
	}
}

func TestExtractFloat(t *testing.T) {
	tests := []struct {
		name     string
		fields   map[string]interface{}
		keys     []string
		expected float64
	}{
		{
			name: "float64 value",
			fields: map[string]interface{}{
				"Hours": 8.5,
			},
			keys:     []string{"Hours"},
			expected: 8.5,
		},
		{
			name: "int value",
			fields: map[string]interface{}{
				"Points": 5,
			},
			keys:     []string{"Points"},
			expected: 5.0,
		},
		{
			name: "string value",
			fields: map[string]interface{}{
				"Hours": "7.5",
			},
			keys:     []string{"Hours"},
			expected: 7.5,
		},
		{
			name: "multiple keys",
			fields: map[string]interface{}{
				"Worker Hours": 8.0,
			},
			keys:     []string{"Hours", "Worker Hours", "Time"},
			expected: 8.0,
		},
		{
			name:     "not found",
			fields:   map[string]interface{}{},
			keys:     []string{"Hours"},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFloat(tt.fields, tt.keys...)
			if result != tt.expected {
				t.Errorf("extractFloat() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractString(t *testing.T) {
	tests := []struct {
		name     string
		fields   map[string]interface{}
		keys     []string
		expected string
	}{
		{
			name: "string value",
			fields: map[string]interface{}{
				"Size": "M",
			},
			keys:     []string{"Size"},
			expected: "M",
		},
		{
			name: "multiple keys",
			fields: map[string]interface{}{
				"Complexity": "High",
			},
			keys:     []string{"Size", "Complexity"},
			expected: "High",
		},
		{
			name:     "not found",
			fields:   map[string]interface{}{},
			keys:     []string{"Size"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractString(tt.fields, tt.keys...)
			if result != tt.expected {
				t.Errorf("extractString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMostCommon(t *testing.T) {
	tests := []struct {
		name          string
		items         []string
		expected      string
		validResults  []string // For non-deterministic cases like ties
	}{
		{
			name:     "single most common",
			items:    []string{"M", "M", "M", "S", "L"},
			expected: "M",
		},
		{
			name:         "tie",
			items:        []string{"M", "M", "S", "S"},
			validResults: []string{"M", "S"}, // Either is acceptable in a tie
		},
		{
			name:     "single item",
			items:    []string{"M"},
			expected: "M",
		},
		{
			name:     "empty",
			items:    []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mostCommon(tt.items)

			if tt.validResults != nil {
				// Check if result is in validResults
				valid := false
				for _, v := range tt.validResults {
					if result == v {
						valid = true
						break
					}
				}
				if !valid {
					t.Errorf("mostCommon() = %v, want one of %v", result, tt.validResults)
				}
			} else if result != tt.expected {
				t.Errorf("mostCommon() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSizeToPoints(t *testing.T) {
	tests := []struct {
		size     ai.Size
		expected float64
	}{
		{ai.SizeXS, 1.0},
		{ai.SizeS, 2.0},
		{ai.SizeM, 5.0},
		{ai.SizeL, 8.0},
		{ai.SizeXL, 13.0},
		{ai.Size("unknown"), 3.0},
	}

	for _, tt := range tests {
		t.Run(string(tt.size), func(t *testing.T) {
			result := sizeToPoints(tt.size)
			if result != tt.expected {
				t.Errorf("sizeToPoints(%v) = %v, want %v", tt.size, result, tt.expected)
			}
		})
	}
}

func TestEstimationResultGetConfidenceLevel(t *testing.T) {
	tests := []struct {
		name       string
		confidence float64
		expected   string
	}{
		{"high confidence", 0.85, "high"},
		{"medium confidence", 0.65, "medium"},
		{"low confidence", 0.35, "low"},
		{"nil estimation", 0.0, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &EstimationResult{
				Task: &Task{Title: "Test"},
			}

			if tt.confidence > 0 {
				result.Estimation = &ai.EstimationResponse{
					ConfidenceScore: tt.confidence,
				}
			}

			level := result.GetConfidenceLevel()
			if level != tt.expected {
				t.Errorf("GetConfidenceLevel() = %v, want %v", level, tt.expected)
			}
		})
	}
}


func TestConfidenceScore(t *testing.T) {
	tests := []struct {
		name     string
		result   *EstimationResult
		expected float64
	}{
		{
			name: "with estimation",
			result: &EstimationResult{
				Estimation: &ai.EstimationResponse{
					ConfidenceScore: 0.85,
				},
			},
			expected: 0.85,
		},
		{
			name: "nil estimation",
			result: &EstimationResult{
				Estimation: nil,
			},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := tt.result.ConfidenceScore()
			if score != tt.expected {
				t.Errorf("ConfidenceScore() = %v, want %v", score, tt.expected)
			}
		})
	}
}
