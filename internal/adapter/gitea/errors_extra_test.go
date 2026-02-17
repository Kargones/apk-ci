package gitea_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Kargones/apk-ci/internal/adapter/gitea"
	"github.com/Kargones/apk-ci/internal/pkg/apperrors"
)

// --- Tests for ErrorCode() and As() methods (apperrors integration) ---

func TestGiteaError_ErrorCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      *gitea.GiteaError
		expected string
	}{
		{
			name:     "CONNECT_FAILED",
			err:      &gitea.GiteaError{Code: gitea.ErrGiteaConnect},
			expected: gitea.ErrGiteaConnect,
		},
		{
			name:     "API_FAILED",
			err:      &gitea.GiteaError{Code: gitea.ErrGiteaAPI},
			expected: gitea.ErrGiteaAPI,
		},
		{
			name:     "AUTH_FAILED",
			err:      &gitea.GiteaError{Code: gitea.ErrGiteaAuth},
			expected: gitea.ErrGiteaAuth,
		},
		{
			name:     "TIMEOUT",
			err:      &gitea.GiteaError{Code: gitea.ErrGiteaTimeout},
			expected: gitea.ErrGiteaTimeout,
		},
		{
			name:     "NOT_FOUND",
			err:      &gitea.GiteaError{Code: gitea.ErrGiteaNotFound},
			expected: gitea.ErrGiteaNotFound,
		},
		{
			name:     "VALIDATION_FAILED",
			err:      &gitea.GiteaError{Code: gitea.ErrGiteaValidation},
			expected: gitea.ErrGiteaValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.err.ErrorCode(); got != tt.expected {
				t.Errorf("ErrorCode() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestValidationError_ErrorCode(t *testing.T) {
	t.Parallel()

	err := gitea.NewValidationError("field", "message")
	if got := err.ErrorCode(); got != gitea.ErrGiteaValidation {
		t.Errorf("ErrorCode() = %q, want %q", got, gitea.ErrGiteaValidation)
	}
}

func TestGiteaError_As_AppError(t *testing.T) {
	t.Parallel()

	cause := errors.New("original cause")
	giteaErr := &gitea.GiteaError{
		Code:    gitea.ErrGiteaNotFound,
		Message: "resource not found",
		Cause:   cause,
	}

	var appErr *apperrors.AppError
	if !errors.As(giteaErr, &appErr) {
		t.Fatal("errors.As должен конвертировать GiteaError в AppError")
	}

	if appErr.Code != gitea.ErrGiteaNotFound {
		t.Errorf("AppError.Code = %q, want %q", appErr.Code, gitea.ErrGiteaNotFound)
	}
	if appErr.Message != "resource not found" {
		t.Errorf("AppError.Message = %q, want %q", appErr.Message, "resource not found")
	}
	if appErr.Cause != cause {
		t.Errorf("AppError.Cause = %v, want %v", appErr.Cause, cause)
	}
}

func TestGiteaError_As_WrongType(t *testing.T) {
	t.Parallel()

	giteaErr := &gitea.GiteaError{
		Code:    gitea.ErrGiteaNotFound,
		Message: "not found",
	}

	// Попробуем преобразовать в ValidationError - должно быть false
	var valErr *gitea.ValidationError
	if errors.As(giteaErr, &valErr) {
		t.Error("errors.As должен вернуть false для GiteaError -> ValidationError")
	}
}

func TestValidationError_As_AppError(t *testing.T) {
	t.Parallel()

	cause := errors.New("parse error")
	valErr := &gitea.ValidationError{
		Field:   "title",
		Message: "cannot be empty",
		Cause:   cause,
	}

	var appErr *apperrors.AppError
	if !errors.As(valErr, &appErr) {
		t.Fatal("errors.As должен конвертировать ValidationError в AppError")
	}

	if appErr.Code != gitea.ErrGiteaValidation {
		t.Errorf("AppError.Code = %q, want %q", appErr.Code, gitea.ErrGiteaValidation)
	}
	// Message должен содержать поле
	if appErr.Message != "поле 'title': cannot be empty" {
		t.Errorf("AppError.Message = %q, want %q", appErr.Message, "поле 'title': cannot be empty")
	}
	if appErr.Cause != cause {
		t.Errorf("AppError.Cause = %v, want %v", appErr.Cause, cause)
	}
}

func TestValidationError_As_WrongType(t *testing.T) {
	t.Parallel()

	valErr := &gitea.ValidationError{
		Field:   "field",
		Message: "invalid",
	}

	// Попробуем преобразовать в GiteaError - должно быть false
	var giteaErr *gitea.GiteaError
	if errors.As(valErr, &giteaErr) {
		t.Error("errors.As должен вернуть false для ValidationError -> GiteaError")
	}
}

func TestGiteaError_WrappedError_ErrorCode(t *testing.T) {
	t.Parallel()

	// Проверяем что ErrorCode работает через errors.As
	innerErr := gitea.NewGiteaError(gitea.ErrGiteaTimeout, "timeout", nil)
	wrappedErr := fmt.Errorf("operation failed: %w", innerErr)

	// errors.As должен находить GiteaError
	var giteaErr *gitea.GiteaError
	if !errors.As(wrappedErr, &giteaErr) {
		t.Fatal("errors.As должен находить GiteaError в wrapped error")
	}

	if giteaErr.ErrorCode() != gitea.ErrGiteaTimeout {
		t.Errorf("ErrorCode() = %q, want %q", giteaErr.ErrorCode(), gitea.ErrGiteaTimeout)
	}
}
