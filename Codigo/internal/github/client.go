package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"set/internal/logger"
)

const (
	// GitHubAPIBaseURL is the base URL for GitHub API v3
	GitHubAPIBaseURL = "https://api.github.com"

	// DefaultTimeout for HTTP requests
	DefaultTimeout = 30 * time.Second

	// RateLimitBuffer is the minimum remaining requests before we start slowing down
	RateLimitBuffer = 100
)

// Client is a GitHub API client
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	rateLimit  *RateLimit
}

// NewClient creates a new GitHub API client
func NewClient(token string) *Client {
	return &Client{
		baseURL: GitHubAPIBaseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		rateLimit: &RateLimit{},
	}
}

// GetRepository fetches repository information
func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*Repository, error) {
	url := fmt.Sprintf("%s/repos/%s/%s", c.baseURL, owner, repo)

	var repository Repository
	if err := c.doRequest(ctx, "GET", url, nil, &repository); err != nil {
		return nil, fmt.Errorf("failed to fetch repository: %w", err)
	}

	return &repository, nil
}

// GetRateLimit fetches the current rate limit status
func (c *Client) GetRateLimit(ctx context.Context) (*RateLimit, error) {
	url := fmt.Sprintf("%s/rate_limit", c.baseURL)

	var response struct {
		Resources struct {
			Core RateLimit `json:"core"`
		} `json:"resources"`
	}

	if err := c.doRequest(ctx, "GET", url, nil, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch rate limit: %w", err)
	}

	return &response.Resources.Core, nil
}

// doRequest performs an HTTP request with proper authentication and error handling
func (c *Client) doRequest(ctx context.Context, method, url string, params map[string]string, result interface{}) error {
	// Check rate limit before making request
	if err := c.checkRateLimit(ctx); err != nil {
		return err
	}

	// Build URL with query parameters
	if len(params) > 0 {
		u, err := addQueryParams(url, params)
		if err != nil {
			return fmt.Errorf("failed to build URL: %w", err)
		}
		url = u
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "SET-CLI/1.0")

	// Perform request
	logger.Debugf("GitHub API request: %s %s", method, url)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Update rate limit from response headers
	c.updateRateLimitFromHeaders(resp.Header)

	// Handle non-2xx responses
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Decode response
	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// checkRateLimit checks if we're approaching rate limits and waits if necessary
func (c *Client) checkRateLimit(ctx context.Context) error {
	if c.rateLimit.Remaining <= 0 {
		// We've hit the rate limit
		waitTime := time.Until(c.rateLimit.Reset)
		if waitTime > 0 {
			logger.Warnf("Rate limit exceeded. Waiting %s until reset...", waitTime)
			select {
			case <-time.After(waitTime):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	} else if c.rateLimit.Remaining < RateLimitBuffer {
		// We're getting low, add a small delay
		delay := 500 * time.Millisecond
		logger.Debugf("Rate limit low (%d remaining). Adding %s delay", c.rateLimit.Remaining, delay)
		select {
		case <-time.After(delay):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// updateRateLimitFromHeaders updates the rate limit from response headers
func (c *Client) updateRateLimitFromHeaders(headers http.Header) {
	if limit := headers.Get("X-RateLimit-Limit"); limit != "" {
		if val, err := strconv.Atoi(limit); err == nil {
			c.rateLimit.Limit = val
		}
	}

	if remaining := headers.Get("X-RateLimit-Remaining"); remaining != "" {
		if val, err := strconv.Atoi(remaining); err == nil {
			c.rateLimit.Remaining = val
		}
	}

	if reset := headers.Get("X-RateLimit-Reset"); reset != "" {
		if val, err := strconv.ParseInt(reset, 10, 64); err == nil {
			c.rateLimit.Reset = time.Unix(val, 0)
		}
	}

	logger.Debugf("Rate limit: %d/%d (resets at %s)", c.rateLimit.Remaining, c.rateLimit.Limit, c.rateLimit.Reset.Format(time.RFC3339))
}

// parseLinkHeader parses the Link header for pagination
func parseLinkHeader(linkHeader string) map[string]string {
	links := make(map[string]string)

	if linkHeader == "" {
		return links
	}

	// Split by comma to get individual links
	parts := strings.Split(linkHeader, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)

		// Split by semicolon to separate URL and rel
		segments := strings.Split(part, ";")
		if len(segments) < 2 {
			continue
		}

		// Extract URL (remove angle brackets)
		urlPart := strings.TrimSpace(segments[0])
		urlPart = strings.Trim(urlPart, "<>")

		// Extract rel value
		relPart := strings.TrimSpace(segments[1])
		relPart = strings.TrimPrefix(relPart, `rel="`)
		relPart = strings.TrimSuffix(relPart, `"`)

		links[relPart] = urlPart
	}

	return links
}

// addQueryParams adds query parameters to a URL
func addQueryParams(baseURL string, params map[string]string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	q := u.Query()
	for key, value := range params {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// GetCurrentRateLimit returns the current rate limit state
func (c *Client) GetCurrentRateLimit() *RateLimit {
	return c.rateLimit
}

// buildRequest creates an HTTP request with query parameters
func (c *Client) buildRequest(ctx context.Context, method, url string, params map[string]string) (*http.Request, error) {
	// Build URL with query parameters
	if len(params) > 0 {
		u, err := addQueryParams(url, params)
		if err != nil {
			return nil, fmt.Errorf("failed to build URL: %w", err)
		}
		url = u
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "SET-CLI/1.0")

	return req, nil
}

// doRequestRaw performs an HTTP request and returns the raw response
func (c *Client) doRequestRaw(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Check rate limit before making request
	if err := c.checkRateLimit(ctx); err != nil {
		return nil, err
	}

	// Perform request
	logger.Debugf("GitHub API request: %s %s", req.Method, req.URL.String())
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Update rate limit from response headers
	c.updateRateLimitFromHeaders(resp.Header)

	// Handle non-2xx responses
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// decodeJSON is a helper to decode JSON from a reader
func decodeJSON(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

// extractPageFromURL extracts the page number from a URL
func extractPageFromURL(urlStr string) int {
	u, err := url.Parse(urlStr)
	if err != nil {
		return 0
	}

	pageStr := u.Query().Get("page")
	if pageStr == "" {
		return 0
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil {
		return 0
	}

	return page
}
