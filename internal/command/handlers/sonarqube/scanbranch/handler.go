// Package scanbranch реализует NR-команду nr-sq-scan-branch
// для сканирования ветки на качество кода через SonarQube.
package scanbranch

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
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
	errBranchInvalidFormat = shared.ErrBranchInvalidFormat
	errBranchMissing       = shared.ErrBranchMissing
	errSonarQubeAPI        = shared.ErrSonarQubeAPI
	errGiteaAPI            = shared.ErrGiteaAPI
	errConfigMissing       = shared.ErrConfigMissing
)

func RegisterCmd() error {
	// Deprecated: alias "sq-scan-branch" retained for backward compatibility. Remove in v2.0.0 (Epic 7).
	return command.RegisterWithAlias(&ScanBranchHandler{}, constants.ActSQScanBranch)
}

// ScanBranchData содержит результат сканирования ветки.
type ScanBranchData struct {
	// Branch — имя отсканированной ветки
	Branch string `json:"branch"`
	// ProjectKey — ключ проекта в SonarQube
	ProjectKey string `json:"project_key"`
	// CommitsScanned — количество отсканированных коммитов
	CommitsScanned int `json:"commits_scanned"`
	// SkippedCount — количество пропущенных коммитов (уже отсканированы в SonarQube)
	SkippedCount int `json:"skipped_count"`
	// NoRelevantChangesCount — количество коммитов без изменений в каталогах конфигурации
	NoRelevantChangesCount int `json:"no_relevant_changes_count,omitempty"`
	// ScanResults — результаты по каждому коммиту
	ScanResults []CommitScanResult `json:"scan_results,omitempty"`
	// NoChanges — true если в коммитах нет изменений в конфигурации
	NoChanges bool `json:"no_changes,omitempty"`
}

// CommitScanResult содержит результат сканирования одного коммита.
type CommitScanResult struct {
	// CommitSHA — хеш коммита
	CommitSHA string `json:"commit_sha"`
	// AnalysisID — ID анализа в SonarQube
	AnalysisID string `json:"analysis_id"`
	// Status — статус анализа (SUCCESS, FAILED)
	Status string `json:"status"`
	// ErrorMessage — сообщение об ошибке (если есть)
	ErrorMessage string `json:"error_message,omitempty"`
}

