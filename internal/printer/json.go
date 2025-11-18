package printer

import (
	"encoding/json"
	"io"
	"os"

	"github.com/funnyzak/reqtap/internal/logger"
	"github.com/funnyzak/reqtap/pkg/request"
)

// JSONPrinter 以 JSON 行输出请求
type JSONPrinter struct {
	encoder *json.Encoder
	logger  logger.Logger
	out     io.Writer
}

// NewJSONPrinter 创建 JSON 输出器
func NewJSONPrinter(log logger.Logger) *JSONPrinter {
	out := os.Stdout
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	return &JSONPrinter{encoder: encoder, logger: log, out: out}
}

// SetOutput 替换输出目标，便于测试
func (p *JSONPrinter) SetOutput(w io.Writer) {
	if w == nil {
		w = os.Stdout
	}
	p.out = w
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	p.encoder = encoder
}

type jsonRequestEnvelope struct {
	Type     string               `json:"type"`
	ID       uint64               `json:"id"`
	Request  *request.RequestData `json:"request"`
	BodyText string               `json:"body_text,omitempty"`
}

// PrintRequest 输出请求 JSON
func (p *JSONPrinter) PrintRequest(data *request.RequestData) error {
	env := jsonRequestEnvelope{
		Type:    "request",
		ID:      nextRequestNumber(),
		Request: data,
	}
	if !data.IsBinary && len(data.Body) > 0 {
		env.BodyText = string(data.Body)
	}
	if err := p.encoder.Encode(env); err != nil {
		if p.logger != nil {
			p.logger.Error("Failed to encode request JSON", "error", err)
		}
		return err
	}
	return nil
}
