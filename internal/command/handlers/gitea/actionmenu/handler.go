// Package actionmenu реализует NR-команду nr-action-menu-build
// для построения динамического меню действий в Gitea.
package actionmenu

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/gitea"
	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/command/handlers/gitea/shared"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/dryrun"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"
	templateprocessor "github.com/Kargones/apk-ci/internal/util"
)

// Коды ошибок — используем shared константы для соблюдения DRY.
// Локальные алиасы для краткости.
const (
	errConfigMissing    = shared.ErrConfigMissing
	errMissingOwnerRepo = shared.ErrMissingOwnerRepo
	errNoDatabases      = shared.ErrNoDatabases
	errGiteaAPI         = shared.ErrGiteaAPI
	errTemplateProcess  = shared.ErrTemplateProcess
	errSyncFailed       = shared.ErrSyncFailed
)

// init регистрирует команду nr-action-menu-build с deprecated alias action-menu-build.
// TODO: Deprecated alias "action-menu-build" будет удалён в v2.0.0 / Epic 7.
// После полной миграции на NR-архитектуру, использовать только "nr-action-menu-build".
func init() {
	command.RegisterWithAlias(&ActionMenuHandler{}, constants.ActionMenuBuildName)
}

// ActionMenuData содержит результат построения меню действий.
type ActionMenuData struct {
	// StateChanged — были ли внесены изменения
	StateChanged bool `json:"state_changed"`
	// AddedFiles — количество добавленных файлов
	AddedFiles int `json:"added_files"`
	// UpdatedFiles — количество обновлённых файлов
	UpdatedFiles int `json:"updated_files"`
	// DeletedFiles — количество удалённых файлов
	DeletedFiles int `json:"deleted_files"`
	// TotalGenerated — общее количество сгенерированных файлов
	TotalGenerated int `json:"total_generated"`
	// TotalCurrent — количество существующих файлов до синхронизации
	TotalCurrent int `json:"total_current"`
	// DatabasesProcessed — количество обработанных баз данных
	DatabasesProcessed int `json:"databases_processed"`
	// ForceUpdate — был ли включён режим принудительного обновления
	ForceUpdate bool `json:"force_update"`
	// ProjectYamlChanged — был ли изменён project.yaml
	ProjectYamlChanged bool `json:"project_yaml_changed"`
	// SyncedFiles — список синхронизированных файлов (опционально)
	SyncedFiles []SyncedFileInfo `json:"synced_files,omitempty"`
}

// SyncedFileInfo информация о синхронизированном файле.
type SyncedFileInfo struct {
	// Path — путь к файлу
	Path string `json:"path"`
	// Operation — тип операции: "create", "update", "delete"
	Operation string `json:"operation"`
}

// ProjectDatabase представляет информацию о базе данных проекта.
type ProjectDatabase struct {
	Name        string
	Description string
	Prod        bool
}

// FileInfo представляет информацию о файле.
type FileInfo struct {
	Path    string
	Content string
	SHA     string // SHA-256 хеш контента для сравнения
	GitSHA  string // Git blob SHA для API операций (только для текущих файлов)
}

// writeText выводит результаты построения меню в человекочитаемом формате.
func (d *ActionMenuData) writeText(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "══════════════════════════════════════════════════════\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "📋 Построение меню действий\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "══════════════════════════════════════════════════════\n"); err != nil {
		return err
	}

	if !d.StateChanged && !d.ProjectYamlChanged && !d.ForceUpdate {
		if _, err := fmt.Fprintf(w, "\nℹ️ Изменения в project.yaml не обнаружены.\n"); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "   Построение меню не требуется.\n"); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "══════════════════════════════════════════════════════\n"); err != nil {
			return err
		}
		return nil
	}

	forceStr := "нет"
	if d.ForceUpdate {
		forceStr = "да"
	}
	changedStr := "нет"
	if d.ProjectYamlChanged {
		changedStr = "да"
	}

	if _, err := fmt.Fprintf(w, "Принудительное обновление: %s\n", forceStr); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Изменения в project.yaml: %s\n\n", changedStr); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "📊 Обработка:\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  Баз данных обработано: %d\n", d.DatabasesProcessed); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  Файлов сгенерировано: %d\n", d.TotalGenerated); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  Файлов существовало: %d\n\n", d.TotalCurrent); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "📁 Синхронизация:\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  ✅ Добавлено: %d\n", d.AddedFiles); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  🔄 Обновлено: %d\n", d.UpdatedFiles); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  🗑️ Удалено: %d\n\n", d.DeletedFiles); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "══════════════════════════════════════════════════════\n"); err != nil {
		return err
	}
	if d.StateChanged {
		if _, err := fmt.Fprintf(w, "✅ Меню действий обновлено успешно\n"); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(w, "ℹ️ Меню действий актуально, изменений нет\n"); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "══════════════════════════════════════════════════════\n"); err != nil {
		return err
	}

	return nil
}

