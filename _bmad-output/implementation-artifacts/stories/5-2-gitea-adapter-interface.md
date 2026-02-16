# Story 5.2: Gitea Adapter Interface

Status: done

## Story

As a разработчик,
I want иметь абстракцию над Gitea API,
so that я могу тестировать без реального сервера.

## Acceptance Criteria

1. [AC1] Interface GiteaClient определён в `internal/adapter/gitea/interfaces.go`
2. [AC2] Role-based interfaces разделены по ISP: PRReader, CommitReader, FileReader, BranchManager, ReleaseReader
3. [AC3] Можно подставить mock для тестов (интерфейсы соответствуют ISP)
4. [AC4] Композитный интерфейс Client объединяет все role-based интерфейсы
5. [AC5] Типизированные ошибки с кодами: ErrGiteaConnect, ErrGiteaAPI, ErrGiteaAuth, ErrGiteaNotFound

## Tasks / Subtasks

- [x] Task 1: Создать файл `internal/adapter/gitea/interfaces.go` (AC: #1, #2, #4)
  - [x] Subtask 1.1: Определить структуры данных (Repository, PR, PRResponse, Commit, Branch, Issue, FileInfo, Release)
  - [x] Subtask 1.2: Определить PRReader interface (GetPR, ListOpenPRs, ConflictPR, ConflictFilesPR)
  - [x] Subtask 1.3: Определить CommitReader interface (GetCommits, GetLatestCommit, GetCommitFiles, GetCommitsBetween, GetFirstCommitOfBranch, GetBranchCommitRange)
  - [x] Subtask 1.4: Определить FileReader interface (GetFileContent, GetRepositoryContents, AnalyzeProjectStructure)
  - [x] Subtask 1.5: Определить BranchManager interface (GetBranches, CreateBranch, DeleteBranch)
  - [x] Subtask 1.6: Определить ReleaseReader interface (GetLatestRelease, GetReleaseByTag)
  - [x] Subtask 1.7: Определить IssueManager interface (GetIssue, AddIssueComment, CloseIssue)
  - [x] Subtask 1.8: Определить PRManager interface (CreatePR, CreatePRWithOptions, MergePR, ClosePR)
  - [x] Subtask 1.9: Определить RepositoryWriter interface (SetRepositoryState для batch операций)
  - [x] Subtask 1.10: Определить TeamReader interface (IsUserInTeam, GetTeamMembers)
  - [x] Subtask 1.11: Определить OrgReader interface (SearchOrgRepos)
  - [x] Subtask 1.12: Определить композитный Client interface объединяющий все role-based интерфейсы
- [x] Task 2: Создать файл `internal/adapter/gitea/errors.go` (AC: #5)
  - [x] Subtask 2.1: Определить константы кодов ошибок (ErrGiteaConnect, ErrGiteaAPI, ErrGiteaAuth, ErrGiteaTimeout, ErrGiteaNotFound, ErrGiteaValidation)
  - [x] Subtask 2.2: Определить типы ошибок GiteaError и ValidationError
  - [x] Subtask 2.3: Реализовать Unwrap() для поддержки errors.Is/As
  - [x] Subtask 2.4: Добавить хелперы IsNotFoundError, IsAuthError, IsTimeoutError, IsConnectionError
- [x] Task 3: Написать тесты для интерфейсов (AC: #3)
  - [x] Subtask 3.1: Создать `internal/adapter/gitea/giteatest/mock.go` с MockClient для тестирования
  - [x] Subtask 3.2: Добавить compile-time проверки реализации интерфейсов
  - [x] Subtask 3.3: Создать вспомогательные конструкторы (NewMockClient, NewMockClientWithPR, NewMockClientWithCommits)
  - [x] Subtask 3.4: Создать тестовые данные (PRData, CommitData, BranchData)
  - [x] Subtask 3.5: Создать doc.go для giteatest пакета
  - [x] Subtask 3.6: Создать interfaces_test.go с проверкой ISP композиции
  - [x] Subtask 3.7: Создать errors_test.go для тестирования ошибок
  - [x] Subtask 3.8: Создать mock_test.go с примерами ISP использования

### Review Follow-ups (AI)

- [ ] [AI-Review][HIGH] Нет production-реализации Client — все хендлеры возвращают "Gitea клиент не настроен" [interfaces.go:359-371]
- [ ] [AI-Review][MEDIUM] Дублирование кода ошибок между пакетами — GiteaError и SonarQubeError идентичны по структуре, 2x164 строки [errors.go:24-34]
- [ ] [AI-Review][MEDIUM] ListOpenPRs без пагинации — для репозиториев с 100+ PR результат усечён [interfaces.go:265-266]
- [ ] [AI-Review][MEDIUM] BranchManager.GetBranches принимает repo string — единственный метод требующий имя репозитория [interfaces.go:300-307]
- [ ] [AI-Review][LOW] MockClient.GetPR: PR.ID == prNumber — в реальном API ID и Number разные значения [giteatest/mock.go:85-96]
- [ ] [AI-Review][LOW] CommitAuthor.Date — string вместо time.Time — несогласованность с sonarqube пакетом [interfaces.go:112-115]

## Dev Notes

### Архитектурные паттерны и ограничения

**Interface Segregation Principle (ISP):**
- Разбить на сфокусированные интерфейсы по ролям операций с Gitea API
- Композитный интерфейс Client объединяет все role-based интерфейсы
- Следовать паттерну SonarQube adapter: [Source: internal/adapter/sonarqube/interfaces.go]
- Следовать паттерну RAC adapter: [Source: internal/adapter/onec/rac/interfaces.go]
- Следовать паттерну MSSQL adapter: [Source: internal/adapter/mssql/interfaces.go]

**ADR-003: Role-based Interface Segregation** [Source: _bmad-output/project-planning-artifacts/architecture.md#ADR-003]
- Широкие интерфейсы нарушают ISP — разделить на role-based интерфейсы
- Меньше зависимостей, проще мокирование

### Структура интерфейсов (ISP-compliant)

```go
// PRReader предоставляет операции для чтения информации о Pull Requests.
type PRReader interface {
    GetPR(ctx context.Context, prNumber int64) (*PRResponse, error)
    ListOpenPRs(ctx context.Context) ([]PR, error)
    ConflictPR(ctx context.Context, prNumber int64) (bool, error)
    ConflictFilesPR(ctx context.Context, prNumber int64) ([]string, error)
}

// CommitReader предоставляет операции для чтения информации о коммитах.
type CommitReader interface {
    GetCommits(ctx context.Context, branch string, limit int) ([]Commit, error)
    GetLatestCommit(ctx context.Context, branch string) (*Commit, error)
    GetCommitFiles(ctx context.Context, commitSHA string) ([]CommitFile, error)
    GetCommitsBetween(ctx context.Context, baseCommitSHA, headCommitSHA string) ([]Commit, error)
    GetFirstCommitOfBranch(ctx context.Context, branch, baseBranch string) (*Commit, error)
    GetBranchCommitRange(ctx context.Context, branch string) (*BranchCommitRange, error)
}

// FileReader предоставляет операции для чтения файлов из репозитория.
type FileReader interface {
    GetFileContent(ctx context.Context, fileName string) ([]byte, error)
    GetRepositoryContents(ctx context.Context, filepath, branch string) ([]FileInfo, error)
    AnalyzeProjectStructure(ctx context.Context, branch string) ([]string, error)
}

// BranchManager предоставляет операции для управления ветками.
type BranchManager interface {
    GetBranches(ctx context.Context, repo string) ([]Branch, error)
    CreateBranch(ctx context.Context) error
    DeleteBranch(ctx context.Context) error
}

// ReleaseReader предоставляет операции для чтения информации о релизах.
type ReleaseReader interface {
    GetLatestRelease(ctx context.Context) (*Release, error)
    GetReleaseByTag(ctx context.Context, tag string) (*Release, error)
}

// IssueManager предоставляет операции для работы с задачами.
type IssueManager interface {
    GetIssue(ctx context.Context, issueNumber int64) (*Issue, error)
    AddIssueComment(ctx context.Context, issueNumber int64, commentText string) error
    CloseIssue(ctx context.Context, issueNumber int64) error
}

// PRManager предоставляет операции для управления Pull Requests.
type PRManager interface {
    CreatePR(ctx context.Context, head string) (PR, error)
    CreatePRWithOptions(ctx context.Context, opts CreatePROptions) (*PRResponse, error)
    MergePR(ctx context.Context, prNumber int64) error
    ClosePR(ctx context.Context, prNumber int64) error
}

// RepositoryWriter предоставляет операции для записи в репозиторий.
type RepositoryWriter interface {
    SetRepositoryState(ctx context.Context, operations []BatchOperation, branch, commitMessage string) error
}

// TeamReader предоставляет операции для чтения информации о командах.
type TeamReader interface {
    IsUserInTeam(ctx context.Context, username, orgName, teamName string) (bool, error)
    GetTeamMembers(ctx context.Context, orgName, teamName string) ([]string, error)
}

// OrgReader предоставляет операции для чтения информации об организациях.
type OrgReader interface {
    SearchOrgRepos(ctx context.Context, orgName string) ([]Repository, error)
}

// Client — композитный интерфейс, объединяющий все операции Gitea.
type Client interface {
    PRReader
    CommitReader
    FileReader
    BranchManager
    ReleaseReader
    IssueManager
    PRManager
    RepositoryWriter
    TeamReader
    OrgReader
}
```

### Структуры данных

Переиспользовать и адаптировать из legacy `internal/entity/gitea/gitea.go`:
- PR, PRData, PRResponse, Branch, Commit, CommitFile, BranchCommitRange
- Issue, FileInfo, Release, ReleaseAsset, Repository
- CreatePROptions, BatchOperation, ChangeFileOperation

**ВАЖНО:** Структуры должны повторять JSON-теги из legacy для совместимости с Gitea API.

### Коды ошибок

```go
const (
    ErrGiteaConnect    = "GITEA.CONNECT_FAILED"
    ErrGiteaAPI        = "GITEA.API_FAILED"
    ErrGiteaAuth       = "GITEA.AUTH_FAILED"
    ErrGiteaTimeout    = "GITEA.TIMEOUT"
    ErrGiteaNotFound   = "GITEA.NOT_FOUND"
    ErrGiteaValidation = "GITEA.VALIDATION_FAILED"
)
```

### Тестирование

**Mock Pattern** (по образцу sonarqubetest/MockClient):
- Создать `giteatest/mock.go`
- Использовать функциональные поля для каждого метода
- Позволяет настраивать поведение mock в каждом тесте
- Compile-time проверки реализации интерфейсов

```go
type MockClient struct {
    // PRReader
    GetPRFunc         func(ctx context.Context, prNumber int64) (*PRResponse, error)
    ListOpenPRsFunc   func(ctx context.Context) ([]PR, error)
    ConflictPRFunc    func(ctx context.Context, prNumber int64) (bool, error)
    ConflictFilesPRFunc func(ctx context.Context, prNumber int64) ([]string, error)
    // ... остальные методы
}
```

### Git Intelligence (Previous Story 5-1 Learnings)

История 5-1 (SonarQube Adapter Interface) была успешно реализована. Ключевые уроки:

1. **ISP Composition Test:** Добавить `interfaces_test.go` с проверкой, что композитный Client включает все role-based интерфейсы
2. **Error Helpers с errors.As:** Все `IsXxxError` функции должны использовать `errors.As` для поддержки wrapped errors
3. **ValidationError.Unwrap():** ValidationError также должен поддерживать Unwrap() для цепочки ошибок
4. **doc.go для тестового пакета:** Создать `giteatest/doc.go` с описанием пакета
5. **Example тесты с Output:** Example функции должны иметь `// Output:` комментарии для проверки

### Маппинг legacy API -> NR интерфейсов

| Legacy метод | NR Interface | NR метод |
|--------------|--------------|----------|
| GetIssue | IssueManager | GetIssue |
| GetFileContent | FileReader | GetFileContent |
| AddIssueComment | IssueManager | AddIssueComment |
| CloseIssue | IssueManager | CloseIssue |
| ConflictPR | PRReader | ConflictPR |
| ConflictFilesPR | PRReader | ConflictFilesPR |
| GetRepositoryContents | FileReader | GetRepositoryContents |
| AnalyzeProjectStructure | FileReader | AnalyzeProjectStructure |
| GetLatestCommit | CommitReader | GetLatestCommit |
| GetCommitFiles | CommitReader | GetCommitFiles |
| GetLatestRelease | ReleaseReader | GetLatestRelease |
| GetReleaseByTag | ReleaseReader | GetReleaseByTag |
| IsUserInTeam | TeamReader | IsUserInTeam |
| GetCommits | CommitReader | GetCommits |
| GetFirstCommitOfBranch | CommitReader | GetFirstCommitOfBranch |
| GetCommitsBetween | CommitReader | GetCommitsBetween |
| GetBranchCommitRange | CommitReader | GetBranchCommitRange |
| ActivePR | PRReader | ListOpenPRs |
| DeleteTestBranch | BranchManager | DeleteBranch |
| CreateTestBranch | BranchManager | CreateBranch |
| CreatePR | PRManager | CreatePR |
| CreatePRWithOptions | PRManager | CreatePRWithOptions |
| MergePR | PRManager | MergePR |
| ClosePR | PRManager | ClosePR |
| SetRepositoryState | RepositoryWriter | SetRepositoryState |
| GetTeamMembers | TeamReader | GetTeamMembers |
| GetBranches | BranchManager | GetBranches |
| SearchOrgRepos | OrgReader | SearchOrgRepos |

**НЕ включать в NR интерфейсы (legacy-specific с slog зависимостью):**
- `GetConfigData(l *slog.Logger, ...)` — используется только для загрузки конфигурации
- `AnalyzeProject(l *slog.Logger, ...)` — дублирует AnalyzeProjectStructure
- `MergePR(prNumber int64, l *slog.Logger)` — логгер передаётся через контекст в NR

### Project Structure Notes

**Новые файлы:**
- `internal/adapter/gitea/interfaces.go` — ISP-compliant интерфейсы
- `internal/adapter/gitea/errors.go` — типы и коды ошибок
- `internal/adapter/gitea/errors_test.go` — тесты ошибок
- `internal/adapter/gitea/interfaces_test.go` — тесты ISP композиции
- `internal/adapter/gitea/giteatest/mock.go` — mock для тестов
- `internal/adapter/gitea/giteatest/mock_test.go` — тесты mock и примеры ISP
- `internal/adapter/gitea/giteatest/doc.go` — документация пакета

**НЕ изменять legacy код:**
- `internal/entity/gitea/` — оставить как есть до полной миграции

### References

- [Source: internal/adapter/sonarqube/interfaces.go] — образец ISP-compliant интерфейсов (Story 5-1)
- [Source: internal/adapter/sonarqube/errors.go] — образец типов ошибок (Story 5-1)
- [Source: internal/adapter/sonarqube/sonarqubetest/mock.go] — образец mock pattern (Story 5-1)
- [Source: internal/adapter/onec/rac/interfaces.go] — образец ISP-compliant интерфейсов (Story 2-1)
- [Source: internal/adapter/mssql/interfaces.go] — образец ISP-compliant интерфейсов (Story 3-1)
- [Source: internal/entity/gitea/interfaces.go] — legacy интерфейсы для маппинга
- [Source: internal/entity/gitea/gitea.go] — legacy структуры данных для переиспользования
- [Source: _bmad-output/project-planning-artifacts/architecture.md#ADR-003] — ADR для ISP
- [Source: _bmad-output/project-planning-artifacts/epics/epic-5-quality-integration.md#Story-5.2] — исходные требования

### Review Follow-ups (AI Code Review #34)

- [ ] [AI-Review][HIGH] Нет production-реализации Client — хендлеры возвращают "не настроен" (TODO H-6) [interfaces.go:359-371]
- [ ] [AI-Review][HIGH] MockClient.GetPR: PR.ID == prNumber — в реальном API ID и Number разные [giteatest/mock.go:85-96]
- [ ] [AI-Review][MEDIUM] ListOpenPRs без пагинации — для 100+ PR результат усечён [interfaces.go:265-266]
- [ ] [AI-Review][MEDIUM] BranchManager.GetBranches принимает repo string — нарушает ISP [interfaces.go:300-307]
- [ ] [AI-Review][MEDIUM] Дублирование кода ошибок между sonarqube и gitea пакетами (~164 строки) [errors.go:24-34]

## Dev Agent Record

### Agent Model Used

Claude Opus 4.5 (claude-opus-4-5-20251101)

### Debug Log References

### Completion Notes List

- Реализован полный набор ISP-compliant интерфейсов для Gitea API по образцу SonarQube adapter
- Создано 10 role-based интерфейсов: PRReader, CommitReader, FileReader, BranchManager, ReleaseReader, IssueManager, PRManager, RepositoryWriter, TeamReader, OrgReader
- Композитный интерфейс Client объединяет все role-based интерфейсы
- Типизированные ошибки с кодами и helper функциями для проверки типа ошибок
- MockClient с функциональными полями для гибкого тестирования
- Compile-time проверки реализации всех интерфейсов
- Тестовые данные и конструкторы для типовых сценариев
- Example функции с проверяемым Output
- Все тесты проходят успешно

**Code Review #1 исправления:**
- H-1: BranchManager.CreateBranch/DeleteBranch — добавлены параметры (newBranch, baseBranch) и (branchName)
- H-3: Удалён бесполезный TestInterfaceMethodSignatures (compile-time проверки в mock.go)
- M-1: Добавлен IsAPIError helper для проверки ErrGiteaAPI
- M-2: Добавлены тесты для IsAPIError
- L-1: Добавлен doc.go для основного пакета с полной документацией

### Change Log

- 2026-02-04: Реализована Story 5-2 Gitea Adapter Interface
- 2026-02-04: Code Review #1 — исправлены 3 HIGH + 2 MEDIUM issues

### File List

- internal/adapter/gitea/interfaces.go (new)
- internal/adapter/gitea/errors.go (new)
- internal/adapter/gitea/errors_test.go (new)
- internal/adapter/gitea/interfaces_test.go (new)
- internal/adapter/gitea/doc.go (new) — добавлен в code review
- internal/adapter/gitea/giteatest/mock.go (new)
- internal/adapter/gitea/giteatest/mock_test.go (new)
- internal/adapter/gitea/giteatest/doc.go (new)
- _bmad-output/implementation-artifacts/sprint-artifacts/sprint-status.yaml (modified)
