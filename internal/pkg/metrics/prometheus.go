package metrics

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/pkg/logging"
	"github.com/Kargones/apk-ci/internal/pkg/urlutil"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

// PrometheusCollector реализует Collector с Prometheus метриками.
// Отправляет метрики в Pushgateway при вызове Push().
type PrometheusCollector struct {
	config   Config
	logger   logging.Logger
	registry *prometheus.Registry

	// Метрики
	commandDuration *prometheus.HistogramVec
	commandSuccess  *prometheus.CounterVec
	commandError    *prometheus.CounterVec

	// Instance label (hostname)
	instance string
}

// NewPrometheusCollector создаёт PrometheusCollector с указанной конфигурацией.
// Регистрирует метрики:
//   - benadis_command_duration_seconds (histogram)
//   - benadis_command_success_total (counter)
//   - benadis_command_error_total (counter)
func NewPrometheusCollector(config Config, logger logging.Logger) (*PrometheusCollector, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	instance := config.InstanceLabel
	if instance == "" {
		hostname, err := os.Hostname()
		if err != nil {
			logger.Warn("не удалось получить hostname для metrics instance label, используется 'unknown'",
				"error", err.Error())
			hostname = "unknown"
		}
		instance = hostname
	}

	registry := prometheus.NewRegistry()

	// Histogram для duration (в секундах)
	// Buckets покрывают диапазон от быстрых команд (0.1s) до очень долгих (10 минут)
	commandDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "benadis",
			Name:      "command_duration_seconds",
			Help:      "Duration of command execution in seconds",
			Buckets:   []float64{0.1, 0.5, 1, 5, 10, 30, 60, 120, 300, 600},
		},
		[]string{"command", "infobase", "status"},
	)

	// Counter для успешных команд.
	// Примечание: success/error counters дублируют histogram counts (duration_seconds_count
	// с label status), но оставлены для удобства — простые PromQL запросы без агрегации
	// по histogram, и для case когда histogram buckets не нужны.
	commandSuccess := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "benadis",
			Name:      "command_success_total",
			Help:      "Total number of successful command executions",
		},
		[]string{"command", "infobase"},
	)

	// Counter для ошибок
	commandError := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "benadis",
			Name:      "command_error_total",
			Help:      "Total number of failed command executions",
		},
		[]string{"command", "infobase"},
	)

	// Регистрируем все метрики атомарно.
	// Используем Register вместо MustRegister для избежания panic.
	// Ошибка возможна только при дублировании имён метрик в одном registry.
	collectors := []prometheus.Collector{commandDuration, commandSuccess, commandError}
	for _, c := range collectors {
		if err := registry.Register(c); err != nil {
			return nil, fmt.Errorf("ошибка регистрации метрики: %w", err)
		}
	}

	return &PrometheusCollector{
		config:          config,
		logger:          logger,
		registry:        registry,
		commandDuration: commandDuration,
		commandSuccess:  commandSuccess,
		commandError:    commandError,
		instance:        instance,
	}, nil
}

// RecordCommandStart записывает начало выполнения команды.
// Для CLI не требуется отслеживать "in-flight" — записываем только при завершении.
func (c *PrometheusCollector) RecordCommandStart(command, infobase string) {
	// No-op для CLI — метрики записываются при завершении
	c.logger.Debug("metrics: command started",
		"command", command,
		"infobase", infobase,
	)
}

// maxLabelLength — максимальная длина значения label для защиты от cardinality explosion.
const maxLabelLength = 128

// sanitizeLabel обрезает значение label до допустимой длины и удаляет
// контрольные символы (\n, \r, \0), которые могут нарушить Prometheus text format.
// Обрезка выполняется по рунам (не по байтам) для корректной работы с UTF-8.
func sanitizeLabel(value string) string {
	// Удаляем контрольные символы, опасные для Prometheus text format
	clean := strings.Map(func(r rune) rune {
		if r < 0x20 { // контрольные символы: \n, \r, \t, \0 и др.
			return '_'
		}
		return r
	}, value)

	runes := []rune(clean)
	if len(runes) > maxLabelLength {
		return string(runes[:maxLabelLength])
	}
	return clean
}

// RecordCommandEnd записывает завершение команды.
// Обновляет histogram duration и counter success/error.
func (c *PrometheusCollector) RecordCommandEnd(command, infobase string, duration time.Duration, success bool) {
	status := "success"
	if !success {
		status = "error"
	}

	// Sanitize labels для защиты от cardinality explosion
	command = sanitizeLabel(command)
	infobase = sanitizeLabel(infobase)

	// Histogram observation
	c.commandDuration.WithLabelValues(command, infobase, status).Observe(duration.Seconds())

	// Counter increment
	if success {
		c.commandSuccess.WithLabelValues(command, infobase).Inc()
	} else {
		c.commandError.WithLabelValues(command, infobase).Inc()
	}

	c.logger.Debug("metrics: command ended",
		"command", command,
		"infobase", infobase,
		"duration_ms", duration.Milliseconds(),
		"success", success,
	)
}

// Push отправляет метрики в Pushgateway.
// Возвращает nil даже при ошибке — ошибки логируются (AC8).
func (c *PrometheusCollector) Push(ctx context.Context) error {
	if c.config.PushgatewayURL == "" {
		c.logger.Debug("metrics: pushgateway URL not configured, skipping push")
		return nil
	}

	// Проверяем контекст
	select {
	case <-ctx.Done():
		c.logger.Debug("metrics push отменён")
		return nil
	default:
	}

	pusher := push.New(c.config.PushgatewayURL, c.config.JobName).
		Gatherer(c.registry).
		Grouping("instance", c.instance)

	// Устанавливаем таймаут через контекст
	pushCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	// Push с контекстом
	if err := pusher.PushContext(pushCtx); err != nil {
		c.logger.Error("ошибка отправки метрик в Pushgateway",
			"error", err.Error(),
			"url", urlutil.MaskURL(c.config.PushgatewayURL),
			"job", c.config.JobName,
		)
		// Возвращаем nil — ошибка метрик не критична (AC8)
		return nil
	}

	c.logger.Info("метрики отправлены в Pushgateway",
		"url", urlutil.MaskURL(c.config.PushgatewayURL),
		"job", c.config.JobName,
		"instance", c.instance,
	)
	return nil
}

// GetRegistry возвращает внутренний registry для тестирования.
// Примечание: экспортируется только для unit-тестов.
func (c *PrometheusCollector) GetRegistry() *prometheus.Registry {
	return c.registry
}

