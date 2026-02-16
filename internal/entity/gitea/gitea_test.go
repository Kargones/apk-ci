package gitea

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestNewGiteaAPI тестирует создание нового экземпляра API
func TestNewGiteaAPI(t *testing.T) {
	config := Config{
		GiteaURL:    "https://gitea.example.com",
		Owner:       "testowner",
		Repo:        "testrepo",
		AccessToken: "testtoken",
		BaseBranch:  "main",
		NewBranch:   "feature",
		Command:     "test",
	}

	api := NewGiteaAPI(config)

	if api == nil {
		t.Fatal("NewGiteaAPI returned nil")
	}

	if api.GiteaURL != config.GiteaURL {
		t.Errorf("Expected GiteaURL %s, got %s", config.GiteaURL, api.GiteaURL)
	}

	if api.Owner != config.Owner {
		t.Errorf("Expected Owner %s, got %s", config.Owner, api.Owner)
	}

	if api.Repo != config.Repo {
		t.Errorf("Expected Repo %s, got %s", config.Repo, api.Repo)
	}

	if api.AccessToken != config.AccessToken {
		t.Errorf("Expected AccessToken %s, got %s", config.AccessToken, api.AccessToken)
	}

	if api.BaseBranch != config.BaseBranch {
		t.Errorf("Expected BaseBranch %s, got %s", config.BaseBranch, api.BaseBranch)
	}

	if api.NewBranch != config.NewBranch {
		t.Errorf("Expected NewBranch %s, got %s", config.NewBranch, api.NewBranch)
	}

	if api.Command != config.Command {
		t.Errorf("Expected Command %s, got %s", config.Command, api.Command)
	}
}

// TestGetIssue тестирует получение issue
func TestGetIssue(t *testing.T) {
	tests := []struct {
		name         string
		issueNumber  int64
		responseCode int
		responseBody string
		expectError  bool
	}{
		{
			name:         "successful get issue",
			issueNumber:  123,
			responseCode: 200,
			responseBody: `{"id":123,"number":123,"title":"Test Issue","body":"Test body","state":"open","user":{"login":"testuser"},"created_at":"2023-01-01T00:00:00Z","updated_at":"2023-01-01T00:00:00Z"}`,
			expectError:  false,
		},
		{
			name:         "issue not found",
			issueNumber:  404,
			responseCode: 404,
			responseBody: `{"message":"Not Found"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := fmt.Sprintf("/api/v1/repos/testowner/testrepo/issues/%d", tt.issueNumber)
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
				GiteaURL:    server.URL,
				Owner:       "testowner",
				Repo:        "testrepo",
				AccessToken: "testtoken",
			}

			issue, err := api.GetIssue(tt.issueNumber)

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

			if issue == nil {
				t.Error("Expected issue but got nil")
				return
			}

			if issue.ID != tt.issueNumber {
				t.Errorf("Expected issue ID %d, got %d", tt.issueNumber, issue.ID)
			}
		})
	}
}

// TestGetRepositoryContents тестирует получение содержимого репозитория
func TestGetRepositoryContents(t *testing.T) {
	tests := []struct {
		name          string
		filepath      string
		branch        string
		responseCode  int
		responseBody  string
		expectError   bool
		expectedCount int
	}{
		{
			name:          "successful get repository contents",
			filepath:      "src",
			branch:        "main",
			responseCode:  200,
			responseBody:  `[{"name":"file1.go","path":"src/file1.go","type":"file"},{"name":"file2.go","path":"src/file2.go","type":"file"}]`,
			expectError:   false,
			expectedCount: 2,
		},
		{
			name:         "directory not found",
			filepath:     "notfound",
			branch:       "main",
			responseCode: 404,
			responseBody: `{"message":"Not Found"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := fmt.Sprintf("/api/v1/repos/testowner/testrepo/contents/%s", tt.filepath)
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
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
				GiteaURL:    server.URL,
				Owner:       "testowner",
				Repo:        "testrepo",
				AccessToken: "testtoken",
			}

			contents, err := api.GetRepositoryContents(tt.filepath, tt.branch)

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

			if len(contents) != tt.expectedCount {
				t.Errorf("Expected %d files, got %d", tt.expectedCount, len(contents))
			}
		})
	}
}

