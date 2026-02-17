// Package filer предоставляет унифицированный интерфейс для работы с файловой системой,
// поддерживая как дисковое хранение, так и файловую систему в памяти.
package filer

import (
	"io"
	"os"
)

// FileReader defines read-only file system operations.
type FileReader interface {
	ReadFile(filename string) ([]byte, error)
	ReadDir(dirname string) ([]os.DirEntry, error)
	Open(name string) (File, error)
	Stat(name string) (os.FileInfo, error)
	IsNotExist(err error) bool
}

// FileWriter defines write/mutate file system operations.
type FileWriter interface {
	WriteFile(filename string, data []byte, perm os.FileMode) error
	Create(name string) (File, error)
	OpenFile(name string, flag int, perm os.FileMode) (File, error)
	Rename(oldpath, newpath string) error
	Remove(name string) error
	RemoveAll(path string) error
}

// DirManager defines directory management operations.
type DirManager interface {
	MkdirAll(path string, perm os.FileMode) error
	MkdirTemp(dir, pattern string) (string, error)
}

// TempFileManager defines temporary file operations.
type TempFileManager interface {
	CreateTemp(dir, pattern string) (File, error)
}

// PermissionManager defines file permission operations.
type PermissionManager interface {
	Chmod(name string, mode os.FileMode) error
	Chown(name string, uid, gid int) error
}

// WorkDirManager defines working directory operations.
type WorkDirManager interface {
	Getwd() (string, error)
	Chdir(dir string) error
}

// FileSystem определяет контракт для всех файловых операций.
// Интерфейс обеспечивает абстракцию над различными типами файловых систем
// и предоставляет единый API для файловых операций.
type FileSystem interface {
	FileReader
	FileWriter
	DirManager
	TempFileManager
	PermissionManager
	WorkDirManager
}

// File представляет интерфейс для работы с файловыми дескрипторами.
// Интерфейс объединяет стандартные интерфейсы Go для работы с файлами
// и добавляет дополнительные методы для управления файлами.
type File interface {
	// Встроенные интерфейсы для работы с потоками данных
	io.Reader
	io.Writer
	io.Closer
	io.Seeker
	
	// Name возвращает имя файла, как представлено в вызове Open.
	Name() string
	
	// Stat возвращает FileInfo, описывающий файл.
	Stat() (os.FileInfo, error)
	
	// Sync фиксирует текущее содержимое файла в стабильном хранилище.
	Sync() error
	
	// Truncate изменяет размер файла.
	Truncate(size int64) error
	
	// Chmod изменяет режим файла на mode.
	Chmod(mode os.FileMode) error
	
	// Chown изменяет числовые uid и gid файла.
	Chown(uid, gid int) error
}
