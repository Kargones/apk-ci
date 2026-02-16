// Package app содержит логику для публикации расширений 1C.
// Реализует механизм поиска подписанных репозиториев через файл project.yaml.
package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
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

// SyncResult представляет результат синхронизации расширения в целевой репозиторий.
// Содержит информацию о выполненных операциях, созданной ветке и возможных ошибках.
//
// Поля:
//   - Subscriber: целевой репозиторий-подписчик
//   - FilesCreated: количество файлов, созданных в целевом каталоге
//   - FilesDeleted: количество файлов, удалённых из целевого каталога
//   - NewBranch: имя созданной ветки для изменений (формат: update-{extName}-{version})
//   - CommitSHA: SHA хеш коммита с изменениями
//   - Error: ошибка синхронизации (nil при успехе)
type SyncResult struct {
	// Subscriber — целевой репозиторий-подписчик
	Subscriber SubscribedRepo

	// FilesCreated — количество созданных файлов
	FilesCreated int

	// FilesDeleted — количество удалённых файлов
	FilesDeleted int

	// NewBranch — имя созданной ветки
	NewBranch string

	// CommitSHA — SHA коммита с изменениями
	CommitSHA string

	// Error — ошибка синхронизации (nil при успехе)
	Error error
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

// GetSourceFiles рекурсивно получает все файлы из исходного каталога репозитория
// и формирует список операций "create" для каждого файла.
//
// Функция выполняет:
// 1. Получение списка файлов и подкаталогов в указанном каталоге
// 2. Для каждого файла: получение содержимого и формирование операции create
// 3. Для каждого подкаталога: рекурсивный вызов для обработки вложенных файлов
//
// Параметры:
//   - api: клиент Gitea API (использует api.Owner и api.Repo)
//   - sourceDir: путь к исходному каталогу в репозитории
//   - branch: имя ветки для получения содержимого
//
// Возвращает:
//   - []gitea.ChangeFileOperation: список операций create с относительными путями
//   - error: ошибка при получении содержимого или nil при успехе
func GetSourceFiles(api *gitea.API, sourceDir, branch string) ([]gitea.ChangeFileOperation, error) {
	operations, err := getSourceFilesRecursive(api, sourceDir, "", branch)
	if err != nil {
		return nil, err
	}

	// AC5: Пустой исходный каталог -> error
	if len(operations) == 0 {
		return nil, fmt.Errorf("исходный каталог %s пустой или не содержит файлов", sourceDir)
	}

	return operations, nil
}

// GetTargetFilesToDelete получает список файлов в целевом каталоге и формирует операции удаления.
// Функция рекурсивно обходит указанный каталог и создаёт операции "delete" для всех файлов.
//
// Особенности:
//   - Несуществующий или пустой каталог не является ошибкой (возвращает пустой slice)
//   - Для каждой операции delete требуется SHA файла
//   - Полный путь файла сохраняется в операции для точного удаления
//
// Параметры:
//   - api: клиент Gitea API (использует api.Owner и api.Repo)
//   - targetDir: путь к целевому каталогу в репозитории
//   - branch: имя ветки для получения содержимого
//
// Возвращает:
//   - []gitea.ChangeFileOperation: список операций delete с полными путями и SHA
//   - error: ошибка при получении содержимого или nil при успехе
func GetTargetFilesToDelete(api *gitea.API, targetDir, branch string) ([]gitea.ChangeFileOperation, error) {
	return getTargetFilesRecursive(api, targetDir, branch)
}

// getTargetFilesRecursive рекурсивно обходит целевой каталог и собирает операции удаления.
func getTargetFilesRecursive(api *gitea.API, currentDir, branch string) ([]gitea.ChangeFileOperation, error) {
	// Получаем содержимое текущего каталога
	contents, err := api.GetRepositoryContents(currentDir, branch)
	if err != nil {
		// Несуществующий каталог не является ошибкой - просто нечего удалять
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "статус 404") {
			return []gitea.ChangeFileOperation{}, nil
		}
		return nil, fmt.Errorf("ошибка получения содержимого каталога %s: %w", currentDir, err)
	}

	var operations []gitea.ChangeFileOperation

	for _, item := range contents {
		switch item.Type {
		case "dir":
			// Рекурсивный вызов для подкаталога
			// Используем path.Join вместо filepath.Join для кросс-платформенной совместимости
			// (Gitea API всегда ожидает Unix-пути с прямыми слешами)
			subOps, err := getTargetFilesRecursive(
				api,
				path.Join(currentDir, item.Name),
				branch,
			)
			if err != nil {
				return nil, err
			}
			operations = append(operations, subOps...)

		case "file":
			// Формируем операцию delete с полным путём и SHA
			operations = append(operations, gitea.ChangeFileOperation{
				Operation: "delete",
				Path:      item.Path,
				SHA:       item.SHA,
			})
		}
	}

	return operations, nil
}

