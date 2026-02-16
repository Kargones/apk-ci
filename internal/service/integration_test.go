package service

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/Kargones/apk-ci/internal/config"
)

// TestGiteaServiceIntegration тестирует интеграцию GiteaService с реальными компонентами
func TestGiteaServiceIntegration(t *testing.T) {
	// Пропускаем интеграционные тесты если не установлена переменная окружения
	if os.Getenv("INTEGRATION_TESTS") == "" {
		t.Skip("Skipping integration tests. Set INTEGRATION_TESTS=1 to run.")
	}

	// Создаем тестовую конфигурацию
	cfg := &config.Config{
		GiteaURL:    os.Getenv("TEST_GITEA_URL"),
		Owner:       os.Getenv("TEST_GITEA_OWNER"),
		Repo:        os.Getenv("TEST_GITEA_REPO"),
		AccessToken: os.Getenv("TEST_GITEA_TOKEN"),
		BaseBranch:  "main",
		NewBranch:   "test-integration",
		Command:     "test",
	}

	// Проверяем, что все необходимые переменные окружения установлены
	if cfg.GiteaURL == "" || cfg.Owner == "" || cfg.Repo == "" || cfg.AccessToken == "" {
		t.Skip("Skipping integration tests. Required environment variables not set: TEST_GITEA_URL, TEST_GITEA_OWNER, TEST_GITEA_REPO, TEST_GITEA_TOKEN")
	}

	// Создаем GiteaService через фабрику
	factory := NewGiteaFactory()
	service, err := factory.CreateGiteaService(cfg)
	if err != nil {
		t.Fatalf("Failed to create GiteaService: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	// Тестируем анализ проекта
	t.Run("AnalyzeProject", func(t *testing.T) {
		err := service.AnalyzeProject(ctx, logger, cfg.BaseBranch)
		if err != nil {
			t.Errorf("AnalyzeProject failed: %v", err)
		}
	})

	// Тестируем получение API и конфигурации
	t.Run("GetAPIAndConfig", func(t *testing.T) {
		api := service.GetAPI()
		if api == nil {
			t.Error("GetAPI returned nil")
		}

		config := service.GetConfig()
		if config == nil {
			t.Error("GetConfig returned nil")
			return
		}

		if config.GiteaURL != cfg.GiteaURL {
			t.Errorf("Expected GiteaURL %s, got %s", cfg.GiteaURL, config.GiteaURL)
		}
	})
}

// TestGiteaFactoryIntegration тестирует интеграцию GiteaFactory
func TestGiteaFactoryIntegration(t *testing.T) {
	cfg := &config.Config{
		GiteaURL:    "https://gitea.example.com",
		Owner:       "testowner",
		Repo:        "testrepo",
		AccessToken: "testtoken",
		BaseBranch:  "main",
		NewBranch:   "feature",
		Command:     "test",
	}

	factory := NewGiteaFactory()

	// Тестируем создание конфигурации Gitea
	t.Run("CreateGiteaConfig", func(t *testing.T) {
		giteaConfig := factory.CreateGiteaConfig(cfg)
		if giteaConfig.GiteaURL != cfg.GiteaURL {
			t.Errorf("Expected GiteaURL %s, got %s", cfg.GiteaURL, giteaConfig.GiteaURL)
		}

		if giteaConfig.Owner != cfg.Owner {
			t.Errorf("Expected Owner %s, got %s", cfg.Owner, giteaConfig.Owner)
		}

		if giteaConfig.Repo != cfg.Repo {
			t.Errorf("Expected Repo %s, got %s", cfg.Repo, giteaConfig.Repo)
		}
	})

	// Тестируем создание GiteaService
	t.Run("CreateGiteaService", func(t *testing.T) {
		service, err := factory.CreateGiteaService(cfg)
		if err != nil {
			t.Fatalf("Failed to create GiteaService: %v", err)
		}

		if service == nil {
			t.Fatal("CreateGiteaService returned nil")
		}

		api := service.GetAPI()
		if api == nil {
			t.Error("Service API is nil")
		}

		config := service.GetConfig()
		if config == nil {
			t.Error("Service config is nil")
		}
	})
}

// TestConfigAnalyzerIntegration тестирует интеграцию ConfigAnalyzer
func TestConfigAnalyzerIntegration(t *testing.T) {
	// Пропускаем интеграционные тесты если не установлена переменная окружения
	if os.Getenv("INTEGRATION_TESTS") == "" {
		t.Skip("Skipping integration tests. Set INTEGRATION_TESTS=1 to run.")
	}

	cfg := &config.Config{
		GiteaURL:    os.Getenv("TEST_GITEA_URL"),
		Owner:       os.Getenv("TEST_GITEA_OWNER"),
		Repo:        os.Getenv("TEST_GITEA_REPO"),
		AccessToken: os.Getenv("TEST_GITEA_TOKEN"),
		BaseBranch:  "main",
		NewBranch:   "feature",
		Command:     "test",
	}

	// Проверяем, что все необходимые переменные окружения установлены
	if cfg.GiteaURL == "" || cfg.Owner == "" || cfg.Repo == "" || cfg.AccessToken == "" {
		t.Skip("Skipping integration tests. Required environment variables not set: TEST_GITEA_URL, TEST_GITEA_OWNER, TEST_GITEA_REPO, TEST_GITEA_TOKEN")
	}

	analyzer := NewConfigAnalyzer(cfg)

	if analyzer == nil {
		t.Fatal("NewConfigAnalyzer returned nil")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Тестируем анализ проекта
	err := analyzer.AnalyzeProject(logger, "main")
	if err != nil {
		t.Errorf("AnalyzeProject failed: %v", err)
	}
}
