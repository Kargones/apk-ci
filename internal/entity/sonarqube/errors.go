// Package sonarqube provides error types for SonarQube integration.
// This package defines the error types that can occur during SonarQube operations,
// including API errors, scanner errors, and validation errors.
package sonarqube

import (
	"fmt"

	"github.com/Kargones/apk-ci/internal/pkg/apperrors"
)

// Error represents errors from SonarQube API.
// This error type is used when SonarQube API returns an error response.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details"`
}

// Error returns the error message for SonarQubeError.
func (e *Error) Error() string {
	if e.Details != "" {
		return e.Message + ": " + e.Details
	}
	return e.Message
}

// ScannerError represents errors from sonar-scanner execution.
// This error type is used when the sonar-scanner process fails.
type ScannerError struct {
	ExitCode int    `json:"exitCode"`
	Output   string `json:"output"`
	ErrorMsg string `json:"error"`
}

// Error returns the error message for ScannerError.
func (e *ScannerError) Error() string {
	if e.ErrorMsg != "" {
		return e.ErrorMsg
	}
	return "Scanner execution failed with exit code " + string(rune(e.ExitCode))
}

// ValidationError represents validation errors for SonarQube operations.
// This error type is used when input parameters or configuration fail validation.
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

// Error returns the error message for ValidationError.
func (e *ValidationError) Error() string {
	return e.Message
}

// ErrorCode возвращает машиночитаемый код ошибки.
// Реализует интерфейс apperrors.Coded.
func (e *Error) ErrorCode() string {
	return fmt.Sprintf("SONARQUBE.API_ERROR_%d", e.Code)
}

// Unwrap returns nil (Error does not wrap another error).
func (e *Error) Unwrap() error {
	return nil
}

// As поддерживает преобразование Error в apperrors.AppError через errors.As.
func (e *Error) As(target interface{}) bool {
	if t, ok := target.(**apperrors.AppError); ok {
		*t = &apperrors.AppError{
			Code:    e.ErrorCode(),
			Message: e.Message,
		}
		return true
	}
	return false
}

// ErrorCode возвращает машиночитаемый код ошибки сканера.
// Реализует интерфейс apperrors.Coded.
func (e *ScannerError) ErrorCode() string {
	return "SONARQUBE.SCANNER_FAILED"
}

// Unwrap returns nil (ScannerError does not wrap another error).
func (e *ScannerError) Unwrap() error {
	return nil
}

// As поддерживает преобразование ScannerError в apperrors.AppError через errors.As.
func (e *ScannerError) As(target interface{}) bool {
	if t, ok := target.(**apperrors.AppError); ok {
		*t = &apperrors.AppError{
			Code:    e.ErrorCode(),
			Message: e.Error(),
		}
		return true
	}
	return false
}

// ErrorCode возвращает машиночитаемый код ошибки валидации.
// Реализует интерфейс apperrors.Coded.
func (e *ValidationError) ErrorCode() string {
	return "SONARQUBE.VALIDATION_FAILED"
}

// Unwrap returns nil (ValidationError does not wrap another error).
func (e *ValidationError) Unwrap() error {
	return nil
}

// As поддерживает преобразование ValidationError в apperrors.AppError через errors.As.
func (e *ValidationError) As(target interface{}) bool {
	if t, ok := target.(**apperrors.AppError); ok {
		*t = &apperrors.AppError{
			Code:    e.ErrorCode(),
			Message: e.Message,
		}
		return true
	}
	return false
}
