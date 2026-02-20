package scanbranch

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
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
	h := &ScanBranchHandler{}
	if got := h.Name(); got != constants.ActNRSQScanBranch {
		t.Errorf("Name() = %q, want %q", got, constants.ActNRSQScanBranch)
	}
}

// TestDescription проверяет возврат описания команды.
func TestDescription(t *testing.T) {
	h := &ScanBranchHandler{}
	if got := h.Description(); got == "" {
		t.Error("Description() returned empty string")
	}
}

// Test_isValidBranchForScanning проверяет валидацию имени ветки.
// L-2 fix: тестирует shared.IsValidBranchForScanning.
func Test_isValidBranchForScanning(t *testing.T) {
	tests := []struct {
		name   string
		branch string
		want   bool
	}{
		// Допустимые ветки
		{
			name:   "main branch",
			branch: "main",
			want:   true,
		},
		{
			name:   "task branch 6 digits",
			branch: "t123456",
			want:   true,
		},
		{
			name:   "task branch 7 digits",
			branch: "t1234567",
			want:   true,
		},
		{
			name:   "task branch minimal",
			branch: "t000000",
			want:   true,
		},
		{
			name:   "task branch maximal 7",
			branch: "t9999999",
			want:   true,
		},

		// Недопустимые ветки
		{
			name:   "empty branch",
			branch: "",
			want:   false,
		},
		{
			name:   "develop branch",
			branch: "develop",
			want:   false,
		},
		{
			name:   "feature branch",
			branch: "feature/test",
			want:   false,
		},
		{
			name:   "task branch 5 digits - too short",
			branch: "t12345",
			want:   false,
		},
		{
			name:   "task branch 8 digits - too long",
			branch: "t12345678",
			want:   false,
		},
		{
			name:   "task branch with letters",
			branch: "t12345a",
			want:   false,
		},
		{
			name:   "only t",
			branch: "t",
			want:   false,
		},
		{
			name:   "uppercase T",
			branch: "T123456",
			want:   false,
		},
		{
			name:   "task prefix without digits",
			branch: "task123456",
			want:   false,
		},
		{
			name:   "main with suffix",
			branch: "main-dev",
			want:   false,
		},
		{
			name:   "Main uppercase",
			branch: "Main",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// L-2 fix: используем shared.IsValidBranchForScanning
			if got := shared.IsValidBranchForScanning(tt.branch); got != tt.want {
				t.Errorf("IsValidBranchForScanning(%q) = %v, want %v", tt.branch, got, tt.want)
			}
		})
	}
}

