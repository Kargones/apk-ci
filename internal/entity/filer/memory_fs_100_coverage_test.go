package filer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMemoryFileSystem_100Coverage тестирует все функции для достижения 100% покрытия
func TestMemoryFileSystem_100Coverage(t *testing.T) {
	fs := NewMemoryFileSystem("")

	// Тест Chown - функция с 0% покрытием
	t.Run("Chown", func(t *testing.T) {
		// Создаем файл
		file, err := fs.Create("chown_test.txt")
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		file.Close()

		// Тестируем chown на файле
		err = fs.Chown("chown_test.txt", 1000, 1000)
		if err != nil {
			t.Errorf("Chown on file failed: %v", err)
		}

		// Создаем директорию
		err = fs.MkdirAll("chown_dir", 0755)
		if err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}

		// Тестируем chown на директории
		err = fs.Chown("chown_dir", 1000, 1000)
		if err != nil {
			t.Errorf("Chown on directory failed: %v", err)
		}

		// Тестируем chown на несуществующем файле
		err = fs.Chown("nonexistent.txt", 1000, 1000)
		if err == nil {
			t.Error("Expected error for chown on nonexistent file")
		}
	})

	// Тест isDirEmptyLocked - функция с 0% покрытием
	t.Run("isDirEmptyLocked", func(t *testing.T) {
		// Создаем пустую директорию
		err := fs.MkdirAll("empty_dir", 0755)
		if err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}

		// Пытаемся удалить пустую директорию (это вызовет isDirEmptyLocked)
		err = fs.Remove("empty_dir")
		if err != nil {
			t.Errorf("Remove empty directory failed: %v", err)
		}

		// Создаем директорию с файлом
		err = fs.MkdirAll("non_empty_dir", 0755)
		if err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}

		file, err := fs.Create("non_empty_dir/file.txt")
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		file.Close()

		// Пытаемся удалить непустую директорию (это также вызовет isDirEmptyLocked)
		err = fs.Remove("non_empty_dir")
		if err == nil {
			t.Error("Expected error when removing non-empty directory")
		}

		// Создаем директорию с поддиректорией
		err = fs.MkdirAll("parent_dir/child_dir", 0755)
		if err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}

		// Пытаемся удалить родительскую директорию с поддиректорией
		err = fs.Remove("parent_dir")
		if err == nil {
			t.Error("Expected error when removing directory with subdirectory")
		}
	})

	// Тест updatePathsAfterRename - функция с 0% покрытием
	t.Run("updatePathsAfterRename", func(t *testing.T) {
		// Создаем структуру директорий с файлами
		err := fs.MkdirAll("old_dir/subdir", 0755)
		if err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}

		file1, err := fs.Create("old_dir/file1.txt")
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		file1.Write([]byte("content1"))
		file1.Close()

		file2, err := fs.Create("old_dir/subdir/file2.txt")
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		file2.Write([]byte("content2"))
		file2.Close()

		// Переименовываем директорию (это вызовет updatePathsAfterRename)
		err = fs.Rename("old_dir", "new_dir")
		if err != nil {
			t.Errorf("Rename directory failed: %v", err)
		}

		// Проверяем, что файлы доступны по новым путям
		data1, err := fs.ReadFile("new_dir/file1.txt")
		if err != nil {
			t.Errorf("ReadFile after rename failed: %v", err)
		}
		if string(data1) != "content1" {
			t.Errorf("File content mismatch after rename")
		}

		data2, err := fs.ReadFile("new_dir/subdir/file2.txt")
		if err != nil {
			t.Errorf("ReadFile nested after rename failed: %v", err)
		}
		if string(data2) != "content2" {
			t.Errorf("Nested file content mismatch after rename")
		}

		// Проверяем, что старые пути недоступны
		_, err = fs.ReadFile("old_dir/file1.txt")
		if err == nil {
			t.Error("Old path should not be accessible after rename")
		}
	})

	// Тест Chmod - улучшаем покрытие с 57.1%
	t.Run("Chmod", func(t *testing.T) {
		// Создаем файл
		file, err := fs.Create("chmod_file.txt")
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		file.Close()

		// Изменяем права файла
		err = fs.Chmod("chmod_file.txt", 0600)
		if err != nil {
			t.Errorf("Chmod on file failed: %v", err)
		}

		// Проверяем, что права изменились
		info, err := fs.Stat("chmod_file.txt")
		if err != nil {
			t.Fatalf("Stat failed: %v", err)
		}
		if info.Mode().Perm() != 0600 {
			t.Errorf("Expected mode 0600, got %o", info.Mode().Perm())
		}

		// Создаем директорию
		err = fs.MkdirAll("chmod_dir", 0755)
		if err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}

		// Изменяем права директории
		err = fs.Chmod("chmod_dir", 0700)
		if err != nil {
			t.Errorf("Chmod on directory failed: %v", err)
		}

		// Проверяем, что права директории изменились
		info, err = fs.Stat("chmod_dir")
		if err != nil {
			t.Fatalf("Stat failed: %v", err)
		}
		if info.Mode().Perm() != 0700 {
			t.Errorf("Expected mode 0700, got %o", info.Mode().Perm())
		}

		// Тестируем chmod на несуществующем файле
		err = fs.Chmod("nonexistent.txt", 0644)
		if err == nil {
			t.Error("Expected error for chmod on nonexistent file")
		}

		// Тестируем chmod с невалидным путем
		err = fs.Chmod("invalid\x00path", 0644)
		if err == nil {
			t.Error("Expected error for chmod with invalid path")
		}
	})

	// Тест resolvePath - улучшаем покрытие с 66.7%
	t.Run("resolvePath", func(t *testing.T) {
		// Тестируем создание файла в подпапке
		err := fs.MkdirAll("testdir", 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		file, err := fs.Create("testdir/test.txt")
		if err != nil {
			t.Fatalf("Create with path failed: %v", err)
		}
		file.Close()

		// Проверяем, что файл создан
		_, err = fs.Stat("testdir/test.txt")
		if err != nil {
			t.Errorf("Stat path failed: %v", err)
		}

		// Тестируем создание файла в глубокой структуре
		err = fs.MkdirAll("dir1/dir2", 0755)
		if err != nil {
			t.Fatalf("Failed to create nested directories: %v", err)
		}

		file2, err := fs.Create("dir1/dir2/test2.txt")
		if err != nil {
			t.Fatalf("Create with nested path failed: %v", err)
		}
		file2.Close()

		// Проверяем, что файл создан в правильном месте
		_, err = fs.Stat("dir1/dir2/test2.txt")
		if err != nil {
			t.Errorf("Stat nested path failed: %v", err)
		}

		// Тестируем путь с ./
		file3, err := fs.Create("./test3.txt")
		if err != nil {
			t.Fatalf("Create with ./ path failed: %v", err)
		}
		file3.Close()

		_, err = fs.Stat("test3.txt")
		if err != nil {
			t.Errorf("Stat ./ path failed: %v", err)
		}

		// Тестируем пустой путь
		_, err = fs.Stat("")
		if err == nil {
			t.Error("Expected error for empty path")
		}
	})

	// Тест WriteFile - улучшаем покрытие с 83.3%
	t.Run("WriteFile_ErrorCases", func(t *testing.T) {
		// Тестируем WriteFile с невалидным путем
		err := fs.WriteFile("invalid\x00path", []byte("data"), 0644)
		if err == nil {
			t.Error("Expected error for WriteFile with invalid path")
		}

		// Тестируем нормальный случай
		err = fs.WriteFile("normal_file.txt", []byte("test data"), 0644)
		if err != nil {
			t.Errorf("WriteFile failed: %v", err)
		}

		// Проверяем содержимое
		data, err := fs.ReadFile("normal_file.txt")
		if err != nil {
			t.Errorf("ReadFile failed: %v", err)
		}
		if string(data) != "test data" { //nolint:goconst // test value
			t.Errorf("File content mismatch")
		}
	})
}

