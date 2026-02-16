package sonarqube

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/entity/sonarqube"
)

func TestNewErrorHandlingService(t *testing.T) {
	maxRetries := 3
	retryDelay := 100 * time.Millisecond
	maxDelay := 1 * time.Second
	failureThreshold := 5
	successThreshold := 2
	timeout := 30 * time.Second

	service := NewErrorHandlingService(maxRetries, retryDelay, maxDelay, failureThreshold, successThreshold, timeout)

	if service.maxRetries != maxRetries {
		t.Errorf("Expected maxRetries %d, got %d", maxRetries, service.maxRetries)
	}
	if service.retryDelay != retryDelay {
		t.Errorf("Expected retryDelay %v, got %v", retryDelay, service.retryDelay)
	}
	if service.maxDelay != maxDelay {
		t.Errorf("Expected maxDelay %v, got %v", maxDelay, service.maxDelay)
	}
	if service.failureThreshold != failureThreshold {
		t.Errorf("Expected failureThreshold %d, got %d", failureThreshold, service.failureThreshold)
	}
	if service.successThreshold != successThreshold {
		t.Errorf("Expected successThreshold %d, got %d", successThreshold, service.successThreshold)
	}
	if service.timeout != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, service.timeout)
	}
	if service.currentState != Closed {
		t.Errorf("Expected initial state Closed, got %v", service.currentState)
	}
	if service.failureCount != 0 {
		t.Errorf("Expected initial failureCount 0, got %d", service.failureCount)
	}
	if service.successCount != 0 {
		t.Errorf("Expected initial successCount 0, got %d", service.successCount)
	}
}

func TestExecuteWithRetry_Success(t *testing.T) {
	service := NewErrorHandlingService(3, 10*time.Millisecond, 100*time.Millisecond, 5, 2, 30*time.Second)
	ctx := context.Background()

	callCount := 0
	fn := func() error {
		callCount++
		return nil
	}

	err := service.ExecuteWithRetry(ctx, fn)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected function to be called once, got %d", callCount)
	}
}

func TestExecuteWithRetry_SuccessAfterRetries(t *testing.T) {
	service := NewErrorHandlingService(3, 10*time.Millisecond, 100*time.Millisecond, 5, 2, 30*time.Second)
	ctx := context.Background()

	callCount := 0
	fn := func() error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	err := service.ExecuteWithRetry(ctx, fn)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if callCount != 3 {
		t.Errorf("Expected function to be called 3 times, got %d", callCount)
	}
}

func TestExecuteWithRetry_FailureAfterMaxRetries(t *testing.T) {
	service := NewErrorHandlingService(2, 10*time.Millisecond, 100*time.Millisecond, 5, 2, 30*time.Second)
	ctx := context.Background()

	callCount := 0
	expectedError := errors.New("persistent error")
	fn := func() error {
		callCount++
		return expectedError
	}

	err := service.ExecuteWithRetry(ctx, fn)

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if callCount != 3 { // maxRetries + 1
		t.Errorf("Expected function to be called 3 times, got %d", callCount)
	}
}

