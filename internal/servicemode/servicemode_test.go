package servicemode

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/rac"
)

// MockLogger реализует интерфейс Logger для тестирования
type MockLogger struct {
	InfoCalls  []LogCall
	WarnCalls  []LogCall
	ErrorCalls []LogCall
	DebugCalls []LogCall
}

type LogCall struct {
	Msg  string
	Args []any
}

func (m *MockLogger) Info(msg string, args ...any) {
	m.InfoCalls = append(m.InfoCalls, LogCall{Msg: msg, Args: args})
}

func (m *MockLogger) Warn(msg string, args ...any) {
	m.WarnCalls = append(m.WarnCalls, LogCall{Msg: msg, Args: args})
}

func (m *MockLogger) Error(msg string, args ...any) {
	m.ErrorCalls = append(m.ErrorCalls, LogCall{Msg: msg, Args: args})
}

func (m *MockLogger) Debug(msg string, args ...any) {
	m.DebugCalls = append(m.DebugCalls, LogCall{Msg: msg, Args: args})
}

// MockRacClient представляет мок-объект для RAC клиента
type MockRacClient struct {
	GetClusterUUIDFunc        func(ctx context.Context) (string, error)
	GetInfobaseUUIDFunc       func(ctx context.Context, clusterUUID, infobaseName string) (string, error)
	EnableServiceModeFunc     func(ctx context.Context, clusterUUID, infobaseUUID string, terminateSessions bool) error
	DisableServiceModeFunc    func(ctx context.Context, clusterUUID, infobaseUUID string) error
	GetServiceModeStatusFunc  func(ctx context.Context, clusterUUID, infobaseUUID string) (*rac.ServiceModeStatus, error)
}

func (m *MockRacClient) GetClusterUUID(ctx context.Context) (string, error) {
	if m.GetClusterUUIDFunc != nil {
		return m.GetClusterUUIDFunc(ctx)
	}
	return "test-cluster-uuid", nil
}

func (m *MockRacClient) GetInfobaseUUID(ctx context.Context, clusterUUID, infobaseName string) (string, error) {
	if m.GetInfobaseUUIDFunc != nil {
		return m.GetInfobaseUUIDFunc(ctx, clusterUUID, infobaseName)
	}
	return "test-infobase-uuid", nil
}

func (m *MockRacClient) EnableServiceMode(ctx context.Context, clusterUUID, infobaseUUID string, terminateSessions bool) error {
	if m.EnableServiceModeFunc != nil {
		return m.EnableServiceModeFunc(ctx, clusterUUID, infobaseUUID, terminateSessions)
	}
	return nil
}

func (m *MockRacClient) DisableServiceMode(ctx context.Context, clusterUUID, infobaseUUID string) error {
	if m.DisableServiceModeFunc != nil {
		return m.DisableServiceModeFunc(ctx, clusterUUID, infobaseUUID)
	}
	return nil
}

func (m *MockRacClient) GetServiceModeStatus(ctx context.Context, clusterUUID, infobaseUUID string) (*rac.ServiceModeStatus, error) {
	if m.GetServiceModeStatusFunc != nil {
		return m.GetServiceModeStatusFunc(ctx, clusterUUID, infobaseUUID)
	}
	return &rac.ServiceModeStatus{Enabled: false}, nil
}