// TestAnalyzeProject тестирует анализ проекта
func TestAnalyzeProject(t *testing.T) {
	tests := []struct {
		name          string
		branch        string
		responseCode  int
		responseBody  string
		expectError   bool
		expectedCount int
	}{
		{
			name:          "successful analyze project",
			branch:        "main",
			responseCode:  200,
			responseBody:  `[{"name":"MyProject","type":"dir"},{"name":"MyProject.Extension1","type":"dir"}]`,
			expectError:   false,
			expectedCount: 2,
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
				if strings.Contains(r.URL.Path, "/contents") {
					w.WriteHeader(tt.responseCode)
					if _, err := w.Write([]byte(tt.responseBody)); err != nil {
						t.Errorf("Failed to write response: %v", err)
					}
				} else {
					w.WriteHeader(404)
				}
			}))
			defer server.Close()

			api := &API{
				GiteaURL:    server.URL,
				Owner:       "testowner",
				Repo:        "testrepo",
				AccessToken: "testtoken",
			}

			result, err := api.AnalyzeProject(tt.branch)

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

			if len(result) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(result))
			}
		})
	}
}

// TestConflictPR тестирует проверку конфликтов в PR
func TestConflictPR(t *testing.T) {
	tests := []struct {
		name             string
		prNumber         int64
		responseCode     int
		responseBody     string
		expectError      bool
		expectedConflict bool
	}{
		{
			name:             "PR with conflicts",
			prNumber:         123,
			responseCode:     200,
			responseBody:     `{"mergeable":false}`,
			expectError:      false,
			expectedConflict: true,
		},
		{
			name:             "PR without conflicts",
			prNumber:         124,
			responseCode:     200,
			responseBody:     `{"mergeable":true}`,
			expectError:      false,
			expectedConflict: false,
		},
		{
			name:         "PR not found",
			prNumber:     404,
			responseCode: 404,
			responseBody: `{"message":"Not Found"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := fmt.Sprintf("/api/v1/repos/testowner/testrepo/pulls/%d", tt.prNumber)
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
				GiteaURL:    server.URL,
				Owner:       "testowner",
				Repo:        "testrepo",
				AccessToken: "testtoken",
			}

			hasConflict, err := api.ConflictPR(tt.prNumber)

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

			if hasConflict != tt.expectedConflict {
				t.Errorf("Expected conflict %v, got %v", tt.expectedConflict, hasConflict)
			}
		})
	}
}

// TestConflictPRWithAsyncCheck тестирует поведение ConflictPR при асинхронной проверке конфликтов (статус "checking")
func TestConflictPRWithAsyncCheck(t *testing.T) {
	tests := []struct {
		name                string
		prNumber            int64
		initialResponseBody string
		finalResponseBody   string
		expectedConflict    bool
		expectError         bool
	}{
		{
			name:                "Async check transitions to success",
			prNumber:            125,
			initialResponseBody: `{"number":125,"mergeable":false,"mergeable_state":"checking"}`,
			finalResponseBody:   `{"number":125,"mergeable":true,"mergeable_state":"success"}`,
			expectedConflict:    false,
			expectError:         false,
		},
		{
			name:                "Async check transitions to conflict",
			prNumber:            126,
			initialResponseBody: `{"number":126,"mergeable":false,"mergeable_state":"checking"}`,
			finalResponseBody:   `{"number":126,"mergeable":false,"mergeable_state":"conflict"}`,
			expectedConflict:    true,
			expectError:         false,
		},
		{
			name:                "Mergeable_state is behind",
			prNumber:            127,
			initialResponseBody: `{"number":127,"mergeable":false,"mergeable_state":"behind"}`,
			finalResponseBody:   `{"number":127,"mergeable":false,"mergeable_state":"behind"}`,
			expectedConflict:    true,
			expectError:         false,
		},
		{
			name:                "Mergeable_state is blocked",
			prNumber:            128,
			initialResponseBody: `{"number":128,"mergeable":false,"mergeable_state":"blocked"}`,
			finalResponseBody:   `{"number":128,"mergeable":false,"mergeable_state":"blocked"}`,
			expectedConflict:    true,
			expectError:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := fmt.Sprintf("/api/v1/repos/testowner/testrepo/pulls/%d", tt.prNumber)
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}
				w.WriteHeader(200)

				// При первом вызове возвращаем начальное состояние, при последующих - финальное
				callCount++
				if callCount == 1 {
					if _, err := w.Write([]byte(tt.initialResponseBody)); err != nil {
						t.Errorf("Failed to write response: %v", err)
					}
				} else {
					if _, err := w.Write([]byte(tt.finalResponseBody)); err != nil {
						t.Errorf("Failed to write response: %v", err)
					}
				}
			}))
			defer server.Close()

			api := &API{
				GiteaURL:    server.URL,
				Owner:       "testowner",
				Repo:        "testrepo",
				AccessToken: "testtoken",
			}

			hasConflict, err := api.ConflictPR(tt.prNumber)

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

			if hasConflict != tt.expectedConflict {
				t.Errorf("Expected conflict %v, got %v", tt.expectedConflict, hasConflict)
			}

			// Проверяем, что функция действительно делала повторные запросы при статусе "checking"
			if tt.initialResponseBody != tt.finalResponseBody {
				if callCount < 2 {
					t.Errorf("Expected at least 2 API calls for async check, got %d", callCount)
				}
			}
		})
	}
}

// TestAnalyzeProjectStructure тестирует функцию analyzeProjectStructure
func TestAnalyzeProjectStructure(t *testing.T) {
	tests := []struct {
		name        string
		directories []string
		expected    []string
		expectError bool
	}{
		{
			name:        "empty directories",
			directories: []string{},
			expected:    []string{},
			expectError: false,
		},
		{
			name:        "single project without extensions",
			directories: []string{"MyProject"},
			expected:    []string{"MyProject"},
			expectError: false,
		},
		{
			name:        "project with extensions",
			directories: []string{"MyProject", "MyProject.Extension1", "MyProject.Extension2"},
			expected:    []string{"MyProject", "Extension1", "Extension2"},
			expectError: false,
		},
		{
			name:        "only extensions without main project",
			directories: []string{"Project.Extension1", "Project.Extension2"},
			expected:    []string{},
			expectError: false,
		},
		{
			name:        "mixed directories with project and extensions",
			directories: []string{"MyProject", "MyProject.Ext1", "OtherDir.Something", "MyProject.Ext2"},
			expected:    []string{"MyProject", "Ext1", "Ext2"},
			expectError: false,
		},
		{
			name:        "project with partial extension match",
			directories: []string{"MyProject", "MyProjectExtension", "MyProject.RealExt"},
			expected:    []string{"MyProject", "RealExt"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzeProjectStructure(tt.directories)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d elements, got %d. Expected: %v, Got: %v", len(tt.expected), len(result), tt.expected, result)
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Expected element %d to be %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}

// TestUpdateConfig тестирует обновление конфигурации
func TestUpdateConfig(t *testing.T) {
	initialConfig := Config{
		GiteaURL:    "https://gitea.example.com",
		Owner:       "testowner",
		Repo:        "testrepo",
		AccessToken: "testtoken",
		BaseBranch:  "main",
		NewBranch:   "feature",
		Command:     "test",
	}

	api := NewGiteaAPI(initialConfig)

	newConfig := Config{
		GiteaURL:    "https://newgitea.example.com",
		Owner:       "newowner",
		Repo:        "newrepo",
		AccessToken: "newtoken",
		BaseBranch:  "develop",
		NewBranch:   "newfeature",
		Command:     "newtest",
	}

	api.UpdateConfig(newConfig)

	if api.GiteaURL != newConfig.GiteaURL {
		t.Errorf("Expected GiteaURL %s, got %s", newConfig.GiteaURL, api.GiteaURL)
	}

	if api.Owner != newConfig.Owner {
		t.Errorf("Expected Owner %s, got %s", newConfig.Owner, api.Owner)
	}

	if api.Repo != newConfig.Repo {
		t.Errorf("Expected Repo %s, got %s", newConfig.Repo, api.Repo)
	}

	if api.AccessToken != newConfig.AccessToken {
		t.Errorf("Expected AccessToken %s, got %s", newConfig.AccessToken, api.AccessToken)
	}

	if api.BaseBranch != newConfig.BaseBranch {
		t.Errorf("Expected BaseBranch %s, got %s", newConfig.BaseBranch, api.BaseBranch)
	}

	if api.NewBranch != newConfig.NewBranch {
		t.Errorf("Expected NewBranch %s, got %s", newConfig.NewBranch, api.NewBranch)
	}

	if api.Command != newConfig.Command {
		t.Errorf("Expected Command %s, got %s", newConfig.Command, api.Command)
	}
}

// TestGetFileContent тестирует получение содержимого файла
func TestGetFileContent(t *testing.T) {
	tests := []struct {
		name            string
		fileName        string
		responseCode    int
		responseBody    string
		expectError     bool
		expectedContent string
	}{
		{
			name:            "successful get file content",
			fileName:        "test.txt",
			responseCode:    200,
			responseBody:    `{"content":"dGVzdCBjb250ZW50","encoding":"base64"}`,
			expectError:     false,
			expectedContent: "test content",
		},
		{
			name:         "file not found",
			fileName:     "notfound.txt",
			responseCode: 404,
			responseBody: `{"message":"Not Found"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := fmt.Sprintf("/api/v1/repos/testowner/testrepo/contents/%s", tt.fileName)
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
				GiteaURL:    server.URL,
				Owner:       "testowner",
				Repo:        "testrepo",
				AccessToken: "testtoken",
				BaseBranch:  "main",
			}

			content, err := api.GetFileContent(tt.fileName)

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

			if string(content) != tt.expectedContent {
				t.Errorf("Expected content %s, got %s", tt.expectedContent, string(content))
			}
		})
	}
}

