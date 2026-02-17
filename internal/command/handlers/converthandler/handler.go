// Package converthandler реализует NR-команду nr-convert
// для конвертации между форматами EDT и XML.
package converthandler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/dryrun"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"
)

// Compile-time interface check (AC-8).
var _ command.Handler = (*ConvertHandler)(nil)

func RegisterCmd() {
	// AC-6: Deprecated alias через DeprecatedBridge
	command.RegisterWithAlias(&ConvertHandler{}, constants.ActConvert)
}

// Допустимые направления конвертации.
const (
	DirectionEdt2xml = "edt2xml"
	DirectionXml2edt = "xml2edt"
)

// defaultEdtTimeout — таймаут по умолчанию для операций EDT (M-2 fix).
const defaultEdtTimeout = 30 * time.Minute

// validatePath проверяет путь на path traversal атаки (H-2 fix).
// Возвращает ошибку если путь содержит опасные компоненты.
//
// Ограничение: не проверяет symlinks — /tmp/symlink_to_etc пройдёт валидацию.
// TODO: Для полной защиты вызывающий код должен использовать filepath.EvalSymlinks()
// после validatePath для существующих путей (Review #34).
func validatePath(path string) error {
	// Путь должен быть абсолютным
	if !filepath.IsAbs(path) {
		return fmt.Errorf("путь должен быть абсолютным: %s", path)
	}

	// Проверяем на попытки выхода из директории ДО очистки
	// (после Clean путь нормализуется и .. исчезают)
	if strings.Contains(path, "..") {
		return fmt.Errorf("путь содержит недопустимые компоненты (..): %s", path)
	}

	return nil
}

// ConvertData содержит данные ответа о конвертации (AC-4).
type ConvertData struct {
	// StateChanged — изменилось ли состояние системы (AC-7)
	StateChanged bool `json:"state_changed"`
	// SourcePath — путь к исходным данным
	SourcePath string `json:"source_path"`
	// TargetPath — путь к результату
	TargetPath string `json:"target_path"`
	// Direction — направление конвертации (edt2xml/xml2edt)
	Direction string `json:"direction"`
	// ToolUsed — использованный инструмент (1cedtcli)
	ToolUsed string `json:"tool_used"`
	// DurationMs — длительность операции в миллисекундах
	DurationMs int64 `json:"duration_ms"`
}

// writeText выводит результат в человекочитаемом формате (AC-5).
func (d *ConvertData) writeText(w io.Writer) error {
	statusText := "успешно"
	if !d.StateChanged {
		statusText = "без изменений"
	}

	_, err := fmt.Fprintf(w, "Конвертация: %s\n", statusText)
	if err != nil {
		return err
	}

	if _, err = fmt.Fprintf(w, "\nСводка:\n"); err != nil {
		return err
	}

	if _, err = fmt.Fprintf(w, "  Направление: %s\n", d.Direction); err != nil {
		return err
	}

	if _, err = fmt.Fprintf(w, "  Исходный путь: %s\n", d.SourcePath); err != nil {
		return err
	}

	if _, err = fmt.Fprintf(w, "  Целевой путь: %s\n", d.TargetPath); err != nil {
		return err
	}

	if _, err = fmt.Fprintf(w, "  Инструмент: %s\n", d.ToolUsed); err != nil {
		return err
	}

	if _, err = fmt.Fprintf(w, "  Длительность: %d мс\n", d.DurationMs); err != nil {
		return err
	}

	return nil
}

// Converter — интерфейс для конвертации (AC-8: тестируемость).
type Converter interface {
	// Convert выполняет конвертацию между форматами EDT и XML
	Convert(ctx context.Context, l *slog.Logger, cfg *config.Config, direction, pathIn, pathOut string) error
}

// ConvertHandler обрабатывает команду nr-convert.
type ConvertHandler struct {
	// converter — опциональный конвертер (nil в production, mock в тестах)
	converter Converter
}

// Name возвращает имя команды.
func (h *ConvertHandler) Name() string {
	return constants.ActNRConvert
}

