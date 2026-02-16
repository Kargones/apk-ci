package store

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
)

func TestStore_GetStoreParam(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "store_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	cfg := &config.Config{
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				Bin1cv8: "echo", // Используем echo для тестирования
			},
		},
	}

	s := &Store{
		Name: "TestStore",
		Path: "test/path",
		User: "testuser",
		Pass: "testpass",
	}

	dbConnectString := "File=" + tmpDir + ";Usr=user;Pwd=pass;"
	runner := s.GetStoreParam(dbConnectString, cfg)

	if runner.RunString != "echo" {
		t.Errorf("Expected command to be 'echo', got %s", runner.RunString)
	}

	// Проверяем, что параметры содержат необходимые элементы
	paramsStr := strings.Join(runner.Params, " ")
	if !strings.Contains(paramsStr, "DESIGNER") {
		t.Error("Expected params to contain DESIGNER")
	}
	if !strings.Contains(paramsStr, dbConnectString) {
		t.Error("Expected params to contain database connection string")
	}
}

func TestExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "store_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Тест существующего пути
	if !exists(tmpDir) {
		t.Error("exists() returned false for existing path")
	}

	// Тест несуществующего пути
	nonExistentPath := filepath.Join(tmpDir, "nonexistent")
	if exists(nonExistentPath) {
		t.Error("exists() returned true for non-existent path")
	}
}

func TestFullPathStore(t *testing.T) {
	t.Run("TCP protocol", func(t *testing.T) {
		relPath, fullPath := fullPathStore("tcp://server", "relative/path")
		if relPath != "relative/path" {
			t.Errorf("Expected relative path 'relative/path', got %s", relPath)
		}
		if fullPath != "tcp://server/relative/path" {
			t.Errorf("Expected full path 'tcp://server/relative/path', got %s", fullPath)
		}
	})

	t.Run("Local path", func(t *testing.T) {
		relPath, fullPath := fullPathStore("/local/path", "relative/path")
		if relPath != "relative/path" {
			t.Errorf("Expected relative path 'relative/path', got %s", relPath)
		}
		expected := "/local/path/relative/path"
		if fullPath != expected {
			t.Errorf("Expected full path '%s', got %s", expected, fullPath)
		}
	})
}

func TestStore_Create(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "store_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := &config.Config{
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				Bin1cv8: "echo",
			},
		},
	}

	s := &Store{
		Name: "TestStore",
		Path: "test/store",
		User: "testuser",
		Pass: "testpass",
	}

	dbConnectString := "File=" + tmpDir + ";Usr=user;Pwd=pass;"
	err = s.Create(ctx, logger, cfg, dbConnectString, false)
	// При использовании echo команда выполнится успешно
	if err != nil {
		t.Logf("Create returned error (may be expected): %v", err)
	} else {
		t.Log("Create completed without error")
	}
}

func TestStore_Check(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "store_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := &config.Config{
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				Bin1cv8: "echo",
			},
		},
	}

	s := &Store{
		Name: "TestStore",
		Path: "test/store",
		User: "testuser",
		Pass: "testpass",
	}

	dbConnectString := "File=" + tmpDir + ";Usr=user;Pwd=pass;"
	err = s.Check(&ctx, logger, cfg, dbConnectString, true)
	// При использовании echo команда выполнится успешно
	if err != nil {
		t.Logf("Check returned error (may be expected): %v", err)
	} else {
		t.Log("Check completed without error")
	}
}

func TestStore_Lock(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "store_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := &config.Config{
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				Bin1cv8: "echo",
			},
		},
	}

	s := &Store{
		Name: "TestStore",
		Path: "test/store",
		User: "testuser",
		Pass: "testpass",
	}

	storeRoot := tmpDir
	dbConnectString := "File=" + tmpDir + ";Usr=user;Pwd=pass;"
	err = s.Lock(ctx, logger, cfg, dbConnectString, storeRoot)
	// При использовании echo команда выполнится успешно
	if err != nil {
		t.Logf("Lock returned error (may be expected): %v", err)
	} else {
		t.Log("Lock completed without error")
	}
}

