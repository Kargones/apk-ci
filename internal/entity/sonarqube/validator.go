// Package sonarqube provides configuration validation for SonarScanner.
// This package contains validation logic for scanner configuration,
// properties validation, and configuration rules enforcement.
package sonarqube

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
)

// ExtendedValidationError represents a configuration validation error with error code.
type ExtendedValidationError struct {
	*ValidationError
	Code string `json:"code"`
}

// Error implements the error interface for ExtendedValidationError.
func (e *ExtendedValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s (value: '%s', code: %s)",
		e.Field, e.Message, e.Value, e.Code)
}

// ValidationResult holds the result of configuration validation.
type ValidationResult struct {
	Valid    bool                       `json:"valid"`
	Errors   []*ExtendedValidationError `json:"errors"`
	Warnings []*ExtendedValidationError `json:"warnings"`
}

// AddError adds a validation error to the result.
func (r *ValidationResult) AddError(field, value, message, code string) {
	r.Valid = false
	r.Errors = append(r.Errors, &ExtendedValidationError{
		ValidationError: &ValidationError{
			Field:   field,
			Value:   value,
			Message: message,
		},
		Code: code,
	})
}

// AddWarning adds a validation warning to the result.
func (r *ValidationResult) AddWarning(field, value, message, code string) {
	r.Warnings = append(r.Warnings, &ExtendedValidationError{
		ValidationError: &ValidationError{
			Field:   field,
			Value:   value,
			Message: message,
		},
		Code: code,
	})
}

// HasErrors returns true if there are validation errors.
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// HasWarnings returns true if there are validation warnings.
func (r *ValidationResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// ConfigValidator provides comprehensive configuration validation.
type ConfigValidator struct {
	// validProjectKeyPattern defines the valid pattern for project keys
	validProjectKeyPattern *regexp.Regexp

	// validVersionPattern defines the valid pattern for versions
	validVersionPattern *regexp.Regexp

	// requiredProperties defines the list of required SonarQube properties
	requiredProperties []string

	// validPropertyPatterns defines validation patterns for specific properties
	validPropertyPatterns map[string]*regexp.Regexp
}

// NewConfigValidator creates a new configuration validator.
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		validProjectKeyPattern: regexp.MustCompile(`^[a-zA-Z0-9_\-.:]+$`),
		validVersionPattern:    regexp.MustCompile(`^\d+\.\d+(\.\d+)?(-[a-zA-Z0-9\-]+)?$`),
		requiredProperties: []string{
			"sonar.host.url",
			"sonar.projectKey",
		},
		validPropertyPatterns: map[string]*regexp.Regexp{
			"sonar.projectKey":     regexp.MustCompile(`^[a-zA-Z0-9_\-.:]+$`),
			"sonar.projectVersion": regexp.MustCompile(`^[a-zA-Z0-9_\-.:]+$`),
			"sonar.language":       regexp.MustCompile(`^[a-z]+$`),
			"sonar.sourceEncoding": regexp.MustCompile(`^[a-zA-Z0-9\-]+$`),
		},
	}
}

// ValidateConfig performs comprehensive validation of scanner configuration.
//
// Parameters:
//   - cfg: scanner configuration to validate
//
// Returns:
//   - *ValidationResult: validation result with errors and warnings
func (v *ConfigValidator) ValidateConfig(cfg *config.ScannerConfig) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   make([]*ExtendedValidationError, 0),
		Warnings: make([]*ExtendedValidationError, 0),
	}

	if cfg == nil {
		result.AddError("config", "nil", "Configuration cannot be nil", "CFG_NULL")
		return result
	}

	// Validate basic configuration fields
	v.validateBasicFields(cfg, result)

	// Validate URLs
	v.validateURLs(cfg, result)

	// Validate paths
	v.validatePaths(cfg, result)

	// Validate timeouts
	v.validateTimeouts(cfg, result)

	// Validate properties
	v.validateProperties(cfg, result)

	// Validate version
	v.validateVersion(cfg, result)

	return result
}

