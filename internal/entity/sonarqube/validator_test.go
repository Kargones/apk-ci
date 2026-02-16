package sonarqube

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/stretchr/testify/assert"
)

// TestNewConfigValidator тестирует создание нового валидатора конфигурации
func TestNewConfigValidator(t *testing.T) {
	validator := NewConfigValidator()

	assert.NotNil(t, validator)
	assert.NotNil(t, validator.validProjectKeyPattern)
	assert.NotNil(t, validator.validVersionPattern)
	assert.NotEmpty(t, validator.requiredProperties)
	assert.NotEmpty(t, validator.validPropertyPatterns)
}

// TestValidationResult_AddError тестирует добавление ошибки валидации
func TestValidationResult_AddError(t *testing.T) {
	result := &ValidationResult{Valid: true}

	result.AddError("testField", "testValue", "test message", "TEST_CODE")

	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, "testField", result.Errors[0].Field)
	assert.Equal(t, "testValue", result.Errors[0].Value)
	assert.Equal(t, "test message", result.Errors[0].Message)
	assert.Equal(t, "TEST_CODE", result.Errors[0].Code)
}

// TestValidationResult_AddWarning тестирует добавление предупреждения валидации
func TestValidationResult_AddWarning(t *testing.T) {
	result := &ValidationResult{Valid: true}

	result.AddWarning("testField", "testValue", "test warning", "WARN_CODE")

	assert.True(t, result.Valid) // Предупреждения не делают результат невалидным
	assert.Len(t, result.Warnings, 1)
	assert.Equal(t, "testField", result.Warnings[0].Field)
	assert.Equal(t, "testValue", result.Warnings[0].Value)
	assert.Equal(t, "test warning", result.Warnings[0].Message)
	assert.Equal(t, "WARN_CODE", result.Warnings[0].Code)
}

// TestValidationResult_HasErrors тестирует проверку наличия ошибок
func TestValidationResult_HasErrors(t *testing.T) {
	result := &ValidationResult{}

	assert.False(t, result.HasErrors())

	result.AddError("field", "value", "message", "CODE")
	assert.True(t, result.HasErrors())
}

// TestValidationResult_HasWarnings тестирует проверку наличия предупреждений
func TestValidationResult_HasWarnings(t *testing.T) {
	result := &ValidationResult{}

	assert.False(t, result.HasWarnings())

	result.AddWarning("field", "value", "message", "CODE")
	assert.True(t, result.HasWarnings())
}

// TestExtendedValidationError_Error тестирует форматирование ошибки
func TestExtendedValidationError_Error(t *testing.T) {
	err := &ExtendedValidationError{
		ValidationError: &ValidationError{
			Field:   "testField",
			Value:   "testValue",
			Message: "test message",
		},
		Code: "TEST_CODE",
	}

	expected := "validation error for field 'testField': test message (value: 'testValue', code: TEST_CODE)"
	assert.Equal(t, expected, err.Error())
}

