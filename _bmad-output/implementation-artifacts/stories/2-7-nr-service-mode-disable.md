# Story 2.7: nr-service-mode-disable

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a DevOps-инженер,
I want отключить сервисный режим информационной базы через NR-команду,
So that пользователи могут возобновить работу с базой после завершения обслуживания.

## Acceptance Criteria

1. **AC-1**: Команда `BR_COMMAND=nr-service-mode-disable BR_INFOBASE_NAME=MyBase` отключает сервисный режим
2. **AC-2**: Сервисный режим отключён: сессии разрешены (`sessions-deny=off`), регулярные задания разблокированы (условно — см. legacy-паттерн)
3. **AC-3**: Команда идемпотентна: повторный вызов при уже отключённом режиме — success + `"already_disabled": true` (FR62)
4. **AC-4**: JSON output содержит: `disabled`, `already_disabled`, `scheduled_jobs_unblocked`, `infobase_name` + metadata (duration_ms, trace_id, api_version)
5. **AC-5**: Text output содержит человекочитаемый статус отключения
6. **AC-6**: Команда зарегистрирована в Command Registry через `init()` и доступна через `command.Get("nr-service-mode-disable")`
7. **AC-7**: Legacy команда `service-mode-disable` привязана как deprecated alias через `RegisterWithAlias`
8. **AC-8**: trace_id присутствует в логах и JSON-ответе
9. **AC-9**: При отсутствии `BR_INFOBASE_NAME` — возврат структурированной ошибки (code + message)
10. **AC-10**: Верификация: после отключения вызвать `VerifyServiceMode(ctx, clusterUUID, infobaseUUID, false)` для проверки
11. **AC-11**: Unit-тесты покрывают: регистрацию, deprecated alias, text output, JSON output, идемпотентность (already disabled), ошибки RAC (табличные), верификацию, nil config, text error

## Tasks / Subtasks

- [x] Task 1: Добавить константу `ActNRServiceModeDisable` (AC: 6, 7)
  - [x] 1.1 В `internal/constants/constants.go` добавить `ActNRServiceModeDisable = "nr-service-mode-disable"`
- [x] Task 2: Создать пакет handler `internal/command/handlers/servicemodedisablehandler/` (AC: 6, 7)
  - [x] 2.1 Создать `handler.go` со struct `ServiceModeDisableHandler`
  - [x] 2.2 Реализовать `Name() string` — возвращает `constants.ActNRServiceModeDisable`
  - [x] 2.3 Реализовать `Description() string` — "Отключение сервисного режима информационной базы"
  - [x] 2.4 В `init()` зарегистрировать через `command.RegisterWithAlias(&ServiceModeDisableHandler{}, constants.ActServiceModeDisable)` — привязать legacy alias
- [x] Task 3: Реализовать `Execute` метод (AC: 1-5, 8-10)
  - [x] 3.1 Получить `start := time.Now()` для duration
  - [x] 3.2 Получить traceID из context или сгенерировать новый
  - [x] 3.3 Валидировать `cfg.InfobaseName` — при пустом значении вернуть структурированную ошибку (AC-9)
  - [x] 3.4 Создать или использовать injected RAC клиент (паттерн из Story 2.5)
  - [x] 3.5 Получить cluster info: `racClient.GetClusterInfo(ctx)`
  - [x] 3.6 Получить infobase info: `racClient.GetInfobaseInfo(ctx, clusterUUID, infobaseName)`
  - [x] 3.7 **Проверка идемпотентности**: `racClient.GetServiceModeStatus(ctx, clusterUUID, infobaseUUID)` — если уже отключён, вернуть success с `already_disabled: true` (AC-3)
  - [x] 3.8 Вызвать `racClient.DisableServiceMode(ctx, clusterUUID, infobaseUUID)`
  - [x] 3.9 **Верификация**: `racClient.VerifyServiceMode(ctx, clusterUUID, infobaseUUID, false)` — убедиться что режим отключился (AC-10)
  - [x] 3.10 Сформировать `ServiceModeDisableData` для ответа
  - [x] 3.11 Для текстового формата — вывести через `writeText`
  - [x] 3.12 Для JSON формата — вывести `output.Result` с Data, Metadata
