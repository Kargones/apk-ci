package sonarqube

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/entity/sonarqube"
)

// TestNewBranchScannerService тестирует создание нового сервиса
func TestNewBranchScannerService(t *testing.T) {
	branchScanner := &sonarqube.BranchScannerEntity{}
	config := &config.ScannerConfig{
		WorkDir: "/tmp/test",
		TempDir: "/tmp",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	service := NewBranchScannerService(branchScanner, config, logger)

	if service == nil {
		t.Errorf("Expected service to be created")
		return
	}
	if service.branchScanner != branchScanner {
		t.Errorf("Expected branch scanner to be set")
	}
	if service.config != config {
		t.Errorf("Expected config to be set")
	}
	if service.logger != logger {
		t.Errorf("Expected logger to be set")
	}
	if service.retryConfig == nil {
		t.Errorf("Expected retry config to be initialized")
	}
}

// TestBranchScannerService_ValidateBranchForScanning тестирует валидацию ветки
func TestBranchScannerService_ValidateBranchForScanning(t *testing.T) {
	service := createTestService()
	ctx := context.Background()

	tests := []struct {
		name        string
		branchName  string
		projectPath string
		expectError bool
	}{
		{
			name:        "valid branch and path",
			branchName:  "main",
			projectPath: "/tmp/test-repo",
			expectError: false,
		},
		{
			name:        "empty branch name",
			branchName:  "",
			projectPath: "/tmp/test-repo",
			expectError: true,
		},
		{
			name:        "empty project path",
			branchName:  "main",
			projectPath: "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateBranchForScanning(ctx, tt.branchName, tt.projectPath)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestBranchScannerService_validateScanRequest тестирует валидацию запроса
func TestBranchScannerService_validateScanRequest(t *testing.T) {
	service := createTestService()

	tests := []struct {
		name        string
		request     *BranchScanRequest
		expectError bool
	}{
		{
			name: "valid request",
			request: &BranchScanRequest{
				ProjectKey:  "test/project",
				ProjectName: "Test Project",
				ProjectPath: "/tmp/test",
				BranchName:  "main",
				Owner:       "testowner",
				Repository:  "testrepo",
			},
			expectError: false,
		},
		{
			name:        "nil request",
			request:     nil,
			expectError: true,
		},
		{
			name: "empty project key",
			request: &BranchScanRequest{
				ProjectKey:  "",
				ProjectName: "Test Project",
				ProjectPath: "/tmp/test",
				BranchName:  "main",
				Owner:       "testowner",
				Repository:  "testrepo",
			},
			expectError: true,
		},
		{
			name: "empty branch name",
			request: &BranchScanRequest{
				ProjectKey:  "test/project",
				ProjectName: "Test Project",
				ProjectPath: "/tmp/test",
				BranchName:  "",
				Owner:       "testowner",
				Repository:  "testrepo",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateScanRequest(tt.request)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestBranchScannerService_isRetryableError тестирует определение повторяемых ошибок
func TestBranchScannerService_isRetryableError(t *testing.T) {
	service := createTestService()

	tests := []struct {
		name     string
		errorMsg string
		expected bool
	}{
		{
			name:     "connection error",
			errorMsg: "connection refused",
			expected: true,
		},
		{
			name:     "timeout error",
			errorMsg: "request timeout",
			expected: true,
		},
		{
			name:     "network error",
			errorMsg: "network error occurred",
			expected: true,
		},
		{
			name:     "validation error",
			errorMsg: "validation failed",
			expected: false,
		},
		{
			name:     "authentication error",
			errorMsg: "authentication failed",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &testError{message: tt.errorMsg}
			result := service.isRetryableError(err)
			if result != tt.expected {
				t.Errorf("Expected %v for error '%s', got %v", tt.expected, tt.errorMsg, result)
			}
		})
	}
}

// TestBranchScannerService_calculateRetryDelay тестирует расчет задержки повтора
func TestBranchScannerService_calculateRetryDelay(t *testing.T) {
	service := createTestService()

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{
			name:     "first attempt",
			attempt:  1,
			expected: 1 * time.Second,
		},
		{
			name:     "second attempt",
			attempt:  2,
			expected: 2 * time.Second,
		},
		{
			name:     "third attempt",
			attempt:  3,
			expected: 4 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculateRetryDelay(tt.attempt)
			if result != tt.expected {
				t.Errorf("Expected %v for attempt %d, got %v", tt.expected, tt.attempt, result)
			}
		})
	}
}

// TestBranchScannerService_GetBranchScanHistory тестирует получение истории сканирования
func TestBranchScannerService_GetBranchScanHistory(t *testing.T) {
	service := createTestService()
	ctx := context.Background()

	result, err := service.GetBranchScanHistory(ctx, "test/project", "main")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result == nil {
		t.Errorf("Expected result to be not nil")
	}
	// Пока что возвращается пустой массив
	if len(result) != 0 {
		t.Errorf("Expected empty result array, got length: %d", len(result))
	}
}

// TestBranchScannerService_CancelBranchScan тестирует отмену сканирования
func TestBranchScannerService_CancelBranchScan(t *testing.T) {
	service := createTestService()
	ctx := context.Background()

	err := service.CancelBranchScan(ctx, "test-scan-123")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// TestGenerateScanID тестирует генерацию ID сканирования
func TestGenerateScanID(t *testing.T) {
	projectKey := "test/project"
	branchName := "main"

	scanID := generateScanID(projectKey, branchName)

	if scanID == "" {
		t.Errorf("Expected scan ID to be generated")
	}
	if len(scanID) < len(projectKey)+len(branchName) {
		t.Errorf("Expected scan ID to contain project key and branch name")
	}
}

// TestGetDefaultRetryConfig тестирует конфигурацию повторов по умолчанию
func TestGetDefaultRetryConfig(t *testing.T) {
	config := getDefaultRetryConfig()

	if config == nil {
		t.Errorf("Expected retry config to be created")
		return
	}
	if config.MaxAttempts <= 0 {
		t.Errorf("Expected max attempts to be positive")
	}
	if config.InitialDelay <= 0 {
		t.Errorf("Expected initial delay to be positive")
	}
	if config.BackoffFactor <= 1.0 {
		t.Errorf("Expected backoff factor to be greater than 1")
	}
	if len(config.RetryableErrors) == 0 {
		t.Errorf("Expected retryable errors to be configured")
	}
}

// Helper functions and types

// createTestService создает тестовый сервис для использования в тестах
func createTestService() *BranchScannerService {
	branchScanner := &sonarqube.BranchScannerEntity{}
	config := &config.ScannerConfig{
		WorkDir: "/tmp/test",
		TempDir: "/tmp",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	return NewBranchScannerService(branchScanner, config, logger)
}

// testError реализует интерфейс error для тестирования
type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}