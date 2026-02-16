# Декомпозиция задач рефакторинга benadis-runner

**Дата:** 2025-11-26
**Автор:** XoR
**Версия:** 1.0

---

## Общая информация

- **Выделенный бюджет:** 40 часов/месяц на рефакторинг
- **Исполнитель:** 1 разработчик
- **Методология оценки:** Пессимистичная (включает тестирование, code review, документацию)

---

## Сводная таблица по эпикам

| Epic | Название | Историй | Общая оценка | Месяцев при 40ч |
|------|----------|---------|--------------|-----------------|
| 1 | Architectural Foundation | 9 | 72 ч | 2 |
| 2 | Service Mode Management | 8 | 56 ч | 1.5 |
| 3 | Database Operations | 6 | 48 ч | 1.5 |
| 4 | Configuration Sync | 7 | 64 ч | 2 |
| 5 | Quality & Integration | 9 | 56 ч | 1.5 |
| 6 | Advanced Observability | 9 | 72 ч | 2 |
| 7 | Finalization | 10 | 56 ч | 1.5 |
| **ИТОГО** | | **58** | **424 ч** | **~11 месяцев** |

---

## Epic 1: Architectural Foundation

**Цель:** Создать архитектурную основу для новых команд
**Общая оценка:** 72 часа (2 месяца)

### Задачи Epic 1

| ID | Задача | Описание | Результат | Оценка | Зависимости |
|----|--------|----------|-----------|--------|-------------|
| **1.1** | **Command Registry с self-registration** | Создать пакет `internal/command/` с Registry pattern. Handler interface: `Name()`, `Execute(ctx, cfg)`. Регистрация через `init()`. Main.go: fallback на legacy switch. | Файлы: `registry.go`, `handler.go`. Тесты. Новые команды добавляются без изменения main.go. | 8 ч | — |
| **1.2** | **NR-Migration Bridge (DeprecatedBridge)** | Реализовать `RegisterWithAlias()` для поддержки deprecated команд. Warning в stderr при вызове старого имени. | Файл: `deprecated.go`. Механизм миграции с warning. | 4 ч | 1.1 |
| **1.3** | **OutputWriter + Structured Errors** | Создать пакет `internal/pkg/output/` с Writer interface. Форматы: text, JSON. AppError с Code/Message/Cause. Golden tests для JSON. | Файлы: `writer.go`, `text.go`, `json.go`, `errors.go`. JSON Schema. | 12 ч | — |
| **1.4** | **Logger interface + slog adapter** | Создать пакет `internal/pkg/logging/` с Logger interface поверх slog. Уровни: DEBUG, INFO, WARN, ERROR. Вывод только в stderr. | Файлы: `logger.go`, `slog.go`. Golangci-lint правило для fmt.Print*. | 8 ч | — |
| **1.5** | **Trace ID generation** | Генерация UUID v4 trace_id при старте команды. Добавление в context. Включение в логи и JSON output. | Файл: `internal/pkg/tracing/traceid.go`. Корреляция логов. | 4 ч | 1.4 |
| **1.6** | **Config extensions** | Расширить `internal/config/` секциями: `implementations`, `logging`. Optional поля с defaults. Backward compatibility test. | Структуры: `ImplementationsConfig`, `LoggingConfig`. Тесты с production конфигами. | 8 ч | — |
| **1.7** | **Wire DI setup + providers** | Создать пакет `internal/di/` с Wire definitions. Providers: Config, Logger, OutputWriter. `go generate` в CI. | Файлы: `wire.go`, `providers.go`, `wire_gen.go`. Unit-тесты providers. | 12 ч | 1.1, 1.3, 1.4 |
| **1.8** | **First NR-command: nr-version** | Реализовать nr-version: версия, Go версия, дата сборки. Через Registry. JSON/text output. Makefile target с ldflags. | Файл: `handlers/version/version.go`. Proof of concept архитектуры. | 8 ч | 1.1-1.7 |
| **1.9** | **Auto-generated help from Registry** | Команда help с автогенерацией списка команд из Registry. Группировка: NR vs legacy. Пометка [deprecated]. | Файл: `help.go`. Description() в Handler interface. | 8 ч | 1.1, 1.2, 1.8 |

### График Epic 1

