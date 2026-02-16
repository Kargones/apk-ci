// Package sonarqube provides implementation of branch scanning functionality.
// This package contains the business logic for scanning branches,
// including branch data retrieval, project management, and scanner execution.
package sonarqube

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/entity/gitea"
	"github.com/Kargones/apk-ci/internal/entity/sonarqube"
	"github.com/Kargones/apk-ci/internal/git"
)

// BranchScanningService provides business logic for branch scanning operations.
// This service layer implements the logic for scanning branches,
// including branch data retrieval, project management, and scanner execution.
type BranchScanningService struct {
	// sonarQubeService is the SonarQube service for project management
	sonarQubeService *Service

	// scannerService is the scanner service for executing scans
	scannerService *SonarScannerService

	// giteaAPI is the Gitea API client for retrieving branch data
	giteaAPI gitea.APIInterface

	// logger is the structured logger for this service
	logger *slog.Logger

	// config is the application configuration
	config *config.Config
}

// NewBranchScanningService creates a new instance of BranchScanningService.
// This function initializes the service with the provided dependencies.
//
// Parameters:
//   - sonarQubeService: SonarQube service for project management
//   - scannerService: scanner service for executing scans
//   - giteaAPI: Gitea API client for retrieving branch data
//   - logger: structured logger instance
//   - config: application configuration
//
// Returns:
//   - *BranchScanningService: initialized branch scanning service
func NewBranchScanningService(
	sonarQubeService *Service,
	scannerService *SonarScannerService,
	giteaAPI gitea.APIInterface,
	logger *slog.Logger,
	config *config.Config) *BranchScanningService {
	return &BranchScanningService{
		sonarQubeService: sonarQubeService,
		scannerService:   scannerService,
		giteaAPI:         giteaAPI,
		logger:           logger,
		config:           config,
	}
}

// CheckScanBranch determines which commits need to be scanned and returns filtered commit hashes.
// This method handles the logic for determining commits to scan and filtering out already analyzed ones.
//
// Parameters:
//   - ctx: context for the operation
//   - params: branch scanning parameters
//
// Returns:
//   - []string: filtered list of commit hashes to scan
//   - error: error if checking fails
func (b *BranchScanningService) CheckScanBranch(ctx context.Context, params *sonarqube.ScanBranchParams) ([]string, error) {
	var err error
	b.logger.Debug("Checking branch for commits to scan", "owner", params.Owner, "repo", params.Repo, "branch", params.Branch)

	// Check if branch is the main branch (removed isMainBranch variable as it's no longer used)
	// This is a simplified implementation - in a real implementation,
	// you would need a more robust way to determine the main branch

	// Generate project key
	projectKey := fmt.Sprintf("%s_%s_%s", params.Owner, params.Repo, params.Branch)

	// Get or create project in SonarQube
	var project *sonarqube.Project
	project, err = b.sonarQubeService.GetProject(ctx, projectKey)
	if err != nil {
		b.logger.Error("Failed to get or create project in SonarQube", "error", err)
		return nil, fmt.Errorf("failed to get or create project: %w", err)
	}

	// Determine commit hashes to scan
	var commitsToScan []string

	commitHash := params.CommitHash
	if commitHash == "" {
		// Use GetBranchCommitRange to get commits for scanning
		commitRange, errGetBranchCommitRange := b.giteaAPI.GetBranchCommitRange(params.Branch)
		if errGetBranchCommitRange != nil {
			b.logger.Warn("Failed to get branch commit range, using latest commit", "error", errGetBranchCommitRange)
			return []string{}, errGetBranchCommitRange
		}
		// Convert BranchCommitRange to commit hashes array
		if commitRange.FirstCommit != nil {
			commitsToScan = append(commitsToScan, commitRange.FirstCommit.SHA)
		}
		if commitRange.LastCommit != nil && (commitRange.FirstCommit == nil || commitRange.FirstCommit.SHA != commitRange.LastCommit.SHA) {
			commitsToScan = append(commitsToScan, commitRange.LastCommit.SHA)
		}
		b.logger.Debug("Added commits from branch range to scan queue", "firstCommit",
			func() string {
				if commitRange.FirstCommit != nil {
					return commitRange.FirstCommit.SHA
				}
				return "none"
			}(),
			"lastCommit",
			func() string {
				if commitRange.LastCommit != nil {
					return commitRange.LastCommit.SHA
				}
				return "none"
			}())
	} else {
		// If commit hash is provided, scan that specific commit
		commitsToScan = append(commitsToScan, commitHash)
		b.logger.Debug("Added specific commit to scan queue", "commitHash", commitHash)
	}

	// Check for existing analyses and filter out already scanned commits
	b.logger.Debug("Checking for existing analyses", "projectKey", project.Key, "totalCommits", len(commitsToScan))
	existingAnalyses, err := b.sonarQubeService.GetAnalyses(ctx, project.Key)
	if err != nil {
		b.logger.Warn("Failed to get existing analyses, proceeding with all scans", "error", err)
	} else {
		// Create a map of existing revisions for faster lookup
		existingRevisions := make(map[string]bool)
		for _, analysis := range existingAnalyses {
			existingRevisions[analysis.Revision] = true
		}

		// Filter out commits that already have analyses
		var commitsToScanFiltered []string
		for _, commit := range commitsToScan {
			if existingRevisions[commit] {
				b.logger.Info("Analysis already exists for commit, skipping", "commitHash", commit)
			} else {
				commitsToScanFiltered = append(commitsToScanFiltered, commit)
			}
		}
		commitsToScan = commitsToScanFiltered
	}

	b.logger.Debug("Commits to scan determined", "commitsCount", len(commitsToScan))
	return commitsToScan, nil
}

