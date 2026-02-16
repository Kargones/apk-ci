package gitea

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestGetBranchCommitRange тестирует получение диапазона коммитов ветки
func TestGetBranchCommitRange(t *testing.T) {
	tests := []struct {
		name         string
		branch       string
		isMainBranch bool
		expectError  bool
	}{
		{
			name:         "main branch commit range",
			branch:       "main",
			isMainBranch: true,
			expectError:  false,
		},
		{
			name:         "master branch commit range",
			branch:       "master",
			isMainBranch: true,
			expectError:  false,
		},
		{
			name:         "feature branch commit range",
			branch:       "feature-branch",
			isMainBranch: false,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++

				if tt.isMainBranch {
					// Для главной ветки
					if requestCount == 1 {
						// Запрос последнего коммита
						if strings.Contains(r.URL.Path, "/commits") && r.URL.Query().Get("limit") == "1" {
							w.WriteHeader(http.StatusOK)
							w.Write([]byte(`[{"sha":"latest123","commit":{"message":"Latest commit"}}]`))
							return
						}
					}
					if requestCount == 2 {
						// Запрос тегов для поиска sq-start
						if strings.Contains(r.URL.Path, "/tags") {
							w.WriteHeader(http.StatusOK)
							w.Write([]byte(`[{"name":"sq-start","commit":{"sha":"start123"}},{"name":"v1.0","commit":{"sha":"tag123"}}]`))
							return
						}
					}
					if requestCount == 3 {
						// Запрос коммита по SHA для sq-start тега
						if strings.Contains(r.URL.Path, "/commits") && r.URL.Query().Get("sha") == "start123" {
							w.WriteHeader(http.StatusOK)
							w.Write([]byte(`[{"sha":"start123","commit":{"message":"Start commit"}}]`))
							return
						}
					}
				} else {
					// Для feature ветки
					if requestCount == 1 {
						// Запрос последнего коммита
						if strings.Contains(r.URL.Path, "/commits") && r.URL.Query().Get("limit") == "1" {
							w.WriteHeader(http.StatusOK)
							w.Write([]byte(`[{"sha":"feature123","commit":{"message":"Feature commit"}}]`))
							return
						}
					}
					if requestCount == 2 {
						// Запрос сравнения веток
						if strings.Contains(r.URL.Path, "/compare/") {
							w.WriteHeader(http.StatusOK)
							w.Write([]byte(`{"merge_base_commit":{"sha":"base123","commit":{"message":"Base commit"}},"commits":[]}`))
							return
						}
					}
				}

				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"message":"Not Found"}`))
			}))
			defer server.Close()

			api := &API{
				GiteaURL:   server.URL,
				Owner:      "testowner",
				Repo:       "testrepo",
				BaseBranch: "main",
			}

			result, err := api.GetBranchCommitRange(tt.branch)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil {
					t.Error("Expected result but got nil")
				}
				if result != nil {
					if result.FirstCommit == nil {
						t.Error("Expected FirstCommit but got nil")
					}
					if result.LastCommit == nil {
						t.Error("Expected LastCommit but got nil")
					}
				}
			}
		})
	}
}

// TestGetMainBranchCommitRange тестирует получение диапазона коммитов главной ветки
func TestGetMainBranchCommitRange(t *testing.T) {
	tests := []struct {
		name               string
		branch             string
		hasTaggedCommit    bool
		latestCommitError  bool
		taggedCommitError  bool
		expectError        bool
	}{
		{
			name:            "successful with tagged commit",
			branch:          "main",
			hasTaggedCommit: true,
			expectError:     false,
		},
		{
			name:               "successful without tagged commit",
			branch:             "main",
			hasTaggedCommit:    false,
			taggedCommitError:  true, // Симулируем ошибку поиска тега
			expectError:        false,
		},
		{
			name:              "latest commit error",
			branch:            "main",
			latestCommitError: true,
			expectError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++

				if requestCount == 1 {
					// Запрос последнего коммита
					if tt.latestCommitError {
						w.WriteHeader(http.StatusNotFound)
						w.Write([]byte(`{"message":"Not Found"}`))
						return
					}
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`[{"sha":"latest123","commit":{"message":"Latest commit"}}]`))
					return
				}

				if requestCount == 2 {
					// Запрос тегов
					if tt.taggedCommitError {
						w.WriteHeader(http.StatusNotFound)
						w.Write([]byte(`{"message":"Not Found"}`))
						return
					}
					if tt.hasTaggedCommit {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`[{"name":"sq-start","commit":{"sha":"start123"}}]`))
						return
					}
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`[{"name":"v1.0","commit":{"sha":"tag123"}}]`))
					return
				}

				if requestCount == 3 {
					if tt.hasTaggedCommit {
						// Запрос коммита по SHA для sq-start тега
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`[{"sha":"start123","commit":{"message":"Start commit"}}]`))
						return
					} else {
						// Запрос первого коммита в истории
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`[{"sha":"oldest123","commit":{"message":"Oldest commit"}}]`))
						return
					}
				}

				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"message":"Not Found"}`))
			}))
			defer server.Close()

			api := &API{
				GiteaURL: server.URL,
				Owner:    "testowner",
				Repo:     "testrepo",
			}

			result, err := api.getMainBranchCommitRange(tt.branch)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil {
					t.Error("Expected result but got nil")
				}
			}
		})
	}
}

