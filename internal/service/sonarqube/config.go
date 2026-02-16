// Package sonarqube provides implementation of configuration management functionality.
// This package contains the implementation of configuration validation,
// environment variable support, and configuration hot-reloading.
package sonarqube

import (
	"os"
	"strconv"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/entity/sonarqube"
)

// ConfigService provides functionality for configuration management.
// This service layer implements configuration validation, environment variable support,
// and configuration hot-reloading.
type ConfigService struct {
	// appConfig contains the application configuration
	appConfig *config.AppConfig
	
	// secretConfig contains the secret configuration
	secretConfig *config.SecretConfig
}

// NewConfigService creates a new instance of ConfigService.
// This function initializes the service with the provided configurations.
//
// Parameters:
//   - appConfig: application configuration
//   - secretConfig: secret configuration
//
// Returns:
//   - *ConfigService: initialized configuration service
func NewConfigService(appConfig *config.AppConfig, secretConfig *config.SecretConfig) *ConfigService {
	return &ConfigService{
		appConfig:    appConfig,
		secretConfig: secretConfig,
	}
}

// ValidateConfig validates the SonarQube configuration.
// This method validates the SonarQube configuration, checking that
// required fields are set and have valid values.
//
// Returns:
//   - error: error if configuration is invalid
func (c *ConfigService) ValidateConfig() error {
	// Validate SonarQube configuration
	if c.appConfig.SonarQube.URL == "" {
		return &sonarqube.ValidationError{
			Field:   "sonarqube.url",
			Message: "SonarQube URL is required",
		}
	}
	
	if c.appConfig.SonarQube.Timeout <= 0 {
		return &sonarqube.ValidationError{
			Field:   "sonarqube.timeout",
			Message: "SonarQube timeout must be positive",
		}
	}
	
	if c.appConfig.SonarQube.RetryAttempts < 0 {
		return &sonarqube.ValidationError{
			Field:   "sonarqube.retryAttempts",
			Message: "SonarQube retry attempts must be non-negative",
		}
	}
	
	if c.appConfig.SonarQube.RetryDelay <= 0 {
		return &sonarqube.ValidationError{
			Field:   "sonarqube.retryDelay",
			Message: "SonarQube retry delay must be positive",
		}
	}
	
	if c.appConfig.SonarQube.QualityGateTimeout <= 0 {
		return &sonarqube.ValidationError{
			Field:   "sonarqube.qualityGateTimeout",
			Message: "SonarQube quality gate timeout must be positive",
		}
	}
	
	// Validate Scanner configuration
	if c.appConfig.Scanner.ScannerURL == "" {
		return &sonarqube.ValidationError{
			Field:   "scanner.scannerUrl",
			Message: "Scanner URL is required",
	}
	}
	
	if c.appConfig.Scanner.Timeout <= 0 {
		return &sonarqube.ValidationError{
			Field:   "scanner.timeout",
			Message: "Scanner timeout must be positive",
		}
	}
	
	return nil
}

