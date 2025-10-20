package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"set/internal/github"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestStore(t *testing.T) (*Store, string) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	store, err := NewStore(dbPath)
	require.NoError(t, err)
	require.NotNil(t, store)

	return store, dbPath
}

func TestNewStore(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	store, err := NewStore(dbPath)
	require.NoError(t, err)
	require.NotNil(t, store)
	defer store.Close()

	assert.Equal(t, dbPath, store.path)
	assert.NotNil(t, store.db)

	// Verify database file was created
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)
}

func TestSaveAndGetIssue(t *testing.T) {
	store, _ := setupTestStore(t)
	defer store.Close()

	createdAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	issues := []*github.Issue{
		{
			ID:        123,
			Number:    1,
			Title:     "Test Issue",
			Body:      "Test body",
			State:     "open",
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
	}

	// Save issue
	err := store.SaveIssues(issues)
	require.NoError(t, err)

	// Retrieve issue
	retrieved, err := store.GetIssue(1)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, int64(123), retrieved.ID)
	assert.Equal(t, 1, retrieved.Number)
	assert.Equal(t, "Test Issue", retrieved.Title)
	assert.Equal(t, "Test body", retrieved.Body)
	assert.Equal(t, "open", retrieved.State)
}

func TestGetIssue_NotFound(t *testing.T) {
	store, _ := setupTestStore(t)
	defer store.Close()

	_, err := store.GetIssue(999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSaveAndGetPullRequest(t *testing.T) {
	store, _ := setupTestStore(t)
	defer store.Close()

	createdAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	mergedAt := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)

	prs := []*github.PullRequest{
		{
			ID:        456,
			Number:    2,
			Title:     "Test PR",
			Body:      "Test PR body",
			State:     "merged",
			Merged:    true,
			CreatedAt: createdAt,
			MergedAt:  &mergedAt,
			Commits:   5,
			Additions: 100,
			Deletions: 50,
		},
	}

	// Save PR
	err := store.SavePullRequests(prs)
	require.NoError(t, err)

	// Retrieve PR
	retrieved, err := store.GetPullRequest(2)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, int64(456), retrieved.ID)
	assert.Equal(t, 2, retrieved.Number)
	assert.Equal(t, "Test PR", retrieved.Title)
	assert.Equal(t, "merged", retrieved.State)
	assert.True(t, retrieved.Merged)
	assert.Equal(t, 5, retrieved.Commits)
	assert.Equal(t, 100, retrieved.Additions)
	assert.Equal(t, 50, retrieved.Deletions)
}

func TestGetAllIssues(t *testing.T) {
	store, _ := setupTestStore(t)
	defer store.Close()

	issues := []*github.Issue{
		{Number: 1, Title: "Issue 1", State: "open"},
		{Number: 2, Title: "Issue 2", State: "closed"},
		{Number: 3, Title: "Issue 3", State: "open"},
	}

	err := store.SaveIssues(issues)
	require.NoError(t, err)

	retrieved, err := store.GetAllIssues()
	require.NoError(t, err)
	assert.Len(t, retrieved, 3)
}

func TestGetAllPullRequests(t *testing.T) {
	store, _ := setupTestStore(t)
	defer store.Close()

	prs := []*github.PullRequest{
		{Number: 1, Title: "PR 1", State: "open"},
		{Number: 2, Title: "PR 2", State: "merged"},
	}

	err := store.SavePullRequests(prs)
	require.NoError(t, err)

	retrieved, err := store.GetAllPullRequests()
	require.NoError(t, err)
	assert.Len(t, retrieved, 2)
}

func TestSetAndGetLastSync(t *testing.T) {
	store, _ := setupTestStore(t)
	defer store.Close()

	syncTime := time.Date(2024, 1, 15, 12, 30, 0, 0, time.UTC)

	// Set last sync
	err := store.SetLastSync(syncTime)
	require.NoError(t, err)

	// Get last sync
	retrieved, err := store.GetLastSync()
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	// Compare times (allowing for slight precision differences)
	assert.True(t, syncTime.Equal(*retrieved))
}

func TestGetLastSync_NoSync(t *testing.T) {
	store, _ := setupTestStore(t)
	defer store.Close()

	retrieved, err := store.GetLastSync()
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestSetAndGetRepository(t *testing.T) {
	store, _ := setupTestStore(t)
	defer store.Close()

	repo := &github.Repository{
		ID:          789,
		Name:        "test-repo",
		FullName:    "user/test-repo",
		Description: "A test repository",
		Private:     false,
		URL:         "https://github.com/user/test-repo",
	}

	// Set repository
	err := store.SetRepository(repo)
	require.NoError(t, err)

	// Get repository
	retrieved, err := store.GetRepository()
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, int64(789), retrieved.ID)
	assert.Equal(t, "test-repo", retrieved.Name)
	assert.Equal(t, "user/test-repo", retrieved.FullName)
	assert.Equal(t, "A test repository", retrieved.Description)
	assert.False(t, retrieved.Private)
}

