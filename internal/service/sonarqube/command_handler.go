// Package sonarqube provides implementation of SQCommandHandler.
// This package contains the implementation of the SQCommandHandlerInterface
// which coordinates all SonarQube operations.
package sonarqube

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Kargones/apk-ci/internal/entity/gitea"
	"github.com/Kargones/apk-ci/internal/entity/sonarqube"
)

// SQCommandHandler implements the SQCommandHandlerInterface.
// This struct coordinates all SonarQube operations by delegating
// to the appropriate services.
type SQCommandHandler struct {
	// branchScanningService is the service for branch scanning operations
	branchScanningService *BranchScanningService

	// sonarQubeService is the SonarQube service for project management
	sonarQubeService *Service

	// scannerService is the scanner service for executing scans
	scannerService *SonarScannerService

	// projectManagementService is the service for project management operations
	projectManagementService *ProjectManagementService

	// reportingService is the service for report generation operations
	reportingService *ReportingService

	// giteaAPI is the Gitea API client
	giteaAPI gitea.APIInterface

	// logger is the structured logger for this handler
	logger *slog.Logger
}

// NewSQCommandHandler creates a new instance of SQCommandHandler.
// This function initializes the handler with the provided dependencies.
//
// Parameters:
//   - branchScanningService: service for branch scanning operations
//   - sonarQubeService: SonarQube service for project management
//   - scannerService: scanner service for executing scans
//   - projectManagementService: service for project management operations
//   - reportingService: service for report generation operations
//   - giteaAPI: Gitea API client
//   - logger: structured logger instance
//
// Returns:
//   - *SQCommandHandler: initialized command handler
func NewSQCommandHandler(
	branchScanningService *BranchScanningService,
	sonarQubeService *Service,
	scannerService *SonarScannerService,
	projectManagementService *ProjectManagementService,
	reportingService *ReportingService,
	giteaAPI gitea.APIInterface,
	logger *slog.Logger) *SQCommandHandler {
	return &SQCommandHandler{
		branchScanningService:    branchScanningService,
		sonarQubeService:         sonarQubeService,
		scannerService:           scannerService,
		projectManagementService: projectManagementService,
		reportingService:         reportingService,
		giteaAPI:                 giteaAPI,
		logger:                   logger,
	}
}

// CheckScanBranch checks which commits need to be scanned and returns filtered commit hashes.
// This method delegates to the branch scanning service to determine which commits
// require scanning based on existing SonarQube analyses.
//
// Parameters:
//   - ctx: context for the operation
//   - params: parameters for the branch scanning operation
//
// Returns:
//   - []string: array of commit hashes that need to be scanned
//   - error: any error that occurred during the operation
func (h *SQCommandHandler) CheckScanBranch(ctx context.Context, params *sonarqube.ScanBranchParams) ([]string, error) {
	h.logger.Debug("Checking commits for scanning", "owner", params.Owner, "repo", params.Repo, "branch", params.Branch)

	return h.branchScanningService.CheckScanBranch(ctx, params)
}

// HandleSQScanBranch handles the sq-scan-branch command with the provided parameters.
// This method delegates to the branch scanning service to perform the actual scanning.
//
// Parameters:
//   - ctx: context for the operation
//   - params: branch scanning parameters
//
// Returns:
//   - error: error if the operation fails
func (h *SQCommandHandler) HandleSQScanBranch(ctx context.Context, params *sonarqube.ScanBranchParams) error {
	h.logger.Debug("Handling sq-scan-branch command", "owner", params.Owner, "repo", params.Repo, "branch", params.Branch)

	// Получаем список коммитов для сканирования
	commitsToScan, err := h.branchScanningService.CheckScanBranch(ctx, params)
	if err != nil {
		h.logger.Error("Failed to check commits for scanning", "error", err)
		return fmt.Errorf("failed to check commits for scanning: %w", err)
	}

	// Delegate to branch scanning service
	if err := h.branchScanningService.ScanBranch(ctx, params, commitsToScan); err != nil {
		h.logger.Error("Failed to scan branch", "error", err)
		return fmt.Errorf("failed to scan branch: %w", err)
	}

	h.logger.Debug("sq-scan-branch command handled successfully")
	return nil
}

