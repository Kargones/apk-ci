package gitea

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Kargones/apk-ci/internal/constants"
)

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
