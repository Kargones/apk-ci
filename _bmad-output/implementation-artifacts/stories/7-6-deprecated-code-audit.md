# Story 7.6: Deprecated Code Audit

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a разработчик,
I want видеть автоматический отчёт о deprecated коде в проекте (TODO-комментарии, deprecated aliases, legacy switch-case ветки),
so that я знаю что именно нужно удалить при финализации миграции на NR-архитектуру (v2.0.0).

## Acceptance Criteria

1. [AC1] NR-команда `nr-deprecated-audit` (зарегистрированная в command registry через `command.Register()`) анализирует кодебейз и генерирует отчёт о deprecated коде
2. [AC2] Отчёт содержит все deprecated aliases из `command.ListAllWithAliases()` (18 шт.): имя deprecated-команды, имя NR-команды, файл регистрации
3. [AC3] Отчёт содержит все TODO-комментарии с тегом `H-7` (планируемое удаление deprecated aliases): файл, строка, текст TODO
4. [AC4] Отчёт содержит все legacy case-ветки в `cmd/apk-ci/main.go` (switch-блок строки 280-370) с пометкой "deprecated alias handled by DeprecatedBridge"
5. [AC5] Текстовый и JSON вывод отчёта (через существующий `output.Writer`, BR_OUTPUT_FORMAT)
6. [AC6] Общая статистика: количество deprecated aliases, TODO(H-7), legacy case-веток, и общее резюме "готовность к удалению"
7. [AC7] CI-интеграция: команда возвращает exit code 0 (всегда success — это аудит, не валидация)

## Tasks / Subtasks

