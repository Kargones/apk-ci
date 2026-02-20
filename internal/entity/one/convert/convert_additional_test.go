package convert

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/entity/one/designer"
	"github.com/Kargones/apk-ci/internal/entity/one/store"
)

// TestExistsFunction проверяет функцию exists
func TestExistsFunction(t *testing.T) {
	tmpDir := t.TempDir()

	// Создаем тестовый файл
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"existing file", testFile, true},
		{"existing directory", tmpDir, true},
		{"non-existing path", filepath.Join(tmpDir, "nonexistent"), false},
		{"empty path", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := exists(tt.path)
			if result != tt.expected {
				t.Errorf("exists(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// TestConfig_SaveAdditional проверяет сохранение конфигурации в файл
func TestConfig_SaveAdditional(t *testing.T) {
	tmpDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	cfg := &config.Config{
		WorkDir: tmpDir,
	}

	convertConfig := &Config{
		StoreRoot: "test/store/root",
		OneDB: designer.OneDb{
			DbConnectString: "/S server\\db",
			User:            "testuser",
			Pass:            "testpass",
		},
		Pair: []Pair{
			{
				Source: Source{
					Name:    "Main",
					Main:    true,
					RelPath: "src/main",
				},
				Store: store.Store{
					Name: "Main",
				},
			},
		},
	}

	configPath := filepath.Join(tmpDir, "convert_config_additional.json")
	err := convertConfig.Save(ctx, logger, cfg, configPath)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Проверяем, что файл создан
	if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
		t.Errorf("Config file was not created at %s", configPath)
	}

	// Проверяем содержимое файла
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Config file is empty")
	}
}

// TestConfig_InitDb_ExistingDb проверяет InitDb с существующей базой
func TestConfig_InitDb_ExistingDb(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()
	tmpDir := t.TempDir()

	cfg := &config.Config{
		WorkDir: tmpDir,
	}

	convertConfig := &Config{
		OneDB: designer.OneDb{
			DbExist: true,
		},
	}

	err := convertConfig.InitDb(ctx, logger, cfg)
	if err != nil {
		t.Errorf("InitDb() with existing DB should not return error, got: %v", err)
	}
}

// TestConfig_InitDb_ServerDb проверяет InitDb с серверной базой
func TestConfig_InitDb_ServerDb(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()
	tmpDir := t.TempDir()

	cfg := &config.Config{
		WorkDir: tmpDir,
	}

	convertConfig := &Config{
		OneDB: designer.OneDb{
			ServerDb: true,
		},
	}

	err := convertConfig.InitDb(ctx, logger, cfg)
	if err != nil {
		t.Errorf("InitDb() with server DB should not return error, got: %v", err)
	}
}

// TestPair_SourceName проверяет поля Source
func TestPair_SourceName(t *testing.T) {
	pair := Pair{
		Source: Source{
			Name:    "TestExtension",
			Main:    false,
			RelPath: "src/extension",
		},
	}

	if pair.Source.Name != "TestExtension" {
		t.Errorf("Source.Name = %q, want %q", pair.Source.Name, "TestExtension")
	}
	if pair.Source.Main {
		t.Error("Source.Main should be false")
	}
	if pair.Source.RelPath != "src/extension" {
		t.Errorf("Source.RelPath = %q, want %q", pair.Source.RelPath, "src/extension")
	}
}

// TestConfig_EmptyPair проверяет поведение с пустым списком Pair
func TestConfig_EmptyPair(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()
	tmpDir := t.TempDir()

	cfg := &config.Config{
		WorkDir: tmpDir,
	}

	convertConfig := &Config{
		OneDB: designer.OneDb{
			DbExist: true,
		},
		Pair: []Pair{},
	}

	// Операции с пустым списком Pair должны проходить без ошибок
	err := convertConfig.StoreLock(ctx, logger, cfg)
	if err != nil {
		t.Errorf("StoreLock() with empty Pair should not return error, got: %v", err)
	}

	err = convertConfig.StoreUnBind(ctx, logger, cfg)
	if err != nil {
		t.Errorf("StoreUnBind() with empty Pair should not return error, got: %v", err)
	}

	err = convertConfig.DbUpdate(ctx, logger, cfg)
	if err != nil {
		t.Errorf("DbUpdate() with empty Pair should not return error, got: %v", err)
	}

	err = convertConfig.StoreCommit(ctx, logger, cfg)
	if err != nil {
		t.Errorf("StoreCommit() with empty Pair should not return error, got: %v", err)
	}
}
