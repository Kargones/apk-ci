package sonarqube

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// GetAnalyses retrieves analyses for a project from SonarQube.
// This method retrieves a list of analyses for the specified project.
//
// Parameters:
//   - ctx: context for the request
//   - projectKey: key of the project
//
// Returns:
//   - []Analysis: list of analyses
//   - error: error if analysis retrieval fails
func (s *Entity) GetAnalyses(ctx context.Context, projectKey string) ([]Analysis, error) {
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
//   - ctx: context for the request
//   - analysisID: ID of the analysis
//
// Returns:
//   - *AnalysisStatus: analysis status
//   - error: error if status retrieval fails
func (s *Entity) GetAnalysisStatus(ctx context.Context, analysisID string) (*AnalysisStatus, error) {
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
//   - ctx: context for the request
//   - projectKey: key of the project
//   - params: issue parameters
//
// Returns:
//   - []Issue: list of issues
//   - error: error if issue retrieval fails
func (s *Entity) GetIssues(ctx context.Context, projectKey string, params *IssueParams) ([]Issue, error) {
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
//   - ctx: context for the request
//   - projectKey: key of the project
//
// Returns:
//   - *QualityGateStatus: quality gate status
//   - error: error if status retrieval fails
func (s *Entity) GetQualityGateStatus(ctx context.Context, projectKey string) (*QualityGateStatus, error) {
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
//   - ctx: context for the request
//   - projectKey: key of the project
//   - metricKeys: list of metric keys
//
// Returns:
//   - *Metrics: project metrics
//   - error: error if metrics retrieval fails
func (s *Entity) GetMetrics(ctx context.Context, projectKey string, metricKeys []string) (*Metrics, error) {
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
//   - ctx: context for the request
//   - projectKey: key of the project
//
// Returns:
//   - []QualityProfile: list of quality profiles
//   - error: error if profile retrieval fails
func (s *Entity) GetQualityProfiles(ctx context.Context, projectKey string) ([]QualityProfile, error) {
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
// Parameters:
//   - ctx: context for the request
//
// Returns:
//   - []QualityGate: list of quality gates
//   - error: error if gate retrieval fails
func (s *Entity) GetQualityGates(ctx context.Context) ([]QualityGate, error) {
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
//   - ctx: context for the request
//   - params: rule parameters
//
// Returns:
//   - []Rule: list of rules
//   - error: error if rule retrieval fails
func (s *Entity) GetRules(ctx context.Context, params *RuleParams) ([]Rule, error) {
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
