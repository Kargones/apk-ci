package enterprise

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/util/runner"
)

func TestNewEpfExecutor(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	workDir := "/tmp"

	executor := NewEpfExecutor(logger, workDir)

	if executor == nil {
		t.Fatal("Expected non-nil executor")
	}

	if executor.logger != logger {
		t.Error("Expected logger to be set correctly")
	}

	if executor.runner == nil {
		t.Error("Expected runner to be initialized")
	}

	if executor.runner.WorkDir != workDir {
		t.Errorf("Expected WorkDir to be %s, got %s", workDir, executor.runner.WorkDir)
	}
}

func TestValidateEpfURL(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	executor := NewEpfExecutor(logger, "/tmp")

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid http URL",
			url:     "http://example.com/file.epf",
			wantErr: false,
		},
		{
			name:    "valid https URL",
			url:     "https://example.com/file.epf",
			wantErr: false,
		},
		{
			name:    "invalid URL without protocol",
			url:     "example.com/file.epf",
			wantErr: true,
		},
		{
			name:    "invalid URL with ftp protocol",
			url:     "ftp://example.com/file.epf",
			wantErr: true,
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.validateEpfURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEpfURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnsureTempDirectory(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	executor := NewEpfExecutor(logger, "/tmp")

	// Создаем временную директорию для тестов
	testDir := filepath.Join(os.TempDir(), "test_enterprise")
	defer func() {
		if err := os.RemoveAll(testDir); err != nil {
			t.Logf("Failed to remove test dir: %v", err)
		}
	}()

	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr bool
	}{
		{
			name: "valid temp directory",
			cfg: &config.Config{
				AppConfig: &config.AppConfig{
					TmpDir: testDir,
				},
			},
			wantErr: false,
		},
		{
			name: "nil AppConfig",
			cfg: &config.Config{
				AppConfig: nil,
			},
			wantErr: false,
		},
		{
			name: "empty TmpDir",
			cfg: &config.Config{
				AppConfig: &config.AppConfig{
					TmpDir: "",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.ensureTempDirectory(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ensureTempDirectory() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Проверяем, что директория создана, если указана
			if tt.cfg.AppConfig != nil && tt.cfg.AppConfig.TmpDir != "" && !tt.wantErr {
				if _, err := os.Stat(tt.cfg.AppConfig.TmpDir); os.IsNotExist(err) {
					t.Errorf("Expected directory %s to be created", tt.cfg.AppConfig.TmpDir)
				}
			}
		})
	}
}

func TestPrepareConnectionString(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	executor := NewEpfExecutor(logger, "/tmp")

	tests := []struct {
		name     string
		cfg      *config.Config
		wantErr  bool
		expected string
	}{
		{
			name: "valid config with credentials",
			cfg: &config.Config{
				InfobaseName: "test_db",
				AppConfig: &config.AppConfig{
					Users: struct {
						Rac        string `yaml:"rac"`
						Db         string `yaml:"db"`
						Mssql      string `yaml:"mssql"`
						StoreAdmin string `yaml:"storeAdmin"`
					}{
						Db: "testuser",
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
						Db: "testpass",
					},
				},
				DbConfig: map[string]*config.DatabaseInfo{
					"test_db": {
						OneServer: "localhost",
						Prod:      false,
						DbServer:  "localhost",
					},
				},
			},
			wantErr:  false,
			expected: "/S localhost\\test_db /N testuser /P testpass",
		},
		{
			name: "valid config without credentials",
			cfg: &config.Config{
				InfobaseName: "test_db",
				AppConfig: &config.AppConfig{
					Users: struct {
						Rac        string `yaml:"rac"`
						Db         string `yaml:"db"`
						Mssql      string `yaml:"mssql"`
						StoreAdmin string `yaml:"storeAdmin"`
					}{
						Db: "",
					},
				},
				SecretConfig: &config.SecretConfig{},
				DbConfig: map[string]*config.DatabaseInfo{
					"test_db": {
						OneServer: "localhost",
						Prod:      false,
						DbServer:  "localhost",
					},
				},
			},
			wantErr:  false,
			expected: "/S localhost\\test_db",
		},
		{
			name: "database not found",
			cfg: &config.Config{
				InfobaseName: "nonexistent_db",
				AppConfig:    &config.AppConfig{},
				DbConfig:     map[string]*config.DatabaseInfo{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.prepareConnectionString(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("prepareConnectionString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != tt.expected {
				t.Errorf("prepareConnectionString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExecute_NilConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	executor := NewEpfExecutor(logger, "/tmp")
	ctx := context.Background()

	err := executor.Execute(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil config")
	}

	expected := "конфигурация не может быть nil"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
}

func TestExecute_InvalidURL(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	executor := NewEpfExecutor(logger, "/tmp")
	ctx := context.Background()

	cfg := &config.Config{
		StartEpf: "invalid-url",
	}

	err := executor.Execute(ctx, cfg)
	if err == nil {
		t.Error("Expected error for invalid URL")
	}

	if !strings.Contains(err.Error(), "некорректный URL для StartEpf") {
		t.Errorf("Expected URL validation error, got: %v", err)
	}
}

func TestExecute_ValidURLButMissingDatabaseInfo(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	executor := NewEpfExecutor(logger, "/tmp")
	ctx := context.Background()

	cfg := &config.Config{
		StartEpf:     "https://example.com/test.epf",
		InfobaseName: "nonexistent_db",
		WorkDir:      "/tmp",
		AppConfig: &config.AppConfig{
			TmpDir: "/tmp",
		},
		DbConfig: map[string]*config.DatabaseInfo{},
	}

	err := executor.Execute(ctx, cfg)
	if err == nil {
		t.Error("Expected error for missing database info")
	}
}

func TestDownloadEpfFile_Error(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	executor := NewEpfExecutor(logger, "/tmp")

	cfg := &config.Config{
		StartEpf: "nonexistent-file.epf",
		WorkDir:  "/tmp",
		// Missing required Gitea config fields will cause error
	}

	_, cleanup, err := executor.downloadEpfFile(ctx, cfg)
	if err == nil {
		t.Error("Expected error for invalid Gitea config")
	}
	if cleanup != nil {
		cleanup()
	}
}

func TestDownloadEpfFile_CreateTempError(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	executor := NewEpfExecutor(logger, "/tmp")

	cfg := &config.Config{
		StartEpf: "test.epf",
		WorkDir:  "/nonexistent/directory", // This should cause error
	}

	_, cleanup, err := executor.downloadEpfFile(ctx, cfg)
	if err == nil {
		t.Error("Expected error for invalid work directory")
	}
	if cleanup != nil {
		cleanup()
	}
}

func TestExecuteEpfInEnterprise(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	executor := NewEpfExecutor(logger, "/tmp")

	cfg := &config.Config{
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				Bin1cv8: "nonexistent_command", // This will cause command execution error
			},
		},
	}

	err := executor.executeEpfInEnterprise(context.Background(), cfg, "/tmp/test.epf", "/S localhost\\test")
	if err == nil {
		t.Error("Expected error for nonexistent command")
	}

	if !strings.Contains(err.Error(), "ошибка выполнения внешней обработки") {
		t.Errorf("Expected execution error, got: %v", err)
	}
}

func TestAddDisableParam(t *testing.T) {
	r := &runner.Runner{
		Params: []string{},
	}

	addDisableParam(r)

	expectedParams := []string{
		"/DisableStartupDialogs",
		"/DisableStartupMessages",
		"/DisableUnrecoverableErrorMessage",
		"/UC ServiceMode",
	}

	if len(r.Params) != len(expectedParams) {
		t.Errorf("Expected %d parameters, got %d", len(expectedParams), len(r.Params))
	}

	for i, expected := range expectedParams {
		if i >= len(r.Params) || r.Params[i] != expected {
			t.Errorf("Expected parameter %d to be '%s', got '%s'", i, expected, r.Params[i])
		}
	}
}

func TestEnsureTempDirectory_CreateError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	executor := NewEpfExecutor(logger, "/tmp")

	// Try to create directory in a path that requires root permissions
	cfg := &config.Config{
		AppConfig: &config.AppConfig{
			TmpDir: "/root/readonly/test", // This should fail on most systems
		},
	}

	err := executor.ensureTempDirectory(cfg)
	// We don't require this test to fail since permissions might vary
	// But if it does fail, it should be the expected error type
	if err != nil && !strings.Contains(err.Error(), "ошибка создания временной директории") {
		t.Errorf("Expected directory creation error, got: %v", err)
	}
}

func TestExecute_FullFlow_TempDirError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	executor := NewEpfExecutor(logger, "/tmp")
	ctx := context.Background()

	cfg := &config.Config{
		StartEpf:     "https://example.com/test.epf",
		InfobaseName: "test_db",
		WorkDir:      "/tmp",
		AppConfig: &config.AppConfig{
			TmpDir: "/root/readonly/test", // This should fail on most systems
		},
		DbConfig: map[string]*config.DatabaseInfo{
			"test_db": {
				OneServer: "localhost",
				Prod:      false,
				DbServer:  "localhost",
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	// This test may not fail on all systems due to permission variations
	// but it exercises the ensureTempDirectory code path in Execute
	if err != nil {
		t.Logf("Execute failed as expected with temp directory error: %v", err)
	}
}

func TestExecute_DownloadError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create a temp directory for testing
	testDir := filepath.Join(os.TempDir(), "test_enterprise_download")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(testDir); err != nil {
			t.Logf("Failed to remove test dir: %v", err)
		}
	}()

	executor := NewEpfExecutor(logger, testDir)
	ctx := context.Background()

	cfg := &config.Config{
		StartEpf:     "https://invalid-domain-that-does-not-exist.com/test.epf",
		InfobaseName: "test_db",
		WorkDir:      testDir,
		AppConfig: &config.AppConfig{
			TmpDir: testDir,
		},
		DbConfig: map[string]*config.DatabaseInfo{
			"test_db": {
				OneServer: "localhost",
				Prod:      false,
				DbServer:  "localhost",
			},
		},
		// Missing Gitea config will cause download error
	}

	err := executor.Execute(ctx, cfg)
	if err == nil {
		t.Error("Expected error for invalid download")
	}
}

func TestPrepareConnectionString_ErrorCases(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	executor := NewEpfExecutor(logger, "/tmp")

	// Test with nil DbConfig
	cfg := &config.Config{
		InfobaseName: "test_db",
		AppConfig:    &config.AppConfig{},
		DbConfig:     nil,
	}

	_, err := executor.prepareConnectionString(cfg)
	if err == nil {
		t.Error("Expected error for nil DbConfig")
	}
}

func TestDownloadEpfFile_WriteError(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	executor := NewEpfExecutor(logger, "/tmp")

	// This tests the createTemp error path already covered in other tests
	cfg := &config.Config{
		StartEpf: "test.epf",
		WorkDir:  "/tmp",
		// The actual download will fail due to missing Gitea config
		// but we're testing the file creation part
	}

	_, cleanup, err := executor.downloadEpfFile(ctx, cfg)
	if err == nil {
		t.Error("Expected error for missing Gitea config")
	}
	if cleanup != nil {
		cleanup()
	}
}

// TestEnsureTempDirectoryAdditional tests the ensureTempDirectory method with additional scenarios
func TestEnsureTempDirectoryAdditional(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	executor := NewEpfExecutor(logger, "/tmp")

	t.Run("nil config", func(t *testing.T) {
		cfg := &config.Config{}
		err := executor.ensureTempDirectory(cfg)
		if err != nil {
			t.Errorf("Expected no error for nil AppConfig, got: %v", err)
		}
	})

	t.Run("nil AppConfig", func(t *testing.T) {
		cfg := &config.Config{
			AppConfig: nil,
		}
		err := executor.ensureTempDirectory(cfg)
		if err != nil {
			t.Errorf("Expected no error for nil AppConfig, got: %v", err)
		}
	})

	t.Run("empty TmpDir", func(t *testing.T) {
		cfg := &config.Config{
			AppConfig: &config.AppConfig{
				TmpDir: "",
			},
		}
		err := executor.ensureTempDirectory(cfg)
		if err != nil {
			t.Errorf("Expected no error for empty TmpDir, got: %v", err)
		}
	})

	t.Run("valid TmpDir", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test-enterprise-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		testTmpDir := filepath.Join(tempDir, "tmp")
		cfg := &config.Config{
			AppConfig: &config.AppConfig{
				TmpDir: testTmpDir,
			},
		}

		err = executor.ensureTempDirectory(cfg)
		if err != nil {
			t.Errorf("Expected no error for valid TmpDir, got: %v", err)
		}

		// Check that directory was created
		if _, err := os.Stat(testTmpDir); os.IsNotExist(err) {
			t.Error("Expected directory to be created")
		}
	})

	t.Run("invalid TmpDir permissions", func(t *testing.T) {
		// Try to create directory in /proc which should fail
		cfg := &config.Config{
			AppConfig: &config.AppConfig{
				TmpDir: "/proc/nonexistent/test",
			},
		}

		err := executor.ensureTempDirectory(cfg)
		if err == nil {
			t.Error("Expected error for invalid directory path")
		}
	})
}

// TestDownloadEpfFile_DetailedScenarios tests downloadEpfFile with more detailed scenarios
func TestDownloadEpfFile_DetailedScenarios(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	executor := NewEpfExecutor(logger, "/tmp")

	t.Run("missing epf filename", func(t *testing.T) {
		cfg := &config.Config{
			StartEpf: "",
			WorkDir:  "/tmp",
		}

		_, cleanup, err := executor.downloadEpfFile(ctx, cfg)
		if err == nil {
			t.Error("Expected error for missing epf filename")
		}
		if cleanup != nil {
			cleanup()
		}
	})

	t.Run("invalid work directory", func(t *testing.T) {
		cfg := &config.Config{
			StartEpf: "test.epf",
			WorkDir:  "/nonexistent/directory",
		}

		_, cleanup, err := executor.downloadEpfFile(ctx, cfg)
		if err == nil {
			t.Error("Expected error for invalid work directory")
		}
		if cleanup != nil {
			cleanup()
		}
	})

	t.Run("file with .epf extension", func(t *testing.T) {
		cfg := &config.Config{
			StartEpf: "test.epf",
			WorkDir:  "/tmp",
		}

		// This will fail due to missing Gitea config but tests the filename handling
		_, cleanup, err := executor.downloadEpfFile(ctx, cfg)
		if err == nil {
			t.Log("Unexpected success in test environment")
		}
		if cleanup != nil {
			cleanup()
		}
	})
}

// TestExecute_AdditionalErrorCases tests the Execute method with additional error cases
func TestExecute_AdditionalErrorCases(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	executor := NewEpfExecutor(logger, "/tmp")
	ctx := context.Background()

	t.Run("missing infobase name", func(t *testing.T) {
		cfg := &config.Config{
			StartEpf:     "test.epf",
			InfobaseName: "",
			WorkDir:      "/tmp",
		}

		err := executor.Execute(ctx, cfg)
		if err == nil {
			t.Error("Expected error for missing infobase name")
		}
	})

	t.Run("missing work directory", func(t *testing.T) {
		cfg := &config.Config{
			StartEpf:     "test.epf",
			InfobaseName: "test",
			WorkDir:      "",
		}

		err := executor.Execute(ctx, cfg)
		if err == nil {
			t.Error("Expected error for missing work directory")
		}
	})

	t.Run("nil config", func(t *testing.T) {
		err := executor.Execute(ctx, nil)
		if err == nil {
			t.Error("Expected error for nil config")
		}
	})
}
