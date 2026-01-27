# Authentication and Context Management Specification

> Status: Draft
> Version: 2.0
> Last Updated: 2026-01-27

## Overview

This document defines the authentication, authorization, and context management systems for the Notes Editor application. It covers bearer token authentication, cookie-based session management, person context resolution, theme preferences, path utilities for multi-user file access, and environment configuration management.

The authentication and context systems serve as foundational dependencies for all other application specifications.

**Implementation:** Go backend (`server/internal/auth/`), React context providers (`clients/web/src/context/`)

---

## Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NOTES_TOKEN` | No | `VJY9EoAf1xx1bO-LaduCmItwRitCFm9BPuQZ8jd0tcg` | Bearer token for API authentication |

### Constants

```go
var (
    Persons = map[string]bool{"sebastian": true, "petra": true}
    Themes  = map[string]bool{"dark": true, "light": true}
)
```

### Cookies

| Cookie Name | Purpose | Max Age | Attributes |
|-------------|---------|---------|------------|
| `notes_token` | Session authentication | Session | `httponly=True`, `samesite=lax` |
| `notes_person` | Person context persistence | 1 year | `samesite=lax` |
| `notes_theme` | Theme preference persistence | 1 year | `samesite=lax` |

### Request Headers

| Header | Purpose | Required |
|--------|---------|----------|
| `Authorization` | Bearer token authentication | No (if cookie present) |
| `X-Notes-Person` | Person context override | No (falls back to cookie) |

---

## Authentication

The application uses a global FastAPI dependency to enforce authentication on all routes except explicitly exempted paths.

### Security Configuration

```python
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials

security = HTTPBearer(auto_error=False)
```

**Behavior**: `auto_error=False` allows the middleware to handle missing credentials gracefully rather than raising automatic 401 errors.

### Authentication Middleware

#### `require_auth`

**Signature**:
```python
def require_auth(
    request: Request,
    credentials: HTTPAuthorizationCredentials = Depends(security),
) -> None
```

**Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| `request` | `Request` | FastAPI request object |
| `credentials` | `HTTPAuthorizationCredentials` | Extracted bearer token (optional) |

**Returns**: `None` on success

**Raises**: `HTTPException(status_code=401, detail="Unauthorized")` on failure

**Authentication Flow**:

1. **Path Exemption Check**: If request path starts with `/login` or `/api/linkedin/oauth/callback`, authentication is bypassed.

2. **Bearer Token Validation**: If `Authorization: Bearer <token>` header is present:
   - Verify scheme is `bearer` (case-insensitive)
   - Compare token against `NOTES_TOKEN` using `secrets.compare_digest()` (constant-time comparison)
   - Return on match

3. **Cookie Token Validation**: Check `notes_token` cookie:
   - Compare cookie value against `NOTES_TOKEN` using `secrets.compare_digest()`
   - Return on match

4. **Rejection**: If no valid authentication method succeeds, raise HTTP 401.

**Security Features**:
- Constant-time string comparison prevents timing attacks
- Supports both header-based (API clients) and cookie-based (web browser) authentication
- Exempted paths are checked first to avoid unnecessary processing

### Global Application Dependency

```python
app = FastAPI(dependencies=[Depends(require_auth)])
```

All routes inherit this authentication requirement unless explicitly exempted by path matching in the middleware.

---

## Login Route

### `POST /login`

**Purpose**: Authenticate user and establish cookie-based session.

**Authentication**: Not required (exempted path)

**Request Body** (form-encoded):
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `token` | string | Yes | Authentication token to validate |

**Response (302)**: Redirect to `/` with session cookie set.

**Response (401)**:
```
Invalid token
```

**Behavior**:
1. Validate token against `NOTES_TOKEN` using `secrets.compare_digest()`
2. If invalid, return HTML response with "Invalid token" and status 401
3. If valid:
   - Create `RedirectResponse` to `/` with status 302
   - Set `notes_token` cookie with validated token
   - Cookie attributes: `httponly=True`, `samesite=lax`

**Cookie Security**:
- `httponly=True`: Prevents JavaScript access (XSS mitigation)
- `samesite=lax`: CSRF protection while allowing normal navigation

---

## Person Context Management

Person context determines which user's data is accessed. It scopes all file operations to a specific person's directory within the vault.

### `get_person`

**Signature**:
```python
def get_person(request: Request) -> str | None
```

**Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| `request` | `Request` | FastAPI request object |

**Returns**: Lowercase person identifier (`"sebastian"` or `"petra"`) or `None` if not set/invalid.

**Resolution Priority**:
1. `X-Notes-Person` header
2. `notes_person` cookie
3. `None` (no valid person found)

**Behavior**:
- Extract candidate from header or cookie
- Strip whitespace and convert to lowercase
- Validate against `PERSONS` set
- Return validated person or `None`

