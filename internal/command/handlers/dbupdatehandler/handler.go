// Package dbupdatehandler реализует NR-команду nr-dbupdate
// для обновления структуры базы данных по конфигурации 1C.
package dbupdatehandler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/onec"
	"github.com/Kargones/apk-ci/internal/adapter/onec/rac"
	"github.com/Kargones/apk-ci/internal/command"
	errhandler "github.com/Kargones/apk-ci/internal/command/handlers/shared"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/alerting"
	"github.com/Kargones/apk-ci/internal/pkg/dryrun"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"
)

// Коды ошибок для команды nr-dbupdate.
const (
	ErrDbUpdateValidation       = "DBUPDATE.VALIDATION_FAILED"
	ErrDbUpdateConfig           = "DBUPDATE.CONFIG_ERROR"
	ErrDbUpdateFailed           = "DBUPDATE.UPDATE_FAILED"
	ErrDbUpdateSecondPassFailed = "DBUPDATE.SECOND_PASS_FAILED"
	ErrDbUpdateTimeout          = "DBUPDATE.TIMEOUT"
	ErrDbUpdateAutoDeps         = "DBUPDATE.AUTO_DEPS_FAILED"

	// defaultTimeout — таймаут по умолчанию для обновления БД.
	defaultTimeout = 30 * time.Minute

	// maxMessages — максимальное количество сообщений в результате (C1 fix).
	maxMessages = 100

	// cleanupTimeout — таймаут для cleanup операций (M1 fix).
	// H4 note: Этот таймаут покрывает 3 RAC вызова в disableServiceModeIfNeeded:
	// GetClusterInfo + GetInfobaseInfo + DisableServiceMode.
	// При необходимости увеличить, учитывая сетевые задержки.
	cleanupTimeout = 30 * time.Second

	// racOperationTimeout — таймаут для операций RAC (H3 fix).
	racOperationTimeout = 60 * time.Second
)

func RegisterCmd() error {
	// Deprecated: alias "dbupdate" retained for backward compatibility. Remove in v2.0.0 (Epic 7).
	return command.RegisterWithAlias(&DbUpdateHandler{}, constants.ActDbupdate)
}

// DbUpdateData содержит данные ответа о результате обновления.
type DbUpdateData struct {
	// InfobaseName — имя информационной базы
	InfobaseName string `json:"infobase_name"`
	// Extension — имя расширения (если обновлялось)
	Extension string `json:"extension,omitempty"`
	// Success — успешно ли обновление
	Success bool `json:"success"`
	// Messages — сообщения от платформы
	Messages []string `json:"messages,omitempty"`
	// DurationMs — время выполнения в миллисекундах
	DurationMs int64 `json:"duration_ms"`
	// AutoDeps — был ли использован режим автоматического управления зависимостями
	AutoDeps bool `json:"auto_deps"`
}

