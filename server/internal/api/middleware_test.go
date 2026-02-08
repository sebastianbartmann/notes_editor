package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"notes-editor/internal/auth"
)

func TestAuthMiddleware(t *testing.T) {
	const validToken = "secret-token-123"
	middleware := AuthMiddleware(validToken)

	// Dummy handler that returns 200 OK if reached
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "valid token",
			authHeader: "Bearer secret-token-123",
			wantStatus: http.StatusOK,
			wantBody:   "OK",
		},
		{
			name:       "missing authorization header",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid token",
			authHeader: "Bearer wrong-token",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing Bearer prefix",
			authHeader: "secret-token-123",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Basic auth instead of Bearer",
			authHeader: "Basic dXNlcjpwYXNz",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Bearer with empty token",
			authHeader: "Bearer ",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "partial token match (prefix)",
			authHeader: "Bearer secret-token-12",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "partial token match (suffix)",
			authHeader: "Bearer ecret-token-123",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "token with extra characters",
			authHeader: "Bearer secret-token-123extra",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "lowercase bearer",
			authHeader: "bearer secret-token-123",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "multiple spaces in header",
			authHeader: "Bearer  secret-token-123",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/daily", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantBody != "" && rec.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", rec.Body.String(), tt.wantBody)
			}

			// Verify unauthorized responses have correct error format
			if tt.wantStatus == http.StatusUnauthorized {
				var errResp ErrorResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
					t.Errorf("failed to parse error response: %v", err)
				}
				if errResp.Detail != "Unauthorized" {
					t.Errorf("error detail = %q, want %q", errResp.Detail, "Unauthorized")
				}
			}
		})
	}
}

func TestAuthMiddleware_SkipsLinkedInCallback(t *testing.T) {
	middleware := AuthMiddleware("secret-token")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request to LinkedIn callback should skip auth
	req := httptest.NewRequest("GET", "/api/linkedin/oauth/callback?code=abc", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("LinkedIn callback should skip auth, got status %d", rec.Code)
	}
}

func TestPersonMiddleware(t *testing.T) {
	tests := []struct {
		name         string
		personHeader string
		wantStatus   int
		wantPerson   string
	}{
		{
			name:         "valid person sebastian",
			personHeader: "sebastian",
			wantStatus:   http.StatusOK,
			wantPerson:   "sebastian",
		},
		{
			name:         "valid person petra",
			personHeader: "petra",
			wantStatus:   http.StatusOK,
			wantPerson:   "petra",
		},
		{
			name:         "invalid person",
			personHeader: "hacker",
			wantStatus:   http.StatusBadRequest,
		},
		{
			name:         "empty person header (allowed)",
			personHeader: "",
			wantStatus:   http.StatusOK,
			wantPerson:   "",
		},
		{
			name:         "case sensitive - Sebastian rejected",
			personHeader: "Sebastian",
			wantStatus:   http.StatusBadRequest,
		},
		{
			name:         "case sensitive - PETRA rejected",
			personHeader: "PETRA",
			wantStatus:   http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPerson string
			handler := PersonMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPerson = auth.PersonFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/api/daily", nil)
			if tt.personHeader != "" {
				req.Header.Set("X-Notes-Person", tt.personHeader)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK && gotPerson != tt.wantPerson {
				t.Errorf("person in context = %q, want %q", gotPerson, tt.wantPerson)
			}

			// Verify bad request has correct error format
			if tt.wantStatus == http.StatusBadRequest {
				var errResp ErrorResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
					t.Errorf("failed to parse error response: %v", err)
				}
				if errResp.Detail != "Invalid person" {
					t.Errorf("error detail = %q, want %q", errResp.Detail, "Invalid person")
				}
			}
		})
	}
}

func TestRequirePerson(t *testing.T) {
	t.Run("returns person when set", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/daily", nil)
		req = req.WithContext(auth.WithPerson(req.Context(), "sebastian"))
		rec := httptest.NewRecorder()

		person, ok := requirePerson(rec, req)

		if !ok {
			t.Error("requirePerson returned false, want true")
		}
		if person != "sebastian" {
			t.Errorf("person = %q, want %q", person, "sebastian")
		}
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("writes error when not set", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/daily", nil)
		rec := httptest.NewRecorder()

		person, ok := requirePerson(rec, req)

		if ok {
			t.Error("requirePerson returned true, want false")
		}
		if person != "" {
			t.Errorf("person = %q, want empty", person)
		}
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}

		var errResp ErrorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
			t.Errorf("failed to parse error response: %v", err)
		}
		if errResp.Detail != "Person not selected" {
			t.Errorf("error detail = %q, want %q", errResp.Detail, "Person not selected")
		}
	})
}

func TestRecovererMiddleware(t *testing.T) {
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	handler := RecovererMiddleware(panicHandler)

	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()

	// Should not panic
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Errorf("failed to parse error response: %v", err)
	}
	if errResp.Detail != "Internal server error" {
		t.Errorf("error detail = %q, want %q", errResp.Detail, "Internal server error")
	}
}

func TestLoggingMiddlewarePreservesFlusher(t *testing.T) {
	handler := LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := w.(http.Flusher); !ok {
			t.Fatal("response writer does not implement http.Flusher")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
