package gitea

import (
	"context"
)

// -------------------------------------------------------------------
// Структуры данных
// -------------------------------------------------------------------

// Repository представляет полную информацию о репозитории в Gitea.
type Repository struct {
	// ID — внутренний идентификатор репозитория
	ID int64 `json:"id"`
	// Name — имя репозитория
	Name string `json:"name"`
	// FullName — полное имя репозитория (owner/name)
	FullName string `json:"full_name"`
	// Owner — владелец репозитория
	Owner RepositoryOwner `json:"owner"`
	// DefaultBranch — ветка по умолчанию
	DefaultBranch string `json:"default_branch"`
	// Private — является ли репозиторий приватным
	Private bool `json:"private"`
	// Fork — является ли репозиторий форком
	Fork bool `json:"fork"`
}

// RepositoryOwner представляет владельца репозитория.
type RepositoryOwner struct {
	// ID — идентификатор владельца
	ID int64 `json:"id"`
	// Login — логин владельца
	Login string `json:"login"`
	// Type — тип владельца ("User" или "Organization")
	Type string `json:"type"`
}

// PR представляет Pull Request (краткая форма).
type PR struct {
	// ID — внутренний идентификатор PR
	ID int64
	// Number — номер PR в репозитории
	Number int64
	// Base — имя целевой ветки
	Base string
	// Head — имя исходной ветки
	Head string
}

// PRResponse представляет ответ API при создании или получении Pull Request.
type PRResponse struct {
	// ID — внутренний идентификатор PR в Gitea
	ID int64 `json:"id"`
	// Number — номер PR в репозитории
	Number int64 `json:"number"`
	// HTMLURL — URL для просмотра PR в веб-интерфейсе
	HTMLURL string `json:"html_url"`
	// State — состояние PR (open, closed, merged)
	State string `json:"state"`
	// Title — заголовок PR
	Title string `json:"title"`
	// Body — описание PR
	Body string `json:"body"`
	// Mergeable — можно ли автоматически слить PR
	Mergeable bool `json:"mergeable"`
	// Base — целевая ветка
	Base Branch `json:"base"`
	// Head — исходная ветка
	Head Branch `json:"head"`
}

// Branch представляет ветку репозитория.
type Branch struct {
	// Label — метка ветки (owner:branch)
	Label string `json:"label"`
	// Name — имя ветки
	Name string `json:"name"`
	// Commit — информация о коммите на вершине ветки
	Commit BranchCommit `json:"commit"`
}

// BranchCommit представляет краткую информацию о коммите в ветке.
type BranchCommit struct {
	// ID — SHA коммита
	ID string `json:"id"`
}

// Commit представляет информацию о коммите.
type Commit struct {
	// SHA — хеш коммита
	SHA string `json:"sha"`
	// Commit — детали коммита
	Commit CommitDetails `json:"commit"`
}

// CommitDetails представляет детальную информацию о коммите.
type CommitDetails struct {
	// Author — автор коммита
	Author CommitAuthor `json:"author"`
	// Committer — коммиттер
	Committer CommitAuthor `json:"committer"`
	// Message — сообщение коммита
	Message string `json:"message"`
}

// CommitAuthor представляет автора или коммиттера.
type CommitAuthor struct {
	// Name — имя
	Name string `json:"name"`
	// Email — email
	Email string `json:"email"`
	// Date — дата в строковом формате
	Date string `json:"date"`
}

// CommitFile представляет файл в коммите.
type CommitFile struct {
	// Filename — имя файла
	Filename string `json:"filename"`
	// Status — статус изменения (added, modified, removed)
	Status string `json:"status"`
	// Patch — патч с изменениями
	Patch string `json:"patch"`
}

// BranchCommitRange представляет диапазон коммитов в ветке.
type BranchCommitRange struct {
	// FirstCommit — первый коммит ветки
	FirstCommit *Commit `json:"first_commit"`
	// LastCommit — последний коммит ветки
	LastCommit *Commit `json:"last_commit"`
}

// Issue представляет задачу в Gitea.
type Issue struct {
	// ID — внутренний идентификатор задачи
	ID int64 `json:"id"`
	// Number — номер задачи в репозитории
	Number int64 `json:"number"`
	// Title — заголовок задачи
	Title string `json:"title"`
	// Body — описание задачи
	Body string `json:"body"`
	// State — состояние (open, closed)
	State string `json:"state"`
	// User — автор задачи
	User IssueUser `json:"user"`
	// CreatedAt — дата создания
	CreatedAt string `json:"created_at"`
	// UpdatedAt — дата обновления
	UpdatedAt string `json:"updated_at"`
}

