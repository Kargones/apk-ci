# Story 1.5: Trace ID generation

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a DevOps-инженер,
I want иметь trace_id в каждой записи лога,
so that я могу коррелировать логи одной операции.

## Acceptance Criteria

| # | Критерий | Тестируемость |
|---|----------|---------------|
| AC1 | Given команда начинает выполнение, When генерируется trace_id, Then trace_id имеет формат UUID v4 или 16-byte hex string | Unit test: ValidateTraceIDFormat |
| AC2 | Given trace_id сгенерирован, When добавляется в context, Then trace_id доступен через TraceIDFromContext(ctx) | Unit test: WithTraceID/TraceIDFromContext roundtrip |
| AC3 | Given context содержит trace_id, When происходит логирование, Then все записи лога содержат trace_id | Integration test: Logger.With("trace_id", TraceIDFromContext(ctx)) |
| AC4 | Given команда завершается, When результат сериализуется, Then trace_id включается в JSON output metadata | Integration test: Result.Metadata.TraceID заполнен |

## Tasks / Subtasks

- [x] **Task 1: Создать пакет tracing** (AC: 1)
  - [x] 1.1 Создать директорию `internal/pkg/tracing/`
  - [x] 1.2 Создать `internal/pkg/tracing/traceid.go` с функцией GenerateTraceID() string
  - [x] 1.3 Использовать crypto/rand для генерации 16 байт
  - [x] 1.4 Форматировать как hex string (32 символа)
  - [x] 1.5 Добавить godoc комментарии на русском языке

- [x] **Task 2: Реализовать context integration** (AC: 2)
  - [x] 2.1 Создать `internal/pkg/tracing/context.go`
  - [x] 2.2 Определить private key: `type traceIDKey struct{}`
  - [x] 2.3 Создать `WithTraceID(ctx context.Context, id string) context.Context`
  - [x] 2.4 Создать `TraceIDFromContext(ctx context.Context) string` (возвращает "" если нет)
  - [x] 2.5 Добавить godoc комментарии

- [x] **Task 3: Написать Unit Tests** (AC: 1, 2)
  - [x] 3.1 Создать `internal/pkg/tracing/traceid_test.go`
  - [x] 3.2 TestGenerateTraceID_Format — валидный hex формат, 32 символа
  - [x] 3.3 TestGenerateTraceID_Unique — два вызова возвращают разные ID
  - [x] 3.4 TestFallbackTraceID_Format/Unique/Concurrent — тесты fallback функции
  - [x] 3.5 Создать `internal/pkg/tracing/context_test.go`
  - [x] 3.6 TestWithTraceID_AddsToContext — trace_id извлекается
  - [x] 3.7 TestTraceIDFromContext_EmptyContext — возвращает пустую строку
  - [x] 3.8 TestTraceIDFromContext_WrongKey — возвращает пустую строку

- [x] **Task 4: Integration с Logger** (AC: 3, 4-partial)
  - [x] 4.1 Создать `internal/pkg/tracing/integration_test.go`
  - [x] 4.2 Тест: Logger.With("trace_id", TraceIDFromContext(ctx)).Info() содержит trace_id
  - [x] 4.3 Документировать паттерн использования в godoc
  - [ ] 4.4 AC4: Заполнение Result.Metadata.TraceID — отложено до Story 1.7 (Wire DI)

- [x] **Task 5: Документация и CI**
  - [x] 5.1 Добавить godoc комментарии к публичным типам и функциям
  - [x] 5.2 Проверить что golangci-lint проходит: `make lint`
  - [x] 5.3 Убедиться что все тесты проходят: `go test ./internal/pkg/tracing/... -v`
  - [x] 5.4 Убедиться что race detector проходит: `go test -race ./internal/pkg/tracing/...`

### Review Follow-ups (AI)

- [ ] [AI-Review][HIGH] fallbackTraceID: uint64 cast отрицательного UnixNano — формат %016x может сработать, но неочевидно [traceid.go:56-62]
- [ ] [AI-Review][MEDIUM] fallbackCounter — пакетная глобальная переменная, не сбрасывается между тестами [traceid.go:26]
- [ ] [AI-Review][MEDIUM] WithTraceID не валидирует формат id — можно передать пустую строку или произвольный текст [context.go:17-18]
- [ ] [AI-Review][LOW] GenerateTraceID не принимает context — не соответствует Go идиомам [traceid.go:40-46]

## Dev Notes

### Критический контекст для реализации

**Архитектурное решение из Tech Spec (AC6):**
- trace_id в формате UUID v4 или 16-byte hex string
- Использовать crypto/rand для генерации (безопасный random)
- trace_id добавляется в context на старте команды
- Все последующие логи получают trace_id через Logger.With()
- trace_id включается в Result.Metadata для JSON output

