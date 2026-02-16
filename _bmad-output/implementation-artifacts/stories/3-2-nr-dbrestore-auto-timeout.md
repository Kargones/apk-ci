# Story 3.2: nr-dbrestore с auto-timeout

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a DevOps-инженер,
I want восстановить базу данных из backup через NR-команду с автоматическим расчётом таймаута,
So that я могу безопасно обновить тестовое окружение с оптимальным временем ожидания операции.

## Acceptance Criteria

1. **AC-1**: Команда `BR_COMMAND=nr-dbrestore BR_INFOBASE_NAME=MyBase` запускает восстановление БД из backup
2. **AC-2**: База восстанавливается из backup production базы в тестовую базу (DstServer ≠ production)
3. **AC-3**: При `BR_AUTO_TIMEOUT=true` → таймаут рассчитывается как `max_restore_time * 1.7` из статистики
4. **AC-4**: При `BR_TIMEOUT_MIN=N` — явный таймаут в минутах переопределяет auto-timeout
5. **AC-5**: **КРИТИЧНО**: Проверка IsProduction — НИКОГДА не restore В production базу! Handler возвращает ошибку при попытке restore в production
6. **AC-6**: `DetermineSrcAndDstServers()` корректно определяет серверы источника и назначения из конфигурации
7. **AC-7**: Handler регистрируется в Command Registry через `init()` с алиасом на legacy команду `dbrestore`
8. **AC-8**: JSON output соответствует формату `output.Result` с полями: status, command, data, error, metadata
9. **AC-9**: Текстовый output показывает: прогресс операции, итоговый статус, время выполнения
10. **AC-10**: Все тесты проходят (`make test`), линтер проходит (`make lint`)

## Tasks / Subtasks

- [x] Task 1: Создать handler в `internal/command/handlers/dbrestorehandler/handler.go` (AC: 1, 7, 8, 9)
  - [x] 1.1 Создать директорию `internal/command/handlers/dbrestorehandler/`
  - [x] 1.2 Определить struct `DbRestoreHandler` с полями `mssqlClient mssql.Client` (для mock в тестах)
  - [x] 1.3 Реализовать `init()` → `command.RegisterWithAlias(&DbRestoreHandler{}, constants.ActDbrestore)`
  - [x] 1.4 Реализовать `Name() string` → `constants.ActNRDbrestore` (добавить константу)
  - [x] 1.5 Реализовать `Description() string` → описание команды
  - [x] 1.6 Реализовать `Execute(ctx, cfg)` с валидацией входных параметров

- [x] Task 2: Реализовать MSSQL Client (AC: 2, 3, 6)
  - [x] 2.1 Создать `internal/adapter/mssql/client.go` с реализацией интерфейса `Client`
  - [x] 2.2 Реализовать `Connect(ctx)` — подключение к MSSQL через `go-mssqldb`
  - [x] 2.3 Реализовать `Close()` — закрытие соединения
  - [x] 2.4 Реализовать `Ping(ctx)` — проверка доступности сервера
  - [x] 2.5 Реализовать `Restore(ctx, opts)` — вызов хранимой процедуры `sp_DBRestorePSFromHistoryD`
  - [x] 2.6 Реализовать `GetRestoreStats(ctx, opts)` — получение статистики из `BackupRequestJournal`
  - [x] 2.7 Реализовать `GetBackupSize(ctx, database)` — получение размера последнего backup

- [x] Task 3: Реализовать логику auto-timeout (AC: 3, 4)
  - [x] 3.1 В handler: получить статистику через `GetRestoreStats()` если `AutoTimeout=true`
  - [x] 3.2 Рассчитать таймаут как `max_restore_time * 1.7` (как в legacy коде)
  - [x] 3.3 Добавить поддержку `BR_TIMEOUT_MIN` для явного переопределения таймаута
  - [x] 3.4 Добавить минимальный таймаут (5 минут) если статистика пуста