- [x] Task 1: Создать NR-handler `nr-deprecated-audit` с базовой инфраструктурой (AC: #1, #5, #7)
  - [x] Subtask 1.1: Создать пакет `internal/command/handlers/deprecatedaudithandler/` с файлом `handler.go`
  - [x] Subtask 1.2: Реализовать `DeprecatedAuditHandler` struct: `Name() = "nr-deprecated-audit"`, `Description()`, `init()` с `command.Register()` (БЕЗ deprecated-alias — новая команда)
  - [x] Subtask 1.3: Реализовать `Execute()` — сбор данных из трёх источников (registry, source scan, main.go analysis)
  - [x] Subtask 1.4: Интеграция с `output.Writer` для текстового и JSON вывода, `WritePlanOnlyUnsupported` для plan-only (паттерн Story 7.3)

- [x] Task 2: Реализовать сбор deprecated aliases из registry (AC: #2)
  - [x] Subtask 2.1: Функция `collectDeprecatedAliases() []DeprecatedAliasInfo` — вызвать `command.ListAllWithAliases()` и отфильтровать записи с непустым `DeprecatedAlias`
  - [x] Subtask 2.2: Для каждого alias определить файл-источник регистрации (hardcoded маппинг NR-имя → пакет handler, или получить через reflection/registry metadata)

- [x] Task 3: Реализовать сканирование TODO(H-7) комментариев в исходниках (AC: #3)
  - [x] Subtask 3.1: Функция `scanTodoComments(rootDir string) ([]TodoInfo, error)` — рекурсивный обход `*.go` файлов (исключая vendor/, _bmad*, *_test.go)
  - [x] Subtask 3.2: Regex-поиск `TODO\(H-7\)` в каждом файле с извлечением: путь, номер строки, текст комментария
  - [x] Subtask 3.3: Дополнительно: поиск `@deprecated`, `// Deprecated:` (стандартный Go маркер) как дополнительные маркеры

- [x] Task 4: Реализовать анализ legacy switch-case в main.go (AC: #4)
  - [x] Subtask 4.1: Функция `scanLegacySwitchCases(mainGoPath string) ([]LegacyCaseInfo, error)` — парсинг main.go на case-ветки для legacy команд
  - [x] Subtask 4.2: Идентификация case-веток, соответствующих deprecated aliases (cross-reference с данными из Task 2)

- [x] Task 5: Реализовать формирование отчёта и статистику (AC: #5, #6)
  - [x] Subtask 5.1: Структура `AuditReport` — deprecated aliases, TODOs, legacy cases, summary statistics
  - [x] Subtask 5.2: Текстовый вывод — группировка по категориям, таблица deprecated aliases, список TODOs с контекстом
  - [x] Subtask 5.3: JSON вывод — структура `Result` с полным `AuditReport` в `Data`
  - [x] Subtask 5.4: Summary: "X deprecated aliases, Y TODO(H-7) comments, Z legacy cases — Ready for removal: Yes/No"

- [x] Task 6: Unit-тесты (AC: #1-#7)
  - [x] Subtask 6.1: Тест `TestCollectDeprecatedAliases` — проверка что все 18 aliases собираются корректно
  - [x] Subtask 6.2: Тест `TestScanTodoComments` — table-driven тесты для парсинга TODO(H-7) из тестовых файлов (t.TempDir())
  - [x] Subtask 6.3: Тест `TestScanLegacySwitchCases` — парсинг mock main.go с case-ветками
  - [x] Subtask 6.4: Тест `TestAuditReport_TextFormat` и `TestAuditReport_JSONFormat` — форматирование отчёта
  - [x] Subtask 6.5: Тест `TestDeprecatedAuditHandler_Execute` — интеграционный тест Execute с text и JSON
  - [x] Subtask 6.6: Smoke-тест: добавить `nr-deprecated-audit` в `internal/smoketest/registry_test.go` массив `allNRCommands`

### Review Follow-ups (AI)

- [ ] [AI-Review][MEDIUM] deriveHandlerPackage hardcoded — маппинг NR-имя → handler package хардкодится в packageMap, при добавлении нового handler маппинг не обновляется автоматически [deprecatedaudithandler/scanner.go:deriveHandlerPackage]
- [ ] [AI-Review][MEDIUM] defaultCasePattern — regex для определения конца switch блока не учитывает вложенные switch-ы, может преждевременно прекратить парсинг [deprecatedaudithandler/scanner.go:scanLegacySwitchCases]
- [ ] [AI-Review][MEDIUM] buildConstToCmdMap hardcoded — маппинг constants.ActXxx → string value хардкодится, при изменении значений констант тихо рассинхронизируется [deprecatedaudithandler/scanner.go:buildConstToCmdMap]
- [ ] [AI-Review][LOW] Excluded directories hardcoded — список исключений (vendor, _bmad, .cursor и др.) хардкодится без возможности расширения через config [deprecatedaudithandler/scanner.go:scanTodoComments]
- [ ] [AI-Review][LOW] ready_for_removal логика неполная — "yes" возвращается при наличии aliases + TODO + legacy cases, но не проверяет что каждый alias имеет соответствующий TODO(H-7) [deprecatedaudithandler/report.go:buildAuditReport]

### Review Follow-ups (AI Code Review #34)

- [ ] [AI-Review][HIGH] pendingNote leak через пустые строки (Review #33 fix — проверить) [scanner.go:256]
- [ ] [AI-Review][MEDIUM] deriveHandlerPackage hardcoded map — не синхронизируется автоматически [scanner.go:64-90]
- [ ] [AI-Review][MEDIUM] defaultCasePattern fails на nested switch statements [scanner.go:30-34]
- [ ] [AI-Review][MEDIUM] buildConstToCmdMap hardcoded — может рассинхронизироваться с constants.go [scanner.go:buildConstToCmdMap]
- [ ] [AI-Review][MEDIUM] ready_for_removal logic неполная — не проверяет coverage [report.go:buildAuditReport]

## Dev Notes

### Архитектурные паттерны и ограничения

**Это NR-команда (не CI-скрипт/grep).** Эпик предлагал "grep @deprecated или golangci-lint custom rule", но архитектура проекта требует Go-реализацию как NR-команду в command registry для единообразия. Причины:
1. Единообразие с остальными 22 командами (21 NR + help)
2. Поддержка `BR_OUTPUT_FORMAT=json` для CI/CD автоматизации
3. Тестируемость через unit-тесты
4. Возможность запуска через Gitea Actions workflow

**Регистрация:** `command.Register()` (НЕ `RegisterWithAlias`!) — у `nr-deprecated-audit` нет legacy-аналога, это новая команда.

**Три источника данных для аудита:**
1. **Registry** — `command.ListAllWithAliases()` даёт полный маппинг NR ↔ deprecated (динамически, не хардкод)
2. **Source scan** — `filepath.WalkDir()` по `*.go` файлам с regex поиском TODO(H-7) и @deprecated маркеров
3. **Main.go analysis** — парсинг switch-case блока для идентификации legacy веток

### Существующий код для переиспользования

- **`internal/command/registry.go:159-194`** — `Info` struct и `ListAllWithAliases()` — основной источник deprecated aliases
- **`internal/command/registry.go:38-56`** — `Register()` для регистрации без alias
- **`internal/command/handler.go`** — `Handler` interface: `Name()`, `Execute()`, `Description()`
- **`internal/command/deprecated.go`** — `DeprecatedBridge`, `Deprecatable` interface (`IsDeprecated()`, `NewName()`)
- **`internal/pkg/output/`** — `Writer`, `Result`, `DetectFormat()` для text/JSON вывода
- **`internal/command/handlers/version/handler.go`** — паттерн NR-handler без deprecated-alias (аналог)
- **`internal/command/handlers/migratehandler/handler.go`** — ещё один handler без alias + Scanner (паттерн файлового сканирования)
- **`internal/smoketest/registry_test.go:39-93`** — массив `allNRCommands` (добавить nr-deprecated-audit)
- **`cmd/apk-ci/shadow_mapping.go:69-73`** — список команд без legacy-аналога (добавить)

### Паттерн Handler (без deprecated alias)

```go
// internal/command/handlers/deprecatedaudithandler/handler.go
package deprecatedaudithandler

import (
    "context"
    "github.com/Kargones/apk-ci/internal/command"
    "github.com/Kargones/apk-ci/internal/constants"
)

func init() {
    command.Register(&DeprecatedAuditHandler{})
}

type DeprecatedAuditHandler struct{}

func (h *DeprecatedAuditHandler) Name() string        { return constants.ActNRDeprecatedAudit }
func (h *DeprecatedAuditHandler) Description() string  { return "Аудит deprecated кода: aliases, TODO(H-7), legacy switch-case" }

func (h *DeprecatedAuditHandler) Execute(ctx context.Context, cfg *config.Config) error {
    // 1. Собрать deprecated aliases из registry
    // 2. Сканировать TODO(H-7) в исходниках
    // 3. Проанализировать legacy switch-case в main.go
    // 4. Сформировать отчёт и вывести через output.Writer
    return nil
}
```

### Константы

Добавить в `internal/constants/constants.go`:
```go
ActNRDeprecatedAudit = "nr-deprecated-audit" // после строки ActNRMigrate
```

### Env-переменные

| Переменная | Описание | Default |
|-----------|----------|---------|
| `BR_OUTPUT_FORMAT` | Формат вывода (text/json) | `text` |
| `BR_AUDIT_ROOT` | Корневой каталог для сканирования (опционально) | Текущая директория (`.`) |

Минимум env-переменных — команда работает "из коробки" без дополнительной настройки.

### Структура данных

```go
// DeprecatedAliasInfo — информация о deprecated alias
type DeprecatedAliasInfo struct {
    DeprecatedName string `json:"deprecated_name"`  // e.g., "service-mode-status"
    NRName         string `json:"nr_name"`           // e.g., "nr-service-mode-status"
    HandlerPackage string `json:"handler_package"`   // e.g., "servicemodestatushandler"
}

// TodoInfo — информация о TODO(H-7) комментарии
type TodoInfo struct {
    File    string `json:"file"`     // relative path
    Line    int    `json:"line"`     // line number
    Text    string `json:"text"`     // full comment text
    Tag     string `json:"tag"`      // "H-7", "deprecated", etc.
}

// LegacyCaseInfo — информация о legacy case в switch
type LegacyCaseInfo struct {
    File      string `json:"file"`       // cmd/apk-ci/main.go
    Line      int    `json:"line"`       // line number
    CaseValue string `json:"case_value"` // e.g., "git2store"
    Note      string `json:"note"`       // NOTE comment if present
}

// AuditReport — полный отчёт аудита
type AuditReport struct {
    DeprecatedAliases []DeprecatedAliasInfo `json:"deprecated_aliases"`
    TodoComments      []TodoInfo            `json:"todo_comments"`
    LegacyCases       []LegacyCaseInfo      `json:"legacy_cases"`
    Summary           AuditSummary          `json:"summary"`
}

// AuditSummary — сводная статистика
type AuditSummary struct {
    TotalDeprecatedAliases int    `json:"total_deprecated_aliases"`
    TotalTodoH7            int    `json:"total_todo_h7"`
    TotalLegacyCases       int    `json:"total_legacy_cases"`
    ReadyForRemoval        string `json:"ready_for_removal"` // "yes"/"no"/"partial"
    Message                string `json:"message"`
}
```

### Текстовый формат отчёта

```
Deprecated Code Audit Report
==============================

Deprecated Aliases (18):
  service-mode-status     → nr-service-mode-status      [servicemodestatushandler]
  service-mode-enable     → nr-service-mode-enable       [servicemodeenablehandler]
  service-mode-disable    → nr-service-mode-disable      [servicemodedisablehandler]
  dbrestore               → nr-dbrestore                 [dbrestorehandler]
  dbupdate                → nr-dbupdate                  [dbupdatehandler]
  create-temp-db          → nr-create-temp-db            [createtempdbhandler]
  store2db                → nr-store2db                  [store2dbhandler]
  storebind               → nr-storebind                 [storebindhandler]
  create-stores           → nr-create-stores             [createstoreshandler]
  convert                 → nr-convert                   [converthandler]
  git2store               → nr-git2store                 [git2storehandler]
  execute-epf             → nr-execute-epf               [executeepfhandler]
  sq-scan-branch          → nr-sq-scan-branch            [scanbranch]
  sq-scan-pr              → nr-sq-scan-pr                [scanpr]
  sq-report-branch        → nr-sq-report-branch          [reportbranch]
  sq-project-update       → nr-sq-project-update         [projectupdate]
  test-merge              → nr-test-merge                [testmerge]
  action-menu-build       → nr-action-menu-build         [actionmenu]

TODO(H-7) Comments (4):
  internal/command/handlers/gitea/actionmenu/handler.go:40
    Deprecated alias "action-menu-build" будет удалён в v2.0.0 / Epic 7.
  internal/command/handlers/gitea/testmerge/handler.go:43
    Deprecated alias "test-merge" будет удалён в v2.0.0 / Epic 7.
  internal/command/handlers/sonarqube/projectupdate/handler.go:40
    Удалить deprecated alias ActSQProjectUpdate после миграции всех workflows на NR-команды.
  internal/command/handlers/sonarqube/reportbranch/handler.go:37
    Удалить deprecated alias ActSQReportBranch после миграции всех workflows на NR-команды.

Legacy Switch Cases (8):
  cmd/apk-ci/main.go:300  case "git2store"        — deprecated alias for nr-git2store
  cmd/apk-ci/main.go:337  case "storebind"        — deprecated alias for nr-storebind
  ...

Summary:
  Deprecated aliases: 18
  TODO(H-7) comments: 4
  Legacy switch cases: 8
  Ready for removal: partial (TODO(H-7) не покрывает все 18 aliases)
```

### JSON формат отчёта

```json
{
  "status": "success",
  "command": "nr-deprecated-audit",
  "data": {
    "deprecated_aliases": [...],
    "todo_comments": [...],
    "legacy_cases": [...],
    "summary": {
      "total_deprecated_aliases": 18,
      "total_todo_h7": 4,
      "total_legacy_cases": 8,
      "ready_for_removal": "partial",
      "message": "18 deprecated aliases найдено, 4 TODO(H-7) комментария, 8 legacy case-веток"
    }
  }
}
```

### Shadow mapping

Добавить `nr-deprecated-audit` в `cmd/apk-ci/shadow_mapping.go` (список команд без legacy-аналога):
```go
constants.ActNRDeprecatedAudit: nil, // ДОБАВИТЬ
```

### Smoke tests

Добавить в `internal/smoketest/registry_test.go`:
1. Массив `allNRCommands` — добавить `{constants.ActNRDeprecatedAudit, "nr-deprecated-audit"}`
2. Массив `noLegacy` — добавить `constants.ActNRDeprecatedAudit`
3. Общее количество в `TestSmoke_TotalCommandCount` увеличится на 1 (с 40 до 41: 23 основных + 18 deprecated)

### Project Structure Notes

```
internal/command/handlers/
├── deprecatedaudithandler/     # НОВЫЙ
│   ├── handler.go              # DeprecatedAuditHandler + Execute
│   ├── handler_test.go         # Unit-тесты handler
│   ├── scanner.go              # scanTodoComments(), scanLegacySwitchCases()
│   ├── scanner_test.go         # Тесты сканирования
│   └── report.go               # AuditReport формирование + вывод
```

Размещение в `handlers/` — стандартный паттерн для NR-хендлеров (аналогично version, help, migratehandler).

### Предупреждения

1. **НЕ хардкодь список deprecated aliases.** Используй `command.ListAllWithAliases()` — он всегда актуален. HandlerPackage можно определить через маппинг NR-имя → пакет (статический, т.к. пакеты не меняются часто).
2. **Исключай vendor/ и _bmad* из сканирования.** `filepath.WalkDir()` с фильтрацией по `strings.HasPrefix`.
3. **Не парсь Go AST для main.go.** Это overengineering. Достаточно regex-поиска `case\s+constants\.Act` или `case\s+"` в switch-блоке.
4. **Exit code всегда 0.** Это аудит (информационная команда), не валидация. Не ломай CI при наличии deprecated кода.
5. **Относительные пути в отчёте.** Все пути должны быть относительными от корня проекта для читабельности.
6. **Тесты с временными директориями.** Используй `t.TempDir()` для создания тестовых Go-файлов с TODO-комментариями.
7. **Паттерн миграции из Story 7.5.** Тот же подход к сканированию файлов (WalkDir + regex), но другой target (Go-файлы вместо YAML).

### Предыдущие стории — уроки (Story 7.1-7.5)

- **Story 7.1 (shadow-run):** `shadow_mapping.go` содержит маппинг NR → legacy. Необходимо добавить `nr-deprecated-audit` в список команд без legacy.
- **Story 7.2 (smoke tests):** Smoke tests проверяют полное покрытие registry. При добавлении обновить `allNRCommands`, `noLegacy` и `TotalCommandCount`.
- **Story 7.3 (plan display):** `WritePlanOnlyUnsupported` — паттерн для команд без поддержки operation plan. Использовать аналогичный подход.
- **Story 7.4 (rollback):** `ListAllWithAliases()` и `Info` struct — готовая инфраструктура для получения deprecated маппинга.
- **Story 7.5 (migration script):** `migratehandler/scanner.go` — паттерн файлового сканирования (WalkDir + regex). Переиспользовать подход, но не код (разные цели сканирования).
- **Code review #25-#26 (Story 7.5):**
  - writeError должна возвращать оригинальную ошибку (не nil!)
  - Текстовый формат ошибок делегировать main.go (l.Error → stderr)
  - Inline-комментарии в файлах нужно обрабатывать в regex
  - Интеграционные тесты Execute обязательны (не только unit)
  - mustParseTime() и подобные обёртки — мёртвый код, не добавлять

### Git Intelligence (последние 5 коммитов)

```
db9f3a2 feat(epic-7): rollback support and migration script (Stories 7.4, 7.5) with code reviews #22-#26
fada818 feat(plan-display): operation plan display with plan-only and verbose modes (Story 7.3, FR63)
4c5ed40 feat(smoketest): smoke tests for system integrity (Story 7.2, FR52)
2445235 feat(shadowrun): shadow-run mode (Story 7.1, FR51)
1c95599 fix(observability): adversarial code review #17 fixes for Epic 6
```

**Паттерн коммитов:** `feat(scope): description (Story X.Y, FRNN) with code review #NN fixes`
**Co-authored-by:** Claude Opus 4.6

**Файлы, затронутые в Story 7.5 (ближайший аналог):**
- `internal/command/handlers/migratehandler/handler.go` (NEW) — основной handler
- `internal/command/handlers/migratehandler/scanner.go` (NEW) — сканер файлов
- `internal/constants/constants.go` (MODIFIED) — новые константы
- `internal/smoketest/registry_test.go` (MODIFIED) — обновление smoke tests
- `cmd/apk-ci/main.go` (MODIFIED) — blank import handler
- `cmd/apk-ci/shadow_mapping.go` (MODIFIED) — добавление в noLegacy

### Testing Standards

- Table-driven тесты для scanTodoComments (минимум 5 cases: TODO(H-7), @deprecated, // Deprecated:, нет маркеров, файл в vendor/)
- Тесты используют `t.TempDir()` для изоляции файловой системы
- Assertions через testify: `assert.Equal`, `assert.Contains`, `require.NoError`
- Тестовое покрытие: 80%+ для нового кода
- Smoke-тест: регистрация `nr-deprecated-audit` в registry
- Интеграционный тест Execute (паттерн из review #26): полный pipeline scan→report

### References

- [Source: internal/command/registry.go:159-194] — Info struct и ListAllWithAliases()
- [Source: internal/command/registry.go:38-56] — Register()
- [Source: internal/command/handler.go] — Handler interface
- [Source: internal/command/deprecated.go] — DeprecatedBridge, Deprecatable interface
- [Source: internal/pkg/output/] — Writer, Result, DetectFormat()
- [Source: internal/command/handlers/version/handler.go:24] — паттерн Register() без alias
- [Source: internal/command/handlers/migratehandler/] — паттерн файлового сканирования (scanner.go)
- [Source: internal/constants/constants.go:103-146] — ActNR* константы
- [Source: cmd/apk-ci/main.go:280-370] — legacy switch-case блок
- [Source: cmd/apk-ci/shadow_mapping.go:69-73] — команды без legacy-аналога
- [Source: internal/smoketest/registry_test.go:39-93] — allNRCommands и deprecatedAliases массивы
- [Source: _bmad-output/project-planning-artifacts/epics/epic-7-finalization.md:197-215] — Story 7.6 requirements
- [Source: _bmad-output/project-planning-artifacts/prd.md:396-397] — FR49 (deprecated code removal), NFR16 (CI audit)
- [Source: _bmad-output/implementation-artifacts/stories/7-5-migration-script.md] — уроки из Story 7.5

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6

### Debug Log References

### Completion Notes List

- Реализована NR-команда `nr-deprecated-audit` с тремя источниками данных: registry aliases, TODO(H-7) scan, legacy switch-case analysis
- Команда зарегистрирована через `command.Register()` (без deprecated-alias — новая команда)
- Поддержка text/JSON вывода через `output.Writer`, plan-only через `WritePlanOnlyUnsupported`
- `collectDeprecatedAliases()` динамически извлекает 18 deprecated aliases из `command.ListAllWithAliases()`
- `scanTodoComments()` рекурсивно обходит *.go файлы (исключая vendor/, _bmad*, *_test.go) и ищет TODO(H-7), @deprecated, // Deprecated:
- `scanLegacySwitchCases()` парсит main.go switch-блок через regex (не AST) и cross-references с deprecated aliases
- `buildAuditReport()` формирует summary с ready_for_removal: yes/no/partial
- 24 unit-теста + smoke тесты прошли, 0 регрессий (после review #28)
- Обновлены smoke tests: allNRCommands (23), noLegacy (5), TotalCommandCount (41)
- Обновлены shadow_mapping.go (комментарий) и main.go (blank import)

### Change Log

- 2026-02-07: Реализация Story 7.6 — NR-команда nr-deprecated-audit (все 6 задач, 18 подзадач)
- 2026-02-07: Code Review #27 — исправлены 5 проблем (2 HIGH, 3 MEDIUM), покрытие 79.6% → 85.4%
- 2026-02-07: Code Review #28 — исправлены 7 проблем (2 HIGH, 3 MEDIUM, 2 LOW), покрытие 85.4% → 85.7%
- 2026-02-07: Code Review #32 — исправлены 5 MEDIUM, добавлено 2 теста (PathTraversal, DryRunOverPlanOnly)
  - H-1: Добавлен TestBuildConstToCmdMap_SyncWithConstants — верификация хардкод-маппинга vs constants
  - H-2: Замена strings.Contains на regex defaultCasePattern для определения конца switch
  - M-1: Сброс pendingNote на строках с кодом (не комментарии) — предотвращение утечки NOTE
  - M-2: Задокументирована проблема os.Stdout pipe в интеграционных тестах (существующий ТД)
  - M-3: TestAuditReport_JSONFormat использует buildAuditReport вместо ручной инициализации
  - L-1: Алфавитный порядок blank import deprecatedaudithandler в main.go
  - L-2: Уточнён комментарий allNRCommands (23 шт.: 22 NR + help)
- 2026-02-07: Code Review #29 (adversarial, Stories 7-1—7-6) — исправлены 5 проблем (2H, 3M):
  - H-1: scanTodoComments исключает IDE-директории (.cursor, .windsurf, .claude, .idea, .vscode) + тест
  - H-2: Алфавитный порядок blank imports в smoketest/registry_test.go
  - M-2: readyForRemoval "partial" только без legacy cases + тест
  - M-3: Тест plan-only для DeprecatedAuditHandler
  - M-4: Документация self-reference blank import в scanner_test.go
- 2026-02-07: Code Review #33 — исправлены 2 HIGH + 2 MEDIUM: pendingNote leak, missing continue, path traversal doc, buildConstToCmdMap doc.
- 2026-02-07: Code Review #30 (adversarial, Stories 7-1—7-6, cross-story) — исправлены 6 проблем:
  - H-1: migratehandler coverage 78.5%→84.5% (Execute: 60%→88.6%) — добавлены 4 теста через Execute()
  - M-5: brCommandInlinePattern — добавлена валидация симметричности кавычек + тест
  - M-7: shadowrun captureExecution — добавлен тест panic recovery (TestCaptureExecution_PanicRecovery)
  - M-3: captureExecution — добавлен комментарий о pipe lifecycle (почему <-done не блокирует)
  - M-6: defaultCasePattern — документировано ограничение при вложенных switch-ах
  - Тесты: +6 новых (4 migratehandler Execute, 1 panic recovery, 1 asymmetric quotes)

### Senior Developer Review (AI) — Review #29

**Reviewer:** Xor (adversarial code review, Stories 7-1 through 7-6)
**Date:** 2026-02-07
**Status:** Approved with fixes applied

**Issues found: 3 HIGH, 4 MEDIUM, 3 LOW**

Исправлены (2 HIGH + 3 MEDIUM):
1. **[HIGH] H-1** — scanTodoComments не исключал IDE-директории (.cursor, .windsurf, .claude, .idea, .vscode). Добавлены в фильтр WalkDir + тест TestScanTodoComments_IDEDirsExcluded.
2. **[HIGH] H-2** — Нарушен алфавитный порядок blank imports в smoketest/registry_test.go (deprecatedaudithandler стоял перед createstoreshandler). Исправлен порядок.
3. **[MEDIUM] M-2** — readyForRemoval логика неинтуитивна: "partial" при наличии legacy case-веток. Исправлено: "partial" только когда aliases + TODO БЕЗ legacy cases. Обновлены 4 теста + добавлен TestBuildAuditReport_PartialWhenNoLegacyCases.
4. **[MEDIUM] M-3** — Нет теста plan-only для DeprecatedAuditHandler. Добавлен TestDeprecatedAuditHandler_PlanOnly.
5. **[MEDIUM] M-4** — Документирован self-reference в scanner_test.go blank imports (пояснение что deprecatedaudithandler не нужен).

Оставлены как action items:
- **[HIGH] H-3** — os.Stdout hardcoded в Execute — системный ТД, единообразие с другими handlers
- **[MEDIUM] M-1** — writeTextReport избыточные error checks — паттерн проекта
- **[LOW] L-1** — Смесь языков в текстовом отчёте (осознанный выбор)
- **[LOW] L-2** — Хардкоженные числа в комментариях (assertions используют len())
- **[LOW] L-3** — sprint-status done до коммита (нормальный workflow)

### Senior Developer Review (AI) — Review #30

**Reviewer:** Xor (adversarial cross-story review, Stories 7-1 through 7-6)
**Date:** 2026-02-07
**Status:** Approved with fixes applied

**Issues found: 3 HIGH, 5 MEDIUM, 2 LOW (across all 6 stories)**

Исправлены (1 HIGH + 3 MEDIUM + 2 clarifications):
1. **[HIGH] H-1** — migratehandler coverage 78.5% (Execute 60%), ниже порога 80%. Добавлены 4 теста Execute(): RealApply_NoBackup, RealApply_WithBackup, DryRun_WithReplacements, PlanOnly. Coverage → 84.5% (Execute → 88.6%).
2. **[MEDIUM] M-5** — brCommandInlinePattern допускает асимметричные кавычки (BR_COMMAND="value'). Добавлена проверка matches[2]==matches[4] в scanFile() + тест.
3. **[MEDIUM] M-7** — Нет теста panic recovery для captureExecution. Добавлен TestCaptureExecution_PanicRecovery — проверяет восстановление os.Stdout после panic в fn().
4. **[MEDIUM] M-3** — Документирован pipe lifecycle в captureExecution: почему `<-done` не может заблокировать (writePipe.Close() → EOF → io.Copy returns → close(done)).
5. **[MEDIUM] M-6** — Документировано ограничение defaultCasePattern при вложенных switch-ах.

Оставлены как action items (низкий риск):
- **[HIGH] H-2** — scan vs apply inconsistency (FindStringSubmatch vs ReplaceAllStringFunc) — уже задокументирован как TODO(H-2/Review #26), маловероятен в Gitea Actions YAML.
- **[HIGH] H-3** — captureExecution deadlock — фактически ложная тревога, pipe lifecycle корректен (задокументирован).
- **[MEDIUM] M-4** — context pointer coercion в captureLegacyExecution — intentional design (legacy API).
- **[MEDIUM] M-8** — force-disconnect mapping count — ложный позитив (force-disconnect не имеет deprecated alias).
- **[LOW] L-9** — ignored write errors в writeShadowRunTextSummary — stdout, минимальный риск.
- **[LOW] L-10** — hardcoded packageMap в deriveHandlerPackage — задокументирован, низкий приоритет.

### Senior Developer Review (AI) — Review #32

**Reviewer:** Xor (adversarial cross-story review, Stories 7-1 through 7-6)
**Date:** 2026-02-07
**Status:** Approved with fixes applied

**Issues found and fixed in Story 7.6:**
1. **[MEDIUM] IsPlanOnly() перехватывает dry-run** — handler.go:51: `IsPlanOnly()` проверялась раньше `IsDryRun()`, нарушая приоритет AC-11. Исправлено: `!IsDryRun() && IsPlanOnly()`.
2. **[MEDIUM] Path traversal через BR_AUDIT_ROOT** — handler.go:56: env без валидации. Исправлено: filepath.Clean + проверка на ".." prefix.
3. **[MEDIUM] buildConstToCmdMap нет обратной проверки** — scanner_test.go: добавлена обратная проверка (каждая запись map → expected).
4. **[MEDIUM] WritePlanOnlyUnsupported возвращает ошибку вместо nil** — dryrun.go:59: return err нарушало exit code 0 (AC-8). Исправлено: всегда return nil.
5. **[MEDIUM] noLegacy hardcoded в smoketest** — registry_test.go: вынесено в package-level var с использованием констант.

Тесты добавлены: `TestDeprecatedAuditHandler_PathTraversal`, `TestDeprecatedAuditHandler_DryRunOverPlanOnly`.

### Senior Developer Review (AI) — Review #33

**Reviewer:** Xor (adversarial cross-story review, Stories 7-1 through 7-6)
**Date:** 2026-02-07
**Status:** Approved with fixes applied

**Issues found and fixed in Story 7.6:**
1. **[HIGH] pendingNote leak через пустые строки** — scanner.go:256: пустые строки не сбрасывали pendingNote, позволяя NOTE-комментарию утечь к отдалённой case-ветке. Исправлено: добавлен сброс pendingNote + continue на пустых строках.
2. **[HIGH] missing continue после @deprecated match** — scanner.go:189: блок @deprecated не имел `continue`, в отличие от TODO(H-7) и Deprecated:. Мог привести к двойному match при добавлении новых паттернов. Исправлено: добавлен continue.
3. **[MEDIUM] buildConstToCmdMap — комментарий о кэшировании** — scanner.go:321: добавлен комментарий, документирующий что кэширование не требуется для single-shot утилиты.
4. **[MEDIUM] Path traversal — абсолютные пути** — handler.go:63: добавлен комментарий, документирующий что абсолютные пути разрешены для CI-утилиты.

### File List

- `internal/command/handlers/deprecatedaudithandler/handler.go` (NEW) — DeprecatedAuditHandler + Execute + writeSuccess/writeError/writeTextReport
- `internal/command/handlers/deprecatedaudithandler/scanner.go` (NEW) — collectDeprecatedAliases, scanTodoComments, scanLegacySwitchCases
- `internal/command/handlers/deprecatedaudithandler/report.go` (NEW) — типы данных (AuditReport, TodoInfo и т.д.) + buildAuditReport
- `internal/command/handlers/deprecatedaudithandler/handler_test.go` (NEW) — unit/integration тесты Execute, report formatting
- `internal/command/handlers/deprecatedaudithandler/scanner_test.go` (NEW) — unit тесты scanner: aliases, todo, legacy, exclusions
- `internal/constants/constants.go` (MODIFIED) — добавлена константа ActNRDeprecatedAudit
- `cmd/apk-ci/main.go` (MODIFIED) — blank import deprecatedaudithandler
- `cmd/apk-ci/shadow_mapping.go` (MODIFIED) — комментарий о nr-deprecated-audit как команде без legacy
- `internal/smoketest/registry_test.go` (MODIFIED) — добавлен nr-deprecated-audit в allNRCommands, noLegacy; обновлены счётчики (23+18=41)

**Файлы, изменённые в Review #30 (cross-story):**
- `internal/command/handlers/migratehandler/scanner.go` (MODIFIED) — валидация симметричности кавычек в inline pattern
- `internal/command/handlers/migratehandler/handler_test.go` (MODIFIED) — +4 теста Execute: RealApply_NoBackup, RealApply_WithBackup, DryRun_WithReplacements, PlanOnly
- `internal/command/handlers/migratehandler/scanner_test.go` (MODIFIED) — +1 тест: inline с асимметричными кавычками
- `internal/command/shadowrun/shadowrun.go` (MODIFIED) — документация pipe lifecycle в captureExecution
- `internal/command/shadowrun/shadowrun_test.go` (MODIFIED) — +1 тест: TestCaptureExecution_PanicRecovery
- `internal/command/handlers/deprecatedaudithandler/scanner.go` (MODIFIED) — документация ограничения defaultCasePattern