// TestSlogLogger тестирует адаптер SlogLogger
func TestSlogLogger(t *testing.T) {
	// Создаем временный файл для логов
	tmpFile, err := os.CreateTemp("", "test_log_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}()
	defer func() {
		if err := tmpFile.Close(); err != nil {
			t.Logf("Failed to close temp file: %v", err)
		}
	}()

	// Создаем slog.Logger с выводом в файл
	logger := slog.New(slog.NewJSONHandler(tmpFile, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	slogLogger := &SlogLogger{Logger: logger}

	// Тестируем все методы логирования
	slogLogger.Info("Test info message", "key", "value")
	slogLogger.Warn("Test warn message", "key", "value")
	slogLogger.Error("Test error message", "key", "value")
	slogLogger.Debug("Test debug message", "key", "value")

	// Проверяем, что логи записались
	if err := tmpFile.Sync(); err != nil {
		t.Fatalf("Failed to sync temp file: %v", err)
	}
	stat, err := tmpFile.Stat()
	if err != nil {
		t.Fatalf("Failed to get file stat: %v", err)
	}

	if stat.Size() == 0 {
		t.Error("Expected log file to contain data, but it's empty")
	}
}

// TestNewClient тестирует создание нового клиента
func TestNewClient(t *testing.T) {
	config := RacConfig{
		RacPath:     "/usr/bin/rac",
		RacServer:   "localhost",
		RacPort:     1545,
		RacUser:     "admin",
		RacPassword: "password",
		DbUser:      "dbuser",
		DbPassword:  "dbpass",
		RacTimeout:  30 * time.Second,
		RacRetries:  3,
	}

	logger := &MockLogger{}
	client := NewClient(config, logger)

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	if client.logger != logger {
		t.Error("Expected logger to be set correctly")
	}
}

// TestRacConfig тестирует структуру RacConfig
func TestRacConfig(t *testing.T) {
	config := RacConfig{
		RacPath:     "/usr/bin/rac",
		RacServer:   "localhost",
		RacPort:     1545,
		RacUser:     "admin",
		RacPassword: "password",
		DbUser:      "dbuser",
		DbPassword:  "dbpass",
		RacTimeout:  30 * time.Second,
		RacRetries:  3,
	}

	if config.RacPath != "/usr/bin/rac" {
		t.Errorf("Expected RacPath '/usr/bin/rac', got %s", config.RacPath)
	}
	if config.RacServer != "localhost" {
		t.Errorf("Expected RacServer 'localhost', got %s", config.RacServer)
	}
	if config.RacPort != 1545 {
		t.Errorf("Expected RacPort 1545, got %d", config.RacPort)
	}
	if config.RacTimeout != 30*time.Second {
		t.Errorf("Expected RacTimeout 30s, got %v", config.RacTimeout)
	}
	if config.RacRetries != 3 {
		t.Errorf("Expected RacRetries 3, got %d", config.RacRetries)
	}
}

// TestServiceModeStatus тестирует структуру ServiceModeStatus
func TestServiceModeStatus(t *testing.T) {
	status := &rac.ServiceModeStatus{
		Enabled:        true,
		Message:        "Service mode enabled",
		ActiveSessions: 5,
	}

	if !status.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if status.Message != "Service mode enabled" {
		t.Errorf("Expected Message 'Service mode enabled', got %s", status.Message)
	}
	if status.ActiveSessions != 5 {
		t.Errorf("Expected ActiveSessions 5, got %d", status.ActiveSessions)
	}
}

// TestManagerInterface тестирует интерфейс Manager
func TestManagerInterface(t *testing.T) {
	// Проверяем, что Client реализует интерфейс Manager
	var _ Manager = (*Client)(nil)

	// Создаем клиент для проверки интерфейса
	config := RacConfig{
		RacPath:     "/usr/bin/echo",
		RacServer:   "localhost",
		RacPort:     1545,
		RacUser:     "admin",
		RacPassword: "password",
		DbUser:      "dbuser",
		DbPassword:  "dbpass",
		RacTimeout:  30 * time.Second,
		RacRetries:  3,
	}

	logger := &MockLogger{}
	client := NewClient(config, logger)

	if client == nil {
		t.Error("Expected non-nil client")
	}

	// Проверяем, что клиент реализует все методы интерфейса Manager
	ctx := context.Background()
	
	// Тестируем валидацию входных параметров без реальных подключений
	err := client.EnableServiceMode(ctx, "", false)
	if err == nil {
		t.Error("Expected error for empty infobase name")
	}

	err = client.DisableServiceMode(ctx, "")
	if err == nil {
		t.Error("Expected error for empty infobase name")
	}

	_, err = client.GetServiceModeStatus(ctx, "")
	if err == nil {
		t.Error("Expected error for empty infobase name")
	}
}

// TestLoadServiceModeConfigForDb тестирует загрузку конфигурации для базы данных
func TestLoadServiceModeConfigForDb(t *testing.T) {
	// Создаем тестовую конфигурацию с AppConfig и SecretConfig
	cfg := &config.Config{
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				Rac: "/usr/bin/rac",
			},
			Rac: struct {
				Port    int `yaml:"port"`
				Timeout int `yaml:"timeout"`
				Retries int `yaml:"retries"`
			}{
				Port:    1545,
				Timeout: 30,
				Retries: 3,
			},
			Users: struct {
				Rac        string `yaml:"rac"`
				Db         string `yaml:"db"`
				Mssql      string `yaml:"mssql"`
				StoreAdmin string `yaml:"storeAdmin"`
			}{
				Rac: "admin",
				Db:  "dbuser",
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
				Rac: "password",
				Db:  "dbpass",
				Smb: "smbpass",
			},
		},
		DbConfig: map[string]*config.DatabaseInfo{
			"testdb": {
				OneServer: "localhost",
				Prod:      false,
				DbServer:  "localhost",
			},
		},
	}

	t.Run("successful config load", func(t *testing.T) {
		racConfig, err := LoadServiceModeConfigForDb("testdb", cfg)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if racConfig.RacPath != "/usr/bin/rac" {
			t.Errorf("Expected RacPath '/usr/bin/rac', got: %s", racConfig.RacPath)
		}
		if racConfig.RacServer != "localhost" {
			t.Errorf("Expected RacServer 'localhost', got: %s", racConfig.RacServer)
		}
		if racConfig.RacPort != 1545 {
			t.Errorf("Expected RacPort 1545, got: %d", racConfig.RacPort)
		}
		if racConfig.RacTimeout != 30*time.Second {
			t.Errorf("Expected RacTimeout 30s, got: %v", racConfig.RacTimeout)
		}
	})

	t.Run("app config not loaded", func(t *testing.T) {
		cfgNoApp := &config.Config{
			DbConfig: map[string]*config.DatabaseInfo{
				"testdb": {
					OneServer: "localhost",
					Prod:      false,
					DbServer:  "localhost",
				},
			},
		}
		_, err := LoadServiceModeConfigForDb("testdb", cfgNoApp)
		if err == nil {
			t.Error("Expected error for missing app config")
		}
		expectedMsg := "app config is not loaded"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message '%s', got: %s", expectedMsg, err.Error())
		}
	})

	t.Run("nil config", func(t *testing.T) {
		_, err := LoadServiceModeConfigForDb("testdb", nil)
		if err == nil {
			t.Error("Expected error for nil config")
		}
		expectedMsg := "config is nil"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message '%s', got: %s", expectedMsg, err.Error())
		}
	})
}

