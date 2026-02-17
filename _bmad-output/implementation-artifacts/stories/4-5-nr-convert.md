# Story 4.5: nr-convert (FR19-20)

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 1C-разработчик,
I want конвертировать между форматами EDT и XML через NR-команду,
So that могу работать с разными инструментами в автоматизированном pipeline.

## Acceptance Criteria

1. **AC-1**: `BR_COMMAND=nr-convert BR_SOURCE=/path/edt BR_TARGET=/path/xml BR_DIRECTION=edt2xml` конвертирует EDT → XML
2. **AC-2**: `BR_DIRECTION=xml2edt` конвертирует XML → EDT
3. **AC-3**: Инструмент выбирается через config.implementations.config_export (1cv8/1cedtcli)
4. **AC-4**: JSON output содержит: `status`, `state_changed`, `source_path`, `target_path`, `direction`, `tool_used`, `duration_ms`
5. **AC-5**: Text output показывает человекочитаемую информацию о результате конвертации
6. **AC-6**: Deprecated alias `convert` работает через DeprecatedBridge с warning
7. **AC-7**: При успешной конвертации `state_changed: true`
8. **AC-8**: Unit-тесты покрывают все сценарии (success, error, both directions)
9. **AC-9**: Все тесты проходят (`make test`), линтер проходит (`make lint`)
10. **AC-10**: При ошибке возвращается структурированная ошибка с кодом `ERR_CONVERT`
11. **AC-11**: Progress logging: `validating → preparing → converting → completing`

## Tasks / Subtasks

- [x] Task 1: Создать Handler (AC: 1, 2, 4, 5, 6)
  - [x] 1.1 Создать пакет `internal/command/handlers/converthandler/`
  - [x] 1.2 Определить `ConvertHandler` struct с интерфейсом Converter
  - [x] 1.3 Реализовать `Name()` → `"nr-convert"`
  - [x] 1.4 Реализовать `Description()` → "Конвертация между форматами EDT и XML"
  - [x] 1.5 Зарегистрировать через `init()` + `command.RegisterWithAlias()`
  - [x] 1.6 Добавить compile-time interface check

- [x] Task 2: Определить Data Structures (AC: 4, 5)
  - [x] 2.1 `ConvertData` struct с полями: `StateChanged`, `SourcePath`, `TargetPath`, `Direction`, `ToolUsed`, `DurationMs`
  - [x] 2.2 Реализовать `writeText()` для человекочитаемого вывода

- [x] Task 3: Реализовать Execute метод (AC: 1, 2, 3, 7, 10, 11)
  - [x] 3.1 Валидация BR_SOURCE (обязательный)
  - [x] 3.2 Валидация BR_TARGET (обязательный)
  - [x] 3.3 Валидация BR_DIRECTION (обязательный: `edt2xml` или `xml2edt`)
  - [x] 3.4 Проверка существования source path через os.Stat
  - [x] 3.5 Progress: `validating → preparing → converting → completing`
  - [x] 3.6 Выбор инструмента через OneCFactory (config.implementations.config_export)
  - [x] 3.7 Вызов edt.Cli.Convert() для выполнения конвертации
  - [x] 3.8 Собрать результат и вернуть через OutputWriter

- [x] Task 4: Определить интерфейс Converter (AC: 8)
  - [x] 4.1 `Converter` interface для тестируемости (абстрагирует edt.Cli)
  - [x] 4.2 Реализовать mock implementation для тестов

- [x] Task 5: Интеграция с OutputWriter (AC: 4, 5)
  - [x] 5.1 JSON output: использовать `output.WriteJSON()`
  - [x] 5.2 Text output: использовать `ConvertData.writeText()`
  - [x] 5.3 Error output: использовать `output.WriteError()` с кодом `ERR_CONVERT`

