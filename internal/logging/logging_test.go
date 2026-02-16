package logging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"testing"
)

// TestNewSlogAdapter проверяет создание SlogAdapter
func TestNewSlogAdapter(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	adapter := NewSlogAdapter(logger)
	
	if adapter == nil {
		t.Fatal("Expected non-nil adapter")
	}
	
	if adapter.logger != logger {
		t.Error("Expected logger to be set correctly")
	}
	
	if adapter.IsDebugMode() {
		t.Error("Expected debug mode to be disabled by default")
	}
}

// TestSlogAdapterDebugMode проверяет управление режимом отладки
func TestSlogAdapterDebugMode(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	adapter := NewSlogAdapter(logger)
	
	// Проверяем начальное состояние
	if adapter.IsDebugMode() {
		t.Error("Expected debug mode to be disabled initially")
	}
	
	// Включаем режим отладки
	adapter.EnableDebugMode()
	if !adapter.IsDebugMode() {
		t.Error("Expected debug mode to be enabled")
	}
	
	// Отключаем режим отладки
	adapter.DisableDebugMode()
	if adapter.IsDebugMode() {
		t.Error("Expected debug mode to be disabled")
	}
}

// TestSlogAdapterCorrelationID проверяет работу с correlation ID
func TestSlogAdapterCorrelationID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	adapter := NewSlogAdapter(logger)
	
	// Генерируем correlation ID
	correlationID := adapter.GenerateCorrelationID()
	if correlationID == "" {
		t.Error("Expected non-empty correlation ID")
	}
	
	// Проверяем уникальность
	correlationID2 := adapter.GenerateCorrelationID()
	if correlationID == correlationID2 {
		t.Error("Expected unique correlation IDs")
	}
	
	// Проверяем работу с контекстом
	ctx := context.Background()
	ctxWithID := adapter.WithCorrelationID(ctx, correlationID)
	
	retrievedID := adapter.GetCorrelationID(ctxWithID)
	if retrievedID != correlationID {
		t.Errorf("Expected correlation ID %s, got %s", correlationID, retrievedID)
	}
	
	// Проверяем пустой контекст
	emptyID := adapter.GetCorrelationID(ctx)
	if emptyID != "" {
		t.Error("Expected empty correlation ID for context without ID")
	}
}

// TestSlogAdapterLogOperation проверяет логирование операций
func TestSlogAdapterLogOperation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	adapter := NewSlogAdapter(logger)
	ctx := context.Background()
	
	// Тест успешной операции
	err := adapter.LogOperation(ctx, "test_operation", func() error {
		return nil
	})
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Тест операции с ошибкой
	testErr := errors.New("test error")
	err = adapter.LogOperation(ctx, "failing_operation", func() error {
		return testErr
	})
	
	if err != testErr {
		t.Errorf("Expected error %v, got %v", testErr, err)
	}
}

// TestNewSimpleLoggerAdapter проверяет создание SimpleLoggerAdapter
func TestNewSimpleLoggerAdapter(t *testing.T) {
	mockLogger := &mockSimpleLogger{}
	adapter := NewSimpleLoggerAdapter(mockLogger)
	
	if adapter == nil {
		t.Fatal("Expected non-nil adapter")
	}
	
	if adapter.logger != mockLogger {
		t.Error("Expected logger to be set correctly")
	}
}

// TestLogLevel проверяет работу с уровнями логирования
func TestLogLevel(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
	}
	
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.level.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.level.String())
			}
		})
	}
}

// TestLogLevelToSlogLevel проверяет конвертацию в slog.Level
func TestLogLevelToSlogLevel(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected slog.Level
	}{
		{LevelDebug, slog.LevelDebug},
		{LevelInfo, slog.LevelInfo},
		{LevelWarn, slog.LevelWarn},
		{LevelError, slog.LevelError},
	}
	
	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			if tt.level.ToSlogLevel() != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, tt.level.ToSlogLevel())
			}
		})
	}
}

// TestDefaultLogger проверяет создание логгера по умолчанию
func TestDefaultLogger(t *testing.T) {
	logger := DefaultLogger()
	if logger == nil {
		t.Error("Expected non-nil logger")
	}
}

// TestDebugLogger проверяет создание отладочного логгера
func TestDebugLogger(t *testing.T) {
	logger := DebugLogger()
	if logger == nil {
		t.Error("Expected non-nil logger")
	}
}

// TestTextLogger проверяет создание текстового логгера
func TestTextLogger(t *testing.T) {
	logger := TextLogger()
	if logger == nil {
		t.Error("Expected non-nil logger")
	}
}

// TestNewStructuredLogger проверяет создание структурированного логгера
func TestNewStructuredLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	structuredLogger := NewStructuredLogger(logger)
	
	if structuredLogger == nil {
		t.Error("Expected non-nil structured logger")
	}
}

// TestWithFields проверяет добавление полей в контекст
func TestWithFields(t *testing.T) {
	ctx := context.Background()
	fields := map[string]any{
		"key1": "value1",
		"key2": 42,
	}
	
	ctxWithFields := WithFields(ctx, fields)
	if ctxWithFields == nil {
		t.Error("Expected non-nil context")
	}
	
	// Проверяем получение поля
	value := GetField(ctxWithFields, "key1")
	if value != "value1" {
		t.Errorf("Expected 'value1', got %v", value)
	}
	
	// Проверяем несуществующее поле
	value = GetField(ctxWithFields, "nonexistent")
	if value != nil {
		t.Errorf("Expected nil for nonexistent field, got %v", value)
	}
}

