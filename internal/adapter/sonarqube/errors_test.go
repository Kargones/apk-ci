package sonarqube

import (
	"errors"
	"testing"
)

// TestSonarQubeError_Error проверяет форматирование сообщения об ошибке.
func TestSonarQubeError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      *SonarQubeError
		expected string
	}{
		{
			name: "без причины",
			err: &SonarQubeError{
				Code:    ErrSonarQubeConnect,
				Message: "Не удалось подключиться к серверу",
			},
			expected: "[SONARQUBE.CONNECT_FAILED] Не удалось подключиться к серверу",
		},
		{
			name: "с причиной",
			err: &SonarQubeError{
				Code:    ErrSonarQubeAPI,
				Message: "Ошибка API",
				Cause:   errors.New("connection refused"),
			},
			expected: "[SONARQUBE.API_FAILED] Ошибка API: connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestSonarQubeError_Unwrap проверяет извлечение оригинальной ошибки.
func TestSonarQubeError_Unwrap(t *testing.T) {
	cause := errors.New("original error")
	err := &SonarQubeError{
		Code:    ErrSonarQubeConnect,
		Message: "Ошибка подключения",
		Cause:   cause,
	}

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}
}

// TestNewSonarQubeError проверяет конструктор ошибки.
func TestNewSonarQubeError(t *testing.T) {
	cause := errors.New("timeout")
	err := NewSonarQubeError(ErrSonarQubeTimeout, "Таймаут операции", cause)

	if err.Code != ErrSonarQubeTimeout {
		t.Errorf("Code = %q, want %q", err.Code, ErrSonarQubeTimeout)
	}
	if err.Message != "Таймаут операции" {
		t.Errorf("Message = %q, want %q", err.Message, "Таймаут операции")
	}
	if err.Cause != cause {
		t.Errorf("Cause = %v, want %v", err.Cause, cause)
	}
}

// TestNewSonarQubeErrorWithStatus проверяет конструктор ошибки с HTTP статусом.
func TestNewSonarQubeErrorWithStatus(t *testing.T) {
	err := NewSonarQubeErrorWithStatus(ErrSonarQubeAuth, "Неавторизован", 401, nil)

	if err.Code != ErrSonarQubeAuth {
		t.Errorf("Code = %q, want %q", err.Code, ErrSonarQubeAuth)
	}
	if err.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want %d", err.StatusCode, 401)
	}
}

// TestValidationError_Error проверяет форматирование ошибки валидации.
func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{
		Field:   "projectKey",
		Message: "не может быть пустым",
	}

	expected := "[SONARQUBE.VALIDATION_FAILED] поле 'projectKey': не может быть пустым"
	got := err.Error()
	if got != expected {
		t.Errorf("Error() = %q, want %q", got, expected)
	}
}

// TestNewValidationError проверяет конструктор ошибки валидации.
func TestNewValidationError(t *testing.T) {
	err := NewValidationError("name", "слишком короткое")

	if err.Field != "name" {
		t.Errorf("Field = %q, want %q", err.Field, "name")
	}
	if err.Message != "слишком короткое" {
		t.Errorf("Message = %q, want %q", err.Message, "слишком короткое")
	}
}

