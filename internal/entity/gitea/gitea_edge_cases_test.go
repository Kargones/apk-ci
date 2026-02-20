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

// TestSendReqEdgeCases тестирует edge cases функции sendReq
func TestSendReqEdgeCases(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name         string
		reqBody      string
		method       string
		responseCode int
		responseBody string
		expectError  bool
	}{
		{
			name:         "successful request with empty body",
			reqBody:      "",
			method:       "GET",
			responseCode: 200,
			responseBody: "success",
			expectError:  false,
		},
		{
			name:         "successful request with body",
			reqBody:      `{"test":"data"}`,
			method:       "POST",
			responseCode: 201,
			responseBody: "created",
			expectError:  false,
		},
		{
			name:         "request with various status codes",
			reqBody:      "",
			method:       "GET",
			responseCode: 404,
			responseBody: "not found",
			expectError:  false, // sendReq doesn't return error for status codes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.method {
					t.Errorf("Expected method %s, got %s", tt.method, r.Method)
				}
				w.WriteHeader(tt.responseCode)
				if _, err := w.Write([]byte(tt.responseBody)); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			api := &API{
				GiteaURL:    server.URL,
				AccessToken: "testtoken",
			}

			statusCode, body, err := api.sendReq(ctx, server.URL+"/test", tt.reqBody, tt.method)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if statusCode != tt.responseCode {
					t.Errorf("Expected status code %d, got %d", tt.responseCode, statusCode)
				}
				if body != tt.responseBody {
					t.Errorf("Expected body %s, got %s", tt.responseBody, body)
				}
			}
		})
	}
}

