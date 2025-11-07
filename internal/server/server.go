package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
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
	config    *config.Config
	logger    logger.Logger
	handler   *Handler
	forwarder *forwarder.Forwarder
	printer   *printer.ConsolePrinter
	httpSrv   *http.Server
	web       *web.Service
}

// New creates a new server instance
func New(cfg *config.Config, log logger.Logger) *Server {
	// Create console printer
	printer := printer.NewConsolePrinter(log)

	// Create forwarder
	forwardTimeout := time.Duration(cfg.Forward.Timeout) * time.Second

	forwarder := forwarder.NewForwarder(
		log,
		forwardTimeout,
		cfg.Forward.MaxRetries,
		cfg.Forward.MaxConcurrent,
	)

	// Create server configuration
	serverConfig := &ServerConfig{
		Port:        cfg.Server.Port,
		Path:        cfg.Server.Path,
		ForwardURLs: cfg.Forward.URLs,
		ForwardOpts: ForwardOptions{
			Timeout:       30, // Default 30 seconds
			MaxRetries:    cfg.Forward.MaxRetries,
			MaxConcurrent: cfg.Forward.MaxConcurrent,
		},
	}

	// Create web service if enabled
	var webService *web.Service
	if cfg.Web.Enable {
		webService = web.NewService(&cfg.Web, log)
	}

	// Create handler
	handler := NewHandler(printer, forwarder, log, serverConfig, webService)

	return &Server{
		config:    cfg,
		logger:    log,
		handler:   handler,
		forwarder: forwarder,
		printer:   printer,
		web:       webService,
	}
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

	// Graceful shutdown
	if err := s.httpSrv.Shutdown(ctx); err != nil {
		s.logger.Error("Server forced to shutdown", "error", err)
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
		err := s.httpSrv.Shutdown(ctx)
		if s.web != nil {
			s.web.Close()
		}
		return err
	}
	return nil
}
