# Story 2.3: nr-service-mode-status

Status: done

## Story

As a DevOps-инженер,
I want проверить статус сервисного режима информационной базы через NR-команду,
So that я знаю можно ли работать с базой и вижу текущее состояние блокировки.

## Acceptance Criteria

1. **AC-1**: Команда `BR_COMMAND=nr-service-mode-status BR_INFOBASE_NAME=MyBase` возвращает статус сервисного режима
2. **AC-2**: Текстовый вывод содержит: enabled/disabled, message, scheduled_jobs_blocked
3. **AC-3**: JSON вывод содержит все поля `ServiceModeStatus` + metadata (trace_id, duration_ms, api_version)
4. **AC-4**: Команда зарегистрирована в Command Registry через `init()` и доступна через `command.Get("nr-service-mode-status")`
5. **AC-5**: Legacy команда `service-mode-status` привязана как deprecated alias через `RegisterWithAlias`
6. **AC-6**: trace_id присутствует в логах и JSON-ответе
7. **AC-7**: При отсутствии `BR_INFOBASE_NAME` — возврат структурированной ошибки (code + message)
8. **AC-8**: Unit-тесты покрывают: регистрацию, text output, JSON output, ошибки

## Tasks / Subtasks

- [x] Task 1: Добавить константу `ActNRServiceModeStatus` (AC: 4)
  - [x] 1.1 В `internal/constants/constants.go` добавить `ActNRServiceModeStatus = "nr-service-mode-status"`
- [x] Task 2: Создать пакет handler `internal/command/handlers/servicemodestatushandler/` (AC: 1, 4, 5)
  - [x] 2.1 Создать `handler.go` со struct `ServiceModeStatusHandler`
  - [x] 2.2 Реализовать `Name() string` — возвращает `constants.ActNRServiceModeStatus`
  - [x] 2.3 Реализовать `Description() string` — "Проверка статуса сервисного режима информационной базы"
  - [x] 2.4 В `init()` зарегистрировать через `command.RegisterWithAlias(&ServiceModeStatusHandler{}, constants.ActServiceModeStatus)` — привязать legacy alias
- [x] Task 3: Реализовать `Execute` метод (AC: 1, 2, 3, 6, 7)
  - [x] 3.1 Получить `start := time.Now()` для duration
  - [x] 3.2 Получить traceID из context или сгенерировать новый
  - [x] 3.3 Валидировать `cfg.InfobaseName` — при пустом значении вернуть ошибку через `output.Result` с `ErrorInfo`
  - [x] 3.4 Создать NR RAC клиент `rac.NewClient(opts)` из `cfg` (собрать `ClientOptions` из config)
  - [x] 3.5 Получить cluster info: `racClient.GetClusterInfo(ctx)`
  - [x] 3.6 Получить infobase info: `racClient.GetInfobaseInfo(ctx, clusterUUID, infobaseName)`
  - [x] 3.7 Получить статус: `racClient.GetServiceModeStatus(ctx, clusterUUID, infobaseUUID)`
  - [x] 3.8 Сформировать `ServiceModeStatusData` для ответа
  - [x] 3.9 Для текстового формата — вывести человекочитаемый текст через `writeText`
  - [x] 3.10 Для JSON формата — вывести `output.Result` с Data, Metadata
- [x] Task 4: Определить data struct для ответа (AC: 2, 3)
  - [x] 4.1 Создать `ServiceModeStatusData` с полями: `Enabled bool`, `Message string`, `ScheduledJobsBlocked bool`, `ActiveSessions int`, `InfobaseName string`
  - [x] 4.2 Реализовать `writeText(w io.Writer) error` для человекочитаемого формата
- [x] Task 5: Зарегистрировать blank import в main.go (AC: 4)
  - [x] 5.1 Добавить `_ "github.com/Kargones/apk-ci/internal/command/handlers/servicemodestatushandler"` в `cmd/benadis-runner/main.go`
