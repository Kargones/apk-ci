// Package app содержит тесты для функций extension_publish
package app

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/entity/gitea"
)

// testLogger создаёт логгер для тестов, который не выводит ничего
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// TestParseSubscriptionID_Valid проверяет парсинг корректных идентификаторов подписок
func TestParseSubscriptionID_Valid(t *testing.T) {
	tests := []struct {
		name           string
		subscriptionID string
		wantOrg        string
		wantRepo       string
		wantExtDir     string
	}{
		{
			name:           "простой формат с одним каталогом",
			subscriptionID: "lib_ssl_апкБСП",
			wantOrg:        "lib",
			wantRepo:       "ssl",
			wantExtDir:     "апкБСП",
		},
		{
			name:           "формат с вложенным каталогом",
			subscriptionID: "lib_common_cfe_utils",
			wantOrg:        "lib",
			wantRepo:       "common",
			wantExtDir:     "cfe/utils",
		},
		{
			name:           "формат с глубоко вложенным каталогом",
			subscriptionID: "MyOrg_MyProject_extensions_v2_common",
			wantOrg:        "MyOrg",
			wantRepo:       "MyProject",
			wantExtDir:     "extensions/v2/common",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org, repo, extDir, err := ParseSubscriptionID(tt.subscriptionID)
			if err != nil {
				t.Fatalf("ParseSubscriptionID(%q) вернул ошибку: %v", tt.subscriptionID, err)
			}
			if org != tt.wantOrg {
				t.Errorf("org = %q, want %q", org, tt.wantOrg)
			}
			if repo != tt.wantRepo {
				t.Errorf("repo = %q, want %q", repo, tt.wantRepo)
			}
			if extDir != tt.wantExtDir {
				t.Errorf("extDir = %q, want %q", extDir, tt.wantExtDir)
			}
		})
	}
}

// TestParseSubscriptionID_Invalid проверяет обработку некорректных идентификаторов подписок
func TestParseSubscriptionID_Invalid(t *testing.T) {
	tests := []struct {
		name           string
		subscriptionID string
	}{
		{
			name:           "только одна часть",
			subscriptionID: "main",
		},
		{
			name:           "только две части",
			subscriptionID: "lib_ssl",
		},
		{
			name:           "пустая строка",
			subscriptionID: "",
		},
		{
			name:           "пустая организация",
			subscriptionID: "_ssl_апкБСП",
		},
		{
			name:           "пустой репозиторий",
			subscriptionID: "lib__апкБСП",
		},
		{
			name:           "пустой каталог",
			subscriptionID: "lib_ssl_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org, repo, extDir, err := ParseSubscriptionID(tt.subscriptionID)
			if err == nil {
				t.Errorf("ParseSubscriptionID(%q) не вернул ошибку, got org=%q, repo=%q, extDir=%q",
					tt.subscriptionID, org, repo, extDir)
			}
		})
	}
}

// TestFindSubscribedRepos_Success проверяет успешный поиск подписчиков через project.yaml
func TestFindSubscribedRepos_Success(t *testing.T) {
	// project.yaml содержимое для TargetOrg/TargetRepo (подписка на cfe)
	targetRepoProjectYAML := `subscriptions:
  - SourceOrg_source-repo_cfe
`
	// project.yaml содержимое для OtherOrg/OtherRepo (подписка на cfe/common)
	otherRepoProjectYAML := `subscriptions:
  - SourceOrg_source-repo_cfe_common
`

	// Создаём тестовый сервер, который обрабатывает разные эндпоинты
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		// Запрос списка организаций пользователя
		case r.URL.Path == "/api/v1/user/orgs":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if r.URL.Query().Get("page") == "1" || r.URL.Query().Get("page") == "" {
				_, _ = w.Write([]byte(`[
					{"id": 1, "name": "TargetOrg", "full_name": "Target Organization", "username": "TargetOrg"},
					{"id": 2, "name": "OtherOrg", "full_name": "Other Organization", "username": "OtherOrg"},
					{"id": 3, "name": "SourceOrg", "full_name": "Source Organization", "username": "SourceOrg"}
				]`))
			} else {
				_, _ = w.Write([]byte(`[]`))
			}

		// Запрос репозиториев организации TargetOrg
		case r.URL.Path == "/api/v1/orgs/TargetOrg/repos":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if r.URL.Query().Get("page") == "1" || r.URL.Query().Get("page") == "" {
				_, _ = w.Write([]byte(`[
					{"id": 1, "name": "TargetRepo", "full_name": "TargetOrg/TargetRepo", "owner": {"id": 1, "login": "TargetOrg", "type": "Organization"}, "default_branch": "main", "private": false, "fork": false}
				]`))
			} else {
				_, _ = w.Write([]byte(`[]`))
			}

		// Запрос репозиториев организации OtherOrg
		case r.URL.Path == "/api/v1/orgs/OtherOrg/repos":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if r.URL.Query().Get("page") == "1" || r.URL.Query().Get("page") == "" {
				_, _ = w.Write([]byte(`[
					{"id": 2, "name": "OtherRepo", "full_name": "OtherOrg/OtherRepo", "owner": {"id": 2, "login": "OtherOrg", "type": "Organization"}, "default_branch": "develop", "private": false, "fork": false}
				]`))
			} else {
				_, _ = w.Write([]byte(`[]`))
			}

		// Запрос репозиториев организации SourceOrg (пустой)
		case r.URL.Path == "/api/v1/orgs/SourceOrg/repos":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))

		// project.yaml для TargetOrg/TargetRepo
		case r.URL.Path == "/api/v1/repos/TargetOrg/TargetRepo/contents/project.yaml":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"content": "%s", "encoding": "base64"}`,
				base64EncodeString(targetRepoProjectYAML))))

		// project.yaml для OtherOrg/OtherRepo
		case r.URL.Path == "/api/v1/repos/OtherOrg/OtherRepo/contents/project.yaml":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"content": "%s", "encoding": "base64"}`,
				base64EncodeString(otherRepoProjectYAML))))

		default:
			// Файл не найден
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	api := &gitea.API{
		GiteaURL:    server.URL,
		Owner:       "SourceOrg",
		AccessToken: "test-token",
	}

	extensions := []string{"cfe", "cfe/common"}
	subscribers, err := FindSubscribedRepos(testLogger(), api, "source-repo", extensions)
	if err != nil {
		t.Fatalf("FindSubscribedRepos вернул ошибку: %v", err)
	}

	if len(subscribers) != 2 {
		t.Fatalf("Ожидалось 2 подписчика, получено %d", len(subscribers))
	}

	// Проверяем первого подписчика
	if subscribers[0].Organization != "TargetOrg" {
		t.Errorf("Subscriber[0].Organization = %q, want %q", subscribers[0].Organization, "TargetOrg")
	}
	if subscribers[0].Repository != "TargetRepo" {
		t.Errorf("Subscriber[0].Repository = %q, want %q", subscribers[0].Repository, "TargetRepo")
	}
	if subscribers[0].TargetDirectory != "cfe" {
		t.Errorf("Subscriber[0].TargetDirectory = %q, want %q", subscribers[0].TargetDirectory, "cfe")
	}
	if subscribers[0].TargetBranch != "main" {
		t.Errorf("Subscriber[0].TargetBranch = %q, want %q", subscribers[0].TargetBranch, "main")
	}

	// Проверяем второго подписчика
	if subscribers[1].Organization != "OtherOrg" {
		t.Errorf("Subscriber[1].Organization = %q, want %q", subscribers[1].Organization, "OtherOrg")
	}
	if subscribers[1].Repository != "OtherRepo" {
		t.Errorf("Subscriber[1].Repository = %q, want %q", subscribers[1].Repository, "OtherRepo")
	}
	if subscribers[1].TargetDirectory != "cfe/common" {
		t.Errorf("Subscriber[1].TargetDirectory = %q, want %q", subscribers[1].TargetDirectory, "cfe/common")
	}
	if subscribers[1].TargetBranch != "develop" {
		t.Errorf("Subscriber[1].TargetBranch = %q, want %q", subscribers[1].TargetBranch, "develop")
	}
}

