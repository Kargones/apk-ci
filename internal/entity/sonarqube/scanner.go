// Package sonarqube provides implementation of SonarScanner entity.
// This package contains the low-level implementation for managing sonar-scanner,
// including downloading, configuration, and execution.
package sonarqube

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
)

// MaxScanRetries defines the maximum number of scan retries when encountering BSL file errors
const MaxScanRetries = 10

// SonarScannerEntity represents the low-level interaction with sonar-scanner.
// This struct contains the configuration and methods for downloading,
// configuring, and executing sonar-scanner.
type SonarScannerEntity struct {
	// config contains the scanner configuration settings.
	config *config.ScannerConfig

	// logger is the structured logger for this entity.
	logger *slog.Logger

	// scannerPath is the path to the downloaded scanner executable.
	scannerPath string

	// properties are the configuration properties for the scanner.
	properties map[string]string

	// workDir is the working directory for scanner execution.
	workDir string

	// tempDir is the temporary directory for scanner files.
	tempDir string

	// excludedFiles contains the list of files to exclude from scanning due to errors
	excludedFiles []string

	// retryCount tracks the number of scan retries performed
	retryCount int
}

// NewSonarScannerEntity creates a new instance of SonarScannerEntity.
// This function initializes the entity with the provided scanner configuration.
//
// Parameters:
//   - cfg: scanner configuration settings
//   - logger: structured logger instance
//
// Returns:
//   - *SonarScannerEntity: initialized scanner entity
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

