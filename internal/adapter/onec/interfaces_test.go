// Package onec_test содержит тесты для интерфейсов операций 1C.
package onec_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Kargones/apk-ci/internal/adapter/onec"
)

// TestExportOptions_Fields проверяет что ExportOptions содержит все необходимые поля.
func TestExportOptions_Fields(t *testing.T) {
	opts := onec.ExportOptions{
		ConnectString: "/S server\\base /N user /P pass",
		OutputPath:    "/tmp/export",
		Extension:     "TestExtension",
		Timeout:       5 * time.Minute,
	}

	assert.Equal(t, "/S server\\base /N user /P pass", opts.ConnectString)
	assert.Equal(t, "/tmp/export", opts.OutputPath)
	assert.Equal(t, "TestExtension", opts.Extension)
	assert.Equal(t, 5*time.Minute, opts.Timeout)
}

// TestExportResult_Fields проверяет что ExportResult содержит все необходимые поля.
func TestExportResult_Fields(t *testing.T) {
	result := onec.ExportResult{
		Success:    true,
		OutputPath: "/tmp/export",
		Messages:   []string{"Выгрузка завершена"},
		DurationMs: 1234,
	}

	assert.True(t, result.Success)
	assert.Equal(t, "/tmp/export", result.OutputPath)
	assert.Equal(t, []string{"Выгрузка завершена"}, result.Messages)
	assert.Equal(t, int64(1234), result.DurationMs)
}

// TestCreateDBOptions_Fields проверяет что CreateDBOptions содержит все необходимые поля.
func TestCreateDBOptions_Fields(t *testing.T) {
	opts := onec.CreateDBOptions{
		DbPath:      "/tmp/testdb",
		ServerBased: true,
		Server:      "server1c",
		DbName:      "testbase",
		Timeout:     10 * time.Minute,
	}

	assert.Equal(t, "/tmp/testdb", opts.DbPath)
	assert.True(t, opts.ServerBased)
	assert.Equal(t, "server1c", opts.Server)
	assert.Equal(t, "testbase", opts.DbName)
	assert.Equal(t, 10*time.Minute, opts.Timeout)
}

// TestCreateDBResult_Fields проверяет что CreateDBResult содержит все необходимые поля.
func TestCreateDBResult_Fields(t *testing.T) {
	createdAt := time.Now()
	result := onec.CreateDBResult{
		Success:       true,
		ConnectString: "/F /tmp/testdb",
		DbPath:        "/tmp/testdb",
		CreatedAt:     createdAt,
		DurationMs:    5678,
	}

	assert.True(t, result.Success)
	assert.Equal(t, "/F /tmp/testdb", result.ConnectString)
	assert.Equal(t, "/tmp/testdb", result.DbPath)
	assert.Equal(t, createdAt, result.CreatedAt)
	assert.Equal(t, int64(5678), result.DurationMs)
}

// mockConfigExporter — mock реализация для тестирования интерфейса ConfigExporter.
type mockConfigExporter struct {
	exportCalled bool
	exportOpts   onec.ExportOptions
	exportResult *onec.ExportResult
	exportErr    error
}

func (m *mockConfigExporter) Export(_ context.Context, opts onec.ExportOptions) (*onec.ExportResult, error) {
	m.exportCalled = true
	m.exportOpts = opts
	return m.exportResult, m.exportErr
}

// Compile-time проверка что mock реализует интерфейс.
var _ onec.ConfigExporter = (*mockConfigExporter)(nil)

// TestConfigExporter_Interface проверяет что интерфейс ConfigExporter работает корректно.
func TestConfigExporter_Interface(t *testing.T) {
	mock := &mockConfigExporter{
		exportResult: &onec.ExportResult{
			Success:    true,
			OutputPath: "/output",
		},
	}

	opts := onec.ExportOptions{
		ConnectString: "/F /tmp/db",
		OutputPath:    "/output",
	}

	result, err := mock.Export(context.Background(), opts)

	assert.NoError(t, err)
	assert.True(t, mock.exportCalled)
	assert.Equal(t, opts, mock.exportOpts)
	assert.True(t, result.Success)
}

// mockDatabaseCreator — mock реализация для тестирования интерфейса DatabaseCreator.
type mockDatabaseCreator struct {
	createCalled bool
	createOpts   onec.CreateDBOptions
	createResult *onec.CreateDBResult
	createErr    error
}

func (m *mockDatabaseCreator) CreateDB(_ context.Context, opts onec.CreateDBOptions) (*onec.CreateDBResult, error) {
	m.createCalled = true
	m.createOpts = opts
	return m.createResult, m.createErr
}

// Compile-time проверка что mock реализует интерфейс.
var _ onec.DatabaseCreator = (*mockDatabaseCreator)(nil)

// TestDatabaseCreator_Interface проверяет что интерфейс DatabaseCreator работает корректно.
func TestDatabaseCreator_Interface(t *testing.T) {
	now := time.Now()
	mock := &mockDatabaseCreator{
		createResult: &onec.CreateDBResult{
			ConnectString: "/F /tmp/newdb",
			DbPath:        "/tmp/newdb",
			CreatedAt:     now,
		},
	}

	opts := onec.CreateDBOptions{
		DbPath:  "/tmp/newdb",
		Timeout: 5 * time.Minute,
	}

	result, err := mock.CreateDB(context.Background(), opts)

	assert.NoError(t, err)
	assert.True(t, mock.createCalled)
	assert.Equal(t, opts, mock.createOpts)
	assert.Equal(t, "/F /tmp/newdb", result.ConnectString)
}
