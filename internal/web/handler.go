package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"

	"github.com/funnyzak/reqtap/internal/config"
	"github.com/funnyzak/reqtap/internal/logger"
	"github.com/funnyzak/reqtap/internal/static"
	"github.com/funnyzak/reqtap/pkg/request"
)

const (
	sessionCookieName = "reqtap_session"
	configPlaceholder = "<!--REQTAP_CONFIG-->"
	loginPageName     = "login.html"
	indexPageName     = "index.html"
	defaultListLimit  = 100
	maxListLimit      = 500
	contextSessionKey = contextKey("web_session")
	contentTypeJSON   = "application/json"
	contentTypeHTML   = "text/html; charset=utf-8"
	roleAdmin         = "admin"
	roleViewer        = "viewer"
)

type contextKey string

// Service bundles web UI and API capabilities.
type Service struct {
	cfg         *config.WebConfig
	logger      logger.Logger
	store       *RequestStore
	auth        *AuthManager
	hub         *WebsocketHub
	staticFS    fs.FS
	files       http.Handler
	formats     []string
	cleanupStop chan struct{}
	cleanupWG   sync.WaitGroup
}

// NewService builds a Service from configuration.
func NewService(cfg *config.WebConfig, log logger.Logger) *Service {
	store := NewRequestStore(cfg.MaxRequests)
	hub := NewWebsocketHub(log)
	auth := NewAuthManager(cfg.Auth)
	formats := AllowedFormats(cfg.Export.Formats)
	assets := static.Assets

	svc := &Service{
		cfg:      cfg,
		logger:   log,
		store:    store,
		auth:     auth,
		hub:      hub,
		staticFS: assets,
		files:    http.FileServer(http.FS(assets)),
		formats:  formats,
	}

	if svc.auth.Enabled() {
		svc.startSessionCleanup()
	}

	return svc
}

// RegisterRoutes wires HTTP routes into the provided router.
func (s *Service) RegisterRoutes(router *mux.Router) {
	if s == nil || !s.cfg.Enable {
		return
	}

	adminBase := normalizePath(s.cfg.AdminPath)
	webBase := normalizePath(s.cfg.Path)

	// API routes
	apiRouter := router.PathPrefix(adminBase).Subrouter()
	apiRouter.HandleFunc("/auth/login", s.handleLogin).Methods(http.MethodPost)
	apiRouter.HandleFunc("/auth/logout", s.handleLogout).Methods(http.MethodPost)
	apiRouter.Handle("/auth/me", s.authMiddleware(http.HandlerFunc(s.handleMe))).Methods(http.MethodGet)
	apiRouter.Handle("/requests", s.authMiddleware(http.HandlerFunc(s.handleRequests))).Methods(http.MethodGet)
	apiRouter.Handle("/export", s.authMiddleware(http.HandlerFunc(s.handleExport))).Methods(http.MethodGet)
	apiRouter.Handle("/ws", s.authMiddleware(http.HandlerFunc(s.handleWebsocket))).Methods(http.MethodGet)

	// Static routes
	if webBase == "/" {
		router.HandleFunc("/", s.wrapPage(indexPageName, true)).Methods(http.MethodGet)
	} else {
		router.HandleFunc(webBase, s.redirectTo(webBase+"/")).Methods(http.MethodGet)
		router.HandleFunc(webBase+"/", s.wrapPage(indexPageName, true)).Methods(http.MethodGet)
	}
	router.HandleFunc(fmt.Sprintf("%s/login", webBase), s.wrapPage(loginPageName, true)).Methods(http.MethodGet)

	staticPrefix := webBase
	if staticPrefix == "/" {
		staticPrefix = ""
	}
	router.PathPrefix(staticPrefix + "/").Handler(http.StripPrefix(webBase, s.files))
}

// Record stores the request and pushes to websocket clients.
func (s *Service) Record(data *request.RequestData) {
	if s == nil || !s.cfg.Enable {
		return
	}

	record := s.store.Add(data)
	s.hub.Broadcast(map[string]interface{}{
		"type": "request",
		"data": record,
	})
}

// Close releases resources.
func (s *Service) Close() {
	if s == nil {
		return
	}
	if s.cleanupStop != nil {
		close(s.cleanupStop)
		s.cleanupWG.Wait()
		s.cleanupStop = nil
	}
	s.hub.Close()
}

func (s *Service) startSessionCleanup() {
	s.cleanupStop = make(chan struct{})
	s.cleanupWG.Add(1)
	go func() {
		defer s.cleanupWG.Done()
		ticker := time.NewTicker(s.sessionCleanupInterval())
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.auth.Cleanup()
			case <-s.cleanupStop:
				return
			}
		}
	}()
}

func (s *Service) sessionCleanupInterval() time.Duration {
	if s == nil || s.cfg == nil {
		return time.Minute
	}
	timeout := s.cfg.Auth.SessionTimeout
	if timeout <= 0 {
		return time.Minute
	}
	interval := timeout / 2
	if interval < time.Minute {
		interval = time.Minute
	}
	if interval > 10*time.Minute {
		interval = 10 * time.Minute
	}
	return interval
}

func (s *Service) handleRequests(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit := parseIntDefault(query.Get("limit"), defaultListLimit)
	if limit > maxListLimit {
		limit = maxListLimit
	}
	offset := parseIntDefault(query.Get("offset"), 0)

	items, total := s.store.List(ListOptions{
		Search: query.Get("search"),
		Method: query.Get("method"),
		Limit:  limit,
		Offset: offset,
	})

	resp := map[string]interface{}{
		"data":   items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}
	s.respondJSON(w, http.StatusOK, resp)
}