- [x] Task 6: Написать тесты (AC: 8, 9)
  - [x] 6.1 `handler_test.go`: unit-тесты Execute
  - [x] 6.2 Тест success case (edt2xml)
  - [x] 6.3 Тест success case (xml2edt)
  - [x] 6.4 Тест error case (нет BR_SOURCE)
  - [x] 6.5 Тест error case (нет BR_TARGET)
  - [x] 6.6 Тест error case (нет BR_DIRECTION)
  - [x] 6.7 Тест error case (недопустимый BR_DIRECTION)
  - [x] 6.8 Тест error case (source path не существует)
  - [x] 6.9 Тест error case (ошибка конвертации)
  - [x] 6.10 Тест deprecated alias через registry
  - [x] 6.11 Тест progress logs (перехват slog)
  - [x] 6.12 Тест compile-time interface check

- [x] Task 7: Валидация (AC: 9)
  - [x] 7.1 Запустить `make test` — все тесты проходят
  - [x] 7.2 Запустить `make lint` — golangci-lint проходит
  - [x] 7.3 Проверить что legacy команда продолжает работать

### Review Follow-ups (AI Code Review #34)

- [ ] [AI-Review][HIGH] validatePath() не проверяет symlinks — /tmp/symlink_to_etc пройдёт [handler.go:42-55]
- [ ] [AI-Review][HIGH] cfg.AppConfig не проверяется на nil перед доступом к EdtTimeout [handler.go:228-230]
- [ ] [AI-Review][MEDIUM] cli.Convert() результат через скрытый side-effect cli.LastErr [production.go:58]
- [ ] [AI-Review][MEDIUM] os.Stat(source) выполняется ДО validatePath(source) — неправильный порядок [handler.go:191-195]

## Dev Notes

### Архитектурные ограничения

- **Все комментарии на русском языке** (CLAUDE.md)
- **Command Registry Pattern** — self-registration через `init()` + `command.RegisterWithAlias()`
- **Dual output** — JSON (BR_OUTPUT_FORMAT=json) / текст (по умолчанию)
- **StateChanged field** — обязателен для операций изменения состояния
- **НЕ менять legacy код** — `edt.Cli`, `edt.Convert` остаются, NR-handler переиспользует логику

### Существующая Legacy-реализация

**Точка входа** (`cmd/apk-ci/main.go:74-75`):
```go
case constants.ActConvert:
    err = app.Convert(&ctx, l, cfg)
```

**Оркестрация** (`internal/app/app.go:68-131`):
```go
func Convert(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
    // 1. Создает временную директорию
    cfg.RepPath, err = os.MkdirTemp(cfg.WorkDir, "s")

    // 2. Клонирует репозиторий
    g, err := InitGit(l, cfg)
    g.Clone(ctx, l)

    // 3. Переключается на main
    g.Branch = "main"
    g.Switch(*ctx, l)

    // 4. Загружает конфигурацию конвертора
    c := &edt.Convert{}
    c.MustLoad(l, cfg)

    // 5. Переключается на исходную ветку
    g.Branch = c.Source.Branch
    g.Switch(*ctx, l)

    // 6. Выполняет конвертацию с таймаутом
    edtCtx, cancel := context.WithTimeout(*ctx, cfg.AppConfig.EdtTimeout)
    defer cancel()
    c.Convert(&edtCtx, l, cfg)
}
```

**EDT Cli** (`internal/entity/one/edt/edt.go:64-72`):
```go
type Cli struct {
    CliPath   string  // Путь к 1cedtcli
    Direction string  // xml2edt или edt2xml
    PathIn    string  // Исходный путь
    PathOut   string  // Целевой путь
    WorkSpace string  // Рабочая область EDT
    Operation string
    LastErr   error
}
```

