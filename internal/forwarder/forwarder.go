package forwarder

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/funnyzak/reqtap/internal/logger"
	"github.com/funnyzak/reqtap/pkg/request"
)

// Forwarder request forwarder
type Forwarder struct {
	client        *http.Client
	logger        logger.Logger
	timeout       time.Duration
	retries       int
	maxConcurrent int
	workerPool    chan struct{}
	mu            sync.Mutex
	cond          *sync.Cond
	closed        bool
	activeCalls   int
	pathStrategy  *pathStrategy
}

type pathStrategyMode string

const (
	pathModeAppend      pathStrategyMode = "append"
	pathModeStripPrefix pathStrategyMode = "strip_prefix"
	pathModeRewrite     pathStrategyMode = "rewrite"
)

// Options 转发器配置
type Options struct {
	Timeout               time.Duration
	Retries               int
	MaxConcurrent         int
	MaxIdleConns          int
	MaxIdleConnsPerHost   int
	MaxConnsPerHost       int
	IdleConnTimeout       time.Duration
	ResponseHeaderTimeout time.Duration
	TLSHandshakeTimeout   time.Duration
	ExpectContinueTimeout time.Duration
	TLSInsecureSkipVerify bool
	PathStrategy          PathStrategyOptions
}

// PathStrategyOptions configures how request paths are rewritten before forwarding
type PathStrategyOptions struct {
	Mode        string
	StripPrefix string
	Rules       []RewriteRuleOption
}

// RewriteRuleOption describes a single rewrite rule definition
type RewriteRuleOption struct {
	Name    string
	Match   string
	Replace string
	Regex   bool
}

// ErrForwarderClosed indicates the forwarder has been shut down.
var ErrForwarderClosed = errors.New("forwarder is closed")

// NewForwarder creates new forwarder
func NewForwarder(logger logger.Logger, opts Options) *Forwarder {
	if opts.MaxConcurrent <= 0 {
		opts.MaxConcurrent = 10 // 默认并发控制
	}

	transport := &http.Transport{
		MaxIdleConns:        positiveOrDefault(opts.MaxIdleConns, 200),
		MaxIdleConnsPerHost: positiveOrDefault(opts.MaxIdleConnsPerHost, opts.MaxConcurrent),
		MaxConnsPerHost:     positiveOrDefault(opts.MaxConnsPerHost, opts.MaxConcurrent*2),
		IdleConnTimeout:     durationOrDefault(opts.IdleConnTimeout, 90*time.Second),
		ResponseHeaderTimeout: durationOrDefault(
			opts.ResponseHeaderTimeout,
			15*time.Second,
		),
		TLSHandshakeTimeout:   durationOrDefault(opts.TLSHandshakeTimeout, 10*time.Second),
		ExpectContinueTimeout: durationOrDefault(opts.ExpectContinueTimeout, 1*time.Second),
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: opts.TLSInsecureSkipVerify,
		},
	}

	f := &Forwarder{
		client: &http.Client{
			Timeout:   opts.Timeout,
			Transport: transport,
		},
		logger:        logger,
		timeout:       opts.Timeout,
		retries:       opts.Retries,
		maxConcurrent: opts.MaxConcurrent,
		workerPool:    make(chan struct{}, opts.MaxConcurrent),
		pathStrategy:  newPathStrategy(opts.PathStrategy, logger),
	}
	f.cond = sync.NewCond(&f.mu)
	return f
}

// Forward forwards request to all configured URLs
func (f *Forwarder) Forward(ctx context.Context, data *request.RequestData, urls []string) error {
	if len(urls) == 0 {
		return nil
	}

	f.mu.Lock()
	if f.closed {
		f.mu.Unlock()
		return ErrForwarderClosed
	}
	f.activeCalls++
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		f.activeCalls--
		if f.activeCalls == 0 {
			f.cond.Broadcast()
		}
		f.mu.Unlock()
	}()

	// Concurrently forward to all target URLs
	var wg sync.WaitGroup
	for _, url := range urls {
		wg.Add(1)
		go func(targetURL string) {
			defer wg.Done()

			// Get worker token (control concurrent count)
			f.workerPool <- struct{}{}
			defer func() { <-f.workerPool }()

			f.forwardToURL(ctx, data, targetURL)
		}(url)
	}

	wg.Wait()
	return nil
}

