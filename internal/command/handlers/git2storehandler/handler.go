// Package git2storehandler реализует NR-команду nr-git2store
// для синхронизации EDT из Git в хранилище 1C.
package git2storehandler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
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

// Compile-time interface check (AC-1).
var _ command.Handler = (*Git2StoreHandler)(nil)

func init() {
	// AC-6: Deprecated alias через DeprecatedBridge
	command.RegisterWithAlias(&Git2StoreHandler{}, constants.ActGit2store)
}

// Stage constants для workflow (AC-2, AC-3).
const (
	StageValidating     = "validating"
	StageCreatingBackup = "creating_backup"
	StageCloning        = "cloning"
	StageCheckoutEdt    = "checkout_edt"
	StageCreatingTempDb = "creating_temp_db"
	StageLoadingConfig  = "loading_config"
	StageCheckoutXml    = "checkout_xml"
	StageInitDb         = "init_db"
	StageUnbinding      = "unbinding"
	StageLoadingDb      = "loading_db"
	StageUpdatingDb1    = "updating_db_1"
	StageDumpingDb      = "dumping_db"
	StageBinding        = "binding"
	StageUpdatingDb2    = "updating_db_2"
	StageLocking        = "locking"
	StageMerging        = "merging"
	StageUpdatingDb3    = "updating_db_3"
	StageCommitting     = "committing"
)

// allStages содержит все этапы workflow в порядке выполнения.
var allStages = []string{
	StageValidating,
	StageCreatingBackup,
	StageCloning,
	StageCheckoutEdt,
	StageCreatingTempDb,
	StageLoadingConfig,
	StageCheckoutXml,
	StageInitDb,
	StageUnbinding,
	StageLoadingDb,
	StageUpdatingDb1,
	StageDumpingDb,
	StageBinding,
	StageUpdatingDb2,
	StageLocking,
	StageMerging,
	StageUpdatingDb3,
	StageCommitting,
}

// Git2StoreData содержит данные ответа о синхронизации Git → Store (AC-4).
type Git2StoreData struct {
	// StateChanged — изменилось ли состояние системы (AC-12)
	StateChanged bool `json:"state_changed"`
	// StagesCompleted — список завершённых этапов
	StagesCompleted []StageResult `json:"stages_completed"`
	// StageCurrent — текущий/последний этап
	StageCurrent string `json:"stage_current"`
	// BackupPath — путь к backup хранилища (AC-8)
	BackupPath string `json:"backup_path"`
	// DurationMs — длительность операции в миллисекундах
	DurationMs int64 `json:"duration_ms"`
	// Errors — список ошибок (AC-4). Без omitempty чтобы всегда выводить пустой массив.
	Errors []string `json:"errors"`
}

// StageResult результат выполнения этапа (AC-3).
type StageResult struct {
	// Name — имя этапа
	Name string `json:"name"`
	// Success — успешно ли выполнен
	Success bool `json:"success"`
	// DurationMs — длительность этапа в миллисекундах
	DurationMs int64 `json:"duration_ms"`
	// Error — ошибка этапа (если была)
	Error string `json:"error,omitempty"`
}