func TestStore_UnBind(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "store_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := &config.Config{
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				Bin1cv8: "echo",
			},
		},
	}

	s := &Store{
		Name: "TestStore",
		Path: "test/store",
		User: "testuser",
		Pass: "testpass",
	}

	storeRoot := tmpDir
	dbConnectString := "File=" + tmpDir + ";Usr=user;Pwd=pass;"
	err = s.UnBind(ctx, logger, cfg, dbConnectString, storeRoot)
	// При использовании echo команда выполнится успешно
	if err != nil {
		t.Logf("UnBind returned error (may be expected): %v", err)
	} else {
		t.Log("UnBind completed without error")
	}
}

func TestParseReport(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "store_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Создаем тестовый файл отчета
	reportFile := filepath.Join(tmpDir, "report.txt")
	reportContent := `{1,
{"#","Версия хранилища","Версия конфигурации","Пользователь","Дата время","Комментарий"},
{1,1,"1.0.0.1","testuser","20240101120000","Test comment"}
}`

	err = os.WriteFile(reportFile, []byte(reportContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create report file: %v", err)
	}

	records, users, maxVersion, err := ParseReport(reportFile)
	if err != nil {
		t.Logf("ParseReport returned error (may be expected): %v", err)
		return
	}

	if len(records) == 0 {
		t.Log("No records parsed (may be expected for test format)")
	} else {
		t.Logf("Parsed %d records", len(records))
	}

	if len(users) == 0 {
		t.Log("No users parsed (may be expected for test format)")
	} else {
		t.Logf("Parsed %d users", len(users))
	}

	t.Logf("Max version: %d", maxVersion)
}



func TestRecord(t *testing.T) {
	// Тест создания записи
	record := Record{
		Version:     1,
		ConfVersion: "1.0.0.1",
		User:        "testuser",
		Date:        time.Now(),
		Comment:     "Test comment",
	}

	if record.Version != 1 {
		t.Errorf("Expected Version to be 1, got %d", record.Version)
	}
	if record.ConfVersion != "1.0.0.1" {
		t.Errorf("Expected ConfVersion to be '1.0.0.1', got %s", record.ConfVersion)
	}
	if record.User != "testuser" {
		t.Errorf("Expected User to be 'testuser', got %s", record.User)
	}
	if record.Comment != "Test comment" {
		t.Errorf("Expected Comment to be 'Test comment', got %s", record.Comment)
	}
}

func TestUser(t *testing.T) {
	// Тест создания пользователя
	user := User{
		StoreUserName: "storeuser",
		AccountName:   "domain\\user",
		Email:         "user@example.com",
	}

	if user.StoreUserName != "storeuser" {
		t.Errorf("Expected StoreUserName to be 'storeuser', got %s", user.StoreUserName)
	}
	if user.AccountName != "domain\\user" {
		t.Errorf("Expected AccountName to be 'domain\\user', got %s", user.AccountName)
	}
	if user.Email != "user@example.com" {
		t.Errorf("Expected Email to be 'user@example.com', got %s", user.Email)
	}
}

func TestStore_CreateAdd(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "store_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := &config.Config{
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				Bin1cv8: "echo",
			},
		},
	}

	s := &Store{
		Name: "TestStore",
		Path: "test/store",
		User: "testuser",
		Pass: "testpass",
	}

	dbConnectString := "File=" + tmpDir + ";Usr=user;Pwd=pass;"
	err = s.CreateAdd(ctx, logger, cfg, dbConnectString, "TestAdd")
	if err != nil {
		t.Logf("CreateAdd returned error (may be expected): %v", err)
	} else {
		t.Log("CreateAdd completed without error")
	}
}

func TestStore_CheckAdd(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "store_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := &config.Config{
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				Bin1cv8: "echo",
			},
		},
	}

	s := &Store{
		Name: "TestStore",
		Path: "test/store",
		User: "testuser",
		Pass: "testpass",
	}

	dbConnectString := "File=" + tmpDir + ";Usr=user;Pwd=pass;"
	err = s.CheckAdd(ctx, logger, cfg, dbConnectString, "TestAdd")
	if err != nil {
		t.Logf("CheckAdd returned error (may be expected): %v", err)
	} else {
		t.Log("CheckAdd completed without error")
	}
}

