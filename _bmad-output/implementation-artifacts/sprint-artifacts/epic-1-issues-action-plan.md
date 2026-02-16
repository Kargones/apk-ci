# Epic 1: Architectural Foundation — План решения выявленных проблем

**Дата аудита:** 2026-01-27
**Проверяющие:** Charlie (Senior Dev), Dana (QA Engineer), Claude Opus 4.5
**Охват:** Все 9 историй Epic 1 (internal/command, internal/pkg/output, internal/pkg/apperrors, internal/pkg/logging, internal/pkg/tracing, internal/config, internal/di, internal/command/handlers/*, cmd/benadis-runner/main.go)

---

## Сводка

| Severity | Количество |
|----------|-----------|
| CRITICAL | 5 |
| HIGH     | 8 |
| MEDIUM   | 10 |
| LOW      | 8 |
| **Итого** | **31** |

---

## CRITICAL — Необходимо исправить до начала Epic 2

### C1: Wire DI принимает nil Config без ошибки
**Файлы:** `internal/di/providers.go:28`, `internal/di/integration_test.go:149`
**Описание:** `InitializeApp(nil)` создаёт App с `Config: nil`. Downstream код (handlers, legacy commands) обращается к `cfg.InfobaseName` и т.д. — получит panic.
**Решение:**
- [ ] Добавить проверку `if cfg == nil { return nil, errors.New("config is required") }` в `InitializeApp` или в начало `ProvideLogger`
- [ ] Обновить тест `nil config` — ожидать ошибку вместо success
- [ ] Убедиться что wire_gen.go перегенерирован

### C2: CaptureStdout модифицирует глобальный os.Stdout без синхронизации
**Файл:** `internal/pkg/testutil/capture.go:13-30`
**Описание:** При параллельном запуске тестов два вызова CaptureStdout перезапишут os.Stdout друг друга. Тесты станут flaky.
**Решение:**
- [ ] Добавить `sync.Mutex` в testutil для захвата stdout
- [ ] Или рефакторинг handlers для приёма `io.Writer` вместо жёсткого `os.Stdout` (предпочтительно, но больший scope — можно отложить на Epic 2)
- [ ] Пометить тесты использующие CaptureStdout как несовместимые с `t.Parallel()`

### C3: Race condition при регистрации deprecated handler в integration тестах help
**Файл:** `internal/command/handlers/help/integration_test.go:28-36`
**Описание:** `sync.Once` для регистрации deprecated handler в тестах не гарантирует порядок выполнения между тестовыми файлами. Глобальный registry модифицируется без изоляции.
**Решение:**
- [ ] Использовать `TestMain(m *testing.M)` для однократной регистрации fixture до запуска тестов
- [ ] Или добавить `clearRegistry()` + повторную регистрацию в каждом тесте, требующем контролируемого состояния registry

### C4: os.Pipe() ошибка игнорируется в deprecated_test.go
**Файл:** `internal/command/deprecated_test.go:42`
**Описание:** `r, w, _ := os.Pipe()` — при ошибке создания pipe, r и w будут nil → panic.
**Решение:**
- [ ] Заменить на `r, w, err := os.Pipe()` + `require.NoError(t, err)` (по аналогии с testutil/capture.go)

### C5: ImplementationsConfig.Validate() разрешает пустые строки
**Файл:** `internal/config/config.go:307-318`
**Описание:** Validation map содержит `"": true`, что позволяет пройти валидацию с пустым ConfigExport/DBCreate. Но defaults не применяются автоматически — downstream код получит пустую строку вместо "1cv8".
**Решение:**
- [ ] Убрать `"": true` из validation maps
- [ ] Или применять defaults внутри Validate() для пустых значений: `if c.ConfigExport == "" { c.ConfigExport = "1cv8" }`

---

## HIGH — Исправить в ближайшей итерации

### H1: Жёсткая привязка к os.Stdout/os.Stderr в handlers
**Файлы:** `handlers/version/version.go:92,108`, `handlers/help/help.go:98,114`, `command/deprecated.go:86`
**Описание:** Handlers пишут напрямую в os.Stdout/os.Stderr, что затрудняет тестирование и делает невозможным переключение вывода (например, в файл).
**Решение:**
- [ ] Добавить `io.Writer` как параметр Execute или как поле handler struct (через DI)
- [ ] В переходный период — задокументировать ограничение и оставить CaptureStdout как workaround
- [ ] **Приоритет:** Решить при рефакторинге Handler interface в Epic 2

### H2: JSONWriter не отключает HTML escaping
**Файл:** `internal/pkg/output/json.go:20`
**Описание:** `json.Encoder` по умолчанию эскейпит `<`, `>`, `&` в `\u003c` и т.д. Для CLI-инструмента это нежелательно.
**Решение:**
- [ ] Добавить `encoder.SetEscapeHTML(false)` после `encoder.SetIndent("", "  ")`
- [ ] Обновить golden files если они содержат HTML-символы

### H3: Нет тестов Writer при ошибках I/O
**Файлы:** `internal/pkg/output/json_test.go`, `text_test.go`
**Описание:** Все тесты используют `bytes.Buffer` который никогда не возвращает ошибку. Нет тестов для частичной записи, закрытого pipe, disk full.
**Решение:**
- [ ] Создать `failingWriter` mock в testutil
- [ ] Добавить тесты: `TestJSONWriter_Write_IOError`, `TestTextWriter_Write_IOError`

### H4: Нет тестов NewLoggerWithWriter с nil writer
**Файл:** `internal/pkg/logging/factory.go:22-38`
**Описание:** Передача nil writer вызовет panic в slog.NewJSONHandler/NewTextHandler.
**Решение:**
- [ ] Добавить проверку `if w == nil { panic("writer must not be nil") }` в NewLoggerWithWriter
- [ ] Добавить тест `TestNewLoggerWithWriter_NilWriter_Panics`

### H5: SlogAdapter.With() не валидирует чётность аргументов
**Файл:** `internal/pkg/logging/slog.go:39`
**Описание:** `logger.With("key")` (нечётное количество args) молча создаёт `{"key": "{missing}"}` вместо ошибки.
**Решение:**
- [ ] Добавить проверку `if len(args) % 2 != 0` с warning или panic
- [ ] Или задокументировать поведение slog при нечётном количестве args
- [ ] Добавить тест `TestSlogAdapter_With_OddArgs`

### H6: Description() методы с 0% покрытия тестами
**Файлы:** `deprecated.go:53`, `version.go:69`
**Описание:** Методы Description(), IsDeprecated(), NewName() в DeprecatedBridge и Description() в VersionHandler не покрыты прямыми тестами.
**Решение:**
- [ ] Добавить явные тесты: `TestDeprecatedBridge_Description`, `TestDeprecatedBridge_IsDeprecated`, `TestDeprecatedBridge_NewName`
- [ ] Добавить `TestVersionHandler_Description`

### H7: fmt.Fprintf ошибка игнорируется в DeprecatedBridge.Execute()
**Файл:** `internal/command/deprecated.go:86`
**Описание:** `fmt.Fprintf(os.Stderr, ...)` — ошибка записи warning в stderr игнорируется.
**Решение:**
- [ ] Логировать ошибку через Logger (после рефакторинга на Logger в Epic 2)
- [ ] Или: `if _, err := fmt.Fprintf(...); err != nil { return err }`

### H8: Недокументированные и неконсистентные exit codes в main.go
**Файл:** `cmd/benadis-runner/main.go`
**Описание:** Exit codes: 2 (unknown), 5 (config), 6 (convert), 7 (store2db, git2store), 8 (service-mode, dbrestore) — нет единой системы.
**Решение:**
- [ ] Создать константы `ExitCode*` в `internal/constants/constants.go`
- [ ] Задокументировать маппинг exit code → тип ошибки
- [ ] **Приоритет:** При миграции legacy команд в Epic 2+

---

## MEDIUM — Запланировать исправление

### M1: Golden tests сравнивают JSON как строки (byte-exact)
**Файлы:** `output/json_test.go:43-52`, `handlers/help/golden_test.go`, `handlers/version/golden_test.go`
**Описание:** Изменение отступов или порядка полей JSON сломает тесты даже при семантической эквивалентности.
**Решение:** Парсить оба JSON и сравнивать как объекты, или явно задокументировать что byte-exact сравнение — by design для стабильности формата.

### M2: captureStderr в deprecated_test.go дублирует логику testutil
**Файл:** `internal/command/deprecated_test.go:39-56`
**Описание:** Собственная реализация captureStderrWithCleanup, а не переиспользование testutil.
**Решение:** Создать `testutil.CaptureStderr()` по аналогии с CaptureStdout.

### M3: TraceID: collision probability для fallback не документирована
**Файл:** `internal/pkg/tracing/traceid.go:39-46`
**Описание:** Fallback использует timestamp+counter (2^64 уникальных ID) — менее безопасно чем crypto/rand (2^128).
**Решение:** Добавить godoc комментарий о collision probability в fallback режиме.

### M4: Factory NewWriter() молча fallback на TextWriter для неизвестных форматов
**Файл:** `internal/pkg/output/factory.go:12-22`
**Описание:** `NewWriter("jso")` (опечатка) молча вернёт TextWriter вместо ошибки.
**Решение:** Рассмотреть возврат error для неизвестных форматов или warning в stderr.

### M5: AppError.Error() возвращает разные форматы с/без Cause
**Файл:** `internal/pkg/apperrors/errors.go:48-53`
**Описание:** С cause: `"CODE: Message (cause)"`, без: `"CODE: Message"` — неконсистентный формат для парсинга.
**Решение:** Задокументировать формат или унифицировать.

### M6: NopLogger тест проверяет реализацию вместо поведения
**Файл:** `internal/pkg/logging/nop_test.go:54`
**Описание:** `_, ok := childLogger.(*NopLogger)` — type assertion вместо проверки поведения.
**Решение:** Заменить на проверку что child logger не паникует при вызове методов.

### M7: API Version "v1" захардкожена в тестах без константы
**Файлы:** Множество test файлов используют строку `"v1"` напрямую
**Описание:** Нет константы `APIVersionV1` — при смене версии нужен grep по всем файлам.
**Решение:** Использовать `constants.APIVersion` (уже существует) во всех тестах.

### M8: Wire wildcard binding `wire.Struct(new(App), "*")`
**Файл:** `internal/di/wire.go:24`
**Описание:** Wildcard привязка менее явная чем перечисление полей.
**Решение:** Заменить на `wire.Struct(new(App), "Config", "Logger", "OutputWriter", "TraceID")` для явности.

### M9: Makefile test-nr-version зависит от Gitea API
**Файл:** `Makefile:227-233`
**Описание:** Target запускает бинарник который вызывает config.MustLoad() → обращение к Gitea API → fail в CI без сети.
**Решение:** Добавить skip condition или mock config для standalone тестирования.

### M10: Пустая команда → "help" без проверки что help зарегистрирован
**Файл:** `cmd/benadis-runner/main.go:36-39`
**Описание:** Если blank import для help удалён, код молча fallthrough в legacy switch с непонятной ошибкой.
**Решение:** Добавить panic/assert если help handler не найден в registry.

---

## LOW — Минорные улучшения

### L1: parseLevel() в logging/factory.go — приватная, но потенциально полезна
**Решение:** Рассмотреть экспорт как `ParseLevel()` если нужна внешняя валидация.

### L2: JSON Schema не указывает trace_id как optional
**Файл:** `output/testdata/schema/result.schema.json`
**Решение:** Обновить schema — пометить trace_id как optional.

### L3: Godoc для Description() в handler.go слишком минималистичен
**Решение:** Добавить информацию о формате и требованиях к локализации.

### L4: Help текст идёт в stdout, а не stderr (конвенция Unix)
**Решение:** Оставить как есть — help это результат команды, не диагностика.

### L5: Blank imports в main.go — добавить инструкцию для разработчиков
**Решение:** Расширить комментарий: "New NR-commands must be added here".

### L6: io.Copy ошибки игнорируются в deprecated_test.go:53
**Решение:** Добавить `require.NoError` для io.Copy.

### L7: Race condition window в RegisterWithAlias() задокументирован, но не защищён
**Решение:** Оставить как есть — init() однопоточен по гарантиям Go runtime. Комментарий достаточен.

### L8: TextWriter ошибка сериализации Data — сообщение без подсказки о типах
**Решение:** Расширить error message: "Data must contain JSON-serializable types".

---

## Матрица приоритетов для Epic 2

| Категория | Блокирует Epic 2 | До релиза Epic 1 | В процессе Epic 2 |
|-----------|:-:|:-:|:-:|
| C1 (nil Config DI) | **Да** | | |
| C2 (CaptureStdout race) | | **Да** | |
| C3 (test registry race) | | **Да** | |
| C4 (os.Pipe error) | | **Да** | |
| C5 (Validate empty) | **Да** | | |
| H1 (os.Stdout hardcode) | | | **Epic 2 рефакторинг** |
| H2 (HTML escaping) | | **Да** | |
| H3 (IO error tests) | | | **Да** |
| H4 (nil writer) | | **Да** | |
| H5 (odd args With) | | | **Да** |
| H6 (0% coverage) | | **Да** | |
| H7 (fmt.Fprintf error) | | | **Epic 2 рефакторинг** |
| H8 (exit codes) | | | **Epic 2+ миграция** |
| M1-M10 | | | По мере возможности |

---

## Рекомендуемый порядок исправления

**Фаза 1: Блокеры Epic 2 (перед началом Epic 2)**
1. C1 — Wire DI nil Config
2. C5 — ImplementationsConfig validation

**Фаза 2: Стабильность тестов (перед релизом Epic 1)**
3. C2 — CaptureStdout синхронизация
4. C3 — Integration test registry isolation
5. C4 — os.Pipe error handling
6. H4 — nil writer panic
7. H6 — 0% coverage для Description/IsDeprecated/NewName

**Фаза 3: Качество кода (в процессе Epic 2)**
8. H2 — SetEscapeHTML(false)
9. H3 — IO error tests
10. H5 — With() odd args
11. M2 — testutil.CaptureStderr
12. Остальные MEDIUM и LOW — по мере работы с затронутыми файлами

---

_Документ создан в рамках ретроспективы Epic 1, 2026-01-27_