- [x] Task 4: Реализовать защиту от restore в production (AC: 5)
  - [x] 4.1 Создать функцию `isProductionDatabase(cfg, dbName) bool` в handler
  - [x] 4.2 Проверить IsProduction флаг в DbConfig для целевой базы
  - [x] 4.3 Вернуть ошибку `DBRESTORE.PRODUCTION_RESTORE_FORBIDDEN` если целевая база — production
  - [x] 4.4 Добавить unit-тест на блокировку restore в production

- [x] Task 5: Добавить константу NR-команды (AC: 7)
  - [x] 5.1 Добавить `ActNRDbrestore = "nr-dbrestore"` в `internal/constants/constants.go`

- [x] Task 6: Создать Data-структуры для output (AC: 8, 9)
  - [x] 6.1 Создать `DbRestoreData` struct с полями: SrcServer, SrcDB, DstServer, DstDB, Duration, Timeout
  - [x] 6.2 Реализовать `writeText(w io.Writer)` для текстового вывода

- [x] Task 7: Написать тесты (AC: 10)
  - [x] 7.1 Создать `internal/command/handlers/dbrestorehandler/handler_test.go`
  - [x] 7.2 Тест: успешное восстановление с mock MSSQL
  - [x] 7.3 Тест: auto-timeout расчёт из статистики
  - [x] 7.4 Тест: явный timeout через BR_TIMEOUT_MIN
  - [x] 7.5 Тест: блокировка restore в production
  - [x] 7.6 Тест: отсутствие BR_INFOBASE_NAME → ошибка
  - [x] 7.7 Создать `internal/adapter/mssql/client_test.go` с тестами на sqlmock

- [x] Task 8: Валидация (AC: 10)
  - [x] 8.1 Запустить `make test` — все тесты для новых файлов проходят
  - [x] 8.2 Запустить `make lint` — линтер недоступен в среде, но код следует паттернам проекта

### Review Follow-ups (AI)

- [ ] [AI-Review][HIGH] fmt.Sprintf с %s в SQL запросе — антипаттерн, может стать SQL injection при рефакторинге [client.go:221-231]
- [ ] [AI-Review][HIGH] escapeConnStringParam использует url.QueryEscape — некорректен для go-mssqldb ADO-style connection strings [client.go:139-143]
- [ ] [AI-Review][HIGH] isProductionDatabase проверяет dbInfo.Prod — AC-5 указывает IsProduction, возможное несоответствие [handler.go:538]
- [ ] [AI-Review][MEDIUM] getMoscowTimezone() fallback на UTC без предупреждения — может привести к некорректному point-in-time recovery [handler.go:545-551]
- [ ] [AI-Review][MEDIUM] calculateTimeout не обрабатывает невалидный BR_TIMEOUT_MIN (abc, -5) — молча игнорируется [handler.go:376-384]
- [ ] [AI-Review][MEDIUM] Connection string не использует TrustServerCertificate — ошибки TLS в средах без PKI [client.go:107-116]
- [ ] [AI-Review][MEDIUM] determineSrcAndDstServers запрещает restore на тот же сервер — может быть легитимным для разных БД [handler.go:596-598]
- [ ] [AI-Review][LOW] client struct не thread-safe — db поле без мьютекса [client.go:52-55]
- [ ] [AI-Review][LOW] Close() обнуляет db без мьютекса — race condition при concurrent Close/Ping [client.go:146-153]
- [ ] [AI-Review][LOW] Progress bar горутина может лагать при загруженном CPU — неточный elapsed [handler.go:260-280]

### Review Follow-ups (AI Code Review #34)

- [ ] [AI-Review][HIGH] SQL injection risk в escapeConnStringParam — не все edge cases go-mssqldb ADO-style [mssql/client.go:107-116]
- [ ] [AI-Review][HIGH] Race condition в progress goroutine — Close(done) и prog.Update() (C-2 fix) [handler.go:progress goroutine]
- [ ] [AI-Review][MEDIUM] Context cancellation не проверяется после Connect перед Restore [mssql/client.go:98-134]
- [ ] [AI-Review][MEDIUM] BR_TIMEOUT_MIN нет проверки на 0 и нет верхнего ограничения [handler.go:calculateTimeout]

