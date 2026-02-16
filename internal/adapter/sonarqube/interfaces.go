// Package sonarqube определяет интерфейсы и типы данных для работы с SonarQube API.
// Пакет предоставляет абстракцию над SonarQube операциями, разделённую по принципу ISP
// (Interface Segregation Principle) на сфокусированные интерфейсы:
// ProjectsAPI, AnalysesAPI, IssuesAPI, QualityGatesAPI, MetricsAPI.
// Композитный интерфейс Client объединяет все вышеперечисленные.
package sonarqube

import (
	"context"
	"time"
)

// -------------------------------------------------------------------
// Структуры данных
// -------------------------------------------------------------------

// Project представляет проект в SonarQube.
type Project struct {
	// Key — уникальный идентификатор проекта
	Key string
	// Name — отображаемое имя проекта
	Name string
	// Description — описание проекта
	Description string
	// Qualifier — тип компонента (TRK для проекта)
	Qualifier string
	// Visibility — видимость проекта (public/private)
	Visibility string
	// LastAnalysisDate — дата последнего анализа
	LastAnalysisDate *time.Time
	// Tags — теги проекта
	Tags []string
}

// Analysis представляет результат анализа кода в SonarQube.
type Analysis struct {
	// ID — уникальный идентификатор анализа
	ID string
	// ProjectKey — ключ проекта
	ProjectKey string
	// Date — дата и время анализа
	Date time.Time
	// Revision — ревизия исходного кода (commit hash)
	Revision string
	// Version — версия проекта при анализе
	Version string
}

// AnalysisStatus представляет статус выполнения анализа.
type AnalysisStatus struct {
	// TaskID — идентификатор задачи анализа
	TaskID string
	// Status — текущий статус (PENDING, IN_PROGRESS, SUCCESS, FAILED, CANCELED)
	Status string
	// AnalysisID — идентификатор анализа (доступен после завершения)
	AnalysisID string
	// ErrorMessage — сообщение об ошибке (если статус FAILED)
	ErrorMessage string
}

// AnalysisResult представляет результат запуска анализа.
type AnalysisResult struct {
	// TaskID — идентификатор задачи для отслеживания прогресса
	TaskID string
	// ProjectKey — ключ проекта
	ProjectKey string
	// AnalysisID — идентификатор анализа (может быть пустым до завершения)
	AnalysisID string
}

// Issue представляет проблему качества кода в SonarQube.
type Issue struct {
	// Key — уникальный идентификатор проблемы
	Key string
	// Rule — ключ правила (например, "go:S1234")
	Rule string
	// Severity — серьёзность (BLOCKER, CRITICAL, MAJOR, MINOR, INFO)
	Severity string
	// Component — компонент (файл) с проблемой
	Component string
	// Line — номер строки
	Line int
	// Message — описание проблемы
	Message string
	// Type — тип проблемы (BUG, VULNERABILITY, CODE_SMELL)
	Type string
	// CreatedAt — дата создания
	CreatedAt time.Time
	// Status — статус проблемы (OPEN, CONFIRMED, RESOLVED, etc.)
	Status string
}

// QualityGateStatus представляет статус Quality Gate для проекта.
type QualityGateStatus struct {
	// Status — общий статус (OK, WARN, ERROR)
	Status string
	// Conditions — список условий с их статусами
	Conditions []QualityCondition
}

// QualityCondition представляет условие Quality Gate.
type QualityCondition struct {
	// Metric — имя метрики
	Metric string
	// Operator — оператор сравнения (GT, LT, EQ)
	Operator string
	// ErrorThreshold — порог ошибки
	ErrorThreshold string
	// ActualValue — фактическое значение
	ActualValue string
	// Status — статус условия (OK, WARN, ERROR)
	Status string
}

// QualityGate представляет Quality Gate в SonarQube.
type QualityGate struct {
	// ID — идентификатор Quality Gate
	ID int
	// Name — имя Quality Gate
	Name string
	// IsDefault — является ли Quality Gate по умолчанию
	IsDefault bool
	// Conditions — условия Quality Gate
	Conditions []QualityGateCondition
}

// QualityGateCondition представляет условие для определения Quality Gate.
type QualityGateCondition struct {
	// ID — идентификатор условия
	ID int
	// Metric — имя метрики
	Metric string
	// Operator — оператор сравнения
	Operator string
	// ErrorThreshold — порог ошибки
	ErrorThreshold string
}

// Metrics представляет метрики проекта.
type Metrics struct {
	// ProjectKey — ключ проекта
	ProjectKey string
	// Measures — значения метрик (имя -> значение)
	Measures map[string]string
}

// -------------------------------------------------------------------
// Options структуры для методов API
// -------------------------------------------------------------------

// CreateProjectOptions содержит параметры для создания проекта.
type CreateProjectOptions struct {
	// Key — уникальный ключ проекта (обязательный)
	Key string
	// Name — отображаемое имя проекта (обязательный)
	Name string
	// Visibility — видимость проекта (public/private)
	Visibility string
}

