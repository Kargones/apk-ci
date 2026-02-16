# Story 1.9: Auto-generated help from Registry

Status: done

## Story

As a DevOps-инженер,
I want видеть список всех доступных команд при BR_COMMAND=help или пустом BR_COMMAND,
so that я знаю какие команды существуют без чтения документации.

## Acceptance Criteria

| # | Критерий | Тестируемость |
|---|----------|---------------|
| AC1 | Given BR_COMMAND=help, When приложение запускается, Then stdout содержит список всех зарегистрированных команд с описаниями | Unit test + Integration test |
| AC2 | Given BR_COMMAND="" (пустой), When приложение запускается, Then выводится help (аналогично BR_COMMAND=help) | Integration test |
| AC3 | Given deprecated команда зарегистрирована через RegisterWithAlias, When help выводится, Then deprecated команда помечена как [deprecated] с указанием нового имени | Unit test |
| AC4 | Given NR-команды и legacy-команды, When help выводится, Then NR-команды (prefix "nr-") группируются отдельно от legacy-команд | Unit test |
| AC5 | Given BR_OUTPUT_FORMAT=json, When help выполняется, Then stdout содержит ТОЛЬКО валидный JSON с Result структурой (status, command, data, metadata) | Golden file test |
| AC6 | Given BR_OUTPUT_FORMAT=text (или не задан), When help выполняется, Then stdout содержит человекочитаемый формат с группировкой | Unit test |
| AC7 | Given help команда, When она выполняется, Then команда зарегистрирована через Registry (command.Register в init()) | Registry test |
| AC8 | Given help команда, When она выполняется, Then trace_id присутствует в логах (stderr) и в metadata (stdout JSON) | Integration test |

## Tasks / Subtasks

- [x] **Task 1: Расширить Handler interface — добавить Description()** (AC: 1, 3)
  - [x] 1.1 Добавить метод `Description() string` в interface Handler (`internal/command/handler.go`)
  - [x] 1.2 Обновить `VersionHandler` — добавить `Description() string` → `"Вывод информации о версии приложения"`
  - [x] 1.3 Обновить `DeprecatedBridge` — делегировать `Description()` на `actual.Description()`
  - [x] 1.4 Убедиться что компиляция проходит — все реализации Handler обновлены

- [x] **Task 2: Добавить метод IsDeprecated в Registry** (AC: 3)
  - [x] 2.1 Добавить функцию `IsDeprecated(name string) (bool, string)` в `internal/command/registry.go` — возвращает true + newName если handler является DeprecatedBridge
  - [x] 2.2 Альтернативный подход: type assertion `handler.(*DeprecatedBridge)` или опциональный interface `Deprecatable`

- [x] **Task 3: Создать handler для help** (AC: 1, 7)
  - [x] 3.1 Создать файл `internal/command/handlers/help/help.go`
  - [x] 3.2 Реализовать struct `HelpHandler` с методами `Name()` → `"help"`, `Description()` → `"Вывод списка доступных команд"`, `Execute(ctx, cfg)`
  - [x] 3.3 В `init()` вызвать `command.Register(&HelpHandler{})`
  - [x] 3.4 Добавить blank import `_ ".../internal/command/handlers/help"` в main.go
  - [x] 3.5 Добавить константу `ActHelp = "help"` в `internal/constants/constants.go`

- [x] **Task 4: Реализовать бизнес-логику help** (AC: 1, 3, 4, 5, 6, 8)
  - [x] 4.1 Определить struct `HelpData` для payload:
    ```go
    type HelpData struct {
        NRCommands    []CommandInfo `json:"nr_commands"`
        LegacyCommands []CommandInfo `json:"legacy_commands"`
    }
    type CommandInfo struct {
        Name        string `json:"name"`
        Description string `json:"description"`
        Deprecated  bool   `json:"deprecated,omitempty"`
        NewName     string `json:"new_name,omitempty"`
    }
    ```
  - [x] 4.2 В `Execute()` вызвать `command.All()` для получения всех handlers
  - [x] 4.3 Для каждого handler: получить Name(), Description(), проверить deprecated статус
  - [x] 4.4 Разделить на NR-команды (prefix `nr-`) и legacy (остальные)
  - [x] 4.5 Сортировать по алфавиту внутри каждой группы
  - [x] 4.6 Для text формата: вывести через специализированный writeText (аналогично version handler)
  - [x] 4.7 Для JSON формата: вывести через output.Result + output.Writer

