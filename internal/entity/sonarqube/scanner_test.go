// Package sonarqube provides tests for SonarScanner entity implementation.
package sonarqube

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/stretchr/testify/assert"
)

// mockExitError implements exec.ExitError interface for testing
type mockExitError struct {
	exitCode int
}

func (e *mockExitError) Error() string {
	return "exit status " + string(rune(e.exitCode + '0'))
}

func (e *mockExitError) ExitCode() int {
	return e.exitCode
}

// TestNewSonarScannerEntity tests the creation of a new SonarScannerEntity.
func TestNewSonarScannerEntity(t *testing.T) {
	cfg := &config.ScannerConfig{
		ScannerURL:     "http://localhost:9000",
		ScannerVersion: "4.6.2",
		JavaOpts:       "-Xmx512m",
		Properties:     map[string]string{"sonar.host.url": "http://localhost:9000"},
		Timeout:        30 * time.Second,
		WorkDir:        "/tmp",
		TempDir:        "/tmp",
	}
	
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	entity := NewSonarScannerEntity(cfg, logger)
	
	assert.NotNil(t, entity)
	assert.Equal(t, cfg, entity.config)
	assert.Equal(t, logger, entity.logger)
	assert.Empty(t, entity.scannerPath)
	assert.Empty(t, entity.properties)
	assert.Equal(t, cfg.WorkDir, entity.workDir)
	assert.Equal(t, cfg.TempDir, entity.tempDir)
}

