# Story 1.8: First NR-command: nr-version

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a DevOps-инженер,
I want выполнить nr-version команду,
so that я могу проверить что новая архитектура (Registry + Wire DI + OutputWriter + Logger + TraceID) работает end-to-end.

## Acceptance Criteria

| # | Критерий | Тестируемость |
|---|----------|---------------|
| AC1 | Given BR_COMMAND=nr-version, When приложение запускается, Then stdout содержит версию приложения (constants.Version), Go версию (runtime.Version()), дату сборки (constants.PreCommitHash) | Unit test + Integration test |
| AC2 | Given BR_OUTPUT_FORMAT=json, When nr-version выполняется, Then stdout содержит ТОЛЬКО валидный JSON с полями status, command, data, metadata | Golden file test |
| AC3 | Given BR_OUTPUT_FORMAT=text (или не задан), When nr-version выполняется, Then stdout содержит человекочитаемый формат | Unit test |
| AC4 | Given команда nr-version, When она выполняется, Then команда зарегистрирована через Registry (command.Register в init()) | Registry test |
| AC5 | Given команда nr-version, When она выполняется, Then trace_id присутствует в логах (stderr) и в metadata (stdout JSON) | Integration test |
| AC6 | Given успешное выполнение, When процесс завершается, Then exit code = 0 | Integration test (собранный бинарник) |
| AC7 | Given build info недоступен (не ldflags, не version.go), When nr-version выполняется, Then version="dev", commit="unknown" (fallback) | Unit test |
| AC8 | Given собранный бинарник (не `go run`), When BR_COMMAND=nr-version, Then результат корректен (CI job тестирует бинарник) | CI integration test |
| AC9 | Given stdout, When nr-version выполняется в JSON режиме, Then stdout содержит ТОЛЬКО Result JSON — логи идут ТОЛЬКО в stderr | Integration test |

## Tasks / Subtasks

- [x] **Task 1: Создать handler для nr-version** (AC: 1, 4)
  - [x] 1.1 Создать директорию `internal/command/handlers/version/`
  - [x] 1.2 Создать файл `internal/command/handlers/version/version.go`
  - [x] 1.3 Реализовать struct `VersionHandler` с методами `Name() string` → `"nr-version"` и `Execute(ctx, cfg) error`
  - [x] 1.4 В `init()` вызвать `command.Register(&VersionHandler{})`
  - [x] 1.5 Добавить blank import `_ "github.com/Kargones/apk-ci/internal/command/handlers/version"` в main.go

- [x] **Task 2: Реализовать бизнес-логику nr-version** (AC: 1, 2, 3, 5, 7)
  - [x] 2.1 Определить struct `VersionData` для payload:
    ```go
    type VersionData struct {
        Version   string `json:"version"`    // constants.Version
        GoVersion string `json:"go_version"` // runtime.Version()
        Commit    string `json:"commit"`     // constants.PreCommitHash
    }
    ```
  - [x] 2.2 В `Execute()` собрать `VersionData` из `constants.Version`, `runtime.Version()`, `constants.PreCommitHash`
  - [x] 2.3 Реализовать fallback: если `constants.Version == ""` → version="dev", если `constants.PreCommitHash == ""` → commit="unknown"
  - [x] 2.4 Создать `output.Result` с Status="success", Command="nr-version", Data=VersionData, Metadata с TraceID и DurationMs
  - [x] 2.5 Получить TraceID из context: `tracing.TraceIDFromContext(ctx)`. Если пустой — сгенерировать через `tracing.GenerateTraceID()`
  - [x] 2.6 Вычислить DurationMs (time.Since от начала Execute)
  - [x] 2.7 Получить OutputWriter: `output.NewWriter(os.Getenv("BR_OUTPUT_FORMAT"))` или из DI если доступен
  - [x] 2.8 Вывести через `writer.Write(os.Stdout, &result)`

- [x] **Task 3: Добавить константу команды** (AC: 4)
  - [x] 3.1 Добавить `ActNRVersion = "nr-version"` в `internal/constants/constants.go` (секция действий)
  - [x] 3.2 Использовать константу в handler: `func (h *VersionHandler) Name() string { return constants.ActNRVersion }`

