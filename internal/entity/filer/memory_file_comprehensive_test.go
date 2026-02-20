package filer

import (
	"io"
	"os"
	"testing"
	"time"
)

func TestMemoryFile_Read(t *testing.T) {
	file := NewMemoryFile("/tmp/test.txt", 0644)
	testData := []byte("Hello, World!")
	
	// Записываем данные
	file.Write(testData)
	file.Seek(0, io.SeekStart) // Возвращаемся в начало
	
	tests := []struct {
		name     string
		bufSize  int
		expected string
		wantErr  bool
	}{
		{
			name:     "Read full data",
			bufSize:  20,
			expected: "Hello, World!",
			wantErr:  false,
		},
		{
			name:     "Read partial data",
			bufSize:  5,
			expected: "Hello",
			wantErr:  false,
		},
		{
			name:     "Read with small buffer",
			bufSize:  1,
			expected: "H",
			wantErr:  false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file.Seek(0, io.SeekStart) // Сброс позиции
			buf := make([]byte, tt.bufSize)
			n, err := file.Read(buf)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("Read() expected error, got nil")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Read() error = %v", err)
				return
			}
			
			if string(buf[:n]) != tt.expected {
				t.Errorf("Read() got %s, expected %s", string(buf[:n]), tt.expected)
			}
		})
	}
}

func TestMemoryFile_ReadEOF(t *testing.T) {
	file := NewMemoryFile("/tmp/test.txt", 0644)
	file.Write([]byte("test"))
	
	// Читаем до конца файла
	file.Seek(0, io.SeekStart)
	buf := make([]byte, 4)
	n, err := file.Read(buf)
	if err != nil || n != 4 {
		t.Errorf("First read failed: n=%d, err=%v", n, err)
	}
	
	// Следующее чтение должно вернуть EOF
	n, err = file.Read(buf)
	if err != io.EOF {
		t.Errorf("Read() expected EOF, got %v", err)
	}
	if n != 0 {
		t.Errorf("Read() expected 0 bytes, got %d", n)
	}
}

func TestMemoryFile_ReadClosed(t *testing.T) {
	file := NewMemoryFile("/tmp/test.txt", 0644)
	file.Write([]byte("test"))
	file.Close()
	
	// Попытка чтения из закрытого файла
	buf := make([]byte, 10)
	_, err := file.Read(buf)
	if err != ErrFileClosed {
		t.Errorf("Read() expected ErrFileClosed, got %v", err)
	}
}

func TestMemoryFile_Write(t *testing.T) {
	file := NewMemoryFile("/tmp/test.txt", 0644)
	
	tests := []struct {
		name     string
		data     []byte
		expected int
		wantErr  bool
	}{
		{
			name:     "Write normal data",
			data:     []byte("Hello"),
			expected: 5,
			wantErr:  false,
		},
		{
			name:     "Write empty data",
			data:     []byte{},
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "Write large data",
			data:     make([]byte, 1000),
			expected: 1000,
			wantErr:  false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file.Seek(0, io.SeekStart) // Сброс позиции
			n, err := file.Write(tt.data)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("Write() expected error, got nil")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Write() error = %v", err)
				return
			}
			
			if n != tt.expected {
				t.Errorf("Write() wrote %d bytes, expected %d", n, tt.expected)
			}
		})
	}
}

func TestMemoryFile_WriteAppend(t *testing.T) {
	file := NewMemoryFile("/tmp/test.txt", 0644)
	
	// Записываем первую порцию данных
	n1, err := file.Write([]byte("Hello"))
	if err != nil || n1 != 5 {
		t.Errorf("First write failed: n=%d, err=%v", n1, err)
	}
	
	// Записываем вторую порцию данных
	n2, err := file.Write([]byte(", World!"))
	if err != nil || n2 != 8 {
		t.Errorf("Second write failed: n=%d, err=%v", n2, err)
	}
	
	// Проверяем общий размер
	if file.Size() != 13 {
		t.Errorf("File size got %d, expected 13", file.Size())
	}
	
	// Читаем все данные
	file.Seek(0, io.SeekStart)
	buf := make([]byte, 20)
	n, err := file.Read(buf)
	if err != nil {
		t.Errorf("Read() error = %v", err)
	}
	
	if string(buf[:n]) != "Hello, World!" {
		t.Errorf("Read() got %s, expected 'Hello, World!'", string(buf[:n]))
	}
}

