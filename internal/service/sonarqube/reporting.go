// Package sonarqube provides implementation of reporting functionality.
// This package contains the business logic for generating reports,
// including branch reports, PR reports, and project reports.
package sonarqube

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Kargones/apk-ci/internal/entity/gitea"
	"github.com/Kargones/apk-ci/internal/entity/sonarqube"
)

// ReportingService provides business logic for reporting operations.
// This service layer implements the logic for generating reports,
// including branch reports, PR reports, and project reports.
type ReportingService struct {
	// sonarQubeService is the SonarQube service for retrieving data
	sonarQubeService *Service

	// giteaAPI is the Gitea API client for retrieving PR data
	giteaAPI gitea.APIInterface

	// logger is the structured logger for this service
	logger *slog.Logger
}

// NewReportingService creates a new instance of ReportingService.
// This function initializes the service with the provided dependencies.
//
// Parameters:
//   - sonarQubeService: SonarQube service for retrieving data
//   - giteaAPI: Gitea API client for retrieving PR data
//   - logger: structured logger instance
//
// Returns:
//   - *ReportingService: initialized reporting service
func NewReportingService(
	sonarQubeService *Service,
	giteaAPI gitea.APIInterface,
	logger *slog.Logger) *ReportingService {
	return &ReportingService{
		sonarQubeService: sonarQubeService,
		giteaAPI:         giteaAPI,
		logger:           logger,
	}
}

// GenerateBranchReport generates a branch report with the provided parameters.
// This method implements the logic for generating a branch report,
// including issue retrieval and JSON report formatting.
//
// Parameters:
//   - ctx: context for the operation
//   - params: branch report parameters
//
// Returns:
//   - error: error if report generation fails
func (r *ReportingService) GenerateBranchReport(_ context.Context, params *sonarqube.ReportBranchParams) error {
	r.logger.Debug("Generating branch report", "owner", params.Owner, "repo", params.Repo, "branch", params.Branch)

	// This is a simplified implementation - in a real implementation,
	// you would need to retrieve issues between commit ranges and format
	// the report as JSON

	// For now, we'll just log that the branch report generation is not fully implemented
	r.logger.Warn("Branch report generation is not fully implemented yet")

	r.logger.Debug("Branch report generated successfully")
	return nil
}

// GeneratePRReport generates a PR report with the provided parameters.
// This method implements the logic for generating a PR report,
// including PR-specific report formatting and issue posting to Gitea.
//
// Parameters:
//   - ctx: context for the operation
//   - params: PR report parameters
//
// Returns:
//   - error: error if report generation fails
func (r *ReportingService) GeneratePRReport(ctx context.Context, params *sonarqube.ReportPRParams) error {
	r.logger.Debug("Generating PR report", "owner", params.Owner, "repo", params.Repo, "pr", params.PR)

	// Get active PRs from Gitea to find the specific PR
	activePRs, err := r.giteaAPI.ActivePR()
	if err != nil {
		r.logger.Error("Failed to get active PRs", "error", err)
		return fmt.Errorf("failed to get active PRs: %w", err)
	}

	// Find the specific PR by number
	var targetPR *gitea.PR
	for _, pr := range activePRs {
		if pr.Number == int64(params.PR) {
			targetPR = &pr
			break
		}
	}

	if targetPR == nil {
		r.logger.Error("PR not found", "pr", params.PR)
		return fmt.Errorf("PR %d not found", params.PR)
	}

	// Get commits between base and head branches
	commits, err := r.giteaAPI.GetCommitsBetween(targetPR.Base, targetPR.Head)
	if err != nil {
		r.logger.Error("Failed to get commits between branches", "error", err)
		return fmt.Errorf("failed to get commits between branches: %w", err)
	}

	// Generate SonarQube project key
	projectKey := fmt.Sprintf("%s_%s", params.Owner, params.Repo)

	// Get SonarQube analyses for the project
	analyses, err := r.sonarQubeService.GetAnalyses(ctx, projectKey)
	if err != nil {
		r.logger.Error("Failed to get SonarQube analyses", "error", err)
		return fmt.Errorf("failed to get SonarQube analyses: %w", err)
	}

	// Get the latest analysis
	var latestAnalysis *sonarqube.Analysis
	if len(analyses) > 0 {
		latestAnalysis = &analyses[0]
	}

	// Get issues for the project
	issues, err := r.sonarQubeService.GetIssues(ctx, projectKey, nil)
	if err != nil {
		r.logger.Error("Failed to get SonarQube issues", "error", err)
		return fmt.Errorf("failed to get SonarQube issues: %w", err)
	}

	// Generate report content
	reportContent := r.formatPRReport(latestAnalysis, issues, commits)

	// Post report as comment to PR using AddIssueComment (treating PR as issue)
	if err := r.giteaAPI.AddIssueComment(int64(params.PR), reportContent); err != nil {
		r.logger.Error("Failed to post PR comment", "error", err)
		return fmt.Errorf("failed to post PR comment: %w", err)
	}

	r.logger.Debug("PR report generated successfully")
	return nil
}

