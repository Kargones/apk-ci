// Package config provides configuration structures for SonarQube integration.
// This package defines the configuration structures for SonarQube and sonar-scanner
// that extend the existing application configuration.
package config

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// SonarQubeConfig represents the configuration for SonarQube integration.
// This struct defines the settings needed to connect to and interact with
// a SonarQube server, including authentication, timeouts, and project settings.
type SonarQubeConfig struct {
	// URL is the base URL of the SonarQube server.
	URL string `yaml:"url" env:"SONARQUBE_URL"`

	// Token is the authentication token for accessing the SonarQube API.
	// This should be stored securely and not committed to version control.
	Token string `yaml:"token" env:"SONARQUBE_TOKEN"`

	// Timeout is the maximum time to wait for SonarQube API requests.
	Timeout time.Duration `yaml:"timeout" env:"SONARQUBE_TIMEOUT"`

	// RetryAttempts is the number of retry attempts for failed API requests.
	RetryAttempts int `yaml:"retryAttempts" env:"SONARQUBE_RETRY_ATTEMPTS"`

	// RetryDelay is the initial delay between retry attempts, which will
	// increase exponentially for subsequent retries.
	RetryDelay time.Duration `yaml:"retryDelay" env:"SONARQUBE_RETRY_DELAY"`

	// ProjectPrefix is the prefix to use when creating SonarQube project keys.
	// This helps organize projects in SonarQube by adding a consistent prefix.
	ProjectPrefix string `yaml:"projectPrefix" env:"SONARQUBE_PROJECT_PREFIX"`

	// DefaultVisibility is the default visibility setting for new projects.
	// Valid values are "private" and "public".
	DefaultVisibility string `yaml:"defaultVisibility" env:"SONARQUBE_DEFAULT_VISIBILITY"`

	// QualityGateTimeout is the maximum time to wait for quality gate status.
	QualityGateTimeout time.Duration `yaml:"qualityGateTimeout" env:"SONARQUBE_QUALITY_GATE_TIMEOUT"`

	// DisableBranchAnalysis disables branch analysis for Community Edition compatibility.
	// When true, sonar.branch.name parameter will not be used.
	DisableBranchAnalysis bool `yaml:"disableBranchAnalysis" env:"SONARQUBE_DISABLE_BRANCH_ANALYSIS"`
}

// ScannerConfig represents the configuration for sonar-scanner.
// This struct defines the settings needed to download, configure, and
// execute the sonar-scanner tool for code analysis.
type ScannerConfig struct {
	// ScannerURL is the URL where the sonar-scanner can be downloaded.
	ScannerURL string `yaml:"scannerUrl" env:"SONARQUBE_SCANNER_URL"`

	// ScannerVersion is the version of sonar-scanner to use.
	ScannerVersion string `yaml:"scannerVersion" env:"SONARQUBE_SCANNER_VERSION"`

	// JavaOpts are the JVM options to pass to the sonar-scanner.
	JavaOpts string `yaml:"javaOpts" env:"SONARQUBE_JAVA_OPTS"`

	// Properties are additional properties to pass to the sonar-scanner.
	Properties map[string]string `yaml:"properties"`

	// Timeout is the maximum time to wait for sonar-scanner execution.
	Timeout time.Duration `yaml:"timeout" env:"SONARQUBE_SCANNER_TIMEOUT"`

	// WorkDir is the working directory for sonar-scanner execution.
	WorkDir string `yaml:"workDir" env:"SONARQUBE_SCANNER_WORK_DIR"`

	// TempDir is the temporary directory for sonar-scanner files.
	TempDir string `yaml:"tempDir" env:"SONARQUBE_SCANNER_TEMP_DIR"`
}

// GetDefaultSonarQubeConfig returns the default SonarQube configuration.
// This function provides sensible default values for SonarQube settings.
func GetDefaultSonarQubeConfig() *SonarQubeConfig {
	return &SonarQubeConfig{
		URL:                   "http://localhost:9000",
		Timeout:               30 * time.Second,
		RetryAttempts:         3,
		RetryDelay:            5 * time.Second,
		ProjectPrefix:         "apk-ci",
		DefaultVisibility:     "private",
		QualityGateTimeout:    300 * time.Second,
		DisableBranchAnalysis: true, // Default to true for Community Edition compatibility
	}
}

// Validate validates the SonarQube configuration.
// This method checks if the configuration values are valid and returns an error if not.
func (s *SonarQubeConfig) Validate() error {
	if s.URL == "" {
		return fmt.Errorf("SonarQube URL is required")
	}

	if s.Token == "" {
		return fmt.Errorf("SonarQube token is required")
	}

	if s.Timeout <= 0 {
		return fmt.Errorf("SonarQube timeout must be positive")
	}

	if s.RetryAttempts < 0 {
		return fmt.Errorf("SonarQube retry attempts cannot be negative")
	}

	if s.RetryDelay <= 0 {
		return fmt.Errorf("SonarQube retry delay must be positive")
	}

	if s.DefaultVisibility != "private" && s.DefaultVisibility != "public" {
		return fmt.Errorf("SonarQube default visibility must be 'private' or 'public'")
	}

	if s.QualityGateTimeout <= 0 {
		return fmt.Errorf("SonarQube quality gate timeout must be positive")
	}

	return nil
}

