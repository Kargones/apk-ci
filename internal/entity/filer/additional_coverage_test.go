package filer

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// TestMemoryFS_CreateTemp тестирует CreateTemp с различными сценариями
func TestMemoryFS_CreateTemp(t *testing.T) {
	mfs := NewMemoryFileSystem("")

	// Создаем директорию для тестов
	err := mfs.MkdirAll("tmp", 0755)
	if err != nil {
		t.Fatalf("Failed to create tmp directory: %v", err)
	}

	// Тест с валидным префиксом
	file, err := mfs.CreateTemp("tmp", "test_")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer file.Close()

	// Проверяем, что файл создан
	name := file.Name()
	if !strings.Contains(name, "test_") {
		t.Error("Temp file name should contain prefix")
	}

	// Тест с пустым префиксом
	file2, err := mfs.CreateTemp("tmp", "")
	if err != nil {
		t.Fatalf("Failed to create temp file with empty prefix: %v", err)
	}
	defer file2.Close()

	// Тест с несуществующей директорией (MemoryFS создает директории автоматически)
	file5, err := mfs.CreateTemp("nonexistent", "test_")
	if err != nil {
		t.Logf("CreateTemp in nonexistent directory: %v", err)
	} else {
		file5.Close()
	}

	// Тест с корневой директорией
	file6, err := mfs.CreateTemp(".", "root_")
	if err != nil {
		t.Logf("CreateTemp in root directory: %v", err)
	} else {
		if err := file6.Close(); err != nil {
			t.Logf("Failed to close file6: %v", err)
		}
	}
}

// TestMemoryFile_WriteAt_Extended тестирует WriteAt с различными сценариями
func TestMemoryFile_WriteAt_Extended(t *testing.T) {
	mfs := NewMemoryFileSystem("")
	file, err := mfs.Create("test.txt")
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()

	mf := file.(*MemoryFile)

	// Записываем начальные данные
	initialData := []byte("Hello World")
	_, err = mf.Write(initialData)
	if err != nil {
		t.Fatalf("Failed to write initial data: %v", err)
	}

	// Тест WriteAt в середину
	newData := []byte("XXX")
	n, err := mf.WriteAt(newData, 6)
	if err != nil {
		t.Fatalf("Failed to write at position: %v", err)
	}
	if n != len(newData) {
		t.Errorf("Expected to write %d bytes, got %d", len(newData), n)
	}

	// Проверяем результат
	mf.Seek(0, io.SeekStart)
	result := make([]byte, 20)
	n, _ = mf.Read(result)
	resultStr := string(result[:n])
	expected := "Hello XXXld"
	if resultStr != expected {
		t.Errorf("Expected %q, got %q", expected, resultStr)
	}

	// Тест WriteAt за пределами файла
	n, err = mf.WriteAt([]byte("END"), 20)
	if err != nil {
		t.Fatalf("Failed to write beyond file: %v", err)
	}
	if n != 3 {
		t.Errorf("Expected to write 3 bytes, got %d", n)
	}

	// Тест WriteAt с отрицательным offset
	_, err = mf.WriteAt([]byte("test"), -1)
	if err == nil {
		t.Error("Expected error when writing at negative offset")
	}
}

