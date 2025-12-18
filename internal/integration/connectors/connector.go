// Package connectors provides integration connectors for third-party services
package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Connector defines the interface for all integration connectors
type Connector interface {
	// Name returns the connector name
	Name() string
	
	// Type returns the connector type
	Type() string
	
	// Operations returns supported operations
	Operations() []Operation
	
	// Execute executes an operation
	Execute(ctx context.Context, operation string, params map[string]interface{}, credentials map[string]interface{}) (map[string]interface{}, error)
	
	// TestConnection tests the connection
	TestConnection(ctx context.Context, credentials map[string]interface{}) error
}

// Operation represents a connector operation
type Operation struct {
	Name        string
	Description string
	Parameters  []Parameter
	Returns     []Parameter
}

// Parameter represents an operation parameter
type Parameter struct {
	Name        string
	Type        string
	Description string
	Required    bool
	Default     interface{}
}

// BaseConnector provides common functionality
type BaseConnector struct {
	httpClient *http.Client
}

// NewBaseConnector creates a new base connector
func NewBaseConnector() *BaseConnector {
	return &BaseConnector{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// DoRequest performs an HTTP request
func (c *BaseConnector) DoRequest(ctx context.Context, method, url string, body io.Reader, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(data))
	}

	return data, nil
}

// ConnectorRegistry holds all registered connectors
type ConnectorRegistry struct {
	connectors map[string]Connector
}

// NewConnectorRegistry creates a new connector registry
func NewConnectorRegistry() *ConnectorRegistry {
	return &ConnectorRegistry{
		connectors: make(map[string]Connector),
	}
}

// Register registers a connector
func (r *ConnectorRegistry) Register(connector Connector) {
	r.connectors[connector.Type()] = connector
}

// Get returns a connector by type
func (r *ConnectorRegistry) Get(connectorType string) (Connector, bool) {
	c, ok := r.connectors[connectorType]
	return c, ok
}

// List returns all registered connectors
func (r *ConnectorRegistry) List() []string {
	types := make([]string, 0, len(r.connectors))
	for t := range r.connectors {
		types = append(types, t)
	}
	return types
}

// GoogleSheetsConnector provides Google Sheets integration
type GoogleSheetsConnector struct {
	*BaseConnector
}

// NewGoogleSheetsConnector creates a new Google Sheets connector
func NewGoogleSheetsConnector() *GoogleSheetsConnector {
	return &GoogleSheetsConnector{BaseConnector: NewBaseConnector()}
}

func (c *GoogleSheetsConnector) Name() string { return "Google Sheets" }
func (c *GoogleSheetsConnector) Type() string { return "google_sheets" }

func (c *GoogleSheetsConnector) Operations() []Operation {
	return []Operation{
		{
			Name:        "read",
			Description: "Read data from a spreadsheet",
			Parameters: []Parameter{
				{Name: "spreadsheetId", Type: "string", Required: true},
				{Name: "range", Type: "string", Required: true},
			},
		},
		{
			Name:        "write",
			Description: "Write data to a spreadsheet",
			Parameters: []Parameter{
				{Name: "spreadsheetId", Type: "string", Required: true},
				{Name: "range", Type: "string", Required: true},
				{Name: "values", Type: "array", Required: true},
			},
		},
		{
			Name:        "append",
			Description: "Append data to a spreadsheet",
			Parameters: []Parameter{
				{Name: "spreadsheetId", Type: "string", Required: true},
				{Name: "range", Type: "string", Required: true},
				{Name: "values", Type: "array", Required: true},
			},
		},
		{
			Name:        "clear",
			Description: "Clear data from a range",
			Parameters: []Parameter{
				{Name: "spreadsheetId", Type: "string", Required: true},
				{Name: "range", Type: "string", Required: true},
			},
		},
	}
}

