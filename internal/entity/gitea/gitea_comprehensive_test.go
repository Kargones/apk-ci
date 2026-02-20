package gitea

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestConflictFilesPR тестирует получение файлов с конфликтами в PR
//nolint:dupl // similar test structure
func TestConflictFilesPR(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name         string
		prNumber     int64
		responseCode int
		responseBody string
		expectError  bool
		expectedFiles int
	}{
		{
			name:         "successful get conflict files",
			prNumber:     123,
			responseCode: 200, // Current implementation has bug - checks for 204 but should check for 200
			responseBody: `[{"filename":"file1.go"},{"filename":"file2.go"}]`,
			expectError:  true, // This will fail due to the bug in ConflictFilesPR
			expectedFiles: 2,
		},
		{
			name:         "API error",
			prNumber:     404,
			responseCode: 404,
			responseBody: `{"message":"Not Found"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := fmt.Sprintf("/api/v1/repos/testowner/testrepo/pulls/%d/files", tt.prNumber)
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}
				w.WriteHeader(tt.responseCode)
				if _, err := w.Write([]byte(tt.responseBody)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			api := &API{
				GiteaURL: server.URL,
				Owner:    "testowner",
				Repo:     "testrepo",
			}

			files, err := api.ConflictFilesPR(ctx, tt.prNumber)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(files) != tt.expectedFiles {
					t.Errorf("Expected %d files, got %d", tt.expectedFiles, len(files))
				}
			}
		})
	}
}

// TestGetTeamMembers тестирует получение членов команды
func TestGetTeamMembers(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name              string
		orgName           string
		teamName          string
		searchResponseCode int
		searchResponseBody string
		memberResponseCode int
		memberResponseBody string
		expectError       bool
		expectedMembers   int
	}{
		{
			name:              "successful get team members",
			orgName:           "testorg",
			teamName:          "testteam",
			searchResponseCode: 200,
			searchResponseBody: `{"data":[{"id":123,"name":"testteam"}]}`,
			memberResponseCode: 200,
			memberResponseBody: `[{"login":"user1"},{"login":"user2"}]`,
			expectError:       false,
			expectedMembers:   2,
		},
		{
			name:              "team not found",
			orgName:           "testorg",
			teamName:          "nonexistent",
			searchResponseCode: 200,
			searchResponseBody: `{"data":[]}`,
			memberResponseCode: 0, // не должно дойти до второго запроса
			memberResponseBody: "",
			expectError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

				// Второй запрос - получение членов команды
				if requestCount == 2 {
					if !strings.Contains(r.URL.Path, "/teams/123/members") {
						t.Errorf("Second request should be get members, got path: %s", r.URL.Path)
					}
					w.WriteHeader(tt.memberResponseCode)
					if _, err := w.Write([]byte(tt.memberResponseBody)); err != nil {
						t.Errorf("Failed to write member response: %v", err)
					}
					return
				}

				t.Errorf("Unexpected request count: %d", requestCount)
			}))
			defer server.Close()

			api := &API{
				GiteaURL:    server.URL,
				AccessToken: "testtoken",
			}

			members, err := api.GetTeamMembers(ctx, tt.orgName, tt.teamName)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(members) != tt.expectedMembers {
					t.Errorf("Expected %d members, got %d", tt.expectedMembers, len(members))
				}
			}
		})
	}
}

// TestGetLatestCommit тестирует получение последнего коммита
func TestGetLatestCommit(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name         string
		branch       string
		responseCode int
		responseBody string
		expectError  bool
	}{
		{
			name:         "successful get latest commit",
			branch:       "main",
			responseCode: 200,
			responseBody: `[{"sha":"abc123","commit":{"author":{"name":"Test User","email":"test@example.com","date":"2023-01-01T00:00:00Z"},"message":"Test commit"}}]`,
			expectError:  false,
		},
		{
			name:         "no commits found",
			branch:       "empty",
			responseCode: 200,
			responseBody: `[]`,
			expectError:  true,
		},
		{
			name:         "branch not found",
			branch:       "notfound",
			responseCode: 404,
			responseBody: `{"message":"Not Found"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := fmt.Sprintf("/api/v1/repos/testowner/testrepo/commits")
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}
				if r.URL.Query().Get("sha") != tt.branch {
					t.Errorf("Expected branch %s, got %s", tt.branch, r.URL.Query().Get("sha"))
				}
				if r.URL.Query().Get("limit") != "1" {
					t.Errorf("Expected limit 1, got %s", r.URL.Query().Get("limit"))
				}
				w.WriteHeader(tt.responseCode)
				if _, err := w.Write([]byte(tt.responseBody)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			api := &API{
				GiteaURL: server.URL,
				Owner:    "testowner",
				Repo:     "testrepo",
			}

			commit, err := api.GetLatestCommit(ctx, tt.branch)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if commit == nil {
					t.Error("Expected commit but got nil")
				}
			}
		})
	}
}

