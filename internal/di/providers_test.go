package di

import (
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/pkg/alerting"
	"github.com/Kargones/apk-ci/internal/pkg/logging"
	"github.com/Kargones/apk-ci/internal/pkg/metrics"
	"github.com/Kargones/apk-ci/internal/pkg/output"
)

// withEnvVar устанавливает переменную окружения и возвращает функцию для восстановления.
// Используется в тестах для изоляции изменений переменных окружения.
func withEnvVar(t *testing.T, key, value string) func() {
	t.Helper()
	origValue, existed := os.LookupEnv(key)

	if value == "" {
		require.NoError(t, _ = os.Unsetenv(key))
	} else {
		require.NoError(t, _ = os.Setenv(key, value))
	}

	return func() {
		if existed {
			_ = os.Setenv(key, origValue)
		} else {
			_ = os.Unsetenv(key)
		}
	}
}

// TestProvideLogger_ReturnsNonNil проверяет, что ProvideLogger возвращает non-nil Logger.
// AC5: Given каждый provider определён, When запускаются unit-тесты,
// Then каждый provider возвращает non-nil значение.
func TestProvideLogger_ReturnsNonNil(t *testing.T) {
	// Arrange
	cfg := &config.Config{
		LoggingConfig: &config.LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}

	// Act
	logger := ProvideLogger(cfg)

	// Assert
	assert.NotNil(t, logger, "ProvideLogger должен возвращать non-nil Logger")
}

// TestProvideLogger_WithNilConfig проверяет работу провайдера при nil Config.
// Должен использовать значения по умолчанию и возвращать non-nil Logger.
func TestProvideLogger_WithNilConfig(t *testing.T) {
	// Arrange - nil config
	var cfg *config.Config

	// Act
	logger := ProvideLogger(cfg)

	// Assert
	assert.NotNil(t, logger, "ProvideLogger должен возвращать non-nil Logger даже при nil Config")
}

// TestProvideLogger_WithNilLoggingConfig проверяет работу при Config без LoggingConfig.
func TestProvideLogger_WithNilLoggingConfig(t *testing.T) {
	// Arrange
	cfg := &config.Config{
		LoggingConfig: nil,
	}

	// Act
	logger := ProvideLogger(cfg)

	// Assert
	assert.NotNil(t, logger, "ProvideLogger должен возвращать non-nil Logger при nil LoggingConfig")
}

// TestProvideOutputWriter_ReturnsNonNil проверяет, что ProvideOutputWriter возвращает non-nil Writer.
// AC5: Given каждый provider определён, When запускаются unit-тесты,
// Then каждый provider возвращает non-nil значение.
func TestProvideOutputWriter_ReturnsNonNil(t *testing.T) {
	// Arrange - очистка переменной окружения
	cleanup := withEnvVar(t, "BR_OUTPUT_FORMAT", "")
	defer cleanup()

	// Act
	writer := ProvideOutputWriter()

	// Assert
	assert.NotNil(t, writer, "ProvideOutputWriter должен возвращать non-nil Writer")
}

// TestProvideOutputWriter_JSONFormat проверяет создание JSONWriter при format="json".
// AC5: TestProvideOutputWriter_JSONFormat — при format="json" возвращает JSONWriter.
func TestProvideOutputWriter_JSONFormat(t *testing.T) {
	// Arrange
	cleanup := withEnvVar(t, "BR_OUTPUT_FORMAT", "json")
	defer cleanup()

	// Act
	writer := ProvideOutputWriter()

	// Assert
	require.NotNil(t, writer, "Writer должен быть non-nil")
	// Проверяем тип через интерфейс — JSONWriter создаётся factory
	// Проверяем что это тот же Writer, что и при прямом создании
	expectedWriter := output.NewWriter("json")
	assert.IsType(t, expectedWriter, writer, "При BR_OUTPUT_FORMAT=json должен создаваться JSONWriter")
}

// TestProvideOutputWriter_TextFormat проверяет создание TextWriter при format="text".
// AC5: TestProvideOutputWriter_TextFormat — при format="text" возвращает TextWriter.
func TestProvideOutputWriter_TextFormat(t *testing.T) {
	// Arrange
	cleanup := withEnvVar(t, "BR_OUTPUT_FORMAT", "text")
	defer cleanup()

	// Act
	writer := ProvideOutputWriter()

	// Assert
	require.NotNil(t, writer, "Writer должен быть non-nil")
	expectedWriter := output.NewWriter("text")
	assert.IsType(t, expectedWriter, writer, "При BR_OUTPUT_FORMAT=text должен создаваться TextWriter")
}

