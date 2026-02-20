// Package store2dbhandler реализует NR-команду nr-store2db
// для загрузки конфигурации из хранилища в базу данных 1C.
package store2dbhandler

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

// storeVersionLatest — значение по умолчанию для версии хранилища (M-4 fix).
const storeVersionLatest = "latest"

// Compile-time interface check (M-3 fix).
var _ command.Handler = (*Store2DbHandler)(nil)

func RegisterCmd() error {
	// Deprecated: alias "store2db" retained for backward compatibility. Remove in v2.0.0 (Epic 7).
	return command.RegisterWithAlias(&Store2DbHandler{}, constants.ActStore2db)
}

// Store2DbData содержит данные ответа о загрузке конфигурации из хранилища.
type Store2DbData struct {
	// StateChanged — изменилось ли состояние системы
	StateChanged bool `json:"state_changed"`
	// InfobaseName — имя информационной базы
	InfobaseName string `json:"infobase_name"`
	// StoreVersion — версия хранилища (latest если не указана)
	StoreVersion string `json:"store_version"`
	// DurationMs — длительность операции в миллисекундах (AC-4 fix: H-1)
	DurationMs int64 `json:"duration_ms"`
	// MainConfigLoaded — успешно ли загружена основная конфигурация
	MainConfigLoaded bool `json:"main_config_loaded"`
	// ExtensionsLoaded — результаты загрузки расширений (M-1 fix: omitempty для консистентности)
	ExtensionsLoaded []ExtensionLoadResult `json:"extensions_loaded,omitempty"`
}

// ExtensionLoadResult результат загрузки расширения.
type ExtensionLoadResult struct {
	// Name — имя расширения
	Name string `json:"name"`
	// Success — успешно ли загружено
	Success bool `json:"success"`
	// Error — описание ошибки (если не успешно)
	Error string `json:"error,omitempty"`
}

