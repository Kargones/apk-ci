// Package gitea предоставляет API для работы с Gitea
package gitea

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/constants"
)

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

// CreateTestBranch создает новую тестовую ветку в репозитории Gitea.
// Создает ветку на основе указанной базовой ветки для выполнения тестирования
// или разработки новых функций без влияния на основную ветку.
// Возвращает:
//   - error: ошибка создания ветки или nil при успехе
func (g *API) CreateTestBranch() error {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/branches", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo)
	reqBody := fmt.Sprintf(`{
"new_branch_name": "%s",
"old_branch_name": "%s",
"old_ref_name": "refs/heads/%s"
}`, g.NewBranch, g.BaseBranch, g.BaseBranch)

	statusCode, _, err := g.sendReq(urlString, reqBody, "POST")

	if statusCode != http.StatusCreated {
		return fmt.Errorf("ошибка при создании ветки: %v %v", statusCode, err)
	}
	return nil
}

// GetIssue получает информацию о задаче (issue) из репозитория Gitea.
// Извлекает детальную информацию о задаче по её номеру, включая описание,
// статус, назначенных пользователей и комментарии.
// Параметры:
//   - issueNumber: номер задачи для получения информации
//
// Возвращает:
//   - *Issue: указатель на структуру с информацией о задаче
//   - error: ошибка получения задачи или nil при успехе
func (g *API) GetIssue(issueNumber int64) (*Issue, error) {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/issues/%d", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, issueNumber)

	statusCode, body, err := g.sendReq(urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении задачи %d: статус %d", issueNumber, statusCode)
	}

	r := strings.NewReader(body)
	var issue Issue
	if err := json.NewDecoder(r).Decode(&issue); err != nil {
		return nil, fmt.Errorf("ошибка при декодировании ответа: %v", err)
	}

	return &issue, nil
}

// GetFileContent получает содержимое файла из репозитория Gitea или по прямому URL.
// Извлекает содержимое указанного файла из корня репозитория для анализа
// или обработки. Возвращает декодированное содержимое файла.
// Если fileName содержит полный URL (начинающийся с http:// или https://),
// то используется этот URL напрямую без построения пути через API Gitea.
// Параметры:
//   - fileName: имя файла в корне репозитория или полный URL для загрузки
//
// Возвращает:
//   - []byte: содержимое файла в виде массива байт
//   - error: ошибка получения файла или nil при успехе
func (g *API) GetFileContent(fileName string) ([]byte, error) {
	var urlString string
	if strings.HasPrefix(fileName, "http://") || strings.HasPrefix(fileName, "https://") {
		urlString = fileName
	} else {
		urlString = fmt.Sprintf("%s/api/%s/repos/%s/%s/contents/%s", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, fileName)
	}

	statusCode, body, err := g.sendReq(urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении файла %s: статус %d", fileName, statusCode)
	}

	r := strings.NewReader(body)
	var fileData struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.NewDecoder(r).Decode(&fileData); err != nil {
		return nil, fmt.Errorf("ошибка при декодировании ответа: %v", err)
	}

	// Декодируем base64 содержимое
	if fileData.Encoding == "base64" {
		content := strings.ReplaceAll(fileData.Content, "\n", "")
		decodedBytes, err := base64.StdEncoding.DecodeString(content)
		if err != nil {
			return nil, fmt.Errorf("ошибка при декодировании base64: %v", err)
		}
		return decodedBytes, nil
	}

	return nil, nil
}

// AddIssueComment добавляет комментарий к задаче в репозитории Gitea.
// Создает новый комментарий к указанной задаче для обсуждения,
// предоставления обратной связи или документирования изменений.
// Параметры:
//   - issueNumber: номер задачи для добавления комментария
//   - commentText: текст комментария
//
// Возвращает:
//   - error: ошибка добавления комментария или nil при успехе
func (g *API) AddIssueComment(issueNumber int64, commentText string) error {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/issues/%d/comments", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, issueNumber)
	reqBody := fmt.Sprintf(`{"body":"%s"}`, strings.ReplaceAll(commentText, "\"", "\\\""))

	statusCode, _, err := g.sendReq(urlString, reqBody, "POST")
	if err != nil {
		return fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}

	if statusCode != http.StatusCreated {
		return fmt.Errorf("ошибка при добавлении комментария к задаче %d: статус %d", issueNumber, statusCode)
	}

	return nil
}

// CloseIssue закрывает задачу в репозитории Gitea.
// Изменяет статус задачи на "закрыто", указывая на завершение работы
// над задачей или её решение.
// Параметры:
//   - issueNumber: номер задачи для закрытия
//
// Возвращает:
//   - error: ошибка закрытия задачи или nil при успехе
func (g *API) CloseIssue(issueNumber int64) error {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/issues/%d", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, issueNumber)
	reqBody := `{"state":"closed"}`

	statusCode, _, err := g.sendReq(urlString, reqBody, "PATCH")
	if err != nil {
		return fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}

	if statusCode != http.StatusCreated {
		return fmt.Errorf("ошибка при закрытии задачи %d: статус %d", issueNumber, statusCode)
	}

	return nil
}

func (g *API) sendReq(urlString, reqBody, respType string) (int, string, error) {
	var client *http.Client
	var req *http.Request
	var err error
	if reqBody == "" {
		req, err = http.NewRequest(respType, urlString, nil)
	} else {
		req, err = http.NewRequest(respType, urlString, bytes.NewBuffer([]byte(reqBody)))
	}
	if err != nil {
		return -1, "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", g.AccessToken))
	req.Header.Set("Content-Type", "application/json")

	client = &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return -1, "", err
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		return -1, "", err
	}
	bodyString := string(bodyBytes)
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("Failed to close response body: %v", closeErr)
		}
	}()

	return resp.StatusCode, bodyString, err
}

// DeleteTestBranch удаляет тестовую ветку из репозитория Gitea.
// Очищает временные ветки после завершения тестирования или разработки
// для поддержания чистоты репозитория.
// Возвращает:
//   - error: ошибка удаления ветки или nil при успехе
func (g *API) DeleteTestBranch() error {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/branches/%s", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, g.NewBranch)

	statusCode, _, err := g.sendReq(urlString, "", "DELETE")
	if statusCode != http.StatusNoContent {
		return fmt.Errorf("ошибка при создании ветки: %v %v", statusCode, err)
	}
	return nil
}

// MergePR выполняет слияние запроса на изменение в репозитории Gitea.
// Объединяет изменения из ветки запроса в целевую ветку после
// прохождения проверок и одобрения.
// Параметры:
//   - prNumber: номер запроса на изменение для слияния
//   - l: логгер для записи отладочной информации
//
// Возвращает:
//   - error: ошибка слияния или nil при успехе
func (g *API) MergePR(prNumber int64, l *slog.Logger) error {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/pulls/%d/merge", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, prNumber)
	reqBody := `{
		"Do": "merge",
		"delete_branch_after_merge": false,
		"force_merge": true,
		"merge_when_checks_succeed": false
		}`

	statusCode, bodyString, err := g.sendReq(urlString, reqBody, "POST")
	l.Debug("Ответ сервера при слиянии PR",
		slog.String("BR_ACTION", "merge-pr"),
		slog.Int64("BR_PR_NUMBER", prNumber),
		slog.String("BR_RESPONSE", bodyString),
	)
	if statusCode != http.StatusOK {
		return fmt.Errorf("ошибка при слиянии PR: %d %d %e", prNumber, statusCode, err)
	}
	return nil
}

