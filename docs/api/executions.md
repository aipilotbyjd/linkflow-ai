# Executions API

## Overview

The Executions API allows you to run workflows, monitor progress, and retrieve execution history.

## Execution Object

```json
{
  "id": "exec-123456",
  "workflowId": "wf-789",
  "workflowName": "My Workflow",
  "status": "completed",
  "mode": "manual",
  "startedAt": "2024-12-19T12:00:00Z",
  "finishedAt": "2024-12-19T12:00:30Z",
  "duration": 30000,
  "nodeExecutions": [
    {
      "nodeId": "node-1",
      "nodeName": "Start",
      "status": "completed",
      "startedAt": "2024-12-19T12:00:00Z",
      "finishedAt": "2024-12-19T12:00:01Z",
      "input": {},
      "output": {"triggered": true}
    }
  ],
  "error": null,
  "retryCount": 0,
  "triggeredBy": "usr-123"
}
```

## Execute Workflow

### POST /api/v1/execute

Execute a workflow immediately.

**Request:**
```json
{
  "workflow": {
    "id": "wf-123",
    "name": "My Workflow",
    "nodes": [...],
    "connections": [...]
  },
  "options": {
    "inputData": {
      "key": "value"
    },
    "runPartial": false,
    "startNodeId": null
  }
}
```

**Alternative - Execute by ID:**
```json
{
  "workflowId": "wf-123",
  "options": {
    "inputData": {
      "key": "value"
    }
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "executionId": "exec-123456",
    "status": "completed",
    "outputs": {
      "node-final": {
        "result": "success",
        "data": {...}
      }
    },
    "duration": 5230,
    "nodeCount": 5
  }
}
```

### Async Execution

For long-running workflows, use async mode:

```json
{
  "workflowId": "wf-123",
  "options": {
    "async": true,
    "webhookUrl": "https://yoursite.com/callback"
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "executionId": "exec-123456",
    "status": "running",
    "message": "Execution started. Use WebSocket or webhook for updates."
  }
}
```

## List Executions

### GET /api/v1/executions

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `workflowId` | string | Filter by workflow |
| `status` | string | Filter: `pending`, `running`, `completed`, `failed`, `cancelled` |
| `mode` | string | Filter: `manual`, `schedule`, `webhook`, `api` |
| `startedAfter` | datetime | Filter by start time |
| `startedBefore` | datetime | Filter by start time |
| `limit` | integer | Results per page (default: 20) |
| `offset` | integer | Pagination offset |

**Response:**
```json
{
  "success": true,
  "data": {
    "executions": [
      {
        "id": "exec-123",
        "workflowId": "wf-789",
        "workflowName": "Daily Report",
        "status": "completed",
        "mode": "schedule",
        "startedAt": "2024-12-19T12:00:00Z",
        "finishedAt": "2024-12-19T12:00:30Z",
        "duration": 30000
      }
    ]
  },
  "pagination": {
    "total": 100,
    "limit": 20,
    "offset": 0
  }
}
```

## Get Execution

### GET /api/v1/executions/{id}

**Query Parameters:**
- `includeData`: Include full node input/output data (default: false)

**Response:**
```json
{
  "success": true,
  "data": {
    "execution": {
      "id": "exec-123",
      "workflowId": "wf-789",
      "status": "completed",
      "nodeExecutions": [
        {
          "nodeId": "node-1",
          "nodeName": "HTTP Request",
          "status": "completed",
          "startedAt": "2024-12-19T12:00:01Z",
          "finishedAt": "2024-12-19T12:00:02Z",
          "input": {"url": "https://api.example.com"},
          "output": {"statusCode": 200, "body": {...}}
        }
      ],
      "logs": [
        {
          "timestamp": "2024-12-19T12:00:00Z",
          "level": "info",
          "message": "Execution started",
          "nodeId": null
        }
      ]
    }
  }
}
```

## Cancel Execution