// base64EncodeString кодирует строку в base64
func base64EncodeString(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// TestFindSubscribedRepos_NoSubscribers проверяет случай без подписчиков (project.yaml без нужных подписок)
func TestFindSubscribedRepos_NoSubscribers(t *testing.T) {
	// project.yaml с подпиской на другой репозиторий
	projectYAML := `subscriptions:
  - OtherOrg_other-repo_cfe
`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/user/orgs":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if r.URL.Query().Get("page") == "1" || r.URL.Query().Get("page") == "" {
				_, _ = w.Write([]byte(`[
					{"id": 1, "name": "TargetOrg", "full_name": "Target Organization", "username": "TargetOrg"}
				]`))
			} else {
				_, _ = w.Write([]byte(`[]`))
			}

		case r.URL.Path == "/api/v1/orgs/TargetOrg/repos":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if r.URL.Query().Get("page") == "1" || r.URL.Query().Get("page") == "" {
				_, _ = w.Write([]byte(`[
					{"id": 1, "name": "TargetRepo", "full_name": "TargetOrg/TargetRepo", "owner": {"id": 1, "login": "TargetOrg", "type": "Organization"}, "default_branch": "main", "private": false, "fork": false}
				]`))
			} else {
				_, _ = w.Write([]byte(`[]`))
			}

		// project.yaml с подпиской на другой репозиторий
		case r.URL.Path == "/api/v1/repos/TargetOrg/TargetRepo/contents/project.yaml":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"content": "%s", "encoding": "base64"}`,
				base64EncodeString(projectYAML))))

		default:
			// Файл не найден
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	api := &gitea.API{
		GiteaURL:    server.URL,
		Owner:       "SourceOrg",
		AccessToken: "test-token",
	}

	extensions := []string{"cfe"}
	subscribers, err := FindSubscribedRepos(testLogger(), api, "source-repo", extensions)
	if err != nil {
		t.Fatalf("FindSubscribedRepos вернул ошибку: %v", err)
	}

	if len(subscribers) != 0 {
		t.Errorf("Ожидалось 0 подписчиков, получено %d", len(subscribers))
	}
}

// TestFindSubscribedRepos_EmptyExtensions проверяет случай с пустым списком расширений
func TestFindSubscribedRepos_EmptyExtensions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Сервер не должен получать никаких запросов
		t.Error("Не ожидались запросы к серверу при пустом списке расширений")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	api := &gitea.API{
		GiteaURL:    server.URL,
		Owner:       "SourceOrg",
		AccessToken: "test-token",
	}

	extensions := []string{}
	subscribers, err := FindSubscribedRepos(testLogger(), api, "source-repo", extensions)
	if err != nil {
		t.Fatalf("FindSubscribedRepos вернул ошибку: %v", err)
	}

	if len(subscribers) != 0 {
		t.Errorf("Ожидалось 0 подписчиков, получено %d", len(subscribers))
	}
}

// TestFindSubscribedRepos_NoProjectYAML проверяет случай когда project.yaml не существует
func TestFindSubscribedRepos_NoProjectYAML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/user/orgs":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if r.URL.Query().Get("page") == "1" || r.URL.Query().Get("page") == "" {
				_, _ = w.Write([]byte(`[
					{"id": 1, "name": "TargetOrg", "full_name": "Target Organization", "username": "TargetOrg"}
				]`))
			} else {
				_, _ = w.Write([]byte(`[]`))
			}

		case r.URL.Path == "/api/v1/orgs/TargetOrg/repos":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if r.URL.Query().Get("page") == "1" || r.URL.Query().Get("page") == "" {
				_, _ = w.Write([]byte(`[
					{"id": 1, "name": "TargetRepo", "full_name": "TargetOrg/TargetRepo", "owner": {"id": 1, "login": "TargetOrg", "type": "Organization"}, "default_branch": "main", "private": false, "fork": false}
				]`))
			} else {
				_, _ = w.Write([]byte(`[]`))
			}

		default:
			// project.yaml не существует - возвращаем 404
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	api := &gitea.API{
		GiteaURL:    server.URL,
		Owner:       "SourceOrg",
		AccessToken: "test-token",
	}

	extensions := []string{"cfe"}
	subscribers, err := FindSubscribedRepos(testLogger(), api, "source-repo", extensions)
	if err != nil {
		t.Fatalf("FindSubscribedRepos вернул ошибку: %v", err)
	}

	if len(subscribers) != 0 {
		t.Errorf("Ожидалось 0 подписчиков (project.yaml не существует), получено %d", len(subscribers))
	}
}

// TestFindSubscribedRepos_OrgsAPIError проверяет обработку ошибки при получении организаций
func TestFindSubscribedRepos_OrgsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/user/orgs" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message": "internal error"}`))
		}
	}))
	defer server.Close()

	api := &gitea.API{
		GiteaURL:    server.URL,
		Owner:       "SourceOrg",
		AccessToken: "test-token",
	}

	extensions := []string{"cfe"}
	_, err := FindSubscribedRepos(testLogger(), api, "source-repo", extensions)
	if err == nil {
		t.Fatal("Ожидалась ошибка при проблеме с API организаций")
	}
}

// TestFindSubscribedRepos_OrgReposAPIError проверяет продолжение работы при ошибке получения репозиториев организации
func TestFindSubscribedRepos_OrgReposAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/user/orgs":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if r.URL.Query().Get("page") == "1" || r.URL.Query().Get("page") == "" {
				_, _ = w.Write([]byte(`[
					{"id": 1, "name": "TargetOrg", "full_name": "Target Organization", "username": "TargetOrg"}
				]`))
			} else {
				_, _ = w.Write([]byte(`[]`))
			}

		case r.URL.Path == "/api/v1/orgs/TargetOrg/repos":
			// Ошибка при получении репозиториев
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message": "internal error"}`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	api := &gitea.API{
		GiteaURL:    server.URL,
		Owner:       "SourceOrg",
		AccessToken: "test-token",
	}

	extensions := []string{"cfe"}
	// Функция должна продолжить работу, но не найти подписчиков
	subscribers, err := FindSubscribedRepos(testLogger(), api, "source-repo", extensions)
	if err != nil {
		t.Fatalf("FindSubscribedRepos не должен возвращать ошибку при проблеме с одной организацией: %v", err)
	}

	if len(subscribers) != 0 {
		t.Errorf("Ожидалось 0 подписчиков, получено %d", len(subscribers))
	}
}

// ============================================================================
// Тесты для Story 0.4: Синхронизация каталога расширения
// ============================================================================

// TestSyncResult_Structure проверяет создание структуры SyncResult (AC1)
func TestSyncResult_Structure(t *testing.T) {
	subscriber := SubscribedRepo{
		Organization:    "TestOrg",
		Repository:      "TestRepo",
		TargetBranch:    "main",
		TargetDirectory: "cfe/CommonExt",
		SubscriptionID:  "TestOrg_TestRepo_cfe_CommonExt",
	}

	result := SyncResult{
		Subscriber:   subscriber,
		FilesCreated: 5,
		FilesDeleted: 3,
		NewBranch:    "update-commonext-1.2.3",
		CommitSHA:    "abc123def456",
		Error:        nil,
	}

	// Проверяем все поля структуры
	if result.Subscriber.Organization != "TestOrg" {
		t.Errorf("Subscriber.Organization = %q, want %q", result.Subscriber.Organization, "TestOrg")
	}
	if result.FilesCreated != 5 {
		t.Errorf("FilesCreated = %d, want %d", result.FilesCreated, 5)
	}
	if result.FilesDeleted != 3 {
		t.Errorf("FilesDeleted = %d, want %d", result.FilesDeleted, 3)
	}
	if result.NewBranch != "update-commonext-1.2.3" {
		t.Errorf("NewBranch = %q, want %q", result.NewBranch, "update-commonext-1.2.3")
	}
	if result.CommitSHA != "abc123def456" {
		t.Errorf("CommitSHA = %q, want %q", result.CommitSHA, "abc123def456")
	}
	if result.Error != nil {
		t.Errorf("Error = %v, want nil", result.Error)
	}
}

// TestSyncResult_WithError проверяет SyncResult с ошибкой
func TestSyncResult_WithError(t *testing.T) {
	testErr := fmt.Errorf("тестовая ошибка синхронизации")

	result := SyncResult{
		Subscriber: SubscribedRepo{
			Organization: "TestOrg",
			Repository:   "TestRepo",
		},
		Error: testErr,
	}

	if result.Error == nil {
		t.Fatal("Error должен быть не nil")
	}
	if result.Error.Error() != "тестовая ошибка синхронизации" {
		t.Errorf("Error.Error() = %q, want %q", result.Error.Error(), "тестовая ошибка синхронизации")
	}
}

// TestGetSourceFiles_Success проверяет успешное получение файлов из исходного каталога (AC2)
func TestGetSourceFiles_Success(t *testing.T) {
	// Создаём тестовый сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		// Получение содержимого корневого каталога
		case "/api/v1/repos/SourceOrg/source-repo/contents/extensions/CommonExt":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[
				{"name": "file1.txt", "path": "extensions/CommonExt/file1.txt", "type": "file", "sha": "sha1"},
				{"name": "subdir", "path": "extensions/CommonExt/subdir", "type": "dir", "sha": "sha2"}
			]`))

		// Получение содержимого подкаталога
		case "/api/v1/repos/SourceOrg/source-repo/contents/extensions/CommonExt/subdir":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[
				{"name": "file2.txt", "path": "extensions/CommonExt/subdir/file2.txt", "type": "file", "sha": "sha3"}
			]`))

		// Получение содержимого файла file1.txt
		case "/api/v1/repos/SourceOrg/source-repo/contents/extensions/CommonExt/file1.txt":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// "content1" в base64 = "Y29udGVudDE="
			_, _ = w.Write([]byte(`{"content": "Y29udGVudDE=", "encoding": "base64"}`))

		// Получение содержимого файла file2.txt
		case "/api/v1/repos/SourceOrg/source-repo/contents/extensions/CommonExt/subdir/file2.txt":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// "content2" в base64 = "Y29udGVudDI="
			_, _ = w.Write([]byte(`{"content": "Y29udGVudDI=", "encoding": "base64"}`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	api := &gitea.API{
		GiteaURL:    server.URL,
		Owner:       "SourceOrg",
		Repo:        "source-repo",
		AccessToken: "test-token",
	}

	operations, err := GetSourceFiles(api, "extensions/CommonExt", "main")
	if err != nil {
		t.Fatalf("GetSourceFiles вернул ошибку: %v", err)
	}

	// Ожидаем 2 файла
	if len(operations) != 2 {
		t.Fatalf("Ожидалось 2 операции, получено %d", len(operations))
	}

	// Проверяем первую операцию (file1.txt)
	foundFile1 := false
	foundFile2 := false
	for _, op := range operations {
		if op.Operation != "create" {
			t.Errorf("Operation = %q, want %q", op.Operation, "create")
		}
		if op.Path == "file1.txt" {
			foundFile1 = true
			// "content1" в base64 = "Y29udGVudDE="
			if op.Content != "Y29udGVudDE=" {
				t.Errorf("file1.txt Content = %q, want %q", op.Content, "Y29udGVudDE=")
			}
		}
		if op.Path == "subdir/file2.txt" {
			foundFile2 = true
			// "content2" в base64 = "Y29udGVudDI="
			if op.Content != "Y29udGVudDI=" {
				t.Errorf("file2.txt Content = %q, want %q", op.Content, "Y29udGVudDI=")
			}
		}
	}

	if !foundFile1 {
		t.Error("Не найден file1.txt в операциях")
	}
	if !foundFile2 {
		t.Error("Не найден subdir/file2.txt в операциях")
	}
}

// TestGetSourceFiles_EmptyDirectory проверяет обработку пустого каталога (AC2, AC5)
func TestGetSourceFiles_EmptyDirectory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/repos/SourceOrg/source-repo/contents/empty-dir" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	api := &gitea.API{
		GiteaURL:    server.URL,
		Owner:       "SourceOrg",
		Repo:        "source-repo",
		AccessToken: "test-token",
	}

	_, err := GetSourceFiles(api, "empty-dir", "main")
	if err == nil {
		t.Fatal("Ожидалась ошибка при пустом каталоге")
	}
}

