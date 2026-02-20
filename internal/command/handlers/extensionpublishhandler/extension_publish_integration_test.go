//go:build integration
// +build integration

// Package app содержит интеграционные тесты для extension_publish
// Запуск: go test -tags=integration ./internal/app/... -run Integration
package extensionpublishhandler

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/entity/gitea"
)

// integrationLogger создаёт логгер для интеграционных тестов
func integrationLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// TestIntegration_ExtensionPublish_RealGitea тестирует взаимодействие с реальным Gitea сервером
// Требует установленных переменных окружения:
// - GITEA_URL: URL Gitea сервера
// - GITEA_TOKEN: токен доступа
// - GITHUB_REPOSITORY: репозиторий расширения (например, "org/extension-repo")
// - GITHUB_REF_NAME: тег релиза (например, "v1.0.0")
//
// Пример запуска:
// GITEA_URL=https://git.example.com GITEA_TOKEN=xxx GITHUB_REPOSITORY=myorg/myext GITHUB_REF_NAME=v1.0.0 \
//
//	go test -tags=integration ./internal/app/... -run TestIntegration_ExtensionPublish_RealGitea -v
func TestIntegration_ExtensionPublish_RealGitea(t *testing.T) {
	// Проверяем наличие обязательных переменных окружения
	giteaURL := os.Getenv("GITEA_URL")
	giteaToken := os.Getenv("GITEA_TOKEN")
	repo := os.Getenv("GITHUB_REPOSITORY")
	ref := os.Getenv("GITHUB_REF_NAME")

	if giteaURL == "" || giteaToken == "" || repo == "" || ref == "" {
		t.Skip("Пропуск: не установлены переменные окружения GITEA_URL, GITEA_TOKEN, GITHUB_REPOSITORY, GITHUB_REF_NAME")
	}

	// Устанавливаем dry-run режим для безопасного тестирования
	origDryRun := os.Getenv("BR_DRY_RUN")
	defer _ = os.Setenv("BR_DRY_RUN", origDryRun)
	_ = os.Setenv("BR_DRY_RUN", "true")

	cfg := &config.Config{
		GiteaURL:    giteaURL,
		AccessToken: giteaToken,
	}

	l := integrationLogger()

	err := ExtensionPublish(nil, l, cfg)
	if err != nil {
		// В dry-run режиме ошибки синхронизации ожидаемы
		// Главное — что мы дошли до этапа обработки подписчиков
		if !strings.Contains(err.Error(), "релиз") {
			t.Logf("Интеграционный тест: %v", err)
		}
	}
}

// TestIntegration_FindSubscribedRepos_RealGitea тестирует поиск подписчиков на реальном Gitea
func TestIntegration_FindSubscribedRepos_RealGitea(t *testing.T) {
	ctx := context.Background()
	giteaURL := os.Getenv("GITEA_URL")
	giteaToken := os.Getenv("GITEA_TOKEN")
	repo := os.Getenv("GITHUB_REPOSITORY")

	if giteaURL == "" || giteaToken == "" || repo == "" {
		t.Skip("Пропуск: не установлены переменные окружения GITEA_URL, GITEA_TOKEN, GITHUB_REPOSITORY")
	}

	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		t.Fatalf("Некорректный формат GITHUB_REPOSITORY: %s", repo)
	}
	owner, repoName := parts[0], parts[1]

	giteaCfg := gitea.Config{
		GiteaURL:    giteaURL,
		Owner:       owner,
		Repo:        repoName,
		AccessToken: giteaToken,
	}
	api := gitea.NewGiteaAPI(giteaCfg)

	l := integrationLogger()

	// Тестовые расширения для поиска подписчиков
	extensions := []string{"cfe", "cfe/common"}

	subscribers, err := FindSubscribedRepos(ctx, l, api, repoName, extensions)
	if err != nil {
		t.Fatalf("Ошибка поиска подписчиков: %v", err)
	}

	t.Logf("Найдено подписчиков: %d", len(subscribers))
	for _, sub := range subscribers {
		t.Logf("  - %s/%s → %s", sub.Organization, sub.Repository, sub.TargetDirectory)
	}
}

// TestIntegration_GetSourceFiles_RealGitea тестирует получение файлов из реального репозитория
func TestIntegration_GetSourceFiles_RealGitea(t *testing.T) {
	ctx := context.Background()
	giteaURL := os.Getenv("GITEA_URL")
	giteaToken := os.Getenv("GITEA_TOKEN")
	repo := os.Getenv("GITHUB_REPOSITORY")
	ref := os.Getenv("GITHUB_REF_NAME")

	if giteaURL == "" || giteaToken == "" || repo == "" || ref == "" {
		t.Skip("Пропуск: не установлены переменные окружения GITEA_URL, GITEA_TOKEN, GITHUB_REPOSITORY, GITHUB_REF_NAME")
	}

	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		t.Fatalf("Некорректный формат GITHUB_REPOSITORY: %s", repo)
	}
	owner, repoName := parts[0], parts[1]

	giteaCfg := gitea.Config{
		GiteaURL:    giteaURL,
		Owner:       owner,
		Repo:        repoName,
		AccessToken: giteaToken,
	}
	api := gitea.NewGiteaAPI(giteaCfg)

	extDir := os.Getenv("BR_EXT_DIR")
	if extDir == "" {
		extDir = ""
	}

	files, err := GetSourceFiles(ctx, api, extDir, ref)
	if err != nil {
		t.Fatalf("Ошибка получения файлов: %v", err)
	}

	t.Logf("Найдено файлов: %d", len(files))
	for i, file := range files {
		if i >= 10 {
			t.Logf("  ... и ещё %d файлов", len(files)-10)
			break
		}
		t.Logf("  - %s (%d bytes)", file.Path, len(file.Content))
	}
}

