package filer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// TestMemoryFileSystem_Coverage тестирует все методы MemoryFileSystem
func TestMemoryFileSystem_Coverage(t *testing.T) {
	mfs := NewMemoryFileSystem("/tmp")

	// Тест Create
	file, err := mfs.Create("test.txt")
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	if file == nil {
		t.Fatal("Created file is nil")
	}
	file.Close()

	// Тест Open существующего файла
	file, err = mfs.Open("test.txt")
	if err != nil {
		t.Fatalf("Failed to open existing file: %v", err)
	}
	file.Close()

	// Тест Open несуществующего файла
	_, err = mfs.Open("nonexistent.txt")
	if err == nil {
		t.Error("Expected error when opening nonexistent file")
	}

	// Тест OpenFile с различными флагами
	file, err = mfs.OpenFile("openfile_test.txt", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open file with flags: %v", err)
	}
	file.Close()

	// Тест Remove
	err = mfs.Remove("test.txt")
	if err != nil {
		t.Fatalf("Failed to remove file: %v", err)
	}

	// Тест Remove несуществующего файла
	err = mfs.Remove("nonexistent.txt")
	if err == nil {
		t.Error("Expected error when removing nonexistent file")
	}

	// Тест Stat существующего файла
	file, _ = mfs.Create("stat_test.txt")
	file.Close()
	info, err := mfs.Stat("stat_test.txt")
	if err != nil {
		t.Fatalf("Failed to stat existing file: %v", err)
	}
	if info == nil {
		t.Fatal("Stat returned nil info")
	}

	// Тест Stat несуществующего файла
	_, err = mfs.Stat("nonexistent.txt")
	if err == nil {
		t.Error("Expected error when stating nonexistent file")
	}

	// Тест проверки существования через Stat
	_, err = mfs.Stat("stat_test.txt")
	if err != nil {
		t.Error("File should exist")
	}
	_, err = mfs.Stat("nonexistent.txt")
	if err == nil {
		t.Error("Nonexistent file should return error")
	}

	// Тест проверки типа через FileInfo
	info, err = mfs.Stat("stat_test.txt")
	if err != nil {
		t.Fatalf("Failed to get file info: %v", err)
	}
	if info.IsDir() {
		t.Error("File should not be identified as directory")
	}
}

// TestMemoryFileSystem_EdgeCases тестирует граничные случаи
func TestMemoryFileSystem_EdgeCases(t *testing.T) {
	mfs := NewMemoryFileSystem("/tmp")

	// Тест с пустым именем файла
	_, err := mfs.Create("")
	if err == nil {
		t.Error("Expected error when creating file with empty name")
	}

	// Тест с очень длинным именем файла
	longName := strings.Repeat("a", 1000)
	file, err := mfs.Create(longName)
	if err != nil {
		t.Fatalf("Failed to create file with long name: %v", err)
	}
	file.Close()

	// Тест создания файла с тем же именем (должен перезаписать)
	file1, err := mfs.Create("duplicate.txt")
	if err != nil {
		t.Fatalf("Failed to create first file: %v", err)
	}
	file1.Write([]byte("first content"))
	file1.Close()

	file2, err := mfs.Create("duplicate.txt")
	if err != nil {
		t.Fatalf("Failed to create duplicate file: %v", err)
	}
	file2.Write([]byte("second content"))
	file2.Close()

	// Проверяем, что содержимое перезаписано
	file, err = mfs.Open("duplicate.txt")
	if err != nil {
		t.Fatalf("Failed to open duplicate file: %v", err)
	}
	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("Failed to read duplicate file: %v", err)
	}
	file.Close()

	if string(content) != "second content" {
		t.Errorf("Expected 'second content', got '%s'", string(content))
	}
}

// TestMemoryFile_Coverage тестирует все методы MemoryFile
func TestMemoryFile_Coverage(t *testing.T) {
	mfs := NewMemoryFileSystem("/tmp")
	file, err := mfs.Create("memory_file_test.txt")
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Тест Write
	testData := []byte("Hello, World!")
	n, err := file.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write to file: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testData), n)
	}

	// Тест Seek к началу
	offset, err := file.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatalf("Failed to seek to start: %v", err)
	}
	if offset != 0 {
		t.Errorf("Expected offset 0, got %d", offset)
	}

	// Тест Read
	buffer := make([]byte, len(testData))
	n, err = file.Read(buffer)
	if err != nil {
		t.Fatalf("Failed to read from file: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Expected to read %d bytes, read %d", len(testData), n)
	}
	if !bytes.Equal(buffer, testData) {
		t.Errorf("Read data doesn't match written data")
	}

	// Тест Seek к концу
	offset, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		t.Fatalf("Failed to seek to end: %v", err)
	}
	if offset != int64(len(testData)) {
		t.Errorf("Expected offset %d, got %d", len(testData), offset)
	}

	// Тест Seek с текущей позиции
	offset, err = file.Seek(-5, io.SeekCurrent)
	if err != nil {
		t.Fatalf("Failed to seek from current: %v", err)
	}
	expectedOffset := int64(len(testData)) - 5
	if offset != expectedOffset {
		t.Errorf("Expected offset %d, got %d", expectedOffset, offset)
	}

	// Тест Close
	err = file.Close()
	if err != nil {
		t.Fatalf("Failed to close file: %v", err)
	}

	// Тест операций после закрытия
	_, err = file.Write([]byte("test"))
	if err == nil {
		t.Error("Expected error when writing to closed file")
	}

	_, err = file.Read(make([]byte, 10))
	if err == nil {
		t.Error("Expected error when reading from closed file")
	}

	_, err = file.Seek(0, io.SeekStart)
	if err == nil {
		t.Error("Expected error when seeking in closed file")
	}
}