// TestGetFeatureBranchCommitRange тестирует получение диапазона коммитов feature ветки
func TestGetFeatureBranchCommitRange(t *testing.T) {
	tests := []struct {
		name              string
		branch            string
		latestCommitError bool
		compareError      bool
		hasMergeBase      bool
		expectError       bool
	}{
		{
			name:         "successful with merge base",
			branch:       "feature",
			hasMergeBase: true,
			expectError:  false,
		},
		{
			name:         "successful without merge base",
			branch:       "feature",
			hasMergeBase: false,
			expectError:  false,
		},
		{
			name:              "latest commit error",
			branch:            "feature",
			latestCommitError: true,
			expectError:       true,
		},
		{
			name:         "compare error",
			branch:       "feature",
			compareError: true,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++

				if requestCount == 1 {
					// Запрос последнего коммита
					if tt.latestCommitError {
						w.WriteHeader(http.StatusNotFound)
						w.Write([]byte(`{"message":"Not Found"}`))
						return
					}
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`[{"sha":"feature123","commit":{"message":"Feature commit"}}]`))
					return
				}

				if requestCount == 2 {
					// Запрос сравнения веток
					if tt.compareError {
						w.WriteHeader(http.StatusNotFound)
						w.Write([]byte(`{"message":"Not Found"}`))
						return
					}
					if tt.hasMergeBase {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`{"merge_base_commit":{"sha":"base123","commit":{"message":"Base commit"}},"commits":[]}`))
						return
					} else {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`{"commits":[]}`))
						return
					}
				}

				if requestCount >= 3 && !tt.hasMergeBase {
					// Запросы для findMergeBase (head commits, then base commits)
					if requestCount == 3 {
						// Head commits request for findMergeBase
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`[{"sha":"head123","commit":{"message":"Head commit"}}]`))
						return
					}
					if requestCount == 4 {
						// Base commits request for findMergeBase - put head123 in the middle, not at the end
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`[{"sha":"latest123","commit":{"message":"Latest commit"}},{"sha":"head123","commit":{"message":"Head commit"}},{"sha":"older123","commit":{"message":"Older commit"}}]`))
						return
					}
				}

				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"message":"Not Found"}`))
			}))
			defer server.Close()

			api := &API{
				GiteaURL:   server.URL,
				Owner:      "testowner",
				Repo:       "testrepo",
				BaseBranch: "main",
			}

			result, err := api.getFeatureBranchCommitRange(tt.branch)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil {
					t.Error("Expected result but got nil")
				}
			}
		})
	}
}

// TestFindCommitWithTag тестирует поиск коммита с тегом
func TestFindCommitWithTag(t *testing.T) {
	tests := []struct {
		name         string
		branch       string
		tag          string
		responseCode int
		responseBody string
		expectError  bool
	}{
		{
			name:         "successful find commit with tag",
			branch:       "main",
			tag:          "sq-start",
			responseCode: 200,
			responseBody: `[{"name":"sq-start","commit":{"sha":"start123"}},{"name":"v1.0","commit":{"sha":"tag123"}}]`,
			expectError:  false,
		},
		{
			name:         "tag not found",
			branch:       "main",
			tag:          "nonexistent",
			responseCode: 200,
			responseBody: `[{"name":"v1.0","commit":{"sha":"tag123"}}]`,
			expectError:  true,
		},
		{
			name:         "tags request error",
			branch:       "main",
			tag:          "sq-start",
			responseCode: 404,
			responseBody: `{"message":"Not Found"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++

				if requestCount == 1 {
					// Запрос тегов
					if !strings.Contains(r.URL.Path, "/tags") {
						t.Errorf("Expected tags request, got path: %s", r.URL.Path)
					}
					w.WriteHeader(tt.responseCode)
					w.Write([]byte(tt.responseBody))
					return
				}

				if requestCount == 2 && !tt.expectError {
					// Запрос коммита по SHA
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`[{"sha":"start123","commit":{"message":"Start commit"}}]`))
					return
				}

				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"message":"Not Found"}`))
			}))
			defer server.Close()

			api := &API{
				GiteaURL: server.URL,
				Owner:    "testowner",
				Repo:     "testrepo",
			}

			commit, err := api.findCommitWithTag(tt.branch, tt.tag)

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

// TestGetFirstCommitInHistory тестирует получение первого коммита в истории
func TestGetFirstCommitInHistory(t *testing.T) {
	tests := []struct {
		name         string
		branch       string
		responseCode int
		responseBody string
		expectError  bool
	}{
		{
			name:         "successful get first commit",
			branch:       "main",
			responseCode: 200,
			responseBody: `[{"sha":"newest","commit":{"message":"Newest"}},{"sha":"oldest","commit":{"message":"Oldest"}}]`,
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
				if !strings.Contains(r.URL.Path, "/commits") {
					t.Errorf("Expected commits request, got path: %s", r.URL.Path)
				}
				if r.URL.Query().Get("sha") != tt.branch {
					t.Errorf("Expected branch %s, got %s", tt.branch, r.URL.Query().Get("sha"))
				}
				w.WriteHeader(tt.responseCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			api := &API{
				GiteaURL: server.URL,
				Owner:    "testowner",
				Repo:     "testrepo",
			}

			commit, err := api.getFirstCommitInHistory(tt.branch)

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

// TestCompareBranches тестирует сравнение веток
func TestCompareBranches(t *testing.T) {
	tests := []struct {
		name              string
		base              string
		head              string
		compareResponse   string
		compareError      bool
		hasMergeBase      bool
		needsFindMergeBase bool
		expectError       bool
	}{
		{
			name:            "successful compare with merge base",
			base:            "main",
			head:            "feature",
			compareResponse: `{"merge_base_commit":{"sha":"base123","commit":{"message":"Base"}},"commits":[]}`,
			hasMergeBase:    true,
			expectError:     false,
		},
		{
			name:               "successful compare without merge base",
			base:               "main",
			head:               "feature",
			compareResponse:    `{"commits":[]}`,
			hasMergeBase:       false,
			needsFindMergeBase: true,
			expectError:        false,
		},
		{
			name:         "compare request error",
			base:         "main",
			head:         "notfound",
			compareError: true,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++

				if requestCount == 1 {
					// Запрос сравнения
					if !strings.Contains(r.URL.Path, "/compare/") {
						t.Errorf("Expected compare request, got path: %s", r.URL.Path)
					}
					if tt.compareError {
						w.WriteHeader(http.StatusNotFound)
						w.Write([]byte(`{"message":"Not Found"}`))
						return
					}
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(tt.compareResponse))
					return
				}

				if requestCount >= 2 && tt.needsFindMergeBase {
					// Запросы для findMergeBase - возвращаем коммиты
					if requestCount == 2 {
						// Head commits for findMergeBase
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`[{"sha":"head123","commit":{"message":"Head commit"}}]`))
						return
					}
					if requestCount == 3 {
						// Base commits for findMergeBase - put head123 in the middle, not at the end
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`[{"sha":"latest123","commit":{"message":"Latest commit"}},{"sha":"head123","commit":{"message":"Head commit"}},{"sha":"older123","commit":{"message":"Older commit"}}]`))
						return
					}
				}

				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"message":"Not Found"}`))
			}))
			defer server.Close()

			api := &API{
				GiteaURL: server.URL,
				Owner:    "testowner",
				Repo:     "testrepo",
			}

			result, err := api.compareBranches(tt.base, tt.head)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil {
					t.Error("Expected result but got nil")
				}
				if result != nil && result.MergeBaseCommit == nil {
					t.Error("Expected MergeBaseCommit but got nil")
				}
			}
		})
	}
}

