# Epic 8: Закрытие технического долга

**Статус:** 🔴 Не начат
**Приоритет:** Высокий (P0/P1 — блокирует production-ready)
**Риск:** 🟡 Средний
**Stories:** 0/9 (0%)
**FRs:** Покрывает H-1—H-9, M-1—M-11, L-1, Security
**Аудит:** 2026-02-07

---

## 📊 Gap Analysis (Аудит 2026-02-07)

### Статус реализации: 🔴 НЕ НАЧАТ

| Компонент | План | Реализация | Статус |
|-----------|------|------------|--------|
| Alerter интеграция (H-1, H-9) | Вызов alerter.Send() в handlers | ❌ ~5500 строк мёртвого кода | 🔴 |
| RAC Client Factory (H-2) | `rac/factory.go` | ❌ Дублирование в 4 handlers (~260 строк) | 🔴 |
| Production HTTP Clients (H-6) | `adapter/sonarqube/client.go`, `adapter/gitea/client.go` | ❌ 9 фабрик возвращают nil | 🔴 |
| Config рефакторинг (M-4, M-10, M-11) | Единый LoggingConfig, sync.Once | ❌ Dual source of truth | 🔴 |
| DI и Logger (M-3) | `di.InitializeApp(cfg)` в main.go | ❌ Прямые вызовы провайдеров | 🔴 |
| Git2Store backup (H-4) | Полный backup через 1cv8 | ❌ Только метаданные (backup_info.txt) | 🔴 |
| SQ permissions (H-8) | SetProjectPermissions через Web API | ❌ Только логирование | 🔴 |
| Безопасность | Credentials в config, TOCTOU fix | ❌ Hardcoded DefaultUser/DefaultPass | 🔴 |
| Интеграционные тесты (H-3) | RAC enable/disable тесты | ❌ Только unit-тесты парсинга | 🔴 |

### Текущее состояние кода

```
ТЕХНИЧЕСКИЙ ДОЛГ:
├── internal/pkg/alerting/           ~5500 строк (мёртвый код)
│   ├── alerter.go                   ✅ Полная реализация
│   ├── email.go                     ✅ SMTP + TLS + templates
│   ├── telegram.go                  ✅ Bot API + Markdown
│   ├── webhook.go                   ✅ HTTP POST + retry
│   ├── rules.go                     ✅ Rules Engine
│   └── multi.go                     ✅ Multi-channel orchestrator
│   └── НО: Ни один handler не вызывает Send()  🔴
│
├── internal/command/handlers/       ~260 строк дублирования
│   ├── servicemodestatushandler/    createRACClient() 🔴 H-2
│   ├── servicemodeenablehandler/    createRACClient() 🔴 H-2
│   ├── servicemodedisablehandler/   createRACClient() 🔴 H-2
│   └── forcedisconnecthandler/      createRACClient() 🔴 H-2
│
├── internal/command/handlers/sonarqube/  9 фабрик nil 🔴 H-6
│   ├── scanbranch/                  createGiteaClient + createSonarQubeClient
│   ├── scanpr/                      createGiteaClient + createSonarQubeClient
│   ├── reportbranch/                createSonarQubeClient
│   └── projectupdate/               SetProjectPermissions stub 🔴 H-8
│
├── internal/command/handlers/gitea/     3 фабрики nil 🔴 H-6
│   ├── actionmenu/                  createGiteaClient
│   └── testmerge/                   createGiteaClient
│
├── internal/config/config.go        M-1, M-2, M-3, M-4, M-10 🟡
├── internal/pkg/tracing/provider.go M-11 (sync.Once) 🟡
├── internal/pkg/logging/config.go   M-10 (dual source of truth) 🟡
├── internal/constants/constants.go  DefaultUser/DefaultPass hardcoded 🔴 Security
├── internal/adapter/onec/rac/client.go  TOCTOU race condition 🟡
├── Dockerfile.debug                 Delve 0.0.0.0 binding 🟡
│
└── internal/service/sonarqube/      ~729 строк stubs
    ├── branch_scanner_service.go    4 stub-метода
    └── reporting.go                 1 stub-метод + TODO-список
```

### 🔒 Prerequisites

**Зависимости внутри Epic 8:**
- Story 8.7 зависит от Story 8.3 (production SQ/Gitea клиенты)
- Story 8.9 — финальная, после всех остальных

