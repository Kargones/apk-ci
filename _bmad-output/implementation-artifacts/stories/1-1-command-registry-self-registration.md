# Story 1.1: Command Registry с self-registration

Status: done

## Story

As a разработчик,
I want регистрировать новые команды через init() без изменения main.go,
so that добавление команд соответствует Open/Closed Principle.

## Acceptance Criteria

| # | Критерий | Тестируемость |
|---|----------|---------------|
| AC1 | Given новый handler реализующий interface Handler, When handler вызывает `command.Register()` в init(), Then команда доступна через `command.Get(name)` | Unit test: регистрация + получение |
| AC2 | Registry возвращает `(nil, false)` для несуществующих команд | Unit test: Get для unknown |
| AC3 | Повторная регистрация с тем же именем вызывает panic (programming error) | Unit test: двойная регистрация |
| AC4 | main.go: сначала проверяет Registry через `command.Get()`, потом fallback на legacy switch | Integration test: оба пути |
| AC5 | Логируется какой путь выбран ("registry" или "legacy") для диагностики | Log inspection test |

## Tasks / Subtasks

- [x] **Task 1: Создать Handler interface** (AC: 1)
  - [x] 1.1 Создать директорию `internal/command/`
  - [x] 1.2 Создать файл `internal/command/handler.go`
  - [x] 1.3 Определить интерфейс Handler с методами `Name() string` и `Execute(ctx context.Context, cfg *config.Config) error`
  - [x] 1.4 Добавить godoc комментарии на русском языке

- [x] **Task 2: Реализовать Command Registry** (AC: 1, 2, 3)
  - [x] 2.1 Создать файл `internal/command/registry.go`
  - [x] 2.2 Реализовать `var registry = make(map[string]Handler)` с sync.RWMutex
  - [x] 2.3 Реализовать `func Register(h Handler)` с panic при дублировании
  - [x] 2.4 Добавить валидацию: panic при nil handler или empty name
  - [x] 2.5 Реализовать `func Get(name string) (Handler, bool)`
  - [x] 2.6 Добавить unexported `clearRegistry()` для тестов
  - [x] 2.7 Написать unit tests для Register и Get (включая nil/empty/race tests)

- [x] **Task 3: Интегрировать Registry в main.go** (AC: 4, 5)
  - [x] 3.1 Добавить import `"github.com/Kargones/apk-ci/internal/command"`
  - [x] 3.2 Добавить blank import для триггера init(): `import _ ".../internal/command/handlers"` (когда появятся handlers) — N/A, будет добавлен в Story 1.8
  - [x] 3.3 Добавить проверку Registry ПЕРЕД legacy switch (строка 30)
  - [x] 3.4 Реализовать fallback логику на legacy switch если Registry не нашёл команду
  - [x] 3.5 Добавить логирование выбранного пути ("registry" / "legacy") через cfg.Logger
  - [x] 3.6 Написать integration test для обоих путей
  - [x] 3.7 Написать test TestRegistryNotEmpty_AfterImport (защита от забытого import) — N/A, будет добавлен в Story 1.8 когда появится первый handler

- [x] **Task 4: Документация и CI**
  - [x] 4.1 Добавить godoc комментарии к публичным функциям
  - [x] 4.2 Проверить что golangci-lint проходит: `make lint`
  - [x] 4.3 Убедиться что `go test ./internal/command/...` проходит
  - [x] 4.4 Убедиться что `go test -race ./internal/command/...` проходит

### Review Follow-ups (AI)

- [ ] [AI-Review][HIGH] Regex `^[a-z][a-z0-9-]*$` допускает завершающий дефис (например, "cmd-"), что может быть нежелательным [registry.go:17]
- [ ] [AI-Review][HIGH] RegisterWithAlias: двойной захват mutex. Между двумя захватами есть окно для race condition при вызове не из init() [registry.go:115-146]
- [ ] [AI-Review][HIGH] ListAllWithAliases использует type assertion на неэкспортируемый тип DeprecatedBridge — tight coupling [registry.go:162-194]
- [ ] [AI-Review][MEDIUM] All() возвращает map с мутабельными Handler значениями — caller может мутировать состояние handler'ов [registry.go:70-78]
- [ ] [AI-Review][MEDIUM] TestConcurrentAccess: нет гарантии одновременного старта горутин, стоит использовать barrier [registry_test.go:86-121]
- [ ] [AI-Review][LOW] Info struct добавлен позже (Epic 7), но не документирован в story 1.1 [registry.go:149-157]
- [ ] [AI-Review][LOW] Compile-time check в handler_test.go избыточен — дублирует проверку из deprecated.go:22 [handler_test.go]

