// Package createtempdbhandler реализует NR-команду nr-create-temp-db
// для создания временной локальной базы данных 1C с расширениями.
package createtempdbhandler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/onec"
	"github.com/Kargones/apk-ci/internal/command"
	errhandler "github.com/Kargones/apk-ci/internal/command/handlers/shared"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/dryrun"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"
)

// Коды ошибок для команды nr-create-temp-db.
const (
	ErrCreateTempDbValidation = "CREATETEMPDB.VALIDATION_FAILED"
	ErrCreateTempDbFailed     = "CREATETEMPDB.CREATE_FAILED"
	ErrExtensionAddFailed     = "CREATETEMPDB.EXTENSION_FAILED"
	ErrContextCancelled       = "CREATETEMPDB.CONTEXT_CANCELLED"

	// defaultTimeout — таймаут по умолчанию для создания БД.
	defaultTimeout = 30 * time.Minute

	// maxPathLength — максимальная длина пути к БД (255 символов для большинства FS).
	maxPathLength = 255

	// maxExtensions — максимальное количество расширений для предотвращения DoS.
	// M-5 fix: ограничиваем количество для защиты от злоупотреблений.
	maxExtensions = 50
)

func RegisterCmd() error {
	// Deprecated: alias "create-temp-db" retained for backward compatibility. Remove in v2.0.0 (Epic 7).
	return command.RegisterWithAlias(&CreateTempDbHandler{}, constants.ActCreateTempDb)
}

// CreateTempDbData содержит результат создания временной БД для JSON вывода.
type CreateTempDbData struct {
	// ConnectString — строка подключения "/F <path>"
	ConnectString string `json:"connect_string"`
	// DbPath — полный путь к созданной БД
	DbPath string `json:"db_path"`
	// Extensions — список добавленных расширений
	Extensions []string `json:"extensions,omitempty"`
	// TTLHours — TTL в часах (0 = без TTL)
	TTLHours int `json:"ttl_hours,omitempty"`
	// CreatedAt — время создания в формате ISO 8601
	CreatedAt string `json:"created_at"`
	// DurationMs — время выполнения в миллисекундах
	DurationMs int64 `json:"duration_ms"`
}

// writeText выводит результат создания в человекочитаемом формате.
func (d *CreateTempDbData) writeText(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "Временная база данных создана\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Путь: %s\n", d.DbPath); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Строка подключения: %s\n", d.ConnectString); err != nil {
		return err
	}

	if len(d.Extensions) > 0 {
		if _, err := fmt.Fprintf(w, "Расширения: %s\n", strings.Join(d.Extensions, ", ")); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(w, "Расширения: нет\n"); err != nil {
			return err
		}
	}

	if d.TTLHours > 0 {
		if _, err := fmt.Fprintf(w, "TTL: %d часов\n", d.TTLHours); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "Время создания: %s\n", d.CreatedAt); err != nil {
		return err
	}

	// Корректный вывод для очень быстрых операций
	var durationStr string
	if d.DurationMs == 0 {
		durationStr = "< 1ms"
	} else {
		duration := time.Duration(d.DurationMs) * time.Millisecond
		durationStr = duration.Round(time.Millisecond).String()
	}
	if _, err := fmt.Fprintf(w, "Время выполнения: %s\n", durationStr); err != nil {
		return err
	}

	return nil
}

// TTLMetadata структура для файла .ttl.
type TTLMetadata struct {
	CreatedAt time.Time `json:"created_at"`
	TTLHours  int       `json:"ttl_hours"`
	ExpiresAt time.Time `json:"expires_at"`
}

// CreateTempDbHandler обрабатывает команду nr-create-temp-db.
type CreateTempDbHandler struct {
	// dbCreator — клиент для создания БД; если nil — создаётся реальный клиент
	dbCreator onec.TempDatabaseCreator
	// verbosePlan — план операций для verbose режима (Story 7.3), добавляется в JSON результат
	verbosePlan *output.DryRunPlan
}

// Name возвращает имя команды.
func (h *CreateTempDbHandler) Name() string {
	return constants.ActNRCreateTempDb
}

// Description возвращает описание команды для вывода в help.
// AC-10: включает описание BR_DRY_RUN для документации.
func (h *CreateTempDbHandler) Description() string {
	return "Создать временную локальную базу данных с расширениями. " +
		"Переменная BR_DRY_RUN=true выводит план операций без выполнения"
}

// tempDbExecContext holds shared state for Execute.
type tempDbExecContext struct {
	start      time.Time
	traceID    string
	format     string
	log        *slog.Logger
	dbPath     string
	extensions []string
	timeout    time.Duration
	ttlHours   int
}

