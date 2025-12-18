// Package nodes provides GitHub node implementation
package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/node/runtime"
)

func init() {
	runtime.Register(&GitHubNode{})
}

// GitHubNode implements GitHub operations
type GitHubNode struct {
	client *http.Client
}

func (n *GitHubNode) GetType() string { return "github" }
func (n *GitHubNode) Validate(config map[string]interface{}) error { return nil }

func (n *GitHubNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "github",
		Name:        "GitHub",
		Description: "Interact with GitHub repositories, issues, and pull requests",
		Category:    "integration",
		Version:     "1.0.0",
		Icon:        "github",
		Inputs:      []runtime.PortDefinition{{Name: "main", Type: "main"}},
		Outputs:     []runtime.PortDefinition{{Name: "main", Type: "main"}},
		Properties: []runtime.PropertyDefinition{
			{Name: "operation", Type: "select", Required: true, Options: []runtime.PropertyOption{
				{Label: "List Repos", Value: "listRepos"}, {Label: "Get Repo", Value: "getRepo"},
				{Label: "List Issues", Value: "listIssues"}, {Label: "Create Issue", Value: "createIssue"},
				{Label: "List PRs", Value: "listPRs"}, {Label: "Create PR", Value: "createPR"},
			}},
			{Name: "owner", Type: "string"},
			{Name: "repo", Type: "string"},
			{Name: "issueNumber", Type: "number"},
			{Name: "prNumber", Type: "number"},
			{Name: "title", Type: "string"},
			{Name: "body", Type: "string"},
			{Name: "head", Type: "string"},
			{Name: "base", Type: "string"},
			{Name: "path", Type: "string"},
			{Name: "sha", Type: "string"},
		},
	}
}