## Dev Notes

### Критический контекст для реализации

**Точка интеграции в main.go (строка 30):**
```go
// ПОСЛЕ строки 29 (l.Debug("Информация о сборке"...))
// ПЕРЕД switch cfg.Command (строка 30)

// Проверяем Registry перед legacy switch
if handler, ok := command.Get(cfg.Command); ok {
    l.Debug("Выполнение команды через registry", slog.String("command", cfg.Command))
    if err := handler.Execute(ctx, cfg); err != nil {
        l.Error("Ошибка выполнения команды",
            slog.String("command", cfg.Command),
            slog.String("error", err.Error()),
        )
        os.Exit(1)
    }
    return // Успешное выполнение через registry
}
l.Debug("Команда не найдена в registry, fallback на legacy switch", slog.String("command", cfg.Command))

switch cfg.Command {
// ... существующий код ...
```

### Архитектурные ограничения

| Ограничение | Описание | Почему критично |
|-------------|----------|-----------------|
| **Dependency** | `internal/command` зависит ТОЛЬКО от `internal/config` и stdlib | Import cycle сломает компиляцию |
| **Thread-safety** | sync.RWMutex для registry | Параллельные тесты + будущая расширяемость |
| **Init-safety** | НЕ использовать cfg.Logger в init()/Register() | cfg ещё не загружен в момент init() |
| **Validation** | Panic при nil handler или empty name | Programming error, не runtime |
| **Code style** | Godoc на русском, логи на русском | Требование проекта (CLAUDE.md) |

### Паттерн Self-Registration

```go
// internal/command/handlers/version/version.go (пример для Story 1.8)
func init() {
    command.Register(&VersionHandler{})
}

type VersionHandler struct{}

func (h *VersionHandler) Name() string { return "nr-version" }
func (h *VersionHandler) Execute(ctx context.Context, cfg *config.Config) error {
    // Реализация
    return nil
}
```

### Data Flow

```
Go runtime init() → handler init() → command.Register(handler)
    ↓
main() → config.MustLoad() → command.Get(cfg.Command)
    ↓
found=true → handler.Execute(ctx, cfg) → exit
    ↓
found=false → legacy switch → execute app.* function → exit
```

### Риски и митигации

| ID | Риск | Probability | Impact | Митигация |
|----|------|-------------|--------|-----------|
| R1 | Import cycle между command и config | Medium | Critical | Constraint: ТОЛЬКО config и stdlib |
| R2 | Забыли blank import → registry пустой | High | High | Test TestRegistryNotEmpty_AfterImport + явный subtask |
| R3 | Flaky tests из-за shared state | High | High | clearRegistry() + go test -race |
| R4 | Legacy fallback не работает | Medium | Medium | Integration test для обоих путей |
| R5 | Неинформативный panic | Medium | Medium | Понятные сообщения: "command: nil handler", "command: empty handler name" |

### Примеры кода

**Handler Interface (internal/command/handler.go):**
```go
// Package command предоставляет интерфейсы и реестр для команд приложения.
package command

import (
    "context"
    "github.com/Kargones/apk-ci/internal/config"
)

// Handler определяет интерфейс обработчика команды.
// Каждая команда приложения должна реализовывать этот интерфейс.
type Handler interface {
    // Name возвращает имя команды для регистрации в реестре.
    // Должно соответствовать константам из internal/constants (например, "service-mode-status").
    Name() string

    // Execute выполняет команду с переданным контекстом и конфигурацией.
    // Возвращает ошибку если выполнение завершилось неуспешно.
    Execute(ctx context.Context, cfg *config.Config) error
}
```

**Registry (internal/command/registry.go):**
```go
package command

import "sync"

var (
    registry = make(map[string]Handler)
    mu       sync.RWMutex
)

// Register регистрирует обработчик команды в глобальном реестре.
// Паникует если:
// - h == nil (programming error)
// - h.Name() == "" (programming error)
// - команда с таким именем уже зарегистрирована
func Register(h Handler) {
    if h == nil {
        panic("command: nil handler")
    }
    name := h.Name()
    if name == "" {
        panic("command: empty handler name")
    }

    mu.Lock()
    defer mu.Unlock()

    if _, exists := registry[name]; exists {
        panic("command: duplicate handler registration for " + name)
    }
    registry[name] = h
}

// Get возвращает обработчик команды по имени.
// Возвращает (nil, false) если команда не зарегистрирована.
func Get(name string) (Handler, bool) {
    mu.RLock()
    defer mu.RUnlock()
    h, ok := registry[name]
    return h, ok
}

// clearRegistry очищает реестр. Используется только в тестах.
func clearRegistry() {
    mu.Lock()
    defer mu.Unlock()
    registry = make(map[string]Handler)
}
```