// writeText выводит результат обновления в человекочитаемом формате.
func (d *DbUpdateData) writeText(w io.Writer) error {
	status := "✅ Обновление завершено успешно"
	if !d.Success {
		status = "❌ Обновление завершено с ошибками"
	}

	_, err := fmt.Fprintf(w, "%s\n", status)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "База данных: %s\n", d.InfobaseName)
	if err != nil {
		return err
	}

	if d.Extension != "" {
		_, err = fmt.Fprintf(w, "Расширение: %s\n", d.Extension)
		if err != nil {
			return err
		}
	}

	duration := time.Duration(d.DurationMs) * time.Millisecond
	_, err = fmt.Fprintf(w, "Время выполнения: %v\n", duration.Round(time.Millisecond))
	if err != nil {
		return err
	}

	if d.AutoDeps {
		_, err = fmt.Fprintf(w, "Auto-deps: включён\n")
		if err != nil {
			return err
		}
	}

	if len(d.Messages) > 0 {
		_, err = fmt.Fprintf(w, "\nСообщения:\n")
		if err != nil {
			return err
		}
		for _, msg := range d.Messages {
			_, err = fmt.Fprintf(w, "  - %s\n", msg)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// DbUpdateHandler обрабатывает команду nr-dbupdate.
type DbUpdateHandler struct {
	// oneCClient — клиент 1C для тестирования; если nil — создаётся реальный клиент
	oneCClient onec.DatabaseUpdater
	// racClient — клиент RAC для тестирования; если nil — создаётся реальный клиент
	racClient rac.Client
	// verbosePlan — план операций для verbose режима (Story 7.3), добавляется в JSON результат
	verbosePlan *output.DryRunPlan
}

// Name возвращает имя команды.
func (h *DbUpdateHandler) Name() string {
	return constants.ActNRDbupdate
}

// Description возвращает описание команды для вывода в help.
// AC-10: включает описание BR_DRY_RUN для документации.
func (h *DbUpdateHandler) Description() string {
	return "Обновить структуру базы данных по конфигурации. " +
		"Переменная BR_DRY_RUN=true выводит план операций без выполнения"
}

// dbUpdateContext holds shared state for Execute.
type dbUpdateContext struct {
	start         time.Time
	traceID       string
	format        string
	log           *slog.Logger
	connectString string
	extension     string
	timeout       time.Duration
}

// validateDbUpdate validates config and prepares the execution context.
func (h *DbUpdateHandler) validateDbUpdate(ctx context.Context, cfg *config.Config) (*dbUpdateContext, *config.DatabaseInfo, error) {
	ec := &dbUpdateContext{start: time.Now()}
	ec.traceID = tracing.TraceIDFromContext(ctx)
	if ec.traceID == "" {
		ec.traceID = tracing.GenerateTraceID()
	}
	ec.format = os.Getenv("BR_OUTPUT_FORMAT")
	ec.log = slog.Default().With(slog.String("trace_id", ec.traceID), slog.String("command", constants.ActNRDbupdate))

	if err := ctx.Err(); err != nil {
		ec.log.Warn("Context отменён до начала выполнения", slog.String("error", err.Error()))
		return nil, nil, h.writeError(ctx, cfg, ec.format, ec.traceID, ec.start, ErrDbUpdateFailed, "операция отменена: "+err.Error())
	}

	if cfg == nil || cfg.InfobaseName == "" {
		ec.log.Error("Не указано имя информационной базы")
		return nil, nil, h.writeError(ctx, cfg, ec.format, ec.traceID, ec.start, ErrDbUpdateValidation, "BR_INFOBASE_NAME обязателен")
	}
	ec.log = ec.log.With(slog.String("infobase", cfg.InfobaseName))
	ec.log.Info("Запуск обновления структуры базы данных")

	dbInfo := cfg.GetDatabaseInfo(cfg.InfobaseName)
	if dbInfo == nil {
		ec.log.Error("Информационная база не найдена в конфигурации")
		return nil, nil, h.writeError(ctx, cfg, ec.format, ec.traceID, ec.start, ErrDbUpdateConfig,
			fmt.Sprintf("информационная база '%s' не найдена в конфигурации", cfg.InfobaseName))
	}

	if cfg.AppConfig == nil || cfg.AppConfig.Paths.Bin1cv8 == "" {
		ec.log.Error("Путь к 1cv8 не указан в конфигурации")
		return nil, nil, h.writeError(ctx, cfg, ec.format, ec.traceID, ec.start, ErrDbUpdateConfig,
			"путь к 1cv8 не указан в конфигурации (app.yaml:paths.bin1cv8)")
	}

	if cfg.WorkDir == "" { ec.log.Warn("WorkDir не указан, используется системная временная директория") }
	if cfg.TmpDir == "" { ec.log.Warn("TmpDir не указан, используется системная временная директория") }

	ec.connectString = h.buildConnectString(dbInfo, cfg)
	ec.extension = os.Getenv("BR_EXTENSION")
	ec.timeout = h.getTimeout()

	return ec, dbInfo, nil
}

// handleDbPreviewModes handles dry-run, plan-only, and verbose modes.
func (h *DbUpdateHandler) handleDbPreviewModes(ec *dbUpdateContext, cfg *config.Config, dbInfo *config.DatabaseInfo) (bool, error) {
	if dryrun.IsDryRun() {
		ec.log.Info("Dry-run режим: построение плана")
		return true, h.executeDryRun(cfg, dbInfo, ec.connectString, ec.extension, ec.timeout, ec.format, ec.traceID, ec.start)
	}
	if dryrun.IsPlanOnly() {
		ec.log.Info("Plan-only режим: отображение плана операций")
		plan := h.buildPlan(cfg, dbInfo, ec.connectString, ec.extension, ec.timeout)
		return true, output.WritePlanOnlyResult(os.Stdout, ec.format, constants.ActNRDbupdate, ec.traceID, constants.APIVersion, ec.start, plan)
	}
	if dryrun.IsVerbose() {
		ec.log.Info("Verbose режим: отображение плана перед выполнением")
		plan := h.buildPlan(cfg, dbInfo, ec.connectString, ec.extension, ec.timeout)
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

// setupAutoDeps configures auto-deps mode and returns cleanup state.
func (h *DbUpdateHandler) setupAutoDeps(ctx context.Context, cfg *config.Config, dbInfo *config.DatabaseInfo, log *slog.Logger) (bool, bool, rac.Client) {
	autoDeps := os.Getenv("BR_AUTO_DEPS") == "true"
	if !autoDeps {
		return false, false, nil
	}
	log.Info("Auto-deps режим включён")
	racClient := h.getOrCreateRacClient(cfg, dbInfo, log)
	if racClient == nil {
		log.Warn("RAC клиент недоступен, продолжаем без auto-deps")
		return false, false, nil
	}
	racCtx, racCancel := context.WithTimeout(ctx, racOperationTimeout)
	alreadyEnabled, smErr := h.enableServiceModeIfNeeded(racCtx, cfg, racClient, log)
	racCancel()
	if smErr != nil {
		log.Warn("Не удалось включить сервисный режим", slog.String("error", smErr.Error()))
		return false, false, nil
	}
	return true, !alreadyEnabled, racClient
}

// runExtensionSecondPass runs the second update pass for extensions.
func (h *DbUpdateHandler) runExtensionSecondPass(ctx context.Context, ec *dbUpdateContext, cfg *config.Config, client onec.DatabaseUpdater, opts onec.UpdateOptions, result *onec.UpdateResult, prog interface{ Update(int64, string) }) error {
	ec.log.Info("Первый проход обновления расширения завершён",
		slog.String("extension", ec.extension),
		slog.Int64("first_pass_duration_ms", result.DurationMs),
		slog.Bool("first_pass_success", result.Success))
	if !result.Success {
		ec.log.Warn("Первый проход расширения не успешен, но продолжаем второй проход")
	}
	if ctx.Err() != nil {
		ec.log.Error("Контекст отменён перед вторым проходом", slog.String("error", ctx.Err().Error()))
		return h.writeError(ctx, cfg, ec.format, ec.traceID, ec.start, ErrDbUpdateFailed,
			fmt.Sprintf("операция отменена перед вторым проходом: %s", ctx.Err().Error()))
	}
	prog.Update(0, "Второй проход обновления расширения...")
	ec.log.Info("Второй проход обновления для расширения", slog.String("extension", ec.extension))
	result2, err := client.UpdateDBCfg(ctx, opts)
	if err != nil {
		ec.log.Error("Ошибка второго прохода обновления", slog.String("error", err.Error()))
		return h.writeError(ctx, cfg, ec.format, ec.traceID, ec.start, ErrDbUpdateSecondPassFailed,
			fmt.Sprintf("ошибка второго прохода обновления расширения: %s", err.Error()))
	}
	result.Messages = append(result.Messages, result2.Messages...)
	if len(result.Messages) > maxMessages {
		result.Messages = result.Messages[:maxMessages]
		ec.log.Warn("Количество сообщений превысило лимит, обрезано", slog.Int("max", maxMessages))
	}
	result.DurationMs += result2.DurationMs
	return nil
}

// Execute выполняет команду nr-dbupdate.
func (h *DbUpdateHandler) Execute(ctx context.Context, cfg *config.Config) error {
	ec, dbInfo, err := h.validateDbUpdate(ctx, cfg)
	if err != nil {
		return err
	}

	if handled, pErr := h.handleDbPreviewModes(ec, cfg, dbInfo); handled {
		return pErr
	}

	client := h.getOrCreateOneCClient(cfg)

	autoDeps, weEnabledServiceMode, racClient := h.setupAutoDeps(ctx, cfg, dbInfo, ec.log)

	defer func() {
		if autoDeps && weEnabledServiceMode && racClient != nil {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), cleanupTimeout)
			defer cleanupCancel()
			h.disableServiceModeIfNeeded(cleanupCtx, cfg, racClient, ec.log)
		}
	}()

	prog := h.createProgress()
	prog.Start("Обновление структуры базы данных...")
	defer prog.Finish()

	opts := onec.UpdateOptions{
		ConnectString: ec.connectString, Extension: ec.extension,
		Timeout: ec.timeout, Bin1cv8: cfg.AppConfig.Paths.Bin1cv8,
	}

	result, err := client.UpdateDBCfg(ctx, opts)
	if err != nil {
		ec.log.Error("Ошибка обновления структуры БД", slog.String("error", err.Error()))
		return h.writeError(ctx, cfg, ec.format, ec.traceID, ec.start, ErrDbUpdateFailed, err.Error())
	}
	if len(result.Messages) > maxMessages {
		result.Messages = result.Messages[:maxMessages]
		ec.log.Warn("Количество сообщений превысило лимит после первого прохода, обрезано", slog.Int("max", maxMessages))
	}

	if ec.extension != "" {
		if err := h.runExtensionSecondPass(ctx, ec, cfg, client, opts, result, prog); err != nil {
			return err
		}
	}

	duration := time.Since(ec.start)
	if result.Success {
		ec.log.Info("Обновление завершено успешно", slog.Duration("duration", duration))
	} else {
		ec.log.Warn("Обновление завершено с предупреждениями",
			slog.Duration("duration", duration), slog.Int("messages_count", len(result.Messages)))
	}

	data := &DbUpdateData{
		InfobaseName: cfg.InfobaseName, Extension: ec.extension,
		Success: result.Success, Messages: result.Messages,
		DurationMs: duration.Milliseconds(), AutoDeps: autoDeps,
	}

	if ec.format != output.FormatJSON {
		return data.writeText(os.Stdout)
	}

	resultOutput := &output.Result{
		Status: output.StatusSuccess, Command: constants.ActNRDbupdate,
		Data: data, Plan: h.verbosePlan,
		Metadata: &output.Metadata{
			DurationMs: duration.Milliseconds(), TraceID: ec.traceID, APIVersion: constants.APIVersion,
		},
	}
	writer := output.NewWriter(ec.format)
	return writer.Write(os.Stdout, resultOutput)
}

// writeError выводит структурированную ошибку и возвращает error.
func (h *DbUpdateHandler) writeError(ctx context.Context, cfg *config.Config, format, traceID string, start time.Time, code, message string) error {
	// Отправка алерта с детальным кодом ошибки (#59)
	if cfg != nil && cfg.Alerter != nil {
		_ = cfg.Alerter.Send(ctx, alerting.Alert{
			ErrorCode: code,
			Message:   message,
			Command:   constants.ActNRDbupdate,
			Infobase:  cfg.InfobaseName,
			TraceID:   traceID,
			Timestamp: time.Now(),
			Severity:  alerting.SeverityCritical,
		})
	}
	// Текстовый формат — человекочитаемый вывод ошибки
	if format != output.FormatJSON {
		return errhandler.HandleError(message, code)
	}

	// JSON формат — структурированный вывод
	result := &output.Result{
		Status:  output.StatusError,
		Command: constants.ActNRDbupdate,
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
