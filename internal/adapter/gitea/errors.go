package gitea

import (
	"errors"
	"fmt"

	"github.com/Kargones/apk-ci/internal/pkg/apperrors"
)

// Коды ошибок для Gitea операций.
const (
	// ErrGiteaConnect — ошибка подключения к серверу Gitea
	ErrGiteaConnect = "GITEA.CONNECT_FAILED"
	// ErrGiteaAPI — общая ошибка API Gitea
	ErrGiteaAPI = "GITEA.API_FAILED"
	// ErrGiteaAuth — ошибка аутентификации
	ErrGiteaAuth = "GITEA.AUTH_FAILED"
	// ErrGiteaTimeout — превышено время ожидания операции
	ErrGiteaTimeout = "GITEA.TIMEOUT"
	// ErrGiteaNotFound — ресурс не найден
	ErrGiteaNotFound = "GITEA.NOT_FOUND"
	// ErrGiteaValidation — ошибка валидации входных данных
	ErrGiteaValidation = "GITEA.VALIDATION_FAILED"
)

// GiteaError представляет ошибку при работе с Gitea API.
type GiteaError struct {
	// Code — код ошибки (одна из констант ErrGitea*)
	Code string
	// Message — человекочитаемое описание ошибки
	Message string
	// Cause — оригинальная ошибка (если есть)
	Cause error
	// StatusCode — HTTP статус код ответа (если применимо)
	StatusCode int
}

// Error реализует интерфейс error.
func (e *GiteaError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap возвращает оригинальную ошибку для использования с errors.Is/As.
func (e *GiteaError) Unwrap() error {
	return e.Cause
}

// NewGiteaError создаёт новую ошибку Gitea.
func NewGiteaError(code, message string, cause error) *GiteaError {
	return &GiteaError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// NewGiteaErrorWithStatus создаёт новую ошибку Gitea с HTTP статус кодом.
func NewGiteaErrorWithStatus(code, message string, statusCode int, cause error) *GiteaError {
	return &GiteaError{
		Code:       code,
		Message:    message,
		Cause:      cause,
		StatusCode: statusCode,
	}
}

// ValidationError представляет ошибку валидации входных данных.
type ValidationError struct {
	// Field — имя поля с ошибкой
	Field string
	// Message — описание ошибки
	Message string
	// Cause — оригинальная ошибка (если есть)
	Cause error
}

// Error реализует интерфейс error.
func (e *ValidationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] поле '%s': %s: %v", ErrGiteaValidation, e.Field, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] поле '%s': %s", ErrGiteaValidation, e.Field, e.Message)
}

// Unwrap возвращает оригинальную ошибку для использования с errors.Is/As.
func (e *ValidationError) Unwrap() error {
	return e.Cause
}

// NewValidationError создаёт новую ошибку валидации.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// NewValidationErrorWithCause создаёт новую ошибку валидации с оригинальной ошибкой.
func NewValidationErrorWithCause(field, message string, cause error) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
		Cause:   cause,
	}
}

// IsNotFoundError проверяет, является ли ошибка ошибкой "не найдено".
// Поддерживает wrapped errors через errors.As.
func IsNotFoundError(err error) bool {
	var giteaErr *GiteaError
	if errors.As(err, &giteaErr) {
		return giteaErr.Code == ErrGiteaNotFound
	}
	return false
}

// IsAuthError проверяет, является ли ошибка ошибкой аутентификации.
// Поддерживает wrapped errors через errors.As.
func IsAuthError(err error) bool {
	var giteaErr *GiteaError
	if errors.As(err, &giteaErr) {
		return giteaErr.Code == ErrGiteaAuth
	}
	return false
}

// IsTimeoutError проверяет, является ли ошибка ошибкой таймаута.
// Поддерживает wrapped errors через errors.As.
func IsTimeoutError(err error) bool {
	var giteaErr *GiteaError
	if errors.As(err, &giteaErr) {
		return giteaErr.Code == ErrGiteaTimeout
	}
	return false
}

// IsConnectionError проверяет, является ли ошибка ошибкой подключения.
// Поддерживает wrapped errors через errors.As.
func IsConnectionError(err error) bool {
	var giteaErr *GiteaError
	if errors.As(err, &giteaErr) {
		return giteaErr.Code == ErrGiteaConnect
	}
	return false
}

// IsAPIError проверяет, является ли ошибка общей ошибкой API.
// Поддерживает wrapped errors через errors.As.
func IsAPIError(err error) bool {
	var giteaErr *GiteaError
	if errors.As(err, &giteaErr) {
		return giteaErr.Code == ErrGiteaAPI
	}
	return false
}

// IsValidationError проверяет, является ли ошибка ошибкой валидации.
// Поддерживает wrapped errors через errors.As.
func IsValidationError(err error) bool {
	var valErr *ValidationError
	return errors.As(err, &valErr)
}

// ErrorCode возвращает машиночитаемый код ошибки.
// Реализует интерфейс apperrors.Coded.
func (e *GiteaError) ErrorCode() string {
	return e.Code
}

// As поддерживает преобразование GiteaError в apperrors.AppError через errors.As.
func (e *GiteaError) As(target interface{}) bool {
	if t, ok := target.(**apperrors.AppError); ok {
		*t = &apperrors.AppError{
			Code:    e.Code,
			Message: e.Message,
			Cause:   e.Cause,
		}
		return true
	}
	return false
}

// As поддерживает преобразование ValidationError в apperrors.AppError через errors.As.
func (e *ValidationError) As(target interface{}) bool {
	if t, ok := target.(**apperrors.AppError); ok {
		*t = &apperrors.AppError{
			Code:    ErrGiteaValidation,
			Message: fmt.Sprintf("поле '%s': %s", e.Field, e.Message),
			Cause:   e.Cause,
		}
		return true
	}
	return false
}

// ErrorCode возвращает машиночитаемый код ошибки валидации.
// Реализует интерфейс apperrors.Coded.
func (e *ValidationError) ErrorCode() string {
	return ErrGiteaValidation
}
