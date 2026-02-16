// Package deprecatedaudithandler реализует NR-команду nr-deprecated-audit
// для аудита deprecated кода в проекте apk-ci.
package deprecatedaudithandler

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

func init() {
	command.Register(&DeprecatedAuditHandler{})
}

// DeprecatedAuditHandler обрабатывает команду nr-deprecated-audit —
// аудит deprecated кода: aliases, TODO(H-7), legacy switch-case.
type DeprecatedAuditHandler struct{}

// Name возвращает имя команды.
func (h *DeprecatedAuditHandler) Name() string {
	return constants.ActNRDeprecatedAudit
}

// Description возвращает описание команды для вывода в help.
func (h *DeprecatedAuditHandler) Description() string {
	return "Аудит deprecated кода: aliases, TODO(H-7), legacy switch-case"
}

// Execute выполняет команду nr-deprecated-audit.
func (h *DeprecatedAuditHandler) Execute(ctx context.Context, cfg *config.Config) error {
	start := time.Now()

	traceID := tracing.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = tracing.GenerateTraceID()
	}

	format := os.Getenv("BR_OUTPUT_FORMAT")

	// Story 7.3 AC-8: plan-only для команд без поддержки плана.
	// Review #32: проверяем !IsDryRun() — dry-run имеет приоритет над plan-only (AC-11).
	if !dryrun.IsDryRun() && dryrun.IsPlanOnly() {
		return dryrun.WritePlanOnlyUnsupported(os.Stdout, constants.ActNRDeprecatedAudit)
	}

	// Определяем корневой каталог для сканирования
	rootDir := os.Getenv("BR_AUDIT_ROOT")
	if rootDir == "" {
		rootDir = "."
	}

	// Review #32: валидация пути — защита от path traversal через BR_AUDIT_ROOT.
	// Review #33: абсолютные пути разрешены намеренно — утилита запускается в CI
	// с доверенными env-переменными, абсолютный путь может быть необходим для
	// сканирования workspace вне текущей директории.
	rootDir = filepath.Clean(rootDir)
	if !filepath.IsAbs(rootDir) && strings.HasPrefix(rootDir, "..") {
		return writeError(os.Stdout, format, traceID, start,
			fmt.Errorf("BR_AUDIT_ROOT содержит недопустимый путь: %s (выход за пределы рабочей директории)", rootDir))
	}

	// Путь к main.go для анализа legacy switch-case (относительно rootDir)
	mainGoPath := filepath.Join(rootDir, "cmd", "apk-ci", "main.go")

	// 1. Собрать deprecated aliases из registry
	aliases := collectDeprecatedAliases()

	// 2. Сканировать TODO(H-7) и deprecated маркеры в исходниках
	todos, err := scanTodoComments(rootDir)
	if err != nil {
		return writeError(os.Stdout, format, traceID, start, err)
	}

	// 3. Проанализировать legacy switch-case в main.go
	legacyCases, err := scanLegacySwitchCases(mainGoPath, rootDir, aliases)
	if err != nil {
		return writeError(os.Stdout, format, traceID, start, err)
	}

	// 4. Сформировать отчёт
	report := buildAuditReport(aliases, todos, legacyCases)

	return writeSuccess(os.Stdout, format, traceID, start, report)
}

// writeSuccess выводит успешный результат аудита.
func writeSuccess(w io.Writer, format, traceID string, start time.Time, report *AuditReport) error {
	if format != output.FormatJSON {
		return writeTextReport(w, report)
	}

	result := &output.Result{
		Status:  output.StatusSuccess,
		Command: constants.ActNRDeprecatedAudit,
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

// writeError выводит структурированную ошибку (JSON) и возвращает оригинальную ошибку.
// В текстовом формате вывод ошибки делегируется main.go через l.Error().
func writeError(w io.Writer, format, traceID string, start time.Time, err error) error {
	if format == output.FormatJSON {
		result := &output.Result{
			Status:  output.StatusError,
			Command: constants.ActNRDeprecatedAudit,
			Error: &output.ErrorInfo{
				Code:    "AUDIT.FAILED",
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

// writeTextReport выводит отчёт в текстовом формате.
func writeTextReport(w io.Writer, report *AuditReport) error {
	_, err := fmt.Fprintln(w, "Deprecated Code Audit Report")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, "==============================")
	if err != nil {
		return err
	}

	// Deprecated Aliases
	_, err = fmt.Fprintf(w, "\nDeprecated Aliases (%d):\n", len(report.DeprecatedAliases))
	if err != nil {
		return err
	}
	for _, a := range report.DeprecatedAliases {
		_, err = fmt.Fprintf(w, "  %-24s → %-30s [%s]\n", a.DeprecatedName, a.NRName, a.HandlerPackage)
		if err != nil {
			return err
		}
	}

	// TODO Comments
	_, err = fmt.Fprintf(w, "\nTODO(H-7) Comments (%d):\n", len(report.TodoComments))
	if err != nil {
		return err
	}
	for _, t := range report.TodoComments {
		_, err = fmt.Fprintf(w, "  %s:%d [%s]\n    %s\n", t.File, t.Line, t.Tag, t.Text)
		if err != nil {
			return err
		}
	}

	// Legacy Cases
	_, err = fmt.Fprintf(w, "\nLegacy Switch Cases (%d):\n", len(report.LegacyCases))
	if err != nil {
		return err
	}
	for _, c := range report.LegacyCases {
		note := ""
		if c.Note != "" {
			note = " — " + c.Note
		}
		_, err = fmt.Fprintf(w, "  %s:%d  case %-24s%s\n", c.File, c.Line, c.CaseValue, note)
		if err != nil {
			return err
		}
	}

	// Summary
	_, err = fmt.Fprintf(w, "\nSummary:\n")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "  Deprecated aliases: %d\n", report.Summary.TotalDeprecatedAliases)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "  TODO(H-7) comments: %d\n", report.Summary.TotalTodoH7)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "  Legacy switch cases: %d\n", report.Summary.TotalLegacyCases)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "  Ready for removal: %s\n", report.Summary.ReadyForRemoval)
	if err != nil {
		return err
	}
	if report.Summary.Message != "" {
		_, err = fmt.Fprintf(w, "  %s\n", report.Summary.Message)
		if err != nil {
			return err
		}
	}

	return nil
}
