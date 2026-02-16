package logging

// Поддерживаемые форматы вывода логов.
const (
	FormatJSON = "json"
	FormatText = "text"
)

// Поддерживаемые уровни логирования.
const (
	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
)

// Поддерживаемые типы вывода логов.
const (
	OutputStderr = "stderr"
	OutputFile   = "file"
)

// Значения по умолчанию для Config.
// Единый источник истины — используется в ProvideLogger и getDefaultLoggingConfig.
const (
	DefaultLevel      = LevelInfo
	DefaultFormat     = FormatText
	DefaultOutput     = OutputStderr
	DefaultFilePath   = "/var/log/apk-ci.log"
	DefaultMaxSize    = 100 // MB
	DefaultMaxBackups = 3
	DefaultMaxAge     = 7 // days
	DefaultCompress   = true
)

// DefaultConfig возвращает Config со значениями по умолчанию.
func DefaultConfig() Config {
	return Config{
		Level:      DefaultLevel,
		Format:     DefaultFormat,
		Output:     DefaultOutput,
		FilePath:   DefaultFilePath,
		MaxSize:    DefaultMaxSize,
		MaxBackups: DefaultMaxBackups,
		MaxAge:     DefaultMaxAge,
		Compress:   DefaultCompress,
	}
}

// Config содержит настройки логирования.
type Config struct {
	// Format определяет формат вывода: "json" или "text".
	// По умолчанию: "text".
	Format string

	// Level определяет минимальный уровень логирования.
	// По умолчанию: "info".
	// Допустимые значения: "debug", "info", "warn", "error".
	Level string

	// Output определяет куда выводить логи: "stderr" или "file".
	// По умолчанию: "stderr" (backward compatible).
	Output string

	// FilePath задаёт путь к файлу логов (при output="file").
	// По умолчанию: "/var/log/apk-ci.log".
	FilePath string

	// MaxSize задаёт максимальный размер файла в мегабайтах
	// перед ротацией. По умолчанию: 100 МБ.
	MaxSize int

	// MaxBackups задаёт количество backup файлов.
	// По умолчанию: 3.
	MaxBackups int

	// MaxAge задаёт максимальный возраст backup файлов в днях.
	// По умолчанию: 7 дней.
	MaxAge int

	// Compress определяет сжимать ли backup файлы в gzip.
	// По умолчанию: true.
	Compress bool
}
