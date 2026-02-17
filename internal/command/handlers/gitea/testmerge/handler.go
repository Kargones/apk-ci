// Package testmerge реализует NR-команду nr-test-merge
// для проверки конфликтов слияния всех открытых PR.
package testmerge

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Kargones/apk-ci/internal/adapter/gitea"
	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/command/handlers/gitea/shared"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/dryrun"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"
	errhandler "github.com/Kargones/apk-ci/internal/command/handlers/shared"
)

// testBranchPrefix — префикс для имени тестовой ветки.
const testBranchPrefix = "test-merge-"

// generateTestBranchName генерирует уникальное имя тестовой ветки с timestamp.
func generateTestBranchName() string {
	return testBranchPrefix + time.Now().Format("20060102-150405")
}

// Коды ошибок — используем shared константы для соблюдения DRY.
// Локальные алиасы для краткости.
const (
	errConfigMissing    = shared.ErrConfigMissing
	errMissingOwnerRepo = shared.ErrMissingOwnerRepo
	errGiteaAPI         = shared.ErrGiteaAPI
	errBranchCreate     = shared.ErrBranchCreate
)

// init регистрирует команду nr-test-merge с deprecated alias test-merge.
// TODO(#61): Deprecated alias "test-merge" будет удалён в v2.0.0 / Epic 7.
// После полной миграции на NR-архитектуру, использовать только "nr-test-merge".
func RegisterCmd() error {
	return command.RegisterWithAlias(&TestMergeHandler{}, constants.ActTestMerge)
}

// TestMergeData содержит результат проверки конфликтов слияния.
type TestMergeData struct {
	// TotalPRs — общее количество проверенных PR
	TotalPRs int `json:"total_prs"`
	// MergeablePRs — количество PR без конфликтов
	MergeablePRs int `json:"mergeable_prs"`
	// ConflictPRs — количество PR с конфликтами
	ConflictPRs int `json:"conflict_prs"`
	// ClosedPRs — количество закрытых PR из-за конфликтов
	ClosedPRs int `json:"closed_prs"`
	// PRResults — детальные результаты для каждого PR
	PRResults []PRMergeResult `json:"pr_results"`
	// TestBranch — имя использованной тестовой ветки
	TestBranch string `json:"test_branch"`
	// BaseBranch — базовая ветка для тестирования
	BaseBranch string `json:"base_branch"`
}

// PRMergeResult содержит результат проверки одного PR.
type PRMergeResult struct {
	// PRNumber — номер PR в репозитории
	PRNumber int64 `json:"pr_number"`
	// HeadBranch — исходная ветка PR
	HeadBranch string `json:"head_branch"`
	// BaseBranch — целевая ветка PR
	BaseBranch string `json:"base_branch"`
	// HasConflict — есть ли конфликт
	HasConflict bool `json:"has_conflict"`
	// MergeResult — результат попытки merge ("success", "conflict", "error", "merge_failed")
	MergeResult string `json:"merge_result"`
	// ConflictFiles — список файлов с конфликтами (если есть)
	ConflictFiles []string `json:"conflict_files,omitempty"`
	// Closed — был ли PR закрыт из-за конфликта
	Closed bool `json:"closed"`
	// ErrorMessage — сообщение об ошибке (если есть)
	ErrorMessage string `json:"error_message,omitempty"`
}

// truncateString обрезает строку до указанной длины с учётом Unicode символов.
func truncateString(s string, maxLen int) string {
	runeCount := utf8.RuneCountInString(s)
	if runeCount <= maxLen {
		return s
	}
	if maxLen <= 3 {
		runes := []rune(s)
		return string(runes[:maxLen])
	}
	runes := []rune(s)
	return string(runes[:maxLen-3]) + "..."
}

