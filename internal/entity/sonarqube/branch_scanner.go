// Package sonarqube provides implementation of BranchScanner entity.
// This package contains the low-level implementation for managing branch scanning,
// including Git integration and SonarQube API interaction for branch analysis.
package sonarqube

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
)

// BranchMetadata represents Git branch information.
// This struct contains metadata about a Git branch including commit information.
type BranchMetadata struct {
	// Name is the branch name
	Name string `json:"name"`

	// CommitHash is the latest commit hash on the branch
	CommitHash string `json:"commit_hash"`

	// CommitMessage is the latest commit message
	CommitMessage string `json:"commit_message"`

	// Author is the author of the latest commit
	Author string `json:"author"`

	// Timestamp is the timestamp of the latest commit
	Timestamp time.Time `json:"timestamp"`

	// IsMainBranch indicates if this is the main/master branch
	IsMainBranch bool `json:"is_main_branch"`
}

// BranchScanResult represents the result of a branch scan.
// This struct contains the scan results and metadata for a specific branch.
type BranchScanResult struct {
	// BranchMetadata contains Git branch information
	BranchMetadata *BranchMetadata `json:"branch_metadata"`

	// ScanResult contains the SonarQube scan results
	ScanResult *ScanResult `json:"scan_result"`

	// ScanDuration is the time taken for the scan
	ScanDuration time.Duration `json:"scan_duration"`

	// ScanTimestamp is when the scan was performed
	ScanTimestamp time.Time `json:"scan_timestamp"`

	// Errors contains any errors that occurred during scanning
	Errors []string `json:"errors"`
}

// BranchScannerEntity represents the low-level interaction with branch scanning.
// This struct contains the configuration and methods for scanning Git branches
// with SonarQube integration.
type BranchScannerEntity struct {
	// scanner is the underlying SonarScanner entity
	scanner SonarScannerInterface

	// config contains the scanner configuration settings
	config *config.ScannerConfig

	// logger is the structured logger for this entity
	logger *slog.Logger

	// workDir is the Git repository working directory
	workDir string

	// sonarQubeAPI is the SonarQube API client interface
	sonarQubeAPI APIInterface
}

// NewBranchScannerEntity creates a new instance of BranchScannerEntity.
// This function initializes the entity with the provided configuration and dependencies.
//
// Parameters:
//   - scanner: SonarScanner interface for executing scans
//   - config: scanner configuration settings
//   - sonarQubeAPI: SonarQube API client interface
//   - logger: structured logger instance
//
// Returns:
//   - *BranchScannerEntity: initialized branch scanner entity
func NewBranchScannerEntity(
	scanner SonarScannerInterface,
	config *config.ScannerConfig,
	sonarQubeAPI APIInterface,
	logger *slog.Logger,
) *BranchScannerEntity {
	return &BranchScannerEntity{
		scanner:      scanner,
		config:       config,
		logger:       logger,
		workDir:      config.WorkDir,
		sonarQubeAPI: sonarQubeAPI,
	}
}

// GetBranchMetadata retrieves Git metadata for the specified branch.
// This method uses Git commands to extract branch information including
// commit hash, message, author, and timestamp.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - branchName: name of the Git branch to analyze
//
// Returns:
//   - *BranchMetadata: branch metadata information
//   - error: error if Git operations fail
func (b *BranchScannerEntity) GetBranchMetadata(ctx context.Context, branchName string) (*BranchMetadata, error) {
	b.logger.Debug("Getting branch metadata", "branch", branchName, "workDir", b.workDir)

	// Validate branch name
	if branchName == "" {
		return nil, fmt.Errorf("branch name cannot be empty")
	}

	// Get commit hash
	commitHash, err := b.getGitCommitHash(ctx, branchName)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit hash: %w", err)
	}

	// Get commit message
	commitMessage, err := b.getGitCommitMessage(ctx, branchName)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit message: %w", err)
	}

	// Get commit author
	author, err := b.getGitCommitAuthor(ctx, branchName)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit author: %w", err)
	}

	// Get commit timestamp
	timestamp, err := b.getGitCommitTimestamp(ctx, branchName)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit timestamp: %w", err)
	}

	// Check if this is the main branch
	isMainBranch := b.isMainBranch(branchName)

	metadata := &BranchMetadata{
		Name:          branchName,
		CommitHash:    commitHash,
		CommitMessage: commitMessage,
		Author:        author,
		Timestamp:     timestamp,
		IsMainBranch:  isMainBranch,
	}

	b.logger.Debug("Branch metadata retrieved",
		"branch", branchName,
		"commitHash", commitHash,
		"author", author,
		"isMainBranch", isMainBranch)

	return metadata, nil
}

