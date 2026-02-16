// Package sonarqube provides implementation of error handling functionality.
// This package contains the implementation of typed error system,
// retry mechanisms, and circuit breaker patterns.
package sonarqube

import (
	"context"
	"fmt"
	"time"

	"github.com/Kargones/apk-ci/internal/entity/sonarqube"
)

// CircuitBreakerState represents the state of the circuit breaker.
type CircuitBreakerState int

const (
	// Closed state - requests are allowed
	Closed CircuitBreakerState = iota
	// Open state - requests are blocked
	Open
	// HalfOpen state - limited requests are allowed to test if service is back
	HalfOpen
)

// ErrorHandlingService provides functionality for error handling and resilience patterns.
// This service layer implements typed error system, retry mechanisms,
// and circuit breaker patterns.
type ErrorHandlingService struct {
	// maxRetries is the maximum number of retries for failed operations
	maxRetries int

	// retryDelay is the initial delay between retries
	retryDelay time.Duration

	// maxDelay is the maximum delay between retries
	maxDelay time.Duration

	// Circuit breaker fields
	// failureThreshold is the number of failures that will cause the circuit to open
	failureThreshold int

	// successThreshold is the number of successes needed to close the circuit
	successThreshold int

	// timeout is the time after which the circuit will try to close again
	timeout time.Duration

	// currentState is the current state of the circuit breaker
	currentState CircuitBreakerState

	// failureCount is the current number of consecutive failures
	failureCount int

	// successCount is the current number of consecutive successes
	successCount int

	// lastFailureTime is the time of the last failure
	lastFailureTime time.Time
}

// NewErrorHandlingService creates a new instance of ErrorHandlingService.
// This function initializes the service with the provided configuration.
//
// Parameters:
//   - maxRetries: maximum number of retries for failed operations
//   - retryDelay: initial delay between retries
//   - maxDelay: maximum delay between retries
//   - failureThreshold: number of failures that will cause the circuit to open
//   - successThreshold: number of successes needed to close the circuit
//   - timeout: time after which the circuit will try to close again
//
// Returns:
//   - *ErrorHandlingService: initialized error handling service
func NewErrorHandlingService(maxRetries int, retryDelay, maxDelay time.Duration,
	failureThreshold, successThreshold int, timeout time.Duration) *ErrorHandlingService {
	return &ErrorHandlingService{
		maxRetries:       maxRetries,
		retryDelay:       retryDelay,
		maxDelay:         maxDelay,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		timeout:          timeout,
		currentState:     Closed,
		failureCount:     0,
		successCount:     0,
	}
}

// ExecuteWithRetry executes the provided function with retry mechanism.
// This method implements exponential backoff retry mechanism for failed operations.
// It also integrates with the circuit breaker pattern.
//
// Parameters:
//   - ctx: context for the operation
//   - fn: function to execute
//
// Returns:
//   - error: error if all retries fail or circuit breaker is open
func (e *ErrorHandlingService) ExecuteWithRetry(ctx context.Context, fn func() error) error {
	// Check circuit breaker state
	if err := e.checkCircuitBreaker(); err != nil {
		return err
	}

	var lastErr error

	delay := e.retryDelay

	for i := 0; i <= e.maxRetries; i++ {
		// Execute the function
		err := fn()

		// If successful, update circuit breaker and return nil
		if err == nil {
			e.onSuccess()
			return nil
		}

		// Update circuit breaker with failure
		e.onFailure()

		// Store the last error
		lastErr = err

		// If this was the last attempt, break
		if i == e.maxRetries {
			break
		}

		// Wait for the delay or context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}

		// Increase delay exponentially, but cap it at maxDelay
		delay *= 2
		if delay > e.maxDelay {
			delay = e.maxDelay
		}
	}

	return fmt.Errorf("operation failed after %d retries: %w", e.maxRetries+1, lastErr)
}

// IsSonarQubeError checks if the provided error is a SonarQube error.
// This method checks if the error is of type SonarQubeError.
//
// Parameters:
//   - err: error to check
//
// Returns:
//   - bool: true if the error is a SonarQube error
func (e *ErrorHandlingService) IsSonarQubeError(err error) bool {
	_, ok := err.(*sonarqube.Error)
	return ok
}

