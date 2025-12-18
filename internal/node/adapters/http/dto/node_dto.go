package dto

import (
	"errors"
	"time"
)

// CreateNodeRequest represents a request to create a node
type CreateNodeRequest struct {
	Name             string         `json:"name"`
	Type             string         `json:"type"`
	Category         string         `json:"category"`
	Description      string         `json:"description"`
	Icon             string         `json:"icon,omitempty"`
	Color            string         `json:"color,omitempty"`
	Inputs           []NodePort     `json:"inputs,omitempty"`
	Outputs          []NodePort     `json:"outputs,omitempty"`
	Properties       []NodeProperty `json:"properties,omitempty"`
	ExecutionHandler string         `json:"executionHandler,omitempty"`
	Documentation    string         `json:"documentation,omitempty"`
	Tags             []string       `json:"tags,omitempty"`
	IsPremium        bool           `json:"isPremium,omitempty"`
}

// Validate validates the create node request
func (r *CreateNodeRequest) Validate() error {
	if r.Name == "" {
		return errors.New("node name is required")
	}
	if r.Type == "" {
		return errors.New("node type is required")
	}
	if r.Category == "" {
		return errors.New("node category is required")
	}
	return nil
}

// UpdateNodeRequest represents a request to update a node
type UpdateNodeRequest struct {
	Name             string         `json:"name,omitempty"`
	Description      string         `json:"description,omitempty"`
	Icon             string         `json:"icon,omitempty"`
	Color            string         `json:"color,omitempty"`
	Inputs           []NodePort     `json:"inputs,omitempty"`
	Outputs          []NodePort     `json:"outputs,omitempty"`
	Properties       []NodeProperty `json:"properties,omitempty"`
	ExecutionHandler string         `json:"executionHandler,omitempty"`
	Documentation    string         `json:"documentation,omitempty"`
	Tags             []string       `json:"tags,omitempty"`
	Status           string         `json:"status,omitempty"`
}

// NodeResponse represents a node response
type NodeResponse struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Type             string         `json:"type"`
	Category         string         `json:"category"`
	Description      string         `json:"description"`
	Icon             string         `json:"icon"`
	Color            string         `json:"color"`
	Version          string         `json:"version"`
	Status           string         `json:"status"`
	Inputs           []NodePort     `json:"inputs"`
	Outputs          []NodePort     `json:"outputs"`
	Properties       []NodeProperty `json:"properties"`
	Tags             []string       `json:"tags"`
	IsSystem         bool           `json:"isSystem"`
	IsPremium        bool           `json:"isPremium"`
	ExecutionHandler string         `json:"executionHandler"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

// NodePort represents an input or output port
type NodePort struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Required     bool                   `json:"required"`
	Multiple     bool                   `json:"multiple"`
	Description  string                 `json:"description,omitempty"`
	DefaultValue interface{}           `json:"defaultValue,omitempty"`
	Schema       map[string]interface{} `json:"schema,omitempty"`
}

// NodeProperty represents a configuration property
type NodeProperty struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Required     bool                   `json:"required"`
	Description  string                 `json:"description,omitempty"`
	DefaultValue interface{}           `json:"defaultValue,omitempty"`
	Options      []PropertyOption       `json:"options,omitempty"`
	Validation   map[string]interface{} `json:"validation,omitempty"`
	Placeholder  string                 `json:"placeholder,omitempty"`
	Hidden       bool                   `json:"hidden,omitempty"`
}

// PropertyOption represents an option for select/multiselect properties
type PropertyOption struct {
	Label string      `json:"label"`
	Value interface{} `json:"value"`
}

// ListNodesResponse represents a list of nodes response
type ListNodesResponse struct {
	Items      []NodeResponse `json:"items"`
	Total      int64          `json:"total"`
	Pagination Pagination     `json:"pagination"`
}

// Pagination represents pagination information
type Pagination struct {
	Offset int   `json:"offset"`
	Limit  int   `json:"limit"`
	Total  int64 `json:"total"`
}