// TestGetSourceFiles_DirectoryNotFound проверяет обработку несуществующего каталога (AC2)
func TestGetSourceFiles_DirectoryNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "not found"}`))
	}))
	defer server.Close()

	api := &gitea.API{
		GiteaURL:    server.URL,
		Owner:       "SourceOrg",
		Repo:        "source-repo",
		AccessToken: "test-token",
	}

	_, err := GetSourceFiles(api, "non-existent", "main")
	if err == nil {
		t.Fatal("Ожидалась ошибка при несуществующем каталоге")
	}
}

// TestGetTargetFilesToDelete_Success проверяет успешное получение операций удаления (AC3)
func TestGetTargetFilesToDelete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		// Получение содержимого целевого каталога
		case "/api/v1/repos/TargetOrg/target-repo/contents/cfe/CommonExt":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[
				{"name": "oldfile1.txt", "path": "cfe/CommonExt/oldfile1.txt", "type": "file", "sha": "sha_old1"},
				{"name": "subdir", "path": "cfe/CommonExt/subdir", "type": "dir", "sha": "sha_dir"}
			]`))

		// Получение содержимого подкаталога
		case "/api/v1/repos/TargetOrg/target-repo/contents/cfe/CommonExt/subdir":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[
				{"name": "oldfile2.txt", "path": "cfe/CommonExt/subdir/oldfile2.txt", "type": "file", "sha": "sha_old2"}
			]`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	api := &gitea.API{
		GiteaURL:    server.URL,
		Owner:       "TargetOrg",
		Repo:        "target-repo",
		AccessToken: "test-token",
	}

	operations, err := GetTargetFilesToDelete(api, "cfe/CommonExt", "main")
	if err != nil {
		t.Fatalf("GetTargetFilesToDelete вернул ошибку: %v", err)
	}

	// Ожидаем 2 операции delete
	if len(operations) != 2 {
		t.Fatalf("Ожидалось 2 операции delete, получено %d", len(operations))
	}

	// Проверяем операции
	for _, op := range operations {
		if op.Operation != "delete" {
			t.Errorf("Operation = %q, want %q", op.Operation, "delete")
		}
		if op.SHA == "" {
			t.Errorf("SHA не должен быть пустым для операции delete")
		}
	}

	// Проверяем наличие конкретных файлов
	foundOld1 := false
	foundOld2 := false
	for _, op := range operations {
		if op.Path == "cfe/CommonExt/oldfile1.txt" {
			foundOld1 = true
			if op.SHA != "sha_old1" {
				t.Errorf("oldfile1.txt SHA = %q, want %q", op.SHA, "sha_old1")
			}
		}
		if op.Path == "cfe/CommonExt/subdir/oldfile2.txt" {
			foundOld2 = true
			if op.SHA != "sha_old2" {
				t.Errorf("oldfile2.txt SHA = %q, want %q", op.SHA, "sha_old2")
			}
		}
	}

	if !foundOld1 {
		t.Error("Не найден cfe/CommonExt/oldfile1.txt в операциях")
	}
	if !foundOld2 {
		t.Error("Не найден cfe/CommonExt/subdir/oldfile2.txt в операциях")
	}
}

// TestGetTargetFilesToDelete_EmptyDirectory проверяет пустой целевой каталог (AC3)
func TestGetTargetFilesToDelete_EmptyDirectory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/repos/TargetOrg/target-repo/contents/cfe/NewExt" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	api := &gitea.API{
		GiteaURL:    server.URL,
		Owner:       "TargetOrg",
		Repo:        "target-repo",
		AccessToken: "test-token",
	}

	operations, err := GetTargetFilesToDelete(api, "cfe/NewExt", "main")
	if err != nil {
		t.Fatalf("GetTargetFilesToDelete вернул ошибку: %v", err)
	}

	// Пустой каталог - 0 операций, это не ошибка
	if len(operations) != 0 {
		t.Errorf("Ожидалось 0 операций для пустого каталога, получено %d", len(operations))
	}
}

// TestGetTargetFilesToDelete_DirectoryNotExists проверяет несуществующий целевой каталог (AC3)
func TestGetTargetFilesToDelete_DirectoryNotExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "not found"}`))
	}))
	defer server.Close()

	api := &gitea.API{
		GiteaURL:    server.URL,
		Owner:       "TargetOrg",
		Repo:        "target-repo",
		AccessToken: "test-token",
	}

	operations, err := GetTargetFilesToDelete(api, "non-existent", "main")
	if err != nil {
		t.Fatalf("GetTargetFilesToDelete вернул ошибку: %v", err)
	}

	// Несуществующий каталог - 0 операций, это не ошибка
	if len(operations) != 0 {
		t.Errorf("Ожидалось 0 операций для несуществующего каталога, получено %d", len(operations))
	}
}

// TestGetTargetFilesToDelete_RecursiveError проверяет ошибку при рекурсивном обходе подкаталога
func TestGetTargetFilesToDelete_RecursiveError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		// Корневой каталог возвращает подкаталог
		case "/api/v1/repos/TargetOrg/target-repo/contents/cfe/CommonExt":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[
				{"name": "subdir", "path": "cfe/CommonExt/subdir", "type": "dir", "sha": "sha_dir"}
			]`))

		// Подкаталог возвращает ошибку сервера (не 404)
		case "/api/v1/repos/TargetOrg/target-repo/contents/cfe/CommonExt/subdir":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message": "internal server error"}`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	api := &gitea.API{
		GiteaURL:    server.URL,
		Owner:       "TargetOrg",
		Repo:        "target-repo",
		AccessToken: "test-token",
	}

	_, err := GetTargetFilesToDelete(api, "cfe/CommonExt", "main")
	if err == nil {
		t.Fatal("Ожидалась ошибка при ошибке рекурсивного обхода")
	}
	if !strings.Contains(err.Error(), "ошибка получения содержимого каталога") {
		t.Errorf("Ошибка должна содержать информацию о каталоге: %v", err)
	}
}

// TestGetTargetFilesToDelete_APIError проверяет обработку HTTP ошибки (не 404)
func TestGetTargetFilesToDelete_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message": "access denied"}`))
	}))
	defer server.Close()

	api := &gitea.API{
		GiteaURL:    server.URL,
		Owner:       "TargetOrg",
		Repo:        "target-repo",
		AccessToken: "test-token",
	}

	_, err := GetTargetFilesToDelete(api, "cfe/CommonExt", "main")
	if err == nil {
		t.Fatal("Ожидалась ошибка при HTTP 403")
	}
	if !strings.Contains(err.Error(), "ошибка получения содержимого каталога") {
		t.Errorf("Ошибка должна содержать информацию о каталоге: %v", err)
	}
}

// TestGenerateBranchName проверяет генерацию имени ветки (AC4)
func TestGenerateBranchName(t *testing.T) {
	tests := []struct {
		name     string
		extName  string
		version  string
		expected string
	}{
		{
			name:     "обычное имя с версией",
			extName:  "CommonExt",
			version:  "v1.2.3",
			expected: "update-commonext-1.2.3",
		},
		{
			name:     "версия без префикса v",
			extName:  "MyExtension",
			version:  "2.0.0",
			expected: "update-myextension-2.0.0",
		},
		{
			name:     "имя с пробелами заменяется на дефисы",
			extName:  "Some Ext",
			version:  "v1.0.0",
			expected: "update-some-ext-1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateBranchName(tt.extName, tt.version)
			if result != tt.expected {
				t.Errorf("GenerateBranchName(%q, %q) = %q, want %q", tt.extName, tt.version, result, tt.expected)
			}
		})
	}
}

// TestGenerateCommitMessage проверяет генерацию commit message (AC4, BR2)
func TestGenerateCommitMessage(t *testing.T) {
	tests := []struct {
		name     string
		extName  string
		version  string
		expected string
	}{
		{
			name:     "стандартный формат",
			extName:  "CommonExt",
			version:  "v1.2.3",
			expected: "chore(ext): update CommonExt to v1.2.3",
		},
		{
			name:     "с пустым v префиксом",
			extName:  "MyExt",
			version:  "1.0.0",
			expected: "chore(ext): update MyExt to 1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateCommitMessage(tt.extName, tt.version)
			if result != tt.expected {
				t.Errorf("GenerateCommitMessage(%q, %q) = %q, want %q", tt.extName, tt.version, result, tt.expected)
			}
		})
	}
}