```
Месяц 1 (40ч):
├── 1.1 Command Registry (8ч)
├── 1.3 OutputWriter + Errors (12ч)
├── 1.4 Logger (8ч)
├── 1.6 Config extensions (8ч)
└── Частично 1.2 (4ч)

Месяц 2 (32ч):
├── 1.2 DeprecatedBridge (завершение)
├── 1.5 Trace ID (4ч)
├── 1.7 Wire DI (12ч)
├── 1.8 nr-version (8ч)
└── 1.9 Help (8ч)
```

---

## Epic 2: Service Mode Management

**Цель:** Proof of Concept — первые реальные команды в production
**Общая оценка:** 56 часов (1.5 месяца)
**Зависимости:** Epic 1

### Задачи Epic 2

| ID | Задача | Описание | Результат | Оценка | Зависимости |
|----|--------|----------|-----------|--------|-------------|
| **2.1** | **RAC Adapter Interface** | Создать интерфейс RACClient в `internal/adapter/onec/rac/`. Методы: GetClusterInfo, GetInfobases, GetSessions, SetServiceMode. | Файл: `interfaces.go`. Возможность мокирования. | 4 ч | Epic 1 |
| **2.2** | **RAC Client Implementation** | Реализовать RACClient поверх существующего `internal/rac/`. Парсинг ошибок в AppError. Timeout через config. | Файл: `client.go`. Интеграция с legacy кодом. | 8 ч | 2.1 |
| **2.3** | **nr-service-mode-status** | NR-команда для проверки статуса. Вывод: enabled/disabled, message, scheduled_jobs_blocked. JSON/text format. | Файл: `handlers/servicemode/status.go`. Domain: `domain/servicemode/`. | 8 ч | 2.1, 2.2 |
| **2.4** | **Session Info (FR66)** | Расширить status информацией о сессиях: active_sessions_count, sessions[] с user_name, host, started_at. | Расширение 2.3. Pain point: "Нет инфо о сессиях". | 4 ч | 2.3 |
| **2.5** | **nr-service-mode-enable** | NR-команда включения. Параметры: message, permission_code. Идемпотентность (повторный вызов = success). | Файл: `handlers/servicemode/enable.go`. Блокировка пользователей. | 8 ч | 2.3 |
| **2.6** | **Force Disconnect Sessions (FR9)** | Флаг BR_FORCE_DISCONNECT для принудительного завершения сессий. Grace period. По умолчанию выключено. | Расширение 2.5. Опасная операция с явным флагом. | 6 ч | 2.5 |
| **2.7** | **nr-service-mode-disable** | NR-команда отключения. Идемпотентность. Разблокировка регулярных заданий. | Файл: `handlers/servicemode/disable.go`. | 6 ч | 2.5 |
| **2.8** | **State-Aware Execution (FR60-62)** | Проверка текущего состояния перед операцией. Поля: already_enabled, state_changed. Паттерн: Check → Act → Verify. | Идемпотентные операции. JSON: state_changed field. | 8 ч | 2.5, 2.7 |

### Контрольная точка

После Epic 2: **3 NR-команды** (service-mode-status, enable, disable) должны использоваться минимум в **3 production pipelines**.

---

## Epic 3: Database Operations

**Цель:** Перенести критичные операции с БД на новую архитектуру
**Общая оценка:** 48 часов (1.5 месяца)
**Зависимости:** Epic 1, Epic 2 (для service mode при restore)

### Задачи Epic 3

| ID | Задача | Описание | Результат | Оценка | Зависимости |
|----|--------|----------|-----------|--------|-------------|
| **3.1** | **MSSQL Adapter Interface** | Интерфейс DatabaseRestorer в `internal/adapter/mssql/`. Методы: Restore, GetBackupSize, GetDatabaseSize. | Файл: `interfaces.go`. Мокирование для тестов. | 4 ч | Epic 1 |
| **3.2** | **nr-dbrestore с auto-timeout** | NR-команда restore. Auto-timeout по размеру backup (size_gb * 10 + 5 мин). IsProduction проверка — НИКОГДА в production. | Файл: `handlers/database/restore.go`. Формула таймаута. | 12 ч | 3.1, Epic 2 |
| **3.3** | **Progress Bar (FR67)** | Progress bar для операций > 30 сек. Формат: `[=====>    ] 45% | ETA: 2m 30s`. В stderr, не ломает JSON. | Пакет: `internal/pkg/progress/`. Pain point: "Нет прогресса". | 8 ч | 3.2 |
| **3.4** | **nr-dbupdate** | NR-команда обновления структуры. Двойное выполнение для расширений. Auto-deps: service mode если флаг. Summary: кол-во изменённых объектов. | Файл: `handlers/database/update.go`. | 8 ч | 3.2 |
| **3.5** | **nr-create-temp-db** | NR-команда создания временной файловой БД. Параметры: extensions, TTL. Путь в результате. | Файл: `handlers/database/createtemp.go`. Тестовые окружения. | 6 ч | Epic 1 |
| **3.6** | **Dry-run режим (FR58)** | BR_DRY_RUN=true для вывода плана без выполнения. Паттерн: BuildPlan() → return plan | ExecutePlan(). JSON: dry_run field. | Расширение 3.2, 3.4. Безопасная проверка. | 10 ч | 3.2, 3.4 |

