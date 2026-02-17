# Story 2.6: Force Disconnect Sessions

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a DevOps-инженер,
I want принудительно завершить сессии пользователей информационной базы через отдельную NR-команду,
So that я могу освободить базу от активных подключений перед обслуживанием, независимо от включения сервисного режима.

## Acceptance Criteria

1. **AC-1**: Команда `BR_COMMAND=nr-force-disconnect-sessions BR_INFOBASE_NAME=MyBase` завершает все активные сессии информационной базы
2. **AC-2**: По умолчанию завершаются ВСЕ сессии (`BR_FORCE_DISCONNECT=true` не требуется — сама команда уже является force disconnect)
3. **AC-3**: `BR_DISCONNECT_DELAY_SEC` — grace period в секундах перед завершением (по умолчанию: 0 — немедленное завершение)
4. **AC-4**: JSON output содержит: `terminated_sessions_count`, `sessions` (список завершённых сессий с user_name, app_id, host), `delay_sec`, `infobase_name` + metadata (duration_ms, trace_id, api_version)
5. **AC-5**: Text output содержит: количество завершённых сессий, список пользователей, задержку
6. **AC-6**: Идемпотентность: если нет активных сессий → success + `terminated_sessions_count: 0` + `"no_active_sessions": true`
7. **AC-7**: Команда зарегистрирована в Command Registry через `init()` и доступна через `command.Get("nr-force-disconnect-sessions")`
8. **AC-8**: Legacy команды нет — это новая функциональность, deprecated alias не нужен
9. **AC-9**: trace_id присутствует в логах и JSON-ответе
10. **AC-10**: При отсутствии `BR_INFOBASE_NAME` — возврат структурированной ошибки (code + message)
11. **AC-11**: Вывод списка сессий ПЕРЕД завершением (для аудита: кого завершили)
12. **AC-12**: При частичном завершении (некоторые сессии не удалось завершить) — success с `partial_failure: true` и списком ошибок в `errors[]`
13. **AC-13**: Unit-тесты покрывают: регистрацию, text output, JSON output, пустые сессии (идемпотентность), успешное завершение, частичные ошибки, delay, ошибки RAC

## Tasks / Subtasks

- [x] Task 1: Добавить константу `ActNRForceDisconnectSessions` (AC: 7)
  - [x] 1.1 В `internal/constants/constants.go` добавить `ActNRForceDisconnectSessions = "nr-force-disconnect-sessions"`
- [x] Task 2: Создать пакет handler `internal/command/handlers/forcedisconnecthandler/` (AC: 7, 8)
  - [x] 2.1 Создать `handler.go` со struct `ForceDisconnectHandler`
  - [x] 2.2 Реализовать `Name() string` — возвращает `constants.ActNRForceDisconnectSessions`
  - [x] 2.3 Реализовать `Description() string` — "Принудительное завершение сессий информационной базы"
  - [x] 2.4 В `init()` зарегистрировать через `command.Register(&ForceDisconnectHandler{})` — без legacy alias (AC-8)
- [x] Task 3: Реализовать `Execute` метод (AC: 1-6, 9-12)
  - [x] 3.1 Получить `start := time.Now()` для duration
  - [x] 3.2 Получить traceID из context или сгенерировать новый
  - [x] 3.3 Валидировать `cfg.InfobaseName` — при пустом значении вернуть структурированную ошибку (AC-10)
  - [x] 3.4 Прочитать параметр `BR_DISCONNECT_DELAY_SEC` из env (по умолчанию 0) (AC-3)
  - [x] 3.5 Создать или использовать injected RAC клиент (паттерн из Story 2.5)
  - [x] 3.6 Получить cluster info: `racClient.GetClusterInfo(ctx)`
  - [x] 3.7 Получить infobase info: `racClient.GetInfobaseInfo(ctx, clusterUUID, infobaseName)`
  - [x] 3.8 Получить список сессий: `racClient.GetSessions(ctx, clusterUUID, infobaseUUID)` (AC-11)
  - [x] 3.9 **Идемпотентность**: если `len(sessions) == 0` → вернуть success с `no_active_sessions: true` (AC-6)
  - [x] 3.10 Логировать список сессий перед завершением (аудит) (AC-11)
  - [x] 3.11 Если `delaySec > 0` — выполнить `time.Sleep(delay)` с логированием (AC-3)
  - [x] 3.12 Завершить каждую сессию через `racClient.TerminateSession(ctx, clusterUUID, sessionID)` — собрать результаты (AC-1, AC-12)
  - [x] 3.13 Сформировать `ForceDisconnectData` с результатами (AC-4, AC-5, AC-12)
  - [x] 3.14 Для текстового формата — вывести человекочитаемый текст через `writeText`
  - [x] 3.15 Для JSON формата — вывести `output.Result` с Data, Metadata
