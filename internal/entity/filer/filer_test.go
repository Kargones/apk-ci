package filer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestFactory тестирует фабрику файловых систем
func TestFactory(t *testing.T) {
	// Тест создания фабрики
	factory, err := NewFactory()
	if err != nil {
		t.Fatalf("Ошибка создания фабрики: %v", err)
	}

	if factory == nil {
		t.Fatal("Фабрика не должна быть nil")
	}

	// Тест создания дисковой файловой системы
	config := Config{
		Type:     DiskFS,
		BasePath: "/tmp/test",
	}

	fs, err := factory.CreateFileSystem(DiskFS, config)
	if err != nil {
		t.Fatalf("Ошибка создания дисковой файловой системы: %v", err)
	}

	if fs == nil {
		t.Fatal("Файловая система не должна быть nil")
	}

	// Тест создания файловой системы в памяти
	memConfig := Config{
		Type:   MemoryFS,
		UseRAM: false,
	}

	memFS, err := factory.CreateFileSystem(MemoryFS, memConfig)
	if err != nil {
		t.Fatalf("Ошибка создания файловой системы в памяти: %v", err)
	}

	if memFS == nil {
		t.Fatal("Файловая система в памяти не должна быть nil")
	}
}

// TestMemoryFileSystem тестирует файловую систему в памяти
func TestMemoryFileSystem(t *testing.T) {
	fs := NewMemoryFileSystem("test")

	// Тест создания файла
	t.Run("CreateFile", func(t *testing.T) {
		file, err := fs.Create("test.txt")
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
		defer file.Close()

		if file == nil {
			t.Fatal("File is nil")
		}

		// Записываем данные
		data := []byte("Hello, World!")
		n, err := file.Write(data)
		if err != nil {
			t.Fatalf("Failed to write to file: %v", err)
		}
		if n != len(data) {
			t.Fatalf("Expected to write %d bytes, wrote %d", len(data), n)
		}
	})

	// Тест открытия файла
	t.Run("OpenFile", func(t *testing.T) {
		// Сначала создаем файл
		file, err := fs.Create("open_test.txt")
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
		data := []byte("Test data")
		file.Write(data)
		file.Close()

		// Открываем файл для чтения
		file, err = fs.Open("open_test.txt")
		if err != nil {
			t.Fatalf("Failed to open file: %v", err)
		}
		defer file.Close()

		// Читаем данные
		readData, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if !bytes.Equal(data, readData) {
			t.Fatalf("Expected %s, got %s", string(data), string(readData))
		}
	})

	// Тест создания директории
	t.Run("MkdirAll", func(t *testing.T) {
		err := fs.MkdirAll("dir1/dir2/dir3", 0755)
		if err != nil {
			t.Fatalf("Failed to create directories: %v", err)
		}

		// Проверяем, что директория существует
		info, err := fs.Stat("dir1/dir2/dir3")
		if err != nil {
			t.Fatalf("Failed to stat directory: %v", err)
		}

		if !info.IsDir() {
			t.Fatal("Expected directory")
		}
	})

	// Тест чтения директории
	t.Run("ReadDir", func(t *testing.T) {
		// Создаем директорию и файлы
		fs.MkdirAll("readdir_test", 0755)
		fs.Create("readdir_test/file1.txt")
		fs.Create("readdir_test/file2.txt")
		fs.MkdirAll("readdir_test/subdir", 0755)

		entries, err := fs.ReadDir("readdir_test")
		if err != nil {
			t.Fatalf("Failed to read directory: %v", err)
		}

		if len(entries) != 3 {
			t.Fatalf("Expected 3 entries, got %d", len(entries))
		}

		// Проверяем имена
		names := make([]string, len(entries))
		for i, entry := range entries {
			names[i] = entry.Name()
		}

		expected := []string{"file1.txt", "file2.txt", "subdir"}
		for _, name := range expected {
			found := false
			for _, actualName := range names {
				if name == actualName {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("Expected to find %s in directory", name)
			}
		}
	})

	// Тест удаления файла
	t.Run("RemoveFile", func(t *testing.T) {
		// Создаем файл
		file, err := fs.Create("remove_test.txt")
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
		file.Close()

		// Удаляем файл
		err = fs.Remove("remove_test.txt")
		if err != nil {
			t.Fatalf("Failed to remove file: %v", err)
		}

		// Проверяем, что файл не существует
		_, err = fs.Stat("remove_test.txt")
		if !fs.IsNotExist(err) {
			t.Fatal("File should not exist after removal")
		}
	})

	// Тест переименования файла
	t.Run("RenameFile", func(t *testing.T) {
		// Создаем файл
		file, err := fs.Create("rename_old.txt")
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
		data := []byte("rename test")
		file.Write(data)
		file.Close()

		// Переименовываем файл
		err = fs.Rename("rename_old.txt", "rename_new.txt")
		if err != nil {
			t.Fatalf("Failed to rename file: %v", err)
		}

		// Проверяем, что старый файл не существует
		_, err = fs.Stat("rename_old.txt")
		if !fs.IsNotExist(err) {
			t.Fatal("Old file should not exist after rename")
		}

		// Проверяем, что новый файл существует и содержит правильные данные
		file, err = fs.Open("rename_new.txt")
		if err != nil {
			t.Fatalf("Failed to open renamed file: %v", err)
		}
		defer file.Close()

		readData, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("Failed to read renamed file: %v", err)
		}

		if !bytes.Equal(data, readData) {
			t.Fatalf("Expected %s, got %s", string(data), string(readData))
		}
	})

	// Тест WriteFile и ReadFile
	t.Run("WriteReadFile", func(t *testing.T) {
		data := []byte("WriteFile test data")
		err := fs.WriteFile("writefile_test.txt", data, 0644)
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}

		readData, err := fs.ReadFile("writefile_test.txt")
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if !bytes.Equal(data, readData) {
			t.Fatalf("Expected %s, got %s", string(data), string(readData))
		}
	})

	// Тест временных файлов
	t.Run("CreateTemp", func(t *testing.T) {
		file, err := fs.CreateTemp("", "temp_test_")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer file.Close()

		if file == nil {
			t.Fatal("Temp file is nil")
		}

		// Записываем данные
		data := []byte("temp file data")
		n, err := file.Write(data)
		if err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
		if n != len(data) {
			t.Fatalf("Expected to write %d bytes, wrote %d", len(data), n)
		}
	})

	// Тест временных директорий
	t.Run("MkdirTemp", func(t *testing.T) {
		tempDir, err := fs.MkdirTemp("", "temp_dir_")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}

		if tempDir == "" {
			t.Fatal("Temp directory path is empty")
		}

		// Проверяем, что директория существует
		info, err := fs.Stat(tempDir)
		if err != nil {
			t.Fatalf("Failed to stat temp directory: %v", err)
		}

		if !info.IsDir() {
			t.Fatal("Expected directory")
		}
	})
}

