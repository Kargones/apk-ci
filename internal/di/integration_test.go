package di

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/pkg/output"
)

// TestInitializeApp_FullPipeline проверяет полный цикл инициализации App.
// AC2: Given провайдеры для Config, Logger, OutputWriter определены,
// When вызывается NewApp(), Then возвращается инициализированный App struct с non-nil зависимостями.
func TestInitializeApp_FullPipeline(t *testing.T) {
	// Arrange — создаём реалистичный Config
	cfg := &config.Config{
		Command: "test-command",
		Actor:   "test-user",
		Env:     "test",
		LoggingConfig: &config.LoggingConfig{
			Level:  "debug",
			Format: "text",
		},
	}

	// Act — инициализируем App через Wire DI
	app, err := InitializeApp(cfg)

	// Assert — проверяем успешную инициализацию
	require.NoError(t, err, "InitializeApp должен успешно инициализировать App")
	require.NotNil(t, app, "App должен быть non-nil")

	// Проверяем Config
	assert.Same(t, cfg, app.Config, "App.Config должен быть переданным Config")
	assert.Equal(t, "test-command", app.Config.Command)

	// Проверяем Logger
	require.NotNil(t, app.Logger, "App.Logger должен быть non-nil")
	// Logger должен уметь логировать без паники
	assert.NotPanics(t, func() {
		app.Logger.Info("Тестовое сообщение", "key", "value")
		app.Logger.Debug("Debug сообщение")
		app.Logger.With("trace_id", app.TraceID).Info("С trace_id")
	}, "Logger должен работать корректно")

	// Проверяем OutputWriter
	require.NotNil(t, app.OutputWriter, "App.OutputWriter должен быть non-nil")

	// Проверяем TraceID
	assert.NotEmpty(t, app.TraceID, "App.TraceID должен быть непустым")
	assert.Len(t, app.TraceID, 32, "App.TraceID должен иметь длину 32 символа")
}

// TestInitializeApp_OutputWriterUsage проверяет использование OutputWriter из App.
// Демонстрирует что инициализированный App может использоваться для вывода результатов.
func TestInitializeApp_OutputWriterUsage(t *testing.T) {
	// Arrange
	cfg := &config.Config{}
	app, err := InitializeApp(cfg)
	require.NoError(t, err)
	require.NotNil(t, app)

	result := &output.Result{
		Status:  "success",
		Command: "test-command",
		Data: map[string]any{
			"trace_id": app.TraceID,
		},
		Metadata: &output.Metadata{
			TraceID: app.TraceID,
		},
	}

	// Act — записываем результат через OutputWriter
	var buf bytes.Buffer
	err = app.OutputWriter.Write(&buf, result)

	// Assert
	assert.NoError(t, err, "OutputWriter.Write должен успешно записать результат")
	assert.NotEmpty(t, buf.String(), "OutputWriter должен записать непустой вывод")
	assert.Contains(t, buf.String(), "test-command", "Вывод должен содержать имя команды")
}

// TestInitializeApp_LoggerWithTraceID проверяет совместное использование Logger и TraceID.
// Демонстрирует паттерн добавления trace_id к логгеру.
func TestInitializeApp_LoggerWithTraceID(t *testing.T) {
	// Arrange
	cfg := &config.Config{
		LoggingConfig: &config.LoggingConfig{
			Level:  "debug",
			Format: "json",
		},
	}

	// Act
	app, err := InitializeApp(cfg)
	require.NoError(t, err)

	// Создаём логгер с trace_id
	loggerWithTrace := app.Logger.With("trace_id", app.TraceID)

	// Assert — проверяем что логгер с trace_id работает
	assert.NotPanics(t, func() {
		loggerWithTrace.Info("Операция началась", "command", "test")
		loggerWithTrace.Debug("Детали операции")
		loggerWithTrace.Warn("Предупреждение")
	}, "Logger с trace_id должен работать корректно")
}

// TestInitializeApp_MultipleInitializations проверяет множественные инициализации.
// Каждая инициализация должна создавать независимый App с уникальным TraceID.
func TestInitializeApp_MultipleInitializations(t *testing.T) {
	// Arrange
	cfg := &config.Config{}
	const count = 5
	apps := make([]*App, count)
	traceIDs := make(map[string]bool)

	// Act — создаём несколько App
	for i := range count {
		app, err := InitializeApp(cfg)
		require.NoError(t, err)
		apps[i] = app
		traceIDs[app.TraceID] = true
	}

	// Assert — каждый App должен иметь уникальный TraceID
	assert.Len(t, traceIDs, count, "Все TraceID должны быть уникальными")

	// Все App должны быть независимыми
	for i := range count {
		assert.NotNil(t, apps[i], "Каждый App должен быть non-nil")
		assert.NotNil(t, apps[i].Logger, "Каждый App должен иметь Logger")
		assert.NotNil(t, apps[i].OutputWriter, "Каждый App должен иметь OutputWriter")
	}
}

// TestInitializeApp_DifferentConfigs проверяет инициализацию с разными конфигурациями.
func TestInitializeApp_DifferentConfigs(t *testing.T) {
	testCases := []struct {
		name      string
		config    *config.Config
		wantError bool
	}{
		{
			name:      "nil config",
			config:    nil,
			wantError: false, // nil Config gracefully handled — все провайдеры используют defaults
		},
		{
			name:      "empty config",
			config:    &config.Config{},
			wantError: false,
		},
		{
			name: "full config",
			config: &config.Config{
				Command: "full-test",
				Actor:   "test-actor",
				LoggingConfig: &config.LoggingConfig{
					Level:  "error",
					Format: "json",
				},
			},
			wantError: false,
		},
		{
			name: "config with only logging",
			config: &config.Config{
				LoggingConfig: &config.LoggingConfig{
					Level:  "warn",
					Format: "text",
				},
			},
			wantError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			app, err := InitializeApp(tc.config)

			// Assert
			if tc.wantError {
				assert.Error(t, err)
				assert.Nil(t, app)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, app)
				assert.NotNil(t, app.Logger)
				assert.NotNil(t, app.OutputWriter)
				assert.NotEmpty(t, app.TraceID)
			}
		})
	}
}