func (c *GoogleSheetsConnector) Execute(ctx context.Context, operation string, params map[string]interface{}, credentials map[string]interface{}) (map[string]interface{}, error) {
	accessToken, _ := credentials["access_token"].(string)
	if accessToken == "" {
		return nil, fmt.Errorf("access_token required")
	}

	spreadsheetID, _ := params["spreadsheetId"].(string)
	rangeStr, _ := params["range"].(string)

	baseURL := "https://sheets.googleapis.com/v4/spreadsheets"
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
		"Content-Type":  "application/json",
	}

	switch operation {
	case "read":
		url := fmt.Sprintf("%s/%s/values/%s", baseURL, spreadsheetID, rangeStr)
		data, err := c.DoRequest(ctx, "GET", url, nil, headers)
		if err != nil {
			return nil, err
		}
		var result map[string]interface{}
		json.Unmarshal(data, &result)
		return result, nil

	case "write":
		url := fmt.Sprintf("%s/%s/values/%s?valueInputOption=USER_ENTERED", baseURL, spreadsheetID, rangeStr)
		values := params["values"]
		body := map[string]interface{}{"values": values}
		bodyBytes, _ := json.Marshal(body)
		data, err := c.DoRequest(ctx, "PUT", url, strings.NewReader(string(bodyBytes)), headers)
		if err != nil {
			return nil, err
		}
		var result map[string]interface{}
		json.Unmarshal(data, &result)
		return result, nil

	case "append":
		url := fmt.Sprintf("%s/%s/values/%s:append?valueInputOption=USER_ENTERED&insertDataOption=INSERT_ROWS", baseURL, spreadsheetID, rangeStr)
		values := params["values"]
		body := map[string]interface{}{"values": values}
		bodyBytes, _ := json.Marshal(body)
		data, err := c.DoRequest(ctx, "POST", url, strings.NewReader(string(bodyBytes)), headers)
		if err != nil {
			return nil, err
		}
		var result map[string]interface{}
		json.Unmarshal(data, &result)
		return result, nil

	case "clear":
		url := fmt.Sprintf("%s/%s/values/%s:clear", baseURL, spreadsheetID, rangeStr)
		data, err := c.DoRequest(ctx, "POST", url, strings.NewReader("{}"), headers)
		if err != nil {
			return nil, err
		}
		var result map[string]interface{}
		json.Unmarshal(data, &result)
		return result, nil

	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

func (c *GoogleSheetsConnector) TestConnection(ctx context.Context, credentials map[string]interface{}) error {
	accessToken, _ := credentials["access_token"].(string)
	if accessToken == "" {
		return fmt.Errorf("access_token required")
	}

	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}

	_, err := c.DoRequest(ctx, "GET", "https://sheets.googleapis.com/v4/spreadsheets", nil, headers)
	return err
}

// GitHubConnector provides GitHub integration
type GitHubConnector struct {
	*BaseConnector
}

// NewGitHubConnector creates a new GitHub connector
func NewGitHubConnector() *GitHubConnector {
	return &GitHubConnector{BaseConnector: NewBaseConnector()}
}

func (c *GitHubConnector) Name() string { return "GitHub" }
func (c *GitHubConnector) Type() string { return "github" }

func (c *GitHubConnector) Operations() []Operation {
	return []Operation{
		{Name: "list_repos", Description: "List repositories"},
		{Name: "get_repo", Description: "Get repository details", Parameters: []Parameter{{Name: "owner", Type: "string", Required: true}, {Name: "repo", Type: "string", Required: true}}},
		{Name: "list_issues", Description: "List issues", Parameters: []Parameter{{Name: "owner", Type: "string", Required: true}, {Name: "repo", Type: "string", Required: true}}},
		{Name: "create_issue", Description: "Create an issue", Parameters: []Parameter{{Name: "owner", Type: "string", Required: true}, {Name: "repo", Type: "string", Required: true}, {Name: "title", Type: "string", Required: true}, {Name: "body", Type: "string"}}},
		{Name: "list_prs", Description: "List pull requests", Parameters: []Parameter{{Name: "owner", Type: "string", Required: true}, {Name: "repo", Type: "string", Required: true}}},
		{Name: "create_pr", Description: "Create a pull request", Parameters: []Parameter{{Name: "owner", Type: "string", Required: true}, {Name: "repo", Type: "string", Required: true}, {Name: "title", Type: "string", Required: true}, {Name: "head", Type: "string", Required: true}, {Name: "base", Type: "string", Required: true}}},
	}
}