- [x] Task 4: Определить data struct для ответа (AC: 4, 5)
  - [x] 4.1 Создать `ForceDisconnectData` с полями: `TerminatedSessionsCount int`, `NoActiveSessions bool`, `DelaySec int`, `InfobaseName string`, `PartialFailure bool`, `Sessions []DisconnectedSessionInfo`, `Errors []string`
  - [x] 4.2 Создать `DisconnectedSessionInfo` с полями: `UserName string`, `AppID string`, `Host string`, `SessionID string`
  - [x] 4.3 Реализовать `writeText(w io.Writer) error` для человекочитаемого формата
- [x] Task 5: Зарегистрировать blank import в main.go (AC: 7)
  - [x] 5.1 Добавить `_ "github.com/Kargones/apk-ci/internal/command/handlers/forcedisconnecthandler"` в `cmd/apk-ci/main.go`
- [x] Task 6: Написать unit-тесты (AC: 13)
  - [x] 6.1 `TestForceDisconnectHandler_Name` — проверка имени
  - [x] 6.2 `TestForceDisconnectHandler_Description` — не пустое
  - [x] 6.3 `TestForceDisconnectHandler_Registration` — `command.Get("nr-force-disconnect-sessions")` успешен
  - [x] 6.4 `TestForceDisconnectHandler_Execute_TextOutput` — завершение 2 сессий, проверка текстового вывода
  - [x] 6.5 `TestForceDisconnectHandler_Execute_JSONOutput` — проверка JSON с парсингом `output.Result`
  - [x] 6.6 `TestForceDisconnectHandler_Execute_NoSessions` — идемпотентность: нет сессий → success + no_active_sessions=true
  - [x] 6.7 `TestForceDisconnectHandler_Execute_WithDelay` — `BR_DISCONNECT_DELAY_SEC=1`, проверка что delay применяется (можно проверить через data.DelaySec)
  - [x] 6.8 `TestForceDisconnectHandler_Execute_PartialFailure` — одна сессия из двух не завершается → partial_failure=true + errors
  - [x] 6.9 `TestForceDisconnectHandler_Execute_NoInfobase` — ошибка при пустом infobase name
  - [x] 6.10 `TestForceDisconnectHandler_Execute_RACErrors` — табличные тесты для ClusterError, InfobaseError, SessionsError
  - [x] 6.11 `TestForceDisconnectHandler_Execute_NilConfig` — nil config → ошибка
  - [x] 6.12 `TestForceDisconnectHandler_Execute_TextError` — ошибка в текстовом формате

### Review Follow-ups (AI)

- [ ] [AI-Review][HIGH] Handler использует command.Register вместо RegisterWithAlias — нет deprecated alias, нет теста подтверждающего intent [handler.go:24]
- [ ] [AI-Review][MEDIUM] callCount в тестах использует обычный int без atomic [handler_test.go:236,467]
- [ ] [AI-Review][MEDIUM] PartialFailure: handler возвращает nil error (success status) при partial_failure=true — двойная семантика [handler.go:275-286]
- [ ] [AI-Review][MEDIUM] Grace period блокирует goroutine на time.After — sequential execution при нескольких базах [handler.go:236-244]
- [ ] [AI-Review][LOW] ForceDisconnectData.Errors — []string без структурированности, нет error code [handler.go:59]
- [ ] [AI-Review][LOW] Нет теста для createRACClient ошибок в forcedisconnecthandler [handler_test.go]

### Review Follow-ups (AI Code Review #34)

- [ ] [AI-Review][MEDIUM] StateChanged семантика неоднозначна для partial failure [handler.go:278]
- [ ] [AI-Review][MEDIUM] Grace period — context-aware delay заменил time.Sleep (уже исправлено) [handler.go:236-244]

## Dev Notes

### Архитектурные ограничения

