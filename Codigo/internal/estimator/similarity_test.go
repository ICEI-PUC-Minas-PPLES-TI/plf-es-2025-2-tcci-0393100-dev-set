package estimator

import (
	"math"
	"testing"

	"set/internal/github"
)

func TestCalculateTextSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		text1    string
		text2    string
		expected float64
		minScore float64
	}{
		{
			name:     "identical texts",
			text1:    "Add user authentication",
			text2:    "Add user authentication",
			expected: 1.0,
			minScore: 0.99,
		},
		{
			name:     "similar texts",
			text1:    "Add user authentication with OAuth",
			text2:    "Add user authentication",
			expected: 0.75,
			minScore: 0.6,
		},
		{
			name:     "different texts",
			text1:    "Add user authentication",
			text2:    "Fix database migration bug",
			expected: 0.0,
			minScore: 0.0,
		},
		{
			name:     "empty texts",
			text1:    "",
			text2:    "",
			expected: 0.0,
			minScore: 0.0,
		},
		{
			name:     "one empty",
			text1:    "Add user authentication",
			text2:    "",
			expected: 0.0,
			minScore: 0.0,
		},
		{
			name:     "case insensitive",
			text1:    "Add User Authentication",
			text2:    "add user authentication",
			expected: 1.0,
			minScore: 0.99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateTextSimilarity(tt.text1, tt.text2)
			if score < tt.minScore {
				t.Errorf("calculateTextSimilarity() = %v, want >= %v", score, tt.minScore)
			}
		})
	}
}

func TestCalculateLabelSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		labels1  []string
		labels2  []string
		expected float64
	}{
		{
			name:     "identical labels",
			labels1:  []string{"bug", "high-priority"},
			labels2:  []string{"bug", "high-priority"},
			expected: 1.0,
		},
		{
			name:     "partial match",
			labels1:  []string{"bug", "high-priority"},
			labels2:  []string{"bug", "medium-priority"},
			expected: 0.33,
		},
		{
			name:     "no match",
			labels1:  []string{"bug", "frontend"},
			labels2:  []string{"feature", "backend"},
			expected: 0.0,
		},
		{
			name:     "both empty",
			labels1:  []string{},
			labels2:  []string{},
			expected: 1.0,
		},
		{
			name:     "one empty",
			labels1:  []string{"bug"},
			labels2:  []string{},
			expected: 0.0,
		},
		{
			name:     "case insensitive",
			labels1:  []string{"Bug", "High-Priority"},
			labels2:  []string{"bug", "high-priority"},
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateLabelSimilarity(tt.labels1, tt.labels2)
			// Allow some floating point tolerance
			if score < tt.expected-0.1 || score > tt.expected+0.1 {
				t.Errorf("calculateLabelSimilarity() = %v, want ~%v", score, tt.expected)
			}
		})
	}
}

func TestJaccardSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		set1     []string
		set2     []string
		expected float64
	}{
		{
			name:     "identical sets",
			set1:     []string{"a", "b", "c"},
			set2:     []string{"a", "b", "c"},
			expected: 1.0,
		},
		{
			name:     "partial overlap",
			set1:     []string{"a", "b", "c"},
			set2:     []string{"b", "c", "d"},
			expected: 0.5,
		},
		{
			name:     "no overlap",
			set1:     []string{"a", "b"},
			set2:     []string{"c", "d"},
			expected: 0.0,
		},
		{
			name:     "both empty",
			set1:     []string{},
			set2:     []string{},
			expected: 1.0,
		},
		{
			name:     "single element match",
			set1:     []string{"a"},
			set2:     []string{"a"},
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := jaccardSimilarity(tt.set1, tt.set2)
			if score != tt.expected {
				t.Errorf("jaccardSimilarity() = %v, want %v", score, tt.expected)
			}
		})
	}
}

func TestCalculateSimilarity(t *testing.T) {
	tests := []struct {
		name       string
		task       *Task
		historical *HistoricalTask
		minScore   float64
		maxScore   float64
	}{
		{
			name: "identical task",
			task: &Task{
				Title:       "Add user authentication",
				Description: "Implement OAuth 2.0",
				Labels:      []string{"feature", "security"},
			},
			historical: &HistoricalTask{
				Issue: &github.Issue{
					Title: "Add user authentication",
					Body:  "Implement OAuth 2.0",
					Labels: []github.Label{
						{Name: "feature"},
						{Name: "security"},
					},
				},
			},
			minScore: 0.9,
			maxScore: 1.0,
		},
		{
			name: "similar task",
			task: &Task{
				Title:       "Add user login",
				Description: "OAuth authentication",
				Labels:      []string{"feature"},
			},
			historical: &HistoricalTask{
				Issue: &github.Issue{
					Title: "Add user authentication",
					Body:  "Implement OAuth 2.0",
					Labels: []github.Label{
						{Name: "feature"},
						{Name: "security"},
					},
				},
			},
			minScore: 0.2,
			maxScore: 0.7,
		},
		{
			name: "different task",
			task: &Task{
				Title:  "Fix database bug",
				Labels: []string{"bug"},
			},
			historical: &HistoricalTask{
				Issue: &github.Issue{
					Title: "Add user authentication",
					Labels: []github.Label{
						{Name: "feature"},
					},
				},
			},
			minScore: 0.0,
			maxScore: 0.3,
		},
		{
			name: "nil historical issue",
			task: &Task{
				Title: "Add feature",
			},
			historical: &HistoricalTask{
				Issue: nil,
			},
			minScore: 0.0,
			maxScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateSimilarity(tt.task, tt.historical)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("CalculateSimilarity() = %v, want between %v and %v", score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestFindSimilarTasks(t *testing.T) {
	historical := []*HistoricalTask{
		{
			Issue: &github.Issue{
				Number: 1,
				Title:  "Add user authentication",
				Body:   "Implement OAuth 2.0 login",
				Labels: []github.Label{
					{Name: "feature"},
					{Name: "security"},
				},
			},
			ActualHours: 8.0,
		},
		{
			Issue: &github.Issue{
				Number: 2,
				Title:  "Add user login page",
				Body:   "Create login form",
				Labels: []github.Label{
					{Name: "feature"},
					{Name: "frontend"},
				},
			},
			ActualHours: 4.0,
		},
		{
			Issue: &github.Issue{
				Number: 3,
				Title:  "Fix database migration",
				Body:   "Migration failing in production",
				Labels: []github.Label{
					{Name: "bug"},
					{Name: "database"},
				},
			},
			ActualHours: 2.0,
		},
	}

	tests := []struct {
		name             string
		task             *Task
		minMatches       int
		maxMatches       int
		minTopSimilarity float64
		config           *EstimationConfig
	}{
		{
			name: "find authentication tasks",
			task: &Task{
				Title:       "Add OAuth authentication",
				Description: "Implement OAuth 2.0",
				Labels:      []string{"feature", "security"},
			},
			minMatches:       1,
			maxMatches:       2,
			minTopSimilarity: 0.4,
			config:           DefaultEstimationConfig(),
		},
		{
			name: "find login tasks",
			task: &Task{
				Title:  "Add user login",
				Labels: []string{"feature"},
			},
			minMatches:       2,
			maxMatches:       2,
			minTopSimilarity: 0.3,
			config:           DefaultEstimationConfig(),
		},
		{
			name: "find bug tasks",
			task: &Task{
				Title:  "Fix database issue",
				Labels: []string{"bug"},
			},
			minMatches:       1,
			maxMatches:       1,
			minTopSimilarity: 0.3,
			config:           DefaultEstimationConfig(),
		},
		{
			name: "limit max results",
			task: &Task{
				Title:  "Add feature",
				Labels: []string{"feature"},
			},
			minMatches: 1,
			maxMatches: 2,
			config: &EstimationConfig{
				MaxSimilarTasks:        2,
				MinSimilarityThreshold: 0.2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := FindSimilarTasks(tt.task, historical, tt.config)

			if len(matches) < tt.minMatches {
				t.Errorf("FindSimilarTasks() returned %d matches, want at least %d", len(matches), tt.minMatches)
			}

			if len(matches) > tt.maxMatches {
				t.Errorf("FindSimilarTasks() returned %d matches, want at most %d", len(matches), tt.maxMatches)
			}

			if len(matches) > 0 {
				// Check that results are sorted by similarity
				for i := 1; i < len(matches); i++ {
					if matches[i].Similarity > matches[i-1].Similarity {
						t.Errorf("Results not sorted by similarity: [%d]=%v > [%d]=%v",
							i, matches[i].Similarity, i-1, matches[i-1].Similarity)
					}
				}

				// Check minimum similarity threshold
				if matches[0].Similarity < tt.minTopSimilarity {
					t.Errorf("Top match similarity = %v, want >= %v",
						matches[0].Similarity, tt.minTopSimilarity)
				}
			}
		})
	}
}

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase conversion",
			input:    "Add User Authentication",
			expected: "add user authentication",
		},
		{
			name:     "remove special characters",
			input:    "Add user auth! @#$ %^&",
			expected: "add user auth  ",
		},
		{
			name:     "preserve numbers",
			input:    "Fix bug in OAuth2.0",
			expected: "fix bug in oauth20",
		},
		{
			name:     "trim whitespace",
			input:    "  Add feature  ",
			expected: "add feature",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeText(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeText() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractWords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		minWords int
		contains []string
	}{
		{
			name:     "simple text",
			input:    "add user authentication",
			minWords: 3,
			contains: []string{"add", "user", "authentication"},
		},
		{
			name:     "ignore short words",
			input:    "a an the add user",
			minWords: 2,
			contains: []string{"add", "user"},
		},
		{
			name:     "unique words only",
			input:    "add add user user authentication",
			minWords: 3,
			contains: []string{"add", "user", "authentication"},
		},
		{
			name:     "empty string",
			input:    "",
			minWords: 0,
			contains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			words := extractWords(tt.input)
			if len(words) < tt.minWords {
				t.Errorf("extractWords() returned %d words, want at least %d", len(words), tt.minWords)
			}

			// Check that expected words are present
			wordMap := make(map[string]bool)
			for _, w := range words {
				wordMap[w] = true
			}

			for _, expectedWord := range tt.contains {
				if !wordMap[expectedWord] {
					t.Errorf("extractWords() missing expected word %q", expectedWord)
				}
			}
		})
	}
}

func TestCalculateConfidence(t *testing.T) {
	tests := []struct {
		name       string
		similar    []*SimilarityMatch
		dataPoints int
		minConf    float64
		maxConf    float64
	}{
		{
			name: "high confidence - many similar tasks",
			similar: []*SimilarityMatch{
				{Similarity: 0.9},
				{Similarity: 0.85},
				{Similarity: 0.8},
			},
			dataPoints: 3,
			minConf:    0.7,
			maxConf:    0.9,
		},
		{
			name: "medium confidence - moderate similarity",
			similar: []*SimilarityMatch{
				{Similarity: 0.6},
				{Similarity: 0.5},
			},
			dataPoints: 2,
			minConf:    0.4,
			maxConf:    0.7,
		},
		{
			name: "low confidence - low similarity",
			similar: []*SimilarityMatch{
				{Similarity: 0.3},
			},
			dataPoints: 1,
			minConf:    0.3,
			maxConf:    0.5,
		},
		{
			name:       "no data",
			similar:    []*SimilarityMatch{},
			dataPoints: 0,
			minConf:    0.3,
			maxConf:    0.3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := calculateConfidence(tt.similar, tt.dataPoints)

			if confidence < tt.minConf || confidence > tt.maxConf {
				t.Errorf("calculateConfidence() = %v, want between %v and %v",
					confidence, tt.minConf, tt.maxConf)
			}

			// Confidence should always be capped between 0.3 and 0.9
			if confidence < 0.3 || confidence > 0.9 {
				t.Errorf("calculateConfidence() = %v, should be between 0.3 and 0.9", confidence)
			}
		})
	}
}

func TestCalculateContextSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		context1 map[string]interface{}
		context2 map[string]interface{}
		expected float64
	}{
		{
			name:     "both empty",
			context1: map[string]interface{}{},
			context2: map[string]interface{}{},
			expected: 1.0,
		},
		{
			name:     "one empty",
			context1: map[string]interface{}{"key": "value"},
			context2: map[string]interface{}{},
			expected: 0.0,
		},
		{
			name:     "identical",
			context1: map[string]interface{}{"key": "value", "num": 42},
			context2: map[string]interface{}{"key": "value", "num": 42},
			expected: 1.0,
		},
		{
			name:     "partial match",
			context1: map[string]interface{}{"key1": "value1", "key2": "value2"},
			context2: map[string]interface{}{"key1": "value1", "key3": "value3"},
			expected: 0.33, // 1 match out of 3 total keys
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateContextSimilarity(tt.context1, tt.context2)
			if math.Abs(result-tt.expected) > 0.01 {
				t.Errorf("calculateContextSimilarity() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		freq1    map[string]int
		freq2    map[string]int
		expected float64
	}{
		{
			name:     "empty maps",
			freq1:    map[string]int{},
			freq2:    map[string]int{},
			expected: 0.0,
		},
		{
			name:     "one empty",
			freq1:    map[string]int{"word": 1},
			freq2:    map[string]int{},
			expected: 0.0,
		},
		{
			name:     "identical",
			freq1:    map[string]int{"hello": 2, "world": 1},
			freq2:    map[string]int{"hello": 2, "world": 1},
			expected: 1.0,
		},
		{
			name:     "partial overlap",
			freq1:    map[string]int{"hello": 1, "world": 1},
			freq2:    map[string]int{"hello": 1, "test": 1},
			expected: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.freq1, tt.freq2)
			if math.Abs(result-tt.expected) > 0.01 {
				t.Errorf("cosineSimilarity() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCompareValues(t *testing.T) {
	tests := []struct {
		name     string
		v1       interface{}
		v2       interface{}
		expected bool
	}{
		{
			name:     "equal strings",
			v1:       "test",
			v2:       "test",
			expected: true,
		},
		{
			name:     "case insensitive",
			v1:       "Test",
			v2:       "TEST",
			expected: true,
		},
		{
			name:     "whitespace trimmed",
			v1:       "  test  ",
			v2:       "test",
			expected: true,
		},
		{
			name:     "different strings",
			v1:       "hello",
			v2:       "world",
			expected: false,
		},
		{
			name:     "equal numbers",
			v1:       42,
			v2:       42,
			expected: true,
		},
		{
			name:     "nil values",
			v1:       nil,
			v2:       nil,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareValues(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("compareValues(%v, %v) = %v, want %v", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{
			name:     "nil",
			value:    nil,
			expected: "",
		},
		{
			name:     "string",
			value:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "int",
			value:    42,
			expected: "42",
		},
		{
			name:     "float",
			value:    3.14,
			expected: "3.14",
		},
		{
			name:     "bool",
			value:    true,
			expected: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toString(tt.value)
			if result != tt.expected {
				t.Errorf("toString(%v) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}
