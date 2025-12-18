package helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestServer wraps httptest.Server with additional helper methods
type TestServer struct {
	*httptest.Server
	t      *testing.T
	token  string
	userID string
}

// NewTestServer creates a new test server with the given handler
func NewTestServer(t *testing.T, handler http.Handler) *TestServer {
	return &TestServer{
		Server: httptest.NewServer(handler),
		t:      t,
	}
}

// SetAuthToken sets the authentication token for requests
func (s *TestServer) SetAuthToken(token string) {
	s.token = token
}

// SetUserID sets the user ID for requests
func (s *TestServer) SetUserID(userID string) {
	s.userID = userID
}

// Request makes an HTTP request to the test server
func (s *TestServer) Request(method, path string, body interface{}) *TestResponse {
	var bodyReader *bytes.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		require.NoError(s.t, err)
		bodyReader = bytes.NewReader(bodyBytes)
	}

	var req *http.Request
	var err error
	if bodyReader != nil {
		req, err = http.NewRequest(method, s.URL+path, bodyReader)
	} else {
		req, err = http.NewRequest(method, s.URL+path, nil)
	}
	require.NoError(s.t, err)

	req.Header.Set("Content-Type", "application/json")
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}
	if s.userID != "" {
		req.Header.Set("X-User-ID", s.userID)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	require.NoError(s.t, err)

	return &TestResponse{
		Response: resp,
		t:        s.t,
	}
}

// GET makes a GET request
func (s *TestServer) GET(path string) *TestResponse {
	return s.Request("GET", path, nil)
}

// POST makes a POST request
func (s *TestServer) POST(path string, body interface{}) *TestResponse {
	return s.Request("POST", path, body)
}

// PUT makes a PUT request
func (s *TestServer) PUT(path string, body interface{}) *TestResponse {
	return s.Request("PUT", path, body)
}

// DELETE makes a DELETE request
func (s *TestServer) DELETE(path string) *TestResponse {
	return s.Request("DELETE", path, nil)
}

// PATCH makes a PATCH request
func (s *TestServer) PATCH(path string, body interface{}) *TestResponse {
	return s.Request("PATCH", path, body)
}

// TestResponse wraps http.Response with assertion helpers
type TestResponse struct {
	*http.Response
	t *testing.T
}

// ExpectStatus asserts the response status code
func (r *TestResponse) ExpectStatus(code int) *TestResponse {
	require.Equal(r.t, code, r.StatusCode, "unexpected status code")
	return r
}

// ExpectOK asserts the response status is 200 OK
func (r *TestResponse) ExpectOK() *TestResponse {
	return r.ExpectStatus(http.StatusOK)
}

// ExpectCreated asserts the response status is 201 Created
func (r *TestResponse) ExpectCreated() *TestResponse {
	return r.ExpectStatus(http.StatusCreated)
}

// ExpectNoContent asserts the response status is 204 No Content
func (r *TestResponse) ExpectNoContent() *TestResponse {
	return r.ExpectStatus(http.StatusNoContent)
}

// ExpectBadRequest asserts the response status is 400 Bad Request
func (r *TestResponse) ExpectBadRequest() *TestResponse {
	return r.ExpectStatus(http.StatusBadRequest)
}

// ExpectUnauthorized asserts the response status is 401 Unauthorized
func (r *TestResponse) ExpectUnauthorized() *TestResponse {
	return r.ExpectStatus(http.StatusUnauthorized)
}

// ExpectForbidden asserts the response status is 403 Forbidden
func (r *TestResponse) ExpectForbidden() *TestResponse {
	return r.ExpectStatus(http.StatusForbidden)
}

// ExpectNotFound asserts the response status is 404 Not Found
func (r *TestResponse) ExpectNotFound() *TestResponse {
	return r.ExpectStatus(http.StatusNotFound)
}

// JSON decodes the response body into the provided interface
func (r *TestResponse) JSON(v interface{}) *TestResponse {
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(v)
	require.NoError(r.t, err)
	return r
}

// ExpectJSON asserts the response body matches the expected JSON
func (r *TestResponse) ExpectJSON(expected interface{}) *TestResponse {
	defer r.Body.Close()
	
	expectedBytes, err := json.Marshal(expected)
	require.NoError(r.t, err)
	
	var actual interface{}
	err = json.NewDecoder(r.Body).Decode(&actual)
	require.NoError(r.t, err)
	
	actualBytes, err := json.Marshal(actual)
	require.NoError(r.t, err)
	
	require.JSONEq(r.t, string(expectedBytes), string(actualBytes))
	return r
}

