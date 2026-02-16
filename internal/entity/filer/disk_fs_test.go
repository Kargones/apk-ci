package filer

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestDiskFileSystem_Basic(t *testing.T) {
	tempDir := t.TempDir()
	config := Config{
		Type:     DiskFS,
		BasePath: tempDir,
	}
	
	fs, err := NewDiskFileSystem(config)
	if err != nil {
		t.Fatalf("Failed to create DiskFileSystem: %v", err)
	}
	
	// Тест создания директории
	err = fs.MkdirAll("test/subdir", 0755)
	if err != nil {
		t.Errorf("MkdirAll failed: %v", err)
	}
	
	// Проверяем, что директория создана
	if _, err := os.Stat(filepath.Join(tempDir, "test/subdir")); os.IsNotExist(err) {
		t.Error("Directory was not created")
	}
}

func TestDiskFileSystem_FileOperations(t *testing.T) {
	tempDir := t.TempDir()
	config := Config{
		Type:     DiskFS,
		BasePath: tempDir,
	}
	
	fs, err := NewDiskFileSystem(config)
	if err != nil {
		t.Fatalf("Failed to create DiskFileSystem: %v", err)
	}
	
	// Создание файла
	file, err := fs.Create("test.txt")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer file.Close()
	
	// Запись в файл
	testData := []byte("Hello, World!")
	n, err := file.Write(testData)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Write returned %d, expected %d", n, len(testData))
	}
	
	file.Close()
	
	// Чтение файла
	data, err := fs.ReadFile("test.txt")
	if err != nil {
		t.Errorf("ReadFile failed: %v", err)
	}
	if string(data) != string(testData) {
		t.Errorf("ReadFile returned %q, expected %q", string(data), string(testData))
	}
	
	// Удаление файла
	err = fs.Remove("test.txt")
	if err != nil {
		t.Errorf("Remove failed: %v", err)
	}
	
	// Проверяем, что файл удален
	if _, err := fs.Stat("test.txt"); !fs.IsNotExist(err) {
		t.Error("File was not removed")
	}
}

func TestDiskFileSystem_InvalidPath(t *testing.T) {
	config := Config{
		Type:     DiskFS,
		BasePath: "", // Пустой путь
	}
	
	_, err := NewDiskFileSystem(config)
	if err == nil {
		t.Error("Expected error for empty base path")
	}
}

func TestDiskFileSystem_ReadDir(t *testing.T) {
	tempDir := t.TempDir()
	config := Config{
		Type:     DiskFS,
		BasePath: tempDir,
	}
	
	fs, err := NewDiskFileSystem(config)
	if err != nil {
		t.Fatalf("Failed to create DiskFileSystem: %v", err)
	}
	
	// Создаем директорию и файлы
	err = fs.MkdirAll("readdir_test", 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	
	// Создаем файлы
	files := []string{"file1.txt", "file2.txt"}
	for _, filename := range files {
		file, createErr := fs.Create("readdir_test/" + filename)
		if createErr != nil {
			t.Fatalf("Failed to create file %s: %v", filename, createErr)
		}
		file.Close()
	}
	
	// Читаем содержимое директории
	entries, err := fs.ReadDir("readdir_test")
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}
	
	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}
}

