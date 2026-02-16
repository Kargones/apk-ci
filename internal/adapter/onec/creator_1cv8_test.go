// Package onec содержит тесты для Creator1cv8.
package onec

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCreator1cv8_CompileTimeInterface проверяет что Creator1cv8 реализует DatabaseCreator.
func TestCreator1cv8_CompileTimeInterface(_ *testing.T) {
	// Compile-time проверка — если этот код компилируется, интерфейс реализован
	var _ DatabaseCreator = (*Creator1cv8)(nil)
}

// TestNewCreator1cv8_CreatesCreator проверяет создание Creator1cv8.
func TestNewCreator1cv8_CreatesCreator(t *testing.T) {
	creator := NewCreator1cv8("/usr/bin/1cv8", "/work", "/tmp")

	assert.NotNil(t, creator)
	assert.Equal(t, "/usr/bin/1cv8", creator.bin1cv8)
	assert.Equal(t, "/work", creator.workDir)
	assert.Equal(t, "/tmp", creator.tmpDir)
}

// TestNewCreator1cv8_EmptyPaths проверяет создание с пустыми путями.
func TestNewCreator1cv8_EmptyPaths(t *testing.T) {
	creator := NewCreator1cv8("", "", "")

	assert.NotNil(t, creator)
	assert.Empty(t, creator.bin1cv8)
	assert.Empty(t, creator.workDir)
	assert.Empty(t, creator.tmpDir)
}

// TestCreator1cv8_CreateDB_EmptyBinPath проверяет ошибку при пустом пути к бинарнику.
func TestCreator1cv8_CreateDB_EmptyBinPath(t *testing.T) {
	creator := NewCreator1cv8("", "/work", "/tmp")

	result, err := creator.CreateDB(context.Background(), CreateDBOptions{
		DbPath: "/tmp/newdb",
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "путь к 1cv8 не указан")
}

// TestCreator1cv8_BuildConnectString проверяет формирование строки подключения.
func TestCreator1cv8_BuildConnectString(t *testing.T) {
	creator := NewCreator1cv8("/usr/bin/1cv8", "/work", "/tmp")

	tests := []struct {
		name     string
		opts     CreateDBOptions
		expected string
	}{
		{
			name: "файловая база",
			opts: CreateDBOptions{
				DbPath:      "/tmp/mydb",
				ServerBased: false,
			},
			expected: `File="/tmp/mydb"`,
		},
		{
			name: "серверная база",
			opts: CreateDBOptions{
				Server:      "server1c",
				DbName:      "testbase",
				ServerBased: true,
			},
			expected: `Srvr="server1c";Ref="testbase"`,
		},
		{
			name: "серверная база с точками в имени",
			opts: CreateDBOptions{
				Server:      "srv.apkholding.ru",
				DbName:      "prod.main",
				ServerBased: true,
			},
			expected: `Srvr="srv.apkholding.ru";Ref="prod.main"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := creator.buildConnectString(tt.opts)
			assert.Equal(t, tt.expected, result)
		})
	}
}