**Unit Tests (internal/command/registry_test.go):**
```go
package command

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/Kargones/apk-ci/internal/config"
)

// mockHandler - тестовый обработчик команды
type mockHandler struct {
    name string
}

func (m *mockHandler) Name() string { return m.name }
func (m *mockHandler) Execute(_ context.Context, _ *config.Config) error { return nil }

func TestRegister_Success(t *testing.T) {
    clearRegistry()

    h := &mockHandler{name: "test-command"}
    Register(h)

    got, ok := Get("test-command")
    require.True(t, ok)
    assert.Equal(t, h, got)
}

func TestRegister_Duplicate_Panics(t *testing.T) {
    clearRegistry()

    h1 := &mockHandler{name: "dup-command"}
    h2 := &mockHandler{name: "dup-command"}

    Register(h1)

    assert.PanicsWithValue(t, "command: duplicate handler registration for dup-command", func() {
        Register(h2)
    })
}

func TestRegister_NilHandler_Panics(t *testing.T) {
    clearRegistry()

    assert.PanicsWithValue(t, "command: nil handler", func() {
        Register(nil)
    })
}

func TestRegister_EmptyName_Panics(t *testing.T) {
    clearRegistry()

    h := &mockHandler{name: ""}

    assert.PanicsWithValue(t, "command: empty handler name", func() {
        Register(h)
    })
}

func TestGet_NotFound(t *testing.T) {
    clearRegistry()

    got, ok := Get("non-existent")
    assert.False(t, ok)
    assert.Nil(t, got)
}

func TestGet_Found(t *testing.T) {
    clearRegistry()

    h := &mockHandler{name: "existing"}
    Register(h)

    got, ok := Get("existing")
    require.True(t, ok)
    assert.Equal(t, h, got)
}
```

### Project Structure Notes

**Создаваемые файлы:**
```
internal/
└── command/           # НОВАЯ директория
    ├── handler.go     # Handler interface
    ├── handler_test.go # Compile check test (опционально)
    ├── registry.go    # Registry implementation
    └── registry_test.go # Unit tests
```

**Изменяемые файлы:**
- `cmd/apk-ci/main.go` — добавление проверки Registry перед switch

**Не изменять:**
- `internal/app/*` — вызовы app.* функций остаются в legacy switch
- `internal/config/*` — структура Config не меняется
- `internal/constants/*` — константы не меняются

### Testing Standards

- Framework: testify/assert, testify/require
- Pattern: Table-driven tests где применимо
- Naming: `Test{FunctionName}_{Scenario}`
- Location: `*_test.go` рядом с тестируемым файлом
- Run: `go test ./internal/command/... -v`
- Race: `go test ./internal/command/... -race`

### Обязательные тесты

| Тест | Описание | AC |
|------|----------|-----|
| TestRegister_Success | Регистрация и получение handler | AC1 |
| TestRegister_Duplicate_Panics | Panic при двойной регистрации | AC3 |
| TestRegister_NilHandler_Panics | Panic при nil handler | AC3 |
| TestRegister_EmptyName_Panics | Panic при empty name | AC3 |
| TestGet_NotFound | (nil, false) для несуществующей команды | AC2 |
| TestGet_Found | (handler, true) для зарегистрированной | AC1 |
| TestConcurrentAccess | Race condition check с -race | — |

### References

