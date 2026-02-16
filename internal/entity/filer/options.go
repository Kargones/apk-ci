package filer

// Option представляет функциональную опцию для настройки конфигурации файловой системы.
// Использует паттерн "Functional Options" для гибкой настройки параметров.
type Option func(*Config)

// WithDiskFS настраивает файловую систему для работы с диском.
// Если basePath пустой, будет использован путь по умолчанию.
func WithDiskFS(basePath string) Option {
	return func(c *Config) {
		c.Type = DiskFS
		c.BasePath = basePath
		c.UseRAM = false
	}
}

// WithMemoryFS настраивает файловую систему для работы в памяти.
// По умолчанию не использует RAM-диск.
func WithMemoryFS() Option {
	return func(c *Config) {
		c.Type = MemoryFS
		c.BasePath = ""
		c.UseRAM = false
	}
}

// WithRAMDisk включает использование RAM-диска (/dev/shm) для файловой системы в памяти.
// Применимо только для MemoryFS на Linux системах.
// Если RAM-диск недоступен, будет использован fallback на обычную временную директорию.
func WithRAMDisk() Option {
	return func(c *Config) {
		c.UseRAM = true
	}
}

// WithBasePath устанавливает базовый путь для файловой системы.
// Применимо для DiskFS.
func WithBasePath(path string) Option {
	return func(c *Config) {
		c.BasePath = path
	}
}

// WithType устанавливает тип файловой системы.
func WithType(fsType FSType) Option {
	return func(c *Config) {
		c.Type = fsType
	}
}

// ApplyOptions применяет список опций к конфигурации.
// Возвращает новую конфигурацию с примененными опциями.
func ApplyOptions(base Config, options ...Option) Config {
	config := base
	for _, option := range options {
		option(&config)
	}
	return config
}

// NewConfig создает новую конфигурацию с применением опций.
// Начинает с конфигурации по умолчанию.
func NewConfig(options ...Option) Config {
	return ApplyOptions(DefaultConfig(), options...)
}