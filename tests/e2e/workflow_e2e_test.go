package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	baseURL = "http://localhost:8000" // API Gateway URL
	timeout = 30 * time.Second
)

type E2ETestContext struct {
	t      *testing.T
	client *http.Client
	token  string
	userID string
	orgID  string
}

func TestE2EWorkflowJourney(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	ctx := &E2ETestContext{
		t: t,
		client: &http.Client{
			Timeout: timeout,
		},
	}

	// Step 1: Register user
	t.Run("RegisterUser", ctx.testRegisterUser)

	// Step 2: Login
	t.Run("Login", ctx.testLogin)

	// Step 3: Create workflow
	t.Run("CreateWorkflow", ctx.testCreateWorkflow)

	// Step 4: Add nodes to workflow
	t.Run("AddNodes", ctx.testAddNodes)

	// Step 5: Activate workflow
	t.Run("ActivateWorkflow", ctx.testActivateWorkflow)

	// Step 6: Execute workflow
	t.Run("ExecuteWorkflow", ctx.testExecuteWorkflow)

	// Step 7: Check execution status
	t.Run("CheckExecutionStatus", ctx.testCheckExecutionStatus)

	// Step 8: Schedule workflow
	t.Run("ScheduleWorkflow", ctx.testScheduleWorkflow)

	// Step 9: Create webhook
	t.Run("CreateWebhook", ctx.testCreateWebhook)

	// Step 10: Test notifications
	t.Run("TestNotifications", ctx.testNotifications)
}

func (ctx *E2ETestContext) testRegisterUser(t *testing.T) {
	payload := map[string]interface{}{
		"email":            "e2e-test@linkflow.ai",
		"password":         "SecurePassword123!",
		"firstName":        "E2E",
		"lastName":         "Test",
		"organizationName": "E2E Test Org",
	}

	resp := ctx.makeRequest("POST", "/api/v1/auth/register", payload, "")
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		t.Log("User already exists, skipping registration")
		return
	}

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	
	user := result["user"].(map[string]interface{})
	ctx.userID = user["id"].(string)
	ctx.orgID = user["organizationId"].(string)
}

