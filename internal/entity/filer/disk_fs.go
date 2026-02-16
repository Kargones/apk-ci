package filer

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// DiskFileSystem реализует файловую систему на основе диска.
type DiskFileSystem struct {
	mu        sync.RWMutex
	basePath  string
	pathUtils *PathUtils
	config    Config
}

// Убеждаемся, что DiskFileSystem реализует интерфейс FileSystem
var _ FileSystem = (*DiskFileSystem)(nil)

// NewDiskFileSystem создает новую дисковую файловую систему.
func NewDiskFileSystem(config Config) (*DiskFileSystem, error) {
	pathUtils := NewPathUtils()

	basePath, err := pathUtils.NormalizePath(config.BasePath)
	if err != nil {
		return nil, fmt.Errorf("недопустимый базовый путь: %w", err)
	}

	if err := pathUtils.EnsureDir(basePath); err != nil {
		return nil, fmt.Errorf("не удалось создать базовую директорию: %w", err)
	}

	return &DiskFileSystem{
		basePath:  basePath,
		pathUtils: pathUtils,
		config:    config,
	}, nil
}

// getFullPath преобразует относительный путь в полный путь.
func (dfs *DiskFileSystem) getFullPath(relPath string) (string, error) {
	fullPath, err := dfs.pathUtils.JoinPath(dfs.basePath, relPath)
	if err != nil {
		return "", err
	}

	isSubPath, err := dfs.pathUtils.IsSubPath(dfs.basePath, fullPath)
	if err != nil || !isSubPath {
		return "", fmt.Errorf("путь выходит за пределы файловой системы: %s", relPath)
	}

	return fullPath, nil
}

// Mkdir создает новую директорию с указанным именем и правами доступа.
func (dfs *DiskFileSystem) Mkdir(name string, perm os.FileMode) error {
	dfs.mu.Lock()
	defer dfs.mu.Unlock()

	fullPath, err := dfs.getFullPath(name)
	if err != nil {
		return err
	}

	return os.Mkdir(fullPath, perm)
}

// MkdirTemp создает новую временную директорию в директории dir.
func (dfs *DiskFileSystem) MkdirTemp(dir, pattern string) (string, error) {
	dfs.mu.Lock()
	defer dfs.mu.Unlock()

	fullDir, err := dfs.getFullPath(dir)
	if err != nil {
		return "", err
	}

	tempDir, err := os.MkdirTemp(fullDir, pattern)
	if err != nil {
		return "", err
	}

	// Возвращаем относительный путь
	relPath, err := filepath.Rel(dfs.basePath, tempDir)
	if err != nil {
		return "", err
	}

	return relPath, nil
}

// MkdirAll создает директорию с именем path вместе с любыми необходимыми родительскими директориями.
func (dfs *DiskFileSystem) MkdirAll(path string, perm os.FileMode) error {
	dfs.mu.Lock()
	defer dfs.mu.Unlock()

	fullPath, err := dfs.getFullPath(path)
	if err != nil {
		return err
	}

	return os.MkdirAll(fullPath, perm)
}

// RemoveAll удаляет path и любые дочерние элементы, которые он содержит.
func (dfs *DiskFileSystem) RemoveAll(path string) error {
	dfs.mu.Lock()
	defer dfs.mu.Unlock()

	fullPath, err := dfs.getFullPath(path)
	if err != nil {
		return err
	}

	return os.RemoveAll(fullPath)
}

// ReadDir читает именованную директорию и возвращает список записей директории.
func (dfs *DiskFileSystem) ReadDir(dirname string) ([]os.DirEntry, error) {
	dfs.mu.RLock()
	defer dfs.mu.RUnlock()

	fullPath, err := dfs.getFullPath(dirname)
	if err != nil {
		return nil, err
	}

	return os.ReadDir(fullPath)
}

// Getwd возвращает корневое имя пути, соответствующее текущей директории.
func (dfs *DiskFileSystem) Getwd() (string, error) {
	return "/", nil // Возвращаем корень файловой системы
}

// Chdir изменяет текущую рабочую директорию на именованную директорию.
func (dfs *DiskFileSystem) Chdir(_ string) error {
	// В контексте виртуальной файловой системы это может быть no-op
	// или можно реализовать внутреннее состояние текущей директории
	return nil
}

