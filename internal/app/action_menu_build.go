// Package app содержит функциональность построения меню действий для gitea Actions
package app

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/entity/gitea"
	"github.com/Kargones/apk-ci/internal/service"
	util "github.com/Kargones/apk-ci/internal/util"
)

// TemplateData представляет данные для заполнения YAML шаблонов.
// Содержит информацию о задаче, базах данных и ветках для генерации
// конфигурационных файлов GitHub Actions.
type TemplateData struct {
	TaskName      string
	DatabaseName  string
	SourceBranch  string
	TargetBranch  string
	TestDatabases []string
	ProdDatabases []string
}

// ProjectDatabase представляет информацию о базе данных проекта.
// Содержит метаданные о базе данных, включая её название, описание
// и признак продуктивности.
type ProjectDatabase struct {
	Name        string
	Description string
	Prod        bool
}

// FileInfo представляет информацию о генерируемом файле.
// Содержит путь к файлу, его содержимое и SHA-хеш для проверки
// целостности и отслеживания изменений.
type FileInfo struct {
	Path    string
	Content string
	SHA     string
}

// ActionMenuBuildError представляет кастомную ошибку для функции ActionMenuBuild.
// Предоставляет детализированную информацию об ошибках, возникающих
// в процессе построения меню действий, включая операцию, причину и детали.
type ActionMenuBuildError struct {
	Operation string
	Cause     error
	Details   string
}

// Error реализует интерфейс error для ActionMenuBuildError
// Error возвращает строковое представление ошибки ActionMenuBuildError.
// Форматирует сообщение об ошибке, включая операцию, причину и детали.
// Возвращает:
//   - string: отформатированное сообщение об ошибке
func (e *ActionMenuBuildError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("ActionMenuBuild error in %s: %s (details: %s)", e.Operation, e.Cause.Error(), e.Details)
	}
	return fmt.Sprintf("ActionMenuBuild error in %s (details: %s)", e.Operation, e.Details)
}

// Unwrap позволяет использовать errors.Is и errors.As
// Unwrap возвращает исходную ошибку, обернутую в ActionMenuBuildError.
// Позволяет использовать функции errors.Is и errors.As для проверки типа ошибки.
// Возвращает:
//   - error: исходная ошибка или nil, если она отсутствует
func (e *ActionMenuBuildError) Unwrap() error {
	return e.Cause
}

// newActionMenuBuildError создает новую ошибку ActionMenuBuild
func newActionMenuBuildError(operation string, cause error, details string) *ActionMenuBuildError {
	return &ActionMenuBuildError{
		Operation: operation,
		Cause:     cause,
		Details:   details,
	}
}

// checkProjectYamlChanges проверяет, был ли изменён файл project.yaml в последнем коммите
func checkProjectYamlChanges(_ context.Context, l *slog.Logger, cfg *config.Config) (bool, error) {
	l.Debug("Проверка изменений файла project.yaml в последнем коммите")

	// Создание GiteaService через фабрику
	factory := &service.GiteaFactory{}
	giteaService, err := factory.CreateGiteaService(cfg)
	if err != nil {
		l.Error("Ошибка создания GiteaService", slog.String("error", err.Error()))
		return false, err
	}

	// Получение Gitea API из сервиса
	giteaAPI := giteaService.GetAPI()
	giteaConfig := giteaService.GetConfig()

	// Получение последнего коммита из основной ветки
	latestCommit, err := giteaAPI.GetLatestCommit(giteaConfig.BaseBranch)
	if err != nil {
		l.Error("Ошибка при получении последнего коммита", slog.String("error", err.Error()))
		return false, newActionMenuBuildError("получение последнего коммита", err, "не удалось получить информацию о коммите")
	}

	l.Debug("Получен последний коммит", slog.String("sha", latestCommit.SHA))

	// Получение списка файлов, измененных в последнем коммите
	commitFiles, err := giteaAPI.GetCommitFiles(latestCommit.SHA)
	if err != nil {
		l.Error("Ошибка при получении файлов коммита", slog.String("error", err.Error()))
		return false, newActionMenuBuildError("получение файлов коммита", err, "не удалось получить список измененных файлов")
	}

	// Проверка, был ли изменен файл project.yaml
	for _, file := range commitFiles {
		if file.Filename == "project.yaml" {
			l.Info("Файл project.yaml был изменен в последнем коммите",
				slog.String("status", file.Status),
				slog.String("commit_sha", latestCommit.SHA))
			return true, nil
		}
	}

	l.Debug("Файл project.yaml не был изменен в последнем коммите")
	return false, nil
}

