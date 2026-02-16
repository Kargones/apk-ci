package filer

import (
	"os"
	"testing"
)

// TestDiskFileSystem_Mkdir тестирует создание директории
func TestDiskFileSystem_Mkdir(t *testing.T) {
	// Создаем временную директорию для тестов
	tempDir := t.TempDir()
	
	// Создаем DiskFileSystem
	config := Config{
		Type:     DiskFS,
		BasePath: tempDir,
	}
	fs, err := NewDiskFileSystem(config)
	if err != nil {
		t.Fatalf("Failed to create DiskFileSystem: %v", err)
	}
	
	// Тест успешного создания директории
	err = fs.Mkdir("testdir", 0755)
	if err != nil {
		t.Errorf("Mkdir failed: %v", err)
	}
	
	// Проверяем, что директория создана
	info, err := fs.Stat("testdir")
	if err != nil {
		t.Errorf("Stat failed: %v", err)
	}
	if !info.IsDir() {
		t.Error("Expected directory")
	}
	
	// Тест создания директории с недопустимым путем
	err = fs.Mkdir("../../../invalid", 0755)
	if err == nil {
		t.Error("Mkdir should fail with invalid path")
	}
}

// TestMemoryFile_AdditionalMethods тестирует дополнительные методы MemoryFile
func TestMemoryFile_AdditionalMethods(t *testing.T) {
	// Создаем MemoryFileSystem
	fs := NewMemoryFileSystem("test")
	
	// Создаем файл
	file, err := fs.Create("test.txt")
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()
	
	// Приводим к типу MemoryFile для доступа к дополнительным методам
	memFile, ok := file.(*MemoryFile)
	if !ok {
		t.Fatal("Expected MemoryFile type")
	}
	
	// Тест Sync
	err = memFile.Sync()
	if err != nil {
		t.Errorf("Sync failed: %v", err)
	}
	
	// Тест IsEmpty
	if !memFile.IsEmpty() {
		t.Error("New file should be empty")
	}
	
	// Тест IsClosed
	if memFile.IsClosed() {
		t.Error("New file should not be closed")
	}
	
	// Тест Size
	if memFile.Size() != 0 {
		t.Errorf("Expected size 0, got %d", memFile.Size())
	}
	
	// Закрываем файл
	memFile.Close()
	
	// Проверяем, что файл закрыт
	if !memFile.IsClosed() {
		t.Error("File should be closed after Close()")
	}
	
	// Тест Size на закрытом файле
	if memFile.Size() != 0 {
		t.Errorf("Expected size 0, got %d", memFile.Size())
	}
}

// TestMemoryFile_GetSetFlagAdditional тестирует GetFlag и SetFlag
func TestMemoryFile_GetSetFlagAdditional(t *testing.T) {
	// Создаем MemoryFileSystem
	fs := NewMemoryFileSystem("/tmp")
	
	// Создаем файл
	file, err := fs.Create("test.txt")
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()
	
	// Приводим к типу MemoryFile для доступа к дополнительным методам
	memFile, ok := file.(*MemoryFile)
	if !ok {
		t.Fatal("Expected MemoryFile type")
	}
	
	// Проверяем начальное значение флага (по умолчанию O_RDWR)
	initialFlag := memFile.GetFlag()
	if initialFlag != os.O_RDWR {
		t.Errorf("Expected flag %d, got %d", os.O_RDWR, initialFlag)
	}
	
	// Устанавливаем новый флаг
	newFlag := os.O_RDONLY
	memFile.SetFlag(newFlag)
	
	// Проверяем новое значение флага
	if memFile.GetFlag() != newFlag {
		t.Errorf("Expected flag %d, got %d", newFlag, memFile.GetFlag())
	}
}

// TestPathUtils_IsValidPathChar тестирует проверку допустимых символов в пути
func TestPathUtils_IsValidPathChar(t *testing.T) {
	pathUtils := NewPathUtils()
	
	// Тест допустимых путей
	validPaths := []string{"test.txt", "dir/file.txt", "test_file-123.txt"}
	for _, path := range validPaths {
		if err := pathUtils.ValidatePath(path); err != nil {
			t.Errorf("Path %s should be valid: %v", path, err)
		}
	}
	
	// Тест недопустимых путей
	invalidPaths := []string{"test\x00.txt", "test\x01.txt", "test*.txt", "test?.txt"}
	for _, path := range invalidPaths {
		if err := pathUtils.ValidatePath(path); err == nil {
			t.Errorf("Path %s should be invalid", path)
		}
	}
}

// TestOptions_Additional тестирует дополнительные опции конфигурации
func TestOptions_Additional(t *testing.T) {
	// Тест базовой конфигурации
	baseConfig := Config{
		Type: DiskFS,
	}
	
	// Применяем опции
	config := ApplyOptions(baseConfig, WithType(MemoryFS), WithBasePath("/test"))
	
	if config.Type != MemoryFS {
		t.Errorf("Expected type %v, got %v", MemoryFS, config.Type)
	}
	
	if config.BasePath != "/test" {
		t.Errorf("Expected base path '/test', got '%s'", config.BasePath)
	}
	
	// Тест создания новой конфигурации
	newConfig := NewConfig(WithType(DiskFS), WithBasePath("/test"))
	
	if newConfig.Type != DiskFS {
		t.Errorf("Expected type %v, got %v", DiskFS, newConfig.Type)
	}
	
	if newConfig.BasePath != "/test" {
		t.Errorf("Expected base path '/test', got '%s'", newConfig.BasePath)
	}
}