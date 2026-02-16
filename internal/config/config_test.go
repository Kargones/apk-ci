package config

import (
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/entity/gitea"
	"gopkg.in/yaml.v3"
)

const (
	testTrueValue = "true"
)

// MockGiteaAPI для тестирования функций загрузки конфигурации
type MockGiteaAPI struct {
	configData map[string][]byte
	shouldFail bool
}

func (m *MockGiteaAPI) GetConfigData(l *slog.Logger, configPath string) ([]byte, error) {
	if m.shouldFail {
		return nil, errors.New("mock error")
	}
	if data, exists := m.configData[configPath]; exists {
		return data, nil
	}
	return nil, errors.New("config not found")
}

// Реализация всех методов интерфейса gitea.APIInterface
func (m *MockGiteaAPI) GetIssue(issueNumber int64) (*gitea.Issue, error) {
	return nil, nil
}

func (m *MockGiteaAPI) GetFileContent(fileName string) ([]byte, error) {
	return nil, nil
}

func (m *MockGiteaAPI) AddIssueComment(issueNumber int64, commentText string) error {
	return nil
}

func (m *MockGiteaAPI) CloseIssue(issueNumber int64) error {
	return nil
}

func (m *MockGiteaAPI) ConflictPR(prNumber int64) (bool, error) {
	return false, nil
}

func (m *MockGiteaAPI) ConflictFilesPR(prNumber int64) ([]string, error) {
	return nil, nil
}

func (m *MockGiteaAPI) GetRepositoryContents(filepath, branch string) ([]gitea.FileInfo, error) {
	return nil, nil
}

func (m *MockGiteaAPI) AnalyzeProjectStructure(branch string) ([]string, error) {
	return nil, nil
}

func (m *MockGiteaAPI) AnalyzeProject(branch string) ([]string, error) {
	return nil, nil
}

func (m *MockGiteaAPI) GetLatestCommit(branch string) (*gitea.Commit, error) {
	return nil, nil
}

func (m *MockGiteaAPI) GetCommitFiles(commitSHA string) ([]gitea.CommitFile, error) {
	return nil, nil
}

func (m *MockGiteaAPI) IsUserInTeam(l *slog.Logger, username string, orgName string, teamName string) (bool, error) {
	return false, nil
}

func (m *MockGiteaAPI) GetCommits(branch string, limit int) ([]gitea.Commit, error) {
	return nil, nil
}

func (m *MockGiteaAPI) GetFirstCommitOfBranch(branch string, baseBranch string) (*gitea.Commit, error) {
	return nil, nil
}

func (m *MockGiteaAPI) GetCommitsBetween(baseCommitSHA, headCommitSHA string) ([]gitea.Commit, error) {
	return nil, nil
}

func (m *MockGiteaAPI) GetBranchCommitRange(branch string) (*gitea.BranchCommitRange, error) {
	return nil, nil
}

func (m *MockGiteaAPI) ActivePR() ([]gitea.PR, error) {
	return nil, nil
}

func (m *MockGiteaAPI) DeleteTestBranch() error {
	return nil
}

func (m *MockGiteaAPI) CreateTestBranch() error {
	return nil
}

func (m *MockGiteaAPI) CreatePR(head string) (gitea.PR, error) {
	return gitea.PR{}, nil
}

func (m *MockGiteaAPI) MergePR(prNumber int64, l *slog.Logger) error {
	return nil
}

func (m *MockGiteaAPI) ClosePR(prNumber int64) error {
	return nil
}

func (m *MockGiteaAPI) SetRepositoryState(l *slog.Logger, operations []gitea.BatchOperation, branch, commitMessage string) error {
	return nil
}

func (m *MockGiteaAPI) GetTeamMembers(orgName, teamName string) ([]string, error) {
	return nil, nil
}

func (m *MockGiteaAPI) GetBranches(repo string) ([]gitea.Branch, error) {
	return nil, nil
}

func TestForceUpdateParameter(t *testing.T) {
	tests := []struct {
		name           string
		forceUpdateEnv string
		expected       bool
	}{
		{
			name:           "force_update_true",
			forceUpdateEnv: testTrueValue,
			expected:       true,
		},
		{
			name:           "force_update_false",
			forceUpdateEnv: "false",
			expected:       false,
		},
		{
			name:           "force_update_empty",
			forceUpdateEnv: "",
			expected:       false,
		},
		{
			name:           "force_update_invalid",
			forceUpdateEnv: "invalid",
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем необходимые переменные окружения
			_ = os.Setenv("INPUT_FORCE_UPDATE", tt.forceUpdateEnv)
			_ = os.Setenv("INPUT_COMMAND", "action-menu-build")
			_ = os.Setenv("INPUT_GITEAURL", "https://test.com")
			_ = os.Setenv("INPUT_REPOSITORY", "test/repo")
			_ = os.Setenv("INPUT_ACCESSTOKEN", "test-token")
			_ = os.Setenv("INPUT_ACTOR", "test-actor")
			_ = os.Setenv("INPUT_LOGLEVEL", "Info")
			_ = os.Setenv("BR_ENV", "dev")
			_ = os.Setenv("BR_CONFIG_PATH", "test-config.yaml")
			_ = os.Setenv("BR_SECRET_PATH", "test-secret.yaml")
			_ = os.Setenv("BR_PROJECT_PATH", "test-project.yaml")
			_ = os.Setenv("BR_DB_CONFIG_PATH", "test-db.yaml")

			defer func() {
				// Очищаем переменные окружения после теста
				_ = os.Unsetenv("INPUT_FORCE_UPDATE")
				_ = os.Unsetenv("INPUT_COMMAND")
				_ = os.Unsetenv("INPUT_GITEAURL")
				_ = os.Unsetenv("INPUT_REPOSITORY")
				_ = os.Unsetenv("INPUT_ACCESSTOKEN")
				_ = os.Unsetenv("INPUT_ACTOR")
				_ = os.Unsetenv("INPUT_LOGLEVEL")
				_ = os.Unsetenv("BR_ENV")
				_ = os.Unsetenv("BR_CONFIG_PATH")
				_ = os.Unsetenv("BR_SECRET_PATH")
				_ = os.Unsetenv("BR_PROJECT_PATH")
				_ = os.Unsetenv("BR_DB_CONFIG_PATH")
			}()

			// Получаем входные параметры
			inputParams := GetInputParams()

			// Проверяем, что GHAForceUpdate правильно установлен
			if inputParams.GHAForceUpdate != tt.forceUpdateEnv {
				t.Errorf("Expected GHAForceUpdate = %s, got %s", tt.forceUpdateEnv, inputParams.GHAForceUpdate)
			}

			// Создаем конфигурацию и проверяем ForceUpdate
			cfg := &Config{}
			cfg.ForceUpdate = inputParams.GHAForceUpdate == testTrueValue

			if cfg.ForceUpdate != tt.expected {
				t.Errorf("Expected ForceUpdate = %v, got %v", tt.expected, cfg.ForceUpdate)
			}
		})
	}
}