// writeText выводит результат в человекочитаемом формате (AC-5).
func (d *Git2StoreData) writeText(w io.Writer) error {
	// Статус операции
	statusText := "успешно"
	if !d.StateChanged {
		statusText = "без изменений"
	}

	_, err := fmt.Fprintf(w, "Синхронизация Git → хранилище 1C: %s\n", statusText)
	if err != nil {
		return err
	}

	// Progress bar
	completedCount := 0
	for _, stage := range d.StagesCompleted {
		if stage.Success {
			completedCount++
		}
	}

	if _, err = fmt.Fprintf(w, "\nПрогресс: [%d/%d] этапов\n", completedCount, len(allStages)); err != nil {
		return err
	}

	// Summary
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

	// Выводим детали этапов
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

// GitOperator — интерфейс для Git операций (для тестируемости).
type GitOperator interface {
	// Clone клонирует репозиторий
	Clone(ctx context.Context, l *slog.Logger) error
	// Switch переключается на ветку
	Switch(ctx context.Context, l *slog.Logger) error
	// SetBranch устанавливает ветку
	SetBranch(branch string)
}

// ConvertConfigOperator — интерфейс для операций convert.Config (для тестируемости).
type ConvertConfigOperator interface {
	// Load загружает конфигурацию конвертации
	Load(ctx context.Context, l *slog.Logger, cfg *config.Config, infobaseName string) error
	// InitDb инициализирует базу данных
	InitDb(ctx context.Context, l *slog.Logger, cfg *config.Config) error
	// StoreUnBind отвязывает от хранилища
	StoreUnBind(ctx context.Context, l *slog.Logger, cfg *config.Config) error
	// LoadDb загружает конфигурацию в БД
	LoadDb(ctx context.Context, l *slog.Logger, cfg *config.Config) error
	// DbUpdate обновляет БД
	DbUpdate(ctx context.Context, l *slog.Logger, cfg *config.Config) error
	// DumpDb выгружает БД
	DumpDb(ctx context.Context, l *slog.Logger, cfg *config.Config) error
	// StoreBind привязывает к хранилищу
	StoreBind(ctx context.Context, l *slog.Logger, cfg *config.Config) error
	// StoreLock блокирует объекты в хранилище
	StoreLock(ctx context.Context, l *slog.Logger, cfg *config.Config) error
	// Merge выполняет слияние
	Merge(ctx context.Context, l *slog.Logger, cfg *config.Config) error
	// StoreCommit фиксирует изменения в хранилище
	StoreCommit(ctx context.Context, l *slog.Logger, cfg *config.Config) error
	// SetOneDB устанавливает параметры временной БД
	SetOneDB(dbConnectString, user, pass string)
}

// BackupCreator — интерфейс для создания backup (AC-8, для тестируемости).
type BackupCreator interface {
	// CreateBackup создаёт резервную копию хранилища
	CreateBackup(cfg *config.Config, storeRoot string) (string, error)
}

// TempDbCreator — интерфейс для создания временной БД (для тестируемости).
type TempDbCreator interface {
	// CreateTempDb создаёт временную базу данных и возвращает строку подключения
	CreateTempDb(ctx context.Context, l *slog.Logger, cfg *config.Config) (string, error)
}

// GitFactory — интерфейс для создания GitOperator (для тестируемости).
type GitFactory interface {
	// CreateGit создаёт GitOperator
	CreateGit(l *slog.Logger, cfg *config.Config) (GitOperator, error)
}

// ConvertConfigFactory — интерфейс для создания ConvertConfigOperator (для тестируемости).
type ConvertConfigFactory interface {
	// CreateConvertConfig создаёт ConvertConfigOperator
	CreateConvertConfig() ConvertConfigOperator
}

// Git2StoreHandler обрабатывает команду nr-git2store.
//
// TODO: [H-5] Рефакторинг: вынести фабрики (gitFactory, convertConfigFactory,
// backupCreator, tempDbCreator) в общий пакет internal/factory или использовать
// Wire для DI. H-2 (createRACClient) уже решён — см. racutil.NewClient().
type Git2StoreHandler struct {
	// gitFactory — опциональная фабрика Git (nil в production, mock в тестах)
	gitFactory GitFactory
	// convertConfigFactory — опциональная фабрика ConvertConfig (nil в production, mock в тестах)
	convertConfigFactory ConvertConfigFactory
	// backupCreator — опциональный создатель backup (nil в production, mock в тестах)
	backupCreator BackupCreator
	// tempDbCreator — опциональный создатель временной БД (nil в production, mock в тестах)
	tempDbCreator TempDbCreator
	// verbosePlan — план операций для verbose режима (Story 7.3), добавляется в JSON результат
	verbosePlan *output.DryRunPlan
}

// Name возвращает имя команды.
func (h *Git2StoreHandler) Name() string {
	return constants.ActNRGit2store
}

// Description возвращает описание команды для вывода в help.
func (h *Git2StoreHandler) Description() string {
	return "Синхронизация Git → хранилище 1C"
}

// defaultGit2StoreTimeout — timeout по умолчанию для операции git2store (2 часа).
// Операции clone, update DB и commit могут занимать значительное время.
const defaultGit2StoreTimeout = 2 * time.Hour

// Execute выполняет команду nr-git2store (AC-1, AC-2).
func (h *Git2StoreHandler) Execute(ctx context.Context, cfg *config.Config) error {
	start := time.Now()

	// Устанавливаем timeout для всей операции
	timeout := defaultGit2StoreTimeout
	if envTimeout := os.Getenv("BR_GIT2STORE_TIMEOUT"); envTimeout != "" {
		if parsed, err := time.ParseDuration(envTimeout); err == nil {
			timeout = parsed
		} else {
			// Используем slog.Default() так как локальный log ещё не инициализирован
			slog.Default().Warn("Невалидный формат BR_GIT2STORE_TIMEOUT, используется значение по умолчанию",
				slog.String("value", envTimeout),
				slog.String("default", defaultGit2StoreTimeout.String()),
				slog.String("error", err.Error()))
		}
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	traceID := tracing.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = tracing.GenerateTraceID()
	}

	format := os.Getenv("BR_OUTPUT_FORMAT")
	log := slog.Default().With(
		slog.String("trace_id", traceID),
		slog.String("command", constants.ActNRGit2store),
	)

	data := &Git2StoreData{
		StagesCompleted: make([]StageResult, 0),
		Errors:          make([]string, 0),
	}

	// === РЕЖИМЫ ПРЕДПРОСМОТРА (порядок приоритетов!) ===

	// 1. Dry-run: план без выполнения (высший приоритет)
	if dryrun.IsDryRun() {
		return h.executeDryRun(cfg, format, traceID, start, log)
	}

	// 2. Plan-only: показать план, не выполнять (Story 7.3 AC-1)
	if dryrun.IsPlanOnly() {
		log.Info("Plan-only режим: отображение плана операций")
		plan := h.buildPlan(cfg)
		return output.WritePlanOnlyResult(os.Stdout, format, constants.ActNRGit2store, traceID, constants.APIVersion, start, plan)
	}

	// 3. Verbose: показать план, ПОТОМ выполнить (Story 7.3 AC-4)
	if dryrun.IsVerbose() {
		log.Info("Verbose режим: отображение плана перед выполнением")
		plan := h.buildPlan(cfg)
		if format != output.FormatJSON {
			if err := plan.WritePlanText(os.Stdout); err != nil {
				log.Warn("Не удалось вывести план операций", slog.String("error", err.Error()))
			}
			fmt.Fprintln(os.Stdout)
		}
		h.verbosePlan = plan
	}
	// Verbose fall-through by design: план отображён, продолжаем реальное выполнение

	// Stage: validating (AC-2)
	if err := h.executeStageValidating(cfg, log, data); err != nil {
		return h.writeStageError(format, traceID, start, data, err)
	}

	// Генерируем storeRoot
	storeRoot := constants.StoreRoot + cfg.Owner + "/" + cfg.Repo

	// Stage: creating_backup (AC-8, MANDATORY!)
	backupPath, err := h.executeStageCreatingBackup(cfg, log, data, storeRoot)
	if err != nil {
		return h.writeStageError(format, traceID, start, data, err)
	}
	data.BackupPath = backupPath

	// Stage: cloning (AC-2)
	gitOp, err := h.executeStageCloning(ctx, cfg, log, data)
	if err != nil {
		return h.writeStageError(format, traceID, start, data, err)
	}
	// Очистка временной директории репозитория после завершения (успех или ошибка)
	defer func() {
		if cfg.RepPath != "" {
			if removeErr := os.RemoveAll(cfg.RepPath); removeErr != nil {
				log.Warn("Не удалось удалить временную директорию репозитория",
					slog.String("path", cfg.RepPath),
					slog.String("error", removeErr.Error()))
			} else {
				log.Debug("Временная директория репозитория удалена", slog.String("path", cfg.RepPath))
			}
		}
	}()

	// Stage: checkout_edt (AC-2)
	if err := h.executeStageCheckoutEdt(ctx, cfg, log, data, gitOp); err != nil {
		return h.writeStageError(format, traceID, start, data, err)
	}

	// Stage: creating_temp_db (optional, если StoreDb == LocalBase)
	ccOp := h.createConvertConfig()
	var tempDbPath string // H-1: Путь для cleanup временной БД
	// H-1 fix: Проверка cfg.ProjectConfig на nil перед доступом к StoreDb
	if cfg.ProjectConfig != nil && cfg.ProjectConfig.StoreDb == constants.LocalBase {
		var tempErr error
		tempDbPath, tempErr = h.executeStageCreatingTempDb(ctx, cfg, log, data, ccOp)
		if tempErr != nil {
			return h.writeStageError(format, traceID, start, data, tempErr)
		}
		// H-1: Cleanup временной БД после завершения (успех или ошибка)
		defer func() {
			if tempDbPath != "" {
				if removeErr := os.RemoveAll(tempDbPath); removeErr != nil {
					log.Warn("Не удалось удалить временную БД",
						slog.String("path", tempDbPath),
						slog.String("error", removeErr.Error()))
				} else {
					log.Debug("Временная БД удалена", slog.String("path", tempDbPath))
				}
			}
		}()
	}

	// Stage: loading_config (AC-2)
	if err := h.executeStageLoadingConfig(ctx, cfg, log, data, ccOp); err != nil {
		return h.writeStageError(format, traceID, start, data, err)
	}

	// Stage: checkout_xml (AC-2)
	if err := h.executeStageCheckoutXml(ctx, cfg, log, data, gitOp); err != nil {
		return h.writeStageError(format, traceID, start, data, err)
	}

	// Stage: init_db (AC-2)
	if err := h.executeStageInitDb(ctx, cfg, log, data, ccOp); err != nil {
		return h.writeStageError(format, traceID, start, data, err)
	}

	// Stage: unbinding (AC-2)
	if err := h.executeStageUnbinding(ctx, cfg, log, data, ccOp); err != nil {
		return h.writeStageError(format, traceID, start, data, err)
	}

	// Stage: loading_db (AC-2)
	if err := h.executeStageLoadingDb(ctx, cfg, log, data, ccOp); err != nil {
		return h.writeStageError(format, traceID, start, data, err)
	}

	// Stage: updating_db_1 (AC-2)
	if err := h.executeStageUpdatingDb(ctx, cfg, log, data, ccOp, StageUpdatingDb1); err != nil {
		return h.writeStageError(format, traceID, start, data, err)
	}

	// Stage: dumping_db (AC-2)
	if err := h.executeStageDumpingDb(ctx, cfg, log, data, ccOp); err != nil {
		return h.writeStageError(format, traceID, start, data, err)
	}

	// Stage: binding (AC-2)
	if err := h.executeStageBinding(ctx, cfg, log, data, ccOp); err != nil {
		return h.writeStageError(format, traceID, start, data, err)
	}

	// Stage: updating_db_2 (AC-2)
	if err := h.executeStageUpdatingDb(ctx, cfg, log, data, ccOp, StageUpdatingDb2); err != nil {
		return h.writeStageError(format, traceID, start, data, err)
	}

	// Stage: locking (AC-2)
	if err := h.executeStageLocking(ctx, cfg, log, data, ccOp); err != nil {
		return h.writeStageError(format, traceID, start, data, err)
	}

	// Stage: merging (AC-2)
	if err := h.executeStageMerging(ctx, cfg, log, data, ccOp); err != nil {
		return h.writeStageError(format, traceID, start, data, err)
	}

	// Stage: updating_db_3 (AC-2, AC-13 для расширений)
	if err := h.executeStageUpdatingDb(ctx, cfg, log, data, ccOp, StageUpdatingDb3); err != nil {
		return h.writeStageError(format, traceID, start, data, err)
	}

	// Stage: committing (AC-2)
	if err := h.executeStageCommitting(ctx, cfg, log, data, ccOp); err != nil {
		return h.writeStageError(format, traceID, start, data, err)
	}

	// Формирование результата (AC-4, AC-12)
	data.DurationMs = time.Since(start).Milliseconds()
	data.StateChanged = true           // AC-12: успешная синхронизация всегда меняет состояние
	data.StageCurrent = "completed"    // M-2: Устанавливаем финальный статус
	// data.Errors уже инициализирован как пустой срез в начале Execute (AC-4)

	log.Info("Синхронизация Git → хранилище 1C успешно завершена",
		slog.String("backup_path", data.BackupPath),
		slog.Int("stages_completed", len(data.StagesCompleted)),
		slog.Int64("duration_ms", data.DurationMs))

	return h.writeSuccess(format, traceID, data)
}

// buildPlan создаёт план операций для предпросмотра.
// Используется в dry-run, plan-only и verbose режимах.
// Story 7.3: извлечено из executeDryRun для переиспользования.
// Конвертирует allStages в output.PlanStep[] для единообразия с другими handlers.
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

	// Добавляем расширения к первому шагу если есть
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

// executeStageValidating выполняет этап валидации (AC-2).
func (h *Git2StoreHandler) executeStageValidating(cfg *config.Config, log *slog.Logger, data *Git2StoreData) error {
	stageStart := time.Now()
	stageName := StageValidating
	log.Info(stageName + ": проверка параметров")
	data.StageCurrent = stageName

	// Валидация конфигурации
	if cfg == nil {
		return h.recordStageError(data, stageName, stageStart, "CONFIG.MISSING", "Конфигурация не указана")
	}

	if cfg.AppConfig == nil || cfg.AppConfig.Paths.Bin1cv8 == "" {
		return h.recordStageError(data, stageName, stageStart, "CONFIG.BIN1CV8_MISSING",
			"Не указан путь к 1cv8 (AppConfig.Paths.Bin1cv8)")
	}

	if cfg.TmpDir == "" {
		return h.recordStageError(data, stageName, stageStart, "CONFIG.TMPDIR_MISSING",
			"Не указана временная директория (TmpDir)")
	}

	if cfg.Owner == "" {
		return h.recordStageError(data, stageName, stageStart, "CONFIG.OWNER_MISSING",
			"Не указан владелец репозитория (Owner)")
	}

	if cfg.Repo == "" {
		return h.recordStageError(data, stageName, stageStart, "CONFIG.REPO_MISSING",
			"Не указано имя репозитория (Repo)")
	}

	if cfg.WorkDir == "" {
		return h.recordStageError(data, stageName, stageStart, "CONFIG.WORKDIR_MISSING",
			"Не указана рабочая директория (WorkDir)")
	}

	if cfg.InfobaseName == "" {
		return h.recordStageError(data, stageName, stageStart, "CONFIG.INFOBASE_MISSING",
			"Не указано имя информационной базы (BR_INFOBASE_NAME)")
	}

	log.Info("Параметры валидации пройдены",
		slog.String("owner", cfg.Owner),
		slog.String("repo", cfg.Repo),
		slog.Int("extensions_count", len(cfg.AddArray)))

	h.recordStageSuccess(data, stageName, stageStart)
	return nil
}

// executeStageCreatingBackup создаёт резервную копию хранилища (AC-8).
func (h *Git2StoreHandler) executeStageCreatingBackup(cfg *config.Config, log *slog.Logger, data *Git2StoreData, storeRoot string) (string, error) {
	stageStart := time.Now()
	stageName := StageCreatingBackup
	log.Info(stageName + ": создание резервной копии хранилища")
	data.StageCurrent = stageName

	backupPath, err := h.createBackup(cfg, storeRoot)
	if err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_BACKUP", err.Error())
		return "", fmt.Errorf("ERR_GIT2STORE_BACKUP: %s", err.Error())
	}

	log.Info(stageName+": резервная копия создана", slog.String("backup_path", backupPath))
	h.recordStageSuccess(data, stageName, stageStart)
	return backupPath, nil
}