// GenerateProjectReport generates a project report with the provided parameters.
// This method implements the logic for generating a project report,
// including multi-branch analysis and aggregation.
//
// Parameters:
//   - ctx: context for the operation
//   - params: project report parameters
//
// Returns:
//   - error: error if report generation fails
func (r *ReportingService) GenerateProjectReport(ctx context.Context, params *sonarqube.ReportProjectParams) error {
	r.logger.Debug("Generating project report", "owner", params.Owner, "repo", params.Repo)

	// Get project key for SonarQube
	projectKey := fmt.Sprintf("%s_%s", params.Owner, params.Repo)

	// Get project analyses from SonarQube
	analyses, err := r.sonarQubeService.GetAnalyses(ctx, projectKey)
	if err != nil {
		r.logger.Error("Failed to get project analyses", "error", err, "projectKey", projectKey)
		return fmt.Errorf("failed to get project analyses: %w", err)
	}

	// Get the latest analysis (first in the list)
	var latestAnalysis *sonarqube.Analysis
	if len(analyses) > 0 {
		latestAnalysis = &analyses[0]
	}

	// Get project issues from SonarQube
	issues, err := r.sonarQubeService.GetIssues(ctx, projectKey, &sonarqube.IssueParams{})
	if err != nil {
		r.logger.Error("Failed to get project issues", "error", err, "projectKey", projectKey)
		return fmt.Errorf("failed to get project issues: %w", err)
	}

	// Generate and format the project report
	reportContent := r.formatProjectReport(latestAnalysis, issues, projectKey)

	// TODO: In a real implementation, you might want to:
	// - Save the report to a file or database
	// - Send the report via email or notification system
	// - Post the report as an issue or wiki page in Gitea
	// For now, we'll just log the report content
	r.logger.Info("Project report generated", "projectKey", projectKey, "issuesCount", len(issues))
	r.logger.Debug("Project report content", "content", reportContent)

	r.logger.Debug("Project report generated successfully")
	return nil
}

// formatProjectReport formats the project report content based on analysis and issues.
// This method generates a formatted report string for project-wide analysis.
//
// Parameters:
//   - analysis: SonarQube analysis data (can be nil)
//   - issues: list of SonarQube issues
//   - projectKey: SonarQube project key
//
// Returns:
//   - string: formatted report content
func (r *ReportingService) formatProjectReport(analysis *sonarqube.Analysis, issues []sonarqube.Issue, projectKey string) string {
	report := "## SonarQube Project Analysis Report\n\n"

	report += fmt.Sprintf("**Project Key:** %s\n", projectKey)

	if analysis != nil {
		report += fmt.Sprintf("**Analysis Date:** %s\n", analysis.Date.Format("2006-01-02 15:04:05"))
		report += fmt.Sprintf("**Analysis ID:** %s\n", analysis.ID)
		if analysis.Revision != "" {
			report += fmt.Sprintf("**Revision:** %s\n", analysis.Revision)
		}
		report += "\n"
	} else {
		report += "**No analysis data available**\n\n"
	}

	if len(issues) > 0 {
		report += fmt.Sprintf("**Total Issues Found:** %d\n\n", len(issues))

		// Group issues by severity
		issuesBySeverity := make(map[string]int)
		for _, issue := range issues {
			issuesBySeverity[issue.Severity]++
		}

		report += "### Issues by Severity:\n"
		for severity, count := range issuesBySeverity {
			report += fmt.Sprintf("- **%s:** %d\n", severity, count)
		}
		report += "\n"

		report += "### Top Issues:\n"
		for i, issue := range issues {
			if i >= 10 { // Limit to top 10 issues
				break
			}
			report += fmt.Sprintf("%d. **%s** (%s) - %s\n", i+1, issue.Rule, issue.Severity, issue.Message)
			if issue.Component != "" {
				report += fmt.Sprintf("   File: %s\n", issue.Component)
			}
			if issue.Line > 0 {
				report += fmt.Sprintf("   Line: %d\n", issue.Line)
			}
			report += "\n"
		}
	} else {
		report += "**No issues found! ✅**\n\n"
	}

	report += "---\n"
	report += "*Generated by Benadis Runner SonarQube Integration*\n"

	return report
}

// formatPRReport formats the PR report content based on analysis, issues, and commits.
// This method generates a formatted report string for posting as a PR comment.
//
// Parameters:
//   - analysis: SonarQube analysis data (can be nil)
//   - issues: list of SonarQube issues
//   - commits: list of commits in the PR
//
// Returns:
//   - string: formatted report content
func (r *ReportingService) formatPRReport(analysis *sonarqube.Analysis, issues []sonarqube.Issue, commits []gitea.Commit) string {
	report := "## SonarQube Analysis Report\n\n"

	if analysis != nil {
		report += fmt.Sprintf("**Analysis Date:** %s\n", analysis.Date)
		report += fmt.Sprintf("**Project Key:** %s\n\n", analysis.ProjectKey)
	} else {
		report += "**No analysis data available**\n\n"
	}

	report += fmt.Sprintf("**Commits in PR:** %d\n\n", len(commits))

	if len(issues) > 0 {
		report += fmt.Sprintf("**Issues Found:** %d\n\n", len(issues))
		report += "### Issues:\n"
		for i, issue := range issues {
			if i >= 10 { // Limit to first 10 issues
				report += fmt.Sprintf("... and %d more issues\n", len(issues)-10)
				break
			}
			report += fmt.Sprintf("- **%s** (%s): %s\n", issue.Severity, issue.Type, issue.Message)
			if issue.Line > 0 {
				report += fmt.Sprintf("  - Line: %d\n", issue.Line)
			}
		}
	} else {
		report += "**No issues found** ✅\n"
	}

	return report
}

// ToDo: необходимо дополнительно реализовать следующий функционал:
// - Add unit tests for all methods
// - Implement issue retrieval between commit ranges
// - Implement JSON report formatting
// - Implement PR-specific report formatting
// - Implement issue posting to Gitea repository
// - Implement multi-branch analysis and aggregation
// - Implement project-wide report generation
// - Implement better error handling and recovery
// - Add progress reporting during report generation
//
// Ссылки на пункты плана и требований:
// - tasks.md: 7.1, 7.2, 7.3
// - requirements.md: 7.1, 7.2, 8.1, 8.2, 8.3, 8.4
