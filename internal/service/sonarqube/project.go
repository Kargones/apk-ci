// Package sonarqube provides implementation of project management functionality.
// This package contains the business logic for managing SonarQube projects,
// including metadata synchronization and administrator synchronization.
package sonarqube

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Kargones/apk-ci/internal/entity/gitea"
	"github.com/Kargones/apk-ci/internal/entity/sonarqube"
)

// ProjectManagementService provides business logic for project management operations.
// This service layer implements the logic for managing SonarQube projects,
// including metadata synchronization and administrator synchronization.
type ProjectManagementService struct {
	// sonarQubeService is the SonarQube service for project management
	sonarQubeService *Service

	// branchScanningService is the branch scanning service for scanning branches
	branchScanningService *BranchScanningService

	// giteaAPI is the Gitea API client for retrieving project data
	giteaAPI gitea.APIInterface

	// logger is the structured logger for this service
	logger *slog.Logger
}

// NewProjectManagementService creates a new instance of ProjectManagementService.
// This function initializes the service with the provided dependencies.
//
// Parameters:
//   - sonarQubeService: SonarQube service for project management
//   - branchScanningService: branch scanning service for scanning branches
//   - giteaAPI: Gitea API client for retrieving project data
//   - logger: structured logger instance
//
// Returns:
//   - *ProjectManagementService: initialized project management service
func NewProjectManagementService(
	sonarQubeService *Service,
	branchScanningService *BranchScanningService,
	giteaAPI gitea.APIInterface,
	logger *slog.Logger) *ProjectManagementService {
	return &ProjectManagementService{
		sonarQubeService:      sonarQubeService,
		branchScanningService: branchScanningService,
		giteaAPI:              giteaAPI,
		logger:                logger,
	}
}

// UpdateProject updates a project with the provided parameters.
// This method implements the logic for updating a project,
// including metadata synchronization and administrator synchronization.
//
// Parameters:
//   - ctx: context for the operation
//   - params: project update parameters
//
// Returns:
//   - error: error if update fails
func (p *ProjectManagementService) UpdateProject(ctx context.Context, params *sonarqube.ProjectUpdateParams) error {
	p.logger.Debug("Updating project", "owner", params.Owner, "repo", params.Repo)

	// Get README.md content from Gitea
	readmeContent, err := p.giteaAPI.GetFileContent("README.md")
	if err != nil {
		p.logger.Warn("Failed to retrieve README.md from Gitea", "error", err)
		// Continue with empty content if README.md is not found
		readmeContent = []byte{}
	}

	// Update project description in SonarQube
	projectKey := fmt.Sprintf("%s_%s", params.Owner, params.Repo)
	if err := p.sonarQubeService.UpdateProjectDescription(ctx, projectKey, string(readmeContent)); err != nil {
		p.logger.Error("Failed to update project description in SonarQube", "error", err)
		return fmt.Errorf("failed to update project description: %w", err)
	}

	// Synchronize administrators with Gitea teams
	if err := p.syncAdministrators(ctx, params.Owner, params.Repo, projectKey); err != nil {
		p.logger.Error("Failed to synchronize administrators with Gitea teams", "error", err)
		return fmt.Errorf("failed to synchronize administrators: %w", err)
	}

	p.logger.Debug("Project updated successfully")
	return nil
}

// syncAdministrators synchronizes project administrators with Gitea teams.
// This method synchronizes the list of project administrators with the
// owners and dev teams from the Gitea repository.
//
// Parameters:
//   - ctx: context for the operation
//   - owner: repository owner
//   - repo: repository name
//   - projectKey: SonarQube project key
//
// Returns:
//   - error: error if synchronization fails
func (p *ProjectManagementService) syncAdministrators(ctx context.Context, owner, _, projectKey string) error {
	// Get members of the owners team
	owners, err := p.giteaAPI.GetTeamMembers(owner, "owners")
	if err != nil {
		p.logger.Error("Failed to get owners team members", "error", err)
		return fmt.Errorf("failed to get owners team members: %w", err)
	}

	// Get members of the dev team
	devs, err := p.giteaAPI.GetTeamMembers(owner, "dev")
	if err != nil {
		p.logger.Error("Failed to get dev team members", "error", err)
		return fmt.Errorf("failed to get dev team members: %w", err)
	}

	// Combine owners and devs
	administrators := make([]string, 0, len(owners)+len(devs))
	administrators = append(administrators, owners...)
	administrators = append(administrators, devs...)

	// Update project administrators in SonarQube
	if err := p.sonarQubeService.UpdateProjectAdministrators(ctx, projectKey, administrators); err != nil {
		p.logger.Error("Failed to update project administrators in SonarQube", "error", err)
		return fmt.Errorf("failed to update project administrators: %w", err)
	}

	return nil
}