// TestAddIssueComment тестирует добавление комментария к issue
func TestAddIssueComment(t *testing.T) {
	tests := []struct {
		name         string
		issueNumber  int64
		commentText  string
		responseCode int
		expectError  bool
	}{
		{
			name:         "successful add comment",
			issueNumber:  123,
			commentText:  "Test comment",
			responseCode: 201,
			expectError:  false,
		},
		{
			name:         "issue not found",
			issueNumber:  404,
			commentText:  "Test comment",
			responseCode: 404,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := fmt.Sprintf("/api/v1/repos/testowner/testrepo/issues/%d/comments", tt.issueNumber)
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}
				if r.Method != "POST" {
					t.Errorf("Expected POST method, got %s", r.Method)
				}

				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("Failed to read request body: %v", err)
					return
				}
				var comment map[string]string
				if err := json.Unmarshal(body, &comment); err != nil {
					t.Errorf("Failed to unmarshal JSON: %v", err)
					return
				}
				if comment["body"] != tt.commentText {
					t.Errorf("Expected comment body %s, got %s", tt.commentText, comment["body"])
				}

				w.WriteHeader(tt.responseCode)
			}))
			defer server.Close()

			api := &API{
				GiteaURL:    server.URL,
				Owner:       "testowner",
				Repo:        "testrepo",
				AccessToken: "testtoken",
			}

			err := api.AddIssueComment(tt.issueNumber, tt.commentText)

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

