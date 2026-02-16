# Architecture

## Executive Summary

benadis-runner v2.0 — архитектурная трансформация CLI-инструмента автоматизации для 1C:Enterprise. Ключевые принципы: SOLID-совместимость, DI через Wire, Command Registry для расширяемости, Strategy pattern для сменных реализаций (1cv8/ibcmd/native), и полный observability-стек (slog + OpenTelemetry).

## Decision Summary

| Category | Decision | Version | Affects FR Categories | Rationale |
| -------- | -------- | ------- | --------------------- | --------- |
| DI Container | Wire (google/wire) | v0.6.0 | FR1-FR5 (Архитектура) | Compile-time DI, нет runtime overhead |
| Command Pattern | Command Registry + init() | — | FR47-FR50 (Миграция) | OCP — новые команды без изменения main |
| Interface Segregation | Role-based interfaces | — | FR1-FR5 | ISP — минимальные зависимости |
| Logging | slog (stdlib) | Go 1.21+ | FR32-FR35 | Stdlib, JSON, zero deps |
| Tracing | OpenTelemetry | v1.28+ | FR41-FR43 | Vendor-neutral, async export |
| Output Format | OutputWriter interface | — | FR29-FR31 | Text/JSON/YAML через единый контракт |
| Error Handling | Structured AppError | — | FR29-FR31 | Коды + wrapping для автоматизации |
| Strategy Pattern | Interface + Factory | — | FR2, FR18, FR20 | Сменные реализации через config |
| Configuration | cleanenv | v1.5.0 | FR28 | Сохранение совместимости |
| Database | go-mssqldb | v1.7+ | FR10-FR13 | Существующая интеграция |

## Project Structure

```
benadis-runner/
├── cmd/
│   └── benadis-runner/
│       └── main.go              # Точка входа, инициализация Wire
├── internal/
│   ├── di/                      # Wire providers и injectors
│   │   ├── wire.go              # Wire definitions
│   │   └── wire_gen.go          # Generated
│   ├── command/                 # Command Registry
│   │   ├── registry.go          # Реестр команд
│   │   ├── handler.go           # Base handler interface
│   │   └── handlers/            # Конкретные handlers
│   │       ├── servicemode/     # service-mode-* commands
│   │       ├── database/        # dbrestore, dbupdate, create-temp-db
│   │       ├── store/           # store2db, storebind, git2store, create-stores
│   │       ├── convert/         # convert command
│   │       ├── sonarqube/       # sq-* commands
│   │       └── gitea/           # action-menu-build, test-merge
│   ├── domain/                  # Чистая бизнес-логика
│   │   ├── servicemode/         # Доменная логика сервисного режима
│   │   ├── database/            # Доменная логика БД операций
│   │   ├── store/               # Доменная логика хранилищ
│   │   └── convert/             # Доменная логика конвертации
│   ├── service/                 # Orchestration layer
│   │   ├── servicemode.go       # Оркестрация service-mode
│   │   ├── database.go          # Оркестрация DB operations
│   │   ├── store.go             # Оркестрация store operations
│   │   └── sonarqube.go         # Оркестрация SQ operations
│   ├── adapter/                 # External integrations
│   │   ├── onec/                # 1C:Enterprise adapters
│   │   │   ├── interfaces.go    # ConfigExporter, DatabaseCreator, etc.
│   │   │   ├── onecv8/          # 1cv8 implementation
│   │   │   ├── ibcmd/           # ibcmd implementation
│   │   │   └── rac/             # RAC client
│   │   ├── gitea/               # Gitea adapter
│   │   │   ├── interfaces.go    # PRReader, CommitReader, etc.
│   │   │   └── client.go        # HTTP client implementation
│   │   ├── sonarqube/           # SonarQube adapter
│   │   │   ├── interfaces.go    # ProjectsAPI, AnalysesAPI, etc.
│   │   │   └── client.go        # HTTP client implementation
│   │   └── mssql/               # MSSQL adapter
│   │       ├── interfaces.go    # DatabaseRestorer, etc.
│   │       └── client.go        # go-mssqldb wrapper
│   ├── config/                  # Configuration
│   │   ├── config.go            # Main config struct
│   │   ├── loader.go            # Multi-source loader
│   │   └── validation.go        # Config validation
│   └── pkg/                     # Shared utilities
│       ├── logging/             # Logger interface + slog adapter
│       │   ├── logger.go        # Interface definition
│       │   └── slog.go          # slog implementation
│       ├── output/              # Output formatting
│       │   ├── writer.go        # OutputWriter interface
│       │   ├── text.go          # Text formatter
│       │   ├── json.go          # JSON formatter
│       │   └── yaml.go          # YAML formatter
│       ├── errors/              # Structured errors
│       │   └── errors.go        # AppError type
│       └── tracing/             # OpenTelemetry setup
│           └── otel.go          # Tracer initialization
├── docs/
│   └── architecture/
│       └── adr/                 # Architecture Decision Records
├── bdocs/                       # BMAD documentation
├── vendor/                      # Vendored dependencies
├── Makefile
├── go.mod
└── go.sum
```

