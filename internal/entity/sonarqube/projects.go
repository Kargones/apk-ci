package sonarqube

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// CreateProject creates a new project in SonarQube.
// This method creates a new project with the specified owner, repo, and branch.
//
// Parameters:
//   - ctx: context for the request
//   - owner: project owner
//   - repo: project repository
//   - branch: project branch
//
// Returns:
//   - *Project: created project
//   - error: error if project creation fails
func (s *Entity) CreateProject(ctx context.Context, owner, repo, branch string) (*Project, error) {
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
//   - ctx: context for the request
//   - projectKey: key of the project to retrieve
//
// Returns:
//   - *Project: retrieved project
//   - error: error if project retrieval fails
func (s *Entity) GetProject(ctx context.Context, projectKey string) (*Project, error) {
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
//   - ctx: context for the request
//   - projectKey: key of the project to update
//   - updates: project updates
//
// Returns:
//   - error: error if project update fails
func (s *Entity) UpdateProject(ctx context.Context, projectKey string, updates *ProjectUpdate) error {
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
		if err := s.SetProjectTags(ctx, projectKey, updates.Tags); err != nil {
			return fmt.Errorf("failed to set project tags during update: %w", err)
		}
	}

	return nil
}

// DeleteProject deletes a project from SonarQube by its project key.
// This method deletes a project from the API.
//
// Parameters:
//   - ctx: context for the request
//   - projectKey: key of the project to delete
//
// Returns:
//   - error: error if project deletion fails
func (s *Entity) DeleteProject(ctx context.Context, projectKey string) error {
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
//   - ctx: context for the request
//   - owner: project owner
//   - repo: project repository
//
// Returns:
//   - []Project: list of projects
//   - error: error if project listing fails
func (s *Entity) ListProjects(ctx context.Context, owner, repo string) ([]Project, error) {
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

// SetProjectTags sets tags on a project in SonarQube.
// Requires 'Administer' rights on the specified project.
// API: POST api/project_tags/set (since 6.4)
func (s *Entity) SetProjectTags(ctx context.Context, projectKey string, tags []string) error {
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
