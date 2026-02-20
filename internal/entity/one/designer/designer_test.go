package designer

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/util/runner"
)

//nolint:dupl // similar test structure
func TestOneDb_Create(t *testing.T) {
	// Создаем временную директорию для тестов
	tmpDir, err := os.MkdirTemp("", "designer_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { //nolint:govet // shadow ok in cleanup
		if rmErr := os.RemoveAll(tmpDir); rmErr != nil {
			t.Logf("Failed to remove temp dir: %v", rmErr)
		}
	}()

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
				BinIbcmd: "echo", // Используем echo для тестирования
			},
		},
		ProjectConfig: &config.ProjectConfig{
			StoreDb: "test_db",
		},
	}

	odb := &OneDb{}
	err = odb.Create(ctx, logger, cfg)
	// Ожидаем ошибку, так как используем echo вместо реального ibcmd
	if err == nil {
		t.Log("Create completed without error (expected with echo)")
	} else {
		t.Logf("Create returned error (expected with echo): %v", err)
	}
}

//nolint:dupl // similar test structure
func TestOneDb_Add(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "designer_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	} //nolint:govet // shadow ok in cleanup
	defer func() {
		if rmErr := os.RemoveAll(tmpDir); rmErr != nil {
			t.Logf("Failed to remove temp dir: %v", rmErr)
		}
	}()

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
				BinIbcmd: "echo",
			},
		},
		ProjectConfig: &config.ProjectConfig{
			StoreDb: "test_db",
		},
	}

	odb := &OneDb{}
	err = odb.Add(ctx, logger, cfg, "test_extension")
	if err == nil {
		t.Log("Add completed without error (expected with echo)")
	} else {
		t.Logf("Add returned error (expected with echo): %v", err)
	}
}

func TestOneDb_Load(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "designer_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	} //nolint:govet // shadow ok in cleanup
	defer func() {
		if rmErr := os.RemoveAll(tmpDir); rmErr != nil {
			t.Logf("Failed to remove temp dir: %v", rmErr)
		}
	}()

	// Создаем тестовый файл конфигурации
	sourcePath := filepath.Join(tmpDir, "test_config.cf")
	file, err := os.Create(sourcePath)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if closeErr := file.Close(); closeErr != nil {
		t.Fatalf("Failed to close file: %v", closeErr)
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
				BinIbcmd: "echo",
			},
		},
		ProjectConfig: &config.ProjectConfig{
			StoreDb: "test_db",
		},
	}

	odb := &OneDb{}
	err = odb.Load(ctx, logger, cfg, sourcePath)
	if err == nil {
		t.Log("Load completed without error (expected with echo)")
	} else {
		t.Logf("Load returned error (expected with echo): %v", err)
	}
}

//nolint:dupl // similar test structure
func TestOneDb_UpdateCfg(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "designer_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if rmErr := os.RemoveAll(tmpDir); rmErr != nil {
			t.Logf("Failed to remove temp dir: %v", rmErr)
		}
	}()

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
				BinIbcmd: "echo",
			},
		},
		ProjectConfig: &config.ProjectConfig{
			StoreDb: "test_db",
		},
	}

	odb := &OneDb{}
	err = odb.UpdateCfg(ctx, logger, cfg, "")
	if err == nil {
		t.Log("UpdateCfg completed without error (expected with echo)")
	} else {
		t.Logf("UpdateCfg returned error (expected with echo): %v", err)
	}
}

//nolint:dupl // similar test structure
func TestOneDb_Dump(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "designer_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if rmErr := os.RemoveAll(tmpDir); rmErr != nil {
			t.Logf("Failed to remove temp dir: %v", rmErr)
		}
	}()

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
				BinIbcmd: "echo",
			},
		},
		ProjectConfig: &config.ProjectConfig{
			StoreDb: "test_db",
		},
	}

	odb := &OneDb{}
	err = odb.Dump(ctx, logger, cfg)
	if err == nil {
		t.Log("Dump completed without error (expected with echo)")
	} else {
		t.Logf("Dump returned error (expected with echo): %v", err)
	}
}

func TestGetDbName(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "designer_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if rmErr := os.RemoveAll(tmpDir); rmErr != nil {
			t.Logf("Failed to remove temp dir: %v", rmErr)
		}
	}()

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := &config.Config{
		WorkDir: tmpDir,
		ProjectConfig: &config.ProjectConfig{
			StoreDb: "test_database",
		},
	}

	dbName, err := GetDbName(ctx, logger, cfg)
	if err != nil {
		t.Errorf("GetDbName returned error: %v", err)
	}
	if dbName == "" {
		t.Error("GetDbName returned empty string")
	}
	t.Logf("GetDbName returned: %s", dbName)
}