// TestMemoryFile тестирует файл в памяти
func TestMemoryFile(t *testing.T) {
	// Тест записи и чтения
	t.Run("WriteRead", func(t *testing.T) {
		file := NewMemoryFile("test.txt", 0644)
		data := []byte("Hello, Memory File!")
		n, err := file.Write(data)
		if err != nil {
			t.Fatalf("Failed to write: %v", err)
		}
		if n != len(data) {
			t.Fatalf("Expected to write %d bytes, wrote %d", len(data), n)
		}

		// Сбрасываем позицию
		file.Seek(0, io.SeekStart)

		readData := make([]byte, len(data))
		n, err = file.Read(readData)
		if err != nil {
			t.Fatalf("Failed to read: %v", err)
		}
		if n != len(data) {
			t.Fatalf("Expected to read %d bytes, read %d", len(data), n)
		}

		if !bytes.Equal(data, readData) {
			t.Fatalf("Expected %s, got %s", string(data), string(readData))
		}
	})

	// Тест Seek
	t.Run("Seek", func(t *testing.T) {
		file := NewMemoryFile("test.txt", 0644)
		data := []byte("0123456789")
		file.Write(data)

		// Seek к позиции 5
		pos, err := file.Seek(5, io.SeekStart)
		if err != nil {
			t.Fatalf("Failed to seek: %v", err)
		}
		if pos != 5 {
			t.Fatalf("Expected position 5, got %d", pos)
		}

		// Читаем один байт
		buf := make([]byte, 1)
		n, err := file.Read(buf)
		if err != nil {
			t.Fatalf("Failed to read: %v", err)
		}
		if n != 1 {
			t.Fatalf("Expected to read 1 byte, read %d", n)
		}
		if buf[0] != '5' {
			t.Fatalf("Expected '5', got '%c'", buf[0])
		}
	})

	// Тест Truncate
	t.Run("Truncate", func(t *testing.T) {
		file := NewMemoryFile("test.txt", 0644)
		data := []byte("This is a long string")
		file.Write(data)

		// Обрезаем до 4 байт
		err := file.Truncate(4)
		if err != nil {
			t.Fatalf("Failed to truncate: %v", err)
		}

		// Проверяем размер
		info, err := file.Stat()
		if err != nil {
			t.Fatalf("Failed to stat: %v", err)
		}
		if info.Size() != 4 {
			t.Fatalf("Expected size 4, got %d", info.Size())
		}

		// Читаем данные
		file.Seek(0, io.SeekStart)
		readData, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("Failed to read: %v", err)
		}
		expected := "This"
		if string(readData) != expected {
			t.Fatalf("Expected %s, got %s", expected, string(readData))
		}
	})

	// Тест Chmod
	t.Run("Chmod", func(t *testing.T) {
		file := NewMemoryFile("test.txt", 0644)
		err := file.Chmod(0755)
		if err != nil {
			t.Fatalf("Failed to chmod: %v", err)
		}

		info, err := file.Stat()
		if err != nil {
			t.Fatalf("Failed to stat: %v", err)
		}

		if info.Mode().Perm() != 0755 {
			t.Fatalf("Expected mode 0755, got %o", info.Mode().Perm())
		}
	})
}

