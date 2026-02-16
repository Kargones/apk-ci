package filer

import (
	"testing"
)

// TestPathUtils_GetFileExtensionAdditional тестирует GetFileExtension с различными сценариями
func TestPathUtils_GetFileExtensionAdditional(t *testing.T) {
	utils := NewPathUtils()
	
	testCases := []struct {
		name     string
		filename string
		expected string
	}{
		{
			name:     "Simple extension",
			filename: "file.txt",
			expected: ".txt",
		},
		{
			name:     "Multiple dots",
			filename: "archive.tar.gz",
			expected: ".gz",
		},
		{
			name:     "No extension",
			filename: "filename",
			expected: "",
		},
		{
			name:     "Hidden file with extension",
			filename: ".hidden.txt",
			expected: ".txt",
		},
		{
			name:     "Hidden file without extension",
			filename: ".hidden",
			expected: "",
		},
		{
			name:     "Path with extension",
			filename: "/path/to/file.pdf",
			expected: ".pdf",
		},
		{
			name:     "Windows path",
			filename: "C:\\path\\file.doc",
			expected: ".doc",
		},
		{
			name:     "Empty string",
			filename: "",
			expected: "",
		},
		{
			name:     "Only dot",
			filename: ".",
			expected: "",
		},
		{
			name:     "Dot at end",
			filename: "file.",
			expected: "",
		},
		{
			name:     "Multiple extensions",
			filename: "file.backup.old.txt",
			expected: ".txt",
		},
		{
			name:     "Uppercase extension",
			filename: "FILE.TXT",
			expected: ".TXT",
		},
		{
			name:     "Special characters in extension",
			filename: "file.test.tar.gz",
			expected: ".gz",
		},
		{
			name:     "Underscore in filename",
			filename: "my_file_backup.log",
			expected: ".log",
		},
		{
			name:     "Hyphen in filename",
			filename: "test-file-backup.log",
			expected: ".log",
		},
		{
			name:     "Numbers in extension",
			filename: "data123.backup456",
			expected: ".backup456",
		},
		{
			name:     "Very long filename",
			filename: "this_is_a_very_long_filename_with_many_characters_and_a_txt_extension.txt",
			expected: ".txt",
		},
		{
			name:     "Single character extension",
			filename: "file.a",
			expected: ".a",
		},
		{
			name:     "Extension with special characters",
			filename: "file.t~xt",
			expected: ".t~xt",
		},
		{
			name:     "Filename with spaces",
			filename: "my document.pdf",
			expected: ".pdf",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := utils.GetFileExtension(tc.filename)
			if result != tc.expected {
				t.Errorf("GetFileExtension(%q) = %q, expected %q", tc.filename, result, tc.expected)
			}
		})
	}
}