// TestIsNotFoundError проверяет определение ошибки "не найдено".
func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "ошибка not found",
			err:      &SonarQubeError{Code: ErrSonarQubeNotFound, Message: "Проект не найден"},
			expected: true,
		},
		{
			name:     "другая ошибка SonarQube",
			err:      &SonarQubeError{Code: ErrSonarQubeAPI, Message: "Ошибка API"},
			expected: false,
		},
		{
			name:     "обычная ошибка",
			err:      errors.New("some error"),
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
			got := IsNotFoundError(tt.err)
			if got != tt.expected {
				t.Errorf("IsNotFoundError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestIsAuthError проверяет определение ошибки аутентификации.
func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "ошибка auth",
			err:      &SonarQubeError{Code: ErrSonarQubeAuth, Message: "Неверный токен"},
			expected: true,
		},
		{
			name:     "другая ошибка SonarQube",
			err:      &SonarQubeError{Code: ErrSonarQubeConnect, Message: "Нет соединения"},
			expected: false,
		},
		{
			name:     "обычная ошибка",
			err:      errors.New("auth failed"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAuthError(tt.err)
			if got != tt.expected {
				t.Errorf("IsAuthError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestIsTimeoutError проверяет определение ошибки таймаута.
func TestIsTimeoutError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "ошибка timeout",
			err:      &SonarQubeError{Code: ErrSonarQubeTimeout, Message: "Превышено время ожидания"},
			expected: true,
		},
		{
			name:     "другая ошибка SonarQube",
			err:      &SonarQubeError{Code: ErrSonarQubeAPI, Message: "Ошибка API"},
			expected: false,
		},
		{
			name:     "обычная ошибка",
			err:      errors.New("timeout"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTimeoutError(tt.err)
			if got != tt.expected {
				t.Errorf("IsTimeoutError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestErrorsIs проверяет совместимость с errors.Is.
func TestErrorsIs(t *testing.T) {
	cause := errors.New("original")
	wrapped := NewSonarQubeError(ErrSonarQubeAPI, "API error", cause)

	if !errors.Is(wrapped, cause) {
		t.Error("errors.Is должен находить оригинальную ошибку через Unwrap")
	}
}

// TestIsConnectionError проверяет определение ошибки подключения.
func TestIsConnectionError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "ошибка connect",
			err:      &SonarQubeError{Code: ErrSonarQubeConnect, Message: "Сервер недоступен"},
			expected: true,
		},
		{
			name:     "другая ошибка SonarQube",
			err:      &SonarQubeError{Code: ErrSonarQubeAPI, Message: "Ошибка API"},
			expected: false,
		},
		{
			name:     "обычная ошибка",
			err:      errors.New("connection refused"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsConnectionError(tt.err)
			if got != tt.expected {
				t.Errorf("IsConnectionError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestIsErrorFunctionsWithWrappedErrors проверяет, что Is*Error функции
// работают с wrapped errors через errors.As.
func TestIsErrorFunctionsWithWrappedErrors(t *testing.T) {
	tests := []struct {
		name     string
		sqErr    *SonarQubeError
		checkFn  func(error) bool
		expected bool
	}{
		{
			name:     "wrapped NotFoundError",
			sqErr:    &SonarQubeError{Code: ErrSonarQubeNotFound, Message: "Not found"},
			checkFn:  IsNotFoundError,
			expected: true,
		},
		{
			name:     "wrapped AuthError",
			sqErr:    &SonarQubeError{Code: ErrSonarQubeAuth, Message: "Auth failed"},
			checkFn:  IsAuthError,
			expected: true,
		},
		{
			name:     "wrapped TimeoutError",
			sqErr:    &SonarQubeError{Code: ErrSonarQubeTimeout, Message: "Timeout"},
			checkFn:  IsTimeoutError,
			expected: true,
		},
		{
			name:     "wrapped ConnectionError",
			sqErr:    &SonarQubeError{Code: ErrSonarQubeConnect, Message: "Connect failed"},
			checkFn:  IsConnectionError,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Оборачиваем ошибку через fmt.Errorf
			wrapped := wrapError(tt.sqErr)

			got := tt.checkFn(wrapped)
			if got != tt.expected {
				t.Errorf("checkFn(wrapped) = %v, want %v", got, tt.expected)
			}
		})
	}
}

// wrapError оборачивает ошибку через fmt.Errorf для тестирования errors.As
func wrapError(err error) error {
	return &wrappedError{msg: "context", cause: err}
}

type wrappedError struct {
	msg   string
	cause error
}

func (e *wrappedError) Error() string { return e.msg + ": " + e.cause.Error() }
func (e *wrappedError) Unwrap() error { return e.cause }

// TestValidationErrorWithCause проверяет ValidationError с оригинальной ошибкой.
func TestValidationErrorWithCause(t *testing.T) {
	cause := errors.New("parse error")
	err := NewValidationErrorWithCause("projectKey", "неверный формат", cause)

	// Проверяем Error()
	expected := "[SONARQUBE.VALIDATION_FAILED] поле 'projectKey': неверный формат: parse error"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}

	// Проверяем Unwrap()
	if err.Unwrap() != cause {
		t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), cause)
	}

	// Проверяем errors.Is
	if !errors.Is(err, cause) {
		t.Error("errors.Is должен находить cause через Unwrap")
	}
}

// TestValidationErrorUnwrapNil проверяет Unwrap для ValidationError без cause.
func TestValidationErrorUnwrapNil(t *testing.T) {
	err := NewValidationError("field", "message")

	if err.Unwrap() != nil {
		t.Errorf("Unwrap() = %v, want nil", err.Unwrap())
	}
}

// TestValidationErrorAsCauseInSonarQubeError проверяет использование
// ValidationError как Cause в SonarQubeError.
func TestValidationErrorAsCauseInSonarQubeError(t *testing.T) {
	t.Parallel()

	valErr := NewValidationError("projectKey", "не может быть пустым")
	sqErr := NewSonarQubeError(ErrSonarQubeValidation, "Ошибка валидации", valErr)

	// Проверяем форматирование
	expected := "[SONARQUBE.VALIDATION_FAILED] Ошибка валидации: [SONARQUBE.VALIDATION_FAILED] поле 'projectKey': не может быть пустым"
	if sqErr.Error() != expected {
		t.Errorf("Error() = %q, want %q", sqErr.Error(), expected)
	}

	// Проверяем errors.As для извлечения ValidationError
	var extractedValErr *ValidationError
	if !errors.As(sqErr, &extractedValErr) {
		t.Error("errors.As должен извлекать ValidationError из SonarQubeError")
	}
	if extractedValErr.Field != "projectKey" {
		t.Errorf("Field = %q, want %q", extractedValErr.Field, "projectKey")
	}
}

// TestIsAPIError проверяет определение общей ошибки API.
func TestIsAPIError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "ошибка API",
			err:      &SonarQubeError{Code: ErrSonarQubeAPI, Message: "Ошибка API"},
			expected: true,
		},
		{
			name:     "другая ошибка SonarQube",
			err:      &SonarQubeError{Code: ErrSonarQubeConnect, Message: "Нет соединения"},
			expected: false,
		},
		{
			name:     "wrapped ошибка API",
			err:      wrapError(&SonarQubeError{Code: ErrSonarQubeAPI, Message: "API error"}),
			expected: true,
		},
		{
			name:     "обычная ошибка",
			err:      errors.New("api error"),
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
			got := IsAPIError(tt.err)
			if got != tt.expected {
				t.Errorf("IsAPIError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestIsValidationErrorFunc проверяет определение ошибки валидации.
func TestIsValidationErrorFunc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "ошибка валидации",
			err:      NewValidationError("field", "invalid"),
			expected: true,
		},
		{
			name:     "ошибка валидации с cause",
			err:      NewValidationErrorWithCause("field", "invalid", errors.New("cause")),
			expected: true,
		},
		{
			name:     "wrapped ошибка валидации",
			err:      wrapError(NewValidationError("field", "invalid")),
			expected: true,
		},
		{
			name:     "SonarQubeError (не ValidationError)",
			err:      &SonarQubeError{Code: ErrSonarQubeValidation, Message: "validation"},
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
			got := IsValidationError(tt.err)
			if got != tt.expected {
				t.Errorf("IsValidationError() = %v, want %v", got, tt.expected)
			}
		})
	}
}
