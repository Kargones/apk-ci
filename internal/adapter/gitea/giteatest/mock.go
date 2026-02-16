// Package giteatest предоставляет тестовые утилиты для пакета gitea:
// мок-реализации интерфейсов и вспомогательные конструкторы.
package giteatest

import (
	"context"

	"github.com/Kargones/apk-ci/internal/adapter/gitea"
)

// Compile-time проверки реализации интерфейсов
var (
	_ gitea.Client           = (*MockClient)(nil)
	_ gitea.PRReader         = (*MockClient)(nil)
	_ gitea.CommitReader     = (*MockClient)(nil)
	_ gitea.FileReader       = (*MockClient)(nil)
	_ gitea.BranchManager    = (*MockClient)(nil)
	_ gitea.ReleaseReader    = (*MockClient)(nil)
	_ gitea.IssueManager     = (*MockClient)(nil)
	_ gitea.PRManager        = (*MockClient)(nil)
	_ gitea.RepositoryWriter = (*MockClient)(nil)
	_ gitea.TeamReader       = (*MockClient)(nil)
	_ gitea.OrgReader        = (*MockClient)(nil)
)

// MockClient — мок-реализация gitea.Client для тестирования.
// Использует функциональные поля для гибкой настройки поведения в тестах.
type MockClient struct {
	// PRReader
	GetPRFunc           func(ctx context.Context, prNumber int64) (*gitea.PRResponse, error)
	ListOpenPRsFunc     func(ctx context.Context) ([]gitea.PR, error)
	ConflictPRFunc      func(ctx context.Context, prNumber int64) (bool, error)
	ConflictFilesPRFunc func(ctx context.Context, prNumber int64) ([]string, error)

	// CommitReader
	GetCommitsFunc            func(ctx context.Context, branch string, limit int) ([]gitea.Commit, error)
	GetLatestCommitFunc       func(ctx context.Context, branch string) (*gitea.Commit, error)
	GetCommitFilesFunc        func(ctx context.Context, commitSHA string) ([]gitea.CommitFile, error)
	GetCommitsBetweenFunc     func(ctx context.Context, baseCommitSHA, headCommitSHA string) ([]gitea.Commit, error)
	GetFirstCommitOfBranchFunc func(ctx context.Context, branch, baseBranch string) (*gitea.Commit, error)
	GetBranchCommitRangeFunc  func(ctx context.Context, branch string) (*gitea.BranchCommitRange, error)

	// FileReader
	GetFileContentFunc          func(ctx context.Context, fileName string) ([]byte, error)
	GetRepositoryContentsFunc   func(ctx context.Context, filepath, branch string) ([]gitea.FileInfo, error)
	AnalyzeProjectStructureFunc func(ctx context.Context, branch string) ([]string, error)

	// BranchManager
	GetBranchesFunc   func(ctx context.Context, repo string) ([]gitea.Branch, error)
	CreateBranchFunc  func(ctx context.Context, newBranch, baseBranch string) error
	DeleteBranchFunc  func(ctx context.Context, branchName string) error

	// ReleaseReader
	GetLatestReleaseFunc func(ctx context.Context) (*gitea.Release, error)
	GetReleaseByTagFunc  func(ctx context.Context, tag string) (*gitea.Release, error)

	// IssueManager
	GetIssueFunc        func(ctx context.Context, issueNumber int64) (*gitea.Issue, error)
	AddIssueCommentFunc func(ctx context.Context, issueNumber int64, commentText string) error
	CloseIssueFunc      func(ctx context.Context, issueNumber int64) error

	// PRManager
	CreatePRFunc            func(ctx context.Context, head string) (gitea.PR, error)
	CreatePRWithOptionsFunc func(ctx context.Context, opts gitea.CreatePROptions) (*gitea.PRResponse, error)
	MergePRFunc             func(ctx context.Context, prNumber int64) error
	ClosePRFunc             func(ctx context.Context, prNumber int64) error

	// RepositoryWriter
	SetRepositoryStateFunc func(ctx context.Context, operations []gitea.BatchOperation, branch, commitMessage string) error

	// TeamReader
	IsUserInTeamFunc   func(ctx context.Context, username, orgName, teamName string) (bool, error)
	GetTeamMembersFunc func(ctx context.Context, orgName, teamName string) ([]string, error)

	// OrgReader
	SearchOrgReposFunc func(ctx context.Context, orgName string) ([]gitea.Repository, error)
}

// -------------------------------------------------------------------
// PRReader implementation
// -------------------------------------------------------------------