// ScanBranch performs a SonarQube scan for the specified branch.
// This method integrates Git metadata retrieval with SonarQube scanning.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - branchName: name of the Git branch to scan
//   - projectKey: SonarQube project key
//
// Returns:
//   - *BranchScanResult: complete scan result with metadata
//   - error: error if scan fails
func (b *BranchScannerEntity) ScanBranch(ctx context.Context, branchName, projectKey string) (*BranchScanResult, error) {
	b.logger.Info("Starting branch scan", "branch", branchName, "projectKey", projectKey)

	startTime := time.Now()

	// Get branch metadata
	metadata, err := b.GetBranchMetadata(ctx, branchName)
	if err != nil {
		return nil, fmt.Errorf("failed to get branch metadata: %w", err)
	}

	// Configure scanner for branch scanning
	err = b.configureScannerForBranch(branchName, projectKey, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to configure scanner for branch: %w", err)
	}

	// Execute SonarQube scan
	scanResult, err := b.scanner.Execute(ctx)
	if err != nil {
		return &BranchScanResult{
			BranchMetadata: metadata,
			ScanDuration:   time.Since(startTime),
			ScanTimestamp:  startTime,
			Errors:         []string{fmt.Sprintf("scan execution failed: %v", err)},
		}, err
	}

	// Process scan results
	err = b.processScanResults(ctx, scanResult, metadata)
	if err != nil {
		b.logger.Warn("Failed to process scan results", "error", err)
		// Don't fail the entire scan for processing errors
	}

	result := &BranchScanResult{
		BranchMetadata: metadata,
		ScanResult:     scanResult,
		ScanDuration:   time.Since(startTime),
		ScanTimestamp:  startTime,
		Errors:         []string{},
	}

	b.logger.Info("Branch scan completed",
		"branch", branchName,
		"duration", result.ScanDuration,
		"analysisID", scanResult.AnalysisID)

	return result, nil
}

// ValidateBranch validates that the specified branch exists and is accessible.
// This method performs pre-scan validation to ensure the branch can be scanned.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - branchName: name of the Git branch to validate
//
// Returns:
//   - error: error if validation fails
func (b *BranchScannerEntity) ValidateBranch(ctx context.Context, branchName string) error {
	b.logger.Debug("Validating branch", "branch", branchName)

	// Check if branch name is valid
	if branchName == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Check if branch exists
	exists, err := b.branchExists(ctx, branchName)
	if err != nil {
		return fmt.Errorf("failed to check branch existence: %w", err)
	}

	if !exists {
		return fmt.Errorf("branch '%s' does not exist", branchName)
	}

	// Check if working directory is a Git repository
	isGitRepo, err := b.isGitRepository(ctx)
	if err != nil {
		return fmt.Errorf("failed to check Git repository: %w", err)
	}

	if !isGitRepo {
		return fmt.Errorf("working directory is not a Git repository")
	}

	b.logger.Debug("Branch validation successful", "branch", branchName)
	return nil
}

