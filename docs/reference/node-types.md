# Node Types Reference

Complete reference of all available node types in LinkFlow AI.

## Trigger Nodes

Trigger nodes start workflow execution.

### Manual Trigger

**Type:** `manual_trigger`

Starts workflow when manually executed.

```json
{
  "type": "manual_trigger",
  "name": "Start",
  "config": {}
}
```

**Output:**
```json
{
  "triggered": true,
  "timestamp": "2024-12-19T12:00:00Z",
  "triggeredBy": "usr-123"
}
```

---

### Schedule Trigger

**Type:** `schedule_trigger`

Executes workflow on a cron schedule.

```json
{
  "type": "schedule_trigger",
  "name": "Daily at 9 AM",
  "config": {
    "cron": "0 9 * * *",
    "timezone": "America/New_York"
  }
}
```

**Config Options:**

| Option | Type | Description |
|--------|------|-------------|
| `cron` | string | Cron expression |
| `timezone` | string | IANA timezone |

**Output:**
```json
{
  "triggered": true,
  "scheduledTime": "2024-12-19T09:00:00Z",
  "executionTime": "2024-12-19T09:00:01Z"
}
```

---

### Webhook Trigger

**Type:** `webhook_trigger`

Receives incoming HTTP requests.

```json
{
  "type": "webhook_trigger",
  "name": "Receive Webhook",
  "config": {
    "method": "POST",
    "path": "/custom-path",
    "authentication": "none"
  }
}
```

**Output:**
```json
{
  "headers": {...},
  "query": {...},
  "body": {...},
  "method": "POST",
  "path": "/webhook/abc123"
}
```

---

### Interval Trigger

**Type:** `interval_trigger`

Executes at fixed time intervals.

```json
{
  "type": "interval_trigger",
  "name": "Every 5 Minutes",
  "config": {
    "interval": 5,
    "unit": "minutes"
  }
}
```

**Config Options:**

| Option | Type | Values |
|--------|------|--------|
| `interval` | integer | Interval value |
| `unit` | string | `seconds`, `minutes`, `hours`, `days` |

---

## Control Flow Nodes

### IF

**Type:** `if`

Conditional branching based on expression.

```json
{
  "type": "if",
  "name": "Check Condition",
  "config": {
    "condition": "{{ $input.value > 100 }}"
  }
}
```

**Output Ports:**
- `true` - When condition is true
- `false` - When condition is false

---

### Switch

**Type:** `switch`

Multiple condition branches.

```json
{
  "type": "switch",
  "name": "Route by Type",
  "config": {
    "mode": "expression",
    "expression": "{{ $input.type }}",
    "rules": [
      {"value": "email", "output": 0},
      {"value": "sms", "output": 1},
      {"value": "push", "output": 2}
    ],
    "fallback": 3
  }
}
```

---

### Loop

**Type:** `loop`

Iterate over array items.

```json
{
  "type": "loop",
  "name": "Process Items",
  "config": {
    "items": "{{ $input.items }}",
    "batchSize": 10,
    "parallel": true
  }
}
```

**Loop Variables:**
- `$item` - Current item
- `$index` - Current index
- `$first` - Is first item
- `$last` - Is last item

---

### Merge

**Type:** `merge`

Combine data from multiple branches.

```json
{
  "type": "merge",
  "name": "Combine Results",
  "config": {
    "mode": "append",
    "waitForAll": true
  }
}
```

**Modes:**
- `append` - Concatenate arrays
- `combine` - Merge objects
- `chooseBranch` - Pick first completed

---

### Split In Batches

**Type:** `split_in_batches`

Process items in batches.

```json
{
  "type": "split_in_batches",
  "name": "Batch Process",
  "config": {
    "batchSize": 50,
    "items": "{{ $input.records }}"
  }
}
```

---

## Data Nodes

### Set

**Type:** `set`

Set or transform data values.

```json
{
  "type": "set",
  "name": "Transform Data",
  "config": {
    "values": {
      "fullName": "{{ $input.firstName }} {{ $input.lastName }}",
      "timestamp": "{{ $now() }}",
      "isActive": true
    },
    "keepOnlySet": false
  }
}
```

---

### Code

**Type:** `code`

Execute custom JavaScript code.

```json
{
  "type": "code",
  "name": "Custom Logic",
  "config": {
    "language": "javascript",
    "code": "const items = $input.items.filter(i => i.active);\nreturn { filteredItems: items, count: items.length };"
  }
}
```

**Available Variables:**
- `$input` - Input data
- `$node` - Access other nodes' data
- `$env` - Environment variables
- `$json` - JSON utilities

---

### HTTP Request

**Type:** `http_request`

Make HTTP requests to external APIs.

```json
{
  "type": "http_request",
  "name": "API Call",
  "config": {
    "url": "https://api.example.com/data",
    "method": "POST",
    "headers": {
      "Authorization": "Bearer {{ $credentials.apiKey }}",
      "Content-Type": "application/json"
    },
    "body": {
      "query": "{{ $input.searchTerm }}"
    },
    "timeout": 30000,
    "retry": {
      "enabled": true,
      "maxRetries": 3,
      "retryOn": [429, 500, 502, 503]
    }
  }
}
```

**Output:**
```json
{
  "statusCode": 200,
  "headers": {...},
  "body": {...}
}
```

---

## Integration Nodes

### Slack

**Type:** `slack`

Send messages and interact with Slack.

```json
{
  "type": "slack",
  "name": "Send Notification",
  "config": {
    "operation": "send_message",
    "channel": "#alerts",
    "message": "New alert: {{ $input.message }}",
    "blocks": [...],
    "credential": "cred-slack-123"
  }
}
```

