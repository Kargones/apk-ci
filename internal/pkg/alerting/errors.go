package alerting

import "errors"

// Ошибки валидации конфигурации.
var (
	// ErrSMTPHostRequired — SMTP host не указан.
	ErrSMTPHostRequired = errors.New("alerting: smtp_host is required when email channel is enabled")

	// ErrFromRequired — адрес отправителя не указан.
	ErrFromRequired = errors.New("alerting: from address is required when email channel is enabled")

	// ErrToRequired — список получателей пуст.
	ErrToRequired = errors.New("alerting: at least one recipient is required when email channel is enabled")

	// ErrEmailAddressInvalid — email адрес содержит управляющие символы (CRLF injection).
	ErrEmailAddressInvalid = errors.New("alerting: email address contains invalid characters (control chars)")

	// ErrTelegramBotTokenRequired — bot token не указан.
	ErrTelegramBotTokenRequired = errors.New("alerting: bot_token is required when telegram channel is enabled")

	// ErrTelegramChatIDRequired — chat_id не указан.
	ErrTelegramChatIDRequired = errors.New("alerting: at least one chat_id is required when telegram channel is enabled")

	// ErrTelegramChatIDInvalid — chat_id имеет невалидный формат (ожидается числовой ID или @username).
	ErrTelegramChatIDInvalid = errors.New("alerting: chat_id must be a numeric ID or @username")

	// ErrWebhookURLRequired — URL для webhook не указан.
	ErrWebhookURLRequired = errors.New("alerting: at least one url is required when webhook channel is enabled")

	// ErrWebhookURLInvalid — URL имеет невалидный формат.
	ErrWebhookURLInvalid = errors.New("alerting: webhook url has invalid format (must have scheme and host)")

	// ErrWebhookHeaderInvalid — HTTP заголовок содержит недопустимые символы.
	ErrWebhookHeaderInvalid = errors.New("alerting: webhook header contains invalid characters (\\r or \\n)")
)

// Ошибки отправки.
var (
	// ErrSMTPConnection — ошибка подключения к SMTP серверу.
	ErrSMTPConnection = errors.New("alerting: failed to connect to SMTP server")

	// ErrSMTPAuth — ошибка аутентификации SMTP.
	ErrSMTPAuth = errors.New("alerting: SMTP authentication failed")

	// ErrSMTPSend — ошибка отправки email.
	ErrSMTPSend = errors.New("alerting: failed to send email")

	// ErrRateLimited — алерт был подавлен rate limiter.
	ErrRateLimited = errors.New("alerting: alert rate limited")
)
