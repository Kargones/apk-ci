package config

import (
	"log/slog"

	"github.com/ilyakaznacheev/cleanenv"
)

// LoggingConfig содержит настройки для логирования.
//
// TODO: Dual source of truth — LoggingConfig здесь и logging.Config
// в internal/pkg/logging/config.go дублируют поля. При добавлении новых опций нужно
// менять оба места и синхронизировать defaults. Рефакторинг: использовать одну структуру.
type LoggingConfig struct {
	// Level - уровень логирования (debug, info, warn, error)
	Level string `yaml:"level" env:"BR_LOG_LEVEL" env-default:"info"`

	// Format - формат логов (json, text)
	Format string `yaml:"format" env:"BR_LOG_FORMAT" env-default:"text"`

	// Output - вывод логов (stdout, stderr, file)
	Output string `yaml:"output" env:"BR_LOG_OUTPUT" env-default:"stderr"`

	// FilePath - путь к файлу логов (если output=file)
	FilePath string `yaml:"filePath" env:"BR_LOG_FILE_PATH"`

	// MaxSize - максимальный размер файла лога в MB
	MaxSize int `yaml:"maxSize" env:"BR_LOG_MAX_SIZE" env-default:"100"`

	// MaxBackups - максимальное количество backup файлов
	MaxBackups int `yaml:"maxBackups" env:"BR_LOG_MAX_BACKUPS" env-default:"3"`

	// MaxAge - максимальный возраст backup файлов в днях
	MaxAge int `yaml:"maxAge" env:"BR_LOG_MAX_AGE" env-default:"7"`

	// Compress - сжимать ли backup файлы
	// TODO: bool с env-default:"true" — в YAML yaml:"compress" false будет
	// перезаписан env-default при cleanenv.ReadEnv. Поведение корректно только при чтении
	// из env. Для YAML-source используется getDefaultLoggingConfig() где Compress=true.
	Compress bool `yaml:"compress" env:"BR_LOG_COMPRESS" env-default:"true"`
}
// loadLoggingConfig загружает конфигурацию логирования из AppConfig, переменных окружения или устанавливает значения по умолчанию.
// Переменные окружения BR_LOG_* переопределяют значения из AppConfig (AC4).
func loadLoggingConfig(l *slog.Logger, cfg *Config) (*LoggingConfig, error) {
	// Проверяем, есть ли конфигурация в AppConfig
	if cfg.AppConfig != nil && (cfg.AppConfig.Logging != LoggingConfig{}) {
		loggingConfig := &cfg.AppConfig.Logging
		// Применяем env override для AppConfig (симметрично с loadImplementationsConfig)
		if err := cleanenv.ReadEnv(loggingConfig); err != nil {
			l.Warn("Ошибка загрузки Logging конфигурации из переменных окружения",
				slog.String("error", err.Error()),
			)
		}
		l.Info("Logging конфигурация загружена из AppConfig",
			slog.String("level", loggingConfig.Level),
			slog.String("format", loggingConfig.Format),
		)
		return loggingConfig, nil
	}

	loggingConfig := getDefaultLoggingConfig()

	if err := cleanenv.ReadEnv(loggingConfig); err != nil {
		l.Warn("Ошибка загрузки Logging конфигурации из переменных окружения",
			slog.String("error", err.Error()),
		)
	}

	l.Debug("Logging конфигурация: используются значения по умолчанию",
		slog.String("level", loggingConfig.Level),
		slog.String("format", loggingConfig.Format),
	)

	return loggingConfig, nil
}
// getDefaultLoggingConfig возвращает конфигурацию логирования по умолчанию.
// ВАЖНО: Значения ДОЛЖНЫ совпадать с константами logging.DefaultXxx из
// internal/pkg/logging/config.go — единственный источник истины для defaults.
func getDefaultLoggingConfig() *LoggingConfig {
	return &LoggingConfig{
		Level:      "info",                         // logging.DefaultLevel
		Format:     "text",                         // logging.DefaultFormat
		Output:     "stderr",                       // logging.DefaultOutput
		FilePath:   "/var/log/apk-ci.log",  // logging.DefaultFilePath
		MaxSize:    100,                            // logging.DefaultMaxSize
		MaxBackups: 3,                              // logging.DefaultMaxBackups
		MaxAge:     7,                              // logging.DefaultMaxAge
		Compress:   true,                           // logging.DefaultCompress
	}
}
