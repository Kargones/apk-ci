package dbrestore

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/DATA-DOG/go-sqlmock"
)

func TestNew(t *testing.T) {
	dbRestore := New()
	if dbRestore == nil {
		t.Fatal("Expected non-nil DBRestore")
	}
	// New() создает пустую структуру без значений по умолчанию
	if dbRestore.Server != "" {
		t.Errorf("Expected empty server, got %s", dbRestore.Server)
	}
	if dbRestore.Port != 0 {
		t.Errorf("Expected port 0, got %d", dbRestore.Port)
	}
	if dbRestore.Database != "" {
		t.Errorf("Expected empty database, got %s", dbRestore.Database)
	}
	if dbRestore.Timeout != 0 {
		t.Errorf("Expected timeout 0, got %v", dbRestore.Timeout)
	}
	if dbRestore.AutoTimeOut {
		t.Error("Expected AutoTimeOut to be false")
	}
}

func TestSetupDBRestoreDefaults(t *testing.T) {
	dbR := &DBRestore{}
	setupDBRestoreDefaults(dbR)
	
	if dbR.Server != DefaultServer {
		t.Errorf("Expected server %s, got %s", DefaultServer, dbR.Server)
	}
	if dbR.Port != DefaultPort {
		t.Errorf("Expected port %d, got %d", DefaultPort, dbR.Port)
	}
	if dbR.Database != "master" { //nolint:goconst // test value
		t.Errorf("Expected database master, got %s", dbR.Database)
	}
	// setupDBRestoreDefaults устанавливает AutoTimeOut в true только если Timeout == 0
	if dbR.Timeout != 0 {
		t.Errorf("Expected timeout 0, got %v", dbR.Timeout)
	}
	if !dbR.AutoTimeOut {
		t.Error("Expected AutoTimeOut to be true")
	}
}

func TestDBRestore_Init(t *testing.T) {
	tests := []struct {
		name     string
		yamlData string
		wantErr  bool
	}{
		{
			name: "valid yaml",
			yamlData: `tempdbrestore:
  server: test-server
  port: 1433
  user: testuser
  password: testpass
  database: testdb
  timeout: 1h`,
			wantErr: false,
		},
		{
			name:     "invalid yaml",
			yamlData: "invalid: yaml: content: [",
			wantErr:  true,
		},
		{
			name:     "missing section",
			yamlData: "other: value",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbR := New()
			err := dbR.Init([]byte(tt.yamlData))
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.name == "valid yaml" {
					if dbR.Server != "test-server" {
						t.Errorf("Expected server test-server, got %s", dbR.Server)
					}
					if dbR.Port != 1433 {
						t.Errorf("Expected port 1433, got %d", dbR.Port)
					}
					if dbR.User != "testuser" {
						t.Errorf("Expected user testuser, got %s", dbR.User)
					}
					if dbR.Password != "testpass" {
						t.Errorf("Expected password testpass, got %s", dbR.Password)
					}
					if dbR.Database != "testdb" {
						t.Errorf("Expected database testdb, got %s", dbR.Database)
					}
					if dbR.Timeout != time.Hour {
						t.Errorf("Expected timeout 1h, got %v", dbR.Timeout)
					}
				}
			}
		})
	}
}

func TestDBRestore_Connect(t *testing.T) {
	// Тест подключения с недопустимыми параметрами
	dbR := &DBRestore{
		Server:   "invalid-server",
		Port:     9999,
		User:     "invalid-user",
		Password: "invalid-password",
		Database: "invalid-db",
	}

	ctx := context.Background()
	err := dbR.Connect(ctx)
	// Ожидаем ошибку подключения к несуществующему серверу
	if err == nil {
		t.Error("Expected error but got none")
	}
	if err != nil && err.Error() == "" {
		t.Error("Expected error message")
	}
}

