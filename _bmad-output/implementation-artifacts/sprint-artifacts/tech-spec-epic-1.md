# Epic Technical Specification: Architectural Foundation

Date: 2025-11-26
Author: BMad
Epic ID: 1
Status: Draft

---

## Overview

Epic 1 "Architectural Foundation" создаёт SOLID-совместимую архитектурную основу для benadis-runner v2.0. Это фундамент для всей трансформации — от механического объединения утилит к расширяемой платформе с plug-in архитектурой. Epic включает реализацию Command Registry с self-registration (OCP), NR-Migration Bridge для безопасной миграции, OutputWriter для структурированного вывода (text/JSON), Logger interface с slog адаптером, генерацию trace_id, расширение конфигурации, настройку Wire DI и создание первой NR-команды (nr-version) как proof of concept.

После завершения Epic 1 новые команды добавляются без изменения main.go, все зависимости внедряются через Wire, результаты команд выводятся в структурированном формате, а deprecated-команды логируются с рекомендацией миграции. Это создаёт основу для параллельного существования старых и новых команд во время миграции (FR47-FR50).

**Архитектурный выбор обоснован first principles:**
- Каждое решение проверено на соответствие реальным потребностям (не "best practice ради best practice")
- Wire выбран после анализа альтернатив (manual DI, dig) — compile-time проверки критичны
- slog выбран как stdlib решение без внешних зависимостей

**Strategic Focus (из SWOT):**
- Максимизировать compile-time safety (Wire, строгая типизация)
- trace_id как инвестиция в observability (Epic 6)
- Adapter pattern для изоляции от внешних изменений (1C Platform, APIs)

**Stakeholder Engagement (из Stakeholder Mapping):**
- DevOps (ключевые): демо nr-version после Story 1.8, канал для feedback
- Техлид (ключевой): ADR review до кодирования, weekly architecture sync
- 1C-разработчики: коммуникация "существующие команды работают без изменений"
- Аналитики/Тестировщики: roadmap preview (Epic 5 и Epic 3 соответственно)

## Objectives and Scope

**In Scope (MVP):**
- Command Registry с паттерном self-registration через init() (FR1, FR3, FR47)
- DeprecatedBridge для логирования использования deprecated-команд (FR48, FR50)
- OutputWriter interface с Text и JSON форматтерами (FR29-FR31)
- Structured AppError с кодами ошибок и wrapping (FR29)
- Logger interface с slog адаптером, JSON формат логов (FR33-FR34)
- Trace ID генерация и добавление в context (FR35, FR42)
- Расширение конфигурации: секции implementations и logging (FR2, FR28)
- Wire DI setup с providers для Config, Logger, OutputWriter (FR4-FR5)
- Первая NR-команда nr-version через Registry (FR47)
- Auto-generated help из Registry (FR3)

**Out of Scope (отложено на следующие эпики):**
- YAML formatter (Epic 7, Story 7.10)
- Log file rotation (Epic 6, Story 6.1)
- OpenTelemetry export (Epic 6, Story 6.7)
- Реальные команды (service-mode-*, dbrestore и др.) — Epic 2-5
- Алертинг (Epic 6)
- Prometheus метрики (Epic 6)

**Pre-mortem Mitigations (включены в AC):**
- CI check: новые handlers ДОЛЖНЫ использовать command.Register()
- Wire providers работают без изменения legacy-кода в internal/app/
- Тест backward compatibility для существующих production-конфигов
- Тест "чистоты" stdout — только Result JSON
- DoD: nr-version интегрирован минимум в 1 production pipeline

**Devil's Advocate Mitigations:**
- Wire: README в internal/di/ с примерами добавления providers
- Registry: функция ListAll() для отладки и тестирования
- Fallback: временный механизм, удаляется в Epic 7 вместе с legacy switch
- Trace ID: подготовка инфраструктуры для OpenTelemetry (Epic 6)

**Scope Protection (из SWOT Threats):**
- Строгое следование Out of Scope — любые additions требуют отдельного discussion
- nr-version ДОЛЖЕН быть в production до начала Epic 2 (proof of value)
- Shadow-run для любой NR-команды перед production (начиная с Epic 2)

## Stakeholder Communication Plan

| Milestone | Стейкхолдер | Действие |
|-----------|-------------|----------|
| До начала Epic 1 | Техлид | Review ADR-001..005 |
| После Story 1.7 | Техлид | Wire integration review |
| После Story 1.8 | DevOps | Демо nr-version, сбор feedback |
| После Story 1.8 | 1C-разработчики | Коммуникация: backward compatibility |
| Production deploy | DevOps | Интеграция nr-version в 1 pipeline |

## System Architecture Alignment

