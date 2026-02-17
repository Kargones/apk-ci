# Epic 5: Quality & Integration

**Статус:** 🟠 Legacy существует, NR не начат
**Приоритет:** Средний
**Риск:** 🟡 Средний
**Stories:** 0/9 NR (legacy работает)
**FRs:** FR22-28, FR68
**Аудит:** 2026-01-26

---

## 📊 Gap Analysis (Аудит 2026-01-26)

### Статус реализации: 🟠 Legacy существует, NR не начат

| Компонент | План (NR) | Legacy реализация | Статус |
|-----------|-----------|-------------------|--------|
| SonarQube Adapter | `internal/adapter/sonarqube/` | `internal/entity/sonarqube/` | 🟠 Legacy |
| Gitea Adapter | `internal/adapter/gitea/` | `internal/entity/gitea/` | 🟠 Legacy |
| nr-sq-scan-branch | Command Registry | `main.go:194` (switch-case) | 🟠 Legacy |
| nr-sq-scan-pr | Command Registry | `main.go:204` (switch-case) | 🟠 Legacy |
| nr-sq-report-branch | Command Registry | `main.go:224` (switch-case) | 🟠 Legacy |
| nr-sq-project-update | Command Registry | `main.go:214` (switch-case) | 🟠 Legacy |
| nr-test-merge | Command Registry | `main.go:234` (switch-case) | 🟠 Legacy |
| nr-action-menu-build | Command Registry | `main.go:144` (switch-case) | 🟠 Legacy |
| Command Summary (FR68) | OutputWriter extension | ❌ Не реализовано | 🔴 |

### Текущее состояние кода

```
LEGACY РЕАЛИЗАЦИЯ:
├── internal/entity/sonarqube/              ✅ SonarQube client
│   ├── service.go                          ✅ Service layer
│   ├── scanner.go                          ✅ Scan logic
│   └── branch_scanner.go                   ✅ Branch scanning
├── internal/service/sonarqube/             ✅ Command handlers
├── internal/entity/gitea/                  ✅ Gitea API client
├── internal/app/app.go                     ✅ SQ* функции
└── cmd/apk-ci/main.go              ✅ switch-case

NR АРХИТЕКТУРА (ОЖИДАЕТСЯ):
├── internal/command/handlers/sonarqube/    ❌ НЕ СУЩЕСТВУЕТ
├── internal/command/handlers/gitea/        ❌ НЕ СУЩЕСТВУЕТ
├── internal/adapter/sonarqube/interfaces.go ❌ НЕ СУЩЕСТВУЕТ
└── internal/adapter/gitea/interfaces.go    ❌ НЕ СУЩЕСТВУЕТ
```

### 🔒 Prerequisite

**Требует Epic 1!** Без Command Registry невозможно создать NR-команды.

### Legacy команды в production

| Команда | Статус | Тестовое покрытие |
|---------|--------|-------------------|
| sq-scan-branch | ✅ Работает | Есть тесты |
| sq-scan-pr | ✅ Работает | Есть тесты |
| sq-report-branch | ✅ Работает | Есть тесты |
| sq-project-update | ✅ Работает | Есть тесты |
| test-merge | ✅ Работает | — |
| action-menu-build | ✅ Работает | Есть тесты |

### Stories Progress

| Story | Название | Статус |
|-------|----------|--------|
| 5.1 | SonarQube Adapter Interface | 🟠 Legacy есть |
| 5.2 | Gitea Adapter Interface | 🟠 Legacy есть |
| 5.3 | nr-sq-scan-branch | 🟠 Legacy есть |
| 5.4 | nr-sq-scan-pr | 🟠 Legacy есть |
| 5.5 | nr-sq-report-branch | 🟠 Legacy есть |
| 5.6 | nr-sq-project-update | 🟠 Legacy есть |
| 5.7 | nr-test-merge | 🟠 Legacy есть |
| 5.8 | nr-action-menu-build | 🟠 Legacy есть |
| 5.9 | Command Summary (FR68) | 🔴 Ждёт Epic 1 |

---

## Цель

Реализовать интеграцию с SonarQube и Gitea на новой архитектуре.

## Ценность

Отчёты о качестве кода прямо в CLI. Решение Pain Point "Переключение в браузер".

---

## Stories

### Story 5.1: SonarQube Adapter Interface

**Приоритет:** P0 | **Размер:** S | **Риск:** Low
**Prerequisites:** Epic 1

**As a** разработчик,
**I want** иметь абстракцию над SonarQube API,
**So that** я могу тестировать без реального сервера.

**Acceptance Criteria:**

- [ ] Interface SonarQubeClient определён
- [ ] Методы: CreateProject, RunAnalysis, GetIssues, GetQualityGate
- [ ] Можно подставить mock для тестов

**Technical Notes:**
- Файл: `internal/adapter/sonarqube/interfaces.go`

---

### Story 5.2: Gitea Adapter Interface

**Приоритет:** P0 | **Размер:** S | **Риск:** Low
**Prerequisites:** Epic 1

**As a** разработчик,
**I want** иметь абстракцию над Gitea API,
**So that** я могу тестировать без реального сервера.

**Acceptance Criteria:**

- [ ] Interface GiteaClient определён
- [ ] Role-based interfaces: PRReader, CommitReader, FileReader
- [ ] Можно подставить mock для тестов

**Technical Notes:**
- Файл: `internal/adapter/gitea/interfaces.go`
- Ref: Architecture ADR-003

---

