// Package scanpr реализует NR-команду nr-sq-scan-pr
// для сканирования pull request на качество кода через SonarQube.
package scanpr

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/gitea"
	"github.com/Kargones/apk-ci/internal/adapter/sonarqube"
	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/command/handlers/sonarqube/shared"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/dryrun"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"
)

// Коды ошибок — используем shared константы для соблюдения DRY.
// Локальные алиасы для краткости.
const (
	errPRMissing     = shared.ErrPRMissing
	errPRInvalid     = shared.ErrPRInvalid
	errPRNotFound    = shared.ErrPRNotFound
	errPRNotOpen     = shared.ErrPRNotOpen
	errSonarQubeAPI  = shared.ErrSonarQubeAPI
	errGiteaAPI      = shared.ErrGiteaAPI
	errConfigMissing = shared.ErrConfigMissing
)

// init регистрирует команду nr-sq-scan-pr с deprecated alias sq-scan-pr.
// Зависит от инициализации пакетов command и constants.
func init() {
	// ActSQScanPR — deprecated alias для обратной совместимости
	command.RegisterWithAlias(&ScanPRHandler{}, constants.ActSQScanPR)
}

// ScanPRData содержит результат сканирования pull request.
type ScanPRData struct {
	// PRNumber — номер pull request
	PRNumber int64 `json:"pr_number"`
	// PRTitle — заголовок pull request
	PRTitle string `json:"pr_title"`
	// HeadBranch — исходная ветка PR
	HeadBranch string `json:"head_branch"`
	// BaseBranch — целевая ветка PR
	BaseBranch string `json:"base_branch"`
	// ProjectKey — ключ проекта в SonarQube
	ProjectKey string `json:"project_key"`
	// CommitSHA — SHA последнего коммита в PR
	CommitSHA string `json:"commit_sha"`
	// CommitsScanned — количество отсканированных коммитов (1 для PR)
	CommitsScanned int `json:"commits_scanned"`
	// Scanned — было ли выполнено сканирование
	Scanned bool `json:"scanned"`
	// AlreadyScanned — true если коммит уже был отсканирован
	AlreadyScanned bool `json:"already_scanned,omitempty"`
	// NoRelevantChanges — true если нет изменений в конфигурации
	NoRelevantChanges bool `json:"no_relevant_changes,omitempty"`
	// ScanResult — результат сканирования
	ScanResult *ScanResult `json:"scan_result,omitempty"`
}

// ScanResult содержит результат анализа качества.
// TODO: Добавить получение метрик NewIssues, NewBugs, NewVulnerabilities, NewCodeSmells
// через sonarqube.GetMeasures() API. Текущая реализация возвращает только QualityGateStatus.
type ScanResult struct {
	// AnalysisID — ID анализа в SonarQube
	AnalysisID string `json:"analysis_id"`
	// Status — статус анализа (SUCCESS, FAILED)
	Status string `json:"status"`
	// QualityGateStatus — статус quality gate (OK, ERROR, WARN)
	QualityGateStatus string `json:"quality_gate_status,omitempty"`
	// ErrorMessage — сообщение об ошибке (если есть)
	ErrorMessage string `json:"error_message,omitempty"`
}

