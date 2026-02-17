# Story 2.5: nr-service-mode-enable

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a DevOps-инженер,
I want включить сервисный режим информационной базы через NR-команду,
So that я могу безопасно выполнять операции с базой, заблокировав доступ пользователей и регламентные задания.

## Acceptance Criteria

1. **AC-1**: Команда `BR_COMMAND=nr-service-mode-enable BR_INFOBASE_NAME=MyBase` включает сервисный режим
2. **AC-2**: Сервисный режим включён: сессии запрещены (`sessions-deny=on`), регламентные задания заблокированы (`scheduled-jobs-deny=on`)
3. **AC-3**: `BR_SERVICE_MODE_MESSAGE` — кастомное сообщение для пользователей (по умолчанию: "Система находится в режиме обслуживания")
4. **AC-4**: `BR_SERVICE_MODE_PERMISSION_CODE` — код разрешения (по умолчанию: "ServiceMode")
5. **AC-5**: Команда идемпотентна: повторный вызов при уже включённом режиме → success + `"already_enabled": true` (FR62)
6. **AC-6**: При `BR_TERMINATE_SESSIONS=true` — все активные сессии завершаются после включения режима
7. **AC-7**: Команда зарегистрирована в Command Registry через `init()` и доступна через `command.Get("nr-service-mode-enable")`
8. **AC-8**: Legacy команда `service-mode-enable` привязана как deprecated alias через `RegisterWithAlias`
9. **AC-9**: JSON output содержит: `enabled`, `message`, `permission_code`, `scheduled_jobs_blocked`, `already_enabled`, `terminated_sessions_count` + metadata
10. **AC-10**: Text output содержит человекочитаемый статус включения
11. **AC-11**: trace_id присутствует в логах и JSON-ответе
12. **AC-12**: При отсутствии `BR_INFOBASE_NAME` — возврат структурированной ошибки (code + message)
13. **AC-13**: Unit-тесты покрывают: регистрацию, text output, JSON output, идемпотентность, terminate sessions, ошибки

## Tasks / Subtasks

- [x] Task 1: Добавить константу `ActNRServiceModeEnable` (AC: 7, 8)
  - [x] 1.1 В `internal/constants/constants.go` добавить `ActNRServiceModeEnable = "nr-service-mode-enable"`
- [x] Task 2: Создать пакет handler `internal/command/handlers/servicemodeenablehandler/` (AC: 7, 8)
  - [x] 2.1 Создать `handler.go` со struct `ServiceModeEnableHandler`
  - [x] 2.2 Реализовать `Name() string` — возвращает `constants.ActNRServiceModeEnable`
  - [x] 2.3 Реализовать `Description() string` — "Включение сервисного режима информационной базы"
  - [x] 2.4 В `init()` зарегистрировать через `command.RegisterWithAlias(&ServiceModeEnableHandler{}, constants.ActServiceModeEnable)` — привязать legacy alias
- [x] Task 3: Реализовать `Execute` метод (AC: 1-6, 9-12)
  - [x] 3.1 Получить `start := time.Now()` для duration
  - [x] 3.2 Получить traceID из context или сгенерировать новый
  - [x] 3.3 Валидировать `cfg.InfobaseName` — при пустом значении вернуть структурированную ошибку
  - [x] 3.4 Прочитать параметры: `BR_SERVICE_MODE_MESSAGE` (по умолчанию `constants.DefaultServiceModeMessage`), `BR_SERVICE_MODE_PERMISSION_CODE` (по умолчанию "ServiceMode"), `cfg.TerminateSessions`
  - [x] 3.5 Создать или использовать injected RAC клиент (паттерн из Story 2.3)
  - [x] 3.6 Получить cluster info: `racClient.GetClusterInfo(ctx)`
  - [x] 3.7 Получить infobase info: `racClient.GetInfobaseInfo(ctx, clusterUUID, infobaseName)`
  - [x] 3.8 **Проверка идемпотентности**: `racClient.GetServiceModeStatus(ctx, clusterUUID, infobaseUUID)` — если уже включён, вернуть success с `already_enabled: true`
  - [x] 3.9 Вызвать `racClient.EnableServiceMode(ctx, clusterUUID, infobaseUUID, terminateSessions)`
  - [x] 3.10 **Верификация**: `racClient.VerifyServiceMode(ctx, clusterUUID, infobaseUUID, true)` — убедиться что режим включился
  - [x] 3.11 Получить количество завершённых сессий (если terminateSessions=true) через проверку sessions до/после
  - [x] 3.12 Сформировать `ServiceModeEnableData` для ответа
  - [x] 3.13 Для текстового формата — вывести человекочитаемый текст через `writeText`
  - [x] 3.14 Для JSON формата — вывести `output.Result` с Data, Metadata