// TestSyncExtensionToRepo_Success проверяет успешную синхронизацию (AC4)
func TestSyncExtensionToRepo_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		// Source: Получение содержимого исходного каталога
		case r.URL.Path == "/api/v1/repos/SourceOrg/source-repo/contents/extensions/CommonExt" && r.URL.Query().Get("ref") == "main":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[
				{"name": "newfile.txt", "path": "extensions/CommonExt/newfile.txt", "type": "file", "sha": "sha_new"}
			]`))

		// Source: Получение содержимого файла
		case r.URL.Path == "/api/v1/repos/SourceOrg/source-repo/contents/extensions/CommonExt/newfile.txt":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// "new content" в base64 = "bmV3IGNvbnRlbnQ="
			_, _ = w.Write([]byte(`{"content": "bmV3IGNvbnRlbnQ=", "encoding": "base64"}`))

		// Target: Получение содержимого целевого каталога (пустой)
		case r.URL.Path == "/api/v1/repos/TargetOrg/target-repo/contents/cfe/CommonExt":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))

		// Target: Batch commit с созданием ветки
		case r.URL.Path == "/api/v1/repos/TargetOrg/target-repo/contents" && r.Method == "POST":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"commit": {"sha": "commit_sha_123"}}`))

		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"message": "not found: %s"}`, r.URL.Path)))
		}
	}))
	defer server.Close()

	// Исходный репозиторий API
	sourceAPI := &gitea.API{
		GiteaURL:    server.URL,
		Owner:       "SourceOrg",
		Repo:        "source-repo",
		AccessToken: "test-token",
	}

	// Целевой репозиторий API
	targetAPI := &gitea.API{
		GiteaURL:    server.URL,
		Owner:       "TargetOrg",
		Repo:        "target-repo",
		AccessToken: "test-token",
	}

	subscriber := SubscribedRepo{
		Organization:    "TargetOrg",
		Repository:      "target-repo",
		TargetBranch:    "main",
		TargetDirectory: "cfe/CommonExt",
		SubscriptionID:  "TargetOrg_target-repo_cfe_CommonExt",
	}

	result, err := SyncExtensionToRepo(testLogger(), sourceAPI, targetAPI, subscriber, "extensions/CommonExt", "main", "CommonExt", "CommonExt", "v1.2.3")
	if err != nil {
		t.Fatalf("SyncExtensionToRepo вернул ошибку: %v", err)
	}

	// Проверяем результат
	if result.Error != nil {
		t.Errorf("result.Error = %v, want nil", result.Error)
	}
	if result.FilesCreated != 1 {
		t.Errorf("FilesCreated = %d, want %d", result.FilesCreated, 1)
	}
	if result.FilesDeleted != 0 {
		t.Errorf("FilesDeleted = %d, want %d", result.FilesDeleted, 0)
	}
	if result.NewBranch != "update-commonext-1.2.3" {
		t.Errorf("NewBranch = %q, want %q", result.NewBranch, "update-commonext-1.2.3")
	}
}

// TestSyncExtensionToRepo_EmptySource проверяет ошибку при пустом источнике (AC5)
func TestSyncExtensionToRepo_EmptySource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Source: Пустой каталог
		if r.URL.Path == "/api/v1/repos/SourceOrg/source-repo/contents/empty" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	sourceAPI := &gitea.API{
		GiteaURL:    server.URL,
		Owner:       "SourceOrg",
		Repo:        "source-repo",
		AccessToken: "test-token",
	}

	targetAPI := &gitea.API{
		GiteaURL:    server.URL,
		Owner:       "TargetOrg",
		Repo:        "target-repo",
		AccessToken: "test-token",
	}

	subscriber := SubscribedRepo{
		Organization:    "TargetOrg",
		Repository:      "target-repo",
		TargetBranch:    "main",
		TargetDirectory: "cfe/MyExt",
	}

	_, err := SyncExtensionToRepo(testLogger(), sourceAPI, targetAPI, subscriber, "empty", "main", "MyExt", "MyExt", "v1.0.0")
	if err == nil {
		t.Fatal("Ожидалась ошибка при пустом исходном каталоге")
	}
}

// ============================================================================
// Тесты для Story 0.5: Создание PR с полной информацией
// ============================================================================

// TestBuildExtensionPRBody_FullRelease проверяет формирование body с полным релизом (AC3)
func TestBuildExtensionPRBody_FullRelease(t *testing.T) {
	release := &gitea.Release{
		TagName: "v1.2.3",
		Body:    "### Changes\n- Added new feature\n- Fixed bug",
	}

	body := BuildExtensionPRBody(release, "SourceOrg/extensions", "CommonExt", "https://gitea.example.com/SourceOrg/extensions/releases/tag/v1.2.3")

	// Проверяем наличие обязательных элементов
	if !strings.Contains(body, "## Extension Update") {
		t.Error("Body должен содержать заголовок 'Extension Update'")
	}
	if !strings.Contains(body, "**Extension:** CommonExt") {
		t.Error("Body должен содержать имя расширения")
	}
	if !strings.Contains(body, "**Version:** v1.2.3") {
		t.Error("Body должен содержать версию")
	}
	if !strings.Contains(body, "[SourceOrg/extensions]") {
		t.Error("Body должен содержать ссылку на источник")
	}
	if !strings.Contains(body, "### Release Notes") {
		t.Error("Body должен содержать секцию Release Notes")
	}
	if !strings.Contains(body, "Added new feature") {
		t.Error("Body должен содержать release notes")
	}
	if !strings.Contains(body, "benadis-runner extension-publish") {
		t.Error("Body должен содержать подпись автоматического создания")
	}
}

// TestBuildExtensionPRBody_EmptyReleaseNotes проверяет обработку пустых release notes (AC3)
func TestBuildExtensionPRBody_EmptyReleaseNotes(t *testing.T) {
	release := &gitea.Release{
		TagName: "v1.0.0",
		Body:    "",
	}

	body := BuildExtensionPRBody(release, "SourceOrg/extensions", "MyExt", "")

	if !strings.Contains(body, "_No release notes provided._") {
		t.Error("Body должен содержать placeholder для пустых release notes")
	}
}

// TestBuildExtensionPRBody_NilRelease проверяет обработку nil release (AC3)
func TestBuildExtensionPRBody_NilRelease(t *testing.T) {
	body := BuildExtensionPRBody(nil, "SourceOrg/extensions", "TestExt", "")

	if !strings.Contains(body, "**Extension:** TestExt") {
		t.Error("Body должен содержать имя расширения даже без релиза")
	}
	if !strings.Contains(body, "_No release notes provided._") {
		t.Error("Body должен содержать placeholder для nil release")
	}
	// Версия не должна присутствовать при nil release
	if strings.Contains(body, "**Version:**") {
		t.Error("Body не должен содержать версию при nil release")
	}
}

// TestBuildExtensionPRBody_WithoutURL проверяет формирование body без URL (AC3)
func TestBuildExtensionPRBody_WithoutURL(t *testing.T) {
	release := &gitea.Release{
		TagName: "v2.0.0",
		Body:    "Some notes",
	}

	body := BuildExtensionPRBody(release, "SourceOrg/extensions", "MyExt", "")

	// Должен показать источник без ссылки
	if !strings.Contains(body, "**Source:** SourceOrg/extensions") {
		t.Error("Body должен содержать источник без ссылки")
	}
	// Не должен быть markdown ссылкой
	if strings.Contains(body, "[SourceOrg/extensions](") {
		t.Error("Body не должен содержать markdown ссылку без URL")
	}
}

// TestBuildExtensionPRTitle проверяет формирование заголовка PR (BR1)
func TestBuildExtensionPRTitle(t *testing.T) {
	tests := []struct {
		name     string
		extName  string
		version  string
		expected string
	}{
		{
			name:     "стандартный формат",
			extName:  "CommonExt",
			version:  "v1.2.3",
			expected: "Update CommonExt to v1.2.3",
		},
		{
			name:     "без префикса v",
			extName:  "MyExtension",
			version:  "2.0.0",
			expected: "Update MyExtension to 2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title := BuildExtensionPRTitle(tt.extName, tt.version)
			if title != tt.expected {
				t.Errorf("BuildExtensionPRTitle(%q, %q) = %q, want %q", tt.extName, tt.version, title, tt.expected)
			}
		})
	}
}

// TestCreateExtensionPR_Success проверяет успешное создание PR (AC4)
func TestCreateExtensionPR_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/repos/TargetOrg/target-repo/pulls" && r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{
				"id": 123,
				"number": 45,
				"html_url": "https://gitea.example.com/TargetOrg/target-repo/pulls/45",
				"state": "open",
				"title": "Update CommonExt to v1.2.3",
				"body": "## Extension Update..."
			}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	api := &gitea.API{
		GiteaURL:    server.URL,
		Owner:       "TargetOrg",
		Repo:        "target-repo",
		AccessToken: "test-token",
	}

	syncResult := &SyncResult{
		Subscriber: SubscribedRepo{
			Organization:    "TargetOrg",
			Repository:      "target-repo",
			TargetBranch:    "main",
			TargetDirectory: "cfe/CommonExt",
		},
		NewBranch:    "update-commonext-1.2.3",
		FilesCreated: 5,
	}

	release := &gitea.Release{
		TagName: "v1.2.3",
		Body:    "Release notes here",
	}

	pr, err := CreateExtensionPR(testLogger(), api, syncResult, release, "CommonExt", "SourceOrg/extensions", "https://gitea.example.com/release")
	if err != nil {
		t.Fatalf("CreateExtensionPR вернул ошибку: %v", err)
	}

	if pr.Number != 45 {
		t.Errorf("PR.Number = %d, want %d", pr.Number, 45)
	}
	if pr.HTMLURL != "https://gitea.example.com/TargetOrg/target-repo/pulls/45" {
		t.Errorf("PR.HTMLURL = %q, неверный URL", pr.HTMLURL)
	}
}

// TestCreateExtensionPR_ExistingPR проверяет обработку существующего PR (AC5)
func TestCreateExtensionPR_ExistingPR(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		// Попытка создать PR возвращает конфликт
		case r.URL.Path == "/api/v1/repos/TargetOrg/target-repo/pulls" && r.Method == "POST":
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"message": "pull request already exists"}`))

		// Получение списка PR для поиска существующего
		case r.URL.Path == "/api/v1/repos/TargetOrg/target-repo/pulls" && r.Method == "GET":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{
				"id": 100,
				"number": 10,
				"html_url": "https://gitea.example.com/TargetOrg/target-repo/pulls/10",
				"state": "open",
				"title": "Existing PR",
				"body": "...",
				"head": {"ref": "update-commonext-1.2.3"},
				"base": {"ref": "main"}
			}]`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	api := &gitea.API{
		GiteaURL:    server.URL,
		Owner:       "TargetOrg",
		Repo:        "target-repo",
		AccessToken: "test-token",
	}

	syncResult := &SyncResult{
		Subscriber: SubscribedRepo{
			Organization:    "TargetOrg",
			Repository:      "target-repo",
			TargetBranch:    "main",
			TargetDirectory: "cfe/CommonExt",
		},
		NewBranch: "update-commonext-1.2.3",
	}

	release := &gitea.Release{
		TagName: "v1.2.3",
	}

	pr, err := CreateExtensionPR(testLogger(), api, syncResult, release, "CommonExt", "SourceOrg/extensions", "")
	if err != nil {
		t.Fatalf("CreateExtensionPR вернул ошибку: %v", err)
	}

	// Должен вернуть существующий PR
	if pr.Number != 10 {
		t.Errorf("PR.Number = %d, want %d (существующий PR)", pr.Number, 10)
	}
}

// TestCreateExtensionPR_BranchNotExists проверяет ошибку при несуществующей ветке (AC5)
func TestCreateExtensionPR_BranchNotExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "branch not found"}`))
	}))
	defer server.Close()

	api := &gitea.API{
		GiteaURL:    server.URL,
		Owner:       "TargetOrg",
		Repo:        "target-repo",
		AccessToken: "test-token",
	}

	syncResult := &SyncResult{
		Subscriber: SubscribedRepo{
			TargetBranch: "main",
		},
		NewBranch: "non-existent-branch",
	}

	_, err := CreateExtensionPR(testLogger(), api, syncResult, nil, "TestExt", "SourceOrg/ext", "")
	if err == nil {
		t.Fatal("Ожидалась ошибка при несуществующей ветке")
	}
	if !strings.Contains(err.Error(), "ветка не существует") {
		t.Errorf("Ошибка должна содержать информацию о несуществующей ветке: %v", err)
	}
}

// TestCreateExtensionPR_NilRelease проверяет создание PR без релиза (AC4)
func TestCreateExtensionPR_NilRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/repos/TargetOrg/target-repo/pulls" && r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{
				"id": 200,
				"number": 20,
				"html_url": "https://gitea.example.com/TargetOrg/target-repo/pulls/20",
				"state": "open"
			}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	api := &gitea.API{
		GiteaURL:    server.URL,
		Owner:       "TargetOrg",
		Repo:        "target-repo",
		AccessToken: "test-token",
	}

	syncResult := &SyncResult{
		Subscriber: SubscribedRepo{
			TargetBranch: "main",
		},
		NewBranch: "update-myext-1.0.0",
	}

	pr, err := CreateExtensionPR(testLogger(), api, syncResult, nil, "MyExt", "SourceOrg/ext", "")
	if err != nil {
		t.Fatalf("CreateExtensionPR вернул ошибку: %v", err)
	}

	if pr.Number != 20 {
		t.Errorf("PR.Number = %d, want %d", pr.Number, 20)
	}
}