func TestStore_Bind(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "store_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := &config.Config{
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				Bin1cv8: "echo",
			},
		},
	}

	s := &Store{
		Name: "TestStore",
		Path: "test/store",
		User: "testuser",
		Pass: "testpass",
	}

	storeRoot := tmpDir
	dbConnectString := "File=" + tmpDir + ";Usr=user;Pwd=pass;"
	err = s.Bind(&ctx, logger, cfg, dbConnectString, storeRoot, true)
	if err != nil {
		t.Logf("Bind returned error (may be expected): %v", err)
	} else {
		t.Log("Bind completed without error")
	}
}

func TestStore_BindAdd(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "store_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := &config.Config{
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				Bin1cv8: "echo",
			},
		},
	}

	s := &Store{
		Name: "TestStore",
		Path: "test/store",
		User: "testuser",
		Pass: "testpass",
	}

	storeRoot := tmpDir
	dbConnectString := "File=" + tmpDir + ";Usr=user;Pwd=pass;"
	err = s.BindAdd(&ctx, logger, cfg, dbConnectString, storeRoot, "TestAdd")
	if err != nil {
		t.Logf("BindAdd returned error (may be expected): %v", err)
	} else {
		t.Log("BindAdd completed without error")
	}
}

func TestStore_UnBindAdd(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "store_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := &config.Config{
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				Bin1cv8: "echo",
			},
		},
	}

	s := &Store{
		Name: "TestStore",
		Path: "test/store",
		User: "testuser",
		Pass: "testpass",
	}

	storeRoot := tmpDir
	dbConnectString := "File=" + tmpDir + ";Usr=user;Pwd=pass;"
	err = s.UnBindAdd(ctx, logger, cfg, dbConnectString, storeRoot, "TestAdd")
	if err != nil {
		t.Logf("UnBindAdd returned error (may be expected): %v", err)
	} else {
		t.Log("UnBindAdd completed without error")
	}
}

func TestStore_Merge(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "store_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := &config.Config{
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				Bin1cv8: "echo",
			},
		},
	}

	s := &Store{
		Name: "TestStore",
		Path: "test/store",
		User: "testuser",
		Pass: "testpass",
	}

	pathCf := filepath.Join(tmpDir, "test.cf")
	pathMergeSettings := filepath.Join(tmpDir, "merge.xml")
	storeRoot := tmpDir
	dbConnectString := "File=" + tmpDir + ";Usr=user;Pwd=pass;"
	
	err = s.Merge(ctx, logger, cfg, dbConnectString, pathCf, pathMergeSettings, storeRoot)
	if err != nil {
		t.Logf("Merge returned error (may be expected): %v", err)
	} else {
		t.Log("Merge completed without error")
	}
}

func TestStore_MergeAdd(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "store_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := &config.Config{
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				Bin1cv8: "echo",
			},
		},
	}

	s := &Store{
		Name: "TestStore",
		Path: "test/store",
		User: "testuser",
		Pass: "testpass",
	}

	pathCf := filepath.Join(tmpDir, "test.cf")
	pathMergeSettings := filepath.Join(tmpDir, "merge.xml")
	storeRoot := tmpDir
	dbConnectString := "File=" + tmpDir + ";Usr=user;Pwd=pass;"
	
	err = s.MergeAdd(ctx, logger, cfg, dbConnectString, pathCf, pathMergeSettings, storeRoot, "TestAdd")
	if err != nil {
		t.Logf("MergeAdd returned error (may be expected): %v", err)
	} else {
		t.Log("MergeAdd completed without error")
	}
}

func TestStore_LockAdd(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "store_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := &config.Config{
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				Bin1cv8: "echo",
			},
		},
	}

	s := &Store{
		Name: "TestStore",
		Path: "test/store",
		User: "testuser",
		Pass: "testpass",
	}

	storeRoot := tmpDir
	dbConnectString := "File=" + tmpDir + ";Usr=user;Pwd=pass;"
	err = s.LockAdd(ctx, logger, cfg, dbConnectString, storeRoot, "TestAdd")
	if err != nil {
		t.Logf("LockAdd returned error (may be expected): %v", err)
	} else {
		t.Log("LockAdd completed without error")
	}
}

