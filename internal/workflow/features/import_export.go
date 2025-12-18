// Package features provides workflow import/export functionality
package features

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/linkflow-ai/linkflow-ai/internal/workflow/domain/model"
)

// ExportFormat represents the export format
type ExportFormat string

const (
	ExportFormatJSON ExportFormat = "json"
	ExportFormatYAML ExportFormat = "yaml"
)

// WorkflowExport represents an exported workflow
type WorkflowExport struct {
	Version     string                 `json:"version"`
	ExportedAt  time.Time              `json:"exportedAt"`
	ExportedBy  string                 `json:"exportedBy,omitempty"`
	Workflow    WorkflowData           `json:"workflow"`
	Credentials []CredentialExport     `json:"credentials,omitempty"`
	Variables   map[string]interface{} `json:"variables,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// WorkflowData represents workflow data for export/import
type WorkflowData struct {
	ID          string             `json:"id,omitempty"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Nodes       []model.Node       `json:"nodes"`
	Connections []model.Connection `json:"connections"`
	Settings    model.Settings     `json:"settings"`
	Tags        []string           `json:"tags,omitempty"`
}

// CredentialExport represents credential reference in export
type CredentialExport struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	NodeID      string `json:"nodeId"`
	Placeholder string `json:"placeholder"` // For import mapping
}

// ImportOptions holds import configuration
type ImportOptions struct {
	UserID           string
	WorkspaceID      string
	OverwriteExisting bool
	CredentialMapping map[string]string // Maps placeholder to actual credential ID
	VariableMapping   map[string]interface{}
	Prefix           string // Prefix for imported workflow name
}

// ImportResult represents the result of an import operation
type ImportResult struct {
	WorkflowID       string
	WorkflowName     string
	NodesImported    int
	ConnectionsImported int
	CredentialsNeeded []CredentialExport
	Warnings         []string
	Success          bool
}

// WorkflowImportExport handles workflow import/export
type WorkflowImportExport struct {
	currentVersion string
}

// NewWorkflowImportExport creates a new import/export handler
func NewWorkflowImportExport() *WorkflowImportExport {
	return &WorkflowImportExport{
		currentVersion: "1.0.0",
	}
}

