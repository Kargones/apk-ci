// Package reportbranch реализует NR-команду nr-sq-report-branch
// для генерации отчёта о качестве ветки из SonarQube.
package reportbranch

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/sonarqube"
	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/command/handlers/sonarqube/shared"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/dryrun"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"
)

// Коды ошибок — используем shared константы.
// Локальные алиасы для краткости.
const (
	errBranchMissing    = shared.ErrBranchMissing
	errProjectNotFound  = shared.ErrProjectNotFound
	errSonarQubeAPI     = shared.ErrSonarQubeAPI
	errConfigMissing    = shared.ErrConfigMissing
	errMissingOwnerRepo = shared.ErrMissingOwnerRepo
)

func init() {
	// Deprecated alias: "sq-report-branch" -> "nr-sq-report-branch"
	// Legacy команда сохраняется для обратной совместимости до полной миграции на NR.
	// TODO(H-7): Удалить deprecated alias ActSQReportBranch после миграции всех workflows на NR-команды.
	// Планируемая версия удаления: v2.0.0 или после завершения Epic 7.
	command.RegisterWithAlias(&ReportBranchHandler{}, constants.ActSQReportBranch)
}

// BranchReportData содержит отчёт о качестве ветки.
//
// ВАЖНО: Metrics и IssuesSummary получены из разных API SonarQube:
// - Metrics — агрегированные метрики проекта (api/measures/component), включая исторические данные
// - IssuesSummary — только OPEN issues на момент запроса (api/issues/search?statuses=OPEN)
//
// Поэтому Metrics.Bugs может не совпадать с IssuesSummary.ByType["BUG"].
// Например, если баг был закрыт недавно, он ещё может быть в агрегированных метриках,
// но не будет в OPEN issues.
type BranchReportData struct {
	// Branch — имя ветки
	Branch string `json:"branch"`
	// ProjectKey — ключ проекта в SonarQube
	ProjectKey string `json:"project_key"`
	// QualityGateStatus — статус Quality Gate (OK, ERROR, WARN)
	QualityGateStatus string `json:"quality_gate_status"`
	// Metrics — агрегированные метрики качества из api/measures/component
	Metrics *QualityMetrics `json:"metrics"`
	// IssuesSummary — breakdown OPEN issues по типам и severity из api/issues/search
	IssuesSummary *IssuesSummary `json:"issues_summary"`
	// Comparison — сравнение с base-веткой
	Comparison *BranchComparison `json:"comparison,omitempty"`
}

// QualityMetrics содержит качественные метрики проекта.
type QualityMetrics struct {
	// Bugs — количество багов
	Bugs int `json:"bugs"`
	// Vulnerabilities — количество уязвимостей
	Vulnerabilities int `json:"vulnerabilities"`
	// CodeSmells — количество code smells
	CodeSmells int `json:"code_smells"`
	// Coverage — покрытие кода тестами (в процентах)
	Coverage float64 `json:"coverage"`
	// DuplicatedLinesDensity — процент дублирования кода
	DuplicatedLinesDensity float64 `json:"duplicated_lines_density"`
	// Ncloc — количество строк кода (non-comment lines of code)
	Ncloc int `json:"ncloc"`
}

// IssuesSummary содержит breakdown проблем по типам и severity.
type IssuesSummary struct {
	// Total — общее количество проблем
	Total int `json:"total"`
	// ByType — breakdown по типам (BUG, VULNERABILITY, CODE_SMELL)
	ByType map[string]int `json:"by_type"`
	// BySeverity — breakdown по severity (BLOCKER, CRITICAL, MAJOR, MINOR, INFO)
	BySeverity map[string]int `json:"by_severity"`
}