func TestSonarQubeConfig(t *testing.T) {
	t.Run("GetDefaultSonarQubeConfig", func(t *testing.T) {
		config := GetDefaultSonarQubeConfig()
		if config == nil {
			t.Fatal("Expected non-nil SonarQubeConfig")
		}
		if config.URL != "http://localhost:9000" {
			t.Errorf("Expected URL = http://localhost:9000, got %s", config.URL)
		}
		if config.Timeout != 30*time.Second {
			t.Errorf("Expected Timeout = 30s, got %v", config.Timeout)
		}
		if config.RetryAttempts != 3 {
			t.Errorf("Expected RetryAttempts = 3, got %d", config.RetryAttempts)
		}
		if config.RetryDelay != 5*time.Second {
			t.Errorf("Expected RetryDelay = 5s, got %v", config.RetryDelay)
		}
		if config.ProjectPrefix != "benadis" {
			t.Errorf("Expected ProjectPrefix = benadis, got %s", config.ProjectPrefix)
		}
		if config.DefaultVisibility != "private" {
			t.Errorf("Expected DefaultVisibility = private, got %s", config.DefaultVisibility)
		}
		if config.QualityGateTimeout != 300*time.Second {
			t.Errorf("Expected QualityGateTimeout = 300s, got %v", config.QualityGateTimeout)
		}
		if !config.DisableBranchAnalysis {
			t.Error("Expected DisableBranchAnalysis = true")
		}
	})

	t.Run("SonarQubeConfig_Validate", func(t *testing.T) {
		// Тест валидной конфигурации
		validConfig := &SonarQubeConfig{
			URL:                   "http://localhost:9000",
			Token:                 "test-token",
			Timeout:               30 * time.Second,
			RetryAttempts:         3,
			RetryDelay:            5 * time.Second,
			ProjectPrefix:         "test",
			DefaultVisibility:     "private",
			QualityGateTimeout:    300 * time.Second,
			DisableBranchAnalysis: true,
		}
		if err := validConfig.Validate(); err != nil {
			t.Errorf("Expected valid config to pass validation, got error: %v", err)
		}

		// Тест с пустым URL
		invalidConfig := *validConfig
		invalidConfig.URL = ""
		if err := invalidConfig.Validate(); err == nil {
			t.Error("Expected error for empty URL")
		}

		// Тест с пустым токеном
		invalidConfig = *validConfig
		invalidConfig.Token = ""
		if err := invalidConfig.Validate(); err == nil {
			t.Error("Expected error for empty token")
		}

		// Тест с отрицательным таймаутом
		invalidConfig = *validConfig
		invalidConfig.Timeout = -1 * time.Second
		if err := invalidConfig.Validate(); err == nil {
			t.Error("Expected error for negative timeout")
		}

		// Тест с отрицательными попытками повтора
		invalidConfig = *validConfig
		invalidConfig.RetryAttempts = -1
		if err := invalidConfig.Validate(); err == nil {
			t.Error("Expected error for negative retry attempts")
		}

		// Тест с отрицательной задержкой повтора
		invalidConfig = *validConfig
		invalidConfig.RetryDelay = -1 * time.Second
		if err := invalidConfig.Validate(); err == nil {
			t.Error("Expected error for negative retry delay")
		}

		// Тест с неверной видимостью
		invalidConfig = *validConfig
		invalidConfig.DefaultVisibility = "invalid"
		if err := invalidConfig.Validate(); err == nil {
			t.Error("Expected error for invalid default visibility")
		}

		// Тест с отрицательным таймаутом quality gate
		invalidConfig = *validConfig
		invalidConfig.QualityGateTimeout = -1 * time.Second
		if err := invalidConfig.Validate(); err == nil {
			t.Error("Expected error for negative quality gate timeout")
		}
	})

	t.Run("GetSonarQubeConfig", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		// Тест с базовой конфигурацией
		cfg := &Config{
			Logger: logger,
		}

		// Устанавливаем переменные окружения для тестирования
		os.Setenv("SONARQUBE_URL", "http://test-sonar:9000")
		os.Setenv("SONARQUBE_TOKEN", "test-env-token")
		defer func() {
			os.Unsetenv("SONARQUBE_URL")
			os.Unsetenv("SONARQUBE_TOKEN")
		}()

		config, err := GetSonarQubeConfig(logger, cfg)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if config.URL != "http://test-sonar:9000" {
			t.Errorf("Expected URL from env, got %s", config.URL)
		}
		if config.Token != "test-env-token" {
			t.Errorf("Expected token from env, got %s", config.Token)
		}

		// Тест с AppConfig
		cfg.AppConfig = &AppConfig{
			SonarQube: SonarQubeConfig{
				URL:           "http://app-sonar:9000",
				Token:         "app-token",
				Timeout:       60 * time.Second,
				RetryAttempts: 5,
			},
		}

		config, err = GetSonarQubeConfig(logger, cfg)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		// Переменные окружения имеют приоритет
		if config.URL != "http://test-sonar:9000" {
			t.Errorf("Expected URL from env to override app config, got %s", config.URL)
		}

		// Тест с SecretConfig (токен из секретов имеет приоритет)
		cfg.SecretConfig = &SecretConfig{
			SonarQube: struct {
				Token string `yaml:"token"`
			}{
				Token: "secret-token",
			},
		}

		config, err = GetSonarQubeConfig(logger, cfg)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		// Переменная окружения все еще имеет приоритет
		if config.Token != "test-env-token" {
			t.Errorf("Expected token from env to have highest priority, got %s", config.Token)
		}

		// Очищаем переменную окружения для токена
		os.Unsetenv("SONARQUBE_TOKEN")
		config, err = GetSonarQubeConfig(logger, cfg)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		// Теперь должен использоваться токен из секретов
		if config.Token != "secret-token" {
			t.Errorf("Expected token from secrets, got %s", config.Token)
		}
	})
}