// TestDiskFileSystem_Rename тестирует переименование файлов
func TestDiskFileSystem_Rename(t *testing.T) {
	tempDir := t.TempDir()
	config := Config{
		Type:     DiskFS,
		BasePath: tempDir,
	}
	
	fs, err := NewDiskFileSystem(config)
	if err != nil {
		t.Fatalf("Failed to create DiskFileSystem: %v", err)
	}
	
	// Создаем файл с данными
	testData := []byte("Rename test data")
	err = fs.WriteFile("old_name.txt", testData, 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	
	// Переименовываем файл
	err = fs.Rename("old_name.txt", "new_name.txt")
	if err != nil {
		t.Fatalf("Failed to rename file: %v", err)
	}
	
	// Проверяем, что старый файл не существует
	_, err = fs.Stat("old_name.txt")
	if !fs.IsNotExist(err) {
		t.Error("Old file should not exist after rename")
	}
	
	// Проверяем, что новый файл существует
	readData, err := fs.ReadFile("new_name.txt")
	if err != nil {
		t.Fatalf("Failed to read renamed file: %v", err)
	}
	
	if string(testData) != string(readData) {
		t.Errorf("Expected %s, got %s", string(testData), string(readData))
	}
}

// TestDiskFileSystem_CreateTemp тестирует создание временных файлов
func TestDiskFileSystem_CreateTemp(t *testing.T) {
	tempDir := t.TempDir()
	config := Config{
		Type:     DiskFS,
		BasePath: tempDir,
	}
	
	fs, err := NewDiskFileSystem(config)
	if err != nil {
		t.Fatalf("Failed to create DiskFileSystem: %v", err)
	}
	
	// Создаем временный файл
	file, err := fs.CreateTemp(".", "temp_test_")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer file.Close()
	
	// Записываем данные
	testData := []byte("Temporary file data")
	n, err := file.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testData), n)
	}
}

// TestDiskFileSystem_MkdirTemp тестирует создание временных директорий
func TestDiskFileSystem_MkdirTemp(t *testing.T) {
	tempDir := t.TempDir()
	config := Config{
		Type:     DiskFS,
		BasePath: tempDir,
	}
	
	fs, err := NewDiskFileSystem(config)
	if err != nil {
		t.Fatalf("Failed to create DiskFileSystem: %v", err)
	}
	
	// Создаем временную директорию
	tempDirPath, err := fs.MkdirTemp(".", "temp_dir_")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	
	if tempDirPath == "" {
		t.Error("Temp directory path is empty")
	}
	
	// Проверяем, что директория существует
	info, err := fs.Stat(tempDirPath)
	if err != nil {
		t.Fatalf("Failed to stat temp directory: %v", err)
	}
	
	if !info.IsDir() {
		t.Error("Expected directory")
	}
}

// TestDiskFileSystem_Chmod тестирует изменение прав доступа
func TestDiskFileSystem_Chmod(t *testing.T) {
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
	file, err := fs.Create("chmod_test.txt")
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	file.Close()
	
	// Изменяем права доступа
	err = fs.Chmod("chmod_test.txt", 0600)
	if err != nil {
		t.Fatalf("Failed to chmod file: %v", err)
	}
	
	// Проверяем права доступа
	info, err := fs.Stat("chmod_test.txt")
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}
	
	if info.Mode().Perm() != 0600 {
		t.Errorf("Expected permissions 0600, got %o", info.Mode().Perm())
	}
}

// TestDiskFileSystem_ErrorCases тестирует обработку ошибок
func TestDiskFileSystem_ErrorCases(t *testing.T) {
	tempDir := t.TempDir()
	config := Config{
		Type:     DiskFS,
		BasePath: tempDir,
	}
	
	fs, err := NewDiskFileSystem(config)
	if err != nil {
		t.Fatalf("Failed to create DiskFileSystem: %v", err)
	}
	
	// Тест открытия несуществующего файла
	_, err = fs.Open("nonexistent.txt")
	if err == nil {
		t.Error("Expected error when opening nonexistent file")
	}
	if !fs.IsNotExist(err) {
		t.Error("Error should be IsNotExist")
	}
	
	// Тест удаления несуществующего файла
	err = fs.Remove("nonexistent.txt")
	if err == nil {
		t.Error("Expected error when removing nonexistent file")
	}
	
	// Тест чтения несуществующего файла
	_, err = fs.ReadFile("nonexistent.txt")
	if err == nil {
		t.Error("Expected error when reading nonexistent file")
	}
}

