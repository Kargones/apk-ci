package reportbranch

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/Kargones/apk-ci/internal/adapter/sonarqube"
	"github.com/Kargones/apk-ci/internal/adapter/sonarqube/sonarqubetest"
	"github.com/Kargones/apk-ci/internal/command/handlers/sonarqube/shared"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecute_NilConfig проверяет обработку nil конфигурации.
func TestExecute_NilConfig(t *testing.T) {
	h := &ReportBranchHandler{}

	err := h.Execute(context.Background(), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), shared.ErrConfigMissing)
}

// TestExecute_MissingBranch проверяет обработку отсутствующей ветки.
func TestExecute_MissingBranch(t *testing.T) {
	h := &ReportBranchHandler{}
	cfg := &config.Config{
		Owner: "owner",
		Repo:  "repo",
	}

	err := h.Execute(context.Background(), cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), shared.ErrBranchMissing)
}

// TestExecute_MissingOwnerRepo проверяет обработку отсутствующих owner/repo.
func TestExecute_MissingOwnerRepo(t *testing.T) {
	tests := []struct {
		name  string
		owner string
		repo  string
	}{
		{name: "missing owner", owner: "", repo: "repo"},
		{name: "missing repo", owner: "owner", repo: ""},
		{name: "both missing", owner: "", repo: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &ReportBranchHandler{}
			cfg := &config.Config{
				BranchForScan: "feature-123",
				Owner:         tt.owner,
				Repo:          tt.repo,
			}

			err := h.Execute(context.Background(), cfg)

			require.Error(t, err)
			assert.Contains(t, err.Error(), shared.ErrMissingOwnerRepo)
		})
	}
}

// TestExecute_NilSonarQubeClient проверяет обработку nil SonarQube клиента.
func TestExecute_NilSonarQubeClient(t *testing.T) {
	h := &ReportBranchHandler{
		sonarqubeClient: nil,
	}
	cfg := &config.Config{
		BranchForScan: "feature-123",
		Owner:         "owner",
		Repo:          "repo",
	}

	err := h.Execute(context.Background(), cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), shared.ErrConfigMissing)
}

// TestExecute_ProjectNotFound проверяет обработку отсутствующего проекта.
func TestExecute_ProjectNotFound(t *testing.T) {
	sqClient := &sonarqubetest.MockClient{
		GetProjectFunc: func(_ context.Context, _ string) (*sonarqube.Project, error) {
			return nil, errors.New("project not found")
		},
	}

	h := &ReportBranchHandler{sonarqubeClient: sqClient}
	cfg := &config.Config{
		BranchForScan: "feature-123",
		Owner:         "owner",
		Repo:          "repo",
	}

	err := h.Execute(context.Background(), cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), shared.ErrProjectNotFound)
}

// TestExecute_Success проверяет успешное выполнение команды.
func TestExecute_Success(t *testing.T) {
	sqClient := &sonarqubetest.MockClient{
		GetProjectFunc: func(_ context.Context, key string) (*sonarqube.Project, error) {
			return &sonarqube.Project{Key: key}, nil
		},
		GetMetricsFunc: func(_ context.Context, projectKey string, metricKeys []string) (*sonarqube.Metrics, error) {
			return &sonarqube.Metrics{
				ProjectKey: projectKey,
				Measures: map[string]string{
					"bugs":                      "3",
					"vulnerabilities":           "1",
					"code_smells":               "15",
					"coverage":                  "78.5",
					"duplicated_lines_density":  "2.3",
					"ncloc":                     "1500",
				},
			}, nil
		},
		GetQualityGateStatusFunc: func(_ context.Context, _ string) (*sonarqube.QualityGateStatus, error) {
			return &sonarqube.QualityGateStatus{Status: "OK"}, nil
		},
		GetIssuesFunc: func(_ context.Context, _ sonarqube.GetIssuesOptions) ([]sonarqube.Issue, error) {
			return []sonarqube.Issue{
				{Key: "1", Type: "BUG", Severity: "MAJOR"},
				{Key: "2", Type: "CODE_SMELL", Severity: "MINOR"},
				{Key: "3", Type: "CODE_SMELL", Severity: "INFO"},
			}, nil
		},
	}

	h := &ReportBranchHandler{sonarqubeClient: sqClient}
	cfg := &config.Config{
		BranchForScan: "feature-123",
		Owner:         "owner",
		Repo:          "repo",
	}

	err := h.Execute(context.Background(), cfg)

	require.NoError(t, err)
}