// TestDiskFS_Rename тестирует Rename с различными сценариями
func TestDiskFS_Rename(t *testing.T) {
	tempDir := t.TempDir()
	config := Config{
		Type:     DiskFS,
		BasePath: tempDir,
	}
	dfs, err := NewDiskFileSystem(config)
	if err != nil {
		t.Fatalf("Failed to create DiskFileSystem: %v", err)
	}

	// Создаем тестовый файл
	srcPath := "source.txt"
	dstPath := "dest.txt"

	err = dfs.WriteFile(srcPath, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Тест успешного переименования
	err = dfs.Rename(srcPath, dstPath)
	if err != nil {
		t.Fatalf("Failed to rename file: %v", err)
	}

	// Проверяем, что файл переименован
	if _, statErr := dfs.Stat(srcPath); statErr == nil {
		t.Error("Source file should not exist after rename")
	}
	if _, statErr := dfs.Stat(dstPath); statErr != nil {
		t.Error("Destination file should exist after rename")
	}

	// Тест переименования несуществующего файла
	err = dfs.Rename("nonexistent.txt", "dest2.txt")
	if err == nil {
		t.Error("Expected error when renaming nonexistent file")
	}

	// Тест переименования в существующий файл
	err = dfs.WriteFile(srcPath, []byte("new content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create new source file: %v", err)
	}

	err = dfs.Rename(srcPath, dstPath) // dstPath уже существует
	if err != nil {
		t.Logf("Rename to existing file may fail on some systems: %v", err)
	}
}

// TestDiskFS_WriteFile тестирует WriteFile с различными сценариями
func TestDiskFS_WriteFile(t *testing.T) {
	tempDir := t.TempDir()
	config := Config{
		Type:     DiskFS,
		BasePath: tempDir,
	}
	dfs, err := NewDiskFileSystem(config)
	if err != nil {
		t.Fatalf("Failed to create DiskFileSystem: %v", err)
	}

	// Тест записи в новый файл
	filePath := "test.txt" //nolint:goconst // test value
	data := []byte("Hello, World!")
	err = dfs.WriteFile(filePath, data, 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Проверяем содержимое
	readData, err := dfs.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(readData) != string(data) {
		t.Errorf("Expected %q, got %q", string(data), string(readData))
	}

	// Тест перезаписи существующего файла
	newData := []byte("New content")
	err = dfs.WriteFile(filePath, newData, 0644)
	if err != nil {
		t.Fatalf("Failed to overwrite file: %v", err)
	}

	readData, err = dfs.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read overwritten file: %v", err)
	}
	if string(readData) != string(newData) {
		t.Errorf("Expected %q, got %q", string(newData), string(readData))
	}

	// Тест записи в несуществующую директорию (может создаваться автоматически)
	invalidPath := "nonexistent/test.txt"
	err = dfs.WriteFile(invalidPath, data, 0644)
	if err != nil {
		t.Logf("WriteFile to nonexistent directory failed as expected: %v", err)
	} else {
		// Если файл создался, проверим его содержимое
		readData, readErr := dfs.ReadFile(invalidPath)
		if readErr == nil && string(readData) == string(data) {
			t.Log("WriteFile successfully created directory and file")
		}
	}

	// Тест с невалидными правами доступа
	restrictedPath := "restricted.txt"
	err = dfs.WriteFile(restrictedPath, data, 0000)
	if err != nil {
		t.Fatalf("Failed to write file with restricted permissions: %v", err)
	}

	// Проверяем права доступа
	info, err := dfs.Stat(restrictedPath)
	if err != nil {
		t.Fatalf("Failed to stat restricted file: %v", err)
	}
	if info.Mode().Perm() != 0000 {
		t.Errorf("Expected permissions 0000, got %o", info.Mode().Perm())
	}
}

// TestErrorHandling_DetermineSeverity тестирует determineSeverity
func TestErrorHandling_DetermineSeverity(t *testing.T) {
	// Тест с различными типами ошибок
	tests := []struct {
		name     string
		err      error
		expected ErrorSeverity
	}{
		{"nil error", nil, SeverityInfo},
		{"permission denied", os.ErrPermission, SeverityError},
		{"file not found", os.ErrNotExist, SeverityWarning},
		{"file exists", os.ErrExist, SeverityWarning},
		{"generic error", errors.New("generic error"), SeverityError},
		{"timeout error", errors.New("timeout"), SeverityError},
		{"network error", errors.New("network unreachable"), SeverityError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineSeverity(tt.err)
			if result != tt.expected {
				t.Errorf("Expected severity %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestErrorHandling_IsRetryableError тестирует IsRetryableError
func TestErrorHandling_IsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"permission denied", os.ErrPermission, false},
		{"file not found", os.ErrNotExist, false},
		{"timeout error", errors.New("timeout"), false},
		{"temporary error", errors.New("temporary failure"), false},
		{"network error", errors.New("network unreachable"), false},
		{"generic error", errors.New("generic error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected retryable %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestFactory_CreateTempFileSystem_Additional тестирует CreateTempFileSystem
func TestFactory_CreateTempFileSystem_Additional(t *testing.T) {
	factory, err := NewFactory()
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	defer factory.Cleanup()

	// Тест создания временной файловой системы
	fs, err := factory.CreateTempFileSystem(DiskFS)
	if err != nil {
		t.Fatalf("Failed to create temp filesystem: %v", err)
	}

	// Проверяем, что это DiskFileSystem
	if _, ok := fs.(*DiskFileSystem); !ok {
		t.Error("Expected DiskFileSystem")
	}

	// Тест создания файла во временной системе
	testFile := "test.txt"
	err = fs.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to write to temp filesystem: %v", err)
	}

	// Проверяем, что файл существует
	_, err = fs.Stat(testFile)
	if err != nil {
		t.Fatalf("File should exist in temp filesystem: %v", err)
	}
}

// TestMemoryFS_ReadDir тестирует ReadDir с различными сценариями
func TestMemoryFS_ReadDir(t *testing.T) {
	mfs := NewMemoryFileSystem("")

	// Создаем тестовую структуру
	mfs.MkdirAll("test", 0755)
	mfs.WriteFile("test/file1.txt", []byte("content1"), 0644)
	mfs.WriteFile("test/file2.txt", []byte("content2"), 0644)
	mfs.MkdirAll("test/subdir", 0755)

	// Тест чтения директории
	entries, err := mfs.ReadDir("test")
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(entries))
	}

	// Проверяем имена файлов
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}

	expectedNames := []string{"file1.txt", "file2.txt", "subdir"}
	for _, expected := range expectedNames {
		found := false
		for _, name := range names {
			if name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find %s in directory listing", expected)
		}
	}

	// Тест чтения несуществующей директории
	_, err = mfs.ReadDir("nonexistent")
	if err == nil {
		t.Error("Expected error when reading nonexistent directory")
	}

	// Тест чтения файла как директории
	_, err = mfs.ReadDir("test/file1.txt")
	if err == nil {
		t.Error("Expected error when reading file as directory")
	}
}

// TestTempManager_CleanupOldDirs_Extended тестирует CleanupOldDirs с различными сценариями
func TestTempManager_CleanupOldDirs_Extended(t *testing.T) {
	tm := NewTempManager(false)
	defer tm.CleanupAll()

	// Создаем старые директории через TempManager
	oldDir1, err := tm.CreateTempDir("old1", true)
	if err != nil {
		t.Fatalf("Failed to create old directory 1: %v", err)
	}

	oldDir2, err := tm.CreateTempDir("old2", true)
	if err != nil {
		t.Fatalf("Failed to create old directory 2: %v", err)
	}

	newDir, err := tm.CreateTempDir("new", true)
	if err != nil {
		t.Fatalf("Failed to create new directory: %v", err)
	}

	// Устанавливаем старое время модификации для старых директорий
	oldTime := time.Now().Add(-25 * time.Hour)
	os.Chtimes(oldDir1.Path, oldTime, oldTime)
	os.Chtimes(oldDir2.Path, oldTime, oldTime)

	// Тест очистки старых директорий
	err = tm.CleanupOldDirs(24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to cleanup old directories: %v", err)
	}

	// Проверяем, что старые директории удалены
	if _, statErr := os.Stat(oldDir1.Path); statErr == nil {
		t.Error("Old directory should be removed")
	}
	if _, statErr2 := os.Stat(oldDir2.Path); statErr2 == nil {
		t.Error("Old directory should be removed")
	}

	// Проверяем, что новая директория осталась
	if _, statErr3 := os.Stat(newDir.Path); statErr3 != nil {
		t.Error("New directory should remain")
	}

	// Тест с нулевым maxAge
	err = tm.CleanupOldDirs(0)
	if err != nil {
		t.Fatalf("Failed to cleanup with zero maxAge: %v", err)
	}
}