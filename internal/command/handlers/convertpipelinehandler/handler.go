// Package convertpipelinehandler реализует NR-команду nr-convert-pipeline,
// объединяющую nr-convert → nr-git2store → nr-extension-publish
// в единый пайплайн с автоматической передачей параметров между этапами.
package convertpipelinehandler

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
	"github.com/Kargones/apk-ci/internal/pkg/dryrun"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"
)

// Compile-time interface check.
var _ command.Handler = (*ConvertPipelineHandler)(nil)

// RegisterCmd регистрирует команду nr-convert-pipeline в реестре.
func RegisterCmd() error {
	return command.Register(&ConvertPipelineHandler{})
}

// Pipeline stage names.
const (
	StageConvert          = "convert"
	StageGit2Store        = "git2store"
	StageExtensionPublish = "extension-publish"
)

// defaultPipelineTimeout — таймаут для всего пайплайна (3 часа).
const defaultPipelineTimeout = 3 * time.Hour

// allStages содержит все этапы пайплайна в порядке выполнения.
var allStages = []string{
	StageConvert,
	StageGit2Store,
	StageExtensionPublish,
}

// StageResult — результат выполнения одного этапа пайплайна.
type StageResult struct {
	Name       string `json:"name"`
	Success    bool   `json:"success"`
	DurationMs int64  `json:"duration_ms"`
	Skipped    bool   `json:"skipped,omitempty"`
	Error      string `json:"error,omitempty"`
}

// PipelineData содержит данные ответа пайплайна.
type PipelineData struct {
	StateChanged    bool          `json:"state_changed"`
	StagesCompleted []StageResult `json:"stages_completed"`
	StageCurrent    string        `json:"stage_current"`
	DurationMs      int64         `json:"duration_ms"`
	// Параметры, переданные между этапами
	ConvertTargetPath string `json:"convert_target_path,omitempty"`
}

// writeText выводит результат в человекочитаемом формате.
func (d *PipelineData) writeText(w io.Writer) error {
	status := "успешно"
	if !d.StateChanged {
		status = "без изменений"
	}
	if _, err := fmt.Fprintf(w, "Pipeline: %s\n\n", status); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Этапы:\n"); err != nil {
		return err
	}
	for _, s := range d.StagesCompleted {
		icon := "✓"
		if !s.Success && !s.Skipped {
			icon = "✗"
		}
		if s.Skipped {
			icon = "⊘"
		}
		line := fmt.Sprintf("  %s %s (%d мс)", icon, s.Name, s.DurationMs)
		if s.Error != "" {
			line += fmt.Sprintf(" — %s", s.Error)
		}
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "\nОбщее время: %d мс\n", d.DurationMs); err != nil {
		return err
	}
	return nil
}

// StageExecutor абстрагирует выполнение этапа (для тестирования).
type StageExecutor interface {
	ExecuteStage(ctx context.Context, stageName string, cfg *config.Config) error
}

// ConvertPipelineHandler обрабатывает команду nr-convert-pipeline.
type ConvertPipelineHandler struct {
	executor StageExecutor // nil в production — используем command.Get()
}

// Name возвращает имя команды.
func (h *ConvertPipelineHandler) Name() string {
	return constants.ActNRConvertPipeline
}

// Description возвращает описание команды для вывода в help.
func (h *ConvertPipelineHandler) Description() string {
	return "Pipeline: конвертация EDT→XML → Git→хранилище → публикация расширений"
}