// BranchComparison содержит сравнение с base-веткой.
type BranchComparison struct {
	// BaseBranch — имя base-ветки (обычно "main")
	BaseBranch string `json:"base_branch"`
	// BaseProjectKey — ключ base-проекта
	BaseProjectKey string `json:"base_project_key"`
	// NewBugs — новые баги относительно base
	NewBugs int `json:"new_bugs"`
	// NewVulnerabilities — новые уязвимости относительно base
	NewVulnerabilities int `json:"new_vulnerabilities"`
	// NewCodeSmells — новые code smells относительно base
	NewCodeSmells int `json:"new_code_smells"`
	// CoverageDelta — изменение покрытия (в процентных пунктах)
	CoverageDelta float64 `json:"coverage_delta"`
	// BaseNotFound — true если base-проект не найден в SonarQube
	BaseNotFound bool `json:"base_not_found,omitempty"`
}

// writeText выводит отчёт в человекочитаемом формате с цветовой индикацией.
func (d *BranchReportData) writeText(w io.Writer) error {
	// Заголовок
	if _, err := fmt.Fprintf(w, "══════════════════════════════════════════════════════\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "📊 Отчёт о качестве ветки: %s\n", d.Branch); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "══════════════════════════════════════════════════════\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Проект: %s\n", d.ProjectKey); err != nil {
		return err
	}

	// Quality Gate с индикацией
	qgIcon := qualityGateIcon(d.QualityGateStatus)
	if _, err := fmt.Fprintf(w, "Quality Gate: %s %s\n\n", qgIcon, d.QualityGateStatus); err != nil {
		return err
	}

	// Метрики
	if d.Metrics != nil {
		if _, err := fmt.Fprintln(w, "📈 Метрики:"); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  Баги:          %d\n", d.Metrics.Bugs); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  Уязвимости:    %d\n", d.Metrics.Vulnerabilities); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  Code Smells:   %d\n", d.Metrics.CodeSmells); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  Покрытие:      %.1f%%\n", d.Metrics.Coverage); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  Дублирование:  %.1f%%\n", d.Metrics.DuplicatedLinesDensity); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  Строк кода:    %d\n\n", d.Metrics.Ncloc); err != nil {
			return err
		}
	}

	// Issues summary
	if d.IssuesSummary != nil {
		if _, err := fmt.Fprintf(w, "📋 Проблемы (всего: %d):\n", d.IssuesSummary.Total); err != nil {
			return err
		}
		// M-3 fix: defensive nil checks для maps
		byType := d.IssuesSummary.ByType
		if byType == nil {
			byType = make(map[string]int)
		}
		bySeverity := d.IssuesSummary.BySeverity
		if bySeverity == nil {
			bySeverity = make(map[string]int)
		}
		if _, err := fmt.Fprintf(w, "  По типу:       BUG=%d, VULNERABILITY=%d, CODE_SMELL=%d\n",
			byType["BUG"],
			byType["VULNERABILITY"],
			byType["CODE_SMELL"]); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  По важности:   BLOCKER=%d, CRITICAL=%d, MAJOR=%d, MINOR=%d, INFO=%d\n\n",
			bySeverity["BLOCKER"],
			bySeverity["CRITICAL"],
			bySeverity["MAJOR"],
			bySeverity["MINOR"],
			bySeverity["INFO"]); err != nil {
			return err
		}
	}

	// Сравнение с base-веткой
	if d.Comparison != nil {
		if _, err := fmt.Fprintf(w, "📊 Сравнение с %s:\n", d.Comparison.BaseBranch); err != nil {
			return err
		}
		if d.Comparison.BaseNotFound {
			if _, err := fmt.Fprintln(w, "  ⚠️  Base-проект не найден в SonarQube"); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(w, "  Новые баги:         %s\n", formatDelta(d.Comparison.NewBugs)); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(w, "  Новые уязвимости:   %s\n", formatDelta(d.Comparison.NewVulnerabilities)); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(w, "  Новые code smells:  %s\n", formatDelta(d.Comparison.NewCodeSmells)); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(w, "  Изменение покрытия: %s\n", formatCoverageDelta(d.Comparison.CoverageDelta)); err != nil {
				return err
			}
		}
	}

	if _, err := fmt.Fprintf(w, "══════════════════════════════════════════════════════\n"); err != nil {
		return err
	}

	return nil
}

