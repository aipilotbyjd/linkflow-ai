// Package nodes provides Gmail node implementation
package nodes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/node/runtime"
)

func init() {
	runtime.Register(&GmailNode{})
}

// GmailNode implements Gmail operations
type GmailNode struct {
	client *http.Client
}

func (n *GmailNode) GetType() string { return "gmail" }
func (n *GmailNode) Validate(config map[string]interface{}) error { return nil }

func (n *GmailNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "gmail",
		Name:        "Gmail",
		Description: "Send and read emails via Gmail API",
		Category:    "integration",
		Version:     "1.0.0",
		Icon:        "gmail",
		Inputs:      []runtime.PortDefinition{{Name: "main", Type: "main"}},
		Outputs:     []runtime.PortDefinition{{Name: "main", Type: "main"}},
		Properties: []runtime.PropertyDefinition{
			{Name: "operation", Type: "select", Required: true, Options: []runtime.PropertyOption{
				{Label: "Send", Value: "send"}, {Label: "Reply", Value: "reply"}, {Label: "List", Value: "list"},
				{Label: "Get", Value: "get"}, {Label: "Delete", Value: "delete"},
			}},
			{Name: "to", Type: "string"},
			{Name: "subject", Type: "string"},
			{Name: "body", Type: "string"},
			{Name: "bodyType", Type: "select", Default: "text", Options: []runtime.PropertyOption{{Label: "Text", Value: "text"}, {Label: "HTML", Value: "html"}}},
			{Name: "messageId", Type: "string"},
			{Name: "maxResults", Type: "number", Default: 10},
			{Name: "query", Type: "string"},
		},
	}
}

func (n *GmailNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	if n.client == nil {
		n.client = &http.Client{Timeout: 30 * time.Second}
	}

	operation, _ := input.NodeConfig["operation"].(string)
	accessToken, _ := input.Credentials["access_token"].(string)

	if accessToken == "" {
		return nil, fmt.Errorf("access_token required")
	}

	baseURL := "https://gmail.googleapis.com/gmail/v1/users/me"
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
		"Content-Type":  "application/json",
	}

	var result map[string]interface{}
	var err error

	switch operation {
	case "send":
		result, err = n.sendEmail(ctx, baseURL, input.NodeConfig, headers)
	case "reply":
		result, err = n.replyEmail(ctx, baseURL, input.NodeConfig, headers)
	case "list":
		result, err = n.listEmails(ctx, baseURL, input.NodeConfig, headers)
	case "get":
		messageID, _ := input.NodeConfig["messageId"].(string)
		result, err = n.getEmail(ctx, baseURL, messageID, headers)
	case "delete":
		messageID, _ := input.NodeConfig["messageId"].(string)
		result, err = n.deleteEmail(ctx, baseURL, messageID, headers)
	case "addLabel":
		messageID, _ := input.NodeConfig["messageId"].(string)
		labelIDs, _ := input.NodeConfig["labelIds"].([]interface{})
		result, err = n.modifyLabels(ctx, baseURL, messageID, labelIDs, nil, headers)
	case "removeLabel":
		messageID, _ := input.NodeConfig["messageId"].(string)
		labelIDs, _ := input.NodeConfig["labelIds"].([]interface{})
		result, err = n.modifyLabels(ctx, baseURL, messageID, nil, labelIDs, headers)
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}

	if err != nil {
		return &runtime.ExecutionOutput{Error: err}, nil
	}

	return &runtime.ExecutionOutput{Data: result}, nil
}

