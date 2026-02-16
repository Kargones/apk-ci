// Package sonarqube provides implementation of pull request scanning functionality.
// This package contains the business logic for scanning pull requests,
// including PR data retrieval and delegation to branch scanning.
package sonarqube

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/entity/gitea"
	"github.com/Kargones/apk-ci/internal/entity/sonarqube"
)

// PRScanningService provides business logic for pull request scanning operations.
// This service layer implements the logic for scanning pull requests,
// including PR data retrieval and delegation to branch scanning.
type PRScanningService struct {
	// branchScanningService is the service for branch scanning operations
	branchScanningService *BranchScanningService
	
	// giteaAPI is the Gitea API client for retrieving PR data
	giteaAPI gitea.APIInterface
	
	// logger is the structured logger for this service
	logger *slog.Logger
	
	// config is the application configuration
	config *config.Config
}

// NewPRScanningService creates a new instance of PRScanningService.
// This function initializes the service with the provided dependencies.
//
// Parameters:
//   - branchScanningService: service for branch scanning operations
//   - giteaAPI: Gitea API client for retrieving PR data
//   - logger: structured logger instance
//
// Returns:
//   - *PRScanningService: initialized PR scanning service
func NewPRScanningService(
	branchScanningService *BranchScanningService,
	giteaAPI gitea.APIInterface,
	logger *slog.Logger,
	config *config.Config) *PRScanningService {
	return &PRScanningService{
		branchScanningService: branchScanningService,
		giteaAPI:              giteaAPI,
		logger:                logger,
		config:                config,
	}
}

// ScanPR scans a pull request with the provided parameters.
// This method implements the logic for scanning a pull request,
// including PR data retrieval and delegation to branch scanning.
//
// Parameters:
//   - ctx: context for the operation
//   - params: PR scanning parameters
//
// Returns:
//   - error: error if scanning fails
func (p *PRScanningService) ScanPR(ctx context.Context, params *sonarqube.ScanPRParams) error {
	p.logger.Debug("Scanning pull request", "owner", params.Owner, "repo", params.Repo, "pr", params.PR)
	
	// Get active PRs from Gitea
	prs, err := p.giteaAPI.ActivePR()
	if err != nil {
		p.logger.Error("Failed to retrieve active PRs from Gitea", "error", err)
		return fmt.Errorf("failed to retrieve active PRs: %w", err)
	}
	
	// Find the PR with the specified number
	var targetPR *gitea.PR
	for _, pr := range prs {
		if pr.Number == int64(params.PR) {
			targetPR = &pr
			break
		}
	}
	
	// If PR not found, return an error
	if targetPR == nil {
		p.logger.Error("PR not found", "pr", params.PR)
		return fmt.Errorf("PR %d not found", params.PR)
	}
	
	// Extract source branch information from PR data
	// Head field contains the source branch name
	sourceBranch := targetPR.Head
	
	// Create branch scanning parameters
	branchParams := &sonarqube.ScanBranchParams{
		Owner:     params.Owner,
		Repo:      params.Repo,
		Branch:    sourceBranch,
		SourceDir: p.config.WorkDir,
	}
	
	// Получаем список коммитов для сканирования
	commitsToScan, err := p.branchScanningService.CheckScanBranch(ctx, branchParams)
	if err != nil {
		p.logger.Error("Failed to check commits for scanning", "error", err)
		return fmt.Errorf("failed to check commits for scanning: %w", err)
	}

	// Delegate to branch scanning service for actual scanning
	if err := p.branchScanningService.ScanBranch(ctx, branchParams, commitsToScan); err != nil {
		p.logger.Error("Failed to scan source branch", "error", err)
		return fmt.Errorf("failed to scan source branch: %w", err)
	}
	
	p.logger.Debug("Pull request scanned successfully")
	return nil
}

// ToDo: необходимо дополнительно реализовать следующий функционал:
// - Add unit tests for all methods
// - Implement better error handling and recovery
// - Add progress reporting during scanning
//
// Ссылки на пункты плана и требований:
// - tasks.md: 5.1
// - requirements.md: 2.1, 2.2, 2.3