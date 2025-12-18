// Package api provides HTTP client for LinkFlow API
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client is the LinkFlow API client
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Token      string
	APIKey     string

	// Service clients
	Workflows    *WorkflowsClient
	Executions   *ExecutionsClient
	Schedules    *SchedulesClient
	Credentials  *CredentialsClient
	Users        *UsersClient
	Webhooks     *WebhooksClient
}

// ClientOption configures the client
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.HTTPClient = client
	}
}

// WithToken sets the JWT token
func WithToken(token string) ClientOption {
	return func(c *Client) {
		c.Token = token
	}
}

// WithAPIKey sets the API key
func WithAPIKey(apiKey string) ClientOption {
	return func(c *Client) {
		c.APIKey = apiKey
	}
}

// WithTimeout sets the HTTP client timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.HTTPClient.Timeout = timeout
	}
}

// NewClient creates a new LinkFlow API client
func NewClient(baseURL string, opts ...ClientOption) *Client {
	c := &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	// Initialize service clients
	c.Workflows = &WorkflowsClient{client: c}
	c.Executions = &ExecutionsClient{client: c}
	c.Schedules = &SchedulesClient{client: c}
	c.Credentials = &CredentialsClient{client: c}
	c.Users = &UsersClient{client: c}
	c.Webhooks = &WebhooksClient{client: c}

	return c
}

// Request makes an HTTP request
func (c *Client) Request(ctx context.Context, method, path string, body, result interface{}) error {
	u, err := url.Parse(c.BaseURL + path)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	} else if c.APIKey != "" {
		req.Header.Set("X-API-Key", c.APIKey)
	}

	// Execute request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.ErrorInfo.Message != "" {
			return &apiErr
		}
		return fmt.Errorf("API error: %d %s", resp.StatusCode, string(respBody))
	}

	// Parse result
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// APIError represents an API error response
type APIError struct {
	ErrorInfo struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.ErrorInfo.Code, e.ErrorInfo.Message)
}

// ListOptions contains common list parameters
type ListOptions struct {
	Page   int
	Limit  int
	Sort   string
	Filter map[string]string
}

// ToQuery converts options to query string
func (o *ListOptions) ToQuery() string {
	v := url.Values{}
	if o.Page > 0 {
		v.Set("page", fmt.Sprintf("%d", o.Page))
	}
	if o.Limit > 0 {
		v.Set("limit", fmt.Sprintf("%d", o.Limit))
	}
	if o.Sort != "" {
		v.Set("sort", o.Sort)
	}
	for key, val := range o.Filter {
		v.Set(key, val)
	}
	if len(v) > 0 {
		return "?" + v.Encode()
	}
	return ""
}

// ListResponse is a generic list response
type ListResponse[T any] struct {
	Items []T   `json:"items"`
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
}

// WorkflowsClient handles workflow operations
type WorkflowsClient struct {
	client *Client
}