- [x] Task 4: Определить data struct для ответа (AC: 9, 10)
  - [x] 4.1 Создать `ServiceModeEnableData` с полями: `Enabled bool`, `Message string`, `PermissionCode string`, `ScheduledJobsBlocked bool`, `AlreadyEnabled bool`, `TerminatedSessionsCount int`, `InfobaseName string`
  - [x] 4.2 Реализовать `writeText(w io.Writer) error` для человекочитаемого формата
- [x] Task 5: Зарегистрировать blank import в main.go (AC: 7)
  - [x] 5.1 Добавить `_ "github.com/Kargones/apk-ci/internal/command/handlers/servicemodeenablehandler"` в `cmd/apk-ci/main.go`
  - [x] 5.2 Удалить legacy case `constants.ActServiceModeEnable` из switch в main.go (NR-handler полностью заменяет legacy)
- [x] Task 6: Написать unit-тесты (AC: 13)
  - [x] 6.1 `TestServiceModeEnableHandler_Name` — проверка имени
  - [x] 6.2 `TestServiceModeEnableHandler_Description` — не пустое
  - [x] 6.3 `TestServiceModeEnableHandler_Registration` — `command.Get("nr-service-mode-enable")` успешен
  - [x] 6.4 `TestServiceModeEnableHandler_DeprecatedAlias` — `command.Get("service-mode-enable")` тоже работает
  - [x] 6.5 `TestServiceModeEnableHandler_Execute_TextOutput` — проверка текстового вывода с mock RAC
  - [x] 6.6 `TestServiceModeEnableHandler_Execute_JSONOutput` — проверка JSON с парсингом `output.Result`
  - [x] 6.7 `TestServiceModeEnableHandler_Execute_AlreadyEnabled` — идемпотентность: режим уже включён → success + already_enabled=true
  - [x] 6.8 `TestServiceModeEnableHandler_Execute_WithTerminateSessions` — при BR_TERMINATE_SESSIONS=true сессии завершаются
  - [x] 6.9 `TestServiceModeEnableHandler_Execute_NoInfobase` — ошибка при пустом infobase name
  - [x] 6.10 `TestServiceModeEnableHandler_Execute_RACError` — обработка ошибок RAC (табличные тесты для ClusterError, InfobaseError, EnableError)
  - [x] 6.11 `TestServiceModeEnableHandler_Execute_VerifyFailed` — ошибка верификации после включения

### Review Follow-ups (AI)

- [ ] [AI-Review][HIGH] newEnableMock использует callCount int32 БЕЗ atomic операций — data race при t.Parallel() [handler_test.go:455-456]
- [ ] [AI-Review][HIGH] BR_SERVICE_MODE_MESSAGE/PERMISSION_CODE отображаются в response, но НЕ передаются в RAC — misleading поведение [handler.go:117-128]
- [ ] [AI-Review][MEDIUM] sessionsCount берётся из status.ActiveSessions ДО вызова EnableServiceMode — приблизительное значение [handler.go:194-200]
- [ ] [AI-Review][MEDIUM] TerminateSessions=true но EnableServiceMode внутри RAC проглатывает ошибку terminate [handler.go:130]
- [ ] [AI-Review][MEDIUM] Нет теста для network-ошибки VerifyServiceMode (только logical mismatch) [handler_test.go:352-384]
- [ ] [AI-Review][LOW] TestServiceModeEnableHandler_Execute_CustomEnvParams не проверяет что env НЕ передан в RAC [handler_test.go:419-445]

### Review Follow-ups (AI Code Review #34)

- [ ] [AI-Review][HIGH] EnableServiceMode проглатывает ошибку TerminateAllSessions — partial failure не отражён [rac/client.go:492-497]
- [ ] [AI-Review][MEDIUM] BR_SERVICE_MODE_MESSAGE/PERMISSION_CODE не передаются в RAC — используются hardcoded [handler.go:117-128, client.go:461,479]
- [ ] [AI-Review][MEDIUM] sessionsCount приблизительный — берётся ДО EnableServiceMode [handler.go:194-200]

