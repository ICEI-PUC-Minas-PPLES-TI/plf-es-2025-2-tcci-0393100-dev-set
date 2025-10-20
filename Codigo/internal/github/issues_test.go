package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFetchIssues_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/owner/repo/issues", r.URL.Path)
		assert.Contains(t, r.Header.Get("Authorization"), "token")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		issues := []Issue{
			{
				ID:     1,
				Number: 1,
				Title:  "Test Issue",
				State:  "open",
			},
			{
				ID:     2,
				Number: 2,
				Title:  "Test Issue 2",
				State:  "open",
			},
		}
		json.NewEncoder(w).Encode(issues)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	opts := &FetchOptions{
		State:   "open",
		Page:    1,
		PerPage: 100,
	}

	issues, err := client.FetchIssues(ctx, "owner", "repo", opts)

	assert.NoError(t, err)
	assert.Len(t, issues, 2)
	assert.Equal(t, "Test Issue", issues[0].Title)
}

func TestFetchIssues_WithPagination(t *testing.T) {
	pageCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageCount++

		w.Header().Set("Content-Type", "application/json")

		if pageCount == 1 {
			// First page with Link header
			w.Header().Set("Link", `<http://api.github.com/repos/owner/repo/issues?page=2>; rel="next"`)
			issues := []Issue{{ID: 1, Number: 1, Title: "Issue 1"}}
			json.NewEncoder(w).Encode(issues)
		} else {
			// Second page without Link header
			issues := []Issue{{ID: 2, Number: 2, Title: "Issue 2"}}
			json.NewEncoder(w).Encode(issues)
		}
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	issues, err := client.FetchIssues(ctx, "owner", "repo", nil)

	assert.NoError(t, err)
	assert.Len(t, issues, 2)
	assert.Equal(t, 2, pageCount)
}

func TestFetchIssues_FiltersPullRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		issues := []Issue{
			{ID: 1, Number: 1, Title: "Real Issue", PullRequest: nil},
			{ID: 2, Number: 2, Title: "Pull Request", PullRequest: &struct{}{}},
		}
		json.NewEncoder(w).Encode(issues)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	issues, err := client.FetchIssues(ctx, "owner", "repo", nil)

	assert.NoError(t, err)
	assert.Len(t, issues, 1)
	assert.Equal(t, "Real Issue", issues[0].Title)
}

func TestFetchIssue_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/owner/repo/issues/42", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		issue := Issue{
			ID:     42,
			Number: 42,
			Title:  "Single Issue",
			State:  "open",
		}
		json.NewEncoder(w).Encode(issue)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	issue, err := client.FetchIssue(ctx, "owner", "repo", 42)

	assert.NoError(t, err)
	assert.NotNil(t, issue)
	assert.Equal(t, int64(42), issue.ID)
	assert.Equal(t, "Single Issue", issue.Title)
}

func TestFetchIssuesSince_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify since parameter is present
		since := r.URL.Query().Get("since")
		assert.NotEmpty(t, since)

		w.Header().Set("Content-Type", "application/json")
		issues := []Issue{
			{ID: 1, Number: 1, Title: "Recent Issue"},
		}
		json.NewEncoder(w).Encode(issues)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	since := time.Now().Add(-24 * time.Hour)
	issues, err := client.FetchIssuesSince(ctx, "owner", "repo", &since)

	assert.NoError(t, err)
	assert.Len(t, issues, 1)
}

func TestCountIssues_WithPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Link", `<http://api.github.com/repos/owner/repo/issues?page=10>; rel="last"`)

		issues := []Issue{{ID: 1, Number: 1}}
		json.NewEncoder(w).Encode(issues)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	count, err := client.CountIssues(ctx, "owner", "repo", "open")

	assert.NoError(t, err)
	assert.Equal(t, 10, count)
}

func TestCountIssues_NoPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		issues := []Issue{
			{ID: 1, Number: 1},
			{ID: 2, Number: 2},
			{ID: 3, Number: 3},
		}
		json.NewEncoder(w).Encode(issues)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	count, err := client.CountIssues(ctx, "owner", "repo", "open")

	assert.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestFetchIssues_ContextCancellation(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Link", `<http://api.github.com/repos/owner/repo/issues?page=2>; rel="next"`)

		issues := []Issue{{ID: int64(callCount), Number: callCount}}
		json.NewEncoder(w).Encode(issues)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.FetchIssues(ctx, "owner", "repo", nil)

	// Should get context cancellation error
	assert.Error(t, err)
}

func TestFetchIssuesPage_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	opts := DefaultFetchOptions()
	_, _, err := client.fetchIssuesPage(ctx, "owner", "repo", opts, 1)

	assert.Error(t, err)
}

func TestFetchIssues_DefaultOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify default options are applied
		assert.Equal(t, "all", r.URL.Query().Get("state"))
		assert.Equal(t, "100", r.URL.Query().Get("per_page"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Issue{})
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	_, err := client.FetchIssues(ctx, "owner", "repo", nil)

	assert.NoError(t, err)
}

func TestCountIssues_InvalidLastPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Link", `<http://api.github.com/repos/owner/repo/issues?page=invalid>; rel="last"`)

		json.NewEncoder(w).Encode([]Issue{{ID: 1}})
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	count, err := client.CountIssues(ctx, "owner", "repo", "open")

	assert.NoError(t, err)
	assert.Equal(t, 1, count) // Falls back to counting items in response
}

func TestFetchIssue_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	_, err := client.FetchIssue(ctx, "owner", "repo", 999)

	assert.Error(t, err)
}

func TestFetchIssuesPage_DecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	opts := DefaultFetchOptions()
	_, _, err := client.fetchIssuesPage(ctx, "owner", "repo", opts, 1)

	assert.Error(t, err)
}

func TestCountIssues_BuildRequestError(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	// Invalid URL should cause buildRequest to fail
	client.baseURL = "://invalid-url"

	_, err := client.CountIssues(ctx, "owner", "repo", "open")

	assert.Error(t, err)
}

func TestFetchIssues_WithSince(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify since parameter is present
		sinceParam := r.URL.Query().Get("since")
		assert.NotEmpty(t, sinceParam)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Issue{{ID: 1, Number: 1, Title: "Recent"}})
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	since := time.Now().Add(-24 * time.Hour)
	opts := &FetchOptions{
		State: "all",
		Since: &since,
	}

	issues, err := client.FetchIssues(ctx, "owner", "repo", opts)

	assert.NoError(t, err)
	assert.Len(t, issues, 1)
}

func TestCountIssues_DoRequestError(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	// Invalid baseURL will cause doRequestRaw to fail
	client.baseURL = "http://invalid-server-that-does-not-exist-12345.com"

	_, err := client.CountIssues(ctx, "owner", "repo", "open")

	assert.Error(t, err)
}
