# Quick Start Guide

Get up and running with LinkFlow AI in 5 minutes.

## Prerequisites

Ensure you have completed the [Installation Guide](installation.md).

## Step 1: Start the Server

```bash
# Start the API server
go run ./cmd/services/api
```

The server will start on `http://localhost:8080`.

## Step 2: Create Your First Workflow

### Using the API

Create a simple workflow that triggers on a schedule and sends an HTTP request:

```bash
curl -X POST http://localhost:8080/api/v1/execute \
  -H "Content-Type: application/json" \
  -d '{
    "workflow": {
      "id": "my-first-workflow",
      "name": "My First Workflow",
      "nodes": [
        {
          "id": "trigger",
          "type": "manual_trigger",
          "name": "Start",
          "config": {},
          "position": {"x": 100, "y": 100}
        },
        {
          "id": "http",
          "type": "http_request",
          "name": "Fetch Data",
          "config": {
            "url": "https://api.github.com/zen",
            "method": "GET"
          },
          "position": {"x": 300, "y": 100}
        }
      ],
      "connections": [
        {
          "source": "trigger",
          "target": "http",
          "sourcePort": "output",
          "targetPort": "input"
        }
      ]
    },
    "options": {}
  }'
```

### Response

```json
{
  "executionId": "exec-123456",
  "status": "completed",
  "outputs": {
    "http": {
      "statusCode": 200,
      "body": "Keep it logically awesome."
    }
  }
}
```

## Step 3: Add Conditional Logic

Create a workflow with IF/ELSE branching:

```bash
curl -X POST http://localhost:8080/api/v1/execute \
  -H "Content-Type: application/json" \
  -d '{
    "workflow": {
      "id": "conditional-workflow",
      "name": "Conditional Workflow",
      "nodes": [
        {
          "id": "trigger",
          "type": "manual_trigger",
          "name": "Start",
          "config": {},
          "position": {"x": 100, "y": 100}
        },
        {
          "id": "set_data",
          "type": "set",
          "name": "Set Value",
          "config": {
            "values": {
              "score": 85
            }
          },
          "position": {"x": 300, "y": 100}
        },
        {
          "id": "check",
          "type": "if",
          "name": "Check Score",
          "config": {
            "condition": "{{ $input.score > 80 }}"
          },
          "position": {"x": 500, "y": 100}
        }
      ],
      "connections": [
        {"source": "trigger", "target": "set_data", "sourcePort": "output", "targetPort": "input"},
        {"source": "set_data", "target": "check", "sourcePort": "output", "targetPort": "input"}
      ]
    }
  }'
```

## Step 4: Use Integrations

### Send a Slack Message

```bash
curl -X POST http://localhost:8080/api/v1/execute \
  -H "Content-Type: application/json" \
  -d '{
    "workflow": {
      "id": "slack-workflow",
      "name": "Slack Notification",
      "nodes": [
        {
          "id": "trigger",
          "type": "manual_trigger",
          "name": "Start",
          "config": {}
        },
        {
          "id": "slack",
          "type": "slack",
          "name": "Send Message",
          "config": {
            "operation": "send_message",
            "channel": "#general",
            "message": "Hello from LinkFlow AI!"
          }
        }
      ],
      "connections": [
        {"source": "trigger", "target": "slack", "sourcePort": "output", "targetPort": "input"}
      ]
    }
  }'
```

## Step 5: Check Execution History

List recent executions:

```bash
curl http://localhost:8080/api/v1/executions
```

Get details of a specific execution:

```bash
curl http://localhost:8080/api/v1/executions/exec-123456
```

## Available Node Types

| Category | Nodes |
|----------|-------|
| **Triggers** | Manual, Schedule, Webhook, Interval |
| **Control Flow** | IF, Switch, Loop, Merge, Split |
| **Data** | Set, Code, HTTP Request |
| **Integrations** | Slack, Email, GitHub, PostgreSQL, MySQL, MongoDB, Google Sheets, Notion, Airtable, S3, Discord, Telegram |

## Expression Syntax

Use expressions to access data from previous nodes:

```javascript
// Access input data
{{ $input.fieldName }}

// Access data from specific node
{{ $node.nodeName.data.fieldName }}

// Use built-in functions
{{ $uppercase($input.name) }}
{{ $now() }}
{{ $formatDate($input.date, "YYYY-MM-DD") }}
```

## Next Steps

- [Creating Workflows](../guides/creating-workflows.md)
- [Node Types Reference](../reference/node-types.md)
- [Expression Functions](../reference/expression-functions.md)
- [API Documentation](../api/overview.md)
