package extensionpublishhandler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// PublishStatus представляет статус публикации расширения в репозиторий.
type PublishStatus string

const (
	// StatusSuccess — успешная публикация (PR создан)
	StatusSuccess PublishStatus = "success"
	// StatusFailed — ошибка публикации
	StatusFailed PublishStatus = "failed"
	// StatusSkipped — пропущено (репозиторий недоступен, dry-run, уже обновлён)
	StatusSkipped PublishStatus = "skipped"
)

// PublishResult представляет результат публикации расширения для одного подписчика.
// Содержит информацию о репозитории подписчика, созданном PR и возможных ошибках.
type PublishResult struct {
	// Subscriber — целевой репозиторий подписчика
	Subscriber SubscribedRepo `json:"subscriber"`

	// Status — статус публикации (success/failed/skipped)
	Status PublishStatus `json:"status"`

	// SyncResult — результат синхронизации файлов (не сериализуется в JSON)
	SyncResult *SyncResult `json:"-"`

	// PRNumber — номер созданного PR (0 если PR не создан)
	PRNumber int `json:"pr_number,omitempty"`

	// PRURL — URL созданного PR
	PRURL string `json:"pr_url,omitempty"`

	// Error — ошибка при публикации (не сериализуется в JSON)
	Error error `json:"-"`

	// ErrorMessage — человекочитаемое описание ошибки (для JSON)
	ErrorMessage string `json:"error,omitempty"`

	// DurationMs — время выполнения операции в миллисекундах
	DurationMs int64 `json:"duration_ms"`
}

// PublishReport представляет полный отчёт о публикации расширения.
// Содержит информацию об источнике, времени выполнения и результатах для каждого подписчика.
type PublishReport struct {
	// ExtensionName — имя публикуемого расширения
	ExtensionName string `json:"extension_name"`

	// Version — версия расширения
	Version string `json:"version"`

	// SourceRepo — полное имя исходного репозитория (owner/repo)
	SourceRepo string `json:"source_repo"`

	// StartTime — время начала публикации
	StartTime time.Time `json:"start_time"`

	// EndTime — время завершения публикации
	EndTime time.Time `json:"end_time"`

	// Results — результаты для каждого подписчика
	Results []PublishResult `json:"results"`
}

// SuccessCount возвращает количество успешных публикаций.
func (r *PublishReport) SuccessCount() int {
	count := 0
	for _, res := range r.Results {
		if res.Status == StatusSuccess {
			count++
		}
	}
	return count
}

// FailedCount возвращает количество неудачных публикаций.
func (r *PublishReport) FailedCount() int {
	count := 0
	for _, res := range r.Results {
		if res.Status == StatusFailed {
			count++
		}
	}
	return count
}

// SkippedCount возвращает количество пропущенных публикаций.
func (r *PublishReport) SkippedCount() int {
	count := 0
	for _, res := range r.Results {
		if res.Status == StatusSkipped {
			count++
		}
	}
	return count
}

// HasErrors возвращает true, если есть хотя бы одна неудачная публикация.
func (r *PublishReport) HasErrors() bool {
	return r.FailedCount() > 0
}

// TotalDuration возвращает общую длительность операции.
func (r *PublishReport) TotalDuration() time.Duration {
	return r.EndTime.Sub(r.StartTime)
}

// ReportJSONOutput структура для JSON-сериализации отчёта с summary.
type ReportJSONOutput struct {
	ExtensionName string          `json:"extension_name"`
	Version       string          `json:"version"`
	SourceRepo    string          `json:"source_repo"`
	StartTime     time.Time       `json:"start_time"`
	EndTime       time.Time       `json:"end_time"`
	Results       []PublishResult `json:"results"`
	Summary       ReportSummary   `json:"summary"`
}

// ReportSummary содержит итоговую статистику для JSON-вывода.
type ReportSummary struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