**Конвертация** (`internal/entity/one/edt/edt.go:136-247`):
```go
func (c *Convert) Convert(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
    r := Cli{}
    r.Init(cfg)

    // Определяет направление по Source.Format
    switch c.Source.Format {
    case formatXML:
        r.Direction = XML2edt
    case formatEDT:
        r.Direction = Edt2xml
    }

    // Создает рабочую область
    WorkSpace, err := os.MkdirTemp(cfg.WorkDir, "ws")
    r.WorkSpace = WorkSpace

    // Для каждого mapping выполняет конвертацию
    for i, m := range c.Mappings {
        r.PathIn = path.Join(repSourcePath, m.SourcePath)
        r.PathOut = path.Join(repDistinationPath, m.DistinationPath)
        r.Convert(ctx, l, cfg)
    }
}
```

### ВАЖНО: NR-версия vs Legacy

Legacy команда `convert` выполняет **полный workflow** (clone → convert → commit). NR-команда `nr-convert` должна выполнять **только конвертацию** без git операций.

**NR-convert workflow:**
1. Валидация входных параметров (BR_SOURCE, BR_TARGET, BR_DIRECTION)
2. Проверка существования source path
3. Инициализация edt.Cli
4. Выполнение конвертации r.Convert()
5. Возврат результата

**Legacy convert workflow:**
1. Клонирование репозитория
2. Переключение веток
3. Загрузка конфигурации конвертора
4. Конвертация
5. Коммит результатов

### Паттерн реализации (из Story 4-4)

**Handler structure:**
```go
// internal/command/handlers/converthandler/handler.go
package converthandler

import (
    "context"
    "fmt"
    "io"
    "log/slog"
    "os"
    "time"

    "github.com/Kargones/apk-ci/internal/command"
    "github.com/Kargones/apk-ci/internal/config"
    "github.com/Kargones/apk-ci/internal/constants"
    "github.com/Kargones/apk-ci/internal/entity/one/edt"
    "github.com/Kargones/apk-ci/internal/pkg/output"
    "github.com/Kargones/apk-ci/internal/pkg/tracing"
)

// Compile-time interface check
var _ command.Handler = (*ConvertHandler)(nil)

func init() {
    command.RegisterWithAlias(&ConvertHandler{}, constants.ActConvert)
}

// Допустимые направления конвертации
const (
    DirectionEdt2xml = "edt2xml"
    DirectionXml2edt = "xml2edt"
)

// ConvertData содержит данные ответа о конвертации.
type ConvertData struct {
    // StateChanged — изменилось ли состояние системы
    StateChanged bool `json:"state_changed"`
    // SourcePath — путь к исходным данным
    SourcePath string `json:"source_path"`
    // TargetPath — путь к результату
    TargetPath string `json:"target_path"`
    // Direction — направление конвертации (edt2xml/xml2edt)
    Direction string `json:"direction"`
    // ToolUsed — использованный инструмент (1cedtcli)
    ToolUsed string `json:"tool_used"`
    // DurationMs — длительность операции в миллисекундах
    DurationMs int64 `json:"duration_ms"`
}

// writeText выводит результат в человекочитаемом формате.
func (d *ConvertData) writeText(w io.Writer) error {
    // Реализовать вывод
}

// Converter — интерфейс для конвертации (для тестируемости).
type Converter interface {
    Convert(ctx *context.Context, l *slog.Logger, cfg *config.Config, direction, pathIn, pathOut string) error
}

// ConvertHandler обрабатывает команду nr-convert.
type ConvertHandler struct {
    converter Converter
}

// Name возвращает имя команды.
func (h *ConvertHandler) Name() string {
    return constants.ActNRConvert
}

// Description возвращает описание команды для help.
func (h *ConvertHandler) Description() string {
    return "Конвертация между форматами EDT и XML"
}

// Execute выполняет команду.
func (h *ConvertHandler) Execute(ctx context.Context, cfg *config.Config) error {
    start := time.Now()
    traceID := tracing.TraceIDFromContext(ctx)
    if traceID == "" {
        traceID = tracing.GenerateTraceID()
    }
    format := os.Getenv("BR_OUTPUT_FORMAT")
    log := slog.Default().With(
        slog.String("trace_id", traceID),
        slog.String("command", constants.ActNRConvert),
    )

    // Progress: validating (AC-11)
    log.Info("validating: проверка параметров")

    // Валидация
    source := os.Getenv("BR_SOURCE")
    if source == "" {
        return h.writeError(format, traceID, start, "CONFIG.SOURCE_MISSING",
            "Не указан путь к исходным данным (BR_SOURCE)")
    }

    target := os.Getenv("BR_TARGET")
    if target == "" {
        return h.writeError(format, traceID, start, "CONFIG.TARGET_MISSING",
            "Не указан путь к результату (BR_TARGET)")
    }

    direction := os.Getenv("BR_DIRECTION")
    if direction == "" {
        return h.writeError(format, traceID, start, "CONFIG.DIRECTION_MISSING",
            "Не указано направление конвертации (BR_DIRECTION)")
    }

    if direction != DirectionEdt2xml && direction != DirectionXml2edt {
        return h.writeError(format, traceID, start, "CONFIG.DIRECTION_INVALID",
            fmt.Sprintf("Недопустимое направление '%s', ожидается: %s или %s",
                direction, DirectionEdt2xml, DirectionXml2edt))
    }

    // Проверка существования source path
    if _, err := os.Stat(source); os.IsNotExist(err) {
        return h.writeError(format, traceID, start, "ERR_SOURCE_NOT_FOUND",
            fmt.Sprintf("Исходный путь не существует: %s", source))
    }

    log = log.With(
        slog.String("source", source),
        slog.String("target", target),
        slog.String("direction", direction),
    )

    // Progress: preparing (AC-11)
    log.Info("preparing: подготовка к конвертации")

    // Progress: converting (AC-11)
    log.Info("converting: выполнение конвертации")

    // ... конвертация через edt.Cli или mock

    // Progress: completing (AC-11)
    log.Info("completing: завершение операции")
}
```