- [x] Task 4: Определить data struct для ответа (AC: 4, 5)
  - [x] 4.1 Создать `ServiceModeDisableData` с полями: `Disabled bool`, `AlreadyDisabled bool`, `ScheduledJobsUnblocked bool`, `InfobaseName string`
  - [x] 4.2 Реализовать `writeText(w io.Writer) error` для человекочитаемого формата
- [x] Task 5: Обновить main.go (AC: 6, 7)
  - [x] 5.1 Добавить `_ "github.com/Kargones/apk-ci/internal/command/handlers/servicemodedisablehandler"` в `cmd/benadis-runner/main.go`
  - [x] 5.2 Удалить legacy case `constants.ActServiceModeDisable` из switch в main.go (NR-handler полностью заменяет legacy)
- [x] Task 6: Написать unit-тесты (AC: 11)
  - [x] 6.1 `TestServiceModeDisableHandler_Name` — проверка имени
  - [x] 6.2 `TestServiceModeDisableHandler_Description` — не пустое
  - [x] 6.3 `TestServiceModeDisableHandler_Registration` — `command.Get("nr-service-mode-disable")` успешен
  - [x] 6.4 `TestServiceModeDisableHandler_DeprecatedAlias` — `command.Get("service-mode-disable")` тоже работает
  - [x] 6.5 `TestServiceModeDisableHandler_Execute_TextOutput` — проверка текстового вывода с mock RAC
  - [x] 6.6 `TestServiceModeDisableHandler_Execute_JSONOutput` — проверка JSON с парсингом `output.Result`
  - [x] 6.7 `TestServiceModeDisableHandler_Execute_AlreadyDisabled` — идемпотентность: режим уже отключён — success + already_disabled=true
  - [x] 6.8 `TestServiceModeDisableHandler_Execute_NoInfobase` — ошибка при пустом infobase name
  - [x] 6.9 `TestServiceModeDisableHandler_Execute_RACError` — табличные тесты: ClusterError, InfobaseError, DisableError
  - [x] 6.10 `TestServiceModeDisableHandler_Execute_VerifyFailed` — ошибка верификации после отключения
  - [x] 6.11 `TestServiceModeDisableHandler_Execute_NilConfig` — nil config — ошибка
  - [x] 6.12 `TestServiceModeDisableHandler_Execute_TextErrorOutput` — ошибка в текстовом формате
  - [x] 6.13 `TestServiceModeDisableHandler_CreateRACClient_Errors` — табличные тесты: NilAppConfig, EmptyServer

## Dev Notes

### Архитектурные ограничения

- **ADR-002: Command Registry** — handler регистрируется в `init()` через `command.RegisterWithAlias`. Legacy alias `service-mode-disable`.
- **ADR-003: ISP** — использовать `rac.Client` (композитный интерфейс). Методы: `GetClusterInfo`, `GetInfobaseInfo`, `GetServiceModeStatus`, `DisableServiceMode`, `VerifyServiceMode`.
- **ADR-001: Wire DI** — на текущем этапе handler создаёт RAC клиент напрямую в Execute. Wire-провайдер будет добавлен позже.
- **Все комментарии на русском языке** (CLAUDE.md).
- **Logger только в stderr**, stdout зарезервирован для output (JSON/text).
- **НЕ менять интерфейсы RAC** — Story 2.1 и 2.2 закончены, их код не трогать.

### Паттерн реализации — следовать `servicemodeenablehandler/handler.go`

Каноничный пример NR-handler для disable: `internal/command/handlers/servicemodeenablehandler/handler.go` (Story 2.5). Disable — зеркальная операция enable. Структура handler:

```go
package servicemodedisablehandler

func init() {
    command.RegisterWithAlias(&ServiceModeDisableHandler{}, constants.ActServiceModeDisable)
}

type ServiceModeDisableHandler struct {
    racClient rac.Client // nil в production, mock в тестах
}

func (h *ServiceModeDisableHandler) Name() string {
    return constants.ActNRServiceModeDisable
}

func (h *ServiceModeDisableHandler) Description() string {
    return "Отключение сервисного режима информационной базы"
}

func (h *ServiceModeDisableHandler) Execute(ctx context.Context, cfg *config.Config) error {
    start := time.Now()
    traceID := tracing.TraceIDFromContext(ctx)
    if traceID == "" {
        traceID = tracing.GenerateTraceID()
    }
    // ... бизнес-логика ...
}
```

