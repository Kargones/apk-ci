# Архитектура Epic-0: Extension Publish

**Версия:** 1.1
**Дата:** 2025-12-08
**Автор:** Уинстон (Architect Agent)
**Статус:** Draft

---

## 1. Обзор

### 1.1 Цель

Команда `extension-publish` автоматизирует распространение обновлений расширений 1C между репозиториями. При создании релиза в репозитории расширения система находит все "подписанные" репозитории и создаёт PR с обновлённой версией.

### 1.2 Триггер

```yaml
on:
  release:
    types: [published]
```

### 1.3 Принципы проектирования

- **Boring Technology**: Использование существующих паттернов Gitea API
- **Fail-Safe**: Ошибка в одном репозитории не блокирует остальные
- **Transparency**: Детальное логирование и итоговый отчёт
- **Idempotency**: Повторный запуск безопасен

---

## 2. Диаграмма компонентов

```mermaid
flowchart TB
    subgraph Runner["Gitea Actions Runner"]
        subgraph BR["apk-ci"]
            Main["main.go<br/>(Entry Point)"]
            Registry["Command Handler<br/>(Registry)"]
            ExtService["ExtensionPublish<br/>Service"]

            subgraph Capabilities["Gitea API Capabilities"]
                ReleaseReader["ReleaseReader"]
                RepoSearcher["RepoSearcher"]
                ContentSyncer["ContentSyncer"]
                PRCreator["PRCreator"]
            end

            GiteaEntity["Gitea API<br/>(entity)"]
        end
    end

    subgraph GiteaServer["Gitea Server"]
        ReleasesAPI["Releases API"]
        ReposSearch["Repos Search API"]
        ContentsAPI["Contents API"]
        PRAPI["PR API"]
    end

    Main -->|"BR_COMMAND=<br/>extension-publish"| Registry
    Registry --> ExtService
    ExtService --> ReleaseReader
    ExtService --> RepoSearcher
    ExtService --> ContentSyncer
    ExtService --> PRCreator

    ReleaseReader --> GiteaEntity
    RepoSearcher --> GiteaEntity
    ContentSyncer --> GiteaEntity
    PRCreator --> GiteaEntity

    GiteaEntity -->|"HTTP/REST"| ReleasesAPI
    GiteaEntity -->|"HTTP/REST"| ReposSearch
    GiteaEntity -->|"HTTP/REST"| ContentsAPI
    GiteaEntity -->|"HTTP/REST"| PRAPI

    style Runner fill:#e1f5fe
    style BR fill:#b3e5fc
    style GiteaServer fill:#fff3e0
    style Capabilities fill:#e8f5e9
```

### 2.1 C4 Context Diagram

```mermaid
C4Context
    title Контекст системы Extension Publish

    Person(dev, "DevOps Engineer", "Создаёт релизы расширений")

    System(br, "apk-ci", "CLI инструмент автоматизации")

    System_Ext(gitea, "Gitea Server", "Git-репозитории,<br/>Releases, PRs")
    System_Ext(ga, "Gitea Actions", "CI/CD runner")

    Rel(dev, gitea, "Создаёт релиз", "Web UI")
    Rel(gitea, ga, "Триггер on:release", "Webhook")
    Rel(ga, br, "Запускает", "BR_COMMAND")
    Rel(br, gitea, "Release API,<br/>Search API,<br/>Contents API,<br/>PR API", "REST/HTTP")

    UpdateLayoutConfig($c4ShapeInRow="3", $c4BoundaryInRow="1")
```

### 2.2 Компоненты

| Компонент | Пакет | Ответственность |
|-----------|-------|-----------------|
| Entry Point | `cmd/apk-ci/main.go` | Маршрутизация команды |
| ExtensionPublish Service | `internal/app/extension_publish.go` | Оркестрация процесса публикации |
| ReleaseReader | `internal/entity/gitea/` | Получение информации о релизах |
| RepoSearcher | `internal/entity/gitea/` | Поиск репозиториев и веток-подписок |
| ContentSyncer | `internal/entity/gitea/` | Синхронизация файлов между репозиториями |
| PRCreator | `internal/entity/gitea/` | Создание Pull Request с информацией |

