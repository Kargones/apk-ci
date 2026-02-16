// Package dbrestorehandler реализует NR-команду nr-dbrestore
// для восстановления базы данных из резервной копии с автоматическим расчётом таймаута.
package dbrestorehandler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/mssql"
	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/entity/dbrestore"
	"github.com/Kargones/apk-ci/internal/pkg/dryrun"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/progress"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"
	errhandler "github.com/Kargones/apk-ci/internal/command/handlers/shared"
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

func init() {
	command.RegisterWithAlias(&DbRestoreHandler{}, constants.ActDbrestore)
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
			if err := plan.WritePlanText(os.Stdout); err != nil {
				log.Warn("Не удалось вывести план операций", slog.String("error", err.Error()))
			}
			fmt.Fprintln(os.Stdout)
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

// calculateTimeout определяет таймаут для операции восстановления.
// При BR_TIMEOUT_MIN используется явное значение, при BR_AUTO_TIMEOUT=true —
// расчёт на основе статистики, иначе минимальный таймаут.
// Принимает уже подключённый mssqlClient для избежания двойного подключения.
// Возвращает: timeout, autoTimeout, hasStats (для выбора progress типа),
// estimatedDuration (реальная оценка времени, не timeout с множителем).
func (h *DbRestoreHandler) calculateTimeout(ctx context.Context, cfg *config.Config, mssqlClient mssql.Client, srcDB, dstServer string, log *slog.Logger) (time.Duration, bool, bool, time.Duration) {
	// Проверяем явный таймаут через BR_TIMEOUT_MIN (наивысший приоритет)
	if timeoutMinStr := os.Getenv("BR_TIMEOUT_MIN"); timeoutMinStr != "" {
		if timeoutMin, err := strconv.Atoi(timeoutMinStr); err == nil && timeoutMin > 0 {
			log.Info("Используется явный таймаут из BR_TIMEOUT_MIN",
				slog.Int("minutes", timeoutMin))
			// hasStats=true — пользователь явно указал timeout, считаем что знает длительность
			explicitTimeout := time.Duration(timeoutMin) * time.Minute
			// Для явного timeout используем его же как estimated (пользователь знает лучше)
			return explicitTimeout, false, true, explicitTimeout
		}
	}

	// Проверяем auto-timeout используя общий helper
	autoTimeout := h.isAutoTimeoutEnabled(cfg)

	if !autoTimeout {
		log.Info("Auto-timeout отключён, используется минимальный таймаут")
		// hasStats=false — не знаем реальную длительность, будет SpinnerProgress
		return minTimeout, false, false, 0
	}

	// Используем уже подключённый клиент для получения статистики (M4 fix - используем helper)
	moscowTZ := getMoscowTimezone()
	// L1 fix - используем константу вместо магического числа
	timeToStatistic := time.Now().In(moscowTZ).AddDate(0, 0, -statisticPeriodDays).Format("2006-01-02T15:04:05")

	stats, err := mssqlClient.GetRestoreStats(ctx, mssql.StatsOptions{
		SrcDB:           srcDB,
		DstServer:       dstServer,
		TimeToStatistic: timeToStatistic,
	})
	if err != nil {
		log.Warn("Не удалось получить статистику восстановления, используется минимальный таймаут",
			slog.String("error", err.Error()))
		// hasStats=false — статистика недоступна, Task 7.4: SpinnerProgress
		return minTimeout, true, false, 0
	}

	if !stats.HasData || stats.MaxRestoreTimeSec == 0 {
		log.Info("Нет данных статистики, используется минимальный таймаут")
		// hasStats=false — нет данных, Task 7.4: SpinnerProgress
		return minTimeout, true, false, 0
	}

	// estimatedDuration — реальная оценка на основе MaxRestoreTimeSec (без множителя)
	estimatedDuration := time.Duration(stats.MaxRestoreTimeSec) * time.Second

	// Рассчитываем таймаут как max_restore_time * 1.7 (с запасом)
	calculatedTimeout := time.Duration(float64(stats.MaxRestoreTimeSec)*timeoutMultiplier) * time.Second
	log.Info("Auto-timeout рассчитан на основе статистики",
		slog.Int64("max_restore_time_sec", stats.MaxRestoreTimeSec),
		slog.Duration("calculated_timeout", calculatedTimeout),
		slog.Duration("estimated_duration", estimatedDuration))

	// Убеждаемся, что таймаут не меньше минимального
	if calculatedTimeout < minTimeout {
		return minTimeout, true, true, estimatedDuration
	}

	return calculatedTimeout, true, true, estimatedDuration
}

// createMSSQLClient создаёт MSSQL клиент из конфигурации.
// dstServer — целевой сервер для подключения (из DbConfig).
func (h *DbRestoreHandler) createMSSQLClient(cfg *config.Config, dstServer string) (mssql.Client, error) {
	if cfg.AppConfig == nil {
		return nil, fmt.Errorf("конфигурация приложения не загружена")
	}

	// Используем сервер из параметра (определён через determineSrcAndDstServers)
	server := dstServer
	if server == "" {
		return nil, fmt.Errorf("не указан целевой сервер для подключения")
	}

	database := "master" // база данных по умолчанию для выполнения restore
	if cfg.AppConfig.Dbrestore.Database != "" {
		database = cfg.AppConfig.Dbrestore.Database
	}

	// Пользователь
	user := cfg.AppConfig.Users.Mssql
	if user == "" {
		user = "gitops"
	}

	// Пароль
	var password string
	if cfg.SecretConfig != nil {
		password = cfg.SecretConfig.Passwords.Mssql
	}

	// Таймаут подключения
	timeout := 30 * time.Second
	if cfg.AppConfig.Dbrestore.Timeout != "" {
		if parsed, err := time.ParseDuration(cfg.AppConfig.Dbrestore.Timeout); err == nil {
			timeout = parsed
		}
	}

	opts := mssql.ClientOptions{
		Server:   server,
		Port:     1433,
		User:     user,
		Password: password,
		Database: database,
		Timeout:  timeout,
	}

	return mssql.NewClient(opts)
}

// progressMinDuration — минимальная ожидаемая длительность для показа progress bar (AC-1).
const progressMinDuration = 30 * time.Second

// createProgress создаёт progress bar для отображения прогресса восстановления.
// timeout используется для проверки порога AC-1 (30 секунд).
// hasStats указывает, доступна ли статистика для оценки длительности (Task 7.4).
// createProgress создаёт progress bar для отображения прогресса восстановления.
// estimatedDuration — реальная оценка времени выполнения (не timeout с множителем).
func (h *DbRestoreHandler) createProgress(timeout time.Duration, hasStats bool, estimatedDuration time.Duration) progress.Progress {
	// AC-1: показываем progress только для операций > 30 секунд
	// Если ожидаемое время меньше порога — возвращаем NoopProgress
	if timeout < progressMinDuration {
		return &progress.NoopProgress{}
	}

	// Task 7.4: если статистика недоступна — используем SpinnerProgress (Total=0)
	var total int64
	if hasStats && estimatedDuration > 0 {
		// Total — реальная оценка времени восстановления (не timeout с множителем)
		total = estimatedDuration.Milliseconds()
	}
	// Если hasStats=false или estimatedDuration=0, total=0 — factory вернёт SpinnerProgress

	progressOpts := progress.Options{
		Total:            total,
		Output:           os.Stderr, // важно: stderr, чтобы не ломать JSON output в stdout
		ShowETA:          true,
		ThrottleInterval: time.Second,
	}

	return progress.New(progressOpts)
}

// getUser возвращает имя пользователя для операции восстановления.
func (h *DbRestoreHandler) getUser(cfg *config.Config) string {
	if cfg.Actor != "" {
		return cfg.Actor
	}
	if cfg.AppConfig != nil && cfg.AppConfig.Users.Mssql != "" {
		return cfg.AppConfig.Users.Mssql
	}
	return "gitops"
}

// isProductionDatabase проверяет, является ли база данных production.
// КРИТИЧНО: Восстановление в production базу ЗАПРЕЩЕНО!
func isProductionDatabase(cfg *config.Config, dbName string) bool {
	if cfg == nil || cfg.DbConfig == nil {
		return false
	}
	if dbInfo, ok := cfg.DbConfig[dbName]; ok {
		return dbInfo.Prod
	}
	return false
}

// getMoscowTimezone возвращает временную зону Europe/Moscow или UTC как fallback.
// Используется для форматирования дат в запросах к MSSQL (M4 fix - устранение дублирования).
func getMoscowTimezone() *time.Location {
	tz, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return tz
}

// determineSrcAndDstServers определяет серверы источника и назначения.
// Использует FindProductionDatabase для поиска связанной production базы.
// КРИТИЧНО: Проверяет что DstServer ≠ SrcServer для защиты от перезаписи production данных.
func determineSrcAndDstServers(cfg *config.Config, dstDB string) (srcDB, srcServer, dstServer string, err error) {
	if cfg == nil || cfg.DbConfig == nil || cfg.ProjectConfig == nil {
		return "", "", "", fmt.Errorf("отсутствует конфигурация DbConfig или ProjectConfig")
	}

	// Находим production базу, связанную с целевой базой
	prodDbName := dbrestore.FindProductionDatabase(cfg.ProjectConfig, dstDB)
	if prodDbName == "" {
		return "", "", "", fmt.Errorf("не найдена production база для '%s'", dstDB)
	}

	slog.Default().Debug("Найдена production база",
		slog.String("prod_db", prodDbName),
		slog.String("dst_db", dstDB))

	// Получаем информацию о production базе (источник)
	srcDbInfo, exists := cfg.DbConfig[prodDbName]
	if !exists {
		return "", "", "", fmt.Errorf("production база '%s' не найдена в DbConfig", prodDbName)
	}

	// Получаем информацию о целевой базе
	dstDbInfo, exists := cfg.DbConfig[dstDB]
	if !exists {
		return "", "", "", fmt.Errorf("целевая база '%s' не найдена в DbConfig", dstDB)
	}

	srcDB = prodDbName
	srcServer = srcDbInfo.DbServer
	dstServer = dstDbInfo.DbServer

	if srcServer == "" {
		return "", "", "", fmt.Errorf("не указан сервер для production базы '%s'", prodDbName)
	}
	if dstServer == "" {
		return "", "", "", fmt.Errorf("не указан сервер для целевой базы '%s'", dstDB)
	}

	// КРИТИЧНО: Проверка что восстановление идёт на другой сервер
	// Защита от случайной перезаписи production данных на том же сервере
	if srcServer == dstServer {
		return "", "", "", fmt.Errorf("восстановление на тот же сервер '%s' запрещено: источник и назначение должны быть на разных серверах", srcServer)
	}

	return srcDB, srcServer, dstServer, nil
}

// buildPlan создаёт план операций для предпросмотра.
// Используется в dry-run, plan-only и verbose режимах.
// Story 7.3: извлечено из executeDryRun для переиспользования.
func (h *DbRestoreHandler) buildPlan(cfg *config.Config, srcDB, srcServer, dstServer string) *output.DryRunPlan {
	timeout := h.getDryRunTimeout()

	steps := []output.PlanStep{
		{
			Order:     1,
			Operation: "Проверка production флага",
			Parameters: map[string]any{
				"database":      cfg.InfobaseName,
				"is_production": false,
			},
			ExpectedChanges: []string{"Нет изменений — только валидация"},
		},
		{
			Order:     2,
			Operation: "Подключение к MSSQL серверу",
			Parameters: map[string]any{
				"server":   dstServer,
				"database": "master",
			},
			ExpectedChanges: []string{"Установка соединения с сервером"},
		},
		{
			Order:     3,
			Operation: "Восстановление базы данных",
			Parameters: map[string]any{
				"src_server":   srcServer,
				"src_db":       srcDB,
				"dst_server":   dstServer,
				"dst_db":       cfg.InfobaseName,
				"timeout":      timeout.String(),
				"auto_timeout": h.getDryRunAutoTimeoutInfo(cfg),
			},
			ExpectedChanges: []string{
				fmt.Sprintf("База %s будет восстановлена из %s/%s", cfg.InfobaseName, srcServer, srcDB),
				"Все данные в целевой базе будут перезаписаны",
			},
		},
	}

	return dryrun.BuildPlanWithSummary(
		constants.ActNRDbrestore,
		steps,
		fmt.Sprintf("Восстановление %s/%s → %s/%s", srcServer, srcDB, dstServer, cfg.InfobaseName),
	)
}

// executeDryRun выполняет dry-run режим для команды nr-dbrestore.
// AC-1: Возвращает план действий БЕЗ выполнения.
// AC-2: План содержит операции, параметры, ожидаемые изменения.
// AC-8: НЕ выполняются SQL запросы.
func (h *DbRestoreHandler) executeDryRun(
	cfg *config.Config,
	srcDB, srcServer, dstServer string,
	format, traceID string,
	start time.Time,
	log *slog.Logger,
) error {
	log.Info("Построение плана восстановления (dry-run)")
	plan := h.buildPlan(cfg, srcDB, srcServer, dstServer)
	return output.WriteDryRunResult(os.Stdout, format, constants.ActNRDbrestore, traceID, constants.APIVersion, start, plan)
}

// getDryRunTimeout возвращает таймаут для dry-run плана.
// В dry-run режиме не подключаемся к БД, поэтому используем:
// 1. BR_TIMEOUT_MIN если задан
// 2. minTimeout (5 минут) как fallback
func (h *DbRestoreHandler) getDryRunTimeout() time.Duration {
	if timeoutMinStr := os.Getenv("BR_TIMEOUT_MIN"); timeoutMinStr != "" {
		if timeoutMin, err := strconv.Atoi(timeoutMinStr); err == nil && timeoutMin > 0 {
			return time.Duration(timeoutMin) * time.Minute
		}
	}
	return minTimeout
}

// getDryRunAutoTimeoutInfo возвращает информацию об auto-timeout для dry-run плана.
func (h *DbRestoreHandler) getDryRunAutoTimeoutInfo(cfg *config.Config) string {
	if timeoutMinStr := os.Getenv("BR_TIMEOUT_MIN"); timeoutMinStr != "" {
		return "отключён (BR_TIMEOUT_MIN задан явно)"
	}
	if h.isAutoTimeoutEnabled(cfg) {
		return "включён (будет рассчитан по статистике)"
	}
	return "отключён (BR_AUTO_TIMEOUT=false)"
}

// isAutoTimeoutEnabled проверяет включён ли auto-timeout.
// Логика: env переменная имеет приоритет над AppConfig.
// Допустимые значения для включения: "true", "1".
// По умолчанию (если не задано) — включён.
func (h *DbRestoreHandler) isAutoTimeoutEnabled(cfg *config.Config) bool {
	autoTimeoutStr := os.Getenv("BR_AUTO_TIMEOUT")
	if autoTimeoutStr != "" {
		return autoTimeoutStr == "true" || autoTimeoutStr == "1"
	}
	// Env переменная не задана — проверяем AppConfig, иначе true по умолчанию
	if cfg != nil && cfg.AppConfig != nil {
		return cfg.AppConfig.Dbrestore.Autotimeout
	}
	return true // по умолчанию включён
}

// writePlanOnlyResult и writeDryRunResult перенесены в output.WritePlanOnlyResult/WriteDryRunResult (CR-7.3 #2, #3).