## FR Category to Architecture Mapping

| FR Category | Package | Handler |
|-------------|---------|---------|
| Архитектура (FR1-FR5) | `internal/di/`, `internal/command/` | — |
| Сервисный режим (FR6-FR9) | `internal/domain/servicemode/`, `internal/adapter/onec/rac/` | `handlers/servicemode/` |
| Операции с БД (FR10-FR13) | `internal/domain/database/`, `internal/adapter/mssql/` | `handlers/database/` |
| Синхронизация (FR14-FR18) | `internal/domain/store/`, `internal/adapter/onec/` | `handlers/store/` |
| Конвертация (FR19-FR20) | `internal/domain/convert/`, `internal/adapter/onec/` | `handlers/convert/` |
| Выполнение EPF (FR21) | `internal/adapter/onec/onecv8/` | `handlers/convert/` |
| SonarQube (FR22-FR25) | `internal/service/sonarqube.go`, `internal/adapter/sonarqube/` | `handlers/sonarqube/` |
| Gitea (FR26-FR28) | `internal/adapter/gitea/` | `handlers/gitea/` |
| Вывод (FR29-FR31) | `internal/pkg/output/` | — |
| Логирование (FR32-FR35) | `internal/pkg/logging/` | — |
| Алертинг (FR36-FR40) | `internal/pkg/alerting/` (future) | — |
| Трассировка (FR41-FR43) | `internal/pkg/tracing/` | — |
| Отладка (FR44-FR46) | Build flags + Delve | — |
| Миграция (FR47-FR50) | `internal/command/registry.go` | — |

## Technology Stack Details

### Core Technologies

| Technology | Version | Purpose |
|------------|---------|---------|
| Go | 1.25.1 | Primary language |
| Wire | v0.6.0 | Compile-time DI |
| slog | stdlib (Go 1.21+) | Structured logging |
| OpenTelemetry Go | v1.28+ | Distributed tracing |
| cleanenv | v1.5.0 | Configuration loading |
| go-mssqldb | v1.7+ | MSSQL driver |
| testify | v1.11+ | Testing assertions |
| sqlmock | v1.5+ | Database mocking |
| golangci-lint | v2 | Linting (20+ linters) |

### Integration Points

```
┌─────────────────────────────────────────────────────────────────┐
│                        benadis-runner                           │
├─────────────────────────────────────────────────────────────────┤
│  Command Layer                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ Registry → Handler → Service → Domain → Adapter          │  │
│  └──────────────────────────────────────────────────────────┘  │
└───────────┬──────────────┬──────────────┬──────────────┬───────┘
            │              │              │              │
            ▼              ▼              ▼              ▼
      ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
      │ 1C:Ent   │  │  Gitea   │  │ SonarQube│  │  MSSQL   │
      │ 1cv8     │  │   API    │  │   API    │  │  Server  │
      │ ibcmd    │  │          │  │          │  │          │
      │ rac      │  │          │  │          │  │          │
      └──────────┘  └──────────┘  └──────────┘  └──────────┘
```

