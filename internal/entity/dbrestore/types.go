// Package dbrestore предоставляет функционал для восстановления баз данных MS SQL Server
// из исторических резервных копий с автоматическим расчетом таймаутов на основе статистики.
//
// Основные возможности:
// - Подключение к MS SQL Server
// - Восстановление баз данных из исторических бэкапов
// - Получение статистики времени восстановления
// - Автоматический расчет таймаутов на основе исторических данных
// - Гибкая конфигурация через YAML файлы
package dbrestore

import (
	"database/sql"
	"time"
)

const (
	// DefaultPort - порт по умолчанию для подключения к MS SQL Server отвечающему за восстановление
	DefaultPort = 1433
	// DefaultServer - сервер по умолчанию для подключения к MS SQL Server.
	DefaultServer = "MSK-SQL-SVC-01"
)

// DBRestore представляет клиент для восстановления баз данных MS SQL Server.
// Содержит все необходимые параметры для подключения к серверу и выполнения операций восстановления.
type DBRestore struct {
	// Db - активное подключение к базе данных
	Db *sql.DB

	// Параметры подключения к серверу
	Server   string `yaml:"server"`   // Адрес сервера MS SQL
	User     string `yaml:"user"`     // Имя пользователя для подключения
	Password string `yaml:"password"` // Пароль пользователя
	Port     int    `yaml:"port"`     // Порт сервера (по умолчанию 1433)
	Database string `yaml:"database"` // База данных для подключения (обычно master)

	// Параметры восстановления
	Timeout         time.Duration `yaml:"timeout"`        // Таймаут операции восстановления
	TimeToRestore   string        `yaml:"time2restore"`   // Время восстановления в формате RFC3339
	TimeToStatistic string        `yaml:"time2statistic"` // Начальная дата для сбора статистики
	AutoTimeOut     bool          `yaml:"autotimeout"`    // Автоматический расчет таймаута на основе статистики
	Description     string        `yaml:"description"`    // Описание задачи восстановления

	// Параметры источника и назначения
	SrcServer string `yaml:"srcServer"` // Сервер-источник
	SrcDB     string `yaml:"srcDB"`     // База данных-источник
	DstServer string `yaml:"dstServer"` // Сервер назначения
	DstDB     string `yaml:"dstDB"`     // База данных назначения
}

// RestoreStats содержит статистику времени восстановления баз данных.
// Используется для анализа производительности и автоматического расчета таймаутов.
type RestoreStats struct {
	// AvgRestoreTimeSecond - среднее время восстановления в секундах
	AvgRestoreTimeSecond sql.NullInt64
	// MaxRestoreTimeSecond - максимальное время восстановления в секундах
	MaxRestoreTimeSecond sql.NullInt64
}

// New создает новый экземпляр DBRestore с настройками по умолчанию.
// Возвращает указатель на пустую структуру DBRestore, которую необходимо
// дополнительно настроить перед использованием.
// Возвращает:
//   - *DBRestore: новый экземпляр структуры DBRestore
func New() *DBRestore {
	return &DBRestore{}
}
