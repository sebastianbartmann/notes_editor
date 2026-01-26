# Settings & Configuration

## Purpose

The Settings feature manages user preferences and server configuration for the Notes Editor application. It handles person selection (which data space to use), theme selection (dark/light mode), and server-side environment variable management. Settings are stored client-side to maintain a stateless server architecture.

## Configuration Options

### Person Selection

Determines which person's data space (vault subfolder) the client accesses:

| Value | Description |
|-------|-------------|
| `sebastian` | Primary user's data space |
| `petra` | Secondary user's data space |

When no person is selected, users are redirected to the settings page. Most features require a valid person selection to function.

### Theme Selection

Controls the visual appearance:

| Value | Description |
|-------|-------------|
| `dark` | Dark color scheme (default) |
| `light` | Light color scheme |

The theme is applied via a CSS class (`theme-dark` or `theme-light`) on the `<body>` element.

### Environment Variables

Server-side `.env` file containing configuration for external services:

**File location:** `server/web_app/.env`

Common variables:
- `NOTES_TOKEN` - Authentication token for API access
- `ANTHROPIC_API_KEY` - Claude API key for chat feature
- `LINKEDIN_ACCESS_TOKEN` - LinkedIn API token
- `LINKEDIN_CLIENT_ID` - LinkedIn OAuth client ID
- `LINKEDIN_CLIENT_SECRET` - LinkedIn OAuth client secret

Changes to `.env` are reloaded immediately via `load_dotenv(override=True)`.

## Storage Mechanisms

### Web (Cookies)

| Cookie | Purpose | Max Age |
|--------|---------|---------|
| `notes_person` | Selected person | 1 year |
| `notes_theme` | Selected theme | 1 year |
| `notes_token` | Authentication token | Session |

Cookies use `samesite=lax` for CSRF protection. The authentication token cookie is `httponly`.

### Android (SharedPreferences)

The `UserSettings` singleton manages persistent settings:

```kotlin
object UserSettings {
    private const val PREFS_NAME = "notes_settings"
    private const val KEY_PERSON = "person_root"
    private const val KEY_THEME = "theme"
}
```

| Key | Purpose | Default |
|-----|---------|---------|
| `person_root` | Selected person | `null` |
| `theme` | Selected theme | `"dark"` |

Settings are exposed as Compose state (`mutableStateOf`) for reactive UI updates.

## Person Selection Mechanism

### Resolution Order

Person is determined from (first match wins):
1. `X-Notes-Person` HTTP header (API clients)
2. `notes_person` cookie (web browser)

### Server-Side Logic

```python
PERSONS = {"sebastian", "petra"}
PERSON_COOKIE = "notes_person"
PERSON_HEADER = "X-Notes-Person"

def get_person(request: Request) -> str | None:
    candidate = (
        request.headers.get(PERSON_HEADER)
        or request.cookies.get(PERSON_COOKIE)
        or ""
    ).strip().lower()
    return candidate if candidate in PERSONS else None
```

### Enforcement

Two helper functions enforce person selection:

| Function | Behavior |
|----------|----------|
| `ensure_person_or_redirect()` | Returns person or `RedirectResponse("/settings")` for web pages |
| `ensure_person()` | Returns person or raises `HTTPException(400)` for API endpoints |

## API Endpoints

### GET /settings

**Response:** HTML page

Renders the settings page with current person, theme, and environment variable content.

### POST /settings

**Form data (one of):**
- `person` - New person selection
- `theme` - New theme selection

**Response:** Redirect to `/settings` with updated cookie

Sets the corresponding cookie and redirects back to settings page. Returns HTTP 400 if the value is invalid or nothing is provided.

### POST /settings/env

**Form data:**
- `env_content` (required) - Full `.env` file content

**Response:** Redirect to `/settings`

Overwrites the server `.env` file with the provided content. Content is normalized (CRLF to LF) and a trailing newline is ensured. Environment is reloaded immediately.

### GET /api/settings/env

**Response:**
```json
{
  "success": true,
  "content": "NOTES_TOKEN=...\nANTHROPIC_API_KEY=..."
}
```

Returns the current `.env` file content as JSON. Used by Android client.

### POST /api/settings/env

**Form data:**
- `env_content` (required) - Full `.env` file content

**Response:**
```json
{
  "success": true,
  "message": "Saved"
}
```

JSON API equivalent of `POST /settings/env` for programmatic access.

## Platform Differences

### Web (FastAPI + HTMX)

**Person selection:**
- Buttons for each person, styled to show current selection
- Form POST to `/settings` sets cookie

**Theme selection:**
- Dark/Light toggle buttons
- Form POST to `/settings` sets cookie

**Environment editor:**
- Monospace `<textarea>` for editing
- Form POST to `/settings/env` saves content
- Changes require server restart for some values (e.g., `NOTES_TOKEN`)

**Redirects:**
- Pages without person selection redirect to `/settings`
- After saving, redirects back to `/settings`

### Android (Jetpack Compose)

**Person selection:**
- `CompactButton` for each person with highlight styling
- Updates `UserSettings.person` directly
- Triggers Compose recomposition for reactive UI

**Theme selection:**
- `CompactButton` for Dark/Light with highlight styling
- Updates `UserSettings.theme` directly
- App theme updates reactively

**Environment editor:**
- `CompactTextField` with monospace styling
- Fetches content via `GET /api/settings/env` on screen open
- Saves via `POST /api/settings/env`
- Shows status messages for success/failure

**API integration:**
- `UserSettings.person` value sent in `X-Notes-Person` header on all requests
- `UserSettings.theme` used locally only (not sent to server)

### Key Differences

| Aspect | Web | Android |
|--------|-----|---------|
| Person storage | `notes_person` cookie | SharedPreferences |
| Theme storage | `notes_theme` cookie | SharedPreferences |
| Person header | Not sent (uses cookie) | `X-Notes-Person` header |
| Env save flow | Form POST + redirect | API POST + status message |
| State reactivity | Page reload | Compose recomposition |

## Security Considerations

### Environment Variables

The `.env` editor exposes sensitive values (API keys, secrets) to authenticated users. Access is protected by:
1. Authentication via `notes_token` cookie or Bearer token
2. No anonymous access allowed

### Cookie Security

- `httponly=True` on `notes_token` prevents JavaScript access
- `samesite=lax` provides basic CSRF protection
- Person/theme cookies are not sensitive (no httponly flag)

### Input Validation

- Person values validated against `PERSONS` set
- Theme values validated against `THEMES` set
- Invalid values return HTTP 400

## Implementation Files

| File | Purpose |
|------|---------|
| `server/web_app/main.py` | Settings routes, get_person/get_theme helpers, env read/write |
| `server/web_app/templates/settings.html` | Web UI for settings page |
| `app/android/.../UserSettings.kt` | Android SharedPreferences singleton |
| `app/android/.../SettingsScreen.kt` | Android Compose settings UI |
| `app/android/.../ApiClient.kt` | Android API calls for env endpoints |