// generateFiles генерирует все необходимые файлы действий на основе конфигурации
func generateFiles(_ context.Context, l *slog.Logger, cfg *config.Config, databases []ProjectDatabase, debug bool) ([]FileInfo, error) {
	l.Debug("Генерация файлов действий")

	files := []FileInfo{}

	// Подготовка списков баз данных
	testDatabases := []string{}
	prodDatabases := []string{}
	for _, db := range databases {
		if db.Prod {
			prodDatabases = append(prodDatabases, db.Name)
		} else {
			testDatabases = append(testDatabases, db.Name)
		}
	}
	if len(prodDatabases) == 0 || len(testDatabases) == 0 {
		return nil, newActionMenuBuildError("generateFiles", errors.New("no prod or test databases"), "Нет прод или тестовых баз данных")
	}
	ReplacementStringTestDatabase := testDatabases[0]
	ReplacementStringProdDatabase := prodDatabases[0]
	ReplacementStringTestDatabaseAll := "\n          - " + strings.Join(testDatabases, "\n          - ")
	ReplacementStringProdDatabaseAll := "\n          - " + strings.Join(prodDatabases, "\n          - ")
	ReplacementRules := []util.ReplacementRule{
		{
			SearchString:      "$TestBaseReplace$",
			ReplacementString: ReplacementStringTestDatabase,
		},
		{
			SearchString:      "$TestBaseReplaceAll$",
			ReplacementString: ReplacementStringTestDatabaseAll,
		},
		{
			SearchString:      "$ProdBaseReplace$",
			ReplacementString: ReplacementStringProdDatabase,
		},
		{
			SearchString:      "$ProdBaseReplaceAll$",
			ReplacementString: ReplacementStringProdDatabaseAll,
		},
	}

	// Объединяем массив строк MenuMain в одну строку
	menuMainContent := strings.Join(cfg.MenuMain, "\n")
	TemplateResult, err := util.ProcessMultipleTemplates(menuMainContent, ReplacementRules)
	if err != nil {
		return nil, newActionMenuBuildError("generateFiles", err, "Ошибка генерации файлов")
	}
	if TemplateResult == nil {
		return nil, newActionMenuBuildError("generateFiles", errors.New("no template result"), "Нет результата шаблона")
	}
	if debug {
		// Объединяем массив строк MenuDebug в одну строку
		menuDebugContent := strings.Join(cfg.MenuDebug, "\n")
		TemplateResultDebug, err := util.ProcessMultipleTemplates(menuDebugContent, ReplacementRules)
		if err == nil {
			TemplateResult = append(TemplateResult, TemplateResultDebug...)
		}
	}

	for _, template := range TemplateResult {
		// Вычисляем SHA-хеш содержимого файла
		hash := sha256.Sum256([]byte(template.Result))
		sha := hex.EncodeToString(hash[:])

		files = append(files, FileInfo{
			Path:    constants.GiteaWorkflowsPath + "/" + template.FileName,
			Content: template.Result,
			SHA:     sha,
		})
	}

	l.Debug("Генерация файлов завершена", slog.Int("files_generated", len(files)))
	return files, nil
}

// getCurrentActions получает список текущих файлов действий из репозитория
func getCurrentActions(_ context.Context, l *slog.Logger, cfg *config.Config) ([]FileInfo, error) {
	l.Debug("Получение списка текущих действий")

	// Создание GiteaService через фабрику
	factory := &service.GiteaFactory{}
	giteaService, err := factory.CreateGiteaService(cfg)
	if err != nil {
		l.Error("Ошибка создания GiteaService", slog.String("error", err.Error()))
		return nil, err
	}

	// Получение Gitea API из сервиса
	giteaAPI := giteaService.GetAPI()
	giteaConfig := giteaService.GetConfig()

	// Получение содержимого директории .gitea/workflows
	contents, err := giteaAPI.GetRepositoryContents(constants.GiteaWorkflowsPath, giteaConfig.BaseBranch)
	if err != nil {
		// Отсутствие каталога не является ошибкой
		l.Debug("Каталог .gitea/workflows не найден, возвращается пустой список", slog.String("error", err.Error()))
	}

	var actionFiles []FileInfo

	// Фильтрация файлов действий
	for _, file := range contents {
		// Проверяем, что это файл в директории .gitea/workflows
		if strings.HasPrefix(file.Path, constants.GiteaWorkflowsPath) && (strings.HasSuffix(file.Name, ".yml") || strings.HasSuffix(file.Name, ".yaml")) {
			// Получаем содержимое файла
			content, err := giteaAPI.GetFileContent(file.Path)
			if err != nil {
				l.Warn("Не удалось получить содержимое файла", slog.String("path", file.Path), slog.String("error", err.Error()))
				continue
			}

			// Преобразуем содержимое в строку
			contentStr := string(content)

			// Используем SHA из Gitea API вместо локального вычисления
			actionFiles = append(actionFiles, FileInfo{
				Path:    file.Path,
				Content: contentStr,
				SHA:     file.SHA, // Используем SHA из API
			})
		}
	}

	l.Info("Получен список текущих действий", slog.Int("action_files_count", len(actionFiles)))
	return actionFiles, nil
}

