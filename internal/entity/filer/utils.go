package filer

import (
	"github.com/Kargones/apk-ci/internal/constants"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// PathUtils предоставляет утилиты для работы с путями файловой системы.
type PathUtils struct{}

// NewPathUtils создает новый экземпляр PathUtils.
func NewPathUtils() *PathUtils {
	return &PathUtils{}
}

// NormalizePath нормализует путь, удаляя избыточные разделители и относительные компоненты.
// Возвращает абсолютный путь или ошибку при невалидном пути.
func (pu *PathUtils) NormalizePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("путь не может быть пустым")
	}

	// Проверка на недопустимые символы
	for _, char := range path {
		if !pu.isValidPathChar(char) {
			return "", fmt.Errorf("недопустимый символ в пути: %c", char)
		}
	}

	// Очистка пути от избыточных разделителей
	cleanPath := filepath.Clean(path)
	
	// Преобразование в абсолютный путь
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("не удалось получить абсолютный путь: %w", err)
	}

	return absPath, nil
}

// ValidatePath проверяет, является ли путь безопасным для использования.
// Проверяет на наличие опасных символов и path traversal атак.
func (pu *PathUtils) ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("путь не может быть пустым")
	}

	// Проверка на недопустимые символы
	for _, char := range path {
		if !pu.isValidPathChar(char) {
			return fmt.Errorf("недопустимый символ в пути: %c", char)
		}
	}

	// Проверка на path traversal
	if strings.Contains(path, "..") {
		return fmt.Errorf("обнаружена попытка path traversal в пути: %s", path)
	}

	// Проверка на абсолютные пути в относительном контексте
	if filepath.IsAbs(path) {
		return fmt.Errorf("абсолютные пути не разрешены: %s", path)
	}

	return nil
}

// isValidPathChar проверяет, является ли символ допустимым для использования в пути.
func (pu *PathUtils) isValidPathChar(char rune) bool {
	// Разрешенные символы: буквы, цифры, дефис, подчеркивание, точка, слеш, пробел
	return unicode.IsLetter(char) || unicode.IsDigit(char) || 
		char == '-' || char == '_' || char == '.' || char == '/' || char == '\\' || char == ' '
}

// JoinPath безопасно объединяет компоненты пути.
// Проверяет каждый компонент на безопасность перед объединением.
func (pu *PathUtils) JoinPath(base string, components ...string) (string, error) {
	if base == "" {
		return "", fmt.Errorf("базовый путь не может быть пустым")
	}

	// Валидация всех компонентов
	for i, component := range components {
		if err := pu.ValidatePath(component); err != nil {
			return "", fmt.Errorf("недопустимый компонент пути [%d]: %w", i, err)
		}
	}

	// Объединение путей
	allComponents := append([]string{base}, components...)
	result := filepath.Join(allComponents...)

	return pu.NormalizePath(result)
}

// EnsureDir создает директорию, если она не существует.
// Создает все промежуточные директории при необходимости.
func (pu *PathUtils) EnsureDir(path string) error {
	normalizedPath, err := pu.NormalizePath(path)
	if err != nil {
		return fmt.Errorf("не удалось нормализовать путь: %w", err)
	}

	// Проверка существования директории
	if info, err := os.Stat(normalizedPath); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("путь существует, но не является директорией: %s", normalizedPath)
		}
		return nil // Директория уже существует
	}

	// Создание директории с правами constants.DirPermStandard
	if err := os.MkdirAll(normalizedPath, constants.DirPermStandard); err != nil {
		return fmt.Errorf("не удалось создать директорию: %w", err)
	}

	return nil
}

// IsSubPath проверяет, является ли childPath подпутем parentPath.
// Используется для предотвращения выхода за пределы разрешенной директории.
func (pu *PathUtils) IsSubPath(parentPath, childPath string) (bool, error) {
	normalizedParent, err := pu.NormalizePath(parentPath)
	if err != nil {
		return false, fmt.Errorf("не удалось нормализовать родительский путь: %w", err)
	}

	normalizedChild, err := pu.NormalizePath(childPath)
	if err != nil {
		return false, fmt.Errorf("не удалось нормализовать дочерний путь: %w", err)
	}

	// Проверка, что дочерний путь начинается с родительского
	relPath, err := filepath.Rel(normalizedParent, normalizedChild)
	if err != nil {
		return false, fmt.Errorf("не удалось получить относительный путь: %w", err)
	}

	// Если относительный путь начинается с "..", то это не подпуть
	return !strings.HasPrefix(relPath, ".."), nil
}

// GetFileExtension возвращает расширение файла с точкой.
func (pu *PathUtils) GetFileExtension(filename string) string {
	if filename == "" || filename == "." {
		return ""
	}
	
	// Получаем базовое имя файла (без пути)
	base := filepath.Base(filename)
	
	// Если файл начинается с точки и не содержит других точек, это скрытый файл без расширения
	if strings.HasPrefix(base, ".") && strings.Count(base, ".") == 1 {
		return ""
	}
	
	// Если файл заканчивается точкой, расширения нет
	if strings.HasSuffix(base, ".") {
		return ""
	}
	
	return filepath.Ext(filename)
}

// GetBaseName возвращает имя файла без расширения.
func (pu *PathUtils) GetBaseName(filename string) string {
	if filename == "" {
		return ""
	}
	
	// Обработка Windows путей на Unix системах
	if strings.Contains(filename, "\\") {
		// Разделяем по обратным слешам и берем последний элемент
		parts := strings.Split(filename, "\\")
		filename = parts[len(parts)-1]
	}
	
	base := filepath.Base(filename)
	
	// Специальные случаи
	if base == "." {
		return "."
	}
	
	// Если файл начинается с точки и не содержит других точек, это скрытый файл без расширения
	if strings.HasPrefix(base, ".") && strings.Count(base, ".") == 1 {
		return base
	}
	
	// Если файл заканчивается точкой, убираем точку
	if strings.HasSuffix(base, ".") {
		return base[:len(base)-1]
	}
	
	// Обычная логика с расширением
	ext := filepath.Ext(base)
	if ext != "" {
		return base[:len(base)-len(ext)]
	}
	return base
}

// GetOptimalTempDir возвращает оптимальную директорию для временных файлов.
// Приоритет: RAM-диск (/dev/shm) -> системная временная директория.
func GetOptimalTempDir() string {
	if IsRAMDiskAvailable() {
		return RAMDiskPath //nolint:goconst // test value
	}
	return os.TempDir()
}

// IsRAMDiskAvailable проверяет доступность RAM-диска (/dev/shm).
// Возвращает true, если RAM-диск доступен для использования.
func IsRAMDiskAvailable() bool {
	// Проверка существования и доступности /dev/shm
	info, err := os.Stat(RAMDiskPath)
	if err != nil {
		return false
	}

	return info.IsDir()
}