**Интеграция с Logger (Story 1.4):**
```go
// Паттерн использования в main.go (будет интегрировано в Story 1.7):
traceID := tracing.GenerateTraceID()
ctx := tracing.WithTraceID(ctx, traceID)
logger := logger.With("trace_id", tracing.TraceIDFromContext(ctx))
```

**Интеграция с OutputWriter (Story 1.3):**
```go
// Result.Metadata.TraceID заполняется перед выводом:
result.Metadata = &output.Metadata{
    TraceID:    tracing.TraceIDFromContext(ctx),
    DurationMs: elapsed.Milliseconds(),
    APIVersion: "v1",
}
```

**OpenTelemetry compatibility (подготовка к Epic 6):**
- 16-byte hex (32 символа) совместим с W3C Trace Context format
- В Epic 6 можно будет экспортировать trace_id в OpenTelemetry

### Data Structures из Tech Spec

**TraceID генерация (internal/pkg/tracing/traceid.go):**
```go
// Package tracing предоставляет функции для генерации и управления trace ID.
// Trace ID используется для корреляции логов одной операции.
package tracing

import (
    "crypto/rand"
    "encoding/hex"
)

// GenerateTraceID генерирует уникальный trace ID.
// Формат: 32 символа hex (16 байт), например: "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6".
//
// Использует crypto/rand для криптографически безопасной генерации.
// При ошибке crypto/rand возвращает fallback значение с timestamp.
func GenerateTraceID() string {
    b := make([]byte, 16)
    if _, err := rand.Read(b); err != nil {
        // Fallback на timestamp-based ID (не должно происходить в production)
        return fallbackTraceID()
    }
    return hex.EncodeToString(b)
}

// fallbackTraceID генерирует ID на основе текущего времени.
// Используется только если crypto/rand недоступен.
func fallbackTraceID() string {
    // Implementation: timestamp + counter
}
```

**Context integration (internal/pkg/tracing/context.go):**
```go
package tracing

import "context"

// traceIDKey — ключ для хранения trace ID в context.
// Приватный тип предотвращает коллизии ключей.
type traceIDKey struct{}

// WithTraceID возвращает новый context с добавленным trace ID.
//
// Пример использования:
//
//     traceID := tracing.GenerateTraceID()
//     ctx = tracing.WithTraceID(ctx, traceID)
func WithTraceID(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, traceIDKey{}, id)
}

// TraceIDFromContext извлекает trace ID из context.
// Возвращает пустую строку если trace ID не установлен.
//
// Пример использования:
//
//     logger.With("trace_id", tracing.TraceIDFromContext(ctx)).Info("message")
func TraceIDFromContext(ctx context.Context) string {
    if id, ok := ctx.Value(traceIDKey{}).(string); ok {
        return id
    }
    return ""
}
```

### Зависимости

| Зависимость | Статус | Влияние |
|-------------|--------|---------|
| Story 1.1 (Command Registry) | done | Registry не использует trace_id напрямую |
| Story 1.2 (DeprecatedBridge) | done | Warning может включать trace_id через Logger.With() |
| Story 1.3 (OutputWriter) | done | Result.Metadata.TraceID будет заполняться |
| Story 1.4 (Logger interface) | done | Logger.With("trace_id", id) — основной паттерн |
| Story 1.6 (Config extensions) | pending | Нет прямой зависимости |
| Story 1.7 (Wire DI) | pending | Trace ID генерация будет в main.go до Wire init |
| Story 1.8 (nr-version) | pending | Первая команда с trace_id в логах |

### Риски и митигации

| ID | Риск | Probability | Impact | Митигация |
|----|------|-------------|--------|-----------|
| R1 | crypto/rand недоступен | Very Low | Low | Fallback на timestamp-based ID |
| R2 | Context key collision | Very Low | Medium | Private type для key |
| R3 | Забыли добавить trace_id в логи | Medium | Medium | Документация + example в godoc |
| R4 | Trace ID не в OpenTelemetry формате | Low | Low | 16-byte hex совместим с W3C |

### Pre-mortem Failure Modes из Tech Spec

| FM | Failure Mode | AC Coverage |
|----|--------------|-------------|
| — | Нет специфичных FM для Story 1.5 | AC1-AC4 покрывают основные сценарии |

### Связь с предыдущими Stories

**Что переиспользуем из Story 1.3-1.4:**
- Стиль godoc комментариев на русском
- Паттерн interface + implementation в отдельных файлах
- testify/assert + testify/require
- Naming conventions: PascalCase для экспортируемых типов
- Структура пакета: `*.go` + `*_test.go`