---

## 3. Схема данных

### 3.0 ER-диаграмма

```mermaid
erDiagram
    Release ||--o{ ReleaseAsset : contains
    Release ||--|| User : author
    Repository ||--|| User : owner
    Repository ||--o{ Branch : has
    Subscription ||--|| Repository : targets
    PublishReport ||--o{ PublishResult : contains
    PublishResult ||--|| Subscription : for

    Release {
        int64 id PK
        string tag_name
        string name
        string body
        bool draft
        bool prerelease
        time created_at
        time published_at
        string html_url
    }

    ReleaseAsset {
        int64 id PK
        string name
        int64 size
        int64 download_count
        string download_url
    }

    User {
        int64 id PK
        string login
        string full_name
        string email
    }

    Repository {
        int64 id PK
        string name
        string full_name
        string default_branch
        bool private
    }

    Branch {
        string name PK
        string commit_sha
    }

    Subscription {
        string owner
        string repo
        string branch_name
        string target_dir
        string ext_name
    }

    PublishResult {
        bool success
        int64 pr_number
        string pr_url
        string error
    }

    PublishReport {
        string source_repo
        string version
        int total_targets
    }
```

### 3.1 Структура Release (NEW)

```go
// internal/entity/gitea/types.go

// Release представляет релиз в Gitea
type Release struct {
    ID          int64         `json:"id"`
    TagName     string        `json:"tag_name"`
    Name        string        `json:"name"`
    Body        string        `json:"body"`
    Draft       bool          `json:"draft"`
    Prerelease  bool          `json:"prerelease"`
    CreatedAt   time.Time     `json:"created_at"`
    PublishedAt time.Time     `json:"published_at"`
    Author      *User         `json:"author"`
    Assets      []ReleaseAsset `json:"assets"`
    HTMLURL     string        `json:"html_url"`
    TarballURL  string        `json:"tarball_url"`
    ZipballURL  string        `json:"zipball_url"`
}

// ReleaseAsset представляет файл, прикреплённый к релизу
type ReleaseAsset struct {
    ID            int64     `json:"id"`
    Name          string    `json:"name"`
    Size          int64     `json:"size"`
    DownloadCount int64     `json:"download_count"`
    CreatedAt     time.Time `json:"created_at"`
    UUID          string    `json:"uuid"`
    DownloadURL   string    `json:"browser_download_url"`
}

// User представляет пользователя Gitea
type User struct {
    ID        int64  `json:"id"`
    Login     string `json:"login"`
    FullName  string `json:"full_name"`
    Email     string `json:"email"`
    AvatarURL string `json:"avatar_url"`
}
```

### 3.2 Структура Subscription (подписка)

```go
// internal/app/extension_publish.go

// Subscription представляет подписку репозитория на расширение
type Subscription struct {
    // Целевой репозиторий
    Owner       string
    Repo        string
    // Ветка подписки в целевом репозитории
    BranchName  string
    // Целевая директория для расширения
    TargetDir   string
    // Оригинальное имя расширения
    ExtName     string
}

// PublishResult представляет результат публикации в один репозиторий
type PublishResult struct {
    Subscription Subscription
    Success      bool
    PRNumber     int64
    PRURL        string
    Error        error
}

// PublishReport представляет итоговый отчёт о публикации
type PublishReport struct {
    SourceRepo   string
    Version      string
    TotalTargets int
    Successful   []PublishResult
    Failed       []PublishResult
    Skipped      []PublishResult
}
```

### 3.3 Структура Repository (расширение существующей)

