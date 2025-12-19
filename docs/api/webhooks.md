# Webhooks API

## Overview

Webhooks allow external services to trigger workflow executions via HTTP requests.

## Webhook Object

```json
{
  "id": "wh-123456",
  "workflowId": "wf-789",
  "name": "GitHub Webhook",
  "path": "/webhook/abc123def456",
  "url": "https://api.linkflow.ai/webhook/abc123def456",
  "method": "POST",
  "isActive": true,
  "authentication": {
    "type": "header",
    "headerName": "X-Hub-Signature-256"
  },
  "lastTriggered": "2024-12-19T12:00:00Z",
  "triggerCount": 150,
  "createdAt": "2024-12-01T00:00:00Z"
}
```

## List Webhooks

### GET /api/v1/webhooks

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `workflowId` | string | Filter by workflow |
| `isActive` | boolean | Filter by active status |
| `limit` | integer | Results per page |
| `offset` | integer | Pagination offset |

**Response:**
```json
{
  "success": true,
  "data": {
    "webhooks": [
      {
        "id": "wh-123",
        "workflowId": "wf-789",
        "name": "GitHub Webhook",
        "url": "https://api.linkflow.ai/webhook/abc123",
        "isActive": true,
        "triggerCount": 150
      }
    ]
  }
}
```

## Get Webhook

### GET /api/v1/webhooks/{id}

**Response:**
```json
{
  "success": true,
  "data": {
    "webhook": {
      "id": "wh-123",
      "workflowId": "wf-789",
      "name": "GitHub Webhook",
      "path": "/webhook/abc123def456",
      "url": "https://api.linkflow.ai/webhook/abc123def456",
      "method": "POST",
      "isActive": true,
      "authentication": {
        "type": "header",
        "headerName": "X-Hub-Signature-256",
        "secret": "********"
      },
      "responseMode": "lastNode",
      "responseCode": 200,
      "headers": {
        "Content-Type": "application/json"
      }
    }
  }
}
```

## Create Webhook

### POST /api/v1/webhooks

**Request:**
```json
{
  "workflowId": "wf-789",
  "name": "GitHub Webhook",
  "method": "POST",
  "authentication": {
    "type": "header",
    "headerName": "X-Hub-Signature-256",
    "secret": "your-webhook-secret"
  },
  "responseMode": "lastNode",
  "options": {
    "rawBody": true,
    "timeout": 30000
  }
}
```

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "webhook": {
      "id": "wh-new-123",
      "url": "https://api.linkflow.ai/webhook/xyz789abc123",
      "path": "/webhook/xyz789abc123"
    }
  }
}
```

## Update Webhook

### PUT /api/v1/webhooks/{id}

**Request:**
```json
{
  "name": "Updated Webhook Name",
  "isActive": true,
  "authentication": {
    "type": "basic",
    "username": "user",
    "password": "newpassword"
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "webhook": {
      "id": "wh-123",
      "name": "Updated Webhook Name"
    }
  }
}
```

## Delete Webhook

### DELETE /api/v1/webhooks/{id}

**Response (204 No Content)**

## Regenerate Webhook Path

### POST /api/v1/webhooks/{id}/regenerate

Generate a new webhook URL (invalidates old URL).

**Response:**
```json
{
  "success": true,
  "data": {
    "webhook": {
      "id": "wh-123",
      "url": "https://api.linkflow.ai/webhook/newpath123",
      "path": "/webhook/newpath123"
    }
  }
}
```

## Trigger Webhook

### POST /webhook/{token}

Trigger a workflow via webhook.

**Request:**
```bash
curl -X POST https://api.linkflow.ai/webhook/abc123def456 \
  -H "Content-Type: application/json" \
  -H "X-Hub-Signature-256: sha256=..." \
  -d '{"event": "push", "repository": "my-repo"}'
