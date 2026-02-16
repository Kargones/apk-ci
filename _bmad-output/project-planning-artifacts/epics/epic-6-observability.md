# Epic 6: Advanced Observability

**Статус:** 🔴 Не начат
**Приоритет:** Средний
**Риск:** 🟡 Средний
**Stories:** 0/9 (0%)
**FRs:** FR36-40, FR44-46, FR53-54, FR57
**Аудит:** 2026-01-26

---

## 📊 Gap Analysis (Аудит 2026-01-26)

### Статус реализации: 🔴 НЕ НАЧАТ

| Компонент | План | Реализация | Статус |
|-----------|------|------------|--------|
| Log File Rotation | `lumberjack` integration | ❌ Не существует | 🔴 |
| Email Alerting | `internal/pkg/alerting/email.go` | ❌ Не существует | 🔴 |
| Telegram Alerting | `internal/pkg/alerting/telegram.go` | ❌ Не существует | 🔴 |
| Webhook Alerting | `internal/pkg/alerting/webhook.go` | ❌ Не существует | 🔴 |
| Prometheus Metrics | `internal/pkg/metrics/prometheus.go` | ❌ Не существует | 🔴 |
| Alert Rules Config | `internal/pkg/alerting/rules.go` | ❌ Не существует | 🔴 |
| OpenTelemetry Export | `go.opentelemetry.io/otel` | ❌ Не существует | 🔴 |
| Trace Sampling | TraceIDRatioBased sampler | ❌ Не существует | 🔴 |
| Delve Debugging | Makefile target | ❌ Не существует | 🔴 |

### Текущее состояние кода

```
ТЕКУЩЕЕ ЛОГИРОВАНИЕ:
├── log/slog используется напрямую          ✅ Базовое
├── Логи в stderr                           ✅ Работает
└── JSON формат                             ⚠️ Частично

ОЖИДАЕТСЯ:
├── internal/pkg/alerting/                  ❌ НЕ СУЩЕСТВУЕТ
├── internal/pkg/metrics/                   ❌ НЕ СУЩЕСТВУЕТ
├── internal/pkg/tracing/ (advanced)        ❌ НЕ СУЩЕСТВУЕТ
├── OpenTelemetry integration               ❌ НЕ СУЩЕСТВУЕТ
└── Delve debug mode                        ❌ НЕ СУЩЕСТВУЕТ
```

### 🔒 Prerequisite

**Требует Epic 1 (базовый tracing) + Epics 2-5!**

Epic 6 расширяет observability после того, как NR-команды созданы.

### Stories Progress

| Story | Название | Статус |
|-------|----------|--------|
| 6.1 | Log File Rotation | 🔴 Не начат |
| 6.2 | Email Alerting | 🔴 Не начат |
| 6.3 | Telegram Alerting | 🔴 Не начат |
| 6.4 | Webhook Alerting | 🔴 Не начат |
| 6.5 | Prometheus Metrics | 🔴 Не начат |
| 6.6 | Alert Rules Config | 🔴 Не начат |
| 6.7 | OpenTelemetry Export | 🔴 Не начат |
| 6.8 | Trace Sampling | 🔴 Не начат |
| 6.9 | Delve Debugging | 🔴 Не начат |

### Зависимости

```
Epic 1 (базовый tracing)
    │
    ├── Epic 2-5 (NR-команды)
    │       │
    │       └── Epic 6 (Advanced Observability)
    │               │
    │               └── Epic 7 (Finalization)
```

---

## Цель

Реализовать продвинутый observability: алертинг, Prometheus метрики, Delve debugging.

## Ценность

Полная диагностика без доступа к production серверам.

---

## Stories

### Story 6.1: Log File Rotation (FR32)

**Приоритет:** P1 | **Размер:** S | **Риск:** Low
**Prerequisites:** Epic 1 (Story 1.4)

**As a** DevOps-инженер,
**I want** чтобы логи ротировались автоматически,
**So that** диск не переполняется.

**Acceptance Criteria:**

- [ ] logging.file настроен → файл ротируется при max_size_mb
- [ ] Старый файл архивируется
- [ ] Хранится max_files архивов
- [ ] Настройка: max_size_mb, max_files в config

**Technical Notes:**
- Библиотека: lumberjack или natefinch/lumberjack

---

### Story 6.2: Email Alerting (FR36)

**Приоритет:** P1 | **Размер:** M | **Риск:** Low
**Prerequisites:** Epic 1

**As a** DevOps-инженер,
**I want** получать алерты на email,
**So that** я знаю о проблемах.

**Acceptance Criteria:**

- [ ] alerting.channels содержит email → отправляется при критической ошибке
- [ ] Настройка: smtp_host, from, to, subject_template
- [ ] Rate limiting: не более 1 email в 5 минут на один тип ошибки

**Technical Notes:**
- Файл: `internal/pkg/alerting/email.go`

---

### Story 6.3: Telegram Alerting (FR37)

**Приоритет:** P1 | **Размер:** S | **Риск:** Low
**Prerequisites:** Story 6.2

**As a** DevOps-инженер,
**I want** получать алерты в Telegram,
**So that** я сразу вижу уведомления.

