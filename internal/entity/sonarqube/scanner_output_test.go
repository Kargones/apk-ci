package sonarqube

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestEntity() *SonarScannerEntity {
	cfg := &config.ScannerConfig{Timeout: 30 * time.Second}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewSonarScannerEntity(cfg, logger)
}

func TestParseOutput_ExecutionTime(t *testing.T) {
	e := newTestEntity()
	tests := []struct {
		name   string
		output string
		key    string
		value  string
	}{
		{"seconds", "Total time: 12.345 s", "execution_time", "12.345s"},
		{"minutes", "Total time: 2:30 min", "execution_time", "2:30min"},
		{"memory", "Final Memory: 128M/512M", "memory_used", "128M"},
		{"memory_total", "Final Memory: 128M/512M", "memory_total", "512M"},
		{"files_progress", "Analyzing 15/100 files", "files_processed", "15"},
		{"files_total", "Processed 50/200 files", "files_total", "200"},
		{"technical_debt", "Technical Debt 5d", "technical_debt", "5d"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ScanResult{Errors: []string{}, Metrics: map[string]string{}}
			err := e.parseOutput(tt.output, result)
			require.NoError(t, err)
			assert.Equal(t, tt.value, result.Metrics[tt.key])
		})
	}
}

func TestParseOutput_ErrorsAndWarnings(t *testing.T) {
	e := newTestEntity()
	output := "ERROR: First error\nERROR: Second error\nWARN: A warning\nWARN: Another warning"
	result := &ScanResult{Errors: []string{}, Metrics: map[string]string{}}
	e.parseOutput(output, result)

	assert.Contains(t, result.Errors, "First error")
	assert.Contains(t, result.Errors, "Second error")
	assert.Contains(t, result.Errors, "Warning: A warning")
	assert.Contains(t, result.Errors, "Warning: Another warning")
	assert.Equal(t, "2", result.Metrics["warnings"])
}

func TestParseOutput_TaskURL(t *testing.T) {
	e := newTestEntity()
	output := "INFO: More about the report at https://sonar.example.com/api/ce/task?id=ABC123"
	result := &ScanResult{Errors: []string{}, Metrics: map[string]string{}}
	e.parseOutput(output, result)
	assert.Equal(t, "https://sonar.example.com/api/ce/task?id=ABC123", result.Metrics["task_url"])
}

func TestParseOutput_EmptyOutput(t *testing.T) {
	e := newTestEntity()
	result := &ScanResult{Errors: []string{}, Metrics: map[string]string{}}
	err := e.parseOutput("", result)
	assert.NoError(t, err)
	assert.Empty(t, result.Errors)
}