Epic 1 реализует архитектурные решения из ADR-001 (Wire DI), ADR-002 (Command Registry), ADR-003 (Role-based interfaces) и ADR-005 (slog logging).

**Ключевые компоненты:**
- `internal/command/registry.go` — Command Registry (ADR-002)
- `internal/command/deprecated.go` — DeprecatedBridge (NR-Migration)
- `internal/pkg/output/` — OutputWriter interface + Text/JSON форматтеры
- `internal/pkg/errors/errors.go` — Structured AppError
- `internal/pkg/logging/` — Logger interface + slog адаптер (ADR-005)
- `internal/pkg/tracing/traceid.go` — Trace ID генерация
- `internal/config/config.go` — Расширенная конфигурация
- `internal/di/` — Wire providers и injectors (ADR-001)
- `internal/command/handlers/version/version.go` — nr-version handler

**Архитектурные ограничения:**
- Thread-safe не требуется для Registry (регистрация только в init)
- Logger пишет ТОЛЬКО в stderr (stdout только для Result)
- Все новые поля конфигурации optional с defaults (backward compatibility)
- Wire compile-time DI, ошибки на этапе компиляции

**Критические точки интеграции (из Pre-mortem):**
- main.go: fallback на legacy switch если Registry не находит команду
- Config: все новые поля optional с разумными defaults
- Output: api_version в metadata для версионирования формата

**First Principles Validation:**
- Command Registry (не простой map): требуется metadata для help и deprecated warnings
- Wire DI (не manual): экономия boilerplate оправдывает learning curve; fallback — manual DI если Wire не интегрируется с legacy
- Logger interface (не прямой slog): необходим для mock в тестах
- Trace ID в Epic 1 (не отложен): FR35 требует корреляцию логов с первого дня

**Performance Validation (Story 1.4):**
- Benchmark: logging overhead < 1ms per 1000 log entries

## Detailed Design

### Services and Modules

| Модуль | Ответственность | Входы | Выходы | Owner |
|--------|-----------------|-------|--------|-------|
| `internal/command/registry.go` | Регистрация и поиск handlers | Handler impl | Handler by name | Story 1.1 |
| `internal/command/deprecated.go` | Wrapper для deprecated команд | Handler, old/new names | Warning + delegation | Story 1.2 |
| `internal/pkg/output/` | Форматирование результатов | Result struct | Text/JSON string | Story 1.3 |
| `internal/pkg/errors/` | Structured errors | Code, Message, Cause | AppError | Story 1.3 |
| `internal/pkg/logging/` | Абстракция логирования | Log entries | JSON to stderr | Story 1.4 |
| `internal/pkg/tracing/` | Генерация trace_id | — | UUID string | Story 1.5 |
| `internal/config/` | Расширенная конфигурация | YAML files, env vars | Config struct | Story 1.6 |
| `internal/di/` | Wire providers и injectors | Config | Initialized deps | Story 1.7 |
| `internal/command/handlers/version/` | nr-version handler | Config | Version info | Story 1.8 |
| `internal/command/help.go` | Auto-generated help | Registry | Help text | Story 1.9 |

### Data Models and Contracts

**Result (output contract):**
```go
// internal/pkg/output/result.go
type Result struct {
    Status   string      `json:"status"`             // "success" | "error"
    Command  string      `json:"command"`            // e.g., "nr-version"
    Data     interface{} `json:"data,omitempty"`     // command-specific payload
    Error    *ErrorInfo  `json:"error,omitempty"`    // present if status="error"
    Metadata *Metadata   `json:"metadata,omitempty"` // timing, trace, api version
}

type ErrorInfo struct {
    Code    string `json:"code"`    // e.g., "ERR_CONFIG_LOAD"
    Message string `json:"message"` // human-readable
}

type Metadata struct {
    DurationMs int64  `json:"duration_ms"`
    TraceID    string `json:"trace_id,omitempty"`
    APIVersion string `json:"api_version"` // "v1"
}

// Summary for command metrics (FR68 preparation, из Decision Matrix)
type Summary struct {
    ItemsProcessed int      `json:"items_processed,omitempty"`
    Warnings       []string `json:"warnings,omitempty"`
    // command-specific metrics added in Epic 2-5
}
```

**Result Extensions (из Decision Matrix):**
- Summary секция для command metrics (FR68 preparation)
- Backward compatible: omitempty для всех новых полей

**Data Type Safety (из Root Cause Analysis):**
- Каждый handler определяет typed Data struct (не interface{})
- Golden test для каждого command output schema
- Пример: `VersionData`, `HelpData`, etc.

**Error Code Registry (из Root Cause Analysis):**
- Все error codes регистрируются в `internal/pkg/errors/registry.go`
- NewAppError() валидирует код против registry
- CI check: новые error codes требуют review

