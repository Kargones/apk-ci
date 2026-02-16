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
// Используется при создании Alerter через NewAlerter().
type Config struct {
	// Enabled — включён ли алертинг (по умолчанию false).
	Enabled bool

	// RateLimitWindow — минимальный интервал между алертами одного типа.
	// По умолчанию: 5 минут.
	RateLimitWindow time.Duration

	// Email — конфигурация email канала.
	Email EmailConfig

	// Telegram — конфигурация telegram канала.
	Telegram TelegramConfig

	// Webhook — конфигурация webhook канала.
	Webhook WebhookConfig
}

// EmailConfig содержит настройки email канала.
type EmailConfig struct {
	// Enabled — включён ли email канал.
	Enabled bool

	// SMTPHost — адрес SMTP сервера.
	SMTPHost string

	// SMTPPort — порт SMTP сервера (25, 465, 587).
	// По умолчанию: 587 (StartTLS).
	SMTPPort int

	// SMTPUser — пользователь для SMTP авторизации.
	SMTPUser string

	// SMTPPassword — пароль для SMTP авторизации.
	SMTPPassword string

	// UseTLS — использовать TLS (StartTLS для 587, implicit для 465).
	// По умолчанию: true.
	UseTLS bool

	// From — адрес отправителя.
	From string

	// To — список получателей.
	To []string

	// SubjectTemplate — шаблон темы письма.
	// Placeholders: {{.ErrorCode}}, {{.Command}}, {{.Infobase}}
	// По умолчанию: "[apk-ci] {{.ErrorCode}}: {{.Command}}"
	SubjectTemplate string

	// Timeout — таймаут SMTP операций.
	// По умолчанию: 30 секунд.
	Timeout time.Duration
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
	}
}

// Validate проверяет корректность конфигурации.
// Возвращает ошибку если обязательные поля не заполнены.
func (c *Config) Validate() error {
	// Если alerting отключён — валидация не требуется
	if !c.Enabled {
		return nil
	}

	// Если email канал включён — проверяем обязательные поля
	if c.Email.Enabled {
		if err := c.Email.Validate(); err != nil {
			return err
		}
	}

	// Если telegram канал включён — проверяем обязательные поля
	if c.Telegram.Enabled {
		if err := c.Telegram.Validate(); err != nil {
			return err
		}
	}

	// Если webhook канал включён — проверяем обязательные поля
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

	// H-1/Review #9: Защита от CRLF injection в email заголовках.
	// Управляющие символы в From/To могут внедрить произвольные SMTP заголовки.
	// M-4/Review #10: используем email-specific валидатор (HTAB запрещён в email адресах).
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