### Создание RAC клиента из config — КОПИРОВАТЬ паттерн из Story 2.5

Метод `createRACClient(cfg)` идентичен `servicemodeenablehandler.createRACClient`. Реализовать как отдельную копию (повторное использование будет через DI позже):

```go
func (h *ServiceModeDisableHandler) createRACClient(cfg *config.Config) (rac.Client, error) {
    if cfg.AppConfig == nil {
        return nil, fmt.Errorf("конфигурация приложения не загружена")
    }
    server := cfg.GetOneServer(cfg.InfobaseName)
    if server == "" {
        if cfg.RacConfig != nil && cfg.RacConfig.RacServer != "" {
            server = cfg.RacConfig.RacServer
        } else {
            return nil, fmt.Errorf("не удалось определить сервер для информационной базы '%s'", cfg.InfobaseName)
        }
    }
    port := strconv.Itoa(cfg.AppConfig.Rac.Port)
    if port == "0" {
        port = "1545"
    }
    timeout := time.Duration(cfg.AppConfig.Rac.Timeout) * time.Second
    if timeout == 0 {
        timeout = 30 * time.Second
    }
    opts := rac.ClientOptions{
        RACPath:      cfg.AppConfig.Paths.Rac,
        Server:       server,
        Port:         port,
        Timeout:      timeout,
        ClusterUser:  cfg.AppConfig.Users.Rac,
        ClusterPass:  "",
        InfobaseUser: cfg.AppConfig.Users.Db,
        InfobasePass: "",
    }
    if cfg.SecretConfig != nil {
        opts.ClusterPass = cfg.SecretConfig.Passwords.Rac
        opts.InfobasePass = cfg.SecretConfig.Passwords.Db
    }
    return rac.NewClient(opts)
}
```

### Ключевая логика — идемпотентность (AC-3, FR62)

**Паттерн: Check → Act → Verify** (зеркало Story 2.5 enable)

```go
// 1. Check: проверить текущий статус
status, err := racClient.GetServiceModeStatus(ctx, clusterInfo.UUID, infobaseInfo.UUID)
if err != nil {
    // Не критично — продолжаем с отключением (fail-open для check)
    log.Warn("Не удалось проверить текущий статус перед отключением", slog.String("error", err.Error()))
}

// 2. Идемпотентность: уже отключён?
if status != nil && !status.Enabled {
    log.Info("Сервисный режим уже отключён", slog.String("infobase", cfg.InfobaseName))
    data := &ServiceModeDisableData{
        Disabled:              true,
        AlreadyDisabled:       true,
        ScheduledJobsUnblocked: !status.ScheduledJobsBlocked,
        InfobaseName:          cfg.InfobaseName,
    }
    return h.outputResult(format, data, traceID, start)
}

// 3. Act: отключить
err = racClient.DisableServiceMode(ctx, clusterInfo.UUID, infobaseInfo.UUID)

// 4. Verify: убедиться
err = racClient.VerifyServiceMode(ctx, clusterInfo.UUID, infobaseInfo.UUID, false)
```

### Важное замечание про RAC DisableServiceMode

В `client.go:455-495` метод `DisableServiceMode`:
- Получает текущий статус для условного снятия блокировки регламентных заданий
- **Legacy-паттерн**: если `denied-message` оканчивается на "." — регламентные задания НЕ разблокируются (они были заблокированы отдельно от сервисного режима)
- Снимает: `sessions-deny=off`, `denied-from=`, `denied-message=`, `permission-code=`
- Условно: `scheduled-jobs-deny=off` (только если нет маркера ".")

Handler НЕ должен менять поведение `DisableServiceMode` — это реализовано в RAC client. `ScheduledJobsUnblocked` в ответе определяется после верификации, а не предполагается.

### Data struct для ответа

```go
// ServiceModeDisableData содержит данные ответа об отключении сервисного режима.
type ServiceModeDisableData struct {
    // Disabled — отключён ли сервисный режим
    Disabled bool `json:"disabled"`
    // AlreadyDisabled — был ли режим уже отключён до вызова
    AlreadyDisabled bool `json:"already_disabled"`
    // ScheduledJobsUnblocked — разблокированы ли регламентные задания
    ScheduledJobsUnblocked bool `json:"scheduled_jobs_unblocked"`
    // InfobaseName — имя информационной базы
    InfobaseName string `json:"infobase_name"`
}
```

