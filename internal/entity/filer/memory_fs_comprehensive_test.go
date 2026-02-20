package filer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMemoryFileSystem_Create(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{
			name:     "Valid file creation",
			filename: "test.txt",
			wantErr:  false,
		},
		{
			name:     "File with path",
			filename: "subdir/test.txt",
			wantErr:  false,
		},
		{
			name:     "Invalid path with null byte",
			filename: "test\x00.txt",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewMemoryFileSystem("/tmp")
			file, err := fs.Create(tt.filename)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("Create() expected error, got nil")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if file == nil {
				t.Errorf("Create() returned nil file")
				return
			}
			
			defer file.Close()
			
			// Проверяем, что файл можно записать
			n, err := file.Write([]byte("test data"))
			if err != nil {
				t.Errorf("Write() error = %v", err)
			}
			if n != 9 {
				t.Errorf("Write() wrote %d bytes, expected 9", n)
			}
		})
	}
}

func TestMemoryFileSystem_CreateTemp(t *testing.T) {
	fs := NewMemoryFileSystem("/tmp")
	
	// Тест создания временного файла
	file, err := fs.CreateTemp("", "test_")
	if err != nil {
		t.Errorf("CreateTemp() error = %v", err)
		return
	}
	defer file.Close()
	
	// Проверяем имя файла
	name := file.Name()
	if !strings.Contains(name, "test_") {
		t.Errorf("CreateTemp() name %s doesn't contain pattern", name)
	}
	
	// Проверяем, что файл можно использовать
	n, err := file.Write([]byte("temp data"))
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}
	if n != 9 {
		t.Errorf("Write() wrote %d bytes, expected 9", n)
	}
}

func TestMemoryFileSystem_Open(t *testing.T) {
	fs := NewMemoryFileSystem("/tmp")
	
	// Создаем файл сначала
	file, err := fs.Create("test.txt")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	file.Write([]byte("test content"))
	file.Close()
	
	// Тестируем открытие существующего файла
	file, err = fs.Open("test.txt")
	if err != nil {
		t.Errorf("Open() error = %v", err)
		return
	}
	defer file.Close()
	
	// Читаем содержимое
	data := make([]byte, 20)
	n, err := file.Read(data)
	if err != nil {
		t.Errorf("Read() error = %v", err)
	}
	if string(data[:n]) != "test content" { //nolint:goconst // test value
		t.Errorf("Read() got %s, expected 'test content'", string(data[:n]))
	}
	
	// Тестируем открытие несуществующего файла
	_, err = fs.Open("nonexistent.txt")
	if err == nil {
		t.Errorf("Open() expected error for nonexistent file")
	}
}

func TestMemoryFileSystem_OpenFile(t *testing.T) {
	fs := NewMemoryFileSystem("/tmp")
	
	tests := []struct {
		name     string
		filename string
		flag     int
		perm     os.FileMode
		setup    func() error
		wantErr  bool
	}{
		{
			name:     "Create new file",
			filename: "new.txt",
			flag:     os.O_CREATE | os.O_WRONLY,
			perm:     0644,
			setup:    func() error { return nil },
			wantErr:  false,
		},
		{
			name:     "Open existing file",
			filename: "existing.txt",
			flag:     os.O_RDONLY,
			perm:     0644,
			setup: func() error {
				file, err := fs.Create("existing.txt")
				if err != nil {
					return err
				}
				file.Write([]byte("existing content"))
				return file.Close()
			},
			wantErr: false,
		},
		{
			name:     "Open nonexistent without create",
			filename: "missing.txt",
			flag:     os.O_RDONLY,
			perm:     0644,
			setup:    func() error { return nil },
			wantErr:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.setup(); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
			
			file, err := fs.OpenFile(tt.filename, tt.flag, tt.perm)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("OpenFile() expected error, got nil")
				}
				return
			}
			
			if err != nil {
				t.Errorf("OpenFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			defer file.Close()
		})
	}
}

