// Package config содержит тесты для конфигурации SonarQube.
package config

import (
	"os"
	"testing"
	"time"
)

// TestGetDefaultSonarQubeConfig тестирует получение конфигурации SonarQube по умолчанию
func TestGetDefaultSonarQubeConfig(t *testing.T) {
	config := GetDefaultSonarQubeConfig()

	if config == nil {
		t.Fatal("Expected non-nil config")
	}

	if config.URL != "http://localhost:9000" {
		t.Errorf("Expected URL 'http://localhost:9000', got '%s'", config.URL)
	}

	if config.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", config.Timeout)
	}

	if config.RetryAttempts != 3 {
		t.Errorf("Expected retry attempts 3, got %d", config.RetryAttempts)
	}

	if config.RetryDelay != 5*time.Second {
		t.Errorf("Expected retry delay 5s, got %v", config.RetryDelay)
	}

	if config.ProjectPrefix != "benadis" {
		t.Errorf("Expected project prefix 'benadis', got '%s'", config.ProjectPrefix)
	}

	if config.DefaultVisibility != "private" {
		t.Errorf("Expected default visibility 'private', got '%s'", config.DefaultVisibility)
	}

	if config.QualityGateTimeout != 300*time.Second {
		t.Errorf("Expected quality gate timeout 300s, got %v", config.QualityGateTimeout)
	}

	if !config.DisableBranchAnalysis {
		t.Error("Expected DisableBranchAnalysis to be true by default")
	}
}

