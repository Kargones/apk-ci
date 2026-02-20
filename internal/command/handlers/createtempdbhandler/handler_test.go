package createtempdbhandler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/onec"
	"github.com/Kargones/apk-ci/internal/adapter/onec/onectest"
	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
)

// generateManyExtensions генерирует строку с указанным количеством расширений.
// Используется для тестирования лимита maxExtensions (M-5 fix).
func generateManyExtensions(count int) string {
	exts := make([]string, count)
	for i := 0; i < count; i++ {
		exts[i] = fmt.Sprintf("ext%d", i)
	}
	return strings.Join(exts, ",")
}

func TestCreateTempDbHandler_Registration(t *testing.T) {
	// Проверяем что handler зарегистрирован через init()
	// Основная команда
	h, ok := command.Get(constants.ActNRCreateTempDb)
	if !ok {
		t.Errorf("Handler for %q not registered", constants.ActNRCreateTempDb)
		return
	}

	if h.Name() != constants.ActNRCreateTempDb {
		t.Errorf("Handler Name() = %q, want %q", h.Name(), constants.ActNRCreateTempDb)
	}

	// Deprecated alias
	aliasH, ok := command.Get(constants.ActCreateTempDb)
	if !ok {
		t.Errorf("Deprecated alias %q not registered", constants.ActCreateTempDb)
		return
	}

	// Alias должен возвращать то же имя что и основной handler
	// (через DeprecatedBridge.Name() -> deprecated name)
	if aliasH.Name() != constants.ActCreateTempDb {
		t.Errorf("Alias Name() = %q, want %q", aliasH.Name(), constants.ActCreateTempDb)
	}
}

func TestCreateTempDbHandler_Name(t *testing.T) {
	h := &CreateTempDbHandler{}
	got := h.Name()
	want := constants.ActNRCreateTempDb

	if got != want {
		t.Errorf("Name() = %q, want %q", got, want)
	}
}

func TestCreateTempDbHandler_Description(t *testing.T) {
	h := &CreateTempDbHandler{}
	got := h.Description()

	if got == "" {
		t.Error("Description() returned empty string")
	}

	// Должно содержать ключевые слова
	if !strings.Contains(got, "временн") && !strings.Contains(got, "базу данных") {
		t.Errorf("Description() = %q, expected to contain keywords about temporary database", got)
	}
}

func TestCreateTempDbHandler_Execute_Success_EmptyDB(t *testing.T) {
	// Подготовка: создаём временную директорию для теста
	tmpDir := t.TempDir()

	// H4 fix: создаём фейковый файл ibcmd для прохождения проверки существования
	ibcmdPath := filepath.Join(tmpDir, "ibcmd")
	if err := os.WriteFile(ibcmdPath, []byte("#!/bin/bash\necho 'mock'"), 0755); err != nil {
		t.Fatalf("Failed to create mock ibcmd: %v", err)
	}

	// Mock клиент
	mockCreator := onectest.NewMockTempDatabaseCreator()

	// Handler с mock
	h := &CreateTempDbHandler{
		dbCreator: mockCreator,
	}

	// Конфигурация
	cfg := &config.Config{
		TmpDir: tmpDir,
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				BinIbcmd: ibcmdPath,
			},
		},
	}

	// M2 fix: используем t.Setenv для автоматического cleanup
	t.Setenv("BR_EXTENSIONS", "")
	t.Setenv("BR_TTL_HOURS", "")
	t.Setenv("BR_OUTPUT_FORMAT", "")
	t.Setenv("BR_SHOW_PROGRESS", "false") // отключаем progress для тестов

	// Выполнение
	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	// Проверки
	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}

	if mockCreator.CreateTempDBCallCount != 1 {
		t.Errorf("CreateTempDB was called %d times, want 1", mockCreator.CreateTempDBCallCount)
	}

	// Проверяем что расширения пустые
	if len(mockCreator.LastCreateTempDBOptions.Extensions) != 0 {
		t.Errorf("Extensions = %v, want empty", mockCreator.LastCreateTempDBOptions.Extensions)
	}
}

func TestCreateTempDbHandler_Execute_Success_WithExtensions(t *testing.T) {
	tmpDir := t.TempDir()

	// H4 fix: создаём фейковый файл ibcmd
	ibcmdPath := filepath.Join(tmpDir, "ibcmd")
	if err := os.WriteFile(ibcmdPath, []byte("#!/bin/bash\necho 'mock'"), 0755); err != nil {
		t.Fatalf("Failed to create mock ibcmd: %v", err)
	}

	mockCreator := onectest.NewMockTempDatabaseCreator()

	h := &CreateTempDbHandler{
		dbCreator: mockCreator,
	}

	cfg := &config.Config{
		TmpDir: tmpDir,
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				BinIbcmd: ibcmdPath,
			},
		},
	}

	// M2 fix: используем t.Setenv
	t.Setenv("BR_EXTENSIONS", "ext1,ext2,ext3")
	t.Setenv("BR_TTL_HOURS", "")
	t.Setenv("BR_OUTPUT_FORMAT", "")
	t.Setenv("BR_SHOW_PROGRESS", "false")

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}

	// Проверяем расширения
	got := mockCreator.LastCreateTempDBOptions.Extensions
	want := []string{"ext1", "ext2", "ext3"}

	if len(got) != len(want) {
		t.Errorf("Extensions count = %d, want %d", len(got), len(want))
	}

	for i, ext := range want {
		if got[i] != ext {
			t.Errorf("Extensions[%d] = %q, want %q", i, got[i], ext)
		}
	}
}

func TestCreateTempDbHandler_Execute_ParseExtensions_WithSpaces(t *testing.T) {
	tmpDir := t.TempDir()

	// H4 fix: создаём фейковый файл ibcmd
	ibcmdPath := filepath.Join(tmpDir, "ibcmd")
	if err := os.WriteFile(ibcmdPath, []byte("#!/bin/bash\necho 'mock'"), 0755); err != nil {
		t.Fatalf("Failed to create mock ibcmd: %v", err)
	}

	mockCreator := onectest.NewMockTempDatabaseCreator()

	h := &CreateTempDbHandler{
		dbCreator: mockCreator,
	}

	cfg := &config.Config{
		TmpDir: tmpDir,
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				BinIbcmd: ibcmdPath,
			},
		},
	}

	// M2 fix: используем t.Setenv
	t.Setenv("BR_EXTENSIONS", " ext1 , ext2 , ext3 ")
	t.Setenv("BR_OUTPUT_FORMAT", "")
	t.Setenv("BR_SHOW_PROGRESS", "false")

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}

	// Пробелы должны быть удалены
	got := mockCreator.LastCreateTempDBOptions.Extensions
	want := []string{"ext1", "ext2", "ext3"}

	for i, ext := range want {
		if got[i] != ext {
			t.Errorf("Extensions[%d] = %q, want %q (spaces should be trimmed)", i, got[i], ext)
		}
	}
}