- [x] **Task 4: Написать Unit Tests** (AC: 1, 2, 3, 7)
  - [x] 4.1 Создать `internal/command/handlers/version/version_test.go`
  - [x] 4.2 `TestVersionHandler_Name` — проверка имени "nr-version"
  - [x] 4.3 `TestVersionHandler_Execute_Success` — проверка что Execute возвращает nil
  - [x] 4.4 `TestVersionHandler_Execute_JSONOutput` — при BR_OUTPUT_FORMAT=json stdout содержит валидный JSON с полями version, go_version, commit
  - [x] 4.5 `TestVersionHandler_Execute_TextOutput` — при BR_OUTPUT_FORMAT=text stdout содержит человекочитаемый формат
  - [x] 4.6 `TestVersionHandler_Execute_Metadata` — metadata содержит trace_id, duration_ms, api_version
  - [x] 4.7 `TestVersionHandler_Registration` — проверка что handler зарегистрирован в registry через init()

- [x] **Task 5: Написать Golden File Tests** (AC: 2, 9)
  - [x] 5.1 Создать `internal/command/handlers/version/testdata/` директорию
  - [x] 5.2 Создать golden file `testdata/version_json_output.golden` с ожидаемой структурой JSON
  - [x] 5.3 `TestVersionHandler_GoldenJSON` — сравнение JSON структуры (поля и типы, не значения)
  - [x] 5.4 Проверка что stdout содержит ТОЛЬКО JSON (нет логов, нет лишнего текста)

- [x] **Task 6: Написать Integration Test** (AC: 6, 8, 9)
  - [x] 6.1 Создать `internal/command/handlers/version/integration_test.go`
  - [x] 6.2 `TestVersionHandler_Integration_RegistryAndLegacy` — Registry + legacy switch работают вместе, nr-version идёт через Registry
  - [x] 6.3 `TestVersionHandler_Integration_StdoutStderrSeparation` — логи в stderr, результат в stdout

- [x] **Task 7: Обновить Makefile для CI** (AC: 8)
  - [x] 7.1 Добавить target `test-nr-version` для тестирования собранного бинарника:
    ```makefile
    test-nr-version: build
    	@echo "Тестирование nr-version на собранном бинарнике..."
    	BR_COMMAND=nr-version $(BUILD_DIR)/$(APP_NAME) | jq .
    ```
  - [x] 7.2 Убедиться что `make check` включает тесты handler

### Review Follow-ups (AI)

- [ ] [AI-Review][HIGH] Execute не использует Wire DI — handler обходит DI pipeline, создаёт Writer через os.Getenv [version.go:130-170]
- [ ] [AI-Review][HIGH] os.Getenv("BR_OUTPUT_FORMAT") вызывается в Execute — формат определяется при каждом вызове через env var [version.go:140-141]
- [ ] [AI-Review][MEDIUM] RollbackEntry/RollbackMapping — scope creep из Story 7.4 [version.go:39-50]
- [ ] [AI-Review][MEDIUM] Fallback TraceID генерация может отличаться от TraceID в логах [version.go:136-139]
- [ ] [AI-Review][MEDIUM] Текстовый формат не использует output.Writer — нарушает единообразие [version.go:153]
- [ ] [AI-Review][LOW] dryrun.IsPlanOnly() — scope creep из Story 7.3 [version.go:144-146]

## Dev Notes

### Критический контекст для реализации

**Story 1.8 — это ВАЛИДАЦИЯ всей архитектуры Epic 1.** Это первая NR-команда, проходящая через полный pipeline: Registry → Handler → OutputWriter → Logger + TraceID. Если nr-version работает правильно — архитектура доказала свою работоспособность.

### Зависимости от предыдущих Stories (ВСЕ ЗАВЕРШЕНЫ)

