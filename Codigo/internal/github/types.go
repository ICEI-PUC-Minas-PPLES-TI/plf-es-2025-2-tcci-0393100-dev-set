package github

import "time"

// Repository represents a GitHub repository
type Repository struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	FullName    string    `json:"full_name"`
	Description string    `json:"description"`
	Private     bool      `json:"private"`
	URL         string    `json:"html_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Issue represents a GitHub issue
type Issue struct {
	ID           int64                  `json:"id"`
	Number       int                    `json:"number"`
	Title        string                 `json:"title"`
	Body         string                 `json:"body"`
	State        string                 `json:"state"` // open, closed
	Labels       []Label                `json:"labels"`
	Assignees    []User                 `json:"assignees"`
	Milestone    *Milestone             `json:"milestone"`
	User         User                   `json:"user"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	ClosedAt     *time.Time             `json:"closed_at"`
	Comments     int                    `json:"comments"`
	URL          string                 `json:"html_url"`
	PullRequest  *struct{}              `json:"pull_request,omitempty"`  // Present if issue is a PR
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"` // GitHub Projects custom fields
}

// PullRequest represents a GitHub pull request
type PullRequest struct {
	ID           int64                  `json:"id"`
	Number       int                    `json:"number"`
	Title        string                 `json:"title"`
	Body         string                 `json:"body"`
	State        string                 `json:"state"` // open, closed, merged
	Labels       []Label                `json:"labels"`
	Assignees    []User                 `json:"assignees"`
	User         User                   `json:"user"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	ClosedAt     *time.Time             `json:"closed_at"`
	MergedAt     *time.Time             `json:"merged_at"`
	Merged       bool                   `json:"merged"`
	Comments     int                    `json:"comments"`
	Commits      int                    `json:"commits"`
	Additions    int                    `json:"additions"`
	Deletions    int                    `json:"deletions"`
	ChangedFiles int                    `json:"changed_files"`
	URL          string                 `json:"html_url"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"` // GitHub Projects custom fields
}

// Label represents a GitHub label
type Label struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

// User represents a GitHub user
type User struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
	URL       string `json:"html_url"`
}

// Milestone represents a GitHub milestone
type Milestone struct {
	ID          int64      `json:"id"`
	Number      int        `json:"number"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	State       string     `json:"state"`
	DueOn       *time.Time `json:"due_on"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ClosedAt    *time.Time `json:"closed_at"`
}

// RateLimit represents GitHub API rate limit information
type RateLimit struct {
	Limit     int       `json:"limit"`
	Remaining int       `json:"remaining"`
	Reset     time.Time `json:"reset"`
}

// FetchOptions contains options for fetching data from GitHub
type FetchOptions struct {
	State               string     // all, open, closed
	Since               *time.Time // Only issues updated at or after this time
	Page                int        // Page number for pagination
	PerPage             int        // Results per page (max 100)
	Sort                string     // created, updated, comments
	Direction           string     // asc, desc
	IncludeCustomFields bool       // Whether to fetch GitHub Projects custom fields
}

// DefaultFetchOptions returns sensible defaults for fetching
func DefaultFetchOptions() *FetchOptions {
	return &FetchOptions{
		State:               "all",
		Page:                1,
		PerPage:             100,
		Sort:                "updated",
		Direction:           "desc",
		IncludeCustomFields: false, // Default to false for performance
	}
}

// IsPullRequest returns true if the issue is actually a pull request
func (i *Issue) IsPullRequest() bool {
	return i.PullRequest != nil
}

// CalculateDuration returns the duration between creation and closure
func (i *Issue) CalculateDuration() *time.Duration {
	if i.ClosedAt == nil {
		return nil
	}
	duration := i.ClosedAt.Sub(i.CreatedAt)
	return &duration
}

// CalculateMergeTime returns the time between PR creation and merge
func (pr *PullRequest) CalculateMergeTime() *time.Duration {
	if pr.MergedAt == nil {
		return nil
	}
	duration := pr.MergedAt.Sub(pr.CreatedAt)
	return &duration
}

// ProjectV2 represents a GitHub Project (Projects V2)
type ProjectV2 struct {
	ID          string `json:"id"`
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Closed      bool   `json:"closed"`
	Public      bool   `json:"public"`
}

// ProjectV2Item represents an item in a GitHub Project V2
type ProjectV2Item struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // ISSUE, PULL_REQUEST, DRAFT_ISSUE
	FieldValues map[string]interface{} `json:"field_values"`
}

// ProjectV2Field represents a field definition in a GitHub Project V2
type ProjectV2Field struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	DataType string `json:"data_type"` // TEXT, NUMBER, DATE, SINGLE_SELECT, ITERATION
}

// ProjectV2SingleSelectOption represents an option for a single-select field
type ProjectV2SingleSelectOption struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// GraphQLQuery represents a GitHub GraphQL query request
type GraphQLQuery struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// GraphQLResponse represents a GitHub GraphQL response
type GraphQLResponse struct {
	Data   interface{}    `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error
type GraphQLError struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}
