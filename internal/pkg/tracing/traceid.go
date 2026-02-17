// Package tracing предоставляет функции для генерации и управления trace ID.
// Trace ID используется для корреляции логов одной операции.
//
// Формат trace ID: 32-символьный hex string (16 байт), например:
//
//	"a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6"
//
// Это совместимо с W3C Trace Context format для будущей интеграции с OpenTelemetry.
//
// Пример использования:
//
//	traceID := tracing.GenerateTraceID()
//	ctx := tracing.WithTraceID(ctx, traceID)
//	logger.With("trace_id", tracing.TraceIDFromContext(ctx)).Info("Операция началась")
package tracing

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"
)

// fallbackCounter используется для генерации уникальных fallback ID.
// Является atomic-счётчиком, не может быть константой.
var fallbackCounter atomic.Uint64

// GenerateTraceID генерирует уникальный trace ID.
// Формат: 32 символа hex (16 байт), например: "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6".
//
// Использует crypto/rand для криптографически безопасной генерации.
// При ошибке crypto/rand (что практически невозможно на современных системах)
// возвращает fallback значение на основе timestamp и счётчика.
//
// Пример:
//
//	traceID := tracing.GenerateTraceID()
//	// traceID: "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6"
func GenerateTraceID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback на timestamp-based ID (практически никогда не должно происходить)
		return fallbackTraceID()
	}
	return hex.EncodeToString(b)
}

// fallbackTraceID генерирует ID на основе текущего времени и счётчика.
// Используется только если crypto/rand недоступен.
// Формат обеспечивает уникальность через комбинацию timestamp (наносекунды) и счётчика.
//
// H7 safety: %016x для int64/uint64 гарантирует ровно 16 hex символов:
// - int64 timestamp: максимум 0x7fffffffffffffff (16 символов)
// - uint64 counter: максимум 0xffffffffffffffff (16 символов)
// Итого: всегда ровно 32 символа.
func fallbackTraceID() string {
	counter := fallbackCounter.Add(1)
	// Cast к uint64 для однозначности (int64 UnixNano может быть отрицательным на древних системах)
	timestamp := uint64(time.Now().UnixNano())
	// Формируем 32-символьный hex string из timestamp (16 символов) и counter (16 символов)
	return fmt.Sprintf("%016x%016x", timestamp, counter)
}
