package alerting

import (
	"fmt"

	"github.com/Kargones/apk-ci/internal/pkg/logging"
)

// NewAlerter создаёт Alerter на основе конфигурации и правил фильтрации.
// Если alerting отключён (enabled=false) — возвращает NopAlerter.
// Если нет настроенных каналов — возвращает NopAlerter.
// Иначе возвращает MultiChannelAlerter с именованными каналами и RulesEngine
// для per-channel фильтрации алертов.
//
// Пример использования:
//
//	config := alerting.Config{
//	    Enabled: true,
//	    Email: alerting.EmailConfig{
//	        Enabled:  true,
//	        SMTPHost: "smtp.example.com",
//	        From:     "alerts@example.com",
//	        To:       []string{"devops@example.com"},
//	    },
//	    Telegram: alerting.TelegramConfig{
//	        Enabled:  true,
//	        BotToken: "123456:ABC-DEF",
//	        ChatIDs:  []string{"-1001234567890"},
//	    },
//	}
//	rules := alerting.RulesConfig{MinSeverity: "WARNING"}
//	alerter, err := alerting.NewAlerter(config, rules, logger)
func NewAlerter(config Config, rules RulesConfig, logger logging.Logger) (Alerter, error) {
	// Если alerting отключён — возвращаем NopAlerter
	if !config.Enabled {
		return NewNopAlerter(), nil
	}

	// Валидируем конфигурацию
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Создаём общий rate limiter для всех каналов
	rateLimitWindow := config.RateLimitWindow
	if rateLimitWindow == 0 {
		rateLimitWindow = DefaultRateLimitWindow
	}
	rateLimiter := NewRateLimiter(rateLimitWindow)

	// Создаём именованные каналы напрямую для корректного per-channel rules маппинга.
	// Rate limiter передаётся как nil в индивидуальные каналы — rate limiting
	// теперь применяется один раз на уровне MultiChannelAlerter, чтобы все каналы
	// получали алерт или ни один (H-2 fix).
	namedChannels := make(map[string]Alerter)

	// Email канал
	if config.Email.Enabled {
		emailAlerter, err := NewEmailAlerter(config.Email, nil, logger)
		if err != nil {
			return nil, fmt.Errorf("создание email alerter: %w", err)
		}
		namedChannels[ChannelEmail] = emailAlerter
	}

	// Telegram канал
	if config.Telegram.Enabled {
		telegramAlerter, err := NewTelegramAlerter(config.Telegram, nil, logger)
		if err != nil {
			return nil, fmt.Errorf("создание telegram alerter: %w", err)
		}
		namedChannels[ChannelTelegram] = telegramAlerter
	}

	// Webhook канал
	if config.Webhook.Enabled {
		webhookAlerter, err := NewWebhookAlerter(config.Webhook, nil, logger)
		if err != nil {
			return nil, fmt.Errorf("создание webhook alerter: %w", err)
		}
		namedChannels[ChannelWebhook] = webhookAlerter
	}

	if len(namedChannels) == 0 {
		logger.Warn("alerting включён, но нет настроенных каналов — используется NopAlerter")
		return NewNopAlerter(), nil
	}

	// M-7/Review #15: Предупреждение о footgun с channel override без minSeverity.
	// Channel override ПОЛНОСТЬЮ заменяет глобальные правила. Если пользователь
	// забудет указать minSeverity в override, будет использован INFO (default).
	if rules.MinSeverity != "" {
		for name, ch := range rules.Channels {
			if ch.MinSeverity == "" {
				logger.Warn("channel override без minSeverity — будет использован INFO, а не глобальный",
					"channel", name,
					"global_min_severity", rules.MinSeverity,
				)
			}
		}
	}

	// Создаём rules engine для фильтрации алертов
	rulesEngine := NewRulesEngine(rules)

	return NewMultiChannelAlerter(namedChannels, rulesEngine, rateLimiter, logger), nil
}
