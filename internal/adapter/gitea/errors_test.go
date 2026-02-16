package gitea_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Kargones/apk-ci/internal/adapter/gitea"
)

func TestGiteaError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      *gitea.GiteaError
		expected string
	}{
		{
			name: "без причины",
			err: &gitea.GiteaError{
				Code:    gitea.ErrGiteaNotFound,
				Message: "PR #42 не найден",
			},
			expected: "[GITEA.NOT_FOUND] PR #42 не найден",
		},
		{
			name: "с причиной",
			err: &gitea.GiteaError{
				Code:    gitea.ErrGiteaConnect,
				Message: "не удалось подключиться",
				Cause:   errors.New("connection refused"),
			},
			expected: "[GITEA.CONNECT_FAILED] не удалось подключиться: connection refused",
		},
		{
			name: "с HTTP статус кодом",
			err: &gitea.GiteaError{
				Code:       gitea.ErrGiteaAuth,
				Message:    "ошибка аутентификации",
				StatusCode: 401,
			},
			expected: "[GITEA.AUTH_FAILED] ошибка аутентификации",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGiteaError_Unwrap(t *testing.T) {
	t.Parallel()

	cause := errors.New("original error")
	err := &gitea.GiteaError{
		Code:    gitea.ErrGiteaAPI,
		Message: "API error",
		Cause:   cause,
	}

	if unwrapped := err.Unwrap(); unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}
}

func TestGiteaError_Unwrap_NilCause(t *testing.T) {
	t.Parallel()

	err := &gitea.GiteaError{
		Code:    gitea.ErrGiteaAPI,
		Message: "API error",
	}

	if unwrapped := err.Unwrap(); unwrapped != nil {
		t.Errorf("Unwrap() = %v, want nil", unwrapped)
	}
}

func TestNewGiteaError(t *testing.T) {
	t.Parallel()

	cause := errors.New("test cause")
	err := gitea.NewGiteaError(gitea.ErrGiteaTimeout, "timeout occurred", cause)

	if err.Code != gitea.ErrGiteaTimeout {
		t.Errorf("Code = %q, want %q", err.Code, gitea.ErrGiteaTimeout)
	}
	if err.Message != "timeout occurred" {
		t.Errorf("Message = %q, want %q", err.Message, "timeout occurred")
	}
	if err.Cause != cause {
		t.Errorf("Cause = %v, want %v", err.Cause, cause)
	}
}

func TestNewGiteaErrorWithStatus(t *testing.T) {
	t.Parallel()

	cause := errors.New("not found")
	err := gitea.NewGiteaErrorWithStatus(gitea.ErrGiteaNotFound, "resource not found", 404, cause)

	if err.Code != gitea.ErrGiteaNotFound {
		t.Errorf("Code = %q, want %q", err.Code, gitea.ErrGiteaNotFound)
	}
	if err.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want %d", err.StatusCode, 404)
	}
	if err.Cause != cause {
		t.Errorf("Cause = %v, want %v", err.Cause, cause)
	}
}

func TestValidationError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      *gitea.ValidationError
		expected string
	}{
		{
			name: "без причины",
			err: &gitea.ValidationError{
				Field:   "title",
				Message: "не может быть пустым",
			},
			expected: "[GITEA.VALIDATION_FAILED] поле 'title': не может быть пустым",
		},
		{
			name: "с причиной",
			err: &gitea.ValidationError{
				Field:   "branch",
				Message: "некорректный формат",
				Cause:   errors.New("invalid characters"),
			},
			expected: "[GITEA.VALIDATION_FAILED] поле 'branch': некорректный формат: invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestValidationError_Unwrap(t *testing.T) {
	t.Parallel()

	cause := errors.New("validation cause")
	err := &gitea.ValidationError{
		Field:   "head",
		Message: "invalid",
		Cause:   cause,
	}

	if unwrapped := err.Unwrap(); unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}
}