// GetTargetFilesMap возвращает map существующих файлов в целевом каталоге.
// Ключ - относительный путь от targetDir, значение - SHA файла.
// Используется для определения операции (create/update) при синхронизации.
//
// Параметры:
//   - api: клиент Gitea API
//   - targetDir: путь к целевому каталогу в репозитории
//   - branch: имя ветки для получения содержимого
//
// Возвращает:
//   - map[string]string: карта путей к SHA файлов
//   - error: ошибка при получении содержимого или nil при успехе
func GetTargetFilesMap(api *gitea.API, targetDir, branch string) (map[string]string, error) {
	return getTargetFilesMapRecursive(api, targetDir, targetDir, branch)
}

// getTargetFilesMapRecursive рекурсивно обходит каталог и собирает карту файлов.
// baseDir - базовый каталог для вычисления относительных путей.
func getTargetFilesMapRecursive(api *gitea.API, currentDir, baseDir, branch string) (map[string]string, error) {
	contents, err := api.GetRepositoryContents(currentDir, branch)
	if err != nil {
		// Несуществующий каталог не является ошибкой - просто пустая карта
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "статус 404") {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("ошибка получения содержимого каталога %s: %w", currentDir, err)
	}

	result := make(map[string]string)

	for _, item := range contents {
		switch item.Type {
		case "dir":
			// Рекурсивный вызов для подкаталога
			subMap, err := getTargetFilesMapRecursive(
				api,
				path.Join(currentDir, item.Name),
				baseDir,
				branch,
			)
			if err != nil {
				return nil, err
			}
			// Объединяем карты
			for k, v := range subMap {
				result[k] = v
			}

		case "file":
			// Вычисляем относительный путь от baseDir
			relativePath := strings.TrimPrefix(item.Path, baseDir+"/")
			result[relativePath] = item.SHA
		}
	}

	return result, nil
}

// GenerateBranchName генерирует имя ветки для обновления расширения.
// Формат: update-{extName}-{version}
// - extName приводится к нижнему регистру
// - пробелы заменяются на дефисы
// - префикс "v" удаляется из версии
//
// Параметры:
//   - extName: имя расширения
//   - version: версия расширения
//
// Возвращает:
//   - string: имя ветки в формате update-{extname}-{version}
func GenerateBranchName(extName, version string) string {
	// Приводим имя к нижнему регистру и заменяем пробелы на дефисы
	normalizedName := strings.ToLower(extName)
	normalizedName = strings.ReplaceAll(normalizedName, " ", "-")

	// Удаляем префикс "v" из версии
	normalizedVersion := strings.TrimPrefix(version, "v")

	return fmt.Sprintf("update-%s-%s", normalizedName, normalizedVersion)
}

// GenerateCommitMessage генерирует сообщение коммита для обновления расширения.
// Формат: chore(ext): update {extName} to {version}
//
// Параметры:
//   - extName: имя расширения
//   - version: версия расширения
//
// Возвращает:
//   - string: сообщение коммита
func GenerateCommitMessage(extName, version string) string {
	return fmt.Sprintf("chore(ext): update %s to %s", extName, version)
}

