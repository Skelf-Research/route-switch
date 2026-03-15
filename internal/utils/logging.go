package utils

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
)

var (
	defaultLogger *slog.Logger
	loggerMu      sync.RWMutex
)

func init() {
	// Initialize with a default JSON handler for structured logging
	defaultLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// LogLevel represents logging levels
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

// LoggerConfig configures the logger
type LoggerConfig struct {
	Level      LogLevel
	Output     io.Writer
	JSONFormat bool
	AddSource  bool
}

// InitLogger initializes the global logger with the given configuration
func InitLogger(cfg LoggerConfig) {
	loggerMu.Lock()
	defer loggerMu.Unlock()

	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	level := slog.LevelInfo
	switch cfg.Level {
	case LevelDebug:
		level = slog.LevelDebug
	case LevelInfo:
		level = slog.LevelInfo
	case LevelWarn:
		level = slog.LevelWarn
	case LevelError:
		level = slog.LevelError
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
	}

	var handler slog.Handler
	if cfg.JSONFormat {
		handler = slog.NewJSONHandler(output, opts)
	} else {
		handler = slog.NewTextHandler(output, opts)
	}

	defaultLogger = slog.New(handler)
}

// Logger returns the default logger instance
func Logger() *slog.Logger {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	return defaultLogger
}

// WithContext returns a logger with context values
func WithContext(ctx context.Context) *slog.Logger {
	return Logger()
}

// LogEvent emits a structured log line for observability (backwards compatible)
func LogEvent(event string, fields map[string]interface{}) {
	attrs := make([]any, 0, len(fields)*2+2)
	attrs = append(attrs, "event", event)
	for k, v := range fields {
		attrs = append(attrs, k, v)
	}
	Logger().Info("event", attrs...)
}

// Debug logs a debug message with optional key-value pairs
func Debug(msg string, args ...any) {
	Logger().Debug(msg, args...)
}

// Info logs an info message with optional key-value pairs
func Info(msg string, args ...any) {
	Logger().Info(msg, args...)
}

// Warn logs a warning message with optional key-value pairs
func Warn(msg string, args ...any) {
	Logger().Warn(msg, args...)
}

// Error logs an error message with optional key-value pairs
func Error(msg string, args ...any) {
	Logger().Error(msg, args...)
}

// LogRequest logs an HTTP request with relevant details
func LogRequest(method, path string, statusCode int, durationMS float64, fields ...any) {
	args := []any{
		"method", method,
		"path", path,
		"status", statusCode,
		"duration_ms", durationMS,
	}
	args = append(args, fields...)
	Logger().Info("http_request", args...)
}

// LogModelCall logs a model provider call with relevant details
func LogModelCall(provider, model string, success bool, durationMS float64, cost float64, fields ...any) {
	args := []any{
		"provider", provider,
		"model", model,
		"success", success,
		"duration_ms", durationMS,
		"cost", cost,
	}
	args = append(args, fields...)
	Logger().Info("model_call", args...)
}

// LogOptimization logs an optimization run with relevant details
func LogOptimization(templateID, model string, score float64, durationMS float64, fields ...any) {
	args := []any{
		"template_id", templateID,
		"model", model,
		"score", score,
		"duration_ms", durationMS,
	}
	args = append(args, fields...)
	Logger().Info("optimization", args...)
}

// LogError logs an error with context
func LogError(err error, msg string, fields ...any) {
	args := []any{"error", err.Error()}
	args = append(args, fields...)
	Logger().Error(msg, args...)
}
