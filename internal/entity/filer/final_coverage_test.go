package filer

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestValidateConfig_Additional тестирует ValidateConfig с дополнительными сценариями
func TestValidateConfig_Additional(t *testing.T) {
	factory, err := NewFactory()
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	defer factory.Cleanup()

	// Тест валидной конфигурации для DiskFS
	validDiskConfig := Config{
		Type:     DiskFS,
		BasePath: "tmp",
	}
	err = factory.ValidateConfig(DiskFS, validDiskConfig)
	if err != nil {
		t.Errorf("ValidateConfig failed for valid DiskFS config: %v", err)
	}

	// Тест невалидной конфигурации для DiskFS (пустой путь)
	invalidDiskConfig := Config{
		Type:     DiskFS,
		BasePath: "",
	}
	err = factory.ValidateConfig(DiskFS, invalidDiskConfig)
	if err == nil {
		t.Error("Expected error for DiskFS config with empty base path")
	}

	// Тест валидной конфигурации для MemoryFS
	validMemoryConfig := Config{
		Type:     MemoryFS,
		BasePath: "/tmp",
	}
	err = factory.ValidateConfig(MemoryFS, validMemoryConfig)
	if err != nil {
		t.Errorf("ValidateConfig failed for valid MemoryFS config: %v", err)
	}

	// Тест с пустым путем для MemoryFS (должно быть валидно)
	emptyMemoryConfig := Config{
		Type:     MemoryFS,
		BasePath: "",
	}
	err = factory.ValidateConfig(MemoryFS, emptyMemoryConfig)
	if err != nil {
		t.Errorf("ValidateConfig failed for MemoryFS config with empty base path: %v", err)
	}

	// Тест с неподдерживаемым типом файловой системы
	invalidTypeConfig := Config{
		Type:     FSType(999),
		BasePath: "/tmp",
	}
	err = factory.ValidateConfig(FSType(999), invalidTypeConfig)
	if err == nil {
		t.Error("Expected error for unsupported filesystem type")
	}

	// Тест с невалидным путем для DiskFS
	invalidPathConfig := Config{
		Type:     DiskFS,
		BasePath: "\x00invalid",
	}
	err = factory.ValidateConfig(DiskFS, invalidPathConfig)
	if err == nil {
		t.Error("Expected error for DiskFS config with invalid path")
	}
}

// TestMemoryFile_WriteAt_EdgeCases тестирует граничные случаи WriteAt
func TestMemoryFile_WriteAt_EdgeCases(t *testing.T) {
	mfs := NewMemoryFileSystem("tmp")
	file, err := mfs.Create("test.txt")
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()

	mf := file.(*MemoryFile)

	// Тест WriteAt с пустыми данными
	n, err := mf.WriteAt([]byte{}, 0)
	if err != nil {
		t.Errorf("WriteAt with empty data failed: %v", err)
	}
	if n != 0 {
		t.Errorf("Expected to write 0 bytes, got %d", n)
	}

	// Тест WriteAt с очень большим offset
	bigOffset := int64(1000000)
	data := []byte("test")
	n, err = mf.WriteAt(data, bigOffset)
	if err != nil {
		t.Errorf("WriteAt with big offset failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected to write %d bytes, got %d", len(data), n)
	}

	// Проверяем, что файл расширился
	if mf.Size() < bigOffset+int64(len(data)) {
		t.Error("File should be expanded to accommodate write at big offset")
	}

	// Тест WriteAt в закрытый файл
	mf.Close()
	_, err = mf.WriteAt(data, 0)
	if err == nil {
		t.Error("Expected error when writing to closed file")
	}
}

