package printer

import (
	"sync/atomic"

	"github.com/funnyzak/reqtap/internal/config"
	"github.com/funnyzak/reqtap/internal/logger"
	"github.com/funnyzak/reqtap/pkg/i18n"
	"github.com/funnyzak/reqtap/pkg/request"
)

// Printer 抽象输出接口
type Printer interface {
	PrintRequest(*request.RequestData) error
}

var globalRequestCounter uint64

func nextRequestNumber() uint64 {
	return atomic.AddUint64(&globalRequestCounter, 1)
}

// New 创建指定模式的 Printer
func New(mode string, log logger.Logger, cfg *config.OutputConfig, translator *i18n.Translator, locale string) Printer {
	if cfg == nil {
		cfg = &config.OutputConfig{}
	}
	switch mode {
	case "json":
		return NewJSONPrinter(log)
	default:
		return NewConsolePrinter(log, &cfg.BodyView, translator, locale)
	}
}