// qualityGateIcon возвращает иконку для статуса Quality Gate.
func qualityGateIcon(status string) string {
	switch status {
	case "OK":
		return "✅"
	case "ERROR":
		return "❌"
	case "WARN":
		return "⚠️"
	default:
		return "❓"
	}
}

// formatDelta форматирует дельту для текстового вывода.
func formatDelta(delta int) string {
	if delta > 0 {
		return fmt.Sprintf("+%d", delta)
	}
	return fmt.Sprintf("%d", delta)
}

// formatCoverageDelta форматирует изменение покрытия для текстового вывода.
func formatCoverageDelta(delta float64) string {
	if delta > 0 {
		return fmt.Sprintf("+%.1f%%", delta)
	}
	return fmt.Sprintf("%.1f%%", delta)
}

// L-2 fix: isValidBranchForScanning перенесена в shared.IsValidBranchForScanning

// ReportBranchHandler обрабатывает команду nr-sq-report-branch.
type ReportBranchHandler struct {
	// sonarqubeClient — клиент для работы с SonarQube API.
	// Может быть nil в production (создаётся через фабрику).
	// В тестах инъектируется напрямую.
	sonarqubeClient sonarqube.Client
}

// Name возвращает имя команды.
func (h *ReportBranchHandler) Name() string {
	return constants.ActNRSQReportBranch
}

// Description возвращает описание команды для вывода в help.
func (h *ReportBranchHandler) Description() string {
	return "Получить отчёт о качестве ветки из SonarQube"
}