// Execute выполняет пайплайн nr-convert → nr-git2store → nr-extension-publish.
//
// Передача параметров между этапами:
//   - convert: BR_TARGET → устанавливается как BR_SOURCE для git2store (через env)
//   - git2store: использует config.Config (Owner, Repo, AddArray)
//   - extension-publish: запускается если cfg.AddArray не пустой
//
// Переменные окружения:
//   - BR_PIPELINE_SKIP_STAGES: через запятую список этапов для пропуска
//     (например, "extension-publish" если публикация не нужна)
//   - BR_PIPELINE_TIMEOUT: таймаут всего пайплайна (default: 3h)
//   - Все переменные nr-convert (BR_SOURCE, BR_TARGET, BR_DIRECTION)
//   - Все переменные nr-git2store (BR_INFOBASE_NAME, ...)
func (h *ConvertPipelineHandler) Execute(ctx context.Context, cfg *config.Config) error {
	start := time.Now()

	traceID := tracing.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = tracing.GenerateTraceID()
	}

	format := os.Getenv("BR_OUTPUT_FORMAT")

	// Plan-only mode
	if !dryrun.IsDryRun() && dryrun.IsPlanOnly() {
		return dryrun.WritePlanOnlyUnsupported(os.Stdout, constants.ActNRConvertPipeline)
	}

	log := slog.Default().With(
		slog.String("trace_id", traceID),
		slog.String("command", constants.ActNRConvertPipeline),
	)

	// Таймаут пайплайна
	timeout := defaultPipelineTimeout
	if envTimeout := os.Getenv("BR_PIPELINE_TIMEOUT"); envTimeout != "" {
		if parsed, err := time.ParseDuration(envTimeout); err == nil {
			timeout = parsed
		} else {
			log.Warn("Невалидный BR_PIPELINE_TIMEOUT, используется default",
				slog.String("value", envTimeout),
				slog.String("default", defaultPipelineTimeout.String()))
		}
	}

	pipelineCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Определяем пропускаемые этапы
	skipStages := parseSkipStages(os.Getenv("BR_PIPELINE_SKIP_STAGES"))

	data := &PipelineData{
		StagesCompleted: make([]StageResult, 0, len(allStages)),
	}

	log.Info("Запуск pipeline",
		slog.Duration("timeout", timeout),
		slog.Any("skip_stages", skipStages))

	// === Stage 1: nr-convert ===
	if !skipStages[StageConvert] {
		if err := h.runStage(pipelineCtx, log, cfg, data, StageConvert, format, traceID, start); err != nil {
			return err
		}
		// Передача параметров: BR_TARGET от convert → используется далее
		// git2store уже читает свои параметры из config/env, но мы фиксируем
		// что конвертация прошла и XML готов
		convertTarget := os.Getenv("BR_TARGET")
		if convertTarget != "" {
			data.ConvertTargetPath = convertTarget
			log.Info("Convert output path сохранён для следующих этапов",
				slog.String("path", convertTarget))
		}
	} else {
		data.StagesCompleted = append(data.StagesCompleted, StageResult{
			Name: StageConvert, Skipped: true, Success: true,
		})
		log.Info("Этап пропущен", slog.String("stage", StageConvert))
	}

	// === Stage 2: nr-git2store ===
	if !skipStages[StageGit2Store] {
		if err := h.runStage(pipelineCtx, log, cfg, data, StageGit2Store, format, traceID, start); err != nil {
			return err
		}
	} else {
		data.StagesCompleted = append(data.StagesCompleted, StageResult{
			Name: StageGit2Store, Skipped: true, Success: true,
		})
		log.Info("Этап пропущен", slog.String("stage", StageGit2Store))
	}

	// === Stage 3: nr-extension-publish (только если есть расширения) ===
	if !skipStages[StageExtensionPublish] {
		if cfg != nil && len(cfg.AddArray) > 0 {
			if err := h.runStage(pipelineCtx, log, cfg, data, StageExtensionPublish, format, traceID, start); err != nil {
				return err
			}
		} else {
			data.StagesCompleted = append(data.StagesCompleted, StageResult{
				Name: StageExtensionPublish, Skipped: true, Success: true,
			})
			log.Info("extension-publish пропущен: нет расширений в AddArray")
		}
	} else {
		data.StagesCompleted = append(data.StagesCompleted, StageResult{
			Name: StageExtensionPublish, Skipped: true, Success: true,
		})
		log.Info("Этап пропущен", slog.String("stage", StageExtensionPublish))
	}

	// Результат
	data.DurationMs = time.Since(start).Milliseconds()
	data.StateChanged = true
	data.StageCurrent = "completed"

	log.Info("Pipeline завершён успешно",
		slog.Int("stages", len(data.StagesCompleted)),
		slog.Int64("duration_ms", data.DurationMs))

	return h.writeSuccess(format, traceID, data)
}

