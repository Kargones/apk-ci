// Package sonarqube предоставляет адаптер для работы с SonarQube API.
package sonarqube

import (
	"context"
	"fmt"
	"log/slog"

	entity_sq "github.com/Kargones/apk-ci/internal/entity/sonarqube"
)

// Compile-time проверка реализации интерфейса.
var _ Client = (*APIClient)(nil)

// APIClient реализует интерфейс Client, делегируя вызовы entity/sonarqube.Entity.
type APIClient struct {
	entity *entity_sq.Entity
	logger *slog.Logger
}

// NewAPIClient создаёт новый APIClient, оборачивающий entity/sonarqube.Entity.
func NewAPIClient(entity *entity_sq.Entity) *APIClient {
	return &APIClient{
		entity: entity,
		logger: slog.Default(),
	}
}

// NewAPIClientWithLogger создаёт APIClient с пользовательским логгером.
func NewAPIClientWithLogger(entity *entity_sq.Entity, logger *slog.Logger) *APIClient {
	return &APIClient{
		entity: entity,
		logger: logger,
	}
}

// -------------------------------------------------------------------
// ProjectsAPI
// -------------------------------------------------------------------

func (c *APIClient) CreateProject(ctx context.Context, opts CreateProjectOptions) (*Project, error) {
	// Entity CreateProject принимает (owner, repo, branch), но adapter использует опции.
	// Извлекаем ключ и имя напрямую через entity, создавая проект по key/name.
	// Поскольку entity API имеет другую сигнатуру, делегируем через GetProject после создания.
	// Используем opts.Key и opts.Name напрямую — нужно вызвать entity напрямую.
	//
	// Entity.CreateProject генерирует ключ из owner/repo/branch, что не подходит.
	// Поэтому используем entity для HTTP-запроса напрямую через GetProject/etc.
	// Для MVP: вызываем entity с пустыми owner/repo/branch — ключ будет некорректным.
	// TODO: рефакторинг entity.CreateProject для принятия opts.

	// Workaround: entity.CreateProject строит ключ сам, но мы хотим контролировать ключ.
	// Вызываем entity с фиктивными параметрами — неудовлетворительно.
	// Пока возвращаем ошибку "not implemented" для CreateProject через adapter.
	return nil, fmt.Errorf("CreateProject через adapter: not yet fully implemented, используйте entity напрямую")
}

func (c *APIClient) GetProject(ctx context.Context, projectKey string) (*Project, error) {
	ep, err := c.entity.GetProject(ctx, projectKey)
	if err != nil {
		return nil, err
	}
	return convertEntityProject(ep), nil
}

func (c *APIClient) UpdateProject(ctx context.Context, projectKey string, opts UpdateProjectOptions) error {
	updates := &entity_sq.ProjectUpdate{
		Name:        opts.Name,
		Description: opts.Description,
		Visibility:  opts.Visibility,
	}
	return c.entity.UpdateProject(ctx, projectKey, updates)
}

func (c *APIClient) DeleteProject(ctx context.Context, projectKey string) error {
	return c.entity.DeleteProject(ctx, projectKey)
}

func (c *APIClient) ListProjects(ctx context.Context, opts ListProjectsOptions) ([]Project, error) {
	// Entity ListProjects принимает (owner, repo), adapter принимает opts с Query.
	// Используем Query как owner_repo pattern.
	entityProjects, err := c.entity.ListProjects(ctx, opts.Query, "")
	if err != nil {
		return nil, err
	}
	result := make([]Project, len(entityProjects))
	for i, ep := range entityProjects {
		result[i] = *convertEntityProject(&ep)
	}
	return result, nil
}

func (c *APIClient) SetProjectTags(ctx context.Context, projectKey string, tags []string) error {
	return c.entity.SetProjectTags(ctx, projectKey, tags)
}

// -------------------------------------------------------------------
// AnalysesAPI
// -------------------------------------------------------------------

