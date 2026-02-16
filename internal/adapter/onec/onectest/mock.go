// Package onectest предоставляет mock-реализации интерфейсов пакета onec для тестирования.
package onectest

import (
	"context"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/onec"
)

// MockDatabaseUpdater — mock-реализация интерфейса DatabaseUpdater для тестирования.
type MockDatabaseUpdater struct {
	// UpdateDBCfgFunc — функция, вызываемая при UpdateDBCfg.
	// Если nil, возвращается успешный результат по умолчанию.
	UpdateDBCfgFunc func(ctx context.Context, opts onec.UpdateOptions) (*onec.UpdateResult, error)
	// UpdateDBCfgCallCount — количество вызовов UpdateDBCfg
	UpdateDBCfgCallCount int
	// LastUpdateOptions — последние переданные опции
	LastUpdateOptions onec.UpdateOptions
}

// Compile-time проверка интерфейса.
var _ onec.DatabaseUpdater = (*MockDatabaseUpdater)(nil)

// NewMockDatabaseUpdater создаёт mock с успешным поведением по умолчанию.
func NewMockDatabaseUpdater() *MockDatabaseUpdater {
	return &MockDatabaseUpdater{
		UpdateDBCfgFunc: func(ctx context.Context, opts onec.UpdateOptions) (*onec.UpdateResult, error) {
			return &onec.UpdateResult{
				Success:    true,
				Messages:   []string{"Обновление конфигурации успешно завершено"},
				DurationMs: 1000,
			}, nil
		},
	}
}

// UpdateDBCfg вызывает mock-функцию или возвращает результат по умолчанию.
func (m *MockDatabaseUpdater) UpdateDBCfg(ctx context.Context, opts onec.UpdateOptions) (*onec.UpdateResult, error) {
	m.UpdateDBCfgCallCount++
	m.LastUpdateOptions = opts

	if m.UpdateDBCfgFunc != nil {
		return m.UpdateDBCfgFunc(ctx, opts)
	}

	// Результат по умолчанию
	return &onec.UpdateResult{
		Success:    true,
		Messages:   []string{"Обновление конфигурации успешно завершено"},
		DurationMs: 1000,
	}, nil
}

// MockTempDatabaseCreator — mock-реализация интерфейса TempDatabaseCreator для тестирования.
type MockTempDatabaseCreator struct {
	// CreateTempDBFunc — функция, вызываемая при CreateTempDB.
	// Если nil, возвращается успешный результат по умолчанию.
	CreateTempDBFunc func(ctx context.Context, opts onec.CreateTempDBOptions) (*onec.TempDBResult, error)
	// CreateTempDBCallCount — количество вызовов CreateTempDB
	CreateTempDBCallCount int
	// LastCreateTempDBOptions — последние переданные опции
	LastCreateTempDBOptions onec.CreateTempDBOptions
}

// Compile-time проверка интерфейса.
var _ onec.TempDatabaseCreator = (*MockTempDatabaseCreator)(nil)

// NewMockTempDatabaseCreator создаёт mock с успешным поведением по умолчанию.
func NewMockTempDatabaseCreator() *MockTempDatabaseCreator {
	return &MockTempDatabaseCreator{
		CreateTempDBFunc: func(ctx context.Context, opts onec.CreateTempDBOptions) (*onec.TempDBResult, error) {
			return &onec.TempDBResult{
				ConnectString: "/F " + opts.DbPath,
				DbPath:        opts.DbPath,
				Extensions:    opts.Extensions,
				CreatedAt:     time.Now(),
				DurationMs:    100,
			}, nil
		},
	}
}

// CreateTempDB вызывает mock-функцию или возвращает результат по умолчанию.
func (m *MockTempDatabaseCreator) CreateTempDB(ctx context.Context, opts onec.CreateTempDBOptions) (*onec.TempDBResult, error) {
	m.CreateTempDBCallCount++
	m.LastCreateTempDBOptions = opts

	if m.CreateTempDBFunc != nil {
		return m.CreateTempDBFunc(ctx, opts)
	}

	// Результат по умолчанию
	return &onec.TempDBResult{
		ConnectString: "/F " + opts.DbPath,
		DbPath:        opts.DbPath,
		Extensions:    opts.Extensions,
		CreatedAt:     time.Now(),
		DurationMs:    100,
	}, nil
}