**Внешние зависимости:**
- Epic 1 (Command Registry, Wire DI) — уже реализован
- Epic 2 (RAC Adapter) — уже реализован

### Stories Progress

| Story | Название | Приоритет | Размер | Статус |
|-------|----------|-----------|--------|--------|
| 8.1 | Интеграция Alerter в command handlers | P0 | L | 🔴 Не начат |
| 8.2 | Извлечение RAC Client Factory | P0 | S | 🔴 Не начат |
| 8.3 | Production HTTP клиенты SonarQube и Gitea | P0 | L | 🔴 Не начат |
| 8.4 | Рефакторинг конфигурации | P1 | M | 🔴 Не начат |
| 8.5 | DI и Logger рефакторинг | P1 | M | 🔴 Не начат |
| 8.6 | Git2Store backup и рефакторинг фабрик | P1 | M | 🔴 Не начат |
| 8.7 | SonarQube доработки | P1 | S | 🔴 Ждёт 8.3 |
| 8.8 | Безопасность и race conditions | P0 | S | 🔴 Не начат |
| 8.9 | Тесты, cleanup и документация | P2 | S | 🔴 Ждёт 8.1-8.8 |

---

## Цель

Закрыть **91 TODO** и **334 AI-Review action items**, накопленных в Epics 1-7. Устранить мёртвый код, дублирование, security-проблемы и недореализованные фабрики, блокирующие production-использование NR-команд.

## Ценность

- **Production-ready:** NR-команды SonarQube/Gitea станут работоспособны (H-6)
- **Безопасность:** Устранение hardcoded credentials и TOCTOU race condition
- **Качество кода:** Устранение ~260 строк дублирования, интеграция ~5500 строк alerting
- **Maintainability:** Единый source of truth для конфигураций, DI через Wire

## Волны выполнения

```
Волна 1 (параллельно, без зависимостей):
  [8.2 RAC Factory]  [8.4 Config Cleanup]  [8.8 Security]
         │                    │                    │
         ▼                    ▼                    ▼
Волна 2 (после 8.2 и 8.4):
  [8.5 DI+Logger]  [8.6 Git2Store+Factories]
         │                    │
         ▼                    ▼
Волна 3 (параллельно):
  [8.1 Alerter Integration]  [8.3 HTTP Clients SQ/Gitea]
         │                         │
         ▼                         ▼
Волна 4 (после 8.3):
  [8.7 SQ Features] ◄─────────────┘
         │
         ▼
Волна 5 (финализация):
  [8.9 Tests & Docs]
```

---

## Stories

### Story 8.1: Интеграция Alerter в command handlers (H-1, H-9)

**Приоритет:** P0 | **Размер:** L | **Риск:** Medium
**Prerequisites:** Story 8.5 (DI рефакторинг — logger через DI)

**As a** DevOps-инженер,
**I want** получать алерты при ошибках и успехах команд,
**So that** я оперативно реагирую на проблемы в pipeline.

**Acceptance Criteria:**

- [ ] `main.go` вызывает `di.ProvideAlerter(cfg)` для создания Alerter
- [ ] Alerter передаётся в command handlers через context или middleware
- [ ] При ошибке Execute() автоматически вызывается `alerter.Send()` с severity=Error
- [ ] При успехе критических операций (dbrestore, git2store) — severity=Info
- [ ] Паттерн: post-execution hook в registry или middleware wrapper
- [ ] Существующие тесты alerting проходят без изменений
- [ ] NopAlerter используется при отключённом алертинге (без overhead)
- [ ] `grep -r "TODO(H-1" --include="*.go" | wc -l` возвращает 0
- [ ] `grep -r "TODO(H-9" --include="*.go" | wc -l` возвращает 0

**Technical Notes:**

Текущее состояние:
- `internal/pkg/alerting/` содержит полную реализацию (~5500 строк с тестами): email, telegram, webhook, rules engine, rate limiting
- `internal/di/providers.go:130` — ProvideAlerter() готов, но не вызывается из main.go
- `cmd/apk-ci/main.go:221` — TODO о необходимости интеграции
- Handler interface (`internal/command/handler.go:12`) принимает только `(ctx, cfg)` — нет DI-зависимостей

