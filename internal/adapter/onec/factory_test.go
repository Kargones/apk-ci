// Package onec_test содержит тесты для OneCFactory.
package onec_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kargones/apk-ci/internal/adapter/onec"
	"github.com/Kargones/apk-ci/internal/config"
)

// makeConfig создаёт конфигурацию с заданными параметрами для тестов.
func makeConfig(configExport, dbCreate, bin1cv8, binIbcmd string) *config.Config {
	appCfg := &config.AppConfig{
		Implementations: config.ImplementationsConfig{
			ConfigExport: configExport,
			DBCreate:     dbCreate,
		},
	}
	appCfg.Paths.Bin1cv8 = bin1cv8
	appCfg.Paths.BinIbcmd = binIbcmd

	return &config.Config{
		AppConfig: appCfg,
	}
}

// TestOneCFactory_NewConfigExporter_1cv8 проверяет создание 1cv8 ConfigExporter.
func TestOneCFactory_NewConfigExporter_1cv8(t *testing.T) {
	cfg := makeConfig("1cv8", "", "/usr/bin/1cv8", "")
	factory := onec.NewFactory(cfg)

	exporter, err := factory.NewConfigExporter()

	require.NoError(t, err)
	assert.NotNil(t, exporter)
}

// TestOneCFactory_NewConfigExporter_ibcmd проверяет создание ibcmd ConfigExporter.
func TestOneCFactory_NewConfigExporter_ibcmd(t *testing.T) {
	cfg := makeConfig("ibcmd", "", "", "/usr/bin/ibcmd")
	factory := onec.NewFactory(cfg)

	exporter, err := factory.NewConfigExporter()

	require.NoError(t, err)
	assert.NotNil(t, exporter)
}

// TestOneCFactory_NewConfigExporter_native проверяет что native возвращает ошибку (не реализовано).
func TestOneCFactory_NewConfigExporter_native(t *testing.T) {
	cfg := makeConfig("native", "", "", "")
	factory := onec.NewFactory(cfg)

	exporter, err := factory.NewConfigExporter()

	assert.Nil(t, exporter)
	require.Error(t, err)
	assert.True(t, errors.Is(err, onec.ErrInvalidImplementation))
	assert.Contains(t, err.Error(), "not implemented")
}

// TestOneCFactory_NewConfigExporter_invalid проверяет ошибку при невалидном значении.
func TestOneCFactory_NewConfigExporter_invalid(t *testing.T) {
	cfg := makeConfig("invalid_value", "", "", "")
	factory := onec.NewFactory(cfg)

	exporter, err := factory.NewConfigExporter()

	assert.Nil(t, exporter)
	require.Error(t, err)
	assert.True(t, errors.Is(err, onec.ErrInvalidImplementation))
	assert.Contains(t, err.Error(), "invalid_value")
}

// TestOneCFactory_NewConfigExporter_default проверяет default значение (1cv8).
func TestOneCFactory_NewConfigExporter_default(t *testing.T) {
	cfg := makeConfig("", "", "/usr/bin/1cv8", "") // empty = default
	factory := onec.NewFactory(cfg)

	exporter, err := factory.NewConfigExporter()

	require.NoError(t, err)
	assert.NotNil(t, exporter)
}

// TestOneCFactory_NewDatabaseCreator_1cv8 проверяет создание 1cv8 DatabaseCreator.
func TestOneCFactory_NewDatabaseCreator_1cv8(t *testing.T) {
	cfg := makeConfig("", "1cv8", "/usr/bin/1cv8", "")
	factory := onec.NewFactory(cfg)

	creator, err := factory.NewDatabaseCreator()

	require.NoError(t, err)
	assert.NotNil(t, creator)
}

// TestOneCFactory_NewDatabaseCreator_ibcmd проверяет создание ibcmd DatabaseCreator.
func TestOneCFactory_NewDatabaseCreator_ibcmd(t *testing.T) {
	cfg := makeConfig("", "ibcmd", "", "/usr/bin/ibcmd")
	factory := onec.NewFactory(cfg)

	creator, err := factory.NewDatabaseCreator()

	require.NoError(t, err)
	assert.NotNil(t, creator)
}

