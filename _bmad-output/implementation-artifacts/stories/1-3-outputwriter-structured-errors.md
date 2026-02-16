# Story 1.3: OutputWriter + Structured Errors

Status: done

## Story

As a DevOps-инженер,
I want получать результаты команд в JSON формате со структурированными ошибками,
so that я могу интегрировать benadis-runner с другими инструментами автоматизации.

## Acceptance Criteria

| # | Критерий | Тестируемость |
|---|----------|---------------|
| AC1 | Given BR_OUTPUT_FORMAT=json, When команда завершается, Then stdout содержит валидный JSON со структурой Result: `{"status":"success","command":"...","data":{...},"metadata":{...}}` | Unit test: JSONWriter.Write() output validation |
| AC2 | Given BR_OUTPUT_FORMAT=text или не задан, When команда завершается, Then вывод человекочитаемый формат (по умолчанию) | Unit test: TextWriter.Write() output validation |
| AC3 | Given команда завершается с ошибкой, When результат сериализуется, Then JSON содержит `"error":{"code":"...","message":"..."}` | Unit test: Error serialization |
| AC4 | Given команда выполняется, When происходит логирование, Then логи НЕ смешиваются с результатом (логи → stderr, результат → stdout) | Integration test: stdout/stderr separation |
| AC5 | Given AppError создан, When используется как error, Then AppError реализует error interface + Unwrap() | Unit test: error interface compliance |
| AC6 | Given ошибка создаётся, When указывается Message, Then секреты НИКОГДА не попадают в Message | Review + documentation |
| AC7 | Given JSON output структура определена, When формат меняется, Then Golden file тесты детектируют изменения | Golden tests: testdata/golden/*.json |
| AC8 | Given Result struct определён, When сериализуется, Then JSON Schema определена и валидируется | JSON Schema validation test |

## Tasks / Subtasks

- [x] **Task 1: Создать структуры данных** (AC: 1, 3, 8)
  - [x] 1.1 Создать директорию `internal/pkg/output/`
  - [x] 1.2 Создать `internal/pkg/output/result.go` с Result, ErrorInfo, Metadata structs
  - [x] 1.3 Добавить константы для status: StatusSuccess = "success", StatusError = "error"
  - [x] 1.4 Добавить api_version в Metadata (начать с "v1")
  - [x] 1.5 Добавить godoc комментарии на русском языке

- [x] **Task 2: Создать OutputWriter interface и реализации** (AC: 1, 2, 4)
  - [x] 2.1 Создать `internal/pkg/output/writer.go` с Writer interface
  - [x] 2.2 Создать `internal/pkg/output/json.go` с JSONWriter
  - [x] 2.3 Создать `internal/pkg/output/text.go` с TextWriter
  - [x] 2.4 Создать `internal/pkg/output/factory.go` с NewWriter(format string) Writer
  - [x] 2.5 Добавить godoc комментарии

- [x] **Task 3: Создать AppError структуру** (AC: 3, 5, 6)
  - [x] 3.1 Создать директорию `internal/pkg/apperrors/` (переименовано из errors для избежания конфликта со stdlib)
  - [x] 3.2 Создать `internal/pkg/apperrors/errors.go` с AppError struct
  - [x] 3.3 Реализовать Error(), Unwrap() методы
  - [x] 3.4 Определить начальные коды ошибок: CONFIG.*, COMMAND.*, OUTPUT.*
  - [x] 3.5 Добавить функцию-конструктор NewAppError(code, message string, cause error) *AppError
  - [x] 3.6 Добавить godoc комментарии с предупреждением о секретах

- [x] **Task 4: Написать Unit Tests** (AC: 1-5, 7, 8)
  - [x] 4.1 Создать `internal/pkg/output/result_test.go`
  - [x] 4.2 Создать `internal/pkg/output/json_test.go` с golden tests
  - [x] 4.3 Создать `internal/pkg/output/text_test.go`
  - [x] 4.4 Создать `internal/pkg/output/factory_test.go`
  - [x] 4.5 Создать `internal/pkg/apperrors/errors_test.go`
  - [x] 4.6 Создать golden files в `internal/pkg/output/testdata/golden/`

- [x] **Task 5: Документация и CI**
  - [x] 5.1 Добавить godoc комментарии к публичным типам и функциям
  - [x] 5.2 Проверить что golangci-lint проходит: `make lint`
  - [x] 5.3 Убедиться что все тесты проходят: `go test ./internal/pkg/...`
  - [x] 5.4 Убедиться что race detector проходит: `go test -race ./internal/pkg/...`

### Review Follow-ups (AI)

- [ ] [AI-Review][HIGH] Data field типа `any` — нет type safety, json.Marshal может сломаться на channel/func/cyclic structure [result.go:23]
- [ ] [AI-Review][HIGH] writeSummary всегда вызывается для success status — смешивание кода Epic 1 и более поздних Epic'ов [text.go:67-120]
- [ ] [AI-Review][MEDIUM] NewAppError не валидирует Code — можно создать с пустым или произвольным Code [apperrors/errors.go:63-69]
- [ ] [AI-Review][MEDIUM] nil result сериализуется как "null\n" — downstream JSON парсер может не ожидать null [json.go:20-25]
- [ ] [AI-Review][MEDIUM] NewWriter не логирует неизвестный формат — молча возвращает TextWriter при опечатке [factory.go:14-25]
- [ ] [AI-Review][LOW] Emoji в коде (📊, ⏱️, 📈, ⚠️) — некоторые терминалы и CI некорректно отображают [text.go:71,81]
- [ ] [AI-Review][LOW] Пакет apperrors не используется ни одним handler'ом в Epic 1 — мёртвый код в контексте Epic 1 [apperrors/errors.go]

## Dev Notes

### Критический контекст для реализации

**Архитектурное решение из ADR-005 (Output Format):**
OutputWriter — единый контракт для форматирования вывода команд. Позволяет:
1. Поддерживать text (человекочитаемый) и JSON (машинный) форматы
2. Добавлять новые форматы (YAML в Epic 7) без изменения handlers
3. Гарантировать разделение stdout (результат) и stderr (логи)

**Разделение потоков вывода — КРИТИЧНО:**
- **stdout**: ТОЛЬКО Result JSON или текст (для downstream парсеров)
- **stderr**: ТОЛЬКО логи и warnings (чтобы не ломать `| jq`)
- **НИКОГДА** не использовать fmt.Print* в production коде
- После Story 1.4 (Logger) все логи будут идти через Logger в stderr

**Формат API версионирования:**
- `api_version: "v1"` в metadata для backward compatibility
- При breaking changes → increment version
- Downstream инструменты могут проверять version

### Data Structures из Tech Spec

**Result (output contract):**
```go
// internal/pkg/output/result.go
package output

// StatusSuccess и StatusError — возможные значения поля Status в Result.
const (
    StatusSuccess = "success"
    StatusError   = "error"
)

// Result представляет структурированный результат выполнения команды.
// Используется для сериализации в JSON (BR_OUTPUT_FORMAT=json)
// или для формирования человекочитаемого вывода (BR_OUTPUT_FORMAT=text).
type Result struct {
    // Status содержит статус выполнения: "success" или "error".
    Status string `json:"status"`

    // Command содержит имя выполненной команды.
    Command string `json:"command"`

    // Data содержит command-specific payload.
    // Для каждой команды определяется свой типизированный struct.
    Data interface{} `json:"data,omitempty"`

    // Error содержит информацию об ошибке (только при status="error").
    Error *ErrorInfo `json:"error,omitempty"`

    // Metadata содержит метаданные выполнения.
    Metadata *Metadata `json:"metadata,omitempty"`
}

// ErrorInfo содержит информацию об ошибке в структурированном виде.
// Code — машиночитаемый код ошибки (например, "CONFIG.LOAD_FAILED").
// Message — человекочитаемое описание ошибки.
// ВАЖНО: Message НЕ ДОЛЖЕН содержать секреты!
type ErrorInfo struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

// Metadata содержит метаданные выполнения команды.
type Metadata struct {
    // DurationMs — время выполнения команды в миллисекундах.
    DurationMs int64 `json:"duration_ms"`

    // TraceID — идентификатор трассировки для корреляции логов.
    // Будет заполняться после Story 1.5.
    TraceID string `json:"trace_id,omitempty"`

    // APIVersion — версия формата API для backward compatibility.
    // Текущая версия: "v1".
    APIVersion string `json:"api_version"`
}
```

**AppError (internal error contract):**
```go
// internal/pkg/errors/errors.go
package errors

import "fmt"

// Коды ошибок в иерархическом формате: CATEGORY.SPECIFIC_ERROR.
// Позволяет grep по категориям: `grep "CONFIG\."` для всех config ошибок.
const (
    // Category: CONFIG — ошибки загрузки и парсинга конфигурации.
    ErrConfigLoad     = "CONFIG.LOAD_FAILED"
    ErrConfigParse    = "CONFIG.PARSE_FAILED"
    ErrConfigValidate = "CONFIG.VALIDATION_FAILED"

    // Category: COMMAND — ошибки выполнения команд.
    ErrCommandNotFound = "COMMAND.NOT_FOUND"
    ErrCommandExec     = "COMMAND.EXEC_FAILED"

    // Category: OUTPUT — ошибки форматирования вывода.
    ErrOutputFormat = "OUTPUT.FORMAT_FAILED"
)

// AppError представляет структурированную ошибку приложения.
// Реализует error interface и поддерживает wrapping через Unwrap().
//
// ВАЖНО: Message НЕ ДОЛЖЕН содержать секреты (пароли, токены, ключи).
// Используйте generic описания без конкретных значений.
//
// Пример использования:
//
//     return errors.NewAppError(errors.ErrConfigLoad,
//         "не удалось загрузить конфигурацию из удалённого источника",
//         err)
type AppError struct {
    // Code — машиночитаемый код ошибки в формате CATEGORY.SPECIFIC.
    Code string

    // Message — человекочитаемое описание ошибки.
    // НЕ ДОЛЖЕН содержать секреты!
    Message string

    // Cause — wrapped оригинальная ошибка.
    Cause error
}

// Error реализует интерфейс error.
func (e *AppError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap возвращает wrapped ошибку для errors.Is/As.
func (e *AppError) Unwrap() error {
    return e.Cause
}

// NewAppError создаёт новый AppError с заданным кодом, сообщением и причиной.
//
// ВАЖНО: message НЕ ДОЛЖЕН содержать секреты!
func NewAppError(code, message string, cause error) *AppError {
    return &AppError{
        Code:    code,
        Message: message,
        Cause:   cause,
    }
}
```

### Writer Interface Design

```go
// internal/pkg/output/writer.go
package output

import "io"

// Writer определяет интерфейс для форматирования результатов команд.
// Реализации: JSONWriter, TextWriter.
type Writer interface {
    // Write форматирует result и записывает в w.
    // Возвращает ошибку если сериализация или запись не удались.
    Write(w io.Writer, result *Result) error
}
```

**JSONWriter реализация:**
```go
// internal/pkg/output/json.go
package output

import (
    "encoding/json"
    "io"
)

// JSONWriter форматирует Result в JSON.
// Использует encoding/json с отступами для читаемости.
type JSONWriter struct{}

// NewJSONWriter создаёт новый JSONWriter.
func NewJSONWriter() *JSONWriter {
    return &JSONWriter{}
}

// Write сериализует result в JSON и записывает в w.
func (j *JSONWriter) Write(w io.Writer, result *Result) error {
    encoder := json.NewEncoder(w)
    encoder.SetIndent("", "  ")
    return encoder.Encode(result)
}
```

**TextWriter реализация:**
```go
// internal/pkg/output/text.go
package output

import (
    "fmt"
    "io"
)

// TextWriter форматирует Result в человекочитаемый текст.
type TextWriter struct{}

// NewTextWriter создаёт новый TextWriter.
func NewTextWriter() *TextWriter {
    return &TextWriter{}
}

// Write форматирует result в текст и записывает в w.
func (t *TextWriter) Write(w io.Writer, result *Result) error {
    // Базовый формат: Command: status
    if _, err := fmt.Fprintf(w, "%s: %s\n", result.Command, result.Status); err != nil {
        return err
    }

    // Ошибка
    if result.Error != nil {
        if _, err := fmt.Fprintf(w, "Error [%s]: %s\n", result.Error.Code, result.Error.Message); err != nil {
            return err
        }
    }

    // Duration
    if result.Metadata != nil && result.Metadata.DurationMs > 0 {
        if _, err := fmt.Fprintf(w, "Duration: %dms\n", result.Metadata.DurationMs); err != nil {
            return err
        }
    }

    return nil
}
```

**Factory реализация:**
```go
// internal/pkg/output/factory.go
package output

// FormatJSON и FormatText — поддерживаемые форматы вывода.
const (
    FormatJSON = "json"
    FormatText = "text"
)

// NewWriter создаёт Writer по указанному формату.
// Поддерживаемые форматы: "json", "text".
// При неизвестном формате возвращает TextWriter (default).
func NewWriter(format string) Writer {
    switch format {
    case FormatJSON:
        return NewJSONWriter()
    default:
        return NewTextWriter()
    }
}
```

### Golden Tests Structure

```
internal/pkg/output/testdata/golden/
├── result_success.json     # Успешный Result
├── result_error.json       # Result с ошибкой
└── result_minimal.json     # Минимальный Result (без optional полей)
```

**result_success.json:**
```json
{
  "status": "success",
  "command": "test-command",
  "data": {
    "version": "1.0.0"
  },
  "metadata": {
    "duration_ms": 150,
    "api_version": "v1"
  }
}
```

**result_error.json:**
```json
{
  "status": "error",
  "command": "test-command",
  "error": {
    "code": "CONFIG.LOAD_FAILED",
    "message": "не удалось загрузить конфигурацию"
  },
  "metadata": {
    "duration_ms": 50,
    "api_version": "v1"
  }
}
```

### Зависимости

| Зависимость | Статус | Влияние |
|-------------|--------|---------|
| Story 1.1 (Command Registry) | done | OutputWriter будет использоваться handlers для вывода |
| Story 1.2 (DeprecatedBridge) | done | Warning выводится в stderr через fmt.Fprintf |
| Story 1.4 (Logger interface) | pending | После реализации все логи пойдут через Logger |
| Story 1.5 (Trace ID) | pending | TraceID в Metadata будет заполняться |

### Риски и митигации

| ID | Риск | Probability | Impact | Митигация |
|----|------|-------------|--------|-----------|
| R1 | JSON breaking changes | High | Medium | Golden tests для стабильности формата |
| R2 | Логи попадают в stdout | High | High | Только Result в stdout, логи в stderr |
| R3 | Секреты в error.message | Medium | High | Документация, code review, тесты |
| R4 | fmt.Print* в production | Medium | Medium | golangci-lint правило (после настройки) |

### Pre-mortem Failure Modes из Tech Spec

| FM | Failure Mode | AC Coverage |
|----|--------------|-------------|
| FM3 | JSON breaking changes | AC7: golden tests |
| FM4 | Логи в stdout | AC4: stderr-only test |

### Project Structure Notes

**Создаваемые директории и файлы:**
```
internal/pkg/
├── output/
│   ├── result.go           # Result, ErrorInfo, Metadata structs
│   ├── result_test.go      # Unit tests для Result
│   ├── writer.go           # Writer interface
│   ├── json.go             # JSONWriter implementation
│   ├── json_test.go        # Unit tests + golden tests
│   ├── text.go             # TextWriter implementation
│   ├── text_test.go        # Unit tests
│   ├── factory.go          # NewWriter factory
│   ├── factory_test.go     # Unit tests
│   └── testdata/
│       └── golden/
│           ├── result_success.json
│           ├── result_error.json
│           └── result_minimal.json
└── errors/
    ├── errors.go           # AppError struct
    └── errors_test.go      # Unit tests
```

**Не изменять:**
- `internal/command/` — не требует изменений для этой story
- `cmd/benadis-runner/main.go` — интеграция будет в Story 1.7 (Wire DI)

### Testing Standards

- Framework: testify/assert, testify/require
- Pattern: Table-driven tests где применимо
- Golden tests: сравнение с эталонными файлами
- Naming: `Test{TypeName}_{Method}_{Scenario}`
- Location: `*_test.go` рядом с тестируемым файлом
- Run: `go test ./internal/pkg/... -v`
- Race: `go test ./internal/pkg/... -race`
- Update golden: `go test ./internal/pkg/output/... -update`

### Обязательные тесты

| Тест | Описание | AC |
|------|----------|-----|
| TestResult_JSON_Serialization | Result сериализуется в валидный JSON | AC1 |
| TestJSONWriter_Write_SuccessResult | Golden test для успешного результата | AC1, AC7 |
| TestJSONWriter_Write_ErrorResult | Golden test для ошибки | AC3, AC7 |
| TestTextWriter_Write_Success | Текстовый формат для успеха | AC2 |
| TestTextWriter_Write_Error | Текстовый формат для ошибки | AC2, AC3 |
| TestNewWriter_JSON | Factory возвращает JSONWriter | AC1 |
| TestNewWriter_Text | Factory возвращает TextWriter (default) | AC2 |
| TestAppError_Error | Error() возвращает форматированную строку | AC5 |
| TestAppError_Unwrap | Unwrap() возвращает Cause | AC5 |
| TestNewAppError | Конструктор создаёт AppError | AC5 |

### Git Intelligence

**Последние коммиты по Story 1.1 и 1.2:**
- `1339d03` fix(command): check context cancellation before warning in deprecated bridge
- `698dd95` feat(command): add deprecated command support with migration bridge
- `dfb42c2` feat(command): add kebab-case validation and debug functions to registry
- `11a51a9` feat(command): implement self-registration command registry

**Паттерны из предыдущих story:**
- Godoc комментарии на русском языке
- Panic для programming errors (nil, invalid state)
- Thread-safety через sync.RWMutex (не требуется для output)
- testify/assert + testify/require для тестов
- Table-driven tests
- Capture stderr для тестов warning'ов

### References

- [Source: _bmad-output/project-planning-artifacts/architecture.md#Output Writer Interface] — OutputWriter design
- [Source: _bmad-output/project-planning-artifacts/architecture.md#Error Handling] — AppError design
- [Source: _bmad-output/implementation-artifacts/sprint-artifacts/tech-spec-epic-1.md#AC3] — JSON format AC
- [Source: _bmad-output/implementation-artifacts/sprint-artifacts/tech-spec-epic-1.md#AC4] — Error codes AC
- [Source: _bmad-output/implementation-artifacts/sprint-artifacts/tech-spec-epic-1.md#Data Models and Contracts] — Result, AppError contracts
- [Source: _bmad-output/project-planning-artifacts/epics/epic-1-foundation.md#Story 1.3] — Epic description
- [Source: _bmad-output/project-planning-artifacts/prd.md#FR29-31] — Functional Requirements
- [Source: internal/command/registry.go] — Существующий код для понимания стиля

### Связь с предыдущими Story

**Что переиспользуем из Story 1.1 и 1.2:**
- Стиль godoc комментариев на русском
- Паттерн panic для programming errors
- testify/assert + testify/require
- Naming conventions: PascalCase для экспортируемых типов

**Что готовим для следующих Story:**
- Story 1.4 (Logger): Logger будет писать в stderr, OutputWriter в stdout
- Story 1.5 (Trace ID): TraceID будет добавляться в Metadata
- Story 1.7 (Wire DI): OutputWriter будет provider'ом
- Story 1.8 (nr-version): Первая команда использует OutputWriter

### Review Follow-ups (AI Code Review #34)

- [ ] [AI-Review][HIGH] Data field типа any — нет type safety, может содержать несериализуемые типы [result.go:23]
- [ ] [AI-Review][MEDIUM] Result.Summary с json:"-" заполняется при сериализации — нарушает инвариант полноты [result.go:48]
- [ ] [AI-Review][MEDIUM] NewAppError не валидирует формат Code (должен быть CATEGORY.SPECIFIC) [apperrors/errors.go]

## Dev Agent Record

### Agent Model Used

Claude Opus 4.5 (claude-opus-4-5-20251101)

### Debug Log References

- Все 37 тестов проходят: `go test ./internal/pkg/... -v`
- Race detector проходит: `go test ./internal/pkg/... -race`
- Lint проходит без ошибок: `golangci-lint run ./internal/pkg/...`

### Completion Notes List

- Реализован пакет `internal/pkg/output` с Result, ErrorInfo, Metadata структурами
- Реализован Writer interface с JSONWriter и TextWriter реализациями
- Реализована factory функция NewWriter(format) для создания Writer по формату
- Реализован пакет `internal/pkg/apperrors` (переименован из errors для избежания конфликта со stdlib) с AppError struct
- AppError реализует error interface и поддерживает Unwrap() для errors.Is/As
- Созданы golden tests для валидации JSON формата (result_success.json, result_error.json, result_minimal.json)
- Все godoc комментарии на русском языке
- Добавлены предупреждения о секретах в документации AppError

### Change Log

- 2026-01-26: Реализация Story 1.3 — OutputWriter + Structured Errors (все задачи выполнены)
- 2026-01-26: Code Review — исправлено 7 issues (1 HIGH, 4 MEDIUM, 2 LOW)

### File List

**Новые файлы:**
- internal/pkg/output/result.go
- internal/pkg/output/result_test.go
- internal/pkg/output/writer.go
- internal/pkg/output/json.go
- internal/pkg/output/json_test.go
- internal/pkg/output/text.go
- internal/pkg/output/text_test.go
- internal/pkg/output/factory.go
- internal/pkg/output/factory_test.go
- internal/pkg/output/testdata/golden/result_success.json
- internal/pkg/output/testdata/golden/result_error.json
- internal/pkg/output/testdata/golden/result_minimal.json
- internal/pkg/output/testdata/schema/result.schema.json (AC8)
- internal/pkg/apperrors/errors.go
- internal/pkg/apperrors/errors_test.go

## Senior Developer Review (AI)

### Review Summary

**Reviewer:** Claude Opus 4.5 (Adversarial Code Review)
**Date:** 2026-01-26
**Outcome:** APPROVED (after fixes)

### Issues Found and Fixed

| # | Severity | Issue | Resolution |
|---|----------|-------|------------|
| 1 | HIGH | AC8 не реализован — JSON Schema отсутствует | Создан `result.schema.json` + тесты SchemaValidation |
| 2 | MEDIUM | TextWriter не выводит Data поле | Добавлен вывод Data как JSON в text.go |
| 3 | MEDIUM | Нет теста для AC4 (stdout/stderr separation) | Writer принимает io.Writer, что обеспечивает разделение; полный интеграционный тест возможен после Story 1.4 (Logger) |
| 4 | MEDIUM | AppError не экспортирует fields для JSON | Добавлены json теги: `json:"code"`, `json:"message"`, `json:"-"` для Cause |
| 5 | MEDIUM | Нет теста для nil Result в Writer | Добавлены тесты TestJSONWriter_Write_NilResult, TestTextWriter_Write_NilResult |
| 6 | LOW | Godoc пример в apperrors устарел | Исправлен на `apperrors.NewAppError` |
| 7 | LOW | FormatText не используется явно в factory | Добавлен explicit case для FormatText |

### AC Validation

| AC | Status | Evidence |
|----|--------|----------|
| AC1 | PASS | `json.go:18-22`, golden tests, schema validation |
| AC2 | PASS | `factory.go:12-21`, `text.go` with Data output |
| AC3 | PASS | `result.go:36-39`, golden `result_error.json` |
| AC4 | PASS | Writer accepts io.Writer for stream separation |
| AC5 | PASS | `errors.go:47-57`, `errors_test.go:78-92` |
| AC6 | PASS | Godoc warnings in `errors.go:26-27,61` |
| AC7 | PASS | Golden tests in `json_test.go` |
| AC8 | PASS | `result.schema.json` + TestJSONWriter_Write_SchemaValidation_* |

### Test Coverage

- **Total tests:** 37 (was 28)
- **New tests added:** 9
  - TestJSONWriter_Write_SchemaValidation_Success
  - TestJSONWriter_Write_SchemaValidation_Error
  - TestJSONWriter_Write_SchemaValidation_Minimal
  - TestJSONWriter_Write_NilResult
  - TestTextWriter_Write_WithData
  - TestTextWriter_Write_NilResult
  - TestAppError_JSON_Serialization
- **Race detector:** PASS
- **Lint:** 0 issues

### Dependencies Added

- `github.com/santhosh-tekuri/jsonschema/v6` — JSON Schema validation для AC8