- [x] Task 6: Написать unit-тесты (AC: 8)
  - [x] 6.1 `TestServiceModeStatusHandler_Name` — проверка имени
  - [x] 6.2 `TestServiceModeStatusHandler_Description` — не пустое
  - [x] 6.3 `TestServiceModeStatusHandler_Registration` — `command.Get("nr-service-mode-status")` успешен
  - [x] 6.4 `TestServiceModeStatusHandler_DeprecatedAlias` — `command.Get("service-mode-status")` тоже работает
  - [x] 6.5 `TestServiceModeStatusHandler_Execute_TextOutput` — проверка текстового вывода с mock RAC
  - [x] 6.6 `TestServiceModeStatusHandler_Execute_JSONOutput` — проверка JSON с парсингом `output.Result`
  - [x] 6.7 `TestServiceModeStatusHandler_Execute_NoInfobase` — ошибка при пустом infobase name
  - [x] 6.8 `TestServiceModeStatusHandler_Execute_RACError` — обработка ошибки RAC клиента

### Review Follow-ups (AI)

- [ ] [AI-Review][HIGH] createRACClient дублируется ~60 строк в 4 handler'ах — maintenance burden при фиксе багов [handler.go:300-357]
- [ ] [AI-Review][MEDIUM] writeError записывает JSON в os.Stdout и возвращает error — двойной error-reporting [handler.go:261-291]
- [ ] [AI-Review][MEDIUM] outputResult — функция на handler'е, но не использует h — нарушение SRP [handler.go:239-258]
- [ ] [AI-Review][MEDIUM] Execute читает BR_OUTPUT_FORMAT из os.Getenv вместо cfg — затрудняет тестирование [handler.go:133]
- [ ] [AI-Review][LOW] sessionsData инициализируется make([]SessionInfoData, 0) вместо var — JSON выведет [] вместо null [handler.go:204]
- [ ] [AI-Review][LOW] Тесты не проверяют production path createRACClient через Execute [handler_test.go:284-315]

### Review Follow-ups (AI Code Review #34)

- [ ] [AI-Review][HIGH] createRACClient дублируется в 4 handler-ах — DRY нарушение (TODO H-2) [servicemodestatushandler/handler.go:~300]
- [ ] [AI-Review][MEDIUM] Execute читает BR_OUTPUT_FORMAT из os.Getenv — затрудняет тестирование [handler.go:133]
- [ ] [AI-Review][MEDIUM] writeError пишет JSON в stdout и возвращает error — двойная обработка ошибок [handler.go:260-291]

## Dev Notes

### Архитектурные ограничения

- **ADR-002: Command Registry** — handler регистрируется в `init()` через `command.Register` или `command.RegisterWithAlias`. Новые команды добавляются без изменения main.go (кроме blank import).
- **ADR-003: ISP** — handler использует NR RAC клиент (`internal/adapter/onec/rac/Client`), НЕ legacy `internal/servicemode/`. Это первая NR-команда, использующая новый RAC adapter.
- **ADR-001: Wire DI** — на текущем этапе handler создаёт RAC клиент напрямую в Execute. Wire-провайдер будет добавлен позже (не в этой story).
- **Все комментарии на русском языке** (CLAUDE.md).
- **Logger только в stderr**, stdout зарезервирован для output (JSON/text).

### Паттерн реализации — следовать `version.go`

Каноничный пример: `internal/command/handlers/version/version.go`. Структура handler:

```go
package servicemodestatushandler

func init() {
    command.RegisterWithAlias(&ServiceModeStatusHandler{}, constants.ActServiceModeStatus)
}

type ServiceModeStatusHandler struct{}

func (h *ServiceModeStatusHandler) Name() string {
    return constants.ActNRServiceModeStatus
}

func (h *ServiceModeStatusHandler) Description() string {
    return "Проверка статуса сервисного режима информационной базы"
}

func (h *ServiceModeStatusHandler) Execute(ctx context.Context, cfg *config.Config) error {
    start := time.Now()
    traceID := tracing.TraceIDFromContext(ctx)
    if traceID == "" {
        traceID = tracing.GenerateTraceID()
    }
    // ... бизнес-логика ...
}
```

