# Story 2.8: State-Aware Execution (FR60-62)

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a система,
I want проверять текущее состояние перед операцией и возвращать флаг state_changed,
So that операции идемпотентны, безопасны и CI/CD может программно определить было ли произведено реальное изменение.

## Acceptance Criteria

1. **AC-1**: enable когда уже включён → success + `"already_enabled": true` + `"state_changed": false`
2. **AC-2**: disable когда уже выключен → success + `"already_disabled": true` + `"state_changed": false`
3. **AC-3**: Логируется текущее состояние перед изменением (slog.Info с полями enabled, scheduled_jobs_blocked, active_sessions)
4. **AC-4**: JSON output enable содержит `"state_changed": true/false`
5. **AC-5**: JSON output disable содержит `"state_changed": true/false`
6. **AC-6**: Text output при state_changed=false выводит "(состояние не изменено)" / "(уже был включён)" / "(уже был отключён)"
7. **AC-7**: force-disconnect при отсутствии сессий: `"no_active_sessions": true` + `"state_changed": false`
8. **AC-8**: Unit-тесты для всех state_changed сценариев в каждом handler
9. **AC-9**: Существующие тесты не ломаются — `state_changed` добавляется без нарушения backward compatibility JSON-контракта

## Tasks / Subtasks

- [x] Task 1: Добавить поле `StateChanged` в `ServiceModeEnableData` (AC: 1, 4, 6)
  - [x] 1.1 В `servicemodeenablehandler/handler.go` добавить `StateChanged bool \`json:"state_changed"\`` в struct `ServiceModeEnableData`
  - [x] 1.2 В путь `already_enabled` (строка 163): установить `StateChanged: false`
  - [x] 1.3 В путь нормального включения (строка 208): установить `StateChanged: true`
  - [x] 1.4 Обновить `writeText`: при `StateChanged == false` добавить "(состояние не изменено)" к выводу
- [x] Task 2: Добавить поле `StateChanged` в `ServiceModeDisableData` (AC: 2, 5, 6)
  - [x] 2.1 В `servicemodedisablehandler/handler.go` добавить `StateChanged bool \`json:"state_changed"\`` в struct `ServiceModeDisableData`
  - [x] 2.2 В путь `already_disabled` (строка 143): установить `StateChanged: false`
  - [x] 2.3 В путь нормального отключения (строка 184): установить `StateChanged: true`
  - [x] 2.4 Обновить `writeText`: при `StateChanged == false` добавить "(состояние не изменено)" к выводу
- [x] Task 3: Добавить поле `StateChanged` в `ForceDisconnectData` (AC: 7)
  - [x] 3.1 В `forcedisconnecthandler/handler.go` добавить `StateChanged bool \`json:"state_changed"\`` в struct `ForceDisconnectData`
  - [x] 3.2 В путь `no_active_sessions` (нет сессий для завершения): установить `StateChanged: false`
  - [x] 3.3 В путь нормального завершения сессий: установить `StateChanged: true`
- [x] Task 4: Добавить логирование состояния перед изменением (AC: 3)
  - [x] 4.1 В `servicemodeenablehandler/handler.go`: после GetServiceModeStatus (строка 155), добавить slog.Info с полями enabled, scheduled_jobs_blocked, active_sessions
  - [x] 4.2 В `servicemodedisablehandler/handler.go`: после GetServiceModeStatus (строка 135), добавить slog.Info с полями enabled, scheduled_jobs_blocked
  - [x] 4.3 В `forcedisconnecthandler/handler.go`: после получения списка сессий, добавить slog.Info с количеством сессий
- [x] Task 5: Обновить unit-тесты (AC: 8, 9)
  - [x] 5.1 `servicemodeenablehandler/handler_test.go`: проверить `state_changed: false` в тесте AlreadyEnabled, `state_changed: true` в тесте JSONOutput
  - [x] 5.2 `servicemodedisablehandler/handler_test.go`: проверить `state_changed: false` в тесте AlreadyDisabled, `state_changed: true` в тесте JSONOutput
  - [x] 5.3 `forcedisconnecthandler/handler_test.go`: проверить `state_changed: false` в тесте NoActiveSessions, `state_changed: true` в тесте JSONOutput
  - [x] 5.4 Запустить все существующие тесты — убедиться в отсутствии регрессий

### Review Follow-ups (AI)

- [ ] [AI-Review][MEDIUM] StateChanged=true при enable даже если terminate sessions проваливается — не отражает полноту операции [servicemodeenablehandler/handler.go:241]
- [ ] [AI-Review][MEDIUM] ForceDisconnect: StateChanged семантика "хотя бы одна сессия" отличается от бинарной enable/disable [forcedisconnecthandler/handler.go:278]
- [ ] [AI-Review][LOW] Status handler не имеет state_changed — JSON потребитель ожидающий единый формат не получит поле [servicemodestatushandler/handler.go:42-43]
- [ ] [AI-Review][LOW] Нет matrix test проверяющий state_changed семантику во всех handler'ах одновременно