func TestMemoryFile_WriteClosed(t *testing.T) {
	file := NewMemoryFile("/tmp/test.txt", 0644)
	file.Close()
	
	// Попытка записи в закрытый файл
	_, err := file.Write([]byte("test"))
	if err != ErrFileClosed {
		t.Errorf("Write() expected ErrFileClosed, got %v", err)
	}
}

func TestMemoryFile_WriteReadOnly(t *testing.T) {
	file := NewMemoryFile("/tmp/test.txt", 0644)
	file.SetFlag(os.O_RDONLY)
	
	// Попытка записи в файл только для чтения
	_, err := file.Write([]byte("test"))
	if err != ErrReadOnlyFile {
		t.Errorf("Write() expected ErrReadOnlyFile, got %v", err)
	}
}

func TestMemoryFile_Seek(t *testing.T) {
	file := NewMemoryFile("/tmp/test.txt", 0644)
	testData := []byte("0123456789")
	file.Write(testData)
	
	tests := []struct {
		name     string
		offset   int64
		whence   int
		expected int64
		wantErr  bool
	}{
		{
			name:     "Seek to start",
			offset:   0,
			whence:   io.SeekStart,
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "Seek to middle",
			offset:   5,
			whence:   io.SeekStart,
			expected: 5,
			wantErr:  false,
		},
		{
			name:     "Seek to end",
			offset:   0,
			whence:   io.SeekEnd,
			expected: 10,
			wantErr:  false,
		},
		{
			name:     "Seek relative forward",
			offset:   3,
			whence:   io.SeekCurrent,
			expected: 8, // 5 + 3
			wantErr:  false,
		},
		{
			name:     "Seek relative backward",
			offset:   -2,
			whence:   io.SeekCurrent,
			expected: 6, // 8 - 2
			wantErr:  false,
		},
		{
			name:     "Seek before start",
			offset:   -1,
			whence:   io.SeekStart,
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "Seek beyond end",
			offset:   20,
			whence:   io.SeekStart,
			expected: 20,
			wantErr:  false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем начальную позицию для относительных тестов
			if tt.whence == io.SeekCurrent {
				switch tt.name {
				case "Seek relative forward":
					file.Seek(5, io.SeekStart)
				case "Seek relative backward":
					file.Seek(8, io.SeekStart) // Позиция после предыдущего теста
				}
			} else {
				// Для абсолютных тестов сбрасываем позицию
				file.Seek(0, io.SeekStart)
			}
			
			pos, err := file.Seek(tt.offset, tt.whence)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("Seek() expected error, got nil")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Seek() error = %v", err)
				return
			}
			
			if pos != tt.expected {
				t.Errorf("Seek() position got %d, expected %d", pos, tt.expected)
			}
		})
	}
}

func TestMemoryFile_SeekClosed(t *testing.T) {
	file := NewMemoryFile("/tmp/test.txt", 0644)
	file.Close()
	
	// Попытка seek в закрытом файле
	_, err := file.Seek(0, io.SeekStart)
	if err != ErrFileClosed {
		t.Errorf("Seek() expected ErrFileClosed, got %v", err)
	}
}

func TestMemoryFile_Close(t *testing.T) {
	file := NewMemoryFile("/tmp/test.txt", 0644)
	
	// Проверяем, что файл не закрыт
	if file.IsClosed() {
		t.Errorf("IsClosed() got true, expected false")
	}
	
	// Закрываем файл
	err := file.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
	
	// Проверяем, что файл закрыт
	if !file.IsClosed() {
		t.Errorf("IsClosed() got false, expected true")
	}
	
	// Повторное закрытие не должно вызывать ошибку
	err = file.Close()
	if err != nil {
		t.Errorf("Second Close() error = %v", err)
	}
}

func TestMemoryFile_Stat(t *testing.T) {
	file := NewMemoryFile("/tmp/test.txt", 0644)
	file.Write([]byte("test data"))
	
	info, err := file.Stat()
	if err != nil {
		t.Errorf("Stat() error = %v", err)
		return
	}
	
	if info.Name() != "test.txt" { //nolint:goconst // test value
		t.Errorf("Stat() name got %s, expected test.txt", info.Name())
	}
	
	if info.Size() != 9 {
		t.Errorf("Stat() size got %d, expected 9", info.Size())
	}
	
	if info.Mode() != 0644 {
		t.Errorf("Stat() mode got %o, expected %o", info.Mode(), 0644)
	}
	
	if info.IsDir() {
		t.Errorf("Stat() IsDir() got true, expected false")
	}
}