// ActionMenuHandler обрабатывает команду nr-action-menu-build.
type ActionMenuHandler struct {
	// giteaClient — клиент для работы с Gitea API.
	// Может быть nil в production (требуется реализация фабрики).
	// В тестах инъектируется напрямую.
	giteaClient gitea.Client
}

// Name возвращает имя команды.
func (h *ActionMenuHandler) Name() string {
	return constants.ActNRActionMenuBuild
}

// Description возвращает описание команды для вывода в help.
func (h *ActionMenuHandler) Description() string {
	return "Построить динамическое меню действий из конфигурации"
}

// Execute выполняет команду nr-action-menu-build.
func (h *ActionMenuHandler) Execute(ctx context.Context, cfg *config.Config) error {
	start := time.Now()

	traceID := tracing.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = tracing.GenerateTraceID()
	}

	format := os.Getenv("BR_OUTPUT_FORMAT")

	// Story 7.3 AC-8: plan-only для команд без поддержки плана
	// Review #36: !IsDryRun() — dry-run имеет приоритет над plan-only (AC-11).
	if !dryrun.IsDryRun() && dryrun.IsPlanOnly() {
		return dryrun.WritePlanOnlyUnsupported(os.Stdout, constants.ActNRActionMenuBuild)
	}

	log := slog.Default().With(slog.String("trace_id", traceID), slog.String("command", constants.ActNRActionMenuBuild))

	// 1. Валидация конфигурации (AC: #1)
	if cfg == nil {
		log.Error("Конфигурация не загружена")
		return h.writeError(format, traceID, start,
			errConfigMissing,
			"Конфигурация не загружена")
	}

	// 2. Получение и валидация Owner/Repo (AC: #2, #8)
	owner := cfg.Owner
	repo := cfg.Repo
	if owner == "" || repo == "" {
		log.Error("Не указаны owner или repo")
		return h.writeError(format, traceID, start,
			errMissingOwnerRepo,
			"Не указаны владелец (BR_OWNER) или репозиторий (BR_REPO)")
	}

	baseBranch := cfg.BaseBranch
	if baseBranch == "" {
		baseBranch = "main"
	}

	log.Info("Запуск построения меню действий",
		slog.String("owner", owner),
		slog.String("repo", repo),
		slog.String("base_branch", baseBranch),
		slog.Bool("force_update", cfg.ForceUpdate))

	// Получение Gitea клиента (AC: #8)
	// TODO: Реализовать фабрику createGiteaClient(cfg) для создания реального клиента.
	// Текущая реализация требует DI через поле giteaClient (используется в тестах).
	client := h.giteaClient
	if client == nil {
		log.Error("Gitea клиент не настроен")
		return h.writeError(format, traceID, start,
			errConfigMissing,
			"Gitea клиент не настроен — требуется реализация фабрики createGiteaClient()")
	}

	// 3. Проверка изменений project.yaml (если не ForceUpdate) (AC: #4)
	projectYamlChanged := true
	if !cfg.ForceUpdate {
		changed, err := h.checkProjectYamlChanges(ctx, client, baseBranch, log)
		if err != nil {
			log.Warn("Не удалось проверить изменения project.yaml, продолжаем в любом случае",
				slog.String("error", err.Error()))
		} else {
			projectYamlChanged = changed
		}

		if !projectYamlChanged {
			log.Info("Изменения в project.yaml не обнаружены, пропускаем построение меню")
			return h.writeSuccess(format, traceID, start, &ActionMenuData{
				StateChanged:       false,
				ForceUpdate:        false,
				ProjectYamlChanged: false,
			})
		}
	}

	// 4. Анализ конфигурации баз данных (AC: #2)
	databases := h.extractDatabases(cfg, log)
	if len(databases) == 0 {
		log.Warn("Базы данных не найдены в конфигурации")
		return h.writeSuccess(format, traceID, start, &ActionMenuData{
			StateChanged:       false,
			ForceUpdate:        cfg.ForceUpdate,
			ProjectYamlChanged: projectYamlChanged,
		})
	}

	// 5. Генерация новых файлов (AC: #2, #3)
	// Предупреждение: если MenuMain пуст, все существующие workflow файлы будут удалены
	if len(cfg.MenuMain) == 0 {
		log.Warn("MenuMain пуст — все существующие workflow файлы будут удалены при синхронизации")
	}

	newFiles, err := h.generateFiles(cfg, databases, log)
	if err != nil {
		log.Error("Не удалось сгенерировать файлы", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start, errTemplateProcess, err.Error())
	}

	// 6. Получение текущих файлов (AC: #5)
	currentFiles, err := h.getCurrentFiles(ctx, client, baseBranch, log)
	if err != nil {
		log.Warn("Не удалось получить текущие файлы, считаем пустым",
			slog.String("error", err.Error()))
		currentFiles = []FileInfo{}
	}

	// 7. Атомарная синхронизация (AC: #5, #10)
	added, updated, deleted, syncedFiles, err := h.syncFiles(ctx, client, baseBranch, currentFiles, newFiles, log)
	if err != nil {
		log.Error("Не удалось выполнить синхронизацию файлов", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start, errSyncFailed, err.Error())
	}

	stateChanged := added+updated+deleted > 0

	log.Info("Построение меню действий завершено",
		slog.Int("added", added),
		slog.Int("updated", updated),
		slog.Int("deleted", deleted),
		slog.Bool("state_changed", stateChanged))

	return h.writeSuccess(format, traceID, start, &ActionMenuData{
		StateChanged:       stateChanged,
		AddedFiles:         added,
		UpdatedFiles:       updated,
		DeletedFiles:       deleted,
		TotalGenerated:     len(newFiles),
		TotalCurrent:       len(currentFiles),
		DatabasesProcessed: len(databases),
		ForceUpdate:        cfg.ForceUpdate,
		ProjectYamlChanged: projectYamlChanged,
		SyncedFiles:        syncedFiles,
	})
}

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

