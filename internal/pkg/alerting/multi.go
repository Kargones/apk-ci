package alerting

import (
	"context"
	"sort"

	"github.com/Kargones/apk-ci/internal/pkg/logging"
)

// MultiChannelAlerter отправляет алерты через несколько каналов с фильтрацией по правилам.
type MultiChannelAlerter struct {
	channels     map[string]Alerter
	channelNames []string // отсортированные имена каналов для детерминистичного порядка
	rules        *RulesEngine
	rateLimiter  *RateLimiter // rate limiter на уровне multi-channel (общий для всех каналов)
	logger       logging.Logger
}

// NewMultiChannelAlerter создаёт alerter с несколькими каналами и правилами фильтрации.
// rateLimiter применяется ОДИН РАЗ перед отправкой во все каналы (а не в каждом канале отдельно).
func NewMultiChannelAlerter(channels map[string]Alerter, rules *RulesEngine, rateLimiter *RateLimiter, logger logging.Logger) *MultiChannelAlerter {
	// Сортируем имена каналов для детерминистичного порядка отправки
	names := make([]string, 0, len(channels))
	for name := range channels {
		names = append(names, name)
	}
	sort.Strings(names)

	return &MultiChannelAlerter{
		channels:     channels,
		channelNames: names,
		rules:        rules,
		rateLimiter:  rateLimiter,
		logger:       logger,
	}
}

// Send отправляет алерт через все настроенные каналы, проверяя правила.
// Rate limiting проверяется ОДИН РАЗ на уровне MultiChannelAlerter — если алерт подавлен,
// он не отправляется ни в один канал. Это предотвращает ситуацию, когда первый канал
// "съедает" rate limit и остальные каналы не получают алерт.
// Каналы обрабатываются в алфавитном порядке для предсказуемого поведения.
// Ошибки логируются внутри каждого канала, этот метод всегда возвращает nil.
func (m *MultiChannelAlerter) Send(ctx context.Context, alert Alert) error {
	// Rate limiting на уровне multi-channel — единая проверка для всех каналов
	if m.rateLimiter != nil && !m.rateLimiter.Allow(alert.ErrorCode) {
		m.logger.Debug("алерт подавлен rate limiter",
			"error_code", alert.ErrorCode,
		)
		return nil
	}

	sentCount := 0
	skippedCount := 0
	for _, name := range m.channelNames {
		// Проверяем контекст перед отправкой в каждый канал
		select {
		case <-ctx.Done():
			return nil // Отмена — не ошибка
		default:
		}

		ch := m.channels[name]

		// Проверяем правила фильтрации для каждого канала
		if m.rules != nil && !m.rules.Evaluate(alert, name) {
			m.logger.Debug("алерт отклонён правилами",
				"channel", name,
				"error_code", alert.ErrorCode,
				"command", alert.Command,
				"severity", alert.Severity.String(),
			)
			skippedCount++
			continue
		}

		_ = ch.Send(ctx, alert) // Ошибки логируются внутри каждого канала
		sentCount++
	}

	// M-3/Review #10: итоговый лог о результате рассылки по каналам.
	if sentCount > 0 {
		m.logger.Debug("multi-channel рассылка завершена",
			"error_code", alert.ErrorCode,
			"channels_sent", sentCount,
			"channels_skipped", skippedCount,
			"channels_total", len(m.channelNames),
		)
	}
	return nil
}
