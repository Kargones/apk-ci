# Epic 3: Database Operations

**Статус:** 🟠 Legacy существует, NR не начат
**Приоритет:** Высокий
**Риск:** 🔴 Высокий
**Stories:** 0/6 NR (legacy работает)
**FRs:** FR10-13, FR58, FR67
**Аудит:** 2026-01-26

---

## 📊 Gap Analysis (Аудит 2026-01-26)

### Статус реализации: 🟠 Legacy существует, NR не начат

| Компонент | План (NR) | Legacy реализация | Статус |
|-----------|-----------|-------------------|--------|
| MSSQL Adapter Interface | `internal/adapter/mssql/interfaces.go` | ❌ Нет интерфейса | 🔴 |
| nr-dbrestore | Command Registry | `main.go:133` (switch-case) | 🟠 Legacy |
| nr-dbupdate | Command Registry | `main.go:122` (switch-case) | 🟠 Legacy |
| nr-create-temp-db | Command Registry | `main.go:164` (switch-case) | 🟠 Legacy |
| Progress Bar (FR67) | `internal/pkg/progress/` | ❌ Не реализовано | 🔴 |
| Dry-run режим (FR58) | Story 3.6 | ❌ Не реализовано | 🔴 |
| Auto-timeout (FR11) | Story 3.2 | ✅ Реализовано | ✅ |

### Текущее состояние кода

```
LEGACY РЕАЛИЗАЦИЯ:
├── internal/entity/dbrestore/dbrestore.go  ✅ DbRestore logic
├── internal/app/app.go                     ✅ DbRestore*, DbUpdate*
├── internal/service/                       ✅ Сервисный слой
└── cmd/apk-ci/main.go              ✅ switch-case

NR АРХИТЕКТУРА (ОЖИДАЕТСЯ):
├── internal/command/handlers/database/     ❌ НЕ СУЩЕСТВУЕТ
├── internal/adapter/mssql/interfaces.go    ❌ НЕ СУЩЕСТВУЕТ
└── internal/pkg/progress/                  ❌ НЕ СУЩЕСТВУЕТ
```

### 🔒 Prerequisite

**Требует Epic 1 + Epic 2!**
- Epic 1: Command Registry
- Epic 2: Service Mode (для блокировки базы при restore)

### Legacy команды в production

| Команда | Статус | Auto-timeout | Progress |
|---------|--------|--------------|----------|
| dbrestore | ✅ Работает | ✅ | ❌ |
| dbupdate | ✅ Работает | N/A | ❌ |
| create-temp-db | ✅ Работает | N/A | ❌ |

### Stories Progress

| Story | Название | Статус |
|-------|----------|--------|
| 3.1 | MSSQL Adapter Interface | 🔴 Ждёт Epic 1 |
| 3.2 | nr-dbrestore с auto-timeout | 🟠 Auto-timeout есть |
| 3.3 | Progress Bar (FR67) | 🔴 Не начат |
| 3.4 | nr-dbupdate | 🟠 Legacy есть |
| 3.5 | nr-create-temp-db | 🟠 Legacy есть |
| 3.6 | Dry-run режим (FR58) | 🔴 Не начат |

---

## Цель

Реализовать операции с базами данных (restore, update, create) на новой архитектуре с progress reporting.

## Ценность

Restore/update баз с progress bar и dry-run режимом. Решение Pain Point "Нет прогресса долгих операций".

---

## Критические ограничения

⚠️ **НИКОГДА не restore В production базу!**

Проверка `IsProduction` обязательна перед любой деструктивной операцией.

---

## Stories

### Story 3.1: MSSQL Adapter Interface

**Приоритет:** P0 | **Размер:** S | **Риск:** Low
**Prerequisites:** Epic 1

**As a** разработчик,
**I want** иметь абстракцию над MSSQL операциями,
**So that** я могу тестировать без реального SQL Server.

**Acceptance Criteria:**

- [ ] Interface DatabaseRestorer определён
- [ ] Методы: Restore, GetBackupSize, GetDatabaseSize
- [ ] Interface в `internal/adapter/mssql/interfaces.go`
- [ ] Можно подставить mock для тестов

**Technical Notes:**
- Файл: `internal/adapter/mssql/interfaces.go`
- Существующий код: `internal/service/dbrestore.go`

---

### Story 3.2: nr-dbrestore с auto-timeout (FR10-11)

**Приоритет:** P0 | **Размер:** L | **Риск:** High
**Prerequisites:** Story 3.1, Epic 2 (service mode для блокировки)

**As a** DevOps-инженер,
**I want** восстановить базу данных из backup,
**So that** я могу обновить тестовое окружение.

**Acceptance Criteria:**