- [x] **Task 5: Обработать пустой BR_COMMAND** (AC: 2)
  - [x] 5.1 В main.go: если `cfg.Command == ""` → установить `cfg.Command = "help"` перед проверкой Registry
  - [x] 5.2 Альтернативно: добавить проверку пустой команды перед Registry lookup

- [x] **Task 6: Включить legacy-команды в help** (AC: 4)
  - [x] 6.1 Определить список legacy-команд с описаниями как map или slice в help handler
  - [x] 6.2 Источник: константы из `internal/constants/constants.go` (ActConvert, ActDbrestore и т.д.)
  - [x] 6.3 Legacy-команды не в registry — описания захардкодить в help handler

- [x] **Task 7: Написать Unit Tests** (AC: 1, 3, 4, 5, 6)
  - [x] 7.1 Создать `internal/command/handlers/help/help_test.go`
  - [x] 7.2 `TestHelpHandler_Name` — проверка имени "help"
  - [x] 7.3 `TestHelpHandler_Description` — проверка описания
  - [x] 7.4 `TestHelpHandler_Execute_TextOutput` — проверка текстового вывода с группировкой
  - [x] 7.5 `TestHelpHandler_Execute_JSONOutput` — проверка JSON вывода с HelpData
  - [x] 7.6 `TestHelpHandler_DeprecatedMarking` — deprecated команды помечены [deprecated]
  - [x] 7.7 `TestHelpHandler_NRGrouping` — NR-команды отдельно от legacy
  - [x] 7.8 `TestHelpHandler_Sorting` — команды отсортированы по алфавиту

- [x] **Task 8: Написать Golden File Tests** (AC: 5)
  - [x] 8.1 Создать `internal/command/handlers/help/testdata/` директорию
  - [x] 8.2 Golden file `testdata/help_json_output.golden` с JSON структурой
  - [x] 8.3 `TestHelpHandler_GoldenJSON` — сравнение JSON структуры

- [x] **Task 9: Написать Integration Tests** (AC: 2, 7, 8)
  - [x] 9.1 Создать `internal/command/handlers/help/integration_test.go`
  - [x] 9.2 `TestHelpHandler_Integration_EmptyCommand` — пустой BR_COMMAND → help
  - [x] 9.3 `TestHelpHandler_Integration_Registration` — help зарегистрирован в registry
  - [x] 9.4 `TestHelpHandler_Integration_StdoutStderrSeparation` — логи в stderr, результат в stdout

### Review Follow-ups (AI)

- [ ] [AI-Review][HIGH] legacyCommands — захардкоженный map, может рассинхронизироваться при изменениях [help.go:48-68]
- [ ] [AI-Review][HIGH] Execute не использует Wire DI — handler обходит DI [help.go:84-121]
- [ ] [AI-Review][MEDIUM] buildData() включает deprecated bridges в NR-команды — может сбивать с толку [help.go:128-158]
- [ ] [AI-Review][MEDIUM] writeText использует strings.Builder vs fmt.Fprintf в version.go — несогласованность стиля [help.go:162-203]
- [ ] [AI-Review][MEDIUM] Опции BR_DRY_RUN, BR_PLAN_ONLY, BR_VERBOSE, BR_SHADOW_RUN — scope creep из Epic 7 [help.go:196-199]
- [ ] [AI-Review][LOW] help.Handler stuttering при использовании — было бы лучше help.Command [help.go:71]
- [ ] [AI-Review][LOW] init() без recover — при panic из-за дублирования нет информативного сообщения [help.go:22-24]

## Dev Notes

### Критический контекст для реализации

**Story 1.9 — завершающая story Epic 1.** Она добавляет discoverability ко всем зарегистрированным командам. После этой story пользователь может узнать обо всех доступных командах без документации.

### КРИТИЧЕСКОЕ ИЗМЕНЕНИЕ: Расширение Handler interface

