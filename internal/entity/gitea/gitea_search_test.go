package gitea

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRepositoryStructure проверяет корректность структуры Repository и её JSON-маршалинг.
func TestRepositoryStructure(t *testing.T) {
	// Пример JSON ответа от Gitea API
	jsonData := `{
		"id": 123,
		"name": "test-repo",
		"full_name": "myorg/test-repo",
		"owner": {
			"id": 456,
			"login": "myorg",
			"type": "Organization"
		},
		"default_branch": "main",
		"private": false,
		"fork": true
	}`

	var repo Repository
	err := json.Unmarshal([]byte(jsonData), &repo)
	if err != nil {
		t.Fatalf("Ошибка при разборе JSON: %v", err)
	}

	// Проверяем все поля
	if repo.ID != 123 {
		t.Errorf("Ожидался ID=123, получен %d", repo.ID)
	}
	if repo.Name != "test-repo" {
		t.Errorf("Ожидалось Name='test-repo', получено '%s'", repo.Name)
	}
	if repo.FullName != "myorg/test-repo" {
		t.Errorf("Ожидалось FullName='myorg/test-repo', получено '%s'", repo.FullName)
	}
	if repo.Owner.ID != 456 {
		t.Errorf("Ожидался Owner.ID=456, получен %d", repo.Owner.ID)
	}
	if repo.Owner.Login != "myorg" {
		t.Errorf("Ожидалось Owner.Login='myorg', получено '%s'", repo.Owner.Login)
	}
	if repo.Owner.Type != "Organization" {
		t.Errorf("Ожидалось Owner.Type='Organization', получено '%s'", repo.Owner.Type)
	}
	if repo.DefaultBranch != "main" {
		t.Errorf("Ожидалось DefaultBranch='main', получено '%s'", repo.DefaultBranch)
	}
	if repo.Private != false {
		t.Errorf("Ожидалось Private=false, получено %v", repo.Private)
	}
	if repo.Fork != true {
		t.Errorf("Ожидалось Fork=true, получено %v", repo.Fork)
	}
}

// TestSearchOrgRepos_Success проверяет успешный поиск репозиториев организации.
func TestSearchOrgRepos_Success(t *testing.T) {
	// Создаём тестовый сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем путь запроса
		expectedPath := "/api/v1/orgs/testorg/repos"
		if r.URL.Path != expectedPath {
			t.Errorf("Ожидался путь %s, получен %s", expectedPath, r.URL.Path)
		}

		// Проверяем метод
		if r.Method != "GET" {
			t.Errorf("Ожидался метод GET, получен %s", r.Method)
		}

		// Проверяем параметры пагинации
		page := r.URL.Query().Get("page")
		limit := r.URL.Query().Get("limit")

		// На первой странице возвращаем репозитории
		if page == "1" || page == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[
				{
					"id": 1,
					"name": "repo1",
					"full_name": "testorg/repo1",
					"owner": {"id": 100, "login": "testorg", "type": "Organization"},
					"default_branch": "main",
					"private": false,
					"fork": false
				},
				{
					"id": 2,
					"name": "repo2",
					"full_name": "testorg/repo2",
					"owner": {"id": 100, "login": "testorg", "type": "Organization"},
					"default_branch": "develop",
					"private": true,
					"fork": false
				}
			]`))
		} else {
			// На остальных страницах возвращаем пустой массив
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		}

		// Проверяем лимит
		if limit != "50" && limit != "" {
			t.Errorf("Ожидался limit=50, получен %s", limit)
		}
	}))
	defer server.Close()

	// Создаём API клиент
	api := &API{
		GiteaURL:    server.URL,
		AccessToken: "test-token",
	}

	// Вызываем тестируемый метод
	repos, err := api.SearchOrgRepos("testorg")
	if err != nil {
		t.Fatalf("Ошибка при вызове SearchOrgRepos: %v", err)
	}

	// Проверяем результат
	if len(repos) != 2 {
		t.Fatalf("Ожидалось 2 репозитория, получено %d", len(repos))
	}

	if repos[0].Name != "repo1" {
		t.Errorf("Ожидалось Name='repo1', получено '%s'", repos[0].Name)
	}
	if repos[1].Name != "repo2" {
		t.Errorf("Ожидалось Name='repo2', получено '%s'", repos[1].Name)
	}
}

// TestSearchOrgRepos_NotFound проверяет обработку несуществующей организации.
func TestSearchOrgRepos_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Organization not found"}`))
	}))
	defer server.Close()

	api := &API{
		GiteaURL:    server.URL,
		AccessToken: "test-token",
	}

	repos, err := api.SearchOrgRepos("nonexistent")
	if err != nil {
		t.Fatalf("Ошибка не ожидалась для 404, получена: %v", err)
	}

	if len(repos) != 0 {
		t.Errorf("Ожидался пустой slice для 404, получено %d репозиториев", len(repos))
	}
}

