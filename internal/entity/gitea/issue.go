package gitea

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Kargones/apk-ci/internal/constants"
)

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
		return nil, fmt.Errorf("ошибка при выполнении запроса: %w", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении задачи %d: статус %d", issueNumber, statusCode)
	}

	r := strings.NewReader(body)
	var issue Issue
	if err := json.NewDecoder(r).Decode(&issue); err != nil {
		return nil, fmt.Errorf("ошибка при декодировании ответа: %w", err)
	}

	return &issue, nil
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
		return fmt.Errorf("ошибка при выполнении запроса: %w", err)
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
		return fmt.Errorf("ошибка при выполнении запроса: %w", err)
	}

	if statusCode != http.StatusCreated {
		return fmt.Errorf("ошибка при закрытии задачи %d: статус %d", issueNumber, statusCode)
	}

	return nil
}