func (ctx *E2ETestContext) testLogin(t *testing.T) {
	payload := map[string]interface{}{
		"email":    "e2e-test@linkflow.ai",
		"password": "SecurePassword123!",
	}

	resp := ctx.makeRequest("POST", "/api/v1/auth/login", payload, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	
	tokens := result["tokens"].(map[string]interface{})
	ctx.token = tokens["accessToken"].(string)
	assert.NotEmpty(t, ctx.token)
}

var workflowID string

func (ctx *E2ETestContext) testCreateWorkflow(t *testing.T) {
	payload := map[string]interface{}{
		"name":        "E2E Test Workflow",
		"description": "Automated E2E test workflow",
		"tags":        []string{"e2e", "test", "automated"},
	}

	resp := ctx.makeRequest("POST", "/api/v1/workflows", payload, ctx.token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	
	workflowID = result["id"].(string)
	assert.NotEmpty(t, workflowID)
	assert.Equal(t, "draft", result["status"])
}

func (ctx *E2ETestContext) testAddNodes(t *testing.T) {
	payload := map[string]interface{}{
		"nodes": []map[string]interface{}{
			{
				"id":   "trigger-1",
				"type": "trigger",
				"name": "HTTP Trigger",
				"config": map[string]interface{}{
					"method": "POST",
					"path":   "/webhook/start",
				},
			},
			{
				"id":   "http-1",
				"type": "action",
				"name": "Call External API",
				"config": map[string]interface{}{
					"url":    "https://api.example.com/process",
					"method": "POST",
					"headers": map[string]string{
						"Content-Type": "application/json",
					},
				},
			},
			{
				"id":   "condition-1",
				"type": "condition",
				"name": "Check Response",
				"config": map[string]interface{}{
					"expression": "response.status == 200",
				},
			},
			{
				"id":   "notify-1",
				"type": "action",
				"name": "Send Notification",
				"config": map[string]interface{}{
					"channel": "email",
					"to":      "admin@linkflow.ai",
					"subject": "Workflow Completed",
				},
			},
		},
		"connections": []map[string]interface{}{
			{
				"id":           "conn-1",
				"sourceNodeId": "trigger-1",
				"targetNodeId": "http-1",
			},
			{
				"id":           "conn-2",
				"sourceNodeId": "http-1",
				"targetNodeId": "condition-1",
			},
			{
				"id":           "conn-3",
				"sourceNodeId": "condition-1",
				"targetNodeId": "notify-1",
			},
		},
	}

	resp := ctx.makeRequest("PUT", fmt.Sprintf("/api/v1/workflows/%s", workflowID), payload, ctx.token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	
	nodes := result["nodes"].([]interface{})
	assert.Len(t, nodes, 4)
}

func (ctx *E2ETestContext) testActivateWorkflow(t *testing.T) {
	resp := ctx.makeRequest("POST", fmt.Sprintf("/api/v1/workflows/%s/activate", workflowID), nil, ctx.token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "active", result["status"])
}

var executionID string

func (ctx *E2ETestContext) testExecuteWorkflow(t *testing.T) {
	payload := map[string]interface{}{
		"input": map[string]interface{}{
			"data": "test data",
			"timestamp": time.Now().Unix(),
		},
		"async": true,
	}

	resp := ctx.makeRequest("POST", fmt.Sprintf("/api/v1/workflows/%s/execute", workflowID), payload, ctx.token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	
	executionID = result["executionId"].(string)
	assert.NotEmpty(t, executionID)
}

func (ctx *E2ETestContext) testCheckExecutionStatus(t *testing.T) {
	// Wait for execution to complete
	time.Sleep(2 * time.Second)

	resp := ctx.makeRequest("GET", fmt.Sprintf("/api/v1/executions/%s", executionID), nil, ctx.token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	
	status := result["status"].(string)
	assert.Contains(t, []string{"completed", "running", "failed"}, status)
}

func (ctx *E2ETestContext) testScheduleWorkflow(t *testing.T) {
	payload := map[string]interface{}{
		"workflowId":  workflowID,
		"name":        "Daily E2E Test",
		"cronExpression": "0 0 * * *", // Daily at midnight
		"timezone":    "UTC",
		"enabled":     true,
	}

	resp := ctx.makeRequest("POST", "/api/v1/schedules", payload, ctx.token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.NotEmpty(t, result["id"])
	assert.Equal(t, "active", result["status"])
}

func (ctx *E2ETestContext) testCreateWebhook(t *testing.T) {
	payload := map[string]interface{}{
		"name":        "E2E Webhook",
		"endpointUrl": "https://example.com/webhook",
		"events":      []string{"workflow.completed", "workflow.failed"},
		"secret":      "webhook-secret-123",
		"enabled":     true,
	}

	resp := ctx.makeRequest("POST", "/api/v1/webhooks", payload, ctx.token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.NotEmpty(t, result["id"])
	assert.NotEmpty(t, result["signature"])
}

func (ctx *E2ETestContext) testNotifications(t *testing.T) {
	payload := map[string]interface{}{
		"title":    "E2E Test Notification",
		"message":  "This is an automated E2E test notification",
		"channels": []string{"in_app", "email"},
		"priority": "high",
		"metadata": map[string]interface{}{
			"workflowId":  workflowID,
			"executionId": executionID,
		},
	}

	resp := ctx.makeRequest("POST", "/api/v1/notifications", payload, ctx.token)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.NotEmpty(t, result["id"])
	assert.Equal(t, "pending", result["status"])
}

// Helper method
func (ctx *E2ETestContext) makeRequest(method, path string, payload interface{}, token string) *http.Response {
	var body []byte
	if payload != nil {
		body, _ = json.Marshal(payload)
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		method,
		baseURL+path,
		bytes.NewBuffer(body),
	)
	require.NoError(ctx.t, err)

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := ctx.client.Do(req)
	require.NoError(ctx.t, err)

	return resp
}
