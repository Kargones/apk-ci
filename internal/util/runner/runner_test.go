package runner

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunner_ClearParams проверяет очистку параметров
func TestRunner_ClearParams(t *testing.T) {
	r := &Runner{
		Params: []string{"param1", "param2", "param3"},
	}
	
	r.ClearParams()
	
	if len(r.Params) != 0 {
		t.Errorf("Expected empty params, got %v", r.Params)
	}
}

// TestRunner_RunCommand проверяет выполнение команд
func TestRunner_RunCommand(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	tmpDir := t.TempDir()
	
	tests := []struct {
		name        string
		runner      *Runner
		expectError bool
		errorMsg    string
	}{
		{
			name: "empty executable",
			runner: &Runner{
				RunString: "",
				Params:    []string{},
				WorkDir:   tmpDir,
				TmpDir:    tmpDir,
			},
			expectError: true,
			errorMsg:    "executable path is empty",
		},
		{
			name: "unsafe parameter with semicolon",
			runner: &Runner{
				RunString: "/usr/bin/echo",
				Params:    []string{"test; rm -rf /"},
				WorkDir:   tmpDir,
				TmpDir:    tmpDir,
			},
			expectError: true,
			errorMsg:    "potentially unsafe parameter detected",
		},
		{
			name: "unsafe parameter with ampersand",
			runner: &Runner{
				RunString: "/usr/bin/echo",
				Params:    []string{"test & rm -rf /"},
				WorkDir:   tmpDir,
				TmpDir:    tmpDir,
			},
			expectError: true,
			errorMsg:    "potentially unsafe parameter detected",
		},
		{
			name: "unsafe parameter with pipe",
			runner: &Runner{
				RunString: "/usr/bin/echo",
				Params:    []string{"test | rm -rf /"},
				WorkDir:   tmpDir,
				TmpDir:    tmpDir,
			},
			expectError: true,
			errorMsg:    "potentially unsafe parameter detected",
		},
		{
			name: "valid echo command",
			runner: &Runner{
				RunString: "/usr/bin/echo",
				Params:    []string{"hello", "world"},
				WorkDir:   tmpDir,
				TmpDir:    tmpDir,
			},
			expectError: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.runner.RunCommand(context.Background(), logger)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
				// Для случаев с ошибками параметры могут не очищаться
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				// Проверяем, что параметры очищены после успешного выполнения
				if len(tt.runner.Params) != 0 {
					t.Errorf("Expected params to be cleared after execution, got %v", tt.runner.Params)
				}
			}
		})
	}
}

// TestRunner_RunCommandWithTempFile проверяет выполнение команд с временными файлами
func TestRunner_RunCommandWithTempFile(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	tmpDir := t.TempDir()
	
	r := &Runner{
		RunString: "/usr/bin/echo",
		Params:    []string{"@", "/c", "test", "/Out"},
		WorkDir:   tmpDir,
		TmpDir:    tmpDir,
	}

	_, err := r.RunCommand(context.Background(), logger)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Проверяем, что OutFileName установлен
	if r.OutFileName == "" {
		t.Error("Expected OutFileName to be set")
	}
}

// TestAppendEnviron проверяет функцию добавления переменных окружения
func TestAppendEnviron(t *testing.T) {
	// Сохраняем исходное окружение
	originalEnv := os.Environ()
	defer func() {
		// Восстанавливаем окружение
		os.Clearenv()
		for _, env := range originalEnv {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				if err := os.Setenv(parts[0], parts[1]); err != nil {
					t.Logf("Failed to restore env var %s: %v", parts[0], err)
				}
			}
		}
	}()
	
	// Устанавливаем тестовое окружение
	os.Clearenv()
	if err := os.Setenv("EXISTING_VAR", "old_value"); err != nil {
		t.Fatalf("Failed to set EXISTING_VAR: %v", err)
	}
	if err := os.Setenv("ANOTHER_VAR", "another_value"); err != nil {
		t.Fatalf("Failed to set ANOTHER_VAR: %v", err)
	}
	
	tests := []struct {
		name     string
		input    []string
		expected map[string]string
	}{
		{
			name:  "add new variable",
			input: []string{"NEW_VAR=new_value"},
			expected: map[string]string{
				"EXISTING_VAR": "old_value",
				"ANOTHER_VAR":  "another_value",
				"NEW_VAR":      "new_value",
			},
		},
		{
			name:  "replace existing variable",
			input: []string{"EXISTING_VAR=new_value"},
			expected: map[string]string{
				"EXISTING_VAR": "new_value",
				"ANOTHER_VAR":  "another_value",
			},
		},
		{
			name:  "invalid format ignored",
			input: []string{"INVALID_FORMAT"},
			expected: map[string]string{
				"EXISTING_VAR": "old_value",
				"ANOTHER_VAR":  "another_value",
			},
		},
		{
			name:  "multiple variables",
			input: []string{"VAR1=value1", "VAR2=value2"},
			expected: map[string]string{
				"EXISTING_VAR": "old_value",
				"ANOTHER_VAR":  "another_value",
				"VAR1":         "value1",
				"VAR2":         "value2",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := appendEnviron(tt.input...)
			
			// Преобразуем результат в map для удобства проверки
			resultMap := make(map[string]string)
			for _, env := range result {
				parts := strings.SplitN(env, "=", 2)
				if len(parts) == 2 {
					resultMap[parts[0]] = parts[1]
				}
			}
			
			// Проверяем ожидаемые переменные
			for key, expectedValue := range tt.expected {
				if actualValue, exists := resultMap[key]; !exists {
					t.Errorf("Expected variable %s to exist", key)
				} else if actualValue != expectedValue {
					t.Errorf("Expected %s=%s, got %s=%s", key, expectedValue, key, actualValue)
				}
			}
		})
	}
}

// TestExists проверяет функцию проверки существования файла
func TestExists(t *testing.T) {
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "existing.txt")
	nonExistingFile := filepath.Join(tmpDir, "nonexisting.txt")
	
	// Создаем существующий файл
	file, err := os.Create(existingFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Failed to close test file: %v", err)
	}
	
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"existing file", existingFile, true},
		{"non-existing file", nonExistingFile, false},
		{"existing directory", tmpDir, true},
		{"empty path", "", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := exists(tt.path)
			if result != tt.expected {
				t.Errorf("exists(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// TestTrimOut проверяет функцию обрезки вывода
func TestTrimOut(t *testing.T) {
	tests := []struct { //nolint:prealloc // test table
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "short output",
			input:    []byte("short message"),
			expected: "short message",
		},
		{
			name:     "empty output",
			input:    []byte(""),
			expected: "",
		},
		{
			name:     "exactly max length",
			input:    make([]byte, maxConsoleOut),
			expected: string(make([]byte, 1020)) + "\n********\n" + string(make([]byte, 1020)),
		},
	}
	
	// Тест для длинного вывода
	longInput := make([]byte, maxConsoleOut+1000)
	for i := range longInput {
		longInput[i] = 'a'
	}
	tests = append(tests, struct {
		name     string
		input    []byte
		expected string
	}{
		name:  "long output",
		input: longInput,
		expected: string(longInput[:1020]) + "\n********\n" + string(longInput[len(longInput)-1020:]),
	})
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TrimOut(tt.input)
			if result != tt.expected {
				t.Errorf("TrimOut() length mismatch: got %d, want %d", len(result), len(tt.expected))
				if len(result) < 100 && len(tt.expected) < 100 {
					t.Errorf("TrimOut() = %q, want %q", result, tt.expected)
				}
			}
		})
	}
}
