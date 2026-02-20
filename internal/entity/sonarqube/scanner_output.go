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

// metricMatcher extracts a metric value from a line using a regex.
type metricMatcher struct {
	regex   *regexp.Regexp
	extract func(matches []string, result *ScanResult)
}

// Compiled regexes for parseOutput (allocated once).
var (
	reAnalysisID    = regexp.MustCompile(`task\?id=([A-Za-z0-9_-]+)`)
	reIssues        = regexp.MustCompile(`(\d+)\s+issues?\s+found`)
	reCoverage      = regexp.MustCompile(`Coverage\s+(\d+\.\d+)%`)
	reDuplicated    = regexp.MustCompile(`Duplicated lines\s+(\d+\.\d+)%`)
	reLines         = regexp.MustCompile(`Lines of code\s+(\d+)`)
	reComplexity    = regexp.MustCompile(`Cyclomatic complexity\s+(\d+)`)
	reTechnicalDebt = regexp.MustCompile(`Technical Debt\s+([0-9]+[dhm]+)`)
	reExecutionTime = regexp.MustCompile(`Total time:\s*([0-9:.]+)\s*(s|min)`)
	reMemory        = regexp.MustCompile(`Final Memory:\s*([0-9]+)M/([0-9]+)M`)
	reTaskURL       = regexp.MustCompile(`(http[s]?://[^\s]+)`)
	reProgress      = regexp.MustCompile(`(\d+)/(\d+)\s+files`)
)

// metricMatchers defines regex-based metric extraction rules.
var metricMatchers = []metricMatcher{
	{regex: reExecutionTime, extract: func(m []string, r *ScanResult) {
		if len(m) > 2 { r.Metrics["execution_time"] = m[1] + m[2] }
	}},
	{regex: reMemory, extract: func(m []string, r *ScanResult) {
		if len(m) > 2 { r.Metrics["memory_used"] = m[1] + "M"; r.Metrics["memory_total"] = m[2] + "M" }
	}},
	{regex: reIssues, extract: func(m []string, r *ScanResult) {
		if len(m) > 1 { r.Metrics["issues_count"] = m[1] }
	}},
	{regex: reCoverage, extract: func(m []string, r *ScanResult) {
		if len(m) > 1 { r.Metrics["coverage"] = m[1] }
	}},
	{regex: reDuplicated, extract: func(m []string, r *ScanResult) {
		if len(m) > 1 { r.Metrics["duplicated_lines"] = m[1] }
	}},
	{regex: reLines, extract: func(m []string, r *ScanResult) {
		if len(m) > 1 { r.Metrics["lines_of_code"] = m[1] }
	}},
	{regex: reComplexity, extract: func(m []string, r *ScanResult) {
		if len(m) > 1 { r.Metrics["cyclomatic_complexity"] = m[1] }
	}},
	{regex: reTechnicalDebt, extract: func(m []string, r *ScanResult) {
		if len(m) > 1 { r.Metrics["technical_debt"] = m[1] }
	}},
}

// parseLineMetrics extracts regex-based metrics from a single line.
func parseLineMetrics(line string, result *ScanResult) {
	for _, m := range metricMatchers {
		if matches := m.regex.FindStringSubmatch(line); matches != nil {
			m.extract(matches, result)
		}
	}
}

// parseLineSpecial handles special (non-regex) line parsing.
func (s *SonarScannerEntity) parseLineSpecial(line string, result *ScanResult) {
	if strings.Contains(line, "ANALYSIS SUCCESSFUL") {
		if matches := reAnalysisID.FindStringSubmatch(line); len(matches) > 1 {
			result.AnalysisID = matches[1]
		}
	}

	if strings.HasPrefix(line, "INFO: Project key:") {
		result.ProjectKey = strings.TrimSpace(strings.TrimPrefix(line, "INFO: Project key:"))
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
		if matches := reTaskURL.FindStringSubmatch(line); len(matches) > 1 {
			result.Metrics["task_url"] = matches[1]
		}
	}

	if (strings.Contains(line, "Analyzing") || strings.Contains(line, "Processed")) {
		if matches := reProgress.FindStringSubmatch(line); len(matches) > 2 {
			result.Metrics["files_processed"] = matches[1]
			result.Metrics["files_total"] = matches[2]
		}
	}
}

