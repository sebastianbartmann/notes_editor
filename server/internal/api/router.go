package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"notes-editor/internal/claude"
	"notes-editor/internal/config"
	"notes-editor/internal/linkedin"
	"notes-editor/internal/vault"
)

// Server holds all dependencies for the HTTP server.
type Server struct {
	config   *config.Config
	store    *vault.Store
	daily    *vault.Daily
	git      *vault.Git
	claude   *claude.Service
	linkedin *linkedin.Service
}

// NewServer creates a new server with all dependencies.
func NewServer(cfg *config.Config) *Server {
	store := vault.NewStore(cfg.NotesRoot)
	daily := vault.NewDaily(store)
	git := vault.NewGit(cfg.NotesRoot)

	var linkedinSvc *linkedin.Service
	if cfg.LinkedIn.AccessToken != "" {
		linkedinSvc = linkedin.NewService(&cfg.LinkedIn, cfg.NotesRoot)
	}

	var claudeSvc *claude.Service
	if cfg.AnthropicKey != "" {
		claudeSvc = claude.NewService(cfg.AnthropicKey, store, linkedinSvc)
	}

	return &Server{
		config:   cfg,
		store:    store,
		daily:    daily,
		git:      git,
		claude:   claudeSvc,
		linkedin: linkedinSvc,
	}
}

// NewRouter creates the HTTP router with all routes configured.
func NewRouter(srv *Server) http.Handler {
	r := chi.NewRouter()

	// Global middleware (no auth)
	r.Use(RecovererMiddleware)
	r.Use(LoggingMiddleware)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Notes-Person"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// API routes with auth
	r.Route("/api", func(r chi.Router) {
		r.Use(AuthMiddleware(srv.config.NotesToken))
		r.Use(PersonMiddleware)

		// Daily note routes
		r.Get("/daily", srv.handleGetDaily)
		r.Post("/save", srv.handleSaveDaily)
		r.Post("/append", srv.handleAppendDaily)
		r.Post("/clear-pinned", srv.handleClearPinned)

		// Todo routes
		r.Post("/todos/add", srv.handleAddTodo)
		r.Post("/todos/toggle", srv.handleToggleTodo)

		// Sleep times routes
		r.Get("/sleep-times", srv.handleGetSleepTimes)
		r.Post("/sleep-times/append", srv.handleAppendSleepTime)
		r.Post("/sleep-times/delete", srv.handleDeleteSleepTime)

		// File routes
		r.Get("/files/list", srv.handleListFiles)
		r.Get("/files/read", srv.handleReadFile)
		r.Post("/files/create", srv.handleCreateFile)
		r.Post("/files/save", srv.handleSaveFile)
		r.Post("/files/delete", srv.handleDeleteFile)
		r.Post("/files/unpin", srv.handleUnpinEntry)

		// Claude routes
		r.Post("/claude/chat", srv.handleClaudeChat)
		r.Post("/claude/chat-stream", srv.handleClaudeChatStream)
		r.Post("/claude/clear", srv.handleClaudeClear)
		r.Get("/claude/history", srv.handleClaudeHistory)

		// Settings routes
		r.Get("/settings/env", srv.handleGetEnv)
		r.Post("/settings/env", srv.handleSetEnv)

		// LinkedIn OAuth
		r.Get("/linkedin/oauth/callback", srv.handleLinkedInCallback)
	})

	// Static file serving for web UI (no auth)
	staticDir := srv.config.StaticDir
	if staticDir == "" {
		staticDir = "./static"
	}
	r.Get("/*", staticFileHandler(staticDir))

	return r
}

// staticFileHandler serves static files and falls back to index.html for SPA routing.
func staticFileHandler(staticDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		fullPath := filepath.Join(staticDir, path)

		// Check if file exists
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			// SPA fallback: serve index.html for non-existent paths
			http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
			return
		}

		// Serve the actual file
		http.ServeFile(w, r, fullPath)
	}
}
