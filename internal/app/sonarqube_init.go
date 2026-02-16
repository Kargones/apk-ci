// Package app provides SonarQube service initialization functionality.
// This file contains functions for initializing SonarQube services with proper
// dependency injection and structured logging with correlation IDs.
package app

import (
	"fmt"
	"log/slog"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/entity/gitea"
	sonarqubeEntity "github.com/Kargones/apk-ci/internal/entity/sonarqube"
	"github.com/Kargones/apk-ci/internal/service/sonarqube"
)

// InitSonarQubeServices initializes all SonarQube services with proper dependency injection.
// This function creates and configures all necessary services for SonarQube integration,
// including structured logging with correlation IDs.
//
// Parameters:
//   - l: structured logger instance
//   - cfg: application configuration
//   - giteaAPI: Gitea API client for repository operations
//
// Returns:
//   - *sonarqube.SQCommandHandler: initialized command handler
//   - error: initialization error or nil on success
func InitSonarQubeServices(l *slog.Logger, cfg *config.Config, giteaAPI gitea.APIInterface) (*sonarqube.SQCommandHandler, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}
	
	if cfg.AppConfig == nil {
		return nil, fmt.Errorf("app configuration cannot be nil")
	}
	
	if cfg.SecretConfig == nil {
		return nil, fmt.Errorf("secret configuration cannot be nil")
	}
	
	// Initialize SonarQube configuration
	sonarQubeConfig := &config.SonarQubeConfig{
		URL:                cfg.AppConfig.SonarQube.URL,
		Token:              cfg.SecretConfig.SonarQube.Token,
		Timeout:            cfg.AppConfig.SonarQube.Timeout,
		RetryAttempts:      cfg.AppConfig.SonarQube.RetryAttempts,
		RetryDelay:         cfg.AppConfig.SonarQube.RetryDelay,
		ProjectPrefix:      cfg.AppConfig.SonarQube.ProjectPrefix,
		DefaultVisibility:  cfg.AppConfig.SonarQube.DefaultVisibility,
		QualityGateTimeout: cfg.AppConfig.SonarQube.QualityGateTimeout,
	}
	
	// Initialize direct SonarQube client without retry mechanism
	sonarQubeClient := sonarqubeEntity.NewEntity(sonarQubeConfig, l)
	
	// Initialize SonarQube service with direct client
	sonarQubeService := sonarqube.NewSonarQubeService(sonarQubeClient, sonarQubeConfig, l)
	
	// Initialize scanner configuration
	scannerConfig := &config.ScannerConfig{
		ScannerURL:     cfg.AppConfig.Scanner.ScannerURL,
		ScannerVersion: cfg.AppConfig.Scanner.ScannerVersion,
		JavaOpts:       cfg.AppConfig.Scanner.JavaOpts,
		Properties:     cfg.AppConfig.Scanner.Properties,
		Timeout:        cfg.AppConfig.Scanner.Timeout,
		WorkDir:        cfg.AppConfig.Scanner.WorkDir,
		TempDir:        cfg.AppConfig.Scanner.TempDir,
	}
	
	// Initialize direct scanner client without retry mechanism
	scannerClient := sonarqubeEntity.NewSonarScannerEntity(scannerConfig, l)
	
	// Initialize scanner service with direct client
	scannerService := sonarqube.NewSonarScannerService(scannerClient, scannerConfig, l)
	
	// Initialize branch scanning service
	branchScanningService := sonarqube.NewBranchScanningService(
		sonarQubeService,
		scannerService,
		giteaAPI,
		l,
		cfg,
	)
	
	// Initialize project management service
	projectService := sonarqube.NewProjectManagementService(sonarQubeService, branchScanningService, giteaAPI, l)
	
	// Initialize reporting service
	reportingService := sonarqube.NewReportingService(sonarQubeService, giteaAPI, l)
	
	// Initialize command handler with all services
	commandHandler := sonarqube.NewSQCommandHandler(
		branchScanningService,
		sonarQubeService,
		scannerService,
		projectService,
		reportingService,
		giteaAPI,
		l,
	)
	
	return commandHandler, nil
}

// InitSonarQubeConfig initializes SonarQube configuration from app config.
// This function extracts SonarQube-specific configuration from the main app config.
//
// Parameters:
//   - cfg: main application configuration
//
// Returns:
//   - *config.SonarQubeConfig: SonarQube configuration
func InitSonarQubeConfig(cfg *config.Config) *config.SonarQubeConfig {
	if cfg == nil || cfg.AppConfig == nil || cfg.SecretConfig == nil {
		return nil
	}
	
	return &config.SonarQubeConfig{
		URL:                cfg.AppConfig.SonarQube.URL,
		Token:              cfg.SecretConfig.SonarQube.Token,
		Timeout:            cfg.AppConfig.SonarQube.Timeout,
		RetryAttempts:      cfg.AppConfig.SonarQube.RetryAttempts,
		RetryDelay:         cfg.AppConfig.SonarQube.RetryDelay,
		ProjectPrefix:      cfg.AppConfig.SonarQube.ProjectPrefix,
		DefaultVisibility:  cfg.AppConfig.SonarQube.DefaultVisibility,
		QualityGateTimeout: cfg.AppConfig.SonarQube.QualityGateTimeout,
	}
}

// InitSonarScannerConfig initializes SonarScanner configuration from app config.
// This function extracts SonarScanner-specific configuration from the main app config.
//
// Parameters:
//   - cfg: main application configuration
//
// Returns:
//   - *config.ScannerConfig: SonarScanner configuration
func InitSonarScannerConfig(cfg *config.Config) *config.ScannerConfig {
	if cfg == nil || cfg.AppConfig == nil {
		return nil
	}
	
	return &config.ScannerConfig{
		ScannerURL:     cfg.AppConfig.Scanner.ScannerURL,
		ScannerVersion: cfg.AppConfig.Scanner.ScannerVersion,
		JavaOpts:       cfg.AppConfig.Scanner.JavaOpts,
		Properties:     cfg.AppConfig.Scanner.Properties,
		Timeout:        cfg.AppConfig.Scanner.Timeout,
		WorkDir:        cfg.AppConfig.Scanner.WorkDir,
		TempDir:        cfg.AppConfig.Scanner.TempDir,
	}
}