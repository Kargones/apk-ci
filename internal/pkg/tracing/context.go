package tracing

import "context"

// traceIDKey — ключ для хранения trace ID в context.
// Приватный тип предотвращает коллизии ключей с другими пакетами.
type traceIDKey struct{}

// WithTraceID возвращает новый context с добавленным trace ID.
// Если trace ID уже установлен, он будет перезаписан.
//
// Пример использования:
//
//	traceID := tracing.GenerateTraceID()
//	ctx = tracing.WithTraceID(ctx, traceID)
//	// Теперь trace_id доступен через TraceIDFromContext(ctx)
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, id)
}

// TraceIDFromContext извлекает trace ID из context.
// Возвращает пустую строку если trace ID не установлен или context == nil.
//
// Пример использования:
//
//	traceID := tracing.TraceIDFromContext(ctx)
//	if traceID != "" {
//	    logger.With("trace_id", traceID).Info("Операция выполнена")
//	}
func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(traceIDKey{}).(string); ok {
		return id
	}
	return ""
}