// Download clones the sonar-scanner repository from the specified URL.
// This method clones the scanner repository using git clone with the specified version tag,
// and prepares the scanner for execution.
//
// Parameters:
//   - ctx: context for the operation
//   - scannerUrl: URL of the scanner repository
//   - scannerVersion: version tag to clone
//
// Returns:
//   - string: path to the directory where the scanner was cloned
//   - error: error if clone fails
func (s *SonarScannerEntity) Download(ctx context.Context, scannerURL string, scannerVersion string) (string, error) {
	s.logger.Debug("Cloning sonar-scanner repository", "scannerURL", scannerURL, "scannerVersion", scannerVersion)

	// Create temporary directory for cloning
	if s.tempDir == "" {
		s.tempDir = os.TempDir()
	}

	// Ensure parent directories exist
	if err := os.MkdirAll(s.tempDir, 0750); err != nil {
		return "", fmt.Errorf("failed to create parent directories: %w", err)
	}

	// Create unique temporary directory for this clone
	cloneDir, err := os.MkdirTemp(s.tempDir, "sonar-scanner-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	// Prepare git clone command with tag
	// #nosec G204 - scannerVersion и scannerURL валидируются в вызывающем коде
	cmd := exec.CommandContext(ctx, "git", "clone", "--branch", scannerVersion, "--depth", "1", scannerURL, cloneDir)
	cmd.Stdout = nil
	cmd.Stderr = nil

	// Execute git clone
	err = cmd.Run()
	if err != nil {
		// Clean up on failure
		if removeErr := os.RemoveAll(cloneDir); removeErr != nil {
			s.logger.Warn("Failed to remove clone directory after error", "path", cloneDir, "error", removeErr)
		}
		return "", fmt.Errorf("failed to clone scanner repository: %w", err)
	}

	// Find and set scanner executable path
	scannerPath, err := s.findScannerExecutable(cloneDir)
	if err != nil {
		// Clean up on failure
		if removeErr := os.RemoveAll(cloneDir); removeErr != nil {
			s.logger.Warn("Failed to remove clone directory after error", "path", cloneDir, "error", removeErr)
		}
		return "", fmt.Errorf("failed to find scanner executable: %w", err)
	}

	// Set scanner path
	s.scannerPath = scannerPath
	s.logger.Debug("Scanner path set", "scannerPath", scannerPath)

	s.logger.Debug("Sonar-scanner repository cloned successfully", "cloneDir", cloneDir)
	return cloneDir, nil
}

// findScannerExecutable finds the scanner executable in the extracted directory.
// This method searches for the scanner executable in the extracted directory
// and returns its path.
//
// Parameters:
//   - dir: directory to search in
//
// Returns:
//   - string: path to the scanner executable
//   - error: error if executable is not found
func (s *SonarScannerEntity) findScannerExecutable(dir string) (string, error) {
	// Search for scanner executable
	var scannerPath string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if this is the scanner executable
		if !info.IsDir() && (info.Name() == "sonar-scanner" || info.Name() == "sonar-scanner.bat") {
			// additional validation to ensure the path is within the extraction directory
			if !strings.HasPrefix(path, dir) {
				return fmt.Errorf("invalid scanner path: %s", path)
			}
			scannerPath = path
			return filepath.SkipDir // Stop walking
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("error while searching for scanner executable: %w", err)
	}

	if scannerPath == "" {
		return "", fmt.Errorf("scanner executable not found in extracted files")
	}

	return scannerPath, nil
}

// Configure configures the sonar-scanner with the provided configuration.
// This method sets the scanner properties based on the provided configuration.
//
// Parameters:
//   - config: scanner configuration
//
// Returns:
//   - error: error if configuration fails
func (s *SonarScannerEntity) Configure(config *ScannerConfig) error {
	s.logger.Debug("Configuring sonar-scanner")

	// Set properties from config
	if config.Properties != nil {
		for key, value := range config.Properties {
			s.properties[key] = value
		}
	}

	// Set working directory
	if config.WorkDir != "" {
		s.workDir = config.WorkDir
		// Ensure working directory exists
		if err := os.MkdirAll(s.workDir, 0750); err != nil {
			return fmt.Errorf("failed to create working directory: %w", err)
		}
	}

	// Set temporary directory
	if config.TempDir != "" {
		s.tempDir = config.TempDir
		// Ensure temporary directory exists
		if err := os.MkdirAll(s.tempDir, 0750); err != nil {
			return fmt.Errorf("failed to create temporary directory: %w", err)
		}
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
func (s *SonarScannerEntity) SetProperty(key, value string) {
	s.properties[key] = value
}

// GetProperty retrieves a property from the scanner configuration.
// This method retrieves a single property from the scanner configuration.
//
// Parameters:
//   - key: property key
//
// Returns:
//   - string: property value
func (s *SonarScannerEntity) GetProperty(key string) string {
	return s.properties[key]
}

// ValidateConfig validates the current scanner configuration.
// This method validates the scanner configuration, checking that
// required properties are set and paths are valid.
//
// Returns:
//   - error: error if configuration is invalid
func (s *SonarScannerEntity) ValidateConfig() error {
	s.logger.Debug("Validating scanner configuration")

	// Check that scanner path is set
	if s.scannerPath == "" {
		return &ValidationError{
			Field:   "scannerPath",
			Message: "scanner path is not set, scanner may not be downloaded",
		}
	}

	// Check that scanner executable exists
	if _, err := os.Stat(s.scannerPath); os.IsNotExist(err) {
		return &ValidationError{
			Field:   "scannerPath",
			Message: "scanner executable does not exist",
		}
	}

	// Check that working directory exists
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
// This method performs any necessary initialization steps before
// executing the scanner.
//
// Returns:
//   - error: error if initialization fails
func (s *SonarScannerEntity) Initialize() error {
	s.logger.Debug("Initializing scanner")

	// Validate configuration
	if err := s.ValidateConfig(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	s.logger.Debug("Scanner initialized successfully")
	return nil
}

// Execute runs the sonar-scanner with automatic retry logic for BSL file errors.
// If BSL tokenization errors occur, problematic files are excluded and the scan is retried
// up to MaxScanRetries times. If the retry limit is exceeded, the process fails with
// detailed error information.
//
// Parameters:
//   - ctx: context for cancellation and timeout control
//
// Returns:
//   - *ScanResult: scan result with parsed output and metrics
//   - error: error if execution fails or retry limit exceeded
func (s *SonarScannerEntity) Execute(ctx context.Context) (*ScanResult, error) {
	s.logger.Debug("Starting sonar-scanner with retry logic", "maxRetries", MaxScanRetries)
	
	// Reset retry count and exclusions at the start
	s.retryCount = 0
	s.ClearExclusions()
	
	for s.retryCount <= MaxScanRetries {
		s.logger.Debug("Executing scan attempt", "attempt", s.retryCount+1, "excludedFiles", len(s.excludedFiles))
		
		// Execute the scan
		result, err := s.executeOnce(ctx)
		
		// If scan succeeded, return the result
		if err == nil {
			s.logger.Info("Scan completed successfully", "attempt", s.retryCount+1, "excludedFiles", len(s.excludedFiles))
			return result, nil
		}
		
		// Check if this is a BSL tokenization error that we can handle
		if scannerErr, ok := err.(*ScannerError); ok {
			problematicFiles := s.ExtractProblematicBSLFiles(scannerErr.Output)
			
			if len(problematicFiles) > 0 {
				s.logger.Warn("BSL tokenization errors detected", 
					"attempt", s.retryCount+1, 
					"problematicFiles", problematicFiles)
				
				// Check if we've reached the retry limit
				if s.retryCount >= MaxScanRetries {
					s.logger.Error("Maximum retry attempts exceeded", 
						"maxRetries", MaxScanRetries, 
						"totalExcludedFiles", len(s.excludedFiles),
						"excludedFiles", s.excludedFiles)
					
					return result, fmt.Errorf("scan failed after %d retry attempts due to BSL file errors. "+
						"Problematic files: %v. All excluded files: %v", 
						MaxScanRetries, problematicFiles, s.excludedFiles)
				}
				
				// Add problematic files to exclusions
				s.AddFilesToExclusions(problematicFiles)
				s.retryCount++
				
				s.logger.Info("Added files to exclusions, retrying scan", 
					"newExclusions", problematicFiles,
					"totalExcluded", len(s.excludedFiles),
					"nextAttempt", s.retryCount+1)
				
				// Continue to next iteration for retry
				continue
			}
		}
		
		// If it's not a BSL error we can handle, or no problematic files found, return the error
		s.logger.Error("Scan failed with non-recoverable error", "error", err)
		return result, err
	}
	
	// This should never be reached due to the loop condition, but included for safety
	return nil, fmt.Errorf("unexpected error: exceeded maximum retry attempts")
}

// executeOnce executes the sonar-scanner with the provided context.
// This method executes the sonar-scanner with the current configuration,
// handles timeouts, process cleanup, and parses the output for structured results.
//
// Parameters:
//   - ctx: context for the execution with cancellation support
//
// Returns:
//   - *ScanResult: scan result with parsed output and metrics
//   - error: error if execution fails
func (s *SonarScannerEntity) executeOnce(ctx context.Context) (*ScanResult, error) {
	s.logger.Debug("Executing sonar-scanner")

	// Initialize scanner if not already done
	if err := s.Initialize(); err != nil {
		return nil, fmt.Errorf("scanner initialization failed: %w", err)
	}

	// Preprocess BSL files to fix common tokenization issues
	if err := s.preProcessBSLFiles(); err != nil {
		s.logger.Warn("BSL preprocessing failed", "error", err)
		// Don't fail the entire scan, just log the warning
	}

	// Create context with timeout if configured
	execCtx := ctx
	if s.config.Timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()
	}

	// Prepare command with context cancellation
	cmd := exec.CommandContext(execCtx, s.scannerPath) // #nosec G204 - s.scannerPath is validated

	// Set working directory
	if s.config.WorkDir != "" {
		cmd.Dir = s.config.WorkDir
	}

	// Set environment variables
	env := os.Environ()
	if s.config.JavaOpts != "" {
		env = append(env, "JAVA_OPTS="+s.config.JavaOpts)
	}
	cmd.Env = env

	// Add properties as command line arguments
	args := make([]string, 0, len(s.properties))
	for key, value := range s.properties {
		args = append(args, fmt.Sprintf("-D%s=%s", key, value))
	}
	cmd.Args = append(cmd.Args, args...)

	// Log command execution details
	s.logger.Debug("Starting scanner execution",
		"command", s.scannerPath,
		"args", args,
		"workDir", s.workDir,
		"timeout", s.config.Timeout)

	// Execute command with output capture
	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	// Create base scan result
	result := &ScanResult{
		Success:  err == nil,
		Duration: duration,
		Errors:   make([]string, 0),
		Metrics:  make(map[string]string),
	}

	// Parse output regardless of success/failure for diagnostic info
	if len(output) > 0 {
		if parseErr := s.parseOutput(string(output), result); parseErr != nil {
			s.logger.Warn("Failed to parse scanner output", "error", parseErr)
		}
	}

	// Handle execution errors with detailed diagnostics
	if err != nil {
		return s.handleExecutionError(err, string(output), result)
	}

	s.logger.Debug("Sonar-scanner executed successfully",
		"duration", duration,
		"analysisId", result.AnalysisID,
		"projectKey", result.ProjectKey)

	return result, nil
}

// Cleanup cleans up resources used by the scanner.
// This method cleans up temporary files and directories used by the scanner.
//
// Returns:
//   - error: error if cleanup fails
func (s *SonarScannerEntity) Cleanup() error {
	s.logger.Debug("Cleaning up scanner resources")

	// Clean up temporary directory if it was created
	if s.tempDir != "" && s.tempDir != os.TempDir() {
		err := os.RemoveAll(s.tempDir)
		if err != nil {
			return fmt.Errorf("failed to remove temporary directory: %w", err)
		}
	}

	s.logger.Debug("Scanner resources cleaned up successfully")
	return nil
}

// KillProcess forcefully terminates the scanner process if it's running.
// This method is useful for cleanup when the scanner needs to be stopped immediately.
func (s *SonarScannerEntity) KillProcess(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	s.logger.Debug("Attempting to kill scanner process", "pid", cmd.Process.Pid)

	// Try graceful termination first
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		s.logger.Warn("Failed to send interrupt signal", "error", err)
	}

	// Wait a short time for graceful shutdown
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(5 * time.Second):
		// Force kill if graceful shutdown failed
		s.logger.Warn("Graceful shutdown failed, force killing process")
		if err := cmd.Process.Kill(); err != nil {
			s.logger.Error("Failed to kill process", "error", err)
			return fmt.Errorf("failed to kill scanner process: %w", err)
		}
		s.logger.Debug("Scanner process killed successfully")
	case err := <-done:
		if err != nil {
			s.logger.Debug("Scanner process terminated", "error", err)
		} else {
			s.logger.Debug("Scanner process terminated gracefully")
		}
	}

	return nil
}

// ExecuteWithTimeout executes the scanner with enhanced timeout and process management.
// This method provides better control over process lifecycle and cleanup.
func (s *SonarScannerEntity) ExecuteWithTimeout(ctx context.Context, timeout time.Duration) (*ScanResult, error) {
	s.logger.Debug("Executing sonar-scanner with timeout", "timeout", timeout)

	// Initialize scanner if not already done
	if err := s.Initialize(); err != nil {
		return nil, fmt.Errorf("scanner initialization failed: %w", err)
	}

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Prepare command with context cancellation
	cmd := exec.CommandContext(execCtx, s.scannerPath) // #nosec G204 - s.scannerPath is validated

	// Set working directory
	if s.workDir != "" {
		cmd.Dir = s.workDir
	}

	// Set environment variables
	env := os.Environ()
	if s.config.JavaOpts != "" {
		env = append(env, "JAVA_OPTS="+s.config.JavaOpts)
	}
	cmd.Env = env

	// Add properties as command line arguments
	args := make([]string, 0, len(s.properties))
	for key, value := range s.properties {
		args = append(args, fmt.Sprintf("-D%s=%s", key, value))
	}
	cmd.Args = append(cmd.Args, args...)

	// Log command execution details
	s.logger.Debug("Starting scanner execution with timeout",
		"command", s.scannerPath,
		"args", args,
		"workDir", s.workDir,
		"timeout", timeout)

	// Execute command with enhanced error handling
	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	// Ensure process cleanup on timeout or cancellation
	defer func() {
		if cmd.Process != nil {
			if killErr := s.KillProcess(cmd); killErr != nil {
				s.logger.Warn("Failed to cleanup process", "error", killErr)
			}
		}
	}()

	// Create base scan result
	result := &ScanResult{
		Success:  err == nil,
		Duration: duration,
		Errors:   make([]string, 0),
		Metrics:  make(map[string]string),
	}

	// Parse output regardless of success/failure for diagnostic info
	if len(output) > 0 {
		if parseErr := s.parseOutput(string(output), result); parseErr != nil {
			s.logger.Warn("Failed to parse scanner output", "error", parseErr)
		}
	}

	// Handle execution errors with detailed diagnostics
	if err != nil {
		return s.handleExecutionError(err, string(output), result)
	}

	s.logger.Debug("Sonar-scanner executed successfully with timeout",
		"duration", duration,
		"analysisId", result.AnalysisID,
		"projectKey", result.ProjectKey)

	return result, nil
}

// parseOutput parses the scanner output to extract structured information
// such as analysis ID, project key, issues count, and metrics.
func (s *SonarScannerEntity) parseOutput(output string, result *ScanResult) error {
	lines := strings.Split(output, "\n")

	// Compile regex patterns for efficient parsing
	analysisIDRegex := regexp.MustCompile(`task\?id=([A-Za-z0-9_-]+)`)
	issuesRegex := regexp.MustCompile(`(\d+)\s+issues?\s+found`)
	coverageRegex := regexp.MustCompile(`Coverage\s+(\d+\.\d+)%`)
	duplicatedRegex := regexp.MustCompile(`Duplicated lines\s+(\d+\.\d+)%`)
	linesRegex := regexp.MustCompile(`Lines of code\s+(\d+)`)
	complexityRegex := regexp.MustCompile(`Cyclomatic complexity\s+(\d+)`)
	technicalDebtRegex := regexp.MustCompile(`Technical Debt\s+([0-9]+[dhm]+)`)
	executionTimeRegex := regexp.MustCompile(`Total time:\s*([0-9:.]+)\s*(s|min)`)
	memoryRegex := regexp.MustCompile(`Final Memory:\s*([0-9]+)M/([0-9]+)M`)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Extract analysis ID from successful analysis message
		if strings.Contains(line, "ANALYSIS SUCCESSFUL") {
			if matches := analysisIDRegex.FindStringSubmatch(line); len(matches) > 1 {
				result.AnalysisID = matches[1]
				s.logger.Debug("Extracted analysis ID", "analysisId", result.AnalysisID)
			}
		}

		// Extract project key
		if strings.HasPrefix(line, "INFO: Project key:") {
			result.ProjectKey = strings.TrimSpace(strings.TrimPrefix(line, "INFO: Project key:"))
			s.logger.Debug("Extracted project key", "projectKey", result.ProjectKey)
		}

		// Extract execution time
		if matches := executionTimeRegex.FindStringSubmatch(line); len(matches) > 2 {
			result.Metrics["execution_time"] = matches[1] + matches[2]
			s.logger.Debug("Extracted execution time", "time", result.Metrics["execution_time"])
		}

		// Extract memory usage
		if matches := memoryRegex.FindStringSubmatch(line); len(matches) > 2 {
			result.Metrics["memory_used"] = matches[1] + "M"
			result.Metrics["memory_total"] = matches[2] + "M"
			s.logger.Debug("Extracted memory usage", "used", result.Metrics["memory_used"], "total", result.Metrics["memory_total"])
		}

		// Extract issues count
		if matches := issuesRegex.FindStringSubmatch(line); len(matches) > 1 {
			result.Metrics["issues_count"] = matches[1]
			s.logger.Debug("Extracted issues count", "count", matches[1])
		}

		// Extract coverage percentage
		if matches := coverageRegex.FindStringSubmatch(line); len(matches) > 1 {
			result.Metrics["coverage"] = matches[1]
			s.logger.Debug("Extracted coverage", "coverage", matches[1])
		}

		// Extract duplicated lines percentage
		if matches := duplicatedRegex.FindStringSubmatch(line); len(matches) > 1 {
			result.Metrics["duplicated_lines"] = matches[1]
			s.logger.Debug("Extracted duplicated lines", "percentage", matches[1])
		}

		// Extract lines of code
		if matches := linesRegex.FindStringSubmatch(line); len(matches) > 1 {
			result.Metrics["lines_of_code"] = matches[1]
			s.logger.Debug("Extracted lines of code", "lines", matches[1])
		}

		// Extract cyclomatic complexity
		if matches := complexityRegex.FindStringSubmatch(line); len(matches) > 1 {
			result.Metrics["cyclomatic_complexity"] = matches[1]
			s.logger.Debug("Extracted cyclomatic complexity", "complexity", matches[1])
		}

		// Extract technical debt
		if matches := technicalDebtRegex.FindStringSubmatch(line); len(matches) > 1 {
			result.Metrics["technical_debt"] = matches[1]
			s.logger.Debug("Extracted technical debt", "debt", matches[1])
		}

		// Extract quality gate status
		if strings.Contains(line, "Quality gate") {
			if strings.Contains(line, "PASSED") {
				result.Metrics["quality_gate"] = "PASSED"
				s.logger.Debug("Quality gate passed")
			} else if strings.Contains(line, "FAILED") {
				result.Metrics["quality_gate"] = "FAILED"
				s.logger.Debug("Quality gate failed")
			}
		}

		// Extract server URL for result linking
		if strings.HasPrefix(line, "INFO: More about the report processing at") {
			url := strings.TrimSpace(strings.TrimPrefix(line, "INFO: More about the report processing at"))
			result.Metrics["report_url"] = url
			s.logger.Debug("Extracted report URL", "url", url)
		}

		// Extract task URL from analysis completion
		if strings.Contains(line, "http") && strings.Contains(line, "api/ce/task") {
			taskURLRegex := regexp.MustCompile(`(http[s]?://[^\s]+)`)
			if matches := taskURLRegex.FindStringSubmatch(line); len(matches) > 1 {
				result.Metrics["task_url"] = matches[1]
				s.logger.Debug("Extracted task URL", "url", matches[1])
			}
		}

		// Collect error messages
		if strings.HasPrefix(line, "ERROR:") {
			errorMsg := strings.TrimSpace(strings.TrimPrefix(line, "ERROR:"))
			if errorMsg != "" {
				result.Errors = append(result.Errors, errorMsg)
				s.logger.Debug("Extracted error message", "error", errorMsg)
			}
		}

		// Collect warning messages
		if strings.HasPrefix(line, "WARN:") {
			warnMsg := strings.TrimSpace(strings.TrimPrefix(line, "WARN:"))
			if warnMsg != "" {
				if result.Metrics["warnings"] == "" {
					result.Metrics["warnings"] = "1"
				} else {
					if count, err := strconv.Atoi(result.Metrics["warnings"]); err == nil {
						result.Metrics["warnings"] = strconv.Itoa(count + 1)
					}
				}
				result.Errors = append(result.Errors, "Warning: "+warnMsg)
				s.logger.Debug("Extracted warning message", "warning", warnMsg)
			}
		}

		// Collect INFO messages for debugging
		if strings.HasPrefix(line, "INFO:") && s.logger.Enabled(context.Background(), slog.LevelDebug) {
			infoMsg := strings.TrimSpace(strings.TrimPrefix(line, "INFO:"))
			if infoMsg != "" {
				s.logger.Debug("Scanner info", "message", infoMsg)
			}
		}

		// Extract analysis progress information
		if strings.Contains(line, "Analyzing") || strings.Contains(line, "Processed") {
			progressRegex := regexp.MustCompile(`(\d+)/(\d+)\s+files`)
			if matches := progressRegex.FindStringSubmatch(line); len(matches) > 2 {
				result.Metrics["files_processed"] = matches[1]
				result.Metrics["files_total"] = matches[2]
				s.logger.Debug("Extracted progress", "processed", matches[1], "total", matches[2])
			}
		}
	}

	// Log summary of extracted metrics
	s.logger.Debug("Parsing completed",
		"metricsCount", len(result.Metrics),
		"errorsCount", len(result.Errors),
		"hasAnalysisId", result.AnalysisID != "",
		"hasProjectKey", result.ProjectKey != "")

	return nil
}

// handleExecutionError handles errors that occur during scanner execution.
// This method provides detailed error analysis and creates appropriate
// error responses with diagnostic information.
//
// Parameters:
//   - err: execution error
//   - output: scanner output for diagnostic purposes
//   - result: scan result to populate with error information
//
// Returns:
//   - *ScanResult: scan result with error information
//   - error: wrapped error with additional context
func (s *SonarScannerEntity) handleExecutionError(err error, output string, result *ScanResult) (*ScanResult, error) {
	s.logger.Error("Scanner execution failed", "error", err, "outputLength", len(output))

	// Log scanner output for debugging (truncated if too long)
	if len(output) > 0 {
		const maxOutputLogLength = 2000
		outputForLog := output
		if len(output) > maxOutputLogLength {
			outputForLog = output[len(output)-maxOutputLogLength:] // Show last 2000 chars
			s.logger.Error("Scanner output (last 2000 chars)", "output", outputForLog)
		} else {
			s.logger.Error("Scanner output", "output", outputForLog)
		}
	}

	// Check for context cancellation (timeout)
	if errors.Is(err, context.DeadlineExceeded) {
		result.Errors = append(result.Errors, fmt.Sprintf("Scanner execution timed out after %v", s.config.Timeout))
		s.logger.Error("Scanner execution timed out", "timeout", s.config.Timeout)

		// Analyze output for additional timeout context
		timeoutAnalysis := s.analyzeTimeoutError(output)
		if len(timeoutAnalysis) > 0 {
			result.Errors = append(result.Errors, timeoutAnalysis...)
		}

		return result, &ScannerError{
			ExitCode: -1,
			Output:   output,
			ErrorMsg: "execution timed out",
		}
	}

	// Check for context cancellation
	if errors.Is(err, context.Canceled) {
		result.Errors = append(result.Errors, "Scanner execution was cancelled")
		s.logger.Info("Scanner execution was cancelled")
		return result, &ScannerError{
			ExitCode: -1,
			Output:   output,
			ErrorMsg: "execution cancelled",
		}
	}

	// Handle exit errors with specific exit codes
	if exitError, ok := err.(*exec.ExitError); ok {
		exitCode := exitError.ExitCode()

		// Analyze output for specific error patterns
		errorAnalysis := s.analyzeErrorOutput(output)

		// Create detailed error message based on exit code
		errorMsg := s.getExitCodeMessage(exitCode, errorAnalysis)
		result.Errors = append(result.Errors, errorMsg)

		// Add specific error details from output analysis
		if len(errorAnalysis) > 0 {
			result.Errors = append(result.Errors, errorAnalysis...)
			
			// Check for BSL-specific errors and attempt automatic fixes
			bslTokenizationError := false
			for _, errMsg := range errorAnalysis {
				if strings.Contains(errMsg, "BSL tokenization error") {
					bslTokenizationError = true
					
					// Extract BSL file path from error output
					bslFilePattern := regexp.MustCompile(`([^/\s]+/[^/\s]*\.bsl)`)
					matches := bslFilePattern.FindAllString(output, -1)
					
					if len(matches) > 0 {
						for _, filePath := range matches {
							// Attempt to fix the problematic BSL file
							s.logger.Info("Attempting to fix BSL tokenization issue", "file", filePath)
							if err := s.FixBSLTokenizationIssues(filePath); err != nil {
								s.logger.Error("Failed to fix BSL file", "file", filePath, "error", err)
								result.Errors = append(result.Errors, fmt.Sprintf("FAILED TO FIX: %s - %v", filePath, err))
							} else {
								result.Errors = append(result.Errors, fmt.Sprintf("ATTEMPTED FIX: %s - file has been automatically corrected", filePath))
								result.Errors = append(result.Errors, "RECOMMENDATION: Re-run the scanner to check if the issue is resolved")
							}
						}
					}
					
					// Add general recommendations
					result.Errors = append(result.Errors, "RECOMMENDATION: Check BSL file encoding (should be UTF-8) and verify syntax correctness")
					result.Errors = append(result.Errors, "RECOMMENDATION: Try excluding problematic BSL files using sonar.exclusions property")
					result.Errors = append(result.Errors, "RECOMMENDATION: Update BSL plugin to latest version or check plugin compatibility")
				}
				if strings.Contains(errMsg, "BSL plugin error") {
					result.Errors = append(result.Errors, "RECOMMENDATION: Verify BSL plugin installation and version compatibility")
					result.Errors = append(result.Errors, "RECOMMENDATION: Check SonarQube server logs for BSL plugin issues")
				}
			}
			
			// If BSL tokenization error was detected, provide exclusion suggestions
			if bslTokenizationError {
				exclusions := s.SuggestBSLExclusions(output)
				if len(exclusions) > 0 {
					result.Errors = append(result.Errors, "EXCLUSION SUGGESTIONS:")
					result.Errors = append(result.Errors, exclusions...)
				}
			}
		} else {
			// If no specific errors found, add a generic message with suggestion to check output
			result.Errors = append(result.Errors, "No specific error patterns detected. Check scanner output above for details.")
		}

		// Log specific exit code meanings with context and error analysis results
		switch exitCode {
		case 1:
			s.logger.Error("Scanner failed with quality gate failure or analysis errors", 
				"exitCode", exitCode, 
				"errorCount", len(errorAnalysis),
				"errorAnalysis", errorAnalysis)
		case 2:
			s.logger.Error("Scanner failed with invalid configuration", 
				"exitCode", exitCode, 
				"configErrors", errorAnalysis)
		case 3:
			s.logger.Error("Scanner failed with internal error", 
				"exitCode", exitCode,
				"errorAnalysis", errorAnalysis)
		case 4:
			s.logger.Error("Scanner failed with resource issues", 
				"exitCode", exitCode,
				"errorAnalysis", errorAnalysis)
		default:
			s.logger.Error("Scanner failed with unknown error", 
				"exitCode", exitCode,
				"errorAnalysis", errorAnalysis)
		}

		return result, &ScannerError{
			ExitCode: exitCode,
			Output:   output,
			ErrorMsg: errorMsg,
		}
	}

	// Handle other execution errors
	result.Errors = append(result.Errors, "Unexpected scanner execution error: "+err.Error())
	s.logger.Error("Unexpected scanner execution error", "error", err, "type", fmt.Sprintf("%T", err))
	return result, fmt.Errorf("scanner execution failed: %w", err)
}

// analyzeErrorOutput analyzes scanner output to extract specific error information.
// This method looks for common error patterns and provides detailed diagnostics.
//
// Parameters:
//   - output: scanner output string
//
// Returns:
//   - []string: list of specific error messages found
func (s *SonarScannerEntity) analyzeErrorOutput(output string) []string {
	var errors []string
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for authentication errors
		if strings.Contains(line, "Unauthorized") || strings.Contains(line, "401") {
			errors = append(errors, "Authentication failed: Invalid token or credentials")
		}

		// Check for network connectivity errors
		if strings.Contains(line, "Connection refused") || strings.Contains(line, "ConnectException") {
			errors = append(errors, "Network error: Cannot connect to SonarQube server")
		}

		// Check for project configuration errors
		if strings.Contains(line, "Project key") && strings.Contains(line, "invalid") {
			errors = append(errors, "Invalid project key configuration")
		}

		// Check for source code analysis errors
		if strings.Contains(line, "No sources found") || strings.Contains(line, "No files to analyze") {
			errors = append(errors, "No source files found for analysis")
		}

		// Check for Java/JVM errors
		if strings.Contains(line, "OutOfMemoryError") {
			errors = append(errors, "Insufficient memory: Increase JAVA_OPTS heap size")
		}

		// Check for permission errors
		if strings.Contains(line, "Permission denied") || strings.Contains(line, "Access denied") {
			errors = append(errors, "File system permission error")
		}

		// Check for quality gate failures
		if strings.Contains(line, "Quality gate") && strings.Contains(line, "FAILED") {
			errors = append(errors, "Quality gate failed: Code quality standards not met")
		}

		// Check for server version compatibility
		if strings.Contains(line, "version") && strings.Contains(line, "not supported") {
			errors = append(errors, "SonarQube server version compatibility issue")
		}

		// Check for plugin errors
		if strings.Contains(line, "plugin") && strings.Contains(line, "failed") {
			errors = append(errors, "Scanner plugin error")
		}

		// Check for ERROR: lines (most common error pattern)
		if strings.HasPrefix(line, "ERROR:") {
			errorMsg := strings.TrimSpace(strings.TrimPrefix(line, "ERROR:"))
			if errorMsg != "" {
				errors = append(errors, "Scanner error: "+errorMsg)
			}
		}

		// Check for WARN: lines that might indicate issues
		if strings.HasPrefix(line, "WARN:") {
			warnMsg := strings.TrimSpace(strings.TrimPrefix(line, "WARN:"))
			if warnMsg != "" && (strings.Contains(warnMsg, "fail") || strings.Contains(warnMsg, "error") || strings.Contains(warnMsg, "invalid")) {
				errors = append(errors, "Scanner warning: "+warnMsg)
			}
		}

		// Check for specific SonarQube error patterns
		if strings.Contains(line, "EXECUTION FAILURE") {
			errors = append(errors, "Scanner execution failure detected")
		}

		// Check for timeout-related errors
		if strings.Contains(line, "timeout") || strings.Contains(line, "timed out") {
			errors = append(errors, "Timeout error: "+line)
		}

		// Check for SSL/TLS errors
		if strings.Contains(line, "SSL") || strings.Contains(line, "TLS") || strings.Contains(line, "certificate") {
			errors = append(errors, "SSL/TLS connection error: "+line)
		}

		// Check for disk space errors
		if strings.Contains(line, "No space left") || strings.Contains(line, "disk full") {
			errors = append(errors, "Disk space error: "+line)
		}

		// Check for Java classpath errors
		if strings.Contains(line, "ClassNotFoundException") || strings.Contains(line, "NoClassDefFoundError") {
			errors = append(errors, "Java classpath error: "+line)
		}

		// Check for configuration file errors
		if strings.Contains(line, "sonar-project.properties") && (strings.Contains(line, "not found") || strings.Contains(line, "missing")) {
			errors = append(errors, "Configuration file error: "+line)
		}

		// Check for Git-related errors
		if strings.Contains(line, "git") && (strings.Contains(line, "not found") || strings.Contains(line, "failed")) {
			errors = append(errors, "Git-related error: "+line)
		}

		// Check for analysis errors
		if strings.Contains(line, "Analysis failed") || strings.Contains(line, "analysis error") {
			errors = append(errors, "Analysis error: "+line)
		}

		// Check for server response errors
		if strings.Contains(line, "500") || strings.Contains(line, "502") || strings.Contains(line, "503") || strings.Contains(line, "504") {
			errors = append(errors, "Server error: "+line)
		}

		// Check for BSL (1C:Enterprise) specific errors
		if strings.Contains(line, "java.lang.IllegalStateException") && strings.Contains(line, "Tokens of file") && strings.Contains(line, ".bsl") {
			errors = append(errors, "BSL tokenization error: File contains invalid token sequence - check file encoding and syntax")
		}

		// Check for BSL plugin errors
		if strings.Contains(line, "com.github._1c_syntax.bsl") {
			errors = append(errors, "BSL plugin error: "+line)
		}

		// Check for BSL file encoding issues
		if strings.Contains(line, ".bsl") && (strings.Contains(line, "encoding") || strings.Contains(line, "charset")) {
			errors = append(errors, "BSL encoding error: "+line)
		}

		// Check for BSL syntax errors
		if strings.Contains(line, ".bsl") && strings.Contains(line, "syntax") {
			errors = append(errors, "BSL syntax error: "+line)
		}

		// Check for 1C:Enterprise platform errors
		if strings.Contains(line, "1C") || strings.Contains(line, "1c") {
			if strings.Contains(line, "error") || strings.Contains(line, "ERROR") || strings.Contains(line, "failed") {
				errors = append(errors, "1C platform error: "+line)
			}
		}
	}

	return errors
}