// syncWorkflowFiles выполняет атомарную синхронизацию файлов:
// - добавляет новые файлы
// - обновляет существующие файлы
// - удаляет устаревшие файлы
// Возвращает количество добавленных, обновленных и удаленных файлов
func syncWorkflowFiles(_ context.Context, l *slog.Logger, cfg *config.Config, currentFiles []FileInfo, newFiles []FileInfo) (int, int, int, error) {
	l.Debug("Начало атомарной синхронизации файлов")

	// Создание GiteaService через фабрику
	factory := &service.GiteaFactory{}
	giteaService, err := factory.CreateGiteaService(cfg)
	if err != nil {
		l.Error("Ошибка создания GiteaService", slog.String("error", err.Error()))
		return 0, 0, 0, err
	}

	// Получение API из сервиса
	giteaAPI := giteaService.GetAPI()
	giteaConfig := giteaService.GetConfig()

	// Создаем карты для быстрого поиска
	currentFileMap := make(map[string]FileInfo)
	for _, file := range currentFiles {
		currentFileMap[file.Path] = file
		l.Debug("Текущий файл", slog.String("fullPath", file.Path), slog.String("sha", file.SHA))
	}

	newFileMap := make(map[string]FileInfo)
	for _, file := range newFiles {
		newFileMap[file.Path] = file
		l.Debug("Новый файл", slog.String("path", file.Path), slog.String("sha", file.SHA))
	}

	var addedCount, updatedCount, deletedCount int
	// Подготавливаем операции для атомарного выполнения
	var operations []gitea.ChangeFileOperation

	// 1. Обработка файлов из newFiles (добавление и обновление)
	for _, newFile := range newFiles {
		if currentFile, exists := currentFileMap[newFile.Path]; exists {
			// Файл существует - проверяем SHA-хеши
			if currentFile.SHA != newFile.SHA {
				// SHA-хеши различаются - добавляем операцию обновления
				l.Debug("Планируется обновление файла (SHA изменился)",
					slog.String("path", newFile.Path),
					slog.String("old_sha", currentFile.SHA),
					slog.String("new_sha", newFile.SHA))

				operations = append(operations, gitea.ChangeFileOperation{
					Operation: "update",
					Path:      newFile.Path,
					Content:   base64.StdEncoding.EncodeToString([]byte(newFile.Content)),
					SHA:       currentFile.SHA,
				})
				updatedCount++
			} else {
				// SHA-хеши совпадают - файл не изменился
				l.Debug("Файл не изменился (SHA совпадает)",
					slog.String("path", newFile.Path),
					slog.String("sha", newFile.SHA))
			}
		} else {
			// Файл не существует - добавляем операцию создания
			l.Debug("Планируется добавление файла", slog.String("path", newFile.Path))
			operations = append(operations, gitea.ChangeFileOperation{
				Operation: "create",
				Path:      newFile.Path,
				Content:   base64.StdEncoding.EncodeToString([]byte(newFile.Content)),
			})
			addedCount++
		}
	}

	// 2. Удаление файлов, которые есть в currentFiles, но отсутствуют в newFiles
	for _, currentFile := range currentFiles {
		if _, exists := newFileMap[currentFile.Path]; !exists {
			// Файл отсутствует в новых файлах - добавляем операцию удаления
			l.Debug("Планируется удаление файла", slog.String("fullPath", currentFile.Path))

			operations = append(operations, gitea.ChangeFileOperation{
				Operation: "delete",
				Path:      currentFile.Path,
				SHA:       currentFile.SHA,
			})
			deletedCount++
		}
	}

	// 3. Выполняем все операции атомарно в одном коммите
	if len(operations) > 0 {
		commitMessage := fmt.Sprintf("Sync workflow files: +%d ~%d -%d", addedCount, updatedCount, deletedCount)

		err = giteaAPI.SetRepositoryState(l, operations, giteaConfig.BaseBranch, commitMessage)
		if err != nil {
			return addedCount, updatedCount, deletedCount, newActionMenuBuildError("syncWorkflowFiles", err, "Ошибка атомарного выполнения операций")
		}
		l.Info("Атомарные операции выполнены успешно", slog.String("commit_message", commitMessage))
	} else {
		l.Info("Нет изменений для синхронизации")
	}

	l.Info("Атомарная синхронизация файлов завершена",
		slog.Int("added", addedCount),
		slog.Int("updated", updatedCount),
		slog.Int("deleted", deletedCount))

	return addedCount, updatedCount, deletedCount, nil
}

