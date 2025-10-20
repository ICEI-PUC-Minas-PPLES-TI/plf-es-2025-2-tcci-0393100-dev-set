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

func TestFetchPullRequests_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/owner/repo/pulls", r.URL.Path)
		assert.Contains(t, r.Header.Get("Authorization"), "token")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		prs := []PullRequest{
			{
				ID:     1,
				Number: 1,
				Title:  "Test PR",
				State:  "open",
			},
			{
				ID:     2,
				Number: 2,
				Title:  "Test PR 2",
				State:  "merged",
			},
		}
		json.NewEncoder(w).Encode(prs)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	opts := &FetchOptions{
		State:   "all",
		Page:    1,
		PerPage: 100,
	}

	prs, err := client.FetchPullRequests(ctx, "owner", "repo", opts)

	assert.NoError(t, err)
	assert.Len(t, prs, 2)
	assert.Equal(t, "Test PR", prs[0].Title)
}

func TestFetchPullRequests_WithPagination(t *testing.T) {
	pageCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageCount++

		w.Header().Set("Content-Type", "application/json")

		if pageCount == 1 {
			w.Header().Set("Link", `<http://api.github.com/repos/owner/repo/pulls?page=2>; rel="next"`)
			prs := []PullRequest{{ID: 1, Number: 1, Title: "PR 1"}}
			json.NewEncoder(w).Encode(prs)
		} else {
			prs := []PullRequest{{ID: 2, Number: 2, Title: "PR 2"}}
			json.NewEncoder(w).Encode(prs)
		}
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	prs, err := client.FetchPullRequests(ctx, "owner", "repo", nil)

	assert.NoError(t, err)
	assert.Len(t, prs, 2)
	assert.Equal(t, 2, pageCount)
}

func TestFetchPullRequest_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/owner/repo/pulls/42", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		pr := PullRequest{
			ID:        42,
			Number:    42,
			Title:     "Single PR",
			State:     "open",
			Merged:    false,
			Commits:   5,
			Additions: 100,
			Deletions: 50,
		}
		json.NewEncoder(w).Encode(pr)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	pr, err := client.FetchPullRequest(ctx, "owner", "repo", 42)

	assert.NoError(t, err)
	assert.NotNil(t, pr)
	assert.Equal(t, int64(42), pr.ID)
	assert.Equal(t, "Single PR", pr.Title)
	assert.Equal(t, 5, pr.Commits)
}

func TestFetchPullRequestsSince_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		prs := []PullRequest{
			{ID: 1, Number: 1, Title: "Recent PR"},
		}
		json.NewEncoder(w).Encode(prs)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	since := time.Now().Add(-24 * time.Hour)
	prs, err := client.FetchPullRequestsSince(ctx, "owner", "repo", &since)

	assert.NoError(t, err)
	assert.Len(t, prs, 1)
}

func TestCountPullRequests_WithPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Link", `<http://api.github.com/repos/owner/repo/pulls?page=15>; rel="last"`)

		prs := []PullRequest{{ID: 1, Number: 1}}
		json.NewEncoder(w).Encode(prs)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	count, err := client.CountPullRequests(ctx, "owner", "repo", "all")

	assert.NoError(t, err)
	assert.Equal(t, 15, count)
}

func TestCountPullRequests_NoPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		prs := []PullRequest{
			{ID: 1, Number: 1},
			{ID: 2, Number: 2},
		}
		json.NewEncoder(w).Encode(prs)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	count, err := client.CountPullRequests(ctx, "owner", "repo", "all")

	assert.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestFetchPullRequests_ContextCancellation(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Link", `<http://api.github.com/repos/owner/repo/pulls?page=2>; rel="next"`)

		prs := []PullRequest{{ID: int64(callCount), Number: callCount}}
		json.NewEncoder(w).Encode(prs)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.FetchPullRequests(ctx, "owner", "repo", nil)

	assert.Error(t, err)
}

func TestFetchPullRequestsPage_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	opts := DefaultFetchOptions()
	_, _, err := client.fetchPullRequestsPage(ctx, "owner", "repo", opts, 1)

	assert.Error(t, err)
}

func TestFetchPullRequests_DefaultOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "all", r.URL.Query().Get("state"))
		assert.Equal(t, "100", r.URL.Query().Get("per_page"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]PullRequest{})
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	_, err := client.FetchPullRequests(ctx, "owner", "repo", nil)

	assert.NoError(t, err)
}

func TestCountPullRequests_InvalidLastPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Link", `<http://api.github.com/repos/owner/repo/pulls?page=invalid>; rel="last"`)

		json.NewEncoder(w).Encode([]PullRequest{{ID: 1}})
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	count, err := client.CountPullRequests(ctx, "owner", "repo", "all")

	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestFetchPullRequest_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL
	ctx := context.Background()

	_, err := client.FetchPullRequest(ctx, "owner", "repo", 999)

	assert.Error(t, err)
}

func TestFetchPullRequestsPage_DecodeError(t *testing.T) {
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
	_, _, err := client.fetchPullRequestsPage(ctx, "owner", "repo", opts, 1)

	assert.Error(t, err)
}

func TestCountPullRequests_BuildRequestError(t *testing.T) {
	client := NewClient("test-token")
	ctx := context.Background()

	client.baseURL = "://invalid-url"

	_, err := client.CountPullRequests(ctx, "owner", "repo", "all")

	assert.Error(t, err)
}