// writeText выводит результаты проверки конфликтов в человекочитаемом формате.
func (d *TestMergeData) writeText(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "══════════════════════════════════════════════════════\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Проверка конфликтов слияния\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "══════════════════════════════════════════════════════\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Базовая ветка: %s\n\n", d.BaseBranch); err != nil {
		return err
	}

	if d.TotalPRs == 0 {
		if _, err := fmt.Fprintf(w, "Нет открытых Pull Requests для проверки.\n"); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "══════════════════════════════════════════════════════\n"); err != nil {
			return err
		}
		return nil
	}

	if _, err := fmt.Fprintf(w, "Результаты проверки:\n\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "| PR # | Ветка           | Статус      | Конфликтные файлы    |\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "|------|-----------------|-------------|----------------------|\n"); err != nil {
		return err
	}

	for _, pr := range d.PRResults {
		status := "OK"
		conflictFiles := ""
		if pr.HasConflict {
			status = "CONFLICT"
			if len(pr.ConflictFiles) > 0 {
				conflictFiles = strings.Join(pr.ConflictFiles, ", ")
				if len(conflictFiles) > 20 {
					conflictFiles = conflictFiles[:17] + "..."
				}
			}
		}
		if _, err := fmt.Fprintf(w, "| #%-4d | %-15s | %-11s | %-20s |\n",
			pr.PRNumber, truncateString(pr.HeadBranch, 15), status, conflictFiles); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "\n══════════════════════════════════════════════════════\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Итого: %d PR проверено\n", d.TotalPRs); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  Без конфликтов: %d\n", d.MergeablePRs); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  С конфликтами: %d (закрыто: %d)\n", d.ConflictPRs, d.ClosedPRs); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "══════════════════════════════════════════════════════\n"); err != nil {
		return err
	}

	return nil
}

// TestMergeHandler обрабатывает команду nr-test-merge.
type TestMergeHandler struct {
	// giteaClient — клиент для работы с Gitea API.
	// Может быть nil в production (требуется реализация фабрики).
	// В тестах инъектируется напрямую.
	giteaClient gitea.Client
}

// Name возвращает имя команды.
func (h *TestMergeHandler) Name() string {
	return constants.ActNRTestMerge
}

// Description возвращает описание команды для вывода в help.
func (h *TestMergeHandler) Description() string {
	return "Проверить конфликты слияния для всех открытых PR"
}