### Переменные окружения

| Переменная | Описание | Обязательная |
|------------|----------|--------------|
| `BR_COMMAND` | Имя команды: `nr-convert` или `convert` | Да |
| `BR_SOURCE` | Путь к исходным данным | Да |
| `BR_TARGET` | Путь к результату | Да |
| `BR_DIRECTION` | Направление: `edt2xml` или `xml2edt` | Да |
| `BR_OUTPUT_FORMAT` | Формат вывода: `json` или пусто (text) | Нет |

### Константы (добавить в constants.go)

```go
// ActNRConvert — NR-команда конвертации форматов EDT/XML
ActNRConvert = "nr-convert"
```

### Project Structure Notes

```
internal/command/handlers/
├── converthandler/           # СОЗДАТЬ
│   ├── handler.go            # ConvertHandler
│   └── handler_test.go       # Тесты
├── createstoreshandler/      # ОБРАЗЕЦ для копирования паттерна
├── storebindhandler/
├── store2dbhandler/
└── ...

internal/constants/
└── constants.go              # Добавить ActNRConvert

internal/entity/one/edt/
└── edt.go                    # Существует — переиспользуем Cli.Convert()
```

### Файлы на создание

| Файл | Описание |
|------|----------|
| `internal/command/handlers/converthandler/handler.go` | ConvertHandler |
| `internal/command/handlers/converthandler/handler_test.go` | Тесты |

### Файлы на изменение

| Файл | Изменение |
|------|-----------|
| `internal/constants/constants.go` | Добавить `ActNRConvert` |
| `cmd/apk-ci/main.go` | Добавить blank import converthandler |

### Файлы НЕ ТРОГАТЬ

- `internal/app/app.go` — legacy оркестрация (Convert)
- `internal/entity/one/edt/edt.go` — бизнес-логика (Cli, Convert)
- `internal/entity/one/convert/convert.go` — другой convert модуль (не путать!)
- Существующие handlers (createstoreshandler, storebindhandler) — только как образец

### Что НЕ делать