func (s *Service) handleExport(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.Export.Enable {
		http.Error(w, "Export disabled", http.StatusForbidden)
		return
	}

	if s.auth.Enabled() {
		session := s.sessionFromContext(r.Context())
		if session != nil && !s.hasRole(session, roleAdmin) {
			http.Error(w, "Forbidden: export requires admin role", http.StatusForbidden)
			return
		}
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}
	format = strings.ToLower(format)

	if !containsFormat(s.formats, format) {
		http.Error(w, fmt.Sprintf("Unsupported export format: %s", format), http.StatusBadRequest)
		return
	}

	opts := ListOptions{
		Search: r.URL.Query().Get("search"),
		Method: r.URL.Query().Get("method"),
		Limit:  0,
		Offset: 0,
	}
	items, _ := s.store.List(opts)

	data, contentType, ext, err := ExportRequests(items, format)
	if err != nil {
		http.Error(w, "Failed to export data", http.StatusInternalServerError)
		s.logger.Error("Export failed", "error", err)
		return
	}

	filename := fmt.Sprintf("reqtap_requests_%d.%s", time.Now().Unix(), ext)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (s *Service) handleLogin(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	session, err := s.auth.Login(creds.Username, creds.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if s.auth.Enabled() {
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    session.ID,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Expires:  session.ExpiresAt,
			Secure:   r.TLS != nil,
		})
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"username": session.Username,
		"role":     session.Role,
		"expires":  session.ExpiresAt,
	})
}

func (s *Service) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		s.auth.Logout(cookie.Value)
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   -1,
			SameSite: http.SameSiteLaxMode,
			Secure:   r.TLS != nil,
		})
	}

	s.respondJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

func (s *Service) handleMe(w http.ResponseWriter, r *http.Request) {
	session := s.sessionFromContext(r.Context())
	if session == nil {
		session = &Session{Username: "guest", Role: "viewer"}
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"username": session.Username,
		"role":     session.Role,
		"auth":     s.auth.Enabled(),
	})
}

func (s *Service) handleWebsocket(w http.ResponseWriter, r *http.Request) {
	if s.auth.Enabled() {
		if _, err := s.auth.Validate(s.extractToken(r)); err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	if _, err := s.hub.Upgrade(w, r); err != nil {
		s.logger.Error("Failed to upgrade websocket", "error", err)
		return
	}
}

func (s *Service) redirectTo(target string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target, http.StatusPermanentRedirect)
	}
}

func (s *Service) wrapPage(page string, injectConfig bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		webBase := normalizePath(s.cfg.Path)
		if s.auth.Enabled() {
			token := s.extractToken(r)
			_, err := s.auth.Validate(token)
			switch {
			case page == indexPageName && err != nil:
				http.Redirect(w, r, fmt.Sprintf("%s/login", webBase), http.StatusFound)
				return
			case page == loginPageName && err == nil:
				http.Redirect(w, r, webBase, http.StatusFound)
				return
			}
		}

		content, err := fs.ReadFile(s.staticFS, page)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		if injectConfig {
			content = s.injectConfig(content)
		}

		w.Header().Set("Content-Type", contentTypeHTML)
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}
}

func (s *Service) injectConfig(content []byte) []byte {
	configScript := map[string]interface{}{
		"apiBase":        normalizePath(s.cfg.AdminPath),
		"wsEndpoint":     joinPath(s.cfg.AdminPath, "/ws"),
		"exportFormats":  s.formats,
		"authEnabled":    s.auth.Enabled(),
		"webBase":        normalizePath(s.cfg.Path),
		"maxRequests":    s.cfg.MaxRequests,
		"exportEnabled":  s.cfg.Export.Enable,
		"sessionTimeout": s.cfg.Auth.SessionTimeout.String(),
		"roleAdmin":      roleAdmin,
		"roleViewer":     roleViewer,
	}

	payload, _ := json.Marshal(configScript)
	script := fmt.Sprintf(`<script>window.__REQTAP__=%s;</script>`, payload)

	return []byte(strings.Replace(string(content), configPlaceholder, script, 1))
}

func (s *Service) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.auth.Enabled() {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextSessionKey, &Session{Username: "guest", Role: "viewer"})))
			return
		}

		token := s.extractToken(r)
		session, err := s.auth.Validate(token)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), contextSessionKey, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Service) extractToken(r *http.Request) string {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		return cookie.Value
	}

	header := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return strings.TrimSpace(header[7:])
	}

	return ""
}

func (s *Service) sessionFromContext(ctx context.Context) *Session {
	if v := ctx.Value(contextSessionKey); v != nil {
		if session, ok := v.(*Session); ok {
			return session
		}
	}
	return nil
}

func (s *Service) respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		s.logger.Error("Failed to encode JSON response", "error", err)
	}
}

func parseIntDefault(value string, def int) int {
	if value == "" {
		return def
	}

	if parsed, err := strconv.Atoi(value); err == nil {
		return parsed
	}
	return def
}

func normalizePath(p string) string {
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if len(p) > 1 && strings.HasSuffix(p, "/") {
		p = strings.TrimRight(p, "/")
	}
	return p
}

func joinPath(base, sub string) string {
	base = normalizePath(base)
	if !strings.HasPrefix(sub, "/") {
		sub = "/" + sub
	}
	return strings.TrimRight(base, "/") + sub
}

func containsFormat(formats []string, target string) bool {
	for _, f := range formats {
		if f == target {
			return true
		}
	}
	return false
}

func (s *Service) hasRole(session *Session, role string) bool {
	if session == nil {
		return false
	}
	return strings.EqualFold(session.Role, role)
}
