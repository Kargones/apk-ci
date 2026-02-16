package config

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/Kargones/apk-ci/internal/pkg/alerting"
	"github.com/ilyakaznacheev/cleanenv"
)

// AlertingConfig содержит настройки для алертинга.
type AlertingConfig struct {
	// Enabled — включён ли алертинг (по умолчанию false).
	Enabled bool `yaml:"enabled" env:"BR_ALERTING_ENABLED" env-default:"false"`

	// RateLimitWindow — минимальный интервал между алертами одного типа.
	RateLimitWindow time.Duration `yaml:"rateLimitWindow" env:"BR_ALERTING_RATE_LIMIT_WINDOW" env-default:"5m"`

	// Email — конфигурация email канала.
	Email EmailChannelConfig `yaml:"email"`

	// Telegram — конфигурация telegram канала.
	Telegram TelegramChannelConfig `yaml:"telegram"`

	// Webhook — конфигурация webhook канала.
	Webhook WebhookChannelConfig `yaml:"webhook"`

	// Rules — правила фильтрации алертов.
	Rules AlertRulesConfig `yaml:"rules"`
}
// AlertRulesConfig содержит настройки правил фильтрации алертов.
type AlertRulesConfig struct {
	// MinSeverity — минимальный уровень severity для отправки алерта.
	// Значения: "INFO", "WARNING", "CRITICAL". По умолчанию: "INFO" (все алерты).
	MinSeverity string `yaml:"minSeverity" env:"BR_ALERTING_RULES_MIN_SEVERITY" env-default:"INFO"`

	// ExcludeErrorCodes — коды ошибок, для которых НЕ отправляются алерты.
	ExcludeErrorCodes []string `yaml:"excludeErrorCodes" env:"BR_ALERTING_RULES_EXCLUDE_ERRORS" env-separator:","`

	// IncludeErrorCodes — если задан, алерты отправляются ТОЛЬКО для этих кодов.
	// Имеет приоритет над ExcludeErrorCodes.
	IncludeErrorCodes []string `yaml:"includeErrorCodes" env:"BR_ALERTING_RULES_INCLUDE_ERRORS" env-separator:","`

	// ExcludeCommands — команды, для которых НЕ отправляются алерты.
	ExcludeCommands []string `yaml:"excludeCommands" env:"BR_ALERTING_RULES_EXCLUDE_COMMANDS" env-separator:","`

	// IncludeCommands — если задан, алерты отправляются ТОЛЬКО для этих команд.
	// Имеет приоритет над ExcludeCommands.
	IncludeCommands []string `yaml:"includeCommands" env:"BR_ALERTING_RULES_INCLUDE_COMMANDS" env-separator:","`

	// ChannelOverrides — правила для конкретных каналов.
	// ВНИМАНИЕ: channel override ПОЛНОСТЬЮ ЗАМЕНЯЕТ глобальные правила для канала,
	// а НЕ мержит с ними. Если указан override с excludeErrorCodes но без minSeverity,
	// будет использован minSeverity=INFO (default), а не глобальный minSeverity.
	//
	// Пример НЕПРАВИЛЬНОЙ конфигурации (email получит ВСЕ severity, не только CRITICAL):
	//   rules:
	//     minSeverity: "CRITICAL"
	//     channels:
	//       email:
	//         excludeErrorCodes: ["ERR_SPAM"]
	//
	// Пример ПРАВИЛЬНОЙ конфигурации (повторяем minSeverity в override):
	//   rules:
	//     minSeverity: "CRITICAL"
	//     channels:
	//       email:
	//         minSeverity: "CRITICAL"
	//         excludeErrorCodes: ["ERR_SPAM"]
	ChannelOverrides map[string]ChannelRuleConfig `yaml:"channels"`
}
// ChannelRuleConfig — правила для конкретного канала алертинга.
type ChannelRuleConfig struct {
	MinSeverity       string   `yaml:"minSeverity"`
	ExcludeErrorCodes []string `yaml:"excludeErrorCodes"`
	IncludeErrorCodes []string `yaml:"includeErrorCodes"`
	ExcludeCommands   []string `yaml:"excludeCommands"`
	IncludeCommands   []string `yaml:"includeCommands"`
}
// EmailChannelConfig содержит настройки email канала.
type EmailChannelConfig struct {
	// Enabled — включён ли email канал.
	Enabled bool `yaml:"enabled" env:"BR_ALERTING_EMAIL_ENABLED" env-default:"false"`

	// SMTPHost — адрес SMTP сервера.
	SMTPHost string `yaml:"smtpHost" env:"BR_ALERTING_SMTP_HOST"`

	// SMTPPort — порт SMTP сервера (25, 465, 587).
	SMTPPort int `yaml:"smtpPort" env:"BR_ALERTING_SMTP_PORT" env-default:"587"`

	// SMTPUser — пользователь для SMTP авторизации.
	SMTPUser string `yaml:"smtpUser" env:"BR_ALERTING_SMTP_USER"`

	// SMTPPassword — пароль для SMTP авторизации.
	SMTPPassword string `yaml:"smtpPassword" env:"BR_ALERTING_SMTP_PASSWORD"`

	// UseTLS — использовать TLS (StartTLS для 587, implicit для 465).
	// TODO: bool с env-default:"true" — аналогично Compress,
	// YAML false может быть перезаписан cleanenv.ReadEnv. Для YAML-source
	// используется getDefaultAlertingConfig() где UseTLS=true.
	UseTLS bool `yaml:"useTLS" env:"BR_ALERTING_SMTP_TLS" env-default:"true"`

	// From — адрес отправителя.
	From string `yaml:"from" env:"BR_ALERTING_EMAIL_FROM"`

	// To — список получателей (comma-separated в env).
	To []string `yaml:"to" env:"BR_ALERTING_EMAIL_TO" env-separator:","`

	// SubjectTemplate — шаблон темы письма.
	// Placeholders: {{.ErrorCode}}, {{.Command}}, {{.Infobase}}
	SubjectTemplate string `yaml:"subjectTemplate" env:"BR_ALERTING_EMAIL_SUBJECT" env-default:"[apk-ci] {{.ErrorCode}}: {{.Command}}"`

	// Timeout — таймаут SMTP операций.
	Timeout time.Duration `yaml:"timeout" env:"BR_ALERTING_SMTP_TIMEOUT" env-default:"30s"`
}
// TelegramChannelConfig содержит настройки telegram канала.
type TelegramChannelConfig struct {
	// Enabled — включён ли telegram канал.
	Enabled bool `yaml:"enabled" env:"BR_ALERTING_TELEGRAM_ENABLED" env-default:"false"`

	// BotToken — токен Telegram бота (получить у @BotFather).
	BotToken string `yaml:"botToken" env:"BR_ALERTING_TELEGRAM_BOT_TOKEN"`

	// ChatIDs — список идентификаторов чатов/групп для отправки.
	// Может быть числовой ID или @username для публичных каналов.
	ChatIDs []string `yaml:"chatIds" env:"BR_ALERTING_TELEGRAM_CHAT_IDS" env-separator:","`

	// Timeout — таймаут HTTP запросов к Telegram API.
	// По умолчанию: 10 секунд.
	Timeout time.Duration `yaml:"timeout" env:"BR_ALERTING_TELEGRAM_TIMEOUT" env-default:"10s"`
}
// WebhookChannelConfig содержит настройки webhook канала.
type WebhookChannelConfig struct {
	// Enabled — включён ли webhook канал.
	Enabled bool `yaml:"enabled" env:"BR_ALERTING_WEBHOOK_ENABLED" env-default:"false"`

	// URLs — список URL для отправки webhook.
	// Алерт отправляется на все указанные URL.
	URLs []string `yaml:"urls" env:"BR_ALERTING_WEBHOOK_URLS" env-separator:","`

	// Headers — дополнительные HTTP заголовки.
	// Используется для Authorization, X-Api-Key и т.д.
	// TODO: Headers доступны только через YAML, не через env.
	// cleanenv не поддерживает map[string]string из env переменных.
	// Для CI/CD использовать YAML или добавить парсинг "Key=Val,Key2=Val2" из env.
	Headers map[string]string `yaml:"headers"`

	// Timeout — таймаут HTTP запросов.
	// По умолчанию: 10 секунд.
	Timeout time.Duration `yaml:"timeout" env:"BR_ALERTING_WEBHOOK_TIMEOUT" env-default:"10s"`

	// MaxRetries — максимальное количество повторных попыток.
	// По умолчанию: 3.
	MaxRetries int `yaml:"maxRetries" env:"BR_ALERTING_WEBHOOK_MAX_RETRIES" env-default:"3"`
}
// isAlertingConfigPresent проверяет, задана ли конфигурация алертинга.
// Возвращает true если хотя бы одно значимое поле отличается от zero value.
func isAlertingConfigPresent(cfg *AlertingConfig) bool {
	if cfg == nil {
		return false
	}
	// Проверяем любое значимое поле (enabled, или настройки email/telegram/webhook)
	return cfg.Enabled ||
		cfg.Email.Enabled || cfg.Email.SMTPHost != "" ||
		cfg.Telegram.Enabled || cfg.Telegram.BotToken != "" ||
		cfg.Webhook.Enabled || len(cfg.Webhook.URLs) > 0
}
// getDefaultAlertingConfig возвращает конфигурацию алертинга по умолчанию.
// Алертинг отключён по умолчанию (AC6).
// M-1/Review #9: используем константы из пакета alerting вместо magic numbers.
func getDefaultAlertingConfig() *AlertingConfig {
	return &AlertingConfig{
		Enabled:         false,
		RateLimitWindow: alerting.DefaultRateLimitWindow,
		Email: EmailChannelConfig{
			Enabled:         false,
			SMTPPort:        alerting.DefaultSMTPPort,
			UseTLS:          true,
			SubjectTemplate: alerting.DefaultSubjectTemplate,
			Timeout:         alerting.DefaultSMTPTimeout,
		},
		Telegram: TelegramChannelConfig{
			Enabled: false,
			Timeout: alerting.DefaultTelegramTimeout,
		},
		Webhook: WebhookChannelConfig{
			Enabled:    false,
			Timeout:    alerting.DefaultWebhookTimeout,
			MaxRetries: alerting.DefaultMaxRetries,
		},
		Rules: AlertRulesConfig{
			MinSeverity: "INFO",
		},
	}
}
// loadAlertingConfig загружает конфигурацию алертинга из AppConfig, переменных окружения или устанавливает значения по умолчанию.
// Переменные окружения BR_ALERTING_* переопределяют значения из AppConfig (AC5).
func loadAlertingConfig(l *slog.Logger, cfg *Config) (*AlertingConfig, error) {
	// Проверяем, есть ли конфигурация в AppConfig
	// Примечание: нельзя сравнивать struct напрямую из-за slice в EmailChannelConfig
	if cfg.AppConfig != nil && isAlertingConfigPresent(&cfg.AppConfig.Alerting) {
		alertingConfig := &cfg.AppConfig.Alerting
		// Применяем env override для AppConfig (симметрично с loadLoggingConfig)
		if err := cleanenv.ReadEnv(alertingConfig); err != nil {
			l.Warn("Ошибка загрузки Alerting конфигурации из переменных окружения",
				slog.String("error", err.Error()),
			)
		}
		l.Info("Alerting конфигурация загружена из AppConfig",
			slog.Bool("enabled", alertingConfig.Enabled),
			slog.Bool("email_enabled", alertingConfig.Email.Enabled),
			slog.Bool("telegram_enabled", alertingConfig.Telegram.Enabled), // L1 fix
		)
		return alertingConfig, nil
	}

	alertingConfig := getDefaultAlertingConfig()

	if err := cleanenv.ReadEnv(alertingConfig); err != nil {
		l.Warn("Ошибка загрузки Alerting конфигурации из переменных окружения",
			slog.String("error", err.Error()),
		)
	}

	l.Debug("Alerting конфигурация: используются значения по умолчанию",
		slog.Bool("enabled", alertingConfig.Enabled),
	)

	return alertingConfig, nil
}
// validateAlertingConfig проверяет корректность конфигурации алертинга при загрузке.
// Проверяет обязательные поля для каждого включённого канала.
//
// M-2/Review #9: Это предварительная (config-level) валидация — проверяет только наличие
// обязательных полей при загрузке конфигурации. Полная валидация (формат URL, CRLF injection
// в email адресах, Header Injection в webhook headers) выполняется в alerting.Config.Validate()
// при создании Alerter через providers.go. Defense-in-depth: fail-fast при явно невалидной конфигурации.
func validateAlertingConfig(ac *AlertingConfig) error {
	if !ac.Enabled {
		return nil
	}
	if ac.Email.Enabled {
		if ac.Email.SMTPHost == "" {
			return fmt.Errorf("alerting.email: SMTP host обязателен")
		}
		if ac.Email.From == "" {
			return fmt.Errorf("alerting.email: адрес отправителя (from) обязателен")
		}
		if len(ac.Email.To) == 0 {
			return fmt.Errorf("alerting.email: хотя бы один получатель (to) обязателен")
		}
	}
	if ac.Telegram.Enabled {
		if ac.Telegram.BotToken == "" {
			return fmt.Errorf("alerting.telegram: bot_token обязателен")
		}
		if len(ac.Telegram.ChatIDs) == 0 {
			return fmt.Errorf("alerting.telegram: хотя бы один chat_id обязателен")
		}
	}
	if ac.Webhook.Enabled && len(ac.Webhook.URLs) == 0 {
		return fmt.Errorf("alerting.webhook: хотя бы один URL обязателен")
	}
	return nil
}