// executeStageCloning клонирует репозиторий (AC-2).
func (h *Git2StoreHandler) executeStageCloning(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData) (GitOperator, error) {
	stageStart := time.Now()
	stageName := StageCloning
	log.Info(stageName + ": клонирование репозитория")
	data.StageCurrent = stageName

	// Создаём временную директорию для репозитория
	repPath, err := os.MkdirTemp(cfg.WorkDir, "s")
	if err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_CLONE", err.Error())
		return nil, fmt.Errorf("ERR_GIT2STORE_CLONE: %s", err.Error())
	}
	cfg.RepPath = repPath

	// Создаём Git оператор
	gitOp, err := h.createGit(log, cfg)
	if err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_CLONE", err.Error())
		return nil, fmt.Errorf("ERR_GIT2STORE_CLONE: %s", err.Error())
	}

	// Клонируем репозиторий
	if err := gitOp.Clone(ctx, log); err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_CLONE", err.Error())
		return nil, fmt.Errorf("ERR_GIT2STORE_CLONE: %s", err.Error())
	}

	log.Info(stageName+": репозиторий клонирован", slog.String("rep_path", repPath))
	h.recordStageSuccess(data, stageName, stageStart)
	return gitOp, nil
}

// executeStageCheckoutEdt переключается на EDT ветку (AC-2).
func (h *Git2StoreHandler) executeStageCheckoutEdt(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData, gitOp GitOperator) error {
	stageStart := time.Now()
	stageName := StageCheckoutEdt
	log.Info(stageName+": переключение на EDT ветку", slog.String("branch", constants.EdtBranch))
	data.StageCurrent = stageName

	gitOp.SetBranch(constants.EdtBranch)
	if err := gitOp.Switch(ctx, log); err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_CHECKOUT", err.Error())
		return fmt.Errorf("ERR_GIT2STORE_CHECKOUT: %s", err.Error())
	}

	log.Info(stageName + ": переключено на EDT ветку")
	h.recordStageSuccess(data, stageName, stageStart)
	return nil
}