// IssueUser представляет пользователя-автора задачи.
type IssueUser struct {
	// Login — логин пользователя
	Login string `json:"login"`
	// ID — идентификатор пользователя
	ID int64 `json:"id"`
}

// FileInfo представляет информацию о файле или каталоге в репозитории.
type FileInfo struct {
	// Name — имя файла/каталога
	Name string `json:"name"`
	// Path — путь к файлу/каталогу
	Path string `json:"path"`
	// SHA — хеш содержимого
	SHA string `json:"sha"`
	// Size — размер файла
	Size int64 `json:"size"`
	// URL — URL для API
	URL string `json:"url"`
	// HTMLURL — URL для веб-интерфейса
	HTMLURL string `json:"html_url"`
	// GitURL — URL для git
	GitURL string `json:"git_url"`
	// DownloadURL — URL для скачивания
	DownloadURL string `json:"download_url"`
	// Type — тип (file, dir)
	Type string `json:"type"`
	// Content — содержимое файла (base64)
	Content string `json:"content"`
	// Encoding — кодировка содержимого
	Encoding string `json:"encoding"`
	// Target — цель символической ссылки
	Target string `json:"target"`
	// Submodule — URL субмодуля
	Submodule string `json:"submodule"`
}

// Release представляет информацию о релизе.
type Release struct {
	// ID — идентификатор релиза
	ID int64 `json:"id"`
	// TagName — имя тега
	TagName string `json:"tag_name"`
	// Name — имя релиза
	Name string `json:"name"`
	// Body — описание релиза
	Body string `json:"body"`
	// Assets — прикреплённые файлы
	Assets []ReleaseAsset `json:"assets"`
	// CreatedAt — дата создания
	CreatedAt string `json:"created_at"`
	// PublishedAt — дата публикации
	PublishedAt string `json:"published_at"`
}

// ReleaseAsset представляет прикреплённый файл к релизу.
type ReleaseAsset struct {
	// ID — идентификатор файла
	ID int64 `json:"id"`
	// Name — имя файла
	Name string `json:"name"`
	// Size — размер файла
	Size int64 `json:"size"`
	// DownloadURL — URL для скачивания
	DownloadURL string `json:"browser_download_url"`
}

// -------------------------------------------------------------------
// Options структуры для методов API
// -------------------------------------------------------------------

// CreatePROptions содержит параметры для создания Pull Request.
type CreatePROptions struct {
	// Title — заголовок Pull Request
	Title string `json:"title"`
	// Body — описание Pull Request (поддерживает markdown)
	Body string `json:"body"`
	// Head — имя исходной ветки с изменениями
	Head string `json:"head"`
	// Base — имя целевой ветки для слияния
	Base string `json:"base"`
	// Assignees — список логинов назначенных пользователей (опционально)
	Assignees []string `json:"assignees,omitempty"`
	// Labels — список ID меток (опционально)
	Labels []int64 `json:"labels,omitempty"`
}

// BatchOperation представляет операцию над файлом в batch запросе.
type BatchOperation struct {
	// Operation — тип операции ("create", "update", "delete")
	Operation string `json:"operation"`
	// Path — путь к файлу
	Path string `json:"path"`
	// Content — содержимое файла (для create/update)
	Content string `json:"content,omitempty"`
	// SHA — SHA файла (для update/delete)
	SHA string `json:"sha,omitempty"`
	// FromPath — исходный путь (для rename)
	FromPath string `json:"from_path,omitempty"`
}

// -------------------------------------------------------------------
// ISP-compliant интерфейсы
// -------------------------------------------------------------------

// PRReader предоставляет операции для чтения информации о Pull Requests.
type PRReader interface {
	// GetPR возвращает информацию о Pull Request по номеру.
	GetPR(ctx context.Context, prNumber int64) (*PRResponse, error)
	// ListOpenPRs возвращает список открытых Pull Requests.
	ListOpenPRs(ctx context.Context) ([]PR, error)
	// ConflictPR проверяет наличие конфликтов в Pull Request.
	ConflictPR(ctx context.Context, prNumber int64) (bool, error)
	// ConflictFilesPR возвращает список файлов с конфликтами в Pull Request.
	ConflictFilesPR(ctx context.Context, prNumber int64) ([]string, error)
}

