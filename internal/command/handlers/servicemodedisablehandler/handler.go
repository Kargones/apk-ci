// Package servicemodedisablehandler реализует NR-команду nr-service-mode-disable
// для отключения сервисного режима информационной базы 1C.
package servicemodedisablehandler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/onec/rac"
	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/command/handlers/racutil"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/dryrun"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"
	errhandler "github.com/Kargones/apk-ci/internal/command/handlers/shared"
)

func RegisterCmd() error {
	// Deprecated: alias "service-mode-disable" retained for backward compatibility. Remove in v2.0.0 (Epic 7).
	return command.RegisterWithAlias(&ServiceModeDisableHandler{}, constants.ActServiceModeDisable)
}

// ServiceModeDisableData содержит данные ответа об отключении сервисного режима.
type ServiceModeDisableData struct {
	// Disabled — отключён ли сервисный режим
	Disabled bool `json:"disabled"`
	// AlreadyDisabled — был ли режим уже отключён до вызова
	AlreadyDisabled bool `json:"already_disabled"`
	// StateChanged — было ли произведено реальное изменение состояния
	StateChanged bool `json:"state_changed"`
	// ScheduledJobsUnblocked — разблокированы ли регламентные задания
	ScheduledJobsUnblocked bool `json:"scheduled_jobs_unblocked"`
	// InfobaseName — имя информационной базы
	InfobaseName string `json:"infobase_name"`
}

// writeText выводит результат отключения сервисного режима в человекочитаемом формате.
func (d *ServiceModeDisableData) writeText(w io.Writer) error {
	disabledText := "ОТКЛЮЧЁН"
	if d.AlreadyDisabled {
		disabledText = "ОТКЛЮЧЁН (уже был отключён)"
	}

	if _, err := fmt.Fprintf(w, "Сервисный режим: %s\nИнформационная база: %s\n",
		disabledText, d.InfobaseName); err != nil {
		return err
	}

	if !d.AlreadyDisabled {
		if d.ScheduledJobsUnblocked {
			if _, err := fmt.Fprintln(w, "Регламентные задания: разблокированы"); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintln(w, "Регламентные задания: не разблокированы (отдельная блокировка)"); err != nil {
				return err
			}
		}
	}
	return nil
}

// ServiceModeDisableHandler обрабатывает команду nr-service-mode-disable.
type ServiceModeDisableHandler struct {
	// racClient — опциональный RAC клиент (nil в production, mock в тестах)
	racClient rac.Client
}

// Name возвращает имя команды.
func (h *ServiceModeDisableHandler) Name() string {
	return constants.ActNRServiceModeDisable
}

// Description возвращает описание команды для вывода в help.
func (h *ServiceModeDisableHandler) Description() string {
	return "Отключение сервисного режима информационной базы"
}