// TestBuildExtensionPRTitle_EmptyExtName проверяет обработку пустого имени расширения
func TestBuildExtensionPRTitle_EmptyExtName(t *testing.T) {
	title := BuildExtensionPRTitle("", "v1.0.0")
	// Должен вернуть "Update  to v1.0.0" (с двойным пробелом)
	expected := "Update  to v1.0.0"
	if title != expected {
		t.Errorf("BuildExtensionPRTitle(\"\", \"v1.0.0\") = %q, want %q", title, expected)
	}
}

// TestBuildExtensionPRBody_EmptyExtName проверяет обработку пустого имени расширения
func TestBuildExtensionPRBody_EmptyExtName(t *testing.T) {
	release := &gitea.Release{
		TagName: "v1.0.0",
		Body:    "Some notes",
	}

	body := BuildExtensionPRBody(release, "", "", "")

	// Body должен формироваться без ошибок даже с пустыми параметрами
	if !strings.Contains(body, "## Extension Update") {
		t.Error("Body должен содержать заголовок даже с пустыми параметрами")
	}
	if !strings.Contains(body, "**Extension:** ") {
		t.Error("Body должен содержать метку Extension даже с пустым именем")
	}
}

// TestExtensionPublish_MissingEnvVars проверяет валидацию переменных окружения (AC4)
func TestExtensionPublish_MissingEnvVars(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		expectError string
	}{
		{
			name: "отсутствует GITHUB_REPOSITORY",
			envVars: map[string]string{
				"GITHUB_REPOSITORY": "",
				"GITHUB_REF_NAME":   "v1.0.0",
			},
			expectError: "GITHUB_REPOSITORY не установлена",
		},
		{
			name: "отсутствует GITHUB_REF_NAME",
			envVars: map[string]string{
				"GITHUB_REPOSITORY": "owner/repo",
				"GITHUB_REF_NAME":   "",
			},
			expectError: "GITHUB_REF_NAME не установлена",
		},
		{
			name: "некорректный формат GITHUB_REPOSITORY - без слеша",
			envVars: map[string]string{
				"GITHUB_REPOSITORY": "invalidformat",
				"GITHUB_REF_NAME":   "v1.0.0",
			},
			expectError: "некорректный формат GITHUB_REPOSITORY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сохраняем исходные значения
			origRepo := os.Getenv("GITHUB_REPOSITORY")
			origRef := os.Getenv("GITHUB_REF_NAME")
			defer func() {
				os.Setenv("GITHUB_REPOSITORY", origRepo)
				os.Setenv("GITHUB_REF_NAME", origRef)
			}()

			// Устанавливаем тестовые значения
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// Создаём минимальную конфигурацию
			cfg := &config.Config{
				GiteaURL:    "https://gitea.example.com",
				AccessToken: "test-token",
			}

			// Создаём логгер
			l := slog.New(slog.NewTextHandler(io.Discard, nil))

			// Вызываем функцию
			err := ExtensionPublish(nil, l, cfg)

			// Проверяем ошибку
			if err == nil {
				t.Fatalf("ожидалась ошибка, содержащая %q", tt.expectError)
			}
			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("ожидалась ошибка содержащая %q, получено: %v", tt.expectError, err)
			}
		})
	}
}

// TestExtensionPublish_MissingConfig проверяет валидацию конфигурации (Code Review Fix)
func TestExtensionPublish_MissingConfig(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.Config
		expectError string
	}{
		{
			name: "отсутствует GiteaURL",
			cfg: &config.Config{
				GiteaURL:    "",
				AccessToken: "test-token",
			},
			expectError: "GiteaURL не настроен",
		},
		{
			name: "отсутствует AccessToken",
			cfg: &config.Config{
				GiteaURL:    "https://gitea.example.com",
				AccessToken: "",
			},
			expectError: "AccessToken не настроен",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сохраняем исходные значения
			origRepo := os.Getenv("GITHUB_REPOSITORY")
			origRef := os.Getenv("GITHUB_REF_NAME")
			defer func() {
				os.Setenv("GITHUB_REPOSITORY", origRepo)
				os.Setenv("GITHUB_REF_NAME", origRef)
			}()

			// Устанавливаем корректные переменные окружения
			os.Setenv("GITHUB_REPOSITORY", "owner/repo")
			os.Setenv("GITHUB_REF_NAME", "v1.0.0")

			// Создаём логгер
			l := slog.New(slog.NewTextHandler(io.Discard, nil))

			// Вызываем функцию
			err := ExtensionPublish(nil, l, tt.cfg)

			// Проверяем ошибку
			if err == nil {
				t.Fatalf("ожидалась ошибка, содержащая %q", tt.expectError)
			}
			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("ожидалась ошибка содержащая %q, получено: %v", tt.expectError, err)
			}
		})
	}
}

// TestPublishResult_Structure проверяет структуру PublishResult
func TestPublishResult_Structure(t *testing.T) {
	// Проверяем, что структура создается корректно
	sub := SubscribedRepo{
		Organization:    "TestOrg",
		Repository:      "TestRepo",
		TargetBranch:    "main",
		TargetDirectory: "cfe",
	}

	syncResult := &SyncResult{
		Subscriber:   sub,
		FilesCreated: 5,
		FilesDeleted: 2,
		NewBranch:    "update-test-1.0.0",
		CommitSHA:    "abc123",
	}

	result := PublishResult{
		Subscriber: sub,
		SyncResult: syncResult,
		PRNumber:   42,
		PRURL:      "https://gitea.example.com/TestOrg/TestRepo/pulls/42",
		Error:      nil,
	}

	// Проверяем поля
	if result.Subscriber.Organization != "TestOrg" {
		t.Errorf("Subscriber.Organization = %q, want %q", result.Subscriber.Organization, "TestOrg")
	}
	if result.PRNumber != 42 {
		t.Errorf("PRNumber = %d, want %d", result.PRNumber, 42)
	}
	if result.SyncResult.FilesCreated != 5 {
		t.Errorf("SyncResult.FilesCreated = %d, want %d", result.SyncResult.FilesCreated, 5)
	}
}

// TestPublishResult_WithError проверяет PublishResult с ошибкой
func TestPublishResult_WithError(t *testing.T) {
	testErr := fmt.Errorf("тестовая ошибка синхронизации")

	result := PublishResult{
		Subscriber: SubscribedRepo{
			Organization: "TestOrg",
			Repository:   "TestRepo",
		},
		Error: testErr,
	}

	if result.Error == nil {
		t.Fatal("ожидалась ошибка")
	}
	if result.Error.Error() != "тестовая ошибка синхронизации" {
		t.Errorf("Error = %q, want %q", result.Error.Error(), "тестовая ошибка синхронизации")
	}
	if result.PRNumber != 0 {
		t.Errorf("PRNumber при ошибке должен быть 0, получено %d", result.PRNumber)
	}
}

// ============================================================================
// Тесты для Story 0.7: Обработка ошибок и отчётность
// ============================================================================

// TestPublishStatus_Constants проверяет константы PublishStatus (AC2)
func TestPublishStatus_Constants(t *testing.T) {
	tests := []struct {
		name   string
		status PublishStatus
		want   string
	}{
		{"StatusSuccess", StatusSuccess, "success"},
		{"StatusFailed", StatusFailed, "failed"},
		{"StatusSkipped", StatusSkipped, "skipped"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, string(tt.status), tt.want)
			}
		})
	}
}

// TestPublishResult_WithStatus проверяет PublishResult со статусом (AC1)
func TestPublishResult_WithStatus(t *testing.T) {
	sub := SubscribedRepo{
		Organization:    "TestOrg",
		Repository:      "TestRepo",
		TargetDirectory: "cfe/CommonExt",
	}

	result := PublishResult{
		Subscriber:   sub,
		Status:       StatusSuccess,
		PRNumber:     123,
		PRURL:        "https://gitea.example.com/pulls/123",
		ErrorMessage: "",
		DurationMs:   5000, // 5 секунд в миллисекундах
	}

	if result.Status != StatusSuccess {
		t.Errorf("Status = %q, want %q", result.Status, StatusSuccess)
	}
	if result.PRNumber != 123 {
		t.Errorf("PRNumber = %d, want %d", result.PRNumber, 123)
	}
	if result.DurationMs != 5000 {
		t.Errorf("DurationMs = %d, want %d", result.DurationMs, 5000)
	}
}

// TestPublishResult_FailedWithErrorMessage проверяет PublishResult с ошибкой (AC1)
func TestPublishResult_FailedWithErrorMessage(t *testing.T) {
	result := PublishResult{
		Subscriber: SubscribedRepo{
			Organization: "TestOrg",
			Repository:   "TestRepo",
		},
		Status:       StatusFailed,
		Error:        fmt.Errorf("permission denied"),
		ErrorMessage: "permission denied",
		DurationMs:   2000, // 2 секунды в миллисекундах
	}

	if result.Status != StatusFailed {
		t.Errorf("Status = %q, want %q", result.Status, StatusFailed)
	}
	if result.ErrorMessage != "permission denied" {
		t.Errorf("ErrorMessage = %q, want %q", result.ErrorMessage, "permission denied")
	}
}

// TestPublishReport_Structure проверяет структуру PublishReport (AC3)
func TestPublishReport_Structure(t *testing.T) {
	startTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 1, 15, 10, 0, 45, 0, time.UTC)

	report := &PublishReport{
		ExtensionName: "CommonExt",
		Version:       "v1.2.3",
		SourceRepo:    "APKHolding/CommonExtRepo",
		StartTime:     startTime,
		EndTime:       endTime,
		Results:       []PublishResult{},
	}

	if report.ExtensionName != "CommonExt" {
		t.Errorf("ExtensionName = %q, want %q", report.ExtensionName, "CommonExt")
	}
	if report.Version != "v1.2.3" {
		t.Errorf("Version = %q, want %q", report.Version, "v1.2.3")
	}
	if report.SourceRepo != "APKHolding/CommonExtRepo" {
		t.Errorf("SourceRepo = %q, want %q", report.SourceRepo, "APKHolding/CommonExtRepo")
	}
}

