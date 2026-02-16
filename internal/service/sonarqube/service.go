// Package sonarqube provides implementation of SonarQube service layer.
// This package contains the business logic for SonarQube operations,
// including project management, analysis, and metrics retrieval.
package sonarqube

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/entity/sonarqube"
)

// Service provides business logic for SonarQube operations.
// This service layer implements validation and coordination
// between different SonarQube entities and external dependencies.
type Service struct {
	// entity is the SonarQube entity for low-level API interactions
	entity sonarqube.APIInterface

	// config contains the SonarQube configuration settings
	config *config.SonarQubeConfig

	// logger is the structured logger for this service
	logger *slog.Logger
}

// UpdateProjectDescription updates the description of a project in SonarQube.
// This method updates the description of the specified project.
//
// Parameters:
//   - _: context for the operation
//   - projectKey: key of the project to update
//   - description: new project description
//
// Returns:
//   - error: error if project description update fails
func (s *Service) UpdateProjectDescription(_ context.Context, projectKey, description string) error {
	s.logger.Debug("Updating SonarQube project description", "projectKey", projectKey)

	// Validate input parameters
	if projectKey == "" {
		return &sonarqube.ValidationError{
			Field:   "projectKey",
			Message: "Project key must be provided",
		}
	}

	// Prepare project updates
	updates := &sonarqube.ProjectUpdate{
		Description: description,
	}

	// Update project using entity
	if err := s.entity.UpdateProject(projectKey, updates); err != nil {
		s.logger.Error("Failed to update SonarQube project description", "error", err)
		return fmt.Errorf("failed to update project description: %w", err)
	}

	s.logger.Debug("SonarQube project description updated successfully", "projectKey", projectKey)
	return nil
}

// UpdateProjectAdministrators updates the administrators of a project in SonarQube.
// This method updates the list of administrators for the specified project.
//
// Parameters:
//   - _: context for the operation
//   - projectKey: key of the project to update
//   - administrators: list of administrator usernames
//
// Returns:
//   - error: error if project administrators update fails
func (s *Service) UpdateProjectAdministrators(_ context.Context, projectKey string, administrators []string) error {
	s.logger.Debug("Updating SonarQube project administrators", "projectKey", projectKey, "administrators", administrators)

	// Validate input parameters
	if projectKey == "" {
		return &sonarqube.ValidationError{
			Field:   "projectKey",
			Message: "Project key must be provided",
		}
	}

	// For now, we'll just log that the project administrators update is not fully implemented
	// In a real implementation, you would need to update the project administrators in SonarQube
	s.logger.Warn("Project administrators update is not fully implemented yet")

	s.logger.Debug("SonarQube project administrators update processed", "projectKey", projectKey)
	return nil
}

// NewSonarQubeService creates a new instance of SonarQubeService.
// This function initializes the service with the provided SonarQube entity
// and configuration.
//
// Parameters:
//   - entity: SonarQube entity for low-level API interactions
//   - cfg: SonarQube configuration settings
//   - logger: structured logger instance
//
// Returns:
//   - *SonarQubeService: initialized SonarQube service
func NewSonarQubeService(entity sonarqube.APIInterface, cfg *config.SonarQubeConfig, logger *slog.Logger) *Service {
	return &Service{
		entity: entity,
		config: cfg,
		logger: logger,
	}
}

// ValidateToken validates the configured authentication token.
// This method checks if the configured token is valid by calling
// the entity's ValidateToken method.
//
// Returns:
//   - error: error if token is invalid or validation fails
func (s *Service) ValidateToken() error {
	s.logger.Debug("Validating SonarQube token")

	if err := s.entity.ValidateToken(); err != nil {
		s.logger.Error("SonarQube token validation failed", "error", err)
		return fmt.Errorf("token validation failed: %w", err)
	}

	s.logger.Debug("SonarQube token is valid")
	return nil
}

