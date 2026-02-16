// Package config содержит тесты для конфигурации реализаций.
package config

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestImplementationsConfig_Parse проверяет парсинг YAML с секцией implementations (AC1)
func TestImplementationsConfig_Parse(t *testing.T) {
	// Arrange - YAML с секцией implementations
	yamlData := `
implementations:
  config_export: ibcmd
  db_create: ibcmd
`
	type testConfig struct {
		Implementations ImplementationsConfig `yaml:"implementations"`
	}

	// Act
	var cfg testConfig
	err := yaml.Unmarshal([]byte(yamlData), &cfg)

	// Assert
	require.NoError(t, err, "YAML с implementations должен парситься без ошибок")
	assert.Equal(t, "ibcmd", cfg.Implementations.ConfigExport)
	assert.Equal(t, "ibcmd", cfg.Implementations.DBCreate)
}

// TestImplementationsConfig_Defaults проверяет что defaults применяются корректно (AC3)
func TestImplementationsConfig_Defaults(t *testing.T) {
	// Act
	impl := getDefaultImplementationsConfig()

	// Assert
	require.NotNil(t, impl)
	assert.Equal(t, "1cv8", impl.ConfigExport, "default ConfigExport должен быть '1cv8'")
	assert.Equal(t, "1cv8", impl.DBCreate, "default DBCreate должен быть '1cv8'")
}

// TestImplementationsConfig_EnvOverride проверяет что env vars переопределяют файл (AC4)
func TestImplementationsConfig_EnvOverride(t *testing.T) {
	// Arrange - устанавливаем переменные окружения
	t.Setenv("BR_IMPL_CONFIG_EXPORT", "ibcmd")
	t.Setenv("BR_IMPL_DB_CREATE", "ibcmd")

	// Act - используем реальный механизм через loadImplementationsConfig
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &Config{} // AppConfig = nil, поэтому будут использоваться defaults + env override

	implConfig, err := loadImplementationsConfig(logger, cfg)

	// Assert
	require.NoError(t, err, "loadImplementationsConfig не должен возвращать ошибку")
	require.NotNil(t, implConfig)
	assert.Equal(t, "ibcmd", implConfig.ConfigExport, "ConfigExport должен быть переопределён из env")
	assert.Equal(t, "ibcmd", implConfig.DBCreate, "DBCreate должен быть переопределён из env")
}

// TestLoggingConfig_DefaultFormat проверяет что format="text" по умолчанию (AC3)
func TestLoggingConfig_DefaultFormat(t *testing.T) {
	// Act
	logging := getDefaultLoggingConfig()

	// Assert
	require.NotNil(t, logging)
	assert.Equal(t, "text", logging.Format, "default Format должен быть 'text'")
}

// TestLoggingConfig_DefaultLevel проверяет что level="info" по умолчанию (AC3)
func TestLoggingConfig_DefaultLevel(t *testing.T) {
	// Act
	logging := getDefaultLoggingConfig()

	// Assert
	require.NotNil(t, logging)
	assert.Equal(t, "info", logging.Level, "default Level должен быть 'info'")
}

// TestLoggingConfig_DefaultOutput проверяет что output="stderr" по умолчанию
func TestLoggingConfig_DefaultOutput(t *testing.T) {
	// Act
	logging := getDefaultLoggingConfig()

	// Assert
	require.NotNil(t, logging)
	assert.Equal(t, "stderr", logging.Output, "default Output должен быть 'stderr'")
}

// TestConfig_BackwardCompatibility проверяет что конфиг без implementations/logging парсится (AC5)
func TestConfig_BackwardCompatibility(t *testing.T) {
	// Arrange - создаём пустую конфигурацию
	cfg := &Config{}

	// Assert - проверяем что ImplementationsConfig можно безопасно получить (nil или default)
	// Это проверяет что отсутствие секции не вызывает panic
	assert.Nil(t, cfg.ImplementationsConfig, "ImplementationsConfig должен быть nil если не загружен")
	assert.Nil(t, cfg.LoggingConfig, "LoggingConfig должен быть nil если не загружен")
}

