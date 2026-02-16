package tracing_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/Kargones/apk-ci/internal/pkg/logging"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLogger_WithTraceID проверяет что Logger.With() включает trace_id в записи лога.
// Это integration test для AC3: "все записи лога содержат trace_id".
func TestLogger_WithTraceID(t *testing.T) {
	// Создаём logger с JSON handler для удобной проверки
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := logging.NewSlogAdapter(slog.New(handler))

	// Генерируем trace_id и добавляем в context
	traceID := tracing.GenerateTraceID()
	ctx := tracing.WithTraceID(context.Background(), traceID)

	// Паттерн использования из Dev Notes
	loggerWithTrace := logger.With("trace_id", tracing.TraceIDFromContext(ctx))

	// Логируем сообщение
	loggerWithTrace.Info("операция выполнена", "action", "test")

	// Парсим JSON и проверяем trace_id
	var logEntry map[string]any
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err, "лог должен быть валидным JSON")

	// Проверяем что trace_id присутствует и имеет правильное значение
	actualTraceID, ok := logEntry["trace_id"].(string)
	assert.True(t, ok, "trace_id должен быть в логе")
	assert.Equal(t, traceID, actualTraceID, "trace_id в логе должен совпадать с сгенерированным")

	// Проверяем что остальные поля тоже присутствуют
	assert.Equal(t, "операция выполнена", logEntry["msg"])
	assert.Equal(t, "test", logEntry["action"])
}

// TestLogger_WithTraceID_MultipleMessages проверяет что trace_id сохраняется во всех сообщениях.
func TestLogger_WithTraceID_MultipleMessages(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := logging.NewSlogAdapter(slog.New(handler))

	traceID := tracing.GenerateTraceID()
	ctx := tracing.WithTraceID(context.Background(), traceID)
	loggerWithTrace := logger.With("trace_id", tracing.TraceIDFromContext(ctx))

	// Логируем несколько сообщений разных уровней
	loggerWithTrace.Debug("debug message")
	loggerWithTrace.Info("info message")
	loggerWithTrace.Warn("warn message")

	// Разбираем лог построчно (каждая строка - отдельный JSON)
	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	require.Len(t, lines, 3, "должно быть 3 записи лога")

	for i, line := range lines {
		var logEntry map[string]any
		err := json.Unmarshal(line, &logEntry)
		require.NoError(t, err, "строка %d должна быть валидным JSON", i)

		actualTraceID, ok := logEntry["trace_id"].(string)
		assert.True(t, ok, "строка %d должна содержать trace_id", i)
		assert.Equal(t, traceID, actualTraceID, "строка %d: trace_id должен совпадать", i)
	}
}

// TestLogger_WithTraceID_DifferentOperations проверяет что разные операции имеют разные trace_id.
func TestLogger_WithTraceID_DifferentOperations(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := logging.NewSlogAdapter(slog.New(handler))

	// Первая операция
	traceID1 := tracing.GenerateTraceID()
	ctx1 := tracing.WithTraceID(context.Background(), traceID1)
	logger.With("trace_id", tracing.TraceIDFromContext(ctx1)).Info("operation 1")

	// Вторая операция
	traceID2 := tracing.GenerateTraceID()
	ctx2 := tracing.WithTraceID(context.Background(), traceID2)
	logger.With("trace_id", tracing.TraceIDFromContext(ctx2)).Info("operation 2")

	// Проверяем что trace_id разные
	assert.NotEqual(t, traceID1, traceID2)

	// Проверяем в логе
	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	require.Len(t, lines, 2)

	var entry1, entry2 map[string]any
	require.NoError(t, json.Unmarshal(lines[0], &entry1))
	require.NoError(t, json.Unmarshal(lines[1], &entry2))

	assert.Equal(t, traceID1, entry1["trace_id"])
	assert.Equal(t, traceID2, entry2["trace_id"])
}

// TestLogger_WithTraceID_EmptyTraceID проверяет поведение когда trace_id пустой.
func TestLogger_WithTraceID_EmptyTraceID(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := logging.NewSlogAdapter(slog.New(handler))

	// Context без trace_id
	ctx := context.Background()
	traceID := tracing.TraceIDFromContext(ctx) // пустая строка

	logger.With("trace_id", traceID).Info("message without trace")

	var logEntry map[string]any
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	// trace_id должен быть пустой строкой
	actualTraceID, ok := logEntry["trace_id"].(string)
	assert.True(t, ok, "trace_id должен присутствовать даже если пустой")
	assert.Empty(t, actualTraceID, "trace_id должен быть пустой строкой")
}

// TestTraceID_Format_OpenTelemetryCompatible проверяет совместимость с W3C Trace Context.
// W3C Trace Context требует 32 символа hex (16 байт) для trace-id.
func TestTraceID_Format_OpenTelemetryCompatible(t *testing.T) {
	traceID := tracing.GenerateTraceID()

	// W3C Trace Context: trace-id = 32HEXDIGLC
	assert.Len(t, traceID, 32, "trace_id должен быть 32 символа для W3C совместимости")

	// Проверяем что все символы lowercase hex
	for _, c := range traceID {
		isLowercaseHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')
		assert.True(t, isLowercaseHex, "символ '%c' не является lowercase hex", c)
	}
}