// TestWithLogger проверяет добавление логгера в контекст
func TestWithLogger(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	structuredLogger := NewSlogAdapter(logger)
	
	ctxWithLogger := WithLogger(ctx, structuredLogger)
	retrievedLogger := LoggerFromContext(ctxWithLogger)
	
	if retrievedLogger != structuredLogger {
		t.Error("Expected same logger instance")
	}
}

// TestCreateOperationContext проверяет создание контекста операции
func TestCreateOperationContext(t *testing.T) {
	parentCtx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	structuredLogger := NewSlogAdapter(logger)
	
	ctx, correlationID := CreateOperationContext(parentCtx, structuredLogger, "test_operation")
	
	if ctx == nil {
		t.Error("Expected non-nil context")
	}
	
	if correlationID == "" {
		t.Error("Expected non-empty correlation ID")
	}
	
	// Проверяем, что логгер добавлен в контекст
	retrievedLogger := LoggerFromContext(ctx)
	if retrievedLogger == nil {
		t.Error("Expected logger in context")
	}
}

// mockSimpleLogger для тестирования SimpleLoggerAdapter
type mockSimpleLogger struct {
	messages []string
}

func (m *mockSimpleLogger) Debug(msg string, args ...any) {
	m.messages = append(m.messages, fmt.Sprintf("DEBUG: %s %v", msg, args))
}

func (m *mockSimpleLogger) Info(msg string, args ...any) {
	m.messages = append(m.messages, fmt.Sprintf("INFO: %s %v", msg, args))
}

func (m *mockSimpleLogger) Warn(msg string, args ...any) {
	m.messages = append(m.messages, fmt.Sprintf("WARN: %s %v", msg, args))
}

func (m *mockSimpleLogger) Error(msg string, args ...any) {
	m.messages = append(m.messages, fmt.Sprintf("ERROR: %s %v", msg, args))
}

// TestGetField проверяет извлечение полей из контекста
func TestGetField(t *testing.T) {
	ctx := context.Background()
	fields := map[string]any{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}
	
	ctx = WithFields(ctx, fields)
	
	// Проверяем извлечение существующих полей
	if value := GetField(ctx, "key1"); value != "value1" {
		t.Errorf("Expected 'value1', got %v", value)
	}
	
	if value := GetField(ctx, "key2"); value != 42 {
		t.Errorf("Expected 42, got %v", value)
	}
	
	if value := GetField(ctx, "key3"); value != true {
		t.Errorf("Expected true, got %v", value)
	}
	
	// Проверяем извлечение несуществующего поля
	if value := GetField(ctx, "nonexistent"); value != nil {
		t.Errorf("Expected nil for nonexistent field, got %v", value)
	}
}

// TestLoggerFromContext проверяет извлечение логгера из контекста
func TestLoggerFromContext(t *testing.T) {
	ctx := context.Background()
	
	// Тест без логгера в контексте - должен вернуть дефолтный
	logger1 := LoggerFromContext(ctx)
	if logger1 == nil {
		t.Error("Expected non-nil logger")
	}
	
	// Тест с логгером в контексте
	slogLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	structuredLogger := NewSlogAdapter(slogLogger)
	ctx = WithLogger(ctx, structuredLogger)
	
	logger2 := LoggerFromContext(ctx)
	if logger2 != structuredLogger {
		t.Error("Expected same logger instance")
	}
}

// TestSimpleLoggerAdapterMethods проверяет методы SimpleLoggerAdapter
func TestSimpleLoggerAdapterMethods(t *testing.T) {
	mock := &mockSimpleLogger{}
	adapter := NewSimpleLoggerAdapter(mock)
	
	adapter.Debug("debug message", "arg1", "arg2")
	adapter.Info("info message", "arg1")
	adapter.Warn("warn message")
	adapter.Error("error message", "error_arg")
	
	expected := []string{
		"DEBUG: debug message [arg1 arg2]",
		"INFO: info message [arg1]",
		"WARN: warn message []",
		"ERROR: error message [error_arg]",
	}
	
	if len(mock.messages) != len(expected) {
		t.Errorf("Expected %d messages, got %d", len(expected), len(mock.messages))
	}
	
	for i, msg := range expected {
		if i < len(mock.messages) && mock.messages[i] != msg {
			t.Errorf("Expected message %d to be '%s', got '%s'", i, msg, mock.messages[i])
		}
	}
}

// TestConvenienceLogFunctions проверяет удобные функции логирования
func TestConvenienceLogFunctions(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	adapter := NewSlogAdapter(logger)
	ctx := context.Background()
	
	// Тестируем функции без паники
	LogError(ctx, adapter, "test error", errors.New("test"))
	LogInfo(ctx, adapter, "test info")
	LogDebug(ctx, adapter, "test debug")
	LogWarn(ctx, adapter, "test warn")
	
	// Если дошли до этой точки без паники, тест прошел
}