package filer

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"
)

// TempManager управляет временными директориями и файлами.
// Обеспечивает автоматическую очистку и безопасное управление ресурсами.
type TempManager struct {
	mu          sync.RWMutex
	tempDirs    map[string]*TempDir // Карта активных временных директорий
	baseDir     string              // Базовая директория для временных файлов
	useRAM      bool                // Использовать RAM-диск (/dev/shm)
	cleanupDone chan struct{}       // Канал для сигнализации завершения очистки
	pathUtils   *PathUtils          // Утилиты для работы с путями
}

// TempDir представляет временную директорию с метаданными.
type TempDir struct {
	Path      string    // Путь к временной директории
	CreatedAt time.Time // Время создания
	Prefix    string    // Префикс имени директории
	AutoClean bool      // Автоматическая очистка при завершении программы
}

// NewTempManager создает новый менеджер временных директорий.
func NewTempManager(useRAM bool) *TempManager {
	tm := &TempManager{
		tempDirs:    make(map[string]*TempDir),
		useRAM:      useRAM,
		cleanupDone: make(chan struct{}),
		pathUtils:   NewPathUtils(),
	}

	// Определение базовой директории
	tm.baseDir = tm.determineBaseDir()

	// Регистрация финализатора для автоматической очистки
	runtime.SetFinalizer(tm, (*TempManager).cleanup)

	return tm
}

// determineBaseDir определяет базовую директорию для временных файлов.
func (tm *TempManager) determineBaseDir() string {
	if tm.useRAM && tm.isRAMDiskAvailable() {
		return "/dev/shm"
	}
	return os.TempDir()
}

// isRAMDiskAvailable проверяет доступность RAM-диска.
func (tm *TempManager) isRAMDiskAvailable() bool {
	if runtime.GOOS != "linux" {
		return false
	}

	// Проверка существования и доступности /dev/shm
	info, err := os.Stat("/dev/shm")
	if err != nil {
		return false
	}

	return info.IsDir()
}

// CreateTempDir создает новую временную директорию.
func (tm *TempManager) CreateTempDir(prefix string, autoClean bool) (*TempDir, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Создание временной директории
	tempPath, err := os.MkdirTemp(tm.baseDir, prefix)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать временную директорию: %w", err)
	}

	// Создание объекта TempDir
	tempDir := &TempDir{
		Path:      tempPath,
		CreatedAt: time.Now(),
		Prefix:    prefix,
		AutoClean: autoClean,
	}

	// Регистрация в менеджере
	tm.tempDirs[tempPath] = tempDir

	return tempDir, nil
}

// CreateTempFile создает временный файл в указанной временной директории.
func (tm *TempManager) CreateTempFile(tempDir *TempDir, filename string) (string, error) {
	if tempDir == nil {
		return "", fmt.Errorf("временная директория не может быть nil")
	}

	// Валидация имени файла
	if err := tm.pathUtils.ValidatePath(filename); err != nil {
		return "", fmt.Errorf("недопустимое имя файла: %w", err)
	}

	// Создание полного пути к файлу
	filePath, err := tm.pathUtils.JoinPath(tempDir.Path, filename)
	if err != nil {
		return "", fmt.Errorf("не удалось создать путь к файлу: %w", err)
	}

	// Проверка, что файл находится внутри временной директории
	isSubPath, err := tm.pathUtils.IsSubPath(tempDir.Path, filePath)
	if err != nil || !isSubPath {
		return "", fmt.Errorf("файл должен находиться внутри временной директории")
	}

	// Создание файла
	// #nosec G304 - filePath уже валидирован выше
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("не удалось создать временный файл: %w", err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("не удалось закрыть временный файл: %w", err)
	}

	return filePath, nil
}

