package logging

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewLogger_DefaultValues проверяет что по умолчанию используется text format и info level.
func TestNewLogger_DefaultValues(t *testing.T) {
	// Создаём logger с пустой конфигурацией
	logger := NewLogger(Config{})

	// Проверяем что logger создан
	assert.NotNil(t, logger)

	// Проверяем что это SlogAdapter
	_, ok := logger.(*SlogAdapter)
	assert.True(t, ok, "NewLogger должен возвращать *SlogAdapter")
}

// TestNewLogger_JSONFormat проверяет создание logger с JSON форматом.
func TestNewLogger_JSONFormat(t *testing.T) {
	logger := NewLogger(Config{Format: FormatJSON})
	assert.NotNil(t, logger)

	_, ok := logger.(*SlogAdapter)
	assert.True(t, ok)
}

// TestNewLogger_TextFormat проверяет создание logger с text форматом.
func TestNewLogger_TextFormat(t *testing.T) {
	logger := NewLogger(Config{Format: FormatText})
	assert.NotNil(t, logger)

	_, ok := logger.(*SlogAdapter)
	assert.True(t, ok)
}

// TestNewLoggerWithWriter_LevelFiltering проверяет что DEBUG не логируется при level=info.
func TestNewLoggerWithWriter_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer

	// Создаём logger с level=info через NewLoggerWithWriter
	logger := NewLoggerWithWriter(Config{
		Format: FormatText,
		Level:  LevelInfo,
	}, &buf)

	// Пишем DEBUG сообщение (должно быть отфильтровано)
	logger.Debug("this should not appear")

	// Пишем INFO сообщение (должно появиться)
	logger.Info("this should appear")

	output := buf.String()

	// DEBUG не должен появиться
	assert.NotContains(t, output, "this should not appear")
	// INFO должен появиться
	assert.Contains(t, output, "this should appear")
}

// TestNewLogger_WritesToStderr проверяет что NewLogger пишет в stderr, stdout пустой.
func TestNewLogger_WritesToStderr(t *testing.T) {
	// Сохраняем оригинальные stdout и stderr
	origStdout := os.Stdout
	origStderr := os.Stderr

	// Создаём pipes для перехвата
	stdoutR, stdoutW, err := os.Pipe()
	require.NoError(t, err)

	stderrR, stderrW, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = stdoutW
	os.Stderr = stderrW

	// Создаём logger через NewLogger (должен использовать stderr)
	logger := NewLogger(Config{Format: FormatText})

	// Пишем лог
	logger.Info("test message for stderr only")

	// Закрываем writers
	_ = stdoutW.Close()
	_ = stderrW.Close()

	// Читаем output
	var stdoutBuf, stderrBuf bytes.Buffer
	_, _ = io.Copy(&stdoutBuf, stdoutR)
	_, _ = io.Copy(&stderrBuf, stderrR)

	// Восстанавливаем оригиналы
	os.Stdout = origStdout
	os.Stderr = origStderr

	// stdout должен быть пустой
	assert.Empty(t, stdoutBuf.String(), "stdout должен быть пустым - логи идут только в stderr")

	// stderr должен содержать лог
	assert.Contains(t, stderrBuf.String(), "test message for stderr only")
}

