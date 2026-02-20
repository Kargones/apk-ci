package gitea

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/Kargones/apk-ci/internal/constants"
)

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
func (g *API) IsUserInTeam(ctx context.Context, l *slog.Logger, username string, orgName string, teamName string) (bool, error) {
	// Сначала найдем команду по имени
	searchURL := fmt.Sprintf("%s/api/%s/orgs/%s/teams/search?q=%s", g.GiteaURL, constants.APIVersion, orgName, teamName)

	statusCode, body, err := g.sendReq(ctx, searchURL, "", "GET")
	if err != nil {
		return false, fmt.Errorf("ошибка при поиске команды: %w", err)
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
		return false, fmt.Errorf("ошибка парсинга ответа поиска команды: %w", err)
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

	statusCode, _, err = g.sendReq(ctx, memberURL, "", "GET")
	if err != nil {
		return false, fmt.Errorf("ошибка при проверке членства: %w", err)
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

// GetTeamMembers получает список членов команды в организации.
// Возвращает массив строк с именами пользователей, являющихся членами указанной команды.
// Параметры:
//   - orgName: имя организации
//   - teamName: имя команды
//
// Возвращает:
//   - []string: список имен пользователей
//   - error: ошибка получения списка или nil при успехе
func (g *API) GetTeamMembers(ctx context.Context, orgName, teamName string) ([]string, error) {
	// Сначала найдем команду по имени
	searchURL := fmt.Sprintf("%s/api/%s/orgs/%s/teams/search?q=%s", g.GiteaURL, constants.APIVersion, orgName, teamName)

	statusCode, body, err := g.sendReq(ctx, searchURL, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при поиске команды: %w", err)
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
		return nil, fmt.Errorf("ошибка парсинга ответа поиска команды: %w", err)
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

	statusCode, body, err = g.sendReq(ctx, membersURL, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении членов команды: %w", err)
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
		return nil, fmt.Errorf("ошибка парсинга ответа членов команды: %w", err)
	}

	// Формируем список имен пользователей
	usernames := make([]string, 0, len(members))
	for _, member := range members {
		usernames = append(usernames, member.Login)
	}

	return usernames, nil
}
