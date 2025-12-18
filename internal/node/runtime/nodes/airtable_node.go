// Package nodes provides Airtable node implementation
package nodes

import (
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

func init() {
	runtime.Register(&AirtableNode{})
}

// AirtableNode implements Airtable operations
type AirtableNode struct {
	client *http.Client
}

func (n *AirtableNode) GetType() string { return "airtable" }
func (n *AirtableNode) Validate(config map[string]interface{}) error { return nil }

func (n *AirtableNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "airtable",
		Name:        "Airtable",
		Description: "Interact with Airtable bases and records",
		Category:    "integration",
		Version:     "1.0.0",
		Icon:        "airtable",
		Inputs:      []runtime.PortDefinition{{Name: "main", Type: "main"}},
		Outputs:     []runtime.PortDefinition{{Name: "main", Type: "main"}},
		Properties: []runtime.PropertyDefinition{
			{Name: "operation", Type: "select", Required: true, Options: []runtime.PropertyOption{
				{Label: "List Bases", Value: "listBases"}, {Label: "List Records", Value: "listRecords"},
				{Label: "Get Record", Value: "getRecord"}, {Label: "Create Record", Value: "createRecord"},
				{Label: "Update Record", Value: "updateRecord"}, {Label: "Delete Record", Value: "deleteRecord"},
			}},
			{Name: "baseId", Type: "string"},
			{Name: "tableId", Type: "string"},
			{Name: "recordId", Type: "string"},
			{Name: "fields", Type: "json"},
			{Name: "filterByFormula", Type: "string"},
			{Name: "maxRecords", Type: "number"},
		},
	}
}

func (n *AirtableNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	if n.client == nil {
		n.client = &http.Client{Timeout: 30 * time.Second}
	}

	operation, _ := input.NodeConfig["operation"].(string)
	accessToken, _ := input.Credentials["access_token"].(string)
	if accessToken == "" {
		accessToken, _ = input.Credentials["token"].(string)
	}

	if accessToken == "" {
		return nil, fmt.Errorf("access_token or token required")
	}

	baseID, _ := input.NodeConfig["baseId"].(string)
	tableID, _ := input.NodeConfig["tableId"].(string)
	baseURL := "https://api.airtable.com/v0"
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
		"Content-Type":  "application/json",
	}

	var result map[string]interface{}
	var err error

	switch operation {
	case "listBases":
		result, err = n.doRequest(ctx, "GET", baseURL+"/meta/bases", nil, headers)
	case "listTables":
		result, err = n.doRequest(ctx, "GET", fmt.Sprintf("%s/meta/bases/%s/tables", baseURL, baseID), nil, headers)

	case "listRecords":
		urlStr := fmt.Sprintf("%s/%s/%s", baseURL, baseID, tableID)
		params := url.Values{}
		if filter, ok := input.NodeConfig["filterByFormula"].(string); ok && filter != "" {
			params.Set("filterByFormula", filter)
		}
		if maxRecords, ok := input.NodeConfig["maxRecords"].(float64); ok {
			params.Set("maxRecords", fmt.Sprintf("%d", int(maxRecords)))
		}
		if view, ok := input.NodeConfig["view"].(string); ok && view != "" {
			params.Set("view", view)
		}
		if len(params) > 0 {
			urlStr += "?" + params.Encode()
		}
		result, err = n.doRequest(ctx, "GET", urlStr, nil, headers)

	case "getRecord":
		recordID, _ := input.NodeConfig["recordId"].(string)
		result, err = n.doRequest(ctx, "GET", fmt.Sprintf("%s/%s/%s/%s", baseURL, baseID, tableID, recordID), nil, headers)

	case "createRecord":
		fields, _ := input.NodeConfig["fields"].(map[string]interface{})
		body := map[string]interface{}{
			"fields": fields,
		}
		result, err = n.doRequest(ctx, "POST", fmt.Sprintf("%s/%s/%s", baseURL, baseID, tableID), body, headers)

	case "createRecords":
		records, _ := input.NodeConfig["records"].([]interface{})
		body := map[string]interface{}{
			"records": records,
		}
		result, err = n.doRequest(ctx, "POST", fmt.Sprintf("%s/%s/%s", baseURL, baseID, tableID), body, headers)

	case "updateRecord":
		recordID, _ := input.NodeConfig["recordId"].(string)
		fields, _ := input.NodeConfig["fields"].(map[string]interface{})
		body := map[string]interface{}{
			"fields": fields,
		}
		result, err = n.doRequest(ctx, "PATCH", fmt.Sprintf("%s/%s/%s/%s", baseURL, baseID, tableID, recordID), body, headers)

	case "updateRecords":
		records, _ := input.NodeConfig["records"].([]interface{})
		body := map[string]interface{}{
			"records": records,
		}
		result, err = n.doRequest(ctx, "PATCH", fmt.Sprintf("%s/%s/%s", baseURL, baseID, tableID), body, headers)

	case "deleteRecord":
		recordID, _ := input.NodeConfig["recordId"].(string)
		result, err = n.doRequest(ctx, "DELETE", fmt.Sprintf("%s/%s/%s/%s", baseURL, baseID, tableID, recordID), nil, headers)

	case "deleteRecords":
		recordIDs, _ := input.NodeConfig["recordIds"].([]interface{})
		params := url.Values{}
		for _, id := range recordIDs {
			if idStr, ok := id.(string); ok {
				params.Add("records[]", idStr)
			}
		}
		urlStr := fmt.Sprintf("%s/%s/%s?%s", baseURL, baseID, tableID, params.Encode())
		result, err = n.doRequest(ctx, "DELETE", urlStr, nil, headers)

	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}

	if err != nil {
		return &runtime.ExecutionOutput{Error: err}, nil
	}

	return &runtime.ExecutionOutput{Data: result}, nil
}

func (n *AirtableNode) doRequest(ctx context.Context, method, urlStr string, body interface{}, headers map[string]string) (map[string]interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		bodyReader = strings.NewReader(string(bodyBytes))
	}

	req, err := http.NewRequestWithContext(ctx, method, urlStr, bodyReader)
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