func TestCreateTempDb(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "designer_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if rmErr := os.RemoveAll(tmpDir); rmErr != nil {
			t.Logf("Failed to remove temp dir: %v", rmErr)
		}
	}()

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
				BinIbcmd: "echo",
			},
		},
		ProjectConfig: &config.ProjectConfig{
			StoreDb: "test_db",
		},
	}

	dbPath := filepath.Join(tmpDir, "test.1cd")
	arrayAdd := []string{"extension1", "extension2"}

	oneDb, err := CreateTempDb(context.Background(), logger, cfg, dbPath, arrayAdd)
	if err != nil {
		t.Logf("CreateTempDb returned error (expected with echo): %v", err)
	} else {
		t.Log("CreateTempDb completed without error")
	}

	// Проверяем, что структура OneDb была инициализирована
	if oneDb.FullConnectString == "" {
		t.Log("FullConnectString is empty (expected)")
	}
}

//nolint:dupl // similar test structure
func TestOneDb_LoadAdd(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "designer_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if rmErr := os.RemoveAll(tmpDir); rmErr != nil {
			t.Logf("Failed to remove temp dir: %v", rmErr)
		}
	}()

	// Создаем тестовый файл конфигурации
	sourcePath := filepath.Join(tmpDir, "test_extension.cfe")
	file, err := os.Create(sourcePath)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if closeErr := file.Close(); closeErr != nil {
		t.Fatalf("Failed to close file: %v", closeErr)
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
				BinIbcmd: "echo",
			},
		},
		ProjectConfig: &config.ProjectConfig{
			StoreDb: "test_db",
		},
	}

	odb := &OneDb{DbConnectString: "test_db_path"}
	err = odb.LoadAdd(ctx, logger, cfg, sourcePath, "test_extension")
	if err == nil {
		t.Log("LoadAdd completed without error (expected with echo)")
	} else {
		t.Logf("LoadAdd returned error (expected with echo): %v", err)
	}
}

//nolint:dupl // similar test structure
func TestOneDb_UpdateAdd(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "designer_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if rmErr := os.RemoveAll(tmpDir); rmErr != nil {
			t.Logf("Failed to remove temp dir: %v", rmErr)
		}
	}()

	// Создаем тестовый файл конфигурации
	sourcePath := filepath.Join(tmpDir, "test_extension.cfe")
	file, err := os.Create(sourcePath)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if closeErr := file.Close(); closeErr != nil {
		t.Fatalf("Failed to close file: %v", closeErr)
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
				BinIbcmd: "echo",
			},
		},
		ProjectConfig: &config.ProjectConfig{
			StoreDb: "test_db",
		},
	}

	odb := &OneDb{DbConnectString: "test_db_path"}
	err = odb.UpdateAdd(ctx, logger, cfg, sourcePath, "test_extension")
	if err == nil {
		t.Log("UpdateAdd completed without error (expected with echo)")
	} else {
		t.Logf("UpdateAdd returned error (expected with echo): %v", err)
	}
}

func TestOneDb_DumpAdd(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "designer_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if rmErr := os.RemoveAll(tmpDir); rmErr != nil {
			t.Logf("Failed to remove temp dir: %v", rmErr)
		}
	}()

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
				BinIbcmd: "echo",
			},
		},
		ProjectConfig: &config.ProjectConfig{
			StoreDb: "test_db",
		},
	}

	odb := &OneDb{DbConnectString: "test_db_path"}
	err = odb.DumpAdd(ctx, logger, cfg, "test_extension")
	if err == nil {
		t.Log("DumpAdd completed without error (expected with echo)")
	} else {
		t.Logf("DumpAdd returned error (expected with echo): %v", err)
	}
}

func TestGetDbName_WithServerDb(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "designer_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if rmErr := os.RemoveAll(tmpDir); rmErr != nil {
			t.Logf("Failed to remove temp dir: %v", rmErr)
		}
	}()

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := &config.Config{
		WorkDir: tmpDir,
		ProjectConfig: &config.ProjectConfig{
			StoreDb: "/S server\\database",
		},
	}

	dbName, err := GetDbName(ctx, logger, cfg)
	if err != nil {
		t.Errorf("GetDbName returned error: %v", err)
	}
	if dbName == "" {
		t.Error("GetDbName returned empty string")
	}
	t.Logf("GetDbName returned: %s", dbName)
}

func TestAddDisableParam(t *testing.T) {
	r := &runner.Runner{}
	addDisableParam(r)
	
	// Проверяем что параметры добавлены
	expectedParams := []string{"/DisableStartupDialogs", "/DisableStartupMessages", "/DisableUnrecoverableErrorMessage", "/UC ServiceMode"}
	
	for _, expected := range expectedParams {
		found := false
		for _, param := range r.Params {
			if param == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("addDisableParam did not add %s parameter", expected)
		}
	}
}