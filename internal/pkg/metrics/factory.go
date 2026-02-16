package metrics

import (
	"github.com/Kargones/apk-ci/internal/pkg/logging"
)

// NewCollector создаёт Collector на основе конфигурации.
// Если metrics отключены (Config.Enabled = false) — возвращает NopCollector.
// Если включены — возвращает PrometheusCollector.
func NewCollector(config Config, logger logging.Logger) (Collector, error) {
	if !config.Enabled {
		return NewNopCollector(), nil
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return NewPrometheusCollector(config, logger)
}