func TestCreateTempDbHandler_Execute_UseAddArrayWhenExtensionsEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	// H4 fix: создаём фейковый файл ibcmd
	ibcmdPath := filepath.Join(tmpDir, "ibcmd")
	if err := os.WriteFile(ibcmdPath, []byte("#!/bin/bash\necho 'mock'"), 0755); err != nil {
		t.Fatalf("Failed to create mock ibcmd: %v", err)
	}

	mockCreator := onectest.NewMockTempDatabaseCreator()

	h := &CreateTempDbHandler{
		dbCreator: mockCreator,
	}

	// Конфигурация с AddArray
	cfg := &config.Config{
		TmpDir:   tmpDir,
		AddArray: []string{"configExt", "reportExt"},
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				BinIbcmd: ibcmdPath,
			},
		},
	}

	// M2 fix: используем t.Setenv
	t.Setenv("BR_EXTENSIONS", "")
	t.Setenv("BR_OUTPUT_FORMAT", "")
	t.Setenv("BR_SHOW_PROGRESS", "false")

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}

	// Должен использоваться cfg.AddArray
	got := mockCreator.LastCreateTempDBOptions.Extensions
	want := []string{"configExt", "reportExt"}

	if len(got) != len(want) {
		t.Errorf("Extensions count = %d, want %d", len(got), len(want))
	}

	for i, ext := range want {
		if got[i] != ext {
			t.Errorf("Extensions[%d] = %q, want %q", i, got[i], ext)
		}
	}
}

func TestCreateTempDbHandler_Execute_CreateDBError(t *testing.T) {
	tmpDir := t.TempDir()

	// H4 fix: создаём фейковый файл ibcmd
	ibcmdPath := filepath.Join(tmpDir, "ibcmd")
	if err := os.WriteFile(ibcmdPath, []byte("#!/bin/bash\necho 'mock'"), 0755); err != nil {
		t.Fatalf("Failed to create mock ibcmd: %v", err)
	}

	mockCreator := &onectest.MockTempDatabaseCreator{
		CreateTempDBFunc: func(ctx context.Context, opts onec.CreateTempDBOptions) (*onec.TempDBResult, error) {
			return nil, errors.New("ошибка создания БД: недостаточно прав")
		},
	}

	h := &CreateTempDbHandler{
		dbCreator: mockCreator,
	}

	cfg := &config.Config{
		TmpDir: tmpDir,
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				BinIbcmd: ibcmdPath,
			},
		},
	}

	// M2 fix: используем t.Setenv
	t.Setenv("BR_EXTENSIONS", "")
	t.Setenv("BR_OUTPUT_FORMAT", "")
	t.Setenv("BR_SHOW_PROGRESS", "false")

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	if err == nil {
		t.Error("Execute() error = nil, want error")
	}

	// Проверяем код ошибки
	if !strings.Contains(err.Error(), ErrCreateTempDbFailed) {
		t.Errorf("Error should contain code %q, got %q", ErrCreateTempDbFailed, err.Error())
	}
}

func TestCreateTempDbHandler_Execute_ExtensionError(t *testing.T) {
	tmpDir := t.TempDir()

	// H4 fix: создаём фейковый файл ibcmd
	ibcmdPath := filepath.Join(tmpDir, "ibcmd")
	if err := os.WriteFile(ibcmdPath, []byte("#!/bin/bash\necho 'mock'"), 0755); err != nil {
		t.Fatalf("Failed to create mock ibcmd: %v", err)
	}

	mockCreator := &onectest.MockTempDatabaseCreator{
		CreateTempDBFunc: func(ctx context.Context, opts onec.CreateTempDBOptions) (*onec.TempDBResult, error) {
			return nil, errors.New("ошибка добавления расширения: расширение уже существует")
		},
	}

	h := &CreateTempDbHandler{
		dbCreator: mockCreator,
	}

	cfg := &config.Config{
		TmpDir: tmpDir,
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				BinIbcmd: ibcmdPath,
			},
		},
	}

	// M2 fix: используем t.Setenv
	t.Setenv("BR_EXTENSIONS", "ext1")
	t.Setenv("BR_OUTPUT_FORMAT", "")
	t.Setenv("BR_SHOW_PROGRESS", "false")

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	if err == nil {
		t.Error("Execute() error = nil, want error")
	}

	// Проверяем код ошибки для расширений
	if !strings.Contains(err.Error(), ErrExtensionAddFailed) {
		t.Errorf("Error should contain code %q, got %q", ErrExtensionAddFailed, err.Error())
	}
}

func TestCreateTempDbHandler_Execute_TTLMetadata(t *testing.T) {
	tmpDir := t.TempDir()

	// H4 fix: создаём фейковый файл ibcmd
	ibcmdPath := filepath.Join(tmpDir, "ibcmd")
	if err := os.WriteFile(ibcmdPath, []byte("#!/bin/bash\necho 'mock'"), 0755); err != nil {
		t.Fatalf("Failed to create mock ibcmd: %v", err)
	}

	var capturedDbPath string
	mockCreator := &onectest.MockTempDatabaseCreator{
		CreateTempDBFunc: func(ctx context.Context, opts onec.CreateTempDBOptions) (*onec.TempDBResult, error) {
			capturedDbPath = opts.DbPath
			return &onec.TempDBResult{
				ConnectString: "/F " + opts.DbPath,
				DbPath:        opts.DbPath,
				Extensions:    opts.Extensions,
				CreatedAt:     time.Now(),
				DurationMs:    100,
			}, nil
		},
	}

	h := &CreateTempDbHandler{
		dbCreator: mockCreator,
	}

	cfg := &config.Config{
		TmpDir: tmpDir,
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				BinIbcmd: ibcmdPath,
			},
		},
	}

	// M2 fix: используем t.Setenv
	t.Setenv("BR_TTL_HOURS", "24")
	t.Setenv("BR_EXTENSIONS", "")
	t.Setenv("BR_OUTPUT_FORMAT", "")
	t.Setenv("BR_SHOW_PROGRESS", "false")

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}

	// Проверяем что .ttl файл создан
	ttlPath := capturedDbPath + ".ttl"
	if _, err := os.Stat(ttlPath); os.IsNotExist(err) {
		t.Errorf(".ttl file was not created at %s", ttlPath)
		return
	}

	// Проверяем содержимое .ttl файла
	data, err := os.ReadFile(ttlPath)
	if err != nil {
		t.Fatalf("Failed to read .ttl file: %v", err)
	}

	var metadata TTLMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("Failed to parse .ttl file: %v", err)
	}

	if metadata.TTLHours != 24 {
		t.Errorf("TTLHours = %d, want 24", metadata.TTLHours)
	}

	// Проверяем что ExpiresAt корректен
	expectedExpiry := metadata.CreatedAt.Add(24 * time.Hour)
	if !metadata.ExpiresAt.Equal(expectedExpiry) {
		t.Errorf("ExpiresAt = %v, want %v", metadata.ExpiresAt, expectedExpiry)
	}

	// M1 fix: проверяем что время в файле сериализуется в RFC3339 формате
	// Для этого проверяем что raw JSON содержит корректный формат
	dataStr := string(data)
	// RFC3339 формат: "2006-01-02T15:04:05Z07:00"
	if !strings.Contains(dataStr, "T") || !strings.Contains(dataStr, "Z") && !strings.Contains(dataStr, "+") && !strings.Contains(dataStr, "-") {
		t.Errorf("TTL metadata time should be in RFC3339 format, got: %s", dataStr)
	}
}