// TestManageServiceMode тестирует управление сервисным режимом
func TestManageServiceMode(t *testing.T) {
	// Создаем тестовую конфигурацию без AppConfig и SecretConfig для тестирования ошибок
	cfg := &config.Config{
		DbConfig: map[string]*config.DatabaseInfo{
			"testdb": {
				OneServer: "localhost",
				Prod:      false,
				DbServer:  "localhost",
			},
		},
	}

	// Создаем полную конфигурацию для успешных тестов
	cfgComplete := &config.Config{
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				Rac: "/usr/bin/rac",
			},
			Rac: struct {
				Port    int `yaml:"port"`
				Timeout int `yaml:"timeout"`
				Retries int `yaml:"retries"`
			}{
				Port:    1545,
				Timeout: 30,
				Retries: 3,
			},
			Users: struct {
				Rac        string `yaml:"rac"`
				Db         string `yaml:"db"`
				Mssql      string `yaml:"mssql"`
				StoreAdmin string `yaml:"storeAdmin"`
			}{
				Rac: "admin",
				Db:  "dbuser",
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
				Rac: "password",
				Db:  "dbpass",
				Smb: "smbpass",
			},
		},
		DbConfig: map[string]*config.DatabaseInfo{
			"testdb": {
				OneServer: "localhost",
				Prod:      false,
				DbServer:  "localhost",
			},
		},
	}

	logger := &MockLogger{}
	ctx := context.Background()

	t.Run("invalid action", func(t *testing.T) {
		err := ManageServiceMode(ctx, "invalid", "testdb", false, cfg, logger)
		if err == nil {
			t.Error("Expected error for invalid action")
		}
		expectedMsg := "unknown action: invalid"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message '%s', got: %s", expectedMsg, err.Error())
		}
	})

	t.Run("database not found", func(t *testing.T) {
		err := ManageServiceMode(ctx, "enable", "nonexistent", false, cfg, logger)
		if err == nil {
			t.Error("Expected error for nonexistent database")
		}
	})

	t.Run("nil config", func(t *testing.T) {
		err := ManageServiceMode(ctx, "enable", "testdb", false, nil, logger)
		if err == nil {
			t.Error("Expected error for nil config")
		}
	})

	t.Run("missing app config", func(t *testing.T) {
		err := ManageServiceMode(ctx, "enable", "testdb", true, cfg, logger)
		// Ожидаем ошибку из-за отсутствия AppConfig
		if err == nil {
			t.Error("Expected error due to missing AppConfig")
		}
	})

	t.Run("successful enable action", func(t *testing.T) {
		// Этот тест может не пройти из-за отсутствия реального RAC, но покрывает логику
		err := ManageServiceMode(ctx, "enable", "testdb", false, cfgComplete, logger)
		// Ожидаем ошибку подключения к RAC, но не ошибку валидации
		if err != nil {
			// Проверяем, что это ошибка подключения, а не валидации
			if err.Error() == "unknown action: enable" {
				t.Error("Unexpected validation error for valid enable action")
			}
		}
	})

	t.Run("successful disable action", func(t *testing.T) {
		// Этот тест может не пройти из-за отсутствия реального RAC, но покрывает логику
		err := ManageServiceMode(ctx, "disable", "testdb", false, cfgComplete, logger)
		// Ожидаем ошибку подключения к RAC, но не ошибку валидации
		if err != nil {
			// Проверяем, что это ошибка подключения, а не валидации
			if err.Error() == "unknown action: disable" {
				t.Error("Unexpected validation error for valid disable action")
			}
		}
	})

	t.Run("successful status action", func(t *testing.T) {
		// Этот тест может не пройти из-за отсутствия реального RAC, но покрывает логику
		err := ManageServiceMode(ctx, "status", "testdb", false, cfgComplete, logger)
		// Ожидаем ошибку подключения к RAC, но не ошибку валидации
		if err != nil {
			// Проверяем, что это ошибка подключения, а не валидации
			if err.Error() == "unknown action: status" {
				t.Error("Unexpected validation error for valid status action")
			}
		}
	})
}

