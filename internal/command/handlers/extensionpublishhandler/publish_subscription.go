package extensionpublishhandler

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/Kargones/apk-ci/internal/entity/gitea"
	"gopkg.in/yaml.v3"
)

// ProjectYAML представляет структуру файла project.yaml в целевом репозитории.
// Используется для определения подписок на расширения.
type ProjectYAML struct {
	// Subscriptions — список подписок в формате {Org}_{Repo}_{ExtDir}
	// Пример: ["lib_ssl_апкБСП", "lib_common_cfe_utils"]
	Subscriptions []string `yaml:"subscriptions"`
}

// GetProjectSubscriptions читает файл project.yaml из репозитория и возвращает список подписок.
// Если файл не существует или секция subscriptions отсутствует — возвращает пустой список.
//
// Параметры:
//   - api: клиент Gitea API для целевого репозитория
//   - branch: ветка для чтения файла
//
// Возвращает:
//   - []string: список подписок в формате {Org}_{Repo}_{ExtDir}
//   - error: ошибка при чтении/парсинге или nil при успехе
func GetProjectSubscriptions(api *gitea.API, branch string) ([]string, error) {
	// Читаем содержимое project.yaml
	content, err := api.GetFileContent("project.yaml")
	if err != nil {
		// Если файл не существует — это не ошибка, просто нет подписок
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "статус 404") {
			return []string{}, nil
		}
		return nil, fmt.Errorf("ошибка чтения project.yaml: %w", err)
	}

	// Парсим YAML
	var projectYAML ProjectYAML
	if err := yaml.Unmarshal(content, &projectYAML); err != nil {
		return nil, fmt.Errorf("ошибка парсинга project.yaml: %w", err)
	}

	// Если секция subscriptions отсутствует или пуста — возвращаем пустой список
	if projectYAML.Subscriptions == nil {
		return []string{}, nil
	}

	return projectYAML.Subscriptions, nil
}

// SubscribedRepo представляет репозиторий, подписанный на обновления расширения.
// Структура содержит информацию о целевом репозитории, куда будет опубликовано расширение.
//
// Механизм подписки работает через файл project.yaml в целевом репозитории:
// - Секция subscriptions содержит список подписок в формате {Org}_{Repo}_{ExtDir}
// - Пример: lib_ssl_апкБСП -> организация lib, репозиторий ssl, каталог апкБСП
type SubscribedRepo struct {
	// Organization — имя организации целевого репозитория
	Organization string `json:"organization"`

	// Repository — имя целевого репозитория
	Repository string `json:"repository"`

	// TargetBranch — ветка по умолчанию целевого репозитория (main/master)
	TargetBranch string `json:"target_branch,omitempty"`

	// TargetDirectory — каталог для размещения расширения в целевом репозитории
	TargetDirectory string `json:"target_directory"`

	// SubscriptionID — идентификатор подписки из project.yaml (для отладки и логирования)
	SubscriptionID string `json:"-"`
}

// ParseSubscriptionID парсит идентификатор подписки и извлекает информацию об источнике.
// Формат подписки: {Org}_{Repo}_{ExtDir}
// - Org: имя организации исходного репозитория
// - Repo: имя исходного репозитория
// - ExtDir: путь к каталогу расширения (символы _ заменяются на /)
//
// Примеры:
//   - "lib_ssl_апкБСП" -> Org=lib, Repo=ssl, Dir=апкБСП
//   - "lib_common_cfe_utils" -> Org=lib, Repo=common, Dir=cfe/utils
//
// Параметры:
//   - subscriptionID: идентификатор подписки для парсинга
//
// Возвращает:
//   - org: имя организации
//   - repo: имя репозитория
//   - extDir: путь к каталогу расширения
//   - error: ошибка парсинга или nil при успехе
func ParseSubscriptionID(subscriptionID string) (org, repo, extDir string, err error) {
	// Разделяем на 3 части: Org, Repo и ExtDir (все остальное)
	parts := strings.SplitN(subscriptionID, "_", 3)
	if len(parts) < 3 {
		return "", "", "", fmt.Errorf("некорректный формат подписки: %s (ожидается {Org}_{Repo}_{ExtDir})", subscriptionID)
	}

	org = parts[0]
	repo = parts[1]
	// Заменяем _ на / в ExtDir для поддержки вложенных каталогов
	extDir = strings.ReplaceAll(parts[2], "_", "/")

	// Проверяем, что все компоненты не пустые
	if org == "" || repo == "" || extDir == "" {
		return "", "", "", fmt.Errorf("пустые компоненты в подписке: %s", subscriptionID)
	}

	return org, repo, extDir, nil
}