// TestOneCFactory_NewDatabaseCreator_invalid проверяет ошибку при невалидном значении.
func TestOneCFactory_NewDatabaseCreator_invalid(t *testing.T) {
	cfg := makeConfig("", "invalid_value", "", "")
	factory := onec.NewFactory(cfg)

	creator, err := factory.NewDatabaseCreator()

	assert.Nil(t, creator)
	require.Error(t, err)
	assert.True(t, errors.Is(err, onec.ErrInvalidImplementation))
	assert.Contains(t, err.Error(), "invalid_value")
}

// TestOneCFactory_NewDatabaseCreator_default проверяет default значение (1cv8).
func TestOneCFactory_NewDatabaseCreator_default(t *testing.T) {
	cfg := makeConfig("", "", "/usr/bin/1cv8", "") // empty = default
	factory := onec.NewFactory(cfg)

	creator, err := factory.NewDatabaseCreator()

	require.NoError(t, err)
	assert.NotNil(t, creator)
}

// TestErrInvalidImplementation_ErrorCode проверяет что ошибка содержит код ERR_INVALID_IMPL.
func TestErrInvalidImplementation_ErrorCode(t *testing.T) {
	assert.Contains(t, onec.ErrInvalidImplementation.Error(), "ERR_INVALID_IMPL")
}

// --- Integration Tests: Factory → Implementation → Method Call ---

// TestFactory_Integration_ConfigExporter_EmptyBinPath проверяет что созданная
// реализация ConfigExporter корректно валидирует пустой путь к бинарнику при вызове Export.
// Это integration тест: Factory создаёт реализацию → реализация вызывает метод → ошибка.
func TestFactory_Integration_ConfigExporter_EmptyBinPath(t *testing.T) {
	tests := []struct {
		name         string
		configExport string
		expectedErr  string
	}{
		{
			name:         "1cv8 с пустым путём",
			configExport: "1cv8",
			expectedErr:  "путь к 1cv8 не указан",
		},
		{
			name:         "ibcmd с пустым путём",
			configExport: "ibcmd",
			expectedErr:  "путь к ibcmd не указан",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange: создаём конфиг БЕЗ путей к бинарникам
			cfg := makeConfig(tt.configExport, "", "", "")
			factory := onec.NewFactory(cfg)

			// Act: Factory создаёт реализацию
			exporter, err := factory.NewConfigExporter()
			require.NoError(t, err, "Factory должен создать реализацию")
			require.NotNil(t, exporter)

			// Act: вызываем Export — должна быть ошибка валидации
			result, err := exporter.Export(t.Context(), onec.ExportOptions{
				ConnectString: "/F /tmp/testdb",
				OutputPath:    "/tmp/export",
			})

			// Assert
			assert.Nil(t, result)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

// TestFactory_Integration_DatabaseCreator_EmptyBinPath проверяет что созданная
// реализация DatabaseCreator корректно валидирует пустой путь к бинарнику при вызове CreateDB.
func TestFactory_Integration_DatabaseCreator_EmptyBinPath(t *testing.T) {
	tests := []struct {
		name        string
		dbCreate    string
		expectedErr string
	}{
		{
			name:        "1cv8 с пустым путём",
			dbCreate:    "1cv8",
			expectedErr: "путь к 1cv8 не указан",
		},
		{
			name:        "ibcmd с пустым путём",
			dbCreate:    "ibcmd",
			expectedErr: "путь к ibcmd не указан",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange: создаём конфиг БЕЗ путей к бинарникам
			cfg := makeConfig("", tt.dbCreate, "", "")
			factory := onec.NewFactory(cfg)

			// Act: Factory создаёт реализацию
			creator, err := factory.NewDatabaseCreator()
			require.NoError(t, err, "Factory должен создать реализацию")
			require.NotNil(t, creator)

			// Act: вызываем CreateDB — должна быть ошибка валидации
			result, err := creator.CreateDB(t.Context(), onec.CreateDBOptions{
				DbPath: "/tmp/newdb",
			})

			// Assert
			assert.Nil(t, result)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}