// TestExecute_WithBaseComparison проверяет сравнение с base-веткой (main).
func TestExecute_WithBaseComparison(t *testing.T) {
	getProjectCalls := 0
	getMetricsCalls := 0

	sqClient := &sonarqubetest.MockClient{
		GetProjectFunc: func(_ context.Context, key string) (*sonarqube.Project, error) {
			getProjectCalls++
			return &sonarqube.Project{Key: key}, nil
		},
		GetMetricsFunc: func(_ context.Context, projectKey string, _ []string) (*sonarqube.Metrics, error) {
			getMetricsCalls++
			if strings.HasSuffix(projectKey, "_main") {
				// Base метрики
				return &sonarqube.Metrics{
					ProjectKey: projectKey,
					Measures: map[string]string{
						"bugs":                      "1",
						"vulnerabilities":           "0",
						"code_smells":               "10",
						"coverage":                  "80.0",
						"duplicated_lines_density":  "2.0",
						"ncloc":                     "1400",
					},
				}, nil
			}
			// Текущие метрики
			return &sonarqube.Metrics{
				ProjectKey: projectKey,
				Measures: map[string]string{
					"bugs":                      "3",
					"vulnerabilities":           "1",
					"code_smells":               "15",
					"coverage":                  "78.5",
					"duplicated_lines_density":  "2.3",
					"ncloc":                     "1500",
				},
			}, nil
		},
		GetQualityGateStatusFunc: func(_ context.Context, _ string) (*sonarqube.QualityGateStatus, error) {
			return &sonarqube.QualityGateStatus{Status: "OK"}, nil
		},
		GetIssuesFunc: func(_ context.Context, _ sonarqube.GetIssuesOptions) ([]sonarqube.Issue, error) {
			return []sonarqube.Issue{}, nil
		},
	}

	h := &ReportBranchHandler{sonarqubeClient: sqClient}
	cfg := &config.Config{
		BranchForScan: "feature-123",
		Owner:         "owner",
		Repo:          "repo",
	}

	err := h.Execute(context.Background(), cfg)

	require.NoError(t, err)
	// Проверяем что GetProject был вызван дважды (для текущего и base проекта)
	assert.Equal(t, 2, getProjectCalls)
	// Проверяем что GetMetrics был вызван дважды (для текущего и base проекта)
	assert.Equal(t, 2, getMetricsCalls)
}

// TestExecute_BaseProjectNotFound проверяет случай когда base-проект не найден.
func TestExecute_BaseProjectNotFound(t *testing.T) {
	sqClient := &sonarqubetest.MockClient{
		GetProjectFunc: func(_ context.Context, key string) (*sonarqube.Project, error) {
			if strings.HasSuffix(key, "_main") {
				return nil, errors.New("project not found")
			}
			return &sonarqube.Project{Key: key}, nil
		},
		GetMetricsFunc: func(_ context.Context, projectKey string, _ []string) (*sonarqube.Metrics, error) {
			return &sonarqube.Metrics{
				ProjectKey: projectKey,
				Measures: map[string]string{
					"bugs":                      "3",
					"vulnerabilities":           "1",
					"code_smells":               "15",
					"coverage":                  "78.5",
					"duplicated_lines_density":  "2.3",
					"ncloc":                     "1500",
				},
			}, nil
		},
		GetQualityGateStatusFunc: func(_ context.Context, _ string) (*sonarqube.QualityGateStatus, error) {
			return &sonarqube.QualityGateStatus{Status: "OK"}, nil
		},
		GetIssuesFunc: func(_ context.Context, _ sonarqube.GetIssuesOptions) ([]sonarqube.Issue, error) {
			return []sonarqube.Issue{}, nil
		},
	}

	h := &ReportBranchHandler{sonarqubeClient: sqClient}
	cfg := &config.Config{
		BranchForScan: "feature-123",
		Owner:         "owner",
		Repo:          "repo",
	}

	err := h.Execute(context.Background(), cfg)

	// Команда должна успешно выполниться даже если base-проект не найден
	require.NoError(t, err)
}