// CommitReader предоставляет операции для чтения информации о коммитах.
type CommitReader interface {
	// GetCommits возвращает список коммитов для ветки.
	GetCommits(ctx context.Context, branch string, limit int) ([]Commit, error)
	// GetLatestCommit возвращает последний коммит ветки.
	GetLatestCommit(ctx context.Context, branch string) (*Commit, error)
	// GetCommitFiles возвращает список файлов, изменённых в коммите.
	GetCommitFiles(ctx context.Context, commitSHA string) ([]CommitFile, error)
	// GetCommitsBetween возвращает коммиты между двумя SHA.
	GetCommitsBetween(ctx context.Context, baseCommitSHA, headCommitSHA string) ([]Commit, error)
	// GetFirstCommitOfBranch возвращает первый коммит ветки относительно базовой.
	GetFirstCommitOfBranch(ctx context.Context, branch, baseBranch string) (*Commit, error)
	// GetBranchCommitRange возвращает диапазон коммитов ветки.
	GetBranchCommitRange(ctx context.Context, branch string) (*BranchCommitRange, error)
}

// FileReader предоставляет операции для чтения файлов из репозитория.
type FileReader interface {
	// GetFileContent возвращает содержимое файла.
	GetFileContent(ctx context.Context, fileName string) ([]byte, error)
	// GetRepositoryContents возвращает содержимое каталога.
	GetRepositoryContents(ctx context.Context, filepath, branch string) ([]FileInfo, error)
	// AnalyzeProjectStructure анализирует структуру проекта.
	AnalyzeProjectStructure(ctx context.Context, branch string) ([]string, error)
}

// BranchManager предоставляет операции для управления ветками.
type BranchManager interface {
	// GetBranches возвращает список веток репозитория.
	GetBranches(ctx context.Context, repo string) ([]Branch, error)
	// CreateBranch создаёт новую ветку на основе базовой.
	CreateBranch(ctx context.Context, newBranch, baseBranch string) error
	// DeleteBranch удаляет ветку по имени.
	DeleteBranch(ctx context.Context, branchName string) error
}

// ReleaseReader предоставляет операции для чтения информации о релизах.
type ReleaseReader interface {
	// GetLatestRelease возвращает последний релиз.
	GetLatestRelease(ctx context.Context) (*Release, error)
	// GetReleaseByTag возвращает релиз по тегу.
	GetReleaseByTag(ctx context.Context, tag string) (*Release, error)
}

// IssueManager предоставляет операции для работы с задачами.
type IssueManager interface {
	// GetIssue возвращает информацию о задаче по номеру.
	GetIssue(ctx context.Context, issueNumber int64) (*Issue, error)
	// AddIssueComment добавляет комментарий к задаче.
	AddIssueComment(ctx context.Context, issueNumber int64, commentText string) error
	// CloseIssue закрывает задачу.
	CloseIssue(ctx context.Context, issueNumber int64) error
}

// PRManager предоставляет операции для управления Pull Requests.
type PRManager interface {
	// CreatePR создаёт новый Pull Request.
	CreatePR(ctx context.Context, head string) (PR, error)
	// CreatePRWithOptions создаёт новый Pull Request с дополнительными опциями.
	CreatePRWithOptions(ctx context.Context, opts CreatePROptions) (*PRResponse, error)
	// MergePR выполняет слияние Pull Request.
	MergePR(ctx context.Context, prNumber int64) error
	// ClosePR закрывает Pull Request без слияния.
	ClosePR(ctx context.Context, prNumber int64) error
}

// RepositoryWriter предоставляет операции для записи в репозиторий.
type RepositoryWriter interface {
	// SetRepositoryState выполняет batch операции над файлами.
	SetRepositoryState(ctx context.Context, operations []BatchOperation, branch, commitMessage string) error
}

// TeamReader предоставляет операции для чтения информации о командах.
type TeamReader interface {
	// IsUserInTeam проверяет, является ли пользователь членом команды.
	IsUserInTeam(ctx context.Context, username, orgName, teamName string) (bool, error)
	// GetTeamMembers возвращает список членов команды.
	GetTeamMembers(ctx context.Context, orgName, teamName string) ([]string, error)
}

// OrgReader предоставляет операции для чтения информации об организациях.
type OrgReader interface {
	// SearchOrgRepos возвращает список репозиториев организации.
	SearchOrgRepos(ctx context.Context, orgName string) ([]Repository, error)
}

// Client — композитный интерфейс, объединяющий все операции Gitea.
type Client interface {
	PRReader
	CommitReader
	FileReader
	BranchManager
	ReleaseReader
	IssueManager
	PRManager
	RepositoryWriter
	TeamReader
	OrgReader
}