### ⚠️ Критические риски Epic 3

- **IsProduction проверка обязательна** — restore В production базу ЗАПРЕЩЁН
- **DetermineSrcAndDstServers()** — продуктивные ИЗ продуктивных В тестовые
- **Backup перед деструктивными операциями** — документировать процедуру

---

## Epic 4: Configuration Sync

**Цель:** Полный цикл EDT → Store → DB на новой архитектуре
**Общая оценка:** 64 часа (2 месяца)
**Зависимости:** Epic 1, Epic 3

### Задачи Epic 4

| ID | Задача | Описание | Результат | Оценка | Зависимости |
|----|--------|----------|-----------|--------|-------------|
| **4.1** | **1C Operations Factory (FR18)** | Strategy + Factory в `internal/adapter/onec/`. Выбор реализации через config: 1cv8/ibcmd/native. Wire provider. | Файл: `factory.go`. Сменные реализации. | 10 ч | Epic 1 |
| **4.2** | **nr-store2db** | NR-команда загрузки из хранилища в БД. Параметр: версия или latest. Progress этапов: connecting → loading → applying. | Файл: `handlers/store/store2db.go`. | 10 ч | 4.1, Epic 2 |
| **4.3** | **nr-storebind** | NR-команда привязки хранилища к БД. Credentials из secret.yaml. | Файл: `handlers/store/bind.go`. | 6 ч | Epic 1 |
| **4.4** | **nr-create-stores** | NR-команда инициализации хранилищ. Список расширений из project.yaml. Summary: созданные хранилища. | Файл: `handlers/store/create.go`. | 6 ч | Epic 1 |
| **4.5** | **nr-convert (FR19-20)** | NR-команда конвертации EDT ↔ XML. Параметры: source, target, direction. Инструмент через config. | Файл: `handlers/convert/convert.go`. | 8 ч | 4.1 |
| **4.6** | **nr-git2store** | NR-команда синхронизации Git → Store. Этапы: clone → checkout → convert → init DB → apply → commit. При ошибке: rollback + отчёт. | Файл: `handlers/store/git2store.go`. Самый сложный workflow. | 16 ч | 4.2, 4.3, 4.5 |
| **4.7** | **nr-execute-epf** | NR-команда выполнения внешней обработки. Параметры: path, infobase, params, timeout. | Файл: `handlers/convert/executeepf.go`. | 8 ч | Epic 1 |

### ⚠️ Критические риски Epic 4

- **git2store — высокий риск** — обязательный backup перед операцией
- **Блокировки хранилища** — механизм retry при конфликтах
- **Сложный workflow** — детальное логирование каждого этапа

---

## Epic 5: Quality & Integration

**Цель:** SonarQube и Gitea интеграции на новой архитектуре
**Общая оценка:** 56 часов (1.5 месяца)
**Зависимости:** Epic 1

### Задачи Epic 5