**Handler interface** в `internal/command/handler.go:15-24` сейчас содержит только:
```go
type Handler interface {
    Name() string
    Execute(ctx context.Context, cfg *config.Config) error
}
```

**Нужно добавить `Description() string`:**
```go
type Handler interface {
    Name() string
    Description() string
    Execute(ctx context.Context, cfg *config.Config) error
}
```

**ПОСЛЕДСТВИЯ:**
- `VersionHandler` (`internal/command/handlers/version/version.go`) — добавить Description()
- `DeprecatedBridge` (`internal/command/deprecated.go`) — добавить Description(), делегировать actual.Description()
- Все тесты, использующие mock handlers — обновить

**ВНИМАНИЕ:** Tech-spec (Epic 1) определяет `Description()` как обязательный метод Handler. Это запланированное изменение, не ломающее runtime — все handlers под нашим контролем.

### Зависимости от предыдущих Stories

| Story | Статус | Ключевые файлы для этой Story |
|-------|--------|-------------------------------|
| 1.1 Command Registry | done | `internal/command/registry.go` — `All()`, `Names()` |
| 1.2 DeprecatedBridge | done | `internal/command/deprecated.go` — нужно определить deprecated |
| 1.3 OutputWriter | done | `internal/pkg/output/` — Result, Writer, factory |
| 1.8 nr-version | done | `internal/command/handlers/version/` — паттерн handler |

### Архитектурные решения (ОБЯЗАТЕЛЬНО следовать)

**Command Registration Pattern (тот же что nr-version, ADR-002):**
```go
// internal/command/handlers/help/help.go
package help

import (
    "context"
    "github.com/Kargones/apk-ci/internal/command"
    "github.com/Kargones/apk-ci/internal/config"
    "github.com/Kargones/apk-ci/internal/constants"
)

func init() {
    command.Register(&HelpHandler{})
}

type HelpHandler struct{}

func (h *HelpHandler) Name() string        { return constants.ActHelp }
func (h *HelpHandler) Description() string { return "Вывод списка доступных команд" }
func (h *HelpHandler) Execute(ctx context.Context, cfg *config.Config) error {
    // ...
}
```

**Blank Import в main.go (ОБЯЗАТЕЛЬНО):**
```go
import (
    _ "github.com/Kargones/apk-ci/internal/command/handlers/help"
    _ "github.com/Kargones/apk-ci/internal/command/handlers/version"
)
```

**Output Pattern (из Story 1.8):**
```go
result := &output.Result{
    Status:  output.StatusSuccess,
    Command: constants.ActHelp,
    Data:    helpData,
    Metadata: &output.Metadata{
        DurationMs: time.Since(start).Milliseconds(),
        TraceID:    traceID,
        APIVersion: constants.APIVersion,
    },
}
writer := output.NewWriter(os.Getenv("BR_OUTPUT_FORMAT"))
return writer.Write(os.Stdout, result)
```

### Определение deprecated статуса

`DeprecatedBridge` — приватный struct в package `command`. Способы определения:
1. **Type assertion** (если добавить экспортируемый метод): `bridge, ok := handler.(*DeprecatedBridge)` — НЕ работает, struct неэкспортируемый
2. **Опциональный interface:** Добавить в registry.go:
   ```go
   // Deprecatable опционально реализуется deprecated handlers.
   type Deprecatable interface {
       IsDeprecated() bool
       NewName() string
   }
   ```
   DeprecatedBridge реализует Deprecatable. Help handler проверяет через type assertion.
3. **Registry metadata:** Хранить deprecated info в registry — усложняет API.

**Рекомендация:** Вариант 2 (опциональный interface Deprecatable) — наиболее чистый, ISP-совместимый подход.

### Legacy-команды в help

Legacy-команды (constants.ActConvert, ActDbrestore и т.д.) НЕ зарегистрированы в Registry. Для полного help их нужно включить с описаниями. Варианты:
1. **Захардкодить в help handler** — просто, но нужно обновлять при добавлении legacy команд
2. **Определить map в constants** — централизованно, но засоряет constants package

**Рекомендация:** Захардкодить в help handler как `legacyCommands` map — это временное решение, legacy-команды будут мигрированы в NR-формат и зарегистрированы в Registry.

