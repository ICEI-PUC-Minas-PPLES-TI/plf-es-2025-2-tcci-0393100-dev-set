package github

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"set/internal/logger"
)

// FetchPullRequests fetches pull requests from a repository with pagination support
// Returns all pull requests from all pages
func (c *Client) FetchPullRequests(ctx context.Context, owner, repo string, opts *FetchOptions) ([]*PullRequest, error) {
	if opts == nil {
		opts = DefaultFetchOptions()
	}

	var allPRs []*PullRequest
	page := opts.Page

	logger.Infof("Fetching pull requests from %s/%s (state: %s)", owner, repo, opts.State)

	for {
		prs, hasNext, err := c.fetchPullRequestsPage(ctx, owner, repo, opts, page)
		if err != nil {
			return allPRs, fmt.Errorf("failed to fetch pull requests page %d: %w", page, err)
		}

		allPRs = append(allPRs, prs...)

		logger.Infof("Fetched page %d: %d pull requests (total so far: %d)", page, len(prs), len(allPRs))

		if !hasNext {
			break
		}

		page++

		// Check if context was cancelled
		select {
		case <-ctx.Done():
			return allPRs, ctx.Err()
		default:
		}
	}

	logger.Infof("Fetch complete: %d total pull requests", len(allPRs))

	// Enrich with custom fields if requested
	if opts.IncludeCustomFields && len(allPRs) > 0 {
		logger.Infof("Enriching pull requests with GitHub Projects custom fields...")
		if err := c.EnrichPullRequestsWithProjectData(ctx, owner, repo, allPRs); err != nil {
			logger.Warnf("Failed to enrich pull requests with custom fields: %v", err)
			// Don't fail the whole operation, just log the warning
		}
	}

	return allPRs, nil
}

// fetchPullRequestsPage fetches a single page of pull requests
func (c *Client) fetchPullRequestsPage(ctx context.Context, owner, repo string, opts *FetchOptions, page int) ([]*PullRequest, bool, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls", c.baseURL, owner, repo)

	// Build query parameters
	params := map[string]string{
		"state":     opts.State,
		"page":      strconv.Itoa(page),
		"per_page":  strconv.Itoa(opts.PerPage),
		"sort":      opts.Sort,
		"direction": opts.Direction,
	}

	// Create request
	req, err := c.buildRequest(ctx, "GET", url, params)
	if err != nil {
		return nil, false, err
	}

	// Perform request
	resp, err := c.doRequestRaw(ctx, req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	// Parse pull requests
	var prs []*PullRequest
	if err := decodeJSON(resp.Body, &prs); err != nil {
		return nil, false, fmt.Errorf("failed to decode pull requests: %w", err)
	}

	// Check for next page
	linkHeader := resp.Header.Get("Link")
	links := parseLinkHeader(linkHeader)
	hasNext := links["next"] != ""

	return prs, hasNext, nil
}

// FetchPullRequest fetches a single pull request by number
func (c *Client) FetchPullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d", c.baseURL, owner, repo, number)

	var pr PullRequest
	if err := c.doRequest(ctx, "GET", url, nil, &pr); err != nil {
		return nil, fmt.Errorf("failed to fetch pull request #%d: %w", number, err)
	}

	return &pr, nil
}

// FetchPullRequestsSince fetches all pull requests updated since a specific time
func (c *Client) FetchPullRequestsSince(ctx context.Context, owner, repo string, since *time.Time) ([]*PullRequest, error) {
	opts := DefaultFetchOptions()
	opts.Since = since

	return c.FetchPullRequests(ctx, owner, repo, opts)
}

// CountPullRequests returns the count of pull requests matching the criteria (lightweight)
func (c *Client) CountPullRequests(ctx context.Context, owner, repo string, state string) (int, error) {
	opts := &FetchOptions{
		State:   state,
		Page:    1,
		PerPage: 1, // Minimal page size
	}

	url := fmt.Sprintf("%s/repos/%s/%s/pulls", c.baseURL, owner, repo)
	params := map[string]string{
		"state":    opts.State,
		"page":     "1",
		"per_page": "1",
	}

	req, err := c.buildRequest(ctx, "GET", url, params)
	if err != nil {
		return 0, err
	}

	resp, err := c.doRequestRaw(ctx, req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// Parse Link header to get total count
	linkHeader := resp.Header.Get("Link")
	links := parseLinkHeader(linkHeader)

	if lastURL := links["last"]; lastURL != "" {
		// Extract page number from last URL
		if page := extractPageFromURL(lastURL); page > 0 {
			// Rough estimate: last_page * per_page
			return page * opts.PerPage, nil
		}
	}

	// If no pagination, count the items in the response
	var prs []*PullRequest
	if err := decodeJSON(resp.Body, &prs); err == nil {
		return len(prs), nil
	}

	return 0, fmt.Errorf("could not determine pull request count")
}
