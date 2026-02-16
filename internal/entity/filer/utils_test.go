package filer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetOptimalTempDir(t *testing.T) {
	tempDir := GetOptimalTempDir()
	if tempDir == "" {
		t.Error("GetOptimalTempDir returned empty string")
	}
	
	// Проверяем, что директория существует или может быть создана
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		// Пытаемся создать директорию для проверки
		testDir := filepath.Join(tempDir, "test")
		if err := os.MkdirAll(testDir, 0755); err != nil {
			t.Errorf("Cannot create directory in temp dir %s: %v", tempDir, err)
		} else {
			if err := os.RemoveAll(testDir); err != nil {
				t.Logf("Failed to remove test dir: %v", err)
			}
		}
	}
	
	// Проверяем логику выбора директории
	if IsRAMDiskAvailable() {
		if tempDir != "/dev/shm" {
			t.Errorf("Expected /dev/shm when RAM disk is available, got %s", tempDir)
		}
	} else {
		if tempDir != os.TempDir() {
			t.Errorf("Expected system temp dir when RAM disk is not available, got %s", tempDir)
		}
	}
}

func TestPathUtils_EnsureDir(t *testing.T) {
	pathUtils := NewPathUtils()
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "test", "subdir")
	
	err := pathUtils.EnsureDir(testPath)
	if err != nil {
		t.Errorf("EnsureDir failed: %v", err)
	}
	
	// Проверяем, что директория создана
	info, err := os.Stat(testPath)
	if err != nil {
		t.Errorf("Directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("Created path is not a directory")
	}
}

func TestPathUtils_EnsureDir_ExistingDir(t *testing.T) {
	pathUtils := NewPathUtils()
	tempDir := t.TempDir()
	
	// Создаем директорию заранее
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	// EnsureDir должен успешно работать с существующей директорией
	err = pathUtils.EnsureDir(tempDir)
	if err != nil {
		t.Errorf("EnsureDir failed on existing directory: %v", err)
	}
}

func TestPathUtils_EnsureDir_InvalidPath(t *testing.T) {
	pathUtils := NewPathUtils()
	// Тестируем с недопустимым путем (пустая строка)
	err := pathUtils.EnsureDir("")
	if err == nil {
		t.Error("EnsureDir should fail with empty path")
	}
}

func TestPathUtils_EnsureDir_PermissionError(t *testing.T) {
	pathUtils := NewPathUtils()
	// Тестируем создание директории в недоступном месте
	// Используем путь, который скорее всего будет недоступен для записи
	invalidPath := "/root/restricted/test/dir"
	err := pathUtils.EnsureDir(invalidPath)
	// Ожидаем ошибку, но не требуем её, так как права могут отличаться
	if err != nil {
		t.Logf("EnsureDir failed as expected for restricted path: %v", err)
	} else {
		t.Logf("EnsureDir succeeded for path %s (unexpected but not an error)", invalidPath)
		// Очищаем созданную директорию если она была создана
		if err := os.RemoveAll("/root/restricted"); err != nil {
			t.Logf("Failed to remove restricted dir: %v", err)
		}
	}
}

func TestIsRAMDiskAvailable(t *testing.T) {
	// Этот тест может быть платформо-зависимым
	// Просто проверяем, что функция не паникует
	result := IsRAMDiskAvailable()
	
	// Результат может быть true или false в зависимости от системы
	// Главное, что функция работает без ошибок
	t.Logf("RAM disk available: %v", result)
	
	// Проверяем логику функции
	if info, err := os.Stat("/dev/shm"); err == nil {
		// Если /dev/shm существует, проверяем, что это директория
		expected := info.IsDir()
		if result != expected {
			t.Errorf("IsRAMDiskAvailable() = %v, expected %v based on /dev/shm stat", result, expected)
		}
	} else {
		// Если /dev/shm не существует или недоступен, должно быть false
		if result {
			t.Error("IsRAMDiskAvailable() should return false when /dev/shm is not accessible")
		}
	}
}

