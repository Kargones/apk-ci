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
	Enabled bool

	// URLs — список URL для отправки webhook.
	URLs []string

	// Headers — дополнительные HTTP заголовки.
	Headers map[string]string

	// Timeout — таймаут HTTP запросов.
	Timeout time.Duration

	// MaxRetries — максимальное количество повторных попыток.
	MaxRetries int
}

// Validate проверяет корректность WebhookConfig.
func (w *WebhookConfig) Validate() error {
	if !w.Enabled {
		return nil
	}
	if len(w.URLs) == 0 {
		return ErrWebhookURLRequired
	}
	// Валидация формата URL
	for _, rawURL := range w.URLs {
		u, err := url.Parse(rawURL)
		if err != nil {
			return ErrWebhookURLInvalid
		}
		// Проверяем что URL имеет scheme и host
		if u.Scheme == "" || u.Host == "" {
			return ErrWebhookURLInvalid
		}
		// Разрешаем только http и https схемы (защита от SSRF через file://, ftp:// и т.д.)
		if u.Scheme != "http" && u.Scheme != "https" {
			return ErrWebhookURLInvalid
		}
	}
	// Валидация HTTP заголовков — защита от HTTP Header Injection (RFC 7230).
	// Запрещены: CR, LF, и control characters (кроме HTAB).
	for key, value := range w.Headers {
		if containsInvalidHTTPHeaderChars(key) || containsInvalidHTTPHeaderChars(value) {
			return ErrWebhookHeaderInvalid
		}
	}
	return nil
}

// containsInvalidHTTPHeaderChars проверяет наличие запрещённых символов в HTTP заголовке.
// По RFC 7230 разрешены HTAB (0x09) и все printable ASCII, запрещены остальные control characters.
// H-1/Review #10: выделена отдельная функция для HTTP — HTAB допустим в HTTP headers.
func containsInvalidHTTPHeaderChars(s string) bool {
	for _, r := range s {
		if r == 0x09 {
			continue // HTAB разрешён в HTTP header values (RFC 7230 Section 3.2.6)
		}
		if r <= 0x1f || r == 0x7f {
			return true
		}
	}
	return false
}

// containsInvalidEmailHeaderChars проверяет наличие запрещённых символов в email заголовке.
// По RFC 5322 все control characters (0x00-0x1f, 0x7f), включая HTAB, запрещены
// в email адресах для защиты от CRLF injection.
// M-4/Review #10: выделена отдельная функция для email — HTAB запрещён.
func containsInvalidEmailHeaderChars(s string) bool {
	for _, r := range s {
		if r <= 0x1f || r == 0x7f {
			return true
		}
	}
	return false
}