// FindSubscribedRepos находит все репозитории, подписанные на обновления из указанного источника.
// Функция ищет репозитории с подписками в файле project.yaml.
//
// Механизм работы:
// 1. Формирует идентификаторы подписок: {org}_{repo}_{extDir} для каждого расширения
// 2. Получает список всех доступных организаций
// 3. Для каждой организации получает список репозиториев
// 4. Для каждого репозитория читает project.yaml и проверяет секцию subscriptions
// 5. Если идентификатор подписки найден в списке — репозиторий является подписчиком
//
// Параметры:
//   - l: логгер для записи информации о процессе поиска
//   - api: клиент Gitea API для выполнения запросов
//   - sourceRepo: имя исходного репозитория
//   - extensions: список расширений (директорий) для поиска подписчиков
//
// Возвращает:
//   - []SubscribedRepo: список валидных подписчиков
//   - error: ошибка при выполнении операций с API или nil при успехе
func FindSubscribedRepos(l *slog.Logger, api *gitea.API, sourceRepo string, extensions []string) ([]SubscribedRepo, error) {
	logger := l

	// Если расширения не указаны, возвращаем пустой список
	if len(extensions) == 0 {
		logger.Info("расширения не указаны, подписчики не найдены")
		return []SubscribedRepo{}, nil
	}

	// Формируем идентификаторы подписок: {org}_{repo}_{extDir}
	// Где org — организация текущего репозитория, repo — имя текущего репозитория
	subscriptionIDs := make(map[string]string) // subscriptionID -> extDir
	for _, ext := range extensions {
		// Заменяем / на _ в пути расширения для формирования идентификатора
		extDir := strings.ReplaceAll(ext, "/", "_")
		subscriptionID := fmt.Sprintf("%s_%s_%s", api.Owner, sourceRepo, extDir)
		subscriptionIDs[subscriptionID] = ext
	}

	logger.Info("сформированы идентификаторы подписок",
		slog.Any("subscription_ids", subscriptionIDs),
	)

	// Получаем список всех доступных организаций
	orgs, err := api.GetUserOrganizations()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения списка организаций: %w", err)
	}

	logger.Info("найдено организаций",
		slog.Int("count", len(orgs)),
	)

	var subscribers []SubscribedRepo

	// Для каждой организации получаем список репозиториев
	for _, org := range orgs {
		// Получаем репозитории организации
		repos, err := api.SearchOrgRepos(org.Username)
		if err != nil {
			// Логируем ошибку, но продолжаем обработку других организаций
			logger.Warn("ошибка получения репозиториев организации",
				slog.String("organization", org.Username),
				slog.String("error", err.Error()),
			)
			continue
		}

		// Для каждого репозитория проверяем наличие подписок в project.yaml
		for _, repo := range repos {
			// Создаём временный API для целевого репозитория
			targetAPI := gitea.NewGiteaAPI(gitea.Config{
				GiteaURL:    api.GiteaURL,
				Owner:       org.Username,
				Repo:        repo.Name,
				AccessToken: api.AccessToken,
				BaseBranch:  repo.DefaultBranch,
			})

			// Читаем подписки из project.yaml
			subscriptions, err := GetProjectSubscriptions(targetAPI, repo.DefaultBranch)
			if err != nil {
				logger.Warn("ошибка чтения подписок репозитория",
					slog.String("organization", org.Username),
					slog.String("repository", repo.Name),
					slog.String("error", err.Error()),
				)
				continue
			}

			// Проверяем, есть ли наши подписки в списке
			for _, sub := range subscriptions {
				if extDir, found := subscriptionIDs[sub]; found {
					// Репозиторий подписан на обновления
					subscriber := SubscribedRepo{
						Organization:    org.Username,
						Repository:      repo.Name,
						TargetBranch:    repo.DefaultBranch,
						TargetDirectory: extDir,
						SubscriptionID:  sub,
					}
					subscribers = append(subscribers, subscriber)

					logger.Info("найден подписчик",
						slog.String("organization", org.Username),
						slog.String("repository", repo.Name),
						slog.String("subscription_id", sub),
						slog.String("target_directory", extDir),
					)
				}
			}
		}
	}

	return subscribers, nil
}
