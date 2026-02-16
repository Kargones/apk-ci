package tracing

import "context"

// NewNopTracerProvider возвращает nop shutdown function.
// Используется когда трейсинг выключен — нулевой overhead.
func NewNopTracerProvider() func(context.Context) error {
	return func(_ context.Context) error { return nil }
}