// TestMemoryFile_EdgeCases тестирует граничные случаи MemoryFile
func TestMemoryFile_EdgeCases(t *testing.T) {
	mfs := NewMemoryFileSystem("/tmp")
	file, err := mfs.Create("edge_case_test.txt")
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()

	// Тест записи пустых данных
	n, err := file.Write([]byte{})
	if err != nil {
		t.Fatalf("Failed to write empty data: %v", err)
	}
	if n != 0 {
		t.Errorf("Expected to write 0 bytes, wrote %d", n)
	}

	// Тест записи nil данных
	n, err = file.Write(nil)
	if err != nil {
		t.Fatalf("Failed to write nil data: %v", err)
	}
	if n != 0 {
		t.Errorf("Expected to write 0 bytes for nil data, wrote %d", n)
	}

	// Тест чтения в пустой буфер
	n, err = file.Read([]byte{})
	if err != nil && err != io.EOF {
		t.Fatalf("Unexpected error when reading into empty buffer: %v", err)
	}
	if n != 0 {
		t.Errorf("Expected to read 0 bytes into empty buffer, read %d", n)
	}

	// Тест чтения в nil буфер
	n, err = file.Read(nil)
	if err != nil && err != io.EOF {
		t.Fatalf("Unexpected error when reading into nil buffer: %v", err)
	}
	if n != 0 {
		t.Errorf("Expected to read 0 bytes into nil buffer, read %d", n)
	}

	// Тест Seek с недопустимыми параметрами
	_, err = file.Seek(0, 999) // Недопустимый whence
	if err == nil {
		t.Error("Expected error for invalid whence parameter")
	}

	// Тест Seek к отрицательной позиции
	_, err = file.Seek(-100, io.SeekStart)
	if err == nil {
		t.Error("Expected error when seeking to negative position")
	}

	// Тест больших данных
	largeData := make([]byte, 1024*1024) // 1MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	n, err = file.Write(largeData)
	if err != nil {
		t.Fatalf("Failed to write large data: %v", err)
	}
	if n != len(largeData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(largeData), n)
	}

	// Тест чтения больших данных
	file.Seek(0, io.SeekStart)
	readBuffer := make([]byte, len(largeData))
	n, err = file.Read(readBuffer)
	if err != nil {
		t.Fatalf("Failed to read large data: %v", err)
	}
	if n != len(largeData) {
		t.Errorf("Expected to read %d bytes, read %d", len(largeData), n)
	}
	if !bytes.Equal(readBuffer, largeData) {
		t.Error("Large data read doesn't match written data")
	}
}

// TestMemoryFileInfo_Coverage тестирует MemoryFileInfo
func TestMemoryFileInfo_Coverage(t *testing.T) {
	mfs := NewMemoryFileSystem("/tmp")
	file, err := mfs.Create("fileinfo_test.txt")
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Записываем данные для установки размера
	testData := []byte("test content for file info")
	file.Write(testData)
	file.Close()

	// Получаем информацию о файле
	info, err := mfs.Stat("fileinfo_test.txt")
	if err != nil {
		t.Fatalf("Failed to get file info: %v", err)
	}

	// Тестируем все методы FileInfo
	if info.Name() != "fileinfo_test.txt" {
		t.Errorf("Expected name 'fileinfo_test.txt', got '%s'", info.Name())
	}

	if info.Size() != int64(len(testData)) {
		t.Errorf("Expected size %d, got %d", len(testData), info.Size())
	}

	if info.IsDir() {
		t.Error("File should not be identified as directory")
	}

	// Проверяем, что ModTime возвращает разумное время
	modTime := info.ModTime()
	if modTime.IsZero() {
		t.Error("ModTime should not be zero")
	}
	if time.Since(modTime) > time.Hour {
		t.Error("ModTime seems too old")
	}

	// Проверяем Mode
	mode := info.Mode()
	if mode == 0 {
		t.Error("Mode should not be zero")
	}

	// Проверяем Sys (может быть nil)
	sys := info.Sys()
	_ = sys // Просто проверяем, что метод не паникует
}

// TestMemoryFileSystem_Concurrent тестирует concurrent доступ
func TestMemoryFileSystem_Concurrent(t *testing.T) {
	mfs := NewMemoryFileSystem("/tmp")

	// Создаем несколько файлов одновременно
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(index int) {
			defer func() { done <- true }()
			filename := fmt.Sprintf("concurrent_test_%d.txt", index)
			file, err := mfs.Create(filename)
			if err != nil {
				t.Errorf("Failed to create file %s: %v", filename, err)
				return
			}
			defer file.Close()

			data := fmt.Sprintf("Content for file %d", index)
			file.Write([]byte(data))
		}(i)
	}

	// Ждем завершения всех горутин
	for i := 0; i < 10; i++ {
		<-done
	}

	// Проверяем, что все файлы созданы
	for i := 0; i < 10; i++ {
		filename := fmt.Sprintf("concurrent_test_%d.txt", i)
		_, err := mfs.Stat(filename)
		if err != nil {
			t.Errorf("File %s was not created: %v", filename, err)
		}
	}
}