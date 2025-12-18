// Package sdk provides a Go client library for LinkFlow AI API
package sdk

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

// Client is the LinkFlow AI API client
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	token      string
	
	// Service clients
	Auth         *AuthService
	Workflows    *WorkflowService
	Executions   *ExecutionService
}

// ClientOption is a function that configures a Client
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithAPIKey sets the API key for authentication
func WithAPIKey(apiKey string) ClientOption {
	return func(c *Client) {
		c.apiKey = apiKey
	}
}

// WithToken sets the JWT token for authentication
func WithToken(token string) ClientOption {
	return func(c *Client) {
		c.token = token
	}
}

// NewClient creates a new LinkFlow AI API client
func NewClient(baseURL string, opts ...ClientOption) *Client {
	c := &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(c)
	}

	// Initialize service clients
	c.Auth = &AuthService{client: c}
	c.Workflows = &WorkflowService{client: c}
	c.Executions = &ExecutionService{client: c}

	return c
}

// request makes an HTTP request to the API
func (c *Client) request(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}
	u.Path = path

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return nil, err
	}

	// Set headers
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Set authentication
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	} else if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	return c.httpClient.Do(req)
}

// decodeResponse decodes the JSON response body
func (c *Client) decodeResponse(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return fmt.Errorf("API error: %d", resp.StatusCode)
		}
		return &errResp
	}

	if v != nil {
		return json.NewDecoder(resp.Body).Decode(v)
	}

	return nil
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func (e *ErrorResponse) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// AuthService handles authentication operations
type AuthService struct {
	client *Client
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(ctx context.Context, email, password string) (*AuthResponse, error) {
	req := LoginRequest{
		Email:    email,
		Password: password,
	}

	resp, err := s.client.request(ctx, "POST", "/api/v1/auth/login", req)
	if err != nil {
		return nil, err
	}

	var authResp AuthResponse
	if err := s.client.decodeResponse(resp, &authResp); err != nil {
		return nil, err
	}

	// Set token for future requests
	s.client.token = authResp.Tokens.AccessToken

	return &authResp, nil
}

// Register creates a new user account
func (s *AuthService) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	resp, err := s.client.request(ctx, "POST", "/api/v1/auth/register", req)
	if err != nil {
		return nil, err
	}

	var authResp AuthResponse
	if err := s.client.decodeResponse(resp, &authResp); err != nil {
		return nil, err
	}

	// Set token for future requests
	s.client.token = authResp.Tokens.AccessToken

	return &authResp, nil
}

// Logout logs out the current user
func (s *AuthService) Logout(ctx context.Context) error {
	resp, err := s.client.request(ctx, "POST", "/api/v1/auth/logout", nil)
	if err != nil {
		return err
	}

	return s.client.decodeResponse(resp, nil)
}