Подход к интеграции:
1. **Middleware pattern** (рекомендуемый): обёртка вокруг `handler.Execute()` в registry
   ```go
   func executeWithAlerting(h Handler, ctx context.Context, cfg *config.Config, a alerting.Alerter) error {
       err := h.Execute(ctx, cfg)
       if err != nil {
           a.Send(ctx, alerting.Alert{Severity: alerting.Error, Command: h.Name(), Error: err})
       }
       return err
   }
   ```
2. **Context injection**: передача Alerter через `context.WithValue()`
3. **Handler interface extension**: добавление `SetAlerter(alerting.Alerter)` (опционально)

Файлы для изменения:
- `cmd/apk-ci/main.go` — создание Alerter, передача в execution path
- `internal/command/registry.go` или новый `internal/command/middleware.go` — middleware
- `internal/di/providers.go` — удаление TODO-комментариев H-1/H-9

---

### Story 8.2: Извлечение RAC Client Factory (H-2)

**Приоритет:** P0 | **Размер:** S | **Риск:** Low
**Prerequisites:** —

**As a** разработчик,
**I want** иметь единую фабрику RAC-клиента,
**So that** изменения в создании клиента применяются в одном месте.

**Acceptance Criteria:**

- [ ] Создан `internal/adapter/onec/rac/factory.go` с функцией `NewClientFromConfig(cfg *config.Config) (Client, error)`
- [ ] Функция содержит всю логику: получение сервера, fallback на RacConfig, timeout с warning, пароли из SecretConfig
- [ ] 4 handler-а (servicemodestatushandler, servicemodeenablehandler, servicemodedisablehandler, forcedisconnecthandler) используют `rac.NewClientFromConfig(cfg)` вместо локальных `createRACClient()`
- [ ] Локальные `createRACClient()` удалены из всех 4 handlers
- [ ] Все существующие тесты проходят
- [ ] `grep -r "TODO(H-2)" --include="*.go" | wc -l` возвращает 0 (кроме handler.go:16 для logger — это M-3)

**Technical Notes:**

Текущий дублированный код (~65 строк в каждом handler):
- `servicemodestatushandler/handler.go:294-357`
- `servicemodeenablehandler/handler.go:309-359`
- `servicemodedisablehandler/handler.go:270-320`
- `forcedisconnecthandler/handler.go:346-389+`

Функция содержит:
1. Проверку `cfg.AppConfig != nil`
2. Получение сервера через `cfg.GetOneServer()` с fallback на `cfg.RacConfig`
3. Парсинг порта (default 1545)
4. Расчёт timeout (default 30s, warning > 5min)
5. Заполнение `rac.ClientOptions` с паролями из `SecretConfig`
6. Диагностические предупреждения при пустых credentials
7. Вызов `rac.NewClient(opts)`

Новый файл `internal/adapter/onec/rac/factory.go`:
```go
// NewClientFromConfig создаёт RAC клиент из конфигурации приложения.
// Извлечён из дублирования в 4 handler-ах (H-2).
func NewClientFromConfig(cfg *config.Config) (Client, error) { ... }
```

---

### Story 8.3: Production HTTP клиенты SonarQube и Gitea (H-6)

**Приоритет:** P0 | **Размер:** L | **Риск:** High
**Prerequisites:** —

**As a** DevOps-инженер,
**I want** чтобы NR-команды SonarQube и Gitea работали в production,
**So that** я могу использовать nr-sq-scan-branch, nr-test-merge и другие команды.

**Acceptance Criteria:**

- [ ] Создан `internal/adapter/sonarqube/client.go` — реализация `sonarqube.Client` через HTTP/REST API
- [ ] Реализованы все методы интерфейсов: ProjectsAPI, AnalysesAPI, IssuesAPI, QualityGatesAPI, MetricsAPI
- [ ] Создан `internal/adapter/gitea/client.go` — реализация `gitea.Client` через Gitea API
- [ ] Реализованы все методы интерфейсов: PRReader, CommitReader, FileReader, BranchManager, ReleaseReader, IssueManager, PRManager, RepositoryWriter, TeamReader, OrgReader
- [ ] 9 фабрик в handlers заменены на вызов `createSonarQubeClient(cfg)`/`createGiteaClient(cfg)` с реальной реализацией
- [ ] Unit-тесты для обоих клиентов (httptest mock server)
- [ ] Интеграция с существующими mock-ами в `sonarqubetest/` и `giteatest/`
- [ ] `grep -r "TODO(H-6)" --include="*.go" | wc -l` возвращает 0
- [ ] Все 7 затронутых NR-команд работают при наличии SQ/Gitea конфигурации

