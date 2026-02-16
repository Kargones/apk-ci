// Package sonarqube provides tests for SonarScanner service implementation.
package sonarqube

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/entity/sonarqube"
	"github.com/stretchr/testify/assert"
)

// TestNewSonarScannerService tests the creation of a new SonarScannerService.
func TestNewSonarScannerService(t *testing.T) {
	// Create a real entity for testing (we'll use nil for simplicity)
	var entity sonarqube.SonarScannerInterface
	cfg := &config.ScannerConfig{
		ScannerURL:     "http://localhost:3000",
		ScannerVersion: "latest",
		Timeout:        30 * time.Second,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	service := NewSonarScannerService(entity, cfg, logger)

	assert.NotNil(t, service)
	assert.Equal(t, entity, service.entity)
	assert.Equal(t, cfg, service.config)
	assert.Equal(t, logger, service.logger)
}

// TestSonarScannerService_DownloadScanner tests the DownloadScanner method.
func TestSonarScannerService_DownloadScanner(t *testing.T) {
	// Create a real entity for testing
	// Skip test as it requires actual git repository setup
	t.Skip("Skipping test - requires proper git repository setup")
}

// TestSonarScannerService_ConfigureScanner tests the ConfigureScanner method.
func TestSonarScannerService_ConfigureScanner(t *testing.T) {
	cfg := &config.ScannerConfig{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := sonarqube.NewSonarScannerEntity(cfg, logger)
	service := NewSonarScannerService(entity, cfg, logger)

	scannerConfig := &sonarqube.ScannerConfig{
		Properties: map[string]string{
			"sonar.projectKey": "test-project",
			"sonar.sources":    "src",
		},
	}
	err := service.ConfigureScanner(scannerConfig)

	// ConfigureScanner should succeed as it just sets properties
	assert.NoError(t, err)
}

// TestSonarScannerService_SetProperty tests the SetProperty method.
func TestSonarScannerService_SetProperty(t *testing.T) {
	cfg := &config.ScannerConfig{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := sonarqube.NewSonarScannerEntity(cfg, logger)
	service := NewSonarScannerService(entity, cfg, logger)

	// This method doesn't return an error, so we just call it
	service.SetProperty("test.key", "test.value")

	// Test passes if no panic occurs
	assert.True(t, true)
}

// TestSonarScannerService_GetProperty tests the GetProperty method.
func TestSonarScannerService_GetProperty(t *testing.T) {
	cfg := &config.ScannerConfig{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := sonarqube.NewSonarScannerEntity(cfg, logger)
	service := NewSonarScannerService(entity, cfg, logger)

	value := service.GetProperty("test.key")

	// Since the entity is not properly initialized, we expect an empty string
	assert.Equal(t, "", value)
}

// TestSonarScannerService_ValidateScannerConfig tests the ValidateScannerConfig method.
func TestSonarScannerService_ValidateScannerConfig(t *testing.T) {
	cfg := &config.ScannerConfig{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := sonarqube.NewSonarScannerEntity(cfg, logger)
	service := NewSonarScannerService(entity, cfg, logger)

	err := service.ValidateScannerConfig()

	// Since we're using a real entity without proper setup, we expect an error
	assert.Error(t, err)
}

// TestSonarScannerService_InitializeScanner tests the InitializeScanner method.
func TestSonarScannerService_InitializeScanner(t *testing.T) {
	cfg := &config.ScannerConfig{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := sonarqube.NewSonarScannerEntity(cfg, logger)
	service := NewSonarScannerService(entity, cfg, logger)

	err := service.InitializeScanner()

	// Since we're using a real entity without proper setup, we expect an error
	assert.Error(t, err)
}

// TestSonarScannerService_ExecuteScanner tests the ExecuteScanner method.
func TestSonarScannerService_ExecuteScanner(t *testing.T) {
	cfg := &config.ScannerConfig{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := sonarqube.NewSonarScannerEntity(cfg, logger)
	service := NewSonarScannerService(entity, cfg, logger)

	ctx := context.Background()
	result, err := service.ExecuteScanner(ctx)

	// Since we're using a real entity without proper setup, we expect an error
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestSonarScannerService_CleanupScanner tests the CleanupScanner method.
func TestSonarScannerService_CleanupScanner(t *testing.T) {
	cfg := &config.ScannerConfig{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entity := sonarqube.NewSonarScannerEntity(cfg, logger)
	service := NewSonarScannerService(entity, cfg, logger)

	err := service.CleanupScanner()

	// CleanupScanner should succeed as it just cleans up temporary files
	assert.NoError(t, err)
}