### Пустая команда (BR_COMMAND="")

Текущее поведение в main.go при пустом Command:
1. `command.Get("")` → (nil, false) — пустая строка не найдена в registry
2. Legacy switch → `default` → "неизвестная команда" → os.Exit(2)

**Нужно:** перехватить пустой Command ДО registry lookup и перенаправить на "help". Место: main.go, перед строкой 36 (Registry check).

```go
// Пустая команда → help
if cfg.Command == "" {
    cfg.Command = "help"
}
```

### Ожидаемый текстовый вывод

```
benadis-runner — инструмент автоматизации 1C:Enterprise

NR-команды:
  help          Вывод списка доступных команд
  nr-version    Вывод информации о версии приложения

Legacy-команды:
  convert                Конвертация форматов данных
  create-stores          Создание хранилищ конфигурации
  create-temp-db         Создание временной базы данных
  dbrestore              Восстановление базы данных
  dbupdate               Обновление базы данных
  execute-epf            Выполнение внешней обработки
  extension-publish      Публикация расширения 1C
  git2store              Синхронизация Git → хранилище 1C
  service-mode-disable   Отключение сервисного режима
  service-mode-enable    Включение сервисного режима
  service-mode-status    Статус сервисного режима
  sq-project-update      Обновление проекта SonarQube
  sq-report-branch       Отчёт по ветке SonarQube
  sq-scan-branch         Сканирование ветки SonarQube
  sq-scan-pr             Сканирование PR SonarQube
  store2db               Загрузка конфигурации из хранилища
  storebind              Привязка хранилища к базе
  test-merge             Проверка конфликтов слияния

Используйте BR_OUTPUT_FORMAT=json для машиночитаемого вывода.
```

### Ожидаемый JSON output

```json
{
  "status": "success",
  "command": "help",
  "data": {
    "nr_commands": [
      {"name": "help", "description": "Вывод списка доступных команд"},
      {"name": "nr-version", "description": "Вывод информации о версии приложения"}
    ],
    "legacy_commands": [
      {"name": "convert", "description": "Конвертация форматов данных"},
      ...
    ]
  },
  "metadata": {
    "duration_ms": 0,
    "trace_id": "...",
    "api_version": "v1"
  }
}
```

### Data Structures

**HelpData (новый struct для help):**
```go
// HelpData содержит информацию обо всех доступных командах.
type HelpData struct {
    // NRCommands — команды нового формата (зарегистрированные в Registry).
    NRCommands []CommandInfo `json:"nr_commands"`
    // LegacyCommands — legacy-команды (обрабатываются через switch в main.go).
    LegacyCommands []CommandInfo `json:"legacy_commands"`
}

// CommandInfo описывает одну команду.
type CommandInfo struct {
    // Name — имя команды.
    Name string `json:"name"`
    // Description — описание команды.
    Description string `json:"description"`
    // Deprecated — true если команда deprecated.
    Deprecated bool `json:"deprecated,omitempty"`
    // NewName — новое имя команды (если deprecated).
    NewName string `json:"new_name,omitempty"`
}
```

### Pre-mortem Failure Modes

| FM | Failure Mode | Митигация |
|----|--------------|-----------|
| FM1 | Description() добавлен в interface, но забыт в mock-handler тестов | Компилятор поймает — interface compliance |
| FM2 | Legacy-команды в help устарели относительно main.go switch | Добавить тест: все константы Act* из constants.go представлены в help |
| FM3 | Help сама себя не показывает в списке | Help зарегистрирована через Registry → автоматически в All() |
| FM4 | Пустая команда не перехватывается | Integration test: BR_COMMAND="" → help output |
| FM5 | Deprecated bridge не определяется | Опциональный interface Deprecatable с тестами |

### Project Structure Notes

**Новые файлы:**
```
internal/command/handlers/help/
├── help.go              # Handler + HelpData + init()
├── help_test.go         # Unit tests
├── integration_test.go  # Integration tests
└── testdata/
    └── help_json_output.golden  # Golden file для JSON
```

