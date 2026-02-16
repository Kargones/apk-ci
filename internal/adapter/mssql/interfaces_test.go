package mssql_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/mssql"
	"github.com/Kargones/apk-ci/internal/adapter/mssql/mssqltest"
)

// TestMockImplementsAllInterfaces проверяет, что mock реализует все интерфейсы.
// Compile-time проверки в mssqltest/mock.go уже гарантируют это,
// но runtime тест подтверждает корректность через type assertions.
func TestMockImplementsAllInterfaces(t *testing.T) {
	mock := mssqltest.NewMockMSSQLClient()

	// Проверяем, что mock реализует все интерфейсы через type assertion
	var client mssql.Client = mock
	if client == nil {
		t.Error("mock должен реализовывать интерфейс Client")
	}

	var connector mssql.DatabaseConnector = mock
	if connector == nil {
		t.Error("mock должен реализовывать интерфейс DatabaseConnector")
	}

	var restorer mssql.DatabaseRestorer = mock
	if restorer == nil {
		t.Error("mock должен реализовывать интерфейс DatabaseRestorer")
	}

	var provider mssql.BackupInfoProvider = mock
	if provider == nil {
		t.Error("mock должен реализовывать интерфейс BackupInfoProvider")
	}
}

// TestMockWithCustomFunctions проверяет, что mock с кастомными функциями
// возвращает ожидаемые данные.
func TestMockWithCustomFunctions(t *testing.T) {
	ctx := context.Background()

	t.Run("Connect с кастомной функцией", func(t *testing.T) {
		expectedErr := errors.New("connection refused")
		mock := &mssqltest.MockMSSQLClient{
			ConnectFunc: func(_ context.Context) error {
				return expectedErr
			},
		}

		err := mock.Connect(ctx)
		if !errors.Is(err, expectedErr) {
			t.Errorf("Connect() вернул %v, ожидалось %v", err, expectedErr)
		}
	})

	t.Run("Ping с кастомной функцией", func(t *testing.T) {
		expectedErr := errors.New("server unreachable")
		mock := &mssqltest.MockMSSQLClient{
			PingFunc: func(_ context.Context) error {
				return expectedErr
			},
		}

		err := mock.Ping(ctx)
		if !errors.Is(err, expectedErr) {
			t.Errorf("Ping() вернул %v, ожидалось %v", err, expectedErr)
		}
	})

	t.Run("Close с кастомной функцией", func(t *testing.T) {
		expectedErr := errors.New("close failed")
		mock := &mssqltest.MockMSSQLClient{
			CloseFunc: func() error {
				return expectedErr
			},
		}

		err := mock.Close()
		if !errors.Is(err, expectedErr) {
			t.Errorf("Close() вернул %v, ожидалось %v", err, expectedErr)
		}
	})

	t.Run("Restore с кастомной функцией", func(t *testing.T) {
		expectedErr := errors.New("restore failed")
		var capturedOpts mssql.RestoreOptions

		mock := &mssqltest.MockMSSQLClient{
			RestoreFunc: func(_ context.Context, opts mssql.RestoreOptions) error {
				capturedOpts = opts
				return expectedErr
			},
		}

		opts := mssql.RestoreOptions{
			SrcServer: "src-server",
			SrcDB:     "src-db",
			DstServer: "dst-server",
			DstDB:     "dst-db",
			Timeout:   5 * time.Minute,
		}

		err := mock.Restore(ctx, opts)
		if !errors.Is(err, expectedErr) {
			t.Errorf("Restore() вернул %v, ожидалось %v", err, expectedErr)
		}
		if capturedOpts.SrcDB != "src-db" {
			t.Errorf("Restore() получил SrcDB=%s, ожидалось src-db", capturedOpts.SrcDB)
		}
		if capturedOpts.DstDB != "dst-db" {
			t.Errorf("Restore() получил DstDB=%s, ожидалось dst-db", capturedOpts.DstDB)
		}
	})

	t.Run("GetRestoreStats с кастомной функцией", func(t *testing.T) {
		expectedStats := &mssql.RestoreStats{
			AvgRestoreTimeSec: 300,
			MaxRestoreTimeSec: 600,
			HasData:           true,
		}

		mock := &mssqltest.MockMSSQLClient{
			GetRestoreStatsFunc: func(_ context.Context, _ mssql.StatsOptions) (*mssql.RestoreStats, error) {
				return expectedStats, nil
			},
		}

		stats, err := mock.GetRestoreStats(ctx, mssql.StatsOptions{SrcDB: "test-db"})
		if err != nil {
			t.Errorf("GetRestoreStats() вернул ошибку: %v", err)
		}
		if stats.AvgRestoreTimeSec != 300 {
			t.Errorf("GetRestoreStats() вернул AvgRestoreTimeSec=%d, ожидалось 300", stats.AvgRestoreTimeSec)
		}
		if stats.MaxRestoreTimeSec != 600 {
			t.Errorf("GetRestoreStats() вернул MaxRestoreTimeSec=%d, ожидалось 600", stats.MaxRestoreTimeSec)
		}
		if !stats.HasData {
			t.Error("GetRestoreStats() вернул HasData=false, ожидалось true")
		}
	})

	t.Run("GetBackupSize с кастомной функцией", func(t *testing.T) {
		expectedSize := int64(1024 * 1024 * 500) // 500 MB
		var capturedDB string

		mock := &mssqltest.MockMSSQLClient{
			GetBackupSizeFunc: func(_ context.Context, database string) (int64, error) {
				capturedDB = database
				return expectedSize, nil
			},
		}

		size, err := mock.GetBackupSize(ctx, "production-db")
		if err != nil {
			t.Errorf("GetBackupSize() вернул ошибку: %v", err)
		}
		if size != expectedSize {
			t.Errorf("GetBackupSize() вернул %d, ожидалось %d", size, expectedSize)
		}
		if capturedDB != "production-db" {
			t.Errorf("GetBackupSize() получил database=%s, ожидалось production-db", capturedDB)
		}
	})
}