// TestPublishReport_SuccessCount проверяет метод SuccessCount (AC3)
func TestPublishReport_SuccessCount(t *testing.T) {
	report := &PublishReport{
		Results: []PublishResult{
			{Status: StatusSuccess},
			{Status: StatusSuccess},
			{Status: StatusFailed},
			{Status: StatusSkipped},
			{Status: StatusSuccess},
		},
	}

	if got := report.SuccessCount(); got != 3 {
		t.Errorf("SuccessCount() = %d, want %d", got, 3)
	}
}

// TestPublishReport_FailedCount проверяет метод FailedCount (AC3)
func TestPublishReport_FailedCount(t *testing.T) {
	report := &PublishReport{
		Results: []PublishResult{
			{Status: StatusSuccess},
			{Status: StatusFailed},
			{Status: StatusFailed},
			{Status: StatusSkipped},
		},
	}

	if got := report.FailedCount(); got != 2 {
		t.Errorf("FailedCount() = %d, want %d", got, 2)
	}
}

// TestPublishReport_SkippedCount проверяет метод SkippedCount (AC3)
func TestPublishReport_SkippedCount(t *testing.T) {
	report := &PublishReport{
		Results: []PublishResult{
			{Status: StatusSuccess},
			{Status: StatusSkipped},
			{Status: StatusSkipped},
			{Status: StatusSkipped},
		},
	}

	if got := report.SkippedCount(); got != 3 {
		t.Errorf("SkippedCount() = %d, want %d", got, 3)
	}
}

// TestPublishReport_HasErrors проверяет метод HasErrors (AC3)
func TestPublishReport_HasErrors(t *testing.T) {
	tests := []struct {
		name    string
		results []PublishResult
		want    bool
	}{
		{
			name: "без ошибок",
			results: []PublishResult{
				{Status: StatusSuccess},
				{Status: StatusSkipped},
			},
			want: false,
		},
		{
			name: "с ошибками",
			results: []PublishResult{
				{Status: StatusSuccess},
				{Status: StatusFailed},
			},
			want: true,
		},
		{
			name:    "пустой список",
			results: []PublishResult{},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &PublishReport{Results: tt.results}
			if got := report.HasErrors(); got != tt.want {
				t.Errorf("HasErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestPublishReport_TotalDuration проверяет метод TotalDuration (AC3)
func TestPublishReport_TotalDuration(t *testing.T) {
	startTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 1, 15, 10, 0, 45, 0, time.UTC)

	report := &PublishReport{
		StartTime: startTime,
		EndTime:   endTime,
	}

	expected := 45 * time.Second
	if got := report.TotalDuration(); got != expected {
		t.Errorf("TotalDuration() = %v, want %v", got, expected)
	}
}

// TestPublishReport_EmptyResults проверяет отчёт с пустыми результатами
func TestPublishReport_EmptyResults(t *testing.T) {
	report := &PublishReport{
		ExtensionName: "TestExt",
		Version:       "v1.0.0",
		Results:       []PublishResult{},
	}

	if report.SuccessCount() != 0 {
		t.Errorf("SuccessCount() должен быть 0 для пустого списка")
	}
	if report.FailedCount() != 0 {
		t.Errorf("FailedCount() должен быть 0 для пустого списка")
	}
	if report.SkippedCount() != 0 {
		t.Errorf("SkippedCount() должен быть 0 для пустого списка")
	}
	if report.HasErrors() {
		t.Errorf("HasErrors() должен быть false для пустого списка")
	}
}

// TestReportResults_TextOutput проверяет текстовый вывод отчёта (AC4)
func TestReportResults_TextOutput(t *testing.T) {
	// Убеждаемся что BR_OUTPUT_JSON не установлен
	origJSON := os.Getenv("BR_OUTPUT_JSON")
	defer os.Setenv("BR_OUTPUT_JSON", origJSON)
	os.Setenv("BR_OUTPUT_JSON", "")

	report := &PublishReport{
		ExtensionName: "TestExt",
		Version:       "v1.0.0",
		SourceRepo:    "TestOrg/TestRepo",
		StartTime:     time.Now(),
		EndTime:       time.Now().Add(30 * time.Second),
		Results: []PublishResult{
			{
				Subscriber: SubscribedRepo{Organization: "Org1", Repository: "Repo1"},
				Status:     StatusSuccess,
				PRNumber:   100,
				PRURL:      "https://example.com/pulls/100",
			},
		},
	}

	// ReportResults не должен возвращать ошибку для текстового вывода
	err := ReportResults(report, testLogger())
	if err != nil {
		t.Errorf("ReportResults вернул ошибку: %v", err)
	}
}

// TestReportResults_JSONOutput проверяет JSON вывод отчёта (AC5)
func TestReportResults_JSONOutput(t *testing.T) {
	// Устанавливаем BR_OUTPUT_JSON=true
	origJSON := os.Getenv("BR_OUTPUT_JSON")
	defer os.Setenv("BR_OUTPUT_JSON", origJSON)
	os.Setenv("BR_OUTPUT_JSON", "true")

	// Перенаправляем stdout для захвата JSON
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	report := &PublishReport{
		ExtensionName: "TestExt",
		Version:       "v1.0.0",
		SourceRepo:    "TestOrg/TestRepo",
		StartTime:     time.Now(),
		EndTime:       time.Now().Add(30 * time.Second),
		Results: []PublishResult{
			{
				Subscriber: SubscribedRepo{Organization: "Org1", Repository: "Repo1"},
				Status:     StatusSuccess,
				PRNumber:   100,
				PRURL:      "https://example.com/pulls/100",
			},
			{
				Subscriber:   SubscribedRepo{Organization: "Org2", Repository: "Repo2"},
				Status:       StatusFailed,
				ErrorMessage: "test error",
			},
		},
	}

	err := ReportResults(report, testLogger())

	// Восстанавливаем stdout
	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("ReportResults вернул ошибку: %v", err)
	}

	// Проверяем что вывод является валидным JSON
	var jsonOutput ReportJSONOutput
	if err := json.Unmarshal(out, &jsonOutput); err != nil {
		t.Fatalf("Ошибка парсинга JSON: %v\nВывод: %s", err, string(out))
	}

	// Проверяем содержимое JSON
	if jsonOutput.ExtensionName != "TestExt" {
		t.Errorf("JSON ExtensionName = %q, want %q", jsonOutput.ExtensionName, "TestExt")
	}
	if jsonOutput.Version != "v1.0.0" {
		t.Errorf("JSON Version = %q, want %q", jsonOutput.Version, "v1.0.0")
	}
	if len(jsonOutput.Results) != 2 {
		t.Errorf("JSON Results length = %d, want %d", len(jsonOutput.Results), 2)
	}

	// Проверяем Summary
	if jsonOutput.Summary.Total != 2 {
		t.Errorf("Summary.Total = %d, want %d", jsonOutput.Summary.Total, 2)
	}
	if jsonOutput.Summary.Success != 1 {
		t.Errorf("Summary.Success = %d, want %d", jsonOutput.Summary.Success, 1)
	}
	if jsonOutput.Summary.Failed != 1 {
		t.Errorf("Summary.Failed = %d, want %d", jsonOutput.Summary.Failed, 1)
	}
}

// TestReportJSONOutput_Structure проверяет структуру JSON вывода (AC5)
func TestReportJSONOutput_Structure(t *testing.T) {
	output := ReportJSONOutput{
		ExtensionName: "TestExt",
		Version:       "v1.0.0",
		SourceRepo:    "TestOrg/TestRepo",
		StartTime:     time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		EndTime:       time.Date(2025, 1, 15, 10, 0, 45, 0, time.UTC),
		Results:       []PublishResult{},
		Summary: ReportSummary{
			Total:   5,
			Success: 3,
			Failed:  1,
			Skipped: 1,
		},
	}

	// Проверяем сериализацию в JSON
	jsonBytes, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Ошибка сериализации в JSON: %v", err)
	}

	// Проверяем наличие обязательных полей
	jsonStr := string(jsonBytes)
	requiredFields := []string{
		`"extension_name"`,
		`"version"`,
		`"source_repo"`,
		`"start_time"`,
		`"end_time"`,
		`"results"`,
		`"summary"`,
		`"total"`,
		`"success"`,
		`"failed"`,
		`"skipped"`,
	}

	for _, field := range requiredFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON должен содержать поле %s", field)
		}
	}
}

// TestPublishResult_JSONSerialization проверяет JSON сериализацию PublishResult (AC5)
func TestPublishResult_JSONSerialization(t *testing.T) {
	result := PublishResult{
		Subscriber: SubscribedRepo{
			Organization:    "TestOrg",
			Repository:      "TestRepo",
			TargetDirectory: "cfe/CommonExt",
		},
		Status:       StatusSuccess,
		PRNumber:     123,
		PRURL:        "https://example.com/pulls/123",
		Error:        fmt.Errorf("internal error"), // Не должен сериализоваться
		ErrorMessage: "visible error",
		DurationMs:   5000, // 5 секунд в миллисекундах
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Ошибка сериализации в JSON: %v", err)
	}

	jsonStr := string(jsonBytes)

	// Error не должен сериализоваться (json:"-")
	// Но ErrorMessage должен присутствовать
	if !strings.Contains(jsonStr, `"error":"visible error"`) {
		t.Errorf("JSON должен содержать error_message")
	}

	// Проверяем структуру subscriber
	if !strings.Contains(jsonStr, `"organization":"TestOrg"`) {
		t.Errorf("JSON должен содержать organization подписчика")
	}
	if !strings.Contains(jsonStr, `"repository":"TestRepo"`) {
		t.Errorf("JSON должен содержать repository подписчика")
	}

	// SubscriptionID не должен сериализоваться (json:"-")
	if strings.Contains(jsonStr, `"subscription_id"`) {
		t.Errorf("JSON не должен содержать subscription_id")
	}

	// Проверяем что duration_ms сериализуется корректно в миллисекундах
	if !strings.Contains(jsonStr, `"duration_ms":5000`) {
		t.Errorf("JSON должен содержать duration_ms:5000 (миллисекунды), получено: %s", jsonStr)
	}
}

// ============================================================================
// Дополнительные тесты для повышения покрытия (AC5)
// ============================================================================

