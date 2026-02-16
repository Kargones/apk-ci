# Story 7.1: Shadow-run Mode (FR51)

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a DevOps-инженер,
I want запустить NR-команду параллельно со старой и сравнить результаты,
so that я уверен что новая версия работает идентично legacy-версии.

## Acceptance Criteria

1. [AC1] Переменная окружения `BR_SHADOW_RUN=true` активирует shadow-run режим для ЛЮБОЙ NR-команды (например, `BR_SHADOW_RUN=true BR_COMMAND=nr-service-mode-status`)
2. [AC2] В shadow-run режиме выполняются ОБЕ версии команды — NR (через command registry) и legacy (через `internal/app/` функции) — последовательно
3. [AC3] Результаты обеих версий автоматически сравниваются: exit code, наличие ошибки, ключевые поля данных
4. [AC4] Если результаты различаются — выводится structured diff с указанием расхождений
5. [AC5] Exit code = 0 если результаты идентичны; exit code = 1 если есть различия (NR-результат всё равно используется как основной)
6. [AC6] JSON вывод (`BR_OUTPUT_FORMAT=json`) содержит секцию `shadow_run` с результатами сравнения
7. [AC7] Текстовый вывод содержит summary сравнения после основного результата
8. [AC8] Если legacy-версия команды не найдена (нет mapping), shadow-run выводит warning и выполняет только NR-версию
9. [AC9] Shadow-run логирует время выполнения обеих версий для анализа производительности
10. [AC10] Shadow-run НЕ изменяет поведение при `BR_SHADOW_RUN=false` или при отсутствии переменной (backward compatible)

## Tasks / Subtasks