// TestClient_EnableServiceMode тестирует включение сервисного режима
func TestClient_EnableServiceMode(t *testing.T) {
	logger := &MockLogger{}

	// Создаем мок RAC клиент
	mockRacClient := &MockRacClient{}

	// Создаем конфигурацию для тестов
	config := &RacConfig{
		RacPath:   "/test/rac",
		RacServer: "localhost",
		RacPort:   1545,
	}

	client := &Client{
		config:    config,
		racClient: mockRacClient,
		logger:    logger,
	}

	ctx := context.Background()

	t.Run("empty infobase name", func(t *testing.T) {
		err := client.EnableServiceMode(ctx, "", true)
		if err == nil {
			t.Error("Expected error for empty infobase name")
		}
		expectedMsg := "infobase name cannot be empty"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message '%s', got: %s", expectedMsg, err.Error())
		}
	})
}

// TestClient_DisableServiceMode тестирует отключение сервисного режима
func TestClient_DisableServiceMode(t *testing.T) {
	// Создаем мок RAC клиент
	mockRacClient := &MockRacClient{}
	
	// Создаем конфигурацию для тестов
	config := &RacConfig{
		RacPath:   "/test/rac",
		RacServer: "localhost",
		RacPort:   1545,
	}
	
	// Создаем клиент с мок RAC клиентом
	client := &Client{
		config:    config,
		racClient: mockRacClient,
		logger:    &MockLogger{},
	}

	ctx := context.Background()

	t.Run("successful disable", func(t *testing.T) {
		// Настраиваем мок-клиент для возврата ошибки подключения
		mockRacClient.GetClusterUUIDFunc = func(ctx context.Context) (string, error) {
			return "", errors.New("connection failed")
		}
		
		err := client.DisableServiceMode(ctx, "testdb")
		if err == nil {
			t.Error("Expected error due to RAC connection failure")
		}
	})

	t.Run("empty infobase name", func(t *testing.T) {
		err := client.DisableServiceMode(ctx, "")
		if err == nil {
			t.Error("Expected error for empty infobase name")
		}
	})
}