func TestDBRestore_Close(t *testing.T) {
	tests := []struct {
		name   string
		setupDB func() *sql.DB
		wantErr bool
	}{
		{
			name: "nil database",
			setupDB: func() *sql.DB {
				return nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbR := &DBRestore{
				Db: tt.setupDB(),
			}
			err := dbR.Close()
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestFindProductionDatabase(t *testing.T) {
	// Тест с nil конфигурацией
	result := FindProductionDatabase(nil, "test-db")
	if result != "" {
		t.Errorf("Expected empty string for nil config, got %s", result)
	}

	// Тест с пустой конфигурацией
	emptyConfig := &config.ProjectConfig{}
	result = FindProductionDatabase(emptyConfig, "test-db")
	if result != "" {
		t.Errorf("Expected empty string for empty config, got %s", result)
	}
}

func TestDBRestore_Restore(t *testing.T) {
	// Тест восстановления без подключения к БД
	dbR := &DBRestore{
		SrcServer: "test-src",
		SrcDB:     "test-src-db",
		DstServer: "test-dst",
		DstDB:     "test-dst-db",
	}

	ctx := context.Background()
	err := dbR.Restore(ctx)
	// Ожидаем ошибку, так как нет подключения к БД
	if err == nil {
		t.Error("Expected error but got none")
	}
}

func TestDBRestore_GetRestoreStats(t *testing.T) {
	// Тест получения статистики без подключения к БД
	dbR := &DBRestore{
		SrcServer: "test-src",
		SrcDB:     "test-src-db",
	}

	ctx := context.Background()
	stats, err := dbR.GetRestoreStats(ctx)
	// Ожидаем ошибку, так как нет подключения к БД
	if err == nil {
		t.Error("Expected error but got none")
	}
	if stats != nil {
		t.Error("Expected nil stats")
	}
}

// TestLoadDBRestoreConfig тестирует загрузку конфигурации DB restore
func TestLoadDBRestoreConfig(t *testing.T) {
	logger := slog.Default()

	// Создаем минимальную конфигурацию для тестирования
	cfg := &config.Config{
		DbConfig: map[string]*config.DatabaseInfo{
			"testdb": {
				OneServer: "localhost",
				Prod:      false,
				DbServer:  "localhost",
			},
		},
	}

	t.Run("missing project config", func(t *testing.T) {
		dbRestore, err := LoadDBRestoreConfig(logger, cfg, "testdb")
		// Ожидаем ошибку из-за неполной конфигурации
		if err == nil {
			t.Error("Expected error due to incomplete configuration")
		}
		_ = dbRestore
	})

	t.Run("nil config", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				// Ожидаем panic для nil config
			}
		}()
		dbRestore, err := LoadDBRestoreConfig(logger, nil, "testdb")
		if err == nil {
			t.Error("Expected error for nil config")
		}
		_ = dbRestore
	})
}

// TestNewFromConfig тестирует создание DBRestore из конфигурации
func TestNewFromConfig(t *testing.T) {
	logger := slog.Default()

	// Создаем минимальную конфигурацию для тестирования
	cfg := &config.Config{
		DbConfig: map[string]*config.DatabaseInfo{
			"testdb": {
				OneServer: "localhost",
				Prod:      false,
				DbServer:  "localhost",
			},
		},
	}

	t.Run("missing project config", func(t *testing.T) {
		dbRestore, err := NewFromConfig(logger, cfg, "testdb")
		// Ожидаем ошибку из-за неполной конфигурации
		if err == nil {
			t.Error("Expected error due to incomplete configuration")
		}
		_ = dbRestore
	})

	t.Run("nil config", func(t *testing.T) {
		dbRestore, err := NewFromConfig(logger, nil, "testdb")
		if err == nil {
			t.Error("Expected error for nil config")
		}
		_ = dbRestore
	})
}

// TestFindProductionDatabaseExtended расширенные тесты для поиска продуктивной БД
func TestFindProductionDatabaseExtended(t *testing.T) {
	// Создаем реалистичную конфигурацию для тестирования с правильной структурой
	cfg := &config.ProjectConfig{
		Prod: map[string]struct {
			DbName     string                 `yaml:"dbName"`
			AddDisable []string               `yaml:"add-disable"`
			Related    map[string]interface{} `yaml:"related"`
		}{
			"prod-db": {
				DbName: "prod-db",
				Related: map[string]interface{}{
					"test-db": "test relation",
				},
			},
		},
	}

	t.Run("find production for test db", func(t *testing.T) {
		result := FindProductionDatabase(cfg, "test-db")
		if result != "prod-db" { //nolint:goconst // test value
			t.Errorf("Expected 'prod-db', got '%s'", result)
		}
	})

	t.Run("find production for unknown db", func(t *testing.T) {
		result := FindProductionDatabase(cfg, "unknown-db")
		if result != "" {
			t.Errorf("Expected empty string for unknown db, got '%s'", result)
		}
	})

	t.Run("already production db", func(t *testing.T) {
		result := FindProductionDatabase(cfg, "prod-db")
		if result != "prod-db" {
			t.Errorf("Expected 'prod-db' for production db itself, got '%s'", result)
		}
	})
}

// TestDBRestore_RestoreExtended расширенные тесты для восстановления БД
func TestDBRestore_RestoreExtended(t *testing.T) {
	ctx := context.Background()

	t.Run("missing source server", func(t *testing.T) {
		dbR := &DBRestore{
			SrcDB:     "test-src-db",
			DstServer: "test-dst",
			DstDB:     "test-dst-db",
		}

		err := dbR.Restore(ctx)
		if err == nil {
			t.Error("Expected error for missing source server")
		}
	})

	t.Run("missing destination server", func(t *testing.T) {
		dbR := &DBRestore{
			SrcServer: "test-src",
			SrcDB:     "test-src-db",
			DstDB:     "test-dst-db",
		}

		err := dbR.Restore(ctx)
		if err == nil {
			t.Error("Expected error for missing destination server")
		}
	})

	t.Run("missing source database", func(t *testing.T) {
		dbR := &DBRestore{
			SrcServer: "test-src",
			DstServer: "test-dst",
			DstDB:     "test-dst-db",
		}

		err := dbR.Restore(ctx)
		if err == nil {
			t.Error("Expected error for missing source database")
		}
	})

	t.Run("missing destination database", func(t *testing.T) {
		dbR := &DBRestore{
			SrcServer: "test-src",
			SrcDB:     "test-src-db",
			DstServer: "test-dst",
		}

		err := dbR.Restore(ctx)
		if err == nil {
			t.Error("Expected error for missing destination database")
		}
	})
}

// TestDBRestore_GetRestoreStatsExtended расширенные тесты для получения статистики
func TestDBRestore_GetRestoreStatsExtended(t *testing.T) {
	ctx := context.Background()

	t.Run("missing source server", func(t *testing.T) {
		dbR := &DBRestore{
			SrcDB: "test-src-db",
		}

		stats, err := dbR.GetRestoreStats(ctx)
		if err == nil {
			t.Error("Expected error for missing source server")
		}
		if stats != nil {
			t.Error("Expected nil stats")
		}
	})

	t.Run("missing source database", func(t *testing.T) {
		dbR := &DBRestore{
			SrcServer: "test-src",
		}

		stats, err := dbR.GetRestoreStats(ctx)
		if err == nil {
			t.Error("Expected error for missing source database")
		}
		if stats != nil {
			t.Error("Expected nil stats")
		}
	})
}

// TestDBRestore_CloseExtended расширенные тесты для закрытия соединения
func TestDBRestore_CloseExtended(t *testing.T) {
	t.Run("close nil database", func(t *testing.T) {
		dbR := &DBRestore{
			Db: nil,
		}
		err := dbR.Close()
		if err != nil {
			t.Errorf("Unexpected error for nil database: %v", err)
		}
	})
}

// TestDBRestore_ConnectExtended расширенные тесты для подключения
func TestDBRestore_ConnectExtended(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid server address", func(t *testing.T) {
		dbR := &DBRestore{
			Server:   "127.0.0.1", // Локальный адрес, который должен быть недоступен для SQL Server
			Port:     1433,
			User:     "sa",
			Password: "invalid",
			Database: "master",
			Timeout:  time.Second * 5,
		}

		err := dbR.Connect(ctx)
		if err == nil {
			// Если соединение успешно, закрываем его
			_ = dbR.Close()
		}
		// Не проверяем ошибку, так как поведение зависит от окружения
	})

	t.Run("empty server", func(t *testing.T) {
		dbR := &DBRestore{
			Server:   "",
			Port:     1433,
			User:     "sa",
			Password: "password",
			Database: "master",
		}

		err := dbR.Connect(ctx)
		if err == nil {
			t.Error("Expected error for empty server")
		}
	})
}

