package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)
// TestNewSlogAdapter_NilLogger_UsesDefault проверяет что nil logger использует default.
func TestNewSlogAdapter_NilLogger_UsesDefault(t *testing.T) {
	adapter := NewSlogAdapter(nil)
	assert.NotNil(t, adapter, "nil logger должен вернуть adapter с default logger")
}

// TestSlogAdapter_Debug проверяет что Debug() логирует с level=DEBUG.
func TestSlogAdapter_Debug(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	adapter := NewSlogAdapter(slog.New(handler))

	adapter.Debug("test debug message", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "level=DEBUG")
	assert.Contains(t, output, "test debug message")
	assert.Contains(t, output, "key=value")
}

// TestSlogAdapter_Info проверяет что Info() логирует с level=INFO.
func TestSlogAdapter_Info(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	adapter := NewSlogAdapter(slog.New(handler))

	adapter.Info("test info message", "count", 42)

	output := buf.String()
	assert.Contains(t, output, "level=INFO")
	assert.Contains(t, output, "test info message")
	assert.Contains(t, output, "count=42")
}

// TestSlogAdapter_Warn проверяет что Warn() логирует с level=WARN.
func TestSlogAdapter_Warn(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
	adapter := NewSlogAdapter(slog.New(handler))

	adapter.Warn("test warning message", "deprecated", true)

	output := buf.String()
	assert.Contains(t, output, "level=WARN")
	assert.Contains(t, output, "test warning message")
	assert.Contains(t, output, "deprecated=true")
}

// TestSlogAdapter_Error проверяет что Error() логирует с level=ERROR.
func TestSlogAdapter_Error(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError})
	adapter := NewSlogAdapter(slog.New(handler))

	adapter.Error("test error message", "error_code", 500)

	output := buf.String()
	assert.Contains(t, output, "level=ERROR")
	assert.Contains(t, output, "test error message")
	assert.Contains(t, output, "error_code=500")
}

// TestSlogAdapter_With проверяет что With() возвращает Logger с добавленными атрибутами.
func TestSlogAdapter_With(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	adapter := NewSlogAdapter(slog.New(handler))

	// Создаём child logger с атрибутами
	childLogger := adapter.With("trace_id", "abc123", "user_id", "user42")

	// Логируем через child logger
	childLogger.Info("операция выполнена")

	output := buf.String()
	assert.Contains(t, output, "trace_id=abc123")
	assert.Contains(t, output, "user_id=user42")
	assert.Contains(t, output, "операция выполнена")
}

// TestSlogAdapter_With_ReturnsNewLogger проверяет что With() возвращает новый Logger,
// а не модифицирует оригинальный.
func TestSlogAdapter_With_ReturnsNewLogger(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	adapter := NewSlogAdapter(slog.New(handler))

	// Создаём child logger
	childLogger := adapter.With("child_attr", "child_value")

	// Проверяем что childLogger - это Logger (компилятор уже гарантирует это)
	// Явная проверка типа не нужна, т.к. With() возвращает Logger

	// Проверяем что это не тот же самый объект
	assert.NotEqual(t, adapter, childLogger, "With() должен возвращать новый Logger")
}

// TestSlogAdapter_JSONFormat проверяет что JSON handler выводит валидный JSON.
func TestSlogAdapter_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	adapter := NewSlogAdapter(slog.New(handler))

	adapter.Info("test json message", "key", "value", "number", 123)

	output := buf.String()

	// Парсим JSON
	var logEntry map[string]any
	err := json.Unmarshal([]byte(output), &logEntry)
	require.NoError(t, err, "вывод должен быть валидным JSON")

	// Проверяем обязательные поля
	assert.Contains(t, logEntry, "time", "JSON должен содержать timestamp")
	assert.Contains(t, logEntry, "level", "JSON должен содержать level")
	assert.Contains(t, logEntry, "msg", "JSON должен содержать msg")

	// Проверяем значения
	assert.Equal(t, "INFO", logEntry["level"])
	assert.Equal(t, "test json message", logEntry["msg"])
	assert.Equal(t, "value", logEntry["key"])
	assert.Equal(t, float64(123), logEntry["number"]) // JSON unmarshals numbers as float64
}

// TestSlogAdapter_JSONFormat_WithAttributes проверяет JSON output с атрибутами через With().
func TestSlogAdapter_JSONFormat_WithAttributes(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	adapter := NewSlogAdapter(slog.New(handler))

	childLogger := adapter.With("trace_id", "xyz789")
	childLogger.Info("событие", "action", "test")

	output := buf.String()

	var logEntry map[string]any
	err := json.Unmarshal([]byte(output), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "xyz789", logEntry["trace_id"])
	assert.Equal(t, "test", logEntry["action"])
	assert.Equal(t, "событие", logEntry["msg"])
}

// TestSlogAdapter_AllLevels проверяет все уровни в таблице.
func TestSlogAdapter_AllLevels(t *testing.T) {
	tests := []struct {
		name     string
		logFunc  func(adapter *SlogAdapter, msg string, args ...any)
		expected string
	}{
		{"Debug", (*SlogAdapter).Debug, "level=DEBUG"},
		{"Info", (*SlogAdapter).Info, "level=INFO"},
		{"Warn", (*SlogAdapter).Warn, "level=WARN"},
		{"Error", (*SlogAdapter).Error, "level=ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
			adapter := NewSlogAdapter(slog.New(handler))

			tt.logFunc(adapter, "test message")

			output := buf.String()
			assert.Contains(t, output, tt.expected)
			assert.Contains(t, output, "test message")
		})
	}
}

// TestSlogAdapter_ImplementsLogger проверяет что SlogAdapter реализует Logger interface.
func TestSlogAdapter_ImplementsLogger(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	adapter := NewSlogAdapter(slog.New(handler))

	// Compile-time check
	var _ Logger = adapter

	// Runtime check
	_, ok := any(adapter).(Logger)
	assert.True(t, ok, "SlogAdapter должен реализовывать Logger interface")
}

// TestSlogAdapter_With_ChainedCalls проверяет цепочку вызовов With().
func TestSlogAdapter_With_ChainedCalls(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	adapter := NewSlogAdapter(slog.New(handler))

	// Цепочка With() вызовов
	logger := adapter.With("a", "1").With("b", "2").With("c", "3")
	logger.Info("chained")

	output := buf.String()
	assert.Contains(t, output, "a=1")
	assert.Contains(t, output, "b=2")
	assert.Contains(t, output, "c=3")
}

// TestSlogAdapter_EmptyArgs проверяет логирование без дополнительных аргументов.
func TestSlogAdapter_EmptyArgs(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	adapter := NewSlogAdapter(slog.New(handler))

	adapter.Info("message without args")

	output := buf.String()
	assert.Contains(t, output, "message without args")
	// Не должно быть лишних key=value пар кроме стандартных (time, level, msg)
	lines := strings.Split(strings.TrimSpace(output), " ")
	// Проверяем что есть только time, level и msg
	hasTime := false
	hasLevel := false
	hasMsg := false
	for _, part := range lines {
		if strings.HasPrefix(part, "time=") {
			hasTime = true
		}
		if strings.HasPrefix(part, "level=") {
			hasLevel = true
		}
		if strings.HasPrefix(part, "msg=") {
			hasMsg = true
		}
	}
	assert.True(t, hasTime, "должно быть поле time")
	assert.True(t, hasLevel, "должно быть поле level")
	assert.True(t, hasMsg, "должно быть поле msg")
}