func TestCountIssues(t *testing.T) {
	store, _ := setupTestStore(t)
	defer store.Close()

	// Initially should be 0
	count, err := store.CountIssues()
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Add some issues
	issues := []*github.Issue{
		{Number: 1, Title: "Issue 1"},
		{Number: 2, Title: "Issue 2"},
		{Number: 3, Title: "Issue 3"},
	}
	err = store.SaveIssues(issues)
	require.NoError(t, err)

	// Count should be 3
	count, err = store.CountIssues()
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestCountPullRequests(t *testing.T) {
	store, _ := setupTestStore(t)
	defer store.Close()

	// Initially should be 0
	count, err := store.CountPullRequests()
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Add some PRs
	prs := []*github.PullRequest{
		{Number: 1, Title: "PR 1"},
		{Number: 2, Title: "PR 2"},
	}
	err = store.SavePullRequests(prs)
	require.NoError(t, err)

	// Count should be 2
	count, err = store.CountPullRequests()
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestClear(t *testing.T) {
	store, _ := setupTestStore(t)
	defer store.Close()

	// Add some data
	issues := []*github.Issue{{Number: 1, Title: "Issue 1"}}
	prs := []*github.PullRequest{{Number: 1, Title: "PR 1"}}
	repo := &github.Repository{ID: 123, Name: "test"}

	err := store.SaveIssues(issues)
	require.NoError(t, err)
	err = store.SavePullRequests(prs)
	require.NoError(t, err)
	err = store.SetRepository(repo)
	require.NoError(t, err)

	// Verify data exists
	issueCount, _ := store.CountIssues()
	prCount, _ := store.CountPullRequests()
	assert.Equal(t, 1, issueCount)
	assert.Equal(t, 1, prCount)

	// Clear storage
	err = store.Clear()
	require.NoError(t, err)

	// Verify data is gone
	issueCount, _ = store.CountIssues()
	prCount, _ = store.CountPullRequests()
	assert.Equal(t, 0, issueCount)
	assert.Equal(t, 0, prCount)

	// Repository should also be gone
	_, err = store.GetRepository()
	assert.Error(t, err)
}

func TestClose(t *testing.T) {
	store, _ := setupTestStore(t)

	err := store.Close()
	assert.NoError(t, err)

	// Closing again should not error
	err = store.Close()
	assert.NoError(t, err)
}

func TestSaveIssues_Batch(t *testing.T) {
	store, _ := setupTestStore(t)
	defer store.Close()

	// Create a large batch
	issues := make([]*github.Issue, 100)
	for i := 0; i < 100; i++ {
		issues[i] = &github.Issue{
			Number: i + 1,
			Title:  fmt.Sprintf("Issue %d", i+1),
		}
	}

	err := store.SaveIssues(issues)
	require.NoError(t, err)

	count, err := store.CountIssues()
	require.NoError(t, err)
	assert.Equal(t, 100, count)
}

func TestNewStore_InvalidPath(t *testing.T) {
	// Try to create store in invalid location
	_, err := NewStore("/invalid/path/that/does/not/exist/test.db")
	assert.Error(t, err)
}

func TestGetPullRequest_NotFound(t *testing.T) {
	store, _ := setupTestStore(t)
	defer store.Close()

	_, err := store.GetPullRequest(99999)
	assert.Error(t, err)
}

func TestGetAllIssues_FilterByState(t *testing.T) {
	store, _ := setupTestStore(t)
	defer store.Close()

	issues := []*github.Issue{
		{Number: 1, Title: "Open Issue", State: "open"},
		{Number: 2, Title: "Closed Issue", State: "closed"},
		{Number: 3, Title: "Open Issue 2", State: "open"},
	}

	err := store.SaveIssues(issues)
	require.NoError(t, err)

	// Get open issues
	openIssues, err := store.GetAllIssues()
	require.NoError(t, err)
	assert.Len(t, openIssues, 3)
}

func TestGetAllPullRequests_FilterByState(t *testing.T) {
	store, _ := setupTestStore(t)
	defer store.Close()

	prs := []*github.PullRequest{
		{Number: 1, Title: "Open PR", State: "open"},
		{Number: 2, Title: "Closed PR", State: "closed"},
		{Number: 3, Title: "Open PR 2", State: "open"},
	}

	err := store.SavePullRequests(prs)
	require.NoError(t, err)

	allPRs, err := store.GetAllPullRequests()
	require.NoError(t, err)
	assert.Len(t, allPRs, 3)
}

func TestSetRepository_EmptyValues(t *testing.T) {
	store, _ := setupTestStore(t)
	defer store.Close()

	emptyRepo := &github.Repository{
		Name:     "",
		FullName: "",
	}

	err := store.SetRepository(emptyRepo)
	assert.NoError(t, err)

	repo, err := store.GetRepository()
	require.NoError(t, err)
	assert.Equal(t, "", repo.Name)
	assert.Equal(t, "", repo.FullName)
}

func TestCountIssues_Empty(t *testing.T) {
	store, _ := setupTestStore(t)
	defer store.Close()

	count, err := store.CountIssues()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestCountPullRequests_Empty(t *testing.T) {
	store, _ := setupTestStore(t)
	defer store.Close()

	count, err := store.CountPullRequests()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestSaveIssues_EmptySlice(t *testing.T) {
	store, _ := setupTestStore(t)
	defer store.Close()

	err := store.SaveIssues([]*github.Issue{})
	assert.NoError(t, err)

	count, err := store.CountIssues()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestSavePullRequests_EmptySlice(t *testing.T) {
	store, _ := setupTestStore(t)
	defer store.Close()

	err := store.SavePullRequests([]*github.PullRequest{})
	assert.NoError(t, err)

	count, err := store.CountPullRequests()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}
