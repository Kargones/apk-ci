package scanpr

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/gitea"
	"github.com/Kargones/apk-ci/internal/adapter/gitea/giteatest"
	"github.com/Kargones/apk-ci/internal/adapter/sonarqube"
	"github.com/Kargones/apk-ci/internal/adapter/sonarqube/sonarqubetest"
	"github.com/Kargones/apk-ci/internal/command/handlers/sonarqube/shared"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
)

// TestName проверяет возврат имени команды.
func TestName(t *testing.T) {
	h := &ScanPRHandler{}
	if got := h.Name(); got != constants.ActNRSQScanPR {
		t.Errorf("Name() = %q, want %q", got, constants.ActNRSQScanPR)
	}
}

// TestDescription проверяет возврат описания команды.
func TestDescription(t *testing.T) {
	h := &ScanPRHandler{}
	if got := h.Description(); got == "" {
		t.Error("Description() returned empty string")
	}
}

// TestExecute_NilConfig проверяет обработку nil конфигурации.
func TestExecute_NilConfig(t *testing.T) {
	h := &ScanPRHandler{}

	err := h.Execute(context.Background(), nil)
	if err == nil {
		t.Error("Execute() expected error for nil config, got nil")
	}

	if err != nil && !contains(err.Error(), shared.ErrConfigMissing) {
		t.Errorf("Execute() error = %v, want error containing %q", err, shared.ErrConfigMissing)
	}
}

// TestExecute_MissingPRNumber проверяет отсутствие номера PR (AC: #2).
func TestExecute_MissingPRNumber(t *testing.T) {
	h := &ScanPRHandler{
		sonarqubeClient: sonarqubetest.NewMockClient(),
		giteaClient:     giteatest.NewMockClient(),
	}
	cfg := &config.Config{
		PRNumber: 0,
		Owner:    "owner",
		Repo:     "repo",
	}

	err := h.Execute(context.Background(), cfg)
	if err == nil {
		t.Error("Execute() expected error for missing PR number, got nil")
	}

	if err != nil && !contains(err.Error(), shared.ErrPRMissing) {
		t.Errorf("Execute() error = %v, want error containing %q", err, shared.ErrPRMissing)
	}
}

// TestExecute_InvalidPRNumber проверяет некорректный номер PR (AC: #2).
func TestExecute_InvalidPRNumber(t *testing.T) {
	h := &ScanPRHandler{
		sonarqubeClient: sonarqubetest.NewMockClient(),
		giteaClient:     giteatest.NewMockClient(),
	}
	cfg := &config.Config{
		PRNumber: -1,
		Owner:    "owner",
		Repo:     "repo",
	}

	err := h.Execute(context.Background(), cfg)
	if err == nil {
		t.Error("Execute() expected error for invalid PR number, got nil")
	}

	if err != nil && !contains(err.Error(), shared.ErrPRInvalid) {
		t.Errorf("Execute() error = %v, want error containing %q", err, shared.ErrPRInvalid)
	}
}

// TestExecute_MissingOwnerRepo проверяет отсутствие owner/repo.
func TestExecute_MissingOwnerRepo(t *testing.T) {
	h := &ScanPRHandler{
		sonarqubeClient: sonarqubetest.NewMockClient(),
		giteaClient:     giteatest.NewMockClient(),
	}

	tests := []struct {
		name  string
		owner string
		repo  string
	}{
		{"missing owner", "", "repo"},
		{"missing repo", "owner", ""},
		{"missing both", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				PRNumber: 123,
				Owner:    tt.owner,
				Repo:     tt.repo,
			}

			err := h.Execute(context.Background(), cfg)
			if err == nil {
				t.Error("Execute() expected error for missing owner/repo, got nil")
			}

			if err != nil && !contains(err.Error(), shared.ErrConfigMissing) {
				t.Errorf("Execute() error = %v, want error containing %q", err, shared.ErrConfigMissing)
			}
		})
	}
}

// TestExecute_NilGiteaClient проверяет обработку nil Gitea клиента.
func TestExecute_NilGiteaClient(t *testing.T) {
	h := &ScanPRHandler{
		sonarqubeClient: sonarqubetest.NewMockClient(),
		giteaClient:     nil, // nil клиент
	}

	cfg := &config.Config{
		PRNumber: 123,
		Owner:    "owner",
		Repo:     "repo",
	}

	err := h.Execute(context.Background(), cfg)
	if err == nil {
		t.Error("Execute() expected error for nil Gitea client, got nil")
	}

	if err != nil && !contains(err.Error(), shared.ErrConfigMissing) {
		t.Errorf("Execute() error = %v, want error containing %q", err, shared.ErrConfigMissing)
	}
}