```go
// internal/entity/gitea/types.go (дополнение)

// Repository представляет репозиторий Gitea (расширенная версия)
type Repository struct {
    ID            int64  `json:"id"`
    Owner         *User  `json:"owner"`
    Name          string `json:"name"`
    FullName      string `json:"full_name"`
    Description   string `json:"description"`
    Private       bool   `json:"private"`
    Fork          bool   `json:"fork"`
    DefaultBranch string `json:"default_branch"`
    HTMLURL       string `json:"html_url"`
    CloneURL      string `json:"clone_url"`
    SSHURL        string `json:"ssh_url"`
}

// SearchReposResult представляет результат поиска репозиториев
type SearchReposResult struct {
    OK   bool         `json:"ok"`
    Data []Repository `json:"data"`
}
```

---

## 4. API Контракты

### 4.1 Новые методы Gitea API

#### 4.1.1 GetLatestRelease

```go
// GetLatestRelease получает последний релиз репозитория.
// Gitea API: GET /repos/{owner}/{repo}/releases/latest
//
// Параметры:
//   - (использует g.Owner, g.Repo из конфигурации API)
//
// Возвращает:
//   - *Release: последний опубликованный релиз
//   - error: ошибка если релизы не найдены или API недоступен
func (g *API) GetLatestRelease() (*Release, error)
```

**Пример ответа Gitea API:**
```json
{
    "id": 123,
    "tag_name": "v1.2.0",
    "name": "Release 1.2.0",
    "body": "## Changes\n- Feature X\n- Bug fix Y",
    "draft": false,
    "prerelease": false,
    "created_at": "2025-01-15T10:00:00Z",
    "published_at": "2025-01-15T10:30:00Z",
    "author": {
        "id": 1,
        "login": "developer"
    },
    "html_url": "https://gitea.example.com/org/repo/releases/tag/v1.2.0"
}
```

#### 4.1.2 GetReleaseByTag

```go
// GetReleaseByTag получает релиз по имени тега.
// Gitea API: GET /repos/{owner}/{repo}/releases/tags/{tag}
//
// Параметры:
//   - tag: имя тега релиза (например, "v1.2.0")
//
// Возвращает:
//   - *Release: релиз с указанным тегом
//   - error: ошибка если тег не найден
func (g *API) GetReleaseByTag(tag string) (*Release, error)
```

#### 4.1.3 SearchAllRepos

```go
// SearchAllRepos ищет все репозитории с поддержкой пагинации.
// Gitea API: GET /repos/search
//
// Параметры:
//   - opts: опции поиска (организация, лимит, страница)
//
// Возвращает:
//   - []Repository: найденные репозитории
//   - error: ошибка поиска
func (g *API) SearchAllRepos(opts SearchReposOptions) ([]Repository, error)

// SearchReposOptions определяет опции поиска репозиториев
type SearchReposOptions struct {
    // Владелец (организация или пользователь)
    Owner string
    // Лимит результатов на страницу (по умолчанию 50, макс 50)
    Limit int
    // Номер страницы (начиная с 1)
    Page int
    // Включать приватные репозитории
    IncludePrivate bool
}
```

**Пример ответа Gitea API:**
```json
{
    "ok": true,
    "data": [
        {
            "id": 1,
            "owner": {"login": "MyOrg"},
            "name": "MyProject",
            "full_name": "MyOrg/MyProject",
            "default_branch": "main"
        }
    ]
}
```

#### 4.1.4 FindSubscribedRepos

```go
// FindSubscribedRepos находит все репозитории, подписанные на расширение.
// Паттерн ветки подписки: {SourceOrg}_{SourceRepo}_{ExtDir}
//
// Параметры:
//   - sourceOrg: организация исходного репозитория
//   - sourceRepo: имя исходного репозитория
//   - extDir: директория расширения (например, "cfe/myext")
//
// Возвращает:
//   - []Subscription: список подписок
//   - error: ошибка поиска
func (g *API) FindSubscribedRepos(sourceOrg, sourceRepo, extDir string) ([]Subscription, error)
```

**Логика поиска:**