```

**Response (depends on responseMode):**
```json
{
  "success": true,
  "executionId": "exec-123456",
  "status": "completed",
  "data": {
    "result": "Processed successfully"
  }
}
```

## Authentication Types

### None

No authentication required.

```json
{
  "authentication": {
    "type": "none"
  }
}
```

### Header Token

Validate a secret token in header.

```json
{
  "authentication": {
    "type": "header",
    "headerName": "X-Webhook-Token",
    "secret": "your-secret-token"
  }
}
```

### HMAC Signature

Validate HMAC signature (GitHub, Stripe style).

```json
{
  "authentication": {
    "type": "hmac",
    "headerName": "X-Hub-Signature-256",
    "secret": "your-hmac-secret",
    "algorithm": "sha256"
  }
}
```

### Basic Auth

HTTP Basic Authentication.

```json
{
  "authentication": {
    "type": "basic",
    "username": "webhook-user",
    "password": "secure-password"
  }
}
```

### JWT

Validate JWT token.

```json
{
  "authentication": {
    "type": "jwt",
    "headerName": "Authorization",
    "secret": "jwt-secret-key"
  }
}
```

## Response Modes

### Immediate (`immediate`)

Respond immediately before execution.

```json
{
  "responseMode": "immediate",
  "responseCode": 202,
  "responseBody": {"message": "Accepted"}
}
```

### Last Node (`lastNode`)

Wait for execution and return last node's output.

```json
{
  "responseMode": "lastNode",
  "responseCode": 200
}
```

### Custom Response (`custom`)

Return specified response after execution.

```json
{
  "responseMode": "custom",
  "responseCode": 200,
  "responseBody": {"status": "processed"},
  "responseHeaders": {
    "X-Custom-Header": "value"
  }
}
```

## Webhook Options

| Option | Type | Description |
|--------|------|-------------|
| `rawBody` | boolean | Pass raw body to workflow |
| `timeout` | integer | Response timeout in ms |
| `allowedMethods` | array | Allowed HTTP methods |
| `ipWhitelist` | array | Allowed IP addresses |
| `rateLimitPerMinute` | integer | Webhook-specific rate limit |

## Testing Webhooks

### POST /api/v1/webhooks/{id}/test

Test webhook with sample payload.

**Request:**
```json
{
  "method": "POST",
  "headers": {
    "Content-Type": "application/json"
  },
  "body": {
    "test": true,
    "data": "sample"
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "executionId": "exec-test-123",
    "status": "completed",
    "response": {...}
  }
}
```

## Webhook Logs

### GET /api/v1/webhooks/{id}/logs

Get recent webhook invocations.

**Query Parameters:**
- `limit`: Number of logs (default: 50)
- `status`: Filter by status

**Response:**
```json
{
  "success": true,
  "data": {
    "logs": [
      {
        "id": "log-123",
        "timestamp": "2024-12-19T12:00:00Z",
        "method": "POST",
        "path": "/webhook/abc123",
        "sourceIP": "192.168.1.1",
        "status": "success",
        "executionId": "exec-456",
        "responseCode": 200,
        "duration": 1500
      }
    ]
  }
}
```

## Integration Examples

### GitHub

```json
{
  "workflowId": "wf-github",
  "name": "GitHub Push Webhook",
  "authentication": {
    "type": "hmac",
    "headerName": "X-Hub-Signature-256",
    "secret": "github-webhook-secret",
    "algorithm": "sha256"
  }
}
```

### Stripe

```json
{
  "workflowId": "wf-stripe",
  "name": "Stripe Events",
  "authentication": {
    "type": "hmac",
    "headerName": "Stripe-Signature",
    "secret": "whsec_...",
    "algorithm": "sha256"
  },
  "options": {
    "rawBody": true
  }
}
```

### Slack

```json
{
  "workflowId": "wf-slack",
  "name": "Slack Commands",
  "authentication": {
    "type": "hmac",
    "headerName": "X-Slack-Signature",
    "secret": "slack-signing-secret",
    "algorithm": "sha256"
  },
  "responseMode": "immediate"
}
```

## Next Steps

- [Workflows API](workflows.md)
- [Executions API](executions.md)
- [Authentication](authentication.md)
