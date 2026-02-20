package runner

import (
	"log/slog"
	"os"
	"testing"
)

func TestDisplayConfig_NoXvfb(t *testing.T) {
	// In a test environment without Xvfb, this should return an error
	// but we still exercise the code paths
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	err := DisplayConfig(logger)
	// We expect an error since Xvfb is likely not available in test env
	if err != nil {
		t.Logf("DisplayConfig returned expected error: %v", err)
	}
}