**Technical Notes:**

Затронутые handlers (9 фабрик, 7 файлов):
1. `sonarqube/scanpr/handler.go:232` — createGiteaClient
2. `sonarqube/scanpr/handler.go:243` — createSonarQubeClient
3. `sonarqube/scanbranch/handler.go:220` — createGiteaClient
4. `sonarqube/scanbranch/handler.go:234` — createSonarQubeClient
5. `sonarqube/reportbranch/handler.go:330` — createSonarQubeClient
6. `gitea/actionmenu/handler.go:245` — createGiteaClient
7. `gitea/testmerge/handler.go:238` — createGiteaClient

Интерфейсы определены в:
- `internal/adapter/sonarqube/interfaces.go` (272-279) — композитный Client
- `internal/adapter/gitea/interfaces.go` (359-371) — композитный Client

Паттерн для подражания: `rac.NewClient()` из `internal/adapter/onec/rac/client.go`

Конфигурация доступна через:
- `cfg.SonarQubeConfig` — URL, Token, Organization
- `cfg.ProjectConfig` — Owner, Repo (для Gitea)
- `cfg.AppConfig.Gitea` — BaseURL, Token

Миграция с legacy: `doc.go:38` — после реализации Client adapter удалить дублирующиеся структуры из `internal/entity/gitea/gitea.go`

---

### Story 8.4: Рефакторинг конфигурации (M-1, M-2, M-3, M-4, M-10, M-11)

**Приоритет:** P1 | **Размер:** M | **Риск:** Medium
**Prerequisites:** —

**As a** разработчик,
**I want** иметь единый source of truth для конфигурации,
**So that** изменения не требуют синхронизации между двумя структурами.

**Acceptance Criteria:**

- [ ] **M-10:** `config.LoggingConfig` и `logging.Config` объединены — одна структура используется обоими пакетами
- [ ] **M-4:** Bool поля с `env-default:"true"` (Compress, UseTLS) обрабатываются корректно: документирован workaround через `getDefault*Config()` или реализован proper fix через pointer types (`*bool`)
- [ ] **M-11:** `otel.SetTracerProvider()` защищён через `sync.Once` в `internal/pkg/tracing/provider.go:85`
- [ ] **M-3:** Webhook Headers: добавлен парсинг `"Key=Val,Key2=Val2"` из env переменной `BR_ALERTING_WEBHOOK_HEADERS` или задокументировано ограничение
- [ ] **M-2:** `SonarQubeConfig.Validate()` и `ScannerConfig.Validate()` вызываются в `MustLoad()` (config.go:1242)
- [ ] **M-1:** При ошибке валидации AlertingConfig заменяется на `getDefaultAlertingConfig()` вместо просто `Enabled=false` (config.go:1304)
- [ ] Все существующие тесты проходят
- [ ] `grep -r "TODO (M-10" --include="*.go" | wc -l` возвращает 0
- [ ] `grep -r "TODO (M-4" --include="*.go" | wc -l` возвращает 0
- [ ] `grep -r "TODO (M-11" --include="*.go" | wc -l` возвращает 0
- [ ] `grep -r "TODO (M-2" --include="*.go" | wc -l` возвращает 0
- [ ] `grep -r "TODO (M-1" --include="*.go" | wc -l` возвращает 0

**Technical Notes:**

M-10 (Dual source of truth):
- `internal/config/config.go:355-385` — `LoggingConfig` с env tags
- `internal/pkg/logging/config.go:50-84` — `Config` без env tags
- Решение: `logging.Config` становится canonical; `config.LoggingConfig` embed-ит или алиасит `logging.Config`
- Либо: `config.LoggingConfig` остаётся для env-парсинга, а `toLoggingConfig()` конвертирует в `logging.Config`

M-4 (bool env-default):
- `config.go:381` — `Compress bool env-default:"true"` — YAML `false` перезаписывается cleanenv
- `config.go:477` — `UseTLS bool env-default:"true"` — аналогично
- Workaround через `getDefaultLoggingConfig()` уже работает для YAML-source
- Proper fix: `*bool` или документация ограничения

M-11 (sync.Once):
- `tracing/provider.go:85` — `otel.SetTracerProvider(tp)` без sync.Once
- Fix: обернуть в `var setProviderOnce sync.Once`