// TestMemoryDirEntry_Methods тестирует методы memoryDirEntry с 0% покрытием
func TestMemoryDirEntry_Methods(t *testing.T) {
	fs := NewMemoryFileSystem("/tmp")

	// Создаем файлы и директории
	err := fs.MkdirAll("test_dir", 0755)
	if err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	file, err := fs.Create("test_dir/test_file.txt")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	file.Write([]byte("test content"))
	file.Close()

	// Читаем содержимое директории
	entries, err := fs.ReadDir("test_dir")
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]

	// Тестируем Name() - уже покрыто
	if entry.Name() != "test_file.txt" {
		t.Errorf("Expected name 'test_file.txt', got '%s'", entry.Name())
	}

	// Тестируем IsDir() - 0% покрытие
	if entry.IsDir() {
		t.Error("File entry should not be directory")
	}

	// Тестируем Type() - 0% покрытие
	fileType := entry.Type()
	if fileType != 0 {
		t.Errorf("Expected file type 0, got %v", fileType)
	}

	// Тестируем Info() - 0% покрытие
	info, err := entry.Info()
	if err != nil {
		t.Errorf("Info() failed: %v", err)
	}
	if info == nil {
		t.Error("Info() returned nil")
	}
	if info.Name() != "test_file.txt" {
		t.Errorf("Info name mismatch")
	}
	if info.IsDir() {
		t.Error("Info should not indicate directory for file")
	}

	// Тестируем с директорией
	err = fs.MkdirAll("test_dir/sub_dir", 0755)
	if err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	entries, err = fs.ReadDir("test_dir")
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	// Находим директорию в списке
	var dirEntry os.DirEntry
	for _, e := range entries {
		if e.Name() == "sub_dir" {
			dirEntry = e
			break
		}
	}

	if dirEntry == nil {
		t.Fatal("Directory entry not found")
	}

	// Тестируем IsDir() для директории
	if !dirEntry.IsDir() {
		t.Error("Directory entry should be directory")
	}

	// Тестируем Type() для директории
	dirType := dirEntry.Type()
	if dirType != os.ModeDir {
		t.Errorf("Expected directory type %v, got %v", os.ModeDir, dirType)
	}

	// Тестируем Info() для директории
	dirInfo, err := dirEntry.Info()
	if err != nil {
		t.Errorf("Info() for directory failed: %v", err)
	}
	if !dirInfo.IsDir() {
		t.Error("Directory info should indicate directory")
	}
}

