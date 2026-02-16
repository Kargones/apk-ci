// Package racutil предоставляет общие утилиты для создания RAC клиента
// из конфигурации приложения. Вынесен из отдельных handler-пакетов для
// устранения дублирования кода (бывший TODO H-2).
package racutil

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/onec/rac"
	"github.com/Kargones/apk-ci/internal/config"
)

// NewClient создаёт RAC клиент из конфигурации.
// Логика определения сервера, порта, таймаута и credentials
// извлечена из handler-пакетов servicemodestatushandler,
// servicemodeenablehandler, servicemodedisablehandler и forcedisconnecthandler.
func NewClient(cfg *config.Config) (rac.Client, error) {
	if cfg.AppConfig == nil {
		return nil, fmt.Errorf("конфигурация приложения не загружена")
	}

	// Получение сервера 1C для информационной базы
	server := cfg.GetOneServer(cfg.InfobaseName)
	if server == "" {
		// Fallback на RacConfig если DbConfig не содержит запись
		if cfg.RacConfig != nil && cfg.RacConfig.RacServer != "" {
			server = cfg.RacConfig.RacServer
		} else {
			return nil, fmt.Errorf("не удалось определить сервер для информационной базы '%s'", cfg.InfobaseName)
		}
	}

	port := strconv.Itoa(cfg.AppConfig.Rac.Port)
	if port == "0" {
		port = "1545"
	}

	timeout := time.Duration(cfg.AppConfig.Rac.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	// Предупреждение о потенциально слишком большом timeout (может заблокировать pipeline)
	if timeout > 5*time.Minute {
		slog.Default().Warn("RAC: timeout превышает 5 минут — возможна длительная блокировка при ошибках",
			slog.Duration("timeout", timeout))
	}

	opts := rac.ClientOptions{
		RACPath:      cfg.AppConfig.Paths.Rac,
		Server:       server,
		Port:         port,
		Timeout:      timeout,
		ClusterUser:  cfg.AppConfig.Users.Rac,
		ClusterPass:  "",
		InfobaseUser: cfg.AppConfig.Users.Db,
		InfobasePass: "",
	}

	// Пароли из SecretConfig
	if cfg.SecretConfig != nil {
		opts.ClusterPass = cfg.SecretConfig.Passwords.Rac
		opts.InfobasePass = cfg.SecretConfig.Passwords.Db
	}

	// Диагностика: предупреждение при отсутствии credentials (может усложнить отладку RAC-ошибок)
	if opts.ClusterUser != "" && opts.ClusterPass == "" {
		slog.Default().Warn("RAC: указан пользователь кластера, но пароль пуст — возможны ошибки аутентификации")
	}
	if opts.InfobaseUser != "" && opts.InfobasePass == "" {
		slog.Default().Warn("RAC: указан пользователь базы, но пароль пуст — возможны ошибки аутентификации")
	}

	return rac.NewClient(opts)
}