// forwardToURL forwards request to single URL (with retry)
func (f *Forwarder) forwardToURL(ctx context.Context, data *request.RequestData, targetURL string) {
	var lastErr error

	for attempt := 0; attempt <= f.retries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			if backoff > 30*time.Second {
				backoff = 30 * time.Second // Maximum backoff time
			}

			select {
			case <-ctx.Done():
				f.logger.Info("Forward cancelled by context",
					"url", targetURL,
					"attempt", attempt+1,
				)
				return
			case <-time.After(backoff):
				// Continue retry
			}
		}

		err := f.doForward(ctx, data, targetURL, attempt)
		if err == nil {
			f.logger.Info("Request forwarded successfully",
				"url", targetURL,
				"method", data.Method,
				"path", data.Path,
				"attempt", attempt+1,
			)
			return
		}

		lastErr = err
		f.logger.Warn("Forward attempt failed",
			"url", targetURL,
			"error", err.Error(),
			"attempt", attempt+1,
		)
	}

	f.logger.Error("All forward attempts failed",
		"url", targetURL,
		"final_error", lastErr.Error(),
		"total_attempts", f.retries+1,
	)
}

// doForward executes single forward
func (f *Forwarder) doForward(ctx context.Context, data *request.RequestData, targetURL string, attempt int) error {
	resolvedPath := data.Path
	var appliedRule string
	if f.pathStrategy != nil {
		resolvedPath, appliedRule = f.pathStrategy.resolve(data.Path)
	}
	// Build target URL
	targetURL = strings.TrimSuffix(targetURL, "/") + resolvedPath
	if data.Query != "" {
		targetURL += "?" + data.Query
	}
	if appliedRule != "" {
		f.logger.Debug("Forward path strategy applied",
			"rule", appliedRule,
			"original_path", data.Path,
			"resolved_path", resolvedPath,
			"url", targetURL,
		)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, data.Method, targetURL, bytes.NewReader(data.Body))
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	// Copy Headers (filter some headers that should not be forwarded)
	for key, values := range data.Headers {
		if f.shouldForwardHeader(key) {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}

	// Set X-Forwarded-* headers
	req.Header.Set("X-Forwarded-For", data.RemoteAddr)
	req.Header.Set("X-Forwarded-Proto", "http")
	req.Header.Set("X-ReqTap-Original-Host", data.Headers.Get("Host"))
	req.Header.Set("X-ReqTap-Forward-Attempt", fmt.Sprintf("%d", attempt+1))

	// Send request
	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			f.logger.Warn("Failed to close response body", "error", cerr)
		}
	}()

	// Read response (avoid connection pool issues)
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		f.logger.Warn("Failed to read response body", "error", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("target returned status %d", resp.StatusCode)
	}

	return nil
}

// shouldForwardHeader determines if specified header should be forwarded
func (f *Forwarder) shouldForwardHeader(key string) bool {
	lowerKey := strings.ToLower(key)

	// Headers that should not be forwarded
	skipHeaders := map[string]bool{
		"host":                true, // Automatically set
		"connection":          true, // Connection related
		"keep-alive":          true,
		"proxy-authenticate":  true,
		"proxy-authorization": true,
		"te":                  true,
		"trailers":            true,
		"transfer-encoding":   true,
		"upgrade":             true,
		"content-length":      true, // Will be automatically recalculated
	}

	if skipHeaders[lowerKey] {
		return false
	}

	// For some sensitive headers, log warning but still forward (can be adjusted as needed)
	sensitiveHeaders := map[string]bool{
		"authorization": true,
		"cookie":        true,
		"set-cookie":    true,
	}

	if sensitiveHeaders[lowerKey] {
		f.logger.Debug("Forwarding sensitive header", "header", key)
	}

	return true
}

// SetMaxConcurrent sets maximum concurrent count
func (f *Forwarder) SetMaxConcurrent(maxConcurrent int) {
	if maxConcurrent <= 0 {
		maxConcurrent = 1
	}

	f.maxConcurrent = maxConcurrent
	newWorkerPool := make(chan struct{}, maxConcurrent)
	close(f.workerPool)
	f.workerPool = newWorkerPool
}

// GetMaxConcurrent gets current maximum concurrent count
func (f *Forwarder) GetMaxConcurrent() int {
	return f.maxConcurrent
}

// Close closes forwarder and cleans up resources
func (f *Forwarder) Close() {
	f.mu.Lock()
	if f.closed {
		f.mu.Unlock()
		return
	}
	f.closed = true
	for f.activeCalls > 0 {
		f.cond.Wait()
	}
	f.mu.Unlock()

	close(f.workerPool)

	// Close idle connections of HTTP client
	if transport, ok := f.client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
}

