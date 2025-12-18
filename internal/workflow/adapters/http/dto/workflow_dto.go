package dto

import (
	"errors"
	"time"
)

// CreateWorkflowRequest represents a request to create a workflow
type CreateWorkflowRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Nodes       []NodeDTO       `json:"nodes,omitempty"`
	Connections []ConnectionDTO `json:"connections,omitempty"`
}

// Validate validates the create workflow request
func (r *CreateWorkflowRequest) Validate() error {
	if r.Name == "" {
		return errors.New("workflow name is required")
	}
	if len(r.Name) < 3 || len(r.Name) > 200 {
		return errors.New("workflow name must be between 3 and 200 characters")
	}
	return nil
}

// UpdateWorkflowRequest represents a request to update a workflow
type UpdateWorkflowRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Nodes       []NodeDTO       `json:"nodes"`
	Connections []ConnectionDTO `json:"connections"`
	Settings    *SettingsDTO    `json:"settings,omitempty"`
}

// DuplicateWorkflowRequest represents a request to duplicate a workflow
type DuplicateWorkflowRequest struct {
	Name string `json:"name"`
}

// WorkflowResponse represents a workflow response
type WorkflowResponse struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Status      string          `json:"status"`
	Nodes       []NodeDTO       `json:"nodes"`
	Connections []ConnectionDTO `json:"connections"`
	Settings    SettingsDTO     `json:"settings"`
	Tags        []string        `json:"tags,omitempty"`
	Version     int             `json:"version"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

// ListWorkflowsResponse represents a list of workflows response
type ListWorkflowsResponse struct {
	Items      []WorkflowResponse `json:"items"`
	Total      int64              `json:"total"`
	Pagination Pagination         `json:"pagination"`
}

// Pagination represents pagination information
type Pagination struct {
	Offset int   `json:"offset"`
	Limit  int   `json:"limit"`
	Total  int64 `json:"total"`
}

// NodeDTO represents a workflow node
type NodeDTO struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
	Position    PositionDTO            `json:"position"`
}

// PositionDTO represents node position
type PositionDTO struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// ConnectionDTO represents a connection between nodes
type ConnectionDTO struct {
	ID           string `json:"id"`
	SourceNodeID string `json:"sourceNodeId"`
	TargetNodeID string `json:"targetNodeId"`
	SourcePort   string `json:"sourcePort,omitempty"`
	TargetPort   string `json:"targetPort,omitempty"`
}

// SettingsDTO represents workflow settings
type SettingsDTO struct {
	MaxExecutionTime int                    `json:"maxExecutionTime"`
	RetryPolicy      RetryPolicyDTO         `json:"retryPolicy"`
	ErrorHandling    string                 `json:"errorHandling"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// RetryPolicyDTO represents retry policy settings
type RetryPolicyDTO struct {
	MaxAttempts int           `json:"maxAttempts"`
	BackoffType string        `json:"backoffType"`
	Delay       time.Duration `json:"delay"`
}