// writeText выводит результаты сканирования в человекочитаемом формате.
func (d *ScanBranchData) writeText(w io.Writer) error {
	_, err := fmt.Fprintf(w, "Сканирование ветки: %s\n", d.Branch)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "Проект SonarQube: %s\n", d.ProjectKey)
	if err != nil {
		return err
	}

	if d.NoChanges {
		_, err = fmt.Fprintln(w, "Статус: нет изменений в каталогах конфигурации")
		if err != nil {
			return err
		}
		if d.NoRelevantChangesCount > 0 {
			_, err = fmt.Fprintf(w, "Коммитов без изменений: %d\n", d.NoRelevantChangesCount)
		}
		return err
	}

	_, err = fmt.Fprintf(w, "Отсканировано коммитов: %d\n", d.CommitsScanned)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "Пропущено (уже сканированы): %d\n", d.SkippedCount)
	if err != nil {
		return err
	}

	if d.NoRelevantChangesCount > 0 {
		_, err = fmt.Fprintf(w, "Без изменений в конфигурации: %d\n", d.NoRelevantChangesCount)
		if err != nil {
			return err
		}
	}

	if len(d.ScanResults) > 0 {
		_, err = fmt.Fprintln(w, "Результаты:")
		if err != nil {
			return err
		}
		for i, r := range d.ScanResults {
			shortSHA := r.CommitSHA
			if shortSHA == "" {
				shortSHA = "unknown"
			} else if len(shortSHA) > 7 {
				shortSHA = shortSHA[:7]
			}
			statusIcon := "✓"
			if r.Status != "SUCCESS" {
				statusIcon = "✗"
			}
			if _, err = fmt.Fprintf(w, "  %d. %s %s — %s\n", i+1, statusIcon, shortSHA, r.Status); err != nil {
				return err
			}
			if r.ErrorMessage != "" {
				if _, err = fmt.Fprintf(w, "     Ошибка: %s\n", r.ErrorMessage); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// ScanBranchHandler обрабатывает команду nr-sq-scan-branch.
type ScanBranchHandler struct {
	// sonarqubeClient — опциональный SonarQube клиент (nil в production, mock в тестах)
	sonarqubeClient sonarqube.Client
	// giteaClient — опциональный Gitea клиент (nil в production, mock в тестах)
	giteaClient gitea.Client
}

// Name возвращает имя команды.
func (h *ScanBranchHandler) Name() string {
	return constants.ActNRSQScanBranch
}

// Description возвращает описание команды для вывода в help.
func (h *ScanBranchHandler) Description() string {
	return "Сканирование ветки на качество кода через SonarQube"
}

// executeContext holds shared state for a single Execute invocation.
type executeContext struct {
	start      time.Time
	traceID    string
	format     string
	log        *slog.Logger
	branch     string
	owner      string
	repo       string
	projectKey string
}

// validateAndSetup validates config and sets up the execution context.
// Returns nil executeContext and an error to return if validation fails.
func (h *ScanBranchHandler) validateAndSetup(cfg *config.Config) (*executeContext, error) {
	ec := &executeContext{start: time.Now()}

	ec.traceID = tracing.TraceIDFromContext(context.Background())
	if ec.traceID == "" {
		ec.traceID = tracing.GenerateTraceID()
	}
	ec.format = os.Getenv("BR_OUTPUT_FORMAT")

	if !dryrun.IsDryRun() && dryrun.IsPlanOnly() {
		return nil, dryrun.WritePlanOnlyUnsupported(os.Stdout, constants.ActNRSQScanBranch)
	}

	ec.log = slog.Default().With(slog.String("trace_id", ec.traceID), slog.String("command", constants.ActNRSQScanBranch))

	if cfg == nil {
		ec.log.Error("Конфигурация не загружена")
		return nil, h.writeError(ec.format, ec.traceID, ec.start, errConfigMissing, "Конфигурация не загружена")
	}

	ec.branch = cfg.BranchForScan
	if ec.branch == "" {
		ec.log.Error("Не указана ветка для сканирования")
		return nil, h.writeError(ec.format, ec.traceID, ec.start, errBranchMissing, "Не указана ветка для сканирования (BR_BRANCH)")
	}
	ec.log = ec.log.With(slog.String("branch", ec.branch))

	if !shared.IsValidBranchForScanning(ec.branch) {
		ec.log.Error("Ветка не соответствует критериям сканирования")
		return nil, h.writeError(ec.format, ec.traceID, ec.start, errBranchInvalidFormat,
			fmt.Sprintf("Ветка '%s' не соответствует критериям: допустимы только 'main' или 't' + 6-7 цифр", ec.branch))
	}

	ec.owner = cfg.Owner
	ec.repo = cfg.Repo
	if ec.owner == "" || ec.repo == "" {
		ec.log.Error("Не указаны owner или repo")
		return nil, h.writeError(ec.format, ec.traceID, ec.start, errConfigMissing, "Не указаны владелец (BR_OWNER) или репозиторий (BR_REPO)")
	}

	ec.projectKey = fmt.Sprintf("%s_%s_%s", ec.owner, ec.repo, ec.branch)
	ec.log = ec.log.With(slog.String("project_key", ec.projectKey))
	ec.log.Info("Запуск сканирования ветки")

	return ec, nil
}

// getClients returns Gitea and SonarQube clients (from handler or newly created).
func (h *ScanBranchHandler) getClients(ec *executeContext, cfg *config.Config) (gitea.Client, sonarqube.Client, error) {
	giteaClient := h.giteaClient
	if giteaClient == nil {
		var err error
		giteaClient, err = errhandler.CreateGiteaClient(cfg)
		if err != nil {
			ec.log.Error("Не удалось создать Gitea клиент", slog.String("error", err.Error()))
			return nil, nil, h.writeError(ec.format, ec.traceID, ec.start, errConfigMissing, "Не удалось создать Gitea клиент: "+err.Error())
		}
	}
	sqClient := h.sonarqubeClient
	if sqClient == nil {
		var err error
		sqClient, err = errhandler.CreateSonarQubeClient(cfg)
		if err != nil {
			ec.log.Error("Не удалось создать SonarQube клиент", slog.String("error", err.Error()))
			return nil, nil, h.writeError(ec.format, ec.traceID, ec.start, errConfigMissing, "Не удалось создать SonarQube клиент: "+err.Error())
		}
	}
	return giteaClient, sqClient, nil
}

// buildCandidates extracts candidate commit SHAs from the branch commit range.
func buildCandidates(commitRange *gitea.BranchCommitRange) []string {
	var candidates []string
	if commitRange.FirstCommit != nil && commitRange.FirstCommit.SHA != "" {
		candidates = append(candidates, commitRange.FirstCommit.SHA)
	}
	if commitRange.LastCommit != nil && commitRange.LastCommit.SHA != "" &&
		(commitRange.FirstCommit == nil || commitRange.FirstCommit.SHA != commitRange.LastCommit.SHA) {
		candidates = append(candidates, commitRange.LastCommit.SHA)
	}
	return candidates
}

// filterRelevantChanges splits candidates into those with changes and counts those without.
func filterRelevantChanges(ctx context.Context, giteaClient gitea.Client, branch string, candidates []string, log *slog.Logger) ([]string, int) {
	var commitsWithChanges []string
	noRelevantChangesCount := 0
	for _, sha := range candidates {
		hasChanges, err := shared.HasRelevantChangesInCommit(ctx, giteaClient, branch, sha)
		if err != nil {
			log.Warn("Ошибка проверки изменений в коммите", slog.String("commit", sha), slog.String("error", err.Error()))
			commitsWithChanges = append(commitsWithChanges, sha)
			continue
		}
		if hasChanges {
			commitsWithChanges = append(commitsWithChanges, sha)
		} else {
			noRelevantChangesCount++
		}
	}
	return commitsWithChanges, noRelevantChangesCount
}

// filterAlreadyScanned removes already-scanned commits and returns the rest with skip count.
func filterAlreadyScanned(ctx context.Context, sqClient sonarqube.Client, projectKey string, commits []string, log *slog.Logger) ([]string, int) {
	analyses, err := sqClient.GetAnalyses(ctx, projectKey)
	if err != nil {
		log.Warn("Не удалось получить список анализов", slog.String("error", err.Error()))
		return commits, 0
	}
	scannedRevisions := make(map[string]bool)
	for _, a := range analyses {
		scannedRevisions[a.Revision] = true
	}
	var toScan []string
	skippedCount := 0
	for _, sha := range commits {
		if scannedRevisions[sha] {
			skippedCount++
			log.Debug("Коммит уже отсканирован, пропускаем", slog.String("commit", sha))
			continue
		}
		toScan = append(toScan, sha)
	}
	return toScan, skippedCount
}

// ensureProject creates the SonarQube project if it does not exist.
func (h *ScanBranchHandler) ensureProject(ctx context.Context, ec *executeContext, sqClient sonarqube.Client) error {
	_, err := sqClient.GetProject(ctx, ec.projectKey)
	if err != nil {
		ec.log.Info("Проект не найден в SonarQube, создаём", slog.String("get_error", err.Error()))
		projectName := fmt.Sprintf("%s/%s (%s)", ec.owner, ec.repo, ec.branch)
		_, err = sqClient.CreateProject(ctx, sonarqube.CreateProjectOptions{
			Key: ec.projectKey, Name: projectName, Visibility: "private",
		})
		if err != nil {
			ec.log.Error("Не удалось создать проект в SonarQube", slog.String("error", err.Error()))
			return h.writeError(ec.format, ec.traceID, ec.start, errSonarQubeAPI,
				fmt.Sprintf("Не удалось создать проект в SonarQube: %v", err))
		}
	}
	return nil
}

// scanCommits runs analysis for each commit and collects results.
func (h *ScanBranchHandler) scanCommits(ctx context.Context, ec *executeContext, sqClient sonarqube.Client, toScan []string) []CommitScanResult {
	var scanResults []CommitScanResult
	for _, sha := range toScan {
		ec.log.Info("Сканирование коммита", slog.String("commit", sha))
		shortSHA := sha
		if len(sha) > 7 {
			shortSHA = sha[:7]
		}
		result, err := sqClient.RunAnalysis(ctx, sonarqube.RunAnalysisOptions{
			ProjectKey: ec.projectKey, Branch: ec.branch,
			Properties: map[string]string{"sonar.projectVersion": shortSHA, "sonar.scm.revision": sha},
		})
		if err != nil {
			ec.log.Error("Не удалось запустить анализ", slog.String("commit", sha), slog.String("error", err.Error()))
			scanResults = append(scanResults, CommitScanResult{CommitSHA: sha, Status: "FAILED", ErrorMessage: err.Error()})
			continue
		}
		status, err := shared.WaitForAnalysisCompletion(ctx, sqClient, result.TaskID, ec.log)
		if err != nil {
			ec.log.Error("Ошибка ожидания завершения анализа", slog.String("commit", sha), slog.String("error", err.Error()))
			scanResults = append(scanResults, CommitScanResult{CommitSHA: sha, AnalysisID: result.AnalysisID, Status: "FAILED", ErrorMessage: err.Error()})
			continue
		}
		if status.AnalysisID == "" && status.Status == "SUCCESS" {
			ec.log.Warn("Анализ завершён успешно, но AnalysisID пустой", slog.String("commit", sha))
		}
		scanResults = append(scanResults, CommitScanResult{CommitSHA: sha, AnalysisID: status.AnalysisID, Status: status.Status, ErrorMessage: status.ErrorMessage})
	}
	return scanResults
}

// Execute выполняет команду nr-sq-scan-branch.
func (h *ScanBranchHandler) Execute(ctx context.Context, cfg *config.Config) error {
	ec, err := h.validateAndSetup(cfg)
	if err != nil {
		return err
	}

	giteaClient, sqClient, err := h.getClients(ec, cfg)
	if err != nil {
		return err
	}

	// Получение диапазона коммитов ветки (AC: #4)
	commitRange, err := giteaClient.GetBranchCommitRange(ctx, ec.branch)
	if err != nil {
		ec.log.Error("Не удалось получить диапазон коммитов", slog.String("error", err.Error()))
		return h.writeError(ec.format, ec.traceID, ec.start, errGiteaAPI,
			fmt.Sprintf("Не удалось получить диапазон коммитов: %v", err))
	}

	candidates := buildCandidates(commitRange)
	if len(candidates) == 0 {
		ec.log.Info("Нет коммитов для сканирования")
		return h.writeSuccess(ec.format, ec.traceID, ec.start, &ScanBranchData{
			Branch: ec.branch, ProjectKey: ec.projectKey, NoChanges: true,
		})
	}

	// Проверка наличия изменений в каталогах конфигурации (AC: #3)
	commitsWithChanges, noRelevantChangesCount := filterRelevantChanges(ctx, giteaClient, ec.branch, candidates, ec.log)

	if len(commitsWithChanges) == 0 {
		ec.log.Info("Нет изменений в каталогах конфигурации")
		return h.writeSuccess(ec.format, ec.traceID, ec.start, &ScanBranchData{
			Branch: ec.branch, ProjectKey: ec.projectKey,
			NoRelevantChangesCount: noRelevantChangesCount, NoChanges: true,
		})
	}

	toScan, skippedCount := filterAlreadyScanned(ctx, sqClient, ec.projectKey, commitsWithChanges, ec.log)

	if len(toScan) == 0 {
		ec.log.Info("Все коммиты уже отсканированы")
		return h.writeSuccess(ec.format, ec.traceID, ec.start, &ScanBranchData{
			Branch: ec.branch, ProjectKey: ec.projectKey, SkippedCount: skippedCount,
		})
	}

	if err := h.ensureProject(ctx, ec, sqClient); err != nil {
		return err
	}

	scanResults := h.scanCommits(ctx, ec, sqClient, toScan)

	data := &ScanBranchData{
		Branch: ec.branch, ProjectKey: ec.projectKey,
		CommitsScanned: len(scanResults), SkippedCount: skippedCount,
		NoRelevantChangesCount: noRelevantChangesCount, ScanResults: scanResults,
	}

	ec.log.Info("Сканирование завершено",
		slog.Int("scanned", data.CommitsScanned), slog.Int("skipped", data.SkippedCount))

	return h.writeSuccess(ec.format, ec.traceID, ec.start, data)
}

// L-2 fix: isValidBranchForScanning перенесена в shared.IsValidBranchForScanning

// DELETED: hasRelevantChangesInCommit — используется shared.HasRelevantChangesInCommit
// DELETED: waitForAnalysisCompletion — используется shared.WaitForAnalysisCompletion
// DRY refactoring: общая логика вынесена в internal/command/handlers/sonarqube/shared/analysis.go

// writeSuccess выводит успешный результат.
func (h *ScanBranchHandler) writeSuccess(format, traceID string, start time.Time, data *ScanBranchData) error {
	// Текстовый формат
	if format != output.FormatJSON {
		return data.writeText(os.Stdout)
	}

	// JSON формат
	result := &output.Result{
		Status:  output.StatusSuccess,
		Command: constants.ActNRSQScanBranch,
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
func (h *ScanBranchHandler) writeError(format, traceID string, start time.Time, code, message string) error {
	// Текстовый формат — человекочитаемый вывод ошибки
	if format != output.FormatJSON {
		return errhandler.HandleError(message, code)
	}

	// JSON формат — структурированный вывод
	result := &output.Result{
		Status:  output.StatusError,
		Command: constants.ActNRSQScanBranch,
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
