// Package sonarqube provides implementation of SonarScanner service layer.
// This package contains the business logic for managing sonar-scanner,
// including lifecycle management, configuration validation, and resource management.
package sonarqube

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/entity/sonarqube"
)

// SonarScannerService provides business logic for sonar-scanner operations.
// This service layer implements lifecycle management, configuration validation,
// and resource management for the sonar-scanner.
type SonarScannerService struct {
	// entity is the SonarScanner entity for low-level operations
	entity sonarqube.SonarScannerInterface
	
	// config contains the scanner configuration settings
	config *config.ScannerConfig
	
	// logger is the structured logger for this service
	logger *slog.Logger
}

// NewSonarScannerService creates a new instance of SonarScannerService.
// This function initializes the service with the provided SonarScanner entity
// and configuration.
//
// Parameters:
//   - entity: SonarScanner entity for low-level operations
//   - cfg: scanner configuration settings
//   - logger: structured logger instance
//
// Returns:
//   - *SonarScannerService: initialized scanner service
func NewSonarScannerService(entity sonarqube.SonarScannerInterface, cfg *config.ScannerConfig, logger *slog.Logger) *SonarScannerService {
	return &SonarScannerService{
		entity: entity,
		config: cfg,
		logger: logger,
	}
}

// DownloadScanner clones the sonar-scanner repository from the configured URL.
// This method clones the scanner repository using git clone with the specified version tag.
//
// Parameters:
//   - ctx: context for the operation
//   - scannerUrl: URL of the scanner repository
//   - scannerVersion: version tag to clone
//
// Returns:
//   - string: path to the directory where the scanner was cloned
//   - error: error if clone fails
func (s *SonarScannerService) DownloadScanner(ctx context.Context, scannerURL string, scannerVersion string) (string, error) {
	s.logger.Debug("Cloning sonar-scanner repository", "scannerURL", scannerURL, "scannerVersion", scannerVersion)
	
	// Clone scanner repository with specified version
	cloneDir, err := s.entity.Download(ctx, scannerURL, scannerVersion)
	if err != nil {
		s.logger.Error("Failed to clone sonar-scanner repository", "error", err)
		return "", fmt.Errorf("failed to clone scanner: %w", err)
	}
	
	s.logger.Debug("Sonar-scanner repository cloned successfully", "cloneDir", cloneDir)
	return cloneDir, nil
}

// ConfigureScanner configures the sonar-scanner with the provided configuration.
// This method configures the scanner with the provided settings.
//
// Parameters:
//   - config: scanner configuration
//
// Returns:
//   - error: error if configuration fails
func (s *SonarScannerService) ConfigureScanner(config *sonarqube.ScannerConfig) error {
	s.logger.Debug("Configuring sonar-scanner")
	
	// Configure scanner
	if err := s.entity.Configure(config); err != nil {
		s.logger.Error("Failed to configure sonar-scanner", "error", err)
		return fmt.Errorf("failed to configure scanner: %w", err)
	}
	
	s.logger.Debug("Sonar-scanner configured successfully")
	return nil
}

// SetProperty sets a property in the scanner configuration.
// This method sets a single property in the scanner configuration.
//
// Parameters:
//   - key: property key
//   - value: property value
func (s *SonarScannerService) SetProperty(key, value string) {
	s.entity.SetProperty(key, value)
}

// GetProperty retrieves a property from the scanner configuration.
// This method retrieves a single property from the scanner configuration.
//
// Parameters:
//   - key: property key
//
// Returns:
//   - string: property value
func (s *SonarScannerService) GetProperty(key string) string {
	return s.entity.GetProperty(key)
}

// ValidateScannerConfig validates the current scanner configuration.
// This method validates the scanner configuration, checking that
// required properties are set and paths are valid.
//
// Returns:
//   - error: error if configuration is invalid
func (s *SonarScannerService) ValidateScannerConfig() error {
	s.logger.Debug("Validating scanner configuration")
	
	// Validate configuration
	if err := s.entity.ValidateConfig(); err != nil {
		s.logger.Error("Scanner configuration validation failed", "error", err)
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	
	s.logger.Debug("Scanner configuration is valid")
	return nil
}

// InitializeScanner initializes the scanner, preparing it for execution.
// This method performs any necessary initialization steps before
// executing the scanner.
//
// Returns:
//   - error: error if initialization fails
func (s *SonarScannerService) InitializeScanner() error {
	s.logger.Debug("Initializing scanner")
	
	// Initialize scanner
	if err := s.entity.Initialize(); err != nil {
		s.logger.Error("Failed to initialize scanner", "error", err)
		return fmt.Errorf("scanner initialization failed: %w", err)
	}
	
	s.logger.Debug("Scanner initialized successfully")
	return nil
}

// ExecuteScanner executes the sonar-scanner with the provided context.
// This method executes the sonar-scanner with the current configuration
// and returns the scan result.
//
// Parameters:
//   - ctx: context for the execution
//
// Returns:
//   - *sonarqube.ScanResult: scan result
//   - error: error if execution fails
func (s *SonarScannerService) ExecuteScanner(ctx context.Context) (*sonarqube.ScanResult, error) {
	s.logger.Debug("Executing sonar-scanner")
	
	// Execute scanner
	result, err := s.entity.Execute(ctx)
	if err != nil {
		s.logger.Error("Failed to execute scanner", "error", err)
		return nil, fmt.Errorf("scanner execution failed: %w", err)
	}
	
	s.logger.Debug("Sonar-scanner executed successfully", "success", result.Success, "duration", result.Duration)
	return result, nil
}

// CleanupScanner cleans up resources used by the scanner.
// This method cleans up temporary files and directories used by the scanner.
//
// Returns:
//   - error: error if cleanup fails
func (s *SonarScannerService) CleanupScanner() error {
	s.logger.Debug("Cleaning up scanner resources")
	
	// Cleanup scanner resources
	if err := s.entity.Cleanup(); err != nil {
		s.logger.Error("Failed to cleanup scanner resources", "error", err)
		return fmt.Errorf("scanner cleanup failed: %w", err)
	}
	
	s.logger.Debug("Scanner resources cleaned up successfully")
	return nil
}

// ToDo: необходимо дополнительно реализовать следующий функционал:
// - Add unit tests for all methods
// - Implement more sophisticated configuration management
// - Add support for different scanner versions
// - Implement better error handling and recovery
// - Add progress reporting during download and execution
//
// Ссылки на пункты плана и требований:
// - tasks.md: 3.3
// - requirements.md: 10.1, 10.2, 10.3