M-2 (Validation):
- `config.go:1242-1256` — Validate() существует, но не вызывается
- Fix: добавить вызов по аналогии с `validateAlertingConfig` (строки 1302-1315)

M-1 (Config replacement):
- `config.go:1304` — `cfg.AlertingConfig.Enabled = false` оставляет невалидные поля
- Fix: `cfg.AlertingConfig = getDefaultAlertingConfig()`

---

### Story 8.5: DI и Logger рефакторинг (M-3, частично H-2)

**Приоритет:** P1 | **Размер:** M | **Риск:** Medium
**Prerequisites:** Story 8.2 (RAC Factory), Story 8.4 (Config cleanup)

**As a** разработчик,
**I want** чтобы main.go использовал Wire DI для инициализации,
**So that** все зависимости управляются централизованно.

**Acceptance Criteria:**

- [ ] `main.go:198` использует `di.InitializeApp(cfg)` вместо прямых вызовов провайдеров
- [ ] Handlers используют DI-инжектированный logger вместо `slog.Default()` (handler.go:16)
- [ ] `migratehandler/handler.go:107` — `buildLegacyToNRMapping` кэшируется через `sync.Once`
- [ ] Alerter, MetricsCollector, TracerShutdown получаются из `di.App` struct
- [ ] `grep -r "TODO (M-3" --include="*.go" | wc -l` возвращает 0 (кроме email.go:312 — RFC 2047, отдельная задача)

**Technical Notes:**

Текущее состояние main.go:
```go
// Строка 198-204:
logAdapter := logging.NewSlogAdapter(l)
metricsCollector := di.ProvideMetricsCollector(cfg, logAdapter)
// ...отдельные вызовы провайдеров
```

Целевое состояние:
```go
app, cleanup, err := di.InitializeApp(cfg)
defer cleanup()
// app.Alerter, app.MetricsCollector, app.TracerShutdown — готовы
```

DI Logger для handlers:
- Текущий Handler interface: `Execute(ctx context.Context, cfg *config.Config) error`
- Вариант 1: Logger через context — `logging.FromContext(ctx)`
- Вариант 2: Расширение config — `cfg.Logger`
- Вариант 3: Functional options в конструкторе handler

Файлы затрагиваемые:
- `cmd/apk-ci/main.go` — основная точка интеграции
- `internal/command/handler.go` — расширение interface или context helper
- `internal/command/handlers/migratehandler/handler.go:107` — sync.Once для mapping

---

### Story 8.6: Git2Store backup и рефакторинг фабрик (H-4, H-5, M-1, M-2)

**Приоритет:** P1 | **Размер:** M | **Риск:** Medium
**Prerequisites:** Story 8.5 (DI рефакторинг)

**As a** DevOps-инженер,
**I want** полноценный backup хранилища 1C перед git2store операциями,
**So that** я могу восстановить хранилище при сбоях.

**Acceptance Criteria:**

- [ ] **H-4:** `createBackupProduction()` выполняет полный backup через 1cv8 Designer:
  - `/ConfigurationRepositoryDumpCfg` для экспорта конфигурации
  - `/ConfigurationRepositoryReport` для сохранения версии
  - Копирование локальных файлов (если файловое хранилище)
- [ ] **H-5:** Фабрики (`gitFactory`, `convertConfigFactory`, `backupCreator`, `tempDbCreator`) вынесены в `internal/factory/` или подключены через Wire DI
- [ ] **M-1:** `createstoreshandler/production.go` рефакторен на struct pattern (по аналогии с `storebindhandler.defaultConvertLoader`)
- [ ] **M-2:** EPF валидация вынесена в общую утилиту `internal/pkg/validation/epf.go` (дублирование в executeepfhandler:254)
- [ ] Backup создаёт реальную копию данных, а не только метаданные
- [ ] Fallback: при недоступности 1cv8 — создаётся backup_info.txt с инструкциями
- [ ] `grep -r "TODO.*H-4" --include="*.go" | wc -l` возвращает 0
- [ ] `grep -r "TODO.*H-5" --include="*.go" | wc -l` возвращает 0

**Technical Notes:**