// validateTempDbConfig validates config and ibcmd binary, returning execution context.
func (h *CreateTempDbHandler) validateTempDbConfig(ctx context.Context, cfg *config.Config) (*tempDbExecContext, error) {
	ec := &tempDbExecContext{start: time.Now()}
	ec.traceID = tracing.TraceIDFromContext(ctx)
	if ec.traceID == "" {
		ec.traceID = tracing.GenerateTraceID()
	}
	ec.format = os.Getenv("BR_OUTPUT_FORMAT")
	ec.log = slog.Default().With(slog.String("trace_id", ec.traceID), slog.String("command", constants.ActNRCreateTempDb))

	if err := ctx.Err(); err != nil {
		ec.log.Warn("Context отменён до начала выполнения", slog.String("error", err.Error()))
		return nil, h.writeError(ec.format, ec.traceID, ec.start, ErrContextCancelled, "операция отменена: "+err.Error())
	}

	if cfg == nil {
		ec.log.Error("Конфигурация не указана")
		return nil, h.writeError(ec.format, ec.traceID, ec.start, ErrCreateTempDbValidation, "конфигурация приложения не указана")
	}
	if cfg.AppConfig == nil || cfg.AppConfig.Paths.BinIbcmd == "" {
		ec.log.Error("Путь к ibcmd не указан в конфигурации")
		return nil, h.writeError(ec.format, ec.traceID, ec.start, ErrCreateTempDbValidation, "путь к ibcmd не указан в конфигурации (app.yaml:paths.binIbcmd)")
	}

	if err := h.validateIbcmdBinary(ec, cfg.AppConfig.Paths.BinIbcmd); err != nil {
		return nil, err
	}

	var pathErr error
	ec.dbPath, pathErr = h.generateDbPath(cfg)
	if pathErr != nil {
		ec.log.Error("Небезопасный путь для временной БД", slog.String("error", pathErr.Error()))
		return nil, h.writeError(ec.format, ec.traceID, ec.start, ErrCreateTempDbValidation, pathErr.Error())
	}
	ec.log.Info("Генерация пути к временной БД", slog.String("path", ec.dbPath))

	parentDir := filepath.Dir(ec.dbPath)
	if err := os.MkdirAll(parentDir, constants.DirPermExec); err != nil {
		ec.log.Error("Не удалось создать директорию для БД", slog.String("path", parentDir), slog.String("error", err.Error()))
		return nil, h.writeError(ec.format, ec.traceID, ec.start, ErrCreateTempDbValidation,
			fmt.Sprintf("не удалось создать директорию %s: %s", parentDir, err.Error()))
	}

	ec.extensions = h.parseExtensions(cfg)
	ec.log.Info("Расширения для добавления", slog.Any("extensions", ec.extensions))
	ec.timeout = h.getTimeout()
	ec.ttlHours = h.getTTLHours()

	return ec, nil
}

// validateIbcmdBinary checks that ibcmd exists and is executable.
func (h *CreateTempDbHandler) validateIbcmdBinary(ec *tempDbExecContext, binPath string) error {
	info, err := os.Stat(binPath)
	if os.IsNotExist(err) {
		ec.log.Error("Файл ibcmd не найден", slog.String("path", binPath))
		return h.writeError(ec.format, ec.traceID, ec.start, ErrCreateTempDbValidation,
			fmt.Sprintf("файл ibcmd не найден: %s", binPath))
	}
	if err != nil {
		ec.log.Error("Ошибка проверки файла ibcmd", slog.String("path", binPath), slog.String("error", err.Error()))
		return h.writeError(ec.format, ec.traceID, ec.start, ErrCreateTempDbValidation,
			fmt.Sprintf("ошибка проверки файла ibcmd: %s", err.Error()))
	}
	if info.Mode()&0111 == 0 {
		ec.log.Error("Файл ibcmd не является исполняемым", slog.String("path", binPath))
		return h.writeError(ec.format, ec.traceID, ec.start, ErrCreateTempDbValidation,
			fmt.Sprintf("файл ibcmd не является исполняемым: %s (mode: %s)", binPath, info.Mode().String()))
	}
	return nil
}

