# LinkFlow AI - API Reference

## Overview

The LinkFlow AI API is a RESTful API that provides programmatic access to the workflow automation platform. All API requests are made over HTTPS to `https://api.linkflow.ai`.

## Authentication

The API uses JWT (JSON Web Token) authentication. You can obtain a token by logging in with your credentials or using an API key.

### Login Authentication

```bash
curl -X POST https://api.linkflow.ai/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "your-password"
  }'
```

Response:
```json
{
  "user": {
    "id": "user-123",
    "email": "user@example.com",
    "firstName": "John",
    "lastName": "Doe"
  },
  "tokens": {
    "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expiresIn": 86400
  }
}
```

### API Key Authentication

```bash
curl -X GET https://api.linkflow.ai/api/v1/workflows \
  -H "X-API-Key: your-api-key"
```

## Rate Limiting

API requests are rate limited to:
- **Authenticated requests**: 1000 requests per minute
- **Unauthenticated requests**: 100 requests per minute

Rate limit information is included in response headers:
- `X-RateLimit-Limit`: Maximum requests allowed
- `X-RateLimit-Remaining`: Requests remaining
- `X-RateLimit-Reset`: Time when the limit resets (Unix timestamp)

## Pagination

List endpoints support pagination using `page` and `pageSize` parameters:

```bash
GET /api/v1/workflows?page=2&pageSize=20
```

Response includes pagination metadata:
```json
{
  "workflows": [...],
  "totalCount": 150,
  "page": 2,
  "pageSize": 20
}
```

## Error Handling

Errors are returned with appropriate HTTP status codes and a JSON error response:

```json
{
  "code": "VALIDATION_ERROR",
  "message": "Invalid request parameters",
  "details": {
    "field": "name",
    "error": "Name is required"
  }
}
```

Common status codes:
- `200 OK` - Request successful
- `201 Created` - Resource created
- `204 No Content` - Request successful, no content returned
- `400 Bad Request` - Invalid request parameters
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Access denied
- `404 Not Found` - Resource not found
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Server error

## Endpoints

### Authentication