| ID | Задача | Описание | Результат | Оценка | Зависимости |
|----|--------|----------|-----------|--------|-------------|
| **5.1** | **SonarQube Adapter Interface** | Интерфейс SonarQubeClient. Методы: CreateProject, RunAnalysis, GetIssues, GetQualityGate. | Файл: `internal/adapter/sonarqube/interfaces.go`. | 4 ч | Epic 1 |
| **5.2** | **Gitea Adapter Interface** | Role-based интерфейсы: PRReader, CommitReader, FileReader. ISP compliance. | Файл: `internal/adapter/gitea/interfaces.go`. | 4 ч | Epic 1 |
| **5.3** | **nr-sq-scan-branch** | NR-команда сканирования ветки. Фильтрация: main или t######. Проверка изменений в каталогах конфигурации. | Файл: `handlers/sonarqube/scanbranch.go`. | 8 ч | 5.1, 5.2 |
| **5.4** | **nr-sq-scan-pr** | NR-команда сканирования PR. Результат: new_issues, quality_gate_status. | Файл: `handlers/sonarqube/scanpr.go`. | 6 ч | 5.3 |
| **5.5** | **nr-sq-report-branch** | NR-команда отчёта по ветке. Summary: bugs, vulnerabilities, code_smells, coverage. Text: читаемый в CLI. | Файл: `handlers/sonarqube/report.go`. Pain point: "переключение в браузер". | 8 ч | 5.3 |
| **5.6** | **nr-sq-project-update** | NR-команда обновления метаданных проекта в SonarQube. | Файл: `handlers/sonarqube/projectupdate.go`. | 4 ч | 5.1 |
| **5.7** | **nr-test-merge** | NR-команда проверки конфликтов всех открытых PR. Результат: список PR с/без конфликтов. | Файл: `handlers/gitea/testmerge.go`. | 8 ч | 5.2 |
| **5.8** | **nr-action-menu-build** | NR-команда построения динамического меню действий из конфигурации. JSON для UI. | Файл: `handlers/gitea/actionmenu.go`. | 6 ч | 5.2 |
| **5.9** | **Command Summary (FR68)** | Summary с метриками после каждой команды: duration, key_metrics, warnings_count. В metadata.summary. | Расширение OutputWriter. | 8 ч | Epic 1 |

---

## Epic 6: Advanced Observability

**Цель:** Полная диагностика без доступа к production
**Общая оценка:** 72 часа (2 месяца)
**Зависимости:** Epic 1

### Задачи Epic 6

| ID | Задача | Описание | Результат | Оценка | Зависимости |
|----|--------|----------|-----------|--------|-------------|
| **6.1** | **Log File Rotation** | Ротация логов при достижении max_size_mb. Хранение max_files архивов. Библиотека: lumberjack. | Расширение logging config. | 6 ч | Epic 1 |
| **6.2** | **Email Alerting** | Отправка email при критических ошибках. Настройка: smtp_host, from, to. Rate limiting: 1 email / 5 мин на тип. | Файл: `internal/pkg/alerting/email.go`. | 8 ч | Epic 1 |
| **6.3** | **Telegram Alerting** | Отправка в Telegram. Настройка: bot_token, chat_id. Markdown форматирование. | Файл: `internal/pkg/alerting/telegram.go`. | 6 ч | 6.2 |
| **6.4** | **Webhook Alerting** | POST на URL с JSON payload. Retry: 3 попытки, exponential backoff. | Файл: `internal/pkg/alerting/webhook.go`. | 6 ч | 6.2 |
| **6.5** | **Prometheus Metrics** | Метрики: command_duration_seconds, command_success_total, command_error_total. Push to Pushgateway. | Файл: `internal/pkg/metrics/prometheus.go`. | 10 ч | Epic 1 |
| **6.6** | **Alert Rules Configuration** | Правила алертинга по error_code, severity, command. Возможность отключения для команд. | Файл: `internal/pkg/alerting/rules.go`. Config section. | 8 ч | 6.2-6.4 |
| **6.7** | **OpenTelemetry Export** | Отправка трейсов в OTLP бэкенд. Span-ы для ключевых этапов. Async export с буферизацией. | Библиотека: opentelemetry-go. OTLP HTTP. | 12 ч | Epic 1 |
| **6.8** | **Trace Sampling** | Настройка sampling_rate (0.0-1.0). Env override: BR_TRACE_SAMPLE_RATE. TraceIDRatioBased sampler. | Расширение 6.7. Баланс детализации/overhead. | 4 ч | 6.7 |
| **6.9** | **Delve Debugging** | BR_DEBUG=true для запуска с Delve. Удалённое подключение: --headless --listen=:2345. Makefile target: debug-run. | Docker: expose 2345. Linux support. | 12 ч | Epic 1 |

---

## Epic 7: Finalization

**Цель:** Завершение миграции и cleanup deprecated кода
**Общая оценка:** 56 часов (1.5 месяца)
**Зависимости:** Epics 1-6

### Задачи Epic 7

