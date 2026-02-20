package gitea

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestGetLatestRelease тестирует получение последнего релиза
func TestGetLatestRelease(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name            string
		responseCode    int
		responseBody    string
		expectError     bool
		expectedTagName string
		expectedID      int64
	}{
		{
			name:         "successful get latest release",
			responseCode: 200,
			responseBody: `{
				"id": 123,
				"tag_name": "v1.0.0",
				"name": "Release 1.0.0",
				"body": "Release notes",
				"assets": [
					{"id": 1, "name": "app.zip", "size": 1024, "browser_download_url": "https://example.com/app.zip"}
				],
				"created_at": "2023-01-01T00:00:00Z",
				"published_at": "2023-01-01T00:00:00Z"
			}`,
			expectError:     false,
			expectedTagName: "v1.0.0",
			expectedID:      123,
		},
		{
			name:         "release not found",
			responseCode: 404,
			responseBody: `{"message":"Not Found"}`,
			expectError:  true,
		},
		{
			name:         "server error",
			responseCode: 500,
			responseBody: `{"message":"Internal Server Error"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/api/v1/repos/testowner/testrepo/releases/latest"
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}
				if r.Method != "GET" {
					t.Errorf("Expected GET method, got %s", r.Method)
				}
				w.WriteHeader(tt.responseCode)
				if _, err := w.Write([]byte(tt.responseBody)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			api := &API{
				GiteaURL:    server.URL,
				Owner:       "testowner",
				Repo:        "testrepo",
				AccessToken: "testtoken",
			}

			release, err := api.GetLatestRelease(ctx)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if release == nil {
				t.Error("Expected release but got nil")
				return
			}

			if release.ID != tt.expectedID {
				t.Errorf("Expected ID %d, got %d", tt.expectedID, release.ID)
			}

			if release.TagName != tt.expectedTagName {
				t.Errorf("Expected tag name %s, got %s", tt.expectedTagName, release.TagName)
			}
		})
	}
}

// TestGetReleaseByTag тестирует получение релиза по тегу
func TestGetReleaseByTag(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name            string
		tag             string
		expectedPath    string
		responseCode    int
		responseBody    string
		expectError     bool
		expectedTagName string
		expectedID      int64
	}{
		{
			name:         "successful get release by tag",
			tag:          "v1.0.0",
			expectedPath: "/api/v1/repos/testowner/testrepo/releases/tags/v1.0.0",
			responseCode: 200,
			responseBody: `{
				"id": 456,
				"tag_name": "v1.0.0",
				"name": "Release 1.0.0",
				"body": "Release notes for v1.0.0",
				"assets": [],
				"created_at": "2023-01-01T00:00:00Z",
				"published_at": "2023-01-01T00:00:00Z"
			}`,
			expectError:     false,
			expectedTagName: "v1.0.0",
			expectedID:      456,
		},
		{
			name:         "tag with special characters",
			tag:          "release/v2.0.0",
			expectedPath: "/api/v1/repos/testowner/testrepo/releases/tags/release%2Fv2.0.0",
			responseCode: 200,
			responseBody: `{
				"id": 789,
				"tag_name": "release/v2.0.0",
				"name": "Release 2.0.0",
				"body": "Release notes for v2.0.0",
				"assets": [],
				"created_at": "2023-06-01T00:00:00Z",
				"published_at": "2023-06-01T00:00:00Z"
			}`,
			expectError:     false,
			expectedTagName: "release/v2.0.0",
			expectedID:      789,
		},
		{
			name:         "tag not found",
			tag:          "v999.0.0",
			expectedPath: "/api/v1/repos/testowner/testrepo/releases/tags/v999.0.0",
			responseCode: 404,
			responseBody: `{"message":"Not Found"}`,
			expectError:  true,
		},
		{
			name:         "server error",
			tag:          "v1.0.0",
			expectedPath: "/api/v1/repos/testowner/testrepo/releases/tags/v1.0.0",
			responseCode: 500,
			responseBody: `{"message":"Internal Server Error"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Используем RawPath для проверки закодированного пути,
				// если RawPath пустой, используем Path (URL без спецсимволов)
				actualPath := r.URL.RawPath
				if actualPath == "" {
					actualPath = r.URL.Path
				}
				if actualPath != tt.expectedPath {
					t.Errorf("Expected path %s, got %s", tt.expectedPath, actualPath)
				}
				if r.Method != "GET" {
					t.Errorf("Expected GET method, got %s", r.Method)
				}
				w.WriteHeader(tt.responseCode)
				if _, err := w.Write([]byte(tt.responseBody)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			api := &API{
				GiteaURL:    server.URL,
				Owner:       "testowner",
				Repo:        "testrepo",
				AccessToken: "testtoken",
			}

			release, err := api.GetReleaseByTag(ctx, tt.tag)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if release == nil {
				t.Error("Expected release but got nil")
				return
			}

			if release.ID != tt.expectedID {
				t.Errorf("Expected ID %d, got %d", tt.expectedID, release.ID)
			}

			if release.TagName != tt.expectedTagName {
				t.Errorf("Expected tag name %s, got %s", tt.expectedTagName, release.TagName)
			}
		})
	}
}