// TestSonarScannerEntity_SetProperty tests the SetProperty method.
func TestSonarScannerEntity_SetProperty(t *testing.T) {
	cfg := &config.ScannerConfig{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewSonarScannerEntity(cfg, logger)
	
	entity.SetProperty("key1", "value1")
	entity.SetProperty("key2", "value2")
	
	assert.Equal(t, "value1", entity.GetProperty("key1"))
	assert.Equal(t, "value2", entity.GetProperty("key2"))
}

// TestSonarScannerEntity_GetProperty tests the GetProperty method.
func TestSonarScannerEntity_GetProperty(t *testing.T) {
	cfg := &config.ScannerConfig{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewSonarScannerEntity(cfg, logger)
	
	// Test getting a property that doesn't exist
	assert.Empty(t, entity.GetProperty("nonexistent"))
	
	// Test getting a property that exists
	entity.SetProperty("key", "value")
	assert.Equal(t, "value", entity.GetProperty("key"))
}

// TestSonarScannerEntity_Configure tests the Configure method.
func TestSonarScannerEntity_Configure(t *testing.T) {
	cfg := &config.ScannerConfig{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewSonarScannerEntity(cfg, logger)
	
	scannerConfig := &ScannerConfig{
		ScannerURL:     "http://localhost:9000",
		ScannerVersion: "4.6.2",
		JavaOpts:       "-Xmx512m",
		Properties:     map[string]string{"sonar.host.url": "http://localhost:9000", "sonar.projectKey": "test"},
		Timeout:        30 * time.Second,
		WorkDir:        "/tmp",
		TempDir:        "/tmp",
	}
	
	err := entity.Configure(scannerConfig)
	assert.NoError(t, err)
	
	// Check that properties were set
	assert.Equal(t, "http://localhost:9000", entity.GetProperty("sonar.host.url"))
	assert.Equal(t, "test", entity.GetProperty("sonar.projectKey"))
	
	// Check that workDir and tempDir were set
	assert.Equal(t, "/tmp", entity.workDir)
	assert.Equal(t, "/tmp", entity.tempDir)
}

// TestSonarScannerEntity_ValidateConfig tests the ValidateConfig method.
func TestSonarScannerEntity_ValidateConfig(t *testing.T) {
	cfg := &config.ScannerConfig{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewSonarScannerEntity(cfg, logger)
	
	// Test with empty scanner path
	err := entity.ValidateConfig()
	assert.Error(t, err)
	assert.IsType(t, &ValidationError{}, err)
	
	// Set a scanner path that doesn't exist
	entity.scannerPath = "/nonexistent/scanner"
	err = entity.ValidateConfig()
	assert.Error(t, err)
	assert.IsType(t, &ValidationError{}, err)
	
	// Create a temporary file to act as the scanner executable
	tmpFile, err := os.CreateTemp("", "scanner")
	assert.NoError(t, err)
	defer func() {
		if removeErr := os.Remove(tmpFile.Name()); removeErr != nil {
			t.Logf("Failed to remove temporary file: %v", removeErr)
		}
	}()
	
	// Set the scanner path to the temporary file
	entity.scannerPath = tmpFile.Name()
	
	// Test with a workDir that doesn't exist
	entity.workDir = "/nonexistent/workdir"
	err = entity.ValidateConfig()
	assert.Error(t, err)
	assert.IsType(t, &ValidationError{}, err)
	
	// Set workDir to an existing directory
	entity.workDir = os.TempDir()
	err = entity.ValidateConfig()
	assert.NoError(t, err)
}

// TestSonarScannerEntity_Initialize tests the Initialize method.
func TestSonarScannerEntity_Initialize(t *testing.T) {
	cfg := &config.ScannerConfig{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewSonarScannerEntity(cfg, logger)
	
	// Test with invalid configuration
	err := entity.Initialize()
	assert.Error(t, err)
	
	// Create a temporary file to act as the scanner executable
	tmpFile, err := os.CreateTemp("", "scanner")
	assert.NoError(t, err)
	defer func() {
		if removeErr := os.Remove(tmpFile.Name()); removeErr != nil {
			t.Logf("Failed to remove temporary file: %v", removeErr)
		}
	}()
	
	// Set the scanner path to the temporary file
	entity.scannerPath = tmpFile.Name()
	
	// Set workDir to an existing directory
	entity.workDir = os.TempDir()
	
	// Test with valid configuration
	err = entity.Initialize()
	assert.NoError(t, err)
}

// TestSonarScannerEntity_Cleanup tests the Cleanup method.
func TestSonarScannerEntity_Cleanup(t *testing.T) {
	cfg := &config.ScannerConfig{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewSonarScannerEntity(cfg, logger)
	
	// Test cleanup with empty tempDir
	err := entity.Cleanup()
	assert.NoError(t, err)
	
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "scanner_test")
	assert.NoError(t, err)
	
	// Set tempDir to the temporary directory
	entity.tempDir = tmpDir
	
	// Test cleanup with valid tempDir
	err = entity.Cleanup()
	assert.NoError(t, err)
	
	// Verify that the directory was removed
	_, err = os.Stat(tmpDir)
	assert.True(t, os.IsNotExist(err))
}

// TestSonarScannerEntity_parseOutput tests the parseOutput method.
func TestSonarScannerEntity_parseOutput(t *testing.T) {
	cfg := &config.ScannerConfig{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewSonarScannerEntity(cfg, logger)
	
	tests := []struct {
		name           string
		output         string
		expectedResult *ScanResult
	}{
		{
			name: "successful analysis with metrics",
			output: `INFO: Project key: test-project
INFO: Analysis report generated in 123ms
INFO: ANALYSIS SUCCESSFUL, you can browse task?id=AXyZ123
INFO: Note that you will be able to access the updated dashboard once the server has processed the submitted analysis report
INFO: More about the report processing at http://localhost:9000/api/ce/task?id=AXyZ123
5 issues found
Coverage 85.5%
Duplicated lines 2.3%
Lines of code 1500
Cyclomatic complexity 45
Quality gate PASSED`,
			expectedResult: &ScanResult{
				AnalysisID: "AXyZ123",
				ProjectKey: "test-project",
				Metrics: map[string]string{
					"issues_count":          "5",
					"coverage":              "85.5",
					"duplicated_lines":      "2.3",
					"lines_of_code":         "1500",
					"cyclomatic_complexity": "45",
					"quality_gate":          "PASSED",
					"report_url":            "http://localhost:9000/api/ce/task?id=AXyZ123",
					"task_url":              "http://localhost:9000/api/ce/task?id=AXyZ123",
				},
				Errors: []string{},
			},
		},
		{
			name: "analysis with errors and warnings",
			output: `INFO: Project key: error-project
ERROR: Invalid configuration
ERROR: Missing required property
WARN: Deprecated property used
WARN: Performance issue detected
Quality gate FAILED`,
			expectedResult: &ScanResult{
				ProjectKey: "error-project",
				Metrics: map[string]string{
					"quality_gate": "FAILED",
					"warnings":     "2",
				},
				Errors: []string{"Invalid configuration", "Missing required property", "Warning: Deprecated property used", "Warning: Performance issue detected"},
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ScanResult{
				Errors:  make([]string, 0),
				Metrics: make(map[string]string),
			}
			
			err := entity.parseOutput(tt.output, result)
			assert.NoError(t, err)
			
			assert.Equal(t, tt.expectedResult.AnalysisID, result.AnalysisID)
			assert.Equal(t, tt.expectedResult.ProjectKey, result.ProjectKey)
			assert.Equal(t, tt.expectedResult.Errors, result.Errors)
			assert.Equal(t, tt.expectedResult.Metrics, result.Metrics)
		})
	}
}

// TestSonarScannerEntity_handleExecutionError tests the handleExecutionError method.
func TestSonarScannerEntity_handleExecutionError(t *testing.T) {
	cfg := &config.ScannerConfig{
		Timeout: 30 * time.Second,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewSonarScannerEntity(cfg, logger)
	
	tests := []struct {
		name          string
		err           error
		output        string
		expectedError string
		expectedCode  int
	}{
		{
			name:          "timeout error",
			err:           context.DeadlineExceeded,
			output:        "Scanner output before timeout",
			expectedError: "execution timed out",
			expectedCode:  -1,
		},
		{
			name:          "cancellation error",
			err:           context.Canceled,
			output:        "Scanner output before cancellation",
			expectedError: "execution cancelled",
			expectedCode:  -1,
		},
		{
			name:          "exit error with code 1",
			err:           &mockExitError{exitCode: 1},
			output:        "Quality gate failed",
			expectedError: "exit status 1",
			expectedCode:  1,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ScanResult{
				Errors:  make([]string, 0),
				Metrics: make(map[string]string),
			}
			
			resultOut, err := entity.handleExecutionError(tt.err, tt.output, result)
			
			assert.NotNil(t, err)
			assert.NotNil(t, resultOut)
			assert.False(t, resultOut.Success)
			
			if scannerErr, ok := err.(*ScannerError); ok {
				assert.Equal(t, tt.expectedCode, scannerErr.ExitCode)
				assert.Contains(t, scannerErr.ErrorMsg, tt.expectedError)
				assert.Equal(t, tt.output, scannerErr.Output)
			} else {
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// TestSonarScannerEntity_KillProcess tests the KillProcess method.
func TestSonarScannerEntity_KillProcess(t *testing.T) {
	cfg := &config.ScannerConfig{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewSonarScannerEntity(cfg, logger)
	
	// Test with nil command
	err := entity.KillProcess(nil)
	assert.NoError(t, err)
	
	// Test with command without process
	cmd := &exec.Cmd{}
	err = entity.KillProcess(cmd)
	assert.NoError(t, err)
}

// TestSonarScannerEntity_ExecuteWithTimeout tests the ExecuteWithTimeout method.
// TestSonarScannerEntity_Execute tests the Execute method.
func TestSonarScannerEntity_Execute(t *testing.T) {
	cfg := &config.ScannerConfig{
		ScannerURL:     "http://localhost:9000",
		ScannerVersion: "4.6.2",
		JavaOpts:       "-Xmx512m",
		Properties:     map[string]string{"sonar.host.url": "http://localhost:9000"},
		Timeout:        30 * time.Second,
		WorkDir:        "/tmp",
		TempDir:        "/tmp",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewSonarScannerEntity(cfg, logger)
	
	// Test execution without scanner initialization
	ctx := context.Background()
	result, err := entity.Execute(ctx)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "scanner initialization failed")
	
	// Test execution with invalid scanner path
	entity.scannerPath = "/nonexistent/scanner"
	result, err = entity.Execute(ctx)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "configuration validation failed")
}

func TestSonarScannerEntity_ExecuteWithTimeout(t *testing.T) {
	cfg := &config.ScannerConfig{
		ScannerURL:     "http://localhost:9000",
		ScannerVersion: "4.6.2",
		JavaOpts:       "-Xmx512m",
		Properties:     map[string]string{"sonar.host.url": "http://localhost:9000"},
		WorkDir:        "/tmp",
		TempDir:        "/tmp",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewSonarScannerEntity(cfg, logger)
	
	// Test timeout functionality
	ctx := context.Background()
	timeout := 1 * time.Millisecond // Very short timeout to trigger timeout
	
	// This should fail because scanner is not initialized
	result, err := entity.ExecuteWithTimeout(ctx, timeout)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "scanner initialization failed")
}