// validateBasicFields validates basic configuration fields.
func (v *ConfigValidator) validateBasicFields(cfg *config.ScannerConfig, result *ValidationResult) {
	// Validate ScannerURL
	if cfg.ScannerURL == "" {
		result.AddError("ScannerURL", "", "Scanner URL is required", "URL_REQUIRED")
	}

	// Validate ScannerVersion
	if cfg.ScannerVersion == "" {
		result.AddError("ScannerVersion", "", "Scanner version is required", "VERSION_REQUIRED")
	}
}

// validateURLs validates URL fields in the configuration.
func (v *ConfigValidator) validateURLs(cfg *config.ScannerConfig, result *ValidationResult) {
	// Validate ScannerURL format
	if cfg.ScannerURL != "" {
		if _, err := url.Parse(cfg.ScannerURL); err != nil {
			result.AddError("ScannerURL", cfg.ScannerURL,
				fmt.Sprintf("Invalid URL format: %v", err), "URL_INVALID")
		} else {
			// Check if URL is reachable (warning only)
			parsedURL, _ := url.Parse(cfg.ScannerURL)
			if parsedURL.Scheme == "" {
				result.AddWarning("ScannerURL", cfg.ScannerURL,
					"URL scheme not specified, assuming HTTP", "URL_NO_SCHEME")
			}
			if parsedURL.Host == "" {
				result.AddError("ScannerURL", cfg.ScannerURL,
					"URL host is required", "URL_NO_HOST")
			}
		}
	}

	// Validate sonar.host.url property if present
	if cfg.Properties != nil {
		if hostURL, exists := cfg.Properties["sonar.host.url"]; exists {
			if _, err := url.Parse(hostURL); err != nil {
				result.AddError("Properties.sonar.host.url", hostURL,
					fmt.Sprintf("Invalid SonarQube host URL: %v", err), "SONAR_URL_INVALID")
			}
		}
	}
}

// validatePaths validates path fields in the configuration.
func (v *ConfigValidator) validatePaths(cfg *config.ScannerConfig, result *ValidationResult) {
	// Validate WorkDir
	if cfg.WorkDir != "" {
		if !filepath.IsAbs(cfg.WorkDir) {
			result.AddWarning("WorkDir", cfg.WorkDir,
				"Working directory should be an absolute path", "PATH_RELATIVE")
		}

		if _, err := os.Stat(cfg.WorkDir); os.IsNotExist(err) {
			result.AddError("WorkDir", cfg.WorkDir,
				"Working directory does not exist", "PATH_NOT_EXISTS")
		} else if err != nil {
			result.AddWarning("WorkDir", cfg.WorkDir,
				fmt.Sprintf("Cannot access working directory: %v", err), "PATH_ACCESS_ERROR")
		}
	}

	// Validate TempDir
	if cfg.TempDir != "" {
		if !filepath.IsAbs(cfg.TempDir) {
			result.AddWarning("TempDir", cfg.TempDir,
				"Temporary directory should be an absolute path", "PATH_RELATIVE")
		}

		if _, err := os.Stat(cfg.TempDir); os.IsNotExist(err) {
			result.AddError("TempDir", cfg.TempDir,
				"Temporary directory does not exist", "PATH_NOT_EXISTS")
		} else if err != nil {
			result.AddWarning("TempDir", cfg.TempDir,
				fmt.Sprintf("Cannot access temporary directory: %v", err), "PATH_ACCESS_ERROR")
		}
	}

	// Validate source directories in properties
	if cfg.Properties != nil {
		if sources, exists := cfg.Properties["sonar.sources"]; exists {
			v.validateSourcePaths(sources, result)
		}
		if tests, exists := cfg.Properties["sonar.tests"]; exists {
			v.validateSourcePaths(tests, result)
		}
	}
}

// validateSourcePaths validates source and test paths.
func (v *ConfigValidator) validateSourcePaths(paths string, result *ValidationResult) {
	pathList := strings.Split(paths, ",")
	for _, path := range pathList {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			result.AddWarning("Properties.sonar.sources", path,
				"Source path does not exist", "SOURCE_PATH_NOT_EXISTS")
		}
	}
}