- [Source: _bmad-output/project-planning-artifacts/architecture.md#Pattern: Command Registry with Self-Registration] — Паттерн
- [Source: _bmad-output/project-planning-artifacts/architecture.md#ADR-002] — ADR
- [Source: _bmad-output/project-planning-artifacts/epics/epic-1-foundation.md#Story 1.1] — Epic description
- [Source: _bmad-output/project-planning-artifacts/prd.md#FR1] — FR1: Регистрация реализаций
- [Source: _bmad-output/project-planning-artifacts/prd.md#FR3] — FR3: Новые реализации без изменения кода
- [Source: _bmad-output/project-planning-artifacts/prd.md#FR47] — FR47: NR-префикс
- [Source: _bmad-output/implementation-artifacts/sprint-artifacts/tech-spec-epic-1.md] — Tech spec
- [Source: _bmad-output/implementation-artifacts/sprint-artifacts/1-1-command-registry-with-self-registration.context.xml] — Полный контекст

### Review Follow-ups (AI Code Review #34)

- [x] [AI-Review][HIGH] ~~Regex ^[a-z][a-z0-9-]*$ допускает имена с завершающим дефисом (cmd-)~~ — ИСПРАВЛЕНО Review #34: regex обновлён на strict kebab-case [registry.go:18]
- [ ] [AI-Review][MEDIUM] Дублирование mutex захвата в RegisterWithAlias — потенциальное окно для race condition [registry.go:115-146]
- [ ] [AI-Review][MEDIUM] ListAllWithAliases использует type assertion на *DeprecatedBridge вместо interface [registry.go:162-194]

## Dev Agent Record

### Context Reference

- [Story Context XML](./../sprint-artifacts/1-1-command-registry-with-self-registration.context.xml) — полный контекст с анализом рисков, design decisions, примерами кода

### Agent Model Used

Claude Opus 4.5 (claude-opus-4-5-20251101)

### Debug Log References

- Все unit тесты пройдены: `go test ./internal/command/... -v` — 8/8 PASS
- Race detection: `go test ./internal/command/... -race` — PASS
- Lint: `golangci-lint run ./internal/command/...` — 0 issues
- Build: `go build ./cmd/apk-ci/...` — успешно
- Integration tests: `TestCommandRegistry_LegacyFallback`, `TestCommandRegistry_GetUnknown` — PASS

### Completion Notes List

- ✅ Создан пакет `internal/command` с Handler interface и Registry
- ✅ Реализована потокобезопасная регистрация команд с sync.RWMutex
- ✅ Добавлена валидация: panic при nil handler, empty name, duplicate registration
- ✅ Интегрировано в main.go: проверка Registry перед legacy switch
- ✅ Добавлено логирование выбранного пути выполнения команды
- ✅ Написаны unit тесты: 8 тестов покрывают все AC
- ✅ Написаны integration тесты: TestCommandRegistry_LegacyFallback, TestCommandRegistry_GetUnknown
- ✅ Код соответствует архитектурным ограничениям (только config и stdlib зависимости)
- ⏭️ Task 3.2, 3.7 отложены до Story 1.8 (когда появится первый handler)

### File List

**Создано:**
- `internal/command/handler.go` — Handler interface
- `internal/command/registry.go` — Registry implementation с валидацией kebab-case и функциями All()/Names()
- `internal/command/registry_test.go` — Unit tests (18 тестов включая AC5, race tests, валидация формата)
- `internal/command/handler_test.go` — Compile-time interface check

**Изменено:**
- `cmd/apk-ci/main.go` — добавлен import command, интеграция Registry
- `cmd/apk-ci/main_test.go` — добавлены integration тесты для Registry, исправлены errcheck, заменена кастомная contains на strings.Contains
- `cmd/apk-ci/yaml_integration_test.go` — заменён deprecated ioutil на os
- `cmd/apk-ci/create_temp_db_test.go` — добавлен nolint для goconst

---

## Change Log

| Дата | Автор | Изменение |
|------|-------|-----------|
| 2025-11-26 | SM Agent | Создан черновик истории из epics.md |
| 2025-11-26 | Story Context Workflow | Создан context.xml с 5 методами анализа |
| 2025-11-26 | Review | Синхронизация .md и .xml: thread-safety, валидация, blank import, риски, тесты |
| 2026-01-26 | Create-Story Workflow | Обогащение контекстом, статус ready-for-dev |
| 2026-01-26 | Dev Agent (Claude Opus 4.5) | Реализация: Handler interface, Registry, интеграция в main.go, unit/integration тесты. Статус → review |
| 2026-01-26 | Code Review (Claude Opus 4.5) | Исправлено: удалён бинарник из git, согласован exit code (1→8), заменён deprecated ioutil на os.ReadFile, улучшен concurrent тест (fmt.Sprintf + assertions). Статус → done |
| 2026-01-26 | Code Review #2 (Claude Opus 4.5) | Adversarial review: исправлено 16 errcheck, deprecated ioutil, gocritic elseif, gosec G304. Добавлено: handler_test.go (compile-time check), TestRegistryPath_AC5, TestConcurrentReadWrite. Все тесты и линтер чистые. |
| 2026-01-26 | Code Review #3 (Claude Opus 4.5) | M1: добавлены All()/Names() для отладки registry. M2: добавлена валидация kebab-case формата имени команды (regex). L2: заменена кастомная contains на strings.Contains. Добавлено 7 новых тестов (18 total). Все тесты проходят, линтер чистый. |