func TestCreateTempDbHandler_Execute_JSONOutput(t *testing.T) {
	tmpDir := t.TempDir()

	// H4 fix: создаём фейковый файл ibcmd
	ibcmdPath := filepath.Join(tmpDir, "ibcmd")
	if err := os.WriteFile(ibcmdPath, []byte("#!/bin/bash\necho 'mock'"), 0755); err != nil {
		t.Fatalf("Failed to create mock ibcmd: %v", err)
	}

	mockCreator := onectest.NewMockTempDatabaseCreator()

	h := &CreateTempDbHandler{
		dbCreator: mockCreator,
	}

	cfg := &config.Config{
		TmpDir: tmpDir,
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				BinIbcmd: ibcmdPath,
			},
		},
	}

	// M2 fix: используем t.Setenv
	t.Setenv("BR_OUTPUT_FORMAT", "json")
	t.Setenv("BR_EXTENSIONS", "ext1,ext2")
	t.Setenv("BR_TTL_HOURS", "48")
	t.Setenv("BR_SHOW_PROGRESS", "false")

	// Перехватываем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}

	// Читаем вывод
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Проверяем что это валидный JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Output is not valid JSON: %v\nOutput: %s", err, output)
	}

	// Проверяем ключевые поля
	if result["status"] != "success" {
		t.Errorf("status = %v, want success", result["status"])
	}

	if result["command"] != constants.ActNRCreateTempDb {
		t.Errorf("command = %v, want %s", result["command"], constants.ActNRCreateTempDb)
	}

	// Проверяем data
	data, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data is not a map")
	}

	if data["connect_string"] == "" {
		t.Error("connect_string is empty")
	}

	if data["db_path"] == "" {
		t.Error("db_path is empty")
	}

	// TTL должен быть в данных
	ttl, ok := data["ttl_hours"].(float64)
	if !ok || int(ttl) != 48 {
		t.Errorf("ttl_hours = %v, want 48", data["ttl_hours"])
	}

	// Проверяем metadata
	metadata, ok := result["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("metadata is not a map")
	}

	if metadata["trace_id"] == "" {
		t.Error("trace_id is empty")
	}

	if metadata["api_version"] != constants.APIVersion {
		t.Errorf("api_version = %v, want %s", metadata["api_version"], constants.APIVersion)
	}
}

func TestCreateTempDbHandler_Execute_TextOutput(t *testing.T) {
	tmpDir := t.TempDir()

	// H4 fix: создаём фейковый файл ibcmd
	ibcmdPath := filepath.Join(tmpDir, "ibcmd")
	if err := os.WriteFile(ibcmdPath, []byte("#!/bin/bash\necho 'mock'"), 0755); err != nil {
		t.Fatalf("Failed to create mock ibcmd: %v", err)
	}

	mockCreator := onectest.NewMockTempDatabaseCreator()

	h := &CreateTempDbHandler{
		dbCreator: mockCreator,
	}

	cfg := &config.Config{
		TmpDir: tmpDir,
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				BinIbcmd: ibcmdPath,
			},
		},
	}

	// M2 fix: используем t.Setenv
	t.Setenv("BR_OUTPUT_FORMAT", "text")
	t.Setenv("BR_EXTENSIONS", "ext1")
	t.Setenv("BR_SHOW_PROGRESS", "false")

	// Перехватываем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}

	// Читаем вывод
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Проверяем ключевые фразы
	expectedPhrases := []string{
		"Временная база данных создана",
		"Путь:",
		"Строка подключения:",
		"Расширения:",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(output, phrase) {
			t.Errorf("Output should contain %q\nGot: %s", phrase, output)
		}
	}
}

func TestCreateTempDbHandler_Execute_ValidationError_NilConfig(t *testing.T) {
	h := &CreateTempDbHandler{}

	// M2 fix: используем t.Setenv
	t.Setenv("BR_OUTPUT_FORMAT", "")
	t.Setenv("BR_SHOW_PROGRESS", "false")

	ctx := context.Background()
	err := h.Execute(ctx, nil)

	if err == nil {
		t.Error("Execute() error = nil, want error for nil config")
	}

	if !strings.Contains(err.Error(), ErrCreateTempDbValidation) {
		t.Errorf("Error should contain code %q, got %q", ErrCreateTempDbValidation, err.Error())
	}
}

func TestCreateTempDbHandler_Execute_ValidationError_NoBinIbcmd(t *testing.T) {
	h := &CreateTempDbHandler{}

	cfg := &config.Config{
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				BinIbcmd: "", // пустой путь
			},
		},
	}

	// M2 fix: используем t.Setenv
	t.Setenv("BR_OUTPUT_FORMAT", "")
	t.Setenv("BR_SHOW_PROGRESS", "false")

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	if err == nil {
		t.Error("Execute() error = nil, want error for missing ibcmd path")
	}

	if !strings.Contains(err.Error(), ErrCreateTempDbValidation) {
		t.Errorf("Error should contain code %q, got %q", ErrCreateTempDbValidation, err.Error())
	}
}

// H4 fix: тест для проверки несуществующего файла ibcmd
func TestCreateTempDbHandler_Execute_ValidationError_IbcmdNotExists(t *testing.T) {
	h := &CreateTempDbHandler{}

	cfg := &config.Config{
		TmpDir: t.TempDir(),
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				BinIbcmd: "/nonexistent/path/to/ibcmd", // несуществующий файл
			},
		},
	}

	t.Setenv("BR_OUTPUT_FORMAT", "")
	t.Setenv("BR_SHOW_PROGRESS", "false")

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	if err == nil {
		t.Error("Execute() error = nil, want error for nonexistent ibcmd")
	}

	if !strings.Contains(err.Error(), ErrCreateTempDbValidation) {
		t.Errorf("Error should contain code %q, got %q", ErrCreateTempDbValidation, err.Error())
	}

	if !strings.Contains(err.Error(), "не найден") {
		t.Errorf("Error message should mention file not found, got %q", err.Error())
	}
}

// H3 fix: тест для проверки отмены context
func TestCreateTempDbHandler_Execute_ContextCancelled(t *testing.T) {
	ctx := context.Background()
	h := &CreateTempDbHandler{}

	t.Setenv("BR_OUTPUT_FORMAT", "")
	t.Setenv("BR_SHOW_PROGRESS", "false")

	// Создаём уже отменённый context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Отменяем сразу

	cfg := &config.Config{
		TmpDir: t.TempDir(),
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				BinIbcmd: "/tmp/fake-ibcmd",
			},
		},
	}

	err := h.Execute(ctx, cfg)

	if err == nil {
		t.Error("Execute() error = nil, want error for cancelled context")
	}

	if !strings.Contains(err.Error(), ErrContextCancelled) {
		t.Errorf("Error should contain code %q, got %q", ErrContextCancelled, err.Error())
	}

	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("Error message should mention context canceled, got %q", err.Error())
	}
}

