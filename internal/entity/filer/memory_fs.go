package filer

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryFileSystem представляет файловую систему в памяти
type MemoryFileSystem struct {
	mu    sync.RWMutex
	files map[string]*MemoryFile
	dirs  map[string]*MemoryDir
	root  string
}

// MemoryDir представляет директорию в памяти
type MemoryDir struct {
	name    string
	mode    os.FileMode
	modTime time.Time
	parent  string
}

// Убеждаемся, что MemoryFileSystem реализует интерфейс FileSystem
var _ FileSystem = (*MemoryFileSystem)(nil)

// NewMemoryFileSystem создает новую файловую систему в памяти
func NewMemoryFileSystem(root string) *MemoryFileSystem {
	fs := &MemoryFileSystem{
		files: make(map[string]*MemoryFile),
		dirs:  make(map[string]*MemoryDir),
		root:  filepath.Clean(root),
	}
	
	// Создаем корневую директорию
	fs.dirs[fs.root] = &MemoryDir{
		name:    filepath.Base(fs.root),
		mode:    0755,
		modTime: time.Now(),
		parent:  "",
	}
	
	return fs
}

// Create создает новый файл
func (fs *MemoryFileSystem) Create(name string) (File, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	if err := ValidatePath(name); err != nil {
		return nil, WrapError("create file", name, err, MemoryFS)
	}
	
	path := fs.resolvePath(name)
	
	// Создаем родительские директории если нужно
	if err := fs.mkdirAllLocked(filepath.Dir(path)); err != nil {
		return nil, WrapError("create parent directories", filepath.Dir(path), err, MemoryFS)
	}
	
	// Создаем файл
	file := NewMemoryFile(path, 0644)
	fs.files[path] = file
	
	return file, nil
}

// CreateTemp создает временный файл
func (fs *MemoryFileSystem) CreateTemp(dir, pattern string) (File, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	if dir == "" {
		dir = "."
	}
	
	if err := ValidatePath(dir); err != nil {
		return nil, WrapError("create temp file", dir, err, MemoryFS)
	}
	
	path := fs.resolvePath(dir)
	
	// Создаем уникальное имя файла
	tempName := fmt.Sprintf("%s%d", pattern, time.Now().UnixNano())
	tempPath := filepath.Join(path, tempName)
	
	// Создаем родительские директории если нужно
	if err := fs.mkdirAllLocked(path); err != nil {
		return nil, WrapError("create temp directories", path, err, MemoryFS)
	}
	
	// Создаем временный файл
	file := NewMemoryFile(tempPath, 0600)
	fs.files[tempPath] = file
	
	return file, nil
}

// Open открывает существующий файл
func (fs *MemoryFileSystem) Open(name string) (File, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	if err := ValidatePath(name); err != nil {
		return nil, WrapError("open file", name, err, MemoryFS)
	}
	
	path := fs.resolvePath(name)
	
	file, exists := fs.files[path]
	if !exists {
		return nil, WrapError("file not found", name, os.ErrNotExist, MemoryFS)
	}
	
	// Возвращаем копию файла для чтения
	return file.Clone(), nil
}

// OpenFile открывает файл с указанными флагами
func (fs *MemoryFileSystem) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	if err := ValidatePath(name); err != nil {
		return nil, WrapError("open file", name, err, MemoryFS)
	}
	
	path := fs.resolvePath(name)
	
	// Проверяем флаги
	if flag&os.O_CREATE != 0 {
		// Создаем родительские директории если нужно
		if err := fs.mkdirAllLocked(filepath.Dir(path)); err != nil {
			return nil, WrapError("create parent directories", filepath.Dir(path), err, MemoryFS)
		}
		
		file, exists := fs.files[path]
		if !exists {
			// Создаем новый файл
			file = NewMemoryFile(path, perm)
			fs.files[path] = file
		} else if flag&os.O_EXCL != 0 {
			return nil, WrapError("file already exists", name, os.ErrExist, MemoryFS)
		}
		
		if flag&os.O_TRUNC != 0 {
			if err := file.Truncate(0); err != nil {
				return nil, WrapError("failed to truncate file", name, err, MemoryFS)
			}
		}
		
		file.SetFlag(flag)
		// Возвращаем оригинальный файл для записи, клон только для чтения
		if flag&(os.O_WRONLY|os.O_RDWR) != 0 {
			return file, nil
		}
		return file.Clone(), nil
	}
	
	// Открываем существующий файл
	file, exists := fs.files[path]
	if !exists {
		return nil, WrapError("file not found", name, os.ErrNotExist, MemoryFS)
	}
	
	// Сначала "открываем" файл, затем применяем операции
	file.closed = false
	
	if flag&os.O_TRUNC != 0 {
		if err := file.Truncate(0); err != nil {
			return nil, WrapError("failed to truncate file", name, err, MemoryFS)
		}
	}
	
	file.SetFlag(flag)
	// Возвращаем оригинальный файл для записи, клон только для чтения
	if flag&(os.O_WRONLY|os.O_RDWR) != 0 {
		return file, nil
	}
	return file.Clone(), nil
}

