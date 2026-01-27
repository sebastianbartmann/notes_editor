# Authentication and Context Management Specification

> Status: Active
> Version: 3.0
> Last Updated: 2026-01-27

## Overview

This document defines the authentication, authorization, and context management systems for the Notes Editor application. It covers bearer token authentication, person context resolution, theme preferences, and path utilities for multi-user file access.

The authentication and context systems serve as foundational dependencies for all other application specifications.

**Implementation:**
- Go backend: `server/internal/auth/`, `server/internal/api/middleware.go`
- React web client: `clients/web/src/context/`

---

## Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NOTES_TOKEN` | No | `VJY9EoAf1xx1bO-LaduCmItwRitCFm9BPuQZ8jd0tcg` | Bearer token for API authentication |

### Constants

```go
// server/internal/auth/person.go
var ValidPersons = []string{"sebastian", "petra"}
```

```typescript
// clients/web/src/context/PersonContext.tsx
type Person = 'sebastian' | 'petra' | null

// clients/web/src/context/ThemeContext.tsx
type Theme = 'dark' | 'light'
```

### Request Headers

| Header | Purpose | Required |
|--------|---------|----------|
| `Authorization` | Bearer token authentication | Yes (all API requests) |
| `X-Notes-Person` | Person context for scoped file access | Yes (most API requests) |

### Client Storage (React Web)

| Key | Purpose | Default |
|-----|---------|---------|
| `notes_token` | Authentication token persistence | `null` |
| `notes_person` | Person context persistence | `null` |
| `notes_theme` | Theme preference persistence | `"dark"` |

---

## Go Backend Authentication

### Package Structure

```
server/internal/auth/
├── auth.go      # Token validation, context helpers
├── person.go    # Valid persons list
└── auth_test.go # Unit tests
```

### Token Validation

```go
// server/internal/auth/auth.go
package auth

import (
    "crypto/subtle"
)

// ValidateToken performs constant-time comparison to prevent timing attacks.
func ValidateToken(provided, expected string) bool {
    return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}
```

**Security Feature**: Uses `crypto/subtle.ConstantTimeCompare` to prevent timing attacks that could leak token information.

### Auth Middleware

```go
// server/internal/api/middleware.go
func AuthMiddleware(token string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Skip auth for OAuth callback
            if strings.HasPrefix(r.URL.Path, "/api/linkedin/oauth/callback") {
                next.ServeHTTP(w, r)
                return
            }

            authHeader := r.Header.Get("Authorization")
            if authHeader == "" {
                writeUnauthorized(w)
                return
            }

            // Extract Bearer token
            parts := strings.SplitN(authHeader, " ", 2)
            if len(parts) != 2 || parts[0] != "Bearer" {
                writeUnauthorized(w)
                return
            }

            if !auth.ValidateToken(parts[1], token) {
                writeUnauthorized(w)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

**Exempted Paths**: `/api/linkedin/oauth/callback` (OAuth flow)

**Authentication Flow**:
1. Check if path is exempted
2. Extract `Authorization` header
3. Validate `Bearer <token>` format
4. Verify token using constant-time comparison
5. Reject with 401 if any step fails

---

## Person Context Management

### Go Backend

Person context is stored in the request context and extracted from the `X-Notes-Person` header.

```go
// server/internal/auth/auth.go
type contextKey string

const personContextKey contextKey = "person"

// PersonFromContext retrieves the person value from the context.
func PersonFromContext(ctx context.Context) string {
    person, ok := ctx.Value(personContextKey).(string)
    if !ok {
        return ""
    }
    return person
}

// WithPerson returns a new context with the person value set.
func WithPerson(ctx context.Context, person string) context.Context {
    return context.WithValue(ctx, personContextKey, person)
}
```

```go
// server/internal/auth/person.go
var ValidPersons = []string{"sebastian", "petra"}

func IsValidPerson(person string) bool {
    for _, p := range ValidPersons {
        if p == person {
            return true
        }
    }
    return false
}
```

### Person Middleware

```go
// server/internal/api/middleware.go
func PersonMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        person := r.Header.Get("X-Notes-Person")
        if person != "" {
            if !auth.IsValidPerson(person) {
                writeBadRequest(w, "Invalid person")
                return
            }
            r = r.WithContext(auth.WithPerson(r.Context(), person))
        }
        next.ServeHTTP(w, r)
    })
}
```

### Handler Helper

```go
// server/internal/api/middleware.go
func requirePerson(w http.ResponseWriter, r *http.Request) (string, bool) {
    person := auth.PersonFromContext(r.Context())
    if person == "" {
        writeBadRequest(w, "Person not selected")
        return "", false
    }
    return person, true
}
```

**Usage in handlers**:
```go
func (s *Server) handleGetDaily(w http.ResponseWriter, r *http.Request) {
    person, ok := requirePerson(w, r)
    if !ok {
        return
    }
    // person is now available for scoped file operations
}
```

---

## React Web Client Context Providers

### AuthContext

```typescript
// clients/web/src/context/AuthContext.tsx
interface AuthState {
  token: string | null
  isAuthenticated: boolean
}

interface AuthContextType extends AuthState {
  login: (token: string) => void
  logout: () => void
}

const STORAGE_KEY = 'notes_token'

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(() => {
    return localStorage.getItem(STORAGE_KEY)
  })

  useEffect(() => {
    if (token) {
      localStorage.setItem(STORAGE_KEY, token)
    } else {
      localStorage.removeItem(STORAGE_KEY)
    }
  }, [token])

  // ... login/logout functions
}
```

**Features**:
- Persists token to localStorage
- Provides `isAuthenticated` computed state
- `login(token)` saves token and authenticates
- `logout()` clears token

### PersonContext

```typescript
// clients/web/src/context/PersonContext.tsx
type Person = 'sebastian' | 'petra' | null

