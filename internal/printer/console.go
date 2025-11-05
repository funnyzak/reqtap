package printer

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/funnyzak/reqtap/internal/logger"
	"github.com/funnyzak/reqtap/pkg/request"
)

// ColorScheme color scheme
type ColorScheme struct {
	MethodGET      *color.Color
	MethodPOST     *color.Color
	MethodPUT      *color.Color
	MethodDELETE   *color.Color
	MethodPATCH    *color.Color
	HeaderKey      *color.Color
	HeaderValue    *color.Color
	Separator      *color.Color
	Timestamp      *color.Color
	BodyContent    *color.Color
	BinaryNotice   *color.Color
	TruncateNotice *color.Color
	RemoteAddr     *color.Color
	Query          *color.Color
}

// NewColorScheme creates a new color scheme
func NewColorScheme() *ColorScheme {
	return &ColorScheme{
		MethodGET:      color.New(color.FgBlue, color.Bold),
		MethodPOST:     color.New(color.FgGreen, color.Bold),
		MethodPUT:      color.New(color.FgYellow, color.Bold),
		MethodDELETE:   color.New(color.FgRed, color.Bold),
		MethodPATCH:    color.New(color.FgMagenta, color.Bold),
		HeaderKey:      color.New(color.FgCyan),
		HeaderValue:    color.New(color.FgWhite),
		Separator:      color.New(color.FgYellow, color.Bold),
		Timestamp:      color.New(color.FgHiBlack),
		BodyContent:    color.New(color.FgWhite),
		BinaryNotice:   color.New(color.FgHiRed, color.Bold),
		TruncateNotice: color.New(color.FgHiYellow, color.Bold),
		RemoteAddr:     color.New(color.FgHiBlue),
		Query:          color.New(color.FgHiMagenta),
	}
}

// ConsolePrinter console printer
type ConsolePrinter struct {
	colorScheme *ColorScheme
	logger      logger.Logger
}

// NewConsolePrinter creates a new console printer
func NewConsolePrinter(logger logger.Logger) *ConsolePrinter {
	return &ConsolePrinter{
		colorScheme: NewColorScheme(),
		logger:      logger,
	}
}

// PrintRequest prints request information to console
func (p *ConsolePrinter) PrintRequest(data *request.RequestData) error {
	scheme := p.colorScheme

	// Separator and timestamp
	timestamp := data.Timestamp.Format("2006-01-02T15:04:05-07:00")
	scheme.Separator.Printf("═ INCOMING REQUEST (%s) ═\n", timestamp)

	// Request line
	methodColor := p.getMethodColor(data.Method)
	methodColor.Printf("[%s] %s", data.Method, data.Path)

	// Query parameters
	if data.Query != "" {
		scheme.Query.Printf("?%s", data.Query)
	}

	// Source address
	fmt.Printf(" ")
	scheme.RemoteAddr.Printf("[FROM: %s]\n", data.RemoteAddr)

	// User-Agent
	if data.UserAgent != "" {
		fmt.Printf("User-Agent: %s\n", data.UserAgent)
	}

	// Headers
	if len(data.Headers) > 0 {
		scheme.Separator.Println("─ Headers ─")
		p.printHeaders(data.Headers)
	}

	// Body
	scheme.Separator.Println("─ Body ─")
	p.printBody(data)

	// End separator
	scheme.Separator.Println("═ END OF REQUEST ═")
	fmt.Println() // Empty line separator

	return nil
}

// getMethodColor gets the corresponding color based on HTTP method
func (p *ConsolePrinter) getMethodColor(method string) *color.Color {
	switch strings.ToUpper(method) {
	case "GET":
		return p.colorScheme.MethodGET
	case "POST":
		return p.colorScheme.MethodPOST
	case "PUT":
		return p.colorScheme.MethodPUT
	case "DELETE":
		return p.colorScheme.MethodDELETE
	case "PATCH":
		return p.colorScheme.MethodPATCH
	default:
		return color.New(color.FgWhite, color.Bold)
	}
}

// printHeaders prints request headers
func (p *ConsolePrinter) printHeaders(headers http.Header) {
	for key, values := range headers {
		lowerKey := strings.ToLower(key)

		// Sensitive information handling
		if p.isSensitiveHeader(lowerKey) {
			p.colorScheme.HeaderKey.Printf("%s: ", key)
			fmt.Println("[REDACTED]")
			continue
		}

		// Skip some less important headers
		if p.shouldSkipHeader(lowerKey) {
			continue
		}

		p.colorScheme.HeaderKey.Printf("%s: ", key)
		p.colorScheme.HeaderValue.Println(strings.Join(values, ", "))
	}
}