// TestReportResultsText_AllStatuses проверяет текстовый вывод отчёта со всеми статусами
func TestReportResultsText_AllStatuses(t *testing.T) {
	// Убеждаемся что BR_OUTPUT_JSON не установлен
	origJSON := os.Getenv("BR_OUTPUT_JSON")
	defer os.Setenv("BR_OUTPUT_JSON", origJSON)
	os.Setenv("BR_OUTPUT_JSON", "")

	report := &PublishReport{
		ExtensionName: "TestExt",
		Version:       "v1.0.0",
		SourceRepo:    "TestOrg/TestRepo",
		StartTime:     time.Now(),
		EndTime:       time.Now().Add(30 * time.Second),
		Results: []PublishResult{
			{
				Subscriber: SubscribedRepo{Organization: "Org1", Repository: "Repo1"},
				Status:     StatusSuccess,
				PRNumber:   100,
				PRURL:      "https://example.com/pulls/100",
			},
			{
				Subscriber:   SubscribedRepo{Organization: "Org2", Repository: "Repo2"},
				Status:       StatusFailed,
				ErrorMessage: "ошибка синхронизации",
			},
			{
				Subscriber:   SubscribedRepo{Organization: "Org3", Repository: "Repo3"},
				Status:       StatusSkipped,
				ErrorMessage: "dry-run mode",
			},
		},
	}

	// ReportResults не должен возвращать ошибку для текстового вывода
	err := ReportResults(report, testLogger())
	if err != nil {
		t.Errorf("ReportResults вернул ошибку: %v", err)
	}
}

// TestReportResultsText_OnlyFailed проверяет текстовый вывод отчёта только с ошибками
func TestReportResultsText_OnlyFailed(t *testing.T) {
	origJSON := os.Getenv("BR_OUTPUT_JSON")
	defer os.Setenv("BR_OUTPUT_JSON", origJSON)
	os.Setenv("BR_OUTPUT_JSON", "")

	report := &PublishReport{
		ExtensionName: "TestExt",
		Version:       "v1.0.0",
		SourceRepo:    "TestOrg/TestRepo",
		StartTime:     time.Now(),
		EndTime:       time.Now().Add(10 * time.Second),
		Results: []PublishResult{
			{
				Subscriber:   SubscribedRepo{Organization: "Org1", Repository: "Repo1"},
				Status:       StatusFailed,
				ErrorMessage: "ошибка 1",
			},
			{
				Subscriber:   SubscribedRepo{Organization: "Org2", Repository: "Repo2"},
				Status:       StatusFailed,
				ErrorMessage: "ошибка 2",
			},
		},
	}

	err := ReportResults(report, testLogger())
	if err != nil {
		t.Errorf("ReportResults вернул ошибку: %v", err)
	}
}

// TestReportResultsText_OnlySkipped проверяет текстовый вывод отчёта только с пропущенными
func TestReportResultsText_OnlySkipped(t *testing.T) {
	origJSON := os.Getenv("BR_OUTPUT_JSON")
	defer os.Setenv("BR_OUTPUT_JSON", origJSON)
	os.Setenv("BR_OUTPUT_JSON", "")

	report := &PublishReport{
		ExtensionName: "TestExt",
		Version:       "v1.0.0",
		SourceRepo:    "TestOrg/TestRepo",
		StartTime:     time.Now(),
		EndTime:       time.Now().Add(5 * time.Second),
		Results: []PublishResult{
			{
				Subscriber:   SubscribedRepo{Organization: "Org1", Repository: "Repo1"},
				Status:       StatusSkipped,
				ErrorMessage: "", // Пустая причина — должна показать "dry-run mode"
			},
			{
				Subscriber:   SubscribedRepo{Organization: "Org2", Repository: "Repo2"},
				Status:       StatusSkipped,
				ErrorMessage: "репозиторий не найден",
			},
		},
	}

	err := ReportResults(report, testLogger())
	if err != nil {
		t.Errorf("ReportResults вернул ошибку: %v", err)
	}
}

// TestExtensionPublish_DryRunMode проверяет dry-run режим команды (AC6)
func TestExtensionPublish_DryRunMode(t *testing.T) {
	// Сохраняем исходные значения
	origRepo := os.Getenv("GITHUB_REPOSITORY")
	origRef := os.Getenv("GITHUB_REF_NAME")
	origExtDir := os.Getenv("BR_EXT_DIR")
	origDryRun := os.Getenv("BR_DRY_RUN")
	defer func() {
		os.Setenv("GITHUB_REPOSITORY", origRepo)
		os.Setenv("GITHUB_REF_NAME", origRef)
		os.Setenv("BR_EXT_DIR", origExtDir)
		os.Setenv("BR_DRY_RUN", origDryRun)
	}()

	// Создаём mock сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		// Запрос релиза по тегу
		case strings.Contains(r.URL.Path, "/releases/tags/"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": 1,
				"tag_name": "v1.0.0",
				"name": "Release 1.0.0",
				"body": "Release notes"
			}`))

		// Запрос веток репозитория
		case strings.Contains(r.URL.Path, "/branches"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[
				{"name": "main"},
				{"name": "TargetOrg_TargetRepo_cfe"}
			]`))

		// Запрос репозиториев организации
		case strings.Contains(r.URL.Path, "/orgs/TargetOrg/repos"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if r.URL.Query().Get("page") == "1" || r.URL.Query().Get("page") == "" {
				_, _ = w.Write([]byte(`[
					{"id": 1, "name": "TargetRepo", "full_name": "TargetOrg/TargetRepo", "owner": {"id": 1, "login": "TargetOrg", "type": "Organization"}, "default_branch": "main"}
				]`))
			} else {
				_, _ = w.Write([]byte(`[]`))
			}

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Устанавливаем переменные окружения
	os.Setenv("GITHUB_REPOSITORY", "SourceOrg/source-repo")
	os.Setenv("GITHUB_REF_NAME", "v1.0.0")
	os.Setenv("BR_EXT_DIR", "")
	os.Setenv("BR_DRY_RUN", "true")

	// Создаём конфигурацию
	cfg := &config.Config{
		GiteaURL:    server.URL,
		AccessToken: "test-token",
	}

	// Вызываем функцию
	err := ExtensionPublish(nil, testLogger(), cfg)

	// В dry-run режиме не должно быть ошибки
	if err != nil {
		t.Errorf("ExtensionPublish в dry-run режиме вернул ошибку: %v", err)
	}
}

// TestExtensionPublish_NoSubscribers проверяет случай без подписчиков
func TestExtensionPublish_NoSubscribers(t *testing.T) {
	origRepo := os.Getenv("GITHUB_REPOSITORY")
	origRef := os.Getenv("GITHUB_REF_NAME")
	defer func() {
		os.Setenv("GITHUB_REPOSITORY", origRepo)
		os.Setenv("GITHUB_REF_NAME", origRef)
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/releases/tags/"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": 1,
				"tag_name": "v1.0.0",
				"name": "Release 1.0.0"
			}`))

		case strings.Contains(r.URL.Path, "/branches"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// Только служебные ветки — без подписчиков
			_, _ = w.Write([]byte(`[
				{"name": "main"},
				{"name": "develop"},
				{"name": "xml"},
				{"name": "edt"}
			]`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	os.Setenv("GITHUB_REPOSITORY", "SourceOrg/source-repo")
	os.Setenv("GITHUB_REF_NAME", "v1.0.0")

	cfg := &config.Config{
		GiteaURL:    server.URL,
		AccessToken: "test-token",
	}

	err := ExtensionPublish(nil, testLogger(), cfg)

	// Без подписчиков не должно быть ошибки
	if err != nil {
		t.Errorf("ExtensionPublish без подписчиков вернул ошибку: %v", err)
	}
}

// TestExtensionPublish_ReleaseNotFound проверяет ошибку при несуществующем релизе
func TestExtensionPublish_ReleaseNotFound(t *testing.T) {
	origRepo := os.Getenv("GITHUB_REPOSITORY")
	origRef := os.Getenv("GITHUB_REF_NAME")
	defer func() {
		os.Setenv("GITHUB_REPOSITORY", origRepo)
		os.Setenv("GITHUB_REF_NAME", origRef)
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Релиз не найден
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "release not found"}`))
	}))
	defer server.Close()

	os.Setenv("GITHUB_REPOSITORY", "SourceOrg/source-repo")
	os.Setenv("GITHUB_REF_NAME", "v999.0.0")

	cfg := &config.Config{
		GiteaURL:    server.URL,
		AccessToken: "test-token",
	}

	err := ExtensionPublish(nil, testLogger(), cfg)

	if err == nil {
		t.Fatal("Ожидалась ошибка при несуществующем релизе")
	}
	if !strings.Contains(err.Error(), "релиз") {
		t.Errorf("Ошибка должна упоминать релиз: %v", err)
	}
}