```mermaid
flowchart LR
    subgraph Source["Исходный репозиторий"]
        SrcRepo["Extensions/CommonExt"]
        SrcDir["cfe/common/"]
    end

    subgraph Target1["Целевой репозиторий 1"]
        TgtRepo1["MyOrg/ProjectA"]
        Branch1["Extensions_CommonExt_cfe/common"]
        TgtDir1["cfe/common/"]
    end

    subgraph Target2["Целевой репозиторий 2"]
        TgtRepo2["MyOrg/ProjectB"]
        Branch2["Extensions_CommonExt_cfe/common"]
        TgtDir2["cfe/common/"]
    end

    SrcRepo -->|"Релиз v1.2.0"| Search["SearchAllRepos()"]
    Search -->|"GetBranches()"| Match["Pattern Match:<br/>{Org}_{Repo}_{ExtDir}"]
    Match -->|"Подписка найдена"| Branch1
    Match -->|"Подписка найдена"| Branch2

    SrcDir -.->|"Sync files"| TgtDir1
    SrcDir -.->|"Sync files"| TgtDir2

    style Source fill:#e8f5e9
    style Target1 fill:#e3f2fd
    style Target2 fill:#e3f2fd
```

**Алгоритм:**
1. Получить все репозитории организации через `SearchAllRepos()`
2. Для каждого репозитория получить список веток через `GetBranches()`
3. Найти ветки с паттерном: `{sourceOrg}_{sourceRepo}_{extDir}`
4. Распарсить имя ветки для определения target directory
5. Вернуть список `Subscription`

#### 4.1.5 CreatePRWithBody (расширение существующего CreatePR)

```go
// CreatePRWithBody создаёт Pull Request с полной информацией.
// Gitea API: POST /repos/{owner}/{repo}/pulls
//
// Параметры:
//   - opts: опции создания PR
//
// Возвращает:
//   - *PRData: созданный PR с номером и URL
//   - error: ошибка создания
func (g *API) CreatePRWithBody(opts CreatePROptions) (*PRData, error)

// CreatePROptions определяет опции создания PR
type CreatePROptions struct {
    // Целевая ветка (куда вливаем)
    Base string
    // Исходная ветка (откуда вливаем)
    Head string
    // Заголовок PR
    Title string
    // Тело PR (markdown)
    Body string
    // Assignees (login пользователей)
    Assignees []string
    // Labels
    Labels []int64
}
```

### 4.2 Расширение интерфейса APIInterface

```go
// internal/entity/gitea/interfaces.go (дополнение)

type APIInterface interface {
    // ... существующие методы ...

    // Методы для работы с релизами
    GetLatestRelease() (*Release, error)
    GetReleaseByTag(tag string) (*Release, error)

    // Методы для поиска репозиториев
    SearchAllRepos(opts SearchReposOptions) ([]Repository, error)
    FindSubscribedRepos(sourceOrg, sourceRepo, extDir string) ([]Subscription, error)

    // Расширенное создание PR
    CreatePRWithBody(opts CreatePROptions) (*PRData, error)
}
```

---

## 5. Точки интеграции

### 5.1 Интеграция с существующим кодом

| Точка интеграции | Файл | Метод | Использование |
|------------------|------|-------|---------------|
| Маршрутизация команд | `cmd/apk-ci/main.go` | switch | Добавить case `CmdExtensionPublish` |
| Константы | `internal/constants/constants.go` | — | Добавить `ActExtensionPublish` |
| Config | `internal/config/config.go` | — | Использовать существующую конфигурацию Gitea |
| Gitea API | `internal/entity/gitea/gitea.go` | Новые методы | Release API, Search API |
| Gitea Types | `internal/entity/gitea/types.go` | NEW | Release, ReleaseAsset, Repository |
| Batch Operations | `internal/entity/gitea/gitea.go` | `SetRepositoryState()` | Синхронизация файлов |
| Branch Operations | `internal/entity/gitea/gitea.go` | `GetBranches()` | Поиск веток-подписок |
| PR Creation | `internal/entity/gitea/gitea.go` | `CreatePR()` → `CreatePRWithBody()` | Создание PR с body |

