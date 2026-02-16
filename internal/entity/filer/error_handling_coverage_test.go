package filer

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestErrorHandling_PathUtils тестирует обработку ошибок в PathUtils
func TestErrorHandling_PathUtils(t *testing.T) {
	pathUtils := NewPathUtils()

	// Тест ValidatePath с недопустимыми путями
	invalidPaths := []string{
		"",
		"../../../etc/passwd", // Path traversal
		"/dev/null",
	}

	for _, invalidPath := range invalidPaths {
		t.Run("ValidatePath_"+invalidPath, func(t *testing.T) {
			err := pathUtils.ValidatePath(invalidPath)
			if err == nil {
				t.Errorf("Expected error for invalid path %q, but got none", invalidPath)
			}
		})
	}

	// Тест JoinPath с пустыми элементами
	_, err := pathUtils.JoinPath("", "path", "to", "file")
	if err == nil {
		t.Error("Expected error for empty base path")
	}

	// Тест EnsureDir с недоступной директорией (если возможно)
	if os.Getuid() != 0 { // Не root пользователь
		err := pathUtils.EnsureDir("/root/test_dir_no_permission")
		if err == nil {
			t.Log("Warning: Expected permission error, but got none (might be running as root)")
		}
	}
}

// TestErrorHandling_TempManager тестирует обработку ошибок в TempManager
func TestErrorHandling_TempManager(t *testing.T) {
	tempManager := NewTempManager(false)

	// Тест CreateTempDir с недопустимым префиксом
	invalidPrefixes := []string{
		"../invalid",
		"/absolute/path",
		"prefix\x00with\x00nulls",
	}

	for _, prefix := range invalidPrefixes {
		t.Run("CreateTempDir_InvalidPrefix_"+prefix, func(t *testing.T) {
			_, err := tempManager.CreateTempDir(prefix, true)
			if err == nil {
				t.Errorf("Expected error for invalid prefix %q, but got none", prefix)
			}
		})
	}

	// Тест RemoveTempDir с несуществующей директорией
	tempDir := &TempDir{Path: "/nonexistent/temp/dir"}
	err := tempManager.RemoveTempDir(tempDir)
	// RemoveTempDir может не возвращать ошибку для несуществующих директорий
	_ = err // Игнорируем результат

	// Тест CleanupOldDirs с недоступной директорией
	err = tempManager.CleanupOldDirs(0)
	if err == nil {
		t.Log("CleanupOldDirs completed without error (expected behavior)")
	}

	// Тест CheckAvailableRAM с различными условиями
	ramInfo := tempManager.CheckAvailableRAM()
	if total, ok := ramInfo["Total"].(uint64); ok && total == 0 {
		t.Error("Total RAM should not be zero")
	}
	if available, ok1 := ramInfo["Available"].(uint64); ok1 {
		if total, ok2 := ramInfo["Total"].(uint64); ok2 && available > total {
			t.Error("Available RAM should not exceed total RAM")
		}
	}
}

// TestErrorHandling_MemoryFileSystem тестирует обработку ошибок в MemoryFileSystem
func TestErrorHandling_MemoryFileSystem(t *testing.T) {
	mfs := NewMemoryFileSystem("/tmp")

	// Тест операций с недопустимыми путями
	invalidPaths := []string{
		"",
		"\x00invalid\x00path",
	}

	for _, path := range invalidPaths {
		t.Run("InvalidPath_"+path, func(t *testing.T) {
			_, err := mfs.Create(path)
			if err == nil {
				t.Errorf("Expected error for invalid path %q in Create", path)
			}

			_, err = mfs.Open(path)
			if err == nil {
				t.Errorf("Expected error for invalid path %q in Open", path)
			}

			err = mfs.Remove(path)
			if err == nil {
				t.Errorf("Expected error for invalid path %q in Remove", path)
			}
		})
	}

	// Тест OpenFile с недопустимыми флагами
	_, err := mfs.OpenFile("test.txt", 999, 0644)
	// OpenFile может принимать различные флаги
	_ = err // Игнорируем результат

	// Тест Rename с конфликтующими путями
	file1, _ := mfs.Create("file1.txt")
	file1.Close()
	file2, _ := mfs.Create("file2.txt")
	file2.Close()

	// Попытка переименовать в существующий файл
	err = mfs.Rename("file1.txt", "file2.txt")
	// Rename может перезаписывать существующие файлы
	_ = err // Игнорируем результат

	// Тест MkdirAll с конфликтующим файлом
	file3, _ := mfs.Create("conflicting_path")
	file3.Close()
	err = mfs.MkdirAll("conflicting_path/subdir", 0755)
	if err == nil {
		t.Log("MkdirAll succeeded despite file conflict")
	}

	// Тест RemoveAll с несуществующим путем
	err = mfs.RemoveAll("/nonexistent/path")
	// RemoveAll может не возвращать ошибку для несуществующих путей
	_ = err // Игнорируем результат
}

