package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"set/internal/logger"
)

const graphqlURL = "https://api.github.com/graphql"

// ExecuteGraphQL executes a GraphQL query against the GitHub API
func (c *Client) ExecuteGraphQL(ctx context.Context, query string, variables map[string]interface{}) (map[string]interface{}, error) {
	gqlQuery := GraphQLQuery{
		Query:     query,
		Variables: variables,
	}

	jsonData, err := json.Marshal(gqlQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GraphQL query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", graphqlURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create GraphQL request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GraphQL request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GraphQL request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var gqlResp struct {
		Data   map[string]interface{} `json:"data"`
		Errors []GraphQLError         `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return nil, fmt.Errorf("failed to decode GraphQL response: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL errors: %v", gqlResp.Errors)
	}

	return gqlResp.Data, nil
}

// FetchProjectV2Fields fetches all fields defined in a GitHub Project V2
func (c *Client) FetchProjectV2Fields(ctx context.Context, owner, projectNumber int) ([]ProjectV2Field, error) {
	query := `
		query($owner: String!, $number: Int!) {
			organization(login: $owner) {
				projectV2(number: $number) {
					fields(first: 100) {
						nodes {
							... on ProjectV2Field {
								id
								name
								dataType
							}
							... on ProjectV2SingleSelectField {
								id
								name
								dataType
								options {
									id
									name
								}
							}
							... on ProjectV2IterationField {
								id
								name
								dataType
							}
						}
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"owner":  owner,
		"number": projectNumber,
	}

	data, err := c.ExecuteGraphQL(ctx, query, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch project fields: %w", err)
	}

	// Parse the response
	var fields []ProjectV2Field
	// Note: This is a simplified parser - you may need to adapt based on the actual response structure
	if org, ok := data["organization"].(map[string]interface{}); ok {
		if project, ok := org["projectV2"].(map[string]interface{}); ok {
			if fieldsData, ok := project["fields"].(map[string]interface{}); ok {
				if nodes, ok := fieldsData["nodes"].([]interface{}); ok {
					for _, node := range nodes {
						if fieldMap, ok := node.(map[string]interface{}); ok {
							field := ProjectV2Field{
								ID:       fieldMap["id"].(string),
								Name:     fieldMap["name"].(string),
								DataType: fieldMap["dataType"].(string),
							}
							fields = append(fields, field)
						}
					}
				}
			}
		}
	}

	return fields, nil
}

// FetchIssueProjectData fetches GitHub Projects V2 custom field data for a specific issue
func (c *Client) FetchIssueProjectData(ctx context.Context, owner, repo string, issueNumber int) (map[string]interface{}, error) {
	query := `
		query($owner: String!, $repo: String!, $number: Int!) {
			repository(owner: $owner, name: $repo) {
				issue(number: $number) {
					projectItems(first: 10) {
						nodes {
							id
							project {
								id
								title
								number
							}
							fieldValues(first: 100) {
								nodes {
									... on ProjectV2ItemFieldTextValue {
										text
										field {
											... on ProjectV2Field {
												name
											}
										}
									}
									... on ProjectV2ItemFieldNumberValue {
										number
										field {
											... on ProjectV2Field {
												name
											}
										}
									}
									... on ProjectV2ItemFieldDateValue {
										date
										field {
											... on ProjectV2Field {
												name
											}
										}
									}
									... on ProjectV2ItemFieldSingleSelectValue {
										name
										field {
											... on ProjectV2SingleSelectField {
												name
											}
										}
									}
									... on ProjectV2ItemFieldIterationValue {
										title
										field {
											... on ProjectV2IterationField {
												name
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"owner":  owner,
		"repo":   repo,
		"number": issueNumber,
	}

	logger.Infof("Fetching project data for issue #%d in %s/%s", issueNumber, owner, repo)

	data, err := c.ExecuteGraphQL(ctx, query, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch issue project data: %w", err)
	}

	// Parse custom fields from the response
	customFields := make(map[string]interface{})

	if repo, ok := data["repository"].(map[string]interface{}); ok {
		if issue, ok := repo["issue"].(map[string]interface{}); ok {
			if projectItems, ok := issue["projectItems"].(map[string]interface{}); ok {
				if nodes, ok := projectItems["nodes"].([]interface{}); ok {
					for _, node := range nodes {
						if itemMap, ok := node.(map[string]interface{}); ok {
							if fieldValues, ok := itemMap["fieldValues"].(map[string]interface{}); ok {
								if valueNodes, ok := fieldValues["nodes"].([]interface{}); ok {
									for _, valueNode := range valueNodes {
										if valueMap, ok := valueNode.(map[string]interface{}); ok {
											// Extract field name and value
											fieldName := extractFieldName(valueMap)
											fieldValue := extractFieldValue(valueMap)

											if fieldName != "" && fieldValue != nil {
												customFields[fieldName] = fieldValue
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	logger.Infof("Found %d custom fields for issue #%d", len(customFields), issueNumber)
	return customFields, nil
}

// FetchPullRequestProjectData fetches GitHub Projects V2 custom field data for a specific PR
func (c *Client) FetchPullRequestProjectData(ctx context.Context, owner, repo string, prNumber int) (map[string]interface{}, error) {
	query := `
		query($owner: String!, $repo: String!, $number: Int!) {
			repository(owner: $owner, name: $repo) {
				pullRequest(number: $number) {
					projectItems(first: 10) {
						nodes {
							id
							project {
								id
								title
								number
							}
							fieldValues(first: 100) {
								nodes {
									... on ProjectV2ItemFieldTextValue {
										text
										field {
											... on ProjectV2Field {
												name
											}
										}
									}
									... on ProjectV2ItemFieldNumberValue {
										number
										field {
											... on ProjectV2Field {
												name
											}
										}
									}
									... on ProjectV2ItemFieldDateValue {
										date
										field {
											... on ProjectV2Field {
												name
											}
										}
									}
									... on ProjectV2ItemFieldSingleSelectValue {
										name
										field {
											... on ProjectV2SingleSelectField {
												name
											}
										}
									}
									... on ProjectV2ItemFieldIterationValue {
										title
										field {
											... on ProjectV2IterationField {
												name
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"owner":  owner,
		"repo":   repo,
		"number": prNumber,
	}

	logger.Infof("Fetching project data for PR #%d in %s/%s", prNumber, owner, repo)

	data, err := c.ExecuteGraphQL(ctx, query, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PR project data: %w", err)
	}

	// Parse custom fields from the response
	customFields := make(map[string]interface{})

	if repo, ok := data["repository"].(map[string]interface{}); ok {
		if pr, ok := repo["pullRequest"].(map[string]interface{}); ok {
			if projectItems, ok := pr["projectItems"].(map[string]interface{}); ok {
				if nodes, ok := projectItems["nodes"].([]interface{}); ok {
					for _, node := range nodes {
						if itemMap, ok := node.(map[string]interface{}); ok {
							if fieldValues, ok := itemMap["fieldValues"].(map[string]interface{}); ok {
								if valueNodes, ok := fieldValues["nodes"].([]interface{}); ok {
									for _, valueNode := range valueNodes {
										if valueMap, ok := valueNode.(map[string]interface{}); ok {
											// Extract field name and value
											fieldName := extractFieldName(valueMap)
											fieldValue := extractFieldValue(valueMap)

											if fieldName != "" && fieldValue != nil {
												customFields[fieldName] = fieldValue
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	logger.Infof("Found %d custom fields for PR #%d", len(customFields), prNumber)
	return customFields, nil
}

// Helper function to extract field name from various field types
func extractFieldName(valueMap map[string]interface{}) string {
	if field, ok := valueMap["field"].(map[string]interface{}); ok {
		if name, ok := field["name"].(string); ok {
			return name
		}
	}
	return ""
}

// Helper function to extract field value from various field types
func extractFieldValue(valueMap map[string]interface{}) interface{} {
	// Try text value
	if text, ok := valueMap["text"].(string); ok {
		return text
	}
	// Try number value
	if number, ok := valueMap["number"].(float64); ok {
		return number
	}
	// Try date value
	if date, ok := valueMap["date"].(string); ok {
		return date
	}
	// Try single select value (name)
	if name, ok := valueMap["name"].(string); ok {
		return name
	}
	// Try iteration value (title)
	if title, ok := valueMap["title"].(string); ok {
		return title
	}
	return nil
}

// EnrichIssuesWithProjectData fetches and adds custom field data to issues
func (c *Client) EnrichIssuesWithProjectData(ctx context.Context, owner, repo string, issues []*Issue) error {
	logger.Infof("Enriching %d issues with project data", len(issues))

	for _, issue := range issues {
		customFields, err := c.FetchIssueProjectData(ctx, owner, repo, issue.Number)
		if err != nil {
			logger.Warnf("Failed to fetch project data for issue #%d: %v", issue.Number, err)
			// Continue with next issue instead of failing completely
			continue
		}

		if len(customFields) > 0 {
			issue.CustomFields = customFields
		}
	}

	logger.Infof("Enrichment complete")
	return nil
}

// EnrichPullRequestsWithProjectData fetches and adds custom field data to pull requests
func (c *Client) EnrichPullRequestsWithProjectData(ctx context.Context, owner, repo string, prs []*PullRequest) error {
	logger.Infof("Enriching %d pull requests with project data", len(prs))

	for _, pr := range prs {
		customFields, err := c.FetchPullRequestProjectData(ctx, owner, repo, pr.Number)
		if err != nil {
			logger.Warnf("Failed to fetch project data for PR #%d: %v", pr.Number, err)
			// Continue with next PR instead of failing completely
			continue
		}

		if len(customFields) > 0 {
			pr.CustomFields = customFields
		}
	}

	logger.Infof("Enrichment complete")
	return nil
}
