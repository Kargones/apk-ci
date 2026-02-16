// Package mssqltest предоставляет тестовые утилиты для пакета mssql:
// мок-реализации интерфейсов и вспомогательные конструкторы.
package mssqltest

import (
	"context"

	"github.com/Kargones/apk-ci/internal/adapter/mssql"
)

// Compile-time проверки реализации интерфейсов
var (
	_ mssql.Client             = (*MockMSSQLClient)(nil)
	_ mssql.DatabaseConnector  = (*MockMSSQLClient)(nil)
	_ mssql.DatabaseRestorer   = (*MockMSSQLClient)(nil)
	_ mssql.BackupInfoProvider = (*MockMSSQLClient)(nil)
)

// MockMSSQLClient — мок-реализация mssql.Client для тестирования.
// Использует функциональные поля для гибкой настройки поведения в тестах.
type MockMSSQLClient struct {
	// ConnectFunc — пользовательская реализация Connect
	ConnectFunc func(ctx context.Context) error
	// CloseFunc — пользовательская реализация Close
	CloseFunc func() error
	// PingFunc — пользовательская реализация Ping
	PingFunc func(ctx context.Context) error
	// RestoreFunc — пользовательская реализация Restore
	RestoreFunc func(ctx context.Context, opts mssql.RestoreOptions) error
	// GetRestoreStatsFunc — пользовательская реализация GetRestoreStats
	GetRestoreStatsFunc func(ctx context.Context, opts mssql.StatsOptions) (*mssql.RestoreStats, error)
	// GetBackupSizeFunc — пользовательская реализация GetBackupSize
	GetBackupSizeFunc func(ctx context.Context, database string) (int64, error)
}

// Connect устанавливает соединение с сервером MSSQL.
// При отсутствии пользовательской функции возвращает nil.
func (m *MockMSSQLClient) Connect(ctx context.Context) error {
	if m.ConnectFunc != nil {
		return m.ConnectFunc(ctx)
	}
	return nil
}

// Close закрывает соединение с сервером.
// При отсутствии пользовательской функции возвращает nil.
func (m *MockMSSQLClient) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// Ping проверяет доступность сервера.
// При отсутствии пользовательской функции возвращает nil.
func (m *MockMSSQLClient) Ping(ctx context.Context) error {
	if m.PingFunc != nil {
		return m.PingFunc(ctx)
	}
	return nil
}

// Restore выполняет восстановление базы данных.
// При отсутствии пользовательской функции возвращает nil.
func (m *MockMSSQLClient) Restore(ctx context.Context, opts mssql.RestoreOptions) error {
	if m.RestoreFunc != nil {
		return m.RestoreFunc(ctx, opts)
	}
	return nil
}

// GetRestoreStats возвращает статистику операций восстановления.
// При отсутствии пользовательской функции возвращает реалистичные тестовые данные.
func (m *MockMSSQLClient) GetRestoreStats(ctx context.Context, opts mssql.StatsOptions) (*mssql.RestoreStats, error) {
	if m.GetRestoreStatsFunc != nil {
		return m.GetRestoreStatsFunc(ctx, opts)
	}
	// Возвращаем реалистичные тестовые данные (по аналогии с RAC mock)
	return &mssql.RestoreStats{
		AvgRestoreTimeSec: 180,  // 3 минуты — типичное время восстановления
		MaxRestoreTimeSec: 600,  // 10 минут — максимальное время
		HasData:           true, // данные статистики доступны
	}, nil
}

// GetBackupSize возвращает размер резервной копии.
// При отсутствии пользовательской функции возвращает реалистичный размер (500 MB).
func (m *MockMSSQLClient) GetBackupSize(ctx context.Context, database string) (int64, error) {
	if m.GetBackupSizeFunc != nil {
		return m.GetBackupSizeFunc(ctx, database)
	}
	// Возвращаем реалистичный размер бэкапа (500 MB)
	return 500 * 1024 * 1024, nil
}

// NewMockMSSQLClient создаёт MockMSSQLClient с дефолтными значениями.
func NewMockMSSQLClient() *MockMSSQLClient {
	return &MockMSSQLClient{}
}

// NewMockMSSQLClientWithRestoreStats создаёт MockMSSQLClient с предзаданной статистикой восстановления.
func NewMockMSSQLClientWithRestoreStats(avgTimeSec, maxTimeSec int64, hasData bool) *MockMSSQLClient {
	return &MockMSSQLClient{
		GetRestoreStatsFunc: func(_ context.Context, _ mssql.StatsOptions) (*mssql.RestoreStats, error) {
			return &mssql.RestoreStats{
				AvgRestoreTimeSec: avgTimeSec,
				MaxRestoreTimeSec: maxTimeSec,
				HasData:           hasData,
			}, nil
		},
	}
}

// NewMockMSSQLClientWithBackupSize создаёт MockMSSQLClient с предзаданным размером бэкапа.
func NewMockMSSQLClientWithBackupSize(sizeBytes int64) *MockMSSQLClient {
	return &MockMSSQLClient{
		GetBackupSizeFunc: func(_ context.Context, _ string) (int64, error) {
			return sizeBytes, nil
		},
	}
}

// NewMockMSSQLClientWithError создаёт MockMSSQLClient, который возвращает ошибку
// для всех операций. Полезно для тестирования error-paths.
func NewMockMSSQLClientWithError(err error) *MockMSSQLClient {
	return &MockMSSQLClient{
		ConnectFunc: func(_ context.Context) error {
			return err
		},
		CloseFunc: func() error {
			return err
		},
		PingFunc: func(_ context.Context) error {
			return err
		},
		RestoreFunc: func(_ context.Context, _ mssql.RestoreOptions) error {
			return err
		},
		GetRestoreStatsFunc: func(_ context.Context, _ mssql.StatsOptions) (*mssql.RestoreStats, error) {
			return nil, err
		},
		GetBackupSizeFunc: func(_ context.Context, _ string) (int64, error) {
			return 0, err
		},
	}
}