### 5.2 Схема интеграции

```mermaid
flowchart TB
    subgraph Existing["Существующий код"]
        direction TB
        MainGo["main.go<br/>+ case: extension-publish"]
        Constants["constants.go<br/>+ ActExtensionPublish"]
        GiteaGo["gitea.go<br/>+ GetLatestRelease()<br/>+ GetReleaseByTag()<br/>+ SearchAllRepos()<br/>+ CreatePRWithBody()"]
        Interfaces["interfaces.go<br/>+ APIInterface расширение"]
    end

    subgraph New["Новый код"]
        direction TB
        ExtPublish["extension_publish.go"]
        Types["types.go<br/>Release, ReleaseAsset,<br/>Repository"]
        Tests["extension_publish_test.go"]
    end

    subgraph Flow["Поток выполнения ExtensionPublish()"]
        direction TB
        Step1["1. GetLatestRelease()"]
        Step2["2. FindSubscribedRepos()"]
        Step2a["   SearchAllRepos()"]
        Step2b["   GetBranches() ✓"]
        Step3["3. SyncExtensionContent()"]
        Step3a["   GetRepositoryContents() ✓"]
        Step3b["   SetRepositoryState() ✓"]
        Step4["4. CreatePRWithBody()"]
        Step5["5. GenerateReport()"]

        Step1 --> Step2
        Step2 --> Step2a
        Step2 --> Step2b
        Step2 --> Step3
        Step3 --> Step3a
        Step3 --> Step3b
        Step3 --> Step4
        Step4 --> Step5
    end

    MainGo -->|"вызывает"| ExtPublish
    Constants -.->|"константа"| MainGo
    ExtPublish -->|"использует"| GiteaGo
    ExtPublish -->|"использует"| Types
    GiteaGo -.->|"реализует"| Interfaces
    Tests -.->|"тестирует"| ExtPublish

    style Existing fill:#e3f2fd
    style New fill:#e8f5e9
    style Flow fill:#fff8e1
```

**Легенда:** ✓ — существующий метод

### 5.3 Файлы для создания/изменения

| Файл | Действие | Описание |
|------|----------|----------|
| `internal/constants/constants.go` | MODIFY | Добавить `ActExtensionPublish = "extension-publish"` |
| `internal/entity/gitea/types.go` | CREATE | Новые структуры Release, ReleaseAsset, Repository |
| `internal/entity/gitea/gitea.go` | MODIFY | Добавить методы для Release API и Search API |
| `internal/entity/gitea/interfaces.go` | MODIFY | Расширить интерфейс APIInterface |
| `internal/app/extension_publish.go` | CREATE | Основная логика команды |
| `internal/app/extension_publish_test.go` | CREATE | Unit-тесты |
| `cmd/apk-ci/main.go` | MODIFY | Добавить case в switch |

---

## 6. Sequence Diagram

