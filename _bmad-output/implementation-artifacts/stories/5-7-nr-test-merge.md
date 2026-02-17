# Story 5.7: nr-test-merge

Status: done

## Story

As a DevOps-инженер,
I want проверить конфликты слияния для всех открытых PR через NR-команду,
so that я знаю какие PR требуют внимания перед merge.

## Acceptance Criteria

1. [AC1] BR_COMMAND=nr-test-merge — команда выполняется через NR Command Registry
2. [AC2] Проверяются все открытые PR репозитория на конфликты слияния
3. [AC3] Создаётся временная тестовая ветка для проверки merge-ability
4. [AC4] Для каждого PR проверяется наличие конфликтов (mergeable check)
5. [AC5] Для каждого PR выполняется попытка тестового merge
6. [AC6] Конфликтные PR закрываются с комментарием о причине
7. [AC7] JSON output возвращает детальный список PR с их статусами и конфликтами
8. [AC8] Text output возвращает читаемый summary с результатом проверки
9. [AC9] Интеграция с NR-адаптером: использует `gitea.Client` (Story 5-2)
10. [AC10] Временная тестовая ветка удаляется после завершения операции (cleanup)
11. [AC11] Deprecated alias: legacy "test-merge" маршрутизируется на "nr-test-merge"

## Tasks / Subtasks