// GetCurrentUser gets the current authenticated user
func (s *AuthService) GetCurrentUser(ctx context.Context) (*User, error) {
	resp, err := s.client.request(ctx, "GET", "/api/v1/auth/me", nil)
	if err != nil {
		return nil, err
	}

	var user User
	if err := s.client.decodeResponse(resp, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// WorkflowService handles workflow operations
type WorkflowService struct {
	client *Client
}

// Create creates a new workflow
func (s *WorkflowService) Create(ctx context.Context, req *CreateWorkflowRequest) (*Workflow, error) {
	resp, err := s.client.request(ctx, "POST", "/api/v1/workflows", req)
	if err != nil {
		return nil, err
	}

	var workflow Workflow
	if err := s.client.decodeResponse(resp, &workflow); err != nil {
		return nil, err
	}

	return &workflow, nil
}

// Get retrieves a workflow by ID
func (s *WorkflowService) Get(ctx context.Context, id string) (*Workflow, error) {
	resp, err := s.client.request(ctx, "GET", fmt.Sprintf("/api/v1/workflows/%s", id), nil)
	if err != nil {
		return nil, err
	}

	var workflow Workflow
	if err := s.client.decodeResponse(resp, &workflow); err != nil {
		return nil, err
	}

	return &workflow, nil
}

// List retrieves a list of workflows
func (s *WorkflowService) List(ctx context.Context, opts *ListOptions) (*WorkflowList, error) {
	path := "/api/v1/workflows"
	if opts != nil {
		params := url.Values{}
		if opts.Page > 0 {
			params.Set("page", fmt.Sprintf("%d", opts.Page))
		}
		if opts.PageSize > 0 {
			params.Set("pageSize", fmt.Sprintf("%d", opts.PageSize))
		}
		if opts.Status != "" {
			params.Set("status", opts.Status)
		}
		if len(params) > 0 {
			path += "?" + params.Encode()
		}
	}

	resp, err := s.client.request(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var list WorkflowList
	if err := s.client.decodeResponse(resp, &list); err != nil {
		return nil, err
	}

	return &list, nil
}

// Update updates an existing workflow
func (s *WorkflowService) Update(ctx context.Context, id string, req *UpdateWorkflowRequest) (*Workflow, error) {
	resp, err := s.client.request(ctx, "PUT", fmt.Sprintf("/api/v1/workflows/%s", id), req)
	if err != nil {
		return nil, err
	}

	var workflow Workflow
	if err := s.client.decodeResponse(resp, &workflow); err != nil {
		return nil, err
	}

	return &workflow, nil
}

// Delete deletes a workflow
func (s *WorkflowService) Delete(ctx context.Context, id string) error {
	resp, err := s.client.request(ctx, "DELETE", fmt.Sprintf("/api/v1/workflows/%s", id), nil)
	if err != nil {
		return err
	}

	return s.client.decodeResponse(resp, nil)
}

// Activate activates a workflow
func (s *WorkflowService) Activate(ctx context.Context, id string) (*Workflow, error) {
	resp, err := s.client.request(ctx, "POST", fmt.Sprintf("/api/v1/workflows/%s/activate", id), nil)
	if err != nil {
		return nil, err
	}

	var workflow Workflow
	if err := s.client.decodeResponse(resp, &workflow); err != nil {
		return nil, err
	}

	return &workflow, nil
}

// Deactivate deactivates a workflow
func (s *WorkflowService) Deactivate(ctx context.Context, id string) (*Workflow, error) {
	resp, err := s.client.request(ctx, "POST", fmt.Sprintf("/api/v1/workflows/%s/deactivate", id), nil)
	if err != nil {
		return nil, err
	}

	var workflow Workflow
	if err := s.client.decodeResponse(resp, &workflow); err != nil {
		return nil, err
	}

	return &workflow, nil
}

// Execute executes a workflow
func (s *WorkflowService) Execute(ctx context.Context, id string, req *ExecuteWorkflowRequest) (*ExecutionResult, error) {
	resp, err := s.client.request(ctx, "POST", fmt.Sprintf("/api/v1/workflows/%s/execute", id), req)
	if err != nil {
		return nil, err
	}

	var result ExecutionResult
	if err := s.client.decodeResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ExecutionService handles execution operations
type ExecutionService struct {
	client *Client
}

// Get retrieves an execution by ID
func (s *ExecutionService) Get(ctx context.Context, id string) (*Execution, error) {
	resp, err := s.client.request(ctx, "GET", fmt.Sprintf("/api/v1/executions/%s", id), nil)
	if err != nil {
		return nil, err
	}

	var execution Execution
	if err := s.client.decodeResponse(resp, &execution); err != nil {
		return nil, err
	}

	return &execution, nil
}

// Cancel cancels a running execution
func (s *ExecutionService) Cancel(ctx context.Context, id string) error {
	resp, err := s.client.request(ctx, "POST", fmt.Sprintf("/api/v1/executions/%s/cancel", id), nil)
	if err != nil {
		return err
	}

	return s.client.decodeResponse(resp, nil)
}

// The remaining service implementations would follow the same pattern...
