// Package sonarqube provides service layer for SonarQube branch scanning operations.
package sonarqube

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/entity/sonarqube"
)

const (
	// StatusFailed represents the failed status for branch scanning operations
	StatusFailed = "FAILED"
)

// BranchScannerService provides high-level orchestration for branch scanning operations.
// It coordinates between different entities and handles business logic.
type BranchScannerService struct {
	branchScanner *sonarqube.BranchScannerEntity
	config        *config.ScannerConfig
	logger        *slog.Logger
	retryConfig   *RetryConfig
}

// BranchScannerInterface defines the interface for branch scanning operations.
type BranchScannerInterface interface {
	// ScanBranch performs a complete branch scan with validation and error handling
	ScanBranch(ctx context.Context, request *BranchScanRequest) (*BranchScanResponse, error)
	
	// ValidateBranchForScanning validates if a branch can be scanned
	ValidateBranchForScanning(ctx context.Context, branchName, projectPath string) error
	
	// GetBranchScanHistory retrieves scan history for a branch
	GetBranchScanHistory(ctx context.Context, projectKey, branchName string) ([]*sonarqube.BranchScanResult, error)
	
	// CancelBranchScan cancels an ongoing branch scan
	CancelBranchScan(ctx context.Context, scanID string) error
}

// BranchScanRequest represents a request to scan a branch.
type BranchScanRequest struct {
	ProjectKey   string            `json:"project_key"`
	ProjectName  string            `json:"project_name"`
	ProjectPath  string            `json:"project_path"`
	BranchName   string            `json:"branch_name"`
	Owner        string            `json:"owner"`
	Repository   string            `json:"repository"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Options      *ScanOptions      `json:"options,omitempty"`
}

// BranchScanResponse represents the response from a branch scan operation.
type BranchScanResponse struct {
	ScanID         string                        `json:"scan_id"`
	Status         string                        `json:"status"`
	BranchMetadata *sonarqube.BranchMetadata     `json:"branch_metadata"`
	ScanResult     *sonarqube.BranchScanResult   `json:"scan_result"`
	Errors         []string                      `json:"errors,omitempty"`
	Warnings       []string                      `json:"warnings,omitempty"`
	Duration       time.Duration                 `json:"duration"`
	Timestamp      time.Time                     `json:"timestamp"`
}

// ScanOptions provides additional options for branch scanning.
type ScanOptions struct {
	SkipValidation    bool              `json:"skip_validation,omitempty"`
	ForceRescan       bool              `json:"force_rescan,omitempty"`
	Timeout           time.Duration     `json:"timeout,omitempty"`
	RetryAttempts     int               `json:"retry_attempts,omitempty"`
	CustomProperties  map[string]string `json:"custom_properties,omitempty"`
	QualityGateCheck  bool              `json:"quality_gate_check,omitempty"`
}

// RetryConfig defines retry behavior for failed operations.
type RetryConfig struct {
	MaxAttempts     int           `json:"max_attempts"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffFactor   float64       `json:"backoff_factor"`
	RetryableErrors []string      `json:"retryable_errors"`
}

// NewBranchScannerService creates a new instance of BranchScannerService.
func NewBranchScannerService(
	branchScanner *sonarqube.BranchScannerEntity,
	config *config.ScannerConfig,
	logger *slog.Logger,
) *BranchScannerService {
	return &BranchScannerService{
		branchScanner: branchScanner,
		config:        config,
		logger:        logger,
		retryConfig:   getDefaultRetryConfig(),
	}
}

