// Package features provides workflow debug mode
package features

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/linkflow-ai/linkflow-ai/internal/workflow/domain/model"
)

// DebugSession represents an active debug session
type DebugSession struct {
	ID           string
	WorkflowID   string
	UserID       string
	Status       DebugStatus
	CurrentNode  string
	Breakpoints  map[string]bool
	WatchedVars  []string
	StepMode     StepMode
	StartedAt    time.Time
	LastActivity time.Time
	ExecutionID  string
	State        *DebugState
	History      []*DebugEvent
	mu           sync.RWMutex
}

// DebugStatus represents debug session status
type DebugStatus string

const (
	DebugStatusIdle     DebugStatus = "idle"
	DebugStatusRunning  DebugStatus = "running"
	DebugStatusPaused   DebugStatus = "paused"
	DebugStatusStepping DebugStatus = "stepping"
	DebugStatusStopped  DebugStatus = "stopped"
)

// StepMode represents the stepping mode
type StepMode string

const (
	StepModeNone     StepMode = "none"
	StepModeInto     StepMode = "step_into"     // Step into sub-workflows
	StepModeOver     StepMode = "step_over"     // Step over sub-workflows
	StepModeOut      StepMode = "step_out"      // Step out of current context
	StepModeToNext   StepMode = "step_to_next"  // Step to next node
	StepModeToCursor StepMode = "step_to_cursor" // Step to specific node
)

// DebugState holds the current debug state
type DebugState struct {
	NodeOutputs  map[string]map[string]interface{} `json:"nodeOutputs"`
	Variables    map[string]interface{}            `json:"variables"`
	TriggerData  map[string]interface{}            `json:"triggerData"`
	CurrentInput map[string]interface{}            `json:"currentInput"`
	CallStack    []StackFrame                      `json:"callStack"`
}

// StackFrame represents a frame in the call stack
type StackFrame struct {
	NodeID       string                 `json:"nodeId"`
	NodeName     string                 `json:"nodeName"`
	NodeType     string                 `json:"nodeType"`
	Input        map[string]interface{} `json:"input"`
	Depth        int                    `json:"depth"`
	SubWorkflow  string                 `json:"subWorkflow,omitempty"`
}

// DebugEvent represents a debug event
type DebugEvent struct {
	ID        string                 `json:"id"`
	Type      DebugEventType         `json:"type"`
	NodeID    string                 `json:"nodeId,omitempty"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// DebugEventType represents the type of debug event
type DebugEventType string

const (
	DebugEventNodeEnter      DebugEventType = "node_enter"
	DebugEventNodeExit       DebugEventType = "node_exit"
	DebugEventNodeError      DebugEventType = "node_error"
	DebugEventBreakpointHit  DebugEventType = "breakpoint_hit"
	DebugEventVariableChange DebugEventType = "variable_change"
	DebugEventStepComplete   DebugEventType = "step_complete"
	DebugEventSessionStart   DebugEventType = "session_start"
	DebugEventSessionEnd     DebugEventType = "session_end"
)

// Breakpoint represents a breakpoint
type Breakpoint struct {
	ID         string                 `json:"id"`
	NodeID     string                 `json:"nodeId"`
	Condition  string                 `json:"condition,omitempty"`
	HitCount   int                    `json:"hitCount"`
	MaxHits    int                    `json:"maxHits,omitempty"`
	Enabled    bool                   `json:"enabled"`
	LogMessage string                 `json:"logMessage,omitempty"`
	Actions    []BreakpointAction     `json:"actions,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// BreakpointAction represents an action to take at breakpoint
type BreakpointAction string

const (
	BreakpointActionPause   BreakpointAction = "pause"
	BreakpointActionLog     BreakpointAction = "log"
	BreakpointActionEval    BreakpointAction = "eval"
	BreakpointActionModify  BreakpointAction = "modify"
)

// WatchExpression represents a watch expression
type WatchExpression struct {
	ID         string      `json:"id"`
	Expression string      `json:"expression"`
	Value      interface{} `json:"value,omitempty"`
	Error      string      `json:"error,omitempty"`
	LastEval   time.Time   `json:"lastEval,omitempty"`
}

// DebugManager manages debug sessions
type DebugManager struct {
	sessions   map[string]*DebugSession
	mu         sync.RWMutex
	maxSessions int
}

// NewDebugManager creates a new debug manager
func NewDebugManager() *DebugManager {
	return &DebugManager{
		sessions:    make(map[string]*DebugSession),
		maxSessions: 100,
	}
}

// CreateSession creates a new debug session
func (m *DebugManager) CreateSession(workflowID, userID string) (*DebugSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if session already exists for this workflow/user
	for _, session := range m.sessions {
		if session.WorkflowID == workflowID && session.UserID == userID && session.Status != DebugStatusStopped {
			return nil, fmt.Errorf("debug session already exists for this workflow")
		}
	}

	session := &DebugSession{
		ID:           uuid.New().String(),
		WorkflowID:   workflowID,
		UserID:       userID,
		Status:       DebugStatusIdle,
		Breakpoints:  make(map[string]bool),
		WatchedVars:  make([]string, 0),
		StepMode:     StepModeNone,
		StartedAt:    time.Now(),
		LastActivity: time.Now(),
		State: &DebugState{
			NodeOutputs: make(map[string]map[string]interface{}),
			Variables:   make(map[string]interface{}),
			CallStack:   make([]StackFrame, 0),
		},
		History: make([]*DebugEvent, 0),
	}

	m.sessions[session.ID] = session

	session.addEvent(DebugEventSessionStart, "", "Debug session started", nil)

	// Cleanup if too many sessions
	if len(m.sessions) > m.maxSessions {
		m.cleanupOldestSession()
	}

	return session, nil
}

// GetSession retrieves a debug session
func (m *DebugManager) GetSession(sessionID string) (*DebugSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, exists := m.sessions[sessionID]
	return session, exists
}

// EndSession ends a debug session
func (m *DebugManager) EndSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	session.Status = DebugStatusStopped
	session.addEvent(DebugEventSessionEnd, "", "Debug session ended", nil)

	return nil
}