// ConflictPR проверяет наличие конфликтов в запросе на изменение.
// Определяет, можно ли автоматически выполнить слияние или требуется
// ручное разрешение конфликтов в коде.
// При работе с большими репозиториями поддерживает асинхронную проверку:
// если Gitea возвращает статус "checking", функция ожидает завершения проверки.
// Параметры:
//   - prNumber: номер запроса на изменение для проверки
//
// Возвращает:
//   - bool: true если есть конфликты, false если их нет
//   - error: ошибка проверки или nil при успехе
func (g *API) ConflictPR(prNumber int64) (bool, error) {
	const (
		maxRetries      = 60             // Максимальное количество попыток (60 * 5 секунд = 5 минут)
		retryInterval   = 5 * time.Second // Интервал между попытками
	)

	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/pulls/%d", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, prNumber)

	for attempt := 0; attempt < maxRetries; attempt++ {
		statusCode, body, _ := g.sendReq(urlString, "", "GET")
		if statusCode != http.StatusOK {
			return true, fmt.Errorf("ошибка при получении данных PR: %d %d", prNumber, statusCode)
		}
		r := strings.NewReader(body)

		var pr PullRequest
		if err := json.NewDecoder(r).Decode(&pr); err != nil {
			return true, err
		}

		// Проверяем статус проверки слияния
		switch pr.MergeableState {
		case "checking":
			// Проверка еще выполняется в фоне, ждем и повторяем запрос
			if attempt < maxRetries-1 {
				log.Printf("PR %d: проверка конфликтов выполняется в фоне, ожидание %v (попытка %d/%d)",
					prNumber, retryInterval, attempt+1, maxRetries)
				time.Sleep(retryInterval)
				continue
			}
			// Достигнут максимум попыток
			return true, fmt.Errorf("тайм-аут ожидания завершения проверки конфликтов для PR %d", prNumber)

		case "conflict", "behind", "blocked":
			// Есть конфликты или другие проблемы, препятствующие слиянию
			return true, nil

		case "success", "unstable", "has_hooks":
			// Можно выполнить слияние (unstable и has_hooks не являются критическими блокерами)
			return false, nil

		default:
			// Для неизвестных статусов или пустого значения используем поле Mergeable
			if pr.Mergeable {
				return false, nil
			}
			return true, nil
		}
	}

	// Не должны сюда попасть, но на всякий случай
	return true, fmt.Errorf("не удалось определить статус конфликта для PR %d", prNumber)
}

// ConflictFilesPR получает список файлов с конфликтами в запросе на изменение.
// Возвращает детальную информацию о файлах, которые имеют конфликты
// и требуют ручного разрешения перед слиянием.
// Параметры:
//   - prNumber: номер запроса на изменение
//
// Возвращает:
//   - []string: список путей к файлам с конфликтами
//   - error: ошибка получения информации или nil при успехе
func (g *API) ConflictFilesPR(prNumber int64) ([]string, error) {
	var err error
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/pulls/%d/files", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, prNumber)
	statusCode, body, _ := g.sendReq(urlString, "", "GET")
	if statusCode != http.StatusNoContent {
		return []string{}, fmt.Errorf("ошибка при получении конфликтующих файлов PR: %d %d", prNumber, statusCode)
	}
	r := strings.NewReader(body)

	var conflictFiles []ConflictFile
	if err = json.NewDecoder(r).Decode(&conflictFiles); err != nil {
		return []string{}, fmt.Errorf("ошибка при декодировании конфликтующих файлов PR: %d %e", prNumber, err)
	}
	files := make([]string, 0, len(conflictFiles))
	for _, file := range conflictFiles {
		files = append(files, file.Filename)
	}
	return files, nil
}

// CreatePR создает новый запрос на изменение в репозитории Gitea.
// Инициирует процесс рецензирования кода путем создания запроса
// на слияние изменений из одной ветки в другую.
// Параметры:
//   - head: исходная ветка с изменениями
//
// Возвращает:
//   - PR: созданный запрос на изменение
//   - error: ошибка создания или nil при успехе
func (g *API) CreatePR(head string) (PR, error) {
	pr := PR{}

	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/pulls", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo)
	reqBody := fmt.Sprintf(`{
		"base": "%s",
		"body": "Test conflict",
		"head": "%s",
		"title": "Test merge %s to %s"
	}`, g.NewBranch, head, head, g.NewBranch)

	statusCode, body, err := g.sendReq(urlString, reqBody, "POST")
	if statusCode != http.StatusCreated {
		return pr, fmt.Errorf("ошибка при создании пулл реквеста: статус %d, ответ: %s, ошибка: %v", statusCode, body, err)
	}
	r := strings.NewReader(body)
	var prResp PRData
	if err := json.NewDecoder(r).Decode(&prResp); err != nil {
		return pr, err
	}
	pr = PR{prResp.ID, prResp.Number, prResp.Base.Label, prResp.Head.Label}

	return pr, nil
}

// CreatePRWithOptions создает новый Pull Request с полными параметрами.
// Позволяет задать заголовок, описание, assignees и labels.
// Обрабатывает случай, когда PR уже существует (HTTP 409).
// Параметры:
//   - opts: опции для создания PR (title, body, head, base, assignees, labels)
//
// Возвращает:
//   - *PRResponse: информация о созданном PR
//   - error: ошибка создания или nil при успехе
func (g *API) CreatePRWithOptions(opts CreatePROptions) (*PRResponse, error) {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/pulls", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo)

	// Сериализуем опции в JSON
	requestBody, err := json.Marshal(opts)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации параметров PR: %v", err)
	}

	statusCode, body, err := g.sendReq(urlString, string(requestBody), "POST")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}

	// Обработка случая, когда PR уже существует
	if statusCode == http.StatusConflict {
		// Пробуем найти существующий PR с теми же head и base
		existingPR, findErr := g.findExistingPR(opts.Head, opts.Base)
		if findErr != nil {
			return nil, fmt.Errorf("PR уже существует, но не удалось его найти: %v", findErr)
		}
		return existingPR, nil
	}

	// Обработка ошибки "ветка не существует"
	if statusCode == http.StatusNotFound {
		return nil, fmt.Errorf("ветка не существует: head=%s или base=%s", opts.Head, opts.Base)
	}

	if statusCode != http.StatusCreated {
		return nil, fmt.Errorf("ошибка при создании PR: статус %d, ответ: %s", statusCode, body)
	}

	var prResp PRResponse
	if err := json.Unmarshal([]byte(body), &prResp); err != nil {
		return nil, fmt.Errorf("ошибка при разборе ответа: %v", err)
	}

	return &prResp, nil
}