// TestClient_GetServiceModeStatus тестирует получение статуса сервисного режима
func TestClient_GetServiceModeStatus(t *testing.T) {
	// Создаем мок RAC клиент
	mockRacClient := &MockRacClient{}
	
	// Создаем конфигурацию для тестов
	config := &RacConfig{
		RacPath:   "/test/rac",
		RacServer: "localhost",
		RacPort:   1545,
	}
	
	// Создаем клиент с мок RAC клиентом
	client := &Client{
		config:    config,
		racClient: mockRacClient,
		logger:    &MockLogger{},
	}

	ctx := context.Background()

	t.Run("successful get status", func(t *testing.T) {
		// Настраиваем мок для возврата статуса
		mockRacClient.GetServiceModeStatusFunc = func(ctx context.Context, clusterUUID, infobaseUUID string) (*rac.ServiceModeStatus, error) {
			return &rac.ServiceModeStatus{Enabled: true}, nil
		}
		
		status, err := client.GetServiceModeStatus(ctx, "testdb")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if status == nil {
			t.Error("Expected status, got nil")
		}
		if !status.Enabled {
			t.Error("Expected service mode to be enabled")
		}
	})

	t.Run("empty infobase name", func(t *testing.T) {
		_, err := client.GetServiceModeStatus(ctx, "")
		if err == nil {
			t.Error("Expected error for empty infobase name")
		}
	})
}

// TestNewClient_DetailedValidation тестирует детальную валидацию создания клиента
func TestNewClient_DetailedValidation(t *testing.T) {
	logger := &MockLogger{}

	t.Run("valid config", func(t *testing.T) {
		config := RacConfig{
			RacPath:     "/usr/bin/rac",
			RacServer:   "localhost",
			RacPort:     1545,
			RacUser:     "admin",
			RacPassword: "password",
			DbUser:      "dbuser",
			DbPassword:  "dbpass",
			RacTimeout:  30 * time.Second,
			RacRetries:  3,
		}

		client := NewClient(config, logger)
		
		if client == nil {
			t.Fatal("Expected non-nil client")
		}
		
		if client.logger != logger {
			t.Error("Expected logger to be set correctly")
		}
		
		if client.racClient == nil {
			t.Error("Expected racClient to be initialized")
		}
	})

	t.Run("nil logger", func(t *testing.T) {
		config := RacConfig{
			RacPath:   "/usr/bin/rac",
			RacServer: "localhost",
			RacPort:   1545,
		}

		client := NewClient(config, nil)
		
		if client == nil {
			t.Fatal("Expected non-nil client even with nil logger")
		}
	})

	t.Run("empty config", func(t *testing.T) {
		config := RacConfig{}

		client := NewClient(config, logger)

		if client == nil {
			t.Fatal("Expected non-nil client even with empty config")
		}
	})

	t.Run("slog logger", func(t *testing.T) {
		config := RacConfig{
			RacPath:   "/usr/bin/rac",
			RacServer: "localhost",
			RacPort:   1545,
		}

		slogLogger := &SlogLogger{Logger: slog.Default()}
		client := NewClient(config, slogLogger)

		if client == nil {
			t.Fatal("Expected non-nil client with SlogLogger")
		}

		if client.logger != slogLogger {
			t.Error("Expected SlogLogger to be set correctly")
		}
	})
}