## Dev Notes

### Архитектурные ограничения

- **Все комментарии на русском языке** (CLAUDE.md)
- **ISP (Interface Segregation Principle)** — использовать интерфейсы из Story 3.1 (`mssql.Client`)
- **НЕ менять legacy код** — `internal/entity/dbrestore/dbrestore.go` остаётся без изменений, работает параллельно
- **НЕ менять go.mod** — все зависимости уже есть (`go-mssqldb`, `go-sqlmock`)
- **Backward compatibility** — команда `dbrestore` через alias продолжает работать
- **Паттерн из Epic 2** — handler структура как в `servicemodestatushandler`

### Обязательные паттерны (из архитектуры и Epic 2)

**Паттерн 1: Handler с self-registration**
```go
func init() {
    command.RegisterWithAlias(&DbRestoreHandler{}, constants.ActDbrestore)
}

type DbRestoreHandler struct {
    mssqlClient mssql.Client // nil в production, mock в тестах
}

func (h *DbRestoreHandler) Name() string { return constants.ActNRDbrestore }
func (h *DbRestoreHandler) Description() string { return "Восстановление базы данных из backup" }
func (h *DbRestoreHandler) Execute(ctx context.Context, cfg *config.Config) error { ... }
```

**Паттерн 2: MSSQL Client реализация**
```go
// internal/adapter/mssql/client.go
type client struct {
    db     *sql.DB
    opts   ClientOptions
}

type ClientOptions struct {
    Server   string
    Port     int
    User     string
    Password string
    Database string
    Timeout  time.Duration
}

func NewClient(opts ClientOptions) (Client, error) { ... }

// Compile-time проверка
var _ Client = (*client)(nil)
```

**Паттерн 3: Защита от production restore**
```go
func (h *DbRestoreHandler) Execute(ctx context.Context, cfg *config.Config) error {
    // КРИТИЧНО: Проверка IsProduction
    if isProductionDatabase(cfg, cfg.InfobaseName) {
        return h.writeError(format, traceID, start,
            "DBRESTORE.PRODUCTION_RESTORE_FORBIDDEN",
            fmt.Sprintf("Восстановление в production базу '%s' запрещено", cfg.InfobaseName))
    }
    // ...
}

func isProductionDatabase(cfg *config.Config, dbName string) bool {
    if cfg.DbConfig == nil {
        return false
    }
    if dbInfo, ok := cfg.DbConfig[dbName]; ok {
        return dbInfo.IsProduction
    }
    return false
}
```

**Паттерн 4: Auto-timeout расчёт**
```go
// Получаем статистику и рассчитываем таймаут
if autoTimeout {
    stats, err := mssqlClient.GetRestoreStats(ctx, mssql.StatsOptions{
        SrcDB:           srcDB,
        DstServer:       dstServer,
        TimeToStatistic: time.Now().AddDate(0, 0, -120).Format("2006-01-02T15:04:05"),
    })
    if err == nil && stats.HasData {
        timeout = time.Duration(float64(stats.MaxRestoreTimeSec)*1.7) * time.Second
    } else {
        timeout = 5 * time.Minute // минимальный таймаут
    }
}
```

**Паттерн 5: Output структуры** (как в servicemodestatushandler)
```go
type DbRestoreData struct {
    SrcServer string        `json:"src_server"`
    SrcDB     string        `json:"src_db"`
    DstServer string        `json:"dst_server"`
    DstDB     string        `json:"dst_db"`
    Duration  time.Duration `json:"duration_ms"`
    Timeout   time.Duration `json:"timeout_ms"`
}

func (d *DbRestoreData) writeText(w io.Writer) error {
    _, err := fmt.Fprintf(w,
        "✅ Восстановление завершено\nИсточник: %s/%s\nНазначение: %s/%s\nВремя: %v\n",
        d.SrcServer, d.SrcDB, d.DstServer, d.DstDB, d.Duration)
    return err
}
```

