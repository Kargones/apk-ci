package config

import (
	"log/slog"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// GitConfig содержит настройки для Git операций.
type GitConfig struct {
	// UserName - имя пользователя Git
	UserName string `yaml:"userName" env:"GIT_USER_NAME"`

	// UserEmail - email пользователя Git
	UserEmail string `yaml:"userEmail" env:"GIT_USER_EMAIL"`

	// DefaultBranch - ветка по умолчанию
	DefaultBranch string `yaml:"defaultBranch" env:"GIT_DEFAULT_BRANCH"`

	// Timeout - таймаут для Git операций
	Timeout time.Duration `yaml:"timeout" env:"GIT_TIMEOUT"`

	// CredentialHelper - настройка credential helper
	CredentialHelper string `yaml:"credentialHelper" env:"GIT_CREDENTIAL_HELPER"`

	// CredentialTimeout - таймаут для кэша credentials
	CredentialTimeout time.Duration `yaml:"credentialTimeout" env:"GIT_CREDENTIAL_TIMEOUT"`
}
// loadGitConfig загружает конфигурацию Git из AppConfig или устанавливает значения по умолчанию
func loadGitConfig(l *slog.Logger, cfg *Config) (*GitConfig, error) {
	// Сначала пытаемся загрузить из AppConfig
	if cfg.AppConfig != nil {
		gitConfig := &cfg.AppConfig.Git
		// Если конфигурация не пустая, используем её
		if gitConfig.UserName != "" || gitConfig.UserEmail != "" {
			l.Info("Git конфигурация загружена из AppConfig")
			return gitConfig, nil
		}
	}

	// Если не удалось загрузить из AppConfig, используем значения по умолчанию
	gitConfig := getDefaultGitConfig()

	// Пытаемся загрузить из переменных окружения
	if err := cleanenv.ReadEnv(gitConfig); err != nil {
		l.Warn("Ошибка загрузки Git конфигурации из переменных окружения",
			slog.String("error", err.Error()),
		)
	}

	return gitConfig, nil
}
// getDefaultGitConfig возвращает конфигурацию Git по умолчанию
func getDefaultGitConfig() *GitConfig {
	return &GitConfig{
		UserName:          "apk-ci",
		UserEmail:         "runner@benadis.ru",
		DefaultBranch:     "main",
		Timeout:           60 * time.Minute,
		CredentialHelper:  "store",
		CredentialTimeout: 24 * time.Hour,
	}
}