H-4 (Backup):
- `git2storehandler/handler.go:1113` — текущая реализация создаёт только `backup_info.txt`
- Полный backup требует доступа к 1cv8 CLI (путь из `cfg.AppConfig.Paths.OneC`)
- Команды 1cv8 для backup:
  ```
  1cv8 DESIGNER /F <db> /ConfigurationRepositoryDumpCfg <output.cf>
  1cv8 DESIGNER /F <db> /ConfigurationRepositoryReport <output.txt>
  ```

H-5 (Фабрики):
- `git2storehandler/handler.go:232-247` — локальные фабрики
- Создать `internal/factory/` с переиспользуемыми фабриками
- Альтернатива: Wire providers в `internal/di/`

M-1 (Struct pattern):
- `createstoreshandler/production.go:3` — production-функции не обёрнуты в struct
- Рефакторинг по аналогии с `storebindhandler.defaultConvertLoader`

M-2 (EPF валидация):
- `executeepfhandler/handler.go:254` — дублирует `enterprise.EpfExecutor.validateEpfURL()`
- Вынести в `internal/pkg/validation/epf.go`

---

### Story 8.7: SonarQube доработки (H-8)

**Приоритет:** P1 | **Размер:** S | **Риск:** Medium
**Prerequisites:** Story 8.3 (Production SQ client)

**As a** DevOps-инженер,
**I want** чтобы nr-sq-project-update синхронизировал администраторов в SonarQube,
**So that** права доступа проекта актуальны.

**Acceptance Criteria:**

- [ ] **H-8:** `SetProjectPermissions` реализован через SonarQube Web API `/api/permissions/add_user`
- [ ] `projectupdate/handler.go:335` — вызывает `sqClient.SetProjectPermissions()` вместо логирования
- [ ] **H-8:** `ScanResult` в scanpr заполняется метриками: NewIssues, NewBugs, NewVulnerabilities, NewCodeSmells
- [ ] `scanpr/handler.go:71` — вызывает `sonarqube.GetMeasures()` для получения метрик
- [ ] **L-1:** Visibility в scanbranch/scanpr конфигурируется через `cfg.SonarQubeConfig.DefaultVisibility` (default: "private")
- [ ] **SourcePath** заполняется в scanbranch:385 и scanpr:358
- [ ] `grep -r "TODO(H-8)" --include="*.go" | wc -l` возвращает 0
- [ ] `grep -r "TODO(L-1)" --include="*.go" | wc -l` возвращает 0

**Technical Notes:**

H-8 (SetProjectPermissions):
- `projectupdate/handler.go:335` — текущий код только логирует администраторов
- SonarQube Web API: `POST /api/permissions/add_user` с параметрами `projectKey`, `login`, `permission`
- Permissions: `admin`, `codeviewer`, `issueadmin`, `securityhotspotadmin`, `scan`, `user`
- Удаление: `POST /api/permissions/remove_user`

H-8 (ScanResult метрики):
- `scanpr/handler.go:71` — ScanResult содержит только QualityGateStatus
- SonarQube API: `GET /api/measures/component?component=<key>&metricKeys=new_bugs,new_vulnerabilities,new_code_smells,new_security_hotspots`
- Добавить поля: `NewIssues`, `NewBugs`, `NewVulnerabilities`, `NewCodeSmells`

L-1 (Visibility):
- `scanbranch/handler.go:358` — hardcoded `"private"`
- Добавить `DefaultVisibility string` в `SonarQubeConfig`

---

### Story 8.8: Безопасность и race conditions

**Приоритет:** P0 | **Размер:** S | **Риск:** Medium
**Prerequisites:** —

**As a** security-инженер,
**I want** устранить hardcoded credentials и задокументировать ограничения,
**So that** код соответствует security best practices.

**Acceptance Criteria:**

- [ ] **Security:** `DefaultUser`/`DefaultPass` (`constants.go:227-230`) вынесены из констант в конфигурацию:
  - `BR_TEMP_DB_USER` / `BR_TEMP_DB_PASS` с fallback на текущие значения ("gitops"/"gitops")
  - `internal/constants/constants.go` больше не содержит паролей
- [ ] **TOCTOU:** Race condition в `rac/client.go:71-79` задокументирован как accepted risk с комментарием CWE-367, или добавлен retry при `exec.CommandContext` failure
- [ ] **Shadow-run:** `captureStdoutMu` в `shadowrun/shadowrun.go:124` задокументирован с TODO для v2.0.0 (io.Writer рефакторинг), или рефакторен на io.Writer pattern
- [ ] **Delve:** `Dockerfile.debug:45` — добавлен `--listen=127.0.0.1:2345` вместо `--listen=:2345`
- [ ] Все изменения покрыты тестами