| ID | Задача | Описание | Результат | Оценка | Зависимости |
|----|--------|----------|-----------|--------|-------------|
| **7.1** | **Shadow-run Mode (FR51)** | BR_SHADOW_RUN=true: выполнение обеих версий (legacy + NR), сравнение результатов, diff при различиях. | Файл: `internal/command/shadowrun.go`. Безопасная миграция. | 10 ч | Epics 2-5 |
| **7.2** | **Smoke Tests (FR52)** | CI pipeline: smoke-тесты на тестовом 1C-сервере после merge в main. Тесты: status, dbrestore (dry-run), convert. | Gitea Actions workflow. | 8 ч | Epics 2-5 |
| **7.3** | **Operation Plan Display (FR63)** | BR_VERBOSE=true: показ плана перед выполнением. --plan-only: прервать после плана. | Расширение dry-run: plan → confirm → execute. | 6 ч | 3.6 |
| **7.4** | **Rollback Support (FR64)** | Документация процедуры rollback: изменение BR_COMMAND с nr-xxx на xxx. Время: < 1 мин. | Runbook документ. | 4 ч | Epic 1 |
| **7.5** | **Migration Script (FR65)** | Скрипт migrate-to-nr.sh: замена BR_COMMAND=xxx → BR_COMMAND=nr-xxx в pipeline файлах. Backup + отчёт. | Shell script или Go утилита. | 6 ч | Epics 2-5 |
| **7.6** | **Deprecated Code Audit** | CI job: отчёт о @deprecated коде. Статистика использования из логов. | golangci-lint custom rule или grep. | 4 ч | Epics 2-5 |
| **7.7** | **Deprecated Code Removal (FR49)** | Удаление deprecated после N дней неиспользования. Auto-PR с manual approve. | Только после подтверждения миграции всех pipelines. | 8 ч | 7.6 |
| **7.8** | **ADR Management (FR55-56)** | Шаблон ADR: Title, Status, Context, Decision, Consequences. Автогенерация индекса. | Шаблон: `docs/architecture/adr/template.md`. | 4 ч | — |
| **7.9** | **Data Masking (FR59)** | BR_MASK_DATA=true при restore: маскирование ФИО, email, телефоны, ИНН. Опциональный SQL скрипт. | Файл: `internal/service/datamasking.go`. | 6 ч | 3.2 |
| **7.10** | **YAML Output Format** | BR_OUTPUT_FORMAT=yaml. Структура идентична JSON. gopkg.in/yaml.v3. | Файл: `internal/pkg/output/yaml.go`. | 4 ч | Epic 1 |

---

## Помесячный план

### Месяц 1-2: Epic 1 (Architectural Foundation)

| Неделя | Задачи | Часов |
|--------|--------|-------|
| 1 | 1.1 Command Registry | 8 |
| 2 | 1.3 OutputWriter (начало) | 8 |
| 3 | 1.3 OutputWriter (завершение), 1.4 Logger | 12 |
| 4 | 1.6 Config, 1.2 DeprecatedBridge | 12 |
| 5 | 1.5 Trace ID, 1.7 Wire DI (начало) | 10 |
| 6 | 1.7 Wire DI (завершение) | 8 |
| 7 | 1.8 nr-version | 8 |
| 8 | 1.9 Help | 8 |

**Контрольная точка:** nr-version работает в production pipeline

### Месяц 3-4: Epic 2 (Service Mode)

| Неделя | Задачи | Часов |
|--------|--------|-------|
| 9 | 2.1 RAC Interface, 2.2 RAC Client | 12 |
| 10 | 2.3 nr-service-mode-status, 2.4 Session Info | 12 |
| 11 | 2.5 nr-service-mode-enable | 8 |
| 12 | 2.6 Force Disconnect, 2.7 disable | 12 |
| 13 | 2.8 State-Aware Execution | 8 |

**Контрольная точка:** 3 service-mode NR-команды в 3+ production pipelines

### Месяц 5-6: Epic 3 (Database Operations)

| Неделя | Задачи | Часов |
|--------|--------|-------|
| 14 | 3.1 MSSQL Interface | 4 |
| 15 | 3.2 nr-dbrestore | 12 |
| 16 | 3.3 Progress Bar | 8 |
| 17 | 3.4 nr-dbupdate | 8 |
| 18 | 3.5 nr-create-temp-db, 3.6 Dry-run (начало) | 10 |
| 19 | 3.6 Dry-run (завершение) | 6 |