- [ ] BR_COMMAND=nr-dbrestore BR_INFOBASE_NAME=MyBase
- [ ] База восстанавливается из backup
- [ ] BR_AUTO_TIMEOUT=true → timeout = backup_size_gb * 10 + 5 минут
- [ ] BR_TIMEOUT_MIN — явный таймаут
- [ ] **ПРОВЕРКА IsProduction** — НИКОГДА restore В production!
- [ ] DetermineSrcAndDstServers() корректно определяет серверы

**Technical Notes:**
- Файл: `internal/command/handlers/database/restore.go`
- ⚠️ Risk: Проверка DetermineSrcAndDstServers()
- Ref: существующий `internal/app/dbrestore.go`

---

### Story 3.3: Progress Bar для долгих операций (FR67)

**Приоритет:** P1 | **Размер:** M | **Риск:** Low
**Prerequisites:** Story 3.2

**As a** DevOps-инженер,
**I want** видеть прогресс долгих операций,
**So that** я знаю сколько ещё ждать.

**Acceptance Criteria:**

- [ ] Операция > 30 сек + BR_SHOW_PROGRESS=true (или tty detected)
- [ ] Формат: `[=====>    ] 45% | ETA: 2m 30s | Restoring...`
- [ ] Non-tty: периодический вывод процентов в лог
- [ ] JSON output содержит progress events (если streaming)
- [ ] Progress в stderr (не ломает JSON output)

**Technical Notes:**
- Файл: `internal/pkg/progress/progress.go`
- Библиотека: github.com/schollz/progressbar или собственная
- Journey Mapping: решает Pain Point "Нет прогресса"

---

### Story 3.4: nr-dbupdate (FR12)

**Приоритет:** P0 | **Размер:** M | **Риск:** Medium
**Prerequisites:** Story 3.2

**As a** DevOps-инженер,
**I want** обновить структуру базы данных,
**So that** конфигурация применяется к базе.

**Acceptance Criteria:**

- [ ] BR_COMMAND=nr-dbupdate BR_INFOBASE_NAME=MyBase
- [ ] Структура базы обновляется по конфигурации
- [ ] Для расширений выполняется дважды (особенность платформы)
- [ ] --auto-deps → сервисный режим включается автоматически (FR61)
- [ ] Summary показывает количество изменённых объектов

**Technical Notes:**
- Файл: `internal/command/handlers/database/update.go`
- 1cv8 DESIGNER /UpdateDBCfg

---

### Story 3.5: nr-create-temp-db (FR13)

**Приоритет:** P1 | **Размер:** M | **Риск:** Low
**Prerequisites:** Epic 1

**As a** тестировщик,
**I want** создать временную базу данных,
**So that** я могу провести изолированное тестирование.

**Acceptance Criteria:**

- [ ] BR_COMMAND=nr-create-temp-db BR_EXTENSIONS=ext1,ext2
- [ ] Создаётся локальная файловая база с расширениями
- [ ] Путь к базе выводится в результате
- [ ] BR_TTL_HOURS — TTL для автоудаления

**Technical Notes:**
- Файл: `internal/command/handlers/database/createtemp.go`
- 1cv8 CREATEINFOBASE
- Journey Mapping: решает Pain Point "Нет auto-cleanup"

---

### Story 3.6: Dry-run режим (FR58)

**Приоритет:** P1 | **Размер:** M | **Риск:** Low
**Prerequisites:** Story 3.2, 3.4

**As a** DevOps-инженер,
**I want** проверить что будет выполнено без реальных изменений,
**So that** я могу безопасно протестировать команды.

**Acceptance Criteria:**

- [ ] BR_DRY_RUN=true → план действий БЕЗ выполнения
- [ ] Plan содержит: операции, параметры, ожидаемые изменения
- [ ] JSON output имеет "dry_run": true
- [ ] exit code = 0 если план валиден

**Technical Notes:**
- Паттерн: BuildPlan() → если dry_run: return plan → иначе: ExecutePlan()

---

## Risk Assessment

| ID | Риск | Вероятность | Импакт | Митигация |
|----|------|-------------|--------|-----------|
| E3-R1 | Restore в production | Низкая | КРИТИЧЕСКИЙ | Проверка IsProduction, WHITELIST разрешённых |
| E3-R2 | Timeout некорректен | Средняя | Средний | Auto-timeout по размеру, manual override |
| E3-R3 | Потеря данных при restore | Низкая | Высокий | Проверка target server, dry-run |

---

## Definition of Done

- [ ] dbrestore/dbupdate работают с progress bar
- [ ] Dry-run режим работает для всех команд
- [ ] Проверка IsProduction покрыта тестами
- [ ] Integration тест с реальным MSSQL (опционально)

---

## Связанные документы

- [Epic Overview](./index.md)
- [Epic 2: Service Mode](./epic-2-service-mode.md) (для блокировки базы)
- [FR Coverage](./fr-coverage.md)

---

_Последнее обновление: 2026-01-26_
_Аудит проведён: 2026-01-26 (BMAD Party Mode)_