**Operations:**
- `send_message` - Send message to channel
- `send_dm` - Send direct message
- `update_message` - Update existing message
- `add_reaction` - Add emoji reaction
- `get_user` - Get user info
- `list_channels` - List channels

---

### Email (SMTP)

**Type:** `email`

Send emails via SMTP.

```json
{
  "type": "email",
  "name": "Send Email",
  "config": {
    "operation": "send",
    "to": "{{ $input.recipient }}",
    "subject": "Report Ready",
    "body": "<h1>Your Report</h1><p>{{ $input.summary }}</p>",
    "bodyType": "html",
    "attachments": [
      {
        "filename": "report.pdf",
        "content": "{{ $input.pdfBase64 }}"
      }
    ],
    "credential": "cred-smtp-123"
  }
}
```

---

### PostgreSQL

**Type:** `postgresql`

Execute PostgreSQL queries.

```json
{
  "type": "postgresql",
  "name": "Query Database",
  "config": {
    "operation": "select",
    "query": "SELECT * FROM users WHERE status = $1",
    "parameters": ["{{ $input.status }}"],
    "credential": "cred-pg-123"
  }
}
```

**Operations:**
- `select` - Run SELECT query
- `insert` - Insert records
- `update` - Update records
- `delete` - Delete records
- `execute` - Run raw SQL

---

### MySQL

**Type:** `mysql`

Execute MySQL queries.

```json
{
  "type": "mysql",
  "name": "MySQL Query",
  "config": {
    "operation": "select",
    "query": "SELECT * FROM orders WHERE date > ?",
    "parameters": ["{{ $input.date }}"],
    "credential": "cred-mysql-123"
  }
}
```

---

### MongoDB

**Type:** `mongodb`

Interact with MongoDB.

```json
{
  "type": "mongodb",
  "name": "Find Documents",
  "config": {
    "operation": "find",
    "collection": "users",
    "filter": {"active": true},
    "projection": {"name": 1, "email": 1},
    "limit": 100,
    "credential": "cred-mongo-123"
  }
}
```

**Operations:**
- `find` - Find documents
- `findOne` - Find single document
- `insertOne` - Insert document
- `insertMany` - Insert multiple
- `updateOne` - Update document
- `updateMany` - Update multiple
- `deleteOne` - Delete document
- `deleteMany` - Delete multiple
- `aggregate` - Aggregation pipeline

---

### GitHub

**Type:** `github`

Interact with GitHub API.

```json
{
  "type": "github",
  "name": "Create Issue",
  "config": {
    "operation": "create_issue",
    "owner": "myorg",
    "repo": "myrepo",
    "title": "{{ $input.title }}",
    "body": "{{ $input.description }}",
    "labels": ["bug"],
    "credential": "cred-github-123"
  }
}
```

**Operations:**
- `create_issue` - Create issue
- `update_issue` - Update issue
- `create_pr` - Create pull request
- `get_repo` - Get repository info
- `list_commits` - List commits
- `create_comment` - Add comment

---

### Google Sheets

**Type:** `google_sheets`

Read/write Google Sheets.

```json
{
  "type": "google_sheets",
  "name": "Append Row",
  "config": {
    "operation": "append",
    "spreadsheetId": "1abc...",
    "sheetName": "Sheet1",
    "values": [
      ["{{ $input.name }}", "{{ $input.email }}", "{{ $now() }}"]
    ],
    "credential": "cred-google-123"
  }
}
```

**Operations:**
- `read` - Read range
- `append` - Append rows
- `update` - Update cells
- `clear` - Clear range

---

### S3

**Type:** `s3`

Interact with AWS S3.

```json
{
  "type": "s3",
  "name": "Upload File",
  "config": {
    "operation": "upload",
    "bucket": "my-bucket",
    "key": "uploads/{{ $input.filename }}",
    "body": "{{ $input.content }}",
    "contentType": "application/json",
    "credential": "cred-aws-123"
  }
}
```

**Operations:**
- `upload` - Upload object
- `download` - Download object
- `delete` - Delete object
- `list` - List objects
- `copy` - Copy object

---

## Utility Nodes

### Wait

**Type:** `wait`

Pause execution for specified duration.

```json
{
  "type": "wait",
  "name": "Wait 5 seconds",
  "config": {
    "duration": 5000,
    "unit": "milliseconds"
  }
}
```

---

### No-Op

**Type:** `noop`

Pass-through node (useful for debugging).

```json
{
  "type": "noop",
  "name": "Debug Point",
  "config": {}
}
```

---

### Stop and Error

**Type:** `stop_and_error`

Stop execution with error.

```json
{
  "type": "stop_and_error",
  "name": "Validation Failed",
  "config": {
    "message": "Invalid input: {{ $input.error }}",
    "errorType": "VALIDATION_ERROR"
  }
}
```

---

### Error Trigger

**Type:** `error_trigger`

Triggered when workflow encounters error.

```json
{
  "type": "error_trigger",
  "name": "Handle Error",
  "config": {}
}
```

**Output:**
```json
{
  "error": {
    "message": "Error message",
    "nodeId": "node-that-failed",
    "timestamp": "..."
  },
  "execution": {
    "id": "exec-123",
    "workflowId": "wf-789"
  }
}
```

## Next Steps

- [Expression Functions](expression-functions.md)
- [Workflows API](../api/workflows.md)
- [Adding Custom Nodes](../guides/adding-nodes.md)