- [x] Task 1: Создать пакет `internal/command/shadowrun/` (AC: #1, #2, #8, #10)
  - [x] Subtask 1.1: Создать `shadowrun.go` — основная структура `Runner` с методами `Execute(ctx, cfg, nrHandler) error`
  - [x] Subtask 1.2: Создать `mapping.go` — маппинг NR-команд на legacy-функции из `internal/app/`
  - [x] Subtask 1.3: Создать `config.go` — чтение `BR_SHADOW_RUN` и валидация
  - [x] Subtask 1.4: Реализовать `IsEnabled()` функцию для проверки активации

- [x] Task 2: Реализовать захват вывода обеих версий (AC: #2, #3, #9)
  - [x] Subtask 2.1: Реализовать захват stdout/stderr NR-команды через `io.Writer` подмену
  - [x] Subtask 2.2: Реализовать захват результатов legacy-команды через вызов `app.*` функций
  - [x] Subtask 2.3: Реализовать замер времени выполнения каждой версии
  - [x] Subtask 2.4: Обработать различия в context: legacy принимает `*context.Context`, NR принимает `context.Context`

- [x] Task 3: Реализовать логику сравнения результатов (AC: #3, #4, #5)
  - [x] Subtask 3.1: Создать `comparison.go` — структура `ComparisonResult` с полями `Match bool`, `Differences []Difference`
  - [x] Subtask 3.2: Сравнение exit code / error: legacy `error == nil` vs NR `error == nil`
  - [x] Subtask 3.3: Сравнение ключевых данных (если доступны через JSON output)
  - [x] Subtask 3.4: Формирование читаемого diff для вывода

- [x] Task 4: Реализовать вывод результатов shadow-run (AC: #4, #6, #7)
  - [x] Subtask 4.1: Создать `output.go` — структура `ShadowRunResult` для JSON-секции
  - [x] Subtask 4.2: JSON-формат: добавить поле `shadow_run` к основному `output.Result`
  - [x] Subtask 4.3: Текстовый формат: summary после основного результата

- [x] Task 5: Интегрировать в main.go (AC: #1, #5, #10)
  - [x] Subtask 5.1: В `cmd/benadis-runner/main.go` добавить проверку `BR_SHADOW_RUN` перед выполнением NR-команды
  - [x] Subtask 5.2: Если shadow-run активен — вызвать `shadowrun.Runner` вместо прямого `handler.Execute()`
  - [x] Subtask 5.3: Правильно обработать exit code (0 если совпадение, 1 если различия)

- [x] Task 6: Написать тесты (AC: #1-#10)
  - [x] Subtask 6.1: Unit-тесты `mapping.go` — проверка маппинга для всех 18 команд с обеими версиями
  - [x] Subtask 6.2: Unit-тесты `comparison.go` — таблица случаев: идентичные, различия в error, различия в данных
  - [x] Subtask 6.3: Unit-тесты `config.go` — BR_SHADOW_RUN=true/false/пусто
  - [x] Subtask 6.4: Integration test `shadowrun_test.go` — полный цикл shadow-run через mock handler
  - [x] Subtask 6.5: Тест backward compatibility — без BR_SHADOW_RUN команда работает как обычно

- [x] Task 7: Добавить константы и обновить help (AC: #1)
  - [x] Subtask 7.1: Добавить `EnvShadowRun = "BR_SHADOW_RUN"` в `internal/constants/constants.go`
  - [x] Subtask 7.2: Обновить help-текст для отображения shadow-run опции

### Review Follow-ups (AI)

- [ ] [AI-Review][HIGH] Global mutex captureStdoutMu — os.Stdout hijack через sync.Mutex делает shadow-run несовместимым с параллельным выполнением и может deadlock при panic в capture fn [shadowrun/shadowrun.go:captureExecution]
- [ ] [AI-Review][MEDIUM] Legacy выполняется после NR для state-changing команд — при BR_SHADOW_RUN=true обе версии выполняются последовательно, legacy может изменить состояние повторно (enable → уже enabled) [shadowrun/shadowrun.go:Execute]
- [ ] [AI-Review][MEDIUM] TrimSpace normalization при сравнении — stdout сравнение через strings.TrimSpace может скрыть значимые различия в whitespace (trailing newlines) [shadowrun/comparison.go:CompareResults]
- [ ] [AI-Review][MEDIUM] runner.Execute использует тот же cfg для обоих — NR и legacy получают один и тот же *config.Config, мутации в одном могут повлиять на другой [shadowrun/shadowrun.go:Execute]
- [ ] [AI-Review][LOW] maxTruncateLen=500 рун — при больших stdout различиях отчёт обрезается без возможности увидеть полный diff [shadowrun/comparison.go]

### Review Follow-ups (AI Code Review #34)

- [ ] [AI-Review][HIGH] shadowMapping захардкожен — не синхронизируется с registry автоматически [shadow_mapping.go]
- [ ] [AI-Review][HIGH] compareResults использует string comparison — не учитывает JSON equivalence [shadow_comparison.go]
- [ ] [AI-Review][MEDIUM] Legacy функции вызываются с nil App — если handler зависит от DI [shadow_runner.go]
- [ ] [AI-Review][MEDIUM] Panic recovery может проглотить stack trace информацию [shadow_runner.go]

## Dev Notes

### Архитектурные паттерны и ограничения

**Это НЕ новый handler-command, а middleware/interceptor для существующих NR-команд.**
Shadow-run вызывается из `main.go` перед (или вместо) прямого вызова `handler.Execute()`. Он оборачивает выполнение NR-команды и дополнительно запускает legacy-версию.

**Критическое различие context:** Legacy функции из `internal/app/` принимают `*context.Context` (указатель), а NR handlers принимают `context.Context` (значение). При вызове legacy-функции из shadow-run необходимо создать отдельный `*context.Context`.

**Маппинг NR → Legacy (18 команд):** [Source: internal/command/registry.go, cmd/benadis-runner/main.go]
```
"nr-service-mode-status"   → app.ServiceModeStatus()
"nr-service-mode-enable"   → app.ServiceModeEnable()
"nr-service-mode-disable"  → app.ServiceModeDisable()
"nr-force-disconnect-sessions" → app.ForceDisconnectSessions() (проверить)
"nr-dbrestore"             → app.DbRestoreWithConfig()
"nr-dbupdate"              → app.DbUpdateWithConfig()
"nr-create-temp-db"        → app.CreateTempDbWrapper()
"nr-store2db"              → app.Store2DbWithConfig()
"nr-storebind"             → app.StoreBind() / convert.StoreBind()
"nr-create-stores"         → app.CreateStoresWrapper()
"nr-convert"               → app.Convert()
"nr-git2store"             → app.Git2Store()
"nr-execute-epf"           → app.ExecuteEpf()
"nr-sq-scan-branch"        → app.SQScanBranch()
"nr-sq-scan-pr"            → app.SQScanPR()
"nr-sq-project-update"     → app.SQProjectUpdate()
"nr-sq-report-branch"      → app.SQReportBranch()
"nr-test-merge"            → app.TestMerge()
"nr-action-menu-build"     → app.ActionMenuBuildWrapper()
```

**Команды БЕЗ legacy-версии** (AC8 — только warning):
- `nr-version` — нет legacy аналога в switch
- `nr-help` — нет legacy аналога

**Захват вывода:** NR-команды пишут в `os.Stdout` напрямую. Для сравнения нужно перехватить stdout через `os.Pipe()` или buffer, затем записать обратно. Legacy-команды тоже пишут в stdout через logger.

**Побочные эффекты:** Некоторые команды изменяют состояние (enable/disable service mode, dbrestore, dbupdate). Shadow-run для таких команд **ОПАСЕН** — legacy-версия может изменить состояние повторно. Варианты:
1. Запускать только NR-версию, legacy — в dry-run (если поддерживается)
2. Запускать обе, но legacy — первой, NR — второй
3. Только для read-only команд (status, scan, report)
4. **Рекомендация:** Пользователь решает через документацию; по умолчанию выполняются обе

### Файловая структура (СОЗДАТЬ)

```
internal/command/shadowrun/
├── shadowrun.go       # Runner — основная логика выполнения shadow-run
├── mapping.go         # LegacyMapping — маппинг NR→legacy функций
├── comparison.go      # CompareResults — сравнение результатов
├── output.go          # ShadowRunResult — структура вывода
├── config.go          # IsEnabled(), конфигурация
├── shadowrun_test.go  # Unit-тесты Runner
├── mapping_test.go    # Unit-тесты маппинга
└── comparison_test.go # Unit-тесты сравнения
```

### Существующий код для переиспользования

- **`internal/command/registry.go`** — `Get()` для получения NR handler
- **`internal/command/deprecated.go`** — `DeprecatedBridge` pattern (вдохновение для обёртки)
- **`internal/pkg/output/`** — `Result`, `Writer`, `NewWriter()` для форматирования
- **`internal/app/app.go`** — все legacy-функции для маппинга
- **`internal/constants/constants.go`** — константы имён команд (Act* и ActNR*)
- **`internal/pkg/tracing/`** — `TraceIDFromContext()`, `GenerateTraceID()`

### Паттерн Handler из проекта [Source: internal/command/handler.go]

```go
type Handler interface {
    Name() string
    Description() string
    Execute(ctx context.Context, cfg *config.Config) error
}
```

### Паттерн вывода из проекта [Source: internal/pkg/output/]

```go
// JSON: output.Result{Status, Command, Data, Error, Metadata}
// Text: data.writeText(os.Stdout)
// Format определяется: os.Getenv("BR_OUTPUT_FORMAT")
```

### Паттерн ошибок [Source: internal/command/handlers/*/handler.go]

```go
// Error codes: "SHADOW.NO_LEGACY_MAPPING", "SHADOW.LEGACY_EXEC_FAILED", "SHADOW.COMPARISON_FAILED"
// writeError(format, traceID, start, code, message)
```

### Env переменные

| Переменная | Тип | Default | Описание |
|-----------|-----|---------|----------|
| `BR_SHADOW_RUN` | bool | false | Активация shadow-run режима |
| `BR_OUTPUT_FORMAT` | string | text | Формат вывода (text/json) |

### Project Structure Notes

- Новый пакет `internal/command/shadowrun/` — рядом с `internal/command/handlers/` и `internal/command/registry.go`
- Shadow-run НЕ является handler (не регистрируется в registry) — это middleware/interceptor
- Маппинг функций из `internal/app/` требует импорта пакета `app` — может создать циклическую зависимость, проверить!
- Если циклическая зависимость — вынести маппинг в `cmd/benadis-runner/` или передавать legacy-функцию как параметр

### Предупреждения

1. **Циклические зависимости:** `internal/command/shadowrun/` импортирует `internal/app/` → проверить что `app` не импортирует `command`. Если да — использовать function type для маппинга
2. **os.Stdout hijack:** Перехват stdout через `os.Pipe()` может вызвать проблемы с concurrent writes. Использовать `bytes.Buffer` с подменой writer в handler
3. **Race conditions:** Legacy и NR выполняются последовательно (НЕ параллельно) чтобы избежать race conditions на shared state
4. **Тест isolation:** Тесты shadow-run НЕ должны вызывать реальные legacy-функции — использовать mock функции

### Testing Standards [Source: docs/quality/testing-strategy.md, architecture.md]

- Table-driven тесты для comparison logic
- Mock-based тесты для Runner (mock NR handler + mock legacy function)
- Формат: `TestShadowRun_<Scenario>` (например, `TestShadowRun_IdenticalResults`, `TestShadowRun_DifferentExitCodes`)
- Assertions через testify: `assert.Equal`, `assert.True`, `require.NoError`
- Тестовое покрытие: 80%+

### Предыдущие стории — уроки (Epic 6)

- **Расширяй, не переписывай** — изменения в main.go должны быть минимальными
- **Backward compatibility критична** — без BR_SHADOW_RUN всё должно работать как раньше
- **Env переменные с разумными defaults** — BR_SHADOW_RUN=false по умолчанию
- **Code review находит 10-17 issues** — готовиться к многократным итерациям

### References

- [Source: internal/command/registry.go] — Register, RegisterWithAlias, Get, DeprecatedBridge
- [Source: internal/command/handler.go] — Handler interface
- [Source: cmd/benadis-runner/main.go] — Двухуровневая диспетчеризация (registry → legacy switch)
- [Source: internal/app/app.go] — Legacy функции всех команд
- [Source: internal/pkg/output/result.go] — Result, Metadata, ErrorInfo
- [Source: internal/constants/constants.go:102-143] — ActNR* константы
- [Source: _bmad-output/project-planning-artifacts/epics/epic-7-finalization.md] — Story 7.1 requirements
- [Source: _bmad-output/project-planning-artifacts/prd.md] — FR51 (shadow-run)
- [Source: _bmad-output/project-planning-artifacts/architecture.md] — Command Registry, DeprecatedBridge patterns

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6

### Debug Log References

- Все тесты shadowrun пакета (38 тестов) прошли успешно
- 9 тестов shadow output (writeShadowRunTextSummary, mergeShadowRunJSON) прошли
- go vet прошёл без ошибок
- Race detector прошёл без ошибок
- Полная регрессия (все пакеты кроме cmd/benadis-runner) без ошибок
- cmd/benadis-runner тесты — предсуществующий fail из-за недоступности Gitea API (не регрессия)
- Покрытие shadowrun: 96.0% (было 93.5%)

### Completion Notes List

- Создан пакет `internal/command/shadowrun/` с 5 файлами: config.go, mapping.go, comparison.go, output.go, shadowrun.go
- Реализован Runner.Execute() — последовательно выполняет NR и legacy, захватывает stdout через os.Pipe(), сравнивает результаты
- LegacyMapping с унифицированной сигнатурой LegacyFunc и обёртками для функций с дополнительными параметрами
- Циклическая зависимость обойдена: маппинг в cmd/benadis-runner/shadow_mapping.go (не в internal/command/shadowrun/)
- CompareResults сравнивает error presence и stdout output (с whitespace normalization, truncation до 500 рун)
- ShadowRunResult с ToJSON() для сериализации duration в миллисекунды
- Интеграция в main.go через executeShadowRun() — минимальные изменения в существующем коде
- mergeShadowRunJSON() — объединяет shadow_run в JSON вывод NR-команды (единый JSON документ)
- writeShadowRunTextSummary(io.Writer) — текстовый summary с plain ASCII маркерами
- Константа EnvShadowRun вынесена в отдельную группу Env-констант в constants.go
- Help обновлён: секция "Опции" с BR_OUTPUT_FORMAT и BR_SHADOW_RUN
- 38 unit/integration тестов shadowrun + 9 тестов shadow output в cmd/benadis-runner

### Code Review Fixes Applied (Review #18)

- [#1 HIGH] os.Stdout hijack: добавлен sync.Mutex (captureStdoutMu) + рефакторинг дублирования capture в общий captureExecution()
- [#2 HIGH] truncate() UTF-8: исправлен на []rune для корректной обрезки кириллицы; добавлены 3 теста UTF-8
- [#3 HIGH] writeShadowRunOutput *os.File → writeShadowRunTextSummary(io.Writer): тестируемая через bytes.Buffer
- [#4 MEDIUM] JSON merge: mergeShadowRunJSON парсит NR output и добавляет shadow_run как поле (единый JSON)
- [#5 MEDIUM] Emoji удалены: заменены на plain ASCII маркеры [OK], [DIFF], [WARNING], [TIME]
- [#6 MEDIUM] State-changing warning: MarkStateChanging() + IsStateChanging() в LegacyMapping, 10 команд помечены
- [#7 MEDIUM] SQScanBranch: расширен комментарий об ограничении (legacy vs NR используют разные механизмы выбора коммитов)
- [#8 MEDIUM] Тесты: 9 новых тестов для writeShadowRunTextSummary и mergeShadowRunJSON в shadow_output_test.go
- [#10 LOW] EnvShadowRun перемещена в отдельную const-группу "Константы переменных окружения"

### File List

**Новые файлы:**
- internal/command/shadowrun/config.go
- internal/command/shadowrun/mapping.go
- internal/command/shadowrun/comparison.go
- internal/command/shadowrun/output.go
- internal/command/shadowrun/shadowrun.go
- internal/command/shadowrun/config_test.go
- internal/command/shadowrun/mapping_test.go
- internal/command/shadowrun/comparison_test.go
- internal/command/shadowrun/shadowrun_test.go
- cmd/benadis-runner/shadow_mapping.go
- cmd/benadis-runner/shadow_output_test.go

**Изменённые файлы:**
- cmd/benadis-runner/main.go (executeShadowRun, mergeShadowRunJSON, writeShadowRunTextSummary)
- internal/constants/constants.go (EnvShadowRun вынесена в отдельную const-группу)
- internal/command/handlers/help/help.go (добавлена секция "Опции" в help вывод)
- _bmad-output/implementation-artifacts/sprint-artifacts/sprint-status.yaml (7-1 → in-progress → review)

### Code Review Fixes Applied (Review #19)

- [#1 HIGH] terminateSessions hardcoded false: shadow_mapping.go — заменён `false` на `cfg.TerminateSessions` для корректного сравнения
- [#2 HIGH] Exit code документация: добавлен комментарий о приоритете exit code (NR-ошибка 8 > различия 1 > успех 0)
- [#3 HIGH] Panic safety captureExecution: рефакторинг на named returns + defer для гарантированного восстановления stdout при panic
- [#5 MEDIUM] Тесты state-changing warning: 2 новых теста (StateChangingWarning, NonStateChanging_NoWarning) с slog capture
- [#6 MEDIUM] BR_OUTPUT_FORMAT case-insensitive: strings.EqualFold вместо прямого сравнения
- [#7 MEDIUM] JSON field order: документирован как приемлемый (RFC 8259)

### Senior Developer Review (AI) — Review #32

**Reviewer:** Xor (adversarial cross-story review, Stories 7-1 through 7-6)
**Date:** 2026-02-07
**Status:** Approved with minor fix

**Issue fixed in Story 7.1:**
1. **[LOW] shadow_mapping.go комментарии** — обновлён комментарий о командах без legacy-аналога (5 команд, 18 из 23 зарегистрированы).

## Change Log

- 2026-02-07: Реализован shadow-run mode (FR51, Story 7.1). Пакет internal/command/shadowrun/ с middleware для сравнения NR и legacy команд. 34 теста. Интеграция в main.go с JSON и текстовым выводом результатов сравнения.
- 2026-02-07: Code Review #18 — исправлены 8 HIGH/MEDIUM issues: UTF-8 truncate, os.Stdout mutex, io.Writer, JSON merge, emoji→ASCII, state-changing warning, тесты. Покрытие 96.0%.
- 2026-02-07: Code Review #19 — исправлены 6 HIGH/MEDIUM issues: terminateSessions fix, panic safety, case-insensitive format, state-changing тесты. 41 тест shadowrun.
- 2026-02-07: Code Review #32 — обновлён комментарий shadow_mapping.go (LOW).