---

### `ensure_person_or_redirect`

**Signature**:
```python
def ensure_person_or_redirect(request: Request) -> str | RedirectResponse
```

**Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| `request` | `Request` | FastAPI request object |

**Returns**: Person identifier string or `RedirectResponse` to `/settings`.

**Behavior**:
- Call `get_person(request)` to resolve person context
- If person is valid, return the person identifier
- If person is `None` or invalid, return `RedirectResponse("/settings")`

**Use Case**: Web routes that need graceful handling of missing person context by redirecting to settings page.

---

### `ensure_person`

**Signature**:
```python
def ensure_person(request: Request) -> str
```

**Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| `request` | `Request` | FastAPI request object |

**Returns**: Person identifier string.

**Raises**: `HTTPException(status_code=400, detail="Person not selected")` if no valid person.

**Behavior**:
- Call `get_person(request)` to resolve person context
- If person is valid, return the person identifier
- If person is `None`, raise HTTP 400 error

**Use Case**: API routes where person context is mandatory and missing context should result in an error response.

---

## Theme Management

### `get_theme`

**Signature**:
```python
def get_theme(request: Request) -> str
```

**Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| `request` | `Request` | FastAPI request object |

**Returns**: Theme identifier (`"dark"` or `"light"`).

**Behavior**:
- Read `notes_theme` cookie value
- Strip whitespace and convert to lowercase
- If value is in `THEMES` set, return it
- Default to `"dark"` if cookie is missing or invalid

**Default**: `"dark"`

---

## Settings Routes

### `POST /settings`

**Purpose**: Update person and/or theme preferences.

**Authentication**: Required

**Request Body** (form-encoded):
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `person` | string | No | Person identifier to set |
| `theme` | string | No | Theme preference to set |

**Response (302)**: Redirect to `/settings` with updated cookies.

**Response (400)**:
```
Invalid person
```
or
```
Invalid theme
```

**Behavior**:
1. Create `RedirectResponse` to `/settings` with status 302
2. If `person` is provided:
   - Normalize: strip whitespace, convert to lowercase
   - Validate against `PERSONS` set
   - If invalid, return HTML response with "Invalid person" and status 400
   - Set `notes_person` cookie with `max_age=31536000` (1 year), `samesite=lax`
3. If `theme` is provided:
   - Normalize: strip whitespace, convert to lowercase
   - Validate against `THEMES` set
   - If invalid, return HTML response with "Invalid theme" and status 400
   - Set `notes_theme` cookie with `max_age=31536000` (1 year), `samesite=lax`
4. Return redirect response

---

## Path Utilities

These utilities manage path translation between person-relative paths and the filesystem structure.

### Directory Structure

```
VAULT_ROOT/
  sebastian/
    daily/
    notes/
    ...
  petra/
    daily/
    notes/
    ...
```

### `person_root_path`

**Signature**:
```python
def person_root_path(person: str) -> Path
```

**Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| `person` | `str` | Person identifier |

**Returns**: `Path` object pointing to person's root directory.

**Behavior**: Returns `VAULT_ROOT / person`.

**Example**:
```python
person_root_path("sebastian")  # -> Path("/vault/sebastian")
```

---

### `person_relative_path`

**Signature**:
```python
def person_relative_path(person: str, relative_path: str) -> str
```

**Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| `person` | `str` | Person identifier |
| `relative_path` | `str` | Path relative to person root (or absolute from vault) |

**Returns**: Normalized path including person prefix.

**Behavior**:
1. Strip whitespace and leading slashes from `relative_path`
2. If normalized path is empty or `"."`, return person identifier only
3. If path already starts with `{person}/` or equals `person`, return as-is
4. Otherwise, prepend `{person}/` to the path

**Examples**:
```python
person_relative_path("sebastian", "")           # -> "sebastian"
person_relative_path("sebastian", ".")          # -> "sebastian"
person_relative_path("sebastian", "daily")      # -> "sebastian/daily"
person_relative_path("sebastian", "/daily")     # -> "sebastian/daily"
person_relative_path("sebastian", "sebastian/daily")  # -> "sebastian/daily"
```

---

### `strip_person_path`

**Signature**:
```python
def strip_person_path(person: str, relative_path: str) -> str
```

**Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| `person` | `str` | Person identifier |
| `relative_path` | `str` | Path that may include person prefix |

**Returns**: Path with person prefix removed.

**Behavior**:
1. If `relative_path` equals `person`, return `"."`
2. If `relative_path` starts with `{person}/`, strip that prefix
3. Otherwise, return path unchanged

**Examples**:
```python
strip_person_path("sebastian", "sebastian")       # -> "."
strip_person_path("sebastian", "sebastian/daily") # -> "daily"
strip_person_path("sebastian", "other/path")      # -> "other/path"
```