**Technical Notes:**

Security (Hardcoded credentials — CWE-798):
- `internal/constants/constants.go:227-230` — `DefaultUser = "gitops"`, `DefaultPass = "gitops"`
- Используется в: `git2storehandler/handler.go:768`, `entity/one/convert/convert.go:163,177`
- Fix: вынести в `config.AppConfig.Defaults.TempDbUser` / `TempDbPass`
- Env: `BR_TEMP_DB_USER`, `BR_TEMP_DB_PASS` с `env-default:"gitops"`

TOCTOU (CWE-367):
- `rac/client.go:71-79` — `os.Stat()` проверяет существование файла перед `exec.CommandContext()`
- Между проверкой и использованием файл может быть изменён
- Риск: LOW (требует локального доступа, CI/CD — доверенное окружение)
- Варианты: (a) документировать как accepted risk, (b) убрать os.Stat и полагаться на exec error

Shadow-run mutex:
- `shadowrun/shadowrun.go:124-128` — глобальный mutex для подмены os.Stdout
- Полный рефакторинг на io.Writer требует изменения Handler interface (v2.0.0)
- Текущая реализация безопасна: slog → stderr, shadow-run последовательный

Delve:
- `Dockerfile.debug:45` — `--listen=:2345` = 0.0.0.0 внутри контейнера
- Fix: `--listen=127.0.0.1:2345` или документировать обязательный `-p 127.0.0.1:2345:2345`
- Уже есть комментарий-предупреждение (строки 40-44)

---

### Story 8.9: Тесты, cleanup и документация

**Приоритет:** P2 | **Размер:** S | **Риск:** Low
**Prerequisites:** Stories 8.1-8.8

**As a** разработчик,
**I want** чтобы все оставшиеся TODO были закрыты и тесты актуальны,
**So that** кодовая база чиста от технического долга.

**Acceptance Criteria:**

- [ ] **H-3:** Интеграционные тесты RAC enable/disable добавлены в `client_test.go` (или задокументированы как requiring 1C environment)
- [ ] **H-3:** Тест ротации логов по размеру добавлен в `factory_test.go:478`
- [ ] SonarQube stubs в `internal/service/sonarqube/` — дореализованы или удалены:
  - `branch_scanner_service.go` — 4 stub-метода (GetBranchScanHistory, CancelBranchScan, ValidateBranchForScanning, performQualityGateCheck)
  - `reporting.go` — GenerateBranchReport stub
  - `service.go` — UpdateProjectAdministrators, ListProjectsWithFilter, IntegrateWithService stubs
- [ ] `internal/app/app.go` TODOs закрыты:
  - Строка 221-222: FirstCommitHash/LastCommitHash заполняются
  - Строка 1238-1254: PR number получается из конфигурации
- [ ] `make check` проходит (fmt, vet, lint, test)
- [ ] `make test-coverage` — покрытие не снизилось
- [ ] `grep -r "TODO(H-" --include="*.go" | wc -l` возвращает 0 (кроме H-7 — deprecated aliases, Epic 7)
- [ ] `grep -r "TODO(M-" --include="*.go" | wc -l` возвращает 0 (кроме M-3/email.go:312 — RFC 2047 edge case)
- [ ] MEMORY.md обновлён: раздел "Технический долг" помечен как закрытый

**Technical Notes:**

H-3 (Интеграционные тесты):
- `client_test.go:13` — TODO об отсутствии интеграционных тестов
- Тесты RAC требуют реального rac бинарника и 1C кластера
- Вариант: build tag `//go:build integration` для запуска в CI с 1C окружением
- Минимум: тесты формирования RAC-команд (аргументы, парсинг stdout)

H-3 (Ротация логов):
- `factory_test.go:478` — тест записи > MaxSize и проверки ротации
- Требует создания файла > MaxSize MB → проверка backup файлов

SonarQube stubs cleanup:
- `branch_scanner_service.go` ~408 строк — 4 stub-метода возвращают пустые значения
- `reporting.go` ~323 строки — GenerateBranchReport() логирует warning и возвращает nil
- `service.go` — 3 stub-метода
- Решение: (a) реализовать через production SQ client (Story 8.3), (b) удалить неиспользуемые stubs

