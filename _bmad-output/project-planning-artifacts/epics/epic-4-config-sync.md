# Epic 4: Configuration Sync

**Статус:** 🟠 Legacy существует, NR не начат
**Приоритет:** Высокий
**Риск:** 🔴 Высокий
**Stories:** 0/7 NR (legacy работает)
**FRs:** FR14-21
**Аудит:** 2026-01-26

---

## 📊 Gap Analysis (Аудит 2026-01-26)

### Статус реализации: 🟠 Legacy существует, NR не начат

| Компонент | План (NR) | Legacy реализация | Статус |
|-----------|-----------|-------------------|--------|
| 1C Operations Factory | `internal/adapter/onec/factory.go` | ❌ Нет фабрики | 🔴 |
| nr-store2db | Command Registry | `main.go:31` (switch-case) | 🟠 Legacy |
| nr-storebind | Command Registry | `main.go:154` (switch-case) | 🟠 Legacy |
| nr-create-stores | Command Registry | `main.go:174` (switch-case) | 🟠 Legacy |
| nr-convert | Command Registry | `main.go:40` (switch-case) | 🟠 Legacy |
| nr-git2store | Command Registry | `main.go:50` (switch-case) | 🟠 Legacy |
| nr-execute-epf | Command Registry | `main.go:184` (switch-case) | 🟠 Legacy |
| Switchable implementations | config.implementations | ❌ Не реализовано | 🔴 |

### Текущее состояние кода

```
LEGACY РЕАЛИЗАЦИЯ:
├── internal/entity/one/convert/convert.go  ✅ Конвертация
├── internal/entity/one/store/              ✅ Store операции
├── internal/entity/one/designer/           ✅ Designer операции
├── internal/entity/one/edt/edt.go          ✅ EDT операции
├── internal/entity/one/enterprise/         ✅ Enterprise операции
├── internal/app/app.go                     ✅ Оркестрация
└── cmd/benadis-runner/main.go              ✅ switch-case

NR АРХИТЕКТУРА (ОЖИДАЕТСЯ):
├── internal/command/handlers/store/        ❌ НЕ СУЩЕСТВУЕТ
├── internal/command/handlers/convert/      ❌ НЕ СУЩЕСТВУЕТ
├── internal/adapter/onec/factory.go        ❌ НЕ СУЩЕСТВУЕТ
└── Switchable 1cv8/ibcmd/native            ❌ НЕ СУЩЕСТВУЕТ
```

### 🔒 Prerequisite

**Требует Epic 1 + Epic 2!**
- Epic 1: Command Registry, Config Extensions (Story 1.6)
- Epic 2: Service Mode (для git2store workflow)

### Legacy команды в production

| Команда | Статус | Сложность |
|---------|--------|-----------|
| store2db | ✅ Работает | M |
| storebind | ✅ Работает | S |
| create-stores | ✅ Работает | M |
| convert | ✅ Работает | M |
| git2store | ✅ Работает | XL |
| execute-epf | ✅ Работает | S |

### Stories Progress

| Story | Название | Статус |
|-------|----------|--------|
| 4.1 | 1C Operations Factory | 🔴 Ждёт Epic 1 |
| 4.2 | nr-store2db | 🟠 Legacy есть |
| 4.3 | nr-storebind | 🟠 Legacy есть |
| 4.4 | nr-create-stores | 🟠 Legacy есть |
| 4.5 | nr-convert | 🟠 Legacy есть |
| 4.6 | nr-git2store | 🟠 Legacy есть |
| 4.7 | nr-execute-epf | 🟠 Legacy есть |

---

## Цель

Реализовать синхронизацию конфигурации (EDT↔Store↔DB) на новой архитектуре с детальным reporting.

## Ценность

Полный цикл EDT→Git→Store→DB с прозрачностью каждого этапа. Решение Pain Point "Чёрный ящик".

---

## Критические ограничения

⚠️ **Обязательный backup перед операциями с хранилищем!**

Блокировки хранилища требуют retry с backoff.

---

## Stories

### Story 4.1: 1C Operations Factory (FR18)

**Приоритет:** P0 | **Размер:** M | **Риск:** Low
**Prerequisites:** Epic 1 (Story 1.6, 1.7)

**As a** система,
**I want** выбирать реализацию операций через конфигурацию,
**So that** можно переключаться между 1cv8/ibcmd/native.

**Acceptance Criteria:**

- [ ] config.implementations.config_export = "ibcmd" → ibcmd реализация
- [ ] Factory регистрируется как Wire provider
- [ ] Поддерживаемые операции: config_export, db_create

**Technical Notes:**
- Файл: `internal/adapter/onec/factory.go`
- Ref: Architecture "Switchable Implementation Strategy"

---

### Story 4.2: nr-store2db (FR14)

**Приоритет:** P0 | **Размер:** M | **Риск:** Medium
**Prerequisites:** Story 4.1, Epic 2

**As a** 1C-разработчик,
**I want** загрузить конфигурацию из хранилища в базу,
**So that** база синхронизирована с хранилищем.

**Acceptance Criteria:**