### Формат текстового вывода

**Успешное отключение:**
```
Сервисный режим: ОТКЛЮЧЁН
Информационная база: MyBase
Регламентные задания: разблокированы
```

**Режим уже был отключён (идемпотентность):**
```
Сервисный режим: ОТКЛЮЧЁН (уже был отключён)
Информационная база: MyBase
```

**Регламентные задания не разблокированы (legacy-маркер):**
```
Сервисный режим: ОТКЛЮЧЁН
Информационная база: MyBase
Регламентные задания: не разблокированы (отдельная блокировка)
```

### Формат JSON вывода

**Успешное отключение:**
```json
{
  "status": "success",
  "command": "nr-service-mode-disable",
  "data": {
    "disabled": true,
    "already_disabled": false,
    "scheduled_jobs_unblocked": true,
    "infobase_name": "MyBase"
  },
  "metadata": {
    "duration_ms": 250,
    "trace_id": "abc123def456...",
    "api_version": "v1"
  }
}
```

**Идемпотентный вызов:**
```json
{
  "status": "success",
  "command": "nr-service-mode-disable",
  "data": {
    "disabled": true,
    "already_disabled": true,
    "scheduled_jobs_unblocked": true,
    "infobase_name": "MyBase"
  },
  "metadata": {
    "duration_ms": 120,
    "trace_id": "abc123...",
    "api_version": "v1"
  }
}
```

### Формат вывода ошибки

Идентичен паттерну Story 2.5 (`writeError`):
```json
{
  "status": "error",
  "command": "nr-service-mode-disable",
  "error": {
    "code": "CONFIG.INFOBASE_MISSING",
    "message": "Не указано имя информационной базы (BR_INFOBASE_NAME)"
  },
  "metadata": {
    "duration_ms": 1,
    "trace_id": "abc123...",
    "api_version": "v1"
  }
}
```

### Deprecated alias — миграция legacy

`RegisterWithAlias` создаёт `DeprecatedBridge`. Когда CI/CD вызывает `service-mode-disable`, мост:
1. Логирует warning в stderr: "Command deprecated, use nr-service-mode-disable"
2. Делегирует выполнение в `ServiceModeDisableHandler.Execute`

После регистрации alias — legacy switch-case в main.go для `ActServiceModeDisable` станет недоступен (registry перехватит). Поэтому legacy case НУЖНО УДАЛИТЬ из main.go.

### Определение ScheduledJobsUnblocked

После успешного `DisableServiceMode` и `VerifyServiceMode`, нужно определить были ли задания разблокированы:

```go
// После DisableServiceMode + VerifyServiceMode
// Получаем актуальный статус для определения состояния заданий
postStatus, err := racClient.GetServiceModeStatus(ctx, clusterInfo.UUID, infobaseInfo.UUID)
scheduledJobsUnblocked := true // по умолчанию
if err == nil && postStatus != nil {
    scheduledJobsUnblocked = !postStatus.ScheduledJobsBlocked
} else {
    // Если не удалось получить статус — считаем что разблокированы (fail-open)
    log.Warn("Не удалось проверить статус регламентных заданий после отключения", slog.String("error", err.Error()))
}
```

### Тестирование — паттерн из servicemodeenablehandler