func TestMemoryFile_Truncate(t *testing.T) {
	file := NewMemoryFile("/tmp/test.txt", 0644)
	file.Write([]byte("Hello, World!"))
	
	// Обрезаем до 5 байт
	err := file.Truncate(5)
	if err != nil {
		t.Errorf("Truncate() error = %v", err)
	}
	
	if file.Size() != 5 {
		t.Errorf("Size() after truncate got %d, expected 5", file.Size())
	}
	
	// Читаем данные
	file.Seek(0, io.SeekStart)
	buf := make([]byte, 10)
	n, err := file.Read(buf)
	if err != nil {
		t.Errorf("Read() after truncate error = %v", err)
	}
	
	if string(buf[:n]) != "Hello" {
		t.Errorf("Read() after truncate got %s, expected 'Hello'", string(buf[:n]))
	}
	
	// Расширяем файл
	err = file.Truncate(10)
	if err != nil {
		t.Errorf("Truncate() expand error = %v", err)
	}
	
	if file.Size() != 10 {
		t.Errorf("Size() after expand got %d, expected 10", file.Size())
	}
}

func TestMemoryFile_TruncateClosed(t *testing.T) {
	file := NewMemoryFile("/tmp/test.txt", 0644)
	file.Close()
	
	// Попытка truncate закрытого файла
	err := file.Truncate(5)
	if err != ErrFileClosed {
		t.Errorf("Truncate() expected ErrFileClosed, got %v", err)
	}
}

func TestMemoryFile_ReadWriteAt(t *testing.T) {
	file := NewMemoryFile("/tmp/test.txt", 0644)
	testData := []byte("0123456789")
	file.Write(testData)
	
	// Тест WriteAt
	n, err := file.WriteAt([]byte("ABC"), 3)
	if err != nil {
		t.Errorf("WriteAt() error = %v", err)
	}
	if n != 3 {
		t.Errorf("WriteAt() wrote %d bytes, expected 3", n)
	}
	
	// Тест ReadAt
	buf := make([]byte, 5)
	n, err = file.ReadAt(buf, 2)
	if err != nil {
		t.Errorf("ReadAt() error = %v", err)
	}
	if n != 5 {
		t.Errorf("ReadAt() read %d bytes, expected 5", n)
	}
	
	if string(buf) != "2ABC6" {
		t.Errorf("ReadAt() got %s, expected '2ABC6'", string(buf))
	}
	
	// Проверяем, что позиция файла не изменилась
	file.Seek(0, io.SeekStart)
	allData := make([]byte, 10)
	n, err = file.Read(allData)
	if err != nil {
		t.Errorf("Read() error = %v", err)
	}
	
	if string(allData[:n]) != "012ABC6789" {
		t.Errorf("Read() got %s, expected '012ABC6789'", string(allData[:n]))
	}
}

func TestMemoryFile_Clone(t *testing.T) {
	original := NewMemoryFile("/tmp/test.txt", 0644)
	original.Write([]byte("test data"))
	original.Seek(5, io.SeekStart)
	
	cloned := original.Clone()
	if cloned == nil {
		t.Errorf("Clone() returned nil")
		return
	}
	
	// Проверяем, что клон имеет те же данные
	cloned.Seek(0, io.SeekStart)
	buf := make([]byte, 20)
	n, err := cloned.Read(buf)
	if err != nil {
		t.Errorf("Clone Read() error = %v", err)
	}
	
	if string(buf[:n]) != "test data" { //nolint:goconst // test value
		t.Errorf("Clone Read() got %s, expected 'test data'", string(buf[:n]))
	}
	
	// Проверяем, что изменения в клоне не влияют на оригинал
	cloned.Write([]byte(" modified"))
	
	original.Seek(0, io.SeekStart)
	n, err = original.Read(buf)
	if err != nil {
		t.Errorf("Original Read() error = %v", err)
	}
	
	if string(buf[:n]) != "test data" { //nolint:goconst // test value
		t.Errorf("Original Read() got %s, expected 'test data'", string(buf[:n]))
	}
}