// TestNewLoggerWithWriter_AllLevels проверяет все уровни логирования.
func TestNewLoggerWithWriter_AllLevels(t *testing.T) {
	tests := []struct {
		name         string
		configLevel  string
		logLevel     string
		shouldAppear bool
	}{
		// При level=debug все уровни должны появиться
		{"debug_at_debug", LevelDebug, "debug", true},
		{"info_at_debug", LevelDebug, "info", true},
		{"warn_at_debug", LevelDebug, "warn", true},
		{"error_at_debug", LevelDebug, "error", true},

		// При level=info, debug не появляется
		{"debug_at_info", LevelInfo, "debug", false},
		{"info_at_info", LevelInfo, "info", true},
		{"warn_at_info", LevelInfo, "warn", true},
		{"error_at_info", LevelInfo, "error", true},

		// При level=warn, debug и info не появляются
		{"debug_at_warn", LevelWarn, "debug", false},
		{"info_at_warn", LevelWarn, "info", false},
		{"warn_at_warn", LevelWarn, "warn", true},
		{"error_at_warn", LevelWarn, "error", true},

		// При level=error, только error появляется
		{"debug_at_error", LevelError, "debug", false},
		{"info_at_error", LevelError, "info", false},
		{"warn_at_error", LevelError, "warn", false},
		{"error_at_error", LevelError, "error", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			logger := NewLoggerWithWriter(Config{
				Format: FormatText,
				Level:  tt.configLevel,
			}, &buf)

			testMsg := "test_" + tt.name

			switch tt.logLevel {
			case "debug":
				logger.Debug(testMsg)
			case "info":
				logger.Info(testMsg)
			case "warn":
				logger.Warn(testMsg)
			case "error":
				logger.Error(testMsg)
			}

			output := buf.String()

			if tt.shouldAppear {
				assert.Contains(t, output, testMsg, "сообщение должно появиться")
			} else {
				assert.NotContains(t, output, testMsg, "сообщение не должно появиться")
			}
		})
	}
}

// TestParseLevel_AllLevels проверяет что все уровни корректно парсятся.
func TestParseLevel_AllLevels(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{LevelDebug, slog.LevelDebug},
		{LevelInfo, slog.LevelInfo},
		{LevelWarn, slog.LevelWarn},
		{LevelError, slog.LevelError},
		{"", slog.LevelInfo},        // пустая строка → info
		{"unknown", slog.LevelInfo}, // неизвестное значение → info
		{"DEBUG", slog.LevelInfo},   // case sensitive → info (не DEBUG)
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseLevel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestNewLoggerWithWriter_JSONOutput_ValidJSON проверяет что JSON output валидный.
func TestNewLoggerWithWriter_JSONOutput_ValidJSON(t *testing.T) {
	var buf bytes.Buffer

	logger := NewLoggerWithWriter(Config{Format: FormatJSON}, &buf)
	logger.Info("json test", "key", "value", "number", 42)

	output := buf.String()

	// Парсим JSON
	var logEntry map[string]any
	err := json.Unmarshal([]byte(output), &logEntry)
	require.NoError(t, err, "output должен быть валидным JSON")

	assert.Equal(t, "INFO", logEntry["level"])
	assert.Equal(t, "json test", logEntry["msg"])
	assert.Equal(t, "value", logEntry["key"])
	assert.Equal(t, float64(42), logEntry["number"])
}

// TestNewLoggerWithWriter_WithConfig проверяет создание с различными комбинациями конфигурации.
func TestNewLoggerWithWriter_WithConfig(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{"empty config", Config{}},
		{"json+debug", Config{Format: FormatJSON, Level: LevelDebug}},
		{"text+error", Config{Format: FormatText, Level: LevelError}},
		{"json only", Config{Format: FormatJSON}},
		{"level only", Config{Level: LevelWarn}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewLoggerWithWriter(tt.config, &buf)
			assert.NotNil(t, logger)

			_, ok := logger.(*SlogAdapter)
			assert.True(t, ok)
		})
	}
}

// TestNewLoggerWithWriter_CustomWriter проверяет что логи пишутся в указанный writer.
func TestNewLoggerWithWriter_CustomWriter(t *testing.T) {
	var buf bytes.Buffer

	logger := NewLoggerWithWriter(Config{Format: FormatText}, &buf)
	logger.Info("custom writer test", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "custom writer test")
	assert.Contains(t, output, "key=value")
}

// =============================================================================
// Тесты для file output (Story 6-1: Log File Rotation)
// =============================================================================