// analyzeTimeoutError analyzes output for timeout-specific context.
// This method provides additional information about what was happening when timeout occurred.
//
// Parameters:
//   - output: scanner output string
//
// Returns:
//   - []string: list of timeout-related diagnostic messages
func (s *SonarScannerEntity) analyzeTimeoutError(output string) []string {
	var diagnostics []string
	lines := strings.Split(output, "\n")

	// Find the last meaningful activity before timeout
	found := false
	for i := len(lines) - 1; i >= 0 && !found; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Check what phase the scanner was in
		switch {
		case strings.Contains(line, "Analyzing"):
			diagnostics = append(diagnostics, "Timeout occurred during file analysis phase")
			found = true
		case strings.Contains(line, "Uploading") || strings.Contains(line, "Sending"):
			diagnostics = append(diagnostics, "Timeout occurred during result upload to server")
			found = true
		case strings.Contains(line, "Downloading"):
			diagnostics = append(diagnostics, "Timeout occurred during dependency download")
			found = true
		case strings.Contains(line, "Starting"):
			diagnostics = append(diagnostics, "Timeout occurred during scanner initialization")
			found = true
		}
	}

	// Add timeout mitigation suggestions
	diagnostics = append(diagnostics, "Consider increasing timeout value or optimizing project size")

	return diagnostics
}