### Создание NR RAC клиента из config

Handler должен собрать `rac.ClientOptions` из `*config.Config`. Данные берутся из:

```go
opts := rac.ClientOptions{
    RACPath:      cfg.AppConfig.Paths.Rac,      // путь к исполняемому файлу rac
    Server:       dbInfo.OneServer,               // сервер 1C для infobase
    Port:         strconv.Itoa(cfg.AppConfig.Rac.Port), // порт RAC (по умолчанию 1545)
    Timeout:      time.Duration(cfg.AppConfig.Rac.Timeout) * time.Second,
    ClusterUser:  cfg.AppConfig.Users.Rac,
    ClusterPass:  cfg.SecretConfig.Passwords.Rac,
    InfobaseUser: cfg.AppConfig.Users.Db,
    InfobasePass: cfg.SecretConfig.Passwords.Db,
    Logger:       slog.Default(), // или создать через logging пакет
}
```

**Получение dbInfo**: `cfg.InfobaseName` содержит имя базы. `cfg.DatabaseInfo[cfg.InfobaseName]` — запись `DatabaseInfo` с полем `OneServer`. Проверить: как legacy `LoadServiceModeConfig` (в `internal/servicemode/servicemode.go:259`) получает server. Использовать `cfg.LoadServiceModeConfig(infobaseName)` если доступен, или собрать opts вручную из cfg полей.

### Формат текстового вывода

```
Сервисный режим: ВКЛЮЧЁН / ВЫКЛЮЧЕН
Информационная база: MyBase
Сообщение: "Система находится в режиме обслуживания"
Регламентные задания: заблокированы / разблокированы
Активные сессии: 5
```

### Формат JSON вывода

```json
{
  "status": "success",
  "command": "nr-service-mode-status",
  "data": {
    "enabled": true,
    "message": "Система находится в режиме обслуживания",
    "scheduled_jobs_blocked": true,
    "active_sessions": 5,
    "infobase_name": "MyBase"
  },
  "metadata": {
    "duration_ms": 150,
    "trace_id": "abc123def456...",
    "api_version": "v1"
  }
}
```

### Формат вывода ошибки (JSON)