// TestSetupDBRestoreDefaultsExtended расширенные тесты для setupDBRestoreDefaults
func TestSetupDBRestoreDefaultsExtended(t *testing.T) {
	t.Run("all fields empty", func(t *testing.T) {
		dbR := &DBRestore{}
		setupDBRestoreDefaults(dbR)

		if dbR.User != "gitops" {
			t.Errorf("Expected user gitops, got %s", dbR.User)
		}
		if dbR.Server != DefaultServer {
			t.Errorf("Expected server %s, got %s", DefaultServer, dbR.Server)
		}
		if dbR.Port != DefaultPort {
			t.Errorf("Expected port %d, got %d", DefaultPort, dbR.Port)
		}
		if dbR.Database != "master" {
			t.Errorf("Expected database master, got %s", dbR.Database)
		}
		if dbR.Description != "gitops db restore task" {
			t.Errorf("Expected description 'gitops db restore task', got %s", dbR.Description)
		}
		if dbR.TimeToRestore == "" {
			t.Error("Expected TimeToRestore to be set")
		}
		if dbR.TimeToStatistic == "" {
			t.Error("Expected TimeToStatistic to be set")
		}
		if !dbR.AutoTimeOut {
			t.Error("Expected AutoTimeOut to be true when Timeout is 0")
		}
	})

	t.Run("partial fields set", func(t *testing.T) {
		dbR := &DBRestore{
			User:        "customuser",
			Server:      "customserver",
			Port:        1234,
			Database:    "customdb",
			Description: "custom description",
			Timeout:     time.Hour,
		}
		setupDBRestoreDefaults(dbR)

		// Проверяем, что предустановленные значения не изменились
		if dbR.User != "customuser" {
			t.Errorf("Expected user customuser, got %s", dbR.User)
		}
		if dbR.Server != "customserver" {
			t.Errorf("Expected server customserver, got %s", dbR.Server)
		}
		if dbR.Port != 1234 {
			t.Errorf("Expected port 1234, got %d", dbR.Port)
		}
		if dbR.Database != "customdb" {
			t.Errorf("Expected database customdb, got %s", dbR.Database)
		}
		if dbR.Description != "custom description" {
			t.Errorf("Expected description 'custom description', got %s", dbR.Description)
		}
		// AutoTimeOut не должен быть установлен, так как Timeout не равен 0
		if dbR.AutoTimeOut {
			t.Error("Expected AutoTimeOut to be false when Timeout is not 0")
		}
	})
}