// Execute выполняет команду nr-test-merge.
func (h *TestMergeHandler) Execute(ctx context.Context, cfg *config.Config) error {
	start := time.Now()

	traceID := tracing.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = tracing.GenerateTraceID()
	}

	format := os.Getenv("BR_OUTPUT_FORMAT")

	// Story 7.3 AC-8: plan-only для команд без поддержки плана
	// Review #36: !IsDryRun() — dry-run имеет приоритет над plan-only (AC-11).
	if !dryrun.IsDryRun() && dryrun.IsPlanOnly() {
		return dryrun.WritePlanOnlyUnsupported(os.Stdout, constants.ActNRTestMerge)
	}

	log := slog.Default().With(slog.String("trace_id", traceID), slog.String("command", constants.ActNRTestMerge))

	// 1. Валидация конфигурации (AC: #1)
	if cfg == nil {
		log.Error("Конфигурация не загружена")
		return h.writeError(format, traceID, start,
			errConfigMissing,
			"Конфигурация не загружена")
	}

	// 2. Получение и валидация Owner/Repo (AC: #2, #9)
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

	log.Info("Запуск проверки конфликтов слияния",
		slog.String("owner", owner),
		slog.String("repo", repo),
		slog.String("base_branch", baseBranch))

	// Получение Gitea клиента (AC: #9)
	// TODO(#58): Реализовать фабрику createGiteaClient(cfg) для создания реального клиента.
	// Текущая реализация требует DI через поле giteaClient (используется в тестах).
	client := h.giteaClient
	if client == nil {
		log.Error("Gitea клиент не настроен")
		return h.writeError(format, traceID, start,
			errConfigMissing,
			"Gitea клиент не настроен — требуется реализация фабрики createGiteaClient()")
	}

	// 3. Получение списка открытых PR (AC: #2)
	activePRs, err := client.ListOpenPRs(ctx)
	if err != nil {
		log.Error("Не удалось получить список открытых PR", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start,
			errGiteaAPI,
			"Не удалось получить список открытых PR: "+err.Error())
	}

	// Генерируем уникальное имя тестовой ветки с timestamp (AC: #3)
	testBranchName := generateTestBranchName()

	if len(activePRs) == 0 {
		log.Info("Нет открытых PR")
		return h.writeSuccess(format, traceID, start, &TestMergeData{
			TotalPRs:   0,
			PRResults:  []PRMergeResult{}, // Пустой массив вместо nil для JSON
			TestBranch: testBranchName,
			BaseBranch: baseBranch,
		})
	}

	log.Debug("Найдено открытых PR", slog.Int("count", len(activePRs)))

	// 4. Cleanup + создание тестовой ветки (AC: #3, #10)
	if delErr := client.DeleteBranch(ctx, testBranchName); delErr != nil {
		log.Debug("test branch cleanup (may not exist)", slog.String("error", delErr.Error()))
	}
	err = client.CreateBranch(ctx, testBranchName, baseBranch)
	if err != nil {
		log.Error("Не удалось создать тестовую ветку", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start,
			errBranchCreate,
			"Не удалось создать тестовую ветку: "+err.Error())
	}

	// Гарантируем удаление тестовой ветки после завершения (AC: #10)
	defer func() {
		if delErr := client.DeleteBranch(ctx, testBranchName); delErr != nil {
			log.Warn("Не удалось удалить тестовую ветку", slog.String("error", delErr.Error()))
		}
	}()

	log.Debug("Тестовая ветка создана", slog.String("branch", testBranchName), slog.String("base", baseBranch))

	// 5. Проверка каждого PR (AC: #4, #5, #6)
	data := &TestMergeData{
		TotalPRs:   len(activePRs),
		TestBranch: testBranchName,
		BaseBranch: baseBranch,
		PRResults:  make([]PRMergeResult, 0, len(activePRs)),
	}

	for _, pr := range activePRs {
		result := h.checkPR(ctx, client, pr, baseBranch, log)
		data.PRResults = append(data.PRResults, result)

		if result.HasConflict {
			data.ConflictPRs++
			if result.Closed {
				data.ClosedPRs++
			}
		} else {
			data.MergeablePRs++
		}
	}

	log.Info("Проверка конфликтов слияния завершена",
		slog.Int("total", data.TotalPRs),
		slog.Int("mergeable", data.MergeablePRs),
		slog.Int("conflicts", data.ConflictPRs),
		slog.Int("closed", data.ClosedPRs))

	return h.writeSuccess(format, traceID, start, data)
}

