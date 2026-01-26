# Authentication & Authorization

## Purpose

Notes Editor uses a simple bearer token authentication system to protect all API endpoints and web pages. A single shared token (`NOTES_TOKEN`) grants full access to the application. This straightforward approach is appropriate for a personal/family application with trusted users.

## Token Configuration

### Server-Side

**Environment Variable:** `NOTES_TOKEN`

The token is loaded from the server's `.env` file at startup:

**Location:** `server/web_app/.env`

```
NOTES_TOKEN=your-secret-token-here
```

If the environment variable is not set, a default fallback token is used (defined in `main.py`). The `.env` file can be edited through the web UI settings page (`/settings`).

### Android Client

The token is hardcoded in the Android app configuration:

**Location:** `app/android/app/src/main/java/com/bartmann/noteseditor/AppConfig.kt`

```kotlin
object AppConfig {
    const val AUTH_TOKEN = "your-secret-token-here"
}
```

The Android app must be rebuilt with the correct token to authenticate successfully.

## Authentication Flow

### Request Validation

The `require_auth` dependency is applied globally to the FastAPI application and checks every incoming request:

```python
app = FastAPI(dependencies=[Depends(require_auth)])
```

The validation logic checks credentials in this order:

1. **Check excluded routes** - Some paths bypass authentication (see below)
2. **Check Authorization header** - Look for `Bearer <token>` format
3. **Check cookie** - Look for `notes_token` cookie
4. **Reject if neither valid** - Return HTTP 401 Unauthorized

### Token Comparison

All token comparisons use `secrets.compare_digest()` for timing-safe string comparison, preventing timing attacks:

```python
if secrets.compare_digest(credentials.credentials, NOTES_TOKEN):
    return  # authenticated
```

### Web Login Flow

1. User visits any protected page without valid auth
2. Request is rejected with 401 Unauthorized
3. User navigates to `/login` and enters the token
4. POST `/login` validates the token
5. On success, sets `notes_token` cookie and redirects to `/`
6. Subsequent requests authenticate via the cookie

### Android Authentication

The Android client sends the token with every request:

```kotlin
private val authHeader = "Bearer ${AppConfig.AUTH_TOKEN}"
```

All API calls include this header:

```kotlin
Request.Builder()
    .url("$baseUrl$path")
    .header("Authorization", authHeader)
    // ...
```

## Login Endpoints

### GET /login

Returns a simple HTML login form.

**Response:** HTML page with token input field and submit button.

No styling or template - uses inline HTML for simplicity.

### POST /login

Validates the submitted token and sets the authentication cookie.

**Form Parameters:**
| Parameter | Required | Description |
|-----------|----------|-------------|
| `token` | Yes | The access token to validate |

**Success Response:**
- HTTP 302 redirect to `/`
- Sets `notes_token` cookie (httponly, samesite=lax)

**Failure Response:**
- HTTP 401 with "Invalid token" message

### Cookie Details

| Attribute | Value |
|-----------|-------|
| Name | `notes_token` |
| HttpOnly | Yes (not accessible via JavaScript) |
| SameSite | Lax (sent with top-level navigations) |
| Secure | Not set (allows HTTP in local network) |
| Max-Age | Session (expires when browser closes) |

## Excluded Routes

The following routes bypass authentication:

| Route | Reason |
|-------|--------|
| `/login` | Login page must be accessible to unauthenticated users |
| `/api/linkedin/oauth/callback` | OAuth callback from LinkedIn service |

The exclusion is implemented via path prefix matching:

```python
if request.url.path.startswith("/login"):
    return
if request.url.path.startswith("/api/linkedin/oauth/callback"):
    return
```

## Platform Differences

### Web

- Primary authentication method is the `notes_token` cookie
- Login page provides user-facing token entry
- Cookie persists across browser sessions (until cleared)
- No logout functionality - clear browser cookies manually

### Android

- Uses Authorization header exclusively (no cookies)
- Token hardcoded in `AppConfig.kt`
- Changing token requires app rebuild
- No login UI - app either works or fails with 401 errors

### API Clients

External API clients should use the Authorization header:

```
Authorization: Bearer <token>
```

This works identically to the Android approach and does not require cookie management.

## Security Considerations

### Strengths

- **Timing-safe comparison** - Prevents timing attacks via `secrets.compare_digest()`
- **HttpOnly cookie** - Prevents XSS-based cookie theft
- **Simple threat model** - Single token for personal/family use is appropriate

### Limitations

- **Shared token** - All users share the same credential (no per-user accounts)
- **No token rotation** - Changing the token requires updating all clients
- **No HTTPS requirement** - Suitable for local network only
- **Hardcoded Android token** - Requires rebuild to change

### Recommendations

For local network deployment (the intended use case):
- Use a strong random token (the default is 32+ characters)
- Keep the token private within the household
- Consider HTTPS if exposing beyond local network

## Key Implementation Files

| File | Purpose |
|------|---------|
| `server/web_app/main.py` | `require_auth` dependency, login routes, token loading |
| `server/web_app/.env` | `NOTES_TOKEN` environment variable |
| `app/android/.../AppConfig.kt` | Hardcoded Android auth token |
| `app/android/.../ApiClient.kt` | Authorization header injection |