## Novel Pattern Designs

### Pattern: Switchable Implementation Strategy

**Purpose**: Позволяет выбирать реализацию операции (1cv8/ibcmd/native) через конфигурацию без изменения кода.

**Components**:
```go
// internal/adapter/onec/interfaces.go
type ConfigExporter interface {
    Export(ctx context.Context, opts ExportOptions) error
}

type DatabaseCreator interface {
    Create(ctx context.Context, opts CreateOptions) error
}

// internal/adapter/onec/factory.go
type OneCFactory struct {
    config *config.Config
}

func (f *OneCFactory) NewConfigExporter() ConfigExporter {
    switch f.config.Implementations.ConfigExport {
    case "ibcmd":
        return ibcmd.NewExporter(f.config)
    case "native":
        return native.NewExporter(f.config)
    default:
        return onecv8.NewExporter(f.config)
    }
}
```

**Data Flow**:
```
Config → Factory → Strategy Selection → Concrete Implementation → Execute
```

**Affects FR**: FR2 (выбор реализации через config), FR18 (инструмент для конфигурации), FR20 (инструмент конвертации)

### Pattern: Command Registry with Self-Registration

**Purpose**: Новые команды добавляются без изменения main.go (OCP).

**Components**:
```go
// internal/command/registry.go
type Handler interface {
    Name() string
    Execute(ctx context.Context, cfg *config.Config) error
}

var registry = make(map[string]Handler)

func Register(h Handler) {
    registry[h.Name()] = h
}

func Get(name string) (Handler, bool) {
    h, ok := registry[name]
    return h, ok
}

// internal/command/handlers/servicemode/status.go
func init() {
    command.Register(&StatusHandler{})
}

type StatusHandler struct{}

func (h *StatusHandler) Name() string { return "service-mode-status" }
```

**Data Flow**:
```
init() → Register → main() → Get(command) → Execute
```

**Affects FR**: FR47-FR50 (NR-миграция), все команды

### Pattern: NR-Migration Bridge

**Purpose**: Поддержка параллельного существования старых и новых команд.

**Components**:
```go
// internal/command/registry.go
func RegisterWithAlias(h Handler, deprecated string) {
    registry[h.Name()] = h
    if deprecated != "" {
        registry[deprecated] = &DeprecatedBridge{
            actual:     h,
            deprecated: deprecated,
            newName:    h.Name(),
        }
    }
}

type DeprecatedBridge struct {
    actual     Handler
    deprecated string
    newName    string
}

func (b *DeprecatedBridge) Execute(ctx context.Context, cfg *config.Config) error {
    log.Warn("Command deprecated",
        "old", b.deprecated,
        "new", b.newName,
        "migration", "Use "+b.newName+" instead")
    return b.actual.Execute(ctx, cfg)
}
```

**Affects FR**: FR47-FR50 (миграция), FR50 (логирование deprecated)

## Implementation Patterns

### Naming Patterns

| Entity | Convention | Example |
|--------|------------|---------|
| Package names | lowercase, short | `servicemode`, `onecv8` |
| Interface names | -er suffix or descriptive | `ConfigExporter`, `PRReader` |
| Struct names | PascalCase | `StatusHandler`, `AppError` |
| File names | snake_case | `service_mode.go`, `wire_gen.go` |
| Constants | PascalCase (exported), camelCase (internal) | `ActServiceModeEnable`, `defaultTimeout` |
| Environment vars | SCREAMING_SNAKE with prefix | `BR_COMMAND`, `BR_OUTPUT_FORMAT` |

### Structure Patterns

| Pattern | Convention |
|---------|------------|
| Tests location | Same package, `_test.go` suffix |
| Mocks location | `internal/mocks/` or inline in tests |
| Wire files | `wire.go` (definitions), `wire_gen.go` (generated) |
| Interfaces | Same package as primary implementation |