// MkdirTemp создает временную директорию
func (fs *MemoryFileSystem) MkdirTemp(dir, pattern string) (string, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	if dir == "" {
		dir = "."
	}
	
	if err := ValidatePath(dir); err != nil {
		return "", WrapError("create temp directory", dir, err, MemoryFS)
	}
	
	path := fs.resolvePath(dir)
	
	// Создаем уникальное имя директории
	tempName := fmt.Sprintf("%s%d", pattern, time.Now().UnixNano())
	tempPath := filepath.Join(path, tempName)
	
	// Создаем родительские директории если нужно
	if err := fs.mkdirAllLocked(path); err != nil {
		return "", WrapError("create temp parent directories", path, err, MemoryFS)
	}
	
	// Создаем временную директорию
	fs.dirs[tempPath] = &MemoryDir{
		name:    tempName,
		mode:    0700,
		modTime: time.Now(),
		parent:  path,
	}
	
	// Возвращаем относительный путь
	relPath, err := filepath.Rel(fs.root, tempPath)
	if err != nil {
		return tempPath, nil
	}
	return relPath, nil
}

// MkdirAll создает директорию и все родительские директории
func (fs *MemoryFileSystem) MkdirAll(name string, _ os.FileMode) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	if err := ValidatePath(name); err != nil {
		return WrapError("mkdir all", name, err, MemoryFS)
	}
	
	path := fs.resolvePath(name)
	
	return fs.mkdirAllLocked(path)
}

// Remove удаляет файл или пустую директорию
func (fs *MemoryFileSystem) Remove(name string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	if err := ValidatePath(name); err != nil {
		return WrapError("remove", name, err, MemoryFS)
	}
	
	path := fs.resolvePath(name)
	
	// Проверяем существование файла
	if _, exists := fs.files[path]; exists {
		delete(fs.files, path)
		return nil
	}
	
	// Проверяем существование директории
	if _, exists := fs.dirs[path]; exists {
		// Проверяем, что директория пуста
		if !fs.isDirEmptyLocked(path) {
			return WrapError("directory not empty", name, fmt.Errorf("directory not empty"), MemoryFS)
		}
		delete(fs.dirs, path)
		return nil
	}
	
	return WrapError("not found", name, os.ErrNotExist, MemoryFS)
}

// RemoveAll удаляет файл или директорию рекурсивно
func (fs *MemoryFileSystem) RemoveAll(name string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	if err := ValidatePath(name); err != nil {
		return WrapError("remove all", name, err, MemoryFS)
	}
	
	path := fs.resolvePath(name)
	
	// Удаляем все файлы в директории
	for filePath := range fs.files {
		if strings.HasPrefix(filePath, path+string(filepath.Separator)) || filePath == path {
			delete(fs.files, filePath)
		}
	}
	
	// Удаляем все поддиректории
	for dirPath := range fs.dirs {
		if strings.HasPrefix(dirPath, path+string(filepath.Separator)) || dirPath == path {
			delete(fs.dirs, dirPath)
		}
	}
	
	return nil
}

// Rename переименовывает файл или директорию
func (fs *MemoryFileSystem) Rename(oldpath, newpath string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	if err := ValidatePath(oldpath); err != nil {
		return WrapError("rename source", oldpath, err, MemoryFS)
	}
	if err := ValidatePath(newpath); err != nil {
		return WrapError("rename destination", newpath, err, MemoryFS)
	}
	
	oldPath := fs.resolvePath(oldpath)
	newPath := fs.resolvePath(newpath)
	
	// Проверяем существование исходного файла
	if file, exists := fs.files[oldPath]; exists {
		// Создаем родительские директории для назначения
		if err := fs.mkdirAllLocked(filepath.Dir(newPath)); err != nil {
			return WrapError("create destination directories", filepath.Dir(newpath), err, MemoryFS)
		}
		
		// Перемещаем файл
		file.path = newPath
		fs.files[newPath] = file
		delete(fs.files, oldPath)
		return nil
	}
	
	// Проверяем существование исходной директории
	if dir, exists := fs.dirs[oldPath]; exists {
		// Создаем родительские директории для назначения
		if err := fs.mkdirAllLocked(filepath.Dir(newPath)); err != nil {
			return WrapError("create destination directories", filepath.Dir(newpath), err, MemoryFS)
		}
		
		// Перемещаем директорию
		dir.name = filepath.Base(newPath)
		dir.parent = filepath.Dir(newPath)
		fs.dirs[newPath] = dir
		delete(fs.dirs, oldPath)
		
		// Обновляем пути всех поддиректорий и файлов
		fs.updatePathsAfterRename(oldPath, newPath)
		return nil
	}
	
	return WrapError("source not found", oldpath, os.ErrNotExist, MemoryFS)
}