// SyncExtensionToRepo синхронизирует файлы расширения из исходного репозитория
// в целевой репозиторий подписчика.
//
// Процесс синхронизации:
// 1. Получает файлы из исходного каталога
// 2. Получает карту существующих файлов в целевом каталоге (путь -> SHA)
// 3. Генерирует имя новой ветки и commit message
// 4. Для каждого исходного файла определяет операцию (create/update)
// 5. Добавляет операции delete для файлов, которые есть только в целевом каталоге
// 6. Выполняет batch commit с созданием новой ветки
//
// Параметры:
//   - l: логгер для записи информации о процессе синхронизации
//   - sourceAPI: API для исходного репозитория (откуда берутся файлы)
//   - targetAPI: API для целевого репозитория (куда записываются файлы)
//   - subscriber: информация о подписчике
//   - sourceDir: путь к каталогу с файлами расширения в исходном репо
//   - sourceBranch: ветка исходного репозитория для получения файлов
//   - targetDir: путь к каталогу для размещения расширения в целевом репо
//   - extName: имя расширения
//   - version: версия расширения
//
// Возвращает:
//   - *SyncResult: результат синхронизации
//   - error: критическая ошибка или nil при успехе
func SyncExtensionToRepo(l *slog.Logger, sourceAPI, targetAPI *gitea.API, subscriber SubscribedRepo, sourceDir, sourceBranch, targetDir, extName, version string) (*SyncResult, error) {
	logger := l

	result := &SyncResult{
		Subscriber: subscriber,
	}

	// Логируем входные параметры для диагностики
	logger.Debug("SyncExtensionToRepo: начало синхронизации",
		slog.String("sourceDir", sourceDir),
		slog.String("sourceBranch", sourceBranch),
		slog.String("targetDir", targetDir),
		slog.String("extName", extName),
		slog.String("version", version),
	)

	// 1. Получаем файлы из исходного каталога
	sourceOps, err := GetSourceFiles(sourceAPI, sourceDir, sourceBranch)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения исходных файлов: %w", err)
	}

	// Логируем исходные файлы для диагностики
	logger.Debug("SyncExtensionToRepo: получено исходных файлов",
		slog.Int("count", len(sourceOps)),
	)

	// 2. Получаем карту существующих файлов в целевом каталоге (путь -> SHA)
	targetFilesMap, err := GetTargetFilesMap(targetAPI, targetDir, subscriber.TargetBranch)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения файлов целевого каталога: %w", err)
	}

	// Логируем целевые файлы для диагностики
	logger.Debug("SyncExtensionToRepo: получено целевых файлов",
		slog.Int("count", len(targetFilesMap)),
	)
	if len(targetFilesMap) > 0 {
		var targetPaths []string
		for p := range targetFilesMap {
			targetPaths = append(targetPaths, p)
			if len(targetPaths) >= 5 {
				break
			}
		}
		logger.Debug("SyncExtensionToRepo: примеры путей в целевом каталоге",
			slog.Any("sample_paths", targetPaths),
		)
	}

	// 3. Генерируем имя ветки и commit message
	branchName := GenerateBranchName(extName, version)
	commitMessage := GenerateCommitMessage(extName, version)

	result.NewBranch = branchName

	// 4. Формируем операции на основе сравнения источника и цели
	var allOperations []gitea.ChangeFileOperation
	sourcePathsSet := make(map[string]struct{})

	// Для каждого исходного файла определяем операцию (create/update)
	for _, op := range sourceOps {
		// Проверка на пустой путь - защита от некорректных данных
		if op.Path == "" {
			logger.Warn("SyncExtensionToRepo: обнаружена операция с пустым путём, пропускаем",
				slog.String("operation", op.Operation),
			)
			continue
		}

		targetPath := path.Join(targetDir, op.Path)
		sourcePathsSet[op.Path] = struct{}{}

		if sha, exists := targetFilesMap[op.Path]; exists {
			// Файл существует в целевом каталоге - используем update
			allOperations = append(allOperations, gitea.ChangeFileOperation{
				Operation: "update",
				Path:      targetPath,
				Content:   op.Content,
				SHA:       sha,
			})
		} else {
			// Файл не существует - используем create
			allOperations = append(allOperations, gitea.ChangeFileOperation{
				Operation: "create",
				Path:      targetPath,
				Content:   op.Content,
			})
		}
	}

	// 5. Добавляем операции delete для файлов, которые есть только в целевом каталоге
	var filesDeleted int
	for targetPath, sha := range targetFilesMap {
		// Проверка на пустой путь - защита от некорректных данных
		if targetPath == "" {
			logger.Warn("SyncExtensionToRepo: обнаружен пустой путь в targetFilesMap, пропускаем")
			continue
		}

		if _, exists := sourcePathsSet[targetPath]; !exists {
			// Файл есть в целевом, но нет в исходном - удаляем
			fullPath := path.Join(targetDir, targetPath)
			allOperations = append(allOperations, gitea.ChangeFileOperation{
				Operation: "delete",
				Path:      fullPath,
				SHA:       sha,
			})
			filesDeleted++
		}
	}

	result.FilesCreated = len(sourceOps)
	result.FilesDeleted = filesDeleted

	// Логируем информацию об операциях для диагностики
	logger.Debug("SyncExtensionToRepo: сформированы операции",
		slog.Int("total_operations", len(allOperations)),
		slog.Int("source_files", len(sourceOps)),
		slog.Int("target_files", len(targetFilesMap)),
		slog.Int("files_to_delete", filesDeleted),
	)

	// Логируем первые несколько путей для диагностики
	if len(allOperations) > 0 {
		var samplePaths []string
		maxSamples := 5
		if len(allOperations) < maxSamples {
			maxSamples = len(allOperations)
		}
		for i := 0; i < maxSamples; i++ {
			samplePaths = append(samplePaths, allOperations[i].Path)
		}
		logger.Debug("SyncExtensionToRepo: примеры путей операций",
			slog.Any("sample_paths", samplePaths),
		)
	}

	// 6. Выполняем batch commit с созданием новой ветки
	commitSHA, err := targetAPI.SetRepositoryStateWithNewBranch(
		logger,
		allOperations,
		subscriber.TargetBranch,
		branchName,
		commitMessage,
	)
	if err != nil {
		result.Error = fmt.Errorf("ошибка выполнения коммита: %w", err)
		return result, nil
	}

	result.CommitSHA = commitSHA

	return result, nil
}

