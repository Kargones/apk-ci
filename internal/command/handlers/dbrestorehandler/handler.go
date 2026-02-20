// Package dbrestorehandler реализует NR-команду nr-dbrestore
// для восстановления базы данных из резервной копии с автоматическим расчётом таймаута.
package dbrestorehandler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/mssql"
	"github.com/Kargones/apk-ci/internal/command"
	errhandler "github.com/Kargones/apk-ci/internal/command/handlers/shared"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/dryrun"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"
)

// Код ошибки для попытки restore в production базу.
const (
	ErrDbRestoreProductionForbidden = "DBRESTORE.PRODUCTION_RESTORE_FORBIDDEN"
	ErrDbRestoreConfigMissing       = "DBRESTORE.CONFIG_MISSING"
	ErrDbRestoreConnectFailed       = "DBRESTORE.CONNECT_FAILED"
	ErrDbRestoreStatsFailed         = "DBRESTORE.STATS_FAILED"
	ErrDbRestoreRestoreFailed       = "DBRESTORE.RESTORE_FAILED"
	ErrDbRestoreServerNotFound      = "DBRESTORE.SERVER_NOT_FOUND"

	// minTimeout — минимальный таймаут если статистика пуста
	minTimeout = 5 * time.Minute
	// timeoutMultiplier — множитель для auto-timeout (1.7 как в legacy коде)
	timeoutMultiplier = 1.7
	// statisticPeriodDays — период в днях для расчёта статистики восстановления (L1 fix)
	statisticPeriodDays = 120
)

func RegisterCmd() error {
	// Deprecated: alias "dbrestore" retained for backward compatibility. Remove in v2.0.0 (Epic 7).
	return command.RegisterWithAlias(&DbRestoreHandler{}, constants.ActDbrestore)
}

// DbRestoreData содержит данные ответа о результате восстановления.
type DbRestoreData struct {
	// SrcServer — сервер-источник резервной копии
	SrcServer string `json:"src_server"`
	// SrcDB — имя базы данных источника
	SrcDB string `json:"src_db"`
	// DstServer — целевой сервер для восстановления
	DstServer string `json:"dst_server"`
	// DstDB — имя целевой базы данных
	DstDB string `json:"dst_db"`
	// DurationMs — время выполнения операции в миллисекундах
	DurationMs int64 `json:"duration_ms"`
	// TimeoutMs — использованный таймаут в миллисекундах
	TimeoutMs int64 `json:"timeout_ms"`
	// AutoTimeout — был ли таймаут рассчитан автоматически
	AutoTimeout bool `json:"auto_timeout"`
}

// writeText выводит результат восстановления в человекочитаемом формате.
func (d *DbRestoreData) writeText(w io.Writer) error {
	autoTimeoutText := "нет"
	if d.AutoTimeout {
		autoTimeoutText = "да"
	}

	duration := time.Duration(d.DurationMs) * time.Millisecond
	timeout := time.Duration(d.TimeoutMs) * time.Millisecond

	_, err := fmt.Fprintf(w,
		"✅ Восстановление завершено\n"+
			"Источник: %s/%s\n"+
			"Назначение: %s/%s\n"+
			"Время выполнения: %v\n"+
			"Таймаут: %v (авто: %s)\n",
		d.SrcServer, d.SrcDB,
		d.DstServer, d.DstDB,
		duration.Round(time.Millisecond),
		timeout.Round(time.Second),
		autoTimeoutText)
	return err
}

// DbRestoreHandler обрабатывает команду nr-dbrestore.
type DbRestoreHandler struct {
	// mssqlClient — опциональный MSSQL клиент (nil в production, mock в тестах)
	mssqlClient mssql.Client
	// verbosePlan — план операций для verbose режима (Story 7.3), добавляется в JSON результат
	verbosePlan *output.DryRunPlan
}

// Name возвращает имя команды.
func (h *DbRestoreHandler) Name() string {
	return constants.ActNRDbrestore
}

// Description возвращает описание команды для вывода в help.
// AC-10: включает описание BR_DRY_RUN для документации.
func (h *DbRestoreHandler) Description() string {
	return "Восстановление базы данных из backup с автоматическим расчётом таймаута. " +
		"Переменная BR_DRY_RUN=true выводит план операций без выполнения"
}

