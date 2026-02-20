package gitea

import (
	"context"
	"log/slog"
)

// Config содержит конфигурацию для Gitea API
type Config struct {
	GiteaURL    string
	Owner       string
	Repo        string
	AccessToken string
	BaseBranch  string
	NewBranch   string
	Command     string
}

// APIInterface определяет интерфейс для работы с Gitea API
type APIInterface interface {
	GetIssue(ctx context.Context, issueNumber int64) (*Issue, error)
	GetFileContent(ctx context.Context, fileName string) ([]byte, error)
	GetConfigData(ctx context.Context, l *slog.Logger, filename string) ([]byte, error)
	AddIssueComment(ctx context.Context, issueNumber int64, commentText string) error
	CloseIssue(ctx context.Context, issueNumber int64) error
	ConflictPR(ctx context.Context, prNumber int64) (bool, error)
	ConflictFilesPR(ctx context.Context, prNumber int64) ([]string, error)
	GetRepositoryContents(ctx context.Context, filepath, branch string) ([]FileInfo, error)
	AnalyzeProjectStructure(ctx context.Context, branch string) ([]string, error)
	AnalyzeProject(ctx context.Context, branch string) ([]string, error)
	GetLatestCommit(ctx context.Context, branch string) (*Commit, error)
	GetCommitFiles(ctx context.Context, commitSHA string) ([]CommitFile, error)
	// Методы для работы с релизами
	GetLatestRelease(ctx context.Context) (*Release, error)
	GetReleaseByTag(ctx context.Context, tag string) (*Release, error)
	IsUserInTeam(ctx context.Context, l *slog.Logger, username string, orgName string, teamName string) (bool, error)
	// Методы для работы с коммитами и историей
	GetCommits(ctx context.Context, branch string, limit int) ([]Commit, error)
	GetFirstCommitOfBranch(ctx context.Context, branch string, baseBranch string) (*Commit, error)
	GetCommitsBetween(ctx context.Context, baseCommitSHA, headCommitSHA string) ([]Commit, error)
	GetBranchCommitRange(ctx context.Context, branch string) (*BranchCommitRange, error)
	// Методы для работы с PR и ветками
	ActivePR(ctx context.Context) ([]PR, error)
	DeleteTestBranch(ctx context.Context) error
	CreateTestBranch(ctx context.Context) error
	CreatePR(ctx context.Context, head string) (PR, error)
	CreatePRWithOptions(ctx context.Context, opts CreatePROptions) (*PRResponse, error)
	MergePR(ctx context.Context, prNumber int64, l *slog.Logger) error
	ClosePR(ctx context.Context, prNumber int64) error
	// Batch операции с файлами
	SetRepositoryState(ctx context.Context, l *slog.Logger, operations []ChangeFileOperation, branch, commitMessage string) error
	// Методы для работы с командами
	GetTeamMembers(ctx context.Context, orgName, teamName string) ([]string, error)
	// Методы для работы с ветками
	GetBranches(ctx context.Context, repo string) ([]Branch, error)
	// Методы для работы с организациями
	SearchOrgRepos(ctx context.Context, orgName string) ([]Repository, error)
}

// ProjectAnalyzer определяет интерфейс для анализа проектов
type ProjectAnalyzer interface {
	AnalyzeProject(ctx context.Context, l *slog.Logger, branch string) error
}