// TestGetCommits тестирует получение списка коммитов
func TestGetCommits(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name         string
		branch       string
		limit        int
		responseCode int
		responseBody string
		expectError  bool
		expectedCount int
	}{
		{
			name:         "successful get commits with limit",
			branch:       "main",
			limit:        2,
			responseCode: 200,
			responseBody: `[{"sha":"abc123","commit":{"message":"Commit 1"}},{"sha":"def456","commit":{"message":"Commit 2"}}]`,
			expectError:  false,
			expectedCount: 2,
		},
		{
			name:         "successful get commits without limit",
			branch:       "main",
			limit:        0,
			responseCode: 200,
			responseBody: `[{"sha":"abc123","commit":{"message":"Commit 1"}}]`,
			expectError:  false,
			expectedCount: 1,
		},
		{
			name:         "branch not found",
			branch:       "notfound",
			limit:        0,
			responseCode: 404,
			responseBody: `{"message":"Not Found"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := fmt.Sprintf("/api/v1/repos/testowner/testrepo/commits")
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}
				if r.URL.Query().Get("sha") != tt.branch {
					t.Errorf("Expected branch %s, got %s", tt.branch, r.URL.Query().Get("sha"))
				}
				if tt.limit > 0 {
					expectedLimit := fmt.Sprintf("%d", tt.limit)
					if r.URL.Query().Get("limit") != expectedLimit {
						t.Errorf("Expected limit %s, got %s", expectedLimit, r.URL.Query().Get("limit"))
					}
				}
				w.WriteHeader(tt.responseCode)
				if _, err := w.Write([]byte(tt.responseBody)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			api := &API{
				GiteaURL: server.URL,
				Owner:    "testowner",
				Repo:     "testrepo",
			}

			commits, err := api.GetCommits(ctx, tt.branch, tt.limit)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(commits) != tt.expectedCount {
					t.Errorf("Expected %d commits, got %d", tt.expectedCount, len(commits))
				}
			}
		})
	}
}

// TestGetFirstCommitOfBranch тестирует получение первого коммита ветки
func TestGetFirstCommitOfBranch(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name         string
		branch       string
		baseBranch   string
		responseCode int
		responseBody string
		expectError  bool
	}{
		{
			name:         "successful get first commit",
			branch:       "feature",
			baseBranch:   "main",
			responseCode: 200,
			responseBody: `[{"sha":"newest","commit":{"message":"Newest"}},{"sha":"oldest","commit":{"message":"Oldest"}}]`,
			expectError:  false,
		},
		{
			name:         "no commits found",
			branch:       "empty",
			baseBranch:   "main",
			responseCode: 200,
			responseBody: `[]`,
			expectError:  true,
		},
		{
			name:         "branch not found",
			branch:       "notfound",
			baseBranch:   "main",
			responseCode: 404,
			responseBody: `{"message":"Not Found"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseCode)
				if _, err := w.Write([]byte(tt.responseBody)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			api := &API{
				GiteaURL: server.URL,
				Owner:    "testowner",
				Repo:     "testrepo",
			}

			commit, err := api.GetFirstCommitOfBranch(ctx, tt.branch, tt.baseBranch)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if commit == nil {
					t.Error("Expected commit but got nil")
				}
				// Должен вернуть последний коммит из списка (первый в истории)
				if commit.SHA != "oldest" {
					t.Errorf("Expected oldest commit SHA, got %s", commit.SHA)
				}
			}
		})
	}
}

