package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"github.com/funnyzak/reqtap/internal/config"
	"github.com/funnyzak/reqtap/internal/forwarder"
	"github.com/funnyzak/reqtap/internal/logger"
	"github.com/funnyzak/reqtap/internal/printer"
	"github.com/funnyzak/reqtap/internal/web"
)

// Server HTTP server
type Server struct {
	config       *config.Config
	logger       logger.Logger
	handler      *Handler
	forwarder    forwarder.Client
	printer      printer.Printer
	httpSrv      *http.Server
	web          *web.Service
	baseCtx      context.Context
	cancel       context.CancelFunc
	processingWG *sync.WaitGroup
}

// New creates a new server instance
func New(cfg *config.Config, log logger.Logger) *Server {
	baseCtx, cancel := context.WithCancel(context.Background())
	procWG := &sync.WaitGroup{}
	// Create printer based on output configuration
	var reqPrinter printer.Printer
	if !cfg.Output.Silence {
		reqPrinter = printer.New(strings.ToLower(cfg.Output.Mode), log)
	}

	// Create forwarder
	forwardTimeout := time.Duration(cfg.Forward.Timeout) * time.Second

	forwarder := forwarder.NewForwarder(log, forwarder.Options{
		Timeout:               forwardTimeout,
		Retries:               cfg.Forward.MaxRetries,
		MaxConcurrent:         cfg.Forward.MaxConcurrent,
		MaxIdleConns:          cfg.Forward.MaxIdleConns,
		MaxIdleConnsPerHost:   cfg.Forward.MaxIdleConnsPerHost,
		MaxConnsPerHost:       cfg.Forward.MaxConnsPerHost,
		IdleConnTimeout:       time.Duration(cfg.Forward.IdleConnTimeout) * time.Second,
		ResponseHeaderTimeout: time.Duration(cfg.Forward.ResponseHeaderTimeout) * time.Second,
		TLSHandshakeTimeout:   time.Duration(cfg.Forward.TLSHandshakeTimeout) * time.Second,
		ExpectContinueTimeout: time.Duration(cfg.Forward.ExpectContinueTimeout) * time.Second,
		TLSInsecureSkipVerify: cfg.Forward.TLSInsecureSkipVerify,
		PathStrategy:          buildForwardPathStrategyOptions(cfg),
		HeaderBlacklist:       cfg.Forward.HeaderBlacklist,
		HeaderWhitelist:       cfg.Forward.HeaderWhitelist,
	})

	// Create server configuration
	serverConfig := &ServerConfig{
		Port:         cfg.Server.Port,
		Path:         cfg.Server.Path,
		MaxBodyBytes: cfg.Server.MaxBodyBytes,
		ForwardURLs:  cfg.Forward.URLs,
		ForwardOpts: ForwardOptions{
			Timeout:       cfg.Forward.Timeout,
			MaxRetries:    cfg.Forward.MaxRetries,
			MaxConcurrent: cfg.Forward.MaxConcurrent,
		},
		Responses: convertImmediateResponseConfigs(cfg.Server.Responses),
	}

	// Create web service if enabled
	var webService *web.Service
	if cfg.Web.Enable {
		webService = web.NewService(&cfg.Web, log)
	}

	// Create handler
	handler := NewHandler(reqPrinter, forwarder, log, serverConfig, webService, baseCtx, procWG)

	return &Server{
		config:       cfg,
		logger:       log,
		handler:      handler,
		forwarder:    forwarder,
		printer:      reqPrinter,
		web:          webService,
		baseCtx:      baseCtx,
		cancel:       cancel,
		processingWG: procWG,
	}
}

