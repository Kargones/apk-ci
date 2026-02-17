package sonarqube

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteOnce_NoInit(t *testing.T) {
	e := newTestEntity()
	_, err := e.executeOnce(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "initialization failed")
}

func TestExecuteOnce_InvalidScanner(t *testing.T) {
	e := newTestEntity()
	e.scannerPath = "/nonexistent"
	e.workDir = os.TempDir()
	_, err := e.executeOnce(context.Background())
	assert.Error(t, err)
}

func TestExecuteOnce_WithWorkingScript(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "exec_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a fake scanner script that outputs expected format
	script := tmpDir + "/scanner.sh"
	os.WriteFile(script, []byte(`#!/bin/sh
echo "INFO: Project key: test-project"
echo "INFO: ANALYSIS SUCCESSFUL, you can browse task?id=TEST123"
echo "Total time: 5.2 s"
echo "Final Memory: 64M/256M"
echo "Coverage 75.0%"
echo "3 issues found"
`), 0755)

	cfg := &config.ScannerConfig{
		WorkDir: tmpDir,
		TempDir: tmpDir,
		Timeout: 30 * time.Second,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	e := NewSonarScannerEntity(cfg, logger)
	e.scannerPath = script
	e.workDir = tmpDir

	result, err := e.executeOnce(context.Background())
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "TEST123", result.AnalysisID)
	assert.Equal(t, "test-project", result.ProjectKey)
	assert.Equal(t, "75.0", result.Metrics["coverage"])
}

func TestExecuteOnce_FailingScript(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "exec_fail_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	script := tmpDir + "/scanner.sh"
	os.WriteFile(script, []byte("#!/bin/sh\necho 'ERROR: Analysis failed'\nexit 1\n"), 0755)

	cfg := &config.ScannerConfig{WorkDir: tmpDir, TempDir: tmpDir}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	e := NewSonarScannerEntity(cfg, logger)
	e.scannerPath = script
	e.workDir = tmpDir

	result, err := e.executeOnce(context.Background())
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
}

func TestExecuteOnce_WithTimeout(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "exec_timeout_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	script := tmpDir + "/scanner.sh"
	os.WriteFile(script, []byte("#!/bin/sh\nsleep 10\n"), 0755)

	cfg := &config.ScannerConfig{WorkDir: tmpDir, TempDir: tmpDir, Timeout: 100 * time.Millisecond}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	e := NewSonarScannerEntity(cfg, logger)
	e.scannerPath = script
	e.workDir = tmpDir

	result, err := e.executeOnce(context.Background())
	assert.Error(t, err)
	assert.NotNil(t, result)
}

func TestExecuteOnce_WithProperties(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "exec_props_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	script := tmpDir + "/scanner.sh"
	os.WriteFile(script, []byte("#!/bin/sh\necho 'INFO: ANALYSIS SUCCESSFUL, task?id=X1'\n"), 0755)

	cfg := &config.ScannerConfig{WorkDir: tmpDir, TempDir: tmpDir, JavaOpts: "-Xmx256m"}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	e := NewSonarScannerEntity(cfg, logger)
	e.scannerPath = script
	e.workDir = tmpDir
	e.SetProperty("sonar.projectKey", "test")

	result, err := e.executeOnce(context.Background())
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestKillProcess_Nil(t *testing.T) {
	e := newTestEntity()
	assert.NoError(t, e.KillProcess(nil))
}

func TestKillProcess_NoProcess(t *testing.T) {
	e := newTestEntity()
	cmd := &exec.Cmd{}
	assert.NoError(t, e.KillProcess(cmd))
}

func TestKillProcess_RunningProcess(t *testing.T) {
	e := newTestEntity()
	cmd := exec.Command("sleep", "60")
	require.NoError(t, cmd.Start())

	err := e.KillProcess(cmd)
	assert.NoError(t, err)
}

func TestExecuteWithTimeout_NoInit(t *testing.T) {
	e := newTestEntity()
	_, err := e.ExecuteWithTimeout(context.Background(), time.Second)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "initialization failed")
}

func TestExecuteWithTimeout_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ewt_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	script := tmpDir + "/scanner.sh"
	os.WriteFile(script, []byte("#!/bin/sh\necho 'INFO: ANALYSIS SUCCESSFUL, task?id=T1'\n"), 0755)

	cfg := &config.ScannerConfig{WorkDir: tmpDir, TempDir: tmpDir}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	e := NewSonarScannerEntity(cfg, logger)
	e.scannerPath = script
	e.workDir = tmpDir

	result, err := e.ExecuteWithTimeout(context.Background(), 10*time.Second)
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestExecuteWithTimeout_Timeout(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ewt_timeout")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	script := tmpDir + "/scanner.sh"
	os.WriteFile(script, []byte("#!/bin/sh\nsleep 60\n"), 0755)

	cfg := &config.ScannerConfig{WorkDir: tmpDir, TempDir: tmpDir}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	e := NewSonarScannerEntity(cfg, logger)
	e.scannerPath = script
	e.workDir = tmpDir

	result, err := e.ExecuteWithTimeout(context.Background(), 200*time.Millisecond)
	assert.Error(t, err)
	assert.NotNil(t, result)
}

func TestExecuteWithTimeout_WithJavaOpts(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ewt_java")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	script := tmpDir + "/scanner.sh"
	os.WriteFile(script, []byte("#!/bin/sh\necho OK\n"), 0755)

	cfg := &config.ScannerConfig{WorkDir: tmpDir, TempDir: tmpDir, JavaOpts: "-Xmx512m"}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	e := NewSonarScannerEntity(cfg, logger)
	e.scannerPath = script
	e.workDir = tmpDir

	result, err := e.ExecuteWithTimeout(context.Background(), 10*time.Second)
	require.NoError(t, err)
	assert.True(t, result.Success)
}
