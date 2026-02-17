// Package dbupdatehandler реализует NR-команду nr-dbupdate
// для обновления структуры базы данных по конфигурации 1C.
package dbupdatehandler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/onec"
	"github.com/Kargones/apk-ci/internal/adapter/onec/rac"
	"github.com/Kargones/apk-ci/internal/command"
	errhandler "github.com/Kargones/apk-ci/internal/command/handlers/shared"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/alerting"
	"github.com/Kargones/apk-ci/internal/pkg/dryrun"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"
)

// Коды ошибок для команды nr-dbupdate.
const (
	ErrDbUpdateValidation       = "DBUPDATE.VALIDATION_FAILED"
	ErrDbUpdateConfig           = "DBUPDATE.CONFIG_ERROR"
	ErrDbUpdateFailed           = "DBUPDATE.UPDATE_FAILED"
	ErrDbUpdateSecondPassFailed = "DBUPDATE.SECOND_PASS_FAILED"
	ErrDbUpdateTimeout          = "DBUPDATE.TIMEOUT"
	ErrDbUpdateAutoDeps         = "DBUPDATE.AUTO_DEPS_FAILED"

	// defaultTimeout — таймаут по умолчанию для обновления БД.
	defaultTimeout = 30 * time.Minute

	// maxMessages — максимальное количество сообщений в результате (C1 fix).
	maxMessages = 100

	// cleanupTimeout — таймаут для cleanup операций (M1 fix).
	// H4 note: Этот таймаут покрывает 3 RAC вызова в disableServiceModeIfNeeded:
	// GetClusterInfo + GetInfobaseInfo + DisableServiceMode.
	// При необходимости увеличить, учитывая сетевые задержки.
	cleanupTimeout = 30 * time.Second

	// racOperationTimeout — таймаут для операций RAC (H3 fix).
	racOperationTimeout = 60 * time.Second
)

func RegisterCmd() error {
	return command.RegisterWithAlias(&DbUpdateHandler{}, constants.ActDbupdate)
}

// DbUpdateData содержит данные ответа о результате обновления.
type DbUpdateData struct {
	// InfobaseName — имя информационной базы
	InfobaseName string `json:"infobase_name"`
	// Extension — имя расширения (если обновлялось)
	Extension string `json:"extension,omitempty"`
	// Success — успешно ли обновление
	Success bool `json:"success"`
	// Messages — сообщения от платформы
	Messages []string `json:"messages,omitempty"`
	// DurationMs — время выполнения в миллисекундах
	DurationMs int64 `json:"duration_ms"`
	// AutoDeps — был ли использован режим автоматического управления зависимостями
	AutoDeps bool `json:"auto_deps"`
}

