package ai

import "time"

// EstimationRequest represents a request for task estimation
type EstimationRequest struct {
	TaskTitle         string                 `json:"task_title"`
	TaskDescription   string                 `json:"task_description"`
	SimilarTasks      []SimilarTask          `json:"similar_tasks,omitempty"`
	Context           map[string]interface{} `json:"context,omitempty"` // Custom fields, labels, etc.
	DatasetStats      *DatasetStats          `json:"dataset_stats,omitempty"`
	SimilarityContext *SimilarityMeta        `json:"similarity_context,omitempty"`
}

// DatasetStats provides statistical context about the historical dataset
type DatasetStats struct {
	TotalTasks        int                       `json:"total_tasks"`
	ClosedTasks       int                       `json:"closed_tasks"`
	AvgHours          float64                   `json:"avg_hours"`
	MedianHours       float64                   `json:"median_hours"`
	TasksBySize       map[string]int            `json:"tasks_by_size"`
	TasksByCategory   map[string]int            `json:"tasks_by_category"`
	CategoryBreakdown map[string]*CategoryStats `json:"category_breakdown,omitempty"`
	PercentileHours   []float64                 `json:"percentile_hours,omitempty"`
}

// CategoryStats provides detailed statistics for a category
type CategoryStats struct {
	Count      int      `json:"count"`
	AvgHours   float64  `json:"avg_hours"`
	MinHours   float64  `json:"min_hours"`
	MaxHours   float64  `json:"max_hours"`
	TaskTitles []string `json:"task_titles,omitempty"`
}

// SimilarityMeta provides metadata about the similarity search
type SimilarityMeta struct {
	ThresholdUsed     float64 `json:"threshold_used"`
	HighestSimilarity float64 `json:"highest_similarity"`
	MatchesFound      int     `json:"matches_found"`
}

// SimilarTask represents a historical task for context
type SimilarTask struct {
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	ActualHours   float64                `json:"actual_hours,omitempty"`
	EstimatedSize string                 `json:"estimated_size,omitempty"` // S, M, L, XL
	StoryPoints   float64                `json:"story_points,omitempty"`
	Labels        []string               `json:"labels,omitempty"`
	CustomFields  map[string]interface{} `json:"custom_fields,omitempty"`
	Similarity    float64                `json:"similarity"` // 0.0 to 1.0
}

// EstimationResponse represents the AI's estimation response
type EstimationResponse struct {
	EstimatedHours    float64  `json:"estimated_hours"`
	EstimatedSize     string   `json:"estimated_size"` // S, M, L, XL
	StoryPoints       float64  `json:"story_points"`
	ConfidenceScore   float64  `json:"confidence_score"` // 0.0 to 1.0
	Reasoning         string   `json:"reasoning"`
	Assumptions       []string `json:"assumptions"`
	Risks             []string `json:"risks"`
	RecommendedAction string   `json:"recommended_action,omitempty"`
}

// OpenAIRequest represents a request to OpenAI API
type OpenAIRequest struct {
	Model               string          `json:"model"`
	Messages            []OpenAIMessage `json:"messages"`
	Temperature         float64         `json:"temperature,omitempty"`
	MaxTokens           int             `json:"max_tokens,omitempty"`
	MaxCompletionTokens int             `json:"max_completion_tokens,omitempty"`
	ResponseFormat      *ResponseFormat `json:"response_format,omitempty"`
}

// OpenAIMessage represents a message in the conversation
type OpenAIMessage struct {
	Role    string `json:"role"` // system, user, assistant
	Content string `json:"content"`
}

// ResponseFormat specifies the response format
type ResponseFormat struct {
	Type string `json:"type"` // "json_object" for JSON mode
}

// OpenAIResponse represents a response from OpenAI API
type OpenAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *OpenAIError `json:"error,omitempty"`
}

// OpenAIError represents an error from OpenAI API
type OpenAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

// AIProvider represents an AI service provider
type AIProvider interface {
	// EstimateTask estimates a task and returns the estimation
	EstimateTask(request *EstimationRequest) (*EstimationResponse, error)

	// GetName returns the provider name
	GetName() string

	// IsAvailable checks if the provider is configured and available
	IsAvailable() bool
}

// UsageMetrics tracks API usage
type UsageMetrics struct {
	Provider        string
	RequestCount    int
	TotalTokens     int
	TotalCost       float64
	LastRequestTime time.Time
	AverageTokens   float64
	ErrorCount      int
}

// Size represents task size estimates
type Size string

const (
	SizeXS Size = "XS" // Extra Small: < 2 hours
	SizeS  Size = "S"  // Small: 2-4 hours
	SizeM  Size = "M"  // Medium: 4-8 hours
	SizeL  Size = "L"  // Large: 8-16 hours
	SizeXL Size = "XL" // Extra Large: 16+ hours
)

// SizeToHours converts size to approximate hours
func SizeToHours(size Size) (min, max float64) {
	switch size {
	case SizeXS:
		return 0.5, 2.0
	case SizeS:
		return 2.0, 4.0
	case SizeM:
		return 4.0, 8.0
	case SizeL:
		return 8.0, 16.0
	case SizeXL:
		return 16.0, 40.0
	default:
		return 0, 0
	}
}

// HoursToSize converts hours to size
func HoursToSize(hours float64) Size {
	switch {
	case hours < 2:
		return SizeXS
	case hours < 4:
		return SizeS
	case hours < 8:
		return SizeM
	case hours < 16:
		return SizeL
	default:
		return SizeXL
	}
}
