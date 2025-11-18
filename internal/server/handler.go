package server

import (
	"context"
	"errors"
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
	printer   printer.Printer
	forwarder *forwarder.Forwarder
	logger    logger.Logger
	config    *ServerConfig
	web       *web.Service
}

// ServerConfig server configuration
type ServerConfig struct {
	Port         int
	Path         string
	MaxBodyBytes int64
	ForwardURLs  []string
	ForwardOpts  ForwardOptions
	Responses    []ImmediateResponseRule
}

// ForwardOptions forwarding options
type ForwardOptions struct {
	Timeout       int // Timeout in seconds
	MaxRetries    int // Maximum retry count
	MaxConcurrent int // Maximum concurrent count
}

// ImmediateResponseRule describes a runtime response rule
type ImmediateResponseRule struct {
	Name       string
	Methods    []string
	Path       string
	PathPrefix string
	Status     int
	Body       string
	Headers    map[string]string
}

var errRequestBodyTooLarge = errors.New("request body exceeds configured limit")

// NewHandler creates a new request handler
func NewHandler(
	printer printer.Printer,
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
	bodyBytes, err := h.readRequestBody(r)
	if err != nil {
		h.handleBodyReadError(w, err)
		return
	}

	// Send immediate response to client
	responseRule := h.sendImmediateResponse(w, r)

	// Process request asynchronously with already read body
	go h.processRequest(r, bodyBytes, responseRule)
}

// sendImmediateResponse sends immediate response
func (h *Handler) sendImmediateResponse(w http.ResponseWriter, r *http.Request) *ImmediateResponseRule {
	responseRule := h.selectResponseRule(r)
	statusCode := http.StatusOK
	body := []byte("ok")
	defaultContentType := "text/plain"

	if responseRule != nil {
		statusCode = responseRule.Status
		body = []byte(responseRule.Body)
		hasContentType := false
		for key, value := range responseRule.Headers {
			if key == "" {
				continue
			}
			w.Header().Set(key, value)
			if strings.EqualFold(key, "Content-Type") {
				hasContentType = true
			}
		}
		if !hasContentType {
			w.Header().Set("Content-Type", defaultContentType)
		}
		h.logger.Debug("Immediate mock response applied",
			"rule", responseRule.Name,
			"status", responseRule.Status,
			"method", r.Method,
			"path", r.URL.Path,
		)
	} else {
		w.Header().Set("Content-Type", defaultContentType)
	}

	w.Header().Set("Server", "ReqTap/1.0")
	w.WriteHeader(statusCode)
	if len(body) > 0 {
		w.Write(body)
	}

	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	return responseRule
}

func (h *Handler) selectResponseRule(r *http.Request) *ImmediateResponseRule {
	if len(h.config.Responses) == 0 {
		return nil
	}

	path := r.URL.Path
	method := strings.ToUpper(r.Method)

	for i := range h.config.Responses {
		rule := &h.config.Responses[i]
		if len(rule.Methods) > 0 {
			matched := false
			for _, allowed := range rule.Methods {
				if method == allowed {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		if rule.Path != "" && rule.Path != path {
			continue
		}

		if rule.PathPrefix != "" && !strings.HasPrefix(path, rule.PathPrefix) {
			continue
		}

		return rule
	}

	return nil
}

// processRequest processes request asynchronously
func (h *Handler) processRequest(r *http.Request, bodyBytes []byte, responseRule *ImmediateResponseRule) {
	// Create request record
	record := request.NewRequestData(r, bodyBytes)
	record.MockResponse = h.toMockResponseSummary(responseRule)

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
		"mock_rule", record.MockResponse.Rule,
		"mock_status", record.MockResponse.Status,
	)

	// Execute printing and forwarding concurrently
	var wg sync.WaitGroup

	// Print to console
	if h.printer != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := h.printer.PrintRequest(record); err != nil {
				h.logger.Error("Failed to print request", "error", err)
			}
		}()
	}

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

func (h *Handler) toMockResponseSummary(rule *ImmediateResponseRule) request.MockResponse {
	if rule == nil {
		return request.MockResponse{Status: http.StatusOK}
	}

	return request.MockResponse{
		Rule:   rule.Name,
		Status: rule.Status,
	}
}

func (h *Handler) readRequestBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()

	if h.config.MaxBodyBytes <= 0 {
		return io.ReadAll(r.Body)
	}

	limited := io.LimitReader(r.Body, h.config.MaxBodyBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > h.config.MaxBodyBytes {
		return nil, errRequestBodyTooLarge
	}
	return body, nil
}

func (h *Handler) handleBodyReadError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errRequestBodyTooLarge):
		h.logger.Warn("Request body exceeds configured limit",
			"limit_bytes", h.config.MaxBodyBytes,
		)
		http.Error(w, "Payload Too Large", http.StatusRequestEntityTooLarge)
	default:
		h.logger.Error("Failed to read request body", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// shouldHandlePath checks if the path should be handled
func (h *Handler) shouldHandlePath(path string) bool {
	if h.config.Path == "/" {
		return true
	}

	return strings.HasPrefix(path, h.config.Path)
}