// writeText выводит результат обновления в человекочитаемом формате.
func (d *DbUpdateData) writeText(w io.Writer) error {
	status := "✅ Обновление завершено успешно"
	if !d.Success {
		status = "❌ Обновление завершено с ошибками"
	}

	_, err := fmt.Fprintf(w, "%s\n", status)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "База данных: %s\n", d.InfobaseName)
	if err != nil {
		return err
	}

	if d.Extension != "" {
		_, err = fmt.Fprintf(w, "Расширение: %s\n", d.Extension)
		if err != nil {
			return err
		}
	}

	duration := time.Duration(d.DurationMs) * time.Millisecond
	_, err = fmt.Fprintf(w, "Время выполнения: %v\n", duration.Round(time.Millisecond))
	if err != nil {
		return err
	}

	if d.AutoDeps {
		_, err = fmt.Fprintf(w, "Auto-deps: включён\n")
		if err != nil {
			return err
		}
	}

	if len(d.Messages) > 0 {
		_, err = fmt.Fprintf(w, "\nСообщения:\n")
		if err != nil {
			return err
		}
		for _, msg := range d.Messages {
			_, err = fmt.Fprintf(w, "  - %s\n", msg)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// DbUpdateHandler обрабатывает команду nr-dbupdate.
type DbUpdateHandler struct {
	// oneCClient — клиент 1C для тестирования; если nil — создаётся реальный клиент
	oneCClient onec.DatabaseUpdater
	// racClient — клиент RAC для тестирования; если nil — создаётся реальный клиент
	racClient rac.Client
	// verbosePlan — план операций для verbose режима (Story 7.3), добавляется в JSON результат
	verbosePlan *output.DryRunPlan
}

// Name возвращает имя команды.
func (h *DbUpdateHandler) Name() string {
	return constants.ActNRDbupdate
}

// Description возвращает описание команды для вывода в help.
// AC-10: включает описание BR_DRY_RUN для документации.
func (h *DbUpdateHandler) Description() string {
	return "Обновить структуру базы данных по конфигурации. " +
		"Переменная BR_DRY_RUN=true выводит план операций без выполнения"
}

// Execute выполняет команду nr-dbupdate.
func (h *DbUpdateHandler) Execute(ctx context.Context, cfg *config.Config) error {
	start := time.Now()

	traceID := tracing.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = tracing.GenerateTraceID()
	}

	format := os.Getenv("BR_OUTPUT_FORMAT")
	log := slog.Default().With(
		slog.String("trace_id", traceID),
		slog.String("command", constants.ActNRDbupdate),
	)

	// 1. Проверка отмены контекста перед началом работы
	if err := ctx.Err(); err != nil {
		log.Warn("Context отменён до начала выполнения", slog.String("error", err.Error()))
		return h.writeError(ctx, cfg, format, traceID, start, ErrDbUpdateFailed,
			"операция отменена: "+err.Error())
	}

	// 2. Валидация
	if cfg == nil || cfg.InfobaseName == "" {
		log.Error("Не указано имя информационной базы")
		return h.writeError(ctx, cfg, format, traceID, start, ErrDbUpdateValidation,
			"BR_INFOBASE_NAME обязателен")
	}

	log = log.With(slog.String("infobase", cfg.InfobaseName))
	log.Info("Запуск обновления структуры базы данных")

	// 3. Получение информации о БД
	dbInfo := cfg.GetDatabaseInfo(cfg.InfobaseName)
	if dbInfo == nil {
		log.Error("Информационная база не найдена в конфигурации", slog.String("infobase", cfg.InfobaseName))
		return h.writeError(ctx, cfg, format, traceID, start, ErrDbUpdateConfig,
			fmt.Sprintf("информационная база '%s' не найдена в конфигурации", cfg.InfobaseName))
	}

	// 4. Проверка пути к 1cv8 до начала операций
	if cfg.AppConfig == nil || cfg.AppConfig.Paths.Bin1cv8 == "" {
		log.Error("Путь к 1cv8 не указан в конфигурации")
		return h.writeError(ctx, cfg, format, traceID, start, ErrDbUpdateConfig,
			"путь к 1cv8 не указан в конфигурации (app.yaml:paths.bin1cv8)")
	}

	// 5. Валидация WorkDir и TmpDir
	if cfg.WorkDir == "" {
		log.Warn("WorkDir не указан, используется системная временная директория")
	}
	if cfg.TmpDir == "" {
		log.Warn("TmpDir не указан, используется системная временная директория")
	}

	// 6. Построение строки подключения
	connectString := h.buildConnectString(dbInfo, cfg)

	// 7. Получение расширения из переменной окружения
	extension := os.Getenv("BR_EXTENSION")

	// 8. Определение таймаута
	timeout := h.getTimeout()

	// === РЕЖИМЫ ПРЕДПРОСМОТРА (порядок приоритетов!) ===

	// 1. Dry-run: план без выполнения (высший приоритет)
	if dryrun.IsDryRun() {
		log.Info("Dry-run режим: построение плана")
		return h.executeDryRun(cfg, dbInfo, connectString, extension, timeout, format, traceID, start)
	}

	// 2. Plan-only: показать план, не выполнять (Story 7.3 AC-1)
	if dryrun.IsPlanOnly() {
		log.Info("Plan-only режим: отображение плана операций")
		plan := h.buildPlan(cfg, dbInfo, connectString, extension, timeout)
		return output.WritePlanOnlyResult(os.Stdout, format, constants.ActNRDbupdate, traceID, constants.APIVersion, start, plan)
	}

	// 3. Verbose: показать план, ПОТОМ выполнить (Story 7.3 AC-4)
	if dryrun.IsVerbose() {
		log.Info("Verbose режим: отображение плана перед выполнением")
		plan := h.buildPlan(cfg, dbInfo, connectString, extension, timeout)
		if format != output.FormatJSON {
			if err := plan.WritePlanText(os.Stdout); err != nil {
				log.Warn("Не удалось вывести план операций", slog.String("error", err.Error()))
			}
			fmt.Fprintln(os.Stdout)
		}
		// Сохраняем план для добавления в JSON результат
		h.verbosePlan = plan
	}
	// Verbose fall-through by design: план отображён, продолжаем реальное выполнение

	// 9. Создание клиента (или использование mock)
	client := h.getOrCreateOneCClient(cfg)

	// 10. Проверка режима auto-deps
	autoDeps := os.Getenv("BR_AUTO_DEPS") == "true"
	// weEnabledServiceMode = true означает, что МЫ включили сервисный режим (он был выключен)
	// weEnabledServiceMode = false означает, что режим УЖЕ был включён до нас
	var weEnabledServiceMode bool
	var racClient rac.Client

	if autoDeps {
		log.Info("Auto-deps режим включён")
		racClient = h.getOrCreateRacClient(cfg, dbInfo, log)
		if racClient != nil {
			// H3 fix: отдельный таймаут для RAC операций
			racCtx, racCancel := context.WithTimeout(ctx, racOperationTimeout)
			var smErr error
			var alreadyEnabled bool
			alreadyEnabled, smErr = h.enableServiceModeIfNeeded(racCtx, cfg, racClient, log)
			weEnabledServiceMode = !alreadyEnabled // мы включили, если НЕ был уже включён
			racCancel()
			if smErr != nil {
				log.Warn("Не удалось включить сервисный режим", slog.String("error", smErr.Error()))
				// Продолжаем без auto-deps
				autoDeps = false
			}
		} else {
			log.Warn("RAC клиент недоступен, продолжаем без auto-deps")
			autoDeps = false
		}
	}

	// 11. Defer для гарантированного восстановления service mode
	// Отключаем только если МЫ включили режим
	defer func() {
		if autoDeps && weEnabledServiceMode && racClient != nil {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), cleanupTimeout)
			defer cleanupCancel()
			h.disableServiceModeIfNeeded(cleanupCtx, cfg, racClient, log)
		}
	}()

	// 12. Progress bar
	prog := h.createProgress()
	prog.Start("Обновление структуры базы данных...")
	defer prog.Finish()

	// 13. Выполнение обновления
	opts := onec.UpdateOptions{
		ConnectString: connectString,
		Extension:     extension,
		Timeout:       timeout,
		Bin1cv8:       cfg.AppConfig.Paths.Bin1cv8,
	}

	result, err := client.UpdateDBCfg(ctx, opts)
	if err != nil {
		log.Error("Ошибка обновления структуры БД", slog.String("error", err.Error()))
		return h.writeError(ctx, cfg, format, traceID, start, ErrDbUpdateFailed, err.Error())
	}

	// M-4 fix: применяем лимит сообщений после первого прохода тоже
	if len(result.Messages) > maxMessages {
		result.Messages = result.Messages[:maxMessages]
		log.Warn("Количество сообщений превысило лимит после первого прохода, обрезано", slog.Int("max", maxMessages))
	}

	// 14. Для расширений — второй проход (особенность платформы 1C)
	if extension != "" {
		log.Info("Первый проход обновления расширения завершён",
			slog.String("extension", extension),
			slog.Int64("first_pass_duration_ms", result.DurationMs),
			slog.Bool("first_pass_success", result.Success))

		if !result.Success {
			log.Warn("Первый проход расширения не успешен, но продолжаем второй проход",
				slog.String("extension", extension))
		}

		// Проверяем отмену контекста перед вторым проходом
		if ctx.Err() != nil {
			log.Error("Контекст отменён перед вторым проходом", slog.String("error", ctx.Err().Error()))
			return h.writeError(ctx, cfg, format, traceID, start, ErrDbUpdateFailed,
				fmt.Sprintf("операция отменена перед вторым проходом: %s", ctx.Err().Error()))
		}

		prog.Update(0, "Второй проход обновления расширения...")
		log.Info("Второй проход обновления для расширения", slog.String("extension", extension))
		result2, err := client.UpdateDBCfg(ctx, opts)
		if err != nil {
			log.Error("Ошибка второго прохода обновления", slog.String("error", err.Error()))
			return h.writeError(ctx, cfg, format, traceID, start, ErrDbUpdateSecondPassFailed,
				fmt.Sprintf("ошибка второго прохода обновления расширения: %s", err.Error()))
		}
		// Объединяем результаты с лимитом на количество сообщений
		result.Messages = append(result.Messages, result2.Messages...)
		if len(result.Messages) > maxMessages {
			result.Messages = result.Messages[:maxMessages]
			log.Warn("Количество сообщений превысило лимит, обрезано", slog.Int("max", maxMessages))
		}
		result.DurationMs += result2.DurationMs
	}

	duration := time.Since(start)
	if result.Success {
		log.Info("Обновление завершено успешно",
			slog.Duration("duration", duration))
	} else {
		log.Warn("Обновление завершено с предупреждениями",
			slog.Duration("duration", duration),
			slog.Int("messages_count", len(result.Messages)))
	}

	// 15. Формирование данных ответа
	data := &DbUpdateData{
		InfobaseName: cfg.InfobaseName,
		Extension:    extension,
		Success:      result.Success,
		Messages:     result.Messages,
		DurationMs:   duration.Milliseconds(),
		AutoDeps:     autoDeps,
	}

	// Текстовый формат
	if format != output.FormatJSON {
		return data.writeText(os.Stdout)
	}

	// JSON формат
	resultOutput := &output.Result{
		Status:  output.StatusSuccess,
		Command: constants.ActNRDbupdate,
		Data:    data,
		Plan:    h.verbosePlan, // Story 7.3 AC-7: verbose JSON включает план
		Metadata: &output.Metadata{
			DurationMs: duration.Milliseconds(),
			TraceID:    traceID,
			APIVersion: constants.APIVersion,
		},
	}

	writer := output.NewWriter(format)
	return writer.Write(os.Stdout, resultOutput)
}

// writeError выводит структурированную ошибку и возвращает error.
func (h *DbUpdateHandler) writeError(ctx context.Context, cfg *config.Config, format, traceID string, start time.Time, code, message string) error {
	// Отправка алерта с детальным кодом ошибки (#59)
	if cfg != nil && cfg.Alerter != nil {
		_ = cfg.Alerter.Send(ctx, alerting.Alert{
			ErrorCode: code,
			Message:   message,
			Command:   constants.ActNRDbupdate,
			Infobase:  cfg.InfobaseName,
			TraceID:   traceID,
			Timestamp: time.Now(),
			Severity:  alerting.SeverityCritical,
		})
	}
	// Текстовый формат — человекочитаемый вывод ошибки
	if format != output.FormatJSON {
		return errhandler.HandleError(message, code)
	}

	// JSON формат — структурированный вывод
	result := &output.Result{
		Status:  output.StatusError,
		Command: constants.ActNRDbupdate,
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