// TestPathValidation тестирует валидацию путей
func TestPathValidation(t *testing.T) {
	pu := NewPathUtils()
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"Valid relative path", "test/file.txt", true},
		{"Path traversal", "../../../etc/passwd", false},
		{"Null byte", "test\x00.txt", false},
		{"Empty path", "", false},
		{"Absolute path", "/tmp/test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pu.ValidatePath(tt.path)
			if tt.expected && err != nil {
				t.Errorf("Ожидался валидный путь, но получена ошибка: %v", err)
			}
			if !tt.expected && err == nil {
				t.Errorf("Ожидалась ошибка для пути: %s", tt.path)
			}
		})
	}
}

// TestTempManager тестирует менеджер временных директорий
func TestTempManager(t *testing.T) {
	// Тест создания менеджера временных директорий
	tm := NewTempManager(false)
	if tm == nil {
		t.Fatal("TempManager не должен быть nil")
	}

	// Тест создания временной директории
	tempDir, err := tm.CreateTempDir("test", true)
	if err != nil {
		t.Fatalf("Ошибка создания временной директории: %v", err)
	}

	if tempDir == nil {
		t.Fatal("Временная директория не должна быть nil")
	}

	// Проверка существования директории
	if _, statErr := os.Stat(tempDir.Path); os.IsNotExist(statErr) {
		t.Fatalf("Временная директория не существует: %s", tempDir.Path)
	}

	// Тест очистки
	err = tm.CleanupAll()
	if err != nil {
		t.Fatalf("Ошибка очистки: %v", err)
	}
}