**AppError (internal error contract):**
```go
// internal/pkg/errors/errors.go
type AppError struct {
    Code    string // Machine-readable: CATEGORY.SPECIFIC_ERROR
    Message string // Human-readable, NO secrets
    Cause   error  // Wrapped original error
}

// Error codes - hierarchical format (из Decision Matrix)
const (
    // Category: CONFIG
    ErrConfigLoad     = "CONFIG.LOAD_FAILED"
    ErrConfigParse    = "CONFIG.PARSE_FAILED"
    ErrConfigValidate = "CONFIG.VALIDATION_FAILED"

    // Category: COMMAND
    ErrCommandNotFound = "COMMAND.NOT_FOUND"
    ErrCommandExec     = "COMMAND.EXEC_FAILED"

    // Category: OUTPUT
    ErrOutputFormat = "OUTPUT.FORMAT_FAILED"

    // Categories for Epic 2-5: DB, ONEC, GITEA, SONARQUBE
)
```

**Error Codes Structure (из Decision Matrix):**
- Hierarchical format: `CATEGORY.SPECIFIC_ERROR`
- Categories: CONFIG, COMMAND, OUTPUT, DB, ONEC, GITEA, SONARQUBE
- Grep-friendly: `grep "CONFIG\."` для всех config ошибок

**Handler (command contract):**
```go
// internal/command/handler.go
type Handler interface {
    Name() string                                    // command name, e.g., "nr-version"
    Description() string                             // for help, e.g., "Show version info"
    Execute(ctx context.Context, cfg *config.Config) error
}

// Optional interfaces for extended functionality (из Decision Matrix)
type Validator interface {
    Validate(cfg *config.Config) error
}

type DryRunner interface {
    DryRun(ctx context.Context, cfg *config.Config) (*Plan, error)
}

type ProgressReporter interface {
    ReportProgress(ctx context.Context, progress float64, message string)
}
```

**Handler Extensibility (из Decision Matrix):**
- Core interface остаётся minimal (3 метода) — ISP compliance
- Optional interfaces для расширенной функциональности:
  - `Validator` — для pre-execution validation
  - `DryRunner` — для dry-run mode (Epic 3)
  - `ProgressReporter` — для progress bar (Epic 3)

**Config extensions (new sections):**
```go
// internal/config/config.go (additions)
type Config struct {
    // ... existing fields ...
    Implementations ImplementationsConfig `yaml:"implementations"`
    Logging         LoggingConfig         `yaml:"logging"`
}

type ImplementationsConfig struct {
    ConfigExport string `yaml:"config_export" env:"BR_IMPL_CONFIG_EXPORT" env-default:"1cv8"`
    DBCreate     string `yaml:"db_create" env:"BR_IMPL_DB_CREATE" env-default:"1cv8"`
}

type LoggingConfig struct {
    Format string `yaml:"format" env:"BR_LOG_FORMAT" env-default:"text"`
    Level  string `yaml:"level" env:"BR_LOG_LEVEL" env-default:"info"`
    File   string `yaml:"file" env:"BR_LOG_FILE"` // optional, empty = stderr only
}

// Command timeout (из Service Blueprint)
CommandTimeout time.Duration `yaml:"command_timeout" env:"BR_COMMAND_TIMEOUT" env-default:"0"` // 0 = no timeout
```

**Config additions (из Service Blueprint):**
- `command_timeout` — опциональный timeout для команд (0 = без ограничений)
- Используется с Context.WithTimeout в main.go

### APIs and Interfaces

**Logger Interface:**
```go
// internal/pkg/logging/logger.go
type Logger interface {
    Debug(ctx context.Context, msg string, args ...any)
    Info(ctx context.Context, msg string, args ...any)
    Warn(ctx context.Context, msg string, args ...any)
    Error(ctx context.Context, msg string, args ...any)
    With(args ...any) Logger  // returns new logger with added context
}
```

**Logger Context Integration (из Fishbone Analysis):**
- Logger методы принимают context первым аргументом
- trace_id автоматически извлекается из context и добавляется в каждую запись

**OutputWriter Interface:**
```go
// internal/pkg/output/writer.go
type Writer interface {
    Write(w io.Writer, result *Result) error
}

// Implementations: TextWriter, JSONWriter
// Factory: NewWriter(format string) Writer
```

**Registry API:**
```go
// internal/command/registry.go
func Register(h Handler)                    // called in init()
func RegisterWithAlias(h Handler, deprecated string) // NR-migration
func Get(name string) (Handler, bool)       // lookup by name
func ListAll() []Handler                    // for help generation
```

**Registry Validation (из Fishbone Analysis):**
- Register() паникует если Description() пустой (programming error)
- Get() возвращает (nil, false) для пустого registry (graceful degradation)