func TestAnalyzeErrorOutput(t *testing.T) {
	e := newTestEntity()
	tests := []struct {
		name     string
		output   string
		contains string
	}{
		{"auth", "Unauthorized access 401", "Authentication failed"},
		{"connection", "Connection refused to server", "Network error"},
		{"no_sources", "No sources found in project", "No source files found"},
		{"oom", "java.lang.OutOfMemoryError", "Insufficient memory"},
		{"permission", "Permission denied: /opt/scanner", "File system permission error"},
		{"quality_gate", "Quality gate FAILED", "Quality gate failed"},
		{"version", "version 7.9 not supported", "version compatibility"},
		{"plugin", "plugin initialization failed", "Scanner plugin error"},
		{"execution_failure", "EXECUTION FAILURE", "Scanner execution failure"},
		{"timeout_line", "Connection timed out after 30s", "Timeout error"},
		{"ssl", "SSL handshake failed", "SSL/TLS connection error"},
		{"disk", "No space left on device", "Disk space error"},
		{"classpath", "ClassNotFoundException: org.sonar", "Java classpath error"},
		{"config_file", "sonar-project.properties not found", "Configuration file error"},
		{"git_error", "git fetch failed", "Git-related error"},
		{"analysis_failed", "Analysis failed due to errors", "Analysis error"},
		{"server_500", "HTTP 500 Internal Server Error", "Server error"},
		{"server_502", "HTTP 502 Bad Gateway", "Server error"},
		{"bsl_token", "java.lang.IllegalStateException: Tokens of file test.bsl invalid", "BSL tokenization error"},
		{"bsl_plugin", "com.github._1c_syntax.bsl.parser error", "BSL plugin error"},
		{"bsl_encoding", "test.bsl encoding issue charset", "BSL encoding error"},
		{"bsl_syntax", "test.bsl syntax error", "BSL syntax error"},
		{"1c_error", "1C platform error occurred", "1C platform error"},
		{"error_prefix", "ERROR: Something went wrong", "Scanner error: Something went wrong"},
		{"warn_with_fail", "WARN: validation failed", "Scanner warning"},
		{"project_key_invalid", "Project key is invalid", "Invalid project key"},
		{"access_denied", "Access denied to resource", "File system permission error"},
		{"noclass", "NoClassDefFoundError in scanner", "Java classpath error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := e.analyzeErrorOutput(tt.output)
			found := false
			for _, err := range errs {
				if assert.ObjectsAreEqual(tt.contains, "") || contains(err, tt.contains) {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected error containing %q in %v", tt.contains, errs)
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestAnalyzeErrorOutput_Empty(t *testing.T) {
	e := newTestEntity()
	errs := e.analyzeErrorOutput("")
	assert.Empty(t, errs)
}

func TestAnalyzeTimeoutError(t *testing.T) {
	e := newTestEntity()
	tests := []struct {
		name     string
		output   string
		contains string
	}{
		{"analyzing", "INFO: Starting\nAnalyzing files...", "file analysis phase"},
		{"uploading", "INFO: Starting\nUploading results", "result upload"},
		{"downloading", "INFO: Starting\nDownloading plugins", "dependency download"},
		{"starting", "Starting scanner initialization", "scanner initialization"},
		{"empty", "", "Consider increasing timeout"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := e.analyzeTimeoutError(tt.output)
			assert.NotEmpty(t, diags)
			found := false
			for _, d := range diags {
				if findSubstring(d, tt.contains) {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected %q in %v", tt.contains, diags)
		})
	}
}

func TestAnalyzeTimeoutError_Sending(t *testing.T) {
	e := newTestEntity()
	diags := e.analyzeTimeoutError("Sending report to server")
	found := false
	for _, d := range diags {
		if findSubstring(d, "result upload") {
			found = true
		}
	}
	assert.True(t, found)
}

func TestGetExitCodeMessage(t *testing.T) {
	e := newTestEntity()
	tests := []struct {
		code     int
		contains string
	}{
		{1, "Quality gate failure"},
		{2, "Invalid configuration"},
		{3, "Internal error"},
		{4, "Insufficient memory"},
		{5, "Network or connectivity"},
		{99, "exit code 99"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("code_%d", tt.code), func(t *testing.T) {
			msg := e.getExitCodeMessage(tt.code, nil)
			assert.Contains(t, msg, tt.contains)
		})
	}
}

func TestGetExitCodeMessage_WithAnalysis(t *testing.T) {
	e := newTestEntity()
	msg := e.getExitCodeMessage(1, []string{"auth failed", "timeout"})
	assert.Contains(t, msg, "auth failed")
	assert.Contains(t, msg, "timeout")
}

func TestHandleExecutionError_DeadlineExceeded(t *testing.T) {
	e := newTestEntity()
	result := &ScanResult{Errors: []string{}, Metrics: map[string]string{}}
	_, err := e.handleExecutionError(context.DeadlineExceeded, "Analyzing files", result)
	scanErr, ok := err.(*ScannerError)
	require.True(t, ok)
	assert.Equal(t, -1, scanErr.ExitCode)
	assert.Contains(t, scanErr.ErrorMsg, "timed out")
}

func TestHandleExecutionError_Canceled(t *testing.T) {
	e := newTestEntity()
	result := &ScanResult{Errors: []string{}, Metrics: map[string]string{}}
	_, err := e.handleExecutionError(context.Canceled, "", result)
	scanErr, ok := err.(*ScannerError)
	require.True(t, ok)
	assert.Equal(t, -1, scanErr.ExitCode)
	assert.Contains(t, scanErr.ErrorMsg, "cancelled")
}

func TestHandleExecutionError_ExitError(t *testing.T) {
	e := newTestEntity()
	// Use a real exec.ExitError by running a command that fails
	cmd := exec.Command("sh", "-c", "exit 2")
	execErr := cmd.Run()
	require.Error(t, execErr)

	result := &ScanResult{Errors: []string{}, Metrics: map[string]string{}}
	_, err := e.handleExecutionError(execErr, "ERROR: config invalid", result)
	scanErr, ok := err.(*ScannerError)
	require.True(t, ok)
	assert.Equal(t, 2, scanErr.ExitCode)
}

func TestHandleExecutionError_ExitCode1(t *testing.T) {
	e := newTestEntity()
	cmd := exec.Command("sh", "-c", "exit 1")
	execErr := cmd.Run()

	result := &ScanResult{Errors: []string{}, Metrics: map[string]string{}}
	_, err := e.handleExecutionError(execErr, "Quality gate FAILED", result)
	scanErr, ok := err.(*ScannerError)
	require.True(t, ok)
	assert.Equal(t, 1, scanErr.ExitCode)
}

func TestHandleExecutionError_ExitCode3(t *testing.T) {
	e := newTestEntity()
	cmd := exec.Command("sh", "-c", "exit 3")
	execErr := cmd.Run()

	result := &ScanResult{Errors: []string{}, Metrics: map[string]string{}}
	e.handleExecutionError(execErr, "Internal scanner error", result)
}

func TestHandleExecutionError_ExitCode4(t *testing.T) {
	e := newTestEntity()
	cmd := exec.Command("sh", "-c", "exit 4")
	execErr := cmd.Run()

	result := &ScanResult{Errors: []string{}, Metrics: map[string]string{}}
	e.handleExecutionError(execErr, "OutOfMemoryError", result)
}

func TestHandleExecutionError_UnknownExitCode(t *testing.T) {
	e := newTestEntity()
	cmd := exec.Command("sh", "-c", "exit 42")
	execErr := cmd.Run()

	result := &ScanResult{Errors: []string{}, Metrics: map[string]string{}}
	_, err := e.handleExecutionError(execErr, "", result)
	scanErr, ok := err.(*ScannerError)
	require.True(t, ok)
	assert.Equal(t, 42, scanErr.ExitCode)
}

func TestHandleExecutionError_BSLTokenization(t *testing.T) {
	e := newTestEntity()
	cmd := exec.Command("sh", "-c", "exit 1")
	execErr := cmd.Run()

	output := "java.lang.IllegalStateException: Tokens of file src/Module.bsl are not valid"
	result := &ScanResult{Errors: []string{}, Metrics: map[string]string{}}
	e.handleExecutionError(execErr, output, result)

	// Should contain BSL-related recommendations
	foundBSL := false
	for _, errMsg := range result.Errors {
		if findSubstring(errMsg, "BSL") || findSubstring(errMsg, "RECOMMENDATION") || findSubstring(errMsg, "EXCLUSION") {
			foundBSL = true
			break
		}
	}
	assert.True(t, foundBSL, "Expected BSL-related errors in %v", result.Errors)
}

func TestHandleExecutionError_BSLPlugin(t *testing.T) {
	e := newTestEntity()
	cmd := exec.Command("sh", "-c", "exit 1")
	execErr := cmd.Run()

	output := "com.github._1c_syntax.bsl.parser crashed"
	result := &ScanResult{Errors: []string{}, Metrics: map[string]string{}}
	e.handleExecutionError(execErr, output, result)

	foundPlugin := false
	for _, errMsg := range result.Errors {
		if findSubstring(errMsg, "BSL plugin") || findSubstring(errMsg, "RECOMMENDATION") {
			foundPlugin = true
			break
		}
	}
	assert.True(t, foundPlugin)
}

func TestHandleExecutionError_UnexpectedError(t *testing.T) {
	e := newTestEntity()
	result := &ScanResult{Errors: []string{}, Metrics: map[string]string{}}
	_, err := e.handleExecutionError(fmt.Errorf("some weird error"), "", result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scanner execution failed")
}

func TestHandleExecutionError_LongOutput(t *testing.T) {
	e := newTestEntity()
	// Generate output > 2000 chars
	longOutput := ""
	for i := 0; i < 300; i++ {
		longOutput += "INFO: Processing line number " + fmt.Sprintf("%d", i) + "\n"
	}
	result := &ScanResult{Errors: []string{}, Metrics: map[string]string{}}
	e.handleExecutionError(context.DeadlineExceeded, longOutput, result)
	// Just verify it doesn't panic
}

func TestHandleExecutionError_NoErrorAnalysis(t *testing.T) {
	e := newTestEntity()
	cmd := exec.Command("sh", "-c", "exit 1")
	execErr := cmd.Run()

	result := &ScanResult{Errors: []string{}, Metrics: map[string]string{}}
	e.handleExecutionError(execErr, "some random output", result)

	foundNoPattern := false
	for _, errMsg := range result.Errors {
		if findSubstring(errMsg, "No specific error patterns") {
			foundNoPattern = true
			break
		}
	}
	assert.True(t, foundNoPattern)
}