func positiveOrDefault(value, def int) int {
	if value > 0 {
		return value
	}
	return def
}

func durationOrDefault(value, def time.Duration) time.Duration {
	if value > 0 {
		return value
	}
	return def
}

type pathStrategy struct {
	mode        pathStrategyMode
	stripPrefix string
	rules       []rewriteRule
}

type rewriteRule struct {
	name    string
	match   string
	replace string
	regex   bool
	expr    *regexp.Regexp
}

func newPathStrategy(opts PathStrategyOptions, log logger.Logger) *pathStrategy {
	mode := pathStrategyMode(strings.ToLower(opts.Mode))
	if mode == "" {
		mode = pathModeAppend
	}

	switch mode {
	case pathModeAppend:
		return nil
	case pathModeStripPrefix:
		prefix := normalizeStripPrefix(opts.StripPrefix)
		if prefix == "" {
			return nil
		}
		return &pathStrategy{mode: mode, stripPrefix: prefix}
	case pathModeRewrite:
		rules := buildRewriteRules(opts.Rules, log)
		if len(rules) == 0 {
			return nil
		}
		return &pathStrategy{mode: mode, rules: rules}
	default:
		return nil
	}
}

func (ps *pathStrategy) resolve(inputPath string) (string, string) {
	if ps == nil {
		return normalizeURLPath(inputPath), ""
	}

	switch ps.mode {
	case pathModeStripPrefix:
		cleanPath := normalizeURLPath(inputPath)
		if ps.stripPrefix != "" && ps.stripPrefix != "/" && strings.HasPrefix(cleanPath, ps.stripPrefix) {
			trimmed := strings.TrimPrefix(cleanPath, ps.stripPrefix)
			if trimmed == "" {
				trimmed = "/"
			}
			return normalizeURLPath(trimmed), string(ps.mode)
		}
		return cleanPath, ""
	case pathModeRewrite:
		cleanPath := normalizeURLPath(inputPath)
		for _, rule := range ps.rules {
			if rule.regex {
				if rule.expr == nil || !rule.expr.MatchString(cleanPath) {
					continue
				}
				replaced := rule.expr.ReplaceAllString(cleanPath, rule.replace)
				return normalizeURLPath(replaced), rule.name
			}
			if rule.match != "" && strings.HasPrefix(cleanPath, rule.match) {
				remainder := strings.TrimPrefix(cleanPath, rule.match)
				newPath := concatURLPath(rule.replace, remainder)
				return normalizeURLPath(newPath), rule.name
			}
		}
		return cleanPath, ""
	default:
		return normalizeURLPath(inputPath), ""
	}
}

func normalizeStripPrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" || prefix == "/" {
		return ""
	}
	return normalizeURLPath(prefix)
}

func buildRewriteRules(options []RewriteRuleOption, log logger.Logger) []rewriteRule {
	var rules []rewriteRule
	for idx, opt := range options {
		rule := rewriteRule{
			name:    opt.Name,
			match:   strings.TrimSpace(opt.Match),
			replace: strings.TrimSpace(opt.Replace),
			regex:   opt.Regex,
		}
		if rule.name == "" {
			rule.name = fmt.Sprintf("rewrite_rule_%d", idx+1)
		}
		if rule.regex {
			if rule.match == "" {
				continue
			}
			expr, err := regexp.Compile(rule.match)
			if err != nil {
				if log != nil {
					log.Warn("Invalid rewrite regex skipped", "rule", rule.name, "error", err)
				}
				continue
			}
			rule.expr = expr
			if rule.replace == "" {
				rule.replace = "/"
			}
		} else {
			rule.match = normalizeURLPath(rule.match)
			if rule.match == "/" {
				continue
			}
			if rule.replace == "" {
				rule.replace = "/"
			}
			rule.replace = normalizeURLPath(rule.replace)
		}
		rules = append(rules, rule)
	}
	return rules
}

func normalizeURLPath(p string) string {
	if p == "" {
		return "/"
	}
	cleaned := path.Clean(p)
	if cleaned == "." {
		cleaned = "/"
	}
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	return cleaned
}

func concatURLPath(base, remainder string) string {
	base = normalizeURLPath(base)
	remainder = strings.TrimLeft(remainder, "/")
	if remainder == "" {
		return base
	}
	if base == "/" {
		return normalizeURLPath("/" + remainder)
	}
	return normalizeURLPath(base + "/" + remainder)
}
