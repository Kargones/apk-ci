package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/entity/gitea"
)

// MockAPI реализует gitea.APIInterface для тестирования
type MockAPI struct {
	activePRs       []gitea.PR
	activePRError   error
	createTestError error
	createPRError   error
	mergePRError    error
	closePRError    error
	deleteTestError error
	conflictPRError error
	conflictResult  bool
}

func (m *MockAPI) ActivePR() ([]gitea.PR, error) {
	return m.activePRs, m.activePRError
}

func (m *MockAPI) CreateTestBranch() error {
	return m.createTestError
}

func (m *MockAPI) DeleteTestBranch() error {
	return m.deleteTestError
}

func (m *MockAPI) CreatePR(head string) (gitea.PR, error) {
	if m.createPRError != nil {
		return gitea.PR{}, m.createPRError
	}
	return gitea.PR{
		Number: 123,
		Head:   head,
		Base:   "test-branch",
	}, nil
}

func (m *MockAPI) MergePR(number int64, l *slog.Logger) error {
	return m.mergePRError
}

func (m *MockAPI) ClosePR(number int64) error {
	return m.closePRError
}

func (m *MockAPI) AddIssueComment(issueNumber int64, commentText string) error {
	return nil
}

// Добавляем остальные методы интерфейса для полной совместимости
func (m *MockAPI) GetIssue(issueNumber int64) (*gitea.Issue, error) {
	return nil, nil
}

func (m *MockAPI) GetFileContent(fileName string) ([]byte, error) {
	return nil, nil
}

func (m *MockAPI) GetConfigData(l *slog.Logger, filename string) ([]byte, error) {
	return nil, nil
}

func (m *MockAPI) CloseIssue(issueNumber int64) error {
	return nil
}

func (m *MockAPI) ConflictPR(prNumber int64) (bool, error) {
	return m.conflictResult, m.conflictPRError
}

func (m *MockAPI) ConflictFilesPR(prNumber int64) ([]string, error) {
	return nil, nil
}

func (m *MockAPI) GetRepositoryContents(filepath, branch string) ([]gitea.FileInfo, error) {
	return nil, nil
}

func (m *MockAPI) AnalyzeProjectStructure(branch string) ([]string, error) {
	return nil, nil
}

func (m *MockAPI) AnalyzeProject(branch string) ([]string, error) {
	return nil, nil
}

func (m *MockAPI) GetLatestCommit(branch string) (*gitea.Commit, error) {
	return nil, nil
}

func (m *MockAPI) GetCommitFiles(commitSHA string) ([]gitea.CommitFile, error) {
	return nil, nil
}

func (m *MockAPI) IsUserInTeam(l *slog.Logger, username string, orgName string, teamName string) (bool, error) {
	return false, nil
}

func (m *MockAPI) GetCommits(branch string, limit int) ([]gitea.Commit, error) {
	return nil, nil
}

func (m *MockAPI) GetFirstCommitOfBranch(branch string, baseBranch string) (*gitea.Commit, error) {
	return nil, nil
}

func (m *MockAPI) GetCommitsBetween(baseCommitSHA, headCommitSHA string) ([]gitea.Commit, error) {
	return nil, nil
}

func (m *MockAPI) GetBranchCommitRange(branch string) (*gitea.BranchCommitRange, error) {
	return nil, nil
}

func (m *MockAPI) SetRepositoryState(l *slog.Logger, operations []gitea.BatchOperation, branch, commitMessage string) error {
	return nil
}

func (m *MockAPI) GetTeamMembers(orgName, teamName string) ([]string, error) {
	return nil, nil
}

func (m *MockAPI) GetBranches(repo string) ([]gitea.Branch, error) {
	return nil, nil
}

func (m *MockAPI) GetLatestRelease() (*gitea.Release, error) {
	return nil, nil
}

func (m *MockAPI) GetReleaseByTag(tag string) (*gitea.Release, error) {
	return nil, nil
}

func (m *MockAPI) SearchOrgRepos(orgName string) ([]gitea.Repository, error) {
	return nil, nil
}

func (m *MockAPI) CreatePRWithOptions(opts gitea.CreatePROptions) (*gitea.PRResponse, error) {
	return nil, nil
}

// MockProjectAnalyzer реализует gitea.ProjectAnalyzer для тестирования
type MockProjectAnalyzer struct {
	analyzeError error
}

func (m *MockProjectAnalyzer) AnalyzeProject(l *slog.Logger, branch string) error {
	return m.analyzeError
}

func TestNewGiteaService(t *testing.T) {
	api := &MockAPI{}
	cfg := &config.Config{
		Owner:      "testowner",
		Repo:       "testrepo",
		BaseBranch: "main",
	}
	analyzer := &MockProjectAnalyzer{}

	service := NewGiteaService(api, cfg, analyzer)

	if service == nil {
		t.Fatal("Expected non-nil GiteaService")
	}
	if service.api == nil {
		t.Error("Expected API to be set correctly")
	}
	if service.config != cfg {
		t.Error("Expected config to be set correctly")
	}
	if service.projectAnalyzer != analyzer {
		t.Error("Expected projectAnalyzer to be set correctly")
	}
}

func TestGiteaService_GetAPI(t *testing.T) {
	api := &MockAPI{}
	cfg := &config.Config{}
	analyzer := &MockProjectAnalyzer{}

	service := NewGiteaService(api, cfg, analyzer)
	result := service.GetAPI()

	if result == nil {
		t.Error("Expected GetAPI to return the correct API")
	}
}

