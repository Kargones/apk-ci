# Story 1.4: Logger interface + slog adapter

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a DevOps-инженер,
I want получать структурированные JSON логи,
so that я могу анализировать их автоматизированными инструментами.

## Acceptance Criteria

| # | Критерий | Тестируемость |
|---|----------|---------------|
| AC1 | Given logging.format=json в конфигурации, When происходит логирование, Then запись содержит: timestamp, level, msg, атрибуты в JSON формате | Unit test: SlogAdapter.Info() output validation |
| AC2 | Поддерживаются уровни логирования: DEBUG, INFO, WARN, ERROR | Unit test: все уровни методов логируют с правильным level |
| AC3 | Уровень логирования настраивается через конфигурацию (logging.level) | Unit test: уровень фильтрует сообщения ниже порога |
| AC4 | `Logger.With(args)` возвращает новый logger с добавленными атрибутами | Unit test: Logger.With() добавляет атрибуты ко всем последующим записям |
| AC5 | Logger пишет ТОЛЬКО в stderr (никогда в stdout) | Integration test: stdout пустой после логирования |
| AC6 | golangci-lint правило запрещает fmt.Print* в production коде (опционально) | Manual verification / future CI enhancement |

## Tasks / Subtasks

- [x] **Task 1: Создать Logger interface** (AC: 1-4)
  - [x] 1.1 Создать директорию `internal/pkg/logging/`
  - [x] 1.2 Создать `internal/pkg/logging/logger.go` с Logger interface
  - [x] 1.3 Определить методы: `Debug(msg string, args ...any)`, `Info(...)`, `Warn(...)`, `Error(...)`
  - [x] 1.4 Добавить метод `With(args ...any) Logger` для создания child logger с атрибутами
  - [x] 1.5 Добавить godoc комментарии на русском языке

- [x] **Task 2: Реализовать SlogAdapter** (AC: 1, 2, 5)
  - [x] 2.1 Создать `internal/pkg/logging/slog.go` с SlogAdapter struct
  - [x] 2.2 Обернуть *slog.Logger как поле структуры
  - [x] 2.3 Реализовать все методы Logger interface делегируя к slog
  - [x] 2.4 Метод With() должен возвращать новый SlogAdapter с child slog.Logger
  - [x] 2.5 Добавить godoc комментарии

- [x] **Task 3: Создать Factory для Logger** (AC: 1, 3, 5)
  - [x] 3.1 Создать `internal/pkg/logging/factory.go` с NewLogger(cfg Config) Logger
  - [x] 3.2 Поддержать format: "json" (slog.JSONHandler), "text" (slog.TextHandler)
  - [x] 3.3 Поддержать level: "debug", "info", "warn", "error" → slog.Level
  - [x] 3.4 Убедиться что handler использует os.Stderr (не os.Stdout!)
  - [x] 3.5 Default: format="text", level="info"

- [x] **Task 4: Создать Config struct** (AC: 3)
  - [x] 4.1 Создать `internal/pkg/logging/config.go` с Config struct
  - [x] 4.2 Поля: Format string, Level string
  - [x] 4.3 Добавить константы для форматов и уровней
  - [x] 4.4 Добавить godoc комментарии

- [x] **Task 5: Написать Unit Tests** (AC: 1-5)
  - [x] 5.1 Создать `internal/pkg/logging/slog_test.go`
  - [x] 5.2 TestSlogAdapter_Debug/Info/Warn/Error — все уровни работают
  - [x] 5.3 TestSlogAdapter_With — добавляет атрибуты
  - [x] 5.4 TestSlogAdapter_JSONFormat — JSON output validation
  - [x] 5.5 TestNewLogger_LevelFiltering — DEBUG не логируется при level=info
  - [x] 5.6 TestNewLogger_WritesToStderr — stdout пустой
  - [x] 5.7 Создать `internal/pkg/logging/factory_test.go`
  - [x] 5.8 TestNewLogger_DefaultValues — text format, info level

- [x] **Task 6: Документация и CI**
  - [x] 6.1 Добавить godoc комментарии к публичным типам и функциям
  - [x] 6.2 Проверить что golangci-lint проходит для logging пакета
  - [x] 6.3 Убедиться что все тесты проходят: `go test ./internal/pkg/logging/...`
  - [x] 6.4 Убедиться что race detector проходит: `go test -race ./internal/pkg/logging/...`

### Review Follow-ups (AI)

