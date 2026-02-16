package filer

import (
	"os"
	"testing"
)

// TestDiskFileSystem_ChownAdditional тестирует изменение владельца файла
func TestDiskFileSystem_ChownAdditional(t *testing.T) {
	tempDir := t.TempDir()
	config := Config{
		Type:     DiskFS,
		BasePath: tempDir,
	}
	
	fs, err := NewDiskFileSystem(config)
	if err != nil {
		t.Fatalf("Failed to create DiskFileSystem: %v", err)
	}
	
	// Создаем файл
	file, err := fs.Create("test.txt")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	file.Close()
	
	// Тест Chown с текущими uid/gid (может не работать в некоторых средах)
	err = fs.Chown("test.txt", os.Getuid(), os.Getgid())
	if err != nil {
		// В некоторых средах Chown может не работать, это нормально
		t.Logf("Chown failed (may be expected in some environments): %v", err)
	}
	
	// Тест Chown с недопустимым путем
	err = fs.Chown("../../../invalid", 0, 0)
	if err == nil {
		t.Error("Chown should fail with invalid path")
	}
	
	// Тест Chown с несуществующим файлом
	err = fs.Chown("nonexistent.txt", 0, 0)
	if err == nil {
		t.Error("Chown should fail with nonexistent file")
	}
}

// TestDiskFileSystem_GetwdAdditional тестирует получение текущей директории
func TestDiskFileSystem_GetwdAdditional(t *testing.T) {
	tempDir := t.TempDir()
	config := Config{
		Type:     DiskFS,
		BasePath: tempDir,
	}
	
	fs, err := NewDiskFileSystem(config)
	if err != nil {
		t.Fatalf("Failed to create DiskFileSystem: %v", err)
	}
	
	// Тест Getwd
	cwd, err := fs.Getwd()
	if err != nil {
		t.Errorf("Getwd failed: %v", err)
	}
	
	// В текущей реализации всегда возвращает "/"
	if cwd != "/" {
		t.Errorf("Expected cwd '/', got %q", cwd)
	}
}

// TestDiskFileSystem_ChdirAdditional тестирует изменение текущей директории
func TestDiskFileSystem_ChdirAdditional(t *testing.T) {
	tempDir := t.TempDir()
	config := Config{
		Type:     DiskFS,
		BasePath: tempDir,
	}
	
	fs, err := NewDiskFileSystem(config)
	if err != nil {
		t.Fatalf("Failed to create DiskFileSystem: %v", err)
	}
	
	// Создаем директорию
	err = fs.MkdirAll("testdir", 0755)
	if err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	
	// Тест Chdir (в текущей реализации это no-op)
	err = fs.Chdir("testdir")
	if err != nil {
		t.Errorf("Chdir failed: %v", err)
	}
	
	// Тест Chdir с недопустимым путем (в текущей реализации это no-op)
	err = fs.Chdir("../../../invalid")
	if err != nil {
		t.Errorf("Chdir failed with invalid path: %v", err)
	}
	
	// Тест Chdir с несуществующей директорией (в текущей реализации это no-op)
	err = fs.Chdir("nonexistent")
	if err != nil {
		t.Errorf("Chdir failed with nonexistent directory: %v", err)
	}
}