```json
{
  "status": "error",
  "command": "nr-service-mode-status",
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

`RegisterWithAlias` создаёт `DeprecatedBridge` (см. `internal/command/deprecated.go`). Когда CI/CD вызывает `service-mode-status`, мост:
1. Логирует warning в stderr: "Command deprecated, use nr-service-mode-status"
2. Делегирует выполнение в `ServiceModeStatusHandler.Execute`

После регистрации alias — legacy switch-case в main.go для `ActServiceModeStatus` станет недоступен (registry перехватит). Это ожидаемое поведение — NR-handler полностью заменяет legacy.

### Существующий код Story 2.1 и 2.2 (НЕ МЕНЯТЬ)

| Файл | Что содержит |
|------|-------------|
| `internal/adapter/onec/rac/interfaces.go` | Интерфейсы: `ClusterProvider`, `InfobaseProvider`, `SessionProvider`, `ServiceModeManager`, `Client`. Data types: `ClusterInfo`, `InfobaseInfo`, `ServiceModeStatus`, `SessionInfo` |
| `internal/adapter/onec/rac/client.go` | `racClient` struct, `NewClient(ClientOptions)`, все 9 методов, парсинг RAC output, коды ошибок RAC.* |
| `internal/adapter/onec/rac/client_test.go` | 34 unit-теста парсинга и ошибок |
| `internal/adapter/onec/rac/ractest/mock.go` | `MockRACClient` с функциональными полями |

### Существующий Command Registry (НЕ МЕНЯТЬ)

| Файл | Что содержит |
|------|-------------|
| `internal/command/registry.go` | `Register`, `RegisterWithAlias`, `Get`, `All`, `Names` |
| `internal/command/handler.go` | Interface `Handler`: `Name()`, `Description()`, `Execute()` |
| `internal/command/deprecated.go` | `DeprecatedBridge`, `Deprecatable` interface |
| `internal/command/handlers/version/version.go` | Каноничный пример NR-handler |
| `internal/command/handlers/help/help.go` | Auto-generated help (автоматически подхватит новую команду) |

### Тестирование — паттерн из version_test.go

```go
func TestServiceModeStatusHandler_Execute_JSONOutput(t *testing.T) {
    t.Setenv("BR_OUTPUT_FORMAT", "json")

    h := &ServiceModeStatusHandler{}
    // Нужен способ инжектировать mock RAC клиент в handler.
    // Варианты:
    // 1. Handler принимает rac.Client через поле (устанавливается в тесте)
    // 2. Handler использует factory function (подменяется в тесте)
    // 3. Handler создаёт клиент из config — тогда тестировать через integration test

    out := testutil.CaptureStdout(t, func() {
        err := h.Execute(ctx, mockCfg)
    })

    var result output.Result
    err := json.Unmarshal([]byte(out), &result)
    require.NoError(t, err)
    assert.Equal(t, "success", result.Status)
}
```

**Рекомендуемый подход для тестируемости**: добавить в struct опциональное поле `racClient rac.Client`. Если `nil` — создавать из config в Execute. Если установлено — использовать напрямую. Это позволит тестировать с mock без Wire DI.

```go
type ServiceModeStatusHandler struct {
    racClient rac.Client // nil в production, mock в тестах
}
```

### Ошибки — использовать output.Result, НЕ return error

При ошибках (отсутствие infobase name, RAC failure) handler должен вывести `output.Result` со `Status: "error"` и `ErrorInfo`, а затем вернуть error из Execute. Это гарантирует структурированный вывод даже при ошибках.

### Что НЕ делать

- НЕ менять `interfaces.go`, `client.go`, `client_test.go`, `ractest/mock.go` (Story 2.1, 2.2)
- НЕ менять legacy код в `internal/rac/`, `internal/servicemode/`
- НЕ менять `registry.go`, `handler.go`, `deprecated.go`
- НЕ добавлять Wire-провайдеры (будет позже)
- НЕ реализовывать session info (Story 2.4)
- НЕ реализовывать enable/disable (Story 2.5, 2.7)
- НЕ добавлять retry logic в handler (RAC клиент не имеет retry by design)

### Project Structure Notes

- Новый пакет: `internal/command/handlers/servicemodestatushandler/` (или `servicemodestatus/`)
- Новый файл: `handler.go` — handler struct + init() + Execute
- Новый файл: `handler_test.go` — unit-тесты
- Изменение: `internal/constants/constants.go` — добавить `ActNRServiceModeStatus`
- Изменение: `cmd/benadis-runner/main.go` — добавить blank import
- Следует паттерну: один handler = один пакет в `handlers/`

### References

- [Source: _bmad-output/project-planning-artifacts/epics/epic-2-service-mode.md#Story 2.3]
- [Source: _bmad-output/project-planning-artifacts/architecture.md#Command Registry, Output Format, Error Handling]
- [Source: internal/command/handlers/version/version.go — каноничный пример NR-handler]
- [Source: internal/command/handler.go — interface Handler]
- [Source: internal/command/registry.go — Register, RegisterWithAlias]
- [Source: internal/command/deprecated.go — DeprecatedBridge]
- [Source: internal/adapter/onec/rac/interfaces.go — Client, ServiceModeStatus]
- [Source: internal/adapter/onec/rac/client.go — NewClient, ClientOptions]
- [Source: internal/adapter/onec/rac/ractest/mock.go — MockRACClient]
- [Source: internal/pkg/output/result.go — Result, Metadata, ErrorInfo]
- [Source: internal/pkg/tracing/ — TraceIDFromContext, GenerateTraceID]
- [Source: internal/servicemode/servicemode.go — legacy reference (LoadServiceModeConfigForDb)]
- [Source: internal/config/config.go — AppConfig.Paths.Rac, AppConfig.Rac, Config.LoadServiceModeConfig]
- [Source: internal/constants/constants.go — ActServiceModeStatus (legacy)]
- [Source: _bmad-output/implementation-artifacts/stories/2-1-rac-adapter-interface.md]
- [Source: _bmad-output/implementation-artifacts/stories/2-2-rac-client-implementation.md]

## Change Log

- 2026-01-27: Реализована NR-команда nr-service-mode-status с полным набором функциональности: handler, text/JSON output, deprecated alias, структурированные ошибки, 10 unit-тестов.
- 2026-01-27: Code Review #1 — исправлены 4 проблемы (H-1: добавлено slog-логирование с trace_id в Execute; H-2: удалён мёртвый legacy case service-mode-status из main.go; M-1: текстовый тест расширен для покрытия ScheduledJobsBlocked; M-2: добавлены табличные тесты ошибок GetInfobaseInfo и GetServiceModeStatus). Итого 11 тестов (включая subtests), линтер 0 проблем.
- 2026-01-27: Code Review #2 — исправлены 4 проблемы (H-1: writeError — консистентный текстовый формат вместо TextWriter; M-1: добавлен TestServiceModeStatusHandler_Execute_TextErrorOutput; M-2: добавлены табличные тесты createRACClient error paths (NilAppConfig, EmptyServer); M-3: writeError JSON-path — ошибка записи логируется вместо игнорирования). Итого 15 тестов (включая subtests), линтер 0 проблем.
- 2026-02-04: Code Review (AI) Batch Epic 2 — M-1: добавлено предупреждение при пустых паролях RAC в createRACClient
- 2026-02-04: Code Review (AI) Epic 2 Final — M-2: добавлено предупреждение при timeout > 5 минут в createRACClient

## Dev Agent Record

### Agent Model Used

Claude Opus 4.5 (claude-opus-4-5-20251101)

### Debug Log References

- Все 13 unit-тестов пройдены (PASS), включая 4 subtest → итого 15 test runs
- Линтер: 0 проблем в изменённых файлах
- Регрессии: нет (pre-existing FAIL в `internal/app` — сетевые тесты, не связаны с изменениями)

### Completion Notes List

- Task 1: Добавлена константа `ActNRServiceModeStatus = "nr-service-mode-status"` в `internal/constants/constants.go`
- Task 2: Создан пакет `internal/command/handlers/servicemodestatushandler/` с `ServiceModeStatusHandler` struct, реализующим `command.Handler` interface. Регистрация через `command.RegisterWithAlias` с deprecated alias `service-mode-status`
- Task 3: Реализован метод `Execute` — получает traceID, валидирует config, создаёт RAC клиент, получает cluster/infobase/status, формирует text или JSON ответ. Ошибки выводятся структурированно через `output.Result` с `ErrorInfo`
- Task 4: Создана `ServiceModeStatusData` struct с JSON тегами и `writeText` методом для человекочитаемого вывода
- Task 5: Добавлен blank import в `cmd/benadis-runner/main.go` для self-registration через `init()`
- Task 6: Написано 10 unit-тестов покрывающих: Name, Description, Registration, DeprecatedAlias, TextOutput, JSONOutput, NoInfobase, NilConfig, RACError, DisabledMode. Используется `ractest.MockRACClient` для тестирования без реального RAC. Handler поддерживает DI через опциональное поле `racClient rac.Client`

### File List

- `internal/constants/constants.go` — изменён (добавлена константа `ActNRServiceModeStatus`)
- `internal/command/handlers/servicemodestatushandler/handler.go` — новый (handler, data struct, Execute, writeText, writeError, createRACClient, slog-логирование с trace_id)
- `internal/command/handlers/servicemodestatushandler/handler_test.go` — новый (11 тестов: Name, Description, Registration, DeprecatedAlias, TextOutput, JSONOutput, NoInfobase, NilConfig, RACError, RACStepErrors/InfobaseError, RACStepErrors/StatusError, DisabledMode)
- `cmd/benadis-runner/main.go` — изменён (добавлен blank import servicemodestatushandler, удалён мёртвый legacy case service-mode-status)
