package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNopLogger_ImplementsLogger проверяет что NopLogger реализует Logger interface.
func TestNopLogger_ImplementsLogger(t *testing.T) {
	nop := NewNopLogger()

	// Проверяем что underlying type — NopLogger
	_, ok := nop.(*NopLogger)
	assert.True(t, ok, "NewNopLogger должен возвращать *NopLogger")

	// Используем как Logger для проверки interface compliance
	useAsLogger(nop)
}

// useAsLogger — helper для compile-time проверки interface compliance.
func useAsLogger(l Logger) {
	_ = l
}

// TestNopLogger_AllMethods проверяет что все методы NopLogger не паникуют.
func TestNopLogger_AllMethods(t *testing.T) {
	logger := NewNopLogger()

	// Не должно быть паники
	assert.NotPanics(t, func() {
		logger.Debug("debug message", "key", "value")
	})
	assert.NotPanics(t, func() {
		logger.Info("info message", "key", "value")
	})
	assert.NotPanics(t, func() {
		logger.Warn("warn message", "key", "value")
	})
	assert.NotPanics(t, func() {
		logger.Error("error message", "key", "value")
	})
}

// TestNopLogger_With проверяет что With() возвращает NopLogger.
func TestNopLogger_With(t *testing.T) {
	logger := NewNopLogger()

	childLogger := logger.With("key", "value")

	// Должен вернуть Logger
	assert.NotNil(t, childLogger)

	// Должен быть тем же NopLogger (singleton behavior)
	_, ok := childLogger.(*NopLogger)
	assert.True(t, ok, "With() должен возвращать NopLogger")
}

// TestNopLogger_With_Chained проверяет цепочку With() вызовов.
func TestNopLogger_With_Chained(t *testing.T) {
	logger := NewNopLogger()

	// Цепочка With() не должна паниковать
	assert.NotPanics(t, func() {
		logger.With("a", 1).With("b", 2).With("c", 3).Info("message")
	})
}

// TestNewNopLogger проверяет что NewNopLogger() возвращает валидный logger.
func TestNewNopLogger(t *testing.T) {
	logger := NewNopLogger()
	assert.NotNil(t, logger)
}