// ScanBranch scans a branch for code quality issues.
// This method orchestrates the scanning process for provided commits.
//
// Parameters:
//   - ctx: context for the operation
//   - params: branch scanning parameters
//   - commitsToScan: list of commit hashes to scan
//
// Returns:
//   - error: error if scanning fails
func (b *BranchScanningService) ScanBranch(ctx context.Context, params *sonarqube.ScanBranchParams, commitsToScan []string) error {
	b.logger.Debug("Starting scan of filtered commits", "commitsToScan", len(commitsToScan))

	// Generate project key
	projectKey := fmt.Sprintf("%s_%s_%s", params.Owner, params.Repo, params.Branch)

	// Get or create project in SonarQube
	var project *sonarqube.Project
	project, err := b.sonarQubeService.GetProject(ctx, projectKey)
	if err != nil {
		b.logger.Error("Failed to get or create project in SonarQube", "error", err)
		return fmt.Errorf("failed to get or create project: %w", err)
	}

	// Scan remaining commits
	for i, commit := range commitsToScan {
		b.logger.Debug("Scanning commit", "index", i+1, "total", len(commitsToScan), "commitHash", commit)
		if err := b.scanCommit(ctx, project, params, commit); err != nil {
			b.logger.Error("Failed to scan commit", "commitHash", commit, "error", err)
			return fmt.Errorf("failed to scan commit %s: %w", commit, err)
		}
	}

	b.logger.Debug("Branch scanned successfully", "projectKey", project.Key)
	return nil
}

