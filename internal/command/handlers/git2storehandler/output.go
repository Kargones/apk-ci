// Package git2storehandler — output formatting and plan building for git2store.
package git2storehandler

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/dryrun"
	"github.com/Kargones/apk-ci/internal/pkg/output"
)

// writeText выводит результат в человекочитаемом формате (AC-5).
func (d *Git2StoreData) writeText(w io.Writer) error {
	statusText := "успешно"
	if !d.StateChanged {
		statusText = "без изменений"
	}

	_, err := fmt.Fprintf(w, "Синхронизация Git → хранилище 1C: %s\n", statusText)
	if err != nil {
		return err
	}

	completedCount := 0
	for _, stage := range d.StagesCompleted {
		if stage.Success {
			completedCount++
		}
	}

	if _, err = fmt.Fprintf(w, "\nПрогресс: [%d/%d] этапов\n", completedCount, len(allStages)); err != nil {
		return err
	}

	if _, err = fmt.Fprintf(w, "\nСводка:\n"); err != nil {
		return err
	}

	if d.BackupPath != "" {
		if _, err = fmt.Fprintf(w, "  Резервная копия: %s\n", d.BackupPath); err != nil {
			return err
		}
	}

	if _, err = fmt.Fprintf(w, "  Текущий этап: %s\n", d.StageCurrent); err != nil {
		return err
	}

	if _, err = fmt.Fprintf(w, "  Длительность: %d мс\n", d.DurationMs); err != nil {
		return err
	}

	if _, err = fmt.Fprintf(w, "\nЭтапы:\n"); err != nil {
		return err
	}

	for _, stage := range d.StagesCompleted {
		status := "✓"
		if !stage.Success {
			status = "✗"
		}
		if _, err = fmt.Fprintf(w, "  %s %s (%d мс)\n", status, stage.Name, stage.DurationMs); err != nil {
			return err
		}
		if stage.Error != "" {
			if _, err = fmt.Fprintf(w, "    Ошибка: %s\n", stage.Error); err != nil {
				return err
			}
		}
	}

	return nil
}

// buildPlan создаёт план операций для предпросмотра.
// Используется в dry-run, plan-only и verbose режимах.
// Story 7.3: извлечено из executeDryRun для переиспользования.
func (h *Git2StoreHandler) buildPlan(cfg *config.Config) *output.DryRunPlan {
	storeRoot := constants.StoreRoot + cfg.Owner + "/" + cfg.Repo

	extensions := ""
	if len(cfg.AddArray) > 0 {
		extensions = fmt.Sprintf("%v", cfg.AddArray)
	}

	steps := []output.PlanStep{
		{
			Order:     1,
			Operation: StageValidating,
			Parameters: map[string]any{
				"owner":      cfg.Owner,
				"repo":       cfg.Repo,
				"infobase":   cfg.InfobaseName,
				"store_root": storeRoot,
			},
			ExpectedChanges: []string{"Проверка параметров конфигурации"},
		},
		{
			Order:     2,
			Operation: StageCreatingBackup,
			Parameters: map[string]any{
				"store_root": storeRoot,
			},
			ExpectedChanges: []string{"Создание резервной копии метаданных хранилища"},
		},
		{
			Order:     3,
			Operation: StageCloning,
			Parameters: map[string]any{
				"owner": cfg.Owner,
				"repo":  cfg.Repo,
			},
			ExpectedChanges: []string{"Клонирование Git-репозитория"},
		},
		{
			Order:     4,
			Operation: StageCheckoutEdt,
			Parameters: map[string]any{
				"branch": constants.EdtBranch,
			},
			ExpectedChanges: []string{"Переключение на ветку EDT"},
		},
		{
			Order:           5,
			Operation:       StageCreatingTempDb,
			ExpectedChanges: []string{"Создание временной базы данных 1C"},
		},
		{
			Order:           6,
			Operation:       StageLoadingConfig,
			ExpectedChanges: []string{"Загрузка конфигурации EDT в временную БД"},
		},
		{
			Order:     7,
			Operation: StageCheckoutXml,
			Parameters: map[string]any{
				"branch": constants.OneCBranch,
			},
			ExpectedChanges: []string{"Переключение на ветку XML"},
		},
		{
			Order:           8,
			Operation:       StageInitDb,
			ExpectedChanges: []string{"Инициализация БД из XML-конфигурации"},
		},
		{
			Order:     9,
			Operation: StageUnbinding,
			Parameters: map[string]any{
				"store_root": storeRoot,
			},
			ExpectedChanges: []string{"Отвязка хранилища конфигурации от БД"},
		},
		{
			Order:           10,
			Operation:       StageLoadingDb,
			ExpectedChanges: []string{"Загрузка конфигурации в БД"},
		},
		{
			Order:           11,
			Operation:       StageUpdatingDb1,
			ExpectedChanges: []string{"Обновление структуры БД (первый проход)"},
		},
		{
			Order:           12,
			Operation:       StageDumpingDb,
			ExpectedChanges: []string{"Выгрузка конфигурации из БД"},
		},
		{
			Order:     13,
			Operation: StageBinding,
			Parameters: map[string]any{
				"store_root": storeRoot,
			},
			ExpectedChanges: []string{"Привязка хранилища конфигурации к БД"},
		},
		{
			Order:           14,
			Operation:       StageUpdatingDb2,
			ExpectedChanges: []string{"Обновление структуры БД (второй проход, расширения)"},
		},
		{
			Order:     15,
			Operation: StageLocking,
			Parameters: map[string]any{
				"store_root": storeRoot,
			},
			ExpectedChanges: []string{"Захват объектов в хранилище"},
		},
		{
			Order:           16,
			Operation:       StageMerging,
			ExpectedChanges: []string{"Объединение конфигураций"},
		},
		{
			Order:           17,
			Operation:       StageUpdatingDb3,
			ExpectedChanges: []string{"Обновление структуры БД (третий проход)"},
		},
		{
			Order:     18,
			Operation: StageCommitting,
			Parameters: map[string]any{
				"store_root": storeRoot,
			},
			ExpectedChanges: []string{"Помещение изменений в хранилище"},
		},
	}

	if extensions != "" {
		steps[0].Parameters["extensions"] = extensions
	}

	return dryrun.BuildPlanWithSummary(
		constants.ActNRGit2store,
		steps,
		fmt.Sprintf("Git → Store синхронизация: %s/%s → %s (%d этапов)", cfg.Owner, cfg.Repo, storeRoot, len(steps)),
	)
}

