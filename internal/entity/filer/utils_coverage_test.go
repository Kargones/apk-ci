package filer

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestPathUtils_NormalizePath_EdgeCases тестирует граничные случаи NormalizePath
func TestPathUtils_NormalizePath_EdgeCases(t *testing.T) {
	pathUtils := NewPathUtils()

	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "Empty path",
			input:    "",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Root path",
			input:    "/",
			expected: "/",
			wantErr:  false,
		},
		{
			name:     "Path with multiple slashes",
			input:    "//path///to////file",
			expected: "/path/to/file",
			wantErr:  false,
		},
		{
			name:     "Path with current directory references",
			input:    "/path/./to/./file",
			expected: "/path/to/file",
			wantErr:  false,
		},
		{
			name:     "Path with parent directory references",
			input:    "/path/to/../file",
			expected: "/path/file",
			wantErr:  false,
		},
		{
			name:     "Complex path with mixed references",
			input:    "/path/./to/../from/./file",
			expected: "/path/from/file",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := pathUtils.NormalizePath(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for input %q, but got none", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for input %q: %v", tt.input, err)
				return
			}

			if result != tt.expected {
				t.Errorf("For input %q, expected %q, got %q", tt.input, tt.expected, result)
			}
		})
	}
}

// TestPathUtils_EnsureDir_EdgeCases тестирует граничные случаи EnsureDir
func TestPathUtils_EnsureDir_EdgeCases(t *testing.T) {
	pathUtils := NewPathUtils()

	// Тест с пустым путем
	err := pathUtils.EnsureDir("")
	if err == nil {
		t.Error("Expected error for empty path")
	}

	// Тест с корневой директорией (должна уже существовать)
	err = pathUtils.EnsureDir("/")
	if err != nil {
		t.Errorf("Unexpected error for root directory: %v", err)
	}

	// Тест с очень длинным путем
	tempDir := t.TempDir()
	longPath := tempDir
	for i := 0; i < 10; i++ {
		longPath = filepath.Join(longPath, "very_long_directory_name_that_should_still_work")
	}

	err = pathUtils.EnsureDir(longPath)
	if err != nil {
		t.Errorf("Unexpected error for long path: %v", err)
	}

	// Проверяем, что директория создана
	if _, statErr := os.Stat(longPath); os.IsNotExist(statErr) {
		t.Error("Long path directory was not created")
	}

	// Тест с путем, содержащим специальные символы (если они разрешены)
	specialPath := filepath.Join(tempDir, "dir-with_special.chars")
	err = pathUtils.EnsureDir(specialPath)
	if err != nil {
		t.Errorf("Unexpected error for path with special characters: %v", err)
	}
}

// TestPathUtils_IsSubPath_EdgeCases тестирует граничные случаи IsSubPath
func TestPathUtils_IsSubPath_EdgeCases(t *testing.T) {
	pathUtils := NewPathUtils()

	tests := []struct {
		name       string
		parentPath string
		childPath  string
		expected   bool
		wantErr    bool
	}{
		{
			name:       "Empty parent path",
			parentPath: "",
			childPath:  "/some/path",
			expected:   false,
			wantErr:    true,
		},
		{
			name:       "Empty child path",
			parentPath: "/some/path",
			childPath:  "",
			expected:   false,
			wantErr:    true,
		},
		{
			name:       "Both paths empty",
			parentPath: "",
			childPath:  "",
			expected:   false,
			wantErr:    true,
		},
		{
			name:       "Root as parent",
			parentPath: "/",
			childPath:  "/any/path",
			expected:   true,
			wantErr:    false,
		},
		{
			name:       "Child is root",
			parentPath: "/some/path",
			childPath:  "/",
			expected:   false,
			wantErr:    false,
		},
		{
			name:       "Identical paths",
			parentPath: "/same/path",
			childPath:  "/same/path",
			expected:   true,
			wantErr:    false,
		},
		{
			name:       "Similar but not subpath",
			parentPath: "/path/to/dir",
			childPath:  "/path/to/directory",
			expected:   false,
			wantErr:    false,
		},
		{
			name:       "Valid subpath",
			parentPath: "/path/to",
			childPath:  "/path/to/subdir",
			expected:   true,
			wantErr:    false,
		},
		{
			name:       "Deep subpath",
			parentPath: "/root",
			childPath:  "/root/level1/level2/level3/file",
			expected:   true,
			wantErr:    false,
		},
		{
			name:       "Path with dots",
			parentPath: "/path/to/../to",
			childPath:  "/path/to/subdir",
			expected:   true,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := pathUtils.IsSubPath(tt.parentPath, tt.childPath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for parent=%q, child=%q, but got none", tt.parentPath, tt.childPath)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for parent=%q, child=%q: %v", tt.parentPath, tt.childPath, err)
				return
			}

			if result != tt.expected {
				t.Errorf("For parent=%q, child=%q, expected %v, got %v", tt.parentPath, tt.childPath, tt.expected, result)
			}
		})
	}
}

