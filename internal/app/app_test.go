package app

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/stretchr/testify/assert"
)

// TestInitGit тестирует инициализацию Git
func TestInitGit(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
		expectNil   bool
	}{
		{
			name: "valid config",
			config: &config.Config{
				GiteaURL:    "https://git.example.com",
				AccessToken: "test-token",
				Owner:       "test-owner",
				Repo:        "test-repo",
				RepPath:     "/tmp/test",
				BaseBranch:  "main",
				WorkDir:     "/tmp/work",
				GitConfig: &config.GitConfig{
					Timeout: 30 * time.Second,
				},
			},
			expectError: false,
			expectNil:   false,
		},
		{
			name: "empty gitea url",
			config: &config.Config{
				GiteaURL:    "",
				AccessToken: "test-token",
				Owner:       "test-owner",
				Repo:        "test-repo",
				GitConfig: &config.GitConfig{
					Timeout: 30 * time.Second,
				},
			},
			expectError: false,
			expectNil:   false, // RepURL будет "//test-owner/test-repo.git", не пустой
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			git, err := InitGit(logger, tt.config)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.expectNil {
				assert.Nil(t, git)
			} else {
				assert.NotNil(t, git)
				if git != nil {
					assert.Equal(t, tt.config.RepPath, git.RepPath)
					assert.Equal(t, tt.config.BaseBranch, git.Branch)
					assert.Equal(t, tt.config.AccessToken, git.Token)
					assert.Equal(t, tt.config.WorkDir, git.WorkDir)
					// Проверяем, что URL репозитория сформирован правильно
					if tt.config.GiteaURL != "" {
						expectedURL := "https://test-token:@git.example.com/test-owner/test-repo.git"
						assert.Equal(t, expectedURL, git.RepURL)
					} else {
						// При пустом GiteaURL RepURL будет "/test-owner/test-repo.git"
						expectedURL := "/test-owner/test-repo.git"
						assert.Equal(t, expectedURL, git.RepURL)
					}
				}
			}
		})
	}
}

// TestNetHaspInit тестирует функцию NetHaspInit
func TestNetHaspInit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	// Функция не должна паниковать
	assert.NotPanics(t, func() {
		NetHaspInit(&ctx, logger)
	})
}

// TestCreateTempDbWrapper тестирует функцию CreateTempDbWrapper
func TestCreateTempDbWrapper(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name: "nil config",
			config: nil,
			expectError: true,
		},
		{
			name: "valid config",
			config: &config.Config{
				TmpDir:   "/tmp",
				AddArray: []string{},
			},
			expectError: true, // Ожидаем ошибку из-за отсутствия реальной базы данных
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil && !tt.expectError {
					t.Errorf("Unexpected panic: %v", r)
				}
			}()

			result, err := CreateTempDbWrapper(&ctx, logger, tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)
			}
		})
	}
}

func TestActionMenuBuildWrapper(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *config.Config
		shouldErr bool
	}{
		{
			name: "valid_config",
			cfg: &config.Config{
				TmpDir: "/tmp",
			},
			shouldErr: true, // Ожидаем ошибку из-за отсутствия реальной базы данных
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

			defer func() {
				if r := recover(); r != nil && !tt.shouldErr {
					t.Errorf("Unexpected panic: %v", r)
				}
			}()

			err := ActionMenuBuildWrapper(&ctx, logger, tt.cfg)
			if (err != nil) != tt.shouldErr {
				t.Errorf("ActionMenuBuildWrapper() error = %v, shouldErr %v", err, tt.shouldErr)
			}
		})
	}
}

func TestCreateStoresWrapper(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *config.Config
		shouldErr bool
	}{
		{
			name: "valid_config",
			cfg: &config.Config{
				TmpDir: "/tmp",
			},
			shouldErr: true, // Ожидаем ошибку из-за отсутствия реальной базы данных
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

			defer func() {
				if r := recover(); r != nil && !tt.shouldErr {
					t.Errorf("Unexpected panic: %v", r)
				}
			}()

			err := CreateStoresWrapper(&ctx, logger, tt.cfg)
			if (err != nil) != tt.shouldErr {
				t.Errorf("CreateStoresWrapper() error = %v, shouldErr %v", err, tt.shouldErr)
			}
		})
	}
}

