package config

import (
	"fmt"
	"log/slog"

	"github.com/ilyakaznacheev/cleanenv"
)

// ResilienceConfig содержит настройки для resilience паттернов.

// ImplementationsConfig содержит настройки выбора реализаций операций.
// Позволяет переключаться между различными инструментами (1cv8/ibcmd/native)
// без изменения кода приложения.
type ImplementationsConfig struct {
	// ConfigExport определяет инструмент для выгрузки конфигурации.
	// Допустимые значения: "1cv8" (default), "ibcmd", "native"
	ConfigExport string `yaml:"config_export" env:"BR_IMPL_CONFIG_EXPORT" env-default:"1cv8"`

	// DBCreate определяет инструмент для создания базы данных.
	// Допустимые значения: "1cv8" (default), "ibcmd"
	DBCreate string `yaml:"db_create" env:"BR_IMPL_DB_CREATE" env-default:"1cv8"`
}
// Validate проверяет корректность значений ImplementationsConfig.
// Возвращает ошибку если значения не соответствуют допустимым.
func (c *ImplementationsConfig) Validate() error {
	// Применяем defaults для пустых значений
	if c.ConfigExport == "" {
		c.ConfigExport = "1cv8"
	}
	if c.DBCreate == "" {
		c.DBCreate = "1cv8"
	}

	validConfigExport := map[string]bool{"1cv8": true, "ibcmd": true, "native": true}
	validDBCreate := map[string]bool{"1cv8": true, "ibcmd": true}

	if !validConfigExport[c.ConfigExport] {
		return fmt.Errorf("недопустимое значение ConfigExport: %q, допустимые: 1cv8, ibcmd, native", c.ConfigExport)
	}
	if !validDBCreate[c.DBCreate] {
		return fmt.Errorf("недопустимое значение DBCreate: %q, допустимые: 1cv8, ibcmd", c.DBCreate)
	}
	return nil
}
// loadImplementationsConfig загружает конфигурацию реализаций из AppConfig, переменных окружения или устанавливает значения по умолчанию
func loadImplementationsConfig(l *slog.Logger, cfg *Config) (*ImplementationsConfig, error) {
	// Проверяем, есть ли конфигурация в AppConfig
	if cfg.AppConfig != nil && (cfg.AppConfig.Implementations != ImplementationsConfig{}) {
		implConfig := &cfg.AppConfig.Implementations
		// Применяем переопределения из переменных окружения
		if err := cleanenv.ReadEnv(implConfig); err != nil {
			l.Warn("Ошибка загрузки Implementations конфигурации из переменных окружения",
				slog.String("error", err.Error()),
			)
		}
		l.Info("Implementations конфигурация загружена из AppConfig",
			slog.String("config_export", implConfig.ConfigExport),
			slog.String("db_create", implConfig.DBCreate),
		)
		return implConfig, nil
	}

	// Если конфигурация не найдена, используем значения по умолчанию
	implConfig := getDefaultImplementationsConfig()

	// Применяем переопределения из переменных окружения
	if err := cleanenv.ReadEnv(implConfig); err != nil {
		l.Warn("Ошибка загрузки Implementations конфигурации из переменных окружения",
			slog.String("error", err.Error()),
		)
	}

	l.Debug("Implementations конфигурация: используются значения по умолчанию",
		slog.String("config_export", implConfig.ConfigExport),
		slog.String("db_create", implConfig.DBCreate),
	)

	return implConfig, nil
}
// getDefaultImplementationsConfig возвращает конфигурацию реализаций по умолчанию
func getDefaultImplementationsConfig() *ImplementationsConfig {
	return &ImplementationsConfig{
		ConfigExport: "1cv8",
		DBCreate:     "1cv8",
	}
}