func TestMemoryFileSystem_MkdirTemp(t *testing.T) {
	fs := NewMemoryFileSystem("/tmp")
	
	// Создаем временную директорию
	dirPath, err := fs.MkdirTemp(".", "test_dir_")
	if err != nil {
		t.Errorf("MkdirTemp() error = %v", err)
		return
	}
	
	// Проверяем, что директория создана
	if !strings.Contains(dirPath, "test_dir_") {
		t.Errorf("MkdirTemp() path %s doesn't contain pattern", dirPath)
	}
	
	// Проверяем, что можем создать файл в этой директории
	file, err := fs.Create(filepath.Join(dirPath, "test.txt"))
	if err != nil {
		t.Errorf("Create() in temp dir error = %v", err)
		return
	}
	defer file.Close()
}

func TestMemoryFileSystem_MkdirAll(t *testing.T) {
	fs := NewMemoryFileSystem("/tmp")
	
	// Создаем вложенные директории
	err := fs.MkdirAll("a/b/c/d", 0755)
	if err != nil {
		t.Errorf("MkdirAll() error = %v", err)
		return
	}
	
	// Проверяем, что можем создать файл в глубокой директории
	file, err := fs.Create("a/b/c/d/test.txt")
	if err != nil {
		t.Errorf("Create() in deep dir error = %v", err)
		return
	}
	defer file.Close()
	
	// Тест с уже существующей директорией
	err = fs.MkdirAll("a/b", 0755)
	if err != nil {
		t.Errorf("MkdirAll() on existing dir error = %v", err)
	}
}

func TestMemoryFileSystem_Remove(t *testing.T) {
	fs := NewMemoryFileSystem("/tmp")
	
	// Создаем файл
	file, err := fs.Create("test.txt")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	file.Close()
	
	// Удаляем файл
	err = fs.Remove("test.txt")
	if err != nil {
		t.Errorf("Remove() error = %v", err)
	}
	
	// Проверяем, что файл удален
	_, err = fs.Open("test.txt")
	if err == nil {
		t.Errorf("Open() expected error after removal")
	}
	
	// Тест удаления несуществующего файла
	err = fs.Remove("nonexistent.txt")
	if err == nil {
		t.Errorf("Remove() expected error for nonexistent file")
	}
}

func TestMemoryFileSystem_RemoveAll(t *testing.T) {
	fs := NewMemoryFileSystem("/tmp")
	
	// Создаем структуру директорий с файлами
	fs.MkdirAll("dir/subdir", 0755)
	file1, _ := fs.Create("dir/file1.txt")
	file1.Close()
	file2, _ := fs.Create("dir/subdir/file2.txt")
	file2.Close()
	
	// Удаляем всю структуру
	err := fs.RemoveAll("dir")
	if err != nil {
		t.Errorf("RemoveAll() error = %v", err)
	}
	
	// Проверяем, что все удалено
	_, err = fs.Open("dir/file1.txt")
	if err == nil {
		t.Errorf("Open() expected error after RemoveAll")
	}
	
	_, err = fs.Open("dir/subdir/file2.txt")
	if err == nil {
		t.Errorf("Open() expected error after RemoveAll")
	}
}

func TestMemoryFileSystem_RenameFile(t *testing.T) {
	fs := NewMemoryFileSystem("/tmp")
	
	// Создаем файл
	file, err := fs.Create("old.txt")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	file.Write([]byte("test content"))
	file.Close()
	
	// Переименовываем файл
	err = fs.Rename("old.txt", "new.txt")
	if err != nil {
		t.Errorf("Rename() error = %v", err)
	}
	
	// Проверяем, что старый файл не существует
	_, err = fs.Open("old.txt")
	if err == nil {
		t.Errorf("Open() expected error for old filename")
	}
	
	// Проверяем, что новый файл существует и содержит данные
	file, err = fs.Open("new.txt")
	if err != nil {
		t.Errorf("Open() error for new filename = %v", err)
		return
	}
	defer file.Close()
	
	data := make([]byte, 20)
	n, _ := file.Read(data)
	if string(data[:n]) != "test content" {
		t.Errorf("Read() got %s, expected 'test content'", string(data[:n]))
	}
}

func TestMemoryFileSystem_ReadDirectory(t *testing.T) {
	fs := NewMemoryFileSystem("/tmp")
	
	// Создаем файлы и директории
	fs.MkdirAll("testdir/subdir", 0755)
	file1, _ := fs.Create("testdir/file1.txt")
	file1.Close()
	file2, _ := fs.Create("testdir/file2.txt")
	file2.Close()
	
	// Читаем содержимое директории
	entries, err := fs.ReadDir("testdir")
	if err != nil {
		t.Errorf("ReadDir() error = %v", err)
		return
	}
	
	// Проверяем количество записей
	if len(entries) != 3 {
		t.Errorf("ReadDir() got %d entries, expected 3", len(entries))
	}
	
	// Проверяем имена
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
			t.Errorf("ReadDir() missing expected entry %s", expected)
		}
	}
}

