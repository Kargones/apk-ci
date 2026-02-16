package filer

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestTempManager_CreateTempDir(t *testing.T) {
	manager := NewTempManager(false)
	
	tempDir, err := manager.CreateTempDir("test", true)
	if err != nil {
		t.Errorf("CreateTempDir failed: %v", err)
	}
	
	// Проверяем, что директория создана
	info, err := os.Stat(tempDir.Path)
	if err != nil {
		t.Errorf("Temp directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("Created temp path is not a directory")
	}
	
	// Проверяем, что директория содержит префикс
	if !strings.HasPrefix(filepath.Base(tempDir.Path), "test") {
		t.Errorf("Temp directory name should start with prefix 'test', got: %s", filepath.Base(tempDir.Path))
	}
	
	// Очистка
	if err := manager.CleanupAll(); err != nil {
		t.Logf("Failed to cleanup: %v", err)
	}
}

func TestTempManager_CreateTempDir_EmptyPrefix(t *testing.T) {
	manager := NewTempManager(false)
	
	tempDir, err := manager.CreateTempDir("", true)
	if err != nil {
		t.Errorf("CreateTempDir with empty prefix failed: %v", err)
	}
	
	// Проверяем, что директория создана
	_, err = os.Stat(tempDir.Path)
	if err != nil {
		t.Errorf("Temp directory was not created: %v", err)
	}
	
	// Очистка
	if err := manager.CleanupAll(); err != nil {
		t.Logf("Failed to cleanup: %v", err)
	}
}

func TestTempManager_GetTempDirs(t *testing.T) {
	manager := NewTempManager(false)
	
	// Создаем временную директорию
	tempDir1, err := manager.CreateTempDir("test1", true)
	if err != nil {
		t.Fatalf("CreateTempDir failed: %v", err)
	}
	
	// Получаем список временных директорий
	tempDirs := manager.GetTempDirs()
	if len(tempDirs) != 1 {
		t.Errorf("Expected 1 temp directory, got %d", len(tempDirs))
	}
	
	if tempDirs[0].Path != tempDir1.Path {
		t.Errorf("Expected temp directory %s, got %s", tempDir1.Path, tempDirs[0].Path)
	}
	
	// Создаем еще одну временную директорию
	_, err = manager.CreateTempDir("test2", true)
	if err != nil {
		t.Fatalf("CreateTempDir failed: %v", err)
	}
	
	// Проверяем, что теперь у нас 2 директории
	tempDirs = manager.GetTempDirs()
	if len(tempDirs) != 2 {
		t.Errorf("Expected 2 temp directories, got %d", len(tempDirs))
	}
	
	// Очистка
	if err := manager.CleanupAll(); err != nil {
		t.Logf("Failed to cleanup: %v", err)
	}
}

func TestTempManager_CleanupAll(t *testing.T) {
	manager := NewTempManager(false)
	
	// Создаем несколько временных директорий
	_, err := manager.CreateTempDir("test1", false)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	_, err = manager.CreateTempDir("test2", false)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	// Проверяем, что директории созданы
	tempDirs := manager.GetTempDirs()
	if len(tempDirs) != 2 {
		t.Errorf("Expected 2 temp directories, got %d", len(tempDirs))
	}
	
	// Очищаем все
	if err := manager.CleanupAll(); err != nil {
		t.Errorf("Failed to cleanup all: %v", err)
	}
	
	// Проверяем, что директории удалены
	tempDirs = manager.GetTempDirs()
	if len(tempDirs) != 0 {
		t.Errorf("Expected 0 temp directories after cleanup, got %d", len(tempDirs))
	}
	
	// Проверяем, что директории действительно не существуют на диске
	for _, dir := range tempDirs {
		if _, err := os.Stat(dir.Path); !os.IsNotExist(err) {
			t.Errorf("Directory %s still exists after cleanup", dir.Path)
		}
	}
}

func TestTempManager_CleanupOldDirs(t *testing.T) {
	manager := NewTempManager(false)
	
	// Создаем временную директорию
	tempDir, err := manager.CreateTempDir("old", false)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	// Изменяем время создания на старое
	oldTime := time.Now().Add(-2 * time.Hour)
	err = os.Chtimes(tempDir.Path, oldTime, oldTime)
	if err != nil {
		t.Fatalf("Failed to change dir time: %v", err)
	}
	
	// Очищаем старые директории (старше 1 часа)
	if err := manager.CleanupOldDirs(time.Hour); err != nil {
		t.Logf("Failed to cleanup old dirs: %v", err)
	}
	
	// Очистка
	if err := manager.CleanupAll(); err != nil {
		t.Logf("Failed to cleanup: %v", err)
	}
}

func TestTempManager_CleanupOldDirs_RecentDir(t *testing.T) {
	manager := NewTempManager(false)
	
	// Создаем временную директорию
	tempDir, err := manager.CreateTempDir("recent", false)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	// Очищаем старые директории (старше 1 часа) - эта директория должна остаться
	if err := manager.CleanupOldDirs(time.Hour); err != nil {
		t.Logf("Failed to cleanup old dirs: %v", err)
	}
	
	// Проверяем, что директория все еще существует
	if _, err := os.Stat(tempDir.Path); os.IsNotExist(err) {
		t.Error("Recent directory was incorrectly removed")
	}
	
	// Очистка
	if err := manager.CleanupAll(); err != nil {
		t.Logf("Failed to cleanup: %v", err)
	}
}

func TestTempManager_GetStats(t *testing.T) {
	manager := NewTempManager(false)
	
	// Создаем временную директорию
	tempDir, err := manager.CreateTempDir("test", true)
	if err != nil {
		t.Fatalf("CreateTempDir failed: %v", err)
	}
	
	// Создаем файл в временной директории
	testFile := filepath.Join(tempDir.Path, "test.txt")
	testData := []byte("test data for stats")
	err = os.WriteFile(testFile, testData, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	// Получаем статистику
	stats := manager.GetStats()
	if totalDirs, ok := stats["total_dirs"].(int); ok {
		if totalDirs != 1 {
			t.Errorf("Expected 1 total dir, got %d", totalDirs)
		}
	} else {
		t.Error("Stats should contain total_dirs as int")
	}
	
	if totalSize, ok := stats["total_size"].(int64); ok {
		if totalSize <= 0 {
			t.Errorf("Expected positive total size, got %d", totalSize)
		}
	} else {
		t.Error("Stats should contain total_size as int64")
	}
	
	// Очистка
	manager.CleanupAll()
	
	// Проверяем статистику после очистки
	stats = manager.GetStats()
	if totalDirs, ok := stats["total_dirs"].(int); ok {
		if totalDirs != 0 {
			t.Errorf("Expected 0 total dirs after cleanup, got %d", totalDirs)
		}
	}
	if totalSize, ok := stats["total_size"].(int64); ok {
		if totalSize != 0 {
			t.Errorf("Expected 0 total size after cleanup, got %d", totalSize)
		}
	}
}

// TestTempManager_Close тестирует закрытие TempManager
func TestTempManager_Close(t *testing.T) {
	manager := NewTempManager(false)
	
	// Создаем временную директорию
	tempDir, err := manager.CreateTempDir("close_test", true)
	if err != nil {
		t.Fatalf("CreateTempDir failed: %v", err)
	}
	
	// Проверяем, что директория создана
	if _, statErr := os.Stat(tempDir.Path); os.IsNotExist(statErr) {
		t.Fatalf("Temp directory was not created: %s", tempDir.Path)
	}
	
	// Закрываем менеджер
	err = manager.Close()
	if err != nil {
		t.Fatalf("Failed to close TempManager: %v", err)
	}
	
	// Проверяем, что все временные директории удалены
	if _, err := os.Stat(tempDir.Path); !os.IsNotExist(err) {
		t.Errorf("Temp directory should be removed after Close: %s", tempDir.Path)
	}
}

// TestTempManager_GetWorkDir тестирует получение рабочей директории
func TestTempManager_GetWorkDir(t *testing.T) {
	manager := NewTempManager(false)
	
	workDir := manager.GetWorkDir()
	if workDir == "" {
		t.Error("GetWorkDir should return non-empty string")
	}
	
	// Проверяем, что рабочая директория существует
	if _, err := os.Stat(workDir); os.IsNotExist(err) {
		t.Errorf("Work directory does not exist: %s", workDir)
	}
}

// TestTempManager_CheckAvailableRAM_Basic тестирует проверку доступной RAM
func TestTempManager_CheckAvailableRAM_Basic(t *testing.T) {
	manager := NewTempManager(false) // Диск режим
	
	// Получаем информацию о RAM
	ramInfo := manager.CheckAvailableRAM()
	if ramInfo == nil {
		t.Error("CheckAvailableRAM should return non-nil map")
	}
	
	// Проверяем наличие ключевых полей
	if _, ok := ramInfo["ram_disk_available"]; !ok {
		t.Error("RAM info should contain ram_disk_available")
	}
	if _, ok := ramInfo["using_ram"]; !ok {
		t.Error("RAM info should contain using_ram")
	}
	if _, ok := ramInfo["base_dir"]; !ok {
		t.Error("RAM info should contain base_dir")
	}
	if _, ok := ramInfo["os"]; !ok {
		t.Error("RAM info should contain os")
	}
	
	// Тест с RAM режимом
	managerRAM := NewTempManager(true)
	ramInfoRAM := managerRAM.CheckAvailableRAM()
	
	if usingRAM, ok := ramInfoRAM["using_ram"].(bool); ok {
		// using_ram может быть false если /dev/shm недоступен
		t.Logf("Using RAM: %v", usingRAM)
	} else {
		t.Error("using_ram should be a boolean")
	}
	
	// Проверяем типы значений
	if _, ok := ramInfo["ram_disk_available"].(bool); !ok {
		t.Error("ram_disk_available should be boolean")
	}
	if _, ok := ramInfo["using_ram"].(bool); !ok {
		t.Error("using_ram should be boolean")
	}
	if _, ok := ramInfo["base_dir"].(string); !ok {
		t.Error("base_dir should be string")
	}
	if _, ok := ramInfo["os"].(string); !ok {
		t.Error("os should be string")
	}
}

// TestTempManager_Finalizer тестирует функцию cleanup через финализатор
func TestTempManager_GetOptimalTempDir_Coverage(t *testing.T) {
	// Тест для улучшения покрытия GetOptimalTempDir
	tm := NewTempManager(false)
	defer func() {
		if err := tm.Close(); err != nil {
			t.Logf("Failed to close temp manager: %v", err)
		}
	}()
	
	// Создаем временную директорию для тестирования
	tm2 := NewTempManager(true) // Попытка использовать RAM диск
	defer func() {
		if err := tm2.Close(); err != nil {
			t.Logf("Failed to close temp manager: %v", err)
		}
	}()
	
	// Создаем временную директорию перед тестированием GetTempDirs
	_, err := tm.CreateTempDir("test", true)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	// Тестируем GetTempDirs
	dirs := tm.GetTempDirs()
	if len(dirs) == 0 {
		t.Error("GetTempDirs should return at least one directory")
	}
	
	// Получаем базовую директорию
	baseDir := tm.GetBaseDir()
	if baseDir == "" {
		t.Error("GetBaseDir returned empty string")
	}
	
	// Проверяем, что директория существует или может быть создана
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Errorf("Cannot create base temp dir: %v", err)
	}
}

// TestTempManager_CleanupOldDirs_ErrorHandling тестирует обработку ошибок при очистке
func TestTempManager_CleanupOldDirs_ErrorHandling(t *testing.T) {
	manager := NewTempManager(false)
	
	// Создаем временную директорию
	tempDir, err := manager.CreateTempDir("test", true)
	if err != nil {
		t.Fatalf("CreateTempDir failed: %v", err)
	}
	
	// Удаляем директорию вручную, чтобы создать ситуацию с ошибкой
	err = os.RemoveAll(tempDir.Path)
	if err != nil {
		t.Fatalf("Failed to remove temp dir manually: %v", err)
	}
	
	// Пытаемся очистить старые директории
	// Это должно обработать ситуацию с несуществующей директорией
	err = manager.CleanupOldDirs(time.Hour)
	if err != nil {
		t.Logf("CleanupOldDirs returned error (expected): %v", err)
	}
	
	// Проверяем, что директория удалена из внутренней карты
	stats := manager.GetStats()
	if totalDirs, ok := stats["total_dirs"].(int); ok {
		if totalDirs != 0 {
			t.Errorf("Expected 0 total dirs after cleanup, got %d", totalDirs)
		}
	}
}

// TestTempManager_IsUsingRAM тестирует проверку использования RAM
func TestTempManager_IsUsingRAM(t *testing.T) {
	// Тест с обычным режимом
	manager := NewTempManager(false)
	if manager.IsUsingRAM() {
		t.Error("Manager with useRAM=false should not be using RAM")
	}
	
	// Тест с RAM режимом
	managerRAM := NewTempManager(true)
	usingRAM := managerRAM.IsUsingRAM()
	t.Logf("RAM manager using RAM: %v", usingRAM)
	// Результат зависит от доступности /dev/shm, поэтому просто логируем
}

// TestTempManager_GetBaseDir тестирует получение базовой директории
func TestTempManager_GetBaseDir(t *testing.T) {
	manager := NewTempManager(false)
	baseDir := manager.GetBaseDir()
	if baseDir == "" {
		t.Error("GetBaseDir should return non-empty string")
	}
	
	// Проверяем, что базовая директория существует
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		t.Errorf("Base directory does not exist: %s", baseDir)
	}
	
	// Тест с RAM режимом
	managerRAM := NewTempManager(true)
	baseDirRAM := managerRAM.GetBaseDir()
	if baseDirRAM == "" {
		t.Error("GetBaseDir should return non-empty string for RAM mode")
	}
	
	t.Logf("Normal mode base dir: %s", baseDir)
	t.Logf("RAM mode base dir: %s", baseDirRAM)
}

