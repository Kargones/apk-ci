// Package migratehandler реализует NR-команду nr-migrate для миграции
// Gitea Actions пайплайнов с legacy-команд на NR-команды.
package migratehandler

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/dryrun"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"
)

func RegisterCmd() {
	command.Register(&MigrateHandler{})
}

// MigrateHandler обрабатывает команду nr-migrate — миграция workflow-файлов
// с legacy BR_COMMAND на NR-команды.
type MigrateHandler struct{}

// Name возвращает имя команды.
func (h *MigrateHandler) Name() string {
	return constants.ActNRMigrate
}

// Description возвращает описание команды для вывода в help.
func (h *MigrateHandler) Description() string {
	return "Миграция пайплайнов с legacy-команд на NR-команды"
}

// MigrationReport содержит результаты миграции.
type MigrationReport struct {
	// ScannedFiles — количество просканированных файлов.
	ScannedFiles int `json:"scanned_files"`
	// ModifiedFiles — количество изменённых файлов.
	ModifiedFiles int `json:"modified_files"`
	// TotalReplacements — общее количество замен.
	TotalReplacements int `json:"total_replacements"`
	// DryRun — true если миграция выполнена в dry-run режиме.
	DryRun bool `json:"dry_run"`
	// Replacements — список всех замен.
	Replacements []ReplacementInfo `json:"replacements"`
	// BackupFiles — список созданных backup-файлов.
	BackupFiles []string `json:"backup_files"`
}

// ReplacementInfo содержит информацию об одной замене.
type ReplacementInfo struct {
	// File — путь к файлу.
	File string `json:"file"`
	// Line — номер строки.
	Line int `json:"line"`
	// OldCommand — старое имя команды.
	OldCommand string `json:"old_command"`
	// NewCommand — новое имя команды.
	NewCommand string `json:"new_command"`
}

// Execute выполняет команду nr-migrate.
func (h *MigrateHandler) Execute(ctx context.Context, cfg *config.Config) error {
	start := time.Now()

	traceID := tracing.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = tracing.GenerateTraceID()
	}

	format := os.Getenv("BR_OUTPUT_FORMAT")

	// Story 7.3 AC-8: plan-only для команд без поддержки плана.
	// Review #32: проверяем !IsDryRun() — dry-run имеет приоритет над plan-only (AC-11).
	if !dryrun.IsDryRun() && dryrun.IsPlanOnly() {
		return dryrun.WritePlanOnlyUnsupported(os.Stdout, constants.ActNRMigrate)
	}

	// Чтение параметров из env
	migratePath := os.Getenv(constants.EnvMigratePath)
	if migratePath == "" {
		migratePath = ".gitea/workflows/"
	}

	// Review #32: валидация пути — защита от path traversal через BR_MIGRATE_PATH.
	// filepath.Clean нормализует ".." и подобные конструкции.
	// Review #33: абсолютные пути разрешены намеренно — утилита запускается в CI
	// с доверенными env-переменными, абсолютный путь необходим для указания
	// директории workflow-файлов в произвольном workspace.
	migratePath = filepath.Clean(migratePath)
	if !filepath.IsAbs(migratePath) && strings.HasPrefix(migratePath, "..") {
		return writeError(os.Stdout, format, constants.ActNRMigrate, traceID, start,
			fmt.Errorf("BR_MIGRATE_PATH содержит недопустимый путь: %s (выход за пределы рабочей директории)", migratePath))
	}

	isDryRun := dryrun.IsDryRun()
	noBackup := strings.EqualFold(os.Getenv(constants.EnvMigrateNoBackup), "true") ||
		os.Getenv(constants.EnvMigrateNoBackup) == "1"

	// Построение маппинга legacy → NR из реестра команд
	// TODO(#58): buildLegacyToNRMapping вызывается при каждом Execute.
	// Для утилиты однократного запуска это некритично, но можно оптимизировать через sync.Once.
	legacyToNR := buildLegacyToNRMapping()

	// Сканирование директории
	yamlFiles, err := scanDirectory(migratePath)
	if err != nil {
		return writeError(os.Stdout, format, constants.ActNRMigrate, traceID, start, err)
	}

	report := &MigrationReport{
		ScannedFiles: len(yamlFiles),
		DryRun:       isDryRun,
	}

	// Обработка каждого файла
	for _, filePath := range yamlFiles {
		replacements, scanErr := scanFile(filePath, legacyToNR)
		if scanErr != nil {
			return writeError(os.Stdout, format, constants.ActNRMigrate, traceID, start, scanErr)
		}
		if len(replacements) == 0 {
			continue
		}

		// Добавляем информацию о заменах в отчёт
		for _, r := range replacements {
			report.Replacements = append(report.Replacements, ReplacementInfo{
				File:       filePath,
				Line:       r.Line,
				OldCommand: r.OldCommand,
				NewCommand: r.NewCommand,
			})
		}
		report.TotalReplacements += len(replacements)
		report.ModifiedFiles++

		// Применяем замены если не dry-run
		if !isDryRun {
			if !noBackup {
				if backupErr := backupFile(filePath); backupErr != nil {
					return writeError(os.Stdout, format, constants.ActNRMigrate, traceID, start, backupErr)
				}
				report.BackupFiles = append(report.BackupFiles, filePath+".bak")
			}
			if applyErr := applyReplacements(filePath, replacements); applyErr != nil {
				return writeError(os.Stdout, format, constants.ActNRMigrate, traceID, start, applyErr)
			}
		}
	}

	return writeSuccess(os.Stdout, format, constants.ActNRMigrate, traceID, start, report)
}