- **ADR-002: Command Registry** — handler регистрируется в `init()` через `command.Register`. Это НОВАЯ команда — deprecated alias НЕ нужен (в отличие от Story 2.5).
- **ADR-003: ISP** — использовать `rac.Client` (композитный интерфейс). Методы: `GetClusterInfo`, `GetInfobaseInfo`, `GetSessions`, `TerminateSession`.
- **ADR-001: Wire DI** — на текущем этапе handler создаёт RAC клиент напрямую в Execute. Wire-провайдер будет добавлен позже.
- **Все комментарии на русском языке** (CLAUDE.md).
- **Logger только в stderr**, stdout зарезервирован для output (JSON/text).
- **НЕ менять интерфейсы RAC** — Story 2.1 и 2.2 закончены, их код не трогать.
- **НЕ использовать `TerminateAllSessions`** — вместо этого завершать поштучно через `TerminateSession`, чтобы собирать информацию о каждой завершённой сессии и обрабатывать частичные ошибки (AC-12).

### Паттерн реализации — следовать `servicemodeenablehandler/handler.go`

Каноничный пример NR-handler в этом epic: `internal/command/handlers/servicemodeenablehandler/handler.go` (Story 2.5). Структура handler:

```go
package forcedisconnecthandler

func init() {
    command.Register(&ForceDisconnectHandler{})
    // БЕЗ alias — это новая команда
}

type ForceDisconnectHandler struct {
    racClient rac.Client // nil в production, mock в тестах
}

func (h *ForceDisconnectHandler) Name() string {
    return constants.ActNRForceDisconnectSessions
}

func (h *ForceDisconnectHandler) Description() string {
    return "Принудительное завершение сессий информационной базы"
}

func (h *ForceDisconnectHandler) Execute(ctx context.Context, cfg *config.Config) error {
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
func (h *ForceDisconnectHandler) createRACClient(cfg *config.Config) (rac.Client, error) {
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
    // ... port, timeout, opts как в servicemodeenablehandler ...
}
```

### Ключевая логика — поштучное завершение с аудитом

**Паттерн: List → Log → Delay → Terminate Each → Report**

```go
// 1. Получить список сессий
sessions, err := racClient.GetSessions(ctx, clusterInfo.UUID, infobaseInfo.UUID)
if err != nil {
    return h.writeError(format, traceID, start, "RAC.SESSIONS_FAILED", ...)
}

// 2. Идемпотентность: нет сессий
if len(sessions) == 0 {
    data := &ForceDisconnectData{
        NoActiveSessions: true,
        InfobaseName:     cfg.InfobaseName,
    }
    return h.outputResult(format, data, traceID, start)
}

// 3. Логировать сессии (аудит)
log.Info("Найдены активные сессии для завершения",
    slog.Int("count", len(sessions)))
for _, s := range sessions {
    log.Info("Сессия для завершения",
        slog.String("user", s.UserName),
        slog.String("app", s.AppID),
        slog.String("host", s.Host),
        slog.String("session_id", s.SessionID))
}

// 4. Delay (если указан)
if delaySec > 0 {
    log.Info("Ожидание grace period", slog.Int("delay_sec", delaySec))
    time.Sleep(time.Duration(delaySec) * time.Second)
}

// 5. Завершить поштучно
var terminated []DisconnectedSessionInfo
var errors []string
for _, s := range sessions {
    if err := racClient.TerminateSession(ctx, clusterInfo.UUID, s.SessionID); err != nil {
        errors = append(errors, fmt.Sprintf("сессия %s (%s): %v", s.SessionID, s.UserName, err))
        log.Warn("Не удалось завершить сессию", slog.String("session_id", s.SessionID), slog.String("error", err.Error()))
    } else {
        terminated = append(terminated, DisconnectedSessionInfo{
            UserName:  s.UserName,
            AppID:     s.AppID,
            Host:      s.Host,
            SessionID: s.SessionID,
        })
    }
}

// 6. Результат
data := &ForceDisconnectData{
    TerminatedSessionsCount: len(terminated),
    Sessions:                terminated,
    DelaySec:                delaySec,
    InfobaseName:            cfg.InfobaseName,
    PartialFailure:          len(errors) > 0,
    Errors:                  errors,
}
```

### Чтение env-переменных