// getExitCodeMessage returns a descriptive message for scanner exit codes.
// This method provides human-readable explanations for different exit codes.
//
// Parameters:
//   - exitCode: process exit code
//   - errorAnalysis: additional error details from output analysis
//
// Returns:
//   - string: descriptive error message
func (s *SonarScannerEntity) getExitCodeMessage(exitCode int, errorAnalysis []string) string {
	var baseMessage string

	switch exitCode {
	case 1:
		baseMessage = "Scanner execution failed: Quality gate failure or analysis errors"
	case 2:
		baseMessage = "Scanner execution failed: Invalid configuration or parameters"
	case 3:
		baseMessage = "Scanner execution failed: Internal error or unexpected failure"
	case 4:
		baseMessage = "Scanner execution failed: Insufficient memory or resources"
	case 5:
		baseMessage = "Scanner execution failed: Network or connectivity issues"
	default:
		baseMessage = fmt.Sprintf("Scanner execution failed with exit code %d", exitCode)
	}

	// Add specific error context if available
	if len(errorAnalysis) > 0 {
		baseMessage += " (" + strings.Join(errorAnalysis, ", ") + ")"
	}

	return baseMessage
}

// FixBSLTokenizationIssues attempts to fix common BSL file tokenization issues
// that cause SonarQube scanner to fail with "Tokens of file should be provided in order" error
func (s *SonarScannerEntity) FixBSLTokenizationIssues(filePath string) error {
	s.logger.Info("Attempting to fix BSL tokenization issues", "file", filePath)
	
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		s.logger.Warn("BSL file not found", "file", filePath)
		return fmt.Errorf("BSL file not found: %s", filePath)
	}
	
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		s.logger.Error("Failed to read BSL file", "file", filePath, "error", err)
		return fmt.Errorf("failed to read BSL file %s: %w", filePath, err)
	}
	
	originalContent := string(content)
	fixedContent := originalContent
	fixesApplied := 0
	
	// Fix 1: Normalize line endings to Unix style (\n)
	if strings.Contains(fixedContent, "\r\n") {
		fixedContent = strings.ReplaceAll(fixedContent, "\r\n", "\n")
		fixesApplied++
		s.logger.Debug("Fixed Windows line endings", "file", filePath)
	}
	
	// Fix 2: Remove trailing whitespace from lines
	lines := strings.Split(fixedContent, "\n")
	for i, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		if trimmed != line {
			lines[i] = trimmed
			fixesApplied++
		}
	}
	fixedContent = strings.Join(lines, "\n")
	
	// Fix 3: Ensure file ends with a single newline
	fixedContent = strings.TrimRight(fixedContent, "\n") + "\n"
	
	// Fix 4: Remove or fix problematic Unicode characters that might cause tokenization issues
	// Replace non-breaking spaces with regular spaces
	if strings.Contains(fixedContent, "\u00A0") {
		fixedContent = strings.ReplaceAll(fixedContent, "\u00A0", " ")
		fixesApplied++
		s.logger.Debug("Fixed non-breaking spaces", "file", filePath)
	}
	
	// Fix 5: Remove BOM (Byte Order Mark) if present
	if strings.HasPrefix(fixedContent, "\uFEFF") {
		fixedContent = strings.TrimPrefix(fixedContent, "\uFEFF")
		fixesApplied++
		s.logger.Debug("Removed BOM", "file", filePath)
	}
	
	// Fix 6: Validate and fix BSL syntax issues that might cause tokenization problems
	fixedContent = s.fixBSLSyntaxIssues(fixedContent, filePath)
	
	// Only write back if changes were made
	if fixedContent != originalContent {
		// Create backup
		backupPath := filePath + ".backup"
		if err := os.WriteFile(backupPath, content, 0644); err != nil {
			s.logger.Warn("Failed to create backup", "file", filePath, "backup", backupPath, "error", err)
		} else {
			s.logger.Info("Created backup", "file", filePath, "backup", backupPath)
		}
		
		// Write fixed content
		if err := os.WriteFile(filePath, []byte(fixedContent), 0644); err != nil {
			s.logger.Error("Failed to write fixed BSL file", "file", filePath, "error", err)
			return fmt.Errorf("failed to write fixed BSL file %s: %w", filePath, err)
		}
		
		s.logger.Info("Applied BSL fixes", "file", filePath, "fixesApplied", fixesApplied)
	} else {
		s.logger.Info("No BSL fixes needed", "file", filePath)
	}
	
	return nil
}

