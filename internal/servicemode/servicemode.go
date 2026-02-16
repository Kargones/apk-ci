// Package servicemode предоставляет функциональность для управления режимом сервиса
package servicemode

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/rac"
)

// Manager представляет интерфейс для управления сервисным режимом информационных баз.
// Предоставляет методы для включения, отключения и получения статуса сервисного режима.
type Manager interface {
	EnableServiceMode(ctx context.Context, infobaseName string, terminateSessions bool) error
	DisableServiceMode(ctx context.Context, infobaseName string) error
	GetServiceModeStatus(ctx context.Context, infobaseName string) (*rac.ServiceModeStatus, error)
}

// RacClientInterface представляет интерфейс для RAC клиента.
// Используется для абстракции RAC операций и возможности тестирования с моками.
type RacClientInterface interface {
	GetClusterUUID(ctx context.Context) (string, error)
	GetInfobaseUUID(ctx context.Context, clusterUUID, infobaseName string) (string, error)
	EnableServiceMode(ctx context.Context, clusterUUID, infobaseUUID string, terminateSessions bool) error
	DisableServiceMode(ctx context.Context, clusterUUID, infobaseUUID string) error
	GetServiceModeStatus(ctx context.Context, clusterUUID, infobaseUUID string) (*rac.ServiceModeStatus, error)
}

// Logger представляет интерфейс для записи логов различных уровней.
// Используется для унифицированного логирования в компонентах управления сервисным режимом.
type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
}

// SlogLogger представляет адаптер для интеграции slog.Logger с интерфейсом Logger.
// Обеспечивает совместимость между стандартным slog и пользовательским интерфейсом логирования.
type SlogLogger struct {
	Logger *slog.Logger
}

// Info записывает информационное сообщение в лог с дополнительными параметрами.
// Параметры:
//   - msg: текст сообщения
//   - args: дополнительные параметры для структурированного логирования
func (s *SlogLogger) Info(msg string, args ...any) {
	s.Logger.Info(msg, args...)
}

// Warn записывает предупреждающее сообщение в лог с дополнительными параметрами.
// Параметры:
//   - msg: текст предупреждения
//   - args: дополнительные параметры для структурированного логирования
func (s *SlogLogger) Warn(msg string, args ...any) {
	s.Logger.Warn(msg, args...)
}

// Error записывает сообщение об ошибке в лог с дополнительными параметрами.
// Параметры:
//   - msg: текст ошибки
//   - args: дополнительные параметры для структурированного логирования
func (s *SlogLogger) Error(msg string, args ...any) {
	s.Logger.Error(msg, args...)
}

// Debug записывает отладочное сообщение в лог с дополнительными параметрами.
// Параметры:
//   - msg: текст отладочного сообщения
//   - args: дополнительные параметры для структурированного логирования
func (s *SlogLogger) Debug(msg string, args ...any) {
	s.Logger.Debug(msg, args...)
}

// RacConfig представляет конфигурацию для подключения к RAC (Remote Administration Console).
// Содержит все необходимые параметры для установки соединения и аутентификации.
type RacConfig struct {
	RacPath     string
	RacServer   string
	RacPort     int
	RacUser     string
	RacPassword string
	DbUser      string
	DbPassword  string
	RacTimeout  time.Duration
	RacRetries  int
}

// Client представляет клиент для управления сервисным режимом через RAC.
// Инкапсулирует RAC клиент и предоставляет высокоуровневые методы управления.
type Client struct {
	config    *RacConfig
	racClient RacClientInterface
	logger    Logger
}

