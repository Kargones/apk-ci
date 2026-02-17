package onec

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTempDbCreator_CreateTempDB_WithValidOptions(t *testing.T) {
	creator := NewTempDbCreator()

	// Create a temp directory for the test
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "testdb")

	result, err := creator.CreateTempDB(context.Background(), CreateTempDBOptions{
		DbPath:   dbPath,
		BinIbcmd: "/nonexistent/ibcmd",
		Timeout:  1 * time.Second,
	})

	// Will fail because ibcmd doesn't exist
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestTempDbCreator_CleanupDb(t *testing.T) {
	creator := NewTempDbCreator()
	logger := slog.Default()

	// Create a temp directory
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "testdb")
	require.NoError(t, os.MkdirAll(testPath, 0755))

	// Cleanup should remove the directory
	err := creator.cleanupDb(testPath, logger)
	require.NoError(t, err)

	// Directory should be gone
	_, statErr := os.Stat(testPath)
	assert.True(t, os.IsNotExist(statErr))
}

func TestTempDbCreator_CleanupDb_NonexistentPath(t *testing.T) {
	creator := NewTempDbCreator()
	logger := slog.Default()

	// Cleanup of nonexistent path should not error
	err := creator.cleanupDb("/nonexistent/path/12345", logger)
	require.NoError(t, err)
}

func TestTempDbCreator_CreateInfobase_EmptyBinIbcmd(t *testing.T) {
	creator := NewTempDbCreator()
	logger := slog.Default()

	err := creator.createInfobase(context.Background(), CreateTempDBOptions{
		DbPath:   "/tmp/testdb",
		BinIbcmd: "",
		Timeout:  1 * time.Second,
	}, logger)

	require.Error(t, err)
}

func TestTempDbCreator_CreateInfobase_TimeoutContext(t *testing.T) {
	creator := NewTempDbCreator()
	logger := slog.Default()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond)

	err := creator.createInfobase(ctx, CreateTempDBOptions{
		DbPath:   "/tmp/testdb",
		BinIbcmd: "/usr/bin/ibcmd",
		Timeout:  5 * time.Second,
	}, logger)

	require.Error(t, err)
}

func TestTempDbCreator_AddExtension_EmptyBinIbcmd(t *testing.T) {
	creator := NewTempDbCreator()
	logger := slog.Default()

	err := creator.addExtension(context.Background(), CreateTempDBOptions{
		DbPath:   "/tmp/testdb",
		BinIbcmd: "",
		Timeout:  1 * time.Second,
	}, "testext", logger)

	require.Error(t, err)
}

func TestTempDbCreator_AddExtension_TimeoutContext(t *testing.T) {
	creator := NewTempDbCreator()
	logger := slog.Default()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond)

	err := creator.addExtension(ctx, CreateTempDBOptions{
		DbPath:   "/tmp/testdb",
		BinIbcmd: "/usr/bin/ibcmd",
		Timeout:  5 * time.Second,
	}, "testext", logger)

	require.Error(t, err)
}

func TestTempDbCreator_CreateTempDB_WithExtensions(t *testing.T) {
	creator := NewTempDbCreator()

	// Create a temp directory for the test
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "testdb")

	result, err := creator.CreateTempDB(context.Background(), CreateTempDBOptions{
		DbPath:     dbPath,
		BinIbcmd:   "/nonexistent/ibcmd",
		Extensions: []string{"ext1", "ext2"},
		Timeout:    1 * time.Second,
	})

	// Will fail because ibcmd doesn't exist
	require.Error(t, err)
	assert.Nil(t, result)
}
