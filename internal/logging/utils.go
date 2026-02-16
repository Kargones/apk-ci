// Package logging provides utilities for structured logging integration.
// This package contains helper functions and utilities that make it easier
// to integrate structured logging throughout the application.
package logging

import (
	"context"
	"log/slog"
	"os"
)

// Context key types for type safety
type contextKey string

const (
	loggerKey contextKey = "logger"
)

// DefaultLogger creates a default structured logger with JSON output.
// This function provides a standard logger configuration for the application.
//
// Returns:
//   - *slog.Logger: configured logger instance
func DefaultLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// DebugLogger creates a debug-enabled structured logger with JSON output.
// This function provides a logger configuration with debug level enabled.
//
// Returns:
//   - *slog.Logger: configured debug logger instance
func DebugLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

// TextLogger creates a structured logger with text output for development.
// This function provides a human-readable logger configuration.
//
// Returns:
//   - *slog.Logger: configured text logger instance
func TextLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// NewStructuredLogger creates a new StructuredLogger implementation.
// This function provides a convenient way to create structured loggers.
//
// Parameters:
//   - logger: the underlying slog.Logger to wrap
//
// Returns:
//   - StructuredLogger: new structured logger instance
func NewStructuredLogger(logger *slog.Logger) StructuredLogger {
	return NewSlogAdapter(logger)
}

// WithFields adds structured fields to a context for logging.
// This function provides a convenient way to add multiple fields to context.
//
// Parameters:
//   - ctx: the context to add fields to
//   - fields: map of field names to values
//
// Returns:
//   - context.Context: context with added fields
func WithFields(ctx context.Context, fields map[string]any) context.Context {
	for key, value := range fields {
		ctx = context.WithValue(ctx, contextKey(key), value)
	}
	return ctx
}

// GetField extracts a field value from context.
// This function provides a convenient way to extract field values.
//
// Parameters:
//   - ctx: the context to extract from
//   - key: the field key to extract
//
// Returns:
//   - any: the field value, or nil if not found
func GetField(ctx context.Context, key string) any {
	return ctx.Value(contextKey(key))
}

// LoggerFromContext extracts a logger from context.
// This function provides a way to get logger from context with fallback.
//
// Parameters:
//   - ctx: the context to extract logger from
//
// Returns:
//   - StructuredLogger: logger instance or default logger
func LoggerFromContext(ctx context.Context) StructuredLogger {
	if logger, ok := ctx.Value(loggerKey).(StructuredLogger); ok {
		return logger
	}
	return NewStructuredLogger(DefaultLogger())
}

// WithLogger adds a logger to context.
// This function provides a way to add logger to context.
//
// Parameters:
//   - ctx: the context to add logger to
//   - logger: the logger to add
//
// Returns:
//   - context.Context: context with added logger
func WithLogger(ctx context.Context, logger StructuredLogger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// CreateOperationContext creates a context for an operation with correlation ID.
// This function provides a convenient way to create operation contexts.
//
// Parameters:
//   - parent: parent context
//   - logger: structured logger
//   - operation: operation name
//
// Returns:
//   - context.Context: context with correlation ID and operation info
//   - string: generated correlation ID
func CreateOperationContext(parent context.Context, logger StructuredLogger, operation string) (context.Context, string) {
	correlationID := logger.GenerateCorrelationID()
	ctx := logger.WithCorrelationID(parent, correlationID)
	ctx = WithFields(ctx, map[string]any{
		"operation": operation,
		"correlation_id": correlationID,
	})
	ctx = WithLogger(ctx, logger)
	return ctx, correlationID
}

// LogError is a convenience function for logging errors with context.
// This function provides a simple way to log errors with correlation tracking.
//
// Parameters:
//   - ctx: context for the operation
//   - logger: structured logger
//   - msg: error message
//   - err: error to log
//   - attrs: additional attributes
func LogError(ctx context.Context, logger StructuredLogger, msg string, err error, attrs ...slog.Attr) {
	logger.LogError(ctx, msg, err, attrs...)
}

// LogInfo is a convenience function for logging info messages with context.
// This function provides a simple way to log info messages with correlation tracking.
//
// Parameters:
//   - ctx: context for the operation
//   - logger: structured logger
//   - msg: info message
//   - attrs: additional attributes
func LogInfo(ctx context.Context, logger StructuredLogger, msg string, attrs ...slog.Attr) {
	logger.LogInfo(ctx, msg, attrs...)
}

// LogDebug is a convenience function for logging debug messages with context.
// This function provides a simple way to log debug messages with correlation tracking.
//
// Parameters:
//   - ctx: context for the operation
//   - logger: structured logger
//   - msg: debug message
//   - attrs: additional attributes
func LogDebug(ctx context.Context, logger StructuredLogger, msg string, attrs ...slog.Attr) {
	logger.LogDebug(ctx, msg, attrs...)
}

// LogWarn is a convenience function for logging warning messages with context.
// This function provides a simple way to log warning messages with correlation tracking.
//
// Parameters:
//   - ctx: context for the operation
//   - logger: structured logger
//   - msg: warning message
//   - attrs: additional attributes
func LogWarn(ctx context.Context, logger StructuredLogger, msg string, attrs ...slog.Attr) {
	logger.LogWarn(ctx, msg, attrs...)
}