#### POST /api/v1/auth/register
Register a new user account.

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "SecurePassword123!",
  "firstName": "John",
  "lastName": "Doe",
  "organizationName": "Acme Corp"
}
```

#### POST /api/v1/auth/login
Login with email and password.

#### POST /api/v1/auth/refresh
Refresh access token using refresh token.

#### POST /api/v1/auth/logout
Logout current session.

#### GET /api/v1/auth/me
Get current authenticated user.

### Workflows

#### GET /api/v1/workflows
List all workflows.

**Query Parameters:**
- `page` (integer): Page number
- `pageSize` (integer): Items per page
- `status` (string): Filter by status (draft, active, inactive, archived)
- `tags` (array): Filter by tags

#### POST /api/v1/workflows
Create a new workflow.

**Request Body:**
```json
{
  "name": "My Workflow",
  "description": "Workflow description",
  "nodes": [
    {
      "id": "node-1",
      "type": "trigger",
      "name": "Start",
      "position": {"x": 100, "y": 100}
    }
  ],
  "connections": [],
  "tags": ["automation", "daily"]
}
```

#### GET /api/v1/workflows/{id}
Get workflow by ID.

#### PUT /api/v1/workflows/{id}
Update workflow.

#### DELETE /api/v1/workflows/{id}
Delete workflow.

#### POST /api/v1/workflows/{id}/activate
Activate workflow.

#### POST /api/v1/workflows/{id}/deactivate
Deactivate workflow.

#### POST /api/v1/workflows/{id}/execute
Execute workflow.

**Request Body:**
```json
{
  "input": {
    "key": "value"
  },
  "async": true
}
```

### Executions

#### GET /api/v1/executions
List workflow executions.

#### GET /api/v1/executions/{id}
Get execution details.

#### POST /api/v1/executions/{id}/cancel
Cancel running execution.

#### GET /api/v1/executions/{id}/logs
Get execution logs.

### Schedules

#### GET /api/v1/schedules
List schedules.

#### POST /api/v1/schedules
Create schedule.

**Request Body:**
```json
{
  "workflowId": "workflow-123",
  "name": "Daily Report",
  "cronExpression": "0 9 * * *",
  "timezone": "America/New_York",
  "enabled": true
}
```

#### GET /api/v1/schedules/{id}
Get schedule details.

#### PUT /api/v1/schedules/{id}
Update schedule.

#### DELETE /api/v1/schedules/{id}
Delete schedule.

#### POST /api/v1/schedules/{id}/pause
Pause schedule.

#### POST /api/v1/schedules/{id}/resume
Resume schedule.

### Webhooks

#### GET /api/v1/webhooks
List webhooks.

#### POST /api/v1/webhooks
Create webhook.

**Request Body:**
```json
{
  "name": "GitHub Webhook",
  "endpointUrl": "https://example.com/webhook",
  "events": ["workflow.completed", "workflow.failed"],
  "secret": "webhook-secret",
  "enabled": true
}
```

#### GET /api/v1/webhooks/{id}
Get webhook details.

#### PUT /api/v1/webhooks/{id}
Update webhook.

#### DELETE /api/v1/webhooks/{id}
Delete webhook.

#### POST /api/v1/webhooks/{id}/test
Test webhook.

### Notifications

#### GET /api/v1/notifications
List notifications.

#### POST /api/v1/notifications
Send notification.

**Request Body:**
```json
{
  "title": "Workflow Completed",
  "message": "Your workflow has completed successfully",
  "channels": ["email", "in_app"],
  "priority": "high",
  "metadata": {
    "workflowId": "workflow-123"
  }
}
```

#### GET /api/v1/notifications/{id}
Get notification details.

#### POST /api/v1/notifications/{id}/read
Mark notification as read.

#### DELETE /api/v1/notifications/{id}
Delete notification.

### Analytics

#### POST /api/v1/analytics/track
Track analytics event.

**Request Body:**
```json
{
  "eventType": "workflow_executed",
  "eventName": "Daily Report Generated",
  "properties": {
    "workflowId": "workflow-123",
    "duration": 1500
  }
}
```

#### GET /api/v1/analytics/metrics
Get analytics metrics.

**Query Parameters:**
- `startDate` (string): Start date (ISO 8601)
- `endDate` (string): End date (ISO 8601)
- `metrics` (array): Metrics to retrieve

#### GET /api/v1/analytics/user
Get user analytics.

### File Storage

#### POST /api/v1/files/upload
Upload file.

**Request:** Multipart form data with file

#### GET /api/v1/files/{id}/download
Download file.

#### DELETE /api/v1/files/{id}
Delete file.

#### GET /api/v1/files
List user files.

### Search

#### POST /api/v1/search
Search across all resources.

**Request Body:**
```json
{
  "query": "search term",
  "indexes": ["workflows", "executions"],
  "filters": {
    "status": "active"
  },
  "from": 0,
  "size": 20
}
```

## WebSocket Events

Connect to WebSocket for real-time events:

```javascript
const ws = new WebSocket('wss://api.linkflow.ai/ws');

ws.onopen = () => {
  ws.send(JSON.stringify({
    type: 'auth',
    token: 'your-jwt-token'
  }));
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Event:', data);
};
```

Event types:
- `execution.started`
- `execution.completed`
- `execution.failed`
- `workflow.updated`
- `notification.new`

## SDK Usage

### Go SDK

```go
import "github.com/linkflow-ai/linkflow-ai/pkg/sdk"

client := sdk.NewClient("https://api.linkflow.ai")

// Login
auth, err := client.Auth.Login(ctx, "user@example.com", "password")

// Create workflow
workflow, err := client.Workflows.Create(ctx, &sdk.CreateWorkflowRequest{
    Name: "My Workflow",
    Description: "Description",
})

// Execute workflow
result, err := client.Workflows.Execute(ctx, workflow.ID, &sdk.ExecuteWorkflowRequest{
    Input: map[string]interface{}{
        "data": "test",
    },
    Async: true,
})
```

### JavaScript SDK

```javascript
import { LinkFlowClient } from '@linkflow/sdk';

const client = new LinkFlowClient({
  baseURL: 'https://api.linkflow.ai',
  apiKey: 'your-api-key'
});

// Create workflow
const workflow = await client.workflows.create({
  name: 'My Workflow',
  description: 'Description'
});

// Execute workflow
const execution = await client.workflows.execute(workflow.id, {
  input: { data: 'test' },
  async: true
});
```

## Support

For API support, please contact:
- Email: api-support@linkflow.ai
- Documentation: https://docs.linkflow.ai
- Status Page: https://status.linkflow.ai
