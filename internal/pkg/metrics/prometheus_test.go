package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/pkg/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPrometheusCollector_RecordCommand проверяет запись метрик (AC2, AC3).
func TestPrometheusCollector_RecordCommand(t *testing.T) {
	config := Config{
		Enabled:        true,
		PushgatewayURL: "http://localhost:9091",
		JobName:        "test-job",
		Timeout:        10 * time.Second,
	}

	logger := logging.NewNopLogger()
	collector, err := NewPrometheusCollector(config, logger)
	require.NoError(t, err)

	// Записываем начало команды (no-op для CLI)
	collector.RecordCommandStart("service-mode-status", "TestDB")

	// Записываем завершение успешной команды
	collector.RecordCommandEnd("service-mode-status", "TestDB", 1500*time.Millisecond, true)

	// Проверяем метрики
	registry := collector.GetRegistry()
	metrics, err := registry.Gather()
	require.NoError(t, err)

	// Проверяем наличие метрик
	found := make(map[string]bool)
	for _, m := range metrics {
		found[m.GetName()] = true
	}

	assert.True(t, found["apk_ci_command_duration_seconds"], "должен быть histogram duration")
	assert.True(t, found["apk_ci_command_success_total"], "должен быть counter success")
}

// TestPrometheusCollector_Push проверяет отправку метрик (AC4).
func TestPrometheusCollector_Push(t *testing.T) {
	// Mock Pushgateway
	var receivedMethod string
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := Config{
		Enabled:        true,
		PushgatewayURL: server.URL,
		JobName:        "apk-ci",
		Timeout:        10 * time.Second,
	}

	logger := logging.NewNopLogger()
	collector, err := NewPrometheusCollector(config, logger)
	require.NoError(t, err)

	// Записываем метрики
	collector.RecordCommandEnd("service-mode-status", "TestDB", 1500*time.Millisecond, true)

	// Push
	err = collector.Push(context.Background())
	assert.NoError(t, err)

	// Проверяем что запрос отправлен
	// Prometheus Pushgateway использует PUT для push операций
	assert.Equal(t, http.MethodPut, receivedMethod)
	assert.Contains(t, receivedPath, "/metrics/job/apk-ci")
}

// TestPrometheusCollector_Disabled проверяет NopCollector (AC6).
func TestPrometheusCollector_Disabled(t *testing.T) {
	config := Config{
		Enabled: false,
	}

	logger := logging.NewNopLogger()
	collector, err := NewCollector(config, logger)
	require.NoError(t, err)

	// Проверяем что это NopCollector
	_, isNop := collector.(*NopCollector)
	assert.True(t, isNop, "при disabled должен быть NopCollector")

	// NopCollector должен работать без ошибок
	collector.RecordCommandStart("test", "db")
	collector.RecordCommandEnd("test", "db", time.Second, true)
	err = collector.Push(context.Background())
	assert.NoError(t, err)
}

// TestPrometheusCollector_PushError проверяет обработку ошибок (AC8).
func TestPrometheusCollector_PushError(t *testing.T) {
	// Mock Pushgateway с ошибкой
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := Config{
		Enabled:        true,
		PushgatewayURL: server.URL,
		JobName:        "apk-ci",
		Timeout:        10 * time.Second,
	}

	logger := logging.NewNopLogger()
	collector, err := NewPrometheusCollector(config, logger)
	require.NoError(t, err)

	// Push — должен вернуть nil даже при ошибке (AC8)
	err = collector.Push(context.Background())
	assert.NoError(t, err, "Push должен возвращать nil даже при ошибке")
}