// Export exports a workflow to the specified format
func (ie *WorkflowImportExport) Export(ctx context.Context, workflow *model.Workflow, format ExportFormat, userID string) ([]byte, error) {
	export := &WorkflowExport{
		Version:    ie.currentVersion,
		ExportedAt: time.Now(),
		ExportedBy: userID,
		Workflow: WorkflowData{
			ID:          workflow.ID().String(),
			Name:        workflow.Name(),
			Description: workflow.Description(),
			Nodes:       workflow.Nodes(),
			Connections: workflow.Connections(),
			Settings:    workflow.Settings(),
		},
		Credentials: ie.extractCredentials(workflow),
		Metadata:    make(map[string]interface{}),
	}

	switch format {
	case ExportFormatJSON:
		return json.MarshalIndent(export, "", "  ")
	case ExportFormatYAML:
		// For YAML, we'd use a YAML library
		// For now, return JSON
		return json.MarshalIndent(export, "", "  ")
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// extractCredentials extracts credential references from workflow
func (ie *WorkflowImportExport) extractCredentials(workflow *model.Workflow) []CredentialExport {
	var credentials []CredentialExport
	seen := make(map[string]bool)

	for _, node := range workflow.Nodes() {
		if credID, ok := node.Config["credentialId"].(string); ok && credID != "" {
			if !seen[credID] {
				seen[credID] = true
				credType, _ := node.Config["credentialType"].(string)
				credentials = append(credentials, CredentialExport{
					ID:          credID,
					Name:        fmt.Sprintf("%s_credential", node.Name),
					Type:        credType,
					NodeID:      node.ID,
					Placeholder: fmt.Sprintf("{{credential:%s}}", node.ID),
				})
			}
		}
	}

	return credentials
}

// Import imports a workflow from exported data
func (ie *WorkflowImportExport) Import(ctx context.Context, data []byte, options *ImportOptions) (*ImportResult, error) {
	result := &ImportResult{
		Warnings: []string{},
	}

	// Parse export data
	var export WorkflowExport
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, fmt.Errorf("failed to parse import data: %w", err)
	}

	// Version check
	if export.Version != ie.currentVersion {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Version mismatch: export is %s, current is %s", export.Version, ie.currentVersion))
	}

	// Create workflow name
	workflowName := export.Workflow.Name
	if options.Prefix != "" {
		workflowName = options.Prefix + " " + workflowName
	}

	// Create new workflow
	workflow, err := model.NewWorkflow(options.UserID, workflowName, export.Workflow.Description)
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	// Map old node IDs to new ones
	nodeIDMap := make(map[string]string)

	// Import nodes
	for _, node := range export.Workflow.Nodes {
		newNodeID := uuid.New().String()
		nodeIDMap[node.ID] = newNodeID

		// Process credentials
		if credID, ok := node.Config["credentialId"].(string); ok && credID != "" {
			if mappedID, exists := options.CredentialMapping[credID]; exists {
				node.Config["credentialId"] = mappedID
			} else {
				// Add to credentials needed
				for _, cred := range export.Credentials {
					if cred.NodeID == node.ID {
						result.CredentialsNeeded = append(result.CredentialsNeeded, cred)
						break
					}
				}
			}
		}

		// Apply variable mapping
		if options.VariableMapping != nil {
			node.Config = ie.applyVariableMapping(node.Config, options.VariableMapping)
		}

		newNode := model.Node{
			ID:          newNodeID,
			Type:        node.Type,
			Name:        node.Name,
			Description: node.Description,
			Config:      node.Config,
			Position:    node.Position,
		}

		if err := workflow.AddNode(newNode); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to add node %s: %v", node.Name, err))
			continue
		}
		result.NodesImported++
	}

	// Import connections with updated node IDs
	for _, conn := range export.Workflow.Connections {
		newSourceID, sourceExists := nodeIDMap[conn.SourceNodeID]
		newTargetID, targetExists := nodeIDMap[conn.TargetNodeID]

		if !sourceExists || !targetExists {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Skipping connection: node not found"))
			continue
		}

		newConn := model.Connection{
			ID:           uuid.New().String(),
			SourceNodeID: newSourceID,
			TargetNodeID: newTargetID,
			SourcePort:   conn.SourcePort,
			TargetPort:   conn.TargetPort,
		}

		if err := workflow.AddConnection(newConn); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to add connection: %v", err))
			continue
		}
		result.ConnectionsImported++
	}

	// Apply settings
	if err := workflow.UpdateSettings(export.Workflow.Settings); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to apply settings: %v", err))
	}

	result.WorkflowID = workflow.ID().String()
	result.WorkflowName = workflow.Name()
	result.Success = true

	return result, nil
}

// applyVariableMapping applies variable substitution to config
func (ie *WorkflowImportExport) applyVariableMapping(config map[string]interface{}, mapping map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	for key, value := range config {
		switch v := value.(type) {
		case string:
			// Check for variable placeholders
			for varName, varValue := range mapping {
				placeholder := "{{" + varName + "}}"
				if v == placeholder {
					result[key] = varValue
					break
				}
			}
			if _, exists := result[key]; !exists {
				result[key] = value
			}
		case map[string]interface{}:
			result[key] = ie.applyVariableMapping(v, mapping)
		default:
			result[key] = value
		}
	}
	
	return result
}