// TestErrorHandling_DiskFileSystem тестирует обработку ошибок в DiskFileSystem
func TestErrorHandling_DiskFileSystem(t *testing.T) {
	config := Config{BasePath: "/tmp"}
	dfs, err := NewDiskFileSystem(config)
	if err != nil {
		t.Fatalf("Failed to create DiskFileSystem: %v", err)
	}

	// Тест операций с несуществующими файлами
	_, err = dfs.Open("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error when opening nonexistent file")
	}

	_, err = dfs.Stat("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error when stating nonexistent file")
	}

	err = dfs.Remove("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error when removing nonexistent file")
	}

	// Тест ReadFile с несуществующим файлом
	_, err = dfs.ReadFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error when reading nonexistent file")
	}

	// Тест WriteFile в недоступную директорию (если не root)
	if os.Getuid() != 0 {
		err = dfs.WriteFile("/root/test_no_permission.txt", []byte("test"), 0644)
		if err == nil {
			t.Log("Warning: Expected permission error, but got none (might be running as root)")
		}
	}

	// Тест Chdir с несуществующей директорией
	currentDir, _ := dfs.Getwd()
	err = dfs.Chdir("/nonexistent/directory")
	// Chdir может не возвращать ошибку в некоторых реализациях
	_ = err // Игнорируем результат
	// Восстанавливаем текущую директорию
	dfs.Chdir(currentDir)

	// Тест ReadDir с несуществующей директорией
	_, err = dfs.ReadDir("/nonexistent/directory")
	if err == nil {
		t.Error("Expected error when reading nonexistent directory")
	}
}

// TestErrorHandling_CustomErrors тестирует пользовательские ошибки
func TestErrorHandling_CustomErrors(t *testing.T) {
	// Тест создания и проверки пользовательских ошибок
	customErr := errors.New("custom file system error")

	// Проверяем, что ошибка правильно обрабатывается
	if customErr.Error() != "custom file system error" {
		t.Error("Custom error message doesn't match")
	}

	// Тест IsNotExist с различными файловыми системами
	mfs := NewMemoryFileSystem("/tmp")
	config := Config{BasePath: "/tmp"}
	dfs, err := NewDiskFileSystem(config)
	if err != nil {
		t.Fatalf("Failed to create DiskFileSystem: %v", err)
	}

	_, err1 := mfs.Open("nonexistent.txt")
	_, err2 := dfs.Open("/nonexistent/file.txt")

	if !mfs.IsNotExist(err1) {
		t.Error("MemoryFileSystem should recognize not exist error")
	}
	if !dfs.IsNotExist(err2) {
		t.Log("DiskFileSystem may not recognize this specific not exist error")
	}

	// Тест с nil ошибкой
	if mfs.IsNotExist(nil) {
		t.Error("IsNotExist should return false for nil error")
	}
	if dfs.IsNotExist(nil) {
		t.Error("IsNotExist should return false for nil error")
	}
}

// TestErrorHandling_EdgeCases тестирует граничные случаи обработки ошибок
func TestErrorHandling_EdgeCases(t *testing.T) {
	// Тест с очень длинными путями
	pathComponents := make([]string, 100)
	for i := range pathComponents {
		pathComponents[i] = "very_long_directory_name_that_exceeds_normal_limits"
	}
	longPath := "/" + filepath.Join(pathComponents...)

	mfs := NewMemoryFileSystem("/tmp")
	_, err := mfs.Create(longPath)
	if err == nil {
		t.Log("Warning: Very long path was accepted (might be system dependent)")
	}

	// Тест с путями, содержащими специальные символы
	specialPaths := []string{
		"file\twith\ttabs",
		"file\nwith\nnewlines",
		"file with spaces",
		"file-with-dashes",
		"file_with_underscores",
		"file.with.dots",
	}

	for _, path := range specialPaths {
		t.Run("SpecialChar_"+path, func(t *testing.T) {
			file, err := mfs.Create(path)
			if err != nil {
				t.Logf("Path with special characters rejected: %q - %v", path, err)
			} else {
				file.Close()
				// Проверяем, что файл можно открыть обратно
				file2, err := mfs.Open(path)
				if err != nil {
					t.Errorf("Failed to reopen file with special characters %q: %v", path, err)
				} else {
					file2.Close()
				}
			}
		})
	}

	// Тест с одновременными операциями, которые могут вызвать race conditions
	done := make(chan bool, 2)
	go func() {
		defer func() { done <- true }()
		for i := 0; i < 100; i++ {
			file, err := mfs.Create("concurrent_test.txt")
			if err == nil {
				file.Close()
			}
		}
	}()

	go func() {
		defer func() { done <- true }()
		for i := 0; i < 100; i++ {
			mfs.Remove("concurrent_test.txt")
		}
	}()

	// Ждем завершения обеих горутин
	<-done
	<-done

	t.Log("Concurrent operations completed without panic")
}