package printer

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"mime"
	"net/url"
	"sort"
	"strings"
	"unicode/utf8"

	nethtml "golang.org/x/net/html"
	"html"

	"github.com/dustin/go-humanize"
	"github.com/funnyzak/reqtap/internal/config"
	"github.com/funnyzak/reqtap/internal/logger"
	"github.com/funnyzak/reqtap/pkg/i18n"
	"github.com/funnyzak/reqtap/pkg/request"
)

type bodyFormatter struct {
	cfg    *config.BodyViewConfig
	logger logger.Logger
	intl   *i18n.Translator
	locale string
}

type formattedBody struct {
	Text    string
	Notices []string
}

func newBodyFormatter(cfg *config.BodyViewConfig, log logger.Logger, translator *i18n.Translator, locale string) *bodyFormatter {
	if cfg == nil {
		cfg = &config.BodyViewConfig{}
	}
	resolved := strings.TrimSpace(locale)
	if resolved == "" && translator != nil {
		resolved = translator.DefaultLocale()
	}
	return &bodyFormatter{cfg: cfg, logger: log, intl: translator, locale: resolved}
}

func (f *bodyFormatter) t(key string) string {
	if f == nil || f.intl == nil {
		return key
	}
	return f.intl.Text(f.locale, key)
}

func (f *bodyFormatter) Format(data *request.RequestData) formattedBody {
	if f == nil || data == nil {
		return formattedBody{}
	}
	body := data.Body
	if len(body) == 0 {
		return formattedBody{}
	}
	if !f.cfg.Enable {
		return formattedBody{Text: string(body)}
	}
	mediaType := normalizeMediaType(data.ContentType)
	if res, ok := f.formatJSON(mediaType, body); ok {
		return res
	}
	if res, ok := f.formatForm(mediaType, body); ok {
		return res
	}
	if res, ok := f.formatXML(mediaType, body); ok {
		return res
	}
	if res, ok := f.formatHTML(mediaType, body); ok {
		return res
	}
	return formattedBody{Text: string(body)}
}

func (f *bodyFormatter) formatJSON(mediaType string, body []byte) (formattedBody, bool) {
	if !f.cfg.Json.Enable {
		return formattedBody{}, false
	}
	if !looksLikeJSON(mediaType, body) {
		return formattedBody{}, false
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || !json.Valid(trimmed) {
		return formattedBody{}, false
	}
	if !f.cfg.Json.Pretty {
		return formattedBody{Text: string(body)}, true
	}
	if f.cfg.Json.MaxIndentBytes > 0 && len(trimmed) > f.cfg.Json.MaxIndentBytes {
		notice := fmt.Sprintf(f.t(keyJSONIndentSkipped), humanize.Bytes(uint64(f.cfg.Json.MaxIndentBytes)))
		return formattedBody{Text: string(body), Notices: []string{notice}}, true
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, trimmed, "", "  "); err != nil {
		if f.logger != nil {
			f.logger.Debug("json indent failed", "error", err)
		}
		return formattedBody{}, false
	}
	return formattedBody{Text: buf.String()}, true
}

func (f *bodyFormatter) formatForm(mediaType string, body []byte) (formattedBody, bool) {
	if !f.cfg.Form.Enable {
		return formattedBody{}, false
	}
	if !strings.Contains(mediaType, "application/x-www-form-urlencoded") {
		return formattedBody{}, false
	}
	values, err := url.ParseQuery(string(body))
	if err != nil {
		if f.logger != nil {
			f.logger.Debug("form parse failed", "error", err)
		}
		return formattedBody{}, false
	}
	if len(values) == 0 {
		return formattedBody{Text: string(body)}, true
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	keyHeader := f.t(keyFormKeyHeader)
	valueHeader := f.t(keyFormValueHeader)
	maxKeyWidth := utf8.RuneCountInString(keyHeader)
	for _, key := range keys {
		if w := utf8.RuneCountInString(key); w > maxKeyWidth {
			maxKeyWidth = w
		}
	}
	var builder strings.Builder
	title := f.t(keyFormTitle)
	if title == "" {
		title = "Form data:"
	}
	builder.WriteString(title + "\n")
	fmt.Fprintf(&builder, "%-*s │ %s\n", maxKeyWidth, keyHeader, valueHeader)
	divider := strings.Repeat("─", maxKeyWidth)
	builder.WriteString(divider + "─┼" + strings.Repeat("─", 40) + "\n")
	for _, key := range keys {
		fmt.Fprintf(&builder, "%-*s │ %s\n", maxKeyWidth, key, strings.Join(values[key], ", "))
	}
	return formattedBody{Text: builder.String()}, true
}

func (f *bodyFormatter) formatXML(mediaType string, body []byte) (formattedBody, bool) {
	if !f.cfg.XML.Enable {
		return formattedBody{}, false
	}
	if !strings.Contains(mediaType, "xml") {
		return formattedBody{}, false
	}
	processed := body
	if f.cfg.XML.StripControl {
		processed = stripControlBytes(processed)
	}
	if !f.cfg.XML.Pretty {
		return formattedBody{Text: string(processed)}, true
	}
	formatted, err := prettyXML(processed)
	if err != nil {
		if f.logger != nil {
			f.logger.Debug("xml pretty failed", "error", err)
		}
		return formattedBody{Text: string(processed)}, true
	}
	return formattedBody{Text: formatted}, true
}

func (f *bodyFormatter) formatHTML(mediaType string, body []byte) (formattedBody, bool) {
	if !f.cfg.HTML.Enable {
		return formattedBody{}, false
	}
	if !strings.Contains(mediaType, "html") && !looksLikeHTML(body) {
		return formattedBody{}, false
	}
	processed := body
	if f.cfg.HTML.StripControl {
		processed = stripControlBytes(processed)
	}
	if !f.cfg.HTML.Pretty {
		return formattedBody{Text: string(processed)}, true
	}
	formatted, err := prettyHTML(processed)
	if err != nil {
		if f.logger != nil {
			f.logger.Debug("html pretty failed", "error", err)
		}
		return formattedBody{Text: string(processed)}, true
	}
	return formattedBody{Text: formatted}, true
}

func normalizeMediaType(contentType string) string {
	if contentType == "" {
		return ""
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(contentType))
	}
	return strings.ToLower(mediaType)
}

func looksLikeJSON(mediaType string, body []byte) bool {
	if strings.Contains(mediaType, "json") {
		return true
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return false
	}
	first := trimmed[0]
	last := trimmed[len(trimmed)-1]
	return (first == '{' && last == '}') || (first == '[' && last == ']')
}

func looksLikeHTML(body []byte) bool {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) < 5 {
		return false
	}
	upper := strings.ToLower(string(trimmed[:5]))
	return strings.HasPrefix(upper, "<html") || strings.HasPrefix(upper, "<!doc")
}