```go
// BR_DISCONNECT_DELAY_SEC — grace period (по умолчанию 0)
delaySec := 0
if delayStr := os.Getenv("BR_DISCONNECT_DELAY_SEC"); delayStr != "" {
    if d, err := strconv.Atoi(delayStr); err == nil && d >= 0 {
        delaySec = d
    } else {
        log.Warn("Некорректное значение BR_DISCONNECT_DELAY_SEC, используется 0", slog.String("value", delayStr))
    }
}
```

### Data struct для ответа

```go
// DisconnectedSessionInfo содержит информацию о завершённой сессии.
type DisconnectedSessionInfo struct {
    UserName  string `json:"user_name"`
    AppID     string `json:"app_id"`
    Host      string `json:"host"`
    SessionID string `json:"session_id"`
}

// ForceDisconnectData содержит данные ответа о принудительном завершении сессий.
type ForceDisconnectData struct {
    TerminatedSessionsCount int                      `json:"terminated_sessions_count"`
    NoActiveSessions        bool                     `json:"no_active_sessions"`
    DelaySec                int                      `json:"delay_sec"`
    InfobaseName            string                   `json:"infobase_name"`
    PartialFailure          bool                     `json:"partial_failure"`
    Sessions                []DisconnectedSessionInfo `json:"sessions"`
    Errors                  []string                 `json:"errors,omitempty"`
}
```

### Формат текстового вывода

**Успешное завершение:**
```
Принудительное завершение сессий: MyBase
Завершено сессий: 3
  1. Иванов (1CV8C) — workstation-01
  2. Петров (1CV8) — workstation-02
  3. Сидоров (WebClient) — 10.0.0.15
```

**Нет активных сессий (идемпотентность):**
```
Принудительное завершение сессий: MyBase
Активных сессий нет
```

**Частичная ошибка:**
```
Принудительное завершение сессий: MyBase
Завершено сессий: 2 из 3
  1. Иванов (1CV8C) — workstation-01
  2. Петров (1CV8) — workstation-02
Ошибки:
  - сессия session-uuid-3 (Сидоров): connection timeout
```

### Формат JSON вывода

**Успешное завершение:**
```json
{
  "status": "success",
  "command": "nr-force-disconnect-sessions",
  "data": {
    "terminated_sessions_count": 2,
    "no_active_sessions": false,
    "delay_sec": 0,
    "infobase_name": "MyBase",
    "partial_failure": false,
    "sessions": [
      {"user_name": "Иванов", "app_id": "1CV8C", "host": "workstation-01", "session_id": "uuid-1"},
      {"user_name": "Петров", "app_id": "1CV8", "host": "workstation-02", "session_id": "uuid-2"}
    ]
  },
  "metadata": {
    "duration_ms": 350,
    "trace_id": "abc123def456...",
    "api_version": "v1"
  }
}
```

**Нет сессий (идемпотентность):**
```json
{
  "status": "success",
  "command": "nr-force-disconnect-sessions",
  "data": {
    "terminated_sessions_count": 0,
    "no_active_sessions": true,
    "delay_sec": 0,
    "infobase_name": "MyBase",
    "partial_failure": false,
    "sessions": []
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
  "command": "nr-force-disconnect-sessions",
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

### Тестирование — паттерн из servicemodeenablehandler

```go
func TestForceDisconnectHandler_Execute_JSONOutput(t *testing.T) {
    t.Setenv("BR_OUTPUT_FORMAT", "json")

    sessions := ractest.SessionData() // 2 тестовые сессии
    mock := ractest.NewMockRACClientWithSessions(sessions)

    h := &ForceDisconnectHandler{racClient: mock}
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
    assert.Equal(t, float64(2), data["terminated_sessions_count"])
    assert.False(t, data["no_active_sessions"].(bool))
}

func TestForceDisconnectHandler_Execute_NoSessions(t *testing.T) {
    t.Setenv("BR_OUTPUT_FORMAT", "json")

    mock := &MockRACClient{
        GetSessionsFunc: func(_ context.Context, _, _ string) ([]rac.SessionInfo, error) {
            return []rac.SessionInfo{}, nil
        },
    }

    h := &ForceDisconnectHandler{racClient: mock}
    cfg := &config.Config{InfobaseName: "TestBase"}

    out := testutil.CaptureStdout(t, func() {
        err := h.Execute(context.Background(), cfg)
        require.NoError(t, err)
    })

    var result output.Result
    err := json.Unmarshal([]byte(out), &result)
    require.NoError(t, err)

    data := result.Data.(map[string]interface{})
    assert.True(t, data["no_active_sessions"].(bool))
    assert.Equal(t, float64(0), data["terminated_sessions_count"])
}

