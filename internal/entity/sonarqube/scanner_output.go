package sonarqube

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// parseOutput parses the scanner output to extract structured information.
func (s *SonarScannerEntity) parseOutput(output string, result *ScanResult) error {
	lines := strings.Split(output, "\n")

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

		if strings.Contains(line, "ANALYSIS SUCCESSFUL") {
			if matches := analysisIDRegex.FindStringSubmatch(line); len(matches) > 1 {
				result.AnalysisID = matches[1]
			}
		}

		if strings.HasPrefix(line, "INFO: Project key:") {
			result.ProjectKey = strings.TrimSpace(strings.TrimPrefix(line, "INFO: Project key:"))
		}

		if matches := executionTimeRegex.FindStringSubmatch(line); len(matches) > 2 {
			result.Metrics["execution_time"] = matches[1] + matches[2]
		}

		if matches := memoryRegex.FindStringSubmatch(line); len(matches) > 2 {
			result.Metrics["memory_used"] = matches[1] + "M"
			result.Metrics["memory_total"] = matches[2] + "M"
		}

		if matches := issuesRegex.FindStringSubmatch(line); len(matches) > 1 {
			result.Metrics["issues_count"] = matches[1]
		}

		if matches := coverageRegex.FindStringSubmatch(line); len(matches) > 1 {
			result.Metrics["coverage"] = matches[1]
		}

		if matches := duplicatedRegex.FindStringSubmatch(line); len(matches) > 1 {
			result.Metrics["duplicated_lines"] = matches[1]
		}

		if matches := linesRegex.FindStringSubmatch(line); len(matches) > 1 {
			result.Metrics["lines_of_code"] = matches[1]
		}

		if matches := complexityRegex.FindStringSubmatch(line); len(matches) > 1 {
			result.Metrics["cyclomatic_complexity"] = matches[1]
		}

		if matches := technicalDebtRegex.FindStringSubmatch(line); len(matches) > 1 {
			result.Metrics["technical_debt"] = matches[1]
		}

		if strings.Contains(line, "Quality gate") {
			if strings.Contains(line, "PASSED") {
				result.Metrics["quality_gate"] = "PASSED"
			} else if strings.Contains(line, "FAILED") {
				result.Metrics["quality_gate"] = "FAILED"
			}
		}

		if strings.HasPrefix(line, "INFO: More about the report processing at") {
			url := strings.TrimSpace(strings.TrimPrefix(line, "INFO: More about the report processing at"))
			result.Metrics["report_url"] = url
		}

		if strings.Contains(line, "http") && strings.Contains(line, "api/ce/task") {
			taskURLRegex := regexp.MustCompile(`(http[s]?://[^\s]+)`)
			if matches := taskURLRegex.FindStringSubmatch(line); len(matches) > 1 {
				result.Metrics["task_url"] = matches[1]
			}
		}

		if strings.HasPrefix(line, "ERROR:") {
			errorMsg := strings.TrimSpace(strings.TrimPrefix(line, "ERROR:"))
			if errorMsg != "" {
				result.Errors = append(result.Errors, errorMsg)
			}
		}

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
			}
		}

		if strings.HasPrefix(line, "INFO:") && s.logger.Enabled(context.TODO(), slog.LevelDebug) {
			infoMsg := strings.TrimSpace(strings.TrimPrefix(line, "INFO:"))
			if infoMsg != "" {
				s.logger.Debug("Scanner info", "message", infoMsg)
			}
		}

		if strings.Contains(line, "Analyzing") || strings.Contains(line, "Processed") {
			progressRegex := regexp.MustCompile(`(\d+)/(\d+)\s+files`)
			if matches := progressRegex.FindStringSubmatch(line); len(matches) > 2 {
				result.Metrics["files_processed"] = matches[1]
				result.Metrics["files_total"] = matches[2]
			}
		}
	}

	s.logger.Debug("Parsing completed",
		"metricsCount", len(result.Metrics),
		"errorsCount", len(result.Errors),
		"hasAnalysisId", result.AnalysisID != "",
		"hasProjectKey", result.ProjectKey != "")

	return nil
}

