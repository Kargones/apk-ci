// Package sonarqube provides error types for SonarQube integration.
// This package defines the error types that can occur during SonarQube operations,
// including API errors, scanner errors, and validation errors.
package sonarqube

// Error represents errors from SonarQube API.
// This error type is used when SonarQube API returns an error response.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details"`
}

// Error returns the error message for SonarQubeError.
func (e *Error) Error() string {
	if e.Details != "" {
		return e.Message + ": " + e.Details
	}
	return e.Message
}

// ScannerError represents errors from sonar-scanner execution.
// This error type is used when the sonar-scanner process fails.
type ScannerError struct {
	ExitCode int    `json:"exitCode"`
	Output   string `json:"output"`
	ErrorMsg string `json:"error"`
}

// Error returns the error message for ScannerError.
func (e *ScannerError) Error() string {
	if e.ErrorMsg != "" {
		return e.ErrorMsg
	}
	return "Scanner execution failed with exit code " + string(rune(e.ExitCode))
}

// ValidationError represents validation errors for SonarQube operations.
// This error type is used when input parameters or configuration fail validation.
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

// Error returns the error message for ValidationError.
func (e *ValidationError) Error() string {
	return e.Message
}
