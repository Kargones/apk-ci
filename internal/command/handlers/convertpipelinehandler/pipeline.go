// Package convertpipelinehandler реализует NR-команду nr-convert-pipeline,
// объединяющую nr-convert → nr-git2store → nr-extension-publish
// в единый пайплайн с атомарными этапами и передачей результатов между ними.
//
// Архитектура:
//
//	PipelineContext хранит аккумулированные данные всех этапов.
//	Каждый этап (Stage) — атомарная единица:
//	  - получает PipelineContext (может использовать результаты предыдущих этапов)
//	  - выполняет свою работу
//	  - записывает свой результат в PipelineContext
//	  - следующий этап может использовать результат или игнорировать его
package convertpipelinehandler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
)

// PipelineContext — контекст пайплайна, аккумулирующий данные между этапами.
// Каждый этап может читать результаты предыдущих и записывать свои.
type PipelineContext struct {
	// Cfg — общая конфигурация приложения.
	Cfg *config.Config

	// Convert — результат этапа nr-convert (nil если этап не выполнялся).
	Convert *ConvertStageResult

	// Git2Store — результат этапа nr-git2store (nil если этап не выполнялся).
	Git2Store *Git2StoreStageResult

	// ExtPublish — результат этапа nr-extension-publish (nil если этап не выполнялся).
	ExtPublish *ExtPublishStageResult
}

// ConvertStageResult — результат этапа конвертации.
type ConvertStageResult struct {
	// SourcePath — исходный путь (BR_SOURCE).
	SourcePath string `json:"source_path"`
	// TargetPath — путь к результату конвертации (BR_TARGET).
	// Может использоваться следующими этапами как входные данные.
	TargetPath string `json:"target_path"`
	// Direction — направление конвертации (edt2xml / xml2edt).
	Direction string `json:"direction"`
}

// Git2StoreStageResult — результат этапа синхронизации в хранилище.
type Git2StoreStageResult struct {
	// StagesCompleted — количество внутренних этапов git2store.
	StagesCompleted int `json:"stages_completed"`
	// BackupPath — путь к backup хранилища.
	BackupPath string `json:"backup_path,omitempty"`
}

// ExtPublishStageResult — результат этапа публикации расширений.
type ExtPublishStageResult struct {
	// ExtensionsPublished — список опубликованных расширений.
	ExtensionsPublished []string `json:"extensions_published,omitempty"`
}

// Stage — атомарный этап пайплайна.
type Stage struct {
	// Name — имя этапа для логов и вывода.
	Name string
	// CommandName — имя NR-команды для вызова через registry.
	CommandName string
	// ShouldRun — определяет нужно ли запускать этап.
	// Получает PipelineContext для принятия решения.
	// Если nil — этап запускается всегда.
	ShouldRun func(pctx *PipelineContext) bool
	// BeforeRun — подготовка перед запуском: передача данных из предыдущих этапов.
	// Может модифицировать cfg или env на основе PipelineContext.
	// Если nil — ничего не делает.
	BeforeRun func(pctx *PipelineContext, log *slog.Logger)
	// AfterRun — сохранение результатов этапа в PipelineContext.
	// Вызывается после успешного выполнения.
	// Если nil — ничего не делает.
	AfterRun func(pctx *PipelineContext, log *slog.Logger)
}

