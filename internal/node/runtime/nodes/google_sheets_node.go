// Package nodes provides Google Sheets node implementation
package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/node/runtime"
)

func init() {
	runtime.Register(&GoogleSheetsNode{})
}

// GoogleSheetsNode implements Google Sheets operations
type GoogleSheetsNode struct {
	client *http.Client
}

// GetType returns the node type
func (n *GoogleSheetsNode) GetType() string {
	return "google_sheets"
}

// Validate validates the node configuration
func (n *GoogleSheetsNode) Validate(config map[string]interface{}) error {
	if _, ok := config["operation"].(string); !ok {
		return fmt.Errorf("operation is required")
	}
	return nil
}

func (n *GoogleSheetsNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "google_sheets",
		Name:        "Google Sheets",
		Description: "Read and write data to Google Sheets",
		Category:    "integration",
		Version:     "1.0.0",
		Icon:        "google-sheets",
		Inputs: []runtime.PortDefinition{
			{Name: "main", Type: "main"},
		},
		Outputs: []runtime.PortDefinition{
			{Name: "main", Type: "main"},
		},
		Properties: []runtime.PropertyDefinition{
			{Name: "operation", Type: "select", Required: true, Options: []runtime.PropertyOption{
				{Label: "Read", Value: "read"}, {Label: "Write", Value: "write"}, {Label: "Append", Value: "append"}, {Label: "Clear", Value: "clear"}, {Label: "Create", Value: "create"},
			}},
			{Name: "spreadsheetId", Type: "string", Required: true},
			{Name: "range", Type: "string", Required: true},
			{Name: "values", Type: "json"},
			{Name: "valueInputOption", Type: "select", Default: "USER_ENTERED", Options: []runtime.PropertyOption{
				{Label: "RAW", Value: "RAW"}, {Label: "USER_ENTERED", Value: "USER_ENTERED"},
			}},
		},
	}
}

func (n *GoogleSheetsNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	if n.client == nil {
		n.client = &http.Client{Timeout: 30 * time.Second}
	}

	operation, _ := input.NodeConfig["operation"].(string)
	spreadsheetID, _ := input.NodeConfig["spreadsheetId"].(string)
	rangeStr, _ := input.NodeConfig["range"].(string)
	accessToken, _ := input.Credentials["access_token"].(string)

	if accessToken == "" {
		return nil, fmt.Errorf("access_token required")
	}

	baseURL := "https://sheets.googleapis.com/v4/spreadsheets"
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
		"Content-Type":  "application/json",
	}

	var result map[string]interface{}
	var err error

	switch operation {
	case "read":
		result, err = n.readSheet(ctx, baseURL, spreadsheetID, rangeStr, headers)
	case "write":
		values := input.NodeConfig["values"]
		valueInput, _ := input.NodeConfig["valueInputOption"].(string)
		result, err = n.writeSheet(ctx, baseURL, spreadsheetID, rangeStr, values, valueInput, headers)
	case "append":
		values := input.NodeConfig["values"]
		valueInput, _ := input.NodeConfig["valueInputOption"].(string)
		result, err = n.appendSheet(ctx, baseURL, spreadsheetID, rangeStr, values, valueInput, headers)
	case "clear":
		result, err = n.clearSheet(ctx, baseURL, spreadsheetID, rangeStr, headers)
	case "create":
		title, _ := input.NodeConfig["title"].(string)
		result, err = n.createSheet(ctx, baseURL, title, headers)
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}

	if err != nil {
		return &runtime.ExecutionOutput{Error: err}, nil
	}

	return &runtime.ExecutionOutput{Data: result}, nil
}

func (n *GoogleSheetsNode) readSheet(ctx context.Context, baseURL, spreadsheetID, rangeStr string, headers map[string]string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/%s/values/%s", baseURL, spreadsheetID, rangeStr)
	return n.doRequest(ctx, "GET", url, nil, headers)
}

func (n *GoogleSheetsNode) writeSheet(ctx context.Context, baseURL, spreadsheetID, rangeStr string, values interface{}, valueInput string, headers map[string]string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/%s/values/%s?valueInputOption=%s", baseURL, spreadsheetID, rangeStr, valueInput)
	body := map[string]interface{}{"values": values}
	return n.doRequest(ctx, "PUT", url, body, headers)
}

func (n *GoogleSheetsNode) appendSheet(ctx context.Context, baseURL, spreadsheetID, rangeStr string, values interface{}, valueInput string, headers map[string]string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/%s/values/%s:append?valueInputOption=%s&insertDataOption=INSERT_ROWS", baseURL, spreadsheetID, rangeStr, valueInput)
	body := map[string]interface{}{"values": values}
	return n.doRequest(ctx, "POST", url, body, headers)
}

func (n *GoogleSheetsNode) clearSheet(ctx context.Context, baseURL, spreadsheetID, rangeStr string, headers map[string]string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/%s/values/%s:clear", baseURL, spreadsheetID, rangeStr)
	return n.doRequest(ctx, "POST", url, map[string]interface{}{}, headers)
}

func (n *GoogleSheetsNode) createSheet(ctx context.Context, baseURL, title string, headers map[string]string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"properties": map[string]interface{}{
			"title": title,
		},
	}
	return n.doRequest(ctx, "POST", baseURL, body, headers)
}

func (n *GoogleSheetsNode) doRequest(ctx context.Context, method, url string, body interface{}, headers map[string]string) (map[string]interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		bodyReader = strings.NewReader(string(bodyBytes))
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed: %s", string(data))
	}

	var result map[string]interface{}
	json.Unmarshal(data, &result)
	return result, nil
}