// writeText выводит результаты сканирования PR в человекочитаемом формате.
func (d *ScanPRData) writeText(w io.Writer) error {
	_, err := fmt.Fprintf(w, "Сканирование PR #%d: %s\n", d.PRNumber, d.PRTitle)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "Ветка: %s → %s\n", d.HeadBranch, d.BaseBranch)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "Проект SonarQube: %s\n", d.ProjectKey)
	if err != nil {
		return err
	}

	shortSHA := d.CommitSHA
	if shortSHA == "" {
		shortSHA = "unknown"
	} else if len(shortSHA) > 7 {
		shortSHA = shortSHA[:7]
	}
	_, err = fmt.Fprintf(w, "Коммит: %s\n", shortSHA)
	if err != nil {
		return err
	}

	if d.AlreadyScanned {
		_, err = fmt.Fprintln(w, "Статус: коммит уже отсканирован")
		return err
	}

	if d.NoRelevantChanges {
		_, err = fmt.Fprintln(w, "Статус: нет изменений в каталогах конфигурации")
		return err
	}

	if !d.Scanned {
		_, err = fmt.Fprintln(w, "Статус: сканирование не выполнено")
		return err
	}

	if d.ScanResult != nil {
		statusIcon := "✓"
		if d.ScanResult.Status != "SUCCESS" {
			statusIcon = "✗"
		}
		_, err = fmt.Fprintf(w, "Результат: %s %s\n", statusIcon, d.ScanResult.Status)
		if err != nil {
			return err
		}

		if d.ScanResult.QualityGateStatus != "" {
			qgIcon := "✓"
			if d.ScanResult.QualityGateStatus != "OK" {
				qgIcon = "✗"
			}
			_, err = fmt.Fprintf(w, "Quality Gate: %s %s\n", qgIcon, d.ScanResult.QualityGateStatus)
			if err != nil {
				return err
			}
		}

		if d.ScanResult.ErrorMessage != "" {
			_, err = fmt.Fprintf(w, "Ошибка: %s\n", d.ScanResult.ErrorMessage)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// ScanPRHandler обрабатывает команду nr-sq-scan-pr.
type ScanPRHandler struct {
	// sonarqubeClient — опциональный SonarQube клиент (nil в production, mock в тестах)
	sonarqubeClient sonarqube.Client
	// giteaClient — опциональный Gitea клиент (nil в production, mock в тестах)
	giteaClient gitea.Client
}

// Name возвращает имя команды.
func (h *ScanPRHandler) Name() string {
	return constants.ActNRSQScanPR
}

// Description возвращает описание команды для вывода в help.
func (h *ScanPRHandler) Description() string {
	return "Сканирование pull request на качество кода через SonarQube"
}

// Execute выполняет команду nr-sq-scan-pr.
func (h *ScanPRHandler) Execute(ctx context.Context, cfg *config.Config) error {
	start := time.Now()

	traceID := tracing.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = tracing.GenerateTraceID()
	}

	format := os.Getenv("BR_OUTPUT_FORMAT")

	// Story 7.3 AC-8: plan-only для команд без поддержки плана
	// Review #36: !IsDryRun() — dry-run имеет приоритет над plan-only (AC-11).
	if !dryrun.IsDryRun() && dryrun.IsPlanOnly() {
		return dryrun.WritePlanOnlyUnsupported(os.Stdout, constants.ActNRSQScanPR)
	}

	log := slog.Default().With(slog.String("trace_id", traceID), slog.String("command", constants.ActNRSQScanPR))

	// Валидация конфигурации (AC: #2)
	if cfg == nil {
		log.Error("Конфигурация не загружена")
		return h.writeError(format, traceID, start,
			errConfigMissing,
			"Конфигурация не загружена")
	}

	// Получение и валидация номера PR (AC: #2)
	prNumber := cfg.PRNumber
	if prNumber == 0 {
		log.Error("Не указан номер PR")
		return h.writeError(format, traceID, start,
			errPRMissing,
			"Не указан номер PR (BR_PR_NUMBER)")
	}
	if prNumber < 0 {
		log.Error("Некорректный номер PR", slog.Int64("pr_number", prNumber))
		return h.writeError(format, traceID, start,
			errPRInvalid,
			fmt.Sprintf("Некорректный номер PR: %d (должен быть положительным)", prNumber))
	}

	log = log.With(slog.Int64("pr_number", prNumber))
	log.Info("Запуск сканирования PR")

	// Получение owner и repo из конфигурации
	owner := cfg.Owner
	repo := cfg.Repo
	if owner == "" || repo == "" {
		log.Error("Не указаны owner или repo")
		return h.writeError(format, traceID, start,
			errConfigMissing,
			"Не указаны владелец (BR_OWNER) или репозиторий (BR_REPO)")
	}

	// Получение Gitea клиента
	// TODO: Реализовать фабрику createGiteaClient(cfg) для создания реального клиента.
	// Текущая реализация требует DI через поле giteaClient (используется в тестах).
	giteaClient := h.giteaClient
	if giteaClient == nil {
		log.Error("Gitea клиент не настроен")
		return h.writeError(format, traceID, start,
			errConfigMissing,
			"Gitea клиент не настроен — требуется реализация фабрики createGiteaClient()")
	}

	// Получение SonarQube клиента
	// TODO: Реализовать фабрику createSonarQubeClient(cfg) для создания реального клиента.
	sqClient := h.sonarqubeClient
	if sqClient == nil {
		log.Error("SonarQube клиент не настроен")
		return h.writeError(format, traceID, start,
			errConfigMissing,
			"SonarQube клиент не настроен — требуется реализация фабрики createSonarQubeClient()")
	}

	// Получение информации о PR из Gitea (AC: #3)
	pr, err := giteaClient.GetPR(ctx, prNumber)
	if err != nil {
		log.Error("Не удалось получить информацию о PR", slog.String("error", err.Error()))
		// Определяем тип ошибки: not found или API error
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "404") {
			return h.writeError(format, traceID, start,
				errPRNotFound,
				fmt.Sprintf("PR #%d не найден в репозитории", prNumber))
		}
		return h.writeError(format, traceID, start,
			errGiteaAPI,
			fmt.Sprintf("Не удалось получить информацию о PR: %v", err))
	}

	// Проверка состояния PR (AC: #3)
	if pr.State != "open" {
		log.Error("PR не в состоянии open", slog.String("state", pr.State))
		return h.writeError(format, traceID, start,
			errPRNotOpen,
			fmt.Sprintf("PR #%d имеет статус '%s', требуется 'open'", prNumber, pr.State))
	}

	// Извлечение параметров из PR (AC: #4)
	headBranch := pr.Head.Name
	baseBranch := pr.Base.Name
	commitSHA := pr.Head.Commit.ID
	prTitle := pr.Title

	log = log.With(
		slog.String("head_branch", headBranch),
		slog.String("base_branch", baseBranch),
		slog.String("commit_sha", commitSHA),
	)

	// Формирование ключа проекта SonarQube (AC: #4)
	projectKey := fmt.Sprintf("%s_%s_%s", owner, repo, headBranch)
	log = log.With(slog.String("project_key", projectKey))

	// Базовая структура результата
	data := &ScanPRData{
		PRNumber:   prNumber,
		PRTitle:    prTitle,
		HeadBranch: headBranch,
		BaseBranch: baseBranch,
		ProjectKey: projectKey,
		CommitSHA:  commitSHA,
	}

	// Проверка изменений в конфигурации (AC: #5)
	hasChanges, err := shared.HasRelevantChangesInCommit(ctx, giteaClient, headBranch, commitSHA)
	if err != nil {
		log.Warn("Ошибка проверки изменений в коммите", slog.String("error", err.Error()))
		// Продолжаем с этим коммитом, т.к. ошибка API не означает отсутствие изменений
		hasChanges = true
	}

	if !hasChanges {
		log.Info("Нет изменений в каталогах конфигурации")
		data.NoRelevantChanges = true
		return h.writeSuccess(format, traceID, start, data)
	}

	// Получение уже отсканированных анализов (AC: #5)
	analyses, err := sqClient.GetAnalyses(ctx, projectKey)
	if err != nil {
		log.Warn("Не удалось получить список анализов", slog.String("error", err.Error()))
		// Продолжаем, считаем что ни один коммит не был отсканирован
		analyses = nil
	}

	// Проверка, был ли коммит уже отсканирован
	for _, a := range analyses {
		if a.Revision == commitSHA {
			log.Info("Коммит уже отсканирован")
			data.AlreadyScanned = true
			return h.writeSuccess(format, traceID, start, data)
		}
	}

	// Получение или создание проекта в SonarQube (AC: #5)
	_, err = sqClient.GetProject(ctx, projectKey)
	if err != nil {
		log.Info("Проект не найден в SonarQube, создаём",
			slog.String("get_error", err.Error()))
		projectName := fmt.Sprintf("%s/%s (%s)", owner, repo, headBranch)
		_, err = sqClient.CreateProject(ctx, sonarqube.CreateProjectOptions{
			Key:        projectKey,
			Name:       projectName,
			Visibility: "private",
		})
		if err != nil {
			log.Error("Не удалось создать проект в SonarQube", slog.String("error", err.Error()))
			return h.writeError(format, traceID, start,
				errSonarQubeAPI,
				fmt.Sprintf("Не удалось создать проект в SonarQube: %v", err))
		}
	}

	// Запуск сканирования (AC: #5)
	log.Info("Сканирование коммита", slog.String("commit", commitSHA))

	shortSHA := commitSHA
	if len(commitSHA) > 7 {
		shortSHA = commitSHA[:7]
	}
	// TODO: SourcePath не заполняется — требуется добавить cfg.WorkDir или cfg.SourcePath
	result, err := sqClient.RunAnalysis(ctx, sonarqube.RunAnalysisOptions{
		ProjectKey: projectKey,
		Branch:     headBranch,
		Properties: map[string]string{
			"sonar.projectVersion": shortSHA,
			"sonar.scm.revision":   commitSHA,
		},
	})
	if err != nil {
		log.Error("Не удалось запустить анализ", slog.String("error", err.Error()))
		data.Scanned = true
		data.CommitsScanned = 1
		data.ScanResult = &ScanResult{
			Status:       "FAILED",
			ErrorMessage: err.Error(),
		}
		return h.writeSuccess(format, traceID, start, data)
	}

	// Ожидание завершения анализа (AC: #5)
	status, err := shared.WaitForAnalysisCompletion(ctx, sqClient, result.TaskID, log)
	if err != nil {
		log.Error("Ошибка ожидания завершения анализа", slog.String("error", err.Error()))
		data.Scanned = true
		data.CommitsScanned = 1
		data.ScanResult = &ScanResult{
			AnalysisID:   result.AnalysisID,
			Status:       "FAILED",
			ErrorMessage: err.Error(),
		}
		return h.writeSuccess(format, traceID, start, data)
	}

	// Формирование результата сканирования
	data.Scanned = true
	data.CommitsScanned = 1
	data.ScanResult = &ScanResult{
		AnalysisID:   status.AnalysisID,
		Status:       status.Status,
		ErrorMessage: status.ErrorMessage,
	}

	// Получение quality gate status если анализ успешен (AC: #6)
	if status.Status == "SUCCESS" {
		qgStatus, qgErr := sqClient.GetQualityGateStatus(ctx, projectKey)
		if qgErr != nil {
			log.Warn("Не удалось получить статус Quality Gate", slog.String("error", qgErr.Error()))
		} else if qgStatus != nil {
			data.ScanResult.QualityGateStatus = qgStatus.Status
		}
	}

	log.Info("Сканирование PR завершено",
		slog.String("status", status.Status),
		slog.String("analysis_id", status.AnalysisID))

	return h.writeSuccess(format, traceID, start, data)
}

// writeSuccess выводит успешный результат.
func (h *ScanPRHandler) writeSuccess(format, traceID string, start time.Time, data *ScanPRData) error {
	// Текстовый формат
	if format != output.FormatJSON {
		return data.writeText(os.Stdout)
	}

	// JSON формат
	result := &output.Result{
		Status:  output.StatusSuccess,
		Command: constants.ActNRSQScanPR,
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
func (h *ScanPRHandler) writeError(format, traceID string, start time.Time, code, message string) error {
	// Текстовый формат — человекочитаемый вывод ошибки
	if format != output.FormatJSON {
		_, _ = fmt.Fprintf(os.Stdout, "Ошибка: %s\nКод: %s\n", message, code)
		return fmt.Errorf("%s: %s", code, message)
	}

	// JSON формат — структурированный вывод
	result := &output.Result{
		Status:  output.StatusError,
		Command: constants.ActNRSQScanPR,
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
