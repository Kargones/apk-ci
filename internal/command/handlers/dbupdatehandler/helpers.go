package dbupdatehandler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/onec"
	"github.com/Kargones/apk-ci/internal/adapter/onec/rac"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/pkg/progress"
)

// buildConnectString строит строку подключения к информационной базе.
//
// БЕЗОПАСНОСТЬ: Пароль включается в строку подключения, но runner.Runner
// автоматически использует файл параметров (@) и маскирует пароли в логах.
// Эта строка НЕ должна логироваться напрямую.
func (h *DbUpdateHandler) buildConnectString(dbInfo *config.DatabaseInfo, cfg *config.Config) string {
	// Формат: /S server\base /N user /P pass
	server := dbInfo.GetServer()

	connectString := fmt.Sprintf("/S %s\\%s", server, cfg.InfobaseName)

	// Добавляем пользователя (из AppConfig.Users.Db)
	user := ""
	if cfg.AppConfig != nil && cfg.AppConfig.Users.Db != "" {
		user = cfg.AppConfig.Users.Db
	}
	if user != "" {
		connectString += fmt.Sprintf(" /N %s", user)
	}

	// Добавляем пароль (из SecretConfig.Passwords.Db)
	pass := ""
	if cfg.SecretConfig != nil && cfg.SecretConfig.Passwords.Db != "" {
		pass = cfg.SecretConfig.Passwords.Db
	}
	if pass != "" {
		connectString += fmt.Sprintf(" /P %s", pass)
	}

	return connectString
}

// getTimeout возвращает таймаут для операции обновления.
func (h *DbUpdateHandler) getTimeout() time.Duration {
	// Проверяем явный таймаут через BR_TIMEOUT_MIN
	if timeoutMinStr := os.Getenv("BR_TIMEOUT_MIN"); timeoutMinStr != "" {
		if timeoutMin, err := strconv.Atoi(timeoutMinStr); err == nil && timeoutMin > 0 {
			return time.Duration(timeoutMin) * time.Minute
		}
	}
	return defaultTimeout
}

// getOrCreateOneCClient возвращает существующий или создаёт новый 1C клиент.
func (h *DbUpdateHandler) getOrCreateOneCClient(cfg *config.Config) onec.DatabaseUpdater {
	if h.oneCClient != nil {
		return h.oneCClient
	}
	return onec.NewUpdater(cfg.AppConfig.Paths.Bin1cv8, cfg.WorkDir, cfg.TmpDir)
}

// getOrCreateRacClient возвращает существующий или создаёт новый RAC клиент.
// M3 fix: dbInfo передаётся как параметр вместо повторного вызова GetDatabaseInfo.
// M3-v2 fix: унифицирована логика fallback сервера с buildConnectString.
// H2 fix: корректная обработка nil log и nil конфигураций.
func (h *DbUpdateHandler) getOrCreateRacClient(cfg *config.Config, dbInfo *config.DatabaseInfo, log *slog.Logger) rac.Client {
	if h.racClient != nil {
		return h.racClient
	}

	// H2 fix: используем default logger если не передан
	if log == nil {
		log = slog.Default()
	}

	server := dbInfo.GetServer()
	if server == "" {
		log.Warn("Сервер 1C не указан в конфигурации БД (ни OneServer, ни DbServer)")
		return nil
	}

	// H2 fix: проверяем nil AppConfig перед доступом к Paths
	if cfg.AppConfig == nil {
		log.Warn("AppConfig не указан в конфигурации")
		return nil
	}

	// Получаем путь к rac
	racPath := cfg.AppConfig.Paths.Rac
	if racPath == "" {
		log.Warn("Путь к RAC не указан в конфигурации")
		return nil
	}

	// Получаем учётные данные
	// H2 fix: AppConfig уже проверен на nil выше
	clusterUser := cfg.AppConfig.Users.Rac
	infobaseUser := cfg.AppConfig.Users.Db

	clusterPass := ""
	infobasePass := ""
	if cfg.SecretConfig != nil {
		clusterPass = cfg.SecretConfig.Passwords.Rac
		infobasePass = cfg.SecretConfig.Passwords.Db
	}

	client, err := rac.NewClient(rac.ClientOptions{
		RACPath:      racPath,
		Server:       server,
		ClusterUser:  clusterUser,
		ClusterPass:  clusterPass,
		InfobaseUser: infobaseUser,
		InfobasePass: infobasePass,
		Logger:       log,
	})
	if err != nil {
		log.Warn("Не удалось создать RAC клиент", slog.String("error", err.Error()))
		return nil
	}

	return client
}

