// Package sonarqube provides implementation of SonarQube entity.
// This package contains the low-level implementation for interacting with SonarQube API,
// including HTTP client configuration, authentication, and basic API methods.
package sonarqube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/Kargones/apk-ci/internal/config"
)

// executeRequest выполняет HTTP-запрос без механизма повторных попыток
func (s *Entity) executeRequest(_ context.Context, req *http.Request) ([]byte, error) {
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if errBody := resp.Body.Close(); errBody != nil {
			s.logger.Error("Failed to close response body", "error", errBody)
		}
	}()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return respBody, nil
	}

	// Handle API errors
	if resp.StatusCode >= 400 {
		// Try to parse error response
		var apiErr struct {
			Errors []struct {
				Msg string `json:"msg"`
			} `json:"errors"`
		}

		if parseErr := json.Unmarshal(respBody, &apiErr); parseErr == nil && len(apiErr.Errors) > 0 {
			return nil, &Error{
				Code:    resp.StatusCode,
				Message: "SonarQube API error",
				Details: apiErr.Errors[0].Msg,
			}
		}

		// If we can't parse the error response, return a generic error
		return nil, &Error{
			Code:    resp.StatusCode,
			Message: fmt.Sprintf("SonarQube API error: %s", resp.Status),
		}
	}

	return respBody, nil
}

// makeFormRequest performs an HTTP request to SonarQube API with form data.
// This method handles the common logic for making HTTP requests to SonarQube API
// with form-encoded data, including authentication, request execution, and error handling.
//
// Parameters:
//   - ctx: context for the request
//   - method: HTTP method (GET, POST, PUT, DELETE)
//   - endpoint: API endpoint path
//   - formData: form data values
//
// Returns:
//   - []byte: response body
//   - error: error if request fails
func (s *Entity) makeFormRequest(ctx context.Context, method, endpoint string, formData url.Values) ([]byte, error) {
	// Construct full URL
	url := s.config.URL + "/api" + endpoint

	// Prepare form data body
	var bodyReader io.Reader
	if formData != nil {
		bodyReader = strings.NewReader(formData.Encode())
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set content type for form data
	if formData != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	// Authenticate request
	if authErr := s.authenticate(req); authErr != nil {
		return nil, fmt.Errorf("authentication failed: %w", authErr)
	}

	// Execute request with retry mechanism
	return s.executeRequest(ctx, req)
}

// Entity represents the low-level interaction with SonarQube API.
// This struct contains the HTTP client configuration and implements basic
// methods for interacting with SonarQube REST API.
type Entity struct {
	// client is the HTTP client used for making requests to SonarQube API.
	client *http.Client

	// config contains the SonarQube configuration settings.
	config *config.SonarQubeConfig

	// logger is the structured logger for this entity.
	logger *slog.Logger
}

// NewEntity creates a new instance of Entity.
// This function initializes the HTTP client with appropriate timeouts and
// configures the entity with the provided SonarQube configuration.
//
// Parameters:
//   - cfg: SonarQube configuration settings
//   - logger: structured logger instance
//
// Returns:
//   - *Entity: initialized SonarQube entity
func NewEntity(cfg *config.SonarQubeConfig, logger *slog.Logger) *Entity {
	// Create HTTP client with timeout from config
	client := &http.Client{
		Timeout: cfg.Timeout,
	}

	return &Entity{
		client: client,
		config: cfg,
		logger: logger,
	}
}

// Authenticate authenticates with SonarQube using the provided token.
// This method validates the token by making a simple API request.
//
// Parameters:
//   - token: authentication token
//
// Returns:
//   - error: error if authentication fails
func (s *Entity) Authenticate(token string) error {
	// Temporarily set the token for validation
	originalToken := s.config.Token
	s.config.Token = token
	defer func() {
		s.config.Token = originalToken
	}()

	// Validate the token
	return s.ValidateToken()
}

// authenticate adds authentication header to the request.
// This method adds the Authorization header with the Bearer token
// from the configuration to the provided HTTP request.
//
// Parameters:
//   - req: HTTP request to authenticate
//
// Returns:
//   - error: error if authentication fails
func (s *Entity) authenticate(req *http.Request) error {
	if s.config.Token == "" {
		return &ValidationError{
			Field:   "token",
			Message: "SonarQube token is not configured",
		}
	}

	// Add Authorization header with Bearer token
	req.Header.Set("Authorization", "Bearer "+s.config.Token)
	return nil
}

// makeRequest performs an HTTP request to SonarQube API.
// This method handles the common logic for making HTTP requests to SonarQube API,
// including authentication, request execution, and error handling.
//
// Parameters:
//   - ctx: context for the request
//   - method: HTTP method (GET, POST, PUT, DELETE)
//   - endpoint: API endpoint path
//   - body: request body (can be nil)
//
// Returns:
//   - []byte: response body
//   - error: error if request fails
func (s *Entity) makeRequest(ctx context.Context, method, endpoint string, body interface{}) ([]byte, error) {
	// Construct full URL
	url := s.config.URL + "/api" + endpoint

	// Serialize request body if provided
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set content type for requests with body
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Authenticate request
	if authErr := s.authenticate(req); authErr != nil {
		return nil, fmt.Errorf("authentication failed: %w", authErr)
	}

	// Execute request with retry mechanism
	return s.executeRequest(ctx, req)
}

// ValidateToken validates the configured authentication token.
// This method checks if the configured token is valid by making a simple
// API request to SonarQube.
//
// Returns:
//   - error: error if token is invalid or validation fails
func (s *Entity) ValidateToken() error {
	ctx := context.Background()

	// Make a simple request to validate the token
	_, err := s.makeRequest(ctx, "GET", "/authentication/validate", nil)
	if err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}

	return nil
}