| Story | Статус | Ключевые файлы |
|-------|--------|-----------------|
| 1.1 Command Registry | done | `internal/command/registry.go`, `handler.go` |
| 1.2 DeprecatedBridge | done | `internal/command/deprecated.go` |
| 1.3 OutputWriter | done | `internal/pkg/output/writer.go`, `result.go`, `json.go`, `text.go`, `factory.go` |
| 1.4 Logger interface | done | `internal/pkg/logging/logger.go`, `slog.go`, `factory.go` |
| 1.5 Trace ID | done | `internal/pkg/tracing/traceid.go`, `context.go` |
| 1.6 Config extensions | done | `internal/config/config.go` (ImplementationsConfig, LoggingConfig) |
| 1.7 Wire DI | done | `internal/di/app.go`, `providers.go`, `wire.go`, `wire_gen.go` |

### Архитектурные решения (ОБЯЗАТЕЛЬНО следовать)

**Command Registration Pattern (ADR-002):**
```go
// internal/command/handlers/version/version.go
package version

import (
    "context"
    "github.com/Kargones/apk-ci/internal/command"
    "github.com/Kargones/apk-ci/internal/config"
    "github.com/Kargones/apk-ci/internal/constants"
)

func init() {
    command.Register(&VersionHandler{})
}

type VersionHandler struct{}

func (h *VersionHandler) Name() string {
    return constants.ActNRVersion
}

func (h *VersionHandler) Execute(ctx context.Context, cfg *config.Config) error {
    // ...
}
```

**Blank Import в main.go (ОБЯЗАТЕЛЬНО):**
Чтобы init() сработал, нужен blank import в main.go:
```go
import (
    // NR-команды: blank import для self-registration через init()
    _ "github.com/Kargones/apk-ci/internal/command/handlers/version"
)
```

**Output Result Pattern (из Story 1.3):**
```go
result := &output.Result{
    Status:  output.StatusSuccess,
    Command: constants.ActNRVersion,
    Data:    versionData,
    Metadata: &output.Metadata{
        DurationMs: time.Since(start).Milliseconds(),
        TraceID:    traceID,
        APIVersion: constants.APIVersion, // "v1"
    },
}
writer := output.NewWriter(os.Getenv("BR_OUTPUT_FORMAT"))
return writer.Write(os.Stdout, result)
```

**Версия приложения — через constants.Version (НЕ ldflags):**
В проекте версия генерируется через `scripts/generate-version.sh` в файл `internal/constants/version.go`. Это НЕ ldflags. Поэтому:
- `constants.Version` — полная версия (формат: `6.1.21.7:d8a0311-debug`)
- `constants.PreCommitHash` — короткий хеш коммита (`d8a0311`)
- `constants.DebugSuffix` — суффикс отладки (`-debug` или `""`)
- Go версия — через `runtime.Version()` (например, `go1.25.1`)

**Handler НЕ использует Wire DI напрямую:**
Handler получает `*config.Config` через параметр Execute. OutputWriter создаётся через factory `output.NewWriter()`. Wire DI используется в main.go для инициализации App struct, но handler пока работает автономно. В будущих stories (Epic 2+) handlers будут получать DI-зависимости.

### Data Structures

**VersionData (новый struct для nr-version):**
```go
// VersionData содержит информацию о версии приложения.
type VersionData struct {
    // Version — полная версия приложения.
    Version string `json:"version"`

    // GoVersion — версия Go, использованная при сборке.
    GoVersion string `json:"go_version"`

    // Commit — хеш коммита на момент сборки.
    Commit string `json:"commit"`
}
```

**Ожидаемый JSON output:**
```json
{
  "status": "success",
  "command": "nr-version",
  "data": {
    "version": "6.1.21.7:d8a0311-debug",
    "go_version": "go1.25.1",
    "commit": "d8a0311"
  },
  "metadata": {
    "duration_ms": 1,
    "trace_id": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4",
    "api_version": "v1"
  }
}
```

**Ожидаемый Text output:**
```
apk-ci version 6.1.21.7:d8a0311-debug
  Go:     go1.25.1
  Commit: d8a0311
```

### main.go Integration

**Текущее поведение main.go (строки 32-44):**
main.go уже проверяет Registry перед legacy switch. Поэтому:
1. Handler регистрируется в init() → попадает в registry
2. main.go вызывает `command.Get("nr-version")` → находит handler
3. Вызывает `handler.Execute(ctx, cfg)` → nr-version работает
4. Никаких изменений в legacy switch НЕ нужно