**Safe Execution Wrapper (из Root Cause Analysis):**
```go
// internal/command/executor.go
func SafeExecute(ctx context.Context, h Handler, cfg *config.Config) error
```
- Оборачивает handler.Execute() в defer/recover
- Конвертирует panic в AppError с code "COMMAND.PANIC"
- Логирует stack trace перед возвратом ошибки

**Context Keys:**
```go
// internal/pkg/tracing/context.go
type traceIDKey struct{}

func WithTraceID(ctx context.Context, id string) context.Context
func TraceIDFromContext(ctx context.Context) string
```

### Workflows and Sequencing

**Command Execution Flow:**
```
┌─────────────────────────────────────────────────────────────────────┐
│                         main.go                                      │
├─────────────────────────────────────────────────────────────────────┤
│ 1. Parse BR_COMMAND from env                                         │
│ 2. Generate trace_id, add to context                                 │
│ 3. Initialize Logger with trace_id                                   │
│ 4. Load Config (cleanenv)                                            │
│ 5. Initialize Wire (NewApp())                                        │
│ 6. command.Get(BR_COMMAND) → handler, found                          │
│    ├─ found=true → handler.Execute(ctx, cfg)                         │
│    └─ found=false → legacy switch (fallback)                         │
│ 7. Format result via OutputWriter                                    │
│ 8. Exit with appropriate code                                        │
└─────────────────────────────────────────────────────────────────────┘
```

**Registry Registration Flow (compile-time):**
```
┌─────────────────────────────────────────────────────────────────────┐
│ Package init() execution order (Go runtime)                          │
├─────────────────────────────────────────────────────────────────────┤
│ 1. internal/command/registry.go init() → creates map                 │
│ 2. internal/command/handlers/version/version.go init()               │
│    └─ command.Register(&VersionHandler{})                            │
│ 3. ... other handlers ...                                            │
│ 4. main.go init() → registry populated                               │
└─────────────────────────────────────────────────────────────────────┘
```

**Wire Dependency Graph:**
```
                    ┌─────────────┐
                    │   Config    │
                    └──────┬──────┘
                           │
           ┌───────────────┼───────────────┐
           │               │               │
           ▼               ▼               ▼
    ┌──────────┐    ┌──────────┐    ┌──────────┐
    │  Logger  │    │ Output   │    │  Tracer  │
    │ (slog)   │    │ Writer   │    │  (ID)    │
    └──────────┘    └──────────┘    └──────────┘
           │               │               │
           └───────────────┼───────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │     App     │
                    │  (injected) │
                    └─────────────┘
```

**Deprecated Command Flow:**
```
BR_COMMAND=old-name
        │
        ▼
command.Get("old-name")
        │
        ▼ (returns DeprecatedBridge)
DeprecatedBridge.Execute()
        │
        ├─► log.Warn("deprecated", "old", "old-name", "new", "nr-new-name")
        │
        └─► actual.Execute(ctx, cfg)
                │
                ▼
           Normal execution
```

**Integration Test Flow (из Fishbone Analysis):**
```
Register(TestHandler) → Get("test") → Execute(ctx, cfg) → OutputWriter.Write() → Verify JSON
```
- Покрывает: Registry → Handler → Output pipeline
- Запускается в CI на каждый PR
- Файл: `internal/command/integration_test.go`

**Service Blueprint Insights:**

**Graceful Shutdown:**
- Signal handler для SIGTERM/SIGINT в main.go
- Flush logs и traces перед exit
- Context cancellation propagation через всю цепочку

**Command Timeout:**
- Опциональный timeout из config: `command_timeout: 5m`
- Context.WithTimeout для long-running commands
- Graceful cancellation с partial result (если возможно)

**Panic Recovery:**
- defer/recover в main.go
- Log panic с stack trace перед exit
- Exit code = 2 для panic (отличается от обычной ошибки)

## Non-Functional Requirements

### Performance

**Измеримые требования (из PRD NFR1-NFR2):**

| Метрика | Целевое значение | Метод измерения |
|---------|------------------|-----------------|
| Время старта приложения | < 500ms | Benchmark: time от запуска до готовности к выполнению команды |
| Overhead от logging | < 1ms на 1000 записей | Benchmark: slog с JSON handler vs NOP handler |
| Overhead от trace_id генерации | < 100μs | Benchmark: crypto/rand UUID generation |
| Registry lookup | O(1) | Map-based implementation |

**Оптимизации:**
- Wire compile-time DI: нет reflection overhead в runtime
- Lazy initialization: компоненты создаются только при использовании
- Logger: асинхронный flush не блокирует основной поток (подготовка к Epic 6)

**Benchmarks (Story 1.4):**
```go
// internal/pkg/logging/benchmark_test.go
func BenchmarkLoggerJSON(b *testing.B) // target: < 1ms per 1000 entries
func BenchmarkLoggerWithTraceID(b *testing.B) // verify trace_id extraction overhead
```