// commitChanges выполняет коммит всех изменений
func commitChanges(_ context.Context, l *slog.Logger, cfg *config.Config, message string) error {
	l.Debug("Коммит изменений", slog.String("message", message))

	// Добавляем временную метку к сообщению коммита
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fullMessage := fmt.Sprintf("%s [%s]", message, timestamp)

	// В реальной реализации здесь должен быть вызов Git API для коммита
	// Для демонстрации просто логируем
	l.Info("Изменения закоммичены",
		slog.String("commit_message", fullMessage),
		slog.String("branch", cfg.BaseBranch))

	return nil
}

// ActionMenuBuild - основная функция для построения меню действий
// Реализует полный алгоритм согласно архитектуре:
// 1. Проверка изменений project.yaml
// 2. Анализ конфигурации
// 3. Получение текущих файлов
// 4. Генерация новых файлов
// 5. Создание/удаление файлов
// 6. Коммит изменений
func ActionMenuBuild(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	l.Info("Запуск ActionMenuBuild")

	// Этап 1: Проверка изменений project.yaml (если не принудительное обновление)
	if !cfg.ForceUpdate {
		l.Debug("Этап 1: Проверка изменений project.yaml")
		hasChanges, err := checkProjectYamlChanges(ctx, l, cfg)
		if err != nil {
			return newActionMenuBuildError("ActionMenuBuild", err, "Ошибка на этапе проверки изменений project.yaml")
		}

		if !hasChanges {
			l.Info("Изменения в project.yaml не обнаружены, построение меню не требуется")
			return nil
		}
	} else {
		l.Info("Принудительное обновление меню действий (force_update=true)")
	}

	// Этап 2: Анализ конфигурации
	l.Debug("Этап 2: Анализ конфигурации проекта")
	var databases []ProjectDatabase
	if cfg.ProjectConfig != nil {
		for dbName, dbInfo := range cfg.ProjectConfig.Prod {
			databases = append(databases, ProjectDatabase{
				Name:        dbName,
				Description: dbInfo.DbName,
				Prod:        true,
			})
			for relatedDbName := range dbInfo.Related {
				databases = append(databases, ProjectDatabase{
					Name:        relatedDbName,
					Description: "",
					Prod:        false,
				})
			}
		}
	}

	if len(databases) == 0 {
		l.Warn("Базы данных не найдены в конфигурации")
		return nil
	}

	// Этап 3: Получение текущих файлов
	l.Debug("Этап 3: Получение списка текущих файлов действий")
	currentFiles, err := getCurrentActions(ctx, l, cfg)
	if err != nil {
		return newActionMenuBuildError("ActionMenuBuild", err, "Ошибка на этапе получения текущих файлов")
	}

	// Этап 4: Генерация новых файлов
	l.Debug("Этап 4: Генерация новых файлов действий")
	newFiles, err := generateFiles(ctx, l, cfg, databases, cfg.ProjectConfig.Debug)
	if err != nil {
		return newActionMenuBuildError("ActionMenuBuild", err, "Ошибка на этапе генерации файлов")
	}

	// Этап 5: Атомарная синхронизация файлов
	l.Debug("Этап 5: Атомарная синхронизация файлов")
	addedCount, updatedCount, deletedCount, err := syncWorkflowFiles(ctx, l, cfg, currentFiles, newFiles)
	if err != nil {
		return newActionMenuBuildError("ActionMenuBuild", err, "Ошибка на этапе синхронизации файлов")
	}

	// Этап 6: Коммит изменений
	l.Debug("Этап 6: Коммит изменений")
	commitMessage := fmt.Sprintf("ActionMenuBuild: синхронизация меню действий (+%d ~%d -%d)", addedCount, updatedCount, deletedCount)
	err = commitChanges(ctx, l, cfg, commitMessage)
	if err != nil {
		return newActionMenuBuildError("ActionMenuBuild", err, "Ошибка на этапе коммита изменений")
	}

	l.Info("ActionMenuBuild успешно завершен",
		slog.Int("databases_processed", len(databases)),
		slog.Int("files_generated", len(newFiles)),
		slog.Int("current_files", len(currentFiles)))

	return nil
}
