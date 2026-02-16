package sonarqube

import (
	"testing"
)

// TestSonarQubeError проверяет функциональность Error
func TestSonarQubeError(t *testing.T) {
	tests := []struct {
		name     string
		error    *Error
		expected string
	}{
		{
			name: "error with details",
			error: &Error{
				Code:    400,
				Message: "Bad Request",
				Details: "Invalid parameter",
			},
			expected: "Bad Request: Invalid parameter",
		},
		{
			name: "error without details",
			error: &Error{
				Code:    500,
				Message: "Internal Server Error",
				Details: "",
			},
			expected: "Internal Server Error",
		},
		{
			name: "empty error",
			error: &Error{
				Code:    0,
				Message: "",
				Details: "",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.error.Error()
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestScannerError проверяет функциональность ScannerError
func TestScannerError(t *testing.T) {
	tests := []struct {
		name     string
		error    *ScannerError
		expected string
	}{
		{
			name: "scanner error with error message",
			error: &ScannerError{
				ExitCode: 1,
				Output:   "Some output",
				ErrorMsg: "Custom error message",
			},
			expected: "Custom error message",
		},
		{
			name: "scanner error without error message",
			error: &ScannerError{
				ExitCode: 2,
				Output:   "Some output",
				ErrorMsg: "",
			},
			expected: "Scanner execution failed with exit code \x02",
		},
		{
			name: "scanner error with zero exit code",
			error: &ScannerError{
				ExitCode: 0,
				Output:   "Success output",
				ErrorMsg: "",
			},
			expected: "Scanner execution failed with exit code \x00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.error.Error()
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestValidationError проверяет функциональность ValidationError
func TestValidationError(t *testing.T) {
	tests := []struct {
		name     string
		error    *ValidationError
		expected string
	}{
		{
			name: "validation error with all fields",
			error: &ValidationError{
				Field:   "projectKey",
				Value:   "invalid-key",
				Message: "Project key must contain only alphanumeric characters",
			},
			expected: "Project key must contain only alphanumeric characters",
		},
		{
			name: "validation error with empty message",
			error: &ValidationError{
				Field:   "url",
				Value:   "invalid-url",
				Message: "",
			},
			expected: "",
		},
		{
			name: "validation error with empty field and value",
			error: &ValidationError{
				Field:   "",
				Value:   "",
				Message: "General validation error",
			},
			expected: "General validation error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.error.Error()
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestErrorTypes проверяет, что все типы ошибок реализуют интерфейс error
func TestErrorTypes(t *testing.T) {
	var _ error = &Error{}
	var _ error = &ScannerError{}
	var _ error = &ValidationError{}
}