**Acceptance Criteria:**

- [ ] alerting.channels содержит telegram → сообщение в Telegram
- [ ] Настройка: bot_token, chat_id
- [ ] Форматирование: markdown с деталями ошибки

**Technical Notes:**
- Файл: `internal/pkg/alerting/telegram.go`

---

### Story 6.4: Webhook Alerting (FR38)

**Приоритет:** P1 | **Размер:** S | **Риск:** Low
**Prerequisites:** Story 6.2

**As a** DevOps-инженер,
**I want** отправлять алерты через webhook,
**So that** могу интегрировать с любой системой.

**Acceptance Criteria:**

- [ ] alerting.channels содержит webhook → POST на URL
- [ ] Payload: JSON с деталями ошибки
- [ ] Retry: 3 попытки с exponential backoff

**Technical Notes:**
- Файл: `internal/pkg/alerting/webhook.go`

---

### Story 6.5: Prometheus Metrics (FR39, FR57)

**Приоритет:** P1 | **Размер:** M | **Риск:** Low
**Prerequisites:** Epic 1

**As a** DevOps-инженер,
**I want** экспортировать метрики в Prometheus формате,
**So that** могу строить дашборды в Grafana.

**Acceptance Criteria:**

- [ ] BR_METRICS_ENABLED=true → метрики записываются
- [ ] Метрики: command_duration_seconds, command_success_total, command_error_total
- [ ] Labels: command, infobase, status
- [ ] Push to Pushgateway (CLI не держит HTTP сервер)

**Technical Notes:**
- Файл: `internal/pkg/metrics/prometheus.go`

---

### Story 6.6: Alert Rules Configuration (FR40)

**Приоритет:** P2 | **Размер:** M | **Риск:** Low
**Prerequisites:** Story 6.2-6.4

**As a** DevOps-инженер,
**I want** настраивать правила алертинга,
**So that** могу контролировать когда срабатывают алерты.

**Acceptance Criteria:**

- [ ] alerting.rules в конфигурации
- [ ] Правила: по error_code, severity, command
- [ ] Можно отключить алерты для определённых команд

**Technical Notes:**
- Файл: `internal/pkg/alerting/rules.go`

---

### Story 6.7: OpenTelemetry Export (FR41, FR43)

**Приоритет:** P1 | **Размер:** L | **Риск:** Medium
**Prerequisites:** Epic 1 (Story 1.5)

**As a** DevOps-инженер,
**I want** отправлять трейсы в OTLP бэкенд,
**So that** могу анализировать распределённые операции.

**Acceptance Criteria:**

- [ ] tracing.enabled=true tracing.endpoint=http://jaeger:4318
- [ ] Трейсы отправляются в бэкенд
- [ ] Span-ы для ключевых этапов операции
- [ ] Async export с буферизацией (FR54)

**Technical Notes:**
- Библиотека: go.opentelemetry.io/otel
- OTLP HTTP exporter

---

### Story 6.8: Trace Sampling (FR53)

**Приоритет:** P2 | **Размер:** S | **Риск:** Low
**Prerequisites:** Story 6.7

**As a** DevOps-инженер,
**I want** настраивать sampling rate для трейсов,
**So that** могу балансировать детализацию и overhead.

**Acceptance Criteria:**

- [ ] tracing.sampling_rate=0.1 → только 10% трейсов
- [ ] sampling_rate: 0.0 (none) - 1.0 (all)
- [ ] BR_TRACE_SAMPLE_RATE переопределяет config

**Technical Notes:**
- OpenTelemetry TraceIDRatioBased sampler

---

### Story 6.9: Delve Debugging (FR44-46)

**Приоритет:** P2 | **Размер:** M | **Риск:** Low
**Prerequisites:** Epic 1

**As a** разработчик,
**I want** запускать приложение в режиме отладки,
**So that** могу диагностировать сложные проблемы.

**Acceptance Criteria:**

- [ ] BR_DEBUG=true или специальный build
- [ ] Delve слушает на указанном порту
- [ ] Удалённое подключение: --headless --listen=:2345
- [ ] Работает на Linux и в Docker
- [ ] Makefile target: debug-run

**Technical Notes:**
- Delve: github.com/go-delve/delve
- Docker: expose port 2345

---

## Risk Assessment

| Риск | Вероятность | Импакт | Митигация |
|------|-------------|--------|-----------|
| Overhead от трейсинга | Средняя | Средний | Sampling, async export |
| Email/Telegram недоступны | Низкая | Низкий | Fallback каналы, retry |
| Pushgateway недоступен | Низкая | Низкий | Локальный файл, retry |

---

## Definition of Done

- [ ] Логи/трейсы/алерты работают в production
- [ ] Prometheus метрики экспортируются
- [ ] Delve debugging работает в Docker

---

## Связанные документы

- [Epic Overview](./index.md)
- [Epic 1: Foundation](./epic-1-foundation.md)
- [FR Coverage](./fr-coverage.md)

---

_Последнее обновление: 2026-01-26_
_Аудит проведён: 2026-01-26 (BMAD Party Mode)_