// TestProvideOutputWriter_DefaultFormat проверяет создание TextWriter при пустом формате.
func TestProvideOutputWriter_DefaultFormat(t *testing.T) {
	// Arrange
	cleanup := withEnvVar(t, "BR_OUTPUT_FORMAT", "")
	defer cleanup()

	// Act
	writer := ProvideOutputWriter()

	// Assert
	require.NotNil(t, writer, "Writer должен быть non-nil")
	expectedWriter := output.NewWriter("text")
	assert.IsType(t, expectedWriter, writer, "По умолчанию должен создаваться TextWriter")
}

// TestProvideTraceID_ReturnsNonEmpty проверяет, что ProvideTraceID возвращает непустую строку.
func TestProvideTraceID_ReturnsNonEmpty(t *testing.T) {
	// Act
	traceID := ProvideTraceID()

	// Assert
	assert.NotEmpty(t, traceID, "ProvideTraceID должен возвращать непустой trace_id")
}

// TestProvideTraceID_ValidFormat проверяет формат trace_id (32-hex chars).
// AC5: TestProvideTraceID_ReturnsValidFormat — trace_id в формате 32-hex chars.
func TestProvideTraceID_ValidFormat(t *testing.T) {
	// Arrange
	hexPattern := regexp.MustCompile(`^[0-9a-f]{32}$`)

	// Act
	traceID := ProvideTraceID()

	// Assert
	assert.Len(t, traceID, 32, "trace_id должен содержать 32 символа")
	assert.Regexp(t, hexPattern, traceID, "trace_id должен содержать только hex символы")
}

// TestProvideTraceID_Uniqueness проверяет уникальность генерируемых trace_id.
func TestProvideTraceID_Uniqueness(t *testing.T) {
	// Arrange
	const iterations = 100
	traceIDs := make(map[string]bool, iterations)

	// Act
	for range iterations {
		traceID := ProvideTraceID()
		traceIDs[traceID] = true
	}

	// Assert
	assert.Len(t, traceIDs, iterations, "Все trace_id должны быть уникальными")
}

// TestInitializeApp_AllFieldsNonNil проверяет инициализацию App со всеми non-nil полями.
// AC2: Given провайдеры для Config, Logger, OutputWriter определены,
// When вызывается NewApp(), Then возвращается инициализированный App struct с non-nil зависимостями.
func TestInitializeApp_AllFieldsNonNil(t *testing.T) {
	// Arrange
	cfg := &config.Config{
		LoggingConfig: &config.LoggingConfig{
			Level:  "debug",
			Format: "json",
		},
	}

	// Act
	app, err := InitializeApp(cfg)

	// Assert
	require.NoError(t, err, "InitializeApp не должен возвращать ошибку")
	require.NotNil(t, app, "InitializeApp должен возвращать non-nil App")

	assert.NotNil(t, app.Config, "App.Config должен быть non-nil")
	assert.Same(t, cfg, app.Config, "App.Config должен быть тем же объектом, что передан в InitializeApp")

	assert.NotNil(t, app.Logger, "App.Logger должен быть non-nil")
	assert.NotNil(t, app.OutputWriter, "App.OutputWriter должен быть non-nil")
	assert.NotEmpty(t, app.TraceID, "App.TraceID должен быть непустым")
}

// M-3 fix: Тесты для ProvideAlerter.

// TestProvideAlerter_NilConfig проверяет что nil Config возвращает NopAlerter.
func TestProvideAlerter_NilConfig(t *testing.T) {
	logger := logging.NewLogger(logging.DefaultConfig())
	result := ProvideAlerter(nil, logger)

	assert.NotNil(t, result)
	_, ok := result.(*alerting.NopAlerter)
	assert.True(t, ok, "при nil Config должен возвращаться NopAlerter")
}

// TestProvideAlerter_NilAlertingConfig проверяет что nil AlertingConfig возвращает NopAlerter.
func TestProvideAlerter_NilAlertingConfig(t *testing.T) {
	cfg := &config.Config{AlertingConfig: nil}
	logger := logging.NewLogger(logging.DefaultConfig())
	result := ProvideAlerter(cfg, logger)

	assert.NotNil(t, result)
	_, ok := result.(*alerting.NopAlerter)
	assert.True(t, ok, "при nil AlertingConfig должен возвращаться NopAlerter")
}

