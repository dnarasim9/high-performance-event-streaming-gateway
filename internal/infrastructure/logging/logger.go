package logging

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/config"
)

// Logger defines the interface for structured logging.
type Logger interface {
	Info(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Debug(msg string, fields ...Field)
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
	WithCorrelationID(id string) Logger
	Sync() error
}

// Field represents a structured log field.
type Field struct {
	Key   string
	Value interface{}
}

// ZapLogger implements Logger using go.uber.org/zap.
type ZapLogger struct {
	logger        *zap.Logger
	correlationID string
}

// NewLogger creates a new Logger based on configuration.
func NewLogger(cfg *config.LogConfig) (Logger, error) {
	var zapCfg zap.Config

	switch cfg.Format {
	case "json":
		zapCfg = zap.NewProductionConfig()
	case "text":
		zapCfg = zap.NewDevelopmentConfig()
	default:
		zapCfg = zap.NewProductionConfig()
	}

	// Set log level
	level := getZapLevel(cfg.Level)
	zapCfg.Level = zap.NewAtomicLevelAt(level)

	// Configure output
	switch cfg.Output {
	case "stdout":
		zapCfg.OutputPaths = []string{"stdout"}
		zapCfg.ErrorOutputPaths = []string{"stderr"}
	case "stderr":
		zapCfg.OutputPaths = []string{"stderr"}
		zapCfg.ErrorOutputPaths = []string{"stderr"}
	default:
		// Assume it's a file path
		zapCfg.OutputPaths = []string{cfg.Output}
		zapCfg.ErrorOutputPaths = []string{cfg.Output}
	}

	zapLogger, err := zapCfg.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	return &ZapLogger{
		logger: zapLogger,
	}, nil
}

// Info logs an info level message.
func (zl *ZapLogger) Info(msg string, fields ...Field) {
	zapFields := convertFields(fields)
	if zl.correlationID != "" {
		zapFields = append(zapFields, zap.String("correlation_id", zl.correlationID))
	}
	zl.logger.Info(msg, zapFields...)
}

// Error logs an error level message.
func (zl *ZapLogger) Error(msg string, fields ...Field) {
	zapFields := convertFields(fields)
	if zl.correlationID != "" {
		zapFields = append(zapFields, zap.String("correlation_id", zl.correlationID))
	}
	zl.logger.Error(msg, zapFields...)
}

// Warn logs a warning level message.
func (zl *ZapLogger) Warn(msg string, fields ...Field) {
	zapFields := convertFields(fields)
	if zl.correlationID != "" {
		zapFields = append(zapFields, zap.String("correlation_id", zl.correlationID))
	}
	zl.logger.Warn(msg, zapFields...)
}

// Debug logs a debug level message.
func (zl *ZapLogger) Debug(msg string, fields ...Field) {
	zapFields := convertFields(fields)
	if zl.correlationID != "" {
		zapFields = append(zapFields, zap.String("correlation_id", zl.correlationID))
	}
	zl.logger.Debug(msg, zapFields...)
}

// WithField returns a new Logger with an additional field.
func (zl *ZapLogger) WithField(key string, value interface{}) Logger {
	return &ZapLogger{
		logger:        zl.logger.With(zap.Any(key, value)),
		correlationID: zl.correlationID,
	}
}

// WithFields returns a new Logger with additional fields.
func (zl *ZapLogger) WithFields(fields map[string]interface{}) Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return &ZapLogger{
		logger:        zl.logger.With(zapFields...),
		correlationID: zl.correlationID,
	}
}

// WithCorrelationID returns a new Logger with a correlation ID.
func (zl *ZapLogger) WithCorrelationID(id string) Logger {
	return &ZapLogger{
		logger:        zl.logger,
		correlationID: id,
	}
}

// Sync flushes any buffered log entries.
func (zl *ZapLogger) Sync() error {
	return zl.logger.Sync()
}

// ContextKey type for storing values in context
type contextKey string

const loggerContextKey contextKey = "logger"
const correlationIDContextKey contextKey = "correlation_id"

// WithContext returns a new context with the logger attached.
func WithContext(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey, logger)
}

// FromContext retrieves the logger from the context.
func FromContext(ctx context.Context) Logger {
	if logger, ok := ctx.Value(loggerContextKey).(Logger); ok {
		return logger
	}
	// Return a no-op logger if none is found
	return &ZapLogger{logger: zap.NewNop()}
}

// WithCorrelationIDContext returns a new context with a correlation ID.
func WithCorrelationIDContext(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDContextKey, id)
}

// GetCorrelationIDFromContext retrieves the correlation ID from the context.
func GetCorrelationIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDContextKey).(string); ok {
		return id
	}
	return ""
}

// RequestLogger returns a Logger configured with request-related information.
func RequestLogger(baseLogger Logger, method, path, correlationID string) Logger {
	return baseLogger.
		WithField("method", method).
		WithField("path", path).
		WithCorrelationID(correlationID)
}

// Helper functions

func getZapLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func convertFields(fields []Field) []zap.Field {
	zapFields := make([]zap.Field, len(fields))
	for i, f := range fields {
		zapFields[i] = zap.Any(f.Key, f.Value)
	}
	return zapFields
}