func TestGiteaService_GetConfig(t *testing.T) {
	api := &MockAPI{}
	cfg := &config.Config{
		Owner: "testowner",
		Repo:  "testrepo",
	}
	analyzer := &MockProjectAnalyzer{}

	service := NewGiteaService(api, cfg, analyzer)
	result := service.GetConfig()

	if result != cfg {
		t.Error("Expected GetConfig to return the correct config")
	}
}

func TestGiteaService_AnalyzeProject(t *testing.T) {
	api := &MockAPI{}
	cfg := &config.Config{}
	analyzer := &MockProjectAnalyzer{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	service := NewGiteaService(api, cfg, analyzer)

	// Тест успешного анализа
	err := service.AnalyzeProject(ctx, logger, "test-branch")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Тест с ошибкой анализа
	analyzer.analyzeError = errors.New("analyze error")
	err = service.AnalyzeProject(ctx, logger, "test-branch")
	if err == nil {
		t.Error("Expected error from analyzer")
	}
}

func TestGiteaService_TestMerge(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()
	cfg := &config.Config{BaseBranch: "main"}
	analyzer := &MockProjectAnalyzer{}

	// Тест с ошибкой получения активных PR
	t.Run("ActivePR error", func(t *testing.T) {
		api := &MockAPI{
			activePRError: errors.New("API error"),
		}
		service := NewGiteaService(api, cfg, analyzer)

		err := service.TestMerge(ctx, logger)
		if err == nil {
			t.Error("Expected error from ActivePR")
		}
	})

	// Тест с ошибкой создания тестовой ветки
	t.Run("CreateTestBranch error", func(t *testing.T) {
		api := &MockAPI{
			activePRs:       []gitea.PR{},
			createTestError: errors.New("create test branch error"),
		}
		service := NewGiteaService(api, cfg, analyzer)

		err := service.TestMerge(ctx, logger)
		if err == nil {
			t.Error("Expected error from CreateTestBranch")
		}
	})

	// Тест успешного выполнения без активных PR
	t.Run("No active PRs", func(t *testing.T) {
		api := &MockAPI{
			activePRs: []gitea.PR{},
		}
		service := NewGiteaService(api, cfg, analyzer)

		err := service.TestMerge(ctx, logger)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	// Тест с активными PR
	t.Run("With active PRs", func(t *testing.T) {
		api := &MockAPI{
			activePRs: []gitea.PR{
				{Number: 1, Head: "feature-1", Base: "main"},
				{Number: 2, Head: "feature-2", Base: "main"},
			},
		}
		service := NewGiteaService(api, cfg, analyzer)

		err := service.TestMerge(ctx, logger)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	// Тест с ошибкой создания PR
	t.Run("CreatePR error", func(t *testing.T) {
		api := &MockAPI{
			activePRs: []gitea.PR{
				{Number: 1, Head: "feature-1", Base: "main"},
			},
			createPRError: errors.New("create PR error"),
		}
		service := NewGiteaService(api, cfg, analyzer)

		err := service.TestMerge(ctx, logger)
		if err == nil {
			t.Error("Expected error from CreatePR")
		}
	})

	// Тест с ошибкой слияния PR
	t.Run("MergePR error", func(t *testing.T) {
		api := &MockAPI{
			activePRs: []gitea.PR{
				{Number: 1, Head: "feature-1", Base: "main"},
			},
			mergePRError: errors.New("merge PR error"),
		}
		service := NewGiteaService(api, cfg, analyzer)

		err := service.TestMerge(ctx, logger)
		if err != nil {
			t.Errorf("TestMerge should not fail on merge error, got: %v", err)
		}
	})

	// Тест с ошибкой удаления тестовой ветки
	t.Run("DeleteTestBranch error", func(t *testing.T) {
		api := &MockAPI{
			activePRs:       []gitea.PR{},
			deleteTestError: errors.New("delete test branch error"),
		}
		service := NewGiteaService(api, cfg, analyzer)

		err := service.TestMerge(ctx, logger)
		if err == nil {
			t.Error("Expected error from DeleteTestBranch")
		}
	})

	// Тест с ошибкой закрытия PR
	t.Run("ClosePR error", func(t *testing.T) {
		api := &MockAPI{
			activePRs: []gitea.PR{
				{Number: 1, Head: "feature-1", Base: "main"},
			},
			mergePRError: errors.New("merge error"),
			closePRError: errors.New("close PR error"),
		}
		service := NewGiteaService(api, cfg, analyzer)

		err := service.TestMerge(ctx, logger)
		if err != nil {
			t.Errorf("TestMerge should handle close PR error gracefully, got: %v", err)
		}
	})

	// Тест с конфликтами в PR
	t.Run("PR with conflicts", func(t *testing.T) {
		api := &MockAPI{
			activePRs: []gitea.PR{
				{Number: 1, Head: "feature-1", Base: "main"},
			},
			conflictResult: true,
		}
		service := NewGiteaService(api, cfg, analyzer)

		err := service.TestMerge(ctx, logger)
		if err != nil {
			t.Errorf("TestMerge should handle conflicts gracefully, got: %v", err)
		}
	})

	// Тест с ошибкой проверки конфликтов
	t.Run("ConflictPR error", func(t *testing.T) {
		api := &MockAPI{
			activePRs: []gitea.PR{
				{Number: 1, Head: "feature-1", Base: "main"},
			},
			conflictPRError: errors.New("conflict check error"),
		}
		service := NewGiteaService(api, cfg, analyzer)

		err := service.TestMerge(ctx, logger)
		if err != nil {
			t.Errorf("TestMerge should handle conflict check error gracefully, got: %v", err)
		}
	})
}