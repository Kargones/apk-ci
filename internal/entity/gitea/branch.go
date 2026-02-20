package gitea

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Kargones/apk-ci/internal/constants"
)

// CreateTestBranch создает новую тестовую ветку в репозитории Gitea.
// Создает ветку на основе указанной базовой ветки для выполнения тестирования
// или разработки новых функций без влияния на основную ветку.
// Возвращает:
//   - error: ошибка создания ветки или nil при успехе
func (g *API) CreateTestBranch(ctx context.Context) error {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/branches", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo)
	reqBody := fmt.Sprintf(`{
"new_branch_name": "%s",
"old_branch_name": "%s",
"old_ref_name": "refs/heads/%s"
}`, g.NewBranch, g.BaseBranch, g.BaseBranch)

	statusCode, _, err := g.sendReq(ctx, urlString, reqBody, "POST")

	if statusCode != http.StatusCreated {
		return fmt.Errorf("ошибка при создании ветки: %v %w", statusCode, err)
	}
	return nil
}

// DeleteTestBranch удаляет тестовую ветку из репозитория Gitea.
// Очищает временные ветки после завершения тестирования или разработки
// для поддержания чистоты репозитория.
// Возвращает:
//   - error: ошибка удаления ветки или nil при успехе
func (g *API) DeleteTestBranch(ctx context.Context) error {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/branches/%s", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, g.NewBranch)

	statusCode, _, err := g.sendReq(ctx, urlString, "", "DELETE")
	if statusCode != http.StatusNoContent {
		return fmt.Errorf("ошибка при создании ветки: %v %w", statusCode, err)
	}
	return nil
}

// GetBranches получает список веток в репозитории.
// Возвращает массив структур Branch с информацией о ветках репозитория.
// Параметры:
//   - repo: имя репозитория
//
// Возвращает:
//   - []Branch: список веток в репозитории
//   - error: ошибка получения списка или nil при успехе
func (g *API) GetBranches(ctx context.Context, repo string) ([]Branch, error) {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/branches", g.GiteaURL, constants.APIVersion, g.Owner, repo)

	statusCode, body, err := g.sendReq(ctx, urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %w", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении веток: статус %d", statusCode)
	}

	var branches []Branch
	err = json.Unmarshal([]byte(body), &branches)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON: %w", err)
	}

	return branches, nil
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
func (g *API) HasBranch(ctx context.Context, owner, repo, branchName string) (bool, error) {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/branches/%s",
		g.GiteaURL, constants.APIVersion, owner, repo, branchName)

	statusCode, _, err := g.sendReq(ctx, urlString, "", "GET")
	if err != nil {
		return false, fmt.Errorf("ошибка при проверке ветки %s в %s/%s: %w", branchName, owner, repo, err)
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
