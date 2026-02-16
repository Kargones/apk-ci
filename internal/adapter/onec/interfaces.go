// Package onec определяет интерфейсы и типы данных для работы с 1C:Предприятие.
// Пакет предоставляет абстракцию над операциями 1C, следуя принципу ISP
// (Interface Segregation Principle) с минимальными сфокусированными интерфейсами.
package onec

import (
	"context"
	"time"
)

// DatabaseUpdater определяет операцию обновления структуры БД.
// Минимальный интерфейс для ISP паттерна.
type DatabaseUpdater interface {
	// UpdateDBCfg выполняет команду 1cv8 DESIGNER /UpdateDBCfg.
	// Применяет изменения конфигурации к базе данных.
	UpdateDBCfg(ctx context.Context, opts UpdateOptions) (*UpdateResult, error)
}

// UpdateOptions параметры для обновления структуры БД.
type UpdateOptions struct {
	// ConnectString — строка подключения "/S server\base /N user /P pass"
	ConnectString string
	// Extension — имя расширения (пусто для основной конфигурации)
	Extension string
	// Timeout — таймаут операции
	Timeout time.Duration
	// Bin1cv8 — путь к исполняемому файлу 1cv8
	Bin1cv8 string
}

// UpdateResult результат обновления структуры БД.
type UpdateResult struct {
	// Success — успешно ли обновление
	Success bool
	// Messages — сообщения от платформы
	Messages []string
	// DurationMs — время выполнения в миллисекундах
	DurationMs int64
}

// TempDatabaseCreator определяет операцию создания временной БД.
// Минимальный интерфейс для ISP паттерна.
type TempDatabaseCreator interface {
	// CreateTempDB создаёт локальную файловую БД через ibcmd.
	CreateTempDB(ctx context.Context, opts CreateTempDBOptions) (*TempDBResult, error)
}

// CreateTempDBOptions параметры для создания временной БД.
type CreateTempDBOptions struct {
	// DbPath — путь к создаваемой БД (директория)
	DbPath string
	// Extensions — список расширений для добавления
	Extensions []string
	// Timeout — таймаут операции
	Timeout time.Duration
	// BinIbcmd — путь к исполняемому файлу ibcmd
	BinIbcmd string
}

// TempDBResult результат создания временной БД.
type TempDBResult struct {
	// ConnectString — строка подключения "/F <path>"
	ConnectString string
	// DbPath — полный путь к созданной БД
	DbPath string
	// Extensions — список добавленных расширений
	Extensions []string
	// CreatedAt — время создания
	CreatedAt time.Time
	// DurationMs — время выполнения в миллисекундах
	DurationMs int64
}

// ConfigExporter определяет операцию выгрузки конфигурации.
// Минимальный интерфейс для ISP паттерна (Interface Segregation Principle).
// Поддерживает выгрузку основной конфигурации и расширений в XML формат.
type ConfigExporter interface {
	// Export выгружает конфигурацию в XML формат.
	// Возвращает результат выгрузки с информацией об успехе и пути к файлам.
	Export(ctx context.Context, opts ExportOptions) (*ExportResult, error)
}

// ExportOptions параметры для выгрузки конфигурации.
type ExportOptions struct {
	// ConnectString — строка подключения "/S server\base /N user /P pass" или "/F path"
	ConnectString string
	// OutputPath — путь для выгрузки XML файлов
	OutputPath string
	// Extension — имя расширения (пусто для основной конфигурации)
	Extension string
	// Timeout — таймаут операции
	Timeout time.Duration
}

// ExportResult результат выгрузки конфигурации.
type ExportResult struct {
	// Success — успешно ли выгрузка
	Success bool
	// OutputPath — путь к выгруженным файлам
	OutputPath string
	// Messages — сообщения от платформы
	Messages []string
	// DurationMs — время выполнения в миллисекундах
	DurationMs int64
}

// DatabaseCreator определяет операцию создания базы данных.
// Минимальный интерфейс для ISP паттерна (Interface Segregation Principle).
// Поддерживает создание файловых и серверных информационных баз.
type DatabaseCreator interface {
	// CreateDB создаёт информационную базу.
	// Возвращает результат создания с информацией о подключении.
	CreateDB(ctx context.Context, opts CreateDBOptions) (*CreateDBResult, error)
}

// CreateDBOptions параметры для создания БД.
type CreateDBOptions struct {
	// DbPath — путь к создаваемой БД (для файловой) или connection info (для серверной)
	DbPath string
	// ServerBased — true для серверной БД, false для файловой
	ServerBased bool
	// Server — сервер 1C (для серверной БД)
	Server string
	// DbName — имя базы данных на сервере
	DbName string
	// Timeout — таймаут операции
	Timeout time.Duration
}

// CreateDBResult результат создания БД.
type CreateDBResult struct {
	// Success — успешно ли создание БД
	Success bool
	// ConnectString — строка подключения к созданной БД
	ConnectString string
	// DbPath — полный путь к созданной БД
	DbPath string
	// CreatedAt — время создания
	CreatedAt time.Time
	// DurationMs — время выполнения в миллисекундах
	DurationMs int64
}
