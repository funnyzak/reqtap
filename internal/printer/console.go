package printer

import (
	"fmt"
	"net/http"
	"os"
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
}

// getTerminalWidth gets the current terminal width with fallback
func (p *ConsolePrinter) getTerminalWidth() int {
	// Check for test environment variable first
	if testWidth := os.Getenv("REQTAP_TEST_WIDTH"); testWidth != "" {
		if width, err := fmt.Sscanf(testWidth, "%d", new(int)); err == nil && width > 0 {
			testW := 0
			fmt.Sscanf(testWidth, "%d", &testW)
			if testW < 40 {
				return 40
			}
			if testW > 150 {
				return 150
			}
			return testW
		}
	}

	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// Fallback to common terminal width
		return 80
	}

	// Ensure absolute minimum width for basic formatting
	if width < 40 {
		return 40
	}

	// Cap at reasonable maximum to prevent extremely long lines
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

		// If adding a space and the word exceeds max width, start a new line
		if currentWidth+1+wordWidth > maxWidth {
			lines = append(lines, currentLine)
			currentLine = word
			currentWidth = wordWidth
		} else {
			// Add word to current line
			if currentLine != "" {
				currentLine += " " + word
				currentWidth += 1 + wordWidth
			} else {
				currentLine = word
				currentWidth = wordWidth
			}
		}
	}

	// Add the last line
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// wrapTextWithIndent wraps text with the specified indentation
func (p *ConsolePrinter) wrapTextWithIndent(text string, maxWidth, indent int) []string {
	if indent >= maxWidth {
		// If indent is too large, just return the text as is
		return []string{text}
	}

	availableWidth := maxWidth - indent
	wrappedLines := p.wrapText(text, availableWidth)

	// Add indentation to all lines except the first
	result := make([]string, len(wrappedLines))
	if len(wrappedLines) > 0 {
		result[0] = wrappedLines[0]
		for i := 1; i < len(wrappedLines); i++ {
			result[i] = strings.Repeat(" ", indent) + wrappedLines[i]
		}
	}

	return result
}

// isCompactMode returns true if terminal width requires compact display
func (p *ConsolePrinter) isCompactMode(width int) bool {
	return width < 60
}

// getIndent returns the appropriate indent size based on terminal width
func (p *ConsolePrinter) getIndent(width int) int {
	if p.isCompactMode(width) {
		return 1 // Compact mode: "│"
	}
	return 3 // Normal mode: "│   "
}

// NewConsolePrinter creates a new console printer
func NewConsolePrinter(logger logger.Logger) *ConsolePrinter {
	return &ConsolePrinter{
		colorScheme: NewColorScheme(),
		logger:      logger,
	}
}

// PrintRequest prints request information to console with enhanced box-drawing format
func (p *ConsolePrinter) PrintRequest(data *request.RequestData) error {
	// Increment and get request number
	requestNum := atomic.AddUint64(&requestCounter, 1)
	timestamp := data.Timestamp.Format("2006-01-02T15:04:05-07:00")

	// Get dynamic terminal width
	boxWidth := p.getTerminalWidth()

	// Print top border with request info
	p.printTopBorder(requestNum, timestamp, boxWidth)

	// Print request line
	p.printRequestLine(data, boxWidth)

	// Print headers section if available
	if len(data.Headers) > 0 {
		p.printSectionSeparator(boxWidth)
		p.printHeadersSection(data.Headers, boxWidth)
	}

	// Print body section
	p.printSectionSeparator(boxWidth)
	p.printBodySection(data, boxWidth)

	// Print bottom border
	p.printBottomBorder(requestNum, boxWidth)

	// Add empty line for separation
	fmt.Println()

	return nil
}