// ValidateExport validates export data
func (ie *WorkflowImportExport) ValidateExport(data []byte) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	var export WorkflowExport
	if err := json.Unmarshal(data, &export); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Invalid JSON: %v", err))
		return result, nil
	}

	// Check version
	if export.Version == "" {
		result.Errors = append(result.Errors, "Missing version field")
		result.Valid = false
	}

	// Check workflow data
	if export.Workflow.Name == "" {
		result.Errors = append(result.Errors, "Missing workflow name")
		result.Valid = false
	}

	if len(export.Workflow.Nodes) == 0 {
		result.Warnings = append(result.Warnings, "Workflow has no nodes")
	}

	// Check for trigger node
	hasTrigger := false
	for _, node := range export.Workflow.Nodes {
		if node.Type == model.NodeTypeTrigger {
			hasTrigger = true
			break
		}
	}
	if !hasTrigger {
		result.Warnings = append(result.Warnings, "Workflow has no trigger node")
	}

	// Check connections reference valid nodes
	nodeIDs := make(map[string]bool)
	for _, node := range export.Workflow.Nodes {
		nodeIDs[node.ID] = true
	}

	for _, conn := range export.Workflow.Connections {
		if !nodeIDs[conn.SourceNodeID] {
			result.Errors = append(result.Errors, fmt.Sprintf("Connection references non-existent source node: %s", conn.SourceNodeID))
			result.Valid = false
		}
		if !nodeIDs[conn.TargetNodeID] {
			result.Errors = append(result.Errors, fmt.Sprintf("Connection references non-existent target node: %s", conn.TargetNodeID))
			result.Valid = false
		}
	}

	return result, nil
}

// ValidationResult holds validation results
type ValidationResult struct {
	Valid    bool
	Errors   []string
	Warnings []string
}

// Duplicate duplicates a workflow
func (ie *WorkflowImportExport) Duplicate(ctx context.Context, workflow *model.Workflow, newName string, userID string) (*model.Workflow, error) {
	// Export the workflow
	data, err := ie.Export(ctx, workflow, ExportFormatJSON, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to export workflow: %w", err)
	}

	// Import with new name
	result, err := ie.Import(ctx, data, &ImportOptions{
		UserID: userID,
		Prefix: "",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to import workflow: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("duplication failed: %v", result.Warnings)
	}

	// Create the duplicated workflow
	duplicated, err := model.NewWorkflow(userID, newName, workflow.Description())
	if err != nil {
		return nil, err
	}

	// Add nodes from original
	for _, node := range workflow.Nodes() {
		newNode := model.Node{
			ID:          uuid.New().String(),
			Type:        node.Type,
			Name:        node.Name,
			Description: node.Description,
			Config:      node.Config,
			Position:    node.Position,
		}
		duplicated.AddNode(newNode)
	}

	return duplicated, nil
}

// BulkExport exports multiple workflows
func (ie *WorkflowImportExport) BulkExport(ctx context.Context, workflows []*model.Workflow, userID string) ([]byte, error) {
	exports := make([]*WorkflowExport, len(workflows))

	for i, workflow := range workflows {
		exports[i] = &WorkflowExport{
			Version:    ie.currentVersion,
			ExportedAt: time.Now(),
			ExportedBy: userID,
			Workflow: WorkflowData{
				ID:          workflow.ID().String(),
				Name:        workflow.Name(),
				Description: workflow.Description(),
				Nodes:       workflow.Nodes(),
				Connections: workflow.Connections(),
				Settings:    workflow.Settings(),
			},
			Credentials: ie.extractCredentials(workflow),
		}
	}

	return json.MarshalIndent(map[string]interface{}{
		"version":    ie.currentVersion,
		"exportedAt": time.Now(),
		"exportedBy": userID,
		"workflows":  exports,
	}, "", "  ")
}

// BulkImport imports multiple workflows
func (ie *WorkflowImportExport) BulkImport(ctx context.Context, data []byte, options *ImportOptions) ([]*ImportResult, error) {
	var bulkExport struct {
		Workflows []*WorkflowExport `json:"workflows"`
	}

	if err := json.Unmarshal(data, &bulkExport); err != nil {
		return nil, fmt.Errorf("failed to parse bulk import data: %w", err)
	}

	results := make([]*ImportResult, len(bulkExport.Workflows))

	for i, export := range bulkExport.Workflows {
		exportData, _ := json.Marshal(export)
		result, err := ie.Import(ctx, exportData, options)
		if err != nil {
			results[i] = &ImportResult{
				Success:  false,
				Warnings: []string{err.Error()},
			}
		} else {
			results[i] = result
		}
	}

	return results, nil
}