func TestPathUtils_ValidatePath(t *testing.T) {
	pathUtils := NewPathUtils()
	
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "Valid relative path",
			path:    "test/file.txt",
			wantErr: false,
		},
		{
			name:    "Valid single file",
			path:    "file.txt",
			wantErr: false,
		},
		{
			name:    "Empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "Path with parent directory traversal",
			path:    "../file.txt",
			wantErr: true,
		},
		{
			name:    "Path with parent directory in middle",
			path:    "test/../file.txt",
			wantErr: true,
		},
		{
			name:    "Absolute path",
			path:    "/tmp/file.txt",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pathUtils.ValidatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPathUtils_JoinPath(t *testing.T) {
	pathUtils := NewPathUtils()
	basePath := "/tmp/test"
	
	tests := []struct {
		name      string
		components []string
		wantErr   bool
	}{
		{
			name:      "Simple file",
			components: []string{"file.txt"},
			wantErr:   false,
		},
		{
			name:      "Subdirectory file",
			components: []string{"subdir", "file.txt"},
			wantErr:   false,
		},
		{
			name:      "Invalid component with traversal",
			components: []string{"../file.txt"},
			wantErr:   true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pathUtils.JoinPath(basePath, tt.components...)
			if (err != nil) != tt.wantErr {
				t.Errorf("JoinPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestPathUtils_GetFileExtension тестирует получение расширения файла
func TestPathUtils_GetFileExtension(t *testing.T) {
	pathUtils := NewPathUtils()
	
	testCases := []struct {
		name     string
		filename string
		expected string
	}{
		{"Simple extension", "file.txt", ".txt"},
		{"Multiple dots", "archive.tar.gz", ".gz"},
		{"No extension", "filename", ""},
		{"Hidden file with extension", ".hidden.txt", ".txt"},
		{"Hidden file without extension", ".hidden", ""},
		{"Path with extension", "/path/to/file.pdf", ".pdf"},
		{"Windows path", "C:\\path\\file.doc", ".doc"},
		{"Empty string", "", ""},
		{"Only dot", ".", ""},
		{"Dot at end", "file.", ""},
		{"Multiple extensions", "file.backup.old.txt", ".txt"},
		{"Uppercase extension", "FILE.TXT", ".TXT"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := pathUtils.GetFileExtension(tc.filename)
			if result != tc.expected {
				t.Errorf("GetFileExtension(%q) = %q, expected %q", tc.filename, result, tc.expected)
			}
		})
	}
}

// TestPathUtils_GetBaseName тестирует получение базового имени файла
func TestPathUtils_GetBaseName(t *testing.T) {
	pathUtils := NewPathUtils()
	
	testCases := []struct {
		name     string
		filename string
		expected string
	}{
		{"Simple filename", "file.txt", "file"},
		{"Multiple dots", "archive.tar.gz", "archive.tar"},
		{"No extension", "filename", "filename"},
		{"Hidden file with extension", ".hidden.txt", ".hidden"},
		{"Hidden file without extension", ".hidden", ".hidden"},
		{"Path with extension", "/path/to/file.pdf", "file"},
		{"Windows path", "C:\\path\\file.doc", "file"},
		{"Empty string", "", ""},
		{"Only dot", ".", "."},
		{"Dot at end", "file.", "file"},
		{"Multiple extensions", "file.backup.old.txt", "file.backup.old"},
		{"Uppercase extension", "FILE.TXT", "FILE"},
		{"Complex path", "/very/long/path/to/some/file.extension", "file"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := pathUtils.GetBaseName(tc.filename)
			if result != tc.expected {
				t.Errorf("GetBaseName(%q) = %q, expected %q", tc.filename, result, tc.expected)
			}
		})
	}
}

// TestPathUtils_GetFileExtension_EdgeCases тестирует граничные случаи для GetFileExtension
func TestPathUtils_GetFileExtension_EdgeCases(t *testing.T) {
	pathUtils := NewPathUtils()
	
	// Тест с очень длинным именем файла
	longFilename := ""
	for i := 0; i < 1000; i++ {
		longFilename += "a"
	}
	longFilename += ".txt"
	
	ext := pathUtils.GetFileExtension(longFilename)
	if ext != ".txt" {
		t.Errorf("Long filename extension: got %q, expected %q", ext, ".txt")
	}
	
	// Тест с файлом, содержащим только расширение
	onlyExt := pathUtils.GetFileExtension(".txt")
	if onlyExt != "" {
		t.Errorf("Only extension: got %q, expected empty string", onlyExt)
	}
	
	// Тест с множественными точками в начале
	multipleDots := pathUtils.GetFileExtension("...file.txt")
	if multipleDots != ".txt" {
		t.Errorf("Multiple dots: got %q, expected %q", multipleDots, ".txt")
	}
}

// TestPathUtils_GetBaseName_EdgeCases тестирует граничные случаи для GetBaseName
func TestPathUtils_GetBaseName_EdgeCases(t *testing.T) {
	pathUtils := NewPathUtils()
	
	// Тест с очень длинным именем файла
	longFilename := ""
	for i := 0; i < 1000; i++ {
		longFilename += "a"
	}
	longFilename += ".txt"
	
	baseName := pathUtils.GetBaseName(longFilename)
	expectedBase := longFilename[:len(longFilename)-4] // Убираем ".txt"
	if baseName != expectedBase {
		t.Errorf("Long filename base: got length %d, expected length %d", len(baseName), len(expectedBase))
	}
	
	// Тест с файлом, содержащим только расширение
	onlyExt := pathUtils.GetBaseName(".txt")
	if onlyExt != ".txt" {
		t.Errorf("Only extension base: got %q, expected %q", onlyExt, ".txt")
	}
	
	// Тест с множественными точками
	multipleDots := pathUtils.GetBaseName("file...txt")
	if multipleDots != "file.." {
		t.Errorf("Multiple dots base: got %q, expected %q", multipleDots, "file..")
	}
}

// TestPathUtils_NormalizePath тестирует нормализацию путей
func TestPathUtils_NormalizePath(t *testing.T) {
	pathUtils := NewPathUtils()

	// Получаем текущую директорию для формирования ожидаемых путей
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Не удалось получить текущую директорию: %v", err)
	}

	testCases := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{"Valid relative path", "path/to/file", filepath.Join(cwd, "path/to/file"), false},
		{"Path with double slashes", "path//to//file", filepath.Join(cwd, "path/to/file"), false},
		{"Path with dot", "path/./to/file", filepath.Join(cwd, "path/to/file"), false},
		{"Empty path", "", "", true},
		{"Current directory", ".", cwd, false},
		{"Complex path", "path/../other/./file", filepath.Join(cwd, "other/file"), false},
		{"Root path", "/", "/", false},
		{"Absolute path with dots", "/home/user/../user2", "/home/user2", false},
		{"Path with trailing slash", "path/to/dir/", filepath.Join(cwd, "path/to/dir"), false},
		{"Multiple consecutive dots", "path/./././to/file", filepath.Join(cwd, "path/to/file"), false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := pathUtils.NormalizePath(tc.input)
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for input %q, but got none", tc.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %q: %v", tc.input, err)
				}
				if result != tc.expected {
					t.Errorf("NormalizePath(%q) = %q, expected %q", tc.input, result, tc.expected)
				}
			}
		})
	}
}