// CreateProject creates a new project in SonarQube.
// This method creates a new project with the specified owner, repo, and branch.
//
// Parameters:
//   - owner: project owner
//   - repo: project repository
//   - branch: project branch
//
// Returns:
//   - *Project: created project
//   - error: error if project creation fails
func (s *Entity) CreateProject(owner, repo, branch string) (*Project, error) {
	ctx := context.Background()

	// Generate project key based on owner, repo, and branch
	projectKey := fmt.Sprintf("%s_%s_%s", owner, repo, branch)
	if s.config.ProjectPrefix != "" {
		projectKey = s.config.ProjectPrefix + "_" + projectKey
	}

	// Generate project name
	projectName := fmt.Sprintf("%s/%s (%s)", owner, repo, branch)

	// Prepare form data
	formData := url.Values{}
	formData.Set("name", projectName)
	formData.Set("project", projectKey)
	formData.Set("visibility", s.config.DefaultVisibility)

	// Make API request to create project
	respBody, err := s.makeFormRequest(ctx, "POST", "/projects/create", formData)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	// Parse response
	var resp struct {
		Project Project `json:"project"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse project creation response: %w", err)
	}

	return &resp.Project, nil
}

// GetProject retrieves a project from SonarQube by its project key.
// This method retrieves project information from the API.
//
// Parameters:
//   - projectKey: key of the project to retrieve
//
// Returns:
//   - *Project: retrieved project
//   - error: error if project retrieval fails
func (s *Entity) GetProject(projectKey string) (*Project, error) {
	ctx := context.Background()

	// Prepare query parameters
	params := url.Values{}
	params.Add("projects", projectKey)
	params.Add("ps", "1") // Page size 1 since we're looking for specific project

	// Make API request to get project using projects/search endpoint
	respBody, err := s.makeRequest(ctx, "GET", "/projects/search?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Parse response
	var resp struct {
		Components []Project `json:"components"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse project retrieval response: %w", err)
	}

	// Check if project was found
	if len(resp.Components) == 0 {
		return nil, &Error{
			Code:    404,
			Message: "Project not found",
			Details: fmt.Sprintf("Project with key '%s' was not found", projectKey),
		}
	}

	// Return the first (and should be only) project
	return &resp.Components[0], nil
}