- [ ] BR_COMMAND=nr-store2db BR_INFOBASE_NAME=MyBase
- [ ] Конфигурация загружается из хранилища
- [ ] BR_STORE_VERSION — версия (или latest)
- [ ] Progress: connecting → loading → applying

**Technical Notes:**
- Файл: `internal/command/handlers/store/store2db.go`
- 1cv8 DESIGNER /ConfigurationRepositoryUpdateCfg

---

### Story 4.3: nr-storebind (FR15)

**Приоритет:** P0 | **Размер:** S | **Риск:** Low
**Prerequisites:** Epic 1

**As a** 1C-разработчик,
**I want** привязать хранилище к базе данных,
**So that** могу работать с версионированием конфигурации.

**Acceptance Criteria:**

- [ ] BR_COMMAND=nr-storebind BR_INFOBASE_NAME=MyBase BR_STORE_PATH=//server/store
- [ ] База привязывается к хранилищу
- [ ] Credentials из secret.yaml

**Technical Notes:**
- Файл: `internal/command/handlers/store/bind.go`
- 1cv8 DESIGNER /ConfigurationRepositoryBindCfg

---

### Story 4.4: nr-create-stores (FR17)

**Приоритет:** P1 | **Размер:** M | **Риск:** Low
**Prerequisites:** Epic 1

**As a** 1C-разработчик,
**I want** инициализировать хранилища для проекта,
**So that** могу начать версионирование новой конфигурации.

**Acceptance Criteria:**

- [ ] BR_COMMAND=nr-create-stores
- [ ] Создаются хранилища для основной конфигурации и расширений
- [ ] Список расширений из project.yaml
- [ ] Summary показывает созданные хранилища

**Technical Notes:**
- Файл: `internal/command/handlers/store/create.go`

---

### Story 4.5: nr-convert (FR19-20)

**Приоритет:** P0 | **Размер:** M | **Риск:** Medium
**Prerequisites:** Story 4.1

**As a** 1C-разработчик,
**I want** конвертировать между форматами EDT и XML,
**So that** могу работать с разными инструментами.

**Acceptance Criteria:**

- [ ] BR_COMMAND=nr-convert BR_SOURCE=/path/edt BR_TARGET=/path/xml BR_DIRECTION=edt2xml
- [ ] Направление: edt2xml или xml2edt
- [ ] Инструмент выбирается через config (1cv8/1cedtcli)

**Technical Notes:**
- Файл: `internal/command/handlers/convert/convert.go`
- 1cedtcli для EDT операций

---

### Story 4.6: nr-git2store (FR16)

**Приоритет:** P0 | **Размер:** XL | **Риск:** High
**Prerequisites:** Story 4.2, 4.3, 4.5

**As a** 1C-разработчик,
**I want** синхронизировать EDT из Git в хранилище 1C,
**So that** изменения из IDE попадают в хранилище автоматически.

**Acceptance Criteria:**

- [ ] BR_COMMAND=nr-git2store
- [ ] Workflow: clone → checkout EDT → convert → checkout XML → init DB → apply → commit to store
- [ ] Каждый этап логируется с progress
- [ ] При ошибке — rollback и детальный отчёт
- [ ] Backup перед операцией (обязательно!)

**Technical Notes:**
- Файл: `internal/command/handlers/store/git2store.go`
- Самый сложный workflow — требует orchestration
- Journey Mapping: решает Pain Point "чёрный ящик"
- ⚠️ Risk: Высокий — обязательный backup

---

### Story 4.7: nr-execute-epf (FR21)

**Приоритет:** P1 | **Размер:** S | **Риск:** Low
**Prerequisites:** Epic 1

**As a** 1C-разработчик,
**I want** выполнить внешнюю обработку,
**So that** могу автоматизировать задачи в 1C.

**Acceptance Criteria:**

- [ ] BR_COMMAND=nr-execute-epf BR_EPF_PATH=/path/to/file.epf BR_INFOBASE_NAME=MyBase
- [ ] Обработка выполняется в 1C Enterprise режиме
- [ ] BR_EPF_PARAMS — параметры
- [ ] Timeout настраивается

**Technical Notes:**
- Файл: `internal/command/handlers/convert/executeepf.go`
- 1cv8 ENTERPRISE /Execute

---

## Risk Assessment

| ID | Риск | Вероятность | Импакт | Митигация |
|----|------|-------------|--------|-----------|
| E4-R1 | Блокировки хранилища | Высокая | Средний | Retry с exponential backoff |
| E4-R2 | Потеря данных при git2store | Низкая | КРИТИЧЕСКИЙ | Обязательный backup, dry-run |
| E4-R3 | EDT/XML несовместимость | Средняя | Средний | Валидация после конвертации |

---

## Definition of Done

- [ ] git2store полностью на новой архитектуре
- [ ] Каждый этап git2store логируется с progress
- [ ] Backup создаётся автоматически
- [ ] Retry при блокировках работает

---

## Связанные документы

- [Epic Overview](./index.md)
- [Epic 1: Foundation](./epic-1-foundation.md)
- [Epic 2: Service Mode](./epic-2-service-mode.md)
- [FR Coverage](./fr-coverage.md)

---

_Последнее обновление: 2026-01-26_
_Аудит проведён: 2026-01-26 (BMAD Party Mode)_