- НЕ переписывать логику edt.Cli.Convert — она работает
- НЕ добавлять git операции (clone, switch, commit) — это для legacy
- НЕ добавлять загрузку конфигурации конвертора из JSON — это для legacy
- НЕ путать с `internal/entity/one/convert/` — это другой модуль (store operations)

### Security Considerations

- **Пути** — валидировать source и target на path traversal
- **Timeout** — использовать context.WithTimeout для длительных операций
- **Workspace** — создавать временную рабочую область EDT в TmpDir

### Error Codes

| Код | Описание |
|-----|----------|
| `CONFIG.SOURCE_MISSING` | Не указан BR_SOURCE |
| `CONFIG.TARGET_MISSING` | Не указан BR_TARGET |
| `CONFIG.DIRECTION_MISSING` | Не указан BR_DIRECTION |
| `CONFIG.DIRECTION_INVALID` | Недопустимое значение BR_DIRECTION |
| `ERR_SOURCE_NOT_FOUND` | Source path не существует |
| `ERR_CONVERT` | Ошибка конвертации |

### References

- [Source: _bmad-output/project-planning-artifacts/epics/epic-4-config-sync.md#Story 4.5]
- [Source: _bmad-output/project-planning-artifacts/architecture.md#Pattern: Command Registry with Self-Registration]
- [Source: _bmad-output/project-planning-artifacts/architecture.md#Pattern: NR-Migration Bridge]
- [Source: internal/app/app.go#Convert — legacy оркестрация]
- [Source: internal/entity/one/edt/edt.go#Cli — EDT CLI wrapper]
- [Source: internal/entity/one/edt/edt.go#Convert — конвертация]
- [Source: internal/command/handlers/createstoreshandler/handler.go — образец паттерна реализации]

### Git Intelligence

Последние коммиты Epic 4:
- `ed252f0 fix(code-review): resolve Story 4-4 adversarial review issues`
- `101ef78 feat(onec): implement nr-create-stores command (Story 4.4)`
- `0e8f6eb feat(onec): implement nr-storebind command (Story 4.3)`
- `a0229af feat(onec): implement nr-store2db command (Story 4.2)`

**Паттерны из git:**
- Commit convention: `feat(onec): description` на английском
- Code review исправления идут отдельными коммитами
- Тесты добавляются вместе с кодом

### Previous Story Intelligence (Story 4-4)

**Ключевые паттерны из Story 4.4:**
- Interface pattern для тестируемости (StoreCreator, TempDbCreator → Converter)
- Compile-time interface check: `var _ command.Handler = (*Handler)(nil)`
- Progress logging: validating → creating_temp_db → creating_main_store (адаптировать для convert)
- DurationMs в data struct
- omitempty для optional fields
- writeText() с human-readable summary
- production.go с lazy-loading production реализаций

**Критические точки:**
- Валидация входных параметров в начале Execute
- Обработка ошибок с типизированными кодами
- writeError для JSON НЕ выводит в stdout для text формата (main.go логирует)

### Технологический контекст

- **Go**: 1.25.1 (из go.mod)
- **1cedtcli**: `ring edt workspace export/import` (EDT CLI)
- **EDT directions**: `edt2xml` (EDT→XML), `xml2edt` (XML→EDT)
- **Runner**: `internal/util/runner/` — обёртка над exec.Command
- **OutputWriter**: `internal/pkg/output/` — JSON/Text форматирование

### EDT CLI Commands

**edt2xml (EDT → XML):**
```bash
ring edt workspace export --project /path/to/edt --configuration-files /path/to/xml
```

**xml2edt (XML → EDT):**
```bash
ring edt workspace import --project /path/to/edt --configuration-files /path/to/xml
```

**Из edt.go:**
```go
const (
    XML2edt string = "xml2edt"
    Edt2xml string = "edt2xml"
    formatXML string = "xml"
    formatEDT string = "edt"
)
```

### Implementation Tips

1. **Начать с constants.go** — добавить ActNRConvert
2. **Скопировать createstoreshandler как базу** — структура идентична
3. **Создать интерфейс Converter** — для тестируемости (абстрагирует edt.Cli)
4. **Мокировать edt.Cli** — unit-тесты без реальных EDT операций
5. **Progress logging детальный** — validating → preparing → converting → completing
6. **Валидация direction** — только `edt2xml` или `xml2edt`
7. **Проверка source path** — os.Stat перед конвертацией
8. **Не забыть blank import в main.go** — иначе init() не вызовется

### Особенности Legacy-реализации

**Инициализация Cli:**
```go
func (r *Cli) Init(cfg *config.Config) {
    r.CliPath = cfg.AppConfig.Paths.Bin1cedtcli
}
```

**Конвертация через Cli:**
```go
func (r *Cli) Convert(ctx *context.Context, l *slog.Logger, cfg *config.Config) {
    run := runner.Runner{}
    run.WorkDir = cfg.WorkDir
    run.TmpDir = cfg.TmpDir
    run.RunString = r.CliPath
    run.Params = []string{
        "edt", "workspace", r.Direction,
        "--workspace", r.WorkSpace,
        "--project", r.PathIn,
        "--configuration-files", r.PathOut,
    }
    // ...
}
```

### Отличия NR-версии от Legacy

| Аспект | Legacy (convert) | NR (nr-convert) |
|--------|------------------|-----------------|
| Git operations | Да (clone, switch) | Нет |
| Config loading | Из JSON файла | Из env vars |
| Multiple mappings | Да (через Mappings) | Нет (один source→target) |
| Commit results | Да | Нет |
| Output | Logs только | JSON/Text structured |
| Входные данные | cfg + convert.json | BR_SOURCE, BR_TARGET, BR_DIRECTION |

## Dev Agent Record

### Agent Model Used

Claude Opus 4.5 (claude-opus-4-5-20251101)

### Debug Log References

### Completion Notes List

- ✅ Реализован ConvertHandler с полным TDD-циклом
- ✅ Добавлена константа ActNRConvert в constants.go
- ✅ Созданы handler.go, production.go, handler_test.go
- ✅ Зарегистрирован deprecated alias "convert" → "nr-convert"
- ✅ 17 unit-тестов покрывают все сценарии (success, error, progress)
- ✅ Все тесты internal/* проходят (включая converthandler)
- ✅ Линтер проходит (предупреждения dupl в тестах — ожидаемо)
- ✅ Legacy команда продолжает работать через DeprecatedBridge

### Code Review Fixes (2026-02-04)

- ✅ **H-1 fix**: Временная директория workspace теперь удаляется через defer в production.go
- ✅ **H-2 fix**: Добавлена валидация path traversal для source и target через validatePath()
- ✅ **M-1 fix**: Добавлено предупреждение при перезаписи существующей target директории
- ✅ **M-2 fix**: Добавлен timeout для операции конвертации через context.WithTimeout
- ✅ **M-3 fix**: Инструмент теперь читается из cfg.ImplementationsConfig.ConfigExport
- ✅ **L-1 fix**: Убрано дублирование полей в финальном логе (source, target, direction уже в .With())
- ✅ Добавлены новые тесты: validatePath, path traversal, timeout, tool from config
- ✅ Все 23 теста проходят

### File List

**Созданы:**
- internal/command/handlers/converthandler/handler.go
- internal/command/handlers/converthandler/production.go
- internal/command/handlers/converthandler/handler_test.go

**Изменены:**
- internal/constants/constants.go (добавлен ActNRConvert)
- cmd/apk-ci/main.go (добавлен blank import converthandler)
- cmd/apk-ci/main_test.go (убран convert из списка legacy)

## Change Log

- 2026-02-04: Story создана с комплексным контекстом на основе Epic 4, архитектуры, и предыдущих stories
- 2026-02-04: Реализация nr-convert завершена — все tasks выполнены, тесты проходят
- 2026-02-04: Code review выполнен — найдено 7 issues (2 HIGH, 3 MEDIUM, 2 LOW), все исправлены
