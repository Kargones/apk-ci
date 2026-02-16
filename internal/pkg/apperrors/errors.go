// Package apperrors предоставляет структурированные ошибки приложения.
// Переименован из errors чтобы избежать конфликта со стандартной библиотекой.
package apperrors

import "fmt"

// Коды ошибок в иерархическом формате: CATEGORY.SPECIFIC_ERROR.
// Позволяет grep по категориям: `grep "CONFIG\."` для всех config ошибок.
const (
	// Category: CONFIG — ошибки загрузки и парсинга конфигурации.
	ErrConfigLoad     = "CONFIG.LOAD_FAILED"
	ErrConfigParse    = "CONFIG.PARSE_FAILED"
	ErrConfigValidate = "CONFIG.VALIDATION_FAILED"

	// Category: COMMAND — ошибки выполнения команд.
	ErrCommandNotFound = "COMMAND.NOT_FOUND"
	ErrCommandExec     = "COMMAND.EXEC_FAILED"

	// Category: OUTPUT — ошибки форматирования вывода.
	ErrOutputFormat = "OUTPUT.FORMAT_FAILED"
)

// AppError представляет структурированную ошибку приложения.
// Реализует error interface и поддерживает wrapping через Unwrap().
//
// ВАЖНО: Message НЕ ДОЛЖЕН содержать секреты (пароли, токены, ключи).
// Используйте generic описания без конкретных значений.
//
// Пример использования:
//
//	return apperrors.NewAppError(apperrors.ErrConfigLoad,
//	    "не удалось загрузить конфигурацию из удалённого источника",
//	    err)
type AppError struct {
	// Code — машиночитаемый код ошибки в формате CATEGORY.SPECIFIC.
	Code string `json:"code"`

	// Message — человекочитаемое описание ошибки.
	// НЕ ДОЛЖЕН содержать секреты!
	Message string `json:"message"`

	// Cause — wrapped оригинальная ошибка.
	// Не сериализуется в JSON для безопасности (может содержать stack trace).
	Cause error `json:"-"`
}

// Error реализует интерфейс error.
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap возвращает wrapped ошибку для errors.Is/As.
func (e *AppError) Unwrap() error {
	return e.Cause
}

// NewAppError создаёт новый AppError с заданным кодом, сообщением и причиной.
//
// ВАЖНО: message НЕ ДОЛЖЕН содержать секреты!
func NewAppError(code, message string, cause error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}