### Format Patterns

**API Response (internal)**:
```go
type Result struct {
    Status   string      `json:"status"`   // "success" | "error"
    Command  string      `json:"command"`
    Data     interface{} `json:"data,omitempty"`
    Error    *ErrorInfo  `json:"error,omitempty"`
    Metadata *Metadata   `json:"metadata,omitempty"`
}

type ErrorInfo struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

type Metadata struct {
    DurationMs int64  `json:"duration_ms"`
    TraceID    string `json:"trace_id,omitempty"`
}
```

**Log Entry (JSON)**:
```json
{
    "timestamp": "2025-01-15T10:30:00Z",
    "level": "INFO",
    "msg": "Command executed",
    "command": "service-mode-status",
    "trace_id": "abc123",
    "duration_ms": 150
}
```

### Error Handling

```go
// internal/pkg/errors/errors.go
type AppError struct {
    Code    string // Machine-readable: "ERR_DB_CONNECTION", "ERR_AUTH_FAILED"
    Message string // Human-readable
    Cause   error  // Wrapped error
}

func (e *AppError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Cause }

// Error codes
const (
    ErrConfigLoad      = "ERR_CONFIG_LOAD"
    ErrDBConnection    = "ERR_DB_CONNECTION"
    ErrDBRestore       = "ERR_DB_RESTORE"
    ErrServiceMode     = "ERR_SERVICE_MODE"
    ErrStoreOperation  = "ERR_STORE_OP"
    ErrGiteaAPI        = "ERR_GITEA_API"
    ErrSonarQubeAPI    = "ERR_SONARQUBE_API"
    ErrOneCOperation   = "ERR_ONEC_OP"
)
```

### Lifecycle Patterns

**Command Lifecycle**:
```
Parse Config → Validate → Initialize Dependencies (Wire) →
Execute Handler → Format Output → Cleanup → Exit
```

**Error Recovery**:
- Network errors: Retry with exponential backoff (max 3 attempts)
- 1C operations: No auto-retry (side effects), log and exit
- Config errors: Fail fast, clear error message

## Consistency Rules

### Naming Conventions

- **Команды**: kebab-case (`service-mode-status`, `sq-scan-branch`)
- **NR-команды**: `nr-` prefix (`nr-service-mode-status`)
- **Env vars**: `BR_` prefix для общих, специфичные префиксы для модулей
- **Exit codes**: 0 = success, 2 = unknown command, 5+ = specific errors

### Code Organization

- Один handler = один файл
- Domain logic не импортирует adapter
- Adapter не содержит бизнес-логику
- Wire providers в отдельном пакете `internal/di`

### Error Handling

- Все ошибки wrapping через `fmt.Errorf("context: %w", err)`
- AppError для user-facing ошибок
- Panic только для programming errors (не для runtime)
- Secrets никогда не попадают в error messages

### Logging Strategy

- **DEBUG**: Детали для troubleshooting
- **INFO**: Значимые события (старт/стоп команды, успешные операции)
- **WARN**: Recoverable issues, deprecated usage
- **ERROR**: Failures requiring attention

Обязательные поля: `timestamp`, `level`, `msg`, `command`, `trace_id`

## Data Architecture

Приложение не имеет собственного persistent storage. Взаимодействует с:

- **MSSQL**: Базы данных 1C (restore, structure operations)
- **Gitea**: Конфигурация, репозитории (read via API)
- **1C Storage**: Хранилища конфигураций (через 1cv8/ibcmd)
- **SonarQube**: Метрики качества (read/write via API)

## API Contracts

### Internal Command Interface

```go
type Handler interface {
    // Name returns the command name (e.g., "service-mode-status")
    Name() string

    // Execute runs the command with given configuration
    Execute(ctx context.Context, cfg *config.Config) error
}
```

### Output Writer Interface

```go
type Writer interface {
    // Write outputs the result in the configured format
    Write(w io.Writer, result *Result) error
}
```