### Security

**Требования из PRD (NFR3-NFR5):**

| Требование | Реализация в Epic 1 |
|------------|---------------------|
| NFR3: Secrets не в логах | Logger маскирует поля с паттернами `password`, `token`, `secret`, `key` |
| NFR4: Secrets через env/files | Config загружает sensitive данные из env vars или protected files |
| NFR5: API на localhost | N/A для Epic 1 (нет API endpoints) |

**Реализация маскирования (Story 1.4):**
```go
// internal/pkg/logging/masking.go
type Secret string

func (s Secret) LogValue() slog.Value {
    return slog.StringValue("***REDACTED***")
}

// Автоматическое маскирование для известных полей
var sensitivePatterns = []string{"password", "token", "secret", "key", "credential"}
```

**Error Message Safety (Story 1.3):**
- AppError.Message НИКОГДА не содержит sensitive данные
- Cause error может содержать детали, но маскируется при сериализации в JSON
- Валидация: unit-тест проверяет что JSON output не содержит patterns из sensitivePatterns

**Input Validation:**
- BR_COMMAND: alphanumeric + hyphen only (regex validation)
- BR_OUTPUT_FORMAT: enum validation (text|json|yaml)
- Config paths: проверка на path traversal (../) запрещена

### Reliability/Availability

**CLI-специфичные требования:**

| Аспект | Требование | Реализация |
|--------|------------|------------|
| Graceful shutdown | Корректное завершение при SIGTERM/SIGINT | Signal handler в main.go, context cancellation |
| Panic recovery | Не crash без диагностики | defer/recover в main.go, log stack trace |
| Config fallback | Работа при недоступности remote config | Продолжить с local defaults, warn в логе |
| Registry fallback | Работа при пустом registry | Fallback на legacy switch в main.go |

**Exit Codes (стандартизация):**
```
0  - Success
1  - General error (AppError)
2  - Panic / Unknown command
3  - Config error
4  - Timeout
5+ - Command-specific errors (определяются в Epic 2-5)
```

**Error Recovery Patterns:**
- Network errors: НЕТ retry в Epic 1 (нет network операций)
- Config errors: Fail fast с понятным сообщением
- Registry errors: Graceful fallback на legacy

**Idempotency:**
- nr-version: идемпотентна по определению (read-only)
- help: идемпотентна по определению (read-only)

### Observability

**Реализация в Epic 1 (FR32-FR35, FR42):**

| Сигнал | Формат | Destination | Требование |
|--------|--------|-------------|------------|
| Logs | JSON | stderr | Каждая запись содержит trace_id |
| Output | JSON/Text | stdout | Структурированный Result с metadata |
| Trace ID | UUID v4 | context → logs → output | Уникальный для каждой операции |

**Обязательные поля в логах:**
```json
{
    "timestamp": "2025-01-15T10:30:00Z",
    "level": "INFO",
    "msg": "Command executed",
    "command": "nr-version",
    "trace_id": "abc123def456",
    "duration_ms": 150
}
```

**Observability Preparation (для Epic 6):**
- Logger interface готов для async export
- Trace ID генерируется в формате, совместимом с OpenTelemetry
- Metadata в Result подготовлена для Prometheus labels

**Diagnostic Capabilities:**
- trace_id позволяет коррелировать все записи одной операции
- duration_ms в каждом логе для timing analysis
- Hierarchical error codes позволяют grep по категориям

**Уровни логирования:**
- DEBUG: Детали для troubleshooting (Wire initialization, config parsing)
- INFO: Значимые события (command start/end, registry lookup)
- WARN: Recoverable issues (deprecated command usage, config fallback)
- ERROR: Failures requiring attention (command execution failed)

## Dependencies and Integrations

### Go Module Dependencies (go.mod additions)

| Зависимость | Версия | Назначение | Критичность |
|-------------|--------|------------|-------------|
| github.com/google/wire | v0.6.0 | Compile-time DI | Высокая |
| — | Go 1.21+ stdlib | slog для structured logging | Критическая |
| — | Go stdlib | crypto/rand для trace_id | Низкая |

**Существующие зависимости (без изменений):**
- github.com/ilyakaznacheev/cleanenv v1.5.0 — загрузка конфигурации
- gopkg.in/yaml.v3 — YAML parsing (для config)

### Build Dependencies

| Инструмент | Версия | Назначение |
|------------|--------|------------|
| Wire CLI | v0.6.0 | Генерация wire_gen.go |
| golangci-lint | v2+ | Linting (включая запрет fmt.Print*) |
| Go | 1.25+ | Требует slog (stdlib с Go 1.21) |

### Internal Integration Points