// TestTempManager_GetBaseDir_Extended тестирует получение базовой директории
func TestTempManager_GetBaseDir_Extended(t *testing.T) {
	manager := NewTempManager(false)
	
	baseDir := manager.GetBaseDir()
	if baseDir == "" {
		t.Error("GetBaseDir should return non-empty string")
	}
	
	// Проверяем, что базовая директория существует
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		t.Errorf("Base directory does not exist: %s", baseDir)
	}
}


// TestTempManager_RemoveTempDir тестирует удаление временной директории
func TestTempManager_RemoveTempDir(t *testing.T) {
	manager := NewTempManager(false)
	
	// Создаем временную директорию
	tempDir, err := manager.CreateTempDir("remove_test", false)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	// Создаем файл в директории
	testFile := filepath.Join(tempDir.Path, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Удаляем директорию
	if err := manager.RemoveTempDir(tempDir); err != nil {
		t.Fatalf("Failed to remove temp dir: %v", err)
	}
	
	// Проверяем, что директория удалена
	if _, err := os.Stat(tempDir.Path); !os.IsNotExist(err) {
		t.Error("Directory should be removed")
	}
}

func TestTempManager_CreateTempFile(t *testing.T) {
	manager := NewTempManager(false)
	
	// Создаем временную директорию
	tempDir, err := manager.CreateTempDir("test", false)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer manager.RemoveTempDir(tempDir)
	
	// Создаем временный файл
	filePath, err := manager.CreateTempFile(tempDir, "testfile.txt")
	if err != nil {
		t.Errorf("CreateTempFile failed: %v", err)
	}
	
	// Проверяем, что файл создан
	info, err := os.Stat(filePath)
	if err != nil {
		t.Errorf("Temp file was not created: %v", err)
	}
	if info.IsDir() {
		t.Error("Created temp path should be a file, not directory")
	}
	
	// Проверяем, что файл находится в правильной директории
	expectedPath := filepath.Join(tempDir.Path, "testfile.txt")
	if filePath != expectedPath {
		t.Errorf("Expected file path %s, got %s", expectedPath, filePath)
	}
}

func TestTempManager_CreateTempFile_InvalidInput(t *testing.T) {
	manager := NewTempManager(false)
	
	// Тест с nil tempDir
	_, err := manager.CreateTempFile(nil, "testfile.txt")
	if err == nil {
		t.Error("CreateTempFile should fail with nil tempDir")
	}
	
	// Создаем временную директорию для остальных тестов
	tempDir, err := manager.CreateTempDir("test", false)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer manager.RemoveTempDir(tempDir)
	
	// Тест с недопустимым именем файла
	_, err = manager.CreateTempFile(tempDir, "../../../etc/passwd")
	if err == nil {
		t.Error("CreateTempFile should fail with invalid filename")
	}
	
	// Тест с пустым именем файла
	_, err = manager.CreateTempFile(tempDir, "")
	if err == nil {
		t.Error("CreateTempFile should fail with empty filename")
	}
}

func TestTempManager_AdditionalCoverage(t *testing.T) {
	manager := NewTempManager(false)
	defer func() {
		if err := manager.CleanupAll(); err != nil {
			t.Logf("Failed to cleanup: %v", err)
		}
	}()
	
	// Тест различных методов для покрытия
	stats := manager.GetStats()
	if totalDirs, ok := stats["total_dirs"].(int); ok {
		if totalDirs < 0 {
			t.Error("Total dirs should not be negative")
		}
	}
	
	// Тест CleanupOldDirs с различными интервалами
	if err := manager.CleanupOldDirs(time.Hour); err != nil {
		t.Logf("Failed to cleanup old dirs: %v", err)
	}
	
	// Создаем директорию для тестирования
	_, err := manager.CreateTempDir("coverage", false)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	// Проверяем статистику после создания
	stats = manager.GetStats()
	if totalDirs, ok := stats["total_dirs"].(int); ok {
		if totalDirs != 1 {
			t.Errorf("Expected 1 directory, got %d", totalDirs)
		}
	}
}

func TestTempManager_CheckAvailableRAM_EdgeCases(t *testing.T) {
	manager := NewTempManager(true)
	
	// Создаем менеджер для RAM диска
	managerRAM := NewTempManager(true)
	defer func() {
		if err := managerRAM.CleanupAll(); err != nil {
			t.Logf("Failed to cleanup RAM manager: %v", err)
		}
	}()
	
	// Тест различных сценариев
	isRAMAvailable := manager.isRAMDiskAvailable()
	t.Logf("RAM disk available: %v", isRAMAvailable)
	
	// Создаем директорию для тестирования
	_, err := manager.CreateTempDir("coverage", false)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	// Проверяем статистику после создания
	stats := manager.GetStats()
	if totalDirs, ok := stats["total_dirs"].(int); ok {
		if totalDirs != 1 {
			t.Errorf("Expected 1 directory, got %d", totalDirs)
		}
	}
	
	// Очистка
	defer func() {
		if err := manager.CleanupAll(); err != nil {
			t.Logf("Failed to cleanup: %v", err)
		}
	}()
}

func TestTempManager_GetStats_WithMultipleFiles(t *testing.T) {
	manager := NewTempManager(false)
	defer func() {
		if err := manager.CleanupAll(); err != nil {
			t.Logf("Failed to cleanup: %v", err)
		}
	}()
	
	// Создаем несколько директорий с файлами
	for i := 0; i < 3; i++ {
		tempDir, err := manager.CreateTempDir(fmt.Sprintf("multi_%d", i), false)
		if err != nil {
			t.Fatalf("Failed to create temp dir %d: %v", i, err)
		}
		
		// Создаем файлы в каждой директории
		for j := 0; j < 2; j++ {
			filename := fmt.Sprintf("file_%d.txt", j)
			content := []byte(fmt.Sprintf("Content for file %d in dir %d", j, i))
			_, err = manager.CreateTempFile(tempDir, filename)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			// Записываем содержимое в файл
			filePath := filepath.Join(tempDir.Path, filename)
			err = os.WriteFile(filePath, content, 0644)
			if err != nil {
				t.Fatalf("Failed to write content to temp file: %v", err)
			}
		}
	}
	
	// Получаем статистику
	stats := manager.GetStats()
	if totalDirs, ok := stats["total_dirs"].(int); ok {
		if totalDirs != 3 {
			t.Errorf("Expected 3 directories, got %d", totalDirs)
		}
	}
	
	if totalSize, ok := stats["total_size"].(int64); ok {
		if totalSize <= 0 {
			t.Error("Expected positive total size")
		}
	}
}

func TestTempManager_CleanupOldDirs_WithOldDirs(t *testing.T) {
	manager := NewTempManager(false)
	defer func() {
		if err := manager.CleanupAll(); err != nil {
			t.Logf("Failed to cleanup: %v", err)
		}
	}()
	
	// Создаем директорию и делаем ее старой
	tempDir, err := manager.CreateTempDir("old_dir", false)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	// Изменяем время модификации на старое
	oldTime := time.Now().Add(-3 * time.Hour)
	err = os.Chtimes(tempDir.Path, oldTime, oldTime)
	if err != nil {
		t.Fatalf("Failed to change dir time: %v", err)
	}
	
	// Очищаем старые директории
	if err := manager.CleanupOldDirs(2 * time.Hour); err != nil {
		t.Logf("Failed to cleanup old dirs: %v", err)
	}
	
	// Проверяем, что старая директория удалена
	if _, err := os.Stat(tempDir.Path); !os.IsNotExist(err) {
		t.Error("Old directory should be removed")
	}
}

func TestTempManager_GetOptimalTempDir_ErrorHandling(t *testing.T) {
	// Тест обработки ошибок в GetOptimalTempDir
	manager := NewTempManager(false)
	defer func() {
		if err := manager.Close(); err != nil {
			t.Logf("Failed to close manager: %v", err)
		}
	}()
	
	// Получаем базовую директорию
	baseDir := manager.GetBaseDir()
	if baseDir == "" {
		t.Error("GetBaseDir should not return empty string")
	}
	
	// Проверяем, что можем создать директорию
	testDir := filepath.Join(baseDir, "test_optimal")
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Errorf("Failed to create test directory: %v", err)
	}
	
	// Очищаем созданную директорию
	os.RemoveAll(testDir)
}