// TestExecute_BaseMetricsError проверяет случай когда GetMetrics для base-проекта возвращает ошибку.
// H-2 fix: добавлен недостающий тест для покрытия edge case в buildComparison.
func TestExecute_BaseMetricsError(t *testing.T) {
	getMetricsCalls := 0

	sqClient := &sonarqubetest.MockClient{
		GetProjectFunc: func(_ context.Context, key string) (*sonarqube.Project, error) {
			// Оба проекта существуют
			return &sonarqube.Project{Key: key}, nil
		},
		GetMetricsFunc: func(_ context.Context, projectKey string, _ []string) (*sonarqube.Metrics, error) {
			getMetricsCalls++
			if strings.HasSuffix(projectKey, "_main") {
				// Ошибка при получении метрик base-проекта
				return nil, errors.New("metrics API error for base project")
			}
			// Успешное получение метрик текущего проекта
			return &sonarqube.Metrics{
				ProjectKey: projectKey,
				Measures: map[string]string{
					"bugs":                     "3",
					"vulnerabilities":          "1",
					"code_smells":              "15",
					"coverage":                 "78.5",
					"duplicated_lines_density": "2.3",
					"ncloc":                    "1500",
				},
			}, nil
		},
		GetQualityGateStatusFunc: func(_ context.Context, _ string) (*sonarqube.QualityGateStatus, error) {
			return &sonarqube.QualityGateStatus{Status: "OK"}, nil
		},
		GetIssuesFunc: func(_ context.Context, _ sonarqube.GetIssuesOptions) ([]sonarqube.Issue, error) {
			return []sonarqube.Issue{}, nil
		},
	}

	h := &ReportBranchHandler{sonarqubeClient: sqClient}
	cfg := &config.Config{
		BranchForScan: "feature-123",
		Owner:         "owner",
		Repo:          "repo",
	}

	err := h.Execute(context.Background(), cfg)

	// Команда должна успешно выполниться даже если метрики base-проекта недоступны
	require.NoError(t, err)
	// Проверяем что GetMetrics был вызван дважды (для текущего и base проекта)
	assert.Equal(t, 2, getMetricsCalls)
}

// TestExecute_JSONOutput проверяет JSON формат вывода.
func TestExecute_JSONOutput(t *testing.T) {
	// Устанавливаем формат JSON
	originalFormat := os.Getenv("BR_OUTPUT_FORMAT")
	_ = os.Setenv("BR_OUTPUT_FORMAT", "json")
	defer os.Setenv("BR_OUTPUT_FORMAT", originalFormat) //nolint:errcheck

	sqClient := &sonarqubetest.MockClient{
		GetProjectFunc: func(_ context.Context, key string) (*sonarqube.Project, error) {
			return &sonarqube.Project{Key: key}, nil
		},
		GetMetricsFunc: func(_ context.Context, projectKey string, _ []string) (*sonarqube.Metrics, error) {
			return &sonarqube.Metrics{
				ProjectKey: projectKey,
				Measures: map[string]string{
					"bugs":                      "3",
					"vulnerabilities":           "1",
					"code_smells":               "15",
					"coverage":                  "78.5",
					"duplicated_lines_density":  "2.3",
					"ncloc":                     "1500",
				},
			}, nil
		},
		GetQualityGateStatusFunc: func(_ context.Context, _ string) (*sonarqube.QualityGateStatus, error) {
			return &sonarqube.QualityGateStatus{Status: "OK"}, nil
		},
		GetIssuesFunc: func(_ context.Context, _ sonarqube.GetIssuesOptions) ([]sonarqube.Issue, error) {
			return []sonarqube.Issue{}, nil
		},
	}

	h := &ReportBranchHandler{sonarqubeClient: sqClient}
	cfg := &config.Config{
		BranchForScan: "feature-123",
		Owner:         "owner",
		Repo:          "repo",
	}

	err := h.Execute(context.Background(), cfg)

	require.NoError(t, err)
}