// NewClient создает новый экземпляр клиента для управления сервисным режимом.
// Инициализирует RAC клиент с переданной конфигурацией и логгером.
// Параметры:
//   - racConfig: конфигурация подключения к RAC
//   - logger: интерфейс для логирования операций
//
// Возвращает:
//   - *Client: новый экземпляр клиента для управления сервисным режимом
func NewClient(racConfig RacConfig, logger Logger) *Client {
	// Логируем параметры конфигурации RAC клиента, если logger не nil
	if logger != nil {
		logger.Debug("Creating new RAC client for service mode",
			"racPath", racConfig.RacPath,
			"racServer", racConfig.RacServer,
			"racPort", racConfig.RacPort,
			"racUser", racConfig.RacUser,
			"racTimeout", racConfig.RacTimeout,
			"racRetries", racConfig.RacRetries,
			"hasDbUser", racConfig.DbUser != "",
			"hasDbPassword", racConfig.DbPassword != "",
			"hasRacPassword", racConfig.RacPassword != "")
	}

	// Преобразуем Logger в *slog.Logger для RAC клиента
	var slogLogger *slog.Logger
	if logger != nil {
		if sl, ok := logger.(*SlogLogger); ok {
			slogLogger = sl.Logger
		} else {
			// Если передан не SlogLogger, создаем дефолтный
			slogLogger = slog.Default()
		}
	} else {
		// Если logger равен nil, создаем дефолтный
		slogLogger = slog.Default()
	}

	racClient := rac.NewClient(
		racConfig.RacPath,
		racConfig.RacServer,
		racConfig.RacPort,
		racConfig.RacUser,
		racConfig.RacPassword,
		racConfig.DbUser,
		racConfig.DbPassword,
		racConfig.RacTimeout,
		racConfig.RacRetries,
		slogLogger,
	)

	return &Client{
		config:    &racConfig,
		racClient: racClient,
		logger:    logger,
	}
}

// NewClientWithRacClient создает новый экземпляр клиента с переданным RAC клиентом.
// Используется для тестирования с мок-объектами.
// Параметры:
//   - racConfig: конфигурация подключения к RAC
//   - racClient: интерфейс RAC клиента (может быть мок-объектом)
//   - logger: интерфейс для логирования операций
//
// Возвращает:
//   - *Client: новый экземпляр клиента для управления сервисным режимом
func NewClientWithRacClient(racConfig RacConfig, racClient RacClientInterface, logger Logger) *Client {
	return &Client{
		config:    &racConfig,
		racClient: racClient,
		logger:    logger,
	}
}

// ManageServiceMode выполняет операции управления сервисным режимом на основе переданного действия.
// Поддерживает операции включения, отключения и получения статуса сервисного режима.
// Параметры:
//   - ctx: контекст выполнения операции
//   - action: действие для выполнения ("enable", "disable", "status")
//   - infobaseName: имя информационной базы
//   - terminateSessions: флаг принудительного завершения сессий (используется только для "enable")
//   - cfg: конфигурация приложения
//   - logger: интерфейс для логирования
//
// Возвращает:
//   - error: ошибка выполнения операции или nil при успехе
func ManageServiceMode(ctx context.Context, action, infobaseName string, terminateSessions bool, cfg *config.Config, logger Logger) error {
	// Сначала проверяем валидность action
	switch action {
	case "enable", "disable", "status":
		// Валидные действия, продолжаем
	default:
		logger.Error("Unknown service mode action", "action", action)
		return fmt.Errorf("unknown action: %s", action)
	}

	config, err := LoadServiceModeConfigForDb(infobaseName, cfg)
	if err != nil {
		return err
	}

	client := NewClient(config, logger)

	switch action {
	case "enable":
		logger.Debug("Executing enable service mode action")
		err := client.EnableServiceMode(ctx, infobaseName, terminateSessions)
		if err != nil {
			logger.Error("Failed to enable service mode",
				"error", err,
				"infobaseName", infobaseName,
				"terminateSessions", terminateSessions)
			return err
		}
		logger.Info("Service mode enabled successfully",
			"infobaseName", infobaseName,
			"terminateSessions", terminateSessions)
		return nil
	case "disable":
		logger.Debug("Executing disable service mode action")
		err := client.DisableServiceMode(ctx, infobaseName)
		if err != nil {
			logger.Error("Failed to disable service mode",
				"error", err,
				"infobaseName", infobaseName)
			return err
		}
		logger.Info("Service mode disabled successfully", "infobaseName", infobaseName)
		return nil
	case "status":
		logger.Debug("Executing get service mode status action")
		status, err := client.GetServiceModeStatus(ctx, infobaseName)
		if err != nil {
			logger.Error("Failed to get service mode status",
				"error", err,
				"infobaseName", infobaseName)
			return err
		}
		logger.Info("Service mode status retrieved successfully",
			"infobaseName", infobaseName,
			"enabled", status.Enabled,
			"active_sessions", status.ActiveSessions)
		return nil
	}
	
	// Этот код никогда не должен выполняться, так как все валидные действия обработаны выше
	return fmt.Errorf("unexpected error in action handling")
}