// executeStageCreatingTempDb создаёт временную БД (AC-2).
// Возвращает путь к созданной БД для последующей очистки (H-1).
func (h *Git2StoreHandler) executeStageCreatingTempDb(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData, ccOp ConvertConfigOperator) (string, error) {
	stageStart := time.Now()
	stageName := StageCreatingTempDb
	log.Info(stageName + ": создание временной базы данных")
	data.StageCurrent = stageName

	dbConnectString, err := h.createTempDb(ctx, log, cfg)
	if err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_TEMP_DB", err.Error())
		return "", fmt.Errorf("ERR_GIT2STORE_TEMP_DB: %s", err.Error())
	}

	// Извлекаем путь к БД из строки подключения для cleanup (H-1)
	// Формат: "/F /path/to/db" для файловой БД
	tempDbPath := dbConnectString
	if strings.HasPrefix(dbConnectString, "/F ") {
		tempDbPath = strings.TrimPrefix(dbConnectString, "/F ")
	}

	// Устанавливаем параметры временной БД в convert.Config
	user := constants.DefaultUser
	pass := constants.DefaultPass
	if cfg.AppConfig != nil && cfg.AppConfig.Users.Db != "" {
		user = cfg.AppConfig.Users.Db
	}
	if cfg.SecretConfig != nil && cfg.SecretConfig.Passwords.Db != "" {
		pass = cfg.SecretConfig.Passwords.Db
	}
	ccOp.SetOneDB(dbConnectString, user, pass)

	log.Info(stageName+": временная БД создана", slog.String("path", tempDbPath))
	h.recordStageSuccess(data, stageName, stageStart)
	return tempDbPath, nil
}

