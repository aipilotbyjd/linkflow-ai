// K6 Load Testing Script for LinkFlow AI
// Run with: k6 run tests/load/k6_load_test.js

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const workflowCreateTrend = new Trend('workflow_create_duration');
const workflowExecuteTrend = new Trend('workflow_execute_duration');
const authLoginTrend = new Trend('auth_login_duration');
const workflowsCreated = new Counter('workflows_created');
const executionsStarted = new Counter('executions_started');

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8000';
const AUTH_TOKEN = __ENV.AUTH_TOKEN || 'test-token';

// Test scenarios
export const options = {
  scenarios: {
    // Smoke test - basic functionality
    smoke: {
      executor: 'constant-vus',
      vus: 1,
      duration: '30s',
      gracefulStop: '5s',
      tags: { test_type: 'smoke' },
      exec: 'smokeTest',
    },
    
    // Load test - normal load
    load: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '2m', target: 50 },   // Ramp up to 50 users
        { duration: '5m', target: 50 },   // Stay at 50 users
        { duration: '2m', target: 100 },  // Ramp up to 100 users
        { duration: '5m', target: 100 },  // Stay at 100 users
        { duration: '2m', target: 0 },    // Ramp down
      ],
      gracefulStop: '30s',
      tags: { test_type: 'load' },
      exec: 'loadTest',
      startTime: '35s', // Start after smoke test
    },
    
    // Stress test - find breaking point
    stress: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '2m', target: 100 },
        { duration: '5m', target: 200 },
        { duration: '5m', target: 300 },
        { duration: '5m', target: 400 },
        { duration: '5m', target: 500 },
        { duration: '5m', target: 0 },
      ],
      gracefulStop: '60s',
      tags: { test_type: 'stress' },
      exec: 'stressTest',
      startTime: '17m',
    },
    
    // Spike test - sudden traffic spike
    spike: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 100 },  // Quick ramp up
        { duration: '1m', target: 100 },
        { duration: '10s', target: 500 },  // Spike!
        { duration: '1m', target: 500 },
        { duration: '10s', target: 100 },  // Quick ramp down
        { duration: '1m', target: 100 },
        { duration: '10s', target: 0 },
      ],
      gracefulStop: '30s',
      tags: { test_type: 'spike' },
      exec: 'spikeTest',
      startTime: '40m',
    },
  },
  
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'], // 95% < 500ms, 99% < 1s
    http_req_failed: ['rate<0.01'],                   // Error rate < 1%
    errors: ['rate<0.05'],                            // Custom error rate < 5%
    workflow_create_duration: ['p(95)<300'],
    workflow_execute_duration: ['p(95)<1000'],
    auth_login_duration: ['p(95)<200'],
  },
};

// Helper functions
function getHeaders(includeAuth = true) {
  const headers = {
    'Content-Type': 'application/json',
  };
  if (includeAuth) {
    headers['Authorization'] = `Bearer ${AUTH_TOKEN}`;
  }
  return headers;
}

function randomString(length) {
  const chars = 'abcdefghijklmnopqrstuvwxyz0123456789';
  let result = '';
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return result;
}

// Test: Health check
function healthCheck() {
  const res = http.get(`${BASE_URL}/health`);
  check(res, {
    'health check status is 200': (r) => r.status === 200,
    'health check response is healthy': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.status === 'healthy';
      } catch {
        return false;
      }
    },
  });
  return res.status === 200;
}

// Test: Authentication
function testLogin() {
  const payload = JSON.stringify({
    email: `test-${randomString(8)}@example.com`,
    password: 'TestPassword123!',
  });
  
  const start = new Date();
  const res = http.post(`${BASE_URL}/api/v1/auth/login`, payload, {
    headers: getHeaders(false),
  });
  authLoginTrend.add(new Date() - start);
  
  const success = check(res, {
    'login status is 200 or 201': (r) => r.status === 200 || r.status === 201,
    'login returns token': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.tokens && body.tokens.accessToken;
      } catch {
        return false;
      }
    },
  });
  
  errorRate.add(!success);
  return res;
}