func TestMemoryFile_SyncOperations(t *testing.T) {
	file := NewMemoryFile("/tmp/test.txt", 0644)
	
	// Sync всегда должен возвращать nil для файлов в памяти
	err := file.Sync()
	if err != nil {
		t.Errorf("Sync() error = %v", err)
	}
	
	// Sync закрытого файла
	file.Close()
	err = file.Sync()
	if err != ErrFileClosed {
		t.Errorf("Sync() expected ErrFileClosed, got %v", err)
	}
}

func TestMemoryFile_Chmod(t *testing.T) {
	file := NewMemoryFile("/tmp/test.txt", 0644)
	
	// Изменяем права
	err := file.Chmod(0755)
	if err != nil {
		t.Errorf("Chmod() error = %v", err)
	}
	
	// Проверяем права через Stat
	info, err := file.Stat()
	if err != nil {
		t.Errorf("Stat() error = %v", err)
		return
	}
	
	if info.Mode() != 0755 {
		t.Errorf("Chmod() mode got %o, expected %o", info.Mode(), 0755)
	}
	
	// Chmod закрытого файла
	file.Close()
	err = file.Chmod(0644)
	if err != ErrFileClosed {
		t.Errorf("Chmod() expected ErrFileClosed, got %v", err)
	}
}

func TestMemoryFile_Chown(t *testing.T) {
	file := NewMemoryFile("/tmp/test.txt", 0644)
	
	// Chown всегда должен возвращать nil для файлов в памяти
	err := file.Chown(1000, 1000)
	if err != nil {
		t.Errorf("Chown() error = %v", err)
	}
	
	// Chown закрытого файла
	file.Close()
	err = file.Chown(1000, 1000)
	if err != ErrFileClosed {
		t.Errorf("Chown() expected ErrFileClosed, got %v", err)
	}
}

func TestMemoryFile_IsEmptyCheck(t *testing.T) {
	file := NewMemoryFile("/tmp/test.txt", 0644)
	
	// Новый файл должен быть пустым
	if !file.IsEmpty() {
		t.Errorf("IsEmpty() got false, expected true for new file")
	}
	
	// После записи файл не должен быть пустым
	file.Write([]byte("test"))
	if file.IsEmpty() {
		t.Errorf("IsEmpty() got true, expected false after write")
	}
	
	// После truncate до 0 файл должен быть пустым
	file.Truncate(0)
	if !file.IsEmpty() {
		t.Errorf("IsEmpty() got false, expected true after truncate to 0")
	}
}

func TestMemoryFile_FlagOperations(t *testing.T) {
	file := NewMemoryFile("/tmp/test.txt", 0644)
	
	// Проверяем начальный флаг
	if file.GetFlag() != os.O_RDWR {
		t.Errorf("GetFlag() got %d, expected %d", file.GetFlag(), os.O_RDWR)
	}
	
	// Устанавливаем новый флаг
	file.SetFlag(os.O_RDONLY)
	if file.GetFlag() != os.O_RDONLY {
		t.Errorf("GetFlag() after SetFlag got %d, expected %d", file.GetFlag(), os.O_RDONLY)
	}
	
	// Проверяем, что запись теперь запрещена
	_, err := file.Write([]byte("test"))
	if err != ErrReadOnlyFile {
		t.Errorf("Write() expected ErrReadOnlyFile, got %v", err)
	}
}

func TestMemoryFile_ConcurrentAccess(t *testing.T) {
	file := NewMemoryFile("/tmp/test.txt", 0644)
	
	// Тест на безопасность при конкурентном доступе
	done := make(chan bool, 2)
	
	// Горутина для записи
	go func() {
		for i := 0; i < 100; i++ {
			file.Write([]byte("a"))
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()
	
	// Горутина для чтения
	go func() {
		for i := 0; i < 100; i++ {
			buf := make([]byte, 1)
			file.ReadAt(buf, 0)
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()
	
	// Ждем завершения обеих горутин
	<-done
	<-done
	
	// Проверяем, что файл не поврежден
	if file.Size() != 100 {
		t.Errorf("Size() after concurrent access got %d, expected 100", file.Size())
	}
}