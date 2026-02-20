// Package sonarqube provides tests for SonarQube entity implementation.
package sonarqube

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/stretchr/testify/assert"
)

const (
	expectedAuth   = "Bearer test-token"
	expectedMethod = "GET"
)

// TestNewSonarQubeEntity tests the creation of a new SonarQubeEntity.
func TestNewSonarQubeEntity(t *testing.T) {
	cfg := &config.SonarQubeConfig{
		URL:     "http://localhost:9000",
		Token:   "test-token",
		Timeout: 30 * time.Second,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	entity := NewEntity(cfg, logger)

	assert.NotNil(t, entity)
	assert.Equal(t, cfg, entity.config)
	assert.Equal(t, logger, entity.logger)
	assert.NotNil(t, entity.client)
	assert.Equal(t, cfg.Timeout, entity.client.Timeout)
}

// TestSonarQubeEntity_authenticate tests the authenticate method.
func TestSonarQubeEntity_authenticate(t *testing.T) {
	cfg := &config.SonarQubeConfig{
		URL:   "http://localhost:9000",
		Token: "test-token",
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewEntity(cfg, logger)

	req, err := http.NewRequest(expectedMethod, "http://localhost:9000", nil)
	assert.NoError(t, err)

	err = entity.authenticate(req)
	assert.NoError(t, err)

	// Check that the Authorization header is set correctly
	expectedAuth := "Bearer " + cfg.Token
	assert.Equal(t, expectedAuth, req.Header.Get("Authorization"))

	// Test with empty token
	cfg.Token = ""
	entity = NewEntity(cfg, logger)

	err = entity.authenticate(req)
	assert.Error(t, err)
	assert.IsType(t, &ValidationError{}, err)
}

// TestSonarQubeEntity_Authenticate tests the Authenticate method.
func TestSonarQubeEntity_Authenticate(t *testing.T) {
	ctx := context.Background()
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the request path
		if r.URL.Path != "/api/authentication/validate" {
			t.Errorf("Expected request to '/api/authentication/validate', got '%s'", r.URL.Path)
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		// Check the authorization header
		auth := r.Header.Get("Authorization")

		if auth != expectedAuth {
			t.Errorf("Expected Authorization header '%s', got '%s'", expectedAuth, auth)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Return a successful response
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"valid": true}`)); err != nil {
			t.Logf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.SonarQubeConfig{
		URL:     server.URL,
		Token:   "original-token",
		Timeout: 30 * time.Second,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewEntity(cfg, logger)

	err := entity.Authenticate(ctx, "test-token")
	assert.NoError(t, err)

	// Verify that the original token is restored
	assert.Equal(t, "original-token", entity.config.Token)
}

// TestSonarQubeEntity_ValidateToken tests the ValidateToken method.
func TestSonarQubeEntity_ValidateToken(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the request path
		if r.URL.Path != "/api/authentication/validate" {
			t.Errorf("Expected request to '/api/authentication/validate', got '%s'", r.URL.Path)
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		// Check the authorization header
		auth := r.Header.Get("Authorization")

		if auth != expectedAuth {
			t.Errorf("Expected Authorization header '%s', got '%s'", expectedAuth, auth)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Return a successful response
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"valid": true}`)); err != nil {
			t.Logf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.SonarQubeConfig{
		URL:     server.URL,
		Token:   "test-token",
		Timeout: 30 * time.Second,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewEntity(cfg, logger)

	err := entity.ValidateToken(context.Background())
	assert.NoError(t, err)
}

// TestSonarQubeEntity_CreateProject tests the CreateProject method.
func TestSonarQubeEntity_CreateProject(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the request path
		if r.URL.Path != "/api/projects/create" {
			t.Errorf("Expected request to '/api/projects/create', got '%s'", r.URL.Path)
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		// Check the method
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got '%s'", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check the authorization header
		auth := r.Header.Get("Authorization")

		if auth != expectedAuth {
			t.Errorf("Expected Authorization header '%s', got '%s'", expectedAuth, auth)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Parse the form data
		err := r.ParseForm()
		if err != nil {
			t.Errorf("Failed to parse form data: %v", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Check required fields
		name := r.FormValue("name")
		if name == "" {
			t.Error("Missing or invalid 'name' field")
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		project := r.FormValue("project")
		if project == "" {
			t.Error("Missing or invalid 'project' field")
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		visibility := r.FormValue("visibility")
		if visibility == "" {
			t.Error("Missing or invalid 'visibility' field")
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Return a successful response
		response := map[string]interface{}{
			"project": map[string]interface{}{
				"key":        project,
				"name":       name,
				"visibility": visibility,
				"tags":       []string{},
				"created":    time.Now().Format(time.RFC3339),
				"updated":    time.Now().Format(time.RFC3339),
				"metadata":   map[string]string{},
			},
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Logf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.SonarQubeConfig{
		URL:               server.URL,
		Token:             "test-token",
		Timeout:           30 * time.Second,
		DefaultVisibility: "public",
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewEntity(cfg, logger)

	project, err := entity.CreateProject(context.Background(), "owner", "repo", "branch")
	assert.NoError(t, err)
	assert.NotNil(t, project)
	assert.Equal(t, "owner_repo_branch", project.Key)
	assert.Equal(t, "owner/repo (branch)", project.Name)
	assert.Equal(t, "public", project.Visibility)
}

// TestSonarQubeEntity_GetProject tests the GetProject method.
func TestSonarQubeEntity_GetProject(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the request path
		if r.URL.Path != "/api/projects/search" {
			t.Errorf("Expected request to '/api/projects/search', got '%s'", r.URL.Path)
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		// Check the method
		if r.Method != expectedMethod {
			t.Errorf("Expected GET request, got '%s'", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check the authorization header
		auth := r.Header.Get("Authorization")

		if auth != expectedAuth {
			t.Errorf("Expected Authorization header '%s', got '%s'", expectedAuth, auth)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check query parameters
		projectKey := r.URL.Query().Get("projects")
		if projectKey == "" {
			t.Error("Missing 'projects' query parameter")
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Check page size parameter
		ps := r.URL.Query().Get("ps")
		if ps != "1" {
			t.Errorf("Expected 'ps' parameter to be '1', got '%s'", ps)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Return a successful response
		response := map[string]interface{}{
			"components": []map[string]interface{}{
				{
					"key":        projectKey,
					"name":       "Test Project",
					"visibility": "public",
					"tags":       []string{},
					"created":    time.Now().Format(time.RFC3339),
					"updated":    time.Now().Format(time.RFC3339),
					"metadata":   map[string]string{},
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Logf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.SonarQubeConfig{
		URL:     server.URL,
		Token:   "test-token",
		Timeout: 30 * time.Second,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewEntity(cfg, logger)

	project, err := entity.GetProject(context.Background(), "test-project")
	assert.NoError(t, err)
	assert.NotNil(t, project)
	assert.Equal(t, "test-project", project.Key)
	assert.Equal(t, "Test Project", project.Name)
	assert.Equal(t, "public", project.Visibility)
}

// TestSonarQubeEntity_GetProject_NotFound tests the GetProject method when project is not found.
func TestSonarQubeEntity_GetProject_NotFound(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Return an empty components response
		response := map[string]interface{}{
			"components": []map[string]interface{}{},
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Logf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.SonarQubeConfig{
		URL:     server.URL,
		Token:   "test-token",
		Timeout: 30 * time.Second,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewEntity(cfg, logger)

	project, err := entity.GetProject(context.Background(), "non-existent-project")
	assert.Error(t, err)
	assert.Nil(t, project)
	assert.IsType(t, &Error{}, err)

	sqErr, ok := err.(*Error)
	assert.True(t, ok)
	assert.Equal(t, 404, sqErr.Code)
	assert.Equal(t, "Project not found", sqErr.Message)
}

// TestSonarQubeEntity_GetProject_RequestError tests the GetProject method when makeRequest fails.
//nolint:dupl // similar test structure
func TestSonarQubeEntity_GetProject_RequestError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte("Internal Server Error")); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.SonarQubeConfig{
		URL:     server.URL,
		Token:   "test-token",
		Timeout: 30 * time.Second,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewEntity(cfg, logger)

	project, err := entity.GetProject(context.Background(), "test-project")
	assert.Error(t, err)
	assert.Nil(t, project)
	assert.Contains(t, err.Error(), "failed to get project")
}

// TestSonarQubeEntity_GetProject_InvalidJSON tests the GetProject method when JSON parsing fails.
//nolint:dupl // similar test structure
func TestSonarQubeEntity_GetProject_InvalidJSON(t *testing.T) {
	// Create a test server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("invalid json response")); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.SonarQubeConfig{
		URL:     server.URL,
		Token:   "test-token",
		Timeout: 30 * time.Second,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewEntity(cfg, logger)

	project, err := entity.GetProject(context.Background(), "test-project")
	assert.Error(t, err)
	assert.Nil(t, project)
	assert.Contains(t, err.Error(), "failed to parse project retrieval response")
}

// TestSonarQubeEntity_UpdateProject tests the UpdateProject method.
// createTestProjectServer creates a test server for project operations
func createTestProjectServer(t *testing.T, expectedPath string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the request path
		if r.URL.Path != expectedPath {
			t.Errorf("Expected request to '%s', got '%s'", expectedPath, r.URL.Path)
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		// Check the method
		const expectedMethod = "POST"
		if r.Method != expectedMethod {
			t.Errorf("Expected %s request, got '%s'", expectedMethod, r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check the authorization header
		auth := r.Header.Get("Authorization")

		if auth != expectedAuth {
			t.Errorf("Expected Authorization header '%s', got '%s'", expectedAuth, auth)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Parse the form data
		err := r.ParseForm()
		if err != nil {
			t.Errorf("Failed to parse form data: %v", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Check required field
		project := r.FormValue("project")
		if project == "" {
			t.Error("Missing or invalid 'project' field")
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Return a successful response
		w.WriteHeader(http.StatusOK)
	}))
}

func TestSonarQubeEntity_UpdateProject(t *testing.T) {
	// Create a test server using the helper function
	server := createTestProjectServer(t, "/api/projects/update")
	defer server.Close()

	cfg := &config.SonarQubeConfig{
		URL:     server.URL,
		Token:   "test-token",
		Timeout: 30 * time.Second,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewEntity(cfg, logger)

	updates := &ProjectUpdate{
		Name:        "Updated Project Name",
		Description: "Updated project description",
		Visibility:  "private",
	}

	err := entity.UpdateProject(context.Background(), "test-project", updates)
	assert.NoError(t, err)
}

// TestSonarQubeEntity_DeleteProject tests the DeleteProject method.
func TestSonarQubeEntity_DeleteProject(t *testing.T) {
	// Create a test server using the helper function
	server := createTestProjectServer(t, "/api/projects/delete")
	defer server.Close()

	cfg := &config.SonarQubeConfig{
		URL:     server.URL,
		Token:   "test-token",
		Timeout: 30 * time.Second,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewEntity(cfg, logger)

	err := entity.DeleteProject(context.Background(), "test-project")
	assert.NoError(t, err)
}

// TestSonarQubeEntity_ListProjects tests the ListProjects method.
func TestSonarQubeEntity_ListProjects(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the request path
		if r.URL.Path != "/api/projects/search" {
			t.Errorf("Expected request to '/api/projects/search', got '%s'", r.URL.Path)
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		// Check the method
		if r.Method != expectedMethod {
			t.Errorf("Expected GET request, got '%s'", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check the authorization header
		auth := r.Header.Get("Authorization")

		if auth != expectedAuth {
			t.Errorf("Expected Authorization header '%s', got '%s'", expectedAuth, auth)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check query parameters
		q := r.URL.Query().Get("q")
		if q == "" {
			t.Error("Missing 'q' query parameter")
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Return a successful response with some projects
		response := map[string]interface{}{
			"components": []map[string]interface{}{
				{
					"key":        "prefix_owner_repo_branch1",
					"name":       "owner/repo (branch1)",
					"visibility": "public",
					"tags":       []string{},
					"created":    time.Now().Format(time.RFC3339),
					"updated":    time.Now().Format(time.RFC3339),
					"metadata":   map[string]string{},
				},
				{
					"key":        "prefix_owner_repo_branch2",
					"name":       "owner/repo (branch2)",
					"visibility": "private",
					"tags":       []string{},
					"created":    time.Now().Format(time.RFC3339),
					"updated":    time.Now().Format(time.RFC3339),
					"metadata":   map[string]string{},
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Logf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.SonarQubeConfig{
		URL:           server.URL,
		Token:         "test-token",
		Timeout:       30 * time.Second,
		ProjectPrefix: "prefix",
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewEntity(cfg, logger)

	projects, err := entity.ListProjects(context.Background(), "owner", "repo")
	assert.NoError(t, err)
	assert.Len(t, projects, 2)

	// Check the first project
	assert.Equal(t, "prefix_owner_repo_branch1", projects[0].Key)
	assert.Equal(t, "owner/repo (branch1)", projects[0].Name)
	assert.Equal(t, "public", projects[0].Visibility)

	// Check the second project
	assert.Equal(t, "prefix_owner_repo_branch2", projects[1].Key)
	assert.Equal(t, "owner/repo (branch2)", projects[1].Name)
	assert.Equal(t, "private", projects[1].Visibility)
}

// TestSonarQubeEntity_GetAnalyses tests the GetAnalyses method.
func TestSonarQubeEntity_GetAnalyses(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the request path
		if r.URL.Path != "/api/project_analyses/search" {
			t.Errorf("Expected request to '/api/project_analyses/search', got '%s'", r.URL.Path)
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		// Check the method
		const expectedMethod = "GET"
		if r.Method != expectedMethod {
			t.Errorf("Expected %s request, got '%s'", expectedMethod, r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check the authorization header
		auth := r.Header.Get("Authorization")
		const expectedAuth = "Bearer test-token"
		if auth != expectedAuth {
			t.Errorf("Expected Authorization header '%s', got '%s'", expectedAuth, auth)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check query parameters
		project := r.URL.Query().Get("project")
		if project == "" {
			t.Error("Missing 'project' query parameter")
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Return a successful response with some analyses
		response := map[string]interface{}{
			"analyses": []map[string]interface{}{
				{
					"id":         "analysis-1",
					"projectKey": project,
					"date":       time.Now().Format(time.RFC3339),
					"revision":   "abc123",
					"status":     map[string]string{"Status": "SUCCESS"},
					"metrics":    map[string]string{},
				},
				{
					"id":         "analysis-2",
					"projectKey": project,
					"date":       time.Now().Format(time.RFC3339),
					"revision":   "def456",
					"status":     map[string]string{"Status": "FAILED"},
					"metrics":    map[string]string{},
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Logf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.SonarQubeConfig{
		URL:     server.URL,
		Token:   "test-token",
		Timeout: 30 * time.Second,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewEntity(cfg, logger)

	analyses, err := entity.GetAnalyses(context.Background(), "test-project")
	assert.NoError(t, err)
	assert.Len(t, analyses, 2)

	// Check the first analysis
	assert.Equal(t, "analysis-1", analyses[0].ID)
	assert.Equal(t, "test-project", analyses[0].ProjectKey)
	assert.Equal(t, "SUCCESS", analyses[0].Status.Status)

	// Check the second analysis
	assert.Equal(t, "analysis-2", analyses[1].ID)
	assert.Equal(t, "test-project", analyses[1].ProjectKey)
	assert.Equal(t, "FAILED", analyses[1].Status.Status)
}

// TestSonarQubeEntity_GetAnalysisStatus tests the GetAnalysisStatus method.
// createTestAnalysisServer creates a test server for analysis operations
func createTestAnalysisServer(t *testing.T, expectedPath string, expectedQueryKey string, responseKey string, responseValue string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the request path
		if r.URL.Path != expectedPath {
			t.Errorf("Expected request to '%s', got '%s'", expectedPath, r.URL.Path)
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		// Check the method
		if r.Method != expectedMethod {
			t.Errorf("Expected GET request, got '%s'", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check the authorization header
		auth := r.Header.Get("Authorization")
		const expectedAuth = "Bearer test-token"
		if auth != expectedAuth {
			t.Errorf("Expected Authorization header '%s', got '%s'", expectedAuth, auth)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check query parameters
		queryValue := r.URL.Query().Get(expectedQueryKey)
		if queryValue == "" {
			t.Errorf("Missing '%s' query parameter", expectedQueryKey)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Return a successful response
		response := map[string]interface{}{
			responseKey: map[string]interface{}{
				"status": responseValue,
			},
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Logf("Failed to encode response: %v", err)
		}
	}))
}

func TestSonarQubeEntity_GetAnalysisStatus(t *testing.T) {
	// Create a test server using the helper function
	server := createTestAnalysisServer(t, "/api/ce/task", "analysisId", "task", "SUCCESS")
	defer server.Close()

	cfg := &config.SonarQubeConfig{
		URL:     server.URL,
		Token:   "test-token",
		Timeout: 30 * time.Second,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewEntity(cfg, logger)

	status, err := entity.GetAnalysisStatus(context.Background(), "analysis-123")
	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, "SUCCESS", status.Status)
}

// TestSonarQubeEntity_GetIssues tests the GetIssues method.
func TestSonarQubeEntity_GetIssues(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the request path
		if r.URL.Path != "/api/issues/search" {
			t.Errorf("Expected request to '/api/issues/search', got '%s'", r.URL.Path)
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		// Check the method
		if r.Method != expectedMethod {
			t.Errorf("Expected GET request, got '%s'", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check the authorization header
		auth := r.Header.Get("Authorization")

		if auth != expectedAuth {
			t.Errorf("Expected Authorization header '%s', got '%s'", expectedAuth, auth)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check query parameters
		componentKeys := r.URL.Query().Get("componentKeys")
		if componentKeys == "" {
			t.Error("Missing 'componentKeys' query parameter")
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Return a successful response with some issues
		response := map[string]interface{}{
			"issues": []map[string]interface{}{
				{
					"key":       "issue-1",
					"rule":      "rule1",
					"severity":  "MAJOR",
					"component": "component1",
					"line":      10,
					"message":   "Issue message 1",
					"type":      "BUG",
					"createdAt": time.Now().Format(time.RFC3339),
					"updatedAt": time.Now().Format(time.RFC3339),
				},
				{
					"key":       "issue-2",
					"rule":      "rule2",
					"severity":  "MINOR",
					"component": "component2",
					"line":      20,
					"message":   "Issue message 2",
					"type":      "CODE_SMELL",
					"createdAt": time.Now().Format(time.RFC3339),
					"updatedAt": time.Now().Format(time.RFC3339),
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Logf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.SonarQubeConfig{
		URL:     server.URL,
		Token:   "test-token",
		Timeout: 30 * time.Second,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewEntity(cfg, logger)

	issues, err := entity.GetIssues(context.Background(), "test-project", nil)
	assert.NoError(t, err)
	assert.Len(t, issues, 2)

	// Check the first issue
	assert.Equal(t, "issue-1", issues[0].Key)
	assert.Equal(t, "rule1", issues[0].Rule)
	assert.Equal(t, "MAJOR", issues[0].Severity)
	assert.Equal(t, "component1", issues[0].Component)
	assert.Equal(t, 10, issues[0].Line)
	assert.Equal(t, "Issue message 1", issues[0].Message)
	assert.Equal(t, "BUG", issues[0].Type)

	// Check the second issue
	assert.Equal(t, "issue-2", issues[1].Key)
	assert.Equal(t, "rule2", issues[1].Rule)
	assert.Equal(t, "MINOR", issues[1].Severity)
	assert.Equal(t, "component2", issues[1].Component)
	assert.Equal(t, 20, issues[1].Line)
	assert.Equal(t, "Issue message 2", issues[1].Message)
	assert.Equal(t, "CODE_SMELL", issues[1].Type)
}

// TestSonarQubeEntity_GetQualityGateStatus tests the GetQualityGateStatus method.
func TestSonarQubeEntity_GetQualityGateStatus(t *testing.T) {
	// Create a test server using the helper function
	server := createTestAnalysisServer(t, "/api/qualitygates/project_status", "projectKey", "projectStatus", "OK")
	defer server.Close()

	cfg := &config.SonarQubeConfig{
		URL:     server.URL,
		Token:   "test-token",
		Timeout: 30 * time.Second,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewEntity(cfg, logger)

	status, err := entity.GetQualityGateStatus(context.Background(), "test-project")
	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, "OK", status.Status)
}

// TestSonarQubeEntity_GetMetrics tests the GetMetrics method.
func TestSonarQubeEntity_GetMetrics(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the request path
		if r.URL.Path != "/api/measures/component" {
			t.Errorf("Expected request to '/api/measures/component', got '%s'", r.URL.Path)
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		// Check the method
		if r.Method != expectedMethod {
			t.Errorf("Expected GET request, got '%s'", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check the authorization header
		auth := r.Header.Get("Authorization")

		if auth != expectedAuth {
			t.Errorf("Expected Authorization header '%s', got '%s'", expectedAuth, auth)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check query parameters
		component := r.URL.Query().Get("component")
		if component == "" {
			t.Error("Missing 'component' query parameter")
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		metricKeys := r.URL.Query().Get("metricKeys")
		if metricKeys == "" {
			t.Error("Missing 'metricKeys' query parameter")
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Return a successful response with some metrics
		response := map[string]interface{}{
			"component": map[string]interface{}{
				"measures": []map[string]interface{}{
					{
						"metric": "coverage",
						"value":  "85.5",
					},
					{
						"metric": "bugs",
						"value":  "5",
					},
					{
						"metric": "code_smells",
						"value":  "10",
					},
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Logf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.SonarQubeConfig{
		URL:     server.URL,
		Token:   "test-token",
		Timeout: 30 * time.Second,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewEntity(cfg, logger)

	metricKeys := []string{"coverage", "bugs", "code_smells"}
	metrics, err := entity.GetMetrics(context.Background(), "test-project", metricKeys)
	assert.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.Len(t, metrics.Metrics, 3)

	// Check the metrics
	assert.Equal(t, 85.5, metrics.Metrics["coverage"])
	assert.Equal(t, 5.0, metrics.Metrics["bugs"])
	assert.Equal(t, 10.0, metrics.Metrics["code_smells"])
}

// TestSonarQubeEntity_GetQualityProfiles tests the GetQualityProfiles method.
func TestSonarQubeEntity_GetQualityProfiles(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the request path
		if r.URL.Path != "/api/qualityprofiles/search" {
			t.Errorf("Expected request to '/api/qualityprofiles/search', got '%s'", r.URL.Path)
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		// Check the method
		if r.Method != expectedMethod {
			t.Errorf("Expected GET request, got '%s'", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check the authorization header
		auth := r.Header.Get("Authorization")

		if auth != expectedAuth {
			t.Errorf("Expected Authorization header '%s', got '%s'", expectedAuth, auth)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check query parameters
		project := r.URL.Query().Get("project")
		if project == "" {
			t.Error("Missing 'project' query parameter")
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Return a successful response with some quality profiles
		response := map[string]interface{}{
			"profiles": []map[string]interface{}{
				{
					"key":       "profile-1",
					"name":      "Profile 1",
					"language":  "go",
					"isDefault": true,
				},
				{
					"key":       "profile-2",
					"name":      "Profile 2",
					"language":  "java",
					"isDefault": false,
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Logf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.SonarQubeConfig{
		URL:     server.URL,
		Token:   "test-token",
		Timeout: 30 * time.Second,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewEntity(cfg, logger)

	profiles, err := entity.GetQualityProfiles(context.Background(), "test-project")
	assert.NoError(t, err)
	assert.Len(t, profiles, 2)

	// Check the first profile
	assert.Equal(t, "profile-1", profiles[0].Key)
	assert.Equal(t, "Profile 1", profiles[0].Name)
	assert.Equal(t, "go", profiles[0].Language)
	assert.True(t, profiles[0].IsDefault)

	// Check the second profile
	assert.Equal(t, "profile-2", profiles[1].Key)
	assert.Equal(t, "Profile 2", profiles[1].Name)
	assert.Equal(t, "java", profiles[1].Language)
	assert.False(t, profiles[1].IsDefault)
}

// TestSonarQubeEntity_GetQualityGates tests the GetQualityGates method.
func TestSonarQubeEntity_GetQualityGates(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the request path
		if r.URL.Path != "/api/qualitygates/list" {
			t.Errorf("Expected request to '/api/qualitygates/list', got '%s'", r.URL.Path)
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		// Check the method
		if r.Method != expectedMethod {
			t.Errorf("Expected GET request, got '%s'", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check the authorization header
		auth := r.Header.Get("Authorization")

		if auth != expectedAuth {
			t.Errorf("Expected Authorization header '%s', got '%s'", expectedAuth, auth)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Return a successful response with some quality gates
		response := map[string]interface{}{
			"qualitygates": []map[string]interface{}{
				{
					"id":        1,
					"name":      "Gate 1",
					"isDefault": true,
				},
				{
					"id":        2,
					"name":      "Gate 2",
					"isDefault": false,
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Logf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.SonarQubeConfig{
		URL:     server.URL,
		Token:   "test-token",
		Timeout: 30 * time.Second,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewEntity(cfg, logger)

	gates, err := entity.GetQualityGates(context.Background())
	assert.NoError(t, err)
	assert.Len(t, gates, 2)

	// Check the first gate
	assert.Equal(t, 1, gates[0].ID)
	assert.Equal(t, "Gate 1", gates[0].Name)
	assert.True(t, gates[0].IsDefault)

	// Check the second gate
	assert.Equal(t, 2, gates[1].ID)
	assert.Equal(t, "Gate 2", gates[1].Name)
	assert.False(t, gates[1].IsDefault)
}

// TestSonarQubeEntity_GetProject_Integration tests the GetProject method with real SonarQube server.
// This test is disabled by default. To run it:
// 1. Set up a real SonarQube server
// 2. Fill in the connection parameters below
// 3. Remove the t.Skip() line
// 4. Run: go test -v ./internal/entity/sonarqube -run TestSonarQubeEntity_GetProject_Integration
func TestSonarQubeEntity_GetProject_Integration(t *testing.T) {
	// Skip this test by default - remove this line to run integration test
	t.Skip("Integration test - requires real SonarQube server")

	// ========== MANUAL CONFIGURATION BLOCK ==========
	// Set env vars: SONARQUBE_URL, SONARQUBE_TOKEN, SONARQUBE_PROJECT
	sonarQubeURL := os.Getenv("SONARQUBE_URL")
	sonarQubeToken := os.Getenv("SONARQUBE_TOKEN")
	projectKey := os.Getenv("SONARQUBE_PROJECT")
	// ================================================

	// Validate that required parameters are filled
	if sonarQubeURL == "" {
		t.Fatal("SONARQUBE_URL env var must be set")
	}
	if sonarQubeToken == "" {
		t.Fatal("SONARQUBE_TOKEN env var must be set")
	}
	if projectKey == "" {
		t.Fatal("SONARQUBE_PROJECT env var must be set")
	}

	// Create configuration
	cfg := &config.SonarQubeConfig{
		URL:     sonarQubeURL,
		Token:   sonarQubeToken,
		Timeout: 30 * time.Second,
	}

	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Create entity
	entity := NewEntity(cfg, logger)

	// Test authentication first
	t.Log("Testing authentication...")
	err := entity.ValidateToken(context.Background())
	if err != nil {
		t.Fatalf("Authentication failed: %v", err)
	}
	t.Log("Authentication successful")

	// Test GetProject
	t.Logf("Getting project with key: %s", projectKey)
	project, err := entity.GetProject(context.Background(), projectKey)

	// Check results
	if err != nil {
		t.Logf("GetProject failed: %v", err)
		// Check if it's a "not found" error
		if sqErr, ok := err.(*Error); ok && sqErr.Code == 404 {
			t.Logf("Project '%s' not found on SonarQube server", projectKey)
			t.Log("This might be expected if the project doesn't exist")
		} else {
			t.Fatalf("Unexpected error: %v", err)
		}
		return
	}

	// Validate project data
	assert.NotNil(t, project, "Project should not be nil")
	assert.Equal(t, projectKey, project.Key, "Project key should match")
	assert.NotEmpty(t, project.Name, "Project name should not be empty")

	// Log project details
	t.Logf("Project found successfully:")
	t.Logf("  Key: %s", project.Key)
	t.Logf("  Name: %s", project.Name)
	t.Logf("  Visibility: %s", project.Visibility)
	if len(project.Tags) > 0 {
		t.Logf("  Tags: %v", project.Tags)
	}
	if !project.Created.IsZero() {
		t.Logf("  Created: %s", project.Created.Format(time.RFC3339))
	}
	if !project.Updated.IsZero() {
		t.Logf("  Updated: %s", project.Updated.Format(time.RFC3339))
	}
}

// TestSonarQubeEntity_GetRules tests the GetRules method.
func TestSonarQubeEntity_GetRules(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the request path
		if r.URL.Path != "/api/rules/search" {
			t.Errorf("Expected request to '/api/rules/search', got '%s'", r.URL.Path)
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		// Check the method
		if r.Method != expectedMethod {
			t.Errorf("Expected GET request, got '%s'", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check the authorization header
		auth := r.Header.Get("Authorization")

		if auth != expectedAuth {
			t.Errorf("Expected Authorization header '%s', got '%s'", expectedAuth, auth)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Return a successful response with some rules
		response := map[string]interface{}{
			"rules": []map[string]interface{}{
				{
					"key":      "rule-1",
					"name":     "Rule 1",
					"severity": "MAJOR",
					"lang":     "go",
					"repo":     "common-go",
					"type":     "BUG",
					"tags":     []string{"bug", "critical"},
				},
				{
					"key":      "rule-2",
					"name":     "Rule 2",
					"severity": "MINOR",
					"lang":     "java",
					"repo":     "common-java",
					"type":     "CODE_SMELL",
					"tags":     []string{"code-smell", "maintainability"},
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Logf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.SonarQubeConfig{
		URL:     server.URL,
		Token:   "test-token",
		Timeout: 30 * time.Second,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewEntity(cfg, logger)

	rules, err := entity.GetRules(context.Background(), nil)
	assert.NoError(t, err)
	assert.Len(t, rules, 2)

	// Check the first rule
	assert.Equal(t, "rule-1", rules[0].Key)
	assert.Equal(t, "Rule 1", rules[0].Name)
	assert.Equal(t, "MAJOR", rules[0].Severity)
	assert.Equal(t, "go", rules[0].Language)
	assert.Equal(t, "common-go", rules[0].Repository)
	assert.Equal(t, "BUG", rules[0].Type)
	assert.Equal(t, []string{"bug", "critical"}, rules[0].Tags)

	// Check the second rule
	assert.Equal(t, "rule-2", rules[1].Key)
	assert.Equal(t, "Rule 2", rules[1].Name)
	assert.Equal(t, "MINOR", rules[1].Severity)
	assert.Equal(t, "java", rules[1].Language)
	assert.Equal(t, "common-java", rules[1].Repository)
	assert.Equal(t, "CODE_SMELL", rules[1].Type)
	assert.Equal(t, []string{"code-smell", "maintainability"}, rules[1].Tags)
}

// TestSonarQubeEntity_SetProjectTags tests the SetProjectTags method.
func TestSonarQubeEntity_SetProjectTags(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/project_tags/set", r.URL.Path)

		// Verify authentication
		auth := r.Header.Get("Authorization")
		assert.Equal(t, expectedAuth, auth)

		// Parse form data
		err := r.ParseForm()
		assert.NoError(t, err)

		// Verify form parameters
		assert.Equal(t, "test-project", r.FormValue("project"))
		assert.Equal(t, "finance,offshore", r.FormValue("tags"))

		// Return success response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create entity with test server URL
	cfg := &config.SonarQubeConfig{
		URL:   server.URL,
		Token: "test-token",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := NewEntity(cfg, logger)

	// Test SetProjectTags
	tags := []string{"finance", "offshore"}
	err := entity.SetProjectTags(context.Background(), "test-project", tags)

	// Verify result
	assert.NoError(t, err)
}
