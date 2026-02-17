package config

import (
	"fmt"
	"log/slog"

	"github.com/Kargones/apk-ci/internal/pkg/alerting"
	"github.com/ilyakaznacheev/cleanenv"
)

// Type aliases — единый источник истины в пакете alerting (issue #81).
// Сохранены для обратной совместимости с существующим кодом.
type AlertingConfig = alerting.Config
type EmailChannelConfig = alerting.EmailConfig
type TelegramChannelConfig = alerting.TelegramConfig
type WebhookChannelConfig = alerting.WebhookConfig
type AlertRulesConfig = alerting.RulesConfig
type ChannelRuleConfig = alerting.ChannelRulesConfig

// isAlertingConfigPresent проверяет, задана ли конфигурация алертинга.
func isAlertingConfigPresent(cfg *AlertingConfig) bool {
	if cfg == nil {
		return false
	}
	return cfg.Enabled ||
		cfg.Email.Enabled || cfg.Email.SMTPHost != "" ||
		cfg.Telegram.Enabled || cfg.Telegram.BotToken != "" ||
		cfg.Webhook.Enabled || len(cfg.Webhook.URLs) > 0
}

// getDefaultAlertingConfig возвращает конфигурацию алертинга по умолчанию.
func getDefaultAlertingConfig() *AlertingConfig {
	cfg := alerting.DefaultConfig()
	return &cfg
}

// loadAlertingConfig загружает конфигурацию алертинга из AppConfig, переменных окружения
// или устанавливает значения по умолчанию.
func loadAlertingConfig(l *slog.Logger, cfg *Config) (*AlertingConfig, error) {
	if cfg.AppConfig != nil && isAlertingConfigPresent(&cfg.AppConfig.Alerting) {
		alertingConfig := &cfg.AppConfig.Alerting
		if err := cleanenv.ReadEnv(alertingConfig); err != nil {
			l.Warn("Ошибка загрузки Alerting конфигурации из переменных окружения",
				slog.String("error", err.Error()),
			)
		}
		l.Info("Alerting конфигурация загружена из AppConfig",
			slog.Bool("enabled", alertingConfig.Enabled),
			slog.Bool("email_enabled", alertingConfig.Email.Enabled),
			slog.Bool("telegram_enabled", alertingConfig.Telegram.Enabled),
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
