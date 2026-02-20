package gitea

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Kargones/apk-ci/internal/constants"
)

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
func (g *API) SearchOrgRepos(ctx context.Context, orgName string) ([]Repository, error) {
	var allRepos []Repository

	for page := 1; page <= SearchOrgReposMaxPages; page++ {
		urlString := fmt.Sprintf("%s/api/%s/orgs/%s/repos?page=%d&limit=%d",
			g.GiteaURL, constants.APIVersion, orgName, page, SearchOrgReposPageLimit)

		statusCode, body, err := g.sendReq(ctx, urlString, "", "GET")
		if err != nil {
			return nil, fmt.Errorf("ошибка при запросе репозиториев организации %s: %w", orgName, err)
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
			return nil, fmt.Errorf("ошибка при разборе JSON репозиториев: %w", err)
		}

		// Пустой ответ — достигли конца списка
		if len(repos) == 0 {
			break
		}

		allRepos = append(allRepos, repos...)
	}

	return allRepos, nil
}

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
func (g *API) GetUserOrganizations(ctx context.Context) ([]Organization, error) {
	var allOrgs []Organization

	for page := 1; page <= GetUserOrgsMaxPages; page++ {
		urlString := fmt.Sprintf("%s/api/%s/user/orgs?page=%d&limit=%d",
			g.GiteaURL, constants.APIVersion, page, GetUserOrgsPageLimit)

		statusCode, body, err := g.sendReq(ctx, urlString, "", "GET")
		if err != nil {
			return nil, fmt.Errorf("ошибка при запросе организаций пользователя: %w", err)
		}

		if statusCode != http.StatusOK {
			return nil, fmt.Errorf("ошибка при получении организаций: статус %d", statusCode)
		}

		var orgs []Organization
		if err := json.Unmarshal([]byte(body), &orgs); err != nil {
			return nil, fmt.Errorf("ошибка при разборе JSON организаций: %w", err)
		}

		// Пустой ответ — достигли конца списка
		if len(orgs) == 0 {
			break
		}

		allOrgs = append(allOrgs, orgs...)
	}

	return allOrgs, nil
}