// IsScannerError checks if the provided error is a scanner error.
// This method checks if the error is of type ScannerError.
//
// Parameters:
//   - err: error to check
//
// Returns:
//   - bool: true if the error is a scanner error
func (e *ErrorHandlingService) IsScannerError(err error) bool {
	_, ok := err.(*sonarqube.ScannerError)
	return ok
}

// IsValidationError checks if the provided error is a validation error.
// This method checks if the error is of type ValidationError.
//
// Parameters:
//   - err: error to check
//
// Returns:
//   - bool: true if the error is a validation error
func (e *ErrorHandlingService) IsValidationError(err error) bool {
	_, ok := err.(*sonarqube.ValidationError)
	return ok
}

// WrapError wraps the provided error with additional context.
// This method wraps the error with additional context information.
//
// Parameters:
//   - err: error to wrap
//   - context: context information
//
// Returns:
//   - error: wrapped error
func (e *ErrorHandlingService) WrapError(err error, context string) error {
	return fmt.Errorf("%s: %w", context, err)
}

// ExecuteWithTimeout executes the provided function with a timeout.
// This method executes the function with a timeout and handles context cancellation.
//
// Parameters:
//   - ctx: context for the operation
//   - timeout: timeout duration
//   - fn: function to execute
//
// Returns:
//   - error: error if the operation times out or is cancelled
func (e *ErrorHandlingService) ExecuteWithTimeout(ctx context.Context, timeout time.Duration, fn func() error) error {
	// Create a context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Channel to receive the result of the function
	resultChan := make(chan error, 1)

	// Execute the function in a goroutine
	go func() {
		resultChan <- fn()
	}()

	// Wait for either the function to complete or the context to be cancelled
	select {
	case err := <-resultChan:
		// Function completed
		return err
	case <-timeoutCtx.Done():
		// Timeout or cancellation
		return timeoutCtx.Err()
	}
}

// checkCircuitBreaker checks if the circuit breaker allows the operation to proceed.
// This method checks the current state of the circuit breaker and returns an error
// if the circuit is open.
//
// Returns:
//   - error: error if the circuit is open, nil otherwise
func (e *ErrorHandlingService) checkCircuitBreaker() error {
	switch e.currentState {
	case Open:
		// Check if timeout has passed
		if time.Since(e.lastFailureTime) >= e.timeout {
			// Move to HalfOpen state
			e.currentState = HalfOpen
			e.successCount = 0
			return nil
		}
		return &sonarqube.Error{
			Code:    503,
			Message: "Service unavailable due to circuit breaker",
			Details: "The circuit breaker is open, preventing further requests",
		}
	case HalfOpen:
		// In HalfOpen state, we allow limited requests
		// For simplicity, we'll allow one request at a time
		return nil
	default:
		// Closed state, allow requests
		return nil
	}
}

// onSuccess updates the circuit breaker state after a successful operation.
// This method should be called after a successful operation to update
// the circuit breaker state.
func (e *ErrorHandlingService) onSuccess() {
	switch e.currentState {
	case HalfOpen:
		e.successCount++
		if e.successCount >= e.successThreshold {
			// Close the circuit
			e.currentState = Closed
			e.failureCount = 0
		}
	default:
		// In Closed state, reset failure count
		e.failureCount = 0
	}
}

// onFailure updates the circuit breaker state after a failed operation.
// This method should be called after a failed operation to update
// the circuit breaker state.
func (e *ErrorHandlingService) onFailure() {
	e.failureCount++
	e.lastFailureTime = time.Now()

	switch e.currentState {
	case HalfOpen:
		// Move back to Open state
		e.currentState = Open
	default:
		if e.failureCount >= e.failureThreshold {
			// Open the circuit
			e.currentState = Open
		}
	}
}

// ToDo: необходимо дополнительно реализовать следующий функционал:
// - Add unit tests for all methods
// - Implement better error handling and recovery
// - Add progress reporting during operations
//
// Ссылки на пункты плана и требований:
// - tasks.md: 9.1, 9.3
// - requirements.md: 9.1, 9.3, 9.4