// TestSearchOrgRepos_Pagination проверяет обработку пагинации.
func TestSearchOrgRepos_Pagination(t *testing.T) {
	pageCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageCount++
		page := r.URL.Query().Get("page")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Возвращаем по одному репозиторию на первых трёх страницах
		if page == "1" || page == "2" || page == "3" {
			_, _ = w.Write([]byte(`[{"id": ` + page + `, "name": "repo` + page + `", "full_name": "org/repo` + page + `", "owner": {"id": 1, "login": "org", "type": "Organization"}, "default_branch": "main", "private": false, "fork": false}]`))
		} else {
			_, _ = w.Write([]byte(`[]`))
		}
	}))
	defer server.Close()

	api := &API{
		GiteaURL:    server.URL,
		AccessToken: "test-token",
	}

	repos, err := api.SearchOrgRepos("org")
	if err != nil {
		t.Fatalf("Ошибка при вызове SearchOrgRepos: %v", err)
	}

	if len(repos) != 3 {
		t.Errorf("Ожидалось 3 репозитория, получено %d", len(repos))
	}

	// Должно быть 4 запроса: 3 страницы с данными + 1 пустая
	if pageCount != 4 {
		t.Errorf("Ожидалось 4 запроса (3 страницы + пустая), выполнено %d", pageCount)
	}
}

// TestSearchOrgRepos_MaxPages проверяет защиту от бесконечного цикла.
func TestSearchOrgRepos_MaxPages(t *testing.T) {
	pageCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		pageCount++
		// Всегда возвращаем один репозиторий
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"id": 1, "name": "repo", "full_name": "org/repo", "owner": {"id": 1, "login": "org", "type": "Organization"}, "default_branch": "main", "private": false, "fork": false}]`))
	}))
	defer server.Close()

	api := &API{
		GiteaURL:    server.URL,
		AccessToken: "test-token",
	}

	repos, err := api.SearchOrgRepos("org")
	if err != nil {
		t.Fatalf("Ошибка при вызове SearchOrgRepos: %v", err)
	}

	// Должно быть ровно 100 запросов (maxPages)
	if pageCount != 100 {
		t.Errorf("Ожидалось 100 запросов (maxPages), выполнено %d", pageCount)
	}

	// Должно быть 100 репозиториев (по одному на страницу)
	if len(repos) != 100 {
		t.Errorf("Ожидалось 100 репозиториев, получено %d", len(repos))
	}
}

// TestSearchOrgRepos_NetworkError проверяет обработку сетевых ошибок.
func TestSearchOrgRepos_NetworkError(t *testing.T) {
	api := &API{
		GiteaURL:    "http://invalid-host-that-does-not-exist:12345",
		AccessToken: "test-token",
	}

	_, err := api.SearchOrgRepos("org")
	if err == nil {
		t.Fatal("Ожидалась ошибка при сетевой проблеме")
	}
}

// TestSearchOrgRepos_InvalidJSON проверяет обработку невалидного JSON в ответе.
func TestSearchOrgRepos_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`invalid json response`))
	}))
	defer server.Close()

	api := &API{
		GiteaURL:    server.URL,
		AccessToken: "test-token",
	}

	_, err := api.SearchOrgRepos("testorg")
	if err == nil {
		t.Fatal("Ожидалась ошибка при невалидном JSON")
	}

	// Проверяем что ошибка содержит информацию о проблеме с JSON
	if !strings.Contains(err.Error(), "JSON") {
		t.Errorf("Ожидалась ошибка с упоминанием JSON, получена: %v", err)
	}
}

// TestSearchOrgRepos_ServerError проверяет обработку HTTP ошибок сервера (500, 403 и т.д.).
func TestSearchOrgRepos_ServerError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"Internal Server Error", http.StatusInternalServerError},
		{"Forbidden", http.StatusForbidden},
		{"Bad Gateway", http.StatusBadGateway},
		{"Service Unavailable", http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(`{"message": "error"}`))
			}))
			defer server.Close()

			api := &API{
				GiteaURL:    server.URL,
				AccessToken: "test-token",
			}

			_, err := api.SearchOrgRepos("testorg")
			if err == nil {
				t.Fatalf("Ожидалась ошибка для статуса %d", tt.statusCode)
			}
		})
	}
}
