// K6 Load Testing Script for LinkFlow AI Workflow Service
// Run with: k6 run workflow_load_test.js

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const workflowCreationTime = new Trend('workflow_creation_time');
const workflowExecutionTime = new Trend('workflow_execution_time');

// Test configuration
export const options = {
  stages: [
    { duration: '2m', target: 10 },   // Ramp up to 10 users
    { duration: '5m', target: 50 },   // Ramp up to 50 users
    { duration: '10m', target: 100 }, // Ramp up to 100 users
    { duration: '5m', target: 200 },  // Spike to 200 users
    { duration: '10m', target: 100 }, // Back to 100 users
    { duration: '5m', target: 50 },   // Scale down to 50 users
    { duration: '2m', target: 0 },    // Ramp down to 0 users
  ],
  thresholds: {
    'http_req_duration': ['p(95)<500', 'p(99)<1000'], // 95% of requests must complete below 500ms
    'errors': ['rate<0.05'], // Error rate must be below 5%
    'workflow_creation_time': ['p(95)<1000'], // 95% of workflow creations below 1s
    'workflow_execution_time': ['p(95)<2000'], // 95% of workflow executions below 2s
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8000';
const AUTH_TOKEN = __ENV.AUTH_TOKEN || '';

// Setup - Run once per VU
export function setup() {
  // Login and get token if not provided
  if (!AUTH_TOKEN) {
    const loginRes = http.post(`${BASE_URL}/api/v1/auth/login`, JSON.stringify({
      email: 'loadtest@linkflow.ai',
      password: 'LoadTest123!',
    }), {
      headers: { 'Content-Type': 'application/json' },
    });

    if (loginRes.status === 200) {
      const tokens = JSON.parse(loginRes.body).tokens;
      return { token: tokens.accessToken };
    }
  }
  return { token: AUTH_TOKEN };
}

// Main test scenario
export default function(data) {
  const token = data.token;
  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`,
  };

  // Scenario 1: Create Workflow
  const createWorkflowPayload = {
    name: `Load Test Workflow ${Date.now()}`,
    description: 'Automated load test workflow',
    nodes: [
      {
        id: 'trigger-1',
        type: 'trigger',
        name: 'Start',
        position: { x: 100, y: 100 },
      },
      {
        id: 'action-1',
        type: 'action',
        name: 'Process',
        position: { x: 300, y: 100 },
      },
      {
        id: 'action-2',
        type: 'action',
        name: 'Complete',
        position: { x: 500, y: 100 },
      },
    ],
    connections: [
      {
        id: 'conn-1',
        sourceNodeId: 'trigger-1',
        targetNodeId: 'action-1',
      },
      {
        id: 'conn-2',
        sourceNodeId: 'action-1',
        targetNodeId: 'action-2',
      },
    ],
  };

  const startCreate = Date.now();
  const createRes = http.post(
    `${BASE_URL}/api/v1/workflows`,
    JSON.stringify(createWorkflowPayload),
    { headers }
  );
  workflowCreationTime.add(Date.now() - startCreate);

  const createSuccess = check(createRes, {
    'workflow created': (r) => r.status === 201,
    'workflow has ID': (r) => JSON.parse(r.body).id !== undefined,
  });
  errorRate.add(!createSuccess);

  if (!createSuccess) {
    console.error(`Failed to create workflow: ${createRes.status} - ${createRes.body}`);
    return;
  }

  const workflowId = JSON.parse(createRes.body).id;
  sleep(1);

  // Scenario 2: Get Workflow
  const getRes = http.get(`${BASE_URL}/api/v1/workflows/${workflowId}`, { headers });
  check(getRes, {
    'workflow retrieved': (r) => r.status === 200,
    'workflow ID matches': (r) => JSON.parse(r.body).id === workflowId,
  });

  sleep(1);

  // Scenario 3: List Workflows
  const listRes = http.get(`${BASE_URL}/api/v1/workflows?page=1&pageSize=10`, { headers });
  check(listRes, {
    'workflows listed': (r) => r.status === 200,
    'has workflows': (r) => JSON.parse(r.body).workflows.length > 0,
  });

  sleep(1);

  // Scenario 4: Activate Workflow
  const activateRes = http.post(
    `${BASE_URL}/api/v1/workflows/${workflowId}/activate`,
    null,
    { headers }
  );
  check(activateRes, {
    'workflow activated': (r) => r.status === 200,
    'status is active': (r) => JSON.parse(r.body).status === 'active',
  });

  sleep(1);

  // Scenario 5: Execute Workflow
  const executePayload = {
    input: {
      testData: `Load test execution ${Date.now()}`,
      timestamp: new Date().toISOString(),
    },
    async: false,
  };

  const startExecute = Date.now();
  const executeRes = http.post(
    `${BASE_URL}/api/v1/workflows/${workflowId}/execute`,
    JSON.stringify(executePayload),
    { headers }
  );
  workflowExecutionTime.add(Date.now() - startExecute);

  const executeSuccess = check(executeRes, {
    'workflow executed': (r) => r.status === 200,
    'has execution ID': (r) => JSON.parse(r.body).executionId !== undefined,
  });
  errorRate.add(!executeSuccess);

  sleep(2);

  // Scenario 6: Check Execution Status
  if (executeSuccess) {
    const executionId = JSON.parse(executeRes.body).executionId;
    const statusRes = http.get(`${BASE_URL}/api/v1/executions/${executionId}`, { headers });
    check(statusRes, {
      'execution status retrieved': (r) => r.status === 200,
      'execution completed or running': (r) => {
        const status = JSON.parse(r.body).status;
        return status === 'completed' || status === 'running';
      },
    });
  }

  sleep(1);

  // Scenario 7: Update Workflow
  const updatePayload = {
    name: `Updated Load Test Workflow ${Date.now()}`,
    description: 'Updated description',
    tags: ['load-test', 'updated'],
  };

  const updateRes = http.put(
    `${BASE_URL}/api/v1/workflows/${workflowId}`,
    JSON.stringify(updatePayload),
    { headers }
  );
  check(updateRes, {
    'workflow updated': (r) => r.status === 200,
    'name updated': (r) => JSON.parse(r.body).name.includes('Updated'),
  });

  sleep(1);

  // Scenario 8: Delete Workflow (cleanup)
  const deleteRes = http.del(`${BASE_URL}/api/v1/workflows/${workflowId}`, null, { headers });
  check(deleteRes, {
    'workflow deleted': (r) => r.status === 204,
  });

  sleep(2);
}

// Teardown - Run once after all VUs finish
export function teardown(data) {
  console.log('Load test completed');
}