// TestDiskFileSystem_PathTraversal тестирует защиту от path traversal
func TestDiskFileSystem_PathTraversal(t *testing.T) {
	tempDir := t.TempDir()
	config := Config{
		Type:     DiskFS,
		BasePath: tempDir,
	}
	
	fs, err := NewDiskFileSystem(config)
	if err != nil {
		t.Fatalf("Failed to create DiskFileSystem: %v", err)
	}
	
	// Тестируем различные попытки path traversal
	maliciousPaths := []string{
		"../../../etc/passwd",
		"test/../../../etc/passwd",
	}
	
	for _, path := range maliciousPaths {
		t.Run("PathTraversal_"+path, func(t *testing.T) {
			_, err := fs.Create(path)
			if err == nil {
				t.Errorf("Expected error for malicious path: %s", path)
			}
			
			_, err = fs.Open(path)
			if err == nil {
				t.Errorf("Expected error for malicious path: %s", path)
			}
		})
	}
}

// TestDiskFileSystem_Stat тестирует получение информации о файле
func TestDiskFileSystem_Stat(t *testing.T) {
	tempDir := t.TempDir()
	config := Config{
		Type:     DiskFS,
		BasePath: tempDir,
	}
	
	fs, err := NewDiskFileSystem(config)
	if err != nil {
		t.Fatalf("Failed to create DiskFileSystem: %v", err)
	}
	
	// Тест с существующим файлом
	filename := "test.txt"
	content := []byte("test content")
	err = fs.WriteFile(filename, content, 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	
	info, err := fs.Stat(filename)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}
	
	if info.Name() != filename {
		t.Errorf("Expected name %s, got %s", filename, info.Name())
	}
	
	if info.Size() != int64(len(content)) {
		t.Errorf("Expected size %d, got %d", len(content), info.Size())
	}
	
	if info.IsDir() {
		t.Error("Expected file, not directory")
	}
	
	// Тест с несуществующим файлом
	_, err = fs.Stat("nonexistent.txt")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
	
	// Тест с директорией
	dirName := "testdir"
	err = fs.MkdirAll(dirName, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	
	dirInfo, err := fs.Stat(dirName)
	if err != nil {
		t.Fatalf("Failed to stat directory: %v", err)
	}
	
	if !dirInfo.IsDir() {
		t.Error("Expected directory")
	}
}



// TestDiskFileSystem_ConcurrentAccess тестирует конкурентный доступ
func TestDiskFileSystem_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	config := Config{
		Type:     DiskFS,
		BasePath: tempDir,
	}
	
	fs, err := NewDiskFileSystem(config)
	if err != nil {
		t.Fatalf("Failed to create DiskFileSystem: %v", err)
	}
	
	// Конкурентная запись в разные файлы
	var wg sync.WaitGroup
	numGoroutines := 10
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			filename := fmt.Sprintf("file_%d.txt", id)
			content := []byte(fmt.Sprintf("content_%d", id))
			
			err := fs.WriteFile(filename, content, 0644)
			if err != nil {
				t.Errorf("Failed to write file %s: %v", filename, err)
				return
			}
			
			readContent, err := fs.ReadFile(filename)
			if err != nil {
				t.Errorf("Failed to read file %s: %v", filename, err)
				return
			}
			
			if string(readContent) != string(content) {
				t.Errorf("Content mismatch for file %s: got %q, expected %q", filename, string(readContent), string(content))
			}
		}(i)
	}
	
	wg.Wait()
	
	// Проверяем, что все файлы созданы
	for i := 0; i < numGoroutines; i++ {
		filename := fmt.Sprintf("file_%d.txt", i)
		_, err := fs.Stat(filename)
		if err != nil {
			t.Errorf("File %s should exist after concurrent write: %v", filename, err)
		}
	}
}

// TestDiskFileSystem_OpenFile тестирует OpenFile с различными флагами
func TestDiskFileSystem_OpenFile(t *testing.T) {
	tempDir := t.TempDir()
	config := Config{
		Type:     DiskFS,
		BasePath: tempDir,
	}
	
	fs, err := NewDiskFileSystem(config)
	if err != nil {
		t.Fatalf("Failed to create DiskFileSystem: %v", err)
	}
	
	// Тест создания файла с OpenFile
	file, err := fs.OpenFile("test.txt", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer file.Close()
	
	// Запись в файл
	testData := []byte("test data")
	n, err := file.Write(testData)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Write returned %d, expected %d", n, len(testData))
	}
	file.Close()
	
	// Тест открытия для чтения
	readFile, err := fs.OpenFile("test.txt", os.O_RDONLY, 0)
	if err != nil {
		t.Fatalf("OpenFile for reading failed: %v", err)
	}
	defer readFile.Close()
	
	// Чтение данных
	buf := make([]byte, len(testData))
	n, err = readFile.Read(buf)
	if err != nil {
		t.Errorf("Read failed: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Read returned %d, expected %d", n, len(testData))
	}
	if string(buf) != string(testData) {
		t.Errorf("Read data %q, expected %q", string(buf), string(testData))
	}
}

