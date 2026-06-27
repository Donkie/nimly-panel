// Package api wires the HTTP routes, middleware and handlers for the panel.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Donkie/nimly-panel/backend/internal/audit"
	"github.com/Donkie/nimly-panel/backend/internal/auth"
	"github.com/Donkie/nimly-panel/backend/internal/lock"
)

// Server holds the dependencies shared by the HTTP handlers.
type Server struct {
	svc   *lock.Service
	auth  *auth.Authenticator
	audit *audit.Logger
	log   *slog.Logger
	static http.Handler
}

// NewServer constructs the API server. static serves the embedded SPA assets.
func NewServer(svc *lock.Service, a *auth.Authenticator, au *audit.Logger, log *slog.Logger, static http.Handler) *Server {
	return &Server{svc: svc, auth: a, audit: au, log: log, static: static}
}

// Handler returns the fully composed HTTP handler (routes + middleware).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Public auth routes.
	mux.HandleFunc("GET /api/auth/login", s.auth.LoginHandler)
	mux.HandleFunc("GET /api/auth/callback", s.auth.CallbackHandler)
	mux.Handle("POST /api/auth/logout", s.protect(http.HandlerFunc(s.auth.LogoutHandler)))

	// Protected API.
	mux.Handle("GET /api/me", s.protect(http.HandlerFunc(s.handleMe)))
	mux.Handle("GET /api/lock", s.protect(http.HandlerFunc(s.handleGetLock)))
	mux.Handle("POST /api/lock/state", s.protect(http.HandlerFunc(s.handleSetLockState)))
	mux.Handle("GET /api/pins", s.protect(http.HandlerFunc(s.handleGetPins)))
	mux.Handle("PUT /api/pins/{user}", s.protect(http.HandlerFunc(s.handleSetPin)))
	mux.Handle("DELETE /api/pins/{user}", s.protect(http.HandlerFunc(s.handleDeletePin)))
	mux.Handle("POST /api/settings/sound-volume", s.protect(http.HandlerFunc(s.handleSoundVolume)))
	mux.Handle("POST /api/settings/auto-relock", s.protect(http.HandlerFunc(s.handleAutoRelock)))
	mux.Handle("POST /api/refresh", s.protect(http.HandlerFunc(s.handleRefresh)))
	mux.Handle("GET /api/stream", s.protect(http.HandlerFunc(s.handleStream)))

	// SPA / static assets.
	mux.Handle("/", s.static)

	// Middleware chain (outermost first): session load/save → security headers.
	h := securityHeaders(mux)
	h = s.auth.SessionManager().LoadAndSave(h)
	h = recoverer(s.log, h)
	return h
}

func (s *Server) protect(h http.Handler) http.Handler {
	return s.auth.RequireAuth(h)
}

func actorOf(u auth.User) string {
	if u.Email != "" {
		return u.Email
	}
	if u.Subject != "" {
		return u.Subject
	}
	return "unknown"
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		// SvelteKit injects a small inline bootstrap script into the static
		// shell; a static SPA can't use per-request nonces, so inline scripts
		// from our own build are allowed. connect-src stays 'self' (API + SSE),
		// and framing/base/object/form are locked down.
		h.Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data:; "+
				"font-src 'self' data:; "+
				"connect-src 'self'; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'; "+
				"object-src 'none'")
		next.ServeHTTP(w, r)
	})
}

func recoverer(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error("panic in handler", "err", rec, "path", r.URL.Path)
				writeErr(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