// TestSonarQubeConfig_Validate тестирует валидацию конфигурации SonarQube
func TestSonarQubeConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *SonarQubeConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: func() *SonarQubeConfig {
				cfg := GetDefaultSonarQubeConfig()
				cfg.Token = "test-token" // Добавляем токен для валидной конфигурации
				return cfg
			}(),
			wantErr: false,
		},
		{
			name: "missing URL",
			config: &SonarQubeConfig{
				Token:              "test-token",
				Timeout:            30 * time.Second,
				RetryAttempts:      3,
				RetryDelay:         5 * time.Second,
				DefaultVisibility:  "private",
				QualityGateTimeout: 300 * time.Second,
			},
			wantErr: true,
			errMsg:  "SonarQube URL is required",
		},
		{
			name: "missing token",
			config: &SonarQubeConfig{
				URL:                "http://localhost:9000",
				Timeout:            30 * time.Second,
				RetryAttempts:      3,
				RetryDelay:         5 * time.Second,
				DefaultVisibility:  "private",
				QualityGateTimeout: 300 * time.Second,
			},
			wantErr: true,
			errMsg:  "SonarQube token is required",
		},
		{
			name: "invalid timeout",
			config: &SonarQubeConfig{
				URL:                "http://localhost:9000",
				Token:              "test-token",
				Timeout:            0,
				RetryAttempts:      3,
				RetryDelay:         5 * time.Second,
				DefaultVisibility:  "private",
				QualityGateTimeout: 300 * time.Second,
			},
			wantErr: true,
			errMsg:  "SonarQube timeout must be positive",
		},
		{
			name: "negative retry attempts",
			config: &SonarQubeConfig{
				URL:                "http://localhost:9000",
				Token:              "test-token",
				Timeout:            30 * time.Second,
				RetryAttempts:      -1,
				RetryDelay:         5 * time.Second,
				DefaultVisibility:  "private",
				QualityGateTimeout: 300 * time.Second,
			},
			wantErr: true,
			errMsg:  "SonarQube retry attempts cannot be negative",
		},
		{
			name: "invalid retry delay",
			config: &SonarQubeConfig{
				URL:                "http://localhost:9000",
				Token:              "test-token",
				Timeout:            30 * time.Second,
				RetryAttempts:      3,
				RetryDelay:         0,
				DefaultVisibility:  "private",
				QualityGateTimeout: 300 * time.Second,
			},
			wantErr: true,
			errMsg:  "SonarQube retry delay must be positive",
		},
		{
			name: "invalid visibility",
			config: &SonarQubeConfig{
				URL:                "http://localhost:9000",
				Token:              "test-token",
				Timeout:            30 * time.Second,
				RetryAttempts:      3,
				RetryDelay:         5 * time.Second,
				DefaultVisibility:  "invalid",
				QualityGateTimeout: 300 * time.Second,
			},
			wantErr: true,
			errMsg:  "SonarQube default visibility must be 'private' or 'public'",
		},
		{
			name: "invalid quality gate timeout",
			config: &SonarQubeConfig{
				URL:                "http://localhost:9000",
				Token:              "test-token",
				Timeout:            30 * time.Second,
				RetryAttempts:      3,
				RetryDelay:         5 * time.Second,
				DefaultVisibility:  "private",
				QualityGateTimeout: 0,
			},
			wantErr: true,
			errMsg:  "SonarQube quality gate timeout must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				} else if err.Error() != tt.errMsg {
					t.Errorf("Expected error '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestGetDefaultScannerConfig тестирует получение конфигурации Scanner по умолчанию
func TestGetDefaultScannerConfig(t *testing.T) {
	config := GetDefaultScannerConfig()

	if config == nil {
		t.Fatal("Expected non-nil config")
	}

	expectedURL := "https://binaries.sonarsource.com/Distribution/sonar-scanner-cli/sonar-scanner-cli-4.8.0.2856-linux.zip"
	if config.ScannerURL != expectedURL {
		t.Errorf("Expected scanner URL '%s', got '%s'", expectedURL, config.ScannerURL)
	}

	if config.ScannerVersion != "4.8.0.2856" {
		t.Errorf("Expected scanner version '4.8.0.2856', got '%s'", config.ScannerVersion)
	}

	if config.JavaOpts != "-Xmx2g" {
		t.Errorf("Expected Java opts '-Xmx2g', got '%s'", config.JavaOpts)
	}

	if config.Properties == nil {
		t.Error("Expected non-nil properties map")
	}

	if config.Timeout != 600*time.Second {
		t.Errorf("Expected timeout 600s, got %v", config.Timeout)
	}

	if config.WorkDir != "/tmp/benadis" {
		t.Errorf("Expected work dir '/tmp/benadis', got '%s'", config.WorkDir)
	}

	if config.TempDir != "/tmp/benadis/scanner/temp" {
		t.Errorf("Expected temp dir '/tmp/benadis/scanner/temp', got '%s'", config.TempDir)
	}
}

// TestScannerConfig_Validate тестирует валидацию конфигурации Scanner
func TestScannerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *ScannerConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			config:  GetDefaultScannerConfig(),
			wantErr: false,
		},
		{
			name: "missing scanner URL",
			config: &ScannerConfig{
				ScannerVersion: "4.8.0.2856",
				Timeout:        600 * time.Second,
			},
			wantErr: true,
			errMsg:  "scanner URL is required",
		},
		{
			name: "missing scanner version",
			config: &ScannerConfig{
				ScannerURL: "https://example.com/scanner.zip",
				Timeout:    600 * time.Second,
			},
			wantErr: true,
			errMsg:  "scanner version is required",
		},
		{
			name: "invalid timeout",
			config: &ScannerConfig{
				ScannerURL:     "https://example.com/scanner.zip",
				ScannerVersion: "4.8.0.2856",
				Timeout:        0,
			},
			wantErr: true,
			errMsg:  "scanner timeout must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				} else if err.Error() != tt.errMsg {
					t.Errorf("Expected error '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestGetSonarQubeConfig тестирует получение конфигурации SonarQube с переменными окружения
func TestGetSonarQubeConfig(t *testing.T) {
	// Сохраняем оригинальные переменные окружения
	originalURL := os.Getenv("SONARQUBE_URL")
	originalToken := os.Getenv("SONARQUBE_TOKEN")

	// Очищаем переменные окружения после теста
	defer func() {
		os.Setenv("SONARQUBE_URL", originalURL)
		os.Setenv("SONARQUBE_TOKEN", originalToken)
	}()

	// Устанавливаем тестовые переменные окружения
	os.Setenv("SONARQUBE_URL", "http://test.sonarqube.com")
	os.Setenv("SONARQUBE_TOKEN", "test-token-123")

	cfg := &Config{}
	config, err := GetSonarQubeConfig(nil, cfg)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if config.URL != "http://test.sonarqube.com" {
		t.Errorf("Expected URL from env var, got '%s'", config.URL)
	}

	if config.Token != "test-token-123" {
		t.Errorf("Expected token from env var, got '%s'", config.Token)
	}
}

// TestGetScannerConfig тестирует получение конфигурации Scanner с переменными окружения
func TestGetScannerConfig(t *testing.T) {
	// Сохраняем оригинальные переменные окружения
	originalURL := os.Getenv("SONARQUBE_SCANNER_URL")
	originalVersion := os.Getenv("SONARQUBE_SCANNER_VERSION")

	// Очищаем переменные окружения после теста
	defer func() {
		os.Setenv("SONARQUBE_SCANNER_URL", originalURL)
		os.Setenv("SONARQUBE_SCANNER_VERSION", originalVersion)
	}()

	// Устанавливаем тестовые переменные окружения
	os.Setenv("SONARQUBE_SCANNER_URL", "http://test.scanner.com/scanner.zip")
	os.Setenv("SONARQUBE_SCANNER_VERSION", "5.0.0")

	cfg := &Config{}
	config, err := GetScannerConfig(nil, cfg)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if config.ScannerURL != "http://test.scanner.com/scanner.zip" {
		t.Errorf("Expected scanner URL from env var, got '%s'", config.ScannerURL)
	}

	if config.ScannerVersion != "5.0.0" {
		t.Errorf("Expected scanner version from env var, got '%s'", config.ScannerVersion)
	}
}