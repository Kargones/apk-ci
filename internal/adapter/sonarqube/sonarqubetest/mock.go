// Package sonarqubetest предоставляет тестовые утилиты для пакета sonarqube:
// мок-реализации интерфейсов и вспомогательные конструкторы.
package sonarqubetest

import (
	"context"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/sonarqube"
)

// Compile-time проверки реализации интерфейсов
var (
	_ sonarqube.Client        = (*MockClient)(nil)
	_ sonarqube.ProjectsAPI   = (*MockClient)(nil)
	_ sonarqube.AnalysesAPI   = (*MockClient)(nil)
	_ sonarqube.IssuesAPI     = (*MockClient)(nil)
	_ sonarqube.QualityGatesAPI = (*MockClient)(nil)
	_ sonarqube.MetricsAPI    = (*MockClient)(nil)
)

// MockClient — мок-реализация sonarqube.Client для тестирования.
// Использует функциональные поля для гибкой настройки поведения в тестах.
type MockClient struct {
	// ProjectsAPI
	CreateProjectFunc   func(ctx context.Context, opts sonarqube.CreateProjectOptions) (*sonarqube.Project, error)
	GetProjectFunc      func(ctx context.Context, projectKey string) (*sonarqube.Project, error)
	UpdateProjectFunc   func(ctx context.Context, projectKey string, opts sonarqube.UpdateProjectOptions) error
	DeleteProjectFunc   func(ctx context.Context, projectKey string) error
	ListProjectsFunc    func(ctx context.Context, opts sonarqube.ListProjectsOptions) ([]sonarqube.Project, error)
	SetProjectTagsFunc  func(ctx context.Context, projectKey string, tags []string) error

	// AnalysesAPI
	RunAnalysisFunc       func(ctx context.Context, opts sonarqube.RunAnalysisOptions) (*sonarqube.AnalysisResult, error)
	GetAnalysesFunc       func(ctx context.Context, projectKey string) ([]sonarqube.Analysis, error)
	GetAnalysisStatusFunc func(ctx context.Context, taskID string) (*sonarqube.AnalysisStatus, error)

	// IssuesAPI
	GetIssuesFunc func(ctx context.Context, opts sonarqube.GetIssuesOptions) ([]sonarqube.Issue, error)

	// QualityGatesAPI
	GetQualityGateStatusFunc func(ctx context.Context, projectKey string) (*sonarqube.QualityGateStatus, error)
	GetQualityGatesFunc      func(ctx context.Context) ([]sonarqube.QualityGate, error)

	// MetricsAPI
	GetMetricsFunc func(ctx context.Context, projectKey string, metricKeys []string) (*sonarqube.Metrics, error)
}

// -------------------------------------------------------------------
// ProjectsAPI implementation
// -------------------------------------------------------------------

// CreateProject создаёт новый проект.
// При отсутствии пользовательской функции возвращает тестовые данные.
func (m *MockClient) CreateProject(ctx context.Context, opts sonarqube.CreateProjectOptions) (*sonarqube.Project, error) {
	if m.CreateProjectFunc != nil {
		return m.CreateProjectFunc(ctx, opts)
	}
	return &sonarqube.Project{
		Key:        opts.Key,
		Name:       opts.Name,
		Visibility: opts.Visibility,
	}, nil
}

// GetProject возвращает информацию о проекте.
// При отсутствии пользовательской функции возвращает тестовые данные.
func (m *MockClient) GetProject(ctx context.Context, projectKey string) (*sonarqube.Project, error) {
	if m.GetProjectFunc != nil {
		return m.GetProjectFunc(ctx, projectKey)
	}
	return &sonarqube.Project{
		Key:         projectKey,
		Name:        "Test Project",
		Description: "Тестовый проект",
		Visibility:  "private",
	}, nil
}

// UpdateProject обновляет проект.
// При отсутствии пользовательской функции возвращает nil.
func (m *MockClient) UpdateProject(ctx context.Context, projectKey string, opts sonarqube.UpdateProjectOptions) error {
	if m.UpdateProjectFunc != nil {
		return m.UpdateProjectFunc(ctx, projectKey, opts)
	}
	return nil
}