### Review Follow-ups (AI Code Review #34)

- [ ] [AI-Review][MEDIUM] Нет state_changed в status handler — инконсистентность с enable/disable/disconnect [servicemodestatushandler/handler.go:42-43]

## Dev Notes

### Архитектурные ограничения

- **Все комментарии на русском языке** (CLAUDE.md).
- **Logger только в stderr**, stdout зарезервирован для output (JSON/text).
- **НЕ менять интерфейсы RAC** — Story 2.1 и 2.2 закончены, их код не трогать.
- **НЕ менять registry, handler interface, deprecated bridge** — Epic 1 (done).
- **НЕ менять output пакет** (`internal/pkg/output/`) — `state_changed` является частью Data struct каждого handler, а не общего Result.
- **НЕ добавлять Wire-провайдеры** (будет позже).
- **НЕ рефакторить createRACClient** — это вне scope (отмечено как Review Follow-up в Story 2.7).
- **Backward compatibility**: добавление нового JSON-поля `state_changed` не ломает существующих потребителей — Go `json.Unmarshal` игнорирует неизвестные поля.

### Объём изменений — МИНИМАЛЬНЫЙ

Это story про добавление одного поля `StateChanged bool` в три существующих data struct и установку его значения в двух местах каждого handler. **НЕ перестраивать**, **НЕ рефакторить**, **НЕ переименовывать**.

Полный объём изменений:
- 3 handler файла — добавить поле в struct + установить значение в 2 путях + обновить writeText + добавить slog.Info
- 3 тестовых файла — добавить проверку state_changed в существующие и новые тесты

### Паттерн изменения — одинаковый для всех трёх handler

**Шаг 1**: Добавить поле в data struct:

```go
// В ServiceModeEnableData / ServiceModeDisableData / ForceDisconnectData:
StateChanged bool `json:"state_changed"`
```

**Шаг 2**: Установить `StateChanged: false` в идемпотентном пути:

```go
// Пример для enable (already enabled):
data := &ServiceModeEnableData{
    Enabled:        true,
    AlreadyEnabled: true,
    StateChanged:   false,  // ← ДОБАВИТЬ
    // ...
}
```

**Шаг 3**: Установить `StateChanged: true` в пути реального изменения:

```go
// Пример для enable (нормальный путь):
data := &ServiceModeEnableData{
    Enabled:        true,
    AlreadyEnabled: false,
    StateChanged:   true,  // ← ДОБАВИТЬ
    // ...
}
```

**Шаг 4**: Обновить текстовый вывод (опционально — текст уже отражает already_enabled/disabled):

Текстовый вывод уже различает "ВКЛЮЧЁН" vs "ВКЛЮЧЁН (уже был включён)". Можно НЕ менять текстовый вывод, так как `already_enabled/already_disabled` уже даёт эту информацию в текстовом формате. Поле `state_changed` в первую очередь для программного потребления (JSON).

### Логирование состояния перед изменением (AC-3)

Добавить после `GetServiceModeStatus` (до проверки идемпотентности):

```go
// В servicemodeenablehandler, после строки 155:
if status != nil {
    log.Info("Текущее состояние перед операцией",
        slog.Bool("enabled", status.Enabled),
        slog.Bool("scheduled_jobs_blocked", status.ScheduledJobsBlocked),
        slog.Int("active_sessions", status.ActiveSessions))
}
```

```go
// В servicemodedisablehandler, после строки 135:
if status != nil {
    log.Info("Текущее состояние перед операцией",
        slog.Bool("enabled", status.Enabled),
        slog.Bool("scheduled_jobs_blocked", status.ScheduledJobsBlocked))
}
```

```go
// В forcedisconnecthandler, после получения сессий:
log.Info("Текущее состояние: активных сессий", slog.Int("count", len(sessions)))
```

### Тестирование — добавить проверку state_changed в существующие тесты

**Паттерн для enable:**

```go
// В TestServiceModeEnableHandler_Execute_JSONOutput — state_changed: true
data := result.Data.(map[string]interface{})
assert.True(t, data["state_changed"].(bool))

// В TestServiceModeEnableHandler_Execute_AlreadyEnabled — state_changed: false
data := result.Data.(map[string]interface{})
assert.False(t, data["state_changed"].(bool))
```

**Паттерн для disable:**