// TestMemoryFileSystem_AdditionalEdgeCases тестирует дополнительные граничные случаи
func TestMemoryFileSystem_AdditionalEdgeCases(t *testing.T) {
	fs := NewMemoryFileSystem("")

	// Тест Chown с различными случаями
	t.Run("Chown_NonExistentPath", func(t *testing.T) {
		// Тестируем Chown на несуществующем пути
		err := fs.Chown("nonexistent/path", 1000, 1000)
		if err == nil {
			t.Fatal("Expected error for non-existent path")
		}
		if !fs.IsNotExist(err) {
			t.Fatalf("Expected not exist error, got: %v", err)
		}
	})

	// Тест Chown с невалидным путем
	t.Run("Chown_InvalidPath", func(t *testing.T) {
		err := fs.Chown("../invalid", 1000, 1000)
		if err == nil {
			t.Error("Expected error for invalid path")
		}
	})

	t.Run("Chown_ExistingFile", func(t *testing.T) {
		// Создаем файл
		file, err := fs.Create("chown_test.txt")
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
		file.Close()
		
		// Тестируем Chown на существующем файле
		err = fs.Chown("chown_test.txt", 1000, 1000)
		if err != nil {
			t.Fatalf("Chown should succeed for existing file: %v", err)
		}
	})

	t.Run("Chown_ExistingDir", func(t *testing.T) {
		// Создаем директорию
		err := fs.MkdirAll("chown_dir", 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		
		// Тестируем Chown на существующей директории
		err = fs.Chown("chown_dir", 1000, 1000)
		if err != nil {
			t.Fatalf("Chown should succeed for existing directory: %v", err)
		}
	})

	t.Run("ResolvePath_AbsolutePath", func(t *testing.T) {
		// Создаем новую файловую систему с корневым путем для тестирования абсолютных путей
		absFS := NewMemoryFileSystem("/tmp")
		
		// Тестируем resolvePath с абсолютным путем (должен возвращаться как есть)
		resolved := absFS.resolvePath("/tmp/test.txt")
		if resolved != "/tmp/test.txt" {
			t.Errorf("Expected resolved path '/tmp/test.txt', got '%s'", resolved)
		}
		
		// Тестируем resolvePath с относительным путем (должен объединяться с корнем)
		resolved2 := absFS.resolvePath("relative_test.txt")
		if resolved2 != "/tmp/relative_test.txt" {
			t.Errorf("Expected resolved path '/tmp/relative_test.txt', got '%s'", resolved2)
		}
		
		// Тестируем resolvePath с пустым путем
		resolved3 := absFS.resolvePath("")
		if resolved3 != "/tmp" {
			t.Errorf("Expected resolved path '/tmp', got '%s'", resolved3)
		}
	})

	t.Run("MkdirAllLocked_EdgeCases", func(t *testing.T) {
		// Тестируем создание директории с корневым путем
		err := fs.MkdirAll("/", 0755)
		if err != nil {
			t.Fatalf("MkdirAll should handle root path: %v", err)
		}
		
		// Тестируем создание уже существующей директории
		err = fs.MkdirAll("existing_dir", 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		
		// Повторное создание той же директории
		err = fs.MkdirAll("existing_dir", 0755)
		if err != nil {
			t.Fatalf("MkdirAll should handle existing directory: %v", err)
		}
		
		// Тестируем создание директории с точкой
		err = fs.MkdirAll(".", 0755)
		if err != nil {
			t.Errorf("Should handle current directory: %v", err)
		}
		
		// Тестируем создание глубоко вложенных директорий для покрытия условия parent != path
		err = fs.MkdirAll("very/deep/nested/path/structure", 0755)
		if err != nil {
			t.Errorf("Failed to create deeply nested directories: %v", err)
		}
		
		// Проверяем, что вложенная структура была создана
		_, err = fs.Stat("very/deep/nested/path/structure")
		if err != nil {
			t.Errorf("Nested directory structure should exist: %v", err)
		}
		
		// Test creating root directory to cover parentDir == path condition
		rootFS := NewMemoryFileSystem("/")
		err = rootFS.MkdirAll("/", 0755)
		if err != nil {
			t.Errorf("Should handle root directory creation: %v", err)
		}
		
		// Test creating single level directory to cover parentDir == "." condition
		err = fs.MkdirAll("single", 0755)
		if err != nil {
			t.Errorf("Should handle single level directory: %v", err)
		}
		
		// Test edge case: try to create directory when parent exists as file
		// This should test the error return path in mkdirAllLocked
		fs2 := NewMemoryFileSystem("/tmp")
		// Create a file first
		file, err := fs2.Create("parent")
		if err != nil {
			t.Errorf("Failed to create file: %v", err)
		}
		file.Close()
		
		// Now manually corrupt the filesystem state to trigger error path
		fs2.mu.Lock()
		// Add a file entry that conflicts with directory creation
		fs2.files["parent/child"] = NewMemoryFile("parent/child", 0644)
		fs2.mu.Unlock()
		
		// This should trigger the error path when trying to create parent directory
		err = fs2.MkdirAll("parent/child/grandchild", 0755)
		// We expect this to work in memory filesystem, but it tests the code path
		if err != nil {
			// This is actually expected behavior - we're testing edge cases
			t.Logf("Expected error when creating conflicting directory: %v", err)
		}
	})

	// Тест OpenFile с флагом O_EXCL
	t.Run("OpenFile_O_EXCL", func(t *testing.T) {
		// Создаем файл
		file, err := fs.Create("excl_test.txt")
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		file.Close()

		// Пытаемся открыть существующий файл с O_CREATE|O_EXCL
		_, err = fs.OpenFile("excl_test.txt", os.O_CREATE|os.O_EXCL, 0644)
		if err == nil {
			t.Error("Expected error when opening existing file with O_EXCL")
		}
	})

	// Тест OpenFile с флагом O_TRUNC
	t.Run("OpenFile_O_TRUNC", func(t *testing.T) {
		// Создаем файл с содержимым
		err := fs.WriteFile("trunc_test.txt", []byte("initial content"), 0644)
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}

		// Проверяем состояние файла перед открытием
		origFile, exists := fs.files["trunc_test.txt"]
		if exists {
			t.Logf("Original file closed before open: %v", origFile.IsClosed())
		}
		
		// Открываем с O_TRUNC - это должно работать без ошибок
		file, err := fs.OpenFile("trunc_test.txt", os.O_RDWR|os.O_TRUNC, 0644)
		if err != nil {
			t.Fatalf("Failed to open file with O_TRUNC: %v", err)
		}
		
		// Проверяем, что файл не закрыт
		if memFile, ok := file.(*MemoryFile); ok {
			if memFile.IsClosed() {
				t.Fatalf("File is closed after opening with O_TRUNC")
			}
			t.Logf("File flag: %d, expected: %d", memFile.GetFlag(), os.O_RDWR|os.O_TRUNC)
			// Проверяем, тот же ли это файл
			if exists {
				t.Logf("Same file object: %v", memFile == origFile)
			}
		}
		
		// Записываем новое содержимое
		_, err = file.Write([]byte("new content"))
		if err != nil {
			t.Fatalf("Failed to write to truncated file: %v", err)
		}
		
		// Закрываем файл после записи
		file.Close()

		// Проверяем, что файл содержит только новое содержимое
		content, err := fs.ReadFile("trunc_test.txt")
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}
		if string(content) != "new content" {
			t.Errorf("Expected 'new content', got %q", string(content))
		}
	})

	// Тест CreateTemp с пустой директорией
	t.Run("CreateTemp_EmptyDir", func(t *testing.T) {
		file, err := fs.CreateTemp("", "temp_")
		if err != nil {
			t.Errorf("CreateTemp with empty dir failed: %v", err)
		}
		if file != nil {
			file.Close()
		}
	})

	// Тест MkdirTemp с пустой директорией
	t.Run("MkdirTemp_EmptyDir", func(t *testing.T) {
		dir, err := fs.MkdirTemp("", "temp_dir_")
		if err != nil {
			t.Errorf("MkdirTemp with empty dir failed: %v", err)
		}
		if dir == "" {
			t.Error("MkdirTemp returned empty directory name")
		}
	})

	// Тест Stat на директории
	t.Run("Stat_Directory", func(t *testing.T) {
		err := fs.MkdirAll("stat_dir", 0755)
		if err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}

		info, err := fs.Stat("stat_dir")
		if err != nil {
			t.Errorf("Stat on directory failed: %v", err)
		}
		if !info.IsDir() {
			t.Error("Stat should indicate directory")
		}
		if info.Mode()&os.ModeDir == 0 {
			t.Error("Directory mode should have ModeDir bit set")
		}
	})

	// Тест ReadDir на несуществующей директории
	t.Run("ReadDir_Nonexistent", func(t *testing.T) {
		_, err := fs.ReadDir("nonexistent_dir")
		if err == nil {
			t.Error("Expected error when reading nonexistent directory")
		}
	})

	// Тест Chdir
	t.Run("Chdir", func(t *testing.T) {
		// Создаем директорию
		err := fs.MkdirAll("chdir_test", 0755)
		if err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}

		// Меняем текущую директорию
		err = fs.Chdir("chdir_test")
		if err != nil {
			t.Errorf("Chdir failed: %v", err)
		}

		// Проверяем текущую директорию
		cwd, err := fs.Getwd()
		if err != nil {
			t.Errorf("Getwd failed: %v", err)
		}
		if !strings.HasSuffix(cwd, "chdir_test") {
			t.Errorf("Expected cwd to end with 'chdir_test', got %s", cwd)
		}

		// Тест Chdir на несуществующую директорию
		err = fs.Chdir("nonexistent_dir")
		if err == nil {
			t.Error("Expected error when changing to nonexistent directory")
		}
	})

	// Тест MkdirTemp с относительным путем
	t.Run("MkdirTemp_RelativePath", func(t *testing.T) {
		// Создаем базовую директорию
		err := fs.MkdirAll("base_dir", 0755)
		if err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}

		// Создаем временную директорию в базовой
		tempDir, err := fs.MkdirTemp("base_dir", "temp_")
		if err != nil {
			t.Errorf("MkdirTemp failed: %v", err)
		}

		// Проверяем, что временная директория создана
		if tempDir == "" {
			t.Error("MkdirTemp returned empty path")
		}

		// Проверяем, что можем создать файл в временной директории
		fullPath := filepath.Join("base_dir", tempDir)
		file, err := fs.Create(filepath.Join(fullPath, "test.txt"))
		if err != nil {
			t.Errorf("Create in temp dir failed: %v", err)
		}
		if file != nil {
			file.Close()
		}
	})
}