// Create создает или обрезает именованный файл.
func (dfs *DiskFileSystem) Create(name string) (File, error) {
	dfs.mu.Lock()
	defer dfs.mu.Unlock()

	fullPath, err := dfs.getFullPath(name)
	if err != nil {
		return nil, err
	}

	// Создание директории для файла если необходимо
	dir := filepath.Dir(fullPath)
	if err := dfs.pathUtils.EnsureDir(dir); err != nil {
		return nil, err
	}

	file, createErr := os.Create(fullPath) // #nosec G304 - fullPath уже валидирован в getFullPath
	if createErr != nil {
		return nil, createErr
	}

	return file, nil
}

// CreateTemp создает новый временный файл в директории dir.
func (dfs *DiskFileSystem) CreateTemp(dir, pattern string) (File, error) {
	dfs.mu.Lock()
	defer dfs.mu.Unlock()

	fullDir, err := dfs.getFullPath(dir)
	if err != nil {
		return nil, err
	}

	return os.CreateTemp(fullDir, pattern)
}

// Open открывает именованный файл для чтения.
func (dfs *DiskFileSystem) Open(name string) (File, error) {
	dfs.mu.RLock()
	defer dfs.mu.RUnlock()

	fullPath, err := dfs.getFullPath(name)
	if err != nil {
		return nil, err
	}

	// #nosec G304 - fullPath уже валидирован в getFullPath
	return os.Open(fullPath)
}

// OpenFile является обобщенной функцией открытия.
func (dfs *DiskFileSystem) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	dfs.mu.RLock()
	defer dfs.mu.RUnlock()

	fullPath, err := dfs.getFullPath(name)
	if err != nil {
		return nil, err
	}

	// #nosec G304 - путь проверен в getFullPath на предмет выхода за пределы базовой директории
	return os.OpenFile(fullPath, flag, perm)
}

// Remove удаляет именованный файл или (пустую) директорию.
func (dfs *DiskFileSystem) Remove(name string) error {
	dfs.mu.Lock()
	defer dfs.mu.Unlock()

	fullPath, err := dfs.getFullPath(name)
	if err != nil {
		return err
	}

	return os.Remove(fullPath)
}

// Rename переименовывает (перемещает) oldpath в newpath.
func (dfs *DiskFileSystem) Rename(oldpath, newpath string) error {
	dfs.mu.Lock()
	defer dfs.mu.Unlock()

	fullOldPath, err := dfs.getFullPath(oldpath)
	if err != nil {
		return err
	}

	fullNewPath, err := dfs.getFullPath(newpath)
	if err != nil {
		return err
	}

	return os.Rename(fullOldPath, fullNewPath)
}

// ReadFile читает именованный файл и возвращает содержимое.
func (dfs *DiskFileSystem) ReadFile(filename string) ([]byte, error) {
	dfs.mu.RLock()
	defer dfs.mu.RUnlock()

	fullPath, err := dfs.getFullPath(filename)
	if err != nil {
		return nil, err
	}

	// #nosec G304 - путь проверен в getFullPath на предмет выхода за пределы базовой директории
	return os.ReadFile(fullPath)
}

// WriteFile записывает данные в именованный файл, создавая его при необходимости.
func (dfs *DiskFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	dfs.mu.Lock()
	defer dfs.mu.Unlock()

	fullPath, err := dfs.getFullPath(filename)
	if err != nil {
		return err
	}

	// Создание директории для файла если необходимо
	dir := filepath.Dir(fullPath)
	if err := dfs.pathUtils.EnsureDir(dir); err != nil {
		return err
	}

	return os.WriteFile(fullPath, data, perm)
}

// Stat возвращает FileInfo, описывающую именованный файл.
func (dfs *DiskFileSystem) Stat(name string) (os.FileInfo, error) {
	dfs.mu.RLock()
	defer dfs.mu.RUnlock()

	fullPath, err := dfs.getFullPath(name)
	if err != nil {
		return nil, err
	}

	return os.Stat(fullPath)
}

// IsNotExist возвращает булево значение, указывающее, известна ли ошибка как сообщающая о том, что файл или директория не существует.
func (dfs *DiskFileSystem) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

// Chmod изменяет режим именованного файла на mode.
func (dfs *DiskFileSystem) Chmod(name string, mode os.FileMode) error {
	dfs.mu.Lock()
	defer dfs.mu.Unlock()

	fullPath, err := dfs.getFullPath(name)
	if err != nil {
		return err
	}

	return os.Chmod(fullPath, mode)
}

// Chown изменяет числовые uid и gid именованного файла.
func (dfs *DiskFileSystem) Chown(name string, uid, gid int) error {
	dfs.mu.Lock()
	defer dfs.mu.Unlock()

	fullPath, err := dfs.getFullPath(name)
	if err != nil {
		return err
	}

	return os.Chown(fullPath, uid, gid)
}