// executeStageLoadingConfig загружает конфигурацию конвертации (AC-2).
func (h *Git2StoreHandler) executeStageLoadingConfig(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData, ccOp ConvertConfigOperator) error {
	stageStart := time.Now()
	stageName := StageLoadingConfig
	log.Info(stageName + ": загрузка конфигурации конвертации")
	data.StageCurrent = stageName

	if err := ccOp.Load(ctx, log, cfg, cfg.InfobaseName); err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_LOAD", err.Error())
		return fmt.Errorf("ERR_GIT2STORE_LOAD: %s", err.Error())
	}

	log.Info(stageName + ": конфигурация загружена")
	h.recordStageSuccess(data, stageName, stageStart)
	return nil
}

// executeStageCheckoutXml переключается на XML ветку (AC-2).
func (h *Git2StoreHandler) executeStageCheckoutXml(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData, gitOp GitOperator) error {
	stageStart := time.Now()
	stageName := StageCheckoutXml
	log.Info(stageName+": переключение на XML ветку", slog.String("branch", constants.OneCBranch))
	data.StageCurrent = stageName

	gitOp.SetBranch(constants.OneCBranch)
	if err := gitOp.Switch(ctx, log); err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_CHECKOUT", err.Error())
		return fmt.Errorf("ERR_GIT2STORE_CHECKOUT: %s", err.Error())
	}

	log.Info(stageName + ": переключено на XML ветку")
	h.recordStageSuccess(data, stageName, stageStart)
	return nil
}