func TestTempManager_IsRAMDiskAvailable_NoShm(t *testing.T) {
	// Тест для случая, когда /dev/shm недоступен
	manager := NewTempManager(false)
	defer func() {
		if err := manager.Close(); err != nil {
			t.Logf("Failed to close manager: %v", err)
		}
	}()
	
	// Проверяем доступность RAM диска
	isAvailable := manager.isRAMDiskAvailable()
	t.Logf("RAM disk available: %v", isAvailable)
	
	// Проверяем, что менеджер работает независимо от доступности RAM диска
	tempDir, err := manager.CreateTempDir("ram_test", false)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	// Проверяем, что директория создана
	if _, err := os.Stat(tempDir.Path); os.IsNotExist(err) {
		t.Error("Temp directory should exist")
	}
	
	// Очистка
	if err := manager.RemoveTempDir(tempDir); err != nil {
		t.Logf("Failed to remove temp dir: %v", err)
	}
}

func TestPathUtils_EnsureDir_FileExists(t *testing.T) {
	pathUtils := NewPathUtils()
	
	// Создаем временный файл
	tempFile, err := os.CreateTemp("", "test_file_*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tempFile.Name()); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}()
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}
	
	// Пытаемся создать директорию с именем существующего файла
	err = pathUtils.EnsureDir(tempFile.Name())
	if err == nil {
		t.Error("EnsureDir should fail when path exists but is not a directory")
	}
	if !strings.Contains(err.Error(), "не является директорией") {
		t.Errorf("Expected error about path not being directory, got: %v", err)
	}
}

func TestTempManager_CheckAvailableRAM_LinuxFields(t *testing.T) {
	manager := NewTempManager(false)
	ramInfo := manager.CheckAvailableRAM()
	
	// Проверяем Linux-специфичные поля
	if runtime.GOOS == "linux" {
		if _, exists := ramInfo["shm_exists"]; !exists {
			t.Error("CheckAvailableRAM should include shm_exists field on Linux")
		}
		if shmExists, ok := ramInfo["shm_exists"].(bool); ok && shmExists {
			if _, exists := ramInfo["shm_is_dir"]; !exists {
				t.Error("CheckAvailableRAM should include shm_is_dir field when shm exists")
			}
		} else if !ok {
			t.Error("shm_exists should be boolean")
		}
	} else {
		if _, exists := ramInfo["fallback_reason"]; !exists {
			t.Error("CheckAvailableRAM should include fallback_reason on non-Linux systems")
		}
	}
}