// TestFindMergeBase тестирует поиск общего предка веток
func TestFindMergeBase(t *testing.T) {
	tests := []struct {
		name        string
		base        string
		head        string
		headCommits string
		baseCommits string
		expectError bool
	}{
		{
			name:        "successful find merge base",
			base:        "main",
			head:        "feature",
			headCommits: `[{"sha":"head123","commit":{"message":"Head"}},{"sha":"base123","commit":{"message":"Base"}}]`,
			baseCommits: `[{"sha":"latest","commit":{"message":"Latest"}},{"sha":"base123","commit":{"message":"Base"}},{"sha":"older","commit":{"message":"Older"}}]`,
			expectError: false,
		},
		{
			name:        "merge base not found",
			base:        "main",
			head:        "feature",
			headCommits: `[{"sha":"head123","commit":{"message":"Head"}},{"sha":"notfound","commit":{"message":"Not Found"}}]`,
			baseCommits: `[{"sha":"latest","commit":{"message":"Latest"}},{"sha":"base123","commit":{"message":"Base"}}]`,
			expectError: false, // Возвращает последний коммит базовой ветки
		},
		{
			name:        "head commits error",
			base:        "main",
			head:        "notfound",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++

				if requestCount == 1 {
					// Запрос коммитов head ветки
					if strings.Contains(r.URL.Query().Get("sha"), tt.head) || strings.Contains(r.URL.Path, tt.head) {
						if tt.expectError && tt.head == "notfound" {
							w.WriteHeader(http.StatusNotFound)
							w.Write([]byte(`{"message":"Not Found"}`))
							return
						}
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(tt.headCommits))
						return
					}
				}

				if requestCount == 2 {
					// Запрос коммитов base ветки
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(tt.baseCommits))
					return
				}

				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"message":"Not Found"}`))
			}))
			defer server.Close()

			api := &API{
				GiteaURL: server.URL,
				Owner:    "testowner",
				Repo:     "testrepo",
			}

			commit, err := api.findMergeBase(tt.base, tt.head)

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

// TestGetAllCommits тестирует получение всех коммитов ветки
func TestGetAllCommits(t *testing.T) {
	tests := []struct {
		name         string
		branch       string
		responseCode int
		responseBody string
		expectError  bool
		expectedCount int
	}{
		{
			name:         "successful get all commits",
			branch:       "main",
			responseCode: 200,
			responseBody: `[{"sha":"commit1","commit":{"message":"Message 1"}},{"sha":"commit2","commit":{"message":"Message 2"}}]`,
			expectError:  false,
			expectedCount: 2,
		},
		{
			name:         "empty commits",
			branch:       "empty",
			responseCode: 200,
			responseBody: `[]`,
			expectError:  false,
			expectedCount: 0,
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
				if !strings.Contains(r.URL.Path, "/commits") {
					t.Errorf("Expected commits request, got path: %s", r.URL.Path)
				}
				if r.URL.Query().Get("sha") != tt.branch {
					t.Errorf("Expected branch %s, got %s", tt.branch, r.URL.Query().Get("sha"))
				}
				w.WriteHeader(tt.responseCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			api := &API{
				GiteaURL: server.URL,
				Owner:    "testowner",
				Repo:     "testrepo",
			}

			commits, err := api.getAllCommits(tt.branch)

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

// TestGetCommitBySHA тестирует получение коммита по SHA
func TestGetCommitBySHA(t *testing.T) {
	tests := []struct {
		name         string
		sha          string
		responseCode int
		responseBody string
		expectError  bool
	}{
		{
			name:         "successful get commit by SHA",
			sha:          "abc123",
			responseCode: 200,
			responseBody: `[{"sha":"abc123","commit":{"message":"Test commit"}},{"sha":"def456","commit":{"message":"Other commit"}}]`,
			expectError:  false,
		},
		{
			name:         "commit not found in response",
			sha:          "notfound",
			responseCode: 200,
			responseBody: `[{"sha":"abc123","commit":{"message":"Test commit"}}]`,
			expectError:  true,
		},
		{
			name:         "empty response",
			sha:          "empty",
			responseCode: 200,
			responseBody: `[]`,
			expectError:  true,
		},
		{
			name:         "API error",
			sha:          "error",
			responseCode: 404,
			responseBody: `{"message":"Not Found"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.Contains(r.URL.Path, "/commits") {
					t.Errorf("Expected commits request, got path: %s", r.URL.Path)
				}
				if r.URL.Query().Get("sha") != tt.sha {
					t.Errorf("Expected SHA %s, got %s", tt.sha, r.URL.Query().Get("sha"))
				}
				w.WriteHeader(tt.responseCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			api := &API{
				GiteaURL: server.URL,
				Owner:    "testowner",
				Repo:     "testrepo",
			}

			commit, err := api.getCommitBySHA(tt.sha)

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
				if commit != nil && commit.SHA != tt.sha {
					t.Errorf("Expected SHA %s, got %s", tt.sha, commit.SHA)
				}
			}
		})
	}
}