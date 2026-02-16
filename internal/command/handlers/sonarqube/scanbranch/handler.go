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

func init() {
	command.RegisterWithAlias(&ScanBranchHandler{}, constants.ActSQScanBranch)
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

// Execute выполняет команду nr-sq-scan-branch.
func (h *ScanBranchHandler) Execute(ctx context.Context, cfg *config.Config) error {
	start := time.Now()

	traceID := tracing.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = tracing.GenerateTraceID()
	}

	format := os.Getenv("BR_OUTPUT_FORMAT")

	// Story 7.3 AC-8: plan-only для команд без поддержки плана
	// Review #36: !IsDryRun() — dry-run имеет приоритет над plan-only (AC-11).
	if !dryrun.IsDryRun() && dryrun.IsPlanOnly() {
		return dryrun.WritePlanOnlyUnsupported(os.Stdout, constants.ActNRSQScanBranch)
	}

	log := slog.Default().With(slog.String("trace_id", traceID), slog.String("command", constants.ActNRSQScanBranch))

	// Валидация конфигурации
	if cfg == nil {
		log.Error("Конфигурация не загружена")
		return h.writeError(format, traceID, start,
			errConfigMissing,
			"Конфигурация не загружена")
	}

	// Получение ветки для сканирования
	branch := cfg.BranchForScan
	if branch == "" {
		log.Error("Не указана ветка для сканирования")
		return h.writeError(format, traceID, start,
			errBranchMissing,
			"Не указана ветка для сканирования (BR_BRANCH)")
	}

	log = log.With(slog.String("branch", branch))

	// Валидация формата ветки (AC: #2)
	// L-2 fix: используем shared.IsValidBranchForScanning вместо локальной функции
	if !shared.IsValidBranchForScanning(branch) {
		// M-3 fix: branch уже добавлен в log context выше, не дублируем
		log.Error("Ветка не соответствует критериям сканирования")
		return h.writeError(format, traceID, start,
			errBranchInvalidFormat,
			fmt.Sprintf("Ветка '%s' не соответствует критериям: допустимы только 'main' или 't' + 6-7 цифр", branch))
	}

	log.Info("Запуск сканирования ветки")

	// Получение owner и repo из конфигурации
	owner := cfg.Owner
	repo := cfg.Repo
	if owner == "" || repo == "" {
		log.Error("Не указаны owner или repo")
		return h.writeError(format, traceID, start,
			errConfigMissing,
			"Не указаны владелец (BR_OWNER) или репозиторий (BR_REPO)")
	}

	// Формирование ключа проекта SonarQube
	projectKey := fmt.Sprintf("%s_%s_%s", owner, repo, branch)
	log = log.With(slog.String("project_key", projectKey))

	// Получение Gitea клиента
	// TODO(H-6): Реализовать фабрику createGiteaClient(cfg) для создания реального клиента.
	// Текущая реализация требует DI через поле giteaClient (используется в тестах).
	// Для production необходимо создать реализацию gitea.Client на основе internal/entity/gitea
	// или написать новую реализацию в internal/adapter/gitea/client.go.
	// См. паттерн: racutil.NewClient()
	giteaClient := h.giteaClient
	if giteaClient == nil {
		log.Error("Gitea клиент не настроен")
		return h.writeError(format, traceID, start,
			errConfigMissing,
			"Gitea клиент не настроен — требуется реализация фабрики createGiteaClient()")
	}

	// Получение SonarQube клиента
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

	// Получение диапазона коммитов ветки (AC: #4)
	commitRange, err := giteaClient.GetBranchCommitRange(ctx, branch)
	if err != nil {
		log.Error("Не удалось получить диапазон коммитов", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start,
			errGiteaAPI,
			fmt.Sprintf("Не удалось получить диапазон коммитов: %v", err))
	}

	// Формирование списка кандидатов для сканирования
	// H-2 fix: проверяем что SHA не пустой перед добавлением в candidates
	var candidates []string
	if commitRange.FirstCommit != nil && commitRange.FirstCommit.SHA != "" {
		candidates = append(candidates, commitRange.FirstCommit.SHA)
	}
	if commitRange.LastCommit != nil && commitRange.LastCommit.SHA != "" &&
		(commitRange.FirstCommit == nil || commitRange.FirstCommit.SHA != commitRange.LastCommit.SHA) {
		candidates = append(candidates, commitRange.LastCommit.SHA)
	}

	if len(candidates) == 0 {
		log.Info("Нет коммитов для сканирования")
		data := &ScanBranchData{
			Branch:         branch,
			ProjectKey:     projectKey,
			CommitsScanned: 0,
			SkippedCount:   0,
			NoChanges:      true,
		}
		return h.writeSuccess(format, traceID, start, data)
	}

	// Проверка наличия изменений в каталогах конфигурации (AC: #3)
	// M-2 fix: подсчитываем коммиты без релевантных изменений даже в смешанном сценарии
	var commitsWithChanges []string
	noRelevantChangesCount := 0
	for _, sha := range candidates {
		hasChanges, err := shared.HasRelevantChangesInCommit(ctx, giteaClient, branch, sha)
		if err != nil {
			log.Warn("Ошибка проверки изменений в коммите",
				slog.String("commit", sha),
				slog.String("error", err.Error()))
			// Продолжаем с этим коммитом, т.к. ошибка API не означает отсутствие изменений
			commitsWithChanges = append(commitsWithChanges, sha)
			continue
		}
		if hasChanges {
			commitsWithChanges = append(commitsWithChanges, sha)
		} else {
			noRelevantChangesCount++
		}
	}

	if len(commitsWithChanges) == 0 {
		log.Info("Нет изменений в каталогах конфигурации")
		data := &ScanBranchData{
			Branch:                 branch,
			ProjectKey:             projectKey,
			CommitsScanned:         0,
			SkippedCount:           0,
			NoRelevantChangesCount: noRelevantChangesCount,
			NoChanges:              true,
		}
		return h.writeSuccess(format, traceID, start, data)
	}

	// Получение уже отсканированных анализов (AC: #4)
	analyses, err := sqClient.GetAnalyses(ctx, projectKey)
	if err != nil {
		log.Warn("Не удалось получить список анализов", slog.String("error", err.Error()))
		// Продолжаем, считаем что ни один коммит не был отсканирован
		analyses = nil
	}

	// Формирование map отсканированных ревизий
	scannedRevisions := make(map[string]bool)
	for _, a := range analyses {
		scannedRevisions[a.Revision] = true
	}

	// Фильтрация уже отсканированных коммитов
	var toScan []string
	skippedCount := 0
	for _, sha := range commitsWithChanges {
		if scannedRevisions[sha] {
			skippedCount++
			log.Debug("Коммит уже отсканирован, пропускаем", slog.String("commit", sha))
			continue
		}
		toScan = append(toScan, sha)
	}

	if len(toScan) == 0 {
		log.Info("Все коммиты уже отсканированы")
		data := &ScanBranchData{
			Branch:         branch,
			ProjectKey:     projectKey,
			CommitsScanned: 0,
			SkippedCount:   skippedCount,
		}
		return h.writeSuccess(format, traceID, start, data)
	}

	// Получение или создание проекта в SonarQube (AC: #5)
	// M-2 fix: логируем ошибку GetProject для диагностики (может быть не только "not found")
	// L-2 fix: убрано дублирование project_key (уже в log.With выше)
	_, err = sqClient.GetProject(ctx, projectKey)
	if err != nil {
		// Предполагаем что проект не найден, но логируем ошибку для диагностики
		log.Info("Проект не найден в SonarQube, создаём",
			slog.String("get_error", err.Error()))
		projectName := fmt.Sprintf("%s/%s (%s)", owner, repo, branch)
		// TODO(L-1): Visibility hardcoded как "private". Для некоторых организаций
		// может требоваться "public". Добавить cfg.SonarQubeVisibility или cfg.DefaultVisibility.
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

	// Запуск сканирования для каждого коммита (AC: #5, #7)
	var scanResults []CommitScanResult
	for _, sha := range toScan {
		log.Info("Сканирование коммита", slog.String("commit", sha))

		// H-1 fix: передаём sonar.scm.revision с полным SHA для корректной фильтрации
		// уже отсканированных коммитов через GetAnalyses (AC #4)
		// Защита от короткого SHA (panic при sha[:7])
		shortSHA := sha
		if len(sha) > 7 {
			shortSHA = sha[:7]
		}
		// TODO(H-7): SourcePath не заполняется — требуется добавить cfg.WorkDir или cfg.SourcePath
		// для указания пути к исходному коду. Без этого sonar-scanner не знает где искать файлы.
		// См. sonarqube.RunAnalysisOptions.SourcePath в internal/adapter/sonarqube/interfaces.go:186
		result, err := sqClient.RunAnalysis(ctx, sonarqube.RunAnalysisOptions{
			ProjectKey: projectKey,
			Branch:     branch,
			Properties: map[string]string{
				"sonar.projectVersion": shortSHA,
				"sonar.scm.revision":   sha, // Полный SHA для Revision в Analysis
			},
		})
		if err != nil {
			log.Error("Не удалось запустить анализ", slog.String("commit", sha), slog.String("error", err.Error()))
			scanResults = append(scanResults, CommitScanResult{
				CommitSHA:    sha,
				Status:       "FAILED",
				ErrorMessage: err.Error(),
			})
			continue
		}

		// Ожидание завершения анализа (AC: #5)
		status, err := shared.WaitForAnalysisCompletion(ctx, sqClient, result.TaskID, log)
		if err != nil {
			log.Error("Ошибка ожидания завершения анализа", slog.String("commit", sha), slog.String("error", err.Error()))
			scanResults = append(scanResults, CommitScanResult{
				CommitSHA:    sha,
				AnalysisID:   result.AnalysisID,
				Status:       "FAILED",
				ErrorMessage: err.Error(),
			})
			continue
		}

		// L-2 fix: проверяем пустой AnalysisID после успешного анализа
		if status.AnalysisID == "" && status.Status == "SUCCESS" {
			log.Warn("Анализ завершён успешно, но AnalysisID пустой", slog.String("commit", sha))
		}

		scanResults = append(scanResults, CommitScanResult{
			CommitSHA:    sha,
			AnalysisID:   status.AnalysisID,
			Status:       status.Status,
			ErrorMessage: status.ErrorMessage,
		})
	}

	// Формирование результата (AC: #5, #6)
	// M-2 fix: включаем NoRelevantChangesCount даже когда есть коммиты с изменениями
	data := &ScanBranchData{
		Branch:                 branch,
		ProjectKey:             projectKey,
		CommitsScanned:         len(scanResults),
		SkippedCount:           skippedCount,
		NoRelevantChangesCount: noRelevantChangesCount,
		ScanResults:            scanResults,
	}

	log.Info("Сканирование завершено",
		slog.Int("scanned", data.CommitsScanned),
		slog.Int("skipped", data.SkippedCount))

	return h.writeSuccess(format, traceID, start, data)
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
		_, _ = fmt.Fprintf(os.Stdout, "Ошибка: %s\nКод: %s\n", message, code)
		return fmt.Errorf("%s: %s", code, message)
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
