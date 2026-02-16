package filer

import (
	"io"
	"testing"
)

// TestMemoryFile_ReadAt тестирует чтение данных с указанной позиции
func TestMemoryFile_ReadAt(t *testing.T) {
	file := NewMemoryFile("test.txt", 0644)
	
	// Записываем данные
	data := []byte("0123456789")
	file.Write(data)
	
	// Читаем с начала
	buf := make([]byte, 5)
	n, err := file.ReadAt(buf, 0)
	if err != nil {
		t.Errorf("ReadAt failed: %v", err)
	}
	if n != 5 {
		t.Errorf("Expected to read 5 bytes, got %d", n)
	}
	if string(buf) != "01234" {
		t.Errorf("Expected '01234', got '%s'", string(buf))
	}
	
	// Читаем с середины
	buf2 := make([]byte, 3)
	n, err = file.ReadAt(buf2, 5)
	if err != nil {
		t.Errorf("ReadAt failed: %v", err)
	}
	if n != 3 {
		t.Errorf("Expected to read 3 bytes, got %d", n)
	}
	if string(buf2) != "567" {
		t.Errorf("Expected '567', got '%s'", string(buf2))
	}
	
	// Читаем с позиции за пределами данных (должен вернуть EOF)
	buf3 := make([]byte, 5)
	n, err = file.ReadAt(buf3, 15)
	if err != io.EOF {
		t.Errorf("Expected EOF, got %v", err)
	}
	if n != 0 {
		t.Errorf("Expected to read 0 bytes, got %d", n)
	}
	
	// Тест с отрицательной позицией
	n, err = file.ReadAt(buf, -1)
	if err != io.EOF {
		t.Errorf("Expected EOF with negative position, got %v", err)
	}
	if n != 0 {
		t.Errorf("Expected to read 0 bytes with negative position, got %d", n)
	}
}

// TestMemoryFile_WriteAt тестирует запись данных в указанную позицию
func TestMemoryFile_WriteAt(t *testing.T) {
	file := NewMemoryFile("test.txt", 0644)
	
	// Записываем данные в начало
	data := []byte("hello")
	n, err := file.WriteAt(data, 0)
	if err != nil {
		t.Errorf("WriteAt failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected to write %d bytes, got %d", len(data), n)
	}
	
	// Проверяем содержимое
	file.Seek(0, io.SeekStart)
	buf := make([]byte, len(data))
	file.Read(buf)
	if string(buf) != "hello" {
		t.Errorf("Expected 'hello', got '%s'", string(buf))
	}
	
	// Записываем данные в середину
	data2 := []byte("world")
	n, err = file.WriteAt(data2, 2)
	if err != nil {
		t.Errorf("WriteAt failed: %v", err)
	}
	if n != len(data2) {
		t.Errorf("Expected to write %d bytes, got %d", len(data2), n)
	}
	
	// Проверяем содержимое
	file.Seek(0, io.SeekStart)
	buf2 := make([]byte, 7)
	file.Read(buf2)
	if string(buf2) != "heworld" {
		t.Errorf("Expected 'heworld', got '%s'", string(buf2))
	}
	
	// Записываем данные с расширением файла
	data3 := []byte("test")
	n, err = file.WriteAt(data3, 10)
	if err != nil {
		t.Errorf("WriteAt failed: %v", err)
	}
	if n != len(data3) {
		t.Errorf("Expected to write %d bytes, got %d", len(data3), n)
	}
	
	// Проверяем размер
	if file.Size() != 14 {
		t.Errorf("Expected size 14, got %d", file.Size())
	}
	
	// Тест с отрицательной позицией
	n, err = file.WriteAt(data, -1)
	if err == nil {
		t.Error("WriteAt should fail with negative position")
	}
	if n != 0 {
		t.Errorf("Expected to write 0 bytes with negative position, got %d", n)
	}
}

// TestMemoryFile_Sync тестирует синхронизацию файла
func TestMemoryFile_Sync(t *testing.T) {
	file := NewMemoryFile("test.txt", 0644)
	
	// Тест Sync на открытом файле
	err := file.Sync()
	if err != nil {
		t.Errorf("Sync failed: %v", err)
	}
	
	// Закрываем файл
	file.Close()
	
	// Тест Sync на закрытом файле
	err = file.Sync()
	if err != ErrFileClosed {
		t.Errorf("Expected ErrFileClosed, got %v", err)
	}
}

// TestMemoryFile_IsEmpty тестирует проверку пустоты файла
func TestMemoryFile_IsEmpty(t *testing.T) {
	file := NewMemoryFile("test.txt", 0644)
	
	// Новый файл должен быть пустым
	if !file.IsEmpty() {
		t.Error("New file should be empty")
	}
	
	// Записываем данные
	file.Write([]byte("test"))
	
	// Файл больше не должен быть пустым
	if file.IsEmpty() {
		t.Error("File with data should not be empty")
	}
	
	// Обрезаем файл до нулевого размера
	file.Truncate(0)
	
	// Файл снова должен быть пустым
	if !file.IsEmpty() {
		t.Error("Truncated file should be empty")
	}
}

// TestMemoryFile_IsClosed тестирует проверку закрытия файла
func TestMemoryFile_IsClosed(t *testing.T) {
	file := NewMemoryFile("test.txt", 0644)
	
	// Новый файл не должен быть закрыт
	if file.IsClosed() {
		t.Error("New file should not be closed")
	}
	
	// Закрываем файл
	file.Close()
	
	// Файл должен быть закрыт
	if !file.IsClosed() {
		t.Error("Closed file should be closed")
	}
}

// TestMemoryFile_Size тестирует получение размера файла
func TestMemoryFile_Size(t *testing.T) {
	file := NewMemoryFile("test.txt", 0644)
	
	// Новый файл должен иметь размер 0
	if file.Size() != 0 {
		t.Errorf("Expected size 0, got %d", file.Size())
	}
	
	// Записываем данные
	data := []byte("hello world")
	file.Write(data)
	
	// Проверяем размер
	if file.Size() != int64(len(data)) {
		t.Errorf("Expected size %d, got %d", len(data), file.Size())
	}
	
	// Обрезаем файл
	file.Truncate(5)
	
	// Проверяем размер после обрезки
	if file.Size() != 5 {
		t.Errorf("Expected size 5 after truncate, got %d", file.Size())
	}
	
	// Закрываем файл
	file.Close()
	
	// Проверяем размер на закрытом файле
	if file.Size() != 5 {
		t.Errorf("Expected size 5 on closed file, got %d", file.Size())
	}
}