// syncFiles выполняет атомарную синхронизацию файлов (AC: #5).
// Возвращает: добавлено, обновлено, удалено, список операций, ошибка.
func (h *ActionMenuHandler) syncFiles(ctx context.Context, client gitea.Client, baseBranch string,
	currentFiles, newFiles []FileInfo, log *slog.Logger) (int, int, int, []SyncedFileInfo, error) {

	log.Debug("Начало атомарной синхронизации файлов")

	// Создание карт для быстрого поиска
	currentFileMap := make(map[string]FileInfo)
	for _, file := range currentFiles {
		currentFileMap[file.Path] = file
	}

	newFileMap := make(map[string]FileInfo)
	for _, file := range newFiles {
		newFileMap[file.Path] = file
	}

	var addedCount, updatedCount, deletedCount int
	var operations []gitea.BatchOperation
	var syncedFiles []SyncedFileInfo

	// 1. Обработка файлов из newFiles (добавление и обновление)
	for _, newFile := range newFiles {
		if currentFile, exists := currentFileMap[newFile.Path]; exists {
			// Файл существует — проверяем SHA-256 хеши контента
			if currentFile.SHA != newFile.SHA {
				log.Debug("Планируется обновление файла (контент изменился)",
					slog.String("path", newFile.Path),
					slog.String("old_sha", currentFile.SHA),
					slog.String("new_sha", newFile.SHA))

				operations = append(operations, gitea.BatchOperation{
					Operation: "update",
					Path:      newFile.Path,
					Content:   base64.StdEncoding.EncodeToString([]byte(newFile.Content)),
					SHA:       currentFile.GitSHA, // Git blob SHA для API
				})
				syncedFiles = append(syncedFiles, SyncedFileInfo{Path: newFile.Path, Operation: "update"})
				updatedCount++
			} else {
				// SHA-256 хеши совпадают — файл не изменился
				log.Debug("Файл не изменился (контент идентичен)",
					slog.String("path", newFile.Path),
					slog.String("sha", newFile.SHA))
			}
		} else {
			// Файл не существует — добавляем
			log.Debug("Планируется добавление файла", slog.String("path", newFile.Path))
			operations = append(operations, gitea.BatchOperation{
				Operation: "create",
				Path:      newFile.Path,
				Content:   base64.StdEncoding.EncodeToString([]byte(newFile.Content)),
			})
			syncedFiles = append(syncedFiles, SyncedFileInfo{Path: newFile.Path, Operation: "create"})
			addedCount++
		}
	}

	// 2. Удаление файлов, которых нет в новых
	for _, currentFile := range currentFiles {
		if _, exists := newFileMap[currentFile.Path]; !exists {
			log.Debug("Планируется удаление файла", slog.String("path", currentFile.Path))
			operations = append(operations, gitea.BatchOperation{
				Operation: "delete",
				Path:      currentFile.Path,
				SHA:       currentFile.GitSHA, // Git blob SHA для API
			})
			syncedFiles = append(syncedFiles, SyncedFileInfo{Path: currentFile.Path, Operation: "delete"})
			deletedCount++
		}
	}

	// 3. Выполняем все операции атомарно (AC: #5)
	if len(operations) > 0 {
		commitMessage := fmt.Sprintf("Sync workflow files: +%d ~%d -%d", addedCount, updatedCount, deletedCount)

		err := client.SetRepositoryState(ctx, operations, baseBranch, commitMessage)
		if err != nil {
			return addedCount, updatedCount, deletedCount, syncedFiles,
				fmt.Errorf("ошибка атомарного выполнения операций: %w", err)
		}
		log.Info("Атомарные операции выполнены успешно", slog.String("commit_message", commitMessage))
	} else {
		log.Info("Нет изменений для синхронизации")
	}

	return addedCount, updatedCount, deletedCount, syncedFiles, nil
}