// UpdateProjectOptions содержит параметры для обновления проекта.
type UpdateProjectOptions struct {
	// Name — новое имя проекта
	Name string
	// Description — новое описание
	Description string
	// Visibility — новая видимость
	Visibility string
}

// ListProjectsOptions содержит параметры для получения списка проектов.
type ListProjectsOptions struct {
	// Query — поисковый запрос по имени или ключу
	Query string
	// Page — номер страницы (начиная с 1, SonarQube API использует 1-based pagination)
	Page int
	// PageSize — размер страницы (по умолчанию 100 в SonarQube)
	PageSize int
}

// RunAnalysisOptions содержит параметры для запуска анализа.
type RunAnalysisOptions struct {
	// ProjectKey — ключ проекта для анализа
	ProjectKey string
	// SourcePath — путь к исходному коду
	SourcePath string
	// Branch — имя ветки (опционально)
	Branch string
	// PullRequest — номер pull request (опционально, для PR анализа)
	PullRequest string
	// Properties — дополнительные свойства анализа
	Properties map[string]string
}

// GetIssuesOptions содержит параметры для получения списка проблем.
type GetIssuesOptions struct {
	// ProjectKey — ключ проекта (обязательный)
	ProjectKey string
	// Branch — имя ветки (опционально)
	Branch string
	// Severities — фильтр по серьёзности (BLOCKER, CRITICAL, etc.)
	Severities []string
	// Types — фильтр по типу (BUG, VULNERABILITY, CODE_SMELL)
	Types []string
	// Statuses — фильтр по статусу (OPEN, CONFIRMED, etc.)
	Statuses []string
	// Page — номер страницы (начиная с 1, SonarQube API использует 1-based pagination)
	Page int
	// PageSize — размер страницы (по умолчанию 100 в SonarQube, максимум 500)
	PageSize int
}

// -------------------------------------------------------------------
// ISP-compliant интерфейсы
// -------------------------------------------------------------------

// ProjectsAPI предоставляет операции для управления проектами в SonarQube.
type ProjectsAPI interface {
	// CreateProject создаёт новый проект в SonarQube.
	CreateProject(ctx context.Context, opts CreateProjectOptions) (*Project, error)
	// GetProject возвращает информацию о проекте по ключу.
	GetProject(ctx context.Context, projectKey string) (*Project, error)
	// UpdateProject обновляет информацию о проекте.
	UpdateProject(ctx context.Context, projectKey string, opts UpdateProjectOptions) error
	// DeleteProject удаляет проект из SonarQube.
	DeleteProject(ctx context.Context, projectKey string) error
	// ListProjects возвращает список проектов с фильтрацией.
	ListProjects(ctx context.Context, opts ListProjectsOptions) ([]Project, error)
	// SetProjectTags устанавливает теги для проекта.
	SetProjectTags(ctx context.Context, projectKey string, tags []string) error
}

// AnalysesAPI предоставляет операции для управления анализами.
//
// Примечание: SonarQube анализ выполняется внешним процессом (sonar-scanner CLI),
// а не напрямую через SonarQube Web API. Метод RunAnalysis в реализации должен:
// 1. Запустить subprocess sonar-scanner с переданными параметрами
// 2. Дождаться завершения сканирования
// 3. Вернуть TaskID для отслеживания статуса через GetAnalysisStatus
//
// Это отличается от чистого HTTP API клиента — см. legacy internal/entity/sonarqube/scanner.go.
type AnalysesAPI interface {
	// RunAnalysis запускает анализ проекта через sonar-scanner subprocess.
	// Возвращает TaskID для отслеживания прогресса через GetAnalysisStatus.
	RunAnalysis(ctx context.Context, opts RunAnalysisOptions) (*AnalysisResult, error)
	// GetAnalyses возвращает список анализов для проекта.
	GetAnalyses(ctx context.Context, projectKey string) ([]Analysis, error)
	// GetAnalysisStatus возвращает статус выполнения анализа по ID задачи.
	GetAnalysisStatus(ctx context.Context, taskID string) (*AnalysisStatus, error)
}

// IssuesAPI предоставляет операции для получения информации о проблемах качества кода.
type IssuesAPI interface {
	// GetIssues возвращает список проблем с фильтрацией.
	GetIssues(ctx context.Context, opts GetIssuesOptions) ([]Issue, error)
}

// QualityGatesAPI предоставляет операции для работы с Quality Gates.
type QualityGatesAPI interface {
	// GetQualityGateStatus возвращает статус Quality Gate для проекта.
	GetQualityGateStatus(ctx context.Context, projectKey string) (*QualityGateStatus, error)
	// GetQualityGates возвращает список доступных Quality Gates.
	GetQualityGates(ctx context.Context) ([]QualityGate, error)
}

// MetricsAPI предоставляет операции для получения метрик проекта.
type MetricsAPI interface {
	// GetMetrics возвращает метрики проекта по указанным ключам метрик.
	GetMetrics(ctx context.Context, projectKey string, metricKeys []string) (*Metrics, error)
}

// Client — композитный интерфейс, объединяющий все операции SonarQube.
type Client interface {
	ProjectsAPI
	AnalysesAPI
	IssuesAPI
	QualityGatesAPI
	MetricsAPI
}
