// Package nodes provides built-in node implementations
package nodes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/node/runtime"
)

// SlackNode implements Slack integration
type SlackNode struct {
	client *http.Client
}

// NewSlackNode creates a new Slack node
func NewSlackNode() *SlackNode {
	return &SlackNode{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// GetType returns the node type
func (n *SlackNode) GetType() string {
	return "slack"
}

// GetMetadata returns node metadata
func (n *SlackNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "slack",
		Name:        "Slack",
		Description: "Send messages and interact with Slack",
		Category:    "integration",
		Icon:        "slack",
		Color:       "#4A154B",
		Version:     "1.0.0",
		Inputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Description: "Input data"},
		},
		Outputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Description: "Slack response"},
		},
		Properties: []runtime.PropertyDefinition{
			{Name: "operation", Type: "select", Required: true, Default: "sendMessage", Description: "Operation", Options: []runtime.PropertyOption{
				{Label: "Send Message", Value: "sendMessage"},
				{Label: "Send Message (Webhook)", Value: "sendWebhook"},
				{Label: "Update Message", Value: "updateMessage"},
				{Label: "Delete Message", Value: "deleteMessage"},
				{Label: "Upload File", Value: "uploadFile"},
				{Label: "Get User", Value: "getUser"},
				{Label: "Get Channel", Value: "getChannel"},
				{Label: "List Channels", Value: "listChannels"},
			}},
			{Name: "channel", Type: "string", Description: "Channel ID or name"},
			{Name: "text", Type: "string", Description: "Message text"},
			{Name: "blocks", Type: "json", Description: "Block Kit blocks (advanced formatting)"},
			{Name: "attachments", Type: "json", Description: "Message attachments"},
			{Name: "webhookUrl", Type: "string", Description: "Webhook URL (for webhook operation)"},
			{Name: "threadTs", Type: "string", Description: "Thread timestamp (for replies)"},
			{Name: "messageTs", Type: "string", Description: "Message timestamp (for update/delete)"},
			{Name: "userId", Type: "string", Description: "User ID (for user operations)"},
		},
		IsTrigger: false,
		IsPremium: false,
	}
}

// Validate validates the node configuration
func (n *SlackNode) Validate(config map[string]interface{}) error {
	operation := getStringConfig(config, "operation", "sendMessage")
	
	switch operation {
	case "sendMessage":
		if getStringConfig(config, "channel", "") == "" {
			return fmt.Errorf("channel is required for sendMessage")
		}
		if getStringConfig(config, "text", "") == "" && config["blocks"] == nil {
			return fmt.Errorf("text or blocks required for sendMessage")
		}
	case "sendWebhook":
		if getStringConfig(config, "webhookUrl", "") == "" {
			return fmt.Errorf("webhookUrl is required for sendWebhook")
		}
	}
	
	return nil
}

// Execute executes the Slack node
func (n *SlackNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	startTime := time.Now()
	output := &runtime.ExecutionOutput{
		Data: make(map[string]interface{}),
		Logs: []runtime.LogEntry{},
	}
	
	operation := getStringConfig(input.NodeConfig, "operation", "sendMessage")
	
	// Get token from credentials
	token := ""
	if input.Credentials != nil {
		token, _ = input.Credentials["accessToken"].(string)
		if token == "" {
			token, _ = input.Credentials["token"].(string)
		}
	}
	
	var result map[string]interface{}
	var err error
	
	switch operation {
	case "sendMessage":
		result, err = n.sendMessage(ctx, token, input.NodeConfig)
	case "sendWebhook":
		result, err = n.sendWebhook(ctx, input.NodeConfig)
	case "updateMessage":
		result, err = n.updateMessage(ctx, token, input.NodeConfig)
	case "deleteMessage":
		result, err = n.deleteMessage(ctx, token, input.NodeConfig)
	case "getUser":
		result, err = n.getUser(ctx, token, input.NodeConfig)
	case "getChannel":
		result, err = n.getChannel(ctx, token, input.NodeConfig)
	case "listChannels":
		result, err = n.listChannels(ctx, token)
	default:
		err = fmt.Errorf("unknown operation: %s", operation)
	}
	
	if err != nil {
		output.Error = err
		output.Logs = append(output.Logs, runtime.LogEntry{
			Level:     "error",
			Message:   fmt.Sprintf("Slack operation failed: %v", err),
			Timestamp: time.Now().UnixMilli(),
			NodeID:    input.NodeID,
		})
		return output, nil
	}
	
	output.Data = result
	output.Logs = append(output.Logs, runtime.LogEntry{
		Level:     "info",
		Message:   fmt.Sprintf("Slack %s completed", operation),
		Timestamp: time.Now().UnixMilli(),
		NodeID:    input.NodeID,
	})
	
	output.Metrics = runtime.ExecutionMetrics{
		StartTime:  startTime.UnixMilli(),
		EndTime:    time.Now().UnixMilli(),
		DurationMs: time.Since(startTime).Milliseconds(),
	}
	
	return output, nil
}

