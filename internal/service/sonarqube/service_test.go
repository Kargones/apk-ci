// Package sonarqube provides tests for SonarQube service implementation.
package sonarqube

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/entity/sonarqube"
	"github.com/stretchr/testify/assert"
)

// MockAPIInterface is a mock implementation of sonarqube.APIInterface
type MockAPIInterface struct {
	authenticateError       error
	validateTokenError      error
	createProjectError      error
	createProjectResult     *sonarqube.Project
	getProjectResult        *sonarqube.Project
	getProjectError         error
	updateProjectError      error
	deleteProjectError      error
	listProjectsResult      []sonarqube.Project
	listProjectsError       error
	getAnalysesResult       []sonarqube.Analysis
	getAnalysesError        error
	getAnalysisStatusResult *sonarqube.AnalysisStatus
	getAnalysisStatusError  error
	getIssuesResult         []sonarqube.Issue
	getIssuesError          error
	getQualityGateResult    *sonarqube.QualityGateStatus
	getQualityGateError     error
	getMetricsResult        *sonarqube.Metrics
	getMetricsError         error
	getQualityProfilesResult []sonarqube.QualityProfile
	getQualityProfilesError  error
	getQualityGatesResult    []sonarqube.QualityGate
	getQualityGatesError     error
	getRulesResult           []sonarqube.Rule
	getRulesError            error
	setProjectTagsError      error
}

func (m *MockAPIInterface) Authenticate(token string) error {
	return m.authenticateError
}

func (m *MockAPIInterface) ValidateToken() error {
	return m.validateTokenError
}

func (m *MockAPIInterface) CreateProject(owner, repo, branch string) (*sonarqube.Project, error) {
	return m.createProjectResult, m.createProjectError
}

func (m *MockAPIInterface) GetProject(projectKey string) (*sonarqube.Project, error) {
	return m.getProjectResult, m.getProjectError
}

func (m *MockAPIInterface) UpdateProject(projectKey string, updates *sonarqube.ProjectUpdate) error {
	return m.updateProjectError
}

func (m *MockAPIInterface) DeleteProject(projectKey string) error {
	return m.deleteProjectError
}

func (m *MockAPIInterface) ListProjects(owner, repo string) ([]sonarqube.Project, error) {
	return m.listProjectsResult, m.listProjectsError
}

func (m *MockAPIInterface) GetAnalyses(projectKey string) ([]sonarqube.Analysis, error) {
	return m.getAnalysesResult, m.getAnalysesError
}

func (m *MockAPIInterface) GetAnalysisStatus(analysisID string) (*sonarqube.AnalysisStatus, error) {
	return m.getAnalysisStatusResult, m.getAnalysisStatusError
}

func (m *MockAPIInterface) GetIssues(projectKey string, params *sonarqube.IssueParams) ([]sonarqube.Issue, error) {
	return m.getIssuesResult, m.getIssuesError
}

func (m *MockAPIInterface) GetQualityGateStatus(projectKey string) (*sonarqube.QualityGateStatus, error) {
	return m.getQualityGateResult, m.getQualityGateError
}

func (m *MockAPIInterface) GetMetrics(projectKey string, metricKeys []string) (*sonarqube.Metrics, error) {
	return m.getMetricsResult, m.getMetricsError
}

func (m *MockAPIInterface) GetQualityProfiles(projectKey string) ([]sonarqube.QualityProfile, error) {
	return m.getQualityProfilesResult, m.getQualityProfilesError
}

func (m *MockAPIInterface) GetQualityGates() ([]sonarqube.QualityGate, error) {
	return m.getQualityGatesResult, m.getQualityGatesError
}

func (m *MockAPIInterface) GetRules(params *sonarqube.RuleParams) ([]sonarqube.Rule, error) {
	return m.getRulesResult, m.getRulesError
}

func (m *MockAPIInterface) SetProjectTags(projectKey string, tags []string) error {
	return m.setProjectTagsError
}