// TestReleaseWithAssets тестирует корректную десериализацию релиза с ассетами
func TestReleaseWithAssets(t *testing.T) {
	ctx := context.Background()
	responseBody := `{
		"id": 100,
		"tag_name": "v3.0.0",
		"name": "Major Release 3.0.0",
		"body": "# Release Notes\n\n- Feature 1\n- Feature 2",
		"assets": [
			{"id": 10, "name": "linux-amd64.tar.gz", "size": 10485760, "browser_download_url": "https://example.com/linux-amd64.tar.gz"},
			{"id": 11, "name": "windows-amd64.zip", "size": 15728640, "browser_download_url": "https://example.com/windows-amd64.zip"},
			{"id": 12, "name": "darwin-amd64.tar.gz", "size": 11534336, "browser_download_url": "https://example.com/darwin-amd64.tar.gz"}
		],
		"created_at": "2023-12-01T12:00:00Z",
		"published_at": "2023-12-01T14:00:00Z"
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if _, err := w.Write([]byte(responseBody)); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	api := &API{
		GiteaURL:    server.URL,
		Owner:       "testowner",
		Repo:        "testrepo",
		AccessToken: "testtoken",
	}

	release, err := api.GetLatestRelease(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Проверяем основные поля
	if release.ID != 100 {
		t.Errorf("Expected ID 100, got %d", release.ID)
	}

	if release.TagName != "v3.0.0" {
		t.Errorf("Expected tag name v3.0.0, got %s", release.TagName)
	}

	if release.Name != "Major Release 3.0.0" {
		t.Errorf("Expected name 'Major Release 3.0.0', got %s", release.Name)
	}

	// Проверяем ассеты
	if len(release.Assets) != 3 {
		t.Fatalf("Expected 3 assets, got %d", len(release.Assets))
	}

	expectedAssets := []struct {
		ID          int64
		Name        string
		Size        int64
		DownloadURL string
	}{
		{10, "linux-amd64.tar.gz", 10485760, "https://example.com/linux-amd64.tar.gz"},
		{11, "windows-amd64.zip", 15728640, "https://example.com/windows-amd64.zip"},
		{12, "darwin-amd64.tar.gz", 11534336, "https://example.com/darwin-amd64.tar.gz"},
	}

	for i, expected := range expectedAssets {
		asset := release.Assets[i]
		if asset.ID != expected.ID {
			t.Errorf("Asset %d: expected ID %d, got %d", i, expected.ID, asset.ID)
		}
		if asset.Name != expected.Name {
			t.Errorf("Asset %d: expected name %s, got %s", i, expected.Name, asset.Name)
		}
		if asset.Size != expected.Size {
			t.Errorf("Asset %d: expected size %d, got %d", i, expected.Size, asset.Size)
		}
		if asset.DownloadURL != expected.DownloadURL {
			t.Errorf("Asset %d: expected download URL %s, got %s", i, expected.DownloadURL, asset.DownloadURL)
		}
	}

	// Проверяем даты
	if release.CreatedAt != "2023-12-01T12:00:00Z" {
		t.Errorf("Expected created_at '2023-12-01T12:00:00Z', got %s", release.CreatedAt)
	}

	if release.PublishedAt != "2023-12-01T14:00:00Z" {
		t.Errorf("Expected published_at '2023-12-01T14:00:00Z', got %s", release.PublishedAt)
	}
}

// TestReleaseStructJSON тестирует корректную сериализацию/десериализацию структур Release и ReleaseAsset
func TestReleaseStructJSON(t *testing.T) {
	original := Release{
		ID:          42,
		TagName:     "v1.2.3",
		Name:        "Test Release",
		Body:        "Release description",
		Assets: []ReleaseAsset{
			{
				ID:          1,
				Name:        "file.zip",
				Size:        1024,
				DownloadURL: "https://example.com/file.zip",
			},
		},
		CreatedAt:   "2023-01-01T00:00:00Z",
		PublishedAt: "2023-01-01T01:00:00Z",
	}

	// Сериализация в JSON
	jsonBytes, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal release: %v", err)
	}

	// Десериализация обратно
	var decoded Release
	err = json.Unmarshal(jsonBytes, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal release: %v", err)
	}

	// Проверка соответствия
	if decoded.ID != original.ID {
		t.Errorf("ID mismatch: expected %d, got %d", original.ID, decoded.ID)
	}
	if decoded.TagName != original.TagName {
		t.Errorf("TagName mismatch: expected %s, got %s", original.TagName, decoded.TagName)
	}
	if decoded.Name != original.Name {
		t.Errorf("Name mismatch: expected %s, got %s", original.Name, decoded.Name)
	}
	if decoded.Body != original.Body {
		t.Errorf("Body mismatch: expected %s, got %s", original.Body, decoded.Body)
	}
	if len(decoded.Assets) != len(original.Assets) {
		t.Fatalf("Assets count mismatch: expected %d, got %d", len(original.Assets), len(decoded.Assets))
	}
	if decoded.Assets[0].DownloadURL != original.Assets[0].DownloadURL {
		t.Errorf("Asset DownloadURL mismatch: expected %s, got %s", original.Assets[0].DownloadURL, decoded.Assets[0].DownloadURL)
	}
}

// TestGetLatestReleaseInvalidJSON тестирует обработку некорректного JSON
func TestGetLatestReleaseInvalidJSON(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if _, err := w.Write([]byte("invalid json")); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	api := &API{
		GiteaURL:    server.URL,
		Owner:       "testowner",
		Repo:        "testrepo",
		AccessToken: "testtoken",
	}

	_, err := api.GetLatestRelease(ctx)
	if err == nil {
		t.Error("Expected error for invalid JSON but got none")
	}
}

// TestGetReleaseByTagEmptyTag тестирует обработку пустого тега
func TestGetReleaseByTagEmptyTag(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v1/repos/testowner/testrepo/releases/tags/"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}
		w.WriteHeader(404)
		if _, err := w.Write([]byte(`{"message":"Not Found"}`)); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	api := &API{
		GiteaURL:    server.URL,
		Owner:       "testowner",
		Repo:        "testrepo",
		AccessToken: "testtoken",
	}

	_, err := api.GetReleaseByTag(ctx, "")
	if err == nil {
		t.Error("Expected error for empty tag but got none")
	}
}

// TestGetLatestReleaseEmptyAssets тестирует релиз без ассетов
func TestGetLatestReleaseEmptyAssets(t *testing.T) {
	ctx := context.Background()
	responseBody := `{
		"id": 200,
		"tag_name": "v0.1.0",
		"name": "Initial Release",
		"body": "First release",
		"assets": [],
		"created_at": "2022-01-01T00:00:00Z",
		"published_at": "2022-01-01T00:00:00Z"
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if _, err := w.Write([]byte(responseBody)); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	api := &API{
		GiteaURL:    server.URL,
		Owner:       "testowner",
		Repo:        "testrepo",
		AccessToken: "testtoken",
	}

	release, err := api.GetLatestRelease(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(release.Assets) != 0 {
		t.Errorf("Expected 0 assets, got %d", len(release.Assets))
	}
}

// TestGetLatestReleaseNetworkError тестирует обработку сетевой ошибки
func TestGetLatestReleaseNetworkError(t *testing.T) {
	ctx := context.Background()
	// Создаём и сразу закрываем сервер для симуляции недоступного хоста
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close() // Закрываем сразу - соединение будет отклонено

	api := &API{
		GiteaURL:    serverURL,
		Owner:       "testowner",
		Repo:        "testrepo",
		AccessToken: "testtoken",
	}

	_, err := api.GetLatestRelease(ctx)
	if err == nil {
		t.Error("Expected network error but got none")
	}

	// Проверяем, что ошибка содержит информацию о проблеме с запросом
	if !strings.Contains(err.Error(), "ошибка при выполнении запроса") {
		t.Errorf("Expected error message to contain 'ошибка при выполнении запроса', got: %s", err.Error())
	}
}

// TestGetReleaseByTagNetworkError тестирует обработку сетевой ошибки для GetReleaseByTag
func TestGetReleaseByTagNetworkError(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close()

	api := &API{
		GiteaURL:    serverURL,
		Owner:       "testowner",
		Repo:        "testrepo",
		AccessToken: "testtoken",
	}

	_, err := api.GetReleaseByTag(ctx, "v1.0.0")
	if err == nil {
		t.Error("Expected network error but got none")
	}
}

// TestReleaseAuthorizationHeader тестирует отправку заголовка авторизации
func TestReleaseAuthorizationHeader(t *testing.T) {
	ctx := context.Background()
	expectedToken := "my-secret-token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		expectedAuth := fmt.Sprintf("token %s", expectedToken)
		if authHeader != expectedAuth {
			t.Errorf("Expected Authorization header '%s', got '%s'", expectedAuth, authHeader)
		}

		w.WriteHeader(200)
		if _, err := w.Write([]byte(`{"id":1,"tag_name":"v1.0.0","name":"Test","body":"","assets":[],"created_at":"","published_at":""}`)); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	api := &API{
		GiteaURL:    server.URL,
		Owner:       "testowner",
		Repo:        "testrepo",
		AccessToken: expectedToken,
	}

	_, err := api.GetLatestRelease(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}
