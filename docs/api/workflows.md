# Workflows API

## Overview

The Workflows API allows you to create, manage, and execute automated workflows.

## Workflow Object

```json
{
  "id": "wf-123456",
  "name": "My Workflow",
  "description": "Automated data processing workflow",
  "status": "active",
  "nodes": [
    {
      "id": "node-1",
      "type": "manual_trigger",
      "name": "Start",
      "config": {},
      "position": {"x": 100, "y": 100}
    },
    {
      "id": "node-2",
      "type": "http_request",
      "name": "Fetch Data",
      "config": {
        "url": "https://api.example.com/data",
        "method": "GET"
      },
      "position": {"x": 300, "y": 100}
    }
  ],
  "connections": [
    {
      "id": "conn-1",
      "source": "node-1",
      "target": "node-2",
      "sourcePort": "output",
      "targetPort": "input"
    }
  ],
  "settings": {
    "timezone": "UTC",
    "errorHandling": "stop",
    "timeout": "30m"
  },
  "version": 1,
  "createdAt": "2024-12-19T12:00:00Z",
  "updatedAt": "2024-12-19T12:00:00Z"
}
```

## List Workflows

### GET /api/v1/workflows

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `status` | string | Filter by status: `draft`, `active`, `inactive`, `archived` |
| `search` | string | Search by name or description |
| `folderId` | string | Filter by folder |
| `tags` | string | Comma-separated tag filter |
| `limit` | integer | Results per page (default: 20, max: 100) |
| `offset` | integer | Pagination offset |
| `sort` | string | Sort field: `name`, `createdAt`, `updatedAt` |
| `order` | string | Sort order: `asc`, `desc` |

**Response:**
```json
{
  "success": true,
  "data": {
    "workflows": [
      {
        "id": "wf-123",
        "name": "Daily Report",
        "status": "active",
        "nodeCount": 5,
        "lastExecuted": "2024-12-19T10:00:00Z",
        "createdAt": "2024-12-01T00:00:00Z"
      }
    ]
  },
  "pagination": {
    "total": 50,
    "limit": 20,
    "offset": 0,
    "hasMore": true
  }
}
```

## Get Workflow

### GET /api/v1/workflows/{id}

**Response:**
```json
{
  "success": true,
  "data": {
    "workflow": {
      "id": "wf-123",
      "name": "My Workflow",
      "description": "...",
      "status": "active",
      "nodes": [...],
      "connections": [...],
      "settings": {...},
      "version": 1,
      "createdAt": "2024-12-19T12:00:00Z",
      "updatedAt": "2024-12-19T12:00:00Z"
    }
  }
}
```

## Create Workflow

### POST /api/v1/workflows

**Request:**
```json
{
  "name": "New Workflow",
  "description": "Workflow description",
  "nodes": [
    {
      "id": "trigger",
      "type": "manual_trigger",
      "name": "Start",
      "config": {},
      "position": {"x": 100, "y": 100}
    }
  ],
  "connections": [],
  "settings": {
    "timezone": "UTC",
    "errorHandling": "stop"
  },
  "folderId": "folder-123",
  "tags": ["automation", "reports"]
}
```

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "workflow": {
      "id": "wf-new-123",
      "name": "New Workflow",
      "status": "draft",
      ...
    }
  }
}
```

## Update Workflow

### PUT /api/v1/workflows/{id}

**Request:**
```json
{
  "name": "Updated Workflow",
  "description": "Updated description",
  "nodes": [...],
  "connections": [...],
  "settings": {...}
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "workflow": {
      "id": "wf-123",
      "version": 2,
      ...
    }
  }
}
```

## Delete Workflow

### DELETE /api/v1/workflows/{id}

**Query Parameters:**
- `permanent`: If `true`, permanently delete. Otherwise, archive.

**Response (204 No Content)**

## Activate Workflow

### POST /api/v1/workflows/{id}/activate

Enable workflow for scheduled/webhook execution.

**Response:**
```json
{
  "success": true,
  "data": {
    "workflow": {
      "id": "wf-123",
      "status": "active"
    }
  }
}
```

## Deactivate Workflow

### POST /api/v1/workflows/{id}/deactivate

Disable workflow execution.

**Response:**
```json
{
  "success": true,
  "data": {
    "workflow": {
      "id": "wf-123",
      "status": "inactive"
    }
  }
}
```

## Duplicate Workflow

### POST /api/v1/workflows/{id}/duplicate

Create a copy of the workflow.

**Request:**
```json
{
  "name": "Copy of My Workflow"
}
```

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "workflow": {
      "id": "wf-new-456",
      "name": "Copy of My Workflow",
      "status": "draft"
    }
  }
}
```

## Export Workflow

### GET /api/v1/workflows/{id}/export

Export workflow as JSON.

**Response:**
```json
{
  "success": true,
  "data": {
    "workflow": {...},
    "exportVersion": "1.0",
    "exportedAt": "2024-12-19T12:00:00Z"
  }
}
```

## Import Workflow

### POST /api/v1/workflows/import

Import workflow from JSON.

**Request:**
```json
{
  "workflow": {...},
  "name": "Imported Workflow",
  "credentialMapping": {
    "old-cred-id": "new-cred-id"
  }
}
```

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "workflow": {
      "id": "wf-imported-123"
    }
  }
}
```

## Node Configuration

### Node Object

```json
{
  "id": "node-123",
  "type": "http_request",
  "name": "Fetch Data",
  "description": "Fetch data from API",
  "config": {
    "url": "https://api.example.com/{{ $input.endpoint }}",
    "method": "GET",
    "headers": {
      "Authorization": "Bearer {{ $credentials.apiKey }}"
    }
  },
  "position": {"x": 300, "y": 100},
  "disabled": false,
  "retryOnFail": true,
  "maxRetries": 3,
  "retryDelay": 1000
}
```

### Connection Object

```json
{
  "id": "conn-123",
  "source": "node-1",
  "target": "node-2",
  "sourcePort": "output",
  "targetPort": "input",
  "condition": "{{ $output.success }}"
}
```

## Workflow Settings

```json
{
  "timezone": "America/New_York",
  "errorHandling": "continue",
  "timeout": "1h",
  "maxConcurrency": 5,
  "saveExecutionProgress": true,
  "staticData": {
    "counter": 0
  }
}
```

| Setting | Type | Description |
|---------|------|-------------|
| `timezone` | string | Timezone for schedule evaluation |
| `errorHandling` | string | `stop`, `continue`, `retry` |
| `timeout` | string | Maximum execution time |
| `maxConcurrency` | integer | Max parallel executions |
| `saveExecutionProgress` | boolean | Save intermediate state |
| `staticData` | object | Persistent workflow data |

## Validation

Workflows are validated on save:

- **Cycle detection**: No circular dependencies
- **Required connections**: All required inputs connected
- **Valid node types**: All nodes are registered
- **Valid expressions**: All expressions parse correctly

**Validation Error:**
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Workflow validation failed",
    "details": [
      {
        "type": "cycle_detected",
        "nodes": ["node-1", "node-2", "node-3"]
      }
    ]
  }
}
```

## Next Steps

- [Executions API](executions.md)
- [Node Types Reference](../reference/node-types.md)
- [Expression Functions](../reference/expression-functions.md)