func TestCreateTempDbHandler_generateDbPath(t *testing.T) {
	h := &CreateTempDbHandler{}

	// Создаём реальную временную директорию для тестов с разрешёнными путями
	realTmpDir := t.TempDir()

	tests := []struct {
		name      string
		tmpDir    string
		wantDir   string
		wantError bool
	}{
		{
			name:      "with allowed real temp path",
			tmpDir:    realTmpDir,
			wantDir:   realTmpDir,
			wantError: false,
		},
		{
			name:      "with empty TmpDir uses default",
			tmpDir:    "",
			wantDir:   constants.TempDir,
			wantError: false,
		},
		{
			name:      "with disallowed path returns error (H2 fix)",
			tmpDir:    "/etc/dangerous",
			wantDir:   "",
			wantError: true,
		},
		{
			name:      "with disallowed home path returns error",
			tmpDir:    "/root/test",
			wantDir:   "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				TmpDir: tt.tmpDir,
			}

			got, err := h.generateDbPath(cfg)

			if tt.wantError {
				if err == nil {
					t.Errorf("generateDbPath() error = nil, want error for path %q", tt.tmpDir)
				}
				return
			}

			if err != nil {
				t.Errorf("generateDbPath() unexpected error = %v", err)
				return
			}

			// Проверяем что путь начинается с правильной директории
			dir := filepath.Dir(got)
			if dir != tt.wantDir {
				t.Errorf("generateDbPath() dir = %q, want %q", dir, tt.wantDir)
			}

			// Проверяем формат имени
			base := filepath.Base(got)
			if !strings.HasPrefix(base, "temp_db_") {
				t.Errorf("generateDbPath() base = %q, want prefix 'temp_db_'", base)
			}
		})
	}
}

func TestCreateTempDbHandler_parseExtensions(t *testing.T) {
	h := &CreateTempDbHandler{}

	tests := []struct {
		name      string
		envValue  string
		cfgAddArr []string
		want      []string
		wantCount int
	}{
		{
			name:      "from BR_EXTENSIONS",
			envValue:  "ext1,ext2,ext3",
			cfgAddArr: []string{"configExt"},
			want:      []string{"ext1", "ext2", "ext3"},
			wantCount: 3,
		},
		{
			name:      "from cfg.AddArray when BR_EXTENSIONS empty",
			envValue:  "",
			cfgAddArr: []string{"configExt", "reportExt"},
			want:      []string{"configExt", "reportExt"},
			wantCount: 2,
		},
		{
			name:      "empty when both sources empty",
			envValue:  "",
			cfgAddArr: nil,
			want:      nil,
			wantCount: 0,
		},
		{
			name:      "trims whitespace",
			envValue:  " ext1 , ext2 ",
			cfgAddArr: nil,
			want:      []string{"ext1", "ext2"},
			wantCount: 2,
		},
		{
			name:      "ignores empty parts",
			envValue:  "ext1,,ext2,",
			cfgAddArr: nil,
			want:      []string{"ext1", "ext2"},
			wantCount: 2,
		},
		{
			name:      "M5_fix_limits_to_maxExtensions",
			envValue:  generateManyExtensions(60), // больше чем maxExtensions=50
			cfgAddArr: nil,
			want:      nil, // проверяем только count
			wantCount: maxExtensions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// M1 fix: используем t.Setenv для thread-safety
			t.Setenv("BR_EXTENSIONS", tt.envValue)

			cfg := &config.Config{
				AddArray: tt.cfgAddArr,
			}

			got := h.parseExtensions(cfg)

			if len(got) != tt.wantCount {
				t.Errorf("parseExtensions() count = %d, want %d", len(got), tt.wantCount)
			}

			for i, ext := range tt.want {
				if i < len(got) && got[i] != ext {
					t.Errorf("parseExtensions()[%d] = %q, want %q", i, got[i], ext)
				}
			}
		})
	}
}