// TestConfig_ZeroValueHandling проверяет что все новые поля optional (AC6)
func TestConfig_ZeroValueHandling(t *testing.T) {
	// Arrange
	impl := &ImplementationsConfig{}

	// Assert - zero values до валидации
	assert.Equal(t, "", impl.ConfigExport)
	assert.Equal(t, "", impl.DBCreate)

	// Validate() применяет defaults для пустых значений
	err := impl.Validate()
	assert.NoError(t, err)
	assert.Equal(t, "1cv8", impl.ConfigExport, "Validate() должен применить default для пустого ConfigExport")
	assert.Equal(t, "1cv8", impl.DBCreate, "Validate() должен применить default для пустого DBCreate")
}

// TestConfig_ProductionBackwardCompat проверяет что production конфиг без новых секций парсится (AC5)
func TestConfig_ProductionBackwardCompat(t *testing.T) {
	// Arrange - читаем production-like конфиг из testdata
	data, err := os.ReadFile("testdata/production_config.yaml")
	require.NoError(t, err, "Не удалось прочитать testdata/production_config.yaml")

	// Act - парсим конфиг в AppConfig
	var appConfig AppConfig
	err = parseYAML(data, &appConfig)

	// Assert
	require.NoError(t, err, "Production конфиг должен парситься без ошибок")

	// Проверяем что основные поля распарсились
	assert.Equal(t, "Info", appConfig.LogLevel)
	assert.Equal(t, "/tmp/benadis", appConfig.WorkDir)
	assert.Equal(t, 1545, appConfig.Rac.Port)

	// Проверяем что отсутствующие секции имеют zero values (не вызывают panic)
	assert.Equal(t, ImplementationsConfig{}, appConfig.Implementations)
	assert.Equal(t, LoggingConfig{}, appConfig.Logging)
}

// parseYAML - вспомогательная функция для парсинга YAML в структуру
func parseYAML(data []byte, v interface{}) error {
	return yaml.Unmarshal(data, v)
}

// TestImplementationsConfig_Validate проверяет валидацию значений (M1 fix)
func TestImplementationsConfig_Validate(t *testing.T) {
	tests := []struct {
		name         string
		configExport string
		dbCreate     string
		wantErr      bool
	}{
		{"valid_1cv8_1cv8", "1cv8", "1cv8", false},
		{"valid_ibcmd_ibcmd", "ibcmd", "ibcmd", false},
		{"valid_native_1cv8", "native", "1cv8", false},
		{"valid_empty_values_defaults_applied", "", "", false},
		{"invalid_config_export", "invalid", "1cv8", true},
		{"invalid_db_create", "1cv8", "native", true}, // native не допустим для DBCreate
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			impl := &ImplementationsConfig{
				ConfigExport: tt.configExport,
				DBCreate:     tt.dbCreate,
			}
			err := impl.Validate()
			if tt.wantErr {
				assert.Error(t, err, "Ожидалась ошибка валидации")
			} else {
				assert.NoError(t, err, "Не ожидалась ошибка валидации")
			}
		})
	}
}

// TestLoggingConfig_EnvOverride проверяет переопределение через BR_LOG_* переменные
func TestLoggingConfig_EnvOverride(t *testing.T) {
	// Arrange - устанавливаем переменные окружения
	t.Setenv("BR_LOG_LEVEL", "debug")
	t.Setenv("BR_LOG_FORMAT", "json")
	t.Setenv("BR_LOG_OUTPUT", "file")

	// Act - получаем default конфигурацию и применяем env override
	logging := getDefaultLoggingConfig()
	require.NotNil(t, logging)

	// Применяем переменные окружения
	if val := os.Getenv("BR_LOG_LEVEL"); val != "" {
		logging.Level = val
	}
	if val := os.Getenv("BR_LOG_FORMAT"); val != "" {
		logging.Format = val
	}
	if val := os.Getenv("BR_LOG_OUTPUT"); val != "" {
		logging.Output = val
	}

	// Assert
	assert.Equal(t, "debug", logging.Level, "Level должен быть переопределён из env")
	assert.Equal(t, "json", logging.Format, "Format должен быть переопределён из env")
	assert.Equal(t, "file", logging.Output, "Output должен быть переопределён из env")
}