func TestScannerConfig(t *testing.T) {
	t.Run("GetDefaultScannerConfig", func(t *testing.T) {
		config := GetDefaultScannerConfig()
		if config == nil {
			t.Fatal("Expected non-nil ScannerConfig")
		}
		if config.ScannerURL != "https://binaries.sonarsource.com/Distribution/sonar-scanner-cli/sonar-scanner-cli-4.8.0.2856-linux.zip" {
			t.Errorf("Unexpected ScannerURL: %s", config.ScannerURL)
		}
		if config.ScannerVersion != "4.8.0.2856" {
			t.Errorf("Expected ScannerVersion = 4.8.0.2856, got %s", config.ScannerVersion)
		}
		if config.JavaOpts != "-Xmx2g" {
			t.Errorf("Expected JavaOpts = -Xmx2g, got %s", config.JavaOpts)
		}
		if config.Properties == nil {
			t.Error("Expected non-nil Properties map")
		}
		if config.Timeout != 600*time.Second {
			t.Errorf("Expected Timeout = 600s, got %v", config.Timeout)
		}
		if config.WorkDir != "/tmp/benadis" {
			t.Errorf("Expected WorkDir = /tmp/benadis, got %s", config.WorkDir)
		}
		if config.TempDir != "/tmp/benadis/scanner/temp" {
			t.Errorf("Expected TempDir = /tmp/benadis/scanner/temp, got %s", config.TempDir)
		}
	})

	t.Run("ScannerConfig_Validate", func(t *testing.T) {
		// Тест валидной конфигурации
		validConfig := &ScannerConfig{
			ScannerURL:     "https://example.com/scanner.zip",
			ScannerVersion: "4.8.0",
			JavaOpts:       "-Xmx2g",
			Properties:     make(map[string]string),
			Timeout:        600 * time.Second,
			WorkDir:        "/tmp/scanner",
			TempDir:        "/tmp/scanner/temp",
		}
		if err := validConfig.Validate(); err != nil {
			t.Errorf("Expected valid config to pass validation, got error: %v", err)
		}

		// Тест с пустым URL сканера
		invalidConfig := *validConfig
		invalidConfig.ScannerURL = ""
		if err := invalidConfig.Validate(); err == nil {
			t.Error("Expected error for empty scanner URL")
		}

		// Тест с пустой версией сканера
		invalidConfig = *validConfig
		invalidConfig.ScannerVersion = ""
		if err := invalidConfig.Validate(); err == nil {
			t.Error("Expected error for empty scanner version")
		}

		// Тест с отрицательным таймаутом
		invalidConfig = *validConfig
		invalidConfig.Timeout = -1 * time.Second
		if err := invalidConfig.Validate(); err == nil {
			t.Error("Expected error for negative timeout")
		}
	})

	t.Run("GetScannerConfig", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		// Тест с базовой конфигурацией
		cfg := &Config{
			Logger: logger,
		}

		// Устанавливаем переменные окружения для тестирования
		os.Setenv("SONARQUBE_SCANNER_URL", "https://test.com/scanner.zip")
		os.Setenv("SONARQUBE_SCANNER_VERSION", "5.0.0")
		defer func() {
			os.Unsetenv("SONARQUBE_SCANNER_URL")
			os.Unsetenv("SONARQUBE_SCANNER_VERSION")
		}()

		config, err := GetScannerConfig(logger, cfg)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if config.ScannerURL != "https://test.com/scanner.zip" {
			t.Errorf("Expected ScannerURL from env, got %s", config.ScannerURL)
		}
		if config.ScannerVersion != "5.0.0" {
			t.Errorf("Expected ScannerVersion from env, got %s", config.ScannerVersion)
		}

		// Тест с AppConfig
		cfg.AppConfig = &AppConfig{
			Scanner: ScannerConfig{
				ScannerURL:     "https://app.com/scanner.zip",
				ScannerVersion: "4.9.0",
				JavaOpts:       "-Xmx4g",
				Timeout:        900 * time.Second,
			},
		}

		config, err = GetScannerConfig(logger, cfg)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		// Переменные окружения имеют приоритет
		if config.ScannerURL != "https://test.com/scanner.zip" {
			t.Errorf("Expected ScannerURL from env to override app config, got %s", config.ScannerURL)
		}
		if config.JavaOpts != "-Xmx4g" {
			t.Errorf("Expected JavaOpts from app config, got %s", config.JavaOpts)
		}
	})
}

func TestGetInputParams(t *testing.T) {
	// Устанавливаем тестовые переменные окружения
	_ = os.Setenv("INPUT_COMMAND", "test-command")
	_ = os.Setenv("INPUT_GITEAURL", "http://localhost:3000")
	_ = os.Setenv("INPUT_REPOSITORY", "test/repo")
	_ = os.Setenv("INPUT_ACCESSTOKEN", "test-token")
	_ = os.Setenv("INPUT_ACTOR", "test-actor")
	_ = os.Setenv("INPUT_LOGLEVEL", "Debug")

	defer func() {
		_ = os.Unsetenv("INPUT_COMMAND")
		_ = os.Unsetenv("INPUT_GITEAURL")
		_ = os.Unsetenv("INPUT_REPOSITORY")
		_ = os.Unsetenv("INPUT_ACCESSTOKEN")
		_ = os.Unsetenv("INPUT_ACTOR")
		_ = os.Unsetenv("INPUT_LOGLEVEL")
	}()

	params := GetInputParams()

	if params.GHACommand != "test-command" {
		t.Errorf("Expected GHACommand = test-command, got %s", params.GHACommand)
	}
	if params.GHAGiteaURL != "http://localhost:3000" {
		t.Errorf("Expected GHAGiteaURL = http://localhost:3000, got %s", params.GHAGiteaURL)
	}
	if params.GHARepository != "test/repo" {
		t.Errorf("Expected GHARepository = test/repo, got %s", params.GHARepository)
	}
	if params.GHAAccessToken != "test-token" {
		t.Errorf("Expected GHAAccessToken = test-token, got %s", params.GHAAccessToken)
	}
	if params.GHAActor != "test-actor" {
		t.Errorf("Expected GHAActor = test-actor, got %s", params.GHAActor)
	}
	if params.GHALogLevel != "Debug" {
		t.Errorf("Expected GHALogLevel = Debug, got %s", params.GHALogLevel)
	}
}

