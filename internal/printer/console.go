package printer

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"unicode/utf8"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/funnyzak/reqtap/internal/logger"
	"github.com/funnyzak/reqtap/pkg/request"
	"golang.org/x/term"
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

// Global request counter
var requestCounter uint64

// ConsolePrinter console printer
type ConsolePrinter struct {
	colorScheme *ColorScheme
	logger      logger.Logger
	out         io.Writer
}

// getTerminalWidth gets the current terminal width with fallback
func (p *ConsolePrinter) getTerminalWidth() int {
	if testWidth := os.Getenv("REQTAP_TEST_WIDTH"); testWidth != "" {
		if width, err := strconv.Atoi(testWidth); err == nil {
			switch {
			case width < 40:
				return 40
			case width > 150:
				return 150
			default:
				return width
			}
		}
	}

	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80
	}

	if width < 40 {
		return 40
	}
	if width > 150 {
		return 150
	}

	return width
}

// wrapText wraps text to fit within the specified width, preserving words
func (p *ConsolePrinter) wrapText(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)

	if len(words) == 0 {
		return []string{""}
	}

	currentLine := words[0]
	currentWidth := utf8.RuneCountInString(currentLine)

	for _, word := range words[1:] {
		wordWidth := utf8.RuneCountInString(word)

		if currentWidth+1+wordWidth > maxWidth {
			lines = append(lines, currentLine)
			currentLine = word
			currentWidth = wordWidth
		} else {
			if currentLine != "" {
				currentLine += " " + word
				currentWidth += 1 + wordWidth
			} else {
				currentLine = word
				currentWidth = wordWidth
			}
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// NewConsolePrinter creates a new console printer
func NewConsolePrinter(logger logger.Logger) *ConsolePrinter {
	return &ConsolePrinter{
		colorScheme: NewColorScheme(),
		logger:      logger,
		out:         os.Stdout,
	}
}

// PrintRequest prints request information using raw HTTP message layout
func (p *ConsolePrinter) PrintRequest(data *request.RequestData) error {
	requestNum := atomic.AddUint64(&requestCounter, 1)
	timestamp := data.Timestamp.Format("2006-01-02T15:04:05-07:00")
	width := p.getTerminalWidth()

	var builder strings.Builder
	p.printSummary(&builder, requestNum, timestamp, data, width)
	p.printRequestLine(&builder, data)
	p.printHeaders(&builder, data.Headers, width)
	builder.WriteString("\n")
	p.printBody(&builder, data)
	builder.WriteString("\n\n")

	_, err := fmt.Fprint(p.out, builder.String())
	return err
}

func (p *ConsolePrinter) printSummary(builder *strings.Builder, requestNum uint64, timestamp string, data *request.RequestData, width int) {
	separator := p.buildSeparator(width)
	builder.WriteString(p.colorScheme.Separator.Sprint(separator))
	builder.WriteString("\n")
	builder.WriteString(p.colorScheme.Separator.Sprintf("Request #%d  %s\n", requestNum, timestamp))
	p.printMetadataLine(builder, data)
	builder.WriteString(p.colorScheme.Separator.Sprint(separator))
	builder.WriteString("\n\n")
}

func (p *ConsolePrinter) buildSeparator(width int) string {
	if width < 40 {
		width = 40
	}
	if width > 150 {
		width = 150
	}
	return strings.Repeat("-", width)
}

func (p *ConsolePrinter) printMetadataLine(builder *strings.Builder, data *request.RequestData) {
	first := true
	addSep := func() {
		if first {
			first = false
			return
		}
		builder.WriteString(" | ")
	}

	if data.RemoteAddr != "" {
		addSep()
		builder.WriteString("Remote: ")
		builder.WriteString(p.colorScheme.RemoteAddr.Sprint(data.RemoteAddr))
	}

	if data.UserAgent != "" {
		addSep()
		builder.WriteString("UA: ")
		builder.WriteString(p.colorScheme.BodyContent.Sprint(data.UserAgent))
	}

	if data.ContentType != "" {
		addSep()
		builder.WriteString("Content-Type: ")
		builder.WriteString(p.colorScheme.HeaderValue.Sprint(data.ContentType))
	}

	addSep()
	builder.WriteString("Size: ")
	builder.WriteString(p.colorScheme.BodyContent.Sprint(humanize.Bytes(uint64(len(data.Body)))))
	builder.WriteString("\n")
}

func (p *ConsolePrinter) printRequestLine(builder *strings.Builder, data *request.RequestData) {
	method := strings.ToUpper(data.Method)
	proto := p.defaultProto(data.Proto)
	path := data.Path
	if path == "" {
		path = "/"
	}

	methodColor := p.getMethodColor(method)
	builder.WriteString(methodColor.Sprintf("%s ", method))
	builder.WriteString(path)

	if data.Query != "" {
		builder.WriteString("?")
		builder.WriteString(p.colorScheme.Query.Sprint(data.Query))
	}

	builder.WriteString(" ")
	builder.WriteString(proto)
	builder.WriteString("\n")
}

func (p *ConsolePrinter) defaultProto(proto string) string {
	if proto != "" {
		return proto
	}
	return "HTTP/1.1"
}

func (p *ConsolePrinter) printHeaders(builder *strings.Builder, headers http.Header, width int) {
	if len(headers) == 0 {
		return
	}

	keys := make([]string, 0, len(headers))
	for key := range headers {
		lowerKey := strings.ToLower(key)
		if p.shouldSkipHeader(lowerKey) {
			continue
		}
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		lowerKey := strings.ToLower(key)
		values := headers[key]
		displayValue := strings.Join(values, ", ")

		if p.isSensitiveHeader(lowerKey) {
			displayValue = "[REDACTED]"
		}

		p.printHeaderLine(builder, key, displayValue, width)
	}
}

func (p *ConsolePrinter) printHeaderLine(builder *strings.Builder, key, value string, width int) {
	if width <= 0 {
		width = 80
	}

	prefix := key + ": "
	available := width - utf8.RuneCountInString(prefix)
	if available < 20 {
		available = 20
	}

	wrappedValues := p.wrapText(value, available)
	if len(wrappedValues) == 0 {
		wrappedValues = []string{""}
	}

	builder.WriteString(p.colorScheme.HeaderKey.Sprint(prefix))
	builder.WriteString(p.colorScheme.HeaderValue.Sprintln(wrappedValues[0]))

	indent := strings.Repeat(" ", utf8.RuneCountInString(prefix))
	for _, line := range wrappedValues[1:] {
		builder.WriteString(indent)
		builder.WriteString(p.colorScheme.HeaderValue.Sprintln(line))
	}
}

func (p *ConsolePrinter) printBody(builder *strings.Builder, data *request.RequestData) {
	bodySize := humanize.Bytes(uint64(len(data.Body)))

	if len(data.Body) == 0 {
		builder.WriteString(p.colorScheme.BodyContent.Sprintf("[Empty Body - %s]", bodySize))
		builder.WriteString("\n")
		return
	}

	if data.IsBinary {
		builder.WriteString(p.colorScheme.BinaryNotice.Sprintf("[Binary Body: %s, %s. Content skipped.]", data.ContentType, bodySize))
		builder.WriteString("\n")
		return
	}

	p.printBodyContent(builder, data.Body)
}

func (p *ConsolePrinter) printBodyContent(builder *strings.Builder, body []byte) {
	content := string(body)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		trimmed := strings.TrimRight(line, "\r")
		if trimmed == "" {
			builder.WriteString("\n")
			continue
		}
		builder.WriteString(p.colorScheme.BodyContent.Sprintln(trimmed))
	}
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