// BuildExtensionPRBody формирует markdown описание для PR обновления расширения.
// Включает версию, release notes и ссылку на релиз.
// Параметры:
//   - release: информация о релизе (может быть nil)
//   - sourceRepo: полное имя исходного репозитория (org/repo)
//   - extName: имя расширения
//   - releaseURL: URL релиза в исходном репозитории
//
// Возвращает:
//   - string: markdown форматированное описание PR
func BuildExtensionPRBody(release *gitea.Release, sourceRepo, extName, releaseURL string) string {
	var sb strings.Builder

	sb.WriteString("## Extension Update\n\n")
	sb.WriteString(fmt.Sprintf("**Extension:** %s\n", extName))

	// Добавляем версию из release (если есть)
	if release != nil && release.TagName != "" {
		sb.WriteString(fmt.Sprintf("**Version:** %s\n", release.TagName))
	}

	// Добавляем ссылку на источник
	if releaseURL != "" {
		sb.WriteString(fmt.Sprintf("**Source:** [%s](%s)\n", sourceRepo, releaseURL))
	} else if sourceRepo != "" {
		sb.WriteString(fmt.Sprintf("**Source:** %s\n", sourceRepo))
	}

	sb.WriteString("\n### Release Notes\n\n")

	// Добавляем release notes (если есть)
	if release != nil && release.Body != "" {
		// Экранируем специальные символы если необходимо
		sb.WriteString(release.Body)
	} else {
		sb.WriteString("_No release notes provided._")
	}

	sb.WriteString("\n\n---\n")
	sb.WriteString("*This PR was automatically created by benadis-runner extension-publish*\n")

	return sb.String()
}

// BuildExtensionPRTitle формирует заголовок PR для обновления расширения.
// Формат: "Update {extName} to {version}"
// Параметры:
//   - extName: имя расширения
//   - version: версия расширения
//
// Возвращает:
//   - string: заголовок PR
func BuildExtensionPRTitle(extName, version string) string {
	return fmt.Sprintf("Update %s to %s", extName, version)
}

// CreateExtensionPR создает Pull Request для обновления расширения.
// Формирует заголовок и описание на основе информации о релизе и синхронизации.
// Параметры:
//   - l: логгер для записи информации о создании PR
//   - api: клиент Gitea API для целевого репозитория
//   - syncResult: результат синхронизации файлов
//   - release: информация о релизе (может быть nil)
//   - extName: имя расширения
//   - sourceRepo: полное имя исходного репозитория
//   - releaseURL: URL релиза в исходном репозитории
//
// Возвращает:
//   - *gitea.PRResponse: информация о созданном PR
//   - error: ошибка создания или nil при успехе
func CreateExtensionPR(l *slog.Logger, api *gitea.API, syncResult *SyncResult, release *gitea.Release, extName, sourceRepo, releaseURL string) (*gitea.PRResponse, error) {
	logger := l

	// Определяем версию для заголовка
	var version string
	if release != nil && release.TagName != "" {
		version = release.TagName
	} else {
		// Если релиз не указан, извлекаем версию из имени ветки
		// Формат ветки: update-{extname}-{version}
		parts := strings.Split(syncResult.NewBranch, "-")
		if len(parts) >= 3 {
			version = parts[len(parts)-1]
		} else {
			version = "unknown"
		}
	}

	// Формируем заголовок и описание PR
	title := BuildExtensionPRTitle(extName, version)
	body := BuildExtensionPRBody(release, sourceRepo, extName, releaseURL)

	// Создаём опции для PR
	opts := gitea.CreatePROptions{
		Title: title,
		Body:  body,
		Head:  syncResult.NewBranch,
		Base:  syncResult.Subscriber.TargetBranch,
	}

	logger.Info("Создание PR для обновления расширения",
		slog.String("extension", extName),
		slog.String("version", version),
		slog.String("head", opts.Head),
		slog.String("base", opts.Base),
	)

	// Создаём PR через API
	pr, err := api.CreatePRWithOptions(opts)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания PR: %w", err)
	}

	logger.Info("PR успешно создан",
		slog.Int64("pr_number", pr.Number),
		slog.String("url", pr.HTMLURL),
	)

	return pr, nil
}

