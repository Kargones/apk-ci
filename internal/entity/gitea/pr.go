package gitea

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/constants"
)

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