// Execute выполняет команду nr-sq-report-branch.
func (h *ReportBranchHandler) Execute(ctx context.Context, cfg *config.Config) error {
	start := time.Now()

	traceID := tracing.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = tracing.GenerateTraceID()
	}

	format := os.Getenv("BR_OUTPUT_FORMAT")

	// Story 7.3 AC-8: plan-only для команд без поддержки плана
	// Review #36: !IsDryRun() — dry-run имеет приоритет над plan-only (AC-11).
	if !dryrun.IsDryRun() && dryrun.IsPlanOnly() {
		return dryrun.WritePlanOnlyUnsupported(os.Stdout, constants.ActNRSQReportBranch)
	}

	log := slog.Default().With(slog.String("trace_id", traceID), slog.String("command", constants.ActNRSQReportBranch))

	// 1. Валидация конфигурации
	if cfg == nil {
		log.Error("Конфигурация не загружена")
		return h.writeError(format, traceID, start,
			errConfigMissing,
			"Конфигурация не загружена")
	}

	// 2. Получение и валидация ветки
	branch := cfg.BranchForScan
	if branch == "" {
		log.Error("Не указана ветка для отчёта")
		return h.writeError(format, traceID, start,
			errBranchMissing,
			"Не указана ветка для отчёта (BR_BRANCH)")
	}

	log = log.With(slog.String("branch", branch))

	// 3. Валидация owner/repo
	owner := cfg.Owner
	repo := cfg.Repo
	if owner == "" || repo == "" {
		log.Error("Не указаны owner или repo")
		return h.writeError(format, traceID, start,
			errMissingOwnerRepo,
			"Не указаны владелец (BR_OWNER) или репозиторий (BR_REPO)")
	}

	// 4. Формирование ключей проектов
	projectKey := fmt.Sprintf("%s_%s_%s", owner, repo, branch)
	baseProjectKey := fmt.Sprintf("%s_%s_%s", owner, repo, constants.BaseBranch)
	log = log.With(slog.String("project_key", projectKey))

	// M-2 fix: Предупреждение если ветка не соответствует паттерну сканирования
	// L-2 fix: используем shared.IsValidBranchForScanning вместо локальной функции
	if !shared.IsValidBranchForScanning(branch) {
		log.Warn("Ветка не соответствует паттерну сканирования (main или t######) — отчёт может быть неполным")
	}

	log.Info("Запуск генерации отчёта о качестве ветки")

	// 5. Получение SonarQube клиента
	// TODO(H-6): Реализовать фабрику createSonarQubeClient(cfg) для создания реального клиента.
	// Текущая реализация требует DI через поле sonarqubeClient (используется в тестах).
	// Для production необходимо создать реализацию sonarqube.Client на основе internal/entity/sonarqube
	// или написать новую реализацию в internal/adapter/sonarqube/client.go.
	sqClient := h.sonarqubeClient
	if sqClient == nil {
		log.Error("SonarQube клиент не настроен")
		return h.writeError(format, traceID, start,
			errConfigMissing,
			"SonarQube клиент не настроен — требуется реализация фабрики createSonarQubeClient()")
	}

	// 6. Проверка существования проекта
	_, err := sqClient.GetProject(ctx, projectKey)
	if err != nil {
		log.Error("Проект не найден в SonarQube", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start,
			errProjectNotFound,
			fmt.Sprintf("Проект '%s' не найден в SonarQube", projectKey))
	}

	// 7. Получение метрик проекта
	metricKeys := []string{
		"bugs",
		"vulnerabilities",
		"code_smells",
		"coverage",
		"duplicated_lines_density",
		"ncloc",
	}
	metrics, err := sqClient.GetMetrics(ctx, projectKey, metricKeys)
	if err != nil {
		log.Error("Не удалось получить метрики", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start,
			errSonarQubeAPI,
			fmt.Sprintf("Не удалось получить метрики: %v", err))
	}

	// 8. Получение статуса Quality Gate
	qgStatus, err := sqClient.GetQualityGateStatus(ctx, projectKey)
	if err != nil {
		log.Error("Не удалось получить статус Quality Gate", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start,
			errSonarQubeAPI,
			fmt.Sprintf("Не удалось получить статус Quality Gate: %v", err))
	}

	// 9. Получение issues для breakdown
	issues, err := sqClient.GetIssues(ctx, sonarqube.GetIssuesOptions{
		ProjectKey: projectKey,
		Statuses:   []string{"OPEN"},
	})
	if err != nil {
		log.Error("Не удалось получить список проблем", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start,
			errSonarQubeAPI,
			fmt.Sprintf("Не удалось получить список проблем: %v", err))
	}
	issuesSummary := buildIssuesSummary(issues)

	// 10. Сравнение с base-веткой (опционально)
	comparison := h.buildComparison(ctx, sqClient, metrics, baseProjectKey, metricKeys, log)

	// 11. Формирование ответа
	data := &BranchReportData{
		Branch:            branch,
		ProjectKey:        projectKey,
		QualityGateStatus: qgStatus.Status,
		Metrics:           buildQualityMetrics(metrics),
		IssuesSummary:     issuesSummary,
		Comparison:        comparison,
	}

	log.Info("Отчёт о качестве сформирован",
		slog.String("quality_gate", qgStatus.Status),
		slog.Int("total_issues", issuesSummary.Total))

	// 12. Вывод результата
	return h.writeSuccess(format, traceID, start, data)
}

// buildIssuesSummary строит сводку по проблемам.
func buildIssuesSummary(issues []sonarqube.Issue) *IssuesSummary {
	summary := &IssuesSummary{
		Total:      len(issues),
		ByType:     make(map[string]int),
		BySeverity: make(map[string]int),
	}

	// Инициализация счётчиков
	for _, t := range []string{"BUG", "VULNERABILITY", "CODE_SMELL"} {
		summary.ByType[t] = 0
	}
	for _, s := range []string{"BLOCKER", "CRITICAL", "MAJOR", "MINOR", "INFO"} {
		summary.BySeverity[s] = 0
	}

	// Подсчёт
	for _, issue := range issues {
		summary.ByType[issue.Type]++
		summary.BySeverity[issue.Severity]++
	}

	return summary
}