## Dev Notes

### Архитектурные ограничения

- **ADR-002: Command Registry** — handler регистрируется в `init()` через `command.RegisterWithAlias`. Legacy alias `service-mode-enable`.
- **ADR-003: ISP** — использовать `rac.Client` (композитный интерфейс). Методы: `GetClusterInfo`, `GetInfobaseInfo`, `GetServiceModeStatus`, `EnableServiceMode`, `VerifyServiceMode`.
- **ADR-001: Wire DI** — на текущем этапе handler создаёт RAC клиент напрямую в Execute. Wire-провайдер будет добавлен позже (не в этой story).
- **Все комментарии на русском языке** (CLAUDE.md).
- **Logger только в stderr**, stdout зарезервирован для output (JSON/text).
- **НЕ менять интерфейсы RAC** — Story 2.1 и 2.2 закончены, их код не трогать.

### Паттерн реализации — следовать `servicemodestatushandler/handler.go`

Каноничный пример NR-handler в этом epic: `internal/command/handlers/servicemodestatushandler/handler.go` (Story 2.3/2.4). Структура handler:

```go
package servicemodeenablehandler

func init() {
    command.RegisterWithAlias(&ServiceModeEnableHandler{}, constants.ActServiceModeEnable)
}

type ServiceModeEnableHandler struct {
    racClient rac.Client // nil в production, mock в тестах
}

func (h *ServiceModeEnableHandler) Name() string {
    return constants.ActNRServiceModeEnable
}

func (h *ServiceModeEnableHandler) Description() string {
    return "Включение сервисного режима информационной базы"
}

func (h *ServiceModeEnableHandler) Execute(ctx context.Context, cfg *config.Config) error {
    start := time.Now()
    traceID := tracing.TraceIDFromContext(ctx)
    if traceID == "" {
        traceID = tracing.GenerateTraceID()
    }
    // ... бизнес-логика ...
}
```

### Создание RAC клиента из config — КОПИРОВАТЬ паттерн из Story 2.3

Метод `createRACClient(cfg)` идентичен `servicemodestatushandler.createRACClient`. Реализовать как отдельную копию (повторное использование будет через DI позже):

```go
func (h *ServiceModeEnableHandler) createRACClient(cfg *config.Config) (rac.Client, error) {
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
    // ... port, timeout, opts как в servicemodestatushandler ...
}
```

### Ключевая логика — идемпотентность (AC-5, FR62)

**Паттерн: Check → Act → Verify**

```go
// 1. Check: проверить текущий статус
status, err := racClient.GetServiceModeStatus(ctx, clusterInfo.UUID, infobaseInfo.UUID)
if err != nil {
    // Не критично — продолжаем с включением (fail-open для check)
    log.Warn("Не удалось проверить текущий статус перед включением", "error", err)
}

// 2. Идемпотентность: уже включён?
if status != nil && status.Enabled {
    log.Info("Сервисный режим уже включён", "infobase", cfg.InfobaseName)
    data := &ServiceModeEnableData{
        Enabled:        true,
        AlreadyEnabled: true,
        Message:        status.Message,
        // ...
    }
    // Вернуть success с already_enabled=true
    return outputResult(format, data, traceID, start)
}

// 3. Act: включить
err = racClient.EnableServiceMode(ctx, clusterInfo.UUID, infobaseInfo.UUID, cfg.TerminateSessions)

// 4. Verify: убедиться
err = racClient.VerifyServiceMode(ctx, clusterInfo.UUID, infobaseInfo.UUID, true)
```

### Чтение env-переменных для параметров

```go
// BR_SERVICE_MODE_MESSAGE — кастомное сообщение
message := os.Getenv("BR_SERVICE_MODE_MESSAGE")
if message == "" {
    message = constants.DefaultServiceModeMessage
}

// BR_SERVICE_MODE_PERMISSION_CODE — код разрешения
permissionCode := os.Getenv("BR_SERVICE_MODE_PERMISSION_CODE")
if permissionCode == "" {
    permissionCode = "ServiceMode"
}

// BR_TERMINATE_SESSIONS — завершение сессий (уже в cfg.TerminateSessions)
terminateSessions := cfg.TerminateSessions
```

