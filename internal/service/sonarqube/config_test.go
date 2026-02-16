package sonarqube

import (
	"os"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/entity/sonarqube"
)

func TestNewConfigService(t *testing.T) {
	appConfig := &config.AppConfig{}
	secretConfig := &config.SecretConfig{}

	service := NewConfigService(appConfig, secretConfig)

	if service == nil {
		t.Fatal("Expected non-nil service")
	}

	if service.appConfig != appConfig {
		t.Error("Expected appConfig to be set correctly")
	}

	if service.secretConfig != secretConfig {
		t.Error("Expected secretConfig to be set correctly")
	}
}

func TestConfigService_ValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		appConfig   *config.AppConfig
		wantErr     bool
		expectedErr string
	}{
		{
			name: "valid config",
			appConfig: &config.AppConfig{
				SonarQube: config.SonarQubeConfig{
					URL:                "https://sonar.example.com",
					Timeout:            30 * time.Second,
					RetryAttempts:      3,
					RetryDelay:         5 * time.Second,
					QualityGateTimeout: 300 * time.Second,
				},
				Scanner: config.ScannerConfig{
					ScannerURL: "https://scanner.example.com",
					Timeout:    60 * time.Second,
				},
			},
			wantErr: false,
		},
		{
			name: "missing SonarQube URL",
			appConfig: &config.AppConfig{
				SonarQube: config.SonarQubeConfig{
					URL:                "",
					Timeout:            30 * time.Second,
					RetryAttempts:      3,
					RetryDelay:         5 * time.Second,
					QualityGateTimeout: 300 * time.Second,
				},
				Scanner: config.ScannerConfig{
					ScannerURL: "https://scanner.example.com",
					Timeout:    60 * time.Second,
				},
			},
			wantErr:     true,
			expectedErr: "SonarQube URL is required",
		},
		{
			name: "invalid SonarQube timeout",
			appConfig: &config.AppConfig{
				SonarQube: config.SonarQubeConfig{
					URL:                "https://sonar.example.com",
					Timeout:            -1 * time.Second,
					RetryAttempts:      3,
					RetryDelay:         5 * time.Second,
					QualityGateTimeout: 300 * time.Second,
				},
				Scanner: config.ScannerConfig{
					ScannerURL: "https://scanner.example.com",
					Timeout:    60 * time.Second,
				},
			},
			wantErr:     true,
			expectedErr: "SonarQube timeout must be positive",
		},
		{
			name: "invalid retry attempts",
			appConfig: &config.AppConfig{
				SonarQube: config.SonarQubeConfig{
					URL:                "https://sonar.example.com",
					Timeout:            30 * time.Second,
					RetryAttempts:      -1,
					RetryDelay:         5 * time.Second,
					QualityGateTimeout: 300 * time.Second,
				},
				Scanner: config.ScannerConfig{
					ScannerURL: "https://scanner.example.com",
					Timeout:    60 * time.Second,
				},
			},
			wantErr:     true,
			expectedErr: "SonarQube retry attempts must be non-negative",
		},
		{
			name: "invalid retry delay",
			appConfig: &config.AppConfig{
				SonarQube: config.SonarQubeConfig{
					URL:                "https://sonar.example.com",
					Timeout:            30 * time.Second,
					RetryAttempts:      3,
					RetryDelay:         -1 * time.Second,
					QualityGateTimeout: 300 * time.Second,
				},
				Scanner: config.ScannerConfig{
					ScannerURL: "https://scanner.example.com",
					Timeout:    60 * time.Second,
				},
			},
			wantErr:     true,
			expectedErr: "SonarQube retry delay must be positive",
		},
		{
			name: "invalid quality gate timeout",
			appConfig: &config.AppConfig{
				SonarQube: config.SonarQubeConfig{
					URL:                "https://sonar.example.com",
					Timeout:            30 * time.Second,
					RetryAttempts:      3,
					RetryDelay:         5 * time.Second,
					QualityGateTimeout: -1 * time.Second,
				},
				Scanner: config.ScannerConfig{
					ScannerURL: "https://scanner.example.com",
					Timeout:    60 * time.Second,
				},
			},
			wantErr:     true,
			expectedErr: "SonarQube quality gate timeout must be positive",
		},
		{
			name: "missing scanner URL",
			appConfig: &config.AppConfig{
				SonarQube: config.SonarQubeConfig{
					URL:                "https://sonar.example.com",
					Timeout:            30 * time.Second,
					RetryAttempts:      3,
					RetryDelay:         5 * time.Second,
					QualityGateTimeout: 300 * time.Second,
				},
				Scanner: config.ScannerConfig{
					ScannerURL: "",
					Timeout:    60 * time.Second,
				},
			},
			wantErr:     true,
			expectedErr: "Scanner URL is required",
		},
		{
			name: "invalid scanner timeout",
			appConfig: &config.AppConfig{
				SonarQube: config.SonarQubeConfig{
					URL:                "https://sonar.example.com",
					Timeout:            30 * time.Second,
					RetryAttempts:      3,
					RetryDelay:         5 * time.Second,
					QualityGateTimeout: 300 * time.Second,
				},
				Scanner: config.ScannerConfig{
					ScannerURL: "https://scanner.example.com",
					Timeout:    -1 * time.Second,
				},
			},
			wantErr:     true,
			expectedErr: "Scanner timeout must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewConfigService(tt.appConfig, &config.SecretConfig{})
			err := service.ValidateConfig()

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if validationErr, ok := err.(*sonarqube.ValidationError); ok {
					if validationErr.Message != tt.expectedErr {
						t.Errorf("ValidateConfig() error message = %v, want %v", validationErr.Message, tt.expectedErr)
					}
				} else {
					t.Errorf("Expected ValidationError, got %T", err)
				}
			}
		})
	}
}