func TestMemoryFileSystem_ReadWriteFile(t *testing.T) {
	fs := NewMemoryFileSystem("/tmp")
	
	testData := []byte("test file content")
	
	// Записываем файл
	err := fs.WriteFile("test.txt", testData, 0644)
	if err != nil {
		t.Errorf("WriteFile() error = %v", err)
		return
	}
	
	// Читаем файл
	data, err := fs.ReadFile("test.txt")
	if err != nil {
		t.Errorf("ReadFile() error = %v", err)
		return
	}
	
	if string(data) != string(testData) {
		t.Errorf("ReadFile() got %s, expected %s", string(data), string(testData))
	}
}

func TestMemoryFileSystem_Stat(t *testing.T) {
	fs := NewMemoryFileSystem("/tmp")
	
	// Создаем файл
	file, err := fs.Create("test.txt")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	file.Write([]byte("test"))
	file.Close()
	
	// Получаем информацию о файле
	info, err := fs.Stat("test.txt")
	if err != nil {
		t.Errorf("Stat() error = %v", err)
		return
	}
	
	if info.Name() != "test.txt" { //nolint:goconst // test value
		t.Errorf("Stat() name got %s, expected test.txt", info.Name())
	}
	
	if info.Size() != 4 {
		t.Errorf("Stat() size got %d, expected 4", info.Size())
	}
	
	if info.IsDir() {
		t.Errorf("Stat() IsDir() got true, expected false")
	}
	
	// Тест для несуществующего файла
	_, err = fs.Stat("nonexistent.txt")
	if err == nil {
		t.Errorf("Stat() expected error for nonexistent file")
	}
}

func TestMemoryFileSystem_Chmod(t *testing.T) {
	fs := NewMemoryFileSystem("/tmp")
	
	// Создаем файл
	file, err := fs.Create("test.txt")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	file.Close()
	
	// Изменяем права
	err = fs.Chmod("test.txt", 0755)
	if err != nil {
		t.Errorf("Chmod() error = %v", err)
	}
	
	// Проверяем права
	info, err := fs.Stat("test.txt")
	if err != nil {
		t.Errorf("Stat() error = %v", err)
		return
	}
	
	if info.Mode() != 0755 {
		t.Errorf("Chmod() mode got %o, expected %o", info.Mode(), 0755)
	}
}

func TestMemoryFileSystem_WorkingDirectory(t *testing.T) {
	fs := NewMemoryFileSystem("/tmp")
	
	// Получаем текущую директорию
	wd, err := fs.Getwd()
	if err != nil {
		t.Errorf("Getwd() error = %v", err)
	}
	
	if wd != "/tmp" {
		t.Errorf("Getwd() got %s, expected /tmp", wd)
	}
	
	// Создаем новую директорию
	fs.MkdirAll("newdir", 0755)
	
	// Меняем рабочую директорию
	err = fs.Chdir("newdir")
	if err != nil {
		t.Errorf("Chdir() error = %v", err)
	}
	
	// Проверяем новую рабочую директорию
	wd, err = fs.Getwd()
	if err != nil {
		t.Errorf("Getwd() after Chdir() error = %v", err)
	}
	
	expected := filepath.Join("/tmp", "newdir")
	if wd != expected {
		t.Errorf("Getwd() after Chdir() got %s, expected %s", wd, expected)
	}
}

func TestMemoryFileSystem_IsNotExist(t *testing.T) {
	fs := NewMemoryFileSystem("/tmp")
	
	// Тестируем с ошибкой "файл не существует"
	_, err := fs.Open("nonexistent.txt")
	if !fs.IsNotExist(err) {
		t.Errorf("IsNotExist() got false, expected true for nonexistent file")
	}
	
	// Тестируем с другой ошибкой
	_, err = fs.Create("test\x00.txt") // invalid path
	if fs.IsNotExist(err) {
		t.Errorf("IsNotExist() got true, expected false for validation error")
	}
}