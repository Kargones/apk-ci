// Package sonarqube provides service layer for SonarScanner management.
// This package contains high-level service implementation for managing
// scanner lifecycle, validation, and resource management.
package sonarqube

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
)

// SonarScannerService provides high-level service for managing SonarScanner operations.
// It handles scanner lifecycle, validation, resource management, and cleanup.
type SonarScannerService struct {
	// scanner is the underlying scanner entity
	scanner SonarScannerInterface
	
	// config holds the scanner configuration
	config *config.ScannerConfig
	
	// logger for service operations
	logger *slog.Logger
	
	// mu protects concurrent access to the service
	mu sync.RWMutex
	
	// isInitialized tracks if the service has been initialized
	isInitialized bool
	
	// isRunning tracks if a scan is currently running
	isRunning bool
	
	// lastScanResult holds the result of the last scan
	lastScanResult *ScanResult
	
	// lastScanTime holds the timestamp of the last scan
	lastScanTime time.Time
	
	// resourceCleanupFuncs holds cleanup functions for resources
	resourceCleanupFuncs []func() error
}

// NewSonarScannerService creates a new SonarScannerService instance.
// It initializes the service with the provided configuration and logger.
//
// Parameters:
//   - cfg: scanner configuration
//   - logger: logger for service operations
//
// Returns:
//   - *SonarScannerService: new service instance
func NewSonarScannerService(cfg *config.ScannerConfig, logger *slog.Logger) *SonarScannerService {
	return &SonarScannerService{
		config:               cfg,
		logger:               logger,
		isInitialized:        false,
		isRunning:            false,
		resourceCleanupFuncs: make([]func() error, 0),
	}
}

// Initialize initializes the scanner service and validates configuration.
// This method must be called before using the service for scanning operations.
//
// Returns:
//   - error: error if initialization fails
func (s *SonarScannerService) Initialize() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.isInitialized {
		s.logger.Debug("Scanner service already initialized")
		return nil
	}
	
	s.logger.Debug("Initializing scanner service")
	
	// Validate configuration
	if err := s.validateConfiguration(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	
	// Create scanner entity
	s.scanner = NewSonarScannerEntity(s.config, s.logger)
	
	// Initialize the scanner
	if err := s.scanner.Initialize(); err != nil {
		return fmt.Errorf("scanner initialization failed: %w", err)
	}
	
	// Register cleanup function for scanner
	s.addResourceCleanup(func() error {
		return s.scanner.Cleanup()
	})
	
	s.isInitialized = true
	s.logger.Debug("Scanner service initialized successfully")
	
	return nil
}

// validateConfiguration validates the scanner configuration.
// This method checks all required configuration parameters and their validity.
//
// Returns:
//   - error: error if validation fails
func (s *SonarScannerService) validateConfiguration() error {
	if s.config == nil {
		return fmt.Errorf("scanner configuration is nil")
	}
	
	// Validate required fields
	if s.config.ScannerURL == "" {
		return fmt.Errorf("scanner URL is required")
	}
	
	if s.config.ScannerVersion == "" {
		return fmt.Errorf("scanner version is required")
	}
	
	// Validate timeout
	if s.config.Timeout <= 0 {
		s.logger.Warn("Invalid timeout, using default", "timeout", s.config.Timeout)
		s.config.Timeout = 30 * time.Minute // Default timeout
	}
	
	// Validate working directory
	if s.config.WorkDir == "" {
		s.logger.Warn("Working directory not specified, using current directory")
		s.config.WorkDir = "."
	}
	
	// Validate temporary directory
	if s.config.TempDir == "" {
		s.logger.Warn("Temporary directory not specified, using system default")
		s.config.TempDir = "/tmp"
	}
	
	// Validate properties
	if s.config.Properties == nil {
		s.config.Properties = make(map[string]string)
	}
	
	// Ensure required properties are set
	if _, exists := s.config.Properties["sonar.host.url"]; !exists {
		if s.config.ScannerURL != "" {
			s.config.Properties["sonar.host.url"] = s.config.ScannerURL
		} else {
			return fmt.Errorf("sonar.host.url property is required")
		}
	}
	
	s.logger.Debug("Configuration validation completed successfully")
	return nil
}

// Scan executes a scan with the provided context and properties.
// This method handles the complete scan lifecycle including validation,
// execution, and cleanup.
//
// Parameters:
//   - ctx: context for the scan operation
//   - properties: additional properties for the scan
//
// Returns:
//   - *ScanResult: scan result
//   - error: error if scan fails
func (s *SonarScannerService) Scan(ctx context.Context, properties map[string]string) (*ScanResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.isInitialized {
		return nil, fmt.Errorf("service not initialized")
	}
	
	if s.isRunning {
		return nil, fmt.Errorf("scan already in progress")
	}
	
	s.isRunning = true
	defer func() {
		s.isRunning = false
	}()
	
	s.logger.Debug("Starting scan operation")
	
	// Set additional properties
	for key, value := range properties {
		s.scanner.SetProperty(key, value)
	}
	
	// Execute scan with timeout
	startTime := time.Now()
	result, err := s.scanner.Execute(ctx)
	duration := time.Since(startTime)
	
	// Store scan result and time
	s.lastScanResult = result
	s.lastScanTime = startTime
	
	if err != nil {
		s.logger.Error("Scan failed", "error", err, "duration", duration)
		return result, fmt.Errorf("scan execution failed: %w", err)
	}
	
	s.logger.Debug("Scan completed successfully", 
		"duration", duration,
		"analysisId", result.AnalysisID,
		"projectKey", result.ProjectKey,
		"success", result.Success)
	
	return result, nil
}

