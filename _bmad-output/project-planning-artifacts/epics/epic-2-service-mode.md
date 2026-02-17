# Epic 2: Service Mode Management

**Статус:** 🟠 Legacy существует, NR не начат
**Приоритет:** Высокий (Proof of Concept)
**Риск:** 🟢 Низкий
**Stories:** 0/8 NR (legacy работает)
**FRs:** FR6-9, FR60-62, FR66
**Аудит:** 2026-01-26

---

## 📊 Gap Analysis (Аудит 2026-01-26)

### Статус реализации: 🟠 Legacy существует, NR не начат

| Компонент | План (NR) | Legacy реализация | Статус |
|-----------|-----------|-------------------|--------|
| RAC Adapter Interface | `internal/adapter/onec/rac/interfaces.go` | `internal/servicemode/servicemode.go:14` | 🟠 Legacy |
| RAC Client | `internal/adapter/onec/rac/client.go` | `internal/rac/` | 🟠 Legacy |
| nr-service-mode-status | Command Registry | `main.go:102` (switch-case) | 🟠 Legacy |
| nr-service-mode-enable | Command Registry | `main.go:61` (switch-case) | 🟠 Legacy |
| nr-service-mode-disable | Command Registry | `main.go:82` (switch-case) | 🟠 Legacy |
| Session Info (FR66) | Story 2.4 | ❌ Не реализовано | 🔴 |
| Force Disconnect (FR9) | Story 2.6 | ⚠️ Частично (флаг есть) | 🟡 |
| Idempotency (FR60-62) | Story 2.8 | ❌ Не реализовано | 🔴 |

### Текущее состояние кода

```
LEGACY РЕАЛИЗАЦИЯ:
├── internal/servicemode/servicemode.go     ✅ Manager interface
├── internal/rac/                           ✅ RAC client
├── internal/app/app.go                     ✅ ServiceMode* функции
└── cmd/apk-ci/main.go:61-121       ✅ switch-case

NR АРХИТЕКТУРА (ОЖИДАЕТСЯ):
├── internal/command/handlers/servicemode/  ❌ НЕ СУЩЕСТВУЕТ
├── internal/adapter/onec/rac/interfaces.go ❌ НЕ СУЩЕСТВУЕТ
└── Command Registry integration            ❌ НЕ СУЩЕСТВУЕТ
```

### 🔒 Prerequisite

**Требует Epic 1!** Без Command Registry невозможно создать NR-команды.

### Legacy команды в production

| Команда | Статус | Используется |
|---------|--------|--------------|
| service-mode-enable | ✅ Работает | В CI/CD |
| service-mode-disable | ✅ Работает | В CI/CD |
| service-mode-status | ✅ Работает | В CI/CD |

### Stories Progress

| Story | Название | Статус |
|-------|----------|--------|
| 2.1 | RAC Adapter Interface | 🔴 Ждёт Epic 1 |
| 2.2 | RAC Client Implementation | 🟠 Legacy есть |
| 2.3 | nr-service-mode-status | 🔴 Ждёт Epic 1 |
| 2.4 | Session Info (FR66) | 🔴 Не начат |
| 2.5 | nr-service-mode-enable | 🔴 Ждёт Epic 1 |
| 2.6 | Force Disconnect (FR9) | 🟡 Частично |
| 2.7 | nr-service-mode-disable | 🔴 Ждёт Epic 1 |
| 2.8 | State-Aware Execution | 🔴 Не начат |

---

## Цель

Реализовать управление сервисным режимом на новой архитектуре. Это Proof of Concept — первая реальная команда, которая будет использоваться в production pipeline.

## Ценность

DevOps может управлять доступом к базам через NR-команды. Валидация архитектуры Epic 1 на реальном use case.

---

## Волны выполнения

```
ВОЛНА 1:     2.1 RAC Adapter Interface
                    │
ВОЛНА 2:     2.2 RAC Client Implementation
                    │
ВОЛНА 3:     2.3 nr-service-mode-status ←── 2.4 Session Info (FR66)
                    │
ВОЛНА 4:     2.5 nr-service-mode-enable ←── 2.6 Force Disconnect (FR9)
                    │
ВОЛНА 5:     2.7 nr-service-mode-disable
                    │
ВОЛНА 6:     2.8 Idempotency + State Check (FR60-62)
```

---

## Stories

### Story 2.1: RAC Adapter Interface

**Приоритет:** P0 | **Размер:** S | **Риск:** Low
**Prerequisites:** Epic 1 (Wire DI)

**As a** разработчик,
**I want** иметь абстракцию над RAC клиентом,
**So that** я могу тестировать команды без реального 1C-сервера.

**Acceptance Criteria:**

- [ ] Interface RACClient определён
- [ ] Методы: GetClusterInfo, GetInfobases, GetSessions, SetServiceMode
- [ ] Interface в `internal/adapter/onec/rac/interfaces.go`
- [ ] Можно подставить mock для тестов

**Technical Notes:**
- Файл: `internal/adapter/onec/rac/interfaces.go`
- Ref: Architecture "Role-based interfaces"

---

### Story 2.2: RAC Client Implementation

**Приоритет:** P0 | **Размер:** M | **Риск:** Medium
**Prerequisites:** Story 2.1

**As a** система,
**I want** выполнять RAC команды через subprocess,
**So that** я могу управлять кластером 1C.

**Acceptance Criteria:**

- [ ] RAC executable доступен по пути из конфигурации
- [ ] Timeout настраивается через конфигурацию
- [ ] Ошибки RAC парсятся в структурированный AppError
- [ ] Credentials передаются безопасно (не в command line где возможно)

**Technical Notes:**
- Файл: `internal/adapter/onec/rac/client.go`
- RAC commands: `rac cluster list`, `rac infobase list`, `rac session list`, etc.
- Ref: существующий `internal/servicemode/`

---

### Story 2.3: nr-service-mode-status

**Приоритет:** P0 | **Размер:** M | **Риск:** Low
**Prerequisites:** Story 2.1, 2.2, Epic 1

**As a** DevOps-инженер,
**I want** проверить статус сервисного режима,
**So that** я знаю можно ли работать с базой.

**Acceptance Criteria:**

- [ ] BR_COMMAND=nr-service-mode-status BR_INFOBASE_NAME=MyBase
- [ ] Вывод: enabled/disabled, message, scheduled_jobs_blocked
- [ ] JSON формат содержит все поля
- [ ] Команда зарегистрирована через Registry
- [ ] trace_id присутствует в логах

**Technical Notes:**
- Файл: `internal/command/handlers/servicemode/status.go`
- Domain: `internal/domain/servicemode/`

---

### Story 2.4: Session Info в service-mode-status (FR66)

**Приоритет:** P1 | **Размер:** S | **Риск:** Low
**Prerequisites:** Story 2.3

**As a** DevOps-инженер,
**I want** видеть количество активных сессий и их владельцев,
**So that** я понимаю кого затронет включение сервисного режима.

**Acceptance Criteria:**

- [ ] Вывод содержит: active_sessions_count
- [ ] sessions[] с user_name, host, started_at
- [ ] JSON output включает полный список сессий
- [ ] Text output показывает summary + top-5 сессий

**Technical Notes:**
- Расширение Story 2.3
- Journey Mapping: решает Pain Point "Нет инфо о сессиях"

---

### Story 2.5: nr-service-mode-enable

**Приоритет:** P0 | **Размер:** M | **Риск:** Low
**Prerequisites:** Story 2.3

**As a** DevOps-инженер,
**I want** включить сервисный режим,
**So that** я могу безопасно выполнять операции с базой.

**Acceptance Criteria:**

- [ ] BR_COMMAND=nr-service-mode-enable BR_INFOBASE_NAME=MyBase
- [ ] Сервисный режим включён, регулярные задания заблокированы
- [ ] BR_SERVICE_MODE_MESSAGE — сообщение для пользователей
- [ ] BR_SERVICE_MODE_PERMISSION_CODE — код разрешения
- [ ] Команда идемпотентна: повторный вызов не ошибка (FR62)

**Technical Notes:**
- Файл: `internal/command/handlers/servicemode/enable.go`
- RAC: `rac infobase update --scheduled-jobs-denied=on`

---

### Story 2.6: Force Disconnect Sessions (FR9)

**Приоритет:** P1 | **Размер:** S | **Риск:** Medium
**Prerequisites:** Story 2.5

**As a** DevOps-инженер,
**I want** принудительно завершить сессии пользователей,
**So that** сервисный режим применяется немедленно.

**Acceptance Criteria:**

- [ ] BR_FORCE_DISCONNECT=true → все активные сессии (кроме текущей) завершаются
- [ ] Выводится количество завершённых сессий
- [ ] По умолчанию флаг выключен (безопасное поведение)
- [ ] BR_DISCONNECT_DELAY_SEC — grace period

**Technical Notes:**
- RAC: `rac session terminate`
- Опасная операция — требует явного флага

---

### Story 2.7: nr-service-mode-disable

**Приоритет:** P0 | **Размер:** S | **Риск:** Low
**Prerequisites:** Story 2.5

**As a** DevOps-инженер,
**I want** отключить сервисный режим,
**So that** пользователи могут работать с базой.

**Acceptance Criteria:**

- [ ] BR_COMMAND=nr-service-mode-disable BR_INFOBASE_NAME=MyBase
- [ ] Сервисный режим отключён, регулярные задания разблокированы
- [ ] Команда идемпотентна: повторный вызов не ошибка (FR62)

**Technical Notes:**
- Файл: `internal/command/handlers/servicemode/disable.go`

---

### Story 2.8: State-Aware Execution (FR60-62)

**Приоритет:** P1 | **Размер:** M | **Риск:** Low
**Prerequisites:** Story 2.5, 2.7

**As a** система,
**I want** проверять текущее состояние перед операцией,
**So that** операции идемпотентны и безопасны.

**Acceptance Criteria:**

- [ ] enable когда уже включён → success + "already_enabled": true
- [ ] disable когда уже выключен → success + "already_disabled": true
- [ ] Логируется текущее состояние перед изменением
- [ ] JSON output содержит "state_changed": true/false

**Technical Notes:**
- Паттерн: Check → Act → Verify
- Ref: PRD "Модель выполнения операций"

---

## Risk Assessment

| Риск | Вероятность | Импакт | Митигация |
|------|-------------|--------|-----------|
| RAC недоступен в CI | Средняя | Средний | Mock client для тестов |
| Неправильные credentials | Низкая | Средний | Валидация при старте |
| Сессии не завершаются | Низкая | Низкий | Retry + timeout |

---

## Definition of Done

- [ ] service-mode-* команды используются в 3+ pipelines
- [ ] Все unit-тесты проходят с mock RAC client
- [ ] Integration тест с реальным RAC (опционально)
- [ ] Документация обновлена

---

## Связанные документы

- [Epic Overview](./index.md)
- [Epic 1: Foundation](./epic-1-foundation.md) (prerequisite)
- [FR Coverage](./fr-coverage.md)

---

_Последнее обновление: 2026-01-26_
_Аудит проведён: 2026-01-26 (BMAD Party Mode)_