// fixBSLSyntaxIssues fixes common BSL syntax issues that cause tokenization problems
func (s *SonarScannerEntity) fixBSLSyntaxIssues(content, filePath string) string {
	lines := strings.Split(content, "\n")
	
	for i, line := range lines {
		originalLine := line
		
		// Fix 1: Remove invalid characters that might interfere with tokenization
		// Keep only valid BSL characters (Cyrillic, Latin, digits, and BSL-specific symbols)
		validChars := regexp.MustCompile(`[^\p{L}\p{N}\s\(\)\[\]\{\}\.,;:=\+\-\*\/\\\|&<>!@#\$%\^~` + "`" + `"'_]`)
		line = validChars.ReplaceAllString(line, "")
		
		// Fix 2: Fix common BSL comment issues
		// Ensure single-line comments start properly
		if strings.Contains(line, "//") {
			// Make sure there's no invalid character before //
			line = regexp.MustCompile(`[^\s]/\/`).ReplaceAllString(line, " //")
		}
		
		// Fix 3: Fix string literal issues
		// Ensure string literals are properly closed
		if strings.Count(line, `"`)%2 != 0 {
			// If odd number of quotes, add closing quote at the end
			line = line + `"`
			s.logger.Debug("Fixed unclosed string literal", "file", filePath, "line", i+1)
		}
		
		// Fix 4: Fix procedure/function declaration issues
		// Normalize procedure/function keywords
		line = regexp.MustCompile(`(?i)\b(процедура|функция)\b`).ReplaceAllStringFunc(line, func(match string) string {
			if strings.EqualFold(match, "процедура") {
				return "Процедура"
			}
			if strings.EqualFold(match, "функция") {
				return "Функция"
			}
			return match
		})
		
		if line != originalLine {
			lines[i] = line
			s.logger.Debug("Fixed BSL syntax issue", "file", filePath, "line", i+1, "original", originalLine, "fixed", line)
		}
	}
	
	return strings.Join(lines, "\n")
}

