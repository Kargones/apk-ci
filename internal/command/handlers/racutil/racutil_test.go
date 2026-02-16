package racutil

import (
	"os"
	"testing"

	"github.com/Kargones/apk-ci/internal/config"
)

// helper: creates a temp file to act as a fake RAC binary.
func createFakeRACBinary(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp("", "fake-rac-*")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

// helper: returns a minimal valid config for NewClient.
func validConfig(racPath, server string) *config.Config {
	return &config.Config{
		InfobaseName: "test-db",
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{Rac: racPath},
			Rac: struct {
				Port    int `yaml:"port"`
				Timeout int `yaml:"timeout"`
				Retries int `yaml:"retries"`
			}{Port: 1545, Timeout: 30},
			Users: struct {
				Rac        string `yaml:"rac"`
				Db         string `yaml:"db"`
				Mssql      string `yaml:"mssql"`
				StoreAdmin string `yaml:"storeAdmin"`
			}{Rac: "admin", Db: "dbuser"},
		},
		DbConfig: map[string]*config.DatabaseInfo{
			"test-db": {OneServer: server},
		},
		SecretConfig: &config.SecretConfig{},
	}
}

func TestNewClient_NilAppConfig(t *testing.T) {
	cfg := &config.Config{AppConfig: nil}
	_, err := NewClient(cfg)
	if err == nil {
		t.Fatal("expected error for nil AppConfig")
	}
	if got := err.Error(); got != "конфигурация приложения не загружена" {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestNewClient_NoServer_NoRacConfig(t *testing.T) {
	racPath := createFakeRACBinary(t)
	cfg := validConfig(racPath, "")
	cfg.DbConfig = nil // no DbConfig → GetOneServer returns ""
	cfg.RacConfig = nil

	_, err := NewClient(cfg)
	if err == nil {
		t.Fatal("expected error when server cannot be determined")
	}
}

func TestNewClient_ServerFromDbConfig(t *testing.T) {
	racPath := createFakeRACBinary(t)
	cfg := validConfig(racPath, "db-server-1")

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewClient_ServerFallbackToRacConfig(t *testing.T) {
	racPath := createFakeRACBinary(t)
	cfg := validConfig(racPath, "")
	cfg.DbConfig = nil // GetOneServer returns ""
	cfg.RacConfig = &config.RacConfig{RacServer: "rac-fallback-server"}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewClient_ServerFallback_EmptyRacServer(t *testing.T) {
	racPath := createFakeRACBinary(t)
	cfg := validConfig(racPath, "")
	cfg.DbConfig = nil
	cfg.RacConfig = &config.RacConfig{RacServer: ""}

	_, err := NewClient(cfg)
	if err == nil {
		t.Fatal("expected error when RacConfig.RacServer is also empty")
	}
}

func TestNewClient_DefaultPort(t *testing.T) {
	racPath := createFakeRACBinary(t)
	cfg := validConfig(racPath, "server1")
	cfg.AppConfig.Rac.Port = 0 // should default to "1545"

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewClient_DefaultTimeout(t *testing.T) {
	racPath := createFakeRACBinary(t)
	cfg := validConfig(racPath, "server1")
	cfg.AppConfig.Rac.Timeout = 0 // should default to 30s

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewClient_LargeTimeout(t *testing.T) {
	racPath := createFakeRACBinary(t)
	cfg := validConfig(racPath, "server1")
	cfg.AppConfig.Rac.Timeout = 600 // 10 minutes, > 5 min warning threshold

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewClient_WithSecrets(t *testing.T) {
	racPath := createFakeRACBinary(t)
	cfg := validConfig(racPath, "server1")
	cfg.SecretConfig = &config.SecretConfig{}
	cfg.SecretConfig.Passwords.Rac = "rac-pass"
	cfg.SecretConfig.Passwords.Db = "db-pass"

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewClient_NilSecretConfig(t *testing.T) {
	racPath := createFakeRACBinary(t)
	cfg := validConfig(racPath, "server1")
	cfg.SecretConfig = nil

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewClient_CustomPort(t *testing.T) {
	racPath := createFakeRACBinary(t)
	cfg := validConfig(racPath, "server1")
	cfg.AppConfig.Rac.Port = 2545

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewClient_EmptyRacPath(t *testing.T) {
	cfg := validConfig("", "server1")

	_, err := NewClient(cfg)
	if err == nil {
		t.Fatal("expected error for empty RAC path")
	}
}

func TestNewClient_NonexistentRacPath(t *testing.T) {
	cfg := validConfig("/nonexistent/rac/path", "server1")

	_, err := NewClient(cfg)
	if err == nil {
		t.Fatal("expected error for nonexistent RAC path")
	}
}

func TestNewClient_WarningClusterUserNoPass(t *testing.T) {
	racPath := createFakeRACBinary(t)
	cfg := validConfig(racPath, "server1")
	cfg.AppConfig.Users.Rac = "admin"
	cfg.SecretConfig = nil // no passwords

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewClient_WarningInfobaseUserNoPass(t *testing.T) {
	racPath := createFakeRACBinary(t)
	cfg := validConfig(racPath, "server1")
	cfg.AppConfig.Users.Db = "dbuser"
	cfg.SecretConfig = nil // no passwords

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewClient_TableDriven(t *testing.T) {
	racPath := createFakeRACBinary(t)

	tests := []struct {
		name      string
		modify    func(*config.Config)
		wantErr   bool
	}{
		{
			name:    "valid minimal config",
			modify:  func(c *config.Config) {},
			wantErr: false,
		},
		{
			name:    "nil AppConfig",
			modify:  func(c *config.Config) { c.AppConfig = nil },
			wantErr: true,
		},
		{
			name: "no server anywhere",
			modify: func(c *config.Config) {
				c.DbConfig = nil
				c.RacConfig = nil
			},
			wantErr: true,
		},
		{
			name: "server from RacConfig fallback",
			modify: func(c *config.Config) {
				c.DbConfig = nil
				c.RacConfig = &config.RacConfig{RacServer: "fallback"}
			},
			wantErr: false,
		},
		{
			name: "port zero defaults to 1545",
			modify: func(c *config.Config) {
				c.AppConfig.Rac.Port = 0
			},
			wantErr: false,
		},
		{
			name: "timeout zero defaults to 30s",
			modify: func(c *config.Config) {
				c.AppConfig.Rac.Timeout = 0
			},
			wantErr: false,
		},
		{
			name: "empty rac path",
			modify: func(c *config.Config) {
				c.AppConfig.Paths.Rac = ""
			},
			wantErr: true,
		},
		{
			name: "nonexistent rac binary",
			modify: func(c *config.Config) {
				c.AppConfig.Paths.Rac = "/no/such/file"
			},
			wantErr: true,
		},
		{
			name: "with all secrets",
			modify: func(c *config.Config) {
				c.SecretConfig = &config.SecretConfig{}
				c.SecretConfig.Passwords.Rac = "pass1"
				c.SecretConfig.Passwords.Db = "pass2"
			},
			wantErr: false,
		},
		{
			name: "nil secret config",
			modify: func(c *config.Config) {
				c.SecretConfig = nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig(racPath, "server1")
			tt.modify(cfg)

			client, err := NewClient(cfg)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if client == nil {
					t.Error("expected non-nil client")
				}
			}
		})
	}
}
