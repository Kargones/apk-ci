// Package forcedisconnecthandler реализует NR-команду nr-force-disconnect-sessions
// для принудительного завершения сессий информационной базы 1C.
package forcedisconnecthandler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
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
	command.Register(&ForceDisconnectHandler{})
}

// DisconnectedSessionInfo содержит информацию о завершённой сессии.
// Структура согласована с SessionInfoData из servicemodestatushandler.
type DisconnectedSessionInfo struct {
	// UserName — имя пользователя
	UserName string `json:"user_name"`
	// AppID — идентификатор приложения
	AppID string `json:"app_id"`
	// Host — хост, с которого была установлена сессия
	Host string `json:"host"`
	// SessionID — идентификатор сессии
	SessionID string `json:"session_id"`
	// StartedAt — время начала сессии (ISO 8601), для консистентности с status command
	StartedAt string `json:"started_at,omitempty"`
}

// ForceDisconnectData содержит данные ответа о принудительном завершении сессий.
type ForceDisconnectData struct {
	// TerminatedSessionsCount — количество завершённых сессий
	TerminatedSessionsCount int `json:"terminated_sessions_count"`
	// NoActiveSessions — не было активных сессий
	NoActiveSessions bool `json:"no_active_sessions"`
	// StateChanged — было ли произведено реальное изменение (завершены ли сессии)
	StateChanged bool `json:"state_changed"`
	// DelaySec — задержка перед завершением в секундах
	DelaySec int `json:"delay_sec"`
	// InfobaseName — имя информационной базы
	InfobaseName string `json:"infobase_name"`
	// PartialFailure — были ли частичные ошибки при завершении
	PartialFailure bool `json:"partial_failure"`
	// Sessions — список завершённых сессий
	Sessions []DisconnectedSessionInfo `json:"sessions"`
	// Errors — список ошибок при завершении сессий
	Errors []string `json:"errors"`
}

// writeText выводит результат принудительного завершения сессий в человекочитаемом формате.
func (d *ForceDisconnectData) writeText(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "Принудительное завершение сессий: %s\n", d.InfobaseName); err != nil {
		return err
	}

	if d.NoActiveSessions {
		_, err := fmt.Fprintln(w, "Активных сессий нет")
		return err
	}

	switch {
	case !d.StateChanged:
		if _, err := fmt.Fprintln(w, "Состояние не изменено: не удалось завершить ни одну сессию"); err != nil {
			return err
		}
	case d.PartialFailure:
		if _, err := fmt.Fprintf(w, "Завершено сессий: %d из %d\n",
			d.TerminatedSessionsCount, d.TerminatedSessionsCount+len(d.Errors)); err != nil {
			return err
		}
	default:
		if _, err := fmt.Fprintf(w, "Завершено сессий: %d\n", d.TerminatedSessionsCount); err != nil {
			return err
		}
	}

	for i, s := range d.Sessions {
		if _, err := fmt.Fprintf(w, "  %d. %s (%s) — %s\n", i+1, s.UserName, s.AppID, s.Host); err != nil {
			return err
		}
	}

	if len(d.Errors) > 0 {
		if _, err := fmt.Fprintln(w, "Ошибки:"); err != nil {
			return err
		}
		for _, e := range d.Errors {
			if _, err := fmt.Fprintf(w, "  - %s\n", e); err != nil {
				return err
			}
		}
	}

	return nil
}

// ForceDisconnectHandler обрабатывает команду nr-force-disconnect-sessions.
type ForceDisconnectHandler struct {
	// racClient — опциональный RAC клиент (nil в production, mock в тестах)
	racClient rac.Client
}

// Name возвращает имя команды.
func (h *ForceDisconnectHandler) Name() string {
	return constants.ActNRForceDisconnectSessions
}

// Description возвращает описание команды для вывода в help.
func (h *ForceDisconnectHandler) Description() string {
	return "Принудительное завершение сессий информационной базы"
}

