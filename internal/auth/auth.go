package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/securecookie"
	"golang.org/x/oauth2"

	"github.com/inelson/finguard/internal/config"
	"github.com/inelson/finguard/internal/models"
	"github.com/inelson/finguard/internal/store"
)

type contextKey string

const userContextKey contextKey = "finguard_user"

const (
	sessionCookieName = "finguard_session"
	stateCookieName   = "finguard_oauth_state"
	sessionMaxAge     = 24 * time.Hour
)

type Manager struct {
	provider     *oidc.Provider
	verifier     *oidc.IDTokenVerifier
	oauth2Config oauth2.Config
	cookie       *securecookie.SecureCookie
	store        store.Store
	logger       *slog.Logger
	disabled     bool
}

type SessionData struct {
	UserID      string    `json:"userId"`
	Email       string    `json:"email"`
	DisplayName string    `json:"displayName"`
	Groups      []string  `json:"groups,omitempty"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

func NewManager(cfg *config.Config, st store.Store, logger *slog.Logger) (*Manager, error) {
	if cfg.AuthDisabled || cfg.OIDCIssuer == "" {
		logger.Info("authentication disabled")
		return &Manager{disabled: true, store: st, logger: logger}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	provider, err := oidc.NewProvider(ctx, cfg.OIDCIssuer)
	if err != nil {
		return nil, fmt.Errorf("oidc provider discovery: %w", err)
	}

	oauth2Config := oauth2.Config{
		ClientID:     cfg.OIDCClientID,
		ClientSecret: cfg.OIDCClientSecret,
		RedirectURL:  cfg.OIDCRedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       cfg.OIDCScopes,
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: cfg.OIDCClientID})

	sessionKey := []byte(cfg.SessionSecret)
	if len(sessionKey) == 0 {
		sessionKey = securecookie.GenerateRandomKey(32)
		logger.Warn("no session secret configured, using random key (sessions will not persist across restarts)")
	}
	sc := securecookie.New(sessionKey, nil)

	return &Manager{
		provider:     provider,
		verifier:     verifier,
		oauth2Config: oauth2Config,
		cookie:       sc,
		store:        st,
		logger:       logger,
	}, nil
}

func (m *Manager) IsDisabled() bool {
	return m.disabled
}

func (m *Manager) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if m.disabled {
		writeJSON(w, http.StatusOK, map[string]string{"status": "auth disabled"})
		return
	}

	state, err := generateState()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate state"})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    state,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, m.oauth2Config.AuthCodeURL(state), http.StatusFound)
}

func (m *Manager) HandleCallback(w http.ResponseWriter, r *http.Request) {
	if m.disabled {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	stateCookie, err := r.Cookie(stateCookieName)
	if err != nil || stateCookie.Value == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing state cookie"})
		return
	}
	if r.URL.Query().Get("state") != stateCookie.Value {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "state mismatch"})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:   stateCookieName,
		Path:   "/",
		MaxAge: -1,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing authorization code"})
		return
	}

	token, err := m.oauth2Config.Exchange(r.Context(), code)
	if err != nil {
		m.logger.Error("token exchange failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "token exchange failed"})
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "no id_token in response"})
		return
	}

	idToken, err := m.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		m.logger.Error("id token verification failed", "error", err)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid id token"})
		return
	}

	var claims struct {
		Email    string   `json:"email"`
		Name     string   `json:"name"`
		Groups   []string `json:"groups"`
		Subject  string   `json:"sub"`
	}
	if err := idToken.Claims(&claims); err != nil {
		m.logger.Error("failed to parse claims", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to parse claims"})
		return
	}

	user, err := m.provisionUser(r.Context(), claims.Subject, claims.Email, claims.Name, claims.Groups)
	if err != nil {
		m.logger.Error("user provisioning failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to provision user"})
		return
	}

	session := SessionData{
		UserID:      user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Groups:      claims.Groups,
		ExpiresAt:   time.Now().Add(sessionMaxAge),
	}

	encoded, err := m.encodeSession(session)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    encoded,
		Path:     "/",
		MaxAge:   int(sessionMaxAge.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusFound)
}

func (m *Manager) HandleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   sessionCookieName,
		Path:   "/",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (m *Manager) HandleUserInfo(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// Middleware returns HTTP middleware that validates sessions.
func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.disabled {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}

		session, err := m.decodeSession(cookie.Value)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid session"})
			return
		}

		if time.Now().After(session.ExpiresAt) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "session expired"})
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserFromContext(ctx context.Context) *SessionData {
	session, _ := ctx.Value(userContextKey).(*SessionData)
	return session
}

func (m *Manager) provisionUser(ctx context.Context, subject, email, name string, groups []string) (*models.User, error) {
	user, err := m.store.GetUserByOIDCSubject(ctx, subject)
	if err != nil {
		return nil, err
	}
	if user != nil {
		return user, nil
	}

	user, err = m.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user != nil {
		return user, nil
	}

	displayName := name
	if displayName == "" {
		displayName = email
	}
	user = &models.User{
		Email:       email,
		DisplayName: displayName,
		OIDCSubject: subject,
	}
	if err := m.store.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	m.logger.Info("provisioned new user from OIDC", "email", email, "subject", subject)

	for _, groupClaim := range groups {
		m.syncGroup(ctx, groupClaim, user.ID)
	}

	return user, nil
}

func (m *Manager) syncGroup(ctx context.Context, claim, userID string) {
	group, err := m.store.GetGroupByOIDCClaim(ctx, claim)
	if err != nil {
		m.logger.Error("failed to get group", "claim", claim, "error", err)
		return
	}
	if group == nil {
		group = &models.Group{
			Name:      claim,
			OIDCClaim: claim,
		}
		if err := m.store.CreateGroup(ctx, group); err != nil {
			m.logger.Error("failed to create group", "claim", claim, "error", err)
			return
		}
	}
	if err := m.store.AddGroupMember(ctx, group.ID, userID); err != nil {
		m.logger.Error("failed to add user to group", "claim", claim, "userId", userID, "error", err)
	}
}

func (m *Manager) encodeSession(session SessionData) (string, error) {
	data, err := json.Marshal(session)
	if err != nil {
		return "", err
	}
	return m.cookie.Encode(sessionCookieName, string(data))
}

func (m *Manager) decodeSession(encoded string) (*SessionData, error) {
	var raw string
	if err := m.cookie.Decode(sessionCookieName, encoded, &raw); err != nil {
		return nil, err
	}
	var session SessionData
	if err := json.Unmarshal([]byte(raw), &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