// TestConfigValidator_ValidateConfig тестирует валидацию конфигурации
func TestConfigValidator_ValidateConfig(t *testing.T) {
	validator := NewConfigValidator()

	// Создаем временную директорию для тестов
	tempDir := t.TempDir()
	// Создаем поддиректорию temp
	tempSubDir := filepath.Join(tempDir, "temp")
	err := os.MkdirAll(tempSubDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	tests := []struct {
		name           string
		config         *config.ScannerConfig
		expectValid    bool
		expectErrors   int
		expectWarnings int
	}{
		{
			name: "valid config",
			config: &config.ScannerConfig{
				ScannerURL:     "https://binaries.sonarsource.com/Distribution/sonar-scanner-cli/sonar-scanner-cli-4.8.0.2856-linux.zip",
				ScannerVersion: "4.8.0",
				JavaOpts:       "-Xmx2g",
				Timeout:        600 * time.Second, // Достаточно длинный таймаут
				WorkDir:        tempDir,
				TempDir:        tempSubDir,
				Properties: map[string]string{
					"sonar.host.url":   "http://localhost:9000",
					"sonar.projectKey": "test-project",
					"sonar.token":      "squ_1234567890abcdef1234567890abcdef12345678", // Длинный токен
				},
			},
			expectValid:    true,
			expectErrors:   0,
			expectWarnings: 0, // Валидная конфигурация
		},
		{
			name: "invalid project key",
			config: &config.ScannerConfig{
				ScannerURL:     "https://binaries.sonarsource.com/Distribution/sonar-scanner-cli/sonar-scanner-cli-4.8.0.2856-linux.zip",
				ScannerVersion: "4.8.0",
				JavaOpts:       "-Xmx2g",
				Timeout:        600 * time.Second,
				WorkDir:        tempDir,
				TempDir:        tempSubDir,
				Properties: map[string]string{
					"sonar.host.url":   "http://localhost:9000",
					"sonar.projectKey": "invalid key with spaces",
					"sonar.token":      "squ_1234567890abcdef1234567890abcdef12345678",
				},
			},
			expectValid:  false,
			expectErrors: 2, // Ошибки валидации ключа проекта (формат + символы)
		},
		{
			name: "missing required properties",
			config: &config.ScannerConfig{
				ScannerURL:     "https://binaries.sonarsource.com/Distribution/sonar-scanner-cli/sonar-scanner-cli-4.8.0.2856-linux.zip",
				ScannerVersion: "4.8.0",
				JavaOpts:       "-Xmx2g",
				Timeout:        600 * time.Second,
				WorkDir:        tempDir,
				TempDir:        tempSubDir,
				Properties:     map[string]string{}, // Пустые свойства
			},
			expectValid:  false,
			expectErrors:   2, // Отсутствуют sonar.host.url и sonar.projectKey
			expectWarnings: 1, // Предупреждение об отсутствии аутентификации
		},
		{
			name: "invalid URL",
			config: &config.ScannerConfig{
				ScannerURL:     "https://binaries.sonarsource.com/Distribution/sonar-scanner-cli/sonar-scanner-cli-4.8.0.2856-linux.zip",
				ScannerVersion: "4.8.0",
				JavaOpts:       "-Xmx2g",
				Timeout:        600 * time.Second,
				WorkDir:        tempDir,
				TempDir:        tempSubDir,
				Properties: map[string]string{
					"sonar.host.url":   "://invalid", // Явно неправильный URL
					"sonar.projectKey": "test-project",
					"sonar.token":      "squ_1234567890abcdef1234567890abcdef12345678",
				},
			},
			expectValid:  false,
			expectErrors: 1, // Ошибка валидации URL
		},
		{
			name: "short timeout",
			config: &config.ScannerConfig{
				ScannerURL:     "https://binaries.sonarsource.com/Distribution/sonar-scanner-cli/sonar-scanner-cli-4.8.0.2856-linux.zip",
				ScannerVersion: "4.8.0",
				JavaOpts:       "-Xmx2g",
				Timeout:        5 * time.Second, // Слишком короткий таймаут
				WorkDir:        tempDir,
				TempDir:        tempSubDir,
				Properties: map[string]string{
					"sonar.host.url":   "http://localhost:9000",
					"sonar.projectKey": "test-project",
					"sonar.token":      "squ_1234567890abcdef1234567890abcdef12345678",
				},
			},
			expectValid:    true, // Короткий таймаут - это предупреждение, не ошибка
			expectErrors:   0,
			expectWarnings: 1, // Предупреждение о таймауте
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateConfig(tt.config)

			assert.Equal(t, tt.expectValid, result.Valid, "Unexpected validation result")
			assert.Len(t, result.Errors, tt.expectErrors, "Unexpected number of errors")
			assert.Len(t, result.Warnings, tt.expectWarnings, "Unexpected number of warnings")
		})
	}
}

// TestConfigValidator_ValidateProjectKey тестирует валидацию ключа проекта
func TestConfigValidator_ValidateProjectKey(t *testing.T) {
	validator := NewConfigValidator()

	tests := []struct {
		name        string
		projectKey  string
		expectValid bool
	}{
		{"valid key with letters", "myproject", true},
		{"valid key with numbers", "project123", true},
		{"valid key with underscores", "my_project", true},
		{"valid key with hyphens", "my-project", true},
		{"valid key with dots", "my.project", true},
		{"valid key with colons", "my:project", true},
		{"invalid key with spaces", "my project", false},
		{"invalid key with special chars", "my@project", false},
		{"empty key", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &config.ScannerConfig{
				ScannerURL:     "https://binaries.sonarsource.com/Distribution/sonar-scanner-cli/sonar-scanner-cli-4.8.0.2856-linux.zip",
				ScannerVersion: "4.8.0.2856",
				JavaOpts:       "-Xmx2g",
				Timeout:        30 * time.Second,
				WorkDir:        "/tmp/benadis",
				TempDir:        "/tmp/benadis/scanner/temp",
				Properties: map[string]string{
					"sonar.host.url":   "http://localhost:9000",
					"sonar.projectKey": tt.projectKey,
				},
			}

			result := validator.ValidateConfig(config)

			if tt.expectValid {
				// Может быть невалидным из-за других полей, но не из-за ключа проекта
				for _, err := range result.Errors {
					assert.NotContains(t, err.Field, "Properties.sonar.projectKey")
				}
			} else {
				// Должна быть ошибка валидации ключа проекта
				found := false
				for _, err := range result.Errors {
					if err.Field == "Properties.sonar.projectKey" {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected validation error for project key")
			}
		})
	}
}