// executeStageInitDb инициализирует базу данных (AC-2).
func (h *Git2StoreHandler) executeStageInitDb(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData, ccOp ConvertConfigOperator) error {
	stageStart := time.Now()
	stageName := StageInitDb
	log.Info(stageName + ": инициализация базы данных")
	data.StageCurrent = stageName

	if err := ccOp.InitDb(ctx, log, cfg); err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_INIT_DB", err.Error())
		return fmt.Errorf("ERR_GIT2STORE_INIT_DB: %s", err.Error())
	}

	log.Info(stageName + ": база данных инициализирована")
	h.recordStageSuccess(data, stageName, stageStart)
	return nil
}

// executeStageUnbinding отключает от хранилища (AC-2).
func (h *Git2StoreHandler) executeStageUnbinding(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData, ccOp ConvertConfigOperator) error {
	stageStart := time.Now()
	stageName := StageUnbinding
	log.Info(stageName + ": отключение от хранилища")
	data.StageCurrent = stageName

	if err := ccOp.StoreUnBind(ctx, log, cfg); err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_UNBIND", err.Error())
		return fmt.Errorf("ERR_GIT2STORE_UNBIND: %s", err.Error())
	}

	log.Info(stageName + ": отключено от хранилища")
	h.recordStageSuccess(data, stageName, stageStart)
	return nil
}

// executeStageLoadingDb загружает конфигурацию в БД (AC-2).
func (h *Git2StoreHandler) executeStageLoadingDb(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData, ccOp ConvertConfigOperator) error {
	stageStart := time.Now()
	stageName := StageLoadingDb
	log.Info(stageName + ": загрузка конфигурации в базу данных")
	data.StageCurrent = stageName

	if err := ccOp.LoadDb(ctx, log, cfg); err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_LOAD_DB", err.Error())
		return fmt.Errorf("ERR_GIT2STORE_LOAD_DB: %s", err.Error())
	}

	log.Info(stageName + ": конфигурация загружена в БД")
	h.recordStageSuccess(data, stageName, stageStart)
	return nil
}

// executeStageUpdatingDb обновляет БД (AC-2).
func (h *Git2StoreHandler) executeStageUpdatingDb(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData, ccOp ConvertConfigOperator, stageName string) error {
	stageStart := time.Now()
	log.Info(stageName + ": обновление базы данных")
	data.StageCurrent = stageName

	if err := ccOp.DbUpdate(ctx, log, cfg); err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_UPDATE", err.Error())
		return fmt.Errorf("ERR_GIT2STORE_UPDATE: %s", err.Error())
	}

	log.Info(stageName + ": база данных обновлена")
	h.recordStageSuccess(data, stageName, stageStart)
	return nil
}