**ЕДИНСТВЕННОЕ изменение в main.go:** добавить blank import.

### Pre-mortem Failure Modes

| FM | Failure Mode | Митигация |
|----|--------------|-----------|
| FM5 | `go run` работает, собранный бинарник — нет | CI job с `make test-nr-version` тестирует собранный бинарник |
| FM9 | Логи попадают в stdout вместе с JSON | Logger пишет в stderr (уже реализовано в Story 1.4). Тест: stderr vs stdout separation |
| FM10 | Blank import забыт → handler не регистрируется | Integration test проверяет registration |
| FM11 | TraceID не попадает в metadata | Unit test проверяет metadata.trace_id != "" |

### Риски и митигации

| ID | Риск | Probability | Impact | Митигация |
|----|------|-------------|--------|-----------|
| R1 | version.go содержит пустые константы при go run | Low | Medium | Fallback значения "dev"/"unknown" |
| R2 | TextWriter не поддерживает VersionData struct | Medium | Low | TextWriter использует fmt.Sprintf для any data |
| R3 | main.go blank import не подхватывается при vendor | Low | Low | `go mod vendor` включает imported packages |

### Project Structure Notes

**Новые файлы (создаются в этой story):**
```
internal/command/handlers/version/
├── version.go            # Handler + VersionData + init()
├── version_test.go       # Unit tests
├── integration_test.go   # Integration tests
└── testdata/
    └── version_json_output.golden  # Golden file для JSON
```

**Изменяемые файлы:**
```
cmd/apk-ci/main.go           # Добавить blank import
internal/constants/constants.go      # Добавить ActNRVersion
Makefile                             # Добавить target test-nr-version
```

### Testing Standards

- Framework: `testify/assert`, `testify/require`
- Pattern: Table-driven tests где применимо
- Naming: `Test{FunctionName}_{Scenario}`
- Run: `go test ./internal/command/handlers/version/... -v`
- Golden tests: `testdata/` директория, `go test -update` для обновления

### Git Intelligence (последние коммиты)

```
1da876c chore: update sprint status and documentation for trace id generation
8677915 test(tracing): add example tests for trace ID utilities
0e61554 chore: update sprint status for story 1-7 to done
4ad59b0 fix(di): add CI workflow and fix documentation
213a29f feat(di): set up wire dependency injection for application components
a85b11e feat(config): add implementations config and update logging defaults
c4d5e08 feat(tracing): implement trace ID generation and context integration
ecd4f8d feat(logging): implement structured logging interface with slog adapter
```

**Паттерны из предыдущих коммитов:**
- Conventional commits: `feat(scope): description`
- Factory functions для создания компонентов
- Unit tests в `*_test.go` рядом с реализацией
- godoc комментарии на русском языке
- Рекомендуемый commit: `feat(version): implement nr-version command for architecture validation`

### Предыдущая Story Intelligence (Story 1.7)

**Ключевые уроки из Story 1.7:**
- Wire DI работает и генерирует `wire_gen.go` корректно
- App struct содержит: Config, Logger, OutputWriter, TraceID
- Провайдеры: `ProvideLogger`, `ProvideOutputWriter`, `ProvideTraceID`
- CI workflow создан в `.gitea/workflows/ci.yaml`
- `make generate-wire` и `make check-wire` работают

**Что использовать из Story 1.7:**
- `output.NewWriter(format)` — factory для получения Writer
- `tracing.GenerateTraceID()` — генерация trace ID
- `tracing.TraceIDFromContext(ctx)` — извлечение trace ID из context

### Связь со следующими Stories

**Story 1.9 (Auto-generated Help):**
- Использует `command.All()` для получения всех зарегистрированных команд
- nr-version будет отображаться в списке help
- Handler interface может быть расширен: `Description() string` (опционально)

**Epic 2+ (NR-команды):**
- nr-version — эталонный паттерн для всех будущих NR-команд
- Каждая NR-команда будет следовать тому же шаблону: handler struct → init() → Register → Execute

### References