// SyncRepository synchronizes a repository with the provided parameters.
// This method implements the logic for synchronizing a repository,
// including branch enumeration and project matching.
//
// Parameters:
//   - ctx: context for the operation
//   - params: repository sync parameters
//
// Returns:
//   - error: error if sync fails
func (p *ProjectManagementService) SyncRepository(ctx context.Context, params *sonarqube.RepoSyncParams) error {
	p.logger.Debug("Synchronizing repository", "owner", params.Owner, "repo", params.Repo)

	// Get branches from Gitea API
	branches, err := p.giteaAPI.GetBranches(params.Repo)
	if err != nil {
		p.logger.Error("Failed to retrieve branches from Gitea", "error", err)
		return fmt.Errorf("failed to retrieve branches: %w", err)
	}

	// Get projects from SonarQube API
	projects, err := p.sonarQubeService.ListProjects(ctx, params.Owner, params.Repo)
	if err != nil {
		p.logger.Error("Failed to retrieve projects from SonarQube", "error", err)
		return fmt.Errorf("failed to retrieve projects: %w", err)
	}

	// Create a map of existing projects for quick lookup
	projectMap := make(map[string]bool)
	for _, project := range projects {
		projectMap[project.Key] = true
	}

	// Process each branch concurrently
	errChan := make(chan error, len(branches))
	semaphore := make(chan struct{}, 10) // Limit concurrent operations to 10

	for _, branch := range branches {
		go func(branch gitea.Branch) {
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			// Generate project key
			projectKey := fmt.Sprintf("%s_%s_%s", params.Owner, params.Repo, branch.Name)

			// Check if project exists
			if projectMap[projectKey] {
				// Project exists, update it
				updateParams := &sonarqube.ProjectUpdateParams{
					Owner: params.Owner,
					Repo:  params.Repo,
				}
				if err := p.UpdateProject(ctx, updateParams); err != nil {
					p.logger.Error("Failed to update project", "projectKey", projectKey, "error", err)
					errChan <- fmt.Errorf("failed to update project %s: %w", projectKey, err)
					return
				}
			} else {
				// Project does not exist, scan the branch
				scanParams := &sonarqube.ScanBranchParams{
					Owner:  params.Owner,
					Repo:   params.Repo,
					Branch: branch.Name,
				}
				// Получаем список коммитов для сканирования
				commitsToScan, err := p.branchScanningService.CheckScanBranch(ctx, scanParams)
				if err != nil {
					p.logger.Error("Failed to check commits for scanning", "branch", branch.Name, "error", err)
					errChan <- fmt.Errorf("failed to check commits for branch %s: %w", branch.Name, err)
					return
				}
				if err := p.branchScanningService.ScanBranch(ctx, scanParams, commitsToScan); err != nil {
					p.logger.Error("Failed to scan branch", "branch", branch.Name, "error", err)
					errChan <- fmt.Errorf("failed to scan branch %s: %w", branch.Name, err)
					return
				}
			}

			errChan <- nil
		}(branch)
	}

	// Wait for all goroutines to complete
	for i := 0; i < len(branches); i++ {
		if err := <-errChan; err != nil {
			p.logger.Error("Error during branch processing", "error", err)
			return fmt.Errorf("error during branch processing: %w", err)
		}
	}

	// Call repo-clear to clean up stale projects
	clearParams := &sonarqube.RepoClearParams{
		Owner: params.Owner,
		Repo:  params.Repo,
		Force: false, // Use non-forceful clear by default
	}
	if err := p.ClearRepository(ctx, clearParams); err != nil {
		p.logger.Error("Failed to clear repository", "error", err)
		return fmt.Errorf("failed to clear repository: %w", err)
	}

	p.logger.Debug("Repository synchronized successfully")
	return nil
}

// ClearRepository clears a repository with the provided parameters.
// This method implements the logic for clearing a repository,
// including project age checking and deletion.
//
// Parameters:
//   - ctx: context for the operation
//   - params: repository clear parameters
//
// Returns:
//   - error: error if clear fails
func (p *ProjectManagementService) ClearRepository(ctx context.Context, params *sonarqube.RepoClearParams) error {
	p.logger.Debug("Clearing repository", "owner", params.Owner, "repo", params.Repo, "force", params.Force)

	// Get projects from SonarQube API
	projects, err := p.sonarQubeService.ListProjects(ctx, params.Owner, params.Repo)
	if err != nil {
		p.logger.Error("Failed to retrieve projects from SonarQube", "error", err)
		return fmt.Errorf("failed to retrieve projects: %w", err)
	}

	// Process each project
	for _, project := range projects {
		// If force flag is set, delete the project immediately
		if params.Force {
			p.logger.Debug("Force deleting project", "projectKey", project.Key)
			if err := p.sonarQubeService.DeleteProject(ctx, project.Key); err != nil {
				p.logger.Error("Failed to delete project", "projectKey", project.Key, "error", err)
				return fmt.Errorf("failed to delete project %s: %w", project.Key, err)
			}
			continue
		}

		// Check project age
		// This is a simplified implementation - in a real implementation,
		// you would need to get the last analysis date and compare it with the current date
		// For now, we'll just log that the project age check is not fully implemented
		p.logger.Warn("Project age check is not fully implemented yet", "projectKey", project.Key)

		// In a real implementation, you would delete the project if it's older than the threshold
		// For now, we'll just log that the project would be deleted
		p.logger.Debug("Project would be deleted if it was older than threshold", "projectKey", project.Key)
	}

	p.logger.Debug("Repository cleared successfully")
	return nil
}

// ToDo: необходимо дополнительно реализовать следующий функционал:
// - Add unit tests for all methods
// - Implement project age checking and deletion logic
// - Implement better error handling and recovery
// - Add progress reporting during operations
//
// Ссылки на пункты плана и требований:
// - tasks.md: 6.1, 6.2, 6.3
// - requirements.md: 3.1, 3.2, 3.3, 4.1, 4.2, 4.3, 4.4, 5.1, 5.2, 5.3, 5.4, 5.5