// Execute выполняет команду nr-force-disconnect-sessions.
func (h *ForceDisconnectHandler) Execute(ctx context.Context, cfg *config.Config) error {
	start := time.Now()

	traceID := tracing.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = tracing.GenerateTraceID()
	}

	format := os.Getenv("BR_OUTPUT_FORMAT")

	// Story 7.3 AC-8: plan-only для команд без поддержки плана
	// Review #36: !IsDryRun() — dry-run имеет приоритет над plan-only (AC-11).
	if !dryrun.IsDryRun() && dryrun.IsPlanOnly() {
		return dryrun.WritePlanOnlyUnsupported(os.Stdout, constants.ActNRForceDisconnectSessions)
	}

	log := slog.Default().With(slog.String("trace_id", traceID), slog.String("command", constants.ActNRForceDisconnectSessions))

	// Валидация наличия имени информационной базы
	if cfg == nil || cfg.InfobaseName == "" {
		log.Error("Не указано имя информационной базы")
		return h.writeError(format, traceID, start,
			"CONFIG.INFOBASE_MISSING",
			"Не указано имя информационной базы (BR_INFOBASE_NAME)")
	}

	log = log.With(slog.String("infobase", cfg.InfobaseName))
	log.Info("Начало обработки команды принудительного завершения сессий")

	// Чтение grace period из переменной окружения.
	// Максимальное значение 300 секунд (5 минут) — достаточно для уведомления пользователей,
	// но не настолько велико, чтобы заблокировать CI/CD pipeline надолго.
	// При некорректном значении (< 0, > 300, не число) используется 0 (немедленное завершение).
	delaySec := 0
	if delayStr := os.Getenv("BR_DISCONNECT_DELAY_SEC"); delayStr != "" {
		if d, err := strconv.Atoi(delayStr); err == nil && d >= 0 && d <= 300 {
			delaySec = d
		} else {
			log.Warn("Некорректное значение BR_DISCONNECT_DELAY_SEC (допустимо 0-300), используется 0", slog.String("value", delayStr))
		}
	}

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

	// Получение списка сессий
	sessions, err := racClient.GetSessions(ctx, clusterInfo.UUID, infobaseInfo.UUID)
	if err != nil {
		log.Error("Не удалось получить список сессий", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start,
			"RAC.SESSIONS_FAILED",
			fmt.Sprintf("Не удалось получить список сессий: %v", err))
	}

	// Логирование текущего состояния: количество активных сессий
	log.Info("Текущее состояние: активных сессий", slog.Int("count", len(sessions)))

	// Идемпотентность: нет активных сессий
	if len(sessions) == 0 {
		log.Info("Активных сессий нет")
		data := &ForceDisconnectData{
			TerminatedSessionsCount: 0,
			NoActiveSessions:        true,
			StateChanged:            false,
			DelaySec:                delaySec,
			InfobaseName:            cfg.InfobaseName,
			Sessions:                make([]DisconnectedSessionInfo, 0),
			Errors:                  make([]string, 0),
		}
		return h.outputResult(format, data, traceID, start)
	}

	// Логирование сессий для аудита (перед завершением)
	log.Info("Найдены активные сессии для завершения", slog.Int("count", len(sessions)))
	for _, s := range sessions {
		log.Info("Сессия для завершения",
			slog.String("user", s.UserName),
			slog.String("app", s.AppID),
			slog.String("host", s.Host),
			slog.String("session_id", s.SessionID))
	}

	// Grace period с поддержкой отмены через context
	if delaySec > 0 {
		log.Info("Ожидание grace period", slog.Int("delay_sec", delaySec))
		select {
		case <-time.After(time.Duration(delaySec) * time.Second):
		case <-ctx.Done():
			log.Warn("Grace period прерван отменой context", slog.String("error", ctx.Err().Error()))
			return ctx.Err()
		}
	}

	// Завершение сессий поштучно
	terminated := make([]DisconnectedSessionInfo, 0)
	terminateErrors := make([]string, 0)
	for _, s := range sessions {
		if err := racClient.TerminateSession(ctx, clusterInfo.UUID, s.SessionID); err != nil {
			terminateErrors = append(terminateErrors, fmt.Sprintf("сессия %s (%s): %v", s.SessionID, s.UserName, err))
			log.Warn("Не удалось завершить сессию",
				slog.String("session_id", s.SessionID),
				slog.String("user", s.UserName),
				slog.String("error", err.Error()))
		} else {
			startedAt := ""
			if !s.StartedAt.IsZero() {
				startedAt = s.StartedAt.Format(time.RFC3339)
			}
			terminated = append(terminated, DisconnectedSessionInfo{
				UserName:  s.UserName,
				AppID:     s.AppID,
				Host:      s.Host,
				SessionID: s.SessionID,
				StartedAt: startedAt,
			})
		}
	}

	log.Info("Завершение сессий выполнено",
		slog.Int("terminated", len(terminated)),
		slog.Int("errors", len(terminateErrors)))

	data := &ForceDisconnectData{
		TerminatedSessionsCount: len(terminated),
		NoActiveSessions:        false,
		StateChanged:            len(terminated) > 0,
		DelaySec:                delaySec,
		InfobaseName:            cfg.InfobaseName,
		PartialFailure:          len(terminateErrors) > 0 && len(terminated) > 0,
		Sessions:                terminated,
		Errors:                  terminateErrors,
	}

	return h.outputResult(format, data, traceID, start)
}

// outputResult форматирует и выводит результат.
func (h *ForceDisconnectHandler) outputResult(format string, data *ForceDisconnectData, traceID string, start time.Time) error {
	// Текстовый формат
	if format != output.FormatJSON {
		return data.writeText(os.Stdout)
	}

	// JSON формат
	result := &output.Result{
		Status:  output.StatusSuccess,
		Command: constants.ActNRForceDisconnectSessions,
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
func (h *ForceDisconnectHandler) writeError(format, traceID string, start time.Time, code, message string) error {
	// Текстовый формат — человекочитаемый вывод ошибки
	if format != output.FormatJSON {
		return errhandler.HandleError(message, code)
	}

	// JSON формат — структурированный вывод
	result := &output.Result{
		Status:  output.StatusError,
		Command: constants.ActNRForceDisconnectSessions,
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