// TestNewFromConfigExtended расширенные тесты для NewFromConfig
func TestNewFromConfigExtended(t *testing.T) {
	logger := slog.Default()

	t.Run("complete valid config", func(t *testing.T) {
		cfg := &config.Config{
			AppConfig: &config.AppConfig{
				Dbrestore: struct {
					Database    string `yaml:"database"`
					Timeout     string `yaml:"timeout"`
					Autotimeout bool   `yaml:"autotimeout"`
				}{
					Database:    "testdb",
					Timeout:     "30m",
					Autotimeout: true,
				},
				Users: struct {
					Rac        string `yaml:"rac"`
					Db         string `yaml:"db"`
					Mssql      string `yaml:"mssql"`
					StoreAdmin string `yaml:"storeAdmin"`
				}{
					Mssql: "testuser",
				},
			},
			SecretConfig: &config.SecretConfig{
				Passwords: struct {
					Rac                string `yaml:"rac"`
					Db                 string `yaml:"db"`
					Mssql              string `yaml:"mssql"`
					StoreAdminPassword string `yaml:"storeAdminPassword"`
					Smb                string `yaml:"smb"`
				}{
					Mssql: "testpass",
				},
			},
			ProjectConfig: &config.ProjectConfig{
				Prod: map[string]struct {
					DbName     string                 `yaml:"dbName"`
					AddDisable []string               `yaml:"add-disable"`
					Related    map[string]interface{} `yaml:"related"`
				}{
					"prod-db": {
						DbName: "prod-db",
						Related: map[string]interface{}{
							"test-db": "test relation",
						},
					},
				},
			},
			DbConfig: map[string]*config.DatabaseInfo{
				"prod-db": {
					DbServer: "prod-server",
				},
				"test-db": {
					DbServer: "test-server",
				},
			},
		}

		dbRestore, err := NewFromConfig(logger, cfg, "test-db")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if dbRestore.Database != "testdb" {
			t.Errorf("Expected database testdb, got %s", dbRestore.Database)
		}
		if dbRestore.User != "testuser" {
			t.Errorf("Expected user testuser, got %s", dbRestore.User)
		}
		if dbRestore.Password != "testpass" {
			t.Errorf("Expected password testpass, got %s", dbRestore.Password)
		}
		if dbRestore.Timeout != 30*time.Minute {
			t.Errorf("Expected timeout 30m, got %v", dbRestore.Timeout)
		}
		if !dbRestore.AutoTimeOut {
			t.Error("Expected AutoTimeOut to be true")
		}
		if dbRestore.SrcServer != "prod-server" {
			t.Errorf("Expected SrcServer prod-server, got %s", dbRestore.SrcServer)
		}
		if dbRestore.SrcDB != "prod-db" {
			t.Errorf("Expected SrcDB prod-db, got %s", dbRestore.SrcDB)
		}
		if dbRestore.DstServer != "test-server" {
			t.Errorf("Expected DstServer test-server, got %s", dbRestore.DstServer)
		}
		if dbRestore.DstDB != "test-db" {
			t.Errorf("Expected DstDB test-db, got %s", dbRestore.DstDB)
		}
	})

	t.Run("missing destination database", func(t *testing.T) {
		cfg := &config.Config{
			ProjectConfig: &config.ProjectConfig{
				Prod: map[string]struct {
					DbName     string                 `yaml:"dbName"`
					AddDisable []string               `yaml:"add-disable"`
					Related    map[string]interface{} `yaml:"related"`
				}{
					"prod-db": {
						DbName: "prod-db",
						Related: map[string]interface{}{
							"test-db": "test relation",
						},
					},
				},
			},
			DbConfig: map[string]*config.DatabaseInfo{
				"prod-db": {
					DbServer: "prod-server",
				},
			},
		}

		_, err := NewFromConfig(logger, cfg, "test-db")
		if err == nil {
			t.Error("Expected error for missing destination database")
		}
		if err != nil && err.Error() != "база данных test-db не найдена в DbConfig" {
			t.Errorf("Expected specific error message, got: %v", err)
		}
	})

	t.Run("missing production database", func(t *testing.T) {
		cfg := &config.Config{
			ProjectConfig: &config.ProjectConfig{
				Prod: map[string]struct {
					DbName     string                 `yaml:"dbName"`
					AddDisable []string               `yaml:"add-disable"`
					Related    map[string]interface{} `yaml:"related"`
				}{},
			},
			DbConfig: map[string]*config.DatabaseInfo{
				"test-db": {
					DbServer: "test-server",
				},
			},
		}

		_, err := NewFromConfig(logger, cfg, "test-db")
		if err == nil {
			t.Error("Expected error for missing production database")
		}
	})

	t.Run("invalid timeout format", func(t *testing.T) {
		cfg := &config.Config{
			AppConfig: &config.AppConfig{
				Dbrestore: struct {
					Database    string `yaml:"database"`
					Timeout     string `yaml:"timeout"`
					Autotimeout bool   `yaml:"autotimeout"`
				}{
					Timeout: "invalid-timeout",
				},
			},
		}

		dbRestore, err := NewFromConfig(logger, cfg, "")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Timeout должен остаться по умолчанию (30 секунд)
		if dbRestore.Timeout != 30*time.Second {
			t.Errorf("Expected default timeout 30s, got %v", dbRestore.Timeout)
		}
	})
}