// GetPR возвращает информацию о Pull Request.
// При отсутствии пользовательской функции возвращает тестовые данные.
func (m *MockClient) GetPR(ctx context.Context, prNumber int64) (*gitea.PRResponse, error) {
	if m.GetPRFunc != nil {
		return m.GetPRFunc(ctx, prNumber)
	}
	return &gitea.PRResponse{
		ID:        prNumber,
		Number:    prNumber,
		State:     "open",
		Title:     "Тестовый PR",
		Mergeable: true,
	}, nil
}

// ListOpenPRs возвращает список открытых Pull Requests.
// При отсутствии пользовательской функции возвращает пустой срез.
func (m *MockClient) ListOpenPRs(ctx context.Context) ([]gitea.PR, error) {
	if m.ListOpenPRsFunc != nil {
		return m.ListOpenPRsFunc(ctx)
	}
	return []gitea.PR{}, nil
}

// ConflictPR проверяет наличие конфликтов.
// При отсутствии пользовательской функции возвращает false.
func (m *MockClient) ConflictPR(ctx context.Context, prNumber int64) (bool, error) {
	if m.ConflictPRFunc != nil {
		return m.ConflictPRFunc(ctx, prNumber)
	}
	return false, nil
}

// ConflictFilesPR возвращает список файлов с конфликтами.
// При отсутствии пользовательской функции возвращает пустой срез.
func (m *MockClient) ConflictFilesPR(ctx context.Context, prNumber int64) ([]string, error) {
	if m.ConflictFilesPRFunc != nil {
		return m.ConflictFilesPRFunc(ctx, prNumber)
	}
	return []string{}, nil
}

// -------------------------------------------------------------------
// CommitReader implementation
// -------------------------------------------------------------------

// GetCommits возвращает список коммитов.
// При отсутствии пользовательской функции возвращает пустой срез.
func (m *MockClient) GetCommits(ctx context.Context, branch string, limit int) ([]gitea.Commit, error) {
	if m.GetCommitsFunc != nil {
		return m.GetCommitsFunc(ctx, branch, limit)
	}
	return []gitea.Commit{}, nil
}

// GetLatestCommit возвращает последний коммит.
// При отсутствии пользовательской функции возвращает тестовые данные.
func (m *MockClient) GetLatestCommit(ctx context.Context, branch string) (*gitea.Commit, error) {
	if m.GetLatestCommitFunc != nil {
		return m.GetLatestCommitFunc(ctx, branch)
	}
	return &gitea.Commit{
		SHA: "abc123def456",
		Commit: gitea.CommitDetails{
			Author: gitea.CommitAuthor{
				Name:  "Test Author",
				Email: "test@example.com",
				Date:  "2026-02-04T10:00:00Z",
			},
			Message: "Тестовый коммит",
		},
	}, nil
}

// GetCommitFiles возвращает файлы коммита.
// При отсутствии пользовательской функции возвращает пустой срез.
func (m *MockClient) GetCommitFiles(ctx context.Context, commitSHA string) ([]gitea.CommitFile, error) {
	if m.GetCommitFilesFunc != nil {
		return m.GetCommitFilesFunc(ctx, commitSHA)
	}
	return []gitea.CommitFile{}, nil
}

// GetCommitsBetween возвращает коммиты между двумя SHA.
// При отсутствии пользовательской функции возвращает пустой срез.
func (m *MockClient) GetCommitsBetween(ctx context.Context, baseCommitSHA, headCommitSHA string) ([]gitea.Commit, error) {
	if m.GetCommitsBetweenFunc != nil {
		return m.GetCommitsBetweenFunc(ctx, baseCommitSHA, headCommitSHA)
	}
	return []gitea.Commit{}, nil
}

// GetFirstCommitOfBranch возвращает первый коммит ветки.
// При отсутствии пользовательской функции возвращает тестовые данные.
func (m *MockClient) GetFirstCommitOfBranch(ctx context.Context, branch, baseBranch string) (*gitea.Commit, error) {
	if m.GetFirstCommitOfBranchFunc != nil {
		return m.GetFirstCommitOfBranchFunc(ctx, branch, baseBranch)
	}
	return &gitea.Commit{
		SHA: "first123",
		Commit: gitea.CommitDetails{
			Message: "Первый коммит ветки",
		},
	}, nil
}

// GetBranchCommitRange возвращает диапазон коммитов ветки.
// При отсутствии пользовательской функции возвращает тестовые данные.
func (m *MockClient) GetBranchCommitRange(ctx context.Context, branch string) (*gitea.BranchCommitRange, error) {
	if m.GetBranchCommitRangeFunc != nil {
		return m.GetBranchCommitRangeFunc(ctx, branch)
	}
	return &gitea.BranchCommitRange{
		FirstCommit: &gitea.Commit{SHA: "first123"},
		LastCommit:  &gitea.Commit{SHA: "last456"},
	}, nil
}

