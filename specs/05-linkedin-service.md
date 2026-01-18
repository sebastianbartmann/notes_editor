# LinkedIn Service Specification

> Status: Draft
> Version: 1.0
> Last Updated: 2026-01-18

## Overview

This document specifies the LinkedIn API integration for the Notes Editor application. The service provides:

1. **OAuth Authentication**: Token exchange and persistence for LinkedIn API access
2. **Content Operations**: Create posts, read comments, post comments and replies
3. **Activity Logging**: CSV-based logging of all LinkedIn activity per person

The LinkedIn service is used by Claude's MCP tools to enable AI-assisted LinkedIn posting and engagement.

---

## Configuration

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `LINKEDIN_CLIENT_ID` | Yes | OAuth application client ID |
| `LINKEDIN_CLIENT_SECRET` | Yes | OAuth application secret |
| `LINKEDIN_REDIRECT_URI` | Yes | OAuth callback URL |
| `LINKEDIN_ACCESS_TOKEN` | Yes* | Current access token (*set after OAuth flow) |

### File Locations

| Path | Description |
|------|-------------|
| `server/web_app/.env` | Environment file storing credentials |
| `{VAULT_ROOT}/{person}/linkedin/posts.csv` | Activity log per person |

### Constants

```python
LINKEDIN_API_BASE = "https://api.linkedin.com"
LINKEDIN_VERSION = ""  # No version header (uses v2 endpoints)
```

---

## OAuth Flow

### Authorization Flow

1. User initiates OAuth via LinkedIn authorization URL
2. LinkedIn redirects to callback with authorization code
3. Server exchanges code for access token
4. Token is persisted to `.env` and runtime environment

### Token Exchange

**Endpoint:** `POST https://www.linkedin.com/oauth/v2/accessToken`

```python
def exchange_code_for_token(auth_code: str) -> dict:
    # POST with form data:
    # - grant_type: "authorization_code"
    # - code: auth_code
    # - redirect_uri: LINKEDIN_REDIRECT_URI
    # - client_id: LINKEDIN_CLIENT_ID
    # - client_secret: LINKEDIN_CLIENT_SECRET
    #
    # Returns: {"access_token": "...", "expires_in": 3600, ...}
```

**Response:**
```json
{
  "access_token": "AQV...",
  "expires_in": 5184000,
  "scope": "r_liteprofile w_member_social",
  "token_type": "Bearer"
}
```

### Token Persistence

```python
def persist_access_token(access_token: str) -> None:
    # Updates LINKEDIN_ACCESS_TOKEN in .env file
    # Sets os.environ["LINKEDIN_ACCESS_TOKEN"]
```

The `update_env_value` helper handles `.env` file modification, preserving comments and other variables.

### REST API Callback

**Endpoint:** `GET /api/linkedin/oauth/callback`

**Query Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `code` | string | Yes | Authorization code from LinkedIn |

**Response (200):**
```json
{
  "success": true,
  "expires_in": 5184000
}
```

**Error Response (400):**
```json
{
  "detail": "LinkedIn token exchange failed: 400 ..."
}
```

**Note:** This endpoint is exempt from bearer token authentication.

---

## API Client

### Request Headers

```python
def get_headers(token: str) -> dict:
    return {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json",
    }
```

No LinkedIn-Versioning header is used; the service relies on v2 endpoint defaults.

### Generic Request Handler

```python
def _request(method: str, url: str, token: str, **kwargs) -> requests.Response:
    # Sends HTTP request with:
    # - Authorization and Content-Type headers
    # - 30-second timeout
    # - Error logging and RuntimeError on failure
```

**Error Handling:**
- Non-2xx responses log to stdout and raise `RuntimeError`
- Error message includes status code and response body

### Get Person URN

Retrieves the authenticated user's LinkedIn person URN.

**Endpoint:** `GET /v2/userinfo`

```python
def get_person_urn(token: str) -> str:
    # Returns: "urn:li:person:{sub}"
```

**Response:**
```json
{
  "sub": "abc123xyz",
  "name": "John Doe",
  "email": "john@example.com"
}
```

---

## Content Operations

### Create Post

Creates a new LinkedIn post as the authenticated user.

```python
def create_post(text: str) -> dict
```

**Endpoint:** `POST /v2/ugcPosts`

**Payload:**
```json
{
  "author": "urn:li:person:{id}",
  "lifecycleState": "PUBLISHED",
  "specificContent": {
    "com.linkedin.ugc.ShareContent": {
      "shareCommentary": {"text": "Post content here"},
      "shareMediaCategory": "NONE"
    }
  },
  "visibility": {"com.linkedin.ugc.MemberNetworkVisibility": "PUBLIC"}
}
```