// getSourceFilesRecursive рекурсивно обходит каталог и собирает файлы.
// basePath используется для формирования относительных путей файлов.
func getSourceFilesRecursive(api *gitea.API, currentDir, basePath, branch string) ([]gitea.ChangeFileOperation, error) {
	// Получаем содержимое текущего каталога
	contents, err := api.GetRepositoryContents(currentDir, branch)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения содержимого каталога %s: %w", currentDir, err)
	}

	var operations []gitea.ChangeFileOperation

	for _, item := range contents {
		// Формируем относительный путь для файла
		// Используем path.Join для кросс-платформенной совместимости (Gitea API ожидает Unix-пути)
		relativePath := item.Name
		if basePath != "" {
			relativePath = path.Join(basePath, item.Name)
		}

		switch item.Type {
		case "dir":
			// Рекурсивный вызов для подкаталога
			subOps, err := getSourceFilesRecursive(
				api,
				path.Join(currentDir, item.Name),
				relativePath,
				branch,
			)
			if err != nil {
				return nil, err
			}
			operations = append(operations, subOps...)

		case "file":
			// Получаем содержимое файла (GetFileContent возвращает уже декодированные байты)
			content, err := api.GetFileContent(item.Path)
			if err != nil {
				return nil, fmt.Errorf("ошибка получения файла %s: %w", item.Path, err)
			}

			// Кодируем содержимое в base64 для передачи в Gitea API
			// (API SetRepositoryState ожидает base64-encoded content)
			encodedContent := base64.StdEncoding.EncodeToString(content)

			operations = append(operations, gitea.ChangeFileOperation{
				Operation: "create",
				Path:      relativePath,
				Content:   encodedContent,
			})
		}
	}

	return operations, nil
}

// PublishStatus представляет статус публикации расширения в репозиторий.
type PublishStatus string

const (
	// StatusSuccess — успешная публикация (PR создан)
	StatusSuccess PublishStatus = "success"
	// StatusFailed — ошибка публикации
	StatusFailed PublishStatus = "failed"
	// StatusSkipped — пропущено (репозиторий недоступен, dry-run, уже обновлён)
	StatusSkipped PublishStatus = "skipped"
)

// PublishResult представляет результат публикации расширения для одного подписчика.
// Содержит информацию о репозитории подписчика, созданном PR и возможных ошибках.
type PublishResult struct {
	// Subscriber — целевой репозиторий подписчика
	Subscriber SubscribedRepo `json:"subscriber"`

	// Status — статус публикации (success/failed/skipped)
	Status PublishStatus `json:"status"`

	// SyncResult — результат синхронизации файлов (не сериализуется в JSON)
	SyncResult *SyncResult `json:"-"`

	// PRNumber — номер созданного PR (0 если PR не создан)
	PRNumber int `json:"pr_number,omitempty"`

	// PRURL — URL созданного PR
	PRURL string `json:"pr_url,omitempty"`

	// Error — ошибка при публикации (не сериализуется в JSON)
	Error error `json:"-"`

	// ErrorMessage — человекочитаемое описание ошибки (для JSON)
	ErrorMessage string `json:"error,omitempty"`

	// DurationMs — время выполнения операции в миллисекундах
	DurationMs int64 `json:"duration_ms"`
}

// PublishReport представляет полный отчёт о публикации расширения.
// Содержит информацию об источнике, времени выполнения и результатах для каждого подписчика.
type PublishReport struct {
	// ExtensionName — имя публикуемого расширения
	ExtensionName string `json:"extension_name"`

	// Version — версия расширения
	Version string `json:"version"`

	// SourceRepo — полное имя исходного репозитория (owner/repo)
	SourceRepo string `json:"source_repo"`

	// StartTime — время начала публикации
	StartTime time.Time `json:"start_time"`

	// EndTime — время завершения публикации
	EndTime time.Time `json:"end_time"`

	// Results — результаты для каждого подписчика
	Results []PublishResult `json:"results"`
}

// SuccessCount возвращает количество успешных публикаций.
func (r *PublishReport) SuccessCount() int {
	count := 0
	for _, res := range r.Results {
		if res.Status == StatusSuccess {
			count++
		}
	}
	return count
}

// FailedCount возвращает количество неудачных публикаций.
func (r *PublishReport) FailedCount() int {
	count := 0
	for _, res := range r.Results {
		if res.Status == StatusFailed {
			count++
		}
	}
	return count
}

// SkippedCount возвращает количество пропущенных публикаций.
func (r *PublishReport) SkippedCount() int {
	count := 0
	for _, res := range r.Results {
		if res.Status == StatusSkipped {
			count++
		}
	}
	return count
}

// HasErrors возвращает true, если есть хотя бы одна неудачная публикация.
func (r *PublishReport) HasErrors() bool {
	return r.FailedCount() > 0
}

// TotalDuration возвращает общую длительность операции.
func (r *PublishReport) TotalDuration() time.Duration {
	return r.EndTime.Sub(r.StartTime)
}