// UpdateProject updates an existing project in SonarQube.
// This method updates an existing project with the provided updates.
//
// Parameters:
//   - projectKey: key of the project to update
//   - updates: project updates
//
// Returns:
//   - error: error if project update fails
func (s *Entity) UpdateProject(projectKey string, updates *ProjectUpdate) error {
	ctx := context.Background()

	// Prepare form data
	formData := url.Values{}
	formData.Set("project", projectKey)

	// Add optional fields to form data
	if updates.Name != "" {
		formData.Set("name", updates.Name)
	}

	if updates.Description != "" {
		formData.Set("description", updates.Description)
	}

	if updates.Visibility != "" {
		formData.Set("visibility", updates.Visibility)
	}

	// Make API request to update project
	_, err := s.makeFormRequest(ctx, "POST", "/projects/update", formData)
	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	// Set project tags if provided
	if len(updates.Tags) > 0 {
		if err := s.SetProjectTags(projectKey, updates.Tags); err != nil {
			return fmt.Errorf("failed to set project tags during update: %w", err)
		}
	}

	return nil
}

// DeleteProject deletes a project from SonarQube by its project key.
// This method deletes a project from the API.
//
// Parameters:
//   - projectKey: key of the project to delete
//
// Returns:
//   - error: error if project deletion fails
func (s *Entity) DeleteProject(projectKey string) error {
	ctx := context.Background()

	// Prepare form data
	formData := url.Values{}
	formData.Set("project", projectKey)

	// Make API request to delete project
	_, err := s.makeFormRequest(ctx, "POST", "/projects/delete", formData)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	return nil
}

// ListProjects lists all projects in SonarQube that match the specified owner and repo.
// This method retrieves a list of projects from the API.
//
// Parameters:
//   - owner: project owner
//   - repo: project repository
//
// Returns:
//   - []Project: list of projects
//   - error: error if project listing fails
func (s *Entity) ListProjects(owner, repo string) ([]Project, error) {
	ctx := context.Background()

	// Prepare query parameters
	params := url.Values{}
	searchQuery := fmt.Sprintf("%s_%s_", owner, repo)
	if s.config.ProjectPrefix != "" {
		searchQuery = s.config.ProjectPrefix + "_" + searchQuery
	}
	params.Add("q", searchQuery)

	// Make API request to list projects
	respBody, err := s.makeRequest(ctx, "GET", "/projects/search?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	// Parse response
	var resp struct {
		Components []Project `json:"components"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse project listing response: %w", err)
	}

	// Filter projects that match the exact owner/repo pattern
	var projects []Project
	prefix := fmt.Sprintf("%s_%s_%s_", s.config.ProjectPrefix, owner, repo)
	if s.config.ProjectPrefix == "" {
		prefix = fmt.Sprintf("%s_%s_", owner, repo)
	}

	for _, project := range resp.Components {
		if strings.HasPrefix(project.Key, prefix) {
			projects = append(projects, project)
		}
	}

	return projects, nil
}

// GetAnalyses retrieves analyses for a project from SonarQube.
// This method retrieves a list of analyses for the specified project.
//
// Parameters:
//   - projectKey: key of the project
//
// Returns:
//   - []Analysis: list of analyses
//   - error: error if analysis retrieval fails
func (s *Entity) GetAnalyses(projectKey string) ([]Analysis, error) {
	ctx := context.Background()

	// Prepare query parameters
	params := url.Values{}
	params.Add("project", projectKey)

	// Make API request to get analyses
	respBody, err := s.makeRequest(ctx, "GET", "/project_analyses/search?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get analyses: %w", err)
	}

	// Parse response
	var resp struct {
		Analyses []Analysis `json:"analyses"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse analyses retrieval response: %w", err)
	}

	return resp.Analyses, nil
}

// GetAnalysisStatus retrieves the status of an analysis by its ID.
// This method retrieves the status of the specified analysis.
//
// Parameters:
//   - analysisID: ID of the analysis
//
// Returns:
//   - *AnalysisStatus: analysis status
//   - error: error if status retrieval fails
func (s *Entity) GetAnalysisStatus(analysisID string) (*AnalysisStatus, error) {
	ctx := context.Background()

	// Prepare query parameters
	params := url.Values{}
	params.Add("analysisId", analysisID)

	// Make API request to get analysis status
	respBody, err := s.makeRequest(ctx, "GET", "/ce/task?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get analysis status: %w", err)
	}

	// Parse response
	var resp struct {
		Task struct {
			Status string `json:"status"`
		} `json:"task"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse analysis status response: %w", err)
	}

	status := &AnalysisStatus{
		Status: resp.Task.Status,
	}

	return status, nil
}

// GetIssues retrieves issues for a project from SonarQube.
// This method retrieves a list of issues for the specified project.
//
// Parameters:
//   - projectKey: key of the project
//   - params: issue parameters
//
// Returns:
//   - []Issue: list of issues
//   - error: error if issue retrieval fails
func (s *Entity) GetIssues(projectKey string, params *IssueParams) ([]Issue, error) {
	ctx := context.Background()

	// Prepare query parameters
	queryParams := url.Values{}
	queryParams.Add("componentKeys", projectKey)

	// Add optional parameters
	if params != nil {
		if len(params.ComponentKeys) > 0 {
			queryParams.Add("componentKeys", strings.Join(params.ComponentKeys, ","))
		}

		if len(params.Rules) > 0 {
			queryParams.Add("rules", strings.Join(params.Rules, ","))
		}

		if len(params.Severities) > 0 {
			queryParams.Add("severities", strings.Join(params.Severities, ","))
		}

		if len(params.Statuses) > 0 {
			queryParams.Add("statuses", strings.Join(params.Statuses, ","))
		}
	}

	// Make API request to get issues
	respBody, err := s.makeRequest(ctx, "GET", "/issues/search?"+queryParams.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get issues: %w", err)
	}

	// Parse response
	var resp struct {
		Issues []Issue `json:"issues"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse issues retrieval response: %w", err)
	}

	return resp.Issues, nil
}