// buildQualityMetrics преобразует метрики SonarQube в QualityMetrics.
func buildQualityMetrics(metrics *sonarqube.Metrics) *QualityMetrics {
	if metrics == nil || metrics.Measures == nil {
		return &QualityMetrics{}
	}

	return &QualityMetrics{
		Bugs:                   parseIntMetric(metrics.Measures, "bugs"),
		Vulnerabilities:        parseIntMetric(metrics.Measures, "vulnerabilities"),
		CodeSmells:             parseIntMetric(metrics.Measures, "code_smells"),
		Coverage:               parseFloatMetric(metrics.Measures, "coverage"),
		DuplicatedLinesDensity: parseFloatMetric(metrics.Measures, "duplicated_lines_density"),
		Ncloc:                  parseIntMetric(metrics.Measures, "ncloc"),
	}
}

// parseIntMetric парсит целочисленную метрику из map.
func parseIntMetric(measures map[string]string, key string) int {
	if val, ok := measures[key]; ok {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return 0
}

// parseFloatMetric парсит метрику с плавающей точкой из map.
func parseFloatMetric(measures map[string]string, key string) float64 {
	if val, ok := measures[key]; ok {
		if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
			return floatVal
		}
	}
	return 0.0
}

// buildComparison строит сравнение с base-веткой.
func (h *ReportBranchHandler) buildComparison(
	ctx context.Context,
	sqClient sonarqube.Client,
	currentMetrics *sonarqube.Metrics,
	baseProjectKey string,
	metricKeys []string,
	log *slog.Logger,
) *BranchComparison {
	comparison := &BranchComparison{
		BaseBranch:     constants.BaseBranch,
		BaseProjectKey: baseProjectKey,
	}

	// Проверяем существование base-проекта
	_, err := sqClient.GetProject(ctx, baseProjectKey)
	if err != nil {
		log.Info("Base-проект не найден", slog.String("base_project_key", baseProjectKey))
		comparison.BaseNotFound = true
		return comparison
	}

	// Получаем метрики base-проекта
	baseMetrics, err := sqClient.GetMetrics(ctx, baseProjectKey, metricKeys)
	if err != nil {
		log.Warn("Не удалось получить метрики base-проекта", slog.String("error", err.Error()))
		comparison.BaseNotFound = true
		return comparison
	}

	// Вычисляем дельту
	currentQM := buildQualityMetrics(currentMetrics)
	baseQM := buildQualityMetrics(baseMetrics)

	comparison.NewBugs = currentQM.Bugs - baseQM.Bugs
	comparison.NewVulnerabilities = currentQM.Vulnerabilities - baseQM.Vulnerabilities
	comparison.NewCodeSmells = currentQM.CodeSmells - baseQM.CodeSmells
	comparison.CoverageDelta = currentQM.Coverage - baseQM.Coverage

	return comparison
}

// writeSuccess выводит успешный результат.
func (h *ReportBranchHandler) writeSuccess(format, traceID string, start time.Time, data *BranchReportData) error {
	// Текстовый формат
	if format != output.FormatJSON {
		return data.writeText(os.Stdout)
	}

	// JSON формат
	result := &output.Result{
		Status:  output.StatusSuccess,
		Command: constants.ActNRSQReportBranch,
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
func (h *ReportBranchHandler) writeError(format, traceID string, start time.Time, code, message string) error {
	// Текстовый формат — человекочитаемый вывод ошибки
	if format != output.FormatJSON {
		_, _ = fmt.Fprintf(os.Stdout, "Ошибка: %s\nКод: %s\n", message, code)
		return fmt.Errorf("%s: %s", code, message)
	}

	// JSON формат — структурированный вывод
	result := &output.Result{
		Status:  output.StatusError,
		Command: constants.ActNRSQReportBranch,
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