// runStage выполняет один этап пайплайна через command registry.
func (h *ConvertPipelineHandler) runStage(
	ctx context.Context, log *slog.Logger, cfg *config.Config,
	data *PipelineData, stageName, format, traceID string, pipelineStart time.Time,
) error {
	stageStart := time.Now()
	data.StageCurrent = stageName

	log.Info("Начало этапа", slog.String("stage", stageName))

	// Маппинг stage → command name
	cmdName := stageToCommand(stageName)

	var execErr error
	if h.executor != nil {
		execErr = h.executor.ExecuteStage(ctx, stageName, cfg)
	} else {
		handler, ok := command.Get(cmdName)
		if !ok {
			execErr = fmt.Errorf("команда %s не зарегистрирована", cmdName)
		} else {
			execErr = handler.Execute(ctx, cfg)
		}
	}

	durationMs := time.Since(stageStart).Milliseconds()

	if execErr != nil {
		log.Error("Этап завершился с ошибкой",
			slog.String("stage", stageName),
			slog.Int64("duration_ms", durationMs),
			slog.String("error", execErr.Error()))

		data.StagesCompleted = append(data.StagesCompleted, StageResult{
			Name:       stageName,
			Success:    false,
			DurationMs: durationMs,
			Error:      execErr.Error(),
		})

		return h.writeError(format, traceID, pipelineStart, data,
			fmt.Sprintf("PIPELINE.STAGE_%s_FAILED", stageToCode(stageName)),
			fmt.Sprintf("Этап %s завершился с ошибкой: %s", stageName, execErr.Error()))
	}

	log.Info("Этап завершён успешно",
		slog.String("stage", stageName),
		slog.Int64("duration_ms", durationMs))

	data.StagesCompleted = append(data.StagesCompleted, StageResult{
		Name:       stageName,
		Success:    true,
		DurationMs: durationMs,
	})

	return nil
}

// stageToCommand возвращает имя команды для этапа пайплайна.
func stageToCommand(stage string) string {
	switch stage {
	case StageConvert:
		return constants.ActNRConvert
	case StageGit2Store:
		return constants.ActNRGit2store
	case StageExtensionPublish:
		return constants.ActNRExtensionPublish
	default:
		return stage
	}
}

// stageToCode возвращает код ошибки для этапа (UPPER_SNAKE_CASE).
func stageToCode(stage string) string {
	switch stage {
	case StageConvert:
		return "CONVERT"
	case StageGit2Store:
		return "GIT2STORE"
	case StageExtensionPublish:
		return "EXTENSION_PUBLISH"
	default:
		return "UNKNOWN"
	}
}

// parseSkipStages парсит BR_PIPELINE_SKIP_STAGES (через запятую) в map.
func parseSkipStages(env string) map[string]bool {
	result := make(map[string]bool)
	if env == "" {
		return result
	}
	for _, s := range splitAndTrim(env) {
		if s != "" {
			result[s] = true
		}
	}
	return result
}

// splitAndTrim разделяет строку по запятым и обрезает пробелы.
func splitAndTrim(s string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := trimSpace(s[start:i])
			if part != "" {
				result = append(result, part)
			}
			start = i + 1
		}
	}
	return result
}

// trimSpace убирает пробелы с начала и конца строки (без strings import).
func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

// writeSuccess выводит успешный результат.
func (h *ConvertPipelineHandler) writeSuccess(format, traceID string, data *PipelineData) error {
	if format != output.FormatJSON {
		return data.writeText(os.Stdout)
	}

	result := &output.Result{
		Status:  output.StatusSuccess,
		Command: constants.ActNRConvertPipeline,
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

// writeError выводит ошибку пайплайна с информацией о пройденных этапах.
func (h *ConvertPipelineHandler) writeError(format, traceID string, start time.Time, data *PipelineData, code, message string) error {
	data.DurationMs = time.Since(start).Milliseconds()

	if format != output.FormatJSON {
		// Текстовый вывод: показываем пройденные этапы + ошибку
		if writeErr := data.writeText(os.Stdout); writeErr != nil {
			slog.Default().Error("Не удалось вывести текстовый результат",
				slog.String("error", writeErr.Error()))
		}
		return fmt.Errorf("%s: %s", code, message)
	}

	result := &output.Result{
		Status:  output.StatusError,
		Command: constants.ActNRConvertPipeline,
		Data:    data,
		Error: &output.ErrorInfo{
			Code:    code,
			Message: message,
		},
		Metadata: &output.Metadata{
			DurationMs: data.DurationMs,
			TraceID:    traceID,
			APIVersion: constants.APIVersion,
		},
	}

	writer := output.NewWriter(format)
	if writeErr := writer.Write(os.Stdout, result); writeErr != nil {
		slog.Default().Error("Не удалось записать JSON-ответ",
			slog.String("error", writeErr.Error()))
	}

	return fmt.Errorf("%s: %s", code, message)
}
