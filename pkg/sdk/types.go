package sdk

import "time"

// Common types used by the SDK

// ListOptions specifies options for listing resources
type ListOptions struct {
	Page     int
	PageSize int
	Status   string
	Tags     []string
	SortBy   string
	SortOrder string
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	User   User   `json:"user"`
	Tokens Tokens `json:"tokens"`
}

// Tokens contains authentication tokens
type Tokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"`
}

// User represents a user
type User struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	FirstName      string    `json:"firstName"`
	LastName       string    `json:"lastName"`
	OrganizationID string    `json:"organizationId"`
	Roles          []string  `json:"roles"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	RememberMe bool   `json:"rememberMe"`
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email            string `json:"email"`
	Password         string `json:"password"`
	FirstName        string `json:"firstName"`
	LastName         string `json:"lastName"`
	OrganizationName string `json:"organizationName,omitempty"`
}

// Workflow represents a workflow
type Workflow struct {
	ID          string            `json:"id"`
	UserID      string            `json:"userId"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Status      string            `json:"status"`
	Version     int               `json:"version"`
	Nodes       []Node            `json:"nodes"`
	Connections []Connection      `json:"connections"`
	Settings    WorkflowSettings  `json:"settings"`
	Tags        []string          `json:"tags"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

// Node represents a workflow node
type Node struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
	Position    Position               `json:"position"`
}

// Position represents a node position
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Connection represents a connection between nodes
type Connection struct {
	ID           string `json:"id"`
	SourceNodeID string `json:"sourceNodeId"`
	TargetNodeID string `json:"targetNodeId"`
	SourcePort   string `json:"sourcePort,omitempty"`
	TargetPort   string `json:"targetPort,omitempty"`
}

// WorkflowSettings contains workflow settings
type WorkflowSettings struct {
	MaxExecutionTime int                    `json:"maxExecutionTime"`
	RetryPolicy      RetryPolicy            `json:"retryPolicy"`
	ErrorHandling    string                 `json:"errorHandling"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxAttempts int    `json:"maxAttempts"`
	BackoffType string `json:"backoffType"`
	Delay       int    `json:"delaySeconds"`
}

// CreateWorkflowRequest represents a request to create a workflow
type CreateWorkflowRequest struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Nodes       []Node           `json:"nodes,omitempty"`
	Connections []Connection     `json:"connections,omitempty"`
	Settings    WorkflowSettings `json:"settings,omitempty"`
	Tags        []string         `json:"tags,omitempty"`
}

// UpdateWorkflowRequest represents a request to update a workflow
type UpdateWorkflowRequest struct {
	Name        string           `json:"name,omitempty"`
	Description string           `json:"description,omitempty"`
	Nodes       []Node           `json:"nodes,omitempty"`
	Connections []Connection     `json:"connections,omitempty"`
	Settings    WorkflowSettings `json:"settings,omitempty"`
	Tags        []string         `json:"tags,omitempty"`
}

// WorkflowList represents a list of workflows
type WorkflowList struct {
	Workflows  []Workflow `json:"workflows"`
	TotalCount int        `json:"totalCount"`
	Page       int        `json:"page"`
	PageSize   int        `json:"pageSize"`
}

// ExecuteWorkflowRequest represents a request to execute a workflow
type ExecuteWorkflowRequest struct {
	Input   map[string]interface{} `json:"input,omitempty"`
	Context map[string]string      `json:"context,omitempty"`
	Async   bool                   `json:"async"`
}

// ExecutionResult represents the result of a workflow execution
type ExecutionResult struct {
	ExecutionID string                 `json:"executionId"`
	Status      string                 `json:"status"`
	Output      map[string]interface{} `json:"output,omitempty"`
	Errors      []ExecutionError       `json:"errors,omitempty"`
	StartedAt   time.Time              `json:"startedAt"`
	CompletedAt *time.Time             `json:"completedAt,omitempty"`
}

// Execution represents a workflow execution
type Execution struct {
	ID          string                 `json:"id"`
	WorkflowID  string                 `json:"workflowId"`
	UserID      string                 `json:"userId"`
	Status      string                 `json:"status"`
	Input       map[string]interface{} `json:"input"`
	Output      map[string]interface{} `json:"output"`
	Errors      []ExecutionError       `json:"errors"`
	StartedAt   time.Time              `json:"startedAt"`
	CompletedAt *time.Time             `json:"completedAt"`
	Duration    int64                  `json:"durationMs"`
}

// ExecutionError represents an execution error
type ExecutionError struct {
	NodeID  string                 `json:"nodeId"`
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Schedule represents a workflow schedule
type Schedule struct {
	ID             string    `json:"id"`
	WorkflowID     string    `json:"workflowId"`
	Name           string    `json:"name"`
	CronExpression string    `json:"cronExpression"`
	Timezone       string    `json:"timezone"`
	Enabled        bool      `json:"enabled"`
	NextRunAt      time.Time `json:"nextRunAt"`
	LastRunAt      *time.Time `json:"lastRunAt"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// Webhook represents a webhook
type Webhook struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	EndpointURL string    `json:"endpointUrl"`
	Events      []string  `json:"events"`
	Secret      string    `json:"secret,omitempty"`
	Enabled     bool      `json:"enabled"`
	Signature   string    `json:"signature"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Notification represents a notification
type Notification struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"userId"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Type      string                 `json:"type"`
	Channels  []string               `json:"channels"`
	Priority  string                 `json:"priority"`
	Status    string                 `json:"status"`
	ReadAt    *time.Time             `json:"readAt"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"createdAt"`
}

// AnalyticsEvent represents an analytics event
type AnalyticsEvent struct {
	EventType  string                 `json:"eventType"`
	EventName  string                 `json:"eventName"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}
