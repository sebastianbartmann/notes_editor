package api

import (
	"bufio"
	"log"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"notes-editor/internal/auth"
)

// AuthMiddleware validates the Bearer token from the Authorization header.
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

// PersonMiddleware extracts the X-Notes-Person header and adds it to context.
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

// requirePerson is a helper that returns the person from context or writes an error.
func requirePerson(w http.ResponseWriter, r *http.Request) (string, bool) {
	person := auth.PersonFromContext(r.Context())
	if person == "" {
		writeBadRequest(w, "Person not selected")
		return "", false
	}
	return person, true
}

// LoggingMiddleware logs HTTP requests.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		log.Printf("%s %s %d %v",
			r.Method,
			r.URL.Path,
			wrapped.status,
			time.Since(start),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return h.Hijack()
}

func (rw *responseWriter) Push(target string, opts *http.PushOptions) error {
	p, ok := rw.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return p.Push(target, opts)
}

// RecovererMiddleware recovers from panics and returns a 500 error.
func RecovererMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v\n%s", err, debug.Stack())
				writeError(w, http.StatusInternalServerError, "Internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