| Компонент | Интеграция | Направление |
|-----------|------------|-------------|
| main.go | Command Registry | main → registry.Get() |
| main.go | Legacy switch | Fallback если registry не найдёт |
| Config loader | Существующий cleanenv | Без изменений API |
| Handlers | Wire-injected App | Handler → App.Logger, App.Output |

### External Dependencies (Epic 1)

**НЕТ внешних зависимостей в Epic 1.**

Epic 1 не взаимодействует с:
- 1C:Enterprise (Epic 2-4)
- Gitea API (Epic 4-5)
- SonarQube API (Epic 5)
- MSSQL (Epic 3)
- OpenTelemetry collector (Epic 6)

Это сознательное решение — Epic 1 создаёт foundation без внешних зависимостей для минимизации risk.

### CI/CD Integration

```yaml
# .gitea/workflows/build.yml additions
- name: Generate Wire
  run: go generate ./internal/di/...

- name: Verify Wire Generated
  run: git diff --exit-code internal/di/wire_gen.go
```

## Acceptance Criteria (Authoritative)

### AC1: Command Registry функционирует
- **Given** новый handler реализующий interface Handler
- **When** handler вызывает command.Register() в init()
- **Then** команда доступна через command.Get(name)
- **And** Registry возвращает (nil, false) для несуществующих команд
- **And** повторная регистрация с тем же именем паникует

### AC2: NR-Migration Bridge работает
- **Given** команда зарегистрирована через RegisterWithAlias(handler, "old-name")
- **When** вызывается старое имя команды
- **Then** выводится warning в stderr с рекомендацией нового имени
- **And** команда выполняется через actual handler

### AC3: OutputWriter форматирует результаты
- **Given** BR_OUTPUT_FORMAT=json
- **When** команда завершается
- **Then** stdout содержит валидный JSON со структурой Result
- **And** metadata содержит api_version, duration_ms, trace_id
- **And** stderr содержит ТОЛЬКО логи (никогда данные)

### AC4: Structured Errors содержат коды
- **Given** операция завершается с ошибкой
- **When** ошибка сериализуется
- **Then** JSON содержит error.code (hierarchical) и error.message
- **And** message НЕ содержит sensitive данные

### AC5: Logger пишет в stderr с trace_id
- **Given** команда выполняется
- **When** происходит логирование
- **Then** каждая запись в JSON формате
- **And** каждая запись содержит trace_id из context
- **And** логи идут ТОЛЬКО в stderr

### AC6: Trace ID генерируется и пропагируется
- **Given** команда начинает выполнение
- **When** генерируется trace_id
- **Then** trace_id в формате UUID v4 или 16-byte hex
- **And** trace_id присутствует во всех логах операции
- **And** trace_id включён в Result.metadata

### AC7: Config расширен новыми секциями
- **Given** конфигурация содержит секции implementations и logging
- **When** конфигурация загружается
- **Then** значения доступны через config struct
- **And** существующие production конфиги парсятся без ошибок (backward compat)

### AC8: Wire DI компилируется и работает
- **Given** Wire definitions в internal/di/wire.go
- **When** выполняется go generate ./internal/di/...
- **Then** генерируется wire_gen.go без ошибок
- **And** NewApp() возвращает инициализированный App с Logger, OutputWriter

### AC9: nr-version работает через Registry
- **Given** BR_COMMAND=nr-version
- **When** команда выполняется
- **Then** выводится версия, Go версия, дата сборки
- **And** вывод соответствует OutputWriter формату
- **And** trace_id присутствует в логах
- **And** exit code = 0

### AC10: Help генерируется из Registry
- **Given** BR_COMMAND=help
- **When** команда выполняется
- **Then** выводится список всех зарегистрированных команд
- **And** deprecated команды помечены [deprecated]
- **And** каждая команда имеет description

### AC11: Fallback на legacy работает
- **Given** BR_COMMAND=existing-legacy-command
- **When** команда не найдена в Registry
- **Then** выполняется через legacy switch в main.go
- **And** логируется какой путь выбран ("registry" или "legacy")

### AC12: Production readiness
- **Given** Epic 1 завершён
- **When** nr-version интегрирован в production pipeline
- **Then** команда работает стабильно минимум 1 неделю
- **And** все CI checks проходят (lint, test, wire generate)

## Traceability Mapping

