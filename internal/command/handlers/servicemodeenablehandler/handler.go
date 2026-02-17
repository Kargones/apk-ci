// Package servicemodeenablehandler реализует NR-команду nr-service-mode-enable
// для включения сервисного режима информационной базы 1C.
package servicemodeenablehandler

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

func RegisterCmd() {
	command.RegisterWithAlias(&ServiceModeEnableHandler{}, constants.ActServiceModeEnable)
}

// ServiceModeEnableData содержит данные ответа о включении сервисного режима.
type ServiceModeEnableData struct {
	// Enabled — включён ли сервисный режим
	Enabled bool `json:"enabled"`
	// AlreadyEnabled — был ли режим уже включён до вызова
	AlreadyEnabled bool `json:"already_enabled"`
	// StateChanged — было ли произведено реальное изменение состояния
	StateChanged bool `json:"state_changed"`
	// Message — сообщение блокировки
	Message string `json:"message"`
	// PermissionCode — код разрешения
	PermissionCode string `json:"permission_code"`
	// ScheduledJobsBlocked — заблокированы ли регламентные задания
	ScheduledJobsBlocked bool `json:"scheduled_jobs_blocked"`
	// TerminatedSessionsCount — количество завершённых сессий
	TerminatedSessionsCount int `json:"terminated_sessions_count"`
	// InfobaseName — имя информационной базы
	InfobaseName string `json:"infobase_name"`
}

// writeText выводит результат включения сервисного режима в человекочитаемом формате.
func (d *ServiceModeEnableData) writeText(w io.Writer) error {
	enabledText := "ВКЛЮЧЁН"
	if d.AlreadyEnabled {
		enabledText = "ВКЛЮЧЁН (уже был включён)"
	}

	_, err := fmt.Fprintf(w, "Сервисный режим: %s\nИнформационная база: %s\nСообщение: \"%s\"\nКод разрешения: %s\n",
		enabledText, d.InfobaseName, d.Message, d.PermissionCode)
	if err != nil {
		return err
	}

	if !d.AlreadyEnabled {
		if _, err = fmt.Fprintln(w, "Регламентные задания: заблокированы"); err != nil {
			return err
		}
		if d.TerminatedSessionsCount > 0 {
			_, err = fmt.Fprintf(w, "Завершено сессий: %d\n", d.TerminatedSessionsCount)
		}
	}
	return err
}

// ServiceModeEnableHandler обрабатывает команду nr-service-mode-enable.
type ServiceModeEnableHandler struct {
	// racClient — опциональный RAC клиент (nil в production, mock в тестах)
	racClient rac.Client
}

// Name возвращает имя команды.
func (h *ServiceModeEnableHandler) Name() string {
	return constants.ActNRServiceModeEnable
}

// Description возвращает описание команды для вывода в help.
func (h *ServiceModeEnableHandler) Description() string {
	return "Включение сервисного режима информационной базы"
}

