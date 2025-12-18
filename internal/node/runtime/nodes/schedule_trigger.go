// Package nodes provides built-in node implementations
package nodes

import (
	"context"
	"fmt"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/node/runtime"
	"github.com/robfig/cron/v3"
)

// ScheduleTriggerNode implements scheduled workflow triggering
type ScheduleTriggerNode struct {
	scheduler *cron.Cron
	entries   map[string]cron.EntryID
}

// NewScheduleTriggerNode creates a new Schedule Trigger node
func NewScheduleTriggerNode() *ScheduleTriggerNode {
	return &ScheduleTriggerNode{
		scheduler: cron.New(cron.WithSeconds()),
		entries:   make(map[string]cron.EntryID),
	}
}

// GetType returns the node type
func (n *ScheduleTriggerNode) GetType() string {
	return "schedule_trigger"
}

// GetTriggerType returns the trigger type
func (n *ScheduleTriggerNode) GetTriggerType() runtime.TriggerType {
	return runtime.TriggerTypeSchedule
}

// GetMetadata returns node metadata
func (n *ScheduleTriggerNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "schedule_trigger",
		Name:        "Schedule Trigger",
		Description: "Trigger workflow on a schedule (cron or interval)",
		Category:    "trigger",
		Icon:        "calendar",
		Color:       "#FF5722",
		Version:     "1.0.0",
		Outputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Description: "Trigger data"},
		},
		Properties: []runtime.PropertyDefinition{
			{Name: "mode", Type: "select", Default: "interval", Description: "Schedule mode", Options: []runtime.PropertyOption{
				{Label: "Interval", Value: "interval"},
				{Label: "Cron Expression", Value: "cron"},
			}},
			{Name: "interval", Type: "number", Default: 60, Description: "Interval in seconds (for interval mode)"},
			{Name: "cronExpression", Type: "string", Description: "Cron expression (for cron mode)", Placeholder: "0 0 * * * *"},
			{Name: "timezone", Type: "string", Default: "UTC", Description: "Timezone for schedule"},
		},
		IsTrigger: true,
	}
}

// Validate validates the node configuration
func (n *ScheduleTriggerNode) Validate(config map[string]interface{}) error {
	mode := getStringConfig(config, "mode", "interval")
	
	if mode == "cron" {
		cronExpr := getStringConfig(config, "cronExpression", "")
		if cronExpr == "" {
			return fmt.Errorf("cron expression is required for cron mode")
		}
		
		// Validate cron expression
		parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		if _, err := parser.Parse(cronExpr); err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
	} else {
		interval := getIntConfig(config, "interval", 60)
		if interval < 1 {
			return fmt.Errorf("interval must be at least 1 second")
		}
	}
	
	return nil
}

// Execute is called when schedule triggers
func (n *ScheduleTriggerNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	startTime := time.Now()
	output := &runtime.ExecutionOutput{
		Data: map[string]interface{}{
			"timestamp":    time.Now().Format(time.RFC3339),
			"triggerType":  "schedule",
			"scheduled":    true,
		},
		Logs: []runtime.LogEntry{
			{
				Level:     "info",
				Message:   "Schedule triggered",
				Timestamp: time.Now().UnixMilli(),
				NodeID:    input.NodeID,
			},
		},
	}
	
	output.Metrics = runtime.ExecutionMetrics{
		StartTime:  startTime.UnixMilli(),
		EndTime:    time.Now().UnixMilli(),
		DurationMs: time.Since(startTime).Milliseconds(),
	}
	
	return output, nil
}

// Start starts the schedule trigger
func (n *ScheduleTriggerNode) Start(ctx context.Context, config map[string]interface{}, callback runtime.TriggerCallback) error {
	workflowID := getStringConfig(config, "workflowId", "")
	mode := getStringConfig(config, "mode", "interval")
	
	var schedule string
	if mode == "cron" {
		schedule = getStringConfig(config, "cronExpression", "0 * * * * *")
	} else {
		interval := getIntConfig(config, "interval", 60)
		schedule = fmt.Sprintf("@every %ds", interval)
	}
	
	// Add cron job
	entryID, err := n.scheduler.AddFunc(schedule, func() {
		data := map[string]interface{}{
			"timestamp":   time.Now().Format(time.RFC3339),
			"triggerType": "schedule",
			"mode":        mode,
		}
		callback(data)
	})
	
	if err != nil {
		return fmt.Errorf("failed to add schedule: %w", err)
	}
	
	n.entries[workflowID] = entryID
	
	// Start scheduler if not running
	n.scheduler.Start()
	
	return nil
}

