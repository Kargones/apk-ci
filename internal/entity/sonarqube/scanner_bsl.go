package sonarqube

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// FixBSLTokenizationIssues attempts to fix common BSL file tokenization issues.
func (s *SonarScannerEntity) FixBSLTokenizationIssues(filePath string) error {
	s.logger.Info("Attempting to fix BSL tokenization issues", "file", filePath)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("BSL file not found: %s", filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read BSL file %s: %w", filePath, err)
	}

	originalContent := string(content)
	fixedContent := originalContent
	fixesApplied := 0

	if strings.Contains(fixedContent, "\r\n") {
		fixedContent = strings.ReplaceAll(fixedContent, "\r\n", "\n")
		fixesApplied++
	}

	lines := strings.Split(fixedContent, "\n")
	for i, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		if trimmed != line {
			lines[i] = trimmed
			fixesApplied++
		}
	}
	fixedContent = strings.Join(lines, "\n")

	fixedContent = strings.TrimRight(fixedContent, "\n") + "\n"

	if strings.Contains(fixedContent, "\u00A0") {
		fixedContent = strings.ReplaceAll(fixedContent, "\u00A0", " ")
		fixesApplied++
	}

	if strings.HasPrefix(fixedContent, "\uFEFF") {
		fixedContent = strings.TrimPrefix(fixedContent, "\uFEFF")
		fixesApplied++
	}

	fixedContent = s.fixBSLSyntaxIssues(fixedContent, filePath)

	if fixedContent != originalContent {
		backupPath := filePath + ".backup"
		if bErr := os.WriteFile(backupPath, content, 0644); bErr != nil {
			s.logger.Warn("Failed to create backup", "file", filePath, "backup", backupPath, "error", bErr)
		}

		if wErr := os.WriteFile(filePath, []byte(fixedContent), 0644); wErr != nil {
			return fmt.Errorf("failed to write fixed BSL file %s: %w", filePath, wErr)
		}

		s.logger.Info("Applied BSL fixes", "file", filePath, "fixesApplied", fixesApplied)
	}

	return nil
}

// fixBSLSyntaxIssues fixes common BSL syntax issues that cause tokenization problems.
func (s *SonarScannerEntity) fixBSLSyntaxIssues(content, filePath string) string {
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		originalLine := line

		validChars := regexp.MustCompile(`[^\p{L}\p{N}\s\(\)\[\]\{\}\.,;:=\+\-\*\/\\\|&<>!@#\$%\^~` + "`" + `"'_]`)
		line = validChars.ReplaceAllString(line, "")

		if strings.Contains(line, "//") {
			line = regexp.MustCompile(`[^\s]/\/`).ReplaceAllString(line, " //")
		}

		if strings.Count(line, `"`)%2 != 0 {
			line = line + `"`
		}

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
		}
	}

	return strings.Join(lines, "\n")
}

// FindAndValidateBSLFiles searches for BSL files in the working directory and categorizes them.
func (s *SonarScannerEntity) FindAndValidateBSLFiles() ([]string, []string, error) {
	var validFiles []string
	var problematicFiles []string

	if s.workDir == "" {
		return validFiles, problematicFiles, fmt.Errorf("working directory not set")
	}

	err := filepath.Walk(s.workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if !info.IsDir() && (strings.HasSuffix(strings.ToLower(path), ".bsl") || strings.HasSuffix(strings.ToLower(path), ".os")) {
			if s.validateBSLFile(path) {
				validFiles = append(validFiles, path)
			} else {
				problematicFiles = append(problematicFiles, path)
			}
		}

		return nil
	})

	if err != nil {
		return validFiles, problematicFiles, fmt.Errorf("failed to walk directory: %w", err)
	}

	return validFiles, problematicFiles, nil
}

// validateBSLFile performs basic validation of a BSL file.
func (s *SonarScannerEntity) validateBSLFile(filePath string) bool {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	contentStr := string(content)

	if strings.Contains(contentStr, "\r\n") && strings.Contains(contentStr, "\n") {
		return false
	}
	if strings.HasPrefix(contentStr, "\uFEFF") {
		return false
	}
	if strings.Contains(contentStr, "\u00A0") {
		return false
	}

	lines := strings.Split(contentStr, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		if strings.Count(line, `"`)%2 != 0 {
			return false
		}
	}

	validChars := regexp.MustCompile(`^[\p{L}\p{N}\s\(\)\[\]\{\}\.,;:=\+\-\*\/\\\|&<>!@#\$%\^~` + "`" + `"'_\r\n]*$`)
	if !validChars.MatchString(contentStr) {
		return false
	}

	return true
}

