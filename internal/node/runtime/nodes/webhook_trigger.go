// Package nodes provides built-in node implementations
package nodes

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/linkflow-ai/linkflow-ai/internal/node/runtime"
)

// WebhookTriggerNode implements webhook trigger functionality
type WebhookTriggerNode struct {
	mu        sync.RWMutex
	callbacks map[string]runtime.TriggerCallback
	paths     map[string]webhookConfig
}

type webhookConfig struct {
	workflowID string
	method     string
	path       string
	secret     string
	headers    map[string]string
}

// Global webhook handler instance
var webhookHandler *WebhookTriggerNode

// NewWebhookTriggerNode creates a new webhook trigger node
func NewWebhookTriggerNode() *WebhookTriggerNode {
	return &WebhookTriggerNode{
		callbacks: make(map[string]runtime.TriggerCallback),
		paths:     make(map[string]webhookConfig),
	}
}

// GetType returns the node type
func (n *WebhookTriggerNode) GetType() string {
	return "webhook_trigger"
}

// GetTriggerType returns the trigger type
func (n *WebhookTriggerNode) GetTriggerType() runtime.TriggerType {
	return runtime.TriggerTypeWebhook
}

// GetMetadata returns node metadata
func (n *WebhookTriggerNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "webhook_trigger",
		Name:        "Webhook",
		Description: "Trigger workflow when receiving HTTP requests",
		Category:    "trigger",
		Icon:        "webhook",
		Color:       "#9C27B0",
		Version:     "1.0.0",
		Outputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Description: "Webhook data"},
		},
		Properties: []runtime.PropertyDefinition{
			{Name: "httpMethod", Type: "select", Required: true, Default: "POST", Description: "HTTP method to accept", Options: []runtime.PropertyOption{
				{Label: "GET", Value: "GET"},
				{Label: "POST", Value: "POST"},
				{Label: "PUT", Value: "PUT"},
				{Label: "PATCH", Value: "PATCH"},
				{Label: "DELETE", Value: "DELETE"},
				{Label: "ANY", Value: "ANY"},
			}},
			{Name: "path", Type: "string", Description: "Custom webhook path (auto-generated if empty)"},
			{Name: "authentication", Type: "select", Default: "none", Description: "Authentication method", Options: []runtime.PropertyOption{
				{Label: "None", Value: "none"},
				{Label: "Basic Auth", Value: "basic"},
				{Label: "Header Auth", Value: "header"},
				{Label: "HMAC Signature", Value: "hmac"},
			}},
			{Name: "responseMode", Type: "select", Default: "onReceived", Description: "When to respond", Options: []runtime.PropertyOption{
				{Label: "When received", Value: "onReceived"},
				{Label: "When execution finishes", Value: "onFinished"},
				{Label: "Custom response node", Value: "custom"},
			}},
			{Name: "responseCode", Type: "number", Default: 200, Description: "Response status code"},
			{Name: "responseData", Type: "string", Default: `{"success": true}`, Description: "Response body"},
			{Name: "responseContentType", Type: "string", Default: "application/json", Description: "Response content type"},
		},
		IsTrigger: true,
	}
}

// Validate validates the node configuration
func (n *WebhookTriggerNode) Validate(config map[string]interface{}) error {
	return nil
}

// Execute is called when webhook is triggered
func (n *WebhookTriggerNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	// For triggers, Execute processes the incoming webhook data
	output := &runtime.ExecutionOutput{
		Data: input.InputData,
		Logs: []runtime.LogEntry{
			{
				Level:     "info",
				Message:   "Webhook triggered",
				Timestamp: time.Now().UnixMilli(),
				NodeID:    input.NodeID,
			},
		},
	}
	
	return output, nil
}