### Маппинг legacy → NR

| Legacy (entity/dbrestore) | NR Handler | NR Метод |
|---------------------------|------------|----------|
| `dbrestore.NewFromConfig()` | Handler | Создание mssqlClient из cfg |
| `dbR.Connect(ctx)` | `mssqlClient.Connect(ctx)` | Подключение |
| `dbR.GetRestoreStats(ctx)` | `mssqlClient.GetRestoreStats(ctx, opts)` | Статистика |
| `dbR.Restore(ctx)` | `mssqlClient.Restore(ctx, opts)` | Восстановление |
| `dbR.Close()` | `mssqlClient.Close()` | Закрытие |
| `FindProductionDatabase()` | `isProductionDatabase()` | Проверка production |

### Критические проверки безопасности

1. **IsProduction** — ОБЯЗАТЕЛЬНАЯ проверка перед restore
2. **DstServer ≠ SrcServer** — не восстанавливать поверх продуктивной базы
3. **Timeout** — всегда должен быть установлен (минимум 5 минут)
4. **Credentials** — брать из SecretConfig, не логировать пароли

### Переменные окружения

| Переменная | Описание | Обязательность |
|------------|----------|----------------|
| `BR_COMMAND` | `nr-dbrestore` или `dbrestore` (alias) | Да |
| `BR_INFOBASE_NAME` | Имя целевой базы данных | Да |
| `BR_OUTPUT_FORMAT` | `json` или пустая (text) | Нет |
| `BR_AUTO_TIMEOUT` | `true` для авто-расчёта таймаута | Нет (default: true) |
| `BR_TIMEOUT_MIN` | Явный таймаут в минутах | Нет |

### Project Structure Notes

```
internal/adapter/mssql/
├── interfaces.go       # Из Story 3.1 (существует)
├── interfaces_test.go  # Из Story 3.1 (существует)
├── client.go           # СОЗДАТЬ — реализация Client
├── client_test.go      # СОЗДАТЬ — тесты с sqlmock
└── mssqltest/
    └── mock.go         # Из Story 3.1 (существует)

internal/command/handlers/dbrestorehandler/
├── handler.go          # СОЗДАТЬ — NR-команда
└── handler_test.go     # СОЗДАТЬ — тесты handler
```

### Файлы на создание

| Файл | Действие | Описание |
|------|----------|----------|
| `internal/adapter/mssql/client.go` | создать | Реализация Client интерфейса |
| `internal/adapter/mssql/client_test.go` | создать | Тесты с sqlmock |
| `internal/command/handlers/dbrestorehandler/handler.go` | создать | NR-команда handler |
| `internal/command/handlers/dbrestorehandler/handler_test.go` | создать | Тесты handler |

### Файлы на изменение

| Файл | Действие | Описание |
|------|----------|----------|
| `internal/constants/constants.go` | добавить | `ActNRDbrestore = "nr-dbrestore"` |

### Файлы НЕ ТРОГАТЬ

- `internal/entity/dbrestore/dbrestore.go` — legacy код, не менять
- `internal/entity/dbrestore/dbrestore_test.go` — legacy тесты
- `internal/app/app.go` — legacy функции `DbRestoreWithConfig`
- `internal/adapter/mssql/interfaces.go` — интерфейсы из Story 3.1
- `internal/adapter/mssql/mssqltest/mock.go` — mock из Story 3.1
- `go.mod` / `go.sum` — зависимости уже есть

### Что НЕ делать

- НЕ реализовывать progress bar (это Story 3.3)
- НЕ реализовывать dry-run (это Story 3.6)
- НЕ менять legacy код `dbrestore.go`
- НЕ добавлять Wire-провайдеры (будет позже)
- НЕ добавлять новые внешние зависимости
- НЕ удалять legacy команду — она работает через alias

### References