- [ ] [AI-Review][HIGH] NewSlogAdapter не проверяет nil logger — nil *slog.Logger вызовет panic [slog.go:13-15]
- [ ] [AI-Review][MEDIUM] NewLogger жёстко связан с lumberjack зависимостью — тяжёлая зависимость для stdlib-only логирования [factory.go:25-41]
- [ ] [AI-Review][MEDIUM] NopLogger.With() возвращает тот же объект — нарушает контракт Logger [nop.go:29]
- [ ] [AI-Review][MEDIUM] DefaultFilePath = "/var/log/apk-ci.log" — hardcoded Linux path, невалидный на Windows [config.go:29]
- [ ] [AI-Review][LOW] TestNewLogger_WritesToStderr модифицирует os.Stdout/os.Stderr — может конфликтовать при параллельном запуске [factory_test.go:72-111]

## Dev Notes

### Критический контекст для реализации

**Архитектурное решение из ADR-005 (slog для Structured Logging):**
- Используем slog из stdlib Go 1.21+ — zero dependencies
- Logger interface абстрагирует slog для:
  1. Тестирования (можно мокировать)
  2. Возможной замены реализации в будущем
  3. Упрощения API (не нужно знать slog internals)

**КРИТИЧНО: Разделение потоков вывода (Pre-mortem FM4):**
- **stderr**: ТОЛЬКО логи через Logger
- **stdout**: ТОЛЬКО результаты команд через OutputWriter (Story 1.3)
- **НИКОГДА** не использовать fmt.Print* в production коде
- Это предотвращает поломку downstream JSON парсеров: `apk-ci | jq`

**Интеграция с существующим кодом:**
- В main.go уже используется `cfg.Logger` типа `*slog.Logger`
- После этой Story: заменить на наш Logger interface для тестируемости
- Wire DI (Story 1.7) будет создавать Logger через NewLogger()

### Data Structures из Tech Spec

**Logger Interface (internal/pkg/logging/logger.go):**
```go
// Package logging предоставляет интерфейс и реализации для структурированного логирования.
package logging

// Logger определяет интерфейс для структурированного логирования.
// Реализации: SlogAdapter (использует slog из stdlib).
//
// Все методы принимают сообщение и опциональные key-value пары:
//
//     logger.Info("Команда выполнена", "command", cmd, "duration_ms", 150)
//
// ВАЖНО: Logger пишет ТОЛЬКО в stderr, никогда в stdout.
// Это критично для корректной работы с OutputWriter.
type Logger interface {
    // Debug записывает сообщение уровня DEBUG.
    // Используется для детальной диагностики.
    Debug(msg string, args ...any)

    // Info записывает сообщение уровня INFO.
    // Используется для значимых событий (старт/стоп, успешные операции).
    Info(msg string, args ...any)

    // Warn записывает сообщение уровня WARN.
    // Используется для recoverable issues, deprecated usage.
    Warn(msg string, args ...any)

    // Error записывает сообщение уровня ERROR.
    // Используется для ошибок требующих внимания.
    Error(msg string, args ...any)

    // With возвращает новый Logger с добавленными атрибутами.
    // Атрибуты будут включены во все последующие записи.
    //
    //     logger.With("trace_id", traceID).Info("Операция началась")
    With(args ...any) Logger
}
```

**SlogAdapter (internal/pkg/logging/slog.go):**
```go
package logging

import "log/slog"

// SlogAdapter реализует Logger interface используя slog из stdlib.
// Это основная production реализация логгера.
type SlogAdapter struct {
    logger *slog.Logger
}

// NewSlogAdapter создаёт новый SlogAdapter с указанным slog.Logger.
// Для создания с конфигурацией используйте NewLogger().
func NewSlogAdapter(logger *slog.Logger) *SlogAdapter {
    return &SlogAdapter{logger: logger}
}

// Debug записывает сообщение уровня DEBUG.
func (s *SlogAdapter) Debug(msg string, args ...any) {
    s.logger.Debug(msg, args...)
}

// Info записывает сообщение уровня INFO.
func (s *SlogAdapter) Info(msg string, args ...any) {
    s.logger.Info(msg, args...)
}

// Warn записывает сообщение уровня WARN.
func (s *SlogAdapter) Warn(msg string, args ...any) {
    s.logger.Warn(msg, args...)
}

// Error записывает сообщение уровня ERROR.
func (s *SlogAdapter) Error(msg string, args ...any) {
    s.logger.Error(msg, args...)
}

// With возвращает новый Logger с добавленными атрибутами.
func (s *SlogAdapter) With(args ...any) Logger {
    return &SlogAdapter{logger: s.logger.With(args...)}
}
```

**LoggingConfig (internal/pkg/logging/config.go):**
```go
package logging

// Поддерживаемые форматы вывода логов.
const (
    FormatJSON = "json"
    FormatText = "text"
)

// Поддерживаемые уровни логирования.
const (
    LevelDebug = "debug"
    LevelInfo  = "info"
    LevelWarn  = "warn"
    LevelError = "error"
)

// LoggingConfig содержит настройки логирования.
type LoggingConfig struct {
    // Format определяет формат вывода: "json" или "text".
    // По умолчанию: "text".
    Format string

    // Level определяет минимальный уровень логирования.
    // По умолчанию: "info".
    // Допустимые значения: "debug", "info", "warn", "error".
    Level string
}
```