**Важно**: `BR_SERVICE_MODE_MESSAGE` и `BR_SERVICE_MODE_PERMISSION_CODE` НЕ используются handler напрямую для RAC вызова. RAC client (`EnableServiceMode`) уже использует `constants.DefaultServiceModeMessage` и захардкоженный "ServiceMode" внутри `client.go:422-431`. Handler читает env-переменные для вывода в response, но сам RAC вызов не принимает message/permission_code параметры. Если нужно передавать кастомные значения в RAC — потребуется расширение `EnableServiceMode` signature, что выходит за рамки этой story. На текущем этапе env-переменные используются только для отчёта в output.

### Важное замечание про RAC EnableServiceMode

В `client.go:404-449` метод `EnableServiceMode`:
- Сам проверяет состояние регламентных заданий и добавляет маркер "." (legacy-паттерн)
- Использует захардкоженное `constants.DefaultServiceModeMessage` и `"ServiceMode"` permission code
- При `terminateSessions=true` вызывает `TerminateAllSessions`
- НЕ принимает message и permission_code как параметры

Handler НЕ должен менять signature `EnableServiceMode` — это нарушит Story 2.1/2.2 интерфейсы. Env-переменные `BR_SERVICE_MODE_MESSAGE` и `BR_SERVICE_MODE_PERMISSION_CODE` используются только для информационного вывода.

### Data struct для ответа

```go
type ServiceModeEnableData struct {
    Enabled                 bool   `json:"enabled"`
    AlreadyEnabled          bool   `json:"already_enabled"`
    Message                 string `json:"message"`
    PermissionCode          string `json:"permission_code"`
    ScheduledJobsBlocked    bool   `json:"scheduled_jobs_blocked"`
    TerminatedSessionsCount int    `json:"terminated_sessions_count"`
    InfobaseName            string `json:"infobase_name"`
}
```

### Формат текстового вывода

**Успешное включение:**
```
Сервисный режим: ВКЛЮЧЁН
Информационная база: MyBase
Сообщение: "Система находится в режиме обслуживания"
Код разрешения: ServiceMode
Регламентные задания: заблокированы
Завершено сессий: 3
```

**Режим уже был включён (идемпотентность):**
```
Сервисный режим: ВКЛЮЧЁН (уже был включён)
Информационная база: MyBase
Сообщение: "Система находится в режиме обслуживания"
```

**При terminateSessions=false (по умолчанию), строка "Завершено сессий:" не выводится.**

### Формат JSON вывода