// ReportJSONOutput структура для JSON-сериализации отчёта с summary.
type ReportJSONOutput struct {
	ExtensionName string          `json:"extension_name"`
	Version       string          `json:"version"`
	SourceRepo    string          `json:"source_repo"`
	StartTime     time.Time       `json:"start_time"`
	EndTime       time.Time       `json:"end_time"`
	Results       []PublishResult `json:"results"`
	Summary       ReportSummary   `json:"summary"`
}

// ReportSummary содержит итоговую статистику для JSON-вывода.
type ReportSummary struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

// ReportResults выводит структурированный отчёт о публикации.
// При BR_OUTPUT_JSON=true выводит JSON в stdout, иначе — форматированный текст в лог.
//
// Параметры:
//   - report: отчёт о публикации
//   - l: логгер для текстового вывода
//
// Возвращает:
//   - error: ошибка сериализации JSON или nil при успехе
func ReportResults(report *PublishReport, l *slog.Logger) error {
	outputJSON := os.Getenv("BR_OUTPUT_JSON") == "true"

	if outputJSON {
		return reportResultsJSON(report)
	}

	reportResultsText(report, l)
	return nil
}

// reportResultsJSON выводит отчёт в формате JSON в stdout.
func reportResultsJSON(report *PublishReport) error {
	output := ReportJSONOutput{
		ExtensionName: report.ExtensionName,
		Version:       report.Version,
		SourceRepo:    report.SourceRepo,
		StartTime:     report.StartTime,
		EndTime:       report.EndTime,
		Results:       report.Results,
		Summary: ReportSummary{
			Total:   len(report.Results),
			Success: report.SuccessCount(),
			Failed:  report.FailedCount(),
			Skipped: report.SkippedCount(),
		},
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// reportResultsText выводит форматированный текстовый отчёт в лог.
func reportResultsText(report *PublishReport, l *slog.Logger) {
	// Заголовок
	l.Info("═══════════════════════════════════════════════════════════════")
	l.Info("               EXTENSION PUBLISH REPORT")
	l.Info("═══════════════════════════════════════════════════════════════")
	l.Info(fmt.Sprintf("Extension: %s", report.ExtensionName))
	l.Info(fmt.Sprintf("Version:   %s", report.Version))
	l.Info(fmt.Sprintf("Source:    %s", report.SourceRepo))
	l.Info(fmt.Sprintf("Duration:  %.1fs", report.TotalDuration().Seconds()))
	l.Info("")

	// Успешные публикации
	successCount := report.SuccessCount()
	if successCount > 0 {
		l.Info("─────────────────────────────────────────────────────────────")
		l.Info(fmt.Sprintf("✓ SUCCESS (%d)", successCount))
		l.Info("─────────────────────────────────────────────────────────────")
		for _, res := range report.Results {
			if res.Status == StatusSuccess {
				target := fmt.Sprintf("%s/%s", res.Subscriber.Organization, res.Subscriber.Repository)
				l.Info(fmt.Sprintf("  • %s → PR #%d (%s)", target, res.PRNumber, res.PRURL))
			}
		}
		l.Info("")
	}

	// Неудачные публикации
	failedCount := report.FailedCount()
	if failedCount > 0 {
		l.Info("─────────────────────────────────────────────────────────────")
		l.Info(fmt.Sprintf("✗ FAILED (%d)", failedCount))
		l.Info("─────────────────────────────────────────────────────────────")
		for _, res := range report.Results {
			if res.Status == StatusFailed {
				target := fmt.Sprintf("%s/%s", res.Subscriber.Organization, res.Subscriber.Repository)
				l.Info(fmt.Sprintf("  • %s: %s", target, res.ErrorMessage))
			}
		}
		l.Info("")
	}

	// Пропущенные публикации
	skippedCount := report.SkippedCount()
	if skippedCount > 0 {
		l.Info("─────────────────────────────────────────────────────────────")
		l.Info(fmt.Sprintf("○ SKIPPED (%d)", skippedCount))
		l.Info("─────────────────────────────────────────────────────────────")
		for _, res := range report.Results {
			if res.Status == StatusSkipped {
				target := fmt.Sprintf("%s/%s", res.Subscriber.Organization, res.Subscriber.Repository)
				reason := res.ErrorMessage
				if reason == "" {
					reason = "dry-run mode"
				}
				l.Info(fmt.Sprintf("  • %s: %s", target, reason))
			}
		}
		l.Info("")
	}

	// Итоговая статистика
	l.Info("═══════════════════════════════════════════════════════════════")
	l.Info(fmt.Sprintf("SUMMARY: %d success, %d failed, %d skipped",
		successCount, failedCount, skippedCount))
	l.Info("═══════════════════════════════════════════════════════════════")
}

// ExtensionPublish выполняет публикацию расширения 1C в репозитории подписчиков.
// Команда выполняет полный цикл:
// 1. Получает информацию о релизе из исходного репозитория
// 2. Находит все репозитории, подписанные на обновления
// 3. Для каждого подписчика синхронизирует файлы и создает Pull Request
//
// Переменные окружения:
//   - GITHUB_REPOSITORY: полное имя исходного репозитория (owner/repo)
//   - GITHUB_REF_NAME: тег релиза (например, v1.2.3)
//   - BR_EXT_DIR: каталог с расширением в исходном репозитории (опционально)
//   - BR_DRY_RUN: если "true", выполняется в режиме без изменений
//
// Параметры:
//   - ctx: контекст выполнения
//   - l: логгер
//   - cfg: конфигурация приложения
//
// Возвращает:
//   - error: агрегированная ошибка или nil при успехе
func ExtensionPublish(_ *context.Context, l *slog.Logger, cfg *config.Config) error {
	// 1. Получение параметров из конфигурации
	releaseTag := cfg.ReleaseTag
	extensions := cfg.AddArray // Список расширений для публикации
	dryRun := cfg.DryRun

	// ToDo: Проверить, что cfg.Repo имеет формат owner/repo
	sourceRepo := fmt.Sprintf("%s/%s", cfg.Owner, cfg.Repo)

	// 2. Валидация обязательных параметров
	if sourceRepo == "" {
		return fmt.Errorf("переменная окружения GITHUB_REPOSITORY не установлена")
	}
	if releaseTag == "" {
		releaseTag = "main"
	}

	// Валидация конфигурации
	if cfg.GiteaURL == "" {
		return fmt.Errorf("GiteaURL не настроен в конфигурации")
	}
	if cfg.AccessToken == "" {
		return fmt.Errorf("AccessToken не настроен в конфигурации")
	}

	// Парсим owner/repo
	parts := strings.SplitN(sourceRepo, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("некорректный формат GITHUB_REPOSITORY: %s (ожидается owner/repo)", sourceRepo)
	}
	owner := parts[0]
	repo := parts[1]

	l.Info("Запуск публикации расширения",
		slog.String("repository", sourceRepo),
		slog.String("release_tag", releaseTag),
		slog.Any("extensions", extensions),
		slog.Bool("dry_run", dryRun),
	)

	// 3. Инициализация Gitea API для исходного репозитория
	// Используем releaseTag для получения файлов из тега релиза
	sourceAPI := gitea.NewGiteaAPI(gitea.Config{
		GiteaURL:    cfg.GiteaURL,
		Owner:       owner,
		Repo:        repo,
		AccessToken: cfg.AccessToken,
		BaseBranch:  "main",
	})

	// 4. Получение информации о релизе
	release, err := sourceAPI.GetReleaseByTag(releaseTag)
	if err != nil {
		return fmt.Errorf("ошибка получения релиза %s: %w", releaseTag, err)
	}

	l.Info("Найден релиз",
		slog.String("tag", release.TagName),
		slog.String("name", release.Name),
	)

	// 5. Поиск подписчиков
	// Расширения уже загружены из конфигурации в cfg.AddArray
	subscribers, err := FindSubscribedRepos(l, sourceAPI, repo, extensions)
	if err != nil {
		return fmt.Errorf("ошибка поиска подписчиков: %w", err)
	}

	if len(subscribers) == 0 {
		l.Info("Подписчики не найдены, завершение")
		return nil
	}

	l.Info("Найдено подписчиков",
		slog.Int("count", len(subscribers)),
	)

	// 6. Инициализация отчёта
	// Имя расширения будет определяться для каждого подписчика индивидуально
	report := &PublishReport{
		ExtensionName: repo, // По умолчанию используем имя репозитория
		Version:       release.TagName,
		SourceRepo:    sourceRepo,
		StartTime:     time.Now(),
	}

	// 8. Обработка каждого подписчика (continue on error)
	for _, sub := range subscribers {
		startTime := time.Now()

		l.Info("Обработка подписчика",
			slog.String("organization", sub.Organization),
			slog.String("repository", sub.Repository),
			slog.String("target_dir", sub.TargetDirectory),
		)

		// Dry-run режим — пропускаем без изменений
		if dryRun {
			l.Info("DRY-RUN: будет синхронизирован",
				slog.String("target", fmt.Sprintf("%s/%s", sub.Organization, sub.Repository)),
				slog.String("dir", sub.TargetDirectory),
			)
			report.Results = append(report.Results, PublishResult{
				Subscriber:   sub,
				Status:       StatusSkipped,
				ErrorMessage: "dry-run mode",
				DurationMs:   time.Since(startTime).Milliseconds(),
			})
			continue
		}

		// Создаём API для целевого репозитория
		targetAPI := gitea.NewGiteaAPI(gitea.Config{
			GiteaURL:    cfg.GiteaURL,
			Owner:       sub.Organization,
			Repo:        sub.Repository,
			AccessToken: cfg.AccessToken,
			BaseBranch:  sub.TargetBranch,
		})
		// Анализируем проект
		analysis, err := targetAPI.AnalyzeProject("main")
		if err != nil {
			l.Error("Ошибка анализа проекта",
				slog.String("error", err.Error()),
			)
			return err
		}
		var targetProjectName string
		// Заполняем поля конфигурации
		if len(analysis) == 0 {
			l.Error("Проект не найден или не соответствует критериям",
				slog.String("organization", sub.Organization),
				slog.String("repository", sub.Repository),
				slog.String("target_branch", sub.TargetBranch),
				slog.String("target_directory", sub.TargetDirectory),
			)

			continue
		} else {
			targetProjectName = analysis[0]
			l.Debug("Результат анализа проекта",
				slog.String("project_name", targetProjectName),
				slog.Any("extensions", cfg.AddArray),
			)
		}

		// Определяем исходный каталог и имя расширения из TargetDirectory подписчика
		sourceDir := cfg.ProjectName + "." + sub.TargetDirectory
		targetDir := targetProjectName + "." + sub.TargetDirectory

		// Синхронизируем файлы
		// extName используется для формирования имени ветки и commit message
		extName := sub.TargetDirectory
		syncResult, err := SyncExtensionToRepo(
			l,
			sourceAPI,
			targetAPI,
			sub,
			sourceDir,
			sourceAPI.BaseBranch,
			targetDir,
			extName,
			release.TagName,
		)
		if err != nil {
			l.Error("Ошибка синхронизации",
				slog.String("target", fmt.Sprintf("%s/%s", sub.Organization, sub.Repository)),
				slog.String("error", err.Error()),
			)
			report.Results = append(report.Results, PublishResult{
				Subscriber:   sub,
				Status:       StatusFailed,
				Error:        err,
				ErrorMessage: err.Error(),
				DurationMs:   time.Since(startTime).Milliseconds(),
			})
			continue // Continue on error — не прерываем цикл
		}

		if syncResult.Error != nil {
			l.Error("Ошибка синхронизации (результат)",
				slog.String("target", fmt.Sprintf("%s/%s", sub.Organization, sub.Repository)),
				slog.String("error", syncResult.Error.Error()),
			)
			report.Results = append(report.Results, PublishResult{
				Subscriber:   sub,
				Status:       StatusFailed,
				SyncResult:   syncResult,
				Error:        syncResult.Error,
				ErrorMessage: syncResult.Error.Error(),
				DurationMs:   time.Since(startTime).Milliseconds(),
			})
			continue // Continue on error
		}

		l.Info("Файлы синхронизированы",
			slog.String("target", fmt.Sprintf("%s/%s", sub.Organization, sub.Repository)),
			slog.String("branch", syncResult.NewBranch),
			slog.Int("files_created", syncResult.FilesCreated),
			slog.Int("files_deleted", syncResult.FilesDeleted),
		)

		// Формируем URL релиза
		releaseURL := fmt.Sprintf("%s/%s/releases/tag/%s", cfg.GiteaURL, sourceRepo, releaseTag)

		// Создаём Pull Request
		pr, err := CreateExtensionPR(l, targetAPI, syncResult, release, extName, sourceRepo, releaseURL)
		if err != nil {
			l.Error("Ошибка создания PR",
				slog.String("target", fmt.Sprintf("%s/%s", sub.Organization, sub.Repository)),
				slog.String("error", err.Error()),
			)
			report.Results = append(report.Results, PublishResult{
				Subscriber:   sub,
				Status:       StatusFailed,
				SyncResult:   syncResult,
				Error:        err,
				ErrorMessage: err.Error(),
				DurationMs:   time.Since(startTime).Milliseconds(),
			})
			continue // Continue on error
		}

		l.Info("PR успешно создан",
			slog.String("target", fmt.Sprintf("%s/%s", sub.Organization, sub.Repository)),
			slog.Int64("pr_number", pr.Number),
			slog.String("url", pr.HTMLURL),
		)

		report.Results = append(report.Results, PublishResult{
			Subscriber: sub,
			Status:     StatusSuccess,
			SyncResult: syncResult,
			PRNumber:   int(pr.Number),
			PRURL:      pr.HTMLURL,
			DurationMs: time.Since(startTime).Milliseconds(),
		})
	}

	// 9. Финализация отчёта
	report.EndTime = time.Now()

	// 10. Вывод отчёта
	if err := ReportResults(report, l); err != nil {
		return fmt.Errorf("ошибка вывода отчёта: %w", err)
	}

	// 11. Return error если есть хотя бы одна ошибка (exit code = 1)
	if report.HasErrors() {
		return fmt.Errorf("публикация завершена с %d ошибками из %d",
			report.FailedCount(), len(report.Results))
	}

	return nil
}
