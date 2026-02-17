// Package servicemodestatushandler реализует NR-команду nr-service-mode-status
// для проверки статуса сервисного режима информационной базы 1C.
package servicemodestatushandler

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
	// Deprecated: alias "service-mode-status" retained for backward compatibility. Remove in v2.0.0 (Epic 7).
	return command.RegisterWithAlias(&ServiceModeStatusHandler{}, constants.ActServiceModeStatus)
}

// SessionInfoData содержит данные о сессии пользователя для JSON-сериализации.
type SessionInfoData struct {
	// UserName — имя пользователя
	UserName string `json:"user_name"`
	// Host — хост, с которого установлена сессия
	Host string `json:"host"`
	// StartedAt — время начала сессии (ISO 8601), пустая строка если время не определено
	StartedAt string `json:"started_at,omitempty"`
	// LastActiveAt — время последней активности (ISO 8601), пустая строка если время не определено
	LastActiveAt string `json:"last_active_at,omitempty"`
	// AppID — идентификатор приложения
	AppID string `json:"app_id"`
}

// ServiceModeStatusData содержит данные ответа о статусе сервисного режима.
// Примечание: поле StateChanged отсутствует, т.к. status — read-only операция,
// не изменяющая состояние системы.
type ServiceModeStatusData struct {
	// Enabled — включён ли сервисный режим
	Enabled bool `json:"enabled"`
	// Message — сообщение блокировки
	Message string `json:"message"`
	// ScheduledJobsBlocked — заблокированы ли регламентные задания
	ScheduledJobsBlocked bool `json:"scheduled_jobs_blocked"`
	// ActiveSessions — количество активных сессий
	ActiveSessions int `json:"active_sessions"`
	// InfobaseName — имя информационной базы
	InfobaseName string `json:"infobase_name"`
	// Sessions — список активных сессий
	Sessions []SessionInfoData `json:"sessions"`
}

// writeText выводит статус сервисного режима в человекочитаемом формате.
func (d *ServiceModeStatusData) writeText(w io.Writer) error {
	enabledText := "ВЫКЛЮЧЕН"
	if d.Enabled {
		enabledText = "ВКЛЮЧЁН"
	}
	jobsText := "разблокированы"
	if d.ScheduledJobsBlocked {
		jobsText = "заблокированы"
	}

	_, err := fmt.Fprintf(w,
		"Сервисный режим: %s\nИнформационная база: %s\nСообщение: \"%s\"\nРегламентные задания: %s\nАктивные сессии: %d\n",
		enabledText, d.InfobaseName, d.Message, jobsText, d.ActiveSessions)
	if err != nil {
		return err
	}

	// Детали сессий
	if _, err = fmt.Fprintln(w, "Детали сессий:"); err != nil {
		return err
	}
	if len(d.Sessions) == 0 {
		if d.ActiveSessions > 0 {
			// ActiveSessions > 0, но список пуст — ошибка получения деталей сессий
			_, err = fmt.Fprintln(w, "  Не удалось получить детали сессий")
		} else {
			_, err = fmt.Fprintln(w, "  Нет активных сессий")
		}
		return err
	}

	limit := min(len(d.Sessions), 5)
	for i := range limit {
		s := d.Sessions[i]
		if _, err = fmt.Fprintf(w, "  %d. %s (%s) — %s, начало: %s\n",
			i+1, s.UserName, s.AppID, s.Host, s.StartedAt); err != nil {
			return err
		}
	}
	if remaining := len(d.Sessions) - 5; remaining > 0 {
		_, err = fmt.Fprintf(w, "  ... и ещё %d сессий\n", remaining)
	}
	return err
}

// ServiceModeStatusHandler обрабатывает команду nr-service-mode-status.
type ServiceModeStatusHandler struct {
	// racClient — опциональный RAC клиент (nil в production, mock в тестах)
	racClient rac.Client
}

// Name возвращает имя команды.
func (h *ServiceModeStatusHandler) Name() string {
	return constants.ActNRServiceModeStatus
}

// Description возвращает описание команды для вывода в help.
func (h *ServiceModeStatusHandler) Description() string {
	return "Проверка статуса сервисного режима информационной базы"
}