```go
// В TestServiceModeDisableHandler_Execute_JSONOutput — state_changed: true
data := result.Data.(map[string]interface{})
assert.True(t, data["state_changed"].(bool))

// В TestServiceModeDisableHandler_Execute_AlreadyDisabled — state_changed: false
data := result.Data.(map[string]interface{})
assert.False(t, data["state_changed"].(bool))
```

**Паттерн для force-disconnect:**

```go
// В TestForceDisconnectHandler_Execute_JSONOutput — state_changed: true
data := result.Data.(map[string]interface{})
assert.True(t, data["state_changed"].(bool))

// В TestForceDisconnectHandler_Execute_NoActiveSessions — state_changed: false
data := result.Data.(map[string]interface{})
assert.False(t, data["state_changed"].(bool))
```

### Project Structure Notes

- **Нет новых файлов** — только изменения в существующих
- **Нет новых пакетов** — только добавление поля и slog-вызова в 3 handler
- **Нет изменений в main.go** — не добавляются новые команды
- **Нет изменений в constants.go** — не добавляются новые константы

### Файлы на изменение

| Файл | Действие | Описание |
|------|----------|----------|
| `internal/command/handlers/servicemodeenablehandler/handler.go` | изменить | Добавить `StateChanged bool` в struct, установить в 2 путях, добавить slog.Info |
| `internal/command/handlers/servicemodeenablehandler/handler_test.go` | изменить | Добавить проверку `state_changed` в тесты JSONOutput и AlreadyEnabled |
| `internal/command/handlers/servicemodedisablehandler/handler.go` | изменить | Добавить `StateChanged bool` в struct, установить в 2 путях, добавить slog.Info |
| `internal/command/handlers/servicemodedisablehandler/handler_test.go` | изменить | Добавить проверку `state_changed` в тесты JSONOutput и AlreadyDisabled |
| `internal/command/handlers/forcedisconnecthandler/handler.go` | изменить | Добавить `StateChanged bool` в struct, установить в 2 путях, добавить slog.Info |
| `internal/command/handlers/forcedisconnecthandler/handler_test.go` | изменить | Добавить проверку `state_changed` в тесты JSONOutput и NoActiveSessions |

### Файлы НЕ ТРОГАТЬ

- `internal/adapter/onec/rac/` — весь пакет (interfaces, client, mock) — Story 2.1, 2.2 (done)
- `internal/command/registry.go`, `handler.go`, `deprecated.go` — Epic 1 (done)
- `internal/pkg/output/` — не менять Result/Metadata/ErrorInfo
- `internal/pkg/tracing/` — не менять
- `internal/constants/constants.go` — не нужны новые константы
- `cmd/apk-ci/main.go` — не нужны изменения
- Legacy код: `internal/rac/`, `internal/servicemode/`, `internal/app/app.go`

### Что НЕ делать

- НЕ менять RAC interfaces или client (Story 2.1, 2.2)
- НЕ создавать новые файлы или пакеты
- НЕ добавлять Wire-провайдеры
- НЕ рефакторить createRACClient (вне scope)
- НЕ менять текстовый вывод если уже отражает already_enabled/disabled (текстовый вывод уже информативен)
- НЕ менять поведение идемпотентности — только ДОБАВИТЬ поле state_changed к уже работающему коду
- НЕ добавлять retry logic
- НЕ реализовывать FR61 (auto-deps) — это отдельный scope
- НЕ реализовывать FR63 (план операций в verbose) — это отдельный scope

### References