### Story 5.3: nr-sq-scan-branch (FR22)

**Приоритет:** P0 | **Размер:** M | **Риск:** Medium
**Prerequisites:** Story 5.1, 5.2

**As a** аналитик,
**I want** сканировать ветку на качество кода,
**So that** я знаю состояние кодовой базы.

**Acceptance Criteria:**

- [ ] BR_COMMAND=nr-sq-scan-branch BR_BRANCH=feature-123
- [ ] SonarQube сканирует коммиты ветки
- [ ] Фильтрация: только "main" или "t######" (6-7 цифр)
- [ ] Проверяет изменения в каталогах конфигурации перед сканированием
- [ ] Пропускает уже сканированные коммиты

**Technical Notes:**
- Файл: `internal/command/handlers/sonarqube/scanbranch.go`

---

### Story 5.4: nr-sq-scan-pr (FR23)

**Приоритет:** P0 | **Размер:** M | **Риск:** Low
**Prerequisites:** Story 5.3

**As a** аналитик,
**I want** сканировать pull request,
**So that** я знаю качество кода до merge.

**Acceptance Criteria:**

- [ ] BR_COMMAND=nr-sq-scan-pr BR_PR_NUMBER=123
- [ ] SonarQube сканирует изменения в PR
- [ ] Результат: new_issues, quality_gate_status

**Technical Notes:**
- Файл: `internal/command/handlers/sonarqube/scanpr.go`

---

### Story 5.5: nr-sq-report-branch (FR25)

**Приоритет:** P0 | **Размер:** M | **Риск:** Low
**Prerequisites:** Story 5.3

**As a** аналитик,
**I want** получить отчёт о качестве ветки,
**So that** я могу принять решение о merge.

**Acceptance Criteria:**

- [ ] BR_COMMAND=nr-sq-report-branch BR_BRANCH=feature-123
- [ ] Отчёт: новые ошибки между base и HEAD
- [ ] Summary: bugs, vulnerabilities, code_smells, coverage
- [ ] JSON output: детальный breakdown
- [ ] Text output: читаемый summary в CLI

**Technical Notes:**
- Файл: `internal/command/handlers/sonarqube/report.go`
- Journey Mapping: решает Pain Point "переключение в браузер"

---

### Story 5.6: nr-sq-project-update (FR24)

**Приоритет:** P1 | **Размер:** S | **Риск:** Low
**Prerequisites:** Story 5.1

**As a** DevOps-инженер,
**I want** обновить метаданные проекта в SonarQube,
**So that** проект настроен правильно.

**Acceptance Criteria:**

- [ ] BR_COMMAND=nr-sq-project-update
- [ ] Метаданные проекта обновляются в SonarQube

**Technical Notes:**
- Файл: `internal/command/handlers/sonarqube/projectupdate.go`

---

### Story 5.7: nr-test-merge (FR26)

**Приоритет:** P1 | **Размер:** M | **Риск:** Low
**Prerequisites:** Story 5.2

**As a** DevOps-инженер,
**I want** проверить конфликты слияния для всех открытых PR,
**So that** я знаю какие PR требуют внимания.

**Acceptance Criteria:**

- [ ] BR_COMMAND=nr-test-merge
- [ ] Проверяются все открытые PR на конфликты
- [ ] Результат: список PR с конфликтами и без
- [ ] JSON output: детали конфликтов

**Technical Notes:**
- Файл: `internal/command/handlers/gitea/testmerge.go`

---

### Story 5.8: nr-action-menu-build (FR27)

**Приоритет:** P2 | **Размер:** S | **Риск:** Low
**Prerequisites:** Story 5.2

**As a** DevOps-инженер,
**I want** построить динамическое меню действий,
**So that** пользователи видят доступные операции.

**Acceptance Criteria:**

- [ ] BR_COMMAND=nr-action-menu-build
- [ ] Меню строится из конфигурации
- [ ] JSON output для интеграции с UI

**Technical Notes:**
- Файл: `internal/command/handlers/gitea/actionmenu.go`

---

### Story 5.9: Command Summary (FR68)

**Приоритет:** P1 | **Размер:** M | **Риск:** Low
**Prerequisites:** Epic 1 (Story 1.3)

**As a** DevOps-инженер,
**I want** видеть summary с ключевыми метриками после каждой команды,
**So that** я сразу понимаю результат.

**Acceptance Criteria:**

- [ ] Любая команда завершается → summary: duration, key_metrics, warnings_count
- [ ] Text output: красивый summary в конце
- [ ] JSON output: metadata.summary object

**Technical Notes:**
- Расширение OutputWriter
- Каждый handler возвращает Summary в Result

---

## Risk Assessment

| Риск | Вероятность | Импакт | Митигация |
|------|-------------|--------|-----------|
| SonarQube API changes | Низкая | Средний | Версионирование API клиента |
| Gitea API rate limits | Средняя | Средний | Пагинация, кэширование |
| Большой объём данных | Средняя | Низкий | Streaming output, pagination |

---

## Definition of Done

- [ ] SQ-отчёты генерируются через NR-команды
- [ ] Summary работает для всех команд
- [ ] Integration тесты с mock серверами

---

## Связанные документы

- [Epic Overview](./index.md)
- [Epic 1: Foundation](./epic-1-foundation.md)
- [FR Coverage](./fr-coverage.md)

---

_Последнее обновление: 2026-01-26_
_Аудит проведён: 2026-01-26 (BMAD Party Mode)_