func TestNewValidationError(t *testing.T) {
	t.Parallel()

	err := gitea.NewValidationError("base", "обязательное поле")

	if err.Field != "base" {
		t.Errorf("Field = %q, want %q", err.Field, "base")
	}
	if err.Message != "обязательное поле" {
		t.Errorf("Message = %q, want %q", err.Message, "обязательное поле")
	}
	if err.Cause != nil {
		t.Errorf("Cause = %v, want nil", err.Cause)
	}
}

func TestNewValidationErrorWithCause(t *testing.T) {
	t.Parallel()

	cause := errors.New("parse error")
	err := gitea.NewValidationErrorWithCause("labels", "некорректный формат", cause)

	if err.Field != "labels" {
		t.Errorf("Field = %q, want %q", err.Field, "labels")
	}
	if err.Message != "некорректный формат" {
		t.Errorf("Message = %q, want %q", err.Message, "некорректный формат")
	}
	if err.Cause != cause {
		t.Errorf("Cause = %v, want %v", err.Cause, cause)
	}
}

func TestIsNotFoundError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "GiteaError с кодом NOT_FOUND",
			err:      gitea.NewGiteaError(gitea.ErrGiteaNotFound, "not found", nil),
			expected: true,
		},
		{
			name:     "GiteaError с другим кодом",
			err:      gitea.NewGiteaError(gitea.ErrGiteaAuth, "auth failed", nil),
			expected: false,
		},
		{
			name:     "wrapped GiteaError с кодом NOT_FOUND",
			err:      fmt.Errorf("context: %w", gitea.NewGiteaError(gitea.ErrGiteaNotFound, "wrapped", nil)),
			expected: true,
		},
		{
			name:     "обычная ошибка",
			err:      errors.New("regular error"),
			expected: false,
		},
		{
			name:     "nil",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := gitea.IsNotFoundError(tt.err); got != tt.expected {
				t.Errorf("IsNotFoundError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsAuthError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "GiteaError с кодом AUTH_FAILED",
			err:      gitea.NewGiteaError(gitea.ErrGiteaAuth, "auth failed", nil),
			expected: true,
		},
		{
			name:     "GiteaError с другим кодом",
			err:      gitea.NewGiteaError(gitea.ErrGiteaNotFound, "not found", nil),
			expected: false,
		},
		{
			name:     "wrapped GiteaError с кодом AUTH_FAILED",
			err:      fmt.Errorf("context: %w", gitea.NewGiteaError(gitea.ErrGiteaAuth, "wrapped", nil)),
			expected: true,
		},
		{
			name:     "обычная ошибка",
			err:      errors.New("regular error"),
			expected: false,
		},
		{
			name:     "nil",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := gitea.IsAuthError(tt.err); got != tt.expected {
				t.Errorf("IsAuthError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsTimeoutError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "GiteaError с кодом TIMEOUT",
			err:      gitea.NewGiteaError(gitea.ErrGiteaTimeout, "timeout", nil),
			expected: true,
		},
		{
			name:     "GiteaError с другим кодом",
			err:      gitea.NewGiteaError(gitea.ErrGiteaAPI, "api error", nil),
			expected: false,
		},
		{
			name:     "wrapped GiteaError с кодом TIMEOUT",
			err:      fmt.Errorf("context: %w", gitea.NewGiteaError(gitea.ErrGiteaTimeout, "wrapped", nil)),
			expected: true,
		},
		{
			name:     "обычная ошибка",
			err:      errors.New("regular error"),
			expected: false,
		},
		{
			name:     "nil",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := gitea.IsTimeoutError(tt.err); got != tt.expected {
				t.Errorf("IsTimeoutError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsConnectionError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "GiteaError с кодом CONNECT_FAILED",
			err:      gitea.NewGiteaError(gitea.ErrGiteaConnect, "connection failed", nil),
			expected: true,
		},
		{
			name:     "GiteaError с другим кодом",
			err:      gitea.NewGiteaError(gitea.ErrGiteaTimeout, "timeout", nil),
			expected: false,
		},
		{
			name:     "wrapped GiteaError с кодом CONNECT_FAILED",
			err:      fmt.Errorf("context: %w", gitea.NewGiteaError(gitea.ErrGiteaConnect, "wrapped", nil)),
			expected: true,
		},
		{
			name:     "обычная ошибка",
			err:      errors.New("regular error"),
			expected: false,
		},
		{
			name:     "nil",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := gitea.IsConnectionError(tt.err); got != tt.expected {
				t.Errorf("IsConnectionError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsAPIError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "GiteaError с кодом API_FAILED",
			err:      gitea.NewGiteaError(gitea.ErrGiteaAPI, "api failed", nil),
			expected: true,
		},
		{
			name:     "GiteaError с другим кодом",
			err:      gitea.NewGiteaError(gitea.ErrGiteaNotFound, "not found", nil),
			expected: false,
		},
		{
			name:     "wrapped GiteaError с кодом API_FAILED",
			err:      fmt.Errorf("context: %w", gitea.NewGiteaError(gitea.ErrGiteaAPI, "wrapped", nil)),
			expected: true,
		},
		{
			name:     "обычная ошибка",
			err:      errors.New("regular error"),
			expected: false,
		},
		{
			name:     "nil",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := gitea.IsAPIError(tt.err); got != tt.expected {
				t.Errorf("IsAPIError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestErrorCodes(t *testing.T) {
	t.Parallel()

	// Проверяем, что все коды ошибок уникальны и имеют корректный формат
	codes := []string{
		gitea.ErrGiteaConnect,
		gitea.ErrGiteaAPI,
		gitea.ErrGiteaAuth,
		gitea.ErrGiteaTimeout,
		gitea.ErrGiteaNotFound,
		gitea.ErrGiteaValidation,
	}

	seen := make(map[string]bool)
	for _, code := range codes {
		if seen[code] {
			t.Errorf("дублирующийся код ошибки: %s", code)
		}
		seen[code] = true

		// Проверяем формат GITEA.*
		if len(code) < 7 || code[:6] != "GITEA." {
			t.Errorf("код ошибки %q должен начинаться с 'GITEA.'", code)
		}
	}
}

func TestErrorsAs(t *testing.T) {
	t.Parallel()

	// Проверяем, что errors.As работает с GiteaError
	giteaErr := gitea.NewGiteaError(gitea.ErrGiteaAPI, "test", nil)
	wrappedErr := fmt.Errorf("wrapped: %w", giteaErr)

	var target *gitea.GiteaError
	if !errors.As(wrappedErr, &target) {
		t.Error("errors.As должен находить GiteaError в wrapped error")
	}
	if target.Code != gitea.ErrGiteaAPI {
		t.Errorf("Code = %q, want %q", target.Code, gitea.ErrGiteaAPI)
	}
}

func TestValidationErrorErrorsAs(t *testing.T) {
	t.Parallel()

	// Проверяем, что errors.As работает с ValidationError
	valErr := gitea.NewValidationError("field", "message")
	wrappedErr := fmt.Errorf("wrapped: %w", valErr)

	var target *gitea.ValidationError
	if !errors.As(wrappedErr, &target) {
		t.Error("errors.As должен находить ValidationError в wrapped error")
	}
	if target.Field != "field" {
		t.Errorf("Field = %q, want %q", target.Field, "field")
	}
}

func TestIsValidationError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "ValidationError",
			err:      gitea.NewValidationError("field", "invalid"),
			expected: true,
		},
		{
			name:     "ValidationError с cause",
			err:      gitea.NewValidationErrorWithCause("field", "invalid", errors.New("cause")),
			expected: true,
		},
		{
			name:     "wrapped ValidationError",
			err:      fmt.Errorf("context: %w", gitea.NewValidationError("field", "invalid")),
			expected: true,
		},
		{
			name:     "GiteaError (не ValidationError)",
			err:      gitea.NewGiteaError(gitea.ErrGiteaValidation, "validation", nil),
			expected: false,
		},
		{
			name:     "обычная ошибка",
			err:      errors.New("validation error"),
			expected: false,
		},
		{
			name:     "nil",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := gitea.IsValidationError(tt.err); got != tt.expected {
				t.Errorf("IsValidationError() = %v, want %v", got, tt.expected)
			}
		})
	}
}
