# Story 1.2: NR-Migration Bridge (DeprecatedBridge)

Status: drafted

## Story

As a DevOps-инженер,
I want получать warning при использовании deprecated команд,
so that я знаю о необходимости миграции на NR-версии.

## Acceptance Criteria

| # | Критерий | Тестируемость |
|---|----------|---------------|
| AC1 | Given команда зарегистрирована через `RegisterWithAlias(handler, "old-name")`, When вызывается старое имя команды, Then выводится warning в stderr с рекомендацией использовать новое имя | Unit test: warning output |
| AC2 | DeprecatedBridge выполняет команду через actual handler после вывода warning | Unit test: delegation |
| AC3 | Warning содержит: старое имя, новое имя, текст "deprecated" | String inspection test |
| AC4 | Warning выводится на каждый вызов (не кэшируется) | Multiple call test |
| AC5 | `RegisterWithAlias` с пустым deprecated string регистрирует handler без bridge | Unit test: empty alias |

## Tasks / Subtasks

- [ ] **Task 1: Создать DeprecatedBridge struct** (AC: 1, 2, 3)
  - [ ] 1.1 Создать файл `internal/command/deprecated.go`
  - [ ] 1.2 Определить struct `DeprecatedBridge` с полями: `actual Handler`, `deprecated string`, `newName string`
  - [ ] 1.3 Реализовать метод `Name() string` возвращающий deprecated имя
  - [ ] 1.4 Реализовать метод `Description() string` возвращающий описание от actual handler + пометка deprecated
  - [ ] 1.5 Реализовать метод `Execute(ctx, cfg) error`:
    - Вывести warning в stderr: `"WARNING: Command '%s' is deprecated. Use '%s' instead."`
    - Вызвать `actual.Execute(ctx, cfg)`
  - [ ] 1.6 Написать unit tests для DeprecatedBridge

- [ ] **Task 2: Добавить RegisterWithAlias в Registry** (AC: 1, 5)
  - [ ] 2.1 Реализовать функцию `RegisterWithAlias(h Handler, deprecated string)` в `registry.go`
  - [ ] 2.2 Логика: если deprecated пустой — просто `Register(h)`, иначе — создать и зарегистрировать DeprecatedBridge
  - [ ] 2.3 Зарегистрировать actual handler под его настоящим именем
  - [ ] 2.4 Зарегистрировать DeprecatedBridge под deprecated именем
  - [ ] 2.5 Написать unit tests для RegisterWithAlias

- [ ] **Task 3: Warning output** (AC: 3, 4)
  - [ ] 3.1 Warning выводить через `fmt.Fprintf(os.Stderr, ...)` (временно, до Story 1.4 Logger)
  - [ ] 3.2 Формат: `"WARNING: Command '%s' is deprecated. Use '%s' instead.\n"`
  - [ ] 3.3 Проверить что warning не влияет на stdout (не ломает JSON parsing)
  - [ ] 3.4 Написать test на вывод warning при каждом вызове

- [ ] **Task 4: Интеграционные тесты**
  - [ ] 4.1 Написать integration test: RegisterWithAlias + Get deprecated → Execute → warning + actual executed
  - [ ] 4.2 Написать test: RegisterWithAlias + Get actual name → Execute без warning
  - [ ] 4.3 Проверить что golangci-lint проходит

## Dev Notes

### Архитектурные ограничения

- **Зависимость**: Требует Story 1.1 (Command Registry) — использует `Register()`, `Get()`, `Handler` interface
- **Warning output**: Временно через `fmt.Fprintf(os.Stderr)`, после Story 1.4 заменить на `Logger.Warn()`
- **Паттерн Decorator**: DeprecatedBridge оборачивает actual handler, добавляя warning behavior
- **Data Flow**: `RegisterWithAlias → Register(actual) + Register(bridge) → Get("old") → bridge.Execute() → warn + actual.Execute()`

### Риски

| ID | Риск | Вероятность | Влияние | Митигация |
|----|------|-------------|---------|-----------|
| R1 | Warning в stdout ломает JSON парсеры | Высокая | Высокое | Использовать stderr для warning |
| R2 | RegisterWithAlias конфликтует с existing Register | Средняя | Среднее | Вызывать Register() внутри, не дублировать логику |
| R3 | Description() от bridge не информативен | Низкая | Низкое | Добавить [deprecated] в description |

### Паттерн NR-Migration Bridge (из Architecture)

```go
// internal/command/deprecated.go
type DeprecatedBridge struct {
    actual     Handler
    deprecated string
    newName    string
}

func (b *DeprecatedBridge) Name() string { return b.deprecated }

func (b *DeprecatedBridge) Execute(ctx context.Context, cfg *config.Config) error {
    fmt.Fprintf(os.Stderr, "WARNING: Command '%s' is deprecated. Use '%s' instead.\n",
        b.deprecated, b.newName)
    return b.actual.Execute(ctx, cfg)
}
```

### Project Structure Notes

- Новый файл: `internal/command/deprecated.go`
- Дополнение к `internal/command/registry.go`: функция `RegisterWithAlias()`
- Соответствует Architecture: `internal/command/` для Command Registry components

### Тестирование

Обязательные тесты:
- `TestDeprecatedBridge_Execute_OutputsWarning` — warning выводится в stderr
- `TestDeprecatedBridge_Execute_DelegatesActual` — actual handler вызывается
- `TestDeprecatedBridge_Name_ReturnsDeprecated` — Name() возвращает deprecated имя
- `TestRegisterWithAlias_EmptyAlias_JustRegisters` — пустой alias = обычная регистрация
- `TestRegisterWithAlias_RegistersBothNames` — регистрируются оба имени (actual и deprecated)
- `TestDeprecatedBridge_WarningEveryCall` — warning на каждый вызов, не кэшируется

### References

- [Source: bdocs/architecture.md#Pattern: NR-Migration Bridge] — Паттерн DeprecatedBridge
- [Source: bdocs/sprint-artifacts/tech-spec-epic-1.md#AC2] — Acceptance Criteria из tech-spec
- [Source: bdocs/epics.md#Story 1.2] — Полное описание истории
- [Source: bdocs/prd.md#FR48] — FR48: Старые команды помечаются @deprecated но продолжают работать
- [Source: bdocs/prd.md#FR50] — FR50: Система логирует использование deprecated-команд с рекомендацией миграции

## Dev Agent Record

### Context Reference

<!-- Path(s) to story context XML will be added here by context workflow -->

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
| 2025-11-26 | SM Agent | Создан черновик истории из epics.md и tech-spec-epic-1.md |
