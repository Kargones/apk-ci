package extensionpublishhandler

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"path"
	"strings"

	"github.com/Kargones/apk-ci/internal/entity/gitea"
)

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
const (
	contentTypeDir  = "dir"
	contentTypeFile = "file"
)

func GetSourceFiles(ctx context.Context, api *gitea.API, sourceDir, branch string) ([]gitea.ChangeFileOperation, error) {
	operations, err := getSourceFilesRecursive(ctx, api, sourceDir, "", branch)
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
func GetTargetFilesToDelete(ctx context.Context, api *gitea.API, targetDir, branch string) ([]gitea.ChangeFileOperation, error) {
	return getTargetFilesRecursive(ctx, api, targetDir, branch)
}

// getTargetFilesRecursive рекурсивно обходит целевой каталог и собирает операции удаления.
func getTargetFilesRecursive(ctx context.Context, api *gitea.API, currentDir, branch string) ([]gitea.ChangeFileOperation, error) {
	// Получаем содержимое текущего каталога
	contents, err := api.GetRepositoryContents(ctx, currentDir, branch)
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
		case contentTypeDir:
			// Рекурсивный вызов для подкаталога
			// Используем path.Join вместо filepath.Join для кросс-платформенной совместимости
			// (Gitea API всегда ожидает Unix-пути с прямыми слешами)
			subOps, err := getTargetFilesRecursive(ctx, 
				api,
				path.Join(currentDir, item.Name),
				branch,
			)
			if err != nil {
				return nil, err
			}
			operations = append(operations, subOps...)

		case contentTypeFile:
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
func GetTargetFilesMap(ctx context.Context, api *gitea.API, targetDir, branch string) (map[string]string, error) {
	return getTargetFilesMapRecursive(ctx, api, targetDir, targetDir, branch)
}

// getTargetFilesMapRecursive рекурсивно обходит каталог и собирает карту файлов.
// baseDir - базовый каталог для вычисления относительных путей.
func getTargetFilesMapRecursive(ctx context.Context, api *gitea.API, currentDir, baseDir, branch string) (map[string]string, error) {
	contents, err := api.GetRepositoryContents(ctx, currentDir, branch)
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
		case contentTypeDir:
			// Рекурсивный вызов для подкаталога
			subMap, err := getTargetFilesMapRecursive(ctx, 
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

		case contentTypeFile:
			// Вычисляем относительный путь от baseDir
			relativePath := strings.TrimPrefix(item.Path, baseDir+"/")
			result[relativePath] = item.SHA
		}
	}

	return result, nil
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
func SyncExtensionToRepo(ctx context.Context, l *slog.Logger, sourceAPI, targetAPI *gitea.API, subscriber SubscribedRepo, sourceDir, sourceBranch, targetDir, extName, version string) (*SyncResult, error) {
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
	sourceOps, err := GetSourceFiles(ctx, sourceAPI, sourceDir, sourceBranch)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения исходных файлов: %w", err)
	}

	// Логируем исходные файлы для диагностики
	logger.Debug("SyncExtensionToRepo: получено исходных файлов",
		slog.Int("count", len(sourceOps)),
	)

	// 2. Получаем карту существующих файлов в целевом каталоге (путь -> SHA)
	targetFilesMap, err := GetTargetFilesMap(ctx, targetAPI, targetDir, subscriber.TargetBranch)
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
	commitSHA, err := targetAPI.SetRepositoryStateWithNewBranch(ctx, 
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

// getSourceFilesRecursive рекурсивно обходит каталог и собирает файлы.
// basePath используется для формирования относительных путей файлов.
func getSourceFilesRecursive(ctx context.Context, api *gitea.API, currentDir, basePath, branch string) ([]gitea.ChangeFileOperation, error) {
	// Получаем содержимое текущего каталога
	contents, err := api.GetRepositoryContents(ctx, currentDir, branch)
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
		case contentTypeDir:
			// Рекурсивный вызов для подкаталога
			subOps, err := getSourceFilesRecursive(ctx, 
				api,
				path.Join(currentDir, item.Name),
				relativePath,
				branch,
			)
			if err != nil {
				return nil, err
			}
			operations = append(operations, subOps...)

		case contentTypeFile:
			// Получаем содержимое файла (GetFileContent возвращает уже декодированные байты)
			content, err := api.GetFileContent(ctx, item.Path)
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