func TestConfigService_LoadConfigFromEnv(t *testing.T) {
	// Save original environment variables
	originalVars := map[string]string{
		"SONARQUBE_URL":                    os.Getenv("SONARQUBE_URL"),
		"SONARQUBE_TOKEN":                  os.Getenv("SONARQUBE_TOKEN"),
		"SONARQUBE_TIMEOUT":                os.Getenv("SONARQUBE_TIMEOUT"),
		"SONARQUBE_RETRY_ATTEMPTS":         os.Getenv("SONARQUBE_RETRY_ATTEMPTS"),
		"SONARQUBE_RETRY_DELAY":            os.Getenv("SONARQUBE_RETRY_DELAY"),
		"SONARQUBE_PROJECT_PREFIX":         os.Getenv("SONARQUBE_PROJECT_PREFIX"),
		"SONARQUBE_DEFAULT_VISIBILITY":     os.Getenv("SONARQUBE_DEFAULT_VISIBILITY"),
		"SONARQUBE_QUALITY_GATE_TIMEOUT":   os.Getenv("SONARQUBE_QUALITY_GATE_TIMEOUT"),
		"SCANNER_URL":                      os.Getenv("SCANNER_URL"),
		"SCANNER_VERSION":                  os.Getenv("SCANNER_VERSION"),
		"SCANNER_JAVA_OPTS":                os.Getenv("SCANNER_JAVA_OPTS"),
		"SCANNER_TIMEOUT":                  os.Getenv("SCANNER_TIMEOUT"),
		"SCANNER_WORK_DIR":                 os.Getenv("SCANNER_WORK_DIR"),
		"SCANNER_TEMP_DIR":                 os.Getenv("SCANNER_TEMP_DIR"),
	}

	// Restore original environment variables after test
	defer func() {
		for key, value := range originalVars {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Set test environment variables
	os.Setenv("SONARQUBE_URL", "https://test-sonar.example.com")
	os.Setenv("SONARQUBE_TOKEN", "test-token")
	os.Setenv("SONARQUBE_TIMEOUT", "45s")
	os.Setenv("SONARQUBE_RETRY_ATTEMPTS", "5")
	os.Setenv("SONARQUBE_RETRY_DELAY", "10s")
	os.Setenv("SONARQUBE_PROJECT_PREFIX", "test-prefix")
	os.Setenv("SONARQUBE_DEFAULT_VISIBILITY", "private")
	os.Setenv("SONARQUBE_QUALITY_GATE_TIMEOUT", "600s")
	os.Setenv("SCANNER_URL", "https://test-scanner.example.com")
	os.Setenv("SCANNER_VERSION", "4.8.0")
	os.Setenv("SCANNER_JAVA_OPTS", "-Xmx1024m")
	os.Setenv("SCANNER_TIMEOUT", "120s")
	os.Setenv("SCANNER_WORK_DIR", "/tmp/work")
	os.Setenv("SCANNER_TEMP_DIR", "/tmp/temp")

	appConfig := &config.AppConfig{
		SonarQube: config.SonarQubeConfig{},
		Scanner:   config.ScannerConfig{},
	}
	secretConfig := &config.SecretConfig{
		SonarQube: struct {
			Token string `yaml:"token"`
		}{},
	}

	service := NewConfigService(appConfig, secretConfig)
	err := service.LoadConfigFromEnv()

	if err != nil {
		t.Errorf("LoadConfigFromEnv() error = %v", err)
	}

	// Verify SonarQube configuration
	if appConfig.SonarQube.URL != "https://test-sonar.example.com" {
		t.Errorf("Expected SonarQube URL to be 'https://test-sonar.example.com', got '%s'", appConfig.SonarQube.URL)
	}

	if secretConfig.SonarQube.Token != "test-token" {
		t.Errorf("Expected SonarQube token to be 'test-token', got '%s'", secretConfig.SonarQube.Token)
	}

	if appConfig.SonarQube.Timeout != 45*time.Second {
		t.Errorf("Expected SonarQube timeout to be 45s, got %v", appConfig.SonarQube.Timeout)
	}

	if appConfig.SonarQube.RetryAttempts != 5 {
		t.Errorf("Expected SonarQube retry attempts to be 5, got %d", appConfig.SonarQube.RetryAttempts)
	}

	if appConfig.SonarQube.RetryDelay != 10*time.Second {
		t.Errorf("Expected SonarQube retry delay to be 10s, got %v", appConfig.SonarQube.RetryDelay)
	}

	if appConfig.SonarQube.ProjectPrefix != "test-prefix" {
		t.Errorf("Expected SonarQube project prefix to be 'test-prefix', got '%s'", appConfig.SonarQube.ProjectPrefix)
	}

	if appConfig.SonarQube.DefaultVisibility != "private" {
		t.Errorf("Expected SonarQube default visibility to be 'private', got '%s'", appConfig.SonarQube.DefaultVisibility)
	}

	if appConfig.SonarQube.QualityGateTimeout != 600*time.Second {
		t.Errorf("Expected SonarQube quality gate timeout to be 600s, got %v", appConfig.SonarQube.QualityGateTimeout)
	}

	// Verify Scanner configuration
	if appConfig.Scanner.ScannerURL != "https://test-scanner.example.com" {
		t.Errorf("Expected Scanner URL to be 'https://test-scanner.example.com', got '%s'", appConfig.Scanner.ScannerURL)
	}

	if appConfig.Scanner.ScannerVersion != "4.8.0" {
		t.Errorf("Expected Scanner version to be '4.8.0', got '%s'", appConfig.Scanner.ScannerVersion)
	}

	if appConfig.Scanner.JavaOpts != "-Xmx1024m" {
		t.Errorf("Expected Scanner Java opts to be '-Xmx1024m', got '%s'", appConfig.Scanner.JavaOpts)
	}

	if appConfig.Scanner.Timeout != 120*time.Second {
		t.Errorf("Expected Scanner timeout to be 120s, got %v", appConfig.Scanner.Timeout)
	}

	if appConfig.Scanner.WorkDir != "/tmp/work" {
		t.Errorf("Expected Scanner work dir to be '/tmp/work', got '%s'", appConfig.Scanner.WorkDir)
	}

	if appConfig.Scanner.TempDir != "/tmp/temp" {
		t.Errorf("Expected Scanner temp dir to be '/tmp/temp', got '%s'", appConfig.Scanner.TempDir)
	}
}

func TestConfigService_GetMethods(t *testing.T) {
	appConfig := &config.AppConfig{
		SonarQube: config.SonarQubeConfig{
			URL: "https://sonar.example.com",
		},
		Scanner: config.ScannerConfig{
			ScannerURL: "https://scanner.example.com",
		},
	}
	secretConfig := &config.SecretConfig{
		SonarQube: struct {
			Token string `yaml:"token"`
		}{
			Token: "test-token",
		},
	}

	service := NewConfigService(appConfig, secretConfig)

	// Test GetSonarQubeConfig
	sonarConfig := service.GetSonarQubeConfig()
	if sonarConfig.URL != "https://sonar.example.com" {
		t.Errorf("Expected SonarQube URL to be 'https://sonar.example.com', got '%s'", sonarConfig.URL)
	}

	// Test GetScannerConfig
	scannerConfig := service.GetScannerConfig()
	if scannerConfig.ScannerURL != "https://scanner.example.com" {
		t.Errorf("Expected Scanner URL to be 'https://scanner.example.com', got '%s'", scannerConfig.ScannerURL)
	}

	// Test GetSonarQubeToken
	token := service.GetSonarQubeToken()
	if token != "test-token" {
		t.Errorf("Expected SonarQube token to be 'test-token', got '%s'", token)
	}
}

func TestConfigService_SetSonarQubeToken(t *testing.T) {
	appConfig := &config.AppConfig{}
	secretConfig := &config.SecretConfig{
		SonarQube: struct {
			Token string `yaml:"token"`
		}{},
	}

	service := NewConfigService(appConfig, secretConfig)

	// Test SetSonarQubeToken
	service.SetSonarQubeToken("new-token")

	if service.GetSonarQubeToken() != "new-token" {
		t.Errorf("Expected SonarQube token to be 'new-token', got '%s'", service.GetSonarQubeToken())
	}
}

func TestConfigService_ReloadConfig(t *testing.T) {
	appConfig := &config.AppConfig{
		SonarQube: config.SonarQubeConfig{
			URL: "https://old-sonar.example.com",
		},
	}
	secretConfig := &config.SecretConfig{
		SonarQube: struct {
			Token string `yaml:"token"`
		}{
			Token: "old-token",
		},
	}

	service := NewConfigService(appConfig, secretConfig)

	// Create new configurations
	newAppConfig := &config.AppConfig{
		SonarQube: config.SonarQubeConfig{
			URL: "https://new-sonar.example.com",
		},
	}
	newSecretConfig := &config.SecretConfig{
		SonarQube: struct {
			Token string `yaml:"token"`
		}{
			Token: "new-token",
		},
	}

	// Test ReloadConfig
	service.ReloadConfig(newAppConfig, newSecretConfig)

	if service.GetSonarQubeConfig().URL != "https://new-sonar.example.com" {
		t.Errorf("Expected SonarQube URL to be 'https://new-sonar.example.com', got '%s'", service.GetSonarQubeConfig().URL)
	}

	if service.GetSonarQubeToken() != "new-token" {
		t.Errorf("Expected SonarQube token to be 'new-token', got '%s'", service.GetSonarQubeToken())
	}
}

func TestConfigService_LoadConfigFromEnv_ErrorHandling(t *testing.T) {
	// Save original environment variables
	originalVars := map[string]string{
		"SONARQUBE_TIMEOUT":        os.Getenv("SONARQUBE_TIMEOUT"),
		"SONARQUBE_RETRY_ATTEMPTS": os.Getenv("SONARQUBE_RETRY_ATTEMPTS"),
		"SONARQUBE_RETRY_DELAY":    os.Getenv("SONARQUBE_RETRY_DELAY"),
		"SCANNER_TIMEOUT":          os.Getenv("SCANNER_TIMEOUT"),
	}

	// Restore original environment variables after test
	defer func() {
		for key, value := range originalVars {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	appConfig := &config.AppConfig{
		SonarQube: config.SonarQubeConfig{
			Timeout:            30 * time.Second,
			RetryAttempts:      3,
			RetryDelay:         5 * time.Second,
		},
		Scanner: config.ScannerConfig{
			Timeout: 60 * time.Second,
		},
	}
	secretConfig := &config.SecretConfig{}

	service := NewConfigService(appConfig, secretConfig)

	// Test with invalid timeout format
	os.Setenv("SONARQUBE_TIMEOUT", "invalid")
	os.Setenv("SONARQUBE_RETRY_ATTEMPTS", "invalid")
	os.Setenv("SONARQUBE_RETRY_DELAY", "invalid")
	os.Setenv("SCANNER_TIMEOUT", "invalid")

	err := service.LoadConfigFromEnv()
	if err != nil {
		t.Errorf("LoadConfigFromEnv() should not error on invalid env vars, got %v", err)
	}

	// Values should remain unchanged due to parsing errors
	if appConfig.SonarQube.Timeout != 30*time.Second {
		t.Errorf("Expected SonarQube timeout to remain 30s, got %v", appConfig.SonarQube.Timeout)
	}

	if appConfig.SonarQube.RetryAttempts != 3 {
		t.Errorf("Expected SonarQube retry attempts to remain 3, got %d", appConfig.SonarQube.RetryAttempts)
	}

	if appConfig.SonarQube.RetryDelay != 5*time.Second {
		t.Errorf("Expected SonarQube retry delay to remain 5s, got %v", appConfig.SonarQube.RetryDelay)
	}

	if appConfig.Scanner.Timeout != 60*time.Second {
		t.Errorf("Expected Scanner timeout to remain 60s, got %v", appConfig.Scanner.Timeout)
	}
}