// TestLoadDBRestoreConfig_ErrorHandling тестирует обработку ошибок в LoadDBRestoreConfig
func TestLoadDBRestoreConfig_ErrorHandling(t *testing.T) {
	logger := slog.Default()

	cfg := &config.Config{
		AppConfig: &config.AppConfig{
			Dbrestore: struct {
				Database    string `yaml:"database"`
				Timeout     string `yaml:"timeout"`
				Autotimeout bool   `yaml:"autotimeout"`
			}{
				Database:    "testdb",
				Timeout:     "1h",
				Autotimeout: true,
			},
			Users: struct {
				Rac        string `yaml:"rac"`
				Db         string `yaml:"db"`
				Mssql      string `yaml:"mssql"`
				StoreAdmin string `yaml:"storeAdmin"`
			}{
				Mssql: "testuser",
			},
		},
		SecretConfig: &config.SecretConfig{
			Passwords: struct {
				Rac                string `yaml:"rac"`
				Db                 string `yaml:"db"`
				Mssql              string `yaml:"mssql"`
				StoreAdminPassword string `yaml:"storeAdminPassword"`
				Smb                string `yaml:"smb"`
			}{
				Mssql: "testpass",
			},
		},
		DbConfig: map[string]*config.DatabaseInfo{
			"testdb": {
				DbServer: "test-server",
			},
		},
		ProjectConfig: &config.ProjectConfig{
			Prod: map[string]struct {
				DbName     string                 `yaml:"dbName"`
				AddDisable []string               `yaml:"add-disable"`
				Related    map[string]interface{} `yaml:"related"`
			}{
				"testdb": {
					DbName: "testdb",
					Related: map[string]interface{}{
						"related-db": "related value",
					},
				},
			},
		},
	}

	_, err := LoadDBRestoreConfig(logger, cfg, "testdb")
	// Ожидаем ошибку, так как конфигурация неполная для работы с related databases
	if err == nil {
		t.Error("Expected error but got none")
	}
}