// GetQualityGateStatus retrieves the quality gate status for a project.
// This method retrieves the quality gate status for the specified project.
//
// Parameters:
//   - projectKey: key of the project
//
// Returns:
//   - *QualityGateStatus: quality gate status
//   - error: error if status retrieval fails
func (s *Entity) GetQualityGateStatus(projectKey string) (*QualityGateStatus, error) {
	ctx := context.Background()

	// Prepare query parameters
	params := url.Values{}
	params.Add("projectKey", projectKey)

	// Make API request to get quality gate status
	respBody, err := s.makeRequest(ctx, "GET", "/qualitygates/project_status?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get quality gate status: %w", err)
	}

	// Parse response
	var resp struct {
		ProjectStatus struct {
			Status string `json:"status"`
		} `json:"projectStatus"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse quality gate status response: %w", err)
	}

	status := &QualityGateStatus{
		Status: resp.ProjectStatus.Status,
	}

	return status, nil
}

// GetMetrics retrieves metrics for a project based on the specified metric keys.
// This method retrieves metrics for the specified project.
//
// Parameters:
//   - projectKey: key of the project
//   - metricKeys: list of metric keys
//
// Returns:
//   - *Metrics: project metrics
//   - error: error if metrics retrieval fails
func (s *Entity) GetMetrics(projectKey string, metricKeys []string) (*Metrics, error) {
	ctx := context.Background()

	// Prepare query parameters
	params := url.Values{}
	params.Add("component", projectKey)
	params.Add("metricKeys", strings.Join(metricKeys, ","))

	// Make API request to get metrics
	respBody, err := s.makeRequest(ctx, "GET", "/measures/component?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	// Parse response
	var resp struct {
		Component struct {
			Measures []struct {
				Metric string `json:"metric"`
				Value  string `json:"value"`
			} `json:"measures"`
		} `json:"component"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse metrics retrieval response: %w", err)
	}

	// Convert measures to metrics map
	metrics := make(map[string]float64)
	for _, measure := range resp.Component.Measures {
		var value float64
		if _, err := fmt.Sscanf(measure.Value, "%f", &value); err != nil {
			s.logger.Warn("Failed to parse metric value", "metric", measure.Metric, "value", measure.Value, "error", err)
			continue
		}
		metrics[measure.Metric] = value
	}

	return &Metrics{Metrics: metrics}, nil
}