// TestMockWithNilFunctions проверяет, что mock с nil-функциями
// возвращает дефолтные значения.
func TestMockWithNilFunctions(t *testing.T) {
	ctx := context.Background()
	mock := mssqltest.NewMockMSSQLClient()

	t.Run("Connect с nil функцией возвращает nil", func(t *testing.T) {
		err := mock.Connect(ctx)
		if err != nil {
			t.Errorf("Connect() вернул %v, ожидалось nil", err)
		}
	})

	t.Run("Close с nil функцией возвращает nil", func(t *testing.T) {
		err := mock.Close()
		if err != nil {
			t.Errorf("Close() вернул %v, ожидалось nil", err)
		}
	})

	t.Run("Ping с nil функцией возвращает nil", func(t *testing.T) {
		err := mock.Ping(ctx)
		if err != nil {
			t.Errorf("Ping() вернул %v, ожидалось nil", err)
		}
	})

	t.Run("Restore с nil функцией возвращает nil", func(t *testing.T) {
		err := mock.Restore(ctx, mssql.RestoreOptions{})
		if err != nil {
			t.Errorf("Restore() вернул %v, ожидалось nil", err)
		}
	})

	t.Run("GetRestoreStats с nil функцией возвращает реалистичные данные", func(t *testing.T) {
		stats, err := mock.GetRestoreStats(ctx, mssql.StatsOptions{})
		if err != nil {
			t.Errorf("GetRestoreStats() вернул ошибку: %v", err)
		}
		if stats == nil {
			t.Fatal("GetRestoreStats() вернул nil stats")
		}
		// Дефолтные реалистичные значения: 180 сек avg, 600 сек max, HasData=true
		if stats.AvgRestoreTimeSec != 180 {
			t.Errorf("GetRestoreStats() вернул AvgRestoreTimeSec=%d, ожидалось 180", stats.AvgRestoreTimeSec)
		}
		if stats.MaxRestoreTimeSec != 600 {
			t.Errorf("GetRestoreStats() вернул MaxRestoreTimeSec=%d, ожидалось 600", stats.MaxRestoreTimeSec)
		}
		if !stats.HasData {
			t.Error("GetRestoreStats() вернул HasData=false, ожидалось true")
		}
	})

	t.Run("GetBackupSize с nil функцией возвращает реалистичный размер", func(t *testing.T) {
		size, err := mock.GetBackupSize(ctx, "any-db")
		if err != nil {
			t.Errorf("GetBackupSize() вернул ошибку: %v", err)
		}
		// Дефолтный реалистичный размер: 500 MB
		expectedSize := int64(500 * 1024 * 1024)
		if size != expectedSize {
			t.Errorf("GetBackupSize() вернул %d, ожидалось %d", size, expectedSize)
		}
	})
}

// TestErrorCodes проверяет наличие и корректность кодов ошибок.
func TestErrorCodes(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "ErrMSSQLConnect",
			code:     mssql.ErrMSSQLConnect,
			expected: "MSSQL.CONNECT_FAILED",
		},
		{
			name:     "ErrMSSQLRestore",
			code:     mssql.ErrMSSQLRestore,
			expected: "MSSQL.RESTORE_FAILED",
		},
		{
			name:     "ErrMSSQLQuery",
			code:     mssql.ErrMSSQLQuery,
			expected: "MSSQL.QUERY_FAILED",
		},
		{
			name:     "ErrMSSQLTimeout",
			code:     mssql.ErrMSSQLTimeout,
			expected: "MSSQL.TIMEOUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code != tt.expected {
				t.Errorf("код ошибки %s = %q, ожидалось %q", tt.name, tt.code, tt.expected)
			}
		})
	}
}

