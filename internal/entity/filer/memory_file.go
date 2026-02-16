package filer

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// MemoryFile представляет файл в файловой системе в памяти
type MemoryFile struct {
	mu      sync.RWMutex
	path    string
	data    []byte
	offset  int64
	mode    os.FileMode
	modTime time.Time
	closed  bool
	flag    int
}

// NewMemoryFile создает новый файл в памяти
func NewMemoryFile(path string, mode os.FileMode) *MemoryFile {
	return &MemoryFile{
		path:    path,
		data:    make([]byte, 0),
		offset:  0,
		mode:    mode,
		modTime: time.Now(),
		closed:  false,
		flag:    os.O_RDWR,
	}
}

// Read читает данные из файла
func (f *MemoryFile) Read(p []byte) (n int, err error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return 0, ErrFileClosed
	}
	
	if f.offset >= int64(len(f.data)) {
		return 0, io.EOF
	}
	
	n = copy(p, f.data[f.offset:])
	f.offset += int64(n)
	
	return n, nil
}

// Write записывает данные в файл
func (f *MemoryFile) Write(p []byte) (n int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.closed {
		return 0, ErrFileClosed
	}
	
	// Проверяем права на запись
	if f.flag&os.O_WRONLY == 0 && f.flag&os.O_RDWR == 0 {
		return 0, ErrReadOnlyFile
	}
	
	// Расширяем данные если нужно
	neededSize := f.offset + int64(len(p))
	if neededSize > int64(len(f.data)) {
		newData := make([]byte, neededSize)
		copy(newData, f.data)
		f.data = newData
	}
	
	n = copy(f.data[f.offset:], p)
	f.offset += int64(n)
	f.modTime = time.Now()
	
	return n, nil
}

// Seek изменяет позицию в файле
func (f *MemoryFile) Seek(offset int64, whence int) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.closed {
		return 0, ErrFileClosed
	}
	
	var newOffset int64
	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset = f.offset + offset
	case io.SeekEnd:
		newOffset = int64(len(f.data)) + offset
	default:
		return 0, ErrUnsupportedOperation
	}
	
	if newOffset < 0 {
		return 0, ErrInvalidPath
	}
	
	f.offset = newOffset
	return f.offset, nil
}

// Close закрывает файл
func (f *MemoryFile) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.closed {
		return nil // Повторное закрытие не является ошибкой
	}
	
	f.closed = true
	return nil
}

// Stat возвращает информацию о файле
func (f *MemoryFile) Stat() (os.FileInfo, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return nil, ErrFileClosed
	}
	
	return &memoryFileInfo{
		name:    f.Name(),
		size:    int64(len(f.data)),
		mode:    f.mode,
		modTime: f.modTime,
		isDir:   false,
	}, nil
}

// Name возвращает имя файла
func (f *MemoryFile) Name() string {
	return filepath.Base(f.path)
}

// Truncate обрезает файл до указанного размера
func (f *MemoryFile) Truncate(size int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.closed {
		return ErrFileClosed
	}
	
	if size < 0 {
		return ErrInvalidPath
	}
	
	switch {
	case size == 0:
		f.data = make([]byte, 0)
	case size < int64(len(f.data)):
		f.data = f.data[:size]
	case size > int64(len(f.data)):
		newData := make([]byte, size)
		copy(newData, f.data)
		f.data = newData
	}
	
	// Корректируем offset если он больше нового размера
	if f.offset > size {
		f.offset = size
	}
	
	f.modTime = time.Now()
	return nil
}

// Sync синхронизирует файл (для совместимости, в памяти ничего не делает)
func (f *MemoryFile) Sync() error {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return ErrFileClosed
	}
	
	return nil
}

// ReadAt читает данные с указанной позиции
func (f *MemoryFile) ReadAt(p []byte, off int64) (n int, err error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return 0, ErrFileClosed
	}
	
	if off < 0 || off >= int64(len(f.data)) {
		return 0, io.EOF
	}
	
	n = copy(p, f.data[off:])
	if n < len(p) {
		err = io.EOF
	}
	
	return n, err
}

// WriteAt записывает данные в указанную позицию
func (f *MemoryFile) WriteAt(p []byte, off int64) (n int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.closed {
		return 0, ErrFileClosed
	}
	
	// Проверяем права на запись
	if f.flag&os.O_WRONLY == 0 && f.flag&os.O_RDWR == 0 {
		return 0, ErrReadOnlyFile
	}
	
	if off < 0 {
		return 0, ErrInvalidPath
	}
	
	// Расширяем данные если нужно
	neededSize := off + int64(len(p))
	if neededSize > int64(len(f.data)) {
		newData := make([]byte, neededSize)
		copy(newData, f.data)
		f.data = newData
	}
	
	n = copy(f.data[off:], p)
	f.modTime = time.Now()
	
	return n, nil
}

// Clone создает копию файла для независимого использования
func (f *MemoryFile) Clone() File {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	clone := &MemoryFile{
		path:    f.path,
		data:    make([]byte, len(f.data)),
		offset:  0, // Новый файл начинается с начала
		mode:    f.mode,
		modTime: f.modTime,
		closed:  false,
		flag:    f.flag,
	}
	
	copy(clone.data, f.data)
	return clone
}

// Size возвращает размер файла
func (f *MemoryFile) Size() int64 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	return int64(len(f.data))
}

// IsEmpty проверяет, пуст ли файл
func (f *MemoryFile) IsEmpty() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	return len(f.data) == 0
}

// IsClosed проверяет, закрыт ли файл
func (f *MemoryFile) IsClosed() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	return f.closed
}

// SetFlag устанавливает флаги файла
func (f *MemoryFile) SetFlag(flag int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	f.flag = flag
}

// GetFlag возвращает флаги файла
func (f *MemoryFile) GetFlag() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	return f.flag
}

// Chmod изменяет режим файла
func (f *MemoryFile) Chmod(mode os.FileMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.closed {
		return ErrFileClosed
	}
	
	f.mode = mode
	f.modTime = time.Now()
	return nil
}

// Chown изменяет владельца файла (заглушка для совместимости)
func (f *MemoryFile) Chown(_, _ int) error {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return ErrFileClosed
	}
	
	// В файловой системе в памяти операция chown не имеет смысла
	// Возвращаем nil для совместимости
	return nil
}