| AC | FR(s) | Spec Section | Component(s) | Test Idea |
|----|-------|--------------|--------------|-----------|
| AC1 | FR1, FR3, FR47 | Services/Modules: registry.go | `internal/command/registry.go` | Unit: Register → Get returns handler; Get unknown → nil,false; Double register → panic |
| AC2 | FR48, FR50 | Services/Modules: deprecated.go | `internal/command/deprecated.go` | Unit: Execute logs warning; Integration: deprecated command works via bridge |
| AC3 | FR29, FR30, FR31 | APIs: OutputWriter Interface | `internal/pkg/output/writer.go`, `json.go`, `text.go` | Unit: JSON valid; Golden test: stable format; Integration: stdout only Result |
| AC4 | FR29 | Data Models: AppError | `internal/pkg/errors/errors.go` | Unit: Code/Message present; No sensitive patterns in serialized error |
| AC5 | FR33, FR34, FR35 | APIs: Logger Interface | `internal/pkg/logging/logger.go`, `slog.go` | Unit: JSON format; trace_id in every entry; Output to stderr only |
| AC6 | FR35, FR42 | Workflows: Trace ID | `internal/pkg/tracing/traceid.go`, `context.go` | Unit: UUID format valid; Integration: trace_id in logs AND output metadata |
| AC7 | FR2, FR28 | Data Models: Config extensions | `internal/config/config.go` | Unit: Parse new sections; Backward compat: existing config parses |
| AC8 | FR4, FR5 | Workflows: Wire DI | `internal/di/wire.go`, `wire_gen.go` | Build: go generate succeeds; Unit: NewApp returns non-nil |
| AC9 | FR47 | Services/Modules: version handler | `internal/command/handlers/version/` | Integration: Full flow Registry→Execute→Output; E2E: built binary works |
| AC10 | FR3 | Services/Modules: help.go | `internal/command/help.go` | Unit: ListAll returns all; Integration: help output contains nr-version |
| AC11 | FR47, FR48 | Workflows: Command Execution Flow | `cmd/benadis-runner/main.go` | Integration: Unknown command → legacy fallback; Log shows path taken |
| AC12 | — | — | Production pipeline | E2E: nr-version in real Gitea Actions workflow |

### FR → Story → AC Mapping

| FR | Story | AC | Verification |
|----|-------|-----|--------------|
| FR1 | 1.1 | AC1 | Unit tests |
| FR2 | 1.6 | AC7 | Unit + backward compat test |
| FR3 | 1.1, 1.9 | AC1, AC10 | Unit + integration |
| FR4 | 1.7 | AC8 | Build + unit |
| FR5 | 1.7 | AC8 | Build + unit |
| FR28 | 1.6 | AC7 | Unit test |
| FR29 | 1.3 | AC3, AC4 | Unit + golden tests |
| FR30 | 1.3 | AC3 | Unit test |
| FR31 | 1.3, 1.4 | AC3, AC5 | Integration test |
| FR33 | 1.4 | AC5 | Unit test |
| FR34 | 1.4, 1.6 | AC5, AC7 | Unit test |
| FR35 | 1.5 | AC5, AC6 | Unit + integration |
| FR42 | 1.5 | AC6 | Unit test |
| FR47 | 1.1, 1.8 | AC1, AC9 | Integration + E2E |
| FR48 | 1.2 | AC2 | Unit + integration |
| FR50 | 1.2 | AC2 | Unit test |

## Risks, Assumptions, Open Questions

### Risks

| ID | Risk | Probability | Impact | Mitigation |
|----|------|-------------|--------|------------|
| R1 | Wire DI не интегрируется с legacy кодом | Medium | High | Wire providers работают независимо от internal/app/; fallback на manual DI если критично |
| R2 | Registry конфликтует с main.go switch | Medium | High | Fallback логика: сначала registry, потом legacy switch; логирование пути для debug |
| R3 | JSON output breaking changes | High | Medium | Golden tests для стабильности; api_version в metadata; JSON Schema validation |
| R4 | Логи смешиваются с output в stdout | High | High | Logger → stderr ONLY; golangci-lint запрет fmt.Print*; integration test проверяет разделение |
| R5 | CI build отличается от local | Low | Medium | CI тестирует собранный бинарник; fallback build info (version="dev") |
| R6 | Новые config поля ломают production | Medium | Medium | ВСЕ новые поля optional с defaults; backward compatibility тест на real production config |

### Assumptions

| ID | Assumption | Validation | If False |
|----|------------|------------|----------|
| A1 | Go 1.25+ доступен в CI/CD | CI использует Go 1.25.4 | Backport slog или внешняя зависимость |
| A2 | golangci-lint v2 поддерживает custom rules | Проверить документацию | Использовать pre-commit hook вместо lint |
| A3 | Wire v0.6.0 stable | Используется в production проектах | Pin version; fallback на manual DI |
| A4 | Существующие production конфиги валидны | Получить примеры от DevOps | Добавить migration path |
| A5 | init() порядок детерминирован в рамках пакета | Go spec гарантирует | Явная инициализация в main |

### Open Questions