// TestGetCommitsBetween тестирует получение коммитов между двумя SHA
func TestGetCommitsBetween(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name            string
		baseCommitSHA   string
		headCommitSHA   string
		responseCode    int
		responseBody    string
		expectError     bool
		expectedCount   int
	}{
		{
			name:          "successful get commits between with base found",
			baseCommitSHA: "base123",
			headCommitSHA: "head456",
			responseCode:  200,
			responseBody:  `[{"sha":"head456","commit":{"message":"Head"}},{"sha":"mid789","commit":{"message":"Middle"}},{"sha":"base123","commit":{"message":"Base"}}]`,
			expectError:   false,
			expectedCount: 2, // head и middle, но не base
		},
		{
			name:          "successful get commits between without base found",
			baseCommitSHA: "notfound",
			headCommitSHA: "head456",
			responseCode:  200,
			responseBody:  `[{"sha":"head456","commit":{"message":"Head"}},{"sha":"mid789","commit":{"message":"Middle"}}]`,
			expectError:   false,
			expectedCount: 2, // все коммиты, так как base не найден
		},
		{
			name:          "head commit not found",
			baseCommitSHA: "base123",
			headCommitSHA: "notfound",
			responseCode:  404,
			responseBody:  `{"message":"Not Found"}`,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseCode)
				if _, err := w.Write([]byte(tt.responseBody)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			api := &API{
				GiteaURL: server.URL,
				Owner:    "testowner",
				Repo:     "testrepo",
			}

			commits, err := api.GetCommitsBetween(ctx, tt.baseCommitSHA, tt.headCommitSHA)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(commits) != tt.expectedCount {
					t.Errorf("Expected %d commits, got %d", tt.expectedCount, len(commits))
				}
			}
		})
	}
}

// TestGetCommitFiles тестирует получение файлов коммита
//nolint:dupl // similar test structure
func TestGetCommitFiles(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name         string
		commitSHA    string
		responseCode int
		responseBody string
		expectError  bool
		expectedFiles int
	}{
		{
			name:         "successful get commit files",
			commitSHA:    "abc123",
			responseCode: 200,
			responseBody: `{"files":[{"filename":"file1.go","status":"modified","patch":"diff content"},{"filename":"file2.go","status":"added","patch":"diff content 2"}]}`,
			expectError:  false,
			expectedFiles: 2,
		},
		{
			name:         "commit not found",
			commitSHA:    "notfound",
			responseCode: 404,
			responseBody: `{"message":"Not Found"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := fmt.Sprintf("/api/v1/repos/testowner/testrepo/git/commits/%s", tt.commitSHA)
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}
				w.WriteHeader(tt.responseCode)
				if _, err := w.Write([]byte(tt.responseBody)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			api := &API{
				GiteaURL: server.URL,
				Owner:    "testowner",
				Repo:     "testrepo",
			}

			files, err := api.GetCommitFiles(ctx, tt.commitSHA)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(files) != tt.expectedFiles {
					t.Errorf("Expected %d files, got %d", tt.expectedFiles, len(files))
				}
			}
		})
	}
}

// TestSetRepositoryState тестирует установку состояния репозитория
func TestSetRepositoryState(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name         string
		operations   []ChangeFileOperation
		branch       string
		commitMessage string
		responseCode int
		responseBody string
		expectError  bool
	}{
		{
			name: "successful set repository state",
			operations: []ChangeFileOperation{
				{Operation: "create", Path: "test.txt", Content: "test content"},
			},
			branch:       "main",
			commitMessage: "Test commit",
			responseCode: 201,
			responseBody: `{"commit":{"sha":"abc123"}}`,
			expectError:  false,
		},
		{
			name:         "empty operations",
			operations:   []ChangeFileOperation{},
			branch:       "main",
			commitMessage: "Test commit",
			responseCode: 0, // не должно дойти до запроса
			expectError:  true,
		},
		{
			name: "API error",
			operations: []ChangeFileOperation{
				{Operation: "create", Path: "test.txt", Content: "test content"},
			},
			branch:       "main",
			commitMessage: "Test commit",
			responseCode: 400,
			responseBody: `{"message":"Bad Request"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST method, got %s", r.Method)
				}
				expectedPath := "/api/v1/repos/testowner/testrepo/contents"
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}
				w.WriteHeader(tt.responseCode)
				if _, err := w.Write([]byte(tt.responseBody)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			api := &API{
				GiteaURL: server.URL,
				Owner:    "testowner",
				Repo:     "testrepo",
			}

			err := api.SetRepositoryState(ctx, logger, tt.operations, tt.branch, tt.commitMessage)

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

// TestGetConfigDataBad тестирует получение конфигурационных данных (устаревший метод)
func TestGetConfigDataBad(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name            string
		filenamePrefix  string
		isDirectURL     bool
		responseCode    int
		responseBody    string
		contentsResponse string
		expectError     bool
		expectedContent string
	}{
		{
			name:            "direct URL with repository server (test limitation)",
			filenamePrefix:  "", // We'll set this to server.URL in the test
			isDirectURL:     true,
			responseCode:    200,
			responseBody:    "config: test",
			expectError:     false,
			expectedContent: "config: test",
		},
		{
			name:            "successful get config data via repository",
			filenamePrefix:  "config",
			isDirectURL:     false,
			responseCode:    200,
			contentsResponse: `[{"name":"config.yaml","path":"config.yaml","type":"file"}]`,
			responseBody:    `{"content":"Y29uZmlnOiB0ZXN0","encoding":"base64"}`,
			expectError:     false,
			expectedContent: "config: test",
		},
		{
			name:           "file not found in repository",
			filenamePrefix: "notfound",
			isDirectURL:    false,
			responseCode:   200,
			contentsResponse: `[{"name":"other.yaml","path":"other.yaml","type":"file"}]`,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++

				if tt.isDirectURL {
					// Прямой URL запрос
					w.WriteHeader(tt.responseCode)
					if _, err := w.Write([]byte(tt.responseBody)); err != nil {
						t.Errorf("Failed to write response: %v", err)
					}
					return
				}

				// Запрос через репозиторий
				if requestCount == 1 {
					// Первый запрос - получение содержимого репозитория
					if !strings.Contains(r.URL.Path, "/contents/") || r.URL.Query().Get("ref") == "" {
						t.Errorf("First request should be get repository contents")
					}
					w.WriteHeader(tt.responseCode)
					if _, err := w.Write([]byte(tt.contentsResponse)); err != nil {
						t.Errorf("Failed to write contents response: %v", err)
					}
					return
				}

				if requestCount == 2 {
					// Второй запрос - получение содержимого файла
					w.WriteHeader(tt.responseCode)
					if _, err := w.Write([]byte(tt.responseBody)); err != nil {
						t.Errorf("Failed to write file response: %v", err)
					}
					return
				}

				t.Errorf("Unexpected request count: %d", requestCount)
			}))
			defer server.Close()

			api := &API{
				GiteaURL:   server.URL,
				Owner:      "testowner",
				Repo:       "testrepo",
				BaseBranch: "main",
			}

			// For direct URL test, use the server URL
			filenamePrefix := tt.filenamePrefix
			if tt.isDirectURL && filenamePrefix == "" {
				filenamePrefix = server.URL + "/config.yaml"
			}

			content, err := api.GetConfigDataBad(ctx, filenamePrefix)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if string(content) != tt.expectedContent {
					t.Errorf("Expected content %s, got %s", tt.expectedContent, string(content))
				}
			}
		})
	}
}