// printTopBorder prints the top border with request number and timestamp
func (p *ConsolePrinter) printTopBorder(requestNum uint64, timestamp string, width int) {
	var border string
	if p.isCompactMode(width) {
		// Compact mode for narrow screens
		title := fmt.Sprintf(" #%d ", requestNum)
		titleWithTime := fmt.Sprintf("%s%s", title, timestamp)

		// If still too wide, truncate timestamp
		if utf8.RuneCountInString(titleWithTime) > width-4 {
			// Keep only the time part of timestamp
			timeOnly := timestamp[11:19] // HH:MM:SS
			titleWithTime = fmt.Sprintf("%s%s", title, timeOnly)

			// If still too wide, use minimal format
			if utf8.RuneCountInString(titleWithTime) > width-4 {
				titleWithTime = fmt.Sprintf("#%d", requestNum)
			}
		}

		totalTitleLen := utf8.RuneCountInString(titleWithTime)
		padding := width - totalTitleLen - 4
		if padding < 0 {
			padding = 0
		}

		border = fmt.Sprintf("┌%s%s%s┐",
			strings.Repeat("─", 1),
			titleWithTime,
			strings.Repeat("─", padding+1))
	} else {
		// Normal mode
		title := fmt.Sprintf(" REQUEST #%d ", requestNum)
		titleWithTime := fmt.Sprintf("%s──(%s)", title, timestamp)

		totalTitleLen := utf8.RuneCountInString(titleWithTime)
		padding := width - totalTitleLen - 4
		if padding < 0 {
			padding = 0
		}

		leftPadding := padding / 2
		rightPadding := padding - leftPadding

		border = fmt.Sprintf("┌%s%s%s┐",
			strings.Repeat("─", leftPadding+2),
			titleWithTime,
			strings.Repeat("─", rightPadding+2))
	}

	p.colorScheme.Separator.Println(border)
}

// printRequestLine prints the HTTP method, path and remote address
func (p *ConsolePrinter) printRequestLine(data *request.RequestData, width int) {
	methodColor := p.getMethodColor(data.Method)
	indent := p.getIndent(width)

	// Print empty line for spacing
	fmt.Println("│")

	// Build request information
	var requestInfo strings.Builder
	fmt.Fprintf(&requestInfo, "[%s] %s", data.Method, data.Path)

	if data.Query != "" {
		requestInfo.WriteString("?")
		requestInfo.WriteString(data.Query)
	}

	// Print request line with proper indentation
	fmt.Print("│", strings.Repeat(" ", indent))
	methodColor.Print(requestInfo.String())
	fmt.Println()

	// Add remote address on next line
	remoteAddrInfo := fmt.Sprintf("[FROM: %s]", data.RemoteAddr)
	fmt.Print("│", strings.Repeat(" ", indent))
	p.colorScheme.RemoteAddr.Println(remoteAddrInfo)

	// Another empty line for spacing
	fmt.Println("│")
}

// printSectionSeparator prints the separator between sections
func (p *ConsolePrinter) printSectionSeparator(width int) {
	p.colorScheme.Separator.Printf("├%s┤\n", strings.Repeat("─", width-2))
}

// printHeadersSection prints the headers section with proper formatting
func (p *ConsolePrinter) printHeadersSection(headers http.Header, width int) {
	indent := p.getIndent(width)
	availableWidth := width - indent - 1 // -1 for border character

	fmt.Println("│")
	fmt.Print("│", strings.Repeat(" ", indent))
	p.colorScheme.Separator.Println("─── Headers ───")
	fmt.Println("│")

	for key, values := range headers {
		lowerKey := strings.ToLower(key)

		// Skip headers that should be hidden
		if p.shouldSkipHeader(lowerKey) {
			continue
		}

		// Build header line
		var headerLine strings.Builder
		headerLine.WriteString(key)
		headerLine.WriteString(": ")

		// Print header value or redacted
		if p.isSensitiveHeader(lowerKey) {
			headerLine.WriteString("[REDACTED]")
		} else {
			headerLine.WriteString(strings.Join(values, ", "))
		}

		// Check if header line needs wrapping
		headerText := headerLine.String()
		if utf8.RuneCountInString(headerText) <= availableWidth {
			// Header fits in one line
			fmt.Print("│", strings.Repeat(" ", indent))
			if p.isSensitiveHeader(lowerKey) {
				p.colorScheme.HeaderKey.Print(key + ": ")
				p.colorScheme.HeaderValue.Println("[REDACTED]")
			} else {
				p.colorScheme.HeaderKey.Print(key + ": ")
				p.colorScheme.HeaderValue.Println(strings.Join(values, ", "))
			}
		} else {
			// Header needs to be wrapped
			wrappedLines := p.wrapTextWithIndent(headerText, width, indent)
			for i, line := range wrappedLines {
				fmt.Print("│")
				if i == 0 {
					// First line: colorize key part
					if colonPos := strings.Index(line, ": "); colonPos != -1 {
						keyPart := line[:colonPos]
						valuePart := line[colonPos+2:]
						p.colorScheme.HeaderKey.Print(keyPart + ": ")
						p.colorScheme.HeaderValue.Println(valuePart)
					} else {
						p.colorScheme.HeaderKey.Println(line)
					}
				} else {
					p.colorScheme.HeaderValue.Println(line)
				}
			}
		}
	}

	fmt.Println("│")
}