// -------------------------------------------------------------------
// FileReader implementation
// -------------------------------------------------------------------

// GetFileContent возвращает содержимое файла.
// При отсутствии пользовательской функции возвращает пустой срез байт.
func (m *MockClient) GetFileContent(ctx context.Context, fileName string) ([]byte, error) {
	if m.GetFileContentFunc != nil {
		return m.GetFileContentFunc(ctx, fileName)
	}
	return []byte{}, nil
}

// GetRepositoryContents возвращает содержимое каталога.
// При отсутствии пользовательской функции возвращает пустой срез.
func (m *MockClient) GetRepositoryContents(ctx context.Context, filepath, branch string) ([]gitea.FileInfo, error) {
	if m.GetRepositoryContentsFunc != nil {
		return m.GetRepositoryContentsFunc(ctx, filepath, branch)
	}
	return []gitea.FileInfo{}, nil
}

// AnalyzeProjectStructure анализирует структуру проекта.
// При отсутствии пользовательской функции возвращает пустой срез.
func (m *MockClient) AnalyzeProjectStructure(ctx context.Context, branch string) ([]string, error) {
	if m.AnalyzeProjectStructureFunc != nil {
		return m.AnalyzeProjectStructureFunc(ctx, branch)
	}
	return []string{}, nil
}

// -------------------------------------------------------------------
// BranchManager implementation
// -------------------------------------------------------------------

// GetBranches возвращает список веток.
// При отсутствии пользовательской функции возвращает пустой срез.
func (m *MockClient) GetBranches(ctx context.Context, repo string) ([]gitea.Branch, error) {
	if m.GetBranchesFunc != nil {
		return m.GetBranchesFunc(ctx, repo)
	}
	return []gitea.Branch{}, nil
}

// CreateBranch создаёт новую ветку на основе базовой.
// При отсутствии пользовательской функции возвращает nil.
func (m *MockClient) CreateBranch(ctx context.Context, newBranch, baseBranch string) error {
	if m.CreateBranchFunc != nil {
		return m.CreateBranchFunc(ctx, newBranch, baseBranch)
	}
	return nil
}

// DeleteBranch удаляет ветку по имени.
// При отсутствии пользовательской функции возвращает nil.
func (m *MockClient) DeleteBranch(ctx context.Context, branchName string) error {
	if m.DeleteBranchFunc != nil {
		return m.DeleteBranchFunc(ctx, branchName)
	}
	return nil
}

// -------------------------------------------------------------------
// ReleaseReader implementation
// -------------------------------------------------------------------

// GetLatestRelease возвращает последний релиз.
// При отсутствии пользовательской функции возвращает тестовые данные.
func (m *MockClient) GetLatestRelease(ctx context.Context) (*gitea.Release, error) {
	if m.GetLatestReleaseFunc != nil {
		return m.GetLatestReleaseFunc(ctx)
	}
	return &gitea.Release{
		ID:      1,
		TagName: "v1.0.0",
		Name:    "Release 1.0.0",
		Body:    "Тестовый релиз",
	}, nil
}

// GetReleaseByTag возвращает релиз по тегу.
// При отсутствии пользовательской функции возвращает тестовые данные.
func (m *MockClient) GetReleaseByTag(ctx context.Context, tag string) (*gitea.Release, error) {
	if m.GetReleaseByTagFunc != nil {
		return m.GetReleaseByTagFunc(ctx, tag)
	}
	return &gitea.Release{
		ID:      1,
		TagName: tag,
		Name:    "Release " + tag,
	}, nil
}

// -------------------------------------------------------------------
// IssueManager implementation
// -------------------------------------------------------------------

// GetIssue возвращает информацию о задаче.
// При отсутствии пользовательской функции возвращает тестовые данные.
func (m *MockClient) GetIssue(ctx context.Context, issueNumber int64) (*gitea.Issue, error) {
	if m.GetIssueFunc != nil {
		return m.GetIssueFunc(ctx, issueNumber)
	}
	return &gitea.Issue{
		ID:     issueNumber,
		Number: issueNumber,
		Title:  "Тестовая задача",
		State:  "open",
	}, nil
}

// AddIssueComment добавляет комментарий к задаче.
// При отсутствии пользовательской функции возвращает nil.
func (m *MockClient) AddIssueComment(ctx context.Context, issueNumber int64, commentText string) error {
	if m.AddIssueCommentFunc != nil {
		return m.AddIssueCommentFunc(ctx, issueNumber, commentText)
	}
	return nil
}