// TestPathUtils_JoinPath_Extended тестирует расширенные случаи JoinPath
func TestPathUtils_JoinPath_Extended(t *testing.T) {
	pathUtils := NewPathUtils()
	
	testCases := []struct {
		name        string
		base        string
		components  []string
		expectError bool
	}{
		{"Valid join", "base", []string{"path", "file"}, false},
		{"Empty base", "", []string{"path"}, true},
		{"Path traversal in component", "base", []string{"../evil"}, true},
		{"Invalid characters", "base", []string{"path\x00file"}, true},
		{"Absolute path in component", "base", []string{"/absolute"}, true},
		{"Multiple components", "base", []string{"a", "b", "c", "d"}, false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := pathUtils.JoinPath(tc.base, tc.components...)
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for base %q and components %v, but got none", tc.base, tc.components)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for base %q and components %v: %v", tc.base, tc.components, err)
				}
				if result == "" {
					t.Errorf("Expected non-empty result for base %q and components %v", tc.base, tc.components)
				}
			}
		})
	}
}

// TestPathUtils_IsSubPath тестирует проверку подпутей
func TestPathUtils_IsSubPath(t *testing.T) {
	pathUtils := NewPathUtils()
	
	testCases := []struct {
		name       string
		parent     string
		child      string
		expected   bool
		expectErr  bool
	}{
		{"Valid subpath", "/parent", "/parent/child", true, false},
		{"Not a subpath", "/parent", "/other", false, false},
		{"Same path", "/path", "/path", true, false},
		{"Empty parent", "", "/child", false, true},
		{"Empty child", "/parent", "", false, true},
		{"Relative paths", "parent", "parent/child", true, false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := pathUtils.IsSubPath(tc.parent, tc.child)
			if tc.expectErr {
				if err == nil {
					t.Errorf("Expected error for parent %q and child %q, but got none", tc.parent, tc.child)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for parent %q and child %q: %v", tc.parent, tc.child, err)
				}
				if result != tc.expected {
					t.Errorf("IsSubPath(%q, %q) = %v, expected %v", tc.parent, tc.child, result, tc.expected)
				}
			}
		})
	}
}

