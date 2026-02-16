package filer

import (
	"testing"
)

// TestMemoryFile_SyncAdditional тестирует Sync дополнительно
func TestMemoryFile_SyncAdditional(t *testing.T) {
	file := NewMemoryFile("test.txt", 0644)
	
	// Тест Sync на новом файле (должен успешно выполниться)
	err := file.Sync()
	if err != nil {
		t.Errorf("Sync on new file failed: %v", err)
	}
	
	// Записываем данные в файл
	data := []byte("test data")
	_, err = file.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	
	// Тест Sync после записи данных
	err = file.Sync()
	if err != nil {
		t.Errorf("Sync after write failed: %v", err)
	}
	
	// Закрываем файл
	err = file.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	
	// Тест Sync на закрытом файле
	err = file.Sync()
	if err != ErrFileClosed {
		t.Errorf("Expected ErrFileClosed on closed file, got %v", err)
	}
	
	// Создаем еще один файл
	file2 := NewMemoryFile("test2.txt", 0644)
	
	// Записываем в него данные
	_, err = file2.Write([]byte("more test data"))
	if err != nil {
		t.Fatalf("Write to second file failed: %v", err)
	}
	
	// Тест Sync на втором файле
	err = file2.Sync()
	if err != nil {
		t.Errorf("Sync on second file failed: %v", err)
	}
	
	// Закрываем второй файл
	file2.Close()
	
	// Тест Sync на закрытом втором файле
	err = file2.Sync()
	if err != ErrFileClosed {
		t.Errorf("Expected ErrFileClosed on closed second file, got %v", err)
	}
}