**Что готовим для следующих Stories:**
- Story 1.6 (Config): Нет прямой связи
- Story 1.7 (Wire DI): trace_id генерируется в main.go до Wire init
- Story 1.8 (nr-version): Первая NR-команда будет иметь trace_id в логах и output

### Git Intelligence (последние коммиты)

```
ecd4f8d feat(logging): implement structured logging interface with slog adapter
f6f3425 feat(output): add JSON schema validation and improve text output
be8c663 feat(output): add structured output writer with JSON and text formats
1339d03 fix(command): check context cancellation before warning in deprecated bridge
698dd95 feat(command): add deprecated command support with migration bridge
```

**Паттерны из предыдущих коммитов:**
- Файлы в internal/pkg/ следуют структуре: `*.go` (code), `*_test.go` (tests)
- Константы для допустимых значений
- godoc с примерами использования
- Private types для context keys

### Project Structure Notes

**Создаваемые директории и файлы:**
```
internal/pkg/
└── tracing/
    ├── traceid.go           # GenerateTraceID function
    ├── traceid_test.go      # Unit tests для генерации
    ├── context.go           # WithTraceID, TraceIDFromContext
    ├── context_test.go      # Unit tests для context
    └── integration_test.go  # Integration tests с Logger
```

**Не изменять (пока):**
- `cmd/apk-ci/main.go` — интеграция будет в Story 1.7 (Wire DI)
- `internal/pkg/logging/` — используем как есть через Logger.With()
- `internal/pkg/output/result.go` — Metadata.TraceID уже определён

**Alignment с архитектурой:**
- Пакет `internal/pkg/tracing/` соответствует project structure из architecture.md
- OpenTelemetry setup (`otel.go`) отложен до Epic 6

### Testing Standards

- Framework: testify/assert, testify/require
- Pattern: Table-driven tests где применимо
- Naming: `Test{FunctionName}_{Scenario}`
- Location: `*_test.go` рядом с тестируемым файлом
- Run: `go test ./internal/pkg/tracing/... -v`
- Race: `go test ./internal/pkg/tracing/... -race`

### Обязательные тесты

| Тест | Описание | AC |
|------|----------|-----|
| TestGenerateTraceID_Format | Возвращает 32-символьный hex string | AC1 |
| TestGenerateTraceID_ValidHex | Все символы [0-9a-f] | AC1 |
| TestGenerateTraceID_Unique | Два вызова возвращают разные значения | AC1 |
| TestWithTraceID_AddsToContext | trace_id добавляется в context | AC2 |
| TestTraceIDFromContext_Success | trace_id извлекается из context | AC2 |
| TestTraceIDFromContext_EmptyContext | Возвращает "" для пустого context | AC2 |
| TestTraceIDFromContext_NilContext | Не паникует для nil context | AC2 |
| TestLogger_WithTraceID | Logger.With() включает trace_id в записи | AC3 |

### References