// TestPathUtils_GetBaseNameAdditional тестирует GetBaseName с различными сценариями
func TestPathUtils_GetBaseNameAdditional(t *testing.T) {
	utils := NewPathUtils()
	
	testCases := []struct {
		name     string
		filename string
		expected string
	}{
		{
			name:     "Simple filename",
			filename: "file.txt",
			expected: "file",
		},
		{
			name:     "Multiple dots",
			filename: "archive.tar.gz",
			expected: "archive.tar",
		},
		{
			name:     "No extension",
			filename: "filename",
			expected: "filename",
		},
		{
			name:     "Hidden file with extension",
			filename: ".hidden.txt",
			expected: ".hidden",
		},
		{
			name:     "Hidden file without extension",
			filename: ".hidden",
			expected: ".hidden",
		},
		{
			name:     "Path with extension",
			filename: "/path/to/file.pdf",
			expected: "file",
		},
		{
			name:     "Windows path",
			filename: "C:\\path\\file.doc",
			expected: "file",
		},
		{
			name:     "Empty string",
			filename: "",
			expected: "",
		},
		{
			name:     "Only dot",
			filename: ".",
			expected: ".",
		},
		{
			name:     "Dot at end",
			filename: "file.",
			expected: "file",
		},
		{
			name:     "Multiple extensions",
			filename: "file.backup.old.txt",
			expected: "file.backup.old",
		},
		{
			name:     "Uppercase extension",
			filename: "FILE.TXT",
			expected: "FILE",
		},
		{
			name:     "Special characters in extension",
			filename: "file.test.tar.gz",
			expected: "file.test.tar",
		},
		{
			name:     "Underscore in filename",
			filename: "my_file_backup.log",
			expected: "my_file_backup",
		},
		{
			name:     "Hyphen in filename",
			filename: "test-file-backup.log",
			expected: "test-file-backup",
		},
		{
			name:     "Numbers in extension",
			filename: "data123.backup456",
			expected: "data123",
		},
		{
			name:     "Very long filename",
			filename: "this_is_a_very_long_filename_with_many_characters_and_a_txt_extension.txt",
			expected: "this_is_a_very_long_filename_with_many_characters_and_a_txt_extension",
		},
		{
			name:     "Single character extension",
			filename: "file.a",
			expected: "file",
		},
		{
			name:     "Extension with special characters",
			filename: "file.t~xt",
			expected: "file",
		},
		{
			name:     "Filename with spaces",
			filename: "my document.pdf",
			expected: "my document",
		},
		{
			name:     "Complex path",
			filename: "/very/long/path/to/some/file.extension",
			expected: "file",
		},
		{
			name:     "Filename with multiple dots at end",
			filename: "file...",
			expected: "file..",
		},
		{
			name:     "Filename with multiple consecutive dots",
			filename: "file...txt",
			expected: "file..",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := utils.GetBaseName(tc.filename)
			if result != tc.expected {
				t.Errorf("GetBaseName(%q) = %q, expected %q", tc.filename, result, tc.expected)
			}
		})
	}
}

// TestPathUtils_EdgeCases тестирует edge cases в PathUtils
func TestPathUtils_EdgeCases(t *testing.T) {
	utils := NewPathUtils()
	
	// Тест NormalizePath с пустым путем
	_, err := utils.NormalizePath("")
	if err == nil {
		t.Error("NormalizePath should fail with empty path")
	}
	
	// Тест ValidatePath с пустым путем
	err = utils.ValidatePath("")
	if err == nil {
		t.Error("ValidatePath should fail with empty path")
	}
	
	// Тест JoinPath с пустым базовым путем
	_, err = utils.JoinPath("", "component")
	if err == nil {
		t.Error("JoinPath should fail with empty base path")
	}
	
	// Тест JoinPath с недопустимым компонентом
	_, err = utils.JoinPath("/tmp", "../../../etc/passwd")
	if err == nil {
		t.Error("JoinPath should fail with invalid component")
	}
	
	// Тест EnsureDir с пустым путем
	err = utils.EnsureDir("")
	if err == nil {
		t.Error("EnsureDir should fail with empty path")
	}
	
	// Тест IsSubPath с пустым родительским путем
	_, err = utils.IsSubPath("", "/tmp")
	if err == nil {
		t.Error("IsSubPath should fail with empty parent path")
	}
	
	// Тест IsSubPath с пустым дочерним путем
	_, err = utils.IsSubPath("/tmp", "")
	if err == nil {
		t.Error("IsSubPath should fail with empty child path")
	}
	
	// Тест GetFileExtension с очень длинным именем файла
	longFilename := ""
	for i := 0; i < 1000; i++ {
		longFilename += "a"
	}
	longFilename += ".txt"
	
	ext := utils.GetFileExtension(longFilename)
	if ext != ".txt" {
		t.Errorf("GetFileExtension with long filename: got %q, expected %q", ext, ".txt")
	}
	
	// Тест GetBaseName с очень длинным именем файла
	baseName := utils.GetBaseName(longFilename)
	expectedBase := longFilename[:len(longFilename)-4] // Убираем ".txt"
	if baseName != expectedBase {
		t.Errorf("GetBaseName with long filename: got length %d, expected length %d", len(baseName), len(expectedBase))
	}
}