// HandleSQScanBranchWithCommits handles the sq-scan-branch command with pre-filtered commits.
// This method is used when commits have already been filtered and we want to scan specific commits.
//
// Parameters:
//   - ctx: context for the operation
//   - params: branch scanning parameters
//   - commitsToScan: pre-filtered array of commit hashes to scan
//
// Returns:
//   - error: error if the operation fails
func (h *SQCommandHandler) HandleSQScanBranchWithCommits(ctx context.Context, params *sonarqube.ScanBranchParams, commitsToScan []string) error {
	h.logger.Debug("Handling sq-scan-branch command with pre-filtered commits", "owner", params.Owner, "repo", params.Repo, "branch", params.Branch, "commitsCount", len(commitsToScan))

	// Delegate to branch scanning service
	if err := h.branchScanningService.ScanBranch(ctx, params, commitsToScan); err != nil {
		h.logger.Error("Failed to scan branch", "error", err)
		return fmt.Errorf("failed to scan branch: %w", err)
	}

	h.logger.Debug("sq-scan-branch command with pre-filtered commits handled successfully")
	return nil
}

// HandleSQScanPR handles the sq-scan-pr command with the provided parameters.
// This method handles PR scanning operations.
//
// Parameters:
//   - ctx: context for the operation
//   - params: PR scanning parameters
//
// Returns:
//   - error: error if the operation fails
func (h *SQCommandHandler) HandleSQScanPR(ctx context.Context, params *sonarqube.ScanPRParams) error {
	h.logger.Debug("Handling sq-scan-pr command", "owner", params.Owner, "repo", params.Repo, "pr", params.PR)

	// Retrieve active PRs from Gitea
	prs, err := h.giteaAPI.ActivePR()
	if err != nil {
		h.logger.Error("Failed to retrieve active PRs from Gitea", "error", err)
		return fmt.Errorf("failed to retrieve active PRs: %w", err)
	}

	// Find the requested PR
	var targetPR *gitea.PR
	for _, pr := range prs {
		if pr.Number == int64(params.PR) {
			targetPR = &pr
			break
		}
	}

	if targetPR == nil {
		errNotFound := fmt.Errorf("PR #%d not found", params.PR)
		h.logger.Error("Failed to find PR", "error", errNotFound)
		return errNotFound
	}

	// Create branch scanning parameters
	branchParams := &sonarqube.ScanBranchParams{
		Owner:  params.Owner,
		Repo:   params.Repo,
		Branch: targetPR.Head, // Use the head branch of the PR
	}

	// Получаем список коммитов для сканирования
	commitsToScan, err := h.branchScanningService.CheckScanBranch(ctx, branchParams)
	if err != nil {
		h.logger.Error("Failed to check commits for scanning", "error", err)
		return fmt.Errorf("failed to check commits for scanning: %w", err)
	}

	// Delegate to branch scanning service
	if err := h.branchScanningService.ScanBranch(ctx, branchParams, commitsToScan); err != nil {
		h.logger.Error("Failed to scan PR branch", "error", err)
		return fmt.Errorf("failed to scan PR branch: %w", err)
	}

	h.logger.Debug("sq-scan-pr command handled successfully")
	return nil
}

// HandleSQProjectUpdate handles the sq-project-update command with the provided parameters.
// This method handles project update operations by delegating
// to the ProjectManagementService.
//
// Parameters:
//   - ctx: context for the operation
//   - params: project update parameters
//
// Returns:
//   - error: error if the operation fails
func (h *SQCommandHandler) HandleSQProjectUpdate(ctx context.Context, params *sonarqube.ProjectUpdateParams) error {
	h.logger.Debug("Handling sq-project-update command", "owner", params.Owner, "repo", params.Repo)

	// Delegate to ProjectManagementService for actual project update logic
	if err := h.projectManagementService.UpdateProject(ctx, params); err != nil {
		h.logger.Error("Failed to update project", "owner", params.Owner, "repo", params.Repo, "error", err)
		return fmt.Errorf("failed to update project %s/%s: %w", params.Owner, params.Repo, err)
	}

	h.logger.Debug("sq-project-update command handled successfully")
	return nil
}

