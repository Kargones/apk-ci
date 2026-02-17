// Package git2storehandler — factory methods and backup production implementation.
package git2storehandler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
)

// createGit создаёт GitOperator через фабрику или production реализацию.
func (h *Git2StoreHandler) createGit(l *slog.Logger, cfg *config.Config) (GitOperator, error) {
	if h.gitFactory != nil {
		return h.gitFactory.CreateGit(l, cfg)
	}
	return createGitProduction(l, cfg)
}

// createConvertConfig создаёт ConvertConfigOperator через фабрику или production реализацию.
func (h *Git2StoreHandler) createConvertConfig() ConvertConfigOperator {
	if h.convertConfigFactory != nil {
		return h.convertConfigFactory.CreateConvertConfig()
	}
	return createConvertConfigProduction()
}

// createBackup создаёт резервную копию хранилища через интерфейс или production реализацию.
func (h *Git2StoreHandler) createBackup(cfg *config.Config, storeRoot string) (string, error) {
	if h.backupCreator != nil {
		return h.backupCreator.CreateBackup(cfg, storeRoot)
	}
	return createBackupProduction(cfg, storeRoot)
}

// createTempDb создаёт временную БД через интерфейс или production реализацию.
func (h *Git2StoreHandler) createTempDb(ctx context.Context, l *slog.Logger, cfg *config.Config) (string, error) {
	if h.tempDbCreator != nil {
		return h.tempDbCreator.CreateTempDb(ctx, l, cfg)
	}
	return createTempDbProduction(ctx, l, cfg)
}

// createBackupProduction создаёт резервную копию хранилища (production реализация).
//
// TODO: Реализовать полноценный backup хранилища 1C.
// Текущая реализация создаёт только метаданные (backup_info.txt) с информацией
// для ручного восстановления. Полный backup требует:
// 1. Вызов 1cv8 DESIGNER /ConfigurationRepositoryDumpCfg для экспорта конфигурации
// 2. Сохранение версии хранилища через /ConfigurationRepositoryReport
// 3. Копирование локальных файлов (если хранилище файловое)
// Это требует расширения convert.Config или отдельного BackupService.
func createBackupProduction(cfg *config.Config, storeRoot string) (string, error) {
	backupDir := filepath.Join(cfg.TmpDir, "backup_"+time.Now().Format("20060102_150405"))
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("не удалось создать директорию backup: %w", err)
	}

	backupInfoPath := filepath.Join(backupDir, "backup_info.txt")
	backupInfo := fmt.Sprintf("=== BACKUP METADATA ===\n"+
		"Store Root: %s\n"+
		"Created: %s\n"+
		"Infobase: %s\n"+
		"Owner: %s\n"+
		"Repo: %s\n"+
		"\nВНИМАНИЕ: Это метаданные для ручного восстановления.\n"+
		"Для восстановления используйте 1cv8 DESIGNER или ibcmd.\n",
		storeRoot, time.Now().Format(time.RFC3339),
		cfg.InfobaseName, cfg.Owner, cfg.Repo)
	if err := os.WriteFile(backupInfoPath, []byte(backupInfo), 0644); err != nil {
		return "", fmt.Errorf("не удалось записать информацию о backup: %w", err)
	}

	return backupDir, nil
}