| ID | Question | Owner | Deadline | Resolution |
|----|----------|-------|----------|------------|
| Q1 | Какой точный формат trace_id? UUID v4 или W3C Trace Context? | Architect | Story 1.5 start | UUID v4 для простоты; W3C-совместимость в Epic 6 |
| Q2 | Нужен ли rate limiting для deprecated warnings? | Tech Lead | Story 1.2 start | Нет — warning на каждый вызов для visibility |
| Q3 | Как тестировать E2E в CI без production infrastructure? | DevOps | Story 1.8 | Mock-free unit tests + integration tests с testify |
| Q4 | Приоритет backward compat vs clean API? | Product Owner | Before Story 1.3 | Backward compat выше — все поля optional |

### Pre-mortem Failure Modes (из Epic Analysis)

| FM | Failure Mode | AC Coverage |
|----|--------------|-------------|
| FM1 | Wire DI не компилируется | AC8: go generate test |
| FM2 | Registry конфликт с main.go | AC11: fallback test |
| FM3 | JSON breaking changes | AC3: golden tests |
| FM4 | Логи в stdout | AC5: stderr-only test |
| FM5 | CI build != local | AC9: built binary test |
| FM6 | Config breaks old files | AC7: backward compat test |

## Test Strategy Summary

### Test Levels

| Level | Scope | Framework | Coverage Target |
|-------|-------|-----------|-----------------|
| Unit | Individual functions/structs | testify/assert | 80%+ для нового кода |
| Integration | Cross-component flows | testify/suite | Key flows: Registry→Handler→Output |
| E2E | Built binary execution | Bash/Make | nr-version full cycle |
| Benchmark | Performance validation | testing.B | Logger overhead < 1ms/1000 entries |

### Test Categories by Story

| Story | Unit Tests | Integration Tests | E2E |
|-------|-----------|-------------------|-----|
| 1.1 Registry | Register, Get, ListAll, panic on duplicate | — | — |
| 1.2 Deprecated | DeprecatedBridge.Execute logs warning | Bridge + actual handler | — |
| 1.3 Output | JSONWriter, TextWriter, Result serialization | stdout/stderr separation | — |
| 1.4 Logger | slog adapter, With(), level filtering | trace_id injection | — |
| 1.5 TraceID | UUID generation, context get/set | trace_id in logs | — |
| 1.6 Config | Parse new sections, defaults, env override | Backward compat with prod config | — |
| 1.7 Wire | Provider functions return non-nil | NewApp() full initialization | — |
| 1.8 nr-version | — | Registry→Execute→Output flow | Built binary outputs JSON |
| 1.9 Help | ListAll contains all handlers | Help output formatting | — |

### Golden Tests (Stability)

```
testdata/
├── golden/
│   ├── result_success.json      # AC3: JSON format stability
│   ├── result_error.json        # AC4: Error format stability
│   └── help_output.txt          # AC10: Help format stability
```

**Golden test workflow:**
1. First run: generate golden file
2. Subsequent runs: compare output to golden
3. Breaking change: explicit update with review

### Edge Cases and Negative Tests

| AC | Edge Case | Expected Behavior |
|----|-----------|-------------------|
| AC1 | Register nil handler | Panic (programming error) |
| AC1 | Register empty name | Panic (programming error) |
| AC2 | RegisterWithAlias empty deprecated | No bridge, just register |
| AC3 | Result with nil Data | Valid JSON with `"data": null` |
| AC4 | Error with Cause containing secret | Cause masked in JSON |
| AC5 | Log with empty message | Valid JSON entry |
| AC6 | TraceIDFromContext on empty context | Return empty string |
| AC7 | Config missing implementations section | Use defaults |
| AC8 | Wire cycle detection | Compile-time error |
| AC11 | BR_COMMAND empty | Help output |

### CI Integration

```yaml
# .gitea/workflows/test.yml
jobs:
  test:
    steps:
      - name: Unit Tests
        run: go test -v -race ./...

      - name: Coverage
        run: go test -coverprofile=coverage.out ./...
        # Target: 80% for new packages

      - name: Wire Generate
        run: |
          go generate ./internal/di/...
          git diff --exit-code internal/di/wire_gen.go

      - name: Golden Tests
        run: go test -v ./... -update=false

      - name: Build Binary
        run: make build

      - name: E2E Test
        run: |
          BR_COMMAND=nr-version BR_OUTPUT_FORMAT=json ./benadis-runner | jq .
          # Verify valid JSON output
```

### Definition of Done (Testing)

- [ ] All unit tests pass
- [ ] Integration tests cover AC1-AC11
- [ ] Golden tests prevent format regression
- [ ] Coverage ≥ 80% for new code
- [ ] Wire generation stable in CI
- [ ] E2E: nr-version outputs valid JSON from built binary
- [ ] Benchmarks: Logger overhead < 1ms per 1000 entries