// scanCommit scans a specific commit.
// This method configures and executes the scanner for a specific commit.
//
// Parameters:
//   - ctx: context for the operation
//   - project: SonarQube project
//   - params: branch scanning parameters
//   - commitHash: commit hash to scan
//
// Returns:
//   - error: error if scanning fails
func (b *BranchScanningService) scanCommit(ctx context.Context, project *sonarqube.Project, params *sonarqube.ScanBranchParams, commitHash string) error {
	// Get SonarQube configuration
	// ToDo: дублирование конфигурации, она уже есть в (*(*(*b).sonarQubeService).config), причем более правильная
	sonarQubeConfig := b.config.GetSonarQubeConfig()
	// Переключаем git-репозиторий к указанному коммиту
	if err := git.CheckoutCommit(ctx, b.logger, params.SourceDir, commitHash); err != nil {
		return fmt.Errorf("ошибка переключения к коммиту %s: %w", commitHash, err)
	}

	// DEBUG: Проверяем содержимое каталога после checkout'а
	if files, err := filepath.Glob(filepath.Join(params.SourceDir, "*")); err == nil {
		// Исключаем .git из списка для более чистого вывода
		var nonGitFiles []string
		var sourceFiles []string
		for _, file := range files {
			if !strings.Contains(file, ".git") {
				nonGitFiles = append(nonGitFiles, file)
				// Проверяем, есть ли каталоги с исходным кодом
				if info, err := os.Stat(file); err == nil && info.IsDir() {
					// Исключаем служебные каталоги
					baseName := filepath.Base(file)
					if baseName != ".gitea" && baseName != ".github" && !strings.HasPrefix(baseName, ".") {
						sourceFiles = append(sourceFiles, file)
					}
				}
			}
		}
		b.logger.Debug("DEBUG: Files in source directory before scanning",
			slog.String("sourceDir", params.SourceDir),
			slog.String("commitHash", commitHash),
			slog.Any("files", nonGitFiles),
			slog.Int("totalFiles", len(files)),
			slog.Int("nonGitFiles", len(nonGitFiles)),
			slog.Any("sourceFiles", sourceFiles),
			slog.Int("sourceFilesCount", len(sourceFiles)))

		// Если нет каталогов с исходным кодом, пропускаем сканирование
		if len(sourceFiles) == 0 {
			b.logger.Info("Skipping scan: no source code directories found in commit",
				slog.String("commitHash", commitHash),
				slog.String("sourceDir", params.SourceDir),
				slog.Any("availableFiles", nonGitFiles))
			return nil
		}
	}

	// Устанавливаем каталог сканирования
	b.scannerService.config.WorkDir = params.SourceDir

	// Configure scanner
	scannerConfig := &sonarqube.ScannerConfig{
		Properties: map[string]string{
			"sonar.projectKey": project.Key,
			"sonar.sources":    ".",
			"sonar.host.url":   sonarQubeConfig.URL,
			"sonar.login":      sonarQubeConfig.Token,
			"sonar.commit.sha": commitHash,
		},
	}

	// Add branch name only if branch analysis is not disabled
	if !sonarQubeConfig.DisableBranchAnalysis {
		scannerConfig.Properties["sonar.branch.name"] = params.Branch
	}

	// Configure scanner service
	if err := b.scannerService.ConfigureScanner(scannerConfig); err != nil {
		b.logger.Error("Failed to configure scanner", "error", err)
		return fmt.Errorf("failed to configure scanner: %w", err)
	}

	// Download scanner if needed
	_, err := b.scannerService.DownloadScanner(ctx, b.config.GetScannerConfig().ScannerURL, b.config.GetScannerConfig().ScannerVersion)
	if err != nil {
		b.logger.Error("Failed to download scanner", "error", err)
		return fmt.Errorf("failed to download scanner: %w", err)
	}

	// Initialize scanner
	if errInitializeScanner := b.scannerService.InitializeScanner(); errInitializeScanner != nil {
		b.logger.Error("Failed to initialize scanner", "error", errInitializeScanner)
		return fmt.Errorf("failed to initialize scanner: %w", errInitializeScanner)
	}

	// Execute scanner
	result, err := b.scannerService.ExecuteScanner(ctx)
	if err != nil {
		b.logger.Error("Failed to execute scanner", "error", err)
		return fmt.Errorf("failed to execute scanner: %w", err)
	}

	// Process scan result
	if !result.Success {
		b.logger.Error("Scanner execution was not successful", "errors", result.Errors)
		return fmt.Errorf("scanner execution failed: %v", result.Errors)
	}

	b.logger.Debug("Commit scanned successfully", "projectKey", project.Key, "commitHash", commitHash, "duration", result.Duration)
	return nil
}

// // scanIncrementalChanges scans incremental changes between two commits.
// // This method configures and executes the scanner for incremental changes.
// //
// // Parameters:
// //   - ctx: context for the operation
// //   - project: SonarQube project
// //   - params: branch scanning parameters
// //   - baseCommitSHA: base commit SHA
// //   - headCommitSHA: head commit SHA
// //
// // Returns:
// //   - error: error if scanning fails
// func (b *BranchScanningService) scanIncrementalChanges(ctx context.Context, project *sonarqube.Project, params *sonarqube.ScanBranchParams, baseCommitSHA, headCommitSHA string) error {
// 	// Get commits between base and head
// 	commits, err := b.giteaAPI.GetCommitsBetween(baseCommitSHA, headCommitSHA)
// 	if err != nil {
// 		b.logger.Error("Failed to get commits between base and head", "error", err)
// 		return fmt.Errorf("failed to get commits between base and head: %w", err)
// 	}

