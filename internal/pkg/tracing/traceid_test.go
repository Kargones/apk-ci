package tracing

import (
	"regexp"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateTraceID_Format проверяет что GenerateTraceID возвращает 32-символьный hex string.
func TestGenerateTraceID_Format(t *testing.T) {
	traceID := GenerateTraceID()

	assert.Len(t, traceID, 32, "trace_id должен быть длиной 32 символа")
}

// TestGenerateTraceID_ValidHex проверяет что trace_id содержит только hex символы [0-9a-f].
func TestGenerateTraceID_ValidHex(t *testing.T) {
	traceID := GenerateTraceID()

	// Регулярное выражение для проверки hex string
	hexPattern := regexp.MustCompile("^[0-9a-f]{32}$")
	assert.True(t, hexPattern.MatchString(traceID), "trace_id должен содержать только hex символы [0-9a-f], got: %s", traceID)
}

// TestGenerateTraceID_Unique проверяет что два вызова возвращают разные значения.
func TestGenerateTraceID_Unique(t *testing.T) {
	traceID1 := GenerateTraceID()
	traceID2 := GenerateTraceID()

	assert.NotEqual(t, traceID1, traceID2, "два вызова GenerateTraceID должны возвращать разные значения")
}

// TestGenerateTraceID_UniqueInConcurrentCalls проверяет уникальность при конкурентных вызовах.
func TestGenerateTraceID_UniqueInConcurrentCalls(t *testing.T) {
	const numGoroutines = 100
	ids := make(chan string, numGoroutines)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			ids <- GenerateTraceID()
		}()
	}

	wg.Wait()
	close(ids)

	// Собираем все ID в map для проверки уникальности
	seen := make(map[string]bool)
	for id := range ids {
		require.False(t, seen[id], "обнаружен дубликат trace_id: %s", id)
		seen[id] = true
	}

	assert.Len(t, seen, numGoroutines, "должно быть сгенерировано %d уникальных trace_id", numGoroutines)
}

// TestGenerateTraceID_MultipleCallsAllValid проверяет что множественные вызовы возвращают валидные ID.
func TestGenerateTraceID_MultipleCallsAllValid(t *testing.T) {
	hexPattern := regexp.MustCompile("^[0-9a-f]{32}$")

	for i := 0; i < 100; i++ {
		traceID := GenerateTraceID()
		assert.Len(t, traceID, 32, "итерация %d: trace_id должен быть длиной 32 символа", i)
		assert.True(t, hexPattern.MatchString(traceID), "итерация %d: trace_id должен быть валидным hex, got: %s", i, traceID)
	}
}

// TestFallbackTraceID_Format проверяет что fallback функция генерирует валидный 32-символьный hex.
// Это тестирует Task 3.4: fallback если crypto/rand недоступен.
//
// Примечание: Error path в GenerateTraceID (когда rand.Read возвращает ошибку) тестируется
// косвенно через эти тесты fallbackTraceID. Прямое тестирование error branch потребовало бы
// мок crypto/rand, что нецелесообразно для практически невозможного сценария.
func TestFallbackTraceID_Format(t *testing.T) {
	hexPattern := regexp.MustCompile("^[0-9a-f]{32}$")

	// Вызываем fallback напрямую (функция приватная, но доступна в том же пакете)
	traceID := fallbackTraceID()

	assert.Len(t, traceID, 32, "fallback trace_id должен быть длиной 32 символа")
	assert.True(t, hexPattern.MatchString(traceID), "fallback trace_id должен быть валидным hex, got: %s", traceID)
}

// TestFallbackTraceID_Unique проверяет что fallback генерирует уникальные ID.
func TestFallbackTraceID_Unique(t *testing.T) {
	id1 := fallbackTraceID()
	id2 := fallbackTraceID()

	assert.NotEqual(t, id1, id2, "два вызова fallbackTraceID должны возвращать разные значения")
}

// TestFallbackTraceID_UniqueInConcurrentCalls проверяет уникальность fallback при конкурентных вызовах.
func TestFallbackTraceID_UniqueInConcurrentCalls(t *testing.T) {
	const numGoroutines = 100
	ids := make(chan string, numGoroutines)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			ids <- fallbackTraceID()
		}()
	}

	wg.Wait()
	close(ids)

	// Собираем все ID в map для проверки уникальности
	seen := make(map[string]bool)
	for id := range ids {
		require.False(t, seen[id], "обнаружен дубликат fallback trace_id: %s", id)
		seen[id] = true
	}

	assert.Len(t, seen, numGoroutines, "должно быть сгенерировано %d уникальных fallback trace_id", numGoroutines)
}