### Logger Interface

```go
type Logger interface {
    Debug(msg string, args ...any)
    Info(msg string, args ...any)
    Warn(msg string, args ...any)
    Error(msg string, args ...any)
    With(args ...any) Logger
}
```

## Security Architecture

### Secrets Management

- Secrets только через environment variables
- Маскирование в логах через `slog.LogValuer`:
```go
type Secret string

func (s Secret) LogValue() slog.Value {
    return slog.StringValue("***REDACTED***")
}
```

### Access Control

- Credentials для 1C, Gitea, SonarQube, MSSQL — из protected config
- API endpoints (если будут) — только localhost по умолчанию
- Нет хранения credentials в коде или логах

### Input Validation

- Все внешние входы валидируются
- Пути проверяются на path traversal
- SQL параметры через prepared statements

## Performance Considerations

| Requirement | Strategy |
|-------------|----------|
| Startup < 500ms | Wire compile-time DI, lazy initialization |
| Observability overhead < 5% | Async log/trace export, sampling |
| Long operations | Progress reporting, context cancellation |
| Network resilience | Timeouts, retries with backoff |

## Deployment Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Gitea Actions Runner                     │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                   benadis-runner                      │  │
│  │  BR_COMMAND=xxx BR_INFOBASE_NAME=yyy ./benadis-runner │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────┬───────────────────────────────────────┘
                      │
        ┌─────────────┼─────────────┬─────────────┐
        ▼             ▼             ▼             ▼
   ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐
   │ 1C Srv  │  │  Gitea  │  │ SonarQ  │  │  MSSQL  │
   │ Cluster │  │  Server │  │  Server │  │  Server │
   └─────────┘  └─────────┘  └─────────┘  └─────────┘
```

**Runtime Environment**: Linux (Ubuntu 20.04+, Docker)
**Build Targets**: linux/amd64, windows/amd64, darwin/amd64

## Development Environment

### Prerequisites

- Go 1.25.1+
- Make
- golangci-lint v2
- Wire (`go install github.com/google/wire/cmd/wire@latest`)
- Docker (для интеграционных тестов)
- Доступ к тестовому 1C-серверу (для e2e тестов)

### Setup Commands

```bash
# Установка зависимостей разработки
make setup-dev

# Генерация Wire
go generate ./internal/di/...

# Сборка
make build

# Тесты
make test

# Lint
make lint

# Полная проверка
make check
```

## Architecture Decision Records (ADRs)

### ADR-001: Wire для Dependency Injection

**Status**: Accepted
**Context**: Нужен DI для устранения DIP-нарушений
**Decision**: Wire (compile-time) вместо runtime DI (dig, fx)
**Consequences**: Ошибки на этапе компиляции, нет reflection overhead

### ADR-002: Command Registry вместо Switch

**Status**: Accepted
**Context**: Switch в main.go нарушает OCP
**Decision**: Registry pattern с self-registration через init()
**Consequences**: Новые команды добавляются без изменения main.go

### ADR-003: Role-based Interface Segregation

**Status**: Accepted
**Context**: Широкие интерфейсы нарушают ISP
**Decision**: Разделить на role-based интерфейсы (PRReader, CommitReader, etc.)
**Consequences**: Меньше зависимостей, проще мокирование

### ADR-004: Strategy Pattern для 1C-операций

**Status**: Accepted
**Context**: Нужна возможность переключать реализации (1cv8/ibcmd/native)
**Decision**: Strategy + Factory, выбор через config
**Consequences**: Гибкость без изменения кода

### ADR-005: slog для Structured Logging

**Status**: Accepted
**Context**: Нужен structured logging без внешних зависимостей
**Decision**: slog (stdlib Go 1.21+) с custom Logger interface
**Consequences**: Zero dependencies, JSON output, LSP compliance

---

_Generated by BMAD Decision Architecture Workflow v1.0_
_Date: 2025-11-25_
_For: BMad_
