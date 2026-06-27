// Package auth implements OIDC (Authorization Code + PKCE) login against
// PocketID, server-side sessions, an optional user allowlist, route-protection
// middleware and CSRF protection for state-changing requests.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/Donkie/nimly-panel/backend/internal/config"
)

// User is the authenticated identity stored in the session.
type User struct {
	Subject string
	Name    string
	Email   string
	Groups  []string
}

type ctxKey int

const userCtxKey ctxKey = 0

// Authenticator handles the OIDC flow and session lifecycle.
type Authenticator struct {
	sessions     *scs.SessionManager
	oauth        oauth2.Config
	verifier     *oidc.IDTokenVerifier
	cfg          *config.Config
	allowGroups  map[string]bool
	allowSubs    map[string]bool
	appBaseURL   string
}

// New constructs an Authenticator, discovering the OIDC provider at the issuer.
func New(ctx context.Context, cfg *config.Config) (*Authenticator, error) {
	provider, err := oidc.NewProvider(ctx, cfg.OIDCIssuer)
	if err != nil {
		return nil, err
	}

	sm := scs.New()
	sm.Lifetime = cfg.SessionLifetime
	sm.IdleTimeout = cfg.SessionIdle
	sm.Cookie.Name = "nimly_session"
	sm.Cookie.HttpOnly = true
	sm.Cookie.SameSite = http.SameSiteLaxMode
	sm.Cookie.Secure = !cfg.DevMode
	sm.Cookie.Path = "/"

	a := &Authenticator{
		sessions: sm,
		oauth: oauth2.Config{
			ClientID:     cfg.OIDCClientID,
			ClientSecret: cfg.OIDCClientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  cfg.OIDCRedirectURL,
			Scopes:       cfg.OIDCScopes,
		},
		verifier:    provider.Verifier(&oidc.Config{ClientID: cfg.OIDCClientID}),
		cfg:         cfg,
		allowGroups: toSet(cfg.AllowedGroups),
		allowSubs:   toSet(cfg.AllowedSubjects),
		appBaseURL:  cfg.AppBaseURL,
	}
	return a, nil
}

// SessionManager returns the scs manager so the server can wrap the mux with
// LoadAndSave.
func (a *Authenticator) SessionManager() *scs.SessionManager { return a.sessions }

// LoginHandler starts the OIDC Authorization Code + PKCE flow.
func (a *Authenticator) LoginHandler(w http.ResponseWriter, r *http.Request) {
	state := randString(24)
	nonce := randString(24)
	verifier := oauth2.GenerateVerifier()

	a.sessions.Put(r.Context(), "oauth_state", state)
	a.sessions.Put(r.Context(), "oauth_nonce", nonce)
	a.sessions.Put(r.Context(), "oauth_verifier", verifier)

	url := a.oauth.AuthCodeURL(state,
		oidc.Nonce(nonce),
		oauth2.AccessTypeOnline,
		oauth2.S256ChallengeOption(verifier),
	)
	http.Redirect(w, r, url, http.StatusFound)
}

// CallbackHandler completes the OIDC flow, validates the ID token, enforces the
// allowlist and establishes the session.
func (a *Authenticator) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		http.Error(w, "authentication failed: "+errMsg, http.StatusUnauthorized)
		return
	}

	wantState := a.sessions.GetString(ctx, "oauth_state")
	if wantState == "" || r.URL.Query().Get("state") != wantState {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}
	verifier := a.sessions.GetString(ctx, "oauth_verifier")
	wantNonce := a.sessions.GetString(ctx, "oauth_nonce")

	token, err := a.oauth.Exchange(ctx, r.URL.Query().Get("code"), oauth2.VerifierOption(verifier))
	if err != nil {
		http.Error(w, "token exchange failed", http.StatusUnauthorized)
		return
	}
	rawID, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "no id_token in response", http.StatusUnauthorized)
		return
	}
	idToken, err := a.verifier.Verify(ctx, rawID)
	if err != nil {
		http.Error(w, "id_token verification failed", http.StatusUnauthorized)
		return
	}
	if idToken.Nonce != wantNonce {
		http.Error(w, "invalid nonce", http.StatusBadRequest)
		return
	}

	var claims struct {
		Subject string   `json:"sub"`
		Name    string   `json:"name"`
		Email   string   `json:"email"`
		Groups  []string `json:"groups"`
	}
	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, "failed to parse claims", http.StatusInternalServerError)
		return
	}
	if claims.Subject == "" {
		claims.Subject = idToken.Subject
	}

	user := User{
		Subject: claims.Subject,
		Name:    claims.Name,
		Email:   claims.Email,
		Groups:  claims.Groups,
	}
	if !a.allowed(user) {
		http.Error(w, "your account is not permitted to access this panel", http.StatusForbidden)
		return
	}

	// Rotate the session token on privilege change to prevent fixation.
	if err := a.sessions.RenewToken(ctx); err != nil {
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}
	a.sessions.Remove(ctx, "oauth_state")
	a.sessions.Remove(ctx, "oauth_nonce")
	a.sessions.Remove(ctx, "oauth_verifier")

	a.sessions.Put(ctx, "authenticated", true)
	a.sessions.Put(ctx, "sub", user.Subject)
	a.sessions.Put(ctx, "name", user.Name)
	a.sessions.Put(ctx, "email", user.Email)
	a.sessions.Put(ctx, "groups", strings.Join(user.Groups, ","))
	a.sessions.Put(ctx, "csrf", randString(24))

	http.Redirect(w, r, a.appBaseURL+"/", http.StatusFound)
}

// LogoutHandler destroys the session.
func (a *Authenticator) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if err := a.sessions.Destroy(r.Context()); err != nil {
		http.Error(w, "logout failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// CSRFToken returns the session CSRF token.
func (a *Authenticator) CSRFToken(r *http.Request) string {
	return a.sessions.GetString(r.Context(), "csrf")
}

// RequireAuth wraps a handler, rejecting unauthenticated requests with 401 and
// enforcing CSRF on unsafe methods.
func (a *Authenticator) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if !a.sessions.GetBool(ctx, "authenticated") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if isUnsafe(r.Method) {
			if r.Header.Get("X-CSRF-Token") != a.sessions.GetString(ctx, "csrf") {
				http.Error(w, "invalid csrf token", http.StatusForbidden)
				return
			}
		}
		u := User{
			Subject: a.sessions.GetString(ctx, "sub"),
			Name:    a.sessions.GetString(ctx, "name"),
			Email:   a.sessions.GetString(ctx, "email"),
			Groups:  splitNonEmpty(a.sessions.GetString(ctx, "groups")),
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(ctx, userCtxKey, u)))
	})
}

// UserFrom returns the authenticated user from the request context.
func UserFrom(r *http.Request) (User, bool) {
	u, ok := r.Context().Value(userCtxKey).(User)
	return u, ok
}

func (a *Authenticator) allowed(u User) bool {
	if len(a.allowSubs) == 0 && len(a.allowGroups) == 0 {
		return true // no allowlist configured → any authenticated user
	}
	if a.allowSubs[u.Subject] {
		return true
	}
	for _, g := range u.Groups {
		if a.allowGroups[g] {
			return true
		}
	}
	return false
}

func isUnsafe(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

func toSet(items []string) map[string]bool {
	if len(items) == 0 {
		return nil
	}
	m := make(map[string]bool, len(items))
	for _, i := range items {
		m[i] = true
	}
	return m
}

func splitNonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

func randString(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failure is fatal-class; fall back to time-based entropy.
		panic(errors.New("crypto/rand failed: " + err.Error()))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