// ScanBranch performs a complete branch scan with validation and error handling.
func (s *BranchScannerService) ScanBranch(ctx context.Context, request *BranchScanRequest) (*BranchScanResponse, error) {
	start := time.Now()
	scanID := generateScanID(request.ProjectKey, request.BranchName)
	
	s.logger.Info("Starting branch scan",
		"scan_id", scanID,
		"project_key", request.ProjectKey,
		"branch_name", request.BranchName,
	)
	
	response := &BranchScanResponse{
		ScanID:    scanID,
		Status:    "STARTED",
		Timestamp: start,
		Errors:    []string{},
		Warnings:  []string{},
	}
	
	// Validate request
	if err := s.validateScanRequest(request); err != nil {
		response.Status = StatusFailed
		response.Errors = append(response.Errors, fmt.Sprintf("Request validation failed: %v", err))
		return response, err
	}
	
	// Pre-scan validation
	if request.Options == nil || !request.Options.SkipValidation {
		if err := s.ValidateBranchForScanning(ctx, request.BranchName, request.ProjectPath); err != nil {
			response.Status = StatusFailed
			response.Errors = append(response.Errors, fmt.Sprintf("Branch validation failed: %v", err))
			return response, err
		}
	}
	
	// Execute scan with retry logic
	scanResult, err := s.executeScanWithRetry(ctx, request)
	if err != nil {
		response.Status = StatusFailed
		response.Errors = append(response.Errors, fmt.Sprintf("Scan execution failed: %v", err))
		response.Duration = time.Since(start)
		return response, err
	}
	
	// Process scan results
	response.ScanResult = scanResult
	response.BranchMetadata = scanResult.BranchMetadata
	response.Status = "COMPLETED"
	response.Duration = time.Since(start)
	
	// Perform quality gate check if requested
	if request.Options != nil && request.Options.QualityGateCheck {
		if err := s.performQualityGateCheck(ctx, scanResult); err != nil {
			response.Warnings = append(response.Warnings, fmt.Sprintf("Quality gate check failed: %v", err))
		}
	}
	
	s.logger.Info("Branch scan completed",
		"scan_id", scanID,
		"status", response.Status,
		"duration", response.Duration,
	)
	
	return response, nil
}

// ValidateBranchForScanning validates if a branch can be scanned.
func (s *BranchScannerService) ValidateBranchForScanning(_ context.Context, branchName, projectPath string) error {
	if branchName == "" {
		return fmt.Errorf("branch name cannot be empty")
	}
	
	if projectPath == "" {
		return fmt.Errorf("project path cannot be empty")
	}
	
	// TODO: Add more sophisticated validation logic
	// - Check if branch exists in Git repository
	// - Validate branch naming conventions
	// - Check if project path is a valid Git repository
	// - Verify SonarQube project exists
	
	s.logger.Debug("Branch validation passed",
		"branch_name", branchName,
		"project_path", projectPath,
	)
	
	return nil
}

// GetBranchScanHistory retrieves scan history for a branch.
func (s *BranchScannerService) GetBranchScanHistory(_ context.Context, projectKey, branchName string) ([]*sonarqube.BranchScanResult, error) {
	s.logger.Debug("Retrieving branch scan history",
		"project_key", projectKey,
		"branch_name", branchName,
	)
	
	// TODO: Implement scan history retrieval
	// - Query SonarQube API for analysis history
	// - Filter by branch name
	// - Return formatted results
	
	return []*sonarqube.BranchScanResult{}, nil
}

// CancelBranchScan cancels an ongoing branch scan.
func (s *BranchScannerService) CancelBranchScan(_ context.Context, scanID string) error {
	s.logger.Info("Cancelling branch scan", "scan_id", scanID)
	
	// TODO: Implement scan cancellation logic
	// - Stop ongoing scanner process
	// - Clean up temporary files
	// - Update scan status
	
	return nil
}

// validateScanRequest validates the scan request parameters.
func (s *BranchScannerService) validateScanRequest(request *BranchScanRequest) error {
	if request == nil {
		return fmt.Errorf("scan request cannot be nil")
	}
	
	if request.ProjectKey == "" {
		return fmt.Errorf("project key cannot be empty")
	}
	
	if request.ProjectName == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	
	if request.ProjectPath == "" {
		return fmt.Errorf("project path cannot be empty")
	}
	
	if request.BranchName == "" {
		return fmt.Errorf("branch name cannot be empty")
	}
	
	if request.Owner == "" {
		return fmt.Errorf("owner cannot be empty")
	}
	
	if request.Repository == "" {
		return fmt.Errorf("repository cannot be empty")
	}
	
	return nil
}