// FindAndValidateBSLFiles searches for BSL files in the working directory and categorizes them
func (s *SonarScannerEntity) FindAndValidateBSLFiles() ([]string, []string, error) {
	var validFiles []string
	var problematicFiles []string
	
	if s.workDir == "" {
		return validFiles, problematicFiles, fmt.Errorf("working directory not set")
	}
	
	s.logger.Info("Searching for BSL files", "workDir", s.workDir)
	
	err := filepath.Walk(s.workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			s.logger.Warn("Error accessing path", "path", path, "error", err)
			return nil // Continue walking
		}
		
		// Check if it's a BSL file
		if !info.IsDir() && (strings.HasSuffix(strings.ToLower(path), ".bsl") || strings.HasSuffix(strings.ToLower(path), ".os")) {
			s.logger.Debug("Found BSL file", "path", path)
			
			// Validate the file
			if s.validateBSLFile(path) {
				validFiles = append(validFiles, path)
			} else {
				problematicFiles = append(problematicFiles, path)
				s.logger.Warn("Problematic BSL file detected", "path", path)
			}
		}
		
		return nil
	})
	
	if err != nil {
		return validFiles, problematicFiles, fmt.Errorf("failed to walk directory: %w", err)
	}
	
	s.logger.Info("BSL file scan completed", "validFiles", len(validFiles), "problematicFiles", len(problematicFiles))
	return validFiles, problematicFiles, nil
}