### Месяц 7-8: Epic 4 (Configuration Sync)

| Неделя | Задачи | Часов |
|--------|--------|-------|
| 20 | 4.1 1C Operations Factory | 10 |
| 21 | 4.2 nr-store2db | 10 |
| 22 | 4.3 nr-storebind, 4.4 nr-create-stores | 12 |
| 23 | 4.5 nr-convert | 8 |
| 24 | 4.6 nr-git2store (начало) | 10 |
| 25 | 4.6 nr-git2store (завершение) | 6 |
| 26 | 4.7 nr-execute-epf | 8 |

### Месяц 9: Epic 5 (Quality & Integration)

| Неделя | Задачи | Часов |
|--------|--------|-------|
| 27 | 5.1 SQ Interface, 5.2 Gitea Interface | 8 |
| 28 | 5.3 nr-sq-scan-branch | 8 |
| 29 | 5.4 nr-sq-scan-pr, 5.5 nr-sq-report-branch | 14 |
| 30 | 5.6-5.9 остальные команды | 22 |

**Контрольная точка:** Все 17 NR-команд реализованы

### Месяц 10-11: Epic 6 (Advanced Observability)

| Неделя | Задачи | Часов |
|--------|--------|-------|
| 31 | 6.1 Log Rotation, 6.2 Email | 14 |
| 32 | 6.3 Telegram, 6.4 Webhook | 12 |
| 33 | 6.5 Prometheus Metrics | 10 |
| 34 | 6.6 Alert Rules, 6.7 OpenTelemetry (начало) | 14 |
| 35 | 6.7 OpenTelemetry (завершение), 6.8 Sampling | 10 |
| 36 | 6.9 Delve Debugging | 12 |

### Месяц 12: Epic 7 (Finalization)

| Неделя | Задачи | Часов |
|--------|--------|-------|
| 37 | 7.1 Shadow-run | 10 |
| 38 | 7.2 Smoke Tests, 7.3 Plan Display | 14 |
| 39 | 7.4-7.6 Rollback, Migration, Audit | 14 |
| 40 | 7.7-7.10 Removal, ADR, Masking, YAML | 18 |

**Итоговая контрольная точка:** Deprecated код удалён, все pipelines на NR-командах

---

## Матрица зависимостей

```
Epic 1 ──────┬──────────────────────────────────────────────────────┐
             │                                                      │
             ▼                                                      │
Epic 2 ──────┬─────────────────────────────────────────┐           │
             │                                          │           │
             ├────────────┬─────────────┬───────────────▼───────────▼
             │            │             │
             ▼            ▼             ▼
          Epic 3       Epic 4       Epic 5
             │            │             │
             └────────────┼─────────────┘
                          │
                          ▼
                       Epic 6
                          │
                          ▼
                       Epic 7
```

---

## Риски и митигация

### Высокий риск

| Риск | Митигация | Задача |
|------|-----------|--------|
| NR ломает production | Shadow-run обязателен, canary deployment | 7.1 |
| DB restore в production | IsProduction check, НИКОГДА restore В prod | 3.2 |
| git2store потеря данных | Обязательный backup, retry при блокировках | 4.6 |

### Средний риск

| Риск | Митигация | Задача |
|------|-----------|--------|
| Wire не компилируется | Unit-тесты providers, `go generate` в CI | 1.7 |
| JSON breaking changes | Golden tests, JSON Schema, API version | 1.3 |
| Config ломает старые файлы | Optional поля, backward compat test | 1.6 |

---

## Метрики успешности

| Метрика | Целевое значение | Измерение |
|---------|------------------|-----------|
| NR-команд в production | 17/17 | CI dashboard |
| Test coverage | ≥ 80% | `go test -cover` |
| Incidents из-за NR | 0 | Incident tracking |
| Время добавления команды | < 4 часов | Time tracking |
| Cyclomatic complexity | < 10 | golangci-lint |

---

## Заключение

Общий объём работ: **~424 часа** (58 задач)

При бюджете **40 часов/месяц** и **1 разработчике** ожидаемый срок: **~11 месяцев**

Критический путь: Epic 1 → Epic 2 → Epic 7 (остальные эпики можно выполнять параллельно после Epic 2)

---

_Документ подготовлен на основе анализа PRD, Architecture и Epics проекта benadis-runner._