// executeScanWithRetry executes the branch scan with retry logic.
func (s *BranchScannerService) executeScanWithRetry(ctx context.Context, request *BranchScanRequest) (*sonarqube.BranchScanResult, error) {
	var lastErr error
	maxAttempts := s.retryConfig.MaxAttempts
	
	if request.Options != nil && request.Options.RetryAttempts > 0 {
		maxAttempts = request.Options.RetryAttempts
	}
	
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		s.logger.Debug("Executing scan attempt",
			"attempt", attempt,
			"max_attempts", maxAttempts,
		)
		
		// Execute scan
		result, err := s.branchScanner.ScanBranch(ctx, request.BranchName, request.ProjectKey)
		if err == nil {
			return result, nil
		}
		
		lastErr = err
		
		// Check if error is retryable
		if !s.isRetryableError(err) {
			s.logger.Error("Non-retryable error encountered", "error", err)
			return nil, err
		}
		
		// Calculate delay for next attempt
		if attempt < maxAttempts {
			delay := s.calculateRetryDelay(attempt)
			s.logger.Warn("Scan attempt failed, retrying",
				"attempt", attempt,
				"error", err,
				"retry_delay", delay,
			)
			
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				// Continue to next attempt
			}
		}
	}
	
	return nil, fmt.Errorf("scan failed after %d attempts: %w", maxAttempts, lastErr)
}

// performQualityGateCheck performs quality gate validation on scan results.
func (s *BranchScannerService) performQualityGateCheck(_ context.Context, result *sonarqube.BranchScanResult) error {
	if result == nil {
		return fmt.Errorf("scan result not available")
	}
	
	// TODO: Implement quality gate check logic
	// - Check quality gate status
	// - Validate against configured thresholds
	// - Generate quality report
	
	s.logger.Debug("Quality gate check completed",
		"scan_id", result.ScanResult.AnalysisID,
		"project_key", result.ScanResult.ProjectKey,
	)
	
	return nil
}

// isRetryableError checks if an error is retryable based on configuration.
func (s *BranchScannerService) isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	
	errorMsg := err.Error()
	for _, retryableError := range s.retryConfig.RetryableErrors {
		if contains(errorMsg, retryableError) {
			return true
		}
	}
	
	// Default retryable errors
	retryablePatterns := []string{
		"connection refused",
		"timeout",
		"temporary failure",
		"network error",
		"service unavailable",
	}
	
	for _, pattern := range retryablePatterns {
		if contains(errorMsg, pattern) {
			return true
		}
	}
	
	return false
}

// calculateRetryDelay calculates the delay for the next retry attempt.
func (s *BranchScannerService) calculateRetryDelay(attempt int) time.Duration {
	delay := s.retryConfig.InitialDelay
	
	// Apply exponential backoff
	for i := 1; i < attempt; i++ {
		delay = time.Duration(float64(delay) * s.retryConfig.BackoffFactor)
	}
	
	// Cap at maximum delay
	if delay > s.retryConfig.MaxDelay {
		delay = s.retryConfig.MaxDelay
	}
	
	return delay
}

// Helper functions

// generateScanID generates a unique scan ID for tracking purposes.
func generateScanID(projectKey, branchName string) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s-%s-%d", projectKey, branchName, timestamp)
}

// getDefaultRetryConfig returns the default retry configuration.
func getDefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		RetryableErrors: []string{
			"connection",
			"timeout",
			"temporary",
		},
	}
}

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		(s == substr || 
		 len(s) > len(substr) && 
		 (s[:len(substr)] == substr || 
		  s[len(s)-len(substr):] == substr ||
		  findSubstring(s, substr)))
}

// findSubstring performs a simple substring search.
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}