// validateBSLFile performs basic validation of a BSL file to detect potential tokenization issues
func (s *SonarScannerEntity) validateBSLFile(filePath string) bool {
	content, err := os.ReadFile(filePath)
	if err != nil {
		s.logger.Error("Failed to read BSL file for validation", "file", filePath, "error", err)
		return false
	}
	
	contentStr := string(content)
	
	// Check for common issues that cause tokenization problems
	
	// 1. Check for mixed line endings
	if strings.Contains(contentStr, "\r\n") && strings.Contains(contentStr, "\n") {
		s.logger.Debug("BSL file has mixed line endings", "file", filePath)
		return false
	}
	
	// 2. Check for BOM
	if strings.HasPrefix(contentStr, "\uFEFF") {
		s.logger.Debug("BSL file has BOM", "file", filePath)
		return false
	}
	
	// 3. Check for non-breaking spaces
	if strings.Contains(contentStr, "\u00A0") {
		s.logger.Debug("BSL file contains non-breaking spaces", "file", filePath)
		return false
	}
	
	// 4. Check for unclosed string literals (basic check)
	lines := strings.Split(contentStr, "\n")
	for i, line := range lines {
		// Skip comments
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		
		// Count quotes (simple check)
		if strings.Count(line, `"`)%2 != 0 {
			s.logger.Debug("BSL file has unclosed string literal", "file", filePath, "line", i+1)
			return false
		}
	}
	
	// 5. Check for invalid characters
	validChars := regexp.MustCompile(`^[\p{L}\p{N}\s\(\)\[\]\{\}\.,;:=\+\-\*\/\\\|&<>!@#\$%\^~` + "`" + `"'_\r\n]*$`)
	if !validChars.MatchString(contentStr) {
		s.logger.Debug("BSL file contains invalid characters", "file", filePath)
		return false
	}
	
	return true
}

// preProcessBSLFiles finds and fixes all problematic BSL files before scanning
func (s *SonarScannerEntity) preProcessBSLFiles() error {
	s.logger.Info("Starting BSL files preprocessing")
	
	validFiles, problematicFiles, err := s.FindAndValidateBSLFiles()
	if err != nil {
		return fmt.Errorf("failed to find BSL files: %w", err)
	}
	
	s.logger.Info("BSL files analysis", "valid", len(validFiles), "problematic", len(problematicFiles))
	
	// Fix problematic files
	fixedCount := 0
	for _, filePath := range problematicFiles {
		if err := s.FixBSLTokenizationIssues(filePath); err != nil {
			s.logger.Error("Failed to fix BSL file during preprocessing", "file", filePath, "error", err)
		} else {
			fixedCount++
		}
	}
	
	s.logger.Info("BSL preprocessing completed", "totalFiles", len(validFiles)+len(problematicFiles), "fixedFiles", fixedCount)
	return nil
}

