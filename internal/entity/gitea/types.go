package gitea

// PR представляет Pull Request
type PR struct {
	ID     int64
	Number int64
	Base   string
	Head   string
}

// Repo представляет репозиторий
type Repo struct {
	DefaultBranch string `json:"default_branch"`
}

// PRData содержит данные Pull Request
type PRData struct {
	ID     int64  `json:"id"`
	Number int64  `json:"number"`
	Base   Branch `json:"base"`
	Head   Branch `json:"head"`
}

// Branch представляет ветку
type Branch struct {
	Label  string `json:"label"`
	Name   string `json:"name"`
	Commit struct {
		ID string `json:"id"`
	} `json:"commit"`
}

// PullRequest представляет информацию о запросе на слияние
type PullRequest struct {
	Number         int    `json:"number"`
	Mergeable      bool   `json:"mergeable"`
	MergeableState string `json:"mergeable_state"` // "checking", "success", "conflict", "behind", "blocked", "unstable", "has_hooks", "unknown"
}

// ConflictFile представляет файл с конфликтом слияния
type ConflictFile struct {
	Filename string `json:"filename"`
}

// Issue представляет задачу в Gitea
type Issue struct {
	ID     int64  `json:"id"`
	Number int64  `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
	User   struct {
		Login string `json:"login"`
		ID    int64  `json:"id"`
	} `json:"user"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// FileInfo представляет информацию о файле или каталоге в репозитории
type FileInfo struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	SHA         string `json:"sha"`
	Size        int64  `json:"size"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	GitURL      string `json:"git_url"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	Encoding    string `json:"encoding"`
	Target      string `json:"target"`
	Submodule   string `json:"submodule"`
}

// ProjectAnalysis представляет результат анализа проекта
type ProjectAnalysis struct {
	ProjectName string   `json:"project_name"`
	Extensions  []string `json:"extensions"`
}

// Repository представляет полную информацию о репозитории в Gitea.
// Используется для получения списка репозиториев организации.
type Repository struct {
	ID            int64           `json:"id"`
	Name          string          `json:"name"`
	FullName      string          `json:"full_name"`
	Owner         RepositoryOwner `json:"owner"`
	DefaultBranch string          `json:"default_branch"`
	Private       bool            `json:"private"`
	Fork          bool            `json:"fork"`
}

// RepositoryOwner представляет владельца репозитория.
// Содержит информацию о пользователе или организации, владеющей репозиторием.
type RepositoryOwner struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Type  string `json:"type"` // "User" или "Organization"
}

// Константы для работы с пагинацией при получении репозиториев организации.
const (
	// SearchOrgReposMaxPages — максимальное количество страниц для запроса репозиториев.
	// Защита от бесконечного цикла. 100 страниц × 50 = 5000 репозиториев максимум.
	SearchOrgReposMaxPages = 100

	// SearchOrgReposPageLimit — количество репозиториев на одной странице.
	// Максимальное значение, поддерживаемое Gitea API.
	SearchOrgReposPageLimit = 50
)

// Commit представляет информацию о коммите
type Commit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Author struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			Date  string `json:"date"`
		} `json:"author"`
		Committer struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			Date  string `json:"date"`
		} `json:"committer"`
		Message string `json:"message"`
	} `json:"commit"`
}

// CommitFile представляет файл в коммите.
// Содержит информацию о файле, который был изменен в коммите,
// включая имя файла, статус изменения и патч с изменениями.
type CommitFile struct {
	Filename string `json:"filename"`
	Status   string `json:"status"`
	Patch    string `json:"patch"`
}

// CompareResult представляет результат сравнения веток.
// Содержит информацию о различиях между двумя ветками,
// включая общий предок (merge base commit).
type CompareResult struct {
	MergeBaseCommit *Commit  `json:"merge_base_commit"`
	Commits         []Commit `json:"commits"`
}

// BranchCommitRange представляет диапазон коммитов в ветке.
// Содержит первый и последний коммит ветки в хронологическом порядке.
type BranchCommitRange struct {
	FirstCommit *Commit `json:"first_commit"`
	LastCommit  *Commit `json:"last_commit"`
}