// DeleteProject удаляет проект.
// При отсутствии пользовательской функции возвращает nil.
func (m *MockClient) DeleteProject(ctx context.Context, projectKey string) error {
	if m.DeleteProjectFunc != nil {
		return m.DeleteProjectFunc(ctx, projectKey)
	}
	return nil
}

// ListProjects возвращает список проектов.
// При отсутствии пользовательской функции возвращает пустой срез.
func (m *MockClient) ListProjects(ctx context.Context, opts sonarqube.ListProjectsOptions) ([]sonarqube.Project, error) {
	if m.ListProjectsFunc != nil {
		return m.ListProjectsFunc(ctx, opts)
	}
	return []sonarqube.Project{}, nil
}

// SetProjectTags устанавливает теги проекта.
// При отсутствии пользовательской функции возвращает nil.
func (m *MockClient) SetProjectTags(ctx context.Context, projectKey string, tags []string) error {
	if m.SetProjectTagsFunc != nil {
		return m.SetProjectTagsFunc(ctx, projectKey, tags)
	}
	return nil
}

// -------------------------------------------------------------------
// AnalysesAPI implementation
// -------------------------------------------------------------------

// RunAnalysis запускает анализ проекта.
// При отсутствии пользовательской функции возвращает тестовые данные.
func (m *MockClient) RunAnalysis(ctx context.Context, opts sonarqube.RunAnalysisOptions) (*sonarqube.AnalysisResult, error) {
	if m.RunAnalysisFunc != nil {
		return m.RunAnalysisFunc(ctx, opts)
	}
	return &sonarqube.AnalysisResult{
		TaskID:     "test-task-id",
		ProjectKey: opts.ProjectKey,
	}, nil
}

// GetAnalyses возвращает список анализов.
// При отсутствии пользовательской функции возвращает пустой срез.
func (m *MockClient) GetAnalyses(ctx context.Context, projectKey string) ([]sonarqube.Analysis, error) {
	if m.GetAnalysesFunc != nil {
		return m.GetAnalysesFunc(ctx, projectKey)
	}
	return []sonarqube.Analysis{}, nil
}

// GetAnalysisStatus возвращает статус анализа.
// При отсутствии пользовательской функции возвращает статус SUCCESS.
func (m *MockClient) GetAnalysisStatus(ctx context.Context, taskID string) (*sonarqube.AnalysisStatus, error) {
	if m.GetAnalysisStatusFunc != nil {
		return m.GetAnalysisStatusFunc(ctx, taskID)
	}
	return &sonarqube.AnalysisStatus{
		TaskID:     taskID,
		Status:     "SUCCESS",
		AnalysisID: "test-analysis-id",
	}, nil
}

// -------------------------------------------------------------------
// IssuesAPI implementation
// -------------------------------------------------------------------

// GetIssues возвращает список проблем.
// При отсутствии пользовательской функции возвращает пустой срез.
func (m *MockClient) GetIssues(ctx context.Context, opts sonarqube.GetIssuesOptions) ([]sonarqube.Issue, error) {
	if m.GetIssuesFunc != nil {
		return m.GetIssuesFunc(ctx, opts)
	}
	return []sonarqube.Issue{}, nil
}

// -------------------------------------------------------------------
// QualityGatesAPI implementation
// -------------------------------------------------------------------

// GetQualityGateStatus возвращает статус Quality Gate.
// При отсутствии пользовательской функции возвращает статус OK.
func (m *MockClient) GetQualityGateStatus(ctx context.Context, projectKey string) (*sonarqube.QualityGateStatus, error) {
	if m.GetQualityGateStatusFunc != nil {
		return m.GetQualityGateStatusFunc(ctx, projectKey)
	}
	return &sonarqube.QualityGateStatus{
		Status:     "OK",
		Conditions: []sonarqube.QualityCondition{},
	}, nil
}

// GetQualityGates возвращает список Quality Gates.
// При отсутствии пользовательской функции возвращает пустой срез.
func (m *MockClient) GetQualityGates(ctx context.Context) ([]sonarqube.QualityGate, error) {
	if m.GetQualityGatesFunc != nil {
		return m.GetQualityGatesFunc(ctx)
	}
	return []sonarqube.QualityGate{}, nil
}

