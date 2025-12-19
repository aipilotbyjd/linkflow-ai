# Authentication API

## Overview

LinkFlow AI supports multiple authentication methods:
- JWT tokens for user authentication
- API keys for programmatic access
- OAuth2 for third-party integrations

## User Registration

### POST /api/v1/auth/register

Create a new user account.

**Request:**
```json
{
  "email": "user@example.com",
  "password": "SecurePassword123!",
  "name": "John Doe"
}
```

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "usr-123456",
      "email": "user@example.com",
      "name": "John Doe",
      "emailVerified": false,
      "createdAt": "2024-12-19T12:00:00Z"
    },
    "message": "Verification email sent"
  }
}
```

**Validation Rules:**
- Email: Valid email format, unique
- Password: Minimum 8 characters, uppercase, lowercase, number
- Name: 2-100 characters

## User Login

### POST /api/v1/auth/login

Authenticate and receive JWT tokens.

**Request:**
```json
{
  "email": "user@example.com",
  "password": "SecurePassword123!"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "accessToken": "eyJhbGciOiJIUzI1NiIs...",
    "refreshToken": "eyJhbGciOiJIUzI1NiIs...",
    "expiresIn": 86400,
    "tokenType": "Bearer",
    "user": {
      "id": "usr-123456",
      "email": "user@example.com",
      "name": "John Doe"
    }
  }
}
```

**Error Response (401 Unauthorized):**
```json
{
  "success": false,
  "error": {
    "code": "INVALID_CREDENTIALS",
    "message": "Invalid email or password"
  }
}
```

## Token Refresh

### POST /api/v1/auth/refresh

Get a new access token using refresh token.

**Request:**
```json
{
  "refreshToken": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "accessToken": "eyJhbGciOiJIUzI1NiIs...",
    "expiresIn": 86400
  }
}
```

## Logout

### POST /api/v1/auth/logout

Invalidate current session.

**Headers:**
```
Authorization: Bearer <access-token>
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "message": "Logged out successfully"
  }
}
```

## Password Reset

### POST /api/v1/auth/forgot-password

Request password reset email.

**Request:**
```json
{
  "email": "user@example.com"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "message": "If the email exists, a reset link has been sent"
  }
}
```

### POST /api/v1/auth/reset-password

Reset password using token from email.

**Request:**
```json
{
  "token": "reset-token-from-email",
  "password": "NewSecurePassword123!"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "message": "Password reset successfully"
  }
}
```

## Email Verification

### POST /api/v1/auth/verify-email

Verify email address.

**Request:**
```json
{
  "token": "verification-token-from-email"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "message": "Email verified successfully"
  }
}
```

### POST /api/v1/auth/resend-verification

Resend verification email.

**Headers:**
```
Authorization: Bearer <access-token>
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "message": "Verification email sent"
  }
}
```

## API Keys

### GET /api/v1/api-keys

List user's API keys.

**Response:**
```json
{
  "success": true,
  "data": {
    "apiKeys": [
      {
        "id": "key-123",
        "name": "Production Key",
        "prefix": "lf_live_abc",
        "scopes": ["workflows:read", "workflows:write", "executions:read"],
        "lastUsed": "2024-12-19T12:00:00Z",
        "createdAt": "2024-12-01T00:00:00Z"
      }
    ]
  }
}
```

### POST /api/v1/api-keys

Create new API key.

**Request:**
```json
{
  "name": "Production Key",
  "scopes": ["workflows:read", "workflows:write", "executions:read"],
  "expiresAt": "2025-12-31T23:59:59Z"
}
```

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "apiKey": {
      "id": "key-123",
      "name": "Production Key",
      "key": "lf_live_abcdefghijklmnop",
      "scopes": ["workflows:read", "workflows:write", "executions:read"],
      "expiresAt": "2025-12-31T23:59:59Z"
    },
    "warning": "Save this key now. It won't be shown again."
  }
}
```

### DELETE /api/v1/api-keys/{id}

Revoke API key.

**Response (204 No Content)**

## Available Scopes

| Scope | Description |
|-------|-------------|
| `workflows:read` | Read workflow definitions |
| `workflows:write` | Create/update/delete workflows |
| `executions:read` | Read execution history |
| `executions:write` | Execute workflows |
| `credentials:read` | Read credential metadata |
| `credentials:write` | Create/update/delete credentials |
| `admin` | Full administrative access |

## OAuth2 Providers

### GET /api/v1/auth/oauth/{provider}

Initiate OAuth flow. Supported providers: `google`, `github`, `microsoft`

**Response (302 Redirect):**
Redirects to provider's authorization page.

### GET /api/v1/auth/oauth/{provider}/callback

OAuth callback handler. Exchanges code for tokens.

**Query Parameters:**
- `code`: Authorization code from provider
- `state`: State parameter for CSRF protection

**Response (302 Redirect):**
Redirects to frontend with tokens or error.

## JWT Token Structure

Access tokens contain:

```json
{
  "userId": "usr-123456",
  "email": "user@example.com",
  "roles": ["user"],
  "tenantId": "ws-789",
  "iat": 1703001600,
  "exp": 1703088000
}
```

## Security Best Practices

1. **Store tokens securely**: Use httpOnly cookies or secure storage
2. **Refresh before expiry**: Implement token refresh logic
3. **Use appropriate scopes**: Request minimum required permissions
4. **Rotate API keys**: Regularly rotate production keys
5. **Monitor usage**: Check for unusual API key usage

## Next Steps

- [Workflows API](workflows.md)
- [Executions API](executions.md)
- [API Overview](overview.md)