func TestCreateTempDbHandler_getTimeout(t *testing.T) {
	h := &CreateTempDbHandler{}

	tests := []struct {
		name     string
		envValue string
		want     time.Duration
	}{
		{
			name:     "default timeout",
			envValue: "",
			want:     defaultTimeout,
		},
		{
			name:     "custom timeout",
			envValue: "60",
			want:     60 * time.Minute,
		},
		{
			name:     "invalid timeout uses default",
			envValue: "invalid",
			want:     defaultTimeout,
		},
		{
			name:     "zero timeout uses default",
			envValue: "0",
			want:     defaultTimeout,
		},
		{
			name:     "negative timeout uses default",
			envValue: "-5",
			want:     defaultTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// M1 fix: используем t.Setenv для thread-safety
			t.Setenv("BR_TIMEOUT_MIN", tt.envValue)

			got := h.getTimeout()

			if got != tt.want {
				t.Errorf("getTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateTempDbHandler_getTTLHours(t *testing.T) {
	h := &CreateTempDbHandler{}

	tests := []struct {
		name     string
		envValue string
		want     int
	}{
		{
			name:     "no TTL",
			envValue: "",
			want:     0,
		},
		{
			name:     "valid TTL",
			envValue: "24",
			want:     24,
		},
		{
			name:     "invalid TTL returns 0",
			envValue: "invalid",
			want:     0,
		},
		{
			name:     "negative TTL returns 0",
			envValue: "-5",
			want:     0,
		},
		{
			name:     "zero TTL",
			envValue: "0",
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// M1 fix: используем t.Setenv для thread-safety
			t.Setenv("BR_TTL_HOURS", tt.envValue)

			got := h.getTTLHours()

			if got != tt.want {
				t.Errorf("getTTLHours() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCreateTempDbData_writeText(t *testing.T) {
	data := &CreateTempDbData{
		ConnectString: "/F /tmp/temp_db_20260203_120000",
		DbPath:        "/tmp/temp_db_20260203_120000",
		Extensions:    []string{"ext1", "ext2"},
		TTLHours:      24,
		CreatedAt:     "2026-02-03T12:00:00Z",
		DurationMs:    1500,
	}

	var buf bytes.Buffer
	err := data.writeText(&buf)

	if err != nil {
		t.Errorf("writeText() error = %v", err)
	}

	output := buf.String()

	expectedPhrases := []string{
		"Временная база данных создана",
		"/tmp/temp_db_20260203_120000",
		"/F /tmp/temp_db_20260203_120000",
		"ext1, ext2",
		"TTL: 24 часов",
		"2026-02-03T12:00:00Z",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(output, phrase) {
			t.Errorf("writeText() output should contain %q\nGot: %s", phrase, output)
		}
	}
}

func TestCreateTempDbData_writeText_NoExtensions(t *testing.T) {
	data := &CreateTempDbData{
		ConnectString: "/F /tmp/temp_db",
		DbPath:        "/tmp/temp_db",
		Extensions:    nil, // нет расширений
		TTLHours:      0,   // нет TTL
		CreatedAt:     "2026-02-03T12:00:00Z",
		DurationMs:    1000,
	}

	var buf bytes.Buffer
	err := data.writeText(&buf)

	if err != nil {
		t.Errorf("writeText() error = %v", err)
	}

	output := buf.String()

	// Должно быть "Расширения: нет"
	if !strings.Contains(output, "Расширения: нет") {
		t.Errorf("writeText() output should contain 'Расширения: нет'\nGot: %s", output)
	}

	// Не должно быть TTL строки
	if strings.Contains(output, "TTL:") {
		t.Errorf("writeText() output should not contain TTL when ttl_hours=0\nGot: %s", output)
	}
}

// TestCreateTempDbHandler_Execute_ValidationError_IbcmdNotExecutable проверяет файл ibcmd без прав на выполнение.
func TestCreateTempDbHandler_Execute_ValidationError_IbcmdNotExecutable(t *testing.T) {
	tmpDir := t.TempDir()

	// Создаём файл ibcmd БЕЗ execute permission (0644 вместо 0755)
	ibcmdPath := filepath.Join(tmpDir, "ibcmd")
	if err := os.WriteFile(ibcmdPath, []byte("not executable"), 0644); err != nil {
		t.Fatalf("Failed to create non-executable ibcmd: %v", err)
	}

	h := &CreateTempDbHandler{}

	cfg := &config.Config{
		TmpDir: tmpDir,
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				BinIbcmd: ibcmdPath,
			},
		},
	}

	t.Setenv("BR_OUTPUT_FORMAT", "")
	t.Setenv("BR_SHOW_PROGRESS", "false")

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	if err == nil {
		t.Error("Execute() error = nil, want error for non-executable ibcmd")
	}

	if !strings.Contains(err.Error(), ErrCreateTempDbValidation) {
		t.Errorf("Error should contain code %q, got %q", ErrCreateTempDbValidation, err.Error())
	}

	if !strings.Contains(err.Error(), "не является исполняемым") {
		t.Errorf("Error message should mention not executable, got %q", err.Error())
	}
}

// TestCreateTempDbHandler_generateDbPath_UniqueWithNanoseconds проверяет уникальность путей.
func TestCreateTempDbHandler_generateDbPath_UniqueWithNanoseconds(t *testing.T) {
	h := &CreateTempDbHandler{}
	// Используем реальную временную директорию для корректной работы M-6 проверки
	realTmpDir := t.TempDir()
	cfg := &config.Config{
		TmpDir: realTmpDir,
	}

	// Генерируем несколько путей подряд — должны быть уникальными
	paths := make(map[string]bool)
	for i := 0; i < 100; i++ {
		path, err := h.generateDbPath(cfg)
		if err != nil {
			t.Fatalf("generateDbPath() error = %v", err)
		}

		if paths[path] {
			t.Errorf("generateDbPath() generated duplicate path: %s", path)
		}
		paths[path] = true
	}

	// Проверяем что пути содержат наносекунды (длинный суффикс)
	for path := range paths {
		base := filepath.Base(path)
		// Формат: temp_db_YYYYMMDD_HHMMSS_NNNNNNNNN
		// Минимальная длина: temp_db_ (8) + YYYYMMDD (8) + _ (1) + HHMMSS (6) + _ (1) + NNNNNNNNN (9) = 33
		if len(base) < 33 {
			t.Errorf("generateDbPath() path too short (expected nanoseconds): %s", base)
		}
	}
}

// TestCreateTempDbData_writeText_ZeroDuration проверяет вывод времени "< 1ms" для быстрых операций.
func TestCreateTempDbData_writeText_ZeroDuration(t *testing.T) {
	data := &CreateTempDbData{
		ConnectString: "/F /tmp/temp_db",
		DbPath:        "/tmp/temp_db",
		Extensions:    nil,
		TTLHours:      0,
		CreatedAt:     "2026-02-03T12:00:00Z",
		DurationMs:    0, // 0 миллисекунд
	}

	var buf bytes.Buffer
	err := data.writeText(&buf)

	if err != nil {
		t.Errorf("writeText() error = %v", err)
	}

	output := buf.String()

	// M2 fix: должно быть "< 1ms" вместо "0s"
	if !strings.Contains(output, "< 1ms") {
		t.Errorf("writeText() output should contain '< 1ms' for zero duration\nGot: %s", output)
	}

	// НЕ должно быть "0s"
	if strings.Contains(output, "0s") {
		t.Errorf("writeText() output should NOT contain '0s'\nGot: %s", output)
	}
}

// ==== DRY-RUN TESTS ====

// TestCreateTempDbHandler_DryRun_Success проверяет успешный dry-run с построением плана.
// AC-1: При BR_DRY_RUN=true команды возвращают план действий БЕЗ выполнения.
func TestCreateTempDbHandler_DryRun_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// H4 fix: создаём фейковый файл ibcmd
	ibcmdPath := filepath.Join(tmpDir, "ibcmd")
	if err := os.WriteFile(ibcmdPath, []byte("#!/bin/bash\necho 'mock'"), 0755); err != nil {
		t.Fatalf("Failed to create mock ibcmd: %v", err)
	}

	// Создаём mock который FAIL-ит при любом вызове
	// AC-8: В dry-run режиме НЕ вызывается client.CreateTempDB()
	mockCreator := &FailOnCallMockCreator{t: t}

	cfg := &config.Config{
		TmpDir: tmpDir,
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				BinIbcmd: ibcmdPath,
			},
		},
	}

	h := &CreateTempDbHandler{dbCreator: mockCreator}

	t.Setenv("BR_DRY_RUN", "true")
	t.Setenv("BR_SHOW_PROGRESS", "false")

	// Перехватываем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	capturedOutput := buf.String()

	// AC-5: exit code = 0 если план валиден
	if err != nil {
		t.Errorf("DryRun Execute() unexpected error = %v", err)
	}

	// AC-4: Text output форматирует план человекочитаемо с заголовком "=== DRY RUN ==="
	expectedParts := []string{
		"=== DRY RUN ===",
		"Команда: nr-create-temp-db",
		"Валидация: ✅ Пройдена",
		"План выполнения:",
		"Валидация конфигурации",
		"Генерация пути к временной базе",
		"Создание базы данных",
		"=== END DRY RUN ===",
	}

	for _, part := range expectedParts {
		if !strings.Contains(capturedOutput, part) {
			t.Errorf("DryRun output should contain %q, got: %s", part, capturedOutput)
		}
	}
}

// TestCreateTempDbHandler_DryRun_JSONOutput проверяет JSON формат dry-run вывода.
// AC-3: JSON output имеет поле "dry_run": true и структуру "plan": {...}.
func TestCreateTempDbHandler_DryRun_JSONOutput(t *testing.T) {
	tmpDir := t.TempDir()

	ibcmdPath := filepath.Join(tmpDir, "ibcmd")
	if err := os.WriteFile(ibcmdPath, []byte("#!/bin/bash\necho 'mock'"), 0755); err != nil {
		t.Fatalf("Failed to create mock ibcmd: %v", err)
	}

	mockCreator := &FailOnCallMockCreator{t: t}

	cfg := &config.Config{
		TmpDir: tmpDir,
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				BinIbcmd: ibcmdPath,
			},
		},
	}

	h := &CreateTempDbHandler{dbCreator: mockCreator}

	t.Setenv("BR_DRY_RUN", "true")
	t.Setenv("BR_OUTPUT_FORMAT", "json")
	t.Setenv("BR_SHOW_PROGRESS", "false")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)

	if err != nil {
		t.Errorf("DryRun Execute() unexpected error = %v", err)
	}

	var result map[string]interface{}
	if jsonErr := json.Unmarshal(buf.Bytes(), &result); jsonErr != nil {
		t.Errorf("JSON unmarshal error: %v, output: %s", jsonErr, buf.String())
	}

	// AC-3: dry_run: true
	if dryRun, ok := result["dry_run"].(bool); !ok || !dryRun {
		t.Errorf("JSON dry_run = %v, want true", result["dry_run"])
	}

	// AC-3: plan: {...}
	if result["plan"] == nil {
		t.Error("JSON plan should not be nil")
	}

	if result["command"] != constants.ActNRCreateTempDb {
		t.Errorf("command = %v, want %s", result["command"], constants.ActNRCreateTempDb)
	}
}

