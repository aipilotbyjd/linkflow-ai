package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/linkflow-ai/linkflow-ai/internal/platform/config"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/database"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/linkflow-ai/linkflow-ai/internal/workflow/server"
)

type WorkflowIntegrationTestSuite struct {
	suite.Suite
	server   *httptest.Server
	db       *database.DB
	token    string
	userID   string
	orgID    string
}

func (suite *WorkflowIntegrationTestSuite) SetupSuite() {
	// Load test configuration
	cfg, err := config.Load("workflow")
	require.NoError(suite.T(), err)

	// Initialize test database
	db, err := database.New(cfg.Database)
	require.NoError(suite.T(), err)
	suite.db = db

	// Run migrations
	err = suite.runMigrations()
	require.NoError(suite.T(), err)

	// Initialize test server
	log := logger.New(cfg.Logger)
	srv, err := server.New(
		server.WithConfig(cfg),
		server.WithLogger(log),
	)
	require.NoError(suite.T(), err)

	// Start test server
	suite.server = httptest.NewServer(srv.Handler())

	// Create test user and authenticate
	suite.createTestUser()
	suite.authenticate()
}

func (suite *WorkflowIntegrationTestSuite) TearDownSuite() {
	suite.server.Close()
	suite.db.Close()
}

func (suite *WorkflowIntegrationTestSuite) TestCreateWorkflow() {
	tests := []struct {
		name         string
		payload      map[string]interface{}
		expectedCode int
	}{
		{
			name: "valid workflow",
			payload: map[string]interface{}{
				"name":        "Test Workflow",
				"description": "Test Description",
				"nodes": []map[string]interface{}{
					{
						"id":   "node-1",
						"type": "trigger",
						"name": "Start",
						"position": map[string]float64{
							"x": 100,
							"y": 100,
						},
					},
					{
						"id":   "node-2",
						"type": "action",
						"name": "HTTP Request",
						"position": map[string]float64{
							"x": 300,
							"y": 100,
						},
					},
				},
				"connections": []map[string]interface{}{
					{
						"id":           "conn-1",
						"sourceNodeId": "node-1",
						"targetNodeId": "node-2",
					},
				},
			},
			expectedCode: http.StatusCreated,
		},
		{
			name: "workflow without nodes",
			payload: map[string]interface{}{
				"name":        "Empty Workflow",
				"description": "No nodes",
			},
			expectedCode: http.StatusCreated,
		},
		{
			name:         "workflow without name",
			payload:      map[string]interface{}{},
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			body, _ := json.Marshal(tc.payload)
			req, err := http.NewRequest("POST", suite.server.URL+"/api/v1/workflows", bytes.NewBuffer(body))
			require.NoError(suite.T(), err)

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+suite.token)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(suite.T(), err)
			defer resp.Body.Close()

			assert.Equal(suite.T(), tc.expectedCode, resp.StatusCode)

			if tc.expectedCode == http.StatusCreated {
				var result map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(suite.T(), err)

				assert.NotEmpty(suite.T(), result["id"])
				assert.Equal(suite.T(), tc.payload["name"], result["name"])
				assert.Equal(suite.T(), "draft", result["status"])
			}
		})
	}
}

func (suite *WorkflowIntegrationTestSuite) TestListWorkflows() {
	// Create test workflows
	for i := 0; i < 5; i++ {
		suite.createTestWorkflow(fmt.Sprintf("Workflow %d", i+1))
	}

	// Test listing
	req, err := http.NewRequest("GET", suite.server.URL+"/api/v1/workflows", nil)
	require.NoError(suite.T(), err)
	req.Header.Set("Authorization", "Bearer "+suite.token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(suite.T(), err)

	workflows := result["workflows"].([]interface{})
	assert.GreaterOrEqual(suite.T(), len(workflows), 5)
}

func (suite *WorkflowIntegrationTestSuite) TestGetWorkflow() {
	// Create test workflow
	workflowID := suite.createTestWorkflow("Get Test Workflow")

	// Get workflow
	req, err := http.NewRequest("GET", suite.server.URL+"/api/v1/workflows/"+workflowID, nil)
	require.NoError(suite.T(), err)
	req.Header.Set("Authorization", "Bearer "+suite.token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), workflowID, result["id"])
	assert.Equal(suite.T(), "Get Test Workflow", result["name"])
}