// isSensitiveHeader checks if it's sensitive header information
func (p *ConsolePrinter) isSensitiveHeader(key string) bool {
	sensitiveHeaders := map[string]bool{
		"authorization":   true,
		"cookie":          true,
		"set-cookie":      true,
		"x-api-key":       true,
		"x-auth-token":    true,
		"x-csrf-token":    true,
		"x-session-token": true,
	}
	return sensitiveHeaders[key]
}

// shouldSkipHeader checks if header should be skipped from display
func (p *ConsolePrinter) shouldSkipHeader(key string) bool {
	skipHeaders := map[string]bool{
		"connection":        true,
		"keep-alive":        true,
		"proxy-connection":  true,
		"te":                true,
		"trailer":           true,
		"transfer-encoding": true,
		"upgrade":           true,
	}
	return skipHeaders[key]
}

// printBody prints request body
func (p *ConsolePrinter) printBody(data *request.RequestData) {
	if len(data.Body) == 0 {
		fmt.Println("[Empty Body]")
		return
	}

	// Check if it's binary content
	if data.IsBinary {
		p.colorScheme.BinaryNotice.Printf(
			"[Binary Body: %s, %s. Content skipped.]\n",
			data.ContentType,
			humanize.Bytes(uint64(len(data.Body))),
		)
		return
	}

	// Check if truncation is needed
	const maxPrintSize = 4 * 1024 // 4KB
	if len(data.Body) > maxPrintSize {
		p.printBodyContent(data.Body[:maxPrintSize])
		p.colorScheme.TruncateNotice.Printf(
			"\n[... Body truncated. Total size: %s.]\n",
			humanize.Bytes(uint64(len(data.Body))),
		)
	} else {
		p.printBodyContent(data.Body)
	}
}

// printBodyContent prints body content with formatting
func (p *ConsolePrinter) printBodyContent(body []byte) {
	content := string(body)

	// Try to detect content type and format
	if p.isJSONContent(content) {
		// JSON content
		p.colorScheme.BodyContent.Println(content)
	} else if p.isXMLContent(content) {
		// XML content
		p.colorScheme.BodyContent.Println(content)
	} else if p.isFormContent(content) {
		// Form content
		p.colorScheme.BodyContent.Println(content)
	} else {
		// Other text content
		p.colorScheme.BodyContent.Print(content)
	}

	// Ensure ending with newline
	if len(body) > 0 && body[len(body)-1] != '\n' {
		fmt.Println()
	}
}

// isJSONContent detects if it's JSON content
func (p *ConsolePrinter) isJSONContent(content string) bool {
	trimmed := strings.TrimSpace(content)
	return strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") ||
		strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")
}

// isXMLContent detects if it's XML content
func (p *ConsolePrinter) isXMLContent(content string) bool {
	trimmed := strings.TrimSpace(content)
	return strings.HasPrefix(trimmed, "<") && strings.HasSuffix(trimmed, ">")
}

// isFormContent detects if it's form content
func (p *ConsolePrinter) isFormContent(content string) bool {
	return strings.Contains(content, "=") &&
		(strings.Contains(content, "&") || !strings.Contains(content, " "))
}

// isBinaryContent detects if it's binary content (duplicate logic for consistency)
func (p *ConsolePrinter) isBinaryContent(contentType string, body []byte) bool {
	// Check Content-Type
	binaryTypes := []string{
		"image/", "video/", "audio/",
		"application/octet-stream",
		"application/zip", "application/gzip",
		"application/pdf", "application/msword",
		"application/vnd.ms-", "application/vnd.openxmlformats-",
	}

	for _, binaryType := range binaryTypes {
		if len(contentType) >= len(binaryType) && contentType[:len(binaryType)] == binaryType {
			return true
		}
	}

	// Check UTF-8 validity
	if !utf8.Valid(body) {
		return true
	}

	// Check null byte ratio
	nullCount := bytes.Count(body, []byte{0})
	if len(body) > 0 && nullCount > len(body)/10 { // More than 10% are null bytes
		return true
	}

	return false
}