// Execute выполняет команду nr-service-mode-disable.
func (h *ServiceModeDisableHandler) Execute(ctx context.Context, cfg *config.Config) error {
	start := time.Now()

	traceID := tracing.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = tracing.GenerateTraceID()
	}

	format := os.Getenv("BR_OUTPUT_FORMAT")

	// Story 7.3 AC-8: plan-only для команд без поддержки плана
	// Review #36: !IsDryRun() — dry-run имеет приоритет над plan-only (AC-11).
	if !dryrun.IsDryRun() && dryrun.IsPlanOnly() {
		return dryrun.WritePlanOnlyUnsupported(os.Stdout, constants.ActNRServiceModeDisable)
	}

	log := slog.Default().With(slog.String("trace_id", traceID), slog.String("command", constants.ActNRServiceModeDisable))

	// Валидация наличия имени информационной базы
	if cfg == nil || cfg.InfobaseName == "" {
		log.Error("Не указано имя информационной базы")
		return h.writeError(format, traceID, start,
			"CONFIG.INFOBASE_MISSING",
			"Не указано имя информационной базы (BR_INFOBASE_NAME)")
	}

	log = log.With(slog.String("infobase", cfg.InfobaseName))
	log.Info("Начало обработки команды отключения сервисного режима")

	// Получение или создание RAC клиента
	racClient := h.racClient
	if racClient == nil {
		var err error
		racClient, err = racutil.NewClient(cfg)
		if err != nil {
			log.Error("Не удалось создать RAC клиент", slog.String("error", err.Error()))
			return h.writeError(format, traceID, start,
				"RAC.CLIENT_CREATE_FAILED",
				fmt.Sprintf("Не удалось создать RAC клиент: %v", err))
		}
	}

	// Получение информации о кластере
	clusterInfo, err := racClient.GetClusterInfo(ctx)
	if err != nil {
		log.Error("Не удалось получить информацию о кластере", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start,
			"RAC.CLUSTER_FAILED",
			fmt.Sprintf("Не удалось получить информацию о кластере: %v", err))
	}

	// Получение информации об информационной базе
	infobaseInfo, err := racClient.GetInfobaseInfo(ctx, clusterInfo.UUID, cfg.InfobaseName)
	if err != nil {
		log.Error("Не удалось получить информацию об информационной базе", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start,
			"RAC.INFOBASE_FAILED",
			fmt.Sprintf("Не удалось получить информацию об информационной базе: %v", err))
	}

	// Проверка идемпотентности: уже отключён?
	status, err := racClient.GetServiceModeStatus(ctx, clusterInfo.UUID, infobaseInfo.UUID)
	if err != nil {
		// Не критично — продолжаем с отключением (fail-open для check)
		log.Warn("Не удалось проверить текущий статус перед отключением", slog.String("error", err.Error()))
	}

	// Логирование текущего состояния перед изменением
	if status != nil {
		log.Info("Текущее состояние перед операцией",
			slog.Bool("enabled", status.Enabled),
			slog.Bool("scheduled_jobs_blocked", status.ScheduledJobsBlocked),
			slog.Int("active_sessions", status.ActiveSessions))
	}

	if status != nil && !status.Enabled {
		log.Info("Сервисный режим уже отключён")
		data := &ServiceModeDisableData{
			Disabled:               true,
			AlreadyDisabled:        true,
			StateChanged:           false,
			ScheduledJobsUnblocked: !status.ScheduledJobsBlocked,
			InfobaseName:           cfg.InfobaseName,
		}
		return h.outputResult(format, data, traceID, start)
	}

	// Отключение сервисного режима
	log.Info("Вызов DisableServiceMode")
	err = racClient.DisableServiceMode(ctx, clusterInfo.UUID, infobaseInfo.UUID)
	if err != nil {
		log.Error("Не удалось отключить сервисный режим", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start,
			"RAC.DISABLE_FAILED",
			fmt.Sprintf("Не удалось отключить сервисный режим: %v", err))
	}

	// Верификация отключения
	err = racClient.VerifyServiceMode(ctx, clusterInfo.UUID, infobaseInfo.UUID, false)
	if err != nil {
		log.Error("Верификация отключения сервисного режима не прошла", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start,
			"RAC.VERIFY_FAILED",
			fmt.Sprintf("Верификация отключения сервисного режима не прошла: %v", err))
	}

	// Определение состояния регламентных заданий после отключения
	scheduledJobsUnblocked := true
	postStatus, postErr := racClient.GetServiceModeStatus(ctx, clusterInfo.UUID, infobaseInfo.UUID)
	if postErr == nil && postStatus != nil {
		scheduledJobsUnblocked = !postStatus.ScheduledJobsBlocked
	} else if postErr != nil {
		log.Warn("Не удалось проверить статус регламентных заданий после отключения",
			slog.String("error", postErr.Error()))
	}

	log.Info("Сервисный режим успешно отключён",
		slog.Bool("scheduled_jobs_unblocked", scheduledJobsUnblocked))

	data := &ServiceModeDisableData{
		Disabled:               true,
		AlreadyDisabled:        false,
		StateChanged:           true,
		ScheduledJobsUnblocked: scheduledJobsUnblocked,
		InfobaseName:           cfg.InfobaseName,
	}

	return h.outputResult(format, data, traceID, start)
}

// outputResult форматирует и выводит результат.
func (h *ServiceModeDisableHandler) outputResult(format string, data *ServiceModeDisableData, traceID string, start time.Time) error {
	// Текстовый формат
	if format != output.FormatJSON {
		return data.writeText(os.Stdout)
	}

	// JSON формат
	result := &output.Result{
		Status:  output.StatusSuccess,
		Command: constants.ActNRServiceModeDisable,
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
func (h *ServiceModeDisableHandler) writeError(format, traceID string, start time.Time, code, message string) error {
	// Текстовый формат — человекочитаемый вывод ошибки
	if format != output.FormatJSON {
		return errhandler.HandleError(message, code)
	}

	// JSON формат — структурированный вывод
	result := &output.Result{
		Status:  output.StatusError,
		Command: constants.ActNRServiceModeDisable,
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