// ListSessions lists all sessions for a user
func (m *DebugManager) ListSessions(userID string) []*DebugSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sessions []*DebugSession
	for _, session := range m.sessions {
		if session.UserID == userID {
			sessions = append(sessions, session)
		}
	}
	return sessions
}

func (m *DebugManager) cleanupOldestSession() {
	var oldest string
	var oldestTime time.Time

	for id, session := range m.sessions {
		if session.Status == DebugStatusStopped {
			if oldest == "" || session.LastActivity.Before(oldestTime) {
				oldest = id
				oldestTime = session.LastActivity
			}
		}
	}

	if oldest != "" {
		delete(m.sessions, oldest)
	}
}

// DebugSession methods

// SetBreakpoint sets a breakpoint on a node
func (s *DebugSession) SetBreakpoint(nodeID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Breakpoints[nodeID] = true
	s.LastActivity = time.Now()
}

// RemoveBreakpoint removes a breakpoint
func (s *DebugSession) RemoveBreakpoint(nodeID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Breakpoints, nodeID)
	s.LastActivity = time.Now()
}

// ClearBreakpoints clears all breakpoints
func (s *DebugSession) ClearBreakpoints() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Breakpoints = make(map[string]bool)
	s.LastActivity = time.Now()
}

// HasBreakpoint checks if a node has a breakpoint
func (s *DebugSession) HasBreakpoint(nodeID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Breakpoints[nodeID]
}

// AddWatch adds a variable to watch
func (s *DebugSession) AddWatch(varName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.WatchedVars = append(s.WatchedVars, varName)
	s.LastActivity = time.Now()
}

// RemoveWatch removes a watched variable
func (s *DebugSession) RemoveWatch(varName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, v := range s.WatchedVars {
		if v == varName {
			s.WatchedVars = append(s.WatchedVars[:i], s.WatchedVars[i+1:]...)
			break
		}
	}
	s.LastActivity = time.Now()
}

// Start starts the debug session
func (s *DebugSession) Start(triggerData map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = DebugStatusRunning
	s.State.TriggerData = triggerData
	s.ExecutionID = uuid.New().String()
	s.LastActivity = time.Now()
}

// Pause pauses the debug session
func (s *DebugSession) Pause() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = DebugStatusPaused
	s.LastActivity = time.Now()
}

// Resume resumes the debug session
func (s *DebugSession) Resume() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = DebugStatusRunning
	s.StepMode = StepModeNone
	s.LastActivity = time.Now()
}

// StepInto steps into the next node
func (s *DebugSession) StepInto() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = DebugStatusStepping
	s.StepMode = StepModeInto
	s.LastActivity = time.Now()
}

// StepOver steps over the next node
func (s *DebugSession) StepOver() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = DebugStatusStepping
	s.StepMode = StepModeOver
	s.LastActivity = time.Now()
}

// StepOut steps out of current context
func (s *DebugSession) StepOut() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = DebugStatusStepping
	s.StepMode = StepModeOut
	s.LastActivity = time.Now()
}