func (n *GitHubNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	if n.client == nil {
		n.client = &http.Client{Timeout: 30 * time.Second}
	}

	operation, _ := input.NodeConfig["operation"].(string)
	accessToken, _ := input.Credentials["access_token"].(string)
	if accessToken == "" {
		accessToken, _ = input.Credentials["token"].(string)
	}

	if accessToken == "" {
		return nil, fmt.Errorf("access_token or token required")
	}

	owner, _ := input.NodeConfig["owner"].(string)
	repo, _ := input.NodeConfig["repo"].(string)
	baseURL := "https://api.github.com"
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
		"Accept":        "application/vnd.github+json",
		"Content-Type":  "application/json",
	}

	var result map[string]interface{}
	var err error

	switch operation {
	// Repository operations
	case "listRepos":
		result, err = n.doRequest(ctx, "GET", baseURL+"/user/repos", nil, headers)
	case "getRepo":
		result, err = n.doRequest(ctx, "GET", fmt.Sprintf("%s/repos/%s/%s", baseURL, owner, repo), nil, headers)
	case "createRepo":
		body := map[string]interface{}{
			"name":        input.NodeConfig["title"],
			"description": input.NodeConfig["body"],
			"private":     input.NodeConfig["private"],
		}
		result, err = n.doRequest(ctx, "POST", baseURL+"/user/repos", body, headers)

	// Issue operations
	case "listIssues":
		result, err = n.doRequest(ctx, "GET", fmt.Sprintf("%s/repos/%s/%s/issues", baseURL, owner, repo), nil, headers)
	case "getIssue":
		issueNum := int(input.NodeConfig["issueNumber"].(float64))
		result, err = n.doRequest(ctx, "GET", fmt.Sprintf("%s/repos/%s/%s/issues/%d", baseURL, owner, repo, issueNum), nil, headers)
	case "createIssue":
		body := map[string]interface{}{
			"title": input.NodeConfig["title"],
			"body":  input.NodeConfig["body"],
		}
		if labels, ok := input.NodeConfig["labels"].([]interface{}); ok {
			body["labels"] = labels
		}
		if assignees, ok := input.NodeConfig["assignees"].([]interface{}); ok {
			body["assignees"] = assignees
		}
		result, err = n.doRequest(ctx, "POST", fmt.Sprintf("%s/repos/%s/%s/issues", baseURL, owner, repo), body, headers)
	case "updateIssue":
		issueNum := int(input.NodeConfig["issueNumber"].(float64))
		body := map[string]interface{}{}
		if title, ok := input.NodeConfig["title"].(string); ok && title != "" {
			body["title"] = title
		}
		if bodyText, ok := input.NodeConfig["body"].(string); ok && bodyText != "" {
			body["body"] = bodyText
		}
		result, err = n.doRequest(ctx, "PATCH", fmt.Sprintf("%s/repos/%s/%s/issues/%d", baseURL, owner, repo, issueNum), body, headers)
	case "closeIssue":
		issueNum := int(input.NodeConfig["issueNumber"].(float64))
		body := map[string]interface{}{"state": "closed"}
		result, err = n.doRequest(ctx, "PATCH", fmt.Sprintf("%s/repos/%s/%s/issues/%d", baseURL, owner, repo, issueNum), body, headers)

	// PR operations
	case "listPRs":
		result, err = n.doRequest(ctx, "GET", fmt.Sprintf("%s/repos/%s/%s/pulls", baseURL, owner, repo), nil, headers)
	case "getPR":
		prNum := int(input.NodeConfig["prNumber"].(float64))
		result, err = n.doRequest(ctx, "GET", fmt.Sprintf("%s/repos/%s/%s/pulls/%d", baseURL, owner, repo, prNum), nil, headers)
	case "createPR":
		body := map[string]interface{}{
			"title": input.NodeConfig["title"],
			"body":  input.NodeConfig["body"],
			"head":  input.NodeConfig["head"],
			"base":  input.NodeConfig["base"],
		}
		result, err = n.doRequest(ctx, "POST", fmt.Sprintf("%s/repos/%s/%s/pulls", baseURL, owner, repo), body, headers)
	case "mergePR":
		prNum := int(input.NodeConfig["prNumber"].(float64))
		body := map[string]interface{}{
			"commit_message": input.NodeConfig["message"],
		}
		result, err = n.doRequest(ctx, "PUT", fmt.Sprintf("%s/repos/%s/%s/pulls/%d/merge", baseURL, owner, repo, prNum), body, headers)

	// Commit operations
	case "listCommits":
		result, err = n.doRequest(ctx, "GET", fmt.Sprintf("%s/repos/%s/%s/commits", baseURL, owner, repo), nil, headers)
	case "getCommit":
		sha, _ := input.NodeConfig["sha"].(string)
		result, err = n.doRequest(ctx, "GET", fmt.Sprintf("%s/repos/%s/%s/commits/%s", baseURL, owner, repo, sha), nil, headers)

	// Branch operations
	case "listBranches":
		result, err = n.doRequest(ctx, "GET", fmt.Sprintf("%s/repos/%s/%s/branches", baseURL, owner, repo), nil, headers)
	case "createBranch":
		sha, _ := input.NodeConfig["sha"].(string)
		branchName, _ := input.NodeConfig["title"].(string)
		body := map[string]interface{}{
			"ref": "refs/heads/" + branchName,
			"sha": sha,
		}
		result, err = n.doRequest(ctx, "POST", fmt.Sprintf("%s/repos/%s/%s/git/refs", baseURL, owner, repo), body, headers)

	// File operations
	case "getFile":
		path, _ := input.NodeConfig["path"].(string)
		result, err = n.doRequest(ctx, "GET", fmt.Sprintf("%s/repos/%s/%s/contents/%s", baseURL, owner, repo, path), nil, headers)
	case "createFile", "updateFile":
		path, _ := input.NodeConfig["path"].(string)
		content, _ := input.NodeConfig["content"].(string)
		message, _ := input.NodeConfig["message"].(string)
		body := map[string]interface{}{
			"message": message,
			"content": content,
		}
		if sha, ok := input.NodeConfig["sha"].(string); ok && sha != "" {
			body["sha"] = sha
		}
		result, err = n.doRequest(ctx, "PUT", fmt.Sprintf("%s/repos/%s/%s/contents/%s", baseURL, owner, repo, path), body, headers)

	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}

	if err != nil {
		return &runtime.ExecutionOutput{Error: err}, nil
	}

	return &runtime.ExecutionOutput{Data: result}, nil
}

func (n *GitHubNode) doRequest(ctx context.Context, method, url string, body interface{}, headers map[string]string) (map[string]interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		bodyReader = strings.NewReader(string(bodyBytes))
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed: %s", string(data))
	}

	var result map[string]interface{}
	json.Unmarshal(data, &result)
	return result, nil
}
