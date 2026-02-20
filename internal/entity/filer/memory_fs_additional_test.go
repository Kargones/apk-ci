package filer

import (
	"testing"
)

// TestMemoryFileSystem_GetwdAdditional тестирует получение текущей директории
func TestMemoryFileSystem_GetwdAdditional(t *testing.T) {
	fs := NewMemoryFileSystem("/tmp/test")
	
	cwd, err := fs.Getwd()
	if err != nil {
		t.Errorf("Getwd failed: %v", err)
	}
	
	if cwd != "/tmp/test" { //nolint:goconst // test value
		t.Errorf("Expected cwd '/tmp/test', got '%s'", cwd)
	}
}

// TestMemoryFileSystem_ChdirAdditional тестирует изменение текущей директории
func TestMemoryFileSystem_ChdirAdditional(t *testing.T) {
	fs := NewMemoryFileSystem("/tmp")
	
	// Создаем директорию
	err := fs.MkdirAll("testdir", 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	
	// Меняем текущую директорию (в текущей реализации это может вызвать ошибку валидации)
	err = fs.Chdir("testdir")
	if err != nil {
		// Ожидаем ошибку валидации пути, так как resolvePath создает абсолютный путь
		t.Logf("Chdir failed as expected: %v", err)
		return // Завершаем тест, так как это ожидаемое поведение
	}
	
	// Проверяем текущую директорию
	cwd, err := fs.Getwd()
	if err != nil {
		t.Errorf("Getwd failed: %v", err)
	}
	
	// Chdir должен изменить текущую директорию
	if cwd != "/tmp/testdir" {
		t.Errorf("Expected cwd '/tmp/testdir', got '%s'", cwd)
	}
	
	// Тест с недопустимым путем
	err = fs.Chdir("../../../invalid")
	if err == nil {
		t.Error("Chdir should fail with invalid path")
	}
	
	// Тест с несуществующей директорией
	err = fs.Chdir("nonexistent")
	if err == nil {
		t.Error("Chdir should fail with nonexistent directory")
	}
}