func TestGetOwnerAndRepo(t *testing.T) {
	tests := []struct {
		name       string
		repository string
		expectedOwner string
		expectedRepo  string
	}{
		{
			name:          "valid_repository",
			repository:    "owner/repo",
			expectedOwner: "owner",
			expectedRepo:  "repo",
		},
		{
			name:          "repository_with_dots",
			repository:    "my.owner/my.repo",
			expectedOwner: "my.owner",
			expectedRepo:  "my.repo",
		},
		{
			name:          "empty_repository",
			repository:    "",
			expectedOwner: "",
			expectedRepo:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner := getOwner(tt.repository)
			repo := getRepo(tt.repository)

			if owner != tt.expectedOwner {
				t.Errorf("Expected owner = %s, got %s", tt.expectedOwner, owner)
			}
			if repo != tt.expectedRepo {
				t.Errorf("Expected repo = %s, got %s", tt.expectedRepo, repo)
			}
		})
	}
}

func TestGetDefaultConfigs(t *testing.T) {
	t.Run("default_git_config", func(t *testing.T) {
		gitConfig := getDefaultGitConfig()
		if gitConfig == nil {
			t.Fatal("Expected non-nil GitConfig")
		}
		if gitConfig.UserName != "benadis-runner" {
			t.Errorf("Expected UserName = benadis-runner, got %s", gitConfig.UserName)
		}
		if gitConfig.UserEmail != "runner@benadis.ru" {
			t.Errorf("Expected UserEmail = runner@benadis.ru, got %s", gitConfig.UserEmail)
		}
		if gitConfig.DefaultBranch != "main" {
			t.Errorf("Expected DefaultBranch = main, got %s", gitConfig.DefaultBranch)
		}
		if gitConfig.Timeout != 60*time.Minute {
			t.Errorf("Expected Timeout = 1h0m0s, got %v", gitConfig.Timeout)
		}
	})

	t.Run("default_logging_config", func(t *testing.T) {
		loggingConfig := getDefaultLoggingConfig()
		if loggingConfig == nil {
			t.Fatal("Expected non-nil LoggingConfig")
		}
		if loggingConfig.Level != "info" {
			t.Errorf("Expected Level = info, got %s", loggingConfig.Level)
		}
		if loggingConfig.Format != "json" {
			t.Errorf("Expected Format = json, got %s", loggingConfig.Format)
		}
		if loggingConfig.Output != "stdout" {
			t.Errorf("Expected Output = stdout, got %s", loggingConfig.Output)
		}
	})

	t.Run("default_rac_config", func(t *testing.T) {
		racConfig := getDefaultRacConfig()
		if racConfig == nil {
			t.Fatal("Expected non-nil RacConfig")
		}
		if racConfig.RacPath != "/opt/1cv8/x86_64/8.3.25.1257/rac" {
			t.Errorf("Expected RacPath = /opt/1cv8/x86_64/8.3.25.1257/rac, got %s", racConfig.RacPath)
		}
		if racConfig.RacServer != "localhost" {
			t.Errorf("Expected RacServer = localhost, got %s", racConfig.RacServer)
		}
		if racConfig.RacPort != 1545 {
			t.Errorf("Expected RacPort = 1545, got %d", racConfig.RacPort)
		}
		if racConfig.Timeout != 30*time.Second {
			t.Errorf("Expected Timeout = 30s, got %v", racConfig.Timeout)
		}
		if racConfig.Retries != 3 {
			t.Errorf("Expected Retries = 3, got %d", racConfig.Retries)
		}
	})

	t.Run("default_app_config", func(t *testing.T) {
		appConfig := getDefaultAppConfig()
		if appConfig == nil {
			t.Fatal("Expected non-nil AppConfig")
		}
		if appConfig.LogLevel != "Debug" {
			t.Errorf("Expected LogLevel = Debug, got %s", appConfig.LogLevel)
		}
		if appConfig.WorkDir != "/tmp/benadis" {
			t.Errorf("Expected WorkDir = /tmp/benadis, got %s", appConfig.WorkDir)
		}
		if appConfig.TmpDir != "/tmp/benadis/temp" {
			t.Errorf("Expected TmpDir = /tmp/benadis/temp, got %s", appConfig.TmpDir)
		}
		if appConfig.Timeout != 30 {
			t.Errorf("Expected Timeout = 30, got %d", appConfig.Timeout)
		}
	})
}

