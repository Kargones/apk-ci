// Package git2storehandler реализует NR-команду nr-git2store
// для синхронизации EDT из Git в хранилище 1C.
package git2storehandler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
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

func RegisterCmd() {
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
// Effectively constant. Cannot be const: Go does not support const slices.
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
// TODO: Рефакторинг: вынести фабрики (gitFactory, convertConfigFactory,
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