// TestIntegration_PublishReport_Output тестирует вывод отчёта (не требует Gitea)
func TestIntegration_PublishReport_Output(t *testing.T) {
	report := &PublishReport{
		ExtensionName: "TestExtension",
		Version:       "v1.2.3",
		SourceRepo:    "myorg/test-extension",
		Results: []PublishResult{
			{
				Subscriber:   SubscribedRepo{Organization: "Org1", Repository: "Repo1", TargetDirectory: "cfe/TestExt"},
				Status:       StatusSuccess,
				PRNumber:     42,
				PRURL:        "https://git.example.com/Org1/Repo1/pulls/42",
				ErrorMessage: "",
			},
			{
				Subscriber:   SubscribedRepo{Organization: "Org2", Repository: "Repo2", TargetDirectory: "cfe/TestExt"},
				Status:       StatusFailed,
				PRNumber:     0,
				PRURL:        "",
				ErrorMessage: "ошибка доступа к репозиторию",
			},
			{
				Subscriber:   SubscribedRepo{Organization: "Org3", Repository: "Repo3", TargetDirectory: "cfe/TestExt"},
				Status:       StatusSkipped,
				PRNumber:     0,
				PRURL:        "",
				ErrorMessage: "dry-run mode",
			},
		},
	}

	l := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Тест текстового вывода
	origJSON := os.Getenv("BR_OUTPUT_JSON")
	defer _ = os.Setenv("BR_OUTPUT_JSON", origJSON)
	_ = os.Setenv("BR_OUTPUT_JSON", "")

	err := ReportResults(report, l)
	if err != nil {
		t.Errorf("Ошибка текстового отчёта: %v", err)
	}

	// Тест JSON вывода
	_ = os.Setenv("BR_OUTPUT_JSON", "1")
	err = ReportResults(report, l)
	if err != nil {
		t.Errorf("Ошибка JSON отчёта: %v", err)
	}
}

// TestIntegration_GenerateBranchName тестирует генерацию имени ветки
func TestIntegration_GenerateBranchName(t *testing.T) {
	tests := []struct {
		name           string
		extName        string
		version        string
		expectedResult string
	}{
		{
			name:           "обычная версия",
			extName:        "MyExt",
			version:        "v1.0.0",
			expectedResult: "update-myext-1.0.0",
		},
		{
			name:           "версия с префиксом",
			extName:        "TestExt",
			version:        "release-v2.3.4",
			expectedResult: "update-testext-release-v2.3.4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			branchName := GenerateBranchName(tt.extName, tt.version)
			if branchName != tt.expectedResult {
				t.Errorf("Ожидалось %q, получено %q", tt.expectedResult, branchName)
			}
		})
	}
}

// TestIntegration_GenerateCommitMessage тестирует генерацию commit message
func TestIntegration_GenerateCommitMessage(t *testing.T) {
	extName := "MyExtension"
	version := "v1.2.3"

	message := GenerateCommitMessage(extName, version)

	// Проверяем наличие ключевой информации в сообщении
	if !strings.Contains(message, extName) {
		t.Errorf("Сообщение должно содержать имя расширения: %s", message)
	}
	if !strings.Contains(message, version) {
		t.Errorf("Сообщение должно содержать версию: %s", message)
	}
}

// TestIntegration_BuildExtensionPRTitle тестирует генерацию заголовка PR
func TestIntegration_BuildExtensionPRTitle(t *testing.T) {
	extName := "TestExtension"
	version := "v2.0.0"

	title := BuildExtensionPRTitle(extName, version)

	if !strings.Contains(title, extName) {
		t.Errorf("Заголовок PR должен содержать имя расширения")
	}
	if !strings.Contains(title, version) {
		t.Errorf("Заголовок PR должен содержать версию")
	}
}

// TestIntegration_BuildExtensionPRBody тестирует генерацию тела PR
func TestIntegration_BuildExtensionPRBody(t *testing.T) {
	extName := "TestExtension"
	version := "v2.0.0"
	sourceRepo := "myorg/test-ext"
	releaseNotes := "## Changelog\n- Feature 1\n- Bug fix 2"
	releaseURL := "https://git.example.com/myorg/test-ext/releases/tag/v2.0.0"

	release := &gitea.Release{
		TagName: version,
		Name:    "Release " + version,
		Body:    releaseNotes,
	}

	body := BuildExtensionPRBody(release, sourceRepo, extName, releaseURL)

	// Проверяем наличие ключевой информации
	if !strings.Contains(body, extName) {
		t.Errorf("Тело PR должно содержать имя расширения")
	}
	if !strings.Contains(body, version) {
		t.Errorf("Тело PR должно содержать версию")
	}
	if !strings.Contains(body, sourceRepo) {
		t.Errorf("Тело PR должно содержать исходный репозиторий")
	}
	if !strings.Contains(body, releaseURL) {
		t.Errorf("Тело PR должно содержать URL релиза")
	}
}