func (suite *WorkflowIntegrationTestSuite) TestUpdateWorkflow() {
	// Create test workflow
	workflowID := suite.createTestWorkflow("Update Test Workflow")

	// Update workflow
	payload := map[string]interface{}{
		"name":        "Updated Workflow",
		"description": "Updated Description",
		"tags":        []string{"test", "updated"},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("PUT", suite.server.URL+"/api/v1/workflows/"+workflowID, bytes.NewBuffer(body))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "Updated Workflow", result["name"])
	assert.Equal(suite.T(), "Updated Description", result["description"])
}

func (suite *WorkflowIntegrationTestSuite) TestDeleteWorkflow() {
	// Create test workflow
	workflowID := suite.createTestWorkflow("Delete Test Workflow")

	// Delete workflow
	req, err := http.NewRequest("DELETE", suite.server.URL+"/api/v1/workflows/"+workflowID, nil)
	require.NoError(suite.T(), err)
	req.Header.Set("Authorization", "Bearer "+suite.token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusNoContent, resp.StatusCode)

	// Verify workflow is deleted
	req, err = http.NewRequest("GET", suite.server.URL+"/api/v1/workflows/"+workflowID, nil)
	require.NoError(suite.T(), err)
	req.Header.Set("Authorization", "Bearer "+suite.token)

	resp, err = http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)
}

func (suite *WorkflowIntegrationTestSuite) TestActivateWorkflow() {
	// Create test workflow with trigger node
	payload := map[string]interface{}{
		"name":        "Activate Test Workflow",
		"description": "Test activation",
		"nodes": []map[string]interface{}{
			{
				"id":   "node-1",
				"type": "trigger",
				"name": "Start",
			},
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", suite.server.URL+"/api/v1/workflows", bytes.NewBuffer(body))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	var workflow map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&workflow)
	workflowID := workflow["id"].(string)

	// Activate workflow
	req, err = http.NewRequest("POST", suite.server.URL+"/api/v1/workflows/"+workflowID+"/activate", nil)
	require.NoError(suite.T(), err)
	req.Header.Set("Authorization", "Bearer "+suite.token)

	resp, err = http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(suite.T(), "active", result["status"])
}

func (suite *WorkflowIntegrationTestSuite) TestExecuteWorkflow() {
	// Create and activate workflow
	workflowID := suite.createAndActivateTestWorkflow()

	// Execute workflow
	payload := map[string]interface{}{
		"input": map[string]interface{}{
			"message": "Hello World",
		},
		"async": false,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", suite.server.URL+"/api/v1/workflows/"+workflowID+"/execute", bytes.NewBuffer(body))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.NotEmpty(suite.T(), result["executionId"])
}

// Helper methods
func (suite *WorkflowIntegrationTestSuite) runMigrations() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Run test migrations
	_, err := suite.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS workflows (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			status VARCHAR(50) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);
	`)
	return err
}

func (suite *WorkflowIntegrationTestSuite) createTestUser() {
	// Mock user creation
	suite.userID = "test-user-123"
	suite.orgID = "test-org-123"
}

func (suite *WorkflowIntegrationTestSuite) authenticate() {
	// Mock authentication
	suite.token = "test-jwt-token"
}

func (suite *WorkflowIntegrationTestSuite) createTestWorkflow(name string) string {
	// Mock workflow creation
	return fmt.Sprintf("workflow-%d", time.Now().UnixNano())
}

func (suite *WorkflowIntegrationTestSuite) createAndActivateTestWorkflow() string {
	// Mock workflow creation and activation
	return fmt.Sprintf("active-workflow-%d", time.Now().UnixNano())
}

func TestWorkflowIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(WorkflowIntegrationTestSuite))
}