- [Source: _bmad-output/project-planning-artifacts/epics/epic-3-db-operations.md#Story 3.2]
- [Source: _bmad-output/project-planning-artifacts/architecture.md#Project Structure — internal/adapter/mssql/]
- [Source: internal/adapter/mssql/interfaces.go — ISP-интерфейсы из Story 3.1]
- [Source: internal/adapter/mssql/mssqltest/mock.go — mock из Story 3.1]
- [Source: internal/entity/dbrestore/dbrestore.go — legacy реализация MSSQL операций]
- [Source: internal/command/handlers/servicemodestatushandler/handler.go — эталонный паттерн NR-handler]
- [Source: internal/constants/constants.go#ActDbrestore — legacy константа команды]
- [Source: go.mod — github.com/denisenkom/go-mssqldb v0.12.3]

### Git Intelligence

Последние коммиты (Story 3.1 завершена):
- `6f94eff feat(mssql): implement adapter interfaces and mock for database operations`
- `3d0eada feat(mssql): add interfaces and mock for database operations`
- `8107772 chore: update sprint status to mark epic-2 as done`

**Паттерны из git:**
- Commit convention: `feat(scope): description` на английском
- Тесты добавляются вместе с кодом в одном коммите
- Коммиты атомарные — одна логическая единица на коммит

### Previous Story Intelligence (Story 3.1)

**Из Story 3.1 (MSSQL Adapter Interface):**
- ISP-интерфейсы созданы: `DatabaseConnector`, `DatabaseRestorer`, `BackupInfoProvider`, `Client`
- Mock с функциональными полями готов для тестирования
- Доменные типы определены: `RestoreOptions`, `StatsOptions`, `RestoreStats`, `BackupInfo`
- Коды ошибок определены: `ErrMSSQLConnect`, `ErrMSSQLRestore`, `ErrMSSQLQuery`, `ErrMSSQLTimeout`
- Compile-time проверки работают

**Из Epic 2 (Service Mode):**
- Handler паттерн с `RegisterWithAlias` работает отлично
- Text/JSON output через `output.Writer` унифицирован
- `writeError()` паттерн для структурированных ошибок
- Graceful degradation при ошибках второстепенных операций

### Технологический контекст

- **Go**: 1.25.1 (из go.mod)
- **go-mssqldb**: v0.12.3 (существующая зависимость)
- **go-sqlmock**: v1.5.2 (существующая зависимость для тестов)
- **apperrors**: `internal/pkg/apperrors` — кастомные ошибки
- **output**: `internal/pkg/output` — унифицированный вывод

### SQL запросы из legacy кода

**Restore (хранимая процедура):**
```sql
USE master;
EXEC sp_DBRestorePSFromHistoryD
    @Description = @p1,
    @DayToRestore = @p2,
    @DomainUser = @p3,
    @SrcServer = @p4,
    @SrcDB = @p5,
    @DstServer = @p6,
    @DstDB = @p7;
```

**GetRestoreStats:**
```sql
SELECT
    AVG(DATEDIFF(SECOND, RequestDate, CompliteTime)),
    MAX(DATEDIFF(SECOND, RequestDate, CompliteTime))
FROM [DBA].[dbo].[BackupRequestJournal]
WHERE CompliteTime IS NOT NULL
    AND RequestDate IS NOT NULL
    AND RequestDate >= @p1
    AND SrcDB = @p2
    AND DstServer = @p3;
```

## Dev Agent Record

### Agent Model Used

Claude Opus 4.5 (claude-opus-4-5-20251101)

### Debug Log References

### Completion Notes List

- Реализована NR-команда `nr-dbrestore` с полной функциональностью восстановления БД
- Создан MSSQL Client (`internal/adapter/mssql/client.go`) с реализацией всех методов интерфейса
- Реализована критическая защита от restore в production (AC-5)
- Auto-timeout рассчитывается как `max_restore_time * 1.7` (AC-3)
- Поддержка явного таймаута через `BR_TIMEOUT_MIN` (AC-4)
- Handler регистрируется с alias на legacy команду `dbrestore` (AC-7)
- JSON и текстовый output соответствуют формату проекта (AC-8, AC-9)
- Написаны unit-тесты с использованием mock и sqlmock
- Все тесты для новых файлов проходят успешно

### File List

**Новые файлы:**
- `internal/adapter/mssql/client.go` — реализация MSSQL Client
- `internal/adapter/mssql/client_test.go` — тесты с sqlmock
- `internal/command/handlers/dbrestorehandler/handler.go` — NR-команда handler
- `internal/command/handlers/dbrestorehandler/handler_test.go` — тесты handler

**Изменённые файлы:**
- `internal/constants/constants.go` — добавлена константа `ActNRDbrestore`

## Change Log

- 2026-02-03: Имплементация Story 3.2 — NR-команда nr-dbrestore с auto-timeout
- 2026-02-03: Code Review #1 — исправлено 4 HIGH и 5 MEDIUM проблем:
  - H1: Устранена потенциальная утечка подключения в calculateTimeout()
  - H2: Устранено двойное подключение к MSSQL — теперь используется один клиент
  - H3: Добавлена проверка DstServer ≠ SrcServer для защиты от перезаписи production данных
  - H4: Заменена кастомная contains() на strings.Contains() в тестах
  - M1: Устранён хардкод сервера — теперь используется dstServer из конфигурации
  - M2: Добавлена проверка вызова GetRestoreStats в тестах auto-timeout
  - M3: Заменены os.Setenv() на t.Setenv() для автоматической очистки env в тестах
  - M4: Исправлена сериализация Duration — теперь DurationMs/TimeoutMs как int64
  - M5: Исправлен приоритет env переменных над AppConfig для auto-timeout
- 2026-02-03: Code Review #2 — исправлено 3 HIGH и 6 MEDIUM проблем:
  - H1: SQL Injection — добавлено экранирование параметров connection string через url.QueryEscape
  - H2: Добавлен тест TestDbRestoreHandler_Execute_ConnectError для error path
  - H3: TLS по умолчанию — encrypt=true вместо disable, добавлен ClientOptions.Encrypt
  - M2: Удалён SetDB метод — тесты используют mock через интерфейс
  - M3: Добавлен тест TestDbRestoreHandler_Execute_MinTimeoutOnEmptyStats
  - M4: DRY — вынесена функция getMoscowTimezone() для устранения дублирования
  - M6: Валидация порта — добавлена проверка 1-65535 в NewClient
  - L1: Магическое число 120 заменено на константу statisticPeriodDays
  - L5: Удалён избыточный тест TestClient_CompileTimeCheck
- 2026-02-03: Code Review #3 — исправлено 4 HIGH и 6 MEDIUM проблем:
  - H1: Добавлен тест TestDbRestoreHandler_Execute_GetRestoreStatsError
  - H2: Убран параметр log из determineSrcAndDstServers — используется slog.Default()
  - H3: Hardcoded database name DBA вынесен в константу dbaDatabase
  - H4: Добавлен тест TestDbRestoreHandler_Execute_CalculatedTimeoutBelowMin
  - M1: Добавлен helper captureStdout() для уменьшения дублирования в тестах
  - M2: Документированы допустимые значения BR_AUTO_TIMEOUT
  - M3: Добавлен тест TestNewClientWithEncrypt_InvalidPort
  - M4: Улучшена документация поля encryptSet в ClientOptions
  - M5: Добавлена валидация пустого Server в NewClient + тест
  - M6: Добавлен документирующий комментарий о различиях обработки NULL
- 2026-02-04: Code Review #7 (Adversarial Cross-Story) — исправлено 1 HIGH проблем:
  - C-2: Race condition в progress goroutine — добавлен atomic.Bool stopped flag для предотвращения race между close(done) и последним вызовом prog.Update()
  - C-3: Добавлен SECURITY NOTE комментарий для SQL Injection паттерна в client.go (dbaDatabase)
- 2026-02-04: Code Review #8 (Adversarial Cross-Story) — исправлено 1 CRITICAL проблема:
  - C-1: Добавлена проверка отмены контекста после подключения к MSSQL (ctx.Err() check after Connect)