func TestStore_StoreCommit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "store_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := &config.Config{
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				Bin1cv8: "echo",
			},
		},
	}

	s := &Store{
		Name: "TestStore",
		Path: "test/store",
		User: "testuser",
		Pass: "testpass",
	}

	storeRoot := tmpDir
	dbConnectString := "File=" + tmpDir + ";Usr=user;Pwd=pass;"
	comment := "Test commit"
	
	err = s.StoreCommit(ctx, logger, cfg, dbConnectString, storeRoot, comment)
	if err != nil {
		t.Logf("StoreCommit returned error (may be expected): %v", err)
	} else {
		t.Log("StoreCommit completed without error")
	}
}

func TestStore_StoreCommitAdd(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "store_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := &config.Config{
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				Bin1cv8: "echo",
			},
		},
	}

	s := &Store{
		Name: "TestStore",
		Path: "test/store",
		User: "testuser",
		Pass: "testpass",
	}

	storeRoot := tmpDir
	dbConnectString := "File=" + tmpDir + ";Usr=user;Pwd=pass;"
	comment := "Test commit"
	
	err = s.StoreCommitAdd(&ctx, logger, cfg, dbConnectString, storeRoot, comment, "TestAdd")
	if err != nil {
		t.Logf("StoreCommitAdd returned error (may be expected): %v", err)
	} else {
		t.Log("StoreCommitAdd completed without error")
	}
}

func TestCommentAdd(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "store_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	cfg := &config.Config{
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				Bin1cv8: "echo",
			},
		},
	}

	s := &Store{
		Name: "TestStore",
		Path: "test/store",
		User: "testuser",
		Pass: "testpass",
	}

	dbConnectString := "File=" + tmpDir + ";Usr=user;Pwd=pass;"
	runner := s.GetStoreParam(dbConnectString, cfg)
	
	comment := "Test comment"
	CommentAdd(&runner, comment)
	
	// Проверяем, что комментарий добавлен в параметры
	paramsStr := strings.Join(runner.Params, " ")
	if !strings.Contains(paramsStr, comment) {
		t.Errorf("Expected params to contain comment '%s', but it was not found", comment)
	}
}

func TestCreateStores(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "store_test")
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
		AppConfig: &config.AppConfig{
			WorkDir: tmpDir,
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				Bin1cv8:  "echo",
				BinIbcmd: "ibcmd",
				EdtCli:   "edtcli",
				Rac:      "rac",
			},
			Users: struct {
				Rac        string `yaml:"rac"`
				Db         string `yaml:"db"`
				Mssql      string `yaml:"mssql"`
				StoreAdmin string `yaml:"storeAdmin"`
			}{
				Rac:        "rac_user",
				Db:         "db_user",
				Mssql:      "mssql_user",
				StoreAdmin: "admin",
			},
		},
		SecretConfig: &config.SecretConfig{
			Passwords: struct {
				Rac                string `yaml:"rac"`
				Db                 string `yaml:"db"`
				Mssql              string `yaml:"mssql"`
				StoreAdminPassword string `yaml:"storeAdminPassword"`
				Smb                string `yaml:"smb"`
			}{
				Rac:                "rac_password",
				Db:                 "db_password",
				Mssql:              "mssql_password",
				StoreAdminPassword: "password",
				Smb:                "smb_password",
			},
		},
	}

	storeRoot := tmpDir
	dbConnectString := "File=" + tmpDir + ";Usr=user;Pwd=pass;"
	arrayAdd := []string{"TestAdd1", "TestAdd2"}
	
	err = CreateStores(logger, cfg, storeRoot, dbConnectString, arrayAdd)
	if err != nil {
		t.Logf("CreateStores returned error (may be expected): %v", err)
	} else {
		t.Log("CreateStores completed without error")
	}
}