func TestExecuteWithRetry_ContextCancellation(t *testing.T) {
	service := NewErrorHandlingService(5, 100*time.Millisecond, 1*time.Second, 5, 2, 30*time.Second)
	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	fn := func() error {
		callCount++
		if callCount == 2 {
			cancel() // Cancel context after second call
		}
		return errors.New("error")
	}

	err := service.ExecuteWithRetry(ctx, fn)

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

func TestIsSonarQubeError(t *testing.T) {
	service := NewErrorHandlingService(3, 10*time.Millisecond, 100*time.Millisecond, 5, 2, 30*time.Second)

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "SonarQube error",
			err:      &sonarqube.Error{Code: 500, Message: "test error"},
			expected: true,
		},
		{
			name:     "Regular error",
			err:      errors.New("regular error"),
			expected: false,
		},
		{
			name:     "Scanner error",
			err:      &sonarqube.ScannerError{ErrorMsg: "scanner error"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.IsSonarQubeError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsScannerError(t *testing.T) {
	service := NewErrorHandlingService(3, 10*time.Millisecond, 100*time.Millisecond, 5, 2, 30*time.Second)

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Scanner error",
			err:      &sonarqube.ScannerError{ErrorMsg: "scanner error"},
			expected: true,
		},
		{
			name:     "Regular error",
			err:      errors.New("regular error"),
			expected: false,
		},
		{
			name:     "SonarQube error",
			err:      &sonarqube.Error{Code: 500, Message: "test error"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.IsScannerError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsValidationError(t *testing.T) {
	service := NewErrorHandlingService(3, 10*time.Millisecond, 100*time.Millisecond, 5, 2, 30*time.Second)

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Validation error",
			err:      &sonarqube.ValidationError{Field: "test", Message: "validation error"},
			expected: true,
		},
		{
			name:     "Regular error",
			err:      errors.New("regular error"),
			expected: false,
		},
		{
			name:     "SonarQube error",
			err:      &sonarqube.Error{Code: 500, Message: "test error"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.IsValidationError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestWrapError(t *testing.T) {
	service := NewErrorHandlingService(3, 10*time.Millisecond, 100*time.Millisecond, 5, 2, 30*time.Second)

	originalErr := errors.New("original error")
	context := "test context"

	wrappedErr := service.WrapError(originalErr, context)

	expectedMessage := "test context: original error"
	if wrappedErr.Error() != expectedMessage {
		t.Errorf("Expected wrapped error message '%s', got '%s'", expectedMessage, wrappedErr.Error())
	}

	// Check if the original error is wrapped
	if !errors.Is(wrappedErr, originalErr) {
		t.Error("Expected wrapped error to contain original error")
	}
}

func TestExecuteWithTimeout_Success(t *testing.T) {
	service := NewErrorHandlingService(3, 10*time.Millisecond, 100*time.Millisecond, 5, 2, 30*time.Second)
	ctx := context.Background()
	timeout := 100 * time.Millisecond

	fn := func() error {
		time.Sleep(10 * time.Millisecond) // Short operation
		return nil
	}

	err := service.ExecuteWithTimeout(ctx, timeout, fn)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestExecuteWithTimeout_Timeout(t *testing.T) {
	service := NewErrorHandlingService(3, 10*time.Millisecond, 100*time.Millisecond, 5, 2, 30*time.Second)
	ctx := context.Background()
	timeout := 50 * time.Millisecond

	fn := func() error {
		time.Sleep(200 * time.Millisecond) // Long operation
		return nil
	}

	err := service.ExecuteWithTimeout(ctx, timeout, fn)

	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}
}

func TestCircuitBreaker_OpenAndClose(t *testing.T) {
	service := NewErrorHandlingService(2, 10*time.Millisecond, 100*time.Millisecond, 2, 1, 50*time.Millisecond)
	ctx := context.Background()

	// Trigger failures to open circuit
	failingFn := func() error {
		return errors.New("failure")
	}

	// First two failures should work but fail
	for i := 0; i < 2; i++ {
		service.ExecuteWithRetry(ctx, failingFn)
	}

	// Circuit should now be open
	if service.currentState != Open {
		t.Errorf("Expected circuit state to be Open, got %v", service.currentState)
	}

	// Next call should fail immediately due to open circuit
	err := service.ExecuteWithRetry(ctx, failingFn)
	if err == nil {
		t.Error("Expected error due to open circuit, got nil")
	}

	// Wait for timeout to pass
	time.Sleep(60 * time.Millisecond)

	// Circuit should move to HalfOpen and allow one request
	successFn := func() error {
		return nil
	}

	err = service.ExecuteWithRetry(ctx, successFn)
	if err != nil {
		t.Errorf("Expected success after timeout, got %v", err)
	}

	// Circuit should now be closed
	if service.currentState != Closed {
		t.Errorf("Expected circuit state to be Closed, got %v", service.currentState)
	}
}

func TestCircuitBreaker_HalfOpenToOpen(t *testing.T) {
	service := NewErrorHandlingService(1, 10*time.Millisecond, 100*time.Millisecond, 1, 1, 50*time.Millisecond)
	ctx := context.Background()

	// Trigger failure to open circuit
	failingFn := func() error {
		return errors.New("failure")
	}

	service.ExecuteWithRetry(ctx, failingFn)

	// Circuit should be open
	if service.currentState != Open {
		t.Errorf("Expected circuit state to be Open, got %v", service.currentState)
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Try to execute failing function - should move from HalfOpen back to Open
	service.ExecuteWithRetry(ctx, failingFn)

	// Circuit should be open again
	if service.currentState != Open {
		t.Errorf("Expected circuit state to be Open after failure in HalfOpen, got %v", service.currentState)
	}
}