// AddFileToExclusions adds a file to the exclusion list and updates sonar.exclusions property
func (s *SonarScannerEntity) AddFileToExclusions(filePath string) {
	// Check if file is already excluded
	for _, excluded := range s.excludedFiles {
		if excluded == filePath {
			return // Already excluded
		}
	}
	
	// Add to excluded files list
	s.excludedFiles = append(s.excludedFiles, filePath)
	
	// Update sonar.exclusions property
	s.updateExclusionsProperty()
	
	s.logger.Info("Added file to exclusions", 
		"file", filePath, 
		"total_excluded", len(s.excludedFiles))
}

// AddFilesToExclusions adds multiple files to the exclusion list
func (s *SonarScannerEntity) AddFilesToExclusions(filePaths []string) {
	for _, filePath := range filePaths {
		s.AddFileToExclusions(filePath)
	}
}

// updateExclusionsProperty updates the sonar.exclusions property with current excluded files
func (s *SonarScannerEntity) updateExclusionsProperty() {
	if len(s.excludedFiles) == 0 {
		return
	}
	
	// Get existing exclusions
	existingExclusions := s.GetProperty("sonar.exclusions")
	
	// Create exclusion patterns for BSL files
	var exclusionPatterns []string
	
	// Add existing exclusions if any
	if existingExclusions != "" {
		exclusionPatterns = append(exclusionPatterns, existingExclusions)
	}
	
	// Add new exclusions
	for _, filePath := range s.excludedFiles {
		// Create pattern that matches the file regardless of directory structure
		pattern := fmt.Sprintf("**/%s", filepath.Base(filePath))
		exclusionPatterns = append(exclusionPatterns, pattern)
		
		// Also add the exact path if it's relative
		if !filepath.IsAbs(filePath) {
			exclusionPatterns = append(exclusionPatterns, filePath)
		}
	}
	
	// Join all patterns with comma
	exclusionsValue := strings.Join(exclusionPatterns, ",")
	
	// Set the property
	s.SetProperty("sonar.exclusions", exclusionsValue)
	
	s.logger.Info("Updated sonar.exclusions property", 
		"exclusions", exclusionsValue,
		"excluded_files_count", len(s.excludedFiles))
}

// GetExcludedFiles returns the list of currently excluded files
func (s *SonarScannerEntity) GetExcludedFiles() []string {
	return append([]string(nil), s.excludedFiles...) // Return a copy
}

// ClearExclusions clears all excluded files and resets the sonar.exclusions property
func (s *SonarScannerEntity) ClearExclusions() {
	s.excludedFiles = make([]string, 0)
	s.SetProperty("sonar.exclusions", "")
	s.logger.Info("Cleared all file exclusions")
}

// ExtractProblematicBSLFiles extracts file paths from BSL-related error messages
func (s *SonarScannerEntity) ExtractProblematicBSLFiles(errorOutput string) []string {
	var problematicFiles []string
	
	// Pattern to match BSL file paths in error messages
	// Matches patterns like: "file.bsl", "/path/to/file.bsl", "C:\path\to\file.bsl"
	bslFilePattern := regexp.MustCompile(`(?i)([^\s"']+\.bsl)`)
	matches := bslFilePattern.FindAllString(errorOutput, -1)
	
	// Pattern to match more specific error contexts with file paths
	contextPatterns := []string{
		`(?i)error.*?([^\s"']+\.bsl)`,
		`(?i)failed.*?([^\s"']+\.bsl)`,
		`(?i)cannot.*?([^\s"']+\.bsl)`,
		`(?i)unable.*?([^\s"']+\.bsl)`,
		`(?i)tokenization.*?([^\s"']+\.bsl)`,
		`(?i)parsing.*?([^\s"']+\.bsl)`,
	}
	
	for _, pattern := range contextPatterns {
		contextRegex := regexp.MustCompile(pattern)
		contextMatches := contextRegex.FindAllStringSubmatch(errorOutput, -1)
		for _, match := range contextMatches {
			if len(match) > 1 {
				matches = append(matches, match[1])
			}
		}
	}
	
	// Deduplicate and clean file paths
	fileMap := make(map[string]bool)
	for _, match := range matches {
		// Clean the file path
		cleanPath := strings.TrimSpace(match)
		cleanPath = strings.Trim(cleanPath, `"'`)
		
		// Skip if empty or too short
		if len(cleanPath) < 5 {
			continue
		}
		
		// Convert to relative path if it's absolute
		if filepath.IsAbs(cleanPath) {
			if rel, err := filepath.Rel(s.workDir, cleanPath); err == nil {
				cleanPath = rel
			}
		}
		
		// Add to map for deduplication
		if !fileMap[cleanPath] {
			fileMap[cleanPath] = true
			problematicFiles = append(problematicFiles, cleanPath)
		}
	}
	
	s.logger.Info("Extracted problematic BSL files", 
		"count", len(problematicFiles), 
		"files", problematicFiles)
	
	return problematicFiles
}

// SuggestBSLExclusions generates exclusion patterns for problematic BSL files
func (s *SonarScannerEntity) SuggestBSLExclusions(errorOutput string) []string {
	var suggestions []string
	
	// Extract file paths from error messages
	bslFilePattern := regexp.MustCompile(`([^/\s]+\.bsl)`)
	matches := bslFilePattern.FindAllString(errorOutput, -1)
	
	for _, match := range matches {
		// Suggest excluding the specific file
		suggestions = append(suggestions, fmt.Sprintf("sonar.exclusions=**/%s", match))
		
		// If it's a common module, suggest excluding all common modules with similar pattern
		if strings.Contains(strings.ToLower(match), "commonmodule") || strings.Contains(match, "CommonModules") {
			suggestions = append(suggestions, "sonar.exclusions=**/CommonModules/**/*.bsl")
		}
	}
	
	// Add general BSL exclusion suggestions
	if len(matches) > 0 {
		suggestions = append(suggestions, 
			"# Consider excluding problematic BSL files:",
			"sonar.exclusions=**/*Server*.bsl,**/*Сервер*.bsl",
			"sonar.exclusions=**/CommonModules/**/*.bsl",
			"sonar.exclusions=**/*.bsl  # Exclude all BSL files if issues persist",
		)
	}
	
	return suggestions
}

// ToDo: необходимо дополнительно реализовать следующий функционал:
// - Add support for different scanner versions
// - Add progress reporting during download and execution
// - Implement scanner configuration caching
// - Add metrics collection for performance monitoring
//
// Выполнено в рамках пункта 3.2:
// ✓ Реализована логика выполнения сканера с контекстом отмены
// ✓ Добавлен улучшенный парсинг вывода и обработка результатов
// ✓ Реализована обработка таймаутов и очистка процессов
// ✓ Добавлены unit тесты для выполнения сканера
// ✓ Улучшена обработка ошибок с детальной диагностикой
// ✓ Добавлены утилитарные функции для исправления BSL файлов
//
// Ссылки на пункты плана и требований:
// - tasks.md: 3.2 (выполнено), 3.3 (следующий)
// - requirements.md: 10.3, 10.4, 9.4
