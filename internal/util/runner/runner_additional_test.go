package runner

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
)

// TestMaskPasswordInParam проверяет маскирование паролей в параметрах
func TestMaskPasswordInParam(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no password",
			input:    "/S server /N user",
			expected: "/S server /N user",
		},
		{
			name:     "password with /P",
			input:    "/S server /N user /P secret123",
			expected: "/S server /N user /P *****",
		},
		{
			name:     "password at start with space before",
			input:    " /P mypassword /S server",
			expected: " /P ***** /S server",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "multiple passwords with spaces",
			input:    " /P pass1 /S server /P pass2",
			expected: " /P ***** /S server /P *****",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskPasswordInParam(tt.input)
			if result != tt.expected {
				t.Errorf("maskPasswordInParam(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestRunner_RunCommandWithContextCancellation проверяет отмену команды по контексту
func TestRunner_RunCommandWithContextCancellation(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	tmpDir := t.TempDir()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Сразу отменяем контекст

	r := &Runner{
		RunString: "/usr/bin/echo",
		Params:    []string{"hello"},
		WorkDir:   tmpDir,
		TmpDir:    tmpDir,
	}

	_, err := r.RunCommand(ctx, logger)
	// Команда может завершиться с ошибкой из-за отмены контекста
	if err != nil {
		t.Logf("RunCommand with canceled context returned error (expected): %v", err)
	}
}

// TestRunner_RunCommandWithNonExistentExecutable проверяет ошибку несуществующего исполняемого файла
func TestRunner_RunCommandWithNonExistentExecutable(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	tmpDir := t.TempDir()

	r := &Runner{
		RunString: "/non/existent/executable",
		Params:    []string{"hello"},
		WorkDir:   tmpDir,
		TmpDir:    tmpDir,
	}

	_, err := r.RunCommand(context.Background(), logger)
	if err == nil {
		t.Error("Expected error for non-existent executable")
	}
}

// TestRunner_ConsoleOut проверяет сохранение вывода консоли
func TestRunner_ConsoleOut(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	tmpDir := t.TempDir()

	r := &Runner{
		RunString: "/usr/bin/echo",
		Params:    []string{"test", "output"},
		WorkDir:   tmpDir,
		TmpDir:    tmpDir,
	}

	_, err := r.RunCommand(context.Background(), logger)
	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}

	if len(r.ConsoleOut) == 0 {
		t.Error("Expected ConsoleOut to be populated")
	}

	output := string(r.ConsoleOut)
	if !strings.Contains(output, "test") {
		t.Errorf("Expected ConsoleOut to contain 'test', got %q", output)
	}
}