// Execute выполняет команду nr-service-mode-enable.
func (h *ServiceModeEnableHandler) Execute(ctx context.Context, cfg *config.Config) error {
	start := time.Now()

	traceID := tracing.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = tracing.GenerateTraceID()
	}

	format := os.Getenv("BR_OUTPUT_FORMAT")

	// Story 7.3 AC-8: plan-only для команд без поддержки плана
	// Review #36: !IsDryRun() — dry-run имеет приоритет над plan-only (AC-11).
	if !dryrun.IsDryRun() && dryrun.IsPlanOnly() {
		return dryrun.WritePlanOnlyUnsupported(os.Stdout, constants.ActNRServiceModeEnable)
	}

	log := slog.Default().With(slog.String("trace_id", traceID), slog.String("command", constants.ActNRServiceModeEnable))

	// Валидация наличия имени информационной базы
	if cfg == nil || cfg.InfobaseName == "" {
		log.Error("Не указано имя информационной базы")
		return h.writeError(format, traceID, start,
			"CONFIG.INFOBASE_MISSING",
			"Не указано имя информационной базы (BR_INFOBASE_NAME)")
	}

	log = log.With(slog.String("infobase", cfg.InfobaseName))
	log.Info("Начало обработки команды включения сервисного режима")

	// Чтение параметров из переменных окружения.
	// Внимание: env-переменные BR_SERVICE_MODE_MESSAGE и BR_SERVICE_MODE_PERMISSION_CODE
	// используются только для информационного вывода в response. RAC client (EnableServiceMode)
	// использует захардкоженные значения constants.DefaultServiceModeMessage и "ServiceMode".
	message := os.Getenv("BR_SERVICE_MODE_MESSAGE")
	if message == "" {
		message = constants.DefaultServiceModeMessage
	}

	permissionCode := os.Getenv("BR_SERVICE_MODE_PERMISSION_CODE")
	if permissionCode == "" {
		permissionCode = "ServiceMode"
	}

	terminateSessions := cfg.TerminateSessions

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

	// Проверка идемпотентности: уже включён?
	status, err := racClient.GetServiceModeStatus(ctx, clusterInfo.UUID, infobaseInfo.UUID)
	if err != nil {
		// Не критично — продолжаем с включением (fail-open для check)
		log.Warn("Не удалось проверить текущий статус перед включением", slog.String("error", err.Error()))
	}

	// Логирование текущего состояния перед изменением
	if status != nil {
		log.Info("Текущее состояние перед операцией",
			slog.Bool("enabled", status.Enabled),
			slog.Bool("scheduled_jobs_blocked", status.ScheduledJobsBlocked),
			slog.Int("active_sessions", status.ActiveSessions))
	}

	if status != nil && status.Enabled {
		log.Info("Сервисный режим уже включён", slog.String("infobase", cfg.InfobaseName))
		data := &ServiceModeEnableData{
			Enabled:              true,
			AlreadyEnabled:       true,
			StateChanged:         false,
			Message:              status.Message,
			PermissionCode:       permissionCode,
			ScheduledJobsBlocked: status.ScheduledJobsBlocked,
			InfobaseName:         cfg.InfobaseName,
		}
		return h.outputResult(format, data, traceID, start)
	}

	// Подсчёт сессий для отчёта (до включения).
	// Значение приблизительное — берётся из GetServiceModeStatus до вызова EnableServiceMode.
	var sessionsCount int
	if terminateSessions {
		if status != nil {
			sessionsCount = status.ActiveSessions
		} else {
			log.Warn("Подсчёт завершаемых сессий невозможен: статус не был получен")
		}
	}

	// Включение сервисного режима
	log.Info("Вызов EnableServiceMode", slog.Bool("terminate_sessions", terminateSessions))
	err = racClient.EnableServiceMode(ctx, clusterInfo.UUID, infobaseInfo.UUID, terminateSessions)
	if err != nil {
		log.Error("Не удалось включить сервисный режим", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start,
			"RAC.ENABLE_FAILED",
			fmt.Sprintf("Не удалось включить сервисный режим: %v", err))
	}

	// Верификация включения
	err = racClient.VerifyServiceMode(ctx, clusterInfo.UUID, infobaseInfo.UUID, true)
	if err != nil {
		log.Error("Верификация сервисного режима не прошла", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start,
			"RAC.VERIFY_FAILED",
			fmt.Sprintf("Верификация сервисного режима не прошла: %v", err))
	}

	// Определение фактического состояния регламентных заданий после включения
	// (аналогично disable handler — симметричная верификация)
	scheduledJobsBlocked := true
	postStatus, postErr := racClient.GetServiceModeStatus(ctx, clusterInfo.UUID, infobaseInfo.UUID)
	if postErr == nil && postStatus != nil {
		scheduledJobsBlocked = postStatus.ScheduledJobsBlocked
	} else if postErr != nil {
		log.Warn("Не удалось проверить статус регламентных заданий после включения",
			slog.String("error", postErr.Error()))
	}

	log.Info("Сервисный режим успешно включён",
		slog.Bool("terminate_sessions", terminateSessions),
		slog.Int("terminated_sessions_count", sessionsCount),
		slog.Bool("scheduled_jobs_blocked", scheduledJobsBlocked))

	data := &ServiceModeEnableData{
		Enabled:                 true,
		AlreadyEnabled:          false,
		StateChanged:            true,
		Message:                 message,
		PermissionCode:          permissionCode,
		ScheduledJobsBlocked:    scheduledJobsBlocked,
		TerminatedSessionsCount: sessionsCount,
		InfobaseName:            cfg.InfobaseName,
	}

	return h.outputResult(format, data, traceID, start)
}

// outputResult форматирует и выводит результат.
func (h *ServiceModeEnableHandler) outputResult(format string, data *ServiceModeEnableData, traceID string, start time.Time) error {
	// Текстовый формат
	if format != output.FormatJSON {
		return data.writeText(os.Stdout)
	}

	// JSON формат
	result := &output.Result{
		Status:  output.StatusSuccess,
		Command: constants.ActNRServiceModeEnable,
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
func (h *ServiceModeEnableHandler) writeError(format, traceID string, start time.Time, code, message string) error {
	// Текстовый формат — человекочитаемый вывод ошибки
	if format != output.FormatJSON {
		return errhandler.HandleError(message, code)
	}

	// JSON формат — структурированный вывод
	result := &output.Result{
		Status:  output.StatusError,
		Command: constants.ActNRServiceModeEnable,
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