// buildStages возвращает список этапов пайплайна с логикой передачи данных.
func buildStages() []Stage {
	return []Stage{
		{
			Name:        StageConvert,
			CommandName: constants.ActNRConvert,
			// Convert запускается всегда (ShouldRun = nil).
			AfterRun: func(pctx *PipelineContext, log *slog.Logger) {
				pctx.Convert = &ConvertStageResult{
					SourcePath: getenv("BR_SOURCE"),
					TargetPath: getenv("BR_TARGET"),
					Direction:  getenv("BR_DIRECTION"),
				}
				log.Info("Convert результат сохранён в pipeline context",
					slog.String("target_path", pctx.Convert.TargetPath),
					slog.String("direction", pctx.Convert.Direction))
			},
		},
		{
			Name:        StageGit2Store,
			CommandName: constants.ActNRGit2store,
			// Git2Store запускается всегда (ShouldRun = nil).
			BeforeRun: func(pctx *PipelineContext, log *slog.Logger) {
				// Если convert прошёл — его TargetPath уже в файловой системе,
				// git2store читает параметры из config/env самостоятельно.
				// Логируем связь для трассировки.
				if pctx.Convert != nil && pctx.Convert.TargetPath != "" {
					log.Info("Git2Store использует результат Convert",
						slog.String("convert_target", pctx.Convert.TargetPath))
				}
			},
			AfterRun: func(pctx *PipelineContext, log *slog.Logger) {
				pctx.Git2Store = &Git2StoreStageResult{}
				log.Info("Git2Store результат сохранён в pipeline context")
			},
		},
		{
			Name:        StageExtensionPublish,
			CommandName: constants.ActNRExtensionPublish,
			ShouldRun: func(pctx *PipelineContext) bool {
				// Публикация только если есть расширения в конфигурации.
				return pctx.Cfg != nil && len(pctx.Cfg.AddArray) > 0
			},
			AfterRun: func(pctx *PipelineContext, log *slog.Logger) {
				var published []string
				if pctx.Cfg != nil {
					published = pctx.Cfg.AddArray
				}
				pctx.ExtPublish = &ExtPublishStageResult{
					ExtensionsPublished: published,
				}
				log.Info("ExtensionPublish результат сохранён в pipeline context",
					slog.Int("extensions_count", len(published)))
			},
		},
	}
}

// StageOutcome — результат выполнения одного этапа.
type StageOutcome struct {
	Name       string `json:"name"`
	Success    bool   `json:"success"`
	Skipped    bool   `json:"skipped,omitempty"`
	SkipReason string `json:"skip_reason,omitempty"`
	DurationMs int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

// executeStage выполняет один атомарный этап пайплайна.
// Возвращает StageOutcome и error (non-nil если этап провалился и пайплайн должен остановиться).
func executeStage(
	ctx context.Context,
	stage Stage,
	pctx *PipelineContext,
	log *slog.Logger,
	executor StageExecutor,
) (StageOutcome, error) {
	// Проверяем условие запуска
	if stage.ShouldRun != nil && !stage.ShouldRun(pctx) {
		reason := fmt.Sprintf("условие запуска не выполнено для %s", stage.Name)
		log.Info("Этап пропущен", slog.String("stage", stage.Name), slog.String("reason", reason))
		return StageOutcome{
			Name:       stage.Name,
			Skipped:    true,
			Success:    true,
			SkipReason: reason,
		}, nil
	}

	// BeforeRun — подготовка (передача данных из предыдущих этапов)
	if stage.BeforeRun != nil {
		stage.BeforeRun(pctx, log)
	}

	stageStart := time.Now()
	log.Info("Начало этапа", slog.String("stage", stage.Name))

	// Выполняем команду
	var execErr error
	if executor != nil {
		execErr = executor.ExecuteStage(ctx, stage.Name, pctx.Cfg)
	} else {
		handler, ok := command.Get(stage.CommandName)
		if !ok {
			execErr = fmt.Errorf("команда %s не зарегистрирована", stage.CommandName)
		} else {
			execErr = handler.Execute(ctx, pctx.Cfg)
		}
	}

	durationMs := time.Since(stageStart).Milliseconds()

	if execErr != nil {
		log.Error("Этап завершился с ошибкой",
			slog.String("stage", stage.Name),
			slog.Int64("duration_ms", durationMs),
			slog.String("error", execErr.Error()))
		return StageOutcome{
			Name:       stage.Name,
			Success:    false,
			DurationMs: durationMs,
			Error:      execErr.Error(),
		}, execErr
	}

	// AfterRun — сохранение результатов в контекст
	if stage.AfterRun != nil {
		stage.AfterRun(pctx, log)
	}

	log.Info("Этап завершён успешно",
		slog.String("stage", stage.Name),
		slog.Int64("duration_ms", durationMs))

	return StageOutcome{
		Name:       stage.Name,
		Success:    true,
		DurationMs: durationMs,
	}, nil
}
