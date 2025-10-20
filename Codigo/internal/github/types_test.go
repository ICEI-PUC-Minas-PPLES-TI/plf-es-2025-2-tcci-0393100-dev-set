package github

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsPullRequest(t *testing.T) {
	tests := []struct {
		name     string
		issue    Issue
		expected bool
	}{
		{
			name: "Issue without PR field",
			issue: Issue{
				Number:      1,
				Title:       "Test Issue",
				PullRequest: nil,
			},
			expected: false,
		},
		{
			name: "Issue with PR field",
			issue: Issue{
				Number:      2,
				Title:       "Test PR",
				PullRequest: &struct{}{},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.issue.IsPullRequest()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateDuration(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	closedTime := baseTime.Add(24 * time.Hour)

	tests := []struct {
		name     string
		issue    Issue
		expected *time.Duration
	}{
		{
			name: "Open issue",
			issue: Issue{
				CreatedAt: baseTime,
				ClosedAt:  nil,
			},
			expected: nil,
		},
		{
			name: "Closed issue",
			issue: Issue{
				CreatedAt: baseTime,
				ClosedAt:  &closedTime,
			},
			expected: func() *time.Duration {
				d := 24 * time.Hour
				return &d
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.issue.CalculateDuration()
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

func TestCalculateMergeTime(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	mergedTime := baseTime.Add(48 * time.Hour)

	tests := []struct {
		name     string
		pr       PullRequest
		expected *time.Duration
	}{
		{
			name: "Unmerged PR",
			pr: PullRequest{
				CreatedAt: baseTime,
				MergedAt:  nil,
			},
			expected: nil,
		},
		{
			name: "Merged PR",
			pr: PullRequest{
				CreatedAt: baseTime,
				MergedAt:  &mergedTime,
			},
			expected: func() *time.Duration {
				d := 48 * time.Hour
				return &d
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.pr.CalculateMergeTime()
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

func TestDefaultFetchOptions(t *testing.T) {
	opts := DefaultFetchOptions()

	assert.NotNil(t, opts)
	assert.Equal(t, "all", opts.State)
	assert.Equal(t, 1, opts.Page)
	assert.Equal(t, 100, opts.PerPage)
	assert.Equal(t, "updated", opts.Sort)
	assert.Equal(t, "desc", opts.Direction)
	assert.Nil(t, opts.Since)
}