// TestCreateTempDbHandler_DryRun_NoMockCalls проверяет что mock НЕ вызывается в dry-run.
// AC-8: В dry-run режиме НЕ вызывается client.CreateTempDB().
func TestCreateTempDbHandler_DryRun_NoMockCalls(t *testing.T) {
	tmpDir := t.TempDir()

	ibcmdPath := filepath.Join(tmpDir, "ibcmd")
	if err := os.WriteFile(ibcmdPath, []byte("#!/bin/bash\necho 'mock'"), 0755); err != nil {
		t.Fatalf("Failed to create mock ibcmd: %v", err)
	}

	// FailOnCallMockCreator упадёт если вызван
	mockCreator := &FailOnCallMockCreator{t: t}

	cfg := &config.Config{
		TmpDir: tmpDir,
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				BinIbcmd: ibcmdPath,
			},
		},
	}

	h := &CreateTempDbHandler{dbCreator: mockCreator}

	t.Setenv("BR_DRY_RUN", "true")
	t.Setenv("BR_SHOW_PROGRESS", "false")

	// Перехватываем stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	// Если mock был вызван — тест упал бы в FailOnCallMockCreator
	if err != nil {
		t.Errorf("DryRun Execute() unexpected error = %v", err)
	}
}

// TestCreateTempDbHandler_DryRun_WithExtensions проверяет dry-run с расширениями.
func TestCreateTempDbHandler_DryRun_WithExtensions(t *testing.T) {
	tmpDir := t.TempDir()

	ibcmdPath := filepath.Join(tmpDir, "ibcmd")
	if err := os.WriteFile(ibcmdPath, []byte("#!/bin/bash\necho 'mock'"), 0755); err != nil {
		t.Fatalf("Failed to create mock ibcmd: %v", err)
	}

	mockCreator := &FailOnCallMockCreator{t: t}

	cfg := &config.Config{
		TmpDir: tmpDir,
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				BinIbcmd: ibcmdPath,
			},
		},
	}

	h := &CreateTempDbHandler{dbCreator: mockCreator}

	t.Setenv("BR_DRY_RUN", "true")
	t.Setenv("BR_EXTENSIONS", "ext1,ext2")
	t.Setenv("BR_SHOW_PROGRESS", "false")

	// Перехватываем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	capturedOutput := buf.String()

	if err != nil {
		t.Errorf("DryRun Execute() unexpected error = %v", err)
	}

	// Проверяем что расширения упоминаются в плане
	if !strings.Contains(capturedOutput, "ext1, ext2") {
		t.Errorf("DryRun output should contain extensions, got: %s", capturedOutput)
	}

	if !strings.Contains(capturedOutput, "Добавление расширений") {
		t.Errorf("DryRun output should contain extension step, got: %s", capturedOutput)
	}
}

// TestCreateTempDbHandler_DryRun_WithTTL проверяет dry-run с TTL.
func TestCreateTempDbHandler_DryRun_WithTTL(t *testing.T) {
	tmpDir := t.TempDir()

	ibcmdPath := filepath.Join(tmpDir, "ibcmd")
	if err := os.WriteFile(ibcmdPath, []byte("#!/bin/bash\necho 'mock'"), 0755); err != nil {
		t.Fatalf("Failed to create mock ibcmd: %v", err)
	}

	mockCreator := &FailOnCallMockCreator{t: t}

	cfg := &config.Config{
		TmpDir: tmpDir,
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				BinIbcmd: ibcmdPath,
			},
		},
	}

	h := &CreateTempDbHandler{dbCreator: mockCreator}

	t.Setenv("BR_DRY_RUN", "true")
	t.Setenv("BR_TTL_HOURS", "24")
	t.Setenv("BR_SHOW_PROGRESS", "false")

	// Перехватываем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	capturedOutput := buf.String()

	if err != nil {
		t.Errorf("DryRun Execute() unexpected error = %v", err)
	}

	// Проверяем что TTL упоминается в плане
	if !strings.Contains(capturedOutput, "TTL metadata") {
		t.Errorf("DryRun output should contain TTL step, got: %s", capturedOutput)
	}

	if !strings.Contains(capturedOutput, "24 часов") {
		t.Errorf("DryRun output should contain TTL hours, got: %s", capturedOutput)
	}
}

// TestCreateTempDbHandler_DryRun_ValidationError проверяет что dry-run возвращает ошибку валидации.
// AC-6: При ошибке валидации возвращается error с описанием проблемы.
func TestCreateTempDbHandler_DryRun_ValidationError(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.Config
		wantErrCode string
	}{
		{
			name:        "nil config",
			cfg:         nil,
			wantErrCode: ErrCreateTempDbValidation,
		},
		{
			name: "empty ibcmd path",
			cfg: &config.Config{
				AppConfig: &config.AppConfig{
					Paths: struct {
						Bin1cv8  string `yaml:"bin1cv8"`
						BinIbcmd string `yaml:"binIbcmd"`
						EdtCli   string `yaml:"edtCli"`
						Rac      string `yaml:"rac"`
					}{
						BinIbcmd: "", // пустой путь
					},
				},
			},
			wantErrCode: ErrCreateTempDbValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCreator := &FailOnCallMockCreator{t: t}
			h := &CreateTempDbHandler{dbCreator: mockCreator}

			t.Setenv("BR_DRY_RUN", "true")
			t.Setenv("BR_SHOW_PROGRESS", "false")

			// Перехватываем stdout
			oldStdout := os.Stdout
			_, w, _ := os.Pipe()
			os.Stdout = w

			err := h.Execute(context.Background(), tt.cfg)

			_ = w.Close()
			os.Stdout = oldStdout

			// AC-6: Должна быть ошибка валидации
			if err == nil {
				t.Error("DryRun Execute() should return error for invalid config")
			}

			if err != nil && !strings.Contains(err.Error(), tt.wantErrCode) {
				t.Errorf("DryRun Execute() error = %v, want error containing %q", err, tt.wantErrCode)
			}
		})
	}
}

