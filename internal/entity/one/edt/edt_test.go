package edt

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/Kargones/apk-ci/internal/config"
)

func TestConvert_MustLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "edt_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := &config.Config{
		WorkDir: tmpDir,
	}

	c := &Convert{}
	err = c.MustLoad(logger, cfg)
	if err != nil {
		t.Logf("MustLoad returned error (expected): %v", err)
	} else {
		t.Log("MustLoad completed without error")
	}
}

func TestCli_Init(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "edt_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	cfg := &config.Config{
		WorkDir:   tmpDir,
		WorkSpace: tmpDir,
		PathOut:   tmpDir,
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				EdtCli: "echo", // Используем echo для тестирования
			},
		},
	}

	cli := &Cli{}
	cli.Init(cfg)

	if cli.CliPath != "echo" {
		t.Errorf("Expected CliPath to be 'echo', got %s", cli.CliPath)
	}
	if cli.WorkSpace != tmpDir {
		t.Errorf("Expected WorkSpace to be %s, got %s", tmpDir, cli.WorkSpace)
	}
	if cli.PathOut != tmpDir {
		t.Errorf("Expected PathOut to be %s, got %s", tmpDir, cli.PathOut)
	}
}

func TestCli_Convert(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "edt_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Создаем тестовые директории
	pathIn := filepath.Join(tmpDir, "input")
	pathOut := filepath.Join(tmpDir, "output")
	workSpace := filepath.Join(tmpDir, "workspace")

	err = os.MkdirAll(pathIn, 0755)
	if err != nil {
		t.Fatalf("Failed to create input dir: %v", err)
	}
	err = os.MkdirAll(pathOut, 0755)
	if err != nil {
		t.Fatalf("Failed to create output dir: %v", err)
	}
	err = os.MkdirAll(workSpace, 0755)
	if err != nil {
		t.Fatalf("Failed to create workspace dir: %v", err)
	}

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := &config.Config{
		WorkDir: tmpDir,
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				EdtCli: "echo",
			},
		},
	}

	cli := &Cli{
		CliPath:   "echo",
		Direction: "xml2edt",
		PathIn:    pathIn,
		PathOut:   pathOut,
		WorkSpace: workSpace,
		Operation: "convert",
	}

	cli.Convert(ctx, logger, cfg)
	// При использовании echo команда выполнится успешно
	if cli.LastErr != nil {
		t.Logf("Convert returned error (may be expected): %v", cli.LastErr)
	} else {
		t.Log("Convert completed without error")
	}
}

func TestGetComment(t *testing.T) {
	c := &Convert{
		CommitSha1: "abc123",
		Source: Data{
			Format: "xml",
			Branch: "main",
		},
		Distination: Data{
			Format: "edt",
			Branch: "develop",
		},
	}

	comment := GetComment(c)
	if comment == "" {
		t.Error("Expected non-empty comment")
	}
	t.Logf("GetComment returned: %s", comment)
}

func TestExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "edt_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Тест существующего пути
	existsResult, err := exists(tmpDir)
	if err != nil {
		t.Errorf("exists() returned error for existing path: %v", err)
	}
	if !existsResult {
		t.Error("exists() returned false for existing path")
	}

	// Тест несуществующего пути
	existsResult, err = exists(filepath.Join(tmpDir, "nonexistent"))
	if err != nil {
		t.Errorf("exists() returned error for non-existent path: %v", err)
	}
	if existsResult {
		t.Error("exists() returned true for non-existent path")
	}
}

func TestMoveDirContents(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "edt_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Создаем исходную и целевую директории
	srcDir := filepath.Join(tmpDir, "src")
	dstDir := filepath.Join(tmpDir, "dst")

	err = os.MkdirAll(srcDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create src dir: %v", err)
	}
	err = os.MkdirAll(dstDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create dst dir: %v", err)
	}

	// Создаем тестовый файл в исходной директории
	testFile := filepath.Join(srcDir, "test.txt")
	file, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if _, err := file.WriteString("test content"); err != nil {
		t.Fatalf("Failed to write to test file: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Failed to close test file: %v", err)
	}

	// Перемещаем содержимое
	err = MoveDirContents(srcDir, dstDir)
	if err != nil {
		t.Errorf("MoveDirContents returned error: %v", err)
	}

	// Проверяем, что файл переместился
	movedFile := filepath.Join(dstDir, "test.txt")
	if _, err := os.Stat(movedFile); os.IsNotExist(err) {
		t.Error("File was not moved to destination")
	}
}

func TestCleanDirectoryPreservingHidden(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "edt_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Создаем тестовые файлы
	regularFile := filepath.Join(tmpDir, "regular.txt")
	hiddenFile := filepath.Join(tmpDir, ".hidden")

	file, err := os.Create(regularFile)
	if err != nil {
		t.Fatalf("Failed to create regular file: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Failed to close regular file: %v", err)
	}

	file, err = os.Create(hiddenFile)
	if err != nil {
		t.Fatalf("Failed to create hidden file: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Failed to close hidden file: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	err = cleanDirectoryPreservingHidden(tmpDir, logger)
	if err != nil {
		t.Errorf("cleanDirectoryPreservingHidden returned error: %v", err)
	}

	// Проверяем, что обычный файл удален
	if _, err := os.Stat(regularFile); !os.IsNotExist(err) {
		t.Error("Regular file was not deleted")
	}

	// Проверяем, что скрытый файл сохранен
	if _, err := os.Stat(hiddenFile); os.IsNotExist(err) {
		t.Error("Hidden file was deleted")
	}
}