// Execute выполняет команду nr-service-mode-status.
func (h *ServiceModeStatusHandler) Execute(ctx context.Context, cfg *config.Config) error {
	start := time.Now()

	traceID := tracing.TraceIDFromContext(ctx)
	// M-7/Review #17: Fallback на GenerateTraceID() если context propagation сломалась.
	// В production traceID всегда должен быть в context (устанавливается в main.go).
	// Если здесь срабатывает fallback — это сигнал о проблеме в context chain.
	if traceID == "" {
		traceID = tracing.GenerateTraceID()
	}

	format := os.Getenv("BR_OUTPUT_FORMAT")

	// Story 7.3 AC-8: plan-only для команд без поддержки плана
	// Review #36: !IsDryRun() — dry-run имеет приоритет над plan-only (AC-11).
	if !dryrun.IsDryRun() && dryrun.IsPlanOnly() {
		return dryrun.WritePlanOnlyUnsupported(os.Stdout, constants.ActNRServiceModeStatus)
	}

	// Требуется рефакторинг: handlers должны получать logging.Logger через DI.
	log := slog.Default().With(slog.String("trace_id", traceID), slog.String("command", constants.ActNRServiceModeStatus))

	// Валидация наличия имени информационной базы
	if cfg == nil || cfg.InfobaseName == "" {
		log.Error("Не указано имя информационной базы")
		return h.writeError(format, traceID, start,
			"CONFIG.INFOBASE_MISSING",
			"Не указано имя информационной базы (BR_INFOBASE_NAME)")
	}

	log = log.With(slog.String("infobase", cfg.InfobaseName))
	log.Info("Запрос статуса сервисного режима")

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

	// Получение статуса сервисного режима
	status, err := racClient.GetServiceModeStatus(ctx, clusterInfo.UUID, infobaseInfo.UUID)
	if err != nil {
		log.Error("Не удалось получить статус сервисного режима", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start,
			"RAC.STATUS_FAILED",
			fmt.Sprintf("Не удалось получить статус сервисного режима: %v", err))
	}

	// Логирование текущего состояния для диагностики (AC-3 единообразие с enable/disable)
	log.Info("Текущее состояние информационной базы",
		slog.Bool("enabled", status.Enabled),
		slog.Bool("scheduled_jobs_blocked", status.ScheduledJobsBlocked),
		slog.Int("active_sessions", status.ActiveSessions),
		slog.String("message", status.Message))

	// Получение детального списка сессий (graceful degradation при ошибке)
	sessionsData := make([]SessionInfoData, 0)
	sessions, err := racClient.GetSessions(ctx, clusterInfo.UUID, infobaseInfo.UUID)
	if err != nil {
		log.Warn("Не удалось получить список сессий", slog.String("error", err.Error()))
	} else {
		for _, s := range sessions {
			// Форматируем время только если оно не нулевое (RAC может не вернуть время)
			startedAt := ""
			if !s.StartedAt.IsZero() {
				startedAt = s.StartedAt.Format(time.RFC3339)
			}
			lastActiveAt := ""
			if !s.LastActiveAt.IsZero() {
				lastActiveAt = s.LastActiveAt.Format(time.RFC3339)
			}
			sessionsData = append(sessionsData, SessionInfoData{
				UserName:     s.UserName,
				Host:         s.Host,
				StartedAt:    startedAt,
				LastActiveAt: lastActiveAt,
				AppID:        s.AppID,
			})
		}
	}

	// Формирование данных ответа
	data := &ServiceModeStatusData{
		Enabled:              status.Enabled,
		Message:              status.Message,
		ScheduledJobsBlocked: status.ScheduledJobsBlocked,
		ActiveSessions:       status.ActiveSessions,
		InfobaseName:         cfg.InfobaseName,
		Sessions:             sessionsData,
	}

	// Текстовый формат
	if format != output.FormatJSON {
		return data.writeText(os.Stdout)
	}

	// JSON формат
	result := &output.Result{
		Status:  output.StatusSuccess,
		Command: constants.ActNRServiceModeStatus,
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
func (h *ServiceModeStatusHandler) writeError(format, traceID string, start time.Time, code, message string) error {
	// Текстовый формат — человекочитаемый вывод ошибки, консистентный со стилем writeText
	if format != output.FormatJSON {
		return errhandler.HandleError(message, code)
	}

	// JSON формат — структурированный вывод
	result := &output.Result{
		Status:  output.StatusError,
		Command: constants.ActNRServiceModeStatus,
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