// Validate validates the Scanner configuration.
// This method checks if the configuration values are valid and returns an error if not.
func (s *ScannerConfig) Validate() error {
	if s.ScannerURL == "" {
		return fmt.Errorf("scanner URL is required")
	}

	if s.ScannerVersion == "" {
		return fmt.Errorf("scanner version is required")
	}

	if s.Timeout <= 0 {
		return fmt.Errorf("scanner timeout must be positive")
	}

	return nil
}

// GetSonarQubeConfig returns the SonarQube configuration, loading it from AppConfig, SecretConfig and environment variables.
// This method loads the configuration from AppConfig and SecretConfig first, then from environment variables and validates it.
func GetSonarQubeConfig(_ *slog.Logger, cfg *Config) (*SonarQubeConfig, error) {
	config := GetDefaultSonarQubeConfig()

	// Load from AppConfig if available
	if cfg.AppConfig != nil {
		// Copy values from AppConfig.SonarQube to config
		if cfg.AppConfig.SonarQube.URL != "" {
			config.URL = cfg.AppConfig.SonarQube.URL
		}
		if cfg.AppConfig.SonarQube.Token != "" {
			config.Token = cfg.AppConfig.SonarQube.Token
		}
		if cfg.AppConfig.SonarQube.Timeout > 0 {
			config.Timeout = cfg.AppConfig.SonarQube.Timeout
		}
		if cfg.AppConfig.SonarQube.RetryAttempts > 0 {
			config.RetryAttempts = cfg.AppConfig.SonarQube.RetryAttempts
		}
		if cfg.AppConfig.SonarQube.RetryDelay > 0 {
			config.RetryDelay = cfg.AppConfig.SonarQube.RetryDelay
		}
		if cfg.AppConfig.SonarQube.ProjectPrefix != "" {
			config.ProjectPrefix = cfg.AppConfig.SonarQube.ProjectPrefix
		}
		if cfg.AppConfig.SonarQube.DefaultVisibility != "" {
			config.DefaultVisibility = cfg.AppConfig.SonarQube.DefaultVisibility
		}
		if cfg.AppConfig.SonarQube.QualityGateTimeout > 0 {
			config.QualityGateTimeout = cfg.AppConfig.SonarQube.QualityGateTimeout
		}
	}

	// Load from SecretConfig if available (token has priority from secrets)
	if cfg.SecretConfig != nil && cfg.SecretConfig.SonarQube.Token != "" {
		config.Token = cfg.SecretConfig.SonarQube.Token
	}

	// Load from environment variables (highest priority)
	if err := cleanenv.ReadEnv(config); err != nil {
		return nil, fmt.Errorf("failed to read SonarQube config from environment: %w", err)
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid SonarQube configuration: %w", err)
	}

	return config, nil
}

// GetScannerConfig returns the Scanner configuration, loading it from AppConfig and environment variables.
// This method loads the configuration from AppConfig first, then from environment variables and validates it.
func GetScannerConfig(_ *slog.Logger, cfg *Config) (*ScannerConfig, error) {
	config := GetDefaultScannerConfig()

	// Load from AppConfig if available
	if cfg.AppConfig != nil {
		// Copy values from AppConfig.Scanner to config
		if cfg.AppConfig.Scanner.ScannerURL != "" {
			config.ScannerURL = cfg.AppConfig.Scanner.ScannerURL
		}
		if cfg.AppConfig.Scanner.ScannerVersion != "" {
			config.ScannerVersion = cfg.AppConfig.Scanner.ScannerVersion
		}
		if cfg.AppConfig.Scanner.JavaOpts != "" {
			config.JavaOpts = cfg.AppConfig.Scanner.JavaOpts
		}
		if cfg.AppConfig.Scanner.Properties != nil {
			config.Properties = cfg.AppConfig.Scanner.Properties
		}
		if cfg.AppConfig.Scanner.Timeout > 0 {
			config.Timeout = cfg.AppConfig.Scanner.Timeout
		}
		if cfg.AppConfig.Scanner.WorkDir != "" {
			config.WorkDir = cfg.AppConfig.Scanner.WorkDir
		}
		if cfg.AppConfig.Scanner.TempDir != "" {
			config.TempDir = cfg.AppConfig.Scanner.TempDir
		}
	}

	// Load from environment variables (highest priority)
	if err := cleanenv.ReadEnv(config); err != nil {
		return nil, fmt.Errorf("failed to read Scanner config from environment: %w", err)
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Scanner configuration: %w", err)
	}

	return config, nil
}

// GetDefaultScannerConfig returns the default sonar-scanner configuration.
// This function provides sensible default values for scanner settings.
func GetDefaultScannerConfig() *ScannerConfig {
	return &ScannerConfig{
		ScannerURL:     "https://binaries.sonarsource.com/Distribution/sonar-scanner-cli/sonar-scanner-cli-4.8.0.2856-linux.zip",
		ScannerVersion: "4.8.0.2856",
		JavaOpts:       "-Xmx2g",
		Properties:     make(map[string]string),
		Timeout:        600 * time.Second,
		WorkDir:        "/tmp/apk-ci",
		TempDir:        "/tmp/apk-ci/scanner/temp",
	}
}
