// Package sonarqube определяет интерфейсы и типы данных для работы с SonarQube API.
package sonarqube

import (
	"errors"
	"fmt"

	"github.com/Kargones/apk-ci/internal/pkg/apperrors"
)

// Коды ошибок для SonarQube операций.
const (
	// ErrSonarQubeConnect — ошибка подключения к серверу SonarQube
	ErrSonarQubeConnect = "SONARQUBE.CONNECT_FAILED"
	// ErrSonarQubeAPI — общая ошибка API SonarQube
	ErrSonarQubeAPI = "SONARQUBE.API_FAILED"
	// ErrSonarQubeAuth — ошибка аутентификации
	ErrSonarQubeAuth = "SONARQUBE.AUTH_FAILED"
	// ErrSonarQubeTimeout — превышено время ожидания операции
	ErrSonarQubeTimeout = "SONARQUBE.TIMEOUT"
	// ErrSonarQubeNotFound — проект или ресурс не найден
	ErrSonarQubeNotFound = "SONARQUBE.NOT_FOUND"
	// ErrSonarQubeValidation — ошибка валидации входных данных
	ErrSonarQubeValidation = "SONARQUBE.VALIDATION_FAILED"
)

// SonarQubeError представляет ошибку при работе с SonarQube API.
type SonarQubeError struct {
	// Code — код ошибки (одна из констант ErrSonarQube*)
	Code string
	// Message — человекочитаемое описание ошибки
	Message string
	// Cause — оригинальная ошибка (если есть)
	Cause error
	// StatusCode — HTTP статус код ответа (если применимо)
	StatusCode int
}

// Error реализует интерфейс error.
func (e *SonarQubeError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap возвращает оригинальную ошибку для использования с errors.Is/As.
func (e *SonarQubeError) Unwrap() error {
	return e.Cause
}

// NewSonarQubeError создаёт новую ошибку SonarQube.
func NewSonarQubeError(code, message string, cause error) *SonarQubeError {
	return &SonarQubeError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// NewSonarQubeErrorWithStatus создаёт новую ошибку SonarQube с HTTP статус кодом.
func NewSonarQubeErrorWithStatus(code, message string, statusCode int, cause error) *SonarQubeError {
	return &SonarQubeError{
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
		return fmt.Sprintf("[%s] поле '%s': %s: %v", ErrSonarQubeValidation, e.Field, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] поле '%s': %s", ErrSonarQubeValidation, e.Field, e.Message)
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
	var sqErr *SonarQubeError
	if errors.As(err, &sqErr) {
		return sqErr.Code == ErrSonarQubeNotFound
	}
	return false
}

// IsAuthError проверяет, является ли ошибка ошибкой аутентификации.
// Поддерживает wrapped errors через errors.As.
func IsAuthError(err error) bool {
	var sqErr *SonarQubeError
	if errors.As(err, &sqErr) {
		return sqErr.Code == ErrSonarQubeAuth
	}
	return false
}

// IsTimeoutError проверяет, является ли ошибка ошибкой таймаута.
// Поддерживает wrapped errors через errors.As.
func IsTimeoutError(err error) bool {
	var sqErr *SonarQubeError
	if errors.As(err, &sqErr) {
		return sqErr.Code == ErrSonarQubeTimeout
	}
	return false
}

// IsConnectionError проверяет, является ли ошибка ошибкой подключения.
// Поддерживает wrapped errors через errors.As.
func IsConnectionError(err error) bool {
	var sqErr *SonarQubeError
	if errors.As(err, &sqErr) {
		return sqErr.Code == ErrSonarQubeConnect
	}
	return false
}

// IsAPIError проверяет, является ли ошибка общей ошибкой API.
// Поддерживает wrapped errors через errors.As.
func IsAPIError(err error) bool {
	var sqErr *SonarQubeError
	if errors.As(err, &sqErr) {
		return sqErr.Code == ErrSonarQubeAPI
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
func (e *SonarQubeError) ErrorCode() string {
	return e.Code
}

// As поддерживает преобразование SonarQubeError в apperrors.AppError через errors.As.
func (e *SonarQubeError) As(target interface{}) bool {
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
			Code:    ErrSonarQubeValidation,
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
	return ErrSonarQubeValidation
}
