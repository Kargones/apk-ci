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
	errhandler "github.com/Kargones/apk-ci/internal/command/handlers/shared"
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
func RegisterCmd() error {
	// ActSQScanPR — deprecated alias для обратной совместимости
	// Deprecated: alias "sq-scan-pr" retained for backward compatibility. Remove in v2.0.0 (Epic 7).
	return command.RegisterWithAlias(&ScanPRHandler{}, constants.ActSQScanPR)
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

// prExecContext holds shared state for a single PR scan invocation.
type prExecContext struct {
	start   time.Time
	traceID string
	format  string
	log     *slog.Logger
}

// validatePRConfig validates configuration and PR parameters.
func (h *ScanPRHandler) validatePRConfig(cfg *config.Config) (*prExecContext, error) {
	ec := &prExecContext{start: time.Now()}
	ec.traceID = tracing.TraceIDFromContext(context.Background())
	if ec.traceID == "" {
		ec.traceID = tracing.GenerateTraceID()
	}
	ec.format = os.Getenv("BR_OUTPUT_FORMAT")

	if !dryrun.IsDryRun() && dryrun.IsPlanOnly() {
		return nil, dryrun.WritePlanOnlyUnsupported(os.Stdout, constants.ActNRSQScanPR)
	}

	ec.log = slog.Default().With(slog.String("trace_id", ec.traceID), slog.String("command", constants.ActNRSQScanPR))

	if cfg == nil {
		ec.log.Error("Конфигурация не загружена")
		return nil, h.writeError(ec.format, ec.traceID, ec.start, errConfigMissing, "Конфигурация не загружена")
	}

	if cfg.PRNumber == 0 {
		ec.log.Error("Не указан номер PR")
		return nil, h.writeError(ec.format, ec.traceID, ec.start, errPRMissing, "Не указан номер PR (BR_PR_NUMBER)")
	}
	if cfg.PRNumber < 0 {
		ec.log.Error("Некорректный номер PR", slog.Int64("pr_number", cfg.PRNumber))
		return nil, h.writeError(ec.format, ec.traceID, ec.start, errPRInvalid,
			fmt.Sprintf("Некорректный номер PR: %d (должен быть положительным)", cfg.PRNumber))
	}

	ec.log = ec.log.With(slog.Int64("pr_number", cfg.PRNumber))

	if cfg.Owner == "" || cfg.Repo == "" {
		ec.log.Error("Не указаны owner или repo")
		return nil, h.writeError(ec.format, ec.traceID, ec.start, errConfigMissing, "Не указаны владелец (BR_OWNER) или репозиторий (BR_REPO)")
	}

	return ec, nil
}

// getPRClients returns Gitea and SonarQube clients.
func (h *ScanPRHandler) getPRClients(ec *prExecContext, cfg *config.Config) (gitea.Client, sonarqube.Client, error) {
	gc := h.giteaClient
	if gc == nil {
		var err error
		gc, err = errhandler.CreateGiteaClient(cfg)
		if err != nil {
			ec.log.Error("Не удалось создать Gitea клиент", slog.String("error", err.Error()))
			return nil, nil, h.writeError(ec.format, ec.traceID, ec.start, errConfigMissing, "Не удалось создать Gitea клиент: "+err.Error())
		}
	}
	sc := h.sonarqubeClient
	if sc == nil {
		var err error
		sc, err = errhandler.CreateSonarQubeClient(cfg)
		if err != nil {
			ec.log.Error("Не удалось создать SonarQube клиент", slog.String("error", err.Error()))
			return nil, nil, h.writeError(ec.format, ec.traceID, ec.start, errConfigMissing, "Не удалось создать SonarQube клиент: "+err.Error())
		}
	}
	return gc, sc, nil
}

// ensurePRProject creates the SonarQube project if it does not exist.
func (h *ScanPRHandler) ensurePRProject(ctx context.Context, ec *prExecContext, sqClient sonarqube.Client, projectKey, owner, repo, branch string) error {
	_, err := sqClient.GetProject(ctx, projectKey)
	if err != nil {
		ec.log.Info("Проект не найден в SonarQube, создаём", slog.String("get_error", err.Error()))
		projectName := fmt.Sprintf("%s/%s (%s)", owner, repo, branch)
		_, err = sqClient.CreateProject(ctx, sonarqube.CreateProjectOptions{Key: projectKey, Name: projectName, Visibility: "private"})
		if err != nil {
			ec.log.Error("Не удалось создать проект в SonarQube", slog.String("error", err.Error()))
			return h.writeError(ec.format, ec.traceID, ec.start, errSonarQubeAPI,
				fmt.Sprintf("Не удалось создать проект в SonarQube: %v", err))
		}
	}
	return nil
}

// runPRScan runs the analysis and waits for completion, populating data with results.
func (h *ScanPRHandler) runPRScan(ctx context.Context, ec *prExecContext, sqClient sonarqube.Client, data *ScanPRData) {
	ec.log.Info("Сканирование коммита", slog.String("commit", data.CommitSHA))
	shortSHA := data.CommitSHA
	if len(data.CommitSHA) > 7 {
		shortSHA = data.CommitSHA[:7]
	}
	result, err := sqClient.RunAnalysis(ctx, sonarqube.RunAnalysisOptions{
		ProjectKey: data.ProjectKey, Branch: data.HeadBranch,
		Properties: map[string]string{"sonar.projectVersion": shortSHA, "sonar.scm.revision": data.CommitSHA},
	})
	if err != nil {
		ec.log.Error("Не удалось запустить анализ", slog.String("error", err.Error()))
		data.Scanned, data.CommitsScanned = true, 1
		data.ScanResult = &ScanResult{Status: "FAILED", ErrorMessage: err.Error()}
		return
	}

	status, err := shared.WaitForAnalysisCompletion(ctx, sqClient, result.TaskID, ec.log)
	if err != nil {
		ec.log.Error("Ошибка ожидания завершения анализа", slog.String("error", err.Error()))
		data.Scanned, data.CommitsScanned = true, 1
		data.ScanResult = &ScanResult{AnalysisID: result.AnalysisID, Status: "FAILED", ErrorMessage: err.Error()}
		return
	}

	data.Scanned, data.CommitsScanned = true, 1
	data.ScanResult = &ScanResult{AnalysisID: status.AnalysisID, Status: status.Status, ErrorMessage: status.ErrorMessage}

	if status.Status == "SUCCESS" {
		qgStatus, qgErr := sqClient.GetQualityGateStatus(ctx, data.ProjectKey)
		if qgErr != nil {
			ec.log.Warn("Не удалось получить статус Quality Gate", slog.String("error", qgErr.Error()))
		} else if qgStatus != nil {
			data.ScanResult.QualityGateStatus = qgStatus.Status
		}
	}
}

// Execute выполняет команду nr-sq-scan-pr.
func (h *ScanPRHandler) Execute(ctx context.Context, cfg *config.Config) error {
	ec, err := h.validatePRConfig(cfg)
	if err != nil {
		return err
	}
	ec.log.Info("Запуск сканирования PR")

	giteaClient, sqClient, err := h.getPRClients(ec, cfg)
	if err != nil {
		return err
	}

	// Получение информации о PR из Gitea (AC: #3)
	pr, err := giteaClient.GetPR(ctx, cfg.PRNumber)
	if err != nil {
		ec.log.Error("Не удалось получить информацию о PR", slog.String("error", err.Error()))
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "404") {
			return h.writeError(ec.format, ec.traceID, ec.start, errPRNotFound,
				fmt.Sprintf("PR #%d не найден в репозитории", cfg.PRNumber))
		}
		return h.writeError(ec.format, ec.traceID, ec.start, errGiteaAPI,
			fmt.Sprintf("Не удалось получить информацию о PR: %v", err))
	}

	if pr.State != "open" {
		ec.log.Error("PR не в состоянии open", slog.String("state", pr.State))
		return h.writeError(ec.format, ec.traceID, ec.start, errPRNotOpen,
			fmt.Sprintf("PR #%d имеет статус '%s', требуется 'open'", cfg.PRNumber, pr.State))
	}

	projectKey := fmt.Sprintf("%s_%s_%s", cfg.Owner, cfg.Repo, pr.Head.Name)
	ec.log = ec.log.With(slog.String("project_key", projectKey))

	data := &ScanPRData{
		PRNumber: cfg.PRNumber, PRTitle: pr.Title, HeadBranch: pr.Head.Name,
		BaseBranch: pr.Base.Name, ProjectKey: projectKey, CommitSHA: pr.Head.Commit.ID,
	}

	// Проверка изменений в конфигурации (AC: #5)
	hasChanges, chErr := shared.HasRelevantChangesInCommit(ctx, giteaClient, data.HeadBranch, data.CommitSHA)
	if chErr != nil {
		ec.log.Warn("Ошибка проверки изменений в коммите", slog.String("error", chErr.Error()))
		hasChanges = true
	}
	if !hasChanges {
		data.NoRelevantChanges = true
		return h.writeSuccess(ec.format, ec.traceID, ec.start, data)
	}

	// Проверка, был ли коммит уже отсканирован
	analyses, _ := sqClient.GetAnalyses(ctx, projectKey)
	for _, a := range analyses {
		if a.Revision == data.CommitSHA {
			data.AlreadyScanned = true
			return h.writeSuccess(ec.format, ec.traceID, ec.start, data)
		}
	}

	if err := h.ensurePRProject(ctx, ec, sqClient, projectKey, cfg.Owner, cfg.Repo, data.HeadBranch); err != nil {
		return err
	}

	h.runPRScan(ctx, ec, sqClient, data)

	ec.log.Info("Сканирование PR завершено",
		slog.String("status", data.ScanResult.Status),
		slog.String("analysis_id", data.ScanResult.AnalysisID))

	return h.writeSuccess(ec.format, ec.traceID, ec.start, data)
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
		return errhandler.HandleError(message, code)
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