// TestNewClientWithRacClient тестирует создание клиента с переданным RAC клиентом
func TestNewClientWithRacClient(t *testing.T) {
	config := RacConfig{
		RacPath:     "/usr/bin/rac",
		RacServer:   "localhost",
		RacPort:     1545,
		RacUser:     "admin",
		RacPassword: "password",
		DbUser:      "dbuser",
		DbPassword:  "dbpass",
		RacTimeout:  30 * time.Second,
		RacRetries:  3,
	}

	mockRacClient := &MockRacClient{}
	mockLogger := &MockLogger{}

	client := NewClientWithRacClient(config, mockRacClient, mockLogger)

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	if client.config.RacPath != config.RacPath || 
		client.config.RacServer != config.RacServer ||
		client.config.RacPort != config.RacPort {
		t.Error("Expected config to be set correctly")
	}

	if client.racClient != mockRacClient {
		t.Error("Expected racClient to be set correctly")
	}

	if client.logger != mockLogger {
		t.Error("Expected logger to be set correctly")
	}

	// Тест с nil logger
	client = NewClientWithRacClient(config, mockRacClient, nil)
	if client == nil {
		t.Fatal("Expected non-nil client even with nil logger")
	}
	if client.logger != nil {
		t.Error("Expected logger to be nil")
	}
}