// preProcessBSLFiles finds and fixes all problematic BSL files before scanning.
func (s *SonarScannerEntity) preProcessBSLFiles() error {
	_, problematicFiles, err := s.FindAndValidateBSLFiles()
	if err != nil {
		return fmt.Errorf("failed to find BSL files: %w", err)
	}

	for _, filePath := range problematicFiles {
		if fixErr := s.FixBSLTokenizationIssues(filePath); fixErr != nil {
			s.logger.Error("Failed to fix BSL file during preprocessing", "file", filePath, "error", fixErr)
		}
	}

	return nil
}

// AddFileToExclusions adds a file to the exclusion list.
func (s *SonarScannerEntity) AddFileToExclusions(filePath string) {
	for _, excluded := range s.excludedFiles {
		if excluded == filePath {
			return
		}
	}

	s.excludedFiles = append(s.excludedFiles, filePath)
	s.updateExclusionsProperty()
}

// AddFilesToExclusions adds multiple files to the exclusion list.
func (s *SonarScannerEntity) AddFilesToExclusions(filePaths []string) {
	for _, filePath := range filePaths {
		s.AddFileToExclusions(filePath)
	}
}

// updateExclusionsProperty updates the sonar.exclusions property.
func (s *SonarScannerEntity) updateExclusionsProperty() {
	if len(s.excludedFiles) == 0 {
		return
	}

	existingExclusions := s.GetProperty("sonar.exclusions")
	var exclusionPatterns []string

	if existingExclusions != "" {
		exclusionPatterns = append(exclusionPatterns, existingExclusions)
	}

	for _, filePath := range s.excludedFiles {
		pattern := fmt.Sprintf("**/%s", filepath.Base(filePath))
		exclusionPatterns = append(exclusionPatterns, pattern)

		if !filepath.IsAbs(filePath) {
			exclusionPatterns = append(exclusionPatterns, filePath)
		}
	}

	s.SetProperty("sonar.exclusions", strings.Join(exclusionPatterns, ","))
}

// GetExcludedFiles returns the list of currently excluded files.
func (s *SonarScannerEntity) GetExcludedFiles() []string {
	return append([]string(nil), s.excludedFiles...)
}

// ClearExclusions clears all excluded files.
func (s *SonarScannerEntity) ClearExclusions() {
	s.excludedFiles = make([]string, 0)
	s.SetProperty("sonar.exclusions", "")
}

// ExtractProblematicBSLFiles extracts file paths from BSL-related error messages.
func (s *SonarScannerEntity) ExtractProblematicBSLFiles(errorOutput string) []string {
	var problematicFiles []string

	bslFilePattern := regexp.MustCompile(`(?i)([^\s"']+\.bsl)`)
	matches := bslFilePattern.FindAllString(errorOutput, -1)

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

	fileMap := make(map[string]bool)
	for _, match := range matches {
		cleanPath := strings.TrimSpace(match)
		cleanPath = strings.Trim(cleanPath, `"'`)

		if len(cleanPath) < 5 {
			continue
		}

		if filepath.IsAbs(cleanPath) {
			if rel, err := filepath.Rel(s.workDir, cleanPath); err == nil {
				cleanPath = rel
			}
		}

		if !fileMap[cleanPath] {
			fileMap[cleanPath] = true
			problematicFiles = append(problematicFiles, cleanPath)
		}
	}

	return problematicFiles
}

// SuggestBSLExclusions generates exclusion patterns for problematic BSL files.
func (s *SonarScannerEntity) SuggestBSLExclusions(errorOutput string) []string {
	var suggestions []string

	bslFilePattern := regexp.MustCompile(`([^/\s]+\.bsl)`)
	matches := bslFilePattern.FindAllString(errorOutput, -1)

	for _, match := range matches {
		suggestions = append(suggestions, fmt.Sprintf("sonar.exclusions=**/%s", match))

		if strings.Contains(strings.ToLower(match), "commonmodule") || strings.Contains(match, "CommonModules") {
			suggestions = append(suggestions, "sonar.exclusions=**/CommonModules/**/*.bsl")
		}
	}

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
