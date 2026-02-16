package sonarqube

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// executeOnce executes the sonar-scanner with the provided context.
func (s *SonarScannerEntity) executeOnce(ctx context.Context) (*ScanResult, error) {
	s.logger.Debug("Executing sonar-scanner")

	if err := s.Initialize(); err != nil {
		return nil, fmt.Errorf("scanner initialization failed: %w", err)
	}

	if err := s.preProcessBSLFiles(); err != nil {
		s.logger.Warn("BSL preprocessing failed", "error", err)
	}

	execCtx := ctx
	if s.config.Timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(execCtx, s.scannerPath) // #nosec G204 - s.scannerPath is validated

	if s.config.WorkDir != "" {
		cmd.Dir = s.config.WorkDir
	}

	env := os.Environ()
	if s.config.JavaOpts != "" {
		env = append(env, "JAVA_OPTS="+s.config.JavaOpts)
	}
	cmd.Env = env

	args := make([]string, 0, len(s.properties))
	for key, value := range s.properties {
		args = append(args, fmt.Sprintf("-D%s=%s", key, value))
	}
	cmd.Args = append(cmd.Args, args...)

	s.logger.Debug("Starting scanner execution",
		"command", s.scannerPath,
		"args", args,
		"workDir", s.workDir,
		"timeout", s.config.Timeout)

	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	result := &ScanResult{
		Success:  err == nil,
		Duration: duration,
		Errors:   make([]string, 0),
		Metrics:  make(map[string]string),
	}

	if len(output) > 0 {
		if parseErr := s.parseOutput(string(output), result); parseErr != nil {
			s.logger.Warn("Failed to parse scanner output", "error", parseErr)
		}
	}

	if err != nil {
		return s.handleExecutionError(err, string(output), result)
	}

	s.logger.Debug("Sonar-scanner executed successfully",
		"duration", duration,
		"analysisId", result.AnalysisID,
		"projectKey", result.ProjectKey)

	return result, nil
}

// KillProcess forcefully terminates the scanner process if it's running.
func (s *SonarScannerEntity) KillProcess(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	s.logger.Debug("Attempting to kill scanner process", "pid", cmd.Process.Pid)

	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		s.logger.Warn("Failed to send interrupt signal", "error", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(5 * time.Second):
		s.logger.Warn("Graceful shutdown failed, force killing process")
		if err := cmd.Process.Kill(); err != nil {
			s.logger.Error("Failed to kill process", "error", err)
			return fmt.Errorf("failed to kill scanner process: %w", err)
		}
		s.logger.Debug("Scanner process killed successfully")
	case err := <-done:
		if err != nil {
			s.logger.Debug("Scanner process terminated", "error", err)
		} else {
			s.logger.Debug("Scanner process terminated gracefully")
		}
	}

	return nil
}

// ExecuteWithTimeout executes the scanner with enhanced timeout and process management.
func (s *SonarScannerEntity) ExecuteWithTimeout(ctx context.Context, timeout time.Duration) (*ScanResult, error) {
	s.logger.Debug("Executing sonar-scanner with timeout", "timeout", timeout)

	if err := s.Initialize(); err != nil {
		return nil, fmt.Errorf("scanner initialization failed: %w", err)
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, s.scannerPath) // #nosec G204 - s.scannerPath is validated

	if s.workDir != "" {
		cmd.Dir = s.workDir
	}

	env := os.Environ()
	if s.config.JavaOpts != "" {
		env = append(env, "JAVA_OPTS="+s.config.JavaOpts)
	}
	cmd.Env = env

	args := make([]string, 0, len(s.properties))
	for key, value := range s.properties {
		args = append(args, fmt.Sprintf("-D%s=%s", key, value))
	}
	cmd.Args = append(cmd.Args, args...)

	s.logger.Debug("Starting scanner execution with timeout",
		"command", s.scannerPath,
		"args", args,
		"workDir", s.workDir,
		"timeout", timeout)

	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	defer func() {
		if cmd.Process != nil {
			if killErr := s.KillProcess(cmd); killErr != nil {
				s.logger.Warn("Failed to cleanup process", "error", killErr)
			}
		}
	}()

	result := &ScanResult{
		Success:  err == nil,
		Duration: duration,
		Errors:   make([]string, 0),
		Metrics:  make(map[string]string),
	}

	if len(output) > 0 {
		if parseErr := s.parseOutput(string(output), result); parseErr != nil {
			s.logger.Warn("Failed to parse scanner output", "error", parseErr)
		}
	}

	if err != nil {
		return s.handleExecutionError(err, string(output), result)
	}

	s.logger.Debug("Sonar-scanner executed successfully with timeout",
		"duration", duration,
		"analysisId", result.AnalysisID,
		"projectKey", result.ProjectKey)

	return result, nil
}
