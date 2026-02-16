package tracing_test

import (
	"context"
	"fmt"

	"github.com/Kargones/apk-ci/internal/pkg/tracing"
)

// ExampleGenerateTraceID демонстрирует генерацию trace ID.
func ExampleGenerateTraceID() {
	traceID := tracing.GenerateTraceID()

	// trace ID — это 32-символьный hex string
	fmt.Printf("Длина trace ID: %d\n", len(traceID))
	fmt.Printf("Формат: hex string\n")
	// Output:
	// Длина trace ID: 32
	// Формат: hex string
}

// ExampleWithTraceID демонстрирует добавление trace ID в context.
func ExampleWithTraceID() {
	ctx := context.Background()

	// Генерируем и добавляем trace ID
	traceID := tracing.GenerateTraceID()
	ctx = tracing.WithTraceID(ctx, traceID)

	// Извлекаем обратно
	extracted := tracing.TraceIDFromContext(ctx)
	fmt.Printf("Trace ID сохранён: %t\n", extracted == traceID)
	// Output:
	// Trace ID сохранён: true
}

// ExampleTraceIDFromContext демонстрирует извлечение trace ID из context.
func ExampleTraceIDFromContext() {
	// Пустой context — возвращает пустую строку
	ctx := context.Background()
	traceID := tracing.TraceIDFromContext(ctx)
	fmt.Printf("Пустой context: '%s'\n", traceID)

	// Context с trace ID
	ctx = tracing.WithTraceID(ctx, "abc123")
	traceID = tracing.TraceIDFromContext(ctx)
	fmt.Printf("С trace ID: '%s'\n", traceID)
	// Output:
	// Пустой context: ''
	// С trace ID: 'abc123'
}

// ExampleTraceIDFromContext_withLogger демонстрирует типичный паттерн использования с логгером.
func ExampleTraceIDFromContext_withLogger() {
	// Генерируем trace ID на старте операции
	traceID := tracing.GenerateTraceID()
	ctx := tracing.WithTraceID(context.Background(), traceID)

	// В любом месте кода можем получить trace ID для логирования:
	// logger.With("trace_id", tracing.TraceIDFromContext(ctx)).Info("сообщение")

	extracted := tracing.TraceIDFromContext(ctx)
	fmt.Printf("Trace ID доступен: %t\n", len(extracted) == 32)
	// Output:
	// Trace ID доступен: true
}