// CloseIssue закрывает задачу.
// При отсутствии пользовательской функции возвращает nil.
func (m *MockClient) CloseIssue(ctx context.Context, issueNumber int64) error {
	if m.CloseIssueFunc != nil {
		return m.CloseIssueFunc(ctx, issueNumber)
	}
	return nil
}

// -------------------------------------------------------------------
// PRManager implementation
// -------------------------------------------------------------------

// CreatePR создаёт новый Pull Request.
// При отсутствии пользовательской функции возвращает тестовые данные.
func (m *MockClient) CreatePR(ctx context.Context, head string) (gitea.PR, error) {
	if m.CreatePRFunc != nil {
		return m.CreatePRFunc(ctx, head)
	}
	return gitea.PR{
		ID:     1,
		Number: 1,
		Head:   head,
		Base:   "main",
	}, nil
}

// CreatePRWithOptions создаёт новый Pull Request с опциями.
// При отсутствии пользовательской функции возвращает тестовые данные.
func (m *MockClient) CreatePRWithOptions(ctx context.Context, opts gitea.CreatePROptions) (*gitea.PRResponse, error) {
	if m.CreatePRWithOptionsFunc != nil {
		return m.CreatePRWithOptionsFunc(ctx, opts)
	}
	return &gitea.PRResponse{
		ID:        1,
		Number:    1,
		Title:     opts.Title,
		Body:      opts.Body,
		State:     "open",
		Mergeable: true,
	}, nil
}

// MergePR выполняет слияние Pull Request.
// При отсутствии пользовательской функции возвращает nil.
func (m *MockClient) MergePR(ctx context.Context, prNumber int64) error {
	if m.MergePRFunc != nil {
		return m.MergePRFunc(ctx, prNumber)
	}
	return nil
}

// ClosePR закрывает Pull Request.
// При отсутствии пользовательской функции возвращает nil.
func (m *MockClient) ClosePR(ctx context.Context, prNumber int64) error {
	if m.ClosePRFunc != nil {
		return m.ClosePRFunc(ctx, prNumber)
	}
	return nil
}

// -------------------------------------------------------------------
// RepositoryWriter implementation
// -------------------------------------------------------------------

// SetRepositoryState выполняет batch операции.
// При отсутствии пользовательской функции возвращает nil.
func (m *MockClient) SetRepositoryState(ctx context.Context, operations []gitea.BatchOperation, branch, commitMessage string) error {
	if m.SetRepositoryStateFunc != nil {
		return m.SetRepositoryStateFunc(ctx, operations, branch, commitMessage)
	}
	return nil
}

// -------------------------------------------------------------------
// TeamReader implementation
// -------------------------------------------------------------------

// IsUserInTeam проверяет членство пользователя в команде.
// При отсутствии пользовательской функции возвращает false.
func (m *MockClient) IsUserInTeam(ctx context.Context, username, orgName, teamName string) (bool, error) {
	if m.IsUserInTeamFunc != nil {
		return m.IsUserInTeamFunc(ctx, username, orgName, teamName)
	}
	return false, nil
}

// GetTeamMembers возвращает список членов команды.
// При отсутствии пользовательской функции возвращает пустой срез.
func (m *MockClient) GetTeamMembers(ctx context.Context, orgName, teamName string) ([]string, error) {
	if m.GetTeamMembersFunc != nil {
		return m.GetTeamMembersFunc(ctx, orgName, teamName)
	}
	return []string{}, nil
}

// -------------------------------------------------------------------
// OrgReader implementation
// -------------------------------------------------------------------

// SearchOrgRepos возвращает репозитории организации.
// При отсутствии пользовательской функции возвращает пустой срез.
func (m *MockClient) SearchOrgRepos(ctx context.Context, orgName string) ([]gitea.Repository, error) {
	if m.SearchOrgReposFunc != nil {
		return m.SearchOrgReposFunc(ctx, orgName)
	}
	return []gitea.Repository{}, nil
}

// -------------------------------------------------------------------
// Конструкторы для создания MockClient
// -------------------------------------------------------------------

// NewMockClient создаёт MockClient с дефолтными тестовыми данными.
func NewMockClient() *MockClient {
	return &MockClient{}
}

// NewMockClientWithPR создаёт MockClient с предзаданным PR.
func NewMockClientWithPR(pr *gitea.PRResponse) *MockClient {
	return &MockClient{
		GetPRFunc: func(_ context.Context, _ int64) (*gitea.PRResponse, error) {
			return pr, nil
		},
	}
}

