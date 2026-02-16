package filer

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Предопределенные ошибки модуля filer
var (
	// ErrInvalidPath возвращается при недопустимом пути
	ErrInvalidPath = errors.New("недопустимый путь")
	
	// ErrPathTraversal возвращается при попытке выхода за пределы файловой системы
	ErrPathTraversal = errors.New("попытка выхода за пределы файловой системы")
	
	// ErrFileSystemClosed возвращается при операции с закрытой файловой системой
	ErrFileSystemClosed = errors.New("файловая система закрыта")
	
	// ErrFileClosed возвращается при операции с закрытым файлом
	ErrFileClosed = errors.New("файл закрыт")
	
	// ErrReadOnlyFile возвращается при попытке записи в файл только для чтения
	ErrReadOnlyFile = errors.New("файл открыт только для чтения")
	
	// ErrUnsupportedOperation возвращается при неподдерживаемой операции
	ErrUnsupportedOperation = errors.New("неподдерживаемая операция")
	
	// ErrInsufficientPermissions возвращается при недостаточных правах доступа
	ErrInsufficientPermissions = errors.New("недостаточные права доступа")
	
	// ErrFileSystemNotFound возвращается при отсутствии файловой системы
	ErrFileSystemNotFound = errors.New("файловая система не найдена")
	
	// ErrInvalidConfig возвращается при недопустимой конфигурации
	ErrInvalidConfig = errors.New("недопустимая конфигурация")
	
	// ErrResourceExhausted возвращается при исчерпании ресурсов
	ErrResourceExhausted = errors.New("ресурсы исчерпаны")
)

// FileSystemError представляет ошибку файловой системы с дополнительным контекстом.
type FileSystemError struct {
	Op       string // Операция, которая вызвала ошибку
	Path     string // Путь, связанный с ошибкой
	Err      error  // Исходная ошибка
	FSType   FSType // Тип файловой системы
	Severity ErrorSeverity // Серьезность ошибки
}

// ErrorSeverity представляет уровень серьезности ошибки.
type ErrorSeverity int

const (
	// SeverityInfo - информационное сообщение
	SeverityInfo ErrorSeverity = iota
	// SeverityWarning - предупреждение
	SeverityWarning
	// SeverityError - ошибка
	SeverityError
	// SeverityCritical - критическая ошибка
	SeverityCritical
)