// printBodySection prints the body section with size information
func (p *ConsolePrinter) printBodySection(data *request.RequestData, width int) {
	indent := p.getIndent(width)
	bodySize := humanize.Bytes(uint64(len(data.Body)))

	fmt.Println("│")
	fmt.Print("│", strings.Repeat(" ", indent))

	// Adjust body section title for compact mode
	if p.isCompactMode(width) {
		p.colorScheme.Separator.Printf("Body (%s)\n", bodySize)
	} else {
		p.colorScheme.Separator.Printf("─── Body (%s) ───\n", bodySize)
	}
	fmt.Println("│")

	if len(data.Body) == 0 {
		fmt.Print("│", strings.Repeat(" ", indent))
		p.colorScheme.BodyContent.Println("[Empty Body]")
		fmt.Println("│")
		return
	}

	// Check if it's binary content
	if data.IsBinary {
		fmt.Print("│", strings.Repeat(" ", indent))
		p.colorScheme.BinaryNotice.Printf(
			"[Binary Body: %s, %s. Content skipped.]\n",
			data.ContentType,
			bodySize,
		)
		fmt.Println("│")
		return
	}

	// Print body content with proper formatting
	p.printFormattedBody(data.Body, width)
}

// printFormattedBody prints body content with proper indentation and formatting
func (p *ConsolePrinter) printFormattedBody(body []byte, width int) {
	// Calculate available content width (subtract border character and indentation)
	indent := p.getIndent(width)
	availableWidth := width - indent - 1 // -1 for the border character

	if availableWidth <= 0 {
		availableWidth = 10 // Minimum usable width
	}

	content := string(body)

	// Process content line by line to preserve existing line breaks
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		trimmedLine := strings.TrimRight(line, " \t") // Remove trailing spaces but keep leading ones

		if trimmedLine == "" {
			// Empty line - just print the border
			fmt.Println("│")
			continue
		}

		// Check if line needs wrapping
		if utf8.RuneCountInString(trimmedLine) <= availableWidth {
			// Line fits in available width
			fmt.Print("│", strings.Repeat(" ", indent))
			p.colorScheme.BodyContent.Println(trimmedLine)
		} else {
			// Line needs to be wrapped
			wrappedLines := p.wrapTextWithIndent(trimmedLine, width, indent+1)
			for _, wrappedLine := range wrappedLines {
				fmt.Print("│")
				if wrappedLine != "" {
					p.colorScheme.BodyContent.Println(wrappedLine)
				} else {
					fmt.Println()
				}
			}
		}
	}

	fmt.Println("│")
}

// printBottomBorder prints the bottom border
func (p *ConsolePrinter) printBottomBorder(requestNum uint64, width int) {
	var border string
	if p.isCompactMode(width) {
		// Compact mode for narrow screens
		title := fmt.Sprintf(" #%d ", requestNum)

		titleLen := utf8.RuneCountInString(title)
		padding := width - titleLen - 4
		if padding < 0 {
			padding = 0
		}

		border = fmt.Sprintf("└%s%s%s┘",
			strings.Repeat("─", 1),
			title,
			strings.Repeat("─", padding+1))
	} else {
		// Normal mode
		title := fmt.Sprintf(" END OF REQUEST #%d ", requestNum)
		titleLen := utf8.RuneCountInString(title)
		padding := width - titleLen - 4
		if padding < 0 {
			padding = 0
		}

		leftPadding := padding / 2
		rightPadding := padding - leftPadding

		border = fmt.Sprintf("└%s%s%s┘",
			strings.Repeat("─", leftPadding+2),
			title,
			strings.Repeat("─", rightPadding+2))
	}

	p.colorScheme.Separator.Println(border)
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


