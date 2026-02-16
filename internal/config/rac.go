package config

import (
	"log/slog"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// RacConfig содержит настройки для RAC (Remote Administration Console).
type RacConfig struct {
	// RacPath - путь к исполняемому файлу RAC
	RacPath string `yaml:"racPath" env:"RAC_PATH"`

	// RacServer - адрес сервера RAC
	RacServer string `yaml:"racServer" env:"RAC_SERVER"`

	// RacPort - порт сервера RAC
	RacPort int `yaml:"racPort" env:"RAC_PORT"`

	// RacUser - пользователь RAC
	RacUser string `yaml:"racUser" env:"RAC_USER"`

	// RacPassword - пароль пользователя RAC
	RacPassword string `yaml:"racPassword" env:"RAC_PASSWORD"`

	// DbUser - пользователь базы данных
	DbUser string `yaml:"dbUser" env:"RAC_DB_USER"`

	// DbPassword - пароль пользователя базы данных
	DbPassword string `yaml:"dbPassword" env:"RAC_DB_PASSWORD"`

	// Timeout - таймаут для RAC операций
	Timeout time.Duration `yaml:"timeout" env:"RAC_TIMEOUT"`

	// Retries - количество попыток повтора RAC операций
	Retries int `yaml:"retries" env:"RAC_RETRIES"`
}
// loadRacConfig загружает конфигурацию RAC из AppConfig, SecretConfig, переменных окружения или устанавливает значения по умолчанию
func loadRacConfig(l *slog.Logger, cfg *Config) (*RacConfig, error) {
	// Проверяем, есть ли конфигурация в AppConfig
	if cfg.AppConfig != nil {
		racConfig := &RacConfig{
			RacPath:   cfg.AppConfig.Paths.Rac,
			RacServer: "localhost", // значение по умолчанию
			RacPort:   cfg.AppConfig.Rac.Port,
			RacUser:   cfg.AppConfig.Users.Rac,
			DbUser:    cfg.AppConfig.Users.Db,
			Timeout:   time.Duration(cfg.AppConfig.Rac.Timeout) * time.Second,
			Retries:   cfg.AppConfig.Rac.Retries,
		}

		// Дополняем паролями из SecretConfig, если они есть
		if cfg.SecretConfig != nil {
			if cfg.SecretConfig.Passwords.Rac != "" {
				racConfig.RacPassword = cfg.SecretConfig.Passwords.Rac
			}
			if cfg.SecretConfig.Passwords.Db != "" {
				racConfig.DbPassword = cfg.SecretConfig.Passwords.Db
			}
		}

		// Если основные поля заполнены, возвращаем конфигурацию
		if racConfig.RacPath != "" || racConfig.RacPort != 0 {
			return racConfig, nil
		}
	}

	racConfig := getDefaultRacConfig()

	if err := cleanenv.ReadEnv(racConfig); err != nil {
		l.Warn("Ошибка загрузки RAC конфигурации из переменных окружения",
			slog.String("error", err.Error()),
		)
	}

	return racConfig, nil
}
// getDefaultRacConfig возвращает конфигурацию RAC по умолчанию
func getDefaultRacConfig() *RacConfig {
	return &RacConfig{
		RacPath:     "/opt/1cv8/x86_64/8.3.25.1257/rac",
		RacServer:   "localhost",
		RacPort:     1545,
		RacUser:     "",
		RacPassword: "",
		DbUser:      "",
		DbPassword:  "",
		Timeout:     30 * time.Second,
		Retries:     3,
	}
}