func (n *SlackNode) sendMessage(ctx context.Context, token string, config map[string]interface{}) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"channel": getStringConfig(config, "channel", ""),
		"text":    getStringConfig(config, "text", ""),
	}
	
	if blocks := config["blocks"]; blocks != nil {
		payload["blocks"] = blocks
	}
	if attachments := config["attachments"]; attachments != nil {
		payload["attachments"] = attachments
	}
	if threadTs := getStringConfig(config, "threadTs", ""); threadTs != "" {
		payload["thread_ts"] = threadTs
	}
	
	return n.slackAPI(ctx, "POST", "https://slack.com/api/chat.postMessage", token, payload)
}

func (n *SlackNode) sendWebhook(ctx context.Context, config map[string]interface{}) (map[string]interface{}, error) {
	webhookURL := getStringConfig(config, "webhookUrl", "")
	
	payload := map[string]interface{}{
		"text": getStringConfig(config, "text", ""),
	}
	
	if blocks := config["blocks"]; blocks != nil {
		payload["blocks"] = blocks
	}
	if attachments := config["attachments"]; attachments != nil {
		payload["attachments"] = attachments
	}
	
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := n.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	respBody, _ := io.ReadAll(resp.Body)
	
	return map[string]interface{}{
		"ok":         resp.StatusCode == 200,
		"statusCode": resp.StatusCode,
		"response":   string(respBody),
	}, nil
}

func (n *SlackNode) updateMessage(ctx context.Context, token string, config map[string]interface{}) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"channel": getStringConfig(config, "channel", ""),
		"ts":      getStringConfig(config, "messageTs", ""),
		"text":    getStringConfig(config, "text", ""),
	}
	
	if blocks := config["blocks"]; blocks != nil {
		payload["blocks"] = blocks
	}
	
	return n.slackAPI(ctx, "POST", "https://slack.com/api/chat.update", token, payload)
}

func (n *SlackNode) deleteMessage(ctx context.Context, token string, config map[string]interface{}) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"channel": getStringConfig(config, "channel", ""),
		"ts":      getStringConfig(config, "messageTs", ""),
	}
	
	return n.slackAPI(ctx, "POST", "https://slack.com/api/chat.delete", token, payload)
}

func (n *SlackNode) getUser(ctx context.Context, token string, config map[string]interface{}) (map[string]interface{}, error) {
	userID := getStringConfig(config, "userId", "")
	url := fmt.Sprintf("https://slack.com/api/users.info?user=%s", userID)
	
	return n.slackAPI(ctx, "GET", url, token, nil)
}

func (n *SlackNode) getChannel(ctx context.Context, token string, config map[string]interface{}) (map[string]interface{}, error) {
	channelID := getStringConfig(config, "channel", "")
	url := fmt.Sprintf("https://slack.com/api/conversations.info?channel=%s", channelID)
	
	return n.slackAPI(ctx, "GET", url, token, nil)
}

func (n *SlackNode) listChannels(ctx context.Context, token string) (map[string]interface{}, error) {
	return n.slackAPI(ctx, "GET", "https://slack.com/api/conversations.list", token, nil)
}

func (n *SlackNode) slackAPI(ctx context.Context, method, url, token string, payload map[string]interface{}) (map[string]interface{}, error) {
	var body io.Reader
	if payload != nil {
		jsonBody, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(jsonBody)
	}
	
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := n.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	if ok, _ := result["ok"].(bool); !ok {
		errMsg, _ := result["error"].(string)
		return result, fmt.Errorf("Slack API error: %s", errMsg)
	}
	
	return result, nil
}

func init() {
	runtime.Register(NewSlackNode())
}
