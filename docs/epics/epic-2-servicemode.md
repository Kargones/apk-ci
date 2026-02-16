# Epic 2: Service Mode Management

**Цель:** Реализовать управление сервисным режимом на новой архитектуре. Это Proof of Concept — первая реальная команда, которая будет использоваться в production pipeline.

**FRs:** FR6-9, FR60-62, FR66

**Ценность:** DevOps может управлять доступом к базам через NR-команды

### Волны выполнения

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

### Story 2.1: RAC Adapter Interface

**As a** разработчик,
**I want** иметь абстракцию над RAC клиентом,
**So that** я могу тестировать команды без реального 1C-сервера.

**Acceptance Criteria:**

**Given** interface RACClient определён
**When** используется в service layer
**Then** можно подставить mock для тестов

**And** interface содержит методы: GetClusterInfo, GetInfobases, GetSessions, SetServiceMode
**And** interface определён в `internal/adapter/onec/rac/interfaces.go`

**Prerequisites:** Epic 1 (Wire DI)

**Technical Notes:**
- Файл: `internal/adapter/onec/rac/interfaces.go`
- Ref: Architecture "Role-based interfaces"

---

### Story 2.2: RAC Client Implementation

**As a** система,
**I want** выполнять RAC команды через subprocess,
**So that** я могу управлять кластером 1C.

**Acceptance Criteria:**

**Given** RAC executable доступен по пути из конфигурации
**When** вызывается метод RACClient
**Then** выполняется соответствующая RAC команда

**And** timeout настраивается через конфигурацию
**And** ошибки RAC парсятся в структурированный AppError
**And** credentials передаются безопасно (не в command line где возможно)

**Prerequisites:** Story 2.1

**Technical Notes:**
- Файл: `internal/adapter/onec/rac/client.go`
- RAC commands: `rac cluster list`, `rac infobase list`, `rac session list`, etc.
- Ref: существующий `internal/servicemode/`

---

### Story 2.3: nr-service-mode-status

**As a** DevOps-инженер,
**I want** проверить статус сервисного режима,
**So that** я знаю можно ли работать с базой.

**Acceptance Criteria:**

**Given** BR_COMMAND=nr-service-mode-status BR_INFOBASE_NAME=MyBase
**When** команда выполняется
**Then** выводится: enabled/disabled, message, scheduled_jobs_blocked

**And** вывод в JSON формате содержит все поля
**And** команда зарегистрирована через Registry
**And** trace_id присутствует в логах

**Prerequisites:** Story 2.1, 2.2, Epic 1

**Technical Notes:**
- Файл: `internal/command/handlers/servicemode/status.go`
- Domain: `internal/domain/servicemode/`

---

### Story 2.4: Session Info в service-mode-status (FR66)

**As a** DevOps-инженер,
**I want** видеть количество активных сессий и их владельцев,
**So that** я понимаю кого затронет включение сервисного режима.

**Acceptance Criteria:**

**Given** BR_COMMAND=nr-service-mode-status
**When** команда выполняется
**Then** вывод содержит: active_sessions_count, sessions[] с user_name, host, started_at

**And** JSON output включает полный список сессий
**And** Text output показывает summary + top-5 сессий

**Prerequisites:** Story 2.3

**Technical Notes:**
- Расширение Story 2.3
- Journey Mapping: решает Pain Point "Нет инфо о сессиях"

---

### Story 2.5: nr-service-mode-enable

**As a** DevOps-инженер,
**I want** включить сервисный режим,
**So that** я могу безопасно выполнять операции с базой.

**Acceptance Criteria:**

**Given** BR_COMMAND=nr-service-mode-enable BR_INFOBASE_NAME=MyBase
**When** команда выполняется
**Then** сервисный режим включён, регулярные задания заблокированы

**And** можно указать сообщение для пользователей (BR_SERVICE_MODE_MESSAGE)
**And** можно указать код разрешения (BR_SERVICE_MODE_PERMISSION_CODE)
**And** команда идемпотентна: повторный вызов не ошибка (FR62)

**Prerequisites:** Story 2.3

**Technical Notes:**
- Файл: `internal/command/handlers/servicemode/enable.go`
- RAC: `rac infobase update --scheduled-jobs-denied=on`

---

### Story 2.6: Force Disconnect Sessions (FR9)

**As a** DevOps-инженер,
**I want** принудительно завершить сессии пользователей,
**So that** сервисный режим применяется немедленно.

**Acceptance Criteria:**

**Given** BR_COMMAND=nr-service-mode-enable с флагом BR_FORCE_DISCONNECT=true
**When** команда выполняется
**Then** все активные сессии (кроме текущей) завершаются

**And** выводится количество завершённых сессий
**And** по умолчанию флаг выключен (безопасное поведение)
**And** можно указать grace period (BR_DISCONNECT_DELAY_SEC)

**Prerequisites:** Story 2.5

**Technical Notes:**
- RAC: `rac session terminate`
- Опасная операция — требует явного флага

---

### Story 2.7: nr-service-mode-disable

**As a** DevOps-инженер,
**I want** отключить сервисный режим,
**So that** пользователи могут работать с базой.

**Acceptance Criteria:**

**Given** BR_COMMAND=nr-service-mode-disable BR_INFOBASE_NAME=MyBase
**When** команда выполняется
**Then** сервисный режим отключён, регулярные задания разблокированы

**And** команда идемпотентна: повторный вызов не ошибка (FR62)

**Prerequisites:** Story 2.5

**Technical Notes:**
- Файл: `internal/command/handlers/servicemode/disable.go`

---

### Story 2.8: State-Aware Execution (FR60-62)

**As a** система,
**I want** проверять текущее состояние перед операцией,
**So that** операции идемпотентны и безопасны.

**Acceptance Criteria:**

**Given** команда enable вызывается когда режим уже включён
**When** команда выполняется
**Then** возвращается success с флагом "already_enabled": true

**Given** команда disable вызывается когда режим уже выключен
**When** команда выполняется
**Then** возвращается success с флагом "already_disabled": true

**And** логируется текущее состояние перед изменением
**And** JSON output содержит поле "state_changed": true/false

**Prerequisites:** Story 2.5, 2.7

**Technical Notes:**
- Ref: PRD "Модель выполнения операций"
- Паттерн: Check → Act → Verify