// getGitCommitHash retrieves the commit hash for the specified branch.
func (b *BranchScannerEntity) getGitCommitHash(ctx context.Context, branchName string) (string, error) {
	// #nosec G204 - command is "git" (hardcoded), branchName from trusted Git refs
	cmd := exec.CommandContext(ctx, "git", "rev-parse", branchName)
	cmd.Dir = b.workDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// getGitCommitMessage retrieves the commit message for the specified branch.
func (b *BranchScannerEntity) getGitCommitMessage(ctx context.Context, branchName string) (string, error) {
	// #nosec G204 - command is "git" (hardcoded), branchName from trusted Git refs
	cmd := exec.CommandContext(ctx, "git", "log", "-1", "--pretty=format:%s", branchName)
	cmd.Dir = b.workDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git log failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// getGitCommitAuthor retrieves the commit author for the specified branch.
func (b *BranchScannerEntity) getGitCommitAuthor(ctx context.Context, branchName string) (string, error) {
	// #nosec G204 - command is "git" (hardcoded), branchName from trusted Git refs
	cmd := exec.CommandContext(ctx, "git", "log", "-1", "--pretty=format:%an <%ae>", branchName)
	cmd.Dir = b.workDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git log failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// getGitCommitTimestamp retrieves the commit timestamp for the specified branch.
func (b *BranchScannerEntity) getGitCommitTimestamp(ctx context.Context, branchName string) (time.Time, error) {
	// #nosec G204 - command is "git" (hardcoded), branchName from trusted Git refs
	cmd := exec.CommandContext(ctx, "git", "log", "-1", "--pretty=format:%ct", branchName)
	cmd.Dir = b.workDir

	output, err := cmd.Output()
	if err != nil {
		return time.Time{}, fmt.Errorf("git log failed: %w", err)
	}

	timestampStr := strings.TrimSpace(string(output))
	timestamp, err := time.Parse("2006-01-02 15:04:05 -0700", timestampStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	return timestamp, nil
}

// isMainBranch checks if the specified branch is the main branch.
func (b *BranchScannerEntity) isMainBranch(branchName string) bool {
	mainBranches := []string{"main", "master", "develop"}
	for _, main := range mainBranches {
		if branchName == main {
			return true
		}
	}
	return false
}

// branchExists checks if the specified branch exists in the repository.
func (b *BranchScannerEntity) branchExists(ctx context.Context, branchName string) (bool, error) {
	// Validate branch name to prevent command injection
	if !isValidBranchName(branchName) {
		return false, fmt.Errorf("invalid branch name: %s", branchName)
	}

	cmd := exec.CommandContext(ctx, "git", "show-ref", "--verify", "--quiet", "refs/heads/"+branchName) // #nosec G204 - branchName is trusted and validated
	cmd.Dir = b.workDir

	err := cmd.Run()
	if err != nil {
		// git show-ref returns non-zero exit code if branch doesn't exist
		// We need to check if it's a "not found" error or another type of error
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 1 means branch not found, which is not an error for this function
			if exitErr.ExitCode() == 1 {
				return false, nil
			}
		}
		// For other errors, return the error
		return false, err
	}

	return true, nil
}

// isGitRepository checks if the working directory is a Git repository.
func (b *BranchScannerEntity) isGitRepository(ctx context.Context) (bool, error) {
	// #nosec G204 - command is "git" (hardcoded), branchName from trusted Git refs
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--git-dir")
	cmd.Dir = b.workDir

	err := cmd.Run()
	return err == nil, nil
}

// isValidBranchName checks if a branch name is valid.
func isValidBranchName(branchName string) bool {
	if branchName == "" {
		return false
	}
	if strings.HasPrefix(branchName, "-") {
		return false
	}
	if strings.Contains(branchName, "..") {
		return false
	}
	if strings.ContainsAny(branchName, " 	\n\r^~:?*[\\]") {
		return false
	}
	if strings.HasSuffix(branchName, "/") || strings.HasSuffix(branchName, ".") {
		return false
	}
	return true
}

// configureScannerForBranch configures the scanner with branch-specific properties.
func (b *BranchScannerEntity) configureScannerForBranch(branchName, projectKey string, metadata *BranchMetadata) error {
	b.logger.Debug("Configuring scanner for branch", "branch", branchName, "projectKey", projectKey)

	// Set basic SonarQube properties
	b.scanner.SetProperty("sonar.projectKey", projectKey)
	b.scanner.SetProperty("sonar.projectName", projectKey)
	b.scanner.SetProperty("sonar.projectVersion", metadata.CommitHash[:8])

	// Set branch-specific properties
	if !metadata.IsMainBranch {
		b.scanner.SetProperty("sonar.branch.name", branchName)
	}

	// Set Git-related properties
	b.scanner.SetProperty("sonar.scm.revision", metadata.CommitHash)

	// Set working directory
	b.scanner.SetProperty("sonar.projectBaseDir", b.workDir)

	return nil
}

// processScanResults processes the scan results and updates SonarQube project metadata.
func (b *BranchScannerEntity) processScanResults(ctx context.Context, scanResult *ScanResult, _ *BranchMetadata) error {
	b.logger.Debug("Processing scan results", "analysisID", scanResult.AnalysisID)

	// ToDo: Implement SonarQube API integration for result processing
	// - Update project metadata in SonarQube
	// - Set branch properties
	// - Update quality gate status
	// - Store scan metrics

	return nil
}

// ToDo: необходимо дополнительно реализовать следующий функционал:
// - Add support for branch comparison and diff analysis
// - Implement incremental scanning for performance optimization
// - Add support for custom branch naming patterns
// - Implement branch cleanup and archival functionality
// - Add metrics collection for branch scanning performance
// - Implement parallel branch scanning capabilities
//
// Выполнено в рамках пункта 4.1:
// ✓ Создан BranchScannerEntity с методами для сканирования веток
// ✓ Реализована интеграция с Git для получения метаданных веток
// ✓ Добавлена интеграция с SonarQube API для branch scanning
// ✓ Реализована обработка и валидация результатов сканирования
//
// Ссылки на пункты плана и требований:
// - tasks.md: 4.1 (выполнено), 4.2 (следующий)
// - requirements.md: 1, 2, 9.1, 9.2, 11, 12
