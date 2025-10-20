package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ValidationResult holds the result of token validation
type ValidationResult struct {
	Valid              bool
	Username           string
	Scopes             []string
	HasRepoAccess      bool
	HasIssuesAccess    bool
	HasPullsAccess     bool
	HasProjectsAccess  bool
	RepoExists         bool
	RepoName           string
	RepoPrivate        bool
	RateLimit          int
	RateLimitRemaining int
	Error              string
}

// ValidateToken checks if a GitHub token is valid and has necessary permissions
func ValidateToken(token string) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid: false,
	}

	if token == "" {
		result.Error = "Token is empty"
		return result, fmt.Errorf("token is empty")
	}

	// Test token by fetching user info
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to create request: %v", err)
		return result, err
	}

	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to connect to GitHub: %v", err)
		return result, err
	}
	defer resp.Body.Close()

	// Extract rate limit info
	if limit := resp.Header.Get("X-RateLimit-Limit"); limit != "" {
		fmt.Sscanf(limit, "%d", &result.RateLimit)
	}
	if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "" {
		fmt.Sscanf(remaining, "%d", &result.RateLimitRemaining)
	}

	// Extract scopes
	if scopes := resp.Header.Get("X-OAuth-Scopes"); scopes != "" {
		result.Scopes = strings.Split(strings.ReplaceAll(scopes, " ", ""), ",")
	}

	if resp.StatusCode == 401 {
		result.Error = "Invalid token: Authentication failed"
		return result, fmt.Errorf("invalid token")
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		result.Error = fmt.Sprintf("GitHub API error: %s (status %d)", string(body), resp.StatusCode)
		return result, fmt.Errorf("API error: status %d", resp.StatusCode)
	}

	// Parse user info
	var userInfo struct {
		Login string `json:"login"`
		Name  string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		result.Error = fmt.Sprintf("Failed to parse response: %v", err)
		return result, err
	}

	result.Valid = true
	result.Username = userInfo.Login

	// Check for required scopes
	result.HasRepoAccess = hasScope(result.Scopes, "repo") || hasScope(result.Scopes, "public_repo")
	result.HasIssuesAccess = hasScope(result.Scopes, "repo") || hasScope(result.Scopes, "public_repo")
	result.HasPullsAccess = hasScope(result.Scopes, "repo") || hasScope(result.Scopes, "public_repo")
	// Projects V2 requires either 'repo' scope or 'project' scope (read:project for reading)
	result.HasProjectsAccess = hasScope(result.Scopes, "repo") || hasScope(result.Scopes, "project") || hasScope(result.Scopes, "read:project")

	return result, nil
}

// ValidateTokenAndRepo checks if token has access to a specific repository
func ValidateTokenAndRepo(token, repoFullName string) (*ValidationResult, error) {
	// First validate the token
	result, err := ValidateToken(token)
	if err != nil {
		return result, err
	}

	if !result.Valid {
		return result, fmt.Errorf("token is invalid")
	}

	// Check if repository name is provided
	if repoFullName == "" {
		result.Error = "Repository name not provided"
		return result, nil
	}

	// Test access to specific repository
	client := &http.Client{Timeout: 10 * time.Second}
	url := fmt.Sprintf("https://api.github.com/repos/%s", repoFullName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to create request: %v", err)
		return result, err
	}

	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to connect to GitHub: %v", err)
		return result, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		result.Error = fmt.Sprintf("Repository '%s' not found or no access", repoFullName)
		result.RepoExists = false
		return result, nil
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		result.Error = fmt.Sprintf("Cannot access repository: %s (status %d)", string(body), resp.StatusCode)
		result.RepoExists = false
		return result, nil
	}

	// Parse repository info
	var repoInfo struct {
		Name    string `json:"name"`
		Private bool   `json:"private"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		result.Error = fmt.Sprintf("Failed to parse repository info: %v", err)
		return result, err
	}

	result.RepoExists = true
	result.RepoName = repoInfo.Name
	result.RepoPrivate = repoInfo.Private

	return result, nil
}

// CheckRepositoryPermissions checks specific permissions for issues and PRs
func CheckRepositoryPermissions(token, repoFullName string) (issues, pulls bool, err error) {
	client := &http.Client{Timeout: 10 * time.Second}

	// Test issues access
	issuesURL := fmt.Sprintf("https://api.github.com/repos/%s/issues?per_page=1", repoFullName)
	issuesReq, _ := http.NewRequest("GET", issuesURL, nil)
	issuesReq.Header.Set("Authorization", "token "+token)
	issuesReq.Header.Set("Accept", "application/vnd.github.v3+json")

	issuesResp, err := client.Do(issuesReq)
	if err != nil {
		return false, false, err
	}
	defer issuesResp.Body.Close()

	issues = issuesResp.StatusCode == 200

	// Test pull requests access
	pullsURL := fmt.Sprintf("https://api.github.com/repos/%s/pulls?per_page=1", repoFullName)
	pullsReq, _ := http.NewRequest("GET", pullsURL, nil)
	pullsReq.Header.Set("Authorization", "token "+token)
	pullsReq.Header.Set("Accept", "application/vnd.github.v3+json")

	pullsResp, err := client.Do(pullsReq)
	if err != nil {
		return issues, false, err
	}
	defer pullsResp.Body.Close()

	pulls = pullsResp.StatusCode == 200

	return issues, pulls, nil
}

// hasScope checks if a scope is present in the list
func hasScope(scopes []string, scope string) bool {
	for _, s := range scopes {
		if s == scope {
			return true
		}
	}
	return false
}