// validateTimeouts validates timeout configurations.
func (v *ConfigValidator) validateTimeouts(cfg *config.ScannerConfig, result *ValidationResult) {
	// Validate main timeout
	switch {
	case cfg.Timeout <= 0:
		result.AddWarning("Timeout", cfg.Timeout.String(),
			"Timeout should be positive, using default", "TIMEOUT_INVALID")
	case cfg.Timeout < time.Minute:
		result.AddWarning("Timeout", cfg.Timeout.String(),
			"Timeout is very short, may cause scan failures", "TIMEOUT_TOO_SHORT")
	case cfg.Timeout > 24*time.Hour:
		result.AddWarning("Timeout", cfg.Timeout.String(),
			"Timeout is very long, consider reducing it", "TIMEOUT_TOO_LONG")
	}
}

// validateProperties validates SonarQube properties.
func (v *ConfigValidator) validateProperties(cfg *config.ScannerConfig, result *ValidationResult) {
	if cfg.Properties == nil {
		result.AddWarning("Properties", "nil",
			"Properties map is nil, will be initialized", "PROPS_NULL")
		return
	}

	// Check required properties
	for _, requiredProp := range v.requiredProperties {
		if _, exists := cfg.Properties[requiredProp]; !exists {
			result.AddError("Properties", requiredProp,
				fmt.Sprintf("Required property '%s' is missing", requiredProp), "PROP_REQUIRED")
		}
	}

	// Validate property formats
	for key, value := range cfg.Properties {
		v.validatePropertyFormat(key, value, result)
	}

	// Validate specific property combinations
	v.validatePropertyCombinations(cfg.Properties, result)
}

// validatePropertyFormat validates the format of individual properties.
func (v *ConfigValidator) validatePropertyFormat(key, value string, result *ValidationResult) {
	// Check if property has a validation pattern
	if pattern, exists := v.validPropertyPatterns[key]; exists {
		if !pattern.MatchString(value) {
			result.AddError(fmt.Sprintf("Properties.%s", key), value,
				"Property value does not match required pattern", "PROP_FORMAT_INVALID")
		}
	}

	// Validate specific properties
	switch key {
	case "sonar.projectKey":
		if !v.validProjectKeyPattern.MatchString(value) {
			result.AddError(fmt.Sprintf("Properties.%s", key), value,
				"Project key contains invalid characters", "PROJECT_KEY_INVALID")
		}
		if len(value) > 400 {
			result.AddError(fmt.Sprintf("Properties.%s", key), value,
				"Project key is too long (max 400 characters)", "PROJECT_KEY_TOO_LONG")
		}

	case "sonar.projectVersion":
		if len(value) > 100 {
			result.AddWarning(fmt.Sprintf("Properties.%s", key), value,
				"Project version is very long", "PROJECT_VERSION_LONG")
		}

	case "sonar.java.binaries":
		v.validateJavaBinaries(value, result)

	case "sonar.exclusions", "sonar.test.exclusions":
		v.validateExclusionPatterns(key, value, result)

	case "sonar.coverage.exclusions":
		v.validateExclusionPatterns(key, value, result)

	case "sonar.cpd.exclusions":
		v.validateExclusionPatterns(key, value, result)
	}
}

// validateJavaBinaries validates Java binaries paths.
func (v *ConfigValidator) validateJavaBinaries(value string, result *ValidationResult) {
	paths := strings.Split(value, ",")
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			result.AddWarning("Properties.sonar.java.binaries", path,
				"Java binaries path does not exist", "JAVA_BINARIES_NOT_EXISTS")
		}
	}
}

// validateExclusionPatterns validates exclusion patterns.
func (v *ConfigValidator) validateExclusionPatterns(key, value string, result *ValidationResult) {
	patterns := strings.Split(value, ",")
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		// Basic pattern validation
		if strings.Contains(pattern, "**/**") {
			result.AddWarning(fmt.Sprintf("Properties.%s", key), pattern,
				"Redundant pattern detected", "PATTERN_REDUNDANT")
		}
	}
}