// Stop stops the debug session
func (s *DebugSession) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = DebugStatusStopped
	s.LastActivity = time.Now()
}

// EnterNode records entering a node
func (s *DebugSession) EnterNode(nodeID, nodeName, nodeType string, input map[string]interface{}) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.CurrentNode = nodeID
	s.State.CurrentInput = input
	s.State.CallStack = append(s.State.CallStack, StackFrame{
		NodeID:   nodeID,
		NodeName: nodeName,
		NodeType: nodeType,
		Input:    input,
		Depth:    len(s.State.CallStack),
	})

	s.addEvent(DebugEventNodeEnter, nodeID, fmt.Sprintf("Entering node: %s", nodeName), map[string]interface{}{
		"input": input,
	})

	// Check breakpoint
	if s.Breakpoints[nodeID] {
		s.Status = DebugStatusPaused
		s.addEvent(DebugEventBreakpointHit, nodeID, fmt.Sprintf("Breakpoint hit at: %s", nodeName), nil)
		return true // Should pause
	}

	// Check step mode
	if s.StepMode != StepModeNone {
		s.Status = DebugStatusPaused
		s.addEvent(DebugEventStepComplete, nodeID, fmt.Sprintf("Step complete at: %s", nodeName), nil)
		return true // Should pause
	}

	s.LastActivity = time.Now()
	return false
}

// ExitNode records exiting a node
func (s *DebugSession) ExitNode(nodeID string, output map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store output
	s.State.NodeOutputs[nodeID] = output

	// Pop from call stack
	if len(s.State.CallStack) > 0 {
		s.State.CallStack = s.State.CallStack[:len(s.State.CallStack)-1]
	}

	s.addEvent(DebugEventNodeExit, nodeID, "Exiting node", map[string]interface{}{
		"output": output,
	})

	s.LastActivity = time.Now()
}

// NodeError records a node error
func (s *DebugSession) NodeError(nodeID string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.addEvent(DebugEventNodeError, nodeID, fmt.Sprintf("Node error: %v", err), nil)
	s.Status = DebugStatusPaused
	s.LastActivity = time.Now()
}

// SetVariable sets a variable value (for debugging)
func (s *DebugSession) SetVariable(name string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldValue := s.State.Variables[name]
	s.State.Variables[name] = value

	s.addEvent(DebugEventVariableChange, "", fmt.Sprintf("Variable '%s' changed", name), map[string]interface{}{
		"oldValue": oldValue,
		"newValue": value,
	})

	s.LastActivity = time.Now()
}

// GetVariable gets a variable value
func (s *DebugSession) GetVariable(name string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, exists := s.State.Variables[name]
	return value, exists
}

// GetState returns the current debug state
func (s *DebugSession) GetState() *DebugState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State
}

// GetHistory returns debug event history
func (s *DebugSession) GetHistory(limit int) []*DebugEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.History) {
		return s.History
	}
	return s.History[len(s.History)-limit:]
}

func (s *DebugSession) addEvent(eventType DebugEventType, nodeID, message string, data map[string]interface{}) {
	event := &DebugEvent{
		ID:        uuid.New().String(),
		Type:      eventType,
		NodeID:    nodeID,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	}
	s.History = append(s.History, event)
}

// DebugExecutor wraps workflow execution with debug support
type DebugExecutor struct {
	session  *DebugSession
	executor WorkflowExecutor
}

// NewDebugExecutor creates a new debug executor
func NewDebugExecutor(session *DebugSession, executor WorkflowExecutor) *DebugExecutor {
	return &DebugExecutor{
		session:  session,
		executor: executor,
	}
}

// Execute executes a workflow in debug mode
func (d *DebugExecutor) Execute(ctx context.Context, workflow *model.Workflow, input map[string]interface{}, options *ExecutionOptions) (*ExecutionResult, error) {
	d.session.Start(input)

	// Execute with debug hooks
	// In a real implementation, this would integrate with the actual executor
	// and call d.session.EnterNode/ExitNode at appropriate times

	// Flatten node outputs for result
	output := make(map[string]interface{})
	for nodeID, nodeOutput := range d.session.State.NodeOutputs {
		output[nodeID] = nodeOutput
	}

	result := &ExecutionResult{
		ExecutionID: d.session.ExecutionID,
		Status:      "completed",
		Output:      output,
	}

	return result, nil
}

// WaitForResume waits for the debug session to resume
func (d *DebugExecutor) WaitForResume(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if d.session.Status == DebugStatusRunning || d.session.Status == DebugStatusStepping {
				return nil
			}
			if d.session.Status == DebugStatusStopped {
				return fmt.Errorf("debug session stopped")
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}