// ==== PLAN-ONLY TESTS (Story 7.3) ====

// TestCreateTempDbHandler_PlanOnly_TextOutput проверяет текстовый вывод plan-only режима.
// Story 7.3 AC-1: При BR_PLAN_ONLY=true команда выводит план без выполнения.
// Story 7.3 AC-2: Заголовок "=== OPERATION PLAN ===" (не "=== DRY RUN ===").
func TestCreateTempDbHandler_PlanOnly_TextOutput(t *testing.T) {
	tmpDir := t.TempDir()

	ibcmdPath := filepath.Join(tmpDir, "ibcmd")
	if err := os.WriteFile(ibcmdPath, []byte("#!/bin/bash\necho 'mock'"), 0755); err != nil {
		t.Fatalf("Failed to create mock ibcmd: %v", err)
	}

	// FailOnCallMockCreator гарантирует что CreateTempDB НЕ вызывается
	mockCreator := &FailOnCallMockCreator{t: t}

	cfg := &config.Config{
		TmpDir: tmpDir,
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				BinIbcmd: ibcmdPath,
			},
		},
	}

	h := &CreateTempDbHandler{dbCreator: mockCreator}

	t.Setenv("BR_PLAN_ONLY", "true")
	t.Setenv("BR_SHOW_PROGRESS", "false")

	// Перехватываем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	capturedOutput := buf.String()

	// Plan-only не должен возвращать ошибку для валидной конфигурации
	if err != nil {
		t.Errorf("PlanOnly Execute() unexpected error = %v", err)
	}

	// Проверяем наличие заголовков plan-only (НЕ dry-run)
	expectedParts := []string{
		"=== OPERATION PLAN ===",
		"Команда: nr-create-temp-db",
		"Валидация: ✅ Пройдена",
		"=== END OPERATION PLAN ===",
	}

	for _, part := range expectedParts {
		if !strings.Contains(capturedOutput, part) {
			t.Errorf("PlanOnly output should contain %q, got: %s", part, capturedOutput)
		}
	}

	// НЕ должно быть заголовка DRY RUN
	if strings.Contains(capturedOutput, "=== DRY RUN ===") {
		t.Errorf("PlanOnly output should NOT contain '=== DRY RUN ===', got: %s", capturedOutput)
	}
}

// TestCreateTempDbHandler_PlanOnly_JSONOutput проверяет JSON вывод plan-only режима.
// Story 7.3 AC-6: JSON output содержит plan_only: true и plan: {...}.
func TestCreateTempDbHandler_PlanOnly_JSONOutput(t *testing.T) {
	tmpDir := t.TempDir()

	ibcmdPath := filepath.Join(tmpDir, "ibcmd")
	if err := os.WriteFile(ibcmdPath, []byte("#!/bin/bash\necho 'mock'"), 0755); err != nil {
		t.Fatalf("Failed to create mock ibcmd: %v", err)
	}

	mockCreator := &FailOnCallMockCreator{t: t}

	cfg := &config.Config{
		TmpDir: tmpDir,
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				BinIbcmd: ibcmdPath,
			},
		},
	}

	h := &CreateTempDbHandler{dbCreator: mockCreator}

	t.Setenv("BR_PLAN_ONLY", "true")
	t.Setenv("BR_OUTPUT_FORMAT", "json")
	t.Setenv("BR_SHOW_PROGRESS", "false")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)

	if err != nil {
		t.Errorf("PlanOnly Execute() unexpected error = %v", err)
	}

	var result map[string]interface{}
	if jsonErr := json.Unmarshal(buf.Bytes(), &result); jsonErr != nil {
		t.Errorf("JSON unmarshal error: %v, output: %s", jsonErr, buf.String())
	}

	// AC-6: plan_only: true
	if planOnly, ok := result["plan_only"].(bool); !ok || !planOnly {
		t.Errorf("JSON plan_only = %v, want true", result["plan_only"])
	}

	// Plan не должен быть nil
	if result["plan"] == nil {
		t.Error("JSON plan should not be nil")
	}

	// dry_run НЕ должен присутствовать или быть false
	if dryRun, ok := result["dry_run"].(bool); ok && dryRun {
		t.Error("JSON dry_run should NOT be true in plan-only mode")
	}

	if result["command"] != constants.ActNRCreateTempDb {
		t.Errorf("command = %v, want %s", result["command"], constants.ActNRCreateTempDb)
	}
}

// TestCreateTempDbHandler_Priority_DryRunOverPlanOnly проверяет приоритет:
// BR_DRY_RUN > BR_PLAN_ONLY. Если оба заданы, должен сработать dry-run.
func TestCreateTempDbHandler_Priority_DryRunOverPlanOnly(t *testing.T) {
	tmpDir := t.TempDir()

	ibcmdPath := filepath.Join(tmpDir, "ibcmd")
	if err := os.WriteFile(ibcmdPath, []byte("#!/bin/bash\necho 'mock'"), 0755); err != nil {
		t.Fatalf("Failed to create mock ibcmd: %v", err)
	}

	mockCreator := &FailOnCallMockCreator{t: t}

	cfg := &config.Config{
		TmpDir: tmpDir,
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				BinIbcmd: ibcmdPath,
			},
		},
	}

	h := &CreateTempDbHandler{dbCreator: mockCreator}

	// Устанавливаем оба флага
	t.Setenv("BR_DRY_RUN", "true")
	t.Setenv("BR_PLAN_ONLY", "true")
	t.Setenv("BR_SHOW_PROGRESS", "false")

	// Перехватываем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	capturedOutput := buf.String()

	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
	}

	// Должен быть DRY RUN (приоритет выше)
	if !strings.Contains(capturedOutput, "=== DRY RUN ===") {
		t.Errorf("Output should contain '=== DRY RUN ===' (priority over plan-only), got: %s", capturedOutput)
	}

	// НЕ должно быть OPERATION PLAN
	if strings.Contains(capturedOutput, "=== OPERATION PLAN ===") {
		t.Errorf("Output should NOT contain '=== OPERATION PLAN ===' when dry-run has priority, got: %s", capturedOutput)
	}
}

// ==== VERBOSE / PRIORITY TESTS (Story 7.3) ====

