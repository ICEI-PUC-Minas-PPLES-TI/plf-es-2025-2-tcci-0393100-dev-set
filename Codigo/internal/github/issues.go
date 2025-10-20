package github

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"set/internal/logger"
)

// FetchIssues fetches issues from a repository with pagination support
// Returns all issues (excluding pull requests by default) from all pages
func (c *Client) FetchIssues(ctx context.Context, owner, repo string, opts *FetchOptions) ([]*Issue, error) {
	if opts == nil {
		opts = DefaultFetchOptions()
	}

	var allIssues []*Issue
	page := opts.Page

	logger.Infof("Fetching issues from %s/%s (state: %s)", owner, repo, opts.State)

	for {
		issues, hasNext, err := c.fetchIssuesPage(ctx, owner, repo, opts, page)
		if err != nil {
			return allIssues, fmt.Errorf("failed to fetch issues page %d: %w", page, err)
		}

		// Filter out pull requests (they have the pull_request field set)
		for _, issue := range issues {
			if !issue.IsPullRequest() {
				allIssues = append(allIssues, issue)
			}
		}

		logger.Infof("Fetched page %d: %d issues (total so far: %d)", page, len(issues), len(allIssues))

		if !hasNext {
			break
		}

		page++

		// Check if context was cancelled
		select {
		case <-ctx.Done():
			return allIssues, ctx.Err()
		default:
		}
	}

	logger.Infof("Fetch complete: %d total issues", len(allIssues))

	// Enrich with custom fields if requested
	if opts.IncludeCustomFields && len(allIssues) > 0 {
		logger.Infof("Enriching issues with GitHub Projects custom fields...")
		if err := c.EnrichIssuesWithProjectData(ctx, owner, repo, allIssues); err != nil {
			logger.Warnf("Failed to enrich issues with custom fields: %v", err)
			// Don't fail the whole operation, just log the warning
		}
	}

	return allIssues, nil
}

// fetchIssuesPage fetches a single page of issues
func (c *Client) fetchIssuesPage(ctx context.Context, owner, repo string, opts *FetchOptions, page int) ([]*Issue, bool, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues", c.baseURL, owner, repo)

	// Build query parameters
	params := map[string]string{
		"state":     opts.State,
		"page":      strconv.Itoa(page),
		"per_page":  strconv.Itoa(opts.PerPage),
		"sort":      opts.Sort,
		"direction": opts.Direction,
	}

	// Add since parameter if provided
	if opts.Since != nil {
		params["since"] = opts.Since.Format("2006-01-02T15:04:05Z")
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

	// Parse issues
	var issues []*Issue
	if err := decodeJSON(resp.Body, &issues); err != nil {
		return nil, false, fmt.Errorf("failed to decode issues: %w", err)
	}

	// Check for next page
	linkHeader := resp.Header.Get("Link")
	links := parseLinkHeader(linkHeader)
	hasNext := links["next"] != ""

	return issues, hasNext, nil
}

// FetchIssue fetches a single issue by number
func (c *Client) FetchIssue(ctx context.Context, owner, repo string, number int) (*Issue, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d", c.baseURL, owner, repo, number)

	var issue Issue
	if err := c.doRequest(ctx, "GET", url, nil, &issue); err != nil {
		return nil, fmt.Errorf("failed to fetch issue #%d: %w", number, err)
	}

	return &issue, nil
}

// FetchIssuesSince fetches all issues updated since a specific time
func (c *Client) FetchIssuesSince(ctx context.Context, owner, repo string, since *time.Time) ([]*Issue, error) {
	opts := DefaultFetchOptions()
	opts.Since = since

	return c.FetchIssues(ctx, owner, repo, opts)
}

// CountIssues returns the count of issues matching the criteria (lightweight)
func (c *Client) CountIssues(ctx context.Context, owner, repo string, state string) (int, error) {
	opts := &FetchOptions{
		State:   state,
		Page:    1,
		PerPage: 1, // Minimal page size
	}

	url := fmt.Sprintf("%s/repos/%s/%s/issues", c.baseURL, owner, repo)
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
	var issues []*Issue
	if err := decodeJSON(resp.Body, &issues); err == nil {
		return len(issues), nil
	}

	return 0, fmt.Errorf("could not determine issue count")
}