// TestProvideAlerter_DisabledReturnsNop проверяет что Enabled=false возвращает NopAlerter.
func TestProvideAlerter_DisabledReturnsNop(t *testing.T) {
	cfg := &config.Config{
		AlertingConfig: &config.AlertingConfig{
			Enabled: false,
		},
	}
	logger := logging.NewLogger(logging.DefaultConfig())
	result := ProvideAlerter(cfg, logger)

	assert.NotNil(t, result)
	_, ok := result.(*alerting.NopAlerter)
	assert.True(t, ok, "при Enabled=false должен возвращаться NopAlerter")
}

// TestProvideAlerter_EnabledEmailReturnsMultiChannel проверяет маппинг полей конфигурации.
func TestProvideAlerter_EnabledEmailReturnsMultiChannel(t *testing.T) {
	cfg := &config.Config{
		AlertingConfig: &config.AlertingConfig{
			Enabled:         true,
			RateLimitWindow: 10 * time.Minute,
			Email: config.EmailChannelConfig{
				Enabled:  true,
				SMTPHost: "smtp.test.com",
				SMTPPort: 587,
				From:     "test@test.com",
				To:       []string{"devops@test.com"},
			},
		},
	}
	logger := logging.NewLogger(logging.DefaultConfig())
	result := ProvideAlerter(cfg, logger)

	assert.NotNil(t, result)
	_, ok := result.(*alerting.MultiChannelAlerter)
	assert.True(t, ok, "при Enabled=true с email каналом должен возвращаться MultiChannelAlerter")
}

// TestProvideAlerter_ValidationErrorReturnsNop проверяет что ошибка валидации возвращает NopAlerter.
func TestProvideAlerter_ValidationErrorReturnsNop(t *testing.T) {
	cfg := &config.Config{
		AlertingConfig: &config.AlertingConfig{
			Enabled: true,
			Email: config.EmailChannelConfig{
				Enabled:  true,
				SMTPHost: "", // Missing — вызовет ошибку валидации
				From:     "test@test.com",
				To:       []string{"devops@test.com"},
			},
		},
	}
	logger := logging.NewLogger(logging.DefaultConfig())
	result := ProvideAlerter(cfg, logger)

	assert.NotNil(t, result)
	_, ok := result.(*alerting.NopAlerter)
	assert.True(t, ok, "при ошибке валидации должен возвращаться NopAlerter")
}

// M-3 fix: Тесты для ProvideMetricsCollector.

// TestProvideMetricsCollector_NilConfig проверяет что nil Config возвращает NopCollector.
func TestProvideMetricsCollector_NilConfig(t *testing.T) {
	logger := logging.NewLogger(logging.DefaultConfig())
	result := ProvideMetricsCollector(nil, logger)

	assert.NotNil(t, result)
	_, ok := result.(*metrics.NopCollector)
	assert.True(t, ok, "при nil Config должен возвращаться NopCollector")
}

// TestProvideMetricsCollector_NilMetricsConfig проверяет что nil MetricsConfig возвращает NopCollector.
func TestProvideMetricsCollector_NilMetricsConfig(t *testing.T) {
	cfg := &config.Config{MetricsConfig: nil}
	logger := logging.NewLogger(logging.DefaultConfig())
	result := ProvideMetricsCollector(cfg, logger)

	assert.NotNil(t, result)
	_, ok := result.(*metrics.NopCollector)
	assert.True(t, ok, "при nil MetricsConfig должен возвращаться NopCollector")
}

// TestProvideMetricsCollector_DisabledReturnsNop проверяет что Enabled=false возвращает NopCollector.
func TestProvideMetricsCollector_DisabledReturnsNop(t *testing.T) {
	cfg := &config.Config{
		MetricsConfig: &config.MetricsConfig{
			Enabled: false,
		},
	}
	logger := logging.NewLogger(logging.DefaultConfig())
	result := ProvideMetricsCollector(cfg, logger)

	assert.NotNil(t, result)
	_, ok := result.(*metrics.NopCollector)
	assert.True(t, ok, "при Enabled=false должен возвращаться NopCollector")
}

// TestInitializeApp_TraceIDFormat проверяет формат TraceID в инициализированном App.
func TestInitializeApp_TraceIDFormat(t *testing.T) {
	// Arrange
	hexPattern := regexp.MustCompile(`^[0-9a-f]{32}$`)
	cfg := &config.Config{}

	// Act
	app, err := InitializeApp(cfg)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, app)
	assert.Regexp(t, hexPattern, app.TraceID, "App.TraceID должен быть в формате 32-hex chars")
}