interface PersonContextType {
  person: Person
  setPerson: (person: Person) => void
}

const STORAGE_KEY = 'notes_person'

export function PersonProvider({ children }: { children: ReactNode }) {
  const [person, setPersonState] = useState<Person>(() => {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored === 'sebastian' || stored === 'petra') {
      return stored
    }
    return null
  })

  useEffect(() => {
    if (person) {
      localStorage.setItem(STORAGE_KEY, person)
    } else {
      localStorage.removeItem(STORAGE_KEY)
    }
  }, [person])

  // ... setPerson function
}
```

**Features**:
- Validates stored value against valid persons
- Persists to localStorage
- Returns `null` if no valid person selected

### ThemeContext

```typescript
// clients/web/src/context/ThemeContext.tsx
type Theme = 'dark' | 'light'

interface ThemeContextType {
  theme: Theme
  setTheme: (theme: Theme) => void
  toggleTheme: () => void
}

const STORAGE_KEY = 'notes_theme'

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setThemeState] = useState<Theme>(() => {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored === 'dark' || stored === 'light') {
      return stored
    }
    return 'dark'  // Default theme
  })

  useEffect(() => {
    localStorage.setItem(STORAGE_KEY, theme)
    if (theme === 'light') {
      document.body.classList.add('theme-light')
    } else {
      document.body.classList.remove('theme-light')
    }
  }, [theme])

  // ... setTheme, toggleTheme functions
}
```

**Features**:
- Default theme is `"dark"`
- Persists to localStorage
- Applies CSS class to `<body>` element
- Provides `toggleTheme()` convenience method

### Provider Hierarchy

```tsx
// clients/web/src/App.tsx
<AuthProvider>
  <PersonProvider>
    <ThemeProvider>
      <RouterProvider router={router} />
    </ThemeProvider>
  </PersonProvider>
</AuthProvider>
```

---

## API Client Integration

### HTTP Headers

The React API client includes authentication and person context headers on all requests:

```typescript
// clients/web/src/api/client.ts
export function createHeaders(): HeadersInit {
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
  }

  const token = localStorage.getItem('notes_token')
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const person = localStorage.getItem('notes_person')
  if (person) {
    headers['X-Notes-Person'] = person
  }

  return headers
}
```

---

## Directory Structure

Person context scopes all file operations to a person's subdirectory within the vault.

```
VAULT_ROOT/
  sebastian/
    daily/
    notes/
    linkedin/
    ...
  petra/
    daily/
    notes/
    linkedin/
    ...
  sleep_times.md  (shared)
```

---

## Security Considerations

### Token Security

1. **Constant-Time Comparison**: All token comparisons use `crypto/subtle.ConstantTimeCompare` to prevent timing attacks.

2. **Default Token Warning**: The default `NOTES_TOKEN` value should be overridden in production.

3. **Token Storage**:
   - Backend: Environment variable only, never logged
   - Client: localStorage (acceptable for family app, not for sensitive data)

### Path Traversal Prevention

1. **Person Scoping**: All file operations are scoped to a person's directory via the vault package.

2. **Path Validation**: The vault package validates paths and rejects:
   - Empty paths
   - Absolute paths
   - Paths containing `..` traversal

3. **Person Isolation**: Users cannot access other users' files through the API.

### Client-Side Security

1. **No Cookie-Based Auth**: The React SPA uses header-based auth only, avoiding CSRF concerns.

2. **localStorage Persistence**: Token persists across sessions. Users must explicitly logout.

---

## Error Responses

### Authentication Errors

| Status | Detail | Cause |
|--------|--------|-------|
| 401 | `"Unauthorized"` | Missing or invalid `Authorization` header |

### Person Context Errors

| Status | Detail | Cause |
|--------|--------|-------|
| 400 | `"Person not selected"` | Missing `X-Notes-Person` header on required endpoint |
| 400 | `"Invalid person"` | Person value not in `ValidPersons` list |

---

## Integration Notes

This specification provides foundational services used by other specifications:

### Dependent Specifications

| Specification | Dependencies |
|---------------|--------------|
| **01-rest-api-contract.md** | Bearer token validation, person context headers |
| **02-vault-storage-git-sync.md** | Person-scoped file paths |
| **04-claude-service.md** | Authentication middleware, person context |
| **05-linkedin-service.md** | OAuth callback path exemption |
| **06-daily-note-format.md** | Person context for daily note paths |
| **09-sleep-tracking.md** | Shared file access (sleep_times.md) |

### Common Patterns

**Go Handler with Person Context**:
```go
func (s *Server) handleExample(w http.ResponseWriter, r *http.Request) {
    person, ok := requirePerson(w, r)
    if !ok {
        return
    }
    // Use s.vault.ReadFile(person, "path/to/file.md")
}
```

**React Component with Auth Check**:
```tsx
function ProtectedPage() {
  const { isAuthenticated } = useAuth()
  const { person } = usePerson()

  if (!isAuthenticated) {
    return <Navigate to="/login" />
  }

  if (!person) {
    return <Navigate to="/settings" />
  }

  return <Content />
}
```

**API Call with Headers**:
```typescript
const response = await fetch('/api/daily', {
  headers: createHeaders(),
})
```
