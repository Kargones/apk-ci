# Story 1.1: Command Registry с self-registration

Status: drafted

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

- [ ] **Task 1: Создать Handler interface** (AC: 1)
  - [ ] 1.1 Создать файл `internal/command/handler.go`
  - [ ] 1.2 Определить интерфейс Handler с методами `Name() string` и `Execute(ctx context.Context, cfg *config.Config) error`
  - [ ] 1.3 Написать unit test для интерфейса (compile check)

- [ ] **Task 2: Реализовать Command Registry** (AC: 1, 2, 3)
  - [ ] 2.1 Создать файл `internal/command/registry.go`
  - [ ] 2.2 Реализовать `var registry = make(map[string]Handler)` с sync.RWMutex
  - [ ] 2.3 Реализовать `func Register(h Handler)` с panic при дублировании
  - [ ] 2.4 Добавить валидацию: panic при nil handler или empty name
  - [ ] 2.5 Реализовать `func Get(name string) (Handler, bool)`
  - [ ] 2.6 Добавить unexported `clearRegistry()` для тестов
  - [ ] 2.7 Написать unit tests для Register и Get (включая nil/empty/race tests)

- [ ] **Task 3: Интегрировать Registry в main.go** (AC: 4, 5)
  - [ ] 3.1 Добавить blank import для триггера init(): `import _ ".../internal/command/handlers"`
  - [ ] 3.2 Добавить проверку Registry перед legacy switch
  - [ ] 3.3 Реализовать fallback логику на legacy switch
  - [ ] 3.4 Добавить логирование выбранного пути ("registry" / "legacy")
  - [ ] 3.5 Написать integration test для обоих путей
  - [ ] 3.6 Написать test TestRegistryNotEmpty_AfterImport (защита от забытого import)

- [ ] **Task 4: Документация и CI**
  - [ ] 4.1 Добавить godoc комментарии к публичным функциям
  - [ ] 4.2 Проверить что golangci-lint проходит
  - [ ] 4.3 Убедиться что `go test ./internal/command/...` проходит

## Dev Notes

### Архитектурные ограничения

- **Thread-safety**: Требуется sync.RWMutex для тестов (clearRegistry, parallel tests) и будущей расширяемости
- **Паттерн**: Self-registration через init() функции в handler файлах
- **Data Flow**: `init() → Register → main() → Get(command) → Execute`
- **Валидация**: Register() должен проверять nil handler и empty name (panic с информативным сообщением)

### Риски

> Полный анализ рисков см. в [Story Context](./1-1-command-registry-with-self-registration.context.xml) секция `<riskMatrix>`

Ключевые риски:
- **R1 (Critical)**: Import cycle между command и config — митигация: только config и stdlib
- **R2 (High)**: Забыли blank import → registry пустой — митигация: test + явный subtask
- **R3 (High)**: Flaky tests из-за shared state — митигация: clearRegistry() + -race
- **R4 (Medium)**: Legacy fallback не работает — митигация: integration test

### Project Structure Notes

- Новые файлы создаются в `internal/command/`:
  - `internal/command/handler.go` — интерфейс Handler
  - `internal/command/registry.go` — реализация Registry
- Структура соответствует Architecture: `internal/command/` для Command Registry
- Naming convention: lowercase package name `command`, PascalCase для Handler

### Тестирование

> Полный список тестов см. в [Story Context](./1-1-command-registry-with-self-registration.context.xml) секция `<tests>`

Обязательные тесты:
- `TestRegister_Success` — регистрация и получение handler
- `TestRegister_Duplicate_Panics` — panic при двойной регистрации
- `TestRegister_NilHandler_Panics` — panic при nil handler
- `TestRegister_EmptyName_Panics` — panic при empty name
- `TestGet_NotFound` — (nil, false) для несуществующей команды
- `TestGet_Found` — (handler, true) для зарегистрированной
- `TestConcurrentAccess` — race condition check
- `TestRegistryNotEmpty_AfterImport` — защита от забытого blank import

### References

- [Source: bdocs/architecture.md#Pattern: Command Registry with Self-Registration] — Паттерн Command Registry
- [Source: bdocs/architecture.md#ADR-002] — ADR для Command Registry
- [Source: bdocs/epics.md#Story 1.1] — Полное описание истории
- [Source: bdocs/prd.md#FR1] — FR1: Система поддерживает регистрацию реализаций через strategy-интерфейсы
- [Source: bdocs/prd.md#FR3] — FR3: Новые реализации добавляются без изменения существующего кода
- [Source: bdocs/prd.md#FR47] — FR47: Новые команды имеют префикс NR во время миграции

## Dev Agent Record

### Context Reference

- [Story Context XML](./1-1-command-registry-with-self-registration.context.xml) — полный контекст с анализом рисков, design decisions, примерами кода

### Agent Model Used

<!-- Will be filled by dev agent -->

### Debug Log References

<!-- Will be filled during implementation -->

### Completion Notes List

<!-- Will be filled after implementation -->

### File List

<!-- Will be filled after implementation -->

---

## Change Log

| Дата | Автор | Изменение |
|------|-------|-----------|
| 2025-11-26 | SM Agent | Создан черновик истории из epics.md |
| 2025-11-26 | Story Context Workflow | Создан context.xml с 5 методами анализа |
| 2025-11-26 | Review | Синхронизация .md и .xml: thread-safety, валидация, blank import, риски, тесты |