// ScanWithTimeout executes a scan with a specific timeout.
// This method provides enhanced timeout control and process management.
//
// Parameters:
//   - ctx: context for the scan operation
//   - timeout: timeout duration for the scan
//   - properties: additional properties for the scan
//
// Returns:
//   - *ScanResult: scan result
//   - error: error if scan fails
func (s *SonarScannerService) ScanWithTimeout(ctx context.Context, timeout time.Duration, properties map[string]string) (*ScanResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.isInitialized {
		return nil, fmt.Errorf("service not initialized")
	}
	
	if s.isRunning {
		return nil, fmt.Errorf("scan already in progress")
	}
	
	s.isRunning = true
	defer func() {
		s.isRunning = false
	}()
	
	s.logger.Debug("Starting scan operation with timeout", "timeout", timeout)
	
	// Set additional properties
	for key, value := range properties {
		s.scanner.SetProperty(key, value)
	}
	
	// Execute scan with enhanced timeout
	startTime := time.Now()
	result, err := s.scanner.(*SonarScannerEntity).ExecuteWithTimeout(ctx, timeout)
	duration := time.Since(startTime)
	
	// Store scan result and time
	s.lastScanResult = result
	s.lastScanTime = startTime
	
	if err != nil {
		s.logger.Error("Scan with timeout failed", "error", err, "duration", duration, "timeout", timeout)
		return result, fmt.Errorf("scan execution failed: %w", err)
	}
	
	s.logger.Debug("Scan with timeout completed successfully", 
		"duration", duration,
		"timeout", timeout,
		"analysisId", result.AnalysisID,
		"projectKey", result.ProjectKey,
		"success", result.Success)
	
	return result, nil
}

// GetLastScanResult returns the result of the last scan operation.
//
// Returns:
//   - *ScanResult: last scan result, nil if no scan has been performed
//   - time.Time: timestamp of the last scan
func (s *SonarScannerService) GetLastScanResult() (*ScanResult, time.Time) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return s.lastScanResult, s.lastScanTime
}

// IsRunning returns whether a scan is currently in progress.
//
// Returns:
//   - bool: true if a scan is running, false otherwise
func (s *SonarScannerService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return s.isRunning
}

// IsInitialized returns whether the service has been initialized.
//
// Returns:
//   - bool: true if initialized, false otherwise
func (s *SonarScannerService) IsInitialized() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return s.isInitialized
}

// addResourceCleanup adds a cleanup function to be called during service shutdown.
//
// Parameters:
//   - cleanupFunc: function to call for cleanup
func (s *SonarScannerService) addResourceCleanup(cleanupFunc func() error) {
	s.resourceCleanupFuncs = append(s.resourceCleanupFuncs, cleanupFunc)
}

// Cleanup performs cleanup of all resources managed by the service.
// This method should be called when the service is no longer needed.
//
// Returns:
//   - error: error if cleanup fails
func (s *SonarScannerService) Cleanup() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.logger.Debug("Cleaning up scanner service resources")
	
	var errors []error
	
	// Execute all cleanup functions
	for i, cleanupFunc := range s.resourceCleanupFuncs {
		if err := cleanupFunc(); err != nil {
			s.logger.Warn("Cleanup function failed", "index", i, "error", err)
			errors = append(errors, err)
		}
	}
	
	// Reset service state
	s.isInitialized = false
	s.isRunning = false
	s.scanner = nil
	s.lastScanResult = nil
	s.resourceCleanupFuncs = make([]func() error, 0)
	
	if len(errors) > 0 {
		return fmt.Errorf("cleanup completed with %d errors: %v", len(errors), errors)
	}
	
	s.logger.Debug("Scanner service cleanup completed successfully")
	return nil
}

// GetConfig returns a copy of the current configuration.
//
// Returns:
//   - *config.ScannerConfig: copy of the configuration
func (s *SonarScannerService) GetConfig() *config.ScannerConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Return a copy to prevent external modifications
	configCopy := *s.config
	if s.config.Properties != nil {
		configCopy.Properties = make(map[string]string)
		for k, v := range s.config.Properties {
			configCopy.Properties[k] = v
		}
	}
	
	return &configCopy
}

// UpdateProperty updates a scanner property.
//
// Parameters:
//   - key: property key
//   - value: property value
//
// Returns:
//   - error: error if service is not initialized
func (s *SonarScannerService) UpdateProperty(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.isInitialized {
		return fmt.Errorf("service not initialized")
	}
	
	s.scanner.SetProperty(key, value)
	s.logger.Debug("Property updated", "key", key, "value", value)
	
	return nil
}

// GetProperty retrieves a scanner property value.
//
// Parameters:
//   - key: property key
//
// Returns:
//   - string: property value
//   - error: error if service is not initialized
func (s *SonarScannerService) GetProperty(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if !s.isInitialized {
		return "", fmt.Errorf("service not initialized")
	}
	
	return s.scanner.GetProperty(key), nil
}