// ExpectHeader asserts a response header value
func (r *TestResponse) ExpectHeader(key, value string) *TestResponse {
	require.Equal(r.t, value, r.Header.Get(key))
	return r
}

// ExpectHeaderExists asserts a response header exists
func (r *TestResponse) ExpectHeaderExists(key string) *TestResponse {
	require.NotEmpty(r.t, r.Header.Get(key))
	return r
}

// MockService provides a mock HTTP service for testing
type MockService struct {
	server   *httptest.Server
	handlers map[string]http.HandlerFunc
}

// NewMockService creates a new mock service
func NewMockService() *MockService {
	m := &MockService{
		handlers: make(map[string]http.HandlerFunc),
	}
	
	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := fmt.Sprintf("%s:%s", r.Method, r.URL.Path)
		if handler, ok := m.handlers[key]; ok {
			handler(w, r)
			return
		}
		http.NotFound(w, r)
	}))
	
	return m
}

// URL returns the mock service URL
func (m *MockService) URL() string {
	return m.server.URL
}

// Close shuts down the mock service
func (m *MockService) Close() {
	m.server.Close()
}

// On registers a handler for a method/path combination
func (m *MockService) On(method, path string, handler http.HandlerFunc) *MockService {
	m.handlers[fmt.Sprintf("%s:%s", method, path)] = handler
	return m
}

// OnJSON registers a handler that returns JSON
func (m *MockService) OnJSON(method, path string, status int, response interface{}) *MockService {
	return m.On(method, path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(response)
	})
}

// TestContext provides a context with test utilities
type TestContext struct {
	context.Context
	t       *testing.T
	cleanup []func()
}

// NewTestContext creates a new test context
func NewTestContext(t *testing.T) *TestContext {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	tc := &TestContext{
		Context: ctx,
		t:       t,
	}
	tc.cleanup = append(tc.cleanup, cancel)
	return tc
}

// Cleanup registers a cleanup function
func (tc *TestContext) Cleanup(fn func()) {
	tc.cleanup = append(tc.cleanup, fn)
}

// Done runs all cleanup functions
func (tc *TestContext) Done() {
	for i := len(tc.cleanup) - 1; i >= 0; i-- {
		tc.cleanup[i]()
	}
}

// Fixtures provides test data generation
type Fixtures struct{}

// NewFixtures creates a new fixtures helper
func NewFixtures() *Fixtures {
	return &Fixtures{}
}

// Workflow returns a test workflow
func (f *Fixtures) Workflow() map[string]interface{} {
	return map[string]interface{}{
		"name":        "Test Workflow",
		"description": "A workflow for testing",
		"nodes": []map[string]interface{}{
			{
				"id":   "trigger-1",
				"type": "trigger",
				"name": "Manual Trigger",
				"config": map[string]interface{}{
					"triggerType": "manual",
				},
			},
			{
				"id":   "action-1",
				"type": "action",
				"name": "HTTP Request",
				"config": map[string]interface{}{
					"method": "POST",
					"url":    "https://api.example.com/webhook",
				},
			},
		},
		"connections": []map[string]interface{}{
			{
				"id":           "conn-1",
				"sourceNodeId": "trigger-1",
				"targetNodeId": "action-1",
			},
		},
	}
}

// User returns a test user
func (f *Fixtures) User() map[string]interface{} {
	return map[string]interface{}{
		"email":     fmt.Sprintf("test-%d@example.com", time.Now().UnixNano()),
		"password":  "SecurePassword123!",
		"firstName": "Test",
		"lastName":  "User",
	}
}

// Schedule returns a test schedule
func (f *Fixtures) Schedule() map[string]interface{} {
	return map[string]interface{}{
		"name":           "Test Schedule",
		"cronExpression": "0 0 * * *",
		"timezone":       "UTC",
		"enabled":        true,
	}
}

// Webhook returns a test webhook
func (f *Fixtures) Webhook() map[string]interface{} {
	return map[string]interface{}{
		"name":   "Test Webhook",
		"url":    "https://example.com/webhook",
		"secret": "test-secret-key",
		"events": []string{"workflow.completed", "execution.failed"},
	}
}