---

## Environment Configuration

### File Location

The `.env` file is located at `BASE_DIR / ".env"` where `BASE_DIR` is the application base directory.

### `read_env_content`

**Signature**:
```python
def read_env_content() -> str
```

**Returns**: Contents of `.env` file as string, or empty string if file doesn't exist.

**Behavior**:
- Check if `.env` file exists at `BASE_DIR / ".env"`
- If exists, read and return contents
- If not exists, return empty string

---

### `write_env_content`

**Signature**:
```python
def write_env_content(env_content: str) -> None
```

**Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| `env_content` | `str` | New environment file content |

**Returns**: `None`

**Behavior**:
1. **Normalize line endings**: Convert `\r\n` (Windows) to `\n` (Unix)
2. **Ensure trailing newline**: If content is non-empty and doesn't end with `\n`, append one
3. **Create parent directories**: Ensure `BASE_DIR` exists
4. **Write file**: Write normalized content to `BASE_DIR / ".env"`
5. **Reload environment**: Call `load_dotenv(env_path, override=True)` to apply changes immediately

**Side Effects**:
- Creates parent directories if needed
- Modifies filesystem
- Updates process environment variables via `load_dotenv`

---

## Security Considerations

### Token Security

1. **Constant-Time Comparison**: All token comparisons use `secrets.compare_digest()` to prevent timing attacks that could leak token information.

2. **Default Token Warning**: The default `NOTES_TOKEN` value should be overridden in production. Applications should log a warning if the default token is in use.

3. **Token Storage**: Tokens should be stored securely and not committed to version control.

### Cookie Security

1. **HttpOnly Flag**: The `notes_token` cookie uses `httponly=True` to prevent JavaScript access, mitigating XSS attacks.

2. **SameSite Attribute**: All cookies use `samesite=lax` to provide CSRF protection while allowing normal navigation.

3. **No Secure Flag**: Currently cookies don't use the `secure` flag, meaning they're transmitted over HTTP. For production over HTTPS, consider adding `secure=True`.

### Path Traversal Prevention

1. **Person Scoping**: All file operations are scoped to a person's directory, preventing access to other users' data.

2. **Path Normalization**: Leading slashes are stripped to prevent absolute path injection.

3. **No `..` Handling**: The current implementation doesn't explicitly block `..` path traversal. This should be handled at the file operation level.

### Environment File Security

1. **Sensitive Data**: The `.env` file may contain API keys and secrets. Access to environment configuration endpoints should be restricted.

2. **Hot Reload**: `load_dotenv(override=True)` immediately applies changes to the running process, which could affect security if malicious values are written.

---

## Integration Notes

This specification provides foundational services used by other specifications:

### Dependent Specifications

| Specification | Dependencies |
|---------------|--------------|
| **01-rest-api-contract.md** | `require_auth`, `ensure_person`, bearer token validation |
| **02-daily-notes.md** | `person_root_path`, `person_relative_path`, person context |
| **03-file-management.md** | Path utilities, person context for scoped file access |
| **04-claude-chat.md** | Authentication middleware, person context |
| **05-linkedin-service.md** | OAuth callback path exemption, environment config |
| **06-sleep-times.md** | `person_root_path`, person context |
| **07-todos.md** | Person context, path utilities |

### Common Patterns

**API Route with Person Context**:
```python
@app.get("/api/example")
async def example_endpoint(
    request: Request,
    person: str = Depends(ensure_person),
):
    root = person_root_path(person)
    # ... scoped file operations
```

**Web Route with Redirect on Missing Person**:
```python
@app.get("/example")
async def example_page(request: Request):
    result = ensure_person_or_redirect(request)
    if isinstance(result, RedirectResponse):
        return result
    person = result
    # ... render page
```

**Theme-Aware Template Rendering**:
```python
@app.get("/page")
async def page(request: Request):
    theme = get_theme(request)
    return templates.TemplateResponse(
        "page.html",
        {"request": request, "theme": theme}
    )
```

---

## Error Responses

### Authentication Errors

| Status | Detail | Cause |
|--------|--------|-------|
| 401 | `"Unauthorized"` | Missing or invalid authentication |
| 401 | `"Invalid token"` | Wrong token at login |

### Person Context Errors

| Status | Detail | Cause |
|--------|--------|-------|
| 400 | `"Person not selected"` | Missing or invalid person context on required endpoint |
| 400 | `"Invalid person"` | Invalid person value in settings form |

### Settings Errors

| Status | Detail | Cause |
|--------|--------|-------|
| 400 | `"Invalid person"` | Person value not in `PERSONS` set |
| 400 | `"Invalid theme"` | Theme value not in `THEMES` set |
