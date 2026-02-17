package sonarqube

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Download clones the sonar-scanner repository from the specified URL.
func (s *SonarScannerEntity) Download(ctx context.Context, scannerURL string, scannerVersion string) (string, error) {
	s.logger.Debug("Cloning sonar-scanner repository", "scannerURL", scannerURL, "scannerVersion", scannerVersion)

	if s.tempDir == "" {
		s.tempDir = os.TempDir()
	}

	if err := os.MkdirAll(s.tempDir, 0750); err != nil {
		return "", fmt.Errorf("failed to create parent directories: %w", err)
	}

	cloneDir, err := os.MkdirTemp(s.tempDir, "sonar-scanner-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	// #nosec G204 - scannerVersion и scannerURL валидируются в вызывающем коде
	cmd := exec.CommandContext(ctx, "git", "clone", "--branch", scannerVersion, "--depth", "1", scannerURL, cloneDir) // #nosec G204 - git is hardcoded, scannerVersion/URL from validated config
	cmd.Stdout = nil
	cmd.Stderr = nil

	err = cmd.Run()
	if err != nil {
		if removeErr := os.RemoveAll(cloneDir); removeErr != nil {
			s.logger.Warn("Failed to remove clone directory after error", "path", cloneDir, "error", removeErr)
		}
		return "", fmt.Errorf("failed to clone scanner repository: %w", err)
	}

	scannerPath, err := s.findScannerExecutable(cloneDir)
	if err != nil {
		if removeErr := os.RemoveAll(cloneDir); removeErr != nil {
			s.logger.Warn("Failed to remove clone directory after error", "path", cloneDir, "error", removeErr)
		}
		return "", fmt.Errorf("failed to find scanner executable: %w", err)
	}

	s.scannerPath = scannerPath
	s.logger.Debug("Scanner path set", "scannerPath", scannerPath)
	s.logger.Debug("Sonar-scanner repository cloned successfully", "cloneDir", cloneDir)
	return cloneDir, nil
}

// findScannerExecutable finds the scanner executable in the extracted directory.
func (s *SonarScannerEntity) findScannerExecutable(dir string) (string, error) {
	var scannerPath string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && (info.Name() == "sonar-scanner" || info.Name() == "sonar-scanner.bat") {
			if !strings.HasPrefix(path, dir) {
				return fmt.Errorf("invalid scanner path: %s", path)
			}
			scannerPath = path
			return filepath.SkipDir
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("error while searching for scanner executable: %w", err)
	}

	if scannerPath == "" {
		return "", fmt.Errorf("scanner executable not found in extracted files")
	}

	return scannerPath, nil
}