**Response:**
```json
{
  "id": "urn:li:share:123456789"
}
```

### Read Comments

Retrieves comments for a LinkedIn post.

```python
def read_comments(post_urn: str) -> dict
```

**Endpoint:** `GET /v2/socialActions/{encoded_urn}/comments`

**Note:** The post URN is URL-encoded (e.g., `urn:li:share:123` becomes `urn%3Ali%3Ashare%3A123`).

**Response:**
```json
{
  "elements": [
    {
      "id": "urn:li:comment:456",
      "actor": "urn:li:person:abc",
      "message": {"text": "Great post!"},
      "created": {"time": 1705580400000}
    }
  ],
  "paging": {"count": 10, "start": 0}
}
```

### Create Comment

Posts a comment on a LinkedIn post. Supports both top-level comments and replies.

```python
def create_comment(
    post_urn: str,
    text: str,
    parent_comment_urn: str | None = None
) -> dict
```

**Endpoint:** `POST /v2/socialActions/{encoded_urn}/comments`

**Payload (top-level comment):**
```json
{
  "actor": "urn:li:person:{id}",
  "message": {"text": "Comment text here"}
}
```

**Payload (reply to comment):**
```json
{
  "actor": "urn:li:person:{id}",
  "message": {"text": "Reply text here"},
  "parentComment": "urn:li:comment:456"
}
```

**Response:**
```json
{
  "id": "urn:li:comment:789"
}
```

---

## Activity Logging

### Log Format

Activity is logged to CSV files at `{VAULT_ROOT}/{person}/linkedin/posts.csv`.

**CSV Headers:**
```
timestamp,action,post_urn,comment_urn,text,response
```

**Fields:**
| Field | Description |
|-------|-------------|
| `timestamp` | ISO format datetime (YYYY-MM-DD HH:MM:SS) |
| `action` | One of: `post`, `comment`, `reply` |
| `post_urn` | LinkedIn post URN |
| `comment_urn` | LinkedIn comment URN (for replies) |
| `text` | Content that was posted |
| `response` | JSON-encoded API response |

### Logging Functions

```python
def log_post(person: str, text: str, response: dict) -> None:
    # Logs action="post" with post_urn from response

def log_comment(
    person: str,
    text: str,
    response: dict,
    post_urn: str,
    comment_urn: str | None = None,
    action: str = "comment"
) -> None:
    # Logs action="comment" or "reply"
    # comment_urn defaults to response["id"] if not provided
```

### Log File Creation

- Parent directories are created automatically
- CSV header row is written only for new files
- Logs are appended without locking (single-server deployment)

---

## Error Handling

### Environment Errors

| Error | Cause | Handling |
|-------|-------|----------|
| `ValueError` | Missing required env variable | Raised by `_require_env()` |

### OAuth Errors

| Error | Cause | Handling |
|-------|-------|----------|
| `RuntimeError` | Token exchange failed | HTTP 400 response |
| `RuntimeError` | Missing access_token in response | HTTP 400 response |

### API Errors

| Error | Cause | Handling |
|-------|-------|----------|
| `RuntimeError` | Non-2xx API response | Logged and raised |
| `RuntimeError` | Missing 'sub' in userinfo | Raised |
| `requests.Timeout` | Request exceeded 30s | Raised |

### MCP Tool Error Responses

LinkedIn MCP tools return structured errors rather than raising exceptions:

```json
{
  "content": [{"type": "text", "text": "Error: text is required"}],
  "is_error": true
}
```

---

## Integration Notes

### MCP Tools Integration

The LinkedIn service is wrapped by MCP tools in `linkedin_tools.py`:

| MCP Tool | Service Function |
|----------|------------------|
| `linkedin_post` | `create_post()` + `log_post()` |
| `linkedin_read_comments` | `read_comments()` |
| `linkedin_post_comment` | `create_comment()` + `log_comment()` |
| `linkedin_reply_comment` | `create_comment(parent_comment_urn=...)` + `log_comment()` |

### Person Context

MCP tools manage person context via `contextvars`:
- Context is set before Claude queries
- All logging functions receive `person` parameter from context
- Context is reset after query completes

### Vault Integration

Activity logs are stored in the person's vault directory:
```
~/notes/{person}/linkedin/posts.csv
```

This allows logs to be version-controlled with the user's other notes.

### Related Specifications

- [01-rest-api-contract.md](./01-rest-api-contract.md) - OAuth callback endpoint
- [02-vault-storage-git-sync.md](./02-vault-storage-git-sync.md) - VAULT_ROOT and storage paths
- [04-claude-service.md](./04-claude-service.md) - MCP tools and Claude integration