**Успешное включение:**
```json
{
  "status": "success",
  "command": "nr-service-mode-enable",
  "data": {
    "enabled": true,
    "already_enabled": false,
    "message": "Система находится в режиме обслуживания",
    "permission_code": "ServiceMode",
    "scheduled_jobs_blocked": true,
    "terminated_sessions_count": 3,
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
  "command": "nr-service-mode-enable",
  "data": {
    "enabled": true,
    "already_enabled": true,
    "message": "Система находится в режиме обслуживания",
    "permission_code": "ServiceMode",
    "scheduled_jobs_blocked": true,
    "terminated_sessions_count": 0,
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

Идентичен паттерну Story 2.3 (`writeError`):
```json
{
  "status": "error",
  "command": "nr-service-mode-enable",
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

`RegisterWithAlias` создаёт `DeprecatedBridge`. Когда CI/CD вызывает `service-mode-enable`, мост:
1. Логирует warning в stderr: "Command deprecated, use nr-service-mode-enable"
2. Делегирует выполнение в `ServiceModeEnableHandler.Execute`

После регистрации alias — legacy switch-case в main.go для `ActServiceModeEnable` станет недоступен (registry перехватит). Поэтому legacy case НУЖНО УДАЛИТЬ из main.go (аналогично Story 2.3 удалила case для status).

### Тестирование — паттерн из servicemodestatushandler

```go
func TestServiceModeEnableHandler_Execute_JSONOutput(t *testing.T) {
    t.Setenv("BR_OUTPUT_FORMAT", "json")

    mock := ractest.NewMockRACClient()
    // Mock GetServiceModeStatus — режим выключен
    mock.GetServiceModeStatusFunc = func(_ context.Context, _, _ string) (*rac.ServiceModeStatus, error) {
        return &rac.ServiceModeStatus{Enabled: false}, nil
    }
    // Mock EnableServiceMode — успех
    mock.EnableServiceModeFunc = func(_ context.Context, _, _ string, _ bool) error {
        return nil
    }
    // Mock VerifyServiceMode — успех
    mock.VerifyServiceModeFunc = func(_ context.Context, _, _ string, _ bool) error {
        return nil
    }

    h := &ServiceModeEnableHandler{racClient: mock}
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

func TestServiceModeEnableHandler_Execute_AlreadyEnabled(t *testing.T) {
    t.Setenv("BR_OUTPUT_FORMAT", "json")

    mock := ractest.NewMockRACClientWithServiceMode(true, "Режим обслуживания", 0)
    h := &ServiceModeEnableHandler{racClient: mock}
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
    assert.True(t, data["already_enabled"].(bool))
}
```

### Подсчёт завершённых сессий

При `terminateSessions=true`, можно подсчитать завершённые сессии:
1. До `EnableServiceMode` — получить `sessionsCount` через `GetSessions` (count)
2. `EnableServiceMode` внутри уже вызывает `TerminateAllSessions`
3. Записать `sessionsCount` в `TerminatedSessionsCount`

При `terminateSessions=false` → `TerminatedSessionsCount = 0`.

Альтернативный подход (проще): не считать отдельно, а использовать `status.ActiveSessions` из GetServiceModeStatus (полученный при idempotency check). Если режим не был включён — `ActiveSessions` это количество сессий на момент проверки, они все будут завершены. Это приблизительное значение, но достаточное для отчёта.

### Существующий код (НЕ МЕНЯТЬ)

| Файл | Что содержит | Статус |
|------|-------------|--------|
| `internal/adapter/onec/rac/interfaces.go` | Client, ServiceModeManager, EnableServiceMode, VerifyServiceMode | Story 2.1 (done) |
| `internal/adapter/onec/rac/client.go` | EnableServiceMode, VerifyServiceMode, GetServiceModeStatus | Story 2.2 (done) |
| `internal/adapter/onec/rac/client_test.go` | 34 unit-теста | Story 2.2 (done) |
| `internal/adapter/onec/rac/ractest/mock.go` | MockRACClient с EnableServiceModeFunc, VerifyServiceModeFunc | Story 2.1 (done) |
| `internal/command/registry.go` | Register, RegisterWithAlias, Get, All | Epic 1 (done) |
| `internal/command/handler.go` | Interface Handler | Epic 1 (done) |
| `internal/command/deprecated.go` | DeprecatedBridge | Epic 1 (done) |
| `internal/command/handlers/servicemodestatushandler/` | handler.go, handler_test.go | Story 2.3/2.4 (done) |

### Файлы на изменение

| Файл | Действие | Описание |
|------|----------|----------|
| `internal/constants/constants.go` | изменить | Добавить `ActNRServiceModeEnable` |
| `cmd/apk-ci/main.go` | изменить | Добавить blank import, удалить legacy case `ActServiceModeEnable` |
| `internal/command/handlers/servicemodeenablehandler/handler.go` | новый | Handler struct, init(), Execute, writeText, writeError, createRACClient |
| `internal/command/handlers/servicemodeenablehandler/handler_test.go` | новый | Unit-тесты |

### Файлы НЕ ТРОГАТЬ

- `internal/adapter/onec/rac/interfaces.go` — Story 2.1 (done)
- `internal/adapter/onec/rac/client.go` — Story 2.2 (done)
- `internal/adapter/onec/rac/client_test.go` — Story 2.2 (done)
- `internal/adapter/onec/rac/ractest/mock.go` — Story 2.1 (done)
- `internal/command/registry.go`, `handler.go`, `deprecated.go` — не менять
- `internal/command/handlers/servicemodestatushandler/` — не менять
- `internal/pkg/output/` — не менять
- `internal/pkg/tracing/` — не менять
- Legacy код: `internal/rac/`, `internal/servicemode/`, `internal/app/app.go`

### Что НЕ делать

- НЕ менять `interfaces.go`, `client.go`, `client_test.go`, `ractest/mock.go` (Story 2.1, 2.2)
- НЕ менять legacy код в `internal/rac/`, `internal/servicemode/`
- НЕ менять `registry.go`, `handler.go`, `deprecated.go`
- НЕ добавлять Wire-провайдеры (будет позже)
- НЕ реализовывать disable (Story 2.7)
- НЕ реализовывать force disconnect как отдельную функциональность (Story 2.6) — `terminateSessions` уже встроен в `EnableServiceMode`
- НЕ реализовывать state-aware execution (Story 2.8) — но AC-5 (идемпотентность) реализовать в рамках этой story
- НЕ менять signature `EnableServiceMode` для передачи message/permission_code
- НЕ добавлять retry logic в handler (RAC клиент не имеет retry by design)

### Project Structure Notes

- **Новый пакет**: `internal/command/handlers/servicemodeenablehandler/`
- **Новый файл**: `handler.go` — handler struct + init() + Execute + writeText + writeError + createRACClient
- **Новый файл**: `handler_test.go` — unit-тесты
- **Изменение**: `internal/constants/constants.go` — добавить `ActNRServiceModeEnable`
- **Изменение**: `cmd/apk-ci/main.go` — добавить blank import, удалить legacy case `ActServiceModeEnable`
- Следует паттерну: один handler = один пакет в `handlers/`
- Именование пакета: `servicemodeenablehandler` (аналогично `servicemodestatushandler`)

### References

- [Source: _bmad-output/project-planning-artifacts/epics/epic-2-service-mode.md#Story 2.5]
- [Source: _bmad-output/project-planning-artifacts/architecture.md#Command Registry, Output Format, Error Handling]
- [Source: internal/command/handlers/servicemodestatushandler/handler.go — каноничный пример NR-handler для service mode]
- [Source: internal/command/handler.go — interface Handler]
- [Source: internal/command/registry.go — Register, RegisterWithAlias]
- [Source: internal/command/deprecated.go — DeprecatedBridge]
- [Source: internal/adapter/onec/rac/interfaces.go — Client, ServiceModeManager (EnableServiceMode, VerifyServiceMode)]
- [Source: internal/adapter/onec/rac/client.go:404-449 — EnableServiceMode implementation (message, permission code, terminateSessions)]
- [Source: internal/adapter/onec/rac/client.go:539-558 — VerifyServiceMode implementation]
- [Source: internal/adapter/onec/rac/ractest/mock.go — MockRACClient, EnableServiceModeFunc, VerifyServiceModeFunc]
- [Source: internal/pkg/output/result.go — Result, Metadata, ErrorInfo structs]
- [Source: internal/pkg/tracing/ — TraceIDFromContext, GenerateTraceID]
- [Source: internal/constants/constants.go — ActServiceModeEnable (legacy), DefaultServiceModeMessage]
- [Source: internal/config/config.go:186 — TerminateSessions bool, BR_TERMINATE_SESSIONS]
- [Source: cmd/apk-ci/main.go:88-108 — legacy case ActServiceModeEnable (УДАЛИТЬ)]
- [Source: _bmad-output/implementation-artifacts/stories/2-3-nr-service-mode-status.md — предыдущая story с learnings]
- [Source: _bmad-output/implementation-artifacts/stories/2-4-session-info-service-mode-status.md — предыдущая story с learnings]

### Git Intelligence

Последние коммиты релевантны текущей story:
- `add95cd feat(command): add nr-service-mode-status command with session info` — NR-команда status (паттерн handler)
- `d0720ab feat(rac): add infobase authentication and improve service mode handling` — EnableServiceMode/DisableServiceMode в RAC client
- `a2b24e2 feat(rac): implement RAC client for 1C cluster management` — RAC client с GetClusterInfo, GetInfobaseInfo
- `e5589c3 feat(rac): define RAC client interfaces and mock for testing` — интерфейсы и mock

**Паттерны из git:**
- Commit convention: `feat(scope): description` на английском
- Тесты добавляются вместе с кодом в одном коммите
- Mock обновляется в том же коммите что и использующий его код (в данной story mock НЕ меняется)
- Удаление legacy case из main.go — в том же коммите что и blank import

### Previous Story Intelligence (Story 2.3 + 2.4)

**Learnings из Story 2.3:**
1. **Code Review #1** нашёл: отсутствие slog-логирования, мёртвый legacy case в main.go, недостаточное покрытие тестами text output, отсутствие табличных тестов ошибок
2. **Code Review #2** нашёл: writeError использовал TextWriter вместо текстового формата, отсутствие теста text error output, отсутствие тестов error paths для createRACClient
3. **Рекомендуемый подход для тестируемости**: `racClient rac.Client` как поле struct (nil в production, mock в тестах) — **уже реализован**
4. **Паттерн ошибок**: использовать `output.Result` со `Status: "error"` и `ErrorInfo`, затем return error

**Learnings из Story 2.4:**
1. Использовать `make([]XxxData, 0)` для гарантии `[]` в JSON (не `null`)
2. Graceful degradation при ошибках получения дополнительных данных
3. Все 21 тест прошли без регрессий — паттерн надёжный

**Применимость к Story 2.5:**
- Сразу добавить slog-логирование с trace_id для всех этапов (check, enable, verify)
- Сразу удалить legacy case из main.go при добавлении blank import
- Добавить табличные тесты для error paths (ClusterError, InfobaseError, EnableError, VerifyError)
- Реализовать writeError идентично Story 2.3 (text формат: "Ошибка: msg\nКод: code", JSON: output.Result)
- Использовать `racClient rac.Client` как поле struct для testability

## Dev Agent Record

### Agent Model Used

Claude Opus 4.5 (claude-opus-4-5-20251101)

### Debug Log References

### Completion Notes List

- Реализован NR-handler `ServiceModeEnableHandler` по паттерну `servicemodestatushandler` (Story 2.3/2.4)
- Добавлена константа `ActNRServiceModeEnable = "nr-service-mode-enable"` в `constants.go`
- Handler зарегистрирован через `init()` с deprecated alias `service-mode-enable` (через `RegisterWithAlias`)
- Legacy case `ActServiceModeEnable` удалён из switch в `main.go`, blank import добавлен
- Реализована идемпотентность (AC-5): Check → Act → Verify паттерн. При уже включённом режиме возвращается `already_enabled: true`
- Поддержка `BR_TERMINATE_SESSIONS=true` с подсчётом завершённых сессий из `status.ActiveSessions`
- Чтение env-переменных `BR_SERVICE_MODE_MESSAGE` и `BR_SERVICE_MODE_PERMISSION_CODE` для отчёта в output (RAC client использует захардкоженные значения)
- slog-логирование с trace_id на всех этапах (check, enable, verify)
- 15 unit-тестов, включая: регистрацию, deprecated alias, text/JSON output, идемпотентность, terminate sessions, ошибки (табличные тесты для Cluster/Infobase/Enable), верификацию, text error output, nil config, кастомные env params, createRACClient errors
- 0 lint issues в изменённых файлах
- Все существующие тесты проходят без регрессий (кроме предсуществующих сетевых тестов ExtensionPublish)

### Change Log

- 2026-01-27: Реализована Story 2.5 — NR-команда nr-service-mode-enable с идемпотентностью, terminate sessions, deprecated alias, 15 unit-тестов
- 2026-01-27: Code Review — найдено 1 HIGH, 3 MEDIUM, 3 LOW. Исправлены: дублированный slog (H-1), комментарий про env-переменные (M-1), warn при невозможности подсчёта сессий (M-2), TODO для ScheduledJobsBlocked (M-3). Все 15 тестов прошли. Статус → done
- 2026-02-04: Code Review (AI) Batch Epic 2 — H-2: добавлен post-check для ScheduledJobsBlocked (симметрично с disable), M-1: добавлено предупреждение при пустых паролях RAC, обновлён mock newEnableMock для post-check
- 2026-02-04: Code Review (AI) Epic 2 Final — M-2: добавлено предупреждение при timeout > 5 минут в createRACClient

### File List

| Файл | Действие |
|------|----------|
| `internal/constants/constants.go` | изменён — добавлена константа `ActNRServiceModeEnable` |
| `cmd/apk-ci/main.go` | изменён — добавлен blank import `servicemodeenablehandler`, удалён legacy case `ActServiceModeEnable` |
| `internal/command/handlers/servicemodeenablehandler/handler.go` | создан — handler struct, init(), Execute, writeText, writeError, outputResult, createRACClient |
| `internal/command/handlers/servicemodeenablehandler/handler_test.go` | создан — 15 unit-тестов |
| `_bmad-output/implementation-artifacts/sprint-artifacts/sprint-status.yaml` | изменён — статус 2-5 → review |
