package gitea

import (
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
	GetIssue(issueNumber int64) (*Issue, error)
	GetFileContent(fileName string) ([]byte, error)
	GetConfigData(l *slog.Logger, filename string) ([]byte, error)
	AddIssueComment(issueNumber int64, commentText string) error
	CloseIssue(issueNumber int64) error
	ConflictPR(prNumber int64) (bool, error)
	ConflictFilesPR(prNumber int64) ([]string, error)
	GetRepositoryContents(filepath, branch string) ([]FileInfo, error)
	AnalyzeProjectStructure(branch string) ([]string, error)
	AnalyzeProject(branch string) ([]string, error)
	GetLatestCommit(branch string) (*Commit, error)
	GetCommitFiles(commitSHA string) ([]CommitFile, error)
	// Методы для работы с релизами
	GetLatestRelease() (*Release, error)
	GetReleaseByTag(tag string) (*Release, error)
	IsUserInTeam(l *slog.Logger, username string, orgName string, teamName string) (bool, error)
	// Методы для работы с коммитами и историей
	GetCommits(branch string, limit int) ([]Commit, error)
	GetFirstCommitOfBranch(branch string, baseBranch string) (*Commit, error)
	GetCommitsBetween(baseCommitSHA, headCommitSHA string) ([]Commit, error)
	GetBranchCommitRange(branch string) (*BranchCommitRange, error)
	// Методы для работы с PR и ветками
	ActivePR() ([]PR, error)
	DeleteTestBranch() error
	CreateTestBranch() error
	CreatePR(head string) (PR, error)
	CreatePRWithOptions(opts CreatePROptions) (*PRResponse, error)
	MergePR(prNumber int64, l *slog.Logger) error
	ClosePR(prNumber int64) error
	// Batch операции с файлами
	SetRepositoryState(l *slog.Logger, operations []BatchOperation, branch, commitMessage string) error
	// Методы для работы с командами
	GetTeamMembers(orgName, teamName string) ([]string, error)
	// Методы для работы с ветками
	GetBranches(repo string) ([]Branch, error)
	// Методы для работы с организациями
	SearchOrgRepos(orgName string) ([]Repository, error)
}

// ProjectAnalyzer определяет интерфейс для анализа проектов
type ProjectAnalyzer interface {
	AnalyzeProject(l *slog.Logger, branch string) error
}