// parseLineLogEntries handles ERROR, WARN, and INFO prefix lines.
func (s *SonarScannerEntity) parseLineLogEntries(line string, result *ScanResult) {
	if strings.HasPrefix(line, "ERROR:") {
		if msg := strings.TrimSpace(strings.TrimPrefix(line, "ERROR:")); msg != "" {
			result.Errors = append(result.Errors, msg)
		}
	}

	if strings.HasPrefix(line, "WARN:") {
		if msg := strings.TrimSpace(strings.TrimPrefix(line, "WARN:")); msg != "" {
			if result.Metrics["warnings"] == "" {
				result.Metrics["warnings"] = "1"
			} else if count, err := strconv.Atoi(result.Metrics["warnings"]); err == nil {
				result.Metrics["warnings"] = strconv.Itoa(count + 1)
			}
			result.Errors = append(result.Errors, "Warning: "+msg)
		}
	}

	if strings.HasPrefix(line, "INFO:") && s.logger.Enabled(context.Background(), slog.LevelDebug) {
		if msg := strings.TrimSpace(strings.TrimPrefix(line, "INFO:")); msg != "" {
			s.logger.Debug("Scanner info", "message", msg)
		}
	}
}

// parseOutput parses the scanner output to extract structured information.
func (s *SonarScannerEntity) parseOutput(output string, result *ScanResult) error {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		parseLineMetrics(line, result)
		s.parseLineSpecial(line, result)
		s.parseLineLogEntries(line, result)
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

// errorPattern defines a pattern-based error detection rule.
type errorPattern struct {
	// match returns true if the line matches this pattern.
	match func(line string) bool
	// message returns the error message for the matched line.
	// If nil, a static message from staticMsg is used.
	message func(line string) string
	// staticMsg is used when message is nil.
	staticMsg string
}

// containsAll returns true if line contains all specified substrings.
func containsAll(line string, subs ...string) bool {
	for _, s := range subs {
		if !strings.Contains(line, s) {
			return false
		}
	}
	return true
}

// containsAny returns true if line contains at least one of the specified substrings.
func containsAny(line string, subs ...string) bool {
	for _, s := range subs {
		if strings.Contains(line, s) {
			return true
		}
	}
	return false
}

// errorPatterns defines all known error patterns for scanner output analysis.
// Order matters: patterns are checked sequentially for each line.
var errorPatterns = []errorPattern{
	{match: func(l string) bool { return containsAny(l, "Unauthorized", "401") }, staticMsg: "Authentication failed: Invalid token or credentials"},
	{match: func(l string) bool { return containsAny(l, "Connection refused", "ConnectException") }, staticMsg: "Network error: Cannot connect to SonarQube server"},
	{match: func(l string) bool { return containsAll(l, "Project key", "invalid") }, staticMsg: "Invalid project key configuration"},
	{match: func(l string) bool { return containsAny(l, "No sources found", "No files to analyze") }, staticMsg: "No source files found for analysis"},
	{match: func(l string) bool { return strings.Contains(l, "OutOfMemoryError") }, staticMsg: "Insufficient memory: Increase JAVA_OPTS heap size"},
	{match: func(l string) bool { return containsAny(l, "Permission denied", "Access denied") }, staticMsg: "File system permission error"},
	{match: func(l string) bool { return containsAll(l, "Quality gate", "FAILED") }, staticMsg: "Quality gate failed: Code quality standards not met"},
	{match: func(l string) bool { return containsAll(l, "version", "not supported") }, staticMsg: "SonarQube server version compatibility issue"},
	{match: func(l string) bool { return containsAll(l, "plugin", "failed") }, staticMsg: "Scanner plugin error"},
	{match: func(l string) bool { return strings.Contains(l, "EXECUTION FAILURE") }, staticMsg: "Scanner execution failure detected"},
	{match: func(l string) bool { return containsAny(l, "timeout", "timed out") }, message: func(l string) string { return "Timeout error: " + l }},
	{match: func(l string) bool { return containsAny(l, "SSL", "TLS", "certificate") }, message: func(l string) string { return "SSL/TLS connection error: " + l }},
	{match: func(l string) bool { return containsAny(l, "No space left", "disk full") }, message: func(l string) string { return "Disk space error: " + l }},
	{match: func(l string) bool { return containsAny(l, "ClassNotFoundException", "NoClassDefFoundError") }, message: func(l string) string { return "Java classpath error: " + l }},
	{match: func(l string) bool { return containsAll(l, "sonar-project.properties") && containsAny(l, "not found", "missing") }, message: func(l string) string { return "Configuration file error: " + l }},
	{match: func(l string) bool { return strings.Contains(l, "git") && containsAny(l, "not found", "failed") }, message: func(l string) string { return "Git-related error: " + l }},
	{match: func(l string) bool { return containsAny(l, "Analysis failed", "analysis error") }, message: func(l string) string { return "Analysis error: " + l }},
	{match: func(l string) bool { return containsAny(l, "500", "502", "503", "504") }, message: func(l string) string { return "Server error: " + l }},
	{match: func(l string) bool { return containsAll(l, "java.lang.IllegalStateException", "Tokens of file", ".bsl") }, staticMsg: "BSL tokenization error: File contains invalid token sequence - check file encoding and syntax"},
	{match: func(l string) bool { return strings.Contains(l, "com.github._1c_syntax.bsl") }, message: func(l string) string { return "BSL plugin error: " + l }},
	{match: func(l string) bool { return strings.Contains(l, ".bsl") && containsAny(l, "encoding", "charset") }, message: func(l string) string { return "BSL encoding error: " + l }},
	{match: func(l string) bool { return strings.Contains(l, ".bsl") && strings.Contains(l, "syntax") }, message: func(l string) string { return "BSL syntax error: " + l }},
}

// checkPrefixErrors checks for ERROR: and WARN: prefixed lines.
func checkPrefixErrors(line string) []string {
	var errs []string
	if strings.HasPrefix(line, "ERROR:") {
		if msg := strings.TrimSpace(strings.TrimPrefix(line, "ERROR:")); msg != "" {
			errs = append(errs, "Scanner error: "+msg)
		}
	}
	if strings.HasPrefix(line, "WARN:") {
		if msg := strings.TrimSpace(strings.TrimPrefix(line, "WARN:")); msg != "" {
			if containsAny(msg, "fail", "error", "invalid") {
				errs = append(errs, "Scanner warning: "+msg)
			}
		}
	}
	return errs
}

// check1CErrors checks for 1C platform-related errors.
func check1CErrors(line string) []string {
	if containsAny(line, "1C", "1c") && containsAny(line, "error", "ERROR", "failed") {
		return []string{"1C platform error: " + line}
	}
	return nil
}

// analyzeErrorOutput analyzes scanner output to extract specific error information.
func (s *SonarScannerEntity) analyzeErrorOutput(output string) []string {
	var errs []string
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		errs = append(errs, matchErrorPatterns(line)...)
		errs = append(errs, checkPrefixErrors(line)...)
		errs = append(errs, check1CErrors(line)...)
	}

	return errs
}

// matchErrorPatterns checks a line against all known error patterns.
func matchErrorPatterns(line string) []string {
	var errs []string
	for _, p := range errorPatterns {
		if p.match(line) {
			if p.message != nil {
				errs = append(errs, p.message(line))
			} else {
				errs = append(errs, p.staticMsg)
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
