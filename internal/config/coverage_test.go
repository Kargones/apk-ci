package config

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/pkg/alerting"
)

// ============================================================
// alerting_config.go
// ============================================================

func TestIsAlertingConfigPresent(t *testing.T) {
	tests := []struct {
		name string
		cfg  *AlertingConfig
		want bool
	}{
		{"nil config", nil, false},
		{"empty config", &AlertingConfig{}, false},
		{"enabled", &AlertingConfig{Enabled: true}, true},
		{"email enabled", &AlertingConfig{Email: alerting.EmailConfig{Enabled: true}}, true},
		{"email smtp host", &AlertingConfig{Email: alerting.EmailConfig{SMTPHost: "smtp.example.com"}}, true},
		{"telegram enabled", &AlertingConfig{Telegram: alerting.TelegramConfig{Enabled: true}}, true},
		{"telegram bot token", &AlertingConfig{Telegram: alerting.TelegramConfig{BotToken: "tok"}}, true},
		{"webhook enabled", &AlertingConfig{Webhook: alerting.WebhookConfig{Enabled: true}}, true},
		{"webhook urls", &AlertingConfig{Webhook: alerting.WebhookConfig{URLs: []string{"http://hook"}}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAlertingConfigPresent(tt.cfg); got != tt.want {
				t.Errorf("isAlertingConfigPresent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDefaultAlertingConfig(t *testing.T) {
	cfg := getDefaultAlertingConfig()
	if cfg == nil {
		t.Fatal("getDefaultAlertingConfig() returned nil")
	}
	if cfg.Enabled {
		t.Error("default alerting config should be disabled")
	}
}

func TestValidateAlertingConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *AlertingConfig
		wantErr bool
	}{
		{"disabled", &AlertingConfig{Enabled: false}, false},
		{"enabled no channels", &AlertingConfig{Enabled: true}, false},
		{"email no smtp host", &AlertingConfig{
			Enabled: true,
			Email:   alerting.EmailConfig{Enabled: true},
		}, true},
		{"email no from", &AlertingConfig{
			Enabled: true,
			Email:   alerting.EmailConfig{Enabled: true, SMTPHost: "smtp.example.com"},
		}, true},
		{"email no to", &AlertingConfig{
			Enabled: true,
			Email:   alerting.EmailConfig{Enabled: true, SMTPHost: "smtp.example.com", From: "a@b.com"},
		}, true},
		{"email valid", &AlertingConfig{
			Enabled: true,
			Email:   alerting.EmailConfig{Enabled: true, SMTPHost: "smtp.example.com", From: "a@b.com", To: []string{"c@d.com"}},
		}, false},
		{"telegram no token", &AlertingConfig{
			Enabled:  true,
			Telegram: alerting.TelegramConfig{Enabled: true},
		}, true},
		{"telegram no chat ids", &AlertingConfig{
			Enabled:  true,
			Telegram: alerting.TelegramConfig{Enabled: true, BotToken: "tok"},
		}, true},
		{"telegram valid", &AlertingConfig{
			Enabled:  true,
			Telegram: alerting.TelegramConfig{Enabled: true, BotToken: "tok", ChatIDs: []string{"123"}},
		}, false},
		{"webhook no urls", &AlertingConfig{
			Enabled: true,
			Webhook: alerting.WebhookConfig{Enabled: true},
		}, true},
		{"webhook valid", &AlertingConfig{
			Enabled: true,
			Webhook: alerting.WebhookConfig{Enabled: true, URLs: []string{"http://hook"}},
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAlertingConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAlertingConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadAlertingConfig(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	t.Run("from AppConfig", func(t *testing.T) {
		cfg := &Config{
			AppConfig: &AppConfig{
				Alerting: AlertingConfig{Enabled: true},
			},
		}
		result, err := loadAlertingConfig(l, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Enabled {
			t.Error("expected alerting to be enabled")
		}
	})

	t.Run("defaults", func(t *testing.T) {
		cfg := &Config{}
		result, err := loadAlertingConfig(l, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Enabled {
			t.Error("expected alerting to be disabled by default")
		}
	})
}

// ============================================================
// metrics_config.go
// ============================================================

func TestIsMetricsConfigPresent(t *testing.T) {
	tests := []struct {
		name string
		cfg  *MetricsConfig
		want bool
	}{
		{"nil", nil, false},
		{"empty", &MetricsConfig{}, false},
		{"enabled", &MetricsConfig{Enabled: true}, true},
		{"pushgateway url", &MetricsConfig{PushgatewayURL: "http://pg:9091"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isMetricsConfigPresent(tt.cfg); got != tt.want {
				t.Errorf("isMetricsConfigPresent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDefaultMetricsConfig(t *testing.T) {
	cfg := getDefaultMetricsConfig()
	if cfg == nil {
		t.Fatal("returned nil")
	}
	if cfg.Enabled {
		t.Error("should be disabled")
	}
	if cfg.JobName != "apk-ci" {
		t.Errorf("JobName = %q, want apk-ci", cfg.JobName)
	}
	if cfg.Timeout != 10*time.Second {
		t.Errorf("Timeout = %v, want 10s", cfg.Timeout)
	}
}

func TestValidateMetricsConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *MetricsConfig
		wantErr bool
	}{
		{"disabled", &MetricsConfig{Enabled: false}, false},
		{"enabled no url", &MetricsConfig{Enabled: true, Timeout: 10 * time.Second}, true},
		{"enabled bad timeout", &MetricsConfig{Enabled: true, PushgatewayURL: "http://pg:9091", Timeout: 0}, true},
		{"enabled valid", &MetricsConfig{Enabled: true, PushgatewayURL: "http://pg:9091", Timeout: 10 * time.Second}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMetricsConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMetricsConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadMetricsConfig(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	t.Run("from AppConfig", func(t *testing.T) {
		cfg := &Config{
			AppConfig: &AppConfig{
				Metrics: MetricsConfig{Enabled: true, PushgatewayURL: "http://pg:9091"},
			},
		}
		result, err := loadMetricsConfig(l, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Enabled {
			t.Error("expected metrics to be enabled")
		}
	})

	t.Run("defaults", func(t *testing.T) {
		cfg := &Config{}
		result, err := loadMetricsConfig(l, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Enabled {
			t.Error("expected metrics to be disabled by default")
		}
	})
}

// ============================================================
// tracing_config.go
// ============================================================

func TestIsTracingConfigPresent(t *testing.T) {
	tests := []struct {
		name string
		cfg  *TracingConfig
		want bool
	}{
		{"nil", nil, false},
		{"empty", &TracingConfig{}, false},
		{"enabled", &TracingConfig{Enabled: true}, true},
		{"endpoint", &TracingConfig{Endpoint: "http://jaeger:4318"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isTracingConfigPresent(tt.cfg); got != tt.want {
				t.Errorf("isTracingConfigPresent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDefaultTracingConfig(t *testing.T) {
	cfg := getDefaultTracingConfig()
	if cfg == nil {
		t.Fatal("returned nil")
	}
	if cfg.Enabled {
		t.Error("should be disabled")
	}
	if cfg.ServiceName != "apk-ci" {
		t.Errorf("ServiceName = %q", cfg.ServiceName)
	}
	if cfg.SamplingRate != 1.0 {
		t.Errorf("SamplingRate = %v", cfg.SamplingRate)
	}
}

func TestValidateTracingConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *TracingConfig
		wantErr bool
	}{
		{"disabled", &TracingConfig{Enabled: false}, false},
		{"no endpoint", &TracingConfig{Enabled: true, ServiceName: "s", Timeout: 5 * time.Second, SamplingRate: 1.0}, true},
		{"no service name", &TracingConfig{Enabled: true, Endpoint: "http://j:4318", ServiceName: "", Timeout: 5 * time.Second, SamplingRate: 1.0}, true},
		{"bad timeout", &TracingConfig{Enabled: true, Endpoint: "http://j:4318", ServiceName: "s", Timeout: 0, SamplingRate: 1.0}, true},
		{"negative sampling", &TracingConfig{Enabled: true, Endpoint: "http://j:4318", ServiceName: "s", Timeout: 5 * time.Second, SamplingRate: -0.1}, true},
		{"sampling > 1", &TracingConfig{Enabled: true, Endpoint: "http://j:4318", ServiceName: "s", Timeout: 5 * time.Second, SamplingRate: 1.1}, true},
		{"valid", &TracingConfig{Enabled: true, Endpoint: "http://j:4318", ServiceName: "s", Timeout: 5 * time.Second, SamplingRate: 0.5}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTracingConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTracingConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadTracingConfig(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	t.Run("from AppConfig", func(t *testing.T) {
		cfg := &Config{
			AppConfig: &AppConfig{
				Tracing: TracingConfig{Enabled: true, Endpoint: "http://j:4318"},
			},
		}
		result, err := loadTracingConfig(l, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Enabled {
			t.Error("expected tracing to be enabled")
		}
	})

	t.Run("defaults", func(t *testing.T) {
		cfg := &Config{}
		result, err := loadTracingConfig(l, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Enabled {
			t.Error("expected tracing to be disabled by default")
		}
	})
}

// ============================================================
// types.go - GetServer
// ============================================================

func TestDatabaseInfo_GetServer(t *testing.T) {
	tests := []struct {
		name string
		d    DatabaseInfo
		want string
	}{
		{"oneserver set", DatabaseInfo{OneServer: "srv1", DbServer: "srv2"}, "srv1"},
		{"oneserver empty", DatabaseInfo{OneServer: "", DbServer: "srv2"}, "srv2"},
		{"both empty", DatabaseInfo{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.GetServer(); got != tt.want {
				t.Errorf("GetServer() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ============================================================
// loader.go - applyInputParams
// ============================================================

func TestApplyInputParams(t *testing.T) {
	cfg := &Config{}
	ip := &InputParams{
		GHAGiteaURL:      "https://git.example.com",
		GHACommand:        "build",
		GHAIssueNumber:    "42",
		GHAConfigSystem:   "app.yaml",
		GHAConfigProject:  "project.yaml",
		GHAConfigSecret:   "secret.yaml",
		GHAConfigDbData:   "db.yaml",
		GHAMenuMain:       "menu-main.txt",
		GHAMenuDebug:      "menu-debug.txt",
		GHAAccessToken:    "token123",
		GHADbName:         "testdb",
		GHATerminateSessions: "true",
		GHAForceUpdate:    "true",
		GHAStartEpf:       "start.epf",
		GHARepository:     "org/repo",
		GHABranchForScan:  "feature",
		GHACommitHash:     "abc123",
	}

	applyInputParams(cfg, ip)

	if cfg.GiteaURL != "https://git.example.com" {
		t.Errorf("GiteaURL = %q", cfg.GiteaURL)
	}
	if cfg.Command != "build" {
		t.Errorf("Command = %q", cfg.Command)
	}
	if cfg.IssueNumber != 42 {
		t.Errorf("IssueNumber = %d", cfg.IssueNumber)
	}
	if cfg.Owner != "org" {
		t.Errorf("Owner = %q", cfg.Owner)
	}
	if cfg.Repo != "repo" {
		t.Errorf("Repo = %q", cfg.Repo)
	}
	if !cfg.TerminateSessions {
		t.Error("TerminateSessions should be true")
	}
	if !cfg.ForceUpdate {
		t.Error("ForceUpdate should be true")
	}
	if cfg.InfobaseName != "testdb" {
		t.Errorf("InfobaseName = %q", cfg.InfobaseName)
	}
	if cfg.BranchForScan != "feature" {
		t.Errorf("BranchForScan = %q", cfg.BranchForScan)
	}
	if cfg.CommitHash != "abc123" {
		t.Errorf("CommitHash = %q", cfg.CommitHash)
	}
}

func TestApplyInputParams_InvalidIssueNumber(t *testing.T) {
	cfg := &Config{}
	ip := &InputParams{
		GHAIssueNumber: "not-a-number",
		GHARepository:  "org/repo",
	}
	applyInputParams(cfg, ip)
	if cfg.IssueNumber != 0 {
		t.Errorf("IssueNumber should be 0 for invalid input, got %d", cfg.IssueNumber)
	}
}

// ============================================================
// loader.go - loadAllSubConfigs
// ============================================================

func TestLoadAllSubConfigs(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &Config{Logger: l}

	// This will fail to load from gitea (no API configured) and fall back to defaults
	loadAllSubConfigs(context.Background(), l, cfg)

	if cfg.AppConfig == nil {
		t.Error("AppConfig should have default value")
	}
	if cfg.GitConfig == nil {
		t.Error("GitConfig should have default value")
	}
	if cfg.LoggingConfig == nil {
		t.Error("LoggingConfig should have default value")
	}
	if cfg.ImplementationsConfig == nil {
		t.Error("ImplementationsConfig should have default value")
	}
	if cfg.RacConfig == nil {
		t.Error("RacConfig should have default value")
	}
	if cfg.AlertingConfig == nil {
		t.Error("AlertingConfig should have default value")
	}
	if cfg.MetricsConfig == nil {
		t.Error("MetricsConfig should have default value")
	}
	if cfg.TracingConfig == nil {
		t.Error("TracingConfig should have default value")
	}
	if cfg.SonarQubeConfig == nil {
		t.Error("SonarQubeConfig should have default value")
	}
	if cfg.ScannerConfig == nil {
		t.Error("ScannerConfig should have default value")
	}
}

// ============================================================
// loader.go - validateLoadedConfigs
// ============================================================

func TestValidateLoadedConfigs(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	t.Run("invalid implementations reset to default", func(t *testing.T) {
		cfg := &Config{
			ImplementationsConfig: &ImplementationsConfig{ConfigExport: "invalid", DBCreate: "1cv8"},
			AlertingConfig:        getDefaultAlertingConfig(),
			MetricsConfig:         getDefaultMetricsConfig(),
			TracingConfig:         getDefaultTracingConfig(),
		}
		validateLoadedConfigs(context.Background(), l, cfg)
		if cfg.ImplementationsConfig.ConfigExport != "1cv8" {
			t.Errorf("ConfigExport should be reset to default, got %q", cfg.ImplementationsConfig.ConfigExport)
		}
	})

	t.Run("invalid alerting disables", func(t *testing.T) {
		cfg := &Config{
			ImplementationsConfig: getDefaultImplementationsConfig(),
			AlertingConfig:        &AlertingConfig{Enabled: true, Email: alerting.EmailConfig{Enabled: true}},
			MetricsConfig:         getDefaultMetricsConfig(),
			TracingConfig:         getDefaultTracingConfig(),
		}
		validateLoadedConfigs(context.Background(), l, cfg)
		if cfg.AlertingConfig.Enabled {
			t.Error("alerting should be disabled after validation failure")
		}
	})

	t.Run("invalid metrics disables", func(t *testing.T) {
		cfg := &Config{
			ImplementationsConfig: getDefaultImplementationsConfig(),
			AlertingConfig:        getDefaultAlertingConfig(),
			MetricsConfig:         &MetricsConfig{Enabled: true},
			TracingConfig:         getDefaultTracingConfig(),
		}
		validateLoadedConfigs(context.Background(), l, cfg)
		if cfg.MetricsConfig.Enabled {
			t.Error("metrics should be disabled after validation failure")
		}
	})

	t.Run("invalid tracing disables", func(t *testing.T) {
		cfg := &Config{
			ImplementationsConfig: getDefaultImplementationsConfig(),
			AlertingConfig:        getDefaultAlertingConfig(),
			MetricsConfig:         getDefaultMetricsConfig(),
			TracingConfig:         &TracingConfig{Enabled: true, SamplingRate: 1.0, Timeout: 5 * time.Second},
		}
		validateLoadedConfigs(context.Background(), l, cfg)
		if cfg.TracingConfig.Enabled {
			t.Error("tracing should be disabled after validation failure (no endpoint)")
		}
	})

	t.Run("all valid passes", func(t *testing.T) {
		cfg := &Config{
			ImplementationsConfig: getDefaultImplementationsConfig(),
			AlertingConfig: &AlertingConfig{
				Enabled:  true,
				Telegram: alerting.TelegramConfig{Enabled: true, BotToken: "tok", ChatIDs: []string{"1"}},
			},
			MetricsConfig: &MetricsConfig{Enabled: true, PushgatewayURL: "http://pg:9091", Timeout: 10 * time.Second},
			TracingConfig: &TracingConfig{Enabled: true, Endpoint: "http://j:4318", ServiceName: "s", Timeout: 5 * time.Second, SamplingRate: 1.0},
		}
		validateLoadedConfigs(context.Background(), l, cfg)
		if !cfg.AlertingConfig.Enabled {
			t.Error("valid alerting should remain enabled")
		}
		if !cfg.MetricsConfig.Enabled {
			t.Error("valid metrics should remain enabled")
		}
		if !cfg.TracingConfig.Enabled {
			t.Error("valid tracing should remain enabled")
		}
	})
}

// ============================================================
// loader.go - loadMenuMainConfig / loadMenuDebugConfig
// ============================================================

func TestLoadMenuMainConfig_Empty(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &Config{ConfigMenuMain: ""}
	result, err := loadMenuMainConfig(context.Background(), l, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil for empty config path")
	}
}

func TestLoadMenuDebugConfig_Empty(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &Config{ConfigMenuDebug: ""}
	result, err := loadMenuDebugConfig(context.Background(), l, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil for empty config path")
	}
}

// ============================================================
// loader.go - loadGitConfig
// ============================================================

func TestLoadGitConfig(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	t.Run("from AppConfig", func(t *testing.T) {
		cfg := &Config{
			AppConfig: &AppConfig{
				Git: GitConfig{UserName: "test-user", UserEmail: "test@example.com"},
			},
		}
		result, err := loadGitConfig(l, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.UserName != "test-user" {
			t.Errorf("UserName = %q", result.UserName)
		}
	})

	t.Run("defaults", func(t *testing.T) {
		cfg := &Config{}
		result, err := loadGitConfig(l, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.UserName != "apk-ci" {
			t.Errorf("UserName = %q, want apk-ci", result.UserName)
		}
	})
}

// ============================================================
// loader.go - loadRacConfig
// ============================================================

func TestLoadRacConfig(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	t.Run("from AppConfig with secrets", func(t *testing.T) {
		cfg := &Config{
			AppConfig: &AppConfig{
				Paths: struct {
					Bin1cv8  string `yaml:"bin1cv8"`
					BinIbcmd string `yaml:"binIbcmd"`
					EdtCli   string `yaml:"edtCli"`
					Rac      string `yaml:"rac"`
				}{Rac: "/opt/1cv8/rac"},
				Rac: struct {
					Port    int `yaml:"port"`
					Timeout int `yaml:"timeout"`
					Retries int `yaml:"retries"`
				}{Port: 1545, Timeout: 30, Retries: 3},
				Users: struct {
					Rac        string `yaml:"rac"`
					Db         string `yaml:"db"`
					Mssql      string `yaml:"mssql"`
					StoreAdmin string `yaml:"storeAdmin"`
				}{Rac: "admin", Db: "dbuser"},
			},
			SecretConfig: &SecretConfig{},
		}
		cfg.SecretConfig.Passwords.Rac = "racpass"
		cfg.SecretConfig.Passwords.Db = "dbpass"

		result, err := loadRacConfig(l, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.RacPath != "/opt/1cv8/rac" {
			t.Errorf("RacPath = %q", result.RacPath)
		}
		if result.RacPassword != "racpass" {
			t.Errorf("RacPassword = %q", result.RacPassword)
		}
		if result.DbPassword != "dbpass" {
			t.Errorf("DbPassword = %q", result.DbPassword)
		}
	})

	t.Run("defaults", func(t *testing.T) {
		cfg := &Config{}
		result, err := loadRacConfig(l, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.RacPort != 1545 {
			t.Errorf("RacPort = %d", result.RacPort)
		}
	})
}

// ============================================================
// loader.go - loadLoggingConfig
// ============================================================

func TestLoadLoggingConfig(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	t.Run("from AppConfig", func(t *testing.T) {
		cfg := &Config{
			AppConfig: &AppConfig{
				Logging: LoggingConfig{Level: "debug", Format: "json"},
			},
		}
		result, err := loadLoggingConfig(l, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Level != "debug" {
			t.Errorf("Level = %q", result.Level)
		}
	})

	t.Run("defaults", func(t *testing.T) {
		cfg := &Config{}
		result, err := loadLoggingConfig(l, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Level != "info" {
			t.Errorf("Level = %q, want info", result.Level)
		}
	})
}

// ============================================================
// loader.go - loadImplementationsConfig
// ============================================================

func TestLoadImplementationsConfig(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	t.Run("from AppConfig", func(t *testing.T) {
		cfg := &Config{
			AppConfig: &AppConfig{
				Implementations: ImplementationsConfig{ConfigExport: "ibcmd", DBCreate: "ibcmd"},
			},
		}
		result, err := loadImplementationsConfig(l, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ConfigExport != "ibcmd" {
			t.Errorf("ConfigExport = %q", result.ConfigExport)
		}
	})

	t.Run("defaults", func(t *testing.T) {
		cfg := &Config{}
		result, err := loadImplementationsConfig(l, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ConfigExport != "1cv8" {
			t.Errorf("ConfigExport = %q", result.ConfigExport)
		}
	})
}

// ============================================================
// loader.go - loadMenuMainConfig / loadMenuDebugConfig with non-empty paths
// ============================================================

func TestLoadMenuMainConfig_NonEmpty(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &Config{ConfigMenuMain: "menu-main.txt"}
	_, err := loadMenuMainConfig(context.Background(), l, cfg)
	if err == nil {
		t.Error("expected error when gitea API not configured")
	}
}

func TestLoadMenuDebugConfig_NonEmpty(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &Config{ConfigMenuDebug: "menu-debug.txt"}
	_, err := loadMenuDebugConfig(context.Background(), l, cfg)
	if err == nil {
		t.Error("expected error when gitea API not configured")
	}
}

// ============================================================
// loader.go - loadAppConfig, loadProjectConfig, loadSecretConfig, loadDbConfig errors
// ============================================================

func TestLoadAppConfig_Error(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &Config{ConfigSystem: "nonexistent.yaml"}
	_, err := loadAppConfig(context.Background(), l, cfg)
	if err == nil {
		t.Error("expected error")
	}
}

func TestLoadProjectConfig_Error(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &Config{ConfigProject: "nonexistent.yaml"}
	_, err := loadProjectConfig(context.Background(), l, cfg)
	if err == nil {
		t.Error("expected error")
	}
}

func TestLoadSecretConfig_Error(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &Config{ConfigSecret: "nonexistent.yaml"}
	_, err := loadSecretConfig(context.Background(), l, cfg)
	if err == nil {
		t.Error("expected error")
	}
}

func TestLoadDbConfig_Error(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &Config{ConfigDbData: "nonexistent.yaml"}
	_, err := loadDbConfig(context.Background(), l, cfg)
	if err == nil {
		t.Error("expected error")
	}
}

// ============================================================
// config.go - AnalyzeProject
// ============================================================

func TestAnalyzeProject_Error(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &Config{}
	err := cfg.AnalyzeProject(context.Background(), l, "main")
	if err == nil {
		t.Error("expected error when gitea API not configured")
	}
}

// ============================================================
// loader.go - ReloadConfig
// ============================================================

func TestReloadConfig_Error(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &Config{Logger: l}
	err := cfg.ReloadConfig(context.Background())
	if err == nil {
		t.Error("expected error when gitea API not configured")
	}
}

// ============================================================
// database.go - LoadDBRestoreConfig
// ============================================================

func TestLoadDBRestoreConfig_Defaults(t *testing.T) {
	cfg := &Config{}
	result, err := LoadDBRestoreConfig(cfg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Server != "localhost" {
		t.Errorf("Server = %q, want localhost", result.Server)
	}
	if result.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v", result.Timeout)
	}
}

func TestLoadDBRestoreConfig_FromAppConfig(t *testing.T) {
	cfg := &Config{
		AppConfig: &AppConfig{
			Dbrestore: struct {
				Database    string `yaml:"database"`
				Timeout     string `yaml:"timeout"`
				Autotimeout bool   `yaml:"autotimeout"`
			}{
				Database:    "testdb",
				Timeout:     "5m",
				Autotimeout: true,
			},
		},
		SecretConfig: &SecretConfig{},
	}
	cfg.SecretConfig.Passwords.Mssql = "mssqlpass"

	result, err := LoadDBRestoreConfig(cfg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Database != "testdb" {
		t.Errorf("Database = %q", result.Database)
	}
	if result.Timeout != 5*time.Minute {
		t.Errorf("Timeout = %v", result.Timeout)
	}
	if !result.Autotimeout {
		t.Error("Autotimeout should be true")
	}
	if result.Password != "mssqlpass" {
		t.Errorf("Password = %q", result.Password)
	}
}

func TestLoadDBRestoreConfig_InvalidTimeout(t *testing.T) {
	cfg := &Config{
		AppConfig: &AppConfig{
			Dbrestore: struct {
				Database    string `yaml:"database"`
				Timeout     string `yaml:"timeout"`
				Autotimeout bool   `yaml:"autotimeout"`
			}{
				Timeout: "invalid",
			},
		},
	}
	result, err := LoadDBRestoreConfig(cfg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should keep default timeout when parsing fails
	if result.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", result.Timeout)
	}
}

func TestLoadDBRestoreConfig_WithDbName(t *testing.T) {
	cfg := &Config{
		DbConfig: map[string]*DatabaseInfo{
			"mydb": {OneServer: "srv1", Prod: true, DbServer: "dbsrv1"},
		},
		ProjectConfig: &ProjectConfig{
			Prod: map[string]struct {
				DbName     string                 `yaml:"dbName"`
				AddDisable []string               `yaml:"add-disable"`
				Related    map[string]interface{} `yaml:"related"`
			}{
				"mydb": {DbName: "mydb"},
			},
		},
	}
	// This should try DetermineSrcAndDstServers, which may or may not succeed
	_, _ = LoadDBRestoreConfig(cfg, "mydb")
}

// ============================================================
// loader.go - getSlog additional coverage
// ============================================================

func TestGetSlog_DebugUser(t *testing.T) {
	l := getSlog("xor", "Debug")
	if l == nil {
		t.Fatal("getSlog returned nil")
	}
}

func TestGetSlog_WarnLevel(t *testing.T) {
	l := getSlog("xor", "Warn")
	if l == nil {
		t.Fatal("getSlog returned nil")
	}
}

func TestGetSlog_ErrorLevel(t *testing.T) {
	l := getSlog("xor", "Error")
	if l == nil {
		t.Fatal("getSlog returned nil")
	}
}
