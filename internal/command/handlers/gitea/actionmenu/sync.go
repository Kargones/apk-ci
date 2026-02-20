package actionmenu

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"

	"github.com/Kargones/apk-ci/internal/adapter/gitea"
)

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