// TestCloseIssue тестирует закрытие issue
func TestCloseIssue(t *testing.T) {
	tests := []struct {
		name         string
		issueNumber  int64
		responseCode int
		expectError  bool
	}{
		{
			name:         "successful close issue",
			issueNumber:  123,
			responseCode: 201,
			expectError:  false,
		},
		{
			name:         "issue not found",
			issueNumber:  404,
			responseCode: 404,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := fmt.Sprintf("/api/v1/repos/testowner/testrepo/issues/%d", tt.issueNumber)
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}
				if r.Method != "PATCH" {
					t.Errorf("Expected PATCH method, got %s", r.Method)
				}

				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("Failed to read request body: %v", err)
					return
				}
				var issue map[string]string
				if err := json.Unmarshal(body, &issue); err != nil {
					t.Errorf("Failed to unmarshal JSON: %v", err)
					return
				}
				if issue["state"] != "closed" {
					t.Errorf("Expected state closed, got %s", issue["state"])
				}

				w.WriteHeader(tt.responseCode)
			}))
			defer server.Close()

			api := &API{
				GiteaURL:    server.URL,
				Owner:       "testowner",
				Repo:        "testrepo",
				AccessToken: "testtoken",
			}

			err := api.CloseIssue(tt.issueNumber)

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

