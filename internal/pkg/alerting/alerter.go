// Package alerting предоставляет интерфейс и реализации для отправки алертов.
// Поддерживает email, telegram и webhook каналы с rate limiting, TLS и per-channel правилами фильтрации.
package alerting

import (
	"context"
	"time"
)

// Severity определяет уровень критичности алерта.
type Severity int

const (
	// SeverityInfo — информационный алерт.
	SeverityInfo Severity = iota
	// SeverityWarning — предупреждающий алерт.
	SeverityWarning
	// SeverityCritical — критический алерт.
	SeverityCritical
)

// Имена каналов алертинга.
const (
	// ChannelEmail — имя email канала.
	ChannelEmail = "email"
	// ChannelTelegram — имя telegram канала.
	ChannelTelegram = "telegram"
	// ChannelWebhook — имя webhook канала.
	ChannelWebhook = "webhook"
)

// String возвращает строковое представление Severity.
func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "INFO"
	case SeverityWarning:
		return "WARNING"
	case SeverityCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// Alert представляет данные для отправки алерта.
type Alert struct {
	// ErrorCode — код ошибки для rate limiting и идентификации.
	ErrorCode string

	// Message — человекочитаемое сообщение об ошибке.
	Message string

	// TraceID — идентификатор трассировки для корреляции логов.
	TraceID string

	// Timestamp — время возникновения ошибки.
	Timestamp time.Time

	// Command — команда, вызвавшая ошибку.
	Command string

	// Infobase — информационная база (если применимо).
	Infobase string

	// Severity — уровень критичности алерта.
	Severity Severity
}

// Alerter определяет интерфейс для отправки алертов.
// Реализации: EmailAlerter, TelegramAlerter, WebhookAlerter, MultiChannelAlerter, NopAlerter.
//
// ВАЖНО: Alerter не должен прерывать работу приложения при ошибках отправки.
// Все ошибки логируются, приложение продолжает работу.
//
// Design Decision (AC10): Send() всегда возвращает nil для обеспечения
// устойчивости приложения. Ошибки SMTP, rate limiting и другие проблемы
// логируются, но не возвращаются caller'у. Это предотвращает каскадные
// ошибки когда alerting infrastructure недоступна.
type Alerter interface {
	// Send отправляет алерт через настроенные каналы.
	// ВСЕГДА возвращает nil (ошибки логируются, не возвращаются) — см. AC10.
	// При частичной доставке — логирует warning, возвращает nil.
	//
	// Примеры использования:
	//   alerter.Send(ctx, Alert{ErrorCode: "DB_CONN_FAIL", Message: "Ошибка подключения к БД"})
	//
	// Rate limiting применяется по ErrorCode — если алерт с таким кодом
	// был отправлен недавно (в пределах RateLimitWindow), Send возвращает nil
	// без фактической отправки.
	Send(ctx context.Context, alert Alert) error
}