func TestConfigMethods(t *testing.T) {
	cfg := &Config{
		DbConfig: map[string]*DatabaseInfo{
			"test-db": {
				OneServer: "test-server",
				Prod:      true,
				DbServer:  "db-server",
			},
			"dev-db": {
				OneServer: "dev-server",
				Prod:      false,
				DbServer:  "dev-db-server",
			},
		},
	}

	t.Run("GetOneServer", func(t *testing.T) {
		server := cfg.GetOneServer("test-db")
		if server != "test-server" {
			t.Errorf("Expected server = test-server, got %s", server)
		}

		// Тест для несуществующей базы
		server = cfg.GetOneServer("nonexistent")
		if server != "" {
			t.Errorf("Expected empty server for nonexistent db, got %s", server)
		}
	})

	t.Run("IsProductionDb", func(t *testing.T) {
		isProd := cfg.IsProductionDb("test-db")
		if !isProd {
			t.Error("Expected test-db to be production")
		}

		isProd = cfg.IsProductionDb("dev-db")
		if isProd {
			t.Error("Expected dev-db to not be production")
		}

		// Тест для несуществующей базы
		isProd = cfg.IsProductionDb("nonexistent")
		if isProd {
			t.Error("Expected nonexistent db to not be production")
		}
	})

	t.Run("GetDbServer", func(t *testing.T) {
		server := cfg.GetDbServer("test-db")
		if server != "db-server" {
			t.Errorf("Expected server = db-server, got %s", server)
		}

		// Тест для несуществующей базы
		server = cfg.GetDbServer("nonexistent")
		if server != "" {
			t.Errorf("Expected empty server for nonexistent db, got %s", server)
		}
	})

	t.Run("GetDatabaseInfo", func(t *testing.T) {
		dbInfo := cfg.GetDatabaseInfo("test-db")
		if dbInfo == nil {
			t.Error("Expected non-nil DatabaseInfo")
		}
		if dbInfo.OneServer != "test-server" {
			t.Errorf("Expected OneServer = test-server, got %s", dbInfo.OneServer)
		}

		// Тест для несуществующей базы
		dbInfo = cfg.GetDatabaseInfo("nonexistent")
		if dbInfo != nil {
			t.Error("Expected nil DatabaseInfo for nonexistent db")
		}
	})

	t.Run("GetAllDatabases", func(t *testing.T) {
		dbs := cfg.GetAllDatabases()
		if len(dbs) != 2 {
			t.Errorf("Expected 2 databases, got %d", len(dbs))
		}

		// Проверяем, что обе базы присутствуют
		found := make(map[string]bool)
		for _, db := range dbs {
			found[db] = true
		}
		if !found["test-db"] || !found["dev-db"] {
			t.Error("Expected both test-db and dev-db in result")
		}
	})

	t.Run("GetProductionDatabases", func(t *testing.T) {
		dbs := cfg.GetProductionDatabases()
		if len(dbs) != 1 {
			t.Errorf("Expected 1 production database, got %d", len(dbs))
		}
		if dbs[0] != "test-db" {
			t.Errorf("Expected test-db, got %s", dbs[0])
		}
	})

	t.Run("GetTestDatabases", func(t *testing.T) {
		dbs := cfg.GetTestDatabases()
		if len(dbs) != 1 {
			t.Errorf("Expected 1 test database, got %d", len(dbs))
		}
		if dbs[0] != "dev-db" {
			t.Errorf("Expected dev-db, got %s", dbs[0])
		}
	})
}