// Description возвращает описание команды для вывода в help.
func (h *ConvertHandler) Description() string {
	return "Конвертация между форматами EDT и XML"
}

// Execute выполняет команду nr-convert (AC-1, AC-2, AC-7, AC-10, AC-11).
func (h *ConvertHandler) Execute(ctx context.Context, cfg *config.Config) error {
	start := time.Now()

	traceID := tracing.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = tracing.GenerateTraceID()
	}

	format := os.Getenv("BR_OUTPUT_FORMAT")

	// Story 7.3 AC-8: plan-only для команд без поддержки плана
	// Review #36: !IsDryRun() — dry-run имеет приоритет над plan-only (AC-11).
	if !dryrun.IsDryRun() && dryrun.IsPlanOnly() {
		return dryrun.WritePlanOnlyUnsupported(os.Stdout, constants.ActNRConvert)
	}

	log := slog.Default().With(
		slog.String("trace_id", traceID),
		slog.String("command", constants.ActNRConvert),
	)

	// Progress: validating (AC-11)
	log.Info("validating: проверка параметров")

	// Валидация BR_SOURCE (AC-1, AC-2)
	source := os.Getenv("BR_SOURCE")
	if source == "" {
		log.Error("Не указан путь к исходным данным")
		return h.writeError(format, traceID, start, "CONFIG.SOURCE_MISSING",
			"Не указан путь к исходным данным (BR_SOURCE)")
	}

	// Валидация BR_TARGET
	target := os.Getenv("BR_TARGET")
	if target == "" {
		log.Error("Не указан путь к результату")
		return h.writeError(format, traceID, start, "CONFIG.TARGET_MISSING",
			"Не указан путь к результату (BR_TARGET)")
	}

	// Валидация BR_DIRECTION (AC-1, AC-2)
	direction := os.Getenv("BR_DIRECTION")
	if direction == "" {
		log.Error("Не указано направление конвертации")
		return h.writeError(format, traceID, start, "CONFIG.DIRECTION_MISSING",
			"Не указано направление конвертации (BR_DIRECTION)")
	}

	if direction != DirectionEdt2xml && direction != DirectionXml2edt {
		log.Error("Недопустимое направление конвертации",
			slog.String("direction", direction))
		return h.writeError(format, traceID, start, "CONFIG.DIRECTION_INVALID",
			fmt.Sprintf("Недопустимое направление '%s', ожидается: %s или %s",
				direction, DirectionEdt2xml, DirectionXml2edt))
	}

	// Проверка существования source path
	if _, err := os.Stat(source); os.IsNotExist(err) {
		log.Error("Исходный путь не существует", slog.String("source", source))
		return h.writeError(format, traceID, start, "ERR_SOURCE_NOT_FOUND",
			fmt.Sprintf("Исходный путь не существует: %s", source))
	}

	// H-2 fix: Валидация source path на path traversal
	if err := validatePath(source); err != nil {
		log.Error("Недопустимый исходный путь", slog.String("source", source), slog.String("error", err.Error()))
		return h.writeError(format, traceID, start, "ERR_SOURCE_INVALID",
			fmt.Sprintf("Недопустимый исходный путь: %s", err.Error()))
	}

	// H-2 fix: Валидация target path на path traversal
	if err := validatePath(target); err != nil {
		log.Error("Недопустимый целевой путь", slog.String("target", target), slog.String("error", err.Error()))
		return h.writeError(format, traceID, start, "ERR_TARGET_INVALID",
			fmt.Sprintf("Недопустимый целевой путь: %s", err.Error()))
	}

	// M-1 fix: Предупреждение если target директория уже существует
	if info, err := os.Stat(target); err == nil && info.IsDir() {
		log.Warn("Целевая директория уже существует, файлы могут быть перезаписаны",
			slog.String("target", target))
	}

	log = log.With(
		slog.String("source", source),
		slog.String("target", target),
		slog.String("direction", direction),
	)

	// Progress: preparing (AC-11)
	log.Info("preparing: подготовка к конвертации")

	// M-3 fix: Определяем инструмент из конфигурации (AC-3)
	toolUsed := "1cedtcli"
	if cfg != nil && cfg.ImplementationsConfig != nil && cfg.ImplementationsConfig.ConfigExport != "" {
		toolUsed = cfg.ImplementationsConfig.ConfigExport
	}

	// M-2 fix: Применяем timeout для операции конвертации
	edtTimeout := defaultEdtTimeout
	if cfg != nil && cfg.AppConfig != nil && cfg.AppConfig.EdtTimeout > 0 {
		edtTimeout = cfg.AppConfig.EdtTimeout
	}
	convertCtx, cancel := context.WithTimeout(ctx, edtTimeout)
	defer cancel()

	// Progress: converting (AC-11)
	log.Info("converting: выполнение конвертации", slog.Duration("timeout", edtTimeout))

	// Выполняем конвертацию через интерфейс
	if err := h.convert(convertCtx, log, cfg, direction, source, target); err != nil {
		// Проверяем превышение таймаута
		if convertCtx.Err() == context.DeadlineExceeded {
			log.Error("Превышен таймаут конвертации", slog.Duration("timeout", edtTimeout))
			return h.writeError(format, traceID, start, "ERR_CONVERT_TIMEOUT",
				fmt.Sprintf("Превышен таймаут конвертации (%v)", edtTimeout))
		}
		log.Error("Ошибка конвертации", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start, "ERR_CONVERT", err.Error())
	}

	// Progress: completing (AC-11)
	log.Info("completing: завершение операции")

	// Формирование результата (AC-4, AC-7)
	durationMs := time.Since(start).Milliseconds()
	data := &ConvertData{
		StateChanged: true, // AC-7: успешная конвертация всегда меняет состояние
		SourcePath:   source,
		TargetPath:   target,
		Direction:    direction,
		ToolUsed:     toolUsed,
		DurationMs:   durationMs,
	}

	// L-1 fix: source, target, direction уже в log через .With(), не дублируем
	log.Info("Конвертация успешно завершена",
		slog.String("tool_used", toolUsed),
		slog.Int64("duration_ms", durationMs))

	return h.writeSuccess(format, traceID, data)
}