// executeStageDumpingDb выгружает БД (AC-2).
func (h *Git2StoreHandler) executeStageDumpingDb(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData, ccOp ConvertConfigOperator) error {
	stageStart := time.Now()
	stageName := StageDumpingDb
	log.Info(stageName + ": выгрузка базы данных")
	data.StageCurrent = stageName

	if err := ccOp.DumpDb(ctx, log, cfg); err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_DUMP", err.Error())
		return fmt.Errorf("ERR_GIT2STORE_DUMP: %s", err.Error())
	}

	log.Info(stageName + ": база данных выгружена")
	h.recordStageSuccess(data, stageName, stageStart)
	return nil
}

// executeStageBinding привязывает к хранилищу (AC-2).
func (h *Git2StoreHandler) executeStageBinding(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData, ccOp ConvertConfigOperator) error {
	stageStart := time.Now()
	stageName := StageBinding
	log.Info(stageName + ": привязка к хранилищу")
	data.StageCurrent = stageName

	if err := ccOp.StoreBind(ctx, log, cfg); err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_BIND", err.Error())
		return fmt.Errorf("ERR_GIT2STORE_BIND: %s", err.Error())
	}

	log.Info(stageName + ": привязано к хранилищу")
	h.recordStageSuccess(data, stageName, stageStart)
	return nil
}

// executeStageLocking блокирует объекты в хранилище (AC-2).
func (h *Git2StoreHandler) executeStageLocking(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData, ccOp ConvertConfigOperator) error {
	stageStart := time.Now()
	stageName := StageLocking
	log.Info(stageName + ": блокировка объектов в хранилище")
	data.StageCurrent = stageName

	if err := ccOp.StoreLock(ctx, log, cfg); err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_LOCK", err.Error())
		return fmt.Errorf("ERR_GIT2STORE_LOCK: %s", err.Error())
	}

	log.Info(stageName + ": объекты заблокированы")
	h.recordStageSuccess(data, stageName, stageStart)
	return nil
}

// executeStageMerging выполняет слияние (AC-2).
func (h *Git2StoreHandler) executeStageMerging(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData, ccOp ConvertConfigOperator) error {
	stageStart := time.Now()
	stageName := StageMerging
	log.Info(stageName + ": слияние конфигураций")
	data.StageCurrent = stageName

	if err := ccOp.Merge(ctx, log, cfg); err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_MERGE", err.Error())
		return fmt.Errorf("ERR_GIT2STORE_MERGE: %s", err.Error())
	}

	log.Info(stageName + ": слияние выполнено")
	h.recordStageSuccess(data, stageName, stageStart)
	return nil
}

// executeStageCommitting фиксирует изменения в хранилище (AC-2).
func (h *Git2StoreHandler) executeStageCommitting(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData, ccOp ConvertConfigOperator) error {
	stageStart := time.Now()
	stageName := StageCommitting
	log.Info(stageName + ": фиксация изменений в хранилище")
	data.StageCurrent = stageName

	if err := ccOp.StoreCommit(ctx, log, cfg); err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_COMMIT", err.Error())
		return fmt.Errorf("ERR_GIT2STORE_COMMIT: %s", err.Error())
	}

	log.Info(stageName + ": изменения зафиксированы")
	h.recordStageSuccess(data, stageName, stageStart)
	return nil
}

// recordStageSuccess записывает успешный результат этапа.
func (h *Git2StoreHandler) recordStageSuccess(data *Git2StoreData, stageName string, stageStart time.Time) {
	data.StagesCompleted = append(data.StagesCompleted, StageResult{
		Name:       stageName,
		Success:    true,
		DurationMs: time.Since(stageStart).Milliseconds(),
	})
}

// recordStageError записывает ошибку этапа и возвращает форматированную ошибку.
func (h *Git2StoreHandler) recordStageError(data *Git2StoreData, stageName string, stageStart time.Time, code, message string) error {
	data.StagesCompleted = append(data.StagesCompleted, StageResult{
		Name:       stageName,
		Success:    false,
		DurationMs: time.Since(stageStart).Milliseconds(),
		Error:      message,
	})
	return fmt.Errorf("%s: %s", code, message)
}

// createGit создаёт GitOperator через фабрику или production реализацию.
func (h *Git2StoreHandler) createGit(l *slog.Logger, cfg *config.Config) (GitOperator, error) {
	if h.gitFactory != nil {
		return h.gitFactory.CreateGit(l, cfg)
	}
	return createGitProduction(l, cfg)
}

// createConvertConfig создаёт ConvertConfigOperator через фабрику или production реализацию.
func (h *Git2StoreHandler) createConvertConfig() ConvertConfigOperator {
	if h.convertConfigFactory != nil {
		return h.convertConfigFactory.CreateConvertConfig()
	}
	return createConvertConfigProduction()
}

