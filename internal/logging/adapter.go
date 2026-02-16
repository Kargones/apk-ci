// Package logging provides adapters for integrating different logging implementations.
// This package contains adapters that allow existing loggers to work with
// the new structured logging interfaces.
package logging

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// SlogAdapter adapts slog.Logger to implement StructuredLogger interface.
// This adapter provides structured logging capabilities with correlation IDs
// for existing slog.Logger instances.
type SlogAdapter struct {
	// logger is the underlying slog.Logger
	logger *slog.Logger
	
	// debugMode enables verbose logging
	debugMode atomic.Bool
	
	// correlationIDCounter is used to generate unique correlation IDs
	correlationIDCounter uint64
	
	// mutex for thread-safe operations
	mutex sync.Mutex
}

// CorrelationIDKey is the context key for correlation IDs
type CorrelationIDKey string

const (
	// CorrelationID is the key for correlation ID in context
	CorrelationID CorrelationIDKey = "correlation_id"
)

// NewSlogAdapter creates a new SlogAdapter instance.
// This function initializes the adapter with the provided slog.Logger.
//
// Parameters:
//   - logger: the slog.Logger to wrap
//
// Returns:
//   - *SlogAdapter: new adapter instance
func NewSlogAdapter(logger *slog.Logger) *SlogAdapter {
	return &SlogAdapter{
		logger: logger,
	}
}

// LogDebug logs a debug message.
func (s *SlogAdapter) LogDebug(ctx context.Context, msg string, attrs ...slog.Attr) {
	if s.IsDebugMode() {
		args := make([]any, len(attrs))
		for i, attr := range attrs {
			args[i] = attr
		}
		s.logger.DebugContext(ctx, msg, args...)
	}
}

// LogInfo logs an info message.
func (s *SlogAdapter) LogInfo(ctx context.Context, msg string, attrs ...slog.Attr) {
	args := make([]any, len(attrs))
	for i, attr := range attrs {
		args[i] = attr
	}
	s.logger.InfoContext(ctx, msg, args...)
}

// LogWarn logs a warning message.
func (s *SlogAdapter) LogWarn(ctx context.Context, msg string, attrs ...slog.Attr) {
	args := make([]any, len(attrs))
	for i, attr := range attrs {
		args[i] = attr
	}
	s.logger.WarnContext(ctx, msg, args...)
}

// LogError logs an error message.
func (s *SlogAdapter) LogError(ctx context.Context, msg string, err error, attrs ...slog.Attr) {
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}
	
	args := make([]any, len(attrs))
	for i, attr := range attrs {
		args[i] = attr
	}
	s.logger.ErrorContext(ctx, msg, args...)
}

// GenerateCorrelationID generates a new unique correlation ID.
func (s *SlogAdapter) GenerateCorrelationID() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.correlationIDCounter++
	return fmt.Sprintf("req_%d_%d", time.Now().Unix(), s.correlationIDCounter)
}

// WithCorrelationID adds correlation ID to context.
func (s *SlogAdapter) WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, CorrelationID, correlationID)
}

// GetCorrelationID extracts correlation ID from context.
func (s *SlogAdapter) GetCorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(CorrelationID).(string); ok {
		return id
	}
	return ""
}

// LogWithCorrelation logs a message with correlation ID from context.
func (s *SlogAdapter) LogWithCorrelation(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	// Add correlation ID if present
	if correlationID := s.GetCorrelationID(ctx); correlationID != "" {
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
		if s.IsDebugMode() {
			s.logger.DebugContext(ctx, msg, args...)
		}
	case slog.LevelInfo:
		s.logger.InfoContext(ctx, msg, args...)
	case slog.LevelWarn:
		s.logger.WarnContext(ctx, msg, args...)
	case slog.LevelError:
		s.logger.ErrorContext(ctx, msg, args...)
	}
}

// LogOperation logs the start and end of an operation with timing.
func (s *SlogAdapter) LogOperation(ctx context.Context, operation string, fn func() error, attrs ...slog.Attr) error {
	start := time.Now()
	correlationID := s.GetCorrelationID(ctx)
	
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
	s.LogWithCorrelation(ctx, slog.LevelInfo, fmt.Sprintf("Starting operation: %s", operation), startAttrs...)
	
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
		s.LogWithCorrelation(ctx, slog.LevelError, fmt.Sprintf("Operation failed: %s", operation), endAttrs...)
	} else {
		s.LogWithCorrelation(ctx, slog.LevelInfo, fmt.Sprintf("Operation completed: %s", operation), endAttrs...)
	}
	
	return err
}

// IsDebugMode returns whether debug mode is enabled.
func (s *SlogAdapter) IsDebugMode() bool {
	return s.debugMode.Load()
}

// EnableDebugMode enables debug mode with verbose logging.
func (s *SlogAdapter) EnableDebugMode() {
	s.debugMode.Store(true)
}

// DisableDebugMode disables debug mode.
func (s *SlogAdapter) DisableDebugMode() {
	s.debugMode.Store(false)
}

// SimpleLoggerAdapter adapts any logger that implements SimpleLogger interface.
// This adapter provides a bridge between simple loggers and structured logging.
type SimpleLoggerAdapter struct {
	logger SimpleLogger
}

// NewSimpleLoggerAdapter creates a new SimpleLoggerAdapter.
func NewSimpleLoggerAdapter(logger SimpleLogger) *SimpleLoggerAdapter {
	return &SimpleLoggerAdapter{
		logger: logger,
	}
}

// Debug logs a debug message using the simple logger.
func (s *SimpleLoggerAdapter) Debug(msg string, args ...any) {
	s.logger.Debug(msg, args...)
}

// Info logs an info message using the simple logger.
func (s *SimpleLoggerAdapter) Info(msg string, args ...any) {
	s.logger.Info(msg, args...)
}

// Warn logs a warning message using the simple logger.
func (s *SimpleLoggerAdapter) Warn(msg string, args ...any) {
	s.logger.Warn(msg, args...)
}

// Error logs an error message using the simple logger.
func (s *SimpleLoggerAdapter) Error(msg string, args ...any) {
	s.logger.Error(msg, args...)
}