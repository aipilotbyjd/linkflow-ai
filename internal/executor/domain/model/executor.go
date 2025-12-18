package model

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ExecutionEnvironment represents the sandbox environment type
type ExecutionEnvironment string

const (
	EnvV8Isolate   ExecutionEnvironment = "v8_isolate"
	EnvWebAssembly ExecutionEnvironment = "wasm"
	EnvContainer   ExecutionEnvironment = "container"
	EnvNative      ExecutionEnvironment = "native"
)

// ResourceConstraints defines limits for execution
type ResourceConstraints struct {
	MaxCPUTime      time.Duration `json:"maxCpuTime"`
	MaxMemoryMB     int           `json:"maxMemoryMB"`
	MaxNetworkCalls int           `json:"maxNetworkCalls"`
	MaxFileSizeMB   int           `json:"maxFileSizeMB"`
	AllowNetwork    bool          `json:"allowNetwork"`
	AllowFileSystem bool          `json:"allowFileSystem"`
	Timeout         time.Duration `json:"timeout"`
}

// DefaultConstraints returns default resource constraints
func DefaultConstraints() ResourceConstraints {
	return ResourceConstraints{
		MaxCPUTime:      30 * time.Second,
		MaxMemoryMB:     128,
		MaxNetworkCalls: 10,
		MaxFileSizeMB:   10,
		AllowNetwork:    true,
		AllowFileSystem: false,
		Timeout:         60 * time.Second,
	}
}

// SandboxConfig configures the sandbox environment
type SandboxConfig struct {
	Environment  ExecutionEnvironment `json:"environment"`
	Constraints  ResourceConstraints  `json:"constraints"`
	AllowedHosts []string             `json:"allowedHosts"`
	EnvVars      map[string]string    `json:"envVars"`
	SecretRefs   []string             `json:"secretRefs"`
}

// NodeExecutionRequest represents a request to execute a node
type NodeExecutionRequest struct {
	ID            string                 `json:"id"`
	ExecutionID   string                 `json:"executionId"`
	WorkflowID    string                 `json:"workflowId"`
	NodeID        string                 `json:"nodeId"`
	NodeType      string                 `json:"nodeType"`
	Code          string                 `json:"code"`
	Language      string                 `json:"language"`
	Input         map[string]interface{} `json:"input"`
	Context       map[string]interface{} `json:"context"`
	Config        SandboxConfig          `json:"config"`
	Credentials   map[string]string      `json:"credentials"`
	Priority      int                    `json:"priority"`
	RetryCount    int                    `json:"retryCount"`
	MaxRetries    int                    `json:"maxRetries"`
	CreatedAt     time.Time              `json:"createdAt"`
}

// NewNodeExecutionRequest creates a new execution request
func NewNodeExecutionRequest(execID, workflowID, nodeID, nodeType string, input map[string]interface{}) *NodeExecutionRequest {
	return &NodeExecutionRequest{
		ID:          uuid.New().String(),
		ExecutionID: execID,
		WorkflowID:  workflowID,
		NodeID:      nodeID,
		NodeType:    nodeType,
		Input:       input,
		Config: SandboxConfig{
			Environment: EnvNative,
			Constraints: DefaultConstraints(),
		},
		Priority:   1,
		MaxRetries: 3,
		CreatedAt:  time.Now(),
	}
}

// NodeExecutionResult represents the result of a node execution
type NodeExecutionResult struct {
	ID           string                 `json:"id"`
	RequestID    string                 `json:"requestId"`
	Status       ExecutionStatus        `json:"status"`
	Output       map[string]interface{} `json:"output"`
	Error        *ExecutionError        `json:"error"`
	Logs         []LogEntry             `json:"logs"`
	Metrics      ExecutionMetrics       `json:"metrics"`
	StartedAt    time.Time              `json:"startedAt"`
	CompletedAt  time.Time              `json:"completedAt"`
}

// ExecutionStatus represents the status of an execution
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusTimeout   ExecutionStatus = "timeout"
	StatusCancelled ExecutionStatus = "cancelled"
)

// ExecutionError represents an execution error
type ExecutionError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StackTrace string `json:"stackTrace"`
	NodeID     string `json:"nodeId"`
	Retryable  bool   `json:"retryable"`
}

// LogEntry represents a log entry from execution
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	NodeID    string    `json:"nodeId"`
}

// ExecutionMetrics holds execution metrics
type ExecutionMetrics struct {
	CPUTimeMS      int64 `json:"cpuTimeMs"`
	MemoryUsedMB   int   `json:"memoryUsedMB"`
	NetworkCalls   int   `json:"networkCalls"`
	DurationMS     int64 `json:"durationMs"`
	BytesProcessed int64 `json:"bytesProcessed"`
}

// Sandbox represents an isolated execution environment
type Sandbox interface {
	Execute(ctx context.Context, req *NodeExecutionRequest) (*NodeExecutionResult, error)
	Cleanup() error
}

// SandboxPool manages a pool of sandboxes
type SandboxPool struct {
	available chan Sandbox
	maxSize   int
	factory   func() (Sandbox, error)
}

// NewSandboxPool creates a new sandbox pool
func NewSandboxPool(maxSize int, factory func() (Sandbox, error)) *SandboxPool {
	return &SandboxPool{
		available: make(chan Sandbox, maxSize),
		maxSize:   maxSize,
		factory:   factory,
	}
}

// Acquire gets a sandbox from the pool
func (p *SandboxPool) Acquire(ctx context.Context) (Sandbox, error) {
	select {
	case sb := <-p.available:
		return sb, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return p.factory()
	}
}

