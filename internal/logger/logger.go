package logger

import (
	"io"
	"os"
	"strings"

	"github.com/funnyzak/reqtap/internal/config"
	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger logging interface
type Logger interface {
	// Debug logs a Debug event.
	Debug(msg string, fields ...interface{})
	// Info logs an Info event.
	Info(msg string, fields ...interface{})
	// Warn logs a Warn event.
	Warn(msg string, fields ...interface{})
	// Error logs an Error event.
	Error(msg string, fields ...interface{})
	// Fatal logs a Fatal event and terminates the program.
	Fatal(msg string, fields ...interface{})
}

// zerologAdapter zerolog adapter
type zerologAdapter struct {
	logger *zerolog.Logger
}

// addFields adds fields to zerolog event
func (z *zerologAdapter) addFields(event *zerolog.Event, fields ...interface{}) *zerolog.Event {
	if len(fields) == 0 {
		return event
	}

	// Fields should be key-value pairs
	for i := 0; i < len(fields)-1; i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			continue
		}
		value := fields[i+1]

		// Handle different types
		switch v := value.(type) {
		case string:
			event = event.Str(key, v)
		case int:
			event = event.Int(key, v)
		case int64:
			event = event.Int64(key, v)
		case int32:
			event = event.Int32(key, v)
		case uint:
			event = event.Uint(key, v)
		case uint64:
			event = event.Uint64(key, v)
		case uint32:
			event = event.Uint32(key, v)
		case float64:
			event = event.Float64(key, v)
		case float32:
			event = event.Float32(key, v)
		case bool:
			event = event.Bool(key, v)
		case error:
			event = event.AnErr(key, v)
		case []string:
			event = event.Strs(key, v)
		case []interface{}:
			event = event.Interface(key, v)
		default:
			event = event.Interface(key, v)
		}
	}

	return event
}

// Debug implements Logger
func (z *zerologAdapter) Debug(msg string, fields ...interface{}) {
	z.addFields(z.logger.Debug(), fields...).Msg(msg)
}

// Info implements Logger
func (z *zerologAdapter) Info(msg string, fields ...interface{}) {
	z.addFields(z.logger.Info(), fields...).Msg(msg)
}

// Warn implements Logger
func (z *zerologAdapter) Warn(msg string, fields ...interface{}) {
	z.addFields(z.logger.Warn(), fields...).Msg(msg)
}

// Error implements Logger
func (z *zerologAdapter) Error(msg string, fields ...interface{}) {
	z.addFields(z.logger.Error(), fields...).Msg(msg)
}

// Fatal implements Logger
func (z *zerologAdapter) Fatal(msg string, fields ...interface{}) {
	z.addFields(z.logger.Fatal(), fields...).Msg(msg)
}

// NewLogger creates new logger instance
func NewLogger(cfg *config.LogConfig, outputMode string) Logger {
	// Set log level
	logLevel, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}

	var writers []io.Writer
	structured := strings.ToLower(outputMode) == "json"

	if structured {
		writers = append(writers, os.Stdout)
	} else {
		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "2006-01-02 15:04:05",
		}
		writers = append(writers, consoleWriter)
	}

	// If file logging is enabled, add file output
	if cfg.FileLogging.Enable {
		fileWriter := &lumberjack.Logger{
			Filename:   cfg.FileLogging.Path,
			MaxSize:    cfg.FileLogging.MaxSizeMB,
			MaxBackups: cfg.FileLogging.MaxBackups,
			MaxAge:     cfg.FileLogging.MaxAgeDays,
			Compress:   cfg.FileLogging.Compress,
		}
		// File logging uses JSON format (original writer)
		writers = append(writers, fileWriter)
	}

	// Create multi-output writer
	multiWriter := io.MultiWriter(writers...)

	// Create logger
	logger := zerolog.New(multiWriter).Level(logLevel).With().Timestamp().Logger()

	return &zerologAdapter{logger: &logger}
}
