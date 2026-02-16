package config

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// TracingConfig содержит настройки OpenTelemetry трейсинга.
type TracingConfig struct {
	// Enabled включает отправку трейсов в OTLP бэкенд.
	Enabled bool `yaml:"enabled" env:"BR_TRACING_ENABLED" env-default:"false"`

	// Endpoint — URL OTLP HTTP endpoint (например, http://jaeger:4318).
	Endpoint string `yaml:"endpoint" env:"BR_TRACING_ENDPOINT"`

	// ServiceName — имя сервиса для resource attributes.
	ServiceName string `yaml:"serviceName" env:"BR_TRACING_SERVICE_NAME" env-default:"apk-ci"`

	// Environment — окружение (production, staging, development).
	Environment string `yaml:"environment" env:"BR_TRACING_ENVIRONMENT" env-default:"production"`

	// Insecure — использовать HTTP вместо HTTPS для OTLP endpoint.
	// L-11/Review #15: По умолчанию true (HTTP) для совместимости с внутренними сетями.
	// Для production deployment через публичные сети установить false (HTTPS).
	Insecure bool `yaml:"insecure" env:"BR_TRACING_INSECURE" env-default:"true"`

	// Timeout — таймаут для экспорта трейсов.
	Timeout time.Duration `yaml:"timeout" env:"BR_TRACING_TIMEOUT" env-default:"5s"`

	// SamplingRate — доля сэмплируемых трейсов (0.0 — ни один, 1.0 — все).
	SamplingRate float64 `yaml:"samplingRate" env:"BR_TRACING_SAMPLING_RATE" env-default:"1.0"`
}
// isTracingConfigPresent проверяет, задана ли конфигурация трейсинга.
// Возвращает true если хотя бы одно значимое поле отличается от zero value.
func isTracingConfigPresent(cfg *TracingConfig) bool {
	if cfg == nil {
		return false
	}
	return cfg.Enabled || cfg.Endpoint != ""
}
// getDefaultTracingConfig возвращает конфигурацию трейсинга по умолчанию.
// Трейсинг отключён по умолчанию (AC5).
func getDefaultTracingConfig() *TracingConfig {
	return &TracingConfig{
		Enabled:      false,
		Endpoint:     "",
		ServiceName:  "apk-ci",
		Environment:  "production",
		Insecure:     true,
		Timeout:      5 * time.Second,
		SamplingRate: 1.0,
	}
}
// validateTracingConfig проверяет корректность конфигурации трейсинга при загрузке.
// Проверяет обязательные поля при включённом трейсинге.
func validateTracingConfig(tc *TracingConfig) error {
	if !tc.Enabled {
		return nil
	}
	if tc.Endpoint == "" {
		return fmt.Errorf("tracing: endpoint обязателен при enabled=true")
	}
	if tc.ServiceName == "" {
		return fmt.Errorf("tracing: service name обязателен при enabled=true")
	}
	if tc.Timeout <= 0 {
		return fmt.Errorf("tracing: timeout должен быть положительным")
	}
	if tc.SamplingRate < 0.0 || tc.SamplingRate > 1.0 {
		// L-12/Review #15: %g вместо %f для читаемого вывода.
		return fmt.Errorf("tracing: sampling rate должен быть от 0.0 до 1.0, получено: %g", tc.SamplingRate)
	}
	return nil
}
// loadTracingConfig загружает конфигурацию трейсинга из AppConfig, переменных окружения или устанавливает значения по умолчанию.
// Переменные окружения BR_TRACING_* переопределяют значения из AppConfig.
func loadTracingConfig(l *slog.Logger, cfg *Config) (*TracingConfig, error) {
	// Проверяем, есть ли конфигурация в AppConfig
	if cfg.AppConfig != nil && isTracingConfigPresent(&cfg.AppConfig.Tracing) {
		tracingConfig := &cfg.AppConfig.Tracing
		// Применяем env override для AppConfig
		if err := cleanenv.ReadEnv(tracingConfig); err != nil {
			l.Warn("Ошибка загрузки Tracing конфигурации из переменных окружения",
				slog.String("error", err.Error()),
			)
		}
		l.Debug("Tracing конфигурация загружена из AppConfig",
			slog.Bool("enabled", tracingConfig.Enabled),
			slog.String("endpoint", tracingConfig.Endpoint),
			slog.String("service_name", tracingConfig.ServiceName),
		)
		return tracingConfig, nil
	}

	tracingConfig := getDefaultTracingConfig()

	if err := cleanenv.ReadEnv(tracingConfig); err != nil {
		l.Warn("Ошибка загрузки Tracing конфигурации из переменных окружения",
			slog.String("error", err.Error()),
		)
	}

	l.Debug("Tracing конфигурация: используются значения по умолчанию",
		slog.Bool("enabled", tracingConfig.Enabled),
	)

	return tracingConfig, nil
}