// validatePropertyCombinations validates combinations of properties.
func (v *ConfigValidator) validatePropertyCombinations(properties map[string]string, result *ValidationResult) {
	// Check if both sonar.sources and sonar.projectBaseDir are set
	if sources, hasSources := properties["sonar.sources"]; hasSources {
		if baseDir, hasBaseDir := properties["sonar.projectBaseDir"]; hasBaseDir {
			// Validate that sources are relative to base directory
			v.validateSourcesRelativeToBaseDir(sources, baseDir, result)
		}
	}

	// Check for conflicting language settings
	if lang, hasLang := properties["sonar.language"]; hasLang {
		if sources, hasSources := properties["sonar.sources"]; hasSources {
			v.validateLanguageSourcesConsistency(lang, sources, result)
		}
	}

	// Validate authentication properties
	v.validateAuthenticationProperties(properties, result)
}

// validateSourcesRelativeToBaseDir validates that sources are relative to base directory.
func (v *ConfigValidator) validateSourcesRelativeToBaseDir(sources, _ string, result *ValidationResult) {
	sourcePaths := strings.Split(sources, ",")
	// Preallocate slice for efficiency
	validPaths := make([]string, 0, len(sourcePaths))
	for _, sourcePath := range sourcePaths {
		sourcePath = strings.TrimSpace(sourcePath)
		if sourcePath == "" {
			continue
		}
		validPaths = append(validPaths, sourcePath)
	}

	for _, sourcePath := range validPaths {
		if filepath.IsAbs(sourcePath) {
			result.AddWarning("Properties.sonar.sources", sourcePath,
				"Absolute source path may not work correctly with projectBaseDir", "SOURCE_ABSOLUTE_WITH_BASEDIR")
		}
	}
}

// validateLanguageSourcesConsistency validates consistency between language and sources.
func (v *ConfigValidator) validateLanguageSourcesConsistency(language, sources string, result *ValidationResult) {
	// This is a simplified validation - in practice, you might want more sophisticated logic
	if language == "java" {
		if !strings.Contains(sources, ".java") && !strings.Contains(sources, "src/") {
			result.AddWarning("Properties.sonar.language", language,
				"Java language specified but sources don't seem to contain Java files", "LANG_SOURCE_MISMATCH")
		}
	}
}

// validateAuthenticationProperties validates authentication-related properties.
func (v *ConfigValidator) validateAuthenticationProperties(properties map[string]string, result *ValidationResult) {
	hasToken := false
	hasLogin := false

	if token, exists := properties["sonar.token"]; exists && token != "" {
		hasToken = true
		if len(token) < 40 {
			result.AddWarning("Properties.sonar.token", "***",
				"Token seems too short, verify it's correct", "TOKEN_TOO_SHORT")
		}
	}

	if login, exists := properties["sonar.login"]; exists && login != "" {
		hasLogin = true
	}

	if hasToken && hasLogin {
		result.AddWarning("Properties.authentication", "both",
			"Both token and login are specified, token will take precedence", "AUTH_BOTH_SPECIFIED")
	}

	if !hasToken && !hasLogin {
		result.AddWarning("Properties.authentication", "none",
			"No authentication specified, may fail for private SonarQube instances", "AUTH_NONE")
	}
}

// validateVersion validates the scanner version format.
func (v *ConfigValidator) validateVersion(cfg *config.ScannerConfig, result *ValidationResult) {
	if cfg.ScannerVersion != "" {
		if !v.validVersionPattern.MatchString(cfg.ScannerVersion) {
			result.AddWarning("ScannerVersion", cfg.ScannerVersion,
				"Version format may not be standard (expected: x.y.z)", "VERSION_FORMAT_UNUSUAL")
		}

		// Check for very old versions
		if v.isOldVersion(cfg.ScannerVersion) {
			result.AddWarning("ScannerVersion", cfg.ScannerVersion,
				"Scanner version is quite old, consider upgrading", "VERSION_OLD")
		}
	}
}

// isOldVersion checks if the version is considered old.
func (v *ConfigValidator) isOldVersion(version string) bool {
	// Simple version check - in practice, you might want more sophisticated logic
	parts := strings.Split(version, ".")
	if len(parts) >= 2 {
		if major, err := strconv.Atoi(parts[0]); err == nil {
			if major < 4 {
				return true
			}
			if major == 4 {
				if minor, err := strconv.Atoi(parts[1]); err == nil && minor < 6 {
					return true
				}
			}
		}
	}
	return false
}