**Factory (internal/pkg/logging/factory.go):**
```go
package logging

import (
    "log/slog"
    "os"
)

// NewLogger создаёт Logger с заданной конфигурацией.
// Возвращает SlogAdapter настроенный согласно config.
//
// ВАЖНО: Logger всегда пишет в os.Stderr.
func NewLogger(config LoggingConfig) Logger {
    // Определяем уровень
    level := parseLevel(config.Level)

    // Создаём handler
    opts := &slog.HandlerOptions{Level: level}
    var handler slog.Handler

    switch config.Format {
    case FormatJSON:
        handler = slog.NewJSONHandler(os.Stderr, opts)
    default:
        handler = slog.NewTextHandler(os.Stderr, opts)
    }

    return NewSlogAdapter(slog.New(handler))
}

// parseLevel конвертирует строковый уровень в slog.Level.
// При неизвестном значении возвращает slog.LevelInfo.
func parseLevel(level string) slog.Level {
    switch level {
    case LevelDebug:
        return slog.LevelDebug
    case LevelWarn:
        return slog.LevelWarn
    case LevelError:
        return slog.LevelError
    default:
        return slog.LevelInfo
    }
}
```

### Зависимости

| Зависимость | Статус | Влияние |
|-------------|--------|---------|
| Story 1.1 (Command Registry) | done | Registry использует cfg.Logger — можно будет заменить |
| Story 1.2 (DeprecatedBridge) | done | Warning сейчас через fmt.Fprintf — можно будет переписать через Logger |
| Story 1.3 (OutputWriter) | done | OutputWriter → stdout, Logger → stderr — КРИТИЧНО не смешивать |
| Story 1.5 (Trace ID) | pending | TraceID будет добавляться через Logger.With() |
| Story 1.6 (Config extensions) | pending | LoggingConfig будет частью AppConfig |
| Story 1.7 (Wire DI) | pending | Logger будет Wire provider |

### Риски и митигации

| ID | Риск | Probability | Impact | Митигация |
|----|------|-------------|--------|-----------|
| R1 | Логи попадают в stdout вместо stderr | High | High | AC5: Тест что stdout пустой после логирования |
| R2 | slog API меняется | Low | Medium | Interface абстракция — можем заменить реализацию |
| R3 | Перфоманс overhead от обёртки | Low | Low | Прямой вызов slog методов, нет reflection |
| R4 | Забыли добавить trace_id | Medium | Low | Story 1.5 добавит через With() |
| R5 | fmt.Print* в production коде | Medium | High | AC6: golangci-lint правило (опционально) |

### Pre-mortem Failure Modes из Tech Spec

| FM | Failure Mode | AC Coverage |
|----|--------------|-------------|
| FM4 | Логи в stdout ломают JSON парсеры | AC5: Logger → stderr only |

### Связь с предыдущими Stories

**Что переиспользуем из Story 1.1-1.3:**
- Стиль godoc комментариев на русском
- Паттерн interface + implementation в отдельных файлах
- testify/assert + testify/require
- Naming conventions: PascalCase для экспортируемых типов
- Factory pattern для создания реализаций

**Что готовим для следующих Stories:**
- Story 1.5 (Trace ID): Logger.With("trace_id", id) — добавление trace_id к логам
- Story 1.6 (Config): LoggingConfig станет частью AppConfig
- Story 1.7 (Wire DI): Logger будет одним из Wire providers
- Story 1.8 (nr-version): Первая NR-команда будет использовать Logger

### Git Intelligence (последние коммиты)

```
f6f3425 feat(output): add JSON schema validation and improve text output
be8c663 feat(output): add structured output writer with JSON and text formats
1339d03 fix(command): check context cancellation before warning in deprecated bridge
698dd95 feat(command): add deprecated command support with migration bridge
dfb42c2 feat(command): add kebab-case validation and debug functions to registry
```

**Паттерны из предыдущих коммитов:**
- Файлы в internal/pkg/ следуют структуре: `logger.go` (interface), `slog.go` (implementation), `factory.go`, `config.go`
- Golden tests для валидации форматов (можно использовать для JSON логов)
- Константы для допустимых значений (FormatJSON, FormatText)

### Project Structure Notes

**Создаваемые директории и файлы:**
```
internal/pkg/
└── logging/
    ├── logger.go           # Logger interface
    ├── slog.go             # SlogAdapter implementation
    ├── slog_test.go        # Unit tests для SlogAdapter
    ├── factory.go          # NewLogger factory
    ├── factory_test.go     # Unit tests для factory
    └── config.go           # LoggingConfig struct
```