// Release returns a sandbox to the pool
func (p *SandboxPool) Release(sb Sandbox) {
	select {
	case p.available <- sb:
	default:
		sb.Cleanup()
	}
}

// NativeSandbox executes code in native Go environment
type NativeSandbox struct {
	constraints ResourceConstraints
}

// NewNativeSandbox creates a new native sandbox
func NewNativeSandbox(constraints ResourceConstraints) *NativeSandbox {
	return &NativeSandbox{constraints: constraints}
}

// Execute executes a node in the native sandbox
func (s *NativeSandbox) Execute(ctx context.Context, req *NodeExecutionRequest) (*NodeExecutionResult, error) {
	start := time.Now()
	result := &NodeExecutionResult{
		ID:        uuid.New().String(),
		RequestID: req.ID,
		Status:    StatusRunning,
		StartedAt: start,
		Logs:      make([]LogEntry, 0),
	}

	// Create timeout context
	execCtx, cancel := context.WithTimeout(ctx, s.constraints.Timeout)
	defer cancel()

	// Execute based on node type
	output, err := s.executeNode(execCtx, req)
	
	result.CompletedAt = time.Now()
	result.Metrics = ExecutionMetrics{
		DurationMS: time.Since(start).Milliseconds(),
	}

	if err != nil {
		result.Status = StatusFailed
		result.Error = &ExecutionError{
			Code:      "EXECUTION_ERROR",
			Message:   err.Error(),
			NodeID:    req.NodeID,
			Retryable: req.RetryCount < req.MaxRetries,
		}
		return result, nil
	}

	result.Status = StatusCompleted
	result.Output = output
	return result, nil
}

func (s *NativeSandbox) executeNode(ctx context.Context, req *NodeExecutionRequest) (map[string]interface{}, error) {
	select {
	case <-ctx.Done():
		return nil, errors.New("execution timeout")
	default:
	}

	// Simulate node execution based on type
	switch req.NodeType {
	case "http_request":
		return s.executeHTTPRequest(ctx, req)
	case "transform":
		return s.executeTransform(ctx, req)
	case "condition":
		return s.executeCondition(ctx, req)
	case "delay":
		return s.executeDelay(ctx, req)
	default:
		return req.Input, nil
	}
}

func (s *NativeSandbox) executeHTTPRequest(ctx context.Context, req *NodeExecutionRequest) (map[string]interface{}, error) {
	// Simulated HTTP request execution
	return map[string]interface{}{
		"statusCode": 200,
		"body":       map[string]interface{}{"success": true},
		"headers":    map[string]string{"Content-Type": "application/json"},
	}, nil
}

func (s *NativeSandbox) executeTransform(ctx context.Context, req *NodeExecutionRequest) (map[string]interface{}, error) {
	// Pass through with transformation marker
	output := make(map[string]interface{})
	for k, v := range req.Input {
		output[k] = v
	}
	output["_transformed"] = true
	return output, nil
}

func (s *NativeSandbox) executeCondition(ctx context.Context, req *NodeExecutionRequest) (map[string]interface{}, error) {
	// Evaluate condition
	condition, _ := req.Input["condition"].(bool)
	return map[string]interface{}{
		"result": condition,
		"branch": map[bool]string{true: "true", false: "false"}[condition],
	}, nil
}

func (s *NativeSandbox) executeDelay(ctx context.Context, req *NodeExecutionRequest) (map[string]interface{}, error) {
	delayMs, _ := req.Input["delayMs"].(float64)
	if delayMs > 0 && delayMs <= float64(s.constraints.Timeout.Milliseconds()) {
		select {
		case <-time.After(time.Duration(delayMs) * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return req.Input, nil
}

// Cleanup cleans up sandbox resources
func (s *NativeSandbox) Cleanup() error {
	return nil
}

// WorkerPool manages execution workers
type WorkerPool struct {
	workers    int
	queue      chan *NodeExecutionRequest
	results    chan *NodeExecutionResult
	sandboxes  *SandboxPool
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workers int, queueSize int, sandboxPool *SandboxPool) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		workers:   workers,
		queue:     make(chan *NodeExecutionRequest, queueSize),
		results:   make(chan *NodeExecutionResult, queueSize),
		sandboxes: sandboxPool,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start starts the worker pool
func (p *WorkerPool) Start() {
	for i := 0; i < p.workers; i++ {
		go p.worker(i)
	}
}

// Stop stops the worker pool
func (p *WorkerPool) Stop() {
	p.cancel()
}

// Submit submits a request to the pool
func (p *WorkerPool) Submit(req *NodeExecutionRequest) error {
	select {
	case p.queue <- req:
		return nil
	case <-p.ctx.Done():
		return errors.New("worker pool stopped")
	}
}

// Results returns the results channel
func (p *WorkerPool) Results() <-chan *NodeExecutionResult {
	return p.results
}

func (p *WorkerPool) worker(id int) {
	for {
		select {
		case req := <-p.queue:
			sandbox, err := p.sandboxes.Acquire(p.ctx)
			if err != nil {
				p.results <- &NodeExecutionResult{
					RequestID: req.ID,
					Status:    StatusFailed,
					Error:     &ExecutionError{Code: "SANDBOX_ERROR", Message: err.Error()},
				}
				continue
			}
			
			result, _ := sandbox.Execute(p.ctx, req)
			p.sandboxes.Release(sandbox)
			p.results <- result
			
		case <-p.ctx.Done():
			return
		}
	}
}