func convertImmediateResponseConfigs(cfgs []config.ImmediateResponseConfig) []ImmediateResponseRule {
	rules := make([]ImmediateResponseRule, 0, len(cfgs))
	for _, c := range cfgs {
		headers := make(map[string]string, len(c.Headers))
		for k, v := range c.Headers {
			key := http.CanonicalHeaderKey(k)
			headers[key] = v
		}
		rule := ImmediateResponseRule{
			Name:       c.Name,
			Methods:    normalizeMethods(c.Methods),
			Path:       c.Path,
			PathPrefix: c.PathPrefix,
			Status:     c.Status,
			Body:       c.Body,
			Headers:    headers,
		}
		if rule.Name == "" {
			rule.Name = fmt.Sprintf("rule-%d", len(rules)+1)
		}
		if rule.Status == 0 {
			rule.Status = http.StatusOK
		}
		if rule.Headers == nil {
			rule.Headers = map[string]string{}
		}
		rules = append(rules, rule)
	}
	if len(rules) == 0 {
		return []ImmediateResponseRule{{
			Name:   "default-ok",
			Status: http.StatusOK,
			Body:   "ok",
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
		}}
	}
	return rules
}

func normalizeMethods(methods []string) []string {
	if len(methods) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	var normalized []string
	for _, m := range methods {
		upper := strings.ToUpper(strings.TrimSpace(m))
		if upper == "" {
			continue
		}
		if _, ok := seen[upper]; ok {
			continue
		}
		seen[upper] = struct{}{}
		normalized = append(normalized, upper)
	}
	return normalized
}

func buildForwardPathStrategyOptions(cfg *config.Config) forwarder.PathStrategyOptions {
	mode := strings.ToLower(cfg.Forward.PathStrategy.Mode)
	options := forwarder.PathStrategyOptions{
		Mode:        mode,
		StripPrefix: cfg.Forward.PathStrategy.StripPrefix,
		Rules:       convertForwardRewriteRules(cfg.Forward.PathStrategy.Rules),
	}
	if mode == "" {
		return options
	}
	if mode == "strip_prefix" && options.StripPrefix == "" {
		options.StripPrefix = cfg.Server.Path
	}
	return options
}

func convertForwardRewriteRules(cfgRules []config.ForwardRewriteRuleConfig) []forwarder.RewriteRuleOption {
	rules := make([]forwarder.RewriteRuleOption, 0, len(cfgRules))
	for _, rule := range cfgRules {
		rules = append(rules, forwarder.RewriteRuleOption{
			Name:    rule.Name,
			Match:   rule.Match,
			Replace: rule.Replace,
			Regex:   rule.Regex,
		})
	}
	return rules
}

// Start starts the server
func (s *Server) Start() error {
	// Create router
	router := mux.NewRouter()
	if s.web != nil {
		s.web.RegisterRoutes(router)
	}
	router.PathPrefix("/").HandlerFunc(s.handleRequest)

	// Create HTTP server
	s.httpSrv = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.Server.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	s.logger.Info("Starting HTTP server",
		"addr", s.httpSrv.Addr,
		"path", s.config.Server.Path,
	)

	// Start server in goroutine
	go func() {
		if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("Server failed to start", "error", err)
		}
	}()

	// Wait for shutdown signal
	s.waitForShutdown()

	return nil
}

// handleRequest handles HTTP request
func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Check path prefix
	if s.config.Server.Path != "/" && !s.handler.shouldHandlePath(r.URL.Path) {
		http.NotFound(w, r)
		return
	}

	// Call handler
	s.handler.ServeHTTP(w, r)
}

// waitForShutdown waits for shutdown signal
func (s *Server) waitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	s.logger.Info("Shutting down server...")

	// Create shutdown context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if s.cancel != nil {
		s.cancel()
	}

	// Graceful shutdown
	if err := s.httpSrv.Shutdown(ctx); err != nil {
		s.logger.Error("Server forced to shutdown", "error", err)
	}

	if s.processingWG != nil {
		s.processingWG.Wait()
	}

	// Close forwarder
	s.forwarder.Close()
	if s.web != nil {
		s.web.Close()
	}

	s.logger.Info("Server exited")
}

// Stop stops the server
func (s *Server) Stop() error {
	if s.httpSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if s.cancel != nil {
			s.cancel()
		}
		err := s.httpSrv.Shutdown(ctx)
		if s.processingWG != nil {
			s.processingWG.Wait()
		}
		s.forwarder.Close()
		if s.web != nil {
			s.web.Close()
		}
		return err
	}
	return nil
}