- [Source: _bmad-output/project-planning-artifacts/epics/epic-2-service-mode.md#Story 2.8]
- [Source: _bmad-output/project-planning-artifacts/prd.md#FR60-FR62 — Модель выполнения операций]
- [Source: internal/command/handlers/servicemodeenablehandler/handler.go — ServiceModeEnableData struct (строка 27), already_enabled path (строка 161), normal path (строка 208)]
- [Source: internal/command/handlers/servicemodedisablehandler/handler.go — ServiceModeDisableData struct (строка 27), already_disabled path (строка 141), normal path (строка 184)]
- [Source: internal/command/handlers/forcedisconnecthandler/handler.go — ForceDisconnectData struct, no_active_sessions path]
- [Source: internal/adapter/onec/rac/interfaces.go — ServiceModeManager, GetServiceModeStatus]
- [Source: internal/adapter/onec/rac/client.go — GetServiceModeStatus:518, VerifyServiceMode:540]
- [Source: internal/pkg/output/result.go — Result, Metadata structs]

### Git Intelligence

Последние коммиты релевантны текущей story:
- `95823f9 fix(handler): prevent nil pointer dereference in service mode post-check`
- `5701b3e feat: implement nr-service-mode-disable command`
- `d1426df feat(command): add nr-service-mode-disable command handler`
- `3b463fc fix(forcedisconnecthandler): replace time.Sleep with context-aware delay`
- `674f709 feat(command): add nr-force-disconnect-sessions handler`
- `da1a64e feat(command): implement nr-service-mode-enable handler`

**Паттерны из git:**
- Commit convention: `feat(scope): description` на английском
- Тесты добавляются вместе с кодом в одном коммите

### Previous Story Intelligence (Story 2.5, 2.6, 2.7)

**Из Story 2.5 (enable):**
- Идемпотентность реализована: `already_enabled` через `GetServiceModeStatus` перед `EnableServiceMode`
- Паттерн Check → Act → Verify работает
- 15+ тестов, 0 lint issues

**Из Story 2.6 (force-disconnect):**
- Идемпотентность реализована: `no_active_sessions` при пустом списке сессий
- `make([]Type, 0)` для гарантии `[]` в JSON (не `null`)

**Из Story 2.7 (disable):**
- Идемпотентность реализована: `already_disabled` через `GetServiceModeStatus` перед `DisableServiceMode`
- Post-check через `GetServiceModeStatus` для определения `ScheduledJobsUnblocked`
- Fix: nil pointer dereference при `(nil, nil)` от post-check
- 17 тестов PASS, 0 lint

**Применимость к Story 2.8:**
- Story 2.8 — финальное оформление: добавление `state_changed` к уже работающему коду
- Минимальные изменения: поле в struct + значение в 2 путях + slog.Info × 3 handler
- Все handlers уже имеют рабочую идемпотентность, нужно только добавить унифицированный флаг

### Коды ошибок

Не добавляются новые коды ошибок — story не меняет error paths.

## Dev Agent Record

### Agent Model Used

Claude Opus 4.5 (claude-opus-4-5-20251101)

### Debug Log References

- Все 3 handler-пакета: тесты PASS (servicemodeenablehandler, servicemodedisablehandler, forcedisconnecthandler)
- Полный test suite: 0 новых регрессий (pre-existing failures в TestCommandRegistry_LegacyFallback и TestExtensionPublish_MissingEnvVars не связаны с изменениями)
- Lint: 0 issues в изменённых файлах

### Completion Notes List

- Добавлено поле `StateChanged bool` в 3 data struct: ServiceModeEnableData, ServiceModeDisableData, ForceDisconnectData
- Установлено `StateChanged: false` в идемпотентных путях (already_enabled, already_disabled, no_active_sessions)
- Установлено `StateChanged: true` в путях реального изменения
- Добавлено slog.Info логирование состояния перед изменением в каждом handler
- Обновлены тесты: проверка `state_changed` в JSONOutput (true) и AlreadyEnabled/AlreadyDisabled/NoSessions (false) тестах
- Примечание по AC-6: текстовый вывод уже содержит информацию "(уже был включён)"/"(уже был отключён)" через AlreadyEnabled/AlreadyDisabled — дополнительные изменения текстового вывода не потребовались (поле writeText уже информативно). Subtask 1.4 и 2.4 выполнены — text output уже отражает состояние через existing "(уже был включён/отключён)" суффиксы.
- Backward compatibility (AC-9): новое JSON-поле `state_changed` не ломает существующих потребителей — Go `json.Unmarshal` игнорирует неизвестные поля

### Change Log

- 2026-01-28: Реализована Story 2.8 — добавлено поле `state_changed` в 3 handler data struct + slog.Info логирование состояния + тесты
- 2026-01-28: Code Review fixes — (1) добавлено `active_sessions` в slog.Info disable handler (AC-3), (2) исправлена логика `StateChanged` в force-disconnect: false при полном провале terminate, (3) добавлен тест `AllTerminateFailed`
- 2026-01-28: Code Review #2 fixes — (1) AC-6 force-disconnect: добавлен текст "(состояние не изменено)" при AllTerminateFailed в writeText, (2) исправлена семантика `PartialFailure`: true только при реальном частичном результате (some success + some errors), (3) добавлен тест `TextAllTerminateFailed`, (4) lint fix: switch вместо if-else chain

### File List

| Файл | Действие |
|------|----------|
| `internal/command/handlers/servicemodeenablehandler/handler.go` | изменён |
| `internal/command/handlers/servicemodeenablehandler/handler_test.go` | изменён |
| `internal/command/handlers/servicemodedisablehandler/handler.go` | изменён |
| `internal/command/handlers/servicemodedisablehandler/handler_test.go` | изменён |
| `internal/command/handlers/forcedisconnecthandler/handler.go` | изменён |
| `internal/command/handlers/forcedisconnecthandler/handler_test.go` | изменён |
