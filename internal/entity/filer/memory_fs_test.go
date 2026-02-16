package filer

import (
	"bytes"
	"io"
	"testing"
)

func TestMemoryFileSystem_Basic(t *testing.T) {
	fs := NewMemoryFileSystem("/tmp")
	
	// Тест создания директории
	err := fs.MkdirAll("test/subdir", 0755)
	if err != nil {
		t.Errorf("MkdirAll failed: %v", err)
	}
	
	// Проверяем, что директория создана
	info, err := fs.Stat("test/subdir")
	if err != nil {
		t.Errorf("Stat failed: %v", err)
	}
	if !info.IsDir() {
		t.Error("Expected directory")
	}
}

func TestMemoryFileSystem_FileOperations(t *testing.T) {
	fs := NewMemoryFileSystem("/tmp")
	
	// Создание файла
	file, err := fs.Create("test.txt")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer file.Close()
	
	// Запись в файл
	testData := []byte("Hello, Memory World!")
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
	if !bytes.Equal(data, testData) {
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

func TestMemoryFileSystem_ReadDir(t *testing.T) {
	fs := NewMemoryFileSystem("tmp")
	
	// Создаем несколько файлов и директорий
	fs.MkdirAll("dir1", 0755)
	fs.MkdirAll("dir2", 0755)
	
	file1, _ := fs.Create("file1.txt")
	file1.Close()
	file2, _ := fs.Create("file2.txt")
	file2.Close()
	
	// Читаем содержимое директории
	entries, err := fs.ReadDir(".")
	if err != nil {
		t.Errorf("ReadDir failed: %v", err)
	}
	
	if len(entries) != 4 {
		t.Errorf("Expected 4 entries, got %d", len(entries))
	}
}

func TestMemoryFile_SeekOperations(t *testing.T) {
	fs := NewMemoryFileSystem("tmp")
	
	file, err := fs.Create("seek_test.txt")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer file.Close()
	
	// Записываем данные
	testData := []byte("0123456789")
	file.Write(testData)
	
	// Seek к началу
	pos, err := file.Seek(0, io.SeekStart)
	if err != nil {
		t.Errorf("Seek failed: %v", err)
	}
	if pos != 0 {
		t.Errorf("Expected position 0, got %d", pos)
	}
	
	// Читаем первые 5 байт
	buf := make([]byte, 5)
	n, err := file.Read(buf)
	if err != nil {
		t.Errorf("Read failed: %v", err)
	}
	if n != 5 {
		t.Errorf("Expected to read 5 bytes, got %d", n)
	}
	if string(buf) != "01234" {
		t.Errorf("Expected '01234', got '%s'", string(buf))
	}
	
	// Seek к концу
	pos, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		t.Errorf("Seek to end failed: %v", err)
	}
	if pos != int64(len(testData)) {
		t.Errorf("Expected position %d, got %d", len(testData), pos)
	}
}

func TestMemoryFileSystem_Rename(t *testing.T) {
	fs := NewMemoryFileSystem("/tmp")
	
	// Создаем файл
	file, err := fs.Create("old_name.txt")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	file.Write([]byte("test content"))
	file.Close()
	
	// Переименовываем файл
	err = fs.Rename("old_name.txt", "new_name.txt")
	if err != nil {
		t.Errorf("Rename failed: %v", err)
	}
	
	// Проверяем, что старый файл не существует
	if _, statErr := fs.Stat("old_name.txt"); !fs.IsNotExist(statErr) {
		t.Error("Old file still exists")
	}
	
	// Проверяем, что новый файл существует
	data, err := fs.ReadFile("new_name.txt")
	if err != nil {
		t.Errorf("ReadFile failed: %v", err)
	}
	if string(data) != "test content" {
		t.Errorf("File content mismatch")
	}
}