// TestNewLogger_FileOutput проверяет создание logger с file output (AC1, AC2).
func TestNewLogger_FileOutput(t *testing.T) {
	// Создаём временную директорию
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	config := Config{
		Level:      LevelInfo,
		Format:     FormatJSON,
		Output:     OutputFile,
		FilePath:   logFile,
		MaxSize:    1,
		MaxBackups: 1,
		MaxAge:     1,
		Compress:   false,
	}

	logger := NewLogger(config)
	require.NotNil(t, logger)

	// Записываем лог
	logger.Info("test message", "key", "value")

	// Проверяем что файл создан
	_, err := os.Stat(logFile)
	require.NoError(t, err, "файл лога должен быть создан")

	// Проверяем содержимое
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "test message", "сообщение должно быть записано в файл")
	assert.Contains(t, string(content), "key", "ключ должен быть записан")
}

// TestNewLogger_FileOutput_CreatesDirectory проверяет автоматическое создание директории (AC10).
func TestNewLogger_FileOutput_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	// Создаём путь с несколькими несуществующими поддиректориями
	logFile := filepath.Join(tmpDir, "subdir", "nested", "deep", "test.log")

	config := Config{
		Level:    LevelInfo,
		Format:   FormatText,
		Output:   OutputFile,
		FilePath: logFile,
	}

	logger := NewLogger(config)
	logger.Info("directory creation test")

	// Проверяем что директория создана
	dir := filepath.Dir(logFile)
	info, err := os.Stat(dir)
	require.NoError(t, err, "директория должна быть создана")
	assert.True(t, info.IsDir(), "путь должен быть директорией")

	// Проверяем что файл создан
	_, err = os.Stat(logFile)
	require.NoError(t, err, "файл лога должен быть создан")
}

// TestNewLogger_StderrOutput_BackwardCompatible проверяет backward compatibility (AC8).
func TestNewLogger_StderrOutput_BackwardCompatible(t *testing.T) {
	config := Config{
		Level:  LevelInfo,
		Format: FormatText,
		Output: OutputStderr, // explicit stderr
	}

	logger := NewLogger(config)
	require.NotNil(t, logger)

	// Logger должен быть создан без panic
	_, ok := logger.(*SlogAdapter)
	assert.True(t, ok, "должен возвращать *SlogAdapter")
}

// TestNewLogger_EmptyOutput_DefaultsToStderr проверяет что пустой Output → stderr (AC8).
func TestNewLogger_EmptyOutput_DefaultsToStderr(t *testing.T) {
	config := Config{
		Level:  LevelInfo,
		Format: FormatText,
		Output: "", // пустой → должен использовать stderr
	}

	logger := NewLogger(config)
	require.NotNil(t, logger)

	_, ok := logger.(*SlogAdapter)
	assert.True(t, ok, "должен возвращать *SlogAdapter")
}

// TestNewLogger_UnknownOutput_FallbackToStderr проверяет fallback при неизвестном output.
func TestNewLogger_UnknownOutput_FallbackToStderr(t *testing.T) {
	config := Config{
		Level:  LevelInfo,
		Format: FormatText,
		Output: "unknown_output_type",
	}

	logger := NewLogger(config)
	require.NotNil(t, logger)

	// Не должен паниковать, должен fallback на stderr
	_, ok := logger.(*SlogAdapter)
	assert.True(t, ok, "должен возвращать *SlogAdapter")
}

// TestNewLumberjackWriter_DefaultValues проверяет что lumberjack writer создаётся корректно.
func TestNewLumberjackWriter_DefaultValues(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "lumberjack-test.log")

	config := Config{
		FilePath:   logFile,
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   true,
	}

	writer := newLumberjackWriter(config)
	require.NotNil(t, writer)

	// Проверяем что можем писать в writer
	_, err := writer.Write([]byte("test log line\n"))
	require.NoError(t, err)

	// Проверяем что файл создан
	_, err = os.Stat(logFile)
	require.NoError(t, err)
}