// TestPrometheusCollector_Labels проверяет labels (AC3).
func TestPrometheusCollector_Labels(t *testing.T) {
	config := Config{
		Enabled:        true,
		PushgatewayURL: "http://localhost:9091",
		JobName:        "test-job",
		Timeout:        10 * time.Second,
	}

	logger := logging.NewNopLogger()
	collector, err := NewPrometheusCollector(config, logger)
	require.NoError(t, err)

	// Записываем команду
	collector.RecordCommandEnd("service-mode-status", "TestDB", 1500*time.Millisecond, true)

	// Проверяем labels
	registry := collector.GetRegistry()
	metrics, err := registry.Gather()
	require.NoError(t, err)

	for _, m := range metrics {
		if m.GetName() == "apk_ci_command_duration_seconds" {
			for _, metric := range m.GetMetric() {
				labels := make(map[string]string)
				for _, l := range metric.GetLabel() {
					labels[l.GetName()] = l.GetValue()
				}
				assert.Equal(t, "service-mode-status", labels["command"])
				assert.Equal(t, "TestDB", labels["infobase"])
				assert.Equal(t, "success", labels["status"])
			}
		}
	}
}

// TestPrometheusCollector_InstanceLabel проверяет hostname resolution (AC10).
func TestPrometheusCollector_InstanceLabel(t *testing.T) {
	t.Run("with custom instance label", func(t *testing.T) {
		config := Config{
			Enabled:        true,
			PushgatewayURL: "http://localhost:9091",
			JobName:        "test-job",
			Timeout:        10 * time.Second,
			InstanceLabel:  "custom-instance",
		}

		logger := logging.NewNopLogger()
		collector, err := NewPrometheusCollector(config, logger)
		require.NoError(t, err)

		assert.Equal(t, "custom-instance", collector.instance)
	})

	t.Run("without instance label uses hostname", func(t *testing.T) {
		config := Config{
			Enabled:        true,
			PushgatewayURL: "http://localhost:9091",
			JobName:        "test-job",
			Timeout:        10 * time.Second,
			InstanceLabel:  "",
		}

		logger := logging.NewNopLogger()
		collector, err := NewPrometheusCollector(config, logger)
		require.NoError(t, err)

		// Instance должен быть hostname или "unknown"
		assert.NotEmpty(t, collector.instance)
	})
}