// TestGetFileContentEdgeCases тестирует edge cases для GetFileContent
func TestGetFileContentEdgeCases(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name            string
		fileName        string
		responseCode    int
		responseBody    string
		expectError     bool
		expectedContent string
	}{
		{
			name:         "file with direct HTTP URL",
			fileName:     "http://example.com/file.txt",
			responseCode: 200,
			responseBody: `{"content":"dGVzdA==","encoding":"base64"}`,
			expectError:  false,
			expectedContent: "test",
		},
		{
			name:         "file with direct HTTPS URL",
			fileName:     "https://example.com/file.txt",
			responseCode: 200,
			responseBody: `{"content":"dGVzdA==","encoding":"base64"}`,
			expectError:  false,
			expectedContent: "test",
		},
		{
			name:         "file with non-base64 encoding",
			fileName:     "test.txt",
			responseCode: 200,
			responseBody: `{"content":"test","encoding":"plain"}`,
			expectError:  false, // Function returns nil for non-base64, which is handled as success
			expectedContent: "", // Empty content
		},
		{
			name:         "file with malformed base64",
			fileName:     "test.txt",
			responseCode: 200,
			responseBody: `{"content":"invalid-base64!","encoding":"base64"}`,
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

			// For direct URLs, use the server URL
			fileName := tt.fileName
			if strings.HasPrefix(fileName, "http://") || strings.HasPrefix(fileName, "https://") {
				fileName = server.URL + "/file.txt"
			}

			api := &API{
				GiteaURL: server.URL,
				Owner:    "testowner",
				Repo:     "testrepo",
			}

			content, err := api.GetFileContent(ctx, fileName)

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

// TestGetConfigDataEdgeCases тестирует edge cases для GetConfigData
func TestGetConfigDataEdgeCases(t *testing.T) {
	ctx := context.Background()
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
			name:         "empty decoded content",
			filename:     "empty.yaml",
			responseCode: 200,
			responseBody: `{"content":"","encoding":"base64"}`,
			expectError:  true, // Empty content should return error
		},
		{
			name:         "malformed JSON response",
			filename:     "malformed.yaml",
			responseCode: 200,
			responseBody: `{"invalid":json}`,
			expectError:  true,
		},
		{
			name:         "missing content field",
			filename:     "missing.yaml",
			responseCode: 200,
			responseBody: `{"encoding":"base64"}`,
			expectError:  true, // Empty content triggers error
		},
		{
			name:         "content with newlines",
			filename:     "newlines.yaml",
			responseCode: 200,
			responseBody: `{"content":"dGVz\ndA==","encoding":"base64"}`,
			expectError:  false,
			expectedContent: "test",
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
				GiteaURL:   server.URL,
				Owner:      "testowner",
				Repo:       "testrepo",
				BaseBranch: "main",
			}

			content, err := api.GetConfigData(ctx, logger, tt.filename)

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

// TestAnalyzeProjectStructureEdgeCases тестирует edge cases для AnalyzeProjectStructure
func TestAnalyzeProjectStructureEdgeCases(t *testing.T) {
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
			name:         "only hidden directories",
			branch:       "main",
			responseCode: 200,
			responseBody: `[{"name":".git","type":"dir"},{"name":".github","type":"dir"}]`,
			expectError:  false,
			expectedResult: []string{},
		},
		{
			name:         "mixed files and directories",
			branch:       "main",
			responseCode: 200,
			responseBody: `[{"name":"README.md","type":"file"},{"name":"MyProject","type":"dir"},{"name":"MyProject.Ext1","type":"dir"},{"name":"script.sh","type":"file"}]`,
			expectError:  false,
			expectedResult: []string{"MyProject", "Ext1"},
		},
		{
			name:         "multiple projects without extensions",
			branch:       "main",
			responseCode: 200,
			responseBody: `[{"name":"Project1","type":"dir"},{"name":"Project2","type":"dir"}]`,
			expectError:  true, // Multiple directories without extensions should error
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

// TestIsUserInTeamEdgeCases тестирует edge cases для IsUserInTeam
func TestIsUserInTeamEdgeCases(t *testing.T) {
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
			name:              "team search fails",
			username:          "testuser",
			orgName:           "testorg",
			teamName:          "testteam",
			searchResponseCode: 500,
			searchResponseBody: `{"message":"Internal Server Error"}`,
			expectError:       true,
		},
		{
			name:              "malformed search response",
			username:          "testuser",
			orgName:           "testorg",
			teamName:          "testteam",
			searchResponseCode: 200,
			searchResponseBody: `{"invalid":json}`,
			expectError:       true,
		},
		{
			name:              "membership check returns 200 (user is member)",
			username:          "testuser",
			orgName:           "testorg",
			teamName:          "testteam",
			searchResponseCode: 200,
			searchResponseBody: `{"data":[{"id":123,"name":"testteam"}]}`,
			memberResponseCode: 200,
			expectError:       false,
			expectedResult:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++

				if requestCount == 1 {
					// Team search request
					w.WriteHeader(tt.searchResponseCode)
					if _, err := w.Write([]byte(tt.searchResponseBody)); err != nil {
						t.Errorf("Failed to write search response: %v", err)
					}
					return
				}

				if requestCount == 2 && !tt.expectError {
					// Membership check request
					w.WriteHeader(tt.memberResponseCode)
					return
				}

				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"message":"Not Found"}`))
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

// TestConflictPREdgeCases тестирует edge cases для ConflictPR
func TestConflictPREdgeCases(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name             string
		prNumber         int64
		responseCode     int
		responseBody     string
		expectError      bool
		expectedConflict bool
	}{
		{
			name:         "malformed JSON response",
			prNumber:     123,
			responseCode: 200,
			responseBody: `{"invalid":json}`,
			expectError:  true,
		},
		{
			name:             "PR is mergeable",
			prNumber:         123,
			responseCode:     200,
			responseBody:     `{"mergeable":true,"number":123}`,
			expectError:      false,
			expectedConflict: false,
		},
		{
			name:             "PR has conflicts",
			prNumber:         123,
			responseCode:     200,
			responseBody:     `{"mergeable":false,"number":123}`,
			expectError:      false,
			expectedConflict: true,
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

			hasConflict, err := api.ConflictPR(ctx, tt.prNumber)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if hasConflict != tt.expectedConflict {
					t.Errorf("Expected conflict %v, got %v", tt.expectedConflict, hasConflict)
				}
			}
		})
	}
}