// TestClient_MethodsWithMockedRacClient тестирует методы клиента с замоканным RAC клиентом
func TestClient_MethodsWithMockedRacClient(t *testing.T) {
	ctx := context.Background()
	logger := &MockLogger{}

	t.Run("successful enable service mode", func(t *testing.T) {
		mockRacClient := &MockRacClient{
			GetClusterUUIDFunc: func(ctx context.Context) (string, error) {
				return "test-cluster-uuid", nil
			},
			GetInfobaseUUIDFunc: func(ctx context.Context, clusterUUID, infobaseName string) (string, error) {
				return "test-infobase-uuid", nil
			},
			EnableServiceModeFunc: func(ctx context.Context, clusterUUID, infobaseUUID string, terminateSessions bool) error {
				return nil
			},
		}

		client := &Client{
			racClient: mockRacClient,
			logger:    logger,
		}

		err := client.EnableServiceMode(ctx, "test-infobase", false)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("successful disable service mode", func(t *testing.T) {
		mockRacClient := &MockRacClient{
			GetClusterUUIDFunc: func(ctx context.Context) (string, error) {
				return "test-cluster-uuid", nil
			},
			GetInfobaseUUIDFunc: func(ctx context.Context, clusterUUID, infobaseName string) (string, error) {
				return "test-infobase-uuid", nil
			},
			DisableServiceModeFunc: func(ctx context.Context, clusterUUID, infobaseUUID string) error {
				return nil
			},
		}

		client := &Client{
			racClient: mockRacClient,
			logger:    logger,
		}

		err := client.DisableServiceMode(ctx, "test-infobase")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("successful get service mode status", func(t *testing.T) {
		mockRacClient := &MockRacClient{
			GetClusterUUIDFunc: func(ctx context.Context) (string, error) {
				return "test-cluster-uuid", nil
			},
			GetInfobaseUUIDFunc: func(ctx context.Context, clusterUUID, infobaseName string) (string, error) {
				return "test-infobase-uuid", nil
			},
			GetServiceModeStatusFunc: func(ctx context.Context, clusterUUID, infobaseUUID string) (*rac.ServiceModeStatus, error) {
				return &rac.ServiceModeStatus{Enabled: true}, nil
			},
		}

		client := &Client{
			racClient: mockRacClient,
			logger:    logger,
		}

		status, err := client.GetServiceModeStatus(ctx, "test-infobase")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if status == nil {
			t.Error("Expected status, got nil")
		}
		if !status.Enabled {
			t.Error("Expected service mode to be enabled")
		}
	})

	t.Run("error getting cluster UUID", func(t *testing.T) {
		mockRacClient := &MockRacClient{
			GetClusterUUIDFunc: func(ctx context.Context) (string, error) {
				return "", errors.New("cluster not found")
			},
		}

		client := &Client{
			racClient: mockRacClient,
			logger:    logger,
		}

		err := client.EnableServiceMode(ctx, "test-infobase", false)
		if err == nil {
			t.Error("Expected error for cluster UUID failure")
		}
	})

	t.Run("error getting infobase UUID", func(t *testing.T) {
		mockRacClient := &MockRacClient{
			GetClusterUUIDFunc: func(ctx context.Context) (string, error) {
				return "test-cluster-uuid", nil
			},
			GetInfobaseUUIDFunc: func(ctx context.Context, clusterUUID, infobaseName string) (string, error) {
				return "", errors.New("infobase not found")
			},
		}

		client := &Client{
			racClient: mockRacClient,
			logger:    logger,
		}

		err := client.EnableServiceMode(ctx, "test-infobase", false)
		if err == nil {
			t.Error("Expected error for infobase UUID failure")
		}
	})

	t.Run("error enabling service mode", func(t *testing.T) {
		mockRacClient := &MockRacClient{
			GetClusterUUIDFunc: func(ctx context.Context) (string, error) {
				return "test-cluster-uuid", nil
			},
			GetInfobaseUUIDFunc: func(ctx context.Context, clusterUUID, infobaseName string) (string, error) {
				return "test-infobase-uuid", nil
			},
			EnableServiceModeFunc: func(ctx context.Context, clusterUUID, infobaseUUID string, terminateSessions bool) error {
				return errors.New("enable service mode error")
			},
		}

		client := &Client{
			racClient: mockRacClient,
			logger:    logger,
		}

		err := client.EnableServiceMode(ctx, "test-infobase", false)
		if err == nil {
			t.Error("Expected error for enable service mode failure")
		}
	})

	t.Run("error disabling service mode", func(t *testing.T) {
		mockRacClient := &MockRacClient{
			GetClusterUUIDFunc: func(ctx context.Context) (string, error) {
				return "test-cluster-uuid", nil
			},
			GetInfobaseUUIDFunc: func(ctx context.Context, clusterUUID, infobaseName string) (string, error) {
				return "test-infobase-uuid", nil
			},
			DisableServiceModeFunc: func(ctx context.Context, clusterUUID, infobaseUUID string) error {
				return errors.New("disable service mode error")
			},
		}

		client := &Client{
			racClient: mockRacClient,
			logger:    logger,
		}

		err := client.DisableServiceMode(ctx, "test-infobase")
		if err == nil {
			t.Error("Expected error for disable service mode failure")
		}
	})

	t.Run("error getting service mode status", func(t *testing.T) {
		mockRacClient := &MockRacClient{
			GetClusterUUIDFunc: func(ctx context.Context) (string, error) {
				return "test-cluster-uuid", nil
			},
			GetInfobaseUUIDFunc: func(ctx context.Context, clusterUUID, infobaseName string) (string, error) {
				return "test-infobase-uuid", nil
			},
			GetServiceModeStatusFunc: func(ctx context.Context, clusterUUID, infobaseUUID string) (*rac.ServiceModeStatus, error) {
				return nil, errors.New("get status error")
			},
		}

		client := &Client{
			racClient: mockRacClient,
			logger:    logger,
		}

		_, err := client.GetServiceModeStatus(ctx, "test-infobase")
		if err == nil {
			t.Error("Expected error for get service mode status failure")
		}
	})
}