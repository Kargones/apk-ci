// Package mssql определяет интерфейсы и типы данных для работы с Microsoft SQL Server.
// Пакет предоставляет абстракцию над MSSQL операциями, разделённую по принципу ISP
// (Interface Segregation Principle) на сфокусированные интерфейсы:
// DatabaseConnector, DatabaseRestorer, BackupInfoProvider.
// Композитный интерфейс Client объединяет все вышеперечисленные.
package mssql

import (
	"context"
	"time"
)

// Коды ошибок для MSSQL операций.
const (
	// ErrMSSQLConnect — ошибка подключения к серверу MSSQL
	ErrMSSQLConnect = "MSSQL.CONNECT_FAILED"
	// ErrMSSQLRestore — ошибка восстановления базы данных
	ErrMSSQLRestore = "MSSQL.RESTORE_FAILED"
	// ErrMSSQLQuery — ошибка выполнения SQL запроса
	ErrMSSQLQuery = "MSSQL.QUERY_FAILED"
	// ErrMSSQLTimeout — превышено время ожидания операции
	ErrMSSQLTimeout = "MSSQL.TIMEOUT"
)

// RestoreOptions содержит параметры для восстановления базы данных.
type RestoreOptions struct {
	// Description — описание операции восстановления
	Description string
	// TimeToRestore — временная метка для point-in-time recovery (формат SQL Server)
	TimeToRestore string
	// User — пользователь, запустивший операцию
	User string
	// SrcServer — сервер-источник резервной копии
	SrcServer string
	// SrcDB — имя базы данных источника
	SrcDB string
	// DstServer — целевой сервер для восстановления
	DstServer string
	// DstDB — имя целевой базы данных
	DstDB string
	// Timeout — таймаут операции восстановления
	Timeout time.Duration
}

// StatsOptions содержит параметры для получения статистики восстановления.
type StatsOptions struct {
	// SrcDB — имя исходной базы данных
	SrcDB string
	// DstServer — целевой сервер
	DstServer string
	// TimeToStatistic — период для расчёта статистики
	TimeToStatistic string
}

// RestoreStats содержит статистику операций восстановления.
type RestoreStats struct {
	// AvgRestoreTimeSec — среднее время восстановления в секундах
	AvgRestoreTimeSec int64
	// MaxRestoreTimeSec — максимальное время восстановления в секундах
	MaxRestoreTimeSec int64
	// HasData — имеются ли данные статистики
	HasData bool
}

// BackupInfo содержит информацию о резервной копии.
// M-1 note: Структура определена для будущего расширения интерфейса.
// В текущей версии интерфейс BackupInfoProvider.GetBackupSize возвращает только int64.
// При необходимости расширения (дата создания, тип бэкапа, путь к файлу)
// следует изменить сигнатуру GetBackupSize на GetBackupInfo() (*BackupInfo, error).
// Может быть удалён при рефакторинге адаптеров.
type BackupInfo struct {
	// SizeBytes — размер резервной копии в байтах
	SizeBytes int64
	// Database — имя базы данных
	Database string
}

// DatabaseConnector предоставляет операции для подключения к серверу MSSQL.
type DatabaseConnector interface {
	// Connect устанавливает соединение с сервером MSSQL.
	Connect(ctx context.Context) error
	// Close закрывает соединение с сервером.
	Close() error
	// Ping проверяет доступность сервера.
	Ping(ctx context.Context) error
}

// DatabaseRestorer предоставляет операции для восстановления баз данных.
type DatabaseRestorer interface {
	// Restore выполняет восстановление базы данных из резервной копии.
	Restore(ctx context.Context, opts RestoreOptions) error
	// GetRestoreStats возвращает статистику операций восстановления.
	GetRestoreStats(ctx context.Context, opts StatsOptions) (*RestoreStats, error)
}

// BackupInfoProvider предоставляет операции для получения информации о резервных копиях.
type BackupInfoProvider interface {
	// GetBackupSize возвращает размер последней резервной копии базы данных в байтах.
	GetBackupSize(ctx context.Context, database string) (int64, error)
}

// Client — композитный интерфейс, объединяющий все операции MSSQL.
type Client interface {
	DatabaseConnector
	DatabaseRestorer
	BackupInfoProvider
}