// TestExecute_NilSonarQubeClient проверяет обработку nil SonarQube клиента.
func TestExecute_NilSonarQubeClient(t *testing.T) {
	h := &ScanPRHandler{
		sonarqubeClient: nil, // nil клиент
		giteaClient:     giteatest.NewMockClient(),
	}

	cfg := &config.Config{
		PRNumber: 123,
		Owner:    "owner",
		Repo:     "repo",
	}

	err := h.Execute(context.Background(), cfg)
	if err == nil {
		t.Error("Execute() expected error for nil SonarQube client, got nil")
	}

	if err != nil && !contains(err.Error(), shared.ErrConfigMissing) {
		t.Errorf("Execute() error = %v, want error containing %q", err, shared.ErrConfigMissing)
	}
}

// TestExecute_PRNotFound проверяет случай когда PR не найден (AC: #3).
func TestExecute_PRNotFound(t *testing.T) {
	giteaClient := &giteatest.MockClient{
		GetPRFunc: func(_ context.Context, prNumber int64) (*gitea.PRResponse, error) {
			return nil, errors.New("PR #123 not found")
		},
	}

	h := &ScanPRHandler{
		sonarqubeClient: sonarqubetest.NewMockClient(),
		giteaClient:     giteaClient,
	}

	cfg := &config.Config{
		PRNumber: 123,
		Owner:    "owner",
		Repo:     "repo",
	}

	err := h.Execute(context.Background(), cfg)
	if err == nil {
		t.Error("Execute() expected error for PR not found, got nil")
	}

	if err != nil && !contains(err.Error(), shared.ErrPRNotFound) {
		t.Errorf("Execute() error = %v, want error containing %q", err, shared.ErrPRNotFound)
	}
}

// TestExecute_PRClosed проверяет случай когда PR закрыт (AC: #3).
//nolint:dupl // similar test structure
func TestExecute_PRClosed(t *testing.T) {
	giteaClient := &giteatest.MockClient{
		GetPRFunc: func(_ context.Context, prNumber int64) (*gitea.PRResponse, error) {
			return &gitea.PRResponse{
				Number: prNumber,
				State:  "closed",
				Title:  "Test PR",
				Head:   gitea.Branch{Name: "feature", Commit: gitea.BranchCommit{ID: "abc123"}},
				Base:   gitea.Branch{Name: "main"},
			}, nil
		},
	}

	h := &ScanPRHandler{
		sonarqubeClient: sonarqubetest.NewMockClient(),
		giteaClient:     giteaClient,
	}

	cfg := &config.Config{
		PRNumber: 123,
		Owner:    "owner",
		Repo:     "repo",
	}

	err := h.Execute(context.Background(), cfg)
	if err == nil {
		t.Error("Execute() expected error for closed PR, got nil")
	}

	if err != nil && !contains(err.Error(), shared.ErrPRNotOpen) {
		t.Errorf("Execute() error = %v, want error containing %q", err, shared.ErrPRNotOpen)
	}
}

// TestExecute_PRMerged проверяет случай когда PR уже merged.
//nolint:dupl // similar test structure
func TestExecute_PRMerged(t *testing.T) {
	giteaClient := &giteatest.MockClient{
		GetPRFunc: func(_ context.Context, prNumber int64) (*gitea.PRResponse, error) {
			return &gitea.PRResponse{
				Number: prNumber,
				State:  "merged",
				Title:  "Test PR",
				Head:   gitea.Branch{Name: "feature", Commit: gitea.BranchCommit{ID: "abc123"}},
				Base:   gitea.Branch{Name: "main"},
			}, nil
		},
	}

	h := &ScanPRHandler{
		sonarqubeClient: sonarqubetest.NewMockClient(),
		giteaClient:     giteaClient,
	}

	cfg := &config.Config{
		PRNumber: 123,
		Owner:    "owner",
		Repo:     "repo",
	}

	err := h.Execute(context.Background(), cfg)
	if err == nil {
		t.Error("Execute() expected error for merged PR, got nil")
	}

	if err != nil && !contains(err.Error(), shared.ErrPRNotOpen) {
		t.Errorf("Execute() error = %v, want error containing %q", err, shared.ErrPRNotOpen)
	}
}