- [Source: _bmad-output/project-planning-artifacts/architecture.md#Tracing] — Tracing architecture
- [Source: _bmad-output/project-planning-artifacts/architecture.md#Context Keys] — Context key pattern
- [Source: _bmad-output/implementation-artifacts/sprint-artifacts/tech-spec-epic-1.md#AC6] — Trace ID AC
- [Source: _bmad-output/implementation-artifacts/sprint-artifacts/tech-spec-epic-1.md#APIs and Interfaces] — Context Keys definition
- [Source: _bmad-output/project-planning-artifacts/epics/epic-1-foundation.md#Story 1.5] — Epic description
- [Source: _bmad-output/project-planning-artifacts/prd.md#FR35] — Functional Requirement: trace_id в логах
- [Source: _bmad-output/project-planning-artifacts/prd.md#FR42] — Functional Requirement: trace_id в output
- [Source: internal/pkg/logging/] — Logger interface для интеграции
- [Source: internal/pkg/output/result.go] — Metadata.TraceID field

### Review Follow-ups (AI Code Review #34)

- [ ] [AI-Review][MEDIUM] fallbackCounter — глобальная переменная atomic.Uint64, не сбрасывается между тестами [tracing/traceid.go:26]
- [ ] [AI-Review][MEDIUM] WithTraceID не валидирует формат id (пустая строка/произвольный текст) [tracing/context.go]

## Dev Agent Record

### Agent Model Used

Claude Opus 4.5 (claude-opus-4-5-20251101)

### Debug Log References

Нет проблем во время реализации.

### Completion Notes List

**Дата: 2026-01-26**

1. **Task 1: Создать пакет tracing** ✅
   - Создан пакет `internal/pkg/tracing/` с функцией `GenerateTraceID()`
   - Использует `crypto/rand` для криптографически безопасной генерации 16 байт
   - Форматирует как lowercase hex string (32 символа)
   - Реализован fallback на timestamp+counter если crypto/rand недоступен
   - Все godoc комментарии на русском языке

2. **Task 2: Реализовать context integration** ✅
   - Создан `context.go` с функциями `WithTraceID()` и `TraceIDFromContext()`
   - Использует приватный тип `traceIDKey struct{}` для предотвращения коллизий
   - `TraceIDFromContext()` безопасно обрабатывает nil context

3. **Task 3: Написать Unit Tests** ✅
   - `traceid_test.go`: 8 тестов для генерации trace ID (включая 3 теста fallback)
   - `context_test.go`: 9 тестов для context integration
   - Все тесты используют testify/assert и testify/require
   - Покрытие включает конкурентные вызовы, edge cases и fallback функцию

4. **Task 4: Integration с Logger** ✅
   - `integration_test.go`: 5 интеграционных тестов с Logger
   - Проверена совместимость с W3C Trace Context (OpenTelemetry ready)
   - Документирован паттерн использования в godoc

5. **Task 5: Документация и CI** ✅
   - golangci-lint: 0 issues
   - go test: все 26 тестов прошли (включая 4 Example функции)
   - race detector: прошёл успешно

**Acceptance Criteria Verification:**
- AC1 ✅ trace_id в формате 32-символьный hex (16 байт)
- AC2 ✅ WithTraceID/TraceIDFromContext roundtrip работает
- AC3 ✅ Logger.With("trace_id", ...) включает trace_id в записи
- AC4 ⚠️ **PARTIAL** — Metadata.TraceID поле определено в output/result.go, но заполнение будет в Story 1.7 (Wire DI)

### Senior Developer Review (AI)

**Дата:** 2026-01-26
**Reviewer:** Claude Opus 4.5 (Adversarial Code Review)

**Outcome:** ✅ APPROVED with fixes applied

**Findings Summary:**
- 0 CRITICAL issues
- 4 MEDIUM issues (all fixed)
- 3 LOW issues (documented)

**MEDIUM Issues Fixed:**
1. ✅ `example_test.go` добавлен в git staging (был untracked)
2. ✅ Task 4.4 добавлен для явного указания что AC4 отложен до Story 1.7
3. ✅ Добавлен комментарий в `traceid_test.go` о косвенном тестировании error path
4. ✅ sprint-status.yaml синхронизирован со статусом "review"

**LOW Issues (не исправлялись, документированы):**
- LOW-1: Godoc пример может ввести в заблуждение (minor documentation)
- LOW-2: Нет benchmark тестов (nice-to-have для future optimization)
- LOW-3: Нет формальной секции review - исправлено добавлением этой секции

**Verification:**
- ✅ Все 26 тестов проходят
- ✅ Race detector: PASS
- ✅ golangci-lint: 0 issues
- ✅ AC1-AC3: Полностью реализованы
- ⚠️ AC4: Частично (поле готово, заполнение в Story 1.7)

### Change Log

| Дата | Изменение |
|------|-----------|
| 2026-01-26 | Создан пакет tracing с GenerateTraceID(), WithTraceID(), TraceIDFromContext() |
| 2026-01-26 | Добавлены unit и integration тесты (19 тестов) |
| 2026-01-26 | Все проверки lint и race detector пройдены |
| 2026-01-26 | **Code Review**: Добавлены 3 теста для fallbackTraceID() (теперь 22 теста), обновлён комментарий в result.go |
| 2026-01-26 | **Code Review #2**: Добавлены Example функции для godoc (теперь 26 тестов), уточнён статус AC4 как PARTIAL |
| 2026-01-26 | **Senior Dev Review**: 4 MEDIUM fixes applied, review approved |
| 2026-01-26 | **Status → done**: Story complete, AC4 tracked in Task 4.4 for Story 1.7 |

### File List

**Новые файлы:**
- `internal/pkg/tracing/traceid.go` - функция GenerateTraceID()
- `internal/pkg/tracing/traceid_test.go` - unit тесты для traceid (8 тестов, включая fallback)
- `internal/pkg/tracing/context.go` - WithTraceID() и TraceIDFromContext()
- `internal/pkg/tracing/context_test.go` - unit тесты для context (9 тестов)
- `internal/pkg/tracing/integration_test.go` - integration тесты с Logger (5 тестов)
- `internal/pkg/tracing/example_test.go` - Example функции для godoc (4 примера)

**Изменённые файлы:**
- `_bmad-output/implementation-artifacts/sprint-artifacts/sprint-status.yaml` - обновлён статус story
- `internal/pkg/output/result.go` - обновлён комментарий к TraceID (code review fix)

