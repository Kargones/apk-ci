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

// PipelineData — результат всего пайплайна для JSON/text вывода.
type PipelineData struct {
	StateChanged bool                  `json:"state_changed"`
	Stages       []StageOutcome        `json:"stages"`
	Context      *PipelineContextData  `json:"context,omitempty"`
	DurationMs   int64                 `json:"duration_ms"`
}

// PipelineContextData — сериализуемое представление PipelineContext.
// Содержит результаты каждого этапа, доступные для диагностики.
type PipelineContextData struct {
	Convert    *ConvertStageResult    `json:"convert,omitempty"`
	Git2Store  *Git2StoreStageResult  `json:"git2store,omitempty"`
	ExtPublish *ExtPublishStageResult `json:"extension_publish,omitempty"`
}

// writeText выводит результат пайплайна в человекочитаемом формате.
func (d *PipelineData) writeText(w io.Writer) error {
	status := "успешно"
	if !d.StateChanged {
		status = "ошибка"
	}
	if _, err := fmt.Fprintf(w, "Pipeline: %s\n\nЭтапы:\n", status); err != nil {
		return err
	}
	for _, s := range d.Stages {
		icon := "✓"
		if s.Skipped {
			icon = "⊘"
		} else if !s.Success {
			icon = "✗"
		}
		line := fmt.Sprintf("  %s %s (%d мс)", icon, s.Name, s.DurationMs)
		if s.SkipReason != "" {
			line += fmt.Sprintf(" [%s]", s.SkipReason)
		}
		if s.Error != "" {
			line += fmt.Sprintf(" — %s", s.Error)
		}
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}

	// Вывод переданных данных между этапами
	if d.Context != nil {
		if _, err := fmt.Fprintf(w, "\nДанные между этапами:\n"); err != nil {
			return err
		}
		if c := d.Context.Convert; c != nil {
			if _, err := fmt.Fprintf(w, "  convert.target_path: %s\n", c.TargetPath); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(w, "  convert.direction:   %s\n", c.Direction); err != nil {
				return err
			}
		}
		if g := d.Context.Git2Store; g != nil {
			if g.BackupPath != "" {
				if _, err := fmt.Fprintf(w, "  git2store.backup:    %s\n", g.BackupPath); err != nil {
					return err
				}
			}
		}
		if e := d.Context.ExtPublish; e != nil {
			if _, err := fmt.Fprintf(w, "  extensions:          %v\n", e.ExtensionsPublished); err != nil {
				return err
			}
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

// Execute выполняет пайплайн из атомарных этапов с передачей результатов.
//
// Каждый этап:
//  1. Проверяет условие запуска (ShouldRun)
//  2. Получает данные от предыдущих этапов через PipelineContext (BeforeRun)
//  3. Выполняет NR-команду атомарно
//  4. Сохраняет результат в PipelineContext (AfterRun)
//
// Переменные окружения:
//   - BR_PIPELINE_SKIP_STAGES: этапы для пропуска (через запятую)
//   - BR_PIPELINE_TIMEOUT: таймаут всего пайплайна (default: 3h)
func (h *ConvertPipelineHandler) Execute(ctx context.Context, cfg *config.Config) error {
	start := time.Now()

	traceID := tracing.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = tracing.GenerateTraceID()
	}

	format := getenv("BR_OUTPUT_FORMAT")

	if !dryrun.IsDryRun() && dryrun.IsPlanOnly() {
		return dryrun.WritePlanOnlyUnsupported(os.Stdout, constants.ActNRConvertPipeline)
	}

	log := slog.Default().With(
		slog.String("trace_id", traceID),
		slog.String("command", constants.ActNRConvertPipeline),
	)

	// Таймаут пайплайна
	timeout := defaultPipelineTimeout
	if envTimeout := getenv("BR_PIPELINE_TIMEOUT"); envTimeout != "" {
		if parsed, err := time.ParseDuration(envTimeout); err == nil {
			timeout = parsed
		} else {
			log.Warn("Невалидный BR_PIPELINE_TIMEOUT, используется default",
				slog.String("value", envTimeout))
		}
	}

	pipelineCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Skip-список
	skipStages := parseSkipStages(getenv("BR_PIPELINE_SKIP_STAGES"))

	// Pipeline context — аккумулятор данных между этапами
	pctx := &PipelineContext{Cfg: cfg}

	data := &PipelineData{
		Stages: make([]StageOutcome, 0, 3),
	}

	log.Info("Запуск pipeline",
		slog.Duration("timeout", timeout),
		slog.Any("skip_stages", skipStages))

	// Выполняем этапы последовательно
	stages := buildStages()
	for _, stage := range stages {
		// Проверка skip
		if skipStages[stage.Name] {
			outcome := StageOutcome{
				Name:       stage.Name,
				Skipped:    true,
				Success:    true,
				SkipReason: "пропущен через BR_PIPELINE_SKIP_STAGES",
			}
			data.Stages = append(data.Stages, outcome)
			log.Info("Этап пропущен (skip list)", slog.String("stage", stage.Name))
			continue
		}

		outcome, err := executeStage(pipelineCtx, stage, pctx, log, h.executor)
		data.Stages = append(data.Stages, outcome)

		if err != nil {
			// Этап провалился — останавливаем пайплайн
			data.Context = buildContextData(pctx)
			return h.writeError(format, traceID, start, data,
				fmt.Sprintf("PIPELINE.%s_FAILED", stageToCode(stage.Name)),
				fmt.Sprintf("Этап %s: %s", stage.Name, err.Error()))
		}
	}

	// Успех
	data.DurationMs = time.Since(start).Milliseconds()
	data.StateChanged = true
	data.Context = buildContextData(pctx)

	log.Info("Pipeline завершён успешно",
		slog.Int("stages_total", len(data.Stages)),
		slog.Int64("duration_ms", data.DurationMs))

	return h.writeSuccess(format, traceID, data)
}

// buildContextData конвертирует PipelineContext в сериализуемую форму.
func buildContextData(pctx *PipelineContext) *PipelineContextData {
	if pctx == nil {
		return nil
	}
	cd := &PipelineContextData{
		Convert:    pctx.Convert,
		Git2Store:  pctx.Git2Store,
		ExtPublish: pctx.ExtPublish,
	}
	// Не выводим пустой контекст
	if cd.Convert == nil && cd.Git2Store == nil && cd.ExtPublish == nil {
		return nil
	}
	return cd
}

// stageToCode возвращает код ошибки этапа (UPPER_SNAKE_CASE).
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

// parseSkipStages парсит BR_PIPELINE_SKIP_STAGES в map.
func parseSkipStages(env string) map[string]bool {
	result := make(map[string]bool)
	if env == "" {
		return result
	}
	start := 0
	for i := 0; i <= len(env); i++ {
		if i == len(env) || env[i] == ',' {
			s := trimSpace(env[start:i])
			if s != "" {
				result[s] = true
			}
			start = i + 1
		}
	}
	return result
}

// trimSpace убирает пробелы с начала и конца строки.
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
	return output.NewWriter(format).Write(os.Stdout, result)
}

// writeError выводит ошибку пайплайна с информацией о пройденных этапах.
func (h *ConvertPipelineHandler) writeError(format, traceID string, start time.Time, data *PipelineData, code, message string) error {
	data.DurationMs = time.Since(start).Milliseconds()

	if format != output.FormatJSON {
		_ = data.writeText(os.Stdout) //nolint:errcheck
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

	if writeErr := output.NewWriter(format).Write(os.Stdout, result); writeErr != nil {
		slog.Default().Error("Не удалось записать JSON-ответ",
			slog.String("error", writeErr.Error()))
	}
	return fmt.Errorf("%s: %s", code, message)
}
