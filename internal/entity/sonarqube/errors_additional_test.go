package sonarqube

import (
	"errors"
	"testing"

	"github.com/Kargones/apk-ci/internal/pkg/apperrors"
)

// TestSonarQubeError_ErrorCode проверяет метод ErrorCode
func TestSonarQubeError_ErrorCode(t *testing.T) {
	tests := []struct {
		name     string
		error    *Error
		expected string
	}{
		{
			name: "error code 400",
			error: &Error{
				Code:    400,
				Message: "Bad Request",
			},
			expected: "SONARQUBE.API_ERROR_400",
		},
		{
			name: "error code 500",
			error: &Error{
				Code:    500,
				Message: "Internal Server Error",
			},
			expected: "SONARQUBE.API_ERROR_500",
		},
		{
			name: "error code 0",
			error: &Error{
				Code:    0,
				Message: "Unknown",
			},
			expected: "SONARQUBE.API_ERROR_0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.error.ErrorCode()
			if result != tt.expected {
				t.Errorf("ErrorCode() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestSonarQubeError_Unwrap проверяет метод Unwrap
func TestSonarQubeError_Unwrap(t *testing.T) {
	err := &Error{
		Code:    400,
		Message: "Bad Request",
	}

	unwrapped := err.Unwrap()
	if unwrapped != nil {
		t.Errorf("Unwrap() = %v, want nil", unwrapped)
	}
}

// TestSonarQubeError_As проверяет метод As
func TestSonarQubeError_As(t *testing.T) {
	err := &Error{
		Code:    400,
		Message: "Bad Request",
		Details: "Invalid parameter",
	}

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Error("errors.As should convert Error to AppError")
	}

	if appErr.Code != "SONARQUBE.API_ERROR_400" {
		t.Errorf("AppError.Code = %q, want %q", appErr.Code, "SONARQUBE.API_ERROR_400")
	}

	if appErr.Message != "Bad Request" {
		t.Errorf("AppError.Message = %q, want %q", appErr.Message, "Bad Request")
	}
}

// TestScannerError_ErrorCode проверяет метод ErrorCode для ScannerError
func TestScannerError_ErrorCode(t *testing.T) {
	err := &ScannerError{
		ExitCode: 1,
		Output:   "Some output",
		ErrorMsg: "Custom error message",
	}

	expected := "SONARQUBE.SCANNER_FAILED"
	result := err.ErrorCode()
	if result != expected {
		t.Errorf("ErrorCode() = %q, want %q", result, expected)
	}
}

// TestScannerError_Unwrap проверяет метод Unwrap для ScannerError
func TestScannerError_Unwrap(t *testing.T) {
	err := &ScannerError{
		ExitCode: 1,
		Output:   "Some output",
		ErrorMsg: "Custom error message",
	}

	unwrapped := err.Unwrap()
	if unwrapped != nil {
		t.Errorf("Unwrap() = %v, want nil", unwrapped)
	}
}

// TestScannerError_As проверяет метод As для ScannerError
func TestScannerError_As(t *testing.T) {
	err := &ScannerError{
		ExitCode: 1,
		Output:   "Some output",
		ErrorMsg: "Custom error message",
	}

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Error("errors.As should convert ScannerError to AppError")
	}

	if appErr.Code != "SONARQUBE.SCANNER_FAILED" {
		t.Errorf("AppError.Code = %q, want %q", appErr.Code, "SONARQUBE.SCANNER_FAILED")
	}
}

// TestValidationError_ErrorCode проверяет метод ErrorCode для ValidationError
func TestValidationError_ErrorCode(t *testing.T) {
	err := &ValidationError{
		Field:   "projectKey",
		Value:   "invalid-key",
		Message: "Project key must contain only alphanumeric characters",
	}

	expected := "SONARQUBE.VALIDATION_FAILED"
	result := err.ErrorCode()
	if result != expected {
		t.Errorf("ErrorCode() = %q, want %q", result, expected)
	}
}

// TestValidationError_Unwrap проверяет метод Unwrap для ValidationError
func TestValidationError_Unwrap(t *testing.T) {
	err := &ValidationError{
		Field:   "projectKey",
		Value:   "invalid-key",
		Message: "Invalid value",
	}

	unwrapped := err.Unwrap()
	if unwrapped != nil {
		t.Errorf("Unwrap() = %v, want nil", unwrapped)
	}
}

// TestValidationError_As проверяет метод As для ValidationError
func TestValidationError_As(t *testing.T) {
	err := &ValidationError{
		Field:   "projectKey",
		Value:   "invalid-key",
		Message: "Invalid value",
	}

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Error("errors.As should convert ValidationError to AppError")
	}

	if appErr.Code != "SONARQUBE.VALIDATION_FAILED" {
		t.Errorf("AppError.Code = %q, want %q", appErr.Code, "SONARQUBE.VALIDATION_FAILED")
	}

	if appErr.Message != "Invalid value" {
		t.Errorf("AppError.Message = %q, want %q", appErr.Message, "Invalid value")
	}
}

// TestError_As_DirectMethod проверяет метод As напрямую (не через errors.As)
func TestError_As_DirectMethod(t *testing.T) {
	err := &Error{
		Code:    400,
		Message: "Bad Request",
	}

	// Тестируем As напрямую с правильным типом
	var appErr *apperrors.AppError
	result := err.As(&appErr)
	if !result {
		t.Error("As() should return true for **apperrors.AppError target")
	}

	// Тестируем As напрямую с неправильным типом
	var wrongTarget *Error
	result = err.As(&wrongTarget)
	if result {
		t.Error("As() should return false for wrong target type")
	}
}

// TestScannerError_As_DirectMethod проверяет метод As напрямую
func TestScannerError_As_DirectMethod(t *testing.T) {
	err := &ScannerError{
		ExitCode: 1,
		Output:   "Some output",
		ErrorMsg: "Custom error message",
	}

	// Тестируем As напрямую с правильным типом
	var appErr *apperrors.AppError
	result := err.As(&appErr)
	if !result {
		t.Error("As() should return true for **apperrors.AppError target")
	}

	// Тестируем As напрямую с неправильным типом
	var wrongTarget *ScannerError
	result = err.As(&wrongTarget)
	if result {
		t.Error("As() should return false for wrong target type")
	}
}

// TestValidationError_As_DirectMethod проверяет метод As напрямую
func TestValidationError_As_DirectMethod(t *testing.T) {
	err := &ValidationError{
		Field:   "projectKey",
		Value:   "invalid-key",
		Message: "Invalid value",
	}

	// Тестируем As напрямую с правильным типом
	var appErr *apperrors.AppError
	result := err.As(&appErr)
	if !result {
		t.Error("As() should return true for **apperrors.AppError target")
	}

	// Тестируем As напрямую с неправильным типом
	var wrongTarget *ValidationError
	result = err.As(&wrongTarget)
	if result {
		t.Error("As() should return false for wrong target type")
	}
}
