// Package nodes provides built-in node implementations
package nodes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/node/runtime"
)

// HTTPRequestNode implements HTTP request functionality
type HTTPRequestNode struct {
	client *http.Client
}

// NewHTTPRequestNode creates a new HTTP request node
func NewHTTPRequestNode() *HTTPRequestNode {
	return &HTTPRequestNode{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetType returns the node type
func (n *HTTPRequestNode) GetType() string {
	return "http_request"
}

// GetMetadata returns node metadata
func (n *HTTPRequestNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "http_request",
		Name:        "HTTP Request",
		Description: "Make HTTP requests to external APIs and services",
		Category:    "core",
		Icon:        "globe",
		Color:       "#4CAF50",
		Version:     "1.0.0",
		Inputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Required: false, Description: "Input data"},
		},
		Outputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Description: "Response data"},
			{Name: "error", Type: "any", Description: "Error output"},
		},
		Properties: []runtime.PropertyDefinition{
			{Name: "method", Type: "select", Required: true, Default: "GET", Description: "HTTP method", Options: []runtime.PropertyOption{
				{Label: "GET", Value: "GET"},
				{Label: "POST", Value: "POST"},
				{Label: "PUT", Value: "PUT"},
				{Label: "PATCH", Value: "PATCH"},
				{Label: "DELETE", Value: "DELETE"},
				{Label: "HEAD", Value: "HEAD"},
				{Label: "OPTIONS", Value: "OPTIONS"},
			}},
			{Name: "url", Type: "string", Required: true, Description: "Request URL", Placeholder: "https://api.example.com/endpoint"},
			{Name: "authentication", Type: "select", Default: "none", Description: "Authentication type", Options: []runtime.PropertyOption{
				{Label: "None", Value: "none"},
				{Label: "Basic Auth", Value: "basic"},
				{Label: "Bearer Token", Value: "bearer"},
				{Label: "API Key", Value: "apiKey"},
				{Label: "OAuth2", Value: "oauth2"},
			}},
			{Name: "headers", Type: "json", Description: "Request headers"},
			{Name: "queryParams", Type: "json", Description: "Query parameters"},
			{Name: "body", Type: "json", Description: "Request body (for POST/PUT/PATCH)"},
			{Name: "bodyType", Type: "select", Default: "json", Description: "Body content type", Options: []runtime.PropertyOption{
				{Label: "JSON", Value: "json"},
				{Label: "Form Data", Value: "form"},
				{Label: "Form URL Encoded", Value: "urlencoded"},
				{Label: "Raw", Value: "raw"},
			}},
			{Name: "timeout", Type: "number", Default: 30, Description: "Request timeout in seconds"},
			{Name: "followRedirects", Type: "boolean", Default: true, Description: "Follow redirects"},
			{Name: "responseType", Type: "select", Default: "auto", Description: "Response type", Options: []runtime.PropertyOption{
				{Label: "Auto-detect", Value: "auto"},
				{Label: "JSON", Value: "json"},
				{Label: "Text", Value: "text"},
				{Label: "Binary", Value: "binary"},
			}},
		},
		IsTrigger: false,
	}
}

// Validate validates the node configuration
func (n *HTTPRequestNode) Validate(config map[string]interface{}) error {
	if _, ok := config["url"]; !ok {
		return fmt.Errorf("url is required")
	}
	return nil
}