// checkPR проверяет один PR на конфликты.
// baseBranch используется для формирования комментария при закрытии PR.
func (h *TestMergeHandler) checkPR(ctx context.Context, client gitea.Client, pr gitea.PR, baseBranch string, log *slog.Logger) PRMergeResult {
	result := PRMergeResult{
		PRNumber:   pr.Number,
		HeadBranch: pr.Head,
		BaseBranch: pr.Base,
	}

	log.Debug("Проверка PR", slog.Int64("number", pr.Number), slog.String("head", pr.Head))

	// Создаём временный PR в тестовую ветку (AC: #5)
	testPR, err := client.CreatePR(ctx, pr.Head)
	if err != nil {
		log.Warn("Не удалось создать тестовый PR", slog.Int64("number", pr.Number), slog.String("error", err.Error()))
		result.HasConflict = true
		result.MergeResult = "error"
		result.ErrorMessage = err.Error()
		return result
	}

	// Проверяем конфликты (AC: #4)
	hasConflict, err := client.ConflictPR(ctx, testPR.Number)
	if err != nil {
		log.Warn("Ошибка проверки конфликта", slog.Int64("number", testPR.Number), slog.String("error", err.Error()))
		hasConflict = true // Assume conflict on error
	}

	if hasConflict {
		result.HasConflict = true
		result.MergeResult = "conflict"

		// Получаем список конфликтных файлов
		conflictFiles, conflictErr := client.ConflictFilesPR(ctx, testPR.Number)
		if conflictErr != nil {
			log.Warn("Не удалось получить список конфликтных файлов", slog.String("error", conflictErr.Error()))
		}
		result.ConflictFiles = conflictFiles

		// Добавляем комментарий о причине закрытия (AC: #6)
		commentText := h.buildConflictComment(baseBranch, conflictFiles)
		if commentErr := client.AddIssueComment(ctx, pr.Number, commentText); commentErr != nil {
			log.Warn("Не удалось добавить комментарий к PR", slog.Int64("number", pr.Number), slog.String("error", commentErr.Error()))
		}

		// Закрываем оригинальный PR (AC: #6)
		if closeErr := client.ClosePR(ctx, pr.Number); closeErr == nil {
			result.Closed = true
			log.Debug("Закрыт конфликтный PR", slog.Int64("number", pr.Number))
		}

		return result
	}

	// Пытаемся выполнить тестовый merge (AC: #5)
	if mergeErr := client.MergePR(ctx, testPR.Number); mergeErr != nil {
		log.Warn("Ошибка слияния", slog.Int64("number", testPR.Number), slog.String("error", mergeErr.Error()))
		result.HasConflict = true
		result.MergeResult = "merge_failed"
		result.ErrorMessage = mergeErr.Error()

		// Добавляем комментарий о причине закрытия (AC: #6)
		commentText := fmt.Sprintf("PR закрыт автоматически: ошибка слияния с веткой `%s`.\n\nОшибка: %s", baseBranch, mergeErr.Error())
		if commentErr := client.AddIssueComment(ctx, pr.Number, commentText); commentErr != nil {
			log.Warn("Не удалось добавить комментарий к PR", slog.Int64("number", pr.Number), slog.String("error", commentErr.Error()))
		}

		// Закрываем оригинальный PR (AC: #6)
		if closeErr := client.ClosePR(ctx, pr.Number); closeErr == nil {
			result.Closed = true
		}

		return result
	}

	result.HasConflict = false
	result.MergeResult = "success"
	return result
}

// buildConflictComment формирует текст комментария о конфликте для закрытия PR.
func (h *TestMergeHandler) buildConflictComment(baseBranch string, conflictFiles []string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("PR закрыт автоматически: обнаружены конфликты слияния с веткой `%s`.\n\n", baseBranch))

	if len(conflictFiles) > 0 {
		sb.WriteString("**Конфликтные файлы:**\n")
		for _, file := range conflictFiles {
			sb.WriteString(fmt.Sprintf("- `%s`\n", file))
		}
	} else {
		sb.WriteString("Не удалось определить список конфликтных файлов.\n")
	}

	sb.WriteString("\nПожалуйста, разрешите конфликты и создайте новый PR.")
	return sb.String()
}

// writeSuccess выводит успешный результат.
func (h *TestMergeHandler) writeSuccess(format, traceID string, start time.Time, data *TestMergeData) error {
	// Текстовый формат (AC: #8)
	if format != output.FormatJSON {
		return data.writeText(os.Stdout)
	}

	// JSON формат (AC: #7)
	result := &output.Result{
		Status:  output.StatusSuccess,
		Command: constants.ActNRTestMerge,
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
func (h *TestMergeHandler) writeError(format, traceID string, start time.Time, code, message string) error {
	// Текстовый формат — человекочитаемый вывод ошибки
	if format != output.FormatJSON {
		return errhandler.HandleError(message, code)
	}

	// JSON формат — структурированный вывод
	result := &output.Result{
		Status:  output.StatusError,
		Command: constants.ActNRTestMerge,
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