```go
func TestServiceModeDisableHandler_Execute_JSONOutput(t *testing.T) {
    t.Setenv("BR_OUTPUT_FORMAT", "json")

    mock := ractest.NewMockRACClientWithServiceMode(true, "Режим обслуживания", 0)
    // Mock DisableServiceMode — успех
    mock.DisableServiceModeFunc = func(_ context.Context, _, _ string) error {
        return nil
    }
    // Mock VerifyServiceMode — успех
    mock.VerifyServiceModeFunc = func(_ context.Context, _, _ string, _ bool) error {
        return nil
    }

    h := &ServiceModeDisableHandler{racClient: mock}
    cfg := &config.Config{InfobaseName: "TestBase"}

    out := testutil.CaptureStdout(t, func() {
        err := h.Execute(context.Background(), cfg)
        require.NoError(t, err)
    })

    var result output.Result
    err := json.Unmarshal([]byte(out), &result)
    require.NoError(t, err)
    assert.Equal(t, "success", result.Status)
}

func TestServiceModeDisableHandler_Execute_AlreadyDisabled(t *testing.T) {
    t.Setenv("BR_OUTPUT_FORMAT", "json")

    // Mock: режим уже отключён (Enabled: false)
    mock := ractest.NewMockRACClientWithServiceMode(false, "", 0)
    h := &ServiceModeDisableHandler{racClient: mock}
    cfg := &config.Config{InfobaseName: "TestBase"}

    out := testutil.CaptureStdout(t, func() {
        err := h.Execute(context.Background(), cfg)
        require.NoError(t, err)
    })

    var result output.Result
    err := json.Unmarshal([]byte(out), &result)
    require.NoError(t, err)
    assert.Equal(t, "success", result.Status)

    data := result.Data.(map[string]interface{})
    assert.True(t, data["already_disabled"].(bool))
}
```

### Существующий код (НЕ МЕНЯТЬ)

| Файл | Что содержит | Статус |
|------|-------------|--------|
| `internal/adapter/onec/rac/interfaces.go` | Client, ServiceModeManager, DisableServiceMode, VerifyServiceMode | Story 2.1 (done) |
| `internal/adapter/onec/rac/client.go` | DisableServiceMode:455, VerifyServiceMode, GetServiceModeStatus | Story 2.2 (done) |
| `internal/adapter/onec/rac/client_test.go` | 34 unit-теста | Story 2.2 (done) |
| `internal/adapter/onec/rac/ractest/mock.go` | MockRACClient с DisableServiceModeFunc, VerifyServiceModeFunc | Story 2.1 (done) |
| `internal/command/registry.go` | Register, RegisterWithAlias, Get, All | Epic 1 (done) |
| `internal/command/handler.go` | Interface Handler | Epic 1 (done) |
| `internal/command/deprecated.go` | DeprecatedBridge | Epic 1 (done) |
| `internal/command/handlers/servicemodeenablehandler/` | handler.go, handler_test.go | Story 2.5 (done) |
| `internal/command/handlers/servicemodestatushandler/` | handler.go, handler_test.go | Story 2.3/2.4 (done) |
| `internal/command/handlers/forcedisconnecthandler/` | handler.go, handler_test.go | Story 2.6 (done) |

### Файлы на изменение

| Файл | Действие | Описание |
|------|----------|----------|
| `internal/constants/constants.go` | изменить | Добавить `ActNRServiceModeDisable` |
| `cmd/benadis-runner/main.go` | изменить | Добавить blank import `servicemodedisablehandler`, удалить legacy case `ActServiceModeDisable` |
| `internal/command/handlers/servicemodedisablehandler/handler.go` | новый | Handler struct, init(), Execute, writeText, writeError, outputResult, createRACClient |
| `internal/command/handlers/servicemodedisablehandler/handler_test.go` | новый | Unit-тесты |

### Файлы НЕ ТРОГАТЬ

- `internal/adapter/onec/rac/interfaces.go` — Story 2.1 (done)
- `internal/adapter/onec/rac/client.go` — Story 2.2 (done)
- `internal/adapter/onec/rac/client_test.go` — Story 2.2 (done)
- `internal/adapter/onec/rac/ractest/mock.go` — Story 2.1 (done)
- `internal/command/registry.go`, `handler.go`, `deprecated.go` — не менять
- `internal/command/handlers/servicemodestatushandler/` — не менять
- `internal/command/handlers/servicemodeenablehandler/` — не менять
- `internal/command/handlers/forcedisconnecthandler/` — не менять
- `internal/pkg/output/` — не менять
- `internal/pkg/tracing/` — не менять
- Legacy код: `internal/rac/`, `internal/servicemode/`, `internal/app/app.go`

### Что НЕ делать