// TestExecute_AlreadyScanned проверяет пропуск уже отсканированных коммитов (AC: #5).
func TestExecute_AlreadyScanned(t *testing.T) {
	commitSHA := "abc123def456789"
	giteaClient := &giteatest.MockClient{
		GetPRFunc: func(_ context.Context, prNumber int64) (*gitea.PRResponse, error) {
			return &gitea.PRResponse{
				Number: prNumber,
				State:  "open",
				Title:  "Test PR",
				Head:   gitea.Branch{Name: "feature", Commit: gitea.BranchCommit{ID: commitSHA}},
				Base:   gitea.Branch{Name: "main"},
			}, nil
		},
		AnalyzeProjectStructureFunc: func(_ context.Context, _ string) ([]string, error) {
			return []string{"Configuration"}, nil
		},
		GetCommitFilesFunc: func(_ context.Context, _ string) ([]gitea.CommitFile, error) {
			return []gitea.CommitFile{
				{Filename: "Configuration/src/test.bsl", Status: "modified"},
			}, nil
		},
	}

	sqClient := &sonarqubetest.MockClient{
		GetAnalysesFunc: func(_ context.Context, _ string) ([]sonarqube.Analysis, error) {
			return []sonarqube.Analysis{
				{Revision: commitSHA}, // Этот коммит уже отсканирован
			}, nil
		},
	}

	h := &ScanPRHandler{
		sonarqubeClient: sqClient,
		giteaClient:     giteaClient,
	}

	cfg := &config.Config{
		PRNumber: 123,
		Owner:    "owner",
		Repo:     "repo",
	}

	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}
	// Успешное выполнение без вызова RunAnalysis (т.к. коммит уже отсканирован)
}

// TestExecute_NoRelevantChanges проверяет случай без изменений в конфигурации.
func TestExecute_NoRelevantChanges(t *testing.T) {
	giteaClient := &giteatest.MockClient{
		GetPRFunc: func(_ context.Context, prNumber int64) (*gitea.PRResponse, error) {
			return &gitea.PRResponse{
				Number: prNumber,
				State:  "open",
				Title:  "Test PR",
				Head:   gitea.Branch{Name: "feature", Commit: gitea.BranchCommit{ID: "abc123"}},
				Base:   gitea.Branch{Name: "main"},
			}, nil
		},
		AnalyzeProjectStructureFunc: func(_ context.Context, _ string) ([]string, error) {
			return []string{"Configuration"}, nil
		},
		GetCommitFilesFunc: func(_ context.Context, _ string) ([]gitea.CommitFile, error) {
			return []gitea.CommitFile{
				{Filename: "README.md", Status: "modified"},
				{Filename: "docs/guide.md", Status: "added"},
			}, nil
		},
	}

	sqClient := &sonarqubetest.MockClient{}

	h := &ScanPRHandler{
		sonarqubeClient: sqClient,
		giteaClient:     giteaClient,
	}

	cfg := &config.Config{
		PRNumber: 123,
		Owner:    "owner",
		Repo:     "repo",
	}

	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}
}

// TestExecute_Success проверяет успешное сканирование PR (AC: #5, #6).
func TestExecute_Success(t *testing.T) {
	runAnalysisCalled := false
	getQualityGateCalled := false
	commitSHA := "newcommit123456789"

	giteaClient := &giteatest.MockClient{
		GetPRFunc: func(_ context.Context, prNumber int64) (*gitea.PRResponse, error) {
			return &gitea.PRResponse{
				Number: prNumber,
				State:  "open",
				Title:  "Feature: Add new functionality",
				Head:   gitea.Branch{Name: "feature-123", Commit: gitea.BranchCommit{ID: commitSHA}},
				Base:   gitea.Branch{Name: "main"},
			}, nil
		},
		AnalyzeProjectStructureFunc: func(_ context.Context, _ string) ([]string, error) {
			return []string{"Configuration"}, nil
		},
		GetCommitFilesFunc: func(_ context.Context, _ string) ([]gitea.CommitFile, error) {
			return []gitea.CommitFile{
				{Filename: "Configuration/src/CommonModules/Module.bsl", Status: "modified"},
			}, nil
		},
	}

	sqClient := &sonarqubetest.MockClient{
		GetAnalysesFunc: func(_ context.Context, _ string) ([]sonarqube.Analysis, error) {
			return []sonarqube.Analysis{}, nil // Нет предыдущих анализов
		},
		GetProjectFunc: func(_ context.Context, _ string) (*sonarqube.Project, error) {
			return &sonarqube.Project{Key: "owner_repo_feature-123"}, nil
		},
		RunAnalysisFunc: func(_ context.Context, opts sonarqube.RunAnalysisOptions) (*sonarqube.AnalysisResult, error) {
			runAnalysisCalled = true
			return &sonarqube.AnalysisResult{
				TaskID:     "task-1",
				ProjectKey: opts.ProjectKey,
				AnalysisID: "analysis-1",
			}, nil
		},
		GetAnalysisStatusFunc: func(_ context.Context, _ string) (*sonarqube.AnalysisStatus, error) {
			return &sonarqube.AnalysisStatus{
				TaskID:     "task-1",
				Status:     "SUCCESS",
				AnalysisID: "analysis-1",
			}, nil
		},
		GetQualityGateStatusFunc: func(_ context.Context, _ string) (*sonarqube.QualityGateStatus, error) {
			getQualityGateCalled = true
			return &sonarqube.QualityGateStatus{
				Status: "OK",
			}, nil
		},
	}

	h := &ScanPRHandler{
		sonarqubeClient: sqClient,
		giteaClient:     giteaClient,
	}

	cfg := &config.Config{
		PRNumber: 123,
		Owner:    "owner",
		Repo:     "repo",
	}

	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}

	if !runAnalysisCalled {
		t.Error("Execute() expected RunAnalysis to be called")
	}

	if !getQualityGateCalled {
		t.Error("Execute() expected GetQualityGateStatus to be called for successful analysis")
	}
}

