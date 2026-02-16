package config

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/Kargones/apk-ci/internal/pkg/urlutil"
	"github.com/ilyakaznacheev/cleanenv"
)

// MetricsConfig содержит настройки для Prometheus метрик.
type MetricsConfig struct {
	// Enabled — включены ли метрики (по умолчанию false).
	Enabled bool `yaml:"enabled" env:"BR_METRICS_ENABLED" env-default:"false"`

	// PushgatewayURL — URL Prometheus Pushgateway.
	// Пример: "http://pushgateway:9091"
	PushgatewayURL string `yaml:"pushgatewayUrl" env:"BR_METRICS_PUSHGATEWAY_URL"`

	// JobName — имя job для группировки метрик.
	// По умолчанию: "apk-ci"
	JobName string `yaml:"jobName" env:"BR_METRICS_JOB_NAME" env-default:"apk-ci"`

	// Timeout — таймаут HTTP запросов к Pushgateway.
	// По умолчанию: 10 секунд.
	Timeout time.Duration `yaml:"timeout" env:"BR_METRICS_TIMEOUT" env-default:"10s"`

	// InstanceLabel — переопределение instance label.
	// Если пусто — используется hostname.
	InstanceLabel string `yaml:"instanceLabel" env:"BR_METRICS_INSTANCE"`
}
// isMetricsConfigPresent проверяет, задана ли конфигурация метрик.
// Возвращает true если хотя бы одно значимое поле отличается от zero value.
func isMetricsConfigPresent(cfg *MetricsConfig) bool {
	if cfg == nil {
		return false
	}
	// Проверяем любое значимое поле (enabled, или pushgateway URL)
	return cfg.Enabled || cfg.PushgatewayURL != ""
}
// getDefaultMetricsConfig возвращает конфигурацию метрик по умолчанию.
// Метрики отключены по умолчанию (AC6).
func getDefaultMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		Enabled:        false,
		PushgatewayURL: "",
		JobName:        "apk-ci",
		Timeout:        10 * time.Second,
		InstanceLabel:  "",
	}
}
// loadMetricsConfig загружает конфигурацию метрик из AppConfig, переменных окружения или устанавливает значения по умолчанию.
// Переменные окружения BR_METRICS_* переопределяют значения из AppConfig.
func loadMetricsConfig(l *slog.Logger, cfg *Config) (*MetricsConfig, error) {
	// Проверяем, есть ли конфигурация в AppConfig
	if cfg.AppConfig != nil && isMetricsConfigPresent(&cfg.AppConfig.Metrics) {
		metricsConfig := &cfg.AppConfig.Metrics
		// Применяем env override для AppConfig (симметрично с loadAlertingConfig)
		if err := cleanenv.ReadEnv(metricsConfig); err != nil {
			l.Warn("Ошибка загрузки Metrics конфигурации из переменных окружения",
				slog.String("error", err.Error()),
			)
		}
		l.Info("Metrics конфигурация загружена из AppConfig",
			slog.Bool("enabled", metricsConfig.Enabled),
			slog.String("pushgateway_url", urlutil.MaskURL(metricsConfig.PushgatewayURL)),
			slog.String("job_name", metricsConfig.JobName),
		)
		return metricsConfig, nil
	}

	metricsConfig := getDefaultMetricsConfig()

	if err := cleanenv.ReadEnv(metricsConfig); err != nil {
		l.Warn("Ошибка загрузки Metrics конфигурации из переменных окружения",
			slog.String("error", err.Error()),
		)
	}

	l.Debug("Metrics конфигурация: используются значения по умолчанию",
		slog.Bool("enabled", metricsConfig.Enabled),
	)

	return metricsConfig, nil
}
// validateMetricsConfig проверяет корректность конфигурации метрик при загрузке.
// Проверяет обязательные поля при включённых метриках.
func validateMetricsConfig(mc *MetricsConfig) error {
	if !mc.Enabled {
		return nil
	}
	if mc.PushgatewayURL == "" {
		return fmt.Errorf("metrics: pushgateway_url обязателен при enabled=true")
	}
	if mc.Timeout <= 0 {
		return fmt.Errorf("metrics: timeout должен быть положительным")
	}
	return nil
}