// ChangeFileOperation представляет операцию над файлом в batch запросе.
// Описывает операцию, которая должна быть выполнена над файлом в репозитории,
// такую как создание, обновление или удаление файла.
type ChangeFileOperation struct {
	Operation string `json:"operation"` // "create", "update", "delete"
	Path      string `json:"path"`
	Content   string `json:"content,omitempty"`
	SHA       string `json:"sha,omitempty"`
	FromPath  string `json:"from_path,omitempty"`
}

// Identity представляет идентификацию автора или коммиттера.
// Содержит имя и email адрес пользователя для идентификации
// автора или коммиттера в системе контроля версий.
type Identity struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// CommitDateOptions представляет даты для коммита.
// Позволяет указать пользовательские даты для автора и коммиттера
// при создании коммита в репозитории.
type CommitDateOptions struct {
	Author    string `json:"author,omitempty"`
	Committer string `json:"committer,omitempty"`
}

// ChangeFilesOptions представляет запрос для изменения множественных файлов.
// Содержит все необходимые параметры для выполнения пакетных операций
// над файлами в репозитории, включая информацию об авторе, ветке и коммите.
type ChangeFilesOptions struct {
	Author    *Identity             `json:"author,omitempty"`
	Branch    string                `json:"branch,omitempty"`
	Committer *Identity             `json:"committer,omitempty"`
	Dates     *CommitDateOptions    `json:"dates,omitempty"`
	Files     []ChangeFileOperation `json:"files"`
	Message   string                `json:"message,omitempty"`
	NewBranch string                `json:"new_branch,omitempty"`
	Signoff   bool                  `json:"signoff,omitempty"`
}

// BatchOperation - алиас для обратной совместимости
type BatchOperation = ChangeFileOperation

// CommitAuthor - алиас для обратной совместимости
type CommitAuthor = Identity

// CreatePROptions содержит параметры для создания Pull Request.
// Используется для создания PR с полной информацией через Gitea API.
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

// PRResponse представляет ответ API при создании Pull Request.
// Содержит полную информацию о созданном PR.
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
}

// Release представляет информацию о релизе в Gitea.
// Содержит метаданные релиза, включая тег, описание и прикрепленные файлы.
type Release struct {
	ID          int64          `json:"id"`
	TagName     string         `json:"tag_name"`
	Name        string         `json:"name"`
	Body        string         `json:"body"`
	Assets      []ReleaseAsset `json:"assets"`
	CreatedAt   string         `json:"created_at"`
	PublishedAt string         `json:"published_at"`
}

// ReleaseAsset представляет прикрепленный файл к релизу.
// Содержит информацию о файле, включая его размер и ссылку для скачивания.
type ReleaseAsset struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"browser_download_url"`
}

// API предоставляет методы для работы с API Gitea
type API struct {
	GiteaURL    string
	Owner       string
	Repo        string
	AccessToken string
	BaseBranch  string
	NewBranch   string
	Command     string
}

// NewGiteaAPI создает новый экземпляр API для работы с Gitea.
// Инициализирует клиент для взаимодействия с Gitea API, используя конфигурацию
// для выполнения операций с репозиториями и задачами.
// Параметры:
//   - config: конфигурация с настройками подключения к Gitea
//
// Возвращает:
//   - *API: указатель на новый экземпляр API клиента Gitea
func NewGiteaAPI(config Config) *API {
	return &API{
		GiteaURL:    config.GiteaURL,
		Owner:       config.Owner,
		Repo:        config.Repo,
		AccessToken: config.AccessToken,
		BaseBranch:  config.BaseBranch,
		NewBranch:   config.NewBranch,
		Command:     config.Command,
	}
}

// UpdateConfig обновляет конфигурацию API клиента Gitea.
// Изменяет настройки подключения и параметры работы с репозиторием
// для адаптации к новым требованиям или изменениям в проекте.
// Параметры:
//   - config: новая конфигурация с обновленными настройками
func (g *API) UpdateConfig(config Config) {
	g.Command = config.Command
	g.AccessToken = config.AccessToken
	g.GiteaURL = config.GiteaURL
	g.Owner = config.Owner
	g.Repo = config.Repo
	g.BaseBranch = config.BaseBranch
	g.NewBranch = config.NewBranch
}

// Organization представляет организацию в Gitea.
type Organization struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Username string `json:"username"`
}

// Константы для пагинации организаций
const (
	GetUserOrgsMaxPages  = 100
	GetUserOrgsPageLimit = 50
)
