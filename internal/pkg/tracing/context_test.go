package tracing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestWithTraceID_AddsToContext проверяет что WithTraceID добавляет trace_id в context.
func TestWithTraceID_AddsToContext(t *testing.T) {
	ctx := context.Background()
	traceID := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6"

	ctx = WithTraceID(ctx, traceID)

	result := TraceIDFromContext(ctx)
	assert.Equal(t, traceID, result, "trace_id должен быть извлечён из context")
}

// TestTraceIDFromContext_Success проверяет успешное извлечение trace_id.
func TestTraceIDFromContext_Success(t *testing.T) {
	ctx := context.Background()
	expectedID := "1234567890abcdef1234567890abcdef"

	ctx = WithTraceID(ctx, expectedID)
	actualID := TraceIDFromContext(ctx)

	assert.Equal(t, expectedID, actualID)
}

// TestTraceIDFromContext_EmptyContext проверяет что возвращается пустая строка для context без trace_id.
func TestTraceIDFromContext_EmptyContext(t *testing.T) {
	ctx := context.Background()

	result := TraceIDFromContext(ctx)

	assert.Empty(t, result, "должна возвращаться пустая строка для context без trace_id")
}

// TestTraceIDFromContext_NilContext проверяет что функция не паникует для nil context.
func TestTraceIDFromContext_NilContext(t *testing.T) {
	// Не должно паниковать
	assert.NotPanics(t, func() {
		//nolint:staticcheck // SA1012: тестируем nil context специально
		result := TraceIDFromContext(nil)
		assert.Empty(t, result)
	}, "TraceIDFromContext не должен паниковать при nil context")
}

// TestTraceIDFromContext_WrongKey проверяет что возвращается пустая строка если trace_id установлен другим ключом.
func TestTraceIDFromContext_WrongKey(t *testing.T) {
	// Используем другой ключ для установки значения
	type otherKey struct{}
	ctx := context.WithValue(context.Background(), otherKey{}, "some-trace-id")

	result := TraceIDFromContext(ctx)

	assert.Empty(t, result, "должна возвращаться пустая строка если trace_id установлен другим ключом")
}

// TestWithTraceID_OverwritesPrevious проверяет что повторный WithTraceID перезаписывает предыдущий.
func TestWithTraceID_OverwritesPrevious(t *testing.T) {
	ctx := context.Background()

	ctx = WithTraceID(ctx, "first-trace-id")
	ctx = WithTraceID(ctx, "second-trace-id")

	result := TraceIDFromContext(ctx)
	assert.Equal(t, "second-trace-id", result, "последний trace_id должен перезаписать предыдущий")
}

// TestWithTraceID_PreservesOtherValues проверяет что WithTraceID не теряет другие значения в context.
func TestWithTraceID_PreservesOtherValues(t *testing.T) {
	type userIDKey struct{}
	ctx := context.WithValue(context.Background(), userIDKey{}, "user-123")

	ctx = WithTraceID(ctx, "trace-456")

	// Проверяем что trace_id установлен
	assert.Equal(t, "trace-456", TraceIDFromContext(ctx))

	// Проверяем что user_id сохранился
	userID, ok := ctx.Value(userIDKey{}).(string)
	assert.True(t, ok, "user_id должен быть доступен")
	assert.Equal(t, "user-123", userID)
}

// TestWithTraceID_EmptyString проверяет установку пустой строки как trace_id.
func TestWithTraceID_EmptyString(t *testing.T) {
	ctx := context.Background()

	ctx = WithTraceID(ctx, "")

	result := TraceIDFromContext(ctx)
	assert.Empty(t, result, "пустая строка должна возвращаться как пустая строка")
}

// TestTraceIDFromContext_Roundtrip проверяет полный цикл установки и извлечения.
func TestTraceIDFromContext_Roundtrip(t *testing.T) {
	tests := []struct {
		name    string
		traceID string
	}{
		{"стандартный trace_id", "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6"},
		{"минимальный trace_id", "0000000000000000"},
		{"максимальный trace_id", "ffffffffffffffffffffffffffffffff"},
		{"пустая строка", ""},
		{"нестандартная длина", "short"},
		{"с буквами", "abcdef1234567890"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = WithTraceID(ctx, tt.traceID)
			result := TraceIDFromContext(ctx)
			assert.Equal(t, tt.traceID, result)
		})
	}
}