// TestDBRestore_GetRestoreStats_Success тестирует успешное получение статистики
func TestDBRestore_GetRestoreStats_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	dbRestore := &DBRestore{
		Db:               db,
		TimeToStatistic:  "2023-01-01T00:00:00",
		SrcDB:           "sourcedb",
		DstServer:       "dest-server",
		AutoTimeOut:     true,
	}

	rows := sqlmock.NewRows([]string{"avg", "max"}).
		AddRow(sql.NullInt64{Int64: 3600, Valid: true}, sql.NullInt64{Int64: 7200, Valid: true})

	mock.ExpectQuery("SELECT AVG").WillReturnRows(rows)

	ctx := context.Background()
	stats, err := dbRestore.GetRestoreStats(ctx)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}
	if !stats.AvgRestoreTimeSecond.Valid || stats.AvgRestoreTimeSecond.Int64 != 3600 {
		t.Errorf("Expected avg 3600, got %v", stats.AvgRestoreTimeSecond)
	}
	if !stats.MaxRestoreTimeSecond.Valid || stats.MaxRestoreTimeSecond.Int64 != 7200 {
		t.Errorf("Expected max 7200, got %v", stats.MaxRestoreTimeSecond)
	}

	// Проверяем автоматический расчет таймаута
	expectedTimeout := time.Duration(float64(7200*time.Second) * 1.7)
	if dbRestore.Timeout != expectedTimeout {
		t.Errorf("Expected timeout %v, got %v", expectedTimeout, dbRestore.Timeout)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("There were unfulfilled expectations: %v", err)
	}
}