- [x] Task 1: Создать файл `internal/command/handlers/gitea/testmerge/handler.go` (AC: #1, #11)
  - [x] Subtask 1.1: Определить TestMergeHandler struct с полем giteaClient gitea.Client
  - [x] Subtask 1.2: Реализовать init() с command.RegisterWithAlias для "nr-test-merge" и deprecated "test-merge"
  - [x] Subtask 1.3: Реализовать Name() -> "nr-test-merge", Description()
  - [x] Subtask 1.4: Определить TestMergeData struct для JSON response
  - [x] Subtask 1.5: Реализовать writeText() для TestMergeData с табличным отображением PR

- [x] Task 2: Реализовать Execute() с валидацией (AC: #9)
  - [x] Subtask 2.1: Валидировать: cfg != nil, иначе ошибка CONFIG.MISSING
  - [x] Subtask 2.2: Получить Owner и Repo из cfg
  - [x] Subtask 2.3: Валидировать: Owner != "" и Repo != "", иначе ошибка CONFIG.MISSING_OWNER_REPO
  - [x] Subtask 2.4: Получить BaseBranch из cfg (или default "main")

- [x] Task 3: Реализовать получение списка открытых PR (AC: #2, #9)
  - [x] Subtask 3.1: Вызвать giteaClient.ListOpenPRs(ctx) через PRReader interface
  - [x] Subtask 3.2: Обработать случай пустого списка (нет открытых PR) — success без действий
  - [x] Subtask 3.3: Логировать количество найденных открытых PR

- [x] Task 4: Реализовать создание тестовой ветки (AC: #3, #10)
  - [x] Subtask 4.1: Определить имя тестовой ветки: "test-merge-{timestamp}" или из constants
  - [x] Subtask 4.2: Удалить существующую тестовую ветку если есть (cleanup от предыдущего запуска)
  - [x] Subtask 4.3: Создать новую тестовую ветку на основе BaseBranch через giteaClient.CreateBranch
  - [x] Subtask 4.4: Обработать ошибку создания ветки — fatal error

- [x] Task 5: Реализовать проверку конфликтов для каждого PR (AC: #4, #5, #6)
  - [x] Subtask 5.1: Для каждого PR создать временный PR из head ветки в тестовую ветку
  - [x] Subtask 5.2: Проверить mergeable статус через giteaClient.ConflictPR(prNumber)
  - [x] Subtask 5.3: Попытаться выполнить тестовый merge через giteaClient.MergePR
  - [x] Subtask 5.4: Если merge fail или конфликт — закрыть оригинальный PR через giteaClient.ClosePR
  - [x] Subtask 5.5: Собрать результаты для каждого PR (hasConflict, mergeResult, conflictFiles)
  - [x] Subtask 5.6: Получить список конфликтных файлов через giteaClient.ConflictFilesPR (для JSON output)

- [x] Task 6: Реализовать cleanup (AC: #10)
  - [x] Subtask 6.1: Удалить тестовую ветку через giteaClient.DeleteBranch в defer
  - [x] Subtask 6.2: Логировать ошибки cleanup (не fatal, только warning)

- [x] Task 7: Реализовать вывод результатов (AC: #7, #8)
  - [x] Subtask 7.1: JSON format через output.WriteSuccess с TestMergeData
  - [x] Subtask 7.2: Text format через writeText() с табличным отображением PR
  - [x] Subtask 7.3: Обработка ошибок через output.WriteError с кодами CONFIG.*, GITEA.*

- [x] Task 8: Написать unit-тесты (AC: #1-#11)
  - [x] Subtask 8.1: Создать `handler_test.go` с MockClient для gitea
  - [x] Subtask 8.2: TestExecute_NoPRs — нет открытых PR (success)
  - [x] Subtask 8.3: TestExecute_AllMergeable — все PR без конфликтов
  - [x] Subtask 8.4: TestExecute_SomeConflicts — часть PR с конфликтами
  - [x] Subtask 8.5: TestExecute_AllConflicts — все PR с конфликтами
  - [x] Subtask 8.6: TestExecute_CreateBranchError — ошибка создания тестовой ветки
  - [x] Subtask 8.7: TestExecute_MissingConfig — отсутствует конфигурация
  - [x] Subtask 8.8: TestExecute_JSONOutput — проверка JSON формата
  - [x] Subtask 8.9: TestExecute_CleanupOnError — cleanup выполняется даже при ошибках

- [x] Task 9: Добавить константу в constants.go (AC: #1)
  - [x] Subtask 9.1: Добавить ActNRTestMerge = "nr-test-merge"

### Review Follow-ups (AI)

- [ ] [AI-Review][HIGH] Автоматическое закрытие PR при конфликте — деструктивная операция без подтверждения [handler.go:358-370]
- [ ] [AI-Review][HIGH] Команда неработоспособна в production (H-6) — giteaClient nil [handler.go:238-246]
- [ ] [AI-Review][MEDIUM] Нет проверки context.Done() в цикле PR — при 100+ PR отмена не прервёт итерацию [handler.go:299-311]
- [ ] [AI-Review][MEDIUM] CreatePR для тестирования — тестовые PR не закрываются при ошибке, утечка [handler.go:333-341]
- [ ] [AI-Review][MEDIUM] generateTestBranchName — секундная точность, конфликт при параллельном CI [handler.go:29-31]
- [ ] [AI-Review][LOW] Предварительное удаление ветки — silent suppression, race condition с другим процессом [handler.go:273]
- [ ] [AI-Review][LOW] ConflictPR ошибка API = conflict assumed — ложноположительный конфликт закроет PR [handler.go:346-347]

## Dev Notes

### Архитектурные паттерны и ограничения

**Command Handler Pattern** [Source: internal/command/handlers/sonarqube/scanpr/handler.go]
- Self-registration через init() + command.RegisterWithAlias()
- Поддержка deprecated alias ("test-merge" -> "nr-test-merge")
- Dual output: JSON (BR_OUTPUT_FORMAT=json) / текст (по умолчанию)
- Следовать паттерну установленному в Story 5-3, 5-4, 5-5, 5-6

**ISP-compliant Gitea Adapter (Story 5-2):**
- PRReader.ListOpenPRs(ctx) — список открытых PR
- PRReader.ConflictPR(ctx, prNumber) — проверка конфликта
- PRReader.ConflictFilesPR(ctx, prNumber) — список конфликтных файлов
- PRManager.CreatePR(ctx, head) — создание временного PR
- PRManager.MergePR(ctx, prNumber) — тестовый merge
- PRManager.ClosePR(ctx, prNumber) — закрытие конфликтного PR
- BranchManager.CreateBranch(ctx, newBranch, baseBranch) — создание тестовой ветки
- BranchManager.DeleteBranch(ctx, branchName) — удаление тестовой ветки

### Структура handler

```go
package testmerge

import (
    "context"
    "fmt"
    "io"
    "log/slog"
    "os"
    "time"

    "github.com/Kargones/apk-ci/internal/adapter/gitea"
    "github.com/Kargones/apk-ci/internal/command"
    "github.com/Kargones/apk-ci/internal/config"
    "github.com/Kargones/apk-ci/internal/constants"
    "github.com/Kargones/apk-ci/internal/pkg/output"
    "github.com/Kargones/apk-ci/internal/pkg/tracing"
)

// Имя тестовой ветки
const testBranchName = "test-merge-branch"

func init() {
    // TODO(H-7): Deprecated alias "test-merge" будет удалён в v2.0.0 / Epic 7.
    // После полной миграции на NR-архитектуру, использовать только "nr-test-merge".
    command.RegisterWithAlias(&TestMergeHandler{}, constants.ActTestMerge)
}

type TestMergeHandler struct {
    // giteaClient — клиент для работы с Gitea API.
    // Может быть nil в production (создаётся через фабрику).
    // В тестах инъектируется напрямую.
    giteaClient gitea.Client
}

func (h *TestMergeHandler) Name() string { return constants.ActNRTestMerge }
func (h *TestMergeHandler) Description() string {
    return "Проверить конфликты слияния для всех открытых PR"
}
```

### Структуры данных для ответа

```go
// TestMergeData содержит результат проверки конфликтов слияния.
type TestMergeData struct {
    // TotalPRs — общее количество проверенных PR
    TotalPRs int `json:"total_prs"`
    // MergeablePRs — количество PR без конфликтов
    MergeablePRs int `json:"mergeable_prs"`
    // ConflictPRs — количество PR с конфликтами
    ConflictPRs int `json:"conflict_prs"`
    // ClosedPRs — количество закрытых PR из-за конфликтов
    ClosedPRs int `json:"closed_prs"`
    // PRResults — детальные результаты для каждого PR
    PRResults []PRMergeResult `json:"pr_results"`
    // TestBranch — имя использованной тестовой ветки
    TestBranch string `json:"test_branch"`
    // BaseBranch — базовая ветка для тестирования
    BaseBranch string `json:"base_branch"`
}

// PRMergeResult содержит результат проверки одного PR.
type PRMergeResult struct {
    // PRNumber — номер PR в репозитории
    PRNumber int64 `json:"pr_number"`
    // HeadBranch — исходная ветка PR
    HeadBranch string `json:"head_branch"`
    // BaseBranch — целевая ветка PR
    BaseBranch string `json:"base_branch"`
    // HasConflict — есть ли конфликт
    HasConflict bool `json:"has_conflict"`
    // MergeResult — результат попытки merge ("success", "conflict", "error")
    MergeResult string `json:"merge_result"`
    // ConflictFiles — список файлов с конфликтами (если есть)
    ConflictFiles []string `json:"conflict_files,omitempty"`
    // Closed — был ли PR закрыт из-за конфликта
    Closed bool `json:"closed"`
    // ErrorMessage — сообщение об ошибке (если есть)
    ErrorMessage string `json:"error_message,omitempty"`
}
```

### Коды ошибок

```go
// Используем shared коды + новые для test-merge
const (
    errConfigMissing     = "CONFIG.MISSING"           // Nil config
    errMissingOwnerRepo  = "CONFIG.MISSING_OWNER_REPO" // Не указан owner/repo
    errGiteaAPI          = "GITEA.API_FAILED"         // Ошибка API Gitea
    errBranchCreate      = "GITEA.BRANCH_CREATE_FAILED" // Ошибка создания тестовой ветки
    errNoPRs             = "GITEA.NO_OPEN_PRS"        // Нет открытых PR (info, не error)
)
```

### Логика Execute (алгоритм)

```go
func (h *TestMergeHandler) Execute(ctx context.Context, cfg *config.Config) error {
    start := time.Now()
    traceID := tracing.TraceIDFromContext(ctx)
    if traceID == "" {
        traceID = tracing.GenerateTraceID()
    }
    format := os.Getenv("BR_OUTPUT_FORMAT")
    log := slog.Default().With("trace_id", traceID, "command", h.Name())

    // 1. Валидация конфигурации
    if cfg == nil {
        return h.writeError(format, traceID, start, errConfigMissing, "Config not loaded")
    }

    owner := cfg.Owner
    repo := cfg.Repo
    if owner == "" || repo == "" {
        return h.writeError(format, traceID, start, errMissingOwnerRepo, "Owner and Repo are required")
    }

    baseBranch := cfg.BaseBranch
    if baseBranch == "" {
        baseBranch = "main"
    }

    log.Info("Starting test-merge check", "owner", owner, "repo", repo, "base_branch", baseBranch)

    client := h.getGiteaClient(cfg)

    // 2. Получение списка открытых PR
    activePRs, err := client.ListOpenPRs(ctx)
    if err != nil {
        log.Error("Failed to list open PRs", "error", err)
        return h.writeError(format, traceID, start, errGiteaAPI, "Failed to list open PRs: "+err.Error())
    }

    if len(activePRs) == 0 {
        log.Info("No open PRs found")
        return h.writeSuccess(format, traceID, start, &TestMergeData{
            TotalPRs:   0,
            TestBranch: testBranchName,
            BaseBranch: baseBranch,
        })
    }

    log.Debug("Found open PRs", "count", len(activePRs))

    // 3. Cleanup + создание тестовой ветки
    _ = client.DeleteBranch(ctx, testBranchName) // Ignore error — ветка может не существовать
    err = client.CreateBranch(ctx, testBranchName, baseBranch)
    if err != nil {
        log.Error("Failed to create test branch", "error", err)
        return h.writeError(format, traceID, start, errBranchCreate, "Failed to create test branch: "+err.Error())
    }
    defer func() {
        if delErr := client.DeleteBranch(ctx, testBranchName); delErr != nil {
            log.Warn("Failed to delete test branch", "error", delErr)
        }
    }()

    log.Debug("Test branch created", "branch", testBranchName, "base", baseBranch)

    // 4. Проверка каждого PR
    data := &TestMergeData{
        TotalPRs:   len(activePRs),
        TestBranch: testBranchName,
        BaseBranch: baseBranch,
        PRResults:  make([]PRMergeResult, 0, len(activePRs)),
    }

    for _, pr := range activePRs {
        result := h.checkPR(ctx, client, pr, log)
        data.PRResults = append(data.PRResults, result)

        if result.HasConflict {
            data.ConflictPRs++
            if result.Closed {
                data.ClosedPRs++
            }
        } else {
            data.MergeablePRs++
        }
    }

    log.Info("Test-merge completed",
        "total", data.TotalPRs,
        "mergeable", data.MergeablePRs,
        "conflicts", data.ConflictPRs,
        "closed", data.ClosedPRs)

    return h.writeSuccess(format, traceID, start, data)
}

func (h *TestMergeHandler) checkPR(ctx context.Context, client gitea.Client, pr gitea.PR, log *slog.Logger) PRMergeResult {
    result := PRMergeResult{
        PRNumber:   pr.Number,
        HeadBranch: pr.Head,
        BaseBranch: pr.Base,
    }

    log.Debug("Checking PR", "number", pr.Number, "head", pr.Head)

    // Создаём временный PR в тестовую ветку
    testPR, err := client.CreatePR(ctx, pr.Head)
    if err != nil {
        log.Warn("Failed to create test PR", "number", pr.Number, "error", err)
        result.HasConflict = true
        result.MergeResult = "error"
        result.ErrorMessage = err.Error()
        return result
    }

    // Проверяем конфликты
    hasConflict, err := client.ConflictPR(ctx, testPR.Number)
    if err != nil {
        log.Warn("Failed to check conflict", "number", testPR.Number, "error", err)
        hasConflict = true // Assume conflict on error
    }

    if hasConflict {
        result.HasConflict = true
        result.MergeResult = "conflict"

        // Получаем список конфликтных файлов
        conflictFiles, _ := client.ConflictFilesPR(ctx, testPR.Number)
        result.ConflictFiles = conflictFiles

        // Закрываем оригинальный PR
        if closeErr := client.ClosePR(ctx, pr.Number); closeErr == nil {
            result.Closed = true
            log.Debug("Closed conflicting PR", "number", pr.Number)
        }

        return result
    }

    // Пытаемся выполнить тестовый merge
    if mergeErr := client.MergePR(ctx, testPR.Number); mergeErr != nil {
        log.Warn("Merge failed", "number", testPR.Number, "error", mergeErr)
        result.HasConflict = true
        result.MergeResult = "merge_failed"
        result.ErrorMessage = mergeErr.Error()

        // Закрываем оригинальный PR
        if closeErr := client.ClosePR(ctx, pr.Number); closeErr == nil {
            result.Closed = true
        }

        return result
    }

    result.HasConflict = false
    result.MergeResult = "success"
    return result
}
```

### Env переменные

| Переменная | Обязательность | Описание |
|------------|----------------|----------|
| BR_COMMAND | обязательно | "nr-test-merge" |
| BR_OWNER | обязательно | Владелец репозитория |
| BR_REPO | обязательно | Имя репозитория |
| BR_BASE_BRANCH | опционально | Базовая ветка (default: "main") |
| BR_OUTPUT_FORMAT | опционально | "json" для JSON вывода |

### Константы в constants.go

Добавить (если отсутствуют):
```go
// Существующие (legacy)
ActTestMerge = "test-merge"

// NR (новые)
ActNRTestMerge = "nr-test-merge"
```

### Known Limitations (наследуемые от Epic 5)

- **H-6**: Команда работает только с DI-инъекцией клиентов (тесты). Для production требуется реализация фабрики `createGiteaClient()`. Это технический долг задокументирован как TODO(H-6).
- **H-7**: Deprecated alias будет удалён в v2.0.0 / Epic 7.

### Project Structure Notes

**Новые файлы:**
- `internal/command/handlers/gitea/testmerge/handler.go` — NR handler
- `internal/command/handlers/gitea/testmerge/handler_test.go` — unit-тесты

**Зависимости от предыдущих stories:**
- Story 5-2: `internal/adapter/gitea/interfaces.go` — используем Client interface (PRReader, PRManager, BranchManager)
- Story 1-1: `internal/command/registry.go` — RegisterWithAlias
- Story 1-3: `internal/pkg/output/` — OutputWriter для JSON/Text вывода
- Story 1-5: `internal/pkg/tracing/` — TraceID generation

**НЕ изменять legacy код:**
- `internal/service/gitea_service.go:TestMerge()` — legacy service, не трогать
- `internal/app/app.go:TestMerge()` — legacy app function, не трогать

### Legacy бизнес-логика (Reference)

Изучить `internal/service/gitea_service.go:TestMerge()`:
1. Получить список активных PR через `api.ActivePR()`
2. Удалить существующую тестовую ветку (ignore error)
3. Создать новую тестовую ветку на основе BaseBranch
4. Для каждого активного PR:
   - Создать временный PR из head ветки в тестовую ветку
   - Проверить конфликты через `api.ConflictPR()`
   - Попытаться merge через `api.MergePR()`
   - Если конфликт или merge fail → закрыть оригинальный PR
5. Удалить тестовую ветку

### Тестирование

**Mock Pattern** (по образцу sonarqube handlers):
- Использовать `giteatest.MockClient` из Story 5-2
- Табличные тесты для валидации
- Интеграционные тесты с моками для полного flow

```go
func TestExecute_SomeConflicts(t *testing.T) {
    giteaClient := &giteatest.MockClient{
        ListOpenPRsFunc: func(ctx context.Context) ([]gitea.PR, error) {
            return []gitea.PR{
                {Number: 1, Head: "feature-1", Base: "main"},
                {Number: 2, Head: "feature-2", Base: "main"},
            }, nil
        },
        CreateBranchFunc: func(ctx context.Context, newBranch, baseBranch string) error {
            return nil
        },
        DeleteBranchFunc: func(ctx context.Context, branchName string) error {
            return nil
        },
        CreatePRFunc: func(ctx context.Context, head string) (gitea.PR, error) {
            return gitea.PR{Number: 100, Head: head, Base: "test-merge-branch"}, nil
        },
        ConflictPRFunc: func(ctx context.Context, prNumber int64) (bool, error) {
            // PR #100 (from feature-1) has conflict
            if prNumber == 100 {
                return true, nil
            }
            return false, nil
        },
        ConflictFilesPRFunc: func(ctx context.Context, prNumber int64) ([]string, error) {
            if prNumber == 100 {
                return []string{"src/main.go", "config.yaml"}, nil
            }
            return nil, nil
        },
        MergePRFunc: func(ctx context.Context, prNumber int64) error {
            return nil
        },
        ClosePRFunc: func(ctx context.Context, prNumber int64) error {
            return nil
        },
    }

    h := &TestMergeHandler{giteaClient: giteaClient}
    cfg := &config.Config{
        Owner:      "myorg",
        Repo:       "myrepo",
        BaseBranch: "main",
    }

    err := h.Execute(context.Background(), cfg)
    require.NoError(t, err)
}
```

### Примеры реального вывода

**JSON Output (BR_OUTPUT_FORMAT=json):**
```json
{
  "status": "success",
  "command": "nr-test-merge",
  "data": {
    "total_prs": 3,
    "mergeable_prs": 2,
    "conflict_prs": 1,
    "closed_prs": 1,
    "pr_results": [
      {
        "pr_number": 45,
        "head_branch": "feature/login",
        "base_branch": "main",
        "has_conflict": false,
        "merge_result": "success",
        "closed": false
      },
      {
        "pr_number": 47,
        "head_branch": "feature/settings",
        "base_branch": "main",
        "has_conflict": true,
        "merge_result": "conflict",
        "conflict_files": ["src/config.go", "go.mod"],
        "closed": true
      },
      {
        "pr_number": 48,
        "head_branch": "fix/typo",
        "base_branch": "main",
        "has_conflict": false,
        "merge_result": "success",
        "closed": false
      }
    ],
    "test_branch": "test-merge-branch",
    "base_branch": "main"
  },
  "metadata": {
    "duration_ms": 3245,
    "trace_id": "abc123def456",
    "api_version": "v1"
  }
}
```

**Text Output (по умолчанию):**
```
══════════════════════════════════════════════════════
📊 Проверка конфликтов слияния
══════════════════════════════════════════════════════
Репозиторий: myorg/myrepo
Базовая ветка: main

📋 Результаты проверки:

| PR # | Ветка           | Статус    | Конфликтные файлы    |
|------|-----------------|-----------|----------------------|
| #45  | feature/login   | ✅ OK      |                      |
| #47  | feature/settings| ❌ CONFLICT | src/config.go, go.mod |
| #48  | fix/typo        | ✅ OK      |                      |

══════════════════════════════════════════════════════
📈 Итого: 3 PR проверено
  ✅ Без конфликтов: 2
  ❌ С конфликтами: 1 (закрыто: 1)
══════════════════════════════════════════════════════
```

**Text Output без PR:**
```
══════════════════════════════════════════════════════
📊 Проверка конфликтов слияния
══════════════════════════════════════════════════════
Репозиторий: myorg/myrepo
Базовая ветка: main

ℹ️ Нет открытых Pull Requests для проверки.
══════════════════════════════════════════════════════
```

### Форматирование Text Output

```go
func (d *TestMergeData) writeText(w io.Writer) error {
    fmt.Fprintf(w, "══════════════════════════════════════════════════════\n")
    fmt.Fprintf(w, "📊 Проверка конфликтов слияния\n")
    fmt.Fprintf(w, "══════════════════════════════════════════════════════\n")
    fmt.Fprintf(w, "Базовая ветка: %s\n\n", d.BaseBranch)

    if d.TotalPRs == 0 {
        fmt.Fprintf(w, "ℹ️ Нет открытых Pull Requests для проверки.\n")
        fmt.Fprintf(w, "══════════════════════════════════════════════════════\n")
        return nil
    }

    fmt.Fprintf(w, "📋 Результаты проверки:\n\n")
    fmt.Fprintf(w, "| PR # | Ветка           | Статус    | Конфликтные файлы    |\n")
    fmt.Fprintf(w, "|------|-----------------|-----------|----------------------|\n")

    for _, pr := range d.PRResults {
        status := "✅ OK"
        conflictFiles := ""
        if pr.HasConflict {
            status = "❌ CONFLICT"
            if len(pr.ConflictFiles) > 0 {
                conflictFiles = strings.Join(pr.ConflictFiles, ", ")
                if len(conflictFiles) > 20 {
                    conflictFiles = conflictFiles[:17] + "..."
                }
            }
        }
        fmt.Fprintf(w, "| #%-4d | %-15s | %-9s | %-20s |\n",
            pr.PRNumber, truncateString(pr.HeadBranch, 15), status, conflictFiles)
    }

    fmt.Fprintf(w, "\n══════════════════════════════════════════════════════\n")
    fmt.Fprintf(w, "📈 Итого: %d PR проверено\n", d.TotalPRs)
    fmt.Fprintf(w, "  ✅ Без конфликтов: %d\n", d.MergeablePRs)
    fmt.Fprintf(w, "  ❌ С конфликтами: %d (закрыто: %d)\n", d.ConflictPRs, d.ClosedPRs)
    fmt.Fprintf(w, "══════════════════════════════════════════════════════\n")

    return nil
}
```

### Git Intelligence (Previous Stories Learnings)

**Story 5-3 (nr-sq-scan-branch):**
- Dual output через writeSuccess/writeError helper функции
- Коды ошибок в формате NAMESPACE.ERROR_TYPE
- Валидация cfg != nil в начале Execute
- Logging через slog с контекстными полями

**Story 5-4 (nr-sq-scan-pr):**
- Проверка nil клиентов для graceful error handling
- Тесты TestExecute_NilConfig
- shortSHA для отображения (защита от panic при truncation)

**Story 5-6 (nr-sq-project-update):**
- Graceful handling случаев когда часть данных недоступна
- TODO(H-7) для deprecated aliases
- Unicode-aware string operations

**Story 5-2 (Gitea Adapter):**
- PRReader: ListOpenPRs, ConflictPR, ConflictFilesPR
- PRManager: CreatePR, MergePR, ClosePR
- BranchManager: CreateBranch, DeleteBranch

### References

- [Source: internal/command/handlers/sonarqube/scanpr/handler.go] — образец NR handler, переиспользовать паттерны
- [Source: internal/command/handlers/sonarqube/projectupdate/handler.go] — образец NR handler с graceful errors
- [Source: internal/command/registry.go] — RegisterWithAlias pattern
- [Source: internal/adapter/gitea/interfaces.go:261-271] — PRReader interface (ListOpenPRs, ConflictPR, ConflictFilesPR)
- [Source: internal/adapter/gitea/interfaces.go:327-337] — PRManager interface (CreatePR, MergePR, ClosePR)
- [Source: internal/adapter/gitea/interfaces.go:299-307] — BranchManager interface (CreateBranch, DeleteBranch)
- [Source: internal/service/gitea_service.go:37-167] — legacy TestMerge implementation (бизнес-логика)
- [Source: internal/app/app.go:1279-1306] — legacy app function (вызов)
- [Source: _bmad-output/project-planning-artifacts/architecture.md#Pattern-Command-Registry] — архитектурный паттерн
- [Source: _bmad-output/project-planning-artifacts/epics/epic-5-quality-integration.md#Story-5.7] — исходные требования (FR26)

## Review Follow-ups (AI Code Review #34)

- [ ] [AI-Review][HIGH] Автоматическое закрытие PR при конфликте — деструктивная операция без подтверждения [handler.go:358-370]
- [ ] [AI-Review][HIGH] giteaClient nil — нет фабрики (TODO H-6) [handler.go:238-246]
- [ ] [AI-Review][MEDIUM] Нет проверки context.Done() в цикле PR [handler.go:299-311]
- [ ] [AI-Review][MEDIUM] Тестовые PR не закрываются при ошибке CreatePR [handler.go:333-341]
- [ ] [AI-Review][MEDIUM] generateTestBranchName с секундной точностью — race condition при параллельном CI [handler.go:29-31]

## Dev Agent Record

### Agent Model Used

Claude Opus 4.5 (claude-opus-4-5-20251101)

### Debug Log References

- Все 24 unit-теста проходят успешно (было 22, добавлено 2 после code review)
- go vet без ошибок
- go build успешен

### Completion Notes List

1. Создан handler для команды nr-test-merge с full dual output (JSON/Text)
2. Реализована self-registration через init() с deprecated alias "test-merge"
3. Добавлена константа ActNRTestMerge в constants.go
4. Покрыты все AC #1-#11 через unit-тесты
5. Паттерн handler соответствует существующим NR-handlers (scanpr, projectupdate)
6. Обновлён тест TestCommandRegistry_LegacyFallback — удалены sq-* и test-merge из legacy списка (они мигрированы в NR)

### Code Review Fixes (2026-02-05)

**HIGH fixes:**
- [HIGH-1] AC #6: Добавлен комментарий при закрытии PR с описанием конфликта и списком файлов
- [HIGH-2/3] AC #7: Добавлен тест TestExecute_JSONOutput_Structure для проверки JSON структуры

**MEDIUM fixes:**
- [MEDIUM-1] truncateString переписан с unicode/utf8 для корректной работы с Unicode
- [MEDIUM-3] Добавлен тест TestExecute_MergeFailure_WithComment для проверки комментария
- [MEDIUM-4] testBranchName заменён на generateTestBranchName() с timestamp

**LOW fixes:**
- [LOW-3] PRResults теперь возвращает [] вместо null для пустого списка PR

### File List

**Созданы:**
- internal/command/handlers/gitea/testmerge/handler.go (NR handler)
- internal/command/handlers/gitea/testmerge/handler_test.go (22 unit-теста)

**Изменены:**
- internal/constants/constants.go (добавлена ActNRTestMerge)
- cmd/apk-ci/main_test.go (обновлён TestCommandRegistry_LegacyFallback)
- _bmad-output/implementation-artifacts/sprint-artifacts/sprint-status.yaml (status: in-progress → review)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-05 | Claude Opus 4.5 | Реализована NR-команда nr-test-merge с полным покрытием AC #1-#11, 22 unit-теста |
| 2026-02-05 | Claude Opus 4.5 | Code Review: исправлены HIGH/MEDIUM issues (AC #6 комментарий, Unicode truncate, timestamp ветки), 24 теста |

