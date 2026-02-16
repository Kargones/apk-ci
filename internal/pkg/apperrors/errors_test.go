package apperrors

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorCodeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"ErrConfigLoad", ErrConfigLoad, "CONFIG.LOAD_FAILED"},
		{"ErrConfigParse", ErrConfigParse, "CONFIG.PARSE_FAILED"},
		{"ErrConfigValidate", ErrConfigValidate, "CONFIG.VALIDATION_FAILED"},
		{"ErrCommandNotFound", ErrCommandNotFound, "COMMAND.NOT_FOUND"},
		{"ErrCommandExec", ErrCommandExec, "COMMAND.EXEC_FAILED"},
		{"ErrOutputFormat", ErrOutputFormat, "OUTPUT.FORMAT_FAILED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant)
		})
	}
}

func TestAppError_Error_WithCause(t *testing.T) {
	cause := errors.New("оригинальная ошибка")
	appErr := &AppError{
		Code:    ErrConfigLoad,
		Message: "не удалось загрузить конфигурацию",
		Cause:   cause,
	}

	expected := "CONFIG.LOAD_FAILED: не удалось загрузить конфигурацию (оригинальная ошибка)"
	assert.Equal(t, expected, appErr.Error())
}

func TestAppError_Error_WithoutCause(t *testing.T) {
	appErr := &AppError{
		Code:    ErrConfigLoad,
		Message: "не удалось загрузить конфигурацию",
		Cause:   nil,
	}

	expected := "CONFIG.LOAD_FAILED: не удалось загрузить конфигурацию"
	assert.Equal(t, expected, appErr.Error())
}

func TestAppError_Unwrap(t *testing.T) {
	cause := errors.New("оригинальная ошибка")
	appErr := &AppError{
		Code:    ErrConfigLoad,
		Message: "не удалось загрузить конфигурацию",
		Cause:   cause,
	}

	unwrapped := appErr.Unwrap()
	assert.Equal(t, cause, unwrapped)
}

func TestAppError_Unwrap_NilCause(t *testing.T) {
	appErr := &AppError{
		Code:    ErrConfigLoad,
		Message: "не удалось загрузить конфигурацию",
		Cause:   nil,
	}

	unwrapped := appErr.Unwrap()
	assert.Nil(t, unwrapped)
}

func TestAppError_ErrorsIs(t *testing.T) {
	cause := errors.New("оригинальная ошибка")
	appErr := &AppError{
		Code:    ErrConfigLoad,
		Message: "не удалось загрузить конфигурацию",
		Cause:   cause,
	}

	// errors.Is должен найти wrapped ошибку
	assert.True(t, errors.Is(appErr, cause))
}

func TestAppError_ImplementsError(_ *testing.T) {
	var _ error = (*AppError)(nil)
}

func TestNewAppError(t *testing.T) {
	cause := errors.New("оригинальная ошибка")
	appErr := NewAppError(ErrConfigLoad, "не удалось загрузить конфигурацию", cause)

	require.NotNil(t, appErr)
	assert.Equal(t, ErrConfigLoad, appErr.Code)
	assert.Equal(t, "не удалось загрузить конфигурацию", appErr.Message)
	assert.Equal(t, cause, appErr.Cause)
}

func TestNewAppError_NilCause(t *testing.T) {
	appErr := NewAppError(ErrConfigLoad, "не удалось загрузить конфигурацию", nil)

	require.NotNil(t, appErr)
	assert.Equal(t, ErrConfigLoad, appErr.Code)
	assert.Equal(t, "не удалось загрузить конфигурацию", appErr.Message)
	assert.Nil(t, appErr.Cause)
}

func TestAppError_JSON_Serialization(t *testing.T) {
	cause := errors.New("оригинальная ошибка")
	appErr := NewAppError(ErrConfigLoad, "не удалось загрузить конфигурацию", cause)

	data, err := json.Marshal(appErr)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	// Проверяем что json теги работают (lowercase keys)
	assert.Equal(t, ErrConfigLoad, parsed["code"])
	assert.Equal(t, "не удалось загрузить конфигурацию", parsed["message"])

	// Cause не должен сериализоваться (json:"-")
	_, hasCause := parsed["cause"]
	assert.False(t, hasCause, "Cause не должен сериализоваться в JSON")
	_, hasCauseUpper := parsed["Cause"]
	assert.False(t, hasCauseUpper, "Cause не должен сериализоваться в JSON")
}