- НЕ менять `interfaces.go`, `client.go`, `client_test.go`, `ractest/mock.go` (Story 2.1, 2.2)
- НЕ менять legacy код в `internal/rac/`, `internal/servicemode/`
- НЕ менять `registry.go`, `handler.go`, `deprecated.go`
- НЕ добавлять Wire-провайдеры (будет позже)
- НЕ реализовывать state-aware execution (Story 2.8) — но AC-3 (идемпотентность) реализовать в рамках этой story
- НЕ менять поведение `DisableServiceMode` для legacy-паттерна с маркером "."
- НЕ добавлять retry logic в handler
- НЕ добавлять force disconnect в disable (это отдельная команда Story 2.6)

### Project Structure Notes

- **Новый пакет**: `internal/command/handlers/servicemodedisablehandler/`
- **Новый файл**: `handler.go` — handler struct + init() + Execute + writeText + writeError + outputResult + createRACClient
- **Новый файл**: `handler_test.go` — unit-тесты
- **Изменение**: `internal/constants/constants.go` — добавить `ActNRServiceModeDisable`
- **Изменение**: `cmd/benadis-runner/main.go` — добавить blank import, удалить legacy case `ActServiceModeDisable`
- Следует паттерну: один handler = один пакет в `handlers/`
- Именование пакета: `servicemodedisablehandler` (аналогично `servicemodeenablehandler`)

### References