**Изменяемые файлы:**
```
internal/command/handler.go             # Добавить Description() в interface
internal/command/deprecated.go          # Добавить Description() + Deprecatable interface
internal/command/registry.go            # (опционально) добавить Deprecatable interface
internal/command/handlers/version/version.go  # Добавить Description()
internal/constants/constants.go         # Добавить ActHelp
cmd/benadis-runner/main.go             # Blank import + пустая команда → help
```

### Testing Standards

- Framework: `testify/assert`, `testify/require`
- Pattern: Table-driven tests
- Naming: `Test{FunctionName}_{Scenario}`
- Run: `go test ./internal/command/handlers/help/... -v`
- Golden tests: `testdata/` директория
- Проверить что все legacy-команды из constants.go присутствуют в help

### Git Intelligence (последние коммиты)

```
1da876c chore: update sprint status and documentation for trace id generation
8677915 test(tracing): add example tests for trace ID utilities
0e61554 chore: update sprint status for story 1-7 to done
4ad59b0 fix(di): add CI workflow and fix documentation
0ddc00b chore: update story statuses and go.sum dependencies
```

**Паттерны:**
- Conventional commits: `feat(scope): description`
- Factory functions для создания компонентов
- Unit tests в `*_test.go` рядом с реализацией
- godoc комментарии на русском языке
- Рекомендуемый commit: `feat(help): implement auto-generated help from command registry`

### Предыдущая Story Intelligence (Story 1.8)

**Ключевые уроки из Story 1.8:**
- Handler паттерн: struct → init() → Register → Execute → output.Result → writer.Write
- Текстовый формат может быть специализированным (writeText), не обязательно через output.Writer
- `captureStdout` helper для тестирования stdout output
- Golden tests проверяют структуру JSON, не конкретные значения
- `buildVersionData` — отдельная тестируемая функция для подготовки данных

**Что использовать из Story 1.8:**
- Тот же паттерн handler registration и execution
- `output.NewWriter(format)` для JSON вывода
- `tracing.GenerateTraceID()` и `tracing.TraceIDFromContext(ctx)` для trace ID
- Аналогичная структура тестов: unit + golden + integration

### Связь с Epic 2+

После Story 1.9 каждая новая NR-команда автоматически появляется в help (через Registry). Это значит:
- Epic 2 (service-mode) команды будут видны в help сразу после регистрации
- Legacy-команды будут постепенно мигрированы и помечены [deprecated]

### References

