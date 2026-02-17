// Package sonarqube provides implementation of SonarScanner entity.
// This package contains the low-level implementation for managing sonar-scanner,
// including downloading, configuration, and execution.
package sonarqube

import (
	"github.com/Kargones/apk-ci/internal/constants"
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/Kargones/apk-ci/internal/config"
)

// MaxScanRetries defines the maximum number of scan retries when encountering BSL file errors
const MaxScanRetries = 10

// SonarScannerEntity represents the low-level interaction with sonar-scanner.
type SonarScannerEntity struct {
	config        *config.ScannerConfig
	logger        *slog.Logger
	scannerPath   string
	properties    map[string]string
	workDir       string
	tempDir       string
	excludedFiles []string
	retryCount    int
}

// NewSonarScannerEntity creates a new instance of SonarScannerEntity.
func NewSonarScannerEntity(cfg *config.ScannerConfig, logger *slog.Logger) *SonarScannerEntity {
	return &SonarScannerEntity{
		config:        cfg,
		logger:        logger,
		properties:    make(map[string]string),
		workDir:       cfg.WorkDir,
		tempDir:       cfg.TempDir,
		excludedFiles: make([]string, 0),
		retryCount:    0,
	}
}

// Configure configures the sonar-scanner with the provided configuration.
func (s *SonarScannerEntity) Configure(config *ScannerConfig) error {
	s.logger.Debug("Configuring sonar-scanner")

	if config.Properties != nil {
		for key, value := range config.Properties {
			s.properties[key] = value
		}
	}

	if config.WorkDir != "" {
		s.workDir = config.WorkDir
		if err := os.MkdirAll(s.workDir, constants.DirPermStandard); err != nil {
			return fmt.Errorf("failed to create working directory: %w", err)
		}
	}

	if config.TempDir != "" {
		s.tempDir = config.TempDir
		if err := os.MkdirAll(s.tempDir, constants.DirPermStandard); err != nil {
			return fmt.Errorf("failed to create temporary directory: %w", err)
		}
	}

	s.logger.Debug("Sonar-scanner configured successfully")
	return nil
}

// SetProperty sets a property in the scanner configuration.
func (s *SonarScannerEntity) SetProperty(key, value string) {
	s.properties[key] = value
}

// GetProperty retrieves a property from the scanner configuration.
func (s *SonarScannerEntity) GetProperty(key string) string {
	return s.properties[key]
}

// ValidateConfig validates the current scanner configuration.
func (s *SonarScannerEntity) ValidateConfig() error {
	s.logger.Debug("Validating scanner configuration")

	if s.scannerPath == "" {
		return &ValidationError{
			Field:   "scannerPath",
			Message: "scanner path is not set, scanner may not be downloaded",
		}
	}

	if _, err := os.Stat(s.scannerPath); os.IsNotExist(err) {
		return &ValidationError{
			Field:   "scannerPath",
			Message: "scanner executable does not exist",
		}
	}

	if s.workDir != "" {
		if _, err := os.Stat(s.workDir); os.IsNotExist(err) {
			return &ValidationError{
				Field:   "workDir",
				Message: "working directory does not exist",
			}
		}
	}

	s.logger.Debug("Scanner configuration is valid")
	return nil
}

// Initialize initializes the scanner, preparing it for execution.
func (s *SonarScannerEntity) Initialize() error {
	s.logger.Debug("Initializing scanner")

	if err := s.ValidateConfig(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	s.logger.Debug("Scanner initialized successfully")
	return nil
}

// Execute runs the sonar-scanner with automatic retry logic for BSL file errors.
func (s *SonarScannerEntity) Execute(ctx context.Context) (*ScanResult, error) {
	s.logger.Debug("Starting sonar-scanner with retry logic", "maxRetries", MaxScanRetries)

	s.retryCount = 0
	s.ClearExclusions()

	for s.retryCount <= MaxScanRetries {
		s.logger.Debug("Executing scan attempt", "attempt", s.retryCount+1, "excludedFiles", len(s.excludedFiles))

		result, err := s.executeOnce(ctx)

		if err == nil {
			s.logger.Info("Scan completed successfully", "attempt", s.retryCount+1, "excludedFiles", len(s.excludedFiles))
			return result, nil
		}

		if scannerErr, ok := err.(*ScannerError); ok {
			problematicFiles := s.ExtractProblematicBSLFiles(scannerErr.Output)

			if len(problematicFiles) > 0 {
				s.logger.Warn("BSL tokenization errors detected",
					"attempt", s.retryCount+1,
					"problematicFiles", problematicFiles)

				if s.retryCount >= MaxScanRetries {
					s.logger.Error("Maximum retry attempts exceeded",
						"maxRetries", MaxScanRetries,
						"totalExcludedFiles", len(s.excludedFiles),
						"excludedFiles", s.excludedFiles)

					return result, fmt.Errorf("scan failed after %d retry attempts due to BSL file errors. "+
						"Problematic files: %v. All excluded files: %v",
						MaxScanRetries, problematicFiles, s.excludedFiles)
				}

				s.AddFilesToExclusions(problematicFiles)
				s.retryCount++

				s.logger.Info("Added files to exclusions, retrying scan",
					"newExclusions", problematicFiles,
					"totalExcluded", len(s.excludedFiles),
					"nextAttempt", s.retryCount+1)

				continue
			}
		}

		s.logger.Error("Scan failed with non-recoverable error", "error", err)
		return result, err
	}

	return nil, fmt.Errorf("unexpected error: exceeded maximum retry attempts")
}

// Cleanup cleans up resources used by the scanner.
func (s *SonarScannerEntity) Cleanup() error {
	s.logger.Debug("Cleaning up scanner resources")

	if s.tempDir != "" && s.tempDir != os.TempDir() {
		err := os.RemoveAll(s.tempDir)
		if err != nil {
			return fmt.Errorf("failed to remove temporary directory: %w", err)
		}
	}

	s.logger.Debug("Scanner resources cleaned up successfully")
	return nil
}
