package unit_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Workflow represents a workflow entity
type Workflow struct {
	ID          string
	Name        string
	Description string
	Status      string
	Nodes       []Node
	Connections []Connection
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Version     int
}

type Node struct {
	ID       string
	Type     string
	Name     string
	Position Position
	Config   map[string]interface{}
}

type Position struct {
	X int
	Y int
}

type Connection struct {
	Source string
	Target string
	Label  string
}

// NewWorkflow creates a new workflow
func NewWorkflow(name, description string) (*Workflow, error) {
	if name == "" {
		return nil, assert.AnError
	}
	return &Workflow{
		ID:          "wf-" + time.Now().Format("20060102150405"),
		Name:        name,
		Description: description,
		Status:      "draft",
		Nodes:       []Node{},
		Connections: []Connection{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     1,
	}, nil
}

func (w *Workflow) AddNode(node Node) error {
	if node.ID == "" {
		return assert.AnError
	}
	for _, n := range w.Nodes {
		if n.ID == node.ID {
			return assert.AnError
		}
	}
	w.Nodes = append(w.Nodes, node)
	w.UpdatedAt = time.Now()
	return nil
}

func (w *Workflow) AddConnection(conn Connection) error {
	if conn.Source == "" || conn.Target == "" {
		return assert.AnError
	}
	// Validate nodes exist
	sourceExists, targetExists := false, false
	for _, n := range w.Nodes {
		if n.ID == conn.Source {
			sourceExists = true
		}
		if n.ID == conn.Target {
			targetExists = true
		}
	}
	if !sourceExists || !targetExists {
		return assert.AnError
	}
	w.Connections = append(w.Connections, conn)
	return nil
}

func (w *Workflow) Activate() error {
	if len(w.Nodes) == 0 {
		return assert.AnError
	}
	hasTrigger := false
	for _, n := range w.Nodes {
		if n.Type == "trigger" {
			hasTrigger = true
			break
		}
	}
	if !hasTrigger {
		return assert.AnError
	}
	w.Status = "active"
	w.UpdatedAt = time.Now()
	return nil
}

func (w *Workflow) Validate() []string {
	var errors []string
	if w.Name == "" {
		errors = append(errors, "name is required")
	}
	if len(w.Nodes) == 0 {
		errors = append(errors, "workflow must have at least one node")
	}
	return errors
}

// Tests
func TestNewWorkflow(t *testing.T) {
	t.Run("creates workflow with valid inputs", func(t *testing.T) {
		wf, err := NewWorkflow("Test Workflow", "A test workflow")
		require.NoError(t, err)
		assert.NotEmpty(t, wf.ID)
		assert.Equal(t, "Test Workflow", wf.Name)
		assert.Equal(t, "A test workflow", wf.Description)
		assert.Equal(t, "draft", wf.Status)
		assert.Empty(t, wf.Nodes)
		assert.Equal(t, 1, wf.Version)
	})

	t.Run("fails with empty name", func(t *testing.T) {
		wf, err := NewWorkflow("", "Description")
		assert.Error(t, err)
		assert.Nil(t, wf)
	})
}

func TestWorkflow_AddNode(t *testing.T) {
	t.Run("adds valid node", func(t *testing.T) {
		wf, _ := NewWorkflow("Test", "Test")
		node := Node{
			ID:   "node-1",
			Type: "trigger",
			Name: "HTTP Trigger",
		}
		err := wf.AddNode(node)
		assert.NoError(t, err)
		assert.Len(t, wf.Nodes, 1)
	})

	t.Run("fails with empty node ID", func(t *testing.T) {
		wf, _ := NewWorkflow("Test", "Test")
		node := Node{Name: "Invalid"}
		err := wf.AddNode(node)
		assert.Error(t, err)
	})

	t.Run("fails with duplicate node ID", func(t *testing.T) {
		wf, _ := NewWorkflow("Test", "Test")
		node := Node{ID: "node-1", Type: "trigger"}
		_ = wf.AddNode(node)
		err := wf.AddNode(node)
		assert.Error(t, err)
	})
}

func TestWorkflow_AddConnection(t *testing.T) {
	t.Run("adds valid connection", func(t *testing.T) {
		wf, _ := NewWorkflow("Test", "Test")
		_ = wf.AddNode(Node{ID: "node-1", Type: "trigger"})
		_ = wf.AddNode(Node{ID: "node-2", Type: "action"})

		conn := Connection{Source: "node-1", Target: "node-2"}
		err := wf.AddConnection(conn)
		assert.NoError(t, err)
		assert.Len(t, wf.Connections, 1)
	})

	t.Run("fails with non-existent source", func(t *testing.T) {
		wf, _ := NewWorkflow("Test", "Test")
		_ = wf.AddNode(Node{ID: "node-2", Type: "action"})

		conn := Connection{Source: "node-1", Target: "node-2"}
		err := wf.AddConnection(conn)
		assert.Error(t, err)
	})

	t.Run("fails with empty source or target", func(t *testing.T) {
		wf, _ := NewWorkflow("Test", "Test")
		conn := Connection{Source: "", Target: "node-2"}
		err := wf.AddConnection(conn)
		assert.Error(t, err)
	})
}

func TestWorkflow_Activate(t *testing.T) {
	t.Run("activates workflow with trigger", func(t *testing.T) {
		wf, _ := NewWorkflow("Test", "Test")
		_ = wf.AddNode(Node{ID: "node-1", Type: "trigger"})

		err := wf.Activate()
		assert.NoError(t, err)
		assert.Equal(t, "active", wf.Status)
	})

	t.Run("fails without nodes", func(t *testing.T) {
		wf, _ := NewWorkflow("Test", "Test")
		err := wf.Activate()
		assert.Error(t, err)
	})

	t.Run("fails without trigger node", func(t *testing.T) {
		wf, _ := NewWorkflow("Test", "Test")
		_ = wf.AddNode(Node{ID: "node-1", Type: "action"})

		err := wf.Activate()
		assert.Error(t, err)
	})
}

func TestWorkflow_Validate(t *testing.T) {
	t.Run("returns no errors for valid workflow", func(t *testing.T) {
		wf, _ := NewWorkflow("Test", "Test")
		_ = wf.AddNode(Node{ID: "node-1", Type: "trigger"})

		errors := wf.Validate()
		assert.Empty(t, errors)
	})

	t.Run("returns errors for invalid workflow", func(t *testing.T) {
		wf := &Workflow{}
		errors := wf.Validate()
		assert.Contains(t, errors, "name is required")
		assert.Contains(t, errors, "workflow must have at least one node")
	})
}