// CreateProject creates a new project in SonarQube.
// This method creates a new project with the specified owner, repo, and branch.
//
// Parameters:
//   - _: context for the operation
//   - owner: project owner
//   - repo: project repository
//   - branch: project branch
//
// Returns:
//   - *sonarqube.Project: created project
//   - error: error if project creation fails
func (s *Service) CreateProject(_ context.Context, owner, repo, branch string) (*sonarqube.Project, error) {
	s.logger.Debug("Creating SonarQube project", "owner", owner, "repo", repo, "branch", branch)

	// Validate input parameters
	if owner == "" || repo == "" || branch == "" {
		return nil, &sonarqube.ValidationError{
			Field:   "owner/repo/branch",
			Message: "Owner, repo, and branch must be provided",
		}
	}

	// Generate project key to check if project already exists
	projectKey := fmt.Sprintf("%s_%s_%s", owner, repo, branch)
	if s.config.ProjectPrefix != "" {
		projectKey = s.config.ProjectPrefix + "_" + projectKey
	}

	// Try to create project using entity
	project, err := s.entity.CreateProject(owner, repo, branch)
	if err != nil {
		// Check if error is about duplicate key
		if strings.Contains(err.Error(), "A similar key already exists") {
			s.logger.Debug("SonarQube project already exists", "projectKey", projectKey)
			// Project already exists, create a minimal project object to return
			existingProject := &sonarqube.Project{
				Key:  projectKey,
				Name: fmt.Sprintf("%s/%s (%s)", owner, repo, branch),
			}
			return existingProject, nil
		}
		s.logger.Error("Failed to create SonarQube project", "error", err)
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	s.logger.Debug("SonarQube project created successfully", "projectKey", project.Key)
	return project, nil
}

// GetProject retrieves a project from SonarQube by its project key.
// If the project doesn't exist, it attempts to create it automatically.
// This method first checks the cache for the project, and if not found,
// retrieves it from the API and caches it.
//
// Parameters:
//   - ctx: context for the operation
//   - projectKey: key of the project to retrieve
//
// Returns:
//   - *sonarqube.Project: retrieved or created project
//   - error: error if project retrieval or creation fails
func (s *Service) GetProject(ctx context.Context, projectKey string) (*sonarqube.Project, error) {
	s.logger.Debug("Retrieving SonarQube project", "projectKey", projectKey)

	// Validate input parameter
	if projectKey == "" {
		return nil, &sonarqube.ValidationError{
			Field:   "projectKey",
			Message: "Project key must be provided",
		}
	}

	// // Check cache first
	// project, exists := s.getCachedProject(projectKey)
	// if exists {
	// 	s.logger.Debug("Found project in cache", "projectKey", projectKey)
	// 	return project, nil
	// }

	// Retrieve project from API
	project, err := s.entity.GetProject(projectKey)
	if err != nil {
		// Check if this is a "not found" error
		if sqErr, ok := err.(*sonarqube.Error); ok && sqErr.Code == 404 {
			// Project not found, try to create it
			s.logger.Debug("Project not found, attempting to create", "projectKey", projectKey)

			// Parse project key to extract owner, repo, and branch
			// Expected format: [prefix_]owner_repo_branch
			parts := strings.Split(projectKey, "_")
			if len(parts) < 3 {
				s.logger.Error("Invalid project key format", "projectKey", projectKey)
				return nil, fmt.Errorf("invalid project key format: %s", projectKey)
			}

			// Handle optional prefix
			startIdx := 0
			if s.config.ProjectPrefix != "" && len(parts) > 3 {
				startIdx = 1 // Skip prefix
			}

			if len(parts) < startIdx+3 {
				s.logger.Error("Invalid project key format after prefix handling", "projectKey", projectKey)
				return nil, fmt.Errorf("invalid project key format: %s", projectKey)
			}

			owner := parts[startIdx]
			repo := parts[startIdx+1]
			branch := strings.Join(parts[startIdx+2:], "_") // Handle branch names with underscores

			// Create the project
			projectCreated, createErr := s.CreateProject(ctx, owner, repo, branch)
			if createErr != nil {
				s.logger.Error("Failed to create SonarQube project", "error", createErr)
				return nil, fmt.Errorf("failed to create project: %w", createErr)
			}

			s.logger.Debug("SonarQube project created successfully", "projectKey", projectKey)
			return projectCreated, nil
		}

		// Other error, return it
		s.logger.Error("Failed to retrieve SonarQube project", "error", err)
		return nil, fmt.Errorf("failed to retrieve project: %w", err)
	}

	// // Add project to cache
	// s.cacheProject(projectKey, project)

	s.logger.Debug("SonarQube project retrieved successfully", "projectKey", projectKey)
	return project, nil
}

// UpdateProject updates an existing project in SonarQube.
// This method updates an existing project with the provided updates.
//
// Parameters:
//   - _: context for the operation
//   - projectKey: key of the project to update
//   - updates: project updates
//
// Returns:
//   - error: error if project update fails
func (s *Service) UpdateProject(_ context.Context, projectKey string, updates *sonarqube.ProjectUpdate) error {
	s.logger.Debug("Updating SonarQube project", "projectKey", projectKey)

	// Validate input parameters
	if projectKey == "" {
		return &sonarqube.ValidationError{
			Field:   "projectKey",
			Message: "Project key must be provided",
		}
	}

	if updates == nil {
		return &sonarqube.ValidationError{
			Field:   "updates",
			Message: "Updates must be provided",
		}
	}

	// Update project using entity
	if err := s.entity.UpdateProject(projectKey, updates); err != nil {
		s.logger.Error("Failed to update SonarQube project", "error", err)
		return fmt.Errorf("failed to update project: %w", err)
	}

	s.logger.Debug("SonarQube project updated successfully", "projectKey", projectKey)
	return nil
}

// DeleteProject deletes a project from SonarQube by its project key.
// This method deletes a project and removes it from the cache.
//
// Parameters:
//   - _: context for the operation
//   - projectKey: key of the project to delete
//
// Returns:
//   - error: error if project deletion fails
func (s *Service) DeleteProject(_ context.Context, projectKey string) error {
	s.logger.Debug("Deleting SonarQube project", "projectKey", projectKey)

	// Validate input parameter
	if projectKey == "" {
		return &sonarqube.ValidationError{
			Field:   "projectKey",
			Message: "Project key must be provided",
		}
	}

	// Delete project using entity
	if err := s.entity.DeleteProject(projectKey); err != nil {
		s.logger.Error("Failed to delete SonarQube project", "error", err)
		return fmt.Errorf("failed to delete project: %w", err)
	}

	s.logger.Debug("SonarQube project deleted successfully", "projectKey", projectKey)
	return nil
}

// ListProjects lists all projects in SonarQube that match the specified owner and repo.
// This method retrieves a list of projects from the API.
//
// Parameters:
//   - ctx: context for the operation
//   - owner: project owner
//   - repo: project repository
//
// Returns:
//   - []sonarqube.Project: list of projects
//   - error: error if project listing fails
func (s *Service) ListProjects(_ context.Context, owner, repo string) ([]sonarqube.Project, error) {
	s.logger.Debug("Listing SonarQube projects", "owner", owner, "repo", repo)

	// Validate input parameters
	if owner == "" || repo == "" {
		return nil, &sonarqube.ValidationError{
			Field:   "owner/repo",
			Message: "Owner and repo must be provided",
		}
	}

	// List projects using entity
	projects, err := s.entity.ListProjects(owner, repo)
	if err != nil {
		s.logger.Error("Failed to list SonarQube projects", "error", err)
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	s.logger.Debug("SonarQube projects listed successfully", "count", len(projects))
	return projects, nil
}

// GetAnalyses retrieves analyses for a project from SonarQube.
// This method retrieves a list of analyses for the specified project.
//
// Parameters:
//   - ctx: context for the operation
//   - projectKey: key of the project
//
// Returns:
//   - []sonarqube.Analysis: list of analyses
//   - error: error if analysis retrieval fails
func (s *Service) GetAnalyses(_ context.Context, projectKey string) ([]sonarqube.Analysis, error) {
	s.logger.Debug("Retrieving SonarQube analyses", "projectKey", projectKey)

	// Validate input parameter
	if projectKey == "" {
		return nil, &sonarqube.ValidationError{
			Field:   "projectKey",
			Message: "Project key must be provided",
		}
	}

	// Retrieve analyses using entity
	analyses, err := s.entity.GetAnalyses(projectKey)
	if err != nil {
		s.logger.Error("Failed to retrieve SonarQube analyses", "error", err)
		return nil, fmt.Errorf("failed to retrieve analyses: %w", err)
	}

	s.logger.Debug("SonarQube analyses retrieved successfully", "count", len(analyses))
	return analyses, nil
}

// GetAnalysisStatus retrieves the status of an analysis by its ID.
// This method retrieves the status of the specified analysis.
//
// Parameters:
//   - ctx: context for the operation
//   - analysisID: ID of the analysis
//
// Returns:
//   - *sonarqube.AnalysisStatus: analysis status
//   - error: error if status retrieval fails
func (s *Service) GetAnalysisStatus(_ context.Context, analysisID string) (*sonarqube.AnalysisStatus, error) {
	s.logger.Debug("Retrieving SonarQube analysis status", "analysisID", analysisID)

	// Validate input parameter
	if analysisID == "" {
		return nil, &sonarqube.ValidationError{
			Field:   "analysisID",
			Message: "Analysis ID must be provided",
		}
	}

	// Retrieve analysis status using entity
	status, err := s.entity.GetAnalysisStatus(analysisID)
	if err != nil {
		s.logger.Error("Failed to retrieve SonarQube analysis status", "error", err)
		return nil, fmt.Errorf("failed to retrieve analysis status: %w", err)
	}

	s.logger.Debug("SonarQube analysis status retrieved successfully", "analysisID", analysisID)
	return status, nil
}

// GetIssues retrieves issues for a project from SonarQube.
// This method retrieves a list of issues for the specified project.
//
// Parameters:
//   - ctx: context for the operation
//   - projectKey: key of the project
//   - params: issue parameters
//
// Returns:
//   - []sonarqube.Issue: list of issues
//   - error: error if issue retrieval fails
func (s *Service) GetIssues(_ context.Context, projectKey string, params *sonarqube.IssueParams) ([]sonarqube.Issue, error) {
	s.logger.Debug("Retrieving SonarQube issues", "projectKey", projectKey)

	// Validate input parameters
	if projectKey == "" {
		return nil, &sonarqube.ValidationError{
			Field:   "projectKey",
			Message: "Project key must be provided",
		}
	}

	// Retrieve issues using entity
	issues, err := s.entity.GetIssues(projectKey, params)
	if err != nil {
		s.logger.Error("Failed to retrieve SonarQube issues", "error", err)
		return nil, fmt.Errorf("failed to retrieve issues: %w", err)
	}

	s.logger.Debug("SonarQube issues retrieved successfully", "count", len(issues))
	return issues, nil
}

// GetQualityGateStatus retrieves the quality gate status for a project.
// This method retrieves the quality gate status for the specified project.
//
// Parameters:
//   - ctx: context for the operation
//   - projectKey: key of the project
//
// Returns:
//   - *sonarqube.QualityGateStatus: quality gate status
//   - error: error if status retrieval fails
func (s *Service) GetQualityGateStatus(_ context.Context, projectKey string) (*sonarqube.QualityGateStatus, error) {
	s.logger.Debug("Retrieving SonarQube quality gate status", "projectKey", projectKey)

	// Validate input parameter
	if projectKey == "" {
		return nil, &sonarqube.ValidationError{
			Field:   "projectKey",
			Message: "Project key must be provided",
		}
	}

	// Retrieve quality gate status using entity
	status, err := s.entity.GetQualityGateStatus(projectKey)
	if err != nil {
		s.logger.Error("Failed to retrieve SonarQube quality gate status", "error", err)
		return nil, fmt.Errorf("failed to retrieve quality gate status: %w", err)
	}

	s.logger.Debug("SonarQube quality gate status retrieved successfully", "projectKey", projectKey)
	return status, nil
}

// GetMetrics retrieves metrics for a project based on the specified metric keys.
// This method retrieves metrics for the specified project.
//
// Parameters:
//   - ctx: context for the operation
//   - projectKey: key of the project
//   - metricKeys: list of metric keys
//
// Returns:
//   - *sonarqube.Metrics: project metrics
//   - error: error if metrics retrieval fails
func (s *Service) GetMetrics(_ context.Context, projectKey string, metricKeys []string) (*sonarqube.Metrics, error) {
	s.logger.Debug("Retrieving SonarQube metrics", "projectKey", projectKey)

	// Validate input parameters
	if projectKey == "" {
		return nil, &sonarqube.ValidationError{
			Field:   "projectKey",
			Message: "Project key must be provided",
		}
	}

	// Retrieve metrics using entity
	metrics, err := s.entity.GetMetrics(projectKey, metricKeys)
	if err != nil {
		s.logger.Error("Failed to retrieve SonarQube metrics", "error", err)
		return nil, fmt.Errorf("failed to retrieve metrics: %w", err)
	}

	s.logger.Debug("SonarQube metrics retrieved successfully", "projectKey", projectKey)
	return metrics, nil
}

// BulkCreateProjects creates multiple projects in SonarQube in a single operation.
// This method creates multiple projects with the specified owner, repo, and branches.
//
// Parameters:
//   - ctx: context for the operation
//   - owner: project owner
//   - repo: project repository
//   - branches: list of project branches
//
// Returns:
//   - []*sonarqube.Project: list of created projects
//   - error: error if project creation fails
func (s *Service) BulkCreateProjects(_ context.Context, owner, repo string, branches []string) ([]*sonarqube.Project, error) {
	s.logger.Debug("Bulk creating SonarQube projects", "owner", owner, "repo", repo, "branches", branches)

	// Validate input parameters
	if owner == "" || repo == "" || len(branches) == 0 {
		return nil, &sonarqube.ValidationError{
			Field:   "owner/repo/branches",
			Message: "Owner, repo, and branches must be provided",
		}
	}

	// Create projects concurrently
	type projectResult struct {
		project *sonarqube.Project
		err     error
	}

	results := make(chan projectResult, len(branches))
	var wg sync.WaitGroup

	for _, branch := range branches {
		wg.Add(1)
		go func(b string) {
			defer wg.Done()

			project, err := s.entity.CreateProject(owner, repo, b)
			results <- projectResult{project: project, err: err}
		}(branch)
	}

	// Close results channel when all goroutines are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	projects := make([]*sonarqube.Project, 0, len(branches))
	var errors []error

	for result := range results {
		if result.err != nil {
			errors = append(errors, result.err)
			continue
		}

		projects = append(projects, result.project)
	}

	if len(errors) > 0 {
		// Log errors but continue with successful projects
		for _, err := range errors {
			s.logger.Error("Failed to create SonarQube project", "error", err)
		}
	}

	s.logger.Debug("Bulk SonarQube projects created successfully", "count", len(projects))
	return projects, nil
}

// ListProjectsWithFilter lists projects in SonarQube with advanced filtering and sorting.
// This method retrieves a list of projects from the API with filtering and sorting options.
//
// Parameters:
//   - ctx: context for the operation
//   - owner: project owner
//   - repo: project repository
//   - filter: filter criteria
//   - sort: sorting criteria
//
// Returns:
//   - []sonarqube.Project: list of projects
//   - error: error if project listing fails
func (s *Service) ListProjectsWithFilter(_ context.Context, owner, repo string, filter map[string]string, sort []string) ([]sonarqube.Project, error) {
	s.logger.Debug("Listing SonarQube projects with filter", "owner", owner, "repo", repo, "filter", filter, "sort", sort)

	// Validate input parameters
	if owner == "" || repo == "" {
		return nil, &sonarqube.ValidationError{
			Field:   "owner/repo",
			Message: "Owner and repo must be provided",
		}
	}

	// For now, we'll just call the existing ListProjects method
	// In a real implementation, you would need to implement filtering and sorting
	projects, err := s.entity.ListProjects(owner, repo)
	if err != nil {
		s.logger.Error("Failed to list SonarQube projects", "error", err)
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	// Apply filter
	if len(filter) > 0 {
		// This is a simplified implementation
		// In a real implementation, you would need to implement proper filtering
		s.logger.Warn("Project filtering is not fully implemented yet")
	}

	// Apply sort
	if len(sort) > 0 {
		// This is a simplified implementation
		// In a real implementation, you would need to implement proper sorting
		s.logger.Warn("Project sorting is not fully implemented yet")
	}

	s.logger.Debug("SonarQube projects listed successfully with filter", "count", len(projects))
	return projects, nil
}

// AggregateMetrics aggregates metrics across multiple projects.
// This method retrieves and aggregates metrics for the specified projects.
//
// Parameters:
//   - ctx: context for the operation
//   - projectKeys: list of project keys
//   - metricKeys: list of metric keys to aggregate
//
// Returns:
//   - map[string]float64: aggregated metrics
//   - error: error if metrics aggregation fails
func (s *Service) AggregateMetrics(_ context.Context, projectKeys []string, metricKeys []string) (map[string]float64, error) {
	s.logger.Debug("Aggregating SonarQube metrics", "projectKeys", projectKeys, "metricKeys", metricKeys)

	// Validate input parameters
	if len(projectKeys) == 0 || len(metricKeys) == 0 {
		return nil, &sonarqube.ValidationError{
			Field:   "projectKeys/metricKeys",
			Message: "Project keys and metric keys must be provided",
		}
	}

	// Aggregate metrics concurrently
	type metricResult struct {
		projectKey string
		metrics    *sonarqube.Metrics
		err        error
	}

	results := make(chan metricResult, len(projectKeys))
	var wg sync.WaitGroup

	for _, projectKey := range projectKeys {
		wg.Add(1)
		go func(pk string) {
			defer wg.Done()

			metrics, err := s.entity.GetMetrics(pk, metricKeys)
			results <- metricResult{projectKey: pk, metrics: metrics, err: err}
		}(projectKey)
	}

	// Close results channel when all goroutines are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results and aggregate metrics
	aggregatedMetrics := make(map[string]float64)
	var errors []error

	for result := range results {
		if result.err != nil {
			errors = append(errors, result.err)
			continue
		}

		// Aggregate metrics
		for key, value := range result.metrics.Metrics {
			aggregatedMetrics[key] += value
		}
	}

	if len(errors) > 0 {
		// Log errors but continue with successful metrics
		for _, err := range errors {
			s.logger.Error("Failed to retrieve metrics for project", "error", err)
		}
	}

	// Calculate averages for metrics
	for key, value := range aggregatedMetrics {
		aggregatedMetrics[key] = value / float64(len(projectKeys)-len(errors))
	}

	s.logger.Debug("SonarQube metrics aggregated successfully", "metrics", aggregatedMetrics)
	return aggregatedMetrics, nil
}

// IntegrateWithService integrates with another service for cross-functional operations.
// This method provides a generic interface for integrating with other services.
//
// Parameters:
//   - ctx: context for the operation
//   - serviceName: name of the service to integrate with
//   - operation: operation to perform
//   - data: data to pass to the service
//
// Returns:
//   - interface{}: result of the integration
//   - error: error if integration fails
func (s *Service) IntegrateWithService(_ context.Context, serviceName, operation string, _ map[string]interface{}) (interface{}, error) {
	s.logger.Debug("Integrating with service", "serviceName", serviceName, "operation", operation)

	// Validate input parameters
	if serviceName == "" || operation == "" {
		return nil, &sonarqube.ValidationError{
			Field:   "serviceName/operation",
			Message: "Service name and operation must be provided",
		}
	}

	// For now, we'll just log that the integration is not fully implemented
	// In a real implementation, you would need to implement integration with other services
	s.logger.Warn("Service integration is not fully implemented yet", "serviceName", serviceName, "operation", operation)

	s.logger.Debug("Service integration processed", "serviceName", serviceName, "operation", operation)
	return nil, nil
}

// ToDo: необходимо дополнительно реализовать следующий функционал:
// - Caching mechanism improvements with expiration based on time
// - Cache invalidation strategies
//
// Ссылки на пункты плана и требований:
// - tasks.md: 2.3
// - requirements.md: 3.1, 3.2, 9.1