// 	// If there are no commits between base and head, nothing to scan
// 	if len(commits) == 0 {
// 		b.logger.Debug("No commits between base and head, skipping incremental scan")
// 		return nil
// 	}

// 	// Get SonarQube configuration
// 	sonarQubeConfig := b.config.GetSonarQubeConfig()

// 	// Configure scanner for incremental scan
// 	scannerConfig := &sonarqube.ScannerConfig{
// 		Properties: map[string]string{
// 			"sonar.projectKey":           project.Key,
// 			"sonar.sources":              b.config.WorkDir,
// 			"sonar.host.url":             sonarQubeConfig.URL,
// 			"sonar.login":                sonarQubeConfig.Token,
// 			"sonar.commit.sha":           headCommitSHA,
// 			"sonar.branch.target":        "main", // Assuming "main" is the target branch
// 			"sonar.pullrequest.key":      fmt.Sprintf("%s-%s", baseCommitSHA[:8], headCommitSHA[:8]),
// 			"sonar.pullrequest.branch":   params.Branch,
// 			"sonar.pullrequest.base":     baseCommitSHA,
// 			"sonar.pullrequest.provider": "gitea",
// 		},
// 	}

// 	// Add branch name only if branch analysis is not disabled
// 	if !sonarQubeConfig.DisableBranchAnalysis {
// 		scannerConfig.Properties["sonar.branch.name"] = params.Branch
// 	}

// 	// Configure scanner service
// 	if errConfigureScanner := b.scannerService.ConfigureScanner(scannerConfig); errConfigureScanner != nil {
// 		b.logger.Error("Failed to configure scanner", "error", errConfigureScanner)
// 		return fmt.Errorf("failed to configure scanner: %w", errConfigureScanner)
// 	}

// 	// Download scanner if needed
// 	_, errDownloadScanner := b.scannerService.DownloadScanner(ctx, b.config.GetScannerConfig().ScannerURL, b.config.GetScannerConfig().ScannerVersion)
// 	if errDownloadScanner != nil {
// 		b.logger.Error("Failed to download scanner", "error", errDownloadScanner)
// 		return fmt.Errorf("failed to download scanner: %w", errDownloadScanner)
// 	}

// 	// Initialize scanner
// 	if errInitializeScanner := b.scannerService.InitializeScanner(); errInitializeScanner != nil {
// 		b.logger.Error("Failed to initialize scanner", "error", errInitializeScanner)
// 		return fmt.Errorf("failed to initialize scanner: %w", errInitializeScanner)
// 	}

// 	// Execute scanner
// 	result, errExecuteScanner := b.scannerService.ExecuteScanner(ctx)
// 	if errExecuteScanner != nil {
// 		b.logger.Error("Failed to execute scanner", "error", errExecuteScanner)
// 		return fmt.Errorf("failed to execute scanner: %w", errExecuteScanner)
// 	}

// 	// Process scan result
// 	if !result.Success {
// 		b.logger.Error("Scanner execution was not successful", "errors", result.Errors)
// 		return fmt.Errorf("scanner execution failed: %v", result.Errors)
// 	}

// 	b.logger.Debug("Incremental changes scanned successfully", "projectKey", project.Key, "baseCommit", baseCommitSHA, "headCommit", headCommitSHA, "duration", result.Duration)
// 	return nil
// }

// ToDo: необходимо дополнительно реализовать следующий функционал:
// - Add unit tests for all methods
// - Implement more sophisticated branch data retrieval
// - Add support for different branch types (main, feature, release, etc.)
// - Implement better error handling and recovery
// - Add progress reporting during scanning
// - Implement caching for branch data
//
// Ссылки на пункты плана и требований:
// - tasks.md: 4.1, 4.2
// - requirements.md: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6
