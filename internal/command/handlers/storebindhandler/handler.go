// Package storebindhandler реализует NR-команду nr-storebind
// для привязки хранилища конфигурации к базе данных 1C.
package storebindhandler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/entity/one/convert"
	"github.com/Kargones/apk-ci/internal/pkg/dryrun"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"
)

// Compile-time interface check (AC-7: 5.7).
var _ command.Handler = (*StorebindHandler)(nil)

func RegisterCmd() error {
	// AC-5: Deprecated alias через DeprecatedBridge
	return command.RegisterWithAlias(&StorebindHandler{}, constants.ActStoreBind)
}

// StorebindData содержит данные ответа о привязке хранилища (AC-3, Task 2).
type StorebindData struct {
	// StateChanged — изменилось ли состояние системы (AC-6)
	StateChanged bool `json:"state_changed"`
	// InfobaseName — имя информационной базы
	InfobaseName string `json:"infobase_name"`
	// StorePath — путь к хранилищу
	StorePath string `json:"store_path"`
	// DurationMs — длительность операции в миллисекундах
	DurationMs int64 `json:"duration_ms"`
}

// writeText выводит результат в человекочитаемом формате (AC-4).
func (d *StorebindData) writeText(w io.Writer) error {
	statusText := "успешно"
	if !d.StateChanged {
		// StateChanged=false означает что привязка уже существовала, а не ошибку
		statusText = "без изменений"
	}

	_, err := fmt.Fprintf(w, "Привязка хранилища: %s\n", statusText)
	if err != nil {
		return err
	}

	if _, err = fmt.Fprintf(w, "Информационная база: %s\n", d.InfobaseName); err != nil {
		return err
	}

	if _, err = fmt.Fprintf(w, "Путь к хранилищу: %s\n", d.StorePath); err != nil {
		return err
	}

	return nil
}

// StorebindHandler обрабатывает команду nr-storebind.
type StorebindHandler struct {
	// convertLoader — опциональный загрузчик конфигурации (nil в production, mock в тестах)
	convertLoader ConvertLoader
}

// ConvertLoader интерфейс для загрузки и привязки конфигурации.
// Позволяет заменять реализацию в тестах.
type ConvertLoader interface {
	// LoadFromConfig загружает конфигурацию конвертации
	LoadFromConfig(ctx context.Context, l *slog.Logger, cfg *config.Config) (*convert.Config, error)
	// StoreBind выполняет привязку хранилища к базе данных
	StoreBind(cc *convert.Config, ctx context.Context, l *slog.Logger, cfg *config.Config) error
}

// defaultConvertLoader — реализация ConvertLoader по умолчанию.
type defaultConvertLoader struct{}

func (d *defaultConvertLoader) LoadFromConfig(ctx context.Context, l *slog.Logger, cfg *config.Config) (*convert.Config, error) {
	return convert.LoadFromConfig(ctx, l, cfg)
}

func (d *defaultConvertLoader) StoreBind(cc *convert.Config, ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	return cc.StoreBind(ctx, l, cfg)
}

// Name возвращает имя команды.
func (h *StorebindHandler) Name() string {
	return constants.ActNRStorebind
}

// Description возвращает описание команды для вывода в help.
func (h *StorebindHandler) Description() string {
	return "Привязка хранилища конфигурации к базе данных"
}

// Execute выполняет команду nr-storebind (AC-1, AC-2, AC-6, AC-9, AC-10).
func (h *StorebindHandler) Execute(ctx context.Context, cfg *config.Config) error {
	start := time.Now()

	traceID := tracing.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = tracing.GenerateTraceID()
	}

	format := os.Getenv("BR_OUTPUT_FORMAT")

	// Story 7.3 AC-8: plan-only для команд без поддержки плана
	// Review #36: !IsDryRun() — dry-run имеет приоритет над plan-only (AC-11).
	if !dryrun.IsDryRun() && dryrun.IsPlanOnly() {
		return dryrun.WritePlanOnlyUnsupported(os.Stdout, constants.ActNRStorebind)
	}

	log := slog.Default().With(
		slog.String("trace_id", traceID),
		slog.String("command", constants.ActNRStorebind),
	)

	// Progress: validating (AC-10)
	log.Info("validating: проверка параметров")

	// Валидация наличия имени информационной базы (AC-1)
	if cfg == nil || cfg.InfobaseName == "" {
		log.Error("Не указано имя информационной базы")
		return h.writeError(format, traceID, start,
			"CONFIG.INFOBASE_MISSING",
			"Не указано имя информационной базы (BR_INFOBASE_NAME)")
	}

	log = log.With(slog.String("infobase", cfg.InfobaseName))

	// Получение пути к хранилищу из env (опционально — может быть в cfg)
	storePath := os.Getenv("BR_STORE_PATH")

	log.Info("Запуск привязки хранилища",
		slog.String("store_path", storePath))

	// Progress: connecting (AC-10) — загрузка конфигурации подключения
	log.Info("connecting: загрузка конфигурации подключения")

	// Получение или создание loader
	loader := h.convertLoader
	if loader == nil {
		loader = &defaultConvertLoader{}
	}

	// Загрузка конфигурации конвертации (AC-2: credentials из cfg.SecretConfig)
	ctxPtr := ctx
	cc, err := loader.LoadFromConfig(ctxPtr, log, cfg)
	if err != nil {
		log.Error("Ошибка загрузки конфигурации", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start, "ERR_CONFIG_LOAD", err.Error())
	}

	// Получаем путь к хранилищу из конфигурации, если не указан в env
	if storePath == "" {
		storePath = cc.StoreRoot
	}

	// Progress: binding (AC-10) — привязка базы данных к хранилищу
	log.Info("binding: привязка базы данных к хранилищу")

	// Выполнение привязки хранилища (AC-1)
	err = loader.StoreBind(cc, ctxPtr, log, cfg)
	if err != nil {
		log.Error("Ошибка привязки хранилища", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start, "ERR_STORE_BIND", err.Error())
	}

	// Формирование результата (AC-3, AC-6)
	durationMs := time.Since(start).Milliseconds()
	data := &StorebindData{
		StateChanged: true, // AC-6: при успешной привязке
		InfobaseName: cfg.InfobaseName,
		StorePath:    storePath,
		DurationMs:   durationMs,
	}

	log.Info("Привязка хранилища завершена",
		slog.Bool("state_changed", data.StateChanged),
		slog.String("store_path", data.StorePath),
		slog.Int64("duration_ms", durationMs))

	return h.writeSuccess(format, traceID, start, data)
}

// writeSuccess выводит успешный результат (AC-3, AC-4).
func (h *StorebindHandler) writeSuccess(format, traceID string, start time.Time, data *StorebindData) error {
	// Текстовый формат (AC-4)
	if format != output.FormatJSON {
		return data.writeText(os.Stdout)
	}

	// JSON формат (AC-3)
	result := &output.Result{
		Status:  output.StatusSuccess,
		Command: constants.ActNRStorebind,
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

// writeError выводит структурированную ошибку и возвращает error (AC-9).
// Для text формата НЕ выводим в stdout — main.go уже логирует ошибку.
func (h *StorebindHandler) writeError(format, traceID string, start time.Time, code, message string) error {
	// Текстовый формат — только возвращаем error, main.go выведет через logger
	if format != output.FormatJSON {
		return fmt.Errorf("%s: %s", code, message)
	}

	// JSON формат — структурированный вывод (AC-9)
	result := &output.Result{
		Status:  output.StatusError,
		Command: constants.ActNRStorebind,
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