**Не изменять (пока):**
- `cmd/apk-ci/main.go` — интеграция будет в Story 1.7 (Wire DI)
- `internal/command/deprecated.go` — переход на Logger в Story 1.7
- `internal/config/config.go` — LoggingConfig будет добавлена в Story 1.6

### Testing Standards

- Framework: testify/assert, testify/require
- Pattern: Table-driven tests где применимо
- Naming: `Test{TypeName}_{Method}_{Scenario}`
- Location: `*_test.go` рядом с тестируемым файлом
- Run: `go test ./internal/pkg/logging/... -v`
- Race: `go test ./internal/pkg/logging/... -race`

### Обязательные тесты

| Тест | Описание | AC |
|------|----------|-----|
| TestSlogAdapter_Debug | Debug() логирует с level=DEBUG | AC2 |
| TestSlogAdapter_Info | Info() логирует с level=INFO | AC2 |
| TestSlogAdapter_Warn | Warn() логирует с level=WARN | AC2 |
| TestSlogAdapter_Error | Error() логирует с level=ERROR | AC2 |
| TestSlogAdapter_With | With() возвращает Logger с добавленными атрибутами | AC4 |
| TestSlogAdapter_JSONFormat | JSON handler выводит валидный JSON | AC1 |
| TestNewLogger_DefaultValues | По умолчанию text format, info level | AC3 |
| TestNewLogger_LevelFiltering | DEBUG не логируется при level=info | AC3 |
| TestNewLogger_WritesToStderr | Логи пишутся в stderr, stdout пустой | AC5 |
| TestParseLevel_AllLevels | Все уровни корректно парсятся | AC3 |

### References

- [Source: _bmad-output/project-planning-artifacts/architecture.md#ADR-005: slog для Structured Logging]
- [Source: _bmad-output/project-planning-artifacts/architecture.md#Logger Interface]
- [Source: _bmad-output/project-planning-artifacts/epics/epic-1-foundation.md#Story 1.4]
- [Source: _bmad-output/project-planning-artifacts/prd.md#FR32-35] — Functional Requirements для логирования
- [Source: internal/pkg/output/] — Паттерн структуры пакета
- [Source: internal/pkg/apperrors/] — Паттерн error handling

### Review Follow-ups (AI Code Review #34)

- [x] [AI-Review][HIGH] ~~NewSlogAdapter не проверяет nil logger~~ — ИСПРАВЛЕНО Review #34: добавлен panic для nil [slog.go:15-17]
- [ ] [AI-Review][MEDIUM] TestNewLogger_WritesToStderr модифицирует глобальные переменные без t.Cleanup [logging/factory.go]
- [ ] [AI-Review][MEDIUM] NewLogger жёстко связан с lumberjack для файлового логирования [logging/factory.go]

## Dev Agent Record

### Agent Model Used

Claude Opus 4.5 (claude-opus-4-5-20251101)

### Debug Log References

### Completion Notes List

- Реализован Logger interface с методами Debug, Info, Warn, Error и With
- Создан SlogAdapter как основная production реализация, оборачивающая slog из stdlib
- Factory NewLogger() создаёт logger с конфигурируемым форматом (json/text) и уровнем
- Config struct переименован из LoggingConfig во избежание stuttering (logging.Config вместо logging.LoggingConfig)
- Все логи пишутся ТОЛЬКО в stderr, что критично для корректной работы с OutputWriter (stdout)
- Написано 20+ тестов покрывающих все AC, включая:
  - Все уровни логирования (DEBUG, INFO, WARN, ERROR)
  - JSON формат с валидацией структуры
  - Фильтрация по уровню
  - With() для добавления атрибутов
  - Проверка что stdout пустой после логирования
- golangci-lint пройден для logging пакета (0 issues)
- Race detector пройден

### Change Log

- 2026-01-26: Story создана через create-story workflow (BMAD v6.0)
- 2026-01-26: Реализация завершена — Logger interface, SlogAdapter, Factory, Config, тесты
- 2026-01-26: Code Review (AI) — исправлены 3 MEDIUM issues:
  - M1: Добавлен NewLoggerWithWriter для тестирования с custom writer
  - M2: Добавлен NopLogger для unit-тестов других пакетов
  - M3: Рефакторинг тестов — использование NewLoggerWithWriter вместо os.Pipe

### File List

- internal/pkg/logging/logger.go (new)
- internal/pkg/logging/slog.go (new)
- internal/pkg/logging/slog_test.go (new)
- internal/pkg/logging/factory.go (new, updated: added NewLoggerWithWriter)
- internal/pkg/logging/factory_test.go (new, updated: refactored to use NewLoggerWithWriter)
- internal/pkg/logging/config.go (new)
- internal/pkg/logging/nop.go (new: NopLogger for testing)
- internal/pkg/logging/nop_test.go (new: NopLogger tests)
