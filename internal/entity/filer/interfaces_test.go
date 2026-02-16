package filer

import (
	"testing"
)

// TestFileSystemInterface проверяет, что все реализации соответствуют интерфейсу FileSystem
func TestFileSystemInterface(t *testing.T) {
	tests := []struct {
		name string
		fs   FileSystem
	}{
		{
			name: "DiskFileSystem",
			fs:   mustCreateDiskFS(t),
		},
		{
			name: "MemoryFileSystem",
			fs:   NewMemoryFileSystem("/tmp"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Проверяем, что объект реализует интерфейс FileSystem
			var _ = tt.fs
			
			// Базовые операции
			if err := tt.fs.MkdirAll("test", 0755); err != nil {
				t.Errorf("MkdirAll failed: %v", err)
			}
			
			file, err := tt.fs.Create("test/file.txt")
			if err != nil {
				t.Errorf("Create failed: %v", err)
			}
			if file != nil {
				file.Close()
			}
			
			// Очистка
			tt.fs.RemoveAll("test")
		})
	}
}

// mustCreateDiskFS создает DiskFileSystem для тестов
func mustCreateDiskFS(t *testing.T) FileSystem {
	config := Config{
		Type:     DiskFS,
		BasePath: t.TempDir(),
	}
	fs, err := NewDiskFileSystem(config)
	if err != nil {
		t.Fatalf("Failed to create DiskFileSystem: %v", err)
	}
	return fs
}