// String возвращает строковое представление уровня серьезности.
func (s ErrorSeverity) String() string {
	switch s {
	case SeverityInfo:
		return "INFO"
	case SeverityWarning:
		return "WARNING"
	case SeverityError:
		return "ERROR"
	case SeverityCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// Error реализует интерфейс error.
func (e *FileSystemError) Error() string {
	var parts []string
	
	if e.Op != "" {
		parts = append(parts, fmt.Sprintf("операция: %s", e.Op))
	}
	
	if e.Path != "" {
		parts = append(parts, fmt.Sprintf("путь: %s", e.Path))
	}
	
	if e.FSType != 0 {
		parts = append(parts, fmt.Sprintf("тип ФС: %s", e.FSType.String()))
	}
	
	if e.Severity != 0 {
		parts = append(parts, fmt.Sprintf("уровень: %s", e.Severity.String()))
	}
	
	if e.Err != nil {
		parts = append(parts, fmt.Sprintf("ошибка: %s", e.Err.Error()))
	}
	
	return strings.Join(parts, ", ")
}

// Unwrap возвращает исходную ошибку для поддержки errors.Is и errors.As.
func (e *FileSystemError) Unwrap() error {
	return e.Err
}

// NewFileSystemError создает новую ошибку файловой системы.
func NewFileSystemError(op, path string, err error, fsType FSType, severity ErrorSeverity) *FileSystemError {
	return &FileSystemError{
		Op:       op,
		Path:     path,
		Err:      err,
		FSType:   fsType,
		Severity: severity,
	}
}

// WrapError оборачивает ошибку в FileSystemError с дополнительным контекстом.
func WrapError(op, path string, err error, fsType FSType) error {
	if err == nil {
		return nil
	}
	
	// Определение серьезности на основе типа ошибки
	severity := determineSeverity(err)
	
	return NewFileSystemError(op, path, err, fsType, severity)
}

// determineSeverity определяет уровень серьезности ошибки.
func determineSeverity(err error) ErrorSeverity {
	if err == nil {
		return SeverityInfo
	}
	
	// Проверка на критические ошибки
	if errors.Is(err, ErrFileSystemClosed) || errors.Is(err, ErrResourceExhausted) {
		return SeverityCritical
	}
	
	// Проверка на ошибки безопасности
	if errors.Is(err, ErrPathTraversal) || errors.Is(err, ErrInsufficientPermissions) {
		return SeverityError
	}
	
	// Проверка на системные ошибки
	if os.IsNotExist(err) || os.IsExist(err) {
		return SeverityWarning
	}
	
	// По умолчанию - ошибка
	return SeverityError
}

// IsSecurityError проверяет, является ли ошибка связанной с безопасностью.
func IsSecurityError(err error) bool {
	if err == nil {
		return false
	}
	
	return errors.Is(err, ErrPathTraversal) ||
		errors.Is(err, ErrInsufficientPermissions) ||
		errors.Is(err, ErrInvalidPath)
}

// IsCriticalError проверяет, является ли ошибка критической.
func IsCriticalError(err error) bool {
	if err == nil {
		return false
	}
	
	var fsErr *FileSystemError
	if errors.As(err, &fsErr) {
		return fsErr.Severity == SeverityCritical
	}
	
	return errors.Is(err, ErrFileSystemClosed) ||
		errors.Is(err, ErrResourceExhausted)
}

// IsRetryableError проверяет, можно ли повторить операцию после ошибки.
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	
	// Критические ошибки и ошибки безопасности не подлежат повтору
	if IsCriticalError(err) || IsSecurityError(err) {
		return false
	}
	
	// Некоторые системные ошибки можно повторить
	if os.IsTimeout(err) {
		return true
	}
	
	// Проверяем временные ошибки
	if temp, ok := err.(interface{ Temporary() bool }); ok && temp.Temporary() {
		return true
	}
	
	return false
}

// ValidatePath проверяет безопасность пути.
func ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("%w: пустой путь", ErrInvalidPath)
	}
	
	// Проверка на попытки обхода директорий
	if strings.Contains(path, "..") {
		return fmt.Errorf("%w: обнаружен '..' в пути: %s", ErrPathTraversal, path)
	}
	
	// Проверка на абсолютные пути (в контексте виртуальной ФС)
	if strings.HasPrefix(path, "/") && path != "/" {
		return fmt.Errorf("%w: абсолютный путь не разрешен: %s", ErrInvalidPath, path)
	}
	
	// Проверка на недопустимые символы
	for _, char := range path {
		if char < 32 || char == 127 { // Управляющие символы
			return fmt.Errorf("%w: недопустимый символ в пути: %s", ErrInvalidPath, path)
		}
	}
	
	return nil
}

// ValidateConfig проверяет корректность конфигурации.
func ValidateConfig(config Config) error {
	switch config.Type {
	case DiskFS:
		if config.BasePath == "" {
			return fmt.Errorf("%w: базовый путь обязателен для DiskFS", ErrInvalidConfig)
		}
		// Дополнительная проверка базового пути
		if err := ValidatePath(config.BasePath); err != nil {
			return fmt.Errorf("%w: недопустимый базовый путь: %v", ErrInvalidConfig, err)
		}
	case MemoryFS:
		// Для MemoryFS особых требований нет
	default:
		return fmt.Errorf("%w: неподдерживаемый тип файловой системы: %s", ErrInvalidConfig, config.Type.String())
	}
	
	return nil
}

// SafeOperation выполняет операцию с обработкой паники.
func SafeOperation(op string, fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("паника в операции %s: %v", op, r)
		}
	}()
	
	return fn()
}

// ErrorHandler представляет обработчик ошибок.
type ErrorHandler interface {
	// HandleError обрабатывает ошибку
	HandleError(err error)
	// ShouldRetry определяет, следует ли повторить операцию
	ShouldRetry(err error) bool
}

// DefaultErrorHandler представляет обработчик ошибок по умолчанию.
type DefaultErrorHandler struct{}

// HandleError обрабатывает ошибку (по умолчанию - логирование).
func (h *DefaultErrorHandler) HandleError(_ error) {
	// В реальном приложении здесь должно быть логирование
	// Пока просто игнорируем
}

// ShouldRetry определяет, следует ли повторить операцию.
func (h *DefaultErrorHandler) ShouldRetry(err error) bool {
	return IsRetryableError(err)
}