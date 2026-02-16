# Story 5.1: SonarQube Adapter Interface

Status: done

## Story

As a разработчик,
I want иметь абстракцию над SonarQube API,
so that я могу тестировать без реального сервера.

## Acceptance Criteria

1. [AC1] Interface SonarQubeClient определён в `internal/adapter/sonarqube/interfaces.go`
2. [AC2] Определены методы: CreateProject, RunAnalysis, GetIssues, GetQualityGate
3. [AC3] Можно подставить mock для тестов (интерфейсы соответствуют ISP)
4. [AC4] Интерфейсы разделены по принципу ISP (Interface Segregation Principle) на role-based интерфейсы

## Tasks / Subtasks

- [x] Task 1: Создать файл `internal/adapter/sonarqube/interfaces.go` (AC: #1)
  - [x] Subtask 1.1: Определить структуры данных (Project, Analysis, Issue, QualityGateStatus)
  - [x] Subtask 1.2: Определить ProjectsAPI interface (CreateProject, GetProject, UpdateProject, DeleteProject, ListProjects, SetProjectTags)
  - [x] Subtask 1.3: Определить AnalysesAPI interface (RunAnalysis, GetAnalyses, GetAnalysisStatus)
  - [x] Subtask 1.4: Определить IssuesAPI interface (GetIssues)
  - [x] Subtask 1.5: Определить QualityGatesAPI interface (GetQualityGateStatus, GetQualityGates)
  - [x] Subtask 1.6: Определить MetricsAPI interface (GetMetrics)
  - [x] Subtask 1.7: Определить композитный Client interface объединяющий все role-based интерфейсы
- [x] Task 2: Создать файл `internal/adapter/sonarqube/errors.go` (AC: #1)
  - [x] Subtask 2.1: Определить константы кодов ошибок (ErrSonarQubeConnect, ErrSonarQubeAPI, ErrSonarQubeAuth)
  - [x] Subtask 2.2: Определить типы ошибок SonarQubeError и ValidationError
- [x] Task 3: Написать тесты для интерфейсов (AC: #3)
  - [x] Subtask 3.1: Создать `internal/adapter/sonarqube/sonarqubetest/mock_client.go` с MockClient для тестирования
  - [x] Subtask 3.2: Написать пример использования mock в тестах

### Review Follow-ups (AI)

- [ ] [AI-Review][HIGH] Нет production-реализации Client — все хендлеры возвращают ошибку "SonarQube клиент не настроен" [interfaces.go:242-250]
- [ ] [AI-Review][MEDIUM] GetAnalyses без пагинации — при большом количестве анализов только первая страница [interfaces.go:246]
- [ ] [AI-Review][MEDIUM] RunAnalysisOptions.SourcePath никогда не заполняется — sonar-scanner не найдёт исходный код [interfaces.go:182-193]
- [ ] [AI-Review][MEDIUM] SonarQubeError не реализует interface{ HTTPStatusCode() int } — нет метода-геттера [errors.go:26-35]
- [ ] [AI-Review][LOW] Мок CreateProject без функции возвращает успешный проект — может маскировать баги [mock.go:55-64]
- [ ] [AI-Review][LOW] CreateProjectOptions.Visibility не валидируется — принимает произвольную строку [interfaces.go:151-158]

## Dev Notes

### Архитектурные паттерны и ограничения

**Interface Segregation Principle (ISP):**
- Разбить на сфокусированные интерфейсы по ролям: ProjectsAPI, AnalysesAPI, IssuesAPI, QualityGatesAPI, MetricsAPI
- Композитный интерфейс Client объединяет все role-based интерфейсы
- Следовать паттерну RAC adapter: [Source: internal/adapter/onec/rac/interfaces.go]
- Следовать паттерну MSSQL adapter: [Source: internal/adapter/mssql/interfaces.go]

**ADR-003: Role-based Interface Segregation** [Source: _bmad-output/project-planning-artifacts/architecture.md#ADR-003]
- Широкие интерфейсы нарушают ISP — разделить на role-based интерфейсы
- Меньше зависимостей, проще мокирование

### Структура интерфейсов (ISP-compliant)

```go
// ProjectsAPI — управление проектами в SonarQube
type ProjectsAPI interface {
    CreateProject(ctx context.Context, opts CreateProjectOptions) (*Project, error)
    GetProject(ctx context.Context, projectKey string) (*Project, error)
    UpdateProject(ctx context.Context, projectKey string, opts UpdateProjectOptions) error
    DeleteProject(ctx context.Context, projectKey string) error
    ListProjects(ctx context.Context, opts ListProjectsOptions) ([]Project, error)
    SetProjectTags(ctx context.Context, projectKey string, tags []string) error
}

// AnalysesAPI — запуск и получение информации об анализах
type AnalysesAPI interface {
    RunAnalysis(ctx context.Context, opts RunAnalysisOptions) (*AnalysisResult, error)
    GetAnalyses(ctx context.Context, projectKey string) ([]Analysis, error)
    GetAnalysisStatus(ctx context.Context, analysisID string) (*AnalysisStatus, error)
}

// IssuesAPI — получение информации о проблемах качества кода
type IssuesAPI interface {
    GetIssues(ctx context.Context, opts GetIssuesOptions) ([]Issue, error)
}

// QualityGatesAPI — работа с quality gates
type QualityGatesAPI interface {
    GetQualityGateStatus(ctx context.Context, projectKey string) (*QualityGateStatus, error)
    GetQualityGates(ctx context.Context) ([]QualityGate, error)
}

// MetricsAPI — получение метрик проекта
type MetricsAPI interface {
    GetMetrics(ctx context.Context, projectKey string, metricKeys []string) (*Metrics, error)
}

// Client — композитный интерфейс, объединяющий все операции SonarQube
type Client interface {
    ProjectsAPI
    AnalysesAPI
    IssuesAPI
    QualityGatesAPI
    MetricsAPI
}
```

### Структуры данных

Переиспользовать и адаптировать из legacy `internal/entity/sonarqube/interfaces.go`:
- Project, Analysis, Issue, QualityGateStatus, Metrics
- Добавить Options structs для каждого метода с параметрами

### Коды ошибок

```go
const (
    ErrSonarQubeConnect = "SONARQUBE.CONNECT_FAILED"
    ErrSonarQubeAPI     = "SONARQUBE.API_FAILED"
    ErrSonarQubeAuth    = "SONARQUBE.AUTH_FAILED"
    ErrSonarQubeTimeout = "SONARQUBE.TIMEOUT"
)
```

### Тестирование

**Mock Pattern** (по образцу ractest/MockRACClient):
- Создать `sonarqubetest/mock_client.go`
- Использовать функциональные поля для каждого метода
- Позволяет настраивать поведение mock в каждом тесте

```go
type MockClient struct {
    CreateProjectFunc     func(ctx context.Context, opts CreateProjectOptions) (*Project, error)
    GetIssuesFunc         func(ctx context.Context, opts GetIssuesOptions) ([]Issue, error)
    GetQualityGateStatusFunc func(ctx context.Context, projectKey string) (*QualityGateStatus, error)
    // ... остальные методы
}
```

### Project Structure Notes

**Новые файлы:**
- `internal/adapter/sonarqube/interfaces.go` — ISP-compliant интерфейсы
- `internal/adapter/sonarqube/errors.go` — типы и коды ошибок
- `internal/adapter/sonarqube/sonarqubetest/mock_client.go` — mock для тестов

**НЕ изменять legacy код:**
- `internal/entity/sonarqube/` — оставить как есть до полной миграции

### References

- [Source: internal/adapter/onec/rac/interfaces.go] — образец ISP-compliant интерфейсов
- [Source: internal/adapter/mssql/interfaces.go] — образец ISP-compliant интерфейсов
- [Source: internal/adapter/onec/rac/ractest/mock_client.go] — образец mock pattern
- [Source: internal/entity/sonarqube/interfaces.go] — legacy интерфейсы для переиспользования типов
- [Source: _bmad-output/project-planning-artifacts/architecture.md#ADR-003] — ADR для ISP
- [Source: _bmad-output/project-planning-artifacts/epics/epic-5-quality-integration.md#Story-5.1] — исходные требования

### Review Follow-ups (AI Code Review #34)

- [ ] [AI-Review][HIGH] Нет production-реализации Client — хендлеры возвращают "не настроен" (TODO H-6) [interfaces.go:242-250]
- [ ] [AI-Review][HIGH] Mock CreateProject без функции возвращает успешный проект — маскирует баги [sonarqubetest/mock.go:55-64]
- [ ] [AI-Review][MEDIUM] GetAnalyses без пагинации — только первая страница (>100) [interfaces.go:246]
- [ ] [AI-Review][MEDIUM] SonarQubeError без геттера HTTPStatusCode() [errors.go:26-35]
- [ ] [AI-Review][MEDIUM] RunAnalysisOptions.SourcePath не заполняется (TODO H-7) [interfaces.go:182-193]

## Dev Agent Record

### Agent Model Used

Claude Opus 4.5 (claude-opus-4-5-20251101)

### Debug Log References

### Completion Notes List

- Реализован полный набор ISP-compliant интерфейсов для SonarQube API
- Интерфейсы разделены на 5 role-based: ProjectsAPI, AnalysesAPI, IssuesAPI, QualityGatesAPI, MetricsAPI
- Композитный Client объединяет все интерфейсы
- Структуры данных включают: Project, Analysis, AnalysisStatus, AnalysisResult, Issue, QualityGateStatus, QualityGate, Metrics и Options structs
- Типизированные ошибки с кодами: ErrSonarQubeConnect, ErrSonarQubeAPI, ErrSonarQubeAuth, ErrSonarQubeTimeout, ErrSonarQubeNotFound, ErrSonarQubeValidation
- SonarQubeError поддерживает errors.Is/As через Unwrap()
- MockClient с функциональными полями для всех методов по паттерну ractest/MockRACClient
- Compile-time проверки реализации интерфейсов в mock
- Вспомогательные конструкторы: NewMockClient, NewMockClientWithProject, NewMockClientWithQualityGateStatus, NewMockClientWithIssues
- Тестовые данные: ProjectData(), IssueData(), AnalysisData()
- Все unit-тесты проходят (25 тестов)

### File List

- internal/adapter/sonarqube/interfaces.go (новый)
- internal/adapter/sonarqube/interfaces_test.go (новый, code-review fix)
- internal/adapter/sonarqube/errors.go (новый)
- internal/adapter/sonarqube/errors_test.go (новый)
- internal/adapter/sonarqube/sonarqubetest/mock.go (новый)
- internal/adapter/sonarqube/sonarqubetest/mock_test.go (новый)
- internal/adapter/sonarqube/sonarqubetest/doc.go (новый, code-review fix)

## Change Log

- 2026-02-04: Story 5-1 реализована — SonarQube Adapter Interface с ISP-compliant интерфейсами и mock для тестов
- 2026-02-04: Code Review #1 — исправлено 10 issues (5 HIGH, 5 MEDIUM):
  - H-1: IsXxxError функции теперь используют errors.As для поддержки wrapped errors
  - H-2: Добавлена документация к RunAnalysis о subprocess execution
  - H-3: Добавлен interfaces_test.go с проверкой ISP композиции
  - H-4: Добавлены примеры ISP использования в mock_test.go
  - H-5: Добавлен doc.go для sonarqubetest пакета
  - M-1: Добавлен IsConnectionError хелпер
  - M-2: ValidationError теперь поддерживает Unwrap()
  - M-3: Уточнены комментарии для Page (1-based pagination)
  - M-4: Добавлены тесты для ValidationError как Cause
  - M-5: Example тесты теперь имеют Output comments
