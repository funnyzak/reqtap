package server

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/funnyzak/reqtap/internal/forwarder"
	"github.com/funnyzak/reqtap/internal/logger"
	"github.com/funnyzak/reqtap/internal/printer"
	"github.com/funnyzak/reqtap/internal/web"
	"github.com/funnyzak/reqtap/pkg/request"
)

// Handler HTTP request handler
type Handler struct {
	printer   *printer.ConsolePrinter
	forwarder *forwarder.Forwarder
	logger    logger.Logger
	config    *ServerConfig
	web       *web.Service
}

// ServerConfig server configuration
type ServerConfig struct {
	Port        int
	Path        string
	ForwardURLs []string
	ForwardOpts ForwardOptions
}

// ForwardOptions forwarding options
type ForwardOptions struct {
	Timeout       int // Timeout in seconds
	MaxRetries    int // Maximum retry count
	MaxConcurrent int // Maximum concurrent count
}

// NewHandler creates a new request handler
func NewHandler(
	printer *printer.ConsolePrinter,
	forwarder *forwarder.Forwarder,
	logger logger.Logger,
	config *ServerConfig,
	webService *web.Service,
) *Handler {
	return &Handler{
		printer:   printer,
		forwarder: forwarder,
		logger:    logger,
		config:    config,
		web:       webService,
	}
}

// ServeHTTP implements the http.Handler interface
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Read request body before sending response
	// This prevents "http: invalid Read on closed Body" error
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Failed to read request body", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	r.Body.Close()

	// Send immediate response to client
	h.sendImmediateResponse(w)

	// Process request asynchronously with already read body
	go h.processRequest(r, bodyBytes)
}

// sendImmediateResponse sends immediate response
func (h *Handler) sendImmediateResponse(w http.ResponseWriter) {
	// Set response headers
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Server", "ReqTap/1.0")
	w.Header().Set("Connection", "close")

	// Send status code and content
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))

	// Ensure response is sent immediately
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

// processRequest processes request asynchronously
func (h *Handler) processRequest(r *http.Request, bodyBytes []byte) {
	// Create request record
	record := request.NewRequestData(r, bodyBytes)

	// Persist to web store if enabled
	if h.web != nil {
		h.web.Record(record)
	}

	// Log request
	h.logger.Info("Request received",
		"method", record.Method,
		"path", record.Path,
		"remote_addr", record.RemoteAddr,
		"user_agent", record.UserAgent,
		"content_length", record.ContentLength,
		"content_type", record.ContentType,
	)

	// Execute printing and forwarding concurrently
	var wg sync.WaitGroup

	// Print to console
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := h.printer.PrintRequest(record); err != nil {
			h.logger.Error("Failed to print request", "error", err)
		}
	}()

	// Forward request
	if len(h.config.ForwardURLs) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(),
				time.Duration(h.config.ForwardOpts.Timeout)*time.Second)
			defer cancel()

			if err := h.forwarder.Forward(ctx, record, h.config.ForwardURLs); err != nil {
				h.logger.Error("Failed to forward request", "error", err)
			}
		}()
	}

	wg.Wait()
}

// shouldHandlePath checks if the path should be handled
func (h *Handler) shouldHandlePath(path string) bool {
	if h.config.Path == "/" {
		return true
	}

	return strings.HasPrefix(path, h.config.Path)
}
