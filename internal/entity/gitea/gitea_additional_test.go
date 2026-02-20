package gitea

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestCreateTestBranch тестирует создание тестовой ветки
func TestCreateTestBranch(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name         string
		responseCode int
		expectError  bool
	}{
		{
			name:         "successful create branch",
			responseCode: 201,
			expectError:  false,
		},
		{
			name:         "failed create branch",
			responseCode: 400,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST method, got %s", r.Method)
				}
				w.WriteHeader(tt.responseCode)
			}))
			defer server.Close()

			api := &API{
				GiteaURL:   server.URL,
				Owner:      "testowner",
				Repo:       "testrepo",
				BaseBranch: "main",
				NewBranch:  "test-branch",
			}

			err := api.CreateTestBranch(ctx)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestDeleteTestBranch тестирует удаление тестовой ветки
func TestDeleteTestBranch(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name         string
		responseCode int
		expectError  bool
	}{
		{
			name:         "successful delete branch",
			responseCode: 204,
			expectError:  false,
		},
		{
			name:         "failed delete branch",
			responseCode: 404,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "DELETE" {
					t.Errorf("Expected DELETE method, got %s", r.Method)
				}
				w.WriteHeader(tt.responseCode)
			}))
			defer server.Close()

			api := &API{
				GiteaURL:  server.URL,
				Owner:     "testowner",
				Repo:      "testrepo",
				NewBranch: "test-branch",
			}

			err := api.DeleteTestBranch(ctx)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestMergePR тестирует слияние Pull Request
func TestMergePR(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name         string
		prNumber     int64
		responseCode int
		expectError  bool
	}{
		{
			name:         "successful merge PR",
			prNumber:     123,
			responseCode: 200,
			expectError:  false,
		},
		{
			name:         "failed merge PR",
			prNumber:     404,
			responseCode: 404,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST method, got %s", r.Method)
				}
				w.WriteHeader(tt.responseCode)
			}))
			defer server.Close()

			api := &API{
				GiteaURL: server.URL,
				Owner:    "testowner",
				Repo:     "testrepo",
			}

			err := api.MergePR(ctx, tt.prNumber, logger)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestActivePR тестирует получение активных Pull Request
func TestActivePR(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name         string
		responseCode int
		responseBody string
		expectError  bool
		expectedPRs  int
	}{
		{
			name:         "successful get active PRs",
			responseCode: 200,
			responseBody: `[{"id":1,"number":123,"base":{"label":"main","name":"main"},"head":{"label":"feature-1","name":"feature-1"}},{"id":2,"number":124,"base":{"label":"main","name":"main"},"head":{"label":"feature-2","name":"feature-2"}}]`,
			expectError:  false,
			expectedPRs:  2,
		},
		{
			name:         "no active PRs",
			responseCode: 200,
			responseBody: `[]`,
			expectError:  false,
			expectedPRs:  0,
		},
		{
			name:         "failed get active PRs",
			responseCode: 500,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				GiteaURL:   server.URL,
				Owner:      "testowner",
				Repo:       "testrepo",
				BaseBranch: "main",
			}

			prs, err := api.ActivePR(ctx)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(prs) != tt.expectedPRs {
					t.Errorf("Expected %d PRs, got %d", tt.expectedPRs, len(prs))
				}
			}
		})
	}
}

// TestCreatePR тестирует создание Pull Request
func TestCreatePR(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name         string
		head         string
		responseCode int
		responseBody string
		expectError  bool
	}{
		{
			name:         "successful create PR",
			head:         "feature-branch",
			responseCode: 201,
			responseBody: `{"id":123,"number":123,"base":{"label":"test-branch","name":"test-branch"},"head":{"label":"feature-branch","name":"feature-branch"}}`,
			expectError:  false,
		},
		{
			name:         "failed create PR",
			head:         "bad-branch",
			responseCode: 400,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST method, got %s", r.Method)
				}
				w.WriteHeader(tt.responseCode)
				if _, err := w.Write([]byte(tt.responseBody)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			api := &API{
				GiteaURL:   server.URL,
				Owner:      "testowner",
				Repo:       "testrepo",
				BaseBranch: "main",
				NewBranch:  "test-branch",
			}

			pr, err := api.CreatePR(ctx, tt.head)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if pr.Head != tt.head {
					t.Errorf("Expected PR head %s, got %s", tt.head, pr.Head)
				}
			}
		})
	}
}

