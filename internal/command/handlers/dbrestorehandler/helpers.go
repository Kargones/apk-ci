package dbrestorehandler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/mssql"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/entity/dbrestore"
	"github.com/Kargones/apk-ci/internal/pkg/progress"
)

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