// Test: Create Workflow
function testCreateWorkflow() {
  const payload = JSON.stringify({
    name: `Load Test Workflow ${randomString(8)}`,
    description: 'Workflow created during load testing',
    nodes: [
      {
        id: 'trigger-1',
        type: 'trigger',
        name: 'HTTP Trigger',
        config: { triggerType: 'http' },
        position: { x: 100, y: 100 },
      },
      {
        id: 'action-1',
        type: 'action',
        name: 'HTTP Request',
        config: { method: 'POST', url: 'https://httpbin.org/post' },
        position: { x: 300, y: 100 },
      },
    ],
    connections: [
      {
        id: 'conn-1',
        sourceNodeId: 'trigger-1',
        targetNodeId: 'action-1',
      },
    ],
    tags: ['load-test'],
  });
  
  const start = new Date();
  const res = http.post(`${BASE_URL}/api/v1/workflows`, payload, {
    headers: getHeaders(),
  });
  workflowCreateTrend.add(new Date() - start);
  
  const success = check(res, {
    'create workflow status is 201': (r) => r.status === 201,
    'create workflow returns id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.id !== undefined;
      } catch {
        return false;
      }
    },
  });
  
  errorRate.add(!success);
  if (success) {
    workflowsCreated.add(1);
  }
  
  return res;
}

// Test: List Workflows
function testListWorkflows() {
  const res = http.get(`${BASE_URL}/api/v1/workflows?page=1&pageSize=20`, {
    headers: getHeaders(),
  });
  
  const success = check(res, {
    'list workflows status is 200': (r) => r.status === 200,
    'list workflows returns array': (r) => {
      try {
        const body = JSON.parse(r.body);
        return Array.isArray(body.workflows);
      } catch {
        return false;
      }
    },
  });
  
  errorRate.add(!success);
  return res;
}

// Test: Execute Workflow
function testExecuteWorkflow(workflowId) {
  const payload = JSON.stringify({
    input: { testKey: 'testValue' },
    async: true,
  });
  
  const start = new Date();
  const res = http.post(`${BASE_URL}/api/v1/workflows/${workflowId}/execute`, payload, {
    headers: getHeaders(),
  });
  workflowExecuteTrend.add(new Date() - start);
  
  const success = check(res, {
    'execute workflow status is 200 or 202': (r) => r.status === 200 || r.status === 202,
    'execute workflow returns execution id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.executionId !== undefined;
      } catch {
        return false;
      }
    },
  });
  
  errorRate.add(!success);
  if (success) {
    executionsStarted.add(1);
  }
  
  return res;
}

// Test: Search
function testSearch() {
  const payload = JSON.stringify({
    query: 'workflow',
    indexes: ['workflows'],
    from: 0,
    size: 10,
  });
  
  const res = http.post(`${BASE_URL}/api/v1/search`, payload, {
    headers: getHeaders(),
  });
  
  const success = check(res, {
    'search status is 200': (r) => r.status === 200,
  });
  
  errorRate.add(!success);
  return res;
}

// Test: Get Notifications
function testNotifications() {
  const res = http.get(`${BASE_URL}/api/v1/notifications?page=1&pageSize=10`, {
    headers: getHeaders(),
  });
  
  const success = check(res, {
    'notifications status is 200': (r) => r.status === 200,
  });
  
  errorRate.add(!success);
  return res;
}

// Smoke Test
export function smokeTest() {
  group('Smoke Test', () => {
    healthCheck();
    sleep(1);
    testListWorkflows();
    sleep(1);
  });
}

// Load Test
export function loadTest() {
  group('Load Test - Full User Journey', () => {
    // Health check
    if (!healthCheck()) {
      console.error('Health check failed, skipping iteration');
      return;
    }
    
    // Login
    testLogin();
    sleep(0.5);
    
    // List existing workflows
    testListWorkflows();
    sleep(0.5);
    
    // Create a new workflow
    const createRes = testCreateWorkflow();
    sleep(0.5);
    
    // Execute the workflow if created successfully
    if (createRes.status === 201) {
      try {
        const workflow = JSON.parse(createRes.body);
        if (workflow.id) {
          testExecuteWorkflow(workflow.id);
        }
      } catch (e) {
        console.error('Failed to parse workflow response');
      }
    }
    sleep(0.5);
    
    // Search
    testSearch();
    sleep(0.5);
    
    // Check notifications
    testNotifications();
    sleep(1);
  });
}

// Stress Test
export function stressTest() {
  group('Stress Test', () => {
    // Rapid fire requests
    healthCheck();
    testListWorkflows();
    testCreateWorkflow();
    testSearch();
    sleep(0.2);
  });
}

// Spike Test
export function spikeTest() {
  group('Spike Test', () => {
    // Critical operations only
    healthCheck();
    testListWorkflows();
    const createRes = testCreateWorkflow();
    
    if (createRes.status === 201) {
      try {
        const workflow = JSON.parse(createRes.body);
        if (workflow.id) {
          testExecuteWorkflow(workflow.id);
        }
      } catch (e) {
        // Ignore parse errors during spike
      }
    }
    
    sleep(0.1);
  });
}

// Default function (if no scenario specified)
export default function() {
  loadTest();
}