func (c *APIClient) RunAnalysis(_ context.Context, _ RunAnalysisOptions) (*AnalysisResult, error) {
	// RunAnalysis requires sonar-scanner subprocess — not implemented in entity.Entity.
	return nil, fmt.Errorf("RunAnalysis: not yet implemented in adapter (requires sonar-scanner integration)")
}

func (c *APIClient) GetAnalyses(ctx context.Context, projectKey string) ([]Analysis, error) {
	entityAnalyses, err := c.entity.GetAnalyses(ctx, projectKey)
	if err != nil {
		return nil, err
	}
	result := make([]Analysis, len(entityAnalyses))
	for i, ea := range entityAnalyses {
		result[i] = Analysis{
			ID:         ea.ID,
			ProjectKey: ea.ProjectKey,
			Revision:   ea.Revision,
		}
		if ea.Date != nil {
			result[i].Date = ea.Date.Time
		}
	}
	return result, nil
}

func (c *APIClient) GetAnalysisStatus(ctx context.Context, taskID string) (*AnalysisStatus, error) {
	es, err := c.entity.GetAnalysisStatus(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return &AnalysisStatus{
		Status: es.Status,
	}, nil
}

// -------------------------------------------------------------------
// IssuesAPI
// -------------------------------------------------------------------

func (c *APIClient) GetIssues(ctx context.Context, opts GetIssuesOptions) ([]Issue, error) {
	params := &entity_sq.IssueParams{
		Severities: opts.Severities,
		Statuses:   opts.Statuses,
	}
	entityIssues, err := c.entity.GetIssues(ctx, opts.ProjectKey, params)
	if err != nil {
		return nil, err
	}
	result := make([]Issue, len(entityIssues))
	for i, ei := range entityIssues {
		result[i] = Issue{
			Key:       ei.Key,
			Rule:      ei.Rule,
			Severity:  ei.Severity,
			Component: ei.Component,
			Line:      ei.Line,
			Message:   ei.Message,
			Type:      ei.Type,
			CreatedAt: ei.CreatedAt,
			Status:    "",
		}
	}
	return result, nil
}

// -------------------------------------------------------------------
// QualityGatesAPI
// -------------------------------------------------------------------

func (c *APIClient) GetQualityGateStatus(ctx context.Context, projectKey string) (*QualityGateStatus, error) {
	es, err := c.entity.GetQualityGateStatus(ctx, projectKey)
	if err != nil {
		return nil, err
	}
	return &QualityGateStatus{
		Status: es.Status,
	}, nil
}

func (c *APIClient) GetQualityGates(ctx context.Context) ([]QualityGate, error) {
	entityGates, err := c.entity.GetQualityGates(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]QualityGate, len(entityGates))
	for i, eg := range entityGates {
		result[i] = QualityGate{
			ID:        eg.ID,
			Name:      eg.Name,
			IsDefault: eg.IsDefault,
		}
	}
	return result, nil
}

// -------------------------------------------------------------------
// MetricsAPI
// -------------------------------------------------------------------

func (c *APIClient) GetMetrics(ctx context.Context, projectKey string, metricKeys []string) (*Metrics, error) {
	em, err := c.entity.GetMetrics(ctx, projectKey, metricKeys)
	if err != nil {
		return nil, err
	}
	measures := make(map[string]string)
	for k, v := range em.Metrics {
		measures[k] = fmt.Sprintf("%g", v)
	}
	return &Metrics{
		ProjectKey: projectKey,
		Measures:   measures,
	}, nil
}

// -------------------------------------------------------------------
// Helpers
// -------------------------------------------------------------------

func convertEntityProject(ep *entity_sq.Project) *Project {
	p := &Project{
		Key:         ep.Key,
		Name:        ep.Name,
		Description: ep.Description,
		Qualifier:   ep.Qualifier,
		Visibility:  ep.Visibility,
		Tags:        ep.Tags,
	}
	if ep.LastAnalysisDate != nil {
		t := ep.LastAnalysisDate.Time
		p.LastAnalysisDate = &t
	}
	return p
}
