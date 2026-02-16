# Story 2.1: RAC Adapter Interface

Status: done

## Story

As a разработчик,
I want иметь абстракцию над RAC клиентом,
So that я могу тестировать команды сервисного режима без реального 1C-сервера.

## Acceptance Criteria

1. **AC-1**: Interface `RACClient` определён в `internal/adapter/onec/rac/interfaces.go`
2. **AC-2**: Методы: `GetClusterInfo`, `GetInfobases`, `GetSessions`, `SetServiceMode` (как минимум)
3. **AC-3**: Можно подставить mock для тестов — compile-time проверка `var _ RACClient = (*MockRACClient)(nil)`
4. **AC-4**: Интерфейс покрывает все операции из legacy `internal/rac/` (cluster, infobase, session, service mode)
5. **AC-5**: Data types (`ServiceModeStatus`, `SessionInfo`, `ClusterInfo`, `InfobaseInfo`) определены в том же пакете

## Tasks / Subtasks

- [x] Task 1: Создать пакет `internal/adapter/onec/rac/` (AC: 1)
  - [x] 1.1 Создать директорию `internal/adapter/onec/rac/`
  - [x] 1.2 Создать `interfaces.go` с package declaration и godoc
- [x] Task 2: Определить data types (AC: 5)
  - [x] 2.1 `ClusterInfo` — UUID, имя, хост, порт
  - [x] 2.2 `InfobaseInfo` — UUID, имя, описание
  - [x] 2.3 `ServiceModeStatus` — enabled, message, scheduled_jobs_blocked, active_sessions
  - [x] 2.4 `SessionInfo` — session_id, user_name, app_id, host, started_at, last_active_at
- [x] Task 3: Определить интерфейсы по ISP (AC: 1, 2, 4)
  - [x] 3.1 `ClusterProvider` — `GetClusterInfo(ctx) (*ClusterInfo, error)`
  - [x] 3.2 `InfobaseProvider` — `GetInfobaseInfo(ctx, clusterUUID, infobaseName) (*InfobaseInfo, error)`
  - [x] 3.3 `SessionProvider` — `GetSessions(ctx, clusterUUID, infobaseUUID) ([]SessionInfo, error)`, `TerminateSession(ctx, clusterUUID, sessionID) error`, `TerminateAllSessions(ctx, clusterUUID, infobaseUUID) error`
  - [x] 3.4 `ServiceModeManager` — `EnableServiceMode(ctx, clusterUUID, infobaseUUID, terminateSessions) error`, `DisableServiceMode(ctx, clusterUUID, infobaseUUID) error`, `GetServiceModeStatus(ctx, clusterUUID, infobaseUUID) (*ServiceModeStatus, error)`, `VerifyServiceMode(ctx, clusterUUID, infobaseUUID, expectedEnabled) error`
  - [x] 3.5 `Client` — композитный интерфейс, объединяющий все вышеперечисленные (переименован из RACClient по рекомендации линтера для избежания stuttering `rac.RACClient`)
- [x] Task 4: Создать mock для тестов (AC: 3)
  - [x] 4.1 Создать `mock_rac_client.go` в `internal/adapter/onec/rac/`
  - [x] 4.2 Реализовать `MockRACClient` с функциональными полями (паттерн из `servicemode_test.go`)
  - [x] 4.3 Добавить compile-time проверку `var _ Client = (*MockRACClient)(nil)`
- [x] Task 5: Написать тесты (AC: 3)
  - [x] 5.1 Тест compile-time проверки интерфейса
  - [x] 5.2 Тест MockRACClient — проверка дефолтных и пользовательских реализаций
  - [x] 5.3 Тест что интерфейс полностью покрывает legacy-методы из `internal/rac/`

### Review Follow-ups (AI)