// buildLegacyToNRMapping строит маппинг deprecated alias → NR name
// из реестра команд. НЕ хардкодит список — всегда актуален.
func buildLegacyToNRMapping() map[string]string {
	commands := command.ListAllWithAliases()
	result := make(map[string]string, len(commands))
	for _, cmd := range commands {
		if cmd.DeprecatedAlias != "" {
			result[cmd.DeprecatedAlias] = cmd.Name
		}
	}
	return result
}

// writeSuccess выводит успешный результат миграции.
func writeSuccess(w io.Writer, format, cmdName, traceID string, start time.Time, report *MigrationReport) error {
	if format != output.FormatJSON {
		return writeTextReport(w, report)
	}

	result := &output.Result{
		Status:  output.StatusSuccess,
		Command: cmdName,
		Data:    report,
		Metadata: &output.Metadata{
			DurationMs: time.Since(start).Milliseconds(),
			TraceID:    traceID,
			APIVersion: constants.APIVersion,
		},
	}

	writer := output.NewWriter(format)
	return writer.Write(w, result)
}

// writeError выводит структурированную ошибку (JSON) и возвращает оригинальную ошибку
// для корректного exit code в main.go. В текстовом формате вывод ошибки
// делегируется main.go через l.Error() — аналогично другим handler'ам.
func writeError(w io.Writer, format, cmdName, traceID string, start time.Time, err error) error {
	if format == output.FormatJSON {
		result := &output.Result{
			Status:  output.StatusError,
			Command: cmdName,
			Error: &output.ErrorInfo{
				Code:    "MIGRATE.FAILED",
				Message: err.Error(),
			},
			Metadata: &output.Metadata{
				DurationMs: time.Since(start).Milliseconds(),
				TraceID:    traceID,
				APIVersion: constants.APIVersion,
			},
		}

		writer := output.NewWriter(format)
		if writeErr := writer.Write(w, result); writeErr != nil {
			return fmt.Errorf("failed to write error result: %w (original error: %w)", writeErr, err)
		}
	}
	return err
}

// writeTextReport выводит отчёт о миграции в текстовом формате.
func writeTextReport(w io.Writer, report *MigrationReport) error {
	_, err := fmt.Fprintln(w, "Migration Report")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, "================")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "\nScanned: %d files\nModified: %d files\nTotal replacements: %d\n",
		report.ScannedFiles, report.ModifiedFiles, report.TotalReplacements)
	if err != nil {
		return err
	}

	if report.DryRun {
		_, err = fmt.Fprintln(w, "\n[DRY-RUN] Файлы не были изменены")
		if err != nil {
			return err
		}
	}

	// Группировка замен по файлам
	byFile := make(map[string][]ReplacementInfo)
	var fileOrder []string
	for _, r := range report.Replacements {
		if _, seen := byFile[r.File]; !seen {
			fileOrder = append(fileOrder, r.File)
		}
		byFile[r.File] = append(byFile[r.File], r)
	}

	for _, file := range fileOrder {
		replacements := byFile[file]
		_, err = fmt.Fprintf(w, "\nFile: %s\n", file)
		if err != nil {
			return err
		}
		for _, r := range replacements {
			_, err = fmt.Fprintf(w, "  Line %d: BR_COMMAND: %s → %s\n", r.Line, r.OldCommand, r.NewCommand)
			if err != nil {
				return err
			}
		}
	}

	if len(report.BackupFiles) > 0 {
		_, err = fmt.Fprintln(w, "\nBackup files created:")
		if err != nil {
			return err
		}
		for _, f := range report.BackupFiles {
			_, err = fmt.Fprintf(w, "  %s\n", f)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