// writeText выводит результат в человекочитаемом формате.
func (d *Store2DbData) writeText(w io.Writer) error {
	statusText := "успешно"
	if !d.MainConfigLoaded {
		statusText = "ошибка"
	}

	_, err := fmt.Fprintf(w, "Загрузка конфигурации из хранилища: %s\n", statusText)
	if err != nil {
		return err
	}

	if _, err = fmt.Fprintf(w, "Информационная база: %s\n", d.InfobaseName); err != nil {
		return err
	}

	if _, err = fmt.Fprintf(w, "Версия хранилища: %s\n", d.StoreVersion); err != nil {
		return err
	}

	if _, err = fmt.Fprintf(w, "Основная конфигурация: %s\n", boolToStatus(d.MainConfigLoaded)); err != nil {
		return err
	}

	// Вывод информации о расширениях
	if len(d.ExtensionsLoaded) > 0 {
		if _, err = fmt.Fprintln(w, "Расширения:"); err != nil {
			return err
		}
		for i, ext := range d.ExtensionsLoaded {
			status := boolToStatus(ext.Success)
			if ext.Error != "" {
				if _, err = fmt.Fprintf(w, "  %d. %s: %s (%s)\n", i+1, ext.Name, status, ext.Error); err != nil {
					return err
				}
			} else {
				if _, err = fmt.Fprintf(w, "  %d. %s: %s\n", i+1, ext.Name, status); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// boolToStatus преобразует bool в текстовый статус.
func boolToStatus(b bool) string {
	if b {
		return "загружена"
	}
	return "ошибка"
}

// Store2DbHandler обрабатывает команду nr-store2db.
type Store2DbHandler struct {
	// convertLoader — опциональный загрузчик конфигурации (nil в production, mock в тестах)
	convertLoader ConvertLoader
}

// ConvertLoader интерфейс для загрузки и привязки конфигурации.
// Позволяет заменять реализацию в тестах.
type ConvertLoader interface {
	// LoadFromConfig загружает конфигурацию конвертации
	LoadFromConfig(ctx context.Context, l *slog.Logger, cfg *config.Config) (*convert.Config, error)
	// StoreBind выполняет привязку хранилища к базе данных
	StoreBind(ctx context.Context, cc *convert.Config, l *slog.Logger, cfg *config.Config) error
}

// defaultConvertLoader — реализация ConvertLoader по умолчанию
type defaultConvertLoader struct{}

func (d *defaultConvertLoader) LoadFromConfig(ctx context.Context, l *slog.Logger, cfg *config.Config) (*convert.Config, error) {
	return convert.LoadFromConfig(ctx, l, cfg)
}

func (d *defaultConvertLoader) StoreBind(ctx context.Context, cc *convert.Config, l *slog.Logger, cfg *config.Config) error {
	return cc.StoreBind(ctx, l, cfg)
}

// Name возвращает имя команды.
func (h *Store2DbHandler) Name() string {
	return constants.ActNRStore2db
}

// Description возвращает описание команды для вывода в help.
func (h *Store2DbHandler) Description() string {
	return "Загрузка конфигурации из хранилища в базу данных"
}

// Execute выполняет команду nr-store2db.
func (h *Store2DbHandler) Execute(ctx context.Context, cfg *config.Config) error {
	start := time.Now()

	traceID := tracing.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = tracing.GenerateTraceID()
	}

	format := os.Getenv("BR_OUTPUT_FORMAT")

	// Story 7.3 AC-8: plan-only для команд без поддержки плана
	// Review #36: !IsDryRun() — dry-run имеет приоритет над plan-only (AC-11).
	if !dryrun.IsDryRun() && dryrun.IsPlanOnly() {
		return dryrun.WritePlanOnlyUnsupported(os.Stdout, constants.ActNRStore2db)
	}

	log := slog.Default().With(
		slog.String("trace_id", traceID),
		slog.String("command", constants.ActNRStore2db),
	)

	// Валидация наличия имени информационной базы (AC-1)
	if cfg == nil || cfg.InfobaseName == "" {
		log.Error("Не указано имя информационной базы")
		return h.writeError(format, traceID, start,
			"CONFIG.INFOBASE_MISSING",
			"Не указано имя информационной базы (BR_INFOBASE_NAME)")
	}

	log = log.With(slog.String("infobase", cfg.InfobaseName))

	// Получение версии хранилища (AC-2)
	storeVersion := os.Getenv("BR_STORE_VERSION")
	if storeVersion == "" || storeVersion == storeVersionLatest {
		storeVersion = storeVersionLatest
	}

	log.Info("Запуск загрузки конфигурации из хранилища",
		slog.String("store_version", storeVersion))

	// Progress: connecting (AC-3)
	log.Info("connecting: подключение к хранилищу")

	// Получение или создание loader
	loader := h.convertLoader
	if loader == nil {
		loader = &defaultConvertLoader{}
	}

	// Progress: loading (AC-3)
	log.Info("loading: загрузка конфигурации")

	// Загрузка конфигурации конвертации
	ctxPtr := ctx
	cc, err := loader.LoadFromConfig(ctxPtr, log, cfg)
	if err != nil {
		log.Error("Ошибка загрузки конфигурации", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start, "ERR_STORE_OP", err.Error())
	}

	// Progress: applying (AC-3)
	log.Info("applying: применение конфигурации к базе данных")

	// Выполнение привязки хранилища (основная конфигурация + расширения)
	// H-2 fix: StoreBind обрабатывает main + extensions внутренне.
	// При ошибке невозможно определить какие расширения успешны (legacy API ограничение).
	err = loader.StoreBind(ctxPtr, cc, log, cfg)
	if err != nil {
		log.Error("Ошибка привязки хранилища", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start, "ERR_STORE_OP", err.Error())
	}

	// Формирование результата (AC-4, H-1 fix: duration_ms в data)
	durationMs := time.Since(start).Milliseconds()
	data := &Store2DbData{
		StateChanged:     true,
		InfobaseName:     cfg.InfobaseName,
		StoreVersion:     storeVersion,
		DurationMs:       durationMs,
		MainConfigLoaded: true,
	}

	// Добавление результатов загрузки расширений (AC-7)
	// H-2 note: если StoreBind успешен, все расширения загружены успешно
	// (legacy API не предоставляет детализацию по каждому расширению)
	if len(cfg.AddArray) > 0 {
		data.ExtensionsLoaded = make([]ExtensionLoadResult, 0, len(cfg.AddArray))
		for _, ext := range cfg.AddArray {
			data.ExtensionsLoaded = append(data.ExtensionsLoaded, ExtensionLoadResult{
				Name:    ext,
				Success: true,
			})
		}
	}

	log.Info("Загрузка конфигурации из хранилища завершена",
		slog.Bool("state_changed", data.StateChanged),
		slog.Bool("main_config_loaded", data.MainConfigLoaded),
		slog.Int("extensions_count", len(data.ExtensionsLoaded)),
		slog.Int64("duration_ms", durationMs))

	return h.writeSuccess(format, traceID, start, data)
}

// writeSuccess выводит успешный результат.
func (h *Store2DbHandler) writeSuccess(format, traceID string, start time.Time, data *Store2DbData) error {
	// Текстовый формат (AC-5)
	if format != output.FormatJSON {
		return data.writeText(os.Stdout)
	}

	// JSON формат (AC-4)
	result := &output.Result{
		Status:  output.StatusSuccess,
		Command: constants.ActNRStore2db,
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

// writeError выводит структурированную ошибку и возвращает error (AC-10).
// M-2 fix: для text формата НЕ выводим в stdout — main.go уже логирует ошибку.
func (h *Store2DbHandler) writeError(format, traceID string, start time.Time, code, message string) error {
	// Текстовый формат — только возвращаем error, main.go выведет через logger
	if format != output.FormatJSON {
		return fmt.Errorf("%s: %s", code, message)
	}

	// JSON формат — структурированный вывод
	result := &output.Result{
		Status:  output.StatusError,
		Command: constants.ActNRStore2db,
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