// ReportResults выводит структурированный отчёт о публикации.
// При BR_OUTPUT_JSON=true выводит JSON в stdout, иначе — форматированный текст в лог.
//
// Параметры:
//   - report: отчёт о публикации
//   - l: логгер для текстового вывода
//
// Возвращает:
//   - error: ошибка сериализации JSON или nil при успехе
func ReportResults(report *PublishReport, l *slog.Logger) error {
	outputJSON := os.Getenv("BR_OUTPUT_JSON") == "true"

	if outputJSON {
		return reportResultsJSON(report)
	}

	reportResultsText(report, l)
	return nil
}

// reportResultsJSON выводит отчёт в формате JSON в stdout.
func reportResultsJSON(report *PublishReport) error {
	output := ReportJSONOutput{
		ExtensionName: report.ExtensionName,
		Version:       report.Version,
		SourceRepo:    report.SourceRepo,
		StartTime:     report.StartTime,
		EndTime:       report.EndTime,
		Results:       report.Results,
		Summary: ReportSummary{
			Total:   len(report.Results),
			Success: report.SuccessCount(),
			Failed:  report.FailedCount(),
			Skipped: report.SkippedCount(),
		},
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// reportResultsText выводит форматированный текстовый отчёт в лог.
func reportResultsText(report *PublishReport, l *slog.Logger) {
	// Заголовок
	l.Info("═══════════════════════════════════════════════════════════════")
	l.Info("               EXTENSION PUBLISH REPORT")
	l.Info("═══════════════════════════════════════════════════════════════")
	l.Info(fmt.Sprintf("Extension: %s", report.ExtensionName))
	l.Info(fmt.Sprintf("Version:   %s", report.Version))
	l.Info(fmt.Sprintf("Source:    %s", report.SourceRepo))
	l.Info(fmt.Sprintf("Duration:  %.1fs", report.TotalDuration().Seconds()))
	l.Info("")

	// Успешные публикации
	successCount := report.SuccessCount()
	if successCount > 0 {
		l.Info("─────────────────────────────────────────────────────────────")
		l.Info(fmt.Sprintf("✓ SUCCESS (%d)", successCount))
		l.Info("─────────────────────────────────────────────────────────────")
		for _, res := range report.Results {
			if res.Status == StatusSuccess {
				target := fmt.Sprintf("%s/%s", res.Subscriber.Organization, res.Subscriber.Repository)
				l.Info(fmt.Sprintf("  • %s → PR #%d (%s)", target, res.PRNumber, res.PRURL))
			}
		}
		l.Info("")
	}

	// Неудачные публикации
	failedCount := report.FailedCount()
	if failedCount > 0 {
		l.Info("─────────────────────────────────────────────────────────────")
		l.Info(fmt.Sprintf("✗ FAILED (%d)", failedCount))
		l.Info("─────────────────────────────────────────────────────────────")
		for _, res := range report.Results {
			if res.Status == StatusFailed {
				target := fmt.Sprintf("%s/%s", res.Subscriber.Organization, res.Subscriber.Repository)
				l.Info(fmt.Sprintf("  • %s: %s", target, res.ErrorMessage))
			}
		}
		l.Info("")
	}

	// Пропущенные публикации
	skippedCount := report.SkippedCount()
	if skippedCount > 0 {
		l.Info("─────────────────────────────────────────────────────────────")
		l.Info(fmt.Sprintf("○ SKIPPED (%d)", skippedCount))
		l.Info("─────────────────────────────────────────────────────────────")
		for _, res := range report.Results {
			if res.Status == StatusSkipped {
				target := fmt.Sprintf("%s/%s", res.Subscriber.Organization, res.Subscriber.Repository)
				reason := res.ErrorMessage
				if reason == "" {
					reason = "dry-run mode"
				}
				l.Info(fmt.Sprintf("  • %s: %s", target, reason))
			}
		}
		l.Info("")
	}

	// Итоговая статистика
	l.Info("═══════════════════════════════════════════════════════════════")
	l.Info(fmt.Sprintf("SUMMARY: %d success, %d failed, %d skipped",
		successCount, failedCount, skippedCount))
	l.Info("═══════════════════════════════════════════════════════════════")
}