- [Source: _bmad-output/project-planning-artifacts/epics/epic-1-foundation.md#Story 1.8] — Epic description
- [Source: _bmad-output/project-planning-artifacts/architecture.md#Command Registry] — Registry pattern
- [Source: _bmad-output/project-planning-artifacts/architecture.md#Output Writer Interface] — Output contract
- [Source: internal/command/handler.go] — Handler interface
- [Source: internal/command/registry.go] — Registry implementation
- [Source: internal/pkg/output/result.go] — Result struct
- [Source: internal/pkg/output/factory.go] — Output factory
- [Source: internal/constants/version.go] — Version constants
- [Source: internal/constants/constants.go] — Command constants
- [Source: cmd/apk-ci/main.go:32-44] — Registry integration в main
- [Source: internal/di/app.go] — App struct с DI dependencies
- [Source: scripts/generate-version.sh] — Version generation script

### Review Follow-ups (AI Code Review #34)

- [ ] [AI-Review][HIGH] Execute не использует Wire DI — обходит DI pipeline, создаёт Writer через os.Getenv [version/version.go:130-170]
- [ ] [AI-Review][HIGH] os.Getenv("BR_OUTPUT_FORMAT") вызывается в Execute вместо получения из Config [version/version.go:141]
- [ ] [AI-Review][MEDIUM] RollbackMapping — scope creep из Story 7.4 (не должно быть в Story 1.8) [version/version.go:38-50]
- [ ] [AI-Review][MEDIUM] Текстовый формат не использует output.Writer — нарушает единообразие [version/version.go:152-154]
- [ ] [AI-Review][MEDIUM] Fallback TraceID генерация может отличаться от TraceID в логах [version/version.go:136-139]

## Dev Agent Record

### Agent Model Used

Claude Opus 4.5 (claude-opus-4-5-20251101)

### Debug Log References

- Полная регрессия тестов: 12/12 тестов nr-version PASS, предсуществующий FAIL в `internal/app/TestExtensionPublish_MissingEnvVars` (DNS resolve, не связан с изменениями)

### Completion Notes List

- Реализована первая NR-команда `nr-version`, проходящая через полный pipeline: Registry → Handler → OutputWriter + TraceID
- Handler регистрируется через `init()` + `command.Register()`, main.go подключает через blank import
- Поддерживаются оба формата вывода: JSON и Text
- Fallback значения для Version ("dev") и Commit ("unknown") реализованы
- TraceID извлекается из context или генерируется автоматически
- 12 тестов: 8 unit, 2 golden file, 2 integration — все PASS
- Makefile target `test-nr-version` добавлен для CI тестирования собранного бинарника

### Change Log

- 2026-01-27: Реализована Story 1.8 — первая NR-команда nr-version для end-to-end валидации архитектуры Epic 1
- 2026-01-27: Code Review — исправлены 6 проблем (3 HIGH, 3 MEDIUM): текстовый формат приведён в соответствие со спецификацией, fallback-логика извлечена в тестируемую функцию buildVersionData, тесты усилены (captureStdout helper, проверка pipe ошибок, табличные тесты fallback), Makefile target избавлен от зависимости на jq
- 2026-01-27: Code Review #2 — исправлены 7 проблем (3 HIGH, 4 MEDIUM): captureStdout получил defer-восстановление stdout (H1/M4), Makefile test-nr-version добавлена JSON-валидация и предупреждение о Gitea API (H2), документирован выбор текстового формата без metadata (H3/M2/M3), golden тест усилен проверкой типов и отсутствия лишних полей (M1)

### File List

**Новые файлы:**
- `internal/command/handlers/version/version.go` — Handler + VersionData + init()
- `internal/command/handlers/version/version_test.go` — Unit тесты (8 тестов)
- `internal/command/handlers/version/golden_test.go` — Golden file тесты (2 теста)
- `internal/command/handlers/version/integration_test.go` — Integration тесты (2 теста)
- `internal/command/handlers/version/testdata/version_json_output.golden` — Golden file для JSON структуры

**Изменённые файлы:**
- `cmd/apk-ci/main.go` — Добавлен blank import для self-registration
- `internal/constants/constants.go` — Добавлена константа ActNRVersion
- `Makefile` — Добавлен target test-nr-version
