package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

// NewLogger создаёт Logger с заданной конфигурацией.
// Возвращает SlogAdapter настроенный согласно config.
//
// Поддерживаемые режимы вывода (config.Output):
//   - "stderr" или "" (default): логи пишутся в os.Stderr
//   - "file": логи пишутся в файл с автоматической ротацией через lumberjack
//
// При output="file" используются параметры ротации:
//   - MaxSize: максимальный размер файла в MB (default: 100)
//   - MaxBackups: количество backup файлов (default: 3)
//   - MaxAge: возраст backup в днях (default: 7)
//   - Compress: сжатие backup в gzip (default: true)
func NewLogger(config Config) Logger {
	var w io.Writer

	switch config.Output {
	case OutputFile:
		w = newLumberjackWriter(config)
	case OutputStderr, "":
		w = os.Stderr
	default:
		// M-10/Review #16: логируем предупреждение о неизвестном output, чтобы не терять логи молча
		_, _ = os.Stderr.WriteString(fmt.Sprintf( //nolint:errcheck // bootstrap stderr
			"WARNING: неизвестный logging output %q, falling back to stderr\n", config.Output))
		w = os.Stderr
	}

	return NewLoggerWithWriter(config, w)
}

// newLumberjackWriter создаёт io.Writer с ротацией на основе lumberjack.
// Автоматически создаёт директорию для файла логов если не существует.
// При пустом FilePath возвращает os.Stderr как fallback.
func newLumberjackWriter(config Config) io.Writer {
	// Валидация: если FilePath пустой, fallback на stderr (AC2 safety)
	// M-2 fix: логируем warning через stderr, чтобы пользователь знал о fallback
	if config.FilePath == "" {
		_, _ = os.Stderr.WriteString("WARNING: logging output=file but filePath is empty, falling back to stderr\n") //nolint:errcheck // bootstrap stderr
		return os.Stderr
	}

	// Создаём директорию если не существует (AC10)
	dir := filepath.Dir(config.FilePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0750); err != nil {
			_, _ = os.Stderr.WriteString(fmt.Sprintf( //nolint:errcheck // bootstrap stderr
				"WARNING: не удалось создать директорию логов %q: %v, falling back to stderr\n", dir, err))
			return os.Stderr
		}
	}

	return &lumberjack.Logger{
		Filename:   config.FilePath,
		MaxSize:    config.MaxSize,    // MB
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,     // days
		Compress:   config.Compress,
	}
}

// NewLoggerWithWriter создаёт Logger с заданной конфигурацией и writer.
// Используется для тестирования и гибкой настройки вывода.
//
// Для production использования предпочтительнее NewLogger(),
// который выбирает writer на основе config.Output (stderr/file).
func NewLoggerWithWriter(config Config, w io.Writer) Logger {
	// Определяем уровень
	level := parseLevel(config.Level)

	// Создаём handler
	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler

	switch config.Format {
	case FormatJSON:
		handler = slog.NewJSONHandler(w, opts)
	default:
		handler = slog.NewTextHandler(w, opts)
	}

	return NewSlogAdapter(slog.New(handler))
}

// parseLevel конвертирует строковый уровень в slog.Level.
// При неизвестном значении возвращает slog.LevelInfo.
func parseLevel(level string) slog.Level {
	switch level {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		// M6 fix: явный case для LevelInfo для читаемости и maintainability
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		// Неизвестный уровень → используем info как безопасный default
		return slog.LevelInfo
	}
}
