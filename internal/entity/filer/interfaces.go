// Package filer предоставляет унифицированный интерфейс для работы с файловой системой,
// поддерживая как дисковое хранение, так и файловую систему в памяти.
package filer

import (
	"io"
	"os"
)

// FileSystem определяет контракт для всех файловых операций.
// Интерфейс обеспечивает абстракцию над различными типами файловых систем
// и предоставляет единый API для файловых операций.
type FileSystem interface {
	// Операции с директориями
	
	// MkdirTemp создает новую временную директорию в указанной директории
	// с именем, начинающимся с pattern, и возвращает путь к новой директории.
	MkdirTemp(dir, pattern string) (string, error)
	
	// MkdirAll создает директорию с именем path вместе со всеми необходимыми
	// родительскими директориями и возвращает nil или первую встреченную ошибку.
	MkdirAll(path string, perm os.FileMode) error
	
	// RemoveAll удаляет path и все содержимое, которое он содержит.
	RemoveAll(path string) error
	
	// ReadDir читает именованную директорию и возвращает список записей директории,
	// отсортированных по имени файла.
	ReadDir(dirname string) ([]os.DirEntry, error)
	
	// Getwd возвращает корневое имя пути, соответствующее текущей директории.
	Getwd() (string, error)
	
	// Chdir изменяет текущую рабочую директорию на именованную директорию.
	Chdir(dir string) error
	
	// Операции с файлами
	
	// Create создает или обрезает именованный файл.
	Create(name string) (File, error)
	
	// CreateTemp создает новый временный файл в директории dir с именем,
	// начинающимся с pattern, открывает файл для чтения и записи.
	CreateTemp(dir, pattern string) (File, error)
	
	// Open открывает именованный файл для чтения.
	Open(name string) (File, error)
	
	// OpenFile является обобщенной функцией открытия; большинство пользователей
	// будут использовать Open или Create вместо этого.
	OpenFile(name string, flag int, perm os.FileMode) (File, error)
	
	// Remove удаляет именованный файл или (пустую) директорию.
	Remove(name string) error
	
	// Rename переименовывает (перемещает) oldpath в newpath.
	Rename(oldpath, newpath string) error
	
	// Операции чтения/записи
	
	// ReadFile читает именованный файл и возвращает содержимое.
	ReadFile(filename string) ([]byte, error)
	
	// WriteFile записывает данные в именованный файл, создавая его при необходимости.
	WriteFile(filename string, data []byte, perm os.FileMode) error
	
	// Информационные операции
	
	// Stat возвращает FileInfo, описывающий именованный файл.
	Stat(name string) (os.FileInfo, error)
	
	// IsNotExist возвращает булево значение, указывающее, известно ли,
	// что ошибка сообщает о том, что файл или директория не существует.
	IsNotExist(err error) bool
	
	// Операции с правами доступа
	
	// Chmod изменяет режим именованного файла на mode.
	Chmod(name string, mode os.FileMode) error
	
	// Chown изменяет числовые uid и gid именованного файла.
	Chown(name string, uid, gid int) error
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