// executeDryRun выводит план workflow без выполнения (AC-14).
func (h *Git2StoreHandler) executeDryRun(cfg *config.Config, format, traceID string, start time.Time, log *slog.Logger) error {
	log.Info("dry_run: вывод плана без выполнения")

	plan := h.buildPlan(cfg)
	return output.WriteDryRunResult(os.Stdout, format, constants.ActNRGit2store, traceID, constants.APIVersion, start, plan)
}

// writeDryRunResult перенесён в output.WriteDryRunResult (CR-7.3 #3).

// writeSuccess выводит успешный результат (AC-4, AC-5).
func (h *Git2StoreHandler) writeSuccess(format, traceID string, data *Git2StoreData) error {
	if format != output.FormatJSON {
		return data.writeText(os.Stdout)
	}

	result := &output.Result{
		Status:  output.StatusSuccess,
		Command: constants.ActNRGit2store,
		Data:    data,
		Plan:    h.verbosePlan,
		Metadata: &output.Metadata{
			DurationMs: data.DurationMs,
			TraceID:    traceID,
			APIVersion: constants.APIVersion,
		},
	}

	writer := output.NewWriter(format)
	return writer.Write(os.Stdout, result)
}

// writePlanOnlyResult перенесён в output.WritePlanOnlyResult (CR-7.3 #2).

// writeStageError выводит структурированную ошибку и возвращает error (AC-7, AC-11).
func (h *Git2StoreHandler) writeStageError(format, traceID string, start time.Time, data *Git2StoreData, stageErr error) error {
	data.DurationMs = time.Since(start).Milliseconds()

	if format != output.FormatJSON {
		if data.BackupPath != "" {
			return fmt.Errorf("%s (backup: %s)", stageErr.Error(), data.BackupPath)
		}
		return stageErr
	}

	errStr := stageErr.Error()
	code := "ERR_GIT2STORE"
	message := errStr

	if colonIdx := len("ERR_GIT2STORE"); colonIdx < len(errStr) && errStr[colonIdx] == '_' {
		for i := 0; i < len(errStr); i++ {
			if errStr[i] == ':' {
				code = errStr[:i]
				if i+2 < len(errStr) {
					message = errStr[i+2:]
				}
				break
			}
		}
	}

	data.Errors = []string{message}

	result := &output.Result{
		Status:  output.StatusError,
		Command: constants.ActNRGit2store,
		Data:    data,
		Error: &output.ErrorInfo{
			Code:    code,
			Message: message,
		},
		Metadata: &output.Metadata{
			DurationMs: data.DurationMs,
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

	return stageErr
}