App.go TODOs:
- `app.go:221-222` — FirstCommitHash/LastCommitHash нужны для SQ report
- Решение: получать через Gitea API (`giteaClient.GetBranchCommitRange()`)
- `app.go:1238-1254` — PR number = 0
- Решение: добавить `BR_PR_NUMBER` в `config.Config` или получать из Gitea API

---

## Risk Assessment

| Риск | Вероятность | Импакт | Митигация |
|------|-------------|--------|-----------|
| HTTP клиенты SQ/Gitea (8.3) нарушают существующие тесты | Средняя | Высокий | Сохранять совместимость mock-интерфейсов |
| Alerter middleware меняет поведение handlers | Низкая | Средний | NopAlerter по умолчанию, middleware только добавляет пост-обработку |
| M-10 рефакторинг ломает env-парсинг | Средняя | Высокий | Таблица тестов для всех env-переменных логирования |
| Backup через 1cv8 требует версию платформы | Средняя | Средний | Fallback на metadata backup при отсутствии 1cv8 |
| TOCTOU fix убирает полезную раннюю диагностику | Низкая | Низкий | Документировать как accepted risk, не убирать os.Stat |

---

## Покрытие TODO

| Идентификатор | Описание | Story | Файлы |
|---------------|----------|-------|-------|
| H-1 | Alerter не интегрирован | 8.1 | main.go:221, di/providers.go:130 |
| H-2 | createRACClient дублирование | 8.2, 8.5 | 4 handler файла |
| H-3 | Нет интеграционных тестов | 8.9 | client_test.go:13, factory_test.go:478 |
| H-4 | Backup хранилища 1C | 8.6 | git2storehandler:1113 |
| H-5 | Рефакторинг фабрик | 8.6 | git2storehandler:232 |
| H-6 | Production HTTP clients | 8.3 | 9 мест в 7 handler-файлах |
| H-7 | Deprecated aliases | *(Epic 7, не трогаем)* | ~14 мест |
| H-8 | SQ permissions/metrics | 8.7 | projectupdate:335, scanpr:71 |
| H-9 | Alerter dead code | 8.1 | main.go:221 |
| M-1 | Config validation replacement | 8.4, 8.6 | config.go:1304, production.go:3 |
| M-2 | Validation duplication | 8.4, 8.6 | config.go:1242, executeepfhandler:254 |
| M-3 | DI/Wire integration | 8.5 | main.go:198, config.go:524, migratehandler:107 |
| M-4 | Bool env-default issues | 8.4 | config.go:381, 477 |
| M-10 | Dual source of truth | 8.4 | config.go:355, logging/config.go |
| M-11 | sync.Once для tracing | 8.4 | tracing/provider.go:85 |
| L-1 | Visibility hardcoded | 8.7 | scanbranch:358 |
| Security | Hardcoded credentials | 8.8 | constants.go:227-230 |
| Security | TOCTOU race condition | 8.8 | rac/client.go:71-79 |
| Security | Delve 0.0.0.0 | 8.8 | Dockerfile.debug:45 |
| Security | Shadow-run stdout mutex | 8.8 | shadowrun.go:124 |

---

## Definition of Done

- [ ] `make check` проходит без ошибок (fmt, vet, lint, test)
- [ ] `grep -r "TODO(H-" --include="*.go" | wc -l` = 0 (кроме H-7 — Epic 7)
- [ ] `grep -r "TODO(M-" --include="*.go" | wc -l` = 0 (кроме M-3/email.go RFC 2047)
- [ ] `make test-coverage` — покрытие не снизилось
- [ ] Все 7 NR-команд SQ/Gitea функциональны в production (Story 8.3)
- [ ] Alerter интегрирован — email/telegram/webhook алерты работают
- [ ] Нет hardcoded credentials в константах
- [ ] MEMORY.md обновлён

---

## Связанные документы

- [Epic Overview](./index.md)
- [All Epics](./index.md#карта-эпиков)
- [FR Coverage](./fr-coverage.md)
- [Epic 7: Finalization](./epic-7-finalization.md) — H-7 deprecated aliases остаются в Epic 7

---

_Последнее обновление: 2026-02-07_
_Аудит проведён: 2026-02-07 (Epic 8 — Tech Debt Closure)_
