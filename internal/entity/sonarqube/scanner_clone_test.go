package sonarqube

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownload_InvalidURL(t *testing.T) {
	cfg := &config.ScannerConfig{TempDir: os.TempDir()}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	e := NewSonarScannerEntity(cfg, logger)

	ctx := context.Background()
	_, err := e.Download(ctx, "https://invalid.example.com/nonexistent.git", "v1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to clone")
}

func TestDownload_CancelledContext(t *testing.T) {
	cfg := &config.ScannerConfig{TempDir: os.TempDir()}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	e := NewSonarScannerEntity(cfg, logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := e.Download(ctx, "https://github.com/SonarSource/sonar-scanner-cli.git", "5.0.1.3006")
	assert.Error(t, err)
}

func TestDownload_EmptyTempDir(t *testing.T) {
	cfg := &config.ScannerConfig{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	e := NewSonarScannerEntity(cfg, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// With empty tempDir, it should use os.TempDir()
	_, err := e.Download(ctx, "https://invalid.example.com/repo.git", "v1.0")
	assert.Error(t, err) // Will fail on clone, but tempDir should be set
	assert.Equal(t, os.TempDir(), e.tempDir)
}

func TestFindScannerExecutable_Found(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "scanner_find")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cfg := &config.ScannerConfig{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	e := NewSonarScannerEntity(cfg, logger)

	// Create bin directory with scanner executable
	binDir := filepath.Join(tmpDir, "bin")
	os.MkdirAll(binDir, 0755)
	scannerFile := filepath.Join(binDir, "sonar-scanner")
	os.WriteFile(scannerFile, []byte("#!/bin/sh\n"), 0755)

	path, err := e.findScannerExecutable(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, scannerFile, path)
}

func TestFindScannerExecutable_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "scanner_find_empty")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	e := newTestEntity()
	_, err = e.findScannerExecutable(tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestFindScannerExecutable_BatFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "scanner_find_bat")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	e := newTestEntity()

	batFile := filepath.Join(tmpDir, "sonar-scanner.bat")
	os.WriteFile(batFile, []byte("@echo off\n"), 0755)

	path, err := e.findScannerExecutable(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, batFile, path)
}