// Workflow represents a workflow
type Workflow struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Status      string                 `json:"status"`
	Nodes       []map[string]interface{} `json:"nodes"`
	Connections []map[string]interface{} `json:"connections"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

// List returns a list of workflows
func (c *WorkflowsClient) List(ctx context.Context, opts *ListOptions) (*ListResponse[Workflow], error) {
	query := ""
	if opts != nil {
		query = opts.ToQuery()
	}
	var result ListResponse[Workflow]
	err := c.client.Request(ctx, http.MethodGet, "/api/v1/workflows"+query, nil, &result)
	return &result, err
}

// Get returns a workflow by ID
func (c *WorkflowsClient) Get(ctx context.Context, id string) (*Workflow, error) {
	var result Workflow
	err := c.client.Request(ctx, http.MethodGet, "/api/v1/workflows/"+id, nil, &result)
	return &result, err
}

// Create creates a new workflow
func (c *WorkflowsClient) Create(ctx context.Context, workflow *Workflow) (*Workflow, error) {
	var result Workflow
	err := c.client.Request(ctx, http.MethodPost, "/api/v1/workflows", workflow, &result)
	return &result, err
}

// Update updates a workflow
func (c *WorkflowsClient) Update(ctx context.Context, id string, workflow *Workflow) (*Workflow, error) {
	var result Workflow
	err := c.client.Request(ctx, http.MethodPut, "/api/v1/workflows/"+id, workflow, &result)
	return &result, err
}

// Delete deletes a workflow
func (c *WorkflowsClient) Delete(ctx context.Context, id string) error {
	return c.client.Request(ctx, http.MethodDelete, "/api/v1/workflows/"+id, nil, nil)
}

// Execute starts a workflow execution
func (c *WorkflowsClient) Execute(ctx context.Context, id string, input map[string]interface{}) (*Execution, error) {
	var result Execution
	err := c.client.Request(ctx, http.MethodPost, "/api/v1/workflows/"+id+"/execute", input, &result)
	return &result, err
}

// ExecutionsClient handles execution operations
type ExecutionsClient struct {
	client *Client
}

// Execution represents a workflow execution
type Execution struct {
	ID          string                 `json:"id"`
	WorkflowID  string                 `json:"workflowId"`
	Status      string                 `json:"status"`
	TriggerType string                 `json:"triggerType"`
	InputData   map[string]interface{} `json:"inputData"`
	OutputData  map[string]interface{} `json:"outputData"`
	StartedAt   *time.Time             `json:"startedAt"`
	CompletedAt *time.Time             `json:"completedAt"`
	Duration    int64                  `json:"duration"`
	Error       string                 `json:"error,omitempty"`
}

// List returns a list of executions
func (c *ExecutionsClient) List(ctx context.Context, opts *ListOptions) (*ListResponse[Execution], error) {
	query := ""
	if opts != nil {
		query = opts.ToQuery()
	}
	var result ListResponse[Execution]
	err := c.client.Request(ctx, http.MethodGet, "/api/v1/executions"+query, nil, &result)
	return &result, err
}

// Get returns an execution by ID
func (c *ExecutionsClient) Get(ctx context.Context, id string) (*Execution, error) {
	var result Execution
	err := c.client.Request(ctx, http.MethodGet, "/api/v1/executions/"+id, nil, &result)
	return &result, err
}

// Cancel cancels an execution
func (c *ExecutionsClient) Cancel(ctx context.Context, id string) (*Execution, error) {
	var result Execution
	err := c.client.Request(ctx, http.MethodPost, "/api/v1/executions/"+id+"/cancel", nil, &result)
	return &result, err
}

// Retry retries a failed execution
func (c *ExecutionsClient) Retry(ctx context.Context, id string) (*Execution, error) {
	var result Execution
	err := c.client.Request(ctx, http.MethodPost, "/api/v1/executions/"+id+"/retry", nil, &result)
	return &result, err
}

// SchedulesClient handles schedule operations
type SchedulesClient struct {
	client *Client
}

// Schedule represents a workflow schedule
type Schedule struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	WorkflowID     string     `json:"workflowId"`
	CronExpression string     `json:"cronExpression"`
	Timezone       string     `json:"timezone"`
	Status         string     `json:"status"`
	NextRunAt      *time.Time `json:"nextRunAt"`
	LastRunAt      *time.Time `json:"lastRunAt"`
}

// List returns a list of schedules
func (c *SchedulesClient) List(ctx context.Context, opts *ListOptions) (*ListResponse[Schedule], error) {
	query := ""
	if opts != nil {
		query = opts.ToQuery()
	}
	var result ListResponse[Schedule]
	err := c.client.Request(ctx, http.MethodGet, "/api/v1/schedules"+query, nil, &result)
	return &result, err
}

// Create creates a new schedule
func (c *SchedulesClient) Create(ctx context.Context, schedule *Schedule) (*Schedule, error) {
	var result Schedule
	err := c.client.Request(ctx, http.MethodPost, "/api/v1/schedules", schedule, &result)
	return &result, err
}

// Delete deletes a schedule
func (c *SchedulesClient) Delete(ctx context.Context, id string) error {
	return c.client.Request(ctx, http.MethodDelete, "/api/v1/schedules/"+id, nil, nil)
}

// CredentialsClient handles credential operations
type CredentialsClient struct {
	client *Client
}

// Credential represents a credential
type Credential struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Service string `json:"service"`
	Status  string `json:"status"`
}

// List returns a list of credentials
func (c *CredentialsClient) List(ctx context.Context, opts *ListOptions) (*ListResponse[Credential], error) {
	query := ""
	if opts != nil {
		query = opts.ToQuery()
	}
	var result ListResponse[Credential]
	err := c.client.Request(ctx, http.MethodGet, "/api/v1/credentials"+query, nil, &result)
	return &result, err
}

// UsersClient handles user operations
type UsersClient struct {
	client *Client
}

// User represents a user
type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// Me returns the current user
func (c *UsersClient) Me(ctx context.Context) (*User, error) {
	var result User
	err := c.client.Request(ctx, http.MethodGet, "/api/v1/users/me", nil, &result)
	return &result, err
}

// WebhooksClient handles webhook operations
type WebhooksClient struct {
	client *Client
}

// Webhook represents a webhook
type Webhook struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	Events   []string `json:"events"`
	Status   string `json:"status"`
	Secret   string `json:"secret,omitempty"`
}

// List returns a list of webhooks
func (c *WebhooksClient) List(ctx context.Context, opts *ListOptions) (*ListResponse[Webhook], error) {
	query := ""
	if opts != nil {
		query = opts.ToQuery()
	}
	var result ListResponse[Webhook]
	err := c.client.Request(ctx, http.MethodGet, "/api/v1/webhooks"+query, nil, &result)
	return &result, err
}

// Create creates a new webhook
func (c *WebhooksClient) Create(ctx context.Context, webhook *Webhook) (*Webhook, error) {
	var result Webhook
	err := c.client.Request(ctx, http.MethodPost, "/api/v1/webhooks", webhook, &result)
	return &result, err
}
