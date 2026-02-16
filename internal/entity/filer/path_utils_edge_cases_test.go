package filer

import (
	"testing"
)

// TestPathUtils_EdgeCasesAdditional тестирует edge cases в PathUtils дополнительно
func TestPathUtils_EdgeCasesAdditional(t *testing.T) {
	utils := NewPathUtils()
	
	// Тест NormalizePath с различными путями
	testCases := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "Empty path",
			input:       "",
			expectError: true,
		},
		{
			name:        "Current directory",
			input:       ".",
			expectError: false,
		},
		{
			name:        "Parent directory",
			input:       "..",
			expectError: false,
		},
		{
			name:        "Path with dots",
			input:       "./test/./file.txt",
			expectError: false,
		},
		{
			name:        "Path with double dots",
			input:       "test/../file.txt",
			expectError: false,
		},
		{
			name:        "Path with multiple slashes",
			input:       "test///file.txt",
			expectError: false,
		},
		{
			name:        "Path with trailing slash",
			input:       "test/",
			expectError: false,
		},
		{
			name:        "Path with leading slash",
			input:       "/test/file.txt",
			expectError: false,
		},
		{
			name:        "Windows style path",
			input:       "test\\file.txt",
			expectError: false,
		},
		{
			name:        "Complex path",
			input:       "a/b/c/../../../d/e/f",
			expectError: false,
		},
		{
			name:        "Path with unicode",
			input:       "тест/файл.txt",
			expectError: false,
		},
		{
			name:        "Path with spaces",
			input:       "test folder/file name.txt",
			expectError: false,
		},
		{
			name:        "Path with special characters",
			input:       "test/file-name_123.txt",
			expectError: false,
		},
		{
			name:        "Very long path",
			input:       "a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p/q/r/s/t/u/v/w/x/y/z/file.txt",
			expectError: false,
		},
		{
			name:        "Path with control characters",
			input:       "test/\x01file.txt",
			expectError: true,
		},
		{
			name:        "Path with null byte",
			input:       "test/file\x00.txt",
			expectError: true,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := utils.NormalizePath(tc.input)
			if tc.expectError && err == nil {
				t.Errorf("Expected error for input %q, but got none", tc.input)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error for input %q: %v", tc.input, err)
			}
		})
	}
	
	// Тест ValidatePath с различными путями
	validateTestCases := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "Valid relative path",
			input:       "test/file.txt",
			expectError: false,
		},
		{
			name:        "Path with parent directory traversal",
			input:       "../file.txt",
			expectError: true,
		},
		{
			name:        "Path with parent directory in middle",
			input:       "test/../file.txt",
			expectError: true,
		},
		{
			name:        "Absolute path",
			input:       "/tmp/file.txt",
			expectError: true,
		},
		{
			name:        "Empty path",
			input:       "",
			expectError: true,
		},
		{
			name:        "Path with null byte",
			input:       "file\x00.txt",
			expectError: true,
		},
		{
			name:        "Path with control characters",
			input:       "test/\x01file.txt",
			expectError: true,
		},
		{
			name:        "Valid path with unicode",
			input:       "тест/файл.txt",
			expectError: false,
		},
		{
			name:        "Valid path with spaces",
			input:       "test folder/file name.txt",
			expectError: false,
		},
		{
			name:        "Valid path with special characters",
			input:       "test-file_name123.txt",
			expectError: false,
		},
	}
	
	for _, tc := range validateTestCases {
		t.Run("Validate_"+tc.name, func(t *testing.T) {
			err := utils.ValidatePath(tc.input)
			if tc.expectError && err == nil {
				t.Errorf("Expected error for input %q, but got none", tc.input)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error for input %q: %v", tc.input, err)
			}
		})
	}
	
	// Тест JoinPath с различными компонентами
	joinTestCases := []struct {
		name          string
		base          string
		components    []string
		expectError   bool
		expectedError string
	}{
		{
			name:        "Valid join",
			base:        "base",
			components:  []string{"path", "file.txt"},
			expectError: false,
		},
		{
			name:        "Empty base",
			base:        "",
			components:  []string{"path"},
			expectError: true,
		},
		{
			name:        "Path traversal in component",
			base:        "base",
			components:  []string{"../evil"},
			expectError: true,
		},
		{
			name:        "Invalid characters in component",
			base:        "base",
			components:  []string{"path\x00file"},
			expectError: true,
		},
		{
			name:        "Absolute path in component",
			base:        "base",
			components:  []string{"/absolute"},
			expectError: true,
		},
		{
			name:        "Multiple components",
			base:        "base",
			components:  []string{"a", "b", "c", "d"},
			expectError: false,
		},
		{
			name:        "Unicode components",
			base:        "основа",
			components:  []string{"путь", "файл.txt"},
			expectError: false,
		},
		{
			name:        "Component with spaces",
			base:        "base",
			components:  []string{"folder with spaces", "file name.txt"},
			expectError: false,
		},
	}
	
	for _, tc := range joinTestCases {
		t.Run("Join_"+tc.name, func(t *testing.T) {
			_, err := utils.JoinPath(tc.base, tc.components...)
			if tc.expectError && err == nil {
				t.Errorf("Expected error for base %q and components %v, but got none", tc.base, tc.components)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error for base %q and components %v: %v", tc.base, tc.components, err)
			}
		})
	}
}