- [Source: _bmad-output/project-planning-artifacts/epics/epic-2-service-mode.md#Story 2.7]
- [Source: _bmad-output/project-planning-artifacts/architecture.md#Command Registry, Output Format, Error Handling]
- [Source: internal/command/handlers/servicemodeenablehandler/handler.go — каноничный пример NR-handler (зеркальная операция)]
- [Source: internal/command/handler.go — interface Handler]
- [Source: internal/command/registry.go — Register, RegisterWithAlias]
- [Source: internal/command/deprecated.go — DeprecatedBridge]
- [Source: internal/adapter/onec/rac/interfaces.go — Client, ServiceModeManager (DisableServiceMode, VerifyServiceMode)]
- [Source: internal/adapter/onec/rac/client.go:455-495 — DisableServiceMode implementation (legacy-паттерн с маркером ".")]
- [Source: internal/adapter/onec/rac/client.go:539-558 — VerifyServiceMode implementation]
- [Source: internal/adapter/onec/rac/ractest/mock.go — MockRACClient, DisableServiceModeFunc, VerifyServiceModeFunc]
- [Source: internal/pkg/output/result.go — Result, Metadata, ErrorInfo structs]
- [Source: internal/pkg/tracing/ — TraceIDFromContext, GenerateTraceID]
- [Source: internal/constants/constants.go — ActServiceModeDisable (legacy), ActNRServiceModeEnable (паттерн NR)]
- [Source: cmd/benadis-runner/main.go:90-109 — legacy case ActServiceModeDisable (УДАЛИТЬ)]
- [Source: _bmad-output/implementation-artifacts/stories/2-5-nr-service-mode-enable.md — зеркальная story с learnings]
- [Source: _bmad-output/implementation-artifacts/stories/2-6-force-disconnect-sessions.md — предыдущая story с learnings]

### Git Intelligence

Последние коммиты релевантны текущей story:
- `3b463fc fix(forcedisconnecthandler): replace time.Sleep with context-aware delay`
- `674f709 feat(command): add nr-force-disconnect-sessions handler`
- `da1a64e feat(command): implement nr-service-mode-enable handler` — зеркальная NR-команда (паттерн handler)
- `add95cd feat(command): add nr-service-mode-status command with session info`
- `d0720ab feat(rac): add infobase authentication and improve service mode handling` — DisableServiceMode в RAC client

**Паттерны из git:**
- Commit convention: `feat(scope): description` на английском
- Тесты добавляются вместе с кодом в одном коммите
- Mock обновляется в том же коммите что и использующий его код (в данной story mock НЕ меняется)
- Удаление legacy case из main.go — в том же коммите что и blank import

### Previous Story Intelligence (Story 2.5 + 2.6)

**Learnings из Story 2.5 (enable — зеркальная операция):**
1. **Code Review** нашёл: дублированный slog, комментарии про env-переменные, warn при невозможности подсчёта сессий
2. **Рекомендуемый подход для тестируемости**: `racClient rac.Client` как поле struct (nil в production, mock в тестах)
3. **Паттерн ошибок**: использовать `output.Result` со `Status: "error"` и `ErrorInfo`, затем return error
4. **writeError**: для text формата — "Ошибка: msg\nКод: code", для JSON — output.Result
5. **createRACClient**: копировать паттерн (DI будет позже)
6. Все 15 тестов прошли без регрессий

**Learnings из Story 2.6 (force disconnect):**
1. `make([]Type, 0)` для гарантии `[]` в JSON (не `null`) — НЕ нужно для disable (нет массивов)
2. Табличные тесты для error paths — обязательно
3. time.Sleep заменён на select с ctx.Done() — НЕ нужно для disable (нет delay)
4. 20 тестов PASS, 0 lint issues

**Learnings из Story 2.3/2.4:**
1. Graceful degradation при ошибках получения дополнительных данных
2. Табличные тесты для error paths — обязательно

**Применимость к Story 2.7:**
- Disable — зеркальная операция enable, поэтому код проще (нет message, permission_code, terminate_sessions)
- Сразу добавить slog-логирование с trace_id для всех этапов (check, disable, verify)
- Сразу удалить legacy case из main.go при добавлении blank import
- Добавить табличные тесты для error paths (ClusterError, InfobaseError, DisableError, VerifyError)
- Реализовать writeError идентично Story 2.5
- Использовать `racClient rac.Client` как поле struct для testability
- Обязательно проверить `ScheduledJobsUnblocked` через GetServiceModeStatus после disable

### Коды ошибок

```
CONFIG.INFOBASE_MISSING      — BR_INFOBASE_NAME не указана
RAC.CLIENT_CREATE_FAILED     — ошибка создания RAC клиента
RAC.CLUSTER_FAILED           — не удалось получить информацию о кластере
RAC.INFOBASE_FAILED          — информационная база не найдена
RAC.DISABLE_FAILED           — ошибка при отключении режима
RAC.VERIFY_FAILED            — верификация не прошла
```

### Review Follow-ups (AI)

- [ ] [AI-Review][HIGH] Извлечь `createRACClient` в shared utility (затрагивает 4+ handler-а — вне scope текущей story, требует отдельной задачи)
- [ ] [AI-Review][MEDIUM] GetServiceModeStatus внутри вызывает GetSessions — лишний subprocess для подсчёта ActiveSessions [handler.go:191]
- [ ] [AI-Review][MEDIUM] AlreadyDisabled check: fail-open при ошибке GetServiceModeStatus добавляет latency без benefit [handler.go:144-148,170-178]
- [ ] [AI-Review][MEDIUM] Нет аналогичного теста PostStatusNilNil в enable handler [handler_test.go:345-387]
- [ ] [AI-Review][MEDIUM] ScheduledJobsUnblocked default=true при ошибке — optimistic default может ввести в заблуждение [handler.go:190]
- [ ] [AI-Review][LOW] writeText для AlreadyDisabled не выводит info о регламентных заданиях — расхождение JSON и text output [handler.go:53,164]
- [ ] [AI-Review][LOW] Нет plan-only теста TestServiceModeDisableHandler_PlanOnly [handler_test.go]

### Review Follow-ups (AI Code Review #34)

- [ ] [AI-Review][MEDIUM] GetServiceModeStatus вызывается дважды (pre+post check) = 4 subprocess calls [handler.go]
- [ ] [AI-Review][MEDIUM] ScheduledJobsUnblocked default=true при ошибке post-check — оптимистичный но misleading [handler.go:190]

## Senior Developer Review (AI)

**Reviewer:** Xor (via Claude Opus 4.5)
**Date:** 2026-01-28

**Issues Found:** 2 High, 3 Medium, 2 Low
**Issues Fixed:** 5 (H-2, M-1, M-2, M-3, L-2)
**Action Items Created:** 1 (H-1 — вне scope story)

**Fixes Applied:**
1. **H-2** (handler_test.go): Замена `int` callCount на `atomic.Int32` для защиты от race condition
2. **M-1** (handler.go:177): Добавлен `slog.String("error", postErr.Error())` в warn-лог после disable
3. **M-2** (handler_test.go): Добавлен тест `TestServiceModeDisableHandler_Execute_StatusCheckFailOpen` — покрытие fail-open пути
4. **M-3** (handler_test.go): InfobaseError mock переведён на `NewMockRACClient()` вместо struct literal — явные дефолтные функции
5. **L-2** (handler_test.go): Удалён дублированный `t.Setenv` из цикла табличных тестов

**Result:** 14 тестов PASS, 0 lint issues

### Review #2 (2026-01-28)

**Reviewer:** Xor (via Claude Opus 4.5)
**Date:** 2026-01-28

**Issues Found:** 1 High, 2 Medium, 2 Low
**Issues Fixed:** 2 (H-1, M-1)
**Action Items:** M-2 уже зафиксирован ранее, L-1/L-2 информационные

**Fixes Applied:**
1. **H-1** (handler.go:174-179): Потенциальный nil pointer dereference — `postErr.Error()` вызывался в else-ветке, где `postErr` мог быть nil (когда `GetServiceModeStatus` возвращает `(nil, nil)`). Исправлено: `else` заменён на `else if postErr != nil`
2. **M-1** (handler_test.go): Добавлен тест `TestServiceModeDisableHandler_Execute_PostStatusNilNil` — проверяет что handler не паникует при `(nil, nil)` от post-check `GetServiceModeStatus`

**Not Fixed (informational):**
- **M-2**: Дублирование `createRACClient` — уже зафиксировано в Review Follow-ups
- **L-1**: File List не включает sprint-status.yaml (артефакт, не исходный код)
- **L-2**: Аналогичный паттерн в enable handler (вне scope)

**Result:** 17 тестов PASS (включая подтесты), 0 lint issues, 0 race conditions

## Dev Agent Record

### Agent Model Used

Claude Opus 4.5 (claude-opus-4-5-20251101)

### Debug Log References

Нет проблем — все 13 тестов прошли с первого запуска.

### Completion Notes List

- Реализован NR-handler `nr-service-mode-disable` — зеркальная операция `nr-service-mode-enable`
- Паттерн Check → Act → Verify: идемпотентность (AC-3), верификация после отключения (AC-10)
- `ScheduledJobsUnblocked` определяется через `GetServiceModeStatus` после `DisableServiceMode` (не предполагается)
- Legacy case `ActServiceModeDisable` удалён из main.go — NR-handler полностью заменяет через registry + DeprecatedBridge
- 14 unit-тестов: Name, Description, Registration, DeprecatedAlias, TextOutput, JSONOutput, AlreadyDisabled (JSON+Text), ScheduledJobsNotUnblocked, NoInfobase, RACError (3 табличных), VerifyFailed, StatusCheckFailOpen, NilConfig, TextErrorOutput, CreateRACClient (2 табличных)
- 0 lint issues в новых/изменённых файлах
- Предсуществующие тесты в `internal/app` (TestExtensionPublish_MissingEnvVars) падают из-за DNS — не связаны с данной story

### Change Log

- 2026-01-28: Реализована Story 2.7 — NR-команда nr-service-mode-disable с полной поддержкой идемпотентности, верификации, текстового и JSON вывода, deprecated alias и 13 unit-тестами
- 2026-01-28: Code Review — исправлены 5 issues (race condition в mock, missing error в логе, недостающий тест fail-open, хрупкий mock, дублированный t.Setenv). Добавлен 1 action item (shared createRACClient). 14 тестов PASS, 0 lint
- 2026-01-28: Code Review #2 — исправлены 2 issues: nil pointer dereference в post-check (H-1), добавлен тест PostStatusNilNil (M-1). 17 тестов PASS, 0 lint, 0 race
- 2026-02-04: Code Review (AI) Batch Epic 2 — M-1: добавлено предупреждение при пустых паролях RAC в createRACClient
- 2026-02-04: Code Review (AI) Epic 2 Final — M-2: добавлено предупреждение при timeout > 5 минут в createRACClient

### File List

- `internal/constants/constants.go` — добавлена константа `ActNRServiceModeDisable`
- `internal/command/handlers/servicemodedisablehandler/handler.go` — новый: handler struct, init(), Execute, writeText, writeError, outputResult, createRACClient
- `internal/command/handlers/servicemodedisablehandler/handler_test.go` — новый: 17 unit-тестов (включая подтесты)
- `cmd/benadis-runner/main.go` — добавлен blank import servicemodedisablehandler, удалён legacy case ActServiceModeDisable
