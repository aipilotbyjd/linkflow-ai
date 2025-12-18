package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWorkflow(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		workflowName string
		description string
		wantErr     bool
	}{
		{
			name:         "valid workflow",
			userID:       "user-123",
			workflowName: "Test Workflow",
			description:  "Test Description",
			wantErr:      false,
		},
		{
			name:         "empty name",
			userID:       "user-123",
			workflowName: "",
			description:  "Test Description",
			wantErr:      true,
		},
		{
			name:         "empty userID",
			userID:       "",
			workflowName: "Test Workflow",
			description:  "Test Description",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflow, err := NewWorkflow(tt.userID, tt.workflowName, tt.description)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, workflow)
			} else {
				require.NoError(t, err)
				require.NotNil(t, workflow)
				
				assert.Equal(t, tt.workflowName, workflow.Name())
				assert.Equal(t, tt.description, workflow.Description())
				assert.Equal(t, tt.userID, workflow.UserID())
				assert.Equal(t, WorkflowStatusDraft, workflow.Status())
				// Version starts at 0 but might be incremented automatically
				assert.GreaterOrEqual(t, workflow.Version(), 0)
				assert.NotEmpty(t, workflow.ID())
			}
		})
	}
}

func TestWorkflowAddNode(t *testing.T) {
	workflow, err := NewWorkflow("user-123", "Test", "Description")
	require.NoError(t, err)

	node := Node{
		ID:   "node-1",
		Type: NodeTypeAction,
		Name: "HTTP Request",
	}

	err = workflow.AddNode(node)
	assert.NoError(t, err)

	nodes := workflow.Nodes()
	assert.Len(t, nodes, 1)
	assert.Equal(t, node.ID, nodes[0].ID)

	// Test duplicate node
	err = workflow.AddNode(node)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestWorkflowAddConnection(t *testing.T) {
	workflow, err := NewWorkflow("user-123", "Test", "Description")
	require.NoError(t, err)

	// Add nodes
	node1 := Node{ID: "node-1", Type: NodeTypeTrigger, Name: "Start"}
	node2 := Node{ID: "node-2", Type: NodeTypeAction, Name: "Action"}
	
	err = workflow.AddNode(node1)
	require.NoError(t, err)
	err = workflow.AddNode(node2)
	require.NoError(t, err)

	// Add connection
	connection := Connection{
		ID:           "conn-1",
		SourceNodeID: "node-1",
		TargetNodeID: "node-2",
	}

	err = workflow.AddConnection(connection)
	assert.NoError(t, err)

	connections := workflow.Connections()
	assert.Len(t, connections, 1)
	assert.Equal(t, connection.ID, connections[0].ID)
}

func TestWorkflowActivation(t *testing.T) {
	workflow, err := NewWorkflow("user-123", "Test", "Description")
	require.NoError(t, err)

	// Empty workflow should fail activation
	err = workflow.Activate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one node")

	// Add a trigger node
	node := Node{ID: "node-1", Type: NodeTypeTrigger, Name: "Start"}
	err = workflow.AddNode(node)
	require.NoError(t, err)

	// Should activate successfully now
	err = workflow.Activate()
	assert.NoError(t, err)
	assert.Equal(t, WorkflowStatusActive, workflow.Status())
}

func TestWorkflowConnections(t *testing.T) {
	workflow, err := NewWorkflow("user-123", "Test", "Description")
	require.NoError(t, err)

	// Create nodes
	node1 := Node{ID: "node-1", Type: NodeTypeTrigger, Name: "Start"}
	node2 := Node{ID: "node-2", Type: NodeTypeAction, Name: "Action"}
	node3 := Node{ID: "node-3", Type: NodeTypeAction, Name: "End"}

	workflow.AddNode(node1)
	workflow.AddNode(node2)
	workflow.AddNode(node3)

	// Create connections
	workflow.AddConnection(Connection{ID: "c1", SourceNodeID: "node-1", TargetNodeID: "node-2"})
	workflow.AddConnection(Connection{ID: "c2", SourceNodeID: "node-2", TargetNodeID: "node-3"})
	
	connections := workflow.Connections()
	assert.Len(t, connections, 2)
}

func TestWorkflowStatusTransitions(t *testing.T) {
	workflow, err := NewWorkflow("user-123", "Test", "Description")
	require.NoError(t, err)

	// Add a trigger node to make it valid
	workflow.AddNode(Node{ID: "node-1", Type: NodeTypeTrigger, Name: "Start"})

	// Draft -> Active
	err = workflow.Activate()
	assert.NoError(t, err)
	assert.Equal(t, WorkflowStatusActive, workflow.Status())

	// Active -> Inactive (Deactivate)
	err = workflow.Deactivate()
	assert.NoError(t, err)
	assert.Equal(t, WorkflowStatusInactive, workflow.Status())

	// Inactive -> Active
	err = workflow.Activate()
	assert.NoError(t, err)
	assert.Equal(t, WorkflowStatusActive, workflow.Status())

	// Active -> Archived
	err = workflow.Archive()
	assert.NoError(t, err)
	assert.Equal(t, WorkflowStatusArchived, workflow.Status())

	// Archived -> can't activate
	err = workflow.Activate()
	assert.Error(t, err)
}

func TestWorkflowNodeManagement(t *testing.T) {
	workflow, err := NewWorkflow("user-123", "Test", "Description")
	require.NoError(t, err)

	// Add nodes
	node1 := Node{ID: "node-1", Type: NodeTypeTrigger, Name: "Start"}
	node2 := Node{ID: "node-2", Type: NodeTypeAction, Name: "Action"}
	
	err = workflow.AddNode(node1)
	assert.NoError(t, err)
	err = workflow.AddNode(node2)
	assert.NoError(t, err)
	
	assert.Len(t, workflow.Nodes(), 2)

	// Remove a node
	err = workflow.RemoveNode("node-2")
	assert.NoError(t, err)
	assert.Len(t, workflow.Nodes(), 1)

	// Try to remove non-existent node
	err = workflow.RemoveNode("node-999")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestWorkflowArchive(t *testing.T) {
	workflow, err := NewWorkflow("user-123", "Test", "Description")
	require.NoError(t, err)

	// Archive workflow
	err = workflow.Archive()
	assert.NoError(t, err)
	assert.Equal(t, WorkflowStatusArchived, workflow.Status())

	// Can't modify archived workflow
	node := Node{ID: "node-1", Type: NodeTypeTrigger, Name: "Start"}
	err = workflow.AddNode(node)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "archived")
}