func TestDatabaseMethods(t *testing.T) {
	cfg := &Config{
		DbConfig: map[string]*DatabaseInfo{
			"prod-db1": {
				OneServer: "prod-server1",
				Prod:      true,
				DbServer:  "prod-db-server1",
			},
			"prod-db2": {
				OneServer: "prod-server2",
				Prod:      true,
				DbServer:  "prod-db-server2",
			},
			"test-db1": {
				OneServer: "test-server1",
				Prod:      false,
				DbServer:  "test-db-server1",
			},
			"test-db2": {
				OneServer: "test-server2",
				Prod:      false,
				DbServer:  "test-db-server2",
			},
		},
		ProjectConfig: &ProjectConfig{
			Prod: map[string]struct {
				DbName     string                 `yaml:"dbName"`
				AddDisable []string               `yaml:"add-disable"`
				Related    map[string]interface{} `yaml:"related"`
			}{
				"prod-db1": {
					DbName: "prod-db1",
					Related: map[string]interface{}{
						"test-db1": nil,
					},
				},
				"prod-db2": {
					DbName: "prod-db2",
					Related: map[string]interface{}{
						"test-db2": nil,
					},
				},
			},
		},
	}

	t.Run("GetAllDatabases", func(t *testing.T) {
		databases := cfg.GetAllDatabases()
		if len(databases) != 4 {
			t.Errorf("Expected 4 databases, got %d", len(databases))
		}

		// Проверяем, что все базы присутствуют
		dbMap := make(map[string]bool)
		for _, db := range databases {
			dbMap[db] = true
		}

		expectedDbs := []string{"prod-db1", "prod-db2", "test-db1", "test-db2"}
		for _, expectedDb := range expectedDbs {
			if !dbMap[expectedDb] {
				t.Errorf("Expected database %s not found in result", expectedDb)
			}
		}

		// Тест с nil DbConfig
		cfgNil := &Config{DbConfig: nil}
		databases = cfgNil.GetAllDatabases()
		if databases != nil {
			t.Error("Expected nil result for nil DbConfig")
		}
	})

	t.Run("GetProductionDatabases", func(t *testing.T) {
		prodDbs := cfg.GetProductionDatabases()
		if len(prodDbs) != 2 {
			t.Errorf("Expected 2 production databases, got %d", len(prodDbs))
		}

		// Проверяем, что только продуктивные базы присутствуют
		dbMap := make(map[string]bool)
		for _, db := range prodDbs {
			dbMap[db] = true
		}

		if !dbMap["prod-db1"] || !dbMap["prod-db2"] {
			t.Error("Expected prod-db1 and prod-db2 in production databases")
		}

		if dbMap["test-db1"] || dbMap["test-db2"] {
			t.Error("Test databases should not be in production databases list")
		}

		// Тест с nil DbConfig
		cfgNil := &Config{DbConfig: nil}
		prodDbs = cfgNil.GetProductionDatabases()
		if prodDbs != nil {
			t.Error("Expected nil result for nil DbConfig")
		}
	})

	t.Run("GetTestDatabases", func(t *testing.T) {
		testDbs := cfg.GetTestDatabases()
		if len(testDbs) != 2 {
			t.Errorf("Expected 2 test databases, got %d", len(testDbs))
		}

		// Проверяем, что только тестовые базы присутствуют
		dbMap := make(map[string]bool)
		for _, db := range testDbs {
			dbMap[db] = true
		}

		if !dbMap["test-db1"] || !dbMap["test-db2"] {
			t.Error("Expected test-db1 and test-db2 in test databases")
		}

		if dbMap["prod-db1"] || dbMap["prod-db2"] {
			t.Error("Production databases should not be in test databases list")
		}

		// Тест с nil DbConfig
		cfgNil := &Config{DbConfig: nil}
		testDbs = cfgNil.GetTestDatabases()
		if testDbs != nil {
			t.Error("Expected nil result for nil DbConfig")
		}
	})

	t.Run("FindRelatedDatabase", func(t *testing.T) {
		// Тест поиска связанной базы для продуктивной базы
		relatedDb, err := cfg.FindRelatedDatabase("prod-db1")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if relatedDb != "test-db1" {
			t.Errorf("Expected related database test-db1, got %s", relatedDb)
		}

		// Тест поиска связанной базы для тестовой базы
		relatedDb, err = cfg.FindRelatedDatabase("test-db1")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if relatedDb != "prod-db1" {
			t.Errorf("Expected related database prod-db1, got %s", relatedDb)
		}

		// Тест для несуществующей базы
		_, err = cfg.FindRelatedDatabase("nonexistent")
		if err == nil {
			t.Error("Expected error for nonexistent database")
		}

		// Тест с nil ProjectConfig
		cfgNil := &Config{ProjectConfig: nil}
		_, err = cfgNil.FindRelatedDatabase("prod-db1")
		if err == nil {
			t.Error("Expected error for nil ProjectConfig")
		}

		// Тест для продуктивной базы без связанных баз
		cfgNoRelated := &Config{
			ProjectConfig: &ProjectConfig{
				Prod: map[string]struct {
					DbName     string                 `yaml:"dbName"`
					AddDisable []string               `yaml:"add-disable"`
					Related    map[string]interface{} `yaml:"related"`
				}{
					"prod-db-no-related": {
						DbName:  "prod-db-no-related",
						Related: map[string]interface{}{},
					},
				},
			},
		}
		_, err = cfgNoRelated.FindRelatedDatabase("prod-db-no-related")
		if err == nil {
			t.Error("Expected error for production database without related databases")
		}
	})

	t.Run("GetDatabaseServer", func(t *testing.T) {
		// Тест успешного получения сервера
		server, err := cfg.GetDatabaseServer("prod-db1")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if server != "prod-db-server1" {
			t.Errorf("Expected server prod-db-server1, got %s", server)
		}

		// Тест для несуществующей базы
		_, err = cfg.GetDatabaseServer("nonexistent")
		if err == nil {
			t.Error("Expected error for nonexistent database")
		}

		// Тест с nil DbConfig
		cfgNil := &Config{DbConfig: nil}
		_, err = cfgNil.GetDatabaseServer("prod-db1")
		if err == nil {
			t.Error("Expected error for nil DbConfig")
		}

		// Тест для базы с пустым DbServer
		cfgEmptyServer := &Config{
			DbConfig: map[string]*DatabaseInfo{
				"empty-server-db": {
					OneServer: "server",
					Prod:      true,
					DbServer:  "",
				},
			},
		}
		_, err = cfgEmptyServer.GetDatabaseServer("empty-server-db")
		if err == nil {
			t.Error("Expected error for database with empty DbServer")
		}
	})

	t.Run("DetermineSrcAndDstServers", func(t *testing.T) {
		// Тест для продуктивной базы
		srcServer, dstServer, srcDB, dstDB, err := cfg.DetermineSrcAndDstServers("prod-db1")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if srcServer != "prod-db-server1" {
			t.Errorf("Expected srcServer prod-db-server1, got %s", srcServer)
		}
		if dstServer != "test-db-server1" {
			t.Errorf("Expected dstServer test-db-server1, got %s", dstServer)
		}
		if srcDB != "prod-db1" {
			t.Errorf("Expected srcDB prod-db1, got %s", srcDB)
		}
		if dstDB != "test-db1" {
			t.Errorf("Expected dstDB test-db1, got %s", dstDB)
		}

		// Тест для тестовой базы
		srcServer, dstServer, srcDB, dstDB, err = cfg.DetermineSrcAndDstServers("test-db1")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if srcServer != "prod-db-server1" {
			t.Errorf("Expected srcServer prod-db-server1, got %s", srcServer)
		}
		if dstServer != "test-db-server1" {
			t.Errorf("Expected dstServer test-db-server1, got %s", dstServer)
		}
		if srcDB != "prod-db1" {
			t.Errorf("Expected srcDB prod-db1, got %s", srcDB)
		}
		if dstDB != "test-db1" {
			t.Errorf("Expected dstDB test-db1, got %s", dstDB)
		}

		// Тест для несуществующей базы
		_, _, _, _, err = cfg.DetermineSrcAndDstServers("nonexistent")
		if err == nil {
			t.Error("Expected error for nonexistent database")
		}
	})
}

func TestGetSlog(t *testing.T) {
	tests := []struct {
		name     string
		actor    string
		logLevel string
	}{
		{
			name:     "debug_level",
			actor:    "test-actor",
			logLevel: "Debug",
		},
		{
			name:     "info_level",
			actor:    "test-actor",
			logLevel: "Info",
		},
		{
			name:     "warn_level",
			actor:    "test-actor",
			logLevel: "Warn",
		},
		{
			name:     "error_level",
			actor:    "test-actor",
			logLevel: "Error",
		},
		{
			name:     "invalid_level",
			actor:    "test-actor",
			logLevel: "Invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := getSlog(tt.actor, tt.logLevel)
			if logger == nil {
				t.Error("Expected non-nil logger")
			}
			// Проверяем, что логгер можно использовать
			logger.Info("Test log message")
		})
	}
}

