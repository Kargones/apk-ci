// Package logging provides common interfaces and utilities for structured logging.
// This package defines standard interfaces for logging that can be implemented
// by different logging services throughout the application.
package logging

import (
	"context"
	"log/slog"
)

// StructuredLogger defines the interface for structured logging with correlation IDs.
// This interface provides methods for logging with different levels and correlation tracking.
type StructuredLogger interface {
	// Basic logging methods
	LogDebug(ctx context.Context, msg string, attrs ...slog.Attr)
	LogInfo(ctx context.Context, msg string, attrs ...slog.Attr)
	LogWarn(ctx context.Context, msg string, attrs ...slog.Attr)
	LogError(ctx context.Context, msg string, err error, attrs ...slog.Attr)
	
	// Correlation ID methods
	GenerateCorrelationID() string
	WithCorrelationID(ctx context.Context, correlationID string) context.Context
	GetCorrelationID(ctx context.Context) string
	
	// Advanced logging methods
	LogWithCorrelation(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr)
	LogOperation(ctx context.Context, operation string, fn func() error, attrs ...slog.Attr) error
	
	// Debug mode control
	IsDebugMode() bool
	EnableDebugMode()
	DisableDebugMode()
}

// SimpleLogger defines a simplified logging interface for basic use cases.
// This interface provides basic logging methods without correlation tracking.
type SimpleLogger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// LogLevel represents different logging levels.
type LogLevel int

const (
	// LevelDebug represents debug level logging
	LevelDebug LogLevel = iota
	// LevelInfo represents info level logging
	LevelInfo
	// LevelWarn represents warning level logging
	LevelWarn
	// LevelError represents error level logging
	LevelError
)

// String returns the string representation of the log level.
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ToSlogLevel converts LogLevel to slog.Level.
func (l LogLevel) ToSlogLevel() slog.Level {
	switch l {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}