// TestExecute_MetricsAPIError проверяет обработку ошибки GetMetrics.
func TestExecute_MetricsAPIError(t *testing.T) {
	sqClient := &sonarqubetest.MockClient{
		GetProjectFunc: func(_ context.Context, key string) (*sonarqube.Project, error) {
			return &sonarqube.Project{Key: key}, nil
		},
		GetMetricsFunc: func(_ context.Context, _ string, _ []string) (*sonarqube.Metrics, error) {
			return nil, errors.New("metrics API error")
		},
	}

	h := &ReportBranchHandler{sonarqubeClient: sqClient}
	cfg := &config.Config{
		BranchForScan: "feature-123",
		Owner:         "owner",
		Repo:          "repo",
	}

	err := h.Execute(context.Background(), cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), shared.ErrSonarQubeAPI)
}

// TestExecute_QualityGateAPIError проверяет обработку ошибки GetQualityGateStatus.
func TestExecute_QualityGateAPIError(t *testing.T) {
	sqClient := &sonarqubetest.MockClient{
		GetProjectFunc: func(_ context.Context, key string) (*sonarqube.Project, error) {
			return &sonarqube.Project{Key: key}, nil
		},
		GetMetricsFunc: func(_ context.Context, projectKey string, _ []string) (*sonarqube.Metrics, error) {
			return &sonarqube.Metrics{
				ProjectKey: projectKey,
				Measures:   map[string]string{},
			}, nil
		},
		GetQualityGateStatusFunc: func(_ context.Context, _ string) (*sonarqube.QualityGateStatus, error) {
			return nil, errors.New("quality gate API error")
		},
	}

	h := &ReportBranchHandler{sonarqubeClient: sqClient}
	cfg := &config.Config{
		BranchForScan: "feature-123",
		Owner:         "owner",
		Repo:          "repo",
	}

	err := h.Execute(context.Background(), cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), shared.ErrSonarQubeAPI)
}

// TestExecute_IssuesAPIError проверяет обработку ошибки GetIssues.
func TestExecute_IssuesAPIError(t *testing.T) {
	sqClient := &sonarqubetest.MockClient{
		GetProjectFunc: func(_ context.Context, key string) (*sonarqube.Project, error) {
			return &sonarqube.Project{Key: key}, nil
		},
		GetMetricsFunc: func(_ context.Context, projectKey string, _ []string) (*sonarqube.Metrics, error) {
			return &sonarqube.Metrics{
				ProjectKey: projectKey,
				Measures:   map[string]string{},
			}, nil
		},
		GetQualityGateStatusFunc: func(_ context.Context, _ string) (*sonarqube.QualityGateStatus, error) {
			return &sonarqube.QualityGateStatus{Status: "OK"}, nil
		},
		GetIssuesFunc: func(_ context.Context, _ sonarqube.GetIssuesOptions) ([]sonarqube.Issue, error) {
			return nil, errors.New("issues API error")
		},
	}

	h := &ReportBranchHandler{sonarqubeClient: sqClient}
	cfg := &config.Config{
		BranchForScan: "feature-123",
		Owner:         "owner",
		Repo:          "repo",
	}

	err := h.Execute(context.Background(), cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), shared.ErrSonarQubeAPI)
}

// TestBuildIssuesSummary проверяет построение сводки по проблемам.
func TestBuildIssuesSummary(t *testing.T) {
	tests := []struct {
		name            string
		issues          []sonarqube.Issue
		expectedTotal   int
		expectedBugs    int
		expectedCritical int
	}{
		{
			name:            "empty issues",
			issues:          []sonarqube.Issue{},
			expectedTotal:   0,
			expectedBugs:    0,
			expectedCritical: 0,
		},
		{
			name: "mixed issues",
			issues: []sonarqube.Issue{
				{Type: "BUG", Severity: "CRITICAL"},
				{Type: "BUG", Severity: "MAJOR"},
				{Type: "CODE_SMELL", Severity: "MINOR"},
				{Type: "VULNERABILITY", Severity: "BLOCKER"},
			},
			expectedTotal:   4,
			expectedBugs:    2,
			expectedCritical: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := buildIssuesSummary(tt.issues)

			assert.Equal(t, tt.expectedTotal, summary.Total)
			assert.Equal(t, tt.expectedBugs, summary.ByType["BUG"])
			assert.Equal(t, tt.expectedCritical, summary.BySeverity["CRITICAL"])
		})
	}
}

// TestBuildQualityMetrics проверяет построение качественных метрик.
func TestBuildQualityMetrics(t *testing.T) {
	tests := []struct {
		name     string
		metrics  *sonarqube.Metrics
		expected *QualityMetrics
	}{
		{
			name:     "nil metrics",
			metrics:  nil,
			expected: &QualityMetrics{},
		},
		{
			name: "nil measures",
			metrics: &sonarqube.Metrics{
				ProjectKey: "test",
				Measures:   nil,
			},
			expected: &QualityMetrics{},
		},
		{
			name: "valid metrics",
			metrics: &sonarqube.Metrics{
				ProjectKey: "test",
				Measures: map[string]string{
					"bugs":                      "5",
					"vulnerabilities":           "2",
					"code_smells":               "20",
					"coverage":                  "85.5",
					"duplicated_lines_density":  "3.2",
					"ncloc":                     "2000",
				},
			},
			expected: &QualityMetrics{
				Bugs:                   5,
				Vulnerabilities:        2,
				CodeSmells:             20,
				Coverage:               85.5,
				DuplicatedLinesDensity: 3.2,
				Ncloc:                  2000,
			},
		},
		{
			name: "invalid metric values",
			metrics: &sonarqube.Metrics{
				ProjectKey: "test",
				Measures: map[string]string{
					"bugs":     "invalid",
					"coverage": "not-a-number",
				},
			},
			expected: &QualityMetrics{
				Bugs:     0,
				Coverage: 0.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildQualityMetrics(tt.metrics)

			assert.Equal(t, tt.expected.Bugs, result.Bugs)
			assert.Equal(t, tt.expected.Vulnerabilities, result.Vulnerabilities)
			assert.Equal(t, tt.expected.CodeSmells, result.CodeSmells)
			assert.InDelta(t, tt.expected.Coverage, result.Coverage, 0.01)
			assert.InDelta(t, tt.expected.DuplicatedLinesDensity, result.DuplicatedLinesDensity, 0.01)
			assert.Equal(t, tt.expected.Ncloc, result.Ncloc)
		})
	}
}

// TestBranchReportData_WriteText проверяет текстовый вывод отчёта.
func TestBranchReportData_WriteText(t *testing.T) {
	data := &BranchReportData{
		Branch:            "feature-123",
		ProjectKey:        "owner_repo_feature-123",
		QualityGateStatus: "OK",
		Metrics: &QualityMetrics{
			Bugs:                   3,
			Vulnerabilities:        1,
			CodeSmells:             15,
			Coverage:               78.5,
			DuplicatedLinesDensity: 2.3,
			Ncloc:                  1500,
		},
		IssuesSummary: &IssuesSummary{
			Total: 19,
			ByType: map[string]int{
				"BUG":           3,
				"VULNERABILITY": 1,
				"CODE_SMELL":    15,
			},
			BySeverity: map[string]int{
				"BLOCKER":  0,
				"CRITICAL": 1,
				"MAJOR":    5,
				"MINOR":    10,
				"INFO":     3,
			},
		},
		Comparison: &BranchComparison{
			BaseBranch:         "main",
			BaseProjectKey:     "owner_repo_main",
			NewBugs:            2,
			NewVulnerabilities: 1,
			NewCodeSmells:      5,
			CoverageDelta:      -1.5,
		},
	}

	var buf bytes.Buffer
	err := data.writeText(&buf)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "feature-123")
	assert.Contains(t, output, "owner_repo_feature-123")
	assert.Contains(t, output, "✅")  // OK status
	assert.Contains(t, output, "Баги:")
	assert.Contains(t, output, "78.5%")
	assert.Contains(t, output, "Сравнение с main")
	assert.Contains(t, output, "+2") // NewBugs
}

// TestBranchReportData_WriteText_BaseNotFound проверяет вывод когда base не найден.
func TestBranchReportData_WriteText_BaseNotFound(t *testing.T) {
	data := &BranchReportData{
		Branch:            "feature-123",
		ProjectKey:        "owner_repo_feature-123",
		QualityGateStatus: "ERROR",
		Metrics:           &QualityMetrics{},
		IssuesSummary: &IssuesSummary{
			Total:      0,
			ByType:     map[string]int{"BUG": 0, "VULNERABILITY": 0, "CODE_SMELL": 0},
			BySeverity: map[string]int{"BLOCKER": 0, "CRITICAL": 0, "MAJOR": 0, "MINOR": 0, "INFO": 0},
		},
		Comparison: &BranchComparison{
			BaseBranch:   "main",
			BaseNotFound: true,
		},
	}

	var buf bytes.Buffer
	err := data.writeText(&buf)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "❌")  // ERROR status
	assert.Contains(t, output, "Base-проект не найден")
}