// TestDBRestore_GetRestoreStats_NoRows тестирует обработку отсутствующих данных
func TestDBRestore_GetRestoreStats_NoRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	dbRestore := &DBRestore{
		Db:              db,
		TimeToStatistic: "2023-01-01T00:00:00",
		SrcDB:          "sourcedb",
		DstServer:      "dest-server",
	}

	mock.ExpectQuery("SELECT AVG").WillReturnError(sql.ErrNoRows)

	ctx := context.Background()
	stats, err := dbRestore.GetRestoreStats(ctx)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("There were unfulfilled expectations: %v", err)
	}
}

// TestDBRestore_GetRestoreStats_QueryError тестирует ошибку выполнения запроса
func TestDBRestore_GetRestoreStats_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	dbRestore := &DBRestore{
		Db:              db,
		TimeToStatistic: "2023-01-01T00:00:00",
		SrcDB:          "sourcedb",
		DstServer:      "dest-server",
	}

	mock.ExpectQuery("SELECT AVG").WillReturnError(sql.ErrConnDone)

	ctx := context.Background()
	_, err = dbRestore.GetRestoreStats(ctx)

	if err == nil {
		t.Error("Expected error but got none")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("There were unfulfilled expectations: %v", err)
	}
}

// TestDBRestore_Restore_Success тестирует успешное восстановление
func TestDBRestore_Restore_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	dbRestore := &DBRestore{
		Db:            db,
		Description:   "test restore",
		TimeToRestore: "2023-01-01T12:00:00",
		User:          "testuser",
		SrcServer:     "src-server",
		SrcDB:         "srcdb",
		DstServer:     "dst-server",
		DstDB:         "dstdb",
	}

	mock.ExpectExec("USE master").WillReturnResult(sqlmock.NewResult(0, 1))

	ctx := context.Background()
	err = dbRestore.Restore(ctx)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("There were unfulfilled expectations: %v", err)
	}
}

// TestDBRestore_Restore_QueryError тестирует ошибку выполнения запроса восстановления
func TestDBRestore_Restore_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	dbRestore := &DBRestore{
		Db:            db,
		Description:   "test restore",
		TimeToRestore: "2023-01-01T12:00:00",
		User:          "testuser",
		SrcServer:     "src-server",
		SrcDB:         "srcdb",
		DstServer:     "dst-server",
		DstDB:         "dstdb",
	}

	mock.ExpectExec("USE master").WillReturnError(sql.ErrConnDone)

	ctx := context.Background()
	err = dbRestore.Restore(ctx)

	if err == nil {
		t.Error("Expected error but got none")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("There were unfulfilled expectations: %v", err)
	}
}

// TestSetupDBRestoreDefaults_EnvPassword тестирует использование переменной окружения MSSQL_PASSWORD
func TestSetupDBRestoreDefaults_EnvPassword(t *testing.T) {
	// Сохраняем исходное значение переменной окружения
	original := os.Getenv("MSSQL_PASSWORD")
	defer func() {
		if original == "" {
			_ = os.Unsetenv("MSSQL_PASSWORD")
		} else {
			_ = os.Setenv("MSSQL_PASSWORD", original)
		}
	}()

	// Устанавливаем переменную окружения
	_ = os.Setenv("MSSQL_PASSWORD", "env_password")

	dbR := &DBRestore{}
	setupDBRestoreDefaults(dbR)

	if dbR.Password != "env_password" {
		t.Errorf("Expected password from environment variable, got %s", dbR.Password)
	}
}

// TestDBRestore_Close_NonNilDB тестирует закрытие не-nil базы данных
func TestDBRestore_Close_NonNilDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}

	dbRestore := &DBRestore{Db: db}

	mock.ExpectClose()

	err = dbRestore.Close()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("There were unfulfilled expectations: %v", err)
	}
}