// ReadDir читает содержимое директории
func (fs *MemoryFileSystem) ReadDir(name string) ([]os.DirEntry, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	if err := ValidatePath(name); err != nil {
		return nil, WrapError("read directory", name, err, MemoryFS)
	}
	
	path := fs.resolvePath(name)
	
	// Проверяем, что это директория
	if _, exists := fs.dirs[path]; !exists {
		return nil, WrapError("directory not found", name, os.ErrNotExist, MemoryFS)
	}
	
	var entries []os.DirEntry
	
	// Добавляем поддиректории
	for dirPath, dir := range fs.dirs {
		if filepath.Dir(dirPath) == path && dirPath != path {
			entries = append(entries, &memoryDirEntry{
				name:    dir.name,
				isDir:   true,
				mode:    dir.mode,
				modTime: dir.modTime,
			})
		}
	}
	
	// Добавляем файлы
	for filePath, file := range fs.files {
		if filepath.Dir(filePath) == path {
			entries = append(entries, &memoryDirEntry{
				name:    filepath.Base(filePath),
				isDir:   false,
				mode:    file.mode,
				modTime: file.modTime,
				size:    int64(len(file.data)),
			})
		}
	}
	
	// Сортируем по имени
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	
	return entries, nil
}

// Getwd возвращает текущую рабочую директорию
func (fs *MemoryFileSystem) Getwd() (string, error) {
	return fs.root, nil
}

// Chdir изменяет текущую рабочую директорию
func (fs *MemoryFileSystem) Chdir(dir string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	if err := ValidatePath(dir); err != nil {
		return WrapError("change directory", dir, err, MemoryFS)
	}
	
	path := fs.resolvePath(dir)
	
	// Проверяем, что директория существует
	if _, exists := fs.dirs[path]; !exists {
		return WrapError("directory not found", dir, os.ErrNotExist, MemoryFS)
	}
	
	fs.root = path
	return nil
}

// ReadFile читает весь файл
func (fs *MemoryFileSystem) ReadFile(filename string) ([]byte, error) {
	file, err := fs.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
slog.Warn("failed to close memory file", slog.String("error", closeErr.Error()))
		}
	}()
	
	return io.ReadAll(file)
}

// WriteFile записывает данные в файл
func (fs *MemoryFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	file, err := fs.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
slog.Warn("failed to close memory file", slog.String("error", closeErr.Error()))
		}
	}()
	
	_, err = file.Write(data)
	return err
}

// Stat возвращает информацию о файле или директории
func (fs *MemoryFileSystem) Stat(name string) (os.FileInfo, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	if err := ValidatePath(name); err != nil {
		return nil, WrapError("stat", name, err, MemoryFS)
	}
	
	path := fs.resolvePath(name)
	
	// Проверяем файл
	if file, exists := fs.files[path]; exists {
		return &memoryFileInfo{
			name:    filepath.Base(path),
			size:    int64(len(file.data)),
			mode:    file.mode,
			modTime: file.modTime,
			isDir:   false,
		}, nil
	}
	
	// Проверяем директорию
	if dir, exists := fs.dirs[path]; exists {
		return &memoryFileInfo{
			name:    dir.name,
			size:    0,
			mode:    dir.mode | os.ModeDir,
			modTime: dir.modTime,
			isDir:   true,
		}, nil
	}
	
	return nil, WrapError("not found", name, os.ErrNotExist, MemoryFS)
}

// IsNotExist проверяет, является ли ошибка "файл не существует"
func (fs *MemoryFileSystem) IsNotExist(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}

// Chmod изменяет права доступа файла или директории
func (fs *MemoryFileSystem) Chmod(name string, mode os.FileMode) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	if err := ValidatePath(name); err != nil {
		return WrapError("chmod", name, err, MemoryFS)
	}
	
	path := fs.resolvePath(name)
	
	// Проверяем файл
	if file, exists := fs.files[path]; exists {
		file.mode = mode
		file.modTime = time.Now()
		return nil
	}
	
	// Проверяем директорию
	if dir, exists := fs.dirs[path]; exists {
		dir.mode = mode
		dir.modTime = time.Now()
		return nil
	}
	
	return WrapError("not found", name, os.ErrNotExist, MemoryFS)
}

