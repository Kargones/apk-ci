package metrics

import (
	"net/url"
	"time"
)

// Config содержит настройки для сбора и отправки Prometheus метрик.
type Config struct {
	// Enabled — включены ли метрики (по умолчанию false).
	Enabled bool

	// PushgatewayURL — URL Prometheus Pushgateway.
	// Пример: "http://pushgateway:9091"
	PushgatewayURL string

	// JobName — имя job для группировки метрик.
	// По умолчанию: "apk-ci"
	JobName string

	// Timeout — таймаут HTTP запросов к Pushgateway.
	// По умолчанию: 10 секунд.
	Timeout time.Duration

	// InstanceLabel — переопределение instance label.
	// Если пусто — используется hostname.
	InstanceLabel string
}

// Validate проверяет корректность конфигурации.
// Возвращает ошибку если конфигурация невалидна.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil // отключённые метрики валидны
	}

	if c.PushgatewayURL == "" {
		return ErrPushgatewayURLRequired
	}

	// Валидация формата URL
	u, err := url.Parse(c.PushgatewayURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ErrPushgatewayURLInvalid
	}

	if c.JobName == "" {
		return ErrJobNameRequired
	}

	if c.Timeout <= 0 {
		return ErrInvalidTimeout
	}

	return nil
}

// DefaultConfig возвращает конфигурацию по умолчанию.
func DefaultConfig() Config {
	return Config{
		Enabled:        false,
		PushgatewayURL: "",
		JobName:        "apk-ci",
		Timeout:        10 * time.Second,
		InstanceLabel:  "",
	}
}