// Stop stops the schedule trigger
func (n *ScheduleTriggerNode) Stop(ctx context.Context) error {
	n.scheduler.Stop()
	return nil
}

// StopWorkflow stops a specific workflow's schedule
func (n *ScheduleTriggerNode) StopWorkflow(workflowID string) {
	if entryID, exists := n.entries[workflowID]; exists {
		n.scheduler.Remove(entryID)
		delete(n.entries, workflowID)
	}
}

// Global schedule trigger instance
var scheduleTrigger *ScheduleTriggerNode

func init() {
	scheduleTrigger = NewScheduleTriggerNode()
	runtime.Register(scheduleTrigger)
}

// GetScheduleTrigger returns the global schedule trigger
func GetScheduleTrigger() *ScheduleTriggerNode {
	return scheduleTrigger
}

// IntervalTriggerNode is an alias for schedule with interval mode
type IntervalTriggerNode struct {
	*ScheduleTriggerNode
}

// NewIntervalTriggerNode creates a new Interval Trigger node
func NewIntervalTriggerNode() *IntervalTriggerNode {
	return &IntervalTriggerNode{
		ScheduleTriggerNode: NewScheduleTriggerNode(),
	}
}

// GetType returns the node type
func (n *IntervalTriggerNode) GetType() string {
	return "interval_trigger"
}

// GetMetadata returns node metadata
func (n *IntervalTriggerNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "interval_trigger",
		Name:        "Interval",
		Description: "Trigger workflow at regular intervals",
		Category:    "trigger",
		Icon:        "refresh-cw",
		Color:       "#FF5722",
		Version:     "1.0.0",
		Outputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Description: "Trigger data"},
		},
		Properties: []runtime.PropertyDefinition{
			{Name: "interval", Type: "number", Default: 60, Required: true, Description: "Interval in seconds"},
			{Name: "unit", Type: "select", Default: "seconds", Description: "Time unit", Options: []runtime.PropertyOption{
				{Label: "Seconds", Value: "seconds"},
				{Label: "Minutes", Value: "minutes"},
				{Label: "Hours", Value: "hours"},
			}},
		},
		IsTrigger: true,
	}
}

func init() {
	runtime.Register(NewIntervalTriggerNode())
}

// ManualTriggerNode is triggered manually
type ManualTriggerNode struct{}

// NewManualTriggerNode creates a new Manual Trigger node
func NewManualTriggerNode() *ManualTriggerNode {
	return &ManualTriggerNode{}
}

// GetType returns the node type
func (n *ManualTriggerNode) GetType() string {
	return "manual_trigger"
}

// GetMetadata returns node metadata
func (n *ManualTriggerNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "manual_trigger",
		Name:        "Manual Trigger",
		Description: "Manually trigger the workflow",
		Category:    "trigger",
		Icon:        "play",
		Color:       "#4CAF50",
		Version:     "1.0.0",
		Outputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Description: "Input data"},
		},
		Properties: []runtime.PropertyDefinition{
			{Name: "testData", Type: "json", Description: "Test data for manual execution"},
		},
		IsTrigger: true,
	}
}

// Validate validates the node configuration
func (n *ManualTriggerNode) Validate(config map[string]interface{}) error {
	return nil
}

// Execute is called on manual trigger
func (n *ManualTriggerNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	startTime := time.Now()
	
	data := input.InputData
	if data == nil {
		data = make(map[string]interface{})
	}
	
	data["timestamp"] = time.Now().Format(time.RFC3339)
	data["triggerType"] = "manual"
	
	output := &runtime.ExecutionOutput{
		Data: data,
		Logs: []runtime.LogEntry{
			{
				Level:     "info",
				Message:   "Manual trigger executed",
				Timestamp: time.Now().UnixMilli(),
				NodeID:    input.NodeID,
			},
		},
		Metrics: runtime.ExecutionMetrics{
			StartTime:  startTime.UnixMilli(),
			EndTime:    time.Now().UnixMilli(),
			DurationMs: time.Since(startTime).Milliseconds(),
		},
	}
	
	return output, nil
}

func init() {
	runtime.Register(NewManualTriggerNode())
}