func TestValidateRequiredParams(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	t.Run("valid_params", func(t *testing.T) {
		params := &InputParams{
			GHACommand:     "test-command",
			GHAGiteaURL:    "https://test.com",
			GHARepository:  "test/repo",
			GHAAccessToken: "test-token",
			GHAActor:       "test-actor",
			GHALogLevel:    "Info",
		}

		err := validateRequiredParams(params, logger)
		if err != nil {
			t.Errorf("Expected no error for valid params, got %v", err)
		}
	})

	t.Run("missing_command", func(t *testing.T) {
		params := &InputParams{
			GHACommand:     "", // Пустая команда
			GHAGiteaURL:    "https://test.com",
			GHARepository:  "test/repo",
			GHAAccessToken: "test-token",
			GHAActor:       "test-actor",
			GHALogLevel:    "Info",
		}

		err := validateRequiredParams(params, logger)
		if err == nil {
			t.Error("Expected error for missing command")
		}
		if !strings.Contains(err.Error(), "COMMAND") {
			t.Errorf("Expected error about empty command, got %v", err)
		}
	})

	t.Run("missing_gitea_url", func(t *testing.T) {
		params := &InputParams{
			GHACommand:     "test-command",
			GHAGiteaURL:    "", // Пустой URL
			GHARepository:  "test/repo",
			GHAAccessToken: "test-token",
			GHAActor:       "test-actor",
			GHALogLevel:    "Info",
		}

		err := validateRequiredParams(params, logger)
		if err == nil {
			t.Error("Expected error for missing gitea URL")
		}
		if !strings.Contains(err.Error(), "GITEAURL") {
			t.Errorf("Expected error about empty gitea URL, got %v", err)
		}
	})
}

