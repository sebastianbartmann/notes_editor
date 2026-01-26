# LinkedIn Tools

## Purpose

LinkedIn Tools enables automated interaction with LinkedIn through the Claude agent. It provides OAuth2 authentication for secure API access and exposes MCP (Model Context Protocol) tools that allow Claude to create posts, read comments, and post replies on behalf of the authenticated user. All actions are logged to a person-scoped CSV file for audit and reference.

## Architecture

| Component | Description |
|-----------|-------------|
| `linkedin_service.py` | Core service handling OAuth, API calls, and CSV logging |
| `linkedin_tools.py` | MCP tool definitions exposed to Claude agent |
| `.env` file | Stores LinkedIn credentials and access token |
| `{person}/linkedin/posts.csv` | Person-scoped activity log in the vault |

## OAuth2 Authentication Flow

### Prerequisites

The following environment variables must be configured in `server/web_app/.env`:

| Variable | Description |
|----------|-------------|
| `LINKEDIN_CLIENT_ID` | LinkedIn app client ID |
| `LINKEDIN_CLIENT_SECRET` | LinkedIn app client secret |
| `LINKEDIN_REDIRECT_URI` | OAuth callback URL (must match app config) |
| `LINKEDIN_ACCESS_TOKEN` | Stored after successful OAuth (auto-populated) |

### Authorization Flow

1. User initiates OAuth by visiting LinkedIn's authorization URL with the app's client ID and redirect URI
2. After user grants permission, LinkedIn redirects to `/api/linkedin/oauth/callback` with an authorization code
3. The callback endpoint exchanges the code for an access token via LinkedIn's token endpoint
4. The access token is persisted to the `.env` file and loaded into the environment
5. Subsequent API calls use the stored token

### API Endpoint

```
GET /api/linkedin/oauth/callback?code={authorization_code}
```

**Response (success):**
```json
{
  "success": true,
  "expires_in": 5183999
}
```

**Response (error):**
```json
{
  "detail": "LinkedIn token exchange failed: 400 {...}"
}
```

**Note:** This endpoint is exempt from authentication to allow the OAuth callback to complete.

## LinkedIn API Integration

### API Configuration

- **Base URL:** `https://api.linkedin.com`
- **API Version:** v2 endpoints (UGC Posts, Social Actions)
- **Authentication:** Bearer token in Authorization header
- **Content-Type:** `application/json`

### Core API Functions

| Function | Endpoint | Description |
|----------|----------|-------------|
| `get_person_urn()` | `GET /v2/userinfo` | Retrieves authenticated user's person URN |
| `create_post()` | `POST /v2/ugcPosts` | Creates a public text post |
| `read_comments()` | `GET /v2/socialActions/{urn}/comments` | Fetches comments on a post |
| `create_comment()` | `POST /v2/socialActions/{urn}/comments` | Posts a comment or reply |

## MCP Tools

The LinkedIn tools are exposed via an MCP server (`LINKEDIN_MCP_SERVER`) created with `create_sdk_mcp_server("linkedin", tools=LINKEDIN_TOOLS)`. These tools are available to the Claude agent during chat sessions.

### Person Context

Tools require a person context set via `set_current_person(person)` before execution. This ensures activity is logged to the correct person's folder. The context is managed using Python's `contextvars` module.

### Available Tools

#### `linkedin_post`

Create a new LinkedIn post.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `text` | string | Yes | The post content |

**Returns:** Post URN on success, error message on failure

---

#### `linkedin_read_comments`

Read all comments on a LinkedIn post.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `post_urn` | string | Yes | The URN of the post (e.g., `urn:li:ugcPost:123456`) |

**Returns:** JSON response containing comments array

---

#### `linkedin_post_comment`

Post a top-level comment on a LinkedIn post.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `post_urn` | string | Yes | The URN of the post to comment on |
| `text` | string | Yes | The comment content |

**Returns:** Comment URN on success, error message on failure

---

#### `linkedin_reply_comment`

Reply to an existing comment on a LinkedIn post.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `post_urn` | string | Yes | The URN of the post |
| `comment_urn` | string | Yes | The URN of the parent comment to reply to |
| `text` | string | Yes | The reply content |

**Returns:** Reply URN on success, error message on failure

## Activity Logging

All write actions (posts, comments, replies) are logged to a CSV file in the vault.

### Log Location

```
~/notes/{person}/linkedin/posts.csv
```

The directory is created automatically if it doesn't exist.

### CSV Format

| Column | Description |
|--------|-------------|
| `timestamp` | Action timestamp (`YYYY-MM-DD HH:MM:SS`) |
| `action` | Action type: `post`, `comment`, or `reply` |
| `post_urn` | URN of the post (populated for all actions) |
| `comment_urn` | URN of the comment (for replies, the parent comment) |
| `text` | The content that was posted |
| `response` | Full API response as JSON string |

### Example Log Entry

```csv
timestamp,action,post_urn,comment_urn,text,response
2026-01-15 14:30:00,post,urn:li:ugcPost:7123456789,,My LinkedIn post content,"{""id"":""urn:li:ugcPost:7123456789""}"
2026-01-15 14:35:00,comment,urn:li:ugcPost:7123456789,urn:li:comment:123,Great insight!,"{""id"":""urn:li:comment:123""}"
2026-01-15 14:40:00,reply,urn:li:ugcPost:7123456789,urn:li:comment:123,Thanks for your feedback,"{""id"":""urn:li:comment:456""}"
```

## Error Handling

- **Missing environment variables:** Raises `ValueError` with the missing variable name
- **API errors:** Raises `RuntimeError` with status code and response body; errors are logged to stdout
- **Tool errors:** Return structured error response with `is_error: true` flag

## Security Considerations

- Access token is stored in `.env` file (not in vault or version control)
- OAuth callback endpoint is exempt from normal authentication to allow the flow to complete
- All API requests use HTTPS with 30-second timeout
- Person context prevents cross-user logging
