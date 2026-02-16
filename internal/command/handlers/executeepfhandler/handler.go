// Package executeepfhandler реализует NR-команду nr-execute-epf
// для выполнения внешних обработок 1C (.epf).
package executeepfhandler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/entity/one/enterprise"
	"github.com/Kargones/apk-ci/internal/pkg/dryrun"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"
)

// DefaultTimeout — timeout по умолчанию для выполнения EPF (5 минут).
const DefaultTimeout = 300 * time.Second

// Compile-time interface check.
var _ command.Handler = (*ExecuteEpfHandler)(nil)

func init() {
	command.RegisterWithAlias(&ExecuteEpfHandler{}, constants.ActExecuteEpf)
}

// ExecuteEpfData содержит данные ответа о выполнении внешней обработки.
type ExecuteEpfData struct {
	// StateChanged — изменилось ли состояние системы (EPF может изменять данные)
	StateChanged bool `json:"state_changed"`
	// EpfPath — путь/URL к файлу .epf
	EpfPath string `json:"epf_path"`
	// InfobaseName — имя информационной базы
	InfobaseName string `json:"infobase_name"`
	// DurationMs — длительность операции в миллисекундах
	DurationMs int64 `json:"duration_ms"`
}

// writeText выводит результат в человекочитаемом формате.
func (d *ExecuteEpfData) writeText(w io.Writer) error {
	_, err := fmt.Fprintf(w, "Внешняя обработка выполнена успешно\n")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "Файл: %s\n", d.EpfPath)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "База: %s\n", d.InfobaseName)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "Время выполнения: %d мс\n", d.DurationMs)
	return err
}

// EpfExecutor интерфейс для выполнения внешних обработок (для тестируемости).
type EpfExecutor interface {
	Execute(ctx context.Context, cfg *config.Config) error
}

// ExecuteEpfHandler обрабатывает команду nr-execute-epf.
type ExecuteEpfHandler struct {
	// executor — опциональный исполнитель EPF (nil в production, mock в тестах)
	executor EpfExecutor
}

// Name возвращает имя команды.
func (h *ExecuteEpfHandler) Name() string {
	return constants.ActNRExecuteEpf
}

// Description возвращает описание команды для help.
func (h *ExecuteEpfHandler) Description() string {
	return "Выполнение внешней обработки 1C (.epf)"
}

// Execute выполняет команду.
func (h *ExecuteEpfHandler) Execute(ctx context.Context, cfg *config.Config) error {
	start := time.Now()

	traceID := tracing.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = tracing.GenerateTraceID()
	}

	format := os.Getenv("BR_OUTPUT_FORMAT")

	// Story 7.3 AC-8: plan-only для команд без поддержки плана
	// Review #36: !IsDryRun() — dry-run имеет приоритет над plan-only (AC-11).
	if !dryrun.IsDryRun() && dryrun.IsPlanOnly() {
		return dryrun.WritePlanOnlyUnsupported(os.Stdout, constants.ActNRExecuteEpf)
	}

	log := slog.Default().With(
		slog.String("trace_id", traceID),
		slog.String("command", constants.ActNRExecuteEpf),
	)

	// Валидация: проверка конфигурации
	if cfg == nil {
		log.Error("Конфигурация не указана")
		return h.writeError(format, traceID, start, "ERR_EXECUTE_EPF_VALIDATION",
			"Конфигурация не может быть nil")
	}

	// Валидация: BR_EPF_PATH (cfg.StartEpf)
	if cfg.StartEpf == "" {
		log.Error("EPF path не указан")
		return h.writeError(format, traceID, start, "ERR_EXECUTE_EPF_VALIDATION",
			"BR_EPF_PATH (BR_START_EPF) не указан")
	}

	// Валидация: BR_INFOBASE_NAME
	if cfg.InfobaseName == "" {
		log.Error("Infobase name не указан")
		return h.writeError(format, traceID, start, "ERR_EXECUTE_EPF_VALIDATION",
			"BR_INFOBASE_NAME не указан")
	}

	// Валидация формата URL
	if !isValidURL(cfg.StartEpf) {
		log.Error("Невалидный формат URL", slog.String("epf_path", cfg.StartEpf))
		return h.writeError(format, traceID, start, "ERR_EXECUTE_EPF_VALIDATION",
			fmt.Sprintf("некорректный URL для EPF: %s (требуется http:// или https://)", cfg.StartEpf))
	}

	// Timeout из BR_EPF_TIMEOUT (default 300 секунд)
	timeout := DefaultTimeout
	if timeoutStr := os.Getenv("BR_EPF_TIMEOUT"); timeoutStr != "" {
		if t, err := strconv.Atoi(timeoutStr); err == nil && t > 0 {
			timeout = time.Duration(t) * time.Second
		} else {
			log.Warn("Невалидное значение BR_EPF_TIMEOUT, используется default",
				slog.String("value", timeoutStr),
				slog.Int("default_seconds", int(DefaultTimeout.Seconds())),
			)
		}
	}

	log = log.With(slog.String("infobase", cfg.InfobaseName))

	// Создание context с timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	log.Info("Запуск выполнения внешней обработки",
		slog.String("epf_path", safeLogURL(cfg.StartEpf)),
		slog.Duration("timeout", timeout),
	)

	// Получаем исполнитель
	executor := h.getExecutor(cfg)

	// Выполняем EPF
	if err := executor.Execute(ctxWithTimeout, cfg); err != nil {
		log.Error("Ошибка выполнения EPF", slog.String("error", err.Error()))
		// Распознаём тип ошибки для правильного кода (AC-7)
		errCode := "ERR_EXECUTE_EPF_EXECUTION"
		if strings.Contains(err.Error(), "ошибка получения данных .epf файла") ||
			strings.Contains(err.Error(), "ошибка создания временного файла") {
			errCode = "ERR_EXECUTE_EPF_DOWNLOAD"
		}
		return h.writeError(format, traceID, start, errCode, err.Error())
	}

	// Формируем результат
	durationMs := time.Since(start).Milliseconds()
	data := &ExecuteEpfData{
		StateChanged: true, // EPF мог изменить данные
		EpfPath:      cfg.StartEpf,
		InfobaseName: cfg.InfobaseName,
		DurationMs:   durationMs,
	}

	log.Info("Внешняя обработка успешно выполнена",
		slog.String("epf_path", safeLogURL(cfg.StartEpf)),
		slog.Int64("duration_ms", durationMs),
	)

	return h.writeSuccess(format, traceID, start, data)
}

