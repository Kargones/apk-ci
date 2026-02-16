package tracing

import (
	"errors"
	"fmt"
	"net/url"
	"time"
)

// Ошибки валидации конфигурации трейсинга.
var (
	// ErrTracingEndpointRequired — endpoint обязателен при включённом трейсинге.
	ErrTracingEndpointRequired = errors.New("tracing: endpoint обязателен когда tracing включён")

	// ErrTracingServiceNameRequired — service name обязателен.
	ErrTracingServiceNameRequired = errors.New("tracing: service name обязателен")

	// ErrTracingTimeoutInvalid — timeout должен быть положительным.
	ErrTracingTimeoutInvalid = errors.New("tracing: timeout должен быть положительным")

	// ErrTracingEndpointInvalidFormat — endpoint имеет невалидный URL формат.
	ErrTracingEndpointInvalidFormat = errors.New("tracing: endpoint должен быть валидным URL с host (например http://jaeger:4318)")

	// ErrTracingSamplingRateInvalid — sampling rate вне допустимого диапазона [0.0, 1.0].
	// M-7/Review #16: Sentinel error для программной проверки через errors.Is().
	ErrTracingSamplingRateInvalid = errors.New("tracing: sampling rate должен быть от 0.0 до 1.0")
)

// Config содержит настройки для инициализации TracerProvider.
type Config struct {
	// Enabled — включён ли трейсинг.
	Enabled bool

	// Endpoint — URL OTLP HTTP endpoint (например, "jaeger:4318").
	Endpoint string

	// ServiceName — имя сервиса для resource attributes.
	ServiceName string

	// Version — версия сервиса для resource attributes.
	Version string

	// Environment — окружение (production, staging, development).
	Environment string

	// Insecure — использовать HTTP вместо HTTPS.
	// L-11/Review #15: По умолчанию true для совместимости с внутренними сетями.
	// Для production через публичные сети установить false.
	Insecure bool

	// Timeout — таймаут для экспорта трейсов.
	Timeout time.Duration

	// SamplingRate — доля сэмплируемых трейсов (0.0 — ни один, 1.0 — все).
	SamplingRate float64
}

// Validate проверяет корректность конфигурации.
// L-10/Review #15: Используем sentinel errors для проверки через errors.Is().
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.Endpoint == "" {
		return ErrTracingEndpointRequired
	}
	// M-9/Review #15: Проверяем формат endpoint URL.
	if u, err := url.Parse(c.Endpoint); err != nil || u.Host == "" {
		return ErrTracingEndpointInvalidFormat
	}
	if c.ServiceName == "" {
		return ErrTracingServiceNameRequired
	}
	if c.Timeout <= 0 {
		return ErrTracingTimeoutInvalid
	}
	if c.SamplingRate < 0.0 || c.SamplingRate > 1.0 {
		// M-7/Review #16: sentinel error + wrap с конкретным значением для диагностики.
		// L-12/Review #15: %g вместо %f для читаемого вывода (0.5 вместо 0.500000).
		return fmt.Errorf("%w, получено: %g", ErrTracingSamplingRateInvalid, c.SamplingRate)
	}
	return nil
}

// DefaultConfig возвращает конфигурацию по умолчанию (трейсинг выключен).
func DefaultConfig() Config {
	return Config{
		Enabled:      false,
		ServiceName:  "apk-ci",
		Environment:  "production",
		// Review #36: default=false (secure) — production deployments should use TLS by default.
		Insecure:     false,
		Timeout:      5 * time.Second,
		SamplingRate: 1.0,
	}
}
