// Package alerting предоставляет конфигурацию для системы алертинга.
// Этот файл содержит структуры конфигурации и значения по умолчанию.
package alerting

import "time"

// Значения по умолчанию для конфигурации alerting.
const (
	// DefaultRateLimitWindow — интервал между алертами одного типа по умолчанию.
	DefaultRateLimitWindow = 5 * time.Minute

	// DefaultSMTPPort — порт SMTP по умолчанию (StartTLS).
	DefaultSMTPPort = 587

	// DefaultSMTPTimeout — таймаут SMTP операций по умолчанию.
	DefaultSMTPTimeout = 30 * time.Second

	// DefaultSubjectTemplate — шаблон темы письма по умолчанию.
	DefaultSubjectTemplate = "[apk-ci] {{.ErrorCode}}: {{.Command}}"
)

// Config содержит настройки для пакета alerting.
// Используется при создании Alerter через NewAlerter() и как единый источник истины
// для конфигурации алертинга (issue #81: устранение дублирования config/alerting structs).
type Config struct {
	// Enabled — включён ли алертинг (по умолчанию false).
	Enabled bool `yaml:"enabled" env:"BR_ALERTING_ENABLED" env-default:"false"`

	// RateLimitWindow — минимальный интервал между алертами одного типа.
	RateLimitWindow time.Duration `yaml:"rateLimitWindow" env:"BR_ALERTING_RATE_LIMIT_WINDOW" env-default:"5m"`

	// Email — конфигурация email канала.
	Email EmailConfig `yaml:"email"`

	// Telegram — конфигурация telegram канала.
	Telegram TelegramConfig `yaml:"telegram"`

	// Webhook — конфигурация webhook канала.
	Webhook WebhookConfig `yaml:"webhook"`

	// Rules — правила фильтрации алертов.
	Rules RulesConfig `yaml:"rules"`
}

// EmailConfig содержит настройки email канала.
type EmailConfig struct {
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
	UseTLS bool `yaml:"useTLS" env:"BR_ALERTING_SMTP_TLS" env-default:"true"`

	// From — адрес отправителя.
	From string `yaml:"from" env:"BR_ALERTING_EMAIL_FROM"`

	// To — список получателей.
	To []string `yaml:"to" env:"BR_ALERTING_EMAIL_TO" env-separator:","`

	// SubjectTemplate — шаблон темы письма.
	// Placeholders: {{.ErrorCode}}, {{.Command}}, {{.Infobase}}
	SubjectTemplate string `yaml:"subjectTemplate" env:"BR_ALERTING_EMAIL_SUBJECT" env-default:"[apk-ci] {{.ErrorCode}}: {{.Command}}"`

	// Timeout — таймаут SMTP операций.
	Timeout time.Duration `yaml:"timeout" env:"BR_ALERTING_SMTP_TIMEOUT" env-default:"30s"`
}

// DefaultConfig возвращает конфигурацию с значениями по умолчанию.
// Alerting отключён по умолчанию.
func DefaultConfig() Config {
	return Config{
		Enabled:         false,
		RateLimitWindow: DefaultRateLimitWindow,
		Email: EmailConfig{
			Enabled:         false,
			SMTPPort:        DefaultSMTPPort,
			UseTLS:          true,
			SubjectTemplate: DefaultSubjectTemplate,
			Timeout:         DefaultSMTPTimeout,
		},
		Telegram: TelegramConfig{
			Enabled: false,
			Timeout: DefaultTelegramTimeout,
		},
		Webhook: WebhookConfig{
			Enabled:    false,
			Timeout:    DefaultWebhookTimeout,
			MaxRetries: DefaultMaxRetries,
		},
		Rules: RulesConfig{
			MinSeverity: "INFO",
		},
	}
}

// Validate проверяет корректность конфигурации.
// Возвращает ошибку если обязательные поля не заполнены.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Email.Enabled {
		if err := c.Email.Validate(); err != nil {
			return err
		}
	}

	if c.Telegram.Enabled {
		if err := c.Telegram.Validate(); err != nil {
			return err
		}
	}

	if c.Webhook.Enabled {
		if err := c.Webhook.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate проверяет корректность EmailConfig.
func (e *EmailConfig) Validate() error {
	if !e.Enabled {
		return nil
	}

	if e.SMTPHost == "" {
		return ErrSMTPHostRequired
	}

	if e.From == "" {
		return ErrFromRequired
	}

	if containsInvalidEmailHeaderChars(e.From) {
		return ErrEmailAddressInvalid
	}

	if len(e.To) == 0 {
		return ErrToRequired
	}

	for _, to := range e.To {
		if containsInvalidEmailHeaderChars(to) {
			return ErrEmailAddressInvalid
		}
	}

	return nil
}
