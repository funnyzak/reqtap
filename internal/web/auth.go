package web

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/funnyzak/reqtap/internal/config"
)

// Session describes an authenticated user session.
type Session struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
}

// AuthManager performs credential validation and session management.
type AuthManager struct {
	enable bool

	timeout  time.Duration
	users    map[string]config.WebUserConfig
	sessions map[string]*Session
	mu       sync.RWMutex
}

// ErrInvalidCredential indicates username/password mismatch.
var ErrInvalidCredential = errors.New("invalid username or password")

// NewAuthManager creates a new AuthManager from configuration.
func NewAuthManager(cfg config.WebAuthConfig) *AuthManager {
	users := make(map[string]config.WebUserConfig, len(cfg.Users))
	for _, user := range cfg.Users {
		username := strings.ToLower(strings.TrimSpace(user.Username))
		if username == "" {
			continue
		}
		sanitized := user
		sanitized.Role = strings.ToLower(sanitized.Role)
		users[username] = sanitized
	}

	return &AuthManager{
		enable:   cfg.Enable,
		timeout:  cfg.SessionTimeout,
		users:    users,
		sessions: make(map[string]*Session),
	}
}

// Enabled indicates whether authentication is active.
func (a *AuthManager) Enabled() bool {
	return a != nil && a.enable
}

// Login validates credentials and returns a new session.
func (a *AuthManager) Login(username, password string) (*Session, error) {
	if !a.Enabled() {
		// Provide a pseudo session for disabled auth to keep API surface consistent.
		return &Session{
			ID:        "public",
			Username:  "guest",
			Role:      "viewer",
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}, nil
	}

	username = strings.ToLower(strings.TrimSpace(username))
	user, ok := a.users[username]
	if !ok || user.Password != password {
		return nil, ErrInvalidCredential
	}

	session := &Session{
		ID:        randomToken(),
		Username:  user.Username,
		Role:      user.Role,
		ExpiresAt: time.Now().Add(a.timeout),
	}

	a.mu.Lock()
	a.sessions[session.ID] = session
	a.mu.Unlock()

	return session, nil
}

// Validate finds a session by token and ensures it's not expired.
func (a *AuthManager) Validate(token string) (*Session, error) {
	if !a.Enabled() {
		return &Session{
			ID:        "public",
			Username:  "guest",
			Role:      "viewer",
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}, nil
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrInvalidCredential
	}

	a.mu.RLock()
	session, ok := a.sessions[token]
	a.mu.RUnlock()

	if !ok {
		return nil, ErrInvalidCredential
	}

	if time.Now().After(session.ExpiresAt) {
		a.mu.Lock()
		delete(a.sessions, token)
		a.mu.Unlock()
		return nil, ErrInvalidCredential
	}

	return session, nil
}

// Logout removes a session token.
func (a *AuthManager) Logout(token string) {
	if !a.Enabled() {
		return
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return
	}

	a.mu.Lock()
	delete(a.sessions, token)
	a.mu.Unlock()
}

// Cleanup removes expired sessions; called periodically if needed.
func (a *AuthManager) Cleanup() {
	if !a.Enabled() {
		return
	}

	now := time.Now()

	a.mu.Lock()
	defer a.mu.Unlock()

	for token, session := range a.sessions {
		if now.After(session.ExpiresAt) {
			delete(a.sessions, token)
		}
	}
}

func randomToken() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(buf)
}
