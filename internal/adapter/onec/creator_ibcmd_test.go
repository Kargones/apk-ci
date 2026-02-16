// Package onec содержит тесты для CreatorIbcmd.
package onec

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCreatorIbcmd_CompileTimeInterface проверяет что CreatorIbcmd реализует DatabaseCreator.
func TestCreatorIbcmd_CompileTimeInterface(_ *testing.T) {
	// Compile-time проверка — если этот код компилируется, интерфейс реализован
	var _ DatabaseCreator = (*CreatorIbcmd)(nil)
}

// TestNewCreatorIbcmd_CreatesCreator проверяет создание CreatorIbcmd.
func TestNewCreatorIbcmd_CreatesCreator(t *testing.T) {
	creator := NewCreatorIbcmd("/usr/bin/ibcmd")

	assert.NotNil(t, creator)
	assert.Equal(t, "/usr/bin/ibcmd", creator.binIbcmd)
}

// TestNewCreatorIbcmd_EmptyPath проверяет создание с пустым путём.
func TestNewCreatorIbcmd_EmptyPath(t *testing.T) {
	creator := NewCreatorIbcmd("")

	assert.NotNil(t, creator)
	assert.Empty(t, creator.binIbcmd)
}

// TestCreatorIbcmd_CreateDB_EmptyBinPath проверяет ошибку при пустом пути к бинарнику.
func TestCreatorIbcmd_CreateDB_EmptyBinPath(t *testing.T) {
	creator := NewCreatorIbcmd("")

	result, err := creator.CreateDB(context.Background(), CreateDBOptions{
		DbPath: "/tmp/newdb",
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "путь к ibcmd не указан")
}

// TestCreatorIbcmd_BuildConnectString проверяет формирование строки подключения.
func TestCreatorIbcmd_BuildConnectString(t *testing.T) {
	creator := NewCreatorIbcmd("/usr/bin/ibcmd")

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
			expected: "/F /tmp/mydb",
		},
		{
			name: "серверная база",
			opts: CreateDBOptions{
				Server:      "server1c",
				DbName:      "testbase",
				ServerBased: true,
			},
			expected: `/S server1c\testbase`,
		},
		{
			name: "серверная база с пробелами",
			opts: CreateDBOptions{
				Server:      "server1c",
				DbName:      "test base",
				ServerBased: true,
			},
			expected: `/S server1c\test base`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := creator.buildConnectString(tt.opts)
			assert.Equal(t, tt.expected, result)
		})
	}
}