// handleTempDbPreviewModes handles dry-run, plan-only, verbose.
func (h *CreateTempDbHandler) handleTempDbPreviewModes(ec *tempDbExecContext, cfg *config.Config) (bool, error) {
	if dryrun.IsDryRun() {
		ec.log.Info("Dry-run режим: построение плана")
		return true, h.executeDryRun(cfg, ec.dbPath, ec.extensions, ec.timeout, ec.ttlHours, ec.format, ec.traceID, ec.start)
	}
	if dryrun.IsPlanOnly() {
		ec.log.Info("Plan-only режим: отображение плана операций")
		plan := h.buildPlan(cfg, ec.dbPath, ec.extensions, ec.timeout, ec.ttlHours)
		return true, output.WritePlanOnlyResult(os.Stdout, ec.format, constants.ActNRCreateTempDb, ec.traceID, constants.APIVersion, ec.start, plan)
	}
	if dryrun.IsVerbose() {
		ec.log.Info("Verbose режим: отображение плана перед выполнением")
		plan := h.buildPlan(cfg, ec.dbPath, ec.extensions, ec.timeout, ec.ttlHours)
		if ec.format != output.FormatJSON {
			if err := plan.WritePlanText(os.Stdout); err != nil {
				ec.log.Warn("Не удалось вывести план операций", slog.String("error", err.Error()))
			}
			fmt.Fprintln(os.Stdout)
		}
		h.verbosePlan = plan
	}
	return false, nil
}

// classifyCreateError maps a creation error to the appropriate error code.
func classifyCreateError(err error) string {
	switch {
	case errors.Is(err, onec.ErrExtensionAdd):
		return ErrExtensionAddFailed
	case errors.Is(err, onec.ErrContextCancelled):
		return ErrContextCancelled
	case errors.Is(err, onec.ErrInfobaseCreate):
		return ErrCreateTempDbFailed
	case strings.Contains(err.Error(), "расширения") || strings.Contains(err.Error(), "extension"):
		return ErrExtensionAddFailed
	default:
		return ErrCreateTempDbFailed
	}
}

// Execute выполняет команду nr-create-temp-db.
func (h *CreateTempDbHandler) Execute(ctx context.Context, cfg *config.Config) error {
	ec, err := h.validateTempDbConfig(ctx, cfg)
	if err != nil {
		return err
	}

	if handled, pErr := h.handleTempDbPreviewModes(ec, cfg); handled {
		return pErr
	}

	client := h.getOrCreateClient(cfg)

	prog := h.createProgress()
	prog.Start("Создание временной базы данных...")
	defer prog.Finish()

	if err := ctx.Err(); err != nil {
		ec.log.Warn("Context отменён перед созданием БД", slog.String("error", err.Error()))
		return h.writeError(ec.format, ec.traceID, ec.start, ErrContextCancelled, "операция отменена: "+err.Error())
	}

	opts := onec.CreateTempDBOptions{
		DbPath: ec.dbPath, Extensions: ec.extensions,
		Timeout: ec.timeout, BinIbcmd: cfg.AppConfig.Paths.BinIbcmd,
	}

	result, err := client.CreateTempDB(ctx, opts)
	if err != nil {
		ec.log.Error("Ошибка создания временной БД", slog.String("error", err.Error()))
		return h.writeError(ec.format, ec.traceID, ec.start, classifyCreateError(err), err.Error())
	}

	if ec.ttlHours > 0 {
		if ttlErr := h.writeTTLMetadata(ec.dbPath, ec.ttlHours, result.CreatedAt); ttlErr != nil {
			ec.log.Warn("Не удалось записать TTL metadata", slog.String("error", ttlErr.Error()))
		}
	}

	duration := time.Since(ec.start)
	ec.log.Info("Временная база данных создана", slog.String("path", result.DbPath), slog.Duration("duration", duration))

	data := &CreateTempDbData{
		ConnectString: result.ConnectString, DbPath: result.DbPath,
		Extensions: result.Extensions, TTLHours: ec.ttlHours,
		CreatedAt: result.CreatedAt.Format(time.RFC3339), DurationMs: duration.Milliseconds(),
	}

	if ec.format != output.FormatJSON {
		return data.writeText(os.Stdout)
	}

	resultOutput := &output.Result{
		Status: output.StatusSuccess, Command: constants.ActNRCreateTempDb,
		Data: data, Plan: h.verbosePlan,
		Metadata: &output.Metadata{DurationMs: duration.Milliseconds(), TraceID: ec.traceID, APIVersion: constants.APIVersion},
	}
	writer := output.NewWriter(ec.format)
	return writer.Write(os.Stdout, resultOutput)
}

// writeError выводит структурированную ошибку и возвращает error.
func (h *CreateTempDbHandler) writeError(format, traceID string, start time.Time, code, message string) error {
	// Текстовый формат — человекочитаемый вывод ошибки
	if format != output.FormatJSON {
		return errhandler.HandleError(message, code)
	}

	// JSON формат — структурированный вывод
	result := &output.Result{
		Status:  output.StatusError,
		Command: constants.ActNRCreateTempDb,
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
