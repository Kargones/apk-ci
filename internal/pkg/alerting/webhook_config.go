package alerting

import (
	"net/url"
	"time"
)

// Значения по умолчанию для Webhook конфигурации.
const (
	// DefaultWebhookTimeout — таймаут HTTP запросов по умолчанию.
	DefaultWebhookTimeout = 10 * time.Second

	// DefaultMaxRetries — количество повторных попыток по умолчанию.
	DefaultMaxRetries = 3
)

// WebhookConfig содержит настройки webhook канала для alerting пакета.
type WebhookConfig struct {
	// Enabled — включён ли webhook канал.
	Enabled bool `yaml:"enabled" env:"BR_ALERTING_WEBHOOK_ENABLED" env-default:"false"`

	// URLs — список URL для отправки webhook.
	URLs []string `yaml:"urls" env:"BR_ALERTING_WEBHOOK_URLS" env-separator:","`

	// Headers — дополнительные HTTP заголовки.
	Headers map[string]string `yaml:"headers"`

	// Timeout — таймаут HTTP запросов.
	Timeout time.Duration `yaml:"timeout" env:"BR_ALERTING_WEBHOOK_TIMEOUT" env-default:"10s"`

	// MaxRetries — максимальное количество повторных попыток.
	MaxRetries int `yaml:"maxRetries" env:"BR_ALERTING_WEBHOOK_MAX_RETRIES" env-default:"3"`
}

// Validate проверяет корректность WebhookConfig.
func (w *WebhookConfig) Validate() error {
	if !w.Enabled {
		return nil
	}
	if len(w.URLs) == 0 {
		return ErrWebhookURLRequired
	}
	for _, rawURL := range w.URLs {
		u, err := url.Parse(rawURL)
		if err != nil {
			return ErrWebhookURLInvalid
		}
		if u.Scheme == "" || u.Host == "" {
			return ErrWebhookURLInvalid
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return ErrWebhookURLInvalid
		}
	}
	for key, value := range w.Headers {
		if containsInvalidHTTPHeaderChars(key) || containsInvalidHTTPHeaderChars(value) {
			return ErrWebhookHeaderInvalid
		}
	}
	return nil
}

// containsInvalidHTTPHeaderChars проверяет наличие запрещённых символов в HTTP заголовке.
func containsInvalidHTTPHeaderChars(s string) bool {
	for _, r := range s {
		if r == 0x09 {
			continue
		}
		if r <= 0x1f || r == 0x7f {
			return true
		}
	}
	return false
}

// containsInvalidEmailHeaderChars проверяет наличие запрещённых символов в email заголовке.
func containsInvalidEmailHeaderChars(s string) bool {
	for _, r := range s {
		if r <= 0x1f || r == 0x7f {
			return true
		}
	}
	return false
}