// -------------------------------------------------------------------
// MetricsAPI implementation
// -------------------------------------------------------------------

// GetMetrics возвращает метрики проекта.
// При отсутствии пользовательской функции возвращает пустые метрики.
func (m *MockClient) GetMetrics(ctx context.Context, projectKey string, metricKeys []string) (*sonarqube.Metrics, error) {
	if m.GetMetricsFunc != nil {
		return m.GetMetricsFunc(ctx, projectKey, metricKeys)
	}
	return &sonarqube.Metrics{
		ProjectKey: projectKey,
		Measures:   map[string]string{},
	}, nil
}

// -------------------------------------------------------------------
// Конструкторы для создания MockClient
// -------------------------------------------------------------------

// NewMockClient создаёт MockClient с дефолтными тестовыми данными.
func NewMockClient() *MockClient {
	return &MockClient{}
}

// NewMockClientWithProject создаёт MockClient с предзаданным проектом.
func NewMockClientWithProject(project *sonarqube.Project) *MockClient {
	return &MockClient{
		GetProjectFunc: func(_ context.Context, _ string) (*sonarqube.Project, error) {
			return project, nil
		},
	}
}

// NewMockClientWithQualityGateStatus создаёт MockClient с предзаданным статусом Quality Gate.
func NewMockClientWithQualityGateStatus(status string, conditions []sonarqube.QualityCondition) *MockClient {
	return &MockClient{
		GetQualityGateStatusFunc: func(_ context.Context, _ string) (*sonarqube.QualityGateStatus, error) {
			return &sonarqube.QualityGateStatus{
				Status:     status,
				Conditions: conditions,
			}, nil
		},
	}
}

// NewMockClientWithIssues создаёт MockClient с предзаданными проблемами.
func NewMockClientWithIssues(issues []sonarqube.Issue) *MockClient {
	return &MockClient{
		GetIssuesFunc: func(_ context.Context, _ sonarqube.GetIssuesOptions) ([]sonarqube.Issue, error) {
			return issues, nil
		},
	}
}

// -------------------------------------------------------------------
// Тестовые данные
// -------------------------------------------------------------------

// ProjectData возвращает тестовые данные проекта для использования в тестах.
func ProjectData() *sonarqube.Project {
	now := time.Now()
	return &sonarqube.Project{
		Key:              "test-project",
		Name:             "Test Project",
		Description:      "Тестовый проект для unit-тестов",
		Qualifier:        "TRK",
		Visibility:       "private",
		LastAnalysisDate: &now,
		Tags:             []string{"test", "demo"},
	}
}

// IssueData возвращает тестовые данные проблем для использования в тестах.
func IssueData() []sonarqube.Issue {
	return []sonarqube.Issue{
		{
			Key:       "issue-1",
			Rule:      "go:S1234",
			Severity:  "MAJOR",
			Component: "src/main.go",
			Line:      42,
			Message:   "Тестовая проблема 1",
			Type:      "CODE_SMELL",
			CreatedAt: time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
			Status:    "OPEN",
		},
		{
			Key:       "issue-2",
			Rule:      "go:S5678",
			Severity:  "CRITICAL",
			Component: "src/handler.go",
			Line:      100,
			Message:   "Тестовая проблема 2",
			Type:      "BUG",
			CreatedAt: time.Date(2026, 2, 2, 14, 30, 0, 0, time.UTC),
			Status:    "OPEN",
		},
	}
}

// AnalysisData возвращает тестовые данные анализа для использования в тестах.
func AnalysisData() []sonarqube.Analysis {
	return []sonarqube.Analysis{
		{
			ID:         "analysis-1",
			ProjectKey: "test-project",
			Date:       time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
			Revision:   "abc123",
			Version:    "1.0.0",
		},
		{
			ID:         "analysis-2",
			ProjectKey: "test-project",
			Date:       time.Date(2026, 2, 2, 15, 0, 0, 0, time.UTC),
			Revision:   "def456",
			Version:    "1.0.1",
		},
	}
}