// TestMetricsConfig_Validate проверяет валидацию конфигурации (AC5).
func TestMetricsConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr error
	}{
		{
			name: "valid config",
			config: Config{
				Enabled:        true,
				PushgatewayURL: "http://localhost:9091",
				JobName:        "test",
				Timeout:        10 * time.Second,
			},
			wantErr: nil,
		},
		{
			name: "disabled config is always valid",
			config: Config{
				Enabled: false,
			},
			wantErr: nil,
		},
		{
			name: "missing pushgateway URL",
			config: Config{
				Enabled:        true,
				PushgatewayURL: "",
				JobName:        "test",
				Timeout:        10 * time.Second,
			},
			wantErr: ErrPushgatewayURLRequired,
		},
		{
			name: "missing job name",
			config: Config{
				Enabled:        true,
				PushgatewayURL: "http://localhost:9091",
				JobName:        "",
				Timeout:        10 * time.Second,
			},
			wantErr: ErrJobNameRequired,
		},
		{
			name: "invalid timeout",
			config: Config{
				Enabled:        true,
				PushgatewayURL: "http://localhost:9091",
				JobName:        "test",
				Timeout:        0,
			},
			wantErr: ErrInvalidTimeout,
		},
		{
			name: "negative timeout",
			config: Config{
				Enabled:        true,
				PushgatewayURL: "http://localhost:9091",
				JobName:        "test",
				Timeout:        -5 * time.Second,
			},
			wantErr: ErrInvalidTimeout,
		},
		{
			name: "invalid URL format - no scheme",
			config: Config{
				Enabled:        true,
				PushgatewayURL: "localhost:9091",
				JobName:        "test",
				Timeout:        10 * time.Second,
			},
			wantErr: ErrPushgatewayURLInvalid,
		},
		{
			name: "invalid URL format - no host",
			config: Config{
				Enabled:        true,
				PushgatewayURL: "http://",
				JobName:        "test",
				Timeout:        10 * time.Second,
			},
			wantErr: ErrPushgatewayURLInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPrometheusCollector_ErrorStatus проверяет запись ошибок (AC2).
func TestPrometheusCollector_ErrorStatus(t *testing.T) {
	config := Config{
		Enabled:        true,
		PushgatewayURL: "http://localhost:9091",
		JobName:        "test-job",
		Timeout:        10 * time.Second,
	}

	logger := logging.NewNopLogger()
	collector, err := NewPrometheusCollector(config, logger)
	require.NoError(t, err)

	// Записываем ошибочную команду
	collector.RecordCommandEnd("dbrestore", "ProdDB", 5*time.Second, false)

	// Проверяем метрики
	registry := collector.GetRegistry()
	metrics, err := registry.Gather()
	require.NoError(t, err)

	// Проверяем error counter
	for _, m := range metrics {
		if m.GetName() == "apk_ci_command_error_total" {
			for _, metric := range m.GetMetric() {
				counter := metric.GetCounter()
				assert.Equal(t, float64(1), counter.GetValue())
			}
		}
		if m.GetName() == "apk_ci_command_duration_seconds" {
			for _, metric := range m.GetMetric() {
				labels := make(map[string]string)
				for _, l := range metric.GetLabel() {
					labels[l.GetName()] = l.GetValue()
				}
				assert.Equal(t, "error", labels["status"])
			}
		}
	}
}

// TestPrometheusCollector_ContextCancellation проверяет отмену контекста.
func TestPrometheusCollector_ContextCancellation(t *testing.T) {
	config := Config{
		Enabled:        true,
		PushgatewayURL: "http://localhost:9091",
		JobName:        "test-job",
		Timeout:        10 * time.Second,
	}

	logger := logging.NewNopLogger()
	collector, err := NewPrometheusCollector(config, logger)
	require.NoError(t, err)

	// Создаём отменённый контекст
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Push должен вернуть nil при отменённом контексте
	err = collector.Push(ctx)
	assert.NoError(t, err)
}

// TestNopCollector проверяет NopCollector.
func TestNopCollector(t *testing.T) {
	collector := NewNopCollector()

	// Все методы должны работать без паники
	collector.RecordCommandStart("test", "db")
	collector.RecordCommandEnd("test", "db", time.Second, true)
	err := collector.Push(context.Background())
	assert.NoError(t, err)
}

// TestNewCollector_Factory проверяет factory функцию.
func TestNewCollector_Factory(t *testing.T) {
	t.Run("disabled returns NopCollector", func(t *testing.T) {
		config := Config{Enabled: false}
		logger := logging.NewNopLogger()

		collector, err := NewCollector(config, logger)
		require.NoError(t, err)

		_, isNop := collector.(*NopCollector)
		assert.True(t, isNop)
	})

	t.Run("enabled returns PrometheusCollector", func(t *testing.T) {
		config := Config{
			Enabled:        true,
			PushgatewayURL: "http://localhost:9091",
			JobName:        "test",
			Timeout:        10 * time.Second,
		}
		logger := logging.NewNopLogger()

		collector, err := NewCollector(config, logger)
		require.NoError(t, err)

		_, isProm := collector.(*PrometheusCollector)
		assert.True(t, isProm)
	})

	t.Run("invalid config returns error", func(t *testing.T) {
		config := Config{
			Enabled:        true,
			PushgatewayURL: "", // missing
			JobName:        "test",
			Timeout:        10 * time.Second,
		}
		logger := logging.NewNopLogger()

		_, err := NewCollector(config, logger)
		assert.Error(t, err)
	})
}

// TestPrometheusCollector_PushWithoutURL проверяет push без URL.
func TestPrometheusCollector_PushWithoutURL(t *testing.T) {
	// Создаём collector напрямую без валидации (для теста edge case)
	config := Config{
		Enabled:        true,
		PushgatewayURL: "http://test:9091", // нужен для создания
		JobName:        "test-job",
		Timeout:        10 * time.Second,
	}

	logger := logging.NewNopLogger()
	collector, err := NewPrometheusCollector(config, logger)
	require.NoError(t, err)

	// Очищаем URL после создания
	collector.config.PushgatewayURL = ""

	// Push должен пропустить отправку
	err = collector.Push(context.Background())
	assert.NoError(t, err)
}

// TestPrometheusCollector_MultipleRecords проверяет множественные записи.
func TestPrometheusCollector_MultipleRecords(t *testing.T) {
	config := Config{
		Enabled:        true,
		PushgatewayURL: "http://localhost:9091",
		JobName:        "test-job",
		Timeout:        10 * time.Second,
	}

	logger := logging.NewNopLogger()
	collector, err := NewPrometheusCollector(config, logger)
	require.NoError(t, err)

	// Записываем несколько команд
	collector.RecordCommandEnd("cmd1", "db1", 1*time.Second, true)
	collector.RecordCommandEnd("cmd1", "db1", 2*time.Second, true)
	collector.RecordCommandEnd("cmd2", "db2", 3*time.Second, false)

	// Проверяем метрики
	registry := collector.GetRegistry()
	metrics, err := registry.Gather()
	require.NoError(t, err)

	// Должны быть метрики для обеих комбинаций команда+база
	var successCount, errorCount float64
	for _, m := range metrics {
		if m.GetName() == "apk_ci_command_success_total" {
			for _, metric := range m.GetMetric() {
				successCount += metric.GetCounter().GetValue()
			}
		}
		if m.GetName() == "apk_ci_command_error_total" {
			for _, metric := range m.GetMetric() {
				errorCount += metric.GetCounter().GetValue()
			}
		}
	}

	assert.Equal(t, float64(2), successCount, "должно быть 2 успешных вызова")
	assert.Equal(t, float64(1), errorCount, "должен быть 1 ошибочный вызов")
}

// TestPrometheusCollector_HistogramBuckets проверяет bucket'ы гистограммы.
func TestPrometheusCollector_HistogramBuckets(t *testing.T) {
	config := Config{
		Enabled:        true,
		PushgatewayURL: "http://localhost:9091",
		JobName:        "test-job",
		Timeout:        10 * time.Second,
	}

	logger := logging.NewNopLogger()
	collector, err := NewPrometheusCollector(config, logger)
	require.NoError(t, err)

	// Записываем команды с разной длительностью
	collector.RecordCommandEnd("fast", "db", 50*time.Millisecond, true)  // < 0.1s
	collector.RecordCommandEnd("slow", "db", 45*time.Second, true)       // 30-60s bucket
	collector.RecordCommandEnd("very_slow", "db", 5*time.Minute, true)   // 300s bucket

	// Проверяем что histogram содержит все записи
	registry := collector.GetRegistry()
	metrics, err := registry.Gather()
	require.NoError(t, err)

	var histogramFound bool
	for _, m := range metrics {
		if m.GetName() == "apk_ci_command_duration_seconds" {
			histogramFound = true
			// Должно быть 3 записи суммарно
			var totalCount uint64
			for _, metric := range m.GetMetric() {
				histogram := metric.GetHistogram()
				if histogram != nil {
					totalCount += histogram.GetSampleCount()
				}
			}
			assert.Equal(t, uint64(3), totalCount)
		}
	}
	assert.True(t, histogramFound, "histogram должен присутствовать")
}

func TestSanitizeLabel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "короткое значение — без изменений",
			input:    "service-mode-status",
			expected: "service-mode-status",
		},
		{
			name:     "пустая строка — без изменений",
			input:    "",
			expected: "",
		},
		{
			name:     "ровно 128 символов — без изменений",
			input:    strings.Repeat("a", maxLabelLength),
			expected: strings.Repeat("a", maxLabelLength),
		},
		{
			name:     "длинное значение — обрезается до 128",
			input:    strings.Repeat("x", 256),
			expected: strings.Repeat("x", maxLabelLength),
		},
		{
			name:     "кириллица — обрезка по рунам, не по байтам",
			input:    strings.Repeat("Б", 200), // 200 рун × 2 байта = 400 байт
			expected: strings.Repeat("Б", maxLabelLength),
		},
		{
			name:     "контрольные символы заменяются на underscore",
			input:    "command\nwith\rnewlines\x00null",
			expected: "command_with_newlines_null",
		},
		{
			name:     "tab заменяется на underscore",
			input:    "value\twith\ttabs",
			expected: "value_with_tabs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeLabel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