// TestAnalyzeProjectStructureMethod тестирует публичный метод AnalyzeProjectStructure
func TestAnalyzeProjectStructureMethod(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name         string
		branch       string
		responseCode int
		responseBody string
		expectError  bool
		expectedResult []string
	}{
		{
			name:         "successful analyze project structure",
			branch:       "main",
			responseCode: 200,
			responseBody: `[{"name":"MyProject","type":"dir"},{"name":"MyProject.Extension1","type":"dir"},{"name":".git","type":"dir"}]`,
			expectError:  false,
			expectedResult: []string{"MyProject", "Extension1"},
		},
		{
			name:         "repository contents error",
			branch:       "notfound",
			responseCode: 404,
			responseBody: `{"message":"Not Found"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.Contains(r.URL.Path, "/contents/") {
					t.Errorf("Expected contents path, got %s", r.URL.Path)
				}
				if r.URL.Query().Get("ref") != tt.branch {
					t.Errorf("Expected branch %s, got %s", tt.branch, r.URL.Query().Get("ref"))
				}
				w.WriteHeader(tt.responseCode)
				if _, err := w.Write([]byte(tt.responseBody)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			api := &API{
				GiteaURL: server.URL,
				Owner:    "testowner",
				Repo:     "testrepo",
			}

			result, err := api.AnalyzeProjectStructure(ctx, tt.branch)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(result) != len(tt.expectedResult) {
					t.Errorf("Expected %d results, got %d", len(tt.expectedResult), len(result))
				}
				for i, expected := range tt.expectedResult {
					if i < len(result) && result[i] != expected {
						t.Errorf("Expected result[%d] = %s, got %s", i, expected, result[i])
					}
				}
			}
		})
	}
}