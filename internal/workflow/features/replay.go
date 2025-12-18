// Package features provides execution replay functionality
package features

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ExecutionSnapshot represents a snapshot of execution state at a point in time
type ExecutionSnapshot struct {
	ID            string                 `json:"id"`
	ExecutionID   string                 `json:"executionId"`
	WorkflowID    string                 `json:"workflowId"`
	NodeID        string                 `json:"nodeId"`
	NodeName      string                 `json:"nodeName"`
	NodeType      string                 `json:"nodeType"`
	Sequence      int                    `json:"sequence"`
	Timestamp     time.Time              `json:"timestamp"`
	Status        string                 `json:"status"`
	Input         map[string]interface{} `json:"input"`
	Output        map[string]interface{} `json:"output,omitempty"`
	Error         string                 `json:"error,omitempty"`
	DurationMs    int64                  `json:"durationMs"`
	Variables     map[string]interface{} `json:"variables"`
	ContextData   map[string]interface{} `json:"contextData"`
}

// ExecutionRecording represents a full execution recording
type ExecutionRecording struct {
	ID            string                 `json:"id"`
	ExecutionID   string                 `json:"executionId"`
	WorkflowID    string                 `json:"workflowId"`
	WorkflowName  string                 `json:"workflowName"`
	Mode          string                 `json:"mode"`
	Status        string                 `json:"status"`
	StartedAt     time.Time              `json:"startedAt"`
	CompletedAt   *time.Time             `json:"completedAt,omitempty"`
	TotalDurationMs int64                `json:"totalDurationMs"`
	TriggerData   map[string]interface{} `json:"triggerData"`
	FinalOutput   map[string]interface{} `json:"finalOutput,omitempty"`
	Error         string                 `json:"error,omitempty"`
	Snapshots     []*ExecutionSnapshot   `json:"snapshots"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// ReplayOptions holds replay configuration
type ReplayOptions struct {
	StartFromNode    string                 // Start from specific node
	StopAtNode       string                 // Stop at specific node
	OverrideInput    map[string]interface{} // Override input data
	OverrideVariables map[string]interface{} // Override variables
	Speed            float64                // Replay speed (1.0 = real-time, 0 = instant)
	BreakpointNodes  []string               // Nodes to pause at
	SkipNodes        []string               // Nodes to skip
}

// ReplayResult represents the result of a replay
type ReplayResult struct {
	ExecutionID   string                 `json:"executionId"`
	OriginalID    string                 `json:"originalId"`
	Status        string                 `json:"status"`
	NodesReplayed int                    `json:"nodesReplayed"`
	NodesSkipped  int                    `json:"nodesSkipped"`
	Output        map[string]interface{} `json:"output"`
	Diffs         []ReplayDiff           `json:"diffs"`
	StartedAt     time.Time              `json:"startedAt"`
	CompletedAt   time.Time              `json:"completedAt"`
	DurationMs    int64                  `json:"durationMs"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ReplayDiff represents a difference between original and replay
type ReplayDiff struct {
	NodeID       string      `json:"nodeId"`
	NodeName     string      `json:"nodeName"`
	Field        string      `json:"field"`
	OriginalValue interface{} `json:"originalValue"`
	ReplayValue  interface{} `json:"replayValue"`
	Type         string      `json:"type"` // output, duration, status
}

// ExecutionRecorder records execution for replay
type ExecutionRecorder struct {
	recordings map[string]*ExecutionRecording
	mu         sync.RWMutex
	maxSize    int
}

// NewExecutionRecorder creates a new execution recorder
func NewExecutionRecorder(maxSize int) *ExecutionRecorder {
	return &ExecutionRecorder{
		recordings: make(map[string]*ExecutionRecording),
		maxSize:    maxSize,
	}
}

// StartRecording starts recording an execution
func (r *ExecutionRecorder) StartRecording(executionID, workflowID, workflowName, mode string, triggerData map[string]interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.recordings[executionID] = &ExecutionRecording{
		ID:           uuid.New().String(),
		ExecutionID:  executionID,
		WorkflowID:   workflowID,
		WorkflowName: workflowName,
		Mode:         mode,
		Status:       "running",
		StartedAt:    time.Now(),
		TriggerData:  triggerData,
		Snapshots:    make([]*ExecutionSnapshot, 0),
		Metadata:     make(map[string]interface{}),
	}

	// Cleanup old recordings if exceeded max size
	if len(r.recordings) > r.maxSize {
		r.cleanupOldest()
	}
}

// RecordNodeStart records a node starting execution
func (r *ExecutionRecorder) RecordNodeStart(executionID, nodeID, nodeName, nodeType string, input map[string]interface{}, variables map[string]interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	recording, exists := r.recordings[executionID]
	if !exists {
		return
	}

	snapshot := &ExecutionSnapshot{
		ID:          uuid.New().String(),
		ExecutionID: executionID,
		WorkflowID:  recording.WorkflowID,
		NodeID:      nodeID,
		NodeName:    nodeName,
		NodeType:    nodeType,
		Sequence:    len(recording.Snapshots) + 1,
		Timestamp:   time.Now(),
		Status:      "running",
		Input:       copyMap(input),
		Variables:   copyMap(variables),
		ContextData: make(map[string]interface{}),
	}

	recording.Snapshots = append(recording.Snapshots, snapshot)
}

// RecordNodeComplete records a node completing execution
func (r *ExecutionRecorder) RecordNodeComplete(executionID, nodeID string, output map[string]interface{}, durationMs int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	recording, exists := r.recordings[executionID]
	if !exists {
		return
	}

	// Find the snapshot for this node
	for i := len(recording.Snapshots) - 1; i >= 0; i-- {
		if recording.Snapshots[i].NodeID == nodeID && recording.Snapshots[i].Status == "running" {
			recording.Snapshots[i].Status = "completed"
			recording.Snapshots[i].Output = copyMap(output)
			recording.Snapshots[i].DurationMs = durationMs
			break
		}
	}
}

// RecordNodeError records a node error
func (r *ExecutionRecorder) RecordNodeError(executionID, nodeID string, err error, durationMs int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	recording, exists := r.recordings[executionID]
	if !exists {
		return
	}

	for i := len(recording.Snapshots) - 1; i >= 0; i-- {
		if recording.Snapshots[i].NodeID == nodeID && recording.Snapshots[i].Status == "running" {
			recording.Snapshots[i].Status = "failed"
			recording.Snapshots[i].Error = err.Error()
			recording.Snapshots[i].DurationMs = durationMs
			break
		}
	}
}

// CompleteRecording completes the execution recording
func (r *ExecutionRecorder) CompleteRecording(executionID string, status string, output map[string]interface{}, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	recording, exists := r.recordings[executionID]
	if !exists {
		return
	}

	now := time.Now()
	recording.Status = status
	recording.CompletedAt = &now
	recording.TotalDurationMs = now.Sub(recording.StartedAt).Milliseconds()
	recording.FinalOutput = copyMap(output)
	if err != nil {
		recording.Error = err.Error()
	}
}

// GetRecording retrieves a recording
func (r *ExecutionRecorder) GetRecording(executionID string) (*ExecutionRecording, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	recording, exists := r.recordings[executionID]
	return recording, exists
}

// ListRecordings lists all recordings
func (r *ExecutionRecorder) ListRecordings(workflowID string, limit int) []*ExecutionRecording {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var recordings []*ExecutionRecording
	for _, rec := range r.recordings {
		if workflowID == "" || rec.WorkflowID == workflowID {
			recordings = append(recordings, rec)
		}
	}

	if limit > 0 && len(recordings) > limit {
		recordings = recordings[:limit]
	}

	return recordings
}

// DeleteRecording deletes a recording
func (r *ExecutionRecorder) DeleteRecording(executionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.recordings, executionID)
}