// TestExtensionPublish_FullFlow проверяет полный рабочий процесс с успешным созданием PR
func TestExtensionPublish_FullFlow(t *testing.T) {
	origRepo := os.Getenv("GITHUB_REPOSITORY")
	origRef := os.Getenv("GITHUB_REF_NAME")
	origExtDir := os.Getenv("BR_EXT_DIR")
	origDryRun := os.Getenv("BR_DRY_RUN")
	origJSON := os.Getenv("BR_OUTPUT_JSON")
	defer func() {
		os.Setenv("GITHUB_REPOSITORY", origRepo)
		os.Setenv("GITHUB_REF_NAME", origRef)
		os.Setenv("BR_EXT_DIR", origExtDir)
		os.Setenv("BR_DRY_RUN", origDryRun)
		os.Setenv("BR_OUTPUT_JSON", origJSON)
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		// Запрос релиза
		case strings.Contains(r.URL.Path, "/releases/tags/"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": 1,
				"tag_name": "v1.2.3",
				"name": "Release 1.2.3",
				"body": "Release notes here"
			}`))

		// Запрос веток
		case r.URL.Path == "/api/v1/repos/SourceOrg/source-repo/branches":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[
				{"name": "main"},
				{"name": "TargetOrg_TargetRepo_cfe"}
			]`))

		// Запрос репозиториев организации
		case strings.Contains(r.URL.Path, "/orgs/TargetOrg/repos"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if r.URL.Query().Get("page") == "1" || r.URL.Query().Get("page") == "" {
				_, _ = w.Write([]byte(`[
					{"id": 1, "name": "TargetRepo", "full_name": "TargetOrg/TargetRepo", "owner": {"id": 1, "login": "TargetOrg", "type": "Organization"}, "default_branch": "main"}
				]`))
			} else {
				_, _ = w.Write([]byte(`[]`))
			}

		// Запрос содержимого исходного каталога
		case strings.Contains(r.URL.Path, "/repos/SourceOrg/source-repo/contents/") && !strings.Contains(r.URL.Path, "TargetOrg"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if strings.HasSuffix(r.URL.Path, "file.txt") {
				// Содержимое файла
				_, _ = w.Write([]byte(`{"content": "Y29udGVudA==", "encoding": "base64"}`))
			} else {
				// Содержимое каталога
				_, _ = w.Write([]byte(`[
					{"name": "file.txt", "path": "file.txt", "type": "file", "sha": "sha1"}
				]`))
			}

		// Запрос содержимого целевого каталога (пустой)
		case strings.Contains(r.URL.Path, "/repos/TargetOrg/TargetRepo/contents/cfe"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))

		// Batch commit
		case strings.Contains(r.URL.Path, "/repos/TargetOrg/TargetRepo/contents") && r.Method == "POST":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"commit": {"sha": "commit_sha_123"}}`))

		// Создание PR
		case strings.Contains(r.URL.Path, "/repos/TargetOrg/TargetRepo/pulls") && r.Method == "POST":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{
				"id": 100,
				"number": 42,
				"html_url": "https://gitea.example.com/TargetOrg/TargetRepo/pulls/42",
				"state": "open"
			}`))

		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"message": "not found: %s"}`, r.URL.Path)))
		}
	}))
	defer server.Close()

	os.Setenv("GITHUB_REPOSITORY", "SourceOrg/source-repo")
	os.Setenv("GITHUB_REF_NAME", "v1.2.3")
	os.Setenv("BR_EXT_DIR", "")
	os.Setenv("BR_DRY_RUN", "")
	os.Setenv("BR_OUTPUT_JSON", "")

	cfg := &config.Config{
		GiteaURL:    server.URL,
		AccessToken: "test-token",
	}

	err := ExtensionPublish(nil, testLogger(), cfg)

	// Успешное выполнение — без ошибки
	if err != nil {
		t.Errorf("ExtensionPublish вернул ошибку: %v", err)
	}
}

// TestExtensionPublish_WithExtDir проверяет публикацию из конкретного каталога
func TestExtensionPublish_WithExtDir(t *testing.T) {
	origRepo := os.Getenv("GITHUB_REPOSITORY")
	origRef := os.Getenv("GITHUB_REF_NAME")
	origExtDir := os.Getenv("BR_EXT_DIR")
	origDryRun := os.Getenv("BR_DRY_RUN")
	defer func() {
		os.Setenv("GITHUB_REPOSITORY", origRepo)
		os.Setenv("GITHUB_REF_NAME", origRef)
		os.Setenv("BR_EXT_DIR", origExtDir)
		os.Setenv("BR_DRY_RUN", origDryRun)
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/releases/tags/"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": 1,
				"tag_name": "v1.0.0",
				"name": "Release 1.0.0"
			}`))

		case strings.Contains(r.URL.Path, "/branches"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[
				{"name": "main"},
				{"name": "TargetOrg_TargetRepo_cfe"}
			]`))

		case strings.Contains(r.URL.Path, "/orgs/TargetOrg/repos"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if r.URL.Query().Get("page") == "1" || r.URL.Query().Get("page") == "" {
				_, _ = w.Write([]byte(`[
					{"id": 1, "name": "TargetRepo", "full_name": "TargetOrg/TargetRepo", "owner": {"id": 1, "login": "TargetOrg", "type": "Organization"}, "default_branch": "main"}
				]`))
			} else {
				_, _ = w.Write([]byte(`[]`))
			}

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	os.Setenv("GITHUB_REPOSITORY", "SourceOrg/source-repo")
	os.Setenv("GITHUB_REF_NAME", "v1.0.0")
	os.Setenv("BR_EXT_DIR", "extensions/MyExt")
	os.Setenv("BR_DRY_RUN", "true")

	cfg := &config.Config{
		GiteaURL:    server.URL,
		AccessToken: "test-token",
	}

	err := ExtensionPublish(nil, testLogger(), cfg)

	if err != nil {
		t.Errorf("ExtensionPublish с BR_EXT_DIR вернул ошибку: %v", err)
	}
}

// TestExtensionPublish_SyncError проверяет обработку ошибки синхронизации (continue on error)
func TestExtensionPublish_SyncError(t *testing.T) {
	origRepo := os.Getenv("GITHUB_REPOSITORY")
	origRef := os.Getenv("GITHUB_REF_NAME")
	origDryRun := os.Getenv("BR_DRY_RUN")
	defer func() {
		os.Setenv("GITHUB_REPOSITORY", origRepo)
		os.Setenv("GITHUB_REF_NAME", origRef)
		os.Setenv("BR_DRY_RUN", origDryRun)
	}()

	// project.yaml с подпиской
	projectYAML := `subscriptions:
  - SourceOrg_source-repo_cfe
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/releases/tags/"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": 1,
				"tag_name": "v1.0.0",
				"name": "Release 1.0.0"
			}`))

		case r.URL.Path == "/api/v1/user/orgs":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if r.URL.Query().Get("page") == "1" || r.URL.Query().Get("page") == "" {
				_, _ = w.Write([]byte(`[{"id": 1, "name": "TargetOrg", "full_name": "Target Organization", "username": "TargetOrg"}]`))
			} else {
				_, _ = w.Write([]byte(`[]`))
			}

		case strings.Contains(r.URL.Path, "/orgs/TargetOrg/repos"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if r.URL.Query().Get("page") == "1" || r.URL.Query().Get("page") == "" {
				_, _ = w.Write([]byte(`[
					{"id": 1, "name": "TargetRepo", "full_name": "TargetOrg/TargetRepo", "owner": {"id": 1, "login": "TargetOrg", "type": "Organization"}, "default_branch": "main"}
				]`))
			} else {
				_, _ = w.Write([]byte(`[]`))
			}

		// project.yaml для TargetOrg/TargetRepo
		case r.URL.Path == "/api/v1/repos/TargetOrg/TargetRepo/contents/project.yaml":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"content": "%s", "encoding": "base64"}`,
				base64EncodeString(projectYAML))))

		// Исходный каталог возвращает пустой массив — ошибка (пустой исходный каталог)
		case strings.Contains(r.URL.Path, "/repos/SourceOrg/source-repo/contents"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	os.Setenv("GITHUB_REPOSITORY", "SourceOrg/source-repo")
	os.Setenv("GITHUB_REF_NAME", "v1.0.0")
	os.Setenv("BR_DRY_RUN", "")

	cfg := &config.Config{
		GiteaURL:    server.URL,
		AccessToken: "test-token",
		Owner:       "SourceOrg",
		Repo:        "source-repo",
		ReleaseTag:  "v1.0.0",
		AddArray:    []string{"cfe"},
	}

	err := ExtensionPublish(nil, testLogger(), cfg)

	// Должна быть ошибка из-за проблемы синхронизации
	if err == nil {
		t.Fatal("Ожидалась ошибка при проблеме синхронизации")
	}
}

// TestExtensionPublish_PRCreationError проверяет обработку ошибки создания PR
func TestExtensionPublish_PRCreationError(t *testing.T) {
	origRepo := os.Getenv("GITHUB_REPOSITORY")
	origRef := os.Getenv("GITHUB_REF_NAME")
	origDryRun := os.Getenv("BR_DRY_RUN")
	defer func() {
		os.Setenv("GITHUB_REPOSITORY", origRepo)
		os.Setenv("GITHUB_REF_NAME", origRef)
		os.Setenv("BR_DRY_RUN", origDryRun)
	}()

	// project.yaml с подпиской
	projectYAML := `subscriptions:
  - SourceOrg_source-repo_cfe
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/releases/tags/"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "tag_name": "v1.0.0"}`))

		case r.URL.Path == "/api/v1/user/orgs":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if r.URL.Query().Get("page") == "1" || r.URL.Query().Get("page") == "" {
				_, _ = w.Write([]byte(`[{"id": 1, "name": "TargetOrg", "full_name": "Target Organization", "username": "TargetOrg"}]`))
			} else {
				_, _ = w.Write([]byte(`[]`))
			}

		case strings.Contains(r.URL.Path, "/orgs/TargetOrg/repos"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if r.URL.Query().Get("page") == "1" || r.URL.Query().Get("page") == "" {
				_, _ = w.Write([]byte(`[{"id": 1, "name": "TargetRepo", "full_name": "TargetOrg/TargetRepo", "owner": {"id": 1, "login": "TargetOrg", "type": "Organization"}, "default_branch": "main"}]`))
			} else {
				_, _ = w.Write([]byte(`[]`))
			}

		// project.yaml для TargetOrg/TargetRepo
		case r.URL.Path == "/api/v1/repos/TargetOrg/TargetRepo/contents/project.yaml":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"content": "%s", "encoding": "base64"}`,
				base64EncodeString(projectYAML))))

		// Исходные файлы
		case strings.Contains(r.URL.Path, "/repos/SourceOrg/source-repo/contents") && !strings.Contains(r.URL.Path, "TargetOrg"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if strings.HasSuffix(r.URL.Path, "file.txt") {
				_, _ = w.Write([]byte(`{"content": "Y29udGVudA==", "encoding": "base64"}`))
			} else {
				_, _ = w.Write([]byte(`[{"name": "file.txt", "path": "file.txt", "type": "file", "sha": "sha1"}]`))
			}

		// Целевой каталог
		case strings.Contains(r.URL.Path, "/repos/TargetOrg/TargetRepo/contents/cfe"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))

		// Batch commit успешен
		case strings.Contains(r.URL.Path, "/repos/TargetOrg/TargetRepo/contents") && r.Method == "POST":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"commit": {"sha": "abc123"}}`))

		// Создание PR — ошибка (ветка не существует)
		case strings.Contains(r.URL.Path, "/repos/TargetOrg/TargetRepo/pulls"):
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message": "branch not found"}`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	os.Setenv("GITHUB_REPOSITORY", "SourceOrg/source-repo")
	os.Setenv("GITHUB_REF_NAME", "v1.0.0")
	os.Setenv("BR_DRY_RUN", "")

	cfg := &config.Config{
		GiteaURL:    server.URL,
		AccessToken: "test-token",
		Owner:       "SourceOrg",
		Repo:        "source-repo",
		ReleaseTag:  "v1.0.0",
		AddArray:    []string{"cfe"},
	}

	err := ExtensionPublish(nil, testLogger(), cfg)

	// Должна быть ошибка из-за проблемы создания PR
	if err == nil {
		t.Fatal("Ожидалась ошибка при проблеме создания PR")
	}
}