// Chown изменяет владельца файла или директории (заглушка для совместимости)
func (fs *MemoryFileSystem) Chown(name string, _, _ int) error {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	path := fs.resolvePath(name)
	if err := ValidatePath(path); err != nil {
		return WrapError("chown", name, err, MemoryFS)
	}
	
	// Проверяем существование
	_, fileExists := fs.files[path]
	_, dirExists := fs.dirs[path]
	
	if !fileExists && !dirExists {
		return WrapError("not found", name, os.ErrNotExist, MemoryFS)
	}
	
	// В файловой системе в памяти операция chown не имеет смысла
	// Возвращаем nil для совместимости
	return nil
}

// Вспомогательные методы

// resolvePath разрешает относительный путь
func (fs *MemoryFileSystem) resolvePath(name string) string {
	if filepath.IsAbs(name) {
		return filepath.Clean(name)
	}
	return filepath.Join(fs.root, name)
}

// mkdirAllLocked создает все родительские директории (требует блокировки)
func (fs *MemoryFileSystem) mkdirAllLocked(path string) error {
	path = filepath.Clean(path)
	
	// Проверяем, что директория уже существует
	if _, exists := fs.dirs[path]; exists {
		return nil
	}
	
	// Создаем родительскую директорию рекурсивно
	parentDir := filepath.Dir(path)
	if parentDir != path && parentDir != "." {
		if err := fs.mkdirAllLocked(parentDir); err != nil {
			return err
		}
	}
	
	// Создаем текущую директорию
	fs.dirs[path] = &MemoryDir{
		name:    filepath.Base(path),
		mode:    0755,
		modTime: time.Now(),
		parent:  parentDir,
	}
	
	return nil
}

// isDirEmptyLocked проверяет, пуста ли директория (требует блокировки)
func (fs *MemoryFileSystem) isDirEmptyLocked(path string) bool {
	// Проверяем файлы
	for filePath := range fs.files {
		if filepath.Dir(filePath) == path {
			return false
		}
	}
	
	// Проверяем поддиректории
	for dirPath := range fs.dirs {
		if filepath.Dir(dirPath) == path && dirPath != path {
			return false
		}
	}
	
	return true
}

// updatePathsAfterRename обновляет пути после переименования директории
func (fs *MemoryFileSystem) updatePathsAfterRename(oldPath, newPath string) {
	// Обновляем пути файлов
	for filePath, file := range fs.files {
		if strings.HasPrefix(filePath, oldPath+string(filepath.Separator)) {
			newFilePath := strings.Replace(filePath, oldPath, newPath, 1)
			file.path = newFilePath
			fs.files[newFilePath] = file
			delete(fs.files, filePath)
		}
	}
	
	// Обновляем пути директорий
	for dirPath, dir := range fs.dirs {
		if strings.HasPrefix(dirPath, oldPath+string(filepath.Separator)) {
			newDirPath := strings.Replace(dirPath, oldPath, newPath, 1)
			dir.parent = filepath.Dir(newDirPath)
			fs.dirs[newDirPath] = dir
			delete(fs.dirs, dirPath)
		}
	}
}

// memoryDirEntry реализует os.DirEntry для файловой системы в памяти
type memoryDirEntry struct {
	name    string
	isDir   bool
	mode    os.FileMode
	modTime time.Time
	size    int64
}

func (e *memoryDirEntry) Name() string {
	return e.name
}

func (e *memoryDirEntry) IsDir() bool {
	return e.isDir
}

func (e *memoryDirEntry) Type() os.FileMode {
	if e.isDir {
		return os.ModeDir
	}
	return 0
}

func (e *memoryDirEntry) Info() (os.FileInfo, error) {
	return &memoryFileInfo{
		name:    e.name,
		size:    e.size,
		mode:    e.mode,
		modTime: e.modTime,
		isDir:   e.isDir,
	}, nil
}

// memoryFileInfo реализует os.FileInfo для файловой системы в памяти
type memoryFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (fi *memoryFileInfo) Name() string {
	return fi.name
}

func (fi *memoryFileInfo) Size() int64 {
	return fi.size
}

func (fi *memoryFileInfo) Mode() os.FileMode {
	return fi.mode
}

func (fi *memoryFileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi *memoryFileInfo) IsDir() bool {
	return fi.isDir
}

func (fi *memoryFileInfo) Sys() interface{} {
	return nil
}