package actionmenu

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Kargones/apk-ci/internal/adapter/gitea"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	templateprocessor "github.com/Kargones/apk-ci/internal/util"
)

// checkProjectYamlChanges проверяет, был ли изменён project.yaml в последнем коммите.
func (h *ActionMenuHandler) checkProjectYamlChanges(ctx context.Context, client gitea.Client, baseBranch string, log *slog.Logger) (bool, error) {
	log.Debug("Проверка изменений project.yaml в последнем коммите")

	// Получение последнего коммита
	latestCommit, err := client.GetLatestCommit(ctx, baseBranch)
	if err != nil {
		return false, fmt.Errorf("не удалось получить последний коммит: %w", err)
	}

	log.Debug("Получен последний коммит", slog.String("sha", latestCommit.SHA))

	// Получение файлов коммита
	commitFiles, err := client.GetCommitFiles(ctx, latestCommit.SHA)
	if err != nil {
		return false, fmt.Errorf("не удалось получить файлы коммита: %w", err)
	}

	// Поиск project.yaml среди изменённых файлов
	for _, file := range commitFiles {
		if file.Filename == "project.yaml" {
			log.Info("Файл project.yaml был изменён в последнем коммите",
				slog.String("status", file.Status),
				slog.String("commit_sha", latestCommit.SHA))
			return true, nil
		}
	}

	log.Debug("Файл project.yaml не был изменён в последнем коммите")
	return false, nil
}

// extractDatabases извлекает список баз данных из конфигурации.
func (h *ActionMenuHandler) extractDatabases(cfg *config.Config, log *slog.Logger) []ProjectDatabase {
	var databases []ProjectDatabase

	if cfg.ProjectConfig == nil {
		if log != nil {
			log.Debug("ProjectConfig is nil")
		}
		return databases
	}

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

	if log != nil {
		log.Debug("Извлечены базы данных из конфигурации", slog.Int("count", len(databases)))
	}
	return databases
}

// generateFiles генерирует файлы из шаблонов с подстановкой переменных (AC: #2, #3).
func (h *ActionMenuHandler) generateFiles(cfg *config.Config, databases []ProjectDatabase, log *slog.Logger) ([]FileInfo, error) {
	if log != nil {
		log.Debug("Генерация файлов действий")
	}

	// Подготовка списков баз данных
	var testDatabases, prodDatabases []string
	for _, db := range databases {
		if db.Prod {
			prodDatabases = append(prodDatabases, db.Name)
		} else {
			testDatabases = append(testDatabases, db.Name)
		}
	}

	if len(prodDatabases) == 0 || len(testDatabases) == 0 {
		return nil, fmt.Errorf("недостаточно баз данных: нужны prod и test базы (prod=%d, test=%d)",
			len(prodDatabases), len(testDatabases))
	}

	// Правила замены (AC: #3)
	replacementRules := []templateprocessor.ReplacementRule{
		{SearchString: "$TestBaseReplace$", ReplacementString: testDatabases[0]},
		{SearchString: "$TestBaseReplaceAll$", ReplacementString: "\n          - " + strings.Join(testDatabases, "\n          - ")},
		{SearchString: "$ProdBaseReplace$", ReplacementString: prodDatabases[0]},
		{SearchString: "$ProdBaseReplaceAll$", ReplacementString: "\n          - " + strings.Join(prodDatabases, "\n          - ")},
	}

	var files []FileInfo

	// Обработка MenuMain (AC: #2)
	if len(cfg.MenuMain) > 0 {
		menuMainContent := strings.Join(cfg.MenuMain, "\n")
		results, err := templateprocessor.ProcessMultipleTemplates(menuMainContent, replacementRules)
		if err != nil {
			return nil, fmt.Errorf("ошибка обработки MenuMain: %w", err)
		}
		for _, tmpl := range results {
			hash := sha256.Sum256([]byte(tmpl.Result))
			files = append(files, FileInfo{
				Path:    constants.GiteaWorkflowsPath + "/" + tmpl.FileName,
				Content: tmpl.Result,
				SHA:     hex.EncodeToString(hash[:]),
			})
		}
	}

	// Обработка MenuDebug (AC: #2) — только если debug режим
	if cfg.ProjectConfig != nil && cfg.ProjectConfig.Debug && len(cfg.MenuDebug) > 0 {
		menuDebugContent := strings.Join(cfg.MenuDebug, "\n")
		results, err := templateprocessor.ProcessMultipleTemplates(menuDebugContent, replacementRules)
		if err != nil {
			if log != nil {
				log.Warn("Не удалось обработать MenuDebug, пропускаем", slog.String("error", err.Error()))
			}
		} else {
			for _, tmpl := range results {
				hash := sha256.Sum256([]byte(tmpl.Result))
				files = append(files, FileInfo{
					Path:    constants.GiteaWorkflowsPath + "/" + tmpl.FileName,
					Content: tmpl.Result,
					SHA:     hex.EncodeToString(hash[:]),
				})
			}
		}
	}

	if log != nil {
		log.Debug("Файлы сгенерированы", slog.Int("count", len(files)))
	}
	return files, nil
}

// getCurrentFiles получает текущие workflow файлы из репозитория.
// SHA вычисляется локально через SHA-256 от контента для корректного сравнения
// с новыми сгенерированными файлами (Git blob SHA-1 не совместим с SHA-256).
func (h *ActionMenuHandler) getCurrentFiles(ctx context.Context, client gitea.Client, baseBranch string, log *slog.Logger) ([]FileInfo, error) {
	log.Debug("Получение текущих файлов из репозитория")

	// Получение содержимого директории
	contents, err := client.GetRepositoryContents(ctx, constants.GiteaWorkflowsPath, baseBranch)
	if err != nil {
		// Отсутствие каталога не является ошибкой
		log.Debug("Каталог workflows не найден, возвращается пустой список",
			slog.String("error", err.Error()))
		return []FileInfo{}, nil
	}

	var actionFiles []FileInfo

	// Фильтрация только .yml и .yaml файлов
	for _, file := range contents {
		if !strings.HasSuffix(file.Name, ".yml") && !strings.HasSuffix(file.Name, ".yaml") {
			continue
		}

		// Получаем содержимое файла
		content, err := client.GetFileContent(ctx, file.Path)
		if err != nil {
			log.Warn("Не удалось получить содержимое файла",
				slog.String("path", file.Path),
				slog.String("error", err.Error()))
			continue
		}

		// Вычисляем SHA-256 от контента для корректного сравнения с новыми файлами.
		// Git blob SHA-1 (file.SHA от API) не совместим с SHA-256, поэтому
		// используем единый алгоритм хеширования для обеих сторон сравнения.
		contentHash := sha256.Sum256(content)
		contentSHA := hex.EncodeToString(contentHash[:])

		actionFiles = append(actionFiles, FileInfo{
			Path:    file.Path,
			Content: string(content),
			SHA:     contentSHA, // SHA-256 от контента для сравнения
			GitSHA:  file.SHA,   // Git blob SHA для API операций
		})
	}

	log.Info("Получены текущие файлы", slog.Int("count", len(actionFiles)))
	return actionFiles, nil
}