// Execute executes the HTTP request
func (n *HTTPRequestNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	startTime := time.Now()
	output := &runtime.ExecutionOutput{
		Data: make(map[string]interface{}),
		Logs: []runtime.LogEntry{},
	}
	
	// Get configuration
	method := getStringConfig(input.NodeConfig, "method", "GET")
	urlStr := getStringConfig(input.NodeConfig, "url", "")
	headers := getMapConfig(input.NodeConfig, "headers")
	queryParams := getMapConfig(input.NodeConfig, "queryParams")
	body := input.NodeConfig["body"]
	bodyType := getStringConfig(input.NodeConfig, "bodyType", "json")
	timeout := getIntConfig(input.NodeConfig, "timeout", 30)
	authType := getStringConfig(input.NodeConfig, "authentication", "none")
	responseType := getStringConfig(input.NodeConfig, "responseType", "auto")
	
	// Build URL with query params
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		output.Error = fmt.Errorf("invalid URL: %w", err)
		return output, nil
	}
	
	if len(queryParams) > 0 {
		q := parsedURL.Query()
		for k, v := range queryParams {
			q.Set(k, fmt.Sprintf("%v", v))
		}
		parsedURL.RawQuery = q.Encode()
	}
	
	// Prepare request body
	var bodyReader io.Reader
	var contentType string
	
	if body != nil && (method == "POST" || method == "PUT" || method == "PATCH") {
		switch bodyType {
		case "json":
			jsonBody, err := json.Marshal(body)
			if err != nil {
				output.Error = fmt.Errorf("failed to marshal JSON body: %w", err)
				return output, nil
			}
			bodyReader = bytes.NewReader(jsonBody)
			contentType = "application/json"
		case "form", "urlencoded":
			formData := url.Values{}
			if m, ok := body.(map[string]interface{}); ok {
				for k, v := range m {
					formData.Set(k, fmt.Sprintf("%v", v))
				}
			}
			bodyReader = strings.NewReader(formData.Encode())
			contentType = "application/x-www-form-urlencoded"
		case "raw":
			bodyReader = strings.NewReader(fmt.Sprintf("%v", body))
			contentType = "text/plain"
		}
	}
	
	// Create request
	req, err := http.NewRequestWithContext(ctx, method, parsedURL.String(), bodyReader)
	if err != nil {
		output.Error = fmt.Errorf("failed to create request: %w", err)
		return output, nil
	}
	
	// Set headers
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	
	for k, v := range headers {
		req.Header.Set(k, fmt.Sprintf("%v", v))
	}
	
	// Apply authentication
	if err := n.applyAuthentication(req, authType, input.NodeConfig, input.Credentials); err != nil {
		output.Error = fmt.Errorf("authentication error: %w", err)
		return output, nil
	}
	
	// Set timeout
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}
	
	// Log request
	output.Logs = append(output.Logs, runtime.LogEntry{
		Level:     "info",
		Message:   fmt.Sprintf("%s %s", method, parsedURL.String()),
		Timestamp: time.Now().UnixMilli(),
		NodeID:    input.NodeID,
	})
	
	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		output.Error = fmt.Errorf("request failed: %w", err)
		return output, nil
	}
	defer resp.Body.Close()
	
	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		output.Error = fmt.Errorf("failed to read response: %w", err)
		return output, nil
	}
	
	// Parse response
	var responseData interface{}
	
	// Determine response type
	contentTypeHeader := resp.Header.Get("Content-Type")
	if responseType == "auto" {
		if strings.Contains(contentTypeHeader, "application/json") {
			responseType = "json"
		} else if strings.Contains(contentTypeHeader, "text/") {
			responseType = "text"
		} else {
			responseType = "binary"
		}
	}
	
	switch responseType {
	case "json":
		if err := json.Unmarshal(respBody, &responseData); err != nil {
			// If JSON parsing fails, return as text
			responseData = string(respBody)
		}
	case "text":
		responseData = string(respBody)
	case "binary":
		output.Binary = map[string][]byte{"body": respBody}
		responseData = map[string]interface{}{
			"size":     len(respBody),
			"mimeType": contentTypeHeader,
		}
	}
	
	// Build response headers map
	respHeaders := make(map[string]string)
	for k := range resp.Header {
		respHeaders[k] = resp.Header.Get(k)
	}
	
	// Set output data
	output.Data = map[string]interface{}{
		"statusCode":    resp.StatusCode,
		"statusMessage": resp.Status,
		"headers":       respHeaders,
		"body":          responseData,
		"ok":            resp.StatusCode >= 200 && resp.StatusCode < 300,
	}
	
	// Set metrics
	output.Metrics = runtime.ExecutionMetrics{
		StartTime:    startTime.UnixMilli(),
		EndTime:      time.Now().UnixMilli(),
		DurationMs:   time.Since(startTime).Milliseconds(),
		BytesRead:    int64(len(respBody)),
		BytesWritten: req.ContentLength,
	}
	
	// Log response
	output.Logs = append(output.Logs, runtime.LogEntry{
		Level:     "info",
		Message:   fmt.Sprintf("Response: %d %s", resp.StatusCode, resp.Status),
		Timestamp: time.Now().UnixMilli(),
		NodeID:    input.NodeID,
	})
	
	return output, nil
}

func (n *HTTPRequestNode) applyAuthentication(req *http.Request, authType string, config, credentials map[string]interface{}) error {
	switch authType {
	case "basic":
		username := getStringConfig(config, "basicAuthUser", "")
		password := getStringConfig(config, "basicAuthPassword", "")
		if credentials != nil {
			if u, ok := credentials["username"].(string); ok {
				username = u
			}
			if p, ok := credentials["password"].(string); ok {
				password = p
			}
		}
		req.SetBasicAuth(username, password)
		
	case "bearer":
		token := getStringConfig(config, "bearerToken", "")
		if credentials != nil {
			if t, ok := credentials["token"].(string); ok {
				token = t
			}
		}
		req.Header.Set("Authorization", "Bearer "+token)
		
	case "apiKey":
		keyName := getStringConfig(config, "apiKeyName", "X-API-Key")
		keyValue := getStringConfig(config, "apiKeyValue", "")
		keyLocation := getStringConfig(config, "apiKeyLocation", "header")
		
		if credentials != nil {
			if k, ok := credentials["key"].(string); ok {
				keyValue = k
			}
		}
		
		if keyLocation == "header" {
			req.Header.Set(keyName, keyValue)
		} else if keyLocation == "query" {
			q := req.URL.Query()
			q.Set(keyName, keyValue)
			req.URL.RawQuery = q.Encode()
		}
		
	case "oauth2":
		accessToken := ""
		if credentials != nil {
			if t, ok := credentials["access_token"].(string); ok {
				accessToken = t
			}
		}
		if accessToken != "" {
			req.Header.Set("Authorization", "Bearer "+accessToken)
		}
	}
	
	return nil
}

// Helper functions

func getStringConfig(config map[string]interface{}, key, defaultVal string) string {
	if v, ok := config[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultVal
}

func getIntConfig(config map[string]interface{}, key string, defaultVal int) int {
	if v, ok := config[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case float64:
			return int(val)
		}
	}
	return defaultVal
}

func getMapConfig(config map[string]interface{}, key string) map[string]interface{} {
	if v, ok := config[key]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			return m
		}
	}
	return make(map[string]interface{})
}

func init() {
	runtime.Register(NewHTTPRequestNode())
}