// Start starts the webhook trigger
func (n *WebhookTriggerNode) Start(ctx context.Context, config map[string]interface{}, callback runtime.TriggerCallback) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	workflowID := getStringConfig(config, "workflowId", "")
	method := getStringConfig(config, "httpMethod", "POST")
	path := getStringConfig(config, "path", "")
	
	if path == "" {
		path = uuid.New().String()
	}
	
	// Normalize path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	
	// Store callback and config
	n.callbacks[workflowID] = callback
	n.paths[path] = webhookConfig{
		workflowID: workflowID,
		method:     method,
		path:       path,
		secret:     getStringConfig(config, "secret", ""),
	}
	
	return nil
}

// Stop stops the webhook trigger
func (n *WebhookTriggerNode) Stop(ctx context.Context) error {
	// Cleanup would be done in production
	return nil
}

// HandleWebhook handles incoming webhook requests
func (n *WebhookTriggerNode) HandleWebhook(w http.ResponseWriter, r *http.Request, path string) {
	n.mu.RLock()
	config, exists := n.paths[path]
	if !exists {
		n.mu.RUnlock()
		http.Error(w, "Webhook not found", http.StatusNotFound)
		return
	}
	
	callback := n.callbacks[config.workflowID]
	n.mu.RUnlock()
	
	// Check method
	if config.method != "ANY" && r.Method != config.method {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Verify signature if configured
	if config.secret != "" {
		signature := r.Header.Get("X-Webhook-Signature")
		if signature == "" {
			signature = r.Header.Get("X-Hub-Signature-256")
		}
		
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		
		if !verifySignature(body, signature, config.secret) {
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}
		
		// Reset body for parsing
		r.Body = io.NopCloser(strings.NewReader(string(body)))
	}
	
	// Parse request data
	webhookData := map[string]interface{}{
		"method":  r.Method,
		"path":    r.URL.Path,
		"query":   parseQuery(r.URL.Query()),
		"headers": parseHeaders(r.Header),
	}
	
	// Parse body
	if r.Method != "GET" && r.Method != "HEAD" {
		body, err := io.ReadAll(r.Body)
		if err == nil && len(body) > 0 {
			contentType := r.Header.Get("Content-Type")
			
			if strings.Contains(contentType, "application/json") {
				var jsonBody interface{}
				if err := json.Unmarshal(body, &jsonBody); err == nil {
					webhookData["body"] = jsonBody
				} else {
					webhookData["body"] = string(body)
				}
			} else if strings.Contains(contentType, "application/x-www-form-urlencoded") {
				r.Body = io.NopCloser(strings.NewReader(string(body)))
				r.ParseForm()
				formData := make(map[string]interface{})
				for k, v := range r.Form {
					if len(v) == 1 {
						formData[k] = v[0]
					} else {
						formData[k] = v
					}
				}
				webhookData["body"] = formData
			} else {
				webhookData["body"] = string(body)
			}
		}
	}
	
	// Call workflow
	if callback != nil {
		if err := callback(webhookData); err != nil {
			http.Error(w, "Execution failed", http.StatusInternalServerError)
			return
		}
	}
	
	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GetWebhookURL returns the full webhook URL for a workflow
func (n *WebhookTriggerNode) GetWebhookURL(baseURL, path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return fmt.Sprintf("%s/webhook%s", strings.TrimSuffix(baseURL, "/"), path)
}

// Helper functions

func parseQuery(query map[string][]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range query {
		if len(v) == 1 {
			result[k] = v[0]
		} else {
			result[k] = v
		}
	}
	return result
}

func parseHeaders(headers http.Header) map[string]string {
	result := make(map[string]string)
	for k := range headers {
		result[k] = headers.Get(k)
	}
	return result
}

func verifySignature(body []byte, signature, secret string) bool {
	if signature == "" {
		return false
	}
	
	// Remove prefix if present
	signature = strings.TrimPrefix(signature, "sha256=")
	
	// Calculate expected signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	
	return hmac.Equal([]byte(signature), []byte(expected))
}

func init() {
	webhookHandler = NewWebhookTriggerNode()
	runtime.Register(webhookHandler)
}

// GetWebhookHandler returns the global webhook handler
func GetWebhookHandler() *WebhookTriggerNode {
	return webhookHandler
}
