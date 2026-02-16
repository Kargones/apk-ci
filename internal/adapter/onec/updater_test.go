package onec

import (
	"strings"
	"testing"
)

// TestExtractMessages проверяет извлечение сообщений из вывода команды.
func TestExtractMessages(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "пустой ввод",
			input:    "",
			expected: nil,
		},
		{
			name:     "одна строка",
			input:    "Сообщение 1",
			expected: []string{"Сообщение 1"},
		},
		{
			name:     "несколько строк LF",
			input:    "Строка 1\nСтрока 2\nСтрока 3",
			expected: []string{"Строка 1", "Строка 2", "Строка 3"},
		},
		{
			name:     "несколько строк CRLF (Windows)",
			input:    "Строка 1\r\nСтрока 2\r\nСтрока 3",
			expected: []string{"Строка 1", "Строка 2", "Строка 3"},
		},
		{
			name:     "смешанные окончания строк",
			input:    "Строка 1\r\nСтрока 2\nСтрока 3\rСтрока 4",
			expected: []string{"Строка 1", "Строка 2", "Строка 3", "Строка 4"},
		},
		{
			name:     "пустые строки фильтруются",
			input:    "Строка 1\n\n\nСтрока 2\n",
			expected: []string{"Строка 1", "Строка 2"},
		},
		{
			name:     "BOM фильтруется",
			input:    "\ufeff\nСообщение",
			expected: []string{"Сообщение"},
		},
		{
			name:     "только BOM",
			input:    "\ufeff",
			expected: nil,
		},
		{
			name:     "пробелы в начале и конце строк",
			input:    "  Строка 1  \n  Строка 2  ",
			expected: []string{"Строка 1", "Строка 2"},
		},
		{
			name:     "реальный вывод 1C",
			input:    "\ufeff\r\nОбновление конфигурации успешно завершено\r\n",
			expected: []string{"Обновление конфигурации успешно завершено"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractMessages(tt.input)

			if len(got) != len(tt.expected) {
				t.Errorf("extractMessages() вернул %d сообщений, ожидалось %d\nполучено: %v\nожидалось: %v",
					len(got), len(tt.expected), got, tt.expected)
				return
			}

			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("extractMessages()[%d] = %q, ожидалось %q", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

// TestExtractMessages_MaxLimit проверяет лимит на количество сообщений (M4 fix).
func TestExtractMessages_MaxLimit(t *testing.T) {
	// Создаём ввод с 150 строками (больше maxExtractedMessages=100)
	var lines []string
	for i := 0; i < 150; i++ {
		lines = append(lines, "Сообщение")
	}
	input := strings.Join(lines, "\n")

	got := extractMessages(input)

	if len(got) != maxExtractedMessages {
		t.Errorf("extractMessages() вернул %d сообщений, ожидалось максимум %d",
			len(got), maxExtractedMessages)
	}
}

// TestTrimOutput проверяет обрезку вывода до максимальной длины.
func TestTrimOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "короткий вывод без обрезки",
			input:    "Короткое сообщение",
			expected: "Короткое сообщение",
		},
		{
			name:     "пустой вывод",
			input:    "",
			expected: "",
		},
		{
			name:     "ровно 500 символов",
			input:    strings.Repeat("a", 500),
			expected: strings.Repeat("a", 500),
		},
		{
			name:     "501 символ - обрезается",
			input:    strings.Repeat("a", 501),
			expected: strings.Repeat("a", 500) + "...",
		},
		{
			name:     "длинный вывод",
			input:    strings.Repeat("x", 1000),
			expected: strings.Repeat("x", 500) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimOutput(tt.input)
			if got != tt.expected {
				t.Errorf("trimOutput() = %q (len=%d), ожидалось %q (len=%d)",
					truncateForError(got), len(got),
					truncateForError(tt.expected), len(tt.expected))
			}
		})
	}
}

// truncateForError обрезает строку для читаемого вывода в ошибках теста.
func truncateForError(s string) string {
	if len(s) > 50 {
		return s[:50] + "..."
	}
	return s
}

// TestNewUpdater проверяет создание Updater.
func TestNewUpdater(t *testing.T) {
	u := NewUpdater("/path/to/1cv8", "/work/dir", "/tmp/dir")

	if u.bin1cv8 != "/path/to/1cv8" {
		t.Errorf("NewUpdater().bin1cv8 = %q, ожидалось %q", u.bin1cv8, "/path/to/1cv8")
	}
	if u.workDir != "/work/dir" {
		t.Errorf("NewUpdater().workDir = %q, ожидалось %q", u.workDir, "/work/dir")
	}
	if u.tmpDir != "/tmp/dir" {
		t.Errorf("NewUpdater().tmpDir = %q, ожидалось %q", u.tmpDir, "/tmp/dir")
	}
}

// TestNewUpdater_EmptyPaths проверяет создание Updater с пустыми путями.
func TestNewUpdater_EmptyPaths(t *testing.T) {
	u := NewUpdater("", "", "")

	if u.bin1cv8 != "" {
		t.Errorf("NewUpdater().bin1cv8 = %q, ожидалось пустую строку", u.bin1cv8)
	}
	if u.workDir != "" {
		t.Errorf("NewUpdater().workDir = %q, ожидалось пустую строку", u.workDir)
	}
	if u.tmpDir != "" {
		t.Errorf("NewUpdater().tmpDir = %q, ожидалось пустую строку", u.tmpDir)
	}
}