// createBackup создаёт резервную копию хранилища через интерфейс или production реализацию.
func (h *Git2StoreHandler) createBackup(cfg *config.Config, storeRoot string) (string, error) {
	if h.backupCreator != nil {
		return h.backupCreator.CreateBackup(cfg, storeRoot)
	}
	return createBackupProduction(cfg, storeRoot)
}

// createTempDb создаёт временную БД через интерфейс или production реализацию.
func (h *Git2StoreHandler) createTempDb(ctx context.Context, l *slog.Logger, cfg *config.Config) (string, error) {
	if h.tempDbCreator != nil {
		return h.tempDbCreator.CreateTempDb(ctx, l, cfg)
	}
	return createTempDbProduction(ctx, l, cfg)
}

// writeSuccess выводит успешный результат (AC-4, AC-5).
func (h *Git2StoreHandler) writeSuccess(format, traceID string, data *Git2StoreData) error {
	// Текстовый формат (AC-5)
	if format != output.FormatJSON {
		return data.writeText(os.Stdout)
	}

	// JSON формат (AC-4)
	result := &output.Result{
		Status:  output.StatusSuccess,
		Command: constants.ActNRGit2store,
		Data:    data,
		Plan:    h.verbosePlan, // Story 7.3 AC-7: verbose JSON включает план
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
// Для text формата НЕ выводим в stdout — main.go уже логирует ошибку.
func (h *Git2StoreHandler) writeStageError(format, traceID string, start time.Time, data *Git2StoreData, stageErr error) error {
	// Обновляем длительность
	data.DurationMs = time.Since(start).Milliseconds()

	// Текстовый формат — только возвращаем error, main.go выведет через logger
	if format != output.FormatJSON {
		// AC-7: включаем backup path в error message
		if data.BackupPath != "" {
			return fmt.Errorf("%s (backup: %s)", stageErr.Error(), data.BackupPath)
		}
		return stageErr
	}

	// JSON формат — структурированный вывод (AC-11)
	// Извлекаем код и сообщение из ошибки
	errStr := stageErr.Error()
	code := "ERR_GIT2STORE"
	message := errStr

	// Пробуем разобрать формат "CODE: message"
	if colonIdx := len("ERR_GIT2STORE"); colonIdx < len(errStr) && errStr[colonIdx] == '_' {
		// Ищем первое двоеточие после кода
		for i := 0; i < len(errStr); i++ {
			if errStr[i] == ':' {
				code = errStr[:i]
				if i+2 < len(errStr) {
					message = errStr[i+2:] // Пропускаем ": "
				}
				break
			}
		}
	}

	// M-3: Заполняем data.Errors напрямую вместо создания анонимной struct
	data.Errors = []string{message}

	result := &output.Result{
		Status:  output.StatusError,
		Command: constants.ActNRGit2store,
		Data:    data, // M-3: Используем data напрямую, Errors уже заполнен
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

// createBackupProduction создаёт резервную копию хранилища (production реализация).
//
// TODO: [H-4] Реализовать полноценный backup хранилища 1C.
// Текущая реализация создаёт только метаданные (backup_info.txt) с информацией
// для ручного восстановления. Полный backup требует:
// 1. Вызов 1cv8 DESIGNER /ConfigurationRepositoryDumpCfg для экспорта конфигурации
// 2. Сохранение версии хранилища через /ConfigurationRepositoryReport
// 3. Копирование локальных файлов (если хранилище файловое)
// Это требует расширения convert.Config или отдельного BackupService.
func createBackupProduction(cfg *config.Config, storeRoot string) (string, error) {
	// Создаём директорию для backup
	backupDir := filepath.Join(cfg.TmpDir, "backup_"+time.Now().Format("20060102_150405"))
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("не удалось создать директорию backup: %w", err)
	}

	// Записываем информацию о backup (storeRoot для ручного восстановления)
	// ВАЖНО: Это метаданные для ручного восстановления, НЕ полная копия данных!
	backupInfoPath := filepath.Join(backupDir, "backup_info.txt")
	backupInfo := fmt.Sprintf("=== BACKUP METADATA ===\n"+
		"Store Root: %s\n"+
		"Created: %s\n"+
		"Infobase: %s\n"+
		"Owner: %s\n"+
		"Repo: %s\n"+
		"\nВНИМАНИЕ: Это метаданные для ручного восстановления.\n"+
		"Для восстановления используйте 1cv8 DESIGNER или ibcmd.\n",
		storeRoot, time.Now().Format(time.RFC3339),
		cfg.InfobaseName, cfg.Owner, cfg.Repo)
	if err := os.WriteFile(backupInfoPath, []byte(backupInfo), 0644); err != nil {
		return "", fmt.Errorf("не удалось записать информацию о backup: %w", err)
	}

	return backupDir, nil
}