// TestDiskFileSystem_GetCwd тестирует Getwd
func TestDiskFileSystem_GetCwd(t *testing.T) {
	tempDir := t.TempDir()
	config := Config{
		Type:     DiskFS,
		BasePath: tempDir,
	}
	
	fs, err := NewDiskFileSystem(config)
	if err != nil {
		t.Fatalf("Failed to create DiskFileSystem: %v", err)
	}
	
	cwd, err := fs.Getwd()
	if err != nil {
		t.Errorf("Getwd failed: %v", err)
	}
	if cwd != "/" {
		t.Errorf("Expected cwd to be '/', got %q", cwd)
	}
}

// TestDiskFileSystem_Chdir тестирует Chdir
func TestDiskFileSystem_Chdir(t *testing.T) {
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
	
	// Тест Chdir
	err = fs.Chdir("testdir")
	if err != nil {
		t.Errorf("Chdir failed: %v", err)
	}
}

// TestDiskFileSystem_Chown тестирует Chown
func TestDiskFileSystem_Chown(t *testing.T) {
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
	
	// Тест Chown (может не работать в некоторых средах, но покроет код)
	err = fs.Chown("test.txt", os.Getuid(), os.Getgid())
	if err != nil {
		// В некоторых средах Chown может не работать, это нормально
		t.Logf("Chown failed (may be expected in some environments): %v", err)
	}
}