// TestHelperConstructors проверяет вспомогательные конструкторы mock.
func TestHelperConstructors(t *testing.T) {
	ctx := context.Background()

	t.Run("NewMockMSSQLClientWithRestoreStats", func(t *testing.T) {
		mock := mssqltest.NewMockMSSQLClientWithRestoreStats(120, 300, true)
		stats, err := mock.GetRestoreStats(ctx, mssql.StatsOptions{})
		if err != nil {
			t.Errorf("GetRestoreStats() вернул ошибку: %v", err)
		}
		if stats.AvgRestoreTimeSec != 120 {
			t.Errorf("AvgRestoreTimeSec = %d, ожидалось 120", stats.AvgRestoreTimeSec)
		}
		if stats.MaxRestoreTimeSec != 300 {
			t.Errorf("MaxRestoreTimeSec = %d, ожидалось 300", stats.MaxRestoreTimeSec)
		}
		if !stats.HasData {
			t.Error("HasData = false, ожидалось true")
		}
	})

	t.Run("NewMockMSSQLClientWithBackupSize", func(t *testing.T) {
		expectedSize := int64(1073741824) // 1 GB
		mock := mssqltest.NewMockMSSQLClientWithBackupSize(expectedSize)
		size, err := mock.GetBackupSize(ctx, "test-db")
		if err != nil {
			t.Errorf("GetBackupSize() вернул ошибку: %v", err)
		}
		if size != expectedSize {
			t.Errorf("size = %d, ожидалось %d", size, expectedSize)
		}
	})

	t.Run("NewMockMSSQLClientWithError", func(t *testing.T) {
		expectedErr := errors.New("database unavailable")
		mock := mssqltest.NewMockMSSQLClientWithError(expectedErr)

		// Проверяем что все методы возвращают ошибку
		if err := mock.Connect(ctx); !errors.Is(err, expectedErr) {
			t.Errorf("Connect() вернул %v, ожидалось %v", err, expectedErr)
		}
		if err := mock.Close(); !errors.Is(err, expectedErr) {
			t.Errorf("Close() вернул %v, ожидалось %v", err, expectedErr)
		}
		if err := mock.Ping(ctx); !errors.Is(err, expectedErr) {
			t.Errorf("Ping() вернул %v, ожидалось %v", err, expectedErr)
		}
		if err := mock.Restore(ctx, mssql.RestoreOptions{}); !errors.Is(err, expectedErr) {
			t.Errorf("Restore() вернул %v, ожидалось %v", err, expectedErr)
		}
		if _, err := mock.GetRestoreStats(ctx, mssql.StatsOptions{}); !errors.Is(err, expectedErr) {
			t.Errorf("GetRestoreStats() вернул %v, ожидалось %v", err, expectedErr)
		}
		if _, err := mock.GetBackupSize(ctx, "test-db"); !errors.Is(err, expectedErr) {
			t.Errorf("GetBackupSize() вернул %v, ожидалось %v", err, expectedErr)
		}
	})
}

// TestRestoreOptionsFields проверяет, что все поля RestoreOptions доступны.
func TestRestoreOptionsFields(t *testing.T) {
	opts := mssql.RestoreOptions{
		Description:   "Тестовое восстановление",
		TimeToRestore: "2026-01-01 12:00:00",
		User:          "admin",
		SrcServer:     "src-server.local",
		SrcDB:         "source_db",
		DstServer:     "dst-server.local",
		DstDB:         "destination_db",
		Timeout:       10 * time.Minute,
	}

	if opts.Description != "Тестовое восстановление" {
		t.Errorf("Description = %q, ожидалось %q", opts.Description, "Тестовое восстановление")
	}
	if opts.Timeout != 10*time.Minute {
		t.Errorf("Timeout = %v, ожидалось %v", opts.Timeout, 10*time.Minute)
	}
}

// TestStatsOptionsFields проверяет, что все поля StatsOptions доступны.
func TestStatsOptionsFields(t *testing.T) {
	opts := mssql.StatsOptions{
		SrcDB:           "source_db",
		DstServer:       "dst-server.local",
		TimeToStatistic: "30 days",
	}

	if opts.SrcDB != "source_db" {
		t.Errorf("SrcDB = %q, ожидалось %q", opts.SrcDB, "source_db")
	}
	if opts.TimeToStatistic != "30 days" {
		t.Errorf("TimeToStatistic = %q, ожидалось %q", opts.TimeToStatistic, "30 days")
	}
}

// TestBackupInfoFields проверяет, что все поля BackupInfo доступны.
func TestBackupInfoFields(t *testing.T) {
	info := mssql.BackupInfo{
		SizeBytes: 1024 * 1024 * 100,
		Database:  "production_db",
	}

	if info.SizeBytes != 1024*1024*100 {
		t.Errorf("SizeBytes = %d, ожидалось %d", info.SizeBytes, 1024*1024*100)
	}
	if info.Database != "production_db" {
		t.Errorf("Database = %q, ожидалось %q", info.Database, "production_db")
	}
}