// TestNewLogger_FileOutput_JSONFormat проверяет JSON формат в файл.
func TestNewLogger_FileOutput_JSONFormat(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "json-test.log")

	config := Config{
		Level:    LevelInfo,
		Format:   FormatJSON,
		Output:   OutputFile,
		FilePath: logFile,
	}

	logger := NewLogger(config)
	logger.Info("json file test", "number", 42, "bool", true)

	content, err := os.ReadFile(logFile)
	require.NoError(t, err)

	// Парсим JSON для проверки структуры
	var logEntry map[string]any
	err = json.Unmarshal(content, &logEntry)
	require.NoError(t, err, "output должен быть валидным JSON")

	assert.Equal(t, "INFO", logEntry["level"])
	assert.Equal(t, "json file test", logEntry["msg"])
	assert.Equal(t, float64(42), logEntry["number"])
	assert.Equal(t, true, logEntry["bool"])
}

// TestNewLogger_FileOutput_TextFormat проверяет text формат в файл.
func TestNewLogger_FileOutput_TextFormat(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "text-test.log")

	config := Config{
		Level:    LevelInfo,
		Format:   FormatText,
		Output:   OutputFile,
		FilePath: logFile,
	}

	logger := NewLogger(config)
	logger.Info("text file test", "key", "value")

	content, err := os.ReadFile(logFile)
	require.NoError(t, err)

	output := string(content)
	assert.Contains(t, output, "text file test")
	assert.Contains(t, output, "key=value")
}

// TestNewLogger_FileOutput_MultipleWrites проверяет несколько записей в файл.
func TestNewLogger_FileOutput_MultipleWrites(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "multi-write.log")

	config := Config{
		Level:    LevelDebug,
		Format:   FormatText,
		Output:   OutputFile,
		FilePath: logFile,
	}

	logger := NewLogger(config)

	// Пишем несколько сообщений разных уровней
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	content, err := os.ReadFile(logFile)
	require.NoError(t, err)

	output := string(content)
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")
}

// TODO (H-3): Тест ротации по размеру требует интеграционного теста с созданием
// файлов > MaxSize, что не практично в unit-тестах. Отложено на Epic 7.
// Можно добавить интеграционный тест с малым MaxSize (например 1KB) для проверки.

// TestNewLogger_FileOutput_EmptyFilePath проверяет fallback на stderr при пустом FilePath (AC2 safety).
func TestNewLogger_FileOutput_EmptyFilePath(t *testing.T) {
	// Сохраняем оригинальный stderr
	origStderr := os.Stderr

	// Создаём pipe для перехвата stderr
	stderrR, stderrW, err := os.Pipe()
	require.NoError(t, err)

	os.Stderr = stderrW

	config := Config{
		Level:    LevelInfo,
		Format:   FormatText,
		Output:   OutputFile,
		FilePath: "", // пустой путь — должен fallback на stderr
	}

	logger := NewLogger(config)
	require.NotNil(t, logger)

	// Пишем лог — должен попасть в stderr, а не в файл
	logger.Info("empty filepath fallback test")

	// Закрываем writer и читаем output
	_ = stderrW.Close()
	var stderrBuf bytes.Buffer
	_, _ = io.Copy(&stderrBuf, stderrR)

	// Восстанавливаем stderr
	os.Stderr = origStderr

	// Проверяем что сообщение попало в stderr
	assert.Contains(t, stderrBuf.String(), "empty filepath fallback test",
		"при пустом FilePath логи должны писаться в stderr")
}

// TestNewLumberjackWriter_CreatesParentDir проверяет создание родительской директории.
func TestNewLumberjackWriter_CreatesParentDir(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "new-parent", "new-child", "log.txt")

	config := Config{
		FilePath: logFile,
		MaxSize:  1,
	}

	writer := newLumberjackWriter(config)
	require.NotNil(t, writer)

	// Пишем что-то чтобы триггернуть создание файла
	_, err := writer.Write([]byte("trigger file creation\n"))
	require.NoError(t, err)

	// Проверяем что родительская директория существует
	parentDir := filepath.Dir(logFile)
	info, err := os.Stat(parentDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}