// TestCreatePRWithOptions_Success тестирует успешное создание PR
func TestCreatePRWithOptions_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/repos/testowner/testrepo/pulls" && r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{
				"id": 123,
				"number": 45,
				"html_url": "https://gitea.example.com/testowner/testrepo/pulls/45",
				"state": "open",
				"title": "Test PR",
				"body": "Test body",
				"mergeable": true
			}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	api := &API{
		GiteaURL:    server.URL,
		Owner:       "testowner",
		Repo:        "testrepo",
		AccessToken: "testtoken",
	}

	opts := CreatePROptions{
		Title: "Test PR",
		Body:  "Test body",
		Head:  "feature-branch",
		Base:  "main",
	}

	pr, err := api.CreatePRWithOptions(opts)
	if err != nil {
		t.Fatalf("CreatePRWithOptions вернул ошибку: %v", err)
	}

	if pr.Number != 45 {
		t.Errorf("PR.Number = %d, want %d", pr.Number, 45)
	}
	if pr.HTMLURL != "https://gitea.example.com/testowner/testrepo/pulls/45" {
		t.Errorf("PR.HTMLURL = %q, неверный URL", pr.HTMLURL)
	}
	if pr.State != "open" {
		t.Errorf("PR.State = %q, want %q", pr.State, "open")
	}
}

// TestCreatePRWithOptions_Conflict тестирует обработку существующего PR
func TestCreatePRWithOptions_Conflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/repos/testowner/testrepo/pulls" && r.Method == "POST":
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"message": "pull request already exists"}`))

		case r.URL.Path == "/api/v1/repos/testowner/testrepo/pulls" && r.Method == "GET":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{
				"id": 100,
				"number": 10,
				"html_url": "https://gitea.example.com/testowner/testrepo/pulls/10",
				"state": "open",
				"title": "Existing PR",
				"body": "...",
				"head": {"ref": "feature-branch"},
				"base": {"ref": "main"}
			}]`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	api := &API{
		GiteaURL:    server.URL,
		Owner:       "testowner",
		Repo:        "testrepo",
		AccessToken: "testtoken",
	}

	opts := CreatePROptions{
		Title: "Test PR",
		Body:  "Test body",
		Head:  "feature-branch",
		Base:  "main",
	}

	pr, err := api.CreatePRWithOptions(opts)
	if err != nil {
		t.Fatalf("CreatePRWithOptions вернул ошибку: %v", err)
	}

	// Должен вернуть существующий PR
	if pr.Number != 10 {
		t.Errorf("PR.Number = %d, want %d (существующий PR)", pr.Number, 10)
	}
}

