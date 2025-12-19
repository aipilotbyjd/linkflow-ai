# API Overview

LinkFlow AI provides a RESTful API for managing workflows, executions, and integrations.

## Base URL

```
Development: http://localhost:8080/api/v1
Production:  https://api.linkflow.ai/api/v1
```

## Authentication

All API requests (except public endpoints) require authentication.

### JWT Token

Include the JWT token in the Authorization header:

```bash
curl -H "Authorization: Bearer <your-jwt-token>" \
     https://api.linkflow.ai/api/v1/workflows
```

### API Key

For server-to-server communication:

```bash
curl -H "X-API-Key: <your-api-key>" \
     https://api.linkflow.ai/api/v1/workflows
```

## Request Format

- **Content-Type**: `application/json`
- **Accept**: `application/json`

## Response Format

All responses follow this structure:

### Success Response

```json
{
  "success": true,
  "data": {
    // Response data
  },
  "meta": {
    "requestId": "req-123456",
    "timestamp": "2024-12-19T12:00:00Z"
  }
}
```

### Error Response

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid workflow configuration",
    "details": [
      {
        "field": "nodes",
        "message": "At least one node is required"
      }
    ]
  },
  "meta": {
    "requestId": "req-123456",
    "timestamp": "2024-12-19T12:00:00Z"
  }
}
```

## HTTP Status Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 201 | Created |
| 204 | No Content |
| 400 | Bad Request - Invalid input |
| 401 | Unauthorized - Authentication required |
| 403 | Forbidden - Insufficient permissions |
| 404 | Not Found - Resource doesn't exist |
| 409 | Conflict - Resource already exists |
| 422 | Unprocessable Entity - Validation failed |
| 429 | Too Many Requests - Rate limited |
| 500 | Internal Server Error |

## Rate Limiting

API requests are rate limited:

- **Default**: 100 requests/minute per IP
- **Authenticated**: 1000 requests/minute per user

Rate limit headers:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
Retry-After: 60
```

## Pagination

List endpoints support pagination:

```bash
GET /api/v1/workflows?limit=20&offset=0
```

Response includes pagination metadata:

```json
{
  "data": [...],
  "pagination": {
    "total": 150,
    "limit": 20,
    "offset": 0,
    "hasMore": true
  }
}
```

## Filtering & Sorting

### Filtering

```bash
GET /api/v1/workflows?status=active&createdAfter=2024-01-01
```

### Sorting

```bash
GET /api/v1/workflows?sort=createdAt&order=desc
```

## API Endpoints

### Health & Status

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/api/v1/health` | Detailed health status |

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/auth/register` | Register new user |
| POST | `/api/v1/auth/login` | User login |
| POST | `/api/v1/auth/logout` | User logout |
| POST | `/api/v1/auth/refresh` | Refresh JWT token |
| POST | `/api/v1/auth/forgot-password` | Request password reset |
| POST | `/api/v1/auth/reset-password` | Reset password |

### Workflows

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/workflows` | List workflows |
| POST | `/api/v1/workflows` | Create workflow |
| GET | `/api/v1/workflows/{id}` | Get workflow |
| PUT | `/api/v1/workflows/{id}` | Update workflow |
| DELETE | `/api/v1/workflows/{id}` | Delete workflow |
| POST | `/api/v1/workflows/{id}/activate` | Activate workflow |
| POST | `/api/v1/workflows/{id}/deactivate` | Deactivate workflow |
| POST | `/api/v1/workflows/{id}/duplicate` | Duplicate workflow |

### Executions

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/executions` | List executions |
| POST | `/api/v1/execute` | Execute workflow |
| GET | `/api/v1/executions/{id}` | Get execution details |
| POST | `/api/v1/executions/{id}/cancel` | Cancel execution |
| POST | `/api/v1/executions/{id}/retry` | Retry failed execution |

### Nodes

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/nodes` | List available nodes |
| GET | `/api/v1/nodes/{type}` | Get node details |

### Credentials

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/credentials` | List credentials |
| POST | `/api/v1/credentials` | Create credential |
| GET | `/api/v1/credentials/{id}` | Get credential |
| PUT | `/api/v1/credentials/{id}` | Update credential |
| DELETE | `/api/v1/credentials/{id}` | Delete credential |

### Integrations

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/integrations` | List integrations |
| GET | `/api/v1/integrations/{id}/auth` | Get OAuth URL |
| POST | `/api/v1/integrations/{id}/callback` | OAuth callback |
| DELETE | `/api/v1/integrations/{id}` | Disconnect integration |

### Webhooks

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/webhooks` | List webhooks |
| POST | `/api/v1/webhooks` | Create webhook |
| GET | `/api/v1/webhooks/{id}` | Get webhook |
| DELETE | `/api/v1/webhooks/{id}` | Delete webhook |
| POST | `/webhook/{token}` | Incoming webhook |

### Schedules

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/schedules` | List schedules |
| POST | `/api/v1/schedules` | Create schedule |
| PUT | `/api/v1/schedules/{id}` | Update schedule |
| DELETE | `/api/v1/schedules/{id}` | Delete schedule |

### Templates

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/templates` | List templates |
| GET | `/api/v1/templates/{id}` | Get template |
| POST | `/api/v1/templates/{id}/instantiate` | Create workflow from template |

## WebSocket API

Connect to receive real-time updates:

```javascript
const ws = new WebSocket('wss://api.linkflow.ai/ws');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Event:', data.type, data.payload);
};

// Subscribe to execution updates
ws.send(JSON.stringify({
  type: 'subscribe',
  channel: 'execution',
  executionId: 'exec-123'
}));
```

### Event Types

| Event | Description |
|-------|-------------|
| `execution.started` | Execution started |
| `execution.completed` | Execution completed |
| `execution.failed` | Execution failed |
| `node.started` | Node execution started |
| `node.completed` | Node execution completed |
| `node.failed` | Node execution failed |

## SDKs

Official SDKs:
- [Go SDK](https://github.com/linkflow-ai/go-sdk)
- [Node.js SDK](https://github.com/linkflow-ai/node-sdk)
- [Python SDK](https://github.com/linkflow-ai/python-sdk)

## Next Steps

- [Authentication Details](authentication.md)
- [Workflows API](workflows.md)
- [Executions API](executions.md)
- [Webhooks API](webhooks.md)
