package filer

import (
	"fmt"
	"log/slog"
)

// Factory представляет фабрику для создания файловых систем.
type Factory struct {
	tempManager *TempManager
}

// NewFactory создает новую фабрику файловых систем.
const defaultTmpDir = "/tmp"

func NewFactory(options ...Option) (*Factory, error) {
	config := DefaultConfig()
	
	// Применение опций
	for _, option := range options {
		option(&config)
	}

	// Создание менеджера временных директорий
	tempManager := NewTempManager(config.UseRAM)

	return &Factory{
		tempManager: tempManager,
	}, nil
}

// CreateFileSystem создает файловую систему указанного типа.
func (f *Factory) CreateFileSystem(fsType FSType, config Config) (FileSystem, error) {
	switch fsType {
	case DiskFS:
		return f.createDiskFileSystem(config)
	case MemoryFS:
		return f.createMemoryFileSystem(config)
	default:
		return nil, fmt.Errorf("неподдерживаемый тип файловой системы: %s", fsType)
	}
}

// CreateDiskFileSystem создает дисковую файловую систему с указанной конфигурацией.
func (f *Factory) CreateDiskFileSystem(config Config) (FileSystem, error) {
	return f.createDiskFileSystem(config)
}

// CreateMemoryFileSystem создает файловую систему в памяти с указанной конфигурацией.
func (f *Factory) CreateMemoryFileSystem(config Config) (FileSystem, error) {
	return f.createMemoryFileSystem(config)
}

// CreateTempFileSystem создает временную файловую систему.
func (f *Factory) CreateTempFileSystem(fsType FSType) (FileSystem, error) {
	tempDir, err := f.tempManager.CreateTempDir("filer_temp_", true)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать временную директорию: %w", err)
	}

	config := Config{
		Type:     fsType,
		BasePath: tempDir.Path,
		UseRAM:   f.tempManager.IsUsingRAM(),
	}

	fs, err := f.CreateFileSystem(fsType, config)
	if err != nil {
		// Очистка временной директории в случае ошибки
		if cleanupErr := f.tempManager.RemoveTempDir(tempDir); cleanupErr != nil {
slog.Warn("failed to cleanup temp dir", slog.String("error", cleanupErr.Error()))
		}
		return nil, err
	}

	return fs, nil
}

// GetTempManager возвращает менеджер временных директорий.
func (f *Factory) GetTempManager() *TempManager {
	return f.tempManager
}

// Cleanup очищает все ресурсы фабрики.
func (f *Factory) Cleanup() error {
	if f.tempManager != nil {
		return f.tempManager.CleanupAll()
	}
	return nil
}

// createDiskFileSystem создает дисковую файловую систему.
func (f *Factory) createDiskFileSystem(config Config) (FileSystem, error) {
	if config.BasePath == "" {
		return nil, fmt.Errorf("базовый путь не может быть пустым для дисковой файловой системы")
	}

	return NewDiskFileSystem(config)
}

// createMemoryFileSystem создает файловую систему в памяти.
func (f *Factory) createMemoryFileSystem(config Config) (FileSystem, error) {
	root := config.BasePath
	if root == "" {
		root = defaultTmpDir
	}
	
	return NewMemoryFileSystem(root), nil
}

// ValidateConfig проверяет корректность конфигурации для указанного типа файловой системы.
func (f *Factory) ValidateConfig(fsType FSType, config Config) error {
	switch fsType {
	case DiskFS:
		return f.validateDiskConfig(config)
	case MemoryFS:
		return f.validateMemoryConfig(config)
	default:
		return fmt.Errorf("неподдерживаемый тип файловой системы: %s", fsType)
	}
}

// validateDiskConfig проверяет конфигурацию для дисковой файловой системы.
func (f *Factory) validateDiskConfig(config Config) error {
	if config.BasePath == "" {
		return fmt.Errorf("базовый путь обязателен для дисковой файловой системы")
	}

	pathUtils := NewPathUtils()
	if err := pathUtils.ValidatePath(config.BasePath); err != nil {
		return fmt.Errorf("недопустимый базовый путь: %w", err)
	}

	return nil
}

// validateMemoryConfig проверяет конфигурацию для файловой системы в памяти.
func (f *Factory) validateMemoryConfig(_ Config) error {
	// Для файловой системы в памяти особых требований к конфигурации нет
	return nil
}

// GetSupportedTypes возвращает список поддерживаемых типов файловых систем.
func (f *Factory) GetSupportedTypes() []FSType {
	return []FSType{
		DiskFS,
		MemoryFS,
	}
}

// IsTypeSupported проверяет, поддерживается ли указанный тип файловой системы.
func (f *Factory) IsTypeSupported(fsType FSType) bool {
	supportedTypes := f.GetSupportedTypes()
	for _, supportedType := range supportedTypes {
		if supportedType == fsType {
			return true
		}
	}
	return false
}