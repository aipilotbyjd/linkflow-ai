// Package nodes provides Notion node implementation
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
	runtime.Register(&NotionNode{})
}

// NotionNode implements Notion operations
type NotionNode struct {
	client *http.Client
}

func (n *NotionNode) GetType() string { return "notion" }
func (n *NotionNode) Validate(config map[string]interface{}) error { return nil }

func (n *NotionNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "notion",
		Name:        "Notion",
		Description: "Interact with Notion databases and pages",
		Category:    "integration",
		Version:     "1.0.0",
		Icon:        "notion",
		Inputs:      []runtime.PortDefinition{{Name: "main", Type: "main"}},
		Outputs:     []runtime.PortDefinition{{Name: "main", Type: "main"}},
		Properties: []runtime.PropertyDefinition{
			{Name: "operation", Type: "select", Required: true, Options: []runtime.PropertyOption{
				{Label: "List Databases", Value: "listDatabases"}, {Label: "Query Database", Value: "queryDatabase"},
				{Label: "Get Page", Value: "getPage"}, {Label: "Create Page", Value: "createPage"},
				{Label: "Update Page", Value: "updatePage"}, {Label: "Search", Value: "search"},
			}},
			{Name: "databaseId", Type: "string"},
			{Name: "pageId", Type: "string"},
			{Name: "filter", Type: "json"},
			{Name: "properties", Type: "json"},
			{Name: "query", Type: "string"},
			{Name: "parent", Type: "json"},
		},
	}
}

func (n *NotionNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
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

	baseURL := "https://api.notion.com/v1"
	headers := map[string]string{
		"Authorization":  "Bearer " + accessToken,
		"Content-Type":   "application/json",
		"Notion-Version": "2022-06-28",
	}

	var result map[string]interface{}
	var err error

	switch operation {
	// Database operations
	case "listDatabases":
		body := map[string]interface{}{
			"filter": map[string]interface{}{
				"property": "object",
				"value":    "database",
			},
		}
		result, err = n.doRequest(ctx, "POST", baseURL+"/search", body, headers)
	case "queryDatabase":
		databaseID, _ := input.NodeConfig["databaseId"].(string)
		body := map[string]interface{}{}
		if filter, ok := input.NodeConfig["filter"].(map[string]interface{}); ok {
			body["filter"] = filter
		}
		if sorts, ok := input.NodeConfig["sorts"].([]interface{}); ok {
			body["sorts"] = sorts
		}
		result, err = n.doRequest(ctx, "POST", fmt.Sprintf("%s/databases/%s/query", baseURL, databaseID), body, headers)
	case "createDatabase":
		parent, _ := input.NodeConfig["parent"].(map[string]interface{})
		title, _ := input.NodeConfig["title"].(string)
		properties, _ := input.NodeConfig["properties"].(map[string]interface{})
		body := map[string]interface{}{
			"parent": parent,
			"title": []map[string]interface{}{
				{"type": "text", "text": map[string]interface{}{"content": title}},
			},
			"properties": properties,
		}
		result, err = n.doRequest(ctx, "POST", baseURL+"/databases", body, headers)

	// Page operations
	case "getPage":
		pageID, _ := input.NodeConfig["pageId"].(string)
		result, err = n.doRequest(ctx, "GET", fmt.Sprintf("%s/pages/%s", baseURL, pageID), nil, headers)
	case "createPage":
		parent, _ := input.NodeConfig["parent"].(map[string]interface{})
		properties, _ := input.NodeConfig["properties"].(map[string]interface{})
		children, _ := input.NodeConfig["children"].([]interface{})
		body := map[string]interface{}{
			"parent":     parent,
			"properties": properties,
		}
		if len(children) > 0 {
			body["children"] = children
		}
		result, err = n.doRequest(ctx, "POST", baseURL+"/pages", body, headers)
	case "updatePage":
		pageID, _ := input.NodeConfig["pageId"].(string)
		properties, _ := input.NodeConfig["properties"].(map[string]interface{})
		body := map[string]interface{}{
			"properties": properties,
		}
		result, err = n.doRequest(ctx, "PATCH", fmt.Sprintf("%s/pages/%s", baseURL, pageID), body, headers)
	case "archivePage":
		pageID, _ := input.NodeConfig["pageId"].(string)
		body := map[string]interface{}{
			"archived": true,
		}
		result, err = n.doRequest(ctx, "PATCH", fmt.Sprintf("%s/pages/%s", baseURL, pageID), body, headers)

	// Block operations
	case "getBlock":
		blockID, _ := input.NodeConfig["blockId"].(string)
		result, err = n.doRequest(ctx, "GET", fmt.Sprintf("%s/blocks/%s", baseURL, blockID), nil, headers)
	case "getBlockChildren":
		blockID, _ := input.NodeConfig["blockId"].(string)
		result, err = n.doRequest(ctx, "GET", fmt.Sprintf("%s/blocks/%s/children", baseURL, blockID), nil, headers)
	case "appendBlockChildren":
		blockID, _ := input.NodeConfig["blockId"].(string)
		children, _ := input.NodeConfig["children"].([]interface{})
		body := map[string]interface{}{
			"children": children,
		}
		result, err = n.doRequest(ctx, "PATCH", fmt.Sprintf("%s/blocks/%s/children", baseURL, blockID), body, headers)
	case "deleteBlock":
		blockID, _ := input.NodeConfig["blockId"].(string)
		result, err = n.doRequest(ctx, "DELETE", fmt.Sprintf("%s/blocks/%s", baseURL, blockID), nil, headers)

	// Search
	case "search":
		query, _ := input.NodeConfig["query"].(string)
		body := map[string]interface{}{}
		if query != "" {
			body["query"] = query
		}
		if filter, ok := input.NodeConfig["filter"].(map[string]interface{}); ok {
			body["filter"] = filter
		}
		result, err = n.doRequest(ctx, "POST", baseURL+"/search", body, headers)

	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}

	if err != nil {
		return &runtime.ExecutionOutput{Error: err}, nil
	}

	return &runtime.ExecutionOutput{Data: result}, nil
}

func (n *NotionNode) doRequest(ctx context.Context, method, url string, body interface{}, headers map[string]string) (map[string]interface{}, error) {
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