// TestErrorHandling тестирует обработку ошибок
func TestErrorHandling(t *testing.T) {
	// Тест создания FileSystemError
	fsErr := NewFileSystemError("open", "/tmp/test", os.ErrNotExist, DiskFS, SeverityError)
	if fsErr == nil {
		t.Fatal("Ошибка не должна быть nil")
	}

	if fsErr.Op != "open" {
		t.Errorf("Ожидалась операция 'open', получено: %s", fsErr.Op)
	}

	if fsErr.Path != "/tmp/test" {
		t.Errorf("Ожидался путь '/tmp/test', получено: %s", fsErr.Path)
	}

	if fsErr.FSType != DiskFS {
		t.Errorf("Ожидался тип DiskFS, получено: %v", fsErr.FSType)
	}

	if fsErr.Severity != SeverityError {
		t.Errorf("Ожидалась серьезность SeverityError, получено: %v", fsErr.Severity)
	}

	// Тест обертывания ошибки
	wrappedErr := WrapError("read", "/tmp/file", os.ErrPermission, MemoryFS)
	if wrappedErr == nil {
		t.Fatal("Обернутая ошибка не должна быть nil")
	}
}

// TestOptions тестирует функциональные опции
func TestOptions(t *testing.T) {
	// Тест опций конфигурации
	config := DefaultConfig()

	// Применение опций
	WithBasePath("/tmp/custom")(&config)
	if config.BasePath != "/tmp/custom" {
		t.Errorf("Ожидался BasePath '/tmp/custom', получено: %s", config.BasePath)
	}

	WithDiskFS("/tmp/disk")(&config)
	if config.Type != DiskFS {
		t.Error("Type должен быть DiskFS после применения WithDiskFS")
	}

	WithRAMDisk()(&config)
	if !config.UseRAM {
		t.Error("UseRAM должно быть true после применения WithRAMDisk")
	}
}

// BenchmarkMemoryFileSystem бенчмарк для файловой системы в памяти
func BenchmarkMemoryFileSystem(b *testing.B) {
	b.Run("CreateFile", func(b *testing.B) {
		fs := NewMemoryFileSystem("bench")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			filename := filepath.Join("bench", "file", "test", fmt.Sprintf("file_%d.txt", i))
			file, err := fs.Create(filename)
			if err != nil {
				b.Fatalf("Failed to create file: %v", err)
			}
			file.Close()
		}
	})

	b.Run("WriteFile", func(b *testing.B) {
		fs := NewMemoryFileSystem("bench")
		data := bytes.Repeat([]byte("benchmark data "), 64) // 1KB
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			filename := filepath.Join("bench", "write", "test", fmt.Sprintf("file_%d.txt", i))
			err := fs.WriteFile(filename, data, 0644)
			if err != nil {
				b.Fatalf("Failed to write file: %v", err)
			}
		}
	})

	b.Run("ReadFile", func(b *testing.B) {
		fs := NewMemoryFileSystem("bench")
		// Подготавливаем файл
		data := bytes.Repeat([]byte("benchmark data "), 64) // 1KB
		filename := filepath.Join("bench", "read", "test", "file.txt")
		fs.WriteFile(filename, data, 0644)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := fs.ReadFile(filename)
			if err != nil {
				b.Fatalf("Failed to read file: %v", err)
			}
		}
	})
}

// TestConcurrency тестирует конкурентный доступ
func TestConcurrency(t *testing.T) {
	fs := NewMemoryFileSystem("concurrent")

	// Тест конкурентного создания файлов
	t.Run("ConcurrentCreate", func(t *testing.T) {
		const numGoroutines = 10
		const filesPerGoroutine = 10

		done := make(chan bool, numGoroutines)
		errors := make(chan error, numGoroutines*filesPerGoroutine)

		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				for j := 0; j < filesPerGoroutine; j++ {
					filename := fmt.Sprintf("create/goroutine%d/file%d.txt", goroutineID, j)
					file, err := fs.Create(filename)
					if err != nil {
						errors <- err
						continue
					}
					file.Write([]byte("concurrent test"))
					file.Close()
				}
				done <- true
			}(i)
		}

		// Ждем завершения всех горутин
		for i := 0; i < numGoroutines; i++ {
			select {
			case <-done:
				// OK
			case err := <-errors:
				t.Fatalf("Concurrent create failed: %v", err)
			case <-time.After(5 * time.Second):
				t.Fatal("Concurrent create timed out")
			}
		}

		// Проверяем на оставшиеся ошибки
		select {
		case err := <-errors:
			t.Fatalf("Concurrent create failed: %v", err)
		default:
			// OK, нет ошибок
		}
	})
}