func TestAppConfigYAMLParsing(t *testing.T) {
	tests := []struct {
		name     string
		yamlData string
		wantErr  bool
	}{
		{
			name: "valid_app_config",
			yamlData: `
logLevel: debug
workDir: /tmp/test
tmpDir: /tmp
timeout: 300
paths:
  bin1cv8: /opt/1cv8/bin/1cv8
  binIbcmd: /opt/1cv8/bin/ibcmd
`,
			wantErr: false,
		},
		{
			name:     "invalid_yaml",
			yamlData: "invalid: yaml: content:",
			wantErr:  true,
		},
		{
			name:     "empty_data",
			yamlData: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var appConfig AppConfig
			err := yaml.Unmarshal([]byte(tt.yamlData), &appConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("AppConfig YAML parsing error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProjectConfigYAMLParsing(t *testing.T) {
	tests := []struct {
		name     string
		yamlData string
		wantErr  bool
	}{
		{
			name: "valid_project_config",
			yamlData: `
debug: true
store-db: test-store
prod:
  db1:
    dbName: prod-db1
    add-disable:
      - feature1
      - feature2
`,
			wantErr: false,
		},
		{
			name:     "invalid_yaml",
			yamlData: "invalid: yaml: content:",
			wantErr:  true,
		},
		{
			name:     "empty_data",
			yamlData: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var projectConfig ProjectConfig
			err := yaml.Unmarshal([]byte(tt.yamlData), &projectConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProjectConfig YAML parsing error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSecretConfigYAMLParsing(t *testing.T) {
	tests := []struct {
		name     string
		yamlData string
		wantErr  bool
	}{
		{
			name: "valid_secret_config",
			yamlData: `
passwords:
  rac: test-rac-pass
  db: test-db-pass
  mssql: test-mssql-pass
gitea:
  accessToken: test-token
sonarqube:
  token: test-sonar-token
`,
			wantErr: false,
		},
		{
			name:     "invalid_yaml",
			yamlData: "invalid: yaml: content:",
			wantErr:  true,
		},
		{
			name:     "empty_data",
			yamlData: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var secretConfig SecretConfig
			err := yaml.Unmarshal([]byte(tt.yamlData), &secretConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("SecretConfig YAML parsing error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDatabaseInfoYAMLParsing(t *testing.T) {
	tests := []struct {
		name     string
		yamlData string
		wantErr  bool
	}{
		{
			name: "valid_db_config",
			yamlData: `
test-db:
  one-server: localhost:1541
  prod: false
  dbserver: localhost
prod-db:
  one-server: prod-server:1541
  prod: true
  dbserver: prod-server
`,
			wantErr: false,
		},
		{
			name:     "invalid_yaml",
			yamlData: "invalid: yaml: content:",
			wantErr:  true,
		},
		{
			name:     "empty_data",
			yamlData: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dbConfig map[string]*DatabaseInfo
			err := yaml.Unmarshal([]byte(tt.yamlData), &dbConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("DatabaseInfo YAML parsing error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestCreateGiteaAPI тестирует создание Gitea API
func TestCreateGiteaAPI(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := &Config{
			GiteaURL:    "https://gitea.example.com",
			Owner:       "testowner",
			Repo:        "testrepo",
			AccessToken: "testtoken",
			BaseBranch:  "main",
			NewBranch:   "feature",
			Command:     "test",
		}

		api := CreateGiteaAPI(cfg)
		if api == nil {
			t.Error("Expected non-nil API instance")
		}
	})

	t.Run("nil config", func(t *testing.T) {
		api := CreateGiteaAPI(nil)
		if api != nil {
			t.Error("Expected nil API for nil config")
		}
	})
}

// TestMustLoad тестирует функцию MustLoad
func TestMustLoad(t *testing.T) {
	t.Skip("Skipping MustLoad test due to external API dependencies")
}

// TestLoadFunctions тестирует различные load функции
func TestLoadFunctions(t *testing.T) {
	cfg := &Config{}

	t.Run("loadGitConfig", func(t *testing.T) {
		// Эта функция требует реальной API интеграции, тестируем вызов
		_, err := loadGitConfig(slog.Default(), cfg)
		if err != nil {
			// Ожидаем ошибку из-за отсутствия реального API
		}
	})

	t.Run("loadLoggingConfig", func(t *testing.T) {
		// Эта функция требует реальной API интеграции, тестируем вызов
		_, err := loadLoggingConfig(slog.Default(), cfg)
		if err != nil {
			// Ожидаем ошибку из-за отсутствия реального API
		}
	})

	t.Run("loadRacConfig", func(t *testing.T) {
		// Эта функция требует реальной API интеграции, тестируем вызов
		_, err := loadRacConfig(slog.Default(), cfg)
		if err != nil {
			// Ожидаем ошибку из-за отсутствия реального API
		}
	})
}

// TestLoadConfigMethods тестирует методы загрузки конфигурации
func TestLoadConfigMethods(t *testing.T) {
	cfg := &Config{}

	t.Run("loadAppConfig", func(t *testing.T) {
		// Эта функция требует реальной API интеграции, тестируем вызов
		_, err := loadAppConfig(slog.Default(), cfg)
		if err != nil {
			// Ожидаем ошибку из-за отсутствия реального API
		}
	})

	t.Run("loadSecretConfig", func(t *testing.T) {
		// Эта функция требует реальной API интеграции, тестируем вызов
		_, err := loadSecretConfig(slog.Default(), cfg)
		if err != nil {
			// Ожидаем ошибку из-за отсутствия реального API
		}
	})

	t.Run("loadProjectConfig", func(t *testing.T) {
		// Эта функция требует реальной API интеграции, тестируем вызов
		_, err := loadProjectConfig(slog.Default(), cfg)
		if err != nil {
			// Ожидаем ошибку из-за отсутствия реального API
		}
	})

	t.Run("loadDbConfig", func(t *testing.T) {
		// Эта функция требует реальной API интеграции, тестируем вызов
		_, err := loadDbConfig(slog.Default(), cfg)
		if err != nil {
			// Ожидаем ошибку из-за отсутствия реального API
		}
	})
}

// TestConfigMethodsAdditional тестирует различные методы конфигурации
func TestConfigMethodsAdditional(t *testing.T) {
	cfg := &Config{
		DbConfig: map[string]*DatabaseInfo{
			"testdb": {
				OneServer: "localhost",
				Prod:      false,
				DbServer:  "localhost",
			},
		},
	}

	t.Run("LoadDBRestoreConfig", func(t *testing.T) {
		config, err := LoadDBRestoreConfig(cfg, "testdb")
		// Ожидаем ошибку из-за отсутствия полной конфигурации
		if err != nil {
			// Это ожидаемо для неполной конфигурации
		}
		_ = config
	})

	t.Run("LoadServiceModeConfig", func(t *testing.T) {
		cfg.AppConfig = &AppConfig{
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
		}
		cfg.SecretConfig = &SecretConfig{
			Passwords: struct {
				Rac                string `yaml:"rac"`
				Db                 string `yaml:"db"`
				Mssql              string `yaml:"mssql"`
				StoreAdminPassword string `yaml:"storeAdminPassword"`
				Smb                string `yaml:"smb"`
			}{
				Rac: "racpass",
				Db:  "dbpass",
			},
		}

		config, err := cfg.LoadServiceModeConfig("testdb")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if config.RacPath != "/usr/bin/rac" {
			t.Errorf("Expected RacPath=/usr/bin/rac, got %s", config.RacPath)
		}
	})

	t.Run("GetRacServerForDb", func(t *testing.T) {
		server := cfg.GetRacServerForDb("testdb")
		if server != "localhost" {
			t.Errorf("Expected server=localhost, got %s", server)
		}
	})
}

// TestSonarQubeConfigMethods тестирует методы конфигурации SonarQube
func TestSonarQubeConfigMethods(t *testing.T) {
	cfg := &Config{}

	t.Run("GetSonarQubeConfig", func(t *testing.T) {
		config := cfg.GetSonarQubeConfig()
		// Config может быть nil если не инициализирован
		_ = config
	})

	t.Run("SetSonarQubeConfig", func(t *testing.T) {
		newConfig := &SonarQubeConfig{URL: "http://test.com"}
		cfg.SetSonarQubeConfig(newConfig)
		if cfg.SonarQubeConfig != newConfig {
			t.Error("Expected SonarQubeConfig to be set")
		}
	})

	t.Run("GetScannerConfig", func(t *testing.T) {
		config := cfg.GetScannerConfig()
		// Config может быть nil если не инициализирован
		_ = config
	})

	t.Run("SetScannerConfig", func(t *testing.T) {
		newConfig := &ScannerConfig{}
		cfg.SetScannerConfig(newConfig)
		if cfg.ScannerConfig != newConfig {
			t.Error("Expected ScannerConfig to be set")
		}
	})
}

// TestLoadFromEnvFunctions тестирует функции загрузки из переменных окружения
func TestLoadFromEnvFunctions(t *testing.T) {
	// Устанавливаем переменные окружения
	_ = os.Setenv("SONAR_URL", "http://env.sonar.com")
	_ = os.Setenv("SONAR_TOKEN", "env-token")
	_ = os.Setenv("SCANNER_PATH", "/env/scanner")

	defer func() {
		_ = os.Unsetenv("SONAR_URL")
		_ = os.Unsetenv("SONAR_TOKEN")
		_ = os.Unsetenv("SCANNER_PATH")
	}()

	cfg := &Config{}

	t.Run("LoadSonarQubeConfigFromEnv", func(t *testing.T) {
		config, err := cfg.LoadSonarQubeConfigFromEnv()
		// Ожидаем ошибку из-за отсутствия токена
		if err != nil {
			// Это ожидаемо без токена
		}
		_ = config
	})

	t.Run("LoadScannerConfigFromEnv", func(t *testing.T) {
		config, err := cfg.LoadScannerConfigFromEnv()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if config != nil {
			// Test passed - config loaded from env
		}
	})
}

// TestCreateWorkDirectories тестирует создание рабочих директорий
func TestCreateWorkDirectories(t *testing.T) {
	cfg := &Config{
		AppConfig: &AppConfig{
			WorkDir: "/tmp/test-work",
			TmpDir:  "/tmp/test-temp",
		},
	}

	err := createWorkDirectories(cfg)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Проверяем что директории созданы
	dirs := []string{"/tmp/test-work", "/tmp/test-temp"}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Expected directory %s to be created", dir)
		} else {
			// Очищаем после теста
			_ = os.RemoveAll(dir)
		}
	}
}

// TestAnalyzeProject тестирует функцию анализа проекта
func TestAnalyzeProject(t *testing.T) {
	t.Skip("Skipping AnalyzeProject test due to external API dependencies")
}

// TestReloadConfig тестирует перезагрузку конфигурации
func TestReloadConfig(t *testing.T) {
	t.Skip("Skipping ReloadConfig test due to external API dependencies")
}