func (n *GmailNode) sendEmail(ctx context.Context, baseURL string, config map[string]interface{}, headers map[string]string) (map[string]interface{}, error) {
	to, _ := config["to"].(string)
	cc, _ := config["cc"].(string)
	bcc, _ := config["bcc"].(string)
	subject, _ := config["subject"].(string)
	body, _ := config["body"].(string)
	bodyType, _ := config["bodyType"].(string)

	contentType := "text/plain"
	if bodyType == "html" {
		contentType = "text/html"
	}

	// Build raw email
	var emailParts []string
	emailParts = append(emailParts, fmt.Sprintf("To: %s", to))
	if cc != "" {
		emailParts = append(emailParts, fmt.Sprintf("Cc: %s", cc))
	}
	if bcc != "" {
		emailParts = append(emailParts, fmt.Sprintf("Bcc: %s", bcc))
	}
	emailParts = append(emailParts, fmt.Sprintf("Subject: %s", subject))
	emailParts = append(emailParts, fmt.Sprintf("Content-Type: %s; charset=UTF-8", contentType))
	emailParts = append(emailParts, "")
	emailParts = append(emailParts, body)

	rawEmail := strings.Join(emailParts, "\r\n")
	encodedEmail := base64.URLEncoding.EncodeToString([]byte(rawEmail))

	reqBody := map[string]interface{}{
		"raw": encodedEmail,
	}

	return n.doRequest(ctx, "POST", baseURL+"/messages/send", reqBody, headers)
}

func (n *GmailNode) replyEmail(ctx context.Context, baseURL string, config map[string]interface{}, headers map[string]string) (map[string]interface{}, error) {
	messageID, _ := config["messageId"].(string)
	
	// Get original message first
	original, err := n.getEmail(ctx, baseURL, messageID, headers)
	if err != nil {
		return nil, err
	}

	threadID, _ := original["threadId"].(string)
	
	to, _ := config["to"].(string)
	subject, _ := config["subject"].(string)
	body, _ := config["body"].(string)
	bodyType, _ := config["bodyType"].(string)

	contentType := "text/plain"
	if bodyType == "html" {
		contentType = "text/html"
	}

	var emailParts []string
	emailParts = append(emailParts, fmt.Sprintf("To: %s", to))
	emailParts = append(emailParts, fmt.Sprintf("Subject: Re: %s", subject))
	emailParts = append(emailParts, fmt.Sprintf("In-Reply-To: %s", messageID))
	emailParts = append(emailParts, fmt.Sprintf("References: %s", messageID))
	emailParts = append(emailParts, fmt.Sprintf("Content-Type: %s; charset=UTF-8", contentType))
	emailParts = append(emailParts, "")
	emailParts = append(emailParts, body)

	rawEmail := strings.Join(emailParts, "\r\n")
	encodedEmail := base64.URLEncoding.EncodeToString([]byte(rawEmail))

	reqBody := map[string]interface{}{
		"raw":      encodedEmail,
		"threadId": threadID,
	}

	return n.doRequest(ctx, "POST", baseURL+"/messages/send", reqBody, headers)
}

func (n *GmailNode) listEmails(ctx context.Context, baseURL string, config map[string]interface{}, headers map[string]string) (map[string]interface{}, error) {
	maxResults := 10
	if mr, ok := config["maxResults"].(float64); ok {
		maxResults = int(mr)
	}
	query, _ := config["query"].(string)

	url := fmt.Sprintf("%s/messages?maxResults=%d", baseURL, maxResults)
	if query != "" {
		url += "&q=" + query
	}

	return n.doRequest(ctx, "GET", url, nil, headers)
}

func (n *GmailNode) getEmail(ctx context.Context, baseURL, messageID string, headers map[string]string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/messages/%s?format=full", baseURL, messageID)
	return n.doRequest(ctx, "GET", url, nil, headers)
}

func (n *GmailNode) deleteEmail(ctx context.Context, baseURL, messageID string, headers map[string]string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/messages/%s/trash", baseURL, messageID)
	return n.doRequest(ctx, "POST", url, nil, headers)
}

func (n *GmailNode) modifyLabels(ctx context.Context, baseURL, messageID string, addLabels, removeLabels []interface{}, headers map[string]string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/messages/%s/modify", baseURL, messageID)
	body := map[string]interface{}{}
	if len(addLabels) > 0 {
		body["addLabelIds"] = addLabels
	}
	if len(removeLabels) > 0 {
		body["removeLabelIds"] = removeLabels
	}
	return n.doRequest(ctx, "POST", url, body, headers)
}

func (n *GmailNode) doRequest(ctx context.Context, method, url string, body interface{}, headers map[string]string) (map[string]interface{}, error) {
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
