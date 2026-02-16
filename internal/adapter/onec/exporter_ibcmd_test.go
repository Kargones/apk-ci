// Package onec содержит тесты для ExporterIbcmd.
package onec

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestExporterIbcmd_CompileTimeInterface проверяет что ExporterIbcmd реализует ConfigExporter.
func TestExporterIbcmd_CompileTimeInterface(_ *testing.T) {
	// Compile-time проверка — если этот код компилируется, интерфейс реализован
	var _ ConfigExporter = (*ExporterIbcmd)(nil)
}

// TestNewExporterIbcmd_CreatesExporter проверяет создание ExporterIbcmd.
func TestNewExporterIbcmd_CreatesExporter(t *testing.T) {
	exporter := NewExporterIbcmd("/usr/bin/ibcmd")

	assert.NotNil(t, exporter)
	assert.Equal(t, "/usr/bin/ibcmd", exporter.binIbcmd)
}

// TestNewExporterIbcmd_EmptyPath проверяет создание с пустым путём.
func TestNewExporterIbcmd_EmptyPath(t *testing.T) {
	exporter := NewExporterIbcmd("")

	assert.NotNil(t, exporter)
	assert.Empty(t, exporter.binIbcmd)
}

// TestExporterIbcmd_Export_EmptyBinPath проверяет ошибку при пустом пути к бинарнику.
func TestExporterIbcmd_Export_EmptyBinPath(t *testing.T) {
	exporter := NewExporterIbcmd("")

	result, err := exporter.Export(context.Background(), ExportOptions{
		ConnectString: "/F /tmp/testdb",
		OutputPath:    "/tmp/export",
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "путь к ibcmd не указан")
}

// TestExtractDbPath_Variations проверяет извлечение пути БД из connect string.
// ВАЖНО: extractDbPath предназначен только для файловых баз.
func TestExtractDbPath_Variations(t *testing.T) {
	tests := []struct {
		name          string
		connectString string
		expected      string
	}{
		{
			name:          "с /F prefix",
			connectString: "/F /tmp/database",
			expected:      "/tmp/database",
		},
		{
			name:          "без prefix",
			connectString: "/tmp/database",
			expected:      "/tmp/database",
		},
		{
			name:          "с пробелами по краям",
			connectString: "  /F /tmp/database  ",
			expected:      "/tmp/database",
		},
		{
			name:          "только путь с пробелами",
			connectString: "  /tmp/database  ",
			expected:      "/tmp/database",
		},
		{
			name:          "Windows путь",
			connectString: "/F C:\\1c\\database",
			expected:      "C:\\1c\\database",
		},
		{
			name:          "пустая строка",
			connectString: "",
			expected:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDbPath(tt.connectString)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsServerConnectString проверяет определение серверной строки подключения.
func TestIsServerConnectString(t *testing.T) {
	tests := []struct {
		name          string
		connectString string
		expected      bool
	}{
		{
			name:          "файловая /F",
			connectString: "/F /tmp/db",
			expected:      false,
		},
		{
			name:          "файловая без prefix",
			connectString: "/tmp/db",
			expected:      false,
		},
		{
			name:          "серверная /S",
			connectString: `/S server\dbname`,
			expected:      true,
		},
		{
			name:          "серверная /S с пробелами",
			connectString: `  /S server\dbname  `,
			expected:      true,
		},
		{
			name:          "серверная /S с credentials",
			connectString: `/S server\dbname /N user /P pass`,
			expected:      true,
		},
		{
			name:          "серверная Srvr=",
			connectString: `Srvr="server";Ref="dbname"`,
			expected:      true,
		},
		{
			name:          "пустая строка",
			connectString: "",
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isServerConnectString(tt.connectString)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExporterIbcmd_Export_ServerBasedNotSupported проверяет что серверные базы
// возвращают понятную ошибку (H-2 fix).
func TestExporterIbcmd_Export_ServerBasedNotSupported(t *testing.T) {
	exporter := NewExporterIbcmd("/usr/bin/ibcmd")

	tests := []struct {
		name          string
		connectString string
	}{
		{
			name:          "серверная база /S",
			connectString: `/S server\dbname`,
		},
		{
			name:          "серверная база /S с credentials",
			connectString: `/S server\dbname /N user /P pass`,
		},
		{
			name:          "серверная база Srvr=",
			connectString: `Srvr="server";Ref="dbname"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := exporter.Export(context.Background(), ExportOptions{
				ConnectString: tt.connectString,
				OutputPath:    "/tmp/export",
			})

			assert.Nil(t, result)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "серверные базы")
			assert.Contains(t, err.Error(), "1cv8")
		})
	}
}