// enableServiceModeIfNeeded проверяет и включает сервисный режим если нужно.
// Возвращает true если режим был уже включён.
func (h *DbUpdateHandler) enableServiceModeIfNeeded(ctx context.Context, cfg *config.Config, racClient rac.Client, log *slog.Logger) (bool, error) {
	// Получаем информацию о кластере
	clusterInfo, err := racClient.GetClusterInfo(ctx)
	if err != nil {
		return false, fmt.Errorf("не удалось получить информацию о кластере: %w", err)
	}

	// Получаем информацию о базе
	infobaseInfo, err := racClient.GetInfobaseInfo(ctx, clusterInfo.UUID, cfg.InfobaseName)
	if err != nil {
		return false, fmt.Errorf("не удалось получить информацию о базе: %w", err)
	}

	// Проверяем текущий статус
	status, err := racClient.GetServiceModeStatus(ctx, clusterInfo.UUID, infobaseInfo.UUID)
	if err != nil {
		return false, fmt.Errorf("не удалось проверить статус сервисного режима: %w", err)
	}

	if status.Enabled {
		log.Info("Сервисный режим уже включён")
		return true, nil
	}

	// Включаем сервисный режим
	log.Info("Включаем сервисный режим (auto-deps)")
	if err := racClient.EnableServiceMode(ctx, clusterInfo.UUID, infobaseInfo.UUID, false); err != nil {
		return false, fmt.Errorf("не удалось включить сервисный режим: %w", err)
	}

	return false, nil
}

// disableServiceModeIfNeeded отключает сервисный режим.
func (h *DbUpdateHandler) disableServiceModeIfNeeded(ctx context.Context, cfg *config.Config, racClient rac.Client, log *slog.Logger) {
	// Получаем информацию о кластере
	clusterInfo, err := racClient.GetClusterInfo(ctx)
	if err != nil {
		log.Error("Не удалось получить информацию о кластере для отключения service mode", slog.String("error", err.Error()))
		return
	}

	// Получаем информацию о базе
	infobaseInfo, err := racClient.GetInfobaseInfo(ctx, clusterInfo.UUID, cfg.InfobaseName)
	if err != nil {
		log.Error("Не удалось получить информацию о базе для отключения service mode", slog.String("error", err.Error()))
		return
	}

	// Отключаем сервисный режим
	log.Info("Отключаем сервисный режим (auto-deps)")
	if err := racClient.DisableServiceMode(ctx, clusterInfo.UUID, infobaseInfo.UUID); err != nil {
		log.Error("Не удалось отключить сервисный режим", slog.String("error", err.Error()))
	}
}

// createProgress создаёт progress bar для отображения прогресса обновления.
// M-4 fix: используем общий helper progress.NewIndeterminate().
func (h *DbUpdateHandler) createProgress() progress.Progress {
	return progress.NewIndeterminate()
}

// valueOrNone возвращает значение или "нет" если пустое.
func valueOrNone(value string) string {
	if value == "" {
		return "нет"
	}
	return value
}

// getServerFromDbInfo возвращает сервер из DatabaseInfo.
func getServerFromDbInfo(dbInfo *config.DatabaseInfo) string {
	return dbInfo.GetServer()
}