- [Source: _bmad-output/project-planning-artifacts/epics/epic-1-foundation.md#Story 1.9] — Epic description
- [Source: _bmad-output/implementation-artifacts/sprint-artifacts/tech-spec-epic-1.md#AC10] — AC10: Help из Registry
- [Source: internal/command/handler.go] — Текущий Handler interface (БЕЗ Description)
- [Source: internal/command/registry.go] — Registry: All(), Names(), RegisterWithAlias()
- [Source: internal/command/deprecated.go] — DeprecatedBridge struct
- [Source: internal/command/handlers/version/version.go] — Эталонный handler паттерн
- [Source: internal/pkg/output/result.go] — Result struct
- [Source: internal/pkg/output/factory.go] — Output factory
- [Source: internal/constants/constants.go:49-104] — Все Act* константы для legacy-команд
- [Source: cmd/benadis-runner/main.go:35-47] — Registry integration
- [Source: cmd/benadis-runner/main.go:50-280] — Legacy switch (все case для legacy-команд)

### Review Follow-ups (AI Code Review #34)

- [ ] [AI-Review][HIGH] legacyCommands — захардкоженный map, может рассинхронизироваться с constants.go [help/help.go:48-68]
- [ ] [AI-Review][HIGH] Execute не использует Wire DI [help/help.go:84-121]
- [ ] [AI-Review][MEDIUM] buildData() включает deprecated bridges в NR-команды — может сбивать с толку [help/help.go:128-158]
- [ ] [AI-Review][MEDIUM] Несогласованность стиля: strings.Builder vs fmt.Fprintf между help и version handlers [help/help.go:162-203]

## Change Log

- 2026-01-27: Реализована команда help с автоматической генерацией списка команд из Registry. Расширен Handler interface (добавлен Description()). Добавлен Deprecatable interface для определения deprecated статуса. Обработка пустого BR_COMMAND → help.
- 2026-01-27: **Code Review (AI):** Исправлены 4 issue: (H1) усилен тест LegacyCommandsCompleteness — явно учтены все Act* константы включая неактивные; (M1) golden test сравнивает структуру с golden file; (M3) устранён stuttering: HelpData→Data, HelpHandler→Handler; (M4) добавлена compile-time проверка Deprecatable interface.
- 2026-01-27: **Code Review #2 (AI):** Исправлены 4 issue: (H1) тест LegacyCommandsCompleteness — устранена тавтологическая проверка, добавлена реальная валидация размера legacyCommands map и обратная проверка membership; (M1) документирован в комментарии — text формат не включает trace_id/metadata; (M2) добавлен integration тест TestHelpHandler_Integration_DeprecatedInBuildData — регистрирует deprecated handler через RegisterWithAlias и проверяет buildData(); (M3) выделен shared testutil.CaptureStdout — устранено дублирование между help и version тестами.

## Dev Agent Record

### Agent Model Used

Claude Opus 4.5 (claude-opus-4-5-20251101)

### Debug Log References

- Все тесты command пакетов проходят (17 новых тестов help + все существующие)
- Единственный pre-existing failure: TestExtensionPublish_MissingEnvVars в internal/app (сетевой тест, не связан с изменениями)
- Lint: новых проблем в изменённых файлах нет (revive stuttering — pre-existing паттерн, аналогичен version пакету)

### Completion Notes List

- Task 1: Добавлен `Description() string` в Handler interface. Обновлены VersionHandler, DeprecatedBridge и все тестовые mock-структуры (compileTimeHandler, mockHandler, testDeprecatedHandler)
- Task 2: Реализован опциональный interface `Deprecatable` с методами `IsDeprecated() bool` и `NewName() string` в deprecated.go. DeprecatedBridge реализует Deprecatable
- Task 3: Создан help handler с self-registration через init(). Добавлен blank import в main.go и константа ActHelp
- Task 4: Полная бизнес-логика: buildHelpData() собирает NR-команды из Registry и legacy-команды из захардкоженного map. Поддержка text и JSON форматов. Deprecated marking через Deprecatable interface
- Task 5: В main.go добавлена проверка `cfg.Command == ""` → перенаправление на "help"
- Task 6: Legacy-команды определены как map[string]string в help handler (19 команд с описаниями из constants)
- Task 7: 13 unit тестов — Name, Description, TextOutput, JSONOutput, DeprecatedMarking, NRGrouping, Sorting, Registration, LegacyCommandsCompleteness, BuildHelpData, WriteText, WriteText_Deprecated
- Task 8: Golden file test с проверкой JSON структуры (ключи и типы, не конкретные значения)
- Task 9: 4 integration теста — EmptyCommand, Registration, StdoutStderrSeparation, TraceID

### File List

**Новые файлы:**
- internal/command/handlers/help/help.go
- internal/command/handlers/help/help_test.go
- internal/command/handlers/help/golden_test.go
- internal/command/handlers/help/integration_test.go
- internal/command/handlers/help/testdata/help_json_output.golden

- internal/pkg/testutil/capture.go

**Изменённые файлы:**
- internal/command/handler.go — добавлен Description() в Handler interface
- internal/command/deprecated.go — добавлены Description(), Deprecatable interface, IsDeprecated(), NewName()
- internal/command/handler_test.go — добавлен Description() в compileTimeHandler
- internal/command/registry_test.go — добавлен Description() в mockHandler
- internal/command/deprecated_test.go — добавлен Description() в testDeprecatedHandler
- internal/command/handlers/version/version.go — добавлен Description()
- internal/constants/constants.go — добавлен ActHelp
- cmd/benadis-runner/main.go — blank import help + пустая команда → help
- internal/command/handlers/version/version_test.go — замена captureStdout на testutil.CaptureStdout
- internal/command/handlers/version/golden_test.go — замена captureStdout на testutil.CaptureStdout
- internal/command/handlers/version/integration_test.go — замена captureStdout на testutil.CaptureStdout
- _bmad-output/implementation-artifacts/sprint-artifacts/sprint-status.yaml — статус 1-9 обновлён