func stripControlBytes(b []byte) []byte {
	if len(b) == 0 {
		return b
	}
	buf := make([]byte, 0, len(b))
	for _, ch := range b {
		if ch < 0x20 && ch != '\n' && ch != '\r' && ch != '\t' {
			continue
		}
		buf = append(buf, ch)
	}
	return buf
}

func prettyXML(data []byte) (string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var buf bytes.Buffer
	encoder := xml.NewEncoder(&buf)
	encoder.Indent("", "  ")
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		if err := encoder.EncodeToken(token); err != nil {
			return "", err
		}
	}
	if err := encoder.Flush(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func prettyHTML(data []byte) (string, error) {
	node, err := nethtml.Parse(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	var builder strings.Builder
	renderHTMLNode(&builder, node, 0)
	return builder.String(), nil
}

func renderHTMLNode(builder *strings.Builder, node *nethtml.Node, depth int) {
	switch node.Type {
	case nethtml.DocumentNode:
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			renderHTMLNode(builder, child, depth)
		}
	case nethtml.ElementNode:
		indent := strings.Repeat("  ", depth)
		builder.WriteString(indent)
		builder.WriteString("<" + node.Data)
		for _, attr := range node.Attr {
			builder.WriteString(fmt.Sprintf(" %s=\"%s\"", attr.Key, html.EscapeString(attr.Val)))
		}
		if isVoidElement(node.Data) {
			builder.WriteString(" />\n")
			return
		}
		builder.WriteString(">\n")
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			renderHTMLNode(builder, child, depth+1)
		}
		if node.FirstChild != nil {
			builder.WriteString(indent)
		}
		builder.WriteString("</" + node.Data + ">\n")
	case nethtml.TextNode:
		text := strings.TrimSpace(node.Data)
		if text == "" {
			return
		}
		indent := strings.Repeat("  ", depth)
		builder.WriteString(indent)
		builder.WriteString(text)
		builder.WriteString("\n")
	case nethtml.CommentNode:
		indent := strings.Repeat("  ", depth)
		builder.WriteString(indent)
		builder.WriteString("<!--" + strings.TrimSpace(node.Data) + "-->\n")
	}
}

func isVoidElement(tag string) bool {
	switch strings.ToLower(tag) {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "keygen", "link", "meta", "param", "source", "track", "wbr":
		return true
	default:
		return false
	}
}