// LoadServiceModeConfigForDb загружает и преобразует конфигурацию RAC для указанной базы данных.
// Извлекает параметры подключения из общей конфигурации приложения.
// Параметры:
//   - dbName: имя базы данных для загрузки конфигурации
//   - cfg: общая конфигурация приложения
//
// Возвращает:
//   - RacConfig: структура с параметрами подключения к RAC
//   - error: ошибка загрузки конфигурации или nil при успехе
func LoadServiceModeConfigForDb(dbName string, cfg *config.Config) (RacConfig, error) {
	if cfg == nil {
		return RacConfig{}, fmt.Errorf("config is nil")
	}
	
	smc, err := cfg.LoadServiceModeConfig(dbName)
	if err != nil {
		return RacConfig{}, err
	}

	return RacConfig{
		RacPath:     smc.RacPath,
		RacServer:   smc.RacServer,
		RacPort:     smc.RacPort,
		RacUser:     smc.RacUser,
		RacPassword: smc.RacPassword,
		DbUser:      smc.DbUser,
		DbPassword:  smc.DbPassword,
		RacTimeout:  smc.RacTimeout,
		RacRetries:  smc.RacRetries,
	}, nil
}

// EnableServiceMode включает сервисный режим для указанной информационной базы.
// Получает UUID кластера и информационной базы, затем включает блокировку через RAC клиент.
// Параметры:
//   - ctx: контекст выполнения операции
//   - infobaseName: имя информационной базы
//   - terminateSessions: флаг принудительного завершения активных сессий
//
// Возвращает:
//   - error: ошибка выполнения операции или nil при успехе
func (c *Client) EnableServiceMode(ctx context.Context, infobaseName string, terminateSessions bool) error {
	if infobaseName == "" {
		return fmt.Errorf("infobase name cannot be empty")
	}

	c.logger.Debug("Enabling service mode",
		"infobase", infobaseName,
		"terminateSessions", terminateSessions)

	// Получаем UUID кластера
	c.logger.Debug("Getting cluster UUID for service mode operation")
	clusterUUID, err := c.racClient.GetClusterUUID(ctx)
	if err != nil {
		c.logger.Error("Failed to get cluster UUID for service mode",
			"error", err,
			"infobase", infobaseName)
		return fmt.Errorf("failed to get cluster UUID: %w", err)
	}
	c.logger.Debug("Successfully obtained cluster UUID", "clusterUUID", clusterUUID)

	// Получаем UUID информационной базы
	c.logger.Debug("Getting infobase UUID", "infobaseName", infobaseName, "clusterUUID", clusterUUID)
	infobaseUUID, err := c.racClient.GetInfobaseUUID(ctx, clusterUUID, infobaseName)
	if err != nil {
		c.logger.Error("Failed to get infobase UUID for service mode",
			"error", err,
			"infobase", infobaseName,
			"clusterUUID", clusterUUID)
		return fmt.Errorf("failed to get infobase UUID: %w", err)
	}
	c.logger.Debug("Successfully obtained infobase UUID", "infobaseUUID", infobaseUUID)

	// Включаем сервисный режим
	c.logger.Debug("Enabling service mode with RAC client",
		"clusterUUID", clusterUUID,
		"infobaseUUID", infobaseUUID,
		"terminateSessions", terminateSessions)
	return c.racClient.EnableServiceMode(ctx, clusterUUID, infobaseUUID, terminateSessions)
}