// NewMockClientWithCommits создаёт MockClient с предзаданными коммитами.
func NewMockClientWithCommits(commits []gitea.Commit) *MockClient {
	return &MockClient{
		GetCommitsFunc: func(_ context.Context, _ string, _ int) ([]gitea.Commit, error) {
			return commits, nil
		},
	}
}

// NewMockClientWithIssue создаёт MockClient с предзаданной задачей.
func NewMockClientWithIssue(issue *gitea.Issue) *MockClient {
	return &MockClient{
		GetIssueFunc: func(_ context.Context, _ int64) (*gitea.Issue, error) {
			return issue, nil
		},
	}
}

// NewMockClientWithRelease создаёт MockClient с предзаданным релизом.
func NewMockClientWithRelease(release *gitea.Release) *MockClient {
	return &MockClient{
		GetLatestReleaseFunc: func(_ context.Context) (*gitea.Release, error) {
			return release, nil
		},
		GetReleaseByTagFunc: func(_ context.Context, _ string) (*gitea.Release, error) {
			return release, nil
		},
	}
}

// -------------------------------------------------------------------
// Тестовые данные
// -------------------------------------------------------------------

// PRData возвращает тестовые данные PR для использования в тестах.
func PRData() *gitea.PRResponse {
	return &gitea.PRResponse{
		ID:        42,
		Number:    42,
		HTMLURL:   "https://gitea.example.com/owner/repo/pulls/42",
		State:     "open",
		Title:     "Тестовый Pull Request",
		Body:      "Описание тестового PR",
		Mergeable: true,
		Base: gitea.Branch{
			Name: "main",
		},
		Head: gitea.Branch{
			Name: "feature/test",
		},
	}
}

// CommitData возвращает тестовые данные коммитов для использования в тестах.
func CommitData() []gitea.Commit {
	return []gitea.Commit{
		{
			SHA: "abc123",
			Commit: gitea.CommitDetails{
				Author: gitea.CommitAuthor{
					Name:  "Test Author",
					Email: "test@example.com",
					Date:  "2026-02-04T10:00:00Z",
				},
				Message: "feat: добавлен новый функционал",
			},
		},
		{
			SHA: "def456",
			Commit: gitea.CommitDetails{
				Author: gitea.CommitAuthor{
					Name:  "Test Author",
					Email: "test@example.com",
					Date:  "2026-02-04T11:00:00Z",
				},
				Message: "fix: исправлена ошибка",
			},
		},
	}
}

// BranchData возвращает тестовые данные веток для использования в тестах.
func BranchData() []gitea.Branch {
	return []gitea.Branch{
		{
			Name: "main",
			Commit: gitea.BranchCommit{
				ID: "main123",
			},
		},
		{
			Name: "develop",
			Commit: gitea.BranchCommit{
				ID: "dev456",
			},
		},
		{
			Name: "feature/test",
			Commit: gitea.BranchCommit{
				ID: "feat789",
			},
		},
	}
}

// IssueData возвращает тестовые данные задачи для использования в тестах.
func IssueData() *gitea.Issue {
	return &gitea.Issue{
		ID:     100,
		Number: 100,
		Title:  "Тестовая задача",
		Body:   "Описание тестовой задачи",
		State:  "open",
		User: gitea.IssueUser{
			Login: "testuser",
			ID:    1,
		},
		CreatedAt: "2026-02-04T09:00:00Z",
		UpdatedAt: "2026-02-04T10:00:00Z",
	}
}

// ReleaseData возвращает тестовые данные релиза для использования в тестах.
func ReleaseData() *gitea.Release {
	return &gitea.Release{
		ID:          1,
		TagName:     "v1.0.0",
		Name:        "Release 1.0.0",
		Body:        "## Изменения\n- Первый релиз",
		CreatedAt:   "2026-02-01T12:00:00Z",
		PublishedAt: "2026-02-01T12:30:00Z",
		Assets: []gitea.ReleaseAsset{
			{
				ID:          1,
				Name:        "app-linux-amd64",
				Size:        10485760,
				DownloadURL: "https://gitea.example.com/owner/repo/releases/download/v1.0.0/app-linux-amd64",
			},
		},
	}
}

// RepositoryData возвращает тестовые данные репозитория для использования в тестах.
func RepositoryData() *gitea.Repository {
	return &gitea.Repository{
		ID:       1,
		Name:     "test-repo",
		FullName: "owner/test-repo",
		Owner: gitea.RepositoryOwner{
			ID:    1,
			Login: "owner",
			Type:  "Organization",
		},
		DefaultBranch: "main",
		Private:       false,
		Fork:          false,
	}
}