```mermaid
sequenceDiagram
    autonumber
    participant GA as Gitea Actions
    participant Main as apk-ci<br/>(main)
    participant Svc as ExtPublish<br/>Service
    participant API as Gitea API<br/>(entity)
    participant Server as Gitea Server
    participant Target as Target Repos

    GA->>Main: on:release (published)
    activate Main
    Main->>Svc: ExtensionPublish(ctx, cfg)
    activate Svc

    Note over Svc,API: Шаг 1: Получение релиза
    Svc->>API: GetLatestRelease()
    API->>Server: GET /repos/{owner}/{repo}/releases/latest
    Server-->>API: Release JSON
    API-->>Svc: Release{v1.2.0}

    Note over Svc,API: Шаг 2: Поиск подписчиков
    Svc->>API: SearchAllRepos(opts)
    API->>Server: GET /repos/search?owner=...
    Server-->>API: []Repository
    API-->>Svc: []Repository

    loop Для каждого репозитория
        Svc->>API: GetBranches(repo)
        API->>Server: GET /repos/{owner}/{repo}/branches
        Server-->>API: []Branch
        API-->>Svc: []Branch
        Note right of Svc: Поиск веток с паттерном<br/>{Org}_{Repo}_{ExtDir}
    end

    Svc->>Svc: Найдены подписки

    Note over Svc,Target: Шаг 3-4: Публикация в каждый репозиторий
    loop Для каждой подписки (continue on error)
        Svc->>API: GetRepositoryContents(src)
        API->>Server: GET /repos/.../contents
        Server-->>API: []FileInfo
        API-->>Svc: []FileInfo

        Svc->>API: SetRepositoryState(ops, branch)
        API->>Server: POST /repos/.../contents (batch)
        Server->>Target: Commit files
        Target-->>Server: OK
        Server-->>API: Commit SHA
        API-->>Svc: OK

        Svc->>API: CreatePRWithBody(opts)
        API->>Server: POST /repos/.../pulls
        Server->>Target: Create PR
        Target-->>Server: PR #123
        Server-->>API: PRData
        API-->>Svc: PR #123, URL
    end

    Note over Svc: Шаг 5: Генерация отчёта
    Svc->>Svc: GenerateReport()

    Svc-->>Main: PublishReport{success:3, failed:1}
    deactivate Svc

    Main-->>GA: Exit code (0 или 1)
    deactivate Main
```

---

## 7. Обработка ошибок

### 7.0 State Diagram: Жизненный цикл публикации

```mermaid
stateDiagram-v2
    [*] --> Init: BR_COMMAND=extension-publish

    Init --> GetRelease: Запуск
    GetRelease --> ReleaseNotFound: Ошибка API
    GetRelease --> SearchRepos: Релиз получен

    ReleaseNotFound --> [*]: Exit 10

    SearchRepos --> NoSubscriptions: Подписки не найдены
    SearchRepos --> PublishLoop: Найдены подписки

    NoSubscriptions --> GenerateReport: Пустой отчёт
    NoSubscriptions --> [*]: Exit 0

    state PublishLoop {
        [*] --> SyncFiles
        SyncFiles --> CreatePR: OK
        SyncFiles --> RecordError: Ошибка
        CreatePR --> RecordSuccess: PR создан
        CreatePR --> RecordError: Ошибка
        RecordSuccess --> [*]
        RecordError --> [*]
    }

    PublishLoop --> GenerateReport: Все обработаны

    GenerateReport --> Success: Нет ошибок
    GenerateReport --> PartialSuccess: Есть ошибки

    Success --> [*]: Exit 0
    PartialSuccess --> [*]: Exit 1
```

### 7.1 Стратегия Continue-on-Error

```go
type PublishResult struct {
    Subscription Subscription
    Success      bool
    Error        error
}

func (s *ExtensionPublishService) Publish(ctx context.Context) (*PublishReport, error) {
    report := &PublishReport{}

    for _, sub := range subscriptions {
        result := s.publishToRepo(ctx, sub)

        if result.Error != nil {
            report.Failed = append(report.Failed, result)
            s.logger.Error("Ошибка публикации",
                "repo", sub.Repo,
                "error", result.Error)
            continue // Не прерываем цикл
        }

        report.Successful = append(report.Successful, result)
    }

    // Exit code = 1 если есть хотя бы одна ошибка
    if len(report.Failed) > 0 {
        return report, fmt.Errorf("публикация завершена с ошибками: %d из %d",
            len(report.Failed), report.TotalTargets)
    }

    return report, nil
}
```

### 7.2 Коды ошибок

| Код | Описание | Exit Code |
|-----|----------|-----------|
| `ERR_RELEASE_NOT_FOUND` | Релиз не найден | 10 |
| `ERR_NO_SUBSCRIPTIONS` | Нет подписанных репозиториев | 0 (success) |
| `ERR_PARTIAL_PUBLISH` | Часть публикаций не удалась | 1 |
| `ERR_GITEA_API` | Ошибка Gitea API | 11 |
| `ERR_RATE_LIMIT` | Превышен лимит запросов | 12 |