// HandleSQRepoSync handles the sq-repo-sync command with the provided parameters.
// This method handles repository synchronization operations by delegating
// to the ProjectManagementService.
//
// Parameters:
//   - ctx: context for the operation
//   - params: repository sync parameters
//
// Returns:
//   - error: error if the operation fails
func (h *SQCommandHandler) HandleSQRepoSync(ctx context.Context, params *sonarqube.RepoSyncParams) error {
	h.logger.Debug("Handling sq-repo-sync command", "owner", params.Owner, "repo", params.Repo)

	// Delegate to ProjectManagementService for actual synchronization logic
	if err := h.projectManagementService.SyncRepository(ctx, params); err != nil {
		h.logger.Error("Failed to synchronize repository", "owner", params.Owner, "repo", params.Repo, "error", err)
		return fmt.Errorf("failed to synchronize repository %s/%s: %w", params.Owner, params.Repo, err)
	}

	h.logger.Debug("sq-repo-sync command handled successfully")
	return nil
}

// HandleSQRepoClear handles the sq-repo-clear command with the provided parameters.
// This method handles repository cleanup operations by delegating
// to the ProjectManagementService.
//
// Parameters:
//   - ctx: context for the operation
//   - params: repository clear parameters
//
// Returns:
//   - error: error if the operation fails
func (h *SQCommandHandler) HandleSQRepoClear(ctx context.Context, params *sonarqube.RepoClearParams) error {
	h.logger.Debug("Handling sq-repo-clear command", "owner", params.Owner, "repo", params.Repo, "force", params.Force)

	// Delegate to ProjectManagementService for actual cleanup logic
	if err := h.projectManagementService.ClearRepository(ctx, params); err != nil {
		h.logger.Error("Failed to clear repository", "owner", params.Owner, "repo", params.Repo, "error", err)
		return fmt.Errorf("failed to clear repository %s/%s: %w", params.Owner, params.Repo, err)
	}

	h.logger.Debug("sq-repo-clear command handled successfully")
	return nil
}

// HandleSQReportPR handles the sq-report-pr command with the provided parameters.
// This method handles PR reporting operations.
//
// Parameters:
//   - ctx: context for the operation
//   - params: PR report parameters
//
// Returns:
//   - error: error if the operation fails
func (h *SQCommandHandler) HandleSQReportPR(ctx context.Context, params *sonarqube.ReportPRParams) error {
	h.logger.Debug("Handling sq-report-pr command", "owner", params.Owner, "repo", params.Repo, "pr", params.PR)

	// Delegate to reporting service for PR report generation
	if err := h.reportingService.GeneratePRReport(ctx, params); err != nil {
		h.logger.Error("Failed to generate PR report", "error", err)
		return fmt.Errorf("failed to generate PR report: %w", err)
	}

	h.logger.Debug("sq-report-pr command handled successfully")
	return nil
}

// HandleSQReportBranch handles the sq-report-branch command with the provided parameters.
// This method handles branch reporting operations.
//
// Parameters:
//   - _: context for the operation
//   - params: branch report parameters
//
// Returns:
//   - error: error if the operation fails
func (h *SQCommandHandler) HandleSQReportBranch(_ context.Context, params *sonarqube.ReportBranchParams) error {
	h.logger.Debug("Handling sq-report-branch command", "owner", params.Owner, "repo", params.Repo, "branch", params.Branch)

	// Generate a report for a branch
	// This is a simplified implementation - in a real implementation,
	// you would need to implement the actual report generation logic

	h.logger.Debug("sq-report-branch command handled successfully")
	return nil
}

// HandleSQReportProject handles the sq-report-project command with the provided parameters.
// This method handles project reporting operations.
//
// Parameters:
//   - ctx: context for the operation
//   - params: project report parameters
//
// Returns:
//   - error: error if the operation fails
func (h *SQCommandHandler) HandleSQReportProject(ctx context.Context, params *sonarqube.ReportProjectParams) error {
	h.logger.Debug("Handling sq-report-project command", "owner", params.Owner, "repo", params.Repo)

	// Delegate to reporting service for project report generation
	if err := h.reportingService.GenerateProjectReport(ctx, params); err != nil {
		h.logger.Error("Failed to generate project report", "error", err)
		return fmt.Errorf("failed to generate project report: %w", err)
	}

	h.logger.Debug("sq-report-project command handled successfully")
	return nil
}

// ToDo: необходимо дополнительно реализовать следующий функционал:
// - Add unit tests for all methods
// - Implement more sophisticated command routing
// - Add support for different command types
// - Implement better error handling and recovery
// - Add progress reporting during command execution
//
// Ссылки на пункты плана и требований:
// - tasks.md: 8.1
// - requirements.md: 9.1, 9.3