- [ ] [AI-Review][HIGH] SessionInfo.StartedAt/LastActiveAt: time.Time без timezone — RAC возвращает время без timezone, парсинг использует time.Parse без location [interfaces.go:58-60, client.go:309]
- [ ] [AI-Review][MEDIUM] ServiceModeStatus.ActiveSessions заполняется из отдельного RAC-вызова, но контракт интерфейса не документирует это [interfaces.go:44]
- [ ] [AI-Review][MEDIUM] SessionProvider.TerminateAllSessions — orchestration-логика нарушает ISP [interfaces.go:82]
- [ ] [AI-Review][MEDIUM] MockRACClient.VerifyServiceMode дефолт на английском, остальная кодовая база — на русском [ractest/mock.go:138]
- [ ] [AI-Review][LOW] NewMockRACClientWithServiceMode не устанавливает Enable/DisableServiceModeFunc — может маскировать баги [ractest/mock.go:158-174]
- [ ] [AI-Review][LOW] TestCompileTimeInterfaceChecks дублирует compile-time проверки из mock.go:14-20 [interfaces_test.go:15-21]

### Review Follow-ups (AI Code Review #34)

- [ ] [AI-Review][HIGH] SessionInfo.StartedAt/LastActiveAt — time.Time без timezone, RAC возвращает server local time [rac/interfaces.go:47-61]
- [ ] [AI-Review][MEDIUM] ServiceModeStatus.ActiveSessions заполняется из отдельного GetSessions() — TOCTOU race [rac/interfaces.go:42-44]
- [ ] [AI-Review][MEDIUM] TerminateAllSessions — orchestration в interface, нарушает ISP (дубль из Review #31) [rac/interfaces.go:75-83]

## Dev Notes

### Архитектурные ограничения

- **ADR-003: Role-based Interface Segregation** — интерфейсы должны быть маленькими и сфокусированными. НЕ создавать один монолитный интерфейс. Использовать композицию через embedding.
- **ADR-001: Wire DI** — интерфейсы будут инжектиться через Wire. Story 1.7 уже настроила DI-инфраструктуру в `internal/di/`.
- Все комментарии на русском языке (CLAUDE.md требование).

### Существующий код для reference

| Файл | Что содержит |
|------|-------------|
| `internal/rac/rac.go` | Legacy RAC client: `ExecuteCommand`, `GetClusterUUID`, `GetInfobaseUUID` |
| `internal/rac/service_mode.go` | Legacy: `EnableServiceMode`, `DisableServiceMode`, `GetServiceModeStatus`, `GetSessions`, `TerminateSession`, `TerminateAllSessions`, `VerifyServiceMode` |
| `internal/servicemode/servicemode.go` | Legacy Manager interface: `RacClientInterface` — reference для нового `RACClient` |
| `internal/servicemode/servicemode_test.go` | Legacy mock паттерн: `MockRacClient` с функциональными полями |

### Существующие data types в legacy (reference)

```go
// internal/rac/service_mode.go
type ServiceModeStatus struct {
    Enabled        bool
    Message        string
    ActiveSessions int
}

type SessionInfo struct {
    SessionID    string
    UserName     string
    AppID        string
    StartedAt    time.Time
    LastActiveAt time.Time
}
```

### Ключевые решения

1. **Файл размещения**: `internal/adapter/onec/rac/interfaces.go` — по архитектуре, НЕ в `internal/rac/`
2. **Композитный интерфейс**: `RACClient` = `ClusterProvider` + `InfobaseProvider` + `SessionProvider` + `ServiceModeManager`
3. **Data types в том же пакете**: Новые типы НЕ ре-экспортируют legacy; они самостоятельны
4. **Mock в тестовом подпакете**: `ractest/mock.go` или рядом `mock_rac_client.go` — на усмотрение dev

### Паттерн mock из Epic 1

```go
// Функциональные поля для гибкости mock
type MockRACClient struct {
    GetClusterInfoFunc     func(ctx context.Context) (*ClusterInfo, error)
    GetSessionsFunc        func(ctx context.Context, clusterUUID, infobaseUUID string) ([]SessionInfo, error)
    // ... остальные методы
}

func (m *MockRACClient) GetClusterInfo(ctx context.Context) (*ClusterInfo, error) {
    if m.GetClusterInfoFunc != nil {
        return m.GetClusterInfoFunc(ctx)
    }
    return &ClusterInfo{UUID: "test-cluster-uuid", Name: "test-cluster"}, nil
}
```

### Compile-time проверка (обязательно)

```go
var (
    _ RACClient        = (*MockRACClient)(nil)
    _ ClusterProvider   = (*MockRACClient)(nil)
    _ InfobaseProvider  = (*MockRACClient)(nil)
    _ SessionProvider   = (*MockRACClient)(nil)
    _ ServiceModeManager = (*MockRACClient)(nil)
)
```

### Project Structure Notes

- Новый пакет: `internal/adapter/onec/rac/` — директория `adapter/` уже может существовать (проверить)
- Следует паттерну из архитектуры: `adapter/{system}/{subsystem}/`
- НЕ трогать legacy код в `internal/rac/` и `internal/servicemode/` — он продолжает работать в production

### Что НЕ делать

- НЕ реализовывать RAC client (это Story 2.2)
- НЕ менять legacy код
- НЕ добавлять Wire-провайдеры (это будет позже)
- НЕ создавать handler-ы команд (это Stories 2.3-2.7)

### References

- [Source: _bmad-output/project-planning-artifacts/epics/epic-2-service-mode.md#Story 2.1]
- [Source: _bmad-output/project-planning-artifacts/architecture.md#Role-based interfaces]
- [Source: internal/servicemode/servicemode.go — legacy RacClientInterface]
- [Source: internal/rac/service_mode.go — legacy методы и типы]
- [Source: internal/servicemode/servicemode_test.go — mock паттерн]
- [Source: internal/di/ — Wire DI infrastructure from Epic 1]

## Dev Agent Record

### Agent Model Used

Claude Opus 4.5 (claude-opus-4-5-20251101)

### Debug Log References

- Линтер предложил переименовать `RACClient` → `Client` для избежания stuttering (`rac.RACClient`). Применено.
- Все 29 тестов проходят, 0 проблем линтера.
- Существующие тесты проекта не затронуты (регрессий нет).

### Completion Notes List

- Создан пакет `internal/adapter/onec/rac/` с ISP-интерфейсами: `ClusterProvider`, `InfobaseProvider`, `SessionProvider`, `ServiceModeManager`, и композитный `Client`.
- Определены 4 data types: `ClusterInfo`, `InfobaseInfo`, `ServiceModeStatus`, `SessionInfo`. Типы расширены по сравнению с legacy (добавлены `Host` в `SessionInfo`, `ScheduledJobsBlocked` в `ServiceModeStatus`).
- `MockRACClient` с функциональными полями реализован в `ractest/mock.go` с compile-time проверками и вспомогательными конструкторами.
- 29 тестов покрывают compile-time проверки, дефолтное и пользовательское поведение mock, data types и покрытие legacy-методов.
- Legacy код не затронут.

### File List

- `internal/adapter/onec/rac/interfaces.go` (новый) — интерфейсы и data types
- `internal/adapter/onec/rac/ractest/mock.go` (новый) — MockRACClient с compile-time проверками (подпакет ractest)
- `internal/adapter/onec/rac/interfaces_test.go` (новый) — 29 тестов (external test package rac_test)

### Change Log

- 2026-01-27: Реализована Story 2.1 — RAC Adapter Interface. Создан пакет `internal/adapter/onec/rac/` с ISP-интерфейсами, data types, mock и тестами.
- 2026-01-27: Code review — исправлены 4 MEDIUM issues: mock перенесён в подпакет `ractest/` (чистый public API), исправлен godoc пакета, переименован тест legacy-покрытия, добавлена документация зависимости ServiceModeManager от SessionProvider.