// TestClosePR тестирует закрытие Pull Request
func TestClosePR(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name         string
		prNumber     int64
		responseCode int
		expectError  bool
	}{
		{
			name:         "successful close PR",
			prNumber:     123,
			responseCode: 201,
			expectError:  false,
		},
		{
			name:         "failed close PR",
			prNumber:     404,
			responseCode: 404,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "PATCH" {
					t.Errorf("Expected PATCH method, got %s", r.Method)
				}
				w.WriteHeader(tt.responseCode)
			}))
			defer server.Close()

			api := &API{
				GiteaURL: server.URL,
				Owner:    "testowner",
				Repo:     "testrepo",
			}

			err := api.ClosePR(ctx, tt.prNumber)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestIsUserInTeam тестирует проверку участия пользователя в команде
func TestIsUserInTeam(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name              string
		username          string
		orgName           string
		teamName          string
		searchResponseCode int
		searchResponseBody string
		memberResponseCode int
		expectError       bool
		expectedResult    bool
	}{
		{
			name:              "user is in team",
			username:          "testuser",
			orgName:           "testorg",
			teamName:          "testteam",
			searchResponseCode: 200,
			searchResponseBody: `{"data":[{"id":123,"name":"testteam"}]}`,
			memberResponseCode: 200,
			expectError:       false,
			expectedResult:    true,
		},
		{
			name:              "user not in team",
			username:          "testuser",
			orgName:           "testorg",
			teamName:          "testteam",
			searchResponseCode: 200,
			searchResponseBody: `{"data":[{"id":123,"name":"testteam"}]}`,
			memberResponseCode: 404,
			expectError:       false,
			expectedResult:    false,
		},
		{
			name:              "team not found",
			username:          "testuser",
			orgName:           "testorg",
			teamName:          "nonexistent",
			searchResponseCode: 200,
			searchResponseBody: `{"data":[]}`,
			memberResponseCode: 0, // не должно дойти до второго запроса
			expectError:       false,
			expectedResult:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("Expected GET method, got %s", r.Method)
				}
				requestCount++

				// Первый запрос - поиск команды
				if requestCount == 1 {
					if !strings.Contains(r.URL.Path, "/teams/search") {
						t.Errorf("First request should be team search, got path: %s", r.URL.Path)
					}
					w.WriteHeader(tt.searchResponseCode)
					if _, err := w.Write([]byte(tt.searchResponseBody)); err != nil {
						t.Errorf("Failed to write search response: %v", err)
					}
					return
				}

				// Второй запрос - проверка членства (если команда найдена)
				if requestCount == 2 {
					if !strings.Contains(r.URL.Path, "/teams/123/members/") {
						t.Errorf("Second request should be membership check, got path: %s", r.URL.Path)
					}
					w.WriteHeader(tt.memberResponseCode)
					return
				}

				t.Errorf("Unexpected request count: %d", requestCount)
			}))
			defer server.Close()

			api := &API{
				GiteaURL:    server.URL,
				AccessToken: "testtoken",
			}

			result, err := api.IsUserInTeam(ctx, logger, tt.username, tt.orgName, tt.teamName)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expectedResult {
					t.Errorf("Expected result %v, got %v", tt.expectedResult, result)
				}
			}
		})
	}
}


// TestGetBranches тестирует получение списка веток
func TestGetBranches(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name         string
		repo         string
		responseCode int
		responseBody string
		expectError  bool
	}{
		{
			name:         "successful get branches",
			repo:         "testrepo",
			responseCode: 200,
			responseBody: `[{"name":"main","commit":{"id":"abc123"}},{"name":"develop","commit":{"id":"def456"}}]`,
			expectError:  false,
		},
		{
			name:         "failed get branches",
			repo:         "nonexistent",
			responseCode: 404,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				AccessToken: "testtoken",
			}

			branches, err := api.GetBranches(ctx, tt.repo)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(branches) != 2 {
					t.Errorf("Expected 2 branches, got %d", len(branches))
				}
			}
		})
	}
}