// TestQualityGateIcon проверяет получение иконки Quality Gate.
func TestQualityGateIcon(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"OK", "✅"},
		{"ERROR", "❌"},
		{"WARN", "⚠️"},
		{"UNKNOWN", "❓"},
		{"", "❓"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			assert.Equal(t, tt.expected, qualityGateIcon(tt.status))
		})
	}
}

// TestFormatDelta проверяет форматирование дельты.
func TestFormatDelta(t *testing.T) {
	tests := []struct {
		delta    int
		expected string
	}{
		{0, "0"},
		{5, "+5"},
		{-3, "-3"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatDelta(tt.delta))
		})
	}
}

// TestFormatCoverageDelta проверяет форматирование изменения покрытия.
func TestFormatCoverageDelta(t *testing.T) {
	tests := []struct {
		delta    float64
		expected string
	}{
		{0.0, "0.0%"},
		{5.5, "+5.5%"},
		{-3.2, "-3.2%"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatCoverageDelta(tt.delta))
		})
	}
}

// TestHandlerName проверяет имя handler.
func TestHandlerName(t *testing.T) {
	h := &ReportBranchHandler{}
	assert.Equal(t, "nr-sq-report-branch", h.Name())
}

// TestHandlerDescription проверяет описание handler.
func TestHandlerDescription(t *testing.T) {
	h := &ReportBranchHandler{}
	assert.NotEmpty(t, h.Description())
}

// TestIsValidBranchForScanning проверяет валидацию формата ветки.
func TestIsValidBranchForScanning(t *testing.T) {
	tests := []struct {
		branch   string
		expected bool
	}{
		{"main", true},
		{"t123456", true},
		{"t1234567", true},
		{"t12345", false},   // слишком короткий номер
		{"t12345678", false}, // слишком длинный номер
		{"feature-123", false},
		{"develop", false},
		{"t12345a", false}, // буква вместо цифры
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			// L-2 fix: используем shared.IsValidBranchForScanning
			assert.Equal(t, tt.expected, shared.IsValidBranchForScanning(tt.branch))
		})
	}
}

// TestBranchReportData_WriteText_NilMaps проверяет defensive nil checks для maps.
// M-3 fix: тест для проверки что nil maps не вызывают panic.
func TestBranchReportData_WriteText_NilMaps(t *testing.T) {
	data := &BranchReportData{
		Branch:            "feature-123",
		ProjectKey:        "owner_repo_feature-123",
		QualityGateStatus: "OK",
		Metrics:           &QualityMetrics{},
		IssuesSummary: &IssuesSummary{
			Total:      0,
			ByType:     nil, // nil map
			BySeverity: nil, // nil map
		},
	}

	var buf bytes.Buffer
	err := data.writeText(&buf)

	require.NoError(t, err)
	output := buf.String()

	// Проверяем что вывод содержит нули для всех типов и severity
	assert.Contains(t, output, "BUG=0")
	assert.Contains(t, output, "BLOCKER=0")
}

// failingWriter реализует io.Writer, который всегда возвращает ошибку.
// Используется для тестирования error handling в writeText.
type failingWriter struct{}

func (f *failingWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}

// TestBranchReportData_WriteText_WriterError проверяет обработку ошибки записи.
// M-4 fix: тест для проверки error propagation при ошибке io.Writer.
func TestBranchReportData_WriteText_WriterError(t *testing.T) {
	data := &BranchReportData{
		Branch:            "feature-123",
		ProjectKey:        "owner_repo_feature-123",
		QualityGateStatus: "OK",
		Metrics:           &QualityMetrics{Bugs: 1},
		IssuesSummary: &IssuesSummary{
			Total:      1,
			ByType:     map[string]int{"BUG": 1},
			BySeverity: map[string]int{"MAJOR": 1},
		},
	}

	fw := &failingWriter{}
	err := data.writeText(fw)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "write failed")
}