// LoadConfigFromEnv loads configuration from environment variables.
// This method loads configuration from environment variables,
// overriding the existing configuration.
//
// Returns:
//   - error: error if loading fails
func (c *ConfigService) LoadConfigFromEnv() error {
	// Load SonarQube configuration from environment variables
	if sonarQubeURL := os.Getenv("SONARQUBE_URL"); sonarQubeURL != "" {
		c.appConfig.SonarQube.URL = sonarQubeURL
	}
	
	if sonarQubeToken := os.Getenv("SONARQUBE_TOKEN"); sonarQubeToken != "" {
		c.secretConfig.SonarQube.Token = sonarQubeToken
	}
	
	if sonarQubeTimeout := os.Getenv("SONARQUBE_TIMEOUT"); sonarQubeTimeout != "" {
		if timeout, err := time.ParseDuration(sonarQubeTimeout); err == nil {
			c.appConfig.SonarQube.Timeout = timeout
		}
	}
	
	if sonarQubeRetryAttempts := os.Getenv("SONARQUBE_RETRY_ATTEMPTS"); sonarQubeRetryAttempts != "" {
		if attempts, err := strconv.Atoi(sonarQubeRetryAttempts); err == nil {
			c.appConfig.SonarQube.RetryAttempts = attempts
		}
	}
	
	if sonarQubeRetryDelay := os.Getenv("SONARQUBE_RETRY_DELAY"); sonarQubeRetryDelay != "" {
		if delay, err := time.ParseDuration(sonarQubeRetryDelay); err == nil {
			c.appConfig.SonarQube.RetryDelay = delay
		}
	}
	
	if sonarQubeProjectPrefix := os.Getenv("SONARQUBE_PROJECT_PREFIX"); sonarQubeProjectPrefix != "" {
		c.appConfig.SonarQube.ProjectPrefix = sonarQubeProjectPrefix
	}
	
	if sonarQubeDefaultVisibility := os.Getenv("SONARQUBE_DEFAULT_VISIBILITY"); sonarQubeDefaultVisibility != "" {
		c.appConfig.SonarQube.DefaultVisibility = sonarQubeDefaultVisibility
	}
	
	if sonarQubeQualityGateTimeout := os.Getenv("SONARQUBE_QUALITY_GATE_TIMEOUT"); sonarQubeQualityGateTimeout != "" {
		if timeout, err := time.ParseDuration(sonarQubeQualityGateTimeout); err == nil {
			c.appConfig.SonarQube.QualityGateTimeout = timeout
		}
	}
	
	// Load Scanner configuration from environment variables
	if scannerURL := os.Getenv("SCANNER_URL"); scannerURL != "" {
		c.appConfig.Scanner.ScannerURL = scannerURL
	}
	
	if scannerVersion := os.Getenv("SCANNER_VERSION"); scannerVersion != "" {
		c.appConfig.Scanner.ScannerVersion = scannerVersion
	}
	
	if scannerJavaOpts := os.Getenv("SCANNER_JAVA_OPTS"); scannerJavaOpts != "" {
		c.appConfig.Scanner.JavaOpts = scannerJavaOpts
	}
	
	if scannerTimeout := os.Getenv("SCANNER_TIMEOUT"); scannerTimeout != "" {
		if timeout, err := time.ParseDuration(scannerTimeout); err == nil {
			c.appConfig.Scanner.Timeout = timeout
		}
	}
	
	if scannerWorkDir := os.Getenv("SCANNER_WORK_DIR"); scannerWorkDir != "" {
		c.appConfig.Scanner.WorkDir = scannerWorkDir
	}
	
	if scannerTempDir := os.Getenv("SCANNER_TEMP_DIR"); scannerTempDir != "" {
		c.appConfig.Scanner.TempDir = scannerTempDir
	}
	
	return nil
}

// GetSonarQubeConfig returns the SonarQube configuration.
// This method returns the SonarQube configuration.
//
// Returns:
//   - *config.SonarQubeConfig: SonarQube configuration
func (c *ConfigService) GetSonarQubeConfig() *config.SonarQubeConfig {
	return &c.appConfig.SonarQube
}

// GetScannerConfig returns the scanner configuration.
// This method returns the scanner configuration.
//
// Returns:
//   - *config.ScannerConfig: scanner configuration
func (c *ConfigService) GetScannerConfig() *config.ScannerConfig {
	return &c.appConfig.Scanner
}

// GetSonarQubeToken returns the SonarQube token.
// This method returns the SonarQube token from the secret configuration.
//
// Returns:
//   - string: SonarQube token
func (c *ConfigService) GetSonarQubeToken() string {
	return c.secretConfig.SonarQube.Token
}

// SetSonarQubeToken sets the SonarQube token.
// This method sets the SonarQube token in the secret configuration.
//
// Parameters:
//   - token: SonarQube token
func (c *ConfigService) SetSonarQubeToken(token string) {
	c.secretConfig.SonarQube.Token = token
}

// ReloadConfig reloads the configuration from files.
// This method reloads the configuration from files,
// updating the existing configuration.
//
// Parameters:
//   - appConfig: application configuration
//   - secretConfig: secret configuration
func (c *ConfigService) ReloadConfig(appConfig *config.AppConfig, secretConfig *config.SecretConfig) {
	c.appConfig = appConfig
	c.secretConfig = secretConfig
}

// ToDo: необходимо дополнительно реализовать следующий функционал:
// - Add unit tests for all methods
// - Implement configuration hot-reloading capability
// - Implement better error handling and recovery
// - Add progress reporting during operations
//
// Ссылки на пункты плана и требований:
// - tasks.md: 10.1, 10.2
// - requirements.md: 9.2, 9.3