// TestSQProjectUpdate тестирует функцию SQProjectUpdate
func TestSQProjectUpdate(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name: "nil config",
			config: nil,
			expectError: true,
		},
		{
			name: "valid config",
			config: &config.Config{
				Owner: "test-owner",
				Repo:  "test-repo",
				AppConfig: &config.AppConfig{
					WorkDir: "/tmp",
					SonarQube: config.SonarQubeConfig{
						URL: "http://localhost:9000",
					},
					Scanner: config.ScannerConfig{
						ScannerURL: "http://localhost:9000",
					},
				},
				SecretConfig: &config.SecretConfig{
					SonarQube: struct {
						Token string `yaml:"token"`
					}{
						Token: "test-token",
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil && !tt.expectError {
					t.Errorf("Unexpected panic: %v", r)
				}
			}()

			err := SQProjectUpdate(&ctx, logger, tt.config)

			if tt.expectError {
				if err == nil && tt.config == nil {
					// Ожидаем panic для nil config
					t.Error("Expected panic for nil config")
				} else {
					assert.Error(t, err)
				}
			} else {
				// Функция может вернуть ошибку из-за отсутствия SonarQube конфигурации
				// но не должна паниковать
				t.Logf("SQProjectUpdate result: %v", err)
			}
		})
	}
}

// TestSQReportBranch тестирует функцию SQReportBranch
func TestSQReportBranch(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name: "nil config",
			config: nil,
			expectError: true,
		},
		{
			name: "valid config",
			config: &config.Config{
				Owner: "test-owner",
				Repo:  "test-repo",
				AppConfig: &config.AppConfig{
					WorkDir: "/tmp",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil && !tt.expectError {
					t.Errorf("Unexpected panic: %v", r)
				}
			}()

			err := SQReportBranch(&ctx, logger, tt.config)

			if tt.expectError {
				if err == nil && tt.config == nil {
					// Ожидаем panic для nil config
					t.Error("Expected panic for nil config")
				} else {
					assert.Error(t, err)
				}
			} else {
				// Функция может вернуть ошибку из-за отсутствия SonarQube конфигурации
				// но не должна паниковать
				t.Logf("SQReportBranch result: %v", err)
			}
		})
	}
}

func TestSQScanBranch(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *config.Config
		commitHash string
		shouldErr  bool
	}{
		{
			name: "valid_config",
			cfg: &config.Config{
				Owner:        "test-owner",
				Repo:         "test-repo",
				BranchForScan: "main", // Указываем валидную ветку для сканирования
			},
			commitHash: "abc123",
			shouldErr:  true, // Ожидаем ошибку из-за отсутствия реального SonarQube
		},
		{
			name: "invalid_branch",
			cfg: &config.Config{
				Owner:        "test-owner",
				Repo:         "test-repo",
				BranchForScan: "feature-branch", // Невалидная ветка
			},
			commitHash: "abc123",
			shouldErr:  false, // Не ожидаем ошибку, сканирование просто пропускается
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

			defer func() {
				if r := recover(); r != nil && !tt.shouldErr {
					t.Errorf("Unexpected panic: %v", r)
				}
			}()

			err := SQScanBranch(&ctx, logger, tt.cfg, tt.commitHash)
			if (err != nil) != tt.shouldErr {
				t.Errorf("SQScanBranch() error = %v, shouldErr %v", err, tt.shouldErr)
			}
		})
	}
}

// TestIsValidBranchForScanning тестирует функцию isValidBranchForScanning
func TestIsValidBranchForScanning(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		expected bool
	}{
		{"main branch", "main", true},
		{"valid t-branch with 6 digits", "t123456", true},
		{"valid t-branch with 7 digits", "t1234567", true},
		{"invalid t-branch with 5 digits", "t12345", false},
		{"invalid t-branch with 8 digits", "t12345678", false},
		{"invalid t-branch with letters", "t123abc", false},
		{"invalid branch not starting with t", "feature-123456", false},
		{"invalid empty branch", "", false},
		{"invalid only t", "t", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidBranchForScanning(tt.branch)
			if result != tt.expected {
				t.Errorf("isValidBranchForScanning(%s) = %v, expected %v", tt.branch, result, tt.expected)
			}
		})
	}
}