// TestGetOptimalTempDir_Coverage тестирует дополнительные случаи GetOptimalTempDir
func TestGetOptimalTempDir_Coverage(t *testing.T) {
	// Сохраняем оригинальное значение GOOS для восстановления
	originalGOOS := runtime.GOOS
	defer func() {
		// Восстанавливаем оригинальное значение (хотя это не изменит runtime.GOOS)
		_ = originalGOOS
	}()

	// Тестируем функцию на текущей ОС
	tempDir := GetOptimalTempDir()

	// Проверяем, что возвращается непустая строка
	if tempDir == "" {
		t.Error("GetOptimalTempDir should not return empty string")
	}

	// Проверяем, что возвращенная директория существует или может быть создана
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		// Пытаемся создать тестовую поддиректорию
		testPath := filepath.Join(tempDir, "test_coverage")
		if err := os.MkdirAll(testPath, 0755); err != nil {
			t.Errorf("Cannot create test directory in %s: %v", tempDir, err)
		} else {
			if err := os.RemoveAll(testPath); err != nil {
				t.Logf("Failed to remove test path: %v", err)
			}
		}
	}

	// Проверяем логику выбора
	if runtime.GOOS == "linux" {
		// На Linux проверяем, что выбор соответствует доступности RAM-диска
		if IsRAMDiskAvailable() {
			if tempDir != "/dev/shm" {
				t.Errorf("Expected /dev/shm when RAM disk is available on Linux, got %s", tempDir)
			}
		} else {
			if tempDir != os.TempDir() {
				t.Errorf("Expected system temp dir when RAM disk is not available, got %s", tempDir)
			}
		}
	} else {
		// На других ОС должна возвращаться системная временная директория
		if tempDir != os.TempDir() {
			t.Errorf("Expected system temp dir on non-Linux OS, got %s", tempDir)
		}
	}
}

// TestIsRAMDiskAvailable_Coverage тестирует дополнительные случаи IsRAMDiskAvailable
func TestIsRAMDiskAvailable_Coverage(t *testing.T) {
	// Тестируем функцию на текущей ОС
	available := IsRAMDiskAvailable()

	// Проверяем, что функция возвращает boolean
	if available != true && available != false {
		t.Error("IsRAMDiskAvailable should return boolean value")
	}

	// Проверяем логику для разных ОС
	if runtime.GOOS == "linux" {
		// На Linux проверяем реальное состояние /dev/shm
		info, err := os.Stat("/dev/shm")
		expectedAvailable := err == nil && info.IsDir()

		if available != expectedAvailable {
			t.Errorf("Expected IsRAMDiskAvailable to return %v on Linux, got %v", expectedAvailable, available)
		}
	} else {
		// На других ОС всегда должно быть false
		if available {
			t.Error("Expected IsRAMDiskAvailable to return false on non-Linux OS")
		}
	}

	// Дополнительная проверка: если функция возвращает true, /dev/shm должна существовать
	if available {
		if info, err := os.Stat("/dev/shm"); err != nil || !info.IsDir() {
			t.Error("IsRAMDiskAvailable returned true, but /dev/shm is not accessible or not a directory")
		}
	}
}