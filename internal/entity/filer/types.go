package filer

import "os"

// FSType представляет тип файловой системы.
type FSType int

const (
	// DiskFS представляет файловую систему на диске
	DiskFS FSType = iota
	// MemoryFS представляет файловую систему в памяти
	MemoryFS
)

// String возвращает строковое представление типа файловой системы.
func (t FSType) String() string {
	switch t {
	case DiskFS:
		return "DiskFS"
	case MemoryFS:
		return "MemoryFS"
	default:
		return "Unknown"
	}
}

// Config представляет конфигурацию для создания файловой системы.
type Config struct {
	// Type определяет тип файловой системы (DiskFS или MemoryFS)
	Type FSType
	
	// BasePath определяет базовый путь для дисковой файловой системы.
	// Если не указан, будет использован путь по умолчанию.
	BasePath string
	
	// UseRAM указывает, следует ли использовать RAM-диск (/dev/shm) для MemoryFS.
	// Применимо только для MemoryFS на Linux системах.
	UseRAM bool
}

// DefaultConfig возвращает конфигурацию по умолчанию.
func DefaultConfig() Config {
	return Config{
		Type:     DiskFS,
		BasePath: "", // Будет установлен автоматически
		UseRAM:   false,
	}
}

// Константы модуля
const (
	// DefaultDir - имя директории по умолчанию для файловых операций
	DefaultDir = "filer"
	
	// DefaultPerm - права доступа по умолчанию для создаваемых файлов и директорий
	DefaultPerm = 0700
	
	// RAMDiskPath - путь к RAM-диску на Linux системах
	RAMDiskPath = "/dev/shm"
	
	// RAMDiskThreshold - пороговое значение использования RAM (50% от доступной памяти)
	RAMDiskThreshold = 0.5
	
	// MinRAMSize - минимальный размер доступной RAM для использования RAM-диска (в байтах)
	MinRAMSize = 100 * 1024 * 1024 // 100 MB
	
	// MaxFileSize - максимальный размер файла в памяти (в байтах)
	MaxFileSize = 500 * 1024 * 1024 // 500 MB
	
	// TempDirPrefix - префикс для временных директорий
	TempDirPrefix = "filer_"
)

// FileMode константы для различных типов файлов
const (
	// DirMode - права доступа по умолчанию для директорий
	DirMode os.FileMode = 0755
	
	// FileMode - права доступа по умолчанию для файлов
	FileMode os.FileMode = 0644
	
	// TempMode - права доступа для временных файлов и директорий
	TempMode os.FileMode = 0700
)