// GetQualityProfiles retrieves quality profiles for a project from SonarQube.
// This method retrieves a list of quality profiles for the specified project.
//
// Parameters:
//   - projectKey: key of the project
//
// Returns:
//   - []QualityProfile: list of quality profiles
//   - error: error if profile retrieval fails
func (s *Entity) GetQualityProfiles(projectKey string) ([]QualityProfile, error) {
	ctx := context.Background()

	// Prepare query parameters
	params := url.Values{}
	params.Add("project", projectKey)

	// Make API request to get quality profiles
	respBody, err := s.makeRequest(ctx, "GET", "/qualityprofiles/search?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get quality profiles: %w", err)
	}

	// Parse response
	var resp struct {
		Profiles []QualityProfile `json:"profiles"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse quality profiles response: %w", err)
	}

	return resp.Profiles, nil
}

// GetQualityGates retrieves quality gates from SonarQube.
// This method retrieves a list of quality gates from the API.
//
// Returns:
//   - []QualityGate: list of quality gates
//   - error: error if gate retrieval fails
func (s *Entity) GetQualityGates() ([]QualityGate, error) {
	ctx := context.Background()

	// Make API request to get quality gates
	respBody, err := s.makeRequest(ctx, "GET", "/qualitygates/list", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get quality gates: %w", err)
	}

	// Parse response
	var resp struct {
		QualityGates []QualityGate `json:"qualitygates"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse quality gates response: %w", err)
	}

	return resp.QualityGates, nil
}

// GetRules retrieves rules from SonarQube based on the provided parameters.
// This method retrieves a list of rules from the API.
//
// Parameters:
//   - params: rule parameters
//
// Returns:
//   - []Rule: list of rules
//   - error: error if rule retrieval fails
func (s *Entity) GetRules(params *RuleParams) ([]Rule, error) {
	ctx := context.Background()

	// Prepare query parameters
	queryParams := url.Values{}

	// Add optional parameters
	if params != nil {
		if len(params.Repositories) > 0 {
			queryParams.Add("repositories", strings.Join(params.Repositories, ","))
		}

		if len(params.Languages) > 0 {
			queryParams.Add("languages", strings.Join(params.Languages, ","))
		}

		if len(params.Tags) > 0 {
			queryParams.Add("tags", strings.Join(params.Tags, ","))
		}

		if params.IsActive != nil {
			queryParams.Add("isActive", fmt.Sprintf("%t", *params.IsActive))
		}
	}

	// Make API request to get rules
	respBody, err := s.makeRequest(ctx, "GET", "/rules/search?"+queryParams.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules: %w", err)
	}

	// Parse response
	var resp struct {
		Rules []Rule `json:"rules"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse rules response: %w", err)
	}

	return resp.Rules, nil
}

// SetProjectTags sets tags on a project in SonarQube.
// Requires 'Administer' rights on the specified project.
// API: POST api/project_tags/set (since 6.4)
func (s *Entity) SetProjectTags(projectKey string, tags []string) error {
	ctx := context.Background()

	// Prepare form data
	formData := url.Values{}
	formData.Set("project", projectKey)
	formData.Set("tags", strings.Join(tags, ","))

	// Make the request
	_, err := s.makeFormRequest(ctx, "POST", "/project_tags/set", formData)
	if err != nil {
		return fmt.Errorf("failed to set project tags: %w", err)
	}

	s.logger.Info("Project tags set successfully", "project", projectKey, "tags", tags)
	return nil
}

// ToDo: необходимо дополнительно реализовать следующий функционал:
// - Add unit tests for all methods
// - Implement additional API methods as needed
// - Add more detailed error handling and logging
//
// Ссылки на пункты плана и требований:
// - tasks.md: 2.1, 2.2
// - requirements.md: 1.1, 3.1, 3.2, 9.1, 9.2