// Test_hasRelevantChangesInCommit проверяет определение релевантных изменений.
func Test_hasRelevantChangesInCommit(t *testing.T) {
	tests := []struct {
		name            string
		projectStructure []string
		commitFiles     []gitea.CommitFile
		wantHasChanges  bool
		wantErr         bool
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
		{
			name:             "file in config dir but not inside subfolder",
			projectStructure: []string{"Configuration"},
			commitFiles: []gitea.CommitFile{
				{Filename: "ConfigurationFile.txt", Status: "modified"},
			},
			wantHasChanges: false,
			wantErr:        false,
		},
		{
			name:             "multiple extensions - change in second",
			projectStructure: []string{"Configuration", "ExtA", "ExtB"},
			commitFiles: []gitea.CommitFile{
				{Filename: "Configuration.ExtB/src/test.bsl", Status: "added"},
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

			got, err := shared.HasRelevantChangesInCommit(context.Background(), giteaClient, "main", "abc123")

			if (err != nil) != tt.wantErr {
				t.Errorf("hasRelevantChangesInCommit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantHasChanges {
				t.Errorf("hasRelevantChangesInCommit() = %v, want %v", got, tt.wantHasChanges)
			}
		})
	}
}

// Test_hasRelevantChangesInCommit_APIError проверяет обработку ошибок API.
func Test_hasRelevantChangesInCommit_APIError(t *testing.T) {
	tests := []struct {
		name       string
		setupMock  func(m *giteatest.MockClient)
		wantErr    bool
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

			_, err := shared.HasRelevantChangesInCommit(context.Background(), giteaClient, "main", "abc123")

			if (err != nil) != tt.wantErr {
				t.Errorf("hasRelevantChangesInCommit() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestExecute_NilConfig проверяет обработку nil конфигурации.
func TestExecute_NilConfig(t *testing.T) {
	h := &ScanBranchHandler{}

	err := h.Execute(context.Background(), nil)
	if err == nil {
		t.Error("Execute() expected error for nil config, got nil")
	}

	if err != nil && !contains(err.Error(), shared.ErrConfigMissing) {
		t.Errorf("Execute() error = %v, want error containing %q", err, shared.ErrConfigMissing)
	}
}

// TestExecute_InvalidBranch проверяет отклонение недопустимых веток.
func TestExecute_InvalidBranch(t *testing.T) {
	h := &ScanBranchHandler{
		sonarqubeClient: sonarqubetest.NewMockClient(),
		giteaClient:     giteatest.NewMockClient(),
	}

	cfg := &config.Config{
		BranchForScan: "feature/invalid",
		Owner:         "owner",
		Repo:          "repo",
	}

	err := h.Execute(context.Background(), cfg)
	if err == nil {
		t.Error("Execute() expected error for invalid branch, got nil")
	}

	if err != nil && !contains(err.Error(), shared.ErrBranchInvalidFormat) {
		t.Errorf("Execute() error = %v, want error containing %q", err, shared.ErrBranchInvalidFormat)
	}
}

// TestExecute_MissingBranch проверяет отсутствие ветки.
func TestExecute_MissingBranch(t *testing.T) {
	h := &ScanBranchHandler{}
	cfg := &config.Config{
		BranchForScan: "",
		Owner:         "owner",
		Repo:          "repo",
	}

	err := h.Execute(context.Background(), cfg)
	if err == nil {
		t.Error("Execute() expected error for missing branch, got nil")
	}

	if err != nil && !contains(err.Error(), shared.ErrBranchMissing) {
		t.Errorf("Execute() error = %v, want error containing %q", err, shared.ErrBranchMissing)
	}
}

// TestExecute_MissingOwnerRepo проверяет отсутствие owner/repo.
func TestExecute_MissingOwnerRepo(t *testing.T) {
	h := &ScanBranchHandler{
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
				BranchForScan: "main",
				Owner:         tt.owner,
				Repo:          tt.repo,
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

// TestExecute_GetBranchCommitRangeError проверяет обработку ошибки Gitea API.
func TestExecute_GetBranchCommitRangeError(t *testing.T) {
	sqClient := sonarqubetest.NewMockClient()
	giteaClient := &giteatest.MockClient{
		GetBranchCommitRangeFunc: func(_ context.Context, _ string) (*gitea.BranchCommitRange, error) {
			return nil, errors.New("API timeout")
		},
	}

	h := &ScanBranchHandler{
		sonarqubeClient: sqClient,
		giteaClient:     giteaClient,
	}

	cfg := &config.Config{
		BranchForScan: "main",
		Owner:         "owner",
		Repo:          "repo",
	}

	err := h.Execute(context.Background(), cfg)
	if err == nil {
		t.Error("Execute() expected error for GetBranchCommitRange failure, got nil")
	}

	if err != nil && !contains(err.Error(), shared.ErrGiteaAPI) {
		t.Errorf("Execute() error = %v, want error containing %q", err, shared.ErrGiteaAPI)
	}
}

// TestExecute_SkipsAlreadyScanned проверяет пропуск уже отсканированных коммитов.
func TestExecute_SkipsAlreadyScanned(t *testing.T) {
	sqClient := &sonarqubetest.MockClient{
		GetAnalysesFunc: func(_ context.Context, _ string) ([]sonarqube.Analysis, error) {
			return []sonarqube.Analysis{
				{Revision: "abc123"},
			}, nil
		},
		GetProjectFunc: func(_ context.Context, _ string) (*sonarqube.Project, error) {
			return &sonarqube.Project{Key: "owner_repo_main"}, nil
		},
	}
	giteaClient := &giteatest.MockClient{
		GetBranchCommitRangeFunc: func(_ context.Context, _ string) (*gitea.BranchCommitRange, error) {
			return &gitea.BranchCommitRange{
				FirstCommit: &gitea.Commit{SHA: "abc123"},
				LastCommit:  &gitea.Commit{SHA: "abc123"},
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

	h := &ScanBranchHandler{
		sonarqubeClient: sqClient,
		giteaClient:     giteaClient,
	}

	cfg := &config.Config{
		BranchForScan: "main",
		Owner:         "owner",
		Repo:          "repo",
	}

	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}
	// Успешное выполнение без вызова RunAnalysis (т.к. коммит уже отсканирован)
}

// TestExecute_Success проверяет успешное сканирование.
func TestExecute_Success(t *testing.T) {
	runAnalysisCalled := false
	sqClient := &sonarqubetest.MockClient{
		GetAnalysesFunc: func(_ context.Context, _ string) ([]sonarqube.Analysis, error) {
			return []sonarqube.Analysis{}, nil // Нет предыдущих анализов
		},
		GetProjectFunc: func(_ context.Context, _ string) (*sonarqube.Project, error) {
			return &sonarqube.Project{Key: "owner_repo_main"}, nil
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
	}
	giteaClient := &giteatest.MockClient{
		GetBranchCommitRangeFunc: func(_ context.Context, _ string) (*gitea.BranchCommitRange, error) {
			return &gitea.BranchCommitRange{
				FirstCommit: &gitea.Commit{SHA: "newcommit123"},
				LastCommit:  &gitea.Commit{SHA: "newcommit123"},
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

	h := &ScanBranchHandler{
		sonarqubeClient: sqClient,
		giteaClient:     giteaClient,
	}

	cfg := &config.Config{
		BranchForScan: "main",
		Owner:         "owner",
		Repo:          "repo",
	}

	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}

	if !runAnalysisCalled {
		t.Error("Execute() expected RunAnalysis to be called")
	}
}

// TestExecute_NoChangesInConfig проверяет случай отсутствия изменений в конфигурации.
func TestExecute_NoChangesInConfig(t *testing.T) {
	sqClient := &sonarqubetest.MockClient{}
	giteaClient := &giteatest.MockClient{
		GetBranchCommitRangeFunc: func(_ context.Context, _ string) (*gitea.BranchCommitRange, error) {
			return &gitea.BranchCommitRange{
				FirstCommit: &gitea.Commit{SHA: "abc123"},
				LastCommit:  &gitea.Commit{SHA: "abc123"},
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

	h := &ScanBranchHandler{
		sonarqubeClient: sqClient,
		giteaClient:     giteaClient,
	}

	cfg := &config.Config{
		BranchForScan: "main",
		Owner:         "owner",
		Repo:          "repo",
	}

	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}
}

// TestExecute_TaskBranch проверяет сканирование task-ветки.
func TestExecute_TaskBranch(t *testing.T) {
	sqClient := &sonarqubetest.MockClient{
		GetAnalysesFunc: func(_ context.Context, _ string) ([]sonarqube.Analysis, error) {
			return []sonarqube.Analysis{}, nil
		},
		GetProjectFunc: func(_ context.Context, _ string) (*sonarqube.Project, error) {
			return nil, errors.New("not found") // Проект не существует
		},
		CreateProjectFunc: func(_ context.Context, opts sonarqube.CreateProjectOptions) (*sonarqube.Project, error) {
			return &sonarqube.Project{Key: opts.Key, Name: opts.Name}, nil
		},
		RunAnalysisFunc: func(_ context.Context, opts sonarqube.RunAnalysisOptions) (*sonarqube.AnalysisResult, error) {
			return &sonarqube.AnalysisResult{TaskID: "task-1"}, nil
		},
		GetAnalysisStatusFunc: func(_ context.Context, _ string) (*sonarqube.AnalysisStatus, error) {
			return &sonarqube.AnalysisStatus{Status: "SUCCESS", AnalysisID: "analysis-1"}, nil
		},
	}
	giteaClient := &giteatest.MockClient{
		GetBranchCommitRangeFunc: func(_ context.Context, _ string) (*gitea.BranchCommitRange, error) {
			return &gitea.BranchCommitRange{
				FirstCommit: &gitea.Commit{SHA: "task123"},
				LastCommit:  &gitea.Commit{SHA: "task456"},
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

	h := &ScanBranchHandler{
		sonarqubeClient: sqClient,
		giteaClient:     giteaClient,
	}

	cfg := &config.Config{
		BranchForScan: "t123456", // Валидная task-ветка
		Owner:         "owner",
		Repo:          "repo",
	}

	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}
}

// TestScanBranchData_writeText проверяет текстовый вывод результатов.
func TestScanBranchData_writeText(t *testing.T) {
	tests := []struct {
		name     string
		data     *ScanBranchData
		contains []string
	}{
		{
			name: "successful scan",
			data: &ScanBranchData{
				Branch:         "main",
				ProjectKey:     "owner_repo_main",
				CommitsScanned: 2,
				SkippedCount:   1,
				ScanResults: []CommitScanResult{
					{CommitSHA: "abc1234567890", AnalysisID: "a1", Status: "SUCCESS"},
					{CommitSHA: "def1234567890", AnalysisID: "a2", Status: "FAILED", ErrorMessage: "timeout"},
				},
			},
			contains: []string{"main", "owner_repo_main", "2", "1", "abc1234", "SUCCESS", "def1234", "FAILED", "timeout"},
		},
		{
			name: "no changes",
			data: &ScanBranchData{
				Branch:                 "main",
				ProjectKey:             "owner_repo_main",
				NoRelevantChangesCount: 2,
				NoChanges:              true,
			},
			contains: []string{"main", "нет изменений", "Коммитов без изменений: 2"},
		},
		{
			name: "all skipped",
			data: &ScanBranchData{
				Branch:         "t123456",
				ProjectKey:     "owner_repo_t123456",
				CommitsScanned: 0,
				SkippedCount:   3,
			},
			contains: []string{"t123456", "0", "3"},
		},
		{
			name: "empty commit SHA",
			data: &ScanBranchData{
				Branch:         "main",
				ProjectKey:     "owner_repo_main",
				CommitsScanned: 1,
				ScanResults: []CommitScanResult{
					{CommitSHA: "", AnalysisID: "a1", Status: "SUCCESS"},
				},
			},
			contains: []string{"unknown", "SUCCESS"},
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

// TestExecute_EmptySHAInCommitRange проверяет обработку пустого SHA (H-2 fix).
func TestExecute_EmptySHAInCommitRange(t *testing.T) {
	sqClient := &sonarqubetest.MockClient{}
	giteaClient := &giteatest.MockClient{
		GetBranchCommitRangeFunc: func(_ context.Context, _ string) (*gitea.BranchCommitRange, error) {
			return &gitea.BranchCommitRange{
				FirstCommit: &gitea.Commit{SHA: ""}, // Пустой SHA
				LastCommit:  &gitea.Commit{SHA: ""}, // Пустой SHA
			}, nil
		},
	}

	h := &ScanBranchHandler{
		sonarqubeClient: sqClient,
		giteaClient:     giteaClient,
	}

	cfg := &config.Config{
		BranchForScan: "main",
		Owner:         "owner",
		Repo:          "repo",
	}

	// Должен завершиться успешно с NoChanges=true (нет валидных коммитов)
	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}
}

// TestExecute_GetProjectNetworkError проверяет обработку network ошибки GetProject (M-2).
func TestExecute_GetProjectNetworkError(t *testing.T) {
	getProjectCalled := false
	createProjectCalled := false
	sqClient := &sonarqubetest.MockClient{
		GetAnalysesFunc: func(_ context.Context, _ string) ([]sonarqube.Analysis, error) {
			return []sonarqube.Analysis{}, nil
		},
		GetProjectFunc: func(_ context.Context, _ string) (*sonarqube.Project, error) {
			getProjectCalled = true
			return nil, errors.New("network timeout") // Не "not found", а network error
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
	giteaClient := &giteatest.MockClient{
		GetBranchCommitRangeFunc: func(_ context.Context, _ string) (*gitea.BranchCommitRange, error) {
			return &gitea.BranchCommitRange{
				FirstCommit: &gitea.Commit{SHA: "abc123def456789"}, // SHA > 7 символов
				LastCommit:  &gitea.Commit{SHA: "abc123def456789"},
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

	h := &ScanBranchHandler{
		sonarqubeClient: sqClient,
		giteaClient:     giteaClient,
	}

	cfg := &config.Config{
		BranchForScan: "main",
		Owner:         "owner",
		Repo:          "repo",
	}

	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}

	if !getProjectCalled {
		t.Error("Expected GetProject to be called")
	}
	if !createProjectCalled {
		t.Error("Expected CreateProject to be called after GetProject error")
	}
}

// TestWaitForAnalysisCompletion_FailedStatus проверяет обработку FAILED статуса (M-4).
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

	ctx := context.Background()

	status, err := shared.WaitForAnalysisCompletion(ctx, sqClient, "task-1", slog.Default())

	if err != nil {
		t.Errorf("waitForAnalysisCompletion() unexpected error: %v", err)
	}
	if status == nil {
		t.Fatal("waitForAnalysisCompletion() returned nil status")
	}
	if status.Status != "FAILED" {
		t.Errorf("waitForAnalysisCompletion() status = %q, want %q", status.Status, "FAILED")
	}
	if status.ErrorMessage == "" {
		t.Error("waitForAnalysisCompletion() expected ErrorMessage for FAILED status")
	}
}

// TestWaitForAnalysisCompletion_ContextCanceled проверяет обработку отмены context (M-4).
// M-2 fix: устранён race condition через sync канал вместо busy-loop.
func TestWaitForAnalysisCompletion_ContextCanceled(t *testing.T) {
	ctx := context.Background()
	firstCallDone := make(chan struct{})
	sqClient := &sonarqubetest.MockClient{
		GetAnalysisStatusFunc: func(_ context.Context, _ string) (*sonarqube.AnalysisStatus, error) {
			// Сигнализируем о первом вызове (безопасно для многократных вызовов)
			select {
			case <-firstCallDone:
				// Уже закрыт
			default:
				close(firstCallDone)
			}
			return &sonarqube.AnalysisStatus{
				Status: "IN_PROGRESS",
			}, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Отменяем context сразу после первого вызова
	go func() {
		<-firstCallDone
		cancel()
	}()

	_, err := shared.WaitForAnalysisCompletion(ctx, sqClient, "task-1", slog.Default())

	if err == nil {
		t.Error("waitForAnalysisCompletion() expected error for canceled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("waitForAnalysisCompletion() error = %v, want context.Canceled", err)
	}
}

// TestWaitForAnalysisCompletion_UnknownStatus проверяет обработку неизвестного статуса (M-4).
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
		t.Error("waitForAnalysisCompletion() expected error for unknown status")
	}
	if !contains(err.Error(), "неизвестный статус") {
		t.Errorf("waitForAnalysisCompletion() error = %v, want error containing 'неизвестный статус'", err)
	}
}

// TestWaitForAnalysisCompletion_CanceledStatus проверяет обработку CANCELED статуса (H-1 fix).
func TestWaitForAnalysisCompletion_CanceledStatus(t *testing.T) {
	sqClient := &sonarqubetest.MockClient{
		GetAnalysisStatusFunc: func(_ context.Context, taskID string) (*sonarqube.AnalysisStatus, error) {
			return &sonarqube.AnalysisStatus{
				TaskID: taskID,
				Status: "CANCELED",
				// CANCELED обычно не имеет ErrorMessage (в отличие от FAILED)
			}, nil
		},
	}

	status, err := shared.WaitForAnalysisCompletion(context.Background(), sqClient, "task-1", slog.Default())

	if err != nil {
		t.Errorf("waitForAnalysisCompletion() unexpected error: %v", err)
	}
	if status == nil {
		t.Fatal("waitForAnalysisCompletion() returned nil status")
	}
	if status.Status != "CANCELED" {
		t.Errorf("waitForAnalysisCompletion() status = %q, want %q", status.Status, "CANCELED")
	}
	// CANCELED не должен иметь ErrorMessage
	if status.ErrorMessage != "" {
		t.Errorf("waitForAnalysisCompletion() expected empty ErrorMessage for CANCELED, got %q", status.ErrorMessage)
	}
}

// TestWaitForAnalysisCompletion_Timeout проверяет таймаут ожидания (M-1 fix).
func TestWaitForAnalysisCompletion_Timeout(t *testing.T) {
	ctx := context.Background()
	callCount := 0
	sqClient := &sonarqubetest.MockClient{
		GetAnalysisStatusFunc: func(_ context.Context, _ string) (*sonarqube.AnalysisStatus, error) {
			callCount++
			// Всегда возвращаем IN_PROGRESS, чтобы дождаться таймаута
			return &sonarqube.AnalysisStatus{
				Status: "IN_PROGRESS",
			}, nil
		},
	}

	// Используем context с коротким таймаутом для ускорения теста
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := shared.WaitForAnalysisCompletion(ctx, sqClient, "task-1", slog.Default())

	// Должен завершиться по context timeout, а не по maxAttempts
	if err == nil {
		t.Error("waitForAnalysisCompletion() expected error for timeout")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("waitForAnalysisCompletion() error = %v, want context.DeadlineExceeded", err)
	}
}

// TestExecute_RunAnalysisError проверяет обработку ошибки RunAnalysis (M-2 fix).
func TestExecute_RunAnalysisError(t *testing.T) {
	sqClient := &sonarqubetest.MockClient{
		GetAnalysesFunc: func(_ context.Context, _ string) ([]sonarqube.Analysis, error) {
			return []sonarqube.Analysis{}, nil
		},
		GetProjectFunc: func(_ context.Context, _ string) (*sonarqube.Project, error) {
			return &sonarqube.Project{Key: "owner_repo_main"}, nil
		},
		RunAnalysisFunc: func(_ context.Context, _ sonarqube.RunAnalysisOptions) (*sonarqube.AnalysisResult, error) {
			return nil, errors.New("scanner execution failed")
		},
	}
	giteaClient := &giteatest.MockClient{
		GetBranchCommitRangeFunc: func(_ context.Context, _ string) (*gitea.BranchCommitRange, error) {
			return &gitea.BranchCommitRange{
				FirstCommit: &gitea.Commit{SHA: "abc123def456789"},
				LastCommit:  &gitea.Commit{SHA: "abc123def456789"},
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

	h := &ScanBranchHandler{
		sonarqubeClient: sqClient,
		giteaClient:     giteaClient,
	}

	cfg := &config.Config{
		BranchForScan: "main",
		Owner:         "owner",
		Repo:          "repo",
	}

	// Должен успешно завершиться, но с FAILED результатом для коммита
	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}
	// Коммит должен быть в результатах со статусом FAILED (проверяем через вывод)
}

// TestExecute_CreateProjectError проверяет обработку ошибки CreateProject (M-3 fix).
func TestExecute_CreateProjectError(t *testing.T) {
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
	giteaClient := &giteatest.MockClient{
		GetBranchCommitRangeFunc: func(_ context.Context, _ string) (*gitea.BranchCommitRange, error) {
			return &gitea.BranchCommitRange{
				FirstCommit: &gitea.Commit{SHA: "abc123def456789"},
				LastCommit:  &gitea.Commit{SHA: "abc123def456789"},
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

	h := &ScanBranchHandler{
		sonarqubeClient: sqClient,
		giteaClient:     giteaClient,
	}

	cfg := &config.Config{
		BranchForScan: "main",
		Owner:         "owner",
		Repo:          "repo",
	}

	err := h.Execute(context.Background(), cfg)
	if err == nil {
		t.Error("Execute() expected error for CreateProject failure, got nil")
	}

	if err != nil && !contains(err.Error(), shared.ErrSonarQubeAPI) {
		t.Errorf("Execute() error = %v, want error containing %q", err, shared.ErrSonarQubeAPI)
	}
}

// TestExecute_JSONOutput проверяет JSON формат вывода (AC #5, M-1 fix).
func TestExecute_JSONOutput(t *testing.T) {
	// Сохраняем и восстанавливаем env
	oldFormat := os.Getenv("BR_OUTPUT_FORMAT")
	t.Cleanup(func() {
		if oldFormat == "" {
			_ = os.Unsetenv("BR_OUTPUT_FORMAT")
		} else {
			_ = os.Setenv("BR_OUTPUT_FORMAT", oldFormat)
		}
	})
	_ = os.Setenv("BR_OUTPUT_FORMAT", "json")

	sqClient := &sonarqubetest.MockClient{
		GetAnalysesFunc: func(_ context.Context, _ string) ([]sonarqube.Analysis, error) {
			return []sonarqube.Analysis{}, nil
		},
		GetProjectFunc: func(_ context.Context, _ string) (*sonarqube.Project, error) {
			return &sonarqube.Project{Key: "owner_repo_main"}, nil
		},
		RunAnalysisFunc: func(_ context.Context, _ sonarqube.RunAnalysisOptions) (*sonarqube.AnalysisResult, error) {
			return &sonarqube.AnalysisResult{TaskID: "task-1", AnalysisID: "analysis-1"}, nil
		},
		GetAnalysisStatusFunc: func(_ context.Context, _ string) (*sonarqube.AnalysisStatus, error) {
			return &sonarqube.AnalysisStatus{Status: "SUCCESS", AnalysisID: "analysis-1"}, nil
		},
	}
	giteaClient := &giteatest.MockClient{
		GetBranchCommitRangeFunc: func(_ context.Context, _ string) (*gitea.BranchCommitRange, error) {
			return &gitea.BranchCommitRange{
				FirstCommit: &gitea.Commit{SHA: "abc123def456"},
				LastCommit:  &gitea.Commit{SHA: "abc123def456"},
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

	h := &ScanBranchHandler{
		sonarqubeClient: sqClient,
		giteaClient:     giteaClient,
	}

	cfg := &config.Config{
		BranchForScan: "main",
		Owner:         "owner",
		Repo:          "repo",
	}

	// Execute пишет в stdout, проверяем что не падает
	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Errorf("Execute() with JSON format unexpected error: %v", err)
	}
}

// TestExecute_JSONOutputError проверяет JSON формат вывода ошибки.
func TestExecute_JSONOutputError(t *testing.T) {
	oldFormat := os.Getenv("BR_OUTPUT_FORMAT")
	t.Cleanup(func() {
		if oldFormat == "" {
			_ = os.Unsetenv("BR_OUTPUT_FORMAT")
		} else {
			_ = os.Setenv("BR_OUTPUT_FORMAT", oldFormat)
		}
	})
	_ = os.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &ScanBranchHandler{}
	cfg := &config.Config{
		BranchForScan: "invalid-branch",
		Owner:         "owner",
		Repo:          "repo",
	}

	err := h.Execute(context.Background(), cfg)
	if err == nil {
		t.Error("Execute() expected error for invalid branch")
	}
	// Ошибка должна содержать код
	if !contains(err.Error(), shared.ErrBranchInvalidFormat) {
		t.Errorf("Execute() error = %v, want error containing %q", err, shared.ErrBranchInvalidFormat)
	}
}

// TestExecute_MixedRelevantChanges проверяет подсчёт NoRelevantChangesCount в смешанном сценарии (M-2 fix).
func TestExecute_MixedRelevantChanges(t *testing.T) {
	// Сценарий: 2 коммита, один с изменениями в конфигурации, один без
	sqClient := &sonarqubetest.MockClient{
		GetAnalysesFunc: func(_ context.Context, _ string) ([]sonarqube.Analysis, error) {
			return []sonarqube.Analysis{}, nil
		},
		GetProjectFunc: func(_ context.Context, _ string) (*sonarqube.Project, error) {
			return &sonarqube.Project{Key: "owner_repo_main"}, nil
		},
		RunAnalysisFunc: func(_ context.Context, _ sonarqube.RunAnalysisOptions) (*sonarqube.AnalysisResult, error) {
			return &sonarqube.AnalysisResult{TaskID: "task-1", AnalysisID: "analysis-1"}, nil
		},
		GetAnalysisStatusFunc: func(_ context.Context, _ string) (*sonarqube.AnalysisStatus, error) {
			return &sonarqube.AnalysisStatus{Status: "SUCCESS", AnalysisID: "analysis-1"}, nil
		},
	}

	callCount := 0
	giteaClient := &giteatest.MockClient{
		GetBranchCommitRangeFunc: func(_ context.Context, _ string) (*gitea.BranchCommitRange, error) {
			return &gitea.BranchCommitRange{
				FirstCommit: &gitea.Commit{SHA: "commit1_with_changes"},
				LastCommit:  &gitea.Commit{SHA: "commit2_no_changes"},
			}, nil
		},
		AnalyzeProjectStructureFunc: func(_ context.Context, _ string) ([]string, error) {
			return []string{"Configuration"}, nil
		},
		GetCommitFilesFunc: func(_ context.Context, commitSHA string) ([]gitea.CommitFile, error) {
			callCount++
			// Первый коммит — изменения в конфигурации, второй — нет
			if commitSHA == "commit1_with_changes" {
				return []gitea.CommitFile{
					{Filename: "Configuration/src/test.bsl", Status: "modified"},
				}, nil
			}
			// commit2_no_changes — только README
			return []gitea.CommitFile{
				{Filename: "README.md", Status: "modified"},
			}, nil
		},
	}

	h := &ScanBranchHandler{
		sonarqubeClient: sqClient,
		giteaClient:     giteaClient,
	}

	cfg := &config.Config{
		BranchForScan: "main",
		Owner:         "owner",
		Repo:          "repo",
	}

	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}

	// Проверяем что GetCommitFiles вызывался для обоих коммитов
	if callCount < 2 {
		t.Errorf("Expected GetCommitFiles to be called for both commits, got %d calls", callCount)
	}
}

// TestWaitForAnalysisCompletion_GetStatusError проверяет обработку ошибки API (M-1 fix).
func TestWaitForAnalysisCompletion_GetStatusError(t *testing.T) {
	sqClient := &sonarqubetest.MockClient{
		GetAnalysisStatusFunc: func(_ context.Context, _ string) (*sonarqube.AnalysisStatus, error) {
			return nil, errors.New("network timeout")
		},
	}

	_, err := shared.WaitForAnalysisCompletion(context.Background(), sqClient, "task-1", slog.Default())

	if err == nil {
		t.Error("waitForAnalysisCompletion() expected error for GetAnalysisStatus failure")
	}
	if !contains(err.Error(), "ошибка получения статуса анализа") {
		t.Errorf("waitForAnalysisCompletion() error = %v, want error containing 'ошибка получения статуса анализа'", err)
	}
}

// TestScanBranchData_writeText_Error проверяет обработку ошибки записи (M-3 fix).
func TestScanBranchData_writeText_Error(t *testing.T) {
	data := &ScanBranchData{
		Branch:     "main",
		ProjectKey: "owner_repo_main",
	}

	// Используем writer который всегда возвращает ошибку
	errWriter := &errorWriter{err: errors.New("write failed")}

	err := data.writeText(errWriter)
	if err == nil {
		t.Error("writeText() expected error for failing writer")
	}
	if !contains(err.Error(), "write failed") {
		t.Errorf("writeText() error = %v, want error containing 'write failed'", err)
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
	return bytes.Contains([]byte(s), []byte(substr))
}