// RemoveTempDir удаляет временную директорию и все её содержимое.
func (tm *TempManager) RemoveTempDir(tempDir *TempDir) error {
	if tempDir == nil {
		return fmt.Errorf("временная директория не может быть nil")
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Удаление из реестра
	delete(tm.tempDirs, tempDir.Path)

	// Удаление директории с диска
	if err := os.RemoveAll(tempDir.Path); err != nil {
		return fmt.Errorf("не удалось удалить временную директорию: %w", err)
	}

	return nil
}

// GetTempDirs возвращает список всех активных временных директорий.
func (tm *TempManager) GetTempDirs() []*TempDir {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	dirs := make([]*TempDir, 0, len(tm.tempDirs))
	for _, dir := range tm.tempDirs {
		dirs = append(dirs, dir)
	}

	return dirs
}

// CleanupOldDirs удаляет временные директории старше указанного времени.
func (tm *TempManager) CleanupOldDirs(maxAge time.Duration) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	now := time.Now()
	var errors []error

	for path := range tm.tempDirs {
		// Проверяем время модификации директории из файловой системы
		stat, err := os.Stat(path)
		if err != nil {
			// Если директория не существует, удаляем из карты
			if os.IsNotExist(err) {
				delete(tm.tempDirs, path)
			}
			continue
		}

		// Используем время модификации из файловой системы
		if now.Sub(stat.ModTime()) > maxAge {
			if err := os.RemoveAll(path); err != nil {
				errors = append(errors, fmt.Errorf("не удалось удалить %s: %w", path, err))
			} else {
				delete(tm.tempDirs, path)
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("ошибки при очистке: %v", errors)
	}

	return nil
}

// CleanupAll удаляет все временные директории.
func (tm *TempManager) CleanupAll() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	var errors []error

	for path, dir := range tm.tempDirs {
		if dir.AutoClean {
			if err := os.RemoveAll(path); err != nil {
				errors = append(errors, fmt.Errorf("не удалось удалить %s: %w", path, err))
			}
		}
		delete(tm.tempDirs, path)
	}

	if len(errors) > 0 {
		return fmt.Errorf("ошибки при полной очистке: %v", errors)
	}

	return nil
}

// GetBaseDir возвращает базовую директорию для временных файлов.
func (tm *TempManager) GetBaseDir() string {
	return tm.baseDir
}

// IsUsingRAM возвращает true, если используется RAM-диск.
func (tm *TempManager) IsUsingRAM() bool {
	return tm.useRAM && tm.isRAMDiskAvailable()
}

// GetStats возвращает статистику по временным директориям.
func (tm *TempManager) GetStats() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	stats := map[string]interface{}{
		"total_dirs":    len(tm.tempDirs),
		"base_dir":      tm.baseDir,
		"using_ram":     tm.IsUsingRAM(),
		"auto_clean_dirs": 0,
		"total_size":    int64(0),
	}

	autoCleanCount := 0
	var totalSize int64
	for _, dir := range tm.tempDirs {
		if dir.AutoClean {
			autoCleanCount++
		}
		// Вычисляем размер директории
		if dirInfo, err := os.Stat(dir.Path); err == nil {
			totalSize += dirInfo.Size()
		}
	}
	stats["auto_clean_dirs"] = autoCleanCount
	stats["total_size"] = totalSize

	return stats
}

// cleanup выполняет финальную очистку при уничтожении объекта.
func (tm *TempManager) cleanup() {
	if err := tm.CleanupAll(); err != nil {
		// Логируем ошибку очистки, но не можем её вернуть из cleanup
		_ = err
	}
	close(tm.cleanupDone)
}

// Close закрывает менеджер и выполняет очистку.
func (tm *TempManager) Close() error {
	return tm.CleanupAll()
}

// GetWorkDir возвращает рабочую директорию для временных файлов.
// Если используется RAM-диск, возвращает путь к /dev/shm, иначе системную временную директорию.
func (tm *TempManager) GetWorkDir() string {
	return tm.baseDir
}

// CheckAvailableRAM проверяет доступный объем RAM и возможность использования RAM-диска.
// Возвращает информацию о доступности и рекомендации по использованию.
func (tm *TempManager) CheckAvailableRAM() map[string]interface{} {
	result := map[string]interface{}{
		"ram_disk_available": tm.isRAMDiskAvailable(),
		"using_ram":          tm.IsUsingRAM(),
		"base_dir":           tm.baseDir,
		"os":                 runtime.GOOS,
	}

	// Дополнительная проверка для Linux
	if runtime.GOOS == "linux" {
		if info, err := os.Stat("/dev/shm"); err == nil {
			result["shm_exists"] = true
			result["shm_is_dir"] = info.IsDir()
		} else {
			result["shm_exists"] = false
			result["shm_error"] = err.Error()
		}
	} else {
		result["fallback_reason"] = "RAM-диск доступен только на Linux"
	}

	return result
}