---

## 8. Формат вывода

### 8.1 Итоговый отчёт (JSON)

```json
{
    "status": "partial_success",
    "source": {
        "owner": "Extensions",
        "repo": "CommonExtension",
        "version": "v1.2.0",
        "release_url": "https://gitea.example.com/Extensions/CommonExtension/releases/tag/v1.2.0"
    },
    "summary": {
        "total": 5,
        "successful": 4,
        "failed": 1,
        "skipped": 0
    },
    "results": [
        {
            "repo": "MyOrg/ProjectA",
            "branch": "Extensions_CommonExtension_cfe/common",
            "target_dir": "cfe/common",
            "success": true,
            "pr_number": 123,
            "pr_url": "https://gitea.example.com/MyOrg/ProjectA/pulls/123"
        },
        {
            "repo": "MyOrg/ProjectB",
            "branch": "Extensions_CommonExtension_cfe/common",
            "target_dir": "cfe/common",
            "success": false,
            "error": "branch protection: requires review"
        }
    ]
}
```

### 8.2 Текстовый вывод

```
Extension Publish Report
========================
Source: Extensions/CommonExtension v1.2.0
Release: https://gitea.example.com/Extensions/CommonExtension/releases/tag/v1.2.0

Results: 4/5 successful

✓ MyOrg/ProjectA → PR #123
✓ MyOrg/ProjectC → PR #45
✓ MyOrg/ProjectD → PR #67
✓ OtherOrg/ProjectE → PR #12
✗ MyOrg/ProjectB → Error: branch protection requires review

Exit code: 1
```

---

## 9. Риски и митигация

| Риск | Вероятность | Импакт | Митигация |
|------|-------------|--------|-----------|
| API Rate Limiting | Средняя | Высокий | Добавить задержки между запросами, экспоненциальный backoff |
| Массовые изменения | Низкая | Высокий | Dry-run режим (через env var), детальное логирование |
| Конфликты в целевых репозиториях | Средняя | Средний | Создаём PR, не делаем автомерж |
| Неверный паттерн ветки | Низкая | Низкий | Валидация формата ветки |
| Большие расширения | Низкая | Средний | Пагинация в batch операциях |

---

## 10. Тестирование

### 10.1 Unit-тесты

```go
// internal/app/extension_publish_test.go

func TestParseSubscriptionBranch(t *testing.T) {
    tests := []struct {
        name     string
        branch   string
        expected *Subscription
        wantErr  bool
    }{
        {
            name:   "valid branch",
            branch: "Extensions_CommonExt_cfe/common",
            expected: &Subscription{
                Owner:     "Extensions",
                Repo:      "CommonExt",
                TargetDir: "cfe/common",
            },
        },
        {
            name:    "invalid format",
            branch:  "feature/some-feature",
            wantErr: true,
        },
    }
    // ...
}

func TestFindSubscribedRepos(t *testing.T) {
    // Mock Gitea API
    // Test pagination
    // Test branch matching
}
```

### 10.2 Integration-тесты

```go
// internal/app/extension_publish_integration_test.go
// +build integration

func TestExtensionPublishE2E(t *testing.T) {
    // Требует реального Gitea или моки
    // Создать тестовые репозитории
    // Создать ветки-подписки
    // Запустить публикацию
    // Проверить созданные PR
}
```

---

## 11. Связанные документы

- [Epic-0: Extension Publish](../epics/epic-0-extension-publish.md)
- [Архитектура apk-ci v2.0](../architecture.md)
- [Gitea API Documentation](https://docs.gitea.com/api/1.20/)

---

_Документ создан: 2025-12-08_
_Автор: Уинстон (Architect Agent)_
_Ревизия: 1.0_