// TestSQScanPR тестирует функцию SQScanPR
func TestSQScanPR(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name: "nil config",
			config: nil,
			expectError: true,
		},
		{
			name: "valid config",
			config: &config.Config{
				Owner: "test-owner",
				Repo:  "test-repo",
				AppConfig: &config.AppConfig{
					WorkDir: "/tmp",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil && !tt.expectError {
					t.Errorf("Unexpected panic: %v", r)
				}
			}()

			err := SQScanPR(&ctx, logger, tt.config)

			if tt.expectError {
				if err == nil && tt.config == nil {
					// Ожидаем panic для nil config
					t.Error("Expected panic for nil config")
				} else {
					assert.Error(t, err)
				}
			} else {
				// Функция может вернуть ошибку из-за отсутствия SonarQube конфигурации
				// но не должна паниковать
				t.Logf("SQScanPR result: %v", err)
			}
		})
	}
}

// TestConvert тестирует функцию Convert
func TestConvert(t *testing.T) {
	// Создаем временную директорию для тестов
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name: "missing work directory",
			config: &config.Config{
				WorkDir: "/nonexistent/directory",
			},
			expectError: true,
		},
		{
			name: "valid work directory but no git config",
			config: &config.Config{
				WorkDir: tempDir,
				// Не указываем GiteaURL, что приведет к panic в Convert
				// так как функция не проверяет nil после InitGit
			},
			expectError: true, // Ожидаем panic/ошибку из-за nil pointer
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			
			// Для случая с nil git используем defer для перехвата panic
			if tt.name == "valid work directory but no git config" {
				defer func() {
					if r := recover(); r != nil {
						// Ожидаем panic из-за nil pointer
						t.Logf("Получен ожидаемый panic: %v", r)
					}
				}()
			}
			
			err := Convert(&ctx, logger, tt.config)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestServiceModeEnable тестирует функцию ServiceModeEnable
func TestServiceModeEnable(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	tests := []struct {
		name               string
		config             *config.Config
		infobaseName       string
		terminateSessions  bool
		expectError        bool
	}{
		{
			name: "valid config",
			config: &config.Config{
				AppConfig: &config.AppConfig{
					WorkDir: "/tmp",
				},
				DbConfig: map[string]*config.DatabaseInfo{
					"test": {
						OneServer: "localhost:1541",
						DbServer:  "localhost",
					},
				},
			},
			infobaseName:      "test-infobase",
			terminateSessions: false,
			expectError:       true, // Ожидаем ошибку из-за отсутствия реального RAC
		},
		{
			name: "valid config with terminate sessions",
			config: &config.Config{
				AppConfig: &config.AppConfig{
					WorkDir: "/tmp",
				},
				DbConfig: map[string]*config.DatabaseInfo{
					"test": {
						OneServer: "localhost:1541",
						DbServer:  "localhost",
					},
				},
			},
			infobaseName:      "test-infobase",
			terminateSessions: true,
			expectError:       true, // Ожидаем ошибку из-за отсутствия реального RAC
		},
		{
			name: "empty infobase name",
			config: &config.Config{
				AppConfig: &config.AppConfig{
					WorkDir: "/tmp",
				},
				DbConfig: map[string]*config.DatabaseInfo{
					"test": {
						OneServer: "localhost:1541",
						DbServer:  "localhost",
					},
				},
			},
			infobaseName:      "",
			terminateSessions: false,
			expectError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ServiceModeEnable(&ctx, logger, tt.config, tt.infobaseName, tt.terminateSessions)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestServiceModeDisable тестирует функцию ServiceModeDisable
func TestServiceModeDisable(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	tests := []struct {
		name         string
		config       *config.Config
		infobaseName string
		expectError  bool
	}{
		{
			name: "valid config",
			config: &config.Config{
				AppConfig: &config.AppConfig{
					WorkDir: "/tmp",
				},
			},
			infobaseName: "test-infobase",
			expectError:  true, // Ожидаем ошибку из-за отсутствия реального RAC
		},
		{
			name:         "empty infobase name",
			config:       &config.Config{},
			infobaseName: "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ServiceModeDisable(&ctx, logger, tt.config, tt.infobaseName)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestServiceModeStatus тестирует функцию ServiceModeStatus
func TestServiceModeStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	tests := []struct {
		name         string
		config       *config.Config
		infobaseName string
		expectError  bool
	}{
		{
			name: "valid config",
			config: &config.Config{
				AppConfig: &config.AppConfig{
					WorkDir: "/tmp",
				},
			},
			infobaseName: "test-infobase",
			expectError:  true, // Ожидаем ошибку из-за отсутствия реального RAC
		},
		{
			name:         "empty infobase name",
			config:       &config.Config{},
			infobaseName: "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ServiceModeStatus(&ctx, logger, tt.config, tt.infobaseName)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestActServiceModeEnable тестирует функцию ActServiceModeEnable
func TestActServiceModeEnable(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	tests := []struct {
		name               string
		config             *config.Config
		dbName             string
		terminateSessions  bool
		expectError        bool
	}{
		{
			name: "valid config",
			config: &config.Config{
				AppConfig: &config.AppConfig{
					WorkDir: "/tmp",
				},
			},
			dbName:            "test-db",
			terminateSessions: false,
			expectError:       true, // Ожидаем ошибку из-за отсутствия реального RAC
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ActServiceModeEnable(&ctx, logger, tt.config, tt.dbName, tt.terminateSessions)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestActServiceModeDisable тестирует функцию ActServiceModeDisable
func TestActServiceModeDisable(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	tests := []struct {
		name        string
		config      *config.Config
		dbName      string
		expectError bool
	}{
		{
			name: "valid config",
			config: &config.Config{
				AppConfig: &config.AppConfig{
					WorkDir: "/tmp",
				},
			},
			dbName:      "test-db",
			expectError: true, // Ожидаем ошибку из-за отсутствия реального RAC
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ActServiceModeDisable(&ctx, logger, tt.config, tt.dbName)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestActServiceModeStatus тестирует функцию ActServiceModeStatus
func TestActServiceModeStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	tests := []struct {
		name        string
		config      *config.Config
		dbName      string
		expectError bool
	}{
		{
			name: "valid config",
			config: &config.Config{
				AppConfig: &config.AppConfig{
					WorkDir: "/tmp",
				},
			},
			dbName:      "test-db",
			expectError: true, // Ожидаем ошибку из-за отсутствия реального RAC
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ActServiceModeStatus(&ctx, logger, tt.config, tt.dbName, "")

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestGit2Store тестирует функцию Git2Store
func TestGit2Store(t *testing.T) {
	tempDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name: "invalid work directory",
			config: &config.Config{
				WorkDir: "/nonexistent/directory",
			},
			expectError: true,
		},
		{
			name: "valid work directory but no git config",
			config: &config.Config{
				WorkDir: tempDir,
				TmpDir:  tempDir,
				GitConfig: &config.GitConfig{
					Timeout: 30 * time.Second,
				},
			},
			expectError: true, // Ожидаем ошибку из-за отсутствия git конфигурации
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Git2Store(&ctx, logger, tt.config)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestStore2Db тестирует функцию Store2Db
func TestStore2Db(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name: "valid config with infobase name",
			config: &config.Config{
				InfobaseName: "test-infobase",
				AppConfig: &config.AppConfig{
					WorkDir: "/tmp",
				},
				ProjectConfig: &config.ProjectConfig{
					StoreDb: "test-store-db",
				},
			},
			expectError: true, // Ожидаем ошибку из-за отсутствия реальной конфигурации
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Store2Db(&ctx, logger, tt.config)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestStore2DbWithConfig тестирует функцию Store2DbWithConfig
func TestStore2DbWithConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name: "valid config",
			config: &config.Config{
				AppConfig: &config.AppConfig{
					WorkDir: "/tmp",
				},
			},
			expectError: true, // Ожидаем ошибку из-за отсутствия реальной конфигурации
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Store2DbWithConfig(&ctx, logger, tt.config)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestStoreBind тестирует функцию StoreBind
func TestStoreBind(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name: "valid config",
			config: &config.Config{
				AppConfig: &config.AppConfig{
					WorkDir: "/tmp",
				},
			},
			expectError: true, // Ожидаем ошибку из-за отсутствия реальной конфигурации
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := StoreBind(&ctx, logger, tt.config)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestDbUpdate тестирует функцию DbUpdate
func TestDbUpdate(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name: "valid config with infobase name",
			config: &config.Config{
				InfobaseName: "test-infobase",
				AppConfig: &config.AppConfig{
					WorkDir: "/tmp",
				},
				ProjectConfig: &config.ProjectConfig{
					StoreDb: "test-store-db",
				},
			},
			expectError: true, // Ожидаем ошибку из-за отсутствия реальной конфигурации
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DbUpdate(&ctx, logger, tt.config)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestDbUpdateWithConfig тестирует функцию DbUpdateWithConfig
func TestDbUpdateWithConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name: "valid config",
			config: &config.Config{
				AppConfig: &config.AppConfig{
					WorkDir: "/tmp",
				},
				ProjectConfig: &config.ProjectConfig{
					StoreDb: "test-store-db",
				},
			},
			expectError: true, // Ожидаем ошибку из-за отсутствия реальной конфигурации
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DbUpdateWithConfig(&ctx, logger, tt.config)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}


// TestDbRestoreWithConfig тестирует функцию DbRestoreWithConfig
func TestDbRestoreWithConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	tests := []struct {
		name        string
		config      *config.Config
		dbname      string
		expectError bool
	}{
		{
			name: "valid config with db name",
			config: &config.Config{
				AppConfig: &config.AppConfig{
					WorkDir: "/tmp",
				},
			},
			dbname:      "test-db",
			expectError: true, // Ожидаем ошибку из-за отсутствия реальной конфигурации БД
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DbRestoreWithConfig(&ctx, logger, tt.config, tt.dbname)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestTestMerge тестирует функцию TestMerge
func TestTestMerge(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name: "valid config",
			config: &config.Config{
				Owner: "test-owner",
				Repo:  "test-repo",
				AppConfig: &config.AppConfig{
					WorkDir: "/tmp",
				},
			},
			expectError: true, // Ожидаем ошибку из-за отсутствия реального Gitea
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := TestMerge(&ctx, logger, tt.config)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestHasRelevantChangesInCommit тестирует функцию hasRelevantChangesInCommit (private function test via shouldRunScanBranch)
func TestShouldRunScanBranch(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name           string
		config         *config.Config
		branch         string
		commitHash     string
		expectedResult bool
		expectError    bool
	}{
		{
			name:           "invalid branch",
			config:         &config.Config{},
			branch:         "feature-branch",
			commitHash:     "abc123",
			expectedResult: false,
			expectError:    false,
		},
		{
			name:           "valid main branch",
			config:         &config.Config{},
			branch:         "main",
			commitHash:     "",
			expectedResult: true,
			expectError:    false,
		},
		{
			name:           "valid t-branch",
			config:         &config.Config{},
			branch:         "t123456",
			commitHash:     "",
			expectedResult: true,
			expectError:    false,
		},
		{
			name: "valid main branch with commit but no gitea config",
			config: &config.Config{
				GiteaURL: "http://localhost",
				Owner:    "test",
				Repo:     "test",
			},
			branch:         "main",
			commitHash:     "abc123",
			expectedResult: false,
			expectError:    true, // Ожидаем ошибку из-за отсутствия реального Gitea
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := shouldRunScanBranch(logger, tt.config, tt.branch, tt.commitHash)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedResult, result)
		})
	}
}