func (c *GitHubConnector) Execute(ctx context.Context, operation string, params map[string]interface{}, credentials map[string]interface{}) (map[string]interface{}, error) {
	token, _ := credentials["access_token"].(string)
	if token == "" {
		return nil, fmt.Errorf("access_token required")
	}

	baseURL := "https://api.github.com"
	headers := map[string]string{
		"Authorization": "Bearer " + token,
		"Accept":        "application/vnd.github+json",
		"Content-Type":  "application/json",
	}

	owner, _ := params["owner"].(string)
	repo, _ := params["repo"].(string)

	switch operation {
	case "list_repos":
		data, err := c.DoRequest(ctx, "GET", baseURL+"/user/repos", nil, headers)
		if err != nil {
			return nil, err
		}
		var repos []map[string]interface{}
		json.Unmarshal(data, &repos)
		return map[string]interface{}{"repositories": repos}, nil

	case "get_repo":
		url := fmt.Sprintf("%s/repos/%s/%s", baseURL, owner, repo)
		data, err := c.DoRequest(ctx, "GET", url, nil, headers)
		if err != nil {
			return nil, err
		}
		var result map[string]interface{}
		json.Unmarshal(data, &result)
		return result, nil

	case "list_issues":
		url := fmt.Sprintf("%s/repos/%s/%s/issues", baseURL, owner, repo)
		data, err := c.DoRequest(ctx, "GET", url, nil, headers)
		if err != nil {
			return nil, err
		}
		var issues []map[string]interface{}
		json.Unmarshal(data, &issues)
		return map[string]interface{}{"issues": issues}, nil

	case "create_issue":
		url := fmt.Sprintf("%s/repos/%s/%s/issues", baseURL, owner, repo)
		body := map[string]interface{}{
			"title": params["title"],
			"body":  params["body"],
		}
		bodyBytes, _ := json.Marshal(body)
		data, err := c.DoRequest(ctx, "POST", url, strings.NewReader(string(bodyBytes)), headers)
		if err != nil {
			return nil, err
		}
		var result map[string]interface{}
		json.Unmarshal(data, &result)
		return result, nil

	case "list_prs":
		url := fmt.Sprintf("%s/repos/%s/%s/pulls", baseURL, owner, repo)
		data, err := c.DoRequest(ctx, "GET", url, nil, headers)
		if err != nil {
			return nil, err
		}
		var prs []map[string]interface{}
		json.Unmarshal(data, &prs)
		return map[string]interface{}{"pull_requests": prs}, nil

	case "create_pr":
		url := fmt.Sprintf("%s/repos/%s/%s/pulls", baseURL, owner, repo)
		body := map[string]interface{}{
			"title": params["title"],
			"head":  params["head"],
			"base":  params["base"],
			"body":  params["body"],
		}
		bodyBytes, _ := json.Marshal(body)
		data, err := c.DoRequest(ctx, "POST", url, strings.NewReader(string(bodyBytes)), headers)
		if err != nil {
			return nil, err
		}
		var result map[string]interface{}
		json.Unmarshal(data, &result)
		return result, nil

	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

func (c *GitHubConnector) TestConnection(ctx context.Context, credentials map[string]interface{}) error {
	token, _ := credentials["access_token"].(string)
	if token == "" {
		return fmt.Errorf("access_token required")
	}

	headers := map[string]string{
		"Authorization": "Bearer " + token,
		"Accept":        "application/vnd.github+json",
	}

	_, err := c.DoRequest(ctx, "GET", "https://api.github.com/user", nil, headers)
	return err
}

// NotionConnector provides Notion integration
type NotionConnector struct {
	*BaseConnector
}

// NewNotionConnector creates a new Notion connector
func NewNotionConnector() *NotionConnector {
	return &NotionConnector{BaseConnector: NewBaseConnector()}
}

func (c *NotionConnector) Name() string { return "Notion" }
func (c *NotionConnector) Type() string { return "notion" }

func (c *NotionConnector) Operations() []Operation {
	return []Operation{
		{Name: "list_databases", Description: "List databases"},
		{Name: "query_database", Description: "Query a database", Parameters: []Parameter{{Name: "database_id", Type: "string", Required: true}, {Name: "filter", Type: "object"}}},
		{Name: "create_page", Description: "Create a page", Parameters: []Parameter{{Name: "parent", Type: "object", Required: true}, {Name: "properties", Type: "object", Required: true}}},
		{Name: "update_page", Description: "Update a page", Parameters: []Parameter{{Name: "page_id", Type: "string", Required: true}, {Name: "properties", Type: "object", Required: true}}},
		{Name: "get_page", Description: "Get a page", Parameters: []Parameter{{Name: "page_id", Type: "string", Required: true}}},
	}
}

func (c *NotionConnector) Execute(ctx context.Context, operation string, params map[string]interface{}, credentials map[string]interface{}) (map[string]interface{}, error) {
	token, _ := credentials["access_token"].(string)
	if token == "" {
		return nil, fmt.Errorf("access_token required")
	}

	baseURL := "https://api.notion.com/v1"
	headers := map[string]string{
		"Authorization":  "Bearer " + token,
		"Content-Type":   "application/json",
		"Notion-Version": "2022-06-28",
	}

	switch operation {
	case "list_databases":
		body := `{"filter":{"property":"object","value":"database"}}`
		data, err := c.DoRequest(ctx, "POST", baseURL+"/search", strings.NewReader(body), headers)
		if err != nil {
			return nil, err
		}
		var result map[string]interface{}
		json.Unmarshal(data, &result)
		return result, nil

	case "query_database":
		databaseID, _ := params["database_id"].(string)
		url := fmt.Sprintf("%s/databases/%s/query", baseURL, databaseID)
		body := map[string]interface{}{}
		if filter, ok := params["filter"]; ok {
			body["filter"] = filter
		}
		bodyBytes, _ := json.Marshal(body)
		data, err := c.DoRequest(ctx, "POST", url, strings.NewReader(string(bodyBytes)), headers)
		if err != nil {
			return nil, err
		}
		var result map[string]interface{}
		json.Unmarshal(data, &result)
		return result, nil

	case "create_page":
		body := map[string]interface{}{
			"parent":     params["parent"],
			"properties": params["properties"],
		}
		bodyBytes, _ := json.Marshal(body)
		data, err := c.DoRequest(ctx, "POST", baseURL+"/pages", strings.NewReader(string(bodyBytes)), headers)
		if err != nil {
			return nil, err
		}
		var result map[string]interface{}
		json.Unmarshal(data, &result)
		return result, nil

	case "update_page":
		pageID, _ := params["page_id"].(string)
		url := fmt.Sprintf("%s/pages/%s", baseURL, pageID)
		body := map[string]interface{}{
			"properties": params["properties"],
		}
		bodyBytes, _ := json.Marshal(body)
		data, err := c.DoRequest(ctx, "PATCH", url, strings.NewReader(string(bodyBytes)), headers)
		if err != nil {
			return nil, err
		}
		var result map[string]interface{}
		json.Unmarshal(data, &result)
		return result, nil

	case "get_page":
		pageID, _ := params["page_id"].(string)
		url := fmt.Sprintf("%s/pages/%s", baseURL, pageID)
		data, err := c.DoRequest(ctx, "GET", url, nil, headers)
		if err != nil {
			return nil, err
		}
		var result map[string]interface{}
		json.Unmarshal(data, &result)
		return result, nil

	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

func (c *NotionConnector) TestConnection(ctx context.Context, credentials map[string]interface{}) error {
	token, _ := credentials["access_token"].(string)
	if token == "" {
		return fmt.Errorf("access_token required")
	}

	headers := map[string]string{
		"Authorization":  "Bearer " + token,
		"Notion-Version": "2022-06-28",
	}

	_, err := c.DoRequest(ctx, "GET", "https://api.notion.com/v1/users/me", nil, headers)
	return err
}

// AirtableConnector provides Airtable integration
type AirtableConnector struct {
	*BaseConnector
}

// NewAirtableConnector creates a new Airtable connector
func NewAirtableConnector() *AirtableConnector {
	return &AirtableConnector{BaseConnector: NewBaseConnector()}
}

func (c *AirtableConnector) Name() string { return "Airtable" }
func (c *AirtableConnector) Type() string { return "airtable" }

func (c *AirtableConnector) Operations() []Operation {
	return []Operation{
		{Name: "list_bases", Description: "List bases"},
		{Name: "list_records", Description: "List records", Parameters: []Parameter{{Name: "base_id", Type: "string", Required: true}, {Name: "table_name", Type: "string", Required: true}}},
		{Name: "create_record", Description: "Create a record", Parameters: []Parameter{{Name: "base_id", Type: "string", Required: true}, {Name: "table_name", Type: "string", Required: true}, {Name: "fields", Type: "object", Required: true}}},
		{Name: "update_record", Description: "Update a record", Parameters: []Parameter{{Name: "base_id", Type: "string", Required: true}, {Name: "table_name", Type: "string", Required: true}, {Name: "record_id", Type: "string", Required: true}, {Name: "fields", Type: "object", Required: true}}},
		{Name: "delete_record", Description: "Delete a record", Parameters: []Parameter{{Name: "base_id", Type: "string", Required: true}, {Name: "table_name", Type: "string", Required: true}, {Name: "record_id", Type: "string", Required: true}}},
	}
}

func (c *AirtableConnector) Execute(ctx context.Context, operation string, params map[string]interface{}, credentials map[string]interface{}) (map[string]interface{}, error) {
	token, _ := credentials["access_token"].(string)
	if token == "" {
		return nil, fmt.Errorf("access_token required")
	}

	baseID, _ := params["base_id"].(string)
	tableName, _ := params["table_name"].(string)
	baseURL := fmt.Sprintf("https://api.airtable.com/v0/%s/%s", baseID, tableName)

	headers := map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	}

	switch operation {
	case "list_bases":
		data, err := c.DoRequest(ctx, "GET", "https://api.airtable.com/v0/meta/bases", nil, headers)
		if err != nil {
			return nil, err
		}
		var result map[string]interface{}
		json.Unmarshal(data, &result)
		return result, nil

	case "list_records":
		data, err := c.DoRequest(ctx, "GET", baseURL, nil, headers)
		if err != nil {
			return nil, err
		}
		var result map[string]interface{}
		json.Unmarshal(data, &result)
		return result, nil

	case "create_record":
		body := map[string]interface{}{
			"fields": params["fields"],
		}
		bodyBytes, _ := json.Marshal(body)
		data, err := c.DoRequest(ctx, "POST", baseURL, strings.NewReader(string(bodyBytes)), headers)
		if err != nil {
			return nil, err
		}
		var result map[string]interface{}
		json.Unmarshal(data, &result)
		return result, nil

	case "update_record":
		recordID, _ := params["record_id"].(string)
		url := fmt.Sprintf("%s/%s", baseURL, recordID)
		body := map[string]interface{}{
			"fields": params["fields"],
		}
		bodyBytes, _ := json.Marshal(body)
		data, err := c.DoRequest(ctx, "PATCH", url, strings.NewReader(string(bodyBytes)), headers)
		if err != nil {
			return nil, err
		}
		var result map[string]interface{}
		json.Unmarshal(data, &result)
		return result, nil

	case "delete_record":
		recordID, _ := params["record_id"].(string)
		url := fmt.Sprintf("%s/%s", baseURL, recordID)
		data, err := c.DoRequest(ctx, "DELETE", url, nil, headers)
		if err != nil {
			return nil, err
		}
		var result map[string]interface{}
		json.Unmarshal(data, &result)
		return result, nil

	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

func (c *AirtableConnector) TestConnection(ctx context.Context, credentials map[string]interface{}) error {
	token, _ := credentials["access_token"].(string)
	if token == "" {
		return fmt.Errorf("access_token required")
	}

	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}

	_, err := c.DoRequest(ctx, "GET", "https://api.airtable.com/v0/meta/bases", nil, headers)
	return err
}

// DefaultRegistry returns a registry with all default connectors
func DefaultRegistry() *ConnectorRegistry {
	registry := NewConnectorRegistry()
	registry.Register(NewGoogleSheetsConnector())
	registry.Register(NewGitHubConnector())
	registry.Register(NewNotionConnector())
	registry.Register(NewAirtableConnector())
	return registry
}