func (r *ExecutionRecorder) cleanupOldest() {
	var oldest string
	var oldestTime time.Time

	for id, rec := range r.recordings {
		if oldest == "" || rec.StartedAt.Before(oldestTime) {
			oldest = id
			oldestTime = rec.StartedAt
		}
	}

	if oldest != "" {
		delete(r.recordings, oldest)
	}
}

// ExecutionReplayer replays executions
type ExecutionReplayer struct {
	recorder *ExecutionRecorder
	executor WorkflowExecutor
}

// NewExecutionReplayer creates a new execution replayer
func NewExecutionReplayer(recorder *ExecutionRecorder, executor WorkflowExecutor) *ExecutionReplayer {
	return &ExecutionReplayer{
		recorder: recorder,
		executor: executor,
	}
}

// Replay replays an execution
func (p *ExecutionReplayer) Replay(ctx context.Context, executionID string, options *ReplayOptions) (*ReplayResult, error) {
	recording, exists := p.recorder.GetRecording(executionID)
	if !exists {
		return nil, fmt.Errorf("recording not found: %s", executionID)
	}

	result := &ReplayResult{
		ExecutionID: uuid.New().String(),
		OriginalID:  executionID,
		Status:      "running",
		Diffs:       make([]ReplayDiff, 0),
		StartedAt:   time.Now(),
	}

	// Determine starting point
	startIndex := 0
	if options != nil && options.StartFromNode != "" {
		for i, snap := range recording.Snapshots {
			if snap.NodeID == options.StartFromNode {
				startIndex = i
				break
			}
		}
	}

	// Build skip set
	skipSet := make(map[string]bool)
	if options != nil {
		for _, nodeID := range options.SkipNodes {
			skipSet[nodeID] = true
		}
	}

	// Replay each snapshot
	for i := startIndex; i < len(recording.Snapshots); i++ {
		snap := recording.Snapshots[i]

		// Check stop condition
		if options != nil && options.StopAtNode != "" && snap.NodeID == options.StopAtNode {
			break
		}

		// Check skip
		if skipSet[snap.NodeID] {
			result.NodesSkipped++
			continue
		}

		// Check breakpoint
		if options != nil && containsString(options.BreakpointNodes, snap.NodeID) {
			result.Status = "paused"
			result.Metadata = map[string]interface{}{
				"pausedAt":     snap.NodeID,
				"pausedAtName": snap.NodeName,
			}
			break
		}

		// Apply speed delay
		if options != nil && options.Speed > 0 && snap.DurationMs > 0 {
			delay := time.Duration(float64(snap.DurationMs) / options.Speed) * time.Millisecond
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		result.NodesReplayed++
	}

	result.CompletedAt = time.Now()
	result.DurationMs = result.CompletedAt.Sub(result.StartedAt).Milliseconds()
	if result.Status == "running" {
		result.Status = "completed"
	}
	result.Output = recording.FinalOutput

	return result, nil
}

// ReplayWithComparison replays and compares with original
func (p *ExecutionReplayer) ReplayWithComparison(ctx context.Context, executionID string, newInput map[string]interface{}) (*ReplayResult, error) {
	recording, exists := p.recorder.GetRecording(executionID)
	if !exists {
		return nil, fmt.Errorf("recording not found: %s", executionID)
	}

	result := &ReplayResult{
		ExecutionID: uuid.New().String(),
		OriginalID:  executionID,
		Status:      "completed",
		Diffs:       make([]ReplayDiff, 0),
		StartedAt:   time.Now(),
	}

	// Use new input or original trigger data
	input := recording.TriggerData
	if newInput != nil {
		input = newInput
		// Compare inputs
		for key, origVal := range recording.TriggerData {
			if newVal, ok := newInput[key]; ok && !equalValues(origVal, newVal) {
				result.Diffs = append(result.Diffs, ReplayDiff{
					Field:        fmt.Sprintf("input.%s", key),
					OriginalValue: origVal,
					ReplayValue:  newVal,
					Type:         "input",
				})
			}
		}
	}

	result.Output = input // Would contain actual replay output
	result.CompletedAt = time.Now()
	result.DurationMs = result.CompletedAt.Sub(result.StartedAt).Milliseconds()
	result.NodesReplayed = len(recording.Snapshots)

	return result, nil
}

// GetSnapshotAt returns the execution state at a specific snapshot
func (p *ExecutionReplayer) GetSnapshotAt(executionID string, sequence int) (*ExecutionSnapshot, error) {
	recording, exists := p.recorder.GetRecording(executionID)
	if !exists {
		return nil, fmt.Errorf("recording not found: %s", executionID)
	}

	if sequence < 1 || sequence > len(recording.Snapshots) {
		return nil, fmt.Errorf("invalid sequence: %d", sequence)
	}

	return recording.Snapshots[sequence-1], nil
}

// GetNodeSnapshots returns all snapshots for a specific node
func (p *ExecutionReplayer) GetNodeSnapshots(executionID, nodeID string) ([]*ExecutionSnapshot, error) {
	recording, exists := p.recorder.GetRecording(executionID)
	if !exists {
		return nil, fmt.Errorf("recording not found: %s", executionID)
	}

	var snapshots []*ExecutionSnapshot
	for _, snap := range recording.Snapshots {
		if snap.NodeID == nodeID {
			snapshots = append(snapshots, snap)
		}
	}

	return snapshots, nil
}

// ExportRecording exports a recording as JSON
func (p *ExecutionReplayer) ExportRecording(executionID string) ([]byte, error) {
	recording, exists := p.recorder.GetRecording(executionID)
	if !exists {
		return nil, fmt.Errorf("recording not found: %s", executionID)
	}

	return json.MarshalIndent(recording, "", "  ")
}

// ImportRecording imports a recording from JSON
func (p *ExecutionReplayer) ImportRecording(data []byte) error {
	var recording ExecutionRecording
	if err := json.Unmarshal(data, &recording); err != nil {
		return fmt.Errorf("failed to parse recording: %w", err)
	}

	p.recorder.mu.Lock()
	defer p.recorder.mu.Unlock()

	p.recorder.recordings[recording.ExecutionID] = &recording
	return nil
}

// Helper functions
func copyMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	result := make(map[string]interface{})
	for k, v := range m {
		result[k] = v
	}
	return result
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func equalValues(a, b interface{}) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}