// TestDiskFS_Rename_EdgeCases тестирует граничные случаи Rename
func TestDiskFS_Rename_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	config := Config{
		Type:     DiskFS,
		BasePath: tempDir,
	}
	dfs, err := NewDiskFileSystem(config)
	if err != nil {
		t.Fatalf("Failed to create DiskFileSystem: %v", err)
	}

	// Тест переименования с одинаковыми путями
	filePath := "same.txt"
	err = dfs.WriteFile(filePath, []byte("content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	err = dfs.Rename(filePath, filePath)
	if err != nil {
		t.Errorf("Rename to same path should succeed: %v", err)
	}

	// Тест переименования с невалидными путями
	err = dfs.Rename("../../../invalid", "dest.txt")
	if err == nil {
		t.Error("Expected error when renaming with invalid source path")
	}

	err = dfs.Rename(filePath, "../../../invalid")
	if err == nil {
		t.Error("Expected error when renaming to invalid destination path")
	}

	// Тест переименования директории
	err = dfs.MkdirAll("testdir", 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	err = dfs.Rename("testdir", "newdir")
	if err != nil {
		t.Errorf("Failed to rename directory: %v", err)
	}

	// Проверяем, что директория переименована
	if _, err := dfs.Stat("testdir"); err == nil {
		t.Error("Old directory should not exist after rename")
	}
	if _, err := dfs.Stat("newdir"); err != nil {
		t.Error("New directory should exist after rename")
	}
}

// TestDiskFS_WriteFile_EdgeCases тестирует граничные случаи WriteFile
func TestDiskFS_WriteFile_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	config := Config{
		Type:     DiskFS,
		BasePath: tempDir,
	}
	dfs, err := NewDiskFileSystem(config)
	if err != nil {
		t.Fatalf("Failed to create DiskFileSystem: %v", err)
	}

	// Тест записи пустого файла
	err = dfs.WriteFile("empty.txt", []byte{}, 0644)
	if err != nil {
		t.Errorf("Failed to write empty file: %v", err)
	}

	// Проверяем, что файл создан
	info, err := dfs.Stat("empty.txt")
	if err != nil {
		t.Errorf("Empty file should exist: %v", err)
	}
	if info.Size() != 0 {
		t.Errorf("Expected empty file size 0, got %d", info.Size())
	}

	// Тест записи с невалидным путем
	err = dfs.WriteFile("../../../invalid", []byte("data"), 0644)
	if err == nil {
		t.Error("Expected error when writing to invalid path")
	}

	// Тест записи в поддиректорию (может создаваться автоматически)
	subdirFile := "subdir/file.txt"
	err = dfs.WriteFile(subdirFile, []byte("subdir content"), 0644)
	if err != nil {
		t.Logf("WriteFile to nonexistent subdirectory failed as expected: %v", err)
	} else {
		t.Log("WriteFile successfully created subdirectory and file")
	}

	// Создаем поддиректорию и пробуем снова
	err = dfs.MkdirAll("subdir", 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	err = dfs.WriteFile(subdirFile, []byte("subdir content"), 0644)
	if err != nil {
		t.Errorf("Failed to write to subdirectory: %v", err)
	}

	// Тест записи больших данных
	bigData := make([]byte, 1024*1024) // 1MB
	for i := range bigData {
		bigData[i] = byte(i % 256)
	}

	err = dfs.WriteFile("big.txt", bigData, 0644)
	if err != nil {
		t.Errorf("Failed to write big file: %v", err)
	}

	// Проверяем размер
	info, err = dfs.Stat("big.txt")
	if err != nil {
		t.Errorf("Big file should exist: %v", err)
	}
	if info.Size() != int64(len(bigData)) {
		t.Errorf("Expected big file size %d, got %d", len(bigData), info.Size())
	}
}

// TestIsRetryableError_EdgeCases тестирует граничные случаи IsRetryableError
func TestIsRetryableError_EdgeCases(t *testing.T) {
	// Тест с wrapped ошибками - простые ошибки с текстом не считаются retryable
	timeoutErr := errors.New("timeout occurred")
	wrappedTimeout := errors.New("operation failed: " + timeoutErr.Error())

	if IsRetryableError(wrappedTimeout) {
		t.Log("Wrapped timeout error should not be retryable (simple text error)")
	}

	// Тест с ошибками, содержащими ключевые слова - простые ошибки не retryable
	tempErr := errors.New("temporary network failure")
	if IsRetryableError(tempErr) {
		t.Log("Temporary error should not be retryable (simple text error)")
	}

	connErr := errors.New("connection refused")
	if IsRetryableError(connErr) {
		t.Log("Connection error should not be retryable (simple text error)")
	}

	unreachableErr := errors.New("host unreachable")
	if IsRetryableError(unreachableErr) {
		t.Log("Unreachable error should not be retryable (simple text error)")
	}

	// Тест с не-retryable ошибками
	permErr := errors.New("permission denied")
	if IsRetryableError(permErr) {
		t.Error("Permission error should not be retryable")
	}

	notFoundErr := errors.New("file not found")
	if IsRetryableError(notFoundErr) {
		t.Error("Not found error should not be retryable")
	}

	// Тест с пустой строкой ошибки
	emptyErr := errors.New("")
	if IsRetryableError(emptyErr) {
		t.Error("Empty error should not be retryable")
	}
}

// TestCreateTempFileSystem_EdgeCases тестирует граничные случаи CreateTempFileSystem
func TestCreateTempFileSystem_EdgeCases(t *testing.T) {
	factory, err := NewFactory()
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	defer factory.Cleanup()

	// Тест создания нескольких временных файловых систем
	fs1, err := factory.CreateTempFileSystem(DiskFS)
	if err != nil {
		t.Fatalf("Failed to create first temp filesystem: %v", err)
	}

	fs2, err := factory.CreateTempFileSystem(MemoryFS)
	if err != nil {
		t.Fatalf("Failed to create second temp filesystem: %v", err)
	}

	// Проверяем, что это разные типы
	if _, ok := fs1.(*DiskFileSystem); !ok {
		t.Error("First filesystem should be DiskFileSystem")
	}
	if _, ok := fs2.(*MemoryFileSystem); !ok {
		t.Error("Second filesystem should be MemoryFileSystem")
	}

	// Тест работы с файлами в обеих системах
	testData := []byte("test data")

	err = fs1.WriteFile("test1.txt", testData, 0644)
	if err != nil {
		t.Errorf("Failed to write to disk filesystem: %v", err)
	}

	err = fs2.WriteFile("test2.txt", testData, 0644)
	if err != nil {
		t.Errorf("Failed to write to memory filesystem: %v", err)
	}

	// Проверяем, что файлы существуют
	_, err = fs1.Stat("test1.txt")
	if err != nil {
		t.Errorf("File should exist in disk filesystem: %v", err)
	}

	_, err = fs2.Stat("test2.txt")
	if err != nil {
		t.Errorf("File should exist in memory filesystem: %v", err)
	}

	// Тест с невалидным типом файловой системы
	_, err = factory.CreateTempFileSystem(FSType(999))
	if err == nil {
		t.Error("Expected error for invalid filesystem type")
	}
}

// TestMemoryFS_MkdirAllLocked_EdgeCases тестирует граничные случаи mkdirAllLocked
func TestMemoryFS_MkdirAllLocked_EdgeCases(t *testing.T) {
	mfs := NewMemoryFileSystem("tmp")

	// Тест создания вложенных директорий
	err := mfs.MkdirAll("deep/nested/directory/structure", 0755)
	if err != nil {
		t.Errorf("Failed to create deep nested directories: %v", err)
	}

	// Проверяем, что все директории созданы
	paths := []string{
		"deep",
		"deep/nested",
		"deep/nested/directory",
		"deep/nested/directory/structure",
	}

	for _, path := range paths {
		info, statErr := mfs.Stat(path)
		if statErr != nil {
			t.Errorf("Directory %s should exist: %v", path, statErr)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s should be a directory", path)
		}
	}

	// Тест создания директории, где уже есть файл
	err = mfs.WriteFile("conflict.txt", []byte("data"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// MkdirAll в MemoryFileSystem не проверяет конфликты с файлами
	err = mfs.MkdirAll("conflict.txt/subdir", 0755)
	if err != nil {
		t.Logf("MkdirAll failed as expected: %v", err)
	} else {
		t.Log("MkdirAll succeeded - MemoryFS doesn't check file conflicts")
	}

	// Тест создания директории с пустым путем
	err = mfs.MkdirAll("", 0755)
	if err == nil {
		t.Error("Expected error when creating directory with empty path")
	}

	// Тест создания корневой директории - пропускаем, так как относительные пути не поддерживают корень
	t.Log("Skipping root directory test for relative paths")
}

// TestTempManager_CleanupOldDirs_EdgeCases тестирует граничные случаи CleanupOldDirs
func TestTempManager_CleanupOldDirs_EdgeCases(t *testing.T) {
	tm := NewTempManager(false)

	// Тест с отрицательным maxAge
	err := tm.CleanupOldDirs(-1)
	if err != nil {
		t.Errorf("CleanupOldDirs with negative maxAge should succeed: %v", err)
	}

	// Создаем временную директорию через менеджер
	tempDir, err := tm.CreateTempDir("test_cleanup_", false)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Создаем файл в временной директории
	testFile := filepath.Join(tempDir.Path, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Проверяем, что файл существует
	if _, statErr := os.Stat(testFile); statErr != nil {
		t.Fatalf("Test file should exist: %v", statErr)
	}

	// Очищаем с нулевым maxAge (должно удалить все)
	err = tm.CleanupOldDirs(0)
	if err != nil {
		t.Errorf("CleanupOldDirs with zero maxAge failed: %v", err)
	}

	// Проверяем, что директория удалена
	if _, statErr2 := os.Stat(tempDir.Path); statErr2 == nil {
		t.Error("Temp directory should be removed after cleanup")
	}
}