// convert выполняет конвертацию через интерфейс или production реализацию.
func (h *ConvertHandler) convert(ctx context.Context, l *slog.Logger, cfg *config.Config, direction, pathIn, pathOut string) error {
	if h.converter != nil {
		return h.converter.Convert(ctx, l, cfg, direction, pathIn, pathOut)
	}
	// Production: используем edt.Cli
	return convertProduction(ctx, l, cfg, direction, pathIn, pathOut)
}

// writeSuccess выводит успешный результат (AC-4, AC-5).
func (h *ConvertHandler) writeSuccess(format, traceID string, data *ConvertData) error {
	// Текстовый формат (AC-5)
	if format != output.FormatJSON {
		return data.writeText(os.Stdout)
	}

	// JSON формат (AC-4)
	result := &output.Result{
		Status:  output.StatusSuccess,
		Command: constants.ActNRConvert,
		Data:    data,
		Metadata: &output.Metadata{
			DurationMs: data.DurationMs,
			TraceID:    traceID,
			APIVersion: constants.APIVersion,
		},
	}

	writer := output.NewWriter(format)
	return writer.Write(os.Stdout, result)
}

// writeError выводит структурированную ошибку и возвращает error (AC-10).
// Для text формата НЕ выводим в stdout — main.go уже логирует ошибку.
func (h *ConvertHandler) writeError(format, traceID string, start time.Time, code, message string) error {
	// Текстовый формат — только возвращаем error, main.go выведет через logger
	if format != output.FormatJSON {
		return fmt.Errorf("%s: %s", code, message)
	}

	// JSON формат — структурированный вывод (AC-10)
	result := &output.Result{
		Status:  output.StatusError,
		Command: constants.ActNRConvert,
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