// handleExecutionError handles errors that occur during scanner execution.
func (s *SonarScannerEntity) handleExecutionError(err error, output string, result *ScanResult) (*ScanResult, error) {
	s.logger.Error("Scanner execution failed", "error", err, "outputLength", len(output))

	if len(output) > 0 {
		const maxOutputLogLength = 2000
		outputForLog := output
		if len(output) > maxOutputLogLength {
			outputForLog = output[len(output)-maxOutputLogLength:]
			s.logger.Error("Scanner output (last 2000 chars)", "output", outputForLog)
		} else {
			s.logger.Error("Scanner output", "output", outputForLog)
		}
	}

	if errors.Is(err, context.DeadlineExceeded) {
		result.Errors = append(result.Errors, fmt.Sprintf("Scanner execution timed out after %v", s.config.Timeout))
		s.logger.Error("Scanner execution timed out", "timeout", s.config.Timeout)

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

	if errors.Is(err, context.Canceled) {
		result.Errors = append(result.Errors, "Scanner execution was cancelled")
		return result, &ScannerError{
			ExitCode: -1,
			Output:   output,
			ErrorMsg: "execution cancelled",
		}
	}

	if exitError, ok := err.(*exec.ExitError); ok {
		exitCode := exitError.ExitCode()
		errorAnalysis := s.analyzeErrorOutput(output)
		errorMsg := s.getExitCodeMessage(exitCode, errorAnalysis)
		result.Errors = append(result.Errors, errorMsg)

		if len(errorAnalysis) > 0 {
			result.Errors = append(result.Errors, errorAnalysis...)

			bslTokenizationError := false
			for _, errMsg := range errorAnalysis {
				if strings.Contains(errMsg, "BSL tokenization error") {
					bslTokenizationError = true

					bslFilePattern := regexp.MustCompile(`([^/\s]+/[^/\s]*\.bsl)`)
					matches := bslFilePattern.FindAllString(output, -1)

					if len(matches) > 0 {
						for _, filePath := range matches {
							s.logger.Info("Attempting to fix BSL tokenization issue", "file", filePath)
							if fixErr := s.FixBSLTokenizationIssues(filePath); fixErr != nil {
								s.logger.Error("Failed to fix BSL file", "file", filePath, "error", fixErr)
								result.Errors = append(result.Errors, fmt.Sprintf("FAILED TO FIX: %s - %v", filePath, fixErr))
							} else {
								result.Errors = append(result.Errors, fmt.Sprintf("ATTEMPTED FIX: %s - file has been automatically corrected", filePath))
								result.Errors = append(result.Errors, "RECOMMENDATION: Re-run the scanner to check if the issue is resolved")
							}
						}
					}

					result.Errors = append(result.Errors, "RECOMMENDATION: Check BSL file encoding (should be UTF-8) and verify syntax correctness")
					result.Errors = append(result.Errors, "RECOMMENDATION: Try excluding problematic BSL files using sonar.exclusions property")
					result.Errors = append(result.Errors, "RECOMMENDATION: Update BSL plugin to latest version or check plugin compatibility")
				}
				if strings.Contains(errMsg, "BSL plugin error") {
					result.Errors = append(result.Errors, "RECOMMENDATION: Verify BSL plugin installation and version compatibility")
					result.Errors = append(result.Errors, "RECOMMENDATION: Check SonarQube server logs for BSL plugin issues")
				}
			}

			if bslTokenizationError {
				exclusions := s.SuggestBSLExclusions(output)
				if len(exclusions) > 0 {
					result.Errors = append(result.Errors, "EXCLUSION SUGGESTIONS:")
					result.Errors = append(result.Errors, exclusions...)
				}
			}
		} else {
			result.Errors = append(result.Errors, "No specific error patterns detected. Check scanner output above for details.")
		}

		switch exitCode {
		case 1:
			s.logger.Error("Scanner failed with quality gate failure or analysis errors",
				"exitCode", exitCode, "errorCount", len(errorAnalysis), "errorAnalysis", errorAnalysis)
		case 2:
			s.logger.Error("Scanner failed with invalid configuration",
				"exitCode", exitCode, "configErrors", errorAnalysis)
		case 3:
			s.logger.Error("Scanner failed with internal error",
				"exitCode", exitCode, "errorAnalysis", errorAnalysis)
		case 4:
			s.logger.Error("Scanner failed with resource issues",
				"exitCode", exitCode, "errorAnalysis", errorAnalysis)
		default:
			s.logger.Error("Scanner failed with unknown error",
				"exitCode", exitCode, "errorAnalysis", errorAnalysis)
		}

		return result, &ScannerError{
			ExitCode: exitCode,
			Output:   output,
			ErrorMsg: errorMsg,
		}
	}

	result.Errors = append(result.Errors, "Unexpected scanner execution error: "+err.Error())
	s.logger.Error("Unexpected scanner execution error", "error", err, "type", fmt.Sprintf("%T", err))
	return result, fmt.Errorf("scanner execution failed: %w", err)
}

// analyzeErrorOutput analyzes scanner output to extract specific error information.
func (s *SonarScannerEntity) analyzeErrorOutput(output string) []string {
	var errs []string
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "Unauthorized") || strings.Contains(line, "401") {
			errs = append(errs, "Authentication failed: Invalid token or credentials")
		}
		if strings.Contains(line, "Connection refused") || strings.Contains(line, "ConnectException") {
			errs = append(errs, "Network error: Cannot connect to SonarQube server")
		}
		if strings.Contains(line, "Project key") && strings.Contains(line, "invalid") {
			errs = append(errs, "Invalid project key configuration")
		}
		if strings.Contains(line, "No sources found") || strings.Contains(line, "No files to analyze") {
			errs = append(errs, "No source files found for analysis")
		}
		if strings.Contains(line, "OutOfMemoryError") {
			errs = append(errs, "Insufficient memory: Increase JAVA_OPTS heap size")
		}
		if strings.Contains(line, "Permission denied") || strings.Contains(line, "Access denied") {
			errs = append(errs, "File system permission error")
		}
		if strings.Contains(line, "Quality gate") && strings.Contains(line, "FAILED") {
			errs = append(errs, "Quality gate failed: Code quality standards not met")
		}
		if strings.Contains(line, "version") && strings.Contains(line, "not supported") {
			errs = append(errs, "SonarQube server version compatibility issue")
		}
		if strings.Contains(line, "plugin") && strings.Contains(line, "failed") {
			errs = append(errs, "Scanner plugin error")
		}
		if strings.HasPrefix(line, "ERROR:") {
			errorMsg := strings.TrimSpace(strings.TrimPrefix(line, "ERROR:"))
			if errorMsg != "" {
				errs = append(errs, "Scanner error: "+errorMsg)
			}
		}
		if strings.HasPrefix(line, "WARN:") {
			warnMsg := strings.TrimSpace(strings.TrimPrefix(line, "WARN:"))
			if warnMsg != "" && (strings.Contains(warnMsg, "fail") || strings.Contains(warnMsg, "error") || strings.Contains(warnMsg, "invalid")) {
				errs = append(errs, "Scanner warning: "+warnMsg)
			}
		}
		if strings.Contains(line, "EXECUTION FAILURE") {
			errs = append(errs, "Scanner execution failure detected")
		}
		if strings.Contains(line, "timeout") || strings.Contains(line, "timed out") {
			errs = append(errs, "Timeout error: "+line)
		}
		if strings.Contains(line, "SSL") || strings.Contains(line, "TLS") || strings.Contains(line, "certificate") {
			errs = append(errs, "SSL/TLS connection error: "+line)
		}
		if strings.Contains(line, "No space left") || strings.Contains(line, "disk full") {
			errs = append(errs, "Disk space error: "+line)
		}
		if strings.Contains(line, "ClassNotFoundException") || strings.Contains(line, "NoClassDefFoundError") {
			errs = append(errs, "Java classpath error: "+line)
		}
		if strings.Contains(line, "sonar-project.properties") && (strings.Contains(line, "not found") || strings.Contains(line, "missing")) {
			errs = append(errs, "Configuration file error: "+line)
		}
		if strings.Contains(line, "git") && (strings.Contains(line, "not found") || strings.Contains(line, "failed")) {
			errs = append(errs, "Git-related error: "+line)
		}
		if strings.Contains(line, "Analysis failed") || strings.Contains(line, "analysis error") {
			errs = append(errs, "Analysis error: "+line)
		}
		if strings.Contains(line, "500") || strings.Contains(line, "502") || strings.Contains(line, "503") || strings.Contains(line, "504") {
			errs = append(errs, "Server error: "+line)
		}
		if strings.Contains(line, "java.lang.IllegalStateException") && strings.Contains(line, "Tokens of file") && strings.Contains(line, ".bsl") {
			errs = append(errs, "BSL tokenization error: File contains invalid token sequence - check file encoding and syntax")
		}
		if strings.Contains(line, "com.github._1c_syntax.bsl") {
			errs = append(errs, "BSL plugin error: "+line)
		}
		if strings.Contains(line, ".bsl") && (strings.Contains(line, "encoding") || strings.Contains(line, "charset")) {
			errs = append(errs, "BSL encoding error: "+line)
		}
		if strings.Contains(line, ".bsl") && strings.Contains(line, "syntax") {
			errs = append(errs, "BSL syntax error: "+line)
		}
		if strings.Contains(line, "1C") || strings.Contains(line, "1c") {
			if strings.Contains(line, "error") || strings.Contains(line, "ERROR") || strings.Contains(line, "failed") {
				errs = append(errs, "1C platform error: "+line)
			}
		}
	}

	return errs
}

// analyzeTimeoutError analyzes output for timeout-specific context.
func (s *SonarScannerEntity) analyzeTimeoutError(output string) []string {
	var diagnostics []string
	lines := strings.Split(output, "\n")

	found := false
	for i := len(lines) - 1; i >= 0 && !found; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

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

	diagnostics = append(diagnostics, "Consider increasing timeout value or optimizing project size")
	return diagnostics
}

// getExitCodeMessage returns a descriptive message for scanner exit codes.
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

	if len(errorAnalysis) > 0 {
		baseMessage += " (" + strings.Join(errorAnalysis, ", ") + ")"
	}

	return baseMessage
}