// DisableServiceMode отключает сервисный режим для указанной информационной базы.
// Получает UUID кластера и информационной базы, затем снимает блокировку через RAC клиент.
// Параметры:
//   - ctx: контекст выполнения операции
//   - infobaseName: имя информационной базы
//
// Возвращает:
//   - error: ошибка выполнения операции или nil при успехе
func (c *Client) DisableServiceMode(ctx context.Context, infobaseName string) error {
	if infobaseName == "" {
		return fmt.Errorf("infobase name cannot be empty")
	}

	c.logger.Debug("Disabling service mode", "infobase", infobaseName)

	// Получаем UUID кластера
	c.logger.Debug("Getting cluster UUID for service mode disable operation")
	clusterUUID, err := c.racClient.GetClusterUUID(ctx)
	if err != nil {
		c.logger.Error("Failed to get cluster UUID for service mode disable",
			"error", err,
			"infobase", infobaseName)
		return fmt.Errorf("failed to get cluster UUID: %w", err)
	}
	c.logger.Debug("Successfully obtained cluster UUID for disable", "clusterUUID", clusterUUID)

	// Получаем UUID информационной базы
	c.logger.Debug("Getting infobase UUID for disable", "infobaseName", infobaseName, "clusterUUID", clusterUUID)
	infobaseUUID, err := c.racClient.GetInfobaseUUID(ctx, clusterUUID, infobaseName)
	if err != nil {
		c.logger.Error("Failed to get infobase UUID for service mode disable",
			"error", err,
			"infobase", infobaseName,
			"clusterUUID", clusterUUID)
		return fmt.Errorf("failed to get infobase UUID: %w", err)
	}
	c.logger.Debug("Successfully obtained infobase UUID for disable", "infobaseUUID", infobaseUUID)

	// Отключаем сервисный режим
	c.logger.Debug("Disabling service mode with RAC client",
		"clusterUUID", clusterUUID,
		"infobaseUUID", infobaseUUID)
	err = c.racClient.DisableServiceMode(ctx, clusterUUID, infobaseUUID)
	if err != nil {
		c.logger.Error("Failed to disable service mode with RAC client",
			"error", err,
			"clusterUUID", clusterUUID,
			"infobaseUUID", infobaseUUID)
		return err
	}
	c.logger.Debug("Service mode disabled successfully with RAC client")
	return nil
}

// GetServiceModeStatus получает текущий статус сервисного режима для указанной информационной базы.
// Получает UUID кластера и информационной базы, затем запрашивает статус через RAC клиент.
// Параметры:
//   - ctx: контекст выполнения операции
//   - infobaseName: имя информационной базы
//
// Возвращает:
//   - *rac.ServiceModeStatus: структура с информацией о статусе сервисного режима
//   - error: ошибка выполнения операции или nil при успехе
func (c *Client) GetServiceModeStatus(ctx context.Context, infobaseName string) (*rac.ServiceModeStatus, error) {
	if infobaseName == "" {
		return nil, fmt.Errorf("infobase name cannot be empty")
	}

	c.logger.Debug("Getting service mode status", "infobase", infobaseName)

	// Получаем UUID кластера
	c.logger.Debug("Getting cluster UUID for service mode status operation")
	clusterUUID, err := c.racClient.GetClusterUUID(ctx)
	if err != nil {
		c.logger.Error("Failed to get cluster UUID for service mode status",
			"error", err,
			"infobase", infobaseName)
		return nil, fmt.Errorf("failed to get cluster UUID: %w", err)
	}
	c.logger.Debug("Successfully obtained cluster UUID for status", "clusterUUID", clusterUUID)

	// Получаем UUID информационной базы
	c.logger.Debug("Getting infobase UUID for status", "infobaseName", infobaseName, "clusterUUID", clusterUUID)
	infobaseUUID, err := c.racClient.GetInfobaseUUID(ctx, clusterUUID, infobaseName)
	if err != nil {
		c.logger.Error("Failed to get infobase UUID for service mode status",
			"error", err,
			"infobase", infobaseName,
			"clusterUUID", clusterUUID)
		return nil, fmt.Errorf("failed to get infobase UUID: %w", err)
	}
	c.logger.Debug("Successfully obtained infobase UUID for status", "infobaseUUID", infobaseUUID)

	// Получаем статус сервисного режима
	c.logger.Debug("Getting service mode status with RAC client",
		"clusterUUID", clusterUUID,
		"infobaseUUID", infobaseUUID)
	status, err := c.racClient.GetServiceModeStatus(ctx, clusterUUID, infobaseUUID)
	if err != nil {
		c.logger.Error("Failed to get service mode status with RAC client",
			"error", err,
			"clusterUUID", clusterUUID,
			"infobaseUUID", infobaseUUID)
		return nil, err
	}
	c.logger.Debug("Service mode status retrieved successfully with RAC client",
		"enabled", status.Enabled,
		"activeSessions", status.ActiveSessions)
	return status, nil
}