// TestNewSonarQubeService tests the creation of a new SonarQubeService.
func TestNewSonarQubeService(t *testing.T) {
	// Create a mock entity (we'll use a real entity for simplicity)
	cfg := &config.SonarQubeConfig{
		URL:     "http://localhost:9000",
		Token:   "test-token",
		Timeout: 30 * time.Second,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create a real entity for testing
	entity := sonarqube.NewEntity(cfg, logger)

	service := NewSonarQubeService(entity, cfg, logger)

	assert.NotNil(t, service)
	assert.Equal(t, entity, service.entity)
	assert.Equal(t, cfg, service.config)
	assert.Equal(t, logger, service.logger)
}

// TestService_ValidateToken tests the ValidateToken method
func TestService_ValidateToken(t *testing.T) {
	tests := []struct {
		name        string
		mockError   error
		expectedErr bool
	}{
		{
			name:        "successful validation",
			mockError:   nil,
			expectedErr: false,
		},
		{
			name:        "validation error",
			mockError:   errors.New("invalid token"),
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEntity := &MockAPIInterface{
				validateTokenError: tt.mockError,
			}
			cfg := &config.SonarQubeConfig{}
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			service := NewSonarQubeService(mockEntity, cfg, logger)

			err := service.ValidateToken()

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestService_UpdateProjectDescription tests the UpdateProjectDescription method
func TestService_UpdateProjectDescription(t *testing.T) {
	tests := []struct {
		name        string
		projectKey  string
		description string
		expectedErr bool
		errorMsg    string
	}{
		{
			name:        "empty project key",
			projectKey:  "",
			description: "test description",
			expectedErr: true,
			errorMsg:    "Project key must be provided",
		},
		{
			name:        "valid input",
			projectKey:  "test-project",
			description: "test description",
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEntity := &MockAPIInterface{}
			cfg := &config.SonarQubeConfig{}
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			service := NewSonarQubeService(mockEntity, cfg, logger)

			err := service.UpdateProjectDescription(context.Background(), tt.projectKey, tt.description)

			if tt.expectedErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestService_CreateProject tests the CreateProject method
func TestService_CreateProject(t *testing.T) {
	tests := []struct {
		name           string
		owner          string
		repo           string
		branch         string
		getProjectErr  error
		createErr      error
		expectedErr    bool
		expectedResult bool
	}{
		{
			name:           "project already exists",
			owner:          "test-owner",
			repo:           "test-repo",
			branch:         "main",
			getProjectErr:  nil,
			createErr:      nil,
			expectedErr:    false,
			expectedResult: true,
		},
		{
			name:           "create new project success",
			owner:          "test-owner",
			repo:           "test-repo",
			branch:         "main",
			getProjectErr:  errors.New("not found"),
			createErr:      nil,
			expectedErr:    false,
			expectedResult: true,
		},
		{
			name:           "create project fails",
			owner:          "test-owner",
			repo:           "test-repo",
			branch:         "main",
			getProjectErr:  errors.New("not found"),
			createErr:      errors.New("creation failed"),
			expectedErr:    true,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEntity := &MockAPIInterface{
			getProjectError:    tt.getProjectErr,
			createProjectError: tt.createErr,
		}
			
		if tt.getProjectErr == nil {
			mockEntity.getProjectResult = &sonarqube.Project{
				Key:  "test-owner_test-repo_main",
				Name: "test-owner/test-repo (main)",
			}
		}
		
		// Настройка результата для CreateProject при успешном создании
		if tt.createErr == nil {
			mockEntity.createProjectResult = &sonarqube.Project{
				Key:  "test-owner_test-repo_main",
				Name: "test-owner/test-repo (main)",
			}
		}

			cfg := &config.SonarQubeConfig{}
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			service := NewSonarQubeService(mockEntity, cfg, logger)

			result, err := service.CreateProject(context.Background(), tt.owner, tt.repo, tt.branch)

			if tt.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				if tt.expectedResult {
					assert.NotNil(t, result)
				}
			}
		})
	}
}

// TestService_GetProject tests the GetProject method
func TestService_GetProject(t *testing.T) {
	tests := []struct {
		name        string
		projectKey  string
		mockResult  *sonarqube.Project
		mockError   error
		expectedErr bool
	}{
		{
			name:       "successful get",
			projectKey: "test-project",
			mockResult: &sonarqube.Project{
				Key:  "test-project",
				Name: "Test Project",
			},
			mockError:   nil,
			expectedErr: false,
		},
		{
			name:        "project not found",
			projectKey:  "nonexistent",
			mockResult:  nil,
			mockError:   errors.New("not found"),
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEntity := &MockAPIInterface{
				getProjectResult: tt.mockResult,
				getProjectError:  tt.mockError,
			}
			cfg := &config.SonarQubeConfig{}
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			service := NewSonarQubeService(mockEntity, cfg, logger)

			result, err := service.GetProject(context.Background(), tt.projectKey)

			if tt.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.mockResult, result)
			}
		})
	}
}

// TestService_UpdateProject тестирует обновление проекта
func TestService_UpdateProject(t *testing.T) {
	mockAPI := &MockAPIInterface{}
	config := &config.SonarQubeConfig{
		URL:   "http://localhost:9000",
		Token: "test-token",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	service := NewSonarQubeService(mockAPI, config, logger)

	t.Run("successful update project", func(t *testing.T) {
		mockAPI.updateProjectError = nil
		updates := &sonarqube.ProjectUpdate{
			Description: "Updated description",
		}

		err := service.UpdateProject(context.Background(), "test-project", updates)

		assert.NoError(t, err)
	})

	t.Run("update project with error", func(t *testing.T) {
		mockAPI.updateProjectError = errors.New("update failed")
		updates := &sonarqube.ProjectUpdate{
			Description: "Updated description",
		}

		err := service.UpdateProject(context.Background(), "test-project", updates)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update project")
	})

	t.Run("update project with empty key", func(t *testing.T) {
		updates := &sonarqube.ProjectUpdate{
			Description: "Updated description",
		}

		err := service.UpdateProject(context.Background(), "", updates)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Project key must be provided")
	})
}

// TestService_DeleteProject тестирует удаление проекта
func TestService_DeleteProject(t *testing.T) {
	mockAPI := &MockAPIInterface{}
	config := &config.SonarQubeConfig{
		URL:   "http://localhost:9000",
		Token: "test-token",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	service := NewSonarQubeService(mockAPI, config, logger)

	t.Run("successful delete project", func(t *testing.T) {
		mockAPI.deleteProjectError = nil

		err := service.DeleteProject(context.Background(), "test-project")

		assert.NoError(t, err)
	})

	t.Run("delete project with error", func(t *testing.T) {
		mockAPI.deleteProjectError = errors.New("delete failed")

		err := service.DeleteProject(context.Background(), "test-project")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete project")
	})

	t.Run("delete project with empty key", func(t *testing.T) {
		err := service.DeleteProject(context.Background(), "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Project key must be provided")
	})
}

// TestService_ListProjects тестирует получение списка проектов
func TestService_ListProjects(t *testing.T) {
	mockAPI := &MockAPIInterface{}
	config := &config.SonarQubeConfig{
		URL:   "http://localhost:9000",
		Token: "test-token",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	service := NewSonarQubeService(mockAPI, config, logger)

	t.Run("successful list projects", func(t *testing.T) {
		expectedProjects := []sonarqube.Project{
			{Key: "project1", Name: "Project 1"},
			{Key: "project2", Name: "Project 2"},
		}
		mockAPI.listProjectsResult = expectedProjects
		mockAPI.listProjectsError = nil

		projects, err := service.ListProjects(context.Background(), "owner", "repo")

		assert.NoError(t, err)
		assert.Equal(t, expectedProjects, projects)
	})

	t.Run("list projects with error", func(t *testing.T) {
		mockAPI.listProjectsResult = nil
		mockAPI.listProjectsError = errors.New("list failed")

		projects, err := service.ListProjects(context.Background(), "owner", "repo")

		assert.Error(t, err)
		assert.Nil(t, projects)
		assert.Contains(t, err.Error(), "failed to list projects")
	})
}

// TestService_GetAnalyses тестирует получение анализов
func TestService_GetAnalyses(t *testing.T) {
	mockAPI := &MockAPIInterface{}
	config := &config.SonarQubeConfig{
		URL:   "http://localhost:9000",
		Token: "test-token",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	service := NewSonarQubeService(mockAPI, config, logger)

	t.Run("successful get analyses", func(t *testing.T) {
		now := &sonarqube.Time{Time: time.Now()}
		expectedAnalyses := []sonarqube.Analysis{
			{ID: "analysis1", Date: now},
			{ID: "analysis2", Date: now},
		}
		mockAPI.getAnalysesResult = expectedAnalyses
		mockAPI.getAnalysesError = nil

		analyses, err := service.GetAnalyses(context.Background(), "test-project")

		assert.NoError(t, err)
		assert.Equal(t, expectedAnalyses, analyses)
	})

	t.Run("get analyses with error", func(t *testing.T) {
		mockAPI.getAnalysesResult = nil
		mockAPI.getAnalysesError = errors.New("analyses failed")

		analyses, err := service.GetAnalyses(context.Background(), "test-project")

		assert.Error(t, err)
		assert.Nil(t, analyses)
		assert.Contains(t, err.Error(), "failed to retrieve analyses")
	})

	t.Run("get analyses with empty key", func(t *testing.T) {
		analyses, err := service.GetAnalyses(context.Background(), "")

		assert.Error(t, err)
		assert.Nil(t, analyses)
		assert.Contains(t, err.Error(), "Project key must be provided")
	})
}

// TestService_GetMetrics тестирует получение метрик
func TestService_GetMetrics(t *testing.T) {
	mockAPI := &MockAPIInterface{}
	config := &config.SonarQubeConfig{
		URL:   "http://localhost:9000",
		Token: "test-token",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	service := NewSonarQubeService(mockAPI, config, logger)

	t.Run("successful get metrics", func(t *testing.T) {
		expectedMetrics := &sonarqube.Metrics{
			Metrics: map[string]float64{
				"coverage":     85.5,
				"complexity":   100,
				"lines_of_code": 1000,
			},
		}
		mockAPI.getMetricsResult = expectedMetrics
		mockAPI.getMetricsError = nil

		metrics, err := service.GetMetrics(context.Background(), "test-project", []string{"coverage", "complexity"})

		assert.NoError(t, err)
		assert.Equal(t, expectedMetrics, metrics)
	})

	t.Run("get metrics with error", func(t *testing.T) {
		mockAPI.getMetricsResult = nil
		mockAPI.getMetricsError = errors.New("metrics failed")

		metrics, err := service.GetMetrics(context.Background(), "test-project", []string{"coverage"})

		assert.Error(t, err)
		assert.Nil(t, metrics)
		assert.Contains(t, err.Error(), "failed to retrieve metrics")
	})

	t.Run("get metrics with empty key", func(t *testing.T) {
		metrics, err := service.GetMetrics(context.Background(), "", []string{"coverage"})

		assert.Error(t, err)
		assert.Nil(t, metrics)
		assert.Contains(t, err.Error(), "Project key must be provided")
	})
}

// TestService_UpdateProjectAdministrators тестирует обновление администраторов проекта
func TestService_UpdateProjectAdministrators(t *testing.T) {
	config := &config.SonarQubeConfig{
		URL:   "http://localhost:9000",
		Token: "test-token",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mockAPI := &MockAPIInterface{}

	service := NewSonarQubeService(mockAPI, config, logger)

	t.Run("successful update administrators", func(t *testing.T) {
		administrators := []string{"admin1", "admin2"}

		err := service.UpdateProjectAdministrators(context.Background(), "test-project", administrators)

		assert.NoError(t, err)
	})

	t.Run("update administrators with empty key", func(t *testing.T) {
		administrators := []string{"admin1"}

		err := service.UpdateProjectAdministrators(context.Background(), "", administrators)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Project key must be provided")
	})
}

// TestService_GetAnalysisStatus тестирует получение статуса анализа
func TestService_GetAnalysisStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := &config.SonarQubeConfig{}
	mockAPI := &MockAPIInterface{}

	service := NewSonarQubeService(mockAPI, config, logger)

	// Тест успешного получения статуса анализа
	expectedStatus := &sonarqube.AnalysisStatus{
		Status: "SUCCESS",
	}
	mockAPI.getAnalysisStatusResult = expectedStatus

	status, err := service.GetAnalysisStatus(context.Background(), "analysis-123")
	assert.NoError(t, err)
	assert.Equal(t, expectedStatus, status)

	// Тест с ошибкой API
	mockAPI.getAnalysisStatusError = errors.New("API error")
	status, err = service.GetAnalysisStatus(context.Background(), "analysis-123")
	assert.Error(t, err)
	assert.Nil(t, status)

	// Тест с пустым ID анализа
	mockAPI.getAnalysisStatusError = nil
	status, err = service.GetAnalysisStatus(context.Background(), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Analysis ID must be provided")
}

// TestService_GetIssues тестирует получение проблем проекта
func TestService_GetIssues(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := &config.SonarQubeConfig{}
	mockAPI := &MockAPIInterface{}

	service := NewSonarQubeService(mockAPI, config, logger)

	// Тест успешного получения проблем
	expectedIssues := []sonarqube.Issue{
		{
			Key:      "key-1",
			Rule:     "rule-1",
			Severity: "MAJOR",
		},
		{
			Key:      "key-2",
			Rule:     "rule-2",
			Severity: "MINOR",
		},
	}
	mockAPI.getIssuesResult = expectedIssues

	params := &sonarqube.IssueParams{Severities: []string{"MAJOR", "MINOR"}}
	issues, err := service.GetIssues(context.Background(), "test-project", params)
	assert.NoError(t, err)
	assert.Equal(t, expectedIssues, issues)

	// Тест с ошибкой API
	mockAPI.getIssuesError = errors.New("API error")
	issues, err = service.GetIssues(context.Background(), "test-project", params)
	assert.Error(t, err)
	assert.Nil(t, issues)

	// Тест с пустым ключом проекта
	mockAPI.getIssuesError = nil
	issues, err = service.GetIssues(context.Background(), "", params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Project key must be provided")
}

// TestService_GetQualityGateStatus тестирует получение статуса Quality Gate
func TestService_GetQualityGateStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := &config.SonarQubeConfig{}
	mockAPI := &MockAPIInterface{}

	service := NewSonarQubeService(mockAPI, config, logger)

	// Тест успешного получения статуса Quality Gate
	expectedStatus := &sonarqube.QualityGateStatus{
		Status: "OK",
	}
	mockAPI.getQualityGateResult = expectedStatus

	status, err := service.GetQualityGateStatus(context.Background(), "test-project")
	assert.NoError(t, err)
	assert.Equal(t, expectedStatus, status)

	// Тест с ошибкой API
	mockAPI.getQualityGateError = errors.New("API error")
	status, err = service.GetQualityGateStatus(context.Background(), "test-project")
	assert.Error(t, err)
	assert.Nil(t, status)

	// Тест с пустым ключом проекта
	mockAPI.getQualityGateError = nil
	status, err = service.GetQualityGateStatus(context.Background(), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Project key must be provided")
}