### POST /api/v1/executions/{id}/cancel

Cancel a running execution.

**Response:**
```json
{
  "success": true,
  "data": {
    "execution": {
      "id": "exec-123",
      "status": "cancelled",
      "cancelledAt": "2024-12-19T12:05:00Z"
    }
  }
}
```

## Retry Execution

### POST /api/v1/executions/{id}/retry

Retry a failed execution.

**Request:**
```json
{
  "fromNode": "node-3",
  "inputOverrides": {
    "key": "newValue"
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "executionId": "exec-456",
    "originalExecutionId": "exec-123",
    "status": "running"
  }
}
```

## Execution Status

| Status | Description |
|--------|-------------|
| `pending` | Queued, waiting to start |
| `running` | Currently executing |
| `completed` | Finished successfully |
| `failed` | Finished with error |
| `cancelled` | Manually cancelled |
| `paused` | Waiting (debug mode) |

## Node Execution Status

| Status | Description |
|--------|-------------|
| `pending` | Not yet started |
| `running` | Currently executing |
| `completed` | Finished successfully |
| `failed` | Failed with error |
| `skipped` | Skipped (condition not met) |

## Execution Modes

| Mode | Description |
|------|-------------|
| `manual` | Triggered via UI or API |
| `schedule` | Triggered by cron schedule |
| `webhook` | Triggered by incoming webhook |
| `api` | Triggered by external API call |
| `replay` | Re-execution of previous run |

## Real-time Updates

### WebSocket Connection

```javascript
const ws = new WebSocket('wss://api.linkflow.ai/ws');

// Subscribe to execution updates
ws.send(JSON.stringify({
  type: 'subscribe',
  channel: 'execution',
  executionId: 'exec-123'
}));

// Receive updates
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  
  switch(data.type) {
    case 'execution.node.completed':
      console.log('Node completed:', data.payload.nodeId);
      break;
    case 'execution.completed':
      console.log('Execution finished');
      break;
    case 'execution.failed':
      console.log('Execution failed:', data.payload.error);
      break;
  }
};
```

### Webhook Callback

When using async execution with webhook:

```json
// POST to your webhookUrl
{
  "event": "execution.completed",
  "executionId": "exec-123",
  "status": "completed",
  "duration": 5230,
  "outputs": {...},
  "timestamp": "2024-12-19T12:00:30Z"
}
```

## Error Handling

### Execution Error Response

```json
{
  "success": true,
  "data": {
    "execution": {
      "id": "exec-123",
      "status": "failed",
      "error": {
        "message": "HTTP request failed: 404 Not Found",
        "nodeId": "node-3",
        "nodeName": "Fetch Data",
        "code": "HTTP_ERROR",
        "details": {
          "statusCode": 404,
          "response": "Not Found"
        }
      }
    }
  }
}
```

### Common Error Codes

| Code | Description |
|------|-------------|
| `TIMEOUT` | Execution exceeded timeout |
| `NODE_ERROR` | Node execution failed |
| `HTTP_ERROR` | HTTP request failed |
| `EXPRESSION_ERROR` | Expression evaluation failed |
| `CREDENTIAL_ERROR` | Credential access failed |
| `RATE_LIMITED` | External API rate limited |

## Execution Metrics

### GET /api/v1/executions/stats

Get execution statistics.

**Query Parameters:**
- `workflowId`: Filter by workflow
- `period`: `day`, `week`, `month`

**Response:**
```json
{
  "success": true,
  "data": {
    "stats": {
      "total": 1500,
      "completed": 1400,
      "failed": 80,
      "cancelled": 20,
      "avgDuration": 15000,
      "successRate": 93.3,
      "byDay": [
        {"date": "2024-12-19", "count": 50, "failed": 2}
      ]
    }
  }
}
```

## Next Steps

- [Workflows API](workflows.md)
- [Webhooks API](webhooks.md)
- [Node Types Reference](../reference/node-types.md)