// TestDiskFileSystem_AdditionalCoverage добавляет покрытие для граничных случаев
func TestDiskFileSystem_AdditionalCoverage(t *testing.T) {
	tempDir := t.TempDir()
	config := Config{
		Type:     DiskFS,
		BasePath: tempDir,
	}
	
	fs, err := NewDiskFileSystem(config)
	if err != nil {
		t.Fatalf("Failed to create DiskFileSystem: %v", err)
	}
	
	// Тест CreateTemp с ошибкой (несуществующая директория)
	_, err = fs.CreateTemp("../../../nonexistent", "temp")
	if err == nil {
		t.Error("Expected error for CreateTemp in invalid directory")
	}
	
	// Тест Create с ошибкой пути
	_, err = fs.Create("../../../invalid")
	if err == nil {
		t.Error("Expected error for Create with invalid path")
	}
	
	// Тест ReadDir с ошибкой
	_, err = fs.ReadDir("../../../nonexistent")
	if err == nil {
		t.Error("Expected error for ReadDir with invalid path")
	}
	
	// Тест Mkdir
	err = fs.Mkdir("newdir", 0755)
	if err != nil {
		t.Errorf("Mkdir failed: %v", err)
	}
	
	// Проверяем, что директория создана
	info, err := fs.Stat("newdir")
	if err != nil {
		t.Errorf("Stat failed: %v", err)
	}
	if !info.IsDir() {
		t.Error("Expected directory")
	}
	
	// Тест IsNotExist
	err = os.ErrNotExist
	if !fs.IsNotExist(err) {
		t.Error("IsNotExist should return true for os.ErrNotExist")
	}
	
	genericErr := fmt.Errorf("generic error")
	if fs.IsNotExist(genericErr) {
		t.Error("IsNotExist should return false for generic error")
	}
	
	// Тест RemoveAll
	err = fs.MkdirAll("removeall_test/subdir", 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	
	// Создаем файл в директории
	file, err := fs.Create("removeall_test/file.txt")
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	file.Close()
	
	// Удаляем всю директорию
	err = fs.RemoveAll("removeall_test")
	if err != nil {
		t.Errorf("RemoveAll failed: %v", err)
	}
	
	// Проверяем, что директория удалена
	_, err = fs.Stat("removeall_test")
	if !fs.IsNotExist(err) {
		t.Error("Directory should be removed")
	}
}

func TestDiskFileSystem_ErrorHandling(t *testing.T) {
	tempDir := t.TempDir()
	fs := &DiskFileSystem{basePath: tempDir}
	
	// Тест MkdirTemp с недопустимым путем
	_, err := fs.MkdirTemp("../invalid/nonexistent/path", "test")
	if err == nil {
		t.Error("MkdirTemp should fail with invalid path")
	}
	
	// Тест RemoveAll с несуществующим путем
	err = fs.RemoveAll("nonexistent")
	if err != nil {
		t.Errorf("RemoveAll should not fail with nonexistent path: %v", err)
	}
	
	// Тест Create с недопустимым путем
	_, err = fs.Create("../invalid/nonexistent/file.txt")
	if err == nil {
		t.Error("Create should fail with invalid path")
	}
	
	// Тест OpenFile с недопустимыми флагами
	testFile := "test.txt"
	_, err = fs.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	_, err = fs.OpenFile("../invalid/nonexistent/file.txt", os.O_RDONLY, 0644)
	if err == nil {
		t.Error("OpenFile should fail with invalid path")
	}
	
	// Тест Remove с несуществующим файлом
	err = fs.Remove("nonexistent.txt")
	if err == nil {
		t.Error("Remove should fail with nonexistent file")
	}
	
	// Тест Rename с недопустимыми путями
	err = fs.Rename("../invalid/source", "../invalid/dest")
	if err == nil {
		t.Error("Rename should fail with invalid paths")
	}
	
	// Тест ReadFile с несуществующим файлом
	_, err = fs.ReadFile("nonexistent.txt")
	if err == nil {
		t.Error("ReadFile should fail with nonexistent file")
	}
	
	// Тест WriteFile с недопустимым путем
	err = fs.WriteFile("../invalid/nonexistent/file.txt", []byte("test"), 0644)
	if err == nil {
		t.Error("WriteFile should fail with invalid path")
	}
	
	// Тест Stat с несуществующим файлом
	_, err = fs.Stat("nonexistent.txt")
	if err == nil {
		t.Error("Stat should fail with nonexistent file")
	}
	
	// Тест Chmod с несуществующим файлом
	err = fs.Chmod("nonexistent.txt", 0644)
	if err == nil {
		t.Error("Chmod should fail with nonexistent file")
	}
}

func TestDiskFileSystem_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	fs := &DiskFileSystem{basePath: tempDir}
	
	// Тест Rename с одинаковыми путями
	testFile := "test.txt"
	_, err := fs.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	err = fs.Rename(testFile, testFile)
	if err != nil {
		t.Errorf("Rename with same paths should succeed: %v", err)
	}
	
	// Тест WriteFile с пустыми данными
	emptyFile := "empty.txt"
	err = fs.WriteFile(emptyFile, []byte{}, 0644)
	if err != nil {
		t.Errorf("WriteFile with empty data should succeed: %v", err)
	}
	
	// Проверим, что файл создан и пуст
	data, err := fs.ReadFile(emptyFile)
	if err != nil {
		t.Errorf("ReadFile should succeed: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("File should be empty, got %d bytes", len(data))
	}
	
	// Тест RemoveAll с вложенными директориями
	nestedDir := "nested/deep/structure"
	err = fs.MkdirAll(nestedDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}
	
	// Создаем файл в глубокой структуре
	deepFile := "nested/deep/structure/file.txt"
	err = fs.WriteFile(deepFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create deep file: %v", err)
	}
	
	// Удаляем всю структуру
	rootNested := "nested"
	err = fs.RemoveAll(rootNested)
	if err != nil {
		t.Errorf("RemoveAll failed: %v", err)
	}
	
	// Проверяем, что структура удалена
	_, err = fs.Stat(rootNested)
	if !fs.IsNotExist(err) {
		t.Error("Nested directory should be removed")
	}
}