// TestExecute_CreateProjectIfNotExists проверяет создание проекта если он не существует.
func TestExecute_CreateProjectIfNotExists(t *testing.T) {
	createProjectCalled := false

	giteaClient := &giteatest.MockClient{
		GetPRFunc: func(_ context.Context, prNumber int64) (*gitea.PRResponse, error) {
			return &gitea.PRResponse{
				Number: prNumber,
				State:  "open",
				Title:  "Test PR",
				Head:   gitea.Branch{Name: "feature", Commit: gitea.BranchCommit{ID: "abc123def456"}},
				Base:   gitea.Branch{Name: "main"},
			}, nil
		},
		AnalyzeProjectStructureFunc: func(_ context.Context, _ string) ([]string, error) {
			return []string{"Configuration"}, nil
		},
		GetCommitFilesFunc: func(_ context.Context, _ string) ([]gitea.CommitFile, error) {
			return []gitea.CommitFile{
				{Filename: "Configuration/src/test.bsl", Status: "modified"},
			}, nil
		},
	}

	sqClient := &sonarqubetest.MockClient{
		GetAnalysesFunc: func(_ context.Context, _ string) ([]sonarqube.Analysis, error) {
			return []sonarqube.Analysis{}, nil
		},
		GetProjectFunc: func(_ context.Context, _ string) (*sonarqube.Project, error) {
			return nil, errors.New("not found")
		},
		CreateProjectFunc: func(_ context.Context, opts sonarqube.CreateProjectOptions) (*sonarqube.Project, error) {
			createProjectCalled = true
			return &sonarqube.Project{Key: opts.Key}, nil
		},
		RunAnalysisFunc: func(_ context.Context, _ sonarqube.RunAnalysisOptions) (*sonarqube.AnalysisResult, error) {
			return &sonarqube.AnalysisResult{TaskID: "task-1"}, nil
		},
		GetAnalysisStatusFunc: func(_ context.Context, _ string) (*sonarqube.AnalysisStatus, error) {
			return &sonarqube.AnalysisStatus{Status: "SUCCESS", AnalysisID: "a1"}, nil
		},
	}

	h := &ScanPRHandler{
		sonarqubeClient: sqClient,
		giteaClient:     giteaClient,
	}

	cfg := &config.Config{
		PRNumber: 123,
		Owner:    "owner",
		Repo:     "repo",
	}

	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}

	if !createProjectCalled {
		t.Error("Expected CreateProject to be called when project does not exist")
	}
}

// TestExecute_CreateProjectError проверяет обработку ошибки создания проекта.
func TestExecute_CreateProjectError(t *testing.T) {
	giteaClient := &giteatest.MockClient{
		GetPRFunc: func(_ context.Context, prNumber int64) (*gitea.PRResponse, error) {
			return &gitea.PRResponse{
				Number: prNumber,
				State:  "open",
				Title:  "Test PR",
				Head:   gitea.Branch{Name: "feature", Commit: gitea.BranchCommit{ID: "abc123def456"}},
				Base:   gitea.Branch{Name: "main"},
			}, nil
		},
		AnalyzeProjectStructureFunc: func(_ context.Context, _ string) ([]string, error) {
			return []string{"Configuration"}, nil
		},
		GetCommitFilesFunc: func(_ context.Context, _ string) ([]gitea.CommitFile, error) {
			return []gitea.CommitFile{
				{Filename: "Configuration/src/test.bsl", Status: "modified"},
			}, nil
		},
	}

	sqClient := &sonarqubetest.MockClient{
		GetAnalysesFunc: func(_ context.Context, _ string) ([]sonarqube.Analysis, error) {
			return []sonarqube.Analysis{}, nil
		},
		GetProjectFunc: func(_ context.Context, _ string) (*sonarqube.Project, error) {
			return nil, errors.New("not found")
		},
		CreateProjectFunc: func(_ context.Context, _ sonarqube.CreateProjectOptions) (*sonarqube.Project, error) {
			return nil, errors.New("SonarQube API unavailable")
		},
	}

	h := &ScanPRHandler{
		sonarqubeClient: sqClient,
		giteaClient:     giteaClient,
	}

	cfg := &config.Config{
		PRNumber: 123,
		Owner:    "owner",
		Repo:     "repo",
	}

	err := h.Execute(context.Background(), cfg)
	if err == nil {
		t.Error("Execute() expected error for CreateProject failure, got nil")
	}

	if err != nil && !contains(err.Error(), shared.ErrSonarQubeAPI) {
		t.Errorf("Execute() error = %v, want error containing %q", err, shared.ErrSonarQubeAPI)
	}
}

// TestExecute_RunAnalysisError проверяет обработку ошибки RunAnalysis.
func TestExecute_RunAnalysisError(t *testing.T) {
	giteaClient := &giteatest.MockClient{
		GetPRFunc: func(_ context.Context, prNumber int64) (*gitea.PRResponse, error) {
			return &gitea.PRResponse{
				Number: prNumber,
				State:  "open",
				Title:  "Test PR",
				Head:   gitea.Branch{Name: "feature", Commit: gitea.BranchCommit{ID: "abc123def456"}},
				Base:   gitea.Branch{Name: "main"},
			}, nil
		},
		AnalyzeProjectStructureFunc: func(_ context.Context, _ string) ([]string, error) {
			return []string{"Configuration"}, nil
		},
		GetCommitFilesFunc: func(_ context.Context, _ string) ([]gitea.CommitFile, error) {
			return []gitea.CommitFile{
				{Filename: "Configuration/src/test.bsl", Status: "modified"},
			}, nil
		},
	}

	sqClient := &sonarqubetest.MockClient{
		GetAnalysesFunc: func(_ context.Context, _ string) ([]sonarqube.Analysis, error) {
			return []sonarqube.Analysis{}, nil
		},
		GetProjectFunc: func(_ context.Context, _ string) (*sonarqube.Project, error) {
			return &sonarqube.Project{Key: "owner_repo_feature"}, nil
		},
		RunAnalysisFunc: func(_ context.Context, _ sonarqube.RunAnalysisOptions) (*sonarqube.AnalysisResult, error) {
			return nil, errors.New("scanner execution failed")
		},
	}

	h := &ScanPRHandler{
		sonarqubeClient: sqClient,
		giteaClient:     giteaClient,
	}

	cfg := &config.Config{
		PRNumber: 123,
		Owner:    "owner",
		Repo:     "repo",
	}

	// Должен успешно завершиться, но с FAILED результатом
	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}
}

// TestExecute_JSONOutput проверяет JSON формат вывода (AC: #6).
func TestExecute_JSONOutput(t *testing.T) {
	oldFormat := os.Getenv("BR_OUTPUT_FORMAT")
	t.Cleanup(func() {
		if oldFormat == "" {
			_ = os.Unsetenv("BR_OUTPUT_FORMAT")
		} else {
			_ = os.Setenv("BR_OUTPUT_FORMAT", oldFormat)
		}
	})
	_ = os.Setenv("BR_OUTPUT_FORMAT", "json")

	giteaClient := &giteatest.MockClient{
		GetPRFunc: func(_ context.Context, prNumber int64) (*gitea.PRResponse, error) {
			return &gitea.PRResponse{
				Number: prNumber,
				State:  "open",
				Title:  "Test PR",
				Head:   gitea.Branch{Name: "feature", Commit: gitea.BranchCommit{ID: "abc123def456"}},
				Base:   gitea.Branch{Name: "main"},
			}, nil
		},
		AnalyzeProjectStructureFunc: func(_ context.Context, _ string) ([]string, error) {
			return []string{"Configuration"}, nil
		},
		GetCommitFilesFunc: func(_ context.Context, _ string) ([]gitea.CommitFile, error) {
			return []gitea.CommitFile{
				{Filename: "Configuration/src/test.bsl", Status: "modified"},
			}, nil
		},
	}

	sqClient := &sonarqubetest.MockClient{
		GetAnalysesFunc: func(_ context.Context, _ string) ([]sonarqube.Analysis, error) {
			return []sonarqube.Analysis{}, nil
		},
		GetProjectFunc: func(_ context.Context, _ string) (*sonarqube.Project, error) {
			return &sonarqube.Project{Key: "owner_repo_feature"}, nil
		},
		RunAnalysisFunc: func(_ context.Context, _ sonarqube.RunAnalysisOptions) (*sonarqube.AnalysisResult, error) {
			return &sonarqube.AnalysisResult{TaskID: "task-1", AnalysisID: "analysis-1"}, nil
		},
		GetAnalysisStatusFunc: func(_ context.Context, _ string) (*sonarqube.AnalysisStatus, error) {
			return &sonarqube.AnalysisStatus{Status: "SUCCESS", AnalysisID: "analysis-1"}, nil
		},
	}

	h := &ScanPRHandler{
		sonarqubeClient: sqClient,
		giteaClient:     giteaClient,
	}

	cfg := &config.Config{
		PRNumber: 123,
		Owner:    "owner",
		Repo:     "repo",
	}

	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Errorf("Execute() with JSON format unexpected error: %v", err)
	}
}

// TestScanPRData_writeText проверяет текстовый вывод результатов (AC: #7).
func TestScanPRData_writeText(t *testing.T) {
	tests := []struct {
		name     string
		data     *ScanPRData
		contains []string
	}{
		{
			name: "successful scan with quality gate",
			data: &ScanPRData{
				PRNumber:   123,
				PRTitle:    "Feature: Add new functionality",
				HeadBranch: "feature-123",
				BaseBranch: "main",
				ProjectKey: "owner_repo_feature-123",
				CommitSHA:  "abc1234567890",
				Scanned:    true,
				ScanResult: &ScanResult{
					AnalysisID:        "a1",
					Status:            "SUCCESS",
					QualityGateStatus: "OK",
				},
			},
			contains: []string{"PR #123", "Feature", "feature-123", "main", "abc1234", "SUCCESS", "Quality Gate", "OK"},
		},
		{
			name: "already scanned",
			data: &ScanPRData{
				PRNumber:       123,
				PRTitle:        "Test PR",
				HeadBranch:     "feature",
				BaseBranch:     "main",
				ProjectKey:     "owner_repo_feature",
				CommitSHA:      "abc123",
				AlreadyScanned: true,
			},
			contains: []string{"PR #123", "уже отсканирован"},
		},
		{
			name: "no relevant changes",
			data: &ScanPRData{
				PRNumber:          123,
				PRTitle:           "Docs update",
				HeadBranch:        "docs",
				BaseBranch:        "main",
				ProjectKey:        "owner_repo_docs",
				CommitSHA:         "def456",
				NoRelevantChanges: true,
			},
			contains: []string{"PR #123", "нет изменений"},
		},
		{
			name: "failed scan with error",
			data: &ScanPRData{
				PRNumber:   123,
				PRTitle:    "Test PR",
				HeadBranch: "feature",
				BaseBranch: "main",
				ProjectKey: "owner_repo_feature",
				CommitSHA:  "abc123",
				Scanned:    true,
				ScanResult: &ScanResult{
					Status:       "FAILED",
					ErrorMessage: "Scanner timeout",
				},
			},
			contains: []string{"FAILED", "Scanner timeout"},
		},
		{
			name: "empty commit SHA",
			data: &ScanPRData{
				PRNumber:   123,
				PRTitle:    "Test PR",
				HeadBranch: "feature",
				BaseBranch: "main",
				ProjectKey: "owner_repo_feature",
				CommitSHA:  "",
				Scanned:    false,
			},
			contains: []string{"unknown"},
		},
		{
			name: "quality gate failed",
			data: &ScanPRData{
				PRNumber:   123,
				PRTitle:    "Test PR",
				HeadBranch: "feature",
				BaseBranch: "main",
				ProjectKey: "owner_repo_feature",
				CommitSHA:  "abc123456789",
				Scanned:    true,
				ScanResult: &ScanResult{
					AnalysisID:        "a1",
					Status:            "SUCCESS",
					QualityGateStatus: "ERROR",
				},
			},
			contains: []string{"SUCCESS", "Quality Gate", "ERROR"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tt.data.writeText(&buf)
			if err != nil {
				t.Errorf("writeText() error = %v", err)
				return
			}

			output := buf.String()
			for _, s := range tt.contains {
				if !contains(output, s) {
					t.Errorf("writeText() output missing %q, got:\n%s", s, output)
				}
			}
		})
	}
}

// TestScanPRData_writeText_Error проверяет обработку ошибки записи.
func TestScanPRData_writeText_Error(t *testing.T) {
	data := &ScanPRData{
		PRNumber:   123,
		PRTitle:    "Test PR",
		HeadBranch: "feature",
		BaseBranch: "main",
		ProjectKey: "owner_repo_feature",
		CommitSHA:  "abc123",
	}

	errWriter := &errorWriter{err: errors.New("write failed")}

	err := data.writeText(errWriter)
	if err == nil {
		t.Error("writeText() expected error for failing writer")
	}
	if !contains(err.Error(), "write failed") {
		t.Errorf("writeText() error = %v, want error containing 'write failed'", err)
	}
}

// Test_hasRelevantChangesInCommit проверяет определение релевантных изменений.
// Тесты вынесены в shared пакет, здесь проверяем интеграцию.
func Test_hasRelevantChangesInCommit(t *testing.T) {
	tests := []struct {
		name             string
		projectStructure []string
		commitFiles      []gitea.CommitFile
		wantHasChanges   bool
		wantErr          bool
	}{
		{
			name:             "changes in main config",
			projectStructure: []string{"Configuration"},
			commitFiles: []gitea.CommitFile{
				{Filename: "Configuration/src/CommonModules/module.bsl", Status: "modified"},
			},
			wantHasChanges: true,
			wantErr:        false,
		},
		{
			name:             "changes in extension",
			projectStructure: []string{"Configuration", "ExtA"},
			commitFiles: []gitea.CommitFile{
				{Filename: "Configuration.ExtA/src/CommonModules/module.bsl", Status: "added"},
			},
			wantHasChanges: true,
			wantErr:        false,
		},
		{
			name:             "no relevant changes",
			projectStructure: []string{"Configuration"},
			commitFiles: []gitea.CommitFile{
				{Filename: "README.md", Status: "modified"},
				{Filename: "docs/readme.txt", Status: "modified"},
			},
			wantHasChanges: false,
			wantErr:        false,
		},
		{
			name:             "empty project structure - any change is relevant",
			projectStructure: []string{},
			commitFiles: []gitea.CommitFile{
				{Filename: "any/file.txt", Status: "modified"},
			},
			wantHasChanges: true,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			giteaClient := &giteatest.MockClient{
				AnalyzeProjectStructureFunc: func(_ context.Context, _ string) ([]string, error) {
					return tt.projectStructure, nil
				},
				GetCommitFilesFunc: func(_ context.Context, _ string) ([]gitea.CommitFile, error) {
					return tt.commitFiles, nil
				},
			}

			got, err := shared.HasRelevantChangesInCommit(context.Background(), giteaClient, "feature", "abc123")

			if (err != nil) != tt.wantErr {
				t.Errorf("HasRelevantChangesInCommit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantHasChanges {
				t.Errorf("HasRelevantChangesInCommit() = %v, want %v", got, tt.wantHasChanges)
			}
		})
	}
}

