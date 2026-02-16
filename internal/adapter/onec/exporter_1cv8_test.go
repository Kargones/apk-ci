// Package onec содержит тесты для Exporter1cv8.
package onec

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestExporter1cv8_CompileTimeInterface проверяет что Exporter1cv8 реализует ConfigExporter.
func TestExporter1cv8_CompileTimeInterface(_ *testing.T) {
	// Compile-time проверка — если этот код компилируется, интерфейс реализован
	var _ ConfigExporter = (*Exporter1cv8)(nil)
}

// TestNewExporter1cv8_CreatesExporter проверяет создание Exporter1cv8.
func TestNewExporter1cv8_CreatesExporter(t *testing.T) {
	exporter := NewExporter1cv8("/usr/bin/1cv8", "/work", "/tmp")

	assert.NotNil(t, exporter)
	assert.Equal(t, "/usr/bin/1cv8", exporter.bin1cv8)
	assert.Equal(t, "/work", exporter.workDir)
	assert.Equal(t, "/tmp", exporter.tmpDir)
}

// TestNewExporter1cv8_EmptyPaths проверяет создание с пустыми путями.
func TestNewExporter1cv8_EmptyPaths(t *testing.T) {
	exporter := NewExporter1cv8("", "", "")

	assert.NotNil(t, exporter)
	assert.Empty(t, exporter.bin1cv8)
	assert.Empty(t, exporter.workDir)
	assert.Empty(t, exporter.tmpDir)
}

// TestExporter1cv8_Export_EmptyBinPath проверяет ошибку при пустом пути к бинарнику.
func TestExporter1cv8_Export_EmptyBinPath(t *testing.T) {
	exporter := NewExporter1cv8("", "/work", "/tmp")

	result, err := exporter.Export(context.Background(), ExportOptions{
		ConnectString: "/F /tmp/testdb",
		OutputPath:    "/tmp/export",
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "путь к 1cv8 не указан")
}
