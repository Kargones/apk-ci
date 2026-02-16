package alerting

import "time"

// TODO (M3/Review 6-3): Дублирование — TelegramConfig здесь и TelegramChannelConfig
// в config.go практически идентичны. Рефакторинг: объединить в общий пакет internal/alerting/config
// или использовать alerting.TelegramConfig напрямую в config.AppConfig.
// См. также TODO в providers.go:128 для EmailConfig.

// Значения по умолчанию для Telegram конфигурации.
const (
	// DefaultTelegramTimeout — таймаут Telegram API по умолчанию.
	DefaultTelegramTimeout = 10 * time.Second
)

// TelegramConfig содержит настройки telegram канала для alerting пакета.
type TelegramConfig struct {
	// Enabled — включён ли telegram канал.
	Enabled bool

	// BotToken — токен Telegram бота (получить у @BotFather).
	BotToken string

	// ChatIDs — список идентификаторов чатов/групп для отправки.
	ChatIDs []string

	// Timeout — таймаут HTTP запросов к Telegram API.
	Timeout time.Duration
}

// Validate проверяет корректность TelegramConfig.
func (t *TelegramConfig) Validate() error {
	if !t.Enabled {
		return nil
	}
	if t.BotToken == "" {
		return ErrTelegramBotTokenRequired
	}
	if len(t.ChatIDs) == 0 {
		return ErrTelegramChatIDRequired
	}
	// M-4/Review #15: Валидация формата ChatID.
	// Telegram принимает числовой ID (может быть отрицательным для групп) или @username.
	for _, chatID := range t.ChatIDs {
		if chatID == "" {
			return ErrTelegramChatIDInvalid
		}
		if chatID[0] == '@' {
			continue // @username — валидный формат
		}
		// Проверяем что это числовой ID (возможно с минусом для групп)
		start := 0
		if chatID[0] == '-' {
			if len(chatID) == 1 {
				return ErrTelegramChatIDInvalid
			}
			start = 1
		}
		for _, ch := range chatID[start:] {
			if ch < '0' || ch > '9' {
				return ErrTelegramChatIDInvalid
			}
		}
	}
	return nil
}