func TestForceDisconnectHandler_Execute_PartialFailure(t *testing.T) {
    t.Setenv("BR_OUTPUT_FORMAT", "json")

    sessions := ractest.SessionData() // 2 тестовые сессии
    mock := ractest.NewMockRACClientWithSessions(sessions)
    // Первая сессия завершается, вторая — ошибка
    callCount := 0
    mock.TerminateSessionFunc = func(_ context.Context, _, _ string) error {
        callCount++
        if callCount == 2 {
            return fmt.Errorf("connection timeout")
        }
        return nil
    }

    h := &ForceDisconnectHandler{racClient: mock}
    cfg := &config.Config{InfobaseName: "TestBase"}

    out := testutil.CaptureStdout(t, func() {
        err := h.Execute(context.Background(), cfg)
        require.NoError(t, err) // Команда не фейлится при частичных ошибках
    })

    var result output.Result
    err := json.Unmarshal([]byte(out), &result)
    require.NoError(t, err)
    assert.Equal(t, "success", result.Status)

    data := result.Data.(map[string]interface{})
    assert.True(t, data["partial_failure"].(bool))
    assert.Equal(t, float64(1), data["terminated_sessions_count"])
}
```

### Существующий код (НЕ МЕНЯТЬ)

| Файл | Что содержит | Статус |
|------|-------------|--------|
| `internal/adapter/onec/rac/interfaces.go` | Client, SessionProvider, TerminateSession, GetSessions | Story 2.1 (done) |
| `internal/adapter/onec/rac/client.go` | TerminateSession, TerminateAllSessions, GetSessions | Story 2.2 (done) |
| `internal/adapter/onec/rac/client_test.go` | 34 unit-теста | Story 2.2 (done) |
| `internal/adapter/onec/rac/ractest/mock.go` | MockRACClient с TerminateSessionFunc, GetSessionsFunc | Story 2.1 (done) |
| `internal/command/registry.go` | Register, Get, All | Epic 1 (done) |
| `internal/command/handler.go` | Interface Handler | Epic 1 (done) |
| `internal/command/handlers/servicemodeenablehandler/` | handler.go, handler_test.go | Story 2.5 (done) |
| `internal/command/handlers/servicemodestatushandler/` | handler.go, handler_test.go | Story 2.3/2.4 (done) |

### Файлы на изменение

| Файл | Действие | Описание |
|------|----------|----------|
| `internal/constants/constants.go` | изменить | Добавить `ActNRForceDisconnectSessions` |
| `cmd/apk-ci/main.go` | изменить | Добавить blank import `forcedisconnecthandler` |
| `internal/command/handlers/forcedisconnecthandler/handler.go` | новый | Handler struct, init(), Execute, writeText, writeError, createRACClient |
| `internal/command/handlers/forcedisconnecthandler/handler_test.go` | новый | Unit-тесты |

### Файлы НЕ ТРОГАТЬ

- `internal/adapter/onec/rac/interfaces.go` — Story 2.1 (done)
- `internal/adapter/onec/rac/client.go` — Story 2.2 (done)
- `internal/adapter/onec/rac/client_test.go` — Story 2.2 (done)
- `internal/adapter/onec/rac/ractest/mock.go` — Story 2.1 (done)
- `internal/command/registry.go`, `handler.go`, `deprecated.go` — не менять
- `internal/command/handlers/servicemodestatushandler/` — не менять
- `internal/command/handlers/servicemodeenablehandler/` — не менять
- `internal/pkg/output/` — не менять
- `internal/pkg/tracing/` — не менять
- Legacy код: `internal/rac/`, `internal/servicemode/`, `internal/app/app.go`

### Что НЕ делать

- НЕ менять `interfaces.go`, `client.go`, `client_test.go`, `ractest/mock.go` (Story 2.1, 2.2)
- НЕ менять legacy код в `internal/rac/`, `internal/servicemode/`
- НЕ менять `registry.go`, `handler.go`, `deprecated.go`
- НЕ добавлять Wire-провайдеры (будет позже)
- НЕ реализовывать disable (Story 2.7)
- НЕ реализовывать state-aware execution (Story 2.8)
- НЕ использовать `TerminateAllSessions` — завершать поштучно для детального отчёта и partial failure handling
- НЕ добавлять retry logic (RAC клиент не имеет retry by design)
- НЕ менять legacy case `ActServiceModeDisable` в main.go — он ещё используется (Story 2.7 не реализована)

### Project Structure Notes

- **Новый пакет**: `internal/command/handlers/forcedisconnecthandler/`
- **Новый файл**: `handler.go` — handler struct + init() + Execute + writeText + writeError + createRACClient
- **Новый файл**: `handler_test.go` — unit-тесты
- **Изменение**: `internal/constants/constants.go` — добавить `ActNRForceDisconnectSessions`
- **Изменение**: `cmd/apk-ci/main.go` — добавить blank import
- Следует паттерну: один handler = один пакет в `handlers/`
- Именование пакета: `forcedisconnecthandler` (аналогично `servicemodeenablehandler`)

### References

- [Source: _bmad-output/project-planning-artifacts/epics/epic-2-service-mode.md#Story 2.6]
- [Source: _bmad-output/project-planning-artifacts/architecture.md#Command Registry, Output Format, Error Handling]
- [Source: internal/command/handlers/servicemodeenablehandler/handler.go — каноничный пример NR-handler для service mode]
- [Source: internal/command/handler.go — interface Handler]
- [Source: internal/command/registry.go — Register, Get]
- [Source: internal/adapter/onec/rac/interfaces.go — Client, SessionProvider (GetSessions, TerminateSession)]
- [Source: internal/adapter/onec/rac/client.go:358-401 — TerminateSession, TerminateAllSessions implementation]
- [Source: internal/adapter/onec/rac/client.go:306-355 — GetSessions implementation]
- [Source: internal/adapter/onec/rac/ractest/mock.go — MockRACClient, GetSessionsFunc, TerminateSessionFunc]
- [Source: internal/pkg/output/result.go — Result, Metadata, ErrorInfo structs]
- [Source: internal/pkg/tracing/ — TraceIDFromContext, GenerateTraceID]
- [Source: internal/constants/constants.go — текущие константы]
- [Source: cmd/apk-ci/main.go — blank imports, registry flow]
- [Source: _bmad-output/implementation-artifacts/stories/2-5-nr-service-mode-enable.md — предыдущая story с learnings]

### Git Intelligence

Последние коммиты релевантны текущей story:
- `da1a64e feat(command): implement nr-service-mode-enable handler` — NR-handler enable (паттерн handler)
- `add95cd feat(command): add nr-service-mode-status command with session info` — NR-handler status (паттерн handler, session info)
- `d0720ab feat(rac): add infobase authentication and improve service mode handling` — RAC client с GetSessions, TerminateSession, TerminateAllSessions
- `a2b24e2 feat(rac): implement RAC client for 1C cluster management` — RAC client с GetClusterInfo, GetInfobaseInfo
- `e5589c3 feat(rac): define RAC client interfaces and mock for testing` — интерфейсы и mock

**Паттерны из git:**
- Commit convention: `feat(scope): description` на английском
- Тесты добавляются вместе с кодом в одном коммите
- Mock обновляется в том же коммите что и использующий его код (в данной story mock НЕ меняется)
- Blank import в main.go — в том же коммите что и handler

### Previous Story Intelligence (Story 2.5)

**Learnings из Story 2.5:**
1. **Code Review** нашёл: дублированный slog, комментарии про env-переменные, warn при невозможности подсчёта сессий, TODO для ScheduledJobsBlocked
2. **Рекомендуемый подход для тестируемости**: `racClient rac.Client` как поле struct (nil в production, mock в тестах)
3. **Паттерн ошибок**: использовать `output.Result` со `Status: "error"` и `ErrorInfo`, затем return error
4. **writeError**: для text формата — "Ошибка: msg\nКод: code", для JSON — output.Result
5. **createRACClient**: копировать паттерн (DI будет позже)
6. Все 15 тестов прошли без регрессий

**Learnings из Story 2.3/2.4:**
1. Использовать `make([]XxxData, 0)` для гарантии `[]` в JSON (не `null`)
2. Graceful degradation при ошибках получения дополнительных данных
3. Табличные тесты для error paths — обязательно

**Применимость к Story 2.6:**
- Сразу добавить slog-логирование с trace_id для всех этапов
- Использовать `make([]DisconnectedSessionInfo, 0)` для гарантии `[]` в JSON
- Использовать `make([]string, 0)` для Errors в JSON (если нет ошибок — `[]`, не `null`)
- Добавить табличные тесты для error paths (ClusterError, InfobaseError, SessionsError)
- Реализовать writeError идентично Story 2.5
- Использовать `racClient rac.Client` как поле struct для testability
- Не дублировать slog — использовать один log.Info для аудита

## Dev Agent Record

### Agent Model Used

Claude Opus 4.5 (claude-opus-4-5-20251101)

### Debug Log References

### Completion Notes List

- Реализован NR-handler `nr-force-disconnect-sessions` для принудительного завершения сессий информационной базы 1C
- Следует каноничному паттерну из `servicemodeenablehandler` (Story 2.5): struct с `racClient rac.Client`, `init()` с `command.Register`, `Execute`, `outputResult`, `writeError`, `writeText`, `createRACClient`
- Без deprecated alias (AC-8) — это новая команда
- Поштучное завершение сессий через `TerminateSession` (не `TerminateAllSessions`) для детального аудита и partial failure handling (AC-12)
- Идемпотентность: нет сессий → success с `no_active_sessions: true` (AC-6)
- Grace period через `BR_DISCONNECT_DELAY_SEC` env (AC-3)
- `make([]DisconnectedSessionInfo, 0)` и `make([]string, 0)` для гарантии `[]` в JSON (не `null`) — learnings из Story 2.3/2.4
- slog-логирование с trace_id на всех этапах (AC-9, AC-11)
- 12 unit-тестов: все PASS, 0 lint issues в новом коде
- Предсуществующие FAIL в `internal/app` (DNS-зависимые тесты `TestExtensionPublish_MissingEnvVars`) — не регрессия

### Change Log

- 2026-01-27: Реализована Story 2.6 — NR-команда nr-force-disconnect-sessions (handler + 12 unit-тестов)
- 2026-01-27: Code Review #1 — исправлено: omitempty для Errors (nil vs make), добавлены assertions на отсутствие errors в JSON при успехе, дополнен File List
- 2026-01-27: Code Review #2 — исправлено: убран `omitempty` с `Errors` + `make([]string, 0)` для единообразия с `Sessions`; fix отступа закрывающей скобки (no-sessions case); добавлен `TestForceDisconnectHandler_Execute_InvalidDelay` (3 подтеста); обновлены assertions errors в JSON-тестах. Итого: 15 тестов PASS
- 2026-01-28: Code Review #3 — исправлено 6 проблем: (1) [HIGH] time.Sleep заменён на select с ctx.Done() для корректной отмены; (2) [HIGH] тест WithDelay больше не спит 1 секунду; (3) [MEDIUM] добавлен верхний лимит 300 сек для BR_DISCONNECT_DELAY_SEC; (4) [MEDIUM] добавлен тест text partial failure; (5) [MEDIUM] добавлен TODO для createRACClient duplication; (6) [LOW] переименован shadowing `errors` → `terminateErrors`. Добавлены тесты: DelayParsing (3 подтеста), TextPartialFailure, ContextCancellation. Итого: 20 тестов PASS, 0 lint issues, 0.007s
- 2026-02-04: Code Review (AI) Batch Epic 2 — M-1: добавлено предупреждение при пустых паролях RAC, M-6: добавлен комментарий для max delay 300 сек
- 2026-02-04: Code Review (AI) Epic 2 Final — M-2: добавлено предупреждение при timeout > 5 минут в createRACClient

### File List

- `internal/constants/constants.go` — добавлена константа `ActNRForceDisconnectSessions`
- `internal/command/handlers/forcedisconnecthandler/handler.go` — handler struct, init(), Execute, writeText, writeError, createRACClient, data structs
- `internal/command/handlers/forcedisconnecthandler/handler_test.go` — 20 unit-тестов
- `cmd/apk-ci/main.go` — добавлен blank import `forcedisconnecthandler`
- `_bmad-output/implementation-artifacts/sprint-artifacts/sprint-status.yaml` — обновлён статус story 2-6 на review
