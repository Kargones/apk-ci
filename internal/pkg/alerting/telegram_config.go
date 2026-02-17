package alerting

import "time"

// Значения по умолчанию для Telegram конфигурации.
const (
	// DefaultTelegramTimeout — таймаут Telegram API по умолчанию.
	DefaultTelegramTimeout = 10 * time.Second
)

// TelegramConfig содержит настройки telegram канала для alerting пакета.
type TelegramConfig struct {
	// Enabled — включён ли telegram канал.
	Enabled bool `yaml:"enabled" env:"BR_ALERTING_TELEGRAM_ENABLED" env-default:"false"`

	// BotToken — токен Telegram бота (получить у @BotFather).
	BotToken string `yaml:"botToken" env:"BR_ALERTING_TELEGRAM_BOT_TOKEN"`

	// ChatIDs — список идентификаторов чатов/групп для отправки.
	ChatIDs []string `yaml:"chatIds" env:"BR_ALERTING_TELEGRAM_CHAT_IDS" env-separator:","`

	// Timeout — таймаут HTTP запросов к Telegram API.
	Timeout time.Duration `yaml:"timeout" env:"BR_ALERTING_TELEGRAM_TIMEOUT" env-default:"10s"`
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
	for _, chatID := range t.ChatIDs {
		if chatID == "" {
			return ErrTelegramChatIDInvalid
		}
		if chatID[0] == '@' {
			continue
		}
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
