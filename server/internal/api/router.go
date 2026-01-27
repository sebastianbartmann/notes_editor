package api

import (
	"net/http"

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

	// Middleware stack
	r.Use(RecovererMiddleware)
	r.Use(LoggingMiddleware)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Notes-Person"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(AuthMiddleware(srv.config.NotesToken))
	r.Use(PersonMiddleware)

	// Daily note routes
	r.Get("/api/daily", srv.handleGetDaily)
	r.Post("/api/save", srv.handleSaveDaily)
	r.Post("/api/append", srv.handleAppendDaily)
	r.Post("/api/clear-pinned", srv.handleClearPinned)

	// Todo routes
	r.Post("/api/todos/add", srv.handleAddTodo)
	r.Post("/api/todos/toggle", srv.handleToggleTodo)

	// Sleep times routes
	r.Get("/api/sleep-times", srv.handleGetSleepTimes)
	r.Post("/api/sleep-times/append", srv.handleAppendSleepTime)
	r.Post("/api/sleep-times/delete", srv.handleDeleteSleepTime)

	// File routes
	r.Get("/api/files/list", srv.handleListFiles)
	r.Get("/api/files/read", srv.handleReadFile)
	r.Post("/api/files/create", srv.handleCreateFile)
	r.Post("/api/files/save", srv.handleSaveFile)
	r.Post("/api/files/delete", srv.handleDeleteFile)
	r.Post("/api/files/unpin", srv.handleUnpinEntry)

	// Claude routes
	r.Post("/api/claude/chat", srv.handleClaudeChat)
	r.Post("/api/claude/chat-stream", srv.handleClaudeChatStream)
	r.Post("/api/claude/clear", srv.handleClaudeClear)
	r.Get("/api/claude/history", srv.handleClaudeHistory)

	// Settings routes
	r.Get("/api/settings/env", srv.handleGetEnv)
	r.Post("/api/settings/env", srv.handleSetEnv)

	// LinkedIn OAuth
	r.Get("/api/linkedin/oauth/callback", srv.handleLinkedInCallback)

	return r
}