// Execute выполняет команду nr-dbrestore.
func (h *DbRestoreHandler) Execute(ctx context.Context, cfg *config.Config) error {
	start := time.Now()

	traceID := tracing.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = tracing.GenerateTraceID()
	}

	format := os.Getenv("BR_OUTPUT_FORMAT")
	log := slog.Default().With(slog.String("trace_id", traceID), slog.String("command", constants.ActNRDbrestore))

	// Валидация наличия имени информационной базы
	if cfg == nil || cfg.InfobaseName == "" {
		log.Error("Не указано имя информационной базы")
		return h.writeError(format, traceID, start,
			ErrDbRestoreConfigMissing,
			"Не указано имя информационной базы (BR_INFOBASE_NAME)")
	}

	log = log.With(slog.String("infobase", cfg.InfobaseName))
	log.Info("Запуск восстановления базы данных")

	// КРИТИЧНО: Проверка IsProduction — НИКОГДА не restore В production базу!
	if isProductionDatabase(cfg, cfg.InfobaseName) {
		log.Error("Попытка восстановления в production базу", slog.String("database", cfg.InfobaseName))
		return h.writeError(format, traceID, start,
			ErrDbRestoreProductionForbidden,
			fmt.Sprintf("Восстановление в production базу '%s' запрещено", cfg.InfobaseName))
	}

	// Определение серверов источника и назначения
	srcDB, srcServer, dstServer, err := determineSrcAndDstServers(cfg, cfg.InfobaseName)
	if err != nil {
		log.Error("Не удалось определить серверы", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start,
			ErrDbRestoreServerNotFound,
			err.Error())
	}

	log.Info("Серверы определены",
		slog.String("src_server", srcServer),
		slog.String("src_db", srcDB),
		slog.String("dst_server", dstServer),
		slog.String("dst_db", cfg.InfobaseName))

	// === РЕЖИМЫ ПРЕДПРОСМОТРА (порядок приоритетов!) ===

	// 1. Dry-run: план без выполнения (высший приоритет)
	if dryrun.IsDryRun() {
		log.Info("Dry-run режим: построение плана")
		return h.executeDryRun(cfg, srcDB, srcServer, dstServer, format, traceID, start, log)
	}

	// 2. Plan-only: показать план, не выполнять (Story 7.3 AC-1)
	if dryrun.IsPlanOnly() {
		log.Info("Plan-only режим: отображение плана операций")
		plan := h.buildPlan(cfg, srcDB, srcServer, dstServer)
		return output.WritePlanOnlyResult(os.Stdout, format, constants.ActNRDbrestore, traceID, constants.APIVersion, start, plan)
	}

	// 3. Verbose: показать план, ПОТОМ выполнить (Story 7.3 AC-4)
	if dryrun.IsVerbose() {
		log.Info("Verbose режим: отображение плана перед выполнением")
		plan := h.buildPlan(cfg, srcDB, srcServer, dstServer)
		if format != output.FormatJSON {
			if writeErr := plan.WritePlanText(os.Stdout); writeErr != nil {
				log.Warn("Не удалось вывести план операций", slog.String("error", writeErr.Error()))
			}
			fmt.Fprintln(os.Stdout) //nolint:errcheck // writing to stdout
		}
		h.verbosePlan = plan
	}
	// Verbose fall-through by design: план отображён, продолжаем реальное выполнение

	// Получение или создание MSSQL клиента
	mssqlClient := h.mssqlClient
	if mssqlClient == nil {
		mssqlClient, err = h.createMSSQLClient(cfg, dstServer)
		if err != nil {
			log.Error("Не удалось создать MSSQL клиент", slog.String("error", err.Error()))
			return h.writeError(format, traceID, start,
				ErrDbRestoreConnectFailed,
				fmt.Sprintf("Не удалось создать MSSQL клиент: %v", err))
		}
	}

	// Подключение к серверу
	if err := mssqlClient.Connect(ctx); err != nil {
		log.Error("Не удалось подключиться к MSSQL", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start,
			ErrDbRestoreConnectFailed,
			fmt.Sprintf("Не удалось подключиться к MSSQL серверу: %v", err))
	}
	defer func() {
		if closeErr := mssqlClient.Close(); closeErr != nil {
			log.Warn("Ошибка закрытия соединения MSSQL", slog.String("error", closeErr.Error()))
		}
	}()

	// C-1 fix: проверка отмены контекста после подключения
	if ctx.Err() != nil {
		log.Warn("Контекст отменён после подключения к MSSQL", slog.String("error", ctx.Err().Error()))
		return h.writeError(format, traceID, start,
			ErrDbRestoreConnectFailed,
			fmt.Sprintf("Операция отменена после подключения: %v", ctx.Err()))
	}

	// Определение таймаута (используем уже подключённый клиент)
	// Возвращаем также estimatedDuration для корректного Total в progress
	timeout, autoTimeout, hasStats, estimatedDuration := h.calculateTimeout(ctx, cfg, mssqlClient, srcDB, dstServer, log)

	log.Info("Таймаут определён",
		slog.Duration("timeout", timeout),
		slog.Bool("auto", autoTimeout),
		slog.Bool("has_stats", hasStats))

	// Подготовка параметров восстановления (M4 fix - используем helper)
	moscowTZ := getMoscowTimezone()
	nowInMoscow := time.Now().In(moscowTZ)

	restoreOpts := mssql.RestoreOptions{
		Description:   "gitops db restore task (NR)",
		TimeToRestore: nowInMoscow.Format("2006-01-02T15:04:05"),
		User:          h.getUser(cfg),
		SrcServer:     srcServer,
		SrcDB:         srcDB,
		DstServer:     dstServer,
		DstDB:         cfg.InfobaseName,
		Timeout:       timeout,
	}

	log.Info("Начало восстановления базы данных",
		slog.String("time_to_restore", restoreOpts.TimeToRestore))

	// Создаём progress для отображения прогресса восстановления
	// AC-1: показываем progress только для операций > 30 секунд (timeout — приблизительная оценка)
	// AC-7: BR_SHOW_PROGRESS=false отключает progress
	// Task 7.4: если stats недоступна — используем SpinnerProgress (Total=0)
	// Используем estimatedDuration (реальная оценка) вместо timeout (максимум)
	prog := h.createProgress(timeout, hasStats, estimatedDuration)
	prog.Start("Восстановление базы данных...")

	// Запускаем горутину для обновления progress каждую секунду
	// C-2 fix: используем atomic flag для предотвращения race condition
	// между close(done) и последним вызовом prog.Update()
	var wg sync.WaitGroup
	var stopped atomic.Bool
	done := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		var elapsed int64
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Проверяем флаг остановки перед Update() для избежания race
				if stopped.Load() {
					return
				}
				elapsed++
				prog.Update(elapsed*1000, "Восстановление...")
			}
		}
	}()

	// Выполнение восстановления
	restoreErr := mssqlClient.Restore(ctx, restoreOpts)
	stopped.Store(true) // Устанавливаем флаг ДО закрытия канала
	close(done)
	wg.Wait() // Ждём завершения горутины перед Finish()

	if restoreErr != nil {
		prog.Finish()
		log.Error("Ошибка восстановления базы данных", slog.String("error", restoreErr.Error()))
		return h.writeError(format, traceID, start,
			ErrDbRestoreRestoreFailed,
			fmt.Sprintf("Ошибка восстановления базы данных: %v", restoreErr))
	}

	prog.Finish()

	duration := time.Since(start)
	log.Info("Восстановление завершено успешно",
		slog.Duration("duration", duration))

	// Формирование данных ответа
	data := &DbRestoreData{
		SrcServer:   srcServer,
		SrcDB:       srcDB,
		DstServer:   dstServer,
		DstDB:       cfg.InfobaseName,
		DurationMs:  duration.Milliseconds(),
		TimeoutMs:   timeout.Milliseconds(),
		AutoTimeout: autoTimeout,
	}

	// Текстовый формат
	if format != output.FormatJSON {
		return data.writeText(os.Stdout)
	}

	// JSON формат
	result := &output.Result{
		Status:  output.StatusSuccess,
		Command: constants.ActNRDbrestore,
		Data:    data,
		Plan:    h.verbosePlan, // Story 7.3 AC-7: verbose JSON включает план
		Metadata: &output.Metadata{
			DurationMs: duration.Milliseconds(),
			TraceID:    traceID,
			APIVersion: constants.APIVersion,
		},
	}

	writer := output.NewWriter(format)
	return writer.Write(os.Stdout, result)
}

// writeError выводит структурированную ошибку и возвращает error.
func (h *DbRestoreHandler) writeError(format, traceID string, start time.Time, code, message string) error {
	// Текстовый формат — человекочитаемый вывод ошибки
	if format != output.FormatJSON {
		return errhandler.HandleError(message, code)
	}

	// JSON формат — структурированный вывод
	result := &output.Result{
		Status:  output.StatusError,
		Command: constants.ActNRDbrestore,
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