// getExecutor возвращает EpfExecutor (mock в тестах, production в реальном коде).
func (h *ExecuteEpfHandler) getExecutor(cfg *config.Config) EpfExecutor {
	if h.executor != nil {
		return h.executor
	}
	return enterprise.NewEpfExecutor(slog.Default(), cfg.WorkDir)
}

// writeSuccess выводит успешный результат.
func (h *ExecuteEpfHandler) writeSuccess(format, traceID string, start time.Time, data *ExecuteEpfData) error {
	// Текстовый формат (AC-5)
	if format != output.FormatJSON {
		return data.writeText(os.Stdout)
	}

	// JSON формат (AC-4)
	result := &output.Result{
		Status:  output.StatusSuccess,
		Command: constants.ActNRExecuteEpf,
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

// writeError выводит структурированную ошибку и возвращает error (AC-7).
func (h *ExecuteEpfHandler) writeError(format, traceID string, start time.Time, code, message string) error {
	// Текстовый формат — только возвращаем error, main.go выведет через logger
	if format != output.FormatJSON {
		return fmt.Errorf("%s: %s", code, message)
	}

	// JSON формат — структурированный вывод
	result := &output.Result{
		Status:  output.StatusError,
		Command: constants.ActNRExecuteEpf,
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

// isValidURL проверяет что строка является валидным HTTP(S) URL.
// Review #34 fix: используем url.Parse для более строгой валидации вместо HasPrefix.
// TODO: M-2 tech debt — дублирует логику enterprise.EpfExecutor.validateEpfURL().
// При рефакторинге вынести в общую утилиту internal/pkg/validation/.
func isValidURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

// safeLogURL возвращает URL без токенов и credentials для безопасного логирования.
// Обрезает query string (где могут быть access_token), userinfo и длинные URL.
// Review #34 fix: добавлено удаление userinfo (user:password@host).
func safeLogURL(rawURL string) string {
	// Удаляем userinfo (credentials) из URL
	if parsed, err := url.Parse(rawURL); err == nil && parsed.User != nil {
		parsed.User = nil
		rawURL = parsed.String()
	}
	// Удаляем query string где могут быть токены
	if idx := strings.Index(rawURL, "?"); idx > 0 {
		rawURL = rawURL[:idx] + "?..."
	}
	// Обрезаем слишком длинные URL
	if len(rawURL) > 100 {
		return rawURL[:100] + "..."
	}
	return rawURL
}