// writeSuccess выводит успешный результат (AC: #6, #7).
func (h *ActionMenuHandler) writeSuccess(format, traceID string, start time.Time, data *ActionMenuData) error {
	// Текстовый формат (AC: #7)
	if format != output.FormatJSON {
		return data.writeText(os.Stdout)
	}

	// JSON формат (AC: #6)
	result := &output.Result{
		Status:  output.StatusSuccess,
		Command: constants.ActNRActionMenuBuild,
		Data:    data,
		Metadata: &output.Metadata{
			DurationMs: time.Since(start).Milliseconds(),
			TraceID:    traceID,
			APIVersion: constants.APIVersion,
		},
	}

	writer := output.NewWriter(format)
	return writer.Write(os.Stdout, result)
}

// writeError выводит структурированную ошибку и возвращает error.
func (h *ActionMenuHandler) writeError(format, traceID string, start time.Time, code, message string) error {
	// Текстовый формат — человекочитаемый вывод ошибки
	if format != output.FormatJSON {
		_, _ = fmt.Fprintf(os.Stdout, "Ошибка: %s\nКод: %s\n", message, code)
		return fmt.Errorf("%s: %s", code, message)
	}

	// JSON формат — структурированный вывод
	result := &output.Result{
		Status:  output.StatusError,
		Command: constants.ActNRActionMenuBuild,
		Error: &output.ErrorInfo{
			Code:    code,
			Message: message,
		},
		Metadata: &output.Metadata{
			DurationMs: time.Since(start).Milliseconds(),
			TraceID:    traceID,
			APIVersion: constants.APIVersion,
		},
	}

	writer := output.NewWriter(format)
	if writeErr := writer.Write(os.Stdout, result); writeErr != nil {
		slog.Default().Error("Не удалось записать JSON-ответ об ошибке",
			slog.String("trace_id", traceID),
			slog.String("error", writeErr.Error()))
	}

	return fmt.Errorf("%s: %s", code, message)
}