// TestCreatePRWithOptions_BranchNotFound тестирует ошибку при несуществующей ветке
func TestCreatePRWithOptions_BranchNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "branch not found"}`))
	}))
	defer server.Close()

	api := &API{
		GiteaURL:    server.URL,
		Owner:       "testowner",
		Repo:        "testrepo",
		AccessToken: "testtoken",
	}

	opts := CreatePROptions{
		Title: "Test PR",
		Body:  "Test body",
		Head:  "non-existent-branch",
		Base:  "main",
	}

	_, err := api.CreatePRWithOptions(opts)
	if err == nil {
		t.Fatal("Ожидалась ошибка при несуществующей ветке")
	}
	if !strings.Contains(err.Error(), "ветка не существует") {
		t.Errorf("Ошибка должна содержать информацию о несуществующей ветке: %v", err)
	}
}

// TestCreatePRWithOptions_ServerError тестирует обработку ошибки сервера
func TestCreatePRWithOptions_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message": "internal server error"}`))
	}))
	defer server.Close()

	api := &API{
		GiteaURL:    server.URL,
		Owner:       "testowner",
		Repo:        "testrepo",
		AccessToken: "testtoken",
	}

	opts := CreatePROptions{
		Title: "Test PR",
		Body:  "Test body",
		Head:  "feature-branch",
		Base:  "main",
	}

	_, err := api.CreatePRWithOptions(opts)
	if err == nil {
		t.Fatal("Ожидалась ошибка при ошибке сервера")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("Ошибка должна содержать код статуса 500: %v", err)
	}
}

// TestCreatePRWithOptions_InvalidJSON тестирует обработку некорректного JSON ответа
func TestCreatePRWithOptions_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`invalid json response`))
	}))
	defer server.Close()

	api := &API{
		GiteaURL:    server.URL,
		Owner:       "testowner",
		Repo:        "testrepo",
		AccessToken: "testtoken",
	}

	opts := CreatePROptions{
		Title: "Test PR",
		Body:  "Test body",
		Head:  "feature-branch",
		Base:  "main",
	}

	_, err := api.CreatePRWithOptions(opts)
	if err == nil {
		t.Fatal("Ожидалась ошибка при некорректном JSON")
	}
	if !strings.Contains(err.Error(), "разбор") {
		t.Errorf("Ошибка должна содержать информацию о проблеме разбора: %v", err)
	}
}

// TestCreatePRWithOptions_ConflictFindError тестирует ошибку поиска существующего PR
func TestCreatePRWithOptions_ConflictFindError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/repos/testowner/testrepo/pulls" && r.Method == "POST":
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"message": "pull request already exists"}`))

		case r.URL.Path == "/api/v1/repos/testowner/testrepo/pulls" && r.Method == "GET":
			// Ошибка при получении списка PR
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message": "server error"}`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	api := &API{
		GiteaURL:    server.URL,
		Owner:       "testowner",
		Repo:        "testrepo",
		AccessToken: "testtoken",
	}

	opts := CreatePROptions{
		Title: "Test PR",
		Body:  "Test body",
		Head:  "feature-branch",
		Base:  "main",
	}

	_, err := api.CreatePRWithOptions(opts)
	if err == nil {
		t.Fatal("Ожидалась ошибка при невозможности найти существующий PR")
	}
	if !strings.Contains(err.Error(), "уже существует") {
		t.Errorf("Ошибка должна указывать что PR уже существует: %v", err)
	}
}

// TestGetConfigData тестирует получение конфигурационных данных
func TestGetConfigData(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name            string
		filename        string
		responseCode    int
		responseBody    string
		expectError     bool
		expectedContent string
	}{
		{
			name:            "successful get config data",
			filename:        "config.yaml",
			responseCode:    200,
			responseBody:    `{"content":"Y29uZmlnOiB0ZXN0","encoding":"base64"}`,
			expectError:     false,
			expectedContent: "config: test",
		},
		{
			name:         "config file not found",
			filename:     "notfound.yaml",
			responseCode: 404,
			responseBody: `{"message":"Not Found"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, tt.filename) {
					w.WriteHeader(tt.responseCode)
					if _, err := w.Write([]byte(tt.responseBody)); err != nil {
						t.Errorf("Failed to write response: %v", err)
					}
				} else {
					w.WriteHeader(404)
				}
			}))
			defer server.Close()

			api := &API{
				GiteaURL:    server.URL,
				Owner:       "testowner",
				Repo:        "testrepo",
				AccessToken: "testtoken",
				BaseBranch:  "main",
			}

			content, err := api.GetConfigData(logger, tt.filename)

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

			if string(content) != tt.expectedContent {
				t.Errorf("Expected content %s, got %s", tt.expectedContent, string(content))
			}
		})
	}
}