// TestCreateTempDbHandler_Verbose_TextOutput проверяет verbose режим: план выводится ПЕРЕД выполнением.
// Story 7.3 AC-4: В verbose режиме сначала выводится план, затем выполняется операция.
func TestCreateTempDbHandler_Verbose_TextOutput(t *testing.T) {
	tmpDir := t.TempDir()

	ibcmdPath := filepath.Join(tmpDir, "ibcmd")
	if err := os.WriteFile(ibcmdPath, []byte("#!/bin/bash\necho 'mock'"), 0755); err != nil {
		t.Fatalf("Failed to create mock ibcmd: %v", err)
	}

	mockCreator := onectest.NewMockTempDatabaseCreator()

	h := &CreateTempDbHandler{dbCreator: mockCreator}

	cfg := &config.Config{
		TmpDir: tmpDir,
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				BinIbcmd: ibcmdPath,
			},
		},
	}

	t.Setenv("BR_VERBOSE", "true")
	t.Setenv("BR_SHOW_PROGRESS", "false")
	t.Setenv("BR_EXTENSIONS", "")
	t.Setenv("BR_OUTPUT_FORMAT", "")

	// Перехватываем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	capturedOutput := buf.String()

	if err != nil {
		t.Errorf("Verbose Execute() unexpected error = %v", err)
	}

	// Проверяем что план выведен перед выполнением
	if !strings.Contains(capturedOutput, "=== OPERATION PLAN ===") {
		t.Errorf("Verbose output should contain '=== OPERATION PLAN ===', got: %s", capturedOutput)
	}

	// Проверяем что реальное выполнение произошло
	if !strings.Contains(capturedOutput, "Временная база данных создана") {
		t.Errorf("Verbose output should contain 'Временная база данных создана', got: %s", capturedOutput)
	}
}

// TestCreateTempDbHandler_Verbose_JSONOutput проверяет JSON вывод в verbose режиме.
// Story 7.3 AC-7: verbose JSON включает план в результат.
func TestCreateTempDbHandler_Verbose_JSONOutput(t *testing.T) {
	tmpDir := t.TempDir()

	ibcmdPath := filepath.Join(tmpDir, "ibcmd")
	if err := os.WriteFile(ibcmdPath, []byte("#!/bin/bash\necho 'mock'"), 0755); err != nil {
		t.Fatalf("Failed to create mock ibcmd: %v", err)
	}

	mockCreator := onectest.NewMockTempDatabaseCreator()

	h := &CreateTempDbHandler{dbCreator: mockCreator}

	cfg := &config.Config{
		TmpDir: tmpDir,
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				BinIbcmd: ibcmdPath,
			},
		},
	}

	t.Setenv("BR_VERBOSE", "true")
	t.Setenv("BR_OUTPUT_FORMAT", "json")
	t.Setenv("BR_SHOW_PROGRESS", "false")
	t.Setenv("BR_EXTENSIONS", "")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)

	if err != nil {
		t.Errorf("Verbose Execute() unexpected error = %v", err)
	}

	var result map[string]interface{}
	if jsonErr := json.Unmarshal(buf.Bytes(), &result); jsonErr != nil {
		t.Errorf("JSON unmarshal error: %v, output: %s", jsonErr, buf.String())
	}

	// Verbose включает план в JSON результат
	if result["plan"] == nil {
		t.Error("Verbose JSON result.plan should not be nil")
	}

	// Verbose — не plan-only и не dry-run
	if planOnly, ok := result["plan_only"].(bool); ok && planOnly {
		t.Error("Verbose JSON result.plan_only should be false")
	}
	if dryRun, ok := result["dry_run"].(bool); ok && dryRun {
		t.Error("Verbose JSON result.dry_run should be false")
	}

	// Реальное выполнение должно произойти
	if result["status"] != "success" {
		t.Errorf("Verbose JSON result.status = %v, want 'success'", result["status"])
	}

	// Data должна присутствовать (реальное выполнение)
	if result["data"] == nil {
		t.Error("Verbose JSON result.data should not be nil (real execution happened)")
	}
}

// TestCreateTempDbHandler_Priority_DryRunOverVerbose проверяет приоритет dry-run над verbose.
// AC-9: dry-run имеет высший приоритет над verbose.
func TestCreateTempDbHandler_Priority_DryRunOverVerbose(t *testing.T) {
	tmpDir := t.TempDir()

	ibcmdPath := filepath.Join(tmpDir, "ibcmd")
	if err := os.WriteFile(ibcmdPath, []byte("#!/bin/bash\necho 'mock'"), 0755); err != nil {
		t.Fatalf("Failed to create mock ibcmd: %v", err)
	}

	mockCreator := &FailOnCallMockCreator{t: t}

	cfg := &config.Config{
		TmpDir: tmpDir,
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				BinIbcmd: ibcmdPath,
			},
		},
	}

	h := &CreateTempDbHandler{dbCreator: mockCreator}

	t.Setenv("BR_DRY_RUN", "true")
	t.Setenv("BR_VERBOSE", "true")
	t.Setenv("BR_SHOW_PROGRESS", "false")

	// Перехватываем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	capturedOutput := buf.String()

	if err != nil {
		t.Errorf("Priority Execute() unexpected error = %v", err)
	}

	// Должен быть dry-run заголовок, НЕ operation plan
	if !strings.Contains(capturedOutput, "=== DRY RUN ===") {
		t.Errorf("Output should contain '=== DRY RUN ===' (dry-run priority), got: %s", capturedOutput)
	}
	if strings.Contains(capturedOutput, "=== OPERATION PLAN ===") {
		t.Errorf("Output should NOT contain '=== OPERATION PLAN ===' when dry-run active, got: %s", capturedOutput)
	}
}

// TestCreateTempDbHandler_Priority_PlanOnlyOverVerbose проверяет приоритет plan-only над verbose.
// Plan-only останавливает выполнение (показывает план, не выполняет).
// Verbose показывает план и выполняет. Plan-only имеет приоритет.
func TestCreateTempDbHandler_Priority_PlanOnlyOverVerbose(t *testing.T) {
	tmpDir := t.TempDir()

	ibcmdPath := filepath.Join(tmpDir, "ibcmd")
	if err := os.WriteFile(ibcmdPath, []byte("#!/bin/bash\necho 'mock'"), 0755); err != nil {
		t.Fatalf("Failed to create mock ibcmd: %v", err)
	}

	mockCreator := &FailOnCallMockCreator{t: t}

	cfg := &config.Config{
		TmpDir: tmpDir,
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				BinIbcmd: ibcmdPath,
			},
		},
	}

	h := &CreateTempDbHandler{dbCreator: mockCreator}

	t.Setenv("BR_PLAN_ONLY", "true")
	t.Setenv("BR_VERBOSE", "true")
	t.Setenv("BR_SHOW_PROGRESS", "false")

	// Перехватываем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	capturedOutput := buf.String()

	if err != nil {
		t.Errorf("Priority Execute() unexpected error = %v", err)
	}

	// Должен быть plan-only заголовок
	if !strings.Contains(capturedOutput, "=== OPERATION PLAN ===") {
		t.Errorf("Output should contain '=== OPERATION PLAN ===', got: %s", capturedOutput)
	}

	// НЕ должно быть реального выполнения
	if strings.Contains(capturedOutput, "Временная база данных создана") {
		t.Errorf("Output should NOT contain 'Временная база данных создана' (plan-only, no execution), got: %s", capturedOutput)
	}
}

// FailOnCallMockCreator — mock который падает при любом вызове CreateTempDB.
type FailOnCallMockCreator struct {
	t *testing.T
}

func (m *FailOnCallMockCreator) CreateTempDB(ctx context.Context, opts onec.CreateTempDBOptions) (*onec.TempDBResult, error) {
	m.t.Fatal("CreateTempDB() не должен вызываться в dry-run режиме")
	return nil, nil
}
