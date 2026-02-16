// Package sonarqube provides implementation of logging functionality.
// This package contains the implementation of structured logging
// with correlation IDs and debug mode support.
package sonarqube

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
	
	"github.com/Kargones/apk-ci/internal/logging"
)

// LoggingService provides functionality for structured logging.
// This service layer implements comprehensive logging throughout all components,
// correlation IDs for request tracing, and debug mode with verbose logging.
// LoggingService implements the logging.StructuredLogger interface.
type LoggingService struct {
	// logger is the structured logger
	logger *slog.Logger
	
	// debugMode enables verbose logging
	debugMode atomic.Bool
	
	// correlationIDCounter is used to generate unique correlation IDs
	correlationIDCounter uint64
	
	// mutex for thread-safe operations
	mutex sync.Mutex
}

// Ensure LoggingService implements StructuredLogger interface
var _ logging.StructuredLogger = (*LoggingService)(nil)

// CorrelationIDKey is the context key for correlation IDs
type CorrelationIDKey string

const (
	// CorrelationID is the key for correlation ID in context
	CorrelationID CorrelationIDKey = "correlation_id"
)

// NewLoggingService creates a new instance of LoggingService.
// This function initializes the service with the provided logger.
//
// Parameters:
//   - logger: structured logger instance
//
// Returns:
//   - *LoggingService: initialized logging service
func NewLoggingService(logger *slog.Logger) *LoggingService {
	return &LoggingService{
		logger: logger,
	}
}

// EnableDebugMode enables debug mode with verbose logging.
// This method enables debug mode which provides more detailed logging.
func (l *LoggingService) EnableDebugMode() {
	l.debugMode.Store(true)
}

// DisableDebugMode disables debug mode.
// This method disables debug mode.
func (l *LoggingService) DisableDebugMode() {
	l.debugMode.Store(false)
}

// IsDebugMode checks if debug mode is enabled.
// This method checks if debug mode is enabled.
//
// Returns:
//   - bool: true if debug mode is enabled
func (l *LoggingService) IsDebugMode() bool {
	return l.debugMode.Load()
}

// GenerateCorrelationID generates a unique correlation ID.
// This method generates a unique correlation ID for request tracing.
//
// Returns:
//   - string: unique correlation ID
func (l *LoggingService) GenerateCorrelationID() string {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	l.correlationIDCounter++
	return fmt.Sprintf("corr-%d", l.correlationIDCounter)
}

// LogDebug logs a debug message.
// This method logs a debug message if debug mode is enabled.
//
// Parameters:
//   - ctx: context for the operation
//   - msg: message to log
//   - attrs: additional attributes
func (l *LoggingService) LogDebug(ctx context.Context, msg string, attrs ...slog.Attr) {
	if l.IsDebugMode() {
		// Convert slog.Attr to []any
		args := make([]any, len(attrs))
		for i, attr := range attrs {
			args[i] = attr
		}
		l.logger.DebugContext(ctx, msg, args...)
	}
}

// LogInfo logs an info message.
// This method logs an info message.
//
// Parameters:
//   - ctx: context for the operation
//   - msg: message to log
//   - attrs: additional attributes
func (l *LoggingService) LogInfo(ctx context.Context, msg string, attrs ...slog.Attr) {
	// Convert slog.Attr to []any
	args := make([]any, len(attrs))
	for i, attr := range attrs {
		args[i] = attr
	}
	l.logger.InfoContext(ctx, msg, args...)
}

// LogWarn logs a warning message.
// This method logs a warning message.
//
// Parameters:
//   - ctx: context for the operation
//   - msg: message to log
//   - attrs: additional attributes
func (l *LoggingService) LogWarn(ctx context.Context, msg string, attrs ...slog.Attr) {
	// Convert slog.Attr to []any
	args := make([]any, len(attrs))
	for i, attr := range attrs {
		args[i] = attr
	}
	l.logger.WarnContext(ctx, msg, args...)
}

// LogError logs an error message.
// This method logs an error message.
//
// Parameters:
//   - ctx: context for the operation
//   - msg: message to log
//   - err: error to log
//   - attrs: additional attributes
func (l *LoggingService) LogError(ctx context.Context, msg string, err error, attrs ...slog.Attr) {
	// Add error to attributes if it exists
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}
	
	// Convert slog.Attr to []any
	args := make([]any, len(attrs))
	for i, attr := range attrs {
		args[i] = attr
	}
	l.logger.ErrorContext(ctx, msg, args...)
}

// UpdateCorrelationID updates the correlation ID generation to include timestamp.
// This method enhances the existing correlation ID with timestamp for better tracing.
func (l *LoggingService) UpdateCorrelationID() string {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	l.correlationIDCounter++
	return fmt.Sprintf("req_%d_%d", time.Now().Unix(), l.correlationIDCounter)
}

// WithCorrelationID adds correlation ID to context.
// This method adds a correlation ID to the context for request tracing.
//
// Parameters:
//   - ctx: parent context
//   - correlationID: correlation ID to add
//
// Returns:
//   - context.Context: context with correlation ID
func (l *LoggingService) WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, CorrelationID, correlationID)
}

// GetCorrelationID extracts correlation ID from context.
// This method extracts the correlation ID from the context.
//
// Parameters:
//   - ctx: context to extract from
//
// Returns:
//   - string: correlation ID or empty string if not found
func (l *LoggingService) GetCorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(CorrelationID).(string); ok {
		return id
	}
	return ""
}

// LogWithCorrelation logs a message with correlation ID from context.
// This method logs a message with automatic correlation ID extraction.
//
// Parameters:
//   - ctx: context with correlation ID
//   - level: log level
//   - msg: message to log
//   - attrs: additional attributes
func (l *LoggingService) LogWithCorrelation(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	// Add correlation ID if present
	if correlationID := l.GetCorrelationID(ctx); correlationID != "" {
		attrs = append(attrs, slog.String("correlation_id", correlationID))
	}
	
	// Convert slog.Attr to []any
	args := make([]any, len(attrs))
	for i, attr := range attrs {
		args[i] = attr
	}
	
	// Log with appropriate level
	switch level {
	case slog.LevelDebug:
		if l.IsDebugMode() {
			l.logger.DebugContext(ctx, msg, args...)
		}
	case slog.LevelInfo:
		l.logger.InfoContext(ctx, msg, args...)
	case slog.LevelWarn:
		l.logger.WarnContext(ctx, msg, args...)
	case slog.LevelError:
		l.logger.ErrorContext(ctx, msg, args...)
	}
}

// LogOperation logs the start and end of an operation with timing.
// This method provides structured logging for operations with duration tracking.
//
// Parameters:
//   - ctx: context for the operation
//   - operation: name of the operation
//   - fn: function to execute
//   - attrs: additional attributes
//
// Returns:
//   - error: error from the operation if any
func (l *LoggingService) LogOperation(ctx context.Context, operation string, fn func() error, attrs ...slog.Attr) error {
	start := time.Now()
	correlationID := l.GetCorrelationID(ctx)
	
	// Log operation start
	startAttrs := make([]slog.Attr, 0, len(attrs)+4)
	startAttrs = append(startAttrs, attrs...)
	startAttrs = append(startAttrs,
		slog.String("operation", operation),
		slog.String("phase", "start"),
		slog.Time("start_time", start),
	)
	if correlationID != "" {
		startAttrs = append(startAttrs, slog.String("correlation_id", correlationID))
	}
	l.LogWithCorrelation(ctx, slog.LevelInfo, fmt.Sprintf("Starting operation: %s", operation), startAttrs...)
	
	// Execute operation
	err := fn()
	
	// Log operation end
	duration := time.Since(start)
	endAttrs := make([]slog.Attr, 0, len(attrs)+5)
	endAttrs = append(endAttrs, attrs...)
	endAttrs = append(endAttrs,
		slog.String("operation", operation),
		slog.String("phase", "end"),
		slog.Duration("duration", duration),
		slog.Bool("success", err == nil),
	)
	if correlationID != "" {
		endAttrs = append(endAttrs, slog.String("correlation_id", correlationID))
	}
	
	if err != nil {
		endAttrs = append(endAttrs, slog.String("error", err.Error()))
		l.LogWithCorrelation(ctx, slog.LevelError, fmt.Sprintf("Operation failed: %s", operation), endAttrs...)
	} else {
		l.LogWithCorrelation(ctx, slog.LevelInfo, fmt.Sprintf("Operation completed: %s", operation), endAttrs...)
	}
	
	return err
}

// ToDo: необходимо дополнительно реализовать следующий функционал:
// - Add unit tests for all methods
// - Implement better error handling and recovery
// - Add progress reporting during operations
//
// Ссылки на пункты плана и требований:
// - tasks.md: 9.2
// - requirements.md: 9.1