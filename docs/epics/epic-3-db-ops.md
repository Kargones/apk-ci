# Epic 3: Database Operations

**Цель:** Реализовать операции с базами данных (restore, update, create) на новой архитектуре с progress reporting.

**FRs:** FR10-13, FR58, FR67

**Ценность:** Restore/update баз с progress bar и dry-run режимом

---

### Story 3.1: MSSQL Adapter Interface

**As a** разработчик,
**I want** иметь абстракцию над MSSQL операциями,
**So that** я могу тестировать без реального SQL Server.

**Acceptance Criteria:**

**Given** interface DatabaseRestorer определён
**When** используется в service layer
**Then** можно подставить mock для тестов

**And** interface содержит: Restore, GetBackupSize, GetDatabaseSize
**And** определён в `internal/adapter/mssql/interfaces.go`

**Prerequisites:** Epic 1

**Technical Notes:**
- Файл: `internal/adapter/mssql/interfaces.go`
- Существующий код: `internal/service/dbrestore.go`

---

### Story 3.2: nr-dbrestore с auto-timeout (FR10-11)

**As a** DevOps-инженер,
**I want** восстановить базу данных из backup,
**So that** я могу обновить тестовое окружение.

**Acceptance Criteria:**

**Given** BR_COMMAND=nr-dbrestore BR_INFOBASE_NAME=MyBase
**When** команда выполняется
**Then** база восстанавливается из backup

**And** таймаут автоматически рассчитывается по размеру backup (если BR_AUTO_TIMEOUT=true)
**And** формула: timeout_min = backup_size_gb * 10 + 5 (базовый)
**And** можно указать явный таймаут (BR_TIMEOUT_MIN)
**And** НИКОГДА не restore В production базу (проверка IsProduction)

**Prerequisites:** Story 3.1, Epic 2 (service mode для блокировки)

**Technical Notes:**
- Файл: `internal/command/handlers/database/restore.go`
- ⚠️ Risk: Проверка DetermineSrcAndDstServers()
- Ref: существующий `internal/app/dbrestore.go`

---

### Story 3.3: Progress Bar для долгих операций (FR67)

**As a** DevOps-инженер,
**I want** видеть прогресс долгих операций,
**So that** я знаю сколько ещё ждать.

**Acceptance Criteria:**

**Given** операция выполняется дольше 30 секунд
**When** включён progress mode (BR_SHOW_PROGRESS=true или tty detected)
**Then** показывается progress bar с оценкой времени

**And** формат: [=====>    ] 45% | ETA: 2m 30s | Restoring...
**And** в non-tty режиме: периодический вывод процентов в лог
**And** JSON output содержит progress events (если streaming)

**Prerequisites:** Story 3.2

**Technical Notes:**
- Файлы: `internal/pkg/progress/progress.go`
- Библиотека: github.com/schollz/progressbar или собственная
- Работает в stderr (не ломает JSON output)

---

### Story 3.4: nr-dbupdate (FR12)

**As a** DevOps-инженер,
**I want** обновить структуру базы данных,
**So that** конфигурация применяется к базе.

**Acceptance Criteria:**

**Given** BR_COMMAND=nr-dbupdate BR_INFOBASE_NAME=MyBase
**When** команда выполняется
**Then** структура базы обновляется по конфигурации

**And** для расширений выполняется дважды (особенность платформы)
**And** сервисный режим включается автоматически если --auto-deps (FR61)
**And** summary показывает количество изменённых объектов

**Prerequisites:** Story 3.2

**Technical Notes:**
- Файл: `internal/command/handlers/database/update.go`
- 1cv8 DESIGNER /UpdateDBCfg

---

### Story 3.5: nr-create-temp-db (FR13)

**As a** тестировщик,
**I want** создать временную базу данных,
**So that** я могу провести изолированное тестирование.

**Acceptance Criteria:**

**Given** BR_COMMAND=nr-create-temp-db BR_EXTENSIONS=ext1,ext2
**When** команда выполняется
**Then** создаётся локальная файловая база с указанными расширениями

**And** путь к базе выводится в результате
**And** база создаётся во временной директории
**And** можно указать TTL для автоудаления (BR_TTL_HOURS)

**Prerequisites:** Epic 1

**Technical Notes:**
- Файл: `internal/command/handlers/database/createtemp.go`
- 1cv8 CREATEINFOBASE

---

### Story 3.6: Dry-run режим (FR58)

**As a** DevOps-инженер,
**I want** проверить что будет выполнено без реальных изменений,
**So that** я могу безопасно протестировать команды.

**Acceptance Criteria:**

**Given** любая команда с BR_DRY_RUN=true
**When** команда выполняется
**Then** выводится план действий БЕЗ выполнения

**And** plan содержит: операции, параметры, ожидаемые изменения
**And** JSON output имеет поле "dry_run": true
**And** exit code = 0 если план валиден

**Prerequisites:** Story 3.2, 3.4

**Technical Notes:**
- Паттерн: BuildPlan() → если dry_run: return plan → иначе: ExecutePlan()