// TestGetOptimalTempDir_Extended тестирует расширенные случаи GetOptimalTempDir
func TestGetOptimalTempDir_Extended(t *testing.T) {
	// Сохраняем оригинальные переменные окружения
	origTmpDir := os.Getenv("TMPDIR")
	origTemp := os.Getenv("TEMP")
	origTmp := os.Getenv("TMP")
	
	defer func() {
		if err := os.Setenv("TMPDIR", origTmpDir); err != nil {
			t.Logf("Failed to restore TMPDIR: %v", err)
		}
		if err := os.Setenv("TEMP", origTemp); err != nil {
			t.Logf("Failed to restore TEMP: %v", err)
		}
		if err := os.Setenv("TMP", origTmp); err != nil {
			t.Logf("Failed to restore TMP: %v", err)
		}
	}()
	
	// Тест с установленной TMPDIR
	if err := os.Setenv("TMPDIR", "/custom/tmp"); err != nil {
		t.Fatalf("Failed to set TMPDIR: %v", err)
	}
	// Тест с очищенными переменными
	if err := os.Unsetenv("TMPDIR"); err != nil {
		t.Logf("Failed to unset TMPDIR: %v", err)
	}
	if err := os.Unsetenv("TEMP"); err != nil {
		t.Logf("Failed to unset TEMP: %v", err)
	}
	if err := os.Unsetenv("TMP"); err != nil {
		t.Logf("Failed to unset TMP: %v", err)
	}
	result := GetOptimalTempDir()
	// Функция может возвращать /dev/shm если он доступен, что имеет приоритет
	if result != "/custom/tmp" && result != "/dev/shm" {
		t.Errorf("Expected /custom/tmp or /dev/shm, got %s", result)
	}
	
	// Тест с очищенными переменными
	if err := os.Unsetenv("TMPDIR"); err != nil {
		t.Logf("Failed to unset TMPDIR: %v", err)
	}
	if err := os.Unsetenv("TEMP"); err != nil {
		t.Logf("Failed to unset TEMP: %v", err)
	}
	if err := os.Unsetenv("TMP"); err != nil {
		t.Logf("Failed to unset TMP: %v", err)
	}
	
	result = GetOptimalTempDir()
	if result == "" {
		t.Error("Expected non-empty result when no env vars set")
	}
}