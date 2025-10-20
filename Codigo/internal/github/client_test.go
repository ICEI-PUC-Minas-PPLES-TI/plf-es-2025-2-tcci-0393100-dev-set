package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	token := "test-token"
	client := NewClient(token)

	assert.NotNil(t, client)
	assert.Equal(t, GitHubAPIBaseURL, client.baseURL)
	assert.Equal(t, token, client.token)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, DefaultTimeout, client.httpClient.Timeout)
	assert.NotNil(t, client.rateLimit)
}

func TestParseLinkHeader(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected map[string]string
	}{
		{
			name:     "Empty header",
			header:   "",
			expected: map[string]string{},
		},
		{
			name:   "Single link",
			header: `<https://api.github.com/repos/test/test/issues?page=2>; rel="next"`,
			expected: map[string]string{
				"next": "https://api.github.com/repos/test/test/issues?page=2",
			},
		},
		{
			name: "Multiple links",
			header: `<https://api.github.com/repos/test/test/issues?page=2>; rel="next", ` +
				`<https://api.github.com/repos/test/test/issues?page=10>; rel="last"`,
			expected: map[string]string{
				"next": "https://api.github.com/repos/test/test/issues?page=2",
				"last": "https://api.github.com/repos/test/test/issues?page=10",
			},
		},
		{
			name: "All pagination links",
			header: `<https://api.github.com/repos/test/test/issues?page=3>; rel="next", ` +
				`<https://api.github.com/repos/test/test/issues?page=10>; rel="last", ` +
				`<https://api.github.com/repos/test/test/issues?page=1>; rel="first", ` +
				`<https://api.github.com/repos/test/test/issues?page=1>; rel="prev"`,
			expected: map[string]string{
				"next":  "https://api.github.com/repos/test/test/issues?page=3",
				"last":  "https://api.github.com/repos/test/test/issues?page=10",
				"first": "https://api.github.com/repos/test/test/issues?page=1",
				"prev":  "https://api.github.com/repos/test/test/issues?page=1",
			},
		},
		{
			name:     "Malformed link",
			header:   `invalid link format`,
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLinkHeader(tt.header)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAddQueryParams(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		params    map[string]string
		expected  string
		expectErr bool
	}{
		{
			name:     "No parameters",
			baseURL:  "https://api.github.com/repos/test/test",
			params:   map[string]string{},
			expected: "https://api.github.com/repos/test/test",
		},
		{
			name:    "Single parameter",
			baseURL: "https://api.github.com/repos/test/test",
			params: map[string]string{
				"state": "open",
			},
			expected: "https://api.github.com/repos/test/test?state=open",
		},
		{
			name:    "Multiple parameters",
			baseURL: "https://api.github.com/repos/test/test/issues",
			params: map[string]string{
				"state": "all",
				"page":  "1",
			},
			// Note: parameter order may vary
		},
		{
			name:      "Invalid URL",
			baseURL:   "://invalid-url",
			params:    map[string]string{"key": "value"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := addQueryParams(tt.baseURL, tt.params)

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			// For tests with expected URL, check exact match
			if tt.expected != "" && len(tt.params) <= 1 {
				assert.Equal(t, tt.expected, result)
			} else if len(tt.params) > 1 {
				// For multiple params, parse and check components
				parsedURL, err := url.Parse(result)
				assert.NoError(t, err)

				query := parsedURL.Query()
				for key, value := range tt.params {
					assert.Equal(t, value, query.Get(key))
				}
			}
		})
	}
}

func TestExtractPageFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected int
	}{
		{
			name:     "URL with page parameter",
			url:      "https://api.github.com/repos/test/test/issues?page=5",
			expected: 5,
		},
		{
			name:     "URL without page parameter",
			url:      "https://api.github.com/repos/test/test/issues",
			expected: 0,
		},
		{
			name:     "URL with multiple parameters",
			url:      "https://api.github.com/repos/test/test/issues?state=all&page=10&per_page=100",
			expected: 10,
		},
		{
			name:     "Invalid URL",
			url:      "://invalid",
			expected: 0,
		},
		{
			name:     "URL with invalid page value",
			url:      "https://api.github.com/repos/test/test/issues?page=notanumber",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPageFromURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUpdateRateLimitFromHeaders(t *testing.T) {
	client := NewClient("test-token")

	headers := http.Header{}
	headers.Set("X-RateLimit-Limit", "5000")
	headers.Set("X-RateLimit-Remaining", "4999")
	headers.Set("X-RateLimit-Reset", "1704067200") // 2024-01-01 00:00:00 UTC

	client.updateRateLimitFromHeaders(headers)

	assert.Equal(t, 5000, client.rateLimit.Limit)
	assert.Equal(t, 4999, client.rateLimit.Remaining)

	expectedReset := time.Unix(1704067200, 0)
	assert.Equal(t, expectedReset, client.rateLimit.Reset)
}

func TestUpdateRateLimitFromHeaders_InvalidValues(t *testing.T) {
	client := NewClient("test-token")

	// Set initial values
	client.rateLimit.Limit = 100
	client.rateLimit.Remaining = 50

	// Headers with invalid values should not update
	headers := http.Header{}
	headers.Set("X-RateLimit-Limit", "invalid")
	headers.Set("X-RateLimit-Remaining", "invalid")
	headers.Set("X-RateLimit-Reset", "invalid")

	client.updateRateLimitFromHeaders(headers)

	// Values should remain unchanged
	assert.Equal(t, 100, client.rateLimit.Limit)
	assert.Equal(t, 50, client.rateLimit.Remaining)
}

func TestUpdateRateLimitFromHeaders_MissingHeaders(t *testing.T) {
	client := NewClient("test-token")

	// Set initial values
	client.rateLimit.Limit = 100
	client.rateLimit.Remaining = 50

	// Empty headers should not update
	headers := http.Header{}
	client.updateRateLimitFromHeaders(headers)

	// Values should remain unchanged
	assert.Equal(t, 100, client.rateLimit.Limit)
	assert.Equal(t, 50, client.rateLimit.Remaining)
}

func TestGetCurrentRateLimit(t *testing.T) {
	client := NewClient("test-token")

	// Set some rate limit values
	client.rateLimit.Limit = 5000
	client.rateLimit.Remaining = 4500
	client.rateLimit.Reset = time.Now().Add(1 * time.Hour)

	rateLimit := client.GetCurrentRateLimit()

	assert.NotNil(t, rateLimit)
	assert.Equal(t, 5000, rateLimit.Limit)
	assert.Equal(t, 4500, rateLimit.Remaining)
}

func TestBuildRequest(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	req, err := client.buildRequest(ctx, "GET", "/test", nil)

	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, "GET", req.Method)
	assert.Contains(t, req.Header.Get("Authorization"), "token test-token")
	assert.Equal(t, "application/vnd.github.v3+json", req.Header.Get("Accept"))
}

func TestCheckRateLimit(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	// Test with plenty of remaining requests
	client.rateLimit.Remaining = 100
	client.rateLimit.Reset = time.Now().Add(1 * time.Hour)

	err := client.checkRateLimit(ctx)
	assert.NoError(t, err)
}

func TestGetRepository_WithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/owner/repo", r.URL.Path)
		assert.Contains(t, r.Header.Get("Authorization"), "token")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": 123,
			"name": "repo",
			"full_name": "owner/repo",
			"description": "Test",
			"private": false,
			"html_url": "https://github.com/owner/repo"
		}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	repo, err := client.GetRepository(ctx, "owner", "repo")

	assert.NoError(t, err)
	assert.NotNil(t, repo)
	assert.Equal(t, int64(123), repo.ID)
	assert.Equal(t, "repo", repo.Name)
}

func TestGetRateLimit_WithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"resources": {
				"core": {
					"limit": 5000,
					"remaining": 4999,
					"reset": "2024-01-01T00:00:00Z"
				}
			}
		}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	rl, err := client.GetRateLimit(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, rl)
	assert.Equal(t, 5000, rl.Limit)
	assert.Equal(t, 4999, rl.Remaining)
}

func TestBuildRequest_WithParams(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	params := map[string]string{
		"per_page": "100",
		"state":    "open",
	}

	req, err := client.buildRequest(ctx, "GET", "/test", params)

	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Contains(t, req.URL.RawQuery, "per_page=100")
	assert.Contains(t, req.URL.RawQuery, "state=open")
}

func TestDoRequestRaw_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "5000")
		w.Header().Set("X-RateLimit-Remaining", "4999")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"test": "data"}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	ctx := context.Background()

	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	resp, err := client.doRequestRaw(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestDoRequestRaw_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Not found"}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	ctx := context.Background()

	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	_, err := client.doRequestRaw(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestDoRequest_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"name": "test"})
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	var result map[string]string
	err := client.doRequest(ctx, "GET", server.URL, nil, &result)

	assert.NoError(t, err)
	assert.Equal(t, "test", result["name"])
}

func TestCheckRateLimit_LowRemaining(t *testing.T) {
	client := NewClient("test-token")

	// Use a canceled context to avoid waiting
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Set very low remaining count
	client.rateLimit.Remaining = 0
	client.rateLimit.Reset = time.Now().Add(1 * time.Hour)

	err := client.checkRateLimit(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestCheckRateLimit_AlreadyReset(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	// Set rate limit that has already reset (in the past)
	client.rateLimit.Remaining = 0
	client.rateLimit.Reset = time.Now().Add(-1 * time.Hour)

	err := client.checkRateLimit(ctx)
	assert.NoError(t, err)
}

func TestDecodeJSON(t *testing.T) {
	jsonStr := `{"id": 123, "name": "test"}`
	reader := strings.NewReader(jsonStr)

	var result map[string]interface{}
	err := decodeJSON(reader, &result)

	assert.NoError(t, err)
	assert.Equal(t, float64(123), result["id"])
	assert.Equal(t, "test", result["name"])
}

func TestDecodeJSON_InvalidJSON(t *testing.T) {
	reader := strings.NewReader(`{invalid json}`)

	var result map[string]interface{}
	err := decodeJSON(reader, &result)

	assert.Error(t, err)
}

func TestClient_UserAgent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userAgent := r.Header.Get("User-Agent")
		assert.Equal(t, "SET-CLI/1.0", userAgent)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	ctx := context.Background()

	req, _ := client.buildRequest(ctx, "GET", server.URL, nil)
	client.doRequestRaw(ctx, req)
}

func TestGetRepository_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	_, err := client.GetRepository(ctx, "owner", "repo")

	assert.Error(t, err)
}

func TestGetRateLimit_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	_, err := client.GetRateLimit(ctx)

	assert.Error(t, err)
}

func TestBuildRequest_InvalidURL(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	params := map[string]string{
		"test": "value",
	}

	_, err := client.buildRequest(ctx, "GET", "://invalid-url", params)

	assert.Error(t, err)
}

func TestDoRequestRaw_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-token")
	ctx, cancel := context.WithCancel(context.Background())

	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL, nil)

	// Cancel context before request
	cancel()

	_, err := client.doRequestRaw(ctx, req)

	assert.Error(t, err)
}