// Test_hasRelevantChangesInCommit_APIError проверяет обработку ошибок API.
func Test_hasRelevantChangesInCommit_APIError(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(m *giteatest.MockClient)
		wantErr   bool
	}{
		{
			name: "error in AnalyzeProjectStructure",
			setupMock: func(m *giteatest.MockClient) {
				m.AnalyzeProjectStructureFunc = func(_ context.Context, _ string) ([]string, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr: true,
		},
		{
			name: "error in GetCommitFiles",
			setupMock: func(m *giteatest.MockClient) {
				m.AnalyzeProjectStructureFunc = func(_ context.Context, _ string) ([]string, error) {
					return []string{"Configuration"}, nil
				}
				m.GetCommitFilesFunc = func(_ context.Context, _ string) ([]gitea.CommitFile, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			giteaClient := &giteatest.MockClient{}
			tt.setupMock(giteaClient)

			_, err := shared.HasRelevantChangesInCommit(context.Background(), giteaClient, "feature", "abc123")

			if (err != nil) != tt.wantErr {
				t.Errorf("HasRelevantChangesInCommit() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestWaitForAnalysisCompletion_FailedStatus проверяет обработку FAILED статуса.
func TestWaitForAnalysisCompletion_FailedStatus(t *testing.T) {
	sqClient := &sonarqubetest.MockClient{
		GetAnalysisStatusFunc: func(_ context.Context, taskID string) (*sonarqube.AnalysisStatus, error) {
			return &sonarqube.AnalysisStatus{
				TaskID:       taskID,
				Status:       "FAILED",
				ErrorMessage: "Analysis failed due to invalid code",
			}, nil
		},
	}

	status, err := shared.WaitForAnalysisCompletion(context.Background(), sqClient, "task-1", slog.Default())

	if err != nil {
		t.Errorf("WaitForAnalysisCompletion() unexpected error: %v", err)
	}
	if status == nil {
		t.Fatal("WaitForAnalysisCompletion() returned nil status")
	}
	if status.Status != "FAILED" {
		t.Errorf("WaitForAnalysisCompletion() status = %q, want %q", status.Status, "FAILED")
	}
}

// TestWaitForAnalysisCompletion_ContextCanceled проверяет обработку отмены context.
func TestWaitForAnalysisCompletion_ContextCanceled(t *testing.T) {
	ctx := context.Background()
	firstCallDone := make(chan struct{})
	sqClient := &sonarqubetest.MockClient{
		GetAnalysisStatusFunc: func(_ context.Context, _ string) (*sonarqube.AnalysisStatus, error) {
			select {
			case <-firstCallDone:
			default:
				close(firstCallDone)
			}
			return &sonarqube.AnalysisStatus{
				Status: "IN_PROGRESS",
			}, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-firstCallDone
		cancel()
	}()

	_, err := shared.WaitForAnalysisCompletion(ctx, sqClient, "task-1", slog.Default())

	if err == nil {
		t.Error("WaitForAnalysisCompletion() expected error for canceled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("WaitForAnalysisCompletion() error = %v, want context.Canceled", err)
	}
}

// TestWaitForAnalysisCompletion_UnknownStatus проверяет обработку неизвестного статуса.
func TestWaitForAnalysisCompletion_UnknownStatus(t *testing.T) {
	sqClient := &sonarqubetest.MockClient{
		GetAnalysisStatusFunc: func(_ context.Context, _ string) (*sonarqube.AnalysisStatus, error) {
			return &sonarqube.AnalysisStatus{
				Status: "UNKNOWN_STATUS",
			}, nil
		},
	}

	_, err := shared.WaitForAnalysisCompletion(context.Background(), sqClient, "task-1", slog.Default())

	if err == nil {
		t.Error("WaitForAnalysisCompletion() expected error for unknown status")
	}
	if !contains(err.Error(), "неизвестный статус") {
		t.Errorf("WaitForAnalysisCompletion() error = %v, want error containing 'неизвестный статус'", err)
	}
}

// TestWaitForAnalysisCompletion_Timeout проверяет таймаут ожидания.
func TestWaitForAnalysisCompletion_Timeout(t *testing.T) {
	ctx := context.Background()
	sqClient := &sonarqubetest.MockClient{
		GetAnalysisStatusFunc: func(_ context.Context, _ string) (*sonarqube.AnalysisStatus, error) {
			return &sonarqube.AnalysisStatus{
				Status: "IN_PROGRESS",
			}, nil
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := shared.WaitForAnalysisCompletion(ctx, sqClient, "task-1", slog.Default())

	if err == nil {
		t.Error("WaitForAnalysisCompletion() expected error for timeout")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("WaitForAnalysisCompletion() error = %v, want context.DeadlineExceeded", err)
	}
}

// TestWaitForAnalysisCompletion_GetStatusError проверяет обработку ошибки API.
func TestWaitForAnalysisCompletion_GetStatusError(t *testing.T) {
	sqClient := &sonarqubetest.MockClient{
		GetAnalysisStatusFunc: func(_ context.Context, _ string) (*sonarqube.AnalysisStatus, error) {
			return nil, errors.New("network timeout")
		},
	}

	_, err := shared.WaitForAnalysisCompletion(context.Background(), sqClient, "task-1", slog.Default())

	if err == nil {
		t.Error("WaitForAnalysisCompletion() expected error for GetAnalysisStatus failure")
	}
	if !contains(err.Error(), "ошибка получения статуса анализа") {
		t.Errorf("WaitForAnalysisCompletion() error = %v, want error containing 'ошибка получения статуса анализа'", err)
	}
}

// TestExecute_GiteaAPIError проверяет обработку ошибки Gitea API.
func TestExecute_GiteaAPIError(t *testing.T) {
	giteaClient := &giteatest.MockClient{
		GetPRFunc: func(_ context.Context, _ int64) (*gitea.PRResponse, error) {
			return nil, errors.New("connection refused")
		},
	}

	h := &ScanPRHandler{
		sonarqubeClient: sonarqubetest.NewMockClient(),
		giteaClient:     giteaClient,
	}

	cfg := &config.Config{
		PRNumber: 123,
		Owner:    "owner",
		Repo:     "repo",
	}

	err := h.Execute(context.Background(), cfg)
	if err == nil {
		t.Error("Execute() expected error for Gitea API failure")
	}

	if err != nil && !contains(err.Error(), shared.ErrGiteaAPI) {
		t.Errorf("Execute() error = %v, want error containing %q", err, shared.ErrGiteaAPI)
	}
}

// errorWriter — io.Writer который всегда возвращает ошибку.
type errorWriter struct {
	err error
}

func (w *errorWriter) Write(p []byte) (n int, err error) {
	return 0, w.err
}

// contains проверяет наличие подстроки.
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