// findExistingPR ищет существующий открытый PR с заданными head и base ветками.
func (g *API) findExistingPR(head, base string) (*PRResponse, error) {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/pulls?state=open", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo)

	statusCode, body, err := g.sendReq(urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении списка PR: %v", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении списка PR: статус %d", statusCode)
	}

	var prs []struct {
		ID      int64  `json:"id"`
		Number  int64  `json:"number"`
		HTMLURL string `json:"html_url"`
		State   string `json:"state"`
		Title   string `json:"title"`
		Body    string `json:"body"`
		Head    struct {
			Ref string `json:"ref"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
	}

	if err := json.Unmarshal([]byte(body), &prs); err != nil {
		return nil, fmt.Errorf("ошибка при разборе списка PR: %v", err)
	}

	for _, pr := range prs {
		if pr.Head.Ref == head && pr.Base.Ref == base {
			return &PRResponse{
				ID:      pr.ID,
				Number:  pr.Number,
				HTMLURL: pr.HTMLURL,
				State:   pr.State,
				Title:   pr.Title,
				Body:    pr.Body,
			}, nil
		}
	}

	return nil, fmt.Errorf("существующий PR с head=%s и base=%s не найден", head, base)
}

// ActivePR получает список активных запросов на изменение в репозитории.
// Возвращает все открытые запросы на изменение, ожидающие рецензирования
// или слияния.
// Возвращает:
//   - []PR: список активных запросов на изменение
//   - error: ошибка получения списка или nil при успехе
func (g *API) ActivePR() ([]PR, error) {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/pulls?state=open&sort=oldest", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo)
	statusCode, body, _ := g.sendReq(urlString, "", "GET")
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при запросе списка PR: %d", statusCode)
	}
	r := strings.NewReader(body)
	var prResp []PRData
	if err := json.NewDecoder(r).Decode(&prResp); err != nil {
		return nil, err
	}
	prs := []PR{}
	for _, pr := range prResp {
		if pr.Base.Label == g.BaseBranch {
			prs = append(prs, PR{pr.ID, pr.Number, pr.Base.Label, pr.Head.Label})
		}
	}
	return prs, nil
}

// ClosePR закрывает запрос на изменение без слияния.
// Отклоняет запрос на изменение, если изменения больше не нужны
// или не соответствуют требованиям проекта.
// Параметры:
//   - prNumber: номер запроса на изменение для закрытия
//
// Возвращает:
//   - error: ошибка закрытия или nil при успехе
func (g *API) ClosePR(prNumber int64) error {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/pulls/%d", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, prNumber)
	reqBody := `{"state":"closed"}`

	statusCode, _, err := g.sendReq(urlString, reqBody, "PATCH")
	if err != nil {
		return fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}
	if statusCode != http.StatusCreated {
		return fmt.Errorf("ошибка при закрытии PR: %d %d", prNumber, statusCode)
	}
	return nil
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

// IsUserInTeam проверяет членство пользователя в команде организации.
// Определяет, является ли указанный пользователь участником
// определенной команды в организации Gitea для контроля доступа.
// Параметры:
//   - l: логгер для записи отладочной информации
//   - username: имя пользователя для проверки
//   - orgName: имя организации
//   - teamName: имя команды
//
// Возвращает:
//   - bool: true если пользователь в команде, false если нет
//   - error: ошибка проверки или nil при успехе
func (g *API) IsUserInTeam(l *slog.Logger, username string, orgName string, teamName string) (bool, error) {
	// Сначала найдем команду по имени
	searchURL := fmt.Sprintf("%s/api/%s/orgs/%s/teams/search?q=%s", g.GiteaURL, constants.APIVersion, orgName, teamName)

	statusCode, body, err := g.sendReq(searchURL, "", "GET")
	if err != nil {
		return false, fmt.Errorf("ошибка при поиске команды: %v", err)
	}

	if statusCode != http.StatusOK {
		return false, fmt.Errorf("команда не найдена, статус код: %d", statusCode)
	}

	// Парсим ответ для получения ID команды
	var searchResult struct {
		Data []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}

	err = json.Unmarshal([]byte(body), &searchResult)
	if err != nil {
		return false, fmt.Errorf("ошибка парсинга ответа поиска команды: %v", err)
	}

	// Ищем команду с точным совпадением имени
	var teamID int64
	found := false
	for _, team := range searchResult.Data {
		if team.Name == teamName {
			teamID = team.ID
			found = true
			break
		}
	}

	if !found {
		l.Debug("Команда не найдена",
			slog.String("Команда", teamName),
			slog.String("Организация", orgName),
		)
		return false, nil
	}

	// Проверяем членство пользователя в команде
	memberURL := fmt.Sprintf("%s/api/%s/teams/%d/members/%s", g.GiteaURL, constants.APIVersion, teamID, username)

	statusCode, _, err = g.sendReq(memberURL, "", "GET")
	if err != nil {
		return false, fmt.Errorf("ошибка при проверке членства: %v", err)
	}

	l.Debug("Проверка членства пользователя в команде",
		slog.String("Пользователь", username),
		slog.String("Команда", teamName),
		slog.Int64("ID команды", teamID),
		slog.Int("Статус код", statusCode),
	)

	// Статус 204 означает, что пользователь является членом команды
	return statusCode == http.StatusOK, nil
}

// GetRepositoryContents получает содержимое директории репозитория.
// Возвращает список файлов и каталогов из указанного пути в репозитории
// для анализа структуры проекта или навигации по файлам.
// Параметры:
//   - filepath: путь к директории в репозитории
//   - branch: имя ветки для получения содержимого
//
// Возвращает:
//   - []FileInfo: список файлов и каталогов с метаданными
//   - error: ошибка получения содержимого или nil при успехе
func (g *API) GetRepositoryContents(filepath, branch string) ([]FileInfo, error) {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/contents/%s?ref=%s", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, filepath, branch)

	statusCode, body, err := g.sendReq(urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении содержимого репозитория: статус %d", statusCode)
	}

	var files []FileInfo
	err = json.Unmarshal([]byte(body), &files)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON: %v", err)
	}

	return files, nil
}

// Функции TestMerge и AnalyzeProject перенесены в сервисный слой (service/gitea_service.go)

// AnalyzeProjectStructure анализирует структуру проекта 1С.
// Определяет название основного проекта и список расширений конфигурации
// на основе структуры каталогов в репозитории.
// Параметры:
//   - branch: имя ветки для анализа
//
// Возвращает:
//   - []string: массив строк где первый элемент - название проекта, остальные - имена расширений
//   - error: ошибка анализа или nil при успехе
//
// analyzeProjectStructure анализирует структуру проекта 1С.
// Определяет название основного проекта и список расширений конфигурации
// на основе структуры каталогов в репозитории.
// Параметры:
//   - directories: список каталогов в репозитории
//
// Возвращает:
//   - []string: массив строк где первый элемент - название проекта, остальные - имена расширений
//   - error: ошибка анализа или nil при успехе
func analyzeProjectStructure(directories []string) ([]string, error) {
	if len(directories) == 0 {
		return []string{}, nil
	}

	// Находим каталоги без точки в середине
	var dirsWithoutDot []string
	for _, dir := range directories {
		if !strings.Contains(dir, ".") {
			dirsWithoutDot = append(dirsWithoutDot, dir)
		}
	}

	// Если нет каталогов без точки, возвращаем пустой результат
	if len(dirsWithoutDot) == 0 {
		return []string{}, nil
	}

	// Единственный каталог без точки - это название проекта
	projectName := dirsWithoutDot[0]

	// Ищем каталоги вида <projectName>.<расширение> (расширения конфигурации 1С)
	var extensions []string
	for _, dir := range directories {
		if strings.HasPrefix(dir, projectName+".") {
			ext := strings.TrimPrefix(dir, projectName+".")
			if ext != "" {
				extensions = append(extensions, ext)
			}
		}
	}

	// Если не найдены расширения и каталогов без точки больше одного - ошибка
	// (этот случай уже обработан выше, но добавляем для ясности логики)
	if len(extensions) == 0 && len(dirsWithoutDot) > 1 {
		return []string{}, fmt.Errorf("не найдены расширения конфигурации, а каталогов без точки больше одного: %v", dirsWithoutDot)
	}

	// Формируем результат: первый элемент - название проекта, остальные - расширения
	result := []string{projectName}
	result = append(result, extensions...)

	return result, nil
}

// AnalyzeProjectStructure анализирует структуру проекта 1С.
// Определяет название основного проекта и список расширений конфигурации
// на основе структуры каталогов в репозитории.
// Параметры:
//   - branch: имя ветки для анализа
//
// Возвращает:
//   - []string: массив строк где первый элемент - название проекта, остальные - имена расширений
//   - error: ошибка анализа или nil при успехе
func (g *API) AnalyzeProjectStructure(branch string) ([]string, error) {
	files, err := g.GetRepositoryContents(".", branch)
	if err != nil {
		return []string{}, err
	}

	// Получаем только каталоги, не начинающиеся с точки
	var directories []string
	for _, file := range files {
		if file.Type == "dir" && !strings.HasPrefix(file.Name, ".") {
			directories = append(directories, file.Name)
		}
	}

	return analyzeProjectStructure(directories)
}

// GetBranches получает список веток в репозитории.
// Возвращает массив структур Branch с информацией о ветках репозитория.
// Параметры:
//   - repo: имя репозитория
//
// Возвращает:
//   - []Branch: список веток в репозитории
//   - error: ошибка получения списка или nil при успехе
func (g *API) GetBranches(repo string) ([]Branch, error) {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/branches", g.GiteaURL, constants.APIVersion, g.Owner, repo)

	statusCode, body, err := g.sendReq(urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении веток: статус %d", statusCode)
	}

	var branches []Branch
	err = json.Unmarshal([]byte(body), &branches)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON: %v", err)
	}

	return branches, nil
}

// GetTeamMembers получает список членов команды в организации.
// Возвращает массив строк с именами пользователей, являющихся членами указанной команды.
// Параметры:
//   - orgName: имя организации
//   - teamName: имя команды
//
// Возвращает:
//   - []string: список имен пользователей
//   - error: ошибка получения списка или nil при успехе
func (g *API) GetTeamMembers(orgName, teamName string) ([]string, error) {
	// Сначала найдем команду по имени
	searchURL := fmt.Sprintf("%s/api/%s/orgs/%s/teams/search?q=%s", g.GiteaURL, constants.APIVersion, orgName, teamName)

	statusCode, body, err := g.sendReq(searchURL, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при поиске команды: %v", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("команда не найдена, статус код: %d", statusCode)
	}

	// Парсим ответ для получения ID команды
	var searchResult struct {
		Data []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}

	err = json.Unmarshal([]byte(body), &searchResult)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа поиска команды: %v", err)
	}

	// Ищем команду с точным совпадением имени
	var teamID int64
	found := false
	for _, team := range searchResult.Data {
		if team.Name == teamName {
			teamID = team.ID
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("команда %s не найдена в организации %s", teamName, orgName)
	}

	// Получаем членов команды
	membersURL := fmt.Sprintf("%s/api/%s/teams/%d/members", g.GiteaURL, constants.APIVersion, teamID)

	statusCode, body, err = g.sendReq(membersURL, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении членов команды: %v", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении членов команды, статус код: %d", statusCode)
	}

	// Парсим ответ для получения списка членов команды
	var members []struct {
		Login string `json:"login"`
	}

	err = json.Unmarshal([]byte(body), &members)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа членов команды: %v", err)
	}

	// Формируем список имен пользователей
	usernames := make([]string, 0, len(members))
	for _, member := range members {
		usernames = append(usernames, member.Login)
	}

	return usernames, nil
}

// GetLatestCommit получает информацию о последнем коммите в ветке.
// Возвращает метаданные самого свежего коммита для отслеживания
// последних изменений в указанной ветке репозитория.
// Параметры:
//   - branch: имя ветки для получения последнего коммита
//
// Возвращает:
//   - *Commit: указатель на структуру с информацией о коммите
//   - error: ошибка получения коммита или nil при успехе
func (g *API) GetLatestCommit(branch string) (*Commit, error) {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/commits?sha=%s&limit=1", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, branch)

	statusCode, body, err := g.sendReq(urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении коммитов: статус %d", statusCode)
	}

	var commits []Commit
	err = json.Unmarshal([]byte(body), &commits)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON: %v", err)
	}

	if len(commits) == 0 {
		return nil, fmt.Errorf("коммиты не найдены")
	}

	return &commits[0], nil
}

// GetLatestRelease получает информацию о последнем релизе репозитория.
// Возвращает метаданные последнего опубликованного релиза.
// Возвращает:
//   - *Release: указатель на структуру с информацией о релизе
//   - error: ошибка получения релиза или nil при успехе
func (g *API) GetLatestRelease() (*Release, error) {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/releases/latest", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo)

	statusCode, body, err := g.sendReq(urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}

	if statusCode == http.StatusNotFound {
		return nil, fmt.Errorf("релиз не найден")
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении релиза: статус %d", statusCode)
	}

	var release Release
	err = json.Unmarshal([]byte(body), &release)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON: %v", err)
	}

	return &release, nil
}

// GetReleaseByTag получает информацию о релизе по тегу.
// Возвращает метаданные релиза, связанного с указанным тегом.
// Параметры:
//   - tag: имя тега для поиска релиза
//
// Возвращает:
//   - *Release: указатель на структуру с информацией о релизе
//   - error: ошибка получения релиза или nil при успехе
func (g *API) GetReleaseByTag(tag string) (*Release, error) {
	// URL-кодируем тег для безопасной передачи в URL
	// Используем QueryEscape, так как PathEscape не кодирует символ /
	escapedTag := url.QueryEscape(tag)
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/releases/tags/%s", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, escapedTag)

	statusCode, body, err := g.sendReq(urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}

	if statusCode == http.StatusNotFound {
		return nil, fmt.Errorf("релиз с тегом '%s' не найден", tag)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении релиза по тегу '%s': статус %d", tag, statusCode)
	}

	var release Release
	err = json.Unmarshal([]byte(body), &release)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON: %v", err)
	}

	return &release, nil
}

// GetCommits получает список коммитов в ветке.
// Возвращает массив коммитов для указанной ветки.
// Параметры:
//   - branch: имя ветки для получения коммитов
//   - limit: максимальное количество коммитов для получения (0 - без ограничений)
//
// Возвращает:
//   - []Commit: массив коммитов
//   - error: ошибка получения коммитов или nil при успехе
func (g *API) GetCommits(branch string, limit int) ([]Commit, error) {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/commits?sha=%s", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, branch)
	if limit > 0 {
		urlString = fmt.Sprintf("%s&limit=%d", urlString, limit)
	}

	statusCode, body, err := g.sendReq(urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении коммитов: статус %d", statusCode)
	}

	var commits []Commit
	err = json.Unmarshal([]byte(body), &commits)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON: %v", err)
	}

	return commits, nil
}

// GetFirstCommitOfBranch получает первый коммит ветки.
// Возвращает первый коммит в указанной ветке, который не принадлежит базовой ветке.
// Параметры:
//   - branch: имя ветки для получения первого коммита
//   - _baseBranch: имя базовой ветки для сравнения (не используется)
//
// Возвращает:
//   - *Commit: указатель на первый коммит ветки
//   - error: ошибка получения коммита или nil при успехе
func (g *API) GetFirstCommitOfBranch(branch string, _ string) (*Commit, error) {
	// Получаем все коммиты в ветке в обратном порядке (от старых к новым)
	// Это может быть неэффективно для больших историй, но для наших целей подойдет
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/commits?sha=%s&stat=false&verification=false&files=false", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, branch)

	statusCode, body, err := g.sendReq(urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении коммитов: статус %d", statusCode)
	}

	var commits []Commit
	err = json.Unmarshal([]byte(body), &commits)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON: %v", err)
	}

	if len(commits) == 0 {
		return nil, fmt.Errorf("коммиты не найдены в ветке %s", branch)
	}

	// Возвращаем последний коммит из списка (он будет первым в истории ветки)
	// Это упрощенный подход. В реальном сценарии может потребоваться более сложная логика
	// для определения "первого" коммита ветки относительно базовой ветки.
	return &commits[len(commits)-1], nil
}

// GetCommitsBetween получает список коммитов между двумя коммитами.
// Возвращает массив коммитов между указанными SHA.
// Параметры:
//   - baseCommitSHA: SHA базового коммита
//   - headCommitSHA: SHA конечного коммита
//
// Возвращает:
//   - []Commit: массив коммитов между указанными коммитами
//   - error: ошибка получения коммитов или nil при успехе
func (g *API) GetCommitsBetween(baseCommitSHA, headCommitSHA string) ([]Commit, error) {
	// Gitea API не предоставляет прямого метода для получения коммитов между двумя SHA.
	// Мы можем попробовать использовать логику вида `git log base..head`, но через API это сложно.
	// Вместо этого мы получим коммиты от head и будем идти вниз по истории до base.
	// Это упрощенная реализация.

	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/commits?sha=%s", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, headCommitSHA)

	statusCode, body, err := g.sendReq(urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении коммитов: статус %d", statusCode)
	}

	var commits []Commit
	err = json.Unmarshal([]byte(body), &commits)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON: %v", err)
	}

	// Фильтруем коммиты до baseCommitSHA
	result := make([]Commit, 0, len(commits))
	foundBase := false
	for _, commit := range commits {
		if commit.SHA == baseCommitSHA {
			foundBase = true
			break // Не включаем base коммит
		}
		result = append(result, commit)
	}

	if !foundBase {
		// Если base коммит не найден, это может означать, что он не в истории head
		// или что история слишком длинная и не полностью загружена.
		// Возвращаем все найденные коммиты.
		// В реальном сценарии здесь может потребоваться более сложная логика.
		return commits, nil
	}

	return result, nil
}

// GetCommitFiles получает список файлов, измененных в коммите.
// Возвращает информацию о всех файлах, которые были добавлены,
// изменены или удалены в указанном коммите.
// Параметры:
//   - commitSHA: SHA хеш коммита для анализа
//
// Возвращает:
//   - []CommitFile: список файлов с информацией об изменениях
//   - error: ошибка получения файлов или nil при успехе
func (g *API) GetCommitFiles(commitSHA string) ([]CommitFile, error) {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/git/commits/%s", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, commitSHA)

	statusCode, body, err := g.sendReq(urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении информации о коммите: статус %d", statusCode)
	}

	var commitData struct {
		Files []CommitFile `json:"files"`
	}
	err = json.Unmarshal([]byte(body), &commitData)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON: %v", err)
	}

	return commitData.Files, nil
}

// SetRepositoryState устанавливает состояние файлов в репозитории.
// Выполняет множественные операции с файлами (создание, изменение, удаление)
// в рамках одного коммита для атомарного обновления состояния репозитория.
// Параметры:
//   - l: логгер для записи отладочной информации
//   - operations: массив операций для выполнения
//   - branch: имя ветки для применения изменений
//   - commitMessage: сообщение коммита
//
// Возвращает:
//   - error: ошибка выполнения операций или nil при успехе
func (g *API) SetRepositoryState(l *slog.Logger, operations []ChangeFileOperation, branch, commitMessage string) error {
	if len(operations) == 0 {
		return fmt.Errorf("список операций не может быть пустым")
	}

	// Создаем запрос для batch коммита
	request := ChangeFilesOptions{
		Branch: branch,
		Author: &Identity{
			Name:  constants.DefaultCommitAuthorName,
			Email: constants.DefaultCommitAuthorEmail,
		},
		Committer: &Identity{
			Name:  constants.DefaultCommitAuthorName,
			Email: constants.DefaultCommitAuthorEmail,
		},
		Message: commitMessage,
		Files:   operations,
	}

	// Сериализуем запрос в JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("ошибка при сериализации запроса: %v", err)
	}

	// Формируем URL для batch API
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/contents", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo)

	l.Debug("Выполнение batch запроса", "url", urlString, "body", string(requestBody))
	// Выполняем POST запрос
	statusCode, responseBody, err := g.sendReq(urlString, string(requestBody), "POST")
	if err != nil {
		return fmt.Errorf("ошибка при выполнении batch запроса: %v", err)
	}

	// Проверяем статус ответа
	if statusCode != http.StatusOK && statusCode != http.StatusCreated {
		return fmt.Errorf("ошибка при выполнении batch операций: статус %d, ответ: %s", statusCode, responseBody)
	}

	return nil
}

// SetRepositoryStateWithNewBranch выполняет множественные операции с файлами и создаёт новую ветку.
// Аналогичен SetRepositoryState, но дополнительно создаёт новую ветку от указанной базовой ветки.
// Параметры:
//   - l: логгер для записи отладочной информации
//   - operations: массив операций для выполнения
//   - baseBranch: базовая ветка (откуда создаётся новая)
//   - newBranch: имя новой ветки для создания
//   - commitMessage: сообщение коммита
//
// Возвращает:
//   - string: SHA созданного коммита
//   - error: ошибка выполнения операций или nil при успехе
func (g *API) SetRepositoryStateWithNewBranch(l *slog.Logger, operations []ChangeFileOperation, baseBranch, newBranch, commitMessage string) (string, error) {
	if len(operations) == 0 {
		return "", fmt.Errorf("список операций не может быть пустым")
	}

	// Валидация операций - проверяем на пустые пути
	for i, op := range operations {
		if op.Path == "" {
			l.Error("SetRepositoryStateWithNewBranch: обнаружена операция с пустым путём",
				slog.Int("index", i),
				slog.String("operation", op.Operation),
				slog.String("sha", op.SHA),
			)
			return "", fmt.Errorf("операция %d имеет пустой путь (operation=%s)", i, op.Operation)
		}
	}

	// Логируем информацию о запросе
	l.Debug("SetRepositoryStateWithNewBranch: подготовка batch запроса",
		slog.Int("operations_count", len(operations)),
		slog.String("baseBranch", baseBranch),
		slog.String("newBranch", newBranch),
		slog.String("repo", fmt.Sprintf("%s/%s", g.Owner, g.Repo)),
	)

	// Создаем запрос для batch коммита с новой веткой
	request := ChangeFilesOptions{
		Branch:    baseBranch,
		NewBranch: newBranch,
		Author: &Identity{
			Name:  constants.DefaultCommitAuthorName,
			Email: constants.DefaultCommitAuthorEmail,
		},
		Committer: &Identity{
			Name:  constants.DefaultCommitAuthorName,
			Email: constants.DefaultCommitAuthorEmail,
		},
		Message: commitMessage,
		Files:   operations,
	}

	// Сериализуем запрос в JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("ошибка при сериализации запроса: %v", err)
	}

	// Формируем URL для batch API
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/contents", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo)

	// Логируем пути операций для диагностики (без содержимого файлов)
	var operationPaths []string
	for _, op := range operations {
		operationPaths = append(operationPaths, fmt.Sprintf("%s:%s", op.Operation, op.Path))
	}
	l.Debug("SetRepositoryStateWithNewBranch: пути операций",
		slog.Any("operations", operationPaths),
	)

	l.Debug("Выполнение batch запроса с новой веткой", "url", urlString, "newBranch", newBranch)
	// Выполняем POST запрос
	statusCode, responseBody, err := g.sendReq(urlString, string(requestBody), "POST")
	if err != nil {
		return "", fmt.Errorf("ошибка при выполнении batch запроса: %v", err)
	}

	// Проверяем статус ответа
	if statusCode != http.StatusOK && statusCode != http.StatusCreated {
		// Логируем детали ошибки для диагностики
		l.Error("SetRepositoryStateWithNewBranch: ошибка batch операций",
			slog.Int("status_code", statusCode),
			slog.String("response", responseBody),
			slog.String("repo", fmt.Sprintf("%s/%s", g.Owner, g.Repo)),
			slog.String("newBranch", newBranch),
			slog.Int("operations_count", len(operations)),
		)
		return "", fmt.Errorf("ошибка при выполнении batch операций: статус %d, ответ: %s", statusCode, responseBody)
	}

	// Парсим ответ для получения SHA коммита
	var response struct {
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
	}
	if err := json.Unmarshal([]byte(responseBody), &response); err != nil {
		// Если не удалось распарсить, возвращаем пустой SHA, но без ошибки
		l.Warn("Не удалось получить SHA коммита из ответа", "error", err)
		return "", nil
	}

	return response.Commit.SHA, nil
}

// GetConfigData получает данные конфигурации из файла в репозитории.
// Загружает и декодирует содержимое конфигурационного файла для
// использования в настройках приложения или процессах развертывания.
// Параметры:
//   - l: логгер для записи отладочной информации
//   - filename: имя файла конфигурации или URL для загрузки
//
// Возвращает:
//   - []byte: содержимое файла в виде массива байт
//   - error: ошибка получения данных или nil при успехе
func (g *API) GetConfigData(l *slog.Logger, filename string) ([]byte, error) {
	var fileURL string

	l.Debug("GetConfigData started",
		"filename", filename,
		"giteaURL", g.GiteaURL,
		"owner", g.Owner,
		"repo", g.Repo,
		"baseBranch", g.BaseBranch,
		"hasAccessToken", g.AccessToken != "")

	// Determine source based on filename prefix
	if strings.HasPrefix(filename, "https://") {
		l.Debug("Using direct URL", "url", filename)
		fileURL = filename
	} else {
		// Repository files - use current owner/repo
		// Build URL for file content API
		fileURL = fmt.Sprintf("%s/api/v1/repos/%s/%s/contents/%s?ref=%s", g.GiteaURL, g.Owner, g.Repo, filename, g.BaseBranch)
		l.Debug("Built repository file URL",
			"owner", g.Owner,
			"repo", g.Repo,
			"filename", filename,
			"baseBranch", g.BaseBranch,
			"fileURL", fileURL)
	}

	// Make HTTP request
	l.Debug("Creating HTTP request", "fileURL", fileURL, "filename", filename)
	statusCode, respBody, err := g.sendReq(fileURL, "", "GET")
	if err != nil {
		l.Error("Failed to create HTTP request",
			"filename", filename,
			"fileURL", fileURL,
			"error", err)
		return nil, fmt.Errorf("failed to create request for file %s: %w", filename, err)
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP ошибка %d при загрузке %s", statusCode, filename)
	}

	l.Debug("Reading response body")
	body, err := io.ReadAll(strings.NewReader(respBody))
	if err != nil {
		l.Error("Failed to read response body",
			"filename", filename,
			"error", err)
		return nil, fmt.Errorf("failed to read response for file %s: %w", filename, err)
	}

	// Parse JSON response to get content
	var fileData struct {
		Content string `json:"content"`
	}

	l.Debug("Parsing JSON response")
	if unmarshalErr := json.Unmarshal(body, &fileData); unmarshalErr != nil {
		l.Error("Failed to parse JSON response",
			"filename", filename,
			"error", unmarshalErr,
			"responseBody", string(body))
		return nil, fmt.Errorf("failed to parse response for file %s: %w", filename, unmarshalErr)
	}

	l.Debug("JSON response parsed",
		"contentLength", len(fileData.Content),
		"filename", filename)

	// Decode base64 content
	l.Debug("Decoding base64 content")
	content, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(fileData.Content, "\n", ""))
	if err != nil {
		l.Error("Failed to decode base64 content",
			"filename", filename,
			"error", err)
		return nil, fmt.Errorf("failed to decode content for file %s: %w", filename, err)
	}

	l.Debug("Base64 content decoded",
		"decodedLength", len(content),
		"filename", filename)

	// Check if data is empty
	if len(content) == 0 {
		l.Error("Decoded content is empty", "filename", filename)
		return nil, fmt.Errorf("data for file %s is empty", filename)
	}

	l.Debug("GetConfigData completed successfully",
		"filename", filename,
		"finalContentLength", len(content))

	return content, nil
}

// GetConfigDataBad получает данные конфигурации по префиксу имени файла.
// Устаревший метод для поиска и загрузки конфигурационных файлов
// по префиксу имени в корневой директории репозитория.
// Параметры:
//   - filenamePrefix: префикс имени файла для поиска
//
// Возвращает:
//   - []byte: содержимое найденного файла
//   - error: ошибка поиска или загрузки файла или nil при успехе
func (g *API) GetConfigDataBad(filenamePrefix string) ([]byte, error) {
	// Если это прямая ссылка, загружаем напрямую
	if strings.HasPrefix(filenamePrefix, "http://") || strings.HasPrefix(filenamePrefix, "https://") {
		statusCode, body, err := g.sendReq(filenamePrefix, "", "GET")

		if err != nil {
			return nil, fmt.Errorf("ошибка загрузки по URL %s: %v", filenamePrefix, err)
		}

		if statusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP ошибка %d при загрузке %s", statusCode, filenamePrefix)
		}

		return []byte(body), nil
	}

	// Получаем содержимое корневой директории репозитория
	contents, err := g.GetRepositoryContents("", g.BaseBranch)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения содержимого репозитория: %v", err)
	}

	// Ищем файл с нужным префиксом
	for _, file := range contents {
		if file.Type == "file" && strings.HasPrefix(file.Name, filenamePrefix) {
			// Получаем содержимое файла
			urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/contents/%s?ref=%s",
				g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, file.Path, g.BaseBranch)

			statusCode, body, err := g.sendReq(urlString, "", "GET")
			if err != nil {
				return nil, fmt.Errorf("ошибка выполнения запроса: %v", err)
			}

			if statusCode != http.StatusOK {
				return nil, fmt.Errorf("ошибка API: %d - %s", statusCode, body)
			}

			r := strings.NewReader(body)
			var fileInfo FileInfo
			if err := json.NewDecoder(r).Decode(&fileInfo); err != nil {
				return nil, fmt.Errorf("ошибка декодирования JSON: %v", err)
			}

			// Декодируем содержимое из base64
			if fileInfo.Encoding == "base64" {
				content := strings.ReplaceAll(fileInfo.Content, "\n", "")
				decodedBytes, err := base64.StdEncoding.DecodeString(content)
				if err != nil {
					return nil, fmt.Errorf("ошибка декодирования base64: %v", err)
				}
				return decodedBytes, nil
			}

			return []byte(fileInfo.Content), nil
		}
	}

	return nil, fmt.Errorf("файл с префиксом '%s' не найден", filenamePrefix)
}

// AnalyzeProject анализирует структуру проекта 1С в репозитории.
// Определяет название основного проекта и список расширений конфигурации
// на основе анализа структуры каталогов в корне репозитория.
// Логика анализа:
// 1. Получает список каталогов в корне репозитория (исключая скрытые)
// 2. Ищет расширения конфигурации 1С (каталоги вида <projectName>.<расширение>)
// 3. Если каталогов без точки больше одного - возвращает ошибку
// 4. Если только один каталог без точки - возвращает массив с названием проекта и расширениями
// 5. Если не найдены расширения, но каталогов без точки больше одного - возвращает ошибку
// Параметры:
//   - branch: имя ветки для анализа
//
// Возвращает:
//   - []string: массив где первый элемент - название проекта, остальные - расширения
//   - error: ошибка анализа или nil при успехе
//
// AnalyzeProject анализирует структуру проекта 1С в репозитории.
// Определяет название основного проекта и список расширений конфигурации
// на основе анализа структуры каталогов в корне репозитория.
// Логика анализа:
// 1. Получает список каталогов в корне репозитория (исключая скрытые)
// 2. Ищем расширения конфигурации 1С (каталоги вида <projectName>.<расширение>)
// 3. Если каталогов без точки больше одного - возвращает ошибку
// 4. Если только один каталог без точки - возвращает массив с названием проекта и расширениями
// 5. Если не найдены расширения, но каталогов без точки больше одного - возвращает ошибку
// Параметры:
//   - branch: имя ветки для анализа
//
// Возвращает:
//   - []string: массив где первый элемент - название проекта, остальные - расширения
//   - error: ошибка анализа или nil при успехе
func (g *API) AnalyzeProject(branch string) ([]string, error) {
	files, err := g.GetRepositoryContents("", branch)
	if err != nil {
		return []string{}, err
	}

	// Получаем только каталоги, не начинающиеся с точки
	var directories []string
	for _, file := range files {
		if file.Type == "dir" && !strings.HasPrefix(file.Name, ".") {
			directories = append(directories, file.Name)
		}
	}

	return analyzeProjectStructure(directories)
}

// GetBranchCommitRange получает первый и последний коммит в ветке.
// Для веток кроме main получает первый и последний коммит согласно логике ветвления.
// Для ветки main:
//   - для первого коммита: если есть коммит с тегом "sq-start", то берет его, иначе первый коммит
//   - для последнего коммита: берет последний коммит
//
// Коммиты располагаются в строгом порядке от старого к новому.
//
// Параметры:
//   - branch: имя ветки для анализа
//
// Возвращает:
//   - *BranchCommitRange: структура с первым и последним коммитом
//   - error: ошибка получения коммитов или nil при успехе
func (g *API) GetBranchCommitRange(branch string) (*BranchCommitRange, error) {
	if branch == "main" || branch == "master" {
		return g.getMainBranchCommitRange(branch)
	}
	return g.getFeatureBranchCommitRange(branch)
}

// getMainBranchCommitRange получает диапазон коммитов для главной ветки.
// Для первого коммита ищет коммит с тегом "sq-start", если не найден - берет первый коммит.
// Для последнего коммита берет последний коммит в ветке.
func (g *API) getMainBranchCommitRange(branch string) (*BranchCommitRange, error) {
	// Получаем последний коммит
	lastCommit, err := g.GetLatestCommit(branch)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения последнего коммита: %v", err)
	}

	// Ищем коммит с тегом "sq-start"
	firstCommit, err := g.findCommitWithTag(branch, "sq-start")
	if err != nil {
		// Если коммит с тегом не найден, берем первый коммит в истории
		firstCommit, err = g.getFirstCommitInHistory(branch)
		if err != nil {
			return nil, fmt.Errorf("ошибка получения первого коммита: %v", err)
		}
	}

	return &BranchCommitRange{
		FirstCommit: firstCommit,
		LastCommit:  lastCommit,
	}, nil
}

// getFeatureBranchCommitRange получает диапазон коммитов для feature ветки.
// Использует compare API для определения первого коммита (merge base) и последнего коммита.
func (g *API) getFeatureBranchCommitRange(branch string) (*BranchCommitRange, error) {
	// Получаем последний коммит ветки
	lastCommit, err := g.GetLatestCommit(branch)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения последнего коммита: %v", err)
	}

	// Сравниваем с базовой веткой для получения merge base
	baseBranch := g.BaseBranch
	if baseBranch == "" {
		baseBranch = "main"
	}

	compareResult, err := g.compareBranches(baseBranch, branch)
	if err != nil {
		return nil, fmt.Errorf("ошибка сравнения веток: %v", err)
	}

	// Если общий предок не найден, используем первый коммит базовой ветки
	var firstCommit *Commit
	if compareResult.MergeBaseCommit == nil {
		// Fallback: используем первый коммит базовой ветки
		firstCommit, err = g.getFirstCommitInHistory(baseBranch)
		if err != nil {
			return nil, fmt.Errorf("не удалось получить первый коммит базовой ветки %s: %v", baseBranch, err)
		}
	} else {
		firstCommit = compareResult.MergeBaseCommit
	}

	return &BranchCommitRange{
		FirstCommit: firstCommit,
		LastCommit:  lastCommit,
	}, nil
}

// findCommitWithTag ищет коммит с указанным тегом в ветке.
// Возвращает коммит, если тег найден, иначе возвращает ошибку.
func (g *API) findCommitWithTag(_, tag string) (*Commit, error) {
	// Получаем список тегов
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/tags", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo)

	statusCode, body, err := g.sendReq(urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса тегов: %v", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении тегов: статус %d", statusCode)
	}

	var tags []struct {
		Name   string `json:"name"`
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
	}

	err = json.Unmarshal([]byte(body), &tags)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON тегов: %v", err)
	}

	// Ищем тег "sq-start"
	for _, t := range tags {
		if t.Name == tag {
			// Получаем полную информацию о коммите
			return g.getCommitBySHA(t.Commit.SHA)
		}
	}

	return nil, fmt.Errorf("тег %s не найден", tag)
}

// getFirstCommitInHistory получает самый первый коммит в истории ветки.
func (g *API) getFirstCommitInHistory(branch string) (*Commit, error) {
	// Получаем все коммиты в ветке
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/commits?sha=%s&stat=false&verification=false&files=false", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, branch)

	statusCode, body, err := g.sendReq(urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении коммитов: статус %d", statusCode)
	}

	var commits []Commit
	err = json.Unmarshal([]byte(body), &commits)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON: %v", err)
	}

	if len(commits) == 0 {
		return nil, fmt.Errorf("коммиты не найдены в ветке %s", branch)
	}

	// Возвращаем последний коммит из списка (самый старый в истории)
	return &commits[len(commits)-1], nil
}

// compareBranches сравнивает две ветки и возвращает результат сравнения.
// Использует Gitea API для получения общего предка и списка коммитов.
// Гарантирует, что MergeBaseCommit будет найден и возвращен.
func (g *API) compareBranches(base, head string) (*CompareResult, error) {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/compare/%s...%s", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, base, head)

	statusCode, body, err := g.sendReq(urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса сравнения: %v", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при сравнении веток: статус %d", statusCode)
	}

	var result CompareResult
	err = json.Unmarshal([]byte(body), &result)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON результата сравнения: %v", err)
	}

	// Если общий предок не найден через API сравнения, попробуем найти его вручную
	if result.MergeBaseCommit == nil {
		mergeBase, err := g.findMergeBase(base, head)
		if err != nil {
			return nil, fmt.Errorf("не удалось найти общего предка веток %s и %s: %v", base, head, err)
		}
		result.MergeBaseCommit = mergeBase
	}

	return &result, nil
}

// findMergeBase находит коммит в базовой ветке, предшествующий первому коммиту ветки head.
// Определяет состояние базовой ветки до внесения первого изменения в head ветку.
// Возвращает коммит из base ветки, который был актуален до создания head ветки.
func (g *API) findMergeBase(base, head string) (*Commit, error) {
	// Получаем историю коммитов целевой ветки (head)
	headCommits, err := g.getAllCommits(head)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить коммиты ветки head %s: %v", head, err)
	}

	if len(headCommits) == 0 {
		return nil, fmt.Errorf("ветка head %s не содержит коммитов", head)
	}

	// Находим самый первый (самый старый) коммит в head ветке
	// Коммиты в массиве упорядочены от новых к старым, поэтому берем последний элемент
	firstHeadCommit := &headCommits[len(headCommits)-1]

	// Получаем историю коммитов базовой ветки
	baseCommits, err := g.getAllCommits(base)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить коммиты базовой ветки %s: %v", base, err)
	}

	if len(baseCommits) == 0 {
		return nil, fmt.Errorf("базовая ветка %s не содержит коммитов", base)
	}

	// Ищем позицию первого коммита head ветки в базовой ветке
	var baseCommitIndex = -1
	for i, baseCommit := range baseCommits {
		if baseCommit.SHA == firstHeadCommit.SHA {
			baseCommitIndex = i
			break
		}
	}

	// Если первый коммит head ветки не найден в базовой ветке,
	// возвращаем последний коммит базовой ветки
	if baseCommitIndex == -1 {
		return &baseCommits[0], nil // Первый элемент = самый новый коммит
	}

	// Если первый коммит head ветки - это самый первый коммит в базовой ветке,
	// то предыдущего коммита нет
	if baseCommitIndex == len(baseCommits)-1 {
		return nil, fmt.Errorf("первый коммит ветки head %s является первым коммитом в базовой ветке %s", head, base)
	}

	// Возвращаем предыдущий коммит в базовой ветке
	return &baseCommits[baseCommitIndex+1], nil
}

// getAllCommits получает все коммиты ветки.
func (g *API) getAllCommits(branch string) ([]Commit, error) {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/commits?sha=%s", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, branch)

	statusCode, body, err := g.sendReq(urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса коммитов: %v", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении коммитов ветки %s: статус %d", branch, statusCode)
	}

	var commits []Commit
	err = json.Unmarshal([]byte(body), &commits)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON коммитов: %v", err)
	}

	return commits, nil
}

// getCommitBySHA получает информацию о коммите по его SHA.
func (g *API) getCommitBySHA(sha string) (*Commit, error) {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/commits?sha=%s", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, sha)

	statusCode, body, err := g.sendReq(urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса коммита: %v", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении коммита: статус %d", statusCode)
	}

	var commits []Commit
	err = json.Unmarshal([]byte(body), &commits)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON коммита: %v", err)
	}

	if len(commits) == 0 {
		return nil, fmt.Errorf("коммит с SHA %s не найден", sha)
	}

	// Ищем коммит с точным SHA в массиве
	for _, commit := range commits {
		if commit.SHA == sha {
			return &commit, nil
		}
	}

	return nil, fmt.Errorf("коммит с SHA %s не найден", sha)
}

// SearchOrgRepos получает все репозитории организации.
// Выполняет автоматическую обработку пагинации для получения полного списка.
// Параметры:
//   - orgName: имя организации для поиска репозиториев
//
// Возвращает:
//   - []Repository: список всех репозиториев организации
//   - error: ошибка получения репозиториев или nil при успехе
//
// Особенности:
//   - Автоматически обрабатывает пагинацию (лимит 50 на страницу)
//   - Защита от бесконечного цикла (максимум 100 страниц = 5000 репозиториев)
//   - Возвращает пустой slice если организация не найдена (HTTP 404)
func (g *API) SearchOrgRepos(orgName string) ([]Repository, error) {
	var allRepos []Repository

	for page := 1; page <= SearchOrgReposMaxPages; page++ {
		urlString := fmt.Sprintf("%s/api/%s/orgs/%s/repos?page=%d&limit=%d",
			g.GiteaURL, constants.APIVersion, orgName, page, SearchOrgReposPageLimit)

		statusCode, body, err := g.sendReq(urlString, "", "GET")
		if err != nil {
			return nil, fmt.Errorf("ошибка при запросе репозиториев организации %s: %v", orgName, err)
		}

		// Организация не найдена — возвращаем пустой slice
		if statusCode == http.StatusNotFound {
			return []Repository{}, nil
		}

		if statusCode != http.StatusOK {
			return nil, fmt.Errorf("ошибка при получении репозиториев организации %s: статус %d", orgName, statusCode)
		}

		var repos []Repository
		if err := json.Unmarshal([]byte(body), &repos); err != nil {
			return nil, fmt.Errorf("ошибка при разборе JSON репозиториев: %v", err)
		}

		// Пустой ответ — достигли конца списка
		if len(repos) == 0 {
			break
		}

		allRepos = append(allRepos, repos...)
	}

	return allRepos, nil
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

// GetUserOrganizations получает список всех организаций, к которым имеет доступ текущий пользователь.
// Использует токен доступа для аутентификации.
//
// Возвращает:
//   - []Organization: список организаций пользователя
//   - error: ошибка получения или nil при успехе
//
// Особенности:
//   - Автоматически обрабатывает пагинацию (лимит 50 на страницу)
//   - Защита от бесконечного цикла (максимум 100 страниц)
func (g *API) GetUserOrganizations() ([]Organization, error) {
	var allOrgs []Organization

	for page := 1; page <= GetUserOrgsMaxPages; page++ {
		urlString := fmt.Sprintf("%s/api/%s/user/orgs?page=%d&limit=%d",
			g.GiteaURL, constants.APIVersion, page, GetUserOrgsPageLimit)

		statusCode, body, err := g.sendReq(urlString, "", "GET")
		if err != nil {
			return nil, fmt.Errorf("ошибка при запросе организаций пользователя: %v", err)
		}

		if statusCode != http.StatusOK {
			return nil, fmt.Errorf("ошибка при получении организаций: статус %d", statusCode)
		}

		var orgs []Organization
		if err := json.Unmarshal([]byte(body), &orgs); err != nil {
			return nil, fmt.Errorf("ошибка при разборе JSON организаций: %v", err)
		}

		// Пустой ответ — достигли конца списка
		if len(orgs) == 0 {
			break
		}

		allOrgs = append(allOrgs, orgs...)
	}

	return allOrgs, nil
}

// HasBranch проверяет существование ветки в указанном репозитории.
// Использует API для получения информации о конкретной ветке.
//
// Параметры:
//   - owner: владелец репозитория (организация или пользователь)
//   - repo: имя репозитория
//   - branchName: имя ветки для проверки
//
// Возвращает:
//   - bool: true если ветка существует, false если нет
//   - error: ошибка при выполнении запроса или nil при успехе
func (g *API) HasBranch(owner, repo, branchName string) (bool, error) {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/branches/%s",
		g.GiteaURL, constants.APIVersion, owner, repo, branchName)

	statusCode, _, err := g.sendReq(urlString, "", "GET")
	if err != nil {
		return false, fmt.Errorf("ошибка при проверке ветки %s в %s/%s: %v", branchName, owner, repo, err)
	}

	// Ветка существует
	if statusCode == http.StatusOK {
		return true, nil
	}

	// Ветка не найдена
	if statusCode == http.StatusNotFound {
		return false, nil
	}

	return